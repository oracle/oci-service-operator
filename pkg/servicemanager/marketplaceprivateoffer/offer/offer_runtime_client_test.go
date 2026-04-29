/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package offer

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	offersdk "github.com/oracle/oci-go-sdk/v65/marketplaceprivateoffer"
	offerv1beta1 "github.com/oracle/oci-service-operator/api/marketplaceprivateoffer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testOfferID          = "ocid1.offer.oc1..runtime"
	testOfferSeller      = "ocid1.tenancy.oc1..seller"
	testOfferBuyer       = "ocid1.tenancy.oc1..buyer"
	testOfferDisplayName = "offer-runtime"
)

type fakeOfferOCIClient struct {
	createFn   func(context.Context, offersdk.CreateOfferRequest) (offersdk.CreateOfferResponse, error)
	getFn      func(context.Context, offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error)
	internalFn func(context.Context, offersdk.GetOfferInternalDetailRequest) (offersdk.GetOfferInternalDetailResponse, error)
	listFn     func(context.Context, offersdk.ListOffersRequest) (offersdk.ListOffersResponse, error)
	updateFn   func(context.Context, offersdk.UpdateOfferRequest) (offersdk.UpdateOfferResponse, error)
	deleteFn   func(context.Context, offersdk.DeleteOfferRequest) (offersdk.DeleteOfferResponse, error)

	createRequests   []offersdk.CreateOfferRequest
	getRequests      []offersdk.GetOfferRequest
	internalRequests []offersdk.GetOfferInternalDetailRequest
	listRequests     []offersdk.ListOffersRequest
	updateRequests   []offersdk.UpdateOfferRequest
	deleteRequests   []offersdk.DeleteOfferRequest
}

func (f *fakeOfferOCIClient) CreateOffer(ctx context.Context, req offersdk.CreateOfferRequest) (offersdk.CreateOfferResponse, error) {
	f.createRequests = append(f.createRequests, req)
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return offersdk.CreateOfferResponse{}, nil
}

func (f *fakeOfferOCIClient) GetOffer(ctx context.Context, req offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error) {
	f.getRequests = append(f.getRequests, req)
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return offersdk.GetOfferResponse{}, nil
}

func (f *fakeOfferOCIClient) GetOfferInternalDetail(ctx context.Context, req offersdk.GetOfferInternalDetailRequest) (offersdk.GetOfferInternalDetailResponse, error) {
	f.internalRequests = append(f.internalRequests, req)
	if f.internalFn != nil {
		return f.internalFn(ctx, req)
	}
	return offersdk.GetOfferInternalDetailResponse{}, nil
}

func (f *fakeOfferOCIClient) ListOffers(ctx context.Context, req offersdk.ListOffersRequest) (offersdk.ListOffersResponse, error) {
	f.listRequests = append(f.listRequests, req)
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return offersdk.ListOffersResponse{}, nil
}

func (f *fakeOfferOCIClient) UpdateOffer(ctx context.Context, req offersdk.UpdateOfferRequest) (offersdk.UpdateOfferResponse, error) {
	f.updateRequests = append(f.updateRequests, req)
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return offersdk.UpdateOfferResponse{}, nil
}

func (f *fakeOfferOCIClient) DeleteOffer(ctx context.Context, req offersdk.DeleteOfferRequest) (offersdk.DeleteOfferResponse, error) {
	f.deleteRequests = append(f.deleteRequests, req)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return offersdk.DeleteOfferResponse{}, nil
}

func TestOfferRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	t.Parallel()

	got := newOfferRuntimeSemantics()
	if got == nil {
		t.Fatal("newOfferRuntimeSemantics() = nil")
	}
	assertOfferRuntimeSemanticsBasics(t, got)
	assertOfferRuntimeSemanticsSlices(t, got)
	assertOfferListFieldsSellerScoped(t)
}

