/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package limitsincreaserequest

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	limitsincreasesdk "github.com/oracle/oci-go-sdk/v65/limitsincrease"
	limitsincreasev1beta1 "github.com/oracle/oci-service-operator/api/limitsincrease/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testLimitsIncreaseRequestID = "ocid1.limitsincreaserequest.oc1..example"
	testCompartmentID           = "ocid1.compartment.oc1..example"
	testSubscriptionID          = "ocid1.subscription.oc1..example"
)

type fakeLimitsIncreaseRequestOCIClient struct {
	createFn func(context.Context, limitsincreasesdk.CreateLimitsIncreaseRequestRequest) (limitsincreasesdk.CreateLimitsIncreaseRequestResponse, error)
	getFn    func(context.Context, limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error)
	listFn   func(context.Context, limitsincreasesdk.ListLimitsIncreaseRequestsRequest) (limitsincreasesdk.ListLimitsIncreaseRequestsResponse, error)
	updateFn func(context.Context, limitsincreasesdk.UpdateLimitsIncreaseRequestRequest) (limitsincreasesdk.UpdateLimitsIncreaseRequestResponse, error)
	deleteFn func(context.Context, limitsincreasesdk.DeleteLimitsIncreaseRequestRequest) (limitsincreasesdk.DeleteLimitsIncreaseRequestResponse, error)
}

func (f *fakeLimitsIncreaseRequestOCIClient) CreateLimitsIncreaseRequest(ctx context.Context, req limitsincreasesdk.CreateLimitsIncreaseRequestRequest) (limitsincreasesdk.CreateLimitsIncreaseRequestResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return limitsincreasesdk.CreateLimitsIncreaseRequestResponse{}, nil
}

func (f *fakeLimitsIncreaseRequestOCIClient) GetLimitsIncreaseRequest(ctx context.Context, req limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return limitsincreasesdk.GetLimitsIncreaseRequestResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
}

func (f *fakeLimitsIncreaseRequestOCIClient) ListLimitsIncreaseRequests(ctx context.Context, req limitsincreasesdk.ListLimitsIncreaseRequestsRequest) (limitsincreasesdk.ListLimitsIncreaseRequestsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return limitsincreasesdk.ListLimitsIncreaseRequestsResponse{}, nil
}

func (f *fakeLimitsIncreaseRequestOCIClient) UpdateLimitsIncreaseRequest(ctx context.Context, req limitsincreasesdk.UpdateLimitsIncreaseRequestRequest) (limitsincreasesdk.UpdateLimitsIncreaseRequestResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return limitsincreasesdk.UpdateLimitsIncreaseRequestResponse{}, nil
}

func (f *fakeLimitsIncreaseRequestOCIClient) DeleteLimitsIncreaseRequest(ctx context.Context, req limitsincreasesdk.DeleteLimitsIncreaseRequestRequest) (limitsincreasesdk.DeleteLimitsIncreaseRequestResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return limitsincreasesdk.DeleteLimitsIncreaseRequestResponse{}, nil
}

func testLimitsIncreaseRequestClient(fake *fakeLimitsIncreaseRequestOCIClient) LimitsIncreaseRequestServiceClient {
	if fake == nil {
		fake = &fakeLimitsIncreaseRequestOCIClient{}
	}
	hooks := newTestLimitsIncreaseRequestRuntimeHooks(fake)
	applyLimitsIncreaseRequestRuntimeHooks(&hooks)
	delegate := defaultLimitsIncreaseRequestServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*limitsincreasev1beta1.LimitsIncreaseRequest](
			buildLimitsIncreaseRequestGeneratedRuntimeConfig(
				&LimitsIncreaseRequestServiceManager{Log: loggerutil.OSOKLogger{Logger: logr.Discard()}},
				hooks,
			),
		),
	}
	return wrapLimitsIncreaseRequestGeneratedClient(hooks, delegate)
}

func newTestLimitsIncreaseRequestRuntimeHooks(client *fakeLimitsIncreaseRequestOCIClient) LimitsIncreaseRequestRuntimeHooks {
	return LimitsIncreaseRequestRuntimeHooks{
		Create: runtimeOperationHooks[limitsincreasesdk.CreateLimitsIncreaseRequestRequest, limitsincreasesdk.CreateLimitsIncreaseRequestResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateLimitsIncreaseRequestDetails", RequestName: "CreateLimitsIncreaseRequestDetails", Contribution: "body"}},
			Call:   client.CreateLimitsIncreaseRequest,
		},
		Get: runtimeOperationHooks[limitsincreasesdk.GetLimitsIncreaseRequestRequest, limitsincreasesdk.GetLimitsIncreaseRequestResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "LimitsIncreaseRequestId", RequestName: "limitsIncreaseRequestId", Contribution: "path", PreferResourceID: true}},
			Call:   client.GetLimitsIncreaseRequest,
		},
		List: runtimeOperationHooks[limitsincreasesdk.ListLimitsIncreaseRequestsRequest, limitsincreasesdk.ListLimitsIncreaseRequestsResponse]{
			Fields: limitsIncreaseRequestListFields(),
			Call:   client.ListLimitsIncreaseRequests,
		},
		Update: runtimeOperationHooks[limitsincreasesdk.UpdateLimitsIncreaseRequestRequest, limitsincreasesdk.UpdateLimitsIncreaseRequestResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "LimitsIncreaseRequestId", RequestName: "limitsIncreaseRequestId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateLimitsIncreaseRequestDetails", RequestName: "UpdateLimitsIncreaseRequestDetails", Contribution: "body"},
			},
			Call: client.UpdateLimitsIncreaseRequest,
		},
		Delete: runtimeOperationHooks[limitsincreasesdk.DeleteLimitsIncreaseRequestRequest, limitsincreasesdk.DeleteLimitsIncreaseRequestResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "LimitsIncreaseRequestId", RequestName: "limitsIncreaseRequestId", Contribution: "path", PreferResourceID: true}},
			Call:   client.DeleteLimitsIncreaseRequest,
		},
		WrapGeneratedClient: []func(LimitsIncreaseRequestServiceClient) LimitsIncreaseRequestServiceClient{},
	}
}

