/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package productlicense

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	licensemanagersdk "github.com/oracle/oci-go-sdk/v65/licensemanager"
	licensemanagerv1beta1 "github.com/oracle/oci-service-operator/api/licensemanager/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testProductLicenseID = "ocid1.productlicense.oc1..example"
	testCompartmentID    = "ocid1.compartment.oc1..example"
)

type fakeProductLicenseOCIClient struct {
	createFn func(context.Context, licensemanagersdk.CreateProductLicenseRequest) (licensemanagersdk.CreateProductLicenseResponse, error)
	getFn    func(context.Context, licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error)
	listFn   func(context.Context, licensemanagersdk.ListProductLicensesRequest) (licensemanagersdk.ListProductLicensesResponse, error)
	updateFn func(context.Context, licensemanagersdk.UpdateProductLicenseRequest) (licensemanagersdk.UpdateProductLicenseResponse, error)
	deleteFn func(context.Context, licensemanagersdk.DeleteProductLicenseRequest) (licensemanagersdk.DeleteProductLicenseResponse, error)

	createRequests []licensemanagersdk.CreateProductLicenseRequest
	getRequests    []licensemanagersdk.GetProductLicenseRequest
	listRequests   []licensemanagersdk.ListProductLicensesRequest
	updateRequests []licensemanagersdk.UpdateProductLicenseRequest
	deleteRequests []licensemanagersdk.DeleteProductLicenseRequest
}

func (f *fakeProductLicenseOCIClient) CreateProductLicense(
	ctx context.Context,
	request licensemanagersdk.CreateProductLicenseRequest,
) (licensemanagersdk.CreateProductLicenseResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return licensemanagersdk.CreateProductLicenseResponse{}, nil
}

func (f *fakeProductLicenseOCIClient) GetProductLicense(
	ctx context.Context,
	request licensemanagersdk.GetProductLicenseRequest,
) (licensemanagersdk.GetProductLicenseResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return licensemanagersdk.GetProductLicenseResponse{}, nil
}

func (f *fakeProductLicenseOCIClient) ListProductLicenses(
	ctx context.Context,
	request licensemanagersdk.ListProductLicensesRequest,
) (licensemanagersdk.ListProductLicensesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return licensemanagersdk.ListProductLicensesResponse{}, nil
}

func (f *fakeProductLicenseOCIClient) UpdateProductLicense(
	ctx context.Context,
	request licensemanagersdk.UpdateProductLicenseRequest,
) (licensemanagersdk.UpdateProductLicenseResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return licensemanagersdk.UpdateProductLicenseResponse{}, nil
}

func (f *fakeProductLicenseOCIClient) DeleteProductLicense(
	ctx context.Context,
	request licensemanagersdk.DeleteProductLicenseRequest,
) (licensemanagersdk.DeleteProductLicenseResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return licensemanagersdk.DeleteProductLicenseResponse{}, nil
}

func TestProductLicenseRuntimeHooksConfigureReviewedSemantics(t *testing.T) {
	t.Parallel()

	hooks := newProductLicenseRuntimeHooksWithOCIClient(&fakeProductLicenseOCIClient{})
	applyProductLicenseRuntimeHooks(&ProductLicenseServiceManager{}, &hooks, &fakeProductLicenseOCIClient{}, nil)

	if hooks.Semantics == nil {
		t.Fatal("Semantics = nil, want reviewed generatedruntime semantics")
	}
	if got := hooks.Semantics.Async.Strategy; got != "lifecycle" {
		t.Fatalf("Async.Strategy = %q, want lifecycle", got)
	}
	if hooks.Semantics.Delete.Policy != "required" || hooks.Semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", hooks.Semantics.Delete, hooks.Semantics.DeleteFollowUp)
	}
	assertProductLicenseStringSliceContains(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "images", "freeformTags", "definedTags")
	assertProductLicenseStringSliceContains(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "compartmentId", "displayName", "isVendorOracle", "licenseUnit", "vendorName")
	if hooks.StatusHooks.ProjectStatus == nil {
		t.Fatal("StatusHooks.ProjectStatus = nil, want status collision-safe projection")
	}
	if len(hooks.WrapGeneratedClient) == 0 {
		t.Fatal("WrapGeneratedClient = empty, want status collision-safe runtime wrapper")
	}
}

func TestProductLicenseCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := productLicenseResource()
	client := &fakeProductLicenseOCIClient{}
	client.listFn = func(_ context.Context, request licensemanagersdk.ListProductLicensesRequest) (licensemanagersdk.ListProductLicensesResponse, error) {
		requireStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
		return licensemanagersdk.ListProductLicensesResponse{}, nil
	}
	client.createFn = func(_ context.Context, request licensemanagersdk.CreateProductLicenseRequest) (licensemanagersdk.CreateProductLicenseResponse, error) {
		assertProductLicenseCreateRequest(t, request, resource)
		return licensemanagersdk.CreateProductLicenseResponse{
			ProductLicense: productLicenseSDK(testProductLicenseID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
			OpcRequestId:   common.String("opc-create"),
		}, nil
	}
	client.getFn = func(_ context.Context, request licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
		requireStringPtr(t, "get productLicenseId", request.ProductLicenseId, testProductLicenseID)
		return licensemanagersdk.GetProductLicenseResponse{
			ProductLicense: productLicenseSDK(testProductLicenseID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
		}, nil
	}

	response, err := newProductLicenseServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active success without requeue", response)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("Create calls = %d, want 1", len(client.createRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testProductLicenseID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testProductLicenseID)
	}
	if got := resource.Status.Id; got != testProductLicenseID {
		t.Fatalf("status.id = %q, want %q", got, testProductLicenseID)
	}
	if got := resource.Status.Status; got != string(licensemanagersdk.StatusOk) {
		t.Fatalf("status.sdkStatus = %q, want OK", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
}

func TestProductLicenseCreateOrUpdatePreservesCreateIdentityWhenReadAfterWriteIsAmbiguous(t *testing.T) {
	t.Parallel()

	resource := productLicenseResource()
	client := newAmbiguousProductLicenseCreateClient(t, resource)
	serviceClient := newProductLicenseServiceClientWithOCIClient(client)

	assertProductLicenseAmbiguousCreateReadAfterWrite(t, serviceClient, resource)
	assertProductLicenseCreateRetryUsesTrackedIdentity(t, serviceClient, resource, client)
}

func newAmbiguousProductLicenseCreateClient(
	t *testing.T,
	resource *licensemanagerv1beta1.ProductLicense,
) *fakeProductLicenseOCIClient {
	t.Helper()

	client := &fakeProductLicenseOCIClient{}
	client.listFn = func(_ context.Context, request licensemanagersdk.ListProductLicensesRequest) (licensemanagersdk.ListProductLicensesResponse, error) {
		requireStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
		return licensemanagersdk.ListProductLicensesResponse{}, nil
	}
	client.createFn = func(_ context.Context, request licensemanagersdk.CreateProductLicenseRequest) (licensemanagersdk.CreateProductLicenseResponse, error) {
		assertProductLicenseCreateRequest(t, request, resource)
		return licensemanagersdk.CreateProductLicenseResponse{
			ProductLicense:   productLicenseSDK(testProductLicenseID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
			OpcRequestId:     common.String("opc-create"),
			OpcWorkRequestId: common.String("wr-create"),
		}, nil
	}
	getCalls := 0
	client.getFn = func(_ context.Context, request licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
		requireStringPtr(t, "get productLicenseId", request.ProductLicenseId, testProductLicenseID)
		getCalls++
		if getCalls == 1 {
			notFound := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
			notFound.OpcRequestID = ""
			return licensemanagersdk.GetProductLicenseResponse{}, notFound
		}
		return licensemanagersdk.GetProductLicenseResponse{
			ProductLicense: productLicenseSDK(testProductLicenseID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
		}, nil
	}
	return client
}

func assertProductLicenseAmbiguousCreateReadAfterWrite(
	t *testing.T,
	serviceClient ProductLicenseServiceClient,
	resource *licensemanagerv1beta1.ProductLicense,
) {
	t.Helper()

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("CreateOrUpdate() error = %v, want ambiguous read-after-write 404", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful after ambiguous read-after-write", response)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testProductLicenseID {
		t.Fatalf("status.status.ocid after failed read-after-write = %q, want %q", got, testProductLicenseID)
	}
	if got := resource.Status.Id; got != testProductLicenseID {
		t.Fatalf("status.id after failed read-after-write = %q, want %q", got, testProductLicenseID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId after failed read-after-write = %q, want opc-create", got)
	}
	assertProductLicenseCreateWorkRequestTracked(t, resource)
}

func assertProductLicenseCreateWorkRequestTracked(t *testing.T, resource *licensemanagerv1beta1.ProductLicense) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Phase != shared.OSOKAsyncPhaseCreate || current.WorkRequestID != "wr-create" {
		t.Fatalf("status.status.async.current after failed read-after-write = %#v, want create work request wr-create", current)
	}
}

func assertProductLicenseCreateRetryUsesTrackedIdentity(
	t *testing.T,
	serviceClient ProductLicenseServiceClient,
	resource *licensemanagerv1beta1.ProductLicense,
	client *fakeProductLicenseOCIClient,
) {
	t.Helper()

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() retry error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() retry response = %#v, want active success without requeue", response)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("Create calls after retry = %d, want 1", len(client.createRequests))
	}
	if len(client.listRequests) != 1 {
		t.Fatalf("List calls after retry = %d, want only initial bind lookup", len(client.listRequests))
	}
	if len(client.getRequests) != 2 {
		t.Fatalf("Get calls after retry = %d, want failed create follow-up plus tracked-ID retry", len(client.getRequests))
	}
}

func TestProductLicenseCreateOrUpdateBindsExistingFromSecondListPage(t *testing.T) {
	t.Parallel()

	resource := productLicenseResource()
	client := &fakeProductLicenseOCIClient{}
	client.listFn = func(_ context.Context, request licensemanagersdk.ListProductLicensesRequest) (licensemanagersdk.ListProductLicensesResponse, error) {
		requireStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
		switch page := stringPtrValue(request.Page); page {
		case "":
			otherSpec := resource.Spec
			otherSpec.DisplayName = "other"
			return licensemanagersdk.ListProductLicensesResponse{
				ProductLicenseCollection: licensemanagersdk.ProductLicenseCollection{
					Items: []licensemanagersdk.ProductLicenseSummary{
						productLicenseSummary("ocid1.productlicense.oc1..other", otherSpec, licensemanagersdk.LifeCycleStateActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case "page-2":
			return licensemanagersdk.ListProductLicensesResponse{
				ProductLicenseCollection: licensemanagersdk.ProductLicenseCollection{
					Items: []licensemanagersdk.ProductLicenseSummary{
						productLicenseSummary(testProductLicenseID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected page token %q", page)
			return licensemanagersdk.ListProductLicensesResponse{}, nil
		}
	}
	client.getFn = func(_ context.Context, request licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
		requireStringPtr(t, "get productLicenseId", request.ProductLicenseId, testProductLicenseID)
		return licensemanagersdk.GetProductLicenseResponse{
			ProductLicense: productLicenseSDK(testProductLicenseID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
		}, nil
	}

	response, err := newProductLicenseServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active bind without requeue", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("Create calls = %d, want 0", len(client.createRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("List calls = %d, want 2 pages", len(client.listRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testProductLicenseID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testProductLicenseID)
	}
}

func TestProductLicenseCreateOrUpdateRejectsCreateOnlyDriftDuringBindLookup(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		field  string
		mutate func(*licensemanagersdk.ProductLicenseSummary)
	}{
		{
			name:  "licenseUnit",
			field: "licenseUnit",
			mutate: func(summary *licensemanagersdk.ProductLicenseSummary) {
				summary.LicenseUnit = licensemanagersdk.LicenseUnitProcessors
			},
		},
		{
			name:  "isVendorOracle",
			field: "isVendorOracle",
			mutate: func(summary *licensemanagersdk.ProductLicenseSummary) {
				summary.IsVendorOracle = common.Bool(false)
			},
		},
		{
			name:  "vendorName nil",
			field: "vendorName",
			mutate: func(summary *licensemanagersdk.ProductLicenseSummary) {
				summary.VendorName = nil
			},
		},
		{
			name:  "vendorName empty",
			field: "vendorName",
			mutate: func(summary *licensemanagersdk.ProductLicenseSummary) {
				summary.VendorName = common.String("")
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertProductLicenseBindLookupDriftRejected(t, tc.field, tc.mutate)
		})
	}
}

func TestProductLicenseCreateOrUpdateNoopsWithoutMutableDrift(t *testing.T) {
	t.Parallel()

	resource := productLicenseResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProductLicenseID)
	resource.Status.Id = testProductLicenseID
	resource.Spec.FreeformTags = nil
	client := &fakeProductLicenseOCIClient{}
	client.getFn = func(_ context.Context, request licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
		requireStringPtr(t, "get productLicenseId", request.ProductLicenseId, testProductLicenseID)
		current := productLicenseSDK(testProductLicenseID, resource.Spec, licensemanagersdk.LifeCycleStateActive)
		current.FreeformTags = map[string]string{"remote": "owner"}
		return licensemanagersdk.GetProductLicenseResponse{ProductLicense: current}, nil
	}

	response, err := newProductLicenseServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want no-op success", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0", len(client.updateRequests))
	}
}

func TestProductLicenseCreateOrUpdateSendsMutableUpdate(t *testing.T) {
	t.Parallel()

	resource := productLicenseResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProductLicenseID)
	resource.Status.Id = testProductLicenseID
	resource.Spec.Images = []licensemanagerv1beta1.ProductLicenseImage{{
		ListingId:      "listing-new",
		PackageVersion: "2.0",
	}}
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"ops": {"owner": "platform"}}
	currentSpec := productLicenseMutableDriftSpec(resource.Spec)
	client := productLicenseMutableUpdateClient(t, resource, currentSpec)

	response, err := newProductLicenseServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active update without requeue", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("Update calls = %d, want 1", len(client.updateRequests))
	}
	if got := resource.Status.Images[0].ListingId; got != "listing-new" {
		t.Fatalf("status.images[0].listingId = %q, want listing-new", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
}

func TestProductLicenseCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := productLicenseResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProductLicenseID)
	resource.Status.Id = testProductLicenseID
	currentSpec := resource.Spec
	currentSpec.DisplayName = "license-old"
	client := &fakeProductLicenseOCIClient{}
	client.getFn = func(_ context.Context, request licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
		requireStringPtr(t, "get productLicenseId", request.ProductLicenseId, testProductLicenseID)
		return licensemanagersdk.GetProductLicenseResponse{
			ProductLicense: productLicenseSDK(testProductLicenseID, currentSpec, licensemanagersdk.LifeCycleStateActive),
		}, nil
	}
	client.updateFn = func(context.Context, licensemanagersdk.UpdateProductLicenseRequest) (licensemanagersdk.UpdateProductLicenseResponse, error) {
		t.Fatal("UpdateProductLicense should not be called after create-only drift")
		return licensemanagersdk.UpdateProductLicenseResponse{}, nil
	}

	_, err := newProductLicenseServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "displayName") {
		t.Fatalf("CreateOrUpdate() error = %v, want displayName drift rejection", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("Update calls = %d, want 0", len(client.updateRequests))
	}
}

func TestProductLicenseCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := productLicenseResource()
	client := &fakeProductLicenseOCIClient{}
	client.listFn = func(context.Context, licensemanagersdk.ListProductLicensesRequest) (licensemanagersdk.ListProductLicensesResponse, error) {
		return licensemanagersdk.ListProductLicensesResponse{}, nil
	}
	client.createFn = func(context.Context, licensemanagersdk.CreateProductLicenseRequest) (licensemanagersdk.CreateProductLicenseResponse, error) {
		return licensemanagersdk.CreateProductLicenseResponse{}, errortest.NewServiceError(500, "InternalError", "boom")
	}

	response, err := newProductLicenseServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("CreateOrUpdate() error = %v, want OCI error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestProductLicenseDeleteWaitsUntilConfirmedDeleted(t *testing.T) {
	t.Parallel()

	resource := productLicenseResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProductLicenseID)
	resource.Status.Id = testProductLicenseID
	client := &fakeProductLicenseOCIClient{}
	getCalls := 0
	client.getFn = func(_ context.Context, request licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
		requireStringPtr(t, "get productLicenseId", request.ProductLicenseId, testProductLicenseID)
		getCalls++
		state := licensemanagersdk.LifeCycleStateActive
		if getCalls == 2 {
			state = licensemanagersdk.LifeCycleStateDeleted
		}
		return licensemanagersdk.GetProductLicenseResponse{
			ProductLicense: productLicenseSDK(testProductLicenseID, resource.Spec, state),
		}, nil
	}
	client.deleteFn = func(_ context.Context, request licensemanagersdk.DeleteProductLicenseRequest) (licensemanagersdk.DeleteProductLicenseResponse, error) {
		requireStringPtr(t, "delete productLicenseId", request.ProductLicenseId, testProductLicenseID)
		return licensemanagersdk.DeleteProductLicenseResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := newProductLicenseServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after DELETED readback")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("Delete calls = %d, want 1", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete marker")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
}

func TestProductLicenseDeleteRetainsFinalizerWhenReadbackRemainsActive(t *testing.T) {
	t.Parallel()

	resource := productLicenseResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProductLicenseID)
	resource.Status.Id = testProductLicenseID
	client := &fakeProductLicenseOCIClient{}
	client.getFn = func(_ context.Context, request licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
		requireStringPtr(t, "get productLicenseId", request.ProductLicenseId, testProductLicenseID)
		return licensemanagersdk.GetProductLicenseResponse{
			ProductLicense: productLicenseSDK(testProductLicenseID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
		}, nil
	}
	client.deleteFn = func(context.Context, licensemanagersdk.DeleteProductLicenseRequest) (licensemanagersdk.DeleteProductLicenseResponse, error) {
		return licensemanagersdk.DeleteProductLicenseResponse{
			OpcRequestId:     common.String("opc-delete"),
			OpcWorkRequestId: common.String("wr-delete"),
		}, nil
	}

	deleted, err := newProductLicenseServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false until deletion is confirmed")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set before terminal delete confirmation")
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.WorkRequestID != "wr-delete" {
		t.Fatalf("status.status.async.current = %#v, want delete work request tracking", resource.Status.OsokStatus.Async.Current)
	}
	if got := lastProductLicenseCondition(resource); got != shared.Terminating {
		t.Fatalf("last condition = %q, want Terminating", got)
	}
}

func TestProductLicenseDeleteRejectsAuthShapedPostDeleteRead(t *testing.T) {
	t.Parallel()

	resource := productLicenseResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProductLicenseID)
	resource.Status.Id = testProductLicenseID
	client := &fakeProductLicenseOCIClient{}
	getCalls := 0
	client.getFn = func(_ context.Context, request licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
		requireStringPtr(t, "get productLicenseId", request.ProductLicenseId, testProductLicenseID)
		getCalls++
		if getCalls == 2 {
			return licensemanagersdk.GetProductLicenseResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		}
		return licensemanagersdk.GetProductLicenseResponse{
			ProductLicense: productLicenseSDK(testProductLicenseID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
		}, nil
	}
	client.deleteFn = func(context.Context, licensemanagersdk.DeleteProductLicenseRequest) (licensemanagersdk.DeleteProductLicenseResponse, error) {
		return licensemanagersdk.DeleteProductLicenseResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := newProductLicenseServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous post-delete 404 blocker", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false until deletion is confirmed")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("Delete calls = %d, want 1", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set after ambiguous post-delete read")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id from ambiguous confirmation read", got)
	}
}

func TestProductLicenseDeleteResumesPendingDeleteWithoutSecondDeleteCall(t *testing.T) {
	t.Parallel()

	resource := productLicenseResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProductLicenseID)
	resource.Status.Id = testProductLicenseID
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &metav1.Time{},
	}
	client := &fakeProductLicenseOCIClient{}
	client.getFn = func(context.Context, licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
		return licensemanagersdk.GetProductLicenseResponse{
			ProductLicense: productLicenseSDK(testProductLicenseID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
		}, nil
	}
	client.deleteFn = func(context.Context, licensemanagersdk.DeleteProductLicenseRequest) (licensemanagersdk.DeleteProductLicenseResponse, error) {
		t.Fatal("DeleteProductLicense should not be called while delete work request tracking is pending")
		return licensemanagersdk.DeleteProductLicenseResponse{}, nil
	}

	deleted, err := newProductLicenseServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false until confirmation")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("Delete calls = %d, want 0", len(client.deleteRequests))
	}
}

func TestProductLicenseDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	t.Parallel()

	resource := productLicenseResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testProductLicenseID)
	resource.Status.Id = testProductLicenseID
	client := &fakeProductLicenseOCIClient{}
	client.getFn = func(context.Context, licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
		return licensemanagersdk.GetProductLicenseResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}
	client.deleteFn = func(context.Context, licensemanagersdk.DeleteProductLicenseRequest) (licensemanagersdk.DeleteProductLicenseResponse, error) {
		t.Fatal("DeleteProductLicense should not be called after ambiguous pre-delete read")
		return licensemanagersdk.DeleteProductLicenseResponse{}, nil
	}

	deleted, err := newProductLicenseServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 blocker", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous 404")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("Delete calls = %d, want 0", len(client.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestProductLicenseDeleteWithoutTrackedIdentityDoesNotDeleteBySpec(t *testing.T) {
	t.Parallel()

	resource := productLicenseResource()
	client := &fakeProductLicenseOCIClient{}
	client.getFn = func(context.Context, licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
		t.Fatal("GetProductLicense should not be called before OSOK records an OCI identity")
		return licensemanagersdk.GetProductLicenseResponse{}, nil
	}
	client.deleteFn = func(context.Context, licensemanagersdk.DeleteProductLicenseRequest) (licensemanagersdk.DeleteProductLicenseResponse, error) {
		t.Fatal("DeleteProductLicense should not be called before OSOK records an OCI identity")
		return licensemanagersdk.DeleteProductLicenseResponse{}, nil
	}

	deleted, err := newProductLicenseServiceClientWithOCIClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true for untracked CR cleanup")
	}
}

func newProductLicenseServiceClientWithOCIClient(client productLicenseOCIClient) ProductLicenseServiceClient {
	manager := &ProductLicenseServiceManager{}
	hooks := newProductLicenseRuntimeHooksWithOCIClient(client)
	applyProductLicenseRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultProductLicenseServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*licensemanagerv1beta1.ProductLicense](
			buildProductLicenseGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapProductLicenseGeneratedClient(hooks, delegate)
}

func newProductLicenseRuntimeHooksWithOCIClient(client productLicenseOCIClient) ProductLicenseRuntimeHooks {
	return ProductLicenseRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*licensemanagerv1beta1.ProductLicense]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*licensemanagerv1beta1.ProductLicense]{},
		StatusHooks:     generatedruntime.StatusHooks[*licensemanagerv1beta1.ProductLicense]{},
		ParityHooks:     generatedruntime.ParityHooks[*licensemanagerv1beta1.ProductLicense]{},
		Async:           generatedruntime.AsyncHooks[*licensemanagerv1beta1.ProductLicense]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*licensemanagerv1beta1.ProductLicense]{},
		Create: runtimeOperationHooks[licensemanagersdk.CreateProductLicenseRequest, licensemanagersdk.CreateProductLicenseResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateProductLicenseDetails", RequestName: "CreateProductLicenseDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request licensemanagersdk.CreateProductLicenseRequest) (licensemanagersdk.CreateProductLicenseResponse, error) {
				return client.CreateProductLicense(ctx, request)
			},
		},
		Get: runtimeOperationHooks[licensemanagersdk.GetProductLicenseRequest, licensemanagersdk.GetProductLicenseResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ProductLicenseId", RequestName: "productLicenseId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
				return client.GetProductLicense(ctx, request)
			},
		},
		List: runtimeOperationHooks[licensemanagersdk.ListProductLicensesRequest, licensemanagersdk.ListProductLicensesResponse]{
			Fields: productLicenseListFields(),
			Call: func(ctx context.Context, request licensemanagersdk.ListProductLicensesRequest) (licensemanagersdk.ListProductLicensesResponse, error) {
				return client.ListProductLicenses(ctx, request)
			},
		},
		Update: runtimeOperationHooks[licensemanagersdk.UpdateProductLicenseRequest, licensemanagersdk.UpdateProductLicenseResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ProductLicenseId", RequestName: "productLicenseId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateProductLicenseDetails", RequestName: "UpdateProductLicenseDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request licensemanagersdk.UpdateProductLicenseRequest) (licensemanagersdk.UpdateProductLicenseResponse, error) {
				return client.UpdateProductLicense(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[licensemanagersdk.DeleteProductLicenseRequest, licensemanagersdk.DeleteProductLicenseResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ProductLicenseId", RequestName: "productLicenseId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request licensemanagersdk.DeleteProductLicenseRequest) (licensemanagersdk.DeleteProductLicenseResponse, error) {
				return client.DeleteProductLicense(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ProductLicenseServiceClient) ProductLicenseServiceClient{},
	}
}

func productLicenseResource() *licensemanagerv1beta1.ProductLicense {
	return &licensemanagerv1beta1.ProductLicense{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "product-license",
			Namespace: "default",
			UID:       types.UID("uid-product-license"),
		},
		Spec: licensemanagerv1beta1.ProductLicenseSpec{
			CompartmentId:  testCompartmentID,
			IsVendorOracle: true,
			DisplayName:    "product-license",
			LicenseUnit:    string(licensemanagersdk.LicenseUnitOcpu),
			VendorName:     "Oracle",
			Images: []licensemanagerv1beta1.ProductLicenseImage{{
				ListingId:      "listing-1",
				PackageVersion: "1.0",
			}},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags:  map[string]shared.MapValue{"ops": {"owner": "team"}},
		},
	}
}

