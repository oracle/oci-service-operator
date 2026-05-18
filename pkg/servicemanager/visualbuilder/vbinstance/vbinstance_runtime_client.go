/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vbinstance

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	visualbuildersdk "github.com/oracle/oci-go-sdk/v65/visualbuilder"
	visualbuilderv1beta1 "github.com/oracle/oci-service-operator/api/visualbuilder/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

const vbInstanceKind = "VbInstance"

var vbInstanceWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(visualbuildersdk.WorkRequestStatusAccepted),
		string(visualbuildersdk.WorkRequestStatusInProgress),
		string(visualbuildersdk.WorkRequestStatusCanceling),
	},
	SucceededStatusTokens: []string{string(visualbuildersdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(visualbuildersdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(visualbuildersdk.WorkRequestStatusCanceled)},
	CreateActionTokens:    []string{string(visualbuildersdk.WorkRequestOperationTypeCreateVbInstance)},
	UpdateActionTokens:    []string{string(visualbuildersdk.WorkRequestOperationTypeUpdateVbInstance)},
	DeleteActionTokens:    []string{string(visualbuildersdk.WorkRequestOperationTypeDeleteVbInstance)},
}

type vbInstanceOCIClient interface {
	CreateVbInstance(context.Context, visualbuildersdk.CreateVbInstanceRequest) (visualbuildersdk.CreateVbInstanceResponse, error)
	GetVbInstance(context.Context, visualbuildersdk.GetVbInstanceRequest) (visualbuildersdk.GetVbInstanceResponse, error)
	ListVbInstances(context.Context, visualbuildersdk.ListVbInstancesRequest) (visualbuildersdk.ListVbInstancesResponse, error)
	UpdateVbInstance(context.Context, visualbuildersdk.UpdateVbInstanceRequest) (visualbuildersdk.UpdateVbInstanceResponse, error)
	DeleteVbInstance(context.Context, visualbuildersdk.DeleteVbInstanceRequest) (visualbuildersdk.DeleteVbInstanceResponse, error)
	GetWorkRequest(context.Context, visualbuildersdk.GetWorkRequestRequest) (visualbuildersdk.GetWorkRequestResponse, error)
}

func init() {
	registerVbInstanceRuntimeHooksMutator(func(manager *VbInstanceServiceManager, hooks *VbInstanceRuntimeHooks) {
		client, initErr := newVbInstanceOCIClient(manager)
		applyVbInstanceRuntimeHooks(hooks, client, initErr)
	})
}

func newVbInstanceOCIClient(manager *VbInstanceServiceManager) (vbInstanceOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", vbInstanceKind)
	}
	client, err := visualbuildersdk.NewVbInstanceClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyVbInstanceRuntimeHooks(
	hooks *VbInstanceRuntimeHooks,
	client vbInstanceOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedVbInstanceRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *visualbuilderv1beta1.VbInstance, _ string) (any, error) {
		return buildVbInstanceCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *visualbuilderv1beta1.VbInstance,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildVbInstanceUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardVbInstanceExistingBeforeCreate
	hooks.Async.Adapter = vbInstanceWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getVbInstanceWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveVbInstanceWorkRequestAction
	hooks.Async.ResolvePhase = resolveVbInstanceWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverVbInstanceIDFromWorkRequest
	hooks.Async.Message = vbInstanceWorkRequestMessage
}

func newVbInstanceServiceClientWithOCIClient(client vbInstanceOCIClient) VbInstanceServiceClient {
	hooks := newVbInstanceRuntimeHooksWithOCIClient(client)
	applyVbInstanceRuntimeHooks(&hooks, client, nil)
	delegate := defaultVbInstanceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*visualbuilderv1beta1.VbInstance](
			buildVbInstanceGeneratedRuntimeConfig(&VbInstanceServiceManager{}, hooks),
		),
	}
	return wrapVbInstanceGeneratedClient(hooks, delegate)
}

