/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package operationsinsightswarehouseuser

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testWarehouseUserID         = "ocid1.opsiwarehouseuser.oc1..test"
	testWarehouseUserOtherID    = "ocid1.opsiwarehouseuser.oc1..other"
	testWarehouseID             = "ocid1.opsiwarehouse.oc1..test"
	testWarehouseCompartmentID  = "ocid1.compartment.oc1..test"
	testWarehouseUserName       = "warehouse_user"
	testWarehouseUserPassword   = "secret-password"
	testWarehouseUserWorkID     = "ocid1.workrequest.oc1..warehouseuser"
	testWarehouseUserRequestID  = "opc-request-warehouse-user"
	testWarehouseUserRequestID2 = "opc-request-warehouse-user-2"
)

type fakeOperationsInsightsWarehouseUserOCI struct {
	t *testing.T

	createFn         func(context.Context, opsisdk.CreateOperationsInsightsWarehouseUserRequest) (opsisdk.CreateOperationsInsightsWarehouseUserResponse, error)
	getFn            func(context.Context, opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error)
	listFn           func(context.Context, opsisdk.ListOperationsInsightsWarehouseUsersRequest) (opsisdk.ListOperationsInsightsWarehouseUsersResponse, error)
	updateFn         func(context.Context, opsisdk.UpdateOperationsInsightsWarehouseUserRequest) (opsisdk.UpdateOperationsInsightsWarehouseUserResponse, error)
	deleteFn         func(context.Context, opsisdk.DeleteOperationsInsightsWarehouseUserRequest) (opsisdk.DeleteOperationsInsightsWarehouseUserResponse, error)
	getWorkRequestFn func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

func (f *fakeOperationsInsightsWarehouseUserOCI) CreateOperationsInsightsWarehouseUser(ctx context.Context, request opsisdk.CreateOperationsInsightsWarehouseUserRequest) (opsisdk.CreateOperationsInsightsWarehouseUserResponse, error) {
	f.t.Helper()
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	f.t.Fatalf("CreateOperationsInsightsWarehouseUser() was called unexpectedly with %#v", request)
	return opsisdk.CreateOperationsInsightsWarehouseUserResponse{}, nil
}

func (f *fakeOperationsInsightsWarehouseUserOCI) GetOperationsInsightsWarehouseUser(ctx context.Context, request opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
	f.t.Helper()
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	f.t.Fatalf("GetOperationsInsightsWarehouseUser() was called unexpectedly with %#v", request)
	return opsisdk.GetOperationsInsightsWarehouseUserResponse{}, nil
}

func (f *fakeOperationsInsightsWarehouseUserOCI) ListOperationsInsightsWarehouseUsers(ctx context.Context, request opsisdk.ListOperationsInsightsWarehouseUsersRequest) (opsisdk.ListOperationsInsightsWarehouseUsersResponse, error) {
	f.t.Helper()
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	f.t.Fatalf("ListOperationsInsightsWarehouseUsers() was called unexpectedly with %#v", request)
	return opsisdk.ListOperationsInsightsWarehouseUsersResponse{}, nil
}

func (f *fakeOperationsInsightsWarehouseUserOCI) UpdateOperationsInsightsWarehouseUser(ctx context.Context, request opsisdk.UpdateOperationsInsightsWarehouseUserRequest) (opsisdk.UpdateOperationsInsightsWarehouseUserResponse, error) {
	f.t.Helper()
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	f.t.Fatalf("UpdateOperationsInsightsWarehouseUser() was called unexpectedly with %#v", request)
	return opsisdk.UpdateOperationsInsightsWarehouseUserResponse{}, nil
}

func (f *fakeOperationsInsightsWarehouseUserOCI) DeleteOperationsInsightsWarehouseUser(ctx context.Context, request opsisdk.DeleteOperationsInsightsWarehouseUserRequest) (opsisdk.DeleteOperationsInsightsWarehouseUserResponse, error) {
	f.t.Helper()
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	f.t.Fatalf("DeleteOperationsInsightsWarehouseUser() was called unexpectedly with %#v", request)
	return opsisdk.DeleteOperationsInsightsWarehouseUserResponse{}, nil
}

func (f *fakeOperationsInsightsWarehouseUserOCI) GetWorkRequest(ctx context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
	f.t.Helper()
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, request)
	}
	f.t.Fatalf("GetWorkRequest() was called unexpectedly with %#v", request)
	return opsisdk.GetWorkRequestResponse{}, nil
}

func TestOperationsInsightsWarehouseUserRuntimeHooksEncodeReviewedContract(t *testing.T) {
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	hooks := newOperationsInsightsWarehouseUserRuntimeHooksWithOCIClient(fake)
	applyOperationsInsightsWarehouseUserRuntimeHooks(&hooks, fake, nil, testOperationsInsightsWarehouseUserLogger())

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed runtime semantics")
	}
	if got := hooks.Semantics.Async.Strategy; got != "workrequest" {
		t.Fatalf("hooks.Semantics.Async.Strategy = %q, want workrequest", got)
	}
	if hooks.BuildCreateBody == nil || hooks.BuildUpdateBody == nil {
		t.Fatal("create/update body hooks are not configured")
	}
	if hooks.Async.GetWorkRequest == nil || hooks.Async.RecoverResourceID == nil || hooks.Async.ResolvePhase == nil {
		t.Fatal("work-request hooks are incomplete")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("delete error hook is nil, want conservative auth-shaped 404 handling")
	}
}

