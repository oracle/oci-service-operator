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

	"github.com/oracle/oci-go-sdk/v65/common"
	generativeaiagentsdk "github.com/oracle/oci-go-sdk/v65/generativeaiagent"
	generativeaiagentv1beta1 "github.com/oracle/oci-service-operator/api/generativeaiagent/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

var knowledgeBaseWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(generativeaiagentsdk.OperationStatusAccepted),
		string(generativeaiagentsdk.OperationStatusInProgress),
		string(generativeaiagentsdk.OperationStatusWaiting),
		string(generativeaiagentsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(generativeaiagentsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(generativeaiagentsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(generativeaiagentsdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(generativeaiagentsdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(generativeaiagentsdk.OperationTypeCreateKnowledgeBase)},
	UpdateActionTokens:    []string{string(generativeaiagentsdk.OperationTypeUpdateKnowledgeBase)},
	DeleteActionTokens:    []string{string(generativeaiagentsdk.OperationTypeDeleteKnowledgeBase)},
}

type knowledgeBaseOCIClient interface {
	CreateKnowledgeBase(context.Context, generativeaiagentsdk.CreateKnowledgeBaseRequest) (generativeaiagentsdk.CreateKnowledgeBaseResponse, error)
	GetKnowledgeBase(context.Context, generativeaiagentsdk.GetKnowledgeBaseRequest) (generativeaiagentsdk.GetKnowledgeBaseResponse, error)
	ListKnowledgeBases(context.Context, generativeaiagentsdk.ListKnowledgeBasesRequest) (generativeaiagentsdk.ListKnowledgeBasesResponse, error)
	UpdateKnowledgeBase(context.Context, generativeaiagentsdk.UpdateKnowledgeBaseRequest) (generativeaiagentsdk.UpdateKnowledgeBaseResponse, error)
	DeleteKnowledgeBase(context.Context, generativeaiagentsdk.DeleteKnowledgeBaseRequest) (generativeaiagentsdk.DeleteKnowledgeBaseResponse, error)
	GetWorkRequest(context.Context, generativeaiagentsdk.GetWorkRequestRequest) (generativeaiagentsdk.GetWorkRequestResponse, error)
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
	client, err := generativeaiagentsdk.NewGenerativeAiAgentClientWithConfigurationProvider(manager.Provider)
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
	hooks.BuildCreateBody = func(
		ctx context.Context,
		resource *generativeaiagentv1beta1.KnowledgeBase,
		namespace string,
	) (any, error) {
		return buildKnowledgeBaseCreateDetails(ctx, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *generativeaiagentv1beta1.KnowledgeBase,
		namespace string,
		currentResponse any,
	) (any, bool, error) {
		return buildKnowledgeBaseUpdateBody(ctx, resource, namespace, currentResponse)
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
		ServiceClient: generatedruntime.NewServiceClient[*generativeaiagentv1beta1.KnowledgeBase](
			newKnowledgeBaseRuntimeConfig(log, client),
		),
	}
}

func newKnowledgeBaseRuntimeConfig(
	log loggerutil.OSOKLogger,
	client knowledgeBaseOCIClient,
) generatedruntime.Config[*generativeaiagentv1beta1.KnowledgeBase] {
	hooks := newKnowledgeBaseRuntimeHooksWithOCIClient(client)
	applyKnowledgeBaseRuntimeHooks(&hooks, client, nil)
	return buildKnowledgeBaseGeneratedRuntimeConfig(&KnowledgeBaseServiceManager{Log: log}, hooks)
}

func newKnowledgeBaseRuntimeHooksWithOCIClient(client knowledgeBaseOCIClient) KnowledgeBaseRuntimeHooks {
	return KnowledgeBaseRuntimeHooks{
		Semantics: newKnowledgeBaseRuntimeSemantics(),
		Create: runtimeOperationHooks[generativeaiagentsdk.CreateKnowledgeBaseRequest, generativeaiagentsdk.CreateKnowledgeBaseResponse]{
			Fields: knowledgeBaseCreateFields(),
			Call: func(ctx context.Context, request generativeaiagentsdk.CreateKnowledgeBaseRequest) (generativeaiagentsdk.CreateKnowledgeBaseResponse, error) {
				return client.CreateKnowledgeBase(ctx, request)
			},
		},
		Get: runtimeOperationHooks[generativeaiagentsdk.GetKnowledgeBaseRequest, generativeaiagentsdk.GetKnowledgeBaseResponse]{
			Fields: knowledgeBaseGetFields(),
			Call: func(ctx context.Context, request generativeaiagentsdk.GetKnowledgeBaseRequest) (generativeaiagentsdk.GetKnowledgeBaseResponse, error) {
				return client.GetKnowledgeBase(ctx, request)
			},
		},
		List: runtimeOperationHooks[generativeaiagentsdk.ListKnowledgeBasesRequest, generativeaiagentsdk.ListKnowledgeBasesResponse]{
			Fields: knowledgeBaseListFields(),
			Call: func(ctx context.Context, request generativeaiagentsdk.ListKnowledgeBasesRequest) (generativeaiagentsdk.ListKnowledgeBasesResponse, error) {
				return client.ListKnowledgeBases(ctx, request)
			},
		},
		Update: runtimeOperationHooks[generativeaiagentsdk.UpdateKnowledgeBaseRequest, generativeaiagentsdk.UpdateKnowledgeBaseResponse]{
			Fields: knowledgeBaseUpdateFields(),
			Call: func(ctx context.Context, request generativeaiagentsdk.UpdateKnowledgeBaseRequest) (generativeaiagentsdk.UpdateKnowledgeBaseResponse, error) {
				return client.UpdateKnowledgeBase(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[generativeaiagentsdk.DeleteKnowledgeBaseRequest, generativeaiagentsdk.DeleteKnowledgeBaseResponse]{
			Fields: knowledgeBaseDeleteFields(),
			Call: func(ctx context.Context, request generativeaiagentsdk.DeleteKnowledgeBaseRequest) (generativeaiagentsdk.DeleteKnowledgeBaseResponse, error) {
				return client.DeleteKnowledgeBase(ctx, request)
			},
		},
	}
}

func reviewedKnowledgeBaseRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newKnowledgeBaseRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName"},
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
	resource *generativeaiagentv1beta1.KnowledgeBase,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("KnowledgeBase resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildKnowledgeBaseCreateDetails(
	ctx context.Context,
	resource *generativeaiagentv1beta1.KnowledgeBase,
	namespace string,
) (generativeaiagentsdk.CreateKnowledgeBaseDetails, error) {
	if resource == nil {
		return generativeaiagentsdk.CreateKnowledgeBaseDetails{}, fmt.Errorf("KnowledgeBase resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return generativeaiagentsdk.CreateKnowledgeBaseDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return generativeaiagentsdk.CreateKnowledgeBaseDetails{}, fmt.Errorf("marshal resolved KnowledgeBase spec: %w", err)
	}

	var details generativeaiagentsdk.CreateKnowledgeBaseDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return generativeaiagentsdk.CreateKnowledgeBaseDetails{}, fmt.Errorf("decode KnowledgeBase create request body: %w", err)
	}
	if details.IndexConfig == nil {
		return generativeaiagentsdk.CreateKnowledgeBaseDetails{}, fmt.Errorf("KnowledgeBase indexConfig must resolve to a concrete SDK body")
	}
	return details, nil
}

func buildKnowledgeBaseUpdateBody(
	ctx context.Context,
	resource *generativeaiagentv1beta1.KnowledgeBase,
	namespace string,
	currentResponse any,
) (generativeaiagentsdk.UpdateKnowledgeBaseDetails, bool, error) {
	if resource == nil {
		return generativeaiagentsdk.UpdateKnowledgeBaseDetails{}, false, fmt.Errorf("KnowledgeBase resource is nil")
	}

	current, err := knowledgeBaseFromResponse(currentResponse)
	if err != nil {
		return generativeaiagentsdk.UpdateKnowledgeBaseDetails{}, false, err
	}

	desired, err := buildKnowledgeBaseDesiredUpdateDetails(ctx, resource, namespace)
	if err != nil {
		return generativeaiagentsdk.UpdateKnowledgeBaseDetails{}, false, err
	}

	details := generativeaiagentsdk.UpdateKnowledgeBaseDetails{}
	updateNeeded := false

	if value, ok := knowledgeBaseDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = value
		updateNeeded = true
	}
	if value, ok := knowledgeBaseDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = value
		updateNeeded = true
	}
	if value, ok := knowledgeBaseDesiredIndexConfigUpdate(desired.IndexConfig, current.IndexConfig); ok {
		details.IndexConfig = value
		updateNeeded = true
	}
	if value, ok := knowledgeBaseDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = value
		updateNeeded = true
	}
	if value, ok := knowledgeBaseDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = value
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func buildKnowledgeBaseDesiredUpdateDetails(
	ctx context.Context,
	resource *generativeaiagentv1beta1.KnowledgeBase,
	namespace string,
) (generativeaiagentsdk.UpdateKnowledgeBaseDetails, error) {
	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return generativeaiagentsdk.UpdateKnowledgeBaseDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return generativeaiagentsdk.UpdateKnowledgeBaseDetails{}, fmt.Errorf("marshal resolved KnowledgeBase spec: %w", err)
	}

	var details generativeaiagentsdk.UpdateKnowledgeBaseDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return generativeaiagentsdk.UpdateKnowledgeBaseDetails{}, fmt.Errorf("decode KnowledgeBase update request body: %w", err)
	}
	if details.IndexConfig == nil {
		return generativeaiagentsdk.UpdateKnowledgeBaseDetails{}, fmt.Errorf("KnowledgeBase indexConfig must resolve to a concrete SDK body")
	}
	return details, nil
}

func knowledgeBaseFromResponse(currentResponse any) (generativeaiagentsdk.KnowledgeBase, error) {
	switch current := currentResponse.(type) {
	case generativeaiagentsdk.KnowledgeBase:
		return current, nil
	case *generativeaiagentsdk.KnowledgeBase:
		if current == nil {
			return generativeaiagentsdk.KnowledgeBase{}, fmt.Errorf("current KnowledgeBase response is nil")
		}
		return *current, nil
	case generativeaiagentsdk.KnowledgeBaseSummary:
		return generativeaiagentsdk.KnowledgeBase{
			Id:               current.Id,
			DisplayName:      current.DisplayName,
			CompartmentId:    current.CompartmentId,
			TimeCreated:      current.TimeCreated,
			LifecycleState:   current.LifecycleState,
			FreeformTags:     current.FreeformTags,
			DefinedTags:      current.DefinedTags,
			Description:      current.Description,
			TimeUpdated:      current.TimeUpdated,
			LifecycleDetails: current.LifecycleDetails,
			SystemTags:       current.SystemTags,
		}, nil
	case *generativeaiagentsdk.KnowledgeBaseSummary:
		if current == nil {
			return generativeaiagentsdk.KnowledgeBase{}, fmt.Errorf("current KnowledgeBase response is nil")
		}
		return knowledgeBaseFromResponse(*current)
	case generativeaiagentsdk.GetKnowledgeBaseResponse:
		return current.KnowledgeBase, nil
	case *generativeaiagentsdk.GetKnowledgeBaseResponse:
		if current == nil {
			return generativeaiagentsdk.KnowledgeBase{}, fmt.Errorf("current KnowledgeBase response is nil")
		}
		return current.KnowledgeBase, nil
	default:
		return generativeaiagentsdk.KnowledgeBase{}, fmt.Errorf("unexpected current KnowledgeBase response type %T", currentResponse)
	}
}

func knowledgeBaseDesiredStringUpdate(spec string, current *string) (*string, bool) {
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

func knowledgeBaseDesiredIndexConfigUpdate(
	desired generativeaiagentsdk.IndexConfig,
	current generativeaiagentsdk.IndexConfig,
) (generativeaiagentsdk.IndexConfig, bool) {
	if desired == nil {
		return nil, false
	}
	if knowledgeBaseJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
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

	response, err := client.GetWorkRequest(ctx, generativeaiagentsdk.GetWorkRequestRequest{
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
	_ *generativeaiagentv1beta1.KnowledgeBase,
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

func knowledgeBaseWorkRequestFromAny(workRequest any) (generativeaiagentsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case generativeaiagentsdk.WorkRequest:
		return current, nil
	case *generativeaiagentsdk.WorkRequest:
		if current == nil {
			return generativeaiagentsdk.WorkRequest{}, fmt.Errorf("KnowledgeBase work request is nil")
		}
		return *current, nil
	default:
		return generativeaiagentsdk.WorkRequest{}, fmt.Errorf("unexpected KnowledgeBase work request type %T", workRequest)
	}
}

func knowledgeBaseWorkRequestPhaseFromOperationType(
	operationType generativeaiagentsdk.OperationTypeEnum,
) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case generativeaiagentsdk.OperationTypeCreateKnowledgeBase:
		return shared.OSOKAsyncPhaseCreate, true
	case generativeaiagentsdk.OperationTypeUpdateKnowledgeBase:
		return shared.OSOKAsyncPhaseUpdate, true
	case generativeaiagentsdk.OperationTypeDeleteKnowledgeBase:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func knowledgeBaseWorkRequestActionForPhase(
	phase shared.OSOKAsyncPhase,
) generativeaiagentsdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return generativeaiagentsdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return generativeaiagentsdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return generativeaiagentsdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveKnowledgeBaseIDFromResources(
	resources []generativeaiagentsdk.WorkRequestResource,
	action generativeaiagentsdk.ActionTypeEnum,
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

func isKnowledgeBaseWorkRequestResource(resource generativeaiagentsdk.WorkRequestResource) bool {
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
