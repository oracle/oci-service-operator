/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vbsinstance

import (
	"context"
	"maps"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	vbsinstsdk "github.com/oracle/oci-go-sdk/v65/vbsinst"
	vbsinstv1beta1 "github.com/oracle/oci-service-operator/api/vbsinst/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeVbsInstanceOCIClient struct {
	createFn      func(context.Context, vbsinstsdk.CreateVbsInstanceRequest) (vbsinstsdk.CreateVbsInstanceResponse, error)
	getFn         func(context.Context, vbsinstsdk.GetVbsInstanceRequest) (vbsinstsdk.GetVbsInstanceResponse, error)
	listFn        func(context.Context, vbsinstsdk.ListVbsInstancesRequest) (vbsinstsdk.ListVbsInstancesResponse, error)
	updateFn      func(context.Context, vbsinstsdk.UpdateVbsInstanceRequest) (vbsinstsdk.UpdateVbsInstanceResponse, error)
	deleteFn      func(context.Context, vbsinstsdk.DeleteVbsInstanceRequest) (vbsinstsdk.DeleteVbsInstanceResponse, error)
	workRequestFn func(context.Context, vbsinstsdk.GetWorkRequestRequest) (vbsinstsdk.GetWorkRequestResponse, error)
}

func (f *fakeVbsInstanceOCIClient) CreateVbsInstance(
	ctx context.Context,
	req vbsinstsdk.CreateVbsInstanceRequest,
) (vbsinstsdk.CreateVbsInstanceResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return vbsinstsdk.CreateVbsInstanceResponse{}, nil
}

func (f *fakeVbsInstanceOCIClient) GetVbsInstance(
	ctx context.Context,
	req vbsinstsdk.GetVbsInstanceRequest,
) (vbsinstsdk.GetVbsInstanceResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return vbsinstsdk.GetVbsInstanceResponse{}, errortest.NewServiceError(404, "NotFound", "missing VbsInstance")
}

func (f *fakeVbsInstanceOCIClient) ListVbsInstances(
	ctx context.Context,
	req vbsinstsdk.ListVbsInstancesRequest,
) (vbsinstsdk.ListVbsInstancesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return vbsinstsdk.ListVbsInstancesResponse{}, nil
}

func (f *fakeVbsInstanceOCIClient) UpdateVbsInstance(
	ctx context.Context,
	req vbsinstsdk.UpdateVbsInstanceRequest,
) (vbsinstsdk.UpdateVbsInstanceResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return vbsinstsdk.UpdateVbsInstanceResponse{}, nil
}

func (f *fakeVbsInstanceOCIClient) DeleteVbsInstance(
	ctx context.Context,
	req vbsinstsdk.DeleteVbsInstanceRequest,
) (vbsinstsdk.DeleteVbsInstanceResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return vbsinstsdk.DeleteVbsInstanceResponse{}, nil
}

func (f *fakeVbsInstanceOCIClient) GetWorkRequest(
	ctx context.Context,
	req vbsinstsdk.GetWorkRequestRequest,
) (vbsinstsdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return vbsinstsdk.GetWorkRequestResponse{}, nil
}

func TestReviewedVbsInstanceRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedVbsInstanceRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedVbsInstanceRuntimeSemantics() = nil")
	}

	if got.FormalService != "vbsinst" {
		t.Fatalf("FormalService = %q, want vbsinst", got.FormalService)
	}
	if got.FormalSlug != "vbsinstance" {
		t.Fatalf("FormalSlug = %q, want vbsinstance", got.FormalSlug)
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
	assertVbsInstanceStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertVbsInstanceStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertVbsInstanceStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertVbsInstanceStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertVbsInstanceStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertVbsInstanceStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertVbsInstanceStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "name", "id"})
	assertVbsInstanceStringSliceEqual(
		t,
		"Mutation.Mutable",
		got.Mutation.Mutable,
		[]string{"definedTags", "displayName", "freeformTags", "isResourceUsageAgreementGranted", "resourceCompartmentId"},
	)
	assertVbsInstanceStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "name"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetVbsInstance" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want GetWorkRequest -> GetVbsInstance", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetVbsInstance" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want GetWorkRequest -> GetVbsInstance", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetWorkRequest -> GetVbsInstance/ListVbsInstances confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want workrequest-backed confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want none for published runtime", got.AuxiliaryOperations)
	}
}

func TestGuardVbsInstanceExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makeVbsInstanceResource()
	resource.Spec.Name = ""

	decision, err := guardVbsInstanceExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardVbsInstanceExistingBeforeCreate(empty name) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardVbsInstanceExistingBeforeCreate(empty name) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.Name = "vbsinstance-sample"
	decision, err = guardVbsInstanceExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardVbsInstanceExistingBeforeCreate(complete identity) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardVbsInstanceExistingBeforeCreate(complete identity) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestBuildVbsInstanceUpdateBodyPreservesTagClears(t *testing.T) {
	t.Parallel()

	currentResource := makeVbsInstanceResource()
	currentResource.Spec.IsResourceUsageAgreementGranted = true
	currentResource.Spec.ResourceCompartmentId = "ocid1.compartment.oc1..current"

	desired := makeVbsInstanceResource()
	desired.Spec.DisplayName = "vbsinstance-updated"
	desired.Spec.IsResourceUsageAgreementGranted = false
	desired.Spec.ResourceCompartmentId = "ocid1.compartment.oc1..next"
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}

	body, updateNeeded, err := buildVbsInstanceUpdateBody(
		desired,
		vbsinstsdk.GetVbsInstanceResponse{
			VbsInstance: makeSDKVbsInstance(
				"ocid1.vbsinstance.oc1..existing",
				currentResource,
				vbsinstsdk.LifecycleStateActive,
			),
		},
	)
	if err != nil {
		t.Fatalf("buildVbsInstanceUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildVbsInstanceUpdateBody() updateNeeded = false, want true")
	}

	requireVbsInstanceStringPtr(t, "details.displayName", body.DisplayName, desired.Spec.DisplayName)
	requireVbsInstanceBoolPtr(t, "details.isResourceUsageAgreementGranted", body.IsResourceUsageAgreementGranted, false)
	requireVbsInstanceStringPtr(t, "details.resourceCompartmentId", body.ResourceCompartmentId, desired.Spec.ResourceCompartmentId)
	if len(body.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map for clear", body.FreeformTags)
	}
	if len(body.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map for clear", body.DefinedTags)
	}
}

