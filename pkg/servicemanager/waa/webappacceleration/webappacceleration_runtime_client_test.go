/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package webappacceleration

import (
	"context"
	"crypto/rsa"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	waasdk "github.com/oracle/oci-go-sdk/v65/waa"
	waav1beta1 "github.com/oracle/oci-service-operator/api/waa/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testWebAppAccelerationID         = "ocid1.webappacceleration.oc1..example"
	testWebAppAccelerationOtherID    = "ocid1.webappacceleration.oc1..other"
	testWebAppAccelerationCompID     = "ocid1.compartment.oc1..example"
	testWebAppAccelerationMoveCompID = "ocid1.compartment.oc1..moved"
	testWebAppAccelerationPolicyID   = "ocid1.waapolicy.oc1..example"
	testWebAppAccelerationPolicyID2  = "ocid1.waapolicy.oc1..replacement"
	testWebAppAccelerationLBID       = "ocid1.loadbalancer.oc1..example"
	testWebAppAccelerationOtherLBID  = "ocid1.loadbalancer.oc1..other"
)

type erroringWebAppAccelerationConfigProvider struct {
	calls int
}

func (p *erroringWebAppAccelerationConfigProvider) PrivateRSAKey() (*rsa.PrivateKey, error) {
	p.calls++
	return nil, errors.New("web app acceleration provider invalid")
}

func (p *erroringWebAppAccelerationConfigProvider) KeyID() (string, error) {
	p.calls++
	return "", errors.New("web app acceleration provider invalid")
}

func (p *erroringWebAppAccelerationConfigProvider) TenancyOCID() (string, error) {
	p.calls++
	return "", errors.New("web app acceleration provider invalid")
}

func (p *erroringWebAppAccelerationConfigProvider) UserOCID() (string, error) {
	p.calls++
	return "", errors.New("web app acceleration provider invalid")
}

func (p *erroringWebAppAccelerationConfigProvider) KeyFingerprint() (string, error) {
	p.calls++
	return "", errors.New("web app acceleration provider invalid")
}

func (p *erroringWebAppAccelerationConfigProvider) Region() (string, error) {
	p.calls++
	return "", errors.New("web app acceleration provider invalid")
}

func (p *erroringWebAppAccelerationConfigProvider) AuthType() (common.AuthConfig, error) {
	return common.AuthConfig{}, nil
}

type fakeWebAppAccelerationOCIClient struct {
	changeFn func(context.Context, waasdk.ChangeWebAppAccelerationCompartmentRequest) (waasdk.ChangeWebAppAccelerationCompartmentResponse, error)
	createFn func(context.Context, waasdk.CreateWebAppAccelerationRequest) (waasdk.CreateWebAppAccelerationResponse, error)
	getFn    func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error)
	workFn   func(context.Context, waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error)
	listFn   func(context.Context, waasdk.ListWebAppAccelerationsRequest) (waasdk.ListWebAppAccelerationsResponse, error)
	updateFn func(context.Context, waasdk.UpdateWebAppAccelerationRequest) (waasdk.UpdateWebAppAccelerationResponse, error)
	deleteFn func(context.Context, waasdk.DeleteWebAppAccelerationRequest) (waasdk.DeleteWebAppAccelerationResponse, error)
}

func (f *fakeWebAppAccelerationOCIClient) ChangeWebAppAccelerationCompartment(
	ctx context.Context,
	request waasdk.ChangeWebAppAccelerationCompartmentRequest,
) (waasdk.ChangeWebAppAccelerationCompartmentResponse, error) {
	if f.changeFn != nil {
		return f.changeFn(ctx, request)
	}
	return waasdk.ChangeWebAppAccelerationCompartmentResponse{}, nil
}

func (f *fakeWebAppAccelerationOCIClient) CreateWebAppAcceleration(
	ctx context.Context,
	request waasdk.CreateWebAppAccelerationRequest,
) (waasdk.CreateWebAppAccelerationResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return waasdk.CreateWebAppAccelerationResponse{}, nil
}

func (f *fakeWebAppAccelerationOCIClient) GetWebAppAcceleration(
	ctx context.Context,
	request waasdk.GetWebAppAccelerationRequest,
) (waasdk.GetWebAppAccelerationResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return waasdk.GetWebAppAccelerationResponse{}, nil
}

func (f *fakeWebAppAccelerationOCIClient) GetWorkRequest(
	ctx context.Context,
	request waasdk.GetWorkRequestRequest,
) (waasdk.GetWorkRequestResponse, error) {
	if f.workFn != nil {
		return f.workFn(ctx, request)
	}
	return waasdk.GetWorkRequestResponse{}, nil
}

func (f *fakeWebAppAccelerationOCIClient) ListWebAppAccelerations(
	ctx context.Context,
	request waasdk.ListWebAppAccelerationsRequest,
) (waasdk.ListWebAppAccelerationsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return waasdk.ListWebAppAccelerationsResponse{}, nil
}

func (f *fakeWebAppAccelerationOCIClient) UpdateWebAppAcceleration(
	ctx context.Context,
	request waasdk.UpdateWebAppAccelerationRequest,
) (waasdk.UpdateWebAppAccelerationResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return waasdk.UpdateWebAppAccelerationResponse{}, nil
}

func (f *fakeWebAppAccelerationOCIClient) DeleteWebAppAcceleration(
	ctx context.Context,
	request waasdk.DeleteWebAppAccelerationRequest,
) (waasdk.DeleteWebAppAccelerationResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return waasdk.DeleteWebAppAccelerationResponse{}, nil
}

func testWebAppAccelerationClient(fake *fakeWebAppAccelerationOCIClient) WebAppAccelerationServiceClient {
	return newWebAppAccelerationServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeWebAppAccelerationResource() *waav1beta1.WebAppAcceleration {
	return &waav1beta1.WebAppAcceleration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "accel-alpha",
			Namespace: "default",
			UID:       types.UID("webappacceleration-uid"),
		},
		Spec: waav1beta1.WebAppAccelerationSpec{
			CompartmentId:              testWebAppAccelerationCompID,
			WebAppAccelerationPolicyId: testWebAppAccelerationPolicyID,
			BackendType:                webAppAccelerationLoadBalancerBackendType,
			LoadBalancerId:             testWebAppAccelerationLBID,
			DisplayName:                "accel-alpha",
			FreeformTags:               map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
			SystemTags: map[string]shared.MapValue{
				"orcl-cloud": {"free-tier-retained": "true"},
			},
		},
	}
}

func makeWebAppAccelerationJSONResource() *waav1beta1.WebAppAcceleration {
	resource := makeWebAppAccelerationResource()
	resource.Spec.BackendType = ""
	resource.Spec.LoadBalancerId = ""
	resource.Spec.JsonData = `{"backendType":"LOAD_BALANCER","compartmentId":"` + testWebAppAccelerationCompID +
		`","webAppAccelerationPolicyId":"` + testWebAppAccelerationPolicyID +
		`","loadBalancerId":"` + testWebAppAccelerationLBID +
		`","displayName":"` + resource.Spec.DisplayName + `","freeformTags":{"env":"dev"}}`
	return resource
}

func makeSDKWebAppAcceleration(
	id string,
	displayName string,
	policyID string,
	loadBalancerID string,
	state waasdk.WebAppAccelerationLifecycleStateEnum,
) waasdk.WebAppAccelerationLoadBalancer {
	return makeSDKWebAppAccelerationInCompartment(id, displayName, policyID, loadBalancerID, testWebAppAccelerationCompID, state)
}

func makeSDKWebAppAccelerationInCompartment(
	id string,
	displayName string,
	policyID string,
	loadBalancerID string,
	compartmentID string,
	state waasdk.WebAppAccelerationLifecycleStateEnum,
) waasdk.WebAppAccelerationLoadBalancer {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return waasdk.WebAppAccelerationLoadBalancer{
		Id:                         common.String(id),
		DisplayName:                common.String(displayName),
		CompartmentId:              common.String(compartmentID),
		WebAppAccelerationPolicyId: common.String(policyID),
		LoadBalancerId:             common.String(loadBalancerID),
		TimeCreated:                &created,
		FreeformTags:               map[string]string{"env": "dev"},
		DefinedTags:                map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:                 map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
		LifecycleState:             state,
	}
}

func makeSDKWebAppAccelerationSummary(
	id string,
	displayName string,
	policyID string,
	loadBalancerID string,
	state waasdk.WebAppAccelerationLifecycleStateEnum,
) waasdk.WebAppAccelerationLoadBalancerSummary {
	created := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return waasdk.WebAppAccelerationLoadBalancerSummary{
		Id:                         common.String(id),
		DisplayName:                common.String(displayName),
		CompartmentId:              common.String(testWebAppAccelerationCompID),
		WebAppAccelerationPolicyId: common.String(policyID),
		LoadBalancerId:             common.String(loadBalancerID),
		TimeCreated:                &created,
		FreeformTags:               map[string]string{"env": "dev"},
		DefinedTags:                map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:                 map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
		LifecycleState:             state,
	}
}

func makeWebAppAccelerationWorkRequest(
	id string,
	operationType waasdk.WorkRequestOperationTypeEnum,
	status waasdk.WorkRequestStatusEnum,
) waasdk.WorkRequest {
	percentComplete := float32(50)
	return waasdk.WorkRequest{
		Id:              common.String(id),
		OperationType:   operationType,
		Status:          status,
		PercentComplete: &percentComplete,
	}
}

