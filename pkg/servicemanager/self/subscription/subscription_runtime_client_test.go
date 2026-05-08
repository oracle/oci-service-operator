/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package subscription

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	selfsdk "github.com/oracle/oci-go-sdk/v65/self"
	selfv1beta1 "github.com/oracle/oci-service-operator/api/self/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testSubscriptionID          = "ocid1.selfsubscription.oc1..example"
	testSubscriptionOtherID     = "ocid1.selfsubscription.oc1..other"
	testSubscriptionCompartment = "ocid1.compartment.oc1..example"
	testSubscriptionTenant      = "ocid1.tenancy.oc1..example"
	testSubscriptionSeller      = "ocid1.seller.oc1..example"
	testSubscriptionProduct     = "ocid1.product.oc1..example"
	testSubscriptionDisplayName = "self-subscription"
	testSubscriptionWorkRequest = "wr-self-subscription"
)

type fakeSubscriptionOCIClient struct {
	createFn      func(context.Context, selfsdk.CreateSubscriptionRequest) (selfsdk.CreateSubscriptionResponse, error)
	getFn         func(context.Context, selfsdk.GetSubscriptionRequest) (selfsdk.GetSubscriptionResponse, error)
	listFn        func(context.Context, selfsdk.ListSubscriptionsRequest) (selfsdk.ListSubscriptionsResponse, error)
	updateFn      func(context.Context, selfsdk.UpdateSubscriptionRequest) (selfsdk.UpdateSubscriptionResponse, error)
	deleteFn      func(context.Context, selfsdk.DeleteSubscriptionRequest) (selfsdk.DeleteSubscriptionResponse, error)
	workRequestFn func(context.Context, selfsdk.GetWorkRequestRequest) (selfsdk.GetWorkRequestResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int
	workCalls   int
}

func (f *fakeSubscriptionOCIClient) CreateSubscription(ctx context.Context, request selfsdk.CreateSubscriptionRequest) (selfsdk.CreateSubscriptionResponse, error) {
	f.createCalls++
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return selfsdk.CreateSubscriptionResponse{}, nil
}

func (f *fakeSubscriptionOCIClient) GetSubscription(ctx context.Context, request selfsdk.GetSubscriptionRequest) (selfsdk.GetSubscriptionResponse, error) {
	f.getCalls++
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return selfsdk.GetSubscriptionResponse{}, nil
}

func (f *fakeSubscriptionOCIClient) ListSubscriptions(ctx context.Context, request selfsdk.ListSubscriptionsRequest) (selfsdk.ListSubscriptionsResponse, error) {
	f.listCalls++
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return selfsdk.ListSubscriptionsResponse{}, nil
}

func (f *fakeSubscriptionOCIClient) UpdateSubscription(ctx context.Context, request selfsdk.UpdateSubscriptionRequest) (selfsdk.UpdateSubscriptionResponse, error) {
	f.updateCalls++
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return selfsdk.UpdateSubscriptionResponse{}, nil
}

func (f *fakeSubscriptionOCIClient) DeleteSubscription(ctx context.Context, request selfsdk.DeleteSubscriptionRequest) (selfsdk.DeleteSubscriptionResponse, error) {
	f.deleteCalls++
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return selfsdk.DeleteSubscriptionResponse{}, nil
}

func (f *fakeSubscriptionOCIClient) GetWorkRequest(ctx context.Context, request selfsdk.GetWorkRequestRequest) (selfsdk.GetWorkRequestResponse, error) {
	f.workCalls++
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, request)
	}
	return selfsdk.GetWorkRequestResponse{}, nil
}