func assertOfferRuntimeSemanticsBasics(t *testing.T, got *generatedruntime.Semantics) {
	t.Helper()
	if got.FormalService != "marketplaceprivateoffer" {
		t.Fatalf("FormalService = %q, want marketplaceprivateoffer", got.FormalService)
	}
	if got.FormalSlug != "offer" {
		t.Fatalf("FormalSlug = %q, want offer", got.FormalSlug)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want read-after-write", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "read-after-write" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want read-after-write", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want confirm-delete", got.DeleteFollowUp.Strategy)
	}
}

func assertOfferRuntimeSemanticsSlices(t *testing.T, got *generatedruntime.Semantics) {
	t.Helper()
	requireOfferStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	requireOfferStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	requireOfferStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	requireOfferStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	requireOfferStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	requireOfferStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"sellerCompartmentId", "buyerCompartmentId", "displayName", "id"})
	requireOfferStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"sellerCompartmentId"})
}

func assertOfferListFieldsSellerScoped(t *testing.T) {
	t.Helper()
	fields := offerListFields()
	for _, field := range fields {
		if field.FieldName == "BuyerCompartmentId" {
			t.Fatal("offerListFields() includes BuyerCompartmentId; ListOffers accepts either buyer or seller compartment, not both")
		}
	}
}

func TestOfferServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := newOfferRuntimeTestResource()
	var createRequest offersdk.CreateOfferRequest
	fake := &fakeOfferOCIClient{}
	fake.listFn = func(_ context.Context, req offersdk.ListOffersRequest) (offersdk.ListOffersResponse, error) {
		requireOfferStringPtr(t, "list sellerCompartmentId", req.SellerCompartmentId, resource.Spec.SellerCompartmentId)
		requireOfferStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
		if req.BuyerCompartmentId != nil {
			t.Fatal("list buyerCompartmentId is set; Offer list must not send both buyer and seller")
		}
		return offersdk.ListOffersResponse{}, nil
	}
	fake.createFn = func(_ context.Context, req offersdk.CreateOfferRequest) (offersdk.CreateOfferResponse, error) {
		createRequest = req
		if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
			t.Fatal("create opc retry token is empty, want deterministic retry token")
		}
		return offersdk.CreateOfferResponse{
			Offer:        offerFromSpec(testOfferID, resource.Spec, offersdk.OfferLifecycleStateCreating),
			OpcRequestId: common.String("opc-create-1"),
		}, nil
	}
	fake.getFn = func(_ context.Context, req offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error) {
		requireOfferStringPtr(t, "get offerId", req.OfferId, testOfferID)
		return offersdk.GetOfferResponse{
			Offer: offerFromSpec(testOfferID, resource.Spec, offersdk.OfferLifecycleStateActive),
		}, nil
	}
	client := newOfferRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	requireOfferStringPtr(t, "create displayName", createRequest.DisplayName, resource.Spec.DisplayName)
	requireOfferStringPtr(t, "create sellerCompartmentId", createRequest.SellerCompartmentId, resource.Spec.SellerCompartmentId)
	requireOfferStringPtr(t, "create buyerCompartmentId", createRequest.BuyerCompartmentId, resource.Spec.BuyerCompartmentId)
	if createRequest.Pricing != nil {
		t.Fatal("create pricing is set for an empty pricing spec")
	}
	assertOfferStatusIDAndRequestID(t, resource, testOfferID, "opc-create-1")
	if got := resource.Status.LifecycleState; got != string(offersdk.OfferLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestOfferBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	resource := newOfferRuntimeTestResource()
	fake := &fakeOfferOCIClient{}
	fake.listFn = func(_ context.Context, req offersdk.ListOffersRequest) (offersdk.ListOffersResponse, error) {
		return paginatedOfferBindResponse(t, fake, resource, req)
	}
	fake.getFn = func(_ context.Context, req offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error) {
		requireOfferStringPtr(t, "get offerId", req.OfferId, testOfferID)
		return offersdk.GetOfferResponse{Offer: offerFromSpec(testOfferID, resource.Spec, offersdk.OfferLifecycleStateActive)}, nil
	}
	fake.createFn = func(context.Context, offersdk.CreateOfferRequest) (offersdk.CreateOfferResponse, error) {
		t.Fatal("CreateOffer() called when paginated list found a reusable Offer")
		return offersdk.CreateOfferResponse{}, nil
	}
	client := newOfferRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2", len(fake.listRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testOfferID {
		t.Fatalf("status.ocid = %q, want %q", got, testOfferID)
	}
}