func productLicenseSDK(
	id string,
	spec licensemanagerv1beta1.ProductLicenseSpec,
	state licensemanagersdk.LifeCycleStateEnum,
) licensemanagersdk.ProductLicense {
	images := make([]licensemanagersdk.ImageResponse, 0, len(spec.Images))
	for _, image := range spec.Images {
		images = append(images, licensemanagersdk.ImageResponse{
			ListingId:      common.String(image.ListingId),
			PackageVersion: common.String(image.PackageVersion),
		})
	}
	definedTags := map[string]map[string]interface{}{}
	for namespace, values := range spec.DefinedTags {
		definedTags[namespace] = map[string]interface{}{}
		for key, value := range values {
			definedTags[namespace][key] = value
		}
	}
	return licensemanagersdk.ProductLicense{
		Id:                          common.String(id),
		CompartmentId:               common.String(spec.CompartmentId),
		Status:                      licensemanagersdk.StatusOk,
		LicenseUnit:                 licensemanagersdk.LicenseUnitEnum(spec.LicenseUnit),
		IsVendorOracle:              common.Bool(spec.IsVendorOracle),
		DisplayName:                 common.String(spec.DisplayName),
		LifecycleState:              state,
		TotalLicenseUnitsConsumed:   common.Float64(1),
		TotalLicenseRecordCount:     common.Int(2),
		ActiveLicenseRecordCount:    common.Int(1),
		IsOverSubscribed:            common.Bool(false),
		IsUnlimited:                 common.Bool(false),
		VendorName:                  common.String(spec.VendorName),
		Images:                      images,
		FreeformTags:                mapsClone(spec.FreeformTags),
		DefinedTags:                 definedTags,
		TotalActiveLicenseUnitCount: common.Int(10),
	}
}

