/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
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

var agentWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
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
	CreateActionTokens:    []string{string(generativeaiagentsdk.OperationTypeCreateAgent)},
	UpdateActionTokens:    []string{string(generativeaiagentsdk.OperationTypeUpdateAgent)},
	DeleteActionTokens:    []string{string(generativeaiagentsdk.OperationTypeDeleteAgent)},
}

type agentOCIClient interface {
	CreateAgent(context.Context, generativeaiagentsdk.CreateAgentRequest) (generativeaiagentsdk.CreateAgentResponse, error)
	GetAgent(context.Context, generativeaiagentsdk.GetAgentRequest) (generativeaiagentsdk.GetAgentResponse, error)
	ListAgents(context.Context, generativeaiagentsdk.ListAgentsRequest) (generativeaiagentsdk.ListAgentsResponse, error)
	UpdateAgent(context.Context, generativeaiagentsdk.UpdateAgentRequest) (generativeaiagentsdk.UpdateAgentResponse, error)
	DeleteAgent(context.Context, generativeaiagentsdk.DeleteAgentRequest) (generativeaiagentsdk.DeleteAgentResponse, error)
	GetWorkRequest(context.Context, generativeaiagentsdk.GetWorkRequestRequest) (generativeaiagentsdk.GetWorkRequestResponse, error)
}

func init() {
	registerAgentRuntimeHooksMutator(func(manager *AgentServiceManager, hooks *AgentRuntimeHooks) {
		client, initErr := newAgentSDKClient(manager)
		applyAgentRuntimeHooks(hooks, client, initErr)
	})
}

func newAgentSDKClient(manager *AgentServiceManager) (agentOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("Agent service manager is nil")
	}
	client, err := generativeaiagentsdk.NewGenerativeAiAgentClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyAgentRuntimeHooks(
	hooks *AgentRuntimeHooks,
	client agentOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedAgentRuntimeSemantics()
	hooks.BuildCreateBody = func(
		ctx context.Context,
		resource *generativeaiagentv1beta1.Agent,
		namespace string,
	) (any, error) {
		return buildAgentCreateDetails(ctx, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *generativeaiagentv1beta1.Agent,
		namespace string,
		currentResponse any,
	) (any, bool, error) {
		return buildAgentUpdateBody(ctx, resource, namespace, currentResponse)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardAgentExistingBeforeCreate
	hooks.Create.Fields = agentCreateFields()
	hooks.Get.Fields = agentGetFields()
	hooks.List.Fields = agentListFields()
	hooks.Update.Fields = agentUpdateFields()
	hooks.Delete.Fields = agentDeleteFields()
	hooks.Async.Adapter = agentWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getAgentWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveAgentGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveAgentGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverAgentIDFromGeneratedWorkRequest
	hooks.Async.Message = agentGeneratedWorkRequestMessage
}

func newAgentServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client agentOCIClient,
) AgentServiceClient {
	return defaultAgentServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*generativeaiagentv1beta1.Agent](
			newAgentRuntimeConfig(log, client),
		),
	}
}

func newAgentRuntimeConfig(
	log loggerutil.OSOKLogger,
	client agentOCIClient,
) generatedruntime.Config[*generativeaiagentv1beta1.Agent] {
	hooks := newAgentRuntimeHooksWithOCIClient(client)
	applyAgentRuntimeHooks(&hooks, client, nil)
	return buildAgentGeneratedRuntimeConfig(&AgentServiceManager{Log: log}, hooks)
}