func TestVbsInstanceCreateOrUpdateCreatesWhenOnlyDisplayNameMatches(t *testing.T) {
	t.Parallel()

	resource := makeVbsInstanceResource()
	wrongIdentity := makeVbsInstanceResource()
	wrongIdentity.Spec.Name = "different-name"
	createCalls := 0

	client := newTestVbsInstanceClient(&fakeVbsInstanceOCIClient{
		listFn: func(_ context.Context, req vbsinstsdk.ListVbsInstancesRequest) (vbsinstsdk.ListVbsInstancesResponse, error) {
			requireVbsInstanceStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireVbsInstanceStringPtr(t, "list name", req.Name, resource.Spec.Name)
			if req.Id != nil {
				t.Fatalf("list id = %v, want nil before the resource is tracked", req.Id)
			}
			return vbsinstsdk.ListVbsInstancesResponse{
				VbsInstanceSummaryCollection: vbsinstsdk.VbsInstanceSummaryCollection{
					Items: []vbsinstsdk.VbsInstanceSummary{
						makeSDKVbsInstanceSummary("ocid1.vbsinstance.oc1..wrong", wrongIdentity, vbsinstsdk.LifecycleStateActive),
					},
				},
			}, nil
		},
		createFn: func(_ context.Context, req vbsinstsdk.CreateVbsInstanceRequest) (vbsinstsdk.CreateVbsInstanceResponse, error) {
			createCalls++
			requireVbsInstanceStringPtr(t, "create compartmentId", req.CreateVbsInstanceDetails.CompartmentId, resource.Spec.CompartmentId)
			requireVbsInstanceStringPtr(t, "create name", req.CreateVbsInstanceDetails.Name, resource.Spec.Name)
			return vbsinstsdk.CreateVbsInstanceResponse{
				OpcWorkRequestId: common.String("wr-create-displayname-only"),
				OpcRequestId:     common.String("opc-create-displayname-only"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req vbsinstsdk.GetWorkRequestRequest) (vbsinstsdk.GetWorkRequestResponse, error) {
			requireVbsInstanceStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-displayname-only")
			return vbsinstsdk.GetWorkRequestResponse{
				WorkRequest: makeVbsInstanceWorkRequest(
					"wr-create-displayname-only",
					vbsinstsdk.OperationTypeCreateVbsInstance,
					vbsinstsdk.OperationStatusInProgress,
					vbsinstsdk.ActionTypeInProgress,
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
	if createCalls != 1 {
		t.Fatalf("CreateVbsInstance() calls = %d, want 1", createCalls)
	}
	requireVbsInstanceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-displayname-only", shared.OSOKAsyncClassPending)
}

func TestVbsInstanceCreateOrUpdateRejectsAmbiguousNameReuse(t *testing.T) {
	t.Parallel()

	resource := makeVbsInstanceResource()
	createCalls := 0

	client := newTestVbsInstanceClient(&fakeVbsInstanceOCIClient{
		listFn: func(_ context.Context, req vbsinstsdk.ListVbsInstancesRequest) (vbsinstsdk.ListVbsInstancesResponse, error) {
			requireVbsInstanceStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireVbsInstanceStringPtr(t, "list name", req.Name, resource.Spec.Name)
			return vbsinstsdk.ListVbsInstancesResponse{
				VbsInstanceSummaryCollection: vbsinstsdk.VbsInstanceSummaryCollection{
					Items: []vbsinstsdk.VbsInstanceSummary{
						makeSDKVbsInstanceSummary("ocid1.vbsinstance.oc1..first", resource, vbsinstsdk.LifecycleStateActive),
						makeSDKVbsInstanceSummary("ocid1.vbsinstance.oc1..second", resource, vbsinstsdk.LifecycleStateUpdating),
					},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ vbsinstsdk.CreateVbsInstanceRequest) (vbsinstsdk.CreateVbsInstanceResponse, error) {
			createCalls++
			return vbsinstsdk.CreateVbsInstanceResponse{}, nil
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
		t.Fatalf("CreateVbsInstance() calls = %d, want 0 on ambiguous reuse", createCalls)
	}
}

func TestVbsInstanceServiceClientCreatesAndResumesWorkRequest(t *testing.T) {
	t.Parallel()

	const (
		createdID     = "ocid1.vbsinstance.oc1..created"
		workRequestID = "wr-vbsinstance-create"
	)

	resource := makeVbsInstanceResource()
	workRequests := map[string]vbsinstsdk.WorkRequest{
		workRequestID: makeVbsInstanceWorkRequest(
			workRequestID,
			vbsinstsdk.OperationTypeCreateVbsInstance,
			vbsinstsdk.OperationStatusInProgress,
			vbsinstsdk.ActionTypeInProgress,
			"",
		),
	}

	var createRequest vbsinstsdk.CreateVbsInstanceRequest
	getCalls := 0

	client := newTestVbsInstanceClient(&fakeVbsInstanceOCIClient{
		listFn: func(_ context.Context, req vbsinstsdk.ListVbsInstancesRequest) (vbsinstsdk.ListVbsInstancesResponse, error) {
			requireVbsInstanceStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireVbsInstanceStringPtr(t, "list name", req.Name, resource.Spec.Name)
			return vbsinstsdk.ListVbsInstancesResponse{}, nil
		},
		createFn: func(_ context.Context, req vbsinstsdk.CreateVbsInstanceRequest) (vbsinstsdk.CreateVbsInstanceResponse, error) {
			createRequest = req
			return vbsinstsdk.CreateVbsInstanceResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-vbsinstance"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req vbsinstsdk.GetWorkRequestRequest) (vbsinstsdk.GetWorkRequestResponse, error) {
			requireVbsInstanceStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return vbsinstsdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
		getFn: func(_ context.Context, req vbsinstsdk.GetVbsInstanceRequest) (vbsinstsdk.GetVbsInstanceResponse, error) {
			getCalls++
			requireVbsInstanceStringPtr(t, "get vbsInstanceId", req.VbsInstanceId, createdID)
			return vbsinstsdk.GetVbsInstanceResponse{
				VbsInstance: makeSDKVbsInstance(createdID, resource, vbsinstsdk.LifecycleStateActive),
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
	requireVbsInstanceStringPtr(t, "create compartmentId", createRequest.CreateVbsInstanceDetails.CompartmentId, resource.Spec.CompartmentId)
	requireVbsInstanceStringPtr(t, "create name", createRequest.CreateVbsInstanceDetails.Name, resource.Spec.Name)
	requireVbsInstanceStringPtr(t, "create displayName", createRequest.CreateVbsInstanceDetails.DisplayName, resource.Spec.DisplayName)
	requireVbsInstanceBoolPtr(
		t,
		"create isResourceUsageAgreementGranted",
		createRequest.CreateVbsInstanceDetails.IsResourceUsageAgreementGranted,
		true,
	)
	requireVbsInstanceStringPtr(
		t,
		"create resourceCompartmentId",
		createRequest.CreateVbsInstanceDetails.ResourceCompartmentId,
		resource.Spec.ResourceCompartmentId,
	)
	if getCalls != 0 {
		t.Fatalf("GetVbsInstance() calls = %d, want 0 while work request is pending", getCalls)
	}
	requireVbsInstanceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-vbsinstance" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-vbsinstance", got)
	}

	workRequests[workRequestID] = makeVbsInstanceWorkRequest(
		workRequestID,
		vbsinstsdk.OperationTypeCreateVbsInstance,
		vbsinstsdk.OperationStatusSucceeded,
		vbsinstsdk.ActionTypeCreated,
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
		t.Fatalf("GetVbsInstance() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(vbsinstsdk.LifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared", resource.Status.OsokStatus.Async.Current)
	}
}

func TestVbsInstanceServiceClientUpdatesAndResumesWorkRequest(t *testing.T) {
	t.Parallel()

	const (
		existingID    = "ocid1.vbsinstance.oc1..existing"
		workRequestID = "wr-vbsinstance-update"
	)

	currentResource := makeVbsInstanceResource()
	currentResource.Spec.IsResourceUsageAgreementGranted = true
	currentResource.Spec.ResourceCompartmentId = "ocid1.compartment.oc1..current"

	resource := newExistingVbsInstanceResource(existingID)
	resource.Spec = currentResource.Spec
	resource.Spec.DisplayName = "vbsinstance-updated"
	resource.Spec.IsResourceUsageAgreementGranted = false
	resource.Spec.ResourceCompartmentId = "ocid1.compartment.oc1..next"

	workRequests := map[string]vbsinstsdk.WorkRequest{
		workRequestID: makeVbsInstanceWorkRequest(
			workRequestID,
			vbsinstsdk.OperationTypeUpdateVbsInstance,
			vbsinstsdk.OperationStatusInProgress,
			vbsinstsdk.ActionTypeInProgress,
			existingID,
		),
	}

	var updateRequest vbsinstsdk.UpdateVbsInstanceRequest
	getCalls := 0

	client := newTestVbsInstanceClient(&fakeVbsInstanceOCIClient{
		getFn: func(_ context.Context, req vbsinstsdk.GetVbsInstanceRequest) (vbsinstsdk.GetVbsInstanceResponse, error) {
			getCalls++
			requireVbsInstanceStringPtr(t, "get vbsInstanceId", req.VbsInstanceId, existingID)
			if getCalls == 1 {
				return vbsinstsdk.GetVbsInstanceResponse{
					VbsInstance: makeSDKVbsInstance(existingID, currentResource, vbsinstsdk.LifecycleStateActive),
				}, nil
			}
			return vbsinstsdk.GetVbsInstanceResponse{
				VbsInstance: makeSDKVbsInstance(existingID, resource, vbsinstsdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req vbsinstsdk.UpdateVbsInstanceRequest) (vbsinstsdk.UpdateVbsInstanceResponse, error) {
			updateRequest = req
			return vbsinstsdk.UpdateVbsInstanceResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-update-vbsinstance"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req vbsinstsdk.GetWorkRequestRequest) (vbsinstsdk.GetWorkRequestResponse, error) {
			requireVbsInstanceStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return vbsinstsdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful pending update", response)
	}
	requireVbsInstanceStringPtr(t, "update vbsInstanceId", updateRequest.VbsInstanceId, existingID)
	requireVbsInstanceStringPtr(t, "update displayName", updateRequest.UpdateVbsInstanceDetails.DisplayName, resource.Spec.DisplayName)
	requireVbsInstanceBoolPtr(
		t,
		"update isResourceUsageAgreementGranted",
		updateRequest.UpdateVbsInstanceDetails.IsResourceUsageAgreementGranted,
		false,
	)
	requireVbsInstanceStringPtr(
		t,
		"update resourceCompartmentId",
		updateRequest.UpdateVbsInstanceDetails.ResourceCompartmentId,
		resource.Spec.ResourceCompartmentId,
	)
	requireVbsInstanceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, workRequestID, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-vbsinstance" {
		t.Fatalf("status.opcRequestId = %q, want opc-update-vbsinstance", got)
	}

	workRequests[workRequestID] = makeVbsInstanceWorkRequest(
		workRequestID,
		vbsinstsdk.OperationTypeUpdateVbsInstance,
		vbsinstsdk.OperationStatusSucceeded,
		vbsinstsdk.ActionTypeUpdated,
		existingID,
	)

	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() after work request success error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() after work request success response = %#v, want converged success", response)
	}
	if getCalls != 2 {
		t.Fatalf("GetVbsInstance() calls = %d, want 2 reads (pre-update and follow-up)", getCalls)
	}
	if resource.Status.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, resource.Spec.DisplayName)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared", resource.Status.OsokStatus.Async.Current)
	}
}

func TestVbsInstanceDeleteStartsAndResumesWorkRequest(t *testing.T) {
	t.Parallel()

	const (
		existingID    = "ocid1.vbsinstance.oc1..existing"
		workRequestID = "wr-vbsinstance-delete"
	)

	resource := newExistingVbsInstanceResource(existingID)
	workRequests := map[string]vbsinstsdk.WorkRequest{
		workRequestID: makeVbsInstanceWorkRequest(
			workRequestID,
			vbsinstsdk.OperationTypeDeleteVbsInstance,
			vbsinstsdk.OperationStatusInProgress,
			vbsinstsdk.ActionTypeInProgress,
			existingID,
		),
	}

	getCalls := 0
	deleteCalls := 0

	client := newTestVbsInstanceClient(&fakeVbsInstanceOCIClient{
		getFn: func(_ context.Context, req vbsinstsdk.GetVbsInstanceRequest) (vbsinstsdk.GetVbsInstanceResponse, error) {
			getCalls++
			requireVbsInstanceStringPtr(t, "get vbsInstanceId", req.VbsInstanceId, existingID)
			return vbsinstsdk.GetVbsInstanceResponse{}, errortest.NewServiceError(404, "NotFound", "missing after delete")
		},
		deleteFn: func(_ context.Context, req vbsinstsdk.DeleteVbsInstanceRequest) (vbsinstsdk.DeleteVbsInstanceResponse, error) {
			deleteCalls++
			requireVbsInstanceStringPtr(t, "delete vbsInstanceId", req.VbsInstanceId, existingID)
			return vbsinstsdk.DeleteVbsInstanceResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-delete-vbsinstance"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req vbsinstsdk.GetWorkRequestRequest) (vbsinstsdk.GetWorkRequestResponse, error) {
			requireVbsInstanceStringPtr(t, "workRequestId", req.WorkRequestId, workRequestID)
			return vbsinstsdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want pending work request")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteVbsInstance() calls = %d, want 1", deleteCalls)
	}
	requireVbsInstanceAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, workRequestID, shared.OSOKAsyncClassPending)

	workRequests[workRequestID] = makeVbsInstanceWorkRequest(
		workRequestID,
		vbsinstsdk.OperationTypeDeleteVbsInstance,
		vbsinstsdk.OperationStatusSucceeded,
		vbsinstsdk.ActionTypeDeleted,
		existingID,
	)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() after work request success error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() after work request success = false, want deleted")
	}
	if getCalls != 1 {
		t.Fatalf("GetVbsInstance() calls = %d, want 1 confirm read after the delete work request succeeds", getCalls)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared", resource.Status.OsokStatus.Async.Current)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp")
	}
}

func newTestVbsInstanceClient(client *fakeVbsInstanceOCIClient) VbsInstanceServiceClient {
	if client == nil {
		client = &fakeVbsInstanceOCIClient{}
	}
	return newVbsInstanceServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeVbsInstanceResource() *vbsinstv1beta1.VbsInstance {
	return &vbsinstv1beta1.VbsInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vbsinstance-sample",
			Namespace: "default",
		},
		Spec: vbsinstv1beta1.VbsInstanceSpec{
			CompartmentId:                   "ocid1.compartment.oc1..vbsinstanceexample",
			Name:                            "vbsinstance-sample",
			DisplayName:                     "vbsinstance-sample",
			IsResourceUsageAgreementGranted: true,
			ResourceCompartmentId:           "ocid1.compartment.oc1..workload",
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

func newExistingVbsInstanceResource(existingID string) *vbsinstv1beta1.VbsInstance {
	resource := makeVbsInstanceResource()
	resource.Status = vbsinstv1beta1.VbsInstanceStatus{
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(existingID),
		},
		Id: existingID,
	}
	return resource
}

func makeSDKVbsInstance(
	id string,
	resource *vbsinstv1beta1.VbsInstance,
	state vbsinstsdk.LifecycleStateEnum,
) vbsinstsdk.VbsInstance {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return vbsinstsdk.VbsInstance{
		Id:                              common.String(id),
		Name:                            common.String(resource.Spec.Name),
		DisplayName:                     common.String(resource.Spec.DisplayName),
		CompartmentId:                   common.String(resource.Spec.CompartmentId),
		IsResourceUsageAgreementGranted: common.Bool(resource.Spec.IsResourceUsageAgreementGranted),
		ResourceCompartmentId:           common.String(resource.Spec.ResourceCompartmentId),
		VbsAccessUrl:                    common.String("https://vbs.example"),
		TimeCreated:                     &now,
		TimeUpdated:                     &now,
		LifecycleState:                  state,
		LifecyleDetails:                 common.String("state " + string(state)),
		FreeformTags:                    maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:                     sdkVbsInstanceDefinedTags(resource.Spec.DefinedTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeSDKVbsInstanceSummary(
	id string,
	resource *vbsinstv1beta1.VbsInstance,
	state vbsinstsdk.LifecycleStateEnum,
) vbsinstsdk.VbsInstanceSummary {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	return vbsinstsdk.VbsInstanceSummary{
		Id:                              common.String(id),
		Name:                            common.String(resource.Spec.Name),
		DisplayName:                     common.String(resource.Spec.DisplayName),
		CompartmentId:                   common.String(resource.Spec.CompartmentId),
		IsResourceUsageAgreementGranted: common.Bool(resource.Spec.IsResourceUsageAgreementGranted),
		TimeCreated:                     &now,
		TimeUpdated:                     &now,
		LifecycleState:                  state,
		LifecycleDetails:                common.String("state " + string(state)),
		FreeformTags:                    maps.Clone(resource.Spec.FreeformTags),
		DefinedTags:                     sdkVbsInstanceDefinedTags(resource.Spec.DefinedTags),
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {
				"free-tier-retained": "true",
			},
		},
	}
}

func makeVbsInstanceWorkRequest(
	id string,
	operation vbsinstsdk.OperationTypeEnum,
	status vbsinstsdk.OperationStatusEnum,
	action vbsinstsdk.ActionTypeEnum,
	resourceID string,
) vbsinstsdk.WorkRequest {
	now := common.SDKTime{Time: time.Unix(0, 0).UTC()}
	percentComplete := float32(50)
	return vbsinstsdk.WorkRequest{
		OperationType:   operation,
		Status:          status,
		Id:              common.String(id),
		CompartmentId:   common.String("ocid1.compartment.oc1..vbsinstanceexample"),
		Resources:       []vbsinstsdk.WorkRequestResource{{EntityType: common.String("VbsInstance"), ActionType: action, Identifier: common.String(resourceID)}},
		PercentComplete: &percentComplete,
		TimeAccepted:    &now,
	}
}

func sdkVbsInstanceDefinedTags(input map[string]shared.MapValue) map[string]map[string]interface{} {
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

func assertVbsInstanceStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func requireVbsInstanceStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", name, got, want)
	}
}

func requireVbsInstanceBoolPtr(t *testing.T, name string, got *bool, want bool) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %t", name, got, want)
	}
}

func requireVbsInstanceAsyncCurrent(
	t *testing.T,
	resource *vbsinstv1beta1.VbsInstance,
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
