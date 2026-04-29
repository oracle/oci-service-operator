/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networkloadbalancer

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	networkloadbalancersdk "github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	networkloadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/networkloadbalancer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	networkLoadBalancerCompartmentID  = "ocid1.compartment.oc1..exampleuniqueID"
	networkLoadBalancerCreatedID      = "ocid1.networkloadbalancer.oc1..createduniqueID"
	networkLoadBalancerExistingID     = "ocid1.networkloadbalancer.oc1..existinguniqueID"
	networkLoadBalancerDisplayName    = "example_network_load_balancer"
	networkLoadBalancerSubnetID       = "ocid1.subnet.oc1..exampleuniqueID"
	networkLoadBalancerAssignedIPv4   = "10.0.0.10"
	networkLoadBalancerAssignedIPv6   = "2607:9b80:9a0a:9a7e::10"
	networkLoadBalancerSubnetIPv6CIDR = "2607:9b80:9a0a:9a7e::/64"
	networkLoadBalancerReservedIPv4ID = "ocid1.privateip.oc1..reserveduniqueID"
	networkLoadBalancerReservedIPv6ID = "ocid1.ipv6.oc1..reserveduniqueID"
	networkLoadBalancerNSGID          = "ocid1.networksecuritygroup.oc1..nsguniqueID"
	networkLoadBalancerUpdatedNSGID   = "ocid1.networksecuritygroup.oc1..updatednsguniqueID"
)

type fakeNetworkLoadBalancerOCIClient struct {
	createRequests []networkloadbalancersdk.CreateNetworkLoadBalancerRequest
	getRequests    []networkloadbalancersdk.GetNetworkLoadBalancerRequest
	listRequests   []networkloadbalancersdk.ListNetworkLoadBalancersRequest
	updateRequests []networkloadbalancersdk.UpdateNetworkLoadBalancerRequest
	nsgRequests    []networkloadbalancersdk.UpdateNetworkSecurityGroupsRequest
	deleteRequests []networkloadbalancersdk.DeleteNetworkLoadBalancerRequest
	workRequests   []networkloadbalancersdk.GetWorkRequestRequest

	createErr error
	getErr    error
	listErr   error
	updateErr error
	nsgErr    error
	deleteErr error
	workErr   error

	createLifecycleState  networkloadbalancersdk.LifecycleStateEnum
	listPages             []networkloadbalancersdk.ListNetworkLoadBalancersResponse
	networkLoadBalancers  map[string]networkloadbalancersdk.NetworkLoadBalancer
	workRequestByID       map[string]networkloadbalancersdk.WorkRequest
	deleteRemovesResource bool
}

func (f *fakeNetworkLoadBalancerOCIClient) CreateNetworkLoadBalancer(_ context.Context, request networkloadbalancersdk.CreateNetworkLoadBalancerRequest) (networkloadbalancersdk.CreateNetworkLoadBalancerResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return networkloadbalancersdk.CreateNetworkLoadBalancerResponse{}, f.createErr
	}
	if f.networkLoadBalancers == nil {
		f.networkLoadBalancers = map[string]networkloadbalancersdk.NetworkLoadBalancer{}
	}
	lifecycleState := f.createLifecycleState
	if lifecycleState == "" {
		lifecycleState = networkloadbalancersdk.LifecycleStateActive
	}
	resource := networkLoadBalancerFromCreateDetails(networkLoadBalancerCreatedID, request.CreateNetworkLoadBalancerDetails)
	resource.LifecycleState = lifecycleState
	f.networkLoadBalancers[networkLoadBalancerCreatedID] = resource
	return networkloadbalancersdk.CreateNetworkLoadBalancerResponse{
		NetworkLoadBalancer: resource,
		OpcRequestId:        common.String("opc-create-1"),
		OpcWorkRequestId:    common.String("wr-create-1"),
	}, nil
}

func (f *fakeNetworkLoadBalancerOCIClient) GetNetworkLoadBalancer(_ context.Context, request networkloadbalancersdk.GetNetworkLoadBalancerRequest) (networkloadbalancersdk.GetNetworkLoadBalancerResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getErr != nil {
		return networkloadbalancersdk.GetNetworkLoadBalancerResponse{}, f.getErr
	}

	resource, ok := f.networkLoadBalancers[stringValue(request.NetworkLoadBalancerId)]
	if !ok {
		return networkloadbalancersdk.GetNetworkLoadBalancerResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing network load balancer")
	}
	return networkloadbalancersdk.GetNetworkLoadBalancerResponse{
		NetworkLoadBalancer: resource,
		OpcRequestId:        common.String("opc-get-1"),
	}, nil
}

func (f *fakeNetworkLoadBalancerOCIClient) ListNetworkLoadBalancers(_ context.Context, request networkloadbalancersdk.ListNetworkLoadBalancersRequest) (networkloadbalancersdk.ListNetworkLoadBalancersResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listErr != nil {
		return networkloadbalancersdk.ListNetworkLoadBalancersResponse{}, f.listErr
	}
	if len(f.listPages) > 0 {
		pageIndex := 0
		if request.Page != nil {
			pageIndex = 1
		}
		if pageIndex >= len(f.listPages) {
			return networkloadbalancersdk.ListNetworkLoadBalancersResponse{}, nil
		}
		return f.listPages[pageIndex], nil
	}

	var items []networkloadbalancersdk.NetworkLoadBalancerSummary
	for _, resource := range f.networkLoadBalancers {
		if request.CompartmentId != nil && stringValue(resource.CompartmentId) != stringValue(request.CompartmentId) {
			continue
		}
		if request.DisplayName != nil && stringValue(resource.DisplayName) != stringValue(request.DisplayName) {
			continue
		}
		items = append(items, networkLoadBalancerSummaryFromResource(resource))
	}
	return networkloadbalancersdk.ListNetworkLoadBalancersResponse{
		NetworkLoadBalancerCollection: networkloadbalancersdk.NetworkLoadBalancerCollection{Items: items},
		OpcRequestId:                  common.String("opc-list-1"),
	}, nil
}

func (f *fakeNetworkLoadBalancerOCIClient) UpdateNetworkLoadBalancer(_ context.Context, request networkloadbalancersdk.UpdateNetworkLoadBalancerRequest) (networkloadbalancersdk.UpdateNetworkLoadBalancerResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return networkloadbalancersdk.UpdateNetworkLoadBalancerResponse{}, f.updateErr
	}

	resource, ok := f.networkLoadBalancers[stringValue(request.NetworkLoadBalancerId)]
	if !ok {
		return networkloadbalancersdk.UpdateNetworkLoadBalancerResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing network load balancer")
	}
	applyFakeNetworkLoadBalancerScalarUpdate(&resource, request)
	applyFakeNetworkLoadBalancerIPv6Update(&resource, request)
	applyFakeNetworkLoadBalancerTagUpdate(&resource, request)
	resource.LifecycleState = networkloadbalancersdk.LifecycleStateActive
	f.networkLoadBalancers[stringValue(request.NetworkLoadBalancerId)] = resource
	return networkloadbalancersdk.UpdateNetworkLoadBalancerResponse{
		OpcRequestId:     common.String("opc-update-1"),
		OpcWorkRequestId: common.String("wr-update-1"),
	}, nil
}

func (f *fakeNetworkLoadBalancerOCIClient) UpdateNetworkSecurityGroups(_ context.Context, request networkloadbalancersdk.UpdateNetworkSecurityGroupsRequest) (networkloadbalancersdk.UpdateNetworkSecurityGroupsResponse, error) {
	f.nsgRequests = append(f.nsgRequests, request)
	if f.nsgErr != nil {
		return networkloadbalancersdk.UpdateNetworkSecurityGroupsResponse{}, f.nsgErr
	}

	resource, ok := f.networkLoadBalancers[stringValue(request.NetworkLoadBalancerId)]
	if !ok {
		return networkloadbalancersdk.UpdateNetworkSecurityGroupsResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing network load balancer")
	}
	resource.NetworkSecurityGroupIds = append([]string(nil), request.NetworkSecurityGroupIds...)
	resource.LifecycleState = networkloadbalancersdk.LifecycleStateActive
	f.networkLoadBalancers[stringValue(request.NetworkLoadBalancerId)] = resource
	return networkloadbalancersdk.UpdateNetworkSecurityGroupsResponse{
		OpcRequestId:     common.String("opc-update-nsg-1"),
		OpcWorkRequestId: common.String("wr-update-nsg-1"),
	}, nil
}

