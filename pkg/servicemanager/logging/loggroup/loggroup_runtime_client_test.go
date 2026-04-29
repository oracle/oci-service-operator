/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loggroup

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

type logGroupOCIClient interface {
	CreateLogGroup(context.Context, loggingsdk.CreateLogGroupRequest) (loggingsdk.CreateLogGroupResponse, error)
	GetLogGroup(context.Context, loggingsdk.GetLogGroupRequest) (loggingsdk.GetLogGroupResponse, error)
	ListLogGroups(context.Context, loggingsdk.ListLogGroupsRequest) (loggingsdk.ListLogGroupsResponse, error)
	UpdateLogGroup(context.Context, loggingsdk.UpdateLogGroupRequest) (loggingsdk.UpdateLogGroupResponse, error)
	DeleteLogGroup(context.Context, loggingsdk.DeleteLogGroupRequest) (loggingsdk.DeleteLogGroupResponse, error)
	GetWorkRequest(context.Context, loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error)
}

type fakeLogGroupOCIClient struct {
	createFn      func(context.Context, loggingsdk.CreateLogGroupRequest) (loggingsdk.CreateLogGroupResponse, error)
	getFn         func(context.Context, loggingsdk.GetLogGroupRequest) (loggingsdk.GetLogGroupResponse, error)
	listFn        func(context.Context, loggingsdk.ListLogGroupsRequest) (loggingsdk.ListLogGroupsResponse, error)
	updateFn      func(context.Context, loggingsdk.UpdateLogGroupRequest) (loggingsdk.UpdateLogGroupResponse, error)
	deleteFn      func(context.Context, loggingsdk.DeleteLogGroupRequest) (loggingsdk.DeleteLogGroupResponse, error)
	workRequestFn func(context.Context, loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error)
}

func (f *fakeLogGroupOCIClient) CreateLogGroup(ctx context.Context, req loggingsdk.CreateLogGroupRequest) (loggingsdk.CreateLogGroupResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return loggingsdk.CreateLogGroupResponse{}, nil
}

func (f *fakeLogGroupOCIClient) GetLogGroup(ctx context.Context, req loggingsdk.GetLogGroupRequest) (loggingsdk.GetLogGroupResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return loggingsdk.GetLogGroupResponse{}, nil
}

func (f *fakeLogGroupOCIClient) ListLogGroups(ctx context.Context, req loggingsdk.ListLogGroupsRequest) (loggingsdk.ListLogGroupsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return loggingsdk.ListLogGroupsResponse{}, nil
}

func (f *fakeLogGroupOCIClient) UpdateLogGroup(ctx context.Context, req loggingsdk.UpdateLogGroupRequest) (loggingsdk.UpdateLogGroupResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return loggingsdk.UpdateLogGroupResponse{}, nil
}

func (f *fakeLogGroupOCIClient) DeleteLogGroup(ctx context.Context, req loggingsdk.DeleteLogGroupRequest) (loggingsdk.DeleteLogGroupResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return loggingsdk.DeleteLogGroupResponse{}, nil
}

func (f *fakeLogGroupOCIClient) GetWorkRequest(ctx context.Context, req loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return loggingsdk.GetWorkRequestResponse{}, nil
}

func TestLogGroupRuntimeSemanticsEncodesWorkRequestLifecycle(t *testing.T) {
	t.Parallel()

	got := newLogGroupRuntimeSemantics()
	if got == nil {
		t.Fatal("newLogGroupRuntimeSemantics() = nil")
	}

	if got.FormalService != "logging" {
		t.Fatalf("FormalService = %q, want logging", got.FormalService)
	}
	if got.FormalSlug != "loggroup" {
		t.Fatalf("FormalSlug = %q, want loggroup", got.FormalSlug)
	}
	if got.Async == nil {
		t.Fatal("Async = nil, want workrequest contract")
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
	if got.Async.WorkRequest.Source != "service-sdk" {
		t.Fatalf("Async.WorkRequest.Source = %q, want service-sdk", got.Async.WorkRequest.Source)
	}
	assertLogGroupStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
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

	assertLogGroupStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertLogGroupStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertLogGroupStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertLogGroupStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertLogGroupStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertLogGroupStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName"})
	assertLogGroupStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"definedTags", "description", "displayName", "freeformTags"})
	assertLogGroupStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId"})
}

