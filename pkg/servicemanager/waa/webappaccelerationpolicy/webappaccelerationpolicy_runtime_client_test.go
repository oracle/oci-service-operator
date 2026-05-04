/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package webappaccelerationpolicy

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	waasdk "github.com/oracle/oci-go-sdk/v65/waa"
	waav1beta1 "github.com/oracle/oci-service-operator/api/waa/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testWebAppAccelerationPolicyID          = "ocid1.webappaccelerationpolicy.oc1..policy"
	testWebAppAccelerationPolicyOtherID     = "ocid1.webappaccelerationpolicy.oc1..other"
	testWebAppAccelerationPolicyCompartment = "ocid1.compartment.oc1..waa"
	testWebAppAccelerationPolicyName        = "waa-policy"
)

type fakeWebAppAccelerationPolicyOCIClient struct {
	changeCompartmentFn func(context.Context, waasdk.ChangeWebAppAccelerationPolicyCompartmentRequest) (waasdk.ChangeWebAppAccelerationPolicyCompartmentResponse, error)
	createFn            func(context.Context, waasdk.CreateWebAppAccelerationPolicyRequest) (waasdk.CreateWebAppAccelerationPolicyResponse, error)
	getFn               func(context.Context, waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error)
	listFn              func(context.Context, waasdk.ListWebAppAccelerationPoliciesRequest) (waasdk.ListWebAppAccelerationPoliciesResponse, error)
	updateFn            func(context.Context, waasdk.UpdateWebAppAccelerationPolicyRequest) (waasdk.UpdateWebAppAccelerationPolicyResponse, error)
	deleteFn            func(context.Context, waasdk.DeleteWebAppAccelerationPolicyRequest) (waasdk.DeleteWebAppAccelerationPolicyResponse, error)
	getWorkRequestFn    func(context.Context, waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error)
}

func (f *fakeWebAppAccelerationPolicyOCIClient) ChangeWebAppAccelerationPolicyCompartment(
	ctx context.Context,
	request waasdk.ChangeWebAppAccelerationPolicyCompartmentRequest,
) (waasdk.ChangeWebAppAccelerationPolicyCompartmentResponse, error) {
	if f.changeCompartmentFn != nil {
		return f.changeCompartmentFn(ctx, request)
	}
	return waasdk.ChangeWebAppAccelerationPolicyCompartmentResponse{}, nil
}

func (f *fakeWebAppAccelerationPolicyOCIClient) CreateWebAppAccelerationPolicy(
	ctx context.Context,
	request waasdk.CreateWebAppAccelerationPolicyRequest,
) (waasdk.CreateWebAppAccelerationPolicyResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return waasdk.CreateWebAppAccelerationPolicyResponse{}, nil
}

func (f *fakeWebAppAccelerationPolicyOCIClient) GetWebAppAccelerationPolicy(
	ctx context.Context,
	request waasdk.GetWebAppAccelerationPolicyRequest,
) (waasdk.GetWebAppAccelerationPolicyResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return waasdk.GetWebAppAccelerationPolicyResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "policy not found")
}

func (f *fakeWebAppAccelerationPolicyOCIClient) ListWebAppAccelerationPolicies(
	ctx context.Context,
	request waasdk.ListWebAppAccelerationPoliciesRequest,
) (waasdk.ListWebAppAccelerationPoliciesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return waasdk.ListWebAppAccelerationPoliciesResponse{}, nil
}

func (f *fakeWebAppAccelerationPolicyOCIClient) UpdateWebAppAccelerationPolicy(
	ctx context.Context,
	request waasdk.UpdateWebAppAccelerationPolicyRequest,
) (waasdk.UpdateWebAppAccelerationPolicyResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return waasdk.UpdateWebAppAccelerationPolicyResponse{}, nil
}

func (f *fakeWebAppAccelerationPolicyOCIClient) DeleteWebAppAccelerationPolicy(
	ctx context.Context,
	request waasdk.DeleteWebAppAccelerationPolicyRequest,
) (waasdk.DeleteWebAppAccelerationPolicyResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return waasdk.DeleteWebAppAccelerationPolicyResponse{}, nil
}

func (f *fakeWebAppAccelerationPolicyOCIClient) GetWorkRequest(
	ctx context.Context,
	request waasdk.GetWorkRequestRequest,
) (waasdk.GetWorkRequestResponse, error) {
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, request)
	}
	return waasdk.GetWorkRequestResponse{}, nil
}

func testWebAppAccelerationPolicyClient(fake *fakeWebAppAccelerationPolicyOCIClient) WebAppAccelerationPolicyServiceClient {
	return newWebAppAccelerationPolicyServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		fake,
	)
}

func makeWebAppAccelerationPolicyResource() *waav1beta1.WebAppAccelerationPolicy {
	return &waav1beta1.WebAppAccelerationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testWebAppAccelerationPolicyName,
			Namespace: "default",
			UID:       types.UID("webappaccelerationpolicy-uid"),
		},
		Spec: waav1beta1.WebAppAccelerationPolicySpec{
			CompartmentId: testWebAppAccelerationPolicyCompartment,
			DisplayName:   testWebAppAccelerationPolicyName,
			ResponseCachingPolicy: waav1beta1.WebAppAccelerationPolicyResponseCachingPolicy{
				IsResponseHeaderBasedCachingEnabled: true,
			},
			ResponseCompressionPolicy: waav1beta1.WebAppAccelerationPolicyResponseCompressionPolicy{
				GzipCompression: waav1beta1.WebAppAccelerationPolicyResponseCompressionPolicyGzipCompression{
					IsEnabled: true,
				},
			},
			FreeformTags: map[string]string{"env": "test"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
			SystemTags: map[string]shared.MapValue{
				"orcl-cloud": {"free-tier-retained": "true"},
			},
		},
	}
}