func TestWebAppAccelerationRuntimeSemantics(t *testing.T) {
	t.Parallel()

	got := reviewedWebAppAccelerationRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedWebAppAccelerationRuntimeSemantics() = nil")
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
	requireStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{
		"compartmentId",
		"webAppAccelerationPolicyId",
		"backendType",
		"loadBalancerId",
		"displayName",
		"id",
	})
	requireStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{
		"displayName",
		"webAppAccelerationPolicyId",
		"compartmentId",
		"freeformTags",
		"definedTags",
		"systemTags",
	})
	requireStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{
		"backendType",
		"loadBalancerId",
	})
}

func TestWebAppAccelerationCreateOrUpdatePreservesGeneratedOCIInitErrorWhenWrapped(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	provider := &erroringWebAppAccelerationConfigProvider{}
	client := newWebAppAccelerationServiceClient(&WebAppAccelerationServiceManager{
		Provider: provider,
		Log:      loggerutil.OSOKLogger{Logger: logr.Discard()},
	})
	callsAfterInit := provider.calls

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assertWebAppAccelerationInitError(t, err)
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status.status.reason = %q, want Failed", resource.Status.OsokStatus.Reason)
	}
	assertWebAppAccelerationProviderCalls(t, provider, callsAfterInit)
}

func TestWebAppAccelerationDeletePreservesGeneratedOCIInitErrorWhenWrapped(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	provider := &erroringWebAppAccelerationConfigProvider{}
	client := newWebAppAccelerationServiceClient(&WebAppAccelerationServiceManager{
		Provider: provider,
		Log:      loggerutil.OSOKLogger{Logger: logr.Discard()},
	})
	callsAfterInit := provider.calls

	deleted, err := client.Delete(context.Background(), resource)
	assertWebAppAccelerationInitError(t, err)
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	assertWebAppAccelerationProviderCalls(t, provider, callsAfterInit)
}

func TestWebAppAccelerationServiceClientCreatesLoadBalancerAndProjectsStatus(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	var createRequest waasdk.CreateWebAppAccelerationRequest
	listCalls := 0
	createCalls := 0
	getCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		listFn: func(_ context.Context, request waasdk.ListWebAppAccelerationsRequest) (waasdk.ListWebAppAccelerationsResponse, error) {
			listCalls++
			requireStringPtr(t, "list compartmentId", request.CompartmentId, testWebAppAccelerationCompID)
			requireStringPtr(t, "list policyId", request.WebAppAccelerationPolicyId, testWebAppAccelerationPolicyID)
			return waasdk.ListWebAppAccelerationsResponse{
				WebAppAccelerationCollection: waasdk.WebAppAccelerationCollection{},
				OpcRequestId:                 common.String("opc-list"),
			}, nil
		},
		createFn: func(_ context.Context, request waasdk.CreateWebAppAccelerationRequest) (waasdk.CreateWebAppAccelerationResponse, error) {
			createCalls++
			createRequest = request
			return waasdk.CreateWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					waasdk.WebAppAccelerationLifecycleStateCreating,
				),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			getCalls++
			requireStringPtr(t, "get webAppAccelerationId", request.WebAppAccelerationId, testWebAppAccelerationID)
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					waasdk.WebAppAccelerationLifecycleStateActive,
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
	assertWebAppAccelerationCreateRequest(t, resource, createRequest)
	requireStringPtr(t, "create retry token", createRequest.OpcRetryToken, string(resource.UID))
	assertWebAppAccelerationActiveStatus(t, resource, testWebAppAccelerationID, "opc-create")
	if !strings.Contains(resource.Status.JsonData, `"backendType":"LOAD_BALANCER"`) {
		t.Fatalf("status.jsonData = %q, want backend discriminator", resource.Status.JsonData)
	}
}

func TestWebAppAccelerationServiceClientRecordsCreateWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	createCalls := 0
	getCalls := 0
	workCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		listFn: func(context.Context, waasdk.ListWebAppAccelerationsRequest) (waasdk.ListWebAppAccelerationsResponse, error) {
			return waasdk.ListWebAppAccelerationsResponse{}, nil
		},
		createFn: func(context.Context, waasdk.CreateWebAppAccelerationRequest) (waasdk.CreateWebAppAccelerationResponse, error) {
			createCalls++
			return waasdk.CreateWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					waasdk.WebAppAccelerationLifecycleStateCreating,
				),
				OpcRequestId:     common.String("opc-create"),
				OpcWorkRequestId: common.String("wr-create"),
			}, nil
		},
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			getCalls++
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					waasdk.WebAppAccelerationLifecycleStateCreating,
				),
			}, nil
		},
		workFn: func(_ context.Context, request waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error) {
			workCalls++
			requireStringPtr(t, "get workRequestId", request.WorkRequestId, "wr-create")
			return waasdk.GetWorkRequestResponse{
				WorkRequest: makeWebAppAccelerationWorkRequest(
					"wr-create",
					waasdk.WorkRequestOperationTypeCreateWebAppAcceleration,
					waasdk.WorkRequestStatusAccepted,
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while create is pending", response)
	}
	if createCalls != 1 || getCalls != 1 || workCalls != 1 {
		t.Fatalf("call counts create/get/workRequest = %d/%d/%d, want 1/1/1", createCalls, getCalls, workCalls)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", resource.Status.OsokStatus.OpcRequestID)
	}
	assertWebAppAccelerationCreateAsyncWithWorkRequestStatus(
		t,
		resource.Status.OsokStatus.Async.Current,
		waasdk.WorkRequestStatusAccepted,
		shared.OSOKAsyncClassPending,
	)
}

func TestWebAppAccelerationServiceClientCreatesLoadBalancerFromJsonData(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationJSONResource()
	var createRequest waasdk.CreateWebAppAccelerationRequest

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		listFn: func(_ context.Context, request waasdk.ListWebAppAccelerationsRequest) (waasdk.ListWebAppAccelerationsResponse, error) {
			requireStringPtr(t, "list compartmentId", request.CompartmentId, testWebAppAccelerationCompID)
			requireStringPtr(t, "list policyId", request.WebAppAccelerationPolicyId, testWebAppAccelerationPolicyID)
			return waasdk.ListWebAppAccelerationsResponse{}, nil
		},
		createFn: func(_ context.Context, request waasdk.CreateWebAppAccelerationRequest) (waasdk.CreateWebAppAccelerationResponse, error) {
			createRequest = request
			return waasdk.CreateWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					testWebAppAccelerationLBID,
					waasdk.WebAppAccelerationLifecycleStateCreating,
				),
				OpcRequestId: common.String("opc-create"),
			}, nil
		},
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					testWebAppAccelerationLBID,
					waasdk.WebAppAccelerationLifecycleStateActive,
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
	details, ok := createRequest.CreateWebAppAccelerationDetails.(waasdk.CreateWebAppAccelerationLoadBalancerDetails)
	if !ok {
		t.Fatalf("create details type = %T, want CreateWebAppAccelerationLoadBalancerDetails", createRequest.CreateWebAppAccelerationDetails)
	}
	requireStringPtr(t, "create loadBalancerId", details.LoadBalancerId, testWebAppAccelerationLBID)
	assertWebAppAccelerationActiveStatus(t, resource, testWebAppAccelerationID, "opc-create")
}

func TestWebAppAccelerationServiceClientBindsFromPaginatedListWithoutCreate(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	listPages := []waasdk.ListWebAppAccelerationsResponse{
		{
			WebAppAccelerationCollection: waasdk.WebAppAccelerationCollection{
				Items: []waasdk.WebAppAccelerationSummary{
					makeSDKWebAppAccelerationSummary(
						testWebAppAccelerationOtherID,
						resource.Spec.DisplayName,
						resource.Spec.WebAppAccelerationPolicyId,
						testWebAppAccelerationOtherLBID,
						waasdk.WebAppAccelerationLifecycleStateActive,
					),
				},
			},
			OpcNextPage: common.String("page-2"),
		},
		{
			WebAppAccelerationCollection: waasdk.WebAppAccelerationCollection{
				Items: []waasdk.WebAppAccelerationSummary{
					makeSDKWebAppAccelerationSummary(
						testWebAppAccelerationID,
						resource.Spec.DisplayName,
						resource.Spec.WebAppAccelerationPolicyId,
						resource.Spec.LoadBalancerId,
						waasdk.WebAppAccelerationLifecycleStateActive,
					),
				},
			},
			OpcRequestId: common.String("opc-list-2"),
		},
	}
	listCalls := 0
	createCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		listFn: func(_ context.Context, request waasdk.ListWebAppAccelerationsRequest) (waasdk.ListWebAppAccelerationsResponse, error) {
			if listCalls == 0 && request.Page != nil {
				t.Fatalf("first list page = %q, want nil", *request.Page)
			}
			if listCalls == 1 {
				requireStringPtr(t, "second list page", request.Page, "page-2")
			}
			if listCalls >= len(listPages) {
				t.Fatalf("unexpected extra list call %d", listCalls+1)
			}
			response := listPages[listCalls]
			listCalls++
			return response, nil
		},
		createFn: func(context.Context, waasdk.CreateWebAppAccelerationRequest) (waasdk.CreateWebAppAccelerationResponse, error) {
			createCalls++
			return waasdk.CreateWebAppAccelerationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue bind", response)
	}
	if listCalls != 2 {
		t.Fatalf("list calls = %d, want 2", listCalls)
	}
	if createCalls != 0 {
		t.Fatalf("create calls = %d, want 0 after bind", createCalls)
	}
	assertWebAppAccelerationActiveStatus(t, resource, testWebAppAccelerationID, "opc-list-2")
}

