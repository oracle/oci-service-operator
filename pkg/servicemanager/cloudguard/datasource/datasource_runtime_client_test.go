/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package datasource

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testDataSourceCompartmentID = "ocid1.compartment.oc1..datasourcecompartment"
	testDataSourceID            = "ocid1.cloudguarddatasource.oc1..datasource"
	testDataSourceOtherID       = "ocid1.cloudguarddatasource.oc1..other"
	testDataSourceDisplayName   = "runtime-datasource"
)

type fakeDataSourceOCIClient struct {
	createFunc func(context.Context, cloudguardsdk.CreateDataSourceRequest) (cloudguardsdk.CreateDataSourceResponse, error)
	getFunc    func(context.Context, cloudguardsdk.GetDataSourceRequest) (cloudguardsdk.GetDataSourceResponse, error)
	listFunc   func(context.Context, cloudguardsdk.ListDataSourcesRequest) (cloudguardsdk.ListDataSourcesResponse, error)
	updateFunc func(context.Context, cloudguardsdk.UpdateDataSourceRequest) (cloudguardsdk.UpdateDataSourceResponse, error)
	deleteFunc func(context.Context, cloudguardsdk.DeleteDataSourceRequest) (cloudguardsdk.DeleteDataSourceResponse, error)
	workFunc   func(context.Context, cloudguardsdk.GetWorkRequestRequest) (cloudguardsdk.GetWorkRequestResponse, error)

	createRequests []cloudguardsdk.CreateDataSourceRequest
	getRequests    []cloudguardsdk.GetDataSourceRequest
	listRequests   []cloudguardsdk.ListDataSourcesRequest
	updateRequests []cloudguardsdk.UpdateDataSourceRequest
	deleteRequests []cloudguardsdk.DeleteDataSourceRequest
	workRequests   []cloudguardsdk.GetWorkRequestRequest
}

func (f *fakeDataSourceOCIClient) CreateDataSource(
	ctx context.Context,
	request cloudguardsdk.CreateDataSourceRequest,
) (cloudguardsdk.CreateDataSourceResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return cloudguardsdk.CreateDataSourceResponse{}, nil
}

func (f *fakeDataSourceOCIClient) GetDataSource(
	ctx context.Context,
	request cloudguardsdk.GetDataSourceRequest,
) (cloudguardsdk.GetDataSourceResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return cloudguardsdk.GetDataSourceResponse{}, nil
}

func (f *fakeDataSourceOCIClient) ListDataSources(
	ctx context.Context,
	request cloudguardsdk.ListDataSourcesRequest,
) (cloudguardsdk.ListDataSourcesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return cloudguardsdk.ListDataSourcesResponse{}, nil
}

func (f *fakeDataSourceOCIClient) UpdateDataSource(
	ctx context.Context,
	request cloudguardsdk.UpdateDataSourceRequest,
) (cloudguardsdk.UpdateDataSourceResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return cloudguardsdk.UpdateDataSourceResponse{}, nil
}

func (f *fakeDataSourceOCIClient) DeleteDataSource(
	ctx context.Context,
	request cloudguardsdk.DeleteDataSourceRequest,
) (cloudguardsdk.DeleteDataSourceResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return cloudguardsdk.DeleteDataSourceResponse{}, nil
}

func (f *fakeDataSourceOCIClient) GetWorkRequest(
	ctx context.Context,
	request cloudguardsdk.GetWorkRequestRequest,
) (cloudguardsdk.GetWorkRequestResponse, error) {
	f.workRequests = append(f.workRequests, request)
	if f.workFunc != nil {
		return f.workFunc(ctx, request)
	}
	return cloudguardsdk.GetWorkRequestResponse{
		WorkRequest: dataSourceWorkRequest(
			stringValue(request.WorkRequestId),
			dataSourceOperationTypeForWorkRequestID(stringValue(request.WorkRequestId)),
			cloudguardsdk.OperationStatusSucceeded,
			testDataSourceID,
		),
	}, nil
}

//nolint:gocyclo // This single contract test keeps the generatedruntime semantic surface together.
func TestDataSourceRuntimeHooksConfigured(t *testing.T) {
	t.Parallel()

	hooks := newDataSourceRuntimeHooksWithOCIClient(nil)
	applyDataSourceRuntimeHooks(nil, &hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed runtime semantics")
	}
	if hooks.Semantics.Async == nil || hooks.Semantics.Async.Strategy != "workrequest" {
		t.Fatalf("hooks.Semantics.Async = %#v, want workrequest strategy", hooks.Semantics.Async)
	}
	if hooks.Semantics.Async.WorkRequest == nil {
		t.Fatal("hooks.Semantics.Async.WorkRequest = nil, want generated work request semantics")
	}
	if got := hooks.Semantics.FinalizerPolicy; got != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got)
	}
	assertDataSourceContainsAll(t, "Lifecycle.ActiveStates", hooks.Semantics.Lifecycle.ActiveStates, "ACTIVE", "INACTIVE")
	assertDataSourceContainsAll(t, "Delete.PendingStates", hooks.Semantics.Delete.PendingStates, "DELETING")
	assertDataSourceContainsAll(t, "Delete.TerminalStates", hooks.Semantics.Delete.TerminalStates, "DELETED")
	assertDataSourceContainsAll(t, "List.MatchFields", hooks.Semantics.List.MatchFields, "compartmentId", "displayName", "dataSourceFeedProvider")
	assertDataSourceContainsAll(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "dataSourceDetails", "definedTags", "displayName", "freeformTags", "status")
	assertDataSourceContainsAll(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "compartmentId", "dataSourceFeedProvider")
	if hooks.BuildCreateBody == nil || hooks.BuildUpdateBody == nil {
		t.Fatal("BuildCreateBody/BuildUpdateBody = nil, want resource-specific body builders")
	}
	if hooks.Read.Get == nil || hooks.Read.List == nil {
		t.Fatal("Read hooks incomplete, want status adapter and paginated list read")
	}
	if hooks.Async.GetWorkRequest == nil || hooks.Async.ResolvePhase == nil || hooks.Async.RecoverResourceID == nil {
		t.Fatal("async hooks incomplete, want Cloud Guard work request observation")
	}
	if hooks.DeleteHooks.HandleError == nil || len(hooks.WrapGeneratedClient) == 0 {
		t.Fatal("delete hooks incomplete, want conservative finalizer-retaining delete wrapper")
	}
}

func TestDataSourceServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := newDataSourceRuntimeTestResource()
	fake := &fakeDataSourceOCIClient{}
	fake.listFunc = func(_ context.Context, request cloudguardsdk.ListDataSourcesRequest) (cloudguardsdk.ListDataSourcesResponse, error) {
		assertDataSourceBindListRequest(t, request)
		return cloudguardsdk.ListDataSourcesResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request cloudguardsdk.CreateDataSourceRequest) (cloudguardsdk.CreateDataSourceResponse, error) {
		assertDataSourceCreateRequest(t, request, resource)
		return cloudguardsdk.CreateDataSourceResponse{
			OpcRequestId:     common.String("opc-create-1"),
			OpcWorkRequestId: common.String("wr-create-1"),
		}, nil
	}
	fake.getFunc = func(_ context.Context, request cloudguardsdk.GetDataSourceRequest) (cloudguardsdk.GetDataSourceResponse, error) {
		assertDataSourceGetRequest(t, request)
		return cloudguardsdk.GetDataSourceResponse{
			DataSource: dataSourceFromSpec(t, testDataSourceID, resource.Spec, cloudguardsdk.LifecycleStateActive),
		}, nil
	}
	client := newDataSourceRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, dataSourceReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertDataSourceCreateOrUpdateSucceeded(t, response)
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateDataSource() calls = %d, want 1", len(fake.createRequests))
	}
	assertDataSourceListStates(t, fake.listRequests)
	if len(fake.workRequests) != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", len(fake.workRequests))
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("GetDataSource() calls = %d, want work-request follow-up read", len(fake.getRequests))
	}
	assertDataSourceTrackedID(t, resource, testDataSourceID)
	if got := resource.Status.Status; got != string(cloudguardsdk.DataSourceStatusEnabled) {
		t.Fatalf("status.sdkStatus = %q, want ENABLED", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
}

func TestDataSourceServiceClientCreateWorkRequestRetainsIdentityWithoutDuplicateCreate(t *testing.T) {
	t.Parallel()

	resource := newDataSourceRuntimeTestResource()
	fake := &fakeDataSourceOCIClient{}
	fake.listFunc = func(_ context.Context, request cloudguardsdk.ListDataSourcesRequest) (cloudguardsdk.ListDataSourcesResponse, error) {
		assertDataSourceBindListRequest(t, request)
		return cloudguardsdk.ListDataSourcesResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request cloudguardsdk.CreateDataSourceRequest) (cloudguardsdk.CreateDataSourceResponse, error) {
		assertDataSourceCreateRequest(t, request, resource)
		return cloudguardsdk.CreateDataSourceResponse{
			OpcRequestId:     common.String("opc-create-1"),
			OpcWorkRequestId: common.String("wr-create-1"),
		}, nil
	}
	fake.workFunc = dataSourceCreateWorkRequestSequence(t, fake)
	fake.getFunc = func(_ context.Context, request cloudguardsdk.GetDataSourceRequest) (cloudguardsdk.GetDataSourceResponse, error) {
		assertDataSourceGetRequest(t, request)
		return cloudguardsdk.GetDataSourceResponse{
			DataSource: dataSourceFromSpec(t, testDataSourceID, resource.Spec, cloudguardsdk.LifecycleStateCreating),
		}, nil
	}
	client := newDataSourceRuntimeTestClient(fake)

	firstResponse, err := client.CreateOrUpdate(context.Background(), resource, dataSourceReconcileRequest(resource))
	if err != nil {
		t.Fatalf("first CreateOrUpdate() error = %v", err)
	}
	if !firstResponse.IsSuccessful || !firstResponse.ShouldRequeue {
		t.Fatalf("first CreateOrUpdate() = %#v, want successful requeue while work request is pending", firstResponse)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateDataSource() calls after first reconcile = %d, want 1", len(fake.createRequests))
	}
	assertDataSourceWorkRequestAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-1")

	secondResponse, err := client.CreateOrUpdate(context.Background(), resource, dataSourceReconcileRequest(resource))
	if err != nil {
		t.Fatalf("second CreateOrUpdate() error = %v", err)
	}
	if !secondResponse.IsSuccessful || !secondResponse.ShouldRequeue {
		t.Fatalf("second CreateOrUpdate() = %#v, want successful requeue while CREATING", secondResponse)
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateDataSource() calls after second reconcile = %d, want no duplicate create", len(fake.createRequests))
	}
	assertDataSourceTrackedID(t, resource, testDataSourceID)
	assertDataSourcePendingLifecycleAsync(t, resource, cloudguardsdk.LifecycleStateCreating, shared.OSOKAsyncPhaseCreate)
	assertDataSourceListStates(t, fake.listRequests)
	if got := len(fake.getRequests); got != 1 {
		t.Fatalf("GetDataSource() calls = %d, want 1 after work request success", got)
	}
}

func TestDataSourceServiceClientBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	resource := newDataSourceRuntimeTestResource()
	fake := &fakeDataSourceOCIClient{}
	fake.listFunc = func(_ context.Context, request cloudguardsdk.ListDataSourcesRequest) (cloudguardsdk.ListDataSourcesResponse, error) {
		assertDataSourceBindListRequest(t, request)
		if request.LifecycleState != cloudguardsdk.ListDataSourcesLifecycleStateActive {
			return cloudguardsdk.ListDataSourcesResponse{}, nil
		}
		switch stringValue(request.Page) {
		case "":
			return cloudguardsdk.ListDataSourcesResponse{
				DataSourceCollection: cloudguardsdk.DataSourceCollection{
					Items: []cloudguardsdk.DataSourceSummary{
						dataSourceSummaryFromSpec(t, testDataSourceOtherID, otherDataSourceSpec(resource.Spec), cloudguardsdk.LifecycleStateActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return cloudguardsdk.ListDataSourcesResponse{
				DataSourceCollection: cloudguardsdk.DataSourceCollection{
					Items: []cloudguardsdk.DataSourceSummary{
						dataSourceSummaryFromSpec(t, testDataSourceID, resource.Spec, cloudguardsdk.LifecycleStateActive),
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected active list page token %q", stringValue(request.Page))
			return cloudguardsdk.ListDataSourcesResponse{}, nil
		}
	}
	fake.getFunc = func(_ context.Context, request cloudguardsdk.GetDataSourceRequest) (cloudguardsdk.GetDataSourceResponse, error) {
		assertDataSourceGetRequest(t, request)
		return cloudguardsdk.GetDataSourceResponse{
			DataSource: dataSourceFromSpec(t, testDataSourceID, resource.Spec, cloudguardsdk.LifecycleStateActive),
		}, nil
	}
	fake.createFunc = func(context.Context, cloudguardsdk.CreateDataSourceRequest) (cloudguardsdk.CreateDataSourceResponse, error) {
		t.Fatal("CreateDataSource() called; want existing DataSource bind")
		return cloudguardsdk.CreateDataSourceResponse{}, nil
	}
	client := newDataSourceRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, dataSourceReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertDataSourceCreateOrUpdateSucceeded(t, response)
	assertDataSourceListStates(t, fake.listRequests)
	if got, want := len(fake.listRequests), len(dataSourceLookupLifecycleStates())+1; got != want {
		t.Fatalf("ListDataSources() calls = %d, want lifecycle fan-out plus active page", got)
	}
	assertDataSourceTrackedID(t, resource, testDataSourceID)
}

func TestDataSourceServiceClientBindsInactiveDataSource(t *testing.T) {
	t.Parallel()

	resource := newDataSourceRuntimeTestResource()
	resource.Spec.Status = string(cloudguardsdk.DataSourceStatusDisabled)
	fake := &fakeDataSourceOCIClient{}
	fake.listFunc = func(_ context.Context, request cloudguardsdk.ListDataSourcesRequest) (cloudguardsdk.ListDataSourcesResponse, error) {
		assertDataSourceBindListRequest(t, request)
		if request.LifecycleState != cloudguardsdk.ListDataSourcesLifecycleStateInactive {
			return cloudguardsdk.ListDataSourcesResponse{}, nil
		}
		return cloudguardsdk.ListDataSourcesResponse{
			DataSourceCollection: cloudguardsdk.DataSourceCollection{
				Items: []cloudguardsdk.DataSourceSummary{
					dataSourceSummaryFromSpec(t, testDataSourceID, resource.Spec, cloudguardsdk.LifecycleStateInactive),
				},
			},
		}, nil
	}
	fake.getFunc = func(_ context.Context, request cloudguardsdk.GetDataSourceRequest) (cloudguardsdk.GetDataSourceResponse, error) {
		assertDataSourceGetRequest(t, request)
		return cloudguardsdk.GetDataSourceResponse{
			DataSource: dataSourceFromSpec(t, testDataSourceID, resource.Spec, cloudguardsdk.LifecycleStateInactive),
		}, nil
	}
	fake.createFunc = func(context.Context, cloudguardsdk.CreateDataSourceRequest) (cloudguardsdk.CreateDataSourceResponse, error) {
		t.Fatal("CreateDataSource() called; want INACTIVE DataSource bind")
		return cloudguardsdk.CreateDataSourceResponse{}, nil
	}
	client := newDataSourceRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, dataSourceReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertDataSourceCreateOrUpdateSucceeded(t, response)
	assertDataSourceListStates(t, fake.listRequests)
	assertDataSourceTrackedID(t, resource, testDataSourceID)
	if got := resource.Status.LifecycleState; got != string(cloudguardsdk.LifecycleStateInactive) {
		t.Fatalf("status.lifecycleState = %q, want INACTIVE", got)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateDataSource() calls = %d, want 0", len(fake.createRequests))
	}
}

func TestDataSourceServiceClientSkipsUpdateWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := newDataSourceRuntimeTestResource()
	trackDataSourceID(resource, testDataSourceID)
	fake := &fakeDataSourceOCIClient{}
	fake.getFunc = func(_ context.Context, request cloudguardsdk.GetDataSourceRequest) (cloudguardsdk.GetDataSourceResponse, error) {
		assertDataSourceGetRequest(t, request)
		return cloudguardsdk.GetDataSourceResponse{
			DataSource: dataSourceFromSpec(t, testDataSourceID, resource.Spec, cloudguardsdk.LifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, cloudguardsdk.UpdateDataSourceRequest) (cloudguardsdk.UpdateDataSourceResponse, error) {
		t.Fatal("UpdateDataSource() called; want no-op reconcile")
		return cloudguardsdk.UpdateDataSourceResponse{}, nil
	}
	client := newDataSourceRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, dataSourceReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertDataSourceCreateOrUpdateSucceeded(t, response)
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateDataSource() calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestDataSourceServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	desired := newDataSourceRuntimeTestResource()
	trackDataSourceID(desired, testDataSourceID)
	currentSpec := desired.Spec
	currentSpec.DisplayName = "old-datasource"
	currentSpec.Status = string(cloudguardsdk.DataSourceStatusDisabled)
	currentSpec.FreeformTags = map[string]string{"env": "old"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "17"}}
	currentSpec.DataSourceDetails.Query = "search old"

	fake := &fakeDataSourceOCIClient{}
	fake.getFunc = func(_ context.Context, request cloudguardsdk.GetDataSourceRequest) (cloudguardsdk.GetDataSourceResponse, error) {
		assertDataSourceGetRequest(t, request)
		switch len(fake.getRequests) {
		case 1:
			return cloudguardsdk.GetDataSourceResponse{
				DataSource: dataSourceFromSpec(t, testDataSourceID, currentSpec, cloudguardsdk.LifecycleStateActive),
			}, nil
		case 2:
			return cloudguardsdk.GetDataSourceResponse{
				DataSource: dataSourceFromSpec(t, testDataSourceID, desired.Spec, cloudguardsdk.LifecycleStateActive),
			}, nil
		default:
			t.Fatalf("unexpected GetDataSource() call %d", len(fake.getRequests))
			return cloudguardsdk.GetDataSourceResponse{}, nil
		}
	}
	fake.updateFunc = func(_ context.Context, request cloudguardsdk.UpdateDataSourceRequest) (cloudguardsdk.UpdateDataSourceResponse, error) {
		assertDataSourceUpdateRequest(t, request, desired)
		return cloudguardsdk.UpdateDataSourceResponse{
			OpcRequestId:     common.String("opc-update-1"),
			OpcWorkRequestId: common.String("wr-update-1"),
		}, nil
	}
	client := newDataSourceRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), desired, dataSourceReconcileRequest(desired))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertDataSourceCreateOrUpdateSucceeded(t, response)
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateDataSource() calls = %d, want 1", len(fake.updateRequests))
	}
	if got := desired.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestDataSourceServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newDataSourceRuntimeTestResource()
	trackDataSourceID(resource, testDataSourceID)
	fake := &fakeDataSourceOCIClient{}
	fake.getFunc = func(context.Context, cloudguardsdk.GetDataSourceRequest) (cloudguardsdk.GetDataSourceResponse, error) {
		currentSpec := resource.Spec
		currentSpec.CompartmentId = "ocid1.compartment.oc1..observed"
		return cloudguardsdk.GetDataSourceResponse{
			DataSource: dataSourceFromSpec(t, testDataSourceID, currentSpec, cloudguardsdk.LifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, cloudguardsdk.UpdateDataSourceRequest) (cloudguardsdk.UpdateDataSourceResponse, error) {
		t.Fatal("UpdateDataSource() called; want create-only drift rejection")
		return cloudguardsdk.UpdateDataSourceResponse{}, nil
	}
	client := newDataSourceRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, dataSourceReconcileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId replacement rejection", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success for create-only drift")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateDataSource() calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestDataSourceServiceClientCreateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
	serviceErr.OpcRequestID = "opc-create-error-1"

	resource := newDataSourceRuntimeTestResource()
	fake := &fakeDataSourceOCIClient{}
	fake.listFunc = func(context.Context, cloudguardsdk.ListDataSourcesRequest) (cloudguardsdk.ListDataSourcesResponse, error) {
		return cloudguardsdk.ListDataSourcesResponse{}, nil
	}
	fake.createFunc = func(context.Context, cloudguardsdk.CreateDataSourceRequest) (cloudguardsdk.CreateDataSourceResponse, error) {
		return cloudguardsdk.CreateDataSourceResponse{}, serviceErr
	}
	client := newDataSourceRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, dataSourceReconcileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want surfaced OCI create error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-error-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-error-1", got)
	}
}

//nolint:gocyclo // Delete lifecycle sequencing is the contract under test.
func TestDataSourceServiceClientDeleteRetainsFinalizerUntilDeleteConfirmed(t *testing.T) {
	t.Parallel()

	resource := newDataSourceRuntimeTestResource()
	trackDataSourceID(resource, testDataSourceID)
	fake := &fakeDataSourceOCIClient{}
	fake.getFunc = func(_ context.Context, request cloudguardsdk.GetDataSourceRequest) (cloudguardsdk.GetDataSourceResponse, error) {
		assertDataSourceGetRequest(t, request)
		switch len(fake.getRequests) {
		case 1:
			return cloudguardsdk.GetDataSourceResponse{
				DataSource: dataSourceFromSpec(t, testDataSourceID, resource.Spec, cloudguardsdk.LifecycleStateActive),
			}, nil
		case 2:
			return cloudguardsdk.GetDataSourceResponse{
				DataSource: dataSourceFromSpec(t, testDataSourceID, resource.Spec, cloudguardsdk.LifecycleStateDeleting),
			}, nil
		default:
			t.Fatalf("unexpected GetDataSource() call %d", len(fake.getRequests))
			return cloudguardsdk.GetDataSourceResponse{}, nil
		}
	}
	fake.deleteFunc = func(_ context.Context, request cloudguardsdk.DeleteDataSourceRequest) (cloudguardsdk.DeleteDataSourceResponse, error) {
		if got := stringValue(request.DataSourceId); got != testDataSourceID {
			t.Fatalf("delete dataSourceId = %q, want %q", got, testDataSourceID)
		}
		return cloudguardsdk.DeleteDataSourceResponse{
			OpcRequestId:     common.String("opc-delete-1"),
			OpcWorkRequestId: common.String("wr-delete-1"),
		}, nil
	}
	client := newDataSourceRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while DELETING")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteDataSource() calls = %d, want 1", len(fake.deleteRequests))
	}
	if got := resource.Status.LifecycleState; got != string(cloudguardsdk.LifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", got)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Phase != shared.OSOKAsyncPhaseDelete || current.WorkRequestID != "wr-delete-1" {
		t.Fatalf("status.status.async.current = %#v, want delete work request wr-delete-1", current)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set, want finalizer retained")
	}
}

func TestDataSourceServiceClientDeleteWaitsForPendingWriteLifecycle(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name      string
		state     cloudguardsdk.LifecycleStateEnum
		wantPhase shared.OSOKAsyncPhase
	}{
		{
			name:      "creating",
			state:     cloudguardsdk.LifecycleStateCreating,
			wantPhase: shared.OSOKAsyncPhaseCreate,
		},
		{
			name:      "updating",
			state:     cloudguardsdk.LifecycleStateUpdating,
			wantPhase: shared.OSOKAsyncPhaseUpdate,
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runDataSourcePendingWriteDeleteLifecycleCase(t, tt.state, tt.wantPhase)
		})
	}
}