func paginatedOfferBindResponse(
	t *testing.T,
	fake *fakeOfferOCIClient,
	resource *offerv1beta1.Offer,
	req offersdk.ListOffersRequest,
) (offersdk.ListOffersResponse, error) {
	t.Helper()
	requireOfferStringPtr(t, "list sellerCompartmentId", req.SellerCompartmentId, resource.Spec.SellerCompartmentId)
	requireOfferStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
	if req.BuyerCompartmentId != nil {
		t.Fatal("list buyerCompartmentId is set; Offer list must stay seller-scoped")
	}
	switch len(fake.listRequests) {
	case 1:
		return firstOfferBindPage(t, resource, req)
	case 2:
		return secondOfferBindPage(t, resource, req)
	default:
		t.Fatalf("unexpected list request count %d", len(fake.listRequests))
		return offersdk.ListOffersResponse{}, nil
	}
}

func firstOfferBindPage(
	t *testing.T,
	resource *offerv1beta1.Offer,
	req offersdk.ListOffersRequest,
) (offersdk.ListOffersResponse, error) {
	t.Helper()
	if req.Page != nil {
		t.Fatalf("first list page = %q, want nil", offerStringValue(req.Page))
	}
	return offersdk.ListOffersResponse{
		OfferCollection: offersdk.OfferCollection{
			Items: []offersdk.OfferSummary{
				offerSummaryFromSpec("ocid1.offer.oc1..other", resource.Spec, "other-offer", offersdk.OfferLifecycleStateActive),
			},
		},
		OpcNextPage: common.String("page-2"),
	}, nil
}

func secondOfferBindPage(
	t *testing.T,
	resource *offerv1beta1.Offer,
	req offersdk.ListOffersRequest,
) (offersdk.ListOffersResponse, error) {
	t.Helper()
	requireOfferStringPtr(t, "second list page", req.Page, "page-2")
	return offersdk.ListOffersResponse{
		OfferCollection: offersdk.OfferCollection{
			Items: []offersdk.OfferSummary{
				offerSummaryFromSpec(testOfferID, resource.Spec, resource.Spec.DisplayName, offersdk.OfferLifecycleStateActive),
			},
		},
	}, nil
}

func TestOfferNoOpReconcileDoesNotUpdate(t *testing.T) {
	t.Parallel()

	resource := newOfferRuntimeTestResource()
	trackOfferID(resource, testOfferID)
	fake := &fakeOfferOCIClient{}
	fake.getFn = func(context.Context, offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error) {
		return offersdk.GetOfferResponse{Offer: offerFromSpec(testOfferID, resource.Spec, offersdk.OfferLifecycleStateActive)}, nil
	}
	client := newOfferRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
}

