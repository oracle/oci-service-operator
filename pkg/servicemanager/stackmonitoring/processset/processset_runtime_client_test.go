/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package processset

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testProcessSetID            = "ocid1.stackmonitoringprocessset.oc1..processset"
	testProcessSetCompartmentID = "ocid1.compartment.oc1..processset"
	testProcessSetDisplayName   = "process-set-sample"
)

type fakeProcessSetOCIClient struct {
	createFn func(context.Context, stackmonitoringsdk.CreateProcessSetRequest) (stackmonitoringsdk.CreateProcessSetResponse, error)
	getFn    func(context.Context, stackmonitoringsdk.GetProcessSetRequest) (stackmonitoringsdk.GetProcessSetResponse, error)
	listFn   func(context.Context, stackmonitoringsdk.ListProcessSetsRequest) (stackmonitoringsdk.ListProcessSetsResponse, error)
	updateFn func(context.Context, stackmonitoringsdk.UpdateProcessSetRequest) (stackmonitoringsdk.UpdateProcessSetResponse, error)
	deleteFn func(context.Context, stackmonitoringsdk.DeleteProcessSetRequest) (stackmonitoringsdk.DeleteProcessSetResponse, error)
}

func (f *fakeProcessSetOCIClient) CreateProcessSet(
	ctx context.Context,
	request stackmonitoringsdk.CreateProcessSetRequest,
) (stackmonitoringsdk.CreateProcessSetResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return stackmonitoringsdk.CreateProcessSetResponse{}, nil
}

func (f *fakeProcessSetOCIClient) GetProcessSet(
	ctx context.Context,
	request stackmonitoringsdk.GetProcessSetRequest,
) (stackmonitoringsdk.GetProcessSetResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return stackmonitoringsdk.GetProcessSetResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "ProcessSet is missing")
}

func (f *fakeProcessSetOCIClient) ListProcessSets(
	ctx context.Context,
	request stackmonitoringsdk.ListProcessSetsRequest,
) (stackmonitoringsdk.ListProcessSetsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return stackmonitoringsdk.ListProcessSetsResponse{}, nil
}

func (f *fakeProcessSetOCIClient) UpdateProcessSet(
	ctx context.Context,
	request stackmonitoringsdk.UpdateProcessSetRequest,
) (stackmonitoringsdk.UpdateProcessSetResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return stackmonitoringsdk.UpdateProcessSetResponse{}, nil
}

func (f *fakeProcessSetOCIClient) DeleteProcessSet(
	ctx context.Context,
	request stackmonitoringsdk.DeleteProcessSetRequest,
) (stackmonitoringsdk.DeleteProcessSetResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return stackmonitoringsdk.DeleteProcessSetResponse{}, nil
}

