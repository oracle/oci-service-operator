/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package listingrevisionpackage

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeListingRevisionPackageOCIClient struct {
	createRequests []marketplacepublishersdk.CreateListingRevisionPackageRequest
	getRequests    []marketplacepublishersdk.GetListingRevisionPackageRequest
	listRequests   []marketplacepublishersdk.ListListingRevisionPackagesRequest
	updateRequests []marketplacepublishersdk.UpdateListingRevisionPackageRequest
	deleteRequests []marketplacepublishersdk.DeleteListingRevisionPackageRequest

	create func(context.Context, marketplacepublishersdk.CreateListingRevisionPackageRequest) (marketplacepublishersdk.CreateListingRevisionPackageResponse, error)
	get    func(context.Context, marketplacepublishersdk.GetListingRevisionPackageRequest) (marketplacepublishersdk.GetListingRevisionPackageResponse, error)
	list   func(context.Context, marketplacepublishersdk.ListListingRevisionPackagesRequest) (marketplacepublishersdk.ListListingRevisionPackagesResponse, error)
	update func(context.Context, marketplacepublishersdk.UpdateListingRevisionPackageRequest) (marketplacepublishersdk.UpdateListingRevisionPackageResponse, error)
	delete func(context.Context, marketplacepublishersdk.DeleteListingRevisionPackageRequest) (marketplacepublishersdk.DeleteListingRevisionPackageResponse, error)
}

func (f *fakeListingRevisionPackageOCIClient) CreateListingRevisionPackage(
	ctx context.Context,
	request marketplacepublishersdk.CreateListingRevisionPackageRequest,
) (marketplacepublishersdk.CreateListingRevisionPackageResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.create == nil {
		return marketplacepublishersdk.CreateListingRevisionPackageResponse{}, nil
	}
	return f.create(ctx, request)
}

func (f *fakeListingRevisionPackageOCIClient) GetListingRevisionPackage(
	ctx context.Context,
	request marketplacepublishersdk.GetListingRevisionPackageRequest,
) (marketplacepublishersdk.GetListingRevisionPackageResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.get == nil {
		return marketplacepublishersdk.GetListingRevisionPackageResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
	}
	return f.get(ctx, request)
}

func (f *fakeListingRevisionPackageOCIClient) ListListingRevisionPackages(
	ctx context.Context,
	request marketplacepublishersdk.ListListingRevisionPackagesRequest,
) (marketplacepublishersdk.ListListingRevisionPackagesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.list == nil {
		return marketplacepublishersdk.ListListingRevisionPackagesResponse{}, nil
	}
	return f.list(ctx, request)
}

func (f *fakeListingRevisionPackageOCIClient) UpdateListingRevisionPackage(
	ctx context.Context,
	request marketplacepublishersdk.UpdateListingRevisionPackageRequest,
) (marketplacepublishersdk.UpdateListingRevisionPackageResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.update == nil {
		return marketplacepublishersdk.UpdateListingRevisionPackageResponse{}, nil
	}
	return f.update(ctx, request)
}

func (f *fakeListingRevisionPackageOCIClient) DeleteListingRevisionPackage(
	ctx context.Context,
	request marketplacepublishersdk.DeleteListingRevisionPackageRequest,
) (marketplacepublishersdk.DeleteListingRevisionPackageResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete == nil {
		return marketplacepublishersdk.DeleteListingRevisionPackageResponse{}, nil
	}
	return f.delete(ctx, request)
}

func TestListingRevisionPackageCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	resource := testListingRevisionPackageResource()
	createOpc := "create-opc"
	readOpc := "read-opc"

	fakeClient := &fakeListingRevisionPackageOCIClient{}
	fakeClient.list = func(context.Context, marketplacepublishersdk.ListListingRevisionPackagesRequest) (marketplacepublishersdk.ListListingRevisionPackagesResponse, error) {
		return marketplacepublishersdk.ListListingRevisionPackagesResponse{}, nil
	}
	fakeClient.create = func(_ context.Context, request marketplacepublishersdk.CreateListingRevisionPackageRequest) (marketplacepublishersdk.CreateListingRevisionPackageResponse, error) {
		return marketplacepublishersdk.CreateListingRevisionPackageResponse{
			ListingRevisionPackage: testSDKListingRevisionPackage(resource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateCreating),
			OpcRequestId:           &createOpc,
		}, nil
	}
	fakeClient.get = func(_ context.Context, request marketplacepublishersdk.GetListingRevisionPackageRequest) (marketplacepublishersdk.GetListingRevisionPackageResponse, error) {
		if got := stringValue(request.ListingRevisionPackageId); got != "package-1" {
			t.Fatalf("GetListingRevisionPackage id = %q, want package-1", got)
		}
		return marketplacepublishersdk.GetListingRevisionPackageResponse{
			ListingRevisionPackage: testSDKListingRevisionPackage(resource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive),
			OpcRequestId:           &readOpc,
		}, nil
	}

	response, err := newListingRevisionPackageServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fakeClient).
		CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue once follow-up read is ACTIVE")
	}
	if len(fakeClient.createRequests) != 1 {
		t.Fatalf("CreateListingRevisionPackage calls = %d, want 1", len(fakeClient.createRequests))
	}
	assertListingRevisionPackageCreateDetails(t, fakeClient.createRequests[0].CreateListingRevisionPackageDetails, resource)
	assertListingRevisionPackageCreatedStatus(t, resource, createOpc)
}

func TestListingRevisionPackageCreateOrUpdateUsesCreateResponseWhenFollowUpReadIsNotFound(t *testing.T) {
	resource := testListingRevisionPackageResource()
	createOpc := "create-opc"

	fakeClient := &fakeListingRevisionPackageOCIClient{}
	fakeClient.list = func(context.Context, marketplacepublishersdk.ListListingRevisionPackagesRequest) (marketplacepublishersdk.ListListingRevisionPackagesResponse, error) {
		return marketplacepublishersdk.ListListingRevisionPackagesResponse{}, nil
	}
	fakeClient.create = func(context.Context, marketplacepublishersdk.CreateListingRevisionPackageRequest) (marketplacepublishersdk.CreateListingRevisionPackageResponse, error) {
		return marketplacepublishersdk.CreateListingRevisionPackageResponse{
			ListingRevisionPackage: testSDKListingRevisionPackage(resource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive),
			OpcRequestId:           &createOpc,
		}, nil
	}
	fakeClient.get = func(context.Context, marketplacepublishersdk.GetListingRevisionPackageRequest) (marketplacepublishersdk.GetListingRevisionPackageResponse, error) {
		return marketplacepublishersdk.GetListingRevisionPackageResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "eventual read miss")
	}

	response, err := newListingRevisionPackageServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fakeClient).
		CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success from projected create response")
	}
	if resource.Status.Id != "package-1" || resource.Status.Status != string(marketplacepublishersdk.ListingRevisionPackageStatusNew) {
		t.Fatalf("status id/status = %q/%q, want package-1/NEW", resource.Status.Id, resource.Status.Status)
	}
	if resource.Status.OsokStatus.OpcRequestID != createOpc {
		t.Fatalf("status opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, createOpc)
	}
}

