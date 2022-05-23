/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualservice

import (
	"context"
	"errors"

	"github.com/google/go-cmp/cmp"
	"github.com/oracle/oci-go-sdk/v65/common"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/services"
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	meshConversions "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceManager struct {
	client            client.Client
	serviceMeshClient services.ServiceMeshClient
	log               loggerutil.OSOKLogger
	referenceResolver references.Resolver
}

//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=virtualservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=virtualservices/finalizers,verbs=update

func NewVirtualServiceResourceManager(client client.Client, serviceMeshClient services.ServiceMeshClient,
	log loggerutil.OSOKLogger, referenceResolver references.Resolver) manager.CustomResourceHandler {
	return &ResourceManager{
		client:            client,
		serviceMeshClient: serviceMeshClient,
		log:               log,
		referenceResolver: referenceResolver,
	}
}

func (m *ResourceManager) GetResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	virtualService, err := getVirtualService(object)
	if err != nil {
		return err
	}
	if len(virtualService.Status.VirtualServiceId) > 0 {
		sdkVirtualService, err := m.serviceMeshClient.GetVirtualService(ctx, &virtualService.Status.VirtualServiceId)
		if err != nil {
			return err
		}
		details.VsDetails.SdkVs = sdkVirtualService
		return nil
	}
	m.log.InfoLogWithFixedMessage(ctx, "virtualService did not sync to the control plane", "name", virtualService.ObjectMeta.Name)
	return nil
}

func (m *ResourceManager) CreateResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) (bool, error) {
	if details.VsDetails.SdkVs == nil {
		m.log.InfoLogWithFixedMessage(ctx, "creating virtualService...", "name", object.GetName(), "OpcRetryToken", *details.OpcRetryToken)
		sdkVirtualService, err := m.serviceMeshClient.CreateVirtualService(ctx, details.VsDetails.BuildSdkVs, details.OpcRetryToken)
		if err != nil {
			return false, err
		}
		details.VsDetails.SdkVs = sdkVirtualService
		return false, nil
	}
	return true, nil
}

func (m *ResourceManager) UpdateResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	m.log.InfoLogWithFixedMessage(ctx, "updating virtualService...", "name", object.GetName())
	details.VsDetails.SdkVs.LifecycleState = sdk.VirtualServiceLifecycleStateUpdating
	return m.serviceMeshClient.UpdateVirtualService(ctx, details.VsDetails.BuildSdkVs)
}

func (m *ResourceManager) ChangeCompartment(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	virtualService, err := getVirtualService(object)
	if err != nil {
		return err
	}
	m.log.InfoLogWithFixedMessage(ctx, "Moving virtualService to new compartment", "Name", object.GetName())
	// sdkVirtualService state would be Active here, hence update the state to updating for correct status
	details.VsDetails.SdkVs.LifecycleState = sdk.VirtualServiceLifecycleStateUpdating
	return m.serviceMeshClient.ChangeVirtualServiceCompartment(ctx, &virtualService.Status.VirtualServiceId, &virtualService.Spec.CompartmentId)
}

func (m *ResourceManager) DeleteResource(ctx context.Context, object client.Object) error {
	virtualService, err := getVirtualService(object)
	if err != nil {
		return err
	}
	if len(virtualService.Status.VirtualServiceId) == 0 {
		return nil
	}
	m.log.InfoLogWithFixedMessage(ctx, "Deleting virtual service", "Name", object.GetName())
	return m.serviceMeshClient.DeleteVirtualService(ctx, &virtualService.Status.VirtualServiceId)
}

func (m *ResourceManager) VerifyEntityType(object runtime.Object) (client.Object, error) {
	virtualService, ok := object.(*servicemeshapi.VirtualService)
	if !ok {
		return nil, errors.New("object is not a virtual service")
	}
	return virtualService, nil
}

func (m *ResourceManager) VerifyResourceStatus(details *manager.ResourceDetails) (bool, error) {
	if details.VsDetails.SdkVs == nil || details.VsDetails.SdkVs.LifecycleState == sdk.VirtualServiceLifecycleStateActive {
		return true, nil
	}

	state := details.VsDetails.SdkVs.LifecycleState
	// Terminate the reconcile request if resource in the control plane is deleted or failed
	if state == sdk.VirtualServiceLifecycleStateDeleted || state == sdk.VirtualServiceLifecycleStateFailed {
		return false, meshErrors.NewDoNotRequeueError(errors.New("virtual service in the control plane is deleted or failed"))
	}

	return false, meshErrors.NewRequeueOnError(errors.New(meshCommons.UnknownStatus))
}

