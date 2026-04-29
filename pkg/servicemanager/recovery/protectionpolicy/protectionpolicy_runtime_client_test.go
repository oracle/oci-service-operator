/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package protectionpolicy

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	recoverysdk "github.com/oracle/oci-go-sdk/v65/recovery"
	recoveryv1beta1 "github.com/oracle/oci-service-operator/api/recovery/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testProtectionPolicyID            = "ocid1.recoveryprotectionpolicy.oc1..example"
	testProtectionPolicyCompartmentID = "ocid1.compartment.oc1..recovery"
	testProtectionPolicyDisplayName   = "policy-alpha"
)

type fakeProtectionPolicyOCIClient struct {
	createFn         func(context.Context, recoverysdk.CreateProtectionPolicyRequest) (recoverysdk.CreateProtectionPolicyResponse, error)
	getFn            func(context.Context, recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error)
	listFn           func(context.Context, recoverysdk.ListProtectionPoliciesRequest) (recoverysdk.ListProtectionPoliciesResponse, error)
	updateFn         func(context.Context, recoverysdk.UpdateProtectionPolicyRequest) (recoverysdk.UpdateProtectionPolicyResponse, error)
	deleteFn         func(context.Context, recoverysdk.DeleteProtectionPolicyRequest) (recoverysdk.DeleteProtectionPolicyResponse, error)
	getWorkRequestFn func(context.Context, recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error)
}

func (f *fakeProtectionPolicyOCIClient) CreateProtectionPolicy(ctx context.Context, req recoverysdk.CreateProtectionPolicyRequest) (recoverysdk.CreateProtectionPolicyResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return recoverysdk.CreateProtectionPolicyResponse{}, nil
}

func (f *fakeProtectionPolicyOCIClient) GetProtectionPolicy(ctx context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return recoverysdk.GetProtectionPolicyResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
}

func (f *fakeProtectionPolicyOCIClient) ListProtectionPolicies(ctx context.Context, req recoverysdk.ListProtectionPoliciesRequest) (recoverysdk.ListProtectionPoliciesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return recoverysdk.ListProtectionPoliciesResponse{}, nil
}

func (f *fakeProtectionPolicyOCIClient) UpdateProtectionPolicy(ctx context.Context, req recoverysdk.UpdateProtectionPolicyRequest) (recoverysdk.UpdateProtectionPolicyResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return recoverysdk.UpdateProtectionPolicyResponse{}, nil
}

func (f *fakeProtectionPolicyOCIClient) DeleteProtectionPolicy(ctx context.Context, req recoverysdk.DeleteProtectionPolicyRequest) (recoverysdk.DeleteProtectionPolicyResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return recoverysdk.DeleteProtectionPolicyResponse{}, nil
}

func (f *fakeProtectionPolicyOCIClient) GetWorkRequest(ctx context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, req)
	}
	return recoverysdk.GetWorkRequestResponse{}, nil
}

func testProtectionPolicyClient(fake *fakeProtectionPolicyOCIClient) ProtectionPolicyServiceClient {
	if fake == nil {
		fake = &fakeProtectionPolicyOCIClient{}
	}
	return newProtectionPolicyServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		fake,
	)
}

func makeProtectionPolicyResource() *recoveryv1beta1.ProtectionPolicy {
	return &recoveryv1beta1.ProtectionPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-alpha",
			Namespace: "default",
			UID:       "protection-policy-uid",
		},
		Spec: recoveryv1beta1.ProtectionPolicySpec{
			DisplayName:                 testProtectionPolicyDisplayName,
			BackupRetentionPeriodInDays: 35,
			CompartmentId:               testProtectionPolicyCompartmentID,
			MustEnforceCloudLocality:    false,
			FreeformTags:                map[string]string{"managed-by": "osok"},
			DefinedTags:                 map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeSDKProtectionPolicy(id string, state recoverysdk.LifecycleStateEnum) recoverysdk.ProtectionPolicy {
	return makeSDKProtectionPolicyWithDetails(
		id,
		state,
		testProtectionPolicyDisplayName,
		35,
		false,
		map[string]string{"managed-by": "osok"},
		map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	)
}

func makeSDKProtectionPolicyWithDetails(
	id string,
	state recoverysdk.LifecycleStateEnum,
	displayName string,
	retentionDays int,
	mustEnforceCloudLocality bool,
	freeformTags map[string]string,
	definedTags map[string]map[string]interface{},
) recoverysdk.ProtectionPolicy {
	return recoverysdk.ProtectionPolicy{
		Id:                          common.String(id),
		CompartmentId:               common.String(testProtectionPolicyCompartmentID),
		BackupRetentionPeriodInDays: common.Int(retentionDays),
		IsPredefinedPolicy:          common.Bool(false),
		DisplayName:                 common.String(displayName),
		MustEnforceCloudLocality:    common.Bool(mustEnforceCloudLocality),
		LifecycleState:              state,
		FreeformTags:                freeformTags,
		DefinedTags:                 definedTags,
	}
}

func makeSDKProtectionPolicySummary(id string, state recoverysdk.LifecycleStateEnum) recoverysdk.ProtectionPolicySummary {
	return makeSDKProtectionPolicySummaryWithDetails(id, state, testProtectionPolicyDisplayName, 35, false)
}

func makeSDKProtectionPolicySummaryWithDetails(
	id string,
	state recoverysdk.LifecycleStateEnum,
	displayName string,
	retentionDays int,
	mustEnforceCloudLocality bool,
) recoverysdk.ProtectionPolicySummary {
	policy := makeSDKProtectionPolicy(id, state)
	policy.DisplayName = common.String(displayName)
	policy.BackupRetentionPeriodInDays = common.Int(retentionDays)
	policy.MustEnforceCloudLocality = common.Bool(mustEnforceCloudLocality)
	return recoverysdk.ProtectionPolicySummary{
		Id:                          policy.Id,
		CompartmentId:               policy.CompartmentId,
		BackupRetentionPeriodInDays: policy.BackupRetentionPeriodInDays,
		IsPredefinedPolicy:          policy.IsPredefinedPolicy,
		DisplayName:                 policy.DisplayName,
		MustEnforceCloudLocality:    policy.MustEnforceCloudLocality,
		LifecycleState:              policy.LifecycleState,
		FreeformTags:                policy.FreeformTags,
		DefinedTags:                 policy.DefinedTags,
	}
}

func makeProtectionPolicyWorkRequest(
	id string,
	operationType recoverysdk.OperationTypeEnum,
	status recoverysdk.OperationStatusEnum,
	action recoverysdk.ActionTypeEnum,
	protectionPolicyID string,
) recoverysdk.WorkRequest {
	percentComplete := float32(45)
	workRequest := recoverysdk.WorkRequest{
		OperationType:   operationType,
		Status:          status,
		Id:              common.String(id),
		PercentComplete: &percentComplete,
	}
	if protectionPolicyID != "" {
		workRequest.Resources = []recoverysdk.WorkRequestResource{
			{
				EntityType: common.String("ProtectionPolicy"),
				ActionType: action,
				Identifier: common.String(protectionPolicyID),
				EntityUri:  common.String("/20210216/protectionPolicies/" + protectionPolicyID),
			},
		}
	}
	return workRequest
}

func protectionPolicyOperationTypeForPhase(phase shared.OSOKAsyncPhase) recoverysdk.OperationTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return recoverysdk.OperationTypeCreateProtectionPolicy
	case shared.OSOKAsyncPhaseUpdate:
		return recoverysdk.OperationTypeUpdateProtectionPolicy
	case shared.OSOKAsyncPhaseDelete:
		return recoverysdk.OperationTypeDeleteProtectionPolicy
	default:
		return ""
	}
}

