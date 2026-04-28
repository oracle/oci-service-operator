/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package trigger

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	devopssdk "github.com/oracle/oci-go-sdk/v65/devops"
	devopsv1beta1 "github.com/oracle/oci-service-operator/api/devops/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	triggerTestProjectID       = "ocid1.devopsproject.oc1..project"
	triggerTestRepositoryID    = "ocid1.devopsrepository.oc1..repository"
	triggerTestBuildPipelineID = "ocid1.devopsbuildpipeline.oc1..pipeline"
	triggerTestID              = "ocid1.devopstrigger.oc1..trigger"
	triggerTestCompartmentID   = "ocid1.compartment.oc1..compartment"
	triggerTestName            = "trigger-runtime-test"
)

type fakeTriggerOCIClient struct {
	createRequests      []devopssdk.CreateTriggerRequest
	getRequests         []devopssdk.GetTriggerRequest
	listRequests        []devopssdk.ListTriggersRequest
	updateRequests      []devopssdk.UpdateTriggerRequest
	deleteRequests      []devopssdk.DeleteTriggerRequest
	workRequestRequests []devopssdk.GetWorkRequestRequest

	createFn      func(context.Context, devopssdk.CreateTriggerRequest) (devopssdk.CreateTriggerResponse, error)
	getFn         func(context.Context, devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error)
	listFn        func(context.Context, devopssdk.ListTriggersRequest) (devopssdk.ListTriggersResponse, error)
	updateFn      func(context.Context, devopssdk.UpdateTriggerRequest) (devopssdk.UpdateTriggerResponse, error)
	deleteFn      func(context.Context, devopssdk.DeleteTriggerRequest) (devopssdk.DeleteTriggerResponse, error)
	workRequestFn func(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error)
}

func (f *fakeTriggerOCIClient) CreateTrigger(ctx context.Context, request devopssdk.CreateTriggerRequest) (devopssdk.CreateTriggerResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return devopssdk.CreateTriggerResponse{}, nil
}

func (f *fakeTriggerOCIClient) GetTrigger(ctx context.Context, request devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return devopssdk.GetTriggerResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing trigger")
}

func (f *fakeTriggerOCIClient) ListTriggers(ctx context.Context, request devopssdk.ListTriggersRequest) (devopssdk.ListTriggersResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return devopssdk.ListTriggersResponse{}, nil
}

func (f *fakeTriggerOCIClient) UpdateTrigger(ctx context.Context, request devopssdk.UpdateTriggerRequest) (devopssdk.UpdateTriggerResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return devopssdk.UpdateTriggerResponse{}, nil
}

func (f *fakeTriggerOCIClient) DeleteTrigger(ctx context.Context, request devopssdk.DeleteTriggerRequest) (devopssdk.DeleteTriggerResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return devopssdk.DeleteTriggerResponse{}, nil
}

func (f *fakeTriggerOCIClient) GetWorkRequest(ctx context.Context, request devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, request)
	}
	return devopssdk.GetWorkRequestResponse{}, nil
}

func TestTriggerRuntimeSemanticsEncodesWorkRequestAndDeleteContracts(t *testing.T) {
	t.Parallel()

	got := newTriggerRuntimeSemantics()
	if got.FormalService != "devops" || got.FormalSlug != "trigger" {
		t.Fatalf("formal identity = %s/%s, want devops/trigger", got.FormalService, got.FormalSlug)
	}
	if got.Async == nil || got.Async.WorkRequest == nil {
		t.Fatalf("async semantics = %#v, want workrequest", got.Async)
	}
	if got.Async.Strategy != "workrequest" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime workrequest", got.Async)
	}
	assertTriggerStrings(t, "work request phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("finalizer policy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v follow-up %#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	assertTriggerStrings(t, "list match fields", got.List.MatchFields, []string{"projectId", "displayName", "triggerSource", "connectionId", "repositoryId"})
	assertTriggerStrings(t, "force-new fields", got.Mutation.ForceNew, []string{"projectId", "triggerSource"})
}

