/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package subscription

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	onssdk "github.com/oracle/oci-go-sdk/v65/ons"
	onsv1beta1 "github.com/oracle/oci-service-operator/api/ons/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testSubscriptionID          = "ocid1.onssubscription.oc1..example"
	testSubscriptionOtherID     = "ocid1.onssubscription.oc1..other"
	testSubscriptionCompartment = "ocid1.compartment.oc1..example"
	testSubscriptionTopicID     = "ocid1.onstopic.oc1..example"
	testSubscriptionEndpoint    = "ops@example.com"
	testSubscriptionProtocol    = "EMAIL"
)

type fakeSubscriptionOCIClient struct {
	createFn func(context.Context, onssdk.CreateSubscriptionRequest) (onssdk.CreateSubscriptionResponse, error)
	getFn    func(context.Context, onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error)
	listFn   func(context.Context, onssdk.ListSubscriptionsRequest) (onssdk.ListSubscriptionsResponse, error)
	updateFn func(context.Context, onssdk.UpdateSubscriptionRequest) (onssdk.UpdateSubscriptionResponse, error)
	deleteFn func(context.Context, onssdk.DeleteSubscriptionRequest) (onssdk.DeleteSubscriptionResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int
}

func (f *fakeSubscriptionOCIClient) CreateSubscription(
	ctx context.Context,
	request onssdk.CreateSubscriptionRequest,
) (onssdk.CreateSubscriptionResponse, error) {
	f.createCalls++
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return onssdk.CreateSubscriptionResponse{}, nil
}

func (f *fakeSubscriptionOCIClient) GetSubscription(
	ctx context.Context,
	request onssdk.GetSubscriptionRequest,
) (onssdk.GetSubscriptionResponse, error) {
	f.getCalls++
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return onssdk.GetSubscriptionResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "subscription missing")
}

func (f *fakeSubscriptionOCIClient) ListSubscriptions(
	ctx context.Context,
	request onssdk.ListSubscriptionsRequest,
) (onssdk.ListSubscriptionsResponse, error) {
	f.listCalls++
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return onssdk.ListSubscriptionsResponse{}, nil
}

func (f *fakeSubscriptionOCIClient) UpdateSubscription(
	ctx context.Context,
	request onssdk.UpdateSubscriptionRequest,
) (onssdk.UpdateSubscriptionResponse, error) {
	f.updateCalls++
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return onssdk.UpdateSubscriptionResponse{}, nil
}

func (f *fakeSubscriptionOCIClient) DeleteSubscription(
	ctx context.Context,
	request onssdk.DeleteSubscriptionRequest,
) (onssdk.DeleteSubscriptionResponse, error) {
	f.deleteCalls++
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return onssdk.DeleteSubscriptionResponse{}, nil
}

func testSubscriptionClient(fake *fakeSubscriptionOCIClient) SubscriptionServiceClient {
	if fake == nil {
		fake = &fakeSubscriptionOCIClient{}
	}
	return newSubscriptionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func reconcileRequest(resource *onsv1beta1.Subscription) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func makeSubscriptionResource() *onsv1beta1.Subscription {
	return &onsv1beta1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "subscription-sample",
			Namespace: "default",
			UID:       types.UID("subscription-uid"),
		},
		Spec: onsv1beta1.SubscriptionSpec{
			TopicId:       testSubscriptionTopicID,
			CompartmentId: testSubscriptionCompartment,
			Protocol:      testSubscriptionProtocol,
			Endpoint:      testSubscriptionEndpoint,
			Metadata:      "owner=ops",
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
			DeliveryPolicy: onsv1beta1.SubscriptionDeliveryPolicy{
				BackoffRetryPolicy: onsv1beta1.SubscriptionDeliveryPolicyBackoffRetryPolicy{
					MaxRetryDuration: 7200000,
					PolicyType:       string(onssdk.BackoffRetryPolicyPolicyTypeExponential),
				},
			},
		},
	}
}

