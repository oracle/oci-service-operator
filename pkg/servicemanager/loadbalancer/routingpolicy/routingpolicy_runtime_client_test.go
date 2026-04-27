/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package routingpolicy

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	loadbalancersdk "github.com/oracle/oci-go-sdk/v65/loadbalancer"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	routingPolicyLoadBalancerID = "ocid1.loadbalancer.oc1..exampleuniqueID"
	routingPolicyNameValue      = "example_routing_policy"
	routingPolicyBackendSetName = "example_backend_set"
)

type fakeGeneratedRoutingPolicyOCIClient struct {
	createRequests []loadbalancersdk.CreateRoutingPolicyRequest
	getRequests    []loadbalancersdk.GetRoutingPolicyRequest
	listRequests   []loadbalancersdk.ListRoutingPoliciesRequest
	updateRequests []loadbalancersdk.UpdateRoutingPolicyRequest
	deleteRequests []loadbalancersdk.DeleteRoutingPolicyRequest
	workRequests   []loadbalancersdk.GetWorkRequestRequest

	getErr        error
	createErr     error
	listErr       error
	updateErr     error
	deleteErr     error
	getWorkReqErr error

	keepAfterDelete bool
	routingPolicies map[string]loadbalancersdk.RoutingPolicy
	workRequestByID map[string]loadbalancersdk.WorkRequest
}

func (f *fakeGeneratedRoutingPolicyOCIClient) CreateRoutingPolicy(_ context.Context, request loadbalancersdk.CreateRoutingPolicyRequest) (loadbalancersdk.CreateRoutingPolicyResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return loadbalancersdk.CreateRoutingPolicyResponse{}, f.createErr
	}
	f.ensureRoutingPolicies()
	routingPolicy := routingPolicyFromCreateDetails(request.CreateRoutingPolicyDetails)
	f.routingPolicies[stringValue(routingPolicy.Name)] = routingPolicy
	return loadbalancersdk.CreateRoutingPolicyResponse{OpcWorkRequestId: common.String("wr-create-routing-policy")}, nil
}

func (f *fakeGeneratedRoutingPolicyOCIClient) GetRoutingPolicy(_ context.Context, request loadbalancersdk.GetRoutingPolicyRequest) (loadbalancersdk.GetRoutingPolicyResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getErr != nil {
		return loadbalancersdk.GetRoutingPolicyResponse{}, f.getErr
	}
	routingPolicy, ok := f.routingPolicies[stringValue(request.RoutingPolicyName)]
	if !ok {
		return loadbalancersdk.GetRoutingPolicyResponse{}, errortest.NewServiceError(404, "NotFound", "missing routing policy")
	}
	return loadbalancersdk.GetRoutingPolicyResponse{RoutingPolicy: routingPolicy}, nil
}

func (f *fakeGeneratedRoutingPolicyOCIClient) ListRoutingPolicies(_ context.Context, request loadbalancersdk.ListRoutingPoliciesRequest) (loadbalancersdk.ListRoutingPoliciesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listErr != nil {
		return loadbalancersdk.ListRoutingPoliciesResponse{}, f.listErr
	}
	names := make([]string, 0, len(f.routingPolicies))
	for name := range f.routingPolicies {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]loadbalancersdk.RoutingPolicy, 0, len(names))
	for _, name := range names {
		items = append(items, f.routingPolicies[name])
	}
	return loadbalancersdk.ListRoutingPoliciesResponse{Items: items}, nil
}

func (f *fakeGeneratedRoutingPolicyOCIClient) UpdateRoutingPolicy(_ context.Context, request loadbalancersdk.UpdateRoutingPolicyRequest) (loadbalancersdk.UpdateRoutingPolicyResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return loadbalancersdk.UpdateRoutingPolicyResponse{}, f.updateErr
	}
	f.ensureRoutingPolicies()
	name := stringValue(request.RoutingPolicyName)
	existing := f.routingPolicies[name]
	f.routingPolicies[name] = routingPolicyFromUpdateDetails(name, request.UpdateRoutingPolicyDetails, existing)
	return loadbalancersdk.UpdateRoutingPolicyResponse{OpcWorkRequestId: common.String("wr-update-routing-policy")}, nil
}

