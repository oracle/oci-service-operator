/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managementsavedsearch

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	managementdashboardsdk "github.com/oracle/oci-go-sdk/v65/managementdashboard"
	managementdashboardv1beta1 "github.com/oracle/oci-service-operator/api/managementdashboard/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testManagementSavedSearchID = "ocid1.managementsavedsearch.oc1..example"
	testCompartmentID           = "ocid1.compartment.oc1..example"
)

type fakeManagementSavedSearchOCIClient struct {
	create func(context.Context, managementdashboardsdk.CreateManagementSavedSearchRequest) (managementdashboardsdk.CreateManagementSavedSearchResponse, error)
	get    func(context.Context, managementdashboardsdk.GetManagementSavedSearchRequest) (managementdashboardsdk.GetManagementSavedSearchResponse, error)
	list   func(context.Context, managementdashboardsdk.ListManagementSavedSearchesRequest) (managementdashboardsdk.ListManagementSavedSearchesResponse, error)
	update func(context.Context, managementdashboardsdk.UpdateManagementSavedSearchRequest) (managementdashboardsdk.UpdateManagementSavedSearchResponse, error)
	delete func(context.Context, managementdashboardsdk.DeleteManagementSavedSearchRequest) (managementdashboardsdk.DeleteManagementSavedSearchResponse, error)

	createRequests []managementdashboardsdk.CreateManagementSavedSearchRequest
	getRequests    []managementdashboardsdk.GetManagementSavedSearchRequest
	listRequests   []managementdashboardsdk.ListManagementSavedSearchesRequest
	updateRequests []managementdashboardsdk.UpdateManagementSavedSearchRequest
	deleteRequests []managementdashboardsdk.DeleteManagementSavedSearchRequest
}

func (f *fakeManagementSavedSearchOCIClient) CreateManagementSavedSearch(
	ctx context.Context,
	request managementdashboardsdk.CreateManagementSavedSearchRequest,
) (managementdashboardsdk.CreateManagementSavedSearchResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.create == nil {
		return managementdashboardsdk.CreateManagementSavedSearchResponse{}, nil
	}
	return f.create(ctx, request)
}

func (f *fakeManagementSavedSearchOCIClient) GetManagementSavedSearch(
	ctx context.Context,
	request managementdashboardsdk.GetManagementSavedSearchRequest,
) (managementdashboardsdk.GetManagementSavedSearchResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.get == nil {
		return managementdashboardsdk.GetManagementSavedSearchResponse{}, nil
	}
	return f.get(ctx, request)
}

func (f *fakeManagementSavedSearchOCIClient) ListManagementSavedSearches(
	ctx context.Context,
	request managementdashboardsdk.ListManagementSavedSearchesRequest,
) (managementdashboardsdk.ListManagementSavedSearchesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.list == nil {
		return managementdashboardsdk.ListManagementSavedSearchesResponse{}, nil
	}
	return f.list(ctx, request)
}

func (f *fakeManagementSavedSearchOCIClient) UpdateManagementSavedSearch(
	ctx context.Context,
	request managementdashboardsdk.UpdateManagementSavedSearchRequest,
) (managementdashboardsdk.UpdateManagementSavedSearchResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.update == nil {
		return managementdashboardsdk.UpdateManagementSavedSearchResponse{}, nil
	}
	return f.update(ctx, request)
}

func (f *fakeManagementSavedSearchOCIClient) DeleteManagementSavedSearch(
	ctx context.Context,
	request managementdashboardsdk.DeleteManagementSavedSearchRequest,
) (managementdashboardsdk.DeleteManagementSavedSearchResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete == nil {
		return managementdashboardsdk.DeleteManagementSavedSearchResponse{}, nil
	}
	return f.delete(ctx, request)
}

func TestManagementSavedSearchCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	resource := newTestManagementSavedSearch("saved-search-create")
	client := &fakeManagementSavedSearchOCIClient{}
	client.list = func(_ context.Context, request managementdashboardsdk.ListManagementSavedSearchesRequest) (managementdashboardsdk.ListManagementSavedSearchesResponse, error) {
		assertStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
		assertStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
		return managementdashboardsdk.ListManagementSavedSearchesResponse{}, nil
	}
	client.create = func(_ context.Context, request managementdashboardsdk.CreateManagementSavedSearchRequest) (managementdashboardsdk.CreateManagementSavedSearchResponse, error) {
		assertManagementSavedSearchCreateRequest(t, request, resource.Spec)
		if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
			t.Fatal("CreateManagementSavedSearch() opc retry token is empty")
		}
		return managementdashboardsdk.CreateManagementSavedSearchResponse{
			ManagementSavedSearch: sdkManagementSavedSearchFromSpec(testManagementSavedSearchID, resource.Spec, managementdashboardsdk.LifecycleStatesActive),
			OpcRequestId:          common.String("opc-create"),
		}, nil
	}
	client.get = func(_ context.Context, request managementdashboardsdk.GetManagementSavedSearchRequest) (managementdashboardsdk.GetManagementSavedSearchResponse, error) {
		assertStringPtr(t, "get managementSavedSearchId", request.ManagementSavedSearchId, testManagementSavedSearchID)
		return managementdashboardsdk.GetManagementSavedSearchResponse{
			ManagementSavedSearch: sdkManagementSavedSearchFromSpec(testManagementSavedSearchID, resource.Spec, managementdashboardsdk.LifecycleStatesActive),
			OpcRequestId:          common.String("opc-get"),
		}, nil
	}

	response, err := newTestManagementSavedSearchClient(client).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful without requeue", response)
	}
	assertManagementSavedSearchStatus(t, resource, testManagementSavedSearchID, resource.Spec.DisplayName)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, "opc-create")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("CreateManagementSavedSearch() calls = %d, want 1", len(client.createRequests))
	}
}

func TestManagementSavedSearchCreateOrUpdateBindsExistingFromSecondListPage(t *testing.T) {
	resource := newTestManagementSavedSearch("saved-search-bind")
	client := &fakeManagementSavedSearchOCIClient{}
	client.list = func(_ context.Context, request managementdashboardsdk.ListManagementSavedSearchesRequest) (managementdashboardsdk.ListManagementSavedSearchesResponse, error) {
		if request.Page == nil {
			return managementdashboardsdk.ListManagementSavedSearchesResponse{
				ManagementSavedSearchCollection: managementdashboardsdk.ManagementSavedSearchCollection{
					Items: []managementdashboardsdk.ManagementSavedSearchSummary{
						sdkManagementSavedSearchSummary("ocid1.managementsavedsearch.oc1..other", resource.Spec, "other"),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		}
		assertStringPtr(t, "second list page", request.Page, "page-2")
		return managementdashboardsdk.ListManagementSavedSearchesResponse{
			ManagementSavedSearchCollection: managementdashboardsdk.ManagementSavedSearchCollection{
				Items: []managementdashboardsdk.ManagementSavedSearchSummary{
					sdkManagementSavedSearchSummary(testManagementSavedSearchID, resource.Spec, resource.Spec.DisplayName),
				},
			},
		}, nil
	}
	client.get = func(_ context.Context, request managementdashboardsdk.GetManagementSavedSearchRequest) (managementdashboardsdk.GetManagementSavedSearchResponse, error) {
		assertStringPtr(t, "get managementSavedSearchId", request.ManagementSavedSearchId, testManagementSavedSearchID)
		return managementdashboardsdk.GetManagementSavedSearchResponse{
			ManagementSavedSearch: sdkManagementSavedSearchFromSpec(testManagementSavedSearchID, resource.Spec, managementdashboardsdk.LifecycleStatesActive),
		}, nil
	}

	response, err := newTestManagementSavedSearchClient(client).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful bind without requeue", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateManagementSavedSearch() calls = %d, want 0 for bind", len(client.createRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("ListManagementSavedSearches() calls = %d, want 2 pages", len(client.listRequests))
	}
	assertManagementSavedSearchStatus(t, resource, testManagementSavedSearchID, resource.Spec.DisplayName)
}

func TestManagementSavedSearchCreateOrUpdateNoopsWithoutMutableDrift(t *testing.T) {
	resource := newTestManagementSavedSearch("saved-search-noop")
	setManagementSavedSearchStatusID(resource, testManagementSavedSearchID)
	client := &fakeManagementSavedSearchOCIClient{}
	client.get = func(_ context.Context, request managementdashboardsdk.GetManagementSavedSearchRequest) (managementdashboardsdk.GetManagementSavedSearchResponse, error) {
		assertStringPtr(t, "get managementSavedSearchId", request.ManagementSavedSearchId, testManagementSavedSearchID)
		return managementdashboardsdk.GetManagementSavedSearchResponse{
			ManagementSavedSearch: sdkManagementSavedSearchFromSpec(testManagementSavedSearchID, resource.Spec, managementdashboardsdk.LifecycleStatesActive),
		}, nil
	}

	response, err := newTestManagementSavedSearchClient(client).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful no-op", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateManagementSavedSearch() calls = %d, want 0", len(client.updateRequests))
	}
}

func TestManagementSavedSearchCreateOrUpdateSendsMutableUpdate(t *testing.T) {
	resource, currentSpec := newMutableUpdateManagementSavedSearchFixture()
	client := newMutableUpdateManagementSavedSearchClient(t, resource, currentSpec)

	response, err := newTestManagementSavedSearchClient(client).CreateOrUpdate(context.Background(), resource, testRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful update without requeue", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("UpdateManagementSavedSearch() calls = %d, want 1", len(client.updateRequests))
	}
	if got := resource.Status.DisplayName; got != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
}

func TestManagementSavedSearchCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	resource := newTestManagementSavedSearch("saved-search-drift")
	setManagementSavedSearchStatusID(resource, testManagementSavedSearchID)
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"

	currentSpec := resource.Spec
	currentSpec.CompartmentId = testCompartmentID

	client := &fakeManagementSavedSearchOCIClient{}
	client.get = func(_ context.Context, _ managementdashboardsdk.GetManagementSavedSearchRequest) (managementdashboardsdk.GetManagementSavedSearchResponse, error) {
		return managementdashboardsdk.GetManagementSavedSearchResponse{
			ManagementSavedSearch: sdkManagementSavedSearchFromSpec(testManagementSavedSearchID, currentSpec, managementdashboardsdk.LifecycleStatesActive),
		}, nil
	}

	response, err := newTestManagementSavedSearchClient(client).CreateOrUpdate(context.Background(), resource, testRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response.IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateManagementSavedSearch() calls = %d, want 0 after drift rejection", len(client.updateRequests))
	}
}