func runDataSourcePendingWriteDeleteLifecycleCase(
	t *testing.T,
	state cloudguardsdk.LifecycleStateEnum,
	wantPhase shared.OSOKAsyncPhase,
) {
	t.Helper()

	resource := newDataSourceRuntimeTestResource()
	trackDataSourceID(resource, testDataSourceID)
	fake := &fakeDataSourceOCIClient{}
	fake.getFunc = func(_ context.Context, request cloudguardsdk.GetDataSourceRequest) (cloudguardsdk.GetDataSourceResponse, error) {
		assertDataSourceGetRequest(t, request)
		return cloudguardsdk.GetDataSourceResponse{
			DataSource: dataSourceFromSpec(t, testDataSourceID, resource.Spec, state),
		}, nil
	}
	fake.deleteFunc = func(context.Context, cloudguardsdk.DeleteDataSourceRequest) (cloudguardsdk.DeleteDataSourceResponse, error) {
		t.Fatal("DeleteDataSource() called; want pending write lifecycle to retain finalizer")
		return cloudguardsdk.DeleteDataSourceResponse{}, nil
	}
	client := newDataSourceRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteDataSource() calls = %d, want 0", len(fake.deleteRequests))
	}
	if got := resource.Status.LifecycleState; got != string(state) {
		t.Fatalf("status.lifecycleState = %q, want %q", got, state)
	}
	assertDataSourcePendingLifecycleAsync(t, resource, state, wantPhase)
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set, want finalizer retained")
	}
}

