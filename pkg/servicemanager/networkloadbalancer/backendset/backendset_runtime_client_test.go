/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package backendset

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	networkloadbalancersdk "github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	networkloadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/networkloadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	backendSetNetworkLoadBalancerID = "ocid1.networkloadbalancer.oc1..backendset-parent"
	backendSetNameValue             = "backend_set"
)

type fakeGeneratedBackendSetOCIClient struct {
	backendSets     map[string]networkloadbalancersdk.BackendSet
	listPages       []backendSetListPage
	workRequestByID map[string]networkloadbalancersdk.WorkRequest

	createErr error
	getErr    error
	listErr   error
	updateErr error
	deleteErr error

	createRequests []networkloadbalancersdk.CreateBackendSetRequest
	getRequests    []networkloadbalancersdk.GetBackendSetRequest
	listRequests   []networkloadbalancersdk.ListBackendSetsRequest
	updateRequests []networkloadbalancersdk.UpdateBackendSetRequest
	deleteRequests []networkloadbalancersdk.DeleteBackendSetRequest
	workRequests   []networkloadbalancersdk.GetWorkRequestRequest
}

type backendSetListPage struct {
	pageToken string
	nextPage  string
	items     []networkloadbalancersdk.BackendSetSummary
}

func (f *fakeGeneratedBackendSetOCIClient) CreateBackendSet(_ context.Context, request networkloadbalancersdk.CreateBackendSetRequest) (networkloadbalancersdk.CreateBackendSetResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return networkloadbalancersdk.CreateBackendSetResponse{}, f.createErr
	}
	f.ensureMaps()
	name := stringValue(request.Name)
	f.backendSets[name] = backendSetFromCreateDetails(request.CreateBackendSetDetails)
	return networkloadbalancersdk.CreateBackendSetResponse{
		OpcWorkRequestId: common.String("wr-create-backendset"),
		OpcRequestId:     common.String("opc-create"),
	}, nil
}

func (f *fakeGeneratedBackendSetOCIClient) GetBackendSet(_ context.Context, request networkloadbalancersdk.GetBackendSetRequest) (networkloadbalancersdk.GetBackendSetResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getErr != nil {
		return networkloadbalancersdk.GetBackendSetResponse{}, f.getErr
	}
	f.ensureMaps()
	backendSet, ok := f.backendSets[stringValue(request.BackendSetName)]
	if !ok {
		return networkloadbalancersdk.GetBackendSetResponse{}, backendSetNotFoundError()
	}
	return networkloadbalancersdk.GetBackendSetResponse{
		BackendSet:   backendSet,
		OpcRequestId: common.String("opc-get"),
	}, nil
}

func (f *fakeGeneratedBackendSetOCIClient) ListBackendSets(_ context.Context, request networkloadbalancersdk.ListBackendSetsRequest) (networkloadbalancersdk.ListBackendSetsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listErr != nil {
		return networkloadbalancersdk.ListBackendSetsResponse{}, f.listErr
	}
	f.ensureMaps()
	if len(f.listPages) > 0 {
		pageToken := stringValue(request.Page)
		for _, page := range f.listPages {
			if page.pageToken != pageToken {
				continue
			}
			response := networkloadbalancersdk.ListBackendSetsResponse{
				BackendSetCollection: networkloadbalancersdk.BackendSetCollection{Items: page.items},
				OpcRequestId:         common.String("opc-list"),
			}
			if page.nextPage != "" {
				response.OpcNextPage = common.String(page.nextPage)
			}
			return response, nil
		}
		return networkloadbalancersdk.ListBackendSetsResponse{
			BackendSetCollection: networkloadbalancersdk.BackendSetCollection{},
			OpcRequestId:         common.String("opc-list"),
		}, nil
	}

	items := make([]networkloadbalancersdk.BackendSetSummary, 0, len(f.backendSets))
	for _, backendSet := range f.backendSets {
		items = append(items, backendSetSummaryFromBackendSet(backendSet))
	}
	return networkloadbalancersdk.ListBackendSetsResponse{
		BackendSetCollection: networkloadbalancersdk.BackendSetCollection{Items: items},
		OpcRequestId:         common.String("opc-list"),
	}, nil
}

