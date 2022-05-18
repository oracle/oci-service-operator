/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package accesspolicy

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

//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=accesspolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=accesspolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=servicemesh.oci.oracle.com,resources=accesspolicies/finalizers,verbs=update

func NewAccessPolicyResourceManager(client client.Client, serviceMeshClient services.ServiceMeshClient,
	log loggerutil.OSOKLogger, referenceResolver references.Resolver) manager.CustomResourceHandler {
	return &ResourceManager{
		client:            client,
		serviceMeshClient: serviceMeshClient,
		log:               log,
		referenceResolver: referenceResolver,
	}
}

func (m *ResourceManager) GetResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	accessPolicy, err := getAccessPolicy(object)
	if err != nil {
		return err
	}
	if len(accessPolicy.Status.AccessPolicyId) > 0 {
		sdkAccessPolicy, err := m.serviceMeshClient.GetAccessPolicy(ctx, &accessPolicy.Status.AccessPolicyId)
		if err != nil {
			return err
		}
		details.ApDetails.SdkAp = sdkAccessPolicy
		return nil
	}
	m.log.InfoLogWithFixedMessage(ctx, "accessPolicy did not sync to the control plane", "name", accessPolicy.ObjectMeta.Name)
	return nil
}

func (m *ResourceManager) CreateResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) (bool, error) {
	if details.ApDetails.SdkAp == nil {
		m.log.InfoLogWithFixedMessage(ctx, "creating accessPolicy...", "name", object.GetName(), "OpcRetryToken", *details.OpcRetryToken)
		sdkAccessPolicy, err := m.serviceMeshClient.CreateAccessPolicy(ctx, details.ApDetails.BuildSdkAp, details.OpcRetryToken)
		if err != nil {
			return false, err
		}
		details.ApDetails.SdkAp = sdkAccessPolicy
		return false, nil
	}
	return true, nil
}

func (m *ResourceManager) UpdateResource(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	m.log.InfoLogWithFixedMessage(ctx, "updating accessPolicy...", "name", object.GetName())
	details.ApDetails.SdkAp.LifecycleState = sdk.AccessPolicyLifecycleStateUpdating
	return m.serviceMeshClient.UpdateAccessPolicy(ctx, details.ApDetails.BuildSdkAp)
}

func (m *ResourceManager) ChangeCompartment(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	accessPolicy, err := getAccessPolicy(object)
	if err != nil {
		return err
	}
	m.log.InfoLogWithFixedMessage(ctx, "Moving accessPolicy to new compartment", "Name", object.GetName())
	// sdkAccessPolicy state would be Active here, hence update the state to updating for correct status
	details.ApDetails.SdkAp.LifecycleState = sdk.AccessPolicyLifecycleStateUpdating
	return m.serviceMeshClient.ChangeAccessPolicyCompartment(ctx, &accessPolicy.Status.AccessPolicyId, &accessPolicy.Spec.CompartmentId)
}

func (m *ResourceManager) DeleteResource(ctx context.Context, object client.Object) error {
	accessPolicy, err := getAccessPolicy(object)
	if err != nil {
		return err
	}
	if len(accessPolicy.Status.AccessPolicyId) == 0 {
		return nil
	}
	m.log.InfoLogWithFixedMessage(ctx, "Deleting access policy", "Name", object.GetName())
	return m.serviceMeshClient.DeleteAccessPolicy(ctx, &accessPolicy.Status.AccessPolicyId)
}

func (m *ResourceManager) VerifyEntityType(object runtime.Object) (client.Object, error) {
	accessPolicy, ok := object.(*servicemeshapi.AccessPolicy)
	if !ok {
		return nil, errors.New("object is not access policy")
	}
	return accessPolicy, nil
}

func (m *ResourceManager) VerifyResourceStatus(details *manager.ResourceDetails) (bool, error) {
	if details.ApDetails.SdkAp == nil || details.ApDetails.SdkAp.LifecycleState == sdk.AccessPolicyLifecycleStateActive {
		return true, nil
	}

	state := details.ApDetails.SdkAp.LifecycleState
	if state == sdk.AccessPolicyLifecycleStateDeleted || state == sdk.AccessPolicyLifecycleStateFailed {
		return false, nil
	}

	return false, meshErrors.NewRequeueOnError(errors.New(meshCommons.UnknownStatus))
}

func (m *ResourceManager) GetServiceMeshStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	accessPolicy, err := getAccessPolicy(object)
	if err != nil {
		return nil, err
	}
	return &accessPolicy.Status, nil
}

func (m *ResourceManager) GetConditionStatus(details *manager.ResourceDetails) metav1.ConditionStatus {
	state := details.ApDetails.SdkAp.LifecycleState
	return meshCommons.GetConditionStatus(string(state))
}

