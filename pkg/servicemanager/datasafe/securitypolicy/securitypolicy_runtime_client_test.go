/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package securitypolicy

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeSecurityPolicyOCIClient struct {
	createSecurityPolicyFn func(context.Context, datasafesdk.CreateSecurityPolicyRequest) (datasafesdk.CreateSecurityPolicyResponse, error)
	getSecurityPolicyFn    func(context.Context, datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error)
	listSecurityPoliciesFn func(context.Context, datasafesdk.ListSecurityPoliciesRequest) (datasafesdk.ListSecurityPoliciesResponse, error)
	updateSecurityPolicyFn func(context.Context, datasafesdk.UpdateSecurityPolicyRequest) (datasafesdk.UpdateSecurityPolicyResponse, error)
	deleteSecurityPolicyFn func(context.Context, datasafesdk.DeleteSecurityPolicyRequest) (datasafesdk.DeleteSecurityPolicyResponse, error)
}

func (f *fakeSecurityPolicyOCIClient) CreateSecurityPolicy(
	ctx context.Context,
	req datasafesdk.CreateSecurityPolicyRequest,
) (datasafesdk.CreateSecurityPolicyResponse, error) {
	if f.createSecurityPolicyFn != nil {
		return f.createSecurityPolicyFn(ctx, req)
	}
	return datasafesdk.CreateSecurityPolicyResponse{}, nil
}

func (f *fakeSecurityPolicyOCIClient) GetSecurityPolicy(
	ctx context.Context,
	req datasafesdk.GetSecurityPolicyRequest,
) (datasafesdk.GetSecurityPolicyResponse, error) {
	if f.getSecurityPolicyFn != nil {
		return f.getSecurityPolicyFn(ctx, req)
	}
	return datasafesdk.GetSecurityPolicyResponse{}, nil
}

func (f *fakeSecurityPolicyOCIClient) ListSecurityPolicies(
	ctx context.Context,
	req datasafesdk.ListSecurityPoliciesRequest,
) (datasafesdk.ListSecurityPoliciesResponse, error) {
	if f.listSecurityPoliciesFn != nil {
		return f.listSecurityPoliciesFn(ctx, req)
	}
	return datasafesdk.ListSecurityPoliciesResponse{}, nil
}

func (f *fakeSecurityPolicyOCIClient) UpdateSecurityPolicy(
	ctx context.Context,
	req datasafesdk.UpdateSecurityPolicyRequest,
) (datasafesdk.UpdateSecurityPolicyResponse, error) {
	if f.updateSecurityPolicyFn != nil {
		return f.updateSecurityPolicyFn(ctx, req)
	}
	return datasafesdk.UpdateSecurityPolicyResponse{}, nil
}

func (f *fakeSecurityPolicyOCIClient) DeleteSecurityPolicy(
	ctx context.Context,
	req datasafesdk.DeleteSecurityPolicyRequest,
) (datasafesdk.DeleteSecurityPolicyResponse, error) {
	if f.deleteSecurityPolicyFn != nil {
		return f.deleteSecurityPolicyFn(ctx, req)
	}
	return datasafesdk.DeleteSecurityPolicyResponse{}, nil
}

func testSecurityPolicyClient(fake *fakeSecurityPolicyOCIClient) SecurityPolicyServiceClient {
	return newSecurityPolicyServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		fake,
	)
}

func makeSecurityPolicyResource() *datasafev1beta1.SecurityPolicy {
	return &datasafev1beta1.SecurityPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "security-policy",
			Namespace: "default",
		},
		Spec: datasafev1beta1.SecurityPolicySpec{
			CompartmentId: "ocid1.compartment.oc1..example",
			DisplayName:   "security-policy",
			Description:   "desired description",
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func makeSDKSecurityPolicy(
	id string,
	compartmentID string,
	displayName string,
	description string,
	state datasafesdk.SecurityPolicyLifecycleStateEnum,
) datasafesdk.SecurityPolicy {
	policy := datasafesdk.SecurityPolicy{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
		FreeformTags:   map[string]string{"env": "dev"},
		DefinedTags:    map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
	if description != "" {
		policy.Description = common.String(description)
	}
	return policy
}

func makeSDKSecurityPolicySummary(
	id string,
	compartmentID string,
	displayName string,
	state datasafesdk.SecurityPolicyLifecycleStateEnum,
) datasafesdk.SecurityPolicySummary {
	return datasafesdk.SecurityPolicySummary{
		Id:             common.String(id),
		CompartmentId:  common.String(compartmentID),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
	}
}

func requireStringPointer(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", name, got, want)
	}
}

func requireNilStringPointer(t *testing.T, name string, got *string) {
	t.Helper()
	if got != nil {
		t.Fatalf("%s = %v, want nil", name, got)
	}
}

func requireCallCount(t *testing.T, name string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s calls = %d, want %d", name, got, want)
	}
}