func TestManagementSavedSearchCreateOrUpdateRejectsOOBMutableDriftBeforeUpdate(t *testing.T) {
	tests := []struct {
		name       string
		currentOOB *bool
	}{
		{name: "spec and readback are OOB", currentOOB: common.Bool(true)},
		{name: "spec is OOB and readback omits OOB flag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := newTestManagementSavedSearch("saved-search-oob")
			setManagementSavedSearchStatusID(resource, testManagementSavedSearchID)
			resource.Spec.IsOobSavedSearch = true
			resource.Spec.Description = "updated description"

			currentSpec := resource.Spec
			currentSpec.Description = "existing description"
			client := &fakeManagementSavedSearchOCIClient{}
			client.get = func(_ context.Context, _ managementdashboardsdk.GetManagementSavedSearchRequest) (managementdashboardsdk.GetManagementSavedSearchResponse, error) {
				current := sdkManagementSavedSearchFromSpec(testManagementSavedSearchID, currentSpec, managementdashboardsdk.LifecycleStatesActive)
				current.IsOobSavedSearch = tt.currentOOB
				return managementdashboardsdk.GetManagementSavedSearchResponse{ManagementSavedSearch: current}, nil
			}

			response, err := newTestManagementSavedSearchClient(client).CreateOrUpdate(context.Background(), resource, testRequest())
			if err == nil {
				t.Fatal("CreateOrUpdate() error = nil, want OOB mutable drift rejection")
			}
			if response.IsSuccessful {
				t.Fatal("CreateOrUpdate() response.IsSuccessful = true, want false")
			}
			if !strings.Contains(err.Error(), "isOobSavedSearch") {
				t.Fatalf("CreateOrUpdate() error = %v, want isOobSavedSearch rejection", err)
			}
			if len(client.updateRequests) != 0 {
				t.Fatalf("UpdateManagementSavedSearch() calls = %d, want 0 after OOB rejection", len(client.updateRequests))
			}
		})
	}
}