func testSubscriptionClient(fake *fakeSubscriptionOCIClient) SubscriptionServiceClient {
	if fake == nil {
		fake = &fakeSubscriptionOCIClient{}
	}
	return newSubscriptionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeSubscriptionResource() *selfv1beta1.Subscription {
	return &selfv1beta1.Subscription{
		Spec: selfv1beta1.SubscriptionSpec{
			CompartmentId: testSubscriptionCompartment,
			TenantId:      testSubscriptionTenant,
			SellerId:      testSubscriptionSeller,
			ProductId:     testSubscriptionProduct,
			DisplayName:   testSubscriptionDisplayName,
			SourceType:    "OCI_NATIVE",
			Realm:         "OC1",
			Region:        "us-chicago-1",
			AdditionalDetails: []selfv1beta1.SubscriptionAdditionalDetail{
				{Key: "contract", Value: "gold"},
			},
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
			SubscriptionDetails: selfv1beta1.SubscriptionDetails{
				PartnerRegistrationUrl: "https://partner.example.com/register",
				PricingPlan: selfv1beta1.SubscriptionDetailsPricingPlan{
					PlanType:         "FIXED",
					PlanName:         "gold-plan",
					BillingFrequency: "YEARLY",
					Rates: []selfv1beta1.SubscriptionDetailsPricingPlanRate{
						{Currency: "USD", Rate: 42.5},
					},
				},
				BillingDetails: selfv1beta1.SubscriptionDetailsBillingDetails{
					Sku:            "gold-sku",
					MetricType:     "OCPU_HOURS",
					RateAllocation: 100,
					Meters: []selfv1beta1.SubscriptionDetailsBillingDetailsMeter{
						{Name: "meter-a", RateAllocation: 100},
					},
				},
			},
		},
	}
}

func makeSDKSubscription(id string, lifecycleState selfsdk.LifecycleStateEnumEnum, lifecycleDetails selfsdk.LifecycleDetailsEnumEnum) selfsdk.Subscription {
	return selfsdk.Subscription{
		Id:                common.String(id),
		DisplayName:       common.String(testSubscriptionDisplayName),
		CompartmentId:     common.String(testSubscriptionCompartment),
		TenantId:          common.String(testSubscriptionTenant),
		SellerId:          common.String(testSubscriptionSeller),
		ProductId:         common.String(testSubscriptionProduct),
		LifecycleState:    lifecycleState,
		LifecycleDetails:  lifecycleDetails,
		FreeformTags:      map[string]string{"env": "dev"},
		DefinedTags:       map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		AdditionalDetails: []selfsdk.ExtendedMetadata{{Key: common.String("contract"), Value: common.String("gold")}},
		Realm:             common.String("OC1"),
		Region:            common.String("us-chicago-1"),
		SourceType:        selfsdk.SourceTypeOciNative,
		SubscriptionDetails: &selfsdk.SubscriptionDetails{
			PartnerRegistrationUrl: common.String("https://partner.example.com/register"),
			PricingPlan: &selfsdk.PricingPlan{
				PlanType:         selfsdk.PricingPlanPlanTypeFixed,
				PlanName:         common.String("gold-plan"),
				BillingFrequency: selfsdk.PricingPlanBillingFrequencyYearly,
				Rates: []selfsdk.PricingRate{
					{Currency: common.String("USD"), Rate: common.Float32(42.5)},
				},
			},
			BillingDetails: &selfsdk.BillingDetails{
				Sku:            common.String("gold-sku"),
				MetricType:     selfsdk.MetricTypeOcpuHours,
				RateAllocation: common.Float32(100),
				Meters: []selfsdk.Meter{
					{Name: common.String("meter-a"), RateAllocation: common.Float32(100)},
				},
			},
		},
	}
}

func makeSDKSubscriptionSummary(id string, lifecycleState selfsdk.LifecycleStateEnumEnum, lifecycleDetails selfsdk.LifecycleDetailsEnumEnum) selfsdk.SubscriptionSummary {
	return selfsdk.SubscriptionSummary{
		Id:               common.String(id),
		DisplayName:      common.String(testSubscriptionDisplayName),
		CompartmentId:    common.String(testSubscriptionCompartment),
		TenantId:         common.String(testSubscriptionTenant),
		SellerId:         common.String(testSubscriptionSeller),
		ProductId:        common.String(testSubscriptionProduct),
		LifecycleState:   lifecycleState,
		LifecycleDetails: lifecycleDetails,
		FreeformTags:     map[string]string{"env": "dev"},
		DefinedTags:      map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func makeSubscriptionWorkRequest(
	id string,
	status selfsdk.OperationStatusEnum,
	operation selfsdk.OperationTypeEnum,
	action selfsdk.ActionTypeEnum,
	subscriptionID string,
) selfsdk.WorkRequest {
	percentComplete := float32(42)
	workRequest := selfsdk.WorkRequest{
		Id:              common.String(id),
		Status:          status,
		OperationType:   operation,
		CompartmentId:   common.String(testSubscriptionCompartment),
		PercentComplete: &percentComplete,
	}
	if subscriptionID != "" {
		workRequest.Resources = []selfsdk.WorkRequestResource{
			{
				EntityType: common.String("subscription"),
				ActionType: action,
				Identifier: common.String(subscriptionID),
				EntityUri:  common.String("/subscriptions/" + subscriptionID),
			},
		}
	}
	return workRequest
}

func requireAsyncCurrent(
	t *testing.T,
	resource *selfv1beta1.Subscription,
	source shared.OSOKAsyncSource,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want populated async tracker")
	}
	if current.Source != source {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, source)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
}

func TestSubscriptionRuntimeHooksUseReviewedSemantics(t *testing.T) {
	t.Parallel()

	hooks := SubscriptionRuntimeHooks{}
	applySubscriptionRuntimeHooks(&hooks, &fakeSubscriptionOCIClient{}, nil)

	if hooks.Semantics == nil || hooks.Semantics.Async == nil {
		t.Fatal("hooks.Semantics.Async = nil, want reviewed workrequest semantics")
	}
	if hooks.Semantics.Async.Strategy != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", hooks.Semantics.Async.Strategy)
	}
	if hooks.Semantics.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async.Runtime = %q, want generatedruntime", hooks.Semantics.Async.Runtime)
	}
	if hooks.Semantics.Async.WorkRequest == nil {
		t.Fatal("Async.WorkRequest = nil")
	}
	if !jsonEqual(hooks.Semantics.Async.WorkRequest.Phases, []string{"create", "update", "delete"}) {
		t.Fatalf("Async.WorkRequest.Phases = %#v", hooks.Semantics.Async.WorkRequest.Phases)
	}
	if !jsonEqual(hooks.Semantics.List.MatchFields, []string{"compartmentId", "displayName", "id"}) {
		t.Fatalf("List.MatchFields = %#v", hooks.Semantics.List.MatchFields)
	}
	if !jsonEqual(hooks.Semantics.Mutation.Mutable, []string{"displayName", "freeformTags", "definedTags"}) {
		t.Fatalf("Mutation.Mutable = %#v", hooks.Semantics.Mutation.Mutable)
	}
	if !jsonEqual(hooks.Semantics.Mutation.ForceNew, []string{
		"compartmentId", "tenantId", "subscriptionDetails", "sellerId", "productId", "sourceType", "additionalDetails", "realm", "region",
	}) {
		t.Fatalf("Mutation.ForceNew = %#v", hooks.Semantics.Mutation.ForceNew)
	}
	if hooks.BuildCreateBody == nil || hooks.BuildUpdateBody == nil {
		t.Fatal("BuildCreateBody/BuildUpdateBody should be wired")
	}
	if hooks.Identity.Resolve == nil || hooks.Identity.GuardExistingBeforeCreate == nil || hooks.Identity.LookupExisting == nil {
		t.Fatalf("identity hooks = %#v, want reviewed existing-before-create guard", hooks.Identity)
	}
	if hooks.Async.GetWorkRequest == nil || hooks.Async.ResolveAction == nil || hooks.Async.ResolvePhase == nil || hooks.Async.RecoverResourceID == nil {
		t.Fatalf("async hooks = %#v, want reviewed workrequest helpers", hooks.Async)
	}
}

