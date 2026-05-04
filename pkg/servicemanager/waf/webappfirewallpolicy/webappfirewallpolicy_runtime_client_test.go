/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package webappfirewallpolicy

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	wafsdk "github.com/oracle/oci-go-sdk/v65/waf"
	wafv1beta1 "github.com/oracle/oci-service-operator/api/waf/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestWebAppFirewallPolicyCreateStartsWorkRequestAndBuildsPolymorphicBody(t *testing.T) {
	resource := newTestWebAppFirewallPolicy()
	resource.Spec.Actions = []wafv1beta1.WebAppFirewallPolicyAction{
		{
			Name: "block",
			Type: string(wafsdk.ActionTypeReturnHttpResponse),
			Code: 403,
			Headers: []wafv1beta1.WebAppFirewallPolicyActionHeader{
				{Name: "X-WAF", Value: "blocked"},
			},
			Body: wafv1beta1.WebAppFirewallPolicyActionBody{
				Type: string(wafsdk.HttpResponseBodyTypeStaticText),
				Text: "blocked",
			},
		},
	}

	fake := &fakeWebAppFirewallPolicyOCIClient{
		listFunc: func(_ context.Context, _ wafsdk.ListWebAppFirewallPoliciesRequest) (wafsdk.ListWebAppFirewallPoliciesResponse, error) {
			return wafsdk.ListWebAppFirewallPoliciesResponse{}, nil
		},
		createFunc: func(_ context.Context, request wafsdk.CreateWebAppFirewallPolicyRequest) (wafsdk.CreateWebAppFirewallPolicyResponse, error) {
			return wafsdk.CreateWebAppFirewallPolicyResponse{
				WebAppFirewallPolicy: wafsdk.WebAppFirewallPolicy{
					Id:             common.String("policy-create"),
					CompartmentId:  common.String("compartment-a"),
					DisplayName:    common.String("policy-a"),
					LifecycleState: wafsdk.WebAppFirewallPolicyLifecycleStateCreating,
					FreeformTags:   map[string]string{"env": "test"},
				},
				OpcWorkRequestId: common.String("wr-create"),
				OpcRequestId:     common.String("opc-create"),
			}, nil
		},
		getWorkRequestFunc: func(_ context.Context, workRequestID string) (wafsdk.GetWorkRequestResponse, error) {
			if workRequestID != "wr-create" {
				t.Fatalf("GetWorkRequest id = %q, want wr-create", workRequestID)
			}
			return wafsdk.GetWorkRequestResponse{
				WorkRequest: testWebAppFirewallPolicyWorkRequest(
					"wr-create",
					wafsdk.WorkRequestOperationTypeCreateWafPolicy,
					wafsdk.WorkRequestStatusInProgress,
					wafsdk.WorkRequestResourceActionTypeCreated,
					"policy-create",
				),
			}, nil
		},
	}

	response, err := newWebAppFirewallPolicyServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate().ShouldRequeue = false, want true for pending work request")
	}
	if fake.createCalls != 1 {
		t.Fatalf("CreateWebAppFirewallPolicy calls = %d, want 1", fake.createCalls)
	}
	requireWebAppFirewallPolicyReturnHTTPAction(t, fake.createRequest.Actions, "block", 403, "blocked")
	requireWebAppFirewallPolicyAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", shared.OSOKAsyncClassPending)
	requireWebAppFirewallPolicyOpcRequestID(t, resource, "opc-create")
	if got := string(resource.Status.OsokStatus.Ocid); got != "policy-create" {
		t.Fatalf("status.status.ocid = %q, want policy-create", got)
	}
}

