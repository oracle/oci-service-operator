/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package session

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	bastionsdk "github.com/oracle/oci-go-sdk/v65/bastion"
	"github.com/oracle/oci-go-sdk/v65/common"
	bastionv1beta1 "github.com/oracle/oci-service-operator/api/bastion/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testBastionID = "ocid1.bastion.oc1..test"
	testSessionID = "ocid1.bastionsession.oc1..test"
	testPublicKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCtest"
)

type fakeSessionOCIClient struct {
	createSessionFunc   func(context.Context, bastionsdk.CreateSessionRequest) (bastionsdk.CreateSessionResponse, error)
	getSessionFunc      func(context.Context, bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error)
	listSessionsFunc    func(context.Context, bastionsdk.ListSessionsRequest) (bastionsdk.ListSessionsResponse, error)
	updateSessionFunc   func(context.Context, bastionsdk.UpdateSessionRequest) (bastionsdk.UpdateSessionResponse, error)
	deleteSessionFunc   func(context.Context, bastionsdk.DeleteSessionRequest) (bastionsdk.DeleteSessionResponse, error)
	getWorkRequestFunc  func(context.Context, bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error)
	createSessionCalls  []bastionsdk.CreateSessionRequest
	getSessionCalls     []bastionsdk.GetSessionRequest
	listSessionsCalls   []bastionsdk.ListSessionsRequest
	updateSessionCalls  []bastionsdk.UpdateSessionRequest
	deleteSessionCalls  []bastionsdk.DeleteSessionRequest
	getWorkRequestCalls []bastionsdk.GetWorkRequestRequest
}

func (f *fakeSessionOCIClient) CreateSession(ctx context.Context, request bastionsdk.CreateSessionRequest) (bastionsdk.CreateSessionResponse, error) {
	f.createSessionCalls = append(f.createSessionCalls, request)
	if f.createSessionFunc != nil {
		return f.createSessionFunc(ctx, request)
	}
	return bastionsdk.CreateSessionResponse{}, nil
}

func (f *fakeSessionOCIClient) GetSession(ctx context.Context, request bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error) {
	f.getSessionCalls = append(f.getSessionCalls, request)
	if f.getSessionFunc != nil {
		return f.getSessionFunc(ctx, request)
	}
	return bastionsdk.GetSessionResponse{}, nil
}

func (f *fakeSessionOCIClient) ListSessions(ctx context.Context, request bastionsdk.ListSessionsRequest) (bastionsdk.ListSessionsResponse, error) {
	f.listSessionsCalls = append(f.listSessionsCalls, request)
	if f.listSessionsFunc != nil {
		return f.listSessionsFunc(ctx, request)
	}
	return bastionsdk.ListSessionsResponse{}, nil
}

func (f *fakeSessionOCIClient) UpdateSession(ctx context.Context, request bastionsdk.UpdateSessionRequest) (bastionsdk.UpdateSessionResponse, error) {
	f.updateSessionCalls = append(f.updateSessionCalls, request)
	if f.updateSessionFunc != nil {
		return f.updateSessionFunc(ctx, request)
	}
	return bastionsdk.UpdateSessionResponse{}, nil
}

func (f *fakeSessionOCIClient) DeleteSession(ctx context.Context, request bastionsdk.DeleteSessionRequest) (bastionsdk.DeleteSessionResponse, error) {
	f.deleteSessionCalls = append(f.deleteSessionCalls, request)
	if f.deleteSessionFunc != nil {
		return f.deleteSessionFunc(ctx, request)
	}
	return bastionsdk.DeleteSessionResponse{}, nil
}

func (f *fakeSessionOCIClient) GetWorkRequest(ctx context.Context, request bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error) {
	f.getWorkRequestCalls = append(f.getWorkRequestCalls, request)
	if f.getWorkRequestFunc != nil {
		return f.getWorkRequestFunc(ctx, request)
	}
	return bastionsdk.GetWorkRequestResponse{}, nil
}

