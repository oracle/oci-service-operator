/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgatewaydeployment

import (
	"context"
	"errors"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/controllers/servicemesh.oci"
	"github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	customCache "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/k8s/cache"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
)

const IngressGatewaySecretsMountPath = "/etc/oci/secrets" // #nosec G101

type IngressGatewayDeploymentServiceManager struct {
	client            client.Client
	log               loggerutil.OSOKLogger
	clientSet         kubernetes.Interface
	referenceResolver references.Resolver
	Caches            customCache.CacheMapClient
	namespace         string
}

//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=ingressgatewaydeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=ingressgatewaydeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=ingressgatewaydeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;
//+kubebuilder:rbac:groups="",resources=configmaps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch

func NewIngressGatewayDeploymentServiceManager(client client.Client, log loggerutil.OSOKLogger, clientSet kubernetes.Interface,
	referenceResolver references.Resolver, caches customCache.CacheMapClient, namespace string) *IngressGatewayDeploymentServiceManager {
	return &IngressGatewayDeploymentServiceManager{
		client:            client,
		log:               log,
		clientSet:         clientSet,
		referenceResolver: referenceResolver,
		Caches:            caches,
		namespace:         namespace,
	}
}

func GetIngressGatewayDeploymentOwnerWatch(srcType client.Object) servicemeshoci.CustomWatch {
	return servicemeshoci.CustomWatch{
		Src: &source.Kind{Type: srcType},
		EventHandler: &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &servicemeshapi.IngressGatewayDeployment{},
		},
	}
}

func (h *IngressGatewayDeploymentServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	igd, err := verifyEntity(obj)
	if err != nil {
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}
	oldIgd := igd.DeepCopy()
	ingressGatewayRef := meshCommons.ResourceRef{
		Id:     igd.Status.IngressGatewayId,
		Name:   igd.Status.IngressGatewayName,
		MeshId: igd.Status.MeshId,
	}

	h.log.InfoLogWithFixedMessage(ctx, "Start Reconcile")

	if ingressGatewayRef.Id == "" {
		gatewayRef, err := h.referenceResolver.ResolveIngressGatewayIdAndNameAndMeshId(ctx, &igd.Spec.IngressGateway, &igd.ObjectMeta)
		h.UpdateServiceMeshDependenciesActiveStatus(ctx, igd, err)
		if err != nil {
			h.log.ErrorLogWithFixedMessage(ctx, err, "Failed to resolve Ingress gateway")
			return meshErrors.GetOsokResponseByHandlingReconcileError(err)
		}
		ingressGatewayRef = *gatewayRef
	}
	h.log.InfoLogWithFixedMessage(ctx, "Create Deployment")

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       igd.Namespace,
			Name:            igd.Name + string(meshCommons.NativeDeployment),
			OwnerReferences: h.createOwnerReferences(igd),
		},
	}

	opResult, err := controllerutil.CreateOrUpdate(ctx, h.client, &deployment, func() error {
		deploymentSpec, err := h.createDeploymentSpec(igd, ingressGatewayRef)
		if err != nil {
			h.log.ErrorLogWithFixedMessage(ctx, err, "Failed to build Deployment Spec")
			h.UpdateServiceMeshActiveStatus(ctx, igd, err)
			return err
		}
		deployment.Spec = *deploymentSpec

		return nil
	})

	if err != nil {
		h.log.ErrorLogWithFixedMessage(ctx, err, "Failed to create or update Deployment")
		h.UpdateServiceMeshActiveStatus(ctx, igd, err)
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}
	h.log.InfoLogWithFixedMessage(ctx, "Deployment ", "Status", string(opResult))
	h.log.InfoLogWithFixedMessage(ctx, "Creating Horizontal Pod Auto scalar")

	autoscaler := autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       igd.Namespace,
			Name:            igd.Name + string(meshCommons.NativeHorizontalPodAutoScalar),
			OwnerReferences: h.createOwnerReferences(igd),
		},
	}

	opResult, err = controllerutil.CreateOrUpdate(ctx, h.client, &autoscaler, func() error {
		podScalerSpec, err := h.createPodAutoScalerSpec(igd, deployment.Name)
		if err != nil {
			h.log.ErrorLogWithFixedMessage(ctx, err, "Failed to build Pod Scalar Object")
			h.UpdateServiceMeshActiveStatus(ctx, igd, err)
			return err
		}

		autoscaler.Spec = *podScalerSpec
		return nil
	})
	if err != nil {
		h.log.ErrorLogWithFixedMessage(ctx, err, "Failed to create or update  Horizontal Pod Auto scalar")
		h.UpdateServiceMeshActiveStatus(ctx, igd, err)
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}
	h.log.InfoLogWithFixedMessage(ctx, "Created  Horizontal Pod Auto scalar", "Status", string(opResult))

	// Create service if IGD has service object
	if igd.Spec.Service != nil {
		h.log.InfoLogWithFixedMessage(ctx, "Creating Service")
		service := corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       igd.Namespace,
				Name:            igd.Name + string(meshCommons.NativeService),
				OwnerReferences: h.createOwnerReferences(igd),
			},
		}
		opResult, err := controllerutil.CreateOrUpdate(ctx, h.client, &service, func() error {
			portSlice, err := h.getServicePorts(igd)
			if err != nil {
				h.log.ErrorLogWithFixedMessage(ctx, err, "Failed to build service Object "+string(opResult))
				h.UpdateServiceMeshActiveStatus(ctx, igd, err)
				return err
			}

			service.Spec.Ports = portSlice
			service.Spec.Type = igd.Spec.Service.Type
			service.Spec.Selector = map[string]string{
				meshCommons.IngressName: igd.Name,
			}
			service.Annotations = igd.Spec.Service.Annotations
			service.Labels = igd.Spec.Service.Labels

			return nil
		})
		if err != nil {
			h.log.ErrorLogWithFixedMessage(ctx, err, "Failed to create or update Service "+string(opResult))
			h.UpdateServiceMeshActiveStatus(ctx, igd, err)
			return meshErrors.GetOsokResponseByHandlingReconcileError(err)
		}
	}

	h.log.InfoLogWithFixedMessage(ctx, "Created Service ", "Status", string(opResult))
	return h.updateCR(ctx, igd, oldIgd, ingressGatewayRef)
}

func (h *IngressGatewayDeploymentServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	igd, err := verifyEntity(obj)
	if err != nil {
		return false, err
	}

	if core.HasFinalizer(igd, meshCommons.IngressGatewayRouteTableFinalizer) {
		if err = h.removeFinalizer(ctx, igd, meshCommons.IngressGatewayRouteTableFinalizer); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (h *IngressGatewayDeploymentServiceManager) updateCR(ctx context.Context, igd *servicemeshapi.IngressGatewayDeployment, oldIgd *servicemeshapi.IngressGatewayDeployment, ingressGatewayRef meshCommons.ResourceRef) (servicemanager.OSOKResponse, error) {
	needsUpdate := false
	if igd.Status.MeshId == "" {
		igd.Status.MeshId = ingressGatewayRef.MeshId
		needsUpdate = true
	}
	if igd.Status.IngressGatewayId != ingressGatewayRef.Id {
		igd.Status.IngressGatewayId = ingressGatewayRef.Id
		igd.Status.IngressGatewayName = ingressGatewayRef.Name
		needsUpdate = true
	}
	if meshCommons.UpdateServiceMeshCondition(&igd.Status, servicemeshapi.ServiceMeshActive, metav1.ConditionTrue, string(meshCommons.GetReason(metav1.ConditionTrue)), string(meshCommons.GetMessage(meshCommons.Active)), igd.Generation) {
		needsUpdate = true
	}

	if needsUpdate {
		if err := h.client.Status().Patch(ctx, igd, client.MergeFrom(oldIgd)); err != nil {
			return meshErrors.GetOsokResponseByHandlingReconcileError(err)
		}
	}

	return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: meshCommons.RequeueSyncDuration}, nil
}

func (h *IngressGatewayDeploymentServiceManager) getServicePorts(ingressGatewayDeployment *servicemeshapi.IngressGatewayDeployment) ([]corev1.ServicePort, error) {
	if ingressGatewayDeployment.Spec.Service == nil {
		return nil, nil
	}
	h.log.InfoLogWithFixedMessage(nil, "Creation of Service for Ingress is in Progress")

	portMap := make(map[corev1.Protocol][]corev1.ServicePort)
	for _, port := range ingressGatewayDeployment.Spec.Ports {
		if port.ServicePort == nil {
			continue
		}
		if portMap[port.Protocol] == nil {
			portMap[port.Protocol] = make([]corev1.ServicePort, 0)
		}

		portMap[port.Protocol] = append(portMap[port.Protocol], corev1.ServicePort{
			Protocol: port.Protocol,
			Name:     port.Name,
			Port:     *port.ServicePort,
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: *port.Port,
			},
		})
	}

	if len(portMap) == 0 {
		return nil, nil
	}

	if len(portMap) > 1 {
		return nil, errors.New("creation of Service Failed: Multiple ports with different protocols cannot be used")
	}

	portsSlice := make([]corev1.ServicePort, 0)
	for _, portsArr := range portMap {
		portsSlice = append(portsSlice, portsArr...)
	}

	return portsSlice, nil
}

