/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package macorder

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	mngdmacsdk "github.com/oracle/oci-go-sdk/v65/mngdmac"
	mngdmacv1beta1 "github.com/oracle/oci-service-operator/api/mngdmac/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeMacOrderOCIClient struct {
	createFn      func(context.Context, mngdmacsdk.CreateMacOrderRequest) (mngdmacsdk.CreateMacOrderResponse, error)
	getFn         func(context.Context, mngdmacsdk.GetMacOrderRequest) (mngdmacsdk.GetMacOrderResponse, error)
	listFn        func(context.Context, mngdmacsdk.ListMacOrdersRequest) (mngdmacsdk.ListMacOrdersResponse, error)
	updateFn      func(context.Context, mngdmacsdk.UpdateMacOrderRequest) (mngdmacsdk.UpdateMacOrderResponse, error)
	cancelFn      func(context.Context, mngdmacsdk.CancelMacOrderRequest) (mngdmacsdk.CancelMacOrderResponse, error)
	workRequestFn func(context.Context, mngdmacsdk.GetWorkRequestRequest) (mngdmacsdk.GetWorkRequestResponse, error)

	createRequests      []mngdmacsdk.CreateMacOrderRequest
	getRequests         []mngdmacsdk.GetMacOrderRequest
	listRequests        []mngdmacsdk.ListMacOrdersRequest
	updateRequests      []mngdmacsdk.UpdateMacOrderRequest
	cancelRequests      []mngdmacsdk.CancelMacOrderRequest
	workRequestRequests []mngdmacsdk.GetWorkRequestRequest
}

func (f *fakeMacOrderOCIClient) CreateMacOrder(ctx context.Context, request mngdmacsdk.CreateMacOrderRequest) (mngdmacsdk.CreateMacOrderResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return mngdmacsdk.CreateMacOrderResponse{}, nil
}

func (f *fakeMacOrderOCIClient) GetMacOrder(ctx context.Context, request mngdmacsdk.GetMacOrderRequest) (mngdmacsdk.GetMacOrderResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return mngdmacsdk.GetMacOrderResponse{}, nil
}

func (f *fakeMacOrderOCIClient) ListMacOrders(ctx context.Context, request mngdmacsdk.ListMacOrdersRequest) (mngdmacsdk.ListMacOrdersResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return mngdmacsdk.ListMacOrdersResponse{}, nil
}

func (f *fakeMacOrderOCIClient) UpdateMacOrder(ctx context.Context, request mngdmacsdk.UpdateMacOrderRequest) (mngdmacsdk.UpdateMacOrderResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return mngdmacsdk.UpdateMacOrderResponse{}, nil
}

func (f *fakeMacOrderOCIClient) CancelMacOrder(ctx context.Context, request mngdmacsdk.CancelMacOrderRequest) (mngdmacsdk.CancelMacOrderResponse, error) {
	f.cancelRequests = append(f.cancelRequests, request)
	if f.cancelFn != nil {
		return f.cancelFn(ctx, request)
	}
	return mngdmacsdk.CancelMacOrderResponse{}, nil
}

func (f *fakeMacOrderOCIClient) GetWorkRequest(ctx context.Context, request mngdmacsdk.GetWorkRequestRequest) (mngdmacsdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, request)
	}
	return mngdmacsdk.GetWorkRequestResponse{}, nil
}

