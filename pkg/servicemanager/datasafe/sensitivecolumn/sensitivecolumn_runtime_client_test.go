/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sensitivecolumn

import (
	"context"
	"encoding/json"
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
	testSensitiveDataModelID       = "ocid1.sensitivedatamodel.oc1..example"
	testSensitiveDataModelOtherID  = "ocid1.sensitivedatamodel.oc1..other"
	testSensitiveColumnKey         = "42"
	testSensitiveColumnOtherKey    = "43"
	testSensitiveColumnUID         = "sensitive-column-uid"
	testSensitiveColumnWorkRequest = "ocid1.workrequest.oc1..sensitivecolumn"
)

type fakeSensitiveColumnOCIClient struct {
	createFn         func(context.Context, datasafesdk.CreateSensitiveColumnRequest) (datasafesdk.CreateSensitiveColumnResponse, error)
	getFn            func(context.Context, datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error)
	listFn           func(context.Context, datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error)
	updateFn         func(context.Context, datasafesdk.UpdateSensitiveColumnRequest) (datasafesdk.UpdateSensitiveColumnResponse, error)
	deleteFn         func(context.Context, datasafesdk.DeleteSensitiveColumnRequest) (datasafesdk.DeleteSensitiveColumnResponse, error)
	getWorkRequestFn func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
}

func (f *fakeSensitiveColumnOCIClient) CreateSensitiveColumn(
	ctx context.Context,
	request datasafesdk.CreateSensitiveColumnRequest,
) (datasafesdk.CreateSensitiveColumnResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return datasafesdk.CreateSensitiveColumnResponse{}, nil
}

func (f *fakeSensitiveColumnOCIClient) GetSensitiveColumn(
	ctx context.Context,
	request datasafesdk.GetSensitiveColumnRequest,
) (datasafesdk.GetSensitiveColumnResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return datasafesdk.GetSensitiveColumnResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "sensitive column not found")
}

func (f *fakeSensitiveColumnOCIClient) ListSensitiveColumns(
	ctx context.Context,
	request datasafesdk.ListSensitiveColumnsRequest,
) (datasafesdk.ListSensitiveColumnsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return datasafesdk.ListSensitiveColumnsResponse{}, nil
}

func (f *fakeSensitiveColumnOCIClient) UpdateSensitiveColumn(
	ctx context.Context,
	request datasafesdk.UpdateSensitiveColumnRequest,
) (datasafesdk.UpdateSensitiveColumnResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return datasafesdk.UpdateSensitiveColumnResponse{}, nil
}

func (f *fakeSensitiveColumnOCIClient) DeleteSensitiveColumn(
	ctx context.Context,
	request datasafesdk.DeleteSensitiveColumnRequest,
) (datasafesdk.DeleteSensitiveColumnResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return datasafesdk.DeleteSensitiveColumnResponse{OpcRequestId: common.String("opc-delete")}, nil
}

func (f *fakeSensitiveColumnOCIClient) GetWorkRequest(
	ctx context.Context,
	request datasafesdk.GetWorkRequestRequest,
) (datasafesdk.GetWorkRequestResponse, error) {
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, request)
	}
	return datasafesdk.GetWorkRequestResponse{}, nil
}

func testSensitiveColumnClient(fake *fakeSensitiveColumnOCIClient) SensitiveColumnServiceClient {
	if fake == nil {
		fake = &fakeSensitiveColumnOCIClient{}
	}
	manager := &SensitiveColumnServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}}
	hooks := newSensitiveColumnRuntimeHooksWithOCIClient(fake)
	applySensitiveColumnRuntimeHooks(&hooks, fake, nil)
	delegate := defaultSensitiveColumnServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.SensitiveColumn](
			buildSensitiveColumnGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapSensitiveColumnGeneratedClient(hooks, delegate)
}