func newVbInstanceRuntimeHooksWithOCIClient(client vbInstanceOCIClient) VbInstanceRuntimeHooks {
	hooks := newVbInstanceDefaultRuntimeHooks(visualbuildersdk.VbInstanceClient{})
	hooks.Create.Call = func(ctx context.Context, request visualbuildersdk.CreateVbInstanceRequest) (visualbuildersdk.CreateVbInstanceResponse, error) {
		return client.CreateVbInstance(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request visualbuildersdk.GetVbInstanceRequest) (visualbuildersdk.GetVbInstanceResponse, error) {
		return client.GetVbInstance(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request visualbuildersdk.ListVbInstancesRequest) (visualbuildersdk.ListVbInstancesResponse, error) {
		return client.ListVbInstances(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request visualbuildersdk.UpdateVbInstanceRequest) (visualbuildersdk.UpdateVbInstanceResponse, error) {
		return client.UpdateVbInstance(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request visualbuildersdk.DeleteVbInstanceRequest) (visualbuildersdk.DeleteVbInstanceResponse, error) {
		return client.DeleteVbInstance(ctx, request)
	}
	return hooks
}

func reviewedVbInstanceRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "visualbuilder",
		FormalSlug:    "vbinstance",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(visualbuildersdk.VbInstanceLifecycleStateCreating)},
			UpdatingStates:     []string{string(visualbuildersdk.VbInstanceLifecycleStateUpdating)},
			ActiveStates: []string{
				string(visualbuildersdk.VbInstanceLifecycleStateActive),
				string(visualbuildersdk.VbInstanceLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(visualbuildersdk.VbInstanceLifecycleStateDeleting)},
			TerminalStates: []string{string(visualbuildersdk.VbInstanceLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"alternateCustomEndpoints.certificateSecretId",
				"alternateCustomEndpoints.hostname",
				"customEndpoint.certificateSecretId",
				"customEndpoint.hostname",
				"definedTags",
				"displayName",
				"freeformTags",
				"isVisualBuilderEnabled",
				"networkEndpointDetails.allowlistedHttpIps",
				"networkEndpointDetails.allowlistedHttpVcns.allowlistedIpCidrs",
				"networkEndpointDetails.allowlistedHttpVcns.id",
				"networkEndpointDetails.networkEndpointType",
				"networkEndpointDetails.networkSecurityGroupIds",
				"networkEndpointDetails.subnetId",
				"nodeCount",
			},
			ForceNew: []string{
				"compartmentId",
				"consumptionModel",
				"networkEndpointDetails.privateEndpointIp",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: vbInstanceKind, Action: "CreateVbInstance"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: vbInstanceKind, Action: "UpdateVbInstance"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: vbInstanceKind, Action: "DeleteVbInstance"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetVbInstance",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource", EntityType: vbInstanceKind, Action: "GetWorkRequest"},
				{Helper: "tfresource.CreateResource", EntityType: vbInstanceKind, Action: "GetVbInstance"},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetVbInstance",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource", EntityType: vbInstanceKind, Action: "GetWorkRequest"},
				{Helper: "tfresource.UpdateResource", EntityType: vbInstanceKind, Action: "GetVbInstance"},
			},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetVbInstance/ListVbInstances confirm-delete",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource", EntityType: vbInstanceKind, Action: "GetWorkRequest"},
				{Helper: "tfresource.DeleteResource", EntityType: vbInstanceKind, Action: "GetVbInstance"},
			},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
	}
}

