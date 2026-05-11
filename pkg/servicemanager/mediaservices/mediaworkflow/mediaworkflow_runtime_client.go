/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package mediaworkflow

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	mediaservicessdk "github.com/oracle/oci-go-sdk/v65/mediaservices"
	mediaservicesv1beta1 "github.com/oracle/oci-service-operator/api/mediaservices/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type mediaWorkflowOCIClient interface {
	CreateMediaWorkflow(context.Context, mediaservicessdk.CreateMediaWorkflowRequest) (mediaservicessdk.CreateMediaWorkflowResponse, error)
	GetMediaWorkflow(context.Context, mediaservicessdk.GetMediaWorkflowRequest) (mediaservicessdk.GetMediaWorkflowResponse, error)
	ListMediaWorkflows(context.Context, mediaservicessdk.ListMediaWorkflowsRequest) (mediaservicessdk.ListMediaWorkflowsResponse, error)
	UpdateMediaWorkflow(context.Context, mediaservicessdk.UpdateMediaWorkflowRequest) (mediaservicessdk.UpdateMediaWorkflowResponse, error)
	DeleteMediaWorkflow(context.Context, mediaservicessdk.DeleteMediaWorkflowRequest) (mediaservicessdk.DeleteMediaWorkflowResponse, error)
}

func init() {
	registerMediaWorkflowRuntimeHooksMutator(func(_ *MediaWorkflowServiceManager, hooks *MediaWorkflowRuntimeHooks) {
		applyMediaWorkflowRuntimeHooks(hooks)
	})
}

func applyMediaWorkflowRuntimeHooks(hooks *MediaWorkflowRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedMediaWorkflowRuntimeSemantics()
	hooks.Identity.GuardExistingBeforeCreate = guardMediaWorkflowExistingBeforeCreate
	hooks.ParityHooks.NormalizeDesiredState = normalizeMediaWorkflowDesiredState
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateMediaWorkflowCreateOnlyDrift
	hooks.BuildCreateBody = func(
		ctx context.Context,
		resource *mediaservicesv1beta1.MediaWorkflow,
		namespace string,
	) (any, error) {
		return buildMediaWorkflowCreateDetails(ctx, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *mediaservicesv1beta1.MediaWorkflow,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildMediaWorkflowUpdateBody(resource, currentResponse)
	}
	hooks.Create.Fields = mediaWorkflowCreateFields()
	hooks.Get.Fields = mediaWorkflowGetFields()
	hooks.List.Fields = mediaWorkflowListFields()
	wrapMediaWorkflowListPages(hooks)
	hooks.Update.Fields = mediaWorkflowUpdateFields()
	hooks.Delete.Fields = mediaWorkflowDeleteFields()
}

func newMediaWorkflowServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client mediaWorkflowOCIClient,
) MediaWorkflowServiceClient {
	hooks := newMediaWorkflowRuntimeHooksWithOCIClient(client)
	applyMediaWorkflowRuntimeHooks(&hooks)
	delegate := defaultMediaWorkflowServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*mediaservicesv1beta1.MediaWorkflow](
			buildMediaWorkflowGeneratedRuntimeConfig(&MediaWorkflowServiceManager{Log: log}, hooks),
		),
	}
	return wrapMediaWorkflowGeneratedClient(hooks, delegate)
}

