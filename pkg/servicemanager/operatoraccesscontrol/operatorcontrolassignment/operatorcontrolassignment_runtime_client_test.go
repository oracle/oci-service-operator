/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package operatorcontrolassignment

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	operatoraccesscontrolsdk "github.com/oracle/oci-go-sdk/v65/operatoraccesscontrol"
	operatoraccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/operatoraccesscontrol/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testOperatorControlAssignmentID        = "ocid1.operatorcontrolassignment.oc1..assignment"
	testOperatorControlAssignmentOtherID   = "ocid1.operatorcontrolassignment.oc1..other"
	testOperatorControlAssignmentControlID = "ocid1.operatorcontrol.oc1..control"
	testOperatorControlAssignmentResource  = "ocid1.exacc.oc1..resource"
	testOperatorControlAssignmentComp      = "ocid1.compartment.oc1..assignment"
	testOperatorControlAssignmentResComp   = "ocid1.compartment.oc1..resource"
)

type fakeOperatorControlAssignmentOCIClient struct {
	createRequests []operatoraccesscontrolsdk.CreateOperatorControlAssignmentRequest
	getRequests    []operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest
	listRequests   []operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest
	updateRequests []operatoraccesscontrolsdk.UpdateOperatorControlAssignmentRequest
	deleteRequests []operatoraccesscontrolsdk.DeleteOperatorControlAssignmentRequest

	createFn func(context.Context, operatoraccesscontrolsdk.CreateOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.CreateOperatorControlAssignmentResponse, error)
	getFn    func(context.Context, operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error)
	listFn   func(context.Context, operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest) (operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse, error)
	updateFn func(context.Context, operatoraccesscontrolsdk.UpdateOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse, error)
	deleteFn func(context.Context, operatoraccesscontrolsdk.DeleteOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse, error)
}

func (f *fakeOperatorControlAssignmentOCIClient) CreateOperatorControlAssignment(
	ctx context.Context,
	request operatoraccesscontrolsdk.CreateOperatorControlAssignmentRequest,
) (operatoraccesscontrolsdk.CreateOperatorControlAssignmentResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return operatoraccesscontrolsdk.CreateOperatorControlAssignmentResponse{}, nil
}

func (f *fakeOperatorControlAssignmentOCIClient) GetOperatorControlAssignment(
	ctx context.Context,
	request operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest,
) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "assignment not found")
}

func (f *fakeOperatorControlAssignmentOCIClient) ListOperatorControlAssignments(
	ctx context.Context,
	request operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest,
) (operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse{}, nil
}

func (f *fakeOperatorControlAssignmentOCIClient) UpdateOperatorControlAssignment(
	ctx context.Context,
	request operatoraccesscontrolsdk.UpdateOperatorControlAssignmentRequest,
) (operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse{}, nil
}

func (f *fakeOperatorControlAssignmentOCIClient) DeleteOperatorControlAssignment(
	ctx context.Context,
	request operatoraccesscontrolsdk.DeleteOperatorControlAssignmentRequest,
) (operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse{}, nil
}

func newOperatorControlAssignmentTestClient(fake *fakeOperatorControlAssignmentOCIClient) OperatorControlAssignmentServiceClient {
	return newOperatorControlAssignmentServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
}