func newSensitiveColumnRuntimeHooksWithOCIClient(client sensitiveColumnOCIClient) SensitiveColumnRuntimeHooks {
	hooks := newSensitiveColumnDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	hooks.Create.Call = func(ctx context.Context, request datasafesdk.CreateSensitiveColumnRequest) (datasafesdk.CreateSensitiveColumnResponse, error) {
		return client.CreateSensitiveColumn(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
		return client.GetSensitiveColumn(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error) {
		return client.ListSensitiveColumns(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request datasafesdk.UpdateSensitiveColumnRequest) (datasafesdk.UpdateSensitiveColumnResponse, error) {
		return client.UpdateSensitiveColumn(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request datasafesdk.DeleteSensitiveColumnRequest) (datasafesdk.DeleteSensitiveColumnResponse, error) {
		return client.DeleteSensitiveColumn(ctx, request)
	}
	return hooks
}

func TestSensitiveColumnRuntimeSemantics(t *testing.T) {
	semantics := sensitiveColumnRuntimeSemantics()
	if semantics == nil {
		t.Fatal("sensitiveColumnRuntimeSemantics() = nil")
	}
	requireSensitiveColumnAsyncSemantics(t, semantics.Async)
	requireSensitiveColumnFinalizerSemantics(t, semantics.FinalizerPolicy)
	requireSensitiveColumnMutationSemantics(t, semantics.Mutation)
	requireSensitiveColumnListSemantics(t, semantics.List)
}

func requireSensitiveColumnAsyncSemantics(t *testing.T, async *generatedruntime.AsyncSemantics) {
	t.Helper()
	if async == nil {
		t.Fatal("async semantics = nil, want generated workrequest")
	}
	if async.Strategy != "workrequest" || async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generated workrequest", async)
	}
}

func requireSensitiveColumnFinalizerSemantics(t *testing.T, finalizerPolicy string) {
	t.Helper()
	if finalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("finalizer policy = %q, want retain-until-confirmed-delete", finalizerPolicy)
	}
}

func requireSensitiveColumnMutationSemantics(t *testing.T, mutation generatedruntime.MutationSemantics) {
	t.Helper()
	if !containsSensitiveColumnTestString(mutation.ForceNew, "sensitiveDataModelId") {
		t.Fatalf("mutation force-new fields = %#v, want sensitiveDataModelId", mutation.ForceNew)
	}
	if !containsSensitiveColumnTestString(mutation.ForceNew, "schemaName") {
		t.Fatalf("mutation force-new fields = %#v, want schemaName", mutation.ForceNew)
	}
	if !containsSensitiveColumnTestString(mutation.Mutable, "dataType") {
		t.Fatalf("mutation mutable fields = %#v, want dataType", mutation.Mutable)
	}
}

func requireSensitiveColumnListSemantics(t *testing.T, list *generatedruntime.ListSemantics) {
	t.Helper()
	if list == nil || len(list.MatchFields) != 4 {
		t.Fatalf("list semantics = %#v, want four match fields", list)
	}
}

func TestSensitiveColumnCreateRequiresModelAnnotationBeforeOCICalls(t *testing.T) {
	resource := makeSensitiveColumnResource()
	resource.Annotations = nil
	fake := &fakeSensitiveColumnOCIClient{
		createFn: func(context.Context, datasafesdk.CreateSensitiveColumnRequest) (datasafesdk.CreateSensitiveColumnResponse, error) {
			t.Fatal("CreateSensitiveColumn should not be called without parent annotation")
			return datasafesdk.CreateSensitiveColumnResponse{}, nil
		},
		listFn: func(context.Context, datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error) {
			t.Fatal("ListSensitiveColumns should not be called without parent annotation")
			return datasafesdk.ListSensitiveColumnsResponse{}, nil
		},
	}

	response, err := testSensitiveColumnClient(fake).CreateOrUpdate(context.Background(), resource, sensitiveColumnRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing annotation rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), SensitiveColumnSensitiveDataModelIDAnnotation) {
		t.Fatalf("error = %q, want sensitive data model annotation context", err.Error())
	}
}

func TestSensitiveColumnCreateStartsWorkRequest(t *testing.T) {
	resource := makeSensitiveColumnResource()
	var createCalls int
	var listCalls int
	var capturedCreate datasafesdk.CreateSensitiveColumnRequest
	fake := &fakeSensitiveColumnOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error) {
			listCalls++
			requireSensitiveColumnListModel(t, request, testSensitiveDataModelID)
			return datasafesdk.ListSensitiveColumnsResponse{}, nil
		},
		createFn: func(_ context.Context, request datasafesdk.CreateSensitiveColumnRequest) (datasafesdk.CreateSensitiveColumnResponse, error) {
			createCalls++
			capturedCreate = request
			return datasafesdk.CreateSensitiveColumnResponse{
				OpcWorkRequestId: common.String(testSensitiveColumnWorkRequest),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, request datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			if got := sensitiveColumnStringValue(request.WorkRequestId); got != testSensitiveColumnWorkRequest {
				t.Fatalf("workRequestId = %q, want %q", got, testSensitiveColumnWorkRequest)
			}
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: makeSensitiveColumnWorkRequest(datasafesdk.WorkRequestStatusAccepted, datasafesdk.WorkRequestOperationTypeCreateSensitiveColumn, nil),
			}, nil
		},
	}

	response, err := testSensitiveColumnClient(fake).CreateOrUpdate(context.Background(), resource, sensitiveColumnRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false")
	}
	if listCalls != 1 || createCalls != 1 {
		t.Fatalf("listCalls = %d, createCalls = %d, want 1/1", listCalls, createCalls)
	}
	requireSensitiveColumnCreateRequest(t, capturedCreate)
	requireSensitiveColumnOpcRequestID(t, resource, "opc-create")
	requireSensitiveColumnAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending)
}