func TestWebAppAccelerationServiceClientBindsUsingJsonDataLoadBalancerID(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationJSONResource()
	createCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		listFn: func(context.Context, waasdk.ListWebAppAccelerationsRequest) (waasdk.ListWebAppAccelerationsResponse, error) {
			return waasdk.ListWebAppAccelerationsResponse{
				WebAppAccelerationCollection: waasdk.WebAppAccelerationCollection{
					Items: []waasdk.WebAppAccelerationSummary{
						makeSDKWebAppAccelerationSummary(
							testWebAppAccelerationID,
							resource.Spec.DisplayName,
							resource.Spec.WebAppAccelerationPolicyId,
							testWebAppAccelerationLBID,
							waasdk.WebAppAccelerationLifecycleStateActive,
						),
					},
				},
				OpcRequestId: common.String("opc-list"),
			}, nil
		},
		createFn: func(context.Context, waasdk.CreateWebAppAccelerationRequest) (waasdk.CreateWebAppAccelerationResponse, error) {
			createCalls++
			return waasdk.CreateWebAppAccelerationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue bind", response)
	}
	if createCalls != 0 {
		t.Fatalf("create calls = %d, want 0 after jsonData loadBalancerId bind", createCalls)
	}
	assertWebAppAccelerationActiveStatus(t, resource, testWebAppAccelerationID, "opc-list")
}

func TestWebAppAccelerationServiceClientNoopsWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	getCalls := 0
	updateCalls := 0
	createCalls := 0
	listCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(_ context.Context, request waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			getCalls++
			requireStringPtr(t, "get webAppAccelerationId", request.WebAppAccelerationId, testWebAppAccelerationID)
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					waasdk.WebAppAccelerationLifecycleStateActive,
				),
				OpcRequestId: common.String("opc-get"),
			}, nil
		},
		updateFn: func(context.Context, waasdk.UpdateWebAppAccelerationRequest) (waasdk.UpdateWebAppAccelerationResponse, error) {
			updateCalls++
			return waasdk.UpdateWebAppAccelerationResponse{}, nil
		},
		createFn: func(context.Context, waasdk.CreateWebAppAccelerationRequest) (waasdk.CreateWebAppAccelerationResponse, error) {
			createCalls++
			return waasdk.CreateWebAppAccelerationResponse{}, nil
		},
		listFn: func(context.Context, waasdk.ListWebAppAccelerationsRequest) (waasdk.ListWebAppAccelerationsResponse, error) {
			listCalls++
			return waasdk.ListWebAppAccelerationsResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no-op", response)
	}
	if getCalls != 1 || updateCalls != 0 || createCalls != 0 || listCalls != 0 {
		t.Fatalf("call counts get/update/create/list = %d/%d/%d/%d, want 1/0/0/0", getCalls, updateCalls, createCalls, listCalls)
	}
	assertWebAppAccelerationActiveStatus(t, resource, testWebAppAccelerationID, "opc-get")
}

func TestWebAppAccelerationServiceClientDoesNotDriftWhenLoadBalancerIDComesFromJsonData(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationJSONResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	updateCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					testWebAppAccelerationLBID,
					waasdk.WebAppAccelerationLifecycleStateActive,
				),
				OpcRequestId: common.String("opc-get"),
			}, nil
		},
		updateFn: func(context.Context, waasdk.UpdateWebAppAccelerationRequest) (waasdk.UpdateWebAppAccelerationResponse, error) {
			updateCalls++
			return waasdk.UpdateWebAppAccelerationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no-op", response)
	}
	if updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0 when jsonData loadBalancerId matches observed state", updateCalls)
	}
	assertWebAppAccelerationActiveStatus(t, resource, testWebAppAccelerationID, "opc-get")
}

func TestWebAppAccelerationServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	resource.Spec.DisplayName = "accel-renamed"
	resource.Spec.WebAppAccelerationPolicyId = testWebAppAccelerationPolicyID2
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	resource.Spec.SystemTags = map[string]shared.MapValue{"orcl-cloud": {"free-tier-retained": "false"}}

	getCalls := 0
	updateCalls := 0
	var updateRequest waasdk.UpdateWebAppAccelerationRequest

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(_ context.Context, request waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			getCalls++
			requireStringPtr(t, "get webAppAccelerationId", request.WebAppAccelerationId, testWebAppAccelerationID)
			if getCalls == 1 {
				current := makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					"accel-alpha",
					testWebAppAccelerationPolicyID,
					resource.Spec.LoadBalancerId,
					waasdk.WebAppAccelerationLifecycleStateActive,
				)
				current.FreeformTags = map[string]string{"env": "dev"}
				current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}}
				current.SystemTags = map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}}
				return waasdk.GetWebAppAccelerationResponse{WebAppAcceleration: current}, nil
			}
			current := makeSDKWebAppAcceleration(
				testWebAppAccelerationID,
				resource.Spec.DisplayName,
				resource.Spec.WebAppAccelerationPolicyId,
				resource.Spec.LoadBalancerId,
				waasdk.WebAppAccelerationLifecycleStateActive,
			)
			current.FreeformTags = map[string]string{"env": "prod"}
			current.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "99"}}
			current.SystemTags = map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "false"}}
			return waasdk.GetWebAppAccelerationResponse{WebAppAcceleration: current}, nil
		},
		updateFn: func(_ context.Context, request waasdk.UpdateWebAppAccelerationRequest) (waasdk.UpdateWebAppAccelerationResponse, error) {
			updateCalls++
			updateRequest = request
			return waasdk.UpdateWebAppAccelerationResponse{OpcRequestId: common.String("opc-update")}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful update", response)
	}
	if getCalls != 2 || updateCalls != 1 {
		t.Fatalf("call counts get/update = %d/%d, want 2/1", getCalls, updateCalls)
	}
	assertWebAppAccelerationUpdateRequest(t, updateRequest)
	if resource.Status.OsokStatus.OpcRequestID != "opc-update" {
		t.Fatalf("status.opcRequestId = %q, want opc-update", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestWebAppAccelerationServiceClientDoesNotRepeatPendingUpdateWorkRequest(t *testing.T) {
	t.Parallel()

	fixture := newPendingWebAppAccelerationUpdateFixture(t)
	response, err := fixture.client.CreateOrUpdate(context.Background(), fixture.resource, ctrl.Request{})
	fixture.requireFirstReconcile(response, err)

	response, err = fixture.client.CreateOrUpdate(context.Background(), fixture.resource, ctrl.Request{})
	fixture.requireSecondReconcile(response, err)
}

type pendingWebAppAccelerationUpdateFixture struct {
	t           *testing.T
	resource    *waav1beta1.WebAppAcceleration
	client      WebAppAccelerationServiceClient
	getCalls    int
	workCalls   int
	updateCalls int
}

func newPendingWebAppAccelerationUpdateFixture(t *testing.T) *pendingWebAppAccelerationUpdateFixture {
	t.Helper()

	resource := makeWebAppAccelerationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	resource.Spec.DisplayName = "accel-renamed"

	fixture := &pendingWebAppAccelerationUpdateFixture{t: t, resource: resource}
	fixture.client = testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn:    fixture.get,
		workFn:   fixture.workRequest,
		updateFn: fixture.update,
	})
	return fixture
}

func (f *pendingWebAppAccelerationUpdateFixture) get(
	_ context.Context,
	request waasdk.GetWebAppAccelerationRequest,
) (waasdk.GetWebAppAccelerationResponse, error) {
	f.t.Helper()

	f.getCalls++
	requireStringPtr(f.t, "get webAppAccelerationId", request.WebAppAccelerationId, testWebAppAccelerationID)
	return waasdk.GetWebAppAccelerationResponse{
		WebAppAcceleration: makeSDKWebAppAcceleration(
			testWebAppAccelerationID,
			"accel-alpha",
			f.resource.Spec.WebAppAccelerationPolicyId,
			f.resource.Spec.LoadBalancerId,
			waasdk.WebAppAccelerationLifecycleStateActive,
		),
	}, nil
}

func (f *pendingWebAppAccelerationUpdateFixture) workRequest(
	_ context.Context,
	request waasdk.GetWorkRequestRequest,
) (waasdk.GetWorkRequestResponse, error) {
	f.t.Helper()

	f.workCalls++
	requireStringPtr(f.t, "get workRequestId", request.WorkRequestId, "wr-update")
	return waasdk.GetWorkRequestResponse{
		WorkRequest: makeWebAppAccelerationWorkRequest(
			"wr-update",
			waasdk.WorkRequestOperationTypeUpdateWebAppAcceleration,
			waasdk.WorkRequestStatusInProgress,
		),
	}, nil
}

func (f *pendingWebAppAccelerationUpdateFixture) update(
	context.Context,
	waasdk.UpdateWebAppAccelerationRequest,
) (waasdk.UpdateWebAppAccelerationResponse, error) {
	f.updateCalls++
	return waasdk.UpdateWebAppAccelerationResponse{
		OpcRequestId:     common.String("opc-update"),
		OpcWorkRequestId: common.String("wr-update"),
	}, nil
}

