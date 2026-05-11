/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package agent

import (
	"context"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	generativeaiagentsdk "github.com/oracle/oci-go-sdk/v65/generativeaiagent"
	generativeaiagentv1beta1 "github.com/oracle/oci-service-operator/api/generativeaiagent/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeAgentOCIClient struct {
	createFn      func(context.Context, generativeaiagentsdk.CreateAgentRequest) (generativeaiagentsdk.CreateAgentResponse, error)
	getFn         func(context.Context, generativeaiagentsdk.GetAgentRequest) (generativeaiagentsdk.GetAgentResponse, error)
	listFn        func(context.Context, generativeaiagentsdk.ListAgentsRequest) (generativeaiagentsdk.ListAgentsResponse, error)
	updateFn      func(context.Context, generativeaiagentsdk.UpdateAgentRequest) (generativeaiagentsdk.UpdateAgentResponse, error)
	deleteFn      func(context.Context, generativeaiagentsdk.DeleteAgentRequest) (generativeaiagentsdk.DeleteAgentResponse, error)
	workRequestFn func(context.Context, generativeaiagentsdk.GetWorkRequestRequest) (generativeaiagentsdk.GetWorkRequestResponse, error)
}

func (f *fakeAgentOCIClient) CreateAgent(
	ctx context.Context,
	req generativeaiagentsdk.CreateAgentRequest,
) (generativeaiagentsdk.CreateAgentResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return generativeaiagentsdk.CreateAgentResponse{}, nil
}

func (f *fakeAgentOCIClient) GetAgent(
	ctx context.Context,
	req generativeaiagentsdk.GetAgentRequest,
) (generativeaiagentsdk.GetAgentResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return generativeaiagentsdk.GetAgentResponse{}, errortest.NewServiceError(404, "NotFound", "missing")
}

func (f *fakeAgentOCIClient) ListAgents(
	ctx context.Context,
	req generativeaiagentsdk.ListAgentsRequest,
) (generativeaiagentsdk.ListAgentsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return generativeaiagentsdk.ListAgentsResponse{}, nil
}

func (f *fakeAgentOCIClient) UpdateAgent(
	ctx context.Context,
	req generativeaiagentsdk.UpdateAgentRequest,
) (generativeaiagentsdk.UpdateAgentResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return generativeaiagentsdk.UpdateAgentResponse{}, nil
}

func (f *fakeAgentOCIClient) DeleteAgent(
	ctx context.Context,
	req generativeaiagentsdk.DeleteAgentRequest,
) (generativeaiagentsdk.DeleteAgentResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return generativeaiagentsdk.DeleteAgentResponse{}, nil
}

func (f *fakeAgentOCIClient) GetWorkRequest(
	ctx context.Context,
	req generativeaiagentsdk.GetWorkRequestRequest,
) (generativeaiagentsdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return generativeaiagentsdk.GetWorkRequestResponse{}, nil
}

func TestReviewedAgentRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedAgentRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedAgentRuntimeSemantics() = nil")
	}

	if got.FormalService != "generativeaiagent" {
		t.Fatalf("FormalService = %q, want generativeaiagent", got.FormalService)
	}
	if got.FormalSlug != "agent" {
		t.Fatalf("FormalSlug = %q, want agent", got.FormalSlug)
	}
	if got.Async == nil {
		t.Fatal("Async = nil, want workrequest semantics")
	}
	if got.Async.Strategy != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got.Async.Strategy)
	}
	if got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async.Runtime = %q, want generatedruntime", got.Async.Runtime)
	}
	if got.Async.WorkRequest == nil {
		t.Fatal("Async.WorkRequest = nil")
	}
	assertAgentStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertAgentStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertAgentStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertAgentStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertAgentStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertAgentStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertAgentStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName"})
	assertAgentStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"definedTags", "description", "displayName", "freeformTags", "knowledgeBaseIds", "llmConfig", "welcomeMessage"})
	assertAgentStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetAgent" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want workrequest-backed create", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetAgent" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want workrequest-backed update", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetWorkRequest -> GetAgent/ListAgents confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want workrequest-backed confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want none for published runtime", got.AuxiliaryOperations)
	}
}

