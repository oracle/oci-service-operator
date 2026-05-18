/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package aidataplatform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"reflect"
	"strings"

	aidataplatformsdk "github.com/oracle/oci-go-sdk/v65/aidataplatform"
	"github.com/oracle/oci-go-sdk/v65/common"
	aidataplatformv1beta1 "github.com/oracle/oci-service-operator/api/aidataplatform/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	aiDataPlatformKind                               = "AiDataPlatform"
	aiDataPlatformDefaultWorkspaceNameFingerprintKey = "aiDataPlatformDefaultWorkspaceNameSHA256="
)

var aiDataPlatformWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(aidataplatformsdk.OperationStatusAccepted),
		string(aidataplatformsdk.OperationStatusInProgress),
		string(aidataplatformsdk.OperationStatusWaiting),
		string(aidataplatformsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(aidataplatformsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(aidataplatformsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(aidataplatformsdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(aidataplatformsdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(aidataplatformsdk.ActionTypeCreated)},
	UpdateActionTokens:    []string{string(aidataplatformsdk.ActionTypeUpdated)},
	DeleteActionTokens:    []string{string(aidataplatformsdk.ActionTypeDeleted)},
}

type aiDataPlatformOCIClient interface {
	CreateAiDataPlatform(context.Context, aidataplatformsdk.CreateAiDataPlatformRequest) (aidataplatformsdk.CreateAiDataPlatformResponse, error)
	GetAiDataPlatform(context.Context, aidataplatformsdk.GetAiDataPlatformRequest) (aidataplatformsdk.GetAiDataPlatformResponse, error)
	ListAiDataPlatforms(context.Context, aidataplatformsdk.ListAiDataPlatformsRequest) (aidataplatformsdk.ListAiDataPlatformsResponse, error)
	UpdateAiDataPlatform(context.Context, aidataplatformsdk.UpdateAiDataPlatformRequest) (aidataplatformsdk.UpdateAiDataPlatformResponse, error)
	DeleteAiDataPlatform(context.Context, aidataplatformsdk.DeleteAiDataPlatformRequest) (aidataplatformsdk.DeleteAiDataPlatformResponse, error)
	GetWorkRequest(context.Context, aidataplatformsdk.GetWorkRequestRequest) (aidataplatformsdk.GetWorkRequestResponse, error)
}

type aiDataPlatformAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e aiDataPlatformAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e aiDataPlatformAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerAiDataPlatformRuntimeHooksMutator(func(manager *AiDataPlatformServiceManager, hooks *AiDataPlatformRuntimeHooks) {
		client, initErr := newAiDataPlatformSDKClient(manager)
		applyAiDataPlatformRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newAiDataPlatformSDKClient(manager *AiDataPlatformServiceManager) (aiDataPlatformOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("AiDataPlatform service manager is nil")
	}
	client, err := aidataplatformsdk.NewAiDataPlatformClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyAiDataPlatformRuntimeHooks(
	manager *AiDataPlatformServiceManager,
	hooks *AiDataPlatformRuntimeHooks,
	client aiDataPlatformOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newAiDataPlatformRuntimeSemantics()
	hooks.BuildCreateBody = buildAiDataPlatformCreateBody
	hooks.BuildUpdateBody = buildAiDataPlatformUpdateBody
	hooks.Create.Fields = aiDataPlatformCreateFields()
	hooks.Get.Fields = aiDataPlatformGetFields()
	hooks.List.Fields = aiDataPlatformListFields()
	hooks.Update.Fields = aiDataPlatformUpdateFields()
	hooks.Delete.Fields = aiDataPlatformDeleteFields()
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateAiDataPlatformCreateOnlyDrift
	hooks.Read.Get = aiDataPlatformGuardedGetReadOperation(hooks)
	if hooks.List.Call != nil {
		hooks.List.Call = listAiDataPlatformsAllPages(hooks.List.Call)
	}
	hooks.DeleteHooks.HandleError = handleAiDataPlatformDeleteError
	hooks.Async.Adapter = aiDataPlatformWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getAiDataPlatformWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveAiDataPlatformGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveAiDataPlatformGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverAiDataPlatformIDFromGeneratedWorkRequest
	hooks.Async.Message = aiDataPlatformGeneratedWorkRequestMessage
	wrapAiDataPlatformCreateOnlyTracking(hooks)
	wrapAiDataPlatformDeleteConfirmation(hooks)
}

func newAiDataPlatformServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client aiDataPlatformOCIClient,
) AiDataPlatformServiceClient {
	manager := &AiDataPlatformServiceManager{Log: log}
	hooks := newAiDataPlatformRuntimeHooksWithOCIClient(client)
	applyAiDataPlatformRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultAiDataPlatformServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*aidataplatformv1beta1.AiDataPlatform](
			buildAiDataPlatformGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapAiDataPlatformGeneratedClient(hooks, delegate)
}

func newAiDataPlatformRuntimeHooksWithOCIClient(client aiDataPlatformOCIClient) AiDataPlatformRuntimeHooks {
	return AiDataPlatformRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*aidataplatformv1beta1.AiDataPlatform]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*aidataplatformv1beta1.AiDataPlatform]{},
		StatusHooks:     generatedruntime.StatusHooks[*aidataplatformv1beta1.AiDataPlatform]{},
		ParityHooks:     generatedruntime.ParityHooks[*aidataplatformv1beta1.AiDataPlatform]{},
		Async:           generatedruntime.AsyncHooks[*aidataplatformv1beta1.AiDataPlatform]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*aidataplatformv1beta1.AiDataPlatform]{},
		Create: runtimeOperationHooks[aidataplatformsdk.CreateAiDataPlatformRequest, aidataplatformsdk.CreateAiDataPlatformResponse]{
			Fields: aiDataPlatformCreateFields(),
			Call: func(ctx context.Context, request aidataplatformsdk.CreateAiDataPlatformRequest) (aidataplatformsdk.CreateAiDataPlatformResponse, error) {
				if client == nil {
					return aidataplatformsdk.CreateAiDataPlatformResponse{}, fmt.Errorf("AiDataPlatform OCI client is nil")
				}
				return client.CreateAiDataPlatform(ctx, request)
			},
		},
		Get: runtimeOperationHooks[aidataplatformsdk.GetAiDataPlatformRequest, aidataplatformsdk.GetAiDataPlatformResponse]{
			Fields: aiDataPlatformGetFields(),
			Call: func(ctx context.Context, request aidataplatformsdk.GetAiDataPlatformRequest) (aidataplatformsdk.GetAiDataPlatformResponse, error) {
				if client == nil {
					return aidataplatformsdk.GetAiDataPlatformResponse{}, fmt.Errorf("AiDataPlatform OCI client is nil")
				}
				return client.GetAiDataPlatform(ctx, request)
			},
		},
		List: runtimeOperationHooks[aidataplatformsdk.ListAiDataPlatformsRequest, aidataplatformsdk.ListAiDataPlatformsResponse]{
			Fields: aiDataPlatformListFields(),
			Call: func(ctx context.Context, request aidataplatformsdk.ListAiDataPlatformsRequest) (aidataplatformsdk.ListAiDataPlatformsResponse, error) {
				if client == nil {
					return aidataplatformsdk.ListAiDataPlatformsResponse{}, fmt.Errorf("AiDataPlatform OCI client is nil")
				}
				return client.ListAiDataPlatforms(ctx, request)
			},
		},
		Update: runtimeOperationHooks[aidataplatformsdk.UpdateAiDataPlatformRequest, aidataplatformsdk.UpdateAiDataPlatformResponse]{
			Fields: aiDataPlatformUpdateFields(),
			Call: func(ctx context.Context, request aidataplatformsdk.UpdateAiDataPlatformRequest) (aidataplatformsdk.UpdateAiDataPlatformResponse, error) {
				if client == nil {
					return aidataplatformsdk.UpdateAiDataPlatformResponse{}, fmt.Errorf("AiDataPlatform OCI client is nil")
				}
				return client.UpdateAiDataPlatform(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[aidataplatformsdk.DeleteAiDataPlatformRequest, aidataplatformsdk.DeleteAiDataPlatformResponse]{
			Fields: aiDataPlatformDeleteFields(),
			Call: func(ctx context.Context, request aidataplatformsdk.DeleteAiDataPlatformRequest) (aidataplatformsdk.DeleteAiDataPlatformResponse, error) {
				if client == nil {
					return aidataplatformsdk.DeleteAiDataPlatformResponse{}, fmt.Errorf("AiDataPlatform OCI client is nil")
				}
				return client.DeleteAiDataPlatform(ctx, request)
			},
		},
		WrapGeneratedClient: []func(AiDataPlatformServiceClient) AiDataPlatformServiceClient{},
	}
}

func newAiDataPlatformRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "aidataplatform",
		FormalSlug:    "aidataplatform",
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
			ProvisioningStates: []string{string(aidataplatformsdk.AiDataPlatformLifecycleStateCreating)},
			UpdatingStates:     []string{string(aidataplatformsdk.AiDataPlatformLifecycleStateUpdating)},
			ActiveStates:       []string{string(aidataplatformsdk.AiDataPlatformLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(aidataplatformsdk.AiDataPlatformLifecycleStateDeleting)},
			TerminalStates: []string{string(aidataplatformsdk.AiDataPlatformLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"aiDataPlatformType",
				"freeformTags",
				"definedTags",
				"systemTags",
			},
			ForceNew:      []string{"compartmentId", "defaultWorkspaceName"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetAiDataPlatform",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "aidataplatform", Action: "CREATED"},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetAiDataPlatform",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "aidataplatform", Action: "UPDATED"},
			},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "aidataplatform", Action: "DELETED"},
			},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func aiDataPlatformCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateAiDataPlatformDetails", RequestName: "CreateAiDataPlatformDetails", Contribution: "body"},
	}
}

func aiDataPlatformGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AiDataPlatformId", RequestName: "aiDataPlatformId", Contribution: "path", PreferResourceID: true},
	}
}

func aiDataPlatformListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true, LookupPaths: []string{"status.id", "status.status.ocid", "id", "ocid"}},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "ExcludeLifecycleState", RequestName: "excludeLifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "IncludeLegacy", RequestName: "includeLegacy", Contribution: "query"},
	}
}

func aiDataPlatformUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AiDataPlatformId", RequestName: "aiDataPlatformId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateAiDataPlatformDetails", RequestName: "UpdateAiDataPlatformDetails", Contribution: "body"},
	}
}

func aiDataPlatformDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AiDataPlatformId", RequestName: "aiDataPlatformId", Contribution: "path", PreferResourceID: true},
	}
}

func buildAiDataPlatformCreateBody(
	_ context.Context,
	resource *aidataplatformv1beta1.AiDataPlatform,
	_ string,
) (any, error) {
	if resource == nil {
		return aidataplatformsdk.CreateAiDataPlatformDetails{}, fmt.Errorf("AiDataPlatform resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return aidataplatformsdk.CreateAiDataPlatformDetails{}, fmt.Errorf("AiDataPlatform spec is missing required field compartmentId")
	}

	spec := resource.Spec
	body := aidataplatformsdk.CreateAiDataPlatformDetails{
		CompartmentId: common.String(strings.TrimSpace(spec.CompartmentId)),
	}
	if value := optionalAiDataPlatformString(spec.DisplayName); value != nil {
		body.DisplayName = value
	}
	if value := optionalAiDataPlatformString(spec.AiDataPlatformType); value != nil {
		body.AiDataPlatformType = value
	}
	if value := optionalAiDataPlatformString(spec.DefaultWorkspaceName); value != nil {
		body.DefaultWorkspaceName = value
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	if spec.SystemTags != nil {
		body.SystemTags = *util.ConvertToOciDefinedTags(&spec.SystemTags)
	}
	return body, nil
}

func buildAiDataPlatformUpdateBody(
	_ context.Context,
	resource *aidataplatformv1beta1.AiDataPlatform,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return aidataplatformsdk.UpdateAiDataPlatformDetails{}, false, fmt.Errorf("AiDataPlatform resource is nil")
	}

	current, ok := aiDataPlatformProjectionFromResponse(currentResponse)
	if !ok {
		return aidataplatformsdk.UpdateAiDataPlatformDetails{}, false, fmt.Errorf("current AiDataPlatform response does not expose an AiDataPlatform body")
	}

	spec := resource.Spec
	body := aidataplatformsdk.UpdateAiDataPlatformDetails{}
	updateNeeded := false
	if desired, ok := desiredAiDataPlatformStringUpdate(spec.DisplayName, current.DisplayName); ok {
		body.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := desiredAiDataPlatformStringUpdate(spec.AiDataPlatformType, current.AiDataPlatformType); ok {
		body.AiDataPlatformType = desired
		updateNeeded = true
	}
	if desired, ok := desiredAiDataPlatformFreeformTagsUpdate(spec.FreeformTags, current.FreeformTags); ok {
		body.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := desiredAiDataPlatformDefinedTagsUpdate(spec.DefinedTags, current.DefinedTags); ok {
		body.DefinedTags = desired
		updateNeeded = true
	}
	if desired, ok := desiredAiDataPlatformDefinedTagsUpdate(spec.SystemTags, current.SystemTags); ok {
		body.SystemTags = desired
		updateNeeded = true
	}
	return body, updateNeeded, nil
}

type aiDataPlatformStatusProjection struct {
	Id                 string                     `json:"id,omitempty"`
	DisplayName        string                     `json:"displayName,omitempty"`
	CompartmentId      string                     `json:"compartmentId,omitempty"`
	AiDataPlatformType string                     `json:"aiDataPlatformType,omitempty"`
	TimeCreated        string                     `json:"timeCreated,omitempty"`
	LifecycleState     string                     `json:"lifecycleState,omitempty"`
	FreeformTags       map[string]string          `json:"freeformTags,omitempty"`
	DefinedTags        map[string]shared.MapValue `json:"definedTags,omitempty"`
	CreatedBy          string                     `json:"createdBy,omitempty"`
	TimeUpdated        string                     `json:"timeUpdated,omitempty"`
	AliasKey           string                     `json:"aliasKey,omitempty"`
	WebSocketEndpoint  string                     `json:"webSocketEndpoint,omitempty"`
	LifecycleDetails   string                     `json:"lifecycleDetails,omitempty"`
	SystemTags         map[string]shared.MapValue `json:"systemTags,omitempty"`
}

type aiDataPlatformProjectedResponse struct {
	AiDataPlatform aiDataPlatformStatusProjection `presentIn:"body"`
	OpcRequestId   *string                        `presentIn:"header" name:"opc-request-id"`
}

func aiDataPlatformGuardedGetReadOperation(hooks *AiDataPlatformRuntimeHooks) *generatedruntime.Operation {
	if hooks == nil || hooks.Get.Call == nil {
		return nil
	}

	getCall := hooks.Get.Call
	fields := append([]generatedruntime.RequestField(nil), hooks.Get.Fields...)
	return &generatedruntime.Operation{
		NewRequest: func() any { return &aidataplatformsdk.GetAiDataPlatformRequest{} },
		Fields:     fields,
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*aidataplatformsdk.GetAiDataPlatformRequest)
			if !ok {
				return nil, fmt.Errorf("expected *aidataplatform.GetAiDataPlatformRequest, got %T", request)
			}
			response, err := getCall(ctx, *typed)
			if err != nil {
				return nil, aiDataPlatformConservativeNotFoundError("AiDataPlatform read returned ambiguous 404 NotAuthorizedOrNotFound", err)
			}
			return response, nil
		},
	}
}

func listAiDataPlatformsAllPages(
	call func(context.Context, aidataplatformsdk.ListAiDataPlatformsRequest) (aidataplatformsdk.ListAiDataPlatformsResponse, error),
) func(context.Context, aidataplatformsdk.ListAiDataPlatformsRequest) (aidataplatformsdk.ListAiDataPlatformsResponse, error) {
	return func(ctx context.Context, request aidataplatformsdk.ListAiDataPlatformsRequest) (aidataplatformsdk.ListAiDataPlatformsResponse, error) {
		seenPages := map[string]struct{}{}
		var combined aidataplatformsdk.ListAiDataPlatformsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return aidataplatformsdk.ListAiDataPlatformsResponse{}, err
			}
			combined.RawResponse = response.RawResponse
			combined.OpcRequestId = response.OpcRequestId
			combined.Items = append(combined.Items, response.Items...)

			nextPage := strings.TrimSpace(stringValue(response.OpcNextPage))
			if nextPage == "" {
				combined.OpcNextPage = nil
				return combined, nil
			}
			if _, ok := seenPages[nextPage]; ok {
				return aidataplatformsdk.ListAiDataPlatformsResponse{}, fmt.Errorf("AiDataPlatform list pagination repeated page token %q", nextPage)
			}
			seenPages[nextPage] = struct{}{}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func getAiDataPlatformWorkRequest(
	ctx context.Context,
	client aiDataPlatformOCIClient,
	initErr error,
	workRequestID string,
) (aidataplatformsdk.WorkRequest, error) {
	if initErr != nil {
		return aidataplatformsdk.WorkRequest{}, initErr
	}
	if client == nil {
		return aidataplatformsdk.WorkRequest{}, fmt.Errorf("AiDataPlatform OCI client is nil")
	}
	if strings.TrimSpace(workRequestID) == "" {
		return aidataplatformsdk.WorkRequest{}, fmt.Errorf("AiDataPlatform work request id is empty")
	}
	response, err := client.GetWorkRequest(ctx, aidataplatformsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return aidataplatformsdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func resolveAiDataPlatformGeneratedWorkRequestAction(workRequest any) (string, error) {
	typed, err := aiDataPlatformWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveAiDataPlatformWorkRequestAction(typed)
}

func resolveAiDataPlatformGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	typed, err := aiDataPlatformWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := aiDataPlatformWorkRequestPhaseFromOperationType(typed.OperationType)
	return phase, ok, nil
}

func recoverAiDataPlatformIDFromGeneratedWorkRequest(
	_ *aidataplatformv1beta1.AiDataPlatform,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	typed, err := aiDataPlatformWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveAiDataPlatformIDFromWorkRequest(typed, aiDataPlatformWorkRequestActionForPhase(phase))
}

func aiDataPlatformGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	typed, err := aiDataPlatformWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("AiDataPlatform %s work request %s is %s", phase, stringValue(typed.Id), typed.Status)
}

func resolveAiDataPlatformIDFromWorkRequest(
	workRequest aidataplatformsdk.WorkRequest,
	action aidataplatformsdk.ActionTypeEnum,
) (string, error) {
	if id, ok := resolveAiDataPlatformIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveAiDataPlatformIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("AiDataPlatform work request %s does not expose an AiDataPlatform identifier", stringValue(workRequest.Id))
}

func resolveAiDataPlatformIDFromResources(
	resources []aidataplatformsdk.WorkRequestResource,
	action aidataplatformsdk.ActionTypeEnum,
	preferAiDataPlatformOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferAiDataPlatformOnly && !isAiDataPlatformWorkRequestResource(resource) {
			continue
		}

		id := strings.TrimSpace(stringValue(resource.Identifier))
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

func resolveAiDataPlatformWorkRequestAction(workRequest aidataplatformsdk.WorkRequest) (string, error) {
	var action string
	for _, resource := range workRequest.Resources {
		if !isAiDataPlatformWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" || strings.EqualFold(candidate, string(aidataplatformsdk.ActionTypeInProgress)) {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf("AiDataPlatform work request %s exposes conflicting action types %q and %q", stringValue(workRequest.Id), action, candidate)
		}
	}
	return action, nil
}

func aiDataPlatformWorkRequestPhaseFromOperationType(
	operationType aidataplatformsdk.OperationTypeEnum,
) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case aidataplatformsdk.OperationTypeCreateDataLake:
		return shared.OSOKAsyncPhaseCreate, true
	case aidataplatformsdk.OperationTypeUpdateDataLake:
		return shared.OSOKAsyncPhaseUpdate, true
	case aidataplatformsdk.OperationTypeDeleteDataLake:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func aiDataPlatformWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) aidataplatformsdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return aidataplatformsdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return aidataplatformsdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return aidataplatformsdk.ActionTypeDeleted
	default:
		return ""
	}
}