func TestTriggerCreateUsesPolymorphicBodyAndWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newTriggerResource()
	workRequests := map[string]devopssdk.WorkRequest{
		"wr-create": newTriggerWorkRequest("wr-create", devopssdk.OperationTypeCreateTrigger, devopssdk.OperationStatusInProgress, devopssdk.ActionTypeInProgress, triggerTestID),
	}
	var createRequest devopssdk.CreateTriggerRequest
	getCalls := 0

	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, &fakeTriggerOCIClient{
		createFn: func(_ context.Context, request devopssdk.CreateTriggerRequest) (devopssdk.CreateTriggerResponse, error) {
			createRequest = request
			return devopssdk.CreateTriggerResponse{
				TriggerCreateResult: sdkTriggerCreateResult(resource, triggerTestID, devopssdk.TriggerLifecycleStateActive),
				OpcWorkRequestId:    common.String("wr-create"),
				OpcRequestId:        common.String("opc-create"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireTriggerStringPtr(t, "workRequestId", request.WorkRequestId, "wr-create")
			return devopssdk.GetWorkRequestResponse{WorkRequest: workRequests["wr-create"]}, nil
		},
		getFn: func(_ context.Context, request devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
			getCalls++
			requireTriggerStringPtr(t, "triggerId", request.TriggerId, triggerTestID)
			return devopssdk.GetTriggerResponse{Trigger: sdkTrigger(resource, triggerTestID, devopssdk.TriggerLifecycleStateActive)}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireTriggerCreateStarted(t, response, resource, createRequest, getCalls)

	workRequests["wr-create"] = newTriggerWorkRequest("wr-create", devopssdk.OperationTypeCreateTrigger, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeCreated, triggerTestID)
	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() after work request success error = %v", err)
	}
	requireTriggerCreateFinished(t, response, resource, getCalls)
}

func requireTriggerCreateStarted(
	t *testing.T,
	response servicemanager.OSOKResponse,
	resource *devopsv1beta1.Trigger,
	createRequest devopssdk.CreateTriggerRequest,
	getCalls int,
) {
	t.Helper()

	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	details, ok := createRequest.CreateTriggerDetails.(devopssdk.CreateDevopsCodeRepositoryTriggerDetails)
	if !ok {
		t.Fatalf("create details type = %T, want CreateDevopsCodeRepositoryTriggerDetails", createRequest.CreateTriggerDetails)
	}
	requireTriggerStringPtr(t, "create projectId", details.ProjectId, triggerTestProjectID)
	requireTriggerStringPtr(t, "create repositoryId", details.RepositoryId, triggerTestRepositoryID)
	if len(details.Actions) != 1 {
		t.Fatalf("create actions = %d, want 1", len(details.Actions))
	}
	if createRequest.OpcRetryToken == nil || strings.TrimSpace(*createRequest.OpcRetryToken) == "" {
		t.Fatal("create opc retry token is empty")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.opcRequestId = %q, want opc-create", got)
	}
	requireTriggerAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)
	if getCalls != 0 {
		t.Fatalf("GetTrigger() calls = %d, want 0 while work request is pending", getCalls)
	}
}

func requireTriggerCreateFinished(
	t *testing.T,
	response servicemanager.OSOKResponse,
	resource *devopsv1beta1.Trigger,
	getCalls int,
) {
	t.Helper()

	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() after work request success response = %#v, want successful no requeue", response)
	}
	if getCalls != 1 {
		t.Fatalf("GetTrigger() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != triggerTestID {
		t.Fatalf("status.ocid = %q, want %q", got, triggerTestID)
	}
	if got := resource.Status.Id; got != triggerTestID {
		t.Fatalf("status.id = %q, want %q", got, triggerTestID)
	}
	requireTriggerCondition(t, resource, shared.Active, v1.ConditionTrue)
}

func TestTriggerCreateWorkRequestSuccessWaitsForReadbackNotFound(t *testing.T) {
	t.Parallel()

	resource := newTriggerResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fake := &fakeTriggerOCIClient{
		workRequestFn: func(_ context.Context, request devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireTriggerStringPtr(t, "workRequestId", request.WorkRequestId, "wr-create")
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: newTriggerWorkRequest("wr-create", devopssdk.OperationTypeCreateTrigger, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeCreated, triggerTestID),
			}, nil
		},
		getFn: func(_ context.Context, request devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
			requireTriggerStringPtr(t, "triggerId", request.TriggerId, triggerTestID)
			return devopssdk.GetTriggerResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "trigger not yet readable")
		},
	}
	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v, want nil while create readback is not yet visible", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateTrigger() calls = %d, want 0 while resuming work request", len(fake.createRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("GetTrigger() calls = %d, want 1 readback attempt", len(fake.getRequests))
	}
	requireTriggerAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.Async.Current.Message; !strings.Contains(got, "waiting") {
		t.Fatalf("async message = %q, want waiting detail", got)
	}
}