func newMediaWorkflowRuntimeHooksWithOCIClient(client mediaWorkflowOCIClient) MediaWorkflowRuntimeHooks {
	return MediaWorkflowRuntimeHooks{
		Create: runtimeOperationHooks[mediaservicessdk.CreateMediaWorkflowRequest, mediaservicessdk.CreateMediaWorkflowResponse]{
			Fields: mediaWorkflowCreateFields(),
			Call: func(ctx context.Context, request mediaservicessdk.CreateMediaWorkflowRequest) (mediaservicessdk.CreateMediaWorkflowResponse, error) {
				return client.CreateMediaWorkflow(ctx, request)
			},
		},
		Get: runtimeOperationHooks[mediaservicessdk.GetMediaWorkflowRequest, mediaservicessdk.GetMediaWorkflowResponse]{
			Fields: mediaWorkflowGetFields(),
			Call: func(ctx context.Context, request mediaservicessdk.GetMediaWorkflowRequest) (mediaservicessdk.GetMediaWorkflowResponse, error) {
				return client.GetMediaWorkflow(ctx, request)
			},
		},
		List: runtimeOperationHooks[mediaservicessdk.ListMediaWorkflowsRequest, mediaservicessdk.ListMediaWorkflowsResponse]{
			Fields: mediaWorkflowListFields(),
			Call: func(ctx context.Context, request mediaservicessdk.ListMediaWorkflowsRequest) (mediaservicessdk.ListMediaWorkflowsResponse, error) {
				return client.ListMediaWorkflows(ctx, request)
			},
		},
		Update: runtimeOperationHooks[mediaservicessdk.UpdateMediaWorkflowRequest, mediaservicessdk.UpdateMediaWorkflowResponse]{
			Fields: mediaWorkflowUpdateFields(),
			Call: func(ctx context.Context, request mediaservicessdk.UpdateMediaWorkflowRequest) (mediaservicessdk.UpdateMediaWorkflowResponse, error) {
				return client.UpdateMediaWorkflow(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[mediaservicessdk.DeleteMediaWorkflowRequest, mediaservicessdk.DeleteMediaWorkflowResponse]{
			Fields: mediaWorkflowDeleteFields(),
			Call: func(ctx context.Context, request mediaservicessdk.DeleteMediaWorkflowRequest) (mediaservicessdk.DeleteMediaWorkflowResponse, error) {
				return client.DeleteMediaWorkflow(ctx, request)
			},
		},
	}
}

func reviewedMediaWorkflowRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newMediaWorkflowRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName"},
	}
	semantics.Hooks = generatedruntime.HookSet{
		Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "MediaWorkflow", Action: "CreateMediaWorkflow"}},
		Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "MediaWorkflow", Action: "UpdateMediaWorkflow"}},
		Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "MediaWorkflow", Action: "DeleteMediaWorkflow"}},
	}
	semantics.CreateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "read-after-write",
		Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "MediaWorkflow", Action: "GetMediaWorkflow"}},
	}
	semantics.UpdateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "read-after-write",
		Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "MediaWorkflow", Action: "GetMediaWorkflow"}},
	}
	semantics.DeleteFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "confirm-delete",
		Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "MediaWorkflow", Action: "GetMediaWorkflow"}},
	}
	semantics.AuxiliaryOperations = nil
	semantics.Unsupported = nil
	return semantics
}

func mediaWorkflowCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateMediaWorkflowDetails", RequestName: "CreateMediaWorkflowDetails", Contribution: "body"},
	}
}

func mediaWorkflowGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MediaWorkflowId", RequestName: "mediaWorkflowId", Contribution: "path", PreferResourceID: true},
	}
}

func mediaWorkflowListFields() []generatedruntime.RequestField {
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
	}
}

func mediaWorkflowUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MediaWorkflowId", RequestName: "mediaWorkflowId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateMediaWorkflowDetails", RequestName: "UpdateMediaWorkflowDetails", Contribution: "body"},
	}
}

func mediaWorkflowDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "MediaWorkflowId", RequestName: "mediaWorkflowId", Contribution: "path", PreferResourceID: true},
	}
}

func wrapMediaWorkflowListPages(hooks *MediaWorkflowRuntimeHooks) {
	if hooks == nil || hooks.List.Call == nil {
		return
	}

	call := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request mediaservicessdk.ListMediaWorkflowsRequest) (mediaservicessdk.ListMediaWorkflowsResponse, error) {
		return listMediaWorkflowPages(ctx, call, request)
	}
}

func listMediaWorkflowPages(
	ctx context.Context,
	call func(context.Context, mediaservicessdk.ListMediaWorkflowsRequest) (mediaservicessdk.ListMediaWorkflowsResponse, error),
	request mediaservicessdk.ListMediaWorkflowsRequest,
) (mediaservicessdk.ListMediaWorkflowsResponse, error) {
	var combined mediaservicessdk.ListMediaWorkflowsResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return response, err
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.RawResponse = response.RawResponse
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
	}
}