func productLicenseSummary(
	id string,
	spec licensemanagerv1beta1.ProductLicenseSpec,
	state licensemanagersdk.LifeCycleStateEnum,
) licensemanagersdk.ProductLicenseSummary {
	current := productLicenseSDK(id, spec, state)
	return licensemanagersdk.ProductLicenseSummary{
		Id:                          current.Id,
		CompartmentId:               current.CompartmentId,
		Status:                      current.Status,
		LicenseUnit:                 current.LicenseUnit,
		IsVendorOracle:              current.IsVendorOracle,
		DisplayName:                 current.DisplayName,
		VendorName:                  current.VendorName,
		LifecycleState:              current.LifecycleState,
		Images:                      current.Images,
		FreeformTags:                current.FreeformTags,
		DefinedTags:                 current.DefinedTags,
		TotalActiveLicenseUnitCount: current.TotalActiveLicenseUnitCount,
	}
}

func assertProductLicenseBindLookupDriftRejected(
	t *testing.T,
	field string,
	mutate func(*licensemanagersdk.ProductLicenseSummary),
) {
	t.Helper()

	resource := productLicenseResource()
	client := &fakeProductLicenseOCIClient{}
	client.listFn = func(context.Context, licensemanagersdk.ListProductLicensesRequest) (licensemanagersdk.ListProductLicensesResponse, error) {
		summary := productLicenseSummary(testProductLicenseID, resource.Spec, licensemanagersdk.LifeCycleStateActive)
		mutate(&summary)
		return licensemanagersdk.ListProductLicensesResponse{
			ProductLicenseCollection: licensemanagersdk.ProductLicenseCollection{
				Items: []licensemanagersdk.ProductLicenseSummary{summary},
			},
		}, nil
	}
	client.getFn = func(context.Context, licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
		t.Fatal("GetProductLicense should not be called after bind lookup create-only drift")
		return licensemanagersdk.GetProductLicenseResponse{}, nil
	}
	client.createFn = func(context.Context, licensemanagersdk.CreateProductLicenseRequest) (licensemanagersdk.CreateProductLicenseResponse, error) {
		t.Fatal("CreateProductLicense should not be called after bind lookup create-only drift")
		return licensemanagersdk.CreateProductLicenseResponse{}, nil
	}

	_, err := newProductLicenseServiceClientWithOCIClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), field) {
		t.Fatalf("CreateOrUpdate() error = %v, want %s drift rejection", err, field)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("Create calls = %d, want 0", len(client.createRequests))
	}
}