func TestWebAppFirewallPolicyCreateBodyBuildsNestedModules(t *testing.T) {
	resource := newTestWebAppFirewallPolicy()
	resource.Spec.RequestAccessControl = wafv1beta1.WebAppFirewallPolicyRequestAccessControl{
		DefaultActionName: "allow",
		Rules: []wafv1beta1.WebAppFirewallPolicyRequestAccessControlRule{
			{Name: "allow-known", ActionName: "allow", Condition: "http.request.url.path == '/health'", ConditionLanguage: "JMESPATH"},
		},
	}
	resource.Spec.RequestRateLimiting = wafv1beta1.WebAppFirewallPolicyRequestRateLimiting{
		Rules: []wafv1beta1.WebAppFirewallPolicyRequestRateLimitingRule{
			{
				Name:       "rate-limit",
				ActionName: "block",
				Configurations: []wafv1beta1.WebAppFirewallPolicyRequestRateLimitingRuleConfiguration{
					{PeriodInSeconds: 60, RequestsLimit: 10, ActionDurationInSeconds: 0},
				},
			},
		},
	}
	resource.Spec.RequestProtection = wafv1beta1.WebAppFirewallPolicyRequestProtection{
		BodyInspectionSizeLimitInBytes: 8192,
		Rules: []wafv1beta1.WebAppFirewallPolicyRequestProtectionRule{
			{
				Name:       "managed-protection",
				ActionName: "block",
				ProtectionCapabilities: []wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapability{
					{Key: "920370", Version: 1},
				},
				ProtectionCapabilitySettings: wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapabilitySettings{
					AllowedHttpMethods: []string{"GET", "POST"},
				},
			},
		},
	}

	body, err := buildWebAppFirewallPolicyCreateBody(context.Background(), resource, "")
	if err != nil {
		t.Fatalf("buildWebAppFirewallPolicyCreateBody() error = %v", err)
	}
	details, ok := body.(wafsdk.CreateWebAppFirewallPolicyDetails)
	if !ok {
		t.Fatalf("create body type = %T, want wafsdk.CreateWebAppFirewallPolicyDetails", body)
	}
	requireWebAppFirewallPolicyNestedModules(t, details)
}

func TestWebAppFirewallPolicyCreateRejectsUnsupportedActionTypeBeforeOCI(t *testing.T) {
	resource := newTestWebAppFirewallPolicy()
	resource.Spec.Actions = []wafv1beta1.WebAppFirewallPolicyAction{{Name: "mystery", Type: "MYSTERY"}}
	fake := &fakeWebAppFirewallPolicyOCIClient{
		listFunc: func(_ context.Context, _ wafsdk.ListWebAppFirewallPoliciesRequest) (wafsdk.ListWebAppFirewallPoliciesResponse, error) {
			return wafsdk.ListWebAppFirewallPoliciesResponse{}, nil
		},
		createFunc: func(_ context.Context, _ wafsdk.CreateWebAppFirewallPolicyRequest) (wafsdk.CreateWebAppFirewallPolicyResponse, error) {
			t.Fatal("CreateWebAppFirewallPolicy should not be called after unsupported action type")
			return wafsdk.CreateWebAppFirewallPolicyResponse{}, nil
		},
	}

	_, err := newWebAppFirewallPolicyServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want unsupported action type error")
	}
	if !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("CreateOrUpdate() error = %q, want unsupported type", err.Error())
	}
	if fake.createCalls != 0 {
		t.Fatalf("CreateWebAppFirewallPolicy calls = %d, want 0", fake.createCalls)
	}
}

