/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package knowledgebase

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"unicode"

	admsdk "github.com/oracle/oci-go-sdk/v65/adm"
	"github.com/oracle/oci-go-sdk/v65/common"
	admv1beta1 "github.com/oracle/oci-service-operator/api/adm/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

var knowledgeBaseWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(admsdk.OperationStatusAccepted),
		string(admsdk.OperationStatusInProgress),
		string(admsdk.OperationStatusWaiting),
		string(admsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(admsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(admsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(admsdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(admsdk.OperationTypeCreateKnowledgeBase)},
	UpdateActionTokens:    []string{string(admsdk.OperationTypeUpdateKnowledgeBase)},
	DeleteActionTokens:    []string{string(admsdk.OperationTypeDeleteKnowledgeBase)},
}

type knowledgeBaseOCIClient interface {
	CreateKnowledgeBase(context.Context, admsdk.CreateKnowledgeBaseRequest) (admsdk.CreateKnowledgeBaseResponse, error)
	GetKnowledgeBase(context.Context, admsdk.GetKnowledgeBaseRequest) (admsdk.GetKnowledgeBaseResponse, error)
	ListKnowledgeBases(context.Context, admsdk.ListKnowledgeBasesRequest) (admsdk.ListKnowledgeBasesResponse, error)
	UpdateKnowledgeBase(context.Context, admsdk.UpdateKnowledgeBaseRequest) (admsdk.UpdateKnowledgeBaseResponse, error)
	DeleteKnowledgeBase(context.Context, admsdk.DeleteKnowledgeBaseRequest) (admsdk.DeleteKnowledgeBaseResponse, error)
	GetWorkRequest(context.Context, admsdk.GetWorkRequestRequest) (admsdk.GetWorkRequestResponse, error)
}

func init() {
	registerKnowledgeBaseRuntimeHooksMutator(func(manager *KnowledgeBaseServiceManager, hooks *KnowledgeBaseRuntimeHooks) {
		client, initErr := newKnowledgeBaseSDKClient(manager)
		applyKnowledgeBaseRuntimeHooks(hooks, client, initErr)
	})
}

func newKnowledgeBaseSDKClient(manager *KnowledgeBaseServiceManager) (knowledgeBaseOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("KnowledgeBase service manager is nil")
	}
	client, err := admsdk.NewApplicationDependencyManagementClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyKnowledgeBaseRuntimeHooks(
	hooks *KnowledgeBaseRuntimeHooks,
	client knowledgeBaseOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedKnowledgeBaseRuntimeSemantics()
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *admv1beta1.KnowledgeBase,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildKnowledgeBaseUpdateBody(resource, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardKnowledgeBaseExistingBeforeCreate
	hooks.Create.Fields = knowledgeBaseCreateFields()
	hooks.Get.Fields = knowledgeBaseGetFields()
	hooks.List.Fields = knowledgeBaseListFields()
	hooks.Update.Fields = knowledgeBaseUpdateFields()
	hooks.Delete.Fields = knowledgeBaseDeleteFields()
	hooks.Async.Adapter = knowledgeBaseWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getKnowledgeBaseWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveKnowledgeBaseGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveKnowledgeBaseGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverKnowledgeBaseIDFromGeneratedWorkRequest
	hooks.Async.Message = knowledgeBaseGeneratedWorkRequestMessage
}

func newKnowledgeBaseServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client knowledgeBaseOCIClient,
) KnowledgeBaseServiceClient {
	return defaultKnowledgeBaseServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*admv1beta1.KnowledgeBase](
			newKnowledgeBaseRuntimeConfig(log, client),
		),
	}
}

func newKnowledgeBaseRuntimeConfig(
	log loggerutil.OSOKLogger,
	client knowledgeBaseOCIClient,
) generatedruntime.Config[*admv1beta1.KnowledgeBase] {
	hooks := newKnowledgeBaseRuntimeHooksWithOCIClient(client)
	applyKnowledgeBaseRuntimeHooks(&hooks, client, nil)
	return buildKnowledgeBaseGeneratedRuntimeConfig(&KnowledgeBaseServiceManager{Log: log}, hooks)
}

