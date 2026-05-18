/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package maskingcolumn

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testMaskingPolicyID       = "ocid1.maskingpolicy.oc1..example"
	testMaskingPolicyOtherID  = "ocid1.maskingpolicy.oc1..other"
	testMaskingColumnKey      = "42"
	testMaskingColumnOtherKey = "43"
	testMaskingColumnUID      = "masking-column-uid"
	testWorkRequestID         = "ocid1.workrequest.oc1..maskingcolumn"
)

type fakeMaskingColumnOCIClient struct {
	createFn         func(context.Context, datasafesdk.CreateMaskingColumnRequest) (datasafesdk.CreateMaskingColumnResponse, error)
	getFn            func(context.Context, datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error)
	listFn           func(context.Context, datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error)
	updateFn         func(context.Context, datasafesdk.UpdateMaskingColumnRequest) (datasafesdk.UpdateMaskingColumnResponse, error)
	deleteFn         func(context.Context, datasafesdk.DeleteMaskingColumnRequest) (datasafesdk.DeleteMaskingColumnResponse, error)
	getWorkRequestFn func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
}

func (f *fakeMaskingColumnOCIClient) CreateMaskingColumn(
	ctx context.Context,
	request datasafesdk.CreateMaskingColumnRequest,
) (datasafesdk.CreateMaskingColumnResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return datasafesdk.CreateMaskingColumnResponse{}, nil
}

func (f *fakeMaskingColumnOCIClient) GetMaskingColumn(
	ctx context.Context,
	request datasafesdk.GetMaskingColumnRequest,
) (datasafesdk.GetMaskingColumnResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return datasafesdk.GetMaskingColumnResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "masking column not found")
}

func (f *fakeMaskingColumnOCIClient) ListMaskingColumns(
	ctx context.Context,
	request datasafesdk.ListMaskingColumnsRequest,
) (datasafesdk.ListMaskingColumnsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return datasafesdk.ListMaskingColumnsResponse{}, nil
}

func (f *fakeMaskingColumnOCIClient) UpdateMaskingColumn(
	ctx context.Context,
	request datasafesdk.UpdateMaskingColumnRequest,
) (datasafesdk.UpdateMaskingColumnResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return datasafesdk.UpdateMaskingColumnResponse{}, nil
}

func (f *fakeMaskingColumnOCIClient) DeleteMaskingColumn(
	ctx context.Context,
	request datasafesdk.DeleteMaskingColumnRequest,
) (datasafesdk.DeleteMaskingColumnResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return datasafesdk.DeleteMaskingColumnResponse{OpcRequestId: common.String("opc-delete")}, nil
}

func (f *fakeMaskingColumnOCIClient) GetWorkRequest(
	ctx context.Context,
	request datasafesdk.GetWorkRequestRequest,
) (datasafesdk.GetWorkRequestResponse, error) {
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, request)
	}
	return datasafesdk.GetWorkRequestResponse{}, nil
}

func testMaskingColumnClient(fake *fakeMaskingColumnOCIClient) MaskingColumnServiceClient {
	if fake == nil {
		fake = &fakeMaskingColumnOCIClient{}
	}
	manager := &MaskingColumnServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}}
	hooks := newMaskingColumnRuntimeHooksWithOCIClient(fake)
	applyMaskingColumnRuntimeHooks(&hooks, fake, nil)
	delegate := defaultMaskingColumnServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.MaskingColumn](
			buildMaskingColumnGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapMaskingColumnGeneratedClient(hooks, delegate)
}