func TestTriggerCreateWorkRequestSuccessFailsOnAuthShapedReadbackNotFound(t *testing.T) {
	t.Parallel()

	resource := newTriggerResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fake := &fakeTriggerOCIClient{
		workRequestFn: func(_ context.Context, request devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireTriggerStringPtr(t, "workRequestId", request.WorkRequestId, "wr-create")
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: newTriggerWorkRequest("wr-create", devopssdk.OperationTypeCreateTrigger, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeCreated, triggerTestID),
			}, nil
		},
		getFn: func(_ context.Context, request devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
			requireTriggerStringPtr(t, "triggerId", request.TriggerId, triggerTestID)
			return devopssdk.GetTriggerResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
	}
	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want auth-shaped 404 to stay fatal")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateTrigger() calls = %d, want 0 while resuming work request", len(fake.createRequests))
	}
	requireTriggerCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func TestTriggerCreateOrUpdateBindsExistingAcrossListPages(t *testing.T) {
	t.Parallel()

	resource := newTriggerResource()
	fake := &fakeTriggerOCIClient{
		listFn: func(_ context.Context, request devopssdk.ListTriggersRequest) (devopssdk.ListTriggersResponse, error) {
			switch stringValue(request.Page) {
			case "":
				return devopssdk.ListTriggersResponse{
					TriggerCollection: devopssdk.TriggerCollection{Items: []devopssdk.TriggerSummary{
						sdkTriggerSummary(resource, "ocid1.devopstrigger.oc1..other", "other-display"),
					}},
					OpcNextPage: common.String("page-2"),
				}, nil
			case "page-2":
				return devopssdk.ListTriggersResponse{
					TriggerCollection: devopssdk.TriggerCollection{Items: []devopssdk.TriggerSummary{
						sdkTriggerSummary(resource, triggerTestID, resource.Spec.DisplayName),
					}},
				}, nil
			default:
				t.Fatalf("unexpected list page %q", stringValue(request.Page))
				return devopssdk.ListTriggersResponse{}, nil
			}
		},
		getFn: func(_ context.Context, request devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
			requireTriggerStringPtr(t, "triggerId", request.TriggerId, triggerTestID)
			return devopssdk.GetTriggerResponse{Trigger: sdkTrigger(resource, triggerTestID, devopssdk.TriggerLifecycleStateActive)}, nil
		},
	}
	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no requeue", response)
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListTriggers() calls = %d, want 2 pages", len(fake.listRequests))
	}
	if got := stringValue(fake.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second list page = %q, want page-2", got)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateTrigger() calls = %d, want 0 after bind", len(fake.createRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != triggerTestID {
		t.Fatalf("status.ocid = %q, want %q", got, triggerTestID)
	}
}

func TestTriggerCreateOrUpdateNoopsWhenReadbackMatches(t *testing.T) {
	t.Parallel()

	resource := newTriggerResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(triggerTestID)
	fake := &fakeTriggerOCIClient{
		getFn: func(_ context.Context, request devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
			requireTriggerStringPtr(t, "triggerId", request.TriggerId, triggerTestID)
			return devopssdk.GetTriggerResponse{Trigger: sdkTrigger(resource, triggerTestID, devopssdk.TriggerLifecycleStateActive)}, nil
		},
	}
	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no requeue", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateTrigger() calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateTrigger() calls = %d, want 0", len(fake.updateRequests))
	}
	requireTriggerCondition(t, resource, shared.Active, v1.ConditionTrue)
}