func newOperatorControlAssignmentResource() *operatoraccesscontrolv1beta1.OperatorControlAssignment {
	return &operatoraccesscontrolv1beta1.OperatorControlAssignment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "assignment",
			Namespace: "default",
			UID:       types.UID("operatorcontrolassignment-uid"),
		},
		Spec: operatoraccesscontrolv1beta1.OperatorControlAssignmentSpec{
			OperatorControlId:              testOperatorControlAssignmentControlID,
			ResourceId:                     testOperatorControlAssignmentResource,
			ResourceName:                   "target-exacc",
			ResourceType:                   string(operatoraccesscontrolsdk.ResourceTypesExacc),
			ResourceCompartmentId:          testOperatorControlAssignmentResComp,
			CompartmentId:                  testOperatorControlAssignmentComp,
			IsEnforcedAlways:               true,
			TimeAssignmentFrom:             "2026-04-29T10:00:00Z",
			TimeAssignmentTo:               "2026-04-30T10:00:00Z",
			Comment:                        "initial assignment",
			IsLogForwarded:                 true,
			RemoteSyslogServerAddress:      "192.0.2.10",
			RemoteSyslogServerPort:         6514,
			RemoteSyslogServerCACert:       "ca-cert",
			IsHypervisorLogForwarded:       true,
			IsAutoApproveDuringMaintenance: true,
			FreeformTags:                   map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func newTrackedOperatorControlAssignmentResource(id string) *operatoraccesscontrolv1beta1.OperatorControlAssignment {
	resource := newOperatorControlAssignmentResource()
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	return resource
}

func sdkOperatorControlAssignment(
	id string,
	spec operatoraccesscontrolv1beta1.OperatorControlAssignmentSpec,
	state operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesEnum,
) operatoraccesscontrolsdk.OperatorControlAssignment {
	return operatoraccesscontrolsdk.OperatorControlAssignment{
		Id:                             common.String(id),
		OperatorControlId:              common.String(spec.OperatorControlId),
		ResourceId:                     common.String(spec.ResourceId),
		ResourceName:                   common.String(spec.ResourceName),
		ResourceType:                   operatoraccesscontrolsdk.ResourceTypesEnum(spec.ResourceType),
		ResourceCompartmentId:          common.String(spec.ResourceCompartmentId),
		CompartmentId:                  common.String(spec.CompartmentId),
		TimeAssignmentFrom:             mustOperatorControlAssignmentSDKTime(spec.TimeAssignmentFrom),
		TimeAssignmentTo:               mustOperatorControlAssignmentSDKTime(spec.TimeAssignmentTo),
		IsEnforcedAlways:               common.Bool(spec.IsEnforcedAlways),
		LifecycleState:                 state,
		LifecycleDetails:               common.String("state " + string(state)),
		TimeOfAssignment:               mustOperatorControlAssignmentSDKTime("2026-04-29T10:01:00Z"),
		Comment:                        optionalOperatorControlAssignmentString(spec.Comment),
		IsLogForwarded:                 common.Bool(spec.IsLogForwarded),
		RemoteSyslogServerAddress:      optionalOperatorControlAssignmentString(spec.RemoteSyslogServerAddress),
		RemoteSyslogServerPort:         common.Int(spec.RemoteSyslogServerPort),
		RemoteSyslogServerCACert:       optionalOperatorControlAssignmentString(spec.RemoteSyslogServerCACert),
		IsHypervisorLogForwarded:       common.Bool(spec.IsHypervisorLogForwarded),
		OpControlName:                  common.String("operator-control"),
		IsAutoApproveDuringMaintenance: common.Bool(spec.IsAutoApproveDuringMaintenance),
		IsDefaultAssignment:            common.Bool(false),
		FreeformTags:                   cloneOperatorControlAssignmentStringMap(spec.FreeformTags),
		DefinedTags:                    operatorControlAssignmentDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func sdkOperatorControlAssignmentSummary(
	id string,
	spec operatoraccesscontrolv1beta1.OperatorControlAssignmentSpec,
	state operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesEnum,
) operatoraccesscontrolsdk.OperatorControlAssignmentSummary {
	return operatoraccesscontrolsdk.OperatorControlAssignmentSummary{
		Id:                        common.String(id),
		OperatorControlId:         common.String(spec.OperatorControlId),
		ResourceId:                common.String(spec.ResourceId),
		CompartmentId:             common.String(spec.CompartmentId),
		ResourceType:              operatoraccesscontrolsdk.ResourceTypesEnum(spec.ResourceType),
		ResourceName:              common.String(spec.ResourceName),
		OpControlName:             common.String("operator-control"),
		TimeAssignmentFrom:        mustOperatorControlAssignmentSDKTime(spec.TimeAssignmentFrom),
		TimeAssignmentTo:          mustOperatorControlAssignmentSDKTime(spec.TimeAssignmentTo),
		IsEnforcedAlways:          common.Bool(spec.IsEnforcedAlways),
		TimeOfAssignment:          mustOperatorControlAssignmentSDKTime("2026-04-29T10:01:00Z"),
		IsLogForwarded:            common.Bool(spec.IsLogForwarded),
		RemoteSyslogServerAddress: optionalOperatorControlAssignmentString(spec.RemoteSyslogServerAddress),
		RemoteSyslogServerPort:    common.Int(spec.RemoteSyslogServerPort),
		IsHypervisorLogForwarded:  common.Bool(spec.IsHypervisorLogForwarded),
		LifecycleState:            state,
		LifecycleDetails:          common.String("state " + string(state)),
		FreeformTags:              cloneOperatorControlAssignmentStringMap(spec.FreeformTags),
		DefinedTags:               operatorControlAssignmentDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func mustOperatorControlAssignmentSDKTime(value string) *common.SDKTime {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		panic(err)
	}
	return &common.SDKTime{Time: parsed}
}

func TestOperatorControlAssignmentRuntimeSemantics(t *testing.T) {
	got := newOperatorControlAssignmentRuntimeSemantics()
	if got == nil {
		t.Fatal("newOperatorControlAssignmentRuntimeSemantics() = nil")
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("finalizer policy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v follow-up %#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	requireOperatorControlAssignmentStrings(t, "list match fields", got.List.MatchFields, []string{
		"compartmentId",
		"operatorControlId",
		"resourceId",
		"resourceName",
		"resourceType",
	})
	requireOperatorControlAssignmentStrings(t, "force-new fields", got.Mutation.ForceNew, []string{
		"operatorControlId",
		"resourceId",
		"resourceName",
		"resourceType",
		"resourceCompartmentId",
		"compartmentId",
	})
}

func TestOperatorControlAssignmentCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	resource := newOperatorControlAssignmentResource()
	fake := &fakeOperatorControlAssignmentOCIClient{}
	fake.listFn = func(_ context.Context, request operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest) (operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse, error) {
		assertOperatorControlAssignmentListRequest(t, request, resource, "")
		return operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse{}, nil
	}
	fake.createFn = func(_ context.Context, request operatoraccesscontrolsdk.CreateOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.CreateOperatorControlAssignmentResponse, error) {
		assertOperatorControlAssignmentCreateRequest(t, request, resource)
		requireOperatorControlAssignmentStringPtr(t, "create retry token", request.OpcRetryToken, string(resource.UID))
		return operatoraccesscontrolsdk.CreateOperatorControlAssignmentResponse{
			OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, resource.Spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
			OpcRequestId:              common.String("opc-create-1"),
		}, nil
	}
	fake.getFn = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error) {
		requireOperatorControlAssignmentStringPtr(t, "get id", request.OperatorControlAssignmentId, testOperatorControlAssignmentID)
		return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{
			OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, resource.Spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
			OpcRequestId:              common.String("opc-get-1"),
		}, nil
	}

	response, err := newOperatorControlAssignmentTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateOperatorControlAssignment() calls = %d, want 1", len(fake.createRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testOperatorControlAssignmentID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testOperatorControlAssignmentID)
	}
	if got := resource.Status.Id; got != testOperatorControlAssignmentID {
		t.Fatalf("status.id = %q, want %q", got, testOperatorControlAssignmentID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
}

func TestOperatorControlAssignmentCreateOrUpdateBindsFromPaginatedListWithoutDuplicateCreate(t *testing.T) {
	resource := newOperatorControlAssignmentResource()
	fake := &fakeOperatorControlAssignmentOCIClient{}
	fake.listFn = func(_ context.Context, request operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest) (operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse, error) {
		switch page := operatorControlAssignmentStringValue(request.Page); page {
		case "":
			otherSpec := resource.Spec
			otherSpec.ResourceId = "ocid1.exacc.oc1..other"
			return operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse{
				OperatorControlAssignmentCollection: operatoraccesscontrolsdk.OperatorControlAssignmentCollection{
					Items: []operatoraccesscontrolsdk.OperatorControlAssignmentSummary{
						sdkOperatorControlAssignmentSummary(testOperatorControlAssignmentOtherID, otherSpec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse{
				OperatorControlAssignmentCollection: operatoraccesscontrolsdk.OperatorControlAssignmentCollection{
					Items: []operatoraccesscontrolsdk.OperatorControlAssignmentSummary{
						sdkOperatorControlAssignmentSummary(testOperatorControlAssignmentID, resource.Spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
					},
				},
			}, nil
		default:
			t.Fatalf("ListOperatorControlAssignments() page = %q, want empty or page-2", page)
			return operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse{}, nil
		}
	}
	fake.getFn = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error) {
		requireOperatorControlAssignmentStringPtr(t, "get id", request.OperatorControlAssignmentId, testOperatorControlAssignmentID)
		return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{
			OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, resource.Spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
		}, nil
	}

	response, err := newOperatorControlAssignmentTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateOperatorControlAssignment() calls = %d, want 0", len(fake.createRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateOperatorControlAssignment() calls = %d, want 0", len(fake.updateRequests))
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListOperatorControlAssignments() calls = %d, want 2 paginated calls", len(fake.listRequests))
	}
	assertOperatorControlAssignmentListRequest(t, fake.listRequests[0], resource, "")
	assertOperatorControlAssignmentListRequest(t, fake.listRequests[1], resource, "page-2")
	if got := string(resource.Status.OsokStatus.Ocid); got != testOperatorControlAssignmentID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testOperatorControlAssignmentID)
	}
}

func TestOperatorControlAssignmentCreateOrUpdateNoopsWhenObservedMatchesSpec(t *testing.T) {
	resource := newTrackedOperatorControlAssignmentResource(testOperatorControlAssignmentID)
	fake := &fakeOperatorControlAssignmentOCIClient{}
	fake.getFn = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error) {
		requireOperatorControlAssignmentStringPtr(t, "get id", request.OperatorControlAssignmentId, testOperatorControlAssignmentID)
		return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{
			OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, resource.Spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
		}, nil
	}

	response, err := newOperatorControlAssignmentTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if len(fake.listRequests) != 0 || len(fake.createRequests) != 0 || len(fake.updateRequests) != 0 {
		t.Fatalf("OCI calls list/create/update = %d/%d/%d, want only get", len(fake.listRequests), len(fake.createRequests), len(fake.updateRequests))
	}
}

func TestOperatorControlAssignmentCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	resource := newTrackedOperatorControlAssignmentResource(testOperatorControlAssignmentID)
	currentSpec := resource.Spec
	currentSpec.IsEnforcedAlways = true
	currentSpec.Comment = "old assignment"
	currentSpec.IsLogForwarded = true

	resource.Spec.IsEnforcedAlways = false
	resource.Spec.Comment = "updated assignment"
	resource.Spec.IsLogForwarded = false

	fake := &fakeOperatorControlAssignmentOCIClient{}
	getCalls := 0
	fake.getFn = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error) {
		requireOperatorControlAssignmentStringPtr(t, "get id", request.OperatorControlAssignmentId, testOperatorControlAssignmentID)
		getCalls++
		spec := currentSpec
		if getCalls > 1 {
			spec = resource.Spec
		}
		return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{
			OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
		}, nil
	}
	fake.updateFn = func(_ context.Context, request operatoraccesscontrolsdk.UpdateOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse, error) {
		requireOperatorControlAssignmentStringPtr(t, "update id", request.OperatorControlAssignmentId, testOperatorControlAssignmentID)
		requireOperatorControlAssignmentBoolPtr(t, "update isEnforcedAlways", request.IsEnforcedAlways, false)
		requireOperatorControlAssignmentStringPtr(t, "update comment", request.Comment, resource.Spec.Comment)
		requireOperatorControlAssignmentBoolPtr(t, "update isLogForwarded", request.IsLogForwarded, false)
		return operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse{
			OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, resource.Spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
			OpcRequestId:              common.String("opc-update-1"),
		}, nil
	}

	response, err := newOperatorControlAssignmentTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateOperatorControlAssignment() calls = %d, want 1", len(fake.updateRequests))
	}
	if got := resource.Status.Comment; got != resource.Spec.Comment {
		t.Fatalf("status.comment = %q, want %q", got, resource.Spec.Comment)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestOperatorControlAssignmentCreateOrUpdateClearsOptionalTimeAndStringFields(t *testing.T) {
	resource := newTrackedOperatorControlAssignmentResource(testOperatorControlAssignmentID)
	currentSpec := resource.Spec
	resource.Spec.TimeAssignmentFrom = ""
	resource.Spec.Comment = ""

	fake := &fakeOperatorControlAssignmentOCIClient{}
	getCalls := 0
	fake.getFn = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error) {
		requireOperatorControlAssignmentStringPtr(t, "get id", request.OperatorControlAssignmentId, testOperatorControlAssignmentID)
		getCalls++
		spec := currentSpec
		if getCalls > 1 {
			spec = resource.Spec
		}
		return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{
			OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
		}, nil
	}
	fake.updateFn = func(_ context.Context, request operatoraccesscontrolsdk.UpdateOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse, error) {
		requireOperatorControlAssignmentBoolPtr(t, "update isEnforcedAlways", request.IsEnforcedAlways, true)
		if request.TimeAssignmentFrom != nil {
			t.Fatalf("update timeAssignmentFrom = %v, want nil clear", request.TimeAssignmentFrom)
		}
		if request.Comment != nil {
			t.Fatalf("update comment = %q, want nil clear", *request.Comment)
		}
		return operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse{
			OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, resource.Spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
			OpcRequestId:              common.String("opc-update-clear"),
		}, nil
	}

	response, err := newOperatorControlAssignmentTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateOperatorControlAssignment() calls = %d, want 1", len(fake.updateRequests))
	}
	if resource.Status.TimeAssignmentFrom != "" {
		t.Fatalf("status.timeAssignmentFrom = %q, want empty", resource.Status.TimeAssignmentFrom)
	}
	if resource.Status.Comment != "" {
		t.Fatalf("status.comment = %q, want empty", resource.Status.Comment)
	}
}