func TestWebAppFirewallPolicyBindUsesPaginatedList(t *testing.T) {
	resource := newTestWebAppFirewallPolicy()
	fake := &fakeWebAppFirewallPolicyOCIClient{
		listFunc: func(_ context.Context, request wafsdk.ListWebAppFirewallPoliciesRequest) (wafsdk.ListWebAppFirewallPoliciesResponse, error) {
			switch webAppFirewallPolicyStringValue(request.Page) {
			case "":
				return wafsdk.ListWebAppFirewallPoliciesResponse{
					WebAppFirewallPolicyCollection: wafsdk.WebAppFirewallPolicyCollection{
						Items: []wafsdk.WebAppFirewallPolicySummary{
							{
								Id:             common.String("policy-other"),
								CompartmentId:  common.String("compartment-a"),
								DisplayName:    common.String("other"),
								LifecycleState: wafsdk.WebAppFirewallPolicyLifecycleStateActive,
							},
						},
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			case "page-2":
				return wafsdk.ListWebAppFirewallPoliciesResponse{
					WebAppFirewallPolicyCollection: wafsdk.WebAppFirewallPolicyCollection{
						Items: []wafsdk.WebAppFirewallPolicySummary{
							{
								Id:             common.String("policy-existing"),
								CompartmentId:  common.String("compartment-a"),
								DisplayName:    common.String("policy-a"),
								LifecycleState: wafsdk.WebAppFirewallPolicyLifecycleStateActive,
							},
						},
					},
				}, nil
			default:
				t.Fatalf("unexpected list page %q", webAppFirewallPolicyStringValue(request.Page))
				return wafsdk.ListWebAppFirewallPoliciesResponse{}, nil
			}
		},
		getFunc: func(_ context.Context, request wafsdk.GetWebAppFirewallPolicyRequest) (wafsdk.GetWebAppFirewallPolicyResponse, error) {
			if got := webAppFirewallPolicyStringValue(request.WebAppFirewallPolicyId); got != "policy-existing" {
				t.Fatalf("GetWebAppFirewallPolicy id = %q, want policy-existing", got)
			}
			return wafsdk.GetWebAppFirewallPolicyResponse{
				WebAppFirewallPolicy: testWebAppFirewallPolicy("policy-existing", "compartment-a", "policy-a", wafsdk.WebAppFirewallPolicyLifecycleStateActive),
				OpcRequestId:         common.String("opc-get-existing"),
			}, nil
		},
	}

	response, err := newWebAppFirewallPolicyServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate().ShouldRequeue = true, want false for active bind")
	}
	if fake.createCalls != 0 {
		t.Fatalf("CreateWebAppFirewallPolicy calls = %d, want 0", fake.createCalls)
	}
	if fake.listCalls != 2 {
		t.Fatalf("ListWebAppFirewallPolicies calls = %d, want 2", fake.listCalls)
	}
	if got := resource.Status.Id; got != "policy-existing" {
		t.Fatalf("status.id = %q, want policy-existing", got)
	}
}

func TestWebAppFirewallPolicyNoopObservedStateSkipsUpdate(t *testing.T) {
	resource := newTestWebAppFirewallPolicy()
	resource.Status.Id = "policy-existing"
	resource.Status.OsokStatus.Ocid = "policy-existing"
	fake := &fakeWebAppFirewallPolicyOCIClient{
		getFunc: func(_ context.Context, _ wafsdk.GetWebAppFirewallPolicyRequest) (wafsdk.GetWebAppFirewallPolicyResponse, error) {
			return wafsdk.GetWebAppFirewallPolicyResponse{
				WebAppFirewallPolicy: testWebAppFirewallPolicy("policy-existing", "compartment-a", "policy-a", wafsdk.WebAppFirewallPolicyLifecycleStateActive),
				OpcRequestId:         common.String("opc-get"),
			}, nil
		},
		updateFunc: func(_ context.Context, _ wafsdk.UpdateWebAppFirewallPolicyRequest) (wafsdk.UpdateWebAppFirewallPolicyResponse, error) {
			t.Fatal("UpdateWebAppFirewallPolicy should not be called for matching observed state")
			return wafsdk.UpdateWebAppFirewallPolicyResponse{}, nil
		},
	}

	response, err := newWebAppFirewallPolicyServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate().ShouldRequeue = true, want false")
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateWebAppFirewallPolicy calls = %d, want 0", fake.updateCalls)
	}
}