func TestBuildAgentCreateDetailsPreservesConcreteLlmConfig(t *testing.T) {
	t.Parallel()

	resource := makeAgentResource()

	details, err := buildAgentCreateDetails(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildAgentCreateDetails() error = %v", err)
	}

	requireAgentStringPtr(t, "details.compartmentId", details.CompartmentId, resource.Spec.CompartmentId)
	requireAgentStringPtr(t, "details.displayName", details.DisplayName, resource.Spec.DisplayName)
	requireAgentStringPtr(t, "details.description", details.Description, resource.Spec.Description)
	requireAgentStringPtr(t, "details.welcomeMessage", details.WelcomeMessage, resource.Spec.WelcomeMessage)
	requireAgentStringSliceEqual(t, "details.knowledgeBaseIds", details.KnowledgeBaseIds, resource.Spec.KnowledgeBaseIds)
	requireAgentLlmConfigDefaultSelection(t, "details.llmConfig", details.LlmConfig, resource.Spec.LlmConfig)
}

func TestBuildAgentCreateDetailsOmitsEmptyLlmConfig(t *testing.T) {
	t.Parallel()

	resource := makeAgentResource()
	resource.Spec.LlmConfig = generativeaiagentv1beta1.AgentLlmConfig{}

	details, err := buildAgentCreateDetails(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("buildAgentCreateDetails() error = %v", err)
	}
	if details.LlmConfig != nil {
		t.Fatalf("details.LlmConfig = %#v, want nil when spec.llmConfig is empty", details.LlmConfig)
	}
}

func TestBuildAgentUpdateBodyPreservesClearsAndLlmConfigChanges(t *testing.T) {
	t.Parallel()

	currentResource := makeAgentResource()
	currentResource.Spec.Description = "current description"
	currentResource.Spec.WelcomeMessage = "current welcome message"
	currentResource.Spec.KnowledgeBaseIds = []string{"ocid1.knowledgebase.oc1..current"}

	desired := makeAgentResource()
	desired.Spec.Description = ""
	desired.Spec.WelcomeMessage = ""
	desired.Spec.KnowledgeBaseIds = []string{}
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}
	desired.Spec.LlmConfig = generativeaiagentv1beta1.AgentLlmConfig{
		RuntimeVersion: "2024.06.01",
		RoutingLlmCustomization: generativeaiagentv1beta1.AgentLlmConfigRoutingLlmCustomization{
			Instruction: "Use the custom model for routed answers.",
			LlmSelection: generativeaiagentv1beta1.AgentLlmConfigRoutingLlmCustomizationLlmSelection{
				LlmSelectionType: "CUSTOM_GEN_AI_MODEL",
				ModelId:          "ocid1.generativeaimodel.oc1..custom",
			},
			LlmHyperParameters: map[string]shared.JSONValue{
				"temperature": jsonValue("0.1"),
			},
		},
	}

	body, updateNeeded, err := buildAgentUpdateBody(
		context.Background(),
		desired,
		desired.Namespace,
		generativeaiagentsdk.GetAgentResponse{
			Agent: makeSDKAgent(t, "ocid1.agent.oc1..existing", currentResource, generativeaiagentsdk.AgentLifecycleStateActive),
		},
	)
	if err != nil {
		t.Fatalf("buildAgentUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildAgentUpdateBody() updateNeeded = false, want true")
	}

	requireAgentStringPtr(t, "details.description", body.Description, "")
	requireAgentStringPtr(t, "details.welcomeMessage", body.WelcomeMessage, "")
	requireAgentStringSliceEqual(t, "details.knowledgeBaseIds", body.KnowledgeBaseIds, []string{})
	if len(body.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map for clear", body.FreeformTags)
	}
	if len(body.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map for clear", body.DefinedTags)
	}
	requireAgentLlmConfigCustomModelSelection(t, "details.llmConfig", body.LlmConfig, desired.Spec.LlmConfig)
}