func makeLimitsIncreaseRequestResource() *limitsincreasev1beta1.LimitsIncreaseRequest {
	return &limitsincreasev1beta1.LimitsIncreaseRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "limits-request",
			Namespace: "default",
			UID:       "limits-request-uid",
		},
		Spec: limitsincreasev1beta1.LimitsIncreaseRequestSpec{
			DisplayName:    "limits-request",
			CompartmentId:  testCompartmentID,
			Justification:  "need more capacity",
			SubscriptionId: testSubscriptionID,
			LimitsIncreaseItemRequests: []limitsincreasev1beta1.LimitsIncreaseRequestLimitsIncreaseItemRequest{
				{
					ServiceName: "compute",
					LimitName:   "vm-count",
					Region:      "us-ashburn-1",
					Value:       20,
					Scope:       "AD-1",
					QuestionnaireResponse: []limitsincreasev1beta1.LimitsIncreaseRequestLimitsIncreaseItemRequestQuestionnaireResponse{
						{Id: "question-2", QuestionResponse: "second"},
						{Id: "question-1", QuestionResponse: "first"},
					},
				},
				{
					ServiceName: "blockstorage",
					LimitName:   "volume-count",
					Region:      "us-ashburn-1",
					Value:       10,
				},
			},
			FreeformTags: map[string]string{"managed-by": "osok"},
			DefinedTags:  map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeSDKLimitsIncreaseRequest(id string, lifecycleState limitsincreasesdk.LimitsIncreaseRequestLifecycleStateEnum) limitsincreasesdk.LimitsIncreaseRequest {
	return makeSDKLimitsIncreaseRequestWithTags(id, lifecycleState, map[string]string{"managed-by": "osok"}, map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}})
}

func makeSDKLimitsIncreaseRequestWithTags(
	id string,
	lifecycleState limitsincreasesdk.LimitsIncreaseRequestLifecycleStateEnum,
	freeformTags map[string]string,
	definedTags map[string]map[string]interface{},
) limitsincreasesdk.LimitsIncreaseRequest {
	return limitsincreasesdk.LimitsIncreaseRequest{
		Id:             common.String(id),
		DisplayName:    common.String("limits-request"),
		CompartmentId:  common.String(testCompartmentID),
		LifecycleState: lifecycleState,
		LimitsIncreaseItemRequests: []limitsincreasesdk.LimitsIncreaseItemRequest{
			{
				Id:                      common.String("item-2"),
				CompartmentId:           common.String(testCompartmentID),
				Region:                  common.String("us-ashburn-1"),
				ServiceName:             common.String("blockstorage"),
				LimitName:               common.String("volume-count"),
				CurrentValue:            common.Int64(5),
				Value:                   common.Int64(10),
				LimitsIncreaseRequestId: common.String(id),
				LifecycleState:          limitsincreasesdk.LimitsIncreaseItemRequestLifecycleStateSucceeded,
			},
			{
				Id:                      common.String("item-1"),
				CompartmentId:           common.String(testCompartmentID),
				Region:                  common.String("us-ashburn-1"),
				ServiceName:             common.String("compute"),
				LimitName:               common.String("vm-count"),
				CurrentValue:            common.Int64(8),
				Value:                   common.Int64(20),
				LimitsIncreaseRequestId: common.String(id),
				LifecycleState:          limitsincreasesdk.LimitsIncreaseItemRequestLifecycleStateSucceeded,
				Scope:                   common.String("AD-1"),
				QuestionnaireResponse: []limitsincreasesdk.LimitsIncreaseItemQuestionResponse{
					{Id: common.String("question-1"), QuestionText: common.String("ignored"), QuestionResponse: common.String("first")},
					{Id: common.String("question-2"), QuestionText: common.String("ignored"), QuestionResponse: common.String("second")},
				},
			},
		},
		FreeformTags:   freeformTags,
		DefinedTags:    definedTags,
		SubscriptionId: common.String(testSubscriptionID),
		Justification:  common.String("need more capacity"),
	}
}