func (f *fakeGeneratedRoutingPolicyOCIClient) DeleteRoutingPolicy(_ context.Context, request loadbalancersdk.DeleteRoutingPolicyRequest) (loadbalancersdk.DeleteRoutingPolicyResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return loadbalancersdk.DeleteRoutingPolicyResponse{}, f.deleteErr
	}
	if !f.keepAfterDelete {
		delete(f.routingPolicies, stringValue(request.RoutingPolicyName))
	}
	return loadbalancersdk.DeleteRoutingPolicyResponse{OpcWorkRequestId: common.String("wr-delete-routing-policy")}, nil
}

func (f *fakeGeneratedRoutingPolicyOCIClient) GetWorkRequest(_ context.Context, request loadbalancersdk.GetWorkRequestRequest) (loadbalancersdk.GetWorkRequestResponse, error) {
	f.workRequests = append(f.workRequests, request)
	if f.getWorkReqErr != nil {
		return loadbalancersdk.GetWorkRequestResponse{}, f.getWorkReqErr
	}
	workRequest, ok := f.workRequestByID[stringValue(request.WorkRequestId)]
	if !ok {
		return loadbalancersdk.GetWorkRequestResponse{}, errortest.NewServiceError(404, "NotFound", "missing work request")
	}
	return loadbalancersdk.GetWorkRequestResponse{WorkRequest: workRequest}, nil
}

func (f *fakeGeneratedRoutingPolicyOCIClient) ensureRoutingPolicies() {
	if f.routingPolicies == nil {
		f.routingPolicies = map[string]loadbalancersdk.RoutingPolicy{}
	}
}

func newTestRoutingPolicyRuntimeClient(client *fakeGeneratedRoutingPolicyOCIClient) RoutingPolicyServiceClient {
	hooks := newRoutingPolicyRuntimeHooksWithOCIClient(client)
	applyRoutingPolicyRuntimeHooks(&hooks, client, nil)
	config := buildRoutingPolicyGeneratedRuntimeConfig(&RoutingPolicyServiceManager{}, hooks)
	delegate := defaultRoutingPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loadbalancerv1beta1.RoutingPolicy](config),
	}
	return wrapRoutingPolicyGeneratedClient(hooks, delegate)
}

func TestRoutingPolicyRuntimeSemanticsEncodesLifecycleContract(t *testing.T) {
	t.Parallel()

	got := newRoutingPolicyRuntimeSemantics()
	if got == nil {
		t.Fatal("newRoutingPolicyRuntimeSemantics() = nil")
	}
	if got.FormalService != "loadbalancer" {
		t.Fatalf("FormalService = %q, want loadbalancer", got.FormalService)
	}
	if got.FormalSlug != "routingpolicy" {
		t.Fatalf("FormalSlug = %q, want routingpolicy", got.FormalSlug)
	}
	if got.Async == nil {
		t.Fatal("Async = nil, want workrequest semantics")
	}
	if got.Async.Strategy != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got.Async.Strategy)
	}
	if got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async.Runtime = %q, want generatedruntime", got.Async.Runtime)
	}
	if got.Async.FormalClassification != "workrequest" {
		t.Fatalf("Async.FormalClassification = %q, want workrequest", got.Async.FormalClassification)
	}
	if got.Async.WorkRequest == nil {
		t.Fatal("Async.WorkRequest = nil, want workrequest metadata")
	}
	if got.Async.WorkRequest.Source != "service-sdk" {
		t.Fatalf("Async.WorkRequest.Source = %q, want service-sdk", got.Async.WorkRequest.Source)
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

	assertRoutingPolicyStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertRoutingPolicyStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertRoutingPolicyStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"name"})
	assertRoutingPolicyStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"conditionLanguageVersion", "rules"})
	assertRoutingPolicyStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"name"})
	assertRoutingPolicyStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
}