func protectionPolicySucceededWorkRequest(
	t *testing.T,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) func(context.Context, recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
	t.Helper()
	return func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
		requireProtectionPolicyStringPtr(t, "get workRequestId", req.WorkRequestId, workRequestID)
		return recoverysdk.GetWorkRequestResponse{
			WorkRequest: makeProtectionPolicyWorkRequest(
				workRequestID,
				protectionPolicyOperationTypeForPhase(phase),
				recoverysdk.OperationStatusSucceeded,
				protectionPolicyWorkRequestActionForPhase(phase),
				testProtectionPolicyID,
			),
		}, nil
	}
}

func TestProtectionPolicyRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := newProtectionPolicyRuntimeSemantics()
	if got == nil {
		t.Fatal("newProtectionPolicyRuntimeSemantics() = nil")
	}
	if got.Async == nil || got.Async.Strategy != "workrequest" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async = %#v, want workrequest generatedruntime", got.Async)
	}
	if got.Async.WorkRequest == nil || got.Async.WorkRequest.Source != "service-sdk" {
		t.Fatalf("Async.WorkRequest = %#v, want service-sdk work request contract", got.Async.WorkRequest)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete policy = %q follow-up = %q, want retained confirm-delete", got.FinalizerPolicy, got.DeleteFollowUp.Strategy)
	}
	assertProtectionPolicyStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertProtectionPolicyStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "id"})
	assertProtectionPolicyStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertProtectionPolicyStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertProtectionPolicyStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertProtectionPolicyStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETE_SCHEDULED", "DELETING"})
	assertProtectionPolicyStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertProtectionPolicyStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{
		"displayName",
		"backupRetentionPeriodInDays",
		"policyLockedDateTime",
		"freeformTags",
		"definedTags",
	})
	assertProtectionPolicyStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "mustEnforceCloudLocality"})
}

func TestProtectionPolicyCreateOrUpdateCreatesAndRefreshesStatus(t *testing.T) {
	t.Parallel()

	listCalls := 0
	var createRequest recoverysdk.CreateProtectionPolicyRequest
	var getRequest recoverysdk.GetProtectionPolicyRequest
	getWorkRequestCalls := 0
	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		listFn: func(_ context.Context, req recoverysdk.ListProtectionPoliciesRequest) (recoverysdk.ListProtectionPoliciesResponse, error) {
			listCalls++
			assertProtectionPolicyListRequest(t, req, nil)
			return recoverysdk.ListProtectionPoliciesResponse{}, nil
		},
		createFn: func(_ context.Context, req recoverysdk.CreateProtectionPolicyRequest) (recoverysdk.CreateProtectionPolicyResponse, error) {
			createRequest = req
			return recoverysdk.CreateProtectionPolicyResponse{
				ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create-1"),
				OpcRequestId:     common.String("opc-create-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			getWorkRequestCalls++
			requireProtectionPolicyStringPtr(t, "get workRequestId", req.WorkRequestId, "wr-create-1")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectionPolicyWorkRequest(
					"wr-create-1",
					recoverysdk.OperationTypeCreateProtectionPolicy,
					recoverysdk.OperationStatusSucceeded,
					recoverysdk.ActionTypeCreated,
					testProtectionPolicyID,
				),
			}, nil
		},
		getFn: func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
			getRequest = req
			return recoverysdk.GetProtectionPolicyResponse{
				ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateActive),
			}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue create", response)
	}
	if listCalls != 1 {
		t.Fatalf("ListProtectionPolicies() calls = %d, want 1", listCalls)
	}
	if getWorkRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", getWorkRequestCalls)
	}
	assertProtectionPolicyCreateRequest(t, createRequest, resource)
	requireProtectionPolicyStringPtr(t, "get protectionPolicyId", getRequest.ProtectionPolicyId, testProtectionPolicyID)
	assertProtectionPolicyActiveStatus(t, resource, "opc-create-1")
}