func applyFakeNetworkLoadBalancerScalarUpdate(
	resource *networkloadbalancersdk.NetworkLoadBalancer,
	request networkloadbalancersdk.UpdateNetworkLoadBalancerRequest,
) {
	if request.DisplayName != nil {
		resource.DisplayName = request.DisplayName
	}
	if request.IsPreserveSourceDestination != nil {
		resource.IsPreserveSourceDestination = request.IsPreserveSourceDestination
	}
	if request.IsSymmetricHashEnabled != nil {
		resource.IsSymmetricHashEnabled = request.IsSymmetricHashEnabled
	}
	if request.NlbIpVersion != "" {
		resource.NlbIpVersion = request.NlbIpVersion
	}
}

func applyFakeNetworkLoadBalancerIPv6Update(
	resource *networkloadbalancersdk.NetworkLoadBalancer,
	request networkloadbalancersdk.UpdateNetworkLoadBalancerRequest,
) {
	if request.SubnetIpv6Cidr != nil && request.AssignedIpv6 == nil {
		resource.IpAddresses = upsertIPv6Address(resource.IpAddresses, networkLoadBalancerAssignedIPv6, request.ReservedIpv6Id)
	}
	if request.AssignedIpv6 != nil {
		resource.IpAddresses = upsertIPv6Address(resource.IpAddresses, *request.AssignedIpv6, request.ReservedIpv6Id)
	}
	if request.ReservedIpv6Id != nil && request.AssignedIpv6 == nil {
		resource.IpAddresses = upsertIPv6Address(resource.IpAddresses, networkLoadBalancerAssignedIPv6, request.ReservedIpv6Id)
	}
}

func applyFakeNetworkLoadBalancerTagUpdate(
	resource *networkloadbalancersdk.NetworkLoadBalancer,
	request networkloadbalancersdk.UpdateNetworkLoadBalancerRequest,
) {
	if request.FreeformTags != nil {
		resource.FreeformTags = copyStringMap(request.FreeformTags)
	}
	if request.DefinedTags != nil {
		resource.DefinedTags = copyNestedAnyMap(request.DefinedTags)
	}
	if request.SecurityAttributes != nil {
		resource.SecurityAttributes = copyNestedAnyMap(request.SecurityAttributes)
	}
}

func (f *fakeNetworkLoadBalancerOCIClient) DeleteNetworkLoadBalancer(_ context.Context, request networkloadbalancersdk.DeleteNetworkLoadBalancerRequest) (networkloadbalancersdk.DeleteNetworkLoadBalancerResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return networkloadbalancersdk.DeleteNetworkLoadBalancerResponse{}, f.deleteErr
	}

	resource, ok := f.networkLoadBalancers[stringValue(request.NetworkLoadBalancerId)]
	if ok {
		if f.deleteRemovesResource {
			delete(f.networkLoadBalancers, stringValue(request.NetworkLoadBalancerId))
		} else {
			resource.LifecycleState = networkloadbalancersdk.LifecycleStateDeleting
			f.networkLoadBalancers[stringValue(request.NetworkLoadBalancerId)] = resource
		}
	}
	return networkloadbalancersdk.DeleteNetworkLoadBalancerResponse{
		OpcRequestId:     common.String("opc-delete-1"),
		OpcWorkRequestId: common.String("wr-delete-1"),
	}, nil
}

func (f *fakeNetworkLoadBalancerOCIClient) GetWorkRequest(_ context.Context, request networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error) {
	f.workRequests = append(f.workRequests, request)
	if f.workErr != nil {
		return networkloadbalancersdk.GetWorkRequestResponse{}, f.workErr
	}
	workRequestID := stringValue(request.WorkRequestId)
	if workRequest, ok := f.workRequestByID[workRequestID]; ok {
		return networkloadbalancersdk.GetWorkRequestResponse{
			WorkRequest:  workRequest,
			OpcRequestId: common.String("opc-work-1"),
		}, nil
	}
	return networkloadbalancersdk.GetWorkRequestResponse{
		WorkRequest:  defaultNetworkLoadBalancerWorkRequest(workRequestID),
		OpcRequestId: common.String("opc-work-1"),
	}, nil
}

func TestNetworkLoadBalancerRuntimeSemanticsEncodesReviewedLifecycle(t *testing.T) {
	t.Parallel()

	got := requireNetworkLoadBalancerRuntimeSemantics(t, newReviewedNetworkLoadBalancerRuntimeSemantics())
	async := requireNetworkLoadBalancerAsyncSemantics(t, got)
	workRequest := requireNetworkLoadBalancerWorkRequestSemantics(t, async)

	assertNetworkLoadBalancerStringEqual(t, "FormalService", got.FormalService, "networkloadbalancer")
	assertNetworkLoadBalancerStringEqual(t, "FormalSlug", got.FormalSlug, "networkloadbalancer")
	assertNetworkLoadBalancerStringEqual(t, "Async.Strategy", async.Strategy, "workrequest")
	assertNetworkLoadBalancerStringEqual(t, "Async.Runtime", async.Runtime, "generatedruntime")
	assertNetworkLoadBalancerStringEqual(t, "Async.WorkRequest.Source", workRequest.Source, "service-sdk")
	assertNetworkLoadBalancerStringEqual(t, "FinalizerPolicy", got.FinalizerPolicy, "retain-until-confirmed-delete")
	assertNetworkLoadBalancerStringEqual(t, "CreateFollowUp.Strategy", got.CreateFollowUp.Strategy, "GetWorkRequest -> read-after-write")
	assertNetworkLoadBalancerStringEqual(t, "UpdateFollowUp.Strategy", got.UpdateFollowUp.Strategy, "GetWorkRequest -> read-after-write")
	assertNetworkLoadBalancerStringEqual(t, "DeleteFollowUp.Strategy", got.DeleteFollowUp.Strategy, "GetWorkRequest -> confirm-delete")
	assertNetworkLoadBalancerStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertNetworkLoadBalancerStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertNetworkLoadBalancerStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertNetworkLoadBalancerStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertNetworkLoadBalancerStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertNetworkLoadBalancerStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertNetworkLoadBalancerStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"displayName", "compartmentId"})
	assertNetworkLoadBalancerStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{
		"assignedIpv6",
		"definedTags",
		"displayName",
		"freeformTags",
		"isPreserveSourceDestination",
		"isSymmetricHashEnabled",
		"networkSecurityGroupIds",
		"nlbIpVersion",
		"reservedIpv6Id",
		"securityAttributes",
		"subnetIpv6Cidr",
	})
	assertNetworkLoadBalancerStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{
		"assignedPrivateIpv4",
		"backendSets",
		"compartmentId",
		"isPrivate",
		"listeners",
		"reservedIps",
		"subnetId",
	})
}