func TestRoutingPolicyRequestFieldsKeepOperationsScopedToRecordedPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  []generatedruntime.RequestField
		want []generatedruntime.RequestField
	}{
		{
			name: "create",
			got:  routingPolicyCreateFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "CreateRoutingPolicyDetails",
					RequestName:  "CreateRoutingPolicyDetails",
					Contribution: "body",
				},
			},
		},
		{
			name: "get",
			got:  routingPolicyGetFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "RoutingPolicyName",
					RequestName:  "routingPolicyName",
					Contribution: "path",
					LookupPaths:  []string{"status.name", "spec.name", "name"},
				},
			},
		},
		{
			name: "list",
			got:  routingPolicyListFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
			},
		},
		{
			name: "update",
			got:  routingPolicyUpdateFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "RoutingPolicyName",
					RequestName:  "routingPolicyName",
					Contribution: "path",
					LookupPaths:  []string{"status.name", "spec.name", "name"},
				},
				{
					FieldName:    "UpdateRoutingPolicyDetails",
					RequestName:  "UpdateRoutingPolicyDetails",
					Contribution: "body",
				},
			},
		},
		{
			name: "delete",
			got:  routingPolicyDeleteFields(),
			want: []generatedruntime.RequestField{
				{
					FieldName:        "LoadBalancerId",
					RequestName:      "loadBalancerId",
					Contribution:     "path",
					PreferResourceID: true,
					LookupPaths:      []string{"status.status.ocid"},
				},
				{
					FieldName:    "RoutingPolicyName",
					RequestName:  "routingPolicyName",
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

func TestRoutingPolicyWorkRequestAdapterMapsLoadBalancerStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status string
		want   shared.OSOKAsyncNormalizedClass
	}{
		{status: string(loadbalancersdk.WorkRequestLifecycleStateAccepted), want: shared.OSOKAsyncClassPending},
		{status: string(loadbalancersdk.WorkRequestLifecycleStateInProgress), want: shared.OSOKAsyncClassPending},
		{status: string(loadbalancersdk.WorkRequestLifecycleStateSucceeded), want: shared.OSOKAsyncClassSucceeded},
		{status: string(loadbalancersdk.WorkRequestLifecycleStateFailed), want: shared.OSOKAsyncClassFailed},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.status, func(t *testing.T) {
			t.Parallel()
			got, err := routingPolicyWorkRequestAsyncAdapter.Normalize(tc.status)
			if err != nil {
				t.Fatalf("Normalize(%q) error = %v", tc.status, err)
			}
			if got != tc.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestCreateOrUpdateRejectsMissingRoutingPolicyLoadBalancerAnnotation(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedRoutingPolicyResource()
	resource.Annotations = nil
	client := &fakeGeneratedRoutingPolicyOCIClient{}

	response, err := newTestRoutingPolicyRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), routingPolicyLoadBalancerIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want missing %s annotation", err, routingPolicyLoadBalancerIDAnnotation)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failed response", response)
	}
	if len(client.createRequests) != 0 || len(client.getRequests) != 0 || len(client.listRequests) != 0 {
		t.Fatalf("OCI calls = create:%d get:%d list:%d, want none", len(client.createRequests), len(client.getRequests), len(client.listRequests))
	}
}

func TestCreateOrUpdateCreatesThenObservesRoutingPolicy(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedRoutingPolicyOCIClient{
		routingPolicies: map[string]loadbalancersdk.RoutingPolicy{},
		workRequestByID: map[string]loadbalancersdk.WorkRequest{
			"wr-create-routing-policy": routingPolicyWorkRequest("wr-create-routing-policy", loadbalancersdk.WorkRequestLifecycleStateInProgress, "CreateRoutingPolicy"),
		},
	}
	serviceClient := newTestRoutingPolicyRuntimeClient(client)
	resource := makeUntrackedRoutingPolicyResource()

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful create response", response)
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want create work request requeue")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(client.createRequests))
	}
	if len(client.workRequests) != 1 {
		t.Fatalf("work request reads = %d, want 1", len(client.workRequests))
	}
	assertRoutingPolicyPathIdentity(t, client.createRequests[0].LoadBalancerId, common.String(routingPolicyNameValue), routingPolicyLoadBalancerID, routingPolicyNameValue)
	if got := stringValue(client.createRequests[0].CreateRoutingPolicyDetails.Name); got != routingPolicyNameValue {
		t.Fatalf("CreateRoutingPolicyDetails.Name = %q, want %q", got, routingPolicyNameValue)
	}
	assertRoutingPolicySDKRules(t, "create rules", client.createRequests[0].CreateRoutingPolicyDetails.Rules, sdkRoutingPolicyRules(routingPolicyBackendSetName))
	assertRoutingPolicyCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create-routing-policy")

	response, err = serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("pending CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("pending CreateOrUpdate() response = %#v, want successful pending response", response)
	}
	if !response.ShouldRequeue {
		t.Fatal("pending CreateOrUpdate() ShouldRequeue = false, want pending work request requeue")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests after pending resume = %d, want 1", len(client.createRequests))
	}
	if len(client.workRequests) != 2 {
		t.Fatalf("work request reads after pending resume = %d, want 2", len(client.workRequests))
	}
	assertRoutingPolicyCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create-routing-policy")

	client.workRequestByID["wr-create-routing-policy"] = routingPolicyWorkRequest("wr-create-routing-policy", loadbalancersdk.WorkRequestLifecycleStateSucceeded, "CreateRoutingPolicy")
	response, err = serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("succeeded CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("succeeded CreateOrUpdate() response = %#v, want successful observe response", response)
	}
	if response.ShouldRequeue {
		t.Fatal("succeeded CreateOrUpdate() ShouldRequeue = true, want active observation")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests after succeeded resume = %d, want 1", len(client.createRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests after succeeded resume = %d, want 0", len(client.updateRequests))
	}
	if len(client.workRequests) != 3 {
		t.Fatalf("work request reads after succeeded resume = %d, want 3", len(client.workRequests))
	}
	assertRoutingPolicyTrackedStatus(t, resource, routingPolicyLoadBalancerID, routingPolicyNameValue, resource.Spec.ConditionLanguageVersion, resource.Spec.Rules)
	assertRoutingPolicyNoCurrentWorkRequest(t, resource)
}