func TestManagementSavedSearchDeleteRetainsFinalizerUntilReadbackNotFound(t *testing.T) {
	resource := newTestManagementSavedSearch("saved-search-delete")
	setManagementSavedSearchStatusID(resource, testManagementSavedSearchID)
	client := &fakeManagementSavedSearchOCIClient{}
	getCalls := 0
	client.get = func(_ context.Context, request managementdashboardsdk.GetManagementSavedSearchRequest) (managementdashboardsdk.GetManagementSavedSearchResponse, error) {
		assertStringPtr(t, "get managementSavedSearchId", request.ManagementSavedSearchId, testManagementSavedSearchID)
		getCalls++
		if getCalls == 3 {
			return managementdashboardsdk.GetManagementSavedSearchResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
		}
		return managementdashboardsdk.GetManagementSavedSearchResponse{
			ManagementSavedSearch: sdkManagementSavedSearchFromSpec(testManagementSavedSearchID, resource.Spec, managementdashboardsdk.LifecycleStatesActive),
		}, nil
	}
	client.delete = func(_ context.Context, request managementdashboardsdk.DeleteManagementSavedSearchRequest) (managementdashboardsdk.DeleteManagementSavedSearchResponse, error) {
		assertStringPtr(t, "delete managementSavedSearchId", request.ManagementSavedSearchId, testManagementSavedSearchID)
		return managementdashboardsdk.DeleteManagementSavedSearchResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := newTestManagementSavedSearchClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous NotFound confirmation")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("DeleteManagementSavedSearch() calls = %d, want 1", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete timestamp")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want delete request id", got)
	}
}

func TestManagementSavedSearchDeleteKeepsFinalizerWhileReadbackExists(t *testing.T) {
	resource := newTestManagementSavedSearch("saved-search-delete-pending")
	setManagementSavedSearchStatusID(resource, testManagementSavedSearchID)
	client := &fakeManagementSavedSearchOCIClient{}
	client.get = func(_ context.Context, request managementdashboardsdk.GetManagementSavedSearchRequest) (managementdashboardsdk.GetManagementSavedSearchResponse, error) {
		assertStringPtr(t, "get managementSavedSearchId", request.ManagementSavedSearchId, testManagementSavedSearchID)
		return managementdashboardsdk.GetManagementSavedSearchResponse{
			ManagementSavedSearch: sdkManagementSavedSearchFromSpec(testManagementSavedSearchID, resource.Spec, managementdashboardsdk.LifecycleStatesActive),
		}, nil
	}
	client.delete = func(_ context.Context, _ managementdashboardsdk.DeleteManagementSavedSearchRequest) (managementdashboardsdk.DeleteManagementSavedSearchResponse, error) {
		return managementdashboardsdk.DeleteManagementSavedSearchResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := newTestManagementSavedSearchClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI readback still exists")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil before delete confirmation", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.status.reason = %q, want %q", got, shared.Terminating)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want lifecycle delete tracker")
	}
}

func TestManagementSavedSearchDeleteRejectsAuthShapedPreRead(t *testing.T) {
	resource := newTestManagementSavedSearch("saved-search-auth-shaped-delete")
	setManagementSavedSearchStatusID(resource, testManagementSavedSearchID)
	client := &fakeManagementSavedSearchOCIClient{}
	client.get = func(_ context.Context, _ managementdashboardsdk.GetManagementSavedSearchRequest) (managementdashboardsdk.GetManagementSavedSearchResponse, error) {
		return managementdashboardsdk.GetManagementSavedSearchResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}
	client.delete = func(_ context.Context, _ managementdashboardsdk.DeleteManagementSavedSearchRequest) (managementdashboardsdk.DeleteManagementSavedSearchResponse, error) {
		return managementdashboardsdk.DeleteManagementSavedSearchResponse{}, nil
	}

	deleted, err := newTestManagementSavedSearchClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped pre-read rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteManagementSavedSearch() calls = %d, want 0 after auth-shaped pre-read", len(client.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want ambiguous read request id", got)
	}
}

func TestManagementSavedSearchCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	resource := newTestManagementSavedSearch("saved-search-error")
	client := &fakeManagementSavedSearchOCIClient{}
	client.list = func(_ context.Context, _ managementdashboardsdk.ListManagementSavedSearchesRequest) (managementdashboardsdk.ListManagementSavedSearchesResponse, error) {
		return managementdashboardsdk.ListManagementSavedSearchesResponse{}, nil
	}
	client.create = func(_ context.Context, _ managementdashboardsdk.CreateManagementSavedSearchRequest) (managementdashboardsdk.CreateManagementSavedSearchResponse, error) {
		err := errortest.NewServiceError(500, "InternalError", "create failed")
		err.OpcRequestID = "opc-create-error"
		return managementdashboardsdk.CreateManagementSavedSearchResponse{}, err
	}

	response, err := newTestManagementSavedSearchClient(client).CreateOrUpdate(context.Background(), resource, testRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create failure")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() response.IsSuccessful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-error" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-error", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Failed) {
		t.Fatalf("status.status.reason = %q, want %q", got, shared.Failed)
	}
}

func newMutableUpdateManagementSavedSearchFixture() (
	*managementdashboardv1beta1.ManagementSavedSearch,
	managementdashboardv1beta1.ManagementSavedSearchSpec,
) {
	resource := newTestManagementSavedSearch("saved-search-update")
	setManagementSavedSearchStatusID(resource, testManagementSavedSearchID)
	resource.Spec.DisplayName = "saved-search-updated"
	resource.Spec.Description = "updated description"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}
	resource.Spec.Nls = jsonValue(`{"title":"updated"}`)

	currentSpec := resource.Spec
	currentSpec.DisplayName = "saved-search-old"
	currentSpec.Description = "old description"
	currentSpec.FreeformTags = map[string]string{"env": "dev"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	currentSpec.Nls = jsonValue(`{"title":"old"}`)
	return resource, currentSpec
}

func newMutableUpdateManagementSavedSearchClient(
	t *testing.T,
	resource *managementdashboardv1beta1.ManagementSavedSearch,
	currentSpec managementdashboardv1beta1.ManagementSavedSearchSpec,
) *fakeManagementSavedSearchOCIClient {
	t.Helper()
	getCalls := 0
	client := &fakeManagementSavedSearchOCIClient{}
	client.get = func(_ context.Context, request managementdashboardsdk.GetManagementSavedSearchRequest) (managementdashboardsdk.GetManagementSavedSearchResponse, error) {
		assertStringPtr(t, "get managementSavedSearchId", request.ManagementSavedSearchId, testManagementSavedSearchID)
		getCalls++
		if getCalls == 2 {
			return managementdashboardsdk.GetManagementSavedSearchResponse{
				ManagementSavedSearch: sdkManagementSavedSearchFromSpec(testManagementSavedSearchID, resource.Spec, managementdashboardsdk.LifecycleStatesActive),
			}, nil
		}
		return managementdashboardsdk.GetManagementSavedSearchResponse{
			ManagementSavedSearch: sdkManagementSavedSearchFromSpec(testManagementSavedSearchID, currentSpec, managementdashboardsdk.LifecycleStatesActive),
		}, nil
	}
	client.update = func(_ context.Context, request managementdashboardsdk.UpdateManagementSavedSearchRequest) (managementdashboardsdk.UpdateManagementSavedSearchResponse, error) {
		assertManagementSavedSearchUpdateRequest(t, request, resource.Spec)
		return managementdashboardsdk.UpdateManagementSavedSearchResponse{
			ManagementSavedSearch: sdkManagementSavedSearchFromSpec(testManagementSavedSearchID, resource.Spec, managementdashboardsdk.LifecycleStatesActive),
			OpcRequestId:          common.String("opc-update"),
		}, nil
	}
	return client
}

