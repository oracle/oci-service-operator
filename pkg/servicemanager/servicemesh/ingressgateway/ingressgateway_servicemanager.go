/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgateway

import (
	"context"
	"errors"
	"github.com/oracle/oci-go-sdk/v65/common"

	"github.com/google/go-cmp/cmp"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/services"
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	meshConversions "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
)

type ResourceManager struct {
	client            client.Client
	serviceMeshClient services.ServiceMeshClient
	log               loggerutil.OSOKLogger
	referenceResolver references.Resolver
}

//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=ingressgateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=ingressgateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=ingressgateways/finalizers,verbs=update

func NewIngressGatewayResourceManager(client client.Client, serviceMeshClient services.ServiceMeshClient,
	log loggerutil.OSOKLogger, referenceResolver references.Resolver) manager.CustomResourceHandler {
	return &ResourceManager{
		client:            client,
		serviceMeshClient: serviceMeshClient,
		log:               log,
		referenceResolver: referenceResolver,
	}
}

func (m *ResourceManager) GetResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	ingressGateway, err := getIngressGateway(object)
	if err != nil {
		return err
	}
	if len(ingressGateway.Status.IngressGatewayId) > 0 {
		sdkIngressGateway, err := m.serviceMeshClient.GetIngressGateway(ctx, &ingressGateway.Status.IngressGatewayId)
		if err != nil {
			return err
		}
		details.IgDetails.SdkIg = sdkIngressGateway
		return nil
	}
	m.log.InfoLogWithFixedMessage(ctx, "ingressGateway did not sync to the control plane", "name", ingressGateway.ObjectMeta.Name)
	return nil
}

func (m *ResourceManager) CreateResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) (bool, error) {
	if details.IgDetails.SdkIg == nil {
		m.log.InfoLogWithFixedMessage(ctx, "creating IngressGateway...", "name", object.GetName(), "OpcRetryToken", *details.OpcRetryToken)
		sdkIngressGateway, err := m.serviceMeshClient.CreateIngressGateway(ctx, details.IgDetails.BuildSdkIg, details.OpcRetryToken)
		if err != nil {
			return false, err
		}
		details.IgDetails.SdkIg = sdkIngressGateway
		return false, nil
	}
	return true, nil
}

func (m *ResourceManager) UpdateResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	m.log.InfoLogWithFixedMessage(ctx, "updating IngressGateway...", "name", object.GetName())
	details.IgDetails.SdkIg.LifecycleState = sdk.IngressGatewayLifecycleStateUpdating
	return m.serviceMeshClient.UpdateIngressGateway(ctx, details.IgDetails.BuildSdkIg)
}

func (m *ResourceManager) ChangeCompartment(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	ingressGateway, err := getIngressGateway(object)
	if err != nil {
		return err
	}
	m.log.InfoLogWithFixedMessage(ctx, "Moving IngressGateway to new compartment", "Name", object.GetName())
	// sdkIngressGateway state would be Active here, hence update the state to updating for correct status
	details.IgDetails.SdkIg.LifecycleState = sdk.IngressGatewayLifecycleStateUpdating
	return m.serviceMeshClient.ChangeIngressGatewayCompartment(ctx, &ingressGateway.Status.IngressGatewayId, &ingressGateway.Spec.CompartmentId)
}

func (m *ResourceManager) DeleteResource(ctx context.Context, object client.Object) error {
	ingressGateway, err := getIngressGateway(object)
	if err != nil {
		return err
	}
	if len(ingressGateway.Status.IngressGatewayId) == 0 {
		return nil
	}
	m.log.InfoLogWithFixedMessage(ctx, "Deleting ingress gateway", "Name", object.GetName())
	return m.serviceMeshClient.DeleteIngressGateway(ctx, &ingressGateway.Status.IngressGatewayId)
}

func (m *ResourceManager) VerifyEntityType(object runtime.Object) (client.Object, error) {
	ingressGateway, ok := object.(*servicemeshapi.IngressGateway)
	if !ok {
		return nil, errors.New("object is not an ingress gateway")
	}
	return ingressGateway, nil
}

