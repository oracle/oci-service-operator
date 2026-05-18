/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package gdppipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	gdpsdk "github.com/oracle/oci-go-sdk/v65/gdp"
	gdpv1beta1 "github.com/oracle/oci-service-operator/api/gdp/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

var gdpPipelineWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(gdpsdk.OperationStatusAccepted),
		string(gdpsdk.OperationStatusInProgress),
		string(gdpsdk.OperationStatusWaiting),
		string(gdpsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(gdpsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(gdpsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(gdpsdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(gdpsdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(gdpsdk.GdpOperationTypeCreateGdpPipeline)},
	UpdateActionTokens:    []string{string(gdpsdk.GdpOperationTypeUpdateGdpPipeline)},
	DeleteActionTokens:    []string{string(gdpsdk.GdpOperationTypeDeleteGdpPipeline)},
}

type gdpPipelineOCIClient interface {
	CreateGdpPipeline(context.Context, gdpsdk.CreateGdpPipelineRequest) (gdpsdk.CreateGdpPipelineResponse, error)
	GetGdpPipeline(context.Context, gdpsdk.GetGdpPipelineRequest) (gdpsdk.GetGdpPipelineResponse, error)
	ListGdpPipelines(context.Context, gdpsdk.ListGdpPipelinesRequest) (gdpsdk.ListGdpPipelinesResponse, error)
	UpdateGdpPipeline(context.Context, gdpsdk.UpdateGdpPipelineRequest) (gdpsdk.UpdateGdpPipelineResponse, error)
	DeleteGdpPipeline(context.Context, gdpsdk.DeleteGdpPipelineRequest) (gdpsdk.DeleteGdpPipelineResponse, error)
	GetGdpWorkRequest(context.Context, gdpsdk.GetGdpWorkRequestRequest) (gdpsdk.GetGdpWorkRequestResponse, error)
}

type gdpPipelineExpandedList struct {
	Items []gdpsdk.GdpPipeline
}

func init() {
	registerGdpPipelineRuntimeHooksMutator(func(manager *GdpPipelineServiceManager, hooks *GdpPipelineRuntimeHooks) {
		client, initErr := newGdpPipelineSDKClient(manager)
		applyGdpPipelineRuntimeHooks(hooks, client, initErr)
	})
}

func newGdpPipelineSDKClient(manager *GdpPipelineServiceManager) (gdpPipelineOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("GdpPipeline service manager is nil")
	}

	client, err := gdpsdk.NewGuardedDataPipelineClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyGdpPipelineRuntimeHooks(
	hooks *GdpPipelineRuntimeHooks,
	client gdpPipelineOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedGdpPipelineRuntimeSemantics()
	hooks.Identity.Resolve = resolveGdpPipelineLookupIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardGdpPipelineExistingBeforeCreate
	hooks.Identity.LookupExisting = lookupExistingGdpPipeline(client, initErr)
	hooks.Read.List = buildGdpPipelineReadListOperation(client, initErr)
	hooks.BuildCreateBody = func(
		ctx context.Context,
		resource *gdpv1beta1.GdpPipeline,
		namespace string,
	) (any, error) {
		return buildGdpPipelineCreateDetails(ctx, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *gdpv1beta1.GdpPipeline,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildGdpPipelineUpdateBody(resource, currentResponse)
	}
	hooks.Create.Fields = gdpPipelineCreateFields()
	hooks.Get.Fields = gdpPipelineGetFields()
	hooks.List.Fields = gdpPipelineListFields()
	hooks.Update.Fields = gdpPipelineUpdateFields()
	hooks.Delete.Fields = gdpPipelineDeleteFields()
	hooks.Async.Adapter = gdpPipelineWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getGdpPipelineWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveGdpPipelineGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveGdpPipelineGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverGdpPipelineIDFromGeneratedWorkRequest
	hooks.Async.Message = gdpPipelineGeneratedWorkRequestMessage
}

func newGdpPipelineServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client gdpPipelineOCIClient,
) GdpPipelineServiceClient {
	manager := &GdpPipelineServiceManager{Log: log}
	hooks := newGdpPipelineRuntimeHooksWithOCIClient(client)
	applyGdpPipelineRuntimeHooks(&hooks, client, nil)
	delegate := defaultGdpPipelineServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*gdpv1beta1.GdpPipeline](
			buildGdpPipelineGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapGdpPipelineGeneratedClient(hooks, delegate)
}

func newGdpPipelineRuntimeConfig(
	log loggerutil.OSOKLogger,
	client gdpPipelineOCIClient,
) generatedruntime.Config[*gdpv1beta1.GdpPipeline] {
	manager := &GdpPipelineServiceManager{Log: log}
	hooks := newGdpPipelineRuntimeHooksWithOCIClient(client)
	applyGdpPipelineRuntimeHooks(&hooks, client, nil)
	return buildGdpPipelineGeneratedRuntimeConfig(manager, hooks)
}

func newGdpPipelineRuntimeHooksWithOCIClient(client gdpPipelineOCIClient) GdpPipelineRuntimeHooks {
	return GdpPipelineRuntimeHooks{
		Semantics: reviewedGdpPipelineRuntimeSemantics(),
		Read:      generatedruntime.ReadHooks{},
		Create: runtimeOperationHooks[gdpsdk.CreateGdpPipelineRequest, gdpsdk.CreateGdpPipelineResponse]{
			Fields: gdpPipelineCreateFields(),
			Call: func(ctx context.Context, request gdpsdk.CreateGdpPipelineRequest) (gdpsdk.CreateGdpPipelineResponse, error) {
				return client.CreateGdpPipeline(ctx, request)
			},
		},
		Get: runtimeOperationHooks[gdpsdk.GetGdpPipelineRequest, gdpsdk.GetGdpPipelineResponse]{
			Fields: gdpPipelineGetFields(),
			Call: func(ctx context.Context, request gdpsdk.GetGdpPipelineRequest) (gdpsdk.GetGdpPipelineResponse, error) {
				return client.GetGdpPipeline(ctx, request)
			},
		},
		List: runtimeOperationHooks[gdpsdk.ListGdpPipelinesRequest, gdpsdk.ListGdpPipelinesResponse]{
			Fields: gdpPipelineListFields(),
			Call: func(ctx context.Context, request gdpsdk.ListGdpPipelinesRequest) (gdpsdk.ListGdpPipelinesResponse, error) {
				return client.ListGdpPipelines(ctx, request)
			},
		},
		Update: runtimeOperationHooks[gdpsdk.UpdateGdpPipelineRequest, gdpsdk.UpdateGdpPipelineResponse]{
			Fields: gdpPipelineUpdateFields(),
			Call: func(ctx context.Context, request gdpsdk.UpdateGdpPipelineRequest) (gdpsdk.UpdateGdpPipelineResponse, error) {
				return client.UpdateGdpPipeline(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[gdpsdk.DeleteGdpPipelineRequest, gdpsdk.DeleteGdpPipelineResponse]{
			Fields: gdpPipelineDeleteFields(),
			Call: func(ctx context.Context, request gdpsdk.DeleteGdpPipelineRequest) (gdpsdk.DeleteGdpPipelineResponse, error) {
				return client.DeleteGdpPipeline(ctx, request)
			},
		},
	}
}

func reviewedGdpPipelineRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newGdpPipelineRuntimeSemantics()
	semantics.Lifecycle = generatedruntime.LifecycleSemantics{
		ProvisioningStates: []string{string(gdpsdk.GdpPipelineLifecycleStateCreating)},
		UpdatingStates:     []string{string(gdpsdk.GdpPipelineLifecycleStateUpdating)},
		ActiveStates: []string{
			string(gdpsdk.GdpPipelineLifecycleStateActive),
			string(gdpsdk.GdpPipelineLifecycleStateInactive),
		},
	}
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields: []string{
			"bucketDetails",
			"compartmentId",
			"displayName",
			"id",
			"peeringRegion",
			"pipelineType",
		},
	}
	semantics.Hooks = generatedruntime.HookSet{
		Create: []generatedruntime.Hook{
			{Helper: "tfresource.CreateResource"},
			{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "gdppipeline", Action: "CREATED"},
		},
		Update: []generatedruntime.Hook{
			{Helper: "tfresource.UpdateResource"},
			{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "gdppipeline", Action: "UPDATED"},
		},
		Delete: []generatedruntime.Hook{
			{Helper: "tfresource.DeleteResource"},
			{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "gdppipeline", Action: "DELETED"},
		},
	}
	semantics.CreateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetGdpPipeline",
		Hooks:    append([]generatedruntime.Hook(nil), semantics.Hooks.Create...),
	}
	semantics.UpdateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetGdpPipeline",
		Hooks:    append([]generatedruntime.Hook(nil), semantics.Hooks.Update...),
	}
	semantics.DeleteFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetGdpPipeline/ListGdpPipelines confirm-delete",
		Hooks:    append([]generatedruntime.Hook(nil), semantics.Hooks.Delete...),
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func gdpPipelineCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateGdpPipelineDetails", RequestName: "CreateGdpPipelineDetails", Contribution: "body"},
	}
}

func gdpPipelineGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "GdpPipelineId", RequestName: "gdpPipelineId", Contribution: "path", PreferResourceID: true},
	}
}

func gdpPipelineListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
		{FieldName: "GdpPipelineId", RequestName: "gdpPipelineId", Contribution: "query", PreferResourceID: true},
	}
}

func gdpPipelineUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "GdpPipelineId", RequestName: "gdpPipelineId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateGdpPipelineDetails", RequestName: "UpdateGdpPipelineDetails", Contribution: "body"},
	}
}

func gdpPipelineDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "GdpPipelineId", RequestName: "gdpPipelineId", Contribution: "path", PreferResourceID: true},
	}
}

// Expand list reads into full GdpPipeline bodies so read-time identity matching
// can safely compare bucketDetails, which ListGdpPipelines summaries do not expose.
func buildGdpPipelineReadListOperation(
	client gdpPipelineOCIClient,
	initErr error,
) *generatedruntime.Operation {
	return &generatedruntime.Operation{
		NewRequest: func() any { return &gdpsdk.ListGdpPipelinesRequest{} },
		Fields:     gdpPipelineListFields(),
		Call: func(ctx context.Context, request any) (any, error) {
			return expandGdpPipelineReadList(ctx, client, initErr, *request.(*gdpsdk.ListGdpPipelinesRequest))
		},
	}
}

func expandGdpPipelineReadList(
	ctx context.Context,
	client gdpPipelineOCIClient,
	initErr error,
	request gdpsdk.ListGdpPipelinesRequest,
) (gdpPipelineExpandedList, error) {
	if initErr != nil {
		return gdpPipelineExpandedList{}, fmt.Errorf("initialize GdpPipeline OCI client: %w", initErr)
	}
	if client == nil {
		return gdpPipelineExpandedList{}, fmt.Errorf("GdpPipeline OCI client is not configured")
	}

	response, err := client.ListGdpPipelines(ctx, request)
	if err != nil {
		return gdpPipelineExpandedList{}, err
	}

	items := make([]gdpsdk.GdpPipeline, 0, len(response.Items))
	seen := make(map[string]struct{}, len(response.Items))
	for _, summary := range response.Items {
		pipelineID := gdpPipelineStringValue(summary.Id)
		if pipelineID == "" {
			continue
		}
		if _, ok := seen[pipelineID]; ok {
			continue
		}
		seen[pipelineID] = struct{}{}

		full, err := client.GetGdpPipeline(ctx, gdpsdk.GetGdpPipelineRequest{
			GdpPipelineId: common.String(pipelineID),
		})
		if err != nil {
			if gdpPipelineReadNotFound(err) {
				continue
			}
			return gdpPipelineExpandedList{}, err
		}
		items = append(items, full.GdpPipeline)
	}

	return gdpPipelineExpandedList{Items: items}, nil
}