func TestTriggerMutableUpdateUsesConcreteBodyAndWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newTriggerResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(triggerTestID)
	resource.Spec.DisplayName = "updated-trigger"
	workRequests := map[string]devopssdk.WorkRequest{
		"wr-update": newTriggerWorkRequest("wr-update", devopssdk.OperationTypeUpdateTrigger, devopssdk.OperationStatusInProgress, devopssdk.ActionTypeInProgress, triggerTestID),
	}
	currentDisplayName := "old-trigger"
	var updateRequest devopssdk.UpdateTriggerRequest

	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, &fakeTriggerOCIClient{
		getFn: func(_ context.Context, request devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
			requireTriggerStringPtr(t, "triggerId", request.TriggerId, triggerTestID)
			current := sdkTrigger(resource, triggerTestID, devopssdk.TriggerLifecycleStateActive)
			current.DisplayName = common.String(currentDisplayName)
			return devopssdk.GetTriggerResponse{Trigger: current}, nil
		},
		updateFn: func(_ context.Context, request devopssdk.UpdateTriggerRequest) (devopssdk.UpdateTriggerResponse, error) {
			updateRequest = request
			currentDisplayName = resource.Spec.DisplayName
			return devopssdk.UpdateTriggerResponse{
				Trigger:          sdkTrigger(resource, triggerTestID, devopssdk.TriggerLifecycleStateActive),
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireTriggerStringPtr(t, "workRequestId", request.WorkRequestId, "wr-update")
			return devopssdk.GetWorkRequestResponse{WorkRequest: workRequests["wr-update"]}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	requireTriggerStringPtr(t, "update triggerId", updateRequest.TriggerId, triggerTestID)
	details, ok := updateRequest.UpdateTriggerDetails.(devopssdk.UpdateDevopsCodeRepositoryTriggerDetails)
	if !ok {
		t.Fatalf("update details type = %T, want UpdateDevopsCodeRepositoryTriggerDetails", updateRequest.UpdateTriggerDetails)
	}
	requireTriggerStringPtr(t, "update displayName", details.DisplayName, resource.Spec.DisplayName)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.opcRequestId = %q, want opc-update", got)
	}
	requireTriggerAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update", shared.OSOKAsyncClassPending)

	workRequests["wr-update"] = newTriggerWorkRequest("wr-update", devopssdk.OperationTypeUpdateTrigger, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeUpdated, triggerTestID)
	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() after work request success error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() after work request success response = %#v, want successful no requeue", response)
	}
	if got := resource.Status.DisplayName; got != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	requireTriggerCondition(t, resource, shared.Active, v1.ConditionTrue)
}

func TestTriggerRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newTriggerResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(triggerTestID)
	resource.Spec.ProjectId = "ocid1.devopsproject.oc1..different"
	fake := &fakeTriggerOCIClient{
		getFn: func(context.Context, devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
			current := sdkTrigger(resource, triggerTestID, devopssdk.TriggerLifecycleStateActive)
			current.ProjectId = common.String(triggerTestProjectID)
			return devopssdk.GetTriggerResponse{Trigger: current}, nil
		},
	}
	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want projectId drift rejection")
	}
	if !strings.Contains(err.Error(), "projectId") {
		t.Fatalf("CreateOrUpdate() error = %v, want projectId detail", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateTrigger() calls = %d, want 0", len(fake.updateRequests))
	}
	requireTriggerCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func TestTriggerDeleteKeepsFinalizerWhileOCIIsDeleting(t *testing.T) {
	t.Parallel()

	resource := newTriggerResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(triggerTestID)
	fake := &fakeTriggerOCIClient{
		getFn: func(context.Context, devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
			return devopssdk.GetTriggerResponse{Trigger: sdkTrigger(resource, triggerTestID, devopssdk.TriggerLifecycleStateDeleting)}, nil
		},
	}
	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI is DELETING")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteTrigger() calls = %d, want 0 for already pending delete", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set while OCI delete is still pending")
	}
	requireTriggerAsync(t, resource, shared.OSOKAsyncPhaseDelete, "", shared.OSOKAsyncClassPending)
}