func TestWebAppFirewallPolicyMutableUpdateBuildsShallowUpdate(t *testing.T) {
	resource := newTestWebAppFirewallPolicy()
	resource.Spec.DisplayName = "policy-new"
	resource.Spec.FreeformTags = map[string]string{"env": "new"}
	resource.Spec.Actions = []wafv1beta1.WebAppFirewallPolicyAction{{Name: "allow", Type: string(wafsdk.ActionTypeAllow)}}
	resource.Status.Id = "policy-existing"
	resource.Status.OsokStatus.Ocid = "policy-existing"

	fake := &fakeWebAppFirewallPolicyOCIClient{
		getFunc: func(_ context.Context, _ wafsdk.GetWebAppFirewallPolicyRequest) (wafsdk.GetWebAppFirewallPolicyResponse, error) {
			return wafsdk.GetWebAppFirewallPolicyResponse{
				WebAppFirewallPolicy: testWebAppFirewallPolicy("policy-existing", "compartment-a", "policy-old", wafsdk.WebAppFirewallPolicyLifecycleStateActive),
				OpcRequestId:         common.String("opc-get"),
			}, nil
		},
		updateFunc: func(_ context.Context, request wafsdk.UpdateWebAppFirewallPolicyRequest) (wafsdk.UpdateWebAppFirewallPolicyResponse, error) {
			return wafsdk.UpdateWebAppFirewallPolicyResponse{
				OpcWorkRequestId: common.String("wr-update"),
				OpcRequestId:     common.String("opc-update"),
			}, nil
		},
		getWorkRequestFunc: func(_ context.Context, workRequestID string) (wafsdk.GetWorkRequestResponse, error) {
			if workRequestID != "wr-update" {
				t.Fatalf("GetWorkRequest id = %q, want wr-update", workRequestID)
			}
			return wafsdk.GetWorkRequestResponse{
				WorkRequest: testWebAppFirewallPolicyWorkRequest(
					"wr-update",
					wafsdk.WorkRequestOperationTypeUpdateWafPolicy,
					wafsdk.WorkRequestStatusInProgress,
					wafsdk.WorkRequestResourceActionTypeUpdated,
					"policy-existing",
				),
			}, nil
		},
	}

	response, err := newWebAppFirewallPolicyServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate().ShouldRequeue = false, want true")
	}
	if fake.updateCalls != 1 {
		t.Fatalf("UpdateWebAppFirewallPolicy calls = %d, want 1", fake.updateCalls)
	}
	if got := webAppFirewallPolicyStringValue(fake.updateRequest.DisplayName); got != "policy-new" {
		t.Fatalf("update DisplayName = %q, want policy-new", got)
	}
	if got := fake.updateRequest.FreeformTags["env"]; got != "new" {
		t.Fatalf("update FreeformTags[env] = %q, want new", got)
	}
	if _, ok := fake.updateRequest.Actions[0].(wafsdk.AllowAction); !ok {
		t.Fatalf("update action type = %T, want wafsdk.AllowAction", fake.updateRequest.Actions[0])
	}
	requireWebAppFirewallPolicyAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update", shared.OSOKAsyncClassPending)
	requireWebAppFirewallPolicyOpcRequestID(t, resource, "opc-update")
}

func TestWebAppFirewallPolicyCompartmentMoveUsesChangeOperation(t *testing.T) {
	resource := newTestWebAppFirewallPolicy()
	resource.Spec.CompartmentId = "compartment-b"
	resource.Status.Id = "policy-existing"
	resource.Status.OsokStatus.Ocid = "policy-existing"

	fake := &fakeWebAppFirewallPolicyOCIClient{
		getFunc: func(_ context.Context, _ wafsdk.GetWebAppFirewallPolicyRequest) (wafsdk.GetWebAppFirewallPolicyResponse, error) {
			return wafsdk.GetWebAppFirewallPolicyResponse{
				WebAppFirewallPolicy: testWebAppFirewallPolicy("policy-existing", "compartment-a", "policy-a", wafsdk.WebAppFirewallPolicyLifecycleStateActive),
				OpcRequestId:         common.String("opc-get"),
			}, nil
		},
		changeFunc: func(_ context.Context, _ wafsdk.ChangeWebAppFirewallPolicyCompartmentRequest) (wafsdk.ChangeWebAppFirewallPolicyCompartmentResponse, error) {
			return wafsdk.ChangeWebAppFirewallPolicyCompartmentResponse{
				OpcWorkRequestId: common.String("wr-move"),
				OpcRequestId:     common.String("opc-move"),
			}, nil
		},
		updateFunc: func(_ context.Context, _ wafsdk.UpdateWebAppFirewallPolicyRequest) (wafsdk.UpdateWebAppFirewallPolicyResponse, error) {
			t.Fatal("UpdateWebAppFirewallPolicy should not be called for compartment drift")
			return wafsdk.UpdateWebAppFirewallPolicyResponse{}, nil
		},
	}

	response, err := newWebAppFirewallPolicyServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate().ShouldRequeue = false, want true")
	}
	if fake.changeCalls != 1 {
		t.Fatalf("ChangeWebAppFirewallPolicyCompartment calls = %d, want 1", fake.changeCalls)
	}
	if got := webAppFirewallPolicyStringValue(fake.changeRequest.CompartmentId); got != "compartment-b" {
		t.Fatalf("change CompartmentId = %q, want compartment-b", got)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdateWebAppFirewallPolicy calls = %d, want 0", fake.updateCalls)
	}
	requireWebAppFirewallPolicyAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-move", shared.OSOKAsyncClassPending)
	requireWebAppFirewallPolicyOpcRequestID(t, resource, "opc-move")
}