func newAgentRuntimeHooksWithOCIClient(client agentOCIClient) AgentRuntimeHooks {
	return AgentRuntimeHooks{
		Semantics: newAgentRuntimeSemantics(),
		Create: runtimeOperationHooks[generativeaiagentsdk.CreateAgentRequest, generativeaiagentsdk.CreateAgentResponse]{
			Fields: agentCreateFields(),
			Call: func(ctx context.Context, request generativeaiagentsdk.CreateAgentRequest) (generativeaiagentsdk.CreateAgentResponse, error) {
				return client.CreateAgent(ctx, request)
			},
		},
		Get: runtimeOperationHooks[generativeaiagentsdk.GetAgentRequest, generativeaiagentsdk.GetAgentResponse]{
			Fields: agentGetFields(),
			Call: func(ctx context.Context, request generativeaiagentsdk.GetAgentRequest) (generativeaiagentsdk.GetAgentResponse, error) {
				return client.GetAgent(ctx, request)
			},
		},
		List: runtimeOperationHooks[generativeaiagentsdk.ListAgentsRequest, generativeaiagentsdk.ListAgentsResponse]{
			Fields: agentListFields(),
			Call: func(ctx context.Context, request generativeaiagentsdk.ListAgentsRequest) (generativeaiagentsdk.ListAgentsResponse, error) {
				return client.ListAgents(ctx, request)
			},
		},
		Update: runtimeOperationHooks[generativeaiagentsdk.UpdateAgentRequest, generativeaiagentsdk.UpdateAgentResponse]{
			Fields: agentUpdateFields(),
			Call: func(ctx context.Context, request generativeaiagentsdk.UpdateAgentRequest) (generativeaiagentsdk.UpdateAgentResponse, error) {
				return client.UpdateAgent(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[generativeaiagentsdk.DeleteAgentRequest, generativeaiagentsdk.DeleteAgentResponse]{
			Fields: agentDeleteFields(),
			Call: func(ctx context.Context, request generativeaiagentsdk.DeleteAgentRequest) (generativeaiagentsdk.DeleteAgentResponse, error) {
				return client.DeleteAgent(ctx, request)
			},
		},
	}
}

func reviewedAgentRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newAgentRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName"},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func agentCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateAgentDetails", RequestName: "CreateAgentDetails", Contribution: "body"},
	}
}

func agentGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AgentId", RequestName: "agentId", Contribution: "path", PreferResourceID: true},
	}
}

func agentListFields() []generatedruntime.RequestField {
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

func agentUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AgentId", RequestName: "agentId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateAgentDetails", RequestName: "UpdateAgentDetails", Contribution: "body"},
	}
}

func agentDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "AgentId", RequestName: "agentId", Contribution: "path", PreferResourceID: true},
	}
}

func guardAgentExistingBeforeCreate(
	_ context.Context,
	resource *generativeaiagentv1beta1.Agent,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("Agent resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildAgentCreateDetails(
	ctx context.Context,
	resource *generativeaiagentv1beta1.Agent,
	namespace string,
) (generativeaiagentsdk.CreateAgentDetails, error) {
	if resource == nil {
		return generativeaiagentsdk.CreateAgentDetails{}, fmt.Errorf("Agent resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return generativeaiagentsdk.CreateAgentDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return generativeaiagentsdk.CreateAgentDetails{}, fmt.Errorf("marshal resolved Agent spec: %w", err)
	}

	var details generativeaiagentsdk.CreateAgentDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return generativeaiagentsdk.CreateAgentDetails{}, fmt.Errorf("decode Agent create request body: %w", err)
	}

	llmConfig, err := buildAgentLlmConfig(resource.Spec.LlmConfig)
	if err != nil {
		return generativeaiagentsdk.CreateAgentDetails{}, err
	}
	details.LlmConfig = llmConfig

	return details, nil
}

func buildAgentUpdateBody(
	ctx context.Context,
	resource *generativeaiagentv1beta1.Agent,
	namespace string,
	currentResponse any,
) (generativeaiagentsdk.UpdateAgentDetails, bool, error) {
	if resource == nil {
		return generativeaiagentsdk.UpdateAgentDetails{}, false, fmt.Errorf("Agent resource is nil")
	}

	current, err := agentFromResponse(currentResponse)
	if err != nil {
		return generativeaiagentsdk.UpdateAgentDetails{}, false, err
	}

	desired, err := buildAgentDesiredUpdateDetails(ctx, resource, namespace)
	if err != nil {
		return generativeaiagentsdk.UpdateAgentDetails{}, false, err
	}

	details := generativeaiagentsdk.UpdateAgentDetails{}
	updateNeeded := false

	if value, ok := agentDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = value
		updateNeeded = true
	}
	if value, ok := agentDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = value
		updateNeeded = true
	}
	if value, ok := agentDesiredStringUpdate(resource.Spec.WelcomeMessage, current.WelcomeMessage); ok {
		details.WelcomeMessage = value
		updateNeeded = true
	}
	if value, ok := agentDesiredKnowledgeBaseIDsUpdate(resource.Spec.KnowledgeBaseIds, current.KnowledgeBaseIds); ok {
		details.KnowledgeBaseIds = value
		updateNeeded = true
	}
	if value, ok := agentDesiredLlmConfigUpdate(desired.LlmConfig, current.LlmConfig); ok {
		details.LlmConfig = value
		updateNeeded = true
	}
	if value, ok := agentDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = value
		updateNeeded = true
	}
	if value, ok := agentDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = value
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func buildAgentDesiredUpdateDetails(
	ctx context.Context,
	resource *generativeaiagentv1beta1.Agent,
	namespace string,
) (generativeaiagentsdk.UpdateAgentDetails, error) {
	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return generativeaiagentsdk.UpdateAgentDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return generativeaiagentsdk.UpdateAgentDetails{}, fmt.Errorf("marshal resolved Agent spec: %w", err)
	}

	var details generativeaiagentsdk.UpdateAgentDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return generativeaiagentsdk.UpdateAgentDetails{}, fmt.Errorf("decode Agent update request body: %w", err)
	}

	llmConfig, err := buildAgentLlmConfig(resource.Spec.LlmConfig)
	if err != nil {
		return generativeaiagentsdk.UpdateAgentDetails{}, err
	}
	details.LlmConfig = llmConfig

	return details, nil
}