func makeSDKSubscription(id string, state onssdk.SubscriptionLifecycleStateEnum) onssdk.Subscription {
	return onssdk.Subscription{
		Id:             common.String(id),
		TopicId:        common.String(testSubscriptionTopicID),
		CompartmentId:  common.String(testSubscriptionCompartment),
		Protocol:       common.String(testSubscriptionProtocol),
		Endpoint:       common.String(testSubscriptionEndpoint),
		LifecycleState: state,
		CreatedTime:    common.Int64(123),
		DeliverPolicy:  common.String(`{"backoffRetryPolicy":{"maxRetryDuration":7200000,"policyType":"EXPONENTIAL"}}`),
		Etag:           common.String("etag-1"),
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func makeSDKSubscriptionSummary(id string, endpoint string, state onssdk.SubscriptionSummaryLifecycleStateEnum) onssdk.SubscriptionSummary {
	return onssdk.SubscriptionSummary{
		Id:             common.String(id),
		TopicId:        common.String(testSubscriptionTopicID),
		CompartmentId:  common.String(testSubscriptionCompartment),
		Protocol:       common.String(testSubscriptionProtocol),
		Endpoint:       common.String(endpoint),
		LifecycleState: state,
		CreatedTime:    common.Int64(123),
		DeliveryPolicy: &onssdk.DeliveryPolicy{
			BackoffRetryPolicy: &onssdk.BackoffRetryPolicy{
				MaxRetryDuration: common.Int(7200000),
				PolicyType:       onssdk.BackoffRetryPolicyPolicyTypeExponential,
			},
		},
		Etag:         common.String("etag-1"),
		FreeformTags: map[string]string{"env": "dev"},
		DefinedTags:  map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func TestSubscriptionRuntimeSemantics(t *testing.T) {
	t.Parallel()

	hooks := SubscriptionRuntimeHooks{}
	applySubscriptionRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed Subscription semantics")
	}
	if hooks.Semantics.Async == nil || hooks.Semantics.Async.Strategy != "lifecycle" {
		t.Fatalf("Async = %#v, want lifecycle semantics", hooks.Semantics.Async)
	}
	if hooks.Semantics.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", hooks.Semantics.FinalizerPolicy)
	}
	if hooks.Semantics.Delete.Policy != "required" || hooks.Semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp=%#v, want required confirm-delete", hooks.Semantics.Delete, hooks.Semantics.DeleteFollowUp)
	}
	assertStringSliceContainsAll(t, "List.MatchFields", hooks.Semantics.List.MatchFields, "compartmentId", "topicId", "protocol", "endpoint", "id")
	assertStringSliceContainsAll(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "deliveryPolicy", "freeformTags", "definedTags")
	assertStringSliceContainsAll(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "topicId", "compartmentId", "protocol", "endpoint", "metadata")
	if hooks.BuildUpdateBody == nil {
		t.Fatal("BuildUpdateBody = nil, want Subscription-specific mutable update body")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("DeleteHooks.HandleError = nil, want auth-shaped not-found guard")
	}
	if hooks.DeleteHooks.ApplyOutcome == nil {
		t.Fatal("DeleteHooks.ApplyOutcome = nil, want finalizer-safe delete pending handling")
	}
}

func TestSubscriptionCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	var createRequest onssdk.CreateSubscriptionRequest
	var listRequest onssdk.ListSubscriptionsRequest
	var getRequest onssdk.GetSubscriptionRequest
	fake := &fakeSubscriptionOCIClient{
		listFn: func(_ context.Context, request onssdk.ListSubscriptionsRequest) (onssdk.ListSubscriptionsResponse, error) {
			listRequest = request
			return onssdk.ListSubscriptionsResponse{}, nil
		},
		createFn: func(_ context.Context, request onssdk.CreateSubscriptionRequest) (onssdk.CreateSubscriptionResponse, error) {
			createRequest = request
			return onssdk.CreateSubscriptionResponse{
				Subscription: makeSDKSubscription(testSubscriptionID, onssdk.SubscriptionLifecycleStateActive),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error) {
			getRequest = request
			return onssdk.GetSubscriptionResponse{
				Subscription: makeSDKSubscription(testSubscriptionID, onssdk.SubscriptionLifecycleStateActive),
			}, nil
		},
	}

	response, err := testSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful active create", response)
	}
	requireStringPtr(t, "list compartmentId", listRequest.CompartmentId, testSubscriptionCompartment)
	requireStringPtr(t, "list topicId", listRequest.TopicId, testSubscriptionTopicID)
	requireStringPtr(t, "create topicId", createRequest.TopicId, testSubscriptionTopicID)
	requireStringPtr(t, "create compartmentId", createRequest.CompartmentId, testSubscriptionCompartment)
	requireStringPtr(t, "create protocol", createRequest.Protocol, testSubscriptionProtocol)
	requireStringPtr(t, "create endpoint", createRequest.Endpoint, testSubscriptionEndpoint)
	requireStringPtr(t, "create metadata", createRequest.Metadata, "owner=ops")
	if got, want := createRequest.FreeformTags, map[string]string{"env": "dev"}; !jsonEqual(got, want) {
		t.Fatalf("create freeformTags = %#v, want %#v", got, want)
	}
	if got, want := createRequest.DefinedTags["Operations"]["CostCenter"], "42"; got != want {
		t.Fatalf("create definedTags Operations.CostCenter = %#v, want %q", got, want)
	}
	requireStringPtr(t, "get subscriptionId", getRequest.SubscriptionId, testSubscriptionID)
	if got := string(resource.Status.OsokStatus.Ocid); got != testSubscriptionID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testSubscriptionID)
	}
	if got := resource.Status.Id; got != testSubscriptionID {
		t.Fatalf("status.id = %q, want %q", got, testSubscriptionID)
	}
	if got := resource.Status.LifecycleState; got != string(onssdk.SubscriptionLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	assertSubscriptionMetadataFingerprint(t, resource, "owner=ops")
	assertTrailingCondition(t, resource, shared.Active)
}

func TestSubscriptionCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	fake := &fakeSubscriptionOCIClient{}
	fake.listFn = func(_ context.Context, request onssdk.ListSubscriptionsRequest) (onssdk.ListSubscriptionsResponse, error) {
		switch fake.listCalls {
		case 1:
			if request.Page != nil {
				t.Fatalf("first ListSubscriptions page = %q, want nil", *request.Page)
			}
			return onssdk.ListSubscriptionsResponse{
				Items: []onssdk.SubscriptionSummary{
					makeSDKSubscriptionSummary(testSubscriptionOtherID, "other@example.com", onssdk.SubscriptionSummaryLifecycleStateActive),
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case 2:
			requireStringPtr(t, "second ListSubscriptions page", request.Page, "page-2")
			return onssdk.ListSubscriptionsResponse{
				Items: []onssdk.SubscriptionSummary{
					makeSDKSubscriptionSummary(testSubscriptionID, testSubscriptionEndpoint, onssdk.SubscriptionSummaryLifecycleStateActive),
				},
			}, nil
		default:
			t.Fatalf("unexpected ListSubscriptions call %d", fake.listCalls)
			return onssdk.ListSubscriptionsResponse{}, nil
		}
	}
	fake.getFn = func(_ context.Context, request onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error) {
		requireStringPtr(t, "get subscriptionId", request.SubscriptionId, testSubscriptionID)
		return onssdk.GetSubscriptionResponse{
			Subscription: makeSDKSubscription(testSubscriptionID, onssdk.SubscriptionLifecycleStateActive),
		}, nil
	}
	fake.createFn = func(context.Context, onssdk.CreateSubscriptionRequest) (onssdk.CreateSubscriptionResponse, error) {
		t.Fatal("CreateSubscription should not be called when list lookup binds an existing Subscription")
		return onssdk.CreateSubscriptionResponse{}, nil
	}

	response, err := testSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful active bind", response)
	}
	if fake.listCalls != 2 {
		t.Fatalf("ListSubscriptions calls = %d, want 2", fake.listCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testSubscriptionID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testSubscriptionID)
	}
	assertSubscriptionMetadataFingerprint(t, resource, "owner=ops")
	assertTrailingCondition(t, resource, shared.Active)
}

