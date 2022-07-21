/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgatewayroutetable

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

//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=ingressgatewayroutetables,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=ingressgatewayroutetables/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=ingressgatewayroutetables/finalizers,verbs=update

func NewIngressGatewayRouteTableResourceManager(client client.Client, serviceMeshClient services.ServiceMeshClient,
	log loggerutil.OSOKLogger, referenceResolver references.Resolver) manager.CustomResourceHandler {
	return &ResourceManager{
		client:            client,
		serviceMeshClient: serviceMeshClient,
		log:               log,
		referenceResolver: referenceResolver,
	}
}

func (m *ResourceManager) GetResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	igrt, err := getIngressGatewayRouteTable(object)
	if err != nil {
		return err
	}
	if len(igrt.Status.IngressGatewayRouteTableId) > 0 {
		sdkIgrt, err := m.serviceMeshClient.GetIngressGatewayRouteTable(ctx, &igrt.Status.IngressGatewayRouteTableId)
		if err != nil {
			return err
		}
		details.IgrtDetails.SdkIgrt = sdkIgrt
		return nil
	}
	m.log.InfoLogWithFixedMessage(ctx, "ingress gateway route table did not sync to the control plane", "name", igrt.ObjectMeta.Name)
	return nil
}

func (m *ResourceManager) CreateResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) (bool, error) {
	if details.IgrtDetails.SdkIgrt == nil {
		m.log.InfoLogWithFixedMessage(ctx, "creating IngressGatewayRouteTable...", "name", object.GetName(), "OpcRetryToken", *details.OpcRetryToken)
		sdkIgrt, err := m.serviceMeshClient.CreateIngressGatewayRouteTable(ctx, details.IgrtDetails.BuildSdkIgrt, details.OpcRetryToken)
		if err != nil {
			return false, err
		}
		details.IgrtDetails.SdkIgrt = sdkIgrt
		return false, nil
	}
	return true, nil
}

func (m *ResourceManager) UpdateResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	m.log.InfoLogWithFixedMessage(ctx, "updating IngressGatewayRouteTable...", "name", object.GetName())
	details.IgrtDetails.SdkIgrt.LifecycleState = sdk.IngressGatewayRouteTableLifecycleStateUpdating
	return m.serviceMeshClient.UpdateIngressGatewayRouteTable(ctx, details.IgrtDetails.BuildSdkIgrt)
}

func (m *ResourceManager) ChangeCompartment(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	igrt, err := getIngressGatewayRouteTable(object)
	if err != nil {
		return err
	}
	m.log.InfoLogWithFixedMessage(ctx, "Moving ingress gateway route table to new compartment", "Name", object.GetName())
	// sdkIngressGatewayRouteTable state would be Active here, hence update the state to updating for correct status
	details.IgrtDetails.SdkIgrt.LifecycleState = sdk.IngressGatewayRouteTableLifecycleStateUpdating
	return m.serviceMeshClient.ChangeIngressGatewayRouteTableCompartment(ctx, &igrt.Status.IngressGatewayRouteTableId, &igrt.Spec.CompartmentId)
}

func (m *ResourceManager) DeleteResource(ctx context.Context, object client.Object) error {
	igrt, err := getIngressGatewayRouteTable(object)
	if err != nil {
		return err
	}
	if len(igrt.Status.IngressGatewayRouteTableId) == 0 {
		return nil
	}
	m.log.InfoLogWithFixedMessage(ctx, "Deleting ingress gateway route table", "Name", object.GetName())
	return m.serviceMeshClient.DeleteIngressGatewayRouteTable(ctx, &igrt.Status.IngressGatewayRouteTableId)
}

func (m *ResourceManager) VerifyEntityType(object runtime.Object) (client.Object, error) {
	igrt, ok := object.(*servicemeshapi.IngressGatewayRouteTable)
	if !ok {
		return nil, errors.New("object is not an ingress gateway route table")
	}
	return igrt, nil
}