func assertDataSourcePendingLifecycleAsync(
	t *testing.T,
	resource *cloudguardv1beta1.DataSource,
	state cloudguardsdk.LifecycleStateEnum,
	wantPhase shared.OSOKAsyncPhase,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want pending lifecycle")
	}
	got := []any{current.Source, current.Phase, current.RawStatus, current.NormalizedClass}
	want := []any{shared.OSOKAsyncSourceLifecycle, wantPhase, string(state), shared.OSOKAsyncClassPending}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("status.status.async.current = %#v, want source/phase/raw/class %#v", current, want)
	}
}

func assertDataSourceWorkRequestAsync(
	t *testing.T,
	resource *cloudguardv1beta1.DataSource,
	wantPhase shared.OSOKAsyncPhase,
	wantWorkRequestID string,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want pending work request")
	}
	got := []any{current.Source, current.Phase, current.WorkRequestID, current.NormalizedClass}
	want := []any{shared.OSOKAsyncSourceWorkRequest, wantPhase, wantWorkRequestID, shared.OSOKAsyncClassPending}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("status.status.async.current = %#v, want source/phase/workRequest/class %#v", current, want)
	}
}

func TestDataSourceServiceClientDeleteObservesRecordedDeleteWorkRequest(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name      string
		status    cloudguardsdk.OperationStatusEnum
		wantClass shared.OSOKAsyncNormalizedClass
		wantErr   bool
	}{
		{
			name:      "pending work request keeps finalizer",
			status:    cloudguardsdk.OperationStatusInProgress,
			wantClass: shared.OSOKAsyncClassPending,
		},
		{
			name:      "succeeded work request waits for readback confirmation",
			status:    cloudguardsdk.OperationStatusSucceeded,
			wantClass: shared.OSOKAsyncClassSucceeded,
		},
		{
			name:      "failed work request surfaces failure",
			status:    cloudguardsdk.OperationStatusFailed,
			wantClass: shared.OSOKAsyncClassFailed,
			wantErr:   true,
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runDataSourceRecordedDeleteWorkRequestCase(t, tt.status, tt.wantClass, tt.wantErr)
		})
	}
}

