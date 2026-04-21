/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package backendset

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	backendSetLoadBalancerID = "ocid1.loadbalancer.oc1..exampleuniqueID"
	backendSetNameValue      = "example_backend_set"
)

type fakeGeneratedBackendSetOCIClient struct {
	createFn func(context.Context, loadbalancersdk.CreateBackendSetRequest) (loadbalancersdk.CreateBackendSetResponse, error)
	getFn    func(context.Context, loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error)
	listFn   func(context.Context, loadbalancersdk.ListBackendSetsRequest) (loadbalancersdk.ListBackendSetsResponse, error)
	updateFn func(context.Context, loadbalancersdk.UpdateBackendSetRequest) (loadbalancersdk.UpdateBackendSetResponse, error)
	deleteFn func(context.Context, loadbalancersdk.DeleteBackendSetRequest) (loadbalancersdk.DeleteBackendSetResponse, error)
}

type backendSetLookupClient interface {
	GetBackendSet(context.Context, loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error)
}

type fakeBackendSetLookupClient struct {
	requests []loadbalancersdk.GetBackendSetRequest
	response loadbalancersdk.GetBackendSetResponse
	err      error
}

func (f *fakeBackendSetLookupClient) GetBackendSet(_ context.Context, request loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error) {
	f.requests = append(f.requests, request)
	if f.err != nil {
		return loadbalancersdk.GetBackendSetResponse{}, f.err
	}
	return f.response, nil
}

func (f *fakeGeneratedBackendSetOCIClient) CreateBackendSet(ctx context.Context, req loadbalancersdk.CreateBackendSetRequest) (loadbalancersdk.CreateBackendSetResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return loadbalancersdk.CreateBackendSetResponse{}, nil
}

func (f *fakeGeneratedBackendSetOCIClient) GetBackendSet(ctx context.Context, req loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return loadbalancersdk.GetBackendSetResponse{}, errortest.NewServiceError(404, "NotFound", "missing backend set")
}

func (f *fakeGeneratedBackendSetOCIClient) ListBackendSets(ctx context.Context, req loadbalancersdk.ListBackendSetsRequest) (loadbalancersdk.ListBackendSetsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return loadbalancersdk.ListBackendSetsResponse{}, nil
}

func (f *fakeGeneratedBackendSetOCIClient) UpdateBackendSet(ctx context.Context, req loadbalancersdk.UpdateBackendSetRequest) (loadbalancersdk.UpdateBackendSetResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return loadbalancersdk.UpdateBackendSetResponse{}, nil
}

func (f *fakeGeneratedBackendSetOCIClient) DeleteBackendSet(ctx context.Context, req loadbalancersdk.DeleteBackendSetRequest) (loadbalancersdk.DeleteBackendSetResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return loadbalancersdk.DeleteBackendSetResponse{}, nil
}

func newTestGeneratedBackendSetDelegate(client *fakeGeneratedBackendSetOCIClient) BackendSetServiceClient {
	hooks := newBackendSetRuntimeHooksWithOCIClient(client)
	applyBackendSetRuntimeHooks(&hooks)
	config := buildBackendSetGeneratedRuntimeConfig(&BackendSetServiceManager{}, hooks)

	return defaultBackendSetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.BackendSet](config),
	}
}

func newTestBackendSetRuntimeClient(client *fakeGeneratedBackendSetOCIClient) BackendSetServiceClient {
	return newTestBackendSetRuntimeClientWithLookup(client, nil)
}

func newTestBackendSetRuntimeClientWithLookup(client *fakeGeneratedBackendSetOCIClient, lookup backendSetLookupClient) BackendSetServiceClient {
	hooks := newBackendSetRuntimeHooksWithOCIClient(client)
	applyBackendSetRuntimeHooks(&hooks)
	if lookup != nil {
		hooks.Identity.LookupExisting = func(ctx context.Context, resource *loadbalancerv1beta1.BackendSet, identity any) (any, error) {
			if backendSetHasTrackedID(resource) {
				return nil, nil
			}
			resolved := identity.(backendSetIdentity)
			return lookup.GetBackendSet(ctx, loadbalancersdk.GetBackendSetRequest{
				LoadBalancerId: common.String(resolved.loadBalancerID),
				BackendSetName: common.String(resolved.backendSetName),
			})
		}
	}

	config := buildBackendSetGeneratedRuntimeConfig(&BackendSetServiceManager{}, hooks)
	return defaultBackendSetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.BackendSet](config),
	}
}

