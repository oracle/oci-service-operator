/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package privilegedapicontrol

import (
	"context"
	"maps"
	"reflect"
	"strings"
	"testing"
	"time"

	apiaccesscontrolsdk "github.com/oracle/oci-go-sdk/v65/apiaccesscontrol"
	"github.com/oracle/oci-go-sdk/v65/common"
	apiaccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/apiaccesscontrol/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakePrivilegedApiControlOCIClient struct {
	createFn      func(context.Context, apiaccesscontrolsdk.CreatePrivilegedApiControlRequest) (apiaccesscontrolsdk.CreatePrivilegedApiControlResponse, error)
	getFn         func(context.Context, apiaccesscontrolsdk.GetPrivilegedApiControlRequest) (apiaccesscontrolsdk.GetPrivilegedApiControlResponse, error)
	listFn        func(context.Context, apiaccesscontrolsdk.ListPrivilegedApiControlsRequest) (apiaccesscontrolsdk.ListPrivilegedApiControlsResponse, error)
	updateFn      func(context.Context, apiaccesscontrolsdk.UpdatePrivilegedApiControlRequest) (apiaccesscontrolsdk.UpdatePrivilegedApiControlResponse, error)
	deleteFn      func(context.Context, apiaccesscontrolsdk.DeletePrivilegedApiControlRequest) (apiaccesscontrolsdk.DeletePrivilegedApiControlResponse, error)
	workRequestFn func(context.Context, apiaccesscontrolsdk.GetWorkRequestRequest) (apiaccesscontrolsdk.GetWorkRequestResponse, error)
}

func (f *fakePrivilegedApiControlOCIClient) CreatePrivilegedApiControl(
	ctx context.Context,
	req apiaccesscontrolsdk.CreatePrivilegedApiControlRequest,
) (apiaccesscontrolsdk.CreatePrivilegedApiControlResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return apiaccesscontrolsdk.CreatePrivilegedApiControlResponse{}, nil
}

func (f *fakePrivilegedApiControlOCIClient) GetPrivilegedApiControl(
	ctx context.Context,
	req apiaccesscontrolsdk.GetPrivilegedApiControlRequest,
) (apiaccesscontrolsdk.GetPrivilegedApiControlResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return apiaccesscontrolsdk.GetPrivilegedApiControlResponse{}, errortest.NewServiceError(404, "NotFound", "missing")
}

func (f *fakePrivilegedApiControlOCIClient) ListPrivilegedApiControls(
	ctx context.Context,
	req apiaccesscontrolsdk.ListPrivilegedApiControlsRequest,
) (apiaccesscontrolsdk.ListPrivilegedApiControlsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return apiaccesscontrolsdk.ListPrivilegedApiControlsResponse{}, nil
}

func (f *fakePrivilegedApiControlOCIClient) UpdatePrivilegedApiControl(
	ctx context.Context,
	req apiaccesscontrolsdk.UpdatePrivilegedApiControlRequest,
) (apiaccesscontrolsdk.UpdatePrivilegedApiControlResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return apiaccesscontrolsdk.UpdatePrivilegedApiControlResponse{}, nil
}

func (f *fakePrivilegedApiControlOCIClient) DeletePrivilegedApiControl(
	ctx context.Context,
	req apiaccesscontrolsdk.DeletePrivilegedApiControlRequest,
) (apiaccesscontrolsdk.DeletePrivilegedApiControlResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return apiaccesscontrolsdk.DeletePrivilegedApiControlResponse{}, nil
}

func (f *fakePrivilegedApiControlOCIClient) GetWorkRequest(
	ctx context.Context,
	req apiaccesscontrolsdk.GetWorkRequestRequest,
) (apiaccesscontrolsdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return apiaccesscontrolsdk.GetWorkRequestResponse{}, nil
}

func TestReviewedPrivilegedApiControlRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedPrivilegedApiControlRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedPrivilegedApiControlRuntimeSemantics() = nil")
	}

	if got.FormalService != "apiaccesscontrol" {
		t.Fatalf("FormalService = %q, want apiaccesscontrol", got.FormalService)
	}
	if got.FormalSlug != "privilegedapicontrol" {
		t.Fatalf("FormalSlug = %q, want privilegedapicontrol", got.FormalSlug)
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
	assertPrivilegedApiControlStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertPrivilegedApiControlStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertPrivilegedApiControlStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertPrivilegedApiControlStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertPrivilegedApiControlStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertPrivilegedApiControlStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertPrivilegedApiControlStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "resourceType", "id"})
	assertPrivilegedApiControlStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{
		"approverGroupIdList",
		"definedTags",
		"description",
		"displayName",
		"freeformTags",
		"notificationTopicId",
		"numberOfApprovers",
		"privilegedOperationList",
		"resourceType",
		"resources",
	})
	assertPrivilegedApiControlStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetPrivilegedApiControl" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want GetWorkRequest -> GetPrivilegedApiControl", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetPrivilegedApiControl" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want GetWorkRequest -> GetPrivilegedApiControl", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetWorkRequest -> GetPrivilegedApiControl/ListPrivilegedApiControls confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want workrequest-backed confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want none for published runtime", got.AuxiliaryOperations)
	}
}

func TestGuardPrivilegedApiControlExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makePrivilegedApiControlResource()
	resource.Spec.DisplayName = ""

	decision, err := guardPrivilegedApiControlExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardPrivilegedApiControlExistingBeforeCreate(empty displayName) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardPrivilegedApiControlExistingBeforeCreate(empty displayName) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.DisplayName = "privileged-api-control"
	resource.Spec.ResourceType = ""
	decision, err = guardPrivilegedApiControlExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardPrivilegedApiControlExistingBeforeCreate(empty resourceType) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardPrivilegedApiControlExistingBeforeCreate(empty resourceType) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.ResourceType = "core-instance"
	decision, err = guardPrivilegedApiControlExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardPrivilegedApiControlExistingBeforeCreate(complete identity) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardPrivilegedApiControlExistingBeforeCreate(complete identity) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestBuildPrivilegedApiControlUpdateBodyPreservesClearSemantics(t *testing.T) {
	t.Parallel()

	currentResource := makePrivilegedApiControlResource()
	desired := makePrivilegedApiControlResource()
	desired.Spec.Description = ""
	desired.Spec.ApproverGroupIdList = []string{}
	desired.Spec.Resources = []string{}
	desired.Spec.PrivilegedOperationList = []apiaccesscontrolv1beta1.PrivilegedApiControlPrivilegedOperationList{}
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}
	desired.Spec.NotificationTopicId = "ocid1.onstopic.oc1..updated"
	desired.Spec.NumberOfApprovers = 3

	body, updateNeeded, err := buildPrivilegedApiControlUpdateBody(
		desired,
		apiaccesscontrolsdk.GetPrivilegedApiControlResponse{
			PrivilegedApiControl: makeSDKPrivilegedApiControl(
				"ocid1.privilegedapicontrol.oc1..existing",
				currentResource,
				apiaccesscontrolsdk.PrivilegedApiControlLifecycleStateActive,
			),
		},
	)
	if err != nil {
		t.Fatalf("buildPrivilegedApiControlUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildPrivilegedApiControlUpdateBody() updateNeeded = false, want true")
	}

	requirePrivilegedApiControlStringPtr(t, "details.description", body.Description, "")
	requirePrivilegedApiControlStringPtr(t, "details.notificationTopicId", body.NotificationTopicId, desired.Spec.NotificationTopicId)
	requirePrivilegedApiControlIntPtr(t, "details.numberOfApprovers", body.NumberOfApprovers, desired.Spec.NumberOfApprovers)
	if len(body.ApproverGroupIdList) != 0 {
		t.Fatalf("details.ApproverGroupIdList = %#v, want empty slice for clear", body.ApproverGroupIdList)
	}
	if len(body.Resources) != 0 {
		t.Fatalf("details.Resources = %#v, want empty slice for clear", body.Resources)
	}
	if len(body.PrivilegedOperationList) != 0 {
		t.Fatalf("details.PrivilegedOperationList = %#v, want empty slice for clear", body.PrivilegedOperationList)
	}
	if len(body.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map for clear", body.FreeformTags)
	}
	if len(body.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map for clear", body.DefinedTags)
	}
}