func newTestMacOrderClient(fake *fakeMacOrderOCIClient) MacOrderServiceClient {
	return newMacOrderServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func newTestMacOrderResource() *mngdmacv1beta1.MacOrder {
	return &mngdmacv1beta1.MacOrder{
		Spec: mngdmacv1beta1.MacOrderSpec{
			CompartmentId:    "ocid1.compartment.oc1..example",
			OrderDescription: "Initial managed Mac order",
			OrderSize:        1,
			Shape:            string(mngdmacsdk.MacOrderShapeM4ProMacMini64gb2tb),
			CommitmentTerm:   string(mngdmacsdk.MacOrderCommitmentTermYears3),
			DisplayName:      "mac-order-alpha",
			IpRange:          "10.0.0.0/24",
		},
	}
}

func makeSDKMacOrder(
	id string,
	resource *mngdmacv1beta1.MacOrder,
	lifecycleState mngdmacsdk.MacOrderLifecycleStateEnum,
	orderStatus mngdmacsdk.MacOrderOrderStatusEnum,
) mngdmacsdk.MacOrder {
	return mngdmacsdk.MacOrder{
		Id:               common.String(id),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		OrderDescription: common.String(resource.Spec.OrderDescription),
		OrderSize:        common.Int(resource.Spec.OrderSize),
		IsDocusigned:     common.Bool(true),
		Shape:            mngdmacsdk.MacOrderShapeEnum(resource.Spec.Shape),
		CommitmentTerm:   mngdmacsdk.MacOrderCommitmentTermEnum(resource.Spec.CommitmentTerm),
		OrderStatus:      orderStatus,
		LifecycleState:   lifecycleState,
		DisplayName:      common.String(resource.Spec.DisplayName),
		IpRange:          common.String(resource.Spec.IpRange),
	}
}

func makeSDKMacOrderSummary(
	id string,
	resource *mngdmacv1beta1.MacOrder,
	lifecycleState mngdmacsdk.MacOrderLifecycleStateEnum,
	orderStatus mngdmacsdk.MacOrderOrderStatusEnum,
) mngdmacsdk.MacOrderSummary {
	return mngdmacsdk.MacOrderSummary{
		Id:               common.String(id),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		DisplayName:      common.String(resource.Spec.DisplayName),
		OrderDescription: common.String(resource.Spec.OrderDescription),
		OrderSize:        common.Int(resource.Spec.OrderSize),
		IsDocusigned:     common.Bool(true),
		Shape:            mngdmacsdk.MacOrderShapeEnum(resource.Spec.Shape),
		CommitmentTerm:   mngdmacsdk.MacOrderCommitmentTermEnum(resource.Spec.CommitmentTerm),
		OrderStatus:      orderStatus,
		LifecycleState:   lifecycleState,
		IpRange:          common.String(resource.Spec.IpRange),
	}
}

func makeMacOrderWorkRequest(
	id string,
	operationType mngdmacsdk.OperationTypeEnum,
	status mngdmacsdk.OperationStatusEnum,
	action mngdmacsdk.ActionTypeEnum,
	resourceID string,
) mngdmacsdk.WorkRequest {
	resources := []mngdmacsdk.WorkRequestResource{}
	if strings.TrimSpace(resourceID) != "" {
		resources = append(resources, mngdmacsdk.WorkRequestResource{
			EntityType: common.String("MacOrder"),
			ActionType: action,
			Identifier: common.String(resourceID),
		})
	}
	return mngdmacsdk.WorkRequest{
		Id:            common.String(id),
		OperationType: operationType,
		Status:        status,
		CompartmentId: common.String("ocid1.compartment.oc1..example"),
		Resources:     resources,
	}
}

func requireMacOrderStringPtr(t *testing.T, field string, value *string, want string) {
	t.Helper()
	if got := stringValue(value); got != want {
		t.Fatalf("%s = %q, want %q", field, got, want)
	}
}

func requireMacOrderAsyncCurrent(
	t *testing.T,
	resource *mngdmacv1beta1.MacOrder,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	workRequestID string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want populated tracker")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want workrequest", current.Source)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
}

func requireMacOrderTrailingCondition(
	t *testing.T,
	resource *mngdmacv1beta1.MacOrder,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatal("status.conditions = empty, want trailing condition")
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("trailing condition = %q, want %q", got, want)
	}
}

func assertMacOrderStringSliceEqual(t *testing.T, field string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
}