func TestBackendSetRuntimeSemanticsEncodesBaselineLifecycle(t *testing.T) {
	t.Parallel()

	got := newBackendSetRuntimeSemantics()
	if got == nil {
		t.Fatal("newBackendSetRuntimeSemantics() = nil")
	}
	if got.FormalService != "loadbalancer" {
		t.Fatalf("FormalService = %q, want loadbalancer", got.FormalService)
	}
	if got.FormalSlug != "backendset" {
		t.Fatalf("FormalSlug = %q, want backendset", got.FormalSlug)
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

	assertBackendSetStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertBackendSetStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertBackendSetStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"name"})
	assertBackendSetStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{
		"backends",
		"healthChecker",
		"lbCookieSessionPersistenceConfiguration",
		"policy",
		"sessionPersistenceConfiguration",
		"sslConfiguration",
	})
	assertBackendSetStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{
		"loadBalancerId",
		"name",
	})
	if !reflect.DeepEqual(got.Mutation.ConflictsWith, map[string][]string{
		"lbCookieSessionPersistenceConfiguration": {"sessionPersistenceConfiguration"},
		"sessionPersistenceConfiguration":         {"lbCookieSessionPersistenceConfiguration"},
	}) {
		t.Fatalf("Mutation.ConflictsWith = %#v, want mutual session persistence conflict", got.Mutation.ConflictsWith)
	}
}