func (m *ResourceManager) VerifyResourceStatus(details *manager.ResourceDetails) (bool, error) {
	if details.IgrtDetails.SdkIgrt == nil || details.IgrtDetails.SdkIgrt.LifecycleState == sdk.IngressGatewayRouteTableLifecycleStateActive {
		return true, nil
	}

	state := details.IgrtDetails.SdkIgrt.LifecycleState
	// Terminate the reconcile request if resource in the control plane is deleted or failed
	if state == sdk.IngressGatewayRouteTableLifecycleStateDeleted || state == sdk.IngressGatewayRouteTableLifecycleStateFailed {
		return false, meshErrors.NewDoNotRequeueError(errors.New("ingress gateway route table in the control plane is deleted or failed"))
	}

	return false, meshErrors.NewRequeueOnError(errors.New(meshCommons.UnknownStatus))
}

func (m *ResourceManager) GetServiceMeshStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	igrt, err := getIngressGatewayRouteTable(object)
	if err != nil {
		return nil, err
	}
	return &igrt.Status, nil
}

func (m *ResourceManager) GetConditionStatus(details *manager.ResourceDetails) metav1.ConditionStatus {
	state := details.IgrtDetails.SdkIgrt.LifecycleState
	return meshCommons.GetConditionStatus(string(state))
}

func (m *ResourceManager) UpdateStatus(object client.Object, details *manager.ResourceDetails) (bool, error) {
	needsUpdate := false
	igrt, err := getIngressGatewayRouteTable(object)
	if err != nil {
		return false, err
	}
	// Update IngressGatewayId and IngressGatewayName
	if igrt.Status.IngressGatewayId != api.OCID(*details.IgrtDetails.SdkIgrt.IngressGatewayId) {
		igrt.Status.IngressGatewayId = api.OCID(*details.IgrtDetails.SdkIgrt.IngressGatewayId)
		igrt.Status.IngressGatewayName = details.IgrtDetails.Dependencies.IngressGatewayName
		needsUpdate = true
	}

	// Update IngressGatewayRouteTableId
	if igrt.Status.IngressGatewayRouteTableId != api.OCID(*details.IgrtDetails.SdkIgrt.Id) {
		igrt.Status.IngressGatewayRouteTableId = api.OCID(*details.IgrtDetails.SdkIgrt.Id)
		needsUpdate = true
	}

	if !cmp.Equal(igrt.Status.VirtualServiceIdForRules, details.IgrtDetails.Dependencies.VsIdForRules) {
		igrt.Status.VirtualServiceIdForRules = details.IgrtDetails.Dependencies.VsIdForRules
		needsUpdate = true
	}

	// update operator time iff cp has different updated time
	// (we cannot change cp updated time)
	// so change operator updated time to match cp, so resource isnt reconsidered during next reconciliation
	cpTime := details.IgrtDetails.SdkIgrt.TimeUpdated
	operatorTime := igrt.Status.LastUpdatedTime
	if manager.DoTimeStampsDiffer(cpTime, operatorTime) {
		igrt.Status.LastUpdatedTime = (*metav1.Time)(details.IgrtDetails.SdkIgrt.TimeUpdated)
		needsUpdate = true
	}

	return needsUpdate, nil
}

func (m *ResourceManager) GetMessage(details *manager.ResourceDetails) string {
	state := details.IgrtDetails.SdkIgrt.LifecycleState
	return string(meshCommons.GetMessage(string(state)))
}

func (m *ResourceManager) BuildSdk(object client.Object, details *manager.ResourceDetails) error {
	buildSdkIgrt := &sdk.IngressGatewayRouteTable{}
	if details.IgrtDetails.SdkIgrt != nil {
		buildSdkIgrt.FreeformTags = details.IgrtDetails.SdkIgrt.FreeformTags
		buildSdkIgrt.DefinedTags = details.IgrtDetails.SdkIgrt.DefinedTags
	}
	igrt, err := getIngressGatewayRouteTable(object)
	if err != nil {
		return err
	}
	err = meshConversions.ConvertCrdIngressGatewayRouteTableToSdkIngressGatewayRouteTable(igrt, buildSdkIgrt, details.IgrtDetails.Dependencies)
	if err != nil {
		return err
	}
	details.IgrtDetails.BuildSdkIgrt = buildSdkIgrt
	return nil
}