func requireCreateOrUpdateSuccess(t *testing.T, response servicemanager.OSOKResponse, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
}

func requireSecurityPolicyCreateBody(
	t *testing.T,
	request datasafesdk.CreateSecurityPolicyRequest,
	spec datasafev1beta1.SecurityPolicySpec,
) {
	t.Helper()
	requireStringPointer(t, "create compartmentId", request.CompartmentId, spec.CompartmentId)
	requireStringPointer(t, "create displayName", request.DisplayName, spec.DisplayName)
	requireStringPointer(t, "create description", request.Description, spec.Description)
}

func requireSecurityPolicyTrackedStatus(
	t *testing.T,
	resource *datasafev1beta1.SecurityPolicy,
	wantID string,
	wantState datasafesdk.SecurityPolicyLifecycleStateEnum,
) {
	t.Helper()
	if string(resource.Status.OsokStatus.Ocid) != wantID {
		t.Fatalf("status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, wantID)
	}
	if resource.Status.Id != wantID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, wantID)
	}
	if resource.Status.LifecycleState != string(wantState) {
		t.Fatalf("status.lifecycleState = %q, want %s", resource.Status.LifecycleState, wantState)
	}
}

func requireAsyncTracker(
	t *testing.T,
	status shared.OSOKStatus,
	wantPhase shared.OSOKAsyncPhase,
	wantWorkRequestID string,
) {
	t.Helper()
	current := status.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want tracker")
	}
	if current.Phase != wantPhase {
		t.Fatalf("status.async.current.phase = %q, want %s", current.Phase, wantPhase)
	}
	if current.WorkRequestID != wantWorkRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %s", current.WorkRequestID, wantWorkRequestID)
	}
}

func TestSecurityPolicyRuntimeHooksUseReviewedSemantics(t *testing.T) {
	t.Parallel()

	hooks := newSecurityPolicyRuntimeHooks(&SecurityPolicyServiceManager{}, datasafesdk.DataSafeClient{})

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if hooks.Semantics.Delete.Policy != "required" {
		t.Fatalf("delete policy = %q, want required", hooks.Semantics.Delete.Policy)
	}
	if !reflect.DeepEqual(hooks.List.Fields, securityPolicyListFields()) {
		t.Fatalf("list fields = %#v, want %#v", hooks.List.Fields, securityPolicyListFields())
	}
	if hooks.DeleteHooks.ConfirmRead == nil {
		t.Fatal("delete confirm read hook = nil, want conservative confirmation")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("delete error hook = nil, want conservative auth-shaped not-found handling")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("update body hook = nil, want tag-aware update shaping")
	}
}

func TestSecurityPolicyCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	var createRequest datasafesdk.CreateSecurityPolicyRequest
	var getRequest datasafesdk.GetSecurityPolicyRequest
	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		createSecurityPolicyFn: func(_ context.Context, req datasafesdk.CreateSecurityPolicyRequest) (datasafesdk.CreateSecurityPolicyResponse, error) {
			createRequest = req
			return datasafesdk.CreateSecurityPolicyResponse{
				OpcRequestId:     common.String("opc-create-1"),
				OpcWorkRequestId: common.String("wr-create-1"),
				SecurityPolicy: makeSDKSecurityPolicy(
					"ocid1.securitypolicy.oc1..created",
					"ocid1.compartment.oc1..example",
					"security-policy",
					"desired description",
					datasafesdk.SecurityPolicyLifecycleStateCreating,
				),
			}, nil
		},
		getSecurityPolicyFn: func(_ context.Context, req datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
			getRequest = req
			return datasafesdk.GetSecurityPolicyResponse{
				SecurityPolicy: makeSDKSecurityPolicy(
					"ocid1.securitypolicy.oc1..created",
					"ocid1.compartment.oc1..example",
					"security-policy",
					"desired description",
					datasafesdk.SecurityPolicyLifecycleStateActive,
				),
			}, nil
		},
	})

	resource := makeSecurityPolicyResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after ACTIVE readback")
	}
	requireSecurityPolicyCreateBody(t, createRequest, resource.Spec)
	requireStringPointer(t, "get securityPolicyId", getRequest.SecurityPolicyId, "ocid1.securitypolicy.oc1..created")
	requireSecurityPolicyTrackedStatus(t, resource, "ocid1.securitypolicy.oc1..created", datasafesdk.SecurityPolicyLifecycleStateActive)
	if resource.Status.OsokStatus.OpcRequestID != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-1", resource.Status.OsokStatus.OpcRequestID)
	}
}