func TestOfferUpdatesSupportedMutableDrift(t *testing.T) {
	t.Parallel()

	resource := newOfferRuntimeTestResource()
	trackOfferID(resource, testOfferID)
	resource.Spec.DisplayName = "offer-runtime-updated"
	resource.Spec.Description = "updated description"
	resource.Spec.InternalNotes = "updated internal notes"
	resource.Spec.Pricing = offerv1beta1.OfferPricing{
		CurrencyType: "USD",
		TotalAmount:  42,
		BillingCycle: string(offersdk.PricingBillingCycleOneTime),
	}
	resource.Spec.CustomFields = []offerv1beta1.OfferCustomField{{Key: "plan", Value: "private"}}
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}

	oldSpec := newOfferRuntimeTestResource().Spec
	fake := &fakeOfferOCIClient{}
	fake.getFn = func(context.Context, offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error) {
		switch len(fake.getRequests) {
		case 1:
			return offersdk.GetOfferResponse{Offer: offerFromSpec(testOfferID, oldSpec, offersdk.OfferLifecycleStateActive)}, nil
		default:
			return offersdk.GetOfferResponse{Offer: offerFromSpec(testOfferID, resource.Spec, offersdk.OfferLifecycleStateActive)}, nil
		}
	}
	fake.internalFn = func(context.Context, offersdk.GetOfferInternalDetailRequest) (offersdk.GetOfferInternalDetailResponse, error) {
		return offersdk.GetOfferInternalDetailResponse{
			OfferInternalDetail: offersdk.OfferInternalDetail{
				InternalNotes: common.String("old notes"),
				CustomFields:  []offersdk.CustomField{{Key: common.String("plan"), Value: common.String("old")}},
			},
		}, nil
	}
	fake.updateFn = func(_ context.Context, req offersdk.UpdateOfferRequest) (offersdk.UpdateOfferResponse, error) {
		return offersdk.UpdateOfferResponse{
			Offer:        offerFromSpec(testOfferID, resource.Spec, offersdk.OfferLifecycleStateActive),
			OpcRequestId: common.String("opc-update-1"),
		}, nil
	}
	client := newOfferRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("response = %#v, want successful without requeue", response)
	}
	if len(fake.internalRequests) != 2 {
		t.Fatalf("internal detail requests = %d, want 2 update assessment/body reads", len(fake.internalRequests))
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(fake.updateRequests))
	}
	assertOfferMutableUpdateRequest(t, fake.updateRequests[0], resource.Spec)
	assertOfferStatusIDAndRequestID(t, resource, testOfferID, "opc-update-1")
}

func assertOfferMutableUpdateRequest(t *testing.T, update offersdk.UpdateOfferRequest, spec offerv1beta1.OfferSpec) {
	t.Helper()
	requireOfferStringPtr(t, "update offerId", update.OfferId, testOfferID)
	requireOfferStringPtr(t, "update displayName", update.DisplayName, spec.DisplayName)
	requireOfferStringPtr(t, "update description", update.Description, spec.Description)
	requireOfferStringPtr(t, "update internalNotes", update.InternalNotes, spec.InternalNotes)
	assertOfferMutableUpdatePricing(t, update)
	if !reflect.DeepEqual(update.FreeformTags, spec.FreeformTags) {
		t.Fatalf("update freeformTags = %#v, want %#v", update.FreeformTags, spec.FreeformTags)
	}
	if got := update.DefinedTags["Operations"]["CostCenter"]; got != "84" {
		t.Fatalf("update definedTags Operations.CostCenter = %#v, want 84", got)
	}
	if len(update.CustomFields) != 1 || offerStringValue(update.CustomFields[0].Value) != "private" {
		t.Fatalf("update customFields = %#v, want private plan", update.CustomFields)
	}
}

func assertOfferMutableUpdatePricing(t *testing.T, update offersdk.UpdateOfferRequest) {
	t.Helper()
	if update.Pricing == nil {
		t.Fatal("update pricing = nil, want USD/42")
	}
	if offerStringValue(update.Pricing.CurrencyType) != "USD" || offerInt64Value(update.Pricing.TotalAmount) != 42 {
		t.Fatalf("update pricing = %#v, want USD/42", update.Pricing)
	}
}

func TestOfferRejectsSellerCompartmentDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newOfferRuntimeTestResource()
	trackOfferID(resource, testOfferID)
	resource.Spec.SellerCompartmentId = "ocid1.tenancy.oc1..different"

	currentSpec := newOfferRuntimeTestResource().Spec
	fake := &fakeOfferOCIClient{}
	fake.getFn = func(context.Context, offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error) {
		return offersdk.GetOfferResponse{Offer: offerFromSpec(testOfferID, currentSpec, offersdk.OfferLifecycleStateActive)}, nil
	}
	client := newOfferRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "sellerCompartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want sellerCompartmentId drift rejection", err)
	}
	if response.IsSuccessful {
		t.Fatal("response.IsSuccessful = true, want false")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
}

