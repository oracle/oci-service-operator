/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package monitoredregion

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	resourceanalyticssdk "github.com/oracle/oci-go-sdk/v65/resourceanalytics"
	resourceanalyticsv1beta1 "github.com/oracle/oci-service-operator/api/resourceanalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeMonitoredRegionOCIClient struct {
	createMonitoredRegionFn func(context.Context, resourceanalyticssdk.CreateMonitoredRegionRequest) (resourceanalyticssdk.CreateMonitoredRegionResponse, error)
	getMonitoredRegionFn    func(context.Context, resourceanalyticssdk.GetMonitoredRegionRequest) (resourceanalyticssdk.GetMonitoredRegionResponse, error)
	listMonitoredRegionsFn  func(context.Context, resourceanalyticssdk.ListMonitoredRegionsRequest) (resourceanalyticssdk.ListMonitoredRegionsResponse, error)
	deleteMonitoredRegionFn func(context.Context, resourceanalyticssdk.DeleteMonitoredRegionRequest) (resourceanalyticssdk.DeleteMonitoredRegionResponse, error)
}

func (f *fakeMonitoredRegionOCIClient) CreateMonitoredRegion(
	ctx context.Context,
	req resourceanalyticssdk.CreateMonitoredRegionRequest,
) (resourceanalyticssdk.CreateMonitoredRegionResponse, error) {
	if f.createMonitoredRegionFn != nil {
		return f.createMonitoredRegionFn(ctx, req)
	}
	return resourceanalyticssdk.CreateMonitoredRegionResponse{}, nil
}

func (f *fakeMonitoredRegionOCIClient) GetMonitoredRegion(
	ctx context.Context,
	req resourceanalyticssdk.GetMonitoredRegionRequest,
) (resourceanalyticssdk.GetMonitoredRegionResponse, error) {
	if f.getMonitoredRegionFn != nil {
		return f.getMonitoredRegionFn(ctx, req)
	}
	return resourceanalyticssdk.GetMonitoredRegionResponse{}, nil
}

func (f *fakeMonitoredRegionOCIClient) ListMonitoredRegions(
	ctx context.Context,
	req resourceanalyticssdk.ListMonitoredRegionsRequest,
) (resourceanalyticssdk.ListMonitoredRegionsResponse, error) {
	if f.listMonitoredRegionsFn != nil {
		return f.listMonitoredRegionsFn(ctx, req)
	}
	return resourceanalyticssdk.ListMonitoredRegionsResponse{}, nil
}

func (f *fakeMonitoredRegionOCIClient) DeleteMonitoredRegion(
	ctx context.Context,
	req resourceanalyticssdk.DeleteMonitoredRegionRequest,
) (resourceanalyticssdk.DeleteMonitoredRegionResponse, error) {
	if f.deleteMonitoredRegionFn != nil {
		return f.deleteMonitoredRegionFn(ctx, req)
	}
	return resourceanalyticssdk.DeleteMonitoredRegionResponse{}, nil
}

func testMonitoredRegionClient(fake *fakeMonitoredRegionOCIClient) MonitoredRegionServiceClient {
	return newMonitoredRegionServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		fake,
	)
}

func newMonitoredRegionTestResource() *resourceanalyticsv1beta1.MonitoredRegion {
	return &resourceanalyticsv1beta1.MonitoredRegion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "monitored-region",
			Namespace: "default",
		},
		Spec: resourceanalyticsv1beta1.MonitoredRegionSpec{
			ResourceAnalyticsInstanceId: "ocid1.resourceanalyticsinstance.oc1..instance",
			RegionId:                    "us-phoenix-1",
		},
	}
}

func newExistingMonitoredRegionTestResource(id string) *resourceanalyticsv1beta1.MonitoredRegion {
	resource := newMonitoredRegionTestResource()
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	resource.Status.ResourceAnalyticsInstanceId = resource.Spec.ResourceAnalyticsInstanceId
	resource.Status.RegionId = resource.Spec.RegionId
	resource.Status.LifecycleState = string(resourceanalyticssdk.MonitoredRegionLifecycleStateActive)
	return resource
}