func TestReviewedMacOrderRuntimeSemanticsEncodesReviewedContract(t *testing.T) {
	t.Parallel()

	got := reviewedMacOrderRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedMacOrderRuntimeSemantics() = nil")
	}
	if got.FormalService != "mngdmac" {
		t.Fatalf("FormalService = %q, want mngdmac", got.FormalService)
	}
	if got.FormalSlug != "macorder" {
		t.Fatalf("FormalSlug = %q, want macorder", got.FormalSlug)
	}
	if got.Async == nil || got.Async.WorkRequest == nil {
		t.Fatalf("Async = %#v, want workrequest semantics", got.Async)
	}
	if got.Async.Strategy != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got.Async.Strategy)
	}
	if got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async.Runtime = %q, want generatedruntime", got.Async.Runtime)
	}
	if got.Async.WorkRequest.Source != "service-sdk" {
		t.Fatalf("Async.WorkRequest.Source = %q, want service-sdk", got.Async.WorkRequest.Source)
	}
	assertMacOrderStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertMacOrderStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertMacOrderStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertMacOrderStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertMacOrderStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertMacOrderStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertMacOrderStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName"})
	assertMacOrderStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"displayName", "ipRange", "orderDescription", "orderSize", "shape"})
	assertMacOrderStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"commitmentTerm", "compartmentId"})
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetMacOrder" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want GetWorkRequest -> GetMacOrder", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetMacOrder" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want GetWorkRequest -> GetMacOrder", got.UpdateFollowUp.Strategy)
	}
	if !strings.Contains(got.DeleteFollowUp.Strategy, "CancelMacOrder") {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want cancel-backed confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want omission of ChangeMacOrderCompartment", got.AuxiliaryOperations)
	}
}

func TestMacOrderServiceClientCreatesWithWorkRequestFollowUp(t *testing.T) {
	t.Parallel()

	const (
		createdID     = "ocid1.macorder.oc1..created"
		workRequestID = "wr-create-macorder"
	)

	resource := newTestMacOrderResource()
	workRequests := map[string]mngdmacsdk.WorkRequest{
		workRequestID: makeMacOrderWorkRequest(
			workRequestID,
			mngdmacsdk.OperationTypeCreateMacOrder,
			mngdmacsdk.OperationStatusInProgress,
			mngdmacsdk.ActionTypeInProgress,
			createdID,
		),
	}

	client := &fakeMacOrderOCIClient{
		listFn: func(_ context.Context, request mngdmacsdk.ListMacOrdersRequest) (mngdmacsdk.ListMacOrdersResponse, error) {
			requireMacOrderStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireMacOrderStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
			if request.Id != nil {
				t.Fatalf("list id = %v, want nil before create", request.Id)
			}
			return mngdmacsdk.ListMacOrdersResponse{}, nil
		},
		createFn: func(_ context.Context, request mngdmacsdk.CreateMacOrderRequest) (mngdmacsdk.CreateMacOrderResponse, error) {
			return mngdmacsdk.CreateMacOrderResponse{
				MacOrder:         makeSDKMacOrder(createdID, resource, mngdmacsdk.MacOrderLifecycleStateCreating, mngdmacsdk.MacOrderOrderStatusSubmitted),
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-create-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request mngdmacsdk.GetWorkRequestRequest) (mngdmacsdk.GetWorkRequestResponse, error) {
			requireMacOrderStringPtr(t, "workRequestId", request.WorkRequestId, workRequestID)
			return mngdmacsdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
		getFn: func(_ context.Context, request mngdmacsdk.GetMacOrderRequest) (mngdmacsdk.GetMacOrderResponse, error) {
			requireMacOrderStringPtr(t, "get macOrderId", request.MacOrderId, createdID)
			return mngdmacsdk.GetMacOrderResponse{
				MacOrder: makeSDKMacOrder(createdID, resource, mngdmacsdk.MacOrderLifecycleStateActive, mngdmacsdk.MacOrderOrderStatusProvisioning),
			}, nil
		},
	}

	serviceClient := newTestMacOrderClient(client)
	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while work request is pending", response)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("CreateMacOrder() calls = %d, want 1", len(client.createRequests))
	}
	if len(client.getRequests) != 0 {
		t.Fatalf("GetMacOrder() calls = %d, want 0 while work request is pending", len(client.getRequests))
	}
	createRequest := client.createRequests[0]
	requireMacOrderStringPtr(t, "create compartmentId", createRequest.CompartmentId, resource.Spec.CompartmentId)
	requireMacOrderStringPtr(t, "create orderDescription", createRequest.OrderDescription, resource.Spec.OrderDescription)
	requireMacOrderStringPtr(t, "create displayName", createRequest.DisplayName, resource.Spec.DisplayName)
	requireMacOrderStringPtr(t, "create ipRange", createRequest.IpRange, resource.Spec.IpRange)
	if createRequest.OrderSize == nil || *createRequest.OrderSize != resource.Spec.OrderSize {
		t.Fatalf("create orderSize = %#v, want %d", createRequest.OrderSize, resource.Spec.OrderSize)
	}
	requireMacOrderAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, workRequestID)

	workRequests[workRequestID] = makeMacOrderWorkRequest(
		workRequestID,
		mngdmacsdk.OperationTypeCreateMacOrder,
		mngdmacsdk.OperationStatusSucceeded,
		mngdmacsdk.ActionTypeCreated,
		createdID,
	)
	response, err = serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() after work request success error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() after work request success response = %#v, want converged success", response)
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("GetMacOrder() calls = %d, want 1 follow-up read", len(client.getRequests))
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(mngdmacsdk.MacOrderLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker after active follow-up", resource.Status.OsokStatus.Async.Current)
	}
}