func TestOfferDeleteRetainsFinalizerUntilReadbackConfirmsDeletion(t *testing.T) {
	t.Parallel()

	resource := newOfferRuntimeTestResource()
	trackOfferID(resource, testOfferID)
	fake := &fakeOfferOCIClient{}
	fake.getFn = func(context.Context, offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error) {
		return offersdk.GetOfferResponse{Offer: offerFromSpec(testOfferID, resource.Spec, offersdk.OfferLifecycleStateActive)}, nil
	}
	fake.deleteFn = func(context.Context, offersdk.DeleteOfferRequest) (offersdk.DeleteOfferResponse, error) {
		return offersdk.DeleteOfferResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	client := newOfferRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while readback is still ACTIVE")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil until confirmed", resource.Status.OsokStatus.DeletedAt)
	}
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status async current = %#v, want delete pending", current)
	}
}

func TestOfferDeleteResolvesMissingTrackedIDByPaginatedList(t *testing.T) {
	t.Parallel()

	resource := newOfferRuntimeTestResource()
	fake := newOfferDeleteResolveByListFake(t, resource)
	client := newOfferRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	assertOfferDeletePendingAfterListResolution(t, deleted, fake, resource)
}

func newOfferDeleteResolveByListFake(t *testing.T, resource *offerv1beta1.Offer) *fakeOfferOCIClient {
	t.Helper()
	fake := &fakeOfferOCIClient{}
	fake.listFn = func(_ context.Context, req offersdk.ListOffersRequest) (offersdk.ListOffersResponse, error) {
		return paginatedOfferBindResponse(t, fake, resource, req)
	}
	fake.getFn = func(context.Context, offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error) {
		return offersdk.GetOfferResponse{Offer: offerFromSpec(testOfferID, resource.Spec, offersdk.OfferLifecycleStateActive)}, nil
	}
	fake.deleteFn = func(_ context.Context, req offersdk.DeleteOfferRequest) (offersdk.DeleteOfferResponse, error) {
		requireOfferStringPtr(t, "delete offerId", req.OfferId, testOfferID)
		return offersdk.DeleteOfferResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	return fake
}

func assertOfferDeletePendingAfterListResolution(
	t *testing.T,
	deleted bool,
	fake *fakeOfferOCIClient,
	resource *offerv1beta1.Offer,
) {
	t.Helper()
	if deleted {
		t.Fatal("Delete() deleted = true, want false while readback is still ACTIVE")
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2", len(fake.listRequests))
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(fake.deleteRequests))
	}
	if len(fake.getRequests) != 2 {
		t.Fatalf("get requests = %d, want 2", len(fake.getRequests))
	}
	requireOfferStringPtr(t, "first get offerId", fake.getRequests[0].OfferId, testOfferID)
	if got := string(resource.Status.OsokStatus.Ocid); got != testOfferID {
		t.Fatalf("status.ocid = %q, want %q after list resolution", got, testOfferID)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil until confirmed", resource.Status.OsokStatus.DeletedAt)
	}
	if current := resource.Status.OsokStatus.Async.Current; current == nil || current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status async current = %#v, want delete pending", current)
	}
}

func TestOfferDeleteWithMissingTrackedIDAndNoListMatchReleasesFinalizer(t *testing.T) {
	t.Parallel()

	resource := newOfferRuntimeTestResource()
	fake := &fakeOfferOCIClient{}
	fake.listFn = func(_ context.Context, req offersdk.ListOffersRequest) (offersdk.ListOffersResponse, error) {
		requireOfferStringPtr(t, "list sellerCompartmentId", req.SellerCompartmentId, resource.Spec.SellerCompartmentId)
		requireOfferStringPtr(t, "list displayName", req.DisplayName, resource.Spec.DisplayName)
		return offersdk.ListOffersResponse{}, nil
	}
	fake.getFn = func(context.Context, offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error) {
		t.Fatal("GetOffer() called when list resolved no matching Offer")
		return offersdk.GetOfferResponse{}, nil
	}
	fake.deleteFn = func(context.Context, offersdk.DeleteOfferRequest) (offersdk.DeleteOfferResponse, error) {
		t.Fatal("DeleteOffer() called when list resolved no matching Offer")
		return offersdk.DeleteOfferResponse{}, nil
	}
	client := newOfferRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true when list confirms no matching Offer")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want deletion timestamp")
	}
}