func (m *ResourceManager) GetServiceMeshStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	virtualService, err := getVirtualService(object)
	if err != nil {
		return nil, err
	}
	return &virtualService.Status, nil
}

func (m *ResourceManager) GetConditionStatus(details *manager.ResourceDetails) metav1.ConditionStatus {
	state := details.VsDetails.SdkVs.LifecycleState
	return meshCommons.GetConditionStatus(string(state))
}

func (m *ResourceManager) UpdateStatus(object client.Object, details *manager.ResourceDetails) (bool, error) {
	needsUpdate := false
	virtualService, err := getVirtualService(object)
	if err != nil {
		return false, err
	}
	// Update MeshId
	if virtualService.Status.MeshId == "" {
		virtualService.Status.MeshId = api.OCID(*details.VsDetails.SdkVs.MeshId)
		needsUpdate = true
	}
	// Update virtualServiceId
	if virtualService.Status.VirtualServiceId != api.OCID(*details.VsDetails.SdkVs.Id) {
		virtualService.Status.VirtualServiceId = api.OCID(*details.VsDetails.SdkVs.Id)
		needsUpdate = true
	}

	mtls, err := meshConversions.ConvertSdkVsMtlsToCrdVsMtls(details.VsDetails.SdkVs.Mtls)
	if err != nil {
		return false, err
	}
	if mtls != nil && !cmp.Equal(virtualService.Status.VirtualServiceMtls, mtls) {
		virtualService.Status.VirtualServiceMtls = mtls
		needsUpdate = true
	}

	// update operator time iff cp has different updated time
	// (we cannot change cp updated time)
	// so change operator updated time to match cp, so resource isnt reconsidered during next reconciliation
	cpTime := details.VsDetails.SdkVs.TimeUpdated
	operatorTime := virtualService.Status.LastUpdatedTime
	if manager.DoTimeStampsDiffer(cpTime, operatorTime) {
		virtualService.Status.LastUpdatedTime = (*metav1.Time)(details.VsDetails.SdkVs.TimeUpdated)
		needsUpdate = true
	}

	return needsUpdate, nil
}

func (m *ResourceManager) GetMessage(details *manager.ResourceDetails) string {
	state := details.VsDetails.SdkVs.LifecycleState
	return string(meshCommons.GetMessage(string(state)))
}

func (m *ResourceManager) BuildSdk(object client.Object, details *manager.ResourceDetails) error {
	buildSdkVirtualService := &sdk.VirtualService{}
	virtualService, err := getVirtualService(object)
	if err != nil {
		return err
	}
	err = meshConversions.ConvertCrdVirtualServiceToSdkVirtualService(virtualService, buildSdkVirtualService, details.VsDetails.MeshId)
	if err != nil {
		return err
	}
	details.VsDetails.BuildSdkVs = buildSdkVirtualService
	return nil
}

func (m *ResourceManager) HasCompartmentIdChanged(object client.Object, details *manager.ResourceDetails) (bool, error) {
	virtualService, err := getVirtualService(object)
	if err != nil {
		return false, err
	}
	return virtualService.Spec.CompartmentId != api.OCID(*details.VsDetails.SdkVs.CompartmentId), nil
}

func (m *ResourceManager) GetLifecycleState(details *manager.ResourceDetails) string {
	return string(details.VsDetails.SdkVs.LifecycleState)
}

func (m *ResourceManager) ResolveDependencies(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	virtualService, err := getVirtualService(object)
	if err != nil {
		return err
	}
	meshId := virtualService.Status.MeshId
	if meshId == "" {
		meshOcid, err := m.referenceResolver.ResolveMeshId(ctx, &virtualService.Spec.Mesh, &virtualService.ObjectMeta)
		details.VsDetails.MeshId = meshOcid
		return err
	}
	return nil
}

func (m *ResourceManager) GetTimeUpdated(details *manager.ResourceDetails) *common.SDKTime {
	return details.VsDetails.SdkVs.TimeUpdated
}

func (m *ResourceManager) GetFinalizer() string {
	return meshCommons.VirtualServiceFinalizer
}

