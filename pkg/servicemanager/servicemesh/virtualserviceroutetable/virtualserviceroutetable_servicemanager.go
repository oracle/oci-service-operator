/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualserviceroutetable

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

//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=virtualserviceroutetables,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=virtualserviceroutetables/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=virtualserviceroutetables/finalizers,verbs=update

func NewVirtualServiceRouteTableResourceManager(client client.Client, serviceMeshClient services.ServiceMeshClient,
	log loggerutil.OSOKLogger, referenceResolver references.Resolver) manager.CustomResourceHandler {
	return &ResourceManager{
		client:            client,
		serviceMeshClient: serviceMeshClient,
		log:               log,
		referenceResolver: referenceResolver,
	}
}

func (m *ResourceManager) GetResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	vsrt, err := getVirtualServiceRouteTable(object)
	if err != nil {
		return err
	}
	if len(vsrt.Status.VirtualServiceRouteTableId) > 0 {
		sdkVsrt, err := m.serviceMeshClient.GetVirtualServiceRouteTable(ctx, &vsrt.Status.VirtualServiceRouteTableId)
		if err != nil {
			return err
		}
		details.VsrtDetails.SdkVsrt = sdkVsrt
		return nil
	}
	m.log.InfoLogWithFixedMessage(ctx, "virtual service route table did not sync to the control plane", "name", vsrt.ObjectMeta.Name)
	return nil
}

func (m *ResourceManager) CreateResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) (bool, error) {
	if details.VsrtDetails.SdkVsrt == nil {
		m.log.InfoLogWithFixedMessage(ctx, "creating virtualServiceRouteTable...", "name", object.GetName(), "OpcRetryToken", *details.OpcRetryToken)
		sdkVsrt, err := m.serviceMeshClient.CreateVirtualServiceRouteTable(ctx, details.VsrtDetails.BuildSdkVsrt, details.OpcRetryToken)
		if err != nil {
			return false, err
		}
		details.VsrtDetails.SdkVsrt = sdkVsrt
		return false, nil
	}
	return true, nil
}

func (m *ResourceManager) UpdateResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	m.log.InfoLogWithFixedMessage(ctx, "updating virtualServiceRouteTable...", "name", object.GetName())
	details.VsrtDetails.SdkVsrt.LifecycleState = sdk.VirtualServiceRouteTableLifecycleStateUpdating
	return m.serviceMeshClient.UpdateVirtualServiceRouteTable(ctx, details.VsrtDetails.BuildSdkVsrt)
}

func (m *ResourceManager) ChangeCompartment(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	vsrt, err := getVirtualServiceRouteTable(object)
	if err != nil {
		return err
	}
	m.log.InfoLogWithFixedMessage(ctx, "Moving virtual service route table to new compartment", "Name", object.GetName())
	// sdkVirtualServiceRouteTable state would be Active here, hence update the state to updating for correct status
	details.VsrtDetails.SdkVsrt.LifecycleState = sdk.VirtualServiceRouteTableLifecycleStateUpdating
	return m.serviceMeshClient.ChangeVirtualServiceRouteTableCompartment(ctx, &vsrt.Status.VirtualServiceRouteTableId, &vsrt.Spec.CompartmentId)
}

func (m *ResourceManager) DeleteResource(ctx context.Context, object client.Object) error {
	vsrt, err := getVirtualServiceRouteTable(object)
	if err != nil {
		return err
	}
	if len(vsrt.Status.VirtualServiceRouteTableId) == 0 {
		return nil
	}
	m.log.InfoLogWithFixedMessage(ctx, "Deleting virtual service route table", "Name", object.GetName())
	return m.serviceMeshClient.DeleteVirtualServiceRouteTable(ctx, &vsrt.Status.VirtualServiceRouteTableId)
}

func (m *ResourceManager) VerifyEntityType(object runtime.Object) (client.Object, error) {
	vsrt, ok := object.(*servicemeshapi.VirtualServiceRouteTable)
	if !ok {
		return nil, errors.New("object is not a virtual service route table")
	}
	return vsrt, nil
}

