package rule

import (
	"context"
	"strconv"
	"strings"
	"testing"

	eventssdk "github.com/oracle/oci-go-sdk/v65/events"
	eventsv1beta1 "github.com/oracle/oci-service-operator/api/events/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testRuleCompartmentID = "ocid1.compartment.oc1..rule"
	testRuleID            = "ocid1.eventrule.oc1..rule"
	testRuleName          = "test-rule"
	testRuleTopicID       = "ocid1.onstopic.oc1..topic"
	testRuleStreamID      = "ocid1.stream.oc1..stream"
)

type fakeRuleOCIClient struct {
	createRequests  []eventssdk.CreateRuleRequest
	createResponses []eventssdk.CreateRuleResponse
	createErrors    []error
	getRequests     []eventssdk.GetRuleRequest
	getResponses    []eventssdk.GetRuleResponse
	getErrors       []error
	listRequests    []eventssdk.ListRulesRequest
	listResponses   []eventssdk.ListRulesResponse
	listErrors      []error
	updateRequests  []eventssdk.UpdateRuleRequest
	updateResponses []eventssdk.UpdateRuleResponse
	updateErrors    []error
	deleteRequests  []eventssdk.DeleteRuleRequest
	deleteResponses []eventssdk.DeleteRuleResponse
	deleteErrors    []error
}

func (f *fakeRuleOCIClient) CreateRule(_ context.Context, request eventssdk.CreateRuleRequest) (eventssdk.CreateRuleResponse, error) {
	f.createRequests = append(f.createRequests, request)
	index := len(f.createRequests) - 1
	if err := indexedRuleError(f.createErrors, index); err != nil {
		return eventssdk.CreateRuleResponse{}, err
	}
	if index < len(f.createResponses) {
		return f.createResponses[index], nil
	}
	return eventssdk.CreateRuleResponse{}, nil
}

func (f *fakeRuleOCIClient) GetRule(_ context.Context, request eventssdk.GetRuleRequest) (eventssdk.GetRuleResponse, error) {
	f.getRequests = append(f.getRequests, request)
	index := len(f.getRequests) - 1
	if err := indexedRuleError(f.getErrors, index); err != nil {
		return eventssdk.GetRuleResponse{}, err
	}
	if index < len(f.getResponses) {
		return f.getResponses[index], nil
	}
	return eventssdk.GetRuleResponse{}, nil
}

func (f *fakeRuleOCIClient) ListRules(_ context.Context, request eventssdk.ListRulesRequest) (eventssdk.ListRulesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	index := len(f.listRequests) - 1
	if err := indexedRuleError(f.listErrors, index); err != nil {
		return eventssdk.ListRulesResponse{}, err
	}
	if index < len(f.listResponses) {
		return f.listResponses[index], nil
	}
	return eventssdk.ListRulesResponse{}, nil
}

func (f *fakeRuleOCIClient) UpdateRule(_ context.Context, request eventssdk.UpdateRuleRequest) (eventssdk.UpdateRuleResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	index := len(f.updateRequests) - 1
	if err := indexedRuleError(f.updateErrors, index); err != nil {
		return eventssdk.UpdateRuleResponse{}, err
	}
	if index < len(f.updateResponses) {
		return f.updateResponses[index], nil
	}
	return eventssdk.UpdateRuleResponse{}, nil
}

func (f *fakeRuleOCIClient) DeleteRule(_ context.Context, request eventssdk.DeleteRuleRequest) (eventssdk.DeleteRuleResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	index := len(f.deleteRequests) - 1
	if err := indexedRuleError(f.deleteErrors, index); err != nil {
		return eventssdk.DeleteRuleResponse{}, err
	}
	if index < len(f.deleteResponses) {
		return f.deleteResponses[index], nil
	}
	return eventssdk.DeleteRuleResponse{}, nil
}

func indexedRuleError(errors []error, index int) error {
	if index < len(errors) {
		return errors[index]
	}
	return nil
}