func newSDKMonitoredRegion(
	id string,
	resourceAnalyticsInstanceID string,
	regionID string,
	state resourceanalyticssdk.MonitoredRegionLifecycleStateEnum,
) resourceanalyticssdk.MonitoredRegion {
	return resourceanalyticssdk.MonitoredRegion{
		Id:                          common.String(id),
		ResourceAnalyticsInstanceId: common.String(resourceAnalyticsInstanceID),
		RegionId:                    common.String(regionID),
		LifecycleState:              state,
	}
}

func newSDKMonitoredRegionSummary(
	id string,
	resourceAnalyticsInstanceID string,
	regionID string,
	state resourceanalyticssdk.MonitoredRegionLifecycleStateEnum,
) resourceanalyticssdk.MonitoredRegionSummary {
	return resourceanalyticssdk.MonitoredRegionSummary{
		Id:                          common.String(id),
		ResourceAnalyticsInstanceId: common.String(resourceAnalyticsInstanceID),
		RegionId:                    common.String(regionID),
		LifecycleState:              state,
	}
}

type monitoredRegionCreateRefreshRecorder struct {
	createRequest resourceanalyticssdk.CreateMonitoredRegionRequest
	listCalls     int
	getCalls      int
}

type monitoredRegionOCICallCounts struct {
	create int
	get    int
	list   int
	delete int
}

func TestMonitoredRegionCreateOrUpdateCreatesAndRefreshesStatus(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.monitoredregion.oc1..created"

	resource := newMonitoredRegionTestResource()
	recorder := &monitoredRegionCreateRefreshRecorder{}
	client := newMonitoredRegionCreateRefreshClient(t, createdID, recorder)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireMonitoredRegionCreateRefreshResult(t, response, err, resource, recorder, createdID)
}

func newMonitoredRegionCreateRefreshClient(
	t *testing.T,
	createdID string,
	recorder *monitoredRegionCreateRefreshRecorder,
) MonitoredRegionServiceClient {
	t.Helper()

	return testMonitoredRegionClient(&fakeMonitoredRegionOCIClient{
		listMonitoredRegionsFn: func(_ context.Context, req resourceanalyticssdk.ListMonitoredRegionsRequest) (resourceanalyticssdk.ListMonitoredRegionsResponse, error) {
			recorder.listCalls++
			requireStringPtr(t, req.ResourceAnalyticsInstanceId, "ocid1.resourceanalyticsinstance.oc1..instance", "list resourceAnalyticsInstanceId")
			if req.Id != nil {
				t.Fatalf("list id = %v, want nil before create", req.Id)
			}
			return resourceanalyticssdk.ListMonitoredRegionsResponse{}, nil
		},
		createMonitoredRegionFn: func(_ context.Context, req resourceanalyticssdk.CreateMonitoredRegionRequest) (resourceanalyticssdk.CreateMonitoredRegionResponse, error) {
			recorder.createRequest = req
			return resourceanalyticssdk.CreateMonitoredRegionResponse{
				MonitoredRegion: newSDKMonitoredRegion(
					createdID,
					"ocid1.resourceanalyticsinstance.oc1..instance",
					"us-phoenix-1",
					resourceanalyticssdk.MonitoredRegionLifecycleStateCreating,
				),
				OpcRequestId:     common.String("opc-create-1"),
				OpcWorkRequestId: common.String("wr-create-1"),
			}, nil
		},
		getMonitoredRegionFn: func(_ context.Context, req resourceanalyticssdk.GetMonitoredRegionRequest) (resourceanalyticssdk.GetMonitoredRegionResponse, error) {
			recorder.getCalls++
			requireStringPtr(t, req.MonitoredRegionId, createdID, "get monitoredRegionId")
			return resourceanalyticssdk.GetMonitoredRegionResponse{
				MonitoredRegion: newSDKMonitoredRegion(
					createdID,
					"ocid1.resourceanalyticsinstance.oc1..instance",
					"us-phoenix-1",
					resourceanalyticssdk.MonitoredRegionLifecycleStateActive,
				),
			}, nil
		},
	})
}