func guardMediaWorkflowExistingBeforeCreate(
	_ context.Context,
	resource *mediaservicesv1beta1.MediaWorkflow,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("MediaWorkflow resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("MediaWorkflow spec.compartmentId is required")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildMediaWorkflowCreateDetails(
	ctx context.Context,
	resource *mediaservicesv1beta1.MediaWorkflow,
	namespace string,
) (mediaservicessdk.CreateMediaWorkflowDetails, error) {
	if resource == nil {
		return mediaservicessdk.CreateMediaWorkflowDetails{}, fmt.Errorf("MediaWorkflow resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return mediaservicessdk.CreateMediaWorkflowDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return mediaservicessdk.CreateMediaWorkflowDetails{}, fmt.Errorf("marshal resolved MediaWorkflow spec: %w", err)
	}

	var details mediaservicessdk.CreateMediaWorkflowDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return mediaservicessdk.CreateMediaWorkflowDetails{}, fmt.Errorf("decode MediaWorkflow create request body: %w", err)
	}
	for index := range details.Locks {
		details.Locks[index].TimeCreated = nil
	}
	return details, nil
}

func buildMediaWorkflowUpdateBody(
	resource *mediaservicesv1beta1.MediaWorkflow,
	currentResponse any,
) (mediaservicessdk.UpdateMediaWorkflowDetails, bool, error) {
	if resource == nil {
		return mediaservicessdk.UpdateMediaWorkflowDetails{}, false, fmt.Errorf("MediaWorkflow resource is nil")
	}

	current, err := mediaWorkflowRuntimeBody(currentResponse)
	if err != nil {
		return mediaservicessdk.UpdateMediaWorkflowDetails{}, false, err
	}

	spec := resource.Spec
	details := mediaservicessdk.UpdateMediaWorkflowDetails{}
	updateNeeded := false

	if desired, ok := mediaWorkflowDesiredStringUpdate(spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok, err := mediaWorkflowDesiredTasksUpdate(spec.Tasks, current.Tasks); err != nil {
		return mediaservicessdk.UpdateMediaWorkflowDetails{}, false, err
	} else if ok {
		details.Tasks = desired
		updateNeeded = true
	}
	if desired, ok := mediaWorkflowDesiredConfigurationIDsUpdate(spec.MediaWorkflowConfigurationIds, current.MediaWorkflowConfigurationIds); ok {
		details.MediaWorkflowConfigurationIds = desired
		updateNeeded = true
	}
	if desired, ok, err := mediaWorkflowDesiredParametersUpdate(spec.Parameters, current.Parameters); err != nil {
		return mediaservicessdk.UpdateMediaWorkflowDetails{}, false, err
	} else if ok {
		details.Parameters = desired
		updateNeeded = true
	}
	if desired, ok := mediaWorkflowDesiredFreeformTagsUpdate(spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok, err := mediaWorkflowDesiredDefinedTagsUpdate(spec.DefinedTags, current.DefinedTags); err != nil {
		return mediaservicessdk.UpdateMediaWorkflowDetails{}, false, err
	} else if ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func mediaWorkflowRuntimeBody(currentResponse any) (mediaservicessdk.MediaWorkflow, error) {
	switch current := currentResponse.(type) {
	case mediaservicessdk.MediaWorkflow:
		return current, nil
	case *mediaservicessdk.MediaWorkflow:
		if current == nil {
			return mediaservicessdk.MediaWorkflow{}, fmt.Errorf("current MediaWorkflow response is nil")
		}
		return *current, nil
	case mediaservicessdk.MediaWorkflowSummary:
		return mediaservicessdk.MediaWorkflow{
			Id:                            current.Id,
			DisplayName:                   current.DisplayName,
			CompartmentId:                 current.CompartmentId,
			TimeCreated:                   current.TimeCreated,
			TimeUpdated:                   current.TimeUpdated,
			LifecycleState:                current.LifecycleState,
			LifecyleDetails:               current.LifecycleDetails,
			Version:                       current.Version,
			FreeformTags:                  current.FreeformTags,
			DefinedTags:                   current.DefinedTags,
			SystemTags:                    current.SystemTags,
			Locks:                         current.Locks,
			MediaWorkflowConfigurationIds: nil,
			Parameters:                    nil,
			Tasks:                         nil,
		}, nil
	case *mediaservicessdk.MediaWorkflowSummary:
		if current == nil {
			return mediaservicessdk.MediaWorkflow{}, fmt.Errorf("current MediaWorkflow response is nil")
		}
		return mediaWorkflowRuntimeBody(*current)
	case mediaservicessdk.CreateMediaWorkflowResponse:
		return current.MediaWorkflow, nil
	case *mediaservicessdk.CreateMediaWorkflowResponse:
		if current == nil {
			return mediaservicessdk.MediaWorkflow{}, fmt.Errorf("current MediaWorkflow response is nil")
		}
		return current.MediaWorkflow, nil
	case mediaservicessdk.GetMediaWorkflowResponse:
		return current.MediaWorkflow, nil
	case *mediaservicessdk.GetMediaWorkflowResponse:
		if current == nil {
			return mediaservicessdk.MediaWorkflow{}, fmt.Errorf("current MediaWorkflow response is nil")
		}
		return current.MediaWorkflow, nil
	case mediaservicessdk.UpdateMediaWorkflowResponse:
		return current.MediaWorkflow, nil
	case *mediaservicessdk.UpdateMediaWorkflowResponse:
		if current == nil {
			return mediaservicessdk.MediaWorkflow{}, fmt.Errorf("current MediaWorkflow response is nil")
		}
		return current.MediaWorkflow, nil
	default:
		return mediaservicessdk.MediaWorkflow{}, fmt.Errorf("unexpected current MediaWorkflow response type %T", currentResponse)
	}
}

func normalizeMediaWorkflowDesiredState(resource *mediaservicesv1beta1.MediaWorkflow, currentResponse any) {
	if resource == nil || resource.Spec.Locks == nil {
		return
	}
	current, err := mediaWorkflowRuntimeBody(currentResponse)
	if err != nil {
		return
	}
	if mediaWorkflowLocksEqual(resource.Spec.Locks, current.Locks) {
		resource.Spec.Locks = nil
	}
}

func validateMediaWorkflowCreateOnlyDrift(resource *mediaservicesv1beta1.MediaWorkflow, currentResponse any) error {
	if resource == nil || resource.Spec.Locks == nil {
		return nil
	}
	current, err := mediaWorkflowRuntimeBody(currentResponse)
	if err != nil {
		return err
	}
	if mediaWorkflowLocksEqual(resource.Spec.Locks, current.Locks) {
		return nil
	}
	return fmt.Errorf("MediaWorkflow create-only drift detected for locks; replace the resource or restore the desired spec before update")
}

func mediaWorkflowDesiredStringUpdate(spec string, current *string) (*string, bool) {
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

func mediaWorkflowDesiredTasksUpdate(
	spec []mediaservicesv1beta1.MediaWorkflowTask,
	current []mediaservicessdk.MediaWorkflowTask,
) ([]mediaservicessdk.MediaWorkflowTask, bool, error) {
	if spec == nil {
		return nil, false, nil
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false, nil
	}

	desired, err := mediaWorkflowTasksFromSpec(spec)
	if err != nil {
		return nil, false, err
	}
	desiredJSON, err := mediaWorkflowCanonicalJSONString(desired)
	if err != nil {
		return nil, false, fmt.Errorf("normalize desired MediaWorkflow tasks: %w", err)
	}
	currentJSON, err := mediaWorkflowCanonicalJSONString(current)
	if err != nil {
		return nil, false, fmt.Errorf("normalize current MediaWorkflow tasks: %w", err)
	}
	if desiredJSON == currentJSON {
		return nil, false, nil
	}
	return desired, true, nil
}

func mediaWorkflowDesiredConfigurationIDsUpdate(spec []string, current []string) ([]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	desiredJSON, _ := mediaWorkflowCanonicalJSONString(spec)
	currentJSON, _ := mediaWorkflowCanonicalJSONString(current)
	if desiredJSON == currentJSON {
		return nil, false
	}
	return append([]string{}, spec...), true
}

func mediaWorkflowDesiredParametersUpdate(
	spec map[string]shared.JSONValue,
	current map[string]interface{},
) (map[string]interface{}, bool, error) {
	if spec == nil {
		return nil, false, nil
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false, nil
	}

	desired, err := mediaWorkflowParametersFromSpec(spec)
	if err != nil {
		return nil, false, err
	}
	desiredJSON, err := mediaWorkflowCanonicalJSONString(desired)
	if err != nil {
		return nil, false, fmt.Errorf("normalize desired MediaWorkflow parameters: %w", err)
	}
	currentJSON, err := mediaWorkflowCanonicalJSONString(current)
	if err != nil {
		return nil, false, fmt.Errorf("normalize current MediaWorkflow parameters: %w", err)
	}
	if desiredJSON == currentJSON {
		return nil, false, nil
	}
	return desired, true, nil
}

func mediaWorkflowDesiredFreeformTagsUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func mediaWorkflowDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool, error) {
	if spec == nil {
		return nil, false, nil
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false, nil
	}

	desired, err := mediaWorkflowDefinedTagsFromSpec(spec)
	if err != nil {
		return nil, false, err
	}
	desiredJSON, err := mediaWorkflowCanonicalJSONString(desired)
	if err != nil {
		return nil, false, fmt.Errorf("normalize desired MediaWorkflow definedTags: %w", err)
	}
	currentJSON, err := mediaWorkflowCanonicalJSONString(current)
	if err != nil {
		return nil, false, fmt.Errorf("normalize current MediaWorkflow definedTags: %w", err)
	}
	if desiredJSON == currentJSON {
		return nil, false, nil
	}
	return desired, true, nil
}

func mediaWorkflowTasksFromSpec(spec []mediaservicesv1beta1.MediaWorkflowTask) ([]mediaservicessdk.MediaWorkflowTask, error) {
	if spec == nil {
		return nil, nil
	}
	if len(spec) == 0 {
		return []mediaservicessdk.MediaWorkflowTask{}, nil
	}

	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal MediaWorkflow tasks: %w", err)
	}
	var tasks []mediaservicessdk.MediaWorkflowTask
	if err := json.Unmarshal(payload, &tasks); err != nil {
		return nil, fmt.Errorf("decode MediaWorkflow tasks: %w", err)
	}
	return tasks, nil
}

func mediaWorkflowParametersFromSpec(spec map[string]shared.JSONValue) (map[string]interface{}, error) {
	if spec == nil {
		return nil, nil
	}
	if len(spec) == 0 {
		return map[string]interface{}{}, nil
	}

	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal MediaWorkflow parameters: %w", err)
	}
	var parameters map[string]interface{}
	if err := json.Unmarshal(payload, &parameters); err != nil {
		return nil, fmt.Errorf("decode MediaWorkflow parameters: %w", err)
	}
	return parameters, nil
}

func mediaWorkflowDefinedTagsFromSpec(spec map[string]shared.MapValue) (map[string]map[string]interface{}, error) {
	if spec == nil {
		return nil, nil
	}
	if len(spec) == 0 {
		return map[string]map[string]interface{}{}, nil
	}

	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal MediaWorkflow definedTags: %w", err)
	}
	var tags map[string]map[string]interface{}
	if err := json.Unmarshal(payload, &tags); err != nil {
		return nil, fmt.Errorf("decode MediaWorkflow definedTags: %w", err)
	}
	return tags, nil
}

func mediaWorkflowCanonicalJSONString(value any) (string, error) {
	if value == nil {
		return "", nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	switch string(payload) {
	case "", "null", "{}", "[]":
		return "", nil
	}

	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return "", err
	}
	normalized, err := json.Marshal(decoded)
	if err != nil {
		return "", err
	}
	switch string(normalized) {
	case "", "null", "{}", "[]":
		return "", nil
	default:
		return string(normalized), nil
	}
}

func mediaWorkflowLocksEqual(spec []mediaservicesv1beta1.MediaWorkflowLock, current []mediaservicessdk.ResourceLock) bool {
	if len(spec) != len(current) {
		return false
	}
	for index, lock := range spec {
		if lock.Type != string(current[index].Type) ||
			lock.CompartmentId != mediaWorkflowStringValue(current[index].CompartmentId) ||
			lock.RelatedResourceId != mediaWorkflowStringValue(current[index].RelatedResourceId) ||
			lock.Message != mediaWorkflowStringValue(current[index].Message) {
			return false
		}
	}
	return true
}

func mediaWorkflowStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