func TestOperatorControlAssignmentCreateOrUpdateSetsRemoteSyslogPortToZero(t *testing.T) {
	resource := newTrackedOperatorControlAssignmentResource(testOperatorControlAssignmentID)
	currentSpec := resource.Spec
	resource.Spec.RemoteSyslogServerPort = 0

	fake := &fakeOperatorControlAssignmentOCIClient{}
	getCalls := 0
	fake.getFn = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error) {
		requireOperatorControlAssignmentStringPtr(t, "get id", request.OperatorControlAssignmentId, testOperatorControlAssignmentID)
		getCalls++
		spec := currentSpec
		if getCalls > 1 {
			spec = resource.Spec
		}
		return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{
			OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
		}, nil
	}
	fake.updateFn = func(_ context.Context, request operatoraccesscontrolsdk.UpdateOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse, error) {
		requireOperatorControlAssignmentIntPtr(t, "update remoteSyslogServerPort", request.RemoteSyslogServerPort, 0)
		return operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse{
			OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, resource.Spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
			OpcRequestId:              common.String("opc-update-port"),
		}, nil
	}

	response, err := newOperatorControlAssignmentTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateOperatorControlAssignment() calls = %d, want 1", len(fake.updateRequests))
	}
	if got := resource.Status.RemoteSyslogServerPort; got != 0 {
		t.Fatalf("status.remoteSyslogServerPort = %d, want 0", got)
	}
}