func TestNetworkLoadBalancerRequestFieldsKeepTrackedOperationsScopedToRecordedIdentity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  []generatedruntimeRequestField
		want []generatedruntimeRequestField
	}{
		{
			name: "create",
			got:  requestFieldSnapshot(networkLoadBalancerCreateFields()),
			want: []generatedruntimeRequestField{
				{FieldName: "CreateNetworkLoadBalancerDetails", RequestName: "CreateNetworkLoadBalancerDetails", Contribution: "body"},
			},
		},
		{
			name: "get",
			got:  requestFieldSnapshot(networkLoadBalancerGetFields()),
			want: []generatedruntimeRequestField{
				{FieldName: "NetworkLoadBalancerId", RequestName: "networkLoadBalancerId", Contribution: "path", PreferResourceID: true, LookupPaths: "status.id,status.status.ocid"},
			},
		},
		{
			name: "list",
			got:  requestFieldSnapshot(networkLoadBalancerListFields()),
			want: []generatedruntimeRequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: "status.compartmentId,spec.compartmentId"},
				{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: "status.displayName,spec.displayName"},
			},
		},
		{
			name: "update",
			got:  requestFieldSnapshot(networkLoadBalancerUpdateFields()),
			want: []generatedruntimeRequestField{
				{FieldName: "UpdateNetworkLoadBalancerDetails", RequestName: "UpdateNetworkLoadBalancerDetails", Contribution: "body"},
				{FieldName: "NetworkLoadBalancerId", RequestName: "networkLoadBalancerId", Contribution: "path", PreferResourceID: true, LookupPaths: "status.id,status.status.ocid"},
			},
		},
		{
			name: "delete",
			got:  requestFieldSnapshot(networkLoadBalancerDeleteFields()),
			want: []generatedruntimeRequestField{
				{FieldName: "NetworkLoadBalancerId", RequestName: "networkLoadBalancerId", Contribution: "path", PreferResourceID: true, LookupPaths: "status.id,status.status.ocid"},
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

func TestNetworkLoadBalancerWorkRequestAdapterMapsServiceStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status string
		want   shared.OSOKAsyncNormalizedClass
	}{
		{status: string(networkloadbalancersdk.OperationStatusAccepted), want: shared.OSOKAsyncClassPending},
		{status: string(networkloadbalancersdk.OperationStatusInProgress), want: shared.OSOKAsyncClassPending},
		{status: string(networkloadbalancersdk.OperationStatusCanceling), want: shared.OSOKAsyncClassPending},
		{status: string(networkloadbalancersdk.OperationStatusSucceeded), want: shared.OSOKAsyncClassSucceeded},
		{status: string(networkloadbalancersdk.OperationStatusFailed), want: shared.OSOKAsyncClassFailed},
		{status: string(networkloadbalancersdk.OperationStatusCanceled), want: shared.OSOKAsyncClassCanceled},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.status, func(t *testing.T) {
			t.Parallel()
			got, err := networkLoadBalancerWorkRequestAsyncAdapter.Normalize(tc.status)
			if err != nil {
				t.Fatalf("Normalize(%q) error = %v", tc.status, err)
			}
			if got != tc.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestCreateOrUpdateBindsExistingNetworkLoadBalancer(t *testing.T) {
	t.Parallel()

	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: networkLoadBalancerFromResource(networkLoadBalancerExistingID, baseNetworkLoadBalancerResource()),
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := baseNetworkLoadBalancerResource()
	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.listRequests) == 0 {
		t.Fatal("list requests = 0, want bind lookup before create")
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for bind path", len(client.createRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 for no-drift bind path", len(client.updateRequests))
	}
	if got := resource.Status.Id; got != networkLoadBalancerExistingID {
		t.Fatalf("status.id = %q, want %q", got, networkLoadBalancerExistingID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != networkLoadBalancerExistingID {
		t.Fatalf("status.status.ocid = %q, want %q", got, networkLoadBalancerExistingID)
	}
}

func TestCreateOrUpdateCreatesMissingNetworkLoadBalancer(t *testing.T) {
	t.Parallel()

	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := baseNetworkLoadBalancerResource()
	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(client.createRequests))
	}
	if len(client.workRequests) != 1 {
		t.Fatalf("work request reads = %d, want 1", len(client.workRequests))
	}
	if got := stringValue(client.createRequests[0].CompartmentId); got != networkLoadBalancerCompartmentID {
		t.Fatalf("create request compartmentId = %q, want %q", got, networkLoadBalancerCompartmentID)
	}
	if got := stringValue(client.createRequests[0].DisplayName); got != networkLoadBalancerDisplayName {
		t.Fatalf("create request displayName = %q, want %q", got, networkLoadBalancerDisplayName)
	}
	createDetails := client.createRequests[0].CreateNetworkLoadBalancerDetails
	assertBoolPtrEqual(t, "create request isPrivate", createDetails.IsPrivate, false)
	assertBoolPtrEqual(t, "create request isPreserveSourceDestination", createDetails.IsPreserveSourceDestination, false)
	assertBoolPtrEqual(t, "create request isSymmetricHashEnabled", createDetails.IsSymmetricHashEnabled, false)
	if got := resource.Status.Id; got != networkLoadBalancerCreatedID {
		t.Fatalf("status.id = %q, want %q", got, networkLoadBalancerCreatedID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	assertNetworkLoadBalancerNoCurrentWorkRequest(t, resource)
}

func TestCreateOrUpdateObservesPendingCreateWorkRequest(t *testing.T) {
	t.Parallel()

	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{},
		workRequestByID: map[string]networkloadbalancersdk.WorkRequest{
			"wr-create-1": networkLoadBalancerWorkRequest(
				"wr-create-1",
				networkloadbalancersdk.OperationStatusInProgress,
				networkloadbalancersdk.OperationTypeCreateNetworkLoadBalancer,
				networkLoadBalancerCreatedID,
			),
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := baseNetworkLoadBalancerResource()
	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() ShouldRequeue = false, want true while provisioning")
	}
	if len(client.workRequests) != 1 {
		t.Fatalf("work request reads = %d, want 1", len(client.workRequests))
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want pending work request state")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("async.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != shared.OSOKAsyncPhaseCreate {
		t.Fatalf("async.phase = %q, want %q", current.Phase, shared.OSOKAsyncPhaseCreate)
	}
	if current.WorkRequestID != "wr-create-1" {
		t.Fatalf("async.workRequestId = %q, want wr-create-1", current.WorkRequestID)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("async.normalizedClass = %q, want %q", current.NormalizedClass, shared.OSOKAsyncClassPending)
	}
}

func TestCreateOrUpdateSurfacesFailedCreateWorkRequest(t *testing.T) {
	t.Parallel()

	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{},
		workRequestByID: map[string]networkloadbalancersdk.WorkRequest{
			"wr-create-1": networkLoadBalancerWorkRequest(
				"wr-create-1",
				networkloadbalancersdk.OperationStatusFailed,
				networkloadbalancersdk.OperationTypeCreateNetworkLoadBalancer,
				networkLoadBalancerCreatedID,
			),
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := baseNetworkLoadBalancerResource()
	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want failed create work request error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want failed response", response)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(client.createRequests))
	}
	if len(client.workRequests) != 1 {
		t.Fatalf("work request reads = %d, want 1", len(client.workRequests))
	}
	assertNetworkLoadBalancerCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassFailed, "wr-create-1")
}

func TestCreateOrUpdateMutableDriftUpdatesNetworkLoadBalancer(t *testing.T) {
	t.Parallel()

	existingResource := baseNetworkLoadBalancerResource()
	existing := networkLoadBalancerFromResource(networkLoadBalancerExistingID, existingResource)
	existing.IsPreserveSourceDestination = common.Bool(true)

	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: existing,
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := baseNetworkLoadBalancerResource()
	resource.Status.Id = networkLoadBalancerExistingID
	resource.Status.CompartmentId = networkLoadBalancerCompartmentID
	resource.Status.DisplayName = networkLoadBalancerDisplayName
	resource.Status.SubnetId = networkLoadBalancerSubnetID
	resource.Status.IsPreserveSourceDestination = true
	resource.Status.OsokStatus.Ocid = networkLoadBalancerExistingID

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}
	if len(client.workRequests) != 1 {
		t.Fatalf("work request reads = %d, want 1", len(client.workRequests))
	}
	gotBool := client.updateRequests[0].IsPreserveSourceDestination
	if gotBool == nil {
		t.Fatal("update request isPreserveSourceDestination = nil, want explicit false")
	}
	if *gotBool {
		t.Fatalf("update request isPreserveSourceDestination = true, want false")
	}
	if got := stringValue(client.updateRequests[0].NetworkLoadBalancerId); got != networkLoadBalancerExistingID {
		t.Fatalf("update request networkLoadBalancerId = %q, want %q", got, networkLoadBalancerExistingID)
	}
	if resource.Status.IsPreserveSourceDestination {
		t.Fatal("status.isPreserveSourceDestination = true, want false after readback")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
	assertNetworkLoadBalancerNoCurrentWorkRequest(t, resource)
}

func TestCreateOrUpdateSkipsNetworkSecurityGroupUpdateWhenReadbackMatches(t *testing.T) {
	t.Parallel()

	resource := trackedNetworkLoadBalancerResource()
	resource.Spec.NetworkSecurityGroupIds = []string{
		networkLoadBalancerUpdatedNSGID,
		networkLoadBalancerNSGID,
	}
	existing := networkLoadBalancerFromResource(networkLoadBalancerExistingID, resource)
	existing.NetworkSecurityGroupIds = []string{
		networkLoadBalancerNSGID,
		networkLoadBalancerUpdatedNSGID,
	}
	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: existing,
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.nsgRequests) != 0 {
		t.Fatalf("network security group update requests = %d, want 0 for matching readback", len(client.nsgRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("network load balancer update requests = %d, want 0 for matching NSG readback", len(client.updateRequests))
	}
}

func TestCreateOrUpdateNetworkSecurityGroupDriftUsesUpdateNetworkSecurityGroups(t *testing.T) {
	t.Parallel()

	existingResource := baseNetworkLoadBalancerResource()
	existingResource.Spec.NetworkSecurityGroupIds = []string{networkLoadBalancerNSGID}
	existing := networkLoadBalancerFromResource(networkLoadBalancerExistingID, existingResource)
	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: existing,
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := trackedNetworkLoadBalancerResource()
	resource.Spec.NetworkSecurityGroupIds = []string{networkLoadBalancerUpdatedNSGID}

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.nsgRequests) != 1 {
		t.Fatalf("network security group update requests = %d, want 1", len(client.nsgRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("network load balancer update requests = %d, want 0 for NSG drift", len(client.updateRequests))
	}
	if len(client.workRequests) != 1 {
		t.Fatalf("work request reads = %d, want 1", len(client.workRequests))
	}
	nsgUpdate := client.nsgRequests[0]
	if got := stringValue(nsgUpdate.NetworkLoadBalancerId); got != networkLoadBalancerExistingID {
		t.Fatalf("nsg update networkLoadBalancerId = %q, want %q", got, networkLoadBalancerExistingID)
	}
	if !reflect.DeepEqual(nsgUpdate.NetworkSecurityGroupIds, []string{networkLoadBalancerUpdatedNSGID}) {
		t.Fatalf("nsg update networkSecurityGroupIds = %#v, want %#v", nsgUpdate.NetworkSecurityGroupIds, []string{networkLoadBalancerUpdatedNSGID})
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-nsg-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-nsg-1", got)
	}
	assertNetworkLoadBalancerStringSliceEqual(t, "status.networkSecurityGroupIds", resource.Status.NetworkSecurityGroupIds, []string{networkLoadBalancerUpdatedNSGID})
	assertNetworkLoadBalancerNoCurrentWorkRequest(t, resource)
}

func TestCreateOrUpdateResumesPendingNetworkSecurityGroupUpdateWorkRequest(t *testing.T) {
	t.Parallel()

	existingResource := baseNetworkLoadBalancerResource()
	existingResource.Spec.NetworkSecurityGroupIds = []string{networkLoadBalancerNSGID}
	existing := networkLoadBalancerFromResource(networkLoadBalancerExistingID, existingResource)
	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: existing,
		},
		workRequestByID: map[string]networkloadbalancersdk.WorkRequest{
			"wr-update-nsg-1": networkLoadBalancerWorkRequest(
				"wr-update-nsg-1",
				networkloadbalancersdk.OperationStatusInProgress,
				networkloadbalancersdk.OperationTypeUpdateNsgs,
				networkLoadBalancerExistingID,
			),
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := trackedNetworkLoadBalancerResource()
	resource.Spec.NetworkSecurityGroupIds = []string{networkLoadBalancerUpdatedNSGID}

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assertNetworkLoadBalancerPendingUpdateResponse(t, "first reconcile", response, err)
	assertNetworkLoadBalancerCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending, "wr-update-nsg-1")
	if len(client.updateRequests) != 0 {
		t.Fatalf("first reconcile network load balancer update requests = %d, want 0", len(client.updateRequests))
	}
	if len(client.workRequests) != 1 {
		t.Fatalf("first reconcile work request reads = %d, want 1", len(client.workRequests))
	}
	if len(client.nsgRequests) != 1 {
		t.Fatalf("first reconcile network security group update requests = %d, want 1", len(client.nsgRequests))
	}

	response, err = serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assertNetworkLoadBalancerPendingUpdateResponse(t, "second reconcile", response, err)
	assertNetworkLoadBalancerCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending, "wr-update-nsg-1")
	if len(client.updateRequests) != 0 {
		t.Fatalf("second reconcile network load balancer update requests = %d, want 0", len(client.updateRequests))
	}
	if len(client.workRequests) != 2 {
		t.Fatalf("second reconcile work request reads = %d, want 2", len(client.workRequests))
	}
	if len(client.nsgRequests) != 1 {
		t.Fatalf("second reconcile network security group update requests = %d, want 1", len(client.nsgRequests))
	}
}

func TestCreateOrUpdateResumesPendingUpdateWorkRequestWithoutDuplicateMutation(t *testing.T) {
	t.Parallel()

	existing := networkLoadBalancerFromResource(networkLoadBalancerExistingID, baseNetworkLoadBalancerResource())
	existing.IsPreserveSourceDestination = common.Bool(true)
	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: existing,
		},
		workRequestByID: map[string]networkloadbalancersdk.WorkRequest{
			"wr-update-1": networkLoadBalancerWorkRequest(
				"wr-update-1",
				networkloadbalancersdk.OperationStatusInProgress,
				networkloadbalancersdk.OperationTypeUpdateNetworkLoadBalancer,
				networkLoadBalancerExistingID,
			),
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := trackedNetworkLoadBalancerResource()
	resource.Status.IsPreserveSourceDestination = true

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assertNetworkLoadBalancerPendingUpdateResume(t, "first reconcile", response, err, resource, client, 1, 1)

	response, err = serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assertNetworkLoadBalancerPendingUpdateResume(t, "second reconcile", response, err, resource, client, 1, 2)
}

func TestCreateOrUpdateSurfacesFailedUpdateWorkRequest(t *testing.T) {
	t.Parallel()

	existing := networkLoadBalancerFromResource(networkLoadBalancerExistingID, baseNetworkLoadBalancerResource())
	existing.IsPreserveSourceDestination = common.Bool(true)
	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: existing,
		},
		workRequestByID: map[string]networkloadbalancersdk.WorkRequest{
			"wr-update-1": networkLoadBalancerWorkRequest(
				"wr-update-1",
				networkloadbalancersdk.OperationStatusFailed,
				networkloadbalancersdk.OperationTypeUpdateNetworkLoadBalancer,
				networkLoadBalancerExistingID,
			),
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := trackedNetworkLoadBalancerResource()
	resource.Status.IsPreserveSourceDestination = true

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want failed update work request error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want failed response", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}
	if len(client.workRequests) != 1 {
		t.Fatalf("work request reads = %d, want 1", len(client.workRequests))
	}
	assertNetworkLoadBalancerCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassFailed, "wr-update-1")
}

func TestCreateOrUpdateMutableIPv6DriftUpdatesNetworkLoadBalancer(t *testing.T) {
	t.Parallel()

	existing := networkLoadBalancerFromResource(networkLoadBalancerExistingID, baseNetworkLoadBalancerResource())
	existing.NlbIpVersion = networkloadbalancersdk.NlbIpVersionIpv4

	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: existing,
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := baseNetworkLoadBalancerResource()
	resource.Spec.NlbIpVersion = string(networkloadbalancersdk.NlbIpVersionIpv4AndIpv6)
	resource.Spec.SubnetIpv6Cidr = networkLoadBalancerSubnetIPv6CIDR
	resource.Spec.AssignedIpv6 = networkLoadBalancerAssignedIPv6
	resource.Spec.ReservedIpv6Id = networkLoadBalancerReservedIPv6ID
	resource.Status.Id = networkLoadBalancerExistingID
	resource.Status.CompartmentId = networkLoadBalancerCompartmentID
	resource.Status.DisplayName = networkLoadBalancerDisplayName
	resource.Status.SubnetId = networkLoadBalancerSubnetID
	resource.Status.OsokStatus.Ocid = networkLoadBalancerExistingID

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}
	update := client.updateRequests[0]
	if update.NlbIpVersion != networkloadbalancersdk.NlbIpVersionIpv4AndIpv6 {
		t.Fatalf("update request nlbIpVersion = %q, want %q", update.NlbIpVersion, networkloadbalancersdk.NlbIpVersionIpv4AndIpv6)
	}
	if got := stringValue(update.SubnetIpv6Cidr); got != networkLoadBalancerSubnetIPv6CIDR {
		t.Fatalf("update request subnetIpv6Cidr = %q, want %q", got, networkLoadBalancerSubnetIPv6CIDR)
	}
	if got := stringValue(update.AssignedIpv6); got != networkLoadBalancerAssignedIPv6 {
		t.Fatalf("update request assignedIpv6 = %q, want %q", got, networkLoadBalancerAssignedIPv6)
	}
	if got := stringValue(update.ReservedIpv6Id); got != networkLoadBalancerReservedIPv6ID {
		t.Fatalf("update request reservedIpv6Id = %q, want %q", got, networkLoadBalancerReservedIPv6ID)
	}
}

func TestCreateOrUpdateSkipsIPv6UpdateWhenReadbackMatches(t *testing.T) {
	t.Parallel()

	resource := baseNetworkLoadBalancerResource()
	resource.Spec.NlbIpVersion = string(networkloadbalancersdk.NlbIpVersionIpv4AndIpv6)
	resource.Spec.SubnetIpv6Cidr = networkLoadBalancerSubnetIPv6CIDR
	resource.Spec.AssignedIpv6 = networkLoadBalancerAssignedIPv6
	resource.Spec.ReservedIpv6Id = networkLoadBalancerReservedIPv6ID

	existing := networkLoadBalancerFromResource(networkLoadBalancerExistingID, resource)
	existing.IpAddresses = []networkloadbalancersdk.IpAddress{
		{
			IpAddress: common.String(networkLoadBalancerAssignedIPv6),
			IpVersion: networkloadbalancersdk.IpVersionIpv6,
			ReservedIp: &networkloadbalancersdk.ReservedIp{
				Id: common.String(networkLoadBalancerReservedIPv6ID),
			},
		},
	}
	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: existing,
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource.Status.Id = networkLoadBalancerExistingID
	resource.Status.CompartmentId = networkLoadBalancerCompartmentID
	resource.Status.DisplayName = networkLoadBalancerDisplayName
	resource.Status.SubnetId = networkLoadBalancerSubnetID
	resource.Status.OsokStatus.Ocid = networkLoadBalancerExistingID

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 for matching IPv6 readback", len(client.updateRequests))
	}
}