func guardGdpPipelineExistingBeforeCreate(
	_ context.Context,
	resource *gdpv1beta1.GdpPipeline,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("GdpPipeline resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" ||
		strings.TrimSpace(resource.Spec.PipelineType) == "" ||
		strings.TrimSpace(resource.Spec.PeeringRegion) == "" ||
		len(resource.Spec.BucketDetails) == 0 {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func resolveGdpPipelineLookupIdentity(resource *gdpv1beta1.GdpPipeline) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("GdpPipeline resource is nil")
	}
	if trackedID := strings.TrimSpace(gdpPipelineCurrentTrackedID(resource)); trackedID != "" {
		return trackedID, nil
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" ||
		strings.TrimSpace(resource.Spec.PipelineType) == "" ||
		strings.TrimSpace(resource.Spec.PeeringRegion) == "" ||
		len(resource.Spec.BucketDetails) == 0 {
		return nil, nil
	}
	return resource.Spec.DisplayName, nil
}

func lookupExistingGdpPipeline(
	client gdpPipelineOCIClient,
	initErr error,
) func(context.Context, *gdpv1beta1.GdpPipeline, any) (any, error) {
	return func(ctx context.Context, resource *gdpv1beta1.GdpPipeline, _ any) (any, error) {
		if resource == nil {
			return nil, fmt.Errorf("GdpPipeline resource is nil")
		}

		request := gdpsdk.ListGdpPipelinesRequest{}
		trackedID := strings.TrimSpace(gdpPipelineCurrentTrackedID(resource))
		if trackedID != "" {
			request.GdpPipelineId = common.String(trackedID)
		} else {
			if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
				return nil, nil
			}
			request.CompartmentId = common.String(resource.Spec.CompartmentId)
			request.DisplayName = common.String(resource.Spec.DisplayName)
		}

		expanded, err := expandGdpPipelineReadList(ctx, client, initErr, request)
		if err != nil {
			return nil, err
		}

		matches := make([]gdpsdk.GdpPipeline, 0, len(expanded.Items))
		for _, candidate := range expanded.Items {
			switch {
			case trackedID != "" && strings.TrimSpace(gdpPipelineStringValue(candidate.Id)) == trackedID:
				matches = append(matches, candidate)
			case trackedID == "" && gdpPipelineMatchesCreateIdentity(resource, candidate):
				matches = append(matches, candidate)
			}
		}

		switch len(matches) {
		case 0:
			return nil, nil
		case 1:
			return matches[0], nil
		default:
			return nil, fmt.Errorf("GdpPipeline lookup returned multiple exact matches")
		}
	}
}

func buildGdpPipelineCreateDetails(
	ctx context.Context,
	resource *gdpv1beta1.GdpPipeline,
	namespace string,
) (gdpsdk.CreateGdpPipelineDetails, error) {
	if resource == nil {
		return gdpsdk.CreateGdpPipelineDetails{}, fmt.Errorf("GdpPipeline resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return gdpsdk.CreateGdpPipelineDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return gdpsdk.CreateGdpPipelineDetails{}, fmt.Errorf("marshal resolved GdpPipeline spec: %w", err)
	}

	var details gdpsdk.CreateGdpPipelineDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return gdpsdk.CreateGdpPipelineDetails{}, fmt.Errorf("decode GdpPipeline create request body: %w", err)
	}

	return details, nil
}

func buildGdpPipelineUpdateBody(
	resource *gdpv1beta1.GdpPipeline,
	currentResponse any,
) (gdpsdk.UpdateGdpPipelineDetails, bool, error) {
	if resource == nil {
		return gdpsdk.UpdateGdpPipelineDetails{}, false, fmt.Errorf("GdpPipeline resource is nil")
	}

	current, err := gdpPipelineFromResponse(currentResponse)
	if err != nil {
		return gdpsdk.UpdateGdpPipelineDetails{}, false, err
	}

	details := gdpsdk.UpdateGdpPipelineDetails{}
	updateNeeded := false

	if desired, ok := gdpPipelineDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := gdpPipelineDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := gdpPipelineDesiredStringUpdate(resource.Spec.ServiceLogGroupId, current.ServiceLogGroupId); ok {
		details.ServiceLogGroupId = desired
		updateNeeded = true
	}
	if desired, ok := gdpPipelineDesiredStringSliceUpdate(resource.Spec.FileTypes, current.FileTypes); ok {
		details.FileTypes = desired
		updateNeeded = true
	}
	if desired, ok := gdpPipelineDesiredStringUpdate(resource.Spec.AuthorizationDetails, current.AuthorizationDetails); ok {
		details.AuthorizationDetails = desired
		updateNeeded = true
	}
	if desired, ok := gdpPipelineDesiredBoolUpdate(resource.Spec.IsFileOverrideInDestinationEnabled, current.IsFileOverrideInDestinationEnabled); ok {
		details.IsFileOverrideInDestinationEnabled = desired
		updateNeeded = true
	}
	if desired, ok := gdpPipelineDesiredBoolUpdate(resource.Spec.IsScanningEnabled, current.IsScanningEnabled); ok {
		details.IsScanningEnabled = desired
		updateNeeded = true
	}
	if desired, ok := gdpPipelineDesiredBoolUpdate(resource.Spec.IsChunkingEnabled, current.IsChunkingEnabled); ok {
		details.IsChunkingEnabled = desired
		updateNeeded = true
	}
	if desired, ok := gdpPipelineDesiredBoolUpdate(resource.Spec.IsApprovalNeeded, current.IsApprovalNeeded); ok {
		details.IsApprovalNeeded = desired
		updateNeeded = true
	}
	if desired, ok := gdpPipelineDesiredStringUpdate(resource.Spec.ApprovalKeyVaultId, current.ApprovalKeyVaultId); ok {
		details.ApprovalKeyVaultId = desired
		updateNeeded = true
	}
	if desired, ok := gdpPipelineDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := gdpPipelineDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func gdpPipelineFromResponse(currentResponse any) (gdpsdk.GdpPipeline, error) {
	switch current := currentResponse.(type) {
	case gdpsdk.GdpPipeline:
		return current, nil
	case *gdpsdk.GdpPipeline:
		if current == nil {
			return gdpsdk.GdpPipeline{}, fmt.Errorf("current GdpPipeline response is nil")
		}
		return *current, nil
	case gdpsdk.GdpPipelineSummary:
		return gdpsdk.GdpPipeline{
			Id:                                 current.Id,
			CompartmentId:                      current.CompartmentId,
			LifecycleState:                     current.LifecycleState,
			DisplayName:                        current.DisplayName,
			PipelineType:                       current.PipelineType,
			PeeringRegion:                      current.PeeringRegion,
			TimeCreated:                        current.TimeCreated,
			TimeUpdated:                        current.TimeUpdated,
			FreeformTags:                       current.FreeformTags,
			DefinedTags:                        current.DefinedTags,
			Description:                        current.Description,
			ServiceLogGroupId:                  current.ServiceLogGroupId,
			AuthorizationDetails:               current.AuthorizationDetails,
			IsFileOverrideInDestinationEnabled: current.IsFileOverrideInDestinationEnabled,
			IsScanningEnabled:                  current.IsScanningEnabled,
			IsChunkingEnabled:                  current.IsChunkingEnabled,
			IsApprovalNeeded:                   current.IsApprovalNeeded,
			SystemTags:                         current.SystemTags,
		}, nil
	case *gdpsdk.GdpPipelineSummary:
		if current == nil {
			return gdpsdk.GdpPipeline{}, fmt.Errorf("current GdpPipeline response is nil")
		}
		return gdpPipelineFromResponse(*current)
	case gdpsdk.GetGdpPipelineResponse:
		return current.GdpPipeline, nil
	case *gdpsdk.GetGdpPipelineResponse:
		if current == nil {
			return gdpsdk.GdpPipeline{}, fmt.Errorf("current GdpPipeline response is nil")
		}
		return current.GdpPipeline, nil
	default:
		return gdpsdk.GdpPipeline{}, fmt.Errorf("unexpected current GdpPipeline response type %T", currentResponse)
	}
}

func gdpPipelineDesiredStringUpdate(spec string, current *string) (*string, bool) {
	currentValue := ""
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	if spec == "" && current == nil {
		return nil, false
	}
	return common.String(spec), true
}

func gdpPipelineCurrentTrackedID(resource *gdpv1beta1.GdpPipeline) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func gdpPipelineMatchesCreateIdentity(resource *gdpv1beta1.GdpPipeline, current gdpsdk.GdpPipeline) bool {
	if resource == nil {
		return false
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) != gdpPipelineStringValue(current.CompartmentId) {
		return false
	}
	if strings.TrimSpace(resource.Spec.DisplayName) != gdpPipelineStringValue(current.DisplayName) {
		return false
	}
	if strings.TrimSpace(resource.Spec.PipelineType) != string(current.PipelineType) {
		return false
	}
	if strings.TrimSpace(resource.Spec.PeeringRegion) != gdpPipelineStringValue(current.PeeringRegion) {
		return false
	}
	return gdpPipelineJSONEqual(resource.Spec.BucketDetails, current.BucketDetails)
}

func gdpPipelineDesiredBoolUpdate(spec bool, current *bool) (*bool, bool) {
	currentValue := false
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	return common.Bool(spec), true
}

func gdpPipelineDesiredStringSliceUpdate(spec []string, current []string) ([]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 {
		return nil, false
	}
	if gdpPipelineStringSliceEqual(spec, current) {
		return nil, false
	}
	return append([]string(nil), spec...), true
}

func gdpPipelineDesiredFreeformTagsUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if gdpPipelineStringMapEqual(spec, current) {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	return cloneGdpPipelineStringMap(spec), true
}

func gdpPipelineDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := gdpPipelineDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if gdpPipelineJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func cloneGdpPipelineStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func gdpPipelineDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}

	desired := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		desired[namespace] = converted
	}
	return desired
}

func gdpPipelineJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func gdpPipelineStringSliceEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func gdpPipelineStringMapEqual(left map[string]string, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValue := range left {
		if rightValue, ok := right[key]; !ok || leftValue != rightValue {
			return false
		}
	}
	return true
}

func getGdpPipelineWorkRequest(
	ctx context.Context,
	client gdpPipelineOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize GdpPipeline OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("GdpPipeline OCI client is not configured")
	}

	response, err := client.GetGdpWorkRequest(ctx, gdpsdk.GetGdpWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.GdpWorkRequest, nil
}

func resolveGdpPipelineGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := gdpPipelineWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveGdpPipelineGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := gdpPipelineWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := gdpPipelineWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverGdpPipelineIDFromGeneratedWorkRequest(
	_ *gdpv1beta1.GdpPipeline,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := gdpPipelineWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := gdpPipelineWorkRequestActionForPhase(phase)
	if id, ok := resolveGdpPipelineIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveGdpPipelineIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("GdpPipeline work request %s does not expose a pipeline identifier", gdpPipelineStringValue(current.Id))
}

func gdpPipelineGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := gdpPipelineWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("GdpPipeline %s work request %s is %s", phase, gdpPipelineStringValue(current.Id), current.Status)
}

func gdpPipelineWorkRequestFromAny(workRequest any) (gdpsdk.GdpWorkRequest, error) {
	switch current := workRequest.(type) {
	case gdpsdk.GdpWorkRequest:
		return current, nil
	case *gdpsdk.GdpWorkRequest:
		if current == nil {
			return gdpsdk.GdpWorkRequest{}, fmt.Errorf("GdpPipeline work request is nil")
		}
		return *current, nil
	default:
		return gdpsdk.GdpWorkRequest{}, fmt.Errorf("unexpected GdpPipeline work request type %T", workRequest)
	}
}