func makeSDKLimitsIncreaseRequestSummary(id string) limitsincreasesdk.LimitsIncreaseRequestSummary {
	return limitsincreasesdk.LimitsIncreaseRequestSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(testCompartmentID),
		LifecycleState: limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded,
		DisplayName:    common.String("limits-request"),
		ItemsInRequest: common.Int64(2),
		FreeformTags:   map[string]string{"managed-by": "osok"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SubscriptionId: common.String(testSubscriptionID),
		Justification:  common.String("need more capacity"),
	}
}

func paginatedLimitsIncreaseRequestListStub(t *testing.T, listCalls *int) limitsIncreaseRequestListFunc {
	t.Helper()
	return func(_ context.Context, req limitsincreasesdk.ListLimitsIncreaseRequestsRequest) (limitsincreasesdk.ListLimitsIncreaseRequestsResponse, error) {
		(*listCalls)++
		requireStringPtr(t, "list compartmentId", req.CompartmentId, testCompartmentID)
		requireStringPtr(t, "list displayName", req.DisplayName, "limits-request")
		switch *listCalls {
		case 1:
			if req.Page != nil {
				t.Fatalf("first list page = %v, want nil", req.Page)
			}
			return limitsincreasesdk.ListLimitsIncreaseRequestsResponse{OpcNextPage: common.String("page-2")}, nil
		case 2:
			requireStringPtr(t, "second list page", req.Page, "page-2")
			return limitsincreasesdk.ListLimitsIncreaseRequestsResponse{
				LimitsIncreaseRequestCollection: limitsincreasesdk.LimitsIncreaseRequestCollection{
					Items: []limitsincreasesdk.LimitsIncreaseRequestSummary{makeSDKLimitsIncreaseRequestSummary(testLimitsIncreaseRequestID)},
				},
			}, nil
		default:
			t.Fatalf("unexpected ListLimitsIncreaseRequests() call %d", *listCalls)
			return limitsincreasesdk.ListLimitsIncreaseRequestsResponse{}, nil
		}
	}
}

func TestLimitsIncreaseRequestRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := newLimitsIncreaseRequestRuntimeSemantics()
	if got == nil {
		t.Fatal("newLimitsIncreaseRequestRuntimeSemantics() = nil")
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" {
		t.Fatalf("Async = %#v, want lifecycle generatedruntime", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete policy = %q follow-up = %q, want retained confirm-delete", got.FinalizerPolicy, got.DeleteFollowUp.Strategy)
	}
	assertStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"ACCEPTED", "IN_PROGRESS"})
	assertStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"SUCCEEDED", "PARTIALLY_SUCCEEDED"})
	assertStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"definedTags", "freeformTags"})
	assertStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"compartmentId", "displayName", "justification", "limitsIncreaseItemRequests", "subscriptionId"})
}

func TestLimitsIncreaseRequestCreateOrUpdateCreatesAndRefreshesStatus(t *testing.T) {
	t.Parallel()

	var createRequest limitsincreasesdk.CreateLimitsIncreaseRequestRequest
	var getRequest limitsincreasesdk.GetLimitsIncreaseRequestRequest
	listCalls := 0
	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
		listFn: func(_ context.Context, req limitsincreasesdk.ListLimitsIncreaseRequestsRequest) (limitsincreasesdk.ListLimitsIncreaseRequestsResponse, error) {
			listCalls++
			requireStringPtr(t, "list compartmentId", req.CompartmentId, testCompartmentID)
			requireStringPtr(t, "list displayName", req.DisplayName, "limits-request")
			return limitsincreasesdk.ListLimitsIncreaseRequestsResponse{}, nil
		},
		createFn: func(_ context.Context, req limitsincreasesdk.CreateLimitsIncreaseRequestRequest) (limitsincreasesdk.CreateLimitsIncreaseRequestResponse, error) {
			createRequest = req
			return limitsincreasesdk.CreateLimitsIncreaseRequestResponse{
				LimitsIncreaseRequest: makeSDKLimitsIncreaseRequest(testLimitsIncreaseRequestID, limitsincreasesdk.LimitsIncreaseRequestLifecycleStateAccepted),
				OpcRequestId:          common.String("opc-create-1"),
			}, nil
		},
		getFn: func(_ context.Context, req limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
			getRequest = req
			return limitsincreasesdk.GetLimitsIncreaseRequestResponse{
				LimitsIncreaseRequest: makeSDKLimitsIncreaseRequest(testLimitsIncreaseRequestID, limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded),
			}, nil
		},
	})

	resource := makeLimitsIncreaseRequestResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue create", response)
	}
	if listCalls != 1 {
		t.Fatalf("ListLimitsIncreaseRequests() calls = %d, want 1", listCalls)
	}
	assertLimitsIncreaseRequestCreate(t, createRequest, resource)
	requireStringPtr(t, "get limitsIncreaseRequestId", getRequest.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
	assertLimitsIncreaseRequestActiveStatus(t, resource, "opc-create-1")
}