func buildAgentLlmConfig(
	spec generativeaiagentv1beta1.AgentLlmConfig,
) (*generativeaiagentsdk.LlmConfig, error) {
	customization, err := buildAgentLlmCustomization(spec.RoutingLlmCustomization)
	if err != nil {
		return nil, fmt.Errorf("build Agent llmConfig.routingLlmCustomization: %w", err)
	}

	if customization == nil && strings.TrimSpace(spec.RuntimeVersion) == "" {
		return nil, nil
	}

	config := &generativeaiagentsdk.LlmConfig{
		RoutingLlmCustomization: customization,
	}
	if strings.TrimSpace(spec.RuntimeVersion) != "" {
		config.RuntimeVersion = common.String(spec.RuntimeVersion)
	}
	return config, nil
}

func buildAgentLlmCustomization(
	spec generativeaiagentv1beta1.AgentLlmConfigRoutingLlmCustomization,
) (*generativeaiagentsdk.LlmCustomization, error) {
	selection, err := buildAgentLlmSelection(spec.LlmSelection)
	if err != nil {
		return nil, err
	}

	hyperParameters, err := agentLlmHyperParametersFromSpec(spec.LlmHyperParameters)
	if err != nil {
		return nil, err
	}

	if selection == nil && len(hyperParameters) == 0 && strings.TrimSpace(spec.Instruction) == "" {
		return nil, nil
	}

	customization := &generativeaiagentsdk.LlmCustomization{
		LlmSelection:       selection,
		LlmHyperParameters: hyperParameters,
	}
	if strings.TrimSpace(spec.Instruction) != "" {
		customization.Instruction = common.String(spec.Instruction)
	}
	return customization, nil
}