func TestCreateOrUpdateRejectsInvalidSubnetIPv6CIDRBeforeUpdate(t *testing.T) {
	t.Parallel()

	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: networkLoadBalancerFromResource(networkLoadBalancerExistingID, baseNetworkLoadBalancerResource()),
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := baseNetworkLoadBalancerResource()
	resource.Spec.SubnetIpv6Cidr = "not-a-cidr"
	resource.Status.Id = networkLoadBalancerExistingID
	resource.Status.CompartmentId = networkLoadBalancerCompartmentID
	resource.Status.DisplayName = networkLoadBalancerDisplayName
	resource.Status.SubnetId = networkLoadBalancerSubnetID
	resource.Status.OsokStatus.Ocid = networkLoadBalancerExistingID

	_, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want invalid subnetIpv6Cidr rejection")
	}
	if !strings.Contains(err.Error(), "subnetIpv6Cidr") {
		t.Fatalf("CreateOrUpdate() error = %q, want subnetIpv6Cidr rejection", err.Error())
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 after invalid subnetIpv6Cidr", len(client.updateRequests))
	}
}

func TestCreateOrUpdateRejectsCreateOnlyNetworkLoadBalancerDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	existingResource := baseNetworkLoadBalancerResource()
	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: networkLoadBalancerFromResource(networkLoadBalancerExistingID, existingResource),
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := baseNetworkLoadBalancerResource()
	resource.Spec.SubnetId = "ocid1.subnet.oc1..replacementuniqueID"
	resource.Status.Id = networkLoadBalancerExistingID
	resource.Status.CompartmentId = networkLoadBalancerCompartmentID
	resource.Status.DisplayName = networkLoadBalancerDisplayName
	resource.Status.SubnetId = networkLoadBalancerSubnetID
	resource.Status.OsokStatus.Ocid = networkLoadBalancerExistingID

	_, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "subnetId") {
		t.Fatalf("CreateOrUpdate() error = %q, want subnetId drift", err.Error())
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 after create-only drift", len(client.updateRequests))
	}
}