func listSecurityPoliciesAcrossTwoPages(
	t *testing.T,
	listCalls *int,
) func(context.Context, datasafesdk.ListSecurityPoliciesRequest) (datasafesdk.ListSecurityPoliciesResponse, error) {
	return func(_ context.Context, req datasafesdk.ListSecurityPoliciesRequest) (datasafesdk.ListSecurityPoliciesResponse, error) {
		t.Helper()
		(*listCalls)++
		requireStringPointer(t, "list compartmentId", req.CompartmentId, "ocid1.compartment.oc1..example")
		requireStringPointer(t, "list displayName", req.DisplayName, "security-policy")
		requireNilStringPointer(t, "list securityPolicyId", req.SecurityPolicyId)
		if *listCalls == 1 {
			requireNilStringPointer(t, "first list page", req.Page)
			return datasafesdk.ListSecurityPoliciesResponse{
				OpcNextPage: common.String("page-2"),
				SecurityPolicyCollection: datasafesdk.SecurityPolicyCollection{
					Items: []datasafesdk.SecurityPolicySummary{
						makeSDKSecurityPolicySummary(
							"ocid1.securitypolicy.oc1..other",
							"ocid1.compartment.oc1..example",
							"other-policy",
							datasafesdk.SecurityPolicyLifecycleStateActive,
						),
					},
				},
			}, nil
		}
		if *listCalls == 2 {
			requireStringPointer(t, "second list page", req.Page, "page-2")
			return datasafesdk.ListSecurityPoliciesResponse{
				SecurityPolicyCollection: datasafesdk.SecurityPolicyCollection{
					Items: []datasafesdk.SecurityPolicySummary{
						makeSDKSecurityPolicySummary(
							"ocid1.securitypolicy.oc1..existing",
							"ocid1.compartment.oc1..example",
							"security-policy",
							datasafesdk.SecurityPolicyLifecycleStateActive,
						),
					},
				},
			}, nil
		}
		t.Fatalf("unexpected ListSecurityPolicies() call %d", *listCalls)
		return datasafesdk.ListSecurityPoliciesResponse{}, nil
	}
}

func getSecurityPolicyByID(
	t *testing.T,
	getCalls *int,
	wantID string,
) func(context.Context, datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
	return func(_ context.Context, req datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
		t.Helper()
		(*getCalls)++
		requireStringPointer(t, "get securityPolicyId", req.SecurityPolicyId, wantID)
		return datasafesdk.GetSecurityPolicyResponse{
			SecurityPolicy: makeSDKSecurityPolicy(
				wantID,
				"ocid1.compartment.oc1..example",
				"security-policy",
				"desired description",
				datasafesdk.SecurityPolicyLifecycleStateActive,
			),
		}, nil
	}
}

func listReplacementSecurityPolicy(
	t *testing.T,
	listCalls *int,
) func(context.Context, datasafesdk.ListSecurityPoliciesRequest) (datasafesdk.ListSecurityPoliciesResponse, error) {
	return func(_ context.Context, req datasafesdk.ListSecurityPoliciesRequest) (datasafesdk.ListSecurityPoliciesResponse, error) {
		t.Helper()
		(*listCalls)++
		requireStringPointer(t, "list compartmentId", req.CompartmentId, "ocid1.compartment.oc1..example")
		requireStringPointer(t, "list displayName", req.DisplayName, "security-policy")
		requireNilStringPointer(t, "list securityPolicyId", req.SecurityPolicyId)
		return datasafesdk.ListSecurityPoliciesResponse{
			SecurityPolicyCollection: datasafesdk.SecurityPolicyCollection{
				Items: []datasafesdk.SecurityPolicySummary{
					makeSDKSecurityPolicySummary(
						"ocid1.securitypolicy.oc1..replacement",
						"ocid1.compartment.oc1..example",
						"security-policy",
						datasafesdk.SecurityPolicyLifecycleStateActive,
					),
				},
			},
		}, nil
	}
}

func getStaleThenReplacementSecurityPolicy(
	t *testing.T,
	getCalls *int,
) func(context.Context, datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
	return func(_ context.Context, req datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
		t.Helper()
		(*getCalls)++
		if *getCalls == 1 {
			requireStringPointer(t, "get stale securityPolicyId", req.SecurityPolicyId, "ocid1.securitypolicy.oc1..stale")
			return datasafesdk.GetSecurityPolicyResponse{}, errortest.NewServiceError(404, "NotFound", "security policy not found")
		}
		if *getCalls == 2 {
			requireStringPointer(t, "get replacement securityPolicyId", req.SecurityPolicyId, "ocid1.securitypolicy.oc1..replacement")
			return datasafesdk.GetSecurityPolicyResponse{
				SecurityPolicy: makeSDKSecurityPolicy(
					"ocid1.securitypolicy.oc1..replacement",
					"ocid1.compartment.oc1..example",
					"security-policy",
					"desired description",
					datasafesdk.SecurityPolicyLifecycleStateActive,
				),
			}, nil
		}
		t.Fatalf("unexpected GetSecurityPolicy() call %d", *getCalls)
		return datasafesdk.GetSecurityPolicyResponse{}, nil
	}
}