func TestAgentCreateOrUpdateSkipsReuseWhenDisplayNameMissing(t *testing.T) {
	t.Parallel()

	resource := makeAgentResource()
	resource.Spec.DisplayName = ""

	const (
		createdID     = "ocid1.agent.oc1..created"
		workRequestID = "wr-agent-create-empty-name"
	)

	listCalls := 0
	createCalls := 0

	client := newTestAgentClient(&fakeAgentOCIClient{
		listFn: func(_ context.Context, _ generativeaiagentsdk.ListAgentsRequest) (generativeaiagentsdk.ListAgentsResponse, error) {
			listCalls++
			return generativeaiagentsdk.ListAgentsResponse{}, nil
		},
		createFn: func(_ context.Context, req generativeaiagentsdk.CreateAgentRequest) (generativeaiagentsdk.CreateAgentResponse, error) {
			createCalls++
			requireAgentStringPtr(t, "create compartmentId", req.CreateAgentDetails.CompartmentId, resource.Spec.CompartmentId)
			if req.CreateAgentDetails.DisplayName != nil {
				t.Fatalf("create displayName = %v, want nil when spec.displayName is empty", req.CreateAgentDetails.DisplayName)
			}
			return generativeaiagentsdk.CreateAgentResponse{
				Agent:            makeSDKAgent(t, createdID, resource, generativeaiagentsdk.AgentLifecycleStateCreating),
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-agent-empty-name"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req generativeaiagentsdk.GetWorkRequestRequest) (generativeaiagentsdk.GetWorkRequestResponse, error) {
			requireAgentStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return generativeaiagentsdk.GetWorkRequestResponse{
				WorkRequest: makeAgentWorkRequest(
					workRequestID,
					generativeaiagentsdk.OperationTypeCreateAgent,
					generativeaiagentsdk.OperationStatusInProgress,
					generativeaiagentsdk.ActionTypeInProgress,
					createdID,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful pending create", response)
	}
	if listCalls != 0 {
		t.Fatalf("ListAgents() calls = %d, want 0 when displayName is empty", listCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreateAgent() calls = %d, want 1", createCalls)
	}
	requireAgentAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q from create response body", got, createdID)
	}
}

func TestAgentCreateOrUpdateRejectsAmbiguousDisplayNameReuse(t *testing.T) {
	t.Parallel()

	resource := makeAgentResource()
	createCalls := 0

	client := newTestAgentClient(&fakeAgentOCIClient{
		listFn: func(_ context.Context, req generativeaiagentsdk.ListAgentsRequest) (generativeaiagentsdk.ListAgentsResponse, error) {
			requireAgentStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireAgentStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return generativeaiagentsdk.ListAgentsResponse{
				AgentCollection: generativeaiagentsdk.AgentCollection{
					Items: []generativeaiagentsdk.AgentSummary{
						makeSDKAgentSummary(t, "ocid1.agent.oc1..first", resource, generativeaiagentsdk.AgentLifecycleStateActive),
						makeSDKAgentSummary(t, "ocid1.agent.oc1..second", resource, generativeaiagentsdk.AgentLifecycleStateUpdating),
					},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ generativeaiagentsdk.CreateAgentRequest) (generativeaiagentsdk.CreateAgentResponse, error) {
			createCalls++
			return generativeaiagentsdk.CreateAgentResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want ambiguous list match failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful result", response)
	}
	if !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate match failure", err)
	}
	if createCalls != 0 {
		t.Fatalf("CreateAgent() calls = %d, want 0 on ambiguous reuse", createCalls)
	}
}

func TestAgentServiceClientCreatesAndResumesWorkRequest(t *testing.T) {
	t.Parallel()

	const (
		createdID     = "ocid1.agent.oc1..created"
		workRequestID = "wr-agent-create"
	)

	resource := makeAgentResource()
	workRequests := map[string]generativeaiagentsdk.WorkRequest{
		workRequestID: makeAgentWorkRequest(
			workRequestID,
			generativeaiagentsdk.OperationTypeCreateAgent,
			generativeaiagentsdk.OperationStatusInProgress,
			generativeaiagentsdk.ActionTypeInProgress,
			createdID,
		),
	}

	var createRequest generativeaiagentsdk.CreateAgentRequest
	getCalls := 0

	client := newTestAgentClient(&fakeAgentOCIClient{
		listFn: func(_ context.Context, req generativeaiagentsdk.ListAgentsRequest) (generativeaiagentsdk.ListAgentsResponse, error) {
			requireAgentStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireAgentStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return generativeaiagentsdk.ListAgentsResponse{}, nil
		},
		createFn: func(_ context.Context, req generativeaiagentsdk.CreateAgentRequest) (generativeaiagentsdk.CreateAgentResponse, error) {
			createRequest = req
			return generativeaiagentsdk.CreateAgentResponse{
				Agent:            makeSDKAgent(t, createdID, resource, generativeaiagentsdk.AgentLifecycleStateCreating),
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-agent"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req generativeaiagentsdk.GetWorkRequestRequest) (generativeaiagentsdk.GetWorkRequestResponse, error) {
			requireAgentStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return generativeaiagentsdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
		getFn: func(_ context.Context, req generativeaiagentsdk.GetAgentRequest) (generativeaiagentsdk.GetAgentResponse, error) {
			getCalls++
			requireAgentStringPtr(t, "get agentId", req.AgentId, createdID)
			return generativeaiagentsdk.GetAgentResponse{
				Agent: makeSDKAgent(t, createdID, resource, generativeaiagentsdk.AgentLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful pending create", response)
	}
	requireAgentStringPtr(t, "create compartmentId", createRequest.CreateAgentDetails.CompartmentId, resource.Spec.CompartmentId)
	requireAgentStringPtr(t, "create displayName", createRequest.CreateAgentDetails.DisplayName, resource.Spec.DisplayName)
	requireAgentStringPtr(t, "create welcomeMessage", createRequest.CreateAgentDetails.WelcomeMessage, resource.Spec.WelcomeMessage)
	requireAgentLlmConfigDefaultSelection(t, "create llmConfig", createRequest.CreateAgentDetails.LlmConfig, resource.Spec.LlmConfig)
	if getCalls != 0 {
		t.Fatalf("GetAgent() calls = %d, want 0 while work request is pending", getCalls)
	}
	requireAgentAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q from create response body", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(generativeaiagentsdk.AgentLifecycleStateCreating) {
		t.Fatalf("status.lifecycleState = %q, want CREATING", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-agent" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-agent", got)
	}

	workRequests[workRequestID] = makeAgentWorkRequest(
		workRequestID,
		generativeaiagentsdk.OperationTypeCreateAgent,
		generativeaiagentsdk.OperationStatusSucceeded,
		generativeaiagentsdk.ActionTypeCreated,
		createdID,
	)

	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() after work request success error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() after work request success response = %#v, want converged success", response)
	}
	if getCalls != 1 {
		t.Fatalf("GetAgent() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(generativeaiagentsdk.AgentLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared", resource.Status.OsokStatus.Async.Current)
	}
}

func newTestAgentClient(client *fakeAgentOCIClient) AgentServiceClient {
	if client == nil {
		client = &fakeAgentOCIClient{}
	}
	return newAgentServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeAgentResource() *generativeaiagentv1beta1.Agent {
	return &generativeaiagentv1beta1.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent-sample",
			Namespace: "default",
		},
		Spec: generativeaiagentv1beta1.AgentSpec{
			CompartmentId: "ocid1.compartment.oc1..agentexample",
			DisplayName:   "agent-sample",
			Description:   "agent description",
			KnowledgeBaseIds: []string{
				"ocid1.knowledgebase.oc1..kb1",
			},
			WelcomeMessage: "Hello from the published agent runtime.",
			LlmConfig: generativeaiagentv1beta1.AgentLlmConfig{
				RuntimeVersion: "2024.05.31",
				RoutingLlmCustomization: generativeaiagentv1beta1.AgentLlmConfigRoutingLlmCustomization{
					Instruction: "Answer from the attached knowledge base first.",
					LlmSelection: generativeaiagentv1beta1.AgentLlmConfigRoutingLlmCustomizationLlmSelection{
						LlmSelectionType: "DEFAULT",
					},
					LlmHyperParameters: map[string]shared.JSONValue{
						"temperature":      jsonValue("0.25"),
						"useKnowledgeBase": jsonValue("false"),
					},
				},
			},
			FreeformTags: map[string]string{
				"environment": "dev",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func makeSDKAgent(
	t *testing.T,
	id string,
	resource *generativeaiagentv1beta1.Agent,
	state generativeaiagentsdk.AgentLifecycleStateEnum,
) generativeaiagentsdk.Agent {
	t.Helper()

	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	llmConfig, err := buildAgentLlmConfig(resource.Spec.LlmConfig)
	if err != nil {
		t.Fatalf("buildAgentLlmConfig() error = %v", err)
	}
	return generativeaiagentsdk.Agent{
		Id:               common.String(id),
		DisplayName:      optionalAgentString(resource.Spec.DisplayName),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		TimeCreated:      &now,
		LifecycleState:   state,
		FreeformTags:     maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:      sdkDefinedTags(resource.Spec.DefinedTags),
		Description:      optionalAgentString(resource.Spec.Description),
		KnowledgeBaseIds: slices.Clone(resource.Spec.KnowledgeBaseIds),
		WelcomeMessage:   optionalAgentString(resource.Spec.WelcomeMessage),
		LlmConfig:        llmConfig,
		TimeUpdated:      &now,
		LifecycleDetails: common.String("lifecycle detail"),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeSDKAgentSummary(
	t *testing.T,
	id string,
	resource *generativeaiagentv1beta1.Agent,
	state generativeaiagentsdk.AgentLifecycleStateEnum,
) generativeaiagentsdk.AgentSummary {
	t.Helper()

	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	llmConfig, err := buildAgentLlmConfig(resource.Spec.LlmConfig)
	if err != nil {
		t.Fatalf("buildAgentLlmConfig() error = %v", err)
	}
	return generativeaiagentsdk.AgentSummary{
		Id:               common.String(id),
		DisplayName:      optionalAgentString(resource.Spec.DisplayName),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		TimeCreated:      &now,
		LifecycleState:   state,
		FreeformTags:     maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:      sdkDefinedTags(resource.Spec.DefinedTags),
		Description:      optionalAgentString(resource.Spec.Description),
		KnowledgeBaseIds: slices.Clone(resource.Spec.KnowledgeBaseIds),
		WelcomeMessage:   optionalAgentString(resource.Spec.WelcomeMessage),
		LlmConfig:        llmConfig,
		TimeUpdated:      &now,
		LifecycleDetails: common.String("lifecycle detail"),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeAgentWorkRequest(
	id string,
	operation generativeaiagentsdk.OperationTypeEnum,
	status generativeaiagentsdk.OperationStatusEnum,
	action generativeaiagentsdk.ActionTypeEnum,
	resourceID string,
) generativeaiagentsdk.WorkRequest {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	percentComplete := float32(50)
	return generativeaiagentsdk.WorkRequest{
		OperationType:   operation,
		Status:          status,
		Id:              common.String(id),
		CompartmentId:   common.String("ocid1.compartment.oc1..agentexample"),
		Resources:       []generativeaiagentsdk.WorkRequestResource{{EntityType: common.String("Agent"), ActionType: action, Identifier: common.String(resourceID)}},
		PercentComplete: &percentComplete,
		TimeAccepted:    &now,
	}
}

func sdkDefinedTags(input map[string]shared.MapValue) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	out := make(map[string]map[string]interface{}, len(input))
	for namespace, values := range input {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		out[namespace] = converted
	}
	return out
}

func optionalAgentString(value string) *string {
	if value == "" {
		return nil
	}
	return common.String(value)
}

func jsonValue(raw string) shared.JSONValue {
	return shared.JSONValue{Raw: []byte(raw)}
}

func assertAgentStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func requireAgentStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", name, got, want)
	}
}

func requireAgentStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func requireAgentLlmConfigDefaultSelection(
	t *testing.T,
	name string,
	got *generativeaiagentsdk.LlmConfig,
	want generativeaiagentv1beta1.AgentLlmConfig,
) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want populated llmConfig", name)
	}
	requireAgentStringPtr(t, name+".runtimeVersion", got.RuntimeVersion, want.RuntimeVersion)
	if got.RoutingLlmCustomization == nil {
		t.Fatalf("%s.routingLlmCustomization = nil, want populated customization", name)
	}
	requireAgentStringPtr(t, name+".instruction", got.RoutingLlmCustomization.Instruction, want.RoutingLlmCustomization.Instruction)
	if _, ok := got.RoutingLlmCustomization.LlmSelection.(generativeaiagentsdk.DefaultLlmSelection); !ok {
		t.Fatalf("%s.llmSelection type = %T, want DefaultLlmSelection", name, got.RoutingLlmCustomization.LlmSelection)
	}
	if gotValue, ok := got.RoutingLlmCustomization.LlmHyperParameters["useKnowledgeBase"].(bool); !ok || gotValue {
		t.Fatalf("%s.llmHyperParameters[useKnowledgeBase] = %#v, want false", name, got.RoutingLlmCustomization.LlmHyperParameters["useKnowledgeBase"])
	}
	if gotValue, ok := got.RoutingLlmCustomization.LlmHyperParameters["temperature"].(float64); !ok || gotValue != 0.25 {
		t.Fatalf("%s.llmHyperParameters[temperature] = %#v, want 0.25", name, got.RoutingLlmCustomization.LlmHyperParameters["temperature"])
	}
}

func requireAgentLlmConfigCustomModelSelection(
	t *testing.T,
	name string,
	got *generativeaiagentsdk.LlmConfig,
	want generativeaiagentv1beta1.AgentLlmConfig,
) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want populated llmConfig", name)
	}
	requireAgentStringPtr(t, name+".runtimeVersion", got.RuntimeVersion, want.RuntimeVersion)
	if got.RoutingLlmCustomization == nil {
		t.Fatalf("%s.routingLlmCustomization = nil, want populated customization", name)
	}
	requireAgentStringPtr(t, name+".instruction", got.RoutingLlmCustomization.Instruction, want.RoutingLlmCustomization.Instruction)
	selection, ok := got.RoutingLlmCustomization.LlmSelection.(generativeaiagentsdk.CustomGenAiModelLlmSelection)
	if !ok {
		t.Fatalf("%s.llmSelection type = %T, want CustomGenAiModelLlmSelection", name, got.RoutingLlmCustomization.LlmSelection)
	}
	requireAgentStringPtr(t, name+".llmSelection.modelId", selection.ModelId, want.RoutingLlmCustomization.LlmSelection.ModelId)
}

func requireAgentAsyncCurrent(
	t *testing.T,
	resource *generativeaiagentv1beta1.Agent,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want tracked work request")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
}