func (m *ResourceManager) VerifyResourceStatus(details *manager.ResourceDetails) (bool, error) {
	if details.VsrtDetails.SdkVsrt == nil || details.VsrtDetails.SdkVsrt.LifecycleState == sdk.VirtualServiceRouteTableLifecycleStateActive {
		return true, nil
	}

	state := details.VsrtDetails.SdkVsrt.LifecycleState
	// Terminate the reconcile request if resource in the control plane is deleted or failed
	if state == sdk.VirtualServiceRouteTableLifecycleStateDeleted || state == sdk.VirtualServiceRouteTableLifecycleStateFailed {
		return false, meshErrors.NewDoNotRequeueError(errors.New("virtual service route table in the control plane is deleted or failed"))
	}

	return false, meshErrors.NewRequeueOnError(errors.New(meshCommons.UnknownStatus))
}

func (m *ResourceManager) GetServiceMeshStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	vsrt, err := getVirtualServiceRouteTable(object)
	if err != nil {
		return nil, err
	}
	return &vsrt.Status, nil
}

func (m *ResourceManager) GetConditionStatus(details *manager.ResourceDetails) metav1.ConditionStatus {
	state := details.VsrtDetails.SdkVsrt.LifecycleState
	return meshCommons.GetConditionStatus(string(state))
}

func (m *ResourceManager) UpdateStatus(object client.Object, details *manager.ResourceDetails) (bool, error) {
	needsUpdate := false
	vsrt, err := getVirtualServiceRouteTable(object)
	if err != nil {
		return false, err
	}
	if vsrt.Status.VirtualServiceId == "" {
		vsrt.Status.VirtualServiceId = api.OCID(*details.VsrtDetails.SdkVsrt.VirtualServiceId)
		vsrt.Status.VirtualServiceName = details.VsrtDetails.Dependencies.VirtualServiceName
		needsUpdate = true
	}

	if vsrt.Status.VirtualServiceRouteTableId != api.OCID(*details.VsrtDetails.SdkVsrt.Id) {
		vsrt.Status.VirtualServiceRouteTableId = api.OCID(*details.VsrtDetails.SdkVsrt.Id)
		needsUpdate = true
	}

	if !cmp.Equal(vsrt.Status.VirtualDeploymentIdForRules, details.VsrtDetails.Dependencies.VdIdForRules) {
		vsrt.Status.VirtualDeploymentIdForRules = details.VsrtDetails.Dependencies.VdIdForRules
		needsUpdate = true
	}

	// update operator time iff cp has different updated time
	// (we cannot change cp updated time)
	// so change operator updated time to match cp, so resource isnt reconsidered during next reconciliation
	cpTime := details.VsrtDetails.SdkVsrt.TimeUpdated
	operatorTime := vsrt.Status.LastUpdatedTime
	if manager.DoTimeStampsDiffer(cpTime, operatorTime) {
		vsrt.Status.LastUpdatedTime = (*metav1.Time)(details.VsrtDetails.SdkVsrt.TimeUpdated)
		needsUpdate = true
	}

	return needsUpdate, nil
}

func (m *ResourceManager) GetMessage(details *manager.ResourceDetails) string {
	state := details.VsrtDetails.SdkVsrt.LifecycleState
	return string(meshCommons.GetMessage(string(state)))
}

func (m *ResourceManager) BuildSdk(object client.Object, details *manager.ResourceDetails) error {
	buildSdkVsrt := &sdk.VirtualServiceRouteTable{}
	if details.VsrtDetails.SdkVsrt != nil {
		buildSdkVsrt.FreeformTags = details.VsrtDetails.SdkVsrt.FreeformTags
		buildSdkVsrt.DefinedTags = details.VsrtDetails.SdkVsrt.DefinedTags
	}
	vsrt, err := getVirtualServiceRouteTable(object)
	if err != nil {
		return err
	}
	err = meshConversions.ConvertCrdVsrtToSdkVsrt(vsrt, buildSdkVsrt, details.VsrtDetails.Dependencies)
	if err != nil {
		return err
	}
	details.VsrtDetails.BuildSdkVsrt = buildSdkVsrt
	return nil
}