func TestCreateOrUpdateAllowsAssignedPrivateIPv4WhenReadbackMatches(t *testing.T) {
	t.Parallel()

	resource := baseNetworkLoadBalancerResource()
	resource.Spec.AssignedPrivateIpv4 = networkLoadBalancerAssignedIPv4
	existing := networkLoadBalancerFromResource(networkLoadBalancerExistingID, resource)
	existing.IpAddresses = []networkloadbalancersdk.IpAddress{
		{
			IpAddress: common.String(networkLoadBalancerAssignedIPv4),
			IpVersion: networkloadbalancersdk.IpVersionIpv4,
		},
	}
	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: existing,
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource.Status.Id = networkLoadBalancerExistingID
	resource.Status.CompartmentId = networkLoadBalancerCompartmentID
	resource.Status.DisplayName = networkLoadBalancerDisplayName
	resource.Status.SubnetId = networkLoadBalancerSubnetID
	resource.Status.OsokStatus.Ocid = networkLoadBalancerExistingID

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 for matching assignedPrivateIpv4 readback", len(client.updateRequests))
	}
}

func TestCreateOrUpdateRejectsAssignedPrivateIPv4CreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	existing := networkLoadBalancerFromResource(networkLoadBalancerExistingID, baseNetworkLoadBalancerResource())
	existing.IpAddresses = []networkloadbalancersdk.IpAddress{
		{
			IpAddress: common.String("10.0.0.11"),
			IpVersion: networkloadbalancersdk.IpVersionIpv4,
		},
	}
	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: existing,
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := trackedNetworkLoadBalancerResource()
	resource.Spec.AssignedPrivateIpv4 = networkLoadBalancerAssignedIPv4

	_, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want assignedPrivateIpv4 create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "assignedPrivateIpv4") {
		t.Fatalf("CreateOrUpdate() error = %q, want assignedPrivateIpv4 drift", err.Error())
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 after assignedPrivateIpv4 drift", len(client.updateRequests))
	}
}

func TestCreateOrUpdateAllowsReservedIPsWhenReadbackMatches(t *testing.T) {
	t.Parallel()

	resource := baseNetworkLoadBalancerResource()
	resource.Spec.ReservedIps = []networkloadbalancerv1beta1.NetworkLoadBalancerReservedIp{
		{Id: networkLoadBalancerReservedIPv4ID},
	}
	existing := networkLoadBalancerFromResource(networkLoadBalancerExistingID, resource)
	existing.IpAddresses = []networkloadbalancersdk.IpAddress{
		{
			IpAddress: common.String(networkLoadBalancerAssignedIPv4),
			IpVersion: networkloadbalancersdk.IpVersionIpv4,
			ReservedIp: &networkloadbalancersdk.ReservedIp{
				Id: common.String(networkLoadBalancerReservedIPv4ID),
			},
		},
	}
	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: existing,
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource.Status.Id = networkLoadBalancerExistingID
	resource.Status.CompartmentId = networkLoadBalancerCompartmentID
	resource.Status.DisplayName = networkLoadBalancerDisplayName
	resource.Status.SubnetId = networkLoadBalancerSubnetID
	resource.Status.OsokStatus.Ocid = networkLoadBalancerExistingID

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 for matching reservedIps readback", len(client.updateRequests))
	}
}