func TestListingRevisionPackageCreateOrUpdateBindsExistingAcrossListPages(t *testing.T) {
	resource := testListingRevisionPackageResource()
	nextPage := "next"

	fakeClient := &fakeListingRevisionPackageOCIClient{}
	fakeClient.list = func(_ context.Context, request marketplacepublishersdk.ListListingRevisionPackagesRequest) (marketplacepublishersdk.ListListingRevisionPackagesResponse, error) {
		if request.Page == nil {
			return marketplacepublishersdk.ListListingRevisionPackagesResponse{
				ListingRevisionPackageCollection: marketplacepublishersdk.ListingRevisionPackageCollection{
					Items: []marketplacepublishersdk.ListingRevisionPackageSummary{
						testListingRevisionPackageSummary(resource, "other-package", "0.9.0", marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive),
					},
				},
				OpcNextPage: &nextPage,
			}, nil
		}
		return marketplacepublishersdk.ListListingRevisionPackagesResponse{
			ListingRevisionPackageCollection: marketplacepublishersdk.ListingRevisionPackageCollection{
				Items: []marketplacepublishersdk.ListingRevisionPackageSummary{
					testListingRevisionPackageSummary(resource, "package-1", resource.Spec.PackageVersion, marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive),
				},
			},
		}, nil
	}
	fakeClient.get = func(context.Context, marketplacepublishersdk.GetListingRevisionPackageRequest) (marketplacepublishersdk.GetListingRevisionPackageResponse, error) {
		return marketplacepublishersdk.GetListingRevisionPackageResponse{
			ListingRevisionPackage: testSDKListingRevisionPackage(resource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive),
		}, nil
	}

	response, err := newListingRevisionPackageServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fakeClient).
		CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fakeClient.listRequests) != 2 {
		t.Fatalf("ListListingRevisionPackages calls = %d, want 2", len(fakeClient.listRequests))
	}
	if len(fakeClient.createRequests) != 0 {
		t.Fatal("CreateOrUpdate() should bind existing package instead of creating")
	}
	if len(fakeClient.updateRequests) != 0 {
		t.Fatal("CreateOrUpdate() should not update when observed state matches")
	}
	if resource.Status.Id != "package-1" {
		t.Fatalf("status.id = %q, want package-1", resource.Status.Id)
	}
}

func TestListingRevisionPackageCreateOrUpdateBindsAndUpdatesDriftedDisplayName(t *testing.T) {
	resource := testListingRevisionPackageResource()
	currentResource := testListingRevisionPackageResource()
	currentResource.Spec.DisplayName = "previous package"
	updateOpc := "update-opc"

	fakeClient := &fakeListingRevisionPackageOCIClient{}
	fakeClient.list = func(_ context.Context, request marketplacepublishersdk.ListListingRevisionPackagesRequest) (marketplacepublishersdk.ListListingRevisionPackagesResponse, error) {
		assertListingRevisionPackageListRequestOmitsDisplayName(t, request)
		return marketplacepublishersdk.ListListingRevisionPackagesResponse{
			ListingRevisionPackageCollection: marketplacepublishersdk.ListingRevisionPackageCollection{
				Items: []marketplacepublishersdk.ListingRevisionPackageSummary{
					testListingRevisionPackageSummaryWithDisplayName(
						currentResource,
						"package-1",
						resource.Spec.PackageVersion,
						marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive,
						currentResource.Spec.DisplayName,
					),
				},
			},
		}, nil
	}
	getCalls := 0
	fakeClient.get = func(context.Context, marketplacepublishersdk.GetListingRevisionPackageRequest) (marketplacepublishersdk.GetListingRevisionPackageResponse, error) {
		getCalls++
		if getCalls == 1 {
			return marketplacepublishersdk.GetListingRevisionPackageResponse{
				ListingRevisionPackage: testSDKListingRevisionPackageWithDisplayName(
					currentResource,
					"package-1",
					marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive,
					currentResource.Spec.DisplayName,
				),
			}, nil
		}
		return marketplacepublishersdk.GetListingRevisionPackageResponse{
			ListingRevisionPackage: testSDKListingRevisionPackage(resource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive),
		}, nil
	}
	fakeClient.update = func(_ context.Context, request marketplacepublishersdk.UpdateListingRevisionPackageRequest) (marketplacepublishersdk.UpdateListingRevisionPackageResponse, error) {
		return marketplacepublishersdk.UpdateListingRevisionPackageResponse{
			ListingRevisionPackage: testSDKListingRevisionPackage(resource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateUpdating),
			OpcRequestId:           &updateOpc,
		}, nil
	}

	response, err := newListingRevisionPackageServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fakeClient).
		CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fakeClient.createRequests) != 0 {
		t.Fatal("CreateOrUpdate() should bind existing package instead of creating")
	}
	updateDetails := requireSingleListingRevisionPackageUpdateDetails(t, fakeClient)
	if got := stringValue(updateDetails.DisplayName); got != resource.Spec.DisplayName {
		t.Fatalf("update displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
}

func TestListingRevisionPackageCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	resource := testListingRevisionPackageResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("package-1")
	resource.Spec.Description = "new description"
	resource.Spec.AreSecurityUpgradesProvided = true
	resource.Spec.IsDefault = false
	resource.Spec.FreeformTags = map[string]string{"env": "test"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"ns": {"team": "runtime"}}
	updateOpc := "update-opc"

	currentResource := testListingRevisionPackageResource()
	currentResource.Spec.Description = "old description"
	currentResource.Spec.AreSecurityUpgradesProvided = false
	currentResource.Spec.IsDefault = true
	currentResource.Spec.FreeformTags = map[string]string{"env": "old"}
	currentResource.Spec.DefinedTags = map[string]shared.MapValue{"ns": {"team": "old"}}

	fakeClient := &fakeListingRevisionPackageOCIClient{}
	getCalls := 0
	fakeClient.get = func(context.Context, marketplacepublishersdk.GetListingRevisionPackageRequest) (marketplacepublishersdk.GetListingRevisionPackageResponse, error) {
		getCalls++
		if getCalls == 1 {
			return marketplacepublishersdk.GetListingRevisionPackageResponse{
				ListingRevisionPackage: testSDKListingRevisionPackage(currentResource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive),
			}, nil
		}
		return marketplacepublishersdk.GetListingRevisionPackageResponse{
			ListingRevisionPackage: testSDKListingRevisionPackage(resource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive),
		}, nil
	}
	fakeClient.update = func(_ context.Context, request marketplacepublishersdk.UpdateListingRevisionPackageRequest) (marketplacepublishersdk.UpdateListingRevisionPackageResponse, error) {
		return marketplacepublishersdk.UpdateListingRevisionPackageResponse{
			ListingRevisionPackage: testSDKListingRevisionPackage(resource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateUpdating),
			OpcRequestId:           &updateOpc,
		}, nil
	}

	response, err := newListingRevisionPackageServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fakeClient).
		CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	updateDetails := requireSingleListingRevisionPackageUpdateDetails(t, fakeClient)
	assertListingRevisionPackageMutableUpdateDetails(t, updateDetails, resource)
	if resource.Status.OsokStatus.OpcRequestID != updateOpc {
		t.Fatalf("status opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, updateOpc)
	}
}