func requireMonitoredRegionCreateRefreshResult(
	t *testing.T,
	response servicemanager.OSOKResponse,
	err error,
	resource *resourceanalyticsv1beta1.MonitoredRegion,
	recorder *monitoredRegionCreateRefreshRecorder,
	createdID string,
) {
	t.Helper()

	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after ACTIVE readback")
	}
	if recorder.listCalls != 1 {
		t.Fatalf("ListMonitoredRegions() calls = %d, want 1", recorder.listCalls)
	}
	if recorder.getCalls != 1 {
		t.Fatalf("GetMonitoredRegion() calls = %d, want 1", recorder.getCalls)
	}
	requireStringPtr(t, recorder.createRequest.ResourceAnalyticsInstanceId, resource.Spec.ResourceAnalyticsInstanceId, "create resourceAnalyticsInstanceId")
	requireStringPtr(t, recorder.createRequest.RegionId, resource.Spec.RegionId, "create regionId")
	if recorder.createRequest.OpcRetryToken == nil || strings.TrimSpace(*recorder.createRequest.OpcRetryToken) == "" {
		t.Fatal("create opcRetryToken is empty, want deterministic retry token")
	}
	requireMonitoredRegionStatus(t, resource, createdID, "ACTIVE")
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-1", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after ACTIVE readback", resource.Status.OsokStatus.Async.Current)
	}
}

func TestMonitoredRegionCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.monitoredregion.oc1..existing"

	createCalls := 0
	getCalls := 0
	var pages []string

	client := testMonitoredRegionClient(&fakeMonitoredRegionOCIClient{
		listMonitoredRegionsFn: func(_ context.Context, req resourceanalyticssdk.ListMonitoredRegionsRequest) (resourceanalyticssdk.ListMonitoredRegionsResponse, error) {
			pages = append(pages, stringValue(req.Page))
			requireStringPtr(t, req.ResourceAnalyticsInstanceId, "ocid1.resourceanalyticsinstance.oc1..instance", "list resourceAnalyticsInstanceId")
			switch stringValue(req.Page) {
			case "":
				return resourceanalyticssdk.ListMonitoredRegionsResponse{
					MonitoredRegionCollection: resourceanalyticssdk.MonitoredRegionCollection{
						Items: []resourceanalyticssdk.MonitoredRegionSummary{
							newSDKMonitoredRegionSummary(
								"ocid1.monitoredregion.oc1..other",
								"ocid1.resourceanalyticsinstance.oc1..instance",
								"us-ashburn-1",
								resourceanalyticssdk.MonitoredRegionLifecycleStateActive,
							),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case "page-2":
				return resourceanalyticssdk.ListMonitoredRegionsResponse{
					MonitoredRegionCollection: resourceanalyticssdk.MonitoredRegionCollection{
						Items: []resourceanalyticssdk.MonitoredRegionSummary{
							newSDKMonitoredRegionSummary(
								existingID,
								"ocid1.resourceanalyticsinstance.oc1..instance",
								"us-phoenix-1",
								resourceanalyticssdk.MonitoredRegionLifecycleStateActive,
							),
						},
					},
				}, nil
			default:
				t.Fatalf("unexpected list page = %q", stringValue(req.Page))
				return resourceanalyticssdk.ListMonitoredRegionsResponse{}, nil
			}
		},
		createMonitoredRegionFn: func(context.Context, resourceanalyticssdk.CreateMonitoredRegionRequest) (resourceanalyticssdk.CreateMonitoredRegionResponse, error) {
			createCalls++
			return resourceanalyticssdk.CreateMonitoredRegionResponse{}, nil
		},
		getMonitoredRegionFn: func(_ context.Context, req resourceanalyticssdk.GetMonitoredRegionRequest) (resourceanalyticssdk.GetMonitoredRegionResponse, error) {
			getCalls++
			requireStringPtr(t, req.MonitoredRegionId, existingID, "get monitoredRegionId")
			return resourceanalyticssdk.GetMonitoredRegionResponse{
				MonitoredRegion: newSDKMonitoredRegion(
					existingID,
					"ocid1.resourceanalyticsinstance.oc1..instance",
					"us-phoenix-1",
					resourceanalyticssdk.MonitoredRegionLifecycleStateActive,
				),
			}, nil
		},
	})

	resource := newMonitoredRegionTestResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if createCalls != 0 {
		t.Fatalf("CreateMonitoredRegion() calls = %d, want 0", createCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetMonitoredRegion() calls = %d, want 1 after bind", getCalls)
	}
	if got, want := strings.Join(pages, ","), ",page-2"; got != want {
		t.Fatalf("list pages = %q, want %q", got, want)
	}
	requireMonitoredRegionStatus(t, resource, existingID, "ACTIVE")
}

func TestMonitoredRegionCreateOrUpdateRejectsMissingListIdentityBeforeOCI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(*resourceanalyticsv1beta1.MonitoredRegion)
		wantField string
	}{
		{
			name: "empty resourceAnalyticsInstanceId",
			mutate: func(resource *resourceanalyticsv1beta1.MonitoredRegion) {
				resource.Spec.ResourceAnalyticsInstanceId = "   "
			},
			wantField: "resourceAnalyticsInstanceId",
		},
		{
			name: "empty regionId",
			mutate: func(resource *resourceanalyticsv1beta1.MonitoredRegion) {
				resource.Spec.RegionId = "   "
			},
			wantField: "regionId",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			counts := &monitoredRegionOCICallCounts{}
			client := testMonitoredRegionClient(newCountingMonitoredRegionOCIClient(counts))

			resource := newMonitoredRegionTestResource()
			tc.mutate(resource)
			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err == nil {
				t.Fatal("CreateOrUpdate() error = nil, want missing identity rejection")
			}
			if response.IsSuccessful {
				t.Fatal("CreateOrUpdate() should report failure for missing identity")
			}
			requireNoMonitoredRegionOCICalls(t, counts)
			if !strings.Contains(err.Error(), tc.wantField) {
				t.Fatalf("CreateOrUpdate() error = %v, want %s context", err, tc.wantField)
			}
			requireMonitoredRegionFailedStatus(t, resource, tc.wantField)
		})
	}
}

func TestMonitoredRegionCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.monitoredregion.oc1..existing"

	createCalls := 0
	listCalls := 0
	client := testMonitoredRegionClient(&fakeMonitoredRegionOCIClient{
		getMonitoredRegionFn: func(_ context.Context, req resourceanalyticssdk.GetMonitoredRegionRequest) (resourceanalyticssdk.GetMonitoredRegionResponse, error) {
			requireStringPtr(t, req.MonitoredRegionId, existingID, "get monitoredRegionId")
			return resourceanalyticssdk.GetMonitoredRegionResponse{
				MonitoredRegion: newSDKMonitoredRegion(
					existingID,
					"ocid1.resourceanalyticsinstance.oc1..instance",
					"us-phoenix-1",
					resourceanalyticssdk.MonitoredRegionLifecycleStateActive,
				),
			}, nil
		},
		listMonitoredRegionsFn: func(context.Context, resourceanalyticssdk.ListMonitoredRegionsRequest) (resourceanalyticssdk.ListMonitoredRegionsResponse, error) {
			listCalls++
			return resourceanalyticssdk.ListMonitoredRegionsResponse{}, nil
		},
		createMonitoredRegionFn: func(context.Context, resourceanalyticssdk.CreateMonitoredRegionRequest) (resourceanalyticssdk.CreateMonitoredRegionResponse, error) {
			createCalls++
			return resourceanalyticssdk.CreateMonitoredRegionResponse{}, nil
		},
	})

	resource := newExistingMonitoredRegionTestResource(existingID)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if createCalls != 0 {
		t.Fatalf("CreateMonitoredRegion() calls = %d, want 0", createCalls)
	}
	if listCalls != 0 {
		t.Fatalf("ListMonitoredRegions() calls = %d, want 0 when GET succeeds", listCalls)
	}
	requireMonitoredRegionStatus(t, resource, existingID, "ACTIVE")
}