func (f *fakeGeneratedBackendSetOCIClient) UpdateBackendSet(_ context.Context, request networkloadbalancersdk.UpdateBackendSetRequest) (networkloadbalancersdk.UpdateBackendSetResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return networkloadbalancersdk.UpdateBackendSetResponse{}, f.updateErr
	}
	f.ensureMaps()
	name := stringValue(request.BackendSetName)
	f.backendSets[name] = backendSetFromUpdateDetails(name, request.UpdateBackendSetDetails, f.backendSets[name])
	return networkloadbalancersdk.UpdateBackendSetResponse{
		OpcWorkRequestId: common.String("wr-update-backendset"),
		OpcRequestId:     common.String("opc-update"),
	}, nil
}

func (f *fakeGeneratedBackendSetOCIClient) DeleteBackendSet(_ context.Context, request networkloadbalancersdk.DeleteBackendSetRequest) (networkloadbalancersdk.DeleteBackendSetResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return networkloadbalancersdk.DeleteBackendSetResponse{}, f.deleteErr
	}
	return networkloadbalancersdk.DeleteBackendSetResponse{
		OpcWorkRequestId: common.String("wr-delete-backendset"),
		OpcRequestId:     common.String("opc-delete"),
	}, nil
}

func (f *fakeGeneratedBackendSetOCIClient) GetWorkRequest(_ context.Context, request networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error) {
	f.workRequests = append(f.workRequests, request)
	f.ensureMaps()
	workRequest, ok := f.workRequestByID[stringValue(request.WorkRequestId)]
	if !ok {
		return networkloadbalancersdk.GetWorkRequestResponse{}, backendSetNotFoundError()
	}
	return networkloadbalancersdk.GetWorkRequestResponse{
		WorkRequest:  workRequest,
		OpcRequestId: common.String("opc-work-request"),
	}, nil
}

func (f *fakeGeneratedBackendSetOCIClient) ensureMaps() {
	if f.backendSets == nil {
		f.backendSets = map[string]networkloadbalancersdk.BackendSet{}
	}
	if f.workRequestByID == nil {
		f.workRequestByID = map[string]networkloadbalancersdk.WorkRequest{}
	}
}

func TestBackendSetRuntimeSemanticsUsesGeneratedWorkRequests(t *testing.T) {
	t.Parallel()

	got := newBackendSetRuntimeSemantics()
	if got.FormalService != "networkloadbalancer" || got.FormalSlug != "backendset" {
		t.Fatalf("formal identity = %s/%s, want networkloadbalancer/backendset", got.FormalService, got.FormalSlug)
	}
	if got.Async == nil || got.Async.Strategy != "workrequest" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("async semantics = %#v, want generatedruntime workrequest", got.Async)
	}
	assertBackendSetStringSliceEqual(t, "work request phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertBackendSetStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"name"})
	assertBackendSetStringSliceContains(t, "Mutation.Mutable", got.Mutation.Mutable, "healthChecker")
	assertBackendSetStringSliceContains(t, "Mutation.Mutable", got.Mutation.Mutable, "backends")
	assertBackendSetStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"name"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
}

func TestCreateOrUpdateRejectsMissingBackendSetNetworkLoadBalancerAnnotation(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedBackendSetResource()
	resource.Annotations = nil
	client := &fakeGeneratedBackendSetOCIClient{}

	response, err := newTestBackendSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), backendSetNetworkLoadBalancerIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want missing %s annotation", err, backendSetNetworkLoadBalancerIDAnnotation)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failed response", response)
	}
	if len(client.createRequests) != 0 || len(client.getRequests) != 0 || len(client.listRequests) != 0 {
		t.Fatalf("OCI calls = create:%d get:%d list:%d, want none", len(client.createRequests), len(client.getRequests), len(client.listRequests))
	}
}