func TestOperatorControlAssignmentCreateOrUpdateClearsTagMaps(t *testing.T) {
	resource := newTrackedOperatorControlAssignmentResource(testOperatorControlAssignmentID)
	currentSpec := resource.Spec
	resource.Spec.FreeformTags = nil
	resource.Spec.DefinedTags = nil

	fake := &fakeOperatorControlAssignmentOCIClient{}
	getCalls := 0
	fake.getFn = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error) {
		requireOperatorControlAssignmentStringPtr(t, "get id", request.OperatorControlAssignmentId, testOperatorControlAssignmentID)
		getCalls++
		spec := currentSpec
		if getCalls > 1 {
			spec = resource.Spec
		}
		return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{
			OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
		}, nil
	}
	fake.updateFn = func(_ context.Context, request operatoraccesscontrolsdk.UpdateOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse, error) {
		if request.FreeformTags == nil {
			t.Fatal("update freeformTags = nil, want empty map clear")
		}
		if len(request.FreeformTags) != 0 {
			t.Fatalf("update freeformTags = %v, want empty map clear", request.FreeformTags)
		}
		if request.DefinedTags == nil {
			t.Fatal("update definedTags = nil, want empty map clear")
		}
		if len(request.DefinedTags) != 0 {
			t.Fatalf("update definedTags = %v, want empty map clear", request.DefinedTags)
		}
		return operatoraccesscontrolsdk.UpdateOperatorControlAssignmentResponse{
			OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, resource.Spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
			OpcRequestId:              common.String("opc-update-tags"),
		}, nil
	}

	response, err := newOperatorControlAssignmentTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateOperatorControlAssignment() calls = %d, want 1", len(fake.updateRequests))
	}
}

func TestOperatorControlAssignmentCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := newTrackedOperatorControlAssignmentResource(testOperatorControlAssignmentID)
	currentSpec := resource.Spec
	currentSpec.ResourceId = "ocid1.exacc.oc1..old"
	resource.Spec.ResourceId = "ocid1.exacc.oc1..new"

	fake := &fakeOperatorControlAssignmentOCIClient{}
	fake.getFn = func(_ context.Context, _ operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error) {
		return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{
			OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, currentSpec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
		}, nil
	}

	response, err := newOperatorControlAssignmentTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when resourceId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want resourceId create-only drift rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateOperatorControlAssignment() calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestOperatorControlAssignmentDeleteRetainsFinalizerWhileReadbackIsPending(t *testing.T) {
	resource := newTrackedOperatorControlAssignmentResource(testOperatorControlAssignmentID)
	fake := &fakeOperatorControlAssignmentOCIClient{}
	setOperatorControlAssignmentDeleteReadback(t, fake, resource, nil, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesDeleting)
	fake.deleteFn = func(_ context.Context, request operatoraccesscontrolsdk.DeleteOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse, error) {
		requireOperatorControlAssignmentStringPtr(t, "delete id", request.OperatorControlAssignmentId, testOperatorControlAssignmentID)
		return operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}

	deleted, err := newOperatorControlAssignmentTestClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete is pending")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteOperatorControlAssignment() calls = %d, want 1", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current = %#v, want pending delete operation", current)
	}
}