func guardVbInstanceExistingBeforeCreate(
	_ context.Context,
	resource *visualbuilderv1beta1.VbInstance,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("%s resource is nil", vbInstanceKind)
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildVbInstanceCreateBody(resource *visualbuilderv1beta1.VbInstance) (visualbuildersdk.CreateVbInstanceDetails, error) {
	if resource == nil {
		return visualbuildersdk.CreateVbInstanceDetails{}, fmt.Errorf("%s resource is nil", vbInstanceKind)
	}

	details := visualbuildersdk.CreateVbInstanceDetails{
		DisplayName:   common.String(resource.Spec.DisplayName),
		CompartmentId: common.String(resource.Spec.CompartmentId),
		NodeCount:     common.Int(resource.Spec.NodeCount),
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = vbInstanceDefinedTagsFromSpec(resource.Spec.DefinedTags)
	}
	if idcsOpenID := strings.TrimSpace(resource.Spec.IdcsOpenId); idcsOpenID != "" {
		details.IdcsOpenId = common.String(idcsOpenID)
	}
	if resource.Spec.IsVisualBuilderEnabled {
		details.IsVisualBuilderEnabled = common.Bool(true)
	}
	if endpoint, ok, err := buildVbInstanceCreateCustomEndpoint(resource.Spec.CustomEndpoint); err != nil {
		return visualbuildersdk.CreateVbInstanceDetails{}, err
	} else if ok {
		details.CustomEndpoint = endpoint
	}
	if resource.Spec.AlternateCustomEndpoints != nil {
		alternate, err := buildVbInstanceCreateAlternateCustomEndpoints(resource.Spec.AlternateCustomEndpoints)
		if err != nil {
			return visualbuildersdk.CreateVbInstanceDetails{}, err
		}
		details.AlternateCustomEndpoints = alternate
	}
	if consumptionModel := strings.TrimSpace(resource.Spec.ConsumptionModel); consumptionModel != "" {
		details.ConsumptionModel = visualbuildersdk.CreateVbInstanceDetailsConsumptionModelEnum(consumptionModel)
	}
	if endpointDetails, ok, err := buildVbInstanceCreateNetworkEndpointDetails(resource.Spec.NetworkEndpointDetails); err != nil {
		return visualbuildersdk.CreateVbInstanceDetails{}, err
	} else if ok {
		details.NetworkEndpointDetails = endpointDetails
	}

	return details, nil
}

func buildVbInstanceUpdateBody(
	resource *visualbuilderv1beta1.VbInstance,
	currentResponse any,
) (visualbuildersdk.UpdateVbInstanceDetails, bool, error) {
	if resource == nil {
		return visualbuildersdk.UpdateVbInstanceDetails{}, false, fmt.Errorf("%s resource is nil", vbInstanceKind)
	}

	desired, err := desiredVbInstanceUpdateDetails(resource)
	if err != nil {
		return visualbuildersdk.UpdateVbInstanceDetails{}, false, err
	}
	desiredValues, err := vbInstancePrunedJSONMap(desired)
	if err != nil {
		return visualbuildersdk.UpdateVbInstanceDetails{}, false, fmt.Errorf("project desired %s update body: %w", vbInstanceKind, err)
	}
	if len(desiredValues) == 0 {
		return desired, false, nil
	}

	current, err := currentVbInstanceUpdateDetails(currentResponse)
	if err != nil {
		return visualbuildersdk.UpdateVbInstanceDetails{}, false, err
	}
	currentValues, err := vbInstancePrunedJSONMap(current)
	if err != nil {
		return visualbuildersdk.UpdateVbInstanceDetails{}, false, fmt.Errorf("project current %s update body: %w", vbInstanceKind, err)
	}

	return desired, !vbInstanceMapSubsetEqual(desiredValues, currentValues), nil
}

func desiredVbInstanceUpdateDetails(
	resource *visualbuilderv1beta1.VbInstance,
) (visualbuildersdk.UpdateVbInstanceDetails, error) {
	details := visualbuildersdk.UpdateVbInstanceDetails{}

	if displayName := strings.TrimSpace(resource.Spec.DisplayName); displayName != "" {
		details.DisplayName = common.String(displayName)
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = vbInstanceDefinedTagsFromSpec(resource.Spec.DefinedTags)
	}
	if resource.Spec.NodeCount != 0 {
		details.NodeCount = common.Int(resource.Spec.NodeCount)
	}
	if resource.Spec.IsVisualBuilderEnabled {
		details.IsVisualBuilderEnabled = common.Bool(true)
	}
	if endpoint, ok, err := buildVbInstanceUpdateCustomEndpoint(resource.Spec.CustomEndpoint); err != nil {
		return visualbuildersdk.UpdateVbInstanceDetails{}, err
	} else if ok {
		details.CustomEndpoint = endpoint
	}
	if resource.Spec.AlternateCustomEndpoints != nil {
		alternate, err := buildVbInstanceUpdateAlternateCustomEndpoints(resource.Spec.AlternateCustomEndpoints)
		if err != nil {
			return visualbuildersdk.UpdateVbInstanceDetails{}, err
		}
		details.AlternateCustomEndpoints = alternate
	}
	if endpointDetails, ok, err := buildVbInstanceUpdateNetworkEndpointDetails(resource.Spec.NetworkEndpointDetails); err != nil {
		return visualbuildersdk.UpdateVbInstanceDetails{}, err
	} else if ok {
		details.NetworkEndpointDetails = endpointDetails
	}

	return details, nil
}

func currentVbInstanceUpdateDetails(currentResponse any) (visualbuildersdk.UpdateVbInstanceDetails, error) {
	if currentResponse == nil {
		return visualbuildersdk.UpdateVbInstanceDetails{}, nil
	}

	body, err := vbInstanceRuntimeBody(currentResponse)
	if err != nil {
		return visualbuildersdk.UpdateVbInstanceDetails{}, err
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return visualbuildersdk.UpdateVbInstanceDetails{}, fmt.Errorf("marshal current %s response: %w", vbInstanceKind, err)
	}

	var details visualbuildersdk.UpdateVbInstanceDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return visualbuildersdk.UpdateVbInstanceDetails{}, fmt.Errorf("decode current %s update body: %w", vbInstanceKind, err)
	}
	return details, nil
}