func TestSessionCreateOrUpdateCreatesWithWorkRequestTracking(t *testing.T) {
	resource := newTestSession("session-create")
	fake := &fakeSessionOCIClient{
		listSessionsFunc: func(_ context.Context, _ bastionsdk.ListSessionsRequest) (bastionsdk.ListSessionsResponse, error) {
			return bastionsdk.ListSessionsResponse{}, nil
		},
		createSessionFunc:  createSessionWithWorkRequest(t, "session-create", "wr-create-1", "opc-create-1"),
		getWorkRequestFunc: getSessionWorkRequestByID(t, "wr-create-1", newSessionWorkRequest("wr-create-1", bastionsdk.OperationTypeCreateSession, bastionsdk.OperationStatusInProgress, testSessionID)),
	}

	response, err := newTestSessionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if response.IsSuccessful != true || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while work request is pending", response)
	}
	if got := len(fake.createSessionCalls); got != 1 {
		t.Fatalf("CreateSession() calls = %d, want 1", got)
	}
	if got := resource.Status.OsokStatus.Ocid; got != shared.OCID(testSessionID) {
		t.Fatalf("status.status.ocid = %q, want %q", got, testSessionID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.WorkRequestID != "wr-create-1" || current.Phase != shared.OSOKAsyncPhaseCreate {
		t.Fatalf("status.status.async.current = %#v, want create work request wr-create-1", current)
	}
}

func TestSessionCreateOrUpdateBindsExistingSessionFromSecondListPage(t *testing.T) {
	resource := newTestSession("session-bind")
	fake := &fakeSessionOCIClient{
		listSessionsFunc: func(_ context.Context, request bastionsdk.ListSessionsRequest) (bastionsdk.ListSessionsResponse, error) {
			if request.Page == nil {
				return bastionsdk.ListSessionsResponse{OpcNextPage: common.String("page-2")}, nil
			}
			return bastionsdk.ListSessionsResponse{
				Items: []bastionsdk.SessionSummary{
					newSDKSessionSummary(testSessionID, "session-bind", bastionsdk.SessionLifecycleStateActive),
				},
				OpcRequestId: common.String("opc-list-2"),
			}, nil
		},
		getSessionFunc: func(_ context.Context, request bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error) {
			if got, want := stringPointerValue(request.SessionId), testSessionID; got != want {
				t.Fatalf("GetSession() SessionId = %q, want %q", got, want)
			}
			return bastionsdk.GetSessionResponse{
				Session:      newSDKSession(testSessionID, "session-bind", bastionsdk.SessionLifecycleStateActive),
				OpcRequestId: common.String("opc-get-1"),
			}, nil
		},
	}

	response, err := newTestSessionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no requeue", response)
	}
	if got := len(fake.listSessionsCalls); got != 2 {
		t.Fatalf("ListSessions() calls = %d, want 2", got)
	}
	if got := len(fake.createSessionCalls); got != 0 {
		t.Fatalf("CreateSession() calls = %d, want 0", got)
	}
	if got := resource.Status.Id; got != testSessionID {
		t.Fatalf("status.id = %q, want %q", got, testSessionID)
	}
	if got := resource.Status.OsokStatus.Ocid; got != shared.OCID(testSessionID) {
		t.Fatalf("status.status.ocid = %q, want %q", got, testSessionID)
	}
}