func TestProtectionPolicyCreateOrUpdateTracksPendingCreateWorkRequest(t *testing.T) {
	t.Parallel()

	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		listFn: func(context.Context, recoverysdk.ListProtectionPoliciesRequest) (recoverysdk.ListProtectionPoliciesResponse, error) {
			return recoverysdk.ListProtectionPoliciesResponse{}, nil
		},
		createFn: func(context.Context, recoverysdk.CreateProtectionPolicyRequest) (recoverysdk.CreateProtectionPolicyResponse, error) {
			return recoverysdk.CreateProtectionPolicyResponse{
				ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateCreating),
				OpcWorkRequestId: common.String("wr-create-pending"),
				OpcRequestId:     common.String("opc-create-pending"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			requireProtectionPolicyStringPtr(t, "get workRequestId", req.WorkRequestId, "wr-create-pending")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectionPolicyWorkRequest(
					"wr-create-pending",
					recoverysdk.OperationTypeCreateProtectionPolicy,
					recoverysdk.OperationStatusInProgress,
					recoverysdk.ActionTypeInProgress,
					testProtectionPolicyID,
				),
			}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	current := requireProtectionPolicyAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-pending", shared.OSOKAsyncClassPending)
	if current.RawStatus != string(recoverysdk.OperationStatusInProgress) {
		t.Fatalf("status.async.current.rawStatus = %q, want IN_PROGRESS", current.RawStatus)
	}
	if current.RawOperationType != string(recoverysdk.OperationTypeCreateProtectionPolicy) {
		t.Fatalf("status.async.current.rawOperationType = %q, want CREATE_PROTECTION_POLICY", current.RawOperationType)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-pending" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-pending", got)
	}
}

func TestProtectionPolicyCreateOrUpdateBindsExistingAcrossListPages(t *testing.T) {
	t.Parallel()

	createCalls := 0
	getCalls := 0
	listCalls := 0
	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		listFn: paginatedProtectionPolicyListStub(t, &listCalls),
		getFn: func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
			getCalls++
			requireProtectionPolicyStringPtr(t, "get protectionPolicyId", req.ProtectionPolicyId, testProtectionPolicyID)
			return recoverysdk.GetProtectionPolicyResponse{
				ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, recoverysdk.CreateProtectionPolicyRequest) (recoverysdk.CreateProtectionPolicyResponse, error) {
			createCalls++
			return recoverysdk.CreateProtectionPolicyResponse{}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue bind", response)
	}
	if createCalls != 0 {
		t.Fatalf("CreateProtectionPolicy() calls = %d, want 0", createCalls)
	}
	if listCalls != 2 {
		t.Fatalf("ListProtectionPolicies() calls = %d, want 2", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetProtectionPolicy() calls = %d, want 1", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testProtectionPolicyID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testProtectionPolicyID)
	}
}

func TestProtectionPolicyCreateOrUpdateBindsExistingWithMutableDriftThenUpdates(t *testing.T) {
	t.Parallel()

	recorder := &protectionPolicyMutableDriftBindRecorder{}
	client := newProtectionPolicyMutableDriftBindClient(t, recorder)
	resource := makeProtectionPolicyResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertProtectionPolicyMutableDriftBindResult(t, response, recorder, resource)
}

type protectionPolicyMutableDriftBindRecorder struct {
	createCalls   int
	getCalls      int
	listCalls     int
	updateCalls   int
	updateRequest recoverysdk.UpdateProtectionPolicyRequest
}

func newProtectionPolicyMutableDriftBindClient(
	t *testing.T,
	recorder *protectionPolicyMutableDriftBindRecorder,
) ProtectionPolicyServiceClient {
	t.Helper()
	return testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		listFn:           recorder.listExistingMutableDrift(t),
		getFn:            recorder.getExistingThenUpdatedPolicy(t),
		createFn:         recorder.recordCreateCall,
		updateFn:         recorder.recordUpdateCall,
		getWorkRequestFn: protectionPolicySucceededWorkRequest(t, "wr-update-bind", shared.OSOKAsyncPhaseUpdate),
	})
}

func (r *protectionPolicyMutableDriftBindRecorder) listExistingMutableDrift(
	t *testing.T,
) func(context.Context, recoverysdk.ListProtectionPoliciesRequest) (recoverysdk.ListProtectionPoliciesResponse, error) {
	t.Helper()
	return func(_ context.Context, req recoverysdk.ListProtectionPoliciesRequest) (recoverysdk.ListProtectionPoliciesResponse, error) {
		r.listCalls++
		assertProtectionPolicyListRequest(t, req, nil)
		return recoverysdk.ListProtectionPoliciesResponse{
			ProtectionPolicyCollection: recoverysdk.ProtectionPolicyCollection{
				Items: []recoverysdk.ProtectionPolicySummary{
					makeSDKProtectionPolicySummaryWithDetails(
						testProtectionPolicyID,
						recoverysdk.LifecycleStateActive,
						testProtectionPolicyDisplayName,
						21,
						true,
					),
				},
			},
		}, nil
	}
}

func (r *protectionPolicyMutableDriftBindRecorder) getExistingThenUpdatedPolicy(
	t *testing.T,
) func(context.Context, recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
	t.Helper()
	return func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
		r.getCalls++
		requireProtectionPolicyStringPtr(t, "get protectionPolicyId", req.ProtectionPolicyId, testProtectionPolicyID)
		if r.getCalls == 1 {
			return recoverysdk.GetProtectionPolicyResponse{ProtectionPolicy: makeSDKProtectionPolicyWithDetails(
				testProtectionPolicyID,
				recoverysdk.LifecycleStateActive,
				testProtectionPolicyDisplayName,
				21,
				false,
				map[string]string{"managed-by": "legacy"},
				map[string]map[string]interface{}{"Operations": {"CostCenter": "7"}},
			)}, nil
		}
		return recoverysdk.GetProtectionPolicyResponse{
			ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateActive),
		}, nil
	}
}

func (r *protectionPolicyMutableDriftBindRecorder) recordCreateCall(
	context.Context,
	recoverysdk.CreateProtectionPolicyRequest,
) (recoverysdk.CreateProtectionPolicyResponse, error) {
	r.createCalls++
	return recoverysdk.CreateProtectionPolicyResponse{}, nil
}