func getSecurityPolicyForMutableDrift(
	t *testing.T,
	getCalls *int,
) func(context.Context, datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
	return func(_ context.Context, req datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
		t.Helper()
		(*getCalls)++
		requireStringPointer(t, "get securityPolicyId", req.SecurityPolicyId, "ocid1.securitypolicy.oc1..existing")
		description := "stale description"
		state := datasafesdk.SecurityPolicyLifecycleStateActive
		if *getCalls == 2 {
			description = "desired description"
			state = datasafesdk.SecurityPolicyLifecycleStateUpdating
		}
		return datasafesdk.GetSecurityPolicyResponse{
			SecurityPolicy: makeSDKSecurityPolicy(
				"ocid1.securitypolicy.oc1..existing",
				"ocid1.compartment.oc1..example",
				"security-policy",
				description,
				state,
			),
		}, nil
	}
}

func getSecurityPolicyForDelete(
	t *testing.T,
	getCalls *int,
) func(context.Context, datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
	return func(_ context.Context, req datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
		t.Helper()
		(*getCalls)++
		requireStringPointer(t, "get securityPolicyId", req.SecurityPolicyId, "ocid1.securitypolicy.oc1..existing")
		state := datasafesdk.SecurityPolicyLifecycleStateActive
		if *getCalls == 2 {
			state = datasafesdk.SecurityPolicyLifecycleStateDeleting
		}
		return datasafesdk.GetSecurityPolicyResponse{
			SecurityPolicy: makeSDKSecurityPolicy(
				"ocid1.securitypolicy.oc1..existing",
				"ocid1.compartment.oc1..example",
				"security-policy",
				"desired description",
				state,
			),
		}, nil
	}
}

func TestSecurityPolicyCreateOrUpdateBindsExistingPolicyAcrossListPages(t *testing.T) {
	t.Parallel()

	listCalls := 0
	getCalls := 0
	createCalls := 0
	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		listSecurityPoliciesFn: listSecurityPoliciesAcrossTwoPages(t, &listCalls),
		getSecurityPolicyFn:    getSecurityPolicyByID(t, &getCalls, "ocid1.securitypolicy.oc1..existing"),
		createSecurityPolicyFn: func(context.Context, datasafesdk.CreateSecurityPolicyRequest) (datasafesdk.CreateSecurityPolicyResponse, error) {
			createCalls++
			return datasafesdk.CreateSecurityPolicyResponse{}, nil
		},
	})

	resource := makeSecurityPolicyResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireCallCount(t, "CreateSecurityPolicy", createCalls, 0)
	requireCallCount(t, "ListSecurityPolicies", listCalls, 2)
	requireCallCount(t, "GetSecurityPolicy", getCalls, 1)
	if resource.Status.Id != "ocid1.securitypolicy.oc1..existing" {
		t.Fatalf("status.id = %q, want bound security policy ID", resource.Status.Id)
	}
}

func TestSecurityPolicyCreateOrUpdateRebindsWhenTrackedStatusIDIsStale(t *testing.T) {
	t.Parallel()

	getCalls := 0
	listCalls := 0
	createCalls := 0
	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		getSecurityPolicyFn:    getStaleThenReplacementSecurityPolicy(t, &getCalls),
		listSecurityPoliciesFn: listReplacementSecurityPolicy(t, &listCalls),
		createSecurityPolicyFn: func(context.Context, datasafesdk.CreateSecurityPolicyRequest) (datasafesdk.CreateSecurityPolicyResponse, error) {
			createCalls++
			return datasafesdk.CreateSecurityPolicyResponse{}, nil
		},
	})

	resource := makeSecurityPolicyResource()
	resource.Status.Id = "ocid1.securitypolicy.oc1..stale"
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.securitypolicy.oc1..stale")
	resource.Status.CompartmentId = "ocid1.compartment.oc1..old"
	resource.Status.DisplayName = "old-security-policy"
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireCallCount(t, "GetSecurityPolicy", getCalls, 2)
	requireCallCount(t, "ListSecurityPolicies", listCalls, 1)
	requireCallCount(t, "CreateSecurityPolicy", createCalls, 0)
	requireSecurityPolicyTrackedStatus(t, resource, "ocid1.securitypolicy.oc1..replacement", datasafesdk.SecurityPolicyLifecycleStateActive)
}