func assertManagementSavedSearchUpdateRequest(
	t *testing.T,
	request managementdashboardsdk.UpdateManagementSavedSearchRequest,
	spec managementdashboardv1beta1.ManagementSavedSearchSpec,
) {
	t.Helper()
	assertStringPtr(t, "update managementSavedSearchId", request.ManagementSavedSearchId, testManagementSavedSearchID)
	assertStringPtr(t, "update displayName", request.DisplayName, spec.DisplayName)
	assertStringPtr(t, "update description", request.Description, spec.Description)
	if !jsonEquivalent(interfacePointerValue(request.Nls), map[string]interface{}{"title": "updated"}) {
		t.Fatalf("UpdateManagementSavedSearch() nls = %#v, want updated JSON", interfacePointerValue(request.Nls))
	}
	if got := request.FreeformTags["env"]; got != "prod" {
		t.Fatalf("UpdateManagementSavedSearch() freeformTags[env] = %q, want prod", got)
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "84" {
		t.Fatalf("UpdateManagementSavedSearch() definedTags Operations.CostCenter = %#v, want 84", got)
	}
}

func newTestManagementSavedSearchClient(client managementSavedSearchOCIClient) ManagementSavedSearchServiceClient {
	return newManagementSavedSearchServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
}

func newTestManagementSavedSearch(name string) *managementdashboardv1beta1.ManagementSavedSearch {
	return &managementdashboardv1beta1.ManagementSavedSearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       types.UID("uid-" + name),
		},
		Spec: managementdashboardv1beta1.ManagementSavedSearchSpec{
			DisplayName:      name,
			ProviderId:       "log-analytics",
			ProviderVersion:  "3.0.0",
			ProviderName:     "Logging Analytics",
			CompartmentId:    testCompartmentID,
			IsOobSavedSearch: false,
			Description:      "saved search description",
			Nls:              jsonValue(`{"title":"Search"}`),
			Type:             string(managementdashboardsdk.SavedSearchTypesWidgetShowInDashboard),
			UiConfig:         jsonValue(`{"viz":"line"}`),
			DataConfig:       []shared.JSONValue{jsonValue(`{"query":"search *"}`)},
			ScreenImage:      "screen-image",
			MetadataVersion:  "2.0",
			WidgetTemplate:   "template",
			WidgetVM:         "view-model",
			ParametersConfig: []shared.JSONValue{jsonValue(`{"name":"region"}`)},
			FeaturesConfig:   jsonValue(`{"enabled":true}`),
			DrilldownConfig:  []shared.JSONValue{jsonValue(`{"target":"details"}`)},
			FreeformTags:     map[string]string{"env": "dev"},
			DefinedTags:      map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func setManagementSavedSearchStatusID(resource *managementdashboardv1beta1.ManagementSavedSearch, id string) {
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
}

func sdkManagementSavedSearchFromSpec(
	id string,
	spec managementdashboardv1beta1.ManagementSavedSearchSpec,
	state managementdashboardsdk.LifecycleStatesEnum,
) managementdashboardsdk.ManagementSavedSearch {
	return managementdashboardsdk.ManagementSavedSearch{
		Id:               common.String(id),
		DisplayName:      common.String(spec.DisplayName),
		ProviderId:       common.String(spec.ProviderId),
		ProviderVersion:  common.String(spec.ProviderVersion),
		ProviderName:     common.String(spec.ProviderName),
		CompartmentId:    common.String(spec.CompartmentId),
		IsOobSavedSearch: common.Bool(spec.IsOobSavedSearch),
		Description:      common.String(spec.Description),
		Nls:              sdkJSONPointer(mustJSONInterface(spec.Nls)),
		Type:             managementdashboardsdk.SavedSearchTypesEnum(spec.Type),
		UiConfig:         sdkJSONPointer(mustJSONInterface(spec.UiConfig)),
		DataConfig:       mustJSONInterfaceSlice(spec.DataConfig),
		CreatedBy:        common.String("user"),
		UpdatedBy:        common.String("user"),
		ScreenImage:      common.String(spec.ScreenImage),
		MetadataVersion:  common.String(spec.MetadataVersion),
		WidgetTemplate:   common.String(spec.WidgetTemplate),
		WidgetVM:         common.String(spec.WidgetVM),
		LifecycleState:   state,
		ParametersConfig: mustJSONInterfaceSlice(spec.ParametersConfig),
		FeaturesConfig:   sdkJSONPointer(mustJSONInterface(spec.FeaturesConfig)),
		DrilldownConfig:  mustJSONInterfaceSlice(spec.DrilldownConfig),
		FreeformTags:     mapsClone(spec.FreeformTags),
		DefinedTags:      sdkDefinedTags(spec.DefinedTags),
		SystemTags:       map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func sdkManagementSavedSearchSummary(
	id string,
	spec managementdashboardv1beta1.ManagementSavedSearchSpec,
	displayName string,
) managementdashboardsdk.ManagementSavedSearchSummary {
	current := sdkManagementSavedSearchFromSpec(id, spec, managementdashboardsdk.LifecycleStatesActive)
	summary := managementdashboardsdk.ManagementSavedSearchSummary{
		Id:               current.Id,
		DisplayName:      current.DisplayName,
		IsOobSavedSearch: current.IsOobSavedSearch,
		CompartmentId:    current.CompartmentId,
		ProviderId:       current.ProviderId,
		ProviderVersion:  current.ProviderVersion,
		ProviderName:     current.ProviderName,
		Description:      current.Description,
		Nls:              current.Nls,
		Type:             current.Type,
		UiConfig:         current.UiConfig,
		DataConfig:       current.DataConfig,
		CreatedBy:        current.CreatedBy,
		UpdatedBy:        current.UpdatedBy,
		TimeCreated:      current.TimeCreated,
		TimeUpdated:      current.TimeUpdated,
		ScreenImage:      current.ScreenImage,
		MetadataVersion:  current.MetadataVersion,
		WidgetTemplate:   current.WidgetTemplate,
		WidgetVM:         current.WidgetVM,
		LifecycleState:   current.LifecycleState,
		ParametersConfig: current.ParametersConfig,
		FeaturesConfig:   current.FeaturesConfig,
		FreeformTags:     current.FreeformTags,
		DefinedTags:      current.DefinedTags,
		SystemTags:       current.SystemTags,
	}
	summary.DisplayName = common.String(displayName)
	return summary
}

func sdkJSONPointer(value interface{}) *interface{} {
	return &value
}

func mustJSONInterface(value shared.JSONValue) interface{} {
	decoded, err := jsonInterface("test", value)
	if err != nil {
		panic(err)
	}
	return decoded
}

func mustJSONInterfaceSlice(values []shared.JSONValue) []interface{} {
	decoded, err := jsonInterfaceSlice("test", values)
	if err != nil {
		panic(err)
	}
	return decoded
}

func jsonValue(raw string) shared.JSONValue {
	return shared.JSONValue{Raw: []byte(raw)}
}

func sdkDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		children := make(map[string]interface{}, len(values))
		for key, value := range values {
			children[key] = value
		}
		converted[namespace] = children
	}
	return converted
}

func mapsClone(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func testRequest() ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "resource"}}
}

