/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package listener

import (
	"context"
	"fmt"
	"reflect"
	"sort"
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
	listenerNetworkLoadBalancerIDValue = "ocid1.networkloadbalancer.oc1..exampleuniqueID"
	listenerNameValue                  = "example_listener"
)

type fakeListenerOCIClient struct {
	createRequests         []networkloadbalancersdk.CreateListenerRequest
	getRequests            []networkloadbalancersdk.GetListenerRequest
	listRequests           []networkloadbalancersdk.ListListenersRequest
	updateRequests         []networkloadbalancersdk.UpdateListenerRequest
	deleteRequests         []networkloadbalancersdk.DeleteListenerRequest
	getWorkRequestRequests []networkloadbalancersdk.GetWorkRequestRequest

	getErr            error
	createErr         error
	listErr           error
	updateErr         error
	deleteErr         error
	getWorkRequestErr error

	getErrs             []error
	createWorkRequestID string
	updateWorkRequestID string
	deleteWorkRequestID string
	keepAfterDelete     bool
	listPages           [][]networkloadbalancersdk.ListenerSummary
	listeners           map[string]networkloadbalancersdk.Listener
	workRequests        map[string]networkloadbalancersdk.WorkRequest
}

func (f *fakeListenerOCIClient) CreateListener(_ context.Context, request networkloadbalancersdk.CreateListenerRequest) (networkloadbalancersdk.CreateListenerResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return networkloadbalancersdk.CreateListenerResponse{}, f.createErr
	}
	f.ensureListeners()
	listener := listenerFromCreateDetails(request.CreateListenerDetails)
	f.listeners[stringValue(listener.Name)] = listener
	return networkloadbalancersdk.CreateListenerResponse{
		OpcWorkRequestId: common.String(f.createWorkRequestID),
		OpcRequestId:     common.String("opc-create-1"),
	}, nil
}

func (f *fakeListenerOCIClient) GetListener(_ context.Context, request networkloadbalancersdk.GetListenerRequest) (networkloadbalancersdk.GetListenerResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if err, ok := f.nextGetErr(); ok && err != nil {
		return networkloadbalancersdk.GetListenerResponse{}, err
	}
	listener, ok := f.listeners[stringValue(request.ListenerName)]
	if !ok {
		return networkloadbalancersdk.GetListenerResponse{}, errortest.NewServiceError(404, "NotFound", "missing listener")
	}
	return networkloadbalancersdk.GetListenerResponse{Listener: listener}, nil
}

func (f *fakeListenerOCIClient) ListListeners(_ context.Context, request networkloadbalancersdk.ListListenersRequest) (networkloadbalancersdk.ListListenersResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listErr != nil {
		return networkloadbalancersdk.ListListenersResponse{}, f.listErr
	}
	if f.listPages != nil {
		pageIndex := 0
		if page := stringValue(request.Page); page != "" {
			if _, err := fmt.Sscanf(page, "page-%d", &pageIndex); err != nil {
				return networkloadbalancersdk.ListListenersResponse{}, err
			}
		}
		if pageIndex >= len(f.listPages) {
			return networkloadbalancersdk.ListListenersResponse{}, nil
		}
		response := networkloadbalancersdk.ListListenersResponse{
			ListenerCollection: networkloadbalancersdk.ListenerCollection{Items: f.listPages[pageIndex]},
		}
		if pageIndex+1 < len(f.listPages) {
			response.OpcNextPage = common.String(fmt.Sprintf("page-%d", pageIndex+1))
		}
		return response, nil
	}

	names := make([]string, 0, len(f.listeners))
	for name := range f.listeners {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]networkloadbalancersdk.ListenerSummary, 0, len(names))
	for _, name := range names {
		items = append(items, listenerSummaryFromListener(f.listeners[name]))
	}
	return networkloadbalancersdk.ListListenersResponse{
		ListenerCollection: networkloadbalancersdk.ListenerCollection{Items: items},
	}, nil
}