func (m *ResourceManager) UpdateStatus(object client.Object, details *manager.ResourceDetails) (bool, error) {
	needsUpdate := false
	accessPolicy, err := getAccessPolicy(object)
	if err != nil {
		return false, err
	}
	if accessPolicy.Status.MeshId == "" {
		accessPolicy.Status.MeshId = api.OCID(*details.ApDetails.SdkAp.MeshId)
		needsUpdate = true
	}

	if accessPolicy.Status.AccessPolicyId != api.OCID(*details.ApDetails.SdkAp.Id) {
		accessPolicy.Status.AccessPolicyId = api.OCID(*details.ApDetails.SdkAp.Id)
		needsUpdate = true
	}

	if !cmp.Equal(accessPolicy.Status.RefIdForRules, details.ApDetails.Dependencies.RefIdForRules) {
		accessPolicy.Status.RefIdForRules = details.ApDetails.Dependencies.RefIdForRules
		needsUpdate = true
	}

	// update operator time iff cp has different updated time
	// (we cannot change cp updated time)
	// so change operator updated time to match cp, so resource isnt reconsidered during next reconciliation
	cpTime := details.ApDetails.SdkAp.TimeUpdated
	operatorTime := accessPolicy.Status.LastUpdatedTime
	if manager.DoTimeStampsDiffer(cpTime, operatorTime) {
		accessPolicy.Status.LastUpdatedTime = (*metav1.Time)(details.ApDetails.SdkAp.TimeUpdated)
		needsUpdate = true
	}

	return needsUpdate, nil
}

func (m *ResourceManager) GetMessage(details *manager.ResourceDetails) string {
	state := details.ApDetails.SdkAp.LifecycleState
	return string(meshCommons.GetMessage(string(state)))
}

func (m *ResourceManager) BuildSdk(object client.Object, details *manager.ResourceDetails) error {
	buildSdkAccessPolicy := &sdk.AccessPolicy{}
	accessPolicy, err := getAccessPolicy(object)
	if err != nil {
		return err
	}
	meshConversions.ConvertCrdAccessPolicyToSdkAccessPolicy(accessPolicy, buildSdkAccessPolicy, details.ApDetails.Dependencies)
	details.ApDetails.BuildSdkAp = buildSdkAccessPolicy
	return nil
}

func (m *ResourceManager) HasCompartmentIdChanged(object client.Object, details *manager.ResourceDetails) (bool, error) {
	accessPolicy, err := getAccessPolicy(object)
	if err != nil {
		return false, err
	}
	return accessPolicy.Spec.CompartmentId != api.OCID(*details.ApDetails.SdkAp.CompartmentId), nil
}

func (m *ResourceManager) GetLifecycleState(details *manager.ResourceDetails) string {
	return string(details.ApDetails.SdkAp.LifecycleState)
}

func (m *ResourceManager) ResolveDependencies(ctx context.Context, object client.Object, details *manager.ResourceDetails) error {
	accessPolicy, err := getAccessPolicy(object)
	if err != nil {
		return err
	}
	dependencies := &meshConversions.AccessPolicyDependencies{}
	dependencies.MeshId = accessPolicy.Status.MeshId
	// Resolve dependencies of Mesh for AccessPolicy
	if dependencies.MeshId == "" {
		meshID, err := m.referenceResolver.ResolveMeshId(ctx, &accessPolicy.Spec.Mesh, &accessPolicy.ObjectMeta)
		if err != nil {
			return err
		}
		dependencies.MeshId = *meshID
	}
	// Resolve dependencies of VirtualService or IngressGateway for AccessPolicy
	dependencies.RefIdForRules = make([]map[string]api.OCID, len(accessPolicy.Spec.Rules))
	for i, rule := range accessPolicy.Spec.Rules {
		refIds := make(map[string]api.OCID)
		if rule.Source.VirtualService != nil {
			virtualServiceRef, err := m.referenceResolver.ResolveVirtualServiceIdAndName(ctx, rule.Source.VirtualService, &accessPolicy.ObjectMeta)
			if err != nil {
				return err
			}
			refIds[meshCommons.Source] = virtualServiceRef.Id
		} else if rule.Source.IngressGateway != nil {
			ingressGatewayRef, err := m.referenceResolver.ResolveIngressGatewayIdAndNameAndMeshId(ctx, rule.Source.IngressGateway, &accessPolicy.ObjectMeta)
			if err != nil {
				return err
			}
			refIds[meshCommons.Source] = ingressGatewayRef.Id
		}
		if rule.Destination.VirtualService != nil {
			virtualServiceRef, err := m.referenceResolver.ResolveVirtualServiceIdAndName(ctx, rule.Destination.VirtualService, &accessPolicy.ObjectMeta)
			if err != nil {
				return err
			}
			refIds[meshCommons.Destination] = virtualServiceRef.Id
		}
		dependencies.RefIdForRules[i] = refIds
	}
	details.ApDetails.Dependencies = dependencies
	return nil
}

func (m *ResourceManager) GetTimeUpdated(details *manager.ResourceDetails) *common.SDKTime {
	return details.ApDetails.SdkAp.TimeUpdated
}

func (m *ResourceManager) GetFinalizer() string {
	return meshCommons.AccessPolicyFinalizer
}

func (m *ResourceManager) Finalize(ctx context.Context, object client.Object) error {
	// No resource is dependent on the AccessPolicy resource, so nothing to do here
	return nil
}

func getAccessPolicy(object client.Object) (*servicemeshapi.AccessPolicy, error) {
	accessPolicy, ok := object.(*servicemeshapi.AccessPolicy)
	if !ok {
		return nil, errors.New("object is not a access policy")
	}
	return accessPolicy, nil
}

func (m *ResourceManager) HasSdk(details *manager.ResourceDetails) bool {
	return details.ApDetails.SdkAp != nil
}