func TestWebAppFirewallPolicyDeleteWaitsForPendingWorkRequest(t *testing.T) {
	resource := newTestWebAppFirewallPolicy()
	resource.Status.Id = "policy-existing"
	resource.Status.OsokStatus.Ocid = "policy-existing"

	fake := &fakeWebAppFirewallPolicyOCIClient{
		getFunc: func(_ context.Context, _ wafsdk.GetWebAppFirewallPolicyRequest) (wafsdk.GetWebAppFirewallPolicyResponse, error) {
			return wafsdk.GetWebAppFirewallPolicyResponse{
				WebAppFirewallPolicy: testWebAppFirewallPolicy("policy-existing", "compartment-a", "policy-a", wafsdk.WebAppFirewallPolicyLifecycleStateActive),
				OpcRequestId:         common.String("opc-get"),
			}, nil
		},
		deleteFunc: func(_ context.Context, _ wafsdk.DeleteWebAppFirewallPolicyRequest) (wafsdk.DeleteWebAppFirewallPolicyResponse, error) {
			return wafsdk.DeleteWebAppFirewallPolicyResponse{
				OpcWorkRequestId: common.String("wr-delete"),
				OpcRequestId:     common.String("opc-delete"),
			}, nil
		},
		getWorkRequestFunc: func(_ context.Context, workRequestID string) (wafsdk.GetWorkRequestResponse, error) {
			if workRequestID != "wr-delete" {
				t.Fatalf("GetWorkRequest id = %q, want wr-delete", workRequestID)
			}
			return wafsdk.GetWorkRequestResponse{
				WorkRequest: testWebAppFirewallPolicyWorkRequest(
					"wr-delete",
					wafsdk.WorkRequestOperationTypeDeleteWafPolicy,
					wafsdk.WorkRequestStatusInProgress,
					wafsdk.WorkRequestResourceActionTypeDeleted,
					"policy-existing",
				),
			}, nil
		},
	}

	deleted, err := newWebAppFirewallPolicyServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while delete work request is pending")
	}
	requireWebAppFirewallPolicyAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", shared.OSOKAsyncClassPending)
	requireWebAppFirewallPolicyOpcRequestID(t, resource, "opc-delete")
}

func TestWebAppFirewallPolicyDeleteCompletesAfterWorkRequestAndReadbackNotFound(t *testing.T) {
	resource := newTestWebAppFirewallPolicy()
	resource.Status.Id = "policy-existing"
	resource.Status.OsokStatus.Ocid = "policy-existing"
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	fake := &fakeWebAppFirewallPolicyOCIClient{
		getFunc: func(_ context.Context, _ wafsdk.GetWebAppFirewallPolicyRequest) (wafsdk.GetWebAppFirewallPolicyResponse, error) {
			return wafsdk.GetWebAppFirewallPolicyResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "policy gone")
		},
		getWorkRequestFunc: func(_ context.Context, workRequestID string) (wafsdk.GetWorkRequestResponse, error) {
			if workRequestID != "wr-delete" {
				t.Fatalf("GetWorkRequest id = %q, want wr-delete", workRequestID)
			}
			return wafsdk.GetWorkRequestResponse{
				WorkRequest: testWebAppFirewallPolicyWorkRequest(
					"wr-delete",
					wafsdk.WorkRequestOperationTypeDeleteWafPolicy,
					wafsdk.WorkRequestStatusSucceeded,
					wafsdk.WorkRequestResourceActionTypeDeleted,
					"policy-existing",
				),
			}, nil
		},
	}

	deleted, err := newWebAppFirewallPolicyServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after succeeded work request and readback NotFound")
	}
}