func buildAgentLlmSelection(
	spec generativeaiagentv1beta1.AgentLlmConfigRoutingLlmCustomizationLlmSelection,
) (generativeaiagentsdk.LlmSelection, error) {
	payload, err := agentLlmSelectionPayload(spec)
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, nil
	}

	selectionType, err := agentJSONFieldString(payload, "llmSelectionType")
	if err != nil {
		return nil, fmt.Errorf("decode Agent llmSelectionType: %w", err)
	}
	switch strings.ToUpper(strings.TrimSpace(selectionType)) {
	case string(generativeaiagentsdk.LlmSelectionLlmSelectionTypeDefault):
		var selection generativeaiagentsdk.DefaultLlmSelection
		if err := json.Unmarshal(payload, &selection); err != nil {
			return nil, fmt.Errorf("decode Agent DEFAULT llmSelection: %w", err)
		}
		return selection, nil
	case string(generativeaiagentsdk.LlmSelectionLlmSelectionTypeCustomGenAiEndpoint):
		var selection generativeaiagentsdk.CustomGenAiEndpointLlmSelection
		if err := json.Unmarshal(payload, &selection); err != nil {
			return nil, fmt.Errorf("decode Agent CUSTOM_GEN_AI_ENDPOINT llmSelection: %w", err)
		}
		if strings.TrimSpace(stringValue(selection.EndpointId)) == "" {
			return nil, fmt.Errorf("Agent CUSTOM_GEN_AI_ENDPOINT llmSelection requires endpointId")
		}
		return selection, nil
	case string(generativeaiagentsdk.LlmSelectionLlmSelectionTypeCustomGenAiModel):
		var selection generativeaiagentsdk.CustomGenAiModelLlmSelection
		if err := json.Unmarshal(payload, &selection); err != nil {
			return nil, fmt.Errorf("decode Agent CUSTOM_GEN_AI_MODEL llmSelection: %w", err)
		}
		if strings.TrimSpace(stringValue(selection.ModelId)) == "" {
			return nil, fmt.Errorf("Agent CUSTOM_GEN_AI_MODEL llmSelection requires modelId")
		}
		return selection, nil
	case "":
		return nil, fmt.Errorf("Agent llmSelectionType is required")
	default:
		return nil, fmt.Errorf("unsupported Agent llmSelectionType %q", selectionType)
	}
}

func agentLlmSelectionPayload(
	spec generativeaiagentv1beta1.AgentLlmConfigRoutingLlmCustomizationLlmSelection,
) ([]byte, error) {
	if raw := strings.TrimSpace(spec.JsonData); raw != "" {
		conflicts := agentLlmSelectionConflicts(spec)
		if len(conflicts) > 0 {
			return nil, fmt.Errorf("Agent llmSelection.jsonData conflicts with typed field(s): %s", strings.Join(conflicts, ", "))
		}
		return []byte(raw), nil
	}

	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal Agent llmSelection: %w", err)
	}
	if strings.TrimSpace(string(payload)) == "{}" {
		return nil, nil
	}
	return payload, nil
}

func agentLlmSelectionConflicts(
	spec generativeaiagentv1beta1.AgentLlmConfigRoutingLlmCustomizationLlmSelection,
) []string {
	var conflicts []string
	if strings.TrimSpace(spec.LlmSelectionType) != "" {
		conflicts = append(conflicts, "llmSelectionType")
	}
	if strings.TrimSpace(spec.EndpointId) != "" {
		conflicts = append(conflicts, "endpointId")
	}
	if strings.TrimSpace(spec.ModelId) != "" {
		conflicts = append(conflicts, "modelId")
	}
	sort.Strings(conflicts)
	return conflicts
}

func agentLlmHyperParametersFromSpec(
	spec map[string]shared.JSONValue,
) (map[string]interface{}, error) {
	if spec == nil {
		return nil, nil
	}

	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal Agent llmHyperParameters: %w", err)
	}

	var values map[string]interface{}
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, fmt.Errorf("decode Agent llmHyperParameters: %w", err)
	}
	if len(values) == 0 {
		return nil, nil
	}
	return values, nil
}

func agentJSONFieldString(payload []byte, key string) (string, error) {
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return "", err
	}
	raw, ok := decoded[key]
	if !ok || raw == nil {
		return "", nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%s is %T, want string", key, raw)
	}
	return value, nil
}