func TestSessionCreateOrUpdateUpdatesMutableDisplayName(t *testing.T) {
	resource := newTestSession("session-new")
	resource.Status.Id = testSessionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSessionID)

	getCalls := 0
	fake := &fakeSessionOCIClient{
		getSessionFunc: func(_ context.Context, _ bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error) {
			getCalls++
			if getCalls == 1 {
				return bastionsdk.GetSessionResponse{
					Session: newSDKSession(testSessionID, "session-old", bastionsdk.SessionLifecycleStateActive),
				}, nil
			}
			return bastionsdk.GetSessionResponse{
				Session: newSDKSession(testSessionID, "session-new", bastionsdk.SessionLifecycleStateActive),
			}, nil
		},
		updateSessionFunc: func(_ context.Context, request bastionsdk.UpdateSessionRequest) (bastionsdk.UpdateSessionResponse, error) {
			if got, want := stringPointerValue(request.SessionId), testSessionID; got != want {
				t.Fatalf("UpdateSession() SessionId = %q, want %q", got, want)
			}
			if got, want := stringPointerValue(request.DisplayName), "session-new"; got != want {
				t.Fatalf("UpdateSession() DisplayName = %q, want %q", got, want)
			}
			return bastionsdk.UpdateSessionResponse{
				Session:      newSDKSession(testSessionID, "session-new", bastionsdk.SessionLifecycleStateActive),
				OpcRequestId: common.String("opc-update-1"),
			}, nil
		},
	}

	response, err := newTestSessionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no requeue", response)
	}
	if got := len(fake.updateSessionCalls); got != 1 {
		t.Fatalf("UpdateSession() calls = %d, want 1", got)
	}
	if got := resource.Status.DisplayName; got != "session-new" {
		t.Fatalf("status.displayName = %q, want session-new", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestSessionCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := newTestSession("session-drift")
	resource.Spec.BastionId = "ocid1.bastion.oc1..new"
	resource.Status.Id = testSessionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSessionID)

	fake := &fakeSessionOCIClient{
		getSessionFunc: func(_ context.Context, _ bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error) {
			return bastionsdk.GetSessionResponse{
				Session: newSDKSession(testSessionID, "session-drift", bastionsdk.SessionLifecycleStateActive),
			}, nil
		},
	}

	_, err := newTestSessionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when bastionId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want bastionId replacement drift", err)
	}
	if got := len(fake.updateSessionCalls); got != 0 {
		t.Fatalf("UpdateSession() calls = %d, want 0", got)
	}
}

func TestSessionCreateOrUpdateAcceptsMatchingTargetResourceJSONData(t *testing.T) {
	resource := newTestSessionWithTargetJSONData("session-json", "ocid1.instance.oc1..target")
	resource.Status.Id = testSessionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSessionID)

	fake := &fakeSessionOCIClient{
		getSessionFunc: func(_ context.Context, _ bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error) {
			return bastionsdk.GetSessionResponse{
				Session: newSDKSession(testSessionID, "session-json", bastionsdk.SessionLifecycleStateActive),
			}, nil
		},
	}

	response, err := newTestSessionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no requeue", response)
	}
	if got := len(fake.updateSessionCalls); got != 0 {
		t.Fatalf("UpdateSession() calls = %d, want 0", got)
	}
	if got := resource.Spec.TargetResourceDetails.JsonData; got != "" {
		t.Fatalf("spec.targetResourceDetails.jsonData = %q, want normalized empty value", got)
	}
}

func TestSessionCreateOrUpdateRejectsTargetResourceJSONDataDriftBeforeUpdate(t *testing.T) {
	resource := newTestSessionWithTargetJSONData("session-json-drift", "ocid1.instance.oc1..drifted")
	resource.Status.Id = testSessionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSessionID)

	fake := &fakeSessionOCIClient{
		getSessionFunc: func(_ context.Context, _ bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error) {
			return bastionsdk.GetSessionResponse{
				Session: newSDKSession(testSessionID, "session-json-drift", bastionsdk.SessionLifecycleStateActive),
			}, nil
		},
	}

	_, err := newTestSessionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "targetResourceDetails.jsonData changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want targetResourceDetails.jsonData drift", err)
	}
	if got := len(fake.updateSessionCalls); got != 0 {
		t.Fatalf("UpdateSession() calls = %d, want 0", got)
	}
}