func (m *ResourceManager) HasCompartmentIdChanged(object client.Object, details *manager.ResourceDetails) (bool, error) {
	vsrt, err := getVirtualServiceRouteTable(object)
	if err != nil {
		return false, err
	}
	return vsrt.Spec.CompartmentId != api.OCID(*details.VsrtDetails.SdkVsrt.CompartmentId), nil
}

func (m *ResourceManager) GetLifecycleState(details *manager.ResourceDetails) string {
	return string(details.VsrtDetails.SdkVsrt.LifecycleState)
}

func (m *ResourceManager) ResolveDependencies(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	vsrt, err := getVirtualServiceRouteTable(object)
	if err != nil {
		return err
	}
	dependencies := &meshConversions.VSRTDependencies{}
	dependencies.VirtualServiceId = vsrt.Status.VirtualServiceId
	dependencies.VirtualServiceName = vsrt.Status.VirtualServiceName
	// Resolve dependencies of VirtualService for VirtualServiceRouteTable
	if dependencies.VirtualServiceId == "" {
		virtualServiceRef, err := m.referenceResolver.ResolveVirtualServiceIdAndName(ctx, &vsrt.Spec.VirtualService, &vsrt.ObjectMeta)
		if err != nil {
			return err
		}
		dependencies.VirtualServiceId = virtualServiceRef.Id
		dependencies.VirtualServiceName = virtualServiceRef.Name
	}
	// Resolve dependencies of VirtualDeployment for VirtualServiceRouteTable
	dependencies.VdIdForRules = make([][]api.OCID, len(vsrt.Spec.RouteRules))
	for i, routeRule := range vsrt.Spec.RouteRules {
		vdIds, err := m.resolveRouteDependencies(ctx, routeRule, vsrt)
		if err != nil {
			return err
		}
		dependencies.VdIdForRules[i] = vdIds
	}
	details.VsrtDetails.Dependencies = dependencies
	return nil
}

func (m *ResourceManager) GetTimeUpdated(details *manager.ResourceDetails) *common.SDKTime {
	return details.VsrtDetails.SdkVsrt.TimeUpdated
}

func getVirtualDeploymentDestinations(rule servicemeshapi.VirtualServiceTrafficRouteRule) ([]servicemeshapi.VirtualDeploymentTrafficRuleTarget, error) {
	switch {
	case rule.HttpRoute != nil:
		return rule.HttpRoute.Destinations, nil
	case rule.TcpRoute != nil:
		return rule.TcpRoute.Destinations, nil
	case rule.TlsPassthroughRoute != nil:
		return rule.TlsPassthroughRoute.Destinations, nil
	default:
		return nil, errors.New(string(meshCommons.TrafficRouteRuleIsEmpty))
	}
}

func (m *ResourceManager) resolveRouteDependencies(ctx context.Context, routeRule servicemeshapi.VirtualServiceTrafficRouteRule, vsrt *servicemeshapi.VirtualServiceRouteTable) ([]api.OCID, error) {
	destinations, err := getVirtualDeploymentDestinations(routeRule)
	if err != nil {
		return nil, err
	}
	vdIds := make([]api.OCID, len(destinations))
	for j, destination := range destinations {
		virtualDeploymentId, err := m.referenceResolver.ResolveVirtualDeploymentId(ctx, destination.VirtualDeployment, &vsrt.ObjectMeta)
		if err != nil {
			return nil, err
		}
		vdIds[j] = *virtualDeploymentId
	}
	return vdIds, nil
}

func (m *ResourceManager) GetFinalizer() string {
	return meshCommons.VirtualServiceRouteTableFinalizer
}

func (m *ResourceManager) Finalize(ctx context.Context, object client.Object) error {
	return nil
}

func getVirtualServiceRouteTable(object client.Object) (*servicemeshapi.VirtualServiceRouteTable, error) {
	vsrt, ok := object.(*servicemeshapi.VirtualServiceRouteTable)
	if !ok {
		return nil, errors.New("object is not a virtual service route table")
	}
	return vsrt, nil
}

func (m *ResourceManager) HasSdk(details *manager.ResourceDetails) bool {
	return details.VsrtDetails.SdkVsrt != nil
}