func (m *ResourceManager) HasCompartmentIdChanged(object client.Object, details *manager.ResourceDetails) (bool, error) {
	igrt, err := getIngressGatewayRouteTable(object)
	if err != nil {
		return false, err
	}
	return igrt.Spec.CompartmentId != api.OCID(*details.IgrtDetails.SdkIgrt.CompartmentId), nil
}

func (m *ResourceManager) GetLifecycleState(details *manager.ResourceDetails) string {
	return string(details.IgrtDetails.SdkIgrt.LifecycleState)
}

func (m *ResourceManager) ResolveDependencies(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	igrt, err := getIngressGatewayRouteTable(object)
	if err != nil {
		return err
	}
	dependencies := &meshConversions.IGRTDependencies{}
	dependencies.IngressGatewayId = igrt.Status.IngressGatewayId
	dependencies.IngressGatewayName = igrt.Status.IngressGatewayName
	// Resolve dependencies of IngressGateway for IngressGatewayRouteTable
	if dependencies.IngressGatewayId == "" {
		ingressGatewayRef, err := m.referenceResolver.ResolveIngressGatewayIdAndNameAndMeshId(ctx, &igrt.Spec.IngressGateway, &igrt.ObjectMeta)
		if err != nil {
			return err
		}
		dependencies.IngressGatewayId = ingressGatewayRef.Id
		dependencies.IngressGatewayName = ingressGatewayRef.Name
	}
	// Resolve dependencies of VirtualService for IngressGatewayRouteTable
	dependencies.VsIdForRules = make([][]api.OCID, len(igrt.Spec.RouteRules))
	for i, routeRule := range igrt.Spec.RouteRules {
		vsIds, err := m.resolveRouteDependencies(ctx, routeRule, igrt)
		if err != nil {
			return err
		}
		dependencies.VsIdForRules[i] = vsIds
	}
	details.IgrtDetails.Dependencies = dependencies
	return nil
}

func (m *ResourceManager) resolveRouteDependencies(ctx context.Context, routeRule servicemeshapi.IngressGatewayTrafficRouteRule, igrt *servicemeshapi.IngressGatewayRouteTable) ([]api.OCID, error) {
	destinations, err := getVirtualServiceDestinations(routeRule)
	if err != nil {
		return nil, err
	}
	vsIds := make([]api.OCID, len(destinations))
	for j, rule := range destinations {
		virtualServiceRef, err := m.referenceResolver.ResolveVirtualServiceIdAndName(ctx, rule.VirtualService, &igrt.ObjectMeta)
		if err != nil {
			return nil, err
		}
		vsIds[j] = virtualServiceRef.Id
	}
	return vsIds, nil
}

func getVirtualServiceDestinations(rule servicemeshapi.IngressGatewayTrafficRouteRule) ([]servicemeshapi.VirtualServiceTrafficRuleTarget, error) {
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

func (m *ResourceManager) GetTimeUpdated(details *manager.ResourceDetails) *common.SDKTime {
	return details.IgrtDetails.SdkIgrt.TimeUpdated
}

func (m *ResourceManager) GetFinalizer() string {
	return meshCommons.IngressGatewayRouteTableFinalizer
}

func (m *ResourceManager) Finalize(ctx context.Context, object client.Object) error {
	// No resource is dependent on the IngressGatewayRouteTable resource, so nothing to do here
	return nil
}

func getIngressGatewayRouteTable(object client.Object) (*servicemeshapi.IngressGatewayRouteTable, error) {
	igrt, ok := object.(*servicemeshapi.IngressGatewayRouteTable)
	if !ok {
		return nil, errors.New("object is not an ingress gateway route table")
	}
	return igrt, nil
}

func (m *ResourceManager) HasSdk(details *manager.ResourceDetails) bool {
	return details.IgrtDetails.SdkIgrt != nil
}