func TestRuleRuntimeSemanticsAndBodyShaping(t *testing.T) {
	hooks := newRuleDefaultRuntimeHooks(eventssdk.EventsClient{})
	applyRuleRuntimeHooks(&hooks)
	requireRuleRuntimeSemantics(t, hooks.Semantics)

	resource := newTestRule()
	resource.Spec.Actions.Actions[0].IsEnabled = false
	resource.Spec.Actions.Actions = append(resource.Spec.Actions.Actions, eventsv1beta1.RuleActionsAction{
		JsonData:  `{"actionType":"OSS","streamId":"` + testRuleStreamID + `","description":"stream","isEnabled":false}`,
		IsEnabled: true,
	})
	body, err := buildRuleCreateBody(resource)
	if err != nil {
		t.Fatalf("buildRuleCreateBody() error = %v", err)
	}
	requireRuleCreateBodyPreservesActionValues(t, body)
}

func requireRuleRuntimeSemantics(t *testing.T, semantics *generatedruntime.Semantics) {
	t.Helper()

	if semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if got := semantics.Delete.Policy; got != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got)
	}
	if !containsRuleString(semantics.Mutation.Mutable, "actions") {
		t.Fatalf("Mutation.Mutable = %#v, want actions", semantics.Mutation.Mutable)
	}
	if !containsRuleString(semantics.Mutation.ForceNew, "compartmentId") {
		t.Fatalf("Mutation.ForceNew = %#v, want compartmentId", semantics.Mutation.ForceNew)
	}
}

func requireRuleCreateBodyPreservesActionValues(t *testing.T, body eventssdk.CreateRuleDetails) {
	t.Helper()

	if body.Actions == nil || len(body.Actions.Actions) != 2 {
		t.Fatalf("create body actions = %#v, want two actions", body.Actions)
	}
	onsAction, ok := body.Actions.Actions[0].(eventssdk.CreateNotificationServiceActionDetails)
	if !ok {
		t.Fatalf("action[0] type = %T, want CreateNotificationServiceActionDetails", body.Actions.Actions[0])
	}
	if onsAction.IsEnabled == nil || *onsAction.IsEnabled {
		t.Fatalf("ONS action isEnabled = %#v, want explicit false", onsAction.IsEnabled)
	}
	ossAction, ok := body.Actions.Actions[1].(eventssdk.CreateStreamingServiceActionDetails)
	if !ok {
		t.Fatalf("action[1] type = %T, want CreateStreamingServiceActionDetails", body.Actions.Actions[1])
	}
	if got := stringPointerValue(ossAction.StreamId); got != testRuleStreamID {
		t.Fatalf("OSS action streamId = %q, want %q", got, testRuleStreamID)
	}
	if ossAction.IsEnabled == nil || *ossAction.IsEnabled {
		t.Fatalf("OSS action isEnabled = %#v, want explicit false", ossAction.IsEnabled)
	}
}

func TestRuleCreateOrUpdateCreatesAndReadsBack(t *testing.T) {
	resource := newTestRule()
	client := &fakeRuleOCIClient{
		listResponses: []eventssdk.ListRulesResponse{{}},
		createResponses: []eventssdk.CreateRuleResponse{{
			Rule:         sdkRuleFromResource(resource, testRuleID, eventssdk.RuleLifecycleStateCreating),
			OpcRequestId: stringPointer("opc-create-1"),
		}},
		getResponses: []eventssdk.GetRuleResponse{{
			Rule: sdkRuleFromResource(resource, testRuleID, eventssdk.RuleLifecycleStateActive),
		}},
	}
	runtimeClient := newTestRuleServiceClient(client)

	response, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response successful=%t shouldRequeue=%t, want successful non-requeue", response.IsSuccessful, response.ShouldRequeue)
	}
	requireRuleCreateRequest(t, client.createRequests)
	requireRuleActiveStatus(t, resource, "opc-create-1")
}

