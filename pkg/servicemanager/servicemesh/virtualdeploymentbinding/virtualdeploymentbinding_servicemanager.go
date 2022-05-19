/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeploymentbinding

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/services"
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	podpkg "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/pod"
	servicepkg "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/service"
	vdpkg "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/virtualdeployment"
)

type VirtualDeploymentBindingServiceManager struct {
	client            client.Client
	log               loggerutil.OSOKLogger
	clientSet         kubernetes.Interface
	referenceResolver references.Resolver
	meshClient        services.ServiceMeshClient
}

//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=virtualdeploymentbindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=virtualdeploymentbindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=virtualdeploymentbindings/finalizers,verbs=update

func NewVirtualDeploymentBindingServiceManager(client client.Client, log loggerutil.OSOKLogger, clientSet kubernetes.Interface,
	referenceResolver references.Resolver, meshClient services.ServiceMeshClient) *VirtualDeploymentBindingServiceManager {
	return &VirtualDeploymentBindingServiceManager{
		client:            client,
		log:               log,
		clientSet:         clientSet,
		referenceResolver: referenceResolver,
		meshClient:        meshClient,
	}
}

func (h *VirtualDeploymentBindingServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	vdb, err := verifyEntity(obj)
	if err != nil {
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}
	vdId := vdb.Spec.VirtualDeployment.Id
	// If DeploymentId is present assume CP flow, else K8 flow
	if len(vdId) > 0 {
		return h.reconcileCp(ctx, vdb)
	} else {
		return h.reconcileK8s(ctx, vdb)
	}
}

// reconcileCp reconciles the VDB CR when VD OCID is present
func (h *VirtualDeploymentBindingServiceManager) reconcileCp(ctx context.Context, vdb *servicemeshapi.VirtualDeploymentBinding) (servicemanager.OSOKResponse, error) {
	vd, err := h.resolveVirtualDeploymentCp(ctx, vdb)
	if err != nil {
		h.updateCRStatus(ctx, vdb, err)
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}
	if err := h.validateVirtualDeploymentCp(vd); err != nil {
		_ = h.updateServiceMeshConditionWithVDCondition(ctx, vdb, meshCommons.GetConditionStatus(string(vd.LifecycleState)), string(meshCommons.DependenciesNotResolved), err.Error())
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}

	service, err := h.resolveServiceRefK8s(ctx, vdb)
	if err != nil {
		_ = h.updateServiceMeshConditionWithServiceCondition(ctx, vdb, service, string(meshCommons.DependenciesNotResolved), err.Error())
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}
	if err := h.validateService(service); err != nil {
		_ = h.updateServiceMeshConditionWithServiceCondition(ctx, vdb, service, string(meshCommons.DependenciesNotResolved), err.Error())
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}

	// Update the dependencies status condition as resolved
	err = h.updateServiceMeshConditionWithVDCondition(ctx, vdb, metav1.ConditionTrue, string(meshCommons.Successful), string(meshCommons.DependenciesResolved))
	if err != nil {
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}

	if err := h.updateCRCp(ctx, vdb, vd); err != nil {
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}

	return h.evictPodsForVirtualDeploymentBinding(ctx, vdb, service)
}

// reconciles the VDB CR when VD reference is present
func (h *VirtualDeploymentBindingServiceManager) reconcileK8s(ctx context.Context, vdb *servicemeshapi.VirtualDeploymentBinding) (servicemanager.OSOKResponse, error) {
	vd, err := h.resolveVirtualDeploymentK8s(ctx, vdb)
	if err != nil {
		_ = h.updateServiceMeshConditionWithVirtualDeploymentK8s(ctx, vdb, meshCommons.GetConditionStatusFromK8sError(err), string(meshCommons.DependenciesNotResolved), err.Error())
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}
	if err := h.validateVirtualDeploymentK8s(vd); err != nil {
		_ = h.updateServiceMeshConditionWithVirtualDeploymentK8s(ctx, vdb, vdpkg.GetVDActiveStatus(vd), string(meshCommons.DependenciesNotResolved), err.Error())
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}

	service, err := h.resolveServiceRefK8s(ctx, vdb)
	if err != nil {
		_ = h.updateServiceMeshConditionWithServiceK8s(ctx, vdb, service, string(meshCommons.DependenciesNotResolved), err.Error())
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}
	if err := h.validateService(service); err != nil {
		_ = h.updateServiceMeshConditionWithServiceK8s(ctx, vdb, service, string(meshCommons.DependenciesNotResolved), err.Error())
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}

	// Update the dependencies status condition as resolved
	err = h.updateServiceMeshConditionWithVirtualDeploymentK8s(ctx, vdb, metav1.ConditionTrue, string(meshCommons.Successful), string(meshCommons.DependenciesResolved))
	if err != nil {
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}

	if err := h.updateCRK8s(ctx, vdb, vd); err != nil {
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}

	return h.evictPodsForVirtualDeploymentBinding(ctx, vdb, service)
}

