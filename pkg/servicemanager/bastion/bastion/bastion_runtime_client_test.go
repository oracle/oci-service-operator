/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package bastion

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
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testBastionID            = "ocid1.bastion.oc1..example"
	testBastionCompartmentID = "ocid1.compartment.oc1..bastion"
	testBastionSubnetID      = "ocid1.subnet.oc1..bastion"
	testBastionVCNID         = "ocid1.vcn.oc1..bastion"
	testBastionName          = "bastion-alpha"
)

type fakeBastionOCIClient struct {
	createFn      func(context.Context, bastionsdk.CreateBastionRequest) (bastionsdk.CreateBastionResponse, error)
	getFn         func(context.Context, bastionsdk.GetBastionRequest) (bastionsdk.GetBastionResponse, error)
	listFn        func(context.Context, bastionsdk.ListBastionsRequest) (bastionsdk.ListBastionsResponse, error)
	updateFn      func(context.Context, bastionsdk.UpdateBastionRequest) (bastionsdk.UpdateBastionResponse, error)
	deleteFn      func(context.Context, bastionsdk.DeleteBastionRequest) (bastionsdk.DeleteBastionResponse, error)
	workRequestFn func(context.Context, bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error)
}

func (f *fakeBastionOCIClient) CreateBastion(ctx context.Context, req bastionsdk.CreateBastionRequest) (bastionsdk.CreateBastionResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return bastionsdk.CreateBastionResponse{}, nil
}

func (f *fakeBastionOCIClient) GetBastion(ctx context.Context, req bastionsdk.GetBastionRequest) (bastionsdk.GetBastionResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return bastionsdk.GetBastionResponse{}, nil
}

func (f *fakeBastionOCIClient) ListBastions(ctx context.Context, req bastionsdk.ListBastionsRequest) (bastionsdk.ListBastionsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return bastionsdk.ListBastionsResponse{}, nil
}

func (f *fakeBastionOCIClient) UpdateBastion(ctx context.Context, req bastionsdk.UpdateBastionRequest) (bastionsdk.UpdateBastionResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return bastionsdk.UpdateBastionResponse{}, nil
}

func (f *fakeBastionOCIClient) DeleteBastion(ctx context.Context, req bastionsdk.DeleteBastionRequest) (bastionsdk.DeleteBastionResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return bastionsdk.DeleteBastionResponse{}, nil
}

func (f *fakeBastionOCIClient) GetWorkRequest(ctx context.Context, req bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return bastionsdk.GetWorkRequestResponse{}, nil
}

type bastionCreateWorkRequestFixture struct {
	t                   *testing.T
	resource            *bastionv1beta1.Bastion
	createRequest       bastionsdk.CreateBastionRequest
	listCalls           int
	createCalls         int
	getWorkRequestCalls int
	getCalls            int
}

func newBastionCreateWorkRequestFixture(
	t *testing.T,
	resource *bastionv1beta1.Bastion,
) (*fakeBastionOCIClient, *bastionCreateWorkRequestFixture) {
	t.Helper()
	fixture := &bastionCreateWorkRequestFixture{t: t, resource: resource}
	return &fakeBastionOCIClient{
		listFn:        fixture.list,
		createFn:      fixture.create,
		workRequestFn: fixture.workRequest,
		getFn:         fixture.get,
	}, fixture
}

func (f *bastionCreateWorkRequestFixture) list(_ context.Context, req bastionsdk.ListBastionsRequest) (bastionsdk.ListBastionsResponse, error) {
	f.listCalls++
	requireBastionStringPtr(f.t, "list compartmentId", req.CompartmentId, testBastionCompartmentID)
	requireBastionStringPtr(f.t, "list name", req.Name, testBastionName)
	return bastionsdk.ListBastionsResponse{}, nil
}

func (f *bastionCreateWorkRequestFixture) create(_ context.Context, req bastionsdk.CreateBastionRequest) (bastionsdk.CreateBastionResponse, error) {
	f.createCalls++
	f.createRequest = req
	return bastionsdk.CreateBastionResponse{
		Bastion:          makeSDKBastion(testBastionID, f.resource, bastionsdk.BastionLifecycleStateCreating),
		OpcWorkRequestId: common.String("wr-create-1"),
		OpcRequestId:     common.String("opc-create-1"),
	}, nil
}