func TestCreateOrUpdateRejectsReservedIPsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	existing := networkLoadBalancerFromResource(networkLoadBalancerExistingID, baseNetworkLoadBalancerResource())
	existing.IpAddresses = []networkloadbalancersdk.IpAddress{
		{
			IpAddress: common.String(networkLoadBalancerAssignedIPv4),
			IpVersion: networkloadbalancersdk.IpVersionIpv4,
			ReservedIp: &networkloadbalancersdk.ReservedIp{
				Id: common.String("ocid1.privateip.oc1..differentreserveduniqueID"),
			},
		},
	}
	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: existing,
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := trackedNetworkLoadBalancerResource()
	resource.Spec.ReservedIps = []networkloadbalancerv1beta1.NetworkLoadBalancerReservedIp{
		{Id: networkLoadBalancerReservedIPv4ID},
	}

	_, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want reservedIps create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "reservedIps") {
		t.Fatalf("CreateOrUpdate() error = %q, want reservedIps drift", err.Error())
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 after reservedIps drift", len(client.updateRequests))
	}
}

func TestDeleteRetainsFinalizerWhileNetworkLoadBalancerIsDeleting(t *testing.T) {
	t.Parallel()

	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: networkLoadBalancerFromResource(networkLoadBalancerExistingID, baseNetworkLoadBalancerResource()),
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := baseNetworkLoadBalancerResource()
	resource.Status.Id = networkLoadBalancerExistingID
	resource.Status.CompartmentId = networkLoadBalancerCompartmentID
	resource.Status.DisplayName = networkLoadBalancerDisplayName
	resource.Status.OsokStatus.Ocid = networkLoadBalancerExistingID

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while OCI resource is DELETING")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	if len(client.workRequests) != 1 {
		t.Fatalf("work request reads = %d, want 1", len(client.workRequests))
	}
	if got := stringValue(client.deleteRequests[0].NetworkLoadBalancerId); got != networkLoadBalancerExistingID {
		t.Fatalf("delete request networkLoadBalancerId = %q, want %q", got, networkLoadBalancerExistingID)
	}
	assertNetworkLoadBalancerCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "wr-delete-1")
}

func TestDeleteCompletesAfterSucceededNetworkLoadBalancerWorkRequest(t *testing.T) {
	t.Parallel()

	client := &fakeNetworkLoadBalancerOCIClient{
		deleteRemovesResource: true,
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: networkLoadBalancerFromResource(networkLoadBalancerExistingID, baseNetworkLoadBalancerResource()),
		},
		workRequestByID: map[string]networkloadbalancersdk.WorkRequest{
			"wr-delete-1": networkLoadBalancerWorkRequest(
				"wr-delete-1",
				networkloadbalancersdk.OperationStatusSucceeded,
				networkloadbalancersdk.OperationTypeDeleteNetworkLoadBalancer,
				networkLoadBalancerExistingID,
			),
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := trackedNetworkLoadBalancerResource()

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after succeeded delete work request confirms absence")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	if len(client.workRequests) != 1 {
		t.Fatalf("work request reads = %d, want 1", len(client.workRequests))
	}
	assertNetworkLoadBalancerNoCurrentWorkRequest(t, resource)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete timestamp")
	}
}

func TestDeleteSurfacesFailedNetworkLoadBalancerWorkRequest(t *testing.T) {
	t.Parallel()

	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: networkLoadBalancerFromResource(networkLoadBalancerExistingID, baseNetworkLoadBalancerResource()),
		},
		workRequestByID: map[string]networkloadbalancersdk.WorkRequest{
			"wr-delete-1": networkLoadBalancerWorkRequest(
				"wr-delete-1",
				networkloadbalancersdk.OperationStatusFailed,
				networkloadbalancersdk.OperationTypeDeleteNetworkLoadBalancer,
				networkLoadBalancerExistingID,
			),
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := trackedNetworkLoadBalancerResource()

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want failed delete work request error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained after failed work request")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	if len(client.workRequests) != 1 {
		t.Fatalf("work request reads = %d, want 1", len(client.workRequests))
	}
	assertNetworkLoadBalancerCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassFailed, "wr-delete-1")
}

func TestDeleteObservesPendingNetworkLoadBalancerWriteWorkRequestBeforeDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		phase         shared.OSOKAsyncPhase
		workRequestID string
		operation     networkloadbalancersdk.OperationTypeEnum
	}{
		{
			name:          "create",
			phase:         shared.OSOKAsyncPhaseCreate,
			workRequestID: "wr-create-1",
			operation:     networkloadbalancersdk.OperationTypeCreateNetworkLoadBalancer,
		},
		{
			name:          "update",
			phase:         shared.OSOKAsyncPhaseUpdate,
			workRequestID: "wr-update-1",
			operation:     networkloadbalancersdk.OperationTypeUpdateNetworkLoadBalancer,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := &fakeNetworkLoadBalancerOCIClient{
				workRequestByID: map[string]networkloadbalancersdk.WorkRequest{
					tc.workRequestID: networkLoadBalancerWorkRequest(
						tc.workRequestID,
						networkloadbalancersdk.OperationStatusInProgress,
						tc.operation,
						networkLoadBalancerExistingID,
					),
				},
			}
			serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
			resource := trackedNetworkLoadBalancerResource()
			resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
				Source:          shared.OSOKAsyncSourceWorkRequest,
				Phase:           tc.phase,
				WorkRequestID:   tc.workRequestID,
				NormalizedClass: shared.OSOKAsyncClassPending,
			}

			deleted, err := serviceClient.Delete(context.Background(), resource)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if deleted {
				t.Fatal("Delete() deleted = true, want finalizer retained while write work request is pending")
			}
			if len(client.deleteRequests) != 0 {
				t.Fatalf("delete requests = %d, want 0 while write work request is pending", len(client.deleteRequests))
			}
			if len(client.workRequests) != 1 {
				t.Fatalf("work request reads = %d, want 1", len(client.workRequests))
			}
			assertNetworkLoadBalancerCurrentWorkRequest(t, resource, tc.phase, shared.OSOKAsyncClassPending, tc.workRequestID)
		})
	}
}

func TestDeleteRejectsAuthShapedPreDeleteNetworkLoadBalancerConfirmRead(t *testing.T) {
	t.Parallel()

	client := &fakeNetworkLoadBalancerOCIClient{
		getErr: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := trackedNetworkLoadBalancerResource()

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want pre-delete auth-shaped GetNetworkLoadBalancer 404 to stay fatal")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirmation 404", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained after auth-shaped confirm read")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 after auth-shaped confirm read", len(client.deleteRequests))
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1", len(client.getRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDeleteSucceededWorkRequestTreatsAuthShapedNetworkLoadBalancerConfirmationNotFoundAsFatal(t *testing.T) {
	t.Parallel()

	client := &fakeNetworkLoadBalancerOCIClient{
		getErr: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
		workRequestByID: map[string]networkloadbalancersdk.WorkRequest{
			"wr-delete-1": networkLoadBalancerWorkRequest(
				"wr-delete-1",
				networkloadbalancersdk.OperationStatusSucceeded,
				networkloadbalancersdk.OperationTypeDeleteNetworkLoadBalancer,
				networkLoadBalancerExistingID,
			),
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := trackedNetworkLoadBalancerResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   "wr-delete-1",
		NormalizedClass: shared.OSOKAsyncClassPending,
	}

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want succeeded delete work request auth-shaped confirmation 404 to stay fatal")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirmation 404", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained after auth-shaped confirm read")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 after succeeded work request confirmation fails", len(client.deleteRequests))
	}
	if len(client.workRequests) != 1 {
		t.Fatalf("work request reads = %d, want 1", len(client.workRequests))
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1", len(client.getRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	assertNetworkLoadBalancerCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "wr-delete-1")
}