func newMaskingColumnRuntimeHooksWithOCIClient(client maskingColumnOCIClient) MaskingColumnRuntimeHooks {
	hooks := newMaskingColumnDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	hooks.Create.Call = func(ctx context.Context, request datasafesdk.CreateMaskingColumnRequest) (datasafesdk.CreateMaskingColumnResponse, error) {
		return client.CreateMaskingColumn(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error) {
		return client.GetMaskingColumn(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error) {
		return client.ListMaskingColumns(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request datasafesdk.UpdateMaskingColumnRequest) (datasafesdk.UpdateMaskingColumnResponse, error) {
		return client.UpdateMaskingColumn(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request datasafesdk.DeleteMaskingColumnRequest) (datasafesdk.DeleteMaskingColumnResponse, error) {
		return client.DeleteMaskingColumn(ctx, request)
	}
	return hooks
}

func TestMaskingColumnRuntimeSemantics(t *testing.T) {
	semantics := maskingColumnRuntimeSemantics()
	if semantics == nil {
		t.Fatal("maskingColumnRuntimeSemantics() = nil")
	}
	requireMaskingColumnAsyncSemantics(t, semantics.Async)
	requireMaskingColumnFinalizerSemantics(t, semantics.FinalizerPolicy)
	requireMaskingColumnMutationSemantics(t, semantics.Mutation)
	requireMaskingColumnListSemantics(t, semantics.List)
}

func requireMaskingColumnAsyncSemantics(t *testing.T, async *generatedruntime.AsyncSemantics) {
	t.Helper()
	if async == nil {
		t.Fatal("async semantics = nil, want generated workrequest")
	}
	if async.Strategy != "workrequest" || async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generated workrequest", async)
	}
}

func requireMaskingColumnFinalizerSemantics(t *testing.T, finalizerPolicy string) {
	t.Helper()
	if finalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("finalizer policy = %q, want retain-until-confirmed-delete", finalizerPolicy)
	}
}

func requireMaskingColumnMutationSemantics(t *testing.T, mutation generatedruntime.MutationSemantics) {
	t.Helper()
	if !containsMaskingColumnTestString(mutation.ForceNew, "maskingPolicyId") {
		t.Fatalf("mutation force-new fields = %#v, want maskingPolicyId", mutation.ForceNew)
	}
	if !containsMaskingColumnTestString(mutation.ForceNew, "schemaName") {
		t.Fatalf("mutation force-new fields = %#v, want schemaName", mutation.ForceNew)
	}
	if !containsMaskingColumnTestString(mutation.Mutable, "isMaskingEnabled") {
		t.Fatalf("mutation mutable fields = %#v, want isMaskingEnabled", mutation.Mutable)
	}
}

func requireMaskingColumnListSemantics(t *testing.T, list *generatedruntime.ListSemantics) {
	t.Helper()
	if list == nil || len(list.MatchFields) != 4 {
		t.Fatalf("list semantics = %#v, want four match fields", list)
	}
}

func TestMaskingColumnCreateRequiresMaskingPolicyAnnotationBeforeOCICalls(t *testing.T) {
	resource := makeMaskingColumnResource()
	resource.Annotations = nil
	fake := &fakeMaskingColumnOCIClient{
		createFn: func(context.Context, datasafesdk.CreateMaskingColumnRequest) (datasafesdk.CreateMaskingColumnResponse, error) {
			t.Fatal("CreateMaskingColumn should not be called without parent annotation")
			return datasafesdk.CreateMaskingColumnResponse{}, nil
		},
		listFn: func(context.Context, datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error) {
			t.Fatal("ListMaskingColumns should not be called without parent annotation")
			return datasafesdk.ListMaskingColumnsResponse{}, nil
		},
	}

	response, err := testMaskingColumnClient(fake).CreateOrUpdate(context.Background(), resource, maskingColumnRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing annotation rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), MaskingColumnMaskingPolicyIDAnnotation) {
		t.Fatalf("error = %q, want masking policy annotation context", err.Error())
	}
}

func TestMaskingColumnCreateStartsWorkRequestAndPreservesExplicitFalse(t *testing.T) {
	resource := makeMaskingColumnResource()
	var createCalls int
	var listCalls int
	var capturedCreate datasafesdk.CreateMaskingColumnRequest
	fake := &fakeMaskingColumnOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error) {
			listCalls++
			requireMaskingColumnListPolicy(t, request, testMaskingPolicyID)
			return datasafesdk.ListMaskingColumnsResponse{}, nil
		},
		createFn: func(_ context.Context, request datasafesdk.CreateMaskingColumnRequest) (datasafesdk.CreateMaskingColumnResponse, error) {
			createCalls++
			capturedCreate = request
			return datasafesdk.CreateMaskingColumnResponse{
				OpcWorkRequestId: common.String(testWorkRequestID),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, request datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			if got := maskingColumnStringValue(request.WorkRequestId); got != testWorkRequestID {
				t.Fatalf("workRequestId = %q, want %q", got, testWorkRequestID)
			}
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: makeMaskingColumnWorkRequest(datasafesdk.WorkRequestStatusAccepted, datasafesdk.WorkRequestOperationTypeCreateMaskingColumn, nil),
			}, nil
		},
	}

	response, err := testMaskingColumnClient(fake).CreateOrUpdate(context.Background(), resource, maskingColumnRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false")
	}
	if listCalls != 1 || createCalls != 1 {
		t.Fatalf("listCalls = %d, createCalls = %d, want 1/1", listCalls, createCalls)
	}
	requireMaskingColumnCreateRequest(t, capturedCreate)
	requireMaskingColumnOpcRequestID(t, resource, "opc-create")
	requireMaskingColumnAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending)
}

func TestMaskingColumnBindsExistingBeforeCreateAcrossPages(t *testing.T) {
	resource := makeMaskingColumnResource()
	var listPages []string
	fake := &fakeMaskingColumnOCIClient{
		createFn: func(context.Context, datasafesdk.CreateMaskingColumnRequest) (datasafesdk.CreateMaskingColumnResponse, error) {
			t.Fatal("CreateMaskingColumn should not be called when list discovers existing column")
			return datasafesdk.CreateMaskingColumnResponse{}, nil
		},
		listFn: func(_ context.Context, request datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error) {
			requireMaskingColumnListPolicy(t, request, testMaskingPolicyID)
			listPages = append(listPages, maskingColumnStringValue(request.Page))
			if request.Page == nil {
				return datasafesdk.ListMaskingColumnsResponse{
					MaskingColumnCollection: datasafesdk.MaskingColumnCollection{
						Items: []datasafesdk.MaskingColumnSummary{makeSDKMaskingColumnSummary(testMaskingColumnOtherKey, testMaskingPolicyID, "APP", "CUSTOMERS", "EMAIL", datasafesdk.MaskingColumnLifecycleStateActive, false)},
					},
					OpcNextPage: common.String("next"),
				}, nil
			}
			return datasafesdk.ListMaskingColumnsResponse{
				MaskingColumnCollection: datasafesdk.MaskingColumnCollection{
					Items: []datasafesdk.MaskingColumnSummary{makeSDKMaskingColumnSummary(testMaskingColumnKey, testMaskingPolicyID, "APP", "CUSTOMERS", "SSN", datasafesdk.MaskingColumnLifecycleStateActive, false)},
				},
			}, nil
		},
	}

	response, err := testMaskingColumnClient(fake).CreateOrUpdate(context.Background(), resource, maskingColumnRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false")
	}
	if got, want := strings.Join(listPages, ","), ",next"; got != want {
		t.Fatalf("list pages = %q, want %q", got, want)
	}
	requireMaskingColumnProjectedStatus(t, resource, testMaskingColumnKey, testMaskingPolicyID, datasafesdk.MaskingColumnLifecycleStateActive)
}

