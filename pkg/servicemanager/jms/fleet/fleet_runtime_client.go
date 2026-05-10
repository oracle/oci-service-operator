/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	jmssdk "github.com/oracle/oci-go-sdk/v65/jms"
	jmsv1beta1 "github.com/oracle/oci-service-operator/api/jms/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

var fleetWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(jmssdk.OperationStatusAccepted),
		string(jmssdk.OperationStatusInProgress),
		string(jmssdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(jmssdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(jmssdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(jmssdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(jmssdk.OperationTypeCreateFleet)},
	UpdateActionTokens:    []string{string(jmssdk.OperationTypeUpdateFleet)},
	DeleteActionTokens:    []string{string(jmssdk.OperationTypeDeleteFleet)},
}

type fleetOCIClient interface {
	CreateFleet(context.Context, jmssdk.CreateFleetRequest) (jmssdk.CreateFleetResponse, error)
	GetFleet(context.Context, jmssdk.GetFleetRequest) (jmssdk.GetFleetResponse, error)
	ListFleets(context.Context, jmssdk.ListFleetsRequest) (jmssdk.ListFleetsResponse, error)
	UpdateFleet(context.Context, jmssdk.UpdateFleetRequest) (jmssdk.UpdateFleetResponse, error)
	DeleteFleet(context.Context, jmssdk.DeleteFleetRequest) (jmssdk.DeleteFleetResponse, error)
	GetWorkRequest(context.Context, jmssdk.GetWorkRequestRequest) (jmssdk.GetWorkRequestResponse, error)
}

func init() {
	registerFleetRuntimeHooksMutator(func(manager *FleetServiceManager, hooks *FleetRuntimeHooks) {
		client, initErr := newFleetSDKClient(manager)
		applyFleetRuntimeHooks(hooks, client, initErr)
	})
}

func newFleetSDKClient(manager *FleetServiceManager) (fleetOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("Fleet service manager is nil")
	}
	client, err := jmssdk.NewJavaManagementServiceClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyFleetRuntimeHooks(
	hooks *FleetRuntimeHooks,
	client fleetOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedFleetRuntimeSemantics()
	hooks.BuildCreateBody = func(
		_ context.Context,
		resource *jmsv1beta1.Fleet,
		_ string,
	) (any, error) {
		return buildFleetCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *jmsv1beta1.Fleet,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildFleetUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardFleetExistingBeforeCreate
	hooks.Create.Fields = fleetCreateFields()
	hooks.Get.Fields = fleetGetFields()
	hooks.List.Fields = fleetListFields()
	hooks.Update.Fields = fleetUpdateFields()
	hooks.Delete.Fields = fleetDeleteFields()
	hooks.Async.Adapter = fleetWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getFleetWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveFleetGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveFleetGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverFleetIDFromGeneratedWorkRequest
	hooks.Async.Message = fleetGeneratedWorkRequestMessage
}

func newFleetServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client fleetOCIClient,
) FleetServiceClient {
	return defaultFleetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*jmsv1beta1.Fleet](
			newFleetRuntimeConfig(log, client),
		),
	}
}

func newFleetRuntimeConfig(
	log loggerutil.OSOKLogger,
	client fleetOCIClient,
) generatedruntime.Config[*jmsv1beta1.Fleet] {
	hooks := newFleetRuntimeHooksWithOCIClient(client)
	applyFleetRuntimeHooks(&hooks, client, nil)
	return buildFleetGeneratedRuntimeConfig(&FleetServiceManager{Log: log}, hooks)
}