func TestOperatorControlAssignmentDeleteReleasesFinalizerAfterUnambiguousNotFound(t *testing.T) {
	resource := newTrackedOperatorControlAssignmentResource(testOperatorControlAssignmentID)
	fake := &fakeOperatorControlAssignmentOCIClient{}
	setOperatorControlAssignmentDeleteReadback(t, fake, resource, errortest.NewServiceError(404, errorutil.NotFound, "assignment deleted"), "")
	fake.deleteFn = func(_ context.Context, request operatoraccesscontrolsdk.DeleteOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse, error) {
		requireOperatorControlAssignmentStringPtr(t, "delete id", request.OperatorControlAssignmentId, testOperatorControlAssignmentID)
		return operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}

	deleted, err := newOperatorControlAssignmentTestClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not found")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteOperatorControlAssignment() calls = %d, want 1", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
}

func TestOperatorControlAssignmentDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	resource := newTrackedOperatorControlAssignmentResource(testOperatorControlAssignmentID)
	fake := &fakeOperatorControlAssignmentOCIClient{}
	fake.getFn = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error) {
		requireOperatorControlAssignmentStringPtr(t, "get id", request.OperatorControlAssignmentId, testOperatorControlAssignmentID)
		return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
	}
	fake.deleteFn = func(_ context.Context, _ operatoraccesscontrolsdk.DeleteOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse, error) {
		return operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}

	deleted, err := newOperatorControlAssignmentTestClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped pre-delete read", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous pre-delete read")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteOperatorControlAssignment() calls = %d, want 0 after ambiguous pre-delete read", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id from OCI error", got)
	}
}