func TestMaskingColumnCreateWorkRequestSuccessRecordsRecoveredKey(t *testing.T) {
	resource := makeMaskingColumnResource()
	resource.Status.MaskingPolicyId = testMaskingPolicyID
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   testWorkRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	var getCalls int
	fake := &fakeMaskingColumnOCIClient{
		createFn: func(context.Context, datasafesdk.CreateMaskingColumnRequest) (datasafesdk.CreateMaskingColumnResponse, error) {
			t.Fatal("CreateMaskingColumn should not be called while create work request is tracked")
			return datasafesdk.CreateMaskingColumnResponse{}, nil
		},
		listFn: func(context.Context, datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error) {
			t.Fatal("ListMaskingColumns should not be called when work request exposes the masking column key")
			return datasafesdk.ListMaskingColumnsResponse{}, nil
		},
		getWorkRequestFn: func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: makeMaskingColumnWorkRequest(
					datasafesdk.WorkRequestStatusSucceeded,
					datasafesdk.WorkRequestOperationTypeCreateMaskingColumn,
					[]datasafesdk.WorkRequestResource{
						{
							Identifier: common.String(testMaskingColumnKey),
							ActionType: datasafesdk.WorkRequestResourceActionTypeCreated,
						},
					},
				),
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error) {
			getCalls++
			requireMaskingColumnGetRequest(t, request, testMaskingPolicyID, testMaskingColumnKey)
			return datasafesdk.GetMaskingColumnResponse{
				MaskingColumn: makeSDKMaskingColumn(testMaskingColumnKey, testMaskingPolicyID, datasafesdk.MaskingColumnLifecycleStateActive, false),
			}, nil
		},
	}

	response, err := testMaskingColumnClient(fake).CreateOrUpdate(context.Background(), resource, maskingColumnRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false")
	}
	if getCalls != 1 {
		t.Fatalf("getCalls = %d, want 1", getCalls)
	}
	requireMaskingColumnProjectedStatus(t, resource, testMaskingColumnKey, testMaskingPolicyID, datasafesdk.MaskingColumnLifecycleStateActive)
	if current := resource.Status.OsokStatus.Async.Current; current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil after successful work request readback", current)
	}
}

func TestMaskingColumnNoopReconcileUsesRecordedKeyAndSkipsUpdate(t *testing.T) {
	resource := makeMaskingColumnResource()
	markMaskingColumnTracked(resource, testMaskingColumnKey, testMaskingPolicyID)
	var getCalls int
	fake := &fakeMaskingColumnOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error) {
			getCalls++
			requireMaskingColumnGetRequest(t, request, testMaskingPolicyID, testMaskingColumnKey)
			return datasafesdk.GetMaskingColumnResponse{
				MaskingColumn: makeSDKMaskingColumn(testMaskingColumnKey, testMaskingPolicyID, datasafesdk.MaskingColumnLifecycleStateActive, false),
			}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateMaskingColumnRequest) (datasafesdk.UpdateMaskingColumnResponse, error) {
			t.Fatal("UpdateMaskingColumn should not be called when desired and observed state match")
			return datasafesdk.UpdateMaskingColumnResponse{}, nil
		},
	}

	response, err := testMaskingColumnClient(fake).CreateOrUpdate(context.Background(), resource, maskingColumnRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false")
	}
	if getCalls != 1 {
		t.Fatalf("getCalls = %d, want 1", getCalls)
	}
	requireMaskingColumnProjectedStatus(t, resource, testMaskingColumnKey, testMaskingPolicyID, datasafesdk.MaskingColumnLifecycleStateActive)
}

