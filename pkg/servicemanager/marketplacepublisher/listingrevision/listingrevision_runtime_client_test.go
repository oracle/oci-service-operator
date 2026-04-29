/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package listingrevision

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testListingRevisionID      = "ocid1.listingrevision.oc1..example"
	testListingRevisionOtherID = "ocid1.listingrevision.oc1..other"
	testListingID              = "ocid1.marketplacelisting.oc1..example"
	testCompartmentID          = "ocid1.compartment.oc1..example"
)

type fakeListingRevisionOCIClient struct {
	createFn func(context.Context, marketplacepublishersdk.CreateListingRevisionRequest) (marketplacepublishersdk.CreateListingRevisionResponse, error)
	getFn    func(context.Context, marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error)
	listFn   func(context.Context, marketplacepublishersdk.ListListingRevisionsRequest) (marketplacepublishersdk.ListListingRevisionsResponse, error)
	updateFn func(context.Context, marketplacepublishersdk.UpdateListingRevisionRequest) (marketplacepublishersdk.UpdateListingRevisionResponse, error)
	deleteFn func(context.Context, marketplacepublishersdk.DeleteListingRevisionRequest) (marketplacepublishersdk.DeleteListingRevisionResponse, error)
}

func (f *fakeListingRevisionOCIClient) CreateListingRevision(
	ctx context.Context,
	request marketplacepublishersdk.CreateListingRevisionRequest,
) (marketplacepublishersdk.CreateListingRevisionResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return marketplacepublishersdk.CreateListingRevisionResponse{}, nil
}

func (f *fakeListingRevisionOCIClient) GetListingRevision(
	ctx context.Context,
	request marketplacepublishersdk.GetListingRevisionRequest,
) (marketplacepublishersdk.GetListingRevisionResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return marketplacepublishersdk.GetListingRevisionResponse{}, nil
}

func (f *fakeListingRevisionOCIClient) ListListingRevisions(
	ctx context.Context,
	request marketplacepublishersdk.ListListingRevisionsRequest,
) (marketplacepublishersdk.ListListingRevisionsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return marketplacepublishersdk.ListListingRevisionsResponse{}, nil
}

func (f *fakeListingRevisionOCIClient) UpdateListingRevision(
	ctx context.Context,
	request marketplacepublishersdk.UpdateListingRevisionRequest,
) (marketplacepublishersdk.UpdateListingRevisionResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return marketplacepublishersdk.UpdateListingRevisionResponse{}, nil
}

func (f *fakeListingRevisionOCIClient) DeleteListingRevision(
	ctx context.Context,
	request marketplacepublishersdk.DeleteListingRevisionRequest,
) (marketplacepublishersdk.DeleteListingRevisionResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return marketplacepublishersdk.DeleteListingRevisionResponse{}, nil
}

func testListingRevisionClient(fake *fakeListingRevisionOCIClient) ListingRevisionServiceClient {
	return newListingRevisionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeListingRevisionResource() *marketplacepublisherv1beta1.ListingRevision {
	return &marketplacepublisherv1beta1.ListingRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "listing-revision",
			Namespace: "default",
			UID:       types.UID("listing-revision-uid"),
		},
		Spec: marketplacepublisherv1beta1.ListingRevisionSpec{
			ListingId:    testListingID,
			ListingType:  string(marketplacepublishersdk.ListingTypeService),
			DisplayName:  "Partner service",
			Headline:     "Partner service headline",
			Tagline:      "Initial tagline",
			ProductCodes: []string{"COMPUTE"},
			Industries:   []string{"Technology"},
			FreeformTags: map[string]string{"env": "dev"},
		},
	}
}

func makeServiceListingRevisionWithInapplicableFields() *marketplacepublisherv1beta1.ListingRevision {
	resource := makeListingRevisionResource()
	resource.Spec.Products = []marketplacepublisherv1beta1.ListingRevisionProduct{
		{
			Code:       "DATABASE",
			Categories: []string{"APP"},
		},
	}
	resource.Spec.VersionDetails = marketplacepublisherv1beta1.ListingRevisionVersionDetails{
		Number:      "1.0",
		Description: "service-inapplicable version",
		ReleaseDate: "2026-04-29",
	}
	resource.Spec.PricingType = string(marketplacepublishersdk.OciListingRevisionPricingTypePaygo)
	resource.Spec.PricingPlans = []marketplacepublisherv1beta1.ListingRevisionPricingPlan{
		{
			PlanType: string(marketplacepublishersdk.PricingPlanPlanTypeFixed),
			Rates: []marketplacepublisherv1beta1.ListingRevisionPricingPlanRate{
				{
					BillingCurrency: "USD",
					Rate:            1,
				},
			},
		},
	}
	resource.Spec.DemoUrl = "https://example.com/demo"
	resource.Spec.DownloadInfo = marketplacepublisherv1beta1.ListingRevisionDownloadInfo{
		Description: "lead-only download",
		Url:         "https://example.com/download",
	}
	return resource
}