func TestSecurityPolicyCreateOrUpdateSkipsUpdateWhenMutableStateMatches(t *testing.T) {
	t.Parallel()

	getCalls := 0
	updateCalls := 0
	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		getSecurityPolicyFn: func(_ context.Context, req datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
			getCalls++
			if req.SecurityPolicyId == nil || *req.SecurityPolicyId != "ocid1.securitypolicy.oc1..existing" {
				t.Fatalf("get securityPolicyId = %v, want tracked security policy ID", req.SecurityPolicyId)
			}
			return datasafesdk.GetSecurityPolicyResponse{
				SecurityPolicy: makeSDKSecurityPolicy(
					"ocid1.securitypolicy.oc1..existing",
					"ocid1.compartment.oc1..example",
					"security-policy",
					"desired description",
					datasafesdk.SecurityPolicyLifecycleStateActive,
				),
			}, nil
		},
		updateSecurityPolicyFn: func(context.Context, datasafesdk.UpdateSecurityPolicyRequest) (datasafesdk.UpdateSecurityPolicyResponse, error) {
			updateCalls++
			return datasafesdk.UpdateSecurityPolicyResponse{}, nil
		},
	})

	resource := makeSecurityPolicyResource()
	resource.Status.Id = "ocid1.securitypolicy.oc1..existing"
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	}
	if getCalls != 1 {
		t.Fatalf("GetSecurityPolicy() calls = %d, want 1", getCalls)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateSecurityPolicy() calls = %d, want 0", updateCalls)
	}
}

func TestSecurityPolicyCreateOrUpdateUpdatesMutableDrift(t *testing.T) {
	t.Parallel()

	getCalls := 0
	var updateRequest datasafesdk.UpdateSecurityPolicyRequest
	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		getSecurityPolicyFn: getSecurityPolicyForMutableDrift(t, &getCalls),
		updateSecurityPolicyFn: func(_ context.Context, req datasafesdk.UpdateSecurityPolicyRequest) (datasafesdk.UpdateSecurityPolicyResponse, error) {
			updateRequest = req
			return datasafesdk.UpdateSecurityPolicyResponse{
				OpcRequestId:     common.String("opc-update-1"),
				OpcWorkRequestId: common.String("wr-update-1"),
			}, nil
		},
	})

	resource := makeSecurityPolicyResource()
	resource.Status.Id = "ocid1.securitypolicy.oc1..existing"
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue while update readback is UPDATING")
	}
	requireStringPointer(t, "update securityPolicyId", updateRequest.SecurityPolicyId, "ocid1.securitypolicy.oc1..existing")
	requireStringPointer(t, "update description", updateRequest.Description, "desired description")
	requireNilStringPointer(t, "update displayName", updateRequest.DisplayName)
	if resource.Status.OsokStatus.OpcRequestID != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-update-1", resource.Status.OsokStatus.OpcRequestID)
	}
	requireAsyncTracker(t, resource.Status.OsokStatus, shared.OSOKAsyncPhaseUpdate, "wr-update-1")
}