func (f *fakeListenerOCIClient) UpdateListener(_ context.Context, request networkloadbalancersdk.UpdateListenerRequest) (networkloadbalancersdk.UpdateListenerResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return networkloadbalancersdk.UpdateListenerResponse{}, f.updateErr
	}
	f.ensureListeners()
	name := stringValue(request.ListenerName)
	existing := f.listeners[name]
	f.listeners[name] = listenerFromUpdateDetails(name, request.UpdateListenerDetails, existing)
	return networkloadbalancersdk.UpdateListenerResponse{
		OpcWorkRequestId: common.String(f.updateWorkRequestID),
		OpcRequestId:     common.String("opc-update-1"),
	}, nil
}

func (f *fakeListenerOCIClient) DeleteListener(_ context.Context, request networkloadbalancersdk.DeleteListenerRequest) (networkloadbalancersdk.DeleteListenerResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return networkloadbalancersdk.DeleteListenerResponse{}, f.deleteErr
	}
	if !f.keepAfterDelete {
		delete(f.listeners, stringValue(request.ListenerName))
	}
	return networkloadbalancersdk.DeleteListenerResponse{
		OpcWorkRequestId: common.String(f.deleteWorkRequestID),
		OpcRequestId:     common.String("opc-delete-1"),
	}, nil
}

func (f *fakeListenerOCIClient) GetWorkRequest(_ context.Context, request networkloadbalancersdk.GetWorkRequestRequest) (networkloadbalancersdk.GetWorkRequestResponse, error) {
	f.getWorkRequestRequests = append(f.getWorkRequestRequests, request)
	if f.getWorkRequestErr != nil {
		return networkloadbalancersdk.GetWorkRequestResponse{}, f.getWorkRequestErr
	}
	workRequest, ok := f.workRequests[stringValue(request.WorkRequestId)]
	if !ok {
		return networkloadbalancersdk.GetWorkRequestResponse{}, errortest.NewServiceError(404, "NotFound", "missing work request")
	}
	return networkloadbalancersdk.GetWorkRequestResponse{WorkRequest: workRequest}, nil
}

func (f *fakeListenerOCIClient) nextGetErr() (error, bool) {
	if len(f.getErrs) > 0 {
		err := f.getErrs[0]
		f.getErrs = f.getErrs[1:]
		return err, true
	}
	if f.getErr != nil {
		return f.getErr, true
	}
	return nil, false
}

func (f *fakeListenerOCIClient) ensureListeners() {
	if f.listeners == nil {
		f.listeners = map[string]networkloadbalancersdk.Listener{}
	}
}

func newTestListenerRuntimeClient(client *fakeListenerOCIClient) ListenerServiceClient {
	hooks := newListenerRuntimeHooksWithOCIClient(client)
	applyListenerRuntimeHooks(&hooks, client, nil)
	config := buildListenerGeneratedRuntimeConfig(&ListenerServiceManager{}, hooks)
	delegate := defaultListenerServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*networkloadbalancerv1beta1.Listener](config),
	}
	return wrapListenerGeneratedClient(hooks, delegate)
}

func TestListenerRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := listenerRuntimeSemantics()
	if got == nil {
		t.Fatal("listenerRuntimeSemantics() = nil")
	}
	if got.FormalService != "networkloadbalancer" {
		t.Fatalf("FormalService = %q, want networkloadbalancer", got.FormalService)
	}
	if got.FormalSlug != "listener" {
		t.Fatalf("FormalSlug = %q, want listener", got.FormalSlug)
	}
	if got.Async == nil || got.Async.Strategy != "workrequest" || got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async = %#v, want generatedruntime workrequest semantics", got.Async)
	}
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	assertListenerStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertListenerStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"name"})
	assertListenerStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"name", "networkLoadBalancerId"})
}