func (m *ResourceManager) VerifyResourceStatus(details *manager.ResourceDetails) (bool, error) {
	if details.IgDetails.SdkIg == nil || details.IgDetails.SdkIg.LifecycleState == sdk.IngressGatewayLifecycleStateActive {
		return true, nil
	}

	state := details.IgDetails.SdkIg.LifecycleState
	if state == sdk.IngressGatewayLifecycleStateDeleted || state == sdk.IngressGatewayLifecycleStateFailed {
		return false, nil
	}

	return false, meshErrors.NewRequeueOnError(errors.New(meshCommons.UnknownStatus))
}

func (m *ResourceManager) GetServiceMeshStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	IngressGateway, err := getIngressGateway(object)
	if err != nil {
		return nil, err
	}
	return &IngressGateway.Status, nil
}

func (m *ResourceManager) GetConditionStatus(details *manager.ResourceDetails) metav1.ConditionStatus {
	state := details.IgDetails.SdkIg.LifecycleState
	return meshCommons.GetConditionStatus(string(state))
}

func (m *ResourceManager) UpdateStatus(object client.Object, details *manager.ResourceDetails) (bool, error) {
	needsUpdate := false
	ingressGateway, err := getIngressGateway(object)
	if err != nil {
		return false, err
	}
	if ingressGateway.Status.MeshId == "" {
		ingressGateway.Status.MeshId = api.OCID(*details.IgDetails.SdkIg.MeshId)
		needsUpdate = true
	}
	if ingressGateway.Status.IngressGatewayId != api.OCID(*details.IgDetails.SdkIg.Id) {
		ingressGateway.Status.IngressGatewayId = api.OCID(*details.IgDetails.SdkIg.Id)
		needsUpdate = true
	}

	mtls := meshConversions.ConvertSdkIgMtlsToCrdIgMtls(details.IgDetails.SdkIg.Mtls)
	if mtls != nil && !cmp.Equal(ingressGateway.Status.IngressGatewayMtls, mtls) {
		ingressGateway.Status.IngressGatewayMtls = mtls
		needsUpdate = true
	}

	// update operator time iff cp has different updated time
	// (we cannot change cp updated time)
	// so change operator updated time to match cp, so resource isnt reconsidered during next reconciliation
	cpTime := details.IgDetails.SdkIg.TimeUpdated
	operatorTime := ingressGateway.Status.LastUpdatedTime
	if manager.DoTimeStampsDiffer(cpTime, operatorTime) {
		ingressGateway.Status.LastUpdatedTime = (*metav1.Time)(details.IgDetails.SdkIg.TimeUpdated)
		needsUpdate = true
	}

	return needsUpdate, nil
}

func (m *ResourceManager) GetMessage(details *manager.ResourceDetails) string {
	state := details.IgDetails.SdkIg.LifecycleState
	return string(meshCommons.GetMessage(string(state)))
}

func (m *ResourceManager) BuildSdk(object client.Object, details *manager.ResourceDetails) error {
	buildSdkIngressGateway := &sdk.IngressGateway{}
	ingressGateway, err := getIngressGateway(object)
	if err != nil {
		return err
	}
	meshConversions.ConvertCrdIngressGatewayToSdkIngressGateway(ingressGateway, buildSdkIngressGateway, details.IgDetails.MeshId)
	details.IgDetails.BuildSdkIg = buildSdkIngressGateway
	return nil
}

func (m *ResourceManager) HasCompartmentIdChanged(object client.Object, details *manager.ResourceDetails) (bool, error) {
	ingressGateway, err := getIngressGateway(object)
	if err != nil {
		return false, err
	}
	return ingressGateway.Spec.CompartmentId != api.OCID(*details.IgDetails.SdkIg.CompartmentId), nil
}

func (m *ResourceManager) GetLifecycleState(details *manager.ResourceDetails) string {
	return string(details.IgDetails.SdkIg.LifecycleState)
}

