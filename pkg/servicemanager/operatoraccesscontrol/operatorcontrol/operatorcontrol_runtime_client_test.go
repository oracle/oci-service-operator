/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package operatorcontrol

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	operatoraccesscontrolsdk "github.com/oracle/oci-go-sdk/v65/operatoraccesscontrol"
	operatoraccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/operatoraccesscontrol/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testOperatorControlID            = "ocid1.operatorcontrol.oc1..example"
	testOperatorControlOtherID       = "ocid1.operatorcontrol.oc1..other"
	testOperatorControlCompartmentID = "ocid1.compartment.oc1..operatorcontrol"
)

type fakeOperatorControlOCIClient struct {
	create func(context.Context, operatoraccesscontrolsdk.CreateOperatorControlRequest) (operatoraccesscontrolsdk.CreateOperatorControlResponse, error)
	get    func(context.Context, operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error)
	list   func(context.Context, operatoraccesscontrolsdk.ListOperatorControlsRequest) (operatoraccesscontrolsdk.ListOperatorControlsResponse, error)
	update func(context.Context, operatoraccesscontrolsdk.UpdateOperatorControlRequest) (operatoraccesscontrolsdk.UpdateOperatorControlResponse, error)
	delete func(context.Context, operatoraccesscontrolsdk.DeleteOperatorControlRequest) (operatoraccesscontrolsdk.DeleteOperatorControlResponse, error)

	createRequests []operatoraccesscontrolsdk.CreateOperatorControlRequest
	getRequests    []operatoraccesscontrolsdk.GetOperatorControlRequest
	listRequests   []operatoraccesscontrolsdk.ListOperatorControlsRequest
	updateRequests []operatoraccesscontrolsdk.UpdateOperatorControlRequest
	deleteRequests []operatoraccesscontrolsdk.DeleteOperatorControlRequest
}

func (f *fakeOperatorControlOCIClient) CreateOperatorControl(
	ctx context.Context,
	request operatoraccesscontrolsdk.CreateOperatorControlRequest,
) (operatoraccesscontrolsdk.CreateOperatorControlResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.create == nil {
		return operatoraccesscontrolsdk.CreateOperatorControlResponse{}, nil
	}
	return f.create(ctx, request)
}

func (f *fakeOperatorControlOCIClient) GetOperatorControl(
	ctx context.Context,
	request operatoraccesscontrolsdk.GetOperatorControlRequest,
) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.get == nil {
		return operatoraccesscontrolsdk.GetOperatorControlResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
	}
	return f.get(ctx, request)
}

func (f *fakeOperatorControlOCIClient) ListOperatorControls(
	ctx context.Context,
	request operatoraccesscontrolsdk.ListOperatorControlsRequest,
) (operatoraccesscontrolsdk.ListOperatorControlsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.list == nil {
		return operatoraccesscontrolsdk.ListOperatorControlsResponse{}, nil
	}
	return f.list(ctx, request)
}

func (f *fakeOperatorControlOCIClient) UpdateOperatorControl(
	ctx context.Context,
	request operatoraccesscontrolsdk.UpdateOperatorControlRequest,
) (operatoraccesscontrolsdk.UpdateOperatorControlResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.update == nil {
		return operatoraccesscontrolsdk.UpdateOperatorControlResponse{}, nil
	}
	return f.update(ctx, request)
}

func (f *fakeOperatorControlOCIClient) DeleteOperatorControl(
	ctx context.Context,
	request operatoraccesscontrolsdk.DeleteOperatorControlRequest,
) (operatoraccesscontrolsdk.DeleteOperatorControlResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete == nil {
		return operatoraccesscontrolsdk.DeleteOperatorControlResponse{}, nil
	}
	return f.delete(ctx, request)
}

func TestOperatorControlRuntimeSemantics(t *testing.T) {
	semantics := newOperatorControlRuntimeSemantics()
	if semantics == nil {
		t.Fatal("newOperatorControlRuntimeSemantics() = nil")
	}
	if semantics.Async == nil || semantics.Async.Strategy != "lifecycle" || semantics.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", semantics.Async)
	}
	if semantics.FinalizerPolicy != "retain-until-confirmed-delete" || semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete policy = %q follow-up = %q, want retained confirm-delete", semantics.FinalizerPolicy, semantics.DeleteFollowUp.Strategy)
	}
	assertStringSliceContains(t, "active states", semantics.Lifecycle.ActiveStates, "CREATED")
	assertStringSliceContains(t, "active states", semantics.Lifecycle.ActiveStates, "ASSIGNED")
	assertStringSliceContains(t, "terminal states", semantics.Delete.TerminalStates, "DELETED")
	assertStringSliceContains(t, "mutable fields", semantics.Mutation.Mutable, "isFullyPreApproved")
	assertStringSliceContains(t, "force-new fields", semantics.Mutation.ForceNew, "resourceType")
}

func TestOperatorControlCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	resource := newTestOperatorControl()
	resource.Spec.IsFullyPreApproved = false
	client := &fakeOperatorControlOCIClient{}
	client.list = func(_ context.Context, request operatoraccesscontrolsdk.ListOperatorControlsRequest) (operatoraccesscontrolsdk.ListOperatorControlsResponse, error) {
		assertOperatorControlListRequest(t, request, resource.Spec)
		return operatoraccesscontrolsdk.ListOperatorControlsResponse{}, nil
	}
	client.create = func(_ context.Context, request operatoraccesscontrolsdk.CreateOperatorControlRequest) (operatoraccesscontrolsdk.CreateOperatorControlResponse, error) {
		assertOperatorControlCreateRequest(t, request, resource.Spec)
		return operatoraccesscontrolsdk.CreateOperatorControlResponse{
			OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, resource.Spec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated),
			OpcRequestId:    common.String("opc-create"),
		}, nil
	}
	client.get = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
		assertStringPtr(t, "get operatorControlId", request.OperatorControlId, testOperatorControlID)
		return operatoraccesscontrolsdk.GetOperatorControlResponse{
			OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, resource.Spec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated),
		}, nil
	}

	response, err := newTestOperatorControlServiceClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful without requeue", response)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("CreateOperatorControl() calls = %d, want 1", len(client.createRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testOperatorControlID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testOperatorControlID)
	}
	if resource.Status.Id != testOperatorControlID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testOperatorControlID)
	}
	if resource.Status.LifecycleState != "CREATED" {
		t.Fatalf("status.lifecycleState = %q, want CREATED", resource.Status.LifecycleState)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
}

func TestOperatorControlCreateOrUpdateCanonicalizesResourceTypeBeforeCreateLookup(t *testing.T) {
	resource := newTestOperatorControl()
	resource.Spec.ResourceType = strings.ToLower(resource.Spec.ResourceType)
	wantResourceType := string(operatoraccesscontrolsdk.ResourceTypesExacc)
	client := &fakeOperatorControlOCIClient{}
	client.list = func(_ context.Context, request operatoraccesscontrolsdk.ListOperatorControlsRequest) (operatoraccesscontrolsdk.ListOperatorControlsResponse, error) {
		assertStringPtr(t, "list resourceType", request.ResourceType, wantResourceType)
		return operatoraccesscontrolsdk.ListOperatorControlsResponse{}, nil
	}
	client.create = func(_ context.Context, request operatoraccesscontrolsdk.CreateOperatorControlRequest) (operatoraccesscontrolsdk.CreateOperatorControlResponse, error) {
		if got := string(request.ResourceType); got != wantResourceType {
			t.Fatalf("create resourceType = %q, want %q", got, wantResourceType)
		}
		return operatoraccesscontrolsdk.CreateOperatorControlResponse{
			OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, resource.Spec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated),
		}, nil
	}
	client.get = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
		assertStringPtr(t, "get operatorControlId", request.OperatorControlId, testOperatorControlID)
		return operatoraccesscontrolsdk.GetOperatorControlResponse{
			OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, resource.Spec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated),
		}, nil
	}

	response, err := newTestOperatorControlServiceClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful create", response)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("CreateOperatorControl() calls = %d, want 1", len(client.createRequests))
	}
	if got := resource.Spec.ResourceType; got != wantResourceType {
		t.Fatalf("spec.resourceType after reconcile = %q, want %q", got, wantResourceType)
	}
}