func TestSensitiveColumnBindsExistingBeforeCreateAcrossPages(t *testing.T) {
	resource := makeSensitiveColumnResource()
	var listPages []string
	fake := &fakeSensitiveColumnOCIClient{
		createFn: func(context.Context, datasafesdk.CreateSensitiveColumnRequest) (datasafesdk.CreateSensitiveColumnResponse, error) {
			t.Fatal("CreateSensitiveColumn should not be called when list discovers existing column")
			return datasafesdk.CreateSensitiveColumnResponse{}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			requireSensitiveColumnGetRequest(t, request, testSensitiveDataModelID, testSensitiveColumnKey)
			return datasafesdk.GetSensitiveColumnResponse{
				SensitiveColumn: makeSDKSensitiveColumn(testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive),
			}, nil
		},
		listFn: func(_ context.Context, request datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error) {
			requireSensitiveColumnListModel(t, request, testSensitiveDataModelID)
			listPages = append(listPages, sensitiveColumnStringValue(request.Page))
			if request.Page == nil {
				return datasafesdk.ListSensitiveColumnsResponse{
					SensitiveColumnCollection: datasafesdk.SensitiveColumnCollection{
						Items: []datasafesdk.SensitiveColumnSummary{
							makeSDKSensitiveColumnSummary(testSensitiveColumnOtherKey, testSensitiveDataModelID, "APP", "CUSTOMERS", "EMAIL", datasafesdk.SensitiveColumnLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("next"),
				}, nil
			}
			return datasafesdk.ListSensitiveColumnsResponse{
				SensitiveColumnCollection: datasafesdk.SensitiveColumnCollection{
					Items: []datasafesdk.SensitiveColumnSummary{
						makeSDKSensitiveColumnSummary(testSensitiveColumnKey, testSensitiveDataModelID, "APP", "CUSTOMERS", "SSN", datasafesdk.SensitiveColumnLifecycleStateActive),
					},
				},
			}, nil
		},
	}

	response, err := testSensitiveColumnClient(fake).CreateOrUpdate(context.Background(), resource, sensitiveColumnRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false")
	}
	if got, want := strings.Join(listPages, ","), ",next"; got != want {
		t.Fatalf("list pages = %q, want %q", got, want)
	}
	requireSensitiveColumnProjectedStatus(t, resource, testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive)
}

func TestSensitiveColumnCreateWorkRequestSuccessRecordsRecoveredKey(t *testing.T) {
	resource := makeSensitiveColumnResource()
	resource.Status.SensitiveDataModelId = testSensitiveDataModelID
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   testSensitiveColumnWorkRequest,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	var getCalls int
	fake := &fakeSensitiveColumnOCIClient{
		createFn: func(context.Context, datasafesdk.CreateSensitiveColumnRequest) (datasafesdk.CreateSensitiveColumnResponse, error) {
			t.Fatal("CreateSensitiveColumn should not be called while create work request is tracked")
			return datasafesdk.CreateSensitiveColumnResponse{}, nil
		},
		listFn: func(context.Context, datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error) {
			t.Fatal("ListSensitiveColumns should not be called when work request exposes the sensitive column key")
			return datasafesdk.ListSensitiveColumnsResponse{}, nil
		},
		getWorkRequestFn: func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: makeSensitiveColumnWorkRequest(
					datasafesdk.WorkRequestStatusSucceeded,
					datasafesdk.WorkRequestOperationTypeCreateSensitiveColumn,
					[]datasafesdk.WorkRequestResource{
						{
							Identifier: common.String(testSensitiveColumnKey),
							ActionType: datasafesdk.WorkRequestResourceActionTypeCreated,
						},
					},
				),
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			getCalls++
			requireSensitiveColumnGetRequest(t, request, testSensitiveDataModelID, testSensitiveColumnKey)
			return datasafesdk.GetSensitiveColumnResponse{
				SensitiveColumn: makeSDKSensitiveColumn(testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive),
			}, nil
		},
	}

	response, err := testSensitiveColumnClient(fake).CreateOrUpdate(context.Background(), resource, sensitiveColumnRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false")
	}
	if getCalls != 1 {
		t.Fatalf("GetSensitiveColumn calls = %d, want 1", getCalls)
	}
	requireSensitiveColumnProjectedStatus(t, resource, testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive)
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after successful work request", resource.Status.OsokStatus.Async.Current)
	}
}

func TestSensitiveColumnNoopReconcileSkipsUpdate(t *testing.T) {
	resource := makeSensitiveColumnResourceWithTrackedKey()
	var updateCalls int
	fake := &fakeSensitiveColumnOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			requireSensitiveColumnGetRequest(t, request, testSensitiveDataModelID, testSensitiveColumnKey)
			return datasafesdk.GetSensitiveColumnResponse{
				SensitiveColumn: makeSDKSensitiveColumn(testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateSensitiveColumnRequest) (datasafesdk.UpdateSensitiveColumnResponse, error) {
			updateCalls++
			return datasafesdk.UpdateSensitiveColumnResponse{}, nil
		},
	}

	response, err := testSensitiveColumnClient(fake).CreateOrUpdate(context.Background(), resource, sensitiveColumnRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateSensitiveColumn calls = %d, want 0", updateCalls)
	}
	requireSensitiveColumnProjectedStatus(t, resource, testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive)
}

func TestSensitiveColumnProjectsConfidenceLevelDetailsFromGetReadback(t *testing.T) {
	resource := makeSensitiveColumnResourceWithTrackedKey()
	details := []interface{}{
		map[string]interface{}{
			"Operations": []interface{}{
				map[string]interface{}{"CostCenter": "42"},
			},
		},
	}
	fake := &fakeSensitiveColumnOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			requireSensitiveColumnGetRequest(t, request, testSensitiveDataModelID, testSensitiveColumnKey)
			current := makeSDKSensitiveColumn(testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive)
			current.ConfidenceLevel = datasafesdk.ConfidenceLevelEnumHigh
			current.ConfidenceLevelDetails = details
			return datasafesdk.GetSensitiveColumnResponse{SensitiveColumn: current}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateSensitiveColumnRequest) (datasafesdk.UpdateSensitiveColumnResponse, error) {
			t.Fatal("UpdateSensitiveColumn should not be called during no-op confidence detail readback")
			return datasafesdk.UpdateSensitiveColumnResponse{}, nil
		},
	}

	response, err := testSensitiveColumnClient(fake).CreateOrUpdate(context.Background(), resource, sensitiveColumnRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false")
	}
	if got := resource.Status.ConfidenceLevel; got != string(datasafesdk.ConfidenceLevelEnumHigh) {
		t.Fatalf("status.confidenceLevel = %q, want HIGH", got)
	}
	requireSensitiveColumnConfidenceLevelDetails(t, resource)
}

func TestSensitiveColumnDoesNotProjectSampleDataValuesFromGetReadback(t *testing.T) {
	resource := makeSensitiveColumnResourceWithTrackedKey()
	fake := &fakeSensitiveColumnOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			requireSensitiveColumnGetRequest(t, request, testSensitiveDataModelID, testSensitiveColumnKey)
			current := makeSDKSensitiveColumn(testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive)
			current.SampleDataValues = []string{"4111111111111111", "123-45-6789"}
			return datasafesdk.GetSensitiveColumnResponse{SensitiveColumn: current}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateSensitiveColumnRequest) (datasafesdk.UpdateSensitiveColumnResponse, error) {
			t.Fatal("UpdateSensitiveColumn should not be called during no-op sample data readback")
			return datasafesdk.UpdateSensitiveColumnResponse{}, nil
		},
	}

	response, err := testSensitiveColumnClient(fake).CreateOrUpdate(context.Background(), resource, sensitiveColumnRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false")
	}
	if len(resource.Status.SampleDataValues) != 0 {
		t.Fatalf("status.sampleDataValues = %#v, want omitted", resource.Status.SampleDataValues)
	}
}