func TestOfferDeleteReleasesFinalizerAfterNotFoundConfirmation(t *testing.T) {
	t.Parallel()

	resource := newOfferRuntimeTestResource()
	trackOfferID(resource, testOfferID)
	fake := &fakeOfferOCIClient{}
	fake.getFn = func(context.Context, offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error) {
		switch len(fake.getRequests) {
		case 1:
			return offersdk.GetOfferResponse{Offer: offerFromSpec(testOfferID, resource.Spec, offersdk.OfferLifecycleStateActive)}, nil
		case 2:
			return offersdk.GetOfferResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "offer deleted")
		default:
			t.Fatalf("unexpected get request count %d", len(fake.getRequests))
			return offersdk.GetOfferResponse{}, nil
		}
	}
	fake.deleteFn = func(context.Context, offersdk.DeleteOfferRequest) (offersdk.DeleteOfferResponse, error) {
		return offersdk.DeleteOfferResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	client := newOfferRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after not-found confirmation")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want deletion timestamp")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" && got != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want delete or confirm request ID", got)
	}
}

func TestOfferDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := newOfferRuntimeTestResource()
	trackOfferID(resource, testOfferID)
	getErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or missing")
	getErr.OpcRequestID = "opc-auth-read"
	fake := &fakeOfferOCIClient{}
	fake.getFn = func(context.Context, offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error) {
		return offersdk.GetOfferResponse{}, getErr
	}
	fake.deleteFn = func(context.Context, offersdk.DeleteOfferRequest) (offersdk.DeleteOfferResponse, error) {
		t.Fatal("DeleteOffer() called after ambiguous pre-delete confirmation read")
		return offersdk.DeleteOfferResponse{}, nil
	}
	client := newOfferRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative pre-delete confirm-read error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete confirm-read")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 after ambiguous pre-delete confirm-read", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-auth-read" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-auth-read", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestOfferDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := newOfferRuntimeTestResource()
	trackOfferID(resource, testOfferID)
	fake := &fakeOfferOCIClient{}
	fake.getFn = func(context.Context, offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error) {
		return offersdk.GetOfferResponse{Offer: offerFromSpec(testOfferID, resource.Spec, offersdk.OfferLifecycleStateActive)}, nil
	}
	fake.deleteFn = func(context.Context, offersdk.DeleteOfferRequest) (offersdk.DeleteOfferResponse, error) {
		return offersdk.DeleteOfferResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
	}
	client := newOfferRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped 404 error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for auth-shaped 404", resource.Status.OsokStatus.DeletedAt)
	}
}

func newOfferRuntimeTestClient(fake *fakeOfferOCIClient) OfferServiceClient {
	manager := &OfferServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}}
	hooks := newOfferRuntimeHooksWithOCIClient(fake)
	applyOfferRuntimeHooks(&hooks, fake, nil)
	delegate := defaultOfferServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*offerv1beta1.Offer](
			buildOfferGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapOfferGeneratedClient(hooks, delegate)
}

func newOfferRuntimeHooksWithOCIClient(client offerOCIClient) OfferRuntimeHooks {
	return OfferRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*offerv1beta1.Offer]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*offerv1beta1.Offer]{},
		StatusHooks:     generatedruntime.StatusHooks[*offerv1beta1.Offer]{},
		ParityHooks:     generatedruntime.ParityHooks[*offerv1beta1.Offer]{},
		Async:           generatedruntime.AsyncHooks[*offerv1beta1.Offer]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*offerv1beta1.Offer]{},
		Create: runtimeOperationHooks[offersdk.CreateOfferRequest, offersdk.CreateOfferResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateOfferDetails", RequestName: "CreateOfferDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request offersdk.CreateOfferRequest) (offersdk.CreateOfferResponse, error) {
				return client.CreateOffer(ctx, request)
			},
		},
		Get: runtimeOperationHooks[offersdk.GetOfferRequest, offersdk.GetOfferResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OfferId", RequestName: "offerId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request offersdk.GetOfferRequest) (offersdk.GetOfferResponse, error) {
				return client.GetOffer(ctx, request)
			},
		},
		List: runtimeOperationHooks[offersdk.ListOffersRequest, offersdk.ListOffersResponse]{
			Fields: offerListFields(),
			Call: func(ctx context.Context, request offersdk.ListOffersRequest) (offersdk.ListOffersResponse, error) {
				return client.ListOffers(ctx, request)
			},
		},
		Update: runtimeOperationHooks[offersdk.UpdateOfferRequest, offersdk.UpdateOfferResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OfferId", RequestName: "offerId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateOfferDetails", RequestName: "UpdateOfferDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request offersdk.UpdateOfferRequest) (offersdk.UpdateOfferResponse, error) {
				return client.UpdateOffer(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[offersdk.DeleteOfferRequest, offersdk.DeleteOfferResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OfferId", RequestName: "offerId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request offersdk.DeleteOfferRequest) (offersdk.DeleteOfferResponse, error) {
				return client.DeleteOffer(ctx, request)
			},
		},
	}
}