func TestMacOrderServiceClientUpdatesWithWorkRequestFollowUp(t *testing.T) {
	t.Parallel()

	const (
		existingID     = "ocid1.macorder.oc1..existing"
		workRequestID  = "wr-update-macorder"
		updatedDetails = "Updated managed Mac order"
	)

	resource := newTestMacOrderResource()
	resource.Spec.OrderDescription = updatedDetails
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	workRequests := map[string]mngdmacsdk.WorkRequest{
		workRequestID: makeMacOrderWorkRequest(
			workRequestID,
			mngdmacsdk.OperationTypeUpdateMacOrder,
			mngdmacsdk.OperationStatusInProgress,
			mngdmacsdk.ActionTypeInProgress,
			existingID,
		),
	}
	getCalls := 0

	client := &fakeMacOrderOCIClient{
		getFn: func(_ context.Context, request mngdmacsdk.GetMacOrderRequest) (mngdmacsdk.GetMacOrderResponse, error) {
			getCalls++
			requireMacOrderStringPtr(t, "get macOrderId", request.MacOrderId, existingID)
			current := newTestMacOrderResource()
			current.Spec.OrderDescription = "Initial managed Mac order"
			if getCalls == 1 {
				return mngdmacsdk.GetMacOrderResponse{
					MacOrder: makeSDKMacOrder(existingID, current, mngdmacsdk.MacOrderLifecycleStateActive, mngdmacsdk.MacOrderOrderStatusProvisioning),
				}, nil
			}
			return mngdmacsdk.GetMacOrderResponse{
				MacOrder: makeSDKMacOrder(existingID, resource, mngdmacsdk.MacOrderLifecycleStateActive, mngdmacsdk.MacOrderOrderStatusProvisioning),
			}, nil
		},
		updateFn: func(_ context.Context, request mngdmacsdk.UpdateMacOrderRequest) (mngdmacsdk.UpdateMacOrderResponse, error) {
			return mngdmacsdk.UpdateMacOrderResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-update-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request mngdmacsdk.GetWorkRequestRequest) (mngdmacsdk.GetWorkRequestResponse, error) {
			requireMacOrderStringPtr(t, "workRequestId", request.WorkRequestId, workRequestID)
			return mngdmacsdk.GetWorkRequestResponse{WorkRequest: workRequests[workRequestID]}, nil
		},
	}

	serviceClient := newTestMacOrderClient(client)
	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while update work request is pending", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("UpdateMacOrder() calls = %d, want 1", len(client.updateRequests))
	}
	updateRequest := client.updateRequests[0]
	requireMacOrderStringPtr(t, "update macOrderId", updateRequest.MacOrderId, existingID)
	requireMacOrderStringPtr(t, "update orderDescription", updateRequest.OrderDescription, updatedDetails)
	if updateRequest.DisplayName != nil {
		t.Fatalf("update displayName = %v, want nil for unchanged field", updateRequest.DisplayName)
	}
	requireMacOrderAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending, workRequestID)

	workRequests[workRequestID] = makeMacOrderWorkRequest(
		workRequestID,
		mngdmacsdk.OperationTypeUpdateMacOrder,
		mngdmacsdk.OperationStatusSucceeded,
		mngdmacsdk.ActionTypeUpdated,
		existingID,
	)
	response, err = serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() after work request success error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() after work request success response = %#v, want converged success", response)
	}
	if getCalls != 2 {
		t.Fatalf("GetMacOrder() calls = %d, want 2 (pre-update read + follow-up read)", getCalls)
	}
	if got := resource.Status.OrderDescription; got != updatedDetails {
		t.Fatalf("status.orderDescription = %q, want %q", got, updatedDetails)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker after update follow-up", resource.Status.OsokStatus.Async.Current)
	}
}

