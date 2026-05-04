/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package privilegedapicontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"unicode"

	apiaccesscontrolsdk "github.com/oracle/oci-go-sdk/v65/apiaccesscontrol"
	"github.com/oracle/oci-go-sdk/v65/common"
	apiaccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/apiaccesscontrol/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

var privilegedApiControlWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(apiaccesscontrolsdk.OperationStatusAccepted),
		string(apiaccesscontrolsdk.OperationStatusInProgress),
		string(apiaccesscontrolsdk.OperationStatusWaiting),
		string(apiaccesscontrolsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(apiaccesscontrolsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(apiaccesscontrolsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(apiaccesscontrolsdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(apiaccesscontrolsdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(apiaccesscontrolsdk.OperationTypeCreatePrivilegedApiControl)},
	UpdateActionTokens:    []string{string(apiaccesscontrolsdk.OperationTypeUpdatePrivilegedApiControl)},
	DeleteActionTokens:    []string{string(apiaccesscontrolsdk.OperationTypeDeletePrivilegedApiControl)},
}

type privilegedApiControlOCIClient interface {
	CreatePrivilegedApiControl(context.Context, apiaccesscontrolsdk.CreatePrivilegedApiControlRequest) (apiaccesscontrolsdk.CreatePrivilegedApiControlResponse, error)
	GetPrivilegedApiControl(context.Context, apiaccesscontrolsdk.GetPrivilegedApiControlRequest) (apiaccesscontrolsdk.GetPrivilegedApiControlResponse, error)
	ListPrivilegedApiControls(context.Context, apiaccesscontrolsdk.ListPrivilegedApiControlsRequest) (apiaccesscontrolsdk.ListPrivilegedApiControlsResponse, error)
	UpdatePrivilegedApiControl(context.Context, apiaccesscontrolsdk.UpdatePrivilegedApiControlRequest) (apiaccesscontrolsdk.UpdatePrivilegedApiControlResponse, error)
	DeletePrivilegedApiControl(context.Context, apiaccesscontrolsdk.DeletePrivilegedApiControlRequest) (apiaccesscontrolsdk.DeletePrivilegedApiControlResponse, error)
}

type privilegedApiControlWorkRequestClient interface {
	GetWorkRequest(context.Context, apiaccesscontrolsdk.GetWorkRequestRequest) (apiaccesscontrolsdk.GetWorkRequestResponse, error)
}

type privilegedApiControlRuntimeClient interface {
	privilegedApiControlOCIClient
	privilegedApiControlWorkRequestClient
}

func init() {
	registerPrivilegedApiControlRuntimeHooksMutator(func(manager *PrivilegedApiControlServiceManager, hooks *PrivilegedApiControlRuntimeHooks) {
		workRequestClient, initErr := newPrivilegedApiControlWorkRequestClient(manager)
		applyPrivilegedApiControlRuntimeHooks(hooks, workRequestClient, initErr)
	})
}

func newPrivilegedApiControlWorkRequestClient(
	manager *PrivilegedApiControlServiceManager,
) (privilegedApiControlWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("PrivilegedApiControl service manager is nil")
	}
	client, err := apiaccesscontrolsdk.NewPrivilegedApiWorkRequestClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyPrivilegedApiControlRuntimeHooks(
	hooks *PrivilegedApiControlRuntimeHooks,
	workRequestClient privilegedApiControlWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedPrivilegedApiControlRuntimeSemantics()
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *apiaccesscontrolv1beta1.PrivilegedApiControl,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildPrivilegedApiControlUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardPrivilegedApiControlExistingBeforeCreate
	hooks.Create.Fields = privilegedApiControlCreateFields()
	hooks.Get.Fields = privilegedApiControlGetFields()
	hooks.List.Fields = privilegedApiControlListFields()
	hooks.Update.Fields = privilegedApiControlUpdateFields()
	hooks.Delete.Fields = privilegedApiControlDeleteFields()
	hooks.Async.Adapter = privilegedApiControlWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getPrivilegedApiControlWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolvePrivilegedApiControlGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolvePrivilegedApiControlGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverPrivilegedApiControlIDFromGeneratedWorkRequest
	hooks.Async.Message = privilegedApiControlGeneratedWorkRequestMessage
}

func newPrivilegedApiControlServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client privilegedApiControlRuntimeClient,
) PrivilegedApiControlServiceClient {
	hooks := newPrivilegedApiControlRuntimeHooksWithOCIClient(client)
	applyPrivilegedApiControlRuntimeHooks(&hooks, client, nil)
	return defaultPrivilegedApiControlServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*apiaccesscontrolv1beta1.PrivilegedApiControl](
			newPrivilegedApiControlRuntimeConfig(log, hooks),
		),
	}
}