func TestTriggerDeleteTracksWorkRequestUntilReadbackConfirmsDeletion(t *testing.T) {
	t.Parallel()

	resource := newTriggerResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(triggerTestID)
	workRequests := map[string]devopssdk.WorkRequest{
		"wr-delete": newTriggerWorkRequest("wr-delete", devopssdk.OperationTypeDeleteTrigger, devopssdk.OperationStatusInProgress, devopssdk.ActionTypeInProgress, triggerTestID),
	}
	getCalls := 0
	fake := &fakeTriggerOCIClient{
		getFn: func(context.Context, devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
			getCalls++
			if getCalls == 1 {
				return devopssdk.GetTriggerResponse{Trigger: sdkTrigger(resource, triggerTestID, devopssdk.TriggerLifecycleStateActive)}, nil
			}
			return devopssdk.GetTriggerResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "trigger deleted")
		},
		deleteFn: func(context.Context, devopssdk.DeleteTriggerRequest) (devopssdk.DeleteTriggerResponse, error) {
			return devopssdk.DeleteTriggerResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireTriggerStringPtr(t, "workRequestId", request.WorkRequestId, "wr-delete")
			return devopssdk.GetWorkRequestResponse{WorkRequest: workRequests["wr-delete"]}, nil
		},
	}
	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete work request is pending")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete", got)
	}
	requireTriggerAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)

	workRequests["wr-delete"] = newTriggerWorkRequest("wr-delete", devopssdk.OperationTypeDeleteTrigger, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeDeleted, triggerTestID)
	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() after work request success error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after delete readback returns not found")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want confirmed deletion timestamp")
	}
}

func TestTriggerDeleteWaitsForPendingCreateOrUpdateWorkRequest(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		phase shared.OSOKAsyncPhase
		op    devopssdk.OperationTypeEnum
	}{
		{
			name:  "create",
			phase: shared.OSOKAsyncPhaseCreate,
			op:    devopssdk.OperationTypeCreateTrigger,
		},
		{
			name:  "update",
			phase: shared.OSOKAsyncPhaseUpdate,
			op:    devopssdk.OperationTypeUpdateTrigger,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource := newTriggerResource()
			seedTriggerCurrentWorkRequest(resource, tc.phase, "wr-"+tc.name)
			fake := &fakeTriggerOCIClient{
				workRequestFn: func(_ context.Context, request devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
					requireTriggerStringPtr(t, "workRequestId", request.WorkRequestId, "wr-"+tc.name)
					return devopssdk.GetWorkRequestResponse{
						WorkRequest: newTriggerWorkRequest("wr-"+tc.name, tc.op, devopssdk.OperationStatusInProgress, devopssdk.ActionTypeInProgress, triggerTestID),
					}, nil
				},
			}
			client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)

			deleted, err := client.Delete(context.Background(), resource)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if deleted {
				t.Fatal("Delete() deleted = true, want false while write work request is pending")
			}
			if len(fake.deleteRequests) != 0 {
				t.Fatalf("DeleteTrigger() calls = %d, want 0 before write work request finishes", len(fake.deleteRequests))
			}
			requireTriggerAsync(t, resource, tc.phase, "wr-"+tc.name, shared.OSOKAsyncClassPending)
			if got := resource.Status.OsokStatus.Async.Current.Message; !strings.Contains(got, "waiting before delete") {
				t.Fatalf("async message = %q, want waiting-before-delete detail", got)
			}
		})
	}
}

type triggerSucceededWriteWorkRequestCase struct {
	name   string
	phase  shared.OSOKAsyncPhase
	op     devopssdk.OperationTypeEnum
	action devopssdk.ActionTypeEnum
}

func triggerSucceededWriteWorkRequestCases() []triggerSucceededWriteWorkRequestCase {
	return []triggerSucceededWriteWorkRequestCase{
		{
			name:   "create",
			phase:  shared.OSOKAsyncPhaseCreate,
			op:     devopssdk.OperationTypeCreateTrigger,
			action: devopssdk.ActionTypeCreated,
		},
		{
			name:   "update",
			phase:  shared.OSOKAsyncPhaseUpdate,
			op:     devopssdk.OperationTypeUpdateTrigger,
			action: devopssdk.ActionTypeUpdated,
		},
	}
}

func (tc triggerSucceededWriteWorkRequestCase) workRequestID() string {
	return "wr-" + tc.name
}

func (tc triggerSucceededWriteWorkRequestCase) workRequest() devopssdk.WorkRequest {
	return newTriggerWorkRequest(tc.workRequestID(), tc.op, devopssdk.OperationStatusSucceeded, tc.action, triggerTestID)
}

func TestTriggerDeleteStartsAfterCreateOrUpdateWorkRequestSucceeds(t *testing.T) {
	t.Parallel()

	for _, tc := range triggerSucceededWriteWorkRequestCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requireTriggerDeleteStartsAfterSucceededWriteWorkRequest(t, tc)
		})
	}
}