func TestLimitsIncreaseRequestCreateOrUpdateBindsExistingAcrossListPages(t *testing.T) {
	t.Parallel()

	createCalls := 0
	getCalls := 0
	listCalls := 0
	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
		listFn: paginatedLimitsIncreaseRequestListStub(t, &listCalls),
		getFn: func(_ context.Context, req limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
			getCalls++
			requireStringPtr(t, "get limitsIncreaseRequestId", req.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
			return limitsincreasesdk.GetLimitsIncreaseRequestResponse{
				LimitsIncreaseRequest: makeSDKLimitsIncreaseRequest(testLimitsIncreaseRequestID, limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded),
			}, nil
		},
		createFn: func(context.Context, limitsincreasesdk.CreateLimitsIncreaseRequestRequest) (limitsincreasesdk.CreateLimitsIncreaseRequestResponse, error) {
			createCalls++
			return limitsincreasesdk.CreateLimitsIncreaseRequestResponse{}, nil
		},
	})

	resource := makeLimitsIncreaseRequestResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue bind", response)
	}
	if createCalls != 0 {
		t.Fatalf("CreateLimitsIncreaseRequest() calls = %d, want 0", createCalls)
	}
	if listCalls != 2 {
		t.Fatalf("ListLimitsIncreaseRequests() calls = %d, want 2", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetLimitsIncreaseRequest() calls = %d, want 1", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testLimitsIncreaseRequestID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testLimitsIncreaseRequestID)
	}
}

func TestLimitsIncreaseRequestCreateOrUpdateDoesNotUpdateMatchingReadback(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
		getFn: func(_ context.Context, req limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
			requireStringPtr(t, "get limitsIncreaseRequestId", req.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
			return limitsincreasesdk.GetLimitsIncreaseRequestResponse{
				LimitsIncreaseRequest: makeSDKLimitsIncreaseRequest(testLimitsIncreaseRequestID, limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded),
			}, nil
		},
		updateFn: func(context.Context, limitsincreasesdk.UpdateLimitsIncreaseRequestRequest) (limitsincreasesdk.UpdateLimitsIncreaseRequestResponse, error) {
			updateCalls++
			return limitsincreasesdk.UpdateLimitsIncreaseRequestResponse{}, nil
		},
	})

	resource := makeLimitsIncreaseRequestResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLimitsIncreaseRequestID)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue observe", response)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateLimitsIncreaseRequest() calls = %d, want 0", updateCalls)
	}
	if got := resource.Status.LifecycleState; got != "SUCCEEDED" {
		t.Fatalf("status.lifecycleState = %q, want SUCCEEDED", got)
	}
}

func TestLimitsIncreaseRequestCreateOrUpdateUpdatesMutableTagsOnly(t *testing.T) {
	t.Parallel()

	getCalls := 0
	var updateRequest limitsincreasesdk.UpdateLimitsIncreaseRequestRequest
	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
		getFn: func(_ context.Context, req limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
			getCalls++
			requireStringPtr(t, "get limitsIncreaseRequestId", req.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
			tags := map[string]string{"managed-by": "legacy"}
			defined := map[string]map[string]interface{}{"Operations": {"CostCenter": "7"}}
			if getCalls > 1 {
				tags = map[string]string{"managed-by": "osok"}
				defined = map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}}
			}
			return limitsincreasesdk.GetLimitsIncreaseRequestResponse{
				LimitsIncreaseRequest: makeSDKLimitsIncreaseRequestWithTags(testLimitsIncreaseRequestID, limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded, tags, defined),
			}, nil
		},
		updateFn: func(_ context.Context, req limitsincreasesdk.UpdateLimitsIncreaseRequestRequest) (limitsincreasesdk.UpdateLimitsIncreaseRequestResponse, error) {
			updateRequest = req
			return limitsincreasesdk.UpdateLimitsIncreaseRequestResponse{
				LimitsIncreaseRequest: makeSDKLimitsIncreaseRequest(testLimitsIncreaseRequestID, limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded),
				OpcRequestId:          common.String("opc-update-1"),
			}, nil
		},
	})

	resource := makeLimitsIncreaseRequestResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLimitsIncreaseRequestID)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue update", response)
	}
	requireStringPtr(t, "update limitsIncreaseRequestId", updateRequest.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
	if got := updateRequest.FreeformTags["managed-by"]; got != "osok" {
		t.Fatalf("update freeformTags[managed-by] = %q, want osok", got)
	}
	if got := updateRequest.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("update definedTags Operations.CostCenter = %#v, want 42", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func TestLimitsIncreaseRequestBuildUpdateBodyTreatsNilTagsAsOmitted(t *testing.T) {
	t.Parallel()

	resource := makeLimitsIncreaseRequestResource()
	resource.Spec.FreeformTags = nil
	resource.Spec.DefinedTags = nil

	body, updateNeeded, err := buildLimitsIncreaseRequestUpdateBody(
		context.Background(),
		resource,
		"",
		limitsincreasesdk.GetLimitsIncreaseRequestResponse{
			LimitsIncreaseRequest: makeSDKLimitsIncreaseRequest(testLimitsIncreaseRequestID, limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded),
		},
	)
	if err != nil {
		t.Fatalf("buildLimitsIncreaseRequestUpdateBody() error = %v", err)
	}
	if updateNeeded {
		t.Fatalf("buildLimitsIncreaseRequestUpdateBody() updateNeeded = true with nil tag specs; body = %#v", body)
	}
}

func TestLimitsIncreaseRequestCreateOrUpdateClearsExplicitEmptyTags(t *testing.T) {
	t.Parallel()

	resource, updateRequest, updateCalls := runLimitsIncreaseRequestClearExplicitEmptyTags(t)
	if updateCalls != 1 {
		t.Fatalf("UpdateLimitsIncreaseRequest() calls = %d, want 1", updateCalls)
	}
	requireStringPtr(t, "update limitsIncreaseRequestId", updateRequest.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
	assertLimitsIncreaseRequestEmptyTagUpdate(t, updateRequest)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-clear-tags" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-clear-tags", got)
	}
}

func runLimitsIncreaseRequestClearExplicitEmptyTags(t *testing.T) (*limitsincreasev1beta1.LimitsIncreaseRequest, limitsincreasesdk.UpdateLimitsIncreaseRequestRequest, int) {
	t.Helper()

	getCalls := 0
	updateCalls := 0
	var updateRequest limitsincreasesdk.UpdateLimitsIncreaseRequestRequest
	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
		getFn: limitsIncreaseRequestClearTagsGetStub(t, &getCalls),
		updateFn: func(_ context.Context, req limitsincreasesdk.UpdateLimitsIncreaseRequestRequest) (limitsincreasesdk.UpdateLimitsIncreaseRequestResponse, error) {
			updateCalls++
			updateRequest = req
			return limitsincreasesdk.UpdateLimitsIncreaseRequestResponse{
				LimitsIncreaseRequest: makeSDKLimitsIncreaseRequestWithTags(
					testLimitsIncreaseRequestID,
					limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded,
					map[string]string{},
					map[string]map[string]interface{}{},
				),
				OpcRequestId: common.String("opc-clear-tags"),
			}, nil
		},
	})

	resource := makeLimitsIncreaseRequestResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLimitsIncreaseRequestID)
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue tag clear", response)
	}
	return resource, updateRequest, updateCalls
}