func (f *bastionCreateWorkRequestFixture) workRequest(_ context.Context, req bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error) {
	f.getWorkRequestCalls++
	requireBastionStringPtr(f.t, "workRequestId", req.WorkRequestId, "wr-create-1")
	return bastionsdk.GetWorkRequestResponse{
		WorkRequest: makeBastionWorkRequest("wr-create-1", bastionsdk.OperationTypeCreateBastion, bastionsdk.OperationStatusSucceeded, bastionsdk.ActionTypeCreated, testBastionID),
	}, nil
}

func (f *bastionCreateWorkRequestFixture) get(_ context.Context, req bastionsdk.GetBastionRequest) (bastionsdk.GetBastionResponse, error) {
	f.getCalls++
	requireBastionStringPtr(f.t, "get bastionId", req.BastionId, testBastionID)
	return bastionsdk.GetBastionResponse{
		Bastion: makeSDKBastion(testBastionID, f.resource, bastionsdk.BastionLifecycleStateActive),
	}, nil
}

func (f *bastionCreateWorkRequestFixture) requireCalls(t *testing.T) {
	t.Helper()
	if f.listCalls != 1 || f.createCalls != 1 || f.getWorkRequestCalls != 1 || f.getCalls != 1 {
		t.Fatalf("call counts list/create/getWR/get = %d/%d/%d/%d, want 1/1/1/1", f.listCalls, f.createCalls, f.getWorkRequestCalls, f.getCalls)
	}
}

func (f *bastionCreateWorkRequestFixture) requireCreateRequest(t *testing.T, resource *bastionv1beta1.Bastion) {
	t.Helper()
	requireBastionStringPtr(t, "create bastionType", f.createRequest.BastionType, resource.Spec.BastionType)
	requireBastionStringPtr(t, "create compartmentId", f.createRequest.CompartmentId, testBastionCompartmentID)
	requireBastionStringPtr(t, "create targetSubnetId", f.createRequest.TargetSubnetId, testBastionSubnetID)
	requireBastionStringPtr(t, "create name", f.createRequest.Name, testBastionName)
}

func requireBastionSuccessfulResponse(t *testing.T, operation string, response servicemanager.OSOKResponse) {
	t.Helper()
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("%s response = %#v, want successful non-requeue", operation, response)
	}
}

func requireBastionCreatedStatus(t *testing.T, resource *bastionv1beta1.Bastion) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != testBastionID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testBastionID)
	}
	if got := resource.Status.Id; got != testBastionID {
		t.Fatalf("status.id = %q, want %q", got, testBastionID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after completed create", resource.Status.OsokStatus.Async.Current)
	}
	if got := lastBastionConditionType(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func TestBastionRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := newBastionRuntimeSemantics()
	if got == nil {
		t.Fatal("newBastionRuntimeSemantics() = nil")
	}
	if got.Async == nil || got.Async.Strategy != "workrequest" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async = %#v, want generatedruntime workrequest", got.Async)
	}
	if got.Async.WorkRequest == nil || got.Async.WorkRequest.Source != "service-sdk" {
		t.Fatalf("Async.WorkRequest = %#v, want service-sdk work request contract", got.Async.WorkRequest)
	}
	assertBastionStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	assertBastionStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertBastionStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertBastionStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertBastionStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertBastionStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertBastionStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "name", "targetSubnetId", "bastionType"})
	assertBastionStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{
		"maxSessionTtlInSeconds",
		"staticJumpHostIpAddresses",
		"clientCidrBlockAllowList",
		"freeformTags",
		"definedTags",
		"securityAttributes",
	})
	assertBastionStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{
		"bastionType",
		"compartmentId",
		"targetSubnetId",
		"name",
		"phoneBookEntry",
		"dnsProxyStatus",
	})
}

