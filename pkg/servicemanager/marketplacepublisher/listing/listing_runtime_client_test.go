/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package listing

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testListingID     = "ocid1.listing.oc1..example"
	testListingOther  = "ocid1.listing.oc1..other"
	testListingCompID = "ocid1.compartment.oc1..example"
)

type fakeListingOCIClient struct {
	createFn func(context.Context, marketplacepublishersdk.CreateListingRequest) (marketplacepublishersdk.CreateListingResponse, error)
	getFn    func(context.Context, marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error)
	listFn   func(context.Context, marketplacepublishersdk.ListListingsRequest) (marketplacepublishersdk.ListListingsResponse, error)
	updateFn func(context.Context, marketplacepublishersdk.UpdateListingRequest) (marketplacepublishersdk.UpdateListingResponse, error)
	deleteFn func(context.Context, marketplacepublishersdk.DeleteListingRequest) (marketplacepublishersdk.DeleteListingResponse, error)
}

func (f *fakeListingOCIClient) CreateListing(
	ctx context.Context,
	request marketplacepublishersdk.CreateListingRequest,
) (marketplacepublishersdk.CreateListingResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return marketplacepublishersdk.CreateListingResponse{}, nil
}

func (f *fakeListingOCIClient) GetListing(
	ctx context.Context,
	request marketplacepublishersdk.GetListingRequest,
) (marketplacepublishersdk.GetListingResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return marketplacepublishersdk.GetListingResponse{}, nil
}

func (f *fakeListingOCIClient) ListListings(
	ctx context.Context,
	request marketplacepublishersdk.ListListingsRequest,
) (marketplacepublishersdk.ListListingsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return marketplacepublishersdk.ListListingsResponse{}, nil
}

func (f *fakeListingOCIClient) UpdateListing(
	ctx context.Context,
	request marketplacepublishersdk.UpdateListingRequest,
) (marketplacepublishersdk.UpdateListingResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return marketplacepublishersdk.UpdateListingResponse{}, nil
}

func (f *fakeListingOCIClient) DeleteListing(
	ctx context.Context,
	request marketplacepublishersdk.DeleteListingRequest,
) (marketplacepublishersdk.DeleteListingResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return marketplacepublishersdk.DeleteListingResponse{}, nil
}

func testListingClient(fake *fakeListingOCIClient) ListingServiceClient {
	return newListingServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeListingResource() *marketplacepublisherv1beta1.Listing {
	return &marketplacepublisherv1beta1.Listing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "listing-alpha",
			Namespace: "default",
			UID:       types.UID("listing-uid"),
		},
		Spec: marketplacepublisherv1beta1.ListingSpec{
			CompartmentId: testListingCompID,
			Name:          "listing-alpha",
			ListingType:   string(marketplacepublishersdk.ListingTypeOciApplication),
			PackageType:   string(marketplacepublishersdk.PackageTypeStack),
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKListing(
	id string,
	compartmentID string,
	name string,
	listingType marketplacepublishersdk.ListingTypeEnum,
	packageType marketplacepublishersdk.PackageTypeEnum,
	state marketplacepublishersdk.ListingLifecycleStateEnum,
) marketplacepublishersdk.Listing {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 30, 0, 0, time.UTC)}
	return marketplacepublishersdk.Listing{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		PublisherId:    common.String("ocid1.publisher.oc1..example"),
		ListingType:    listingType,
		Name:           common.String(name),
		PackageType:    packageType,
		TimeCreated:    &created,
		TimeUpdated:    &updated,
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:     map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func makeSDKListingSummary(
	id string,
	compartmentID string,
	name string,
	listingType marketplacepublishersdk.ListingTypeEnum,
	packageType marketplacepublishersdk.PackageTypeEnum,
	state marketplacepublishersdk.ListingLifecycleStateEnum,
) marketplacepublishersdk.ListingSummary {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 30, 0, 0, time.UTC)}
	return marketplacepublishersdk.ListingSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		ListingType:    listingType,
		Name:           common.String(name),
		LifecycleState: state,
		PackageType:    packageType,
		TimeCreated:    &created,
		TimeUpdated:    &updated,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:     map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func TestListingRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := reviewedListingRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedListingRuntimeSemantics() = nil")
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	requireStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "name", "listingType", "packageType"})
	requireStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"freeformTags", "definedTags"})
	requireStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "name", "listingType", "packageType"})
}

func TestListingServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeListingResource()
	var listRequest marketplacepublishersdk.ListListingsRequest
	var createRequest marketplacepublishersdk.CreateListingRequest
	var getRequest marketplacepublishersdk.GetListingRequest
	listCalls := 0
	createCalls := 0
	getCalls := 0

	client := testListingClient(&fakeListingOCIClient{
		listFn: func(_ context.Context, request marketplacepublishersdk.ListListingsRequest) (marketplacepublishersdk.ListListingsResponse, error) {
			listCalls++
			listRequest = request
			return marketplacepublishersdk.ListListingsResponse{
				ListingCollection: marketplacepublishersdk.ListingCollection{},
				OpcRequestId:      common.String("opc-list"),
			}, nil
		},
		createFn: func(_ context.Context, request marketplacepublishersdk.CreateListingRequest) (marketplacepublishersdk.CreateListingResponse, error) {
			createCalls++
			createRequest = request
			return marketplacepublishersdk.CreateListingResponse{
				Listing: makeSDKListing(
					testListingID,
					testListingCompID,
					resource.Spec.Name,
					marketplacepublishersdk.ListingTypeOciApplication,
					marketplacepublishersdk.PackageTypeStack,
					marketplacepublishersdk.ListingLifecycleStateActive,
				),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error) {
			getCalls++
			getRequest = request
			return marketplacepublishersdk.GetListingResponse{
				Listing: makeSDKListing(
					testListingID,
					testListingCompID,
					resource.Spec.Name,
					marketplacepublishersdk.ListingTypeOciApplication,
					marketplacepublishersdk.PackageTypeStack,
					marketplacepublishersdk.ListingLifecycleStateActive,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if listCalls != 1 || createCalls != 1 || getCalls != 1 {
		t.Fatalf("call counts list/create/get = %d/%d/%d, want 1/1/1", listCalls, createCalls, getCalls)
	}
	assertListingCreateRequest(t, resource, listRequest, createRequest)
	requireStringPtr(t, "create retry token", createRequest.OpcRetryToken, string(resource.UID))
	requireStringPtr(t, "get listingId", getRequest.ListingId, testListingID)
	assertCreatedListingStatus(t, resource)
}

func assertListingCreateRequest(
	t *testing.T,
	resource *marketplacepublisherv1beta1.Listing,
	listRequest marketplacepublishersdk.ListListingsRequest,
	createRequest marketplacepublishersdk.CreateListingRequest,
) {
	t.Helper()
	requireStringPtr(t, "list compartmentId", listRequest.CompartmentId, testListingCompID)
	requireStringPtr(t, "list name", listRequest.Name, resource.Spec.Name)
	if listRequest.ListingType != marketplacepublishersdk.ListListingsListingTypeOciApplication {
		t.Fatalf("list listingType = %q, want OCI_APPLICATION", listRequest.ListingType)
	}
	if listRequest.LifecycleState != "" {
		t.Fatalf("list lifecycleState = %q, want empty so active resources can bind", listRequest.LifecycleState)
	}
	requireStringPtr(t, "create compartmentId", createRequest.CompartmentId, testListingCompID)
	requireStringPtr(t, "create name", createRequest.Name, resource.Spec.Name)
	if createRequest.ListingType != marketplacepublishersdk.ListingTypeOciApplication {
		t.Fatalf("create listingType = %q, want OCI_APPLICATION", createRequest.ListingType)
	}
	if createRequest.PackageType != marketplacepublishersdk.PackageTypeStack {
		t.Fatalf("create packageType = %q, want STACK", createRequest.PackageType)
	}
	if !reflect.DeepEqual(createRequest.FreeformTags, map[string]string{"env": "dev"}) {
		t.Fatalf("create freeformTags = %#v, want env=dev", createRequest.FreeformTags)
	}
	if got := createRequest.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("create definedTags Operations.CostCenter = %#v, want 42", got)
	}
}

func assertCreatedListingStatus(t *testing.T, resource *marketplacepublisherv1beta1.Listing) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != testListingID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testListingID)
	}
	if resource.Status.Id != testListingID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testListingID)
	}
	if resource.Status.PublisherId != "ocid1.publisher.oc1..example" {
		t.Fatalf("status.publisherId = %q, want publisher id", resource.Status.PublisherId)
	}
	if resource.Status.LifecycleState != string(marketplacepublishersdk.ListingLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", resource.Status.OsokStatus.OpcRequestID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestListingServiceClientBindsFromPaginatedList(t *testing.T) {
	t.Parallel()

	resource := makeListingResource()
	var pages []string
	listCalls := 0
	getCalls := 0

	client := testListingClient(&fakeListingOCIClient{
		listFn: func(_ context.Context, request marketplacepublishersdk.ListListingsRequest) (marketplacepublishersdk.ListListingsResponse, error) {
			listCalls++
			pages = append(pages, stringValue(request.Page))
			if listCalls == 1 {
				return marketplacepublishersdk.ListListingsResponse{
					ListingCollection: marketplacepublishersdk.ListingCollection{
						Items: []marketplacepublishersdk.ListingSummary{
							makeSDKListingSummary(
								testListingOther,
								testListingCompID,
								"other-listing",
								marketplacepublishersdk.ListingTypeOciApplication,
								marketplacepublishersdk.PackageTypeStack,
								marketplacepublishersdk.ListingLifecycleStateActive,
							),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return marketplacepublishersdk.ListListingsResponse{
				ListingCollection: marketplacepublishersdk.ListingCollection{
					Items: []marketplacepublishersdk.ListingSummary{
						makeSDKListingSummary(
							testListingID,
							testListingCompID,
							resource.Spec.Name,
							marketplacepublishersdk.ListingTypeOciApplication,
							marketplacepublishersdk.PackageTypeStack,
							marketplacepublishersdk.ListingLifecycleStateActive,
						),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error) {
			getCalls++
			requireStringPtr(t, "get listingId", request.ListingId, testListingID)
			return marketplacepublishersdk.GetListingResponse{
				Listing: makeSDKListing(
					testListingID,
					testListingCompID,
					resource.Spec.Name,
					marketplacepublishersdk.ListingTypeOciApplication,
					marketplacepublishersdk.PackageTypeStack,
					marketplacepublishersdk.ListingLifecycleStateActive,
				),
			}, nil
		},
		createFn: func(context.Context, marketplacepublishersdk.CreateListingRequest) (marketplacepublishersdk.CreateListingResponse, error) {
			t.Fatal("CreateListing() called for existing listing")
			return marketplacepublishersdk.CreateListingResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if listCalls != 2 || getCalls != 1 {
		t.Fatalf("call counts list/get = %d/%d, want 2/1", listCalls, getCalls)
	}
	if want := []string{"", "page-2"}; !reflect.DeepEqual(pages, want) {
		t.Fatalf("list pages = %#v, want %#v", pages, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testListingID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testListingID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestListingServiceClientNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := makeListingResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingID)
	getCalls := 0
	updateCalls := 0

	client := testListingClient(&fakeListingOCIClient{
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error) {
			getCalls++
			requireStringPtr(t, "get listingId", request.ListingId, testListingID)
			return marketplacepublishersdk.GetListingResponse{
				Listing: makeSDKListing(
					testListingID,
					testListingCompID,
					resource.Spec.Name,
					marketplacepublishersdk.ListingTypeOciApplication,
					marketplacepublishersdk.PackageTypeStack,
					marketplacepublishersdk.ListingLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(context.Context, marketplacepublishersdk.UpdateListingRequest) (marketplacepublishersdk.UpdateListingResponse, error) {
			updateCalls++
			return marketplacepublishersdk.UpdateListingResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if getCalls != 1 || updateCalls != 0 {
		t.Fatalf("call counts get/update = %d/%d, want 1/0", getCalls, updateCalls)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestListingServiceClientUpdatesMutableTags(t *testing.T) {
	t.Parallel()

	resource := makeListingResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingID)
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}
	getCalls := 0
	updateCalls := 0
	var updateRequest marketplacepublishersdk.UpdateListingRequest

	client := testListingClient(&fakeListingOCIClient{
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error) {
			getCalls++
			requireStringPtr(t, "get listingId", request.ListingId, testListingID)
			return marketplacepublishersdk.GetListingResponse{
				Listing: mutableUpdateListingReadback(resource, getCalls),
			}, nil
		},
		updateFn: func(_ context.Context, request marketplacepublishersdk.UpdateListingRequest) (marketplacepublishersdk.UpdateListingResponse, error) {
			updateCalls++
			updateRequest = request
			updated := makeSDKListing(
				testListingID,
				testListingCompID,
				resource.Spec.Name,
				marketplacepublishersdk.ListingTypeOciApplication,
				marketplacepublishersdk.PackageTypeStack,
				marketplacepublishersdk.ListingLifecycleStateActive,
			)
			updated.FreeformTags = map[string]string{"env": "prod"}
			updated.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
			return marketplacepublishersdk.UpdateListingResponse{
				Listing:      updated,
				OpcRequestId: common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	if getCalls != 2 || updateCalls != 1 {
		t.Fatalf("call counts get/update = %d/%d, want 2/1", getCalls, updateCalls)
	}
	requireStringPtr(t, "update listingId", updateRequest.ListingId, testListingID)
	if !reflect.DeepEqual(updateRequest.FreeformTags, map[string]string{"env": "prod"}) {
		t.Fatalf("update freeformTags = %#v, want env=prod", updateRequest.FreeformTags)
	}
	if got := updateRequest.DefinedTags["Operations"]["CostCenter"]; got != "84" {
		t.Fatalf("update definedTags Operations.CostCenter = %#v, want 84", got)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", resource.Status.OsokStatus.OpcRequestID)
	}
	requireLastCondition(t, resource, shared.Active)
}

func mutableUpdateListingReadback(
	resource *marketplacepublisherv1beta1.Listing,
	getCalls int,
) marketplacepublishersdk.Listing {
	current := makeSDKListing(
		testListingID,
		testListingCompID,
		resource.Spec.Name,
		marketplacepublishersdk.ListingTypeOciApplication,
		marketplacepublishersdk.PackageTypeStack,
		marketplacepublishersdk.ListingLifecycleStateActive,
	)
	if getCalls <= 1 {
		return current
	}

	current.FreeformTags = map[string]string{"env": "prod"}
	current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "84"}}
	return current
}

func TestListingServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeListingResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingID)
	updateCalls := 0

	client := testListingClient(&fakeListingOCIClient{
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error) {
			requireStringPtr(t, "get listingId", request.ListingId, testListingID)
			return marketplacepublishersdk.GetListingResponse{
				Listing: makeSDKListing(
					testListingID,
					testListingCompID,
					resource.Spec.Name,
					marketplacepublishersdk.ListingTypeOciApplication,
					marketplacepublishersdk.PackageTypeContainerImage,
					marketplacepublishersdk.ListingLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(context.Context, marketplacepublishersdk.UpdateListingRequest) (marketplacepublishersdk.UpdateListingResponse, error) {
			updateCalls++
			return marketplacepublishersdk.UpdateListingResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateListing() calls = %d, want 0", updateCalls)
	}
	if !strings.Contains(err.Error(), "packageType") {
		t.Fatalf("CreateOrUpdate() error = %v, want packageType drift", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestListingServiceClientRetainsFinalizerUntilDeleteConfirmed(t *testing.T) {
	t.Parallel()

	resource := makeListingResourceWithDeleteState()
	getCalls := 0
	deleteCalls := 0

	client := testListingClient(&fakeListingOCIClient{
		getFn:    listingPendingDeleteGetStub(t, resource, &getCalls),
		deleteFn: listingPendingDeleteDeleteStub(t, &deleteCalls),
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	assertListingDeleteStillPending(t, resource, deleted, getCalls, deleteCalls)
}

func makeListingResourceWithDeleteState() *marketplacepublisherv1beta1.Listing {
	resource := makeListingResource()
	resource.Status.Id = testListingID
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingID)
	resource.Finalizers = []string{core.OSOKFinalizerName}
	return resource
}

func listingPendingDeleteGetStub(
	t *testing.T,
	resource *marketplacepublisherv1beta1.Listing,
	getCalls *int,
) func(context.Context, marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error) {
	t.Helper()
	return func(_ context.Context, request marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error) {
		(*getCalls)++
		requireStringPtr(t, "get listingId", request.ListingId, testListingID)
		state := marketplacepublishersdk.ListingLifecycleStateActive
		if *getCalls == 3 {
			state = marketplacepublishersdk.ListingLifecycleStateDeleting
		}
		return marketplacepublishersdk.GetListingResponse{
			Listing: makeSDKListing(
				testListingID,
				testListingCompID,
				resource.Spec.Name,
				marketplacepublishersdk.ListingTypeOciApplication,
				marketplacepublishersdk.PackageTypeStack,
				state,
			),
		}, nil
	}
}

func listingPendingDeleteDeleteStub(
	t *testing.T,
	deleteCalls *int,
) func(context.Context, marketplacepublishersdk.DeleteListingRequest) (marketplacepublishersdk.DeleteListingResponse, error) {
	t.Helper()
	return func(_ context.Context, request marketplacepublishersdk.DeleteListingRequest) (marketplacepublishersdk.DeleteListingResponse, error) {
		(*deleteCalls)++
		requireStringPtr(t, "delete listingId", request.ListingId, testListingID)
		return marketplacepublishersdk.DeleteListingResponse{OpcRequestId: common.String("opc-delete")}, nil
	}
}

func assertListingDeleteStillPending(
	t *testing.T,
	resource *marketplacepublisherv1beta1.Listing,
	deleted bool,
	getCalls int,
	deleteCalls int,
) {
	t.Helper()
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI reports DELETING")
	}
	if getCalls != 3 || deleteCalls != 1 {
		t.Fatalf("call counts get/delete = %d/%d, want 3/1", getCalls, deleteCalls)
	}
	if !core.HasFinalizer(resource, core.OSOKFinalizerName) {
		t.Fatal("finalizer removed while OCI reports DELETING, want retained")
	}
	if resource.Status.LifecycleState != string(marketplacepublishersdk.ListingLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", resource.Status.OsokStatus.OpcRequestID)
	}
	requireLastCondition(t, resource, shared.Terminating)
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want delete tracker", current)
	}
}

func TestListingServiceClientMarksDeletedAfterUnambiguousNotFound(t *testing.T) {
	t.Parallel()

	resource := makeListingResource()
	resource.Status.Id = testListingID
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingID)
	deleteCalls := 0

	client := testListingClient(&fakeListingOCIClient{
		getFn: func(context.Context, marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error) {
			return marketplacepublishersdk.GetListingResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "listing is gone")
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteListingRequest) (marketplacepublishersdk.DeleteListingResponse, error) {
			deleteCalls++
			return marketplacepublishersdk.DeleteListingResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous NotFound")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteListing() calls = %d, want 0", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestListingServiceClientTreatsAuthShapedDeleteNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := makeListingResource()
	resource.Status.Id = testListingID
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingID)
	deleteCalls := 0

	client := testListingClient(&fakeListingOCIClient{
		getFn: func(context.Context, marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error) {
			return marketplacepublishersdk.GetListingResponse{
				Listing: makeSDKListing(
					testListingID,
					testListingCompID,
					resource.Spec.Name,
					marketplacepublishersdk.ListingTypeOciApplication,
					marketplacepublishersdk.PackageTypeStack,
					marketplacepublishersdk.ListingLifecycleStateActive,
				),
			}, nil
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteListingRequest) (marketplacepublishersdk.DeleteListingResponse, error) {
			deleteCalls++
			return marketplacepublishersdk.DeleteListingResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"authorization or existence is ambiguous",
			)
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteListing() calls = %d, want 1", deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestListingServiceClientRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeListingResource()
	resource.Status.Id = testListingID
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingID)
	resource.Finalizers = []string{core.OSOKFinalizerName}
	deleteCalls := 0
	serviceErr := errortest.NewServiceError(
		404,
		errorutil.NotAuthorizedOrNotFound,
		"authorization or existence is ambiguous",
	)
	serviceErr.OpcRequestID = "opc-confirm-pre"

	client := testListingClient(&fakeListingOCIClient{
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error) {
			requireStringPtr(t, "get listingId", request.ListingId, testListingID)
			return marketplacepublishersdk.GetListingResponse{}, serviceErr
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteListingRequest) (marketplacepublishersdk.DeleteListingResponse, error) {
			deleteCalls++
			return marketplacepublishersdk.DeleteListingResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want pre-delete ambiguous confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete confirm read")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteListing() calls = %d, want 0 after auth-shaped pre-delete confirm read", deleteCalls)
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous pre-delete confirm-read classification", err)
	}
	if !core.HasFinalizer(resource, core.OSOKFinalizerName) {
		t.Fatal("finalizer removed after auth-shaped pre-delete confirm read, want retained")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-confirm-pre" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-confirm-pre", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestListingServiceClientTreatsAuthShapedConfirmReadConservatively(t *testing.T) {
	t.Parallel()

	resource := makeListingResource()
	resource.Status.Id = testListingID
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingID)
	getCalls := 0

	client := testListingClient(&fakeListingOCIClient{
		getFn: func(context.Context, marketplacepublishersdk.GetListingRequest) (marketplacepublishersdk.GetListingResponse, error) {
			getCalls++
			if getCalls < 3 {
				return marketplacepublishersdk.GetListingResponse{
					Listing: makeSDKListing(
						testListingID,
						testListingCompID,
						resource.Spec.Name,
						marketplacepublishersdk.ListingTypeOciApplication,
						marketplacepublishersdk.PackageTypeStack,
						marketplacepublishersdk.ListingLifecycleStateActive,
					),
				}, nil
			}
			return marketplacepublishersdk.GetListingResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"authorization or existence is ambiguous",
			)
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteListingRequest) (marketplacepublishersdk.DeleteListingResponse, error) {
			return marketplacepublishersdk.DeleteListingResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped confirm 404")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestListingServiceClientRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeListingResource()
	client := testListingClient(&fakeListingOCIClient{
		listFn: func(context.Context, marketplacepublishersdk.ListListingsRequest) (marketplacepublishersdk.ListListingsResponse, error) {
			return marketplacepublishersdk.ListListingsResponse{}, nil
		},
		createFn: func(context.Context, marketplacepublishersdk.CreateListingRequest) (marketplacepublishersdk.CreateListingResponse, error) {
			return marketplacepublishersdk.CreateListingResponse{}, errortest.NewServiceError(500, "InternalError", "service unavailable")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
	requireLastCondition(t, resource, shared.Failed)
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

func requireLastCondition(
	t *testing.T,
	resource *marketplacepublisherv1beta1.Listing,
	want shared.OSOKConditionType,
) {
	t.Helper()
	if resource == nil {
		t.Fatal("resource = nil")
	}
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions = empty, want %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func requireStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}