func (f *pendingWebAppAccelerationUpdateFixture) requireFirstReconcile(
	response servicemanager.OSOKResponse,
	err error,
) {
	f.t.Helper()

	if err != nil {
		f.t.Fatalf("first CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		f.t.Fatalf("first CreateOrUpdate() response = %#v, want successful requeue while update readback is stale", response)
	}
	if f.getCalls != 2 || f.updateCalls != 1 || f.workCalls != 1 {
		f.t.Fatalf(
			"first call counts get/update/workRequest = %d/%d/%d, want 2/1/1",
			f.getCalls,
			f.updateCalls,
			f.workCalls,
		)
	}
	if f.resource.Status.OsokStatus.OpcRequestID != "opc-update" {
		f.t.Fatalf("status.status.opcRequestId = %q, want opc-update", f.resource.Status.OsokStatus.OpcRequestID)
	}
	assertWebAppAccelerationUpdateAsyncWithWorkRequestStatus(
		f.t,
		f.resource.Status.OsokStatus.Async.Current,
		waasdk.WorkRequestStatusInProgress,
		shared.OSOKAsyncClassPending,
	)
}

func (f *pendingWebAppAccelerationUpdateFixture) requireSecondReconcile(
	response servicemanager.OSOKResponse,
	err error,
) {
	f.t.Helper()

	if err != nil {
		f.t.Fatalf("second CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		f.t.Fatalf("second CreateOrUpdate() response = %#v, want successful requeue while update work request remains pending", response)
	}
	if f.getCalls != 3 || f.updateCalls != 1 || f.workCalls != 2 {
		f.t.Fatalf(
			"second call counts get/update/workRequest = %d/%d/%d, want 3/1/2",
			f.getCalls,
			f.updateCalls,
			f.workCalls,
		)
	}
	assertWebAppAccelerationUpdateAsyncWithWorkRequestStatus(
		f.t,
		f.resource.Status.OsokStatus.Async.Current,
		waasdk.WorkRequestStatusInProgress,
		shared.OSOKAsyncClassPending,
	)
}

func TestWebAppAccelerationServiceClientFailsPendingUpdateWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:    "wr-update",
		RawOperationType: string(waasdk.WorkRequestOperationTypeUpdateWebAppAcceleration),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		UpdatedAt:        &metav1.Time{},
	}
	resource.Spec.DisplayName = "accel-renamed"
	workCalls := 0
	updateCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					"accel-alpha",
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					waasdk.WebAppAccelerationLifecycleStateActive,
				),
			}, nil
		},
		workFn: func(_ context.Context, request waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error) {
			workCalls++
			requireStringPtr(t, "get workRequestId", request.WorkRequestId, "wr-update")
			return waasdk.GetWorkRequestResponse{
				WorkRequest: makeWebAppAccelerationWorkRequest(
					"wr-update",
					waasdk.WorkRequestOperationTypeUpdateWebAppAcceleration,
					waasdk.WorkRequestStatusFailed,
				),
			}, nil
		},
		updateFn: func(context.Context, waasdk.UpdateWebAppAccelerationRequest) (waasdk.UpdateWebAppAccelerationResponse, error) {
			updateCalls++
			return waasdk.UpdateWebAppAccelerationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "update work request wr-update finished with status FAILED") {
		t.Fatalf("CreateOrUpdate() error = %v, want failed update work request", err)
	}
	if response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful non-requeue", response)
	}
	if workCalls != 1 || updateCalls != 0 {
		t.Fatalf("call counts workRequest/update = %d/%d, want 1/0", workCalls, updateCalls)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", resource.Status.OsokStatus.Reason)
	}
	assertWebAppAccelerationUpdateAsyncWithWorkRequestStatus(
		t,
		resource.Status.OsokStatus.Async.Current,
		waasdk.WorkRequestStatusFailed,
		shared.OSOKAsyncClassFailed,
	)
}

func TestWebAppAccelerationServiceClientKeepsSucceededUpdateWorkRequestPendingForStaleReadback(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:    "wr-update",
		RawOperationType: string(waasdk.WorkRequestOperationTypeUpdateWebAppAcceleration),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		UpdatedAt:        &metav1.Time{},
	}
	resource.Spec.DisplayName = "accel-renamed"
	workCalls := 0
	updateCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					"accel-alpha",
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					waasdk.WebAppAccelerationLifecycleStateActive,
				),
			}, nil
		},
		workFn: func(_ context.Context, request waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error) {
			workCalls++
			requireStringPtr(t, "get workRequestId", request.WorkRequestId, "wr-update")
			return waasdk.GetWorkRequestResponse{
				WorkRequest: makeWebAppAccelerationWorkRequest(
					"wr-update",
					waasdk.WorkRequestOperationTypeUpdateWebAppAcceleration,
					waasdk.WorkRequestStatusSucceeded,
				),
			}, nil
		},
		updateFn: func(context.Context, waasdk.UpdateWebAppAccelerationRequest) (waasdk.UpdateWebAppAccelerationResponse, error) {
			updateCalls++
			return waasdk.UpdateWebAppAccelerationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue for stale readback", response)
	}
	if workCalls != 1 || updateCalls != 0 {
		t.Fatalf("call counts workRequest/update = %d/%d, want 1/0", workCalls, updateCalls)
	}
	assertWebAppAccelerationUpdateAsyncWithWorkRequestStatus(
		t,
		resource.Status.OsokStatus.Async.Current,
		waasdk.WorkRequestStatusSucceeded,
		shared.OSOKAsyncClassPending,
	)
	if !strings.Contains(resource.Status.OsokStatus.Message, "waiting for OCI readback confirmation") {
		t.Fatalf("status.message = %q, want readback wait", resource.Status.OsokStatus.Message)
	}
}

func TestWebAppAccelerationServiceClientMovesCompartmentWithoutRecreate(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	resource.Spec.CompartmentId = testWebAppAccelerationMoveCompID
	getCalls := 0
	changeCalls := 0
	workCalls := 0
	createCalls := 0
	updateCalls := 0
	var changeRequest waasdk.ChangeWebAppAccelerationCompartmentRequest

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(_ context.Context, request waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			getCalls++
			requireStringPtr(t, "get webAppAccelerationId", request.WebAppAccelerationId, testWebAppAccelerationID)
			compartmentID := testWebAppAccelerationCompID
			state := waasdk.WebAppAccelerationLifecycleStateActive
			if getCalls == 2 {
				compartmentID = testWebAppAccelerationMoveCompID
				state = waasdk.WebAppAccelerationLifecycleStateUpdating
			}
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAccelerationInCompartment(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					compartmentID,
					state,
				),
				OpcRequestId: common.String("opc-read-after-move"),
			}, nil
		},
		changeFn: func(_ context.Context, request waasdk.ChangeWebAppAccelerationCompartmentRequest) (waasdk.ChangeWebAppAccelerationCompartmentResponse, error) {
			changeCalls++
			changeRequest = request
			return waasdk.ChangeWebAppAccelerationCompartmentResponse{
				OpcRequestId:     common.String("opc-move"),
				OpcWorkRequestId: common.String("wr-move"),
			}, nil
		},
		workFn: func(_ context.Context, request waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error) {
			workCalls++
			requireStringPtr(t, "get workRequestId", request.WorkRequestId, "wr-move")
			return waasdk.GetWorkRequestResponse{
				WorkRequest: makeWebAppAccelerationWorkRequest(
					"wr-move",
					waasdk.WorkRequestOperationTypeMoveWebAppAcceleration,
					waasdk.WorkRequestStatusInProgress,
				),
			}, nil
		},
		createFn: func(context.Context, waasdk.CreateWebAppAccelerationRequest) (waasdk.CreateWebAppAccelerationResponse, error) {
			createCalls++
			return waasdk.CreateWebAppAccelerationResponse{}, nil
		},
		updateFn: func(context.Context, waasdk.UpdateWebAppAccelerationRequest) (waasdk.UpdateWebAppAccelerationResponse, error) {
			updateCalls++
			return waasdk.UpdateWebAppAccelerationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while move is pending", response)
	}
	if getCalls != 2 || changeCalls != 1 || workCalls != 1 || createCalls != 0 || updateCalls != 0 {
		t.Fatalf(
			"call counts get/change/workRequest/create/update = %d/%d/%d/%d/%d, want 2/1/1/0/0",
			getCalls,
			changeCalls,
			workCalls,
			createCalls,
			updateCalls,
		)
	}
	requireStringPtr(t, "change webAppAccelerationId", changeRequest.WebAppAccelerationId, testWebAppAccelerationID)
	requireStringPtr(t, "change compartmentId", changeRequest.CompartmentId, testWebAppAccelerationMoveCompID)
	assertWebAppAccelerationMovedFields(t, resource)
	assertWebAppAccelerationMoveAsyncWithWorkRequestStatus(
		t,
		resource.Status.OsokStatus.Async.Current,
		waasdk.WorkRequestStatusInProgress,
		shared.OSOKAsyncClassPending,
	)
}