func setOperatorControlAssignmentDeleteReadback(
	t *testing.T,
	fake *fakeOperatorControlAssignmentOCIClient,
	resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
	confirmErr error,
	confirm operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesEnum,
) {
	t.Helper()
	getCalls := 0
	fake.getFn = func(_ context.Context, request operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error) {
		requireOperatorControlAssignmentStringPtr(t, "get id", request.OperatorControlAssignmentId, testOperatorControlAssignmentID)
		getCalls++
		if getCalls <= 2 {
			return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{
				OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, resource.Spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
			}, nil
		}
		if confirmErr != nil {
			return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{}, confirmErr
		}
		return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{
			OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, resource.Spec, confirm),
		}, nil
	}
}

func TestOperatorControlAssignmentDeleteKeepsAuthShapedConfirmReadFatal(t *testing.T) {
	resource := newTrackedOperatorControlAssignmentResource(testOperatorControlAssignmentID)
	fake := &fakeOperatorControlAssignmentOCIClient{}
	getCalls := 0
	fake.getFn = func(_ context.Context, _ operatoraccesscontrolsdk.GetOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse, error) {
		getCalls++
		if getCalls == 1 {
			return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{
				OperatorControlAssignment: sdkOperatorControlAssignment(testOperatorControlAssignmentID, resource.Spec, operatoraccesscontrolsdk.OperatorControlAssignmentLifecycleStatesApplied),
			}, nil
		}
		return operatoraccesscontrolsdk.GetOperatorControlAssignmentResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
	}
	fake.deleteFn = func(_ context.Context, _ operatoraccesscontrolsdk.DeleteOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse, error) {
		return operatoraccesscontrolsdk.DeleteOperatorControlAssignmentResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}

	deleted, err := newOperatorControlAssignmentTestClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped 404", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous confirm read")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteOperatorControlAssignment() calls = %d, want 0 after ambiguous already-pending confirm read", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id from OCI error", got)
	}
}

