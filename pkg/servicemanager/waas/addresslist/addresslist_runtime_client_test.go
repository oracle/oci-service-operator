/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package addresslist

import (
	"context"
	"maps"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testAddressListCompartmentID = "ocid1.compartment.oc1..addresslist"
	testAddressListID            = "ocid1.waasaddresslist.oc1..addresslist"
	testAddressListName          = "address-list-alpha"
)

type fakeAddressListOCIClient struct {
	createFunc            func(context.Context, waassdk.CreateAddressListRequest) (waassdk.CreateAddressListResponse, error)
	getFunc               func(context.Context, waassdk.GetAddressListRequest) (waassdk.GetAddressListResponse, error)
	listFunc              func(context.Context, waassdk.ListAddressListsRequest) (waassdk.ListAddressListsResponse, error)
	updateFunc            func(context.Context, waassdk.UpdateAddressListRequest) (waassdk.UpdateAddressListResponse, error)
	deleteFunc            func(context.Context, waassdk.DeleteAddressListRequest) (waassdk.DeleteAddressListResponse, error)
	changeCompartmentFunc func(context.Context, waassdk.ChangeAddressListCompartmentRequest) (waassdk.ChangeAddressListCompartmentResponse, error)

	createRequests            []waassdk.CreateAddressListRequest
	getRequests               []waassdk.GetAddressListRequest
	listRequests              []waassdk.ListAddressListsRequest
	updateRequests            []waassdk.UpdateAddressListRequest
	deleteRequests            []waassdk.DeleteAddressListRequest
	changeCompartmentRequests []waassdk.ChangeAddressListCompartmentRequest
}

func (f *fakeAddressListOCIClient) CreateAddressList(
	ctx context.Context,
	request waassdk.CreateAddressListRequest,
) (waassdk.CreateAddressListResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return waassdk.CreateAddressListResponse{}, nil
}

func (f *fakeAddressListOCIClient) GetAddressList(
	ctx context.Context,
	request waassdk.GetAddressListRequest,
) (waassdk.GetAddressListResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return waassdk.GetAddressListResponse{}, nil
}

func (f *fakeAddressListOCIClient) ListAddressLists(
	ctx context.Context,
	request waassdk.ListAddressListsRequest,
) (waassdk.ListAddressListsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return waassdk.ListAddressListsResponse{}, nil
}

func (f *fakeAddressListOCIClient) UpdateAddressList(
	ctx context.Context,
	request waassdk.UpdateAddressListRequest,
) (waassdk.UpdateAddressListResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return waassdk.UpdateAddressListResponse{}, nil
}

func (f *fakeAddressListOCIClient) DeleteAddressList(
	ctx context.Context,
	request waassdk.DeleteAddressListRequest,
) (waassdk.DeleteAddressListResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return waassdk.DeleteAddressListResponse{}, nil
}

func (f *fakeAddressListOCIClient) ChangeAddressListCompartment(
	ctx context.Context,
	request waassdk.ChangeAddressListCompartmentRequest,
) (waassdk.ChangeAddressListCompartmentResponse, error) {
	f.changeCompartmentRequests = append(f.changeCompartmentRequests, request)
	if f.changeCompartmentFunc != nil {
		return f.changeCompartmentFunc(ctx, request)
	}
	return waassdk.ChangeAddressListCompartmentResponse{}, nil
}

func TestAddressListRuntimeHooksConfigured(t *testing.T) {
	t.Parallel()

	hooks := newAddressListDefaultRuntimeHooks(waassdk.WaasClient{})
	applyAddressListRuntimeHooks(&hooks, &fakeAddressListOCIClient{}, nil)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if hooks.Semantics.List == nil || !reflect.DeepEqual(hooks.Semantics.List.MatchFields, []string{"compartmentId", "displayName", "id"}) {
		t.Fatalf("hooks.Semantics.List = %#v, want compartment/displayName/id matching", hooks.Semantics.List)
	}
	if hooks.BuildCreateBody == nil {
		t.Fatal("hooks.BuildCreateBody = nil, want reviewed create body")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed update body")
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("hooks.Identity.GuardExistingBeforeCreate = nil, want pre-create guard")
	}
	if hooks.Read.List == nil {
		t.Fatal("hooks.Read.List = nil, want paginated list read")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("hooks.DeleteHooks.HandleError = nil, want conservative delete errors")
	}
	if len(hooks.WrapGeneratedClient) == 0 {
		t.Fatal("hooks.WrapGeneratedClient is empty, want delete confirmation guard")
	}
}