func runDataSourceRecordedDeleteWorkRequestCase(
	t *testing.T,
	status cloudguardsdk.OperationStatusEnum,
	wantClass shared.OSOKAsyncNormalizedClass,
	wantErr bool,
) {
	t.Helper()

	const workRequestID = "wr-delete-existing"
	resource := newDataSourceRuntimeTestResource()
	trackDataSourceID(resource, testDataSourceID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	fake := &fakeDataSourceOCIClient{}
	fake.getFunc = func(_ context.Context, request cloudguardsdk.GetDataSourceRequest) (cloudguardsdk.GetDataSourceResponse, error) {
		assertDataSourceGetRequest(t, request)
		return cloudguardsdk.GetDataSourceResponse{
			DataSource: dataSourceFromSpec(t, testDataSourceID, resource.Spec, cloudguardsdk.LifecycleStateActive),
		}, nil
	}
	fake.workFunc = func(_ context.Context, request cloudguardsdk.GetWorkRequestRequest) (cloudguardsdk.GetWorkRequestResponse, error) {
		if got := stringValue(request.WorkRequestId); got != workRequestID {
			t.Fatalf("GetWorkRequest() workRequestId = %q, want %q", got, workRequestID)
		}
		return cloudguardsdk.GetWorkRequestResponse{
			WorkRequest: dataSourceWorkRequest(workRequestID, cloudguardsdk.OperationTypeDelete, status, testDataSourceID),
		}, nil
	}
	fake.deleteFunc = func(context.Context, cloudguardsdk.DeleteDataSourceRequest) (cloudguardsdk.DeleteDataSourceResponse, error) {
		t.Fatal("DeleteDataSource() called; want recorded delete work request to be observed first")
		return cloudguardsdk.DeleteDataSourceResponse{}, nil
	}
	client := newDataSourceRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	assertDataSourceRecordedDeleteResult(t, deleted, err, wantErr)
	if len(fake.workRequests) != 1 {
		t.Fatalf("GetWorkRequest() calls = %d, want 1", len(fake.workRequests))
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteDataSource() calls = %d, want 0", len(fake.deleteRequests))
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil ||
		current.Source != shared.OSOKAsyncSourceWorkRequest ||
		current.Phase != shared.OSOKAsyncPhaseDelete ||
		current.WorkRequestID != workRequestID ||
		current.NormalizedClass != wantClass {
		t.Fatalf("status.status.async.current = %#v, want delete work request class %q", current, wantClass)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set, want finalizer retained")
	}
}

func assertDataSourceRecordedDeleteResult(t *testing.T, deleted bool, err error, wantErr bool) {
	t.Helper()

	if wantErr {
		if err == nil {
			t.Fatal("Delete() error = nil, want failed work request error")
		}
		if !strings.Contains(err.Error(), "finished with status FAILED") {
			t.Fatalf("Delete() error = %v, want failed work request status", err)
		}
	} else if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
}

func TestDataSourceServiceClientDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-confirm-pre-error-1"

	resource := newDataSourceRuntimeTestResource()
	trackDataSourceID(resource, testDataSourceID)
	fake := &fakeDataSourceOCIClient{}
	fake.getFunc = func(context.Context, cloudguardsdk.GetDataSourceRequest) (cloudguardsdk.GetDataSourceResponse, error) {
		return cloudguardsdk.GetDataSourceResponse{}, serviceErr
	}
	fake.deleteFunc = func(context.Context, cloudguardsdk.DeleteDataSourceRequest) (cloudguardsdk.DeleteDataSourceResponse, error) {
		t.Fatal("DeleteDataSource() called; want auth-shaped pre-delete read to stop delete")
		return cloudguardsdk.DeleteDataSourceResponse{}, nil
	}
	client := newDataSourceRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped pre-delete read to stay fatal")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read 404", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteDataSource() calls = %d, want 0", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-confirm-pre-error-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-confirm-pre-error-1", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set, want finalizer retained")
	}
}

