/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package operationsinsightswarehouse

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testOperationsInsightsWarehouseID             = "ocid1.opsiwarehouse.oc1..test"
	testOperationsInsightsWarehouseOtherID        = "ocid1.opsiwarehouse.oc1..other"
	testOperationsInsightsWarehouseCompartment    = "ocid1.compartment.oc1..test"
	testOperationsInsightsWarehouseOldCompartment = "ocid1.compartment.oc1..old"
	testOperationsInsightsWarehouseName           = "opsi-warehouse"
	testOperationsInsightsWarehouseUpdatedName    = "opsi-warehouse-updated"
	testOperationsInsightsWarehouseWorkRequest    = "ocid1.workrequest.oc1..opsiwarehouse"
	testOperationsInsightsWarehouseOpcRequestID   = "opc-opsiwarehouse-request"
)

type fakeOperationsInsightsWarehouseOCIClient struct {
	t *testing.T

	createFn         func(context.Context, opsisdk.CreateOperationsInsightsWarehouseRequest) (opsisdk.CreateOperationsInsightsWarehouseResponse, error)
	getFn            func(context.Context, opsisdk.GetOperationsInsightsWarehouseRequest) (opsisdk.GetOperationsInsightsWarehouseResponse, error)
	listFn           func(context.Context, opsisdk.ListOperationsInsightsWarehousesRequest) (opsisdk.ListOperationsInsightsWarehousesResponse, error)
	updateFn         func(context.Context, opsisdk.UpdateOperationsInsightsWarehouseRequest) (opsisdk.UpdateOperationsInsightsWarehouseResponse, error)
	deleteFn         func(context.Context, opsisdk.DeleteOperationsInsightsWarehouseRequest) (opsisdk.DeleteOperationsInsightsWarehouseResponse, error)
	getWorkRequestFn func(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

func (f *fakeOperationsInsightsWarehouseOCIClient) CreateOperationsInsightsWarehouse(
	ctx context.Context,
	request opsisdk.CreateOperationsInsightsWarehouseRequest,
) (opsisdk.CreateOperationsInsightsWarehouseResponse, error) {
	f.t.Helper()
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	f.t.Fatalf("CreateOperationsInsightsWarehouse() was called unexpectedly with %#v", request)
	return opsisdk.CreateOperationsInsightsWarehouseResponse{}, nil
}

func (f *fakeOperationsInsightsWarehouseOCIClient) GetOperationsInsightsWarehouse(
	ctx context.Context,
	request opsisdk.GetOperationsInsightsWarehouseRequest,
) (opsisdk.GetOperationsInsightsWarehouseResponse, error) {
	f.t.Helper()
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	f.t.Fatalf("GetOperationsInsightsWarehouse() was called unexpectedly with %#v", request)
	return opsisdk.GetOperationsInsightsWarehouseResponse{}, nil
}

func (f *fakeOperationsInsightsWarehouseOCIClient) ListOperationsInsightsWarehouses(
	ctx context.Context,
	request opsisdk.ListOperationsInsightsWarehousesRequest,
) (opsisdk.ListOperationsInsightsWarehousesResponse, error) {
	f.t.Helper()
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	f.t.Fatalf("ListOperationsInsightsWarehouses() was called unexpectedly with %#v", request)
	return opsisdk.ListOperationsInsightsWarehousesResponse{}, nil
}

func (f *fakeOperationsInsightsWarehouseOCIClient) UpdateOperationsInsightsWarehouse(
	ctx context.Context,
	request opsisdk.UpdateOperationsInsightsWarehouseRequest,
) (opsisdk.UpdateOperationsInsightsWarehouseResponse, error) {
	f.t.Helper()
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	f.t.Fatalf("UpdateOperationsInsightsWarehouse() was called unexpectedly with %#v", request)
	return opsisdk.UpdateOperationsInsightsWarehouseResponse{}, nil
}

func (f *fakeOperationsInsightsWarehouseOCIClient) DeleteOperationsInsightsWarehouse(
	ctx context.Context,
	request opsisdk.DeleteOperationsInsightsWarehouseRequest,
) (opsisdk.DeleteOperationsInsightsWarehouseResponse, error) {
	f.t.Helper()
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	f.t.Fatalf("DeleteOperationsInsightsWarehouse() was called unexpectedly with %#v", request)
	return opsisdk.DeleteOperationsInsightsWarehouseResponse{}, nil
}

func (f *fakeOperationsInsightsWarehouseOCIClient) GetWorkRequest(
	ctx context.Context,
	request opsisdk.GetWorkRequestRequest,
) (opsisdk.GetWorkRequestResponse, error) {
	f.t.Helper()
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, request)
	}
	f.t.Fatalf("GetWorkRequest() was called unexpectedly with %#v", request)
	return opsisdk.GetWorkRequestResponse{}, nil
}