func TestListenerRequestFieldsKeepOperationsScopedToRecordedPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  []generatedruntime.RequestField
		want []generatedruntime.RequestField
	}{
		{
			name: "create",
			got:  listenerCreateFields(),
			want: []generatedruntime.RequestField{
				listenerNetworkLoadBalancerIDField(),
				{
					FieldName:    "CreateListenerDetails",
					RequestName:  "CreateListenerDetails",
					Contribution: "body",
				},
			},
		},
		{
			name: "get",
			got:  listenerGetFields(),
			want: []generatedruntime.RequestField{
				listenerNetworkLoadBalancerIDField(),
				listenerNameField(),
			},
		},
		{
			name: "list",
			got:  listenerListFields(),
			want: []generatedruntime.RequestField{
				listenerNetworkLoadBalancerIDField(),
			},
		},
		{
			name: "update",
			got:  listenerUpdateFields(),
			want: []generatedruntime.RequestField{
				listenerNetworkLoadBalancerIDField(),
				listenerNameField(),
				{
					FieldName:    "UpdateListenerDetails",
					RequestName:  "UpdateListenerDetails",
					Contribution: "body",
				},
			},
		},
		{
			name: "delete",
			got:  listenerDeleteFields(),
			want: []generatedruntime.RequestField{
				listenerNetworkLoadBalancerIDField(),
				listenerNameField(),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !reflect.DeepEqual(tc.got, tc.want) {
				t.Fatalf("%s fields = %#v, want %#v", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestCreateOrUpdateRejectsMissingListenerNetworkLoadBalancerAnnotation(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedListenerResource()
	resource.Annotations = nil
	client := &fakeListenerOCIClient{}

	_, err := newTestListenerRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want missing parent annotation error")
	}
	if !strings.Contains(err.Error(), listenerNetworkLoadBalancerIDAnnotation) {
		t.Fatalf("CreateOrUpdate() error = %v, want %s", err, listenerNetworkLoadBalancerIDAnnotation)
	}
	assertListenerNoOCICalls(t, client)
}

func TestCreateOrUpdateBindsExistingListener(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedListenerResource()
	client := &fakeListenerOCIClient{
		listeners: map[string]networkloadbalancersdk.Listener{
			listenerNameValue: sdkListenerFromSpec(resource.Spec),
		},
	}

	response, err := newTestListenerRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
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
		t.Fatalf("create requests = %d, want 0 for bind path", len(client.createRequests))
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 for no-drift bind path", len(client.updateRequests))
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1 for bind path", len(client.getRequests))
	}
	assertListenerPathIdentity(t, client.getRequests[0].NetworkLoadBalancerId, client.getRequests[0].ListenerName, listenerNetworkLoadBalancerIDValue, listenerNameValue)
	assertListenerTrackedStatus(t, resource, listenerNetworkLoadBalancerIDValue, listenerNameValue)
}

func TestCreateOrUpdateCreatesThenObservesListenerWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeUntrackedListenerResource()
	client := &fakeListenerOCIClient{
		createWorkRequestID: "wr-create-1",
		workRequests: map[string]networkloadbalancersdk.WorkRequest{
			"wr-create-1": listenerWorkRequest("wr-create-1", networkloadbalancersdk.OperationTypeCreateListener, networkloadbalancersdk.OperationStatusAccepted),
		},
	}
	serviceClient := newTestListenerRuntimeClient(client)

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertListenerPendingCreate(t, response, resource, client)

	client.workRequests["wr-create-1"] = listenerWorkRequest("wr-create-1", networkloadbalancersdk.OperationTypeCreateListener, networkloadbalancersdk.OperationStatusSucceeded)
	response, err = serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("observe CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("observe CreateOrUpdate() response = %#v, want successful observe response", response)
	}
	if response.ShouldRequeue {
		t.Fatal("observe CreateOrUpdate() ShouldRequeue = true, want active observation")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests after observe = %d, want 1", len(client.createRequests))
	}
	if len(client.getRequests) == 0 {
		t.Fatal("get requests after work request success = 0, want readback")
	}
	assertListenerTrackedStatus(t, resource, listenerNetworkLoadBalancerIDValue, listenerNameValue)
}

func TestCreateOrUpdateUpdatesMutableListenerFields(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListenerResource()
	resource.Spec.DefaultBackendSetName = "backend_set_b"
	resource.Spec.Port = 443
	resource.Spec.IsPpv2Enabled = false
	client := &fakeListenerOCIClient{
		updateWorkRequestID: "wr-update-1",
		listeners: map[string]networkloadbalancersdk.Listener{
			listenerNameValue: {
				Name:                  common.String(listenerNameValue),
				DefaultBackendSetName: common.String("backend_set_a"),
				Port:                  common.Int(80),
				Protocol:              networkloadbalancersdk.ListenerProtocolsTcp,
				IsPpv2Enabled:         common.Bool(true),
			},
		},
		workRequests: map[string]networkloadbalancersdk.WorkRequest{
			"wr-update-1": listenerWorkRequest("wr-update-1", networkloadbalancersdk.OperationTypeUpdateListener, networkloadbalancersdk.OperationStatusSucceeded),
		},
	}

	response, err := newTestListenerRuntimeClient(client).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful update response", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}
	update := client.updateRequests[0]
	assertListenerPathIdentity(t, update.NetworkLoadBalancerId, update.ListenerName, listenerNetworkLoadBalancerIDValue, listenerNameValue)
	if got := stringValue(update.DefaultBackendSetName); got != "backend_set_b" {
		t.Fatalf("UpdateListenerDetails.DefaultBackendSetName = %q, want backend_set_b", got)
	}
	if got := intValue(update.Port); got != 443 {
		t.Fatalf("UpdateListenerDetails.Port = %d, want 443", got)
	}
	if update.IsPpv2Enabled == nil || *update.IsPpv2Enabled {
		t.Fatalf("UpdateListenerDetails.IsPpv2Enabled = %#v, want explicit false", update.IsPpv2Enabled)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-update-1", got)
	}
	assertListenerTrackedStatus(t, resource, listenerNetworkLoadBalancerIDValue, listenerNameValue)
}

func TestCreateOrUpdateRejectsListenerIdentityDriftBeforeOCI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource *networkloadbalancerv1beta1.Listener
		wantErr  string
	}{
		{
			name: "listener name drift",
			resource: func() *networkloadbalancerv1beta1.Listener {
				resource := makeTrackedListenerResource()
				resource.Spec.Name = "replacement_listener"
				return resource
			}(),
			wantErr: "require replacement when name changes",
		},
		{
			name: "parent annotation drift",
			resource: func() *networkloadbalancerv1beta1.Listener {
				resource := makeTrackedListenerResource()
				resource.Annotations[listenerNetworkLoadBalancerIDAnnotation] = "ocid1.networkloadbalancer.oc1..replacement"
				return resource
			}(),
			wantErr: "changed from recorded networkLoadBalancerId",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeListenerOCIClient{}
			_, err := newTestListenerRuntimeClient(client).CreateOrUpdate(context.Background(), tc.resource, ctrl.Request{})
			if err == nil {
				t.Fatal("CreateOrUpdate() error = nil, want identity drift error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("CreateOrUpdate() error = %v, want containing %q", err, tc.wantErr)
			}
			assertListenerNoOCICalls(t, client)
		})
	}
}