func (h *IngressGatewayDeploymentServiceManager) createPodAutoScalerSpec(ingressGatewayDeployment *servicemeshapi.IngressGatewayDeployment, deploymentName string) (*autoscalingv1.HorizontalPodAutoscalerSpec, error) {

	if ingressGatewayDeployment.Spec.Deployment.Autoscaling == nil {
		return nil, nil
	}

	targetCPUUtilizationPercentage := int32(meshCommons.TargetCPUUtilizationPercentage)
	maxReplicas := ingressGatewayDeployment.Spec.Deployment.Autoscaling.MaxPods
	minReplicas := ingressGatewayDeployment.Spec.Deployment.Autoscaling.MinPods

	if minReplicas > maxReplicas {
		return nil, errors.New("min replica count cannot be greater than max replica count")
	}

	autoScalarSpec := autoscalingv1.HorizontalPodAutoscalerSpec{
		TargetCPUUtilizationPercentage: &targetCPUUtilizationPercentage,
		MaxReplicas:                    maxReplicas,
		MinReplicas:                    &minReplicas,
		ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
			Kind:       "Deployment",
			Name:       deploymentName,
			APIVersion: meshCommons.DeploymentAPIVersion,
		},
	}
	return &autoScalarSpec, nil
}

func (h *IngressGatewayDeploymentServiceManager) createDeploymentSpec(ingressGatewayDeployment *servicemeshapi.IngressGatewayDeployment, ingressGatewayRef meshCommons.ResourceRef) (*appsv1.DeploymentSpec, error) {
	configMap, keyerr := h.Caches.GetConfigMapByKey(h.namespace + "/" + meshCommons.MeshConfigMapName)
	if keyerr != nil {
		h.log.ErrorLogWithFixedMessage(nil, keyerr, "Failed to fetch configmap")
		return nil, keyerr
	}
	envoyImage, envoyImagePresent := configMap.Data[meshCommons.ProxyLabelInMeshConfigMap]
	if !envoyImagePresent {
		return nil, errors.New("no sidecar image found in mesh config map")
	}
	portSlice := []corev1.ContainerPort{
		{
			ContainerPort: meshCommons.StatsPort,
		},
	}
	for _, port := range ingressGatewayDeployment.Spec.Ports {
		portSlice = append(portSlice, corev1.ContainerPort{
			Protocol:      port.Protocol,
			Name:          port.Name,
			ContainerPort: *port.Port,
		})
	}

	secretVolumes := make([]corev1.Volume, 0, len(ingressGatewayDeployment.Spec.Secrets))
	for _, secret := range ingressGatewayDeployment.Spec.Secrets {
		secretVolumes = append(secretVolumes, corev1.Volume{
			Name: secret.SecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secret.SecretName,
				},
			},
		})
	}

	volumeMounts := make([]corev1.VolumeMount, 0, len(secretVolumes))
	for _, volume := range secretVolumes {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volume.Name,
			ReadOnly:  true,
			MountPath: IngressGatewaySecretsMountPath + "/" + volume.Name,
		})
	}

	volumes := secretVolumes
	if ingressGatewayDeployment.Spec.Deployment.MountCertificateChainFromHost != nil &&
		*ingressGatewayDeployment.Spec.Deployment.MountCertificateChainFromHost {
		volumes = append(secretVolumes, meshCommons.PkiVolume)
		volumeMounts = append(volumeMounts, meshCommons.PkiVolumeMount)
	}

	resourceRequests := corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse(string(meshCommons.IngressCPURequestSize)),
			corev1.ResourceMemory: resource.MustParse(string(meshCommons.IngressMemoryRequestSize)),
		},
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse(string(meshCommons.IngressCPULimitSize)),
			corev1.ResourceMemory: resource.MustParse(string(meshCommons.IngressMemoryLimitSize)),
		},
	}

	if ingressGatewayDeployment.Spec.Deployment.Autoscaling != nil && ingressGatewayDeployment.Spec.Deployment.Autoscaling.Resources != nil {
		resourceRequests = *ingressGatewayDeployment.Spec.Deployment.Autoscaling.Resources
	}

	annotations := ingressGatewayDeployment.Annotations
	proxyLogLevel, ok := annotations[meshCommons.ProxyLogLevelAnnotation]
	if !ok {
		proxyLogLevel = string(meshCommons.DefaultProxyLogLevel)
	}

	// Validate the log level present
	if !(meshCommons.IsStringPresent(meshCommons.ProxyLogLevels, strings.ToLower(proxyLogLevel))) {
		return nil, errors.New(string(meshCommons.InValidProxyLogAnnotation))
	}

	proxyContainer := corev1.Container{
		Name:  meshCommons.ProxyContainerName,
		Image: envoyImage,
		Ports: portSlice,
		Env: []corev1.EnvVar{
			{
				Name:  string(meshCommons.DeploymentId),
				Value: string(ingressGatewayRef.Id),
			},
			{
				Name:  string(meshCommons.ProxyLogLevel),
				Value: strings.ToLower(proxyLogLevel),
			},
			// this environment variable is deprecated in favor of POD_IP due to name being very generic
			{
				Name: string(meshCommons.IPAddress),
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			},
			{
				Name: string(meshCommons.PodIp),
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			},
			{
				Name: string(meshCommons.PodUId),
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.uid",
					},
				},
			},
			{
				Name: string(meshCommons.PodName),
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			{
				Name: string(meshCommons.PodNamespace),
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
		},
		Resources:    resourceRequests,
		VolumeMounts: volumeMounts,
	}

	if configMap.Data[meshCommons.MdsEndpointInMeshConfigMap] != "" {
		proxyContainer.Env = append(proxyContainer.Env, corev1.EnvVar{Name: meshCommons.MdsEndpointInMeshConfigMap, Value: configMap.Data[meshCommons.MdsEndpointInMeshConfigMap]})
	}

	deploymentSpec := appsv1.DeploymentSpec{
		Replicas: &ingressGatewayDeployment.Spec.Deployment.Autoscaling.MinPods,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				meshCommons.IngressName: ingressGatewayDeployment.Name,
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					meshCommons.IngressName: ingressGatewayDeployment.Name,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					proxyContainer,
				},
				Volumes: volumes,
			},
		},
	}

	return &deploymentSpec, nil

}