func requireRuleCreateRequest(t *testing.T, requests []eventssdk.CreateRuleRequest) {
	t.Helper()

	if len(requests) != 1 {
		t.Fatalf("CreateRule calls = %d, want 1", len(requests))
	}
	createRequest := requests[0]
	if createRequest.OpcRetryToken == nil || strings.TrimSpace(*createRequest.OpcRetryToken) == "" {
		t.Fatal("CreateRule OpcRetryToken is empty")
	}
	if got := stringPointerValue(createRequest.CompartmentId); got != testRuleCompartmentID {
		t.Fatalf("CreateRule compartmentId = %q, want %q", got, testRuleCompartmentID)
	}
	if createRequest.Actions == nil || len(createRequest.Actions.Actions) != 1 {
		t.Fatalf("CreateRule actions = %#v, want one action", createRequest.Actions)
	}
}

func requireRuleActiveStatus(t *testing.T, resource *eventsv1beta1.Rule, opcRequestID string) {
	t.Helper()

	if resource.Status.OsokStatus.Ocid != shared.OCID(testRuleID) {
		t.Fatalf("status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, testRuleID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != opcRequestID {
		t.Fatalf("status.opcRequestId = %q, want %s", got, opcRequestID)
	}
	if got := resource.Status.LifecycleState; got != string(eventssdk.RuleLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestRuleCreateOrUpdateBindsExistingAcrossListPages(t *testing.T) {
	resource := newTestRule()
	client := &fakeRuleOCIClient{
		listResponses: []eventssdk.ListRulesResponse{
			{OpcNextPage: stringPointer("page-2")},
			{Items: []eventssdk.RuleSummary{sdkRuleSummary(resource, testRuleID, eventssdk.RuleLifecycleStateActive)}},
		},
		getResponses: []eventssdk.GetRuleResponse{{
			Rule: sdkRuleFromResource(resource, testRuleID, eventssdk.RuleLifecycleStateActive),
		}},
	}
	runtimeClient := newTestRuleServiceClient(client)

	if _, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateRule calls = %d, want 0 for bind", len(client.createRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("ListRules calls = %d, want 2", len(client.listRequests))
	}
	if got := stringPointerValue(client.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second ListRules page = %q, want page-2", got)
	}
	if resource.Status.OsokStatus.Ocid != shared.OCID(testRuleID) {
		t.Fatalf("status.ocid = %q, want bound OCID", resource.Status.OsokStatus.Ocid)
	}
}

func TestRuleCreateOrUpdateNoopDoesNotUpdate(t *testing.T) {
	resource := newTestRule()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRuleID)
	client := &fakeRuleOCIClient{
		getResponses: []eventssdk.GetRuleResponse{{
			Rule: sdkRuleFromResource(resource, testRuleID, eventssdk.RuleLifecycleStateActive),
		}},
	}
	runtimeClient := newTestRuleServiceClient(client)

	if _, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateRule calls = %d, want 0", len(client.updateRequests))
	}
}

func TestRuleCreateOrUpdateMutableUpdatePreservesFalseAction(t *testing.T) {
	resource := newTestRule()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRuleID)
	current := sdkRuleFromResource(resource, testRuleID, eventssdk.RuleLifecycleStateActive)
	*current.IsEnabled = true
	current.Actions.Actions[0] = eventssdk.NotificationServiceAction{
		Id:             stringPointer("action-1"),
		LifecycleState: eventssdk.ActionLifecycleStateActive,
		IsEnabled:      boolPointer(true),
		Description:    stringPointer("old"),
		TopicId:        stringPointer(testRuleTopicID),
	}
	resource.Spec.IsEnabled = false
	resource.Spec.Actions.Actions[0].IsEnabled = false
	resource.Spec.Actions.Actions[0].Description = "updated action"
	updated := sdkRuleFromResource(resource, testRuleID, eventssdk.RuleLifecycleStateActive)
	client := &fakeRuleOCIClient{
		getResponses: []eventssdk.GetRuleResponse{
			{Rule: current},
			{Rule: updated},
		},
		updateResponses: []eventssdk.UpdateRuleResponse{{
			Rule:         updated,
			OpcRequestId: stringPointer("opc-update-1"),
		}},
	}
	runtimeClient := newTestRuleServiceClient(client)

	if _, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{}); err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireRuleMutableUpdateRequest(t, client.updateRequests)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-update-1", got)
	}
}