func TestSecurityPolicyCreateOrUpdateClearsMutableStrings(t *testing.T) {
	t.Parallel()

	getCalls := 0
	updateCalls := 0
	var updateRequest datasafesdk.UpdateSecurityPolicyRequest
	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		getSecurityPolicyFn: func(_ context.Context, req datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
			t.Helper()
			getCalls++
			requireStringPointer(t, "get securityPolicyId", req.SecurityPolicyId, "ocid1.securitypolicy.oc1..existing")
			displayName := "stale display"
			description := "stale description"
			if getCalls == 2 {
				displayName = ""
				description = ""
			}
			return datasafesdk.GetSecurityPolicyResponse{
				SecurityPolicy: makeSDKSecurityPolicy(
					"ocid1.securitypolicy.oc1..existing",
					"ocid1.compartment.oc1..example",
					displayName,
					description,
					datasafesdk.SecurityPolicyLifecycleStateActive,
				),
			}, nil
		},
		updateSecurityPolicyFn: func(_ context.Context, req datasafesdk.UpdateSecurityPolicyRequest) (datasafesdk.UpdateSecurityPolicyResponse, error) {
			updateCalls++
			updateRequest = req
			return datasafesdk.UpdateSecurityPolicyResponse{
				OpcRequestId: common.String("opc-update-clear-strings-1"),
			}, nil
		},
	})

	resource := makeSecurityPolicyResource()
	resource.Status.Id = "ocid1.securitypolicy.oc1..existing"
	resource.Spec.DisplayName = ""
	resource.Spec.Description = ""
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireCallCount(t, "GetSecurityPolicy", getCalls, 2)
	requireCallCount(t, "UpdateSecurityPolicy", updateCalls, 1)
	requireStringPointer(t, "update securityPolicyId", updateRequest.SecurityPolicyId, "ocid1.securitypolicy.oc1..existing")
	requireStringPointer(t, "update displayName", updateRequest.DisplayName, "")
	requireStringPointer(t, "update description", updateRequest.Description, "")
	if resource.Status.DisplayName != "" {
		t.Fatalf("status.displayName = %q, want cleared value", resource.Status.DisplayName)
	}
	if resource.Status.Description != "" {
		t.Fatalf("status.description = %q, want cleared value", resource.Status.Description)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update-clear-strings-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-update-clear-strings-1", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestSecurityPolicyCreateOrUpdateClearsExplicitEmptyTagMaps(t *testing.T) {
	t.Parallel()

	getCalls := 0
	updateCalls := 0
	var updateRequest datasafesdk.UpdateSecurityPolicyRequest
	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		getSecurityPolicyFn: func(_ context.Context, req datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
			t.Helper()
			getCalls++
			requireStringPointer(t, "get securityPolicyId", req.SecurityPolicyId, "ocid1.securitypolicy.oc1..existing")
			policy := makeSDKSecurityPolicy(
				"ocid1.securitypolicy.oc1..existing",
				"ocid1.compartment.oc1..example",
				"security-policy",
				"desired description",
				datasafesdk.SecurityPolicyLifecycleStateActive,
			)
			if getCalls == 2 {
				policy.FreeformTags = map[string]string{}
				policy.DefinedTags = map[string]map[string]interface{}{}
			}
			return datasafesdk.GetSecurityPolicyResponse{SecurityPolicy: policy}, nil
		},
		updateSecurityPolicyFn: func(_ context.Context, req datasafesdk.UpdateSecurityPolicyRequest) (datasafesdk.UpdateSecurityPolicyResponse, error) {
			updateCalls++
			updateRequest = req
			return datasafesdk.UpdateSecurityPolicyResponse{
				OpcRequestId: common.String("opc-update-tags-1"),
			}, nil
		},
	})

	resource := makeSecurityPolicyResource()
	resource.Status.Id = "ocid1.securitypolicy.oc1..existing"
	resource.Spec.FreeformTags = map[string]string{}
	resource.Spec.DefinedTags = map[string]shared.MapValue{}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireCallCount(t, "GetSecurityPolicy", getCalls, 2)
	requireCallCount(t, "UpdateSecurityPolicy", updateCalls, 1)
	requireStringPointer(t, "update securityPolicyId", updateRequest.SecurityPolicyId, "ocid1.securitypolicy.oc1..existing")
	requireNilStringPointer(t, "update displayName", updateRequest.DisplayName)
	requireNilStringPointer(t, "update description", updateRequest.Description)
	if updateRequest.FreeformTags == nil {
		t.Fatal("update freeformTags = nil, want explicit empty map")
	}
	if len(updateRequest.FreeformTags) != 0 {
		t.Fatalf("update freeformTags = %#v, want empty map", updateRequest.FreeformTags)
	}
	if updateRequest.DefinedTags == nil {
		t.Fatal("update definedTags = nil, want explicit empty map")
	}
	if len(updateRequest.DefinedTags) != 0 {
		t.Fatalf("update definedTags = %#v, want empty map", updateRequest.DefinedTags)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update-tags-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-update-tags-1", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestSecurityPolicyCreateOrUpdateOmitsNilTagMaps(t *testing.T) {
	t.Parallel()

	getCalls := 0
	updateCalls := 0
	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		getSecurityPolicyFn: func(_ context.Context, req datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
			t.Helper()
			getCalls++
			requireStringPointer(t, "get securityPolicyId", req.SecurityPolicyId, "ocid1.securitypolicy.oc1..existing")
			return datasafesdk.GetSecurityPolicyResponse{
				SecurityPolicy: makeSDKSecurityPolicy(
					"ocid1.securitypolicy.oc1..existing",
					"ocid1.compartment.oc1..example",
					"security-policy",
					"desired description",
					datasafesdk.SecurityPolicyLifecycleStateActive,
				),
			}, nil
		},
		updateSecurityPolicyFn: func(context.Context, datasafesdk.UpdateSecurityPolicyRequest) (datasafesdk.UpdateSecurityPolicyResponse, error) {
			updateCalls++
			return datasafesdk.UpdateSecurityPolicyResponse{}, nil
		},
	})

	resource := makeSecurityPolicyResource()
	resource.Status.Id = "ocid1.securitypolicy.oc1..existing"
	resource.Spec.FreeformTags = nil
	resource.Spec.DefinedTags = nil
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireCallCount(t, "GetSecurityPolicy", getCalls, 1)
	requireCallCount(t, "UpdateSecurityPolicy", updateCalls, 0)
}