func TestCreateOrUpdateBindsExistingRoutingPolicy(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedRoutingPolicyResource()
	client := &fakeGeneratedRoutingPolicyOCIClient{
		routingPolicies: map[string]loadbalancersdk.RoutingPolicy{
			routingPolicyNameValue: sdkRoutingPolicy(resource.Spec.ConditionLanguageVersion, routingPolicyBackendSetName),
		},
	}

	response, err := newTestRoutingPolicyRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful bind response", response)
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want active bind response")
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0", len(client.createRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(client.updateRequests))
	}
	assertRoutingPolicyTrackedStatus(t, resource, routingPolicyLoadBalancerID, routingPolicyNameValue, resource.Spec.ConditionLanguageVersion, resource.Spec.Rules)
}

func TestCreateOrUpdateUpdatesMutableRoutingPolicyFields(t *testing.T) {
	t.Parallel()

	resource := makeTrackedRoutingPolicyResource()
	resource.Spec.Rules[0].Actions[0].BackendSetName = "updated_backend_set"
	client := &fakeGeneratedRoutingPolicyOCIClient{
		routingPolicies: map[string]loadbalancersdk.RoutingPolicy{
			routingPolicyNameValue: sdkRoutingPolicy(resource.Spec.ConditionLanguageVersion, routingPolicyBackendSetName),
		},
		workRequestByID: map[string]loadbalancersdk.WorkRequest{
			"wr-update-routing-policy": routingPolicyWorkRequest("wr-update-routing-policy", loadbalancersdk.WorkRequestLifecycleStateSucceeded, "UpdateRoutingPolicy"),
		},
	}

	response, err := newTestRoutingPolicyRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful update response", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}
	if len(client.workRequests) != 1 {
		t.Fatalf("work request reads = %d, want 1", len(client.workRequests))
	}
	assertRoutingPolicyPathIdentity(t, client.updateRequests[0].LoadBalancerId, client.updateRequests[0].RoutingPolicyName, routingPolicyLoadBalancerID, routingPolicyNameValue)
	assertRoutingPolicySDKRules(t, "update rules", client.updateRequests[0].UpdateRoutingPolicyDetails.Rules, sdkRoutingPolicyRules("updated_backend_set"))
	assertRoutingPolicyTrackedStatus(t, resource, routingPolicyLoadBalancerID, routingPolicyNameValue, resource.Spec.ConditionLanguageVersion, resource.Spec.Rules)
	assertRoutingPolicyNoCurrentWorkRequest(t, resource)
}