func TestSubscriptionCreateOrUpdateRejectsDuplicateListMatchesAcrossPages(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	fake := &fakeSubscriptionOCIClient{}
	fake.listFn = func(_ context.Context, request onssdk.ListSubscriptionsRequest) (onssdk.ListSubscriptionsResponse, error) {
		if fake.listCalls == 1 {
			return onssdk.ListSubscriptionsResponse{
				Items:       []onssdk.SubscriptionSummary{makeSDKSubscriptionSummary(testSubscriptionID, testSubscriptionEndpoint, onssdk.SubscriptionSummaryLifecycleStateActive)},
				OpcNextPage: common.String("page-2"),
			}, nil
		}
		requireStringPtr(t, "second ListSubscriptions page", request.Page, "page-2")
		return onssdk.ListSubscriptionsResponse{
			Items: []onssdk.SubscriptionSummary{
				makeSDKSubscriptionSummary("ocid1.onssubscription.oc1..duplicate", testSubscriptionEndpoint, onssdk.SubscriptionSummaryLifecycleStateActive),
			},
		}, nil
	}

	response, err := testSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err == nil || !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate match error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if fake.createCalls != 0 {
		t.Fatalf("CreateSubscription calls = %d, want 0", fake.createCalls)
	}
}

func TestSubscriptionCreateOrUpdateSkipsNoOpUpdate(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSubscriptionID)
	resource.Status.Id = testSubscriptionID
	resource.Status.DeliveryPolicy = resource.Spec.DeliveryPolicy
	recordSubscriptionMetadataFingerprint(resource)
	fake := &fakeSubscriptionOCIClient{
		getFn: func(_ context.Context, request onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error) {
			requireStringPtr(t, "get subscriptionId", request.SubscriptionId, testSubscriptionID)
			return onssdk.GetSubscriptionResponse{
				Subscription: makeSDKSubscription(testSubscriptionID, onssdk.SubscriptionLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, onssdk.UpdateSubscriptionRequest) (onssdk.UpdateSubscriptionResponse, error) {
			t.Fatal("UpdateSubscription should not be called when mutable fields match observed state")
			return onssdk.UpdateSubscriptionResponse{}, nil
		},
	}

	response, err := testSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no-op", response)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateSubscription calls = %d, want 0", fake.updateCalls)
	}
	assertTrailingCondition(t, resource, shared.Active)
}

func TestSubscriptionCreateOrUpdateAppliesMutableUpdate(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSubscriptionID)
	resource.Status.Id = testSubscriptionID
	resource.Status.DeliveryPolicy = onsv1beta1.SubscriptionDeliveryPolicy{
		BackoffRetryPolicy: onsv1beta1.SubscriptionDeliveryPolicyBackoffRetryPolicy{
			MaxRetryDuration: 3600000,
			PolicyType:       string(onssdk.BackoffRetryPolicyPolicyTypeExponential),
		},
	}
	recordSubscriptionMetadataFingerprint(resource)
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	resource.Spec.DeliveryPolicy.BackoffRetryPolicy.MaxRetryDuration = 9000000

	var updateRequest onssdk.UpdateSubscriptionRequest
	fake := &fakeSubscriptionOCIClient{
		getFn: func(_ context.Context, request onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error) {
			requireStringPtr(t, "get subscriptionId", request.SubscriptionId, testSubscriptionID)
			current := makeSDKSubscription(testSubscriptionID, onssdk.SubscriptionLifecycleStateActive)
			current.FreeformTags = map[string]string{"env": "dev"}
			current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}}
			return onssdk.GetSubscriptionResponse{Subscription: current}, nil
		},
		updateFn: func(_ context.Context, request onssdk.UpdateSubscriptionRequest) (onssdk.UpdateSubscriptionResponse, error) {
			updateRequest = request
			return onssdk.UpdateSubscriptionResponse{
				UpdateSubscriptionDetails: request.UpdateSubscriptionDetails,
				OpcRequestId:              common.String("opc-update"),
			}, nil
		},
	}

	response, err := testSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue after update response", response)
	}
	assertSubscriptionUpdateRequest(t, updateRequest)
	assertSubscriptionUpdateStatus(t, resource)
	assertSubscriptionMetadataFingerprint(t, resource, "owner=ops")
}

func TestSubscriptionCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	resource.Spec.Protocol = "SMS"
	resource.Status.OsokStatus.Ocid = shared.OCID(testSubscriptionID)
	resource.Status.Id = testSubscriptionID
	fake := &fakeSubscriptionOCIClient{
		getFn: func(_ context.Context, _ onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error) {
			return onssdk.GetSubscriptionResponse{
				Subscription: makeSDKSubscription(testSubscriptionID, onssdk.SubscriptionLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, onssdk.UpdateSubscriptionRequest) (onssdk.UpdateSubscriptionResponse, error) {
			t.Fatal("UpdateSubscription should not be called after create-only drift")
			return onssdk.UpdateSubscriptionResponse{}, nil
		},
	}

	response, err := testSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err == nil || !strings.Contains(err.Error(), "replacement when protocol changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want protocol replacement rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateSubscription calls = %d, want 0", fake.updateCalls)
	}
	assertTrailingCondition(t, resource, shared.Failed)
}

func TestSubscriptionCreateOrUpdateRejectsMetadataDriftBeforeMutableUpdate(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSubscriptionID)
	resource.Status.Id = testSubscriptionID
	recordSubscriptionMetadataFingerprint(resource)
	resource.Spec.Metadata = "owner=security"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	fake := &fakeSubscriptionOCIClient{
		getFn: func(_ context.Context, _ onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error) {
			current := makeSDKSubscription(testSubscriptionID, onssdk.SubscriptionLifecycleStateActive)
			current.FreeformTags = map[string]string{"env": "dev"}
			return onssdk.GetSubscriptionResponse{Subscription: current}, nil
		},
		updateFn: func(context.Context, onssdk.UpdateSubscriptionRequest) (onssdk.UpdateSubscriptionResponse, error) {
			t.Fatal("UpdateSubscription should not be called when create-only metadata drifts")
			return onssdk.UpdateSubscriptionResponse{}, nil
		},
	}

	response, err := testSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err == nil || !strings.Contains(err.Error(), "metadata changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want metadata replacement rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateSubscription calls = %d, want 0", fake.updateCalls)
	}
	assertTrailingCondition(t, resource, shared.Failed)
}

func TestSubscriptionCreateOrUpdateRejectsMetadataOnlyDrift(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSubscriptionID)
	resource.Status.Id = testSubscriptionID
	recordSubscriptionMetadataFingerprint(resource)
	resource.Spec.Metadata = "owner=security"
	fake := &fakeSubscriptionOCIClient{
		getFn: func(_ context.Context, _ onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error) {
			return onssdk.GetSubscriptionResponse{
				Subscription: makeSDKSubscription(testSubscriptionID, onssdk.SubscriptionLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, onssdk.UpdateSubscriptionRequest) (onssdk.UpdateSubscriptionResponse, error) {
			t.Fatal("UpdateSubscription should not be called when only create-only metadata drifts")
			return onssdk.UpdateSubscriptionResponse{}, nil
		},
	}

	response, err := testSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err == nil || !strings.Contains(err.Error(), "metadata changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want metadata replacement rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateSubscription calls = %d, want 0", fake.updateCalls)
	}
	assertTrailingCondition(t, resource, shared.Failed)
}