//nolint:gocyclo // Exercises create, identity, and pending work request retry in one flow.
func TestCreateOrUpdateCreatesBackendSetAndTracksPendingWorkRequest(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedBackendSetOCIClient{
		workRequestByID: map[string]networkloadbalancersdk.WorkRequest{
			"wr-create-backendset": backendSetWorkRequest("wr-create-backendset", networkloadbalancersdk.OperationStatusInProgress, networkloadbalancersdk.OperationTypeCreateBackendset),
		},
	}
	serviceClient := newTestBackendSetRuntimeClient(client)
	resource := makeUntrackedBackendSetResource()

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
	assertBackendSetPathIdentity(t, client.createRequests[0].NetworkLoadBalancerId, common.String(backendSetNameValue), backendSetNetworkLoadBalancerID, backendSetNameValue)
	if got := client.createRequests[0].Policy; got != networkloadbalancersdk.NetworkLoadBalancingPolicyFiveTuple {
		t.Fatalf("CreateBackendSetDetails.Policy = %q, want FIVE_TUPLE", got)
	}
	requireBackendSetAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create-backendset")
	assertBackendSetTrackedStatus(t, resource, backendSetNetworkLoadBalancerID, backendSetNameValue)

	response, err = serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("pending CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("pending CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests after pending resume = %d, want 1", len(client.createRequests))
	}
	if len(client.workRequests) != 2 {
		t.Fatalf("work request reads after pending resume = %d, want 2", len(client.workRequests))
	}
	requireBackendSetAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create-backendset")
}

func TestCreateOrUpdateBindsExistingBackendSetWithoutMutation(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedBackendSetResource()
	client := &fakeGeneratedBackendSetOCIClient{
		backendSets: map[string]networkloadbalancersdk.BackendSet{
			backendSetNameValue: sdkBackendSet(string(networkloadbalancersdk.NetworkLoadBalancingPolicyFiveTuple), false),
		},
	}

	response, err := newTestBackendSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful bind response", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() ShouldRequeue = true, want no requeue")
	}
	if len(client.createRequests) != 0 || len(client.updateRequests) != 0 {
		t.Fatalf("mutating calls = create:%d update:%d, want none", len(client.createRequests), len(client.updateRequests))
	}
	assertBackendSetTrackedStatus(t, resource, backendSetNetworkLoadBalancerID, backendSetNameValue)
	assertBackendSetProjectedStatus(t, resource, string(networkloadbalancersdk.NetworkLoadBalancingPolicyFiveTuple), false)
	requireBackendSetNoAsyncCurrent(t, resource)
}

func TestCreateOrUpdateBindsExistingBackendSetFromLaterListPage(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedBackendSetResource()
	otherBackendSet := sdkBackendSet(string(networkloadbalancersdk.NetworkLoadBalancingPolicyFiveTuple), false)
	otherBackendSet.Name = common.String("other_backend_set")
	client := &fakeGeneratedBackendSetOCIClient{
		listPages: []backendSetListPage{
			{
				items: []networkloadbalancersdk.BackendSetSummary{
					backendSetSummaryFromBackendSet(otherBackendSet),
				},
				nextPage: "page-2",
			},
			{
				pageToken: "page-2",
				items: []networkloadbalancersdk.BackendSetSummary{
					backendSetSummaryFromBackendSet(sdkBackendSet(string(networkloadbalancersdk.NetworkLoadBalancingPolicyFiveTuple), false)),
				},
			},
		},
	}

	response, err := newTestBackendSetRuntimeClientWithoutGet(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful bind response", response)
	}
	if len(client.createRequests) != 0 || len(client.updateRequests) != 0 {
		t.Fatalf("mutating calls = create:%d update:%d, want none", len(client.createRequests), len(client.updateRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2 pages", len(client.listRequests))
	}
	if got := stringValue(client.listRequests[0].Page); got != "" {
		t.Fatalf("first ListBackendSets Page = %q, want empty", got)
	}
	if got := stringValue(client.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second ListBackendSets Page = %q, want page-2", got)
	}
	assertBackendSetTrackedStatus(t, resource, backendSetNetworkLoadBalancerID, backendSetNameValue)
	assertBackendSetProjectedStatus(t, resource, string(networkloadbalancersdk.NetworkLoadBalancingPolicyFiveTuple), false)
}

func TestCreateOrUpdateSkipsUpdateForDefaultedBackendSetReadback(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendSetResource()
	resource.Spec.IpVersion = ""
	resource.Spec.Backends[0].Weight = 0
	client := &fakeGeneratedBackendSetOCIClient{
		backendSets: map[string]networkloadbalancersdk.BackendSet{
			backendSetNameValue: sdkDefaultedBackendSetReadback(),
		},
	}

	response, err := newTestBackendSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful observe response", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() ShouldRequeue = true, want no requeue")
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want no update for OCI-defaulted readback", len(client.updateRequests))
	}
}