func (r *protectionPolicyMutableDriftBindRecorder) recordUpdateCall(
	_ context.Context,
	req recoverysdk.UpdateProtectionPolicyRequest,
) (recoverysdk.UpdateProtectionPolicyResponse, error) {
	r.updateCalls++
	r.updateRequest = req
	return recoverysdk.UpdateProtectionPolicyResponse{
		OpcWorkRequestId: common.String("wr-update-bind"),
		OpcRequestId:     common.String("opc-update-bind"),
	}, nil
}

func assertProtectionPolicyMutableDriftBindResult(
	t *testing.T,
	response servicemanager.OSOKResponse,
	recorder *protectionPolicyMutableDriftBindRecorder,
	resource *recoveryv1beta1.ProtectionPolicy,
) {
	t.Helper()
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue bind update", response)
	}
	if recorder.createCalls != 0 {
		t.Fatalf("CreateProtectionPolicy() calls = %d, want 0", recorder.createCalls)
	}
	if recorder.listCalls != 1 || recorder.getCalls != 2 || recorder.updateCalls != 1 {
		t.Fatalf("call counts list/get/update = %d/%d/%d, want 1/2/1", recorder.listCalls, recorder.getCalls, recorder.updateCalls)
	}
	requireProtectionPolicyStringPtr(t, "update protectionPolicyId", recorder.updateRequest.ProtectionPolicyId, testProtectionPolicyID)
	requireProtectionPolicyIntPtr(t, "update backupRetentionPeriodInDays", recorder.updateRequest.BackupRetentionPeriodInDays, resource.Spec.BackupRetentionPeriodInDays)
	if got := recorder.updateRequest.FreeformTags["managed-by"]; got != "osok" {
		t.Fatalf("update freeformTags[managed-by] = %q, want osok", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-bind" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-bind", got)
	}
}

func TestProtectionPolicyCreateOrUpdateDoesNotUpdateMatchingReadback(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
			requireProtectionPolicyStringPtr(t, "get protectionPolicyId", req.ProtectionPolicyId, testProtectionPolicyID)
			return recoverysdk.GetProtectionPolicyResponse{
				ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, recoverysdk.UpdateProtectionPolicyRequest) (recoverysdk.UpdateProtectionPolicyResponse, error) {
			updateCalls++
			return recoverysdk.UpdateProtectionPolicyResponse{}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProtectionPolicyID)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue observe", response)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateProtectionPolicy() calls = %d, want 0", updateCalls)
	}
	if got := resource.Status.LifecycleState; got != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestProtectionPolicyCreateOrUpdateUpdatesMutableFieldsAndClearsTags(t *testing.T) {
	t.Parallel()

	getCalls := 0
	var updateRequest recoverysdk.UpdateProtectionPolicyRequest
	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
			getCalls++
			requireProtectionPolicyStringPtr(t, "get protectionPolicyId", req.ProtectionPolicyId, testProtectionPolicyID)
			if getCalls == 1 {
				return recoverysdk.GetProtectionPolicyResponse{
					ProtectionPolicy: makeSDKProtectionPolicyWithDetails(
						testProtectionPolicyID,
						recoverysdk.LifecycleStateActive,
						"old-policy",
						21,
						false,
						map[string]string{"managed-by": "legacy"},
						map[string]map[string]interface{}{"Operations": {"CostCenter": "7"}},
					),
				}, nil
			}
			return recoverysdk.GetProtectionPolicyResponse{
				ProtectionPolicy: makeSDKProtectionPolicyWithDetails(
					testProtectionPolicyID,
					recoverysdk.LifecycleStateActive,
					"policy-renamed",
					42,
					false,
					map[string]string{},
					map[string]map[string]interface{}{},
				),
			}, nil
		},
		updateFn: func(_ context.Context, req recoverysdk.UpdateProtectionPolicyRequest) (recoverysdk.UpdateProtectionPolicyResponse, error) {
			updateRequest = req
			return recoverysdk.UpdateProtectionPolicyResponse{
				OpcWorkRequestId: common.String("wr-update-1"),
				OpcRequestId:     common.String("opc-update-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			requireProtectionPolicyStringPtr(t, "get workRequestId", req.WorkRequestId, "wr-update-1")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectionPolicyWorkRequest(
					"wr-update-1",
					recoverysdk.OperationTypeUpdateProtectionPolicy,
					recoverysdk.OperationStatusSucceeded,
					recoverysdk.ActionTypeUpdated,
					testProtectionPolicyID,
				),
			}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProtectionPolicyID)
	resource.Spec.DisplayName = "policy-renamed"
	resource.Spec.BackupRetentionPeriodInDays = 42
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue update", response)
	}
	requireProtectionPolicyStringPtr(t, "update protectionPolicyId", updateRequest.ProtectionPolicyId, testProtectionPolicyID)
	requireProtectionPolicyStringPtr(t, "update displayName", updateRequest.DisplayName, "policy-renamed")
	requireProtectionPolicyIntPtr(t, "update backupRetentionPeriodInDays", updateRequest.BackupRetentionPeriodInDays, 42)
	if updateRequest.FreeformTags == nil || len(updateRequest.FreeformTags) != 0 {
		t.Fatalf("update freeformTags = %#v, want explicit empty map", updateRequest.FreeformTags)
	}
	if updateRequest.DefinedTags == nil || len(updateRequest.DefinedTags) != 0 {
		t.Fatalf("update definedTags = %#v, want explicit empty map", updateRequest.DefinedTags)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestProtectionPolicyCreateOrUpdateProjectsFailedUpdateWorkRequest(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	getCalls := 0
	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
			getCalls++
			requireProtectionPolicyStringPtr(t, "get protectionPolicyId", req.ProtectionPolicyId, testProtectionPolicyID)
			return recoverysdk.GetProtectionPolicyResponse{
				ProtectionPolicy: makeSDKProtectionPolicyWithDetails(
					testProtectionPolicyID,
					recoverysdk.LifecycleStateActive,
					"old-policy",
					21,
					false,
					map[string]string{"managed-by": "legacy"},
					map[string]map[string]interface{}{"Operations": {"CostCenter": "7"}},
				),
			}, nil
		},
		updateFn: func(context.Context, recoverysdk.UpdateProtectionPolicyRequest) (recoverysdk.UpdateProtectionPolicyResponse, error) {
			updateCalls++
			return recoverysdk.UpdateProtectionPolicyResponse{
				OpcWorkRequestId: common.String("wr-update-failed"),
				OpcRequestId:     common.String("opc-update-failed"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			requireProtectionPolicyStringPtr(t, "get workRequestId", req.WorkRequestId, "wr-update-failed")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectionPolicyWorkRequest(
					"wr-update-failed",
					recoverysdk.OperationTypeUpdateProtectionPolicy,
					recoverysdk.OperationStatusFailed,
					recoverysdk.ActionTypeUpdated,
					testProtectionPolicyID,
				),
			}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProtectionPolicyID)
	resource.Spec.DisplayName = "policy-renamed"
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want failed work request error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful failed update", response)
	}
	if getCalls != 1 || updateCalls != 1 {
		t.Fatalf("call counts get/update = %d/%d, want 1/1", getCalls, updateCalls)
	}
	requireProtectionPolicyAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update-failed", shared.OSOKAsyncClassFailed)
	if got := lastProtectionPolicyCondition(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func TestProtectionPolicyCreateOrUpdateRejectsCreateOnlyDrift(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mutateDesired func(*recoveryv1beta1.ProtectionPolicy)
		wantField     string
	}{
		{
			name: "compartmentId",
			mutateDesired: func(resource *recoveryv1beta1.ProtectionPolicy) {
				resource.Spec.CompartmentId = "ocid1.compartment.oc1..replacement"
			},
			wantField: "compartmentId",
		},
		{
			name: "mustEnforceCloudLocality",
			mutateDesired: func(resource *recoveryv1beta1.ProtectionPolicy) {
				resource.Spec.MustEnforceCloudLocality = true
			},
			wantField: "mustEnforceCloudLocality",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			updateCalls := 0
			client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
				getFn: func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
					requireProtectionPolicyStringPtr(t, "get protectionPolicyId", req.ProtectionPolicyId, testProtectionPolicyID)
					return recoverysdk.GetProtectionPolicyResponse{
						ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateActive),
					}, nil
				},
				updateFn: func(context.Context, recoverysdk.UpdateProtectionPolicyRequest) (recoverysdk.UpdateProtectionPolicyResponse, error) {
					updateCalls++
					return recoverysdk.UpdateProtectionPolicyResponse{}, nil
				},
			})

			resource := makeProtectionPolicyResource()
			resource.Status.OsokStatus.Ocid = shared.OCID(testProtectionPolicyID)
			tt.mutateDesired(resource)
			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err == nil {
				t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
			}
			if !strings.Contains(err.Error(), tt.wantField) {
				t.Fatalf("CreateOrUpdate() error = %v, want field %q", err, tt.wantField)
			}
			if response.IsSuccessful {
				t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift rejection", response)
			}
			if updateCalls != 0 {
				t.Fatalf("UpdateProtectionPolicy() calls = %d, want 0", updateCalls)
			}
		})
	}
}