func TestMacOrderDeleteMapsToCancelAndCompletesWhenOrderIsCanceled(t *testing.T) {
	t.Parallel()

	const (
		existingID    = "ocid1.macorder.oc1..delete"
		workRequestID = "wr-delete-macorder"
	)

	resource := newTestMacOrderResource()
	resource.Status.Id = existingID
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.DisplayName = resource.Spec.DisplayName
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	getCalls := 0
	client := &fakeMacOrderOCIClient{
		cancelFn: func(_ context.Context, request mngdmacsdk.CancelMacOrderRequest) (mngdmacsdk.CancelMacOrderResponse, error) {
			return mngdmacsdk.CancelMacOrderResponse{
				OpcWorkRequestId: common.String(workRequestID),
				OpcRequestId:     common.String("opc-delete-1"),
			}, nil
		},
		workRequestFn: func(_ context.Context, request mngdmacsdk.GetWorkRequestRequest) (mngdmacsdk.GetWorkRequestResponse, error) {
			requireMacOrderStringPtr(t, "workRequestId", request.WorkRequestId, workRequestID)
			return mngdmacsdk.GetWorkRequestResponse{
				WorkRequest: makeMacOrderWorkRequest(
					workRequestID,
					mngdmacsdk.OperationTypeCancelMacOrder,
					mngdmacsdk.OperationStatusSucceeded,
					mngdmacsdk.ActionTypeDeleted,
					existingID,
				),
			}, nil
		},
		getFn: func(_ context.Context, request mngdmacsdk.GetMacOrderRequest) (mngdmacsdk.GetMacOrderResponse, error) {
			getCalls++
			requireMacOrderStringPtr(t, "get macOrderId", request.MacOrderId, existingID)
			if getCalls == 1 {
				return mngdmacsdk.GetMacOrderResponse{
					MacOrder: makeSDKMacOrder(existingID, resource, mngdmacsdk.MacOrderLifecycleStateActive, mngdmacsdk.MacOrderOrderStatusProvisioning),
				}, nil
			}

			body := makeSDKMacOrder(existingID, resource, mngdmacsdk.MacOrderLifecycleStateActive, mngdmacsdk.MacOrderOrderStatusCanceled)
			return mngdmacsdk.GetMacOrderResponse{MacOrder: body}, nil
		},
	}

	deleted, err := newTestMacOrderClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true after cancel-backed delete confirmation")
	}
	if len(client.cancelRequests) != 1 {
		t.Fatalf("CancelMacOrder() calls = %d, want 1", len(client.cancelRequests))
	}
	cancelRequest := client.cancelRequests[0]
	requireMacOrderStringPtr(t, "cancel macOrderId", cancelRequest.MacOrderId, existingID)
	if cancelRequest.CancelReason != nil {
		t.Fatalf("cancel reason = %v, want nil because delete does not invent a cancel reason", cancelRequest.CancelReason)
	}
	if getCalls != 2 {
		t.Fatalf("GetMacOrder() calls = %d, want 2 (already-pending check + post-workrequest confirm)", getCalls)
	}
	if got := resource.Status.OrderStatus; got != string(mngdmacsdk.MacOrderOrderStatusCanceled) {
		t.Fatalf("status.orderStatus = %q, want CANCELED", got)
	}
	if got := resource.Status.LifecycleState; got != string(mngdmacsdk.MacOrderLifecycleStateDeleted) {
		t.Fatalf("status.lifecycleState = %q, want synthesized DELETED confirmation", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker after delete", resource.Status.OsokStatus.Async.Current)
	}
}