func TestDeleteTreatsNotAuthorizedOrNotFoundAsAmbiguous(t *testing.T) {
	t.Parallel()

	client := &fakeNetworkLoadBalancerOCIClient{
		deleteErr: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: networkLoadBalancerFromResource(networkLoadBalancerExistingID, baseNetworkLoadBalancerResource()),
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := baseNetworkLoadBalancerResource()
	resource.Status.Id = networkLoadBalancerExistingID
	resource.Status.CompartmentId = networkLoadBalancerCompartmentID
	resource.Status.DisplayName = networkLoadBalancerDisplayName
	resource.Status.OsokStatus.Ocid = networkLoadBalancerExistingID

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for ambiguous not found")
	}
	if !strings.Contains(err.Error(), errorutil.NotAuthorizedOrNotFound) {
		t.Fatalf("Delete() error = %q, want %q", err.Error(), errorutil.NotAuthorizedOrNotFound)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestCreateOrUpdateFindsExistingNetworkLoadBalancerAcrossListPages(t *testing.T) {
	t.Parallel()

	existing := networkLoadBalancerFromResource(networkLoadBalancerExistingID, baseNetworkLoadBalancerResource())
	client := &fakeNetworkLoadBalancerOCIClient{
		networkLoadBalancers: map[string]networkloadbalancersdk.NetworkLoadBalancer{
			networkLoadBalancerExistingID: existing,
		},
		listPages: []networkloadbalancersdk.ListNetworkLoadBalancersResponse{
			{
				NetworkLoadBalancerCollection: networkloadbalancersdk.NetworkLoadBalancerCollection{Items: nil},
				OpcNextPage:                   common.String("page-2"),
			},
			{
				NetworkLoadBalancerCollection: networkloadbalancersdk.NetworkLoadBalancerCollection{
					Items: []networkloadbalancersdk.NetworkLoadBalancerSummary{
						networkLoadBalancerSummaryFromResource(existing),
					},
				},
			},
		},
	}
	serviceClient := newNetworkLoadBalancerServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)

	resource := baseNetworkLoadBalancerResource()
	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful response", response)
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2 pages", len(client.listRequests))
	}
	if got := stringValue(client.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second list page = %q, want page-2", got)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for paginated bind path", len(client.createRequests))
	}
	if got := resource.Status.Id; got != networkLoadBalancerExistingID {
		t.Fatalf("status.id = %q, want %q", got, networkLoadBalancerExistingID)
	}
}

type generatedruntimeRequestField struct {
	FieldName        string
	RequestName      string
	Contribution     string
	PreferResourceID bool
	LookupPaths      string
}

func requestFieldSnapshot(fields []generatedruntime.RequestField) []generatedruntimeRequestField {
	snapshot := make([]generatedruntimeRequestField, 0, len(fields))
	for _, field := range fields {
		snapshot = append(snapshot, generatedruntimeRequestField{
			FieldName:        field.FieldName,
			RequestName:      field.RequestName,
			Contribution:     field.Contribution,
			PreferResourceID: field.PreferResourceID,
			LookupPaths:      strings.Join(field.LookupPaths, ","),
		})
	}
	return snapshot
}

func requireNetworkLoadBalancerRuntimeSemantics(
	t *testing.T,
	got *generatedruntime.Semantics,
) *generatedruntime.Semantics {
	t.Helper()
	if got == nil {
		t.Fatal("newReviewedNetworkLoadBalancerRuntimeSemantics() = nil")
	}
	return got
}

func requireNetworkLoadBalancerAsyncSemantics(
	t *testing.T,
	got *generatedruntime.Semantics,
) *generatedruntime.AsyncSemantics {
	t.Helper()
	if got.Async == nil {
		t.Fatal("Async = nil, want workrequest async semantics")
	}
	return got.Async
}

func requireNetworkLoadBalancerWorkRequestSemantics(
	t *testing.T,
	got *generatedruntime.AsyncSemantics,
) *generatedruntime.WorkRequestSemantics {
	t.Helper()
	if got.WorkRequest == nil {
		t.Fatal("Async.WorkRequest = nil, want service-sdk work request semantics")
	}
	return got.WorkRequest
}

func assertNetworkLoadBalancerStringEqual(t *testing.T, field string, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %q, want %q", field, got, want)
	}
}

func assertNetworkLoadBalancerStringSliceEqual(t *testing.T, field string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
}

func assertBoolPtrEqual(t *testing.T, field string, got *bool, want bool) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %v", field, want)
	}
	if *got != want {
		t.Fatalf("%s = %v, want %v", field, *got, want)
	}
}

func assertNetworkLoadBalancerPendingUpdateResponse(
	t *testing.T,
	label string,
	response servicemanager.OSOKResponse,
	err error,
) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s CreateOrUpdate() error = %v", label, err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("%s CreateOrUpdate() response = %+v, want pending update requeue", label, response)
	}
}

func assertNetworkLoadBalancerPendingUpdateResume(
	t *testing.T,
	label string,
	response servicemanager.OSOKResponse,
	err error,
	resource *networkloadbalancerv1beta1.NetworkLoadBalancer,
	client *fakeNetworkLoadBalancerOCIClient,
	wantUpdateRequests int,
	wantWorkRequestReads int,
) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s CreateOrUpdate() error = %v", label, err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("%s CreateOrUpdate() response = %+v, want pending update requeue", label, response)
	}
	if len(client.updateRequests) != wantUpdateRequests {
		t.Fatalf("%s update requests = %d, want %d", label, len(client.updateRequests), wantUpdateRequests)
	}
	if len(client.workRequests) != wantWorkRequestReads {
		t.Fatalf("%s work request reads = %d, want %d", label, len(client.workRequests), wantWorkRequestReads)
	}
	assertNetworkLoadBalancerCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending, "wr-update-1")
}

func baseNetworkLoadBalancerResource() *networkloadbalancerv1beta1.NetworkLoadBalancer {
	return &networkloadbalancerv1beta1.NetworkLoadBalancer{
		Spec: networkloadbalancerv1beta1.NetworkLoadBalancerSpec{
			CompartmentId: networkLoadBalancerCompartmentID,
			DisplayName:   networkLoadBalancerDisplayName,
			SubnetId:      networkLoadBalancerSubnetID,
			FreeformTags: map[string]string{
				"scenario": "runtime",
			},
		},
	}
}

func trackedNetworkLoadBalancerResource() *networkloadbalancerv1beta1.NetworkLoadBalancer {
	resource := baseNetworkLoadBalancerResource()
	resource.Status.Id = networkLoadBalancerExistingID
	resource.Status.CompartmentId = networkLoadBalancerCompartmentID
	resource.Status.DisplayName = networkLoadBalancerDisplayName
	resource.Status.SubnetId = networkLoadBalancerSubnetID
	resource.Status.OsokStatus.Ocid = networkLoadBalancerExistingID
	return resource
}

func defaultNetworkLoadBalancerWorkRequest(workRequestID string) networkloadbalancersdk.WorkRequest {
	switch workRequestID {
	case "wr-update-1":
		return networkLoadBalancerWorkRequest(
			workRequestID,
			networkloadbalancersdk.OperationStatusSucceeded,
			networkloadbalancersdk.OperationTypeUpdateNetworkLoadBalancer,
			networkLoadBalancerExistingID,
		)
	case "wr-update-nsg-1":
		return networkLoadBalancerWorkRequest(
			workRequestID,
			networkloadbalancersdk.OperationStatusSucceeded,
			networkloadbalancersdk.OperationTypeUpdateNsgs,
			networkLoadBalancerExistingID,
		)
	case "wr-delete-1":
		return networkLoadBalancerWorkRequest(
			workRequestID,
			networkloadbalancersdk.OperationStatusInProgress,
			networkloadbalancersdk.OperationTypeDeleteNetworkLoadBalancer,
			networkLoadBalancerExistingID,
		)
	default:
		return networkLoadBalancerWorkRequest(
			workRequestID,
			networkloadbalancersdk.OperationStatusSucceeded,
			networkloadbalancersdk.OperationTypeCreateNetworkLoadBalancer,
			networkLoadBalancerCreatedID,
		)
	}
}