func TestCreateOrUpdateUpdatesMutableBackendSetAndPreservesExplicitFalseBackend(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendSetResource()
	resource.Spec.Policy = string(networkloadbalancersdk.NetworkLoadBalancingPolicyTwoTuple)
	resource.Spec.Backends[0].IsOffline = false
	client := &fakeGeneratedBackendSetOCIClient{
		backendSets: map[string]networkloadbalancersdk.BackendSet{
			backendSetNameValue: sdkBackendSet(string(networkloadbalancersdk.NetworkLoadBalancingPolicyFiveTuple), true),
		},
		workRequestByID: map[string]networkloadbalancersdk.WorkRequest{
			"wr-update-backendset": backendSetWorkRequest("wr-update-backendset", networkloadbalancersdk.OperationStatusSucceeded, networkloadbalancersdk.OperationTypeUpdateBackendset),
		},
	}

	response, err := newTestBackendSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful update response", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() ShouldRequeue = true, want completed update observation")
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}
	updateRequest := client.updateRequests[0]
	assertBackendSetPathIdentity(t, updateRequest.NetworkLoadBalancerId, updateRequest.BackendSetName, backendSetNetworkLoadBalancerID, backendSetNameValue)
	if got := stringValue(updateRequest.Policy); got != string(networkloadbalancersdk.NetworkLoadBalancingPolicyTwoTuple) {
		t.Fatalf("UpdateBackendSetDetails.Policy = %q, want TWO_TUPLE", got)
	}
	if len(updateRequest.Backends) != 1 {
		t.Fatalf("UpdateBackendSetDetails.Backends len = %d, want 1", len(updateRequest.Backends))
	}
	if got := updateRequest.Backends[0].IsOffline; got == nil || *got {
		t.Fatalf("UpdateBackendSetDetails.Backends[0].IsOffline = %#v, want explicit false", got)
	}
	assertBackendSetProjectedStatus(t, resource, string(networkloadbalancersdk.NetworkLoadBalancingPolicyTwoTuple), false)
	requireBackendSetNoAsyncCurrent(t, resource)
}

func TestCreateOrUpdateRejectsBackendSetIdentityDriftBeforeMutation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*networkloadbalancerv1beta1.BackendSet)
		wantErr string
	}{
		{
			name: "network load balancer annotation",
			mutate: func(resource *networkloadbalancerv1beta1.BackendSet) {
				resource.Annotations[backendSetNetworkLoadBalancerIDAnnotation] = "ocid1.networkloadbalancer.oc1..replacement"
			},
			wantErr: backendSetNetworkLoadBalancerIDAnnotation,
		},
		{
			name: "name",
			mutate: func(resource *networkloadbalancerv1beta1.BackendSet) {
				resource.Spec.Name = "replacement_backend_set"
			},
			wantErr: "require replacement when name changes",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource := makeTrackedBackendSetResource()
			tc.mutate(resource)
			client := &fakeGeneratedBackendSetOCIClient{}

			response, err := newTestBackendSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("CreateOrUpdate() error = %v, want containing %q", err, tc.wantErr)
			}
			if response.IsSuccessful {
				t.Fatalf("CreateOrUpdate() response = %#v, want failed response", response)
			}
			if len(client.updateRequests) != 0 || len(client.createRequests) != 0 {
				t.Fatalf("mutating calls = create:%d update:%d, want none", len(client.createRequests), len(client.updateRequests))
			}
		})
	}
}

func TestCreateOrUpdateRecordsBackendSetOCIErrorOpcRequestID(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedBackendSetResource()
	client := &fakeGeneratedBackendSetOCIClient{
		createErr: errortest.NewServiceError(500, "InternalError", "create failed"),
	}

	response, err := newTestBackendSetRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "create failed") {
		t.Fatalf("CreateOrUpdate() error = %v, want create failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want failed response", response)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDeletePollsBackendSetWorkRequestAndConfirmsNotFound(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendSetResource()
	client := &fakeGeneratedBackendSetOCIClient{
		backendSets: map[string]networkloadbalancersdk.BackendSet{
			backendSetNameValue: sdkBackendSet(string(networkloadbalancersdk.NetworkLoadBalancingPolicyFiveTuple), false),
		},
		workRequestByID: map[string]networkloadbalancersdk.WorkRequest{
			"wr-delete-backendset": backendSetWorkRequest("wr-delete-backendset", networkloadbalancersdk.OperationStatusInProgress, networkloadbalancersdk.OperationTypeDeleteBackendset),
		},
	}
	serviceClient := newTestBackendSetRuntimeClient(client)

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want pending delete")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	assertBackendSetPathIdentity(t, client.deleteRequests[0].NetworkLoadBalancerId, client.deleteRequests[0].BackendSetName, backendSetNetworkLoadBalancerID, backendSetNameValue)
	requireBackendSetAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "wr-delete-backendset")

	client.workRequestByID["wr-delete-backendset"] = backendSetWorkRequest("wr-delete-backendset", networkloadbalancersdk.OperationStatusSucceeded, networkloadbalancersdk.OperationTypeDeleteBackendset)
	delete(client.backendSets, backendSetNameValue)
	deleted, err = serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("succeeded Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("succeeded Delete() = false, want confirmed delete")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests after succeeded work request = %d, want 1", len(client.deleteRequests))
	}
	requireBackendSetNoAsyncCurrent(t, resource)
}

func TestDeletePreservesPendingBackendSetCreateWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendSetResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseCreate,
		WorkRequestID:   "wr-create-backendset",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	client := &fakeGeneratedBackendSetOCIClient{
		workRequestByID: map[string]networkloadbalancersdk.WorkRequest{
			"wr-create-backendset": backendSetWorkRequest("wr-create-backendset", networkloadbalancersdk.OperationStatusInProgress, networkloadbalancersdk.OperationTypeCreateBackendset),
		},
	}

	deleted, err := newTestBackendSetRuntimeClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want pending create to keep finalizer")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 while create work request is pending", len(client.deleteRequests))
	}
	requireBackendSetAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create-backendset")
}

func TestDeleteTreatsBackendSetAuthShapedNotFoundAsFatal(t *testing.T) {
	t.Parallel()

	resource := makeTrackedBackendSetResource()
	client := &fakeGeneratedBackendSetOCIClient{
		backendSets: map[string]networkloadbalancersdk.BackendSet{
			backendSetNameValue: sdkBackendSet(string(networkloadbalancersdk.NetworkLoadBalancingPolicyFiveTuple), false),
		},
		deleteErr: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
	}

	deleted, err := newTestBackendSetRuntimeClient(client).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 failure", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want finalizer retained")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDeleteSucceededWorkRequestTreatsGetAuthShapedConfirmationNotFoundAsFatal(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedBackendSetOCIClient{
		getErr: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
	}
	runBackendSetSucceededDeleteWorkRequestAuthShapedConfirmationTest(t, client, newTestBackendSetRuntimeClient, 1, 0)
}

func TestDeleteSucceededWorkRequestTreatsListAuthShapedConfirmationNotFoundAsFatal(t *testing.T) {
	t.Parallel()

	client := &fakeGeneratedBackendSetOCIClient{
		listErr: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
	}
	runBackendSetSucceededDeleteWorkRequestAuthShapedConfirmationTest(t, client, newTestBackendSetRuntimeClientWithoutGetHook, 0, 1)
}

func runBackendSetSucceededDeleteWorkRequestAuthShapedConfirmationTest(
	t *testing.T,
	client *fakeGeneratedBackendSetOCIClient,
	serviceClient func(backendSetRuntimeOCIClient) BackendSetServiceClient,
	wantGetRequests int,
	wantListRequests int,
) {
	t.Helper()

	resource := makeTrackedBackendSetResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete-backendset",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}
	client.workRequestByID = map[string]networkloadbalancersdk.WorkRequest{
		"wr-delete-backendset": backendSetWorkRequest("wr-delete-backendset", networkloadbalancersdk.OperationStatusSucceeded, networkloadbalancersdk.OperationTypeDeleteBackendset),
	}

	deleted, err := serviceClient(client).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirmation 404 failure", err)
	}
	if deleted {
		t.Fatal("Delete() = true, want finalizer retained")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 after succeeded work request confirmation fails", len(client.deleteRequests))
	}
	if got := len(client.getRequests); got != wantGetRequests {
		t.Fatalf("get requests = %d, want %d", got, wantGetRequests)
	}
	if got := len(client.listRequests); got != wantListRequests {
		t.Fatalf("list requests = %d, want %d", got, wantListRequests)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireBackendSetAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "wr-delete-backendset")
}

func newTestBackendSetRuntimeClient(client backendSetRuntimeOCIClient) BackendSetServiceClient {
	return newTestBackendSetRuntimeClientWithConfig(client, nil)
}