func TestSubscriptionCreateOrUpdatePreservesMetadataFingerprintAfterDriftFailure(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSubscriptionID)
	resource.Status.Id = testSubscriptionID
	recordSubscriptionMetadataFingerprint(resource)
	resource.Spec.Metadata = ""
	fake := &fakeSubscriptionOCIClient{
		getFn: func(_ context.Context, _ onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error) {
			return onssdk.GetSubscriptionResponse{
				Subscription: makeSDKSubscription(testSubscriptionID, onssdk.SubscriptionLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, onssdk.UpdateSubscriptionRequest) (onssdk.UpdateSubscriptionResponse, error) {
			t.Fatal("UpdateSubscription should not be called after metadata drift failure")
			return onssdk.UpdateSubscriptionResponse{}, nil
		},
	}
	client := testSubscriptionClient(fake)

	for attempt := 1; attempt <= 2; attempt++ {
		response, err := client.CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
		if err == nil || !strings.Contains(err.Error(), "metadata changes") {
			t.Fatalf("attempt %d CreateOrUpdate() error = %v, want metadata replacement rejection", attempt, err)
		}
		if response.IsSuccessful {
			t.Fatalf("attempt %d CreateOrUpdate() response = %#v, want unsuccessful", attempt, response)
		}
		assertSubscriptionMetadataFingerprint(t, resource, "owner=ops")
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateSubscription calls = %d, want 0", fake.updateCalls)
	}
	assertTrailingCondition(t, resource, shared.Failed)
}

func TestSubscriptionDeleteKeepsFinalizerUntilConfirmed(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSubscriptionID)
	resource.Status.Id = testSubscriptionID
	var deleteRequest onssdk.DeleteSubscriptionRequest
	fake := &fakeSubscriptionOCIClient{}
	fake.getFn = func(_ context.Context, request onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error) {
		requireStringPtr(t, "get subscriptionId", request.SubscriptionId, testSubscriptionID)
		return onssdk.GetSubscriptionResponse{
			Subscription: makeSDKSubscription(testSubscriptionID, onssdk.SubscriptionLifecycleStateActive),
		}, nil
	}
	fake.deleteFn = func(_ context.Context, request onssdk.DeleteSubscriptionRequest) (onssdk.DeleteSubscriptionResponse, error) {
		deleteRequest = request
		return onssdk.DeleteSubscriptionResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := testSubscriptionClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false until OCI deletion is confirmed")
	}
	requireStringPtr(t, "delete subscriptionId", deleteRequest.SubscriptionId, testSubscriptionID)
	if current := resource.Status.OsokStatus.Async.Current; current == nil ||
		current.Phase != shared.OSOKAsyncPhaseDelete ||
		current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.status.async.current = %#v, want pending delete", current)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	assertTrailingCondition(t, resource, shared.Terminating)
}

func TestSubscriptionDeleteRemovesFinalizerAfterConfirmedNotFound(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSubscriptionID)
	resource.Status.Id = testSubscriptionID
	fake := &fakeSubscriptionOCIClient{}
	fake.getFn = func(_ context.Context, _ onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error) {
		if fake.getCalls == 1 {
			return onssdk.GetSubscriptionResponse{
				Subscription: makeSDKSubscription(testSubscriptionID, onssdk.SubscriptionLifecycleStateActive),
			}, nil
		}
		return onssdk.GetSubscriptionResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "subscription deleted")
	}
	fake.deleteFn = func(context.Context, onssdk.DeleteSubscriptionRequest) (onssdk.DeleteSubscriptionResponse, error) {
		return onssdk.DeleteSubscriptionResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := testSubscriptionClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not-found confirmation")
	}
	if fake.getCalls != 2 {
		t.Fatalf("GetSubscription calls = %d, want 2", fake.getCalls)
	}
	assertTrailingCondition(t, resource, shared.Terminating)
}

func TestSubscriptionDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSubscriptionID)
	resource.Status.Id = testSubscriptionID
	fake := &fakeSubscriptionOCIClient{
		getFn: func(_ context.Context, _ onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error) {
			return onssdk.GetSubscriptionResponse{
				Subscription: makeSDKSubscription(testSubscriptionID, onssdk.SubscriptionLifecycleStateActive),
			}, nil
		},
		deleteFn: func(context.Context, onssdk.DeleteSubscriptionRequest) (onssdk.DeleteSubscriptionResponse, error) {
			err := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or missing")
			err.OpcRequestID = "opc-auth"
			return onssdk.DeleteSubscriptionResponse{}, err
		},
	}

	deleted, err := testSubscriptionClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "keeping the finalizer") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped not-found error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped not-found")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-auth" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-auth", got)
	}
}

func TestSubscriptionDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testSubscriptionID)
	resource.Status.Id = testSubscriptionID
	fake := &fakeSubscriptionOCIClient{
		getFn: func(context.Context, onssdk.GetSubscriptionRequest) (onssdk.GetSubscriptionResponse, error) {
			err := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or missing")
			err.OpcRequestID = "opc-auth-read"
			return onssdk.GetSubscriptionResponse{}, err
		},
		deleteFn: func(context.Context, onssdk.DeleteSubscriptionRequest) (onssdk.DeleteSubscriptionResponse, error) {
			t.Fatal("DeleteSubscription should not be called when pre-delete confirmation is ambiguous")
			return onssdk.DeleteSubscriptionResponse{}, nil
		},
	}

	deleted, err := testSubscriptionClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative pre-delete confirm-read error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete confirm-read")
	}
	if fake.deleteCalls != 0 {
		t.Fatalf("DeleteSubscription calls = %d, want 0", fake.deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-auth-read" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-auth-read", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt should stay empty for auth-shaped pre-delete confirm-read")
	}
}

func TestSubscriptionCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeSubscriptionResource()
	fake := &fakeSubscriptionOCIClient{
		listFn: func(context.Context, onssdk.ListSubscriptionsRequest) (onssdk.ListSubscriptionsResponse, error) {
			return onssdk.ListSubscriptionsResponse{}, nil
		},
		createFn: func(context.Context, onssdk.CreateSubscriptionRequest) (onssdk.CreateSubscriptionResponse, error) {
			err := errortest.NewServiceError(500, "InternalError", "service unavailable")
			err.OpcRequestID = "opc-create-error"
			return onssdk.CreateSubscriptionResponse{}, err
		},
	}

	response, err := testSubscriptionClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-error" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-error", got)
	}
	assertTrailingCondition(t, resource, shared.Failed)
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

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func assertSubscriptionUpdateRequest(t *testing.T, updateRequest onssdk.UpdateSubscriptionRequest) {
	t.Helper()
	requireStringPtr(t, "update subscriptionId", updateRequest.SubscriptionId, testSubscriptionID)
	if got, want := updateRequest.FreeformTags, map[string]string{"env": "prod"}; !jsonEqual(got, want) {
		t.Fatalf("update freeformTags = %#v, want %#v", got, want)
	}
	if got, want := updateRequest.DefinedTags["Operations"]["CostCenter"], "99"; got != want {
		t.Fatalf("update definedTags Operations.CostCenter = %#v, want %q", got, want)
	}
	assertSubscriptionUpdatePolicy(t, updateRequest.DeliveryPolicy, 9000000)
}

func assertSubscriptionUpdatePolicy(t *testing.T, policy *onssdk.DeliveryPolicy, wantMaxRetryDuration int) {
	t.Helper()
	if policy == nil || policy.BackoffRetryPolicy == nil {
		t.Fatalf("update deliveryPolicy = %#v, want backoff retry policy", policy)
	}
	if got := intValue(policy.BackoffRetryPolicy.MaxRetryDuration); got != wantMaxRetryDuration {
		t.Fatalf("update deliveryPolicy.maxRetryDuration = %d, want %d", got, wantMaxRetryDuration)
	}
}

func assertSubscriptionUpdateStatus(t *testing.T, resource *onsv1beta1.Subscription) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	if got := resource.Status.DeliveryPolicy.BackoffRetryPolicy.MaxRetryDuration; got != 9000000 {
		t.Fatalf("status.deliveryPolicy.maxRetryDuration = %d, want 9000000", got)
	}
}

func assertSubscriptionMetadataFingerprint(t *testing.T, resource *onsv1beta1.Subscription, wantMetadata string) {
	t.Helper()
	got, ok := subscriptionRecordedMetadataFingerprint(resource)
	if !ok {
		t.Fatalf("subscription metadata fingerprint missing from status.status.message %q", resource.Status.OsokStatus.Message)
	}
	if want := subscriptionMetadataFingerprint(wantMetadata); got != want {
		t.Fatalf("subscription metadata fingerprint = %q, want %q", got, want)
	}
}

func assertTrailingCondition(t *testing.T, resource *onsv1beta1.Subscription, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("trailing condition = %q, want %q", got, want)
	}
}

func assertStringSliceContainsAll(t *testing.T, name string, got []string, want ...string) {
	t.Helper()
	for _, candidate := range want {
		found := false
		for _, value := range got {
			if value == candidate {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s = %#v, missing %q", name, got, candidate)
		}
	}
}