func TestMacOrderDeleteResumesPendingWorkRequestAcrossReconciles(t *testing.T) {
	t.Parallel()

	const (
		existingID    = "ocid1.macorder.oc1..delete-pending"
		workRequestID = "wr-delete-pending"
	)

	resource := newTestMacOrderResource()
	resource.Status.Id = existingID
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.DisplayName = resource.Spec.DisplayName
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	client := &fakeMacOrderOCIClient{
		cancelFn: func(context.Context, mngdmacsdk.CancelMacOrderRequest) (mngdmacsdk.CancelMacOrderResponse, error) {
			t.Fatal("CancelMacOrder() should not be called while a delete work request is already pending")
			return mngdmacsdk.CancelMacOrderResponse{}, nil
		},
		getFn: func(context.Context, mngdmacsdk.GetMacOrderRequest) (mngdmacsdk.GetMacOrderResponse, error) {
			t.Fatal("GetMacOrder() should not be called while a delete work request is still pending")
			return mngdmacsdk.GetMacOrderResponse{}, nil
		},
		workRequestFn: func(_ context.Context, request mngdmacsdk.GetWorkRequestRequest) (mngdmacsdk.GetWorkRequestResponse, error) {
			requireMacOrderStringPtr(t, "workRequestId", request.WorkRequestId, workRequestID)
			return mngdmacsdk.GetWorkRequestResponse{
				WorkRequest: makeMacOrderWorkRequest(
					workRequestID,
					mngdmacsdk.OperationTypeCancelMacOrder,
					mngdmacsdk.OperationStatusInProgress,
					mngdmacsdk.ActionTypeInProgress,
					existingID,
				),
			}, nil
		},
	}

	deleted, err := newTestMacOrderClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false while delete work request is pending")
	}
	if len(client.cancelRequests) != 0 {
		t.Fatalf("CancelMacOrder() calls = %d, want 0", len(client.cancelRequests))
	}
	if len(client.getRequests) != 0 {
		t.Fatalf("GetMacOrder() calls = %d, want 0", len(client.getRequests))
	}
	if len(client.workRequestRequests) != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", len(client.workRequestRequests))
	}
	requireMacOrderAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, workRequestID)
}