func TestMaskingColumnMutableUpdateStartsWorkRequest(t *testing.T) {
	resource := makeMaskingColumnResource()
	markMaskingColumnTracked(resource, testMaskingColumnKey, testMaskingPolicyID)
	var capturedUpdate datasafesdk.UpdateMaskingColumnRequest
	fake := &fakeMaskingColumnOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error) {
			requireMaskingColumnGetRequest(t, request, testMaskingPolicyID, testMaskingColumnKey)
			current := makeSDKMaskingColumn(testMaskingColumnKey, testMaskingPolicyID, datasafesdk.MaskingColumnLifecycleStateActive, true)
			current.MaskingColumnGroup = common.String("old-group")
			return datasafesdk.GetMaskingColumnResponse{MaskingColumn: current}, nil
		},
		updateFn: func(_ context.Context, request datasafesdk.UpdateMaskingColumnRequest) (datasafesdk.UpdateMaskingColumnResponse, error) {
			capturedUpdate = request
			return datasafesdk.UpdateMaskingColumnResponse{
				OpcWorkRequestId: common.String(testWorkRequestID),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
		getWorkRequestFn: func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: makeMaskingColumnWorkRequest(datasafesdk.WorkRequestStatusAccepted, datasafesdk.WorkRequestOperationTypeUpdateMaskingColumn, nil),
			}, nil
		},
	}

	response, err := testMaskingColumnClient(fake).CreateOrUpdate(context.Background(), resource, maskingColumnRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false")
	}
	requireMaskingColumnUpdateRequest(t, capturedUpdate)
	requireMaskingColumnOpcRequestID(t, resource, "opc-update")
	requireMaskingColumnAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending)
}

func TestMaskingColumnRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := makeMaskingColumnResource()
	markMaskingColumnTracked(resource, testMaskingColumnKey, testMaskingPolicyID)
	fake := &fakeMaskingColumnOCIClient{
		getFn: func(context.Context, datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error) {
			current := makeSDKMaskingColumn(testMaskingColumnKey, testMaskingPolicyID, datasafesdk.MaskingColumnLifecycleStateActive, false)
			current.SchemaName = common.String("OTHER")
			return datasafesdk.GetMaskingColumnResponse{MaskingColumn: current}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateMaskingColumnRequest) (datasafesdk.UpdateMaskingColumnResponse, error) {
			t.Fatal("UpdateMaskingColumn should not be called for create-only drift")
			return datasafesdk.UpdateMaskingColumnResponse{}, nil
		},
	}

	response, err := testMaskingColumnClient(fake).CreateOrUpdate(context.Background(), resource, maskingColumnRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), "schemaName") {
		t.Fatalf("error = %q, want schemaName create-only drift context", err.Error())
	}
}

func TestMaskingColumnRejectsParentAnnotationDriftBeforeOCI(t *testing.T) {
	resource := makeMaskingColumnResource()
	markMaskingColumnTracked(resource, testMaskingColumnKey, testMaskingPolicyOtherID)
	fake := &fakeMaskingColumnOCIClient{
		getFn: func(context.Context, datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error) {
			t.Fatal("GetMaskingColumn should not be called after parent annotation drift")
			return datasafesdk.GetMaskingColumnResponse{}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateMaskingColumnRequest) (datasafesdk.UpdateMaskingColumnResponse, error) {
			t.Fatal("UpdateMaskingColumn should not be called after parent annotation drift")
			return datasafesdk.UpdateMaskingColumnResponse{}, nil
		},
	}

	_, err := testMaskingColumnClient(fake).CreateOrUpdate(context.Background(), resource, maskingColumnRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want parent annotation drift rejection")
	}
	if !strings.Contains(err.Error(), "annotation") || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("error = %q, want annotation drift context", err.Error())
	}
}

func TestMaskingColumnDeleteRetainsFinalizerWhenReadbackStillActive(t *testing.T) {
	resource := makeMaskingColumnResource()
	markMaskingColumnTracked(resource, testMaskingColumnKey, testMaskingPolicyID)
	var getCalls int
	var deleteCalls int
	fake := &fakeMaskingColumnOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error) {
			getCalls++
			requireMaskingColumnGetRequest(t, request, testMaskingPolicyID, testMaskingColumnKey)
			return datasafesdk.GetMaskingColumnResponse{
				MaskingColumn: makeSDKMaskingColumn(testMaskingColumnKey, testMaskingPolicyID, datasafesdk.MaskingColumnLifecycleStateActive, false),
			}, nil
		},
		deleteFn: func(_ context.Context, request datasafesdk.DeleteMaskingColumnRequest) (datasafesdk.DeleteMaskingColumnResponse, error) {
			deleteCalls++
			requireMaskingColumnDeleteRequest(t, request, testMaskingPolicyID, testMaskingColumnKey)
			return datasafesdk.DeleteMaskingColumnResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := testMaskingColumnClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while readback is ACTIVE")
	}
	if getCalls != 2 || deleteCalls != 1 {
		t.Fatalf("getCalls = %d, deleteCalls = %d, want 2/1", getCalls, deleteCalls)
	}
	requireMaskingColumnOpcRequestID(t, resource, "opc-delete")
	requireMaskingColumnAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
}

func TestMaskingColumnDeleteBlocksAuthShapedConfirmRead(t *testing.T) {
	resource := makeMaskingColumnResource()
	markMaskingColumnTracked(resource, testMaskingColumnKey, testMaskingPolicyID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	authErr.OpcRequestID = "opc-auth-read"
	fake := &fakeMaskingColumnOCIClient{
		getFn: func(context.Context, datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error) {
			return datasafesdk.GetMaskingColumnResponse{}, authErr
		},
		deleteFn: func(context.Context, datasafesdk.DeleteMaskingColumnRequest) (datasafesdk.DeleteMaskingColumnResponse, error) {
			t.Fatal("DeleteMaskingColumn should not be called after auth-shaped confirm read")
			return datasafesdk.DeleteMaskingColumnResponse{}, nil
		},
	}

	deleted, err := testMaskingColumnClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not-found rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("error = %q, want ambiguous NotAuthorizedOrNotFound context", err.Error())
	}
	requireMaskingColumnOpcRequestID(t, resource, "opc-auth-read")
}