func TestOperationsInsightsWarehouseUserCreateBodyShapesReviewedContract(t *testing.T) {
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	hooks := newOperationsInsightsWarehouseUserRuntimeHooksWithOCIClient(fake)
	applyOperationsInsightsWarehouseUserRuntimeHooks(&hooks, fake, nil, testOperationsInsightsWarehouseUserLogger())
	resource := newTestOperationsInsightsWarehouseUser()
	resource.Spec.IsAwrDataAccess = false
	resource.Spec.IsEmDataAccess = false
	resource.Spec.IsOpsiDataAccess = true
	resource.Spec.FreeformTags = map[string]string{"env": "test"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}

	body, err := hooks.BuildCreateBody(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details := requireOperationsInsightsWarehouseUserCreateDetails(t, body)
	assertOperationsInsightsWarehouseUserCreateAccessBody(t, details)
	requireOperationsInsightsWarehouseUserDefinedTag(t, details.DefinedTags, "Operations", "CostCenter", "42")
}

func TestOperationsInsightsWarehouseUserCreateCompletesWorkRequestAndRecordsStatus(t *testing.T) {
	resource := newTestOperationsInsightsWarehouseUser()
	var createRequest opsisdk.CreateOperationsInsightsWarehouseUserRequest
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	fake.listFn = func(_ context.Context, request opsisdk.ListOperationsInsightsWarehouseUsersRequest) (opsisdk.ListOperationsInsightsWarehouseUsersResponse, error) {
		requireStringPtr(t, "ListOperationsInsightsWarehouseUsersRequest.OperationsInsightsWarehouseId", request.OperationsInsightsWarehouseId, testWarehouseID)
		requireStringPtr(t, "ListOperationsInsightsWarehouseUsersRequest.CompartmentId", request.CompartmentId, testWarehouseCompartmentID)
		requireStringPtr(t, "ListOperationsInsightsWarehouseUsersRequest.DisplayName", request.DisplayName, testWarehouseUserName)
		return opsisdk.ListOperationsInsightsWarehouseUsersResponse{}, nil
	}
	fake.createFn = func(_ context.Context, request opsisdk.CreateOperationsInsightsWarehouseUserRequest) (opsisdk.CreateOperationsInsightsWarehouseUserResponse, error) {
		createRequest = request
		if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
			t.Fatal("CreateOperationsInsightsWarehouseUser opc retry token is empty")
		}
		if request.ConnectionPassword == nil || *request.ConnectionPassword != testWarehouseUserPassword {
			t.Fatal("CreateOperationsInsightsWarehouseUser did not pass the desired password")
		}
		return opsisdk.CreateOperationsInsightsWarehouseUserResponse{
			OpcWorkRequestId: common.String(testWarehouseUserWorkID),
			OpcRequestId:     common.String(testWarehouseUserRequestID),
		}, nil
	}
	fake.getWorkRequestFn = func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testWarehouseUserWorkID)
		return opsisdk.GetWorkRequestResponse{
			WorkRequest: makeWarehouseUserWorkRequest(testWarehouseUserWorkID, opsisdk.OperationStatusSucceeded, opsisdk.OperationTypeCreateOpsiWarehouseUser, opsisdk.ActionTypeCreated, testWarehouseUserID),
		}, nil
	}
	fake.getFn = func(_ context.Context, request opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
		requireStringPtr(t, "GetOperationsInsightsWarehouseUserRequest.OperationsInsightsWarehouseUserId", request.OperationsInsightsWarehouseUserId, testWarehouseUserID)
		return opsisdk.GetOperationsInsightsWarehouseUserResponse{
			OperationsInsightsWarehouseUser: makeSDKOperationsInsightsWarehouseUser(testWarehouseUserID, resource.Spec, opsisdk.OperationsInsightsWarehouseUserLifecycleStateActive),
		}, nil
	}

	response, err := newTestOperationsInsightsWarehouseUserClient(t, fake).CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseUserRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	requireStringPtr(t, "CreateOperationsInsightsWarehouseUserRequest.OperationsInsightsWarehouseId", createRequest.OperationsInsightsWarehouseId, testWarehouseID)
	assertOperationsInsightsWarehouseUserStatusID(t, resource, testWarehouseUserID)
	if got := resource.Status.OsokStatus.OpcRequestID; got != testWarehouseUserRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, testWarehouseUserRequestID)
	}
	if got := resource.Status.ConnectionPassword; got != "" {
		t.Fatalf("status.connectionPassword = %q, want scrubbed", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after succeeded create work request", resource.Status.OsokStatus.Async.Current)
	}
}