func newFleetRuntimeHooksWithOCIClient(client fleetOCIClient) FleetRuntimeHooks {
	return FleetRuntimeHooks{
		Semantics: reviewedFleetRuntimeSemantics(),
		Create: runtimeOperationHooks[jmssdk.CreateFleetRequest, jmssdk.CreateFleetResponse]{
			Fields: fleetCreateFields(),
			Call: func(ctx context.Context, request jmssdk.CreateFleetRequest) (jmssdk.CreateFleetResponse, error) {
				return client.CreateFleet(ctx, request)
			},
		},
		Get: runtimeOperationHooks[jmssdk.GetFleetRequest, jmssdk.GetFleetResponse]{
			Fields: fleetGetFields(),
			Call: func(ctx context.Context, request jmssdk.GetFleetRequest) (jmssdk.GetFleetResponse, error) {
				return client.GetFleet(ctx, request)
			},
		},
		List: runtimeOperationHooks[jmssdk.ListFleetsRequest, jmssdk.ListFleetsResponse]{
			Fields: fleetListFields(),
			Call: func(ctx context.Context, request jmssdk.ListFleetsRequest) (jmssdk.ListFleetsResponse, error) {
				return client.ListFleets(ctx, request)
			},
		},
		Update: runtimeOperationHooks[jmssdk.UpdateFleetRequest, jmssdk.UpdateFleetResponse]{
			Fields: fleetUpdateFields(),
			Call: func(ctx context.Context, request jmssdk.UpdateFleetRequest) (jmssdk.UpdateFleetResponse, error) {
				return client.UpdateFleet(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[jmssdk.DeleteFleetRequest, jmssdk.DeleteFleetResponse]{
			Fields: fleetDeleteFields(),
			Call: func(ctx context.Context, request jmssdk.DeleteFleetRequest) (jmssdk.DeleteFleetResponse, error) {
				return client.DeleteFleet(ctx, request)
			},
		},
	}
}

func reviewedFleetRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newFleetRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName", "id"},
	}
	semantics.Mutation = generatedruntime.MutationSemantics{
		Mutable: []string{
			"definedTags",
			"description",
			"displayName",
			"freeformTags",
			"inventoryLog",
			"isAdvancedFeaturesEnabled",
			"operationLog",
		},
		ForceNew:      []string{"compartmentId"},
		ConflictsWith: map[string][]string{},
	}
	semantics.CreateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetFleet",
		Hooks:    semantics.Hooks.Create,
	}
	semantics.UpdateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetFleet",
		Hooks:    semantics.Hooks.Update,
	}
	semantics.DeleteFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetFleet/ListFleets confirm-delete",
		Hooks:    semantics.Hooks.Delete,
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func fleetCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateFleetDetails", RequestName: "CreateFleetDetails", Contribution: "body"},
	}
}

func fleetGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "FleetId", RequestName: "fleetId", Contribution: "path", PreferResourceID: true},
	}
}

func fleetListFields() []generatedruntime.RequestField {
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
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
	}
}

func fleetUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "FleetId", RequestName: "fleetId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateFleetDetails", RequestName: "UpdateFleetDetails", Contribution: "body"},
	}
}

func fleetDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "FleetId", RequestName: "fleetId", Contribution: "path", PreferResourceID: true},
	}
}