func TestProtectionPolicyDeleteRetainsFinalizerUntilReadbackGone(t *testing.T) {
	t.Parallel()

	getCalls := 0
	var deleteRequest recoverysdk.DeleteProtectionPolicyRequest
	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
			getCalls++
			requireProtectionPolicyStringPtr(t, "get protectionPolicyId", req.ProtectionPolicyId, testProtectionPolicyID)
			if getCalls == 2 {
				return recoverysdk.GetProtectionPolicyResponse{
					ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateDeleting),
				}, nil
			}
			return recoverysdk.GetProtectionPolicyResponse{
				ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req recoverysdk.DeleteProtectionPolicyRequest) (recoverysdk.DeleteProtectionPolicyResponse, error) {
			deleteRequest = req
			return recoverysdk.DeleteProtectionPolicyResponse{
				OpcWorkRequestId: common.String("wr-delete-1"),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			requireProtectionPolicyStringPtr(t, "get workRequestId", req.WorkRequestId, "wr-delete-1")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectionPolicyWorkRequest(
					"wr-delete-1",
					recoverysdk.OperationTypeDeleteProtectionPolicy,
					recoverysdk.OperationStatusSucceeded,
					recoverysdk.ActionTypeDeleted,
					testProtectionPolicyID,
				),
			}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProtectionPolicyID)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while readback still exists")
	}
	if getCalls != 2 {
		t.Fatalf("GetProtectionPolicy() calls = %d, want pre-delete and confirm reads", getCalls)
	}
	requireProtectionPolicyStringPtr(t, "delete protectionPolicyId", deleteRequest.ProtectionPolicyId, testProtectionPolicyID)
	assertProtectionPolicyDeletePendingStatus(t, resource, "opc-delete-1")
	if got := resource.Status.OsokStatus.Async.Current.WorkRequestID; got != "wr-delete-1" {
		t.Fatalf("status.async.current.workRequestId = %q, want wr-delete-1", got)
	}
}