func TestAddressListRuntimeCompartmentMoveHooksConfigured(t *testing.T) {
	t.Parallel()

	hooks := newAddressListDefaultRuntimeHooks(waassdk.WaasClient{})
	applyAddressListRuntimeHooks(&hooks, &fakeAddressListOCIClient{}, nil)

	if hooks.ParityHooks.RequiresParityHandling == nil {
		t.Fatal("hooks.ParityHooks.RequiresParityHandling = nil, want compartment move routing")
	}
	if hooks.ParityHooks.ApplyParityUpdate == nil {
		t.Fatal("hooks.ParityHooks.ApplyParityUpdate = nil, want compartment move implementation")
	}
}

func TestAddressListCreateOrUpdateCreatesAndRefreshesStatus(t *testing.T) {
	t.Parallel()

	fake := &fakeAddressListOCIClient{}
	fake.listFunc = func(_ context.Context, request waassdk.ListAddressListsRequest) (waassdk.ListAddressListsResponse, error) {
		requireAddressListListIdentity(t, request)
		return waassdk.ListAddressListsResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request waassdk.CreateAddressListRequest) (waassdk.CreateAddressListResponse, error) {
		requireAddressListCreateRequest(t, request, newAddressListRuntimeTestResource().Spec)
		return waassdk.CreateAddressListResponse{
			OpcRequestId: common.String("opc-create-1"),
			AddressList:  addressListFromSpec(testAddressListID, newAddressListRuntimeTestResource().Spec, waassdk.LifecycleStatesCreating),
		}, nil
	}
	fake.getFunc = func(_ context.Context, request waassdk.GetAddressListRequest) (waassdk.GetAddressListResponse, error) {
		if got := addressListStringValue(request.AddressListId); got != testAddressListID {
			t.Fatalf("get addressListId = %q, want %q", got, testAddressListID)
		}
		return waassdk.GetAddressListResponse{
			AddressList: addressListFromSpec(testAddressListID, newAddressListRuntimeTestResource().Spec, waassdk.LifecycleStatesActive),
		}, nil
	}
	client := newAddressListRuntimeTestClient(fake)

	resource := newAddressListRuntimeTestResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response successful=%t shouldRequeue=%t, want successful non-requeue", response.IsSuccessful, response.ShouldRequeue)
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("ListAddressLists() calls = %d, want 1 pre-create lookup", len(fake.listRequests))
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateAddressList() calls = %d, want 1", len(fake.createRequests))
	}
	requireAddressListActiveStatus(t, resource, "opc-create-1")
}

func TestAddressListCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	fake := &fakeAddressListOCIClient{}
	fake.listFunc = addressListExistingOnSecondPage(t, fake)
	fake.getFunc = addressListGetResponsesByID(t, fake,
		addressListFromSpec(testAddressListID, newAddressListRuntimeTestResource().Spec, waassdk.LifecycleStatesActive),
	)
	fake.createFunc = failAddressListCreate(t, "CreateAddressList() called; want existing AddressList bind")
	client := newAddressListRuntimeTestClient(fake)

	resource := newAddressListRuntimeTestResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListAddressLists() calls = %d, want 2 paginated calls", len(fake.listRequests))
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateAddressList() calls = %d, want 0", len(fake.createRequests))
	}
	requireAddressListBoundStatus(t, resource)
}