func aiDataPlatformWorkRequestFromAny(workRequest any) (aidataplatformsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case aidataplatformsdk.WorkRequest:
		return current, nil
	case *aidataplatformsdk.WorkRequest:
		if current == nil {
			return aidataplatformsdk.WorkRequest{}, fmt.Errorf("AiDataPlatform work request is nil")
		}
		return *current, nil
	default:
		return aidataplatformsdk.WorkRequest{}, fmt.Errorf("unexpected AiDataPlatform work request type %T", workRequest)
	}
}

func isAiDataPlatformWorkRequestResource(resource aidataplatformsdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	switch entityType {
	case "aidataplatform", "ai_data_platform", "data_lake", "datalake", "data-lake":
		return true
	}
	if strings.Contains(entityType, "data") && strings.Contains(entityType, "lake") {
		return true
	}
	if strings.Contains(entityType, "aidataplatform") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/ai-data-platforms/") || strings.Contains(entityURI, "/data-lakes/")
}

func validateAiDataPlatformCreateOnlyDrift(resource *aidataplatformv1beta1.AiDataPlatform, _ any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", aiDataPlatformKind)
	}
	recorded, ok := aiDataPlatformRecordedDefaultWorkspaceNameFingerprint(resource)
	if !ok {
		if aiDataPlatformHasEstablishedTrackedIdentity(resource) {
			return fmt.Errorf("%s create-only fingerprint is missing for tracked resource; recreate the resource before changing defaultWorkspaceName", aiDataPlatformKind)
		}
		return nil
	}
	desired := aiDataPlatformDefaultWorkspaceNameFingerprint(resource.Spec.DefaultWorkspaceName)
	if desired != recorded {
		return fmt.Errorf("%s formal semantics require replacement when defaultWorkspaceName changes", aiDataPlatformKind)
	}
	return nil
}

