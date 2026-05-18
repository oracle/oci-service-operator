/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package opainstance

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	opasdk "github.com/oracle/oci-go-sdk/v65/opa"
	opav1beta1 "github.com/oracle/oci-service-operator/api/opa/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

var opaInstanceWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(opasdk.OperationStatusAccepted),
		string(opasdk.OperationStatusInProgress),
		string(opasdk.OperationStatusWaiting),
		string(opasdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(opasdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(opasdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(opasdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(opasdk.OperationTypeCreateOpaInstance)},
	UpdateActionTokens:    []string{string(opasdk.OperationTypeUpdateOpaInstance)},
	DeleteActionTokens:    []string{string(opasdk.OperationTypeDeleteOpaInstance)},
}

type opaInstanceOCIClient interface {
	CreateOpaInstance(context.Context, opasdk.CreateOpaInstanceRequest) (opasdk.CreateOpaInstanceResponse, error)
	GetOpaInstance(context.Context, opasdk.GetOpaInstanceRequest) (opasdk.GetOpaInstanceResponse, error)
	ListOpaInstances(context.Context, opasdk.ListOpaInstancesRequest) (opasdk.ListOpaInstancesResponse, error)
	UpdateOpaInstance(context.Context, opasdk.UpdateOpaInstanceRequest) (opasdk.UpdateOpaInstanceResponse, error)
	DeleteOpaInstance(context.Context, opasdk.DeleteOpaInstanceRequest) (opasdk.DeleteOpaInstanceResponse, error)
	GetWorkRequest(context.Context, opasdk.GetWorkRequestRequest) (opasdk.GetWorkRequestResponse, error)
}

func init() {
	registerOpaInstanceRuntimeHooksMutator(func(manager *OpaInstanceServiceManager, hooks *OpaInstanceRuntimeHooks) {
		client, initErr := newOpaInstanceSDKClient(manager)
		applyOpaInstanceRuntimeHooks(hooks, client, initErr)
	})
}

func newOpaInstanceSDKClient(manager *OpaInstanceServiceManager) (opaInstanceOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("OpaInstance service manager is nil")
	}

	client, err := opasdk.NewOpaInstanceClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyOpaInstanceRuntimeHooks(
	hooks *OpaInstanceRuntimeHooks,
	client opaInstanceOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedOpaInstanceRuntimeSemantics()
	hooks.ParityHooks.NormalizeDesiredState = normalizeOpaInstanceDesiredState
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *opav1beta1.OpaInstance,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildOpaInstanceUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardOpaInstanceExistingBeforeCreate
	hooks.Create.Fields = opaInstanceCreateFields()
	hooks.Get.Fields = opaInstanceGetFields()
	hooks.List.Fields = opaInstanceListFields()
	hooks.Update.Fields = opaInstanceUpdateFields()
	hooks.Delete.Fields = opaInstanceDeleteFields()
	hooks.Async.Adapter = opaInstanceWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getOpaInstanceWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveOpaInstanceGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveOpaInstanceGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverOpaInstanceIDFromGeneratedWorkRequest
	hooks.Async.Message = opaInstanceGeneratedWorkRequestMessage
}

func newOpaInstanceServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client opaInstanceOCIClient,
) OpaInstanceServiceClient {
	return defaultOpaInstanceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*opav1beta1.OpaInstance](
			newOpaInstanceRuntimeConfig(log, client),
		),
	}
}

func newOpaInstanceRuntimeConfig(
	log loggerutil.OSOKLogger,
	client opaInstanceOCIClient,
) generatedruntime.Config[*opav1beta1.OpaInstance] {
	hooks := newOpaInstanceRuntimeHooksWithOCIClient(client)
	applyOpaInstanceRuntimeHooks(&hooks, client, nil)
	return buildOpaInstanceGeneratedRuntimeConfig(&OpaInstanceServiceManager{Log: log}, hooks)
}