func TestBackendSetRequestFieldsKeepTrackedOperationsScopedToRecordedPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  []generatedruntime.RequestField
		want []generatedruntime.RequestField
	}{
		{
			name: "create",
			got:  backendSetCreateFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:    "LoadBalancerId",
					RequestName:  "loadBalancerId",
					Contribution: "path",
					LookupPaths:  []string{"status.loadBalancerId", "spec.loadBalancerId"},
				},
				{
					FieldName:    "CreateBackendSetDetails",
					RequestName:  "CreateBackendSetDetails",
					Contribution: "body",
				},
			},
		},
		{
			name: "get",
			got:  backendSetGetFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:    "LoadBalancerId",
					RequestName:  "loadBalancerId",
					Contribution: "path",
					LookupPaths:  []string{"status.loadBalancerId", "spec.loadBalancerId"},
				},
				{
					FieldName:    "BackendSetName",
					RequestName:  "backendSetName",
					Contribution: "path",
					LookupPaths:  []string{"status.name", "spec.name", "name"},
				},
			},
		},
		{
			name: "list",
			got:  backendSetListFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:    "LoadBalancerId",
					RequestName:  "loadBalancerId",
					Contribution: "path",
					LookupPaths:  []string{"status.loadBalancerId", "spec.loadBalancerId"},
				},
			},
		},
		{
			name: "update",
			got:  backendSetUpdateFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:    "LoadBalancerId",
					RequestName:  "loadBalancerId",
					Contribution: "path",
					LookupPaths:  []string{"status.loadBalancerId", "spec.loadBalancerId"},
				},
				{
					FieldName:    "BackendSetName",
					RequestName:  "backendSetName",
					Contribution: "path",
					LookupPaths:  []string{"status.name", "spec.name", "name"},
				},
				{
					FieldName:    "UpdateBackendSetDetails",
					RequestName:  "UpdateBackendSetDetails",
					Contribution: "body",
				},
			},
		},
		{
			name: "delete",
			got:  backendSetDeleteFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:    "LoadBalancerId",
					RequestName:  "loadBalancerId",
					Contribution: "path",
					LookupPaths:  []string{"status.loadBalancerId", "spec.loadBalancerId"},
				},
				{
					FieldName:    "BackendSetName",
					RequestName:  "backendSetName",
					Contribution: "path",
					LookupPaths:  []string{"status.name", "spec.name", "name"},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !reflect.DeepEqual(tc.got, tc.want) {
				t.Fatalf("%s fields = %#v, want %#v", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestCreateOrUpdateCreatesBackendSetWhenMissing(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedBackendSetResource()

	getCalls := 0
	var createRequest loadbalancersdk.CreateBackendSetRequest

	client := newTestBackendSetRuntimeClient(&fakeGeneratedBackendSetOCIClient{
		getFn: func(_ context.Context, req loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error) {
			getCalls++
			assertBackendSetPathIdentity(t, req.LoadBalancerId, req.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
			if getCalls <= 2 {
				return loadbalancersdk.GetBackendSetResponse{}, errortest.NewServiceError(404, "NotFound", "missing backend set")
			}
			return loadbalancersdk.GetBackendSetResponse{BackendSet: sdkBackendSet(resource.Spec.Policy)}, nil
		},
		createFn: func(_ context.Context, req loadbalancersdk.CreateBackendSetRequest) (loadbalancersdk.CreateBackendSetResponse, error) {
			createRequest = req
			assertBackendSetPathIdentity(t, req.LoadBalancerId, common.String(backendSetNameValue), backendSetLoadBalancerID, backendSetNameValue)
			return loadbalancersdk.CreateBackendSetResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful create response", response)
	}
	assertBackendSetPathIdentity(t, createRequest.LoadBalancerId, common.String(backendSetNameValue), backendSetLoadBalancerID, backendSetNameValue)
	if got := stringValue(createRequest.CreateBackendSetDetails.Name); got != backendSetNameValue {
		t.Fatalf("CreateBackendSetDetails.Name = %q, want %q", got, backendSetNameValue)
	}
	if got := stringValue(createRequest.CreateBackendSetDetails.Policy); got != resource.Spec.Policy {
		t.Fatalf("CreateBackendSetDetails.Policy = %q, want %q", got, resource.Spec.Policy)
	}
	if got := resource.Status.LoadBalancerId; got != backendSetLoadBalancerID {
		t.Fatalf("status.loadBalancerId = %q, want %q", got, backendSetLoadBalancerID)
	}
	assertBackendSetTrackedStatus(t, resource, backendSetLoadBalancerID, backendSetNameValue)
}

func TestCreateOrUpdateBindsExistingBackendSetWithoutPreseededOCID(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedBackendSetResource()
	lookup := &fakeBackendSetLookupClient{
		response: loadbalancersdk.GetBackendSetResponse{BackendSet: sdkBackendSet(resource.Spec.Policy)},
	}

	getCalls := 0
	createCalled := false
	updateCalled := false

	client := newTestBackendSetRuntimeClientWithLookup(&fakeGeneratedBackendSetOCIClient{
		getFn: func(_ context.Context, req loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error) {
			getCalls++
			assertBackendSetPathIdentity(t, req.LoadBalancerId, req.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
			return loadbalancersdk.GetBackendSetResponse{BackendSet: sdkBackendSet(resource.Spec.Policy)}, nil
		},
		createFn: func(context.Context, loadbalancersdk.CreateBackendSetRequest) (loadbalancersdk.CreateBackendSetResponse, error) {
			createCalled = true
			return loadbalancersdk.CreateBackendSetResponse{}, nil
		},
		updateFn: func(context.Context, loadbalancersdk.UpdateBackendSetRequest) (loadbalancersdk.UpdateBackendSetResponse, error) {
			updateCalled = true
			return loadbalancersdk.UpdateBackendSetResponse{}, nil
		},
	}, lookup)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful bind response", response)
	}
	if createCalled {
		t.Fatal("CreateBackendSet() called, want bind/observe path")
	}
	if updateCalled {
		t.Fatal("UpdateBackendSet() called, want observe-only bind path")
	}
	if getCalls != 0 {
		t.Fatalf("delegate GetBackendSet() calls = %d, want 0", getCalls)
	}
	if len(lookup.requests) != 1 {
		t.Fatalf("lookup GetBackendSet() calls = %d, want 1", len(lookup.requests))
	}
	assertBackendSetPathIdentity(t, lookup.requests[0].LoadBalancerId, lookup.requests[0].BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
	assertBackendSetTrackedStatus(t, resource, backendSetLoadBalancerID, backendSetNameValue)
}

func TestCreateOrUpdateUpdatesBackendSetAfterCreateWithoutPreseededOCID(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedBackendSetResource()
	phase := "create"
	createGetCalls := 0
	updateGetCalls := 0
	var updateRequest loadbalancersdk.UpdateBackendSetRequest

	client := newTestBackendSetRuntimeClient(&fakeGeneratedBackendSetOCIClient{
		getFn: func(_ context.Context, req loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error) {
			assertBackendSetPathIdentity(t, req.LoadBalancerId, req.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
			switch phase {
			case "create":
				createGetCalls++
				if createGetCalls <= 2 {
					return loadbalancersdk.GetBackendSetResponse{}, errortest.NewServiceError(404, "NotFound", "missing backend set")
				}
				return loadbalancersdk.GetBackendSetResponse{BackendSet: sdkBackendSet("ROUND_ROBIN")}, nil
			case "update":
				updateGetCalls++
				if updateGetCalls == 1 {
					return loadbalancersdk.GetBackendSetResponse{BackendSet: sdkBackendSet("ROUND_ROBIN")}, nil
				}
				return loadbalancersdk.GetBackendSetResponse{BackendSet: sdkBackendSet("LEAST_CONNECTIONS")}, nil
			default:
				t.Fatalf("unexpected phase %q", phase)
				return loadbalancersdk.GetBackendSetResponse{}, nil
			}
		},
		createFn: func(_ context.Context, req loadbalancersdk.CreateBackendSetRequest) (loadbalancersdk.CreateBackendSetResponse, error) {
			assertBackendSetPathIdentity(t, req.LoadBalancerId, common.String(backendSetNameValue), backendSetLoadBalancerID, backendSetNameValue)
			return loadbalancersdk.CreateBackendSetResponse{}, nil
		},
		updateFn: func(_ context.Context, req loadbalancersdk.UpdateBackendSetRequest) (loadbalancersdk.UpdateBackendSetResponse, error) {
			updateRequest = req
			assertBackendSetPathIdentity(t, req.LoadBalancerId, req.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
			return loadbalancersdk.UpdateBackendSetResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("initial CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("initial CreateOrUpdate() response = %#v, want successful create response", response)
	}
	assertBackendSetTrackedStatus(t, resource, backendSetLoadBalancerID, backendSetNameValue)

	phase = "update"
	resource.Spec.Policy = "LEAST_CONNECTIONS"
	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("update CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("update CreateOrUpdate() response = %#v, want successful update response", response)
	}
	assertBackendSetPathIdentity(t, updateRequest.LoadBalancerId, updateRequest.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
	if got := stringValue(updateRequest.UpdateBackendSetDetails.Policy); got != "LEAST_CONNECTIONS" {
		t.Fatalf("UpdateBackendSetDetails.Policy = %q, want %q", got, "LEAST_CONNECTIONS")
	}
	if got := resource.Status.Policy; got != "LEAST_CONNECTIONS" {
		t.Fatalf("status.policy = %q, want %q", got, "LEAST_CONNECTIONS")
	}
	assertBackendSetTrackedStatus(t, resource, backendSetLoadBalancerID, backendSetNameValue)
}

func TestCreateOrUpdatePreservesNestedFalseBackendSetHealthCheckerBool(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendSetResource()
	resource.Spec.HealthChecker.IsForcePlainText = false
	resource.Status.HealthChecker.IsForcePlainText = true

	getCalls := 0
	updateCalled := false
	var updateRequest loadbalancersdk.UpdateBackendSetRequest

	client := newTestBackendSetRuntimeClient(&fakeGeneratedBackendSetOCIClient{
		getFn: func(_ context.Context, req loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error) {
			getCalls++
			assertBackendSetPathIdentity(t, req.LoadBalancerId, req.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
			switch getCalls {
			case 1:
				return loadbalancersdk.GetBackendSetResponse{
					BackendSet: sdkBackendSetWithHealthCheckerForcePlainText(resource.Status.Policy, true),
				}, nil
			case 2:
				return loadbalancersdk.GetBackendSetResponse{
					BackendSet: sdkBackendSetWithHealthCheckerForcePlainText(resource.Status.Policy, false),
				}, nil
			default:
				t.Fatalf("GetBackendSet() called %d times, want 2", getCalls)
				return loadbalancersdk.GetBackendSetResponse{}, nil
			}
		},
		updateFn: func(_ context.Context, req loadbalancersdk.UpdateBackendSetRequest) (loadbalancersdk.UpdateBackendSetResponse, error) {
			updateCalled = true
			updateRequest = req
			assertBackendSetPathIdentity(t, req.LoadBalancerId, req.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
			return loadbalancersdk.UpdateBackendSetResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful update response", response)
	}
	if !updateCalled {
		t.Fatal("UpdateBackendSet() not called, want nested bool update")
	}
	if updateRequest.UpdateBackendSetDetails.HealthChecker == nil {
		t.Fatal("UpdateBackendSetDetails.HealthChecker = nil, want projected health checker")
	}
	got := updateRequest.UpdateBackendSetDetails.HealthChecker.IsForcePlainText
	if got == nil || *got {
		t.Fatalf("UpdateBackendSetDetails.HealthChecker.IsForcePlainText = %#v, want explicit false", got)
	}
	if resource.Status.HealthChecker.IsForcePlainText {
		t.Fatal("status.healthChecker.isForcePlainText = true, want false after update projection")
	}
}

func TestCreateOrUpdateRejectsForceNewBackendSetDrift(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		mutate  func(*loadbalancerv1beta1.BackendSet)
		wantErr string
	}{
		{
			name: "loadBalancerId",
			mutate: func(resource *loadbalancerv1beta1.BackendSet) {
				resource.Spec.LoadBalancerId = "ocid1.loadbalancer.oc1..replacement"
			},
			wantErr: "require replacement when loadBalancerId changes",
		},
		{
			name: "name",
			mutate: func(resource *loadbalancerv1beta1.BackendSet) {
				resource.Spec.Name = "replacement_backend_set"
			},
			wantErr: "require replacement when name changes",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource := makeTrackedBackendSetResource()
			tc.mutate(resource)

			updateCalled := false
			client := newTestBackendSetRuntimeClient(&fakeGeneratedBackendSetOCIClient{
				getFn: func(_ context.Context, req loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error) {
					assertBackendSetPathIdentity(t, req.LoadBalancerId, req.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
					return loadbalancersdk.GetBackendSetResponse{BackendSet: sdkBackendSet("ROUND_ROBIN")}, nil
				},
				updateFn: func(context.Context, loadbalancersdk.UpdateBackendSetRequest) (loadbalancersdk.UpdateBackendSetResponse, error) {
					updateCalled = true
					return loadbalancersdk.UpdateBackendSetResponse{}, nil
				},
			})

			_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("CreateOrUpdate() error = %v, want substring %q", err, tc.wantErr)
			}
			if updateCalled {
				t.Fatal("UpdateBackendSet() called, want force-new drift rejection before update")
			}
		})
	}
}

func TestCreateOrUpdateRejectsConflictingBackendSetPersistenceModes(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendSetResource()
	resource.Spec.SessionPersistenceConfiguration = loadbalancerv1beta1.BackendSetSessionPersistenceConfiguration{
		CookieName: "app-cookie",
	}
	resource.Spec.LbCookieSessionPersistenceConfiguration = loadbalancerv1beta1.BackendSetLbCookieSessionPersistenceConfiguration{
		CookieName: "lb-cookie",
	}

	updateCalled := false
	client := newTestBackendSetRuntimeClient(&fakeGeneratedBackendSetOCIClient{
		getFn: func(_ context.Context, req loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error) {
			assertBackendSetPathIdentity(t, req.LoadBalancerId, req.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
			return loadbalancersdk.GetBackendSetResponse{BackendSet: sdkBackendSet(resource.Status.Policy)}, nil
		},
		updateFn: func(context.Context, loadbalancersdk.UpdateBackendSetRequest) (loadbalancersdk.UpdateBackendSetResponse, error) {
			updateCalled = true
			return loadbalancersdk.UpdateBackendSetResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil ||
		!strings.Contains(err.Error(), "BackendSet formal semantics forbid setting") ||
		!strings.Contains(err.Error(), "sessionPersistenceConfiguration") ||
		!strings.Contains(err.Error(), "lbCookieSessionPersistenceConfiguration") {
		t.Fatalf("CreateOrUpdate() error = %v, want session persistence conflict", err)
	}
	if updateCalled {
		t.Fatal("UpdateBackendSet() called, want conflictsWith rejection before update")
	}
}

func TestDeleteUsesRecordedBackendSetIdentity(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendSetResource()
	resource.Spec.LoadBalancerId = "ocid1.loadbalancer.oc1..replacement"
	resource.Spec.Name = "replacement_backend_set"

	getCalls := 0
	var deleteRequest loadbalancersdk.DeleteBackendSetRequest

	client := newTestBackendSetRuntimeClient(&fakeGeneratedBackendSetOCIClient{
		getFn: func(_ context.Context, req loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error) {
			getCalls++
			assertBackendSetPathIdentity(t, req.LoadBalancerId, req.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
			if getCalls == 1 {
				return loadbalancersdk.GetBackendSetResponse{BackendSet: sdkBackendSet(resource.Status.Policy)}, nil
			}
			return loadbalancersdk.GetBackendSetResponse{}, errortest.NewServiceError(404, "NotFound", "deleted backend set")
		},
		deleteFn: func(_ context.Context, req loadbalancersdk.DeleteBackendSetRequest) (loadbalancersdk.DeleteBackendSetResponse, error) {
			deleteRequest = req
			assertBackendSetPathIdentity(t, req.LoadBalancerId, req.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
			return loadbalancersdk.DeleteBackendSetResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want confirmed delete")
	}
	assertBackendSetPathIdentity(t, deleteRequest.LoadBalancerId, deleteRequest.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
	if got := resource.Status.LoadBalancerId; got != backendSetLoadBalancerID {
		t.Fatalf("status.loadBalancerId = %q, want %q", got, backendSetLoadBalancerID)
	}
}

func TestDeleteSucceedsAfterCreateWithoutPreseededOCID(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedBackendSetResource()
	phase := "create"
	createGetCalls := 0
	deleteGetCalls := 0
	var deleteRequest loadbalancersdk.DeleteBackendSetRequest

	client := newTestBackendSetRuntimeClient(&fakeGeneratedBackendSetOCIClient{
		getFn: func(_ context.Context, req loadbalancersdk.GetBackendSetRequest) (loadbalancersdk.GetBackendSetResponse, error) {
			assertBackendSetPathIdentity(t, req.LoadBalancerId, req.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
			switch phase {
			case "create":
				createGetCalls++
				if createGetCalls <= 2 {
					return loadbalancersdk.GetBackendSetResponse{}, errortest.NewServiceError(404, "NotFound", "missing backend set")
				}
				return loadbalancersdk.GetBackendSetResponse{BackendSet: sdkBackendSet("ROUND_ROBIN")}, nil
			case "delete":
				deleteGetCalls++
				if deleteGetCalls == 1 {
					return loadbalancersdk.GetBackendSetResponse{BackendSet: sdkBackendSet("ROUND_ROBIN")}, nil
				}
				return loadbalancersdk.GetBackendSetResponse{}, errortest.NewServiceError(404, "NotFound", "deleted backend set")
			default:
				t.Fatalf("unexpected phase %q", phase)
				return loadbalancersdk.GetBackendSetResponse{}, nil
			}
		},
		createFn: func(_ context.Context, req loadbalancersdk.CreateBackendSetRequest) (loadbalancersdk.CreateBackendSetResponse, error) {
			assertBackendSetPathIdentity(t, req.LoadBalancerId, common.String(backendSetNameValue), backendSetLoadBalancerID, backendSetNameValue)
			return loadbalancersdk.CreateBackendSetResponse{}, nil
		},
		deleteFn: func(_ context.Context, req loadbalancersdk.DeleteBackendSetRequest) (loadbalancersdk.DeleteBackendSetResponse, error) {
			deleteRequest = req
			assertBackendSetPathIdentity(t, req.LoadBalancerId, req.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
			return loadbalancersdk.DeleteBackendSetResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("initial CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("initial CreateOrUpdate() response = %#v, want successful create response", response)
	}
	assertBackendSetTrackedStatus(t, resource, backendSetLoadBalancerID, backendSetNameValue)

	phase = "delete"
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want confirmed delete")
	}
	assertBackendSetPathIdentity(t, deleteRequest.LoadBalancerId, deleteRequest.BackendSetName, backendSetLoadBalancerID, backendSetNameValue)
	assertBackendSetTrackedStatus(t, resource, backendSetLoadBalancerID, backendSetNameValue)
}

func makeUntrackedBackendSetResource() *loadbalancerv1beta1.BackendSet {
	return &loadbalancerv1beta1.BackendSet{
		Spec: loadbalancerv1beta1.BackendSetSpec{
			LoadBalancerId: backendSetLoadBalancerID,
			Name:           backendSetNameValue,
			Policy:         "ROUND_ROBIN",
			HealthChecker: loadbalancerv1beta1.BackendSetHealthChecker{
				Protocol:   "HTTP",
				UrlPath:    "/healthz",
				Port:       8080,
				ReturnCode: 200,
			},
			Backends: []loadbalancerv1beta1.BackendSetBackend{
				{
					IpAddress: "10.0.0.3",
					Port:      8080,
					Weight:    1,
				},
			},
		},
	}
}

func makeTrackedBackendSetResource() *loadbalancerv1beta1.BackendSet {
	resource := makeUntrackedBackendSetResource()
	resource.Status = loadbalancerv1beta1.BackendSetStatus{
		LoadBalancerId: backendSetLoadBalancerID,
		Name:           backendSetNameValue,
		Policy:         "ROUND_ROBIN",
		HealthChecker: loadbalancerv1beta1.BackendSetHealthChecker{
			Protocol:   "HTTP",
			UrlPath:    "/healthz",
			Port:       8080,
			ReturnCode: 200,
		},
		Backends: []loadbalancerv1beta1.BackendSetBackend{
			{
				IpAddress: "10.0.0.3",
				Port:      8080,
				Weight:    1,
			},
		},
		OsokStatus: shared.OSOKStatus{
			Ocid: backendSetSyntheticOCID(backendSetIdentity{
				loadBalancerID: backendSetLoadBalancerID,
				backendSetName: backendSetNameValue,
			}),
		},
	}
	return resource
}

func sdkBackendSet(policy string) loadbalancersdk.BackendSet {
	return loadbalancersdk.BackendSet{
		Name:   common.String(backendSetNameValue),
		Policy: common.String(policy),
		HealthChecker: &loadbalancersdk.HealthChecker{
			Protocol:   common.String("HTTP"),
			UrlPath:    common.String("/healthz"),
			Port:       common.Int(8080),
			ReturnCode: common.Int(200),
		},
		Backends: []loadbalancersdk.Backend{
			{
				Name:      common.String("10.0.0.3:8080"),
				IpAddress: common.String("10.0.0.3"),
				Port:      common.Int(8080),
				Weight:    common.Int(1),
			},
		},
	}
}

func sdkBackendSetWithHealthCheckerForcePlainText(policy string, isForcePlainText bool) loadbalancersdk.BackendSet {
	backendSet := sdkBackendSet(policy)
	backendSet.HealthChecker.IsForcePlainText = common.Bool(isForcePlainText)
	return backendSet
}

func assertBackendSetPathIdentity(t *testing.T, loadBalancerID, backendSetName *string, wantLoadBalancerID, wantBackendSetName string) {
	t.Helper()
	if got := stringValue(loadBalancerID); got != wantLoadBalancerID {
		t.Fatalf("LoadBalancerId = %q, want %q", got, wantLoadBalancerID)
	}
	if got := stringValue(backendSetName); got != wantBackendSetName {
		t.Fatalf("BackendSetName = %q, want %q", got, wantBackendSetName)
	}
}

func assertBackendSetStringSliceEqual(t *testing.T, name string, got, want []string) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	t.Fatalf("%s = %#v, want %#v", name, got, want)
}

func assertBackendSetTrackedStatus(t *testing.T, resource *loadbalancerv1beta1.BackendSet, wantLoadBalancerID, wantBackendSetName string) {
	t.Helper()
	if resource == nil {
		t.Fatal("resource = nil, want BackendSet")
	}
	if got := resource.Status.LoadBalancerId; got != wantLoadBalancerID {
		t.Fatalf("status.loadBalancerId = %q, want %q", got, wantLoadBalancerID)
	}
	if got := resource.Status.Name; got != wantBackendSetName {
		t.Fatalf("status.name = %q, want %q", got, wantBackendSetName)
	}
	wantTrackedID := backendSetSyntheticOCID(backendSetIdentity{
		loadBalancerID: wantLoadBalancerID,
		backendSetName: wantBackendSetName,
	})
	if got := resource.Status.OsokStatus.Ocid; got != wantTrackedID {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantTrackedID)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
