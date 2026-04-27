/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package logsavedsearch

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	loggingsdk "github.com/oracle/oci-go-sdk/v65/logging"
	loggingv1beta1 "github.com/oracle/oci-service-operator/api/logging/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type logSavedSearchOCIClient interface {
	CreateLogSavedSearch(context.Context, loggingsdk.CreateLogSavedSearchRequest) (loggingsdk.CreateLogSavedSearchResponse, error)
	GetLogSavedSearch(context.Context, loggingsdk.GetLogSavedSearchRequest) (loggingsdk.GetLogSavedSearchResponse, error)
	ListLogSavedSearches(context.Context, loggingsdk.ListLogSavedSearchesRequest) (loggingsdk.ListLogSavedSearchesResponse, error)
	UpdateLogSavedSearch(context.Context, loggingsdk.UpdateLogSavedSearchRequest) (loggingsdk.UpdateLogSavedSearchResponse, error)
	DeleteLogSavedSearch(context.Context, loggingsdk.DeleteLogSavedSearchRequest) (loggingsdk.DeleteLogSavedSearchResponse, error)
}

type fakeLogSavedSearchOCIClient struct {
	createFn func(context.Context, loggingsdk.CreateLogSavedSearchRequest) (loggingsdk.CreateLogSavedSearchResponse, error)
	getFn    func(context.Context, loggingsdk.GetLogSavedSearchRequest) (loggingsdk.GetLogSavedSearchResponse, error)
	listFn   func(context.Context, loggingsdk.ListLogSavedSearchesRequest) (loggingsdk.ListLogSavedSearchesResponse, error)
	updateFn func(context.Context, loggingsdk.UpdateLogSavedSearchRequest) (loggingsdk.UpdateLogSavedSearchResponse, error)
	deleteFn func(context.Context, loggingsdk.DeleteLogSavedSearchRequest) (loggingsdk.DeleteLogSavedSearchResponse, error)
}

func (f *fakeLogSavedSearchOCIClient) CreateLogSavedSearch(ctx context.Context, req loggingsdk.CreateLogSavedSearchRequest) (loggingsdk.CreateLogSavedSearchResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return loggingsdk.CreateLogSavedSearchResponse{}, nil
}

func (f *fakeLogSavedSearchOCIClient) GetLogSavedSearch(ctx context.Context, req loggingsdk.GetLogSavedSearchRequest) (loggingsdk.GetLogSavedSearchResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return loggingsdk.GetLogSavedSearchResponse{}, nil
}

func (f *fakeLogSavedSearchOCIClient) ListLogSavedSearches(ctx context.Context, req loggingsdk.ListLogSavedSearchesRequest) (loggingsdk.ListLogSavedSearchesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return loggingsdk.ListLogSavedSearchesResponse{}, nil
}

func (f *fakeLogSavedSearchOCIClient) UpdateLogSavedSearch(ctx context.Context, req loggingsdk.UpdateLogSavedSearchRequest) (loggingsdk.UpdateLogSavedSearchResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return loggingsdk.UpdateLogSavedSearchResponse{}, nil
}

func (f *fakeLogSavedSearchOCIClient) DeleteLogSavedSearch(ctx context.Context, req loggingsdk.DeleteLogSavedSearchRequest) (loggingsdk.DeleteLogSavedSearchResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return loggingsdk.DeleteLogSavedSearchResponse{}, nil
}

func TestLogSavedSearchRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	t.Parallel()

	got := newLogSavedSearchRuntimeSemantics()
	if got == nil {
		t.Fatal("newLogSavedSearchRuntimeSemantics() = nil")
	}

	if got.FormalService != "logging" {
		t.Fatalf("FormalService = %q, want logging", got.FormalService)
	}
	if got.FormalSlug != "logsavedsearch" {
		t.Fatalf("FormalSlug = %q, want logsavedsearch", got.FormalSlug)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want read-after-write", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want read-after-write", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want confirm-delete", got.DeleteFollowUp.Strategy)
	}

	assertLogSavedSearchStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertLogSavedSearchStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertLogSavedSearchStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertLogSavedSearchStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertLogSavedSearchStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertLogSavedSearchStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "lifecycleState", "name"})
	assertLogSavedSearchStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"definedTags", "description", "freeformTags", "name", "query"})
	assertLogSavedSearchStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId"})
}

func TestLogSavedSearchServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.logsavedsearch.oc1..created"
	resource := makeLogSavedSearchResource()
	getCalls := 0
	var createRequest loggingsdk.CreateLogSavedSearchRequest

	client := newTestLogSavedSearchClient(&fakeLogSavedSearchOCIClient{
		listFn: func(_ context.Context, req loggingsdk.ListLogSavedSearchesRequest) (loggingsdk.ListLogSavedSearchesResponse, error) {
			requireStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireStringPtr(t, "list name", req.Name, resource.Spec.Name)
			return loggingsdk.ListLogSavedSearchesResponse{}, nil
		},
		createFn: func(_ context.Context, req loggingsdk.CreateLogSavedSearchRequest) (loggingsdk.CreateLogSavedSearchResponse, error) {
			createRequest = req
			return loggingsdk.CreateLogSavedSearchResponse{
				LogSavedSearch: makeSDKLogSavedSearch(createdID, resource, loggingsdk.LogSavedSearchLifecycleStateCreating),
				OpcRequestId:   common.String("opc-create-1"),
			}, nil
		},
		getFn: func(_ context.Context, req loggingsdk.GetLogSavedSearchRequest) (loggingsdk.GetLogSavedSearchResponse, error) {
			getCalls++
			requireStringPtr(t, "get logSavedSearchId", req.LogSavedSearchId, createdID)
			return loggingsdk.GetLogSavedSearchResponse{
				LogSavedSearch: makeSDKLogSavedSearch(createdID, resource, loggingsdk.LogSavedSearchLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after follow-up read reports ACTIVE")
	}
	if getCalls != 1 {
		t.Fatalf("GetLogSavedSearch() calls = %d, want 1 follow-up read", getCalls)
	}
	requireStringPtr(t, "create compartmentId", createRequest.CreateLogSavedSearchDetails.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "create name", createRequest.CreateLogSavedSearchDetails.Name, resource.Spec.Name)
	requireStringPtr(t, "create query", createRequest.CreateLogSavedSearchDetails.Query, resource.Spec.Query)
	requireStringPtr(t, "create description", createRequest.CreateLogSavedSearchDetails.Description, resource.Spec.Description)
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-1", got)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(loggingsdk.LogSavedSearchLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestLogSavedSearchServiceClientBindsExistingWithoutCreate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.logsavedsearch.oc1..existing"
	resource := makeLogSavedSearchResource()
	createCalled := false
	updateCalled := false
	getCalls := 0

	client := newTestLogSavedSearchClient(&fakeLogSavedSearchOCIClient{
		listFn: func(_ context.Context, req loggingsdk.ListLogSavedSearchesRequest) (loggingsdk.ListLogSavedSearchesResponse, error) {
			requireStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireStringPtr(t, "list name", req.Name, resource.Spec.Name)
			return loggingsdk.ListLogSavedSearchesResponse{
				LogSavedSearchSummaryCollection: loggingsdk.LogSavedSearchSummaryCollection{
					Items: []loggingsdk.LogSavedSearchSummary{
						makeSDKLogSavedSearchSummary(existingID, resource, loggingsdk.LogSavedSearchLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req loggingsdk.GetLogSavedSearchRequest) (loggingsdk.GetLogSavedSearchResponse, error) {
			getCalls++
			requireStringPtr(t, "get logSavedSearchId", req.LogSavedSearchId, existingID)
			return loggingsdk.GetLogSavedSearchResponse{
				LogSavedSearch: makeSDKLogSavedSearch(existingID, resource, loggingsdk.LogSavedSearchLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, loggingsdk.CreateLogSavedSearchRequest) (loggingsdk.CreateLogSavedSearchResponse, error) {
			createCalled = true
			return loggingsdk.CreateLogSavedSearchResponse{}, nil
		},
		updateFn: func(context.Context, loggingsdk.UpdateLogSavedSearchRequest) (loggingsdk.UpdateLogSavedSearchResponse, error) {
			updateCalled = true
			return loggingsdk.UpdateLogSavedSearchResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if createCalled {
		t.Fatal("CreateLogSavedSearch() should not be called when list finds a reusable match")
	}
	if updateCalled {
		t.Fatal("UpdateLogSavedSearch() should not be called when mutable state already matches")
	}
	if getCalls != 1 {
		t.Fatalf("GetLogSavedSearch() calls = %d, want 1 live assessment read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
		t.Fatalf("status.ocid = %q, want %q", got, existingID)
	}
}

func TestLogSavedSearchServiceClientUpdatesSupportedMutableDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.logsavedsearch.oc1..update"
	resource := makeLogSavedSearchResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.Name = resource.Spec.Name
	resource.Status.Query = "search \"old\""
	resource.Status.Description = "old description"

	var updateRequest loggingsdk.UpdateLogSavedSearchRequest
	getCalls := 0

	client := newTestLogSavedSearchClient(&fakeLogSavedSearchOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetLogSavedSearchRequest) (loggingsdk.GetLogSavedSearchResponse, error) {
			getCalls++
			requireStringPtr(t, "get logSavedSearchId", req.LogSavedSearchId, existingID)
			current := makeSDKLogSavedSearch(existingID, resource, loggingsdk.LogSavedSearchLifecycleStateActive)
			if getCalls == 1 {
				current.Query = common.String("search \"old\"")
				current.Description = common.String("old description")
			}
			return loggingsdk.GetLogSavedSearchResponse{LogSavedSearch: current}, nil
		},
		updateFn: func(_ context.Context, req loggingsdk.UpdateLogSavedSearchRequest) (loggingsdk.UpdateLogSavedSearchResponse, error) {
			updateRequest = req
			updated := makeSDKLogSavedSearch(existingID, resource, loggingsdk.LogSavedSearchLifecycleStateUpdating)
			return loggingsdk.UpdateLogSavedSearchResponse{LogSavedSearch: updated}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after follow-up read reports ACTIVE")
	}
	requireStringPtr(t, "update logSavedSearchId", updateRequest.LogSavedSearchId, existingID)
	requireStringPtr(t, "update query", updateRequest.UpdateLogSavedSearchDetails.Query, resource.Spec.Query)
	requireStringPtr(t, "update description", updateRequest.UpdateLogSavedSearchDetails.Description, resource.Spec.Description)
	if updateRequest.UpdateLogSavedSearchDetails.Name != nil {
		t.Fatalf("update name = %v, want nil because name did not drift", updateRequest.UpdateLogSavedSearchDetails.Name)
	}
	if getCalls != 2 {
		t.Fatalf("GetLogSavedSearch() calls = %d, want live assessment plus update follow-up", getCalls)
	}
	if got := resource.Status.Query; got != resource.Spec.Query {
		t.Fatalf("status.query = %q, want %q", got, resource.Spec.Query)
	}
}

func TestLogSavedSearchServiceClientRejectsCompartmentDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.logsavedsearch.oc1..force-new"
	resource := makeLogSavedSearchResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	resource.Status.CompartmentId = "ocid1.compartment.oc1..old"
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	updateCalled := false

	client := newTestLogSavedSearchClient(&fakeLogSavedSearchOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetLogSavedSearchRequest) (loggingsdk.GetLogSavedSearchResponse, error) {
			requireStringPtr(t, "get logSavedSearchId", req.LogSavedSearchId, existingID)
			current := makeSDKLogSavedSearch(existingID, resource, loggingsdk.LogSavedSearchLifecycleStateActive)
			current.CompartmentId = common.String("ocid1.compartment.oc1..old")
			return loggingsdk.GetLogSavedSearchResponse{LogSavedSearch: current}, nil
		},
		updateFn: func(context.Context, loggingsdk.UpdateLogSavedSearchRequest) (loggingsdk.UpdateLogSavedSearchResponse, error) {
			updateCalled = true
			return loggingsdk.UpdateLogSavedSearchResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId force-new rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if updateCalled {
		t.Fatal("UpdateLogSavedSearch() should not be called after force-new drift rejection")
	}
}

func TestLogSavedSearchCreateOrUpdateClassifiesLifecycleStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		state          loggingsdk.LogSavedSearchLifecycleStateEnum
		wantSuccessful bool
		wantRequeue    bool
		wantReason     shared.OSOKConditionType
	}{
		{
			name:           "creating",
			state:          loggingsdk.LogSavedSearchLifecycleStateCreating,
			wantSuccessful: true,
			wantRequeue:    true,
			wantReason:     shared.Provisioning,
		},
		{
			name:           "updating",
			state:          loggingsdk.LogSavedSearchLifecycleStateUpdating,
			wantSuccessful: true,
			wantRequeue:    true,
			wantReason:     shared.Updating,
		},
		{
			name:           "deleting",
			state:          loggingsdk.LogSavedSearchLifecycleStateDeleting,
			wantSuccessful: true,
			wantRequeue:    true,
			wantReason:     shared.Terminating,
		},
		{
			name:           "active",
			state:          loggingsdk.LogSavedSearchLifecycleStateActive,
			wantSuccessful: true,
			wantRequeue:    false,
			wantReason:     shared.Active,
		},
		{
			name:           "failed",
			state:          loggingsdk.LogSavedSearchLifecycleStateFailed,
			wantSuccessful: false,
			wantRequeue:    false,
			wantReason:     shared.Failed,
		},
		{
			name:           "inactive",
			state:          loggingsdk.LogSavedSearchLifecycleStateInactive,
			wantSuccessful: false,
			wantRequeue:    false,
			wantReason:     shared.Failed,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			const existingID = "ocid1.logsavedsearch.oc1..lifecycle"
			resource := makeLogSavedSearchResource()
			resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
			resource.Status.Id = existingID

			client := newTestLogSavedSearchClient(&fakeLogSavedSearchOCIClient{
				getFn: func(_ context.Context, req loggingsdk.GetLogSavedSearchRequest) (loggingsdk.GetLogSavedSearchResponse, error) {
					requireStringPtr(t, "get logSavedSearchId", req.LogSavedSearchId, existingID)
					return loggingsdk.GetLogSavedSearchResponse{
						LogSavedSearch: makeSDKLogSavedSearch(existingID, resource, tc.state),
					}, nil
				},
			})

			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err != nil && tc.wantSuccessful {
				t.Fatalf("CreateOrUpdate() error = %v", err)
			}
			if response.IsSuccessful != tc.wantSuccessful {
				t.Fatalf("CreateOrUpdate() IsSuccessful = %t, want %t; err=%v", response.IsSuccessful, tc.wantSuccessful, err)
			}
			if response.ShouldRequeue != tc.wantRequeue {
				t.Fatalf("CreateOrUpdate() ShouldRequeue = %t, want %t", response.ShouldRequeue, tc.wantRequeue)
			}
			if got := resource.Status.OsokStatus.Reason; got != string(tc.wantReason) {
				t.Fatalf("status.reason = %q, want %q", got, tc.wantReason)
			}
			if got := resource.Status.LifecycleState; got != string(tc.state) {
				t.Fatalf("status.lifecycleState = %q, want %q", got, tc.state)
			}
		})
	}
}

func TestLogSavedSearchDeleteWaitsForDeletingConfirmation(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.logsavedsearch.oc1..delete"
	resource := makeLogSavedSearchResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	getCalls := 0
	deleteCalls := 0

	client := newTestLogSavedSearchClient(&fakeLogSavedSearchOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetLogSavedSearchRequest) (loggingsdk.GetLogSavedSearchResponse, error) {
			getCalls++
			requireStringPtr(t, "get logSavedSearchId", req.LogSavedSearchId, existingID)
			state := loggingsdk.LogSavedSearchLifecycleStateActive
			if getCalls > 1 {
				state = loggingsdk.LogSavedSearchLifecycleStateDeleting
			}
			return loggingsdk.GetLogSavedSearchResponse{
				LogSavedSearch: makeSDKLogSavedSearch(existingID, resource, state),
			}, nil
		},
		deleteFn: func(_ context.Context, req loggingsdk.DeleteLogSavedSearchRequest) (loggingsdk.DeleteLogSavedSearchResponse, error) {
			deleteCalls++
			requireStringPtr(t, "delete logSavedSearchId", req.LogSavedSearchId, existingID)
			return loggingsdk.DeleteLogSavedSearchResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false while OCI reports DELETING")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteLogSavedSearch() calls = %d, want 1", deleteCalls)
	}
	if got := resource.Status.LifecycleState; got != string(loggingsdk.LogSavedSearchLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want Terminating", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay nil while delete confirmation is pending")
	}
}

func TestLogSavedSearchDeleteConfirmsReadNotFound(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.logsavedsearch.oc1..delete-gone"
	resource := makeLogSavedSearchResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	getCalls := 0

	client := newTestLogSavedSearchClient(&fakeLogSavedSearchOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetLogSavedSearchRequest) (loggingsdk.GetLogSavedSearchResponse, error) {
			getCalls++
			requireStringPtr(t, "get logSavedSearchId", req.LogSavedSearchId, existingID)
			if getCalls > 1 {
				return loggingsdk.GetLogSavedSearchResponse{}, errortest.NewServiceError(404, "NotFound", "LogSavedSearch deleted")
			}
			return loggingsdk.GetLogSavedSearchResponse{
				LogSavedSearch: makeSDKLogSavedSearch(existingID, resource, loggingsdk.LogSavedSearchLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req loggingsdk.DeleteLogSavedSearchRequest) (loggingsdk.DeleteLogSavedSearchResponse, error) {
			requireStringPtr(t, "delete logSavedSearchId", req.LogSavedSearchId, existingID)
			return loggingsdk.DeleteLogSavedSearchResponse{OpcRequestId: common.String("opc-delete-2")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true after confirm read reports NotFound")
	}
	if getCalls != 2 {
		t.Fatalf("GetLogSavedSearch() calls = %d, want pre-delete and confirm-delete reads", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt should be set after confirmed deletion")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want Terminating", got)
	}
}

func newTestLogSavedSearchClient(client logSavedSearchOCIClient) defaultLogSavedSearchServiceClient {
	if client == nil {
		client = &fakeLogSavedSearchOCIClient{}
	}
	hooks := newLogSavedSearchRuntimeHooksWithOCIClient(client)
	applyLogSavedSearchRuntimeHooks(&hooks)
	return defaultLogSavedSearchServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loggingv1beta1.LogSavedSearch](
			buildLogSavedSearchGeneratedRuntimeConfig(
				&LogSavedSearchServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}},
				hooks,
			),
		),
	}
}

func newLogSavedSearchRuntimeHooksWithOCIClient(client logSavedSearchOCIClient) LogSavedSearchRuntimeHooks {
	return LogSavedSearchRuntimeHooks{
		Semantics: newLogSavedSearchRuntimeSemantics(),
		Create: runtimeOperationHooks[loggingsdk.CreateLogSavedSearchRequest, loggingsdk.CreateLogSavedSearchResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateLogSavedSearchDetails", RequestName: "CreateLogSavedSearchDetails", Contribution: "body", PreferResourceID: false}},
			Call: func(ctx context.Context, request loggingsdk.CreateLogSavedSearchRequest) (loggingsdk.CreateLogSavedSearchResponse, error) {
				return client.CreateLogSavedSearch(ctx, request)
			},
		},
		Get: runtimeOperationHooks[loggingsdk.GetLogSavedSearchRequest, loggingsdk.GetLogSavedSearchResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "LogSavedSearchId", RequestName: "logSavedSearchId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request loggingsdk.GetLogSavedSearchRequest) (loggingsdk.GetLogSavedSearchResponse, error) {
				return client.GetLogSavedSearch(ctx, request)
			},
		},
		List: runtimeOperationHooks[loggingsdk.ListLogSavedSearchesRequest, loggingsdk.ListLogSavedSearchesResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
				{FieldName: "LogSavedSearchId", RequestName: "logSavedSearchId", Contribution: "query", PreferResourceID: false},
				{FieldName: "Name", RequestName: "name", Contribution: "query", PreferResourceID: false},
				{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
				{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
				{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
			},
			Call: func(ctx context.Context, request loggingsdk.ListLogSavedSearchesRequest) (loggingsdk.ListLogSavedSearchesResponse, error) {
				return client.ListLogSavedSearches(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loggingsdk.UpdateLogSavedSearchRequest, loggingsdk.UpdateLogSavedSearchResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "LogSavedSearchId", RequestName: "logSavedSearchId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateLogSavedSearchDetails", RequestName: "UpdateLogSavedSearchDetails", Contribution: "body", PreferResourceID: false},
			},
			Call: func(ctx context.Context, request loggingsdk.UpdateLogSavedSearchRequest) (loggingsdk.UpdateLogSavedSearchResponse, error) {
				return client.UpdateLogSavedSearch(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loggingsdk.DeleteLogSavedSearchRequest, loggingsdk.DeleteLogSavedSearchResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "LogSavedSearchId", RequestName: "logSavedSearchId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request loggingsdk.DeleteLogSavedSearchRequest) (loggingsdk.DeleteLogSavedSearchResponse, error) {
				return client.DeleteLogSavedSearch(ctx, request)
			},
		},
		WrapGeneratedClient: []func(LogSavedSearchServiceClient) LogSavedSearchServiceClient{},
	}
}

func makeLogSavedSearchResource() *loggingv1beta1.LogSavedSearch {
	return &loggingv1beta1.LogSavedSearch{
		Spec: loggingv1beta1.LogSavedSearchSpec{
			CompartmentId: "ocid1.compartment.oc1..logsavedsearch",
			Name:          "osok-log-saved-search",
			Query:         "search \"example\"",
			Description:   "example saved search",
			FreeformTags:  map[string]string{"managed-by": "osok"},
		},
	}
}

func makeSDKLogSavedSearch(id string, resource *loggingv1beta1.LogSavedSearch, state loggingsdk.LogSavedSearchLifecycleStateEnum) loggingsdk.LogSavedSearch {
	return loggingsdk.LogSavedSearch{
		Id:             common.String(id),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		Name:           common.String(resource.Spec.Name),
		Query:          common.String(resource.Spec.Query),
		Description:    common.String(resource.Spec.Description),
		FreeformTags:   cloneLogSavedSearchStringMap(resource.Spec.FreeformTags),
		LifecycleState: state,
	}
}

func makeSDKLogSavedSearchSummary(id string, resource *loggingv1beta1.LogSavedSearch, state loggingsdk.LogSavedSearchLifecycleStateEnum) loggingsdk.LogSavedSearchSummary {
	return loggingsdk.LogSavedSearchSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		Name:           common.String(resource.Spec.Name),
		Query:          common.String(resource.Spec.Query),
		Description:    common.String(resource.Spec.Description),
		FreeformTags:   cloneLogSavedSearchStringMap(resource.Spec.FreeformTags),
		LifecycleState: state,
	}
}

func cloneLogSavedSearchStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func assertLogSavedSearchStringSliceEqual(t *testing.T, name string, got, want []string) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	t.Fatalf("%s = %#v, want %#v", name, got, want)
}