func TestDeleteConfirmsListenerRemovalAfterWorkRequest(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListenerResource()
	client := &fakeListenerOCIClient{
		deleteWorkRequestID: "wr-delete-1",
		listeners: map[string]networkloadbalancersdk.Listener{
			listenerNameValue: sdkListenerFromSpec(resource.Spec),
		},
		workRequests: map[string]networkloadbalancersdk.WorkRequest{
			"wr-delete-1": listenerWorkRequest("wr-delete-1", networkloadbalancersdk.OperationTypeDeleteListener, networkloadbalancersdk.OperationStatusSucceeded),
		},
	}

	deleted, err := newTestListenerRuntimeClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want confirmed delete")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	assertListenerPathIdentity(t, client.deleteRequests[0].NetworkLoadBalancerId, client.deleteRequests[0].ListenerName, listenerNetworkLoadBalancerIDValue, listenerNameValue)
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-1", got)
	}
}

func TestDeleteWorkRequestSuccessWithTargetNotFoundIgnoresSiblingListener(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListenerResource()
	siblingName := "sibling_listener"
	client := &fakeListenerOCIClient{
		deleteWorkRequestID: "wr-delete-1",
		listeners: map[string]networkloadbalancersdk.Listener{
			listenerNameValue: sdkListenerFromSpec(resource.Spec),
			siblingName: {
				Name:                  common.String(siblingName),
				DefaultBackendSetName: common.String("backend_set_b"),
				Port:                  common.Int(8080),
				Protocol:              networkloadbalancersdk.ListenerProtocolsTcp,
			},
		},
		workRequests: map[string]networkloadbalancersdk.WorkRequest{
			"wr-delete-1": listenerWorkRequest("wr-delete-1", networkloadbalancersdk.OperationTypeDeleteListener, networkloadbalancersdk.OperationStatusSucceeded),
		},
	}

	deleted, err := newTestListenerRuntimeClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v; get requests = %d, list requests = %d, delete requests = %d", err, len(client.getRequests), len(client.listRequests), len(client.deleteRequests))
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want confirmed delete when target GetListener returns NotFound")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	if _, ok := client.listeners[siblingName]; !ok {
		t.Fatalf("sibling listener %q was removed, want only target listener deleted", siblingName)
	}
	if got := resource.Status.Name; got != listenerNameValue {
		t.Fatalf("status.name = %q, want target listener %q", got, listenerNameValue)
	}
	for i, request := range client.getRequests {
		if got := stringValue(request.ListenerName); got != listenerNameValue {
			t.Fatalf("get request %d listenerName = %q, want %q", i, got, listenerNameValue)
		}
		if got := stringValue(request.NetworkLoadBalancerId); got != listenerNetworkLoadBalancerIDValue {
			t.Fatalf("get request %d networkLoadBalancerId = %q, want %q", i, got, listenerNetworkLoadBalancerIDValue)
		}
	}
}