func newTestBackendSetRuntimeClientWithoutGetHook(client backendSetRuntimeOCIClient) BackendSetServiceClient {
	return newTestBackendSetRuntimeClientWithHooks(client, func(hooks *BackendSetRuntimeHooks) {
		hooks.Get.Call = nil
	})
}

func newTestBackendSetRuntimeClientWithoutGet(client backendSetRuntimeOCIClient) BackendSetServiceClient {
	return newTestBackendSetRuntimeClientWithConfig(client, func(config *generatedruntime.Config[*networkloadbalancerv1beta1.BackendSet]) {
		config.Get = nil
	})
}

func newTestBackendSetRuntimeClientWithHooks(
	client backendSetRuntimeOCIClient,
	configureHooks func(*BackendSetRuntimeHooks),
) BackendSetServiceClient {
	return newTestBackendSetRuntimeClientWithHooksAndConfig(client, configureHooks, nil)
}

func newTestBackendSetRuntimeClientWithConfig(
	client backendSetRuntimeOCIClient,
	configure func(*generatedruntime.Config[*networkloadbalancerv1beta1.BackendSet]),
) BackendSetServiceClient {
	return newTestBackendSetRuntimeClientWithHooksAndConfig(client, nil, configure)
}

func newTestBackendSetRuntimeClientWithHooksAndConfig(
	client backendSetRuntimeOCIClient,
	configureHooks func(*BackendSetRuntimeHooks),
	configureConfig func(*generatedruntime.Config[*networkloadbalancerv1beta1.BackendSet]),
) BackendSetServiceClient {
	hooks := newBackendSetRuntimeHooksWithOCIClient(client)
	if configureHooks != nil {
		configureHooks(&hooks)
	}
	applyBackendSetRuntimeHooks(&hooks, client, nil)
	config := buildBackendSetGeneratedRuntimeConfig(&BackendSetServiceManager{}, hooks)
	if configureConfig != nil {
		configureConfig(&config)
	}
	delegate := defaultBackendSetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkloadbalancerv1beta1.BackendSet](config),
	}
	return wrapBackendSetGeneratedClient(hooks, delegate)
}

func makeUntrackedBackendSetResource() *networkloadbalancerv1beta1.BackendSet {
	return &networkloadbalancerv1beta1.BackendSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backendSetNameValue,
			Namespace: "default",
			Annotations: map[string]string{
				backendSetNetworkLoadBalancerIDAnnotation: backendSetNetworkLoadBalancerID,
			},
		},
		Spec: networkloadbalancerv1beta1.BackendSetSpec{
			Name:      backendSetNameValue,
			Policy:    string(networkloadbalancersdk.NetworkLoadBalancingPolicyFiveTuple),
			IpVersion: string(networkloadbalancersdk.IpVersionIpv4),
			HealthChecker: networkloadbalancerv1beta1.BackendSetHealthChecker{
				Protocol:   string(networkloadbalancersdk.HealthCheckProtocolsHttp),
				UrlPath:    "/healthz",
				Port:       8080,
				ReturnCode: 200,
			},
			Backends: []networkloadbalancerv1beta1.BackendSetBackend{
				{
					IpAddress: "10.0.0.3",
					Port:      8080,
					Weight:    1,
				},
			},
		},
	}
}

func makeTrackedBackendSetResource() *networkloadbalancerv1beta1.BackendSet {
	resource := makeUntrackedBackendSetResource()
	resource.Status = networkloadbalancerv1beta1.BackendSetStatus{
		Name:      backendSetNameValue,
		Policy:    string(networkloadbalancersdk.NetworkLoadBalancingPolicyFiveTuple),
		IpVersion: string(networkloadbalancersdk.IpVersionIpv4),
		HealthChecker: networkloadbalancerv1beta1.BackendSetHealthChecker{
			Protocol:   string(networkloadbalancersdk.HealthCheckProtocolsHttp),
			UrlPath:    "/healthz",
			Port:       8080,
			ReturnCode: 200,
		},
		Backends: []networkloadbalancerv1beta1.BackendSetBackend{
			{
				Name:      "10.0.0.3:8080",
				IpAddress: "10.0.0.3",
				Port:      8080,
				Weight:    1,
			},
		},
		OsokStatus: shared.OSOKStatus{
			Ocid: shared.OCID(backendSetNetworkLoadBalancerID),
		},
	}
	return resource
}