func TestSessionCreateOrUpdateRejectsTargetResourceJSONDataWhenReadbackIsMissing(t *testing.T) {
	resource := newTestSessionWithTargetJSONData("session-json-missing-readback", "ocid1.instance.oc1..target")
	resource.Status.Id = testSessionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSessionID)

	fake := &fakeSessionOCIClient{
		getSessionFunc: func(_ context.Context, _ bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error) {
			return bastionsdk.GetSessionResponse{
				Session: bastionsdk.Session{
					Id:             common.String(testSessionID),
					BastionId:      common.String(testBastionID),
					LifecycleState: bastionsdk.SessionLifecycleStateActive,
					DisplayName:    common.String("session-json-missing-readback"),
				},
			}, nil
		},
	}

	_, err := newTestSessionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "current targetResourceDetails is empty") {
		t.Fatalf("CreateOrUpdate() error = %v, want missing targetResourceDetails readback", err)
	}
	if got := len(fake.updateSessionCalls); got != 0 {
		t.Fatalf("UpdateSession() calls = %d, want 0", got)
	}
}

func TestSessionCreateOrUpdateObservesPendingCreateWorkRequestBeforeTargetJSONDataRead(t *testing.T) {
	resource := newTestSessionWithTargetJSONData("session-json-pending-create", "ocid1.instance.oc1..target")
	resource.Status.Id = testSessionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSessionID)
	markTestWorkRequest(resource, shared.OSOKAsyncPhaseCreate, "wr-create-1")

	fake := &fakeSessionOCIClient{
		getSessionFunc: func(context.Context, bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error) {
			t.Fatal("GetSession() called before pending create work request was observed")
			return bastionsdk.GetSessionResponse{}, nil
		},
		listSessionsFunc: func(context.Context, bastionsdk.ListSessionsRequest) (bastionsdk.ListSessionsResponse, error) {
			t.Fatal("ListSessions() called before pending create work request was observed")
			return bastionsdk.ListSessionsResponse{}, nil
		},
		createSessionFunc: func(context.Context, bastionsdk.CreateSessionRequest) (bastionsdk.CreateSessionResponse, error) {
			t.Fatal("CreateSession() called while pending create work request was tracked")
			return bastionsdk.CreateSessionResponse{}, nil
		},
		getWorkRequestFunc: getSessionWorkRequestByID(t, "wr-create-1", newSessionWorkRequest("wr-create-1", bastionsdk.OperationTypeCreateSession, bastionsdk.OperationStatusInProgress, testSessionID)),
	}

	response, err := newTestSessionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while work request is pending", response)
	}
	if got := len(fake.getWorkRequestCalls); got != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", got)
	}
	if got := len(fake.getSessionCalls); got != 0 {
		t.Fatalf("GetSession() calls = %d, want 0 before pending create is observed", got)
	}
	if got := len(fake.createSessionCalls); got != 0 {
		t.Fatalf("CreateSession() calls = %d, want 0 while pending create is tracked", got)
	}
}

func TestSessionCreateOrUpdateLetsStaleTrackedNotFoundWithTargetJSONDataRecover(t *testing.T) {
	resource := newTestSessionWithTargetJSONData("session-json-stale-tracked", "ocid1.instance.oc1..target")
	resource.Status.Id = "ocid1.bastionsession.oc1..stale"
	resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	notFoundErr := errortest.NewServiceError(404, errorutil.NotFound, "session not found")

	fake := &fakeSessionOCIClient{
		getSessionFunc: func(_ context.Context, request bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error) {
			if got, want := stringPointerValue(request.SessionId), "ocid1.bastionsession.oc1..stale"; got != want {
				t.Fatalf("GetSession() SessionId = %q, want %q", got, want)
			}
			return bastionsdk.GetSessionResponse{}, notFoundErr
		},
		listSessionsFunc: func(context.Context, bastionsdk.ListSessionsRequest) (bastionsdk.ListSessionsResponse, error) {
			return bastionsdk.ListSessionsResponse{}, nil
		},
		createSessionFunc:  createSessionWithWorkRequest(t, "session-json-stale-tracked", "wr-create-1", "opc-create-1"),
		getWorkRequestFunc: getSessionWorkRequestByID(t, "wr-create-1", newSessionWorkRequest("wr-create-1", bastionsdk.OperationTypeCreateSession, bastionsdk.OperationStatusInProgress, testSessionID)),
	}

	response, err := newTestSessionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue after recreating stale tracked identity", response)
	}
	if got := len(fake.getSessionCalls); got != 2 {
		t.Fatalf("GetSession() calls = %d, want wrapper validation plus generatedruntime stale-id check", got)
	}
	if got := len(fake.createSessionCalls); got != 1 {
		t.Fatalf("CreateSession() calls = %d, want 1 after stale tracked identity is cleared", got)
	}
	if got := len(fake.getWorkRequestCalls); got != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", got)
	}
	if got := resource.Status.Id; got != testSessionID {
		t.Fatalf("status.id = %q, want %q", got, testSessionID)
	}
	if got := resource.Status.OsokStatus.Ocid; got != shared.OCID(testSessionID) {
		t.Fatalf("status.status.ocid = %q, want %q", got, testSessionID)
	}
}