func TestSensitiveColumnMutableUpdateStartsWorkRequest(t *testing.T) {
	resource := makeSensitiveColumnResourceWithTrackedKey()
	resource.Spec.DataType = "NUMBER"
	resource.Spec.Status = "INVALID"
	resource.Spec.SensitiveTypeId = "ocid1.sensitivetype.oc1..new"
	resource.Spec.ParentColumnKeys = []string{"parent-2"}
	resource.Spec.RelationType = "APP_DEFINED"
	resource.Spec.AppDefinedChildColumnKeys = []string{"child-1"}
	resource.Spec.DbDefinedChildColumnKeys = []string{}
	var updateCalls int
	var capturedUpdate datasafesdk.UpdateSensitiveColumnRequest
	fake := &fakeSensitiveColumnOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			requireSensitiveColumnGetRequest(t, request, testSensitiveDataModelID, testSensitiveColumnKey)
			return datasafesdk.GetSensitiveColumnResponse{
				SensitiveColumn: makeSDKSensitiveColumn(testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, request datasafesdk.UpdateSensitiveColumnRequest) (datasafesdk.UpdateSensitiveColumnResponse, error) {
			updateCalls++
			capturedUpdate = request
			return datasafesdk.UpdateSensitiveColumnResponse{
				OpcWorkRequestId: common.String(testSensitiveColumnWorkRequest),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
		getWorkRequestFn: func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: makeSensitiveColumnWorkRequest(datasafesdk.WorkRequestStatusAccepted, datasafesdk.WorkRequestOperationTypeUpdateSensitiveColumn, nil),
			}, nil
		},
	}

	response, err := testSensitiveColumnClient(fake).CreateOrUpdate(context.Background(), resource, sensitiveColumnRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateSensitiveColumn calls = %d, want 1", updateCalls)
	}
	requireSensitiveColumnUpdateRequest(t, capturedUpdate)
	requireSensitiveColumnOpcRequestID(t, resource, "opc-update")
	requireSensitiveColumnAsyncCurrent(t, resource, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending)
}

func TestSensitiveColumnCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	resource := makeSensitiveColumnResourceWithTrackedKey()
	resource.Spec.SchemaName = "APP_CHANGED"
	var updateCalls int
	fake := &fakeSensitiveColumnOCIClient{
		getFn: func(context.Context, datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			return datasafesdk.GetSensitiveColumnResponse{
				SensitiveColumn: makeSDKSensitiveColumn(testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateSensitiveColumnRequest) (datasafesdk.UpdateSensitiveColumnResponse, error) {
			updateCalls++
			return datasafesdk.UpdateSensitiveColumnResponse{}, nil
		},
	}

	response, err := testSensitiveColumnClient(fake).CreateOrUpdate(context.Background(), resource, sensitiveColumnRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = true, want false")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateSensitiveColumn calls = %d, want 0", updateCalls)
	}
	if !strings.Contains(err.Error(), "schemaName") {
		t.Fatalf("error = %q, want schemaName drift context", err.Error())
	}
}

func TestSensitiveColumnCreateOrUpdateRejectsChangedParentAnnotation(t *testing.T) {
	resource := makeSensitiveColumnResourceWithTrackedKey()
	resource.Annotations[SensitiveColumnSensitiveDataModelIDAnnotation] = testSensitiveDataModelOtherID
	fake := &fakeSensitiveColumnOCIClient{
		getFn: func(context.Context, datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			t.Fatal("GetSensitiveColumn should not be called after parent annotation drift")
			return datasafesdk.GetSensitiveColumnResponse{}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateSensitiveColumnRequest) (datasafesdk.UpdateSensitiveColumnResponse, error) {
			t.Fatal("UpdateSensitiveColumn should not be called after parent annotation drift")
			return datasafesdk.UpdateSensitiveColumnResponse{}, nil
		},
	}

	response, err := testSensitiveColumnClient(fake).CreateOrUpdate(context.Background(), resource, sensitiveColumnRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want parent annotation drift rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), SensitiveColumnSensitiveDataModelIDAnnotation) {
		t.Fatalf("error = %q, want parent annotation drift context", err.Error())
	}
}