func productLicenseMutableDriftSpec(
	spec licensemanagerv1beta1.ProductLicenseSpec,
) licensemanagerv1beta1.ProductLicenseSpec {
	spec.Images = []licensemanagerv1beta1.ProductLicenseImage{{
		ListingId:      "listing-old",
		PackageVersion: "1.0",
	}}
	spec.FreeformTags = map[string]string{"env": "dev"}
	spec.DefinedTags = map[string]shared.MapValue{"ops": {"owner": "old"}}
	return spec
}

func productLicenseMutableUpdateClient(
	t *testing.T,
	resource *licensemanagerv1beta1.ProductLicense,
	currentSpec licensemanagerv1beta1.ProductLicenseSpec,
) *fakeProductLicenseOCIClient {
	t.Helper()

	client := &fakeProductLicenseOCIClient{}
	getCalls := 0
	client.getFn = func(_ context.Context, request licensemanagersdk.GetProductLicenseRequest) (licensemanagersdk.GetProductLicenseResponse, error) {
		requireStringPtr(t, "get productLicenseId", request.ProductLicenseId, testProductLicenseID)
		getCalls++
		if getCalls == 2 {
			return licensemanagersdk.GetProductLicenseResponse{
				ProductLicense: productLicenseSDK(testProductLicenseID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
			}, nil
		}
		return licensemanagersdk.GetProductLicenseResponse{
			ProductLicense: productLicenseSDK(testProductLicenseID, currentSpec, licensemanagersdk.LifeCycleStateActive),
		}, nil
	}
	client.updateFn = func(_ context.Context, request licensemanagersdk.UpdateProductLicenseRequest) (licensemanagersdk.UpdateProductLicenseResponse, error) {
		assertProductLicenseUpdateRequest(t, request)
		return licensemanagersdk.UpdateProductLicenseResponse{
			ProductLicense: productLicenseSDK(testProductLicenseID, resource.Spec, licensemanagersdk.LifeCycleStateActive),
			OpcRequestId:   common.String("opc-update"),
		}, nil
	}
	return client
}

func assertProductLicenseUpdateRequest(t *testing.T, request licensemanagersdk.UpdateProductLicenseRequest) {
	t.Helper()

	requireStringPtr(t, "update productLicenseId", request.ProductLicenseId, testProductLicenseID)
	if len(request.Images) != 1 {
		t.Fatalf("update images = %d, want 1", len(request.Images))
	}
	requireStringPtr(t, "update images[0].listingId", request.Images[0].ListingId, "listing-new")
	requireStringPtr(t, "update images[0].packageVersion", request.Images[0].PackageVersion, "2.0")
	if got := request.FreeformTags["env"]; got != "prod" {
		t.Fatalf("update freeformTags[env] = %q, want prod", got)
	}
	if got := request.DefinedTags["ops"]["owner"]; got != "platform" {
		t.Fatalf("update definedTags[ops][owner] = %v, want platform", got)
	}
}

func assertProductLicenseCreateRequest(
	t *testing.T,
	request licensemanagersdk.CreateProductLicenseRequest,
	resource *licensemanagerv1beta1.ProductLicense,
) {
	t.Helper()
	requireStringPtr(t, "create compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "create displayName", request.DisplayName, resource.Spec.DisplayName)
	if request.IsVendorOracle == nil || *request.IsVendorOracle != resource.Spec.IsVendorOracle {
		t.Fatalf("create isVendorOracle = %v, want %t", request.IsVendorOracle, resource.Spec.IsVendorOracle)
	}
	if got := string(request.LicenseUnit); got != resource.Spec.LicenseUnit {
		t.Fatalf("create licenseUnit = %q, want %q", got, resource.Spec.LicenseUnit)
	}
	requireStringPtr(t, "create vendorName", request.VendorName, resource.Spec.VendorName)
	if len(request.Images) != 1 {
		t.Fatalf("create images = %d, want 1", len(request.Images))
	}
	requireStringPtr(t, "create images[0].listingId", request.Images[0].ListingId, resource.Spec.Images[0].ListingId)
	requireStringPtr(t, "create images[0].packageVersion", request.Images[0].PackageVersion, resource.Spec.Images[0].PackageVersion)
	if got := request.FreeformTags["env"]; got != "dev" {
		t.Fatalf("create freeformTags[env] = %q, want dev", got)
	}
	if request.OpcRetryToken == nil || *request.OpcRetryToken == "" {
		t.Fatal("create opcRetryToken is empty")
	}
}

func requireStringPtr(t *testing.T, label string, value *string, want string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if got := *value; got != want {
		t.Fatalf("%s = %q, want %q", label, got, want)
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func lastProductLicenseCondition(resource *licensemanagerv1beta1.ProductLicense) shared.OSOKConditionType {
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		return ""
	}
	return conditions[len(conditions)-1].Type
}

func assertProductLicenseStringSliceContains(t *testing.T, label string, got []string, wants ...string) {
	t.Helper()
	values := make(map[string]bool, len(got))
	for _, value := range got {
		values[value] = true
	}
	for _, want := range wants {
		if !values[want] {
			t.Fatalf("%s = %#v, want to contain %q", label, got, want)
		}
	}
}

func mapsClone(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