func makeOciListingRevisionWithInapplicableFields() *marketplacepublisherv1beta1.ListingRevision {
	resource := makeListingRevisionResource()
	resource.Spec.ListingType = string(marketplacepublishersdk.ListingTypeOciApplication)
	resource.Spec.DisplayName = "OCI listing"
	resource.Spec.Headline = "OCI headline"
	resource.Spec.FreeformTags = nil
	resource.Spec.ProductCodes = []string{"SERVICE_ONLY"}
	resource.Spec.Industries = []string{"Service only"}
	resource.Spec.ContactUs = "service-only contact"
	resource.Spec.Products = []marketplacepublisherv1beta1.ListingRevisionProduct{
		{
			Code:       "DATABASE",
			Categories: []string{"APP"},
		},
	}
	resource.Spec.PricingType = string(marketplacepublishersdk.OciListingRevisionPricingTypePaygo)
	return resource
}

func makeSDKServiceListingRevision(
	id string,
	listingID string,
	displayName string,
	tagline string,
	state marketplacepublishersdk.ListingRevisionLifecycleStateEnum,
) marketplacepublishersdk.ServiceListingRevision {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return marketplacepublishersdk.ServiceListingRevision{
		Id:             common.String(id),
		ListingId:      common.String(listingID),
		CompartmentId:  common.String(testCompartmentID),
		DisplayName:    common.String(displayName),
		Headline:       common.String("Partner service headline"),
		TimeCreated:    &created,
		TimeUpdated:    &created,
		ProductCodes:   []string{"COMPUTE"},
		Industries:     []string{"Technology"},
		Tagline:        common.String(tagline),
		FreeformTags:   map[string]string{"env": "dev"},
		Status:         marketplacepublishersdk.ListingRevisionStatusApproved,
		LifecycleState: state,
	}
}

func makeSDKListingRevisionSummary(
	id string,
	listingID string,
	displayName string,
	state marketplacepublishersdk.ListingRevisionLifecycleStateEnum,
) marketplacepublishersdk.ListingRevisionSummary {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return marketplacepublishersdk.ListingRevisionSummary{
		Id:             common.String(id),
		ListingId:      common.String(listingID),
		CompartmentId:  common.String(testCompartmentID),
		ListingType:    marketplacepublishersdk.ListingTypeService,
		DisplayName:    common.String(displayName),
		Status:         marketplacepublishersdk.ListingRevisionStatusApproved,
		LifecycleState: state,
		TimeCreated:    &created,
		TimeUpdated:    &created,
	}
}

type listingRevisionCreateProjectionCalls struct {
	list   int
	create int
	get    int
}

func listingRevisionCreateProjectionClient(
	t *testing.T,
	resource *marketplacepublisherv1beta1.ListingRevision,
	calls *listingRevisionCreateProjectionCalls,
) ListingRevisionServiceClient {
	t.Helper()
	return testListingRevisionClient(&fakeListingRevisionOCIClient{
		listFn: func(_ context.Context, request marketplacepublishersdk.ListListingRevisionsRequest) (marketplacepublishersdk.ListListingRevisionsResponse, error) {
			calls.list++
			requireListingRevisionStringPtr(t, "list listingId", request.ListingId, testListingID)
			requireListingRevisionStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
			return marketplacepublishersdk.ListListingRevisionsResponse{}, nil
		},
		createFn: func(_ context.Context, request marketplacepublishersdk.CreateListingRevisionRequest) (marketplacepublishersdk.CreateListingRevisionResponse, error) {
			calls.create++
			details, ok := request.CreateListingRevisionDetails.(marketplacepublishersdk.CreateServiceListingRevisionDetails)
			if !ok {
				t.Fatalf("CreateListingRevisionDetails = %T, want CreateServiceListingRevisionDetails", request.CreateListingRevisionDetails)
			}
			requireListingRevisionStringPtr(t, "create listingId", details.ListingId, testListingID)
			requireListingRevisionStringPtr(t, "create displayName", details.DisplayName, resource.Spec.DisplayName)
			requireListingRevisionStringPtr(t, "create headline", details.Headline, resource.Spec.Headline)
			if got := request.OpcRetryToken; got == nil || strings.TrimSpace(*got) == "" {
				t.Fatalf("create opcRetryToken = %v, want deterministic token", got)
			}
			return marketplacepublishersdk.CreateListingRevisionResponse{
				ListingRevision: makeSDKServiceListingRevision(
					testListingRevisionID,
					testListingID,
					resource.Spec.DisplayName,
					resource.Spec.Tagline,
					marketplacepublishersdk.ListingRevisionLifecycleStateCreating,
				),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
			calls.get++
			requireListingRevisionStringPtr(t, "get listingRevisionId", request.ListingRevisionId, testListingRevisionID)
			return marketplacepublishersdk.GetListingRevisionResponse{
				ListingRevision: makeSDKServiceListingRevision(
					testListingRevisionID,
					testListingID,
					resource.Spec.DisplayName,
					resource.Spec.Tagline,
					marketplacepublishersdk.ListingRevisionLifecycleStateActive,
				),
				OpcRequestId: common.String("opc-get"),
			}, nil
		},
	})
}

func TestListingRevisionRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := newListingRevisionRuntimeSemantics()
	if got == nil {
		t.Fatal("newListingRevisionRuntimeSemantics() = nil")
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	requireListingRevisionStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"listingId", "displayName", "listingType", "id"})
	requireListingRevisionStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"listingId", "listingType"})
}