func TestListingRevisionPackageCreateOrUpdateRejectsCreateOnlyDrift(t *testing.T) {
	resource := testListingRevisionPackageResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("package-1")
	currentResource := testListingRevisionPackageResource()
	currentResource.Spec.ListingRevisionId = "different-revision"

	fakeClient := &fakeListingRevisionPackageOCIClient{}
	fakeClient.get = func(context.Context, marketplacepublishersdk.GetListingRevisionPackageRequest) (marketplacepublishersdk.GetListingRevisionPackageResponse, error) {
		return marketplacepublishersdk.GetListingRevisionPackageResponse{
			ListingRevisionPackage: testSDKListingRevisionPackage(currentResource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive),
		}, nil
	}

	_, err := newListingRevisionPackageServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fakeClient).
		CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "listingRevisionId") {
		t.Fatalf("CreateOrUpdate() error = %v, want listingRevisionId drift rejection", err)
	}
	if len(fakeClient.updateRequests) != 0 {
		t.Fatal("CreateOrUpdate() should reject create-only drift before OCI update")
	}
}

func TestListingRevisionPackageDeleteRetainsFinalizerWhileDeleteIsPending(t *testing.T) {
	resource := testListingRevisionPackageResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("package-1")
	deleteOpc := "delete-opc"

	getResponses := []marketplacepublishersdk.ListingRevisionPackage{
		testSDKListingRevisionPackage(resource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive),
		testSDKListingRevisionPackage(resource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive),
		testSDKListingRevisionPackage(resource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateDeleting),
	}
	fakeClient := &fakeListingRevisionPackageOCIClient{}
	fakeClient.get = func(context.Context, marketplacepublishersdk.GetListingRevisionPackageRequest) (marketplacepublishersdk.GetListingRevisionPackageResponse, error) {
		if len(getResponses) == 0 {
			t.Fatal("unexpected GetListingRevisionPackage call")
		}
		next := getResponses[0]
		getResponses = getResponses[1:]
		return marketplacepublishersdk.GetListingRevisionPackageResponse{ListingRevisionPackage: next}, nil
	}
	fakeClient.delete = func(context.Context, marketplacepublishersdk.DeleteListingRevisionPackageRequest) (marketplacepublishersdk.DeleteListingRevisionPackageResponse, error) {
		return marketplacepublishersdk.DeleteListingRevisionPackageResponse{OpcRequestId: &deleteOpc}, nil
	}

	deleted, err := newListingRevisionPackageServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fakeClient).
		Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should retain finalizer while OCI lifecycle is DELETING")
	}
	if len(fakeClient.deleteRequests) != 1 {
		t.Fatalf("DeleteListingRevisionPackage calls = %d, want 1", len(fakeClient.deleteRequests))
	}
	if resource.Status.LifecycleState != string(marketplacepublishersdk.ListingRevisionPackageLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status async current = %#v, want delete lifecycle operation", resource.Status.OsokStatus.Async.Current)
	}
	if resource.Status.OsokStatus.OpcRequestID != deleteOpc {
		t.Fatalf("status opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, deleteOpc)
	}
}