func TestSessionDeleteStartsWorkRequestAndRetainsFinalizer(t *testing.T) {
	resource := newTestSession("session-delete")
	resource.Status.Id = testSessionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSessionID)

	fake := &fakeSessionOCIClient{
		getSessionFunc: func(_ context.Context, _ bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error) {
			return bastionsdk.GetSessionResponse{
				Session: newSDKSession(testSessionID, "session-delete", bastionsdk.SessionLifecycleStateActive),
			}, nil
		},
		deleteSessionFunc: func(_ context.Context, request bastionsdk.DeleteSessionRequest) (bastionsdk.DeleteSessionResponse, error) {
			if got, want := stringPointerValue(request.SessionId), testSessionID; got != want {
				t.Fatalf("DeleteSession() SessionId = %q, want %q", got, want)
			}
			return bastionsdk.DeleteSessionResponse{
				OpcWorkRequestId: common.String("wr-delete-1"),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
		getWorkRequestFunc: func(_ context.Context, _ bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error) {
			return bastionsdk.GetWorkRequestResponse{
				WorkRequest: newSessionWorkRequest("wr-delete-1", bastionsdk.OperationTypeDeleteSession, bastionsdk.OperationStatusInProgress, testSessionID),
			}, nil
		},
	}

	deleted, err := newTestSessionClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete work request is pending")
	}
	if got := len(fake.deleteSessionCalls); got != 1 {
		t.Fatalf("DeleteSession() calls = %d, want 1", got)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.WorkRequestID != "wr-delete-1" || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current = %#v, want delete work request wr-delete-1", current)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}
}

func TestSessionDeleteObservesPendingCreateWorkRequestBeforeDelete(t *testing.T) {
	requireSessionDeleteObservesPendingWriteWorkRequest(
		t,
		"session-delete-pending-create",
		shared.OSOKAsyncPhaseCreate,
		"wr-create-1",
		bastionsdk.OperationTypeCreateSession,
	)
}

func TestSessionDeleteObservesPendingUpdateWorkRequestBeforeDelete(t *testing.T) {
	requireSessionDeleteObservesPendingWriteWorkRequest(
		t,
		"session-delete-pending-update",
		shared.OSOKAsyncPhaseUpdate,
		"wr-update-1",
		bastionsdk.OperationTypeEnum("UPDATE_SESSION"),
	)
}

func requireSessionDeleteObservesPendingWriteWorkRequest(
	t *testing.T,
	resourceName string,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	operation bastionsdk.OperationTypeEnum,
) {
	t.Helper()
	resource := newTestSession(resourceName)
	resource.Status.Id = testSessionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSessionID)
	markTestWorkRequest(resource, phase, workRequestID)

	fake := &fakeSessionOCIClient{
		getWorkRequestFunc: func(_ context.Context, request bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error) {
			if got := stringPointerValue(request.WorkRequestId); got != workRequestID {
				t.Fatalf("GetWorkRequest() id = %q, want %q", got, workRequestID)
			}
			return bastionsdk.GetWorkRequestResponse{
				WorkRequest: newSessionWorkRequest(workRequestID, operation, bastionsdk.OperationStatusInProgress, testSessionID),
			}, nil
		},
	}

	deleted, err := newTestSessionClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatalf("Delete() deleted = true, want false while %s work request is pending", phase)
	}
	if got := len(fake.deleteSessionCalls); got != 0 {
		t.Fatalf("DeleteSession() calls = %d, want 0", got)
	}
	if got := len(fake.getSessionCalls); got != 0 {
		t.Fatalf("GetSession() calls = %d, want 0 before pending %s is observed", got, phase)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.WorkRequestID != workRequestID || current.Phase != phase {
		t.Fatalf("status.status.async.current = %#v, want pending %s work request %s", current, phase, workRequestID)
	}
}

func TestSessionDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	resource := newTestSession("session-delete-auth")
	resource.Status.Id = testSessionID
	resource.Status.OsokStatus.Ocid = shared.OCID(testSessionID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	fake := &fakeSessionOCIClient{
		getSessionFunc: func(_ context.Context, _ bastionsdk.GetSessionRequest) (bastionsdk.GetSessionResponse, error) {
			return bastionsdk.GetSessionResponse{}, authErr
		},
	}

	deleted, err := newTestSessionClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped 404", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped confirm read")
	}
	if got := len(fake.deleteSessionCalls); got != 0 {
		t.Fatalf("DeleteSession() calls = %d, want 0", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestSessionCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	resource := newTestSession("session-error")
	fake := &fakeSessionOCIClient{
		listSessionsFunc: func(_ context.Context, _ bastionsdk.ListSessionsRequest) (bastionsdk.ListSessionsResponse, error) {
			return bastionsdk.ListSessionsResponse{}, nil
		},
		createSessionFunc: func(_ context.Context, _ bastionsdk.CreateSessionRequest) (bastionsdk.CreateSessionResponse, error) {
			return bastionsdk.CreateSessionResponse{}, errortest.NewServiceError(500, "InternalError", "request failed")
		},
	}

	response, err := newTestSessionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI service error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func newTestSessionClient(fake *fakeSessionOCIClient) SessionServiceClient {
	return newSessionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func createSessionWithWorkRequest(
	t *testing.T,
	displayName string,
	workRequestID string,
	opcRequestID string,
) func(context.Context, bastionsdk.CreateSessionRequest) (bastionsdk.CreateSessionResponse, error) {
	t.Helper()
	return func(_ context.Context, request bastionsdk.CreateSessionRequest) (bastionsdk.CreateSessionResponse, error) {
		requireCreateSessionRequest(t, request)
		return bastionsdk.CreateSessionResponse{
			Session:          newSDKSession(testSessionID, displayName, bastionsdk.SessionLifecycleStateCreating),
			OpcWorkRequestId: common.String(workRequestID),
			OpcRequestId:     common.String(opcRequestID),
		}, nil
	}
}

func requireCreateSessionRequest(t *testing.T, request bastionsdk.CreateSessionRequest) {
	t.Helper()
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatalf("CreateSession() OpcRetryToken is empty")
	}
	if got, want := stringPointerValue(request.BastionId), testBastionID; got != want {
		t.Fatalf("CreateSession() BastionId = %q, want %q", got, want)
	}
	if _, ok := request.TargetResourceDetails.(bastionsdk.CreateManagedSshSessionTargetResourceDetails); !ok {
		t.Fatalf("CreateSession() TargetResourceDetails = %T, want CreateManagedSshSessionTargetResourceDetails", request.TargetResourceDetails)
	}
}

func getSessionWorkRequestByID(
	t *testing.T,
	workRequestID string,
	workRequest bastionsdk.WorkRequest,
) func(context.Context, bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error) {
	t.Helper()
	return func(_ context.Context, request bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error) {
		if got := stringPointerValue(request.WorkRequestId); got != workRequestID {
			t.Fatalf("GetWorkRequest() id = %q, want %q", got, workRequestID)
		}
		return bastionsdk.GetWorkRequestResponse{WorkRequest: workRequest}, nil
	}
}

func newTestSession(name string) *bastionv1beta1.Session {
	return &bastionv1beta1.Session{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       types.UID("uid-" + name),
		},
		Spec: bastionv1beta1.SessionSpec{
			BastionId: testBastionID,
			TargetResourceDetails: bastionv1beta1.SessionTargetResourceDetails{
				SessionType:                           "MANAGED_SSH",
				TargetResourceOperatingSystemUserName: "opc",
				TargetResourceId:                      "ocid1.instance.oc1..target",
				TargetResourcePrivateIpAddress:        "10.0.0.10",
				TargetResourcePort:                    22,
			},
			KeyDetails: bastionv1beta1.SessionKeyDetails{
				PublicKeyContent: testPublicKey,
			},
			DisplayName:         name,
			KeyType:             "PUB",
			SessionTtlInSeconds: 3600,
		},
	}
}

func newTestSessionWithTargetJSONData(name string, targetResourceID string) *bastionv1beta1.Session {
	resource := newTestSession(name)
	resource.Spec.TargetResourceDetails = bastionv1beta1.SessionTargetResourceDetails{
		JsonData: `{
			"sessionType":"MANAGED_SSH",
			"targetResourceOperatingSystemUserName":"opc",
			"targetResourceId":"` + targetResourceID + `",
			"targetResourcePrivateIpAddress":"10.0.0.10",
			"targetResourcePort":22
		}`,
	}
	return resource
}

func markTestWorkRequest(resource *bastionv1beta1.Session, phase shared.OSOKAsyncPhase, workRequestID string) {
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		RawStatus:       string(bastionsdk.OperationStatusAccepted),
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
}

func newSDKSession(id string, displayName string, state bastionsdk.SessionLifecycleStateEnum) bastionsdk.Session {
	return bastionsdk.Session{
		Id:          common.String(id),
		BastionId:   common.String(testBastionID),
		BastionName: common.String("bastion"),
		TargetResourceDetails: bastionsdk.ManagedSshSessionTargetResourceDetails{
			TargetResourceOperatingSystemUserName: common.String("opc"),
			TargetResourceId:                      common.String("ocid1.instance.oc1..target"),
			TargetResourceDisplayName:             common.String("target"),
			TargetResourcePrivateIpAddress:        common.String("10.0.0.10"),
			TargetResourcePort:                    common.Int(22),
		},
		KeyDetails:          &bastionsdk.PublicKeyDetails{PublicKeyContent: common.String(testPublicKey)},
		LifecycleState:      state,
		SessionTtlInSeconds: common.Int(3600),
		DisplayName:         common.String(displayName),
		KeyType:             bastionsdk.SessionKeyTypePub,
		LifecycleDetails:    common.String(string(state)),
	}
}

func newSDKSessionSummary(id string, displayName string, state bastionsdk.SessionLifecycleStateEnum) bastionsdk.SessionSummary {
	return bastionsdk.SessionSummary{
		Id:          common.String(id),
		BastionId:   common.String(testBastionID),
		BastionName: common.String("bastion"),
		TargetResourceDetails: bastionsdk.ManagedSshSessionTargetResourceDetails{
			TargetResourceOperatingSystemUserName: common.String("opc"),
			TargetResourceId:                      common.String("ocid1.instance.oc1..target"),
			TargetResourceDisplayName:             common.String("target"),
		},
		LifecycleState:      state,
		SessionTtlInSeconds: common.Int(3600),
		DisplayName:         common.String(displayName),
	}
}

func newSessionWorkRequest(
	id string,
	operation bastionsdk.OperationTypeEnum,
	status bastionsdk.OperationStatusEnum,
	resourceID string,
) bastionsdk.WorkRequest {
	return bastionsdk.WorkRequest{
		OperationType: operation,
		Status:        status,
		Id:            common.String(id),
		Resources: []bastionsdk.WorkRequestResource{{
			EntityType: common.String("session"),
			ActionType: sessionWorkRequestAction(operation),
			Identifier: common.String(resourceID),
		}},
	}
}

func sessionWorkRequestAction(operation bastionsdk.OperationTypeEnum) bastionsdk.ActionTypeEnum {
	switch operation {
	case bastionsdk.OperationTypeDeleteSession:
		return bastionsdk.ActionTypeDeleted
	default:
		return bastionsdk.ActionTypeCreated
	}
}
