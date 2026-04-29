/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package unifiedagentconfiguration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	loggingsdk "github.com/oracle/oci-go-sdk/v65/logging"
	loggingv1beta1 "github.com/oracle/oci-service-operator/api/logging/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
)

var unifiedAgentConfigurationWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(loggingsdk.OperationStatusAccepted),
		string(loggingsdk.OperationStatusInProgress),
		string(loggingsdk.OperationStatusCancelling),
	},
	SucceededStatusTokens: []string{string(loggingsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(loggingsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(loggingsdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(loggingsdk.ActionTypesCreated)},
	UpdateActionTokens:    []string{string(loggingsdk.ActionTypesUpdated)},
	DeleteActionTokens:    []string{string(loggingsdk.ActionTypesDeleted)},
}

type unifiedAgentConfigurationOCIClient interface {
	CreateUnifiedAgentConfiguration(context.Context, loggingsdk.CreateUnifiedAgentConfigurationRequest) (loggingsdk.CreateUnifiedAgentConfigurationResponse, error)
	GetUnifiedAgentConfiguration(context.Context, loggingsdk.GetUnifiedAgentConfigurationRequest) (loggingsdk.GetUnifiedAgentConfigurationResponse, error)
	ListUnifiedAgentConfigurations(context.Context, loggingsdk.ListUnifiedAgentConfigurationsRequest) (loggingsdk.ListUnifiedAgentConfigurationsResponse, error)
	UpdateUnifiedAgentConfiguration(context.Context, loggingsdk.UpdateUnifiedAgentConfigurationRequest) (loggingsdk.UpdateUnifiedAgentConfigurationResponse, error)
	DeleteUnifiedAgentConfiguration(context.Context, loggingsdk.DeleteUnifiedAgentConfigurationRequest) (loggingsdk.DeleteUnifiedAgentConfigurationResponse, error)
	GetWorkRequest(context.Context, loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error)
}

func init() {
	registerUnifiedAgentConfigurationRuntimeHooksMutator(func(manager *UnifiedAgentConfigurationServiceManager, hooks *UnifiedAgentConfigurationRuntimeHooks) {
		client, initErr := newUnifiedAgentConfigurationSDKClient(manager)
		applyUnifiedAgentConfigurationRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newUnifiedAgentConfigurationSDKClient(manager *UnifiedAgentConfigurationServiceManager) (unifiedAgentConfigurationOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("UnifiedAgentConfiguration service manager is nil")
	}
	client, err := loggingsdk.NewLoggingManagementClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyUnifiedAgentConfigurationRuntimeHooks(
	_ *UnifiedAgentConfigurationServiceManager,
	hooks *UnifiedAgentConfigurationRuntimeHooks,
	client unifiedAgentConfigurationOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newUnifiedAgentConfigurationRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *loggingv1beta1.UnifiedAgentConfiguration, _ string) (any, error) {
		return buildUnifiedAgentConfigurationCreateDetails(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *loggingv1beta1.UnifiedAgentConfiguration,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildUnifiedAgentConfigurationUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardUnifiedAgentConfigurationExistingBeforeCreate
	hooks.List.Fields = unifiedAgentConfigurationListFields()
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateUnifiedAgentConfigurationCreateOnlyDriftForResponse
	hooks.Async.Adapter = unifiedAgentConfigurationWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getUnifiedAgentConfigurationWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveUnifiedAgentConfigurationGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveUnifiedAgentConfigurationGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverUnifiedAgentConfigurationIDFromGeneratedWorkRequest
	hooks.Async.Message = unifiedAgentConfigurationGeneratedWorkRequestMessage
}

func newUnifiedAgentConfigurationRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "logging",
		FormalSlug:    "unifiedagentconfiguration",
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
			ProvisioningStates: []string{string(loggingsdk.LogLifecycleStateCreating)},
			UpdatingStates:     []string{string(loggingsdk.LogLifecycleStateUpdating)},
			ActiveStates:       []string{string(loggingsdk.LogLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(loggingsdk.LogLifecycleStateDeleting)},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"displayName"},
			// serviceConfiguration is guarded by a normalized resource-local drift check;
			// covering it here avoids false generic drift from SDK polymorphic null fields.
			Mutable:       []string{"displayName", "serviceConfiguration"},
			ForceNew:      []string{"compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "configuration", Action: string(loggingsdk.ActionTypesCreated)},
			},
			Update: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "configuration", Action: string(loggingsdk.ActionTypesUpdated)},
			},
			Delete: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "configuration", Action: string(loggingsdk.ActionTypesDeleted)},
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetUnifiedAgentConfiguration",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "configuration", Action: string(loggingsdk.ActionTypesCreated)},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetUnifiedAgentConfiguration",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "configuration", Action: string(loggingsdk.ActionTypesUpdated)},
			},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "configuration", Action: string(loggingsdk.ActionTypesDeleted)},
			},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func unifiedAgentConfigurationListFields() []generatedruntime.RequestField {
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
			FieldName:    "LifecycleState",
			RequestName:  "lifecycleState",
			Contribution: "query",
			LookupPaths:  []string{"status.lifecycleState", "lifecycleState"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
	}
}