func requireRuleMutableUpdateRequest(t *testing.T, requests []eventssdk.UpdateRuleRequest) {
	t.Helper()

	if len(requests) != 1 {
		t.Fatalf("UpdateRule calls = %d, want 1", len(requests))
	}
	updateDetails := requests[0].UpdateRuleDetails
	if updateDetails.IsEnabled == nil || *updateDetails.IsEnabled {
		t.Fatalf("UpdateRule isEnabled = %#v, want explicit false", updateDetails.IsEnabled)
	}
	if updateDetails.Actions == nil || len(updateDetails.Actions.Actions) != 1 {
		t.Fatalf("UpdateRule actions = %#v, want one action", updateDetails.Actions)
	}
	updatedAction, ok := updateDetails.Actions.Actions[0].(eventssdk.CreateNotificationServiceActionDetails)
	if !ok {
		t.Fatalf("UpdateRule action type = %T, want CreateNotificationServiceActionDetails", updateDetails.Actions.Actions[0])
	}
	if updatedAction.IsEnabled == nil || *updatedAction.IsEnabled {
		t.Fatalf("UpdateRule action isEnabled = %#v, want explicit false", updatedAction.IsEnabled)
	}
	if got := stringPointerValue(updatedAction.Description); got != "updated action" {
		t.Fatalf("UpdateRule action description = %q, want updated action", got)
	}
}

func TestRuleCreateOrUpdateRejectsCompartmentDriftBeforeUpdate(t *testing.T) {
	resource := newTestRule()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRuleID)
	current := sdkRuleFromResource(resource, testRuleID, eventssdk.RuleLifecycleStateActive)
	current.CompartmentId = stringPointer("ocid1.compartment.oc1..old")
	client := &fakeRuleOCIClient{
		getResponses: []eventssdk.GetRuleResponse{{Rule: current}},
	}
	runtimeClient := newTestRuleServiceClient(client)

	_, err := runtimeClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want compartment drift rejection")
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateRule calls = %d, want 0 after drift rejection", len(client.updateRequests))
	}
}

func TestRuleDeleteConfirmation(t *testing.T) {
	t.Run("retains finalizer while lifecycle is deleting", func(t *testing.T) {
		resource := newTestRule()
		resource.Status.OsokStatus.Ocid = shared.OCID(testRuleID)
		client := &fakeRuleOCIClient{
			getResponses: []eventssdk.GetRuleResponse{
				{Rule: sdkRuleFromResource(resource, testRuleID, eventssdk.RuleLifecycleStateActive)},
				{Rule: sdkRuleFromResource(resource, testRuleID, eventssdk.RuleLifecycleStateDeleting)},
			},
			deleteResponses: []eventssdk.DeleteRuleResponse{{OpcRequestId: stringPointer("opc-delete-1")}},
		}
		runtimeClient := newTestRuleServiceClient(client)

		deleted, err := runtimeClient.Delete(context.Background(), resource)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if deleted {
			t.Fatal("Delete() deleted = true, want false while readback is DELETING")
		}
		if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
			t.Fatalf("status.reason = %q, want Terminating", got)
		}
		if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
			t.Fatalf("status.opcRequestId = %q, want opc-delete-1", got)
		}
	})

	t.Run("releases finalizer after unambiguous not found", func(t *testing.T) {
		resource := newTestRule()
		resource.Status.OsokStatus.Ocid = shared.OCID(testRuleID)
		client := &fakeRuleOCIClient{
			getResponses: []eventssdk.GetRuleResponse{{
				Rule: sdkRuleFromResource(resource, testRuleID, eventssdk.RuleLifecycleStateActive),
			}},
			getErrors: []error{
				nil,
				errortest.NewServiceError(404, errorutil.NotFound, "missing"),
			},
			deleteResponses: []eventssdk.DeleteRuleResponse{{OpcRequestId: stringPointer("opc-delete-1")}},
		}
		runtimeClient := newTestRuleServiceClient(client)

		deleted, err := runtimeClient.Delete(context.Background(), resource)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if !deleted {
			t.Fatal("Delete() deleted = false, want true after unambiguous not found")
		}
		if resource.Status.OsokStatus.DeletedAt == nil {
			t.Fatal("status.deletedAt = nil, want deletion timestamp")
		}
	})

}