func buildVbInstanceCreateCustomEndpoint(
	spec visualbuilderv1beta1.VbInstanceCustomEndpoint,
) (*visualbuildersdk.CreateCustomEndpointDetails, bool, error) {
	hostname := strings.TrimSpace(spec.Hostname)
	if hostname == "" {
		return nil, false, nil
	}

	details := &visualbuildersdk.CreateCustomEndpointDetails{
		Hostname: common.String(hostname),
	}
	if certificateSecretID := strings.TrimSpace(spec.CertificateSecretId); certificateSecretID != "" {
		details.CertificateSecretId = common.String(certificateSecretID)
	}
	return details, true, nil
}

func buildVbInstanceCreateAlternateCustomEndpoints(
	spec []visualbuilderv1beta1.VbInstanceAlternateCustomEndpoint,
) ([]visualbuildersdk.CreateCustomEndpointDetails, error) {
	if spec == nil {
		return nil, nil
	}
	if len(spec) == 0 {
		return []visualbuildersdk.CreateCustomEndpointDetails{}, nil
	}

	details := make([]visualbuildersdk.CreateCustomEndpointDetails, 0, len(spec))
	for _, item := range spec {
		hostname := strings.TrimSpace(item.Hostname)
		if hostname == "" {
			return nil, fmt.Errorf("%s alternateCustomEndpoints entries require hostname", vbInstanceKind)
		}
		endpoint := visualbuildersdk.CreateCustomEndpointDetails{
			Hostname: common.String(hostname),
		}
		if certificateSecretID := strings.TrimSpace(item.CertificateSecretId); certificateSecretID != "" {
			endpoint.CertificateSecretId = common.String(certificateSecretID)
		}
		details = append(details, endpoint)
	}
	return details, nil
}

func buildVbInstanceCreateNetworkEndpointDetails(
	spec visualbuilderv1beta1.VbInstanceNetworkEndpointDetails,
) (visualbuildersdk.NetworkEndpointDetails, bool, error) {
	switch strings.ToUpper(strings.TrimSpace(spec.NetworkEndpointType)) {
	case "":
		return nil, false, nil
	case "PUBLIC":
		details := visualbuildersdk.PublicEndpointDetails{}
		if spec.AllowlistedHttpIps != nil {
			details.AllowlistedHttpIps = cloneVbInstanceStringSlice(spec.AllowlistedHttpIps)
		}
		if spec.AllowlistedHttpVcns != nil {
			details.AllowlistedHttpVcns = buildVbInstanceVCNs(spec.AllowlistedHttpVcns)
		}
		return details, true, nil
	case "PRIVATE":
		subnetID := strings.TrimSpace(spec.SubnetId)
		if subnetID == "" {
			return nil, false, fmt.Errorf("%s private networkEndpointDetails require subnetId", vbInstanceKind)
		}
		details := visualbuildersdk.PrivateEndpointDetails{
			SubnetId: common.String(subnetID),
		}
		if spec.NetworkSecurityGroupIds != nil {
			details.NetworkSecurityGroupIds = cloneVbInstanceStringSlice(spec.NetworkSecurityGroupIds)
		}
		if privateEndpointIP := strings.TrimSpace(spec.PrivateEndpointIp); privateEndpointIP != "" {
			details.PrivateEndpointIp = common.String(privateEndpointIP)
		}
		return details, true, nil
	default:
		return nil, false, fmt.Errorf("%s networkEndpointDetails.networkEndpointType %q is unsupported", vbInstanceKind, spec.NetworkEndpointType)
	}
}

func buildVbInstanceUpdateCustomEndpoint(
	spec visualbuilderv1beta1.VbInstanceCustomEndpoint,
) (*visualbuildersdk.UpdateCustomEndpointDetails, bool, error) {
	hostname := strings.TrimSpace(spec.Hostname)
	if hostname == "" {
		return nil, false, nil
	}

	details := &visualbuildersdk.UpdateCustomEndpointDetails{
		Hostname: common.String(hostname),
	}
	if certificateSecretID := strings.TrimSpace(spec.CertificateSecretId); certificateSecretID != "" {
		details.CertificateSecretId = common.String(certificateSecretID)
	}
	return details, true, nil
}