func TestMacOrderDeleteWaitsAfterSucceededWorkRequestUntilCanceledReadback(t *testing.T) {
	t.Parallel()

	const (
		existingID    = "ocid1.macorder.oc1..delete-readback"
		workRequestID = "wr-delete-readback"
	)

	resource := newTestMacOrderResource()
	resource.Status.Id = existingID
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.DisplayName = resource.Spec.DisplayName
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	getCalls := 0
	client := &fakeMacOrderOCIClient{
		cancelFn: func(context.Context, mngdmacsdk.CancelMacOrderRequest) (mngdmacsdk.CancelMacOrderResponse, error) {
			t.Fatal("CancelMacOrder() should not be reissued after the delete work request is already recorded")
			return mngdmacsdk.CancelMacOrderResponse{}, nil
		},
		workRequestFn: func(_ context.Context, request mngdmacsdk.GetWorkRequestRequest) (mngdmacsdk.GetWorkRequestResponse, error) {
			requireMacOrderStringPtr(t, "workRequestId", request.WorkRequestId, workRequestID)
			return mngdmacsdk.GetWorkRequestResponse{
				WorkRequest: makeMacOrderWorkRequest(
					workRequestID,
					mngdmacsdk.OperationTypeCancelMacOrder,
					mngdmacsdk.OperationStatusSucceeded,
					mngdmacsdk.ActionTypeDeleted,
					existingID,
				),
			}, nil
		},
		getFn: func(_ context.Context, request mngdmacsdk.GetMacOrderRequest) (mngdmacsdk.GetMacOrderResponse, error) {
			getCalls++
			requireMacOrderStringPtr(t, "get macOrderId", request.MacOrderId, existingID)
			if getCalls == 1 {
				return mngdmacsdk.GetMacOrderResponse{
					MacOrder: makeSDKMacOrder(existingID, resource, mngdmacsdk.MacOrderLifecycleStateActive, mngdmacsdk.MacOrderOrderStatusProvisioning),
				}, nil
			}
			return mngdmacsdk.GetMacOrderResponse{
				MacOrder: makeSDKMacOrder(existingID, resource, mngdmacsdk.MacOrderLifecycleStateActive, mngdmacsdk.MacOrderOrderStatusCanceled),
			}, nil
		},
	}

	serviceClient := newTestMacOrderClient(client)
	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v, want stale readback polling instead", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false while the canceled readback has not been observed yet")
	}
	if len(client.cancelRequests) != 0 {
		t.Fatalf("CancelMacOrder() calls = %d, want 0", len(client.cancelRequests))
	}
	if getCalls != 1 {
		t.Fatalf("GetMacOrder() calls = %d, want 1 stale readback confirm", getCalls)
	}
	requireMacOrderAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassSucceeded, workRequestID)

	deleted, err = serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() after canceled readback error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true after canceled readback confirmation")
	}
	if getCalls != 2 {
		t.Fatalf("GetMacOrder() calls = %d, want 2 after replaying the succeeded work request", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp after canceled readback")
	}
}

func TestMacOrderCreateDoesNotReuseCanceledListMatch(t *testing.T) {
	t.Parallel()

	const workRequestID = "wr-create-canceled-filtered"

	resource := newTestMacOrderResource()
	client := &fakeMacOrderOCIClient{
		listFn: func(_ context.Context, request mngdmacsdk.ListMacOrdersRequest) (mngdmacsdk.ListMacOrdersResponse, error) {
			requireMacOrderStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireMacOrderStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
			return mngdmacsdk.ListMacOrdersResponse{
				MacOrderCollection: mngdmacsdk.MacOrderCollection{
					Items: []mngdmacsdk.MacOrderSummary{
						makeSDKMacOrderSummary(
							"ocid1.macorder.oc1..canceled",
							resource,
							mngdmacsdk.MacOrderLifecycleStateActive,
							mngdmacsdk.MacOrderOrderStatusCanceled,
						),
					},
				},
			}, nil
		},
		createFn: func(_ context.Context, request mngdmacsdk.CreateMacOrderRequest) (mngdmacsdk.CreateMacOrderResponse, error) {
			return mngdmacsdk.CreateMacOrderResponse{
				MacOrder:         makeSDKMacOrder("ocid1.macorder.oc1..created", resource, mngdmacsdk.MacOrderLifecycleStateCreating, mngdmacsdk.MacOrderOrderStatusSubmitted),
				OpcWorkRequestId: common.String(workRequestID),
			}, nil
		},
		workRequestFn: func(_ context.Context, request mngdmacsdk.GetWorkRequestRequest) (mngdmacsdk.GetWorkRequestResponse, error) {
			requireMacOrderStringPtr(t, "workRequestId", request.WorkRequestId, workRequestID)
			return mngdmacsdk.GetWorkRequestResponse{
				WorkRequest: makeMacOrderWorkRequest(
					workRequestID,
					mngdmacsdk.OperationTypeCreateMacOrder,
					mngdmacsdk.OperationStatusInProgress,
					mngdmacsdk.ActionTypeInProgress,
					"ocid1.macorder.oc1..created",
				),
			}, nil
		},
		getFn: func(context.Context, mngdmacsdk.GetMacOrderRequest) (mngdmacsdk.GetMacOrderResponse, error) {
			t.Fatal("GetMacOrder() should not run when the only exact-name list match is a canceled order")
			return mngdmacsdk.GetMacOrderResponse{}, nil
		},
	}

	response, err := newTestMacOrderClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want create path with pending work request", response)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("CreateMacOrder() calls = %d, want 1 when canceled list matches are filtered from reuse", len(client.createRequests))
	}
	if len(client.getRequests) != 0 {
		t.Fatalf("GetMacOrder() calls = %d, want 0", len(client.getRequests))
	}
}