func TestSecurityPolicyCreateOrUpdateRejectsCompartmentDrift(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		getSecurityPolicyFn: func(_ context.Context, req datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
			if req.SecurityPolicyId == nil || *req.SecurityPolicyId != "ocid1.securitypolicy.oc1..existing" {
				t.Fatalf("get securityPolicyId = %v, want tracked security policy ID", req.SecurityPolicyId)
			}
			return datasafesdk.GetSecurityPolicyResponse{
				SecurityPolicy: makeSDKSecurityPolicy(
					"ocid1.securitypolicy.oc1..existing",
					"ocid1.compartment.oc1..observed",
					"security-policy",
					"desired description",
					datasafesdk.SecurityPolicyLifecycleStateActive,
				),
			}, nil
		},
		updateSecurityPolicyFn: func(context.Context, datasafesdk.UpdateSecurityPolicyRequest) (datasafesdk.UpdateSecurityPolicyResponse, error) {
			updateCalls++
			return datasafesdk.UpdateSecurityPolicyResponse{}, nil
		},
	})

	resource := makeSecurityPolicyResource()
	resource.Status.Id = "ocid1.securitypolicy.oc1..existing"
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..desired"
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want replacement validation error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should fail when compartmentId drift requires replacement")
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateSecurityPolicy() calls = %d, want 0", updateCalls)
	}
	if !strings.Contains(err.Error(), "SecurityPolicy formal semantics require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want compartmentId replacement message", err)
	}
}

func TestSecurityPolicyCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		createSecurityPolicyFn: func(context.Context, datasafesdk.CreateSecurityPolicyRequest) (datasafesdk.CreateSecurityPolicyResponse, error) {
			return datasafesdk.CreateSecurityPolicyResponse{}, errortest.NewServiceError(500, "InternalError", "create failed")
		},
	})

	resource := makeSecurityPolicyResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI service error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report failure after OCI service error")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestSecurityPolicyDeleteRetainsFinalizerUntilDeleteIsConfirmed(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0
	var deleteRequest datasafesdk.DeleteSecurityPolicyRequest
	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		getSecurityPolicyFn: getSecurityPolicyForDelete(t, &getCalls),
		deleteSecurityPolicyFn: func(_ context.Context, req datasafesdk.DeleteSecurityPolicyRequest) (datasafesdk.DeleteSecurityPolicyResponse, error) {
			deleteCalls++
			deleteRequest = req
			return datasafesdk.DeleteSecurityPolicyResponse{
				OpcRequestId:     common.String("opc-delete-1"),
				OpcWorkRequestId: common.String("wr-delete-1"),
			}, nil
		},
	})

	resource := makeSecurityPolicyResource()
	resource.Status.Id = "ocid1.securitypolicy.oc1..existing"
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep the finalizer while OCI reports DELETING")
	}
	requireCallCount(t, "GetSecurityPolicy", getCalls, 2)
	requireCallCount(t, "DeleteSecurityPolicy", deleteCalls, 1)
	requireStringPointer(t, "delete securityPolicyId", deleteRequest.SecurityPolicyId, "ocid1.securitypolicy.oc1..existing")
	if resource.Status.LifecycleState != string(datasafesdk.SecurityPolicyLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want Terminating", resource.Status.OsokStatus.Reason)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-1", resource.Status.OsokStatus.OpcRequestID)
	}
	requireAsyncTracker(t, resource.Status.OsokStatus, shared.OSOKAsyncPhaseDelete, "wr-delete-1")
}

func TestSecurityPolicyDeleteReturnsAuthShapedDeleteErrorAfterPreRead(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0
	var deleteRequest datasafesdk.DeleteSecurityPolicyRequest
	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		getSecurityPolicyFn: func(_ context.Context, req datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
			getCalls++
			requireStringPointer(t, "get securityPolicyId", req.SecurityPolicyId, "ocid1.securitypolicy.oc1..existing")
			return datasafesdk.GetSecurityPolicyResponse{
				SecurityPolicy: makeSDKSecurityPolicy(
					"ocid1.securitypolicy.oc1..existing",
					"ocid1.compartment.oc1..example",
					"security-policy",
					"desired description",
					datasafesdk.SecurityPolicyLifecycleStateActive,
				),
			}, nil
		},
		deleteSecurityPolicyFn: func(_ context.Context, req datasafesdk.DeleteSecurityPolicyRequest) (datasafesdk.DeleteSecurityPolicyResponse, error) {
			deleteCalls++
			deleteRequest = req
			return datasafesdk.DeleteSecurityPolicyResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "ambiguous delete")
		},
	})

	resource := makeSecurityPolicyResource()
	resource.Status.Id = "ocid1.securitypolicy.oc1..existing"
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want conservative auth-shaped delete error")
	}
	if deleted {
		t.Fatal("Delete() should keep the finalizer when DeleteSecurityPolicy returns ambiguous NotAuthorizedOrNotFound")
	}
	requireCallCount(t, "GetSecurityPolicy", getCalls, 1)
	requireCallCount(t, "DeleteSecurityPolicy", deleteCalls, 1)
	requireStringPointer(t, "delete securityPolicyId", deleteRequest.SecurityPolicyId, "ocid1.securitypolicy.oc1..existing")
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous 404 message", err)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay nil when delete returned an ambiguous auth-shaped error")
	}
}