func buildVbInstanceUpdateAlternateCustomEndpoints(
	spec []visualbuilderv1beta1.VbInstanceAlternateCustomEndpoint,
) ([]visualbuildersdk.UpdateCustomEndpointDetails, error) {
	if spec == nil {
		return nil, nil
	}
	if len(spec) == 0 {
		return []visualbuildersdk.UpdateCustomEndpointDetails{}, nil
	}

	details := make([]visualbuildersdk.UpdateCustomEndpointDetails, 0, len(spec))
	for _, item := range spec {
		hostname := strings.TrimSpace(item.Hostname)
		if hostname == "" {
			return nil, fmt.Errorf("%s alternateCustomEndpoints entries require hostname", vbInstanceKind)
		}
		endpoint := visualbuildersdk.UpdateCustomEndpointDetails{
			Hostname: common.String(hostname),
		}
		if certificateSecretID := strings.TrimSpace(item.CertificateSecretId); certificateSecretID != "" {
			endpoint.CertificateSecretId = common.String(certificateSecretID)
		}
		details = append(details, endpoint)
	}
	return details, nil
}

func buildVbInstanceUpdateNetworkEndpointDetails(
	spec visualbuilderv1beta1.VbInstanceNetworkEndpointDetails,
) (visualbuildersdk.UpdateNetworkEndpointDetails, bool, error) {
	switch strings.ToUpper(strings.TrimSpace(spec.NetworkEndpointType)) {
	case "":
		return nil, false, nil
	case "PUBLIC":
		details := visualbuildersdk.UpdatePublicEndpointDetails{}
		if spec.AllowlistedHttpIps != nil {
			details.AllowlistedHttpIps = cloneVbInstanceStringSlice(spec.AllowlistedHttpIps)
		}
		if spec.AllowlistedHttpVcns != nil {
			details.AllowlistedHttpVcns = buildVbInstanceVCNs(spec.AllowlistedHttpVcns)
		}
		return details, true, nil
	case "PRIVATE":
		details := visualbuildersdk.UpdatePrivateEndpointDetails{}
		if subnetID := strings.TrimSpace(spec.SubnetId); subnetID != "" {
			details.SubnetId = common.String(subnetID)
		}
		if spec.NetworkSecurityGroupIds != nil {
			details.NetworkSecurityGroupIds = cloneVbInstanceStringSlice(spec.NetworkSecurityGroupIds)
		}
		return details, true, nil
	default:
		return nil, false, fmt.Errorf("%s networkEndpointDetails.networkEndpointType %q is unsupported", vbInstanceKind, spec.NetworkEndpointType)
	}
}

func buildVbInstanceVCNs(
	spec []visualbuilderv1beta1.VbInstanceNetworkEndpointDetailsAllowlistedHttpVcn,
) []visualbuildersdk.VirtualCloudNetwork {
	if spec == nil {
		return nil
	}
	if len(spec) == 0 {
		return []visualbuildersdk.VirtualCloudNetwork{}
	}

	vcns := make([]visualbuildersdk.VirtualCloudNetwork, 0, len(spec))
	for _, item := range spec {
		vcn := visualbuildersdk.VirtualCloudNetwork{
			Id: common.String(item.Id),
		}
		if item.AllowlistedIpCidrs != nil {
			vcn.AllowlistedIpCidrs = cloneVbInstanceStringSlice(item.AllowlistedIpCidrs)
		}
		vcns = append(vcns, vcn)
	}
	return vcns
}

func vbInstanceRuntimeBody(currentResponse any) (visualbuildersdk.VbInstance, error) {
	switch current := currentResponse.(type) {
	case nil:
		return visualbuildersdk.VbInstance{}, nil
	case visualbuildersdk.VbInstance:
		return current, nil
	case *visualbuildersdk.VbInstance:
		if current == nil {
			return visualbuildersdk.VbInstance{}, fmt.Errorf("current %s response is nil", vbInstanceKind)
		}
		return *current, nil
	case visualbuildersdk.VbInstanceSummary:
		return vbInstanceFromSummary(current), nil
	case *visualbuildersdk.VbInstanceSummary:
		if current == nil {
			return visualbuildersdk.VbInstance{}, fmt.Errorf("current %s response is nil", vbInstanceKind)
		}
		return vbInstanceFromSummary(*current), nil
	case visualbuildersdk.GetVbInstanceResponse:
		return current.VbInstance, nil
	case *visualbuildersdk.GetVbInstanceResponse:
		if current == nil {
			return visualbuildersdk.VbInstance{}, fmt.Errorf("current %s response is nil", vbInstanceKind)
		}
		return current.VbInstance, nil
	default:
		return visualbuildersdk.VbInstance{}, fmt.Errorf("unexpected current %s response type %T", vbInstanceKind, currentResponse)
	}
}