func TestCreateOrUpdateResumesPendingUpdateWorkRequestWithoutDuplicateMutation(t *testing.T) {
	t.Parallel()

	resource := makeTrackedRoutingPolicyResource()
	resource.Spec.Rules[0].Actions[0].BackendSetName = "updated_backend_set"
	client := &fakeGeneratedRoutingPolicyOCIClient{
		routingPolicies: map[string]loadbalancersdk.RoutingPolicy{
			routingPolicyNameValue: sdkRoutingPolicy(resource.Spec.ConditionLanguageVersion, routingPolicyBackendSetName),
		},
		workRequestByID: map[string]loadbalancersdk.WorkRequest{
			"wr-update-routing-policy": routingPolicyWorkRequest("wr-update-routing-policy", loadbalancersdk.WorkRequestLifecycleStateAccepted, "UpdateRoutingPolicy"),
		},
	}
	serviceClient := newTestRoutingPolicyRuntimeClient(client)

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want pending update requeue", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}
	assertRoutingPolicyCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending, "wr-update-routing-policy")

	response, err = serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("pending CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("pending CreateOrUpdate() response = %#v, want pending update requeue", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests after pending resume = %d, want 1", len(client.updateRequests))
	}
	if len(client.workRequests) != 2 {
		t.Fatalf("work request reads after pending resume = %d, want 2", len(client.workRequests))
	}
}

func TestCreateOrUpdateSurfacesFailedRoutingPolicyWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeTrackedRoutingPolicyResource()
	resource.Spec.Rules[0].Actions[0].BackendSetName = "updated_backend_set"
	client := &fakeGeneratedRoutingPolicyOCIClient{
		routingPolicies: map[string]loadbalancersdk.RoutingPolicy{
			routingPolicyNameValue: sdkRoutingPolicy(resource.Spec.ConditionLanguageVersion, routingPolicyBackendSetName),
		},
		workRequestByID: map[string]loadbalancersdk.WorkRequest{
			"wr-update-routing-policy": routingPolicyWorkRequest("wr-update-routing-policy", loadbalancersdk.WorkRequestLifecycleStateFailed, "UpdateRoutingPolicy"),
		},
	}

	response, err := newTestRoutingPolicyRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "FAILED") {
		t.Fatalf("CreateOrUpdate() error = %v, want failed work request error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failed response", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}
	assertRoutingPolicyCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassFailed, "wr-update-routing-policy")
}

func TestCreateOrUpdateRejectsImmutableRoutingPolicyNameDrift(t *testing.T) {
	t.Parallel()

	resource := makeTrackedRoutingPolicyResource()
	resource.Spec.Name = "renamed_routing_policy"
	client := &fakeGeneratedRoutingPolicyOCIClient{
		routingPolicies: map[string]loadbalancersdk.RoutingPolicy{
			routingPolicyNameValue: sdkRoutingPolicy(resource.Spec.ConditionLanguageVersion, routingPolicyBackendSetName),
		},
	}

	response, err := newTestRoutingPolicyRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("CreateOrUpdate() error = %v, want immutable name drift rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failed response", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 for immutable drift", len(client.updateRequests))
	}
}

func TestDeleteRetainsFinalizerUntilRoutingPolicyDeleteConfirmed(t *testing.T) {
	t.Parallel()

	resource := makeTrackedRoutingPolicyResource()
	client := &fakeGeneratedRoutingPolicyOCIClient{
		keepAfterDelete: true,
		routingPolicies: map[string]loadbalancersdk.RoutingPolicy{
			routingPolicyNameValue: sdkRoutingPolicy(resource.Spec.ConditionLanguageVersion, routingPolicyBackendSetName),
		},
		workRequestByID: map[string]loadbalancersdk.WorkRequest{
			"wr-delete-routing-policy": routingPolicyWorkRequest("wr-delete-routing-policy", loadbalancersdk.WorkRequestLifecycleStateInProgress, "DeleteRoutingPolicy"),
		},
	}
	serviceClient := newTestRoutingPolicyRuntimeClient(client)

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while routing policy still exists")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	if len(client.workRequests) != 1 {
		t.Fatalf("work request reads = %d, want 1", len(client.workRequests))
	}
	assertRoutingPolicyPathIdentity(t, client.deleteRequests[0].LoadBalancerId, client.deleteRequests[0].RoutingPolicyName, routingPolicyLoadBalancerID, routingPolicyNameValue)
	assertRoutingPolicyCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "wr-delete-routing-policy")

	deleted, err = serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("pending Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("pending Delete() deleted = true, want finalizer retained while work request is pending")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests after pending resume = %d, want 1", len(client.deleteRequests))
	}
	if len(client.workRequests) != 2 {
		t.Fatalf("work request reads after pending resume = %d, want 2", len(client.workRequests))
	}

	client.workRequestByID["wr-delete-routing-policy"] = routingPolicyWorkRequest("wr-delete-routing-policy", loadbalancersdk.WorkRequestLifecycleStateSucceeded, "DeleteRoutingPolicy")
	deleted, err = serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("succeeded Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("succeeded Delete() deleted = true, want finalizer retained while readback still finds routing policy")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests after succeeded resume = %d, want 1", len(client.deleteRequests))
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status reason = %q, want %q", got, shared.Terminating)
	}

	delete(client.routingPolicies, routingPolicyNameValue)
	deleted, err = serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("confirmed Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("confirmed Delete() deleted = false, want finalizer release after not found confirmation")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests after confirmation = %d, want 1", len(client.deleteRequests))
	}
	assertRoutingPolicyNoCurrentWorkRequest(t, resource)
}