func TestSecurityPolicyDeleteRejectsAuthShapedConfirmRead(t *testing.T) {
	t.Parallel()

	deleteCalls := 0
	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		getSecurityPolicyFn: func(context.Context, datasafesdk.GetSecurityPolicyRequest) (datasafesdk.GetSecurityPolicyResponse, error) {
			return datasafesdk.GetSecurityPolicyResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "ambiguous read")
		},
		deleteSecurityPolicyFn: func(context.Context, datasafesdk.DeleteSecurityPolicyRequest) (datasafesdk.DeleteSecurityPolicyResponse, error) {
			deleteCalls++
			return datasafesdk.DeleteSecurityPolicyResponse{}, nil
		},
	})

	resource := makeSecurityPolicyResource()
	resource.Status.Id = "ocid1.securitypolicy.oc1..existing"
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want conservative auth-shaped readback error")
	}
	if deleted {
		t.Fatal("Delete() should not report deleted for ambiguous NotAuthorizedOrNotFound")
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteSecurityPolicy() calls = %d, want 0 after ambiguous pre-delete read", deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous 404 message", err)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestSecurityPolicyDeleteConfirmsMissingUntrackedPolicyByList(t *testing.T) {
	t.Parallel()

	listCalls := 0
	deleteCalls := 0
	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		listSecurityPoliciesFn: func(_ context.Context, req datasafesdk.ListSecurityPoliciesRequest) (datasafesdk.ListSecurityPoliciesResponse, error) {
			listCalls++
			requireStringPointer(t, "list compartmentId", req.CompartmentId, "ocid1.compartment.oc1..example")
			requireStringPointer(t, "list displayName", req.DisplayName, "security-policy")
			if listCalls == 1 {
				requireNilStringPointer(t, "first list page", req.Page)
				return datasafesdk.ListSecurityPoliciesResponse{
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			if listCalls == 2 {
				requireStringPointer(t, "second list page", req.Page, "page-2")
				return datasafesdk.ListSecurityPoliciesResponse{}, nil
			}
			t.Fatalf("unexpected ListSecurityPolicies() call %d", listCalls)
			return datasafesdk.ListSecurityPoliciesResponse{}, nil
		},
		deleteSecurityPolicyFn: func(context.Context, datasafesdk.DeleteSecurityPolicyRequest) (datasafesdk.DeleteSecurityPolicyResponse, error) {
			deleteCalls++
			return datasafesdk.DeleteSecurityPolicyResponse{}, nil
		},
	})

	resource := makeSecurityPolicyResource()
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should confirm absence when all list pages miss the desired policy")
	}
	requireCallCount(t, "ListSecurityPolicies", listCalls, 2)
	requireCallCount(t, "DeleteSecurityPolicy", deleteCalls, 0)
}

func TestSecurityPolicyDeleteRejectsAuthShapedListConfirmRead(t *testing.T) {
	t.Parallel()

	deleteCalls := 0
	client := testSecurityPolicyClient(&fakeSecurityPolicyOCIClient{
		listSecurityPoliciesFn: func(context.Context, datasafesdk.ListSecurityPoliciesRequest) (datasafesdk.ListSecurityPoliciesResponse, error) {
			return datasafesdk.ListSecurityPoliciesResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "ambiguous list")
		},
		deleteSecurityPolicyFn: func(context.Context, datasafesdk.DeleteSecurityPolicyRequest) (datasafesdk.DeleteSecurityPolicyResponse, error) {
			deleteCalls++
			return datasafesdk.DeleteSecurityPolicyResponse{}, nil
		},
	})

	resource := makeSecurityPolicyResource()
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want conservative auth-shaped list readback error")
	}
	if deleted {
		t.Fatal("Delete() should not report deleted for ambiguous list NotAuthorizedOrNotFound")
	}
	requireCallCount(t, "DeleteSecurityPolicy", deleteCalls, 0)
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want ambiguous 404 message", err)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
}