func TestListingRevisionServiceClientCreatesPolymorphicServiceAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	calls := &listingRevisionCreateProjectionCalls{}
	client := listingRevisionCreateProjectionClient(t, resource, calls)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "listing-revision"}})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if calls.list != 1 || calls.create != 1 || calls.get != 1 {
		t.Fatalf("call counts list/create/get = %d/%d/%d, want 1/1/1", calls.list, calls.create, calls.get)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testListingRevisionID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testListingRevisionID)
	}
	if resource.Status.Status != string(marketplacepublishersdk.ListingRevisionStatusApproved) {
		t.Fatalf("status.sdkStatus = %q, want APPROVED", resource.Status.Status)
	}
	if resource.Status.LifecycleState != string(marketplacepublishersdk.ListingRevisionLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", resource.Status.OsokStatus.OpcRequestID)
	}
	requireListingRevisionLastCondition(t, resource, shared.Active)
}

func TestListingRevisionServiceClientBindsExistingAcrossListPages(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	createCalls := 0
	getCalls := 0
	var pages []string

	client := testListingRevisionClient(&fakeListingRevisionOCIClient{
		listFn: func(_ context.Context, request marketplacepublishersdk.ListListingRevisionsRequest) (marketplacepublishersdk.ListListingRevisionsResponse, error) {
			pages = append(pages, listingRevisionStringValue(request.Page))
			if request.Page == nil {
				return marketplacepublishersdk.ListListingRevisionsResponse{
					ListingRevisionCollection: marketplacepublishersdk.ListingRevisionCollection{
						Items: []marketplacepublishersdk.ListingRevisionSummary{
							makeSDKListingRevisionSummary(testListingRevisionOtherID, testListingID, "other revision", marketplacepublishersdk.ListingRevisionLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return marketplacepublishersdk.ListListingRevisionsResponse{
				ListingRevisionCollection: marketplacepublishersdk.ListingRevisionCollection{
					Items: []marketplacepublishersdk.ListingRevisionSummary{
						makeSDKListingRevisionSummary(testListingRevisionID, testListingID, resource.Spec.DisplayName, marketplacepublishersdk.ListingRevisionLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
			getCalls++
			requireListingRevisionStringPtr(t, "get listingRevisionId", request.ListingRevisionId, testListingRevisionID)
			return marketplacepublishersdk.GetListingRevisionResponse{
				ListingRevision: makeSDKServiceListingRevision(
					testListingRevisionID,
					testListingID,
					resource.Spec.DisplayName,
					resource.Spec.Tagline,
					marketplacepublishersdk.ListingRevisionLifecycleStateActive,
				),
			}, nil
		},
		createFn: func(context.Context, marketplacepublishersdk.CreateListingRevisionRequest) (marketplacepublishersdk.CreateListingRevisionResponse, error) {
			createCalls++
			return marketplacepublishersdk.CreateListingRevisionResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if createCalls != 0 {
		t.Fatalf("CreateListingRevision() calls = %d, want 0 for bind", createCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetListingRevision() calls = %d, want 1 live read after bind", getCalls)
	}
	if got, want := strings.Join(pages, ","), ",page-2"; got != want {
		t.Fatalf("list pages = %q, want %q", got, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testListingRevisionID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testListingRevisionID)
	}
}

func TestListingRevisionServiceClientBindsExistingWithoutSpecStatusFilter(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	resource.Spec.Status = string(marketplacepublishersdk.ListingRevisionStatusPublished)
	createCalls := 0
	getCalls := 0
	updateCalls := 0

	client := testListingRevisionClient(&fakeListingRevisionOCIClient{
		listFn: func(_ context.Context, request marketplacepublishersdk.ListListingRevisionsRequest) (marketplacepublishersdk.ListListingRevisionsResponse, error) {
			requireListingRevisionStringPtr(t, "list listingId", request.ListingId, testListingID)
			requireListingRevisionStringPtr(t, "list displayName", request.DisplayName, resource.Spec.DisplayName)
			if request.ListingRevisionStatus != "" {
				return marketplacepublishersdk.ListListingRevisionsResponse{}, nil
			}
			return marketplacepublishersdk.ListListingRevisionsResponse{
				ListingRevisionCollection: marketplacepublishersdk.ListingRevisionCollection{
					Items: []marketplacepublishersdk.ListingRevisionSummary{
						makeSDKListingRevisionSummary(
							testListingRevisionID,
							testListingID,
							resource.Spec.DisplayName,
							marketplacepublishersdk.ListingRevisionLifecycleStateActive,
						),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
			getCalls++
			requireListingRevisionStringPtr(t, "get listingRevisionId", request.ListingRevisionId, testListingRevisionID)
			return marketplacepublishersdk.GetListingRevisionResponse{
				ListingRevision: makeSDKServiceListingRevision(
					testListingRevisionID,
					testListingID,
					resource.Spec.DisplayName,
					resource.Spec.Tagline,
					marketplacepublishersdk.ListingRevisionLifecycleStateActive,
				),
			}, nil
		},
		createFn: func(context.Context, marketplacepublishersdk.CreateListingRevisionRequest) (marketplacepublishersdk.CreateListingRevisionResponse, error) {
			createCalls++
			return marketplacepublishersdk.CreateListingRevisionResponse{}, nil
		},
		updateFn: func(context.Context, marketplacepublishersdk.UpdateListingRevisionRequest) (marketplacepublishersdk.UpdateListingRevisionResponse, error) {
			updateCalls++
			return marketplacepublishersdk.UpdateListingRevisionResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want existing revision status drift rejection")
	}
	if createCalls != 0 {
		t.Fatalf("CreateListingRevision() calls = %d, want 0 after unfiltered bind lookup", createCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetListingRevision() calls = %d, want 1 live read after bind", getCalls)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateListingRevision() calls = %d, want 0 before status drift rejection", updateCalls)
	}
	if !strings.Contains(err.Error(), "status update is not supported") {
		t.Fatalf("CreateOrUpdate() error = %v, want unsupported status drift", err)
	}
}

func TestListingRevisionServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionID)
	resource.Spec.Tagline = "Updated tagline"
	getCalls := 0
	updateCalls := 0

	client := testListingRevisionClient(&fakeListingRevisionOCIClient{
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
			getCalls++
			requireListingRevisionStringPtr(t, "get listingRevisionId", request.ListingRevisionId, testListingRevisionID)
			tagline := "Initial tagline"
			if getCalls > 1 {
				tagline = resource.Spec.Tagline
			}
			return marketplacepublishersdk.GetListingRevisionResponse{
				ListingRevision: makeSDKServiceListingRevision(
					testListingRevisionID,
					testListingID,
					resource.Spec.DisplayName,
					tagline,
					marketplacepublishersdk.ListingRevisionLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(_ context.Context, request marketplacepublishersdk.UpdateListingRevisionRequest) (marketplacepublishersdk.UpdateListingRevisionResponse, error) {
			updateCalls++
			requireListingRevisionStringPtr(t, "update listingRevisionId", request.ListingRevisionId, testListingRevisionID)
			details, ok := request.UpdateListingRevisionDetails.(marketplacepublishersdk.UpdateServiceListingRevisionDetails)
			if !ok {
				t.Fatalf("UpdateListingRevisionDetails = %T, want UpdateServiceListingRevisionDetails", request.UpdateListingRevisionDetails)
			}
			requireListingRevisionStringPtr(t, "update tagline", details.Tagline, resource.Spec.Tagline)
			return marketplacepublishersdk.UpdateListingRevisionResponse{
				ListingRevision: makeSDKServiceListingRevision(
					testListingRevisionID,
					testListingID,
					resource.Spec.DisplayName,
					resource.Spec.Tagline,
					marketplacepublishersdk.ListingRevisionLifecycleStateUpdating,
				),
				OpcRequestId: common.String("opc-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if getCalls != 2 || updateCalls != 1 {
		t.Fatalf("call counts get/update = %d/%d, want 2/1", getCalls, updateCalls)
	}
	if resource.Status.Tagline != resource.Spec.Tagline {
		t.Fatalf("status.tagline = %q, want %q", resource.Status.Tagline, resource.Spec.Tagline)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestListingRevisionServiceClientUpdatesOciApplicationTypedFalseBool(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionID)
	resource.Spec.ListingType = string(marketplacepublishersdk.ListingTypeOciApplication)
	resource.Spec.DisplayName = "OCI listing"
	resource.Spec.Headline = "OCI headline"
	resource.Spec.Tagline = ""
	resource.Spec.FreeformTags = nil
	resource.Spec.ProductCodes = nil
	resource.Spec.Industries = nil
	resource.Spec.Products = []marketplacepublisherv1beta1.ListingRevisionProduct{
		{
			Code:       "DATABASE",
			Categories: []string{"APP"},
		},
	}
	resource.Spec.PricingType = string(marketplacepublishersdk.OciListingRevisionPricingTypePaygo)
	resource.Spec.IsRoverExportable = false
	getCalls := 0
	updateCalls := 0

	client := testListingRevisionClient(&fakeListingRevisionOCIClient{
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
			getCalls++
			requireListingRevisionStringPtr(t, "get listingRevisionId", request.ListingRevisionId, testListingRevisionID)
			current := makeSDKOciListingRevision(resource)
			current.IsRoverExportable = common.Bool(getCalls == 1)
			return marketplacepublishersdk.GetListingRevisionResponse{ListingRevision: current}, nil
		},
		updateFn: func(_ context.Context, request marketplacepublishersdk.UpdateListingRevisionRequest) (marketplacepublishersdk.UpdateListingRevisionResponse, error) {
			updateCalls++
			requireListingRevisionStringPtr(t, "update listingRevisionId", request.ListingRevisionId, testListingRevisionID)
			details, ok := request.UpdateListingRevisionDetails.(marketplacepublishersdk.UpdateOciListingRevisionDetails)
			if !ok {
				t.Fatalf("UpdateListingRevisionDetails = %T, want UpdateOciListingRevisionDetails", request.UpdateListingRevisionDetails)
			}
			if details.IsRoverExportable == nil || *details.IsRoverExportable {
				t.Fatalf("UpdateOciListingRevisionDetails.IsRoverExportable = %#v, want explicit false", details.IsRoverExportable)
			}
			updated := makeSDKOciListingRevision(resource)
			updated.IsRoverExportable = common.Bool(false)
			return marketplacepublishersdk.UpdateListingRevisionResponse{
				ListingRevision: updated,
				OpcRequestId:    common.String("opc-update-false"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if getCalls != 2 || updateCalls != 1 {
		t.Fatalf("call counts get/update = %d/%d, want 2/1", getCalls, updateCalls)
	}
	if resource.Status.IsRoverExportable {
		t.Fatal("status.isRoverExportable = true, want false")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update-false" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-false", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestListingRevisionUpdatePayloadFiltersInapplicableFieldsByListingType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource *marketplacepublisherv1beta1.ListingRevision
		current  map[string]any
	}{
		{
			name:     "service ignores oci and lead generation fields",
			resource: makeServiceListingRevisionWithInapplicableFields(),
			current: map[string]any{
				"listingId":      testListingID,
				"listingType":    string(marketplacepublishersdk.ListingTypeService),
				"displayName":    "Partner service",
				"headline":       "Partner service headline",
				"tagline":        "Initial tagline",
				"productCodes":   []any{"COMPUTE"},
				"industries":     []any{"Technology"},
				"freeformTags":   map[string]any{"env": "dev"},
				"lifecycleState": string(marketplacepublishersdk.ListingRevisionLifecycleStateActive),
				"id":             testListingRevisionID,
				"sdkStatus":      string(marketplacepublishersdk.ListingRevisionStatusApproved),
			},
		},
		{
			name:     "oci application ignores service fields",
			resource: makeOciListingRevisionWithInapplicableFields(),
			current: map[string]any{
				"listingId":      testListingID,
				"listingType":    string(marketplacepublishersdk.ListingTypeOciApplication),
				"displayName":    "OCI listing",
				"headline":       "OCI headline",
				"tagline":        "Initial tagline",
				"products":       []any{map[string]any{"code": "DATABASE", "categories": []any{"APP"}}},
				"pricingType":    string(marketplacepublishersdk.OciListingRevisionPricingTypePaygo),
				"lifecycleState": string(marketplacepublishersdk.ListingRevisionLifecycleStateActive),
				"id":             testListingRevisionID,
				"sdkStatus":      string(marketplacepublishersdk.ListingRevisionStatusApproved),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, updateNeeded, err := buildListingRevisionUpdateBody(tt.resource, tt.current)
			if err != nil {
				t.Fatalf("buildListingRevisionUpdateBody() error = %v", err)
			}
			if updateNeeded {
				t.Fatal("buildListingRevisionUpdateBody() updateNeeded = true, want false for subtype-inapplicable fields")
			}
		})
	}
}

func TestListingRevisionServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionID)
	updateCalls := 0

	client := testListingRevisionClient(&fakeListingRevisionOCIClient{
		getFn: func(context.Context, marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
			return marketplacepublishersdk.GetListingRevisionResponse{
				ListingRevision: makeSDKServiceListingRevision(
					testListingRevisionID,
					testListingID+"-different",
					resource.Spec.DisplayName,
					resource.Spec.Tagline,
					marketplacepublishersdk.ListingRevisionLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(context.Context, marketplacepublishersdk.UpdateListingRevisionRequest) (marketplacepublishersdk.UpdateListingRevisionResponse, error) {
			updateCalls++
			return marketplacepublishersdk.UpdateListingRevisionResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateListingRevision() calls = %d, want 0 before drift rejection", updateCalls)
	}
	if !strings.Contains(err.Error(), "listingId") {
		t.Fatalf("CreateOrUpdate() error = %v, want listingId drift", err)
	}
}

func TestListingRevisionServiceClientRejectsStatusDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionID)
	resource.Spec.Status = string(marketplacepublishersdk.ListingRevisionStatusPublished)
	updateCalls := 0

	client := testListingRevisionClient(&fakeListingRevisionOCIClient{
		getFn: func(context.Context, marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
			return marketplacepublishersdk.GetListingRevisionResponse{
				ListingRevision: makeSDKServiceListingRevision(
					testListingRevisionID,
					testListingID,
					resource.Spec.DisplayName,
					resource.Spec.Tagline,
					marketplacepublishersdk.ListingRevisionLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(context.Context, marketplacepublishersdk.UpdateListingRevisionRequest) (marketplacepublishersdk.UpdateListingRevisionResponse, error) {
			updateCalls++
			return marketplacepublishersdk.UpdateListingRevisionResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want unsupported status drift rejection")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateListingRevision() calls = %d, want 0 before status drift rejection", updateCalls)
	}
	if !strings.Contains(err.Error(), "status update is not supported") {
		t.Fatalf("CreateOrUpdate() error = %v, want unsupported status drift", err)
	}
}

func TestListingRevisionServiceClientRetainsFinalizerWhileDeleteIsPending(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionID)
	getCalls := 0
	deleteCalls := 0

	client := testListingRevisionClient(&fakeListingRevisionOCIClient{
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
			getCalls++
			requireListingRevisionStringPtr(t, "get listingRevisionId", request.ListingRevisionId, testListingRevisionID)
			state := marketplacepublishersdk.ListingRevisionLifecycleStateActive
			if getCalls > 1 {
				state = marketplacepublishersdk.ListingRevisionLifecycleStateDeleting
			}
			return marketplacepublishersdk.GetListingRevisionResponse{
				ListingRevision: makeSDKServiceListingRevision(
					testListingRevisionID,
					testListingID,
					resource.Spec.DisplayName,
					resource.Spec.Tagline,
					state,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request marketplacepublishersdk.DeleteListingRevisionRequest) (marketplacepublishersdk.DeleteListingRevisionResponse, error) {
			deleteCalls++
			requireListingRevisionStringPtr(t, "delete listingRevisionId", request.ListingRevisionId, testListingRevisionID)
			return marketplacepublishersdk.DeleteListingRevisionResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI lifecycle is DELETING")
	}
	if getCalls != 2 || deleteCalls != 1 {
		t.Fatalf("call counts get/delete = %d/%d, want 2/1", getCalls, deleteCalls)
	}
	if resource.Status.LifecycleState != string(marketplacepublishersdk.ListingRevisionLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", resource.Status.OsokStatus.OpcRequestID)
	}
	requireListingRevisionLastCondition(t, resource, shared.Terminating)
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want delete tracker", current)
	}
}

func TestListingRevisionServiceClientRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionID)
	deleteCalls := 0
	getErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	getErr.OpcRequestID = "opc-get-predelete"

	client := testListingRevisionClient(&fakeListingRevisionOCIClient{
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
			requireListingRevisionStringPtr(t, "get listingRevisionId", request.ListingRevisionId, testListingRevisionID)
			return marketplacepublishersdk.GetListingRevisionResponse{}, getErr
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteListingRevisionRequest) (marketplacepublishersdk.DeleteListingRevisionResponse, error) {
			deleteCalls++
			return marketplacepublishersdk.DeleteListingRevisionResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete read")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteListingRevision() calls = %d, want 0 after ambiguous pre-delete read", deleteCalls)
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous pre-delete confirm-read classification", err)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-get-predelete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-get-predelete", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt is set, want finalizer retained")
	}
}

func TestListingRevisionServiceClientDeleteWithTrackedIDIgnoresOmittedCreateFields(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testListingRevisionID)
	resource.Spec.DisplayName = "drifted current spec"
	resource.Spec.ListingType = ""
	resource.Spec.ProductCodes = nil
	resource.Spec.Industries = nil
	getCalls := 0
	deleteCalls := 0
	listCalls := 0

	client := testListingRevisionClient(&fakeListingRevisionOCIClient{
		listFn: func(context.Context, marketplacepublishersdk.ListListingRevisionsRequest) (marketplacepublishersdk.ListListingRevisionsResponse, error) {
			listCalls++
			return marketplacepublishersdk.ListListingRevisionsResponse{}, nil
		},
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
			getCalls++
			requireListingRevisionStringPtr(t, "get listingRevisionId", request.ListingRevisionId, testListingRevisionID)
			state := marketplacepublishersdk.ListingRevisionLifecycleStateActive
			if getCalls > 1 {
				state = marketplacepublishersdk.ListingRevisionLifecycleStateDeleted
			}
			return marketplacepublishersdk.GetListingRevisionResponse{
				ListingRevision: makeSDKServiceListingRevision(
					testListingRevisionID,
					testListingID,
					"recorded revision",
					"recorded tagline",
					state,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request marketplacepublishersdk.DeleteListingRevisionRequest) (marketplacepublishersdk.DeleteListingRevisionResponse, error) {
			deleteCalls++
			requireListingRevisionStringPtr(t, "delete listingRevisionId", request.ListingRevisionId, testListingRevisionID)
			return marketplacepublishersdk.DeleteListingRevisionResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after terminal delete confirmation")
	}
	if listCalls != 0 {
		t.Fatalf("ListListingRevisions() calls = %d, want 0 when status.status.ocid is recorded", listCalls)
	}
	if getCalls != 2 || deleteCalls != 1 {
		t.Fatalf("call counts get/delete = %d/%d, want 2/1", getCalls, deleteCalls)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deleted marker")
	}
}

func TestListingRevisionServiceClientDeleteWithoutTrackedIDUsesRecordedStatusIdentity(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	resource.Status.ListingId = testListingID
	resource.Status.DisplayName = "recorded revision"
	resource.Status.ListingType = string(marketplacepublishersdk.ListingTypeService)
	resource.Spec.ListingId = testListingID + "-drifted"
	resource.Spec.DisplayName = "current spec revision"
	resource.Spec.ListingType = ""
	resource.Spec.ProductCodes = nil
	resource.Spec.Industries = nil
	listCalls := 0
	getCalls := 0
	deleteCalls := 0

	client := testListingRevisionClient(&fakeListingRevisionOCIClient{
		listFn: func(_ context.Context, request marketplacepublishersdk.ListListingRevisionsRequest) (marketplacepublishersdk.ListListingRevisionsResponse, error) {
			listCalls++
			requireListingRevisionStringPtr(t, "list listingId", request.ListingId, testListingID)
			requireListingRevisionStringPtr(t, "list displayName", request.DisplayName, "recorded revision")
			return marketplacepublishersdk.ListListingRevisionsResponse{
				ListingRevisionCollection: marketplacepublishersdk.ListingRevisionCollection{
					Items: []marketplacepublishersdk.ListingRevisionSummary{
						makeSDKListingRevisionSummary(
							testListingRevisionID,
							testListingID,
							"recorded revision",
							marketplacepublishersdk.ListingRevisionLifecycleStateActive,
						),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
			getCalls++
			requireListingRevisionStringPtr(t, "get listingRevisionId", request.ListingRevisionId, testListingRevisionID)
			state := marketplacepublishersdk.ListingRevisionLifecycleStateActive
			if getCalls > 1 {
				state = marketplacepublishersdk.ListingRevisionLifecycleStateDeleted
			}
			return marketplacepublishersdk.GetListingRevisionResponse{
				ListingRevision: makeSDKServiceListingRevision(
					testListingRevisionID,
					testListingID,
					"recorded revision",
					"recorded tagline",
					state,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request marketplacepublishersdk.DeleteListingRevisionRequest) (marketplacepublishersdk.DeleteListingRevisionResponse, error) {
			deleteCalls++
			requireListingRevisionStringPtr(t, "delete listingRevisionId", request.ListingRevisionId, testListingRevisionID)
			return marketplacepublishersdk.DeleteListingRevisionResponse{OpcRequestId: common.String("opc-delete-status-identity")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after status-identity delete confirmation")
	}
	if listCalls != 1 {
		t.Fatalf("ListListingRevisions() calls = %d, want 1 recorded status lookup", listCalls)
	}
	if getCalls != 2 || deleteCalls != 1 {
		t.Fatalf("call counts get/delete = %d/%d, want 2/1", getCalls, deleteCalls)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-status-identity" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-status-identity", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deleted marker")
	}
}

func TestListingRevisionServiceClientDeleteWithoutTrackedIDRejectsDuplicateListMatches(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	resource.Spec.DisplayName = ""
	deleteCalls := 0
	var pages []string

	client := testListingRevisionClient(&fakeListingRevisionOCIClient{
		listFn: func(_ context.Context, request marketplacepublishersdk.ListListingRevisionsRequest) (marketplacepublishersdk.ListListingRevisionsResponse, error) {
			pages = append(pages, listingRevisionStringValue(request.Page))
			requireListingRevisionStringPtr(t, "list listingId", request.ListingId, testListingID)
			if request.DisplayName != nil {
				t.Fatalf("list displayName = %q, want nil for broad listingId/listingType delete lookup", *request.DisplayName)
			}
			if request.Page == nil {
				return marketplacepublishersdk.ListListingRevisionsResponse{
					ListingRevisionCollection: marketplacepublishersdk.ListingRevisionCollection{
						Items: []marketplacepublishersdk.ListingRevisionSummary{
							makeSDKListingRevisionSummary(testListingRevisionID, testListingID, "first revision", marketplacepublishersdk.ListingRevisionLifecycleStateActive),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return marketplacepublishersdk.ListListingRevisionsResponse{
				ListingRevisionCollection: marketplacepublishersdk.ListingRevisionCollection{
					Items: []marketplacepublishersdk.ListingRevisionSummary{
						makeSDKListingRevisionSummary(testListingRevisionOtherID, testListingID, "second revision", marketplacepublishersdk.ListingRevisionLifecycleStateActive),
					},
				},
			}, nil
		},
		deleteFn: func(context.Context, marketplacepublishersdk.DeleteListingRevisionRequest) (marketplacepublishersdk.DeleteListingRevisionResponse, error) {
			deleteCalls++
			return marketplacepublishersdk.DeleteListingRevisionResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want duplicate-match error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous duplicate matches")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteListingRevision() calls = %d, want 0 for ambiguous duplicate matches", deleteCalls)
	}
	if got, want := strings.Join(pages, ","), ",page-2"; got != want {
		t.Fatalf("list pages = %q, want %q", got, want)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %#v, want nil for ambiguous duplicate matches", resource.Status.OsokStatus.DeletedAt)
	}
	if !strings.Contains(err.Error(), "found 2 listing revisions matching listingId") {
		t.Fatalf("Delete() error = %v, want duplicate-match detail", err)
	}
}

func TestListingRevisionLeadGenJsonDataNormalizesPricingPlansString(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	resource.Spec.JsonData = `{
		"listingId":"ocid1.marketplacelisting.oc1..lead",
		"listingType":"LEAD_GENERATION",
		"displayName":"Lead listing",
		"headline":"Lead headline",
		"products":[{"code":"COMPUTE","categories":["APP"]}],
		"pricingType":"FREE",
		"isRoverExportable":false,
		"pricingPlans":[{"planType":"FIXED","rates":[{"billingCurrency":"USD","rate":0}]}]
	}`
	resource.Spec.ListingId = "ocid1.marketplacelisting.oc1..lead"
	resource.Spec.ListingType = string(marketplacepublishersdk.ListingTypeLeadGeneration)

	body, err := buildListingRevisionCreateBody(resource)
	if err != nil {
		t.Fatalf("buildListingRevisionCreateBody() error = %v", err)
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	details, err := listingRevisionCreateDetailsFromRaw(payload)
	if err != nil {
		t.Fatalf("listingRevisionCreateDetailsFromRaw() error = %v", err)
	}
	leadGen, ok := details.(marketplacepublishersdk.CreateLeadGenListingRevisionDetails)
	if !ok {
		t.Fatalf("details = %T, want CreateLeadGenListingRevisionDetails", details)
	}
	if leadGen.PricingPlans == nil || !strings.Contains(*leadGen.PricingPlans, `"planType":"FIXED"`) {
		t.Fatalf("PricingPlans = %v, want compact JSON string", leadGen.PricingPlans)
	}
	if value, ok := body["isRoverExportable"].(bool); !ok || value {
		t.Fatalf("body.isRoverExportable = %#v, want explicit false preserved from jsonData", body["isRoverExportable"])
	}
}

func TestListingRevisionLeadGenTypedSpecPreservesZeroPricingRate(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	resource.Spec.ListingId = "ocid1.marketplacelisting.oc1..lead"
	resource.Spec.ListingType = string(marketplacepublishersdk.ListingTypeLeadGeneration)
	resource.Spec.DisplayName = "Lead listing"
	resource.Spec.Headline = "Lead headline"
	resource.Spec.Products = []marketplacepublisherv1beta1.ListingRevisionProduct{
		{
			Code:       "COMPUTE",
			Categories: []string{"APP"},
		},
	}
	resource.Spec.PricingType = string(marketplacepublishersdk.OciListingRevisionPricingTypePaygo)
	resource.Spec.PricingPlans = []marketplacepublisherv1beta1.ListingRevisionPricingPlan{
		{
			PlanType: string(marketplacepublishersdk.PricingPlanPlanTypeFixed),
			Rates: []marketplacepublisherv1beta1.ListingRevisionPricingPlanRate{
				{
					BillingCurrency: "USD",
					Rate:            0,
				},
			},
		},
	}

	body, err := buildListingRevisionCreateBody(resource)
	if err != nil {
		t.Fatalf("buildListingRevisionCreateBody() error = %v", err)
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	details, err := listingRevisionCreateDetailsFromRaw(payload)
	if err != nil {
		t.Fatalf("listingRevisionCreateDetailsFromRaw() error = %v", err)
	}
	leadGen, ok := details.(marketplacepublishersdk.CreateLeadGenListingRevisionDetails)
	if !ok {
		t.Fatalf("details = %T, want CreateLeadGenListingRevisionDetails", details)
	}
	if leadGen.PricingPlans == nil || !strings.Contains(*leadGen.PricingPlans, `"rate":0`) {
		t.Fatalf("PricingPlans = %v, want typed zero rate preserved", leadGen.PricingPlans)
	}
}

func TestListingRevisionOciResponseProjectionNormalizesPricingPlansAndSDKStatus(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	resource.Spec.JsonData = `{
		"listingId":"ocid1.marketplacelisting.oc1..oci",
		"listingType":"OCI_APPLICATION",
		"displayName":"OCI listing",
		"headline":"OCI headline",
		"products":[{"code":"DATABASE","categories":["APP"]}],
		"pricingType":"PAYGO"
	}`
	resource.Spec.ListingId = "ocid1.marketplacelisting.oc1..oci"
	resource.Spec.ListingType = string(marketplacepublishersdk.ListingTypeOciApplication)
	createCalls := 0

	client := testListingRevisionClient(&fakeListingRevisionOCIClient{
		listFn: func(context.Context, marketplacepublishersdk.ListListingRevisionsRequest) (marketplacepublishersdk.ListListingRevisionsResponse, error) {
			return marketplacepublishersdk.ListListingRevisionsResponse{}, nil
		},
		createFn: func(context.Context, marketplacepublishersdk.CreateListingRevisionRequest) (marketplacepublishersdk.CreateListingRevisionResponse, error) {
			createCalls++
			return marketplacepublishersdk.CreateListingRevisionResponse{
				ListingRevision: makeSDKOciListingRevision(resource),
				OpcRequestId:    common.String("opc-create"),
			}, nil
		},
		getFn: func(context.Context, marketplacepublishersdk.GetListingRevisionRequest) (marketplacepublishersdk.GetListingRevisionResponse, error) {
			return marketplacepublishersdk.GetListingRevisionResponse{ListingRevision: makeSDKOciListingRevision(resource)}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if createCalls != 1 {
		t.Fatalf("CreateListingRevision() calls = %d, want 1", createCalls)
	}
	if resource.Status.Status != string(marketplacepublishersdk.ListingRevisionStatusApproved) {
		t.Fatalf("status.sdkStatus = %q, want APPROVED", resource.Status.Status)
	}
	if !strings.Contains(resource.Status.PricingPlans, `"planType":"FIXED"`) {
		t.Fatalf("status.pricingPlans = %q, want JSON pricing plan string", resource.Status.PricingPlans)
	}
	if strings.Contains(resource.Status.JsonData, `"sdkStatus"`) {
		t.Fatalf("status.jsonData = %q, want raw SDK JSON with service status field", resource.Status.JsonData)
	}
}

func makeSDKOciListingRevision(resource *marketplacepublisherv1beta1.ListingRevision) marketplacepublishersdk.OciListingRevision {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return marketplacepublishersdk.OciListingRevision{
		Id:            common.String(testListingRevisionID),
		ListingId:     common.String(resource.Spec.ListingId),
		CompartmentId: common.String(testCompartmentID),
		DisplayName:   common.String("OCI listing"),
		Headline:      common.String("OCI headline"),
		TimeCreated:   &created,
		TimeUpdated:   &created,
		Products:      []marketplacepublishersdk.ListingProduct{{Code: common.String("DATABASE"), Categories: []string{"APP"}}},
		PricingType:   marketplacepublishersdk.OciListingRevisionPricingTypePaygo,
		PricingPlans: []marketplacepublishersdk.PricingPlan{marketplacepublishersdk.SaaSPricingPlan{
			Name:             common.String("monthly"),
			PlanDescription:  common.String("monthly plan"),
			BillingFrequency: marketplacepublishersdk.SaaSPricingPlanBillingFrequencyMonthly,
		}},
		Status:         marketplacepublishersdk.ListingRevisionStatusApproved,
		LifecycleState: marketplacepublishersdk.ListingRevisionLifecycleStateActive,
	}
}

func requireListingRevisionStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireListingRevisionLastCondition(
	t *testing.T,
	resource *marketplacepublisherv1beta1.ListingRevision,
	want shared.OSOKConditionType,
) {
	t.Helper()
	if len(resource.Status.OsokStatus.Conditions) == 0 {
		t.Fatalf("status.status.conditions = empty, want %s", want)
	}
	last := resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1]
	if last.Type != want {
		t.Fatalf("last condition = %q, want %q", last.Type, want)
	}
}

func requireListingRevisionStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s = %v, want %v", name, got, want)
		}
	}
}