func TestOperationsInsightsWarehouseUserBindsFromSecondListPageWithoutCreating(t *testing.T) {
	resource := newTestOperationsInsightsWarehouseUser()
	listCalls := 0
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	fake.listFn = func(_ context.Context, request opsisdk.ListOperationsInsightsWarehouseUsersRequest) (opsisdk.ListOperationsInsightsWarehouseUsersResponse, error) {
		listCalls++
		switch listCalls {
		case 1:
			if request.Page != nil {
				t.Fatalf("first list page = %q, want nil", operationsInsightsWarehouseUserString(request.Page))
			}
			return opsisdk.ListOperationsInsightsWarehouseUsersResponse{
				OperationsInsightsWarehouseUserSummaryCollection: opsisdk.OperationsInsightsWarehouseUserSummaryCollection{
					Items: []opsisdk.OperationsInsightsWarehouseUserSummary{
						makeSDKOperationsInsightsWarehouseUserSummary(testWarehouseUserOtherID, "other-user", resource.Spec, opsisdk.OperationsInsightsWarehouseUserLifecycleStateActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case 2:
			requireStringPtr(t, "second list page", request.Page, "page-2")
			return opsisdk.ListOperationsInsightsWarehouseUsersResponse{
				OperationsInsightsWarehouseUserSummaryCollection: opsisdk.OperationsInsightsWarehouseUserSummaryCollection{
					Items: []opsisdk.OperationsInsightsWarehouseUserSummary{
						makeSDKOperationsInsightsWarehouseUserSummary(testWarehouseUserID, resource.Spec.Name, resource.Spec, opsisdk.OperationsInsightsWarehouseUserLifecycleStateActive),
					},
				},
			}, nil
		default:
			t.Fatalf("ListOperationsInsightsWarehouseUsers calls = %d, want 2", listCalls)
			return opsisdk.ListOperationsInsightsWarehouseUsersResponse{}, nil
		}
	}
	fake.createFn = func(context.Context, opsisdk.CreateOperationsInsightsWarehouseUserRequest) (opsisdk.CreateOperationsInsightsWarehouseUserResponse, error) {
		t.Fatal("CreateOperationsInsightsWarehouseUser should not be called when list finds an existing user")
		return opsisdk.CreateOperationsInsightsWarehouseUserResponse{}, nil
	}
	fake.getFn = func(_ context.Context, request opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
		requireStringPtr(t, "GetOperationsInsightsWarehouseUserRequest.OperationsInsightsWarehouseUserId", request.OperationsInsightsWarehouseUserId, testWarehouseUserID)
		return opsisdk.GetOperationsInsightsWarehouseUserResponse{
			OperationsInsightsWarehouseUser: makeSDKOperationsInsightsWarehouseUser(testWarehouseUserID, resource.Spec, opsisdk.OperationsInsightsWarehouseUserLifecycleStateActive),
		}, nil
	}

	response, err := newTestOperationsInsightsWarehouseUserClient(t, fake).CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseUserRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if listCalls != 2 {
		t.Fatalf("ListOperationsInsightsWarehouseUsers calls = %d, want 2", listCalls)
	}
	assertOperationsInsightsWarehouseUserStatusID(t, resource, testWarehouseUserID)
}

func TestOperationsInsightsWarehouseUserNoOpReconcileDoesNotUpdate(t *testing.T) {
	resource := newTestOperationsInsightsWarehouseUser()
	seedOperationsInsightsWarehouseUserStatusID(resource, testWarehouseUserID)
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	fake.getFn = func(_ context.Context, request opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
		requireStringPtr(t, "GetOperationsInsightsWarehouseUserRequest.OperationsInsightsWarehouseUserId", request.OperationsInsightsWarehouseUserId, testWarehouseUserID)
		return opsisdk.GetOperationsInsightsWarehouseUserResponse{
			OperationsInsightsWarehouseUser: makeSDKOperationsInsightsWarehouseUser(testWarehouseUserID, resource.Spec, opsisdk.OperationsInsightsWarehouseUserLifecycleStateActive),
		}, nil
	}
	fake.updateFn = func(context.Context, opsisdk.UpdateOperationsInsightsWarehouseUserRequest) (opsisdk.UpdateOperationsInsightsWarehouseUserResponse, error) {
		t.Fatal("UpdateOperationsInsightsWarehouseUser should not be called for matching readback")
		return opsisdk.UpdateOperationsInsightsWarehouseUserResponse{}, nil
	}

	response, err := newTestOperationsInsightsWarehouseUserClient(t, fake).CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseUserRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
}

func TestOperationsInsightsWarehouseUserMutableUpdateShapesRequestAndRefreshesStatus(t *testing.T) {
	resource := newTestOperationsInsightsWarehouseUserWithMutableDrift()
	seedOperationsInsightsWarehouseUserStatusID(resource, testWarehouseUserID)

	var updateRequest *opsisdk.UpdateOperationsInsightsWarehouseUserRequest
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	fake.getFn = sequentialOperationsInsightsWarehouseUserGet(t, []opsiv1beta1.OperationsInsightsWarehouseUserSpec{
		currentOperationsInsightsWarehouseUserMutableUpdateSpec(),
		resource.Spec,
	})
	fake.updateFn = captureOperationsInsightsWarehouseUserUpdate(t, &updateRequest)
	fake.getWorkRequestFn = succeededOperationsInsightsWarehouseUserUpdateWorkRequest(t)

	response, err := newTestOperationsInsightsWarehouseUserClient(t, fake).CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseUserRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if updateRequest == nil {
		t.Fatal("UpdateOperationsInsightsWarehouseUser was not called")
	}
	assertOperationsInsightsWarehouseUserMutableUpdateRequest(t, *updateRequest)
	if got := resource.Status.IsEmDataAccess; !got {
		t.Fatalf("status.isEmDataAccess = %v, want true", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != testWarehouseUserRequestID2 {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, testWarehouseUserRequestID2)
	}
}

func TestOperationsInsightsWarehouseUserRotatedPasswordUpdatesWhenReadbackOmitsOrRedactsPassword(t *testing.T) {
	for _, tc := range []struct {
		name             string
		readbackPassword *string
	}{
		{
			name:             "omitted",
			readbackPassword: nil,
		},
		{
			name:             "redacted",
			readbackPassword: common.String("********"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertOperationsInsightsWarehouseUserRotatedPasswordUpdate(t, tc.readbackPassword)
		})
	}
}

func TestOperationsInsightsWarehouseUserNoOpReconcileIgnoresRedactedPasswordReadback(t *testing.T) {
	resource := newTestOperationsInsightsWarehouseUser()
	seedOperationsInsightsWarehouseUserStatusID(resource, testWarehouseUserID)
	seedOperationsInsightsWarehouseUserAppliedPasswordFingerprint(t, resource)
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	fake.getFn = func(_ context.Context, request opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
		requireStringPtr(t, "GetOperationsInsightsWarehouseUserRequest.OperationsInsightsWarehouseUserId", request.OperationsInsightsWarehouseUserId, testWarehouseUserID)
		current := makeSDKOperationsInsightsWarehouseUser(testWarehouseUserID, resource.Spec, opsisdk.OperationsInsightsWarehouseUserLifecycleStateActive)
		current.ConnectionPassword = common.String("********")
		return opsisdk.GetOperationsInsightsWarehouseUserResponse{
			OperationsInsightsWarehouseUser: current,
		}, nil
	}
	fake.updateFn = func(context.Context, opsisdk.UpdateOperationsInsightsWarehouseUserRequest) (opsisdk.UpdateOperationsInsightsWarehouseUserResponse, error) {
		t.Fatal("UpdateOperationsInsightsWarehouseUser should not be called for redacted password readback")
		return opsisdk.UpdateOperationsInsightsWarehouseUserResponse{}, nil
	}

	response, err := newTestOperationsInsightsWarehouseUserClient(t, fake).CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseUserRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if got := resource.Status.ConnectionPassword; got != "" {
		t.Fatalf("status.connectionPassword = %q, want scrubbed", got)
	}
	if got := operationsInsightsWarehouseUserAppliedPasswordFingerprint(resource); got == "" {
		t.Fatal("applied password fingerprint = empty, want retained fingerprint")
	}
}

func TestOperationsInsightsWarehouseUserPasswordChangedWhileCreateWorkRequestPendingForcesUpdateAfterRedactedReadback(t *testing.T) {
	resource := newTestOperationsInsightsWarehouseUser()
	originalFingerprint := requireOperationsInsightsWarehouseUserDesiredPasswordFingerprint(t, resource, "original")

	workRequestResponses := []opsisdk.WorkRequest{
		makeWarehouseUserWorkRequest(testWarehouseUserWorkID, opsisdk.OperationStatusInProgress, opsisdk.OperationTypeCreateOpsiWarehouseUser, opsisdk.ActionTypeInProgress, testWarehouseUserID),
		makeWarehouseUserWorkRequest(testWarehouseUserWorkID, opsisdk.OperationStatusSucceeded, opsisdk.OperationTypeCreateOpsiWarehouseUser, opsisdk.ActionTypeCreated, testWarehouseUserID),
		makeWarehouseUserWorkRequest(testWarehouseUserWorkID, opsisdk.OperationStatusSucceeded, opsisdk.OperationTypeUpdateOpsiWarehouseUser, opsisdk.ActionTypeUpdated, testWarehouseUserID),
	}
	var updateRequest *opsisdk.UpdateOperationsInsightsWarehouseUserRequest
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	fake.listFn = func(context.Context, opsisdk.ListOperationsInsightsWarehouseUsersRequest) (opsisdk.ListOperationsInsightsWarehouseUsersResponse, error) {
		return opsisdk.ListOperationsInsightsWarehouseUsersResponse{}, nil
	}
	fake.createFn = func(context.Context, opsisdk.CreateOperationsInsightsWarehouseUserRequest) (opsisdk.CreateOperationsInsightsWarehouseUserResponse, error) {
		return opsisdk.CreateOperationsInsightsWarehouseUserResponse{
			OpcWorkRequestId: common.String(testWarehouseUserWorkID),
			OpcRequestId:     common.String(testWarehouseUserRequestID),
		}, nil
	}
	fake.getWorkRequestFn = nextOperationsInsightsWarehouseUserWorkRequest(t, workRequestResponses)
	fake.getFn = func(context.Context, opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
		current := makeSDKOperationsInsightsWarehouseUser(testWarehouseUserID, resource.Spec, opsisdk.OperationsInsightsWarehouseUserLifecycleStateActive)
		current.ConnectionPassword = nil
		return opsisdk.GetOperationsInsightsWarehouseUserResponse{OperationsInsightsWarehouseUser: current}, nil
	}
	fake.updateFn = captureOperationsInsightsWarehouseUserUpdate(t, &updateRequest)
	client := newTestOperationsInsightsWarehouseUserClient(t, fake)

	response := createOrUpdateOperationsInsightsWarehouseUser(t, client, resource, "initial")
	requireOperationsInsightsWarehouseUserRequeueResponse(t, response, "initial", "pending create")
	requireOperationsInsightsWarehouseUserPendingPasswordFingerprint(t, resource, shared.OSOKAsyncPhaseCreate)

	resource.Spec.ConnectionPassword = "rotated-during-create"
	rotatedFingerprint := requireOperationsInsightsWarehouseUserDesiredPasswordFingerprint(t, resource, "rotated")
	response = createOrUpdateOperationsInsightsWarehouseUser(t, client, resource, "resume")
	requireOperationsInsightsWarehouseUserSuccessfulResponse(t, response, "resume")
	requireOperationsInsightsWarehouseUserAppliedPasswordFingerprint(t, resource, originalFingerprint, "after create completion")
	rejectOperationsInsightsWarehouseUserAppliedPasswordFingerprint(t, resource, rotatedFingerprint)

	response = createOrUpdateOperationsInsightsWarehouseUser(t, client, resource, "follow-up")
	requireOperationsInsightsWarehouseUserSuccessfulResponse(t, response, "follow-up")
	requireOperationsInsightsWarehouseUserUpdateRequest(t, updateRequest, "password changed during create work request", "rotated-during-create")
	assertOperationsInsightsWarehouseUserPasswordScrubbed(t, resource)
}

func TestOperationsInsightsWarehouseUserPasswordChangedWhileUpdateWorkRequestPendingForcesUpdateAfterRedactedReadback(t *testing.T) {
	resource := newTestOperationsInsightsWarehouseUser()
	seedOperationsInsightsWarehouseUserStatusID(resource, testWarehouseUserID)
	seedOperationsInsightsWarehouseUserAppliedPasswordFingerprint(t, resource)
	resource.Spec.ConnectionPassword = "sent-before-pending-update"
	sentFingerprint := requireOperationsInsightsWarehouseUserDesiredPasswordFingerprint(t, resource, "sent")

	workRequestResponses := []opsisdk.WorkRequest{
		makeWarehouseUserWorkRequest(testWarehouseUserWorkID, opsisdk.OperationStatusInProgress, opsisdk.OperationTypeUpdateOpsiWarehouseUser, opsisdk.ActionTypeInProgress, testWarehouseUserID),
		makeWarehouseUserWorkRequest(testWarehouseUserWorkID, opsisdk.OperationStatusSucceeded, opsisdk.OperationTypeUpdateOpsiWarehouseUser, opsisdk.ActionTypeUpdated, testWarehouseUserID),
		makeWarehouseUserWorkRequest(testWarehouseUserWorkID, opsisdk.OperationStatusSucceeded, opsisdk.OperationTypeUpdateOpsiWarehouseUser, opsisdk.ActionTypeUpdated, testWarehouseUserID),
	}
	var updateRequests []opsisdk.UpdateOperationsInsightsWarehouseUserRequest
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	fake.getWorkRequestFn = nextOperationsInsightsWarehouseUserWorkRequest(t, workRequestResponses)
	fake.getFn = func(context.Context, opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
		current := makeSDKOperationsInsightsWarehouseUser(testWarehouseUserID, resource.Spec, opsisdk.OperationsInsightsWarehouseUserLifecycleStateActive)
		current.ConnectionPassword = common.String("********")
		return opsisdk.GetOperationsInsightsWarehouseUserResponse{OperationsInsightsWarehouseUser: current}, nil
	}
	fake.updateFn = func(_ context.Context, request opsisdk.UpdateOperationsInsightsWarehouseUserRequest) (opsisdk.UpdateOperationsInsightsWarehouseUserResponse, error) {
		copiedRequest := request
		updateRequests = append(updateRequests, copiedRequest)
		return opsisdk.UpdateOperationsInsightsWarehouseUserResponse{
			OpcWorkRequestId: common.String(testWarehouseUserWorkID),
			OpcRequestId:     common.String(testWarehouseUserRequestID2),
		}, nil
	}
	client := newTestOperationsInsightsWarehouseUserClient(t, fake)

	response := createOrUpdateOperationsInsightsWarehouseUser(t, client, resource, "initial")
	requireOperationsInsightsWarehouseUserRequeueResponse(t, response, "initial", "pending update")
	requireOperationsInsightsWarehouseUserUpdateRequestCount(t, updateRequests, 1, "initial update")
	requireStringPtr(t, "initial UpdateOperationsInsightsWarehouseUserRequest.ConnectionPassword", updateRequests[0].ConnectionPassword, "sent-before-pending-update")

	resource.Spec.ConnectionPassword = "rotated-during-update"
	rotatedFingerprint := requireOperationsInsightsWarehouseUserDesiredPasswordFingerprint(t, resource, "rotated")
	response = createOrUpdateOperationsInsightsWarehouseUser(t, client, resource, "resume")
	requireOperationsInsightsWarehouseUserSuccessfulResponse(t, response, "resume")
	requireOperationsInsightsWarehouseUserAppliedPasswordFingerprint(t, resource, sentFingerprint, "after update completion")
	rejectOperationsInsightsWarehouseUserAppliedPasswordFingerprint(t, resource, rotatedFingerprint)
	requireOperationsInsightsWarehouseUserUpdateRequestCount(t, updateRequests, 1, "after resume")

	response = createOrUpdateOperationsInsightsWarehouseUser(t, client, resource, "follow-up")
	requireOperationsInsightsWarehouseUserSuccessfulResponse(t, response, "follow-up")
	requireOperationsInsightsWarehouseUserUpdateRequestCount(t, updateRequests, 2, "rotated password update")
	requireStringPtr(t, "rotated UpdateOperationsInsightsWarehouseUserRequest.ConnectionPassword", updateRequests[1].ConnectionPassword, "rotated-during-update")
	assertOperationsInsightsWarehouseUserPasswordScrubbed(t, resource)
}

func TestOperationsInsightsWarehouseUserRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := newTestOperationsInsightsWarehouseUser()
	seedOperationsInsightsWarehouseUserStatusID(resource, testWarehouseUserID)
	resource.Spec.Name = "renamed-user"
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	fake.getFn = func(context.Context, opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
		currentSpec := newTestOperationsInsightsWarehouseUser().Spec
		return opsisdk.GetOperationsInsightsWarehouseUserResponse{
			OperationsInsightsWarehouseUser: makeSDKOperationsInsightsWarehouseUser(testWarehouseUserID, currentSpec, opsisdk.OperationsInsightsWarehouseUserLifecycleStateActive),
		}, nil
	}
	fake.updateFn = func(context.Context, opsisdk.UpdateOperationsInsightsWarehouseUserRequest) (opsisdk.UpdateOperationsInsightsWarehouseUserResponse, error) {
		t.Fatal("UpdateOperationsInsightsWarehouseUser should not be called for create-only drift")
		return opsisdk.UpdateOperationsInsightsWarehouseUserResponse{}, nil
	}

	response, err := newTestOperationsInsightsWarehouseUserClient(t, fake).CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseUserRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), "name changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want name drift", err)
	}
}

func TestOperationsInsightsWarehouseUserDeleteWaitsForPendingCreateWorkRequest(t *testing.T) {
	resource := newTestOperationsInsightsWarehouseUser()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   testWarehouseUserWorkID,
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &metav1.Time{},
	}
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	fake.getWorkRequestFn = func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testWarehouseUserWorkID)
		return opsisdk.GetWorkRequestResponse{
			WorkRequest: makeWarehouseUserWorkRequest(testWarehouseUserWorkID, opsisdk.OperationStatusInProgress, opsisdk.OperationTypeCreateOpsiWarehouseUser, opsisdk.ActionTypeInProgress, testWarehouseUserID),
		}, nil
	}
	fake.deleteFn = func(context.Context, opsisdk.DeleteOperationsInsightsWarehouseUserRequest) (opsisdk.DeleteOperationsInsightsWarehouseUserResponse, error) {
		t.Fatal("DeleteOperationsInsightsWarehouseUser should not be called while create work request is pending")
		return opsisdk.DeleteOperationsInsightsWarehouseUserResponse{}, nil
	}

	deleted, err := newTestOperationsInsightsWarehouseUserClient(t, fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while create is pending")
	}
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseCreate || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current = %#v, want pending create", current)
	}
}