func (m *ResourceManager) Finalize(ctx context.Context, object client.Object) error {
	virtualService, err := getVirtualService(object)
	if err != nil {
		return err
	}
	hasDependencies, err := m.hasVirtualServiceRouteTables(ctx, virtualService)
	if err != nil {
		return err
	}
	if hasDependencies {
		return errors.New("cannot delete virtual service when there are virtual service route table resources associated")
	}

	hasDependencies, err = m.hasIngressGatewayRouteTables(ctx, virtualService)
	if err != nil {
		return err
	}
	if hasDependencies {
		return errors.New("cannot delete virtual service when there are ingress gateway route table resources associated")
	}

	hasDependencies, err = m.hasAccessPolicies(ctx, virtualService)
	if err != nil {
		return err
	}
	if hasDependencies {
		return errors.New("cannot delete virtual service when there are access policy resources associated")
	}

	hasDependencies, err = m.hasVirtualDeployments(ctx, virtualService)
	if err != nil {
		return err
	}
	if hasDependencies {
		return errors.New("cannot delete virtual service when there are virtual deployment resources associated")
	}

	return nil
}

func (m *ResourceManager) hasVirtualServiceRouteTables(ctx context.Context, virtualService *servicemeshapi.VirtualService) (bool, error) {
	vsrtList := &servicemeshapi.VirtualServiceRouteTableList{}

	if err := m.client.List(ctx, vsrtList); err != nil {
		m.log.InfoLogWithFixedMessage(ctx, "unable to list virtualServiceRouteTables for virtual service", "name", virtualService.Name, "namespace", virtualService.Namespace)
		return false, err
	}

	hasDependencies := false
	for _, vsrt := range vsrtList.Items {
		if vsrt.Status.VirtualServiceId == virtualService.Status.VirtualServiceId {
			hasDependencies = true
			break
		}
	}
	return hasDependencies, nil
}

func (m *ResourceManager) hasIngressGatewayRouteTables(ctx context.Context, virtualService *servicemeshapi.VirtualService) (bool, error) {
	igrtList := &servicemeshapi.IngressGatewayRouteTableList{}

	if err := m.client.List(ctx, igrtList); err != nil {
		m.log.InfoLogWithFixedMessage(ctx, "unable to list ingressGatewayRouteTables for virtual service", "name", virtualService.Name, "namespace", virtualService.Namespace)
		return false, err
	}

	for _, igrt := range igrtList.Items {
		for _, rule := range igrt.Status.VirtualServiceIdForRules {
			for _, virtualServiceId := range rule {
				if virtualService.Status.VirtualServiceId == virtualServiceId {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func (m *ResourceManager) hasAccessPolicies(ctx context.Context, virtualService *servicemeshapi.VirtualService) (bool, error) {
	aplist := &servicemeshapi.AccessPolicyList{}

	if err := m.client.List(ctx, aplist); err != nil {
		m.log.InfoLogWithFixedMessage(ctx, "unable to list accessPolicies for virtual service", "name", virtualService.Name, "namespace", virtualService.Namespace)
		return false, err
	}

	for _, ap := range aplist.Items {
		if virtualService.Status.MeshId == ap.Status.MeshId {
			for _, refIdsForRule := range ap.Status.RefIdForRules {
				if virtualService.Status.VirtualServiceId == refIdsForRule[meshCommons.Source] || virtualService.Status.VirtualServiceId == refIdsForRule[meshCommons.Destination] {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func (m *ResourceManager) hasVirtualDeployments(ctx context.Context, virtualService *servicemeshapi.VirtualService) (bool, error) {
	vdList := &servicemeshapi.VirtualDeploymentList{}

	if err := m.client.List(ctx, vdList); err != nil {
		m.log.InfoLogWithFixedMessage(ctx, "unable to list virtualDeployments for virtual service", "name", virtualService.Name, "namespace", virtualService.Namespace)
		return false, err
	}

	hasDependencies := false
	for _, vd := range vdList.Items {
		if vd.Status.VirtualServiceId == virtualService.Status.VirtualServiceId {
			hasDependencies = true
			break
		}
	}
	return hasDependencies, nil
}

func getVirtualService(object client.Object) (*servicemeshapi.VirtualService, error) {
	virtualService, ok := object.(*servicemeshapi.VirtualService)
	if !ok {
		return nil, errors.New("object is not a virtual service")
	}
	return virtualService, nil
}

func (m *ResourceManager) HasSdk(details *manager.ResourceDetails) bool {
	return details.VsDetails.SdkVs != nil
}