func TestDeleteWithPendingCreateWorkRequestRetainsFinalizer(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListenerResource()
	trackListenerWorkRequest(resource, shared.OSOKAsyncPhaseCreate, "wr-create-1", networkloadbalancersdk.OperationTypeCreateListener, networkloadbalancersdk.OperationStatusAccepted)
	client := &fakeListenerOCIClient{
		workRequests: map[string]networkloadbalancersdk.WorkRequest{
			"wr-create-1": listenerWorkRequest("wr-create-1", networkloadbalancersdk.OperationTypeCreateListener, networkloadbalancersdk.OperationStatusInProgress),
		},
	}

	deleted, err := newTestListenerRuntimeClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while create work request is pending")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want none while create work request is pending", len(client.deleteRequests))
	}
	if len(client.getRequests) != 0 {
		t.Fatalf("get requests = %d, want no delete confirmation read while create work request is pending", len(client.getRequests))
	}
	if len(client.getWorkRequestRequests) != 1 {
		t.Fatalf("get work request calls = %d, want 1", len(client.getWorkRequestRequests))
	}
	requireListenerAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-1", shared.OSOKAsyncClassPending)
	assertListenerTrailingCondition(t, resource, shared.Provisioning)
}

func TestDeleteWithPendingUpdateWorkRequestRetainsFinalizer(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListenerResource()
	trackListenerWorkRequest(resource, shared.OSOKAsyncPhaseUpdate, "wr-update-1", networkloadbalancersdk.OperationTypeUpdateListener, networkloadbalancersdk.OperationStatusAccepted)
	client := &fakeListenerOCIClient{
		workRequests: map[string]networkloadbalancersdk.WorkRequest{
			"wr-update-1": listenerWorkRequest("wr-update-1", networkloadbalancersdk.OperationTypeUpdateListener, networkloadbalancersdk.OperationStatusInProgress),
		},
	}

	deleted, err := newTestListenerRuntimeClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while update work request is pending")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want none while update work request is pending", len(client.deleteRequests))
	}
	if len(client.getRequests) != 0 {
		t.Fatalf("get requests = %d, want no delete confirmation read while update work request is pending", len(client.getRequests))
	}
	if len(client.getWorkRequestRequests) != 1 {
		t.Fatalf("get work request calls = %d, want 1", len(client.getWorkRequestRequests))
	}
	requireListenerAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update-1", shared.OSOKAsyncClassPending)
	assertListenerTrailingCondition(t, resource, shared.Updating)
}