func TestOperatorControlCreateOrUpdateBindsExistingFromSecondListPage(t *testing.T) {
	resource := newTestOperatorControl()
	otherSpec := resource.Spec
	otherSpec.OperatorControlName = "other-operator-control"
	client := &fakeOperatorControlOCIClient{}
	client.list = func(_ context.Context, request operatoraccesscontrolsdk.ListOperatorControlsRequest) (operatoraccesscontrolsdk.ListOperatorControlsResponse, error) {
		assertOperatorControlListRequest(t, request, resource.Spec)
		if request.Page == nil {
			return operatoraccesscontrolsdk.ListOperatorControlsResponse{
				OperatorControlCollection: operatoraccesscontrolsdk.OperatorControlCollection{
					Items: []operatoraccesscontrolsdk.OperatorControlSummary{
						sdkOperatorControlSummary(testOperatorControlOtherID, otherSpec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		}
		assertStringPtr(t, "second list page", request.Page, "page-2")
		return operatoraccesscontrolsdk.ListOperatorControlsResponse{
			OperatorControlCollection: operatoraccesscontrolsdk.OperatorControlCollection{
				Items: []operatoraccesscontrolsdk.OperatorControlSummary{
					sdkOperatorControlSummary(testOperatorControlID, resource.Spec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesUnassigned),
				},
			},
		}, nil
	}
	client.get = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
		assertStringPtr(t, "get operatorControlId", request.OperatorControlId, testOperatorControlID)
		return operatoraccesscontrolsdk.GetOperatorControlResponse{
			OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, resource.Spec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesUnassigned),
		}, nil
	}

	response, err := newTestOperatorControlServiceClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful bind without requeue", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateOperatorControl() calls = %d, want 0", len(client.createRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("ListOperatorControls() calls = %d, want 2 pages", len(client.listRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testOperatorControlID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testOperatorControlID)
	}
}

func TestOperatorControlCreateOrUpdateCanonicalizesResourceTypeBeforeBindLookup(t *testing.T) {
	resource := newTestOperatorControl()
	resource.Spec.ResourceType = strings.ToLower(resource.Spec.ResourceType)
	wantResourceType := string(operatoraccesscontrolsdk.ResourceTypesExacc)
	canonicalSpec := resource.Spec
	canonicalSpec.ResourceType = wantResourceType
	client := &fakeOperatorControlOCIClient{}
	client.list = func(_ context.Context, request operatoraccesscontrolsdk.ListOperatorControlsRequest) (operatoraccesscontrolsdk.ListOperatorControlsResponse, error) {
		assertStringPtr(t, "list resourceType", request.ResourceType, wantResourceType)
		return operatoraccesscontrolsdk.ListOperatorControlsResponse{
			OperatorControlCollection: operatoraccesscontrolsdk.OperatorControlCollection{
				Items: []operatoraccesscontrolsdk.OperatorControlSummary{
					sdkOperatorControlSummary(testOperatorControlID, canonicalSpec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesAssigned),
				},
			},
		}, nil
	}
	client.get = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
		assertStringPtr(t, "get operatorControlId", request.OperatorControlId, testOperatorControlID)
		return operatoraccesscontrolsdk.GetOperatorControlResponse{
			OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, canonicalSpec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesAssigned),
		}, nil
	}

	response, err := newTestOperatorControlServiceClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful bind", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateOperatorControl() calls = %d, want 0", len(client.createRequests))
	}
	if got := resource.Spec.ResourceType; got != wantResourceType {
		t.Fatalf("spec.resourceType after bind = %q, want %q", got, wantResourceType)
	}
}

func TestOperatorControlCreateOrUpdateNoopsWithoutMutableDrift(t *testing.T) {
	resource := newTestOperatorControl()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOperatorControlID)
	resource.Status.Id = testOperatorControlID
	client := &fakeOperatorControlOCIClient{}
	client.get = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
		assertStringPtr(t, "get operatorControlId", request.OperatorControlId, testOperatorControlID)
		return operatoraccesscontrolsdk.GetOperatorControlResponse{
			OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, resource.Spec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesAssigned),
		}, nil
	}

	response, err := newTestOperatorControlServiceClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful no-op", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateOperatorControl() calls = %d, want 0", len(client.updateRequests))
	}
}

func TestOperatorControlCreateOrUpdateCanonicalizesResourceTypeBeforeNoopDriftCheck(t *testing.T) {
	resource := newTestOperatorControl()
	resource.Spec.ResourceType = strings.ToLower(resource.Spec.ResourceType)
	wantResourceType := string(operatoraccesscontrolsdk.ResourceTypesExacc)
	resource.Status.OsokStatus.Ocid = shared.OCID(testOperatorControlID)
	resource.Status.Id = testOperatorControlID
	canonicalSpec := resource.Spec
	canonicalSpec.ResourceType = wantResourceType
	client := &fakeOperatorControlOCIClient{}
	client.get = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
		assertStringPtr(t, "get operatorControlId", request.OperatorControlId, testOperatorControlID)
		return operatoraccesscontrolsdk.GetOperatorControlResponse{
			OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, canonicalSpec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesAssigned),
		}, nil
	}

	response, err := newTestOperatorControlServiceClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful no-op", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateOperatorControl() calls = %d, want 0", len(client.updateRequests))
	}
	if got := resource.Spec.ResourceType; got != wantResourceType {
		t.Fatalf("spec.resourceType after no-op = %q, want %q", got, wantResourceType)
	}
}