func TestSubscriptionCreateOrUpdateCreatesAndTracksWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	pending := true
	var createRequest selfsdk.CreateSubscriptionRequest
	fake := &fakeSubscriptionOCIClient{
		createFn: func(_ context.Context, request selfsdk.CreateSubscriptionRequest) (selfsdk.CreateSubscriptionResponse, error) {
			createRequest = request
			return selfsdk.CreateSubscriptionResponse{
				Subscription:     makeSDKSubscription(testSubscriptionID, selfsdk.LifecycleStateEnumInactive, selfsdk.LifecycleDetailsEnumPendingActivation),
				OpcWorkRequestId: common.String(testSubscriptionWorkRequest),
				OpcRequestId:     common.String("opc-create-1"),
			}, nil
		},
		listFn: func(_ context.Context, request selfsdk.ListSubscriptionsRequest) (selfsdk.ListSubscriptionsResponse, error) {
			if got := stringValue(request.CompartmentId); got != testSubscriptionCompartment {
				t.Fatalf("list compartmentId = %q, want %q", got, testSubscriptionCompartment)
			}
			if got := stringValue(request.DisplayName); got != testSubscriptionDisplayName {
				t.Fatalf("list displayName = %q, want %q", got, testSubscriptionDisplayName)
			}
			return selfsdk.ListSubscriptionsResponse{}, nil
		},
		workRequestFn: func(_ context.Context, request selfsdk.GetWorkRequestRequest) (selfsdk.GetWorkRequestResponse, error) {
			if got := stringValue(request.WorkRequestId); got != testSubscriptionWorkRequest {
				t.Fatalf("workRequestId = %q, want %q", got, testSubscriptionWorkRequest)
			}
			if pending {
				return selfsdk.GetWorkRequestResponse{
					WorkRequest: makeSubscriptionWorkRequest(
						testSubscriptionWorkRequest,
						selfsdk.OperationStatusInProgress,
						selfsdk.OperationTypeCreateSubscription,
						selfsdk.ActionTypeInProgress,
						testSubscriptionID,
					),
				}, nil
			}
			return selfsdk.GetWorkRequestResponse{
				WorkRequest: makeSubscriptionWorkRequest(
					testSubscriptionWorkRequest,
					selfsdk.OperationStatusSucceeded,
					selfsdk.OperationTypeCreateSubscription,
					selfsdk.ActionTypeCreated,
					testSubscriptionID,
				),
			}, nil
		},
		getFn: func(_ context.Context, request selfsdk.GetSubscriptionRequest) (selfsdk.GetSubscriptionResponse, error) {
			if got := stringValue(request.SubscriptionId); got != testSubscriptionID {
				t.Fatalf("get subscriptionId = %q, want %q", got, testSubscriptionID)
			}
			return selfsdk.GetSubscriptionResponse{
				Subscription: makeSDKSubscription(testSubscriptionID, selfsdk.LifecycleStateEnumActive, selfsdk.LifecycleDetailsEnumActive),
			}, nil
		},
	}

	response, err := testSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful pending requeue", response)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncSourceWorkRequest, shared.OSOKAsyncPhaseCreate, testSubscriptionWorkRequest, shared.OSOKAsyncClassPending)
	if got := stringValue(createRequest.CreateSubscriptionDetails.CompartmentId); got != testSubscriptionCompartment {
		t.Fatalf("create compartmentId = %q, want %q", got, testSubscriptionCompartment)
	}
	if got := stringValue(createRequest.CreateSubscriptionDetails.TenantId); got != testSubscriptionTenant {
		t.Fatalf("create tenantId = %q, want %q", got, testSubscriptionTenant)
	}
	if got := stringValue(createRequest.CreateSubscriptionDetails.DisplayName); got != testSubscriptionDisplayName {
		t.Fatalf("create displayName = %q, want %q", got, testSubscriptionDisplayName)
	}
	if got := stringValue(createRequest.CreateSubscriptionDetails.Region); got != "us-chicago-1" {
		t.Fatalf("create region = %q, want us-chicago-1", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-1", got)
	}

	pending = false
	response, err = testSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() after work request success error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response after work request success = %#v, want active success", response)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker", resource.Status.OsokStatus.Async.Current)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testSubscriptionID {
		t.Fatalf("status.ocid = %q, want %q", got, testSubscriptionID)
	}
	if got := resource.Status.Id; got != testSubscriptionID {
		t.Fatalf("status.id = %q, want %q", got, testSubscriptionID)
	}
	if got := resource.Status.LifecycleState; got != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestSubscriptionCreateOrUpdateSkipsExistingLookupWithoutDisplayName(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	resource.Spec.DisplayName = ""
	fake := &fakeSubscriptionOCIClient{
		createFn: func(_ context.Context, request selfsdk.CreateSubscriptionRequest) (selfsdk.CreateSubscriptionResponse, error) {
			return selfsdk.CreateSubscriptionResponse{
				Subscription:     makeSDKSubscription(testSubscriptionID, selfsdk.LifecycleStateEnumInactive, selfsdk.LifecycleDetailsEnumPendingActivation),
				OpcWorkRequestId: common.String(testSubscriptionWorkRequest),
			}, nil
		},
		workRequestFn: func(_ context.Context, _ selfsdk.GetWorkRequestRequest) (selfsdk.GetWorkRequestResponse, error) {
			return selfsdk.GetWorkRequestResponse{
				WorkRequest: makeSubscriptionWorkRequest(
					testSubscriptionWorkRequest,
					selfsdk.OperationStatusSucceeded,
					selfsdk.OperationTypeCreateSubscription,
					selfsdk.ActionTypeCreated,
					testSubscriptionID,
				),
			}, nil
		},
		getFn: func(_ context.Context, _ selfsdk.GetSubscriptionRequest) (selfsdk.GetSubscriptionResponse, error) {
			return selfsdk.GetSubscriptionResponse{
				Subscription: makeSDKSubscription(testSubscriptionID, selfsdk.LifecycleStateEnumActive, selfsdk.LifecycleDetailsEnumActive),
			}, nil
		},
	}

	response, err := testSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active success", response)
	}
	if fake.listCalls != 0 {
		t.Fatalf("ListSubscriptions() calls = %d, want 0 when displayName is empty", fake.listCalls)
	}
	if fake.createCalls != 1 {
		t.Fatalf("CreateSubscription() calls = %d, want 1", fake.createCalls)
	}
}