func TestRoutingPolicyCreateBodySupportsJsonDataAction(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedRoutingPolicyResource()
	resource.Spec.Rules[0].Actions = []loadbalancerv1beta1.RoutingPolicyRuleAction{
		{
			JsonData: `{"name":"FORWARD_TO_BACKENDSET","backendSetName":"json_backend_set"}`,
		},
	}

	body, err := buildRoutingPolicyCreateBody(resource)
	if err != nil {
		t.Fatalf("buildRoutingPolicyCreateBody() error = %v", err)
	}
	assertRoutingPolicySDKRules(t, "json action rules", body.Rules, sdkRoutingPolicyRules("json_backend_set"))
}

func makeUntrackedRoutingPolicyResource() *loadbalancerv1beta1.RoutingPolicy {
	return &loadbalancerv1beta1.RoutingPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: routingPolicyNameValue,
			Annotations: map[string]string{
				routingPolicyLoadBalancerIDAnnotation: routingPolicyLoadBalancerID,
			},
		},
		Spec: loadbalancerv1beta1.RoutingPolicySpec{
			Name:                     routingPolicyNameValue,
			ConditionLanguageVersion: "V1",
			Rules:                    apiRoutingPolicyRules(routingPolicyBackendSetName),
		},
	}
}

func makeTrackedRoutingPolicyResource() *loadbalancerv1beta1.RoutingPolicy {
	resource := makeUntrackedRoutingPolicyResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(routingPolicyLoadBalancerID)
	resource.Status.Name = routingPolicyNameValue
	return resource
}

func apiRoutingPolicyRules(backendSetName string) []loadbalancerv1beta1.RoutingPolicyRule {
	return []loadbalancerv1beta1.RoutingPolicyRule{
		{
			Name:      "route-images",
			Condition: "all(http.request.url.path sw '/images')",
			Actions: []loadbalancerv1beta1.RoutingPolicyRuleAction{
				{
					Name:           string(loadbalancersdk.ActionNameForwardToBackendset),
					BackendSetName: backendSetName,
				},
			},
		},
	}
}

func sdkRoutingPolicy(conditionLanguageVersion string, backendSetName string) loadbalancersdk.RoutingPolicy {
	return loadbalancersdk.RoutingPolicy{
		Name:                     common.String(routingPolicyNameValue),
		ConditionLanguageVersion: loadbalancersdk.RoutingPolicyConditionLanguageVersionEnum(conditionLanguageVersion),
		Rules:                    sdkRoutingPolicyRules(backendSetName),
	}
}

func sdkRoutingPolicyRules(backendSetName string) []loadbalancersdk.RoutingRule {
	return []loadbalancersdk.RoutingRule{
		{
			Name:      common.String("route-images"),
			Condition: common.String("all(http.request.url.path sw '/images')"),
			Actions: []loadbalancersdk.Action{
				loadbalancersdk.ForwardToBackendSet{
					BackendSetName: common.String(backendSetName),
				},
			},
		},
	}
}