func TestWebAppAccelerationServiceClientDoesNotRepeatPendingCompartmentMove(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	resource.Status.OsokStatus.OpcRequestID = "opc-move"
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:    "wr-move",
		RawOperationType: string(waasdk.WorkRequestOperationTypeMoveWebAppAcceleration),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		UpdatedAt:        &metav1.Time{},
	}
	resource.Spec.CompartmentId = testWebAppAccelerationMoveCompID
	getCalls := 0
	workCalls := 0
	changeCalls := 0
	updateCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(_ context.Context, request waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			getCalls++
			requireStringPtr(t, "get webAppAccelerationId", request.WebAppAccelerationId, testWebAppAccelerationID)
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAccelerationInCompartment(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					testWebAppAccelerationCompID,
					waasdk.WebAppAccelerationLifecycleStateActive,
				),
				OpcRequestId: common.String("opc-get"),
			}, nil
		},
		changeFn: func(context.Context, waasdk.ChangeWebAppAccelerationCompartmentRequest) (waasdk.ChangeWebAppAccelerationCompartmentResponse, error) {
			changeCalls++
			return waasdk.ChangeWebAppAccelerationCompartmentResponse{}, nil
		},
		workFn: func(_ context.Context, request waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error) {
			workCalls++
			requireStringPtr(t, "get workRequestId", request.WorkRequestId, "wr-move")
			return waasdk.GetWorkRequestResponse{
				WorkRequest: makeWebAppAccelerationWorkRequest(
					"wr-move",
					waasdk.WorkRequestOperationTypeMoveWebAppAcceleration,
					waasdk.WorkRequestStatusInProgress,
				),
			}, nil
		},
		updateFn: func(context.Context, waasdk.UpdateWebAppAccelerationRequest) (waasdk.UpdateWebAppAccelerationResponse, error) {
			updateCalls++
			return waasdk.UpdateWebAppAccelerationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while move is pending", response)
	}
	if getCalls != 1 || workCalls != 1 || changeCalls != 0 || updateCalls != 0 {
		t.Fatalf(
			"call counts get/workRequest/change/update = %d/%d/%d/%d, want 1/1/0/0",
			getCalls,
			workCalls,
			changeCalls,
			updateCalls,
		)
	}
	if resource.Status.CompartmentId != testWebAppAccelerationCompID {
		t.Fatalf("status.compartmentId = %q, want observed old compartment while move is pending", resource.Status.CompartmentId)
	}
	assertWebAppAccelerationMoveAsyncWithWorkRequestStatus(
		t,
		resource.Status.OsokStatus.Async.Current,
		waasdk.WorkRequestStatusInProgress,
		shared.OSOKAsyncClassPending,
	)
}

func TestWebAppAccelerationServiceClientFailsPendingCompartmentMoveWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:    "wr-move",
		RawOperationType: string(waasdk.WorkRequestOperationTypeMoveWebAppAcceleration),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		UpdatedAt:        &metav1.Time{},
	}
	resource.Spec.CompartmentId = testWebAppAccelerationMoveCompID
	workCalls := 0
	changeCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAccelerationInCompartment(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					testWebAppAccelerationCompID,
					waasdk.WebAppAccelerationLifecycleStateActive,
				),
			}, nil
		},
		workFn: func(_ context.Context, request waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error) {
			workCalls++
			requireStringPtr(t, "get workRequestId", request.WorkRequestId, "wr-move")
			return waasdk.GetWorkRequestResponse{
				WorkRequest: makeWebAppAccelerationWorkRequest(
					"wr-move",
					waasdk.WorkRequestOperationTypeMoveWebAppAcceleration,
					waasdk.WorkRequestStatusCanceled,
				),
			}, nil
		},
		changeFn: func(context.Context, waasdk.ChangeWebAppAccelerationCompartmentRequest) (waasdk.ChangeWebAppAccelerationCompartmentResponse, error) {
			changeCalls++
			return waasdk.ChangeWebAppAccelerationCompartmentResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "update work request wr-move finished with status CANCELED") {
		t.Fatalf("CreateOrUpdate() error = %v, want canceled move work request", err)
	}
	if response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful non-requeue", response)
	}
	if workCalls != 1 || changeCalls != 0 {
		t.Fatalf("call counts workRequest/change = %d/%d, want 1/0", workCalls, changeCalls)
	}
	assertWebAppAccelerationMoveAsyncWithWorkRequestStatus(
		t,
		resource.Status.OsokStatus.Async.Current,
		waasdk.WorkRequestStatusCanceled,
		shared.OSOKAsyncClassCanceled,
	)
}

func TestWebAppAccelerationServiceClientKeepsSucceededMoveWorkRequestPendingForStaleReadback(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:    "wr-move",
		RawOperationType: string(waasdk.WorkRequestOperationTypeMoveWebAppAcceleration),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		UpdatedAt:        &metav1.Time{},
	}
	resource.Spec.CompartmentId = testWebAppAccelerationMoveCompID
	workCalls := 0
	changeCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAccelerationInCompartment(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					testWebAppAccelerationCompID,
					waasdk.WebAppAccelerationLifecycleStateActive,
				),
			}, nil
		},
		workFn: func(_ context.Context, request waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error) {
			workCalls++
			requireStringPtr(t, "get workRequestId", request.WorkRequestId, "wr-move")
			return waasdk.GetWorkRequestResponse{
				WorkRequest: makeWebAppAccelerationWorkRequest(
					"wr-move",
					waasdk.WorkRequestOperationTypeMoveWebAppAcceleration,
					waasdk.WorkRequestStatusSucceeded,
				),
			}, nil
		},
		changeFn: func(context.Context, waasdk.ChangeWebAppAccelerationCompartmentRequest) (waasdk.ChangeWebAppAccelerationCompartmentResponse, error) {
			changeCalls++
			return waasdk.ChangeWebAppAccelerationCompartmentResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue for stale move readback", response)
	}
	if workCalls != 1 || changeCalls != 0 {
		t.Fatalf("call counts workRequest/change = %d/%d, want 1/0", workCalls, changeCalls)
	}
	assertWebAppAccelerationMoveAsyncWithWorkRequestStatus(
		t,
		resource.Status.OsokStatus.Async.Current,
		waasdk.WorkRequestStatusSucceeded,
		shared.OSOKAsyncClassPending,
	)
	if !strings.Contains(resource.Status.OsokStatus.Message, "waiting for OCI readback confirmation") {
		t.Fatalf("status.message = %q, want readback wait", resource.Status.OsokStatus.Message)
	}
}

func TestWebAppAccelerationServiceClientRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	resource.Spec.LoadBalancerId = testWebAppAccelerationOtherLBID
	updateCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					testWebAppAccelerationLBID,
					waasdk.WebAppAccelerationLifecycleStateActive,
				),
			}, nil
		},
		updateFn: func(context.Context, waasdk.UpdateWebAppAccelerationRequest) (waasdk.UpdateWebAppAccelerationResponse, error) {
			updateCalls++
			return waasdk.UpdateWebAppAccelerationResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "create-only field drift") || !strings.Contains(err.Error(), "loadBalancerId") {
		t.Fatalf("CreateOrUpdate() error = %v, want create-only loadBalancerId drift", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0 before create-only drift is fixed", updateCalls)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", resource.Status.OsokStatus.Reason)
	}
}

type webAppAccelerationPendingWriteDeleteCase struct {
	name             string
	phase            shared.OSOKAsyncPhase
	workRequestID    string
	rawOperationType waasdk.WorkRequestOperationTypeEnum
	seedOcid         bool
}

func TestWebAppAccelerationDeleteWaitsForPendingWriteWorkRequests(t *testing.T) {
	tests := []webAppAccelerationPendingWriteDeleteCase{
		{
			name:             "create",
			phase:            shared.OSOKAsyncPhaseCreate,
			workRequestID:    "wr-create",
			rawOperationType: waasdk.WorkRequestOperationTypeCreateWebAppAcceleration,
		},
		{
			name:             "update",
			phase:            shared.OSOKAsyncPhaseUpdate,
			workRequestID:    "wr-update",
			rawOperationType: waasdk.WorkRequestOperationTypeUpdateWebAppAcceleration,
			seedOcid:         true,
		},
		{
			name:             "move",
			phase:            shared.OSOKAsyncPhaseUpdate,
			workRequestID:    "wr-move",
			rawOperationType: waasdk.WorkRequestOperationTypeMoveWebAppAcceleration,
			seedOcid:         true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runWebAppAccelerationPendingWriteDeleteCase(t, tt)
		})
	}
}

type webAppAccelerationPendingWriteDeleteFixture struct {
	t           *testing.T
	testCase    webAppAccelerationPendingWriteDeleteCase
	resource    *waav1beta1.WebAppAcceleration
	workCalls   int
	getCalls    int
	deleteCalls int
}

func runWebAppAccelerationPendingWriteDeleteCase(t *testing.T, testCase webAppAccelerationPendingWriteDeleteCase) {
	t.Helper()

	resource := makeWebAppAccelerationResource()
	resource.Finalizers = []string{core.OSOKFinalizerName}
	if testCase.seedOcid {
		resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	}
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            testCase.phase,
		WorkRequestID:    testCase.workRequestID,
		RawOperationType: string(testCase.rawOperationType),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		UpdatedAt:        &metav1.Time{},
	}

	fixture := &webAppAccelerationPendingWriteDeleteFixture{
		t:        t,
		testCase: testCase,
		resource: resource,
	}
	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		workFn:   fixture.workRequest,
		getFn:    fixture.getWebAppAcceleration,
		deleteFn: fixture.deleteWebAppAcceleration,
	})

	deleted, err := client.Delete(context.Background(), resource)
	fixture.requireOutcome(deleted, err)
}