func vbInstanceFromSummary(summary visualbuildersdk.VbInstanceSummary) visualbuildersdk.VbInstance {
	return visualbuildersdk.VbInstance{
		Id:                       summary.Id,
		DisplayName:              summary.DisplayName,
		CompartmentId:            summary.CompartmentId,
		LifecycleState:           visualbuildersdk.VbInstanceLifecycleStateEnum(summary.LifecycleState),
		InstanceUrl:              summary.InstanceUrl,
		NodeCount:                summary.NodeCount,
		TimeCreated:              summary.TimeCreated,
		TimeUpdated:              summary.TimeUpdated,
		StateMessage:             summary.StateMessage,
		FreeformTags:             maps.Clone(summary.FreeformTags),
		DefinedTags:              cloneVbInstanceDefinedTags(summary.DefinedTags),
		SystemTags:               cloneVbInstanceDefinedTags(summary.SystemTags),
		IsVisualBuilderEnabled:   cloneVbInstanceBoolPtr(summary.IsVisualBuilderEnabled),
		CustomEndpoint:           summary.CustomEndpoint,
		AlternateCustomEndpoints: append([]visualbuildersdk.CustomEndpointDetails(nil), summary.AlternateCustomEndpoints...),
		ConsumptionModel:         visualbuildersdk.VbInstanceConsumptionModelEnum(summary.ConsumptionModel),
		NetworkEndpointDetails:   summary.NetworkEndpointDetails,
	}
}

func getVbInstanceWorkRequest(
	ctx context.Context,
	client vbInstanceOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s OCI client: %w", vbInstanceKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s work request client is not configured", vbInstanceKind)
	}

	response, err := client.GetWorkRequest(ctx, visualbuildersdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveVbInstanceWorkRequestAction(workRequest any) (string, error) {
	current, err := vbInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveVbInstanceWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := vbInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case visualbuildersdk.WorkRequestOperationTypeCreateVbInstance:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case visualbuildersdk.WorkRequestOperationTypeUpdateVbInstance:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case visualbuildersdk.WorkRequestOperationTypeDeleteVbInstance:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func recoverVbInstanceIDFromWorkRequest(
	_ *visualbuilderv1beta1.VbInstance,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := vbInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := vbInstanceWorkRequestActionForPhase(phase)
	if id, ok := resolveVbInstanceIDFromWorkRequestResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveVbInstanceIDFromWorkRequestResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("%s %s work request %s did not expose a %s identifier", vbInstanceKind, phase, vbInstanceStringValue(current.Id), vbInstanceKind)
}

func vbInstanceWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := vbInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", vbInstanceKind, phase, vbInstanceStringValue(current.Id), current.Status)
}

func vbInstanceWorkRequestFromAny(workRequest any) (visualbuildersdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case visualbuildersdk.WorkRequest:
		return current, nil
	case *visualbuildersdk.WorkRequest:
		if current == nil {
			return visualbuildersdk.WorkRequest{}, fmt.Errorf("%s work request is nil", vbInstanceKind)
		}
		return *current, nil
	default:
		return visualbuildersdk.WorkRequest{}, fmt.Errorf("unexpected %s work request type %T", vbInstanceKind, workRequest)
	}
}

func vbInstanceWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) visualbuildersdk.WorkRequestResourceActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return visualbuildersdk.WorkRequestResourceActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return visualbuildersdk.WorkRequestResourceActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return visualbuildersdk.WorkRequestResourceActionTypeDeleted
	default:
		return ""
	}
}

func resolveVbInstanceIDFromWorkRequestResources(
	resources []visualbuildersdk.WorkRequestResource,
	action visualbuildersdk.WorkRequestResourceActionTypeEnum,
	preferVbInstanceOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferVbInstanceOnly && !isVbInstanceWorkRequestResource(resource) {
			continue
		}
		id := strings.TrimSpace(vbInstanceStringValue(resource.Identifier))
		if id == "" {
			continue
		}
		if candidate == "" {
			candidate = id
			continue
		}
		if candidate != id {
			return "", false
		}
	}
	return candidate, candidate != ""
}