type maskingColumnPendingWriteDeleteCase struct {
	name      string
	phase     shared.OSOKAsyncPhase
	operation datasafesdk.WorkRequestOperationTypeEnum
	setup     func(*testing.T, *datasafev1beta1.MaskingColumn)
}

func TestMaskingColumnDeleteRetainsFinalizerWhileWriteWorkRequestPending(t *testing.T) {
	tests := []maskingColumnPendingWriteDeleteCase{
		{
			name:      "create",
			phase:     shared.OSOKAsyncPhaseCreate,
			operation: datasafesdk.WorkRequestOperationTypeCreateMaskingColumn,
			setup: func(t *testing.T, resource *datasafev1beta1.MaskingColumn) {
				t.Helper()
				markMaskingColumnSyntheticTracked(t, resource)
			},
		},
		{
			name:      "update",
			phase:     shared.OSOKAsyncPhaseUpdate,
			operation: datasafesdk.WorkRequestOperationTypeUpdateMaskingColumn,
			setup: func(_ *testing.T, resource *datasafev1beta1.MaskingColumn) {
				markMaskingColumnTracked(resource, testMaskingColumnKey, testMaskingPolicyID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runMaskingColumnPendingWriteDeleteCase(t, tt)
		})
	}
}

func runMaskingColumnPendingWriteDeleteCase(t *testing.T, tt maskingColumnPendingWriteDeleteCase) {
	t.Helper()
	resource := makeMaskingColumnResource()
	tt.setup(t, resource)
	markMaskingColumnPendingWrite(resource, tt.phase)
	var getWorkRequestCalls int
	fake := pendingMaskingColumnWriteDeleteFake(t, tt.operation, &getWorkRequestCalls)

	deleted, err := testMaskingColumnClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while write work request is pending")
	}
	if getWorkRequestCalls != 1 {
		t.Fatalf("GetWorkRequest calls = %d, want 1", getWorkRequestCalls)
	}
	requireMaskingColumnAsyncCurrent(t, resource, tt.phase, shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.Async.Current.WorkRequestID; got != testWorkRequestID {
		t.Fatalf("status.status.async.current.workRequestId = %q, want %q", got, testWorkRequestID)
	}
}

func markMaskingColumnPendingWrite(resource *datasafev1beta1.MaskingColumn, phase shared.OSOKAsyncPhase) {
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   testWorkRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
}

func pendingMaskingColumnWriteDeleteFake(
	t *testing.T,
	operation datasafesdk.WorkRequestOperationTypeEnum,
	getWorkRequestCalls *int,
) *fakeMaskingColumnOCIClient {
	t.Helper()
	return &fakeMaskingColumnOCIClient{
		getWorkRequestFn: func(_ context.Context, request datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			(*getWorkRequestCalls)++
			if got := maskingColumnStringValue(request.WorkRequestId); got != testWorkRequestID {
				t.Fatalf("workRequestId = %q, want %q", got, testWorkRequestID)
			}
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: makeMaskingColumnWorkRequest(datasafesdk.WorkRequestStatusAccepted, operation, nil),
			}, nil
		},
		getFn: func(context.Context, datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error) {
			t.Fatal("GetMaskingColumn should not be called while write work request is pending before delete")
			return datasafesdk.GetMaskingColumnResponse{}, nil
		},
		listFn: func(context.Context, datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error) {
			t.Fatal("ListMaskingColumns should not be called while write work request is pending before delete")
			return datasafesdk.ListMaskingColumnsResponse{}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteMaskingColumnRequest) (datasafesdk.DeleteMaskingColumnResponse, error) {
			t.Fatal("DeleteMaskingColumn should not be called while write work request is pending")
			return datasafesdk.DeleteMaskingColumnResponse{}, nil
		},
	}
}