func TestOperationsInsightsWarehouseRuntimeHooksEncodeReviewedContract(t *testing.T) {
	t.Parallel()

	client := &fakeOperationsInsightsWarehouseOCIClient{t: t}
	hooks := newOperationsInsightsWarehouseRuntimeHooksWithOCIClient(client)
	applyOperationsInsightsWarehouseRuntimeHooks(&hooks, client, nil, testOperationsInsightsWarehouseLog())

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed OperationsInsightsWarehouse semantics")
	}
	if got := hooks.Semantics.Async.Strategy; got != "workrequest" {
		t.Fatalf("hooks.Semantics.Async.Strategy = %q, want workrequest", got)
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want custom create body")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want custom update body")
	}
	if hooks.StatusHooks.ProjectStatus == nil {
		t.Fatal("hooks.StatusHooks.ProjectStatus = nil, want custom status projection")
	}
	if hooks.Async.GetWorkRequest == nil || hooks.Async.ResolvePhase == nil || hooks.Async.RecoverResourceID == nil {
		t.Fatal("async hooks are incomplete, want work-request polling hooks")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("hooks.DeleteHooks.HandleError = nil, want conservative delete error handling")
	}
}

func TestOperationsInsightsWarehouseCreateBodyOmitsOptionalZeroAndProjectsTags(t *testing.T) {
	t.Parallel()

	client := &fakeOperationsInsightsWarehouseOCIClient{t: t}
	hooks := newOperationsInsightsWarehouseRuntimeHooksWithOCIClient(client)
	applyOperationsInsightsWarehouseRuntimeHooks(&hooks, client, nil, testOperationsInsightsWarehouseLog())

	resource := newTestOperationsInsightsWarehouseResource()
	resource.Spec.StorageAllocatedInGBs = 0
	resource.Spec.FreeformTags = map[string]string{"env": "test"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	body, err := hooks.BuildCreateBody(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("hooks.BuildCreateBody() error = %v", err)
	}
	createDetails, ok := body.(opsisdk.CreateOperationsInsightsWarehouseDetails)
	if !ok {
		t.Fatalf("hooks.BuildCreateBody() body type = %T, want opsisdk.CreateOperationsInsightsWarehouseDetails", body)
	}
	if createDetails.StorageAllocatedInGBs != nil {
		t.Fatalf("StorageAllocatedInGBs = %v, want nil when spec omits optional storage", *createDetails.StorageAllocatedInGBs)
	}
	if got := createDetails.FreeformTags["env"]; got != "test" {
		t.Fatalf("FreeformTags[env] = %q, want test", got)
	}
	if got := createDetails.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("DefinedTags[Operations][CostCenter] = %#v, want 42", got)
	}
}