func (f *webAppAccelerationPendingWriteDeleteFixture) workRequest(
	_ context.Context,
	request waasdk.GetWorkRequestRequest,
) (waasdk.GetWorkRequestResponse, error) {
	f.t.Helper()

	f.workCalls++
	requireStringPtr(f.t, "get workRequestId", request.WorkRequestId, f.testCase.workRequestID)
	return waasdk.GetWorkRequestResponse{
		WorkRequest: makeWebAppAccelerationWorkRequest(
			f.testCase.workRequestID,
			f.testCase.rawOperationType,
			waasdk.WorkRequestStatusInProgress,
		),
	}, nil
}

func (f *webAppAccelerationPendingWriteDeleteFixture) getWebAppAcceleration(
	context.Context,
	waasdk.GetWebAppAccelerationRequest,
) (waasdk.GetWebAppAccelerationResponse, error) {
	f.getCalls++
	return waasdk.GetWebAppAccelerationResponse{}, nil
}

func (f *webAppAccelerationPendingWriteDeleteFixture) deleteWebAppAcceleration(
	context.Context,
	waasdk.DeleteWebAppAccelerationRequest,
) (waasdk.DeleteWebAppAccelerationResponse, error) {
	f.deleteCalls++
	return waasdk.DeleteWebAppAccelerationResponse{}, nil
}

func (f *webAppAccelerationPendingWriteDeleteFixture) requireOutcome(deleted bool, err error) {
	f.t.Helper()

	if err != nil {
		f.t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		f.t.Fatal("Delete() = true, want false while write work request is pending")
	}
	if f.workCalls != 1 || f.getCalls != 0 || f.deleteCalls != 0 {
		f.t.Fatalf(
			"call counts workRequest/get/delete = %d/%d/%d, want 1/0/0",
			f.workCalls,
			f.getCalls,
			f.deleteCalls,
		)
	}
	assertWebAppAccelerationDeleteStillPending(f.t, f.resource, "write work request is pending")
	assertWebAppAccelerationAsyncWithRawStatus(
		f.t,
		f.resource.Status.OsokStatus.Async.Current,
		f.testCase.phase,
		f.testCase.workRequestID,
		string(f.testCase.rawOperationType),
		string(waasdk.WorkRequestStatusInProgress),
		shared.OSOKAsyncClassPending,
	)
}

type webAppAccelerationPendingLifecycleDeleteCase struct {
	name      string
	state     waasdk.WebAppAccelerationLifecycleStateEnum
	wantPhase shared.OSOKAsyncPhase
}

func TestWebAppAccelerationDeleteRetainsFinalizerWhilePreDeleteReadbackIsCreatingOrUpdating(t *testing.T) {
	tests := []webAppAccelerationPendingLifecycleDeleteCase{
		{
			name:      "creating",
			state:     waasdk.WebAppAccelerationLifecycleStateCreating,
			wantPhase: shared.OSOKAsyncPhaseCreate,
		},
		{
			name:      "updating",
			state:     waasdk.WebAppAccelerationLifecycleStateUpdating,
			wantPhase: shared.OSOKAsyncPhaseUpdate,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runWebAppAccelerationPendingLifecycleDeleteCase(t, tt)
		})
	}
}

type webAppAccelerationPendingLifecycleDeleteFixture struct {
	t           *testing.T
	testCase    webAppAccelerationPendingLifecycleDeleteCase
	resource    *waav1beta1.WebAppAcceleration
	getCalls    int
	deleteCalls int
}

func runWebAppAccelerationPendingLifecycleDeleteCase(
	t *testing.T,
	testCase webAppAccelerationPendingLifecycleDeleteCase,
) {
	t.Helper()

	resource := makeWebAppAccelerationResource()
	resource.Finalizers = []string{core.OSOKFinalizerName}
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	fixture := &webAppAccelerationPendingLifecycleDeleteFixture{
		t:        t,
		testCase: testCase,
		resource: resource,
	}
	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn:    fixture.getWebAppAcceleration,
		deleteFn: fixture.deleteWebAppAcceleration,
	})

	deleted, err := client.Delete(context.Background(), resource)
	fixture.requireOutcome(deleted, err)
}

func (f *webAppAccelerationPendingLifecycleDeleteFixture) getWebAppAcceleration(
	context.Context,
	waasdk.GetWebAppAccelerationRequest,
) (waasdk.GetWebAppAccelerationResponse, error) {
	f.getCalls++
	return waasdk.GetWebAppAccelerationResponse{
		WebAppAcceleration: makeSDKWebAppAcceleration(
			testWebAppAccelerationID,
			f.resource.Spec.DisplayName,
			f.resource.Spec.WebAppAccelerationPolicyId,
			f.resource.Spec.LoadBalancerId,
			f.testCase.state,
		),
	}, nil
}

func (f *webAppAccelerationPendingLifecycleDeleteFixture) deleteWebAppAcceleration(
	context.Context,
	waasdk.DeleteWebAppAccelerationRequest,
) (waasdk.DeleteWebAppAccelerationResponse, error) {
	f.deleteCalls++
	return waasdk.DeleteWebAppAccelerationResponse{}, nil
}

func (f *webAppAccelerationPendingLifecycleDeleteFixture) requireOutcome(deleted bool, err error) {
	f.t.Helper()

	if err != nil {
		f.t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		f.t.Fatal("Delete() = true, want false while write lifecycle readback is pending")
	}
	if f.getCalls != 1 || f.deleteCalls != 0 {
		f.t.Fatalf("call counts get/delete = %d/%d, want 1/0", f.getCalls, f.deleteCalls)
	}
	assertWebAppAccelerationDeleteStillPending(f.t, f.resource, "write lifecycle readback is pending")
	if f.resource.Status.LifecycleState != string(f.testCase.state) {
		f.t.Fatalf("status.lifecycleState = %q, want %s", f.resource.Status.LifecycleState, f.testCase.state)
	}
	assertWebAppAccelerationPendingLifecycleDeleteAsync(
		f.t,
		f.resource.Status.OsokStatus.Async.Current,
		f.testCase.wantPhase,
		f.testCase.state,
	)
}

func assertWebAppAccelerationDeleteStillPending(
	t *testing.T,
	resource *waav1beta1.WebAppAcceleration,
	reason string,
) {
	t.Helper()

	if !core.HasFinalizer(resource, core.OSOKFinalizerName) {
		t.Fatalf("finalizer removed while %s", reason)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt set while %s", reason)
	}
}

func assertWebAppAccelerationPendingLifecycleDeleteAsync(
	t *testing.T,
	current *shared.OSOKAsyncOperation,
	wantPhase shared.OSOKAsyncPhase,
	wantState waasdk.WebAppAccelerationLifecycleStateEnum,
) {
	t.Helper()

	if current == nil {
		t.Fatal("status.status.async.current = nil, want lifecycle pending write")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle ||
		current.Phase != wantPhase ||
		current.RawStatus != string(wantState) ||
		current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.status.async.current = %#v, want lifecycle %s/%s pending", current, wantPhase, wantState)
	}
	if !strings.Contains(current.Message, "retaining finalizer before delete") {
		t.Fatalf("status.status.async.current.message = %q, want retained finalizer message", current.Message)
	}
}

func TestWebAppAccelerationDeleteRetainsFinalizerWhileLifecycleDeleting(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	getCalls := 0
	deleteCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			getCalls++
			state := waasdk.WebAppAccelerationLifecycleStateActive
			if getCalls > 1 {
				state = waasdk.WebAppAccelerationLifecycleStateDeleting
			}
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					state,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request waasdk.DeleteWebAppAccelerationRequest) (waasdk.DeleteWebAppAccelerationResponse, error) {
			deleteCalls++
			requireStringPtr(t, "delete webAppAccelerationId", request.WebAppAccelerationId, testWebAppAccelerationID)
			return waasdk.DeleteWebAppAccelerationResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false while OCI lifecycle is DELETING")
	}
	if getCalls != 2 || deleteCalls != 1 {
		t.Fatalf("call counts get/delete = %d/%d, want 2/1", getCalls, deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set while delete is pending")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want Terminating", resource.Status.OsokStatus.Reason)
	}
	if resource.Status.LifecycleState != string(waasdk.WebAppAccelerationLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
}

func TestWebAppAccelerationDeleteRetainsFinalizerWhenDeleteWorkRequestReadbackIsActive(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Finalizers = []string{core.OSOKFinalizerName}
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	getCalls := 0
	deleteCalls := 0
	workCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			getCalls++
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					waasdk.WebAppAccelerationLifecycleStateActive,
				),
			}, nil
		},
		deleteFn: func(_ context.Context, request waasdk.DeleteWebAppAccelerationRequest) (waasdk.DeleteWebAppAccelerationResponse, error) {
			deleteCalls++
			requireStringPtr(t, "delete webAppAccelerationId", request.WebAppAccelerationId, testWebAppAccelerationID)
			return waasdk.DeleteWebAppAccelerationResponse{
				OpcRequestId:     common.String("opc-delete"),
				OpcWorkRequestId: common.String("wr-delete"),
			}, nil
		},
		workFn: func(_ context.Context, request waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error) {
			workCalls++
			requireStringPtr(t, "get workRequestId", request.WorkRequestId, "wr-delete")
			return waasdk.GetWorkRequestResponse{
				WorkRequest: makeWebAppAccelerationWorkRequest(
					"wr-delete",
					waasdk.WorkRequestOperationTypeDeleteWebAppAcceleration,
					waasdk.WorkRequestStatusInProgress,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false while delete work request readback is still ACTIVE")
	}
	if getCalls != 1 || deleteCalls != 1 || workCalls != 1 {
		t.Fatalf(
			"call counts get/delete/workRequest = %d/%d/%d, want 1/1/1",
			getCalls,
			deleteCalls,
			workCalls,
		)
	}
	if !core.HasFinalizer(resource, core.OSOKFinalizerName) {
		t.Fatal("finalizer removed while delete work request is pending")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set while delete work request is pending")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", resource.Status.OsokStatus.OpcRequestID)
	}
	assertWebAppAccelerationDeleteAsyncWithWorkRequestStatus(
		t,
		resource.Status.OsokStatus.Async.Current,
		waasdk.WorkRequestStatusInProgress,
		shared.OSOKAsyncClassPending,
	)
}