func guardUnifiedAgentConfigurationExistingBeforeCreate(
	_ context.Context,
	resource *loggingv1beta1.UnifiedAgentConfiguration,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("UnifiedAgentConfiguration resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildUnifiedAgentConfigurationCreateDetails(
	resource *loggingv1beta1.UnifiedAgentConfiguration,
) (loggingsdk.CreateUnifiedAgentConfigurationDetails, error) {
	if resource == nil {
		return loggingsdk.CreateUnifiedAgentConfigurationDetails{}, fmt.Errorf("UnifiedAgentConfiguration resource is nil")
	}

	serviceConfiguration, err := decodeUnifiedAgentConfigurationServiceConfiguration(resource.Spec.ServiceConfiguration)
	if err != nil {
		return loggingsdk.CreateUnifiedAgentConfigurationDetails{}, err
	}

	details := loggingsdk.CreateUnifiedAgentConfigurationDetails{
		IsEnabled:            common.Bool(resource.Spec.IsEnabled),
		ServiceConfiguration: serviceConfiguration,
		CompartmentId:        common.String(resource.Spec.CompartmentId),
	}
	if strings.TrimSpace(resource.Spec.DisplayName) != "" {
		details.DisplayName = common.String(resource.Spec.DisplayName)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = cloneUnifiedAgentConfigurationStringMap(resource.Spec.FreeformTags)
	}
	if strings.TrimSpace(resource.Spec.Description) != "" {
		details.Description = common.String(resource.Spec.Description)
	}
	if len(resource.Spec.GroupAssociation.GroupList) != 0 {
		details.GroupAssociation = &loggingsdk.GroupAssociationDetails{
			GroupList: append([]string(nil), resource.Spec.GroupAssociation.GroupList...),
		}
	}
	return details, nil
}

func buildUnifiedAgentConfigurationUpdateBody(
	resource *loggingv1beta1.UnifiedAgentConfiguration,
	currentResponse any,
) (loggingsdk.UpdateUnifiedAgentConfigurationDetails, bool, error) {
	if resource == nil {
		return loggingsdk.UpdateUnifiedAgentConfigurationDetails{}, false, fmt.Errorf("UnifiedAgentConfiguration resource is nil")
	}

	current, ok := unifiedAgentConfigurationFromResponse(currentResponse)
	if !ok {
		return loggingsdk.UpdateUnifiedAgentConfigurationDetails{}, false, fmt.Errorf("current UnifiedAgentConfiguration response does not expose a UnifiedAgentConfiguration body")
	}

	desiredDisplayName := strings.TrimSpace(resource.Spec.DisplayName)
	if desiredDisplayName == "" || stringPtrEqual(current.DisplayName, desiredDisplayName) {
		return loggingsdk.UpdateUnifiedAgentConfigurationDetails{}, false, nil
	}

	serviceConfiguration := current.ServiceConfiguration
	if serviceConfiguration == nil {
		var err error
		serviceConfiguration, err = decodeUnifiedAgentConfigurationServiceConfiguration(resource.Spec.ServiceConfiguration)
		if err != nil {
			return loggingsdk.UpdateUnifiedAgentConfigurationDetails{}, false, err
		}
	}

	isEnabled := current.IsEnabled
	if isEnabled == nil {
		isEnabled = common.Bool(resource.Spec.IsEnabled)
	}

	updateDetails := loggingsdk.UpdateUnifiedAgentConfigurationDetails{
		DisplayName:          common.String(resource.Spec.DisplayName),
		IsEnabled:            isEnabled,
		ServiceConfiguration: serviceConfiguration,
		DefinedTags:          cloneUnifiedAgentConfigurationDefinedTags(current.DefinedTags),
		FreeformTags:         cloneUnifiedAgentConfigurationStringMap(current.FreeformTags),
		Description:          current.Description,
		GroupAssociation:     current.GroupAssociation,
	}
	return updateDetails, true, nil
}

func validateUnifiedAgentConfigurationCreateOnlyDriftForResponse(
	resource *loggingv1beta1.UnifiedAgentConfiguration,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("UnifiedAgentConfiguration resource is nil")
	}
	current, ok := unifiedAgentConfigurationFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current UnifiedAgentConfiguration response does not expose a UnifiedAgentConfiguration body")
	}
	if current.IsEnabled != nil && *current.IsEnabled != resource.Spec.IsEnabled {
		return fmt.Errorf("UnifiedAgentConfiguration formal semantics reject unsupported update drift for isEnabled")
	}
	if current.ServiceConfiguration != nil {
		equal, err := unifiedAgentConfigurationServiceConfigurationEqual(resource.Spec.ServiceConfiguration, current.ServiceConfiguration)
		if err != nil {
			return err
		}
		if !equal {
			return fmt.Errorf("UnifiedAgentConfiguration formal semantics reject unsupported update drift for serviceConfiguration")
		}
	}
	return nil
}