func newOfferRuntimeTestResource() *offerv1beta1.Offer {
	return &offerv1beta1.Offer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "offer-runtime",
			Namespace: "default",
		},
		Spec: offerv1beta1.OfferSpec{
			DisplayName:         testOfferDisplayName,
			SellerCompartmentId: testOfferSeller,
			BuyerCompartmentId:  testOfferBuyer,
			Description:         "runtime offer",
		},
	}
}

func offerFromSpec(id string, spec offerv1beta1.OfferSpec, state offersdk.OfferLifecycleStateEnum) offersdk.Offer {
	now := common.SDKTime{Time: time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)}
	return offersdk.Offer{
		Id:                  common.String(id),
		DisplayName:         common.String(spec.DisplayName),
		SellerCompartmentId: common.String(spec.SellerCompartmentId),
		TimeCreated:         &now,
		LifecycleState:      state,
		FreeformTags:        mapsClone(spec.FreeformTags),
		DefinedTags:         offerDefinedTagsFromSpec(spec.DefinedTags),
		BuyerCompartmentId:  common.String(spec.BuyerCompartmentId),
		Description:         common.String(spec.Description),
		Pricing:             offerPricing(spec.Pricing),
		BuyerInformation:    offerBuyerInformation(spec.BuyerInformation),
		SellerInformation:   offerSellerInformation(spec.SellerInformation),
		ResourceBundles:     offerResourceBundles(spec.ResourceBundles),
	}
}

func offerSummaryFromSpec(id string, spec offerv1beta1.OfferSpec, displayName string, state offersdk.OfferLifecycleStateEnum) offersdk.OfferSummary {
	now := common.SDKTime{Time: time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)}
	return offersdk.OfferSummary{
		Id:                  common.String(id),
		DisplayName:         common.String(displayName),
		BuyerCompartmentId:  common.String(spec.BuyerCompartmentId),
		SellerCompartmentId: common.String(spec.SellerCompartmentId),
		TimeCreated:         &now,
		LifecycleState:      state,
		FreeformTags:        mapsClone(spec.FreeformTags),
		DefinedTags:         offerDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func mapsClone(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func trackOfferID(resource *offerv1beta1.Offer, id string) {
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	resource.Status.Id = id
	resource.Status.DisplayName = resource.Spec.DisplayName
	resource.Status.SellerCompartmentId = resource.Spec.SellerCompartmentId
	resource.Status.BuyerCompartmentId = resource.Spec.BuyerCompartmentId
}

func assertOfferStatusIDAndRequestID(t *testing.T, resource *offerv1beta1.Offer, wantID string, wantRequestID string) {
	t.Helper()
	if got := resource.Status.Id; got != wantID {
		t.Fatalf("status.id = %q, want %q", got, wantID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != wantID {
		t.Fatalf("status.ocid = %q, want %q", got, wantID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != wantRequestID {
		t.Fatalf("status.opcRequestId = %q, want %q", got, wantRequestID)
	}
}

func requireOfferStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireOfferStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

var _ offerOCIClient = (*fakeOfferOCIClient)(nil)