func TestWebAppFirewallPolicyDeleteRejectsAuthShapedNotFound(t *testing.T) {
	resource := newTestWebAppFirewallPolicy()
	resource.Status.Id = "policy-existing"
	resource.Status.OsokStatus.Ocid = "policy-existing"

	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	authErr.OpcRequestID = "opc-auth-delete"
	fake := &fakeWebAppFirewallPolicyOCIClient{
		getFunc: func(_ context.Context, _ wafsdk.GetWebAppFirewallPolicyRequest) (wafsdk.GetWebAppFirewallPolicyResponse, error) {
			return wafsdk.GetWebAppFirewallPolicyResponse{
				WebAppFirewallPolicy: testWebAppFirewallPolicy("policy-existing", "compartment-a", "policy-a", wafsdk.WebAppFirewallPolicyLifecycleStateActive),
				OpcRequestId:         common.String("opc-get"),
			}, nil
		},
		deleteFunc: func(_ context.Context, _ wafsdk.DeleteWebAppFirewallPolicyRequest) (wafsdk.DeleteWebAppFirewallPolicyResponse, error) {
			return wafsdk.DeleteWebAppFirewallPolicyResponse{}, authErr
		},
	}

	deleted, err := newWebAppFirewallPolicyServiceClientWithOCIClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not found")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped not found")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous NotAuthorizedOrNotFound", err.Error())
	}
	requireWebAppFirewallPolicyOpcRequestID(t, resource, "opc-auth-delete")
}

func TestWebAppFirewallPolicyCreateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := newTestWebAppFirewallPolicy()
	createErr := errortest.NewServiceError(500, "InternalError", "create failed")
	createErr.OpcRequestID = "opc-create-error"
	fake := &fakeWebAppFirewallPolicyOCIClient{
		listFunc: func(_ context.Context, _ wafsdk.ListWebAppFirewallPoliciesRequest) (wafsdk.ListWebAppFirewallPoliciesResponse, error) {
			return wafsdk.ListWebAppFirewallPoliciesResponse{}, nil
		},
		createFunc: func(_ context.Context, _ wafsdk.CreateWebAppFirewallPolicyRequest) (wafsdk.CreateWebAppFirewallPolicyResponse, error) {
			return wafsdk.CreateWebAppFirewallPolicyResponse{}, createErr
		},
	}

	_, err := newWebAppFirewallPolicyServiceClientWithOCIClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create error")
	}
	requireWebAppFirewallPolicyOpcRequestID(t, resource, "opc-create-error")
}

type fakeWebAppFirewallPolicyOCIClient struct {
	changeCalls        int
	createCalls        int
	getCalls           int
	listCalls          int
	updateCalls        int
	deleteCalls        int
	getWorkReqCalls    int
	changeRequest      wafsdk.ChangeWebAppFirewallPolicyCompartmentRequest
	createRequest      wafsdk.CreateWebAppFirewallPolicyRequest
	updateRequest      wafsdk.UpdateWebAppFirewallPolicyRequest
	deleteRequest      wafsdk.DeleteWebAppFirewallPolicyRequest
	changeFunc         func(context.Context, wafsdk.ChangeWebAppFirewallPolicyCompartmentRequest) (wafsdk.ChangeWebAppFirewallPolicyCompartmentResponse, error)
	createFunc         func(context.Context, wafsdk.CreateWebAppFirewallPolicyRequest) (wafsdk.CreateWebAppFirewallPolicyResponse, error)
	getFunc            func(context.Context, wafsdk.GetWebAppFirewallPolicyRequest) (wafsdk.GetWebAppFirewallPolicyResponse, error)
	listFunc           func(context.Context, wafsdk.ListWebAppFirewallPoliciesRequest) (wafsdk.ListWebAppFirewallPoliciesResponse, error)
	updateFunc         func(context.Context, wafsdk.UpdateWebAppFirewallPolicyRequest) (wafsdk.UpdateWebAppFirewallPolicyResponse, error)
	deleteFunc         func(context.Context, wafsdk.DeleteWebAppFirewallPolicyRequest) (wafsdk.DeleteWebAppFirewallPolicyResponse, error)
	getWorkRequestFunc func(context.Context, string) (wafsdk.GetWorkRequestResponse, error)
}