func (h *VirtualDeploymentBindingServiceManager) resolveVirtualDeploymentCp(ctx context.Context, vdb *servicemeshapi.VirtualDeploymentBinding) (*sdk.VirtualDeployment, error) {
	return h.meshClient.GetVirtualDeployment(ctx, &vdb.Spec.VirtualDeployment.Id)
}

func (h *VirtualDeploymentBindingServiceManager) updateCRCp(ctx context.Context, vdb *servicemeshapi.VirtualDeploymentBinding, vd *sdk.VirtualDeployment) error {
	oldVDB := vdb.DeepCopy()
	needsUpdate := false
	if vdb.Status.VirtualDeploymentId != api.OCID(*vd.Id) {
		vdb.Status.VirtualDeploymentId = api.OCID(*vd.Id)
		vdb.Status.VirtualServiceId = api.OCID(*vd.VirtualServiceId)
		vdb.Status.VirtualDeploymentName = servicemeshapi.Name(*vd.Name)
		virtualServiceRef, err := h.referenceResolver.ResolveVirtualServiceRefById(ctx, conversions.OCID(*vd.VirtualServiceId))
		if err != nil {
			return err
		}
		vdb.Status.VirtualServiceName = virtualServiceRef.Name
		vdb.Status.MeshId = virtualServiceRef.MeshId
		needsUpdate = true
	}

	state := string(vd.LifecycleState)
	status := meshCommons.GetConditionStatus(state)
	reason := string(meshCommons.GetVirtualDeploymentBindingConditionReason(status))
	message := string(meshCommons.GetVirtualDeploymentBindingConditionMessage(state))
	if meshCommons.UpdateServiceMeshCondition(&vdb.Status, servicemeshapi.ServiceMeshActive, status, reason, message, vdb.Generation) {
		needsUpdate = true
	}

	if needsUpdate {
		if err := h.client.Status().Patch(ctx, vdb, client.MergeFrom(oldVDB)); err != nil {
			return err
		}
	}

	// Requeue request if state is unknown
	if status == metav1.ConditionUnknown {
		return meshErrors.NewRequeueOnError(errors.New(meshCommons.UnknownStatus))
	}

	// For any other state, don't requeue
	return nil
}

func (h *VirtualDeploymentBindingServiceManager) updateCRStatus(ctx context.Context, vdb *servicemeshapi.VirtualDeploymentBinding, err error) {
	if serviceError, ok := err.(common.ServiceError); ok {
		_ = h.updateServiceMeshConditionWithVDCondition(ctx, vdb, meshErrors.GetConditionStatus(serviceError), meshErrors.ResponseStatusText(serviceError), meshErrors.GetErrorMessage(serviceError))
	}
}

func (h *VirtualDeploymentBindingServiceManager) validateVirtualDeploymentK8s(vd *servicemeshapi.VirtualDeployment) error {
	if !vdpkg.IsVdActiveK8s(vd) {
		return meshErrors.NewRequeueOnError(errors.New("virtual deployment is not active yet"))
	}
	// if vdId not set, trigger reconcile again
	if vd.Status.VirtualDeploymentId == "" {
		return meshErrors.NewRequeueOnError(errors.New("virtualDeployment active, and virtualDeploymentId is not set"))
	}
	return nil
}

func (h *VirtualDeploymentBindingServiceManager) validateVirtualDeploymentCp(vd *sdk.VirtualDeployment) error {
	if !vdpkg.IsVdActiveCp(vd) {
		return meshErrors.NewRequeueOnError(errors.New("virtual deployment is not active yet"))
	}
	return nil
}

func (h *VirtualDeploymentBindingServiceManager) validateService(service *corev1.Service) error {
	if !servicepkg.IsServiceActive(service) {
		return meshErrors.NewRequeueOnError(errors.New("service is not active yet"))
	}
	return nil
}

func (h *VirtualDeploymentBindingServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	vdb, err := verifyEntity(obj)
	if err != nil {
		return false, err
	}

	if core.HasFinalizer(vdb, meshCommons.IngressGatewayRouteTableFinalizer) {
		if err = h.removeFinalizer(ctx, vdb, meshCommons.IngressGatewayRouteTableFinalizer); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (h *VirtualDeploymentBindingServiceManager) updateServiceMeshConditionWithVDCondition(ctx context.Context, vdb *servicemeshapi.VirtualDeploymentBinding, status metav1.ConditionStatus, reason string, message string) error {
	oldVdb := vdb.DeepCopy()
	if !meshCommons.UpdateServiceMeshCondition(&vdb.Status, servicemeshapi.ServiceMeshDependenciesActive, status, reason, message, h.getGeneration(vdb)) {
		return nil
	}
	return h.client.Status().Patch(ctx, vdb, client.MergeFrom(oldVdb))
}

func (h *VirtualDeploymentBindingServiceManager) updateServiceMeshConditionWithServiceCondition(ctx context.Context, vdb *servicemeshapi.VirtualDeploymentBinding, service *corev1.Service, reason string, message string) error {
	oldVdb := vdb.DeepCopy()
	status := metav1.ConditionUnknown
	// TODO: check the service status upon migration to kube v1.20
	if !meshCommons.UpdateServiceMeshCondition(&vdb.Status, servicemeshapi.ServiceMeshDependenciesActive, status, reason, message, h.getGeneration(vdb)) {
		return nil
	}
	return h.client.Status().Patch(ctx, vdb, client.MergeFrom(oldVdb))
}

func (h *VirtualDeploymentBindingServiceManager) evictPodsForVirtualDeploymentBinding(ctx context.Context, vdb *servicemeshapi.VirtualDeploymentBinding, service *corev1.Service) (servicemanager.OSOKResponse, error) {
	h.log.InfoLogWithFixedMessage(ctx, "evicting pods for vdb")
	namespaceName := vdb.Spec.Target.Service.ServiceRef.Namespace
	if namespaceName == "" {
		namespaceName = vdb.Namespace
	}

	namespace := &corev1.Namespace{}
	if err := h.client.Get(ctx, types.NamespacedName{Name: namespaceName, Namespace: ""}, namespace); err != nil {
		if kerrors.IsNotFound(err) {
			h.log.InfoLogWithFixedMessage(ctx, "Namespace was deleted", "namespaceName", namespaceName)
			return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: meshCommons.RequeueSyncDuration}, nil
		}
		h.log.ErrorLogWithFixedMessage(ctx, err, "Error reading Namespace object", "namespaceName", namespaceName)
		return meshErrors.GetOsokResponseByHandlingReconcileError(meshErrors.NewRequeueOnError(err))
	}

	namespaceLabels := labels.Set(namespace.Labels)
	if !namespaceLabels.Has(meshCommons.ProxyInjectionLabel) {
		h.log.InfoLogWithFixedMessage(ctx, "Service not part of a mesh enabled namespace")
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: meshCommons.RequeueSyncDuration}, nil
	}
	namespaceInjectionLabel := namespaceLabels.Get(meshCommons.ProxyInjectionLabel)

	labelSelectors := labels.Merge(service.Spec.Selector, vdb.Spec.Target.Service.MatchLabels)
	podList, err := podpkg.ListPodsWithLabels(ctx, h.client, namespaceName, &labelSelectors)
	if err != nil {
		if kerrors.IsNotFound(err) {
			h.log.InfoLogWithFixedMessage(ctx, "no pods found for the VDB")
			return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: meshCommons.RequeueSyncDuration}, nil
		}
		h.log.ErrorLogWithFixedMessage(ctx, err, "error reading pods for VDB")
		return meshErrors.GetOsokResponseByHandlingReconcileError(meshErrors.NewRequeueOnError(err))
	}

	notEvictedPods := 0
	for i := range podList.Items {
		pod := podList.Items[i]
		if podpkg.IsPodContainingServiceMeshProxy(&pod) {
			continue
		}
		podLabels := labels.Set(pod.Labels)
		podInjectionLabel := podpkg.GetPodInjectionLabelValue(&podLabels)
		if podpkg.IsInjectionLabelEnabled(namespaceInjectionLabel, podInjectionLabel) {
			if err := podpkg.EvictPod(ctx, h.clientSet, &pod); err != nil {
				h.log.ErrorLogWithFixedMessage(ctx, err, "Error in eviction", "pod", pod.Name)
				notEvictedPods += 1
			} else {
				h.log.InfoLogWithFixedMessage(ctx, "pod eviction successful", "name", pod.Name)
			}
		}
	}
	if notEvictedPods > 0 {
		h.log.InfoLogWithFixedMessage(ctx, "Pods are yet to be evicted, Reconciling after a minute", "count", strconv.Itoa(notEvictedPods))
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: time.Minute}, nil
	}
	return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: meshCommons.RequeueSyncDuration}, nil
}

func (h *VirtualDeploymentBindingServiceManager) GetCrdStatus(obj runtime.Object) (*api.OSOKStatus, error) {
	return nil, nil
}

func (h *VirtualDeploymentBindingServiceManager) removeFinalizer(ctx context.Context, obj client.Object, finalizer string) error {
	oldObj := obj.DeepCopyObject()
	needsUpdate := false
	if controllerutil.ContainsFinalizer(obj, finalizer) {
		controllerutil.RemoveFinalizer(obj, finalizer)
		needsUpdate = true
	}
	if !needsUpdate {
		return nil
	}
	return h.client.Patch(ctx, obj, client.MergeFrom(oldObj))
}