func limitsIncreaseRequestClearTagsGetStub(t *testing.T, getCalls *int) limitsIncreaseRequestGetFunc {
	t.Helper()
	return func(_ context.Context, req limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
		(*getCalls)++
		requireStringPtr(t, "get limitsIncreaseRequestId", req.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
		freeformTags, definedTags := existingLimitsIncreaseRequestTags()
		if *getCalls > 1 {
			freeformTags, definedTags = emptyLimitsIncreaseRequestTags()
		}
		return limitsincreasesdk.GetLimitsIncreaseRequestResponse{
			LimitsIncreaseRequest: makeSDKLimitsIncreaseRequestWithTags(
				testLimitsIncreaseRequestID,
				limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded,
				freeformTags,
				definedTags,
			),
		}, nil
	}
}

func existingLimitsIncreaseRequestTags() (map[string]string, map[string]map[string]interface{}) {
	return map[string]string{"managed-by": "osok"}, map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}}
}

func emptyLimitsIncreaseRequestTags() (map[string]string, map[string]map[string]interface{}) {
	return map[string]string{}, map[string]map[string]interface{}{}
}

func assertLimitsIncreaseRequestEmptyTagUpdate(t *testing.T, updateRequest limitsincreasesdk.UpdateLimitsIncreaseRequestRequest) {
	t.Helper()
	if updateRequest.FreeformTags == nil || len(updateRequest.FreeformTags) != 0 {
		t.Fatalf("update freeformTags = %#v, want explicit empty map", updateRequest.FreeformTags)
	}
	if updateRequest.DefinedTags == nil || len(updateRequest.DefinedTags) != 0 {
		t.Fatalf("update definedTags = %#v, want explicit empty map", updateRequest.DefinedTags)
	}
}

func TestLimitsIncreaseRequestCreateOrUpdateDoesNotUpdateMatchingEmptyTags(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
		getFn: func(_ context.Context, req limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
			requireStringPtr(t, "get limitsIncreaseRequestId", req.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
			return limitsincreasesdk.GetLimitsIncreaseRequestResponse{
				LimitsIncreaseRequest: makeSDKLimitsIncreaseRequestWithTags(
					testLimitsIncreaseRequestID,
					limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded,
					map[string]string{},
					map[string]map[string]interface{}{},
				),
			}, nil
		},
		updateFn: func(context.Context, limitsincreasesdk.UpdateLimitsIncreaseRequestRequest) (limitsincreasesdk.UpdateLimitsIncreaseRequestResponse, error) {
			updateCalls++
			return limitsincreasesdk.UpdateLimitsIncreaseRequestResponse{}, nil
		},
	})

	resource := makeLimitsIncreaseRequestResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLimitsIncreaseRequestID)
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue observe", response)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateLimitsIncreaseRequest() calls = %d, want 0", updateCalls)
	}
}