func (f *fakeWebAppFirewallPolicyOCIClient) ChangeWebAppFirewallPolicyCompartment(ctx context.Context, request wafsdk.ChangeWebAppFirewallPolicyCompartmentRequest) (wafsdk.ChangeWebAppFirewallPolicyCompartmentResponse, error) {
	f.changeCalls++
	f.changeRequest = request
	if f.changeFunc != nil {
		return f.changeFunc(ctx, request)
	}
	return wafsdk.ChangeWebAppFirewallPolicyCompartmentResponse{}, nil
}

func (f *fakeWebAppFirewallPolicyOCIClient) CreateWebAppFirewallPolicy(ctx context.Context, request wafsdk.CreateWebAppFirewallPolicyRequest) (wafsdk.CreateWebAppFirewallPolicyResponse, error) {
	f.createCalls++
	f.createRequest = request
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return wafsdk.CreateWebAppFirewallPolicyResponse{}, nil
}

func (f *fakeWebAppFirewallPolicyOCIClient) GetWebAppFirewallPolicy(ctx context.Context, request wafsdk.GetWebAppFirewallPolicyRequest) (wafsdk.GetWebAppFirewallPolicyResponse, error) {
	f.getCalls++
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return wafsdk.GetWebAppFirewallPolicyResponse{}, nil
}

func (f *fakeWebAppFirewallPolicyOCIClient) ListWebAppFirewallPolicies(ctx context.Context, request wafsdk.ListWebAppFirewallPoliciesRequest) (wafsdk.ListWebAppFirewallPoliciesResponse, error) {
	f.listCalls++
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return wafsdk.ListWebAppFirewallPoliciesResponse{}, nil
}

func (f *fakeWebAppFirewallPolicyOCIClient) UpdateWebAppFirewallPolicy(ctx context.Context, request wafsdk.UpdateWebAppFirewallPolicyRequest) (wafsdk.UpdateWebAppFirewallPolicyResponse, error) {
	f.updateCalls++
	f.updateRequest = request
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return wafsdk.UpdateWebAppFirewallPolicyResponse{}, nil
}

func (f *fakeWebAppFirewallPolicyOCIClient) DeleteWebAppFirewallPolicy(ctx context.Context, request wafsdk.DeleteWebAppFirewallPolicyRequest) (wafsdk.DeleteWebAppFirewallPolicyResponse, error) {
	f.deleteCalls++
	f.deleteRequest = request
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return wafsdk.DeleteWebAppFirewallPolicyResponse{}, nil
}

func (f *fakeWebAppFirewallPolicyOCIClient) GetWorkRequest(ctx context.Context, request wafsdk.GetWorkRequestRequest) (wafsdk.GetWorkRequestResponse, error) {
	f.getWorkReqCalls++
	if f.getWorkRequestFunc != nil {
		return f.getWorkRequestFunc(ctx, webAppFirewallPolicyStringValue(request.WorkRequestId))
	}
	return wafsdk.GetWorkRequestResponse{}, nil
}

func newTestWebAppFirewallPolicy() *wafv1beta1.WebAppFirewallPolicy {
	return &wafv1beta1.WebAppFirewallPolicy{
		Spec: wafv1beta1.WebAppFirewallPolicySpec{
			CompartmentId: "compartment-a",
			DisplayName:   "policy-a",
			FreeformTags:  map[string]string{"env": "test"},
		},
	}
}