func TestAddressListCreateOrUpdateSkipsUpdateWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	fake := &fakeAddressListOCIClient{}
	fake.getFunc = func(context.Context, waassdk.GetAddressListRequest) (waassdk.GetAddressListResponse, error) {
		return waassdk.GetAddressListResponse{
			AddressList: addressListFromSpec(testAddressListID, newAddressListRuntimeTestResource().Spec, waassdk.LifecycleStatesActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, waassdk.UpdateAddressListRequest) (waassdk.UpdateAddressListResponse, error) {
		t.Fatal("UpdateAddressList() called; want no-op reconcile")
		return waassdk.UpdateAddressListResponse{}, nil
	}
	client := newAddressListRuntimeTestClient(fake)

	resource := newAddressListRuntimeTestResource()
	trackAddressListID(resource, testAddressListID)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateAddressList() calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestAddressListCreateOrUpdateUpdatesMutableFieldsAndClearsExplicitTags(t *testing.T) {
	t.Parallel()

	desired := newAddressListRuntimeTestResource()
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}

	fake := &fakeAddressListOCIClient{}
	fake.getFunc = addressListGetResponsesByID(t, fake,
		addressListFromSpec(testAddressListID, staleAddressListSpec(), waassdk.LifecycleStatesActive),
		addressListFromSpec(testAddressListID, desired.Spec, waassdk.LifecycleStatesActive),
	)
	fake.updateFunc = addressListMutableUpdateFunc(t, desired.Spec)
	client := newAddressListRuntimeTestClient(fake)

	trackAddressListID(desired, testAddressListID)
	response, err := client.CreateOrUpdate(context.Background(), desired, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if got := len(fake.updateRequests); got != 1 {
		t.Fatalf("UpdateAddressList() calls = %d, want 1", got)
	}
	requireAddressListUpdatedStatus(t, desired)
}

func TestAddressListCreateOrUpdateMovesCompartmentDrift(t *testing.T) {
	t.Parallel()

	resource := newAddressListRuntimeTestResource()
	trackAddressListID(resource, testAddressListID)
	currentSpec := resource.Spec
	currentSpec.CompartmentId = "ocid1.compartment.oc1..observed"

	fake := &fakeAddressListOCIClient{}
	fake.getFunc = addressListGetResponsesByID(t, fake,
		addressListFromSpec(testAddressListID, currentSpec, waassdk.LifecycleStatesActive),
		addressListFromSpec(testAddressListID, resource.Spec, waassdk.LifecycleStatesActive),
	)
	fake.changeCompartmentFunc = func(_ context.Context, request waassdk.ChangeAddressListCompartmentRequest) (waassdk.ChangeAddressListCompartmentResponse, error) {
		requireAddressListMoveRequest(t, request, resource.Spec.CompartmentId)
		return waassdk.ChangeAddressListCompartmentResponse{
			OpcRequestId: common.String("opc-move-1"),
		}, nil
	}
	fake.updateFunc = func(context.Context, waassdk.UpdateAddressListRequest) (waassdk.UpdateAddressListResponse, error) {
		t.Fatal("UpdateAddressList() called; want compartment move")
		return waassdk.UpdateAddressListResponse{}, nil
	}
	client := newAddressListRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response successful=%t shouldRequeue=%t, want successful non-requeue", response.IsSuccessful, response.ShouldRequeue)
	}
	if got := len(fake.changeCompartmentRequests); got != 1 {
		t.Fatalf("ChangeAddressListCompartment() calls = %d, want 1", got)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateAddressList() calls = %d, want 0", len(fake.updateRequests))
	}
	if got := len(fake.getRequests); got != 2 {
		t.Fatalf("GetAddressList() calls = %d, want current read and post-move refresh", got)
	}
	if got := resource.Status.CompartmentId; got != resource.Spec.CompartmentId {
		t.Fatalf("status.compartmentId = %q, want moved compartment %q", got, resource.Spec.CompartmentId)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-move-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-move-1", got)
	}
}

func TestAddressListCreateOrUpdateMovesCompartmentThenUpdatesMutableDrift(t *testing.T) {
	t.Parallel()

	resource := newAddressListRuntimeTestResource()
	trackAddressListID(resource, testAddressListID)

	currentSpec := staleAddressListSpec()
	currentSpec.CompartmentId = "ocid1.compartment.oc1..observed"
	postMoveSpec := currentSpec
	postMoveSpec.CompartmentId = resource.Spec.CompartmentId

	fake := &fakeAddressListOCIClient{}
	fake.getFunc = addressListGetResponsesByID(t, fake,
		addressListFromSpec(testAddressListID, currentSpec, waassdk.LifecycleStatesActive),
		addressListFromSpec(testAddressListID, postMoveSpec, waassdk.LifecycleStatesActive),
		addressListFromSpec(testAddressListID, resource.Spec, waassdk.LifecycleStatesActive),
	)
	fake.changeCompartmentFunc = func(_ context.Context, request waassdk.ChangeAddressListCompartmentRequest) (waassdk.ChangeAddressListCompartmentResponse, error) {
		requireAddressListMoveRequest(t, request, resource.Spec.CompartmentId)
		return waassdk.ChangeAddressListCompartmentResponse{
			OpcRequestId: common.String("opc-move-1"),
		}, nil
	}
	fake.updateFunc = addressListMutableUpdateFunc(t, resource.Spec)
	client := newAddressListRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireAddressListMoveThenUpdateResult(t, fake, resource, response)
}

func TestAddressListCompartmentMoveRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(500, errorutil.InternalServerError, "move failed")
	serviceErr.OpcRequestID = "opc-move-error-1"

	resource := newAddressListRuntimeTestResource()
	trackAddressListID(resource, testAddressListID)
	currentSpec := resource.Spec
	currentSpec.CompartmentId = "ocid1.compartment.oc1..observed"

	fake := &fakeAddressListOCIClient{}
	fake.getFunc = addressListGetResponsesByID(t, fake,
		addressListFromSpec(testAddressListID, currentSpec, waassdk.LifecycleStatesActive),
	)
	fake.changeCompartmentFunc = func(_ context.Context, request waassdk.ChangeAddressListCompartmentRequest) (waassdk.ChangeAddressListCompartmentResponse, error) {
		requireAddressListMoveRequest(t, request, resource.Spec.CompartmentId)
		return waassdk.ChangeAddressListCompartmentResponse{}, serviceErr
	}
	fake.updateFunc = func(context.Context, waassdk.UpdateAddressListRequest) (waassdk.UpdateAddressListResponse, error) {
		t.Fatal("UpdateAddressList() called; want compartment move error")
		return waassdk.UpdateAddressListResponse{}, nil
	}
	client := newAddressListRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want compartment move error")
	}
	if !strings.Contains(err.Error(), "move failed") {
		t.Fatalf("CreateOrUpdate() error = %v, want surfaced compartment move error", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success")
	}
	if got := len(fake.changeCompartmentRequests); got != 1 {
		t.Fatalf("ChangeAddressListCompartment() calls = %d, want 1", got)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateAddressList() calls = %d, want 0", len(fake.updateRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-move-error-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-move-error-1", got)
	}
}