func (h *IngressGatewayDeploymentServiceManager) createOwnerReferences(ingressGatewayDeployment *servicemeshapi.IngressGatewayDeployment) []metav1.OwnerReference {
	// If true, AND if the owner has the "foregroundDeletion" finalizer, then
	// the owner cannot be deleted from the key-value store until this
	// reference is removed.
	blockOwnerDeletion := true

	// If true, this reference points to the managing controller.
	controllerPoint := true
	return []metav1.OwnerReference{
		{
			Kind:               ingressGatewayDeployment.Kind,
			Name:               ingressGatewayDeployment.Name,
			BlockOwnerDeletion: &blockOwnerDeletion,
			Controller:         &controllerPoint,
			UID:                ingressGatewayDeployment.UID,
			APIVersion:         ingressGatewayDeployment.APIVersion,
		},
	}
}

func (h *IngressGatewayDeploymentServiceManager) GetCrdStatus(obj runtime.Object) (*api.OSOKStatus, error) {
	return nil, nil
}

func (h *IngressGatewayDeploymentServiceManager) removeFinalizer(ctx context.Context, obj client.Object, finalizer string) error {
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

// UpdateServiceMeshCondition updates the status of the resource
// Called when there is an error from CP
// Additionally, also called when the state is not Active
// We haven't fully synced the spec, hence using the existing generation from the status for generation field
func (h *IngressGatewayDeploymentServiceManager) UpdateServiceMeshCondition(ctx context.Context, igd *servicemeshapi.IngressGatewayDeployment, status metav1.ConditionStatus, reason string, message string, meshConditionType servicemeshapi.ServiceMeshConditionType) error {
	oldIgd := igd.DeepCopy()
	existingCondition := meshCommons.GetServiceMeshCondition(&igd.Status, meshConditionType)
	var generation int64
	if existingCondition == nil {
		generation = 1
	} else {
		generation = igd.Generation
	}
	if !meshCommons.UpdateServiceMeshCondition(&igd.Status, meshConditionType, status, reason, message, generation) {
		return nil
	}
	return h.client.Status().Patch(ctx, igd, client.MergeFrom(oldIgd))
}

func (h *IngressGatewayDeploymentServiceManager) UpdateServiceMeshActiveStatus(ctx context.Context, igd *servicemeshapi.IngressGatewayDeployment, err error) {
	if serviceError, ok := err.(common.ServiceError); ok {
		_ = h.UpdateServiceMeshCondition(ctx, igd, meshErrors.GetConditionStatus(serviceError), meshErrors.ResponseStatusText(serviceError), meshErrors.GetErrorMessage(serviceError), servicemeshapi.ServiceMeshActive)
	} else {
		_ = h.UpdateServiceMeshCondition(ctx, igd, meshCommons.GetConditionStatusFromK8sError(err), string(meshCommons.DependenciesNotResolved), err.Error(), servicemeshapi.ServiceMeshActive)
	}
}

func (h *IngressGatewayDeploymentServiceManager) UpdateServiceMeshDependenciesActiveStatus(ctx context.Context, igd *servicemeshapi.IngressGatewayDeployment, err error) {
	if err == nil {
		_ = h.UpdateServiceMeshCondition(ctx, igd, metav1.ConditionTrue, string(meshCommons.Successful), string(meshCommons.DependenciesResolved), servicemeshapi.ServiceMeshDependenciesActive)
		return
	}
	if serviceError, ok := err.(common.ServiceError); ok {
		_ = h.UpdateServiceMeshCondition(ctx, igd, meshErrors.GetConditionStatus(serviceError), meshErrors.ResponseStatusText(serviceError), meshErrors.GetErrorMessage(serviceError), servicemeshapi.ServiceMeshDependenciesActive)
	} else {
		_ = h.UpdateServiceMeshCondition(ctx, igd, meshCommons.GetConditionStatusFromK8sError(err), string(meshCommons.DependenciesNotResolved), err.Error(), servicemeshapi.ServiceMeshDependenciesActive)
	}
}

func verifyEntity(object runtime.Object) (*servicemeshapi.IngressGatewayDeployment, error) {
	igd, ok := object.(*servicemeshapi.IngressGatewayDeployment)
	if !ok {
		return nil, errors.New("object is not a ingress gateway deployment")
	}
	return igd, nil
}