func wrapAiDataPlatformCreateOnlyTracking(hooks *AiDataPlatformRuntimeHooks) {
	if hooks == nil {
		return
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AiDataPlatformServiceClient) AiDataPlatformServiceClient {
		return aiDataPlatformCreateOnlyTrackingClient{delegate: delegate}
	})
}

type aiDataPlatformCreateOnlyTrackingClient struct {
	delegate AiDataPlatformServiceClient
}

func (c aiDataPlatformCreateOnlyTrackingClient) CreateOrUpdate(
	ctx context.Context,
	resource *aidataplatformv1beta1.AiDataPlatform,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("AiDataPlatform generated runtime delegate is not configured")
	}
	recorded, hasRecorded := aiDataPlatformRecordedDefaultWorkspaceNameFingerprint(resource)
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	switch {
	case hasRecorded && currentAiDataPlatformID(resource) != "":
		setAiDataPlatformDefaultWorkspaceNameFingerprint(resource, recorded)
	case err == nil && response.IsSuccessful && currentAiDataPlatformID(resource) != "":
		recordAiDataPlatformDefaultWorkspaceNameFingerprint(resource)
	}
	return response, err
}

func (c aiDataPlatformCreateOnlyTrackingClient) Delete(
	ctx context.Context,
	resource *aidataplatformv1beta1.AiDataPlatform,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("AiDataPlatform generated runtime delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func wrapAiDataPlatformDeleteConfirmation(hooks *AiDataPlatformRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}

	getAiDataPlatform := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AiDataPlatformServiceClient) AiDataPlatformServiceClient {
		return aiDataPlatformDeleteConfirmationClient{
			delegate:          delegate,
			getAiDataPlatform: getAiDataPlatform,
		}
	})
}