func TestLimitsIncreaseRequestCreateOrUpdateRejectsCreateOnlyDrift(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mutateDesired func(*limitsincreasev1beta1.LimitsIncreaseRequest)
		wantField     string
	}{
		{
			name: "displayName",
			mutateDesired: func(resource *limitsincreasev1beta1.LimitsIncreaseRequest) {
				resource.Spec.DisplayName = "renamed"
			},
			wantField: "displayName",
		},
		{
			name: "limit item value",
			mutateDesired: func(resource *limitsincreasev1beta1.LimitsIncreaseRequest) {
				resource.Spec.LimitsIncreaseItemRequests[0].Value = 99
			},
			wantField: "limitsIncreaseItemRequests",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			updateCalls := 0
			client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
				getFn: func(_ context.Context, req limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
					requireStringPtr(t, "get limitsIncreaseRequestId", req.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
					return limitsincreasesdk.GetLimitsIncreaseRequestResponse{
						LimitsIncreaseRequest: makeSDKLimitsIncreaseRequest(testLimitsIncreaseRequestID, limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded),
					}, nil
				},
				updateFn: func(context.Context, limitsincreasesdk.UpdateLimitsIncreaseRequestRequest) (limitsincreasesdk.UpdateLimitsIncreaseRequestResponse, error) {
					updateCalls++
					return limitsincreasesdk.UpdateLimitsIncreaseRequestResponse{}, nil
				},
			})

			resource := makeLimitsIncreaseRequestResource()
			resource.Status.OsokStatus.Ocid = shared.OCID(testLimitsIncreaseRequestID)
			tt.mutateDesired(resource)
			response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err == nil {
				t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
			}
			if !strings.Contains(err.Error(), tt.wantField) {
				t.Fatalf("CreateOrUpdate() error = %v, want field %q", err, tt.wantField)
			}
			if response.IsSuccessful {
				t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful drift rejection", response)
			}
			if updateCalls != 0 {
				t.Fatalf("UpdateLimitsIncreaseRequest() calls = %d, want 0", updateCalls)
			}
		})
	}
}

func TestLimitsIncreaseRequestDeleteRetainsFinalizerUntilReadbackGone(t *testing.T) {
	t.Parallel()

	getCalls := 0
	var deleteRequest limitsincreasesdk.DeleteLimitsIncreaseRequestRequest
	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
		getFn: func(_ context.Context, req limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
			getCalls++
			requireStringPtr(t, "get limitsIncreaseRequestId", req.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
			return limitsincreasesdk.GetLimitsIncreaseRequestResponse{
				LimitsIncreaseRequest: makeSDKLimitsIncreaseRequest(testLimitsIncreaseRequestID, limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded),
			}, nil
		},
		deleteFn: func(_ context.Context, req limitsincreasesdk.DeleteLimitsIncreaseRequestRequest) (limitsincreasesdk.DeleteLimitsIncreaseRequestResponse, error) {
			deleteRequest = req
			return limitsincreasesdk.DeleteLimitsIncreaseRequestResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	resource := makeLimitsIncreaseRequestResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLimitsIncreaseRequestID)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while readback still exists")
	}
	if getCalls != 2 {
		t.Fatalf("GetLimitsIncreaseRequest() calls = %d, want pre-delete and confirm reads", getCalls)
	}
	requireStringPtr(t, "delete limitsIncreaseRequestId", deleteRequest.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
	assertDeletePendingStatus(t, resource, "opc-delete-1")
}

func TestLimitsIncreaseRequestDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	t.Parallel()

	getCalls := 0
	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
		getFn: func(_ context.Context, req limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
			getCalls++
			requireStringPtr(t, "get limitsIncreaseRequestId", req.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
			if getCalls == 1 {
				return limitsincreasesdk.GetLimitsIncreaseRequestResponse{
					LimitsIncreaseRequest: makeSDKLimitsIncreaseRequest(testLimitsIncreaseRequestID, limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded),
				}, nil
			}
			return limitsincreasesdk.GetLimitsIncreaseRequestResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
		},
		deleteFn: func(context.Context, limitsincreasesdk.DeleteLimitsIncreaseRequestRequest) (limitsincreasesdk.DeleteLimitsIncreaseRequestResponse, error) {
			return limitsincreasesdk.DeleteLimitsIncreaseRequestResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	resource := makeLimitsIncreaseRequestResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLimitsIncreaseRequestID)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want delete confirmed by unambiguous 404")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want delete request ID", got)
	}
}

func TestLimitsIncreaseRequestDeleteSkipsDeleteWhenAlreadyPending(t *testing.T) {
	t.Parallel()

	deleteCalls := 0
	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
		getFn: func(_ context.Context, req limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
			requireStringPtr(t, "get limitsIncreaseRequestId", req.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
			return limitsincreasesdk.GetLimitsIncreaseRequestResponse{
				LimitsIncreaseRequest: makeSDKLimitsIncreaseRequest(testLimitsIncreaseRequestID, limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded),
			}, nil
		},
		deleteFn: func(context.Context, limitsincreasesdk.DeleteLimitsIncreaseRequestRequest) (limitsincreasesdk.DeleteLimitsIncreaseRequestResponse, error) {
			deleteCalls++
			return limitsincreasesdk.DeleteLimitsIncreaseRequestResponse{}, nil
		},
	})

	resource := makeLimitsIncreaseRequestResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLimitsIncreaseRequestID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteLimitsIncreaseRequest() calls = %d, want 0", deleteCalls)
	}
	assertDeletePendingStatus(t, resource, "")
}