func TestSubscriptionCreateOrUpdateBindsUniqueExistingCandidate(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	fake := &fakeSubscriptionOCIClient{
		listFn: func(_ context.Context, request selfsdk.ListSubscriptionsRequest) (selfsdk.ListSubscriptionsResponse, error) {
			if got := stringValue(request.DisplayName); got != testSubscriptionDisplayName {
				t.Fatalf("list displayName = %q, want %q", got, testSubscriptionDisplayName)
			}
			return selfsdk.ListSubscriptionsResponse{
				SubscriptionCollection: selfsdk.SubscriptionCollection{
					Items: []selfsdk.SubscriptionSummary{
						makeSDKSubscriptionSummary(testSubscriptionID, selfsdk.LifecycleStateEnumActive, selfsdk.LifecycleDetailsEnumActive),
						makeSDKSubscriptionSummary(testSubscriptionOtherID, selfsdk.LifecycleStateEnumFailed, selfsdk.LifecycleDetailsEnumFailed),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, request selfsdk.GetSubscriptionRequest) (selfsdk.GetSubscriptionResponse, error) {
			if got := stringValue(request.SubscriptionId); got != testSubscriptionID {
				t.Fatalf("get subscriptionId = %q, want %q", got, testSubscriptionID)
			}
			return selfsdk.GetSubscriptionResponse{
				Subscription: makeSDKSubscription(testSubscriptionID, selfsdk.LifecycleStateEnumActive, selfsdk.LifecycleDetailsEnumActive),
			}, nil
		},
		createFn: func(context.Context, selfsdk.CreateSubscriptionRequest) (selfsdk.CreateSubscriptionResponse, error) {
			t.Fatal("CreateSubscription() should not be called when a unique existing subscription matches")
			return selfsdk.CreateSubscriptionResponse{}, nil
		},
	}

	response, err := testSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active bound success", response)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testSubscriptionID {
		t.Fatalf("status.ocid = %q, want %q", got, testSubscriptionID)
	}
	if fake.createCalls != 0 {
		t.Fatalf("CreateSubscription() calls = %d, want 0", fake.createCalls)
	}
}

func TestSubscriptionLifecycleDetailsDrivePendingRequeue(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSubscriptionID)
	fake := &fakeSubscriptionOCIClient{
		getFn: func(_ context.Context, request selfsdk.GetSubscriptionRequest) (selfsdk.GetSubscriptionResponse, error) {
			if got := stringValue(request.SubscriptionId); got != testSubscriptionID {
				t.Fatalf("get subscriptionId = %q, want %q", got, testSubscriptionID)
			}
			return selfsdk.GetSubscriptionResponse{
				Subscription: makeSDKSubscription(testSubscriptionID, selfsdk.LifecycleStateEnumInactive, selfsdk.LifecycleDetailsEnumPendingActivation),
			}, nil
		},
	}

	response, err := testSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful pending requeue", response)
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncSourceLifecycle, shared.OSOKAsyncPhaseCreate, "", shared.OSOKAsyncClassPending)
	if got := resource.Status.LifecycleState; got != "PENDING_ACTIVATION" {
		t.Fatalf("status.lifecycleState = %q, want normalized pending lifecycle detail", got)
	}
	if got := resource.Status.LifecycleDetails; got != "PENDING_ACTIVATION" {
		t.Fatalf("status.lifecycleDetails = %q, want PENDING_ACTIVATION", got)
	}
}