func TestMacOrderCreateOrUpdateTreatsCanceledTrackedOrderAsTerminalFailure(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.macorder.oc1..canceled"

	resource := newTestMacOrderResource()
	resource.Status.Id = existingID
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.DisplayName = resource.Spec.DisplayName
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	client := &fakeMacOrderOCIClient{
		getFn: func(_ context.Context, request mngdmacsdk.GetMacOrderRequest) (mngdmacsdk.GetMacOrderResponse, error) {
			requireMacOrderStringPtr(t, "get macOrderId", request.MacOrderId, existingID)
			return mngdmacsdk.GetMacOrderResponse{
				MacOrder: makeSDKMacOrder(existingID, resource, mngdmacsdk.MacOrderLifecycleStateActive, mngdmacsdk.MacOrderOrderStatusCanceled),
			}, nil
		},
		updateFn: func(context.Context, mngdmacsdk.UpdateMacOrderRequest) (mngdmacsdk.UpdateMacOrderResponse, error) {
			t.Fatal("UpdateMacOrder() should not be called once a tracked MacOrder is observed as canceled")
			return mngdmacsdk.UpdateMacOrderResponse{}, nil
		},
	}

	response, err := newTestMacOrderClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v, want handled terminal failure without reconcile error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want terminal failed status for canceled tracked orders", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want no requeue for terminal canceled-order failure", response)
	}
	if got := resource.Status.OrderStatus; got != string(mngdmacsdk.MacOrderOrderStatusCanceled) {
		t.Fatalf("status.orderStatus = %q, want CANCELED", got)
	}
	if got := resource.Status.LifecycleState; got != string(mngdmacsdk.MacOrderLifecycleStateFailed) {
		t.Fatalf("status.lifecycleState = %q, want FAILED after canceled-order normalization", got)
	}
	requireMacOrderTrailingCondition(t, resource, shared.Failed)
}

func TestMacOrderCreateOrUpdateRejectsCompartmentMoveAsReplacementOnly(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.macorder.oc1..replacement"

	resource := newTestMacOrderResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	resource.Status.Id = existingID
	resource.Status.CompartmentId = "ocid1.compartment.oc1..old"
	resource.Status.DisplayName = resource.Spec.DisplayName
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	client := &fakeMacOrderOCIClient{
		getFn: func(_ context.Context, request mngdmacsdk.GetMacOrderRequest) (mngdmacsdk.GetMacOrderResponse, error) {
			requireMacOrderStringPtr(t, "get macOrderId", request.MacOrderId, existingID)
			current := newTestMacOrderResource()
			current.Spec.CompartmentId = "ocid1.compartment.oc1..old"
			return mngdmacsdk.GetMacOrderResponse{
				MacOrder: makeSDKMacOrder(existingID, current, mngdmacsdk.MacOrderLifecycleStateActive, mngdmacsdk.MacOrderOrderStatusProvisioning),
			}, nil
		},
		updateFn: func(context.Context, mngdmacsdk.UpdateMacOrderRequest) (mngdmacsdk.UpdateMacOrderResponse, error) {
			t.Fatal("UpdateMacOrder() should not be called when compartmentId drift requires replacement")
			return mngdmacsdk.UpdateMacOrderResponse{}, nil
		},
	}

	response, err := newTestMacOrderClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-only compartment drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful replacement-only drift failure", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateMacOrder() calls = %d, want 0", len(client.updateRequests))
	}
}