func isVbInstanceWorkRequestResource(resource visualbuildersdk.WorkRequestResource) bool {
	return normalizeVbInstanceToken(vbInstanceStringValue(resource.EntityType)) == "vbinstance"
}

func normalizeVbInstanceToken(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(value))
}

func vbInstanceDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	var out map[string]map[string]interface{}
	if err := vbInstanceJSONConvert(spec, &out); err != nil {
		return nil
	}
	if len(out) == 0 {
		return map[string]map[string]interface{}{}
	}
	return out
}

func vbInstanceJSONMap(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func vbInstancePrunedJSONMap(value any) (map[string]any, error) {
	values, err := vbInstanceJSONMap(value)
	if err != nil {
		return nil, err
	}
	pruned, ok := vbInstancePruneJSONValue(values)
	if !ok {
		return map[string]any{}, nil
	}
	prunedMap, ok := pruned.(map[string]any)
	if !ok || prunedMap == nil {
		return map[string]any{}, nil
	}
	return prunedMap, nil
}

func vbInstancePruneJSONValue(value any) (any, bool) {
	switch current := value.(type) {
	case nil:
		return nil, false
	case map[string]any:
		if len(current) == 0 {
			return map[string]any{}, true
		}
		pruned := make(map[string]any, len(current))
		for key, child := range current {
			prunedChild, ok := vbInstancePruneJSONValue(child)
			if !ok {
				continue
			}
			pruned[key] = prunedChild
		}
		if len(pruned) == 0 {
			return nil, false
		}
		return pruned, true
	case []any:
		if len(current) == 0 {
			return []any{}, true
		}
		pruned := make([]any, 0, len(current))
		for _, child := range current {
			prunedChild, ok := vbInstancePruneJSONValue(child)
			if !ok {
				continue
			}
			pruned = append(pruned, prunedChild)
		}
		if len(pruned) == 0 {
			return []any{}, true
		}
		return pruned, true
	default:
		return value, true
	}
}

func vbInstanceJSONConvert(source any, destination any) error {
	payload, err := json.Marshal(source)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, destination)
}

func vbInstanceMapSubsetEqual(want map[string]any, got map[string]any) bool {
	for key, wantValue := range want {
		gotValue, ok := got[key]
		if !ok {
			return false
		}
		if !vbInstanceJSONValueEqual(wantValue, gotValue) {
			return false
		}
	}
	return true
}

func vbInstanceJSONValueEqual(left any, right any) bool {
	leftMap, leftIsMap := left.(map[string]any)
	rightMap, rightIsMap := right.(map[string]any)
	switch {
	case leftIsMap && rightIsMap:
		return vbInstanceMapSubsetEqual(leftMap, rightMap)
	case leftIsMap || rightIsMap:
		return false
	}

	leftSlice, leftIsSlice := left.([]any)
	rightSlice, rightIsSlice := right.([]any)
	switch {
	case leftIsSlice && rightIsSlice:
		if len(leftSlice) != len(rightSlice) {
			return false
		}
		for i := range leftSlice {
			if !vbInstanceJSONValueEqual(leftSlice[i], rightSlice[i]) {
				return false
			}
		}
		return true
	case leftIsSlice || rightIsSlice:
		return false
	}

	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func cloneVbInstanceStringSlice(input []string) []string {
	if input == nil {
		return nil
	}
	if len(input) == 0 {
		return []string{}
	}
	return append([]string(nil), input...)
}

func cloneVbInstanceDefinedTags(input map[string]map[string]interface{}) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(input))
	for key, value := range input {
		cloned[key] = maps.Clone(value)
	}
	return cloned
}

func cloneVbInstanceBoolPtr(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func vbInstanceStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func newVbInstanceRuntimeConfigForTests(log loggerutil.OSOKLogger, client vbInstanceOCIClient) generatedruntime.Config[*visualbuilderv1beta1.VbInstance] {
	hooks := newVbInstanceRuntimeHooksWithOCIClient(client)
	applyVbInstanceRuntimeHooks(&hooks, client, nil)
	return buildVbInstanceGeneratedRuntimeConfig(&VbInstanceServiceManager{Log: log}, hooks)
}