func TestOperationsInsightsWarehouseServiceClientBindsFromSecondListPageWithoutCreating(t *testing.T) {
	t.Parallel()

	resource := newTestOperationsInsightsWarehouseResource()
	var listRequests []opsisdk.ListOperationsInsightsWarehousesRequest
	client := newTestOperationsInsightsWarehouseServiceClient(t, &fakeOperationsInsightsWarehouseOCIClient{
		t: t,
		listFn: func(_ context.Context, request opsisdk.ListOperationsInsightsWarehousesRequest) (opsisdk.ListOperationsInsightsWarehousesResponse, error) {
			listRequests = append(listRequests, request)
			requireStringPtr(t, "ListOperationsInsightsWarehousesRequest.CompartmentId", request.CompartmentId, testOperationsInsightsWarehouseCompartment)
			requireStringPtr(t, "ListOperationsInsightsWarehousesRequest.DisplayName", request.DisplayName, testOperationsInsightsWarehouseName)
			if request.Page == nil {
				otherSpec := resource.Spec
				otherSpec.DisplayName = "other-warehouse"
				return opsisdk.ListOperationsInsightsWarehousesResponse{
					OperationsInsightsWarehouseSummaryCollection: opsisdk.OperationsInsightsWarehouseSummaryCollection{
						Items: []opsisdk.OperationsInsightsWarehouseSummary{
							makeSDKOperationsInsightsWarehouseSummary(testOperationsInsightsWarehouseOtherID, otherSpec, opsisdk.OperationsInsightsWarehouseLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			requireStringPtr(t, "ListOperationsInsightsWarehousesRequest.Page", request.Page, "page-2")
			return opsisdk.ListOperationsInsightsWarehousesResponse{
				OperationsInsightsWarehouseSummaryCollection: opsisdk.OperationsInsightsWarehouseSummaryCollection{
					Items: []opsisdk.OperationsInsightsWarehouseSummary{
						makeSDKOperationsInsightsWarehouseSummary(testOperationsInsightsWarehouseID, resource.Spec, opsisdk.OperationsInsightsWarehouseLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request opsisdk.GetOperationsInsightsWarehouseRequest) (opsisdk.GetOperationsInsightsWarehouseResponse, error) {
			requireStringPtr(t, "GetOperationsInsightsWarehouseRequest.OperationsInsightsWarehouseId", request.OperationsInsightsWarehouseId, testOperationsInsightsWarehouseID)
			return opsisdk.GetOperationsInsightsWarehouseResponse{
				OperationsInsightsWarehouse: makeSDKOperationsInsightsWarehouse(testOperationsInsightsWarehouseID, resource.Spec, opsisdk.OperationsInsightsWarehouseLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseRequest())
	requireCreateOrUpdateSuccess(t, response, err)
	if got := len(listRequests); got != 2 {
		t.Fatalf("ListOperationsInsightsWarehouses() calls = %d, want 2", got)
	}
	if resource.Status.Id != testOperationsInsightsWarehouseID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testOperationsInsightsWarehouseID)
	}
	if resource.Status.OsokStatus.Ocid != shared.OCID(testOperationsInsightsWarehouseID) {
		t.Fatalf("status.status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, testOperationsInsightsWarehouseID)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after active bind", resource.Status.OsokStatus.Async.Current)
	}
}

func TestOperationsInsightsWarehouseServiceClientCreatesAndTracksWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newTestOperationsInsightsWarehouseResource()
	var createRequest opsisdk.CreateOperationsInsightsWarehouseRequest
	client := newTestOperationsInsightsWarehouseServiceClient(t, &fakeOperationsInsightsWarehouseOCIClient{
		t: t,
		listFn: func(_ context.Context, request opsisdk.ListOperationsInsightsWarehousesRequest) (opsisdk.ListOperationsInsightsWarehousesResponse, error) {
			requireStringPtr(t, "ListOperationsInsightsWarehousesRequest.CompartmentId", request.CompartmentId, testOperationsInsightsWarehouseCompartment)
			requireStringPtr(t, "ListOperationsInsightsWarehousesRequest.DisplayName", request.DisplayName, testOperationsInsightsWarehouseName)
			return opsisdk.ListOperationsInsightsWarehousesResponse{}, nil
		},
		createFn: func(_ context.Context, request opsisdk.CreateOperationsInsightsWarehouseRequest) (opsisdk.CreateOperationsInsightsWarehouseResponse, error) {
			createRequest = request
			return opsisdk.CreateOperationsInsightsWarehouseResponse{
				OperationsInsightsWarehouse: makeSDKOperationsInsightsWarehouse(testOperationsInsightsWarehouseID, resource.Spec, opsisdk.OperationsInsightsWarehouseLifecycleStateCreating),
				OpcWorkRequestId:            common.String(testOperationsInsightsWarehouseWorkRequest),
				OpcRequestId:                common.String(testOperationsInsightsWarehouseOpcRequestID),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testOperationsInsightsWarehouseWorkRequest)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: makeOperationsInsightsWarehouseWorkRequest(
					testOperationsInsightsWarehouseWorkRequest,
					opsisdk.OperationTypeCreateOpsiWarehouse,
					opsisdk.OperationStatusInProgress,
					opsisdk.ActionTypeCreated,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseRequest())
	requireCreateOrUpdateSuccess(t, response, err)
	requireRequeue(t, response, true)
	requireStringPtr(t, "CreateOperationsInsightsWarehouseRequest.OpcRetryToken", createRequest.OpcRetryToken, string(resource.UID))
	if createRequest.CpuAllocated == nil || *createRequest.CpuAllocated != resource.Spec.CpuAllocated {
		t.Fatalf("CreateOperationsInsightsWarehouseRequest.CpuAllocated = %v, want %v", createRequest.CpuAllocated, resource.Spec.CpuAllocated)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, testOperationsInsightsWarehouseWorkRequest)
	if got := resource.Status.OsokStatus.OpcRequestID; got != testOperationsInsightsWarehouseOpcRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, testOperationsInsightsWarehouseOpcRequestID)
	}
}

func TestOperationsInsightsWarehouseServiceClientSkipsUpdateWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := newTestOperationsInsightsWarehouseResource()
	recordOperationsInsightsWarehouseID(resource, testOperationsInsightsWarehouseID)
	client := newTestOperationsInsightsWarehouseServiceClient(t, &fakeOperationsInsightsWarehouseOCIClient{
		t: t,
		getFn: func(_ context.Context, request opsisdk.GetOperationsInsightsWarehouseRequest) (opsisdk.GetOperationsInsightsWarehouseResponse, error) {
			requireStringPtr(t, "GetOperationsInsightsWarehouseRequest.OperationsInsightsWarehouseId", request.OperationsInsightsWarehouseId, testOperationsInsightsWarehouseID)
			return opsisdk.GetOperationsInsightsWarehouseResponse{
				OperationsInsightsWarehouse: makeSDKOperationsInsightsWarehouse(testOperationsInsightsWarehouseID, resource.Spec, opsisdk.OperationsInsightsWarehouseLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseRequest())
	requireCreateOrUpdateSuccess(t, response, err)
	requireRequeue(t, response, false)
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after no-op reconcile", resource.Status.OsokStatus.Async.Current)
	}
}

func TestOperationsInsightsWarehouseServiceClientProjectsStatusAndConvertsTagValues(t *testing.T) {
	t.Parallel()

	resource := newTestOperationsInsightsWarehouseResource()
	recordOperationsInsightsWarehouseID(resource, testOperationsInsightsWarehouseID)
	resource.Status.OsokStatus.OpcRequestID = "previous-opc-request"
	resource.Status.OsokStatus.Message = "preserve existing status"

	current := makeSDKOperationsInsightsWarehouse(testOperationsInsightsWarehouseID, resource.Spec, opsisdk.OperationsInsightsWarehouseLifecycleStateActive)
	current.DefinedTags = map[string]map[string]interface{}{
		"Operations": {
			"CostCenter": 42,
			"Enabled":    true,
		},
	}
	current.SystemTags = map[string]map[string]interface{}{
		"orcl-cloud": {
			"free-tier-retained": false,
		},
	}
	client := newTestOperationsInsightsWarehouseServiceClient(t, &fakeOperationsInsightsWarehouseOCIClient{
		t: t,
		getFn: func(_ context.Context, request opsisdk.GetOperationsInsightsWarehouseRequest) (opsisdk.GetOperationsInsightsWarehouseResponse, error) {
			requireStringPtr(t, "GetOperationsInsightsWarehouseRequest.OperationsInsightsWarehouseId", request.OperationsInsightsWarehouseId, testOperationsInsightsWarehouseID)
			return opsisdk.GetOperationsInsightsWarehouseResponse{OperationsInsightsWarehouse: current}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseRequest())
	requireCreateOrUpdateSuccess(t, response, err)
	requireRequeue(t, response, false)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "previous-opc-request" {
		t.Fatalf("status.status.opcRequestId = %q, want previous value preserved", got)
	}
	if got := resource.Status.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("status.definedTags Operations.CostCenter = %q, want 42", got)
	}
	if got := resource.Status.DefinedTags["Operations"]["Enabled"]; got != "true" {
		t.Fatalf("status.definedTags Operations.Enabled = %q, want true", got)
	}
	if got := resource.Status.SystemTags["orcl-cloud"]["free-tier-retained"]; got != "false" {
		t.Fatalf("status.systemTags orcl-cloud.free-tier-retained = %q, want false", got)
	}
}

func TestOperationsInsightsWarehouseServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := newTestOperationsInsightsWarehouseResource()
	recordOperationsInsightsWarehouseID(resource, testOperationsInsightsWarehouseID)
	resource.Spec.DisplayName = testOperationsInsightsWarehouseUpdatedName
	resource.Spec.CpuAllocated = 4
	resource.Spec.StorageAllocatedInGBs = 512
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}

	currentSpec := resource.Spec
	currentSpec.DisplayName = testOperationsInsightsWarehouseName
	currentSpec.CpuAllocated = 2
	currentSpec.StorageAllocatedInGBs = 256
	currentSpec.FreeformTags = map[string]string{"env": "dev"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}

	var updateRequest opsisdk.UpdateOperationsInsightsWarehouseRequest
	client := newTestOperationsInsightsWarehouseServiceClient(t, &fakeOperationsInsightsWarehouseOCIClient{
		t: t,
		getFn: func(_ context.Context, request opsisdk.GetOperationsInsightsWarehouseRequest) (opsisdk.GetOperationsInsightsWarehouseResponse, error) {
			requireStringPtr(t, "GetOperationsInsightsWarehouseRequest.OperationsInsightsWarehouseId", request.OperationsInsightsWarehouseId, testOperationsInsightsWarehouseID)
			return opsisdk.GetOperationsInsightsWarehouseResponse{
				OperationsInsightsWarehouse: makeSDKOperationsInsightsWarehouse(testOperationsInsightsWarehouseID, currentSpec, opsisdk.OperationsInsightsWarehouseLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, request opsisdk.UpdateOperationsInsightsWarehouseRequest) (opsisdk.UpdateOperationsInsightsWarehouseResponse, error) {
			updateRequest = request
			return opsisdk.UpdateOperationsInsightsWarehouseResponse{
				OpcWorkRequestId: common.String(testOperationsInsightsWarehouseWorkRequest),
				OpcRequestId:     common.String(testOperationsInsightsWarehouseOpcRequestID),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testOperationsInsightsWarehouseWorkRequest)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: makeOperationsInsightsWarehouseWorkRequest(
					testOperationsInsightsWarehouseWorkRequest,
					opsisdk.OperationTypeUpdateOpsiWarehouse,
					opsisdk.OperationStatusInProgress,
					opsisdk.ActionTypeUpdated,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseRequest())
	requireCreateOrUpdateSuccess(t, response, err)
	requireRequeue(t, response, true)
	requireStringPtr(t, "UpdateOperationsInsightsWarehouseRequest.OperationsInsightsWarehouseId", updateRequest.OperationsInsightsWarehouseId, testOperationsInsightsWarehouseID)
	requireStringPtr(t, "UpdateOperationsInsightsWarehouseRequest.DisplayName", updateRequest.DisplayName, testOperationsInsightsWarehouseUpdatedName)
	if updateRequest.CpuAllocated == nil || *updateRequest.CpuAllocated != resource.Spec.CpuAllocated {
		t.Fatalf("UpdateOperationsInsightsWarehouseRequest.CpuAllocated = %v, want %v", updateRequest.CpuAllocated, resource.Spec.CpuAllocated)
	}
	if updateRequest.StorageAllocatedInGBs == nil || *updateRequest.StorageAllocatedInGBs != resource.Spec.StorageAllocatedInGBs {
		t.Fatalf("UpdateOperationsInsightsWarehouseRequest.StorageAllocatedInGBs = %v, want %v", updateRequest.StorageAllocatedInGBs, resource.Spec.StorageAllocatedInGBs)
	}
	if got := updateRequest.FreeformTags["env"]; got != "prod" {
		t.Fatalf("UpdateOperationsInsightsWarehouseRequest.FreeformTags[env] = %q, want prod", got)
	}
	if got := updateRequest.DefinedTags["Operations"]["CostCenter"]; got != "99" {
		t.Fatalf("UpdateOperationsInsightsWarehouseRequest.DefinedTags[Operations][CostCenter] = %#v, want 99", got)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending, testOperationsInsightsWarehouseWorkRequest)
	if got := resource.Status.OsokStatus.OpcRequestID; got != testOperationsInsightsWarehouseOpcRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, testOperationsInsightsWarehouseOpcRequestID)
	}
}

func TestOperationsInsightsWarehouseServiceClientRejectsCompartmentDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newTestOperationsInsightsWarehouseResource()
	recordOperationsInsightsWarehouseID(resource, testOperationsInsightsWarehouseID)
	currentSpec := resource.Spec
	currentSpec.CompartmentId = testOperationsInsightsWarehouseOldCompartment
	client := newTestOperationsInsightsWarehouseServiceClient(t, &fakeOperationsInsightsWarehouseOCIClient{
		t: t,
		getFn: func(_ context.Context, request opsisdk.GetOperationsInsightsWarehouseRequest) (opsisdk.GetOperationsInsightsWarehouseResponse, error) {
			requireStringPtr(t, "GetOperationsInsightsWarehouseRequest.OperationsInsightsWarehouseId", request.OperationsInsightsWarehouseId, testOperationsInsightsWarehouseID)
			return opsisdk.GetOperationsInsightsWarehouseResponse{
				OperationsInsightsWarehouse: makeSDKOperationsInsightsWarehouse(testOperationsInsightsWarehouseID, currentSpec, opsisdk.OperationsInsightsWarehouseLifecycleStateActive),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, testOperationsInsightsWarehouseRequest())
	if err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId force-new rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = true, want false after immutable drift rejection")
	}
}

func TestOperationsInsightsWarehouseServiceClientDeleteTracksPendingWorkRequest(t *testing.T) {
	t.Parallel()

	resource := newTestOperationsInsightsWarehouseResource()
	recordOperationsInsightsWarehouseID(resource, testOperationsInsightsWarehouseID)
	getCalls := 0
	client := newTestOperationsInsightsWarehouseServiceClient(t, &fakeOperationsInsightsWarehouseOCIClient{
		t: t,
		getFn: func(_ context.Context, request opsisdk.GetOperationsInsightsWarehouseRequest) (opsisdk.GetOperationsInsightsWarehouseResponse, error) {
			getCalls++
			requireStringPtr(t, "GetOperationsInsightsWarehouseRequest.OperationsInsightsWarehouseId", request.OperationsInsightsWarehouseId, testOperationsInsightsWarehouseID)
			return opsisdk.GetOperationsInsightsWarehouseResponse{
				OperationsInsightsWarehouse: makeSDKOperationsInsightsWarehouse(testOperationsInsightsWarehouseID, resource.Spec, opsisdk.OperationsInsightsWarehouseLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, request opsisdk.DeleteOperationsInsightsWarehouseRequest) (opsisdk.DeleteOperationsInsightsWarehouseResponse, error) {
			requireStringPtr(t, "DeleteOperationsInsightsWarehouseRequest.OperationsInsightsWarehouseId", request.OperationsInsightsWarehouseId, testOperationsInsightsWarehouseID)
			return opsisdk.DeleteOperationsInsightsWarehouseResponse{
				OpcWorkRequestId: common.String(testOperationsInsightsWarehouseWorkRequest),
				OpcRequestId:     common.String(testOperationsInsightsWarehouseOpcRequestID),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testOperationsInsightsWarehouseWorkRequest)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: makeOperationsInsightsWarehouseWorkRequest(
					testOperationsInsightsWarehouseWorkRequest,
					opsisdk.OperationTypeDeleteOpsiWarehouse,
					opsisdk.OperationStatusInProgress,
					opsisdk.ActionTypeDeleted,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete work request is pending")
	}
	if getCalls < 2 {
		t.Fatalf("GetOperationsInsightsWarehouse() calls = %d, want preflight plus already-pending confirmation", getCalls)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, testOperationsInsightsWarehouseWorkRequest)
	if got := resource.Status.OsokStatus.OpcRequestID; got != testOperationsInsightsWarehouseOpcRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, testOperationsInsightsWarehouseOpcRequestID)
	}
}

func TestOperationsInsightsWarehouseDeleteWorkRequestRejectsAuthShaped404Confirmation(t *testing.T) {
	t.Parallel()

	resource := newTestOperationsInsightsWarehouseResource()
	recordOperationsInsightsWarehouseID(resource, testOperationsInsightsWarehouseID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   testOperationsInsightsWarehouseWorkRequest,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	client := newTestOperationsInsightsWarehouseServiceClient(t, &fakeOperationsInsightsWarehouseOCIClient{
		t: t,
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testOperationsInsightsWarehouseWorkRequest)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: makeOperationsInsightsWarehouseWorkRequest(
					testOperationsInsightsWarehouseWorkRequest,
					opsisdk.OperationTypeDeleteOpsiWarehouse,
					opsisdk.OperationStatusSucceeded,
					opsisdk.ActionTypeDeleted,
				),
			}, nil
		},
		getFn: func(_ context.Context, request opsisdk.GetOperationsInsightsWarehouseRequest) (opsisdk.GetOperationsInsightsWarehouseResponse, error) {
			requireStringPtr(t, "GetOperationsInsightsWarehouseRequest.OperationsInsightsWarehouseId", request.OperationsInsightsWarehouseId, testOperationsInsightsWarehouseID)
			return opsisdk.GetOperationsInsightsWarehouseResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped 404 rejection", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous 404")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id from OCI error", got)
	}
}

func TestOperationsInsightsWarehouseDeleteWorkRequestConfirmsUnambiguousNotFound(t *testing.T) {
	t.Parallel()

	resource := newTestOperationsInsightsWarehouseResource()
	recordOperationsInsightsWarehouseID(resource, testOperationsInsightsWarehouseID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   testOperationsInsightsWarehouseWorkRequest,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	client := newTestOperationsInsightsWarehouseServiceClient(t, &fakeOperationsInsightsWarehouseOCIClient{
		t: t,
		getWorkRequestFn: func(_ context.Context, request opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error) {
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, testOperationsInsightsWarehouseWorkRequest)
			return opsisdk.GetWorkRequestResponse{
				WorkRequest: makeOperationsInsightsWarehouseWorkRequest(
					testOperationsInsightsWarehouseWorkRequest,
					opsisdk.OperationTypeDeleteOpsiWarehouse,
					opsisdk.OperationStatusSucceeded,
					opsisdk.ActionTypeDeleted,
				),
			}, nil
		},
		getFn: func(_ context.Context, request opsisdk.GetOperationsInsightsWarehouseRequest) (opsisdk.GetOperationsInsightsWarehouseResponse, error) {
			requireStringPtr(t, "GetOperationsInsightsWarehouseRequest.OperationsInsightsWarehouseId", request.OperationsInsightsWarehouseId, testOperationsInsightsWarehouseID)
			return opsisdk.GetOperationsInsightsWarehouseResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not found")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after confirmed delete", resource.Status.OsokStatus.Async.Current)
	}
}

func newTestOperationsInsightsWarehouseServiceClient(
	t *testing.T,
	client *fakeOperationsInsightsWarehouseOCIClient,
) OperationsInsightsWarehouseServiceClient {
	t.Helper()
	if client == nil {
		client = &fakeOperationsInsightsWarehouseOCIClient{t: t}
	}
	return newOperationsInsightsWarehouseServiceClientWithOCIClient(testOperationsInsightsWarehouseLog(), client)
}

func newTestOperationsInsightsWarehouseResource() *opsiv1beta1.OperationsInsightsWarehouse {
	return &opsiv1beta1.OperationsInsightsWarehouse{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "opsi-warehouse",
			Namespace: "default",
			UID:       "uid-opsi-warehouse",
		},
		Spec: opsiv1beta1.OperationsInsightsWarehouseSpec{
			CompartmentId:         testOperationsInsightsWarehouseCompartment,
			DisplayName:           testOperationsInsightsWarehouseName,
			CpuAllocated:          2,
			ComputeModel:          string(opsisdk.OperationsInsightsWarehouseComputeModelOcpu),
			StorageAllocatedInGBs: 256,
		},
	}
}

func testOperationsInsightsWarehouseRequest() ctrl.Request {
	return ctrl.Request{NamespacedName: k8stypes.NamespacedName{Namespace: "default", Name: "opsi-warehouse"}}
}

func testOperationsInsightsWarehouseLog() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{}
}

func makeSDKOperationsInsightsWarehouse(
	id string,
	spec opsiv1beta1.OperationsInsightsWarehouseSpec,
	state opsisdk.OperationsInsightsWarehouseLifecycleStateEnum,
) opsisdk.OperationsInsightsWarehouse {
	return opsisdk.OperationsInsightsWarehouse{
		Id:                    common.String(id),
		CompartmentId:         common.String(spec.CompartmentId),
		DisplayName:           common.String(spec.DisplayName),
		CpuAllocated:          common.Float64(spec.CpuAllocated),
		LifecycleState:        state,
		ComputeModel:          opsisdk.OperationsInsightsWarehouseComputeModelEnum(spec.ComputeModel),
		StorageAllocatedInGBs: common.Float64(spec.StorageAllocatedInGBs),
		FreeformTags:          cloneOperationsInsightsWarehouseStringMap(spec.FreeformTags),
		DefinedTags:           operationsInsightsWarehouseDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func makeSDKOperationsInsightsWarehouseSummary(
	id string,
	spec opsiv1beta1.OperationsInsightsWarehouseSpec,
	state opsisdk.OperationsInsightsWarehouseLifecycleStateEnum,
) opsisdk.OperationsInsightsWarehouseSummary {
	return opsisdk.OperationsInsightsWarehouseSummary{
		Id:                    common.String(id),
		CompartmentId:         common.String(spec.CompartmentId),
		DisplayName:           common.String(spec.DisplayName),
		CpuAllocated:          common.Float64(spec.CpuAllocated),
		LifecycleState:        state,
		ComputeModel:          opsisdk.OperationsInsightsWarehouseComputeModelEnum(spec.ComputeModel),
		StorageAllocatedInGBs: common.Float64(spec.StorageAllocatedInGBs),
		FreeformTags:          cloneOperationsInsightsWarehouseStringMap(spec.FreeformTags),
		DefinedTags:           operationsInsightsWarehouseDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func makeOperationsInsightsWarehouseWorkRequest(
	id string,
	operationType opsisdk.OperationTypeEnum,
	status opsisdk.OperationStatusEnum,
	action opsisdk.ActionTypeEnum,
) opsisdk.WorkRequest {
	return opsisdk.WorkRequest{
		OperationType:   operationType,
		Status:          status,
		Id:              common.String(id),
		CompartmentId:   common.String(testOperationsInsightsWarehouseCompartment),
		PercentComplete: common.Float32(50),
		Resources: []opsisdk.WorkRequestResource{
			{
				EntityType: common.String("opsiWarehouse"),
				ActionType: action,
				Identifier: common.String(testOperationsInsightsWarehouseID),
			},
		},
	}
}

func requireCreateOrUpdateSuccess(t *testing.T, response servicemanager.OSOKResponse, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = false, want true")
	}
}

func requireRequeue(t *testing.T, response servicemanager.OSOKResponse, want bool) {
	t.Helper()
	if response.ShouldRequeue != want {
		t.Fatalf("ShouldRequeue = %t, want %t", response.ShouldRequeue, want)
	}
}

func requireAsyncCurrent(
	t *testing.T,
	resource *opsiv1beta1.OperationsInsightsWarehouse,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	workRequestID string,
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
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
}

func requireStringPtr(t *testing.T, field string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", field, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", field, *got, want)
	}
}