func TestOperationsInsightsWarehouseUserDeleteAfterSucceededCreateWorkRequestIssuesDelete(t *testing.T) {
	resource := newTestOperationsInsightsWarehouseUser()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   testWarehouseUserWorkID,
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &metav1.Time{},
	}
	deleteCalled := false
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	fake.getWorkRequestFn = func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		return opsisdk.GetWorkRequestResponse{
			WorkRequest: makeWarehouseUserWorkRequest(testWarehouseUserWorkID, opsisdk.OperationStatusSucceeded, opsisdk.OperationTypeCreateOpsiWarehouseUser, opsisdk.ActionTypeCreated, testWarehouseUserID),
		}, nil
	}
	fake.getFn = func(context.Context, opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
		return opsisdk.GetOperationsInsightsWarehouseUserResponse{
			OperationsInsightsWarehouseUser: makeSDKOperationsInsightsWarehouseUser(testWarehouseUserID, resource.Spec, opsisdk.OperationsInsightsWarehouseUserLifecycleStateActive),
		}, nil
	}
	fake.deleteFn = func(_ context.Context, request opsisdk.DeleteOperationsInsightsWarehouseUserRequest) (opsisdk.DeleteOperationsInsightsWarehouseUserResponse, error) {
		deleteCalled = true
		requireStringPtr(t, "DeleteOperationsInsightsWarehouseUserRequest.OperationsInsightsWarehouseUserId", request.OperationsInsightsWarehouseUserId, testWarehouseUserID)
		return opsisdk.DeleteOperationsInsightsWarehouseUserResponse{
			OpcWorkRequestId: common.String(testWarehouseUserWorkID),
			OpcRequestId:     common.String(testWarehouseUserRequestID),
		}, nil
	}

	deleted, err := newTestOperationsInsightsWarehouseUserClient(t, fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false until delete work request confirms")
	}
	if !deleteCalled {
		t.Fatal("DeleteOperationsInsightsWarehouseUser was not called after create work request succeeded")
	}
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want pending delete", current)
	}
}