func TestMonitoredRegionRejectsCreateOnlyDriftWithoutUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.monitoredregion.oc1..existing"

	createCalls := 0
	client := testMonitoredRegionClient(&fakeMonitoredRegionOCIClient{
		getMonitoredRegionFn: func(_ context.Context, req resourceanalyticssdk.GetMonitoredRegionRequest) (resourceanalyticssdk.GetMonitoredRegionResponse, error) {
			requireStringPtr(t, req.MonitoredRegionId, existingID, "get monitoredRegionId")
			return resourceanalyticssdk.GetMonitoredRegionResponse{
				MonitoredRegion: newSDKMonitoredRegion(
					existingID,
					"ocid1.resourceanalyticsinstance.oc1..instance",
					"us-phoenix-1",
					resourceanalyticssdk.MonitoredRegionLifecycleStateActive,
				),
			}, nil
		},
		createMonitoredRegionFn: func(context.Context, resourceanalyticssdk.CreateMonitoredRegionRequest) (resourceanalyticssdk.CreateMonitoredRegionResponse, error) {
			createCalls++
			return resourceanalyticssdk.CreateMonitoredRegionResponse{}, nil
		},
	})

	resource := newExistingMonitoredRegionTestResource(existingID)
	resource.Spec.RegionId = "us-ashburn-1"
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure for create-only drift")
	}
	if createCalls != 0 {
		t.Fatalf("CreateMonitoredRegion() calls = %d, want 0", createCalls)
	}
	if !strings.Contains(err.Error(), "regionId") {
		t.Fatalf("CreateOrUpdate() error = %v, want regionId drift context", err)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", got)
	}
}

func TestMonitoredRegionDeleteRetainsFinalizerWhileLifecycleDeleting(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.monitoredregion.oc1..existing"

	getCalls := 0
	var deleteRequest resourceanalyticssdk.DeleteMonitoredRegionRequest
	client := testMonitoredRegionClient(&fakeMonitoredRegionOCIClient{
		getMonitoredRegionFn: func(_ context.Context, req resourceanalyticssdk.GetMonitoredRegionRequest) (resourceanalyticssdk.GetMonitoredRegionResponse, error) {
			getCalls++
			requireStringPtr(t, req.MonitoredRegionId, existingID, "get monitoredRegionId")
			state := resourceanalyticssdk.MonitoredRegionLifecycleStateActive
			if getCalls == 3 {
				state = resourceanalyticssdk.MonitoredRegionLifecycleStateDeleting
			}
			return resourceanalyticssdk.GetMonitoredRegionResponse{
				MonitoredRegion: newSDKMonitoredRegion(
					existingID,
					"ocid1.resourceanalyticsinstance.oc1..instance",
					"us-phoenix-1",
					state,
				),
			}, nil
		},
		deleteMonitoredRegionFn: func(_ context.Context, req resourceanalyticssdk.DeleteMonitoredRegionRequest) (resourceanalyticssdk.DeleteMonitoredRegionResponse, error) {
			deleteRequest = req
			return resourceanalyticssdk.DeleteMonitoredRegionResponse{
				OpcRequestId:     common.String("opc-delete-1"),
				OpcWorkRequestId: common.String("wr-delete-1"),
			}, nil
		},
	})

	resource := newExistingMonitoredRegionTestResource(existingID)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while OCI lifecycle is DELETING")
	}
	if getCalls != 3 {
		t.Fatalf("GetMonitoredRegion() calls = %d, want 3", getCalls)
	}
	requireStringPtr(t, deleteRequest.MonitoredRegionId, existingID, "delete monitoredRegionId")
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.LifecycleState; got != "DELETING" {
		t.Fatalf("status.lifecycleState = %q, want DELETING", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-1", got)
	}
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want delete phase", current)
	}
}

func TestMonitoredRegionDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.monitoredregion.oc1..existing"

	deleteCalls := 0
	client := testMonitoredRegionClient(&fakeMonitoredRegionOCIClient{
		getMonitoredRegionFn: func(context.Context, resourceanalyticssdk.GetMonitoredRegionRequest) (resourceanalyticssdk.GetMonitoredRegionResponse, error) {
			return resourceanalyticssdk.GetMonitoredRegionResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"not authorized or not found",
			)
		},
		deleteMonitoredRegionFn: func(context.Context, resourceanalyticssdk.DeleteMonitoredRegionRequest) (resourceanalyticssdk.DeleteMonitoredRegionResponse, error) {
			deleteCalls++
			return resourceanalyticssdk.DeleteMonitoredRegionResponse{}, nil
		},
	})

	resource := newExistingMonitoredRegionTestResource(existingID)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous 404 rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteMonitoredRegion() calls = %d, want 0", deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 context", err)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for ambiguous 404", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestMonitoredRegionDeleteWithoutTrackedIDRejectsAuthShapedConfirmReadAfterListMatch(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.monitoredregion.oc1..existing"

	serviceErr := errortest.NewServiceError(
		404,
		errorutil.NotAuthorizedOrNotFound,
		"not authorized or not found",
	)
	serviceErr.OpcRequestID = "opc-confirm-after-list-1"

	listCalls := 0
	getCalls := 0
	deleteCalls := 0
	client := testMonitoredRegionClient(&fakeMonitoredRegionOCIClient{
		listMonitoredRegionsFn: func(_ context.Context, req resourceanalyticssdk.ListMonitoredRegionsRequest) (resourceanalyticssdk.ListMonitoredRegionsResponse, error) {
			listCalls++
			requireStringPtr(t, req.ResourceAnalyticsInstanceId, "ocid1.resourceanalyticsinstance.oc1..instance", "list resourceAnalyticsInstanceId")
			return resourceanalyticssdk.ListMonitoredRegionsResponse{
				MonitoredRegionCollection: resourceanalyticssdk.MonitoredRegionCollection{
					Items: []resourceanalyticssdk.MonitoredRegionSummary{
						newSDKMonitoredRegionSummary(
							existingID,
							"ocid1.resourceanalyticsinstance.oc1..instance",
							"us-phoenix-1",
							resourceanalyticssdk.MonitoredRegionLifecycleStateActive,
						),
					},
				},
			}, nil
		},
		getMonitoredRegionFn: func(_ context.Context, req resourceanalyticssdk.GetMonitoredRegionRequest) (resourceanalyticssdk.GetMonitoredRegionResponse, error) {
			getCalls++
			requireStringPtr(t, req.MonitoredRegionId, existingID, "get monitoredRegionId")
			return resourceanalyticssdk.GetMonitoredRegionResponse{}, serviceErr
		},
		deleteMonitoredRegionFn: func(context.Context, resourceanalyticssdk.DeleteMonitoredRegionRequest) (resourceanalyticssdk.DeleteMonitoredRegionResponse, error) {
			deleteCalls++
			return resourceanalyticssdk.DeleteMonitoredRegionResponse{}, nil
		},
	})

	resource := newMonitoredRegionTestResource()
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous 404 rejection after list match")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if listCalls != 1 {
		t.Fatalf("ListMonitoredRegions() calls = %d, want 1", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetMonitoredRegion() calls = %d, want 1", getCalls)
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteMonitoredRegion() calls = %d, want 0", deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 context", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-confirm-after-list-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-confirm-after-list-1", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for ambiguous 404", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestMonitoredRegionDeleteReleasesUntrackedFinalizerWhenListHasNoMatch(t *testing.T) {
	t.Parallel()

	listCalls := 0
	deleteCalls := 0
	client := testMonitoredRegionClient(&fakeMonitoredRegionOCIClient{
		listMonitoredRegionsFn: func(_ context.Context, req resourceanalyticssdk.ListMonitoredRegionsRequest) (resourceanalyticssdk.ListMonitoredRegionsResponse, error) {
			listCalls++
			requireStringPtr(t, req.ResourceAnalyticsInstanceId, "ocid1.resourceanalyticsinstance.oc1..instance", "list resourceAnalyticsInstanceId")
			if req.Id != nil {
				t.Fatalf("list id = %v, want nil without tracked OCID", req.Id)
			}
			return resourceanalyticssdk.ListMonitoredRegionsResponse{}, nil
		},
		deleteMonitoredRegionFn: func(context.Context, resourceanalyticssdk.DeleteMonitoredRegionRequest) (resourceanalyticssdk.DeleteMonitoredRegionResponse, error) {
			deleteCalls++
			return resourceanalyticssdk.DeleteMonitoredRegionResponse{}, nil
		},
	})

	resource := newMonitoredRegionTestResource()
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after zero-match list confirmation")
	}
	if listCalls != 1 {
		t.Fatalf("ListMonitoredRegions() calls = %d, want 1", listCalls)
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteMonitoredRegion() calls = %d, want 0 without a resolved OCID", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want deletion timestamp after zero-match list confirmation")
	}
}