func assertManagementSavedSearchCreateRequest(
	t *testing.T,
	request managementdashboardsdk.CreateManagementSavedSearchRequest,
	spec managementdashboardv1beta1.ManagementSavedSearchSpec,
) {
	t.Helper()
	assertStringPtr(t, "create displayName", request.DisplayName, spec.DisplayName)
	assertStringPtr(t, "create providerId", request.ProviderId, spec.ProviderId)
	assertStringPtr(t, "create compartmentId", request.CompartmentId, spec.CompartmentId)
	assertBoolPtr(t, "create isOobSavedSearch", request.IsOobSavedSearch, spec.IsOobSavedSearch)
	if request.Type != managementdashboardsdk.SavedSearchTypesEnum(spec.Type) {
		t.Fatalf("CreateManagementSavedSearch() type = %q, want %q", request.Type, spec.Type)
	}
	if !jsonEquivalent(interfacePointerValue(request.Nls), map[string]interface{}{"title": "Search"}) {
		t.Fatalf("CreateManagementSavedSearch() nls = %#v, want title JSON", interfacePointerValue(request.Nls))
	}
	if len(request.DataConfig) != 1 || !jsonEquivalent(request.DataConfig[0], map[string]interface{}{"query": "search *"}) {
		t.Fatalf("CreateManagementSavedSearch() dataConfig = %#v, want query JSON", request.DataConfig)
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("CreateManagementSavedSearch() definedTags Operations.CostCenter = %#v, want 42", got)
	}
}

func assertManagementSavedSearchStatus(
	t *testing.T,
	resource *managementdashboardv1beta1.ManagementSavedSearch,
	id string,
	displayName string,
) {
	t.Helper()
	if got := resource.Status.Id; got != id {
		t.Fatalf("status.id = %q, want %q", got, id)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != id {
		t.Fatalf("status.status.ocid = %q, want %q", got, id)
	}
	if got := resource.Status.DisplayName; got != displayName {
		t.Fatalf("status.displayName = %q, want %q", got, displayName)
	}
	if got := resource.Status.LifecycleState; got != string(managementdashboardsdk.LifecycleStatesActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Active) {
		t.Fatalf("status.status.reason = %q, want %q", got, shared.Active)
	}
}

func assertStringPtr(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func assertBoolPtr(t *testing.T, label string, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %t", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %t, want %t", label, *got, want)
	}
}