func TestListingRevisionPackageDeleteCompletesOnDeletedLifecycle(t *testing.T) {
	resource := testListingRevisionPackageResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("package-1")

	getResponses := []marketplacepublishersdk.ListingRevisionPackage{
		testSDKListingRevisionPackage(resource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive),
		testSDKListingRevisionPackage(resource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateActive),
		testSDKListingRevisionPackage(resource, "package-1", marketplacepublishersdk.ListingRevisionPackageLifecycleStateDeleted),
	}
	fakeClient := &fakeListingRevisionPackageOCIClient{}
	fakeClient.get = func(context.Context, marketplacepublishersdk.GetListingRevisionPackageRequest) (marketplacepublishersdk.GetListingRevisionPackageResponse, error) {
		next := getResponses[0]
		getResponses = getResponses[1:]
		return marketplacepublishersdk.GetListingRevisionPackageResponse{ListingRevisionPackage: next}, nil
	}

	deleted, err := newListingRevisionPackageServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fakeClient).
		Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should release finalizer once OCI lifecycle is DELETED")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("Delete() should mark OSOK status deleted")
	}
}

func TestListingRevisionPackageDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	resource := testListingRevisionPackageResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("package-1")
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	authErr.OpcRequestID = "auth-opc"

	fakeClient := &fakeListingRevisionPackageOCIClient{}
	fakeClient.get = func(context.Context, marketplacepublishersdk.GetListingRevisionPackageRequest) (marketplacepublishersdk.GetListingRevisionPackageResponse, error) {
		return marketplacepublishersdk.GetListingRevisionPackageResponse{}, authErr
	}

	deleted, err := newListingRevisionPackageServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fakeClient).
		Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "refusing to call delete") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped 404 refusal", err)
	}
	if deleted {
		t.Fatal("Delete() should not report deleted for ambiguous auth-shaped 404")
	}
	if len(fakeClient.deleteRequests) != 0 {
		t.Fatal("Delete() should not call OCI delete after ambiguous pre-delete read")
	}
	if resource.Status.OsokStatus.OpcRequestID != "auth-opc" {
		t.Fatalf("status opcRequestId = %q, want auth-opc", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestListingRevisionPackageCreateOrUpdateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := testListingRevisionPackageResource()
	createErr := errortest.NewServiceError(500, "InternalError", "create failed")
	createErr.OpcRequestID = "create-error-opc"

	fakeClient := &fakeListingRevisionPackageOCIClient{}
	fakeClient.list = func(context.Context, marketplacepublishersdk.ListListingRevisionPackagesRequest) (marketplacepublishersdk.ListListingRevisionPackagesResponse, error) {
		return marketplacepublishersdk.ListListingRevisionPackagesResponse{}, nil
	}
	fakeClient.create = func(context.Context, marketplacepublishersdk.CreateListingRevisionPackageRequest) (marketplacepublishersdk.CreateListingRevisionPackageResponse, error) {
		return marketplacepublishersdk.CreateListingRevisionPackageResponse{}, createErr
	}

	response, err := newListingRevisionPackageServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fakeClient).
		CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI failure")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report unsuccessful response for OCI failure")
	}
	if resource.Status.OsokStatus.OpcRequestID != "create-error-opc" {
		t.Fatalf("status opcRequestId = %q, want create-error-opc", resource.Status.OsokStatus.OpcRequestID)
	}
}