func (m *ResourceManager) ResolveDependencies(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	ingressGateway, err := getIngressGateway(object)
	if err != nil {
		return err
	}
	meshId := ingressGateway.Status.MeshId
	if meshId == "" {
		meshOcid, err := m.referenceResolver.ResolveMeshId(ctx, &ingressGateway.Spec.Mesh, &ingressGateway.ObjectMeta)
		details.IgDetails.MeshId = meshOcid
		return err
	}
	return nil
}

func (m *ResourceManager) GetTimeUpdated(details *manager.ResourceDetails) *common.SDKTime {
	return details.IgDetails.SdkIg.TimeUpdated
}

func (m *ResourceManager) GetFinalizer() string {
	return meshCommons.IngressGatewayFinalizer
}

func (m *ResourceManager) Finalize(ctx context.Context, object client.Object) error {
	ingressGateway, err := getIngressGateway(object)
	if err != nil {
		return err
	}

	if hasDependencies, err := m.hasIngressGatewayDeployments(ctx, ingressGateway); err != nil {
		return err
	} else if hasDependencies {
		return errors.New("cannot delete ingress gateway when there are ingress gateway deployment resources associated")
	}

	if hasDependencies, err := m.hasIngressGatewayRouteTables(ctx, ingressGateway); err != nil {
		return err
	} else if hasDependencies {
		return errors.New("cannot delete ingress gateway when there are ingress gateway route table resources associated")
	}

	if hasDependencies, err := m.hasAccessPolicies(ctx, ingressGateway); err != nil {
		return err
	} else if hasDependencies {
		return errors.New("cannot delete ingress gateway when there are access policy resources associated")
	}
	return nil
}

func (m *ResourceManager) hasIngressGatewayDeployments(ctx context.Context, ingressGateway *servicemeshapi.IngressGateway) (bool, error) {
	igdList := &servicemeshapi.IngressGatewayDeploymentList{}

	if err := m.client.List(ctx, igdList); err != nil {
		m.log.InfoLogWithFixedMessage(ctx, "unable to list ingressGatewayDeployments associated with ingressGateway", "name", ingressGateway.Name, "namespace", ingressGateway.Namespace)
		return false, err
	}

	for _, igd := range igdList.Items {
		if igd.Status.IngressGatewayId == ingressGateway.Status.IngressGatewayId {
			return true, nil
		}
	}

	return false, nil
}

func (m *ResourceManager) hasIngressGatewayRouteTables(ctx context.Context, ig *servicemeshapi.IngressGateway) (bool, error) {
	igrtList := &servicemeshapi.IngressGatewayRouteTableList{}

	if err := m.client.List(ctx, igrtList); err != nil {
		m.log.InfoLogWithFixedMessage(ctx, "unable to list ingressGatewayRouteTables for ingressGateway", "name", ig.Name, "namespace", ig.Namespace)
		return false, err
	}

	hasDependencies := false
	for _, igrt := range igrtList.Items {
		if igrt.Status.IngressGatewayId == ig.Status.IngressGatewayId {
			hasDependencies = true
			break
		}
	}
	return hasDependencies, nil
}

func (m *ResourceManager) hasAccessPolicies(ctx context.Context, ingressGateway *servicemeshapi.IngressGateway) (bool, error) {
	apList := &servicemeshapi.AccessPolicyList{}

	if err := m.client.List(ctx, apList); err != nil {
		m.log.InfoLogWithFixedMessage(ctx, "unable to list accessPolicies for ingressGateway", "name", ingressGateway.Name, "namespace", ingressGateway.Namespace)
		return false, err
	}

	for _, ap := range apList.Items {
		if ingressGateway.Status.MeshId == ap.Status.MeshId {
			for _, refIdsForRule := range ap.Status.RefIdForRules {
				if ingressGateway.Status.IngressGatewayId == refIdsForRule[meshCommons.Source] {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func getIngressGateway(object client.Object) (*servicemeshapi.IngressGateway, error) {
	ingressGateway, ok := object.(*servicemeshapi.IngressGateway)
	if !ok {
		return nil, errors.New("object is not an ingress gateway")
	}
	return ingressGateway, nil
}

func (m *ResourceManager) HasSdk(details *manager.ResourceDetails) bool {
	return details.IgDetails.SdkIg != nil
}