func newKnowledgeBaseRuntimeHooksWithOCIClient(client knowledgeBaseOCIClient) KnowledgeBaseRuntimeHooks {
	return KnowledgeBaseRuntimeHooks{
		Semantics: newKnowledgeBaseRuntimeSemantics(),
		Create: runtimeOperationHooks[admsdk.CreateKnowledgeBaseRequest, admsdk.CreateKnowledgeBaseResponse]{
			Fields: knowledgeBaseCreateFields(),
			Call: func(ctx context.Context, request admsdk.CreateKnowledgeBaseRequest) (admsdk.CreateKnowledgeBaseResponse, error) {
				return client.CreateKnowledgeBase(ctx, request)
			},
		},
		Get: runtimeOperationHooks[admsdk.GetKnowledgeBaseRequest, admsdk.GetKnowledgeBaseResponse]{
			Fields: knowledgeBaseGetFields(),
			Call: func(ctx context.Context, request admsdk.GetKnowledgeBaseRequest) (admsdk.GetKnowledgeBaseResponse, error) {
				return client.GetKnowledgeBase(ctx, request)
			},
		},
		List: runtimeOperationHooks[admsdk.ListKnowledgeBasesRequest, admsdk.ListKnowledgeBasesResponse]{
			Fields: knowledgeBaseListFields(),
			Call: func(ctx context.Context, request admsdk.ListKnowledgeBasesRequest) (admsdk.ListKnowledgeBasesResponse, error) {
				return client.ListKnowledgeBases(ctx, request)
			},
		},
		Update: runtimeOperationHooks[admsdk.UpdateKnowledgeBaseRequest, admsdk.UpdateKnowledgeBaseResponse]{
			Fields: knowledgeBaseUpdateFields(),
			Call: func(ctx context.Context, request admsdk.UpdateKnowledgeBaseRequest) (admsdk.UpdateKnowledgeBaseResponse, error) {
				return client.UpdateKnowledgeBase(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[admsdk.DeleteKnowledgeBaseRequest, admsdk.DeleteKnowledgeBaseResponse]{
			Fields: knowledgeBaseDeleteFields(),
			Call: func(ctx context.Context, request admsdk.DeleteKnowledgeBaseRequest) (admsdk.DeleteKnowledgeBaseResponse, error) {
				return client.DeleteKnowledgeBase(ctx, request)
			},
		},
	}
}

func reviewedKnowledgeBaseRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newKnowledgeBaseRuntimeSemantics()
	semantics.Lifecycle = generatedruntime.LifecycleSemantics{
		ProvisioningStates: []string{string(admsdk.KnowledgeBaseLifecycleStateCreating)},
		UpdatingStates:     []string{string(admsdk.KnowledgeBaseLifecycleStateUpdating)},
		ActiveStates:       []string{string(admsdk.KnowledgeBaseLifecycleStateActive)},
	}
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName", "id"},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func knowledgeBaseCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateKnowledgeBaseDetails", RequestName: "CreateKnowledgeBaseDetails", Contribution: "body"},
	}
}

func knowledgeBaseGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "KnowledgeBaseId", RequestName: "knowledgeBaseId", Contribution: "path", PreferResourceID: true},
	}
}

func knowledgeBaseListFields() []generatedruntime.RequestField {
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

func knowledgeBaseUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "KnowledgeBaseId", RequestName: "knowledgeBaseId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateKnowledgeBaseDetails", RequestName: "UpdateKnowledgeBaseDetails", Contribution: "body"},
	}
}

func knowledgeBaseDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "KnowledgeBaseId", RequestName: "knowledgeBaseId", Contribution: "path", PreferResourceID: true},
	}
}

