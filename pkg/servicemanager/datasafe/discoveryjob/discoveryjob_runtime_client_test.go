/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package discoveryjob

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testDiscoveryJobID              = "ocid1.datasafediscoveryjob.oc1..job"
	testDiscoveryJobCompartmentID   = "ocid1.compartment.oc1..discovery"
	testDiscoveryJobSensitiveDataID = "ocid1.datasafesensitivedatamodel.oc1..model"
	testDiscoveryJobTargetID        = "ocid1.datasafetargetdatabase.oc1..target"
	testDiscoveryJobDisplayName     = "customer-discovery"
)

type fakeDiscoveryJobOCIClient struct {
	createFn func(context.Context, datasafesdk.CreateDiscoveryJobRequest) (datasafesdk.CreateDiscoveryJobResponse, error)
	getFn    func(context.Context, datasafesdk.GetDiscoveryJobRequest) (datasafesdk.GetDiscoveryJobResponse, error)
	listFn   func(context.Context, datasafesdk.ListDiscoveryJobsRequest) (datasafesdk.ListDiscoveryJobsResponse, error)
	deleteFn func(context.Context, datasafesdk.DeleteDiscoveryJobRequest) (datasafesdk.DeleteDiscoveryJobResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	deleteCalls int
}

func (f *fakeDiscoveryJobOCIClient) CreateDiscoveryJob(
	ctx context.Context,
	request datasafesdk.CreateDiscoveryJobRequest,
) (datasafesdk.CreateDiscoveryJobResponse, error) {
	f.createCalls++
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return datasafesdk.CreateDiscoveryJobResponse{}, nil
}

func (f *fakeDiscoveryJobOCIClient) GetDiscoveryJob(
	ctx context.Context,
	request datasafesdk.GetDiscoveryJobRequest,
) (datasafesdk.GetDiscoveryJobResponse, error) {
	f.getCalls++
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return datasafesdk.GetDiscoveryJobResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "DiscoveryJob is missing")
}

func (f *fakeDiscoveryJobOCIClient) ListDiscoveryJobs(
	ctx context.Context,
	request datasafesdk.ListDiscoveryJobsRequest,
) (datasafesdk.ListDiscoveryJobsResponse, error) {
	f.listCalls++
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return datasafesdk.ListDiscoveryJobsResponse{}, nil
}

func (f *fakeDiscoveryJobOCIClient) DeleteDiscoveryJob(
	ctx context.Context,
	request datasafesdk.DeleteDiscoveryJobRequest,
) (datasafesdk.DeleteDiscoveryJobResponse, error) {
	f.deleteCalls++
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return datasafesdk.DeleteDiscoveryJobResponse{}, nil
}