func TestOperatorControlCreateOrUpdateSendsMutableUpdate(t *testing.T) {
	resource := newTestOperatorControl()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOperatorControlID)
	resource.Status.Id = testOperatorControlID
	resource.Spec.OperatorControlName = "operator-control-new"
	resource.Spec.ApproverGroupsList = []string{"group-a", "group-b"}
	resource.Spec.IsFullyPreApproved = true
	resource.Spec.Description = "updated description"
	resource.Spec.ApproversList = []string{"user-a"}
	resource.Spec.PreApprovedOpActionList = []string{"REBOOT"}
	resource.Spec.EmailIdList = []string{"approver@example.com"}
	resource.Spec.NumberOfApprovers = 2
	resource.Spec.SystemMessage = "updated message"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}

	currentSpec := resource.Spec
	currentSpec.OperatorControlName = "operator-control-old"
	currentSpec.ApproverGroupsList = []string{"group-a"}
	currentSpec.IsFullyPreApproved = false
	currentSpec.Description = "old description"
	currentSpec.ApproversList = []string{"user-old"}
	currentSpec.PreApprovedOpActionList = []string{"PATCH"}
	currentSpec.EmailIdList = []string{"old@example.com"}
	currentSpec.NumberOfApprovers = 1
	currentSpec.SystemMessage = "old message"
	currentSpec.FreeformTags = map[string]string{"env": "dev"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}

	client := &fakeOperatorControlOCIClient{}
	getCalls := 0
	client.get = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
		assertStringPtr(t, "get operatorControlId", request.OperatorControlId, testOperatorControlID)
		getCalls++
		if getCalls == 1 {
			return operatoraccesscontrolsdk.GetOperatorControlResponse{
				OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, currentSpec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated),
			}, nil
		}
		return operatoraccesscontrolsdk.GetOperatorControlResponse{
			OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, resource.Spec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated),
		}, nil
	}
	client.update = func(_ context.Context, request operatoraccesscontrolsdk.UpdateOperatorControlRequest) (operatoraccesscontrolsdk.UpdateOperatorControlResponse, error) {
		assertStringPtr(t, "update operatorControlId", request.OperatorControlId, testOperatorControlID)
		assertOperatorControlUpdateRequest(t, request, resource.Spec)
		return operatoraccesscontrolsdk.UpdateOperatorControlResponse{
			OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, resource.Spec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated),
			OpcRequestId:    common.String("opc-update"),
		}, nil
	}

	response, err := newTestOperatorControlServiceClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful update", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("UpdateOperatorControl() calls = %d, want 1", len(client.updateRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
}

func TestOperatorControlCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := newTestOperatorControl()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOperatorControlID)
	resource.Status.Id = testOperatorControlID
	oldSpec := resource.Spec
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	resource.Spec.ResourceType = string(operatoraccesscontrolsdk.ResourceTypesAutonomousvmcluster)

	client := &fakeOperatorControlOCIClient{}
	client.get = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
		assertStringPtr(t, "get operatorControlId", request.OperatorControlId, testOperatorControlID)
		return operatoraccesscontrolsdk.GetOperatorControlResponse{
			OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, oldSpec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated),
		}, nil
	}

	response, err := newTestOperatorControlServiceClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want unsuccessful", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateOperatorControl() calls = %d, want 0", len(client.updateRequests))
	}
	if !strings.Contains(err.Error(), "compartmentId") && !strings.Contains(err.Error(), "resourceType") {
		t.Fatalf("CreateOrUpdate() error = %q, want create-only drift field", err.Error())
	}
}

func TestOperatorControlDeleteWaitsForConfirmedDeletion(t *testing.T) {
	resource := newTestOperatorControl()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOperatorControlID)
	resource.Status.Id = testOperatorControlID
	client := &fakeOperatorControlOCIClient{}
	getCalls := 0
	client.get = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
		assertStringPtr(t, "get operatorControlId", request.OperatorControlId, testOperatorControlID)
		getCalls++
		state := operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated
		if getCalls == 3 {
			state = operatoraccesscontrolsdk.OperatorControlLifecycleStatesDeleted
		}
		return operatoraccesscontrolsdk.GetOperatorControlResponse{
			OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, resource.Spec, state),
		}, nil
	}
	client.delete = func(_ context.Context, request operatoraccesscontrolsdk.DeleteOperatorControlRequest) (operatoraccesscontrolsdk.DeleteOperatorControlResponse, error) {
		assertStringPtr(t, "delete operatorControlId", request.OperatorControlId, testOperatorControlID)
		if request.Description != nil {
			t.Fatalf("delete description = %q, want nil", *request.Description)
		}
		return operatoraccesscontrolsdk.DeleteOperatorControlResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := newTestOperatorControlServiceClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after terminal readback")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("DeleteOperatorControl() calls = %d, want 1", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want set after confirmed delete")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
}