func guardKnowledgeBaseExistingBeforeCreate(
	_ context.Context,
	resource *admv1beta1.KnowledgeBase,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("KnowledgeBase resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildKnowledgeBaseUpdateBody(
	resource *admv1beta1.KnowledgeBase,
	currentResponse any,
) (admsdk.UpdateKnowledgeBaseDetails, bool, error) {
	if resource == nil {
		return admsdk.UpdateKnowledgeBaseDetails{}, false, fmt.Errorf("KnowledgeBase resource is nil")
	}

	current, err := knowledgeBaseFromResponse(currentResponse)
	if err != nil {
		return admsdk.UpdateKnowledgeBaseDetails{}, false, err
	}

	details := admsdk.UpdateKnowledgeBaseDetails{}
	updateNeeded := false

	if desired, ok := knowledgeBaseDesiredDisplayNameUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := knowledgeBaseDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := knowledgeBaseDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func knowledgeBaseFromResponse(currentResponse any) (admsdk.KnowledgeBase, error) {
	switch current := currentResponse.(type) {
	case admsdk.KnowledgeBase:
		return current, nil
	case *admsdk.KnowledgeBase:
		if current == nil {
			return admsdk.KnowledgeBase{}, fmt.Errorf("current KnowledgeBase response is nil")
		}
		return *current, nil
	case admsdk.KnowledgeBaseSummary:
		return admsdk.KnowledgeBase{
			Id:             current.Id,
			DisplayName:    current.DisplayName,
			TimeCreated:    current.TimeCreated,
			TimeUpdated:    current.TimeUpdated,
			LifecycleState: current.LifecycleState,
			CompartmentId:  current.CompartmentId,
			FreeformTags:   current.FreeformTags,
			DefinedTags:    current.DefinedTags,
			SystemTags:     current.SystemTags,
		}, nil
	case *admsdk.KnowledgeBaseSummary:
		if current == nil {
			return admsdk.KnowledgeBase{}, fmt.Errorf("current KnowledgeBase response is nil")
		}
		return knowledgeBaseFromResponse(*current)
	case admsdk.GetKnowledgeBaseResponse:
		return current.KnowledgeBase, nil
	case *admsdk.GetKnowledgeBaseResponse:
		if current == nil {
			return admsdk.KnowledgeBase{}, fmt.Errorf("current KnowledgeBase response is nil")
		}
		return current.KnowledgeBase, nil
	default:
		return admsdk.KnowledgeBase{}, fmt.Errorf("unexpected current KnowledgeBase response type %T", currentResponse)
	}
}

func knowledgeBaseDesiredDisplayNameUpdate(spec string, current *string) (*string, bool) {
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

func knowledgeBaseDesiredFreeformTagsUpdate(
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

func knowledgeBaseDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := knowledgeBaseDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if knowledgeBaseJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func knowledgeBaseDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
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

func knowledgeBaseJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func getKnowledgeBaseWorkRequest(
	ctx context.Context,
	client knowledgeBaseOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize KnowledgeBase OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("KnowledgeBase OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, admsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveKnowledgeBaseGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := knowledgeBaseWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveKnowledgeBaseGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := knowledgeBaseWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := knowledgeBaseWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverKnowledgeBaseIDFromGeneratedWorkRequest(
	_ *admv1beta1.KnowledgeBase,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := knowledgeBaseWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := knowledgeBaseWorkRequestActionForPhase(phase)
	if id, ok := resolveKnowledgeBaseIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveKnowledgeBaseIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("KnowledgeBase work request %s does not expose a knowledge base identifier", stringValue(current.Id))
}

func knowledgeBaseGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := knowledgeBaseWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("KnowledgeBase %s work request %s is %s", phase, stringValue(current.Id), current.Status)
}

func knowledgeBaseWorkRequestFromAny(workRequest any) (admsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case admsdk.WorkRequest:
		return current, nil
	case *admsdk.WorkRequest:
		if current == nil {
			return admsdk.WorkRequest{}, fmt.Errorf("KnowledgeBase work request is nil")
		}
		return *current, nil
	default:
		return admsdk.WorkRequest{}, fmt.Errorf("unexpected KnowledgeBase work request type %T", workRequest)
	}
}

func knowledgeBaseWorkRequestPhaseFromOperationType(operationType admsdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case admsdk.OperationTypeCreateKnowledgeBase:
		return shared.OSOKAsyncPhaseCreate, true
	case admsdk.OperationTypeUpdateKnowledgeBase:
		return shared.OSOKAsyncPhaseUpdate, true
	case admsdk.OperationTypeDeleteKnowledgeBase:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func knowledgeBaseWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) admsdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return admsdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return admsdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return admsdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveKnowledgeBaseIDFromResources(
	resources []admsdk.WorkRequestResource,
	action admsdk.ActionTypeEnum,
	preferKnowledgeBaseOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferKnowledgeBaseOnly && !isKnowledgeBaseWorkRequestResource(resource) {
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

func isKnowledgeBaseWorkRequestResource(resource admsdk.WorkRequestResource) bool {
	return normalizeKnowledgeBaseWorkRequestToken(stringValue(resource.EntityType)) == "knowledgebase"
}

func normalizeKnowledgeBaseWorkRequestToken(value string) string {
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