func backendSetNotFoundError() error {
	return errortest.NewServiceError(404, errorutil.NotFound, "backend set not found")
}

func backendSetWorkRequest(
	id string,
	status networkloadbalancersdk.OperationStatusEnum,
	operationType networkloadbalancersdk.OperationTypeEnum,
) networkloadbalancersdk.WorkRequest {
	return networkloadbalancersdk.WorkRequest{
		Id:            common.String(id),
		Status:        status,
		OperationType: operationType,
	}
}

func sdkBackendSet(policy string, isOffline bool) networkloadbalancersdk.BackendSet {
	return networkloadbalancersdk.BackendSet{
		Name:      common.String(backendSetNameValue),
		Policy:    networkloadbalancersdk.NetworkLoadBalancingPolicyEnum(policy),
		IpVersion: networkloadbalancersdk.IpVersionIpv4,
		HealthChecker: &networkloadbalancersdk.HealthChecker{
			Protocol:   networkloadbalancersdk.HealthCheckProtocolsHttp,
			UrlPath:    common.String("/healthz"),
			Port:       common.Int(8080),
			ReturnCode: common.Int(200),
		},
		Backends: []networkloadbalancersdk.Backend{
			{
				Name:      common.String("10.0.0.3:8080"),
				IpAddress: common.String("10.0.0.3"),
				Port:      common.Int(8080),
				Weight:    common.Int(1),
				IsOffline: common.Bool(isOffline),
			},
		},
	}
}

func sdkDefaultedBackendSetReadback() networkloadbalancersdk.BackendSet {
	backendSet := sdkBackendSet(string(networkloadbalancersdk.NetworkLoadBalancingPolicyFiveTuple), false)
	backendSet.HealthChecker.Retries = common.Int(3)
	backendSet.HealthChecker.TimeoutInMillis = common.Int(3000)
	backendSet.HealthChecker.IntervalInMillis = common.Int(10000)
	backendSet.IsPreserveSource = common.Bool(true)
	backendSet.IsFailOpen = common.Bool(false)
	backendSet.IsInstantFailoverEnabled = common.Bool(false)
	backendSet.IsInstantFailoverTcpResetEnabled = common.Bool(true)
	backendSet.AreOperationallyActiveBackendsPreferred = common.Bool(false)
	return backendSet
}

func backendSetFromCreateDetails(details networkloadbalancersdk.CreateBackendSetDetails) networkloadbalancersdk.BackendSet {
	var backendSet networkloadbalancersdk.BackendSet
	roundTripBackendSetJSON(details, &backendSet)
	return backendSet
}

//nolint:gocyclo // Test fake applies only the update fields needed by BackendSet regressions.
func backendSetFromUpdateDetails(
	name string,
	details networkloadbalancersdk.UpdateBackendSetDetails,
	existing networkloadbalancersdk.BackendSet,
) networkloadbalancersdk.BackendSet {
	updated := existing
	if updated.Name == nil {
		updated.Name = common.String(name)
	}
	if details.Policy != nil {
		updated.Policy = networkloadbalancersdk.NetworkLoadBalancingPolicyEnum(*details.Policy)
	}
	if details.HealthChecker != nil {
		var healthChecker networkloadbalancersdk.HealthChecker
		roundTripBackendSetJSON(details.HealthChecker, &healthChecker)
		updated.HealthChecker = &healthChecker
	}
	if details.IpVersion != "" {
		updated.IpVersion = details.IpVersion
	}
	if details.IsPreserveSource != nil {
		updated.IsPreserveSource = details.IsPreserveSource
	}
	if details.IsFailOpen != nil {
		updated.IsFailOpen = details.IsFailOpen
	}
	if details.IsInstantFailoverEnabled != nil {
		updated.IsInstantFailoverEnabled = details.IsInstantFailoverEnabled
	}
	if details.IsInstantFailoverTcpResetEnabled != nil {
		updated.IsInstantFailoverTcpResetEnabled = details.IsInstantFailoverTcpResetEnabled
	}
	if details.AreOperationallyActiveBackendsPreferred != nil {
		updated.AreOperationallyActiveBackendsPreferred = details.AreOperationallyActiveBackendsPreferred
	}
	if details.Backends != nil {
		var backends []networkloadbalancersdk.Backend
		roundTripBackendSetJSON(details.Backends, &backends)
		for i := range backends {
			if backends[i].Name == nil {
				backends[i].Name = common.String(stringValue(backends[i].IpAddress) + ":8080")
			}
		}
		updated.Backends = backends
	}
	return updated
}