func TestDeleteAuthShaped404FromPreDeleteReadIsFatal(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListenerResource()
	client := &fakeListenerOCIClient{
		getErr: listenerAuthShaped404(),
	}

	deleted, err := newTestListenerRuntimeClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped 404 to be fatal")
	}
	if !strings.Contains(err.Error(), errorutil.NotAuthorizedOrNotFound) {
		t.Fatalf("Delete() error = %v, want %s", err, errorutil.NotAuthorizedOrNotFound)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained for auth-shaped 404")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want none after auth-shaped pre-delete read", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %#v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	assertListenerTrailingCondition(t, resource, shared.Failed)
}

func TestDeleteAuthShaped404FromDeleteIsFatal(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListenerResource()
	client := &fakeListenerOCIClient{
		deleteErr: listenerAuthShaped404(),
		listeners: map[string]networkloadbalancersdk.Listener{
			listenerNameValue: sdkListenerFromSpec(resource.Spec),
		},
	}

	deleted, err := newTestListenerRuntimeClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped 404 to be fatal")
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		t.Fatalf("Delete() error still classifies as delete-not-found: %v", err)
	}
	if !strings.Contains(err.Error(), errorutil.NotAuthorizedOrNotFound) {
		t.Fatalf("Delete() error = %v, want %s", err, errorutil.NotAuthorizedOrNotFound)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained for auth-shaped delete error")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %#v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	assertListenerTrailingCondition(t, resource, shared.Failed)
}

func TestDeleteAuthShaped404FromDeleteConfirmationIsFatal(t *testing.T) {
	t.Parallel()

	resource := makeTrackedListenerResource()
	client := &fakeListenerOCIClient{
		getErrs:             []error{nil, nil, listenerAuthShaped404()},
		deleteWorkRequestID: "wr-delete-1",
		listeners: map[string]networkloadbalancersdk.Listener{
			listenerNameValue: sdkListenerFromSpec(resource.Spec),
		},
		workRequests: map[string]networkloadbalancersdk.WorkRequest{
			"wr-delete-1": listenerWorkRequest("wr-delete-1", networkloadbalancersdk.OperationTypeDeleteListener, networkloadbalancersdk.OperationStatusSucceeded),
		},
	}

	deleted, err := newTestListenerRuntimeClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped confirmation 404 to be fatal")
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		t.Fatalf("Delete() error still classifies as delete-not-found: %v", err)
	}
	if !strings.Contains(err.Error(), errorutil.NotAuthorizedOrNotFound) {
		t.Fatalf("Delete() error = %v, want %s", err, errorutil.NotAuthorizedOrNotFound)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained for auth-shaped confirmation error")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %#v, want nil", resource.Status.OsokStatus.DeletedAt)
	}
	requireListenerAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete-1", shared.OSOKAsyncClassFailed)
	assertListenerTrailingCondition(t, resource, shared.Failed)
}