func TestWebAppAccelerationDeleteDoesNotRepeatPendingDeleteWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Finalizers = []string{core.OSOKFinalizerName}
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseDelete,
		WorkRequestID:    "wr-delete",
		RawOperationType: string(waasdk.WorkRequestOperationTypeDeleteWebAppAcceleration),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		UpdatedAt:        &metav1.Time{},
	}
	getCalls := 0
	deleteCalls := 0
	workCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			getCalls++
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					waasdk.WebAppAccelerationLifecycleStateActive,
				),
			}, nil
		},
		deleteFn: func(context.Context, waasdk.DeleteWebAppAccelerationRequest) (waasdk.DeleteWebAppAccelerationResponse, error) {
			deleteCalls++
			return waasdk.DeleteWebAppAccelerationResponse{}, nil
		},
		workFn: func(_ context.Context, request waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error) {
			workCalls++
			requireStringPtr(t, "get workRequestId", request.WorkRequestId, "wr-delete")
			return waasdk.GetWorkRequestResponse{
				WorkRequest: makeWebAppAccelerationWorkRequest(
					"wr-delete",
					waasdk.WorkRequestOperationTypeDeleteWebAppAcceleration,
					waasdk.WorkRequestStatusInProgress,
				),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false while existing delete work request is pending")
	}
	if getCalls != 0 || deleteCalls != 0 || workCalls != 1 {
		t.Fatalf(
			"call counts get/delete/workRequest = %d/%d/%d, want 0/0/1",
			getCalls,
			deleteCalls,
			workCalls,
		)
	}
	if !core.HasFinalizer(resource, core.OSOKFinalizerName) {
		t.Fatal("finalizer removed while existing delete work request is pending")
	}
	assertWebAppAccelerationDeleteAsyncWithWorkRequestStatus(
		t,
		resource.Status.OsokStatus.Async.Current,
		waasdk.WorkRequestStatusInProgress,
		shared.OSOKAsyncClassPending,
	)
}

func TestWebAppAccelerationDeleteFailsPendingDeleteWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Finalizers = []string{core.OSOKFinalizerName}
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseDelete,
		WorkRequestID:    "wr-delete",
		RawOperationType: string(waasdk.WorkRequestOperationTypeDeleteWebAppAcceleration),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		UpdatedAt:        &metav1.Time{},
	}
	workCalls := 0
	getCalls := 0
	deleteCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		workFn: func(_ context.Context, request waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error) {
			workCalls++
			requireStringPtr(t, "get workRequestId", request.WorkRequestId, "wr-delete")
			return waasdk.GetWorkRequestResponse{
				WorkRequest: makeWebAppAccelerationWorkRequest(
					"wr-delete",
					waasdk.WorkRequestOperationTypeDeleteWebAppAcceleration,
					waasdk.WorkRequestStatusFailed,
				),
			}, nil
		},
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			getCalls++
			return waasdk.GetWebAppAccelerationResponse{}, nil
		},
		deleteFn: func(context.Context, waasdk.DeleteWebAppAccelerationRequest) (waasdk.DeleteWebAppAccelerationResponse, error) {
			deleteCalls++
			return waasdk.DeleteWebAppAccelerationResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "delete work request wr-delete finished with status FAILED") {
		t.Fatalf("Delete() error = %v, want failed delete work request", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false for failed delete work request")
	}
	if workCalls != 1 || getCalls != 0 || deleteCalls != 0 {
		t.Fatalf(
			"call counts workRequest/get/delete = %d/%d/%d, want 1/0/0",
			workCalls,
			getCalls,
			deleteCalls,
		)
	}
	if !core.HasFinalizer(resource, core.OSOKFinalizerName) {
		t.Fatal("finalizer removed after failed delete work request")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", resource.Status.OsokStatus.Reason)
	}
	assertWebAppAccelerationDeleteAsyncWithWorkRequestStatus(
		t,
		resource.Status.OsokStatus.Async.Current,
		waasdk.WorkRequestStatusFailed,
		shared.OSOKAsyncClassFailed,
	)
}

func TestWebAppAccelerationDeleteKeepsFinalizerAfterSucceededWorkRequestUntilConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Finalizers = []string{core.OSOKFinalizerName}
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseDelete,
		WorkRequestID:    "wr-delete",
		RawOperationType: string(waasdk.WorkRequestOperationTypeDeleteWebAppAcceleration),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		UpdatedAt:        &metav1.Time{},
	}
	workCalls := 0
	getCalls := 0
	deleteCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		workFn: func(_ context.Context, request waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error) {
			workCalls++
			requireStringPtr(t, "get workRequestId", request.WorkRequestId, "wr-delete")
			return waasdk.GetWorkRequestResponse{
				WorkRequest: makeWebAppAccelerationWorkRequest(
					"wr-delete",
					waasdk.WorkRequestOperationTypeDeleteWebAppAcceleration,
					waasdk.WorkRequestStatusSucceeded,
				),
			}, nil
		},
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			getCalls++
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					waasdk.WebAppAccelerationLifecycleStateActive,
				),
			}, nil
		},
		deleteFn: func(context.Context, waasdk.DeleteWebAppAccelerationRequest) (waasdk.DeleteWebAppAccelerationResponse, error) {
			deleteCalls++
			return waasdk.DeleteWebAppAccelerationResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false until succeeded delete work request has final confirmation")
	}
	if workCalls != 1 || getCalls != 1 || deleteCalls != 0 {
		t.Fatalf(
			"call counts workRequest/get/delete = %d/%d/%d, want 1/1/0",
			workCalls,
			getCalls,
			deleteCalls,
		)
	}
	if !core.HasFinalizer(resource, core.OSOKFinalizerName) {
		t.Fatal("finalizer removed before delete confirmation")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set before delete confirmation")
	}
	assertWebAppAccelerationDeleteAsyncWithWorkRequestStatus(
		t,
		resource.Status.OsokStatus.Async.Current,
		waasdk.WorkRequestStatusSucceeded,
		shared.OSOKAsyncClassPending,
	)
	if !strings.Contains(resource.Status.OsokStatus.Message, "waiting for OCI readback confirmation") {
		t.Fatalf("status.message = %q, want delete confirmation wait", resource.Status.OsokStatus.Message)
	}
}

func TestWebAppAccelerationDeleteConfirmsNotFoundAfterDelete(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	getCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			getCalls++
			if getCalls == 1 {
				return waasdk.GetWebAppAccelerationResponse{
					WebAppAcceleration: makeSDKWebAppAcceleration(
						testWebAppAccelerationID,
						resource.Spec.DisplayName,
						resource.Spec.WebAppAccelerationPolicyId,
						resource.Spec.LoadBalancerId,
						waasdk.WebAppAccelerationLifecycleStateActive,
					),
				}, nil
			}
			return waasdk.GetWebAppAccelerationResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
		},
		deleteFn: func(context.Context, waasdk.DeleteWebAppAccelerationRequest) (waasdk.DeleteWebAppAccelerationResponse, error) {
			return waasdk.DeleteWebAppAccelerationResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true after confirm read returns unambiguous NotFound")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want deletion timestamp")
	}
}