func TestOperatorControlAssignmentCreateRecordsOCIErrorRequestID(t *testing.T) {
	resource := newOperatorControlAssignmentResource()
	fake := &fakeOperatorControlAssignmentOCIClient{}
	fake.listFn = func(context.Context, operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest) (operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse, error) {
		return operatoraccesscontrolsdk.ListOperatorControlAssignmentsResponse{}, nil
	}
	fake.createFn = func(context.Context, operatoraccesscontrolsdk.CreateOperatorControlAssignmentRequest) (operatoraccesscontrolsdk.CreateOperatorControlAssignmentResponse, error) {
		return operatoraccesscontrolsdk.CreateOperatorControlAssignmentResponse{}, errortest.NewServiceError(500, "InternalError", "create failed")
	}

	response, err := newOperatorControlAssignmentTestClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func assertOperatorControlAssignmentCreateRequest(
	t *testing.T,
	request operatoraccesscontrolsdk.CreateOperatorControlAssignmentRequest,
	resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
) {
	t.Helper()
	spec := resource.Spec
	requireOperatorControlAssignmentStringPtr(t, "create operatorControlId", request.OperatorControlId, spec.OperatorControlId)
	requireOperatorControlAssignmentStringPtr(t, "create resourceId", request.ResourceId, spec.ResourceId)
	requireOperatorControlAssignmentStringPtr(t, "create resourceName", request.ResourceName, spec.ResourceName)
	if request.ResourceType != operatoraccesscontrolsdk.ResourceTypesEnum(spec.ResourceType) {
		t.Fatalf("create resourceType = %q, want %q", request.ResourceType, spec.ResourceType)
	}
	requireOperatorControlAssignmentStringPtr(t, "create resourceCompartmentId", request.ResourceCompartmentId, spec.ResourceCompartmentId)
	requireOperatorControlAssignmentStringPtr(t, "create compartmentId", request.CompartmentId, spec.CompartmentId)
	requireOperatorControlAssignmentBoolPtr(t, "create isEnforcedAlways", request.IsEnforcedAlways, spec.IsEnforcedAlways)
	requireOperatorControlAssignmentStringPtr(t, "create comment", request.Comment, spec.Comment)
	requireOperatorControlAssignmentBoolPtr(t, "create isLogForwarded", request.IsLogForwarded, true)
	requireOperatorControlAssignmentStringPtr(t, "create remoteSyslogServerAddress", request.RemoteSyslogServerAddress, spec.RemoteSyslogServerAddress)
	if request.RemoteSyslogServerPort == nil || *request.RemoteSyslogServerPort != spec.RemoteSyslogServerPort {
		t.Fatalf("create remoteSyslogServerPort = %v, want %d", request.RemoteSyslogServerPort, spec.RemoteSyslogServerPort)
	}
	requireOperatorControlAssignmentStringPtr(t, "create remoteSyslogServerCACert", request.RemoteSyslogServerCACert, spec.RemoteSyslogServerCACert)
	requireOperatorControlAssignmentBoolPtr(t, "create isHypervisorLogForwarded", request.IsHypervisorLogForwarded, true)
	requireOperatorControlAssignmentBoolPtr(t, "create isAutoApproveDuringMaintenance", request.IsAutoApproveDuringMaintenance, true)
	if got := request.FreeformTags["env"]; got != "dev" {
		t.Fatalf("create freeformTags[env] = %q, want dev", got)
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("create definedTags Operations.CostCenter = %v, want 42", got)
	}
	if request.TimeAssignmentFrom == nil || !request.TimeAssignmentFrom.Equal(mustOperatorControlAssignmentSDKTime(spec.TimeAssignmentFrom).Time) {
		t.Fatalf("create timeAssignmentFrom = %v, want %s", request.TimeAssignmentFrom, spec.TimeAssignmentFrom)
	}
	if request.TimeAssignmentTo == nil || !request.TimeAssignmentTo.Equal(mustOperatorControlAssignmentSDKTime(spec.TimeAssignmentTo).Time) {
		t.Fatalf("create timeAssignmentTo = %v, want %s", request.TimeAssignmentTo, spec.TimeAssignmentTo)
	}
}

func assertOperatorControlAssignmentListRequest(
	t *testing.T,
	request operatoraccesscontrolsdk.ListOperatorControlAssignmentsRequest,
	resource *operatoraccesscontrolv1beta1.OperatorControlAssignment,
	page string,
) {
	t.Helper()
	requireOperatorControlAssignmentStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireOperatorControlAssignmentStringPtr(t, "list resourceName", request.ResourceName, resource.Spec.ResourceName)
	requireOperatorControlAssignmentStringPtr(t, "list resourceType", request.ResourceType, resource.Spec.ResourceType)
	if got := operatorControlAssignmentStringValue(request.Page); got != page {
		t.Fatalf("list page = %q, want %q", got, page)
	}
	if request.LifecycleState != "" {
		t.Fatalf("list lifecycleState = %q, want empty so delete/create reads do not hide state transitions", request.LifecycleState)
	}
}

func requireOperatorControlAssignmentStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireOperatorControlAssignmentBoolPtr(t *testing.T, name string, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %t", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %t, want %t", name, *got, want)
	}
}

func requireOperatorControlAssignmentIntPtr(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %d", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %d, want %d", name, *got, want)
	}
}

func requireOperatorControlAssignmentStrings(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d: got %v want %v", name, len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q: got %v want %v", name, i, got[i], want[i], got, want)
		}
	}
}