func TestMaskingColumnDeleteUsesResolvedRealKeyForSyntheticTrackedKey(t *testing.T) {
	resource := makeMaskingColumnResource()
	markMaskingColumnSyntheticTracked(t, resource)
	var listCalls int
	var getCalls int
	var deleteCalls int
	fake := &fakeMaskingColumnOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error) {
			listCalls++
			requireMaskingColumnListPolicy(t, request, testMaskingPolicyID)
			return datasafesdk.ListMaskingColumnsResponse{
				MaskingColumnCollection: datasafesdk.MaskingColumnCollection{
					Items: []datasafesdk.MaskingColumnSummary{
						makeSDKMaskingColumnSummary(testMaskingColumnKey, testMaskingPolicyID, "APP", "CUSTOMERS", "SSN", datasafesdk.MaskingColumnLifecycleStateActive, false),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error) {
			getCalls++
			requireMaskingColumnGetRequest(t, request, testMaskingPolicyID, testMaskingColumnKey)
			return datasafesdk.GetMaskingColumnResponse{
				MaskingColumn: makeSDKMaskingColumn(testMaskingColumnKey, testMaskingPolicyID, datasafesdk.MaskingColumnLifecycleStateActive, false),
			}, nil
		},
		deleteFn: func(_ context.Context, request datasafesdk.DeleteMaskingColumnRequest) (datasafesdk.DeleteMaskingColumnResponse, error) {
			deleteCalls++
			requireMaskingColumnDeleteRequest(t, request, testMaskingPolicyID, testMaskingColumnKey)
			return datasafesdk.DeleteMaskingColumnResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := testMaskingColumnClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while readback remains ACTIVE")
	}
	if listCalls != 1 || getCalls != 2 || deleteCalls != 1 {
		t.Fatalf("listCalls = %d, getCalls = %d, deleteCalls = %d, want 1/2/1", listCalls, getCalls, deleteCalls)
	}
	requireMaskingColumnProjectedStatus(t, resource, testMaskingColumnKey, testMaskingPolicyID, datasafesdk.MaskingColumnLifecycleStateActive)
	requireMaskingColumnOpcRequestID(t, resource, "opc-delete")
	requireMaskingColumnAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
}

func TestMaskingColumnDeleteUsesResolvedRealKeyWithoutTrackedKey(t *testing.T) {
	resource := makeMaskingColumnResource()
	var listCalls int
	var getCalls int
	var deleteCalls int
	fake := &fakeMaskingColumnOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error) {
			listCalls++
			requireMaskingColumnListPolicy(t, request, testMaskingPolicyID)
			return datasafesdk.ListMaskingColumnsResponse{
				MaskingColumnCollection: datasafesdk.MaskingColumnCollection{
					Items: []datasafesdk.MaskingColumnSummary{
						makeSDKMaskingColumnSummary(testMaskingColumnKey, testMaskingPolicyID, "APP", "CUSTOMERS", "SSN", datasafesdk.MaskingColumnLifecycleStateActive, false),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error) {
			getCalls++
			requireMaskingColumnGetRequest(t, request, testMaskingPolicyID, testMaskingColumnKey)
			return datasafesdk.GetMaskingColumnResponse{
				MaskingColumn: makeSDKMaskingColumn(testMaskingColumnKey, testMaskingPolicyID, datasafesdk.MaskingColumnLifecycleStateActive, false),
			}, nil
		},
		deleteFn: func(_ context.Context, request datasafesdk.DeleteMaskingColumnRequest) (datasafesdk.DeleteMaskingColumnResponse, error) {
			deleteCalls++
			requireMaskingColumnDeleteRequest(t, request, testMaskingPolicyID, testMaskingColumnKey)
			return datasafesdk.DeleteMaskingColumnResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := testMaskingColumnClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while readback remains ACTIVE")
	}
	if listCalls != 1 || getCalls != 2 || deleteCalls != 1 {
		t.Fatalf("listCalls = %d, getCalls = %d, deleteCalls = %d, want 1/2/1", listCalls, getCalls, deleteCalls)
	}
	requireMaskingColumnProjectedStatus(t, resource, testMaskingColumnKey, testMaskingPolicyID, datasafesdk.MaskingColumnLifecycleStateActive)
	requireMaskingColumnOpcRequestID(t, resource, "opc-delete")
	requireMaskingColumnAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
}

func TestMaskingColumnDeleteRetainsFinalizerWhenSyntheticKeyCannotResolve(t *testing.T) {
	resource := makeMaskingColumnResource()
	markMaskingColumnSyntheticTracked(t, resource)
	var listCalls int
	fake := &fakeMaskingColumnOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListMaskingColumnsRequest) (datasafesdk.ListMaskingColumnsResponse, error) {
			listCalls++
			requireMaskingColumnListPolicy(t, request, testMaskingPolicyID)
			return datasafesdk.ListMaskingColumnsResponse{}, nil
		},
		getFn: func(context.Context, datasafesdk.GetMaskingColumnRequest) (datasafesdk.GetMaskingColumnResponse, error) {
			t.Fatal("GetMaskingColumn should not be called when synthetic key is unresolved")
			return datasafesdk.GetMaskingColumnResponse{}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteMaskingColumnRequest) (datasafesdk.DeleteMaskingColumnResponse, error) {
			t.Fatal("DeleteMaskingColumn should not be called when synthetic key is unresolved")
			return datasafesdk.DeleteMaskingColumnResponse{}, nil
		},
	}

	deleted, err := testMaskingColumnClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false until synthetic key resolves")
	}
	if listCalls != 1 {
		t.Fatalf("ListMaskingColumns calls = %d, want 1", listCalls)
	}
	requireMaskingColumnAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
}

func makeMaskingColumnResource() *datasafev1beta1.MaskingColumn {
	return &datasafev1beta1.MaskingColumn{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "masking-column-alpha",
			Namespace: "default",
			UID:       types.UID(testMaskingColumnUID),
			Annotations: map[string]string{
				MaskingColumnMaskingPolicyIDAnnotation: testMaskingPolicyID,
			},
		},
		Spec: datasafev1beta1.MaskingColumnSpec{
			SchemaName:         "APP",
			ObjectName:         "CUSTOMERS",
			ColumnName:         "SSN",
			ObjectType:         string(datasafesdk.ObjectTypeTable),
			MaskingColumnGroup: "group-a",
			SensitiveTypeId:    "ocid1.sensitivetype.oc1..example",
			IsMaskingEnabled:   false,
			MaskingFormats: []datasafev1beta1.MaskingColumnMaskingFormat{
				{
					Description: "fixed value",
					FormatEntries: []datasafev1beta1.MaskingColumnMaskingFormatFormatEntry{
						{
							Type:        string(datasafesdk.FormatEntryTypeFixedString),
							FixedString: "MASKED",
						},
					},
				},
			},
		},
	}
}