func agentFromResponse(currentResponse any) (generativeaiagentsdk.Agent, error) {
	switch current := currentResponse.(type) {
	case generativeaiagentsdk.Agent:
		return current, nil
	case *generativeaiagentsdk.Agent:
		if current == nil {
			return generativeaiagentsdk.Agent{}, fmt.Errorf("current Agent response is nil")
		}
		return *current, nil
	case generativeaiagentsdk.AgentSummary:
		return generativeaiagentsdk.Agent{
			Id:               current.Id,
			DisplayName:      current.DisplayName,
			CompartmentId:    current.CompartmentId,
			TimeCreated:      current.TimeCreated,
			LifecycleState:   current.LifecycleState,
			FreeformTags:     current.FreeformTags,
			DefinedTags:      current.DefinedTags,
			Description:      current.Description,
			KnowledgeBaseIds: current.KnowledgeBaseIds,
			WelcomeMessage:   current.WelcomeMessage,
			LlmConfig:        current.LlmConfig,
			TimeUpdated:      current.TimeUpdated,
			LifecycleDetails: current.LifecycleDetails,
			SystemTags:       current.SystemTags,
		}, nil
	case *generativeaiagentsdk.AgentSummary:
		if current == nil {
			return generativeaiagentsdk.Agent{}, fmt.Errorf("current Agent response is nil")
		}
		return agentFromResponse(*current)
	case generativeaiagentsdk.GetAgentResponse:
		return current.Agent, nil
	case *generativeaiagentsdk.GetAgentResponse:
		if current == nil {
			return generativeaiagentsdk.Agent{}, fmt.Errorf("current Agent response is nil")
		}
		return current.Agent, nil
	default:
		return generativeaiagentsdk.Agent{}, fmt.Errorf("unexpected current Agent response type %T", currentResponse)
	}
}

func agentDesiredStringUpdate(spec string, current *string) (*string, bool) {
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

func agentDesiredKnowledgeBaseIDsUpdate(
	spec []string,
	current []string,
) ([]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if slices.Equal(spec, current) {
		return nil, false
	}
	return slices.Clone(spec), true
}

func agentDesiredLlmConfigUpdate(
	desired *generativeaiagentsdk.LlmConfig,
	current *generativeaiagentsdk.LlmConfig,
) (*generativeaiagentsdk.LlmConfig, bool) {
	if desired == nil {
		return nil, false
	}
	if current == nil {
		return desired, true
	}
	if agentJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func agentDesiredFreeformTagsUpdate(
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

func agentDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := agentDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if agentJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func agentDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
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

func agentJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func getAgentWorkRequest(
	ctx context.Context,
	client agentOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize Agent OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("Agent OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, generativeaiagentsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveAgentGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := agentWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveAgentGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := agentWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := agentWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverAgentIDFromGeneratedWorkRequest(
	_ *generativeaiagentv1beta1.Agent,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := agentWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := agentWorkRequestActionForPhase(phase)
	if id, ok := resolveAgentIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveAgentIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("Agent work request %s does not expose an agent identifier", stringValue(current.Id))
}

func agentGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := agentWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("Agent %s work request %s is %s", phase, stringValue(current.Id), current.Status)
}

func agentWorkRequestFromAny(workRequest any) (generativeaiagentsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case generativeaiagentsdk.WorkRequest:
		return current, nil
	case *generativeaiagentsdk.WorkRequest:
		if current == nil {
			return generativeaiagentsdk.WorkRequest{}, fmt.Errorf("Agent work request is nil")
		}
		return *current, nil
	default:
		return generativeaiagentsdk.WorkRequest{}, fmt.Errorf("unexpected Agent work request type %T", workRequest)
	}
}

func agentWorkRequestPhaseFromOperationType(
	operationType generativeaiagentsdk.OperationTypeEnum,
) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case generativeaiagentsdk.OperationTypeCreateAgent:
		return shared.OSOKAsyncPhaseCreate, true
	case generativeaiagentsdk.OperationTypeUpdateAgent:
		return shared.OSOKAsyncPhaseUpdate, true
	case generativeaiagentsdk.OperationTypeDeleteAgent:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func agentWorkRequestActionForPhase(
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

func resolveAgentIDFromResources(
	resources []generativeaiagentsdk.WorkRequestResource,
	action generativeaiagentsdk.ActionTypeEnum,
	preferAgentOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferAgentOnly && !isAgentWorkRequestResource(resource) {
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

func isAgentWorkRequestResource(resource generativeaiagentsdk.WorkRequestResource) bool {
	return normalizeAgentWorkRequestToken(stringValue(resource.EntityType)) == "agent"
}

func normalizeAgentWorkRequestToken(value string) string {
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