func TestOperationsInsightsWarehouseUserDeleteAfterSucceededWriteWorkRequestWaitsForReadback(t *testing.T) {
	for _, tc := range []struct {
		name          string
		phase         shared.OSOKAsyncPhase
		operationType opsisdk.OperationTypeEnum
		action        opsisdk.ActionTypeEnum
	}{
		{
			name:          "create",
			phase:         shared.OSOKAsyncPhaseCreate,
			operationType: opsisdk.OperationTypeCreateOpsiWarehouseUser,
			action:        opsisdk.ActionTypeCreated,
		},
		{
			name:          "update",
			phase:         shared.OSOKAsyncPhaseUpdate,
			operationType: opsisdk.OperationTypeUpdateOpsiWarehouseUser,
			action:        opsisdk.ActionTypeUpdated,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource := newTestOperationsInsightsWarehouseUser()
			resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
				Source:          shared.OSOKAsyncSourceWorkRequest,
				Phase:           tc.phase,
				WorkRequestID:   testWarehouseUserWorkID,
				NormalizedClass: shared.OSOKAsyncClassPending,
				UpdatedAt:       &metav1.Time{},
			}
			readbackErr := errortest.NewServiceError(404, errorutil.NotFound, "warehouse user not yet readable")
			readbackErr.OpcRequestID = "opc-readback-404"
			fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
			fake.getWorkRequestFn = func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
				return opsisdk.GetWorkRequestResponse{
					WorkRequest: makeWarehouseUserWorkRequest(testWarehouseUserWorkID, opsisdk.OperationStatusSucceeded, tc.operationType, tc.action, testWarehouseUserID),
				}, nil
			}
			fake.getFn = func(context.Context, opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
				return opsisdk.GetOperationsInsightsWarehouseUserResponse{}, readbackErr
			}
			fake.deleteFn = func(context.Context, opsisdk.DeleteOperationsInsightsWarehouseUserRequest) (opsisdk.DeleteOperationsInsightsWarehouseUserResponse, error) {
				t.Fatal("DeleteOperationsInsightsWarehouseUser should wait for readback before issuing delete")
				return opsisdk.DeleteOperationsInsightsWarehouseUserResponse{}, nil
			}

			deleted, err := newTestOperationsInsightsWarehouseUserClient(t, fake).Delete(context.Background(), resource)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if deleted {
				t.Fatal("Delete() deleted = true, want false while succeeded write readback is missing")
			}
			if resource.Status.OsokStatus.DeletedAt != nil {
				t.Fatalf("status.status.deletedAt = %#v, want nil", resource.Status.OsokStatus.DeletedAt)
			}
			assertOperationsInsightsWarehouseUserPendingReadback(t, resource, tc.phase)
		})
	}
}

func TestOperationsInsightsWarehouseUserDeleteWorkRequestAuthShapedConfirmReadRemainsFatal(t *testing.T) {
	resource := newTestOperationsInsightsWarehouseUser()
	seedOperationsInsightsWarehouseUserStatusID(resource, testWarehouseUserID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   testWarehouseUserWorkID,
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &metav1.Time{},
	}
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	authErr.OpcRequestID = "opc-auth-confirm"
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	fake.getWorkRequestFn = func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		return opsisdk.GetWorkRequestResponse{
			WorkRequest: makeWarehouseUserWorkRequest(testWarehouseUserWorkID, opsisdk.OperationStatusSucceeded, opsisdk.OperationTypeDeleteOpsiWarehouseUser, opsisdk.ActionTypeDeleted, testWarehouseUserID),
		}, nil
	}
	fake.getFn = func(context.Context, opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
		return opsisdk.GetOperationsInsightsWarehouseUserResponse{}, authErr
	}

	deleted, err := newTestOperationsInsightsWarehouseUserClient(t, fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped 404 to remain fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous 404")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %#v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-auth-confirm" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-auth-confirm", got)
	}
}

func TestOperationsInsightsWarehouseUserCreateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := newTestOperationsInsightsWarehouseUser()
	serviceErr := errortest.NewServiceError(500, "InternalError", "create failed")
	serviceErr.OpcRequestID = "opc-create-error"
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	fake.listFn = func(context.Context, opsisdk.ListOperationsInsightsWarehouseUsersRequest) (opsisdk.ListOperationsInsightsWarehouseUsersResponse, error) {
		return opsisdk.ListOperationsInsightsWarehouseUsersResponse{}, nil
	}
	fake.createFn = func(context.Context, opsisdk.CreateOperationsInsightsWarehouseUserRequest) (opsisdk.CreateOperationsInsightsWarehouseUserResponse, error) {
		return opsisdk.CreateOperationsInsightsWarehouseUserResponse{}, serviceErr
	}

	response, err := newTestOperationsInsightsWarehouseUserClient(t, fake).CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseUserRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-error" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-error", got)
	}
	if got := resource.Status.ConnectionPassword; got != "" {
		t.Fatalf("status.connectionPassword = %q, want scrubbed", got)
	}
}

func newTestOperationsInsightsWarehouseUserClient(t *testing.T, fake *fakeOperationsInsightsWarehouseUserOCI) OperationsInsightsWarehouseUserServiceClient {
	t.Helper()
	hooks := newOperationsInsightsWarehouseUserRuntimeHooksWithOCIClient(fake)
	applyOperationsInsightsWarehouseUserRuntimeHooks(&hooks, fake, nil, testOperationsInsightsWarehouseUserLogger())
	manager := &OperationsInsightsWarehouseUserServiceManager{Log: testOperationsInsightsWarehouseUserLogger()}
	config := buildOperationsInsightsWarehouseUserGeneratedRuntimeConfig(manager, hooks)
	delegate := defaultOperationsInsightsWarehouseUserServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.OperationsInsightsWarehouseUser](config),
	}
	return wrapOperationsInsightsWarehouseUserGeneratedClient(hooks, delegate)
}

func assertOperationsInsightsWarehouseUserRotatedPasswordUpdate(t *testing.T, readbackPassword *string) {
	t.Helper()
	resource := newTestOperationsInsightsWarehouseUser()
	seedOperationsInsightsWarehouseUserStatusID(resource, testWarehouseUserID)
	oldFingerprint := seedOperationsInsightsWarehouseUserAppliedPasswordFingerprint(t, resource)
	resource.Spec.ConnectionPassword = "rotated-secret-password"

	getCalls := 0
	var updateRequest *opsisdk.UpdateOperationsInsightsWarehouseUserRequest
	fake := &fakeOperationsInsightsWarehouseUserOCI{t: t}
	fake.getFn = func(_ context.Context, request opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
		getCalls++
		requireStringPtr(t, "GetOperationsInsightsWarehouseUserRequest.OperationsInsightsWarehouseUserId", request.OperationsInsightsWarehouseUserId, testWarehouseUserID)
		current := makeSDKOperationsInsightsWarehouseUser(testWarehouseUserID, resource.Spec, opsisdk.OperationsInsightsWarehouseUserLifecycleStateActive)
		current.ConnectionPassword = readbackPassword
		return opsisdk.GetOperationsInsightsWarehouseUserResponse{
			OperationsInsightsWarehouseUser: current,
		}, nil
	}
	fake.updateFn = captureOperationsInsightsWarehouseUserUpdate(t, &updateRequest)
	fake.getWorkRequestFn = succeededOperationsInsightsWarehouseUserUpdateWorkRequest(t)

	response, err := newTestOperationsInsightsWarehouseUserClient(t, fake).CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseUserRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if updateRequest == nil {
		t.Fatal("UpdateOperationsInsightsWarehouseUser was not called for rotated password")
	}
	requireStringPtr(t, "UpdateOperationsInsightsWarehouseUserRequest.ConnectionPassword", updateRequest.ConnectionPassword, "rotated-secret-password")
	if got := resource.Status.ConnectionPassword; got != "" {
		t.Fatalf("status.connectionPassword = %q, want scrubbed", got)
	}
	newFingerprint := operationsInsightsWarehouseUserAppliedPasswordFingerprint(resource)
	if newFingerprint == "" || newFingerprint == oldFingerprint {
		t.Fatalf("applied password fingerprint = %q, want new fingerprint different from %q", newFingerprint, oldFingerprint)
	}
	if strings.Contains(newFingerprint, "rotated-secret-password") {
		t.Fatal("applied password fingerprint contains the raw password")
	}
	if getCalls != 2 {
		t.Fatalf("GetOperationsInsightsWarehouseUser calls = %d, want initial read plus update follow-up", getCalls)
	}
}

func assertOperationsInsightsWarehouseUserPendingReadback(
	t *testing.T,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	wantPhase shared.OSOKAsyncPhase,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want pending readback")
	}
	if current.Phase != wantPhase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, wantPhase)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, shared.OSOKAsyncClassPending)
	}
	if current.WorkRequestID != testWarehouseUserWorkID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, testWarehouseUserWorkID)
	}
	if !strings.Contains(current.Message, "waiting for readback before delete") {
		t.Fatalf("status.async.current.message = %q, want readback wait", current.Message)
	}
	assertOperationsInsightsWarehouseUserStatusID(t, resource, testWarehouseUserID)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-readback-404" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-readback-404", got)
	}
}

func createOrUpdateOperationsInsightsWarehouseUser(
	t *testing.T,
	client OperationsInsightsWarehouseUserServiceClient,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	stage string,
) servicemanager.OSOKResponse {
	t.Helper()
	response, err := client.CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseUserRequest())
	if err != nil {
		t.Fatalf("%s CreateOrUpdate() error = %v", stage, err)
	}
	return response
}

func requireOperationsInsightsWarehouseUserSuccessfulResponse(t *testing.T, response servicemanager.OSOKResponse, stage string) {
	t.Helper()
	if !response.IsSuccessful {
		t.Fatalf("%s CreateOrUpdate().IsSuccessful = false, want true", stage)
	}
}

func requireOperationsInsightsWarehouseUserRequeueResponse(
	t *testing.T,
	response servicemanager.OSOKResponse,
	stage string,
	reason string,
) {
	t.Helper()
	if !response.ShouldRequeue {
		t.Fatalf("%s CreateOrUpdate().ShouldRequeue = false, want %s requeue", stage, reason)
	}
}

func requireOperationsInsightsWarehouseUserDesiredPasswordFingerprint(
	t *testing.T,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	label string,
) string {
	t.Helper()
	fingerprint, ok := operationsInsightsWarehouseUserDesiredPasswordFingerprint(resource)
	if !ok {
		t.Fatalf("%s operationsInsightsWarehouseUserDesiredPasswordFingerprint() ok = false, want true", label)
	}
	return fingerprint
}

func requireOperationsInsightsWarehouseUserPendingPasswordFingerprint(
	t *testing.T,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	phase shared.OSOKAsyncPhase,
) {
	t.Helper()
	if _, ok := operationsInsightsWarehouseUserPendingPasswordFingerprint(resource, testWarehouseUserWorkID, phase); !ok {
		t.Fatalf("pending %s password fingerprint was not recorded", phase)
	}
}