func testListingRevisionPackageResource() *marketplacepublisherv1beta1.ListingRevisionPackage {
	return &marketplacepublisherv1beta1.ListingRevisionPackage{
		Spec: marketplacepublisherv1beta1.ListingRevisionPackageSpec{
			ListingRevisionId:           "revision-1",
			PackageVersion:              "1.0.0",
			ArtifactId:                  "artifact-1",
			TermId:                      "term-1",
			AreSecurityUpgradesProvided: false,
			DisplayName:                 "package",
			Description:                 "description",
			IsDefault:                   true,
			FreeformTags:                map[string]string{"env": "dev"},
			DefinedTags:                 map[string]shared.MapValue{"ns": {"team": "osok"}},
		},
	}
}

func assertListingRevisionPackageCreateDetails(
	t *testing.T,
	details marketplacepublishersdk.CreateListingRevisionPackageDetails,
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
) {
	t.Helper()
	if got := stringValue(details.ListingRevisionId); got != resource.Spec.ListingRevisionId {
		t.Fatalf("create listingRevisionId = %q, want %q", got, resource.Spec.ListingRevisionId)
	}
	if details.AreSecurityUpgradesProvided == nil || *details.AreSecurityUpgradesProvided {
		t.Fatalf("create areSecurityUpgradesProvided = %v, want explicit false", details.AreSecurityUpgradesProvided)
	}
	if details.IsDefault == nil || !*details.IsDefault {
		t.Fatalf("create isDefault = %v, want true", details.IsDefault)
	}
}