func TestAddressListDeleteRetainsFinalizerUntilLifecycleDeleteConfirmed(t *testing.T) {
	t.Parallel()

	fake := &fakeAddressListOCIClient{}
	fake.getFunc = addressListGetLifecycleSequence(t, fake,
		waassdk.LifecycleStatesActive,
		waassdk.LifecycleStatesActive,
		waassdk.LifecycleStatesDeleting,
		waassdk.LifecycleStatesDeleted,
		waassdk.LifecycleStatesDeleted,
	)
	fake.deleteFunc = func(_ context.Context, request waassdk.DeleteAddressListRequest) (waassdk.DeleteAddressListResponse, error) {
		requireAddressListDeleteID(t, request)
		return waassdk.DeleteAddressListResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	client := newAddressListRuntimeTestClient(fake)

	resource := newAddressListRuntimeTestResource()
	trackAddressListID(resource, testAddressListID)
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first call deleted = true, want finalizer retained while DELETING")
	}
	requireAddressListDeletePendingStatus(t, fake, resource)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second call deleted = false, want finalizer release after DELETED")
	}
	requireAddressListDeleteConfirmedStatus(t, fake, resource)
}

func TestAddressListDeleteRejectsAuthShapedNotFound(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-delete-error-1"

	fake := &fakeAddressListOCIClient{}
	fake.getFunc = func(context.Context, waassdk.GetAddressListRequest) (waassdk.GetAddressListResponse, error) {
		return waassdk.GetAddressListResponse{
			AddressList: addressListFromSpec(testAddressListID, newAddressListRuntimeTestResource().Spec, waassdk.LifecycleStatesActive),
		}, nil
	}
	fake.deleteFunc = func(context.Context, waassdk.DeleteAddressListRequest) (waassdk.DeleteAddressListResponse, error) {
		return waassdk.DeleteAddressListResponse{}, serviceErr
	}
	client := newAddressListRuntimeTestClient(fake)

	resource := newAddressListRuntimeTestResource()
	trackAddressListID(resource, testAddressListID)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped 404 to stay fatal")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped not found", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil for auth-shaped 404", resource.Status.OsokStatus.DeletedAt)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-error-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-error-1", got)
	}
}

func TestAddressListDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-confirm-pre-error-1"

	fake := &fakeAddressListOCIClient{}
	fake.getFunc = func(context.Context, waassdk.GetAddressListRequest) (waassdk.GetAddressListResponse, error) {
		return waassdk.GetAddressListResponse{}, serviceErr
	}
	fake.deleteFunc = func(context.Context, waassdk.DeleteAddressListRequest) (waassdk.DeleteAddressListResponse, error) {
		t.Fatal("DeleteAddressList() called; want pre-delete auth-shaped read to stop delete")
		return waassdk.DeleteAddressListResponse{}, nil
	}
	client := newAddressListRuntimeTestClient(fake)

	resource := newAddressListRuntimeTestResource()
	trackAddressListID(resource, testAddressListID)
	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped pre-delete read to stay fatal")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read 404", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteAddressList() calls = %d, want 0", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-confirm-pre-error-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-confirm-pre-error-1", got)
	}
}