func makeTrackedWebAppAccelerationPolicyResource() *waav1beta1.WebAppAccelerationPolicy {
	resource := makeWebAppAccelerationPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationPolicyID)
	resource.Status.Id = testWebAppAccelerationPolicyID
	resource.Status.CompartmentId = testWebAppAccelerationPolicyCompartment
	resource.Status.DisplayName = testWebAppAccelerationPolicyName
	resource.Status.LifecycleState = string(waasdk.WebAppAccelerationPolicyLifecycleStateActive)
	return resource
}

func makeWebAppAccelerationPolicyRequest(resource *waav1beta1.WebAppAccelerationPolicy) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}}
}

func makeSDKWebAppAccelerationPolicy(
	id string,
	compartmentID string,
	displayName string,
	cachingEnabled bool,
	compressionEnabled bool,
	state waasdk.WebAppAccelerationPolicyLifecycleStateEnum,
) waasdk.WebAppAccelerationPolicy {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 5, 0, 0, time.UTC)}
	return waasdk.WebAppAccelerationPolicy{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		TimeCreated:    &created,
		TimeUpdated:    &updated,
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "test"},
		DefinedTags: map[string]map[string]interface{}{
			"Operations": {"CostCenter": "42"},
		},
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {"free-tier-retained": "true"},
		},
		ResponseCachingPolicy: &waasdk.ResponseCachingPolicy{
			IsResponseHeaderBasedCachingEnabled: common.Bool(cachingEnabled),
		},
		ResponseCompressionPolicy: &waasdk.ResponseCompressionPolicy{
			GzipCompression: &waasdk.GzipCompressionPolicy{IsEnabled: common.Bool(compressionEnabled)},
		},
	}
}

func makeSDKWebAppAccelerationPolicySummary(
	id string,
	compartmentID string,
	displayName string,
	state waasdk.WebAppAccelerationPolicyLifecycleStateEnum,
) waasdk.WebAppAccelerationPolicySummary {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return waasdk.WebAppAccelerationPolicySummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		TimeCreated:    &created,
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "test"},
		DefinedTags: map[string]map[string]interface{}{
			"Operations": {"CostCenter": "42"},
		},
		SystemTags: map[string]map[string]interface{}{
			"orcl-cloud": {"free-tier-retained": "true"},
		},
	}
}

func makeWebAppAccelerationPolicyWorkRequest(
	id string,
	operationType waasdk.WorkRequestOperationTypeEnum,
	status waasdk.WorkRequestStatusEnum,
) waasdk.WorkRequest {
	percentComplete := float32(100)
	return waasdk.WorkRequest{
		Id:              common.String(id),
		OperationType:   operationType,
		Status:          status,
		PercentComplete: &percentComplete,
	}
}

func TestWebAppAccelerationPolicyRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := webAppAccelerationPolicyRuntimeSemantics()
	if got == nil {
		t.Fatal("webAppAccelerationPolicyRuntimeSemantics() = nil")
	}
	if got.Async == nil || got.Async.Strategy != "lifecycle" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime lifecycle", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" || got.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %#v followUp = %#v, want required confirm-delete", got.Delete, got.DeleteFollowUp)
	}
	requireStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"compartmentId", "displayName", "id"})
	requireStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{
		"compartmentId",
		"displayName",
		"responseCachingPolicy",
		"responseCompressionPolicy",
		"freeformTags",
		"definedTags",
		"systemTags",
	})
	if len(got.Mutation.ForceNew) != 0 {
		t.Fatalf("Mutation.ForceNew = %#v, want no replacement-only fields", got.Mutation.ForceNew)
	}
}

func TestWebAppAccelerationPolicyServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationPolicyResource()
	var createRequest waasdk.CreateWebAppAccelerationPolicyRequest
	listCalls := 0
	createCalls := 0
	getCalls := 0
	client := testWebAppAccelerationPolicyClient(&fakeWebAppAccelerationPolicyOCIClient{
		listFn: func(_ context.Context, request waasdk.ListWebAppAccelerationPoliciesRequest) (waasdk.ListWebAppAccelerationPoliciesResponse, error) {
			listCalls++
			requireStringPtr(t, "ListWebAppAccelerationPoliciesRequest.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireStringPtr(t, "ListWebAppAccelerationPoliciesRequest.DisplayName", request.DisplayName, resource.Spec.DisplayName)
			return waasdk.ListWebAppAccelerationPoliciesResponse{}, nil
		},
		createFn: func(_ context.Context, request waasdk.CreateWebAppAccelerationPolicyRequest) (waasdk.CreateWebAppAccelerationPolicyResponse, error) {
			createCalls++
			createRequest = request
			return waasdk.CreateWebAppAccelerationPolicyResponse{
				WebAppAccelerationPolicy: makeSDKWebAppAccelerationPolicy(
					testWebAppAccelerationPolicyID,
					resource.Spec.CompartmentId,
					resource.Spec.DisplayName,
					true,
					true,
					waasdk.WebAppAccelerationPolicyLifecycleStateCreating,
				),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error) {
			getCalls++
			requireStringPtr(t, "GetWebAppAccelerationPolicyRequest.WebAppAccelerationPolicyId", request.WebAppAccelerationPolicyId, testWebAppAccelerationPolicyID)
			return waasdk.GetWebAppAccelerationPolicyResponse{
				WebAppAccelerationPolicy: makeSDKWebAppAccelerationPolicy(
					testWebAppAccelerationPolicyID,
					resource.Spec.CompartmentId,
					resource.Spec.DisplayName,
					true,
					true,
					waasdk.WebAppAccelerationPolicyLifecycleStateActive,
				),
				OpcRequestId: common.String("opc-get"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeWebAppAccelerationPolicyRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if listCalls != 1 || createCalls != 1 || getCalls != 1 {
		t.Fatalf("OCI calls list/create/get = %d/%d/%d, want 1/1/1", listCalls, createCalls, getCalls)
	}
	requireWebAppAccelerationPolicyCreateRequest(t, createRequest, resource)
	requireCreatedWebAppAccelerationPolicyStatus(t, resource)
}

func TestWebAppAccelerationPolicyServiceClientBindsExistingPolicyFromPaginatedList(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationPolicyResource()
	listCalls := 0
	createCalls := 0
	getCalls := 0
	client := testWebAppAccelerationPolicyClient(&fakeWebAppAccelerationPolicyOCIClient{
		listFn: func(_ context.Context, request waasdk.ListWebAppAccelerationPoliciesRequest) (waasdk.ListWebAppAccelerationPoliciesResponse, error) {
			listCalls++
			requireStringPtr(t, "ListWebAppAccelerationPoliciesRequest.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			requireStringPtr(t, "ListWebAppAccelerationPoliciesRequest.DisplayName", request.DisplayName, resource.Spec.DisplayName)
			if listCalls == 1 {
				if request.Page != nil {
					t.Fatalf("first ListWebAppAccelerationPoliciesRequest.Page = %q, want nil", *request.Page)
				}
				return waasdk.ListWebAppAccelerationPoliciesResponse{
					WebAppAccelerationPolicyCollection: waasdk.WebAppAccelerationPolicyCollection{
						Items: []waasdk.WebAppAccelerationPolicySummary{
							makeSDKWebAppAccelerationPolicySummary(
								testWebAppAccelerationPolicyOtherID,
								resource.Spec.CompartmentId,
								"other-policy",
								waasdk.WebAppAccelerationPolicyLifecycleStateActive,
							),
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			requireStringPtr(t, "second ListWebAppAccelerationPoliciesRequest.Page", request.Page, "page-2")
			return waasdk.ListWebAppAccelerationPoliciesResponse{
				WebAppAccelerationPolicyCollection: waasdk.WebAppAccelerationPolicyCollection{
					Items: []waasdk.WebAppAccelerationPolicySummary{
						makeSDKWebAppAccelerationPolicySummary(
							testWebAppAccelerationPolicyID,
							resource.Spec.CompartmentId,
							resource.Spec.DisplayName,
							waasdk.WebAppAccelerationPolicyLifecycleStateActive,
						),
					},
				},
			}, nil
		},
		createFn: func(context.Context, waasdk.CreateWebAppAccelerationPolicyRequest) (waasdk.CreateWebAppAccelerationPolicyResponse, error) {
			createCalls++
			return waasdk.CreateWebAppAccelerationPolicyResponse{}, nil
		},
		getFn: func(_ context.Context, request waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error) {
			getCalls++
			requireStringPtr(t, "GetWebAppAccelerationPolicyRequest.WebAppAccelerationPolicyId", request.WebAppAccelerationPolicyId, testWebAppAccelerationPolicyID)
			return waasdk.GetWebAppAccelerationPolicyResponse{
				WebAppAccelerationPolicy: makeSDKWebAppAccelerationPolicy(
					testWebAppAccelerationPolicyID,
					resource.Spec.CompartmentId,
					resource.Spec.DisplayName,
					true,
					true,
					waasdk.WebAppAccelerationPolicyLifecycleStateActive,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeWebAppAccelerationPolicyRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if listCalls != 2 || getCalls != 1 || createCalls != 0 {
		t.Fatalf("OCI calls list/get/create = %d/%d/%d, want 2/1/0", listCalls, getCalls, createCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testWebAppAccelerationPolicyID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testWebAppAccelerationPolicyID)
	}
}

func TestWebAppAccelerationPolicyServiceClientSkipsNoOpUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedWebAppAccelerationPolicyResource()
	updateCalls := 0
	client := testWebAppAccelerationPolicyClient(&fakeWebAppAccelerationPolicyOCIClient{
		getFn: func(_ context.Context, request waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error) {
			requireStringPtr(t, "GetWebAppAccelerationPolicyRequest.WebAppAccelerationPolicyId", request.WebAppAccelerationPolicyId, testWebAppAccelerationPolicyID)
			return waasdk.GetWebAppAccelerationPolicyResponse{
				WebAppAccelerationPolicy: makeSDKWebAppAccelerationPolicy(
					testWebAppAccelerationPolicyID,
					resource.Spec.CompartmentId,
					resource.Spec.DisplayName,
					true,
					true,
					waasdk.WebAppAccelerationPolicyLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(context.Context, waasdk.UpdateWebAppAccelerationPolicyRequest) (waasdk.UpdateWebAppAccelerationPolicyResponse, error) {
			updateCalls++
			return waasdk.UpdateWebAppAccelerationPolicyResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeWebAppAccelerationPolicyRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateWebAppAccelerationPolicy calls = %d, want 0", updateCalls)
	}
}

func TestBuildWebAppAccelerationPolicyUpdateBodyPreservesOmittedTags(t *testing.T) {
	t.Parallel()

	resource := makeTrackedWebAppAccelerationPolicyResource()
	resource.Spec.FreeformTags = nil
	resource.Spec.DefinedTags = nil
	resource.Spec.SystemTags = nil
	current := makeSDKWebAppAccelerationPolicy(
		testWebAppAccelerationPolicyID,
		testWebAppAccelerationPolicyCompartment,
		testWebAppAccelerationPolicyName,
		true,
		true,
		waasdk.WebAppAccelerationPolicyLifecycleStateActive,
	)

	body, updateNeeded, err := buildWebAppAccelerationPolicyUpdateBody(
		context.Background(),
		resource,
		testWebAppAccelerationPolicyID,
		waasdk.GetWebAppAccelerationPolicyResponse{WebAppAccelerationPolicy: current},
	)
	if err != nil {
		t.Fatalf("buildWebAppAccelerationPolicyUpdateBody() error = %v", err)
	}
	if updateNeeded {
		t.Fatal("buildWebAppAccelerationPolicyUpdateBody() updateNeeded = true, want false when only tag maps are omitted")
	}
	updateBody, ok := body.(waasdk.UpdateWebAppAccelerationPolicyDetails)
	if !ok {
		t.Fatalf("buildWebAppAccelerationPolicyUpdateBody() body type = %T, want UpdateWebAppAccelerationPolicyDetails", body)
	}
	if updateBody.FreeformTags != nil || updateBody.DefinedTags != nil || updateBody.SystemTags != nil {
		t.Fatalf("buildWebAppAccelerationPolicyUpdateBody() tags = %#v/%#v/%#v, want omitted nil maps",
			updateBody.FreeformTags,
			updateBody.DefinedTags,
			updateBody.SystemTags,
		)
	}
}

func TestWebAppAccelerationPolicyServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeTrackedWebAppAccelerationPolicyResource()
	resource.Spec.DisplayName = "waa-policy-updated"
	resource.Spec.ResponseCachingPolicy.IsResponseHeaderBasedCachingEnabled = false
	resource.Spec.ResponseCompressionPolicy.GzipCompression.IsEnabled = false
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	resource.Spec.SystemTags = map[string]shared.MapValue{}

	current := makeSDKWebAppAccelerationPolicy(
		testWebAppAccelerationPolicyID,
		testWebAppAccelerationPolicyCompartment,
		testWebAppAccelerationPolicyName,
		true,
		true,
		waasdk.WebAppAccelerationPolicyLifecycleStateActive,
	)
	updated := makeSDKWebAppAccelerationPolicy(
		testWebAppAccelerationPolicyID,
		testWebAppAccelerationPolicyCompartment,
		resource.Spec.DisplayName,
		false,
		false,
		waasdk.WebAppAccelerationPolicyLifecycleStateActive,
	)
	updated.FreeformTags = map[string]string{}
	updated.DefinedTags = map[string]map[string]interface{}{}
	updated.SystemTags = map[string]map[string]interface{}{}

	getCalls := 0
	updateCalls := 0
	var updateRequest waasdk.UpdateWebAppAccelerationPolicyRequest
	client := testWebAppAccelerationPolicyClient(&fakeWebAppAccelerationPolicyOCIClient{
		getFn: func(_ context.Context, request waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error) {
			getCalls++
			requireStringPtr(t, "GetWebAppAccelerationPolicyRequest.WebAppAccelerationPolicyId", request.WebAppAccelerationPolicyId, testWebAppAccelerationPolicyID)
			if getCalls == 1 {
				return waasdk.GetWebAppAccelerationPolicyResponse{WebAppAccelerationPolicy: current}, nil
			}
			return waasdk.GetWebAppAccelerationPolicyResponse{WebAppAccelerationPolicy: updated}, nil
		},
		updateFn: func(_ context.Context, request waasdk.UpdateWebAppAccelerationPolicyRequest) (waasdk.UpdateWebAppAccelerationPolicyResponse, error) {
			updateCalls++
			updateRequest = request
			return waasdk.UpdateWebAppAccelerationPolicyResponse{OpcRequestId: common.String("opc-update")}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeWebAppAccelerationPolicyRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if updateCalls != 1 || getCalls != 2 {
		t.Fatalf("OCI calls update/get = %d/%d, want 1/2", updateCalls, getCalls)
	}
	requireWebAppAccelerationPolicyMutableUpdateRequest(t, updateRequest, resource)
	requireUpdatedWebAppAccelerationPolicyStatus(t, resource)
}

func TestWebAppAccelerationPolicyServiceClientChangesCompartmentWithMoveWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeTrackedWebAppAccelerationPolicyResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..replacement"
	getCalls := 0
	changeCalls := 0
	updateCalls := 0
	workRequestCalls := 0
	client := testWebAppAccelerationPolicyClient(&fakeWebAppAccelerationPolicyOCIClient{
		getFn: func(_ context.Context, request waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error) {
			getCalls++
			requireStringPtr(t, "GetWebAppAccelerationPolicyRequest.WebAppAccelerationPolicyId", request.WebAppAccelerationPolicyId, testWebAppAccelerationPolicyID)
			return waasdk.GetWebAppAccelerationPolicyResponse{
				WebAppAccelerationPolicy: makeSDKWebAppAccelerationPolicy(
					testWebAppAccelerationPolicyID,
					testWebAppAccelerationPolicyCompartment,
					resource.Spec.DisplayName,
					true,
					true,
					waasdk.WebAppAccelerationPolicyLifecycleStateActive,
				),
			}, nil
		},
		changeCompartmentFn: func(_ context.Context, request waasdk.ChangeWebAppAccelerationPolicyCompartmentRequest) (waasdk.ChangeWebAppAccelerationPolicyCompartmentResponse, error) {
			changeCalls++
			requireStringPtr(t, "ChangeWebAppAccelerationPolicyCompartmentRequest.WebAppAccelerationPolicyId", request.WebAppAccelerationPolicyId, testWebAppAccelerationPolicyID)
			requireStringPtr(t, "ChangeWebAppAccelerationPolicyCompartmentDetails.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
			return waasdk.ChangeWebAppAccelerationPolicyCompartmentResponse{
				OpcWorkRequestId: common.String("wr-move"),
				OpcRequestId:     common.String("opc-move"),
			}, nil
		},
		updateFn: func(context.Context, waasdk.UpdateWebAppAccelerationPolicyRequest) (waasdk.UpdateWebAppAccelerationPolicyResponse, error) {
			updateCalls++
			return waasdk.UpdateWebAppAccelerationPolicyResponse{}, nil
		},
		getWorkRequestFn: func(context.Context, waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			return waasdk.GetWorkRequestResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeWebAppAccelerationPolicyRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireWebAppAccelerationPolicyMoveStarted(t, response, resource, getCalls, changeCalls, workRequestCalls, updateCalls)
}

func TestWebAppAccelerationPolicyServiceClientCompletesPendingCompartmentMove(t *testing.T) {
	t.Parallel()

	resource := makeTrackedWebAppAccelerationPolicyResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..replacement"
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:   "wr-move",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	getCalls := 0
	changeCalls := 0
	workRequestCalls := 0
	updateCalls := 0
	client := testWebAppAccelerationPolicyClient(&fakeWebAppAccelerationPolicyOCIClient{
		getFn: func(_ context.Context, request waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error) {
			getCalls++
			requireStringPtr(t, "GetWebAppAccelerationPolicyRequest.WebAppAccelerationPolicyId", request.WebAppAccelerationPolicyId, testWebAppAccelerationPolicyID)
			compartmentID := testWebAppAccelerationPolicyCompartment
			if getCalls == 1 {
				compartmentID = resource.Spec.CompartmentId
			}
			return waasdk.GetWebAppAccelerationPolicyResponse{
				WebAppAccelerationPolicy: makeSDKWebAppAccelerationPolicy(
					testWebAppAccelerationPolicyID,
					compartmentID,
					resource.Spec.DisplayName,
					true,
					true,
					waasdk.WebAppAccelerationPolicyLifecycleStateActive,
				),
			}, nil
		},
		changeCompartmentFn: func(context.Context, waasdk.ChangeWebAppAccelerationPolicyCompartmentRequest) (waasdk.ChangeWebAppAccelerationPolicyCompartmentResponse, error) {
			changeCalls++
			return waasdk.ChangeWebAppAccelerationPolicyCompartmentResponse{}, nil
		},
		getWorkRequestFn: func(_ context.Context, request waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, "wr-move")
			return waasdk.GetWorkRequestResponse{
				WorkRequest: makeWebAppAccelerationPolicyWorkRequest(
					"wr-move",
					waasdk.WorkRequestOperationTypeMoveWaaPolicy,
					waasdk.WorkRequestStatusSucceeded,
				),
				OpcRequestId: common.String("opc-workrequest"),
			}, nil
		},
		updateFn: func(context.Context, waasdk.UpdateWebAppAccelerationPolicyRequest) (waasdk.UpdateWebAppAccelerationPolicyResponse, error) {
			updateCalls++
			return waasdk.UpdateWebAppAccelerationPolicyResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeWebAppAccelerationPolicyRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireWebAppAccelerationPolicyMoveCompleted(t, response, resource, getCalls, workRequestCalls, changeCalls, updateCalls)
}

func TestWebAppAccelerationPolicyServiceClientKeepsPendingMoveWorkRequestDuringUpdatingReadback(t *testing.T) {
	t.Parallel()

	resource := makeTrackedWebAppAccelerationPolicyResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..replacement"
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:    "wr-move",
		RawOperationType: string(waasdk.WorkRequestOperationTypeMoveWaaPolicy),
		NormalizedClass:  shared.OSOKAsyncClassPending,
	}
	getCalls := 0
	changeCalls := 0
	workRequestCalls := 0
	updateCalls := 0
	client := testWebAppAccelerationPolicyClient(&fakeWebAppAccelerationPolicyOCIClient{
		getFn: func(_ context.Context, request waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error) {
			getCalls++
			requireStringPtr(t, "GetWebAppAccelerationPolicyRequest.WebAppAccelerationPolicyId", request.WebAppAccelerationPolicyId, testWebAppAccelerationPolicyID)
			return waasdk.GetWebAppAccelerationPolicyResponse{
				WebAppAccelerationPolicy: makeSDKWebAppAccelerationPolicy(
					testWebAppAccelerationPolicyID,
					testWebAppAccelerationPolicyCompartment,
					resource.Spec.DisplayName,
					true,
					true,
					waasdk.WebAppAccelerationPolicyLifecycleStateUpdating,
				),
			}, nil
		},
		changeCompartmentFn: func(context.Context, waasdk.ChangeWebAppAccelerationPolicyCompartmentRequest) (waasdk.ChangeWebAppAccelerationPolicyCompartmentResponse, error) {
			changeCalls++
			return waasdk.ChangeWebAppAccelerationPolicyCompartmentResponse{}, nil
		},
		getWorkRequestFn: func(_ context.Context, request waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error) {
			workRequestCalls++
			requireStringPtr(t, "GetWorkRequestRequest.WorkRequestId", request.WorkRequestId, "wr-move")
			return waasdk.GetWorkRequestResponse{
				WorkRequest: makeWebAppAccelerationPolicyWorkRequest(
					"wr-move",
					waasdk.WorkRequestOperationTypeMoveWaaPolicy,
					waasdk.WorkRequestStatusSucceeded,
				),
				OpcRequestId: common.String("opc-workrequest"),
			}, nil
		},
		updateFn: func(context.Context, waasdk.UpdateWebAppAccelerationPolicyRequest) (waasdk.UpdateWebAppAccelerationPolicyResponse, error) {
			updateCalls++
			return waasdk.UpdateWebAppAccelerationPolicyResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeWebAppAccelerationPolicyRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireWebAppAccelerationPolicyMoveReadbackPending(t, response, resource, getCalls, workRequestCalls, changeCalls, updateCalls)
}

func TestWebAppAccelerationPolicyServiceClientRetainsFinalizerWhileDeleteIsPending(t *testing.T) {
	t.Parallel()

	resource := makeTrackedWebAppAccelerationPolicyResource()
	getCalls := 0
	deleteCalls := 0
	client := testWebAppAccelerationPolicyClient(&fakeWebAppAccelerationPolicyOCIClient{
		getFn: func(_ context.Context, request waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error) {
			getCalls++
			requireStringPtr(t, "GetWebAppAccelerationPolicyRequest.WebAppAccelerationPolicyId", request.WebAppAccelerationPolicyId, testWebAppAccelerationPolicyID)
			state := waasdk.WebAppAccelerationPolicyLifecycleStateActive
			if getCalls == 3 {
				state = waasdk.WebAppAccelerationPolicyLifecycleStateDeleting
			}
			return waasdk.GetWebAppAccelerationPolicyResponse{
				WebAppAccelerationPolicy: makeSDKWebAppAccelerationPolicy(
					testWebAppAccelerationPolicyID,
					testWebAppAccelerationPolicyCompartment,
					testWebAppAccelerationPolicyName,
					true,
					true,
					state,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request waasdk.DeleteWebAppAccelerationPolicyRequest) (waasdk.DeleteWebAppAccelerationPolicyResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteWebAppAccelerationPolicyRequest.WebAppAccelerationPolicyId", request.WebAppAccelerationPolicyId, testWebAppAccelerationPolicyID)
			return waasdk.DeleteWebAppAccelerationPolicyResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI lifecycle is DELETING")
	}
	if getCalls != 3 || deleteCalls != 1 {
		t.Fatalf("OCI calls get/delete = %d/%d, want 3/1", getCalls, deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete ||
		resource.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.status.async.current = %#v, want pending delete", resource.Status.OsokStatus.Async.Current)
	}
}

func TestWebAppAccelerationPolicyServiceClientConfirmsDeleteAfterUnambiguousNotFound(t *testing.T) {
	t.Parallel()

	resource := makeTrackedWebAppAccelerationPolicyResource()
	getCalls := 0
	client := testWebAppAccelerationPolicyClient(&fakeWebAppAccelerationPolicyOCIClient{
		getFn: func(context.Context, waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error) {
			getCalls++
			if getCalls < 3 {
				return waasdk.GetWebAppAccelerationPolicyResponse{
					WebAppAccelerationPolicy: makeSDKWebAppAccelerationPolicy(
						testWebAppAccelerationPolicyID,
						testWebAppAccelerationPolicyCompartment,
						testWebAppAccelerationPolicyName,
						true,
						true,
						waasdk.WebAppAccelerationPolicyLifecycleStateActive,
					),
				}, nil
			}
			return waasdk.GetWebAppAccelerationPolicyResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "policy deleted")
		},
		deleteFn: func(context.Context, waasdk.DeleteWebAppAccelerationPolicyRequest) (waasdk.DeleteWebAppAccelerationPolicyResponse, error) {
			return waasdk.DeleteWebAppAccelerationPolicyResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous NotFound confirmation")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
}

func TestWebAppAccelerationPolicyServiceClientRejectsAuthShapedPreDeleteRead(t *testing.T) {
	t.Parallel()

	resource := makeTrackedWebAppAccelerationPolicyResource()
	deleteCalls := 0
	client := testWebAppAccelerationPolicyClient(&fakeWebAppAccelerationPolicyOCIClient{
		getFn: func(context.Context, waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error) {
			return waasdk.GetWebAppAccelerationPolicyResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, waasdk.DeleteWebAppAccelerationPolicyRequest) (waasdk.DeleteWebAppAccelerationPolicyResponse, error) {
			deleteCalls++
			return waasdk.DeleteWebAppAccelerationPolicyResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not-found rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous auth-shaped not-found")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous NotAuthorizedOrNotFound", err)
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteWebAppAccelerationPolicy calls = %d, want 0", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestWebAppAccelerationPolicyServiceClientRecordsOpcRequestIDFromOCIError(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationPolicyResource()
	createErr := errortest.NewServiceError(400, errorutil.InvalidParameter, "invalid policy")
	createErr.OpcRequestID = "opc-create-error"
	client := testWebAppAccelerationPolicyClient(&fakeWebAppAccelerationPolicyOCIClient{
		listFn: func(context.Context, waasdk.ListWebAppAccelerationPoliciesRequest) (waasdk.ListWebAppAccelerationPoliciesResponse, error) {
			return waasdk.ListWebAppAccelerationPoliciesResponse{}, nil
		},
		createFn: func(context.Context, waasdk.CreateWebAppAccelerationPolicyRequest) (waasdk.CreateWebAppAccelerationPolicyResponse, error) {
			return waasdk.CreateWebAppAccelerationPolicyResponse{}, createErr
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeWebAppAccelerationPolicyRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-error" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-error", got)
	}
}

func requireWebAppAccelerationPolicyCreateRequest(
	t *testing.T,
	request waasdk.CreateWebAppAccelerationPolicyRequest,
	resource *waav1beta1.WebAppAccelerationPolicy,
) {
	t.Helper()
	requireStringPtr(t, "CreateWebAppAccelerationPolicyRequest.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "CreateWebAppAccelerationPolicyRequest.DisplayName", request.DisplayName, resource.Spec.DisplayName)
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreateWebAppAccelerationPolicyRequest.OpcRetryToken is empty, want deterministic retry token")
	}
	if request.ResponseCachingPolicy == nil || !boolPtrValue(request.ResponseCachingPolicy.IsResponseHeaderBasedCachingEnabled) {
		t.Fatalf("CreateWebAppAccelerationPolicyRequest.ResponseCachingPolicy = %#v, want enabled", request.ResponseCachingPolicy)
	}
	if request.ResponseCompressionPolicy == nil ||
		request.ResponseCompressionPolicy.GzipCompression == nil ||
		!boolPtrValue(request.ResponseCompressionPolicy.GzipCompression.IsEnabled) {
		t.Fatalf("CreateWebAppAccelerationPolicyRequest.ResponseCompressionPolicy = %#v, want gzip enabled", request.ResponseCompressionPolicy)
	}
}

func requireCreatedWebAppAccelerationPolicyStatus(t *testing.T, resource *waav1beta1.WebAppAccelerationPolicy) {
	t.Helper()
	if resource.Status.Id != testWebAppAccelerationPolicyID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testWebAppAccelerationPolicyID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testWebAppAccelerationPolicyID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testWebAppAccelerationPolicyID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	if resource.Status.LifecycleState != string(waasdk.WebAppAccelerationPolicyLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
}

func requireWebAppAccelerationPolicyMutableUpdateRequest(
	t *testing.T,
	request waasdk.UpdateWebAppAccelerationPolicyRequest,
	resource *waav1beta1.WebAppAccelerationPolicy,
) {
	t.Helper()
	requireStringPtr(t, "UpdateWebAppAccelerationPolicyRequest.WebAppAccelerationPolicyId", request.WebAppAccelerationPolicyId, testWebAppAccelerationPolicyID)
	requireStringPtr(t, "UpdateWebAppAccelerationPolicyRequest.DisplayName", request.DisplayName, resource.Spec.DisplayName)
	if request.ResponseCachingPolicy == nil ||
		boolPtrValue(request.ResponseCachingPolicy.IsResponseHeaderBasedCachingEnabled) {
		t.Fatalf("UpdateWebAppAccelerationPolicyRequest.ResponseCachingPolicy = %#v, want explicit false", request.ResponseCachingPolicy)
	}
	if request.ResponseCompressionPolicy == nil ||
		request.ResponseCompressionPolicy.GzipCompression == nil ||
		boolPtrValue(request.ResponseCompressionPolicy.GzipCompression.IsEnabled) {
		t.Fatalf("UpdateWebAppAccelerationPolicyRequest.ResponseCompressionPolicy = %#v, want explicit gzip false", request.ResponseCompressionPolicy)
	}
	if !reflect.DeepEqual(request.FreeformTags, map[string]string{}) {
		t.Fatalf("UpdateWebAppAccelerationPolicyRequest.FreeformTags = %#v, want empty map clear", request.FreeformTags)
	}
	if !reflect.DeepEqual(request.DefinedTags, map[string]map[string]interface{}{}) {
		t.Fatalf("UpdateWebAppAccelerationPolicyRequest.DefinedTags = %#v, want empty map clear", request.DefinedTags)
	}
	if !reflect.DeepEqual(request.SystemTags, map[string]map[string]interface{}{}) {
		t.Fatalf("UpdateWebAppAccelerationPolicyRequest.SystemTags = %#v, want empty map clear", request.SystemTags)
	}
}

func requireUpdatedWebAppAccelerationPolicyStatus(t *testing.T, resource *waav1beta1.WebAppAccelerationPolicy) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	if resource.Status.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, resource.Spec.DisplayName)
	}
}

func requireWebAppAccelerationPolicyMoveStarted(
	t *testing.T,
	response servicemanager.OSOKResponse,
	resource *waav1beta1.WebAppAccelerationPolicy,
	getCalls int,
	changeCalls int,
	workRequestCalls int,
	updateCalls int,
) {
	t.Helper()
	requireWebAppAccelerationPolicyMoveStartResponse(t, response)
	requireWebAppAccelerationPolicyMoveStartCalls(t, getCalls, changeCalls, workRequestCalls, updateCalls)
	requireWebAppAccelerationPolicyTrackedIdentity(t, resource)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-move" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-move", got)
	}
	requireWebAppAccelerationPolicyPendingMoveAsync(t, resource)
}

func requireWebAppAccelerationPolicyMoveStartResponse(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate().ShouldRequeue = false, want true while compartment move work request is pending")
	}
}

func requireWebAppAccelerationPolicyMoveStartCalls(
	t *testing.T,
	getCalls int,
	changeCalls int,
	workRequestCalls int,
	updateCalls int,
) {
	t.Helper()
	if updateCalls != 0 {
		t.Fatalf("UpdateWebAppAccelerationPolicy calls = %d, want 0", updateCalls)
	}
	if getCalls != 1 || changeCalls != 1 || workRequestCalls != 0 {
		t.Fatalf("OCI calls get/change/workRequest = %d/%d/%d, want 1/1/0", getCalls, changeCalls, workRequestCalls)
	}
}

func requireWebAppAccelerationPolicyTrackedIdentity(t *testing.T, resource *waav1beta1.WebAppAccelerationPolicy) {
	t.Helper()
	if got := resource.Status.Id; got != testWebAppAccelerationPolicyID {
		t.Fatalf("status.id = %q, want %q", got, testWebAppAccelerationPolicyID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testWebAppAccelerationPolicyID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testWebAppAccelerationPolicyID)
	}
}

func requireWebAppAccelerationPolicyPendingMoveAsync(t *testing.T, resource *waav1beta1.WebAppAccelerationPolicy) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want pending update work request")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.status.async.current.source = %q, want workRequest", current.Source)
	}
	if current.Phase != shared.OSOKAsyncPhaseUpdate {
		t.Fatalf("status.status.async.current.phase = %q, want update", current.Phase)
	}
	if current.WorkRequestID != "wr-move" {
		t.Fatalf("status.status.async.current.workRequestId = %q, want wr-move", current.WorkRequestID)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.status.async.current.normalizedClass = %q, want pending", current.NormalizedClass)
	}
}

func requireWebAppAccelerationPolicyMoveCompleted(
	t *testing.T,
	response servicemanager.OSOKResponse,
	resource *waav1beta1.WebAppAccelerationPolicy,
	getCalls int,
	workRequestCalls int,
	changeCalls int,
	updateCalls int,
) {
	t.Helper()
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate().ShouldRequeue = true, want false after compartment move completes")
	}
	if getCalls != 1 || workRequestCalls != 1 || changeCalls != 0 || updateCalls != 0 {
		t.Fatalf("OCI calls get/workRequest/change/update = %d/%d/%d/%d, want 1/1/0/0", getCalls, workRequestCalls, changeCalls, updateCalls)
	}
	if got := resource.Status.CompartmentId; got != resource.Spec.CompartmentId {
		t.Fatalf("status.compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-workrequest" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-workrequest", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want cleared tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func requireWebAppAccelerationPolicyMoveReadbackPending(
	t *testing.T,
	response servicemanager.OSOKResponse,
	resource *waav1beta1.WebAppAccelerationPolicy,
	getCalls int,
	workRequestCalls int,
	changeCalls int,
	updateCalls int,
) {
	t.Helper()
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate().ShouldRequeue = false, want true while move readback is still pending")
	}
	if getCalls != 1 || workRequestCalls != 1 || changeCalls != 0 || updateCalls != 0 {
		t.Fatalf("OCI calls get/workRequest/change/update = %d/%d/%d/%d, want 1/1/0/0", getCalls, workRequestCalls, changeCalls, updateCalls)
	}
	if got := resource.Status.CompartmentId; got != testWebAppAccelerationPolicyCompartment {
		t.Fatalf("status.compartmentId = %q, want old compartment while readback is still pending", got)
	}
	if got := resource.Status.LifecycleState; got != string(waasdk.WebAppAccelerationPolicyLifecycleStateUpdating) {
		t.Fatalf("status.lifecycleState = %q, want UPDATING", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-workrequest" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-workrequest", got)
	}
	requireWebAppAccelerationPolicyPendingMoveAsync(t, resource)
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

func requireStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func boolPtrValue(value *bool) bool {
	return value != nil && *value
}