func TestListListenerRuntimeViewsPaginatesAllPages(t *testing.T) {
	t.Parallel()

	client := &fakeListenerOCIClient{
		listPages: [][]networkloadbalancersdk.ListenerSummary{
			{
				{Name: common.String("first"), DefaultBackendSetName: common.String("backend_set_a"), Port: common.Int(80), Protocol: networkloadbalancersdk.ListenerProtocolsTcp},
			},
			{
				{Name: common.String(listenerNameValue), DefaultBackendSetName: common.String("backend_set_b"), Port: common.Int(443), Protocol: networkloadbalancersdk.ListenerProtocolsTcp},
			},
		},
	}

	got, err := listListenerRuntimeViews(context.Background(), client, networkloadbalancersdk.ListListenersRequest{
		NetworkLoadBalancerId: common.String(listenerNetworkLoadBalancerIDValue),
	})
	if err != nil {
		t.Fatalf("listListenerRuntimeViews() error = %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("listListenerRuntimeViews() items = %d, want 2", len(got.Items))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2", len(client.listRequests))
	}
	if gotPage := stringValue(client.listRequests[1].Page); gotPage != "page-1" {
		t.Fatalf("second list page = %q, want page-1", gotPage)
	}
	if gotName := stringValue(got.Items[1].Name); gotName != listenerNameValue {
		t.Fatalf("second item name = %q, want %q", gotName, listenerNameValue)
	}
	if gotParent := got.Items[1].NetworkLoadBalancerId; gotParent != listenerNetworkLoadBalancerIDValue {
		t.Fatalf("second item networkLoadBalancerId = %q, want %q", gotParent, listenerNetworkLoadBalancerIDValue)
	}
	if gotOCID := got.Items[1].Ocid; gotOCID != "" {
		t.Fatalf("second item ocid = %q, want empty because list summaries do not expose a child listener OCID", gotOCID)
	}
}

func makeUntrackedListenerResource() *networkloadbalancerv1beta1.Listener {
	return &networkloadbalancerv1beta1.Listener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      listenerNameValue,
			Namespace: "default",
			Annotations: map[string]string{
				listenerNetworkLoadBalancerIDAnnotation: listenerNetworkLoadBalancerIDValue,
			},
		},
		Spec: networkloadbalancerv1beta1.ListenerSpec{
			Name:                  listenerNameValue,
			DefaultBackendSetName: "backend_set_a",
			Port:                  80,
			Protocol:              string(networkloadbalancersdk.ListenerProtocolsTcp),
		},
	}
}

func makeTrackedListenerResource() *networkloadbalancerv1beta1.Listener {
	resource := makeUntrackedListenerResource()
	resource.Status.Name = listenerNameValue
	resource.Status.OsokStatus.Ocid = shared.OCID(listenerNetworkLoadBalancerIDValue)
	return resource
}

func sdkListenerFromSpec(spec networkloadbalancerv1beta1.ListenerSpec) networkloadbalancersdk.Listener {
	return networkloadbalancersdk.Listener{
		Name:                  common.String(spec.Name),
		DefaultBackendSetName: common.String(spec.DefaultBackendSetName),
		Port:                  common.Int(spec.Port),
		Protocol:              networkloadbalancersdk.ListenerProtocolsEnum(spec.Protocol),
		IpVersion:             networkloadbalancersdk.IpVersionEnum(spec.IpVersion),
		IsPpv2Enabled:         common.Bool(spec.IsPpv2Enabled),
		TcpIdleTimeout:        intPointer(spec.TcpIdleTimeout),
		UdpIdleTimeout:        intPointer(spec.UdpIdleTimeout),
		L3IpIdleTimeout:       intPointer(spec.L3IpIdleTimeout),
	}
}

func listenerFromCreateDetails(details networkloadbalancersdk.CreateListenerDetails) networkloadbalancersdk.Listener {
	return networkloadbalancersdk.Listener(details)
}

func listenerFromUpdateDetails(name string, details networkloadbalancersdk.UpdateListenerDetails, existing networkloadbalancersdk.Listener) networkloadbalancersdk.Listener {
	listener := existing
	if listener.Name == nil {
		listener.Name = common.String(name)
	}
	if details.DefaultBackendSetName != nil {
		listener.DefaultBackendSetName = details.DefaultBackendSetName
	}
	if details.Port != nil {
		listener.Port = details.Port
	}
	if details.Protocol != "" {
		listener.Protocol = details.Protocol
	}
	if details.IpVersion != "" {
		listener.IpVersion = details.IpVersion
	}
	if details.IsPpv2Enabled != nil {
		listener.IsPpv2Enabled = details.IsPpv2Enabled
	}
	if details.TcpIdleTimeout != nil {
		listener.TcpIdleTimeout = details.TcpIdleTimeout
	}
	if details.UdpIdleTimeout != nil {
		listener.UdpIdleTimeout = details.UdpIdleTimeout
	}
	if details.L3IpIdleTimeout != nil {
		listener.L3IpIdleTimeout = details.L3IpIdleTimeout
	}
	return listener
}