func TestLimitsIncreaseRequestDeleteRetainsFinalizerForPendingWrite(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name  string
		phase shared.OSOKAsyncPhase
	}{
		{name: "create", phase: shared.OSOKAsyncPhaseCreate},
		{name: "update", phase: shared.OSOKAsyncPhaseUpdate},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runLimitsIncreaseRequestPendingWriteDeleteGuardCase(t, tt.phase)
		})
	}
}

func runLimitsIncreaseRequestPendingWriteDeleteGuardCase(t *testing.T, phase shared.OSOKAsyncPhase) {
	t.Helper()

	getCalls := 0
	deleteCalls := 0
	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
		getFn: func(context.Context, limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
			getCalls++
			return limitsincreasesdk.GetLimitsIncreaseRequestResponse{}, nil
		},
		deleteFn: func(context.Context, limitsincreasesdk.DeleteLimitsIncreaseRequestRequest) (limitsincreasesdk.DeleteLimitsIncreaseRequestResponse, error) {
			deleteCalls++
			return limitsincreasesdk.DeleteLimitsIncreaseRequestResponse{}, nil
		},
	})

	resource := makeLimitsIncreaseRequestResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLimitsIncreaseRequestID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Phase:           phase,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if getCalls != 0 {
		t.Fatalf("GetLimitsIncreaseRequest() calls = %d, want 0", getCalls)
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteLimitsIncreaseRequest() calls = %d, want 0", deleteCalls)
	}
	if got := resource.Status.OsokStatus.Async.Current.Phase; got != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", got, phase)
	}
	if got := resource.Status.OsokStatus.Message; got != limitsIncreaseRequestPendingWriteDeleteMessage {
		t.Fatalf("status.status.message = %q, want pending write delete guard", got)
	}
}

func TestLimitsIncreaseRequestDeleteKeepsAlreadyPendingAuthShapedConfirmReadAmbiguous(t *testing.T) {
	t.Parallel()

	deleteCalls := 0
	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
		getFn: func(_ context.Context, req limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
			requireStringPtr(t, "get limitsIncreaseRequestId", req.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
			return limitsincreasesdk.GetLimitsIncreaseRequestResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, limitsincreasesdk.DeleteLimitsIncreaseRequestRequest) (limitsincreasesdk.DeleteLimitsIncreaseRequestResponse, error) {
			deleteCalls++
			return limitsincreasesdk.DeleteLimitsIncreaseRequestResponse{}, nil
		},
	})

	resource := makeLimitsIncreaseRequestResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLimitsIncreaseRequestID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous confirm-read error")
	}
	if !strings.Contains(err.Error(), "retaining finalizer") {
		t.Fatalf("Delete() error = %v, want retaining finalizer", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous confirm read")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteLimitsIncreaseRequest() calls = %d, want 0", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want confirm-read request ID", got)
	}
}

func TestLimitsIncreaseRequestDeleteKeepsPostDeleteAuthShapedConfirmReadAmbiguous(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0
	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
		getFn: func(_ context.Context, req limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
			getCalls++
			requireStringPtr(t, "get limitsIncreaseRequestId", req.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
			if getCalls == 1 {
				return limitsincreasesdk.GetLimitsIncreaseRequestResponse{
					LimitsIncreaseRequest: makeSDKLimitsIncreaseRequest(testLimitsIncreaseRequestID, limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded),
				}, nil
			}
			return limitsincreasesdk.GetLimitsIncreaseRequestResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, limitsincreasesdk.DeleteLimitsIncreaseRequestRequest) (limitsincreasesdk.DeleteLimitsIncreaseRequestResponse, error) {
			deleteCalls++
			return limitsincreasesdk.DeleteLimitsIncreaseRequestResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	resource := makeLimitsIncreaseRequestResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLimitsIncreaseRequestID)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous confirm-read error")
	}
	if !strings.Contains(err.Error(), "retaining finalizer") {
		t.Fatalf("Delete() error = %v, want retaining finalizer", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous confirm read")
	}
	if getCalls != 2 {
		t.Fatalf("GetLimitsIncreaseRequest() calls = %d, want pre-delete and confirm reads", getCalls)
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteLimitsIncreaseRequest() calls = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want confirm-read request ID", got)
	}
}

func TestLimitsIncreaseRequestDeleteKeepsAuthShapedNotFoundAmbiguous(t *testing.T) {
	t.Parallel()

	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
		getFn: func(_ context.Context, req limitsincreasesdk.GetLimitsIncreaseRequestRequest) (limitsincreasesdk.GetLimitsIncreaseRequestResponse, error) {
			requireStringPtr(t, "get limitsIncreaseRequestId", req.LimitsIncreaseRequestId, testLimitsIncreaseRequestID)
			return limitsincreasesdk.GetLimitsIncreaseRequestResponse{
				LimitsIncreaseRequest: makeSDKLimitsIncreaseRequest(testLimitsIncreaseRequestID, limitsincreasesdk.LimitsIncreaseRequestLifecycleStateSucceeded),
			}, nil
		},
		deleteFn: func(context.Context, limitsincreasesdk.DeleteLimitsIncreaseRequestRequest) (limitsincreasesdk.DeleteLimitsIncreaseRequestResponse, error) {
			return limitsincreasesdk.DeleteLimitsIncreaseRequestResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "ambiguous")
		},
	})

	resource := makeLimitsIncreaseRequestResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testLimitsIncreaseRequestID)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped 404 error")
	}
	if !strings.Contains(err.Error(), "retaining finalizer") {
		t.Fatalf("Delete() error = %v, want retaining finalizer", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous auth-shaped 404")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want error request ID", got)
	}
}

func TestLimitsIncreaseRequestCreateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	client := testLimitsIncreaseRequestClient(&fakeLimitsIncreaseRequestOCIClient{
		listFn: func(context.Context, limitsincreasesdk.ListLimitsIncreaseRequestsRequest) (limitsincreasesdk.ListLimitsIncreaseRequestsResponse, error) {
			return limitsincreasesdk.ListLimitsIncreaseRequestsResponse{}, nil
		},
		createFn: func(context.Context, limitsincreasesdk.CreateLimitsIncreaseRequestRequest) (limitsincreasesdk.CreateLimitsIncreaseRequestResponse, error) {
			return limitsincreasesdk.CreateLimitsIncreaseRequestResponse{}, errortest.NewServiceError(400, "InvalidParameter", "bad request")
		},
	})

	resource := makeLimitsIncreaseRequestResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want OCI error request ID", got)
	}
	if got := lastLimitsIncreaseRequestCondition(resource); got != shared.Failed {
		t.Fatalf("last condition = %q, want Failed", got)
	}
}

func assertLimitsIncreaseRequestCreate(t *testing.T, request limitsincreasesdk.CreateLimitsIncreaseRequestRequest, resource *limitsincreasev1beta1.LimitsIncreaseRequest) {
	t.Helper()
	requireStringPtr(t, "create displayName", request.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "create compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "create justification", request.Justification, resource.Spec.Justification)
	requireStringPtr(t, "create subscriptionId", request.SubscriptionId, resource.Spec.SubscriptionId)
	requireStringPtr(t, "create opcRetryToken", request.OpcRetryToken, string(resource.UID))
	if got := request.FreeformTags["managed-by"]; got != "osok" {
		t.Fatalf("create freeformTags[managed-by] = %q, want osok", got)
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("create definedTags Operations.CostCenter = %#v, want 42", got)
	}
	if got, want := len(request.LimitsIncreaseItemRequests), 2; got != want {
		t.Fatalf("create limitsIncreaseItemRequests len = %d, want %d", got, want)
	}
	first := request.LimitsIncreaseItemRequests[0]
	requireStringPtr(t, "first item serviceName", first.ServiceName, "blockstorage")
	requireInt64Ptr(t, "first item value", first.Value, 10)
	second := request.LimitsIncreaseItemRequests[1]
	requireStringPtr(t, "second item serviceName", second.ServiceName, "compute")
	if got, want := len(second.QuestionnaireResponse), 2; got != want {
		t.Fatalf("second item questionnaire len = %d, want %d", got, want)
	}
	requireStringPtr(t, "first questionnaire id", second.QuestionnaireResponse[0].Id, "question-1")
}

func assertLimitsIncreaseRequestActiveStatus(t *testing.T, resource *limitsincreasev1beta1.LimitsIncreaseRequest, wantRequestID string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != testLimitsIncreaseRequestID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testLimitsIncreaseRequestID)
	}
	if got := resource.Status.Id; got != testLimitsIncreaseRequestID {
		t.Fatalf("status.id = %q, want %q", got, testLimitsIncreaseRequestID)
	}
	if got := resource.Status.LifecycleState; got != "SUCCEEDED" {
		t.Fatalf("status.lifecycleState = %q, want SUCCEEDED", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != wantRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, wantRequestID)
	}
	if got := lastLimitsIncreaseRequestCondition(resource); got != shared.Active {
		t.Fatalf("last condition = %q, want Active", got)
	}
}

func assertDeletePendingStatus(t *testing.T, resource *limitsincreasev1beta1.LimitsIncreaseRequest, wantRequestID string) {
	t.Helper()
	if wantRequestID != "" && resource.Status.OsokStatus.OpcRequestID != wantRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, wantRequestID)
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want delete tracker")
	}
	if current.Phase != shared.OSOKAsyncPhaseDelete || current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current = %#v, want pending delete tracker", current)
	}
	if got := lastLimitsIncreaseRequestCondition(resource); got != shared.Terminating {
		t.Fatalf("last condition = %q, want Terminating", got)
	}
}

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", name, got, want)
	}
}

func requireInt64Ptr(t *testing.T, name string, got *int64, want int64) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %d", name, got, want)
	}
}

func assertStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s len = %d, want %d (%#v)", name, len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q", name, i, got[i], want[i])
		}
	}
}

func lastLimitsIncreaseRequestCondition(resource *limitsincreasev1beta1.LimitsIncreaseRequest) shared.OSOKConditionType {
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		return ""
	}
	return shared.OSOKConditionType(resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type)
}