func unifiedAgentConfigurationServiceConfigurationEqual(
	desired loggingv1beta1.UnifiedAgentConfigurationServiceConfiguration,
	current loggingsdk.UnifiedAgentServiceConfigurationDetails,
) (bool, error) {
	desiredNormalized, err := normalizedUnifiedAgentConfigurationJSONValue(desired)
	if err != nil {
		return false, fmt.Errorf("normalize desired UnifiedAgentConfiguration serviceConfiguration: %w", err)
	}
	if desiredObject, ok := desiredNormalized.(map[string]any); ok {
		if _, ok := desiredObject["configurationType"]; !ok {
			desiredObject["configurationType"] = string(loggingsdk.UnifiedAgentServiceConfigurationTypesLogging)
		}
	}

	currentNormalized, err := normalizedUnifiedAgentConfigurationReadbackJSONValue(current)
	if err != nil {
		return false, fmt.Errorf("normalize current UnifiedAgentConfiguration serviceConfiguration: %w", err)
	}

	desiredPayload, err := json.Marshal(desiredNormalized)
	if err != nil {
		return false, fmt.Errorf("marshal desired UnifiedAgentConfiguration serviceConfiguration: %w", err)
	}
	currentPayload, err := json.Marshal(currentNormalized)
	if err != nil {
		return false, fmt.Errorf("marshal current UnifiedAgentConfiguration serviceConfiguration: %w", err)
	}
	return string(desiredPayload) == string(currentPayload), nil
}

func decodeUnifiedAgentConfigurationServiceConfiguration(
	spec loggingv1beta1.UnifiedAgentConfigurationServiceConfiguration,
) (loggingsdk.UnifiedAgentServiceConfigurationDetails, error) {
	normalized, err := normalizedUnifiedAgentConfigurationJSONValue(spec)
	if err != nil {
		return nil, fmt.Errorf("normalize UnifiedAgentConfiguration serviceConfiguration: %w", err)
	}
	if normalized == nil {
		return nil, fmt.Errorf("UnifiedAgentConfiguration serviceConfiguration is empty")
	}
	if object, ok := normalized.(map[string]any); ok {
		if _, ok := object["configurationType"]; !ok {
			object["configurationType"] = string(loggingsdk.UnifiedAgentServiceConfigurationTypesLogging)
		}
	}

	payload, err := json.Marshal(map[string]any{
		"serviceConfiguration": normalized,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal UnifiedAgentConfiguration serviceConfiguration: %w", err)
	}

	var details loggingsdk.CreateUnifiedAgentConfigurationDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return nil, fmt.Errorf("decode UnifiedAgentConfiguration serviceConfiguration: %w", err)
	}
	if details.ServiceConfiguration == nil {
		return nil, fmt.Errorf("UnifiedAgentConfiguration serviceConfiguration did not resolve to an OCI SDK model")
	}
	return details.ServiceConfiguration, nil
}

func normalizedUnifiedAgentConfigurationJSONValue(value any) (any, error) {
	return normalizedUnifiedAgentConfigurationJSONValueWithOptions(value, false)
}