func newDataSourceRuntimeTestClient(fake *fakeDataSourceOCIClient) DataSourceServiceClient {
	hooks := newDataSourceRuntimeHooksWithOCIClient(fake)
	manager := &DataSourceServiceManager{
		Log: loggerutil.OSOKLogger{Logger: logr.Discard()},
	}
	applyDataSourceRuntimeHooksWithWorkRequestClient(manager, &hooks, fake, nil)
	delegate := defaultDataSourceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*cloudguardv1beta1.DataSource](
			buildDataSourceGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapDataSourceGeneratedClient(hooks, delegate)
}

func newDataSourceRuntimeTestResource() *cloudguardv1beta1.DataSource {
	return &cloudguardv1beta1.DataSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "datasource-sample",
			Namespace: "default",
			UID:       types.UID("datasource-uid"),
		},
		Spec: cloudguardv1beta1.DataSourceSpec{
			CompartmentId:          testDataSourceCompartmentID,
			DisplayName:            testDataSourceDisplayName,
			DataSourceFeedProvider: string(cloudguardsdk.DataSourceFeedProviderLoggingquery),
			Status:                 string(cloudguardsdk.DataSourceStatusEnabled),
			DataSourceDetails: cloudguardv1beta1.DataSourceDetails{
				Regions:           []string{"us-ashburn-1"},
				Query:             "search eventName",
				IntervalInMinutes: 5,
				Threshold:         10,
				QueryStartTime: cloudguardv1beta1.DataSourceDetailsQueryStartTime{
					StartPolicyType: string(cloudguardsdk.ContinuousQueryStartPolicyStartPolicyTypeNoDelayStartPolicy),
				},
				LoggingQueryDetails: cloudguardv1beta1.DataSourceDetailsLoggingQueryDetails{
					LoggingQueryType: string(cloudguardsdk.LoggingQueryTypeInsight),
					KeyEntitiesCount: 2,
				},
				Operator:         string(cloudguardsdk.LoggingQueryOperatorTypeGreater),
				LoggingQueryType: string(cloudguardsdk.LoggingQueryTypeInsight),
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func dataSourceReconcileRequest(resource *cloudguardv1beta1.DataSource) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}}
}

func trackDataSourceID(resource *cloudguardv1beta1.DataSource, id string) {
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
}