func TestProtectionPolicyDeleteFreshSucceededWorkRequestRetainsFinalizerWhenReadbackStillActive(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0
	getWorkRequestCalls := 0
	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
			getCalls++
			requireProtectionPolicyStringPtr(t, "get protectionPolicyId", req.ProtectionPolicyId, testProtectionPolicyID)
			return recoverysdk.GetProtectionPolicyResponse{
				ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateActive),
			}, nil
		},
		deleteFn: func(context.Context, recoverysdk.DeleteProtectionPolicyRequest) (recoverysdk.DeleteProtectionPolicyResponse, error) {
			deleteCalls++
			return recoverysdk.DeleteProtectionPolicyResponse{
				OpcWorkRequestId: common.String("wr-delete-1"),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			getWorkRequestCalls++
			requireProtectionPolicyStringPtr(t, "get workRequestId", req.WorkRequestId, "wr-delete-1")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectionPolicyWorkRequest(
					"wr-delete-1",
					recoverysdk.OperationTypeDeleteProtectionPolicy,
					recoverysdk.OperationStatusSucceeded,
					recoverysdk.ActionTypeDeleted,
					testProtectionPolicyID,
				),
			}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProtectionPolicyID)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v, want retained finalizer without lifecycle error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while readback still exists")
	}
	if getCalls != 2 || deleteCalls != 1 || getWorkRequestCalls != 1 {
		t.Fatalf("call counts get/delete/workRequest = %d/%d/%d, want 2/1/1", getCalls, deleteCalls, getWorkRequestCalls)
	}
	assertProtectionPolicyDeletePendingStatus(t, resource, "opc-delete-1")
	if got := resource.Status.OsokStatus.Async.Current.WorkRequestID; got != "wr-delete-1" {
		t.Fatalf("status.async.current.workRequestId = %q, want wr-delete-1", got)
	}
	if got := resource.Status.LifecycleState; got != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestProtectionPolicyDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	t.Parallel()

	getCalls := 0
	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
			getCalls++
			requireProtectionPolicyStringPtr(t, "get protectionPolicyId", req.ProtectionPolicyId, testProtectionPolicyID)
			if getCalls == 1 {
				return recoverysdk.GetProtectionPolicyResponse{
					ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateActive),
				}, nil
			}
			return recoverysdk.GetProtectionPolicyResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
		},
		deleteFn: func(context.Context, recoverysdk.DeleteProtectionPolicyRequest) (recoverysdk.DeleteProtectionPolicyResponse, error) {
			return recoverysdk.DeleteProtectionPolicyResponse{
				OpcWorkRequestId: common.String("wr-delete-1"),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			requireProtectionPolicyStringPtr(t, "get workRequestId", req.WorkRequestId, "wr-delete-1")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectionPolicyWorkRequest(
					"wr-delete-1",
					recoverysdk.OperationTypeDeleteProtectionPolicy,
					recoverysdk.OperationStatusSucceeded,
					recoverysdk.ActionTypeDeleted,
					testProtectionPolicyID,
				),
			}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProtectionPolicyID)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want delete confirmed by unambiguous 404")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want delete request ID", got)
	}
}

func TestProtectionPolicyDeleteKeepsPreDeleteAuthShapedConfirmReadAmbiguous(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0
	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
			getCalls++
			requireProtectionPolicyStringPtr(t, "get protectionPolicyId", req.ProtectionPolicyId, testProtectionPolicyID)
			return recoverysdk.GetProtectionPolicyResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, recoverysdk.DeleteProtectionPolicyRequest) (recoverysdk.DeleteProtectionPolicyResponse, error) {
			deleteCalls++
			t.Fatal("DeleteProtectionPolicy() was called after ambiguous pre-delete confirm read")
			return recoverysdk.DeleteProtectionPolicyResponse{}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProtectionPolicyID)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete confirm-read error")
	}
	if !strings.Contains(err.Error(), "retaining finalizer") {
		t.Fatalf("Delete() error = %v, want retaining finalizer", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous pre-delete confirm read")
	}
	if getCalls != 1 {
		t.Fatalf("GetProtectionPolicy() calls = %d, want 1", getCalls)
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteProtectionPolicy() calls = %d, want 0", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want confirm-read request ID", got)
	}
	assertProtectionPolicyDeletePendingStatus(t, resource, "opc-request-id")
}

func TestProtectionPolicyDeleteSkipsDeleteWhenAlreadyPending(t *testing.T) {
	t.Parallel()

	deleteCalls := 0
	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
			requireProtectionPolicyStringPtr(t, "get protectionPolicyId", req.ProtectionPolicyId, testProtectionPolicyID)
			return recoverysdk.GetProtectionPolicyResponse{
				ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateActive),
			}, nil
		},
		deleteFn: func(context.Context, recoverysdk.DeleteProtectionPolicyRequest) (recoverysdk.DeleteProtectionPolicyResponse, error) {
			deleteCalls++
			return recoverysdk.DeleteProtectionPolicyResponse{}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProtectionPolicyID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteProtectionPolicy() calls = %d, want 0", deleteCalls)
	}
	assertProtectionPolicyDeletePendingStatus(t, resource, "")
}

func TestProtectionPolicyDeleteProjectsFailedDeleteWorkRequest(t *testing.T) {
	t.Parallel()

	deleteCalls := 0
	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		getWorkRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			requireProtectionPolicyStringPtr(t, "get workRequestId", req.WorkRequestId, "wr-delete-failed")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectionPolicyWorkRequest(
					"wr-delete-failed",
					recoverysdk.OperationTypeDeleteProtectionPolicy,
					recoverysdk.OperationStatusFailed,
					recoverysdk.ActionTypeDeleted,
					testProtectionPolicyID,
				),
			}, nil
		},
		deleteFn: func(context.Context, recoverysdk.DeleteProtectionPolicyRequest) (recoverysdk.DeleteProtectionPolicyResponse, error) {
			deleteCalls++
			return recoverysdk.DeleteProtectionPolicyResponse{}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProtectionPolicyID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:        shared.OSOKAsyncSourceWorkRequest,
		Phase:         shared.OSOKAsyncPhaseDelete,
		WorkRequestID: "wr-delete-failed",
	}
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want failed delete work request error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteProtectionPolicy() calls = %d, want 0", deleteCalls)
	}
	requireProtectionPolicyAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete-failed", shared.OSOKAsyncClassFailed)
	if got := lastProtectionPolicyCondition(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func TestProtectionPolicyDeleteSucceededWorkRequestRetainsFinalizerWhenReadbackStillExists(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0
	getWorkRequestCalls := 0
	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
			getCalls++
			requireProtectionPolicyStringPtr(t, "get protectionPolicyId", req.ProtectionPolicyId, testProtectionPolicyID)
			return recoverysdk.GetProtectionPolicyResponse{
				ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateActive),
			}, nil
		},
		getWorkRequestFn: func(ctx context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			getWorkRequestCalls++
			return protectionPolicySucceededWorkRequest(t, "wr-delete-succeeded", shared.OSOKAsyncPhaseDelete)(ctx, req)
		},
		deleteFn: func(context.Context, recoverysdk.DeleteProtectionPolicyRequest) (recoverysdk.DeleteProtectionPolicyResponse, error) {
			deleteCalls++
			return recoverysdk.DeleteProtectionPolicyResponse{}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProtectionPolicyID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete-succeeded",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v, want retained finalizer without lifecycle error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while readback still exists")
	}
	if getWorkRequestCalls != 1 || getCalls != 1 || deleteCalls != 0 {
		t.Fatalf("call counts workRequest/get/delete = %d/%d/%d, want 1/1/0", getWorkRequestCalls, getCalls, deleteCalls)
	}
	assertProtectionPolicyDeletePendingStatus(t, resource, "")
	if got := resource.Status.LifecycleState; got != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestProtectionPolicyDeleteRetainsFinalizerForPendingWrite(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name  string
		phase shared.OSOKAsyncPhase
	}{
		{name: "create", phase: shared.OSOKAsyncPhaseCreate},
		{name: "update", phase: shared.OSOKAsyncPhaseUpdate},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runProtectionPolicyPendingWriteDeleteGuardCase(t, tt.phase)
		})
	}
}