func TestLogGroupServiceClientCreatesWithWorkRequestAndProjectsStatus(t *testing.T) {
	t.Parallel()

	const (
		createdID     = "ocid1.loggroup.oc1..created"
		workRequestID = "wr-loggroup-create"
	)
	resource := makeLogGroupResource()
	workRequests := map[string]loggingsdk.WorkRequest{
		workRequestID: makeLogGroupWorkRequest(workRequestID, loggingsdk.OperationTypesCreateLogGroup, loggingsdk.OperationStatusInProgress, loggingsdk.ActionTypesInProgress, ""),
	}
	var createRequest loggingsdk.CreateLogGroupRequest
	getCalls := 0

	client := newTestLogGroupClient(&fakeLogGroupOCIClient{
		listFn: func(_ context.Context, req loggingsdk.ListLogGroupsRequest) (loggingsdk.ListLogGroupsResponse, error) {
			requireLogGroupStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireLogGroupStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return loggingsdk.ListLogGroupsResponse{}, nil
		},
		createFn: func(_ context.Context, req loggingsdk.CreateLogGroupRequest) (loggingsdk.CreateLogGroupResponse, error) {
			createRequest = req
			return loggingsdk.CreateLogGroupResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error) {
			requireLogGroupStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return loggingsdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
		getFn: func(_ context.Context, req loggingsdk.GetLogGroupRequest) (loggingsdk.GetLogGroupResponse, error) {
			getCalls++
			requireLogGroupStringPtr(t, "get logGroupId", req.LogGroupId, createdID)
			return loggingsdk.GetLogGroupResponse{
				LogGroup: makeSDKLogGroup(createdID, resource, loggingsdk.LogGroupLifecycleStateActive),
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
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while create work request is pending")
	}
	requireLogGroupStringPtr(t, "create compartmentId", createRequest.CreateLogGroupDetails.CompartmentId, resource.Spec.CompartmentId)
	requireLogGroupStringPtr(t, "create displayName", createRequest.CreateLogGroupDetails.DisplayName, resource.Spec.DisplayName)
	requireLogGroupStringPtr(t, "create description", createRequest.CreateLogGroupDetails.Description, resource.Spec.Description)
	requireLogGroupAsyncCurrent(t, resource, shared.OSOKAsyncSourceWorkRequest, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-1", got)
	}
	if getCalls != 0 {
		t.Fatalf("GetLogGroup() calls = %d, want 0 while work request is pending", getCalls)
	}

	workRequests[workRequestID] = makeLogGroupWorkRequest(workRequestID, loggingsdk.OperationTypesCreateLogGroup, loggingsdk.OperationStatusSucceeded, loggingsdk.ActionTypesCreated, createdID)
	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() after work request success error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() after work request success response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after follow-up read reports ACTIVE")
	}
	if getCalls != 1 {
		t.Fatalf("GetLogGroup() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(loggingsdk.LogGroupLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after active read", resource.Status.OsokStatus.Async.Current)
	}
}

func TestLogGroupServiceClientBindsExistingWithoutCreate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.loggroup.oc1..existing"
	resource := makeLogGroupResource()
	createCalled := false
	updateCalled := false
	getCalls := 0

	client := newTestLogGroupClient(&fakeLogGroupOCIClient{
		listFn: func(_ context.Context, req loggingsdk.ListLogGroupsRequest) (loggingsdk.ListLogGroupsResponse, error) {
			requireLogGroupStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireLogGroupStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			return loggingsdk.ListLogGroupsResponse{
				Items: []loggingsdk.LogGroupSummary{
					makeSDKLogGroupSummary(existingID, resource, loggingsdk.LogGroupLifecycleStateActive),
				},
			}, nil
		},
		getFn: func(_ context.Context, req loggingsdk.GetLogGroupRequest) (loggingsdk.GetLogGroupResponse, error) {
			getCalls++
			requireLogGroupStringPtr(t, "get logGroupId", req.LogGroupId, existingID)
			return loggingsdk.GetLogGroupResponse{
				LogGroup: makeSDKLogGroup(existingID, resource, loggingsdk.LogGroupLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, loggingsdk.CreateLogGroupRequest) (loggingsdk.CreateLogGroupResponse, error) {
			createCalled = true
			return loggingsdk.CreateLogGroupResponse{}, nil
		},
		updateFn: func(context.Context, loggingsdk.UpdateLogGroupRequest) (loggingsdk.UpdateLogGroupResponse, error) {
			updateCalled = true
			return loggingsdk.UpdateLogGroupResponse{}, nil
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
		t.Fatal("CreateLogGroup() should not be called when list finds a reusable match")
	}
	if updateCalled {
		t.Fatal("UpdateLogGroup() should not be called when mutable state already matches")
	}
	if getCalls != 1 {
		t.Fatalf("GetLogGroup() calls = %d, want 1 live assessment read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
		t.Fatalf("status.ocid = %q, want %q", got, existingID)
	}
}

func TestLogGroupServiceClientUpdatesSupportedMutableDrift(t *testing.T) {
	t.Parallel()

	const (
		existingID    = "ocid1.loggroup.oc1..update"
		workRequestID = "wr-loggroup-update"
	)
	resource := makeLogGroupResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.DisplayName = "old display"
	resource.Status.Description = "old description"

	var updateRequest loggingsdk.UpdateLogGroupRequest
	getCalls := 0

	client := newTestLogGroupClient(&fakeLogGroupOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetLogGroupRequest) (loggingsdk.GetLogGroupResponse, error) {
			getCalls++
			requireLogGroupStringPtr(t, "get logGroupId", req.LogGroupId, existingID)
			current := makeSDKLogGroup(existingID, resource, loggingsdk.LogGroupLifecycleStateActive)
			if getCalls == 1 {
				current.DisplayName = common.String("old display")
				current.Description = common.String("old description")
			}
			return loggingsdk.GetLogGroupResponse{LogGroup: current}, nil
		},
		updateFn: func(_ context.Context, req loggingsdk.UpdateLogGroupRequest) (loggingsdk.UpdateLogGroupResponse, error) {
			updateRequest = req
			return loggingsdk.UpdateLogGroupResponse{
				OpcWorkRequestId: common.String(workRequestID),
			}, nil
		},
		workRequestFn: func(_ context.Context, req loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error) {
			requireLogGroupStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return loggingsdk.GetWorkRequestResponse{
				WorkRequest: makeLogGroupWorkRequest(workRequestID, loggingsdk.OperationTypesUpdateLogGroup, loggingsdk.OperationStatusSucceeded, loggingsdk.ActionTypesUpdated, existingID),
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
		t.Fatal("CreateOrUpdate() should not requeue after update follow-up read reports ACTIVE")
	}
	requireLogGroupStringPtr(t, "update logGroupId", updateRequest.LogGroupId, existingID)
	requireLogGroupStringPtr(t, "update displayName", updateRequest.UpdateLogGroupDetails.DisplayName, resource.Spec.DisplayName)
	requireLogGroupStringPtr(t, "update description", updateRequest.UpdateLogGroupDetails.Description, resource.Spec.Description)
	if getCalls != 2 {
		t.Fatalf("GetLogGroup() calls = %d, want live assessment plus update follow-up", getCalls)
	}
	if got := resource.Status.DisplayName; got != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := resource.Status.Description; got != resource.Spec.Description {
		t.Fatalf("status.description = %q, want %q", got, resource.Spec.Description)
	}
}

func TestLogGroupServiceClientRejectsCompartmentDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.loggroup.oc1..force-new"
	resource := makeLogGroupResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	resource.Status.CompartmentId = "ocid1.compartment.oc1..old"
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	updateCalled := false

	client := newTestLogGroupClient(&fakeLogGroupOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetLogGroupRequest) (loggingsdk.GetLogGroupResponse, error) {
			requireLogGroupStringPtr(t, "get logGroupId", req.LogGroupId, existingID)
			current := makeSDKLogGroup(existingID, resource, loggingsdk.LogGroupLifecycleStateActive)
			current.CompartmentId = common.String("ocid1.compartment.oc1..old")
			return loggingsdk.GetLogGroupResponse{LogGroup: current}, nil
		},
		updateFn: func(context.Context, loggingsdk.UpdateLogGroupRequest) (loggingsdk.UpdateLogGroupResponse, error) {
			updateCalled = true
			return loggingsdk.UpdateLogGroupResponse{}, nil
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
		t.Fatal("UpdateLogGroup() should not be called after force-new drift rejection")
	}
}

func TestLogGroupCreateOrUpdateClassifiesLifecycleStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		state          loggingsdk.LogGroupLifecycleStateEnum
		wantSuccessful bool
		wantRequeue    bool
		wantReason     shared.OSOKConditionType
	}{
		{
			name:           "creating",
			state:          loggingsdk.LogGroupLifecycleStateCreating,
			wantSuccessful: true,
			wantRequeue:    true,
			wantReason:     shared.Provisioning,
		},
		{
			name:           "updating",
			state:          loggingsdk.LogGroupLifecycleStateUpdating,
			wantSuccessful: true,
			wantRequeue:    true,
			wantReason:     shared.Updating,
		},
		{
			name:           "deleting",
			state:          loggingsdk.LogGroupLifecycleStateDeleting,
			wantSuccessful: true,
			wantRequeue:    true,
			wantReason:     shared.Terminating,
		},
		{
			name:           "active",
			state:          loggingsdk.LogGroupLifecycleStateActive,
			wantSuccessful: true,
			wantRequeue:    false,
			wantReason:     shared.Active,
		},
		{
			name:           "failed",
			state:          loggingsdk.LogGroupLifecycleStateFailed,
			wantSuccessful: false,
			wantRequeue:    false,
			wantReason:     shared.Failed,
		},
		{
			name:           "inactive",
			state:          loggingsdk.LogGroupLifecycleStateInactive,
			wantSuccessful: false,
			wantRequeue:    false,
			wantReason:     shared.Failed,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			const existingID = "ocid1.loggroup.oc1..lifecycle"
			resource := makeLogGroupResource()
			resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
			resource.Status.Id = existingID

			client := newTestLogGroupClient(&fakeLogGroupOCIClient{
				getFn: func(_ context.Context, req loggingsdk.GetLogGroupRequest) (loggingsdk.GetLogGroupResponse, error) {
					requireLogGroupStringPtr(t, "get logGroupId", req.LogGroupId, existingID)
					return loggingsdk.GetLogGroupResponse{
						LogGroup: makeSDKLogGroup(existingID, resource, tc.state),
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

func TestLogGroupDeleteWaitsForWorkRequest(t *testing.T) {
	t.Parallel()

	const (
		existingID    = "ocid1.loggroup.oc1..delete"
		workRequestID = "wr-loggroup-delete"
	)
	resource := makeLogGroupResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	deleteCalls := 0

	client := newTestLogGroupClient(&fakeLogGroupOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetLogGroupRequest) (loggingsdk.GetLogGroupResponse, error) {
			requireLogGroupStringPtr(t, "get logGroupId", req.LogGroupId, existingID)
			return loggingsdk.GetLogGroupResponse{
				LogGroup: makeSDKLogGroup(existingID, resource, loggingsdk.LogGroupLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req loggingsdk.DeleteLogGroupRequest) (loggingsdk.DeleteLogGroupResponse, error) {
			deleteCalls++
			requireLogGroupStringPtr(t, "delete logGroupId", req.LogGroupId, existingID)
			return loggingsdk.DeleteLogGroupResponse{OpcWorkRequestId: common.String(workRequestID)}, nil
		},
		workRequestFn: func(_ context.Context, req loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error) {
			requireLogGroupStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return loggingsdk.GetWorkRequestResponse{
				WorkRequest: makeLogGroupWorkRequest(workRequestID, loggingsdk.OperationTypesDeleteLogGroup, loggingsdk.OperationStatusInProgress, loggingsdk.ActionTypesInProgress, existingID),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false while delete work request is pending")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteLogGroup() calls = %d, want 1", deleteCalls)
	}
	requireLogGroupAsyncCurrent(t, resource, shared.OSOKAsyncSourceWorkRequest, shared.OSOKAsyncPhaseDelete, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want Terminating", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay nil while delete confirmation is pending")
	}
}

func TestLogGroupDeleteKeepsFinalizerWhileReadbackDeleting(t *testing.T) {
	t.Parallel()

	const (
		existingID    = "ocid1.loggroup.oc1..delete-readback"
		workRequestID = "wr-loggroup-delete-readback"
	)
	resource := makeLogGroupResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	getCalls := 0

	client := newTestLogGroupClient(&fakeLogGroupOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetLogGroupRequest) (loggingsdk.GetLogGroupResponse, error) {
			getCalls++
			requireLogGroupStringPtr(t, "get logGroupId", req.LogGroupId, existingID)
			state := loggingsdk.LogGroupLifecycleStateActive
			if getCalls > 1 {
				state = loggingsdk.LogGroupLifecycleStateDeleting
			}
			return loggingsdk.GetLogGroupResponse{
				LogGroup: makeSDKLogGroup(existingID, resource, state),
			}, nil
		},
		deleteFn: func(_ context.Context, req loggingsdk.DeleteLogGroupRequest) (loggingsdk.DeleteLogGroupResponse, error) {
			requireLogGroupStringPtr(t, "delete logGroupId", req.LogGroupId, existingID)
			return loggingsdk.DeleteLogGroupResponse{OpcWorkRequestId: common.String(workRequestID)}, nil
		},
		workRequestFn: func(_ context.Context, req loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error) {
			requireLogGroupStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return loggingsdk.GetWorkRequestResponse{
				WorkRequest: makeLogGroupWorkRequest(workRequestID, loggingsdk.OperationTypesDeleteLogGroup, loggingsdk.OperationStatusSucceeded, loggingsdk.ActionTypesDeleted, existingID),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false while readback still reports DELETING")
	}
	if getCalls != 2 {
		t.Fatalf("GetLogGroup() calls = %d, want pre-delete and confirm-delete reads", getCalls)
	}
	if got := resource.Status.LifecycleState; got != string(loggingsdk.LogGroupLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay nil while readback reports DELETING")
	}
}

func TestLogGroupDeleteConfirmsReadNotFoundAfterSucceededWorkRequest(t *testing.T) {
	t.Parallel()

	const (
		existingID    = "ocid1.loggroup.oc1..delete-gone"
		workRequestID = "wr-loggroup-delete-gone"
	)
	resource := makeLogGroupResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.Id = existingID
	getCalls := 0

	client := newTestLogGroupClient(&fakeLogGroupOCIClient{
		getFn: func(_ context.Context, req loggingsdk.GetLogGroupRequest) (loggingsdk.GetLogGroupResponse, error) {
			getCalls++
			requireLogGroupStringPtr(t, "get logGroupId", req.LogGroupId, existingID)
			if getCalls > 1 {
				return loggingsdk.GetLogGroupResponse{}, errortest.NewServiceError(404, "NotFound", "LogGroup deleted")
			}
			return loggingsdk.GetLogGroupResponse{
				LogGroup: makeSDKLogGroup(existingID, resource, loggingsdk.LogGroupLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req loggingsdk.DeleteLogGroupRequest) (loggingsdk.DeleteLogGroupResponse, error) {
			requireLogGroupStringPtr(t, "delete logGroupId", req.LogGroupId, existingID)
			return loggingsdk.DeleteLogGroupResponse{OpcWorkRequestId: common.String(workRequestID)}, nil
		},
		workRequestFn: func(_ context.Context, req loggingsdk.GetWorkRequestRequest) (loggingsdk.GetWorkRequestResponse, error) {
			requireLogGroupStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return loggingsdk.GetWorkRequestResponse{
				WorkRequest: makeLogGroupWorkRequest(workRequestID, loggingsdk.OperationTypesDeleteLogGroup, loggingsdk.OperationStatusSucceeded, loggingsdk.ActionTypesDeleted, existingID),
			}, nil
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
		t.Fatalf("GetLogGroup() calls = %d, want pre-delete and confirm-delete reads", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt should be set after confirmed deletion")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want Terminating", got)
	}
}

func newTestLogGroupClient(client logGroupOCIClient) defaultLogGroupServiceClient {
	if client == nil {
		client = &fakeLogGroupOCIClient{}
	}
	hooks := newLogGroupRuntimeHooksWithOCIClient(client)
	applyLogGroupRuntimeHooks(&hooks, client, nil)
	return defaultLogGroupServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loggingv1beta1.LogGroup](
			buildLogGroupGeneratedRuntimeConfig(
				&LogGroupServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}},
				hooks,
			),
		),
	}
}

func newLogGroupRuntimeHooksWithOCIClient(client logGroupOCIClient) LogGroupRuntimeHooks {
	return LogGroupRuntimeHooks{
		Semantics: newLogGroupRuntimeSemantics(),
		Create: runtimeOperationHooks[loggingsdk.CreateLogGroupRequest, loggingsdk.CreateLogGroupResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateLogGroupDetails", RequestName: "CreateLogGroupDetails", Contribution: "body", PreferResourceID: false}},
			Call: func(ctx context.Context, request loggingsdk.CreateLogGroupRequest) (loggingsdk.CreateLogGroupResponse, error) {
				return client.CreateLogGroup(ctx, request)
			},
		},
		Get: runtimeOperationHooks[loggingsdk.GetLogGroupRequest, loggingsdk.GetLogGroupResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "LogGroupId", RequestName: "logGroupId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request loggingsdk.GetLogGroupRequest) (loggingsdk.GetLogGroupResponse, error) {
				return client.GetLogGroup(ctx, request)
			},
		},
		List: runtimeOperationHooks[loggingsdk.ListLogGroupsRequest, loggingsdk.ListLogGroupsResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
				{FieldName: "IsCompartmentIdInSubtree", RequestName: "isCompartmentIdInSubtree", Contribution: "query", PreferResourceID: false},
				{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", PreferResourceID: false},
				{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
				{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
				{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
			},
			Call: func(ctx context.Context, request loggingsdk.ListLogGroupsRequest) (loggingsdk.ListLogGroupsResponse, error) {
				return client.ListLogGroups(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loggingsdk.UpdateLogGroupRequest, loggingsdk.UpdateLogGroupResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "LogGroupId", RequestName: "logGroupId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateLogGroupDetails", RequestName: "UpdateLogGroupDetails", Contribution: "body", PreferResourceID: false},
			},
			Call: func(ctx context.Context, request loggingsdk.UpdateLogGroupRequest) (loggingsdk.UpdateLogGroupResponse, error) {
				return client.UpdateLogGroup(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loggingsdk.DeleteLogGroupRequest, loggingsdk.DeleteLogGroupResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "LogGroupId", RequestName: "logGroupId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request loggingsdk.DeleteLogGroupRequest) (loggingsdk.DeleteLogGroupResponse, error) {
				return client.DeleteLogGroup(ctx, request)
			},
		},
		WrapGeneratedClient: []func(LogGroupServiceClient) LogGroupServiceClient{},
	}
}

func makeLogGroupResource() *loggingv1beta1.LogGroup {
	return &loggingv1beta1.LogGroup{
		Spec: loggingv1beta1.LogGroupSpec{
			CompartmentId: "ocid1.compartment.oc1..loggroup",
			DisplayName:   "osok-log-group",
			Description:   "example log group",
			FreeformTags:  map[string]string{"managed-by": "osok"},
		},
	}
}

func makeSDKLogGroup(id string, resource *loggingv1beta1.LogGroup, state loggingsdk.LogGroupLifecycleStateEnum) loggingsdk.LogGroup {
	return loggingsdk.LogGroup{
		Id:             common.String(id),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		DisplayName:    common.String(resource.Spec.DisplayName),
		Description:    common.String(resource.Spec.Description),
		FreeformTags:   cloneLogGroupStringMap(resource.Spec.FreeformTags),
		LifecycleState: state,
	}
}

func makeSDKLogGroupSummary(id string, resource *loggingv1beta1.LogGroup, state loggingsdk.LogGroupLifecycleStateEnum) loggingsdk.LogGroupSummary {
	return loggingsdk.LogGroupSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		DisplayName:    common.String(resource.Spec.DisplayName),
		Description:    common.String(resource.Spec.Description),
		FreeformTags:   cloneLogGroupStringMap(resource.Spec.FreeformTags),
		LifecycleState: state,
	}
}

func makeLogGroupWorkRequest(
	id string,
	operationType loggingsdk.OperationTypesEnum,
	status loggingsdk.OperationStatusEnum,
	actionType loggingsdk.ActionTypesEnum,
	resourceID string,
) loggingsdk.WorkRequest {
	workRequest := loggingsdk.WorkRequest{
		Id:              common.String(id),
		OperationType:   operationType,
		Status:          status,
		CompartmentId:   common.String("ocid1.compartment.oc1..loggroup"),
		PercentComplete: float32Ptr(50),
	}
	if resourceID != "" {
		workRequest.Resources = []loggingsdk.WorkRequestResource{
			{
				EntityType: common.String("loggroup"),
				ActionType: actionType,
				Identifier: common.String(resourceID),
				EntityUri:  common.String("/logGroups/" + resourceID),
			},
		}
	}
	return workRequest
}

func cloneLogGroupStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func float32Ptr(value float32) *float32 {
	return &value
}

func requireLogGroupStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireLogGroupAsyncCurrent(
	t *testing.T,
	resource *loggingv1beta1.LogGroup,
	source shared.OSOKAsyncSource,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil")
	}
	if current.Source != source {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, source)
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

func assertLogGroupStringSliceEqual(t *testing.T, name string, got, want []string) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	t.Fatalf("%s = %#v, want %#v", name, got, want)
}