func TestOperatorControlDeleteRetainsFinalizerWhileReadbackStillActive(t *testing.T) {
	resource := newTestOperatorControl()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOperatorControlID)
	resource.Status.Id = testOperatorControlID
	client := &fakeOperatorControlOCIClient{}
	client.get = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
		assertStringPtr(t, "get operatorControlId", request.OperatorControlId, testOperatorControlID)
		return operatoraccesscontrolsdk.GetOperatorControlResponse{
			OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, resource.Spec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated),
		}, nil
	}
	client.delete = func(_ context.Context, request operatoraccesscontrolsdk.DeleteOperatorControlRequest) (operatoraccesscontrolsdk.DeleteOperatorControlResponse, error) {
		assertStringPtr(t, "delete operatorControlId", request.OperatorControlId, testOperatorControlID)
		return operatoraccesscontrolsdk.DeleteOperatorControlResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := newTestOperatorControlServiceClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while readback is still active")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt is set before terminal delete confirmation")
	}
	if got := lastOperatorControlCondition(resource); got.Type != shared.Terminating || got.Status != corev1.ConditionTrue {
		t.Fatalf("last condition = %+v, want Terminating true", got)
	}
	if !operatorControlDeleteAlreadyPending(resource) {
		t.Fatalf("status.status.async.current = %#v, want pending delete marker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestOperatorControlDeleteAlreadyPendingConfirmsWithoutSecondDelete(t *testing.T) {
	resource := newTestOperatorControl()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOperatorControlID)
	resource.Status.Id = testOperatorControlID
	client := &fakeOperatorControlOCIClient{}
	getCalls := 0
	client.get = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
		assertStringPtr(t, "get operatorControlId", request.OperatorControlId, testOperatorControlID)
		getCalls++
		state := operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated
		if getCalls >= 6 {
			state = operatoraccesscontrolsdk.OperatorControlLifecycleStatesDeleted
		}
		return operatoraccesscontrolsdk.GetOperatorControlResponse{
			OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, resource.Spec, state),
		}, nil
	}
	client.delete = func(_ context.Context, request operatoraccesscontrolsdk.DeleteOperatorControlRequest) (operatoraccesscontrolsdk.DeleteOperatorControlResponse, error) {
		assertStringPtr(t, "delete operatorControlId", request.OperatorControlId, testOperatorControlID)
		return operatoraccesscontrolsdk.DeleteOperatorControlResponse{OpcRequestId: common.String("opc-delete")}, nil
	}
	serviceClient := newTestOperatorControlServiceClient(client)

	assertOperatorControlDeleteStartsPending(t, serviceClient, client, resource)
	assertOperatorControlPendingDeleteRetriesWithoutSecondDelete(t, serviceClient, client, resource)
	assertOperatorControlPendingDeleteConfirms(t, serviceClient, client, resource)
}

func assertOperatorControlDeleteStartsPending(
	t *testing.T,
	serviceClient OperatorControlServiceClient,
	client *fakeOperatorControlOCIClient,
	resource *operatoraccesscontrolv1beta1.OperatorControl,
) {
	t.Helper()
	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("first Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("first Delete() deleted = true, want false while readback is still active")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("DeleteOperatorControl() calls after first delete = %d, want 1", len(client.deleteRequests))
	}
	if !operatorControlDeleteAlreadyPending(resource) {
		t.Fatalf("status.status.async.current after first delete = %#v, want pending delete marker", resource.Status.OsokStatus.Async.Current)
	}
}

func assertOperatorControlPendingDeleteRetriesWithoutSecondDelete(
	t *testing.T,
	serviceClient OperatorControlServiceClient,
	client *fakeOperatorControlOCIClient,
	resource *operatoraccesscontrolv1beta1.OperatorControl,
) {
	t.Helper()
	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("second Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("second Delete() deleted = true, want false before terminal readback")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("DeleteOperatorControl() calls after second delete = %d, want still 1", len(client.deleteRequests))
	}
}

func assertOperatorControlPendingDeleteConfirms(
	t *testing.T,
	serviceClient OperatorControlServiceClient,
	client *fakeOperatorControlOCIClient,
	resource *operatoraccesscontrolv1beta1.OperatorControl,
) {
	t.Helper()
	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("third Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("third Delete() deleted = false, want true after terminal readback")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("DeleteOperatorControl() calls after terminal confirmation = %d, want still 1", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want set after terminal delete confirmation")
	}
}

func TestOperatorControlDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	resource := newTestOperatorControl()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOperatorControlID)
	resource.Status.Id = testOperatorControlID
	client := &fakeOperatorControlOCIClient{}
	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	client.get = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
		assertStringPtr(t, "get operatorControlId", request.OperatorControlId, testOperatorControlID)
		return operatoraccesscontrolsdk.GetOperatorControlResponse{}, serviceErr
	}

	deleted, err := newTestOperatorControlServiceClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteOperatorControl() calls = %d, want 0", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt is set after ambiguous 404")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestOperatorControlDeleteRejectsAuthShapedGeneratedRuntimeConfirmRead(t *testing.T) {
	resource := newTestOperatorControl()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOperatorControlID)
	resource.Status.Id = testOperatorControlID
	client := &fakeOperatorControlOCIClient{}
	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-confirm-read"
	getCalls := 0
	client.get = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlRequest) (operatoraccesscontrolsdk.GetOperatorControlResponse, error) {
		assertStringPtr(t, "get operatorControlId", request.OperatorControlId, testOperatorControlID)
		getCalls++
		if getCalls == 1 {
			return operatoraccesscontrolsdk.GetOperatorControlResponse{
				OperatorControl: sdkOperatorControlFromSpec(testOperatorControlID, resource.Spec, operatoraccesscontrolsdk.OperatorControlLifecycleStatesCreated),
			}, nil
		}
		return operatoraccesscontrolsdk.GetOperatorControlResponse{}, serviceErr
	}
	client.delete = func(_ context.Context, request operatoraccesscontrolsdk.DeleteOperatorControlRequest) (operatoraccesscontrolsdk.DeleteOperatorControlResponse, error) {
		t.Fatal("DeleteOperatorControl should not be called after auth-shaped generatedruntime confirm read")
		return operatoraccesscontrolsdk.DeleteOperatorControlResponse{}, nil
	}

	deleted, err := newTestOperatorControlServiceClient(client).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want auth-shaped generatedruntime confirm-read error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if getCalls != 2 {
		t.Fatalf("GetOperatorControl() calls = %d, want 2", getCalls)
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteOperatorControl() calls = %d, want 0", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt is set after ambiguous generatedruntime confirm read")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-confirm-read" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-confirm-read", got)
	}
}

func TestOperatorControlDeleteSkipsOCIWhileWritePending(t *testing.T) {
	resource := newTestOperatorControl()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOperatorControlID)
	resource.Status.Id = testOperatorControlID
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseUpdate,
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	}
	client := &fakeOperatorControlOCIClient{}

	deleted, err := newTestOperatorControlServiceClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while update is pending")
	}
	if len(client.getRequests) != 0 || len(client.deleteRequests) != 0 {
		t.Fatalf("OCI calls get=%d delete=%d, want none", len(client.getRequests), len(client.deleteRequests))
	}
	if got := resource.Status.OsokStatus.Message; got != operatorControlPendingWriteDeleteMessage {
		t.Fatalf("status message = %q, want pending write message", got)
	}
}

func TestOperatorControlCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	resource := newTestOperatorControl()
	client := &fakeOperatorControlOCIClient{}
	client.list = func(_ context.Context, request operatoraccesscontrolsdk.ListOperatorControlsRequest) (operatoraccesscontrolsdk.ListOperatorControlsResponse, error) {
		assertOperatorControlListRequest(t, request, resource.Spec)
		return operatoraccesscontrolsdk.ListOperatorControlsResponse{}, nil
	}
	serviceErr := errortest.NewServiceError(500, "InternalError", "internal error")
	serviceErr.OpcRequestID = "opc-create-error"
	client.create = func(context.Context, operatoraccesscontrolsdk.CreateOperatorControlRequest) (operatoraccesscontrolsdk.CreateOperatorControlResponse, error) {
		return operatoraccesscontrolsdk.CreateOperatorControlResponse{}, serviceErr
	}

	response, err := newTestOperatorControlServiceClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want unsuccessful", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-error" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-error", got)
	}
}

func newTestOperatorControlServiceClient(client *fakeOperatorControlOCIClient) OperatorControlServiceClient {
	if client == nil {
		client = &fakeOperatorControlOCIClient{}
	}
	return newOperatorControlServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
}