func runProtectionPolicyPendingWriteDeleteGuardCase(t *testing.T, phase shared.OSOKAsyncPhase) {
	t.Helper()

	getCalls := 0
	getWorkRequestCalls := 0
	deleteCalls := 0
	workRequestID := "wr-" + string(phase) + "-pending"
	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		getFn: func(context.Context, recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
			getCalls++
			return recoverysdk.GetProtectionPolicyResponse{}, nil
		},
		getWorkRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			getWorkRequestCalls++
			requireProtectionPolicyStringPtr(t, "get workRequestId", req.WorkRequestId, workRequestID)
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectionPolicyWorkRequest(
					workRequestID,
					protectionPolicyOperationTypeForPhase(phase),
					recoverysdk.OperationStatusInProgress,
					recoverysdk.ActionTypeInProgress,
					testProtectionPolicyID,
				),
			}, nil
		},
		deleteFn: func(context.Context, recoverysdk.DeleteProtectionPolicyRequest) (recoverysdk.DeleteProtectionPolicyResponse, error) {
			deleteCalls++
			return recoverysdk.DeleteProtectionPolicyResponse{}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProtectionPolicyID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if getCalls != 0 {
		t.Fatalf("GetProtectionPolicy() calls = %d, want 0", getCalls)
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteProtectionPolicy() calls = %d, want 0", deleteCalls)
	}
	if getWorkRequestCalls != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", getWorkRequestCalls)
	}
	requireProtectionPolicyAsync(t, resource, phase, workRequestID, shared.OSOKAsyncClassPending)
}

func TestProtectionPolicyDeleteKeepsAuthShapedNotFoundAmbiguous(t *testing.T) {
	t.Parallel()

	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
			requireProtectionPolicyStringPtr(t, "get protectionPolicyId", req.ProtectionPolicyId, testProtectionPolicyID)
			return recoverysdk.GetProtectionPolicyResponse{
				ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateActive),
			}, nil
		},
		deleteFn: func(context.Context, recoverysdk.DeleteProtectionPolicyRequest) (recoverysdk.DeleteProtectionPolicyResponse, error) {
			return recoverysdk.DeleteProtectionPolicyResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
		},
	})

	resource := makeProtectionPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProtectionPolicyID)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped 404 error")
	}
	if !strings.Contains(err.Error(), "retaining finalizer") {
		t.Fatalf("Delete() error = %v, want retaining finalizer", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous auth-shaped 404")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want error request ID", got)
	}
}

func TestProtectionPolicyDeleteKeepsPostDeleteAuthShapedConfirmReadAmbiguous(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0
	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		getFn: func(_ context.Context, req recoverysdk.GetProtectionPolicyRequest) (recoverysdk.GetProtectionPolicyResponse, error) {
			getCalls++
			requireProtectionPolicyStringPtr(t, "get protectionPolicyId", req.ProtectionPolicyId, testProtectionPolicyID)
			if getCalls == 1 {
				return recoverysdk.GetProtectionPolicyResponse{
					ProtectionPolicy: makeSDKProtectionPolicy(testProtectionPolicyID, recoverysdk.LifecycleStateActive),
				}, nil
			}
			return recoverysdk.GetProtectionPolicyResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, recoverysdk.DeleteProtectionPolicyRequest) (recoverysdk.DeleteProtectionPolicyResponse, error) {
			deleteCalls++
			return recoverysdk.DeleteProtectionPolicyResponse{
				OpcWorkRequestId: common.String("wr-delete-1"),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error) {
			requireProtectionPolicyStringPtr(t, "get workRequestId", req.WorkRequestId, "wr-delete-1")
			return recoverysdk.GetWorkRequestResponse{
				WorkRequest: makeProtectionPolicyWorkRequest(
					"wr-delete-1",
					recoverysdk.OperationTypeDeleteProtectionPolicy,
					recoverysdk.OperationStatusSucceeded,
					recoverysdk.ActionTypeDeleted,
					testProtectionPolicyID,
				),
			}, nil
		},
	})

	resource := makeProtectionPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProtectionPolicyID)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous confirm-read error")
	}
	if !strings.Contains(err.Error(), "retaining finalizer") {
		t.Fatalf("Delete() error = %v, want retaining finalizer", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous confirm read")
	}
	if getCalls != 2 {
		t.Fatalf("GetProtectionPolicy() calls = %d, want pre-delete and confirm reads", getCalls)
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteProtectionPolicy() calls = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want confirm-read request ID", got)
	}
}

func TestProtectionPolicyCreateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	client := testProtectionPolicyClient(&fakeProtectionPolicyOCIClient{
		listFn: func(context.Context, recoverysdk.ListProtectionPoliciesRequest) (recoverysdk.ListProtectionPoliciesResponse, error) {
			return recoverysdk.ListProtectionPoliciesResponse{}, nil
		},
		createFn: func(context.Context, recoverysdk.CreateProtectionPolicyRequest) (recoverysdk.CreateProtectionPolicyResponse, error) {
			return recoverysdk.CreateProtectionPolicyResponse{}, errortest.NewServiceError(400, "InvalidParameter", "bad request")
		},
	})

	resource := makeProtectionPolicyResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want OCI error request ID", got)
	}
	if got := lastProtectionPolicyCondition(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func paginatedProtectionPolicyListStub(t *testing.T, listCalls *int) protectionPolicyListCall {
	t.Helper()
	return func(_ context.Context, req recoverysdk.ListProtectionPoliciesRequest) (recoverysdk.ListProtectionPoliciesResponse, error) {
		(*listCalls)++
		assertProtectionPolicyListRequest(t, req, nil)
		if req.Owner != recoverysdk.ListProtectionPoliciesOwnerCustomer {
			t.Fatalf("list owner = %q, want customer", req.Owner)
		}
		switch *listCalls {
		case 1:
			if req.Page != nil {
				t.Fatalf("first list page = %v, want nil", req.Page)
			}
			return recoverysdk.ListProtectionPoliciesResponse{OpcNextPage: common.String("page-2")}, nil
		case 2:
			requireProtectionPolicyStringPtr(t, "second list page", req.Page, "page-2")
			return recoverysdk.ListProtectionPoliciesResponse{
				ProtectionPolicyCollection: recoverysdk.ProtectionPolicyCollection{
					Items: []recoverysdk.ProtectionPolicySummary{
						makeSDKProtectionPolicySummary(testProtectionPolicyID, recoverysdk.LifecycleStateActive),
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected ListProtectionPolicies() call %d", *listCalls)
			return recoverysdk.ListProtectionPoliciesResponse{}, nil
		}
	}
}

func assertProtectionPolicyListRequest(t *testing.T, req recoverysdk.ListProtectionPoliciesRequest, wantPage *string) {
	t.Helper()
	requireProtectionPolicyStringPtr(t, "list compartmentId", req.CompartmentId, testProtectionPolicyCompartmentID)
	requireProtectionPolicyStringPtr(t, "list displayName", req.DisplayName, testProtectionPolicyDisplayName)
	if req.Owner != recoverysdk.ListProtectionPoliciesOwnerCustomer {
		t.Fatalf("list owner = %q, want customer", req.Owner)
	}
	if wantPage != nil {
		requireProtectionPolicyStringPtr(t, "list page", req.Page, *wantPage)
	}
}

func assertProtectionPolicyCreateRequest(
	t *testing.T,
	request recoverysdk.CreateProtectionPolicyRequest,
	resource *recoveryv1beta1.ProtectionPolicy,
) {
	t.Helper()
	requireProtectionPolicyStringPtr(t, "create displayName", request.DisplayName, resource.Spec.DisplayName)
	requireProtectionPolicyStringPtr(t, "create compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireProtectionPolicyIntPtr(t, "create backupRetentionPeriodInDays", request.BackupRetentionPeriodInDays, resource.Spec.BackupRetentionPeriodInDays)
	if request.MustEnforceCloudLocality == nil || *request.MustEnforceCloudLocality {
		t.Fatalf("create mustEnforceCloudLocality = %v, want explicit false", request.MustEnforceCloudLocality)
	}
	requireProtectionPolicyStringPtr(t, "create opcRetryToken", request.OpcRetryToken, string(resource.UID))
	if got := request.FreeformTags["managed-by"]; got != "osok" {
		t.Fatalf("create freeformTags[managed-by] = %q, want osok", got)
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("create definedTags Operations.CostCenter = %#v, want 42", got)
	}
}

func assertProtectionPolicyActiveStatus(t *testing.T, resource *recoveryv1beta1.ProtectionPolicy, wantRequestID string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != testProtectionPolicyID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testProtectionPolicyID)
	}
	if got := resource.Status.Id; got != testProtectionPolicyID {
		t.Fatalf("status.id = %q, want %q", got, testProtectionPolicyID)
	}
	if got := resource.Status.LifecycleState; got != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != wantRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, wantRequestID)
	}
	if got := lastProtectionPolicyCondition(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after active readback", resource.Status.OsokStatus.Async.Current)
	}
}

func assertProtectionPolicyDeletePendingStatus(
	t *testing.T,
	resource *recoveryv1beta1.ProtectionPolicy,
	wantRequestID string,
) {
	t.Helper()
	if wantRequestID != "" && resource.Status.OsokStatus.OpcRequestID != wantRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, wantRequestID)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want delete tracker")
	}
	if current.Phase != shared.OSOKAsyncPhaseDelete || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current = %#v, want pending delete tracker", current)
	}
	if got := lastProtectionPolicyCondition(resource); got != shared.Terminating {
		t.Fatalf("last condition = %q, want Terminating", got)
	}
}

func requireProtectionPolicyAsync(
	t *testing.T,
	resource *recoveryv1beta1.ProtectionPolicy,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) *shared.OSOKAsyncOperation {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want workrequest", current.Source)
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
	return current
}

func requireProtectionPolicyStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", name, got, want)
	}
}

func requireProtectionPolicyIntPtr(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %d", name, got, want)
	}
}

func assertProtectionPolicyStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s len = %d, want %d (%#v)", name, len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q", name, i, got[i], want[i])
		}
	}
}

func lastProtectionPolicyCondition(resource *recoveryv1beta1.ProtectionPolicy) shared.OSOKConditionType {
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		return ""
	}
	return shared.OSOKConditionType(resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type)
}