func testWebAppFirewallPolicy(
	id string,
	compartmentID string,
	displayName string,
	lifecycleState wafsdk.WebAppFirewallPolicyLifecycleStateEnum,
) wafsdk.WebAppFirewallPolicy {
	return wafsdk.WebAppFirewallPolicy{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: lifecycleState,
		FreeformTags:   map[string]string{"env": "test"},
	}
}

func testWebAppFirewallPolicyWorkRequest(
	id string,
	operationType wafsdk.WorkRequestOperationTypeEnum,
	status wafsdk.WorkRequestStatusEnum,
	actionType wafsdk.WorkRequestResourceActionTypeEnum,
	resourceID string,
) wafsdk.WorkRequest {
	return wafsdk.WorkRequest{
		Id:            common.String(id),
		OperationType: operationType,
		Status:        status,
		Resources: []wafsdk.WorkRequestResource{
			{
				EntityType: common.String("webAppFirewallPolicy"),
				ActionType: actionType,
				Identifier: common.String(resourceID),
			},
		},
	}
}

func requireWebAppFirewallPolicyAsync(
	t *testing.T,
	resource *wafv1beta1.WebAppFirewallPolicy,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil")
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

func requireWebAppFirewallPolicyReturnHTTPAction(
	t *testing.T,
	actions []wafsdk.Action,
	wantName string,
	wantCode int,
	wantBodyText string,
) {
	t.Helper()
	if len(actions) != 1 {
		t.Fatalf("create actions length = %d, want 1", len(actions))
	}
	action, ok := actions[0].(wafsdk.ReturnHttpResponseAction)
	if !ok {
		t.Fatalf("create action type = %T, want wafsdk.ReturnHttpResponseAction", actions[0])
	}
	if got := webAppFirewallPolicyStringValue(action.Name); got != wantName {
		t.Fatalf("action.Name = %q, want %q", got, wantName)
	}
	if action.Code == nil || *action.Code != wantCode {
		t.Fatalf("action.Code = %v, want %d", action.Code, wantCode)
	}
	body, ok := action.Body.(wafsdk.StaticTextHttpResponseBody)
	if !ok {
		t.Fatalf("action.Body type = %T, want wafsdk.StaticTextHttpResponseBody", action.Body)
	}
	if got := webAppFirewallPolicyStringValue(body.Text); got != wantBodyText {
		t.Fatalf("body.Text = %q, want %q", got, wantBodyText)
	}
}

func requireWebAppFirewallPolicyNestedModules(t *testing.T, details wafsdk.CreateWebAppFirewallPolicyDetails) {
	t.Helper()
	if got := webAppFirewallPolicyStringValue(details.RequestAccessControl.DefaultActionName); got != "allow" {
		t.Fatalf("requestAccessControl.defaultActionName = %q, want allow", got)
	}
	if got := len(details.RequestAccessControl.Rules); got != 1 {
		t.Fatalf("requestAccessControl.rules length = %d, want 1", got)
	}
	configuration := details.RequestRateLimiting.Rules[0].Configurations[0]
	if configuration.ActionDurationInSeconds == nil || *configuration.ActionDurationInSeconds != 0 {
		t.Fatalf("requestRateLimiting.rules[0].configurations[0].actionDurationInSeconds = %v, want explicit 0", configuration.ActionDurationInSeconds)
	}
	if got := *details.RequestProtection.BodyInspectionSizeLimitInBytes; got != 8192 {
		t.Fatalf("requestProtection.bodyInspectionSizeLimitInBytes = %d, want 8192", got)
	}
	if got := details.RequestProtection.Rules[0].ProtectionCapabilitySettings.AllowedHttpMethods; len(got) != 2 || got[0] != "GET" || got[1] != "POST" {
		t.Fatalf("requestProtection.rules[0].protectionCapabilitySettings.allowedHttpMethods = %#v, want [GET POST]", got)
	}
}

func requireWebAppFirewallPolicyOpcRequestID(t *testing.T, resource *wafv1beta1.WebAppFirewallPolicy, want string) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}