func makeSDKMaskingColumn(
	key string,
	policyID string,
	state datasafesdk.MaskingColumnLifecycleStateEnum,
	enabled bool,
) datasafesdk.MaskingColumn {
	return datasafesdk.MaskingColumn{
		Key:                common.String(key),
		MaskingPolicyId:    common.String(policyID),
		LifecycleState:     state,
		SchemaName:         common.String("APP"),
		ObjectName:         common.String("CUSTOMERS"),
		ColumnName:         common.String("SSN"),
		ObjectType:         datasafesdk.ObjectTypeTable,
		MaskingColumnGroup: common.String("group-a"),
		SensitiveTypeId:    common.String("ocid1.sensitivetype.oc1..example"),
		IsMaskingEnabled:   common.Bool(enabled),
		MaskingFormats:     []datasafesdk.MaskingFormat{testMaskingColumnSDKFormat()},
	}
}

func makeSDKMaskingColumnSummary(
	key string,
	policyID string,
	schemaName string,
	objectName string,
	columnName string,
	state datasafesdk.MaskingColumnLifecycleStateEnum,
	enabled bool,
) datasafesdk.MaskingColumnSummary {
	return datasafesdk.MaskingColumnSummary{
		Key:                common.String(key),
		MaskingPolicyId:    common.String(policyID),
		LifecycleState:     state,
		SchemaName:         common.String(schemaName),
		ObjectName:         common.String(objectName),
		ColumnName:         common.String(columnName),
		ObjectType:         datasafesdk.ObjectTypeTable,
		MaskingColumnGroup: common.String("group-a"),
		SensitiveTypeId:    common.String("ocid1.sensitivetype.oc1..example"),
		IsMaskingEnabled:   common.Bool(enabled),
		MaskingFormats:     []datasafesdk.MaskingFormat{testMaskingColumnSDKFormat()},
	}
}

func testMaskingColumnSDKFormat() datasafesdk.MaskingFormat {
	return datasafesdk.MaskingFormat{
		Description: common.String("fixed value"),
		FormatEntries: []datasafesdk.FormatEntry{
			datasafesdk.FixedStringFormatEntry{FixedString: common.String("MASKED")},
		},
	}
}

func makeMaskingColumnWorkRequest(
	status datasafesdk.WorkRequestStatusEnum,
	operation datasafesdk.WorkRequestOperationTypeEnum,
	resources []datasafesdk.WorkRequestResource,
) datasafesdk.WorkRequest {
	percent := float32(25)
	return datasafesdk.WorkRequest{
		Id:              common.String(testWorkRequestID),
		Status:          status,
		OperationType:   operation,
		PercentComplete: &percent,
		Resources:       resources,
	}
}

func markMaskingColumnTracked(resource *datasafev1beta1.MaskingColumn, key string, policyID string) {
	resource.Status.Key = key
	resource.Status.MaskingPolicyId = policyID
	resource.Status.OsokStatus.Ocid = shared.OCID(key)
	resource.Status.SchemaName = resource.Spec.SchemaName
	resource.Status.ObjectName = resource.Spec.ObjectName
	resource.Status.ColumnName = resource.Spec.ColumnName
}

func markMaskingColumnSyntheticTracked(t *testing.T, resource *datasafev1beta1.MaskingColumn) {
	t.Helper()
	identity, err := resolveMaskingColumnIdentity(resource)
	if err != nil {
		t.Fatalf("resolveMaskingColumnIdentity() error = %v", err)
	}
	resource.Status.MaskingPolicyId = testMaskingPolicyID
	resource.Status.OsokStatus.Ocid = shared.OCID(syntheticMaskingColumnKey(identity.(maskingColumnIdentity)))
	resource.Status.SchemaName = resource.Spec.SchemaName
	resource.Status.ObjectName = resource.Spec.ObjectName
	resource.Status.ColumnName = resource.Spec.ColumnName
}

func maskingColumnRequest(resource *datasafev1beta1.MaskingColumn) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func requireMaskingColumnCreateRequest(t *testing.T, request datasafesdk.CreateMaskingColumnRequest) {
	t.Helper()
	if got := maskingColumnStringValue(request.MaskingPolicyId); got != testMaskingPolicyID {
		t.Fatalf("create maskingPolicyId = %q, want %q", got, testMaskingPolicyID)
	}
	if got := maskingColumnStringValue(request.OpcRetryToken); got != testMaskingColumnUID {
		t.Fatalf("create opcRetryToken = %q, want %q", got, testMaskingColumnUID)
	}
	if got := maskingColumnStringValue(request.SchemaName); got != "APP" {
		t.Fatalf("create schemaName = %q, want APP", got)
	}
	if request.IsMaskingEnabled == nil || *request.IsMaskingEnabled {
		t.Fatalf("create isMaskingEnabled = %#v, want explicit false", request.IsMaskingEnabled)
	}
	if len(request.MaskingFormats) != 1 || len(request.MaskingFormats[0].FormatEntries) != 1 {
		t.Fatalf("create maskingFormats = %#v, want one fixed-string format", request.MaskingFormats)
	}
	entry, ok := request.MaskingFormats[0].FormatEntries[0].(datasafesdk.FixedStringFormatEntry)
	if !ok {
		t.Fatalf("create masking format entry type = %T, want FixedStringFormatEntry", request.MaskingFormats[0].FormatEntries[0])
	}
	if got := maskingColumnStringValue(entry.FixedString); got != "MASKED" {
		t.Fatalf("create fixedString = %q, want MASKED", got)
	}
}