func TestDiscoveryJobRuntimeHooksConfigured(t *testing.T) {
	hooks := newDiscoveryJobDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	applyDiscoveryJobRuntimeHooks(&hooks)

	checks := []struct {
		name string
		ok   bool
	}{
		{name: "Semantics", ok: hooks.Semantics != nil},
		{name: "BuildCreateBody", ok: hooks.BuildCreateBody != nil},
		{name: "Identity.Resolve", ok: hooks.Identity.Resolve != nil},
		{name: "Identity.RecordPath", ok: hooks.Identity.RecordPath != nil},
		{name: "DeleteHooks.HandleError", ok: hooks.DeleteHooks.HandleError != nil},
		{name: "DeleteHooks.ApplyOutcome", ok: hooks.DeleteHooks.ApplyOutcome != nil},
		{name: "StatusHooks.ProjectStatus", ok: hooks.StatusHooks.ProjectStatus != nil},
		{name: "StatusHooks.MarkTerminating", ok: hooks.StatusHooks.MarkTerminating != nil},
		{name: "WrapGeneratedClient", ok: len(hooks.WrapGeneratedClient) > 0},
	}
	for _, check := range checks {
		if !check.ok {
			t.Fatalf("hooks.%s not configured", check.name)
		}
	}

	body, err := hooks.BuildCreateBody(context.Background(), makeDiscoveryJobResource(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(datasafesdk.CreateDiscoveryJobDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want CreateDiscoveryJobDetails", body)
	}
	requireStringPtr(t, "CreateDiscoveryJobDetails.SensitiveDataModelId", details.SensitiveDataModelId, testDiscoveryJobSensitiveDataID)
	requireStringPtr(t, "CreateDiscoveryJobDetails.CompartmentId", details.CompartmentId, testDiscoveryJobCompartmentID)
	requireBoolPtr(t, "CreateDiscoveryJobDetails.IsSampleDataCollectionEnabled", details.IsSampleDataCollectionEnabled, false)
	requireBoolPtr(t, "CreateDiscoveryJobDetails.IsAppDefinedRelationDiscoveryEnabled", details.IsAppDefinedRelationDiscoveryEnabled, true)
	requireTablesForDiscovery(t, details.TablesForDiscovery)
}

func TestDiscoveryJobCreateRecordsIdentityRequestIDAndLifecycle(t *testing.T) {
	resource := makeDiscoveryJobResource()
	created := sdkDiscoveryJob(resource, testDiscoveryJobID, datasafesdk.DiscoveryLifecycleStateCreating)
	client := &fakeDiscoveryJobOCIClient{
		createFn: func(_ context.Context, request datasafesdk.CreateDiscoveryJobRequest) (datasafesdk.CreateDiscoveryJobResponse, error) {
			requireDiscoveryJobCreateRequest(t, request, resource)
			return datasafesdk.CreateDiscoveryJobResponse{
				DiscoveryJob: created,
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetDiscoveryJobRequest) (datasafesdk.GetDiscoveryJobResponse, error) {
			requireStringPtr(t, "GetDiscoveryJobRequest.DiscoveryJobId", request.DiscoveryJobId, testDiscoveryJobID)
			return datasafesdk.GetDiscoveryJobResponse{DiscoveryJob: created}, nil
		},
	}

	response, err := newTestDiscoveryJobClient(client).CreateOrUpdate(context.Background(), resource, discoveryJobRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want requeue while CREATING")
	}
	assertDiscoveryJobCallCount(t, "ListDiscoveryJobs()", client.listCalls, 1)
	assertDiscoveryJobCallCount(t, "CreateDiscoveryJob()", client.createCalls, 1)
	assertDiscoveryJobCallCount(t, "GetDiscoveryJob()", client.getCalls, 1)
	assertDiscoveryJobRecordedID(t, resource, testDiscoveryJobID)
	assertDiscoveryJobOpcRequestID(t, resource, "opc-create")
	if got := resource.Status.SensitiveDataModelId; got != testDiscoveryJobSensitiveDataID {
		t.Fatalf("status.sensitiveDataModelId = %q, want %q", got, testDiscoveryJobSensitiveDataID)
	}
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseCreate {
		t.Fatalf("status.status.async.current = %#v, want create lifecycle tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestDiscoveryJobCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	resource := makeDiscoveryJobResource()
	existing := sdkDiscoveryJob(resource, testDiscoveryJobID, datasafesdk.DiscoveryLifecycleStateActive)
	var pages []string
	client := &fakeDiscoveryJobOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListDiscoveryJobsRequest) (datasafesdk.ListDiscoveryJobsResponse, error) {
			pages = append(pages, stringValue(request.Page))
			requireStringPtr(t, "ListDiscoveryJobsRequest.CompartmentId", request.CompartmentId, testDiscoveryJobCompartmentID)
			requireStringPtr(t, "ListDiscoveryJobsRequest.DisplayName", request.DisplayName, testDiscoveryJobDisplayName)
			requireStringPtr(t, "ListDiscoveryJobsRequest.SensitiveDataModelId", request.SensitiveDataModelId, testDiscoveryJobSensitiveDataID)
			if request.Page == nil {
				return datasafesdk.ListDiscoveryJobsResponse{
					DiscoveryJobCollection: datasafesdk.DiscoveryJobCollection{
						Items: []datasafesdk.DiscoveryJobSummary{
							sdkDiscoveryJobSummary(resource, "ocid1.datasafediscoveryjob.oc1..other", "other-discovery", datasafesdk.DiscoveryLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return datasafesdk.ListDiscoveryJobsResponse{
				DiscoveryJobCollection: datasafesdk.DiscoveryJobCollection{
					Items: []datasafesdk.DiscoveryJobSummary{
						sdkDiscoveryJobSummary(resource, testDiscoveryJobID, testDiscoveryJobDisplayName, datasafesdk.DiscoveryLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetDiscoveryJobRequest) (datasafesdk.GetDiscoveryJobResponse, error) {
			requireStringPtr(t, "GetDiscoveryJobRequest.DiscoveryJobId", request.DiscoveryJobId, testDiscoveryJobID)
			return datasafesdk.GetDiscoveryJobResponse{DiscoveryJob: existing}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateDiscoveryJobRequest) (datasafesdk.CreateDiscoveryJobResponse, error) {
			t.Fatal("CreateDiscoveryJob() called despite existing list match")
			return datasafesdk.CreateDiscoveryJobResponse{}, nil
		},
	}

	response, err := newTestDiscoveryJobClient(client).CreateOrUpdate(context.Background(), resource, discoveryJobRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got := strings.Join(pages, ","); got != ",page-2" {
		t.Fatalf("ListDiscoveryJobs() pages = %q, want \",page-2\"", got)
	}
	assertDiscoveryJobRecordedID(t, resource, testDiscoveryJobID)
	assertDiscoveryJobCallCount(t, "CreateDiscoveryJob()", client.createCalls, 0)
}

func TestDiscoveryJobCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	resource := makeDiscoveryJobResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDiscoveryJobID)
	current := sdkDiscoveryJob(resource, testDiscoveryJobID, datasafesdk.DiscoveryLifecycleStateActive)
	client := &fakeDiscoveryJobOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetDiscoveryJobRequest) (datasafesdk.GetDiscoveryJobResponse, error) {
			requireStringPtr(t, "GetDiscoveryJobRequest.DiscoveryJobId", request.DiscoveryJobId, testDiscoveryJobID)
			return datasafesdk.GetDiscoveryJobResponse{DiscoveryJob: current}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateDiscoveryJobRequest) (datasafesdk.CreateDiscoveryJobResponse, error) {
			t.Fatal("CreateDiscoveryJob() called during no-op reconcile")
			return datasafesdk.CreateDiscoveryJobResponse{}, nil
		},
	}

	response, err := newTestDiscoveryJobClient(client).CreateOrUpdate(context.Background(), resource, discoveryJobRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertDiscoveryJobCallCount(t, "GetDiscoveryJob()", client.getCalls, 1)
	assertDiscoveryJobCallCount(t, "CreateDiscoveryJob()", client.createCalls, 0)
	requireLastCondition(t, resource, shared.Active)
}

func TestDiscoveryJobTrackedIdentityDriftRejectedBeforeOCI(t *testing.T) {
	resource := makeDiscoveryJobResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDiscoveryJobID)
	resource.Status.Id = testDiscoveryJobID
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.SensitiveDataModelId = resource.Spec.SensitiveDataModelId
	resource.Status.DisplayName = resource.Spec.DisplayName
	resource.Status.DiscoveryType = resource.Spec.DiscoveryType
	resource.Spec.DisplayName = "renamed-in-cr"
	client := &fakeDiscoveryJobOCIClient{
		getFn: func(context.Context, datasafesdk.GetDiscoveryJobRequest) (datasafesdk.GetDiscoveryJobResponse, error) {
			t.Fatal("GetDiscoveryJob() called despite tracked identity drift")
			return datasafesdk.GetDiscoveryJobResponse{}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateDiscoveryJobRequest) (datasafesdk.CreateDiscoveryJobResponse, error) {
			t.Fatal("CreateDiscoveryJob() called despite tracked identity drift")
			return datasafesdk.CreateDiscoveryJobResponse{}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteDiscoveryJobRequest) (datasafesdk.DeleteDiscoveryJobResponse, error) {
			t.Fatal("DeleteDiscoveryJob() called during create/update drift handling")
			return datasafesdk.DeleteDiscoveryJobResponse{}, nil
		},
	}

	_, err := newTestDiscoveryJobClient(client).CreateOrUpdate(context.Background(), resource, discoveryJobRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want tracked identity drift rejection")
	}
	if !strings.Contains(err.Error(), "displayName") {
		t.Fatalf("CreateOrUpdate() error = %v, want displayName drift detail", err)
	}
	assertDiscoveryJobCallCount(t, "GetDiscoveryJob()", client.getCalls, 0)
	assertDiscoveryJobCallCount(t, "CreateDiscoveryJob()", client.createCalls, 0)
	assertDiscoveryJobCallCount(t, "DeleteDiscoveryJob()", client.deleteCalls, 0)
	requireLastCondition(t, resource, shared.Failed)
}

func TestDiscoveryJobNoUpdateDriftRejectedBeforeMutatingOCI(t *testing.T) {
	resource := makeDiscoveryJobResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDiscoveryJobID)
	currentResource := makeDiscoveryJobResource()
	current := sdkDiscoveryJob(currentResource, testDiscoveryJobID, datasafesdk.DiscoveryLifecycleStateActive)
	resource.Spec.IsIncludeAllSensitiveTypes = false
	client := &fakeDiscoveryJobOCIClient{
		getFn: func(context.Context, datasafesdk.GetDiscoveryJobRequest) (datasafesdk.GetDiscoveryJobResponse, error) {
			return datasafesdk.GetDiscoveryJobResponse{DiscoveryJob: current}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateDiscoveryJobRequest) (datasafesdk.CreateDiscoveryJobResponse, error) {
			t.Fatal("CreateDiscoveryJob() called despite existing DiscoveryJob drift")
			return datasafesdk.CreateDiscoveryJobResponse{}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteDiscoveryJobRequest) (datasafesdk.DeleteDiscoveryJobResponse, error) {
			t.Fatal("DeleteDiscoveryJob() called during no-update drift handling")
			return datasafesdk.DeleteDiscoveryJobResponse{}, nil
		},
	}

	_, err := newTestDiscoveryJobClient(client).CreateOrUpdate(context.Background(), resource, discoveryJobRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want no-update drift rejection")
	}
	if !strings.Contains(err.Error(), "isIncludeAllSensitiveTypes") {
		t.Fatalf("CreateOrUpdate() error = %v, want isIncludeAllSensitiveTypes drift detail", err)
	}
	assertDiscoveryJobCallCount(t, "CreateDiscoveryJob()", client.createCalls, 0)
	assertDiscoveryJobCallCount(t, "DeleteDiscoveryJob()", client.deleteCalls, 0)
}

func TestDiscoveryJobDeleteRetainsFinalizerWhileReadbackStillActive(t *testing.T) {
	resource := makeDiscoveryJobResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDiscoveryJobID)
	active := sdkDiscoveryJob(resource, testDiscoveryJobID, datasafesdk.DiscoveryLifecycleStateActive)
	getResponses := []datasafesdk.GetDiscoveryJobResponse{
		{DiscoveryJob: active},
		{DiscoveryJob: active},
		{DiscoveryJob: active},
	}
	client := &fakeDiscoveryJobOCIClient{
		getFn: getDiscoveryJobResponses(t, &getResponses),
		deleteFn: func(_ context.Context, request datasafesdk.DeleteDiscoveryJobRequest) (datasafesdk.DeleteDiscoveryJobResponse, error) {
			requireStringPtr(t, "DeleteDiscoveryJobRequest.DiscoveryJobId", request.DiscoveryJobId, testDiscoveryJobID)
			return datasafesdk.DeleteDiscoveryJobResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestDiscoveryJobClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	assertDiscoveryJobCallCount(t, "DeleteDiscoveryJob()", client.deleteCalls, 1)
	assertDiscoveryJobOpcRequestID(t, resource, "opc-delete")
	requireLastCondition(t, resource, shared.Terminating)
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current = %#v, want delete lifecycle tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestDiscoveryJobDeleteByTrackedOCIDIgnoresSpecDrift(t *testing.T) {
	resource := makeDiscoveryJobResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDiscoveryJobID)
	resource.Status.Id = testDiscoveryJobID
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Status.SensitiveDataModelId = resource.Spec.SensitiveDataModelId
	resource.Status.DisplayName = resource.Spec.DisplayName
	resource.Status.DiscoveryType = resource.Spec.DiscoveryType
	resource.Spec.DisplayName = "renamed-in-cr"
	active := sdkDiscoveryJob(resource, testDiscoveryJobID, datasafesdk.DiscoveryLifecycleStateActive)
	deletedJob := sdkDiscoveryJob(resource, testDiscoveryJobID, datasafesdk.DiscoveryLifecycleStateDeleted)
	getResponses := []datasafesdk.GetDiscoveryJobResponse{
		{DiscoveryJob: active},
		{DiscoveryJob: active},
		{DiscoveryJob: deletedJob},
	}
	client := &fakeDiscoveryJobOCIClient{
		getFn: getDiscoveryJobResponses(t, &getResponses),
		listFn: func(context.Context, datasafesdk.ListDiscoveryJobsRequest) (datasafesdk.ListDiscoveryJobsResponse, error) {
			t.Fatal("ListDiscoveryJobs() called despite tracked OCID")
			return datasafesdk.ListDiscoveryJobsResponse{}, nil
		},
		deleteFn: func(_ context.Context, request datasafesdk.DeleteDiscoveryJobRequest) (datasafesdk.DeleteDiscoveryJobResponse, error) {
			requireStringPtr(t, "DeleteDiscoveryJobRequest.DiscoveryJobId", request.DiscoveryJobId, testDiscoveryJobID)
			return datasafesdk.DeleteDiscoveryJobResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestDiscoveryJobClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after deleted readback")
	}
	assertDiscoveryJobCallCount(t, "DeleteDiscoveryJob()", client.deleteCalls, 1)
	assertDiscoveryJobOpcRequestID(t, resource, "opc-delete")
}

func TestDiscoveryJobDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	resource := makeDiscoveryJobResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testDiscoveryJobID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
	authErr.OpcRequestID = "opc-auth"
	client := &fakeDiscoveryJobOCIClient{
		getFn: func(context.Context, datasafesdk.GetDiscoveryJobRequest) (datasafesdk.GetDiscoveryJobResponse, error) {
			return datasafesdk.GetDiscoveryJobResponse{}, authErr
		},
		deleteFn: func(context.Context, datasafesdk.DeleteDiscoveryJobRequest) (datasafesdk.DeleteDiscoveryJobResponse, error) {
			t.Fatal("DeleteDiscoveryJob() called after ambiguous pre-delete read")
			return datasafesdk.DeleteDiscoveryJobResponse{}, nil
		},
	}

	deleted, err := newTestDiscoveryJobClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not found")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want auth-shaped detail", err)
	}
	assertDiscoveryJobCallCount(t, "DeleteDiscoveryJob()", client.deleteCalls, 0)
	assertDiscoveryJobOpcRequestID(t, resource, "opc-auth")
}

func TestDiscoveryJobCreateOrUpdateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := makeDiscoveryJobResource()
	createErr := errortest.NewServiceError(500, "InternalError", "create failed")
	createErr.OpcRequestID = "opc-create-error"
	client := &fakeDiscoveryJobOCIClient{
		createFn: func(context.Context, datasafesdk.CreateDiscoveryJobRequest) (datasafesdk.CreateDiscoveryJobResponse, error) {
			return datasafesdk.CreateDiscoveryJobResponse{}, createErr
		},
	}

	_, err := newTestDiscoveryJobClient(client).CreateOrUpdate(context.Background(), resource, discoveryJobRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	assertDiscoveryJobOpcRequestID(t, resource, "opc-create-error")
	requireLastCondition(t, resource, shared.Failed)
}

func newTestDiscoveryJobClient(client discoveryJobOCIClient) DiscoveryJobServiceClient {
	return newDiscoveryJobServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeDiscoveryJobResource() *datasafev1beta1.DiscoveryJob {
	return &datasafev1beta1.DiscoveryJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "discovery-job",
			Namespace: "default",
		},
		Spec: datasafev1beta1.DiscoveryJobSpec{
			SensitiveDataModelId:                 testDiscoveryJobSensitiveDataID,
			CompartmentId:                        testDiscoveryJobCompartmentID,
			DiscoveryType:                        string(datasafesdk.DiscoveryJobDiscoveryTypeAll),
			DisplayName:                          testDiscoveryJobDisplayName,
			SchemasForDiscovery:                  []string{"APP"},
			TablesForDiscovery:                   []datasafev1beta1.DiscoveryJobTablesForDiscovery{{SchemaName: "APP", TableNames: []string{"CUSTOMERS", "ORDERS"}}},
			SensitiveTypeIdsForDiscovery:         []string{"ocid1.datasafesensitivetype.oc1..type"},
			SensitiveTypeGroupIdsForDiscovery:    []string{"ocid1.datasafesensitivetypegroup.oc1..group"},
			IsSampleDataCollectionEnabled:        false,
			IsAppDefinedRelationDiscoveryEnabled: true,
			IsIncludeAllSchemas:                  false,
			IsIncludeAllSensitiveTypes:           true,
			FreeformTags:                         map[string]string{"owner": "runtime"},
			DefinedTags:                          map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func discoveryJobRequest(resource *datasafev1beta1.DiscoveryJob) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: resource.Name, Namespace: resource.Namespace}}
}

func sdkDiscoveryJob(
	resource *datasafev1beta1.DiscoveryJob,
	id string,
	lifecycleState datasafesdk.DiscoveryLifecycleStateEnum,
) datasafesdk.DiscoveryJob {
	started := common.SDKTime{Time: time.Date(2026, time.May, 5, 10, 0, 0, 0, time.UTC)}
	finished := common.SDKTime{Time: time.Date(2026, time.May, 5, 10, 5, 0, 0, time.UTC)}
	return datasafesdk.DiscoveryJob{
		Id:                                   common.String(id),
		DiscoveryType:                        datasafesdk.DiscoveryJobDiscoveryTypeEnum(resource.Spec.DiscoveryType),
		DisplayName:                          common.String(resource.Spec.DisplayName),
		CompartmentId:                        common.String(resource.Spec.CompartmentId),
		TimeStarted:                          &started,
		TimeFinished:                         &finished,
		LifecycleState:                       lifecycleState,
		SensitiveDataModelId:                 common.String(resource.Spec.SensitiveDataModelId),
		TargetId:                             common.String(testDiscoveryJobTargetID),
		IsSampleDataCollectionEnabled:        common.Bool(resource.Spec.IsSampleDataCollectionEnabled),
		IsAppDefinedRelationDiscoveryEnabled: common.Bool(resource.Spec.IsAppDefinedRelationDiscoveryEnabled),
		IsIncludeAllSchemas:                  common.Bool(resource.Spec.IsIncludeAllSchemas),
		IsIncludeAllSensitiveTypes:           common.Bool(resource.Spec.IsIncludeAllSensitiveTypes),
		TotalSchemasScanned:                  common.Int64(2),
		TotalObjectsScanned:                  common.Int64(3),
		TotalColumnsScanned:                  common.Int64(4),
		TotalNewSensitiveColumns:             common.Int64(5),
		TotalModifiedSensitiveColumns:        common.Int64(6),
		TotalDeletedSensitiveColumns:         common.Int64(7),
		SchemasForDiscovery:                  discoveryJobStringSlice(resource.Spec.SchemasForDiscovery),
		TablesForDiscovery:                   discoveryJobTablesForDiscoveryFromSpec(resource.Spec.TablesForDiscovery),
		SensitiveTypeIdsForDiscovery:         discoveryJobStringSlice(resource.Spec.SensitiveTypeIdsForDiscovery),
		SensitiveTypeGroupIdsForDiscovery:    discoveryJobStringSlice(resource.Spec.SensitiveTypeGroupIdsForDiscovery),
		FreeformTags:                         discoveryJobStringMap(resource.Spec.FreeformTags),
		DefinedTags:                          discoveryJobDefinedTags(resource.Spec.DefinedTags),
		SystemTags:                           map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "false"}},
	}
}

func sdkDiscoveryJobSummary(
	resource *datasafev1beta1.DiscoveryJob,
	id string,
	displayName string,
	lifecycleState datasafesdk.DiscoveryLifecycleStateEnum,
) datasafesdk.DiscoveryJobSummary {
	started := common.SDKTime{Time: time.Date(2026, time.May, 5, 10, 0, 0, 0, time.UTC)}
	finished := common.SDKTime{Time: time.Date(2026, time.May, 5, 10, 5, 0, 0, time.UTC)}
	return datasafesdk.DiscoveryJobSummary{
		Id:                   common.String(id),
		DisplayName:          common.String(displayName),
		TimeStarted:          &started,
		TimeFinished:         &finished,
		SensitiveDataModelId: common.String(resource.Spec.SensitiveDataModelId),
		TargetId:             common.String(testDiscoveryJobTargetID),
		LifecycleState:       lifecycleState,
		DiscoveryType:        datasafesdk.DiscoveryJobDiscoveryTypeEnum(resource.Spec.DiscoveryType),
		CompartmentId:        common.String(resource.Spec.CompartmentId),
		FreeformTags:         discoveryJobStringMap(resource.Spec.FreeformTags),
		DefinedTags:          discoveryJobDefinedTags(resource.Spec.DefinedTags),
	}
}

func getDiscoveryJobResponses(
	t *testing.T,
	responses *[]datasafesdk.GetDiscoveryJobResponse,
) func(context.Context, datasafesdk.GetDiscoveryJobRequest) (datasafesdk.GetDiscoveryJobResponse, error) {
	t.Helper()
	return func(_ context.Context, request datasafesdk.GetDiscoveryJobRequest) (datasafesdk.GetDiscoveryJobResponse, error) {
		requireStringPtr(t, "GetDiscoveryJobRequest.DiscoveryJobId", request.DiscoveryJobId, testDiscoveryJobID)
		if len(*responses) == 0 {
			return datasafesdk.GetDiscoveryJobResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "DiscoveryJob is gone")
		}
		response := (*responses)[0]
		*responses = (*responses)[1:]
		return response, nil
	}
}

func requireDiscoveryJobCreateRequest(
	t *testing.T,
	request datasafesdk.CreateDiscoveryJobRequest,
	resource *datasafev1beta1.DiscoveryJob,
) {
	t.Helper()
	requireStringPtr(t, "CreateDiscoveryJobDetails.SensitiveDataModelId", request.SensitiveDataModelId, resource.Spec.SensitiveDataModelId)
	requireStringPtr(t, "CreateDiscoveryJobDetails.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "CreateDiscoveryJobDetails.DisplayName", request.DisplayName, resource.Spec.DisplayName)
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateDiscoveryJobRequest.OpcRetryToken is empty")
	}
	if request.DiscoveryType != datasafesdk.DiscoveryJobDiscoveryTypeEnum(resource.Spec.DiscoveryType) {
		t.Fatalf("CreateDiscoveryJobDetails.DiscoveryType = %q, want %q", request.DiscoveryType, resource.Spec.DiscoveryType)
	}
	requireBoolPtr(t, "CreateDiscoveryJobDetails.IsSampleDataCollectionEnabled", request.IsSampleDataCollectionEnabled, resource.Spec.IsSampleDataCollectionEnabled)
	requireBoolPtr(t, "CreateDiscoveryJobDetails.IsAppDefinedRelationDiscoveryEnabled", request.IsAppDefinedRelationDiscoveryEnabled, resource.Spec.IsAppDefinedRelationDiscoveryEnabled)
	requireBoolPtr(t, "CreateDiscoveryJobDetails.IsIncludeAllSchemas", request.IsIncludeAllSchemas, resource.Spec.IsIncludeAllSchemas)
	requireBoolPtr(t, "CreateDiscoveryJobDetails.IsIncludeAllSensitiveTypes", request.IsIncludeAllSensitiveTypes, resource.Spec.IsIncludeAllSensitiveTypes)
	requireTablesForDiscovery(t, request.TablesForDiscovery)
}

func requireTablesForDiscovery(t *testing.T, got []datasafesdk.TablesForDiscovery) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("TablesForDiscovery length = %d, want 1", len(got))
	}
	requireStringPtr(t, "TablesForDiscovery[0].SchemaName", got[0].SchemaName, "APP")
	if strings.Join(got[0].TableNames, ",") != "CUSTOMERS,ORDERS" {
		t.Fatalf("TablesForDiscovery[0].TableNames = %#v, want CUSTOMERS, ORDERS", got[0].TableNames)
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

func requireBoolPtr(t *testing.T, name string, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %t", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %t, want %t", name, *got, want)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func assertDiscoveryJobRecordedID(t *testing.T, resource *datasafev1beta1.DiscoveryJob, want string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
}

func assertDiscoveryJobOpcRequestID(t *testing.T, resource *datasafev1beta1.DiscoveryJob, want string) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func assertDiscoveryJobCallCount(t *testing.T, name string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s calls = %d, want %d", name, got, want)
	}
}

func requireLastCondition(
	t *testing.T,
	resource *datasafev1beta1.DiscoveryJob,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions = nil, want %s", want)
	}
	last := conditions[len(conditions)-1]
	if last.Type != want {
		t.Fatalf("last condition type = %s, want %s", last.Type, want)
	}
	if want == shared.Failed && last.Status != corev1.ConditionFalse {
		t.Fatalf("last condition status = %s, want False for Failed", last.Status)
	}
}