func TestSensitiveColumnDeleteRetainsFinalizerUntilDeleteConfirmed(t *testing.T) {
	resource := makeSensitiveColumnResourceWithTrackedKey()
	var getCalls int
	var deleteCalls int
	fake := &fakeSensitiveColumnOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			getCalls++
			requireSensitiveColumnGetRequest(t, request, testSensitiveDataModelID, testSensitiveColumnKey)
			return datasafesdk.GetSensitiveColumnResponse{
				SensitiveColumn: makeSDKSensitiveColumn(testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, request datasafesdk.DeleteSensitiveColumnRequest) (datasafesdk.DeleteSensitiveColumnResponse, error) {
			deleteCalls++
			requireSensitiveColumnDeleteRequest(t, request, testSensitiveDataModelID, testSensitiveColumnKey)
			return datasafesdk.DeleteSensitiveColumnResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := testSensitiveColumnClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false until readback confirms deletion")
	}
	if getCalls != 2 || deleteCalls != 1 {
		t.Fatalf("getCalls = %d, deleteCalls = %d, want 2/1", getCalls, deleteCalls)
	}
	requireSensitiveColumnOpcRequestID(t, resource, "opc-delete")
	requireSensitiveColumnAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
}

func TestSensitiveColumnDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	resource := makeSensitiveColumnResourceWithTrackedKey()
	var getCalls int
	fake := &fakeSensitiveColumnOCIClient{
		getFn: func(context.Context, datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			getCalls++
			if getCalls == 1 {
				return datasafesdk.GetSensitiveColumnResponse{
					SensitiveColumn: makeSDKSensitiveColumn(testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive),
				}, nil
			}
			return datasafesdk.GetSensitiveColumnResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
		},
		deleteFn: func(context.Context, datasafesdk.DeleteSensitiveColumnRequest) (datasafesdk.DeleteSensitiveColumnResponse, error) {
			return datasafesdk.DeleteSensitiveColumnResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := testSensitiveColumnClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not found")
	}
}

func TestSensitiveColumnDeleteConfirmsAbsentUntrackedColumnAfterPagedList(t *testing.T) {
	resource := makeSensitiveColumnResource()
	var listPages []string
	fake := &fakeSensitiveColumnOCIClient{
		getFn: func(context.Context, datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			t.Fatal("GetSensitiveColumn should not be called without a tracked sensitive column key")
			return datasafesdk.GetSensitiveColumnResponse{}, nil
		},
		listFn: func(_ context.Context, request datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error) {
			requireSensitiveColumnListModel(t, request, testSensitiveDataModelID)
			requireSensitiveColumnStringSlice(t, "list schemaName", request.SchemaName, []string{"APP"})
			requireSensitiveColumnStringSlice(t, "list objectName", request.ObjectName, []string{"CUSTOMERS"})
			requireSensitiveColumnStringSlice(t, "list columnName", request.ColumnName, []string{"SSN"})
			listPages = append(listPages, sensitiveColumnStringValue(request.Page))
			if request.Page == nil {
				return datasafesdk.ListSensitiveColumnsResponse{
					SensitiveColumnCollection: datasafesdk.SensitiveColumnCollection{
						Items: []datasafesdk.SensitiveColumnSummary{
							makeSDKSensitiveColumnSummary(testSensitiveColumnOtherKey, testSensitiveDataModelID, "APP", "CUSTOMERS", "EMAIL", datasafesdk.SensitiveColumnLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("next"),
				}, nil
			}
			return datasafesdk.ListSensitiveColumnsResponse{
				SensitiveColumnCollection: datasafesdk.SensitiveColumnCollection{
					Items: []datasafesdk.SensitiveColumnSummary{
						makeSDKSensitiveColumnSummary(testSensitiveColumnOtherKey, testSensitiveDataModelID, "APP", "CUSTOMERS", "PHONE", datasafesdk.SensitiveColumnLifecycleStateActive),
					},
				},
			}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteSensitiveColumnRequest) (datasafesdk.DeleteSensitiveColumnResponse, error) {
			t.Fatal("DeleteSensitiveColumn should not be called after list confirms absence")
			return datasafesdk.DeleteSensitiveColumnResponse{}, nil
		},
	}

	deleted, err := testSensitiveColumnClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after paged list confirms absence")
	}
	if got, want := strings.Join(listPages, ","), ",next"; got != want {
		t.Fatalf("list pages = %q, want %q", got, want)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deleted timestamp")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after confirmed absence", resource.Status.OsokStatus.Async.Current)
	}
}

func TestSensitiveColumnDeleteUsesRecordedParentForTrackedKey(t *testing.T) {
	resource := makeSensitiveColumnResourceWithTrackedKey()
	resource.Annotations[SensitiveColumnSensitiveDataModelIDAnnotation] = testSensitiveDataModelOtherID
	var getCalls int
	var deleteCalls int
	fake := &fakeSensitiveColumnOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			getCalls++
			requireSensitiveColumnGetRequest(t, request, testSensitiveDataModelID, testSensitiveColumnKey)
			return datasafesdk.GetSensitiveColumnResponse{
				SensitiveColumn: makeSDKSensitiveColumn(testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, request datasafesdk.DeleteSensitiveColumnRequest) (datasafesdk.DeleteSensitiveColumnResponse, error) {
			deleteCalls++
			requireSensitiveColumnDeleteRequest(t, request, testSensitiveDataModelID, testSensitiveColumnKey)
			return datasafesdk.DeleteSensitiveColumnResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := testSensitiveColumnClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false until recorded-parent readback confirms deletion")
	}
	if getCalls != 2 || deleteCalls != 1 {
		t.Fatalf("getCalls = %d, deleteCalls = %d, want 2/1", getCalls, deleteCalls)
	}
	if got := resource.Annotations[SensitiveColumnSensitiveDataModelIDAnnotation]; got != testSensitiveDataModelOtherID {
		t.Fatalf("sensitive data model annotation = %q, want restored changed annotation", got)
	}
}

func TestSensitiveColumnDeleteUsesRecordedParentForUntrackedKey(t *testing.T) {
	resource := makeSensitiveColumnResource()
	resource.Status.SensitiveDataModelId = testSensitiveDataModelID
	resource.Annotations[SensitiveColumnSensitiveDataModelIDAnnotation] = testSensitiveDataModelOtherID
	fake := &fakeSensitiveColumnOCIClient{
		getFn: func(context.Context, datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			t.Fatal("GetSensitiveColumn should not be called without a tracked sensitive column key")
			return datasafesdk.GetSensitiveColumnResponse{}, nil
		},
		listFn: func(_ context.Context, request datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error) {
			requireSensitiveColumnListModel(t, request, testSensitiveDataModelID)
			return datasafesdk.ListSensitiveColumnsResponse{}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteSensitiveColumnRequest) (datasafesdk.DeleteSensitiveColumnResponse, error) {
			t.Fatal("DeleteSensitiveColumn should not be called after recorded-parent list confirms absence")
			return datasafesdk.DeleteSensitiveColumnResponse{}, nil
		},
	}

	deleted, err := testSensitiveColumnClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after recorded-parent list confirms absence")
	}
}

func TestSensitiveColumnDeleteUsesRecordedPathForUntrackedKeyWhenSpecDrifts(t *testing.T) {
	resource := makeSensitiveColumnResource()
	resource.Status.SensitiveDataModelId = testSensitiveDataModelID
	resource.Status.SchemaName = "APP"
	resource.Status.ObjectName = "CUSTOMERS"
	resource.Status.ColumnName = "SSN"
	resource.Spec.SchemaName = "APP_CHANGED"
	resource.Spec.ObjectName = "ORDERS"
	resource.Spec.ColumnName = "EMAIL"
	var getCalls int
	var deleteCalls int
	fake := &fakeSensitiveColumnOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error) {
			requireSensitiveColumnListModel(t, request, testSensitiveDataModelID)
			requireSensitiveColumnStringSlice(t, "list schemaName", request.SchemaName, []string{"APP"})
			requireSensitiveColumnStringSlice(t, "list objectName", request.ObjectName, []string{"CUSTOMERS"})
			requireSensitiveColumnStringSlice(t, "list columnName", request.ColumnName, []string{"SSN"})
			return datasafesdk.ListSensitiveColumnsResponse{
				SensitiveColumnCollection: datasafesdk.SensitiveColumnCollection{
					Items: []datasafesdk.SensitiveColumnSummary{
						makeSDKSensitiveColumnSummary(testSensitiveColumnKey, testSensitiveDataModelID, "APP", "CUSTOMERS", "SSN", datasafesdk.SensitiveColumnLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			getCalls++
			requireSensitiveColumnGetRequest(t, request, testSensitiveDataModelID, testSensitiveColumnKey)
			return datasafesdk.GetSensitiveColumnResponse{
				SensitiveColumn: makeSDKSensitiveColumn(testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, request datasafesdk.DeleteSensitiveColumnRequest) (datasafesdk.DeleteSensitiveColumnResponse, error) {
			deleteCalls++
			requireSensitiveColumnDeleteRequest(t, request, testSensitiveDataModelID, testSensitiveColumnKey)
			return datasafesdk.DeleteSensitiveColumnResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := testSensitiveColumnClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false because recorded status path still exists")
	}
	if getCalls != 2 || deleteCalls != 1 {
		t.Fatalf("getCalls = %d, deleteCalls = %d, want 2/1", getCalls, deleteCalls)
	}
	if got := resource.Status.Key; got != testSensitiveColumnKey {
		t.Fatalf("status.key = %q, want %q", got, testSensitiveColumnKey)
	}
}

func TestSensitiveColumnSyntheticDeleteConfirmReadUsesRecordedPathWhenSpecDrifts(t *testing.T) {
	resource := makeSensitiveColumnResource()
	resource.Status.SensitiveDataModelId = testSensitiveDataModelID
	resource.Status.SchemaName = "APP"
	resource.Status.ObjectName = "CUSTOMERS"
	resource.Status.ColumnName = "SSN"
	resource.Spec.SchemaName = "APP_CHANGED"
	resource.Spec.ObjectName = "ORDERS"
	resource.Spec.ColumnName = "EMAIL"
	currentID := syntheticSensitiveColumnKey(sensitiveColumnIdentity{
		sensitiveDataModelID: testSensitiveDataModelID,
		schemaName:           "APP",
		objectName:           "CUSTOMERS",
		columnName:           "SSN",
	})
	confirmRead := sensitiveColumnDeleteConfirmRead(
		nil,
		func(_ context.Context, request datasafesdk.ListSensitiveColumnsRequest) (datasafesdk.ListSensitiveColumnsResponse, error) {
			requireSensitiveColumnListModel(t, request, testSensitiveDataModelID)
			requireSensitiveColumnStringSlice(t, "list schemaName", request.SchemaName, []string{"APP"})
			requireSensitiveColumnStringSlice(t, "list objectName", request.ObjectName, []string{"CUSTOMERS"})
			requireSensitiveColumnStringSlice(t, "list columnName", request.ColumnName, []string{"SSN"})
			return datasafesdk.ListSensitiveColumnsResponse{
				SensitiveColumnCollection: datasafesdk.SensitiveColumnCollection{
					Items: []datasafesdk.SensitiveColumnSummary{
						makeSDKSensitiveColumnSummary(testSensitiveColumnKey, testSensitiveDataModelID, "APP", "CUSTOMERS", "SSN", datasafesdk.SensitiveColumnLifecycleStateActive),
					},
				},
			}, nil
		},
	)

	response, err := confirmRead(context.Background(), resource, currentID)
	if err != nil {
		t.Fatalf("confirmRead() error = %v", err)
	}
	match, ok := response.(datasafesdk.SensitiveColumnSummary)
	if !ok {
		t.Fatalf("confirmRead() response = %T, want SensitiveColumnSummary", response)
	}
	if got := sensitiveColumnStringValue(match.Key); got != testSensitiveColumnKey {
		t.Fatalf("confirmRead() key = %q, want %q", got, testSensitiveColumnKey)
	}
}

func TestSensitiveColumnDeleteBlocksAuthShapedConfirmRead(t *testing.T) {
	resource := makeSensitiveColumnResourceWithTrackedKey()
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	var getCalls int
	fake := &fakeSensitiveColumnOCIClient{
		getFn: func(context.Context, datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			getCalls++
			if getCalls == 1 {
				return datasafesdk.GetSensitiveColumnResponse{
					SensitiveColumn: makeSDKSensitiveColumn(testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive),
				}, nil
			}
			return datasafesdk.GetSensitiveColumnResponse{}, authErr
		},
		deleteFn: func(context.Context, datasafesdk.DeleteSensitiveColumnRequest) (datasafesdk.DeleteSensitiveColumnResponse, error) {
			return datasafesdk.DeleteSensitiveColumnResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := testSensitiveColumnClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous NotAuthorizedOrNotFound rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false on ambiguous NotAuthorizedOrNotFound")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("error = %q, want ambiguous NotAuthorizedOrNotFound context", err.Error())
	}
}

func TestSensitiveColumnDeleteWaitsForPendingWriteWorkRequest(t *testing.T) {
	resource := makeSensitiveColumnResourceWithTrackedKey()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:   testSensitiveColumnWorkRequest,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fake := &fakeSensitiveColumnOCIClient{
		getWorkRequestFn: func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: makeSensitiveColumnWorkRequest(datasafesdk.WorkRequestStatusInProgress, datasafesdk.WorkRequestOperationTypeUpdateSensitiveColumn, nil),
			}, nil
		},
		getFn: func(context.Context, datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			t.Fatal("GetSensitiveColumn should not be called while write work request is pending")
			return datasafesdk.GetSensitiveColumnResponse{}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteSensitiveColumnRequest) (datasafesdk.DeleteSensitiveColumnResponse, error) {
			t.Fatal("DeleteSensitiveColumn should not be called while write work request is pending")
			return datasafesdk.DeleteSensitiveColumnResponse{}, nil
		},
	}

	deleted, err := testSensitiveColumnClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while write work request is pending")
	}
	if resource.Status.OsokStatus.Async.Current == nil || !strings.Contains(resource.Status.OsokStatus.Async.Current.Message, "still in progress") {
		t.Fatalf("status.async.current = %#v, want pending write message", resource.Status.OsokStatus.Async.Current)
	}
}

func TestSensitiveColumnDeleteUsesRecordedParentAfterSucceededPendingWrite(t *testing.T) {
	resource := makeSensitiveColumnResource()
	resource.Status.SensitiveDataModelId = testSensitiveDataModelID
	resource.Annotations[SensitiveColumnSensitiveDataModelIDAnnotation] = testSensitiveDataModelOtherID
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   testSensitiveColumnWorkRequest,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	var getCalls int
	var deleteCalls int
	fake := &fakeSensitiveColumnOCIClient{
		getWorkRequestFn: func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
			return datasafesdk.GetWorkRequestResponse{
				WorkRequest: makeSensitiveColumnWorkRequest(
					datasafesdk.WorkRequestStatusSucceeded,
					datasafesdk.WorkRequestOperationTypeCreateSensitiveColumn,
					[]datasafesdk.WorkRequestResource{
						{
							Identifier: common.String(testSensitiveColumnKey),
							ActionType: datasafesdk.WorkRequestResourceActionTypeCreated,
						},
					},
				),
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetSensitiveColumnRequest) (datasafesdk.GetSensitiveColumnResponse, error) {
			getCalls++
			requireSensitiveColumnGetRequest(t, request, testSensitiveDataModelID, testSensitiveColumnKey)
			return datasafesdk.GetSensitiveColumnResponse{
				SensitiveColumn: makeSDKSensitiveColumn(testSensitiveColumnKey, testSensitiveDataModelID, datasafesdk.SensitiveColumnLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, request datasafesdk.DeleteSensitiveColumnRequest) (datasafesdk.DeleteSensitiveColumnResponse, error) {
			deleteCalls++
			requireSensitiveColumnDeleteRequest(t, request, testSensitiveDataModelID, testSensitiveColumnKey)
			return datasafesdk.DeleteSensitiveColumnResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := testSensitiveColumnClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false until recorded-parent readback confirms deletion")
	}
	if getCalls != 2 || deleteCalls != 1 {
		t.Fatalf("getCalls = %d, deleteCalls = %d, want 2/1", getCalls, deleteCalls)
	}
	if got := resource.Status.Key; got != testSensitiveColumnKey {
		t.Fatalf("status.key = %q, want recovered key", got)
	}
	if got := resource.Annotations[SensitiveColumnSensitiveDataModelIDAnnotation]; got != testSensitiveDataModelOtherID {
		t.Fatalf("sensitive data model annotation = %q, want restored changed annotation", got)
	}
	requireSensitiveColumnOpcRequestID(t, resource, "opc-delete")
	requireSensitiveColumnAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
}

func makeSensitiveColumnResource() *datasafev1beta1.SensitiveColumn {
	return &datasafev1beta1.SensitiveColumn{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sensitive-column",
			Namespace: "default",
			UID:       types.UID(testSensitiveColumnUID),
			Annotations: map[string]string{
				SensitiveColumnSensitiveDataModelIDAnnotation: testSensitiveDataModelID,
			},
		},
		Spec: datasafev1beta1.SensitiveColumnSpec{
			SchemaName:       "APP",
			ObjectName:       "CUSTOMERS",
			ColumnName:       "SSN",
			AppName:          "APP",
			ObjectType:       "TABLE",
			DataType:         "VARCHAR2",
			Status:           "VALID",
			SensitiveTypeId:  "ocid1.sensitivetype.oc1..old",
			ParentColumnKeys: []string{"parent-1"},
			RelationType:     "NONE",
		},
	}
}

func makeSensitiveColumnResourceWithTrackedKey() *datasafev1beta1.SensitiveColumn {
	resource := makeSensitiveColumnResource()
	resource.Status.Key = testSensitiveColumnKey
	resource.Status.SensitiveDataModelId = testSensitiveDataModelID
	resource.Status.Status = string(datasafesdk.SensitiveColumnStatusValid)
	resource.Status.OsokStatus.Ocid = shared.OCID(testSensitiveColumnKey)
	return resource
}

func sensitiveColumnRequest(resource *datasafev1beta1.SensitiveColumn) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}}
}

func makeSDKSensitiveColumn(
	key string,
	modelID string,
	state datasafesdk.SensitiveColumnLifecycleStateEnum,
) datasafesdk.SensitiveColumn {
	return datasafesdk.SensitiveColumn{
		Key:                     common.String(key),
		SensitiveDataModelId:    common.String(modelID),
		LifecycleState:          state,
		AppName:                 common.String("APP"),
		SchemaName:              common.String("APP"),
		ObjectName:              common.String("CUSTOMERS"),
		ColumnName:              common.String("SSN"),
		ObjectType:              datasafesdk.SensitiveColumnObjectTypeTable,
		DataType:                common.String("VARCHAR2"),
		Status:                  datasafesdk.SensitiveColumnStatusValid,
		Source:                  datasafesdk.SensitiveColumnSourceManual,
		RelationType:            datasafesdk.SensitiveColumnRelationTypeNone,
		EstimatedDataValueCount: common.Int64(10),
		SensitiveTypeId:         common.String("ocid1.sensitivetype.oc1..old"),
		ParentColumnKeys:        []string{"parent-1"},
	}
}

func makeSDKSensitiveColumnSummary(
	key string,
	modelID string,
	schema string,
	objectName string,
	column string,
	state datasafesdk.SensitiveColumnLifecycleStateEnum,
) datasafesdk.SensitiveColumnSummary {
	return datasafesdk.SensitiveColumnSummary{
		Key:                     common.String(key),
		SensitiveDataModelId:    common.String(modelID),
		LifecycleState:          state,
		AppName:                 common.String(schema),
		SchemaName:              common.String(schema),
		ObjectName:              common.String(objectName),
		ColumnName:              common.String(column),
		ObjectType:              datasafesdk.SensitiveColumnSummaryObjectTypeTable,
		DataType:                common.String("VARCHAR2"),
		Status:                  datasafesdk.SensitiveColumnSummaryStatusValid,
		Source:                  datasafesdk.SensitiveColumnSummarySourceManual,
		RelationType:            datasafesdk.SensitiveColumnSummaryRelationTypeNone,
		EstimatedDataValueCount: common.Int64(10),
		SensitiveTypeId:         common.String("ocid1.sensitivetype.oc1..old"),
		ParentColumnKeys:        []string{"parent-1"},
	}
}

func makeSensitiveColumnWorkRequest(
	status datasafesdk.WorkRequestStatusEnum,
	operation datasafesdk.WorkRequestOperationTypeEnum,
	resources []datasafesdk.WorkRequestResource,
) datasafesdk.WorkRequest {
	return datasafesdk.WorkRequest{
		Id:              common.String(testSensitiveColumnWorkRequest),
		Status:          status,
		OperationType:   operation,
		PercentComplete: common.Float32(50),
		Resources:       resources,
	}
}

func requireSensitiveColumnCreateRequest(t *testing.T, request datasafesdk.CreateSensitiveColumnRequest) {
	t.Helper()
	requireSensitiveColumnString(t, "create sensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
	details := request.CreateSensitiveColumnDetails
	requireSensitiveColumnString(t, "create schemaName", details.SchemaName, "APP")
	requireSensitiveColumnString(t, "create objectName", details.ObjectName, "CUSTOMERS")
	requireSensitiveColumnString(t, "create columnName", details.ColumnName, "SSN")
	requireSensitiveColumnString(t, "create appName", details.AppName, "APP")
	if details.ObjectType != datasafesdk.CreateSensitiveColumnDetailsObjectTypeTable {
		t.Fatalf("create objectType = %q, want TABLE", details.ObjectType)
	}
	if details.Status != datasafesdk.CreateSensitiveColumnDetailsStatusValid {
		t.Fatalf("create status = %q, want VALID", details.Status)
	}
	if details.RelationType != datasafesdk.CreateSensitiveColumnDetailsRelationTypeNone {
		t.Fatalf("create relationType = %q, want NONE", details.RelationType)
	}
}

func requireSensitiveColumnUpdateRequest(t *testing.T, request datasafesdk.UpdateSensitiveColumnRequest) {
	t.Helper()
	requireSensitiveColumnString(t, "update sensitiveDataModelId", request.SensitiveDataModelId, testSensitiveDataModelID)
	requireSensitiveColumnString(t, "update sensitiveColumnKey", request.SensitiveColumnKey, testSensitiveColumnKey)
	details := request.UpdateSensitiveColumnDetails
	requireSensitiveColumnString(t, "update dataType", details.DataType, "NUMBER")
	requireSensitiveColumnString(t, "update sensitiveTypeId", details.SensitiveTypeId, "ocid1.sensitivetype.oc1..new")
	if details.Status != datasafesdk.UpdateSensitiveColumnDetailsStatusInvalid {
		t.Fatalf("update status = %q, want INVALID", details.Status)
	}
	if details.RelationType != datasafesdk.UpdateSensitiveColumnDetailsRelationTypeAppDefined {
		t.Fatalf("update relationType = %q, want APP_DEFINED", details.RelationType)
	}
	requireSensitiveColumnStringSlice(t, "update parentColumnKeys", details.ParentColumnKeys, []string{"parent-2"})
	requireSensitiveColumnStringSlice(t, "update appDefinedChildColumnKeys", details.AppDefinedChildColumnKeys, []string{"child-1"})
	requireSensitiveColumnStringSlice(t, "update dbDefinedChildColumnKeys", details.DbDefinedChildColumnKeys, []string{})
}

func requireSensitiveColumnGetRequest(t *testing.T, request datasafesdk.GetSensitiveColumnRequest, modelID string, key string) {
	t.Helper()
	requireSensitiveColumnString(t, "get sensitiveDataModelId", request.SensitiveDataModelId, modelID)
	requireSensitiveColumnString(t, "get sensitiveColumnKey", request.SensitiveColumnKey, key)
}

func requireSensitiveColumnDeleteRequest(t *testing.T, request datasafesdk.DeleteSensitiveColumnRequest, modelID string, key string) {
	t.Helper()
	requireSensitiveColumnString(t, "delete sensitiveDataModelId", request.SensitiveDataModelId, modelID)
	requireSensitiveColumnString(t, "delete sensitiveColumnKey", request.SensitiveColumnKey, key)
}

func requireSensitiveColumnListModel(t *testing.T, request datasafesdk.ListSensitiveColumnsRequest, modelID string) {
	t.Helper()
	requireSensitiveColumnString(t, "list sensitiveDataModelId", request.SensitiveDataModelId, modelID)
}

func requireSensitiveColumnProjectedStatus(
	t *testing.T,
	resource *datasafev1beta1.SensitiveColumn,
	key string,
	modelID string,
	state datasafesdk.SensitiveColumnLifecycleStateEnum,
) {
	t.Helper()
	if got := resource.Status.Key; got != key {
		t.Fatalf("status.key = %q, want %q", got, key)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != key {
		t.Fatalf("status.status.ocid = %q, want %q", got, key)
	}
	if got := resource.Status.SensitiveDataModelId; got != modelID {
		t.Fatalf("status.sensitiveDataModelId = %q, want %q", got, modelID)
	}
	if got := resource.Status.LifecycleState; got != string(state) {
		t.Fatalf("status.lifecycleState = %q, want %q", got, state)
	}
}

func requireSensitiveColumnConfidenceLevelDetails(t *testing.T, resource *datasafev1beta1.SensitiveColumn) {
	t.Helper()
	if len(resource.Status.ConfidenceLevelDetails) != 1 {
		t.Fatalf("status.confidenceLevelDetails length = %d, want 1", len(resource.Status.ConfidenceLevelDetails))
	}
	var decoded map[string][]map[string]string
	if err := json.Unmarshal(resource.Status.ConfidenceLevelDetails[0].Raw, &decoded); err != nil {
		t.Fatalf("status.confidenceLevelDetails[0] unmarshal error = %v", err)
	}
	if got := decoded["Operations"][0]["CostCenter"]; got != "42" {
		t.Fatalf("status.confidenceLevelDetails[0].Operations[0].CostCenter = %q, want 42", got)
	}
}

func requireSensitiveColumnOpcRequestID(t *testing.T, resource *datasafev1beta1.SensitiveColumn, want string) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func requireSensitiveColumnAsyncCurrent(
	t *testing.T,
	resource *datasafev1beta1.SensitiveColumn,
	wantPhase shared.OSOKAsyncPhase,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil")
	}
	if current.Phase != wantPhase || current.NormalizedClass != wantClass {
		t.Fatalf("status.async.current phase/class = %q/%q, want %q/%q", current.Phase, current.NormalizedClass, wantPhase, wantClass)
	}
	if current.WorkRequestID == "" && wantPhase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current.workRequestId = empty, want tracked work request")
	}
}

func requireSensitiveColumnString(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func requireSensitiveColumnStringSlice(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", label, got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("%s = %#v, want %#v", label, got, want)
		}
	}
}

func containsSensitiveColumnTestString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