func backendSetSummaryFromBackendSet(backendSet networkloadbalancersdk.BackendSet) networkloadbalancersdk.BackendSetSummary {
	return networkloadbalancersdk.BackendSetSummary{
		Name:                                    backendSet.Name,
		Policy:                                  backendSet.Policy,
		Backends:                                backendSet.Backends,
		HealthChecker:                           backendSet.HealthChecker,
		IsPreserveSource:                        backendSet.IsPreserveSource,
		IsFailOpen:                              backendSet.IsFailOpen,
		IsInstantFailoverEnabled:                backendSet.IsInstantFailoverEnabled,
		IsInstantFailoverTcpResetEnabled:        backendSet.IsInstantFailoverTcpResetEnabled,
		AreOperationallyActiveBackendsPreferred: backendSet.AreOperationallyActiveBackendsPreferred,
		IpVersion:                               backendSet.IpVersion,
	}
}

func roundTripBackendSetJSON(source any, target any) {
	payload, err := json.Marshal(source)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		panic(err)
	}
}

func assertBackendSetPathIdentity(t *testing.T, networkLoadBalancerID, backendSetName *string, wantNetworkLoadBalancerID, wantBackendSetName string) {
	t.Helper()
	if got := stringValue(networkLoadBalancerID); got != wantNetworkLoadBalancerID {
		t.Fatalf("NetworkLoadBalancerId = %q, want %q", got, wantNetworkLoadBalancerID)
	}
	if got := stringValue(backendSetName); got != wantBackendSetName {
		t.Fatalf("BackendSetName = %q, want %q", got, wantBackendSetName)
	}
}

func assertBackendSetTrackedStatus(t *testing.T, resource *networkloadbalancerv1beta1.BackendSet, wantNetworkLoadBalancerID, wantBackendSetName string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != wantNetworkLoadBalancerID {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantNetworkLoadBalancerID)
	}
	if got := resource.Status.Name; got != wantBackendSetName {
		t.Fatalf("status.name = %q, want %q", got, wantBackendSetName)
	}
}

func assertBackendSetProjectedStatus(t *testing.T, resource *networkloadbalancerv1beta1.BackendSet, wantPolicy string, wantOffline bool) {
	t.Helper()
	if got := resource.Status.Policy; got != wantPolicy {
		t.Fatalf("status.policy = %q, want %q", got, wantPolicy)
	}
	if len(resource.Status.Backends) != 1 {
		t.Fatalf("status.backends len = %d, want 1", len(resource.Status.Backends))
	}
	if got := resource.Status.Backends[0].IsOffline; got != wantOffline {
		t.Fatalf("status.backends[0].isOffline = %v, want %v", got, wantOffline)
	}
}

func requireBackendSetAsyncCurrent(
	t *testing.T,
	resource *networkloadbalancerv1beta1.BackendSet,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	workRequestID string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatalf("status.status.async.current = nil, want %s %s", phase, class)
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest ||
		current.Phase != phase ||
		current.NormalizedClass != class ||
		current.WorkRequestID != workRequestID {
		t.Fatalf("status.status.async.current = %#v, want source workrequest phase %s class %s work request %s", current, phase, class, workRequestID)
	}
}

func requireBackendSetNoAsyncCurrent(t *testing.T, resource *networkloadbalancerv1beta1.BackendSet) {
	t.Helper()
	if current := resource.Status.OsokStatus.Async.Current; current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil", current)
	}
}

func assertBackendSetStringSliceEqual(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s len = %d, want %d (%v)", name, len(got), len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q", name, i, got[i], want[i])
		}
	}
}

func assertBackendSetStringSliceContains(t *testing.T, name string, got []string, want string) {
	t.Helper()
	for _, candidate := range got {
		if candidate == want {
			return
		}
	}
	t.Fatalf("%s = %v, want containing %q", name, got, want)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

var _ backendSetRuntimeOCIClient = (*fakeGeneratedBackendSetOCIClient)(nil)
var _ BackendSetServiceClient = backendSetPendingWorkRequestDeleteClient{}
var _ servicemanager.OSOKServiceManager = (*BackendSetServiceManager)(nil)