func newPrivilegedApiControlRuntimeConfig(
	log loggerutil.OSOKLogger,
	hooks PrivilegedApiControlRuntimeHooks,
) generatedruntime.Config[*apiaccesscontrolv1beta1.PrivilegedApiControl] {
	return buildPrivilegedApiControlGeneratedRuntimeConfig(
		&PrivilegedApiControlServiceManager{Log: log},
		hooks,
	)
}

func newPrivilegedApiControlRuntimeHooksWithOCIClient(
	client privilegedApiControlRuntimeClient,
) PrivilegedApiControlRuntimeHooks {
	return PrivilegedApiControlRuntimeHooks{
		Semantics: reviewedPrivilegedApiControlRuntimeSemantics(),
		Create: runtimeOperationHooks[apiaccesscontrolsdk.CreatePrivilegedApiControlRequest, apiaccesscontrolsdk.CreatePrivilegedApiControlResponse]{
			Fields: privilegedApiControlCreateFields(),
			Call: func(ctx context.Context, request apiaccesscontrolsdk.CreatePrivilegedApiControlRequest) (apiaccesscontrolsdk.CreatePrivilegedApiControlResponse, error) {
				return client.CreatePrivilegedApiControl(ctx, request)
			},
		},
		Get: runtimeOperationHooks[apiaccesscontrolsdk.GetPrivilegedApiControlRequest, apiaccesscontrolsdk.GetPrivilegedApiControlResponse]{
			Fields: privilegedApiControlGetFields(),
			Call: func(ctx context.Context, request apiaccesscontrolsdk.GetPrivilegedApiControlRequest) (apiaccesscontrolsdk.GetPrivilegedApiControlResponse, error) {
				return client.GetPrivilegedApiControl(ctx, request)
			},
		},
		List: runtimeOperationHooks[apiaccesscontrolsdk.ListPrivilegedApiControlsRequest, apiaccesscontrolsdk.ListPrivilegedApiControlsResponse]{
			Fields: privilegedApiControlListFields(),
			Call: func(ctx context.Context, request apiaccesscontrolsdk.ListPrivilegedApiControlsRequest) (apiaccesscontrolsdk.ListPrivilegedApiControlsResponse, error) {
				return client.ListPrivilegedApiControls(ctx, request)
			},
		},
		Update: runtimeOperationHooks[apiaccesscontrolsdk.UpdatePrivilegedApiControlRequest, apiaccesscontrolsdk.UpdatePrivilegedApiControlResponse]{
			Fields: privilegedApiControlUpdateFields(),
			Call: func(ctx context.Context, request apiaccesscontrolsdk.UpdatePrivilegedApiControlRequest) (apiaccesscontrolsdk.UpdatePrivilegedApiControlResponse, error) {
				return client.UpdatePrivilegedApiControl(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[apiaccesscontrolsdk.DeletePrivilegedApiControlRequest, apiaccesscontrolsdk.DeletePrivilegedApiControlResponse]{
			Fields: privilegedApiControlDeleteFields(),
			Call: func(ctx context.Context, request apiaccesscontrolsdk.DeletePrivilegedApiControlRequest) (apiaccesscontrolsdk.DeletePrivilegedApiControlResponse, error) {
				return client.DeletePrivilegedApiControl(ctx, request)
			},
		},
	}
}

func reviewedPrivilegedApiControlRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newPrivilegedApiControlRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName", "resourceType", "id"},
	}
	semantics.Hooks = generatedruntime.HookSet{
		Create: []generatedruntime.Hook{
			{Helper: "tfresource.CreateResource"},
			{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "privilegedapicontrol", Action: "CREATED"},
		},
		Update: []generatedruntime.Hook{
			{Helper: "tfresource.UpdateResource"},
			{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "privilegedapicontrol", Action: "UPDATED"},
		},
		Delete: []generatedruntime.Hook{
			{Helper: "tfresource.DeleteResource"},
			{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "privilegedapicontrol", Action: "DELETED"},
		},
	}
	semantics.CreateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetPrivilegedApiControl",
		Hooks:    append([]generatedruntime.Hook(nil), semantics.Hooks.Create...),
	}
	semantics.UpdateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetPrivilegedApiControl",
		Hooks:    append([]generatedruntime.Hook(nil), semantics.Hooks.Update...),
	}
	semantics.DeleteFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetPrivilegedApiControl/ListPrivilegedApiControls confirm-delete",
		Hooks:    append([]generatedruntime.Hook(nil), semantics.Hooks.Delete...),
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func privilegedApiControlCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreatePrivilegedApiControlDetails", RequestName: "CreatePrivilegedApiControlDetails", Contribution: "body"},
	}
}

func privilegedApiControlGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "PrivilegedApiControlId", RequestName: "privilegedApiControlId", Contribution: "path", PreferResourceID: true},
	}
}

func privilegedApiControlListFields() []generatedruntime.RequestField {
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
		{
			FieldName:    "ResourceType",
			RequestName:  "resourceType",
			Contribution: "query",
			LookupPaths:  []string{"status.resourceType", "spec.resourceType", "resourceType"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
	}
}

func privilegedApiControlUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "PrivilegedApiControlId", RequestName: "privilegedApiControlId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdatePrivilegedApiControlDetails", RequestName: "UpdatePrivilegedApiControlDetails", Contribution: "body"},
	}
}

func privilegedApiControlDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "PrivilegedApiControlId", RequestName: "privilegedApiControlId", Contribution: "path", PreferResourceID: true},
	}
}

func guardPrivilegedApiControlExistingBeforeCreate(
	_ context.Context,
	resource *apiaccesscontrolv1beta1.PrivilegedApiControl,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("PrivilegedApiControl resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.DisplayName) == "" ||
		strings.TrimSpace(resource.Spec.ResourceType) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildPrivilegedApiControlUpdateBody(
	resource *apiaccesscontrolv1beta1.PrivilegedApiControl,
	currentResponse any,
) (apiaccesscontrolsdk.UpdatePrivilegedApiControlDetails, bool, error) {
	if resource == nil {
		return apiaccesscontrolsdk.UpdatePrivilegedApiControlDetails{}, false, fmt.Errorf("PrivilegedApiControl resource is nil")
	}

	current, err := privilegedApiControlFromResponse(currentResponse)
	if err != nil {
		return apiaccesscontrolsdk.UpdatePrivilegedApiControlDetails{}, false, err
	}

	details := apiaccesscontrolsdk.UpdatePrivilegedApiControlDetails{}
	updateNeeded := false

	if desired, ok := privilegedApiControlDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := privilegedApiControlDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := privilegedApiControlDesiredStringUpdate(resource.Spec.ResourceType, current.ResourceType); ok {
		details.ResourceType = desired
		updateNeeded = true
	}
	if desired, ok := privilegedApiControlDesiredStringUpdate(resource.Spec.NotificationTopicId, current.NotificationTopicId); ok {
		details.NotificationTopicId = desired
		updateNeeded = true
	}
	if desired, ok := privilegedApiControlDesiredStringSliceUpdate(resource.Spec.ApproverGroupIdList, current.ApproverGroupIdList); ok {
		details.ApproverGroupIdList = desired
		updateNeeded = true
	}
	if desired, ok := privilegedApiControlDesiredStringSliceUpdate(resource.Spec.Resources, current.Resources); ok {
		details.Resources = desired
		updateNeeded = true
	}
	if desired, ok := privilegedApiControlDesiredPrivilegedOperationListUpdate(resource.Spec.PrivilegedOperationList, current.PrivilegedOperationList); ok {
		details.PrivilegedOperationList = desired
		updateNeeded = true
	}
	if desired, ok := privilegedApiControlDesiredNumberOfApproversUpdate(resource.Spec.NumberOfApprovers, current.NumberOfApprovers); ok {
		details.NumberOfApprovers = desired
		updateNeeded = true
	}
	if desired, ok := privilegedApiControlDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := privilegedApiControlDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func privilegedApiControlFromResponse(currentResponse any) (apiaccesscontrolsdk.PrivilegedApiControl, error) {
	switch current := currentResponse.(type) {
	case apiaccesscontrolsdk.PrivilegedApiControl:
		return current, nil
	case *apiaccesscontrolsdk.PrivilegedApiControl:
		if current == nil {
			return apiaccesscontrolsdk.PrivilegedApiControl{}, fmt.Errorf("current PrivilegedApiControl response is nil")
		}
		return *current, nil
	case apiaccesscontrolsdk.PrivilegedApiControlSummary:
		return apiaccesscontrolsdk.PrivilegedApiControl{
			Id:                current.Id,
			DisplayName:       current.DisplayName,
			CompartmentId:     current.CompartmentId,
			TimeCreated:       current.TimeCreated,
			LifecycleState:    current.LifecycleState,
			FreeformTags:      current.FreeformTags,
			DefinedTags:       current.DefinedTags,
			ResourceType:      current.ResourceType,
			NumberOfApprovers: current.NumberOfApprovers,
			TimeUpdated:       current.TimeUpdated,
			TimeDeleted:       current.TimeDeleted,
			LifecycleDetails:  current.LifecycleDetails,
			SystemTags:        current.SystemTags,
		}, nil
	case *apiaccesscontrolsdk.PrivilegedApiControlSummary:
		if current == nil {
			return apiaccesscontrolsdk.PrivilegedApiControl{}, fmt.Errorf("current PrivilegedApiControl response is nil")
		}
		return privilegedApiControlFromResponse(*current)
	case apiaccesscontrolsdk.GetPrivilegedApiControlResponse:
		return current.PrivilegedApiControl, nil
	case *apiaccesscontrolsdk.GetPrivilegedApiControlResponse:
		if current == nil {
			return apiaccesscontrolsdk.PrivilegedApiControl{}, fmt.Errorf("current PrivilegedApiControl response is nil")
		}
		return current.PrivilegedApiControl, nil
	default:
		return apiaccesscontrolsdk.PrivilegedApiControl{}, fmt.Errorf("unexpected current PrivilegedApiControl response type %T", currentResponse)
	}
}

func privilegedApiControlDesiredStringUpdate(spec string, current *string) (*string, bool) {
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

func privilegedApiControlDesiredStringSliceUpdate(spec []string, current []string) ([]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if reflect.DeepEqual(spec, current) {
		return nil, false
	}
	return append([]string(nil), spec...), true
}

func privilegedApiControlDesiredPrivilegedOperationListUpdate(
	spec []apiaccesscontrolv1beta1.PrivilegedApiControlPrivilegedOperationList,
	current []apiaccesscontrolsdk.PrivilegedApiDetails,
) ([]apiaccesscontrolsdk.PrivilegedApiDetails, bool) {
	if spec == nil {
		return nil, false
	}

	desired := privilegedApiControlPrivilegedOperationListFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if reflect.DeepEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func privilegedApiControlPrivilegedOperationListFromSpec(
	spec []apiaccesscontrolv1beta1.PrivilegedApiControlPrivilegedOperationList,
) []apiaccesscontrolsdk.PrivilegedApiDetails {
	if spec == nil {
		return nil
	}

	desired := make([]apiaccesscontrolsdk.PrivilegedApiDetails, 0, len(spec))
	for _, item := range spec {
		next := apiaccesscontrolsdk.PrivilegedApiDetails{
			ApiName: common.String(item.ApiName),
		}
		if item.EntityType != "" {
			next.EntityType = common.String(item.EntityType)
		}
		if item.AttributeNames != nil {
			next.AttributeNames = append([]string(nil), item.AttributeNames...)
		}
		desired = append(desired, next)
	}
	return desired
}

func privilegedApiControlDesiredNumberOfApproversUpdate(spec int, current *int) (*int, bool) {
	if spec == 0 {
		return nil, false
	}
	if current != nil && *current == spec {
		return nil, false
	}
	return common.Int(spec), true
}

func privilegedApiControlDesiredFreeformTagsUpdate(
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

func privilegedApiControlDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := privilegedApiControlDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if privilegedApiControlJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func privilegedApiControlDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
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

func privilegedApiControlJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func getPrivilegedApiControlWorkRequest(
	ctx context.Context,
	client privilegedApiControlWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize PrivilegedApiControl OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("PrivilegedApiControl OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, apiaccesscontrolsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolvePrivilegedApiControlGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := privilegedApiControlWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolvePrivilegedApiControlGeneratedWorkRequestPhase(
	workRequest any,
) (shared.OSOKAsyncPhase, bool, error) {
	current, err := privilegedApiControlWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := privilegedApiControlWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverPrivilegedApiControlIDFromGeneratedWorkRequest(
	_ *apiaccesscontrolv1beta1.PrivilegedApiControl,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := privilegedApiControlWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := privilegedApiControlWorkRequestActionForPhase(phase)
	if id, ok := resolvePrivilegedApiControlIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolvePrivilegedApiControlIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("PrivilegedApiControl work request %s does not expose a privileged API control identifier", stringValue(current.Id))
}

func privilegedApiControlGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := privilegedApiControlWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("PrivilegedApiControl %s work request %s is %s", phase, stringValue(current.Id), current.Status)
}

func privilegedApiControlWorkRequestFromAny(workRequest any) (apiaccesscontrolsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case apiaccesscontrolsdk.WorkRequest:
		return current, nil
	case *apiaccesscontrolsdk.WorkRequest:
		if current == nil {
			return apiaccesscontrolsdk.WorkRequest{}, fmt.Errorf("PrivilegedApiControl work request is nil")
		}
		return *current, nil
	default:
		return apiaccesscontrolsdk.WorkRequest{}, fmt.Errorf("unexpected PrivilegedApiControl work request type %T", workRequest)
	}
}

func privilegedApiControlWorkRequestPhaseFromOperationType(
	operationType apiaccesscontrolsdk.OperationTypeEnum,
) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case apiaccesscontrolsdk.OperationTypeCreatePrivilegedApiControl:
		return shared.OSOKAsyncPhaseCreate, true
	case apiaccesscontrolsdk.OperationTypeUpdatePrivilegedApiControl:
		return shared.OSOKAsyncPhaseUpdate, true
	case apiaccesscontrolsdk.OperationTypeDeletePrivilegedApiControl:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func privilegedApiControlWorkRequestActionForPhase(
	phase shared.OSOKAsyncPhase,
) apiaccesscontrolsdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return apiaccesscontrolsdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return apiaccesscontrolsdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return apiaccesscontrolsdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolvePrivilegedApiControlIDFromResources(
	resources []apiaccesscontrolsdk.WorkRequestResource,
	action apiaccesscontrolsdk.ActionTypeEnum,
	preferPrivilegedApiControlOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferPrivilegedApiControlOnly && !isPrivilegedApiControlWorkRequestResource(resource) {
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

func isPrivilegedApiControlWorkRequestResource(resource apiaccesscontrolsdk.WorkRequestResource) bool {
	return normalizePrivilegedApiControlToken(stringValue(resource.EntityType)) == "privilegedapicontrol"
}

func normalizePrivilegedApiControlToken(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(value))
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