func requireTriggerDeleteStartsAfterSucceededWriteWorkRequest(t *testing.T, tc triggerSucceededWriteWorkRequestCase) {
	t.Helper()

	resource := newTriggerResource()
	seedTriggerCurrentWorkRequest(resource, tc.phase, tc.workRequestID())
	workRequests := map[string]devopssdk.WorkRequest{
		tc.workRequestID(): tc.workRequest(),
		"wr-delete":        newTriggerWorkRequest("wr-delete", devopssdk.OperationTypeDeleteTrigger, devopssdk.OperationStatusInProgress, devopssdk.ActionTypeInProgress, triggerTestID),
	}
	fake := &fakeTriggerOCIClient{
		getFn: func(_ context.Context, request devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
			requireTriggerStringPtr(t, "triggerId", request.TriggerId, triggerTestID)
			return devopssdk.GetTriggerResponse{Trigger: sdkTrigger(resource, triggerTestID, devopssdk.TriggerLifecycleStateActive)}, nil
		},
		deleteFn: func(_ context.Context, request devopssdk.DeleteTriggerRequest) (devopssdk.DeleteTriggerResponse, error) {
			requireTriggerStringPtr(t, "delete triggerId", request.TriggerId, triggerTestID)
			return devopssdk.DeleteTriggerResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
		workRequestFn: newTriggerWorkRequestLookup(t, workRequests),
	}
	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	requireTriggerDeleteStartedAfterWriteSuccess(t, resource, fake, deleted)
}

func requireTriggerDeleteStartedAfterWriteSuccess(
	t *testing.T,
	resource *devopsv1beta1.Trigger,
	fake *fakeTriggerOCIClient,
	deleted bool,
) {
	t.Helper()

	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete work request is pending")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteTrigger() calls = %d, want 1 after write work request succeeds", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete", got)
	}
	requireTriggerAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)
}

func TestTriggerDeleteWaitsWhenSucceededCreateOrUpdateReadbackIsNotVisible(t *testing.T) {
	t.Parallel()

	for _, tc := range triggerSucceededWriteWorkRequestCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requireTriggerDeleteWaitsForSucceededWriteReadback(t, tc)
		})
	}
}

func requireTriggerDeleteWaitsForSucceededWriteReadback(t *testing.T, tc triggerSucceededWriteWorkRequestCase) {
	t.Helper()

	resource := newTriggerResource()
	seedTriggerCurrentWorkRequest(resource, tc.phase, tc.workRequestID())
	fake := &fakeTriggerOCIClient{
		getFn: func(_ context.Context, request devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
			requireTriggerStringPtr(t, "triggerId", request.TriggerId, triggerTestID)
			return devopssdk.GetTriggerResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "trigger not yet readable")
		},
		workRequestFn: newTriggerWorkRequestLookup(t, map[string]devopssdk.WorkRequest{tc.workRequestID(): tc.workRequest()}),
	}
	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v, want nil while readback is not yet visible", err)
	}
	requireTriggerDeleteWaitsForWriteReadback(t, resource, fake, tc, deleted)
}