func TestSubscriptionDeleteWaitsForDeleteLifecycleConfirmation(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSubscriptionID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   testSubscriptionWorkRequest,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	fake := &fakeSubscriptionOCIClient{
		workRequestFn: func(_ context.Context, request selfsdk.GetWorkRequestRequest) (selfsdk.GetWorkRequestResponse, error) {
			if got := stringValue(request.WorkRequestId); got != testSubscriptionWorkRequest {
				t.Fatalf("workRequestId = %q, want %q", got, testSubscriptionWorkRequest)
			}
			return selfsdk.GetWorkRequestResponse{
				WorkRequest: makeSubscriptionWorkRequest(
					testSubscriptionWorkRequest,
					selfsdk.OperationStatusSucceeded,
					selfsdk.OperationTypeDeleteSubscription,
					selfsdk.ActionTypeDeleted,
					testSubscriptionID,
				),
			}, nil
		},
		getFn: func(_ context.Context, request selfsdk.GetSubscriptionRequest) (selfsdk.GetSubscriptionResponse, error) {
			if got := stringValue(request.SubscriptionId); got != testSubscriptionID {
				t.Fatalf("get subscriptionId = %q, want %q", got, testSubscriptionID)
			}
			return selfsdk.GetSubscriptionResponse{
				Subscription: makeSDKSubscription(testSubscriptionID, selfsdk.LifecycleStateEnumInactive, selfsdk.LifecycleDetailsEnumDeleting),
			}, nil
		},
	}

	deleted, err := testSubscriptionClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = deleted, want false while lifecycleDetails is DELETING")
	}
	requireAsyncCurrent(t, resource, shared.OSOKAsyncSourceLifecycle, shared.OSOKAsyncPhaseDelete, testSubscriptionWorkRequest, shared.OSOKAsyncClassPending)
	if got := resource.Status.LifecycleState; got != "DELETING" {
		t.Fatalf("status.lifecycleState = %q, want normalized delete lifecycle detail", got)
	}
}