func TestWebAppAccelerationDeleteKeepsFinalizerOnAmbiguousPreDeleteRead(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Finalizers = []string{core.OSOKFinalizerName}
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)
	getCalls := 0
	deleteCalls := 0

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			getCalls++
			return waasdk.GetWebAppAccelerationResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"not authorized or not found",
			)
		},
		deleteFn: func(context.Context, waasdk.DeleteWebAppAccelerationRequest) (waasdk.DeleteWebAppAccelerationResponse, error) {
			deleteCalls++
			return waasdk.DeleteWebAppAccelerationResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped pre-delete read", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false for ambiguous pre-delete read")
	}
	if getCalls != 1 || deleteCalls != 0 {
		t.Fatalf("call counts get/delete = %d/%d, want 1/0", getCalls, deleteCalls)
	}
	if !core.HasFinalizer(resource, core.OSOKFinalizerName) {
		t.Fatal("finalizer removed after ambiguous pre-delete read")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set for ambiguous pre-delete read")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want service error request id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestWebAppAccelerationDeleteKeepsFinalizerOnAmbiguousNotFound(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testWebAppAccelerationID)

	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		getFn: func(context.Context, waasdk.GetWebAppAccelerationRequest) (waasdk.GetWebAppAccelerationResponse, error) {
			return waasdk.GetWebAppAccelerationResponse{
				WebAppAcceleration: makeSDKWebAppAcceleration(
					testWebAppAccelerationID,
					resource.Spec.DisplayName,
					resource.Spec.WebAppAccelerationPolicyId,
					resource.Spec.LoadBalancerId,
					waasdk.WebAppAccelerationLifecycleStateActive,
				),
			}, nil
		},
		deleteFn: func(context.Context, waasdk.DeleteWebAppAccelerationRequest) (waasdk.DeleteWebAppAccelerationResponse, error) {
			return waasdk.DeleteWebAppAccelerationResponse{}, errortest.NewServiceError(
				404,
				errorutil.NotAuthorizedOrNotFound,
				"not authorized or not found",
			)
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped not-found", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want false for ambiguous auth-shaped not-found")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set for ambiguous auth-shaped not-found")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want service error request id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestWebAppAccelerationCreateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeWebAppAccelerationResource()
	client := testWebAppAccelerationClient(&fakeWebAppAccelerationOCIClient{
		listFn: func(context.Context, waasdk.ListWebAppAccelerationsRequest) (waasdk.ListWebAppAccelerationsResponse, error) {
			return waasdk.ListWebAppAccelerationsResponse{}, nil
		},
		createFn: func(context.Context, waasdk.CreateWebAppAccelerationRequest) (waasdk.CreateWebAppAccelerationResponse, error) {
			return waasdk.CreateWebAppAccelerationResponse{}, errortest.NewServiceError(500, "InternalError", "service unavailable")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "service unavailable") {
		t.Fatalf("CreateOrUpdate() error = %v, want OCI create failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want service error request id", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", resource.Status.OsokStatus.Reason)
	}
}

func assertWebAppAccelerationCreateRequest(
	t *testing.T,
	resource *waav1beta1.WebAppAcceleration,
	request waasdk.CreateWebAppAccelerationRequest,
) {
	t.Helper()

	details, ok := request.CreateWebAppAccelerationDetails.(waasdk.CreateWebAppAccelerationLoadBalancerDetails)
	if !ok {
		t.Fatalf("create details type = %T, want CreateWebAppAccelerationLoadBalancerDetails", request.CreateWebAppAccelerationDetails)
	}
	requireStringPtr(t, "create compartmentId", details.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "create policyId", details.WebAppAccelerationPolicyId, resource.Spec.WebAppAccelerationPolicyId)
	requireStringPtr(t, "create loadBalancerId", details.LoadBalancerId, resource.Spec.LoadBalancerId)
	requireStringPtr(t, "create displayName", details.DisplayName, resource.Spec.DisplayName)
	if !reflect.DeepEqual(details.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("create freeformTags = %#v, want %#v", details.FreeformTags, resource.Spec.FreeformTags)
	}
	wantDefined := map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}}
	if !reflect.DeepEqual(details.DefinedTags, wantDefined) {
		t.Fatalf("create definedTags = %#v, want %#v", details.DefinedTags, wantDefined)
	}
}

func assertWebAppAccelerationUpdateRequest(t *testing.T, request waasdk.UpdateWebAppAccelerationRequest) {
	t.Helper()

	requireStringPtr(t, "update webAppAccelerationId", request.WebAppAccelerationId, testWebAppAccelerationID)
	requireStringPtr(t, "update displayName", request.DisplayName, "accel-renamed")
	requireStringPtr(t, "update policyId", request.WebAppAccelerationPolicyId, testWebAppAccelerationPolicyID2)
	if !reflect.DeepEqual(request.FreeformTags, map[string]string{"env": "prod"}) {
		t.Fatalf("update freeformTags = %#v, want prod tag", request.FreeformTags)
	}
	wantDefined := map[string]map[string]interface{}{"Operations": {"CostCenter": "99"}}
	if !reflect.DeepEqual(request.DefinedTags, wantDefined) {
		t.Fatalf("update definedTags = %#v, want %#v", request.DefinedTags, wantDefined)
	}
	wantSystem := map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "false"}}
	if !reflect.DeepEqual(request.SystemTags, wantSystem) {
		t.Fatalf("update systemTags = %#v, want %#v", request.SystemTags, wantSystem)
	}
}

func assertWebAppAccelerationActiveStatus(
	t *testing.T,
	resource *waav1beta1.WebAppAcceleration,
	wantID string,
	wantOpcRequestID string,
) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != wantID {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantID)
	}
	if resource.Status.Id != wantID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, wantID)
	}
	if resource.Status.LifecycleState != string(waasdk.WebAppAccelerationLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
	if resource.Status.BackendType != webAppAccelerationLoadBalancerBackendType {
		t.Fatalf("status.backendType = %q, want LOAD_BALANCER", resource.Status.BackendType)
	}
	if resource.Status.LoadBalancerId != testWebAppAccelerationLBID {
		t.Fatalf("status.loadBalancerId = %q, want %q", resource.Status.LoadBalancerId, testWebAppAccelerationLBID)
	}
	if resource.Status.OsokStatus.OpcRequestID != wantOpcRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, wantOpcRequestID)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Active) {
		t.Fatalf("status.status.reason = %q, want Active", resource.Status.OsokStatus.Reason)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil after ACTIVE", resource.Status.OsokStatus.Async.Current)
	}
}

func assertWebAppAccelerationMovedFields(t *testing.T, resource *waav1beta1.WebAppAcceleration) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != testWebAppAccelerationID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testWebAppAccelerationID)
	}
	if resource.Status.Id != testWebAppAccelerationID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, testWebAppAccelerationID)
	}
	if resource.Status.CompartmentId != testWebAppAccelerationMoveCompID {
		t.Fatalf("status.compartmentId = %q, want %q", resource.Status.CompartmentId, testWebAppAccelerationMoveCompID)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-move" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-move", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Updating) {
		t.Fatalf("status.status.reason = %q, want Updating", resource.Status.OsokStatus.Reason)
	}
}

func assertWebAppAccelerationCreateAsyncWithWorkRequestStatus(
	t *testing.T,
	current *shared.OSOKAsyncOperation,
	wantRawStatus waasdk.WorkRequestStatusEnum,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	assertWebAppAccelerationAsyncWithRawStatus(
		t,
		current,
		shared.OSOKAsyncPhaseCreate,
		"wr-create",
		string(waasdk.WorkRequestOperationTypeCreateWebAppAcceleration),
		string(wantRawStatus),
		wantClass,
	)
}

func assertWebAppAccelerationUpdateAsyncWithWorkRequestStatus(
	t *testing.T,
	current *shared.OSOKAsyncOperation,
	wantRawStatus waasdk.WorkRequestStatusEnum,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	assertWebAppAccelerationAsyncWithRawStatus(
		t,
		current,
		shared.OSOKAsyncPhaseUpdate,
		"wr-update",
		string(waasdk.WorkRequestOperationTypeUpdateWebAppAcceleration),
		string(wantRawStatus),
		wantClass,
	)
}

func assertWebAppAccelerationAsyncWithRawStatus(
	t *testing.T,
	current *shared.OSOKAsyncOperation,
	wantPhase shared.OSOKAsyncPhase,
	wantWorkRequestID string,
	wantRawOperationType string,
	wantRawStatus string,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	if current == nil {
		t.Fatalf("status.status.async.current = nil, want phase %s work request %s", wantPhase, wantWorkRequestID)
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest ||
		current.Phase != wantPhase ||
		current.WorkRequestID != wantWorkRequestID ||
		current.RawOperationType != wantRawOperationType ||
		current.RawStatus != wantRawStatus ||
		current.NormalizedClass != wantClass {
		t.Fatalf(
			"status.status.async.current = %#v, want phase=%s workRequest=%s operation=%s rawStatus=%s class=%s",
			current,
			wantPhase,
			wantWorkRequestID,
			wantRawOperationType,
			wantRawStatus,
			wantClass,
		)
	}
}

func assertWebAppAccelerationMoveAsyncWithWorkRequestStatus(
	t *testing.T,
	current *shared.OSOKAsyncOperation,
	wantRawStatus waasdk.WorkRequestStatusEnum,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	assertWebAppAccelerationAsyncWithRawStatus(
		t,
		current,
		shared.OSOKAsyncPhaseUpdate,
		"wr-move",
		string(waasdk.WorkRequestOperationTypeMoveWebAppAcceleration),
		string(wantRawStatus),
		wantClass,
	)
}

func assertWebAppAccelerationDeleteAsyncWithWorkRequestStatus(
	t *testing.T,
	current *shared.OSOKAsyncOperation,
	wantRawStatus waasdk.WorkRequestStatusEnum,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	assertWebAppAccelerationAsyncWithRawStatus(
		t,
		current,
		shared.OSOKAsyncPhaseDelete,
		"wr-delete",
		string(waasdk.WorkRequestOperationTypeDeleteWebAppAcceleration),
		string(wantRawStatus),
		wantClass,
	)
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

func assertWebAppAccelerationInitError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("operation error = nil, want OCI client initialization error")
	}
	if !strings.Contains(err.Error(), "initialize WebAppAcceleration OCI client") {
		t.Fatalf("operation error = %v, want WebAppAcceleration OCI client initialization failure", err)
	}
	if !strings.Contains(err.Error(), "web app acceleration provider invalid") {
		t.Fatalf("operation error = %v, want provider failure detail", err)
	}
}

func assertWebAppAccelerationProviderCalls(
	t *testing.T,
	provider *erroringWebAppAccelerationConfigProvider,
	want int,
) {
	t.Helper()

	if provider.calls != want {
		t.Fatalf("provider calls after operation = %d, want %d; runtime wrapper should stop at InitError", provider.calls, want)
	}
}