func requireOperationsInsightsWarehouseUserAppliedPasswordFingerprint(
	t *testing.T,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	want string,
	stage string,
) {
	t.Helper()
	if got := operationsInsightsWarehouseUserAppliedPasswordFingerprint(resource); got != want {
		t.Fatalf("applied password fingerprint %s = %q, want %q", stage, got, want)
	}
}

func rejectOperationsInsightsWarehouseUserAppliedPasswordFingerprint(
	t *testing.T,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	rejected string,
) {
	t.Helper()
	if got := operationsInsightsWarehouseUserAppliedPasswordFingerprint(resource); got == rejected {
		t.Fatalf("applied password fingerprint = rotated fingerprint %q before rotated password was sent", got)
	}
}

func requireOperationsInsightsWarehouseUserUpdateRequest(
	t *testing.T,
	request *opsisdk.UpdateOperationsInsightsWarehouseUserRequest,
	reason string,
	wantPassword string,
) {
	t.Helper()
	if request == nil {
		t.Fatalf("UpdateOperationsInsightsWarehouseUser was not called for %s", reason)
	}
	requireStringPtr(t, "UpdateOperationsInsightsWarehouseUserRequest.ConnectionPassword", request.ConnectionPassword, wantPassword)
}

func requireOperationsInsightsWarehouseUserUpdateRequestCount(
	t *testing.T,
	requests []opsisdk.UpdateOperationsInsightsWarehouseUserRequest,
	want int,
	stage string,
) {
	t.Helper()
	if got := len(requests); got != want {
		t.Fatalf("UpdateOperationsInsightsWarehouseUser calls %s = %d, want %d", stage, got, want)
	}
}

func assertOperationsInsightsWarehouseUserPasswordScrubbed(t *testing.T, resource *opsiv1beta1.OperationsInsightsWarehouseUser) {
	t.Helper()
	if got := resource.Status.ConnectionPassword; got != "" {
		t.Fatalf("status.connectionPassword = %q, want scrubbed", got)
	}
}

func newTestOperationsInsightsWarehouseUser() *opsiv1beta1.OperationsInsightsWarehouseUser {
	return &opsiv1beta1.OperationsInsightsWarehouseUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "warehouse-user",
			Namespace: "default",
			UID:       k8stypes.UID("warehouse-user-uid"),
		},
		Spec: opsiv1beta1.OperationsInsightsWarehouseUserSpec{
			OperationsInsightsWarehouseId: testWarehouseID,
			CompartmentId:                 testWarehouseCompartmentID,
			Name:                          testWarehouseUserName,
			ConnectionPassword:            testWarehouseUserPassword,
			IsAwrDataAccess:               true,
			IsEmDataAccess:                true,
			IsOpsiDataAccess:              true,
			FreeformTags:                  map[string]string{"env": "test"},
			DefinedTags:                   map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func newTestOperationsInsightsWarehouseUserWithMutableDrift() *opsiv1beta1.OperationsInsightsWarehouseUser {
	resource := newTestOperationsInsightsWarehouseUser()
	resource.Spec.IsAwrDataAccess = false
	resource.Spec.IsEmDataAccess = true
	resource.Spec.IsOpsiDataAccess = false
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}
	return resource
}

func currentOperationsInsightsWarehouseUserMutableUpdateSpec() opsiv1beta1.OperationsInsightsWarehouseUserSpec {
	currentSpec := newTestOperationsInsightsWarehouseUser().Spec
	currentSpec.IsAwrDataAccess = true
	currentSpec.IsEmDataAccess = false
	currentSpec.IsOpsiDataAccess = true
	currentSpec.FreeformTags = map[string]string{"env": "test"}
	return currentSpec
}

func sequentialOperationsInsightsWarehouseUserGet(
	t *testing.T,
	specs []opsiv1beta1.OperationsInsightsWarehouseUserSpec,
) func(context.Context, opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
	t.Helper()
	getCalls := 0
	return func(_ context.Context, request opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
		t.Helper()
		requireStringPtr(t, "GetOperationsInsightsWarehouseUserRequest.OperationsInsightsWarehouseUserId", request.OperationsInsightsWarehouseUserId, testWarehouseUserID)
		if len(specs) == 0 {
			t.Fatal("sequentialOperationsInsightsWarehouseUserGet requires at least one response spec")
		}
		index := getCalls
		getCalls++
		if index >= len(specs) {
			index = len(specs) - 1
		}
		return opsisdk.GetOperationsInsightsWarehouseUserResponse{
			OperationsInsightsWarehouseUser: makeSDKOperationsInsightsWarehouseUser(testWarehouseUserID, specs[index], opsisdk.OperationsInsightsWarehouseUserLifecycleStateActive),
		}, nil
	}
}

func captureOperationsInsightsWarehouseUserUpdate(
	t *testing.T,
	captured **opsisdk.UpdateOperationsInsightsWarehouseUserRequest,
) func(context.Context, opsisdk.UpdateOperationsInsightsWarehouseUserRequest) (opsisdk.UpdateOperationsInsightsWarehouseUserResponse, error) {
	t.Helper()
	return func(_ context.Context, request opsisdk.UpdateOperationsInsightsWarehouseUserRequest) (opsisdk.UpdateOperationsInsightsWarehouseUserResponse, error) {
		t.Helper()
		copiedRequest := request
		*captured = &copiedRequest
		requireStringPtr(t, "UpdateOperationsInsightsWarehouseUserRequest.OperationsInsightsWarehouseUserId", request.OperationsInsightsWarehouseUserId, testWarehouseUserID)
		return opsisdk.UpdateOperationsInsightsWarehouseUserResponse{
			OpcWorkRequestId: common.String(testWarehouseUserWorkID),
			OpcRequestId:     common.String(testWarehouseUserRequestID2),
		}, nil
	}
}

func succeededOperationsInsightsWarehouseUserUpdateWorkRequest(
	t *testing.T,
) func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
	t.Helper()
	return func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		t.Helper()
		requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testWarehouseUserWorkID)
		return opsisdk.GetWorkRequestResponse{
			WorkRequest: makeWarehouseUserWorkRequest(testWarehouseUserWorkID, opsisdk.OperationStatusSucceeded, opsisdk.OperationTypeUpdateOpsiWarehouseUser, opsisdk.ActionTypeUpdated, testWarehouseUserID),
		}, nil
	}
}

func nextOperationsInsightsWarehouseUserWorkRequest(
	t *testing.T,
	workRequests []opsisdk.WorkRequest,
) func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
	t.Helper()
	index := 0
	return func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
		t.Helper()
		requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testWarehouseUserWorkID)
		if index >= len(workRequests) {
			t.Fatalf("GetWorkRequest calls = %d, want at most %d", index+1, len(workRequests))
		}
		response := opsisdk.GetWorkRequestResponse{WorkRequest: workRequests[index]}
		index++
		return response, nil
	}
}

func requireOperationsInsightsWarehouseUserCreateDetails(t *testing.T, body any) opsisdk.CreateOperationsInsightsWarehouseUserDetails {
	t.Helper()
	details, ok := body.(opsisdk.CreateOperationsInsightsWarehouseUserDetails)
	if !ok {
		t.Fatalf("create body type = %T, want opsi.CreateOperationsInsightsWarehouseUserDetails", body)
	}
	return details
}