func TestMonitoredRegionDeleteRejectsMissingListIdentityBeforeOCI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(*resourceanalyticsv1beta1.MonitoredRegion)
		wantField string
	}{
		{
			name: "empty resourceAnalyticsInstanceId",
			mutate: func(resource *resourceanalyticsv1beta1.MonitoredRegion) {
				resource.Spec.ResourceAnalyticsInstanceId = "   "
			},
			wantField: "resourceAnalyticsInstanceId",
		},
		{
			name: "empty regionId",
			mutate: func(resource *resourceanalyticsv1beta1.MonitoredRegion) {
				resource.Spec.RegionId = "   "
			},
			wantField: "regionId",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			counts := &monitoredRegionOCICallCounts{}
			client := testMonitoredRegionClient(newCountingMonitoredRegionOCIClient(counts))

			resource := newMonitoredRegionTestResource()
			tc.mutate(resource)
			deleted, err := client.Delete(context.Background(), resource)
			if err == nil {
				t.Fatal("Delete() error = nil, want missing identity rejection")
			}
			if deleted {
				t.Fatal("Delete() deleted = true, want finalizer retained for missing identity")
			}
			requireNoMonitoredRegionOCICalls(t, counts)
			if !strings.Contains(err.Error(), tc.wantField) {
				t.Fatalf("Delete() error = %v, want %s context", err, tc.wantField)
			}
			requireMonitoredRegionFailedStatus(t, resource, tc.wantField)
		})
	}
}

func TestMonitoredRegionCreateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	client := testMonitoredRegionClient(&fakeMonitoredRegionOCIClient{
		createMonitoredRegionFn: func(context.Context, resourceanalyticssdk.CreateMonitoredRegionRequest) (resourceanalyticssdk.CreateMonitoredRegionResponse, error) {
			return resourceanalyticssdk.CreateMonitoredRegionResponse{}, errortest.NewServiceError(
				500,
				"InternalError",
				"create failed",
			)
		},
	})

	resource := newMonitoredRegionTestResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", got)
	}
}

func TestApplyMonitoredRegionRuntimeHooksInstallsReviewedSemantics(t *testing.T) {
	t.Parallel()

	hooks := newMonitoredRegionDefaultRuntimeHooks(resourceanalyticssdk.MonitoredRegionClient{})
	applyMonitoredRegionRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if got := hooks.Semantics.FinalizerPolicy; got != "retain-until-confirmed-delete" {
		t.Fatalf("hooks.Semantics.FinalizerPolicy = %q, want retain-until-confirmed-delete", got)
	}
	if len(hooks.Semantics.Mutation.ForceNew) != 2 {
		t.Fatalf("hooks.Semantics.Mutation.ForceNew = %v, want resourceAnalyticsInstanceId and regionId", hooks.Semantics.Mutation.ForceNew)
	}
}

func TestApplyMonitoredRegionRuntimeHooksInstallsReviewedHooks(t *testing.T) {
	t.Parallel()

	hooks := newMonitoredRegionDefaultRuntimeHooks(resourceanalyticssdk.MonitoredRegionClient{})
	applyMonitoredRegionRuntimeHooks(&hooks)

	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want reviewed create builder")
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("hooks.Identity.GuardExistingBeforeCreate = nil, want pre-create identity validation")
	}
	if hooks.BuildUpdateBody != nil {
		t.Fatal("hooks.BuildUpdateBody is configured, want nil because MonitoredRegion has no UPDATE operation")
	}
	if hooks.ParityHooks.ValidateCreateOnlyDrift == nil {
		t.Fatal("hooks.ParityHooks.ValidateCreateOnlyDrift = nil, want create-only drift guard")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("hooks.DeleteHooks.HandleError = nil, want conservative delete error handler")
	}
}

func TestMonitoredRegionBuildCreateBodyMapsSpec(t *testing.T) {
	t.Parallel()

	hooks := newMonitoredRegionDefaultRuntimeHooks(resourceanalyticssdk.MonitoredRegionClient{})
	applyMonitoredRegionRuntimeHooks(&hooks)

	resource := newMonitoredRegionTestResource()
	body, err := hooks.BuildCreateBody(context.Background(), resource, resource.Namespace)
	if err != nil {
		t.Fatalf("hooks.BuildCreateBody() error = %v", err)
	}
	details, ok := body.(resourceanalyticssdk.CreateMonitoredRegionDetails)
	if !ok {
		t.Fatalf("hooks.BuildCreateBody() body type = %T, want resourceanalytics.CreateMonitoredRegionDetails", body)
	}
	requireStringPtr(t, details.ResourceAnalyticsInstanceId, resource.Spec.ResourceAnalyticsInstanceId, "create body resourceAnalyticsInstanceId")
	requireStringPtr(t, details.RegionId, resource.Spec.RegionId, "create body regionId")
}