func TestBastionServiceClientCreatesThroughWorkRequestAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeBastionResource()
	fake, fixture := newBastionCreateWorkRequestFixture(t, resource)
	client := newTestBastionClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireBastionSuccessfulResponse(t, "CreateOrUpdate()", response)
	fixture.requireCalls(t)
	fixture.requireCreateRequest(t, resource)
	requireBastionCreatedStatus(t, resource)
}

func TestBastionServiceClientBindsExistingFromPagedList(t *testing.T) {
	t.Parallel()

	resource := makeBastionResource()
	listCalls := 0
	createCalled := false

	client := newTestBastionClient(&fakeBastionOCIClient{
		listFn: func(_ context.Context, req bastionsdk.ListBastionsRequest) (bastionsdk.ListBastionsResponse, error) {
			listCalls++
			requireBastionStringPtr(t, "list compartmentId", req.CompartmentId, testBastionCompartmentID)
			requireBastionStringPtr(t, "list name", req.Name, testBastionName)
			switch listCalls {
			case 1:
				if req.Page != nil {
					t.Fatalf("first list page token = %q, want nil", *req.Page)
				}
				return bastionsdk.ListBastionsResponse{
					Items: []bastionsdk.BastionSummary{
						makeSDKBastionSummary("ocid1.bastion.oc1..other", resource, "other", bastionsdk.BastionLifecycleStateActive),
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case 2:
				requireBastionStringPtr(t, "second list page", req.Page, "page-2")
				return bastionsdk.ListBastionsResponse{
					Items: []bastionsdk.BastionSummary{
						makeSDKBastionSummary(testBastionID, resource, testBastionName, bastionsdk.BastionLifecycleStateActive),
					},
				}, nil
			default:
				t.Fatalf("ListBastions call %d, want at most 2", listCalls)
				return bastionsdk.ListBastionsResponse{}, nil
			}
		},
		getFn: func(_ context.Context, req bastionsdk.GetBastionRequest) (bastionsdk.GetBastionResponse, error) {
			requireBastionStringPtr(t, "get bastionId", req.BastionId, testBastionID)
			return bastionsdk.GetBastionResponse{
				Bastion: makeSDKBastion(testBastionID, resource, bastionsdk.BastionLifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, bastionsdk.CreateBastionRequest) (bastionsdk.CreateBastionResponse, error) {
			createCalled = true
			return bastionsdk.CreateBastionResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active bind", response)
	}
	if createCalled {
		t.Fatal("CreateBastion was called after existing Bastion matched by paginated list")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testBastionID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testBastionID)
	}
}

func TestBastionServiceClientNoopReconcileDoesNotUpdate(t *testing.T) {
	t.Parallel()

	resource := makeBastionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testBastionID)
	resource.Status.Id = testBastionID
	updateCalled := false

	client := newTestBastionClient(&fakeBastionOCIClient{
		getFn: func(_ context.Context, req bastionsdk.GetBastionRequest) (bastionsdk.GetBastionResponse, error) {
			requireBastionStringPtr(t, "get bastionId", req.BastionId, testBastionID)
			return bastionsdk.GetBastionResponse{
				Bastion: makeSDKBastion(testBastionID, resource, bastionsdk.BastionLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, bastionsdk.UpdateBastionRequest) (bastionsdk.UpdateBastionResponse, error) {
			updateCalled = true
			return bastionsdk.UpdateBastionResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active no-op", response)
	}
	if updateCalled {
		t.Fatal("UpdateBastion was called for matching observed state")
	}
}

func TestBastionServiceClientUpdatesMutableFieldsThroughWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeBastionResource()
	resource.Spec.MaxSessionTtlInSeconds = 3600
	resource.Status.OsokStatus.Ocid = shared.OCID(testBastionID)
	resource.Status.Id = testBastionID
	getCalls := 0
	updateCalls := 0

	client := newTestBastionClient(&fakeBastionOCIClient{
		getFn: func(_ context.Context, req bastionsdk.GetBastionRequest) (bastionsdk.GetBastionResponse, error) {
			getCalls++
			requireBastionStringPtr(t, "get bastionId", req.BastionId, testBastionID)
			current := makeSDKBastion(testBastionID, resource, bastionsdk.BastionLifecycleStateActive)
			if getCalls == 1 {
				current.MaxSessionTtlInSeconds = common.Int(1800)
			}
			return bastionsdk.GetBastionResponse{Bastion: current}, nil
		},
		updateFn: func(_ context.Context, req bastionsdk.UpdateBastionRequest) (bastionsdk.UpdateBastionResponse, error) {
			updateCalls++
			requireBastionStringPtr(t, "update bastionId", req.BastionId, testBastionID)
			requireBastionIntPtr(t, "update maxSessionTtlInSeconds", req.MaxSessionTtlInSeconds, 3600)
			if len(req.ClientCidrBlockAllowList) != 0 {
				t.Fatalf("update clientCidrBlockAllowList = %#v, want omitted when unchanged", req.ClientCidrBlockAllowList)
			}
			return bastionsdk.UpdateBastionResponse{
				OpcWorkRequestId: common.String("wr-update-1"),
				OpcRequestId:     common.String("opc-update-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error) {
			requireBastionStringPtr(t, "workRequestId", req.WorkRequestId, "wr-update-1")
			return bastionsdk.GetWorkRequestResponse{
				WorkRequest: makeBastionWorkRequest("wr-update-1", bastionsdk.OperationTypeUpdateBastion, bastionsdk.OperationStatusSucceeded, bastionsdk.ActionTypeUpdated, testBastionID),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful update", response)
	}
	if getCalls != 2 || updateCalls != 1 {
		t.Fatalf("call counts get/update = %d/%d, want 2/1", getCalls, updateCalls)
	}
	if got := resource.Status.MaxSessionTtlInSeconds; got != 3600 {
		t.Fatalf("status.maxSessionTtlInSeconds = %d, want 3600", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestBastionServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeBastionResource()
	resource.Spec.TargetSubnetId = "ocid1.subnet.oc1..different"
	resource.Status.OsokStatus.Ocid = shared.OCID(testBastionID)
	resource.Status.Id = testBastionID
	updateCalled := false

	client := newTestBastionClient(&fakeBastionOCIClient{
		getFn: func(context.Context, bastionsdk.GetBastionRequest) (bastionsdk.GetBastionResponse, error) {
			currentResource := makeBastionResource()
			return bastionsdk.GetBastionResponse{
				Bastion: makeSDKBastion(testBastionID, currentResource, bastionsdk.BastionLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, bastionsdk.UpdateBastionRequest) (bastionsdk.UpdateBastionResponse, error) {
			updateCalled = true
			return bastionsdk.UpdateBastionResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "replacement when targetSubnetId changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want targetSubnetId replacement rejection", err.Error())
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if updateCalled {
		t.Fatal("UpdateBastion was called despite create-only drift")
	}
	if got := lastBastionConditionType(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func TestBastionDeleteWaitsForWorkRequestAndConfirmedNotFound(t *testing.T) {
	t.Parallel()

	resource := makeBastionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testBastionID)
	resource.Status.Id = testBastionID
	workRequestStatus := bastionsdk.OperationStatusInProgress
	getCalls := 0
	deleteCalls := 0

	client := newTestBastionClient(&fakeBastionOCIClient{
		getFn: func(context.Context, bastionsdk.GetBastionRequest) (bastionsdk.GetBastionResponse, error) {
			getCalls++
			if getCalls == 1 {
				return bastionsdk.GetBastionResponse{
					Bastion: makeSDKBastion(testBastionID, resource, bastionsdk.BastionLifecycleStateActive),
				}, nil
			}
			return bastionsdk.GetBastionResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "bastion deleted")
		},
		deleteFn: func(_ context.Context, req bastionsdk.DeleteBastionRequest) (bastionsdk.DeleteBastionResponse, error) {
			deleteCalls++
			requireBastionStringPtr(t, "delete bastionId", req.BastionId, testBastionID)
			return bastionsdk.DeleteBastionResponse{
				OpcWorkRequestId: common.String("wr-delete-1"),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error) {
			requireBastionStringPtr(t, "workRequestId", req.WorkRequestId, "wr-delete-1")
			action := bastionsdk.ActionTypeInProgress
			if workRequestStatus == bastionsdk.OperationStatusSucceeded {
				action = bastionsdk.ActionTypeDeleted
			}
			return bastionsdk.GetWorkRequestResponse{
				WorkRequest: makeBastionWorkRequest("wr-delete-1", bastionsdk.OperationTypeDeleteBastion, workRequestStatus, action, testBastionID),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while work request is pending")
	}
	requireBastionAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete-1", shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}

	workRequestStatus = bastionsdk.OperationStatusSucceeded
	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() after work request success error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after work request success and NotFound confirmation")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteBastion calls = %d, want 1", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want deletion timestamp")
	}
}

func TestBastionDeleteSucceededWorkRequestKeepsFinalizerOnAuthShapedConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeBastionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testBastionID)
	resource.Status.Id = testBastionID
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete-1",
		RawStatus:       string(bastionsdk.OperationStatusInProgress),
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	getCalls := 0
	workRequestCalls := 0
	deleteCalled := false
	confirmErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous delete confirmation")
	confirmErr.OpcRequestID = "opc-confirm-delete-1"

	client := newTestBastionClient(&fakeBastionOCIClient{
		getFn: func(_ context.Context, req bastionsdk.GetBastionRequest) (bastionsdk.GetBastionResponse, error) {
			getCalls++
			requireBastionStringPtr(t, "get bastionId", req.BastionId, testBastionID)
			return bastionsdk.GetBastionResponse{}, confirmErr
		},
		workRequestFn: func(_ context.Context, req bastionsdk.GetWorkRequestRequest) (bastionsdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireBastionStringPtr(t, "workRequestId", req.WorkRequestId, "wr-delete-1")
			return bastionsdk.GetWorkRequestResponse{
				WorkRequest: makeBastionWorkRequest("wr-delete-1", bastionsdk.OperationTypeDeleteBastion, bastionsdk.OperationStatusSucceeded, bastionsdk.ActionTypeDeleted, testBastionID),
			}, nil
		},
		deleteFn: func(context.Context, bastionsdk.DeleteBastionRequest) (bastionsdk.DeleteBastionResponse, error) {
			deleteCalled = true
			return bastionsdk.DeleteBastionResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped confirm-read not found to remain fatal")
	}
	if !strings.Contains(err.Error(), "authorization-shaped not found") {
		t.Fatalf("Delete() error = %q, want conservative auth-shaped confirm-read context", err.Error())
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped confirm read")
	}
	if deleteCalled {
		t.Fatal("DeleteBastion was called while resuming a tracked delete work request")
	}
	if workRequestCalls != 1 || getCalls != 1 {
		t.Fatalf("call counts getWorkRequest/get = %d/%d, want 1/1", workRequestCalls, getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-confirm-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-confirm-delete-1", got)
	}
}

func TestBastionDeleteKeepsFinalizerOnAuthShapedNotFound(t *testing.T) {
	t.Parallel()

	resource := makeBastionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testBastionID)
	resource.Status.Id = testBastionID

	client := newTestBastionClient(&fakeBastionOCIClient{
		getFn: func(context.Context, bastionsdk.GetBastionRequest) (bastionsdk.GetBastionResponse, error) {
			return bastionsdk.GetBastionResponse{
				Bastion: makeSDKBastion(testBastionID, resource, bastionsdk.BastionLifecycleStateActive),
			}, nil
		},
		deleteFn: func(context.Context, bastionsdk.DeleteBastionRequest) (bastionsdk.DeleteBastionResponse, error) {
			return bastionsdk.DeleteBastionResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous bastion miss")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped not-found to remain fatal")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want auth-shaped not-found context", err.Error())
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped not-found")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
}

func newTestBastionClient(fake *fakeBastionOCIClient) BastionServiceClient {
	return newBastionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeBastionResource() *bastionv1beta1.Bastion {
	return &bastionv1beta1.Bastion{
		Spec: bastionv1beta1.BastionSpec{
			BastionType:               "standard",
			CompartmentId:             testBastionCompartmentID,
			TargetSubnetId:            testBastionSubnetID,
			Name:                      testBastionName,
			PhoneBookEntry:            "team-alpha",
			StaticJumpHostIpAddresses: []string{"10.0.0.10"},
			ClientCidrBlockAllowList:  []string{"192.0.2.0/24"},
			MaxSessionTtlInSeconds:    1800,
			DnsProxyStatus:            string(bastionsdk.BastionDnsProxyStatusDisabled),
			FreeformTags:              map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
			SecurityAttributes: map[string]shared.MapValue{
				"Oracle-ZPR": {"MaxEgressCount": "42"},
			},
		},
	}
}

func makeSDKBastion(id string, resource *bastionv1beta1.Bastion, state bastionsdk.BastionLifecycleStateEnum) bastionsdk.Bastion {
	ttl := resource.Spec.MaxSessionTtlInSeconds
	return bastionsdk.Bastion{
		BastionType:               common.String(resource.Spec.BastionType),
		Id:                        common.String(id),
		Name:                      common.String(resource.Spec.Name),
		CompartmentId:             common.String(resource.Spec.CompartmentId),
		TargetVcnId:               common.String(testBastionVCNID),
		TargetSubnetId:            common.String(resource.Spec.TargetSubnetId),
		MaxSessionTtlInSeconds:    common.Int(ttl),
		LifecycleState:            state,
		PhoneBookEntry:            common.String(resource.Spec.PhoneBookEntry),
		ClientCidrBlockAllowList:  append([]string(nil), resource.Spec.ClientCidrBlockAllowList...),
		StaticJumpHostIpAddresses: append([]string(nil), resource.Spec.StaticJumpHostIpAddresses...),
		DnsProxyStatus:            bastionsdk.BastionDnsProxyStatusEnum(resource.Spec.DnsProxyStatus),
		FreeformTags:              cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:               sdkMapValue(resource.Spec.DefinedTags),
		SecurityAttributes:        sdkMapValue(resource.Spec.SecurityAttributes),
	}
}

func makeSDKBastionSummary(
	id string,
	resource *bastionv1beta1.Bastion,
	name string,
	state bastionsdk.BastionLifecycleStateEnum,
) bastionsdk.BastionSummary {
	return bastionsdk.BastionSummary{
		BastionType:    common.String(resource.Spec.BastionType),
		Id:             common.String(id),
		Name:           common.String(name),
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		TargetVcnId:    common.String(testBastionVCNID),
		TargetSubnetId: common.String(resource.Spec.TargetSubnetId),
		LifecycleState: state,
		DnsProxyStatus: bastionsdk.BastionDnsProxyStatusEnum(resource.Spec.DnsProxyStatus),
		FreeformTags:   cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:    sdkMapValue(resource.Spec.DefinedTags),
	}
}

func makeBastionWorkRequest(
	id string,
	operationType bastionsdk.OperationTypeEnum,
	status bastionsdk.OperationStatusEnum,
	action bastionsdk.ActionTypeEnum,
	bastionID string,
) bastionsdk.WorkRequest {
	percentComplete := float32(100)
	workRequest := bastionsdk.WorkRequest{
		OperationType:   operationType,
		Status:          status,
		Id:              common.String(id),
		CompartmentId:   common.String(testBastionCompartmentID),
		PercentComplete: &percentComplete,
	}
	if bastionID != "" {
		workRequest.Resources = []bastionsdk.WorkRequestResource{
			{
				EntityType: common.String("bastion"),
				ActionType: action,
				Identifier: common.String(bastionID),
				EntityUri:  common.String("/20210331/bastions/" + bastionID),
			},
		}
	}
	return workRequest
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func sdkMapValue(input map[string]shared.MapValue) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	output := make(map[string]map[string]interface{}, len(input))
	for namespace, values := range input {
		output[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			output[namespace][key] = value
		}
	}
	return output
}

func requireBastionStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireBastionIntPtr(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %d", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %d, want %d", name, *got, want)
	}
}

func requireBastionAsyncCurrent(
	t *testing.T,
	resource *bastionv1beta1.Bastion,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want current async operation")
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

func lastBastionConditionType(resource *bastionv1beta1.Bastion) shared.OSOKConditionType {
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		return ""
	}
	return shared.OSOKConditionType(conditions[len(conditions)-1].Type)
}

func assertBastionStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("%s = %#v, want %#v", name, got, want)
		}
	}
}