func requireTriggerDeleteWaitsForWriteReadback(
	t *testing.T,
	resource *devopsv1beta1.Trigger,
	fake *fakeTriggerOCIClient,
	tc triggerSucceededWriteWorkRequestCase,
	deleted bool,
) {
	t.Helper()

	if deleted {
		t.Fatal("Delete() deleted = true, want false while write readback is not yet visible")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteTrigger() calls = %d, want 0 before write readback is visible", len(fake.deleteRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != triggerTestID {
		t.Fatalf("status.ocid = %q, want %q", got, triggerTestID)
	}
	requireTriggerAsync(t, resource, tc.phase, tc.workRequestID(), shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.Async.Current.Message; !strings.Contains(got, "waiting") {
		t.Fatalf("async message = %q, want waiting detail", got)
	}
}

func newTriggerWorkRequestLookup(
	t *testing.T,
	workRequests map[string]devopssdk.WorkRequest,
) func(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
	t.Helper()

	return func(_ context.Context, request devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
		workRequestID := stringValue(request.WorkRequestId)
		workRequest, ok := workRequests[workRequestID]
		if !ok {
			t.Fatalf("unexpected work request %q", workRequestID)
			return devopssdk.GetWorkRequestResponse{}, nil
		}
		return devopssdk.GetWorkRequestResponse{WorkRequest: workRequest}, nil
	}
}

func TestTriggerDeleteKeepsAuthShapedReadbackFailureAfterWriteWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newTriggerResource()
	seedTriggerCurrentWorkRequest(resource, shared.OSOKAsyncPhaseUpdate, "wr-update")
	fake := &fakeTriggerOCIClient{
		getFn: func(_ context.Context, request devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
			requireTriggerStringPtr(t, "triggerId", request.TriggerId, triggerTestID)
			return devopssdk.GetTriggerResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		workRequestFn: func(_ context.Context, request devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error) {
			requireTriggerStringPtr(t, "workRequestId", request.WorkRequestId, "wr-update")
			return devopssdk.GetWorkRequestResponse{
				WorkRequest: newTriggerWorkRequest("wr-update", devopssdk.OperationTypeUpdateTrigger, devopssdk.OperationStatusSucceeded, devopssdk.ActionTypeUpdated, triggerTestID),
			}, nil
		},
	}
	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped 404 readback to stay fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteTrigger() calls = %d, want 0 after ambiguous readback", len(fake.deleteRequests))
	}
	requireTriggerCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func TestTriggerDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := newTriggerResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(triggerTestID)
	fake := &fakeTriggerOCIClient{
		getFn: func(context.Context, devopssdk.GetTriggerRequest) (devopssdk.GetTriggerResponse, error) {
			return devopssdk.GetTriggerResponse{Trigger: sdkTrigger(resource, triggerTestID, devopssdk.TriggerLifecycleStateActive)}, nil
		},
		deleteFn: func(context.Context, devopssdk.DeleteTriggerRequest) (devopssdk.DeleteTriggerResponse, error) {
			return devopssdk.DeleteTriggerResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
	}
	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want conservative auth-shaped 404 failure")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "ambiguous not-found") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found detail", err)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set after auth-shaped 404")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestTriggerCreateRecordsServiceErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := newTriggerResource()
	fake := &fakeTriggerOCIClient{
		createFn: func(context.Context, devopssdk.CreateTriggerRequest) (devopssdk.CreateTriggerResponse, error) {
			return devopssdk.CreateTriggerResponse{}, errortest.NewServiceError(500, "InternalError", "transient create failure")
		},
	}
	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want service error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", got)
	}
	requireTriggerCondition(t, resource, shared.Failed, v1.ConditionFalse)
}

func TestTriggerCreateRequiresActionsBeforeOCI(t *testing.T) {
	t.Parallel()

	resource := newTriggerResource()
	resource.Spec.JsonData = ""
	fake := &fakeTriggerOCIClient{}
	client := newTriggerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want actions validation")
	}
	if !strings.Contains(err.Error(), "jsonData.actions") {
		t.Fatalf("CreateOrUpdate() error = %v, want jsonData.actions detail", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateTrigger() calls = %d, want 0", len(fake.createRequests))
	}
}

func newTriggerResource() *devopsv1beta1.Trigger {
	return &devopsv1beta1.Trigger{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      triggerTestName,
			UID:       "trigger-test-uid",
		},
		Spec: devopsv1beta1.TriggerSpec{
			JsonData:      triggerActionsJSON(triggerTestBuildPipelineID),
			DisplayName:   triggerTestName,
			Description:   "trigger runtime test",
			ProjectId:     triggerTestProjectID,
			TriggerSource: string(devopssdk.TriggerTriggerSourceDevopsCodeRepository),
			RepositoryId:  triggerTestRepositoryID,
			FreeformTags:  map[string]string{"purpose": "runtime-test"},
			DefinedTags: map[string]shared.MapValue{
				"test": {"owner": "osok"},
			},
		},
	}
}

func triggerActionsJSON(buildPipelineID string) string {
	return fmt.Sprintf(`{"actions":[{"type":"TRIGGER_BUILD_PIPELINE","buildPipelineId":%q}]}`, buildPipelineID)
}

func sdkTrigger(resource *devopsv1beta1.Trigger, id string, lifecycle devopssdk.TriggerLifecycleStateEnum) devopssdk.DevopsCodeRepositoryTrigger {
	return devopssdk.DevopsCodeRepositoryTrigger{
		Id:             common.String(id),
		ProjectId:      common.String(triggerTestProjectID),
		CompartmentId:  common.String(triggerTestCompartmentID),
		RepositoryId:   common.String(resource.Spec.RepositoryId),
		DisplayName:    common.String(resource.Spec.DisplayName),
		Description:    common.String(resource.Spec.Description),
		Actions:        []devopssdk.TriggerAction{sdkTriggerAction()},
		FreeformTags:   resource.Spec.FreeformTags,
		DefinedTags:    map[string]map[string]interface{}{"test": {"owner": "osok"}},
		LifecycleState: lifecycle,
	}
}