func newCountingMonitoredRegionOCIClient(counts *monitoredRegionOCICallCounts) *fakeMonitoredRegionOCIClient {
	return &fakeMonitoredRegionOCIClient{
		createMonitoredRegionFn: func(context.Context, resourceanalyticssdk.CreateMonitoredRegionRequest) (resourceanalyticssdk.CreateMonitoredRegionResponse, error) {
			counts.create++
			return resourceanalyticssdk.CreateMonitoredRegionResponse{}, nil
		},
		getMonitoredRegionFn: func(context.Context, resourceanalyticssdk.GetMonitoredRegionRequest) (resourceanalyticssdk.GetMonitoredRegionResponse, error) {
			counts.get++
			return resourceanalyticssdk.GetMonitoredRegionResponse{}, nil
		},
		listMonitoredRegionsFn: func(context.Context, resourceanalyticssdk.ListMonitoredRegionsRequest) (resourceanalyticssdk.ListMonitoredRegionsResponse, error) {
			counts.list++
			return resourceanalyticssdk.ListMonitoredRegionsResponse{}, nil
		},
		deleteMonitoredRegionFn: func(context.Context, resourceanalyticssdk.DeleteMonitoredRegionRequest) (resourceanalyticssdk.DeleteMonitoredRegionResponse, error) {
			counts.delete++
			return resourceanalyticssdk.DeleteMonitoredRegionResponse{}, nil
		},
	}
}

func requireMonitoredRegionStatus(
	t *testing.T,
	resource *resourceanalyticsv1beta1.MonitoredRegion,
	wantID string,
	wantLifecycleState string,
) {
	t.Helper()

	if got := resource.Status.Id; got != wantID {
		t.Fatalf("status.id = %q, want %q", got, wantID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != wantID {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantID)
	}
	if got := resource.Status.ResourceAnalyticsInstanceId; got != resource.Spec.ResourceAnalyticsInstanceId {
		t.Fatalf("status.resourceAnalyticsInstanceId = %q, want %q", got, resource.Spec.ResourceAnalyticsInstanceId)
	}
	if got := resource.Status.RegionId; got != resource.Spec.RegionId {
		t.Fatalf("status.regionId = %q, want %q", got, resource.Spec.RegionId)
	}
	if got := resource.Status.LifecycleState; got != wantLifecycleState {
		t.Fatalf("status.lifecycleState = %q, want %q", got, wantLifecycleState)
	}
}

func requireNoMonitoredRegionOCICalls(t *testing.T, counts *monitoredRegionOCICallCounts) {
	t.Helper()

	if counts.create != 0 {
		t.Fatalf("CreateMonitoredRegion() calls = %d, want 0", counts.create)
	}
	if counts.get != 0 {
		t.Fatalf("GetMonitoredRegion() calls = %d, want 0", counts.get)
	}
	if counts.list != 0 {
		t.Fatalf("ListMonitoredRegions() calls = %d, want 0", counts.list)
	}
	if counts.delete != 0 {
		t.Fatalf("DeleteMonitoredRegion() calls = %d, want 0", counts.delete)
	}
}

func requireMonitoredRegionFailedStatus(
	t *testing.T,
	resource *resourceanalyticsv1beta1.MonitoredRegion,
	wantField string,
) {
	t.Helper()

	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", got)
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, wantField) {
		t.Fatalf("status.message = %q, want %s context", resource.Status.OsokStatus.Message, wantField)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 {
		t.Fatal("status.conditions is empty, want Failed condition")
	}
	last := resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1]
	if last.Type != shared.Failed {
		t.Fatalf("last condition type = %q, want Failed", last.Type)
	}
}

func requireStringPtr(t *testing.T, got *string, want string, name string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