func assertListingRevisionPackageCreatedStatus(
	t *testing.T,
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
	createOpc string,
) {
	t.Helper()
	if resource.Status.Id != "package-1" || string(resource.Status.OsokStatus.Ocid) != "package-1" {
		t.Fatalf("status id = %q / %q, want package-1", resource.Status.Id, resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.Status != string(marketplacepublishersdk.ListingRevisionPackageStatusNew) {
		t.Fatalf("status.sdkStatus = %q, want NEW", resource.Status.Status)
	}
	if resource.Status.PackageType != string(marketplacepublishersdk.PackageTypeMachineImage) {
		t.Fatalf("status.packageType = %q, want MACHINE_IMAGE", resource.Status.PackageType)
	}
	if !strings.Contains(resource.Status.JsonData, `"packageType":"MACHINE_IMAGE"`) {
		t.Fatalf("status.jsonData = %q, want machine image discriminator", resource.Status.JsonData)
	}
	if resource.Status.OsokStatus.OpcRequestID != createOpc {
		t.Fatalf("status opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, createOpc)
	}
}

func assertListingRevisionPackageListRequestOmitsDisplayName(
	t *testing.T,
	request marketplacepublishersdk.ListListingRevisionPackagesRequest,
) {
	t.Helper()
	if request.DisplayName != nil {
		t.Fatalf("list request displayName = %q, want omitted", *request.DisplayName)
	}
}

func requireSingleListingRevisionPackageUpdateDetails(
	t *testing.T,
	client *fakeListingRevisionPackageOCIClient,
) marketplacepublishersdk.UpdateListingRevisionPackageDetails {
	t.Helper()
	if len(client.updateRequests) != 1 {
		t.Fatalf("UpdateListingRevisionPackage calls = %d, want 1", len(client.updateRequests))
	}
	return client.updateRequests[0].UpdateListingRevisionPackageDetails
}

func assertListingRevisionPackageMutableUpdateDetails(
	t *testing.T,
	details marketplacepublishersdk.UpdateListingRevisionPackageDetails,
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
) {
	t.Helper()
	if got := stringValue(details.Description); got != "new description" {
		t.Fatalf("update description = %q, want new description", got)
	}
	if details.AreSecurityUpgradesProvided == nil || !*details.AreSecurityUpgradesProvided {
		t.Fatalf("update areSecurityUpgradesProvided = %v, want true", details.AreSecurityUpgradesProvided)
	}
	if details.IsDefault == nil || *details.IsDefault {
		t.Fatalf("update isDefault = %v, want explicit false", details.IsDefault)
	}
	if !reflect.DeepEqual(details.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("update freeformTags = %#v, want %#v", details.FreeformTags, resource.Spec.FreeformTags)
	}
	if got := details.DefinedTags["ns"]["team"]; got != "runtime" {
		t.Fatalf("update definedTags ns.team = %#v, want runtime", got)
	}
}

func testSDKListingRevisionPackage(
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
	id string,
	lifecycleState marketplacepublishersdk.ListingRevisionPackageLifecycleStateEnum,
) marketplacepublishersdk.MachineImagePackage {
	return marketplacepublishersdk.MachineImagePackage{
		Id:                          common.String(id),
		DisplayName:                 common.String(resource.Spec.DisplayName),
		ListingRevisionId:           common.String(resource.Spec.ListingRevisionId),
		CompartmentId:               common.String("compartment-1"),
		ArtifactId:                  common.String(resource.Spec.ArtifactId),
		TermId:                      common.String(resource.Spec.TermId),
		PackageVersion:              common.String(resource.Spec.PackageVersion),
		Description:                 common.String(resource.Spec.Description),
		AreSecurityUpgradesProvided: common.Bool(resource.Spec.AreSecurityUpgradesProvided),
		IsDefault:                   common.Bool(resource.Spec.IsDefault),
		LifecycleState:              lifecycleState,
		Status:                      marketplacepublishersdk.ListingRevisionPackageStatusNew,
		FreeformTags:                mapsClone(resource.Spec.FreeformTags),
		DefinedTags:                 testDefinedTags(resource.Spec.DefinedTags),
		MachineImageDetails: &marketplacepublishersdk.MachineImagePackageDetails{
			ComputeImageId: common.String("image-1"),
		},
	}
}

func testSDKListingRevisionPackageWithDisplayName(
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
	id string,
	lifecycleState marketplacepublishersdk.ListingRevisionPackageLifecycleStateEnum,
	displayName string,
) marketplacepublishersdk.MachineImagePackage {
	pkg := testSDKListingRevisionPackage(resource, id, lifecycleState)
	pkg.DisplayName = common.String(displayName)
	return pkg
}

func testListingRevisionPackageSummary(
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
	id string,
	version string,
	lifecycleState marketplacepublishersdk.ListingRevisionPackageLifecycleStateEnum,
) marketplacepublishersdk.ListingRevisionPackageSummary {
	return marketplacepublishersdk.ListingRevisionPackageSummary{
		Id:                          common.String(id),
		ListingRevisionId:           common.String(resource.Spec.ListingRevisionId),
		CompartmentId:               common.String("compartment-1"),
		DisplayName:                 common.String(resource.Spec.DisplayName),
		PackageVersion:              common.String(version),
		PackageType:                 marketplacepublishersdk.PackageTypeMachineImage,
		AreSecurityUpgradesProvided: common.Bool(resource.Spec.AreSecurityUpgradesProvided),
		LifecycleState:              lifecycleState,
		Status:                      marketplacepublishersdk.ListingRevisionPackageStatusNew,
	}
}

func testListingRevisionPackageSummaryWithDisplayName(
	resource *marketplacepublisherv1beta1.ListingRevisionPackage,
	id string,
	version string,
	lifecycleState marketplacepublishersdk.ListingRevisionPackageLifecycleStateEnum,
	displayName string,
) marketplacepublishersdk.ListingRevisionPackageSummary {
	summary := testListingRevisionPackageSummary(resource, id, version, lifecycleState)
	summary.DisplayName = common.String(displayName)
	return summary
}

func testDefinedTags(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		converted[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[namespace][key] = value
		}
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

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