func TestRuleDeleteAuthShapedNotFoundRemainsFatal(t *testing.T) {
	resource := newTestRule()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRuleID)
	client := &fakeRuleOCIClient{
		getResponses: []eventssdk.GetRuleResponse{{
			Rule: sdkRuleFromResource(resource, testRuleID, eventssdk.RuleLifecycleStateActive),
		}},
		deleteErrors: []error{errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "auth ambiguity")},
	}
	runtimeClient := newTestRuleServiceClient(client)

	deleted, err := runtimeClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped not found to remain fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped not found")
	}
	if !strings.Contains(err.Error(), "authorization-shaped not found") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped message", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want surfaced error request id", got)
	}
}

func TestRuleDeletePostDeleteAuthShapedConfirmReadRemainsFatal(t *testing.T) {
	resource := newTestRule()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRuleID)
	client := &fakeRuleOCIClient{
		getResponses: []eventssdk.GetRuleResponse{{
			Rule: sdkRuleFromResource(resource, testRuleID, eventssdk.RuleLifecycleStateActive),
		}},
		getErrors: []error{
			nil,
			errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "auth ambiguity"),
		},
		deleteResponses: []eventssdk.DeleteRuleResponse{{OpcRequestId: stringPointer("opc-delete-1")}},
	}
	runtimeClient := newTestRuleServiceClient(client)

	deleted, err := runtimeClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped confirm-read not found to remain fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped confirm-read not found")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned authorization-shaped not found") {
		t.Fatalf("Delete() error = %v, want conservative confirm-read auth-shaped message", err)
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("DeleteRule calls = %d, want 1 before post-delete confirmation", len(client.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want preserved delete request id", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay empty for auth-shaped confirm-read not found")
	}
}

func TestRuleDeletePreDeleteAuthShapedConfirmReadRemainsFatal(t *testing.T) {
	resource := newTestRule()
	resource.Status.OsokStatus.Ocid = shared.OCID(testRuleID)
	resource.Status.OsokStatus.OpcRequestID = "opc-delete-previous"
	client := &fakeRuleOCIClient{
		getErrors: []error{errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "auth ambiguity")},
	}
	runtimeClient := newTestRuleServiceClient(client)

	deleted, err := runtimeClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped pre-delete confirm-read not found to remain fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete confirm-read not found")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteRule calls = %d, want 0 when pre-delete confirmation is ambiguous", len(client.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-previous" {
		t.Fatalf("status.opcRequestId = %q, want preserved existing request id", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay empty for auth-shaped pre-delete confirm-read not found")
	}
}

func TestRuleDeleteEmptyOcidListFallbackConfirmsNoMatch(t *testing.T) {
	resource := newTestRule()
	resource.DeletionTimestamp = &metav1.Time{}
	client := &fakeRuleOCIClient{
		listResponses: []eventssdk.ListRulesResponse{{}},
	}
	runtimeClient := newTestRuleServiceClient(client)

	deleted, err := runtimeClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true when list fallback finds no match")
	}
	if len(client.listRequests) != 1 {
		t.Fatalf("ListRules calls = %d, want 1", len(client.listRequests))
	}
	if len(client.getRequests) != 0 {
		t.Fatalf("GetRule calls = %d, want 0 without a recorded ocid", len(client.getRequests))
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteRule calls = %d, want 0 when list fallback confirms absence", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want deletion timestamp")
	}
}