func normalizedUnifiedAgentConfigurationReadbackJSONValue(value any) (any, error) {
	return normalizedUnifiedAgentConfigurationJSONValueWithOptions(value, true)
}

func normalizedUnifiedAgentConfigurationJSONValueWithOptions(value any, preserveExplicitZeroValues bool) (any, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	return expandUnifiedAgentConfigurationJSONData(decoded, preserveExplicitZeroValues)
}

func expandUnifiedAgentConfigurationJSONData(value any, preserveExplicitZeroValues bool) (any, error) {
	switch current := value.(type) {
	case map[string]any:
		if raw, ok := meaningfulJSONStringData(current); ok && !hasMeaningfulNonJSONDataField(current) {
			var decoded any
			if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
				return nil, err
			}
			return expandUnifiedAgentConfigurationJSONData(decoded, true)
		}

		delete(current, "jsonData")
		for key, child := range current {
			expanded, err := expandUnifiedAgentConfigurationJSONData(child, preserveExplicitZeroValues)
			if err != nil {
				return nil, err
			}
			if expanded == nil {
				delete(current, key)
				continue
			}
			current[key] = expanded
		}
		if len(current) == 0 {
			return nil, nil
		}
		return current, nil
	case []any:
		out := make([]any, 0, len(current))
		for _, child := range current {
			expanded, err := expandUnifiedAgentConfigurationJSONData(child, preserveExplicitZeroValues)
			if err != nil {
				return nil, err
			}
			if expanded != nil {
				out = append(out, expanded)
			}
		}
		if len(out) == 0 {
			return nil, nil
		}
		return out, nil
	default:
		if !meaningfulUnifiedAgentConfigurationJSONValue(current, preserveExplicitZeroValues) {
			return nil, nil
		}
		return current, nil
	}
}

func meaningfulJSONStringData(values map[string]any) (string, bool) {
	raw, ok := values["jsonData"]
	if !ok {
		return "", false
	}
	rawString, ok := raw.(string)
	if !ok {
		return "", false
	}
	rawString = strings.TrimSpace(rawString)
	return rawString, rawString != ""
}

func hasMeaningfulNonJSONDataField(values map[string]any) bool {
	for key, value := range values {
		if key == "jsonData" {
			continue
		}
		if meaningfulUnifiedAgentConfigurationJSONValue(value, false) {
			return true
		}
	}
	return false
}

func meaningfulUnifiedAgentConfigurationJSONValue(value any, preserveExplicitZeroValues bool) bool {
	switch current := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(current) != ""
	case bool:
		return preserveExplicitZeroValues || current
	case float64:
		return preserveExplicitZeroValues || current != 0
	case []any:
		return len(current) != 0
	case map[string]any:
		return hasMeaningfulNonJSONDataField(current)
	default:
		return true
	}
}

func getUnifiedAgentConfigurationWorkRequest(
	ctx context.Context,
	client unifiedAgentConfigurationOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize UnifiedAgentConfiguration OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("UnifiedAgentConfiguration OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, loggingsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveUnifiedAgentConfigurationGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := unifiedAgentConfigurationWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	if action, ok := resolveUnifiedAgentConfigurationWorkRequestAction(current.Resources, ""); ok {
		return string(action), nil
	}
	return string(unifiedAgentConfigurationWorkRequestActionForOperation(current.OperationType)), nil
}

func resolveUnifiedAgentConfigurationGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := unifiedAgentConfigurationWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := unifiedAgentConfigurationWorkRequestPhaseForOperation(current.OperationType)
	return phase, ok, nil
}

func recoverUnifiedAgentConfigurationIDFromGeneratedWorkRequest(
	_ *loggingv1beta1.UnifiedAgentConfiguration,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := unifiedAgentConfigurationWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	action := unifiedAgentConfigurationWorkRequestActionForPhase(phase)
	if id, ok := resolveUnifiedAgentConfigurationWorkRequestResourceID(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveUnifiedAgentConfigurationWorkRequestResourceID(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("UnifiedAgentConfiguration work request %s did not expose a %s identifier", stringValue(current.Id), phase)
}

func unifiedAgentConfigurationGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := unifiedAgentConfigurationWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("UnifiedAgentConfiguration %s work request %s is %s", phase, stringValue(current.Id), current.Status)
}

func unifiedAgentConfigurationWorkRequestFromAny(workRequest any) (loggingsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case loggingsdk.WorkRequest:
		return current, nil
	case *loggingsdk.WorkRequest:
		if current == nil {
			return loggingsdk.WorkRequest{}, fmt.Errorf("UnifiedAgentConfiguration work request is nil")
		}
		return *current, nil
	default:
		return loggingsdk.WorkRequest{}, fmt.Errorf("unexpected UnifiedAgentConfiguration work request type %T", workRequest)
	}
}