func assertOperationsInsightsWarehouseUserCreateAccessBody(t *testing.T, details opsisdk.CreateOperationsInsightsWarehouseUserDetails) {
	t.Helper()
	if details.IsAwrDataAccess == nil || *details.IsAwrDataAccess {
		t.Fatalf("create body IsAwrDataAccess = %#v, want explicit false", details.IsAwrDataAccess)
	}
	if details.IsEmDataAccess != nil {
		t.Fatalf("create body IsEmDataAccess = %#v, want nil when omitted/false", details.IsEmDataAccess)
	}
	if details.IsOpsiDataAccess == nil || !*details.IsOpsiDataAccess {
		t.Fatalf("create body IsOpsiDataAccess = %#v, want true", details.IsOpsiDataAccess)
	}
}

func assertOperationsInsightsWarehouseUserMutableUpdateRequest(t *testing.T, request opsisdk.UpdateOperationsInsightsWarehouseUserRequest) {
	t.Helper()
	if request.IsAwrDataAccess == nil || *request.IsAwrDataAccess {
		t.Fatalf("UpdateOperationsInsightsWarehouseUser IsAwrDataAccess = %#v, want explicit false", request.IsAwrDataAccess)
	}
	if request.IsEmDataAccess == nil || !*request.IsEmDataAccess {
		t.Fatalf("UpdateOperationsInsightsWarehouseUser IsEmDataAccess = %#v, want true", request.IsEmDataAccess)
	}
	if request.IsOpsiDataAccess == nil || *request.IsOpsiDataAccess {
		t.Fatalf("UpdateOperationsInsightsWarehouseUser IsOpsiDataAccess = %#v, want explicit false", request.IsOpsiDataAccess)
	}
	if got := request.FreeformTags["env"]; got != "prod" {
		t.Fatalf("UpdateOperationsInsightsWarehouseUser freeformTags[env] = %q, want prod", got)
	}
	requireOperationsInsightsWarehouseUserDefinedTag(t, request.DefinedTags, "Operations", "CostCenter", "84")
}

func requireOperationsInsightsWarehouseUserDefinedTag(t *testing.T, tags map[string]map[string]interface{}, namespace, key, want string) {
	t.Helper()
	if got := tags[namespace][key]; got != want {
		t.Fatalf("defined tag %s.%s = %#v, want %s", namespace, key, got, want)
	}
}

func seedOperationsInsightsWarehouseUserStatusID(resource *opsiv1beta1.OperationsInsightsWarehouseUser, id string) {
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
}

func seedOperationsInsightsWarehouseUserAppliedPasswordFingerprint(t *testing.T, resource *opsiv1beta1.OperationsInsightsWarehouseUser) string {
	t.Helper()
	fingerprint, ok := operationsInsightsWarehouseUserDesiredPasswordFingerprint(resource)
	if !ok {
		t.Fatal("operationsInsightsWarehouseUserDesiredPasswordFingerprint() ok = false, want true")
	}
	resource.Status.OsokStatus.Conditions = append(resource.Status.OsokStatus.Conditions, shared.OSOKCondition{
		Type:   shared.Active,
		Reason: operationsInsightsWarehouseUserPasswordReasonPrefix + fingerprint,
	})
	return fingerprint
}

func testOperationsInsightsWarehouseUserRequest() ctrl.Request {
	return ctrl.Request{NamespacedName: k8stypes.NamespacedName{Namespace: "default", Name: "warehouse-user"}}
}

func testOperationsInsightsWarehouseUserLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: logr.Discard()}
}

func makeSDKOperationsInsightsWarehouseUser(
	id string,
	spec opsiv1beta1.OperationsInsightsWarehouseUserSpec,
	state opsisdk.OperationsInsightsWarehouseUserLifecycleStateEnum,
) opsisdk.OperationsInsightsWarehouseUser {
	return opsisdk.OperationsInsightsWarehouseUser{
		OperationsInsightsWarehouseId: common.String(spec.OperationsInsightsWarehouseId),
		Id:                            common.String(id),
		CompartmentId:                 common.String(spec.CompartmentId),
		Name:                          common.String(spec.Name),
		ConnectionPassword:            common.String(spec.ConnectionPassword),
		IsAwrDataAccess:               common.Bool(spec.IsAwrDataAccess),
		IsEmDataAccess:                optionalWarehouseUserBool(spec.IsEmDataAccess),
		IsOpsiDataAccess:              optionalWarehouseUserBool(spec.IsOpsiDataAccess),
		LifecycleState:                state,
		FreeformTags:                  cloneOperationsInsightsWarehouseUserStringMap(spec.FreeformTags),
		DefinedTags:                   operationsInsightsWarehouseUserDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func makeSDKOperationsInsightsWarehouseUserSummary(
	id string,
	name string,
	spec opsiv1beta1.OperationsInsightsWarehouseUserSpec,
	state opsisdk.OperationsInsightsWarehouseUserLifecycleStateEnum,
) opsisdk.OperationsInsightsWarehouseUserSummary {
	return opsisdk.OperationsInsightsWarehouseUserSummary{
		OperationsInsightsWarehouseId: common.String(spec.OperationsInsightsWarehouseId),
		Id:                            common.String(id),
		CompartmentId:                 common.String(spec.CompartmentId),
		Name:                          common.String(name),
		IsAwrDataAccess:               common.Bool(spec.IsAwrDataAccess),
		IsEmDataAccess:                optionalWarehouseUserBool(spec.IsEmDataAccess),
		IsOpsiDataAccess:              optionalWarehouseUserBool(spec.IsOpsiDataAccess),
		LifecycleState:                state,
		FreeformTags:                  cloneOperationsInsightsWarehouseUserStringMap(spec.FreeformTags),
		DefinedTags:                   operationsInsightsWarehouseUserDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func makeWarehouseUserWorkRequest(
	id string,
	status opsisdk.OperationStatusEnum,
	operationType opsisdk.OperationTypeEnum,
	action opsisdk.ActionTypeEnum,
	resourceID string,
) opsisdk.WorkRequest {
	return opsisdk.WorkRequest{
		Id:            common.String(id),
		Status:        status,
		OperationType: operationType,
		CompartmentId: common.String(testWarehouseCompartmentID),
		Resources: []opsisdk.WorkRequestResource{{
			EntityType: common.String("operationsInsightsWarehouseUser"),
			ActionType: action,
			Identifier: common.String(resourceID),
		}},
		PercentComplete: common.Float32(100),
		TimeAccepted:    &common.SDKTime{Time: metav1.Now().Time},
	}
}

func optionalWarehouseUserBool(value bool) *bool {
	if !value {
		return nil
	}
	return common.Bool(value)
}

func requireStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func assertOperationsInsightsWarehouseUserStatusID(t *testing.T, resource *opsiv1beta1.OperationsInsightsWarehouseUser, want string) {
	t.Helper()
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got := resource.Status.OsokStatus.Ocid; got != shared.OCID(want) {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

var _ operationsInsightsWarehouseUserOCIClient = (*fakeOperationsInsightsWarehouseUserOCI)(nil)