func (h *VirtualDeploymentBindingServiceManager) resolveServiceRefK8s(ctx context.Context, vdb *servicemeshapi.VirtualDeploymentBinding) (*corev1.Service, error) {
	serviceRef := vdb.Spec.Target.Service.ServiceRef
	if serviceRef.Namespace == "" {
		serviceRef.Namespace = vdb.Namespace
	}
	return h.referenceResolver.ResolveServiceReference(ctx, &serviceRef)
}

func (h *VirtualDeploymentBindingServiceManager) resolveVirtualDeploymentK8s(ctx context.Context, vdb *servicemeshapi.VirtualDeploymentBinding) (*servicemeshapi.VirtualDeployment, error) {
	vdRef := vdb.Spec.VirtualDeployment.ResourceRef
	if vdRef.Namespace == "" {
		vdRef.Namespace = vdb.Namespace
	}
	return h.referenceResolver.ResolveVirtualDeploymentReference(ctx, vdRef)
}

func (h *VirtualDeploymentBindingServiceManager) updateServiceMeshConditionWithVirtualDeploymentK8s(ctx context.Context, vdb *servicemeshapi.VirtualDeploymentBinding, status metav1.ConditionStatus, reason string, message string) error {
	oldVdb := vdb.DeepCopy()
	if !meshCommons.UpdateServiceMeshCondition(&vdb.Status, servicemeshapi.ServiceMeshDependenciesActive, status, reason, message, h.getGeneration(vdb)) {
		return nil
	}
	return h.client.Status().Patch(ctx, vdb, client.MergeFrom(oldVdb))
}

func (h *VirtualDeploymentBindingServiceManager) updateServiceMeshConditionWithServiceK8s(ctx context.Context, vdb *servicemeshapi.VirtualDeploymentBinding, service *corev1.Service, reason string, message string) error {
	oldVdb := vdb.DeepCopy()
	status := metav1.ConditionUnknown
	if service != nil {
		status = servicepkg.GetServiceActiveStatus(service)
	}
	if !meshCommons.UpdateServiceMeshCondition(&vdb.Status, servicemeshapi.ServiceMeshDependenciesActive, status, reason, message, h.getGeneration(vdb)) {
		return nil
	}
	return h.client.Status().Patch(ctx, vdb, client.MergeFrom(oldVdb))
}

func (h *VirtualDeploymentBindingServiceManager) updateCRK8s(ctx context.Context, vdb *servicemeshapi.VirtualDeploymentBinding, vd *servicemeshapi.VirtualDeployment) error {
	oldVDB := vdb.DeepCopy()
	needsUpdate := false

	// Note: assuming cannot change meshId for a VD resource
	if vdb.Status.VirtualDeploymentId != vd.Status.VirtualDeploymentId {
		vdb.Status.VirtualDeploymentId = vd.Status.VirtualDeploymentId
		vdb.Status.MeshId = vd.Status.MeshId
		vdb.Status.VirtualServiceId = vd.Status.VirtualServiceId
		vdb.Status.VirtualServiceName = vd.Status.VirtualServiceName
		vdb.Status.VirtualDeploymentName = servicemeshapi.Name(*conversions.GetSpecName(vd.Spec.Name, &vd.ObjectMeta))
		needsUpdate = true
	}

	// Status of virtual deployment and service should be True at this point
	status := metav1.ConditionTrue
	if meshCommons.UpdateServiceMeshCondition(&vdb.Status, servicemeshapi.ServiceMeshActive, status, string(meshCommons.GetReason(status)), string(meshCommons.ResourceActiveVDB), vdb.Generation) {
		needsUpdate = true
	}

	if needsUpdate {
		if err := h.client.Status().Patch(ctx, vdb, client.MergeFrom(oldVDB)); err != nil {
			return err
		}
	}

	// Requeue request if state is unknown
	if status == metav1.ConditionUnknown {
		return meshErrors.NewRequeueOnError(errors.New(meshCommons.UnknownStatus))
	}

	// For any other state, don't requeue
	return nil
}

func (h *VirtualDeploymentBindingServiceManager) getGeneration(vdb *servicemeshapi.VirtualDeploymentBinding) int64 {
	existingCondition := meshCommons.GetServiceMeshCondition(&vdb.Status, servicemeshapi.ServiceMeshActive)
	var generation int64
	if existingCondition == nil {
		generation = 1
	} else {
		generation = vdb.Generation
	}
	return generation
}

func verifyEntity(object runtime.Object) (*servicemeshapi.VirtualDeploymentBinding, error) {
	vdb, ok := object.(*servicemeshapi.VirtualDeploymentBinding)
	if !ok {
		return nil, errors.New("object is not a virtual deployment binding")
	}
	return vdb, nil
}