func TestProcessSetRuntimeHooksConfigured(t *testing.T) {
	hooks := newProcessSetDefaultRuntimeHooks(stackmonitoringsdk.StackMonitoringClient{})
	applyProcessSetRuntimeHooks(&hooks)

	checks := []struct {
		name string
		ok   bool
	}{
		{name: "Semantics", ok: hooks.Semantics != nil},
		{name: "BuildCreateBody", ok: hooks.BuildCreateBody != nil},
		{name: "BuildUpdateBody", ok: hooks.BuildUpdateBody != nil},
		{name: "Identity.Resolve", ok: hooks.Identity.Resolve != nil},
		{name: "DeleteHooks.HandleError", ok: hooks.DeleteHooks.HandleError != nil},
		{name: "WrapGeneratedClient", ok: len(hooks.WrapGeneratedClient) > 0},
	}
	for _, check := range checks {
		if !check.ok {
			t.Fatalf("hooks.%s not configured", check.name)
		}
	}

	body, err := hooks.BuildCreateBody(context.Background(), makeProcessSetResource(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(stackmonitoringsdk.CreateProcessSetDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want CreateProcessSetDetails", body)
	}
	requireStringPtr(t, "CreateProcessSetDetails.DisplayName", details.DisplayName, testProcessSetDisplayName)
	requireProcessSetSpecification(t, details.Specification, makeProcessSetResource().Spec.Specification)
}

func TestProcessSetCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	resource := makeProcessSetResource()
	listCalls := 0
	getCalls := 0
	createCalled := false
	updateCalled := false
	client := newTestProcessSetClient(&fakeProcessSetOCIClient{
		listFn: func(_ context.Context, request stackmonitoringsdk.ListProcessSetsRequest) (stackmonitoringsdk.ListProcessSetsResponse, error) {
			listCalls++
			requireStringPtr(t, "ListProcessSetsRequest.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireStringPtr(t, "ListProcessSetsRequest.DisplayName", request.DisplayName, resource.Spec.DisplayName)
			if listCalls == 1 {
				if request.Page != nil {
					t.Fatalf("first ListProcessSetsRequest.Page = %q, want nil", *request.Page)
				}
				return stackmonitoringsdk.ListProcessSetsResponse{
					ProcessSetCollection: stackmonitoringsdk.ProcessSetCollection{
						Items: []stackmonitoringsdk.ProcessSetSummary{
							makeSDKProcessSetSummary("ocid1.stackmonitoringprocessset.oc1..other", stackmonitoringv1beta1.ProcessSetSpec{
								CompartmentId: testProcessSetCompartmentID,
								DisplayName:   "other-process-set",
								Specification: resource.Spec.Specification,
							}, stackmonitoringsdk.LifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			requireStringPtr(t, "second ListProcessSetsRequest.Page", request.Page, "page-2")
			return stackmonitoringsdk.ListProcessSetsResponse{
				ProcessSetCollection: stackmonitoringsdk.ProcessSetCollection{
					Items: []stackmonitoringsdk.ProcessSetSummary{
						makeSDKProcessSetSummary(testProcessSetID, resource.Spec, stackmonitoringsdk.LifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request stackmonitoringsdk.GetProcessSetRequest) (stackmonitoringsdk.GetProcessSetResponse, error) {
			getCalls++
			requireStringPtr(t, "GetProcessSetRequest.ProcessSetId", request.ProcessSetId, testProcessSetID)
			return stackmonitoringsdk.GetProcessSetResponse{
				ProcessSet: makeSDKProcessSet(testProcessSetID, resource.Spec, stackmonitoringsdk.LifecycleStateActive),
			}, nil
		},
		createFn: func(context.Context, stackmonitoringsdk.CreateProcessSetRequest) (stackmonitoringsdk.CreateProcessSetResponse, error) {
			createCalled = true
			return stackmonitoringsdk.CreateProcessSetResponse{}, nil
		},
		updateFn: func(context.Context, stackmonitoringsdk.UpdateProcessSetRequest) (stackmonitoringsdk.UpdateProcessSetResponse, error) {
			updateCalled = true
			return stackmonitoringsdk.UpdateProcessSetResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProcessSetRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if createCalled {
		t.Fatal("CreateProcessSet() called for existing ProcessSet")
	}
	if updateCalled {
		t.Fatal("UpdateProcessSet() called for matching ProcessSet")
	}
	if listCalls != 2 {
		t.Fatalf("ListProcessSets() calls = %d, want 2 paginated calls", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetProcessSet() calls = %d, want 1 live read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testProcessSetID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testProcessSetID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestProcessSetCreateRecordsOCIDRetryTokenRequestIDAndLifecycle(t *testing.T) {
	resource := makeProcessSetResource()
	createCalls := 0
	listCalls := 0
	client := newTestProcessSetClient(&fakeProcessSetOCIClient{
		listFn: func(_ context.Context, request stackmonitoringsdk.ListProcessSetsRequest) (stackmonitoringsdk.ListProcessSetsResponse, error) {
			listCalls++
			requireStringPtr(t, "ListProcessSetsRequest.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireStringPtr(t, "ListProcessSetsRequest.DisplayName", request.DisplayName, resource.Spec.DisplayName)
			return stackmonitoringsdk.ListProcessSetsResponse{}, nil
		},
		createFn: func(_ context.Context, request stackmonitoringsdk.CreateProcessSetRequest) (stackmonitoringsdk.CreateProcessSetResponse, error) {
			createCalls++
			requireProcessSetCreateRequest(t, request, resource)
			return stackmonitoringsdk.CreateProcessSetResponse{
				ProcessSet:   makeSDKProcessSet(testProcessSetID, resource.Spec, stackmonitoringsdk.LifecycleStateCreating),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request stackmonitoringsdk.GetProcessSetRequest) (stackmonitoringsdk.GetProcessSetResponse, error) {
			requireStringPtr(t, "GetProcessSetRequest.ProcessSetId", request.ProcessSetId, testProcessSetID)
			return stackmonitoringsdk.GetProcessSetResponse{
				ProcessSet: makeSDKProcessSet(testProcessSetID, resource.Spec, stackmonitoringsdk.LifecycleStateCreating),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProcessSetRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() shouldRequeue = false, want true for CREATING ProcessSet")
	}
	if createCalls != 1 {
		t.Fatalf("CreateProcessSet() calls = %d, want 1", createCalls)
	}
	if listCalls != 1 {
		t.Fatalf("ListProcessSets() calls = %d, want 1 pre-create lookup", listCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testProcessSetID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testProcessSetID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	requireProcessSetCreatePendingStatus(t, resource)
}

func TestProcessSetCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	resource := makeProcessSetResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProcessSetID)
	updateCalled := false
	client := newTestProcessSetClient(&fakeProcessSetOCIClient{
		getFn: func(_ context.Context, request stackmonitoringsdk.GetProcessSetRequest) (stackmonitoringsdk.GetProcessSetResponse, error) {
			requireStringPtr(t, "GetProcessSetRequest.ProcessSetId", request.ProcessSetId, testProcessSetID)
			return stackmonitoringsdk.GetProcessSetResponse{
				ProcessSet: makeSDKProcessSet(testProcessSetID, resource.Spec, stackmonitoringsdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, stackmonitoringsdk.UpdateProcessSetRequest) (stackmonitoringsdk.UpdateProcessSetResponse, error) {
			updateCalled = true
			return stackmonitoringsdk.UpdateProcessSetResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProcessSetRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateProcessSet() called during no-op reconcile")
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestProcessSetCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	resource := makeProcessSetResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProcessSetID)
	resource.Spec.DisplayName = "updated-process-set"
	resource.Spec.Specification.Items[0].ProcessLineRegexPattern = ".*updated.*"
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	currentSpec := makeProcessSetResource().Spec
	currentSpec.FreeformTags = map[string]string{"env": "test"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	getCalls := 0
	updateCalls := 0
	client := newTestProcessSetClient(&fakeProcessSetOCIClient{
		getFn: func(_ context.Context, request stackmonitoringsdk.GetProcessSetRequest) (stackmonitoringsdk.GetProcessSetResponse, error) {
			getCalls++
			requireStringPtr(t, "GetProcessSetRequest.ProcessSetId", request.ProcessSetId, testProcessSetID)
			if getCalls == 1 {
				return stackmonitoringsdk.GetProcessSetResponse{
					ProcessSet: makeSDKProcessSet(testProcessSetID, currentSpec, stackmonitoringsdk.LifecycleStateActive),
				}, nil
			}
			return stackmonitoringsdk.GetProcessSetResponse{
				ProcessSet: makeSDKProcessSet(testProcessSetID, resource.Spec, stackmonitoringsdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, request stackmonitoringsdk.UpdateProcessSetRequest) (stackmonitoringsdk.UpdateProcessSetResponse, error) {
			updateCalls++
			requireStringPtr(t, "UpdateProcessSetRequest.ProcessSetId", request.ProcessSetId, testProcessSetID)
			requireStringPtr(t, "UpdateProcessSetDetails.DisplayName", request.DisplayName, resource.Spec.DisplayName)
			requireProcessSetSpecification(t, request.Specification, resource.Spec.Specification)
			if len(request.FreeformTags) != 0 {
				t.Fatalf("UpdateProcessSetDetails.FreeformTags = %#v, want explicit empty map", request.FreeformTags)
			}
			if got := request.DefinedTags["Operations"]["CostCenter"]; got != "99" {
				t.Fatalf("UpdateProcessSetDetails.DefinedTags[Operations][CostCenter] = %v, want 99", got)
			}
			return stackmonitoringsdk.UpdateProcessSetResponse{
				ProcessSet:   makeSDKProcessSet(testProcessSetID, resource.Spec, stackmonitoringsdk.LifecycleStateActive),
				OpcRequestId: common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProcessSetRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateProcessSet() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetProcessSet() calls = %d, want current read and update follow-up", getCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestProcessSetCreateOrUpdateRejectsCreateOnlyCompartmentDriftBeforeUpdate(t *testing.T) {
	resource := makeProcessSetResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProcessSetID)
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..different"
	currentSpec := resource.Spec
	currentSpec.CompartmentId = testProcessSetCompartmentID
	updateCalled := false
	client := newTestProcessSetClient(&fakeProcessSetOCIClient{
		getFn: func(_ context.Context, request stackmonitoringsdk.GetProcessSetRequest) (stackmonitoringsdk.GetProcessSetResponse, error) {
			requireStringPtr(t, "GetProcessSetRequest.ProcessSetId", request.ProcessSetId, testProcessSetID)
			return stackmonitoringsdk.GetProcessSetResponse{
				ProcessSet: makeSDKProcessSet(testProcessSetID, currentSpec, stackmonitoringsdk.LifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, stackmonitoringsdk.UpdateProcessSetRequest) (stackmonitoringsdk.UpdateProcessSetResponse, error) {
			updateCalled = true
			return stackmonitoringsdk.UpdateProcessSetResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProcessSetRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateProcessSet() called after create-only compartmentId drift")
	}
	if !strings.Contains(err.Error(), "compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId force-new rejection", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestProcessSetDeleteKeepsFinalizerUntilReadConfirmsNotFound(t *testing.T) {
	resource := makeProcessSetResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProcessSetID)
	getCalls := 0
	deleteCalls := 0
	client := newTestProcessSetClient(&fakeProcessSetOCIClient{
		getFn: func(_ context.Context, request stackmonitoringsdk.GetProcessSetRequest) (stackmonitoringsdk.GetProcessSetResponse, error) {
			return processSetDeleteProgressionRead(t, resource, request, &getCalls)
		},
		deleteFn: func(_ context.Context, request stackmonitoringsdk.DeleteProcessSetRequest) (stackmonitoringsdk.DeleteProcessSetResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteProcessSetRequest.ProcessSetId", request.ProcessSetId, testProcessSetID)
			return stackmonitoringsdk.DeleteProcessSetResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first deleted = true, want false while OCI reports DELETING")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteProcessSet() calls after first delete = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireLastCondition(t, resource, shared.Terminating)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second deleted = false, want true after unambiguous NotFound")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteProcessSet() calls after confirmed delete = %d, want still 1", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestProcessSetDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	resource := makeProcessSetResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProcessSetID)
	client := newTestProcessSetClient(&fakeProcessSetOCIClient{
		getFn: func(_ context.Context, request stackmonitoringsdk.GetProcessSetRequest) (stackmonitoringsdk.GetProcessSetResponse, error) {
			requireStringPtr(t, "GetProcessSetRequest.ProcessSetId", request.ProcessSetId, testProcessSetID)
			return stackmonitoringsdk.GetProcessSetResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
		deleteFn: func(context.Context, stackmonitoringsdk.DeleteProcessSetRequest) (stackmonitoringsdk.DeleteProcessSetResponse, error) {
			t.Fatal("DeleteProcessSet() called after auth-shaped pre-delete read")
			return stackmonitoringsdk.DeleteProcessSetResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete read")
	}
	if !strings.Contains(err.Error(), "delete confirmation read returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestProcessSetDeleteTreatsAuthShapedDeleteNotFoundConservatively(t *testing.T) {
	resource := makeProcessSetResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProcessSetID)
	getCalls := 0
	client := newTestProcessSetClient(&fakeProcessSetOCIClient{
		getFn: func(_ context.Context, request stackmonitoringsdk.GetProcessSetRequest) (stackmonitoringsdk.GetProcessSetResponse, error) {
			getCalls++
			requireStringPtr(t, "GetProcessSetRequest.ProcessSetId", request.ProcessSetId, testProcessSetID)
			return stackmonitoringsdk.GetProcessSetResponse{
				ProcessSet: makeSDKProcessSet(testProcessSetID, resource.Spec, stackmonitoringsdk.LifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, request stackmonitoringsdk.DeleteProcessSetRequest) (stackmonitoringsdk.DeleteProcessSetResponse, error) {
			requireStringPtr(t, "DeleteProcessSetRequest.ProcessSetId", request.ProcessSetId, testProcessSetID)
			return stackmonitoringsdk.DeleteProcessSetResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped delete 404")
	}
	if !strings.Contains(err.Error(), "delete returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous delete classification", err)
	}
	if getCalls != 2 {
		t.Fatalf("GetProcessSet() calls = %d, want preflight and generated confirmation reads", getCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestProcessSetCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	resource := makeProcessSetResource()
	client := newTestProcessSetClient(&fakeProcessSetOCIClient{
		listFn: func(context.Context, stackmonitoringsdk.ListProcessSetsRequest) (stackmonitoringsdk.ListProcessSetsResponse, error) {
			return stackmonitoringsdk.ListProcessSetsResponse{}, nil
		},
		createFn: func(context.Context, stackmonitoringsdk.CreateProcessSetRequest) (stackmonitoringsdk.CreateProcessSetResponse, error) {
			return stackmonitoringsdk.CreateProcessSetResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeProcessSetRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI service error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func newTestProcessSetClient(client processSetOCIClient) ProcessSetServiceClient {
	return newProcessSetServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func processSetDeleteProgressionRead(
	t *testing.T,
	resource *stackmonitoringv1beta1.ProcessSet,
	request stackmonitoringsdk.GetProcessSetRequest,
	getCalls *int,
) (stackmonitoringsdk.GetProcessSetResponse, error) {
	t.Helper()
	*getCalls = *getCalls + 1
	requireStringPtr(t, "GetProcessSetRequest.ProcessSetId", request.ProcessSetId, testProcessSetID)
	switch *getCalls {
	case 1, 2:
		return stackmonitoringsdk.GetProcessSetResponse{
			ProcessSet: makeSDKProcessSet(testProcessSetID, resource.Spec, stackmonitoringsdk.LifecycleStateActive),
		}, nil
	case 3:
		return stackmonitoringsdk.GetProcessSetResponse{
			ProcessSet: makeSDKProcessSet(testProcessSetID, resource.Spec, stackmonitoringsdk.LifecycleStateDeleting),
		}, nil
	default:
		return stackmonitoringsdk.GetProcessSetResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "ProcessSet is gone")
	}
}

func makeProcessSetResource() *stackmonitoringv1beta1.ProcessSet {
	return &stackmonitoringv1beta1.ProcessSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testProcessSetDisplayName,
			Namespace: "default",
		},
		Spec: stackmonitoringv1beta1.ProcessSetSpec{
			CompartmentId: testProcessSetCompartmentID,
			DisplayName:   testProcessSetDisplayName,
			Specification: stackmonitoringv1beta1.ProcessSetSpecification{
				Items: []stackmonitoringv1beta1.ProcessSetSpecificationItem{{
					Label:                   "java",
					ProcessCommand:          "java",
					ProcessUser:             "opc",
					ProcessLineRegexPattern: ".*java.*",
				}},
			},
			FreeformTags: map[string]string{"env": "test"},
			DefinedTags:  map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeProcessSetRequest(resource *stackmonitoringv1beta1.ProcessSet) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func makeSDKProcessSet(
	id string,
	spec stackmonitoringv1beta1.ProcessSetSpec,
	state stackmonitoringsdk.LifecycleStateEnum,
) stackmonitoringsdk.ProcessSet {
	return stackmonitoringsdk.ProcessSet{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		LifecycleState: state,
		DisplayName:    common.String(spec.DisplayName),
		Specification:  processSetSpecification(spec.Specification),
		Revision:       common.String("1"),
		FreeformTags:   cloneProcessSetStringMap(spec.FreeformTags),
		DefinedTags:    processSetDefinedTags(spec.DefinedTags),
	}
}

func makeSDKProcessSetSummary(
	id string,
	spec stackmonitoringv1beta1.ProcessSetSpec,
	state stackmonitoringsdk.LifecycleStateEnum,
) stackmonitoringsdk.ProcessSetSummary {
	return stackmonitoringsdk.ProcessSetSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		LifecycleState: state,
		DisplayName:    common.String(spec.DisplayName),
		Specification:  processSetSpecification(spec.Specification),
		Revision:       common.String("1"),
		FreeformTags:   cloneProcessSetStringMap(spec.FreeformTags),
		DefinedTags:    processSetDefinedTags(spec.DefinedTags),
	}
}

func requireProcessSetCreateRequest(
	t *testing.T,
	request stackmonitoringsdk.CreateProcessSetRequest,
	resource *stackmonitoringv1beta1.ProcessSet,
) {
	t.Helper()
	requireStringPtr(t, "CreateProcessSetDetails.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "CreateProcessSetDetails.DisplayName", request.DisplayName, resource.Spec.DisplayName)
	requireProcessSetSpecification(t, request.Specification, resource.Spec.Specification)
	if request.FreeformTags["env"] != "test" {
		t.Fatalf("CreateProcessSetDetails.FreeformTags[env] = %q, want test", request.FreeformTags["env"])
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("CreateProcessSetDetails.DefinedTags[Operations][CostCenter] = %v, want 42", got)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateProcessSetRequest.OpcRetryToken is empty, want deterministic retry token")
	}
}

func requireProcessSetCreatePendingStatus(t *testing.T, resource *stackmonitoringv1beta1.ProcessSet) {
	t.Helper()
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want lifecycle create tracker")
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceLifecycle ||
		current.Phase != shared.OSOKAsyncPhaseCreate ||
		current.RawStatus != string(stackmonitoringsdk.LifecycleStateCreating) ||
		current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.status.async.current = %#v, want lifecycle create pending CREATING", current)
	}
}

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireProcessSetSpecification(
	t *testing.T,
	got *stackmonitoringsdk.ProcessSetSpecification,
	want stackmonitoringv1beta1.ProcessSetSpecification,
) {
	t.Helper()
	if got == nil {
		t.Fatal("ProcessSetSpecification = nil")
	}
	if want := processSetSpecification(want); !reflect.DeepEqual(got.Items, want.Items) {
		t.Fatalf("ProcessSetSpecification.Items = %#v, want %#v", got.Items, want.Items)
	}
}

func requireLastCondition(
	t *testing.T,
	resource *stackmonitoringv1beta1.ProcessSet,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}