func TestRuleDeleteEmptyOcidListFallbackRetainsFinalizerForDeletingMatch(t *testing.T) {
	resource := newTestRule()
	resource.DeletionTimestamp = &metav1.Time{}
	client := &fakeRuleOCIClient{
		listResponses: []eventssdk.ListRulesResponse{{
			Items: []eventssdk.RuleSummary{sdkRuleSummary(resource, testRuleID, eventssdk.RuleLifecycleStateDeleting)},
		}},
	}
	runtimeClient := newTestRuleServiceClient(client)

	deleted, err := runtimeClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while list fallback finds DELETING")
	}
	requireRuleDeleteFallbackCallCounts(t, client, 1, 0, 0)
	requireRuleDeletingFallbackStatus(t, resource)
}

func requireRuleDeleteFallbackCallCounts(
	t *testing.T,
	client *fakeRuleOCIClient,
	wantList int,
	wantGet int,
	wantDelete int,
) {
	t.Helper()

	if len(client.listRequests) != wantList {
		t.Fatalf("ListRules calls = %d, want %d", len(client.listRequests), wantList)
	}
	if len(client.getRequests) != wantGet {
		t.Fatalf("GetRule calls = %d, want %d", len(client.getRequests), wantGet)
	}
	if len(client.deleteRequests) != wantDelete {
		t.Fatalf("DeleteRule calls = %d, want %d", len(client.deleteRequests), wantDelete)
	}
}

func requireRuleDeletingFallbackStatus(t *testing.T, resource *eventsv1beta1.Rule) {
	t.Helper()

	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay empty while matching Rule is DELETING")
	}
	if got := resource.Status.OsokStatus.Ocid; got != shared.OCID(testRuleID) {
		t.Fatalf("status.ocid = %q, want %q", got, testRuleID)
	}
	if got := resource.Status.LifecycleState; got != string(eventssdk.RuleLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want Terminating", got)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want delete pending state")
	}
	if got := resource.Status.OsokStatus.Async.Current.NormalizedClass; got != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want pending", got)
	}
}

func TestRuleDeleteEmptyOcidListFallbackConfirmsDeletedMatch(t *testing.T) {
	resource := newTestRule()
	resource.DeletionTimestamp = &metav1.Time{}
	client := &fakeRuleOCIClient{
		listResponses: []eventssdk.ListRulesResponse{{
			Items: []eventssdk.RuleSummary{sdkRuleSummary(resource, testRuleID, eventssdk.RuleLifecycleStateDeleted)},
		}},
	}
	runtimeClient := newTestRuleServiceClient(client)

	deleted, err := runtimeClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true when list fallback finds DELETED")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteRule calls = %d, want 0 when list fallback confirms DELETED", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want deletion timestamp")
	}
}

func TestRuleDeleteEmptyOcidListAuthShapedNotFoundRemainsFatal(t *testing.T) {
	resource := newTestRule()
	client := &fakeRuleOCIClient{
		listErrors: []error{errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "auth ambiguity")},
	}
	runtimeClient := newTestRuleServiceClient(client)

	deleted, err := runtimeClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped list not found to remain fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped list not found")
	}
	if !strings.Contains(err.Error(), "authorization-shaped not found") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped message", err)
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteRule calls = %d, want 0 when list fallback is ambiguous", len(client.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want surfaced list error request id", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay empty for auth-shaped list not found")
	}
}

func newTestRuleServiceClient(fake *fakeRuleOCIClient) RuleServiceClient {
	hooks := newRuleDefaultRuntimeHooks(eventssdk.EventsClient{})
	hooks.Create.Call = fake.CreateRule
	hooks.Get.Call = fake.GetRule
	hooks.List.Call = fake.ListRules
	hooks.Update.Call = fake.UpdateRule
	hooks.Delete.Call = fake.DeleteRule
	applyRuleRuntimeHooks(&hooks)

	delegate := defaultRuleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*eventsv1beta1.Rule](
			buildRuleGeneratedRuntimeConfig(&RuleServiceManager{}, hooks),
		),
	}
	return wrapRuleGeneratedClient(hooks, delegate)
}