func routingPolicyFromCreateDetails(details loadbalancersdk.CreateRoutingPolicyDetails) loadbalancersdk.RoutingPolicy {
	return loadbalancersdk.RoutingPolicy{
		Name:                     details.Name,
		ConditionLanguageVersion: loadbalancersdk.RoutingPolicyConditionLanguageVersionEnum(details.ConditionLanguageVersion),
		Rules:                    details.Rules,
	}
}

func routingPolicyFromUpdateDetails(name string, details loadbalancersdk.UpdateRoutingPolicyDetails, existing loadbalancersdk.RoutingPolicy) loadbalancersdk.RoutingPolicy {
	conditionLanguageVersion := loadbalancersdk.RoutingPolicyConditionLanguageVersionEnum(details.ConditionLanguageVersion)
	if conditionLanguageVersion == "" {
		conditionLanguageVersion = existing.ConditionLanguageVersion
	}
	return loadbalancersdk.RoutingPolicy{
		Name:                     common.String(name),
		ConditionLanguageVersion: conditionLanguageVersion,
		Rules:                    details.Rules,
	}
}

func routingPolicyWorkRequest(
	id string,
	state loadbalancersdk.WorkRequestLifecycleStateEnum,
	workType string,
) loadbalancersdk.WorkRequest {
	return loadbalancersdk.WorkRequest{
		Id:             common.String(id),
		LoadBalancerId: common.String(routingPolicyLoadBalancerID),
		Type:           common.String(workType),
		LifecycleState: state,
		Message:        common.String("test work request"),
	}
}

func assertRoutingPolicyPathIdentity(t *testing.T, loadBalancerID *string, routingPolicyName *string, wantLoadBalancerID string, wantRoutingPolicyName string) {
	t.Helper()
	if got := stringValue(loadBalancerID); got != wantLoadBalancerID {
		t.Fatalf("loadBalancerId = %q, want %q", got, wantLoadBalancerID)
	}
	if got := stringValue(routingPolicyName); got != wantRoutingPolicyName {
		t.Fatalf("routingPolicyName = %q, want %q", got, wantRoutingPolicyName)
	}
}

func assertRoutingPolicyTrackedStatus(
	t *testing.T,
	resource *loadbalancerv1beta1.RoutingPolicy,
	wantLoadBalancerID string,
	wantName string,
	wantConditionLanguageVersion string,
	wantRules []loadbalancerv1beta1.RoutingPolicyRule,
) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != wantLoadBalancerID {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantLoadBalancerID)
	}
	if got := resource.Status.Name; got != wantName {
		t.Fatalf("status.name = %q, want %q", got, wantName)
	}
	if got := resource.Status.ConditionLanguageVersion; got != wantConditionLanguageVersion {
		t.Fatalf("status.conditionLanguageVersion = %q, want %q", got, wantConditionLanguageVersion)
	}
	if !reflect.DeepEqual(resource.Status.Rules, wantRules) {
		t.Fatalf("status.rules = %#v, want %#v", resource.Status.Rules, wantRules)
	}
}

func assertRoutingPolicyCurrentWorkRequest(
	t *testing.T,
	resource *loadbalancerv1beta1.RoutingPolicy,
	wantPhase shared.OSOKAsyncPhase,
	wantClass shared.OSOKAsyncNormalizedClass,
	wantWorkRequestID string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want work request tracker")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != wantPhase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, wantPhase)
	}
	if current.NormalizedClass != wantClass {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, wantClass)
	}
	if current.WorkRequestID != wantWorkRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, wantWorkRequestID)
	}
}

func assertRoutingPolicyNoCurrentWorkRequest(t *testing.T, resource *loadbalancerv1beta1.RoutingPolicy) {
	t.Helper()
	if current := resource.Status.OsokStatus.Async.Current; current != nil {
		t.Fatalf("status.async.current = %#v, want nil", current)
	}
}

func assertRoutingPolicySDKRules(t *testing.T, label string, got []loadbalancersdk.RoutingRule, want []loadbalancersdk.RoutingRule) {
	t.Helper()
	gotPayload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal got %s: %v", label, err)
	}
	wantPayload, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal want %s: %v", label, err)
	}
	if string(gotPayload) != string(wantPayload) {
		t.Fatalf("%s = %s, want %s", label, gotPayload, wantPayload)
	}
}

func assertRoutingPolicyStringSliceEqual(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", label, got, want)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