func TestPrivilegedApiControlCreateOrUpdateSkipsReuseWhenDisplayNameMissing(t *testing.T) {
	t.Parallel()

	resource := makePrivilegedApiControlResource()
	resource.Spec.DisplayName = ""

	listCalls := 0
	createCalls := 0

	client := newTestPrivilegedApiControlClient(&fakePrivilegedApiControlOCIClient{
		listFn: func(_ context.Context, _ apiaccesscontrolsdk.ListPrivilegedApiControlsRequest) (apiaccesscontrolsdk.ListPrivilegedApiControlsResponse, error) {
			listCalls++
			return apiaccesscontrolsdk.ListPrivilegedApiControlsResponse{}, nil
		},
		createFn: func(_ context.Context, req apiaccesscontrolsdk.CreatePrivilegedApiControlRequest) (apiaccesscontrolsdk.CreatePrivilegedApiControlResponse, error) {
			createCalls++
			requirePrivilegedApiControlStringPtr(t, "create compartmentId", req.CreatePrivilegedApiControlDetails.CompartmentId, resource.Spec.CompartmentId)
			if req.CreatePrivilegedApiControlDetails.DisplayName != nil {
				t.Fatalf("create displayName = %v, want nil when spec.displayName is empty", req.CreatePrivilegedApiControlDetails.DisplayName)
			}
			return apiaccesscontrolsdk.CreatePrivilegedApiControlResponse{
				OpcWorkRequestId: common.String("wr-create-empty-name"),
				OpcRequestId:     common.String("opc-create-empty-name"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req apiaccesscontrolsdk.GetWorkRequestRequest) (apiaccesscontrolsdk.GetWorkRequestResponse, error) {
			requirePrivilegedApiControlStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-empty-name")
			return apiaccesscontrolsdk.GetWorkRequestResponse{
				WorkRequest: makePrivilegedApiControlWorkRequest(
					"wr-create-empty-name",
					apiaccesscontrolsdk.OperationTypeCreatePrivilegedApiControl,
					apiaccesscontrolsdk.OperationStatusInProgress,
					apiaccesscontrolsdk.ActionTypeInProgress,
					"",
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
		t.Fatalf("ListPrivilegedApiControls() calls = %d, want 0 when displayName is empty", listCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreatePrivilegedApiControl() calls = %d, want 1", createCalls)
	}
	requirePrivilegedApiControlAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-empty-name", shared.OSOKAsyncClassPending)
}

func TestPrivilegedApiControlCreateOrUpdateRejectsAmbiguousReuse(t *testing.T) {
	t.Parallel()

	resource := makePrivilegedApiControlResource()
	createCalls := 0

	client := newTestPrivilegedApiControlClient(&fakePrivilegedApiControlOCIClient{
		listFn: func(_ context.Context, req apiaccesscontrolsdk.ListPrivilegedApiControlsRequest) (apiaccesscontrolsdk.ListPrivilegedApiControlsResponse, error) {
			requirePrivilegedApiControlStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requirePrivilegedApiControlStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			requirePrivilegedApiControlStringPtr(t, "list resourceType", req.ResourceType, resource.Spec.ResourceType)
			if req.LifecycleState != "" {
				t.Fatalf("list lifecycleState = %q, want empty", req.LifecycleState)
			}
			return apiaccesscontrolsdk.ListPrivilegedApiControlsResponse{
				PrivilegedApiControlCollection: apiaccesscontrolsdk.PrivilegedApiControlCollection{
					Items: []apiaccesscontrolsdk.PrivilegedApiControlSummary{
						makeSDKPrivilegedApiControlSummary("ocid1.privilegedapicontrol.oc1..first", resource, apiaccesscontrolsdk.PrivilegedApiControlLifecycleStateActive),
						makeSDKPrivilegedApiControlSummary("ocid1.privilegedapicontrol.oc1..second", resource, apiaccesscontrolsdk.PrivilegedApiControlLifecycleStateUpdating),
					},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ apiaccesscontrolsdk.CreatePrivilegedApiControlRequest) (apiaccesscontrolsdk.CreatePrivilegedApiControlResponse, error) {
			createCalls++
			return apiaccesscontrolsdk.CreatePrivilegedApiControlResponse{}, nil
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
		t.Fatalf("CreatePrivilegedApiControl() calls = %d, want 0 on ambiguous reuse", createCalls)
	}
}

func TestPrivilegedApiControlServiceClientCreatesAndResumesWorkRequest(t *testing.T) {
	t.Parallel()

	const (
		createdID     = "ocid1.privilegedapicontrol.oc1..created"
		workRequestID = "wr-privilegedapicontrol-create"
	)

	resource := makePrivilegedApiControlResource()
	workRequests := map[string]apiaccesscontrolsdk.WorkRequest{
		workRequestID: makePrivilegedApiControlWorkRequest(
			workRequestID,
			apiaccesscontrolsdk.OperationTypeCreatePrivilegedApiControl,
			apiaccesscontrolsdk.OperationStatusInProgress,
			apiaccesscontrolsdk.ActionTypeInProgress,
			"",
		),
	}

	var createRequest apiaccesscontrolsdk.CreatePrivilegedApiControlRequest
	getCalls := 0

	client := newTestPrivilegedApiControlClient(&fakePrivilegedApiControlOCIClient{
		listFn: func(_ context.Context, req apiaccesscontrolsdk.ListPrivilegedApiControlsRequest) (apiaccesscontrolsdk.ListPrivilegedApiControlsResponse, error) {
			requirePrivilegedApiControlStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requirePrivilegedApiControlStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
			requirePrivilegedApiControlStringPtr(t, "list resourceType", req.ResourceType, resource.Spec.ResourceType)
			return apiaccesscontrolsdk.ListPrivilegedApiControlsResponse{}, nil
		},
		createFn: func(_ context.Context, req apiaccesscontrolsdk.CreatePrivilegedApiControlRequest) (apiaccesscontrolsdk.CreatePrivilegedApiControlResponse, error) {
			createRequest = req
			return apiaccesscontrolsdk.CreatePrivilegedApiControlResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-privilegedapicontrol"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req apiaccesscontrolsdk.GetWorkRequestRequest) (apiaccesscontrolsdk.GetWorkRequestResponse, error) {
			requirePrivilegedApiControlStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return apiaccesscontrolsdk.GetWorkRequestResponse{
				WorkRequest: workRequests[workRequestID],
			}, nil
		},
		getFn: func(_ context.Context, req apiaccesscontrolsdk.GetPrivilegedApiControlRequest) (apiaccesscontrolsdk.GetPrivilegedApiControlResponse, error) {
			getCalls++
			requirePrivilegedApiControlStringPtr(t, "get privilegedApiControlId", req.PrivilegedApiControlId, createdID)
			return apiaccesscontrolsdk.GetPrivilegedApiControlResponse{
				PrivilegedApiControl: makeSDKPrivilegedApiControl(
					createdID,
					resource,
					apiaccesscontrolsdk.PrivilegedApiControlLifecycleStateActive,
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
	requirePrivilegedApiControlStringPtr(t, "create compartmentId", createRequest.CreatePrivilegedApiControlDetails.CompartmentId, resource.Spec.CompartmentId)
	requirePrivilegedApiControlStringPtr(t, "create notificationTopicId", createRequest.CreatePrivilegedApiControlDetails.NotificationTopicId, resource.Spec.NotificationTopicId)
	if getCalls != 0 {
		t.Fatalf("GetPrivilegedApiControl() calls = %d, want 0 while work request is pending", getCalls)
	}
	requirePrivilegedApiControlAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-privilegedapicontrol" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-privilegedapicontrol", got)
	}

	workRequests[workRequestID] = makePrivilegedApiControlWorkRequest(
		workRequestID,
		apiaccesscontrolsdk.OperationTypeCreatePrivilegedApiControl,
		apiaccesscontrolsdk.OperationStatusSucceeded,
		apiaccesscontrolsdk.ActionTypeCreated,
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
		t.Fatalf("GetPrivilegedApiControl() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(apiaccesscontrolsdk.PrivilegedApiControlLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared", resource.Status.OsokStatus.Async.Current)
	}
}

func TestPrivilegedApiControlDeleteIgnoresSpecDescriptionField(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.privilegedapicontrol.oc1..existing"

	resource := newExistingPrivilegedApiControlResource(existingID)
	resource.Spec.Description = "resource description must not become delete reason"

	client := newTestPrivilegedApiControlClient(&fakePrivilegedApiControlOCIClient{
		deleteFn: func(_ context.Context, req apiaccesscontrolsdk.DeletePrivilegedApiControlRequest) (apiaccesscontrolsdk.DeletePrivilegedApiControlResponse, error) {
			requirePrivilegedApiControlStringPtr(t, "delete privilegedApiControlId", req.PrivilegedApiControlId, existingID)
			if req.Description != nil {
				t.Fatalf("delete description = %v, want nil", req.Description)
			}
			return apiaccesscontrolsdk.DeletePrivilegedApiControlResponse{}, errortest.NewServiceError(404, "NotFound", "missing")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true after OCI reports NotFound")
	}
}

func TestPrivilegedApiControlCreateOrUpdateClassifiesNeedsAttentionAsFailed(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.privilegedapicontrol.oc1..existing"

	resource := newExistingPrivilegedApiControlResource(existingID)

	client := newTestPrivilegedApiControlClient(&fakePrivilegedApiControlOCIClient{
		getFn: func(_ context.Context, req apiaccesscontrolsdk.GetPrivilegedApiControlRequest) (apiaccesscontrolsdk.GetPrivilegedApiControlResponse, error) {
			requirePrivilegedApiControlStringPtr(t, "get privilegedApiControlId", req.PrivilegedApiControlId, existingID)
			return apiaccesscontrolsdk.GetPrivilegedApiControlResponse{
				PrivilegedApiControl: makeSDKPrivilegedApiControl(
					existingID,
					resource,
					apiaccesscontrolsdk.PrivilegedApiControlLifecycleStateNeedsAttention,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v, want nil lifecycle classification result", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful result", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want terminal failure without requeue", response)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Failed)
	}
	if resource.Status.LifecycleState != string(apiaccesscontrolsdk.PrivilegedApiControlLifecycleStateNeedsAttention) {
		t.Fatalf("status.lifecycleState = %q, want NEEDS_ATTENTION", resource.Status.LifecycleState)
	}
}

func newTestPrivilegedApiControlClient(client *fakePrivilegedApiControlOCIClient) PrivilegedApiControlServiceClient {
	if client == nil {
		client = &fakePrivilegedApiControlOCIClient{}
	}
	return newPrivilegedApiControlServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makePrivilegedApiControlResource() *apiaccesscontrolv1beta1.PrivilegedApiControl {
	return &apiaccesscontrolv1beta1.PrivilegedApiControl{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "privileged-api-control-sample",
			Namespace: "default",
		},
		Spec: apiaccesscontrolv1beta1.PrivilegedApiControlSpec{
			CompartmentId:       "ocid1.compartment.oc1..apiaccesscontrolexample",
			NotificationTopicId: "ocid1.onstopic.oc1..apiaccesscontrol",
			ApproverGroupIdList: []string{
				"ocid1.group.oc1..approverone",
				"ocid1.group.oc1..approvertwo",
			},
			PrivilegedOperationList: []apiaccesscontrolv1beta1.PrivilegedApiControlPrivilegedOperationList{
				{
					ApiName:        "LaunchInstance",
					EntityType:     "instance",
					AttributeNames: []string{"shape"},
				},
			},
			ResourceType:      "core-instance",
			Resources:         []string{"ocid1.instance.oc1..governed"},
			DisplayName:       "privileged-api-control-sample",
			Description:       "protect launch instance",
			NumberOfApprovers: 2,
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

func newExistingPrivilegedApiControlResource(id string) *apiaccesscontrolv1beta1.PrivilegedApiControl {
	resource := makePrivilegedApiControlResource()
	resource.Status = apiaccesscontrolv1beta1.PrivilegedApiControlStatus{
		OsokStatus:    resource.Status.OsokStatus,
		Id:            id,
		DisplayName:   resource.Spec.DisplayName,
		CompartmentId: resource.Spec.CompartmentId,
		ResourceType:  resource.Spec.ResourceType,
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	return resource
}

func makeSDKPrivilegedApiControl(
	id string,
	resource *apiaccesscontrolv1beta1.PrivilegedApiControl,
	state apiaccesscontrolsdk.PrivilegedApiControlLifecycleStateEnum,
) apiaccesscontrolsdk.PrivilegedApiControl {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return apiaccesscontrolsdk.PrivilegedApiControl{
		Id:                      common.String(id),
		DisplayName:             common.String(resource.Spec.DisplayName),
		CompartmentId:           common.String(resource.Spec.CompartmentId),
		TimeCreated:             &now,
		State:                   common.String(string(state)),
		LifecycleState:          state,
		FreeformTags:            maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:             privilegedApiControlDefinedTagsFromSpec(resource.Spec.DefinedTags),
		Description:             common.String(resource.Spec.Description),
		NotificationTopicId:     common.String(resource.Spec.NotificationTopicId),
		ApproverGroupIdList:     append([]string(nil), resource.Spec.ApproverGroupIdList...),
		ResourceType:            common.String(resource.Spec.ResourceType),
		Resources:               append([]string(nil), resource.Spec.Resources...),
		PrivilegedOperationList: privilegedApiControlPrivilegedOperationListFromSpec(resource.Spec.PrivilegedOperationList),
		NumberOfApprovers:       common.Int(resource.Spec.NumberOfApprovers),
		TimeUpdated:             &now,
		LifecycleDetails:        common.String("reviewed runtime"),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeSDKPrivilegedApiControlSummary(
	id string,
	resource *apiaccesscontrolv1beta1.PrivilegedApiControl,
	state apiaccesscontrolsdk.PrivilegedApiControlLifecycleStateEnum,
) apiaccesscontrolsdk.PrivilegedApiControlSummary {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return apiaccesscontrolsdk.PrivilegedApiControlSummary{
		Id:                common.String(id),
		DisplayName:       common.String(resource.Spec.DisplayName),
		CompartmentId:     common.String(resource.Spec.CompartmentId),
		TimeCreated:       &now,
		LifecycleState:    state,
		FreeformTags:      maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:       privilegedApiControlDefinedTagsFromSpec(resource.Spec.DefinedTags),
		ResourceType:      common.String(resource.Spec.ResourceType),
		NumberOfApprovers: common.Int(resource.Spec.NumberOfApprovers),
		TimeUpdated:       &now,
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makePrivilegedApiControlWorkRequest(
	id string,
	operation apiaccesscontrolsdk.OperationTypeEnum,
	status apiaccesscontrolsdk.OperationStatusEnum,
	action apiaccesscontrolsdk.ActionTypeEnum,
	resourceID string,
) apiaccesscontrolsdk.WorkRequest {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	percentComplete := float32(50)
	return apiaccesscontrolsdk.WorkRequest{
		OperationType:   operation,
		Status:          status,
		Id:              common.String(id),
		CompartmentId:   common.String("ocid1.compartment.oc1..apiaccesscontrolexample"),
		Resources:       []apiaccesscontrolsdk.WorkRequestResource{{EntityType: common.String("PrivilegedApiControl"), ActionType: action, Identifier: common.String(resourceID)}},
		PercentComplete: &percentComplete,
		TimeAccepted:    &now,
	}
}

func assertPrivilegedApiControlStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func requirePrivilegedApiControlStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", name, got, want)
	}
}

func requirePrivilegedApiControlIntPtr(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %d", name, got, want)
	}
}

func requirePrivilegedApiControlAsyncCurrent(
	t *testing.T,
	resource *apiaccesscontrolv1beta1.PrivilegedApiControl,
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