func networkLoadBalancerWorkRequest(
	id string,
	status networkloadbalancersdk.OperationStatusEnum,
	operation networkloadbalancersdk.OperationTypeEnum,
	resourceID string,
) networkloadbalancersdk.WorkRequest {
	return networkloadbalancersdk.WorkRequest{
		Id:              common.String(id),
		Status:          status,
		OperationType:   operation,
		CompartmentId:   common.String(networkLoadBalancerCompartmentID),
		PercentComplete: common.Float32(50),
		Resources: []networkloadbalancersdk.WorkRequestResource{
			{
				EntityType: common.String("networkloadbalancer"),
				ActionType: networkLoadBalancerActionForOperation(operation),
				Identifier: common.String(resourceID),
			},
		},
	}
}

func networkLoadBalancerActionForOperation(operation networkloadbalancersdk.OperationTypeEnum) networkloadbalancersdk.ActionTypeEnum {
	switch operation {
	case networkloadbalancersdk.OperationTypeCreateNetworkLoadBalancer:
		return networkloadbalancersdk.ActionTypeCreated
	case networkloadbalancersdk.OperationTypeUpdateNetworkLoadBalancer:
		return networkloadbalancersdk.ActionTypeUpdated
	case networkloadbalancersdk.OperationTypeUpdateNsgs:
		return networkloadbalancersdk.ActionTypeUpdated
	case networkloadbalancersdk.OperationTypeDeleteNetworkLoadBalancer:
		return networkloadbalancersdk.ActionTypeDeleted
	default:
		return networkloadbalancersdk.ActionTypeRelated
	}
}

func assertNetworkLoadBalancerCurrentWorkRequest(
	t *testing.T,
	resource *networkloadbalancerv1beta1.NetworkLoadBalancer,
	wantPhase shared.OSOKAsyncPhase,
	wantClass shared.OSOKAsyncNormalizedClass,
	wantID string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want work request tracker")
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("async.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != wantPhase {
		t.Fatalf("async.phase = %q, want %q", current.Phase, wantPhase)
	}
	if current.NormalizedClass != wantClass {
		t.Fatalf("async.normalizedClass = %q, want %q", current.NormalizedClass, wantClass)
	}
	if current.WorkRequestID != wantID {
		t.Fatalf("async.workRequestId = %q, want %q", current.WorkRequestID, wantID)
	}
}

func assertNetworkLoadBalancerNoCurrentWorkRequest(t *testing.T, resource *networkloadbalancerv1beta1.NetworkLoadBalancer) {
	t.Helper()
	if current := resource.Status.OsokStatus.Async.Current; current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil", current)
	}
}

func networkLoadBalancerFromResource(id string, resource *networkloadbalancerv1beta1.NetworkLoadBalancer) networkloadbalancersdk.NetworkLoadBalancer {
	return networkloadbalancersdk.NetworkLoadBalancer{
		Id:                          common.String(id),
		CompartmentId:               common.String(resource.Spec.CompartmentId),
		DisplayName:                 common.String(resource.Spec.DisplayName),
		LifecycleState:              networkloadbalancersdk.LifecycleStateActive,
		SubnetId:                    common.String(resource.Spec.SubnetId),
		IsPrivate:                   common.Bool(resource.Spec.IsPrivate),
		IsPreserveSourceDestination: common.Bool(resource.Spec.IsPreserveSourceDestination),
		IsSymmetricHashEnabled:      common.Bool(resource.Spec.IsSymmetricHashEnabled),
		NetworkSecurityGroupIds:     append([]string(nil), resource.Spec.NetworkSecurityGroupIds...),
		NlbIpVersion:                networkloadbalancersdk.NlbIpVersionEnum(resource.Spec.NlbIpVersion),
		FreeformTags:                copyStringMap(resource.Spec.FreeformTags),
		DefinedTags:                 copyDefinedTags(resource.Spec.DefinedTags),
		SecurityAttributes:          copyDefinedTags(resource.Spec.SecurityAttributes),
	}
}

func networkLoadBalancerFromCreateDetails(id string, details networkloadbalancersdk.CreateNetworkLoadBalancerDetails) networkloadbalancersdk.NetworkLoadBalancer {
	return networkloadbalancersdk.NetworkLoadBalancer{
		Id:                          common.String(id),
		CompartmentId:               details.CompartmentId,
		DisplayName:                 details.DisplayName,
		LifecycleState:              networkloadbalancersdk.LifecycleStateActive,
		SubnetId:                    details.SubnetId,
		IsPrivate:                   details.IsPrivate,
		IsPreserveSourceDestination: details.IsPreserveSourceDestination,
		IsSymmetricHashEnabled:      details.IsSymmetricHashEnabled,
		NetworkSecurityGroupIds:     append([]string(nil), details.NetworkSecurityGroupIds...),
		NlbIpVersion:                details.NlbIpVersion,
		FreeformTags:                copyStringMap(details.FreeformTags),
		DefinedTags:                 copyNestedAnyMap(details.DefinedTags),
		SecurityAttributes:          copyNestedAnyMap(details.SecurityAttributes),
	}
}

func networkLoadBalancerSummaryFromResource(resource networkloadbalancersdk.NetworkLoadBalancer) networkloadbalancersdk.NetworkLoadBalancerSummary {
	return networkloadbalancersdk.NetworkLoadBalancerSummary{
		Id:                          resource.Id,
		CompartmentId:               resource.CompartmentId,
		DisplayName:                 resource.DisplayName,
		LifecycleState:              resource.LifecycleState,
		SubnetId:                    resource.SubnetId,
		IsPrivate:                   resource.IsPrivate,
		IsPreserveSourceDestination: resource.IsPreserveSourceDestination,
		IsSymmetricHashEnabled:      resource.IsSymmetricHashEnabled,
		NetworkSecurityGroupIds:     append([]string(nil), resource.NetworkSecurityGroupIds...),
		NlbIpVersion:                resource.NlbIpVersion,
		FreeformTags:                copyStringMap(resource.FreeformTags),
		DefinedTags:                 copyNestedAnyMap(resource.DefinedTags),
		SecurityAttributes:          copyNestedAnyMap(resource.SecurityAttributes),
	}
}

func upsertIPv6Address(ipAddresses []networkloadbalancersdk.IpAddress, address string, reservedID *string) []networkloadbalancersdk.IpAddress {
	if len(ipAddresses) == 0 {
		return []networkloadbalancersdk.IpAddress{
			networkLoadBalancerIPv6Address(address, reservedID),
		}
	}

	output := append([]networkloadbalancersdk.IpAddress(nil), ipAddresses...)
	for index, ipAddress := range output {
		if ipAddress.IpVersion != "" && ipAddress.IpVersion != networkloadbalancersdk.IpVersionIpv6 {
			continue
		}
		output[index] = networkLoadBalancerIPv6Address(address, reservedID)
		return output
	}
	return append(output, networkLoadBalancerIPv6Address(address, reservedID))
}

func networkLoadBalancerIPv6Address(address string, reservedID *string) networkloadbalancersdk.IpAddress {
	ipAddress := networkloadbalancersdk.IpAddress{
		IpAddress: common.String(address),
		IpVersion: networkloadbalancersdk.IpVersionIpv6,
	}
	if reservedID != nil {
		ipAddress.ReservedIp = &networkloadbalancersdk.ReservedIp{Id: common.String(*reservedID)}
	}
	return ipAddress
}

func copyStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func copyDefinedTags(input map[string]shared.MapValue) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	output := make(map[string]map[string]interface{}, len(input))
	for namespace, value := range input {
		converted := make(map[string]interface{}, len(value))
		for key, item := range value {
			converted[key] = item
		}
		output[namespace] = converted
	}
	return output
}

func copyNestedAnyMap(input map[string]map[string]interface{}) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	output := make(map[string]map[string]interface{}, len(input))
	for namespace, value := range input {
		output[namespace] = copyAnyMap(value)
	}
	return output
}

func copyAnyMap(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return nil
	}
	output := make(map[string]interface{}, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