func sdkTriggerCreateResult(resource *devopsv1beta1.Trigger, id string, lifecycle devopssdk.TriggerLifecycleStateEnum) devopssdk.DevopsCodeRepositoryTriggerCreateResult {
	return devopssdk.DevopsCodeRepositoryTriggerCreateResult{
		Id:             common.String(id),
		ProjectId:      common.String(triggerTestProjectID),
		CompartmentId:  common.String(triggerTestCompartmentID),
		RepositoryId:   common.String(resource.Spec.RepositoryId),
		DisplayName:    common.String(resource.Spec.DisplayName),
		Description:    common.String(resource.Spec.Description),
		Actions:        []devopssdk.TriggerAction{sdkTriggerAction()},
		FreeformTags:   resource.Spec.FreeformTags,
		DefinedTags:    map[string]map[string]interface{}{"test": {"owner": "osok"}},
		LifecycleState: lifecycle,
	}
}

func sdkTriggerSummary(resource *devopsv1beta1.Trigger, id string, displayName string) devopssdk.DevopsCodeRepositoryTriggerSummary {
	return devopssdk.DevopsCodeRepositoryTriggerSummary{
		Id:             common.String(id),
		ProjectId:      common.String(triggerTestProjectID),
		CompartmentId:  common.String(triggerTestCompartmentID),
		RepositoryId:   common.String(resource.Spec.RepositoryId),
		DisplayName:    common.String(displayName),
		Description:    common.String(resource.Spec.Description),
		FreeformTags:   resource.Spec.FreeformTags,
		DefinedTags:    map[string]map[string]interface{}{"test": {"owner": "osok"}},
		LifecycleState: devopssdk.TriggerLifecycleStateActive,
	}
}

func sdkTriggerAction() devopssdk.TriggerBuildPipelineAction {
	return devopssdk.TriggerBuildPipelineAction{
		BuildPipelineId: common.String(triggerTestBuildPipelineID),
	}
}

func newTriggerWorkRequest(
	id string,
	operationType devopssdk.OperationTypeEnum,
	status devopssdk.OperationStatusEnum,
	action devopssdk.ActionTypeEnum,
	resourceID string,
) devopssdk.WorkRequest {
	return devopssdk.WorkRequest{
		Id:              common.String(id),
		OperationType:   operationType,
		Status:          status,
		CompartmentId:   common.String(triggerTestCompartmentID),
		PercentComplete: common.Float32(50),
		Resources: []devopssdk.WorkRequestResource{
			{
				EntityType: common.String("trigger"),
				ActionType: action,
				Identifier: common.String(resourceID),
			},
		},
	}
}

func seedTriggerCurrentWorkRequest(resource *devopsv1beta1.Trigger, phase shared.OSOKAsyncPhase, workRequestID string) {
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
}

func requireTriggerStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireTriggerAsync(
	t *testing.T,
	resource *devopsv1beta1.Trigger,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatalf("status.async.current = nil, want phase %s", phase)
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest && workRequestID != "" {
		t.Fatalf("async source = %q, want workrequest", current.Source)
	}
	if current.Phase != phase {
		t.Fatalf("async phase = %q, want %q", current.Phase, phase)
	}
	if workRequestID != "" && current.WorkRequestID != workRequestID {
		t.Fatalf("async workRequestID = %q, want %q", current.WorkRequestID, workRequestID)
	}
	if current.NormalizedClass != class {
		t.Fatalf("async class = %q, want %q", current.NormalizedClass, class)
	}
}

func requireTriggerCondition(t *testing.T, resource *devopsv1beta1.Trigger, condition shared.OSOKConditionType, status v1.ConditionStatus) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("conditions = nil, want %s", condition)
	}
	got := conditions[len(conditions)-1]
	if got.Type != condition || got.Status != status {
		t.Fatalf("last condition = %s/%s, want %s/%s", got.Type, got.Status, condition, status)
	}
}

func assertTriggerStrings(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d (%v)", name, len(got), len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q", name, i, got[i], want[i])
		}
	}
}