func newTestOperatorControl() *operatoraccesscontrolv1beta1.OperatorControl {
	return &operatoraccesscontrolv1beta1.OperatorControl{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "operator-control",
			Namespace: "default",
			UID:       "operator-control-uid",
		},
		Spec: operatoraccesscontrolv1beta1.OperatorControlSpec{
			OperatorControlName:     "operator-control",
			ApproverGroupsList:      []string{"group-a"},
			IsFullyPreApproved:      false,
			ResourceType:            string(operatoraccesscontrolsdk.ResourceTypesExacc),
			CompartmentId:           testOperatorControlCompartmentID,
			Description:             "operator control description",
			ApproversList:           []string{"user-a"},
			PreApprovedOpActionList: []string{"PATCH"},
			NumberOfApprovers:       1,
			EmailIdList:             []string{"approver@example.com"},
			SystemMessage:           "system message",
			FreeformTags:            map[string]string{"env": "test"},
			DefinedTags:             map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func sdkOperatorControlFromSpec(
	id string,
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
	lifecycleState operatoraccesscontrolsdk.OperatorControlLifecycleStatesEnum,
) operatoraccesscontrolsdk.OperatorControl {
	resourceType, _ := operatoraccesscontrolsdk.GetMappingResourceTypesEnum(spec.ResourceType)
	return operatoraccesscontrolsdk.OperatorControl{
		Id:                      common.String(id),
		OperatorControlName:     common.String(spec.OperatorControlName),
		CompartmentId:           common.String(spec.CompartmentId),
		Description:             optionalString(spec.Description),
		ApproversList:           slices.Clone(spec.ApproversList),
		ApproverGroupsList:      slices.Clone(spec.ApproverGroupsList),
		PreApprovedOpActionList: slices.Clone(spec.PreApprovedOpActionList),
		IsFullyPreApproved:      common.Bool(spec.IsFullyPreApproved),
		EmailIdList:             slices.Clone(spec.EmailIdList),
		ResourceType:            resourceType,
		SystemMessage:           optionalString(spec.SystemMessage),
		LifecycleState:          lifecycleState,
		NumberOfApprovers:       optionalInt(spec.NumberOfApprovers),
		FreeformTags:            mapsClone(spec.FreeformTags),
		DefinedTags:             operatorControlDefinedTags(spec.DefinedTags),
	}
}

func sdkOperatorControlSummary(
	id string,
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
	lifecycleState operatoraccesscontrolsdk.OperatorControlLifecycleStatesEnum,
) operatoraccesscontrolsdk.OperatorControlSummary {
	resourceType, _ := operatoraccesscontrolsdk.GetMappingResourceTypesEnum(spec.ResourceType)
	return operatoraccesscontrolsdk.OperatorControlSummary{
		Id:                  common.String(id),
		OperatorControlName: common.String(spec.OperatorControlName),
		CompartmentId:       common.String(spec.CompartmentId),
		IsFullyPreApproved:  common.Bool(spec.IsFullyPreApproved),
		ResourceType:        resourceType,
		NumberOfApprovers:   optionalInt(spec.NumberOfApprovers),
		LifecycleState:      lifecycleState,
		FreeformTags:        mapsClone(spec.FreeformTags),
		DefinedTags:         operatorControlDefinedTags(spec.DefinedTags),
	}
}

func assertOperatorControlCreateRequest(
	t *testing.T,
	request operatoraccesscontrolsdk.CreateOperatorControlRequest,
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
) {
	t.Helper()
	details := request.CreateOperatorControlDetails
	assertStringPtr(t, "create operatorControlName", details.OperatorControlName, spec.OperatorControlName)
	assertStringPtr(t, "create compartmentId", details.CompartmentId, spec.CompartmentId)
	if !slices.Equal(details.ApproverGroupsList, spec.ApproverGroupsList) {
		t.Fatalf("create approverGroupsList = %#v, want %#v", details.ApproverGroupsList, spec.ApproverGroupsList)
	}
	if details.IsFullyPreApproved == nil || *details.IsFullyPreApproved != spec.IsFullyPreApproved {
		t.Fatalf("create isFullyPreApproved = %v, want %v", details.IsFullyPreApproved, spec.IsFullyPreApproved)
	}
	if got, want := string(details.ResourceType), spec.ResourceType; got != want {
		t.Fatalf("create resourceType = %q, want %q", got, want)
	}
	assertOptionalCreateFields(t, details, spec)
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("create opc retry token is empty")
	}
}

func assertOptionalCreateFields(
	t *testing.T,
	details operatoraccesscontrolsdk.CreateOperatorControlDetails,
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
) {
	t.Helper()
	assertStringPtr(t, "create description", details.Description, spec.Description)
	assertStringPtr(t, "create systemMessage", details.SystemMessage, spec.SystemMessage)
	if !slices.Equal(details.ApproversList, spec.ApproversList) {
		t.Fatalf("create approversList = %#v, want %#v", details.ApproversList, spec.ApproversList)
	}
	if !slices.Equal(details.PreApprovedOpActionList, spec.PreApprovedOpActionList) {
		t.Fatalf("create preApprovedOpActionList = %#v, want %#v", details.PreApprovedOpActionList, spec.PreApprovedOpActionList)
	}
	if !slices.Equal(details.EmailIdList, spec.EmailIdList) {
		t.Fatalf("create emailIdList = %#v, want %#v", details.EmailIdList, spec.EmailIdList)
	}
	if details.NumberOfApprovers == nil || *details.NumberOfApprovers != spec.NumberOfApprovers {
		t.Fatalf("create numberOfApprovers = %v, want %d", details.NumberOfApprovers, spec.NumberOfApprovers)
	}
	if got := details.FreeformTags["env"]; got != spec.FreeformTags["env"] {
		t.Fatalf("create freeformTags.env = %q, want %q", got, spec.FreeformTags["env"])
	}
	if got := details.DefinedTags["Operations"]["CostCenter"]; got != spec.DefinedTags["Operations"]["CostCenter"] {
		t.Fatalf("create definedTags Operations.CostCenter = %v, want %s", got, spec.DefinedTags["Operations"]["CostCenter"])
	}
}

func assertOperatorControlUpdateRequest(
	t *testing.T,
	request operatoraccesscontrolsdk.UpdateOperatorControlRequest,
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
) {
	t.Helper()
	details := request.UpdateOperatorControlDetails
	assertOperatorControlUpdateStrings(t, details, spec)
	assertOperatorControlUpdateLists(t, details, spec)
	assertOperatorControlUpdatePreapproval(t, details, spec)
	assertOperatorControlUpdateTags(t, details, spec)
}

func assertOperatorControlUpdateStrings(
	t *testing.T,
	details operatoraccesscontrolsdk.UpdateOperatorControlDetails,
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
) {
	t.Helper()
	assertStringPtr(t, "update operatorControlName", details.OperatorControlName, spec.OperatorControlName)
	assertStringPtr(t, "update description", details.Description, spec.Description)
	assertStringPtr(t, "update systemMessage", details.SystemMessage, spec.SystemMessage)
}

func assertOperatorControlUpdateLists(
	t *testing.T,
	details operatoraccesscontrolsdk.UpdateOperatorControlDetails,
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
) {
	t.Helper()
	if !slices.Equal(details.ApproverGroupsList, spec.ApproverGroupsList) {
		t.Fatalf("update approverGroupsList = %#v, want %#v", details.ApproverGroupsList, spec.ApproverGroupsList)
	}
	if !slices.Equal(details.ApproversList, spec.ApproversList) {
		t.Fatalf("update approversList = %#v, want %#v", details.ApproversList, spec.ApproversList)
	}
	if !slices.Equal(details.PreApprovedOpActionList, spec.PreApprovedOpActionList) {
		t.Fatalf("update preApprovedOpActionList = %#v, want %#v", details.PreApprovedOpActionList, spec.PreApprovedOpActionList)
	}
	if !slices.Equal(details.EmailIdList, spec.EmailIdList) {
		t.Fatalf("update emailIdList = %#v, want %#v", details.EmailIdList, spec.EmailIdList)
	}
}

func assertOperatorControlUpdatePreapproval(
	t *testing.T,
	details operatoraccesscontrolsdk.UpdateOperatorControlDetails,
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
) {
	t.Helper()
	if details.IsFullyPreApproved == nil || *details.IsFullyPreApproved != spec.IsFullyPreApproved {
		t.Fatalf("update isFullyPreApproved = %v, want %v", details.IsFullyPreApproved, spec.IsFullyPreApproved)
	}
	if details.NumberOfApprovers == nil || *details.NumberOfApprovers != spec.NumberOfApprovers {
		t.Fatalf("update numberOfApprovers = %v, want %d", details.NumberOfApprovers, spec.NumberOfApprovers)
	}
}

func assertOperatorControlUpdateTags(
	t *testing.T,
	details operatoraccesscontrolsdk.UpdateOperatorControlDetails,
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
) {
	t.Helper()
	if got := details.FreeformTags["env"]; got != spec.FreeformTags["env"] {
		t.Fatalf("update freeformTags.env = %q, want %q", got, spec.FreeformTags["env"])
	}
	if got := details.DefinedTags["Operations"]["CostCenter"]; got != spec.DefinedTags["Operations"]["CostCenter"] {
		t.Fatalf("update definedTags Operations.CostCenter = %v, want %s", got, spec.DefinedTags["Operations"]["CostCenter"])
	}
}

func assertOperatorControlListRequest(
	t *testing.T,
	request operatoraccesscontrolsdk.ListOperatorControlsRequest,
	spec operatoraccesscontrolv1beta1.OperatorControlSpec,
) {
	t.Helper()
	assertStringPtr(t, "list compartmentId", request.CompartmentId, spec.CompartmentId)
	assertStringPtr(t, "list displayName", request.DisplayName, spec.OperatorControlName)
	assertStringPtr(t, "list resourceType", request.ResourceType, spec.ResourceType)
}

func assertStringPtr(t *testing.T, name string, value *string, want string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if got := strings.TrimSpace(*value); got != want {
		t.Fatalf("%s = %q, want %q", name, got, want)
	}
}

func assertStringSliceContains(t *testing.T, name string, values []string, want string) {
	t.Helper()
	if slices.Contains(values, want) {
		return
	}
	t.Fatalf("%s = %#v, want element %q", name, values, want)
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}

func optionalInt(value int) *int {
	if value == 0 {
		return nil
	}
	return common.Int(value)
}

func mapsClone(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	cloned := make(map[string]string, len(value))
	for key, child := range value {
		cloned[key] = child
	}
	return cloned
}

func lastOperatorControlCondition(resource *operatoraccesscontrolv1beta1.OperatorControl) shared.OSOKCondition {
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		return shared.OSOKCondition{}
	}
	return resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1]
}