func dataSourceFromSpec(
	t *testing.T,
	id string,
	spec cloudguardv1beta1.DataSourceSpec,
	state cloudguardsdk.LifecycleStateEnum,
) cloudguardsdk.DataSource {
	t.Helper()

	provider, err := dataSourceFeedProvider(spec.DataSourceFeedProvider)
	if err != nil {
		t.Fatalf("dataSourceFeedProvider: %v", err)
	}
	status, err := dataSourceStatus(spec.Status)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	details, err := buildDataSourceDetails(spec.DataSourceDetails, provider)
	if err != nil {
		t.Fatalf("dataSourceDetails: %v", err)
	}
	return cloudguardsdk.DataSource{
		Id:                     common.String(id),
		CompartmentId:          common.String(spec.CompartmentId),
		DisplayName:            common.String(spec.DisplayName),
		DataSourceFeedProvider: provider,
		Status:                 status,
		DataSourceDetails:      details,
		LifecycleState:         state,
		FreeformTags:           copyStringMap(spec.FreeformTags),
		DefinedTags:            dataSourceDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func dataSourceSummaryFromSpec(
	t *testing.T,
	id string,
	spec cloudguardv1beta1.DataSourceSpec,
	state cloudguardsdk.LifecycleStateEnum,
) cloudguardsdk.DataSourceSummary {
	t.Helper()

	source := dataSourceFromSpec(t, id, spec, state)
	return cloudguardsdk.DataSourceSummary{
		Id:                     source.Id,
		CompartmentId:          source.CompartmentId,
		DisplayName:            source.DisplayName,
		DataSourceFeedProvider: source.DataSourceFeedProvider,
		Status:                 source.Status,
		LifecycleState:         source.LifecycleState,
		FreeformTags:           source.FreeformTags,
		DefinedTags:            source.DefinedTags,
	}
}

func dataSourceCreateWorkRequestSequence(
	t *testing.T,
	fake *fakeDataSourceOCIClient,
) func(context.Context, cloudguardsdk.GetWorkRequestRequest) (cloudguardsdk.GetWorkRequestResponse, error) {
	t.Helper()

	return func(_ context.Context, request cloudguardsdk.GetWorkRequestRequest) (cloudguardsdk.GetWorkRequestResponse, error) {
		if got := stringValue(request.WorkRequestId); got != "wr-create-1" {
			t.Fatalf("GetWorkRequest() workRequestId = %q, want wr-create-1", got)
		}
		switch len(fake.workRequests) {
		case 1:
			return cloudguardsdk.GetWorkRequestResponse{
				WorkRequest: dataSourceWorkRequest(
					"wr-create-1",
					cloudguardsdk.OperationTypeCreate,
					cloudguardsdk.OperationStatusInProgress,
					testDataSourceID,
				),
			}, nil
		case 2:
			return cloudguardsdk.GetWorkRequestResponse{
				WorkRequest: dataSourceWorkRequest(
					"wr-create-1",
					cloudguardsdk.OperationTypeCreate,
					cloudguardsdk.OperationStatusSucceeded,
					testDataSourceID,
				),
			}, nil
		default:
			t.Fatalf("unexpected GetWorkRequest() call %d", len(fake.workRequests))
			return cloudguardsdk.GetWorkRequestResponse{}, nil
		}
	}
}

func dataSourceWorkRequest(
	id string,
	operationType cloudguardsdk.OperationTypeEnum,
	status cloudguardsdk.OperationStatusEnum,
	dataSourceID string,
) cloudguardsdk.WorkRequest {
	percentComplete := float32(100)
	action := cloudguardsdk.ActionTypeInProgress
	switch operationType {
	case cloudguardsdk.OperationTypeCreate:
		action = cloudguardsdk.ActionTypeCreated
	case cloudguardsdk.OperationTypeUpdate:
		action = cloudguardsdk.ActionTypeUpdated
	case cloudguardsdk.OperationTypeDelete:
		action = cloudguardsdk.ActionTypeDeleted
	}
	if status != cloudguardsdk.OperationStatusSucceeded {
		action = cloudguardsdk.ActionTypeInProgress
		percentComplete = 50
	}
	return cloudguardsdk.WorkRequest{
		Id:              common.String(id),
		CompartmentId:   common.String(testDataSourceCompartmentID),
		OperationType:   operationType,
		Status:          status,
		PercentComplete: common.Float32(percentComplete),
		Resources: []cloudguardsdk.WorkRequestResource{
			{
				EntityType: common.String("DataSource"),
				ActionType: action,
				Identifier: common.String(dataSourceID),
				EntityUri:  common.String("/20200131/dataSources/" + dataSourceID),
			},
		},
	}
}

func dataSourceOperationTypeForWorkRequestID(workRequestID string) cloudguardsdk.OperationTypeEnum {
	switch {
	case strings.Contains(workRequestID, "update"):
		return cloudguardsdk.OperationTypeUpdate
	case strings.Contains(workRequestID, "delete"):
		return cloudguardsdk.OperationTypeDelete
	default:
		return cloudguardsdk.OperationTypeCreate
	}
}

func otherDataSourceSpec(spec cloudguardv1beta1.DataSourceSpec) cloudguardv1beta1.DataSourceSpec {
	spec.DisplayName = "other-datasource"
	return spec
}

func assertDataSourceCreateRequest(
	t *testing.T,
	request cloudguardsdk.CreateDataSourceRequest,
	resource *cloudguardv1beta1.DataSource,
) {
	t.Helper()

	if got := stringValue(request.OpcRetryToken); got != string(resource.UID) {
		t.Fatalf("create opcRetryToken = %q, want resource UID", got)
	}
	if got := stringValue(request.CompartmentId); got != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
	}
	if got := stringValue(request.DisplayName); got != resource.Spec.DisplayName {
		t.Fatalf("create displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := request.DataSourceFeedProvider; got != cloudguardsdk.DataSourceFeedProviderLoggingquery {
		t.Fatalf("create dataSourceFeedProvider = %q, want LOGGINGQUERY", got)
	}
	if got := request.Status; got != cloudguardsdk.DataSourceStatusEnabled {
		t.Fatalf("create status = %q, want ENABLED", got)
	}
	if !reflect.DeepEqual(request.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("create freeformTags = %#v, want %#v", request.FreeformTags, resource.Spec.FreeformTags)
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("create definedTags Operations.CostCenter = %#v, want 42", got)
	}
	assertDataSourceLoggingDetails(t, request.DataSourceDetails, resource.Spec.DataSourceDetails)
}

func assertDataSourceUpdateRequest(
	t *testing.T,
	request cloudguardsdk.UpdateDataSourceRequest,
	resource *cloudguardv1beta1.DataSource,
) {
	t.Helper()

	if got := stringValue(request.DataSourceId); got != testDataSourceID {
		t.Fatalf("update dataSourceId = %q, want %q", got, testDataSourceID)
	}
	if got := stringValue(request.DisplayName); got != resource.Spec.DisplayName {
		t.Fatalf("update displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := request.Status; got != cloudguardsdk.DataSourceStatusEnabled {
		t.Fatalf("update status = %q, want ENABLED", got)
	}
	if !reflect.DeepEqual(request.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("update freeformTags = %#v, want %#v", request.FreeformTags, resource.Spec.FreeformTags)
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("update definedTags Operations.CostCenter = %#v, want 42", got)
	}
	assertDataSourceLoggingDetails(t, request.DataSourceDetails, resource.Spec.DataSourceDetails)
}

//nolint:gocyclo // The assertion mirrors the polymorphic logging query request shape.
func assertDataSourceLoggingDetails(
	t *testing.T,
	details cloudguardsdk.DataSourceDetails,
	spec cloudguardv1beta1.DataSourceDetails,
) {
	t.Helper()

	logging, ok := details.(cloudguardsdk.LoggingQueryDataSourceDetails)
	if !ok {
		t.Fatalf("dataSourceDetails = %T, want LoggingQueryDataSourceDetails", details)
	}
	if !reflect.DeepEqual(logging.Regions, spec.Regions) {
		t.Fatalf("dataSourceDetails.regions = %#v, want %#v", logging.Regions, spec.Regions)
	}
	if got := stringValue(logging.Query); got != spec.Query {
		t.Fatalf("dataSourceDetails.query = %q, want %q", got, spec.Query)
	}
	if logging.IntervalInMinutes == nil || *logging.IntervalInMinutes != spec.IntervalInMinutes {
		t.Fatalf("dataSourceDetails.intervalInMinutes = %#v, want %d", logging.IntervalInMinutes, spec.IntervalInMinutes)
	}
	if logging.Threshold == nil || *logging.Threshold != spec.Threshold {
		t.Fatalf("dataSourceDetails.threshold = %#v, want %d", logging.Threshold, spec.Threshold)
	}
	if logging.QueryStartTime == nil {
		t.Fatal("dataSourceDetails.queryStartTime = nil, want no-delay policy")
	}
	if got := logging.Operator; got != cloudguardsdk.LoggingQueryOperatorTypeGreater {
		t.Fatalf("dataSourceDetails.operator = %q, want GREATER", got)
	}
	if got := logging.LoggingQueryType; got != cloudguardsdk.LoggingQueryTypeInsight {
		t.Fatalf("dataSourceDetails.loggingQueryType = %q, want INSIGHT", got)
	}
}

func assertDataSourceBindListRequest(t *testing.T, request cloudguardsdk.ListDataSourcesRequest) {
	t.Helper()

	if got := stringValue(request.CompartmentId); got != testDataSourceCompartmentID {
		t.Fatalf("list compartmentId = %q, want %q", got, testDataSourceCompartmentID)
	}
	if got := stringValue(request.DisplayName); got != testDataSourceDisplayName {
		t.Fatalf("list displayName = %q, want %q", got, testDataSourceDisplayName)
	}
	if got := request.DataSourceFeedProvider; got != cloudguardsdk.ListDataSourcesDataSourceFeedProviderLoggingquery {
		t.Fatalf("list dataSourceFeedProvider = %q, want LOGGINGQUERY", got)
	}
	if request.LifecycleState == "" {
		t.Fatal("list lifecycleState is empty; Cloud Guard defaults empty lifecycleState to ACTIVE")
	}
	if !dataSourceExpectedListState(request.LifecycleState) {
		t.Fatalf("list lifecycleState = %q, want one of %#v", request.LifecycleState, dataSourceLookupLifecycleStates())
	}
}

func assertDataSourceListStates(t *testing.T, requests []cloudguardsdk.ListDataSourcesRequest) {
	t.Helper()

	counts := map[cloudguardsdk.ListDataSourcesLifecycleStateEnum]int{}
	for _, request := range requests {
		counts[request.LifecycleState]++
	}
	for _, state := range dataSourceLookupLifecycleStates() {
		if counts[state] == 0 {
			t.Fatalf("ListDataSources() lifecycle states = %#v, missing %q", counts, state)
		}
	}
}

func dataSourceExpectedListState(state cloudguardsdk.ListDataSourcesLifecycleStateEnum) bool {
	for _, expected := range dataSourceLookupLifecycleStates() {
		if state == expected {
			return true
		}
	}
	return false
}

func assertDataSourceGetRequest(t *testing.T, request cloudguardsdk.GetDataSourceRequest) {
	t.Helper()

	if got := stringValue(request.DataSourceId); got != testDataSourceID {
		t.Fatalf("get dataSourceId = %q, want %q", got, testDataSourceID)
	}
}

func assertDataSourceCreateOrUpdateSucceeded(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()

	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after ACTIVE follow-up")
	}
}

func assertDataSourceTrackedID(t *testing.T, resource *cloudguardv1beta1.DataSource, id string) {
	t.Helper()

	if got := resource.Status.Id; got != id {
		t.Fatalf("status.id = %q, want %q", got, id)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != id {
		t.Fatalf("status.status.ocid = %q, want %q", got, id)
	}
}

func assertDataSourceContainsAll(t *testing.T, name string, got []string, wants ...string) {
	t.Helper()

	for _, want := range wants {
		found := false
		for _, value := range got {
			if value == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s = %#v, missing %q", name, got, want)
		}
	}
}

func copyStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}