func listenerSummaryFromListener(listener networkloadbalancersdk.Listener) networkloadbalancersdk.ListenerSummary {
	return networkloadbalancersdk.ListenerSummary(listener)
}

func listenerWorkRequest(
	id string,
	operation networkloadbalancersdk.OperationTypeEnum,
	status networkloadbalancersdk.OperationStatusEnum,
) networkloadbalancersdk.WorkRequest {
	return networkloadbalancersdk.WorkRequest{
		Id:              common.String(id),
		OperationType:   operation,
		Status:          status,
		PercentComplete: common.Float32(100),
	}
}

func listenerAuthShaped404() error {
	return errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
}

func trackListenerWorkRequest(
	resource *networkloadbalancerv1beta1.Listener,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	operation networkloadbalancersdk.OperationTypeEnum,
	status networkloadbalancersdk.OperationStatusEnum,
) {
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            phase,
		WorkRequestID:    workRequestID,
		RawStatus:        string(status),
		RawOperationType: string(operation),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		UpdatedAt:        &metav1.Time{},
	}
}

func assertListenerPathIdentity(t *testing.T, gotParent *string, gotName *string, wantParent string, wantName string) {
	t.Helper()
	if stringValue(gotParent) != wantParent {
		t.Fatalf("networkLoadBalancerId = %q, want %q", stringValue(gotParent), wantParent)
	}
	if stringValue(gotName) != wantName {
		t.Fatalf("listenerName = %q, want %q", stringValue(gotName), wantName)
	}
}

func assertListenerPendingCreate(
	t *testing.T,
	response servicemanager.OSOKResponse,
	resource *networkloadbalancerv1beta1.Listener,
	client *fakeListenerOCIClient,
) {
	t.Helper()
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful pending create response", response)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(client.createRequests))
	}
	create := client.createRequests[0]
	assertListenerPathIdentity(t, create.NetworkLoadBalancerId, create.Name, listenerNetworkLoadBalancerIDValue, listenerNameValue)
	if got := stringValue(create.DefaultBackendSetName); got != resource.Spec.DefaultBackendSetName {
		t.Fatalf("CreateListenerDetails.DefaultBackendSetName = %q, want %q", got, resource.Spec.DefaultBackendSetName)
	}
	requireListenerAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-1", shared.OSOKAsyncClassPending)
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-create-1", got)
	}
}

func assertListenerTrackedStatus(t *testing.T, resource *networkloadbalancerv1beta1.Listener, wantParent string, wantName string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != wantParent {
		t.Fatalf("status.status.ocid = %q, want %q", got, wantParent)
	}
	if got := resource.Status.Name; got != wantName {
		t.Fatalf("status.name = %q, want %q", got, wantName)
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Active) {
		t.Fatalf("status.status.reason = %q, want %q", got, shared.Active)
	}
}

func requireListenerAsync(
	t *testing.T,
	resource *networkloadbalancerv1beta1.Listener,
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

func assertListenerTrailingCondition(
	t *testing.T,
	resource *networkloadbalancerv1beta1.Listener,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions = nil, want trailing %s condition", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("trailing condition = %q, want %q; conditions = %#v", got, want, conditions)
	}
}

func assertListenerNoOCICalls(t *testing.T, client *fakeListenerOCIClient) {
	t.Helper()
	if got := len(client.createRequests) + len(client.getRequests) + len(client.listRequests) + len(client.updateRequests) + len(client.deleteRequests) + len(client.getWorkRequestRequests); got != 0 {
		t.Fatalf("OCI calls = %d, want 0", got)
	}
}

func assertListenerStringSliceEqual(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}