func guardFleetExistingBeforeCreate(
	_ context.Context,
	resource *jmsv1beta1.Fleet,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("Fleet resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildFleetCreateBody(resource *jmsv1beta1.Fleet) (jmssdk.CreateFleetDetails, error) {
	if resource == nil {
		return jmssdk.CreateFleetDetails{}, fmt.Errorf("Fleet resource is nil")
	}

	displayName := strings.TrimSpace(resource.Spec.DisplayName)
	if displayName == "" {
		return jmssdk.CreateFleetDetails{}, fmt.Errorf("Fleet spec.displayName is required")
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		return jmssdk.CreateFleetDetails{}, fmt.Errorf("Fleet spec.compartmentId is required")
	}

	inventoryLog, err := fleetRequiredCustomLog(resource.Spec.InventoryLog.LogGroupId, resource.Spec.InventoryLog.LogId, "inventoryLog")
	if err != nil {
		return jmssdk.CreateFleetDetails{}, err
	}
	operationLog, err := fleetOptionalCustomLog(resource.Spec.OperationLog.LogGroupId, resource.Spec.OperationLog.LogId, "operationLog")
	if err != nil {
		return jmssdk.CreateFleetDetails{}, err
	}

	details := jmssdk.CreateFleetDetails{
		DisplayName:   common.String(displayName),
		CompartmentId: common.String(compartmentID),
		InventoryLog:  inventoryLog,
	}
	if description := strings.TrimSpace(resource.Spec.Description); description != "" {
		details.Description = common.String(description)
	}
	if operationLog != nil {
		details.OperationLog = operationLog
	}
	if resource.Spec.IsAdvancedFeaturesEnabled {
		details.IsAdvancedFeaturesEnabled = common.Bool(true)
	}
	if len(resource.Spec.FreeformTags) != 0 {
		details.FreeformTags = maps.Clone(resource.Spec.FreeformTags)
	}
	if len(resource.Spec.DefinedTags) != 0 {
		details.DefinedTags = fleetDefinedTagsFromSpec(resource.Spec.DefinedTags)
	}
	return details, nil
}

func buildFleetUpdateBody(
	resource *jmsv1beta1.Fleet,
	currentResponse any,
) (jmssdk.UpdateFleetDetails, bool, error) {
	if resource == nil {
		return jmssdk.UpdateFleetDetails{}, false, fmt.Errorf("Fleet resource is nil")
	}

	current, err := fleetFromResponse(currentResponse)
	if err != nil {
		return jmssdk.UpdateFleetDetails{}, false, err
	}

	inventoryLog, err := fleetRequiredCustomLog(resource.Spec.InventoryLog.LogGroupId, resource.Spec.InventoryLog.LogId, "inventoryLog")
	if err != nil {
		return jmssdk.UpdateFleetDetails{}, false, err
	}
	operationLog, err := fleetOptionalCustomLog(resource.Spec.OperationLog.LogGroupId, resource.Spec.OperationLog.LogId, "operationLog")
	if err != nil {
		return jmssdk.UpdateFleetDetails{}, false, err
	}

	details := jmssdk.UpdateFleetDetails{}
	updateNeeded := false

	if desired, ok := fleetDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := fleetDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := fleetDesiredCustomLogUpdate(inventoryLog, current.InventoryLog); ok {
		details.InventoryLog = desired
		updateNeeded = true
	}
	if desired, ok := fleetDesiredCustomLogUpdate(operationLog, current.OperationLog); ok {
		details.OperationLog = desired
		updateNeeded = true
	}
	if desired, ok := fleetDesiredBoolUpdate(resource.Spec.IsAdvancedFeaturesEnabled, current.IsAdvancedFeaturesEnabled); ok {
		details.IsAdvancedFeaturesEnabled = desired
		updateNeeded = true
	}
	if desired, ok := fleetDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := fleetDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func fleetFromResponse(currentResponse any) (jmssdk.Fleet, error) {
	switch current := currentResponse.(type) {
	case jmssdk.Fleet:
		return current, nil
	case *jmssdk.Fleet:
		if current == nil {
			return jmssdk.Fleet{}, fmt.Errorf("current Fleet response is nil")
		}
		return *current, nil
	case jmssdk.FleetSummary:
		return jmssdk.Fleet{
			Id:                                   current.Id,
			DisplayName:                          current.DisplayName,
			Description:                          current.Description,
			CompartmentId:                        current.CompartmentId,
			ApproximateJreCount:                  current.ApproximateJreCount,
			ApproximateInstallationCount:         current.ApproximateInstallationCount,
			ApproximateApplicationCount:          current.ApproximateApplicationCount,
			ApproximateManagedInstanceCount:      current.ApproximateManagedInstanceCount,
			ApproximateJavaServerCount:           current.ApproximateJavaServerCount,
			ApproximateLibraryCount:              current.ApproximateLibraryCount,
			ApproximateLibraryVulnerabilityCount: current.ApproximateLibraryVulnerabilityCount,
			TimeCreated:                          current.TimeCreated,
			LifecycleState:                       current.LifecycleState,
			InventoryLog:                         current.InventoryLog,
			OperationLog:                         current.OperationLog,
			IsAdvancedFeaturesEnabled:            current.IsAdvancedFeaturesEnabled,
			IsExportSettingEnabled:               current.IsExportSettingEnabled,
			DefinedTags:                          current.DefinedTags,
			FreeformTags:                         current.FreeformTags,
			SystemTags:                           current.SystemTags,
		}, nil
	case *jmssdk.FleetSummary:
		if current == nil {
			return jmssdk.Fleet{}, fmt.Errorf("current Fleet response is nil")
		}
		return fleetFromResponse(*current)
	case jmssdk.GetFleetResponse:
		return current.Fleet, nil
	case *jmssdk.GetFleetResponse:
		if current == nil {
			return jmssdk.Fleet{}, fmt.Errorf("current Fleet response is nil")
		}
		return current.Fleet, nil
	default:
		return jmssdk.Fleet{}, fmt.Errorf("unexpected current Fleet response type %T", currentResponse)
	}
}

func fleetDesiredStringUpdate(spec string, current *string) (*string, bool) {
	currentValue := ""
	if current != nil {
		currentValue = strings.TrimSpace(*current)
	}
	if spec == currentValue {
		return nil, false
	}
	if spec == "" && current == nil {
		return nil, false
	}
	return common.String(spec), true
}

func fleetDesiredCustomLogUpdate(spec *jmssdk.CustomLog, current *jmssdk.CustomLog) (*jmssdk.CustomLog, bool) {
	if spec == nil {
		return nil, false
	}
	if fleetCustomLogEqual(spec, current) {
		return nil, false
	}
	return spec, true
}

func fleetDesiredBoolUpdate(spec bool, current *bool) (*bool, bool) {
	currentValue := false
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	if !spec && current == nil {
		return nil, false
	}
	return common.Bool(spec), true
}

func fleetDesiredFreeformTagsUpdate(
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

func fleetDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := fleetDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if fleetJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func fleetDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
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

func fleetJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func fleetRequiredCustomLog(logGroupID string, logID string, fieldName string) (*jmssdk.CustomLog, error) {
	return fleetCustomLogFromStrings(logGroupID, logID, fieldName, true)
}

func fleetOptionalCustomLog(logGroupID string, logID string, fieldName string) (*jmssdk.CustomLog, error) {
	return fleetCustomLogFromStrings(logGroupID, logID, fieldName, false)
}

func fleetCustomLogFromStrings(
	logGroupID string,
	logID string,
	fieldName string,
	required bool,
) (*jmssdk.CustomLog, error) {
	logGroupID = strings.TrimSpace(logGroupID)
	logID = strings.TrimSpace(logID)
	if logGroupID == "" && logID == "" && !required {
		return nil, nil
	}
	if logGroupID == "" || logID == "" {
		return nil, fmt.Errorf("Fleet spec.%s requires both logGroupId and logId", fieldName)
	}
	return &jmssdk.CustomLog{
		LogGroupId: common.String(logGroupID),
		LogId:      common.String(logID),
	}, nil
}

func fleetCustomLogEqual(left *jmssdk.CustomLog, right *jmssdk.CustomLog) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return fleetStringValue(left.LogGroupId) == fleetStringValue(right.LogGroupId) &&
			fleetStringValue(left.LogId) == fleetStringValue(right.LogId)
	}
}

func getFleetWorkRequest(
	ctx context.Context,
	client fleetOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize Fleet OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("Fleet OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, jmssdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveFleetGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := fleetWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveFleetGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := fleetWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := fleetWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverFleetIDFromGeneratedWorkRequest(
	_ *jmsv1beta1.Fleet,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := fleetWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := fleetWorkRequestActionForPhase(phase)
	if id, ok := resolveFleetIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveFleetIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("Fleet work request %s does not expose a Fleet identifier", fleetStringValue(current.Id))
}

func fleetGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := fleetWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("Fleet %s work request %s is %s", phase, fleetStringValue(current.Id), current.Status)
}

func fleetWorkRequestFromAny(workRequest any) (jmssdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case jmssdk.WorkRequest:
		return current, nil
	case *jmssdk.WorkRequest:
		if current == nil {
			return jmssdk.WorkRequest{}, fmt.Errorf("Fleet work request is nil")
		}
		return *current, nil
	default:
		return jmssdk.WorkRequest{}, fmt.Errorf("unexpected Fleet work request type %T", workRequest)
	}
}

func fleetWorkRequestPhaseFromOperationType(operationType jmssdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case jmssdk.OperationTypeCreateFleet:
		return shared.OSOKAsyncPhaseCreate, true
	case jmssdk.OperationTypeUpdateFleet:
		return shared.OSOKAsyncPhaseUpdate, true
	case jmssdk.OperationTypeDeleteFleet:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func fleetWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) jmssdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return jmssdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return jmssdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return jmssdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveFleetIDFromResources(
	resources []jmssdk.WorkRequestResource,
	action jmssdk.ActionTypeEnum,
	preferFleetOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferFleetOnly && !isFleetWorkRequestResource(resource) {
			continue
		}

		id := strings.TrimSpace(fleetStringValue(resource.Identifier))
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

func isFleetWorkRequestResource(resource jmssdk.WorkRequestResource) bool {
	return strings.Contains(normalizeFleetWorkRequestToken(fleetStringValue(resource.EntityType)), "fleet")
}

func normalizeFleetWorkRequestToken(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(value))
}

func fleetStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