func gdpPipelineWorkRequestPhaseFromOperationType(operationType gdpsdk.GdpOperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case gdpsdk.GdpOperationTypeCreateGdpPipeline:
		return shared.OSOKAsyncPhaseCreate, true
	case gdpsdk.GdpOperationTypeUpdateGdpPipeline:
		return shared.OSOKAsyncPhaseUpdate, true
	case gdpsdk.GdpOperationTypeDeleteGdpPipeline:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func gdpPipelineWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) gdpsdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return gdpsdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return gdpsdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return gdpsdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveGdpPipelineIDFromResources(
	resources []gdpsdk.WorkRequestResource,
	action gdpsdk.ActionTypeEnum,
	preferPipelineOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferPipelineOnly && !isGdpPipelineWorkRequestResource(resource) {
			continue
		}
		id := strings.TrimSpace(gdpPipelineStringValue(resource.Identifier))
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

func isGdpPipelineWorkRequestResource(resource gdpsdk.WorkRequestResource) bool {
	token := normalizeGdpPipelineWorkRequestToken(gdpPipelineStringValue(resource.EntityType))
	return token == "gdppipeline" || token == "pipeline"
}

func normalizeGdpPipelineWorkRequestToken(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(value))
}

func gdpPipelineReadNotFound(err error) bool {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return false
	}
	statusCode := serviceErr.GetHTTPStatusCode()
	errorCode := strings.TrimSpace(serviceErr.GetCode())
	return statusCode == 404 && (errorCode == errorutil.NotFound || errorCode == errorutil.NotAuthorizedOrNotFound)
}

func gdpPipelineStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