func newTestRule() *eventsv1beta1.Rule {
	return &eventsv1beta1.Rule{
		ObjectMeta: metav1.ObjectMeta{Name: testRuleName, Namespace: "default"},
		Spec: eventsv1beta1.RuleSpec{
			CompartmentId: testRuleCompartmentID,
			DisplayName:   testRuleName,
			IsEnabled:     true,
			Condition:     `{"eventType":"com.oraclecloud.objectstorage.createbucket"}`,
			Description:   "rule description",
			FreeformTags:  map[string]string{"env": "test"},
			DefinedTags:   map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
			Actions: eventsv1beta1.RuleActions{
				Actions: []eventsv1beta1.RuleActionsAction{{
					ActionType:  string(eventssdk.ActionDetailsActionTypeOns),
					IsEnabled:   true,
					Description: "notify",
					TopicId:     testRuleTopicID,
				}},
			},
		},
	}
}

func sdkRuleFromResource(
	resource *eventsv1beta1.Rule,
	id string,
	lifecycle eventssdk.RuleLifecycleStateEnum,
) eventssdk.Rule {
	return eventssdk.Rule{
		DisplayName:      stringPointer(resource.Spec.DisplayName),
		LifecycleState:   lifecycle,
		Condition:        stringPointer(resource.Spec.Condition),
		CompartmentId:    stringPointer(resource.Spec.CompartmentId),
		IsEnabled:        boolPointer(resource.Spec.IsEnabled),
		Actions:          sdkRuleActionList(resource.Spec.Actions),
		Id:               stringPointer(id),
		Description:      stringPointer(resource.Spec.Description),
		FreeformTags:     cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:      ruleDefinedTags(resource.Spec.DefinedTags),
		LifecycleMessage: stringPointer(string(lifecycle)),
	}
}

func sdkRuleActionList(actions eventsv1beta1.RuleActions) *eventssdk.ActionList {
	if actions.Actions == nil {
		return nil
	}
	converted := make([]eventssdk.Action, 0, len(actions.Actions))
	for index, action := range actions.Actions {
		actionType, err := ruleActionType(action)
		if err != nil {
			panic(err)
		}
		actionID := "action-" + strconv.Itoa(index+1)
		switch actionType {
		case string(eventssdk.ActionDetailsActionTypeOns):
			converted = append(converted, eventssdk.NotificationServiceAction{
				Id:             stringPointer(actionID),
				LifecycleState: eventssdk.ActionLifecycleStateActive,
				IsEnabled:      boolPointer(action.IsEnabled),
				Description:    stringPointer(action.Description),
				TopicId:        stringPointer(action.TopicId),
			})
		case string(eventssdk.ActionDetailsActionTypeOss):
			converted = append(converted, eventssdk.StreamingServiceAction{
				Id:             stringPointer(actionID),
				LifecycleState: eventssdk.ActionLifecycleStateActive,
				IsEnabled:      boolPointer(action.IsEnabled),
				Description:    stringPointer(action.Description),
				StreamId:       stringPointer(action.StreamId),
			})
		case string(eventssdk.ActionDetailsActionTypeFaas):
			converted = append(converted, eventssdk.FaaSAction{
				Id:             stringPointer(actionID),
				LifecycleState: eventssdk.ActionLifecycleStateActive,
				IsEnabled:      boolPointer(action.IsEnabled),
				Description:    stringPointer(action.Description),
				FunctionId:     stringPointer(action.FunctionId),
			})
		}
	}
	return &eventssdk.ActionList{Actions: converted}
}

func sdkRuleSummary(
	resource *eventsv1beta1.Rule,
	id string,
	lifecycle eventssdk.RuleLifecycleStateEnum,
) eventssdk.RuleSummary {
	return eventssdk.RuleSummary{
		Id:             stringPointer(id),
		DisplayName:    stringPointer(resource.Spec.DisplayName),
		LifecycleState: lifecycle,
		Condition:      stringPointer(resource.Spec.Condition),
		CompartmentId:  stringPointer(resource.Spec.CompartmentId),
		IsEnabled:      boolPointer(resource.Spec.IsEnabled),
		Description:    stringPointer(resource.Spec.Description),
		FreeformTags:   cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:    ruleDefinedTags(resource.Spec.DefinedTags),
	}
}

func containsRuleString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