func requireMaskingColumnUpdateRequest(t *testing.T, request datasafesdk.UpdateMaskingColumnRequest) {
	t.Helper()
	requireMaskingColumnUpdatePath(t, request, testMaskingPolicyID, testMaskingColumnKey)
	if request.IsMaskingEnabled == nil || *request.IsMaskingEnabled {
		t.Fatalf("update isMaskingEnabled = %#v, want explicit false", request.IsMaskingEnabled)
	}
	if got := maskingColumnStringValue(request.MaskingColumnGroup); got != "group-a" {
		t.Fatalf("update maskingColumnGroup = %q, want group-a", got)
	}
}

func requireMaskingColumnUpdatePath(
	t *testing.T,
	request datasafesdk.UpdateMaskingColumnRequest,
	wantPolicyID string,
	wantKey string,
) {
	t.Helper()
	if got := maskingColumnStringValue(request.MaskingPolicyId); got != wantPolicyID {
		t.Fatalf("update maskingPolicyId = %q, want %q", got, wantPolicyID)
	}
	if got := maskingColumnStringValue(request.MaskingColumnKey); got != wantKey {
		t.Fatalf("update maskingColumnKey = %q, want %q", got, wantKey)
	}
}

func requireMaskingColumnGetRequest(
	t *testing.T,
	request datasafesdk.GetMaskingColumnRequest,
	wantPolicyID string,
	wantKey string,
) {
	t.Helper()
	if got := maskingColumnStringValue(request.MaskingPolicyId); got != wantPolicyID {
		t.Fatalf("get maskingPolicyId = %q, want %q", got, wantPolicyID)
	}
	if got := maskingColumnStringValue(request.MaskingColumnKey); got != wantKey {
		t.Fatalf("get maskingColumnKey = %q, want %q", got, wantKey)
	}
}

func requireMaskingColumnDeleteRequest(
	t *testing.T,
	request datasafesdk.DeleteMaskingColumnRequest,
	wantPolicyID string,
	wantKey string,
) {
	t.Helper()
	if got := maskingColumnStringValue(request.MaskingPolicyId); got != wantPolicyID {
		t.Fatalf("delete maskingPolicyId = %q, want %q", got, wantPolicyID)
	}
	if got := maskingColumnStringValue(request.MaskingColumnKey); got != wantKey {
		t.Fatalf("delete maskingColumnKey = %q, want %q", got, wantKey)
	}
}

func requireMaskingColumnListPolicy(
	t *testing.T,
	request datasafesdk.ListMaskingColumnsRequest,
	wantPolicyID string,
) {
	t.Helper()
	if got := maskingColumnStringValue(request.MaskingPolicyId); got != wantPolicyID {
		t.Fatalf("list maskingPolicyId = %q, want %q", got, wantPolicyID)
	}
}

func requireMaskingColumnProjectedStatus(
	t *testing.T,
	resource *datasafev1beta1.MaskingColumn,
	wantKey string,
	wantPolicyID string,
	wantState datasafesdk.MaskingColumnLifecycleStateEnum,
) {
	t.Helper()
	if got := resource.Status.Key; got != wantKey {
		t.Fatalf("status.key = %q, want %q", got, wantKey)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != wantKey {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantKey)
	}
	if got := resource.Status.MaskingPolicyId; got != wantPolicyID {
		t.Fatalf("status.maskingPolicyId = %q, want %q", got, wantPolicyID)
	}
	if got := resource.Status.LifecycleState; got != string(wantState) {
		t.Fatalf("status.lifecycleState = %q, want %q", got, wantState)
	}
}

func requireMaskingColumnOpcRequestID(
	t *testing.T,
	resource *datasafev1beta1.MaskingColumn,
	want string,
) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func requireMaskingColumnAsyncCurrent(
	t *testing.T,
	resource *datasafev1beta1.MaskingColumn,
	wantPhase shared.OSOKAsyncPhase,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatalf("status.status.async.current = nil, want %s %s", wantPhase, wantClass)
	}
	if current.Phase != wantPhase || current.NormalizedClass != wantClass {
		t.Fatalf("status.status.async.current = %#v, want phase %s class %s", current, wantPhase, wantClass)
	}
	if wantPhase != shared.OSOKAsyncPhaseDelete && current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.status.async.current.source = %q, want workrequest", current.Source)
	}
	if wantPhase == shared.OSOKAsyncPhaseDelete && current.Source != shared.OSOKAsyncSourceLifecycle {
		t.Fatalf("status.status.async.current.source = %q, want lifecycle", current.Source)
	}
}

func containsMaskingColumnTestString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