func TestAddressListDeleteEmptyOcidListFallbackConfirmsNoMatchAcrossPages(t *testing.T) {
	t.Parallel()

	fake := &fakeAddressListOCIClient{}
	fake.listFunc = func(_ context.Context, request waassdk.ListAddressListsRequest) (waassdk.ListAddressListsResponse, error) {
		requireAddressListListIdentity(t, request)
		switch len(fake.listRequests) {
		case 1:
			otherSpec := newAddressListRuntimeTestResource().Spec
			otherSpec.DisplayName = "address-list-other"
			return waassdk.ListAddressListsResponse{
				Items:       []waassdk.AddressListSummary{addressListSummaryFromSpec("ocid1.waasaddresslist.oc1..other", otherSpec, waassdk.LifecycleStatesActive)},
				OpcNextPage: common.String("page-2"),
			}, nil
		case 2:
			if got := addressListStringValue(request.Page); got != "page-2" {
				t.Fatalf("second list page = %q, want page-2", got)
			}
			return waassdk.ListAddressListsResponse{}, nil
		default:
			t.Fatalf("unexpected ListAddressLists() call %d", len(fake.listRequests))
			return waassdk.ListAddressListsResponse{}, nil
		}
	}
	fake.deleteFunc = func(context.Context, waassdk.DeleteAddressListRequest) (waassdk.DeleteAddressListResponse, error) {
		t.Fatal("DeleteAddressList() called; want list fallback absence to release finalizer")
		return waassdk.DeleteAddressListResponse{}, nil
	}
	client := newAddressListRuntimeTestClient(fake)

	resource := newAddressListRuntimeTestResource()
	resource.DeletionTimestamp = &metav1.Time{}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true when list fallback finds no match")
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListAddressLists() calls = %d, want 2", len(fake.listRequests))
	}
	if len(fake.getRequests) != 0 {
		t.Fatalf("GetAddressList() calls = %d, want 0 without recorded OCID", len(fake.getRequests))
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteAddressList() calls = %d, want 0 when list fallback confirms absence", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete timestamp")
	}
}

func TestAddressListCreateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
	serviceErr.OpcRequestID = "opc-create-error-1"

	fake := &fakeAddressListOCIClient{}
	fake.listFunc = func(context.Context, waassdk.ListAddressListsRequest) (waassdk.ListAddressListsResponse, error) {
		return waassdk.ListAddressListsResponse{}, nil
	}
	fake.createFunc = func(context.Context, waassdk.CreateAddressListRequest) (waassdk.CreateAddressListResponse, error) {
		return waassdk.CreateAddressListResponse{}, serviceErr
	}
	client := newAddressListRuntimeTestClient(fake)

	resource := newAddressListRuntimeTestResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want surfaced OCI create error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-error-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-error-1", got)
	}
}

func addressListExistingOnSecondPage(
	t *testing.T,
	fake *fakeAddressListOCIClient,
) func(context.Context, waassdk.ListAddressListsRequest) (waassdk.ListAddressListsResponse, error) {
	return func(_ context.Context, request waassdk.ListAddressListsRequest) (waassdk.ListAddressListsResponse, error) {
		t.Helper()
		requireAddressListListIdentity(t, request)
		switch len(fake.listRequests) {
		case 1:
			requireAddressListListPageUnset(t, request)
			otherSpec := newAddressListRuntimeTestResource().Spec
			otherSpec.DisplayName = "address-list-other"
			return waassdk.ListAddressListsResponse{
				Items:       []waassdk.AddressListSummary{addressListSummaryFromSpec("ocid1.waasaddresslist.oc1..other", otherSpec, waassdk.LifecycleStatesActive)},
				OpcNextPage: common.String("page-2"),
			}, nil
		case 2:
			requireAddressListListPage(t, request, "page-2")
			return waassdk.ListAddressListsResponse{
				Items: []waassdk.AddressListSummary{
					addressListSummaryFromSpec(testAddressListID, newAddressListRuntimeTestResource().Spec, waassdk.LifecycleStatesActive),
				},
			}, nil
		default:
			t.Fatalf("unexpected ListAddressLists() call %d", len(fake.listRequests))
			return waassdk.ListAddressListsResponse{}, nil
		}
	}
}

func failAddressListCreate(
	t *testing.T,
	message string,
) func(context.Context, waassdk.CreateAddressListRequest) (waassdk.CreateAddressListResponse, error) {
	return func(context.Context, waassdk.CreateAddressListRequest) (waassdk.CreateAddressListResponse, error) {
		t.Helper()
		t.Fatal(message)
		return waassdk.CreateAddressListResponse{}, nil
	}
}

func addressListGetResponsesByID(
	t *testing.T,
	fake *fakeAddressListOCIClient,
	responses ...waassdk.AddressList,
) func(context.Context, waassdk.GetAddressListRequest) (waassdk.GetAddressListResponse, error) {
	return func(_ context.Context, request waassdk.GetAddressListRequest) (waassdk.GetAddressListResponse, error) {
		t.Helper()
		requireAddressListGetID(t, request)
		index := len(fake.getRequests) - 1
		if index < 0 || index >= len(responses) {
			t.Fatalf("unexpected GetAddressList() call %d", len(fake.getRequests))
		}
		return waassdk.GetAddressListResponse{AddressList: responses[index]}, nil
	}
}