func resolveUnifiedAgentConfigurationWorkRequestAction(
	resources []loggingsdk.WorkRequestResource,
	preferred loggingsdk.ActionTypesEnum,
) (loggingsdk.ActionTypesEnum, bool) {
	var candidate loggingsdk.ActionTypesEnum
	for _, resource := range resources {
		action := resource.ActionType
		if action == "" ||
			action == loggingsdk.ActionTypesInProgress ||
			action == loggingsdk.ActionTypesRelated {
			continue
		}
		if preferred != "" && action != preferred {
			continue
		}
		if candidate == "" {
			candidate = action
			continue
		}
		if candidate != action {
			return "", false
		}
	}
	return candidate, candidate != ""
}

func resolveUnifiedAgentConfigurationWorkRequestResourceID(
	resources []loggingsdk.WorkRequestResource,
	action loggingsdk.ActionTypesEnum,
	preferConfigurationOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferConfigurationOnly && !isUnifiedAgentConfigurationWorkRequestResource(resource) {
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

func isUnifiedAgentConfigurationWorkRequestResource(resource loggingsdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	if strings.Contains(entityType, "configuration") || strings.Contains(entityType, "unifiedagent") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "unifiedagentconfigurations") ||
		strings.Contains(entityURI, "unified-agent-configurations")
}

func unifiedAgentConfigurationWorkRequestPhaseForOperation(
	operation loggingsdk.OperationTypesEnum,
) (shared.OSOKAsyncPhase, bool) {
	switch operation {
	case loggingsdk.OperationTypesCreateConfiguration:
		return shared.OSOKAsyncPhaseCreate, true
	case loggingsdk.OperationTypesUpdateConfiguration:
		return shared.OSOKAsyncPhaseUpdate, true
	case loggingsdk.OperationTypesDeleteConfiguration:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func unifiedAgentConfigurationWorkRequestActionForOperation(
	operation loggingsdk.OperationTypesEnum,
) loggingsdk.ActionTypesEnum {
	switch operation {
	case loggingsdk.OperationTypesCreateConfiguration:
		return loggingsdk.ActionTypesCreated
	case loggingsdk.OperationTypesUpdateConfiguration:
		return loggingsdk.ActionTypesUpdated
	case loggingsdk.OperationTypesDeleteConfiguration:
		return loggingsdk.ActionTypesDeleted
	default:
		return ""
	}
}

func unifiedAgentConfigurationWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) loggingsdk.ActionTypesEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return loggingsdk.ActionTypesCreated
	case shared.OSOKAsyncPhaseUpdate:
		return loggingsdk.ActionTypesUpdated
	case shared.OSOKAsyncPhaseDelete:
		return loggingsdk.ActionTypesDeleted
	default:
		return ""
	}
}

func unifiedAgentConfigurationFromResponse(response any) (loggingsdk.UnifiedAgentConfiguration, bool) {
	switch current := response.(type) {
	case loggingsdk.GetUnifiedAgentConfigurationResponse:
		return current.UnifiedAgentConfiguration, true
	case *loggingsdk.GetUnifiedAgentConfigurationResponse:
		if current == nil {
			return loggingsdk.UnifiedAgentConfiguration{}, false
		}
		return current.UnifiedAgentConfiguration, true
	case loggingsdk.UnifiedAgentConfiguration:
		return current, true
	case *loggingsdk.UnifiedAgentConfiguration:
		if current == nil {
			return loggingsdk.UnifiedAgentConfiguration{}, false
		}
		return *current, true
	default:
		return loggingsdk.UnifiedAgentConfiguration{}, false
	}
}

func cloneUnifiedAgentConfigurationStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneUnifiedAgentConfigurationDefinedTags(input map[string]map[string]interface{}) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(input))
	for key, value := range input {
		if value == nil {
			cloned[key] = nil
			continue
		}
		inner := make(map[string]interface{}, len(value))
		for innerKey, innerValue := range value {
			inner[innerKey] = innerValue
		}
		cloned[key] = inner
	}
	return cloned
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(stringValue(current)) == strings.TrimSpace(desired)
}