type aiDataPlatformDeleteConfirmationClient struct {
	delegate          AiDataPlatformServiceClient
	getAiDataPlatform func(context.Context, aidataplatformsdk.GetAiDataPlatformRequest) (aidataplatformsdk.GetAiDataPlatformResponse, error)
}

func (c aiDataPlatformDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *aidataplatformv1beta1.AiDataPlatform,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("AiDataPlatform generated runtime delegate is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c aiDataPlatformDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *aidataplatformv1beta1.AiDataPlatform,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("AiDataPlatform generated runtime delegate is not configured")
	}
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c aiDataPlatformDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *aidataplatformv1beta1.AiDataPlatform,
) error {
	if c.getAiDataPlatform == nil || resource == nil {
		return nil
	}
	currentID := currentAiDataPlatformID(resource)
	if currentID == "" {
		return nil
	}
	_, err := c.getAiDataPlatform(ctx, aidataplatformsdk.GetAiDataPlatformRequest{
		AiDataPlatformId: common.String(currentID),
	})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return aiDataPlatformAmbiguousNotFoundError{
		message:      "AiDataPlatform delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted",
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func handleAiDataPlatformDeleteError(resource *aidataplatformv1beta1.AiDataPlatform, err error) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return aiDataPlatformAmbiguousNotFoundError{
		message:      "AiDataPlatform delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func aiDataPlatformConservativeNotFoundError(message string, err error) error {
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return aiDataPlatformAmbiguousNotFoundError{
		message:      message,
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func aiDataPlatformProjectionFromResponse(response any) (aiDataPlatformStatusProjection, bool) {
	response = dereferenceAiDataPlatformRuntimeBody(response)
	switch current := response.(type) {
	case aiDataPlatformProjectedResponse:
		return current.AiDataPlatform, true
	case aiDataPlatformStatusProjection:
		return current, true
	case aidataplatformsdk.CreateAiDataPlatformResponse:
		return aiDataPlatformProjectionFromSDK(current.AiDataPlatform), true
	case aidataplatformsdk.GetAiDataPlatformResponse:
		return aiDataPlatformProjectionFromSDK(current.AiDataPlatform), true
	case aidataplatformsdk.AiDataPlatform:
		return aiDataPlatformProjectionFromSDK(current), true
	case aidataplatformsdk.AiDataPlatformSummary:
		return aiDataPlatformProjectionFromSummary(current), true
	default:
		return aiDataPlatformStatusProjection{}, false
	}
}

func dereferenceAiDataPlatformRuntimeBody(response any) any {
	value := reflect.ValueOf(response)
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return nil
	}
	return value.Interface()
}

func aiDataPlatformProjectionFromSDK(current aidataplatformsdk.AiDataPlatform) aiDataPlatformStatusProjection {
	return aiDataPlatformStatusProjection{
		Id:                 stringValue(current.Id),
		DisplayName:        stringValue(current.DisplayName),
		CompartmentId:      stringValue(current.CompartmentId),
		AiDataPlatformType: stringValue(current.AiDataPlatformType),
		TimeCreated:        sdkTimeString(current.TimeCreated),
		LifecycleState:     string(current.LifecycleState),
		FreeformTags:       maps.Clone(current.FreeformTags),
		DefinedTags:        statusTagsFromSDK(current.DefinedTags),
		CreatedBy:          stringValue(current.CreatedBy),
		TimeUpdated:        sdkTimeString(current.TimeUpdated),
		AliasKey:           stringValue(current.AliasKey),
		WebSocketEndpoint:  stringValue(current.WebSocketEndpoint),
		LifecycleDetails:   stringValue(current.LifecycleDetails),
		SystemTags:         statusTagsFromSDK(current.SystemTags),
	}
}

func aiDataPlatformProjectionFromSummary(current aidataplatformsdk.AiDataPlatformSummary) aiDataPlatformStatusProjection {
	return aiDataPlatformStatusProjection{
		Id:                 stringValue(current.Id),
		DisplayName:        stringValue(current.DisplayName),
		CompartmentId:      stringValue(current.CompartmentId),
		AiDataPlatformType: stringValue(current.AiDataPlatformType),
		TimeCreated:        sdkTimeString(current.TimeCreated),
		LifecycleState:     string(current.LifecycleState),
		FreeformTags:       maps.Clone(current.FreeformTags),
		DefinedTags:        statusTagsFromSDK(current.DefinedTags),
		CreatedBy:          stringValue(current.CreatedBy),
		TimeUpdated:        sdkTimeString(current.TimeUpdated),
		AliasKey:           stringValue(current.AliasKey),
		LifecycleDetails:   stringValue(current.LifecycleDetails),
		SystemTags:         statusTagsFromSDK(current.SystemTags),
	}
}

func currentAiDataPlatformID(resource *aidataplatformv1beta1.AiDataPlatform) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func aiDataPlatformHasEstablishedTrackedIdentity(resource *aidataplatformv1beta1.AiDataPlatform) bool {
	return currentAiDataPlatformID(resource) != "" &&
		resource != nil &&
		resource.Status.OsokStatus.CreatedAt != nil
}

func recordAiDataPlatformDefaultWorkspaceNameFingerprint(resource *aidataplatformv1beta1.AiDataPlatform) {
	if resource == nil {
		return
	}
	setAiDataPlatformDefaultWorkspaceNameFingerprint(
		resource,
		aiDataPlatformDefaultWorkspaceNameFingerprint(resource.Spec.DefaultWorkspaceName),
	)
}

func setAiDataPlatformDefaultWorkspaceNameFingerprint(
	resource *aidataplatformv1beta1.AiDataPlatform,
	fingerprint string,
) {
	if resource == nil {
		return
	}
	base := stripAiDataPlatformDefaultWorkspaceNameFingerprint(resource.Status.OsokStatus.Message)
	marker := aiDataPlatformDefaultWorkspaceNameFingerprintKey + fingerprint
	if base == "" {
		resource.Status.OsokStatus.Message = marker
		return
	}
	resource.Status.OsokStatus.Message = base + "; " + marker
}

func aiDataPlatformRecordedDefaultWorkspaceNameFingerprint(
	resource *aidataplatformv1beta1.AiDataPlatform,
) (string, bool) {
	if resource == nil {
		return "", false
	}
	raw := resource.Status.OsokStatus.Message
	index := strings.LastIndex(raw, aiDataPlatformDefaultWorkspaceNameFingerprintKey)
	if index < 0 {
		return "", false
	}
	start := index + len(aiDataPlatformDefaultWorkspaceNameFingerprintKey)
	end := start
	for end < len(raw) && isAiDataPlatformHexDigit(raw[end]) {
		end++
	}
	fingerprint := raw[start:end]
	if len(fingerprint) != sha256.Size*2 {
		return "", false
	}
	if _, err := hex.DecodeString(fingerprint); err != nil {
		return "", false
	}
	return fingerprint, true
}

func stripAiDataPlatformDefaultWorkspaceNameFingerprint(raw string) string {
	raw = strings.TrimSpace(raw)
	index := strings.LastIndex(raw, aiDataPlatformDefaultWorkspaceNameFingerprintKey)
	if index < 0 {
		return raw
	}
	prefix := strings.TrimSpace(strings.TrimRight(raw[:index], "; "))
	start := index + len(aiDataPlatformDefaultWorkspaceNameFingerprintKey)
	end := start
	for end < len(raw) && isAiDataPlatformHexDigit(raw[end]) {
		end++
	}
	suffix := strings.TrimSpace(strings.TrimLeft(raw[end:], "; "))
	switch {
	case prefix == "":
		return suffix
	case suffix == "":
		return prefix
	default:
		return prefix + "; " + suffix
	}
}

func aiDataPlatformDefaultWorkspaceNameFingerprint(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

func isAiDataPlatformHexDigit(value byte) bool {
	return (value >= '0' && value <= '9') ||
		(value >= 'a' && value <= 'f') ||
		(value >= 'A' && value <= 'F')
}

func optionalAiDataPlatformString(value string) *string {
	if value = strings.TrimSpace(value); value != "" {
		return common.String(value)
	}
	return nil
}

func desiredAiDataPlatformStringUpdate(desired string, current string) (*string, bool) {
	desired = strings.TrimSpace(desired)
	if desired == "" || desired == strings.TrimSpace(current) {
		return nil, false
	}
	return common.String(desired), true
}

func desiredAiDataPlatformFreeformTagsUpdate(
	desired map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if desired == nil || maps.Equal(desired, current) {
		return nil, false
	}
	return maps.Clone(desired), true
}

func desiredAiDataPlatformDefinedTagsUpdate(
	desired map[string]shared.MapValue,
	current map[string]shared.MapValue,
) (map[string]map[string]interface{}, bool) {
	if desired == nil || reflect.DeepEqual(desired, current) {
		return nil, false
	}
	return *util.ConvertToOciDefinedTags(&desired), true
}

func statusTagsFromSDK(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if tags == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(tags))
	for namespace, values := range tags {
		if values == nil {
			converted[namespace] = nil
			continue
		}
		convertedValues := make(shared.MapValue, len(values))
		for key, value := range values {
			convertedValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = convertedValues
	}
	return converted
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02T15:04:05.000Z07:00")
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