func addressListGetLifecycleSequence(
	t *testing.T,
	fake *fakeAddressListOCIClient,
	states ...waassdk.LifecycleStatesEnum,
) func(context.Context, waassdk.GetAddressListRequest) (waassdk.GetAddressListResponse, error) {
	spec := newAddressListRuntimeTestResource().Spec
	responses := make([]waassdk.AddressList, 0, len(states))
	for _, state := range states {
		responses = append(responses, addressListFromSpec(testAddressListID, spec, state))
	}
	return addressListGetResponsesByID(t, fake, responses...)
}

func addressListMutableUpdateFunc(
	t *testing.T,
	desired waasv1beta1.AddressListSpec,
) func(context.Context, waassdk.UpdateAddressListRequest) (waassdk.UpdateAddressListResponse, error) {
	return func(_ context.Context, request waassdk.UpdateAddressListRequest) (waassdk.UpdateAddressListResponse, error) {
		t.Helper()
		requireAddressListMutableUpdateRequest(t, request, desired)
		return waassdk.UpdateAddressListResponse{
			OpcRequestId: common.String("opc-update-1"),
			AddressList:  addressListFromSpec(testAddressListID, desired, waassdk.LifecycleStatesUpdating),
		}, nil
	}
}

func staleAddressListSpec() waasv1beta1.AddressListSpec {
	spec := newAddressListRuntimeTestResource().Spec
	spec.DisplayName = "address-list-old"
	spec.Addresses = []string{"203.0.113.10"}
	spec.FreeformTags = map[string]string{"env": "old"}
	spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	return spec
}

func newAddressListRuntimeTestClient(fake *fakeAddressListOCIClient) AddressListServiceClient {
	hooks := newAddressListDefaultRuntimeHooks(waassdk.WaasClient{})
	hooks.Create.Call = fake.CreateAddressList
	hooks.Get.Call = fake.GetAddressList
	hooks.List.Call = fake.ListAddressLists
	hooks.Update.Call = fake.UpdateAddressList
	hooks.Delete.Call = fake.DeleteAddressList
	applyAddressListRuntimeHooks(&hooks, fake, nil)

	manager := &AddressListServiceManager{
		Log: loggerutil.OSOKLogger{Logger: logr.Discard()},
	}
	delegate := defaultAddressListServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*waasv1beta1.AddressList](
			buildAddressListGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapAddressListGeneratedClient(hooks, delegate)
}