func newOpaInstanceRuntimeHooksWithOCIClient(client opaInstanceOCIClient) OpaInstanceRuntimeHooks {
	return OpaInstanceRuntimeHooks{
		Semantics: newOpaInstanceRuntimeSemantics(),
		Create: runtimeOperationHooks[opasdk.CreateOpaInstanceRequest, opasdk.CreateOpaInstanceResponse]{
			Fields: opaInstanceCreateFields(),
			Call: func(ctx context.Context, request opasdk.CreateOpaInstanceRequest) (opasdk.CreateOpaInstanceResponse, error) {
				return client.CreateOpaInstance(ctx, request)
			},
		},
		Get: runtimeOperationHooks[opasdk.GetOpaInstanceRequest, opasdk.GetOpaInstanceResponse]{
			Fields: opaInstanceGetFields(),
			Call: func(ctx context.Context, request opasdk.GetOpaInstanceRequest) (opasdk.GetOpaInstanceResponse, error) {
				return client.GetOpaInstance(ctx, request)
			},
		},
		List: runtimeOperationHooks[opasdk.ListOpaInstancesRequest, opasdk.ListOpaInstancesResponse]{
			Fields: opaInstanceListFields(),
			Call: func(ctx context.Context, request opasdk.ListOpaInstancesRequest) (opasdk.ListOpaInstancesResponse, error) {
				return client.ListOpaInstances(ctx, request)
			},
		},
		Update: runtimeOperationHooks[opasdk.UpdateOpaInstanceRequest, opasdk.UpdateOpaInstanceResponse]{
			Fields: opaInstanceUpdateFields(),
			Call: func(ctx context.Context, request opasdk.UpdateOpaInstanceRequest) (opasdk.UpdateOpaInstanceResponse, error) {
				return client.UpdateOpaInstance(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[opasdk.DeleteOpaInstanceRequest, opasdk.DeleteOpaInstanceResponse]{
			Fields: opaInstanceDeleteFields(),
			Call: func(ctx context.Context, request opasdk.DeleteOpaInstanceRequest) (opasdk.DeleteOpaInstanceResponse, error) {
				return client.DeleteOpaInstance(ctx, request)
			},
		},
	}
}

func reviewedOpaInstanceRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newOpaInstanceRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName"},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func normalizeOpaInstanceDesiredState(resource *opav1beta1.OpaInstance, currentResponse any) {
	if resource == nil || currentResponse == nil {
		return
	}

	// OCI does not echo this create-time credential input back on OpaInstance.
	resource.Spec.IdcsAt = ""
}

func opaInstanceCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateOpaInstanceDetails", RequestName: "CreateOpaInstanceDetails", Contribution: "body"},
	}
}

func opaInstanceGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OpaInstanceId", RequestName: "opaInstanceId", Contribution: "path", PreferResourceID: true},
	}
}

func opaInstanceListFields() []generatedruntime.RequestField {
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

func opaInstanceUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OpaInstanceId", RequestName: "opaInstanceId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateOpaInstanceDetails", RequestName: "UpdateOpaInstanceDetails", Contribution: "body"},
	}
}

func opaInstanceDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OpaInstanceId", RequestName: "opaInstanceId", Contribution: "path", PreferResourceID: true},
	}
}