func newAddressListRuntimeTestResource() *waasv1beta1.AddressList {
	return &waasv1beta1.AddressList{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "address-list-sample",
			Namespace: "default",
			UID:       types.UID("addresslist-uid"),
		},
		Spec: waasv1beta1.AddressListSpec{
			CompartmentId: testAddressListCompartmentID,
			DisplayName:   testAddressListName,
			Addresses:     []string{"192.0.2.10", "198.51.100.0/24"},
			FreeformTags:  map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func trackAddressListID(resource *waasv1beta1.AddressList, id string) {
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
}

func addressListFromSpec(
	id string,
	spec waasv1beta1.AddressListSpec,
	state waassdk.LifecycleStatesEnum,
) waassdk.AddressList {
	addressCount := float32(len(spec.Addresses))
	return waassdk.AddressList{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		DisplayName:    common.String(spec.DisplayName),
		AddressCount:   common.Float32(addressCount),
		Addresses:      append([]string(nil), spec.Addresses...),
		FreeformTags:   maps.Clone(spec.FreeformTags),
		DefinedTags:    addressListDefinedTagsFromSpec(spec.DefinedTags),
		LifecycleState: state,
	}
}

func addressListSummaryFromSpec(
	id string,
	spec waasv1beta1.AddressListSpec,
	state waassdk.LifecycleStatesEnum,
) waassdk.AddressListSummary {
	addressCount := float32(len(spec.Addresses))
	return waassdk.AddressListSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		DisplayName:    common.String(spec.DisplayName),
		AddressCount:   common.Float32(addressCount),
		FreeformTags:   maps.Clone(spec.FreeformTags),
		DefinedTags:    addressListDefinedTagsFromSpec(spec.DefinedTags),
		LifecycleState: state,
	}
}

func requireAddressListListIdentity(t *testing.T, request waassdk.ListAddressListsRequest) {
	t.Helper()

	if got := addressListStringValue(request.CompartmentId); got != testAddressListCompartmentID {
		t.Fatalf("list compartmentId = %q, want %q", got, testAddressListCompartmentID)
	}
	if !reflect.DeepEqual(request.Name, []string{testAddressListName}) {
		t.Fatalf("list name = %#v, want %q filter", request.Name, testAddressListName)
	}
}

func requireAddressListListPage(t *testing.T, request waassdk.ListAddressListsRequest, want string) {
	t.Helper()

	if got := addressListStringValue(request.Page); got != want {
		t.Fatalf("list page = %q, want %q", got, want)
	}
}

func requireAddressListListPageUnset(t *testing.T, request waassdk.ListAddressListsRequest) {
	t.Helper()

	if request.Page != nil {
		t.Fatalf("list page = %q, want nil", addressListStringValue(request.Page))
	}
}

func requireAddressListGetID(t *testing.T, request waassdk.GetAddressListRequest) {
	t.Helper()

	if got := addressListStringValue(request.AddressListId); got != testAddressListID {
		t.Fatalf("get addressListId = %q, want %q", got, testAddressListID)
	}
}

func requireAddressListDeleteID(t *testing.T, request waassdk.DeleteAddressListRequest) {
	t.Helper()

	if got := addressListStringValue(request.AddressListId); got != testAddressListID {
		t.Fatalf("delete addressListId = %q, want %q", got, testAddressListID)
	}
}

func requireAddressListMoveRequest(
	t *testing.T,
	request waassdk.ChangeAddressListCompartmentRequest,
	compartmentID string,
) {
	t.Helper()

	if got := addressListStringValue(request.AddressListId); got != testAddressListID {
		t.Fatalf("move addressListId = %q, want %q", got, testAddressListID)
	}
	if got := addressListStringValue(request.CompartmentId); got != compartmentID {
		t.Fatalf("move compartmentId = %q, want %q", got, compartmentID)
	}
	if got := addressListStringValue(request.OpcRetryToken); got != string(types.UID("addresslist-uid"))+"-move-compartment" {
		t.Fatalf("move opcRetryToken = %q, want resource UID move token", got)
	}
}

func requireAddressListCreateRequest(
	t *testing.T,
	request waassdk.CreateAddressListRequest,
	spec waasv1beta1.AddressListSpec,
) {
	t.Helper()

	if got := addressListStringValue(request.CompartmentId); got != spec.CompartmentId {
		t.Fatalf("create compartmentId = %q, want %q", got, spec.CompartmentId)
	}
	if got := addressListStringValue(request.DisplayName); got != spec.DisplayName {
		t.Fatalf("create displayName = %q, want %q", got, spec.DisplayName)
	}
	if !reflect.DeepEqual(request.Addresses, spec.Addresses) {
		t.Fatalf("create addresses = %#v, want %#v", request.Addresses, spec.Addresses)
	}
	if !reflect.DeepEqual(request.FreeformTags, spec.FreeformTags) {
		t.Fatalf("create freeformTags = %#v, want %#v", request.FreeformTags, spec.FreeformTags)
	}
	if !addressListDefinedTagsEqual(spec.DefinedTags, request.DefinedTags) {
		t.Fatalf("create definedTags = %#v, want %#v", request.DefinedTags, spec.DefinedTags)
	}
	if got := addressListStringValue(request.OpcRetryToken); got != string(types.UID("addresslist-uid")) {
		t.Fatalf("create opcRetryToken = %q, want resource UID", got)
	}
}

func requireAddressListMutableUpdateRequest(
	t *testing.T,
	request waassdk.UpdateAddressListRequest,
	spec waasv1beta1.AddressListSpec,
) {
	t.Helper()

	if got := addressListStringValue(request.AddressListId); got != testAddressListID {
		t.Fatalf("update addressListId = %q, want %q", got, testAddressListID)
	}
	if got := addressListStringValue(request.DisplayName); got != spec.DisplayName {
		t.Fatalf("update displayName = %q, want %q", got, spec.DisplayName)
	}
	if !reflect.DeepEqual(request.Addresses, spec.Addresses) {
		t.Fatalf("update addresses = %#v, want %#v", request.Addresses, spec.Addresses)
	}
	if !reflect.DeepEqual(request.FreeformTags, spec.FreeformTags) {
		t.Fatalf("update freeformTags = %#v, want %#v", request.FreeformTags, spec.FreeformTags)
	}
	if !addressListDefinedTagsEqual(spec.DefinedTags, request.DefinedTags) {
		t.Fatalf("update definedTags = %#v, want %#v", request.DefinedTags, spec.DefinedTags)
	}
}

func requireAddressListActiveStatus(
	t *testing.T,
	resource *waasv1beta1.AddressList,
	opcRequestID string,
) {
	t.Helper()

	if got := resource.Status.Id; got != testAddressListID {
		t.Fatalf("status.id = %q, want %q", got, testAddressListID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testAddressListID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testAddressListID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != opcRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %s", got, opcRequestID)
	}
	if got := resource.Status.LifecycleState; got != string(waassdk.LifecycleStatesActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil for ACTIVE", resource.Status.OsokStatus.Async.Current)
	}
}

func requireAddressListBoundStatus(t *testing.T, resource *waasv1beta1.AddressList) {
	t.Helper()

	if got := resource.Status.Id; got != testAddressListID {
		t.Fatalf("status.id = %q, want bound AddressList ID", got)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testAddressListID {
		t.Fatalf("status.status.ocid = %q, want bound AddressList ID", got)
	}
	if !reflect.DeepEqual(resource.Status.Addresses, newAddressListRuntimeTestResource().Spec.Addresses) {
		t.Fatalf("status.addresses = %#v, want full readback addresses", resource.Status.Addresses)
	}
}

func requireAddressListUpdatedStatus(t *testing.T, resource *waasv1beta1.AddressList) {
	t.Helper()

	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
	if got := resource.Status.LifecycleState; got != string(waassdk.LifecycleStatesActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE after follow-up", got)
	}
}

func requireAddressListMoveThenUpdateResult(
	t *testing.T,
	fake *fakeAddressListOCIClient,
	resource *waasv1beta1.AddressList,
	response servicemanager.OSOKResponse,
) {
	t.Helper()

	requireAddressListSuccessfulNonRequeueResponse(t, response)
	requireAddressListMoveThenUpdateCalls(t, fake)
	requireAddressListMoveThenUpdateStatus(t, resource)
}

func requireAddressListSuccessfulNonRequeueResponse(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()

	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response successful=%t shouldRequeue=%t, want successful non-requeue", response.IsSuccessful, response.ShouldRequeue)
	}
}

func requireAddressListMoveThenUpdateCalls(t *testing.T, fake *fakeAddressListOCIClient) {
	t.Helper()

	if got := len(fake.changeCompartmentRequests); got != 1 {
		t.Fatalf("ChangeAddressListCompartment() calls = %d, want 1", got)
	}
	if got := len(fake.updateRequests); got != 1 {
		t.Fatalf("UpdateAddressList() calls = %d, want 1", got)
	}
	if got := len(fake.getRequests); got != 3 {
		t.Fatalf("GetAddressList() calls = %d, want current read, post-move refresh, and post-update refresh", got)
	}
}

func requireAddressListMoveThenUpdateStatus(t *testing.T, resource *waasv1beta1.AddressList) {
	t.Helper()

	if got := resource.Status.CompartmentId; got != resource.Spec.CompartmentId {
		t.Fatalf("status.compartmentId = %q, want moved compartment %q", got, resource.Spec.CompartmentId)
	}
	if got := resource.Status.DisplayName; got != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if !reflect.DeepEqual(resource.Status.Addresses, resource.Spec.Addresses) {
		t.Fatalf("status.addresses = %#v, want %#v", resource.Status.Addresses, resource.Spec.Addresses)
	}
	if !reflect.DeepEqual(resource.Status.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("status.freeformTags = %#v, want %#v", resource.Status.FreeformTags, resource.Spec.FreeformTags)
	}
	if !reflect.DeepEqual(resource.Status.DefinedTags, resource.Spec.DefinedTags) {
		t.Fatalf("status.definedTags = %#v, want %#v", resource.Status.DefinedTags, resource.Spec.DefinedTags)
	}
	requireAddressListUpdatedStatus(t, resource)
}

func requireAddressListDeletePendingStatus(
	t *testing.T,
	fake *fakeAddressListOCIClient,
	resource *waasv1beta1.AddressList,
) {
	t.Helper()

	if got := len(fake.deleteRequests); got != 1 {
		t.Fatalf("DeleteAddressList() calls after first delete = %d, want 1", got)
	}
	if got := resource.Status.LifecycleState; got != string(waassdk.LifecycleStatesDeleting) {
		t.Fatalf("status.lifecycleState after first delete = %q, want DELETING", got)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want delete lifecycle tracker")
	}
	if got := resource.Status.OsokStatus.Async.Current.Phase; got != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current.phase = %q, want delete", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}
}

func requireAddressListDeleteConfirmedStatus(
	t *testing.T,
	fake *fakeAddressListOCIClient,
	resource *waasv1beta1.AddressList,
) {
	t.Helper()

	if got := len(fake.deleteRequests); got != 1 {
		t.Fatalf("DeleteAddressList() calls after second delete = %d, want no reissue", got)
	}
	if got := resource.Status.LifecycleState; got != string(waassdk.LifecycleStatesDeleted) {
		t.Fatalf("status.lifecycleState after second delete = %q, want DELETED", got)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete timestamp")
	}
}