func guardOpaInstanceExistingBeforeCreate(
	_ context.Context,
	resource *opav1beta1.OpaInstance,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("OpaInstance resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildOpaInstanceUpdateBody(
	resource *opav1beta1.OpaInstance,
	currentResponse any,
) (opasdk.UpdateOpaInstanceDetails, bool, error) {
	if resource == nil {
		return opasdk.UpdateOpaInstanceDetails{}, false, fmt.Errorf("OpaInstance resource is nil")
	}

	current, err := opaInstanceFromResponse(currentResponse)
	if err != nil {
		return opasdk.UpdateOpaInstanceDetails{}, false, err
	}

	details := opasdk.UpdateOpaInstanceDetails{}
	updateNeeded := false

	if desired, ok := opaInstanceDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := opaInstanceDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := opaInstanceDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := opaInstanceDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func opaInstanceFromResponse(currentResponse any) (opasdk.OpaInstance, error) {
	switch current := currentResponse.(type) {
	case opasdk.OpaInstance:
		return current, nil
	case *opasdk.OpaInstance:
		if current == nil {
			return opasdk.OpaInstance{}, fmt.Errorf("current OpaInstance response is nil")
		}
		return *current, nil
	case opasdk.OpaInstanceSummary:
		return opasdk.OpaInstance{
			Id:                  current.Id,
			DisplayName:         current.DisplayName,
			CompartmentId:       current.CompartmentId,
			ShapeName:           current.ShapeName,
			TimeCreated:         current.TimeCreated,
			LifecycleState:      current.LifecycleState,
			Description:         current.Description,
			InstanceUrl:         current.InstanceUrl,
			ConsumptionModel:    current.ConsumptionModel,
			MeteringType:        current.MeteringType,
			TimeUpdated:         current.TimeUpdated,
			IsBreakglassEnabled: current.IsBreakglassEnabled,
			FreeformTags:        current.FreeformTags,
			DefinedTags:         current.DefinedTags,
			SystemTags:          current.SystemTags,
		}, nil
	case *opasdk.OpaInstanceSummary:
		if current == nil {
			return opasdk.OpaInstance{}, fmt.Errorf("current OpaInstance response is nil")
		}
		return opaInstanceFromResponse(*current)
	case opasdk.GetOpaInstanceResponse:
		return current.OpaInstance, nil
	case *opasdk.GetOpaInstanceResponse:
		if current == nil {
			return opasdk.OpaInstance{}, fmt.Errorf("current OpaInstance response is nil")
		}
		return current.OpaInstance, nil
	default:
		return opasdk.OpaInstance{}, fmt.Errorf("unexpected current OpaInstance response type %T", currentResponse)
	}
}

func opaInstanceDesiredStringUpdate(spec string, current *string) (*string, bool) {
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

func opaInstanceDesiredFreeformTagsUpdate(
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

func opaInstanceDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := opaInstanceDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if opaInstanceJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func opaInstanceDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
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

func opaInstanceJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func getOpaInstanceWorkRequest(
	ctx context.Context,
	client opaInstanceOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize OpaInstance OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("OpaInstance OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, opasdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveOpaInstanceGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := opaInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveOpaInstanceGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := opaInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := opaInstanceWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverOpaInstanceIDFromGeneratedWorkRequest(
	_ *opav1beta1.OpaInstance,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := opaInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := opaInstanceWorkRequestActionForPhase(phase)
	if id, ok := resolveOpaInstanceIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveOpaInstanceIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("OpaInstance work request %s does not expose an OpaInstance identifier", stringValue(current.Id))
}

func opaInstanceGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := opaInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("OpaInstance %s work request %s is %s", phase, stringValue(current.Id), current.Status)
}

func opaInstanceWorkRequestFromAny(workRequest any) (opasdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case opasdk.WorkRequest:
		return current, nil
	case *opasdk.WorkRequest:
		if current == nil {
			return opasdk.WorkRequest{}, fmt.Errorf("OpaInstance work request is nil")
		}
		return *current, nil
	default:
		return opasdk.WorkRequest{}, fmt.Errorf("unexpected OpaInstance work request type %T", workRequest)
	}
}

func opaInstanceWorkRequestPhaseFromOperationType(
	operationType opasdk.OperationTypeEnum,
) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case opasdk.OperationTypeCreateOpaInstance:
		return shared.OSOKAsyncPhaseCreate, true
	case opasdk.OperationTypeUpdateOpaInstance:
		return shared.OSOKAsyncPhaseUpdate, true
	case opasdk.OperationTypeDeleteOpaInstance:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func opaInstanceWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) opasdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return opasdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return opasdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return opasdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveOpaInstanceIDFromResources(
	resources []opasdk.WorkRequestResource,
	action opasdk.ActionTypeEnum,
	preferOpaInstanceOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferOpaInstanceOnly && !isOpaInstanceWorkRequestResource(resource) {
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

func isOpaInstanceWorkRequestResource(resource opasdk.WorkRequestResource) bool {
	return normalizeOpaInstanceWorkRequestToken(stringValue(resource.EntityType)) == "opainstance"
}

func normalizeOpaInstanceWorkRequestToken(value string) string {
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
	return strings.TrimSpace(*value)
}
