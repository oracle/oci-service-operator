/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networkaddresslist

import (
	"context"
	"reflect"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	wafsdk "github.com/oracle/oci-go-sdk/v65/waf"
	wafv1beta1 "github.com/oracle/oci-service-operator/api/waf/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeNetworkAddressListOCIClient struct {
	t *testing.T

	createRequests []wafsdk.CreateNetworkAddressListRequest
	getRequests    []wafsdk.GetNetworkAddressListRequest
	listRequests   []wafsdk.ListNetworkAddressListsRequest
	updateRequests []wafsdk.UpdateNetworkAddressListRequest
	deleteRequests []wafsdk.DeleteNetworkAddressListRequest

	create func(context.Context, wafsdk.CreateNetworkAddressListRequest) (wafsdk.CreateNetworkAddressListResponse, error)
	get    func(context.Context, wafsdk.GetNetworkAddressListRequest) (wafsdk.GetNetworkAddressListResponse, error)
	list   func(context.Context, wafsdk.ListNetworkAddressListsRequest) (wafsdk.ListNetworkAddressListsResponse, error)
	update func(context.Context, wafsdk.UpdateNetworkAddressListRequest) (wafsdk.UpdateNetworkAddressListResponse, error)
	delete func(context.Context, wafsdk.DeleteNetworkAddressListRequest) (wafsdk.DeleteNetworkAddressListResponse, error)
}

func (f *fakeNetworkAddressListOCIClient) CreateNetworkAddressList(
	ctx context.Context,
	request wafsdk.CreateNetworkAddressListRequest,
) (wafsdk.CreateNetworkAddressListResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.create == nil {
		f.t.Fatalf("unexpected CreateNetworkAddressList call")
	}
	return f.create(ctx, request)
}

func (f *fakeNetworkAddressListOCIClient) GetNetworkAddressList(
	ctx context.Context,
	request wafsdk.GetNetworkAddressListRequest,
) (wafsdk.GetNetworkAddressListResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.get == nil {
		f.t.Fatalf("unexpected GetNetworkAddressList call")
	}
	return f.get(ctx, request)
}

func (f *fakeNetworkAddressListOCIClient) ListNetworkAddressLists(
	ctx context.Context,
	request wafsdk.ListNetworkAddressListsRequest,
) (wafsdk.ListNetworkAddressListsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.list == nil {
		f.t.Fatalf("unexpected ListNetworkAddressLists call")
	}
	return f.list(ctx, request)
}

func (f *fakeNetworkAddressListOCIClient) UpdateNetworkAddressList(
	ctx context.Context,
	request wafsdk.UpdateNetworkAddressListRequest,
) (wafsdk.UpdateNetworkAddressListResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.update == nil {
		f.t.Fatalf("unexpected UpdateNetworkAddressList call")
	}
	return f.update(ctx, request)
}

func (f *fakeNetworkAddressListOCIClient) DeleteNetworkAddressList(
	ctx context.Context,
	request wafsdk.DeleteNetworkAddressListRequest,
) (wafsdk.DeleteNetworkAddressListResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete == nil {
		f.t.Fatalf("unexpected DeleteNetworkAddressList call")
	}
	return f.delete(ctx, request)
}

func TestNetworkAddressListCreateOrUpdateCreatesAddressesAndRecordsStatus(t *testing.T) {
	const (
		compartmentID = "ocid1.compartment.oc1..create"
		resourceID    = "ocid1.networkaddresslist.oc1..create"
	)

	resource := newNetworkAddressList("sample", wafv1beta1.NetworkAddressListSpec{
		CompartmentId: compartmentID,
		DisplayName:   "sample",
		Type:          networkAddressListTypeAddresses,
		Addresses:     []string{"10.0.0.0/24"},
		FreeformTags:  map[string]string{"env": "dev"},
	})
	fake := &fakeNetworkAddressListOCIClient{t: t}
	fake.list = func(context.Context, wafsdk.ListNetworkAddressListsRequest) (wafsdk.ListNetworkAddressListsResponse, error) {
		return wafsdk.ListNetworkAddressListsResponse{}, nil
	}
	fake.create = func(_ context.Context, request wafsdk.CreateNetworkAddressListRequest) (wafsdk.CreateNetworkAddressListResponse, error) {
		details, ok := request.CreateNetworkAddressListDetails.(wafsdk.CreateNetworkAddressListAddressesDetails)
		if !ok {
			t.Fatalf("create body type = %T, want CreateNetworkAddressListAddressesDetails", request.CreateNetworkAddressListDetails)
		}
		if got, want := stringValue(details.CompartmentId), compartmentID; got != want {
			t.Fatalf("create compartmentId = %q, want %q", got, want)
		}
		if got, want := details.Addresses, []string{"10.0.0.0/24"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("create addresses = %#v, want %#v", got, want)
		}
		return wafsdk.CreateNetworkAddressListResponse{
			NetworkAddressList: networkAddressListAddresses(resourceID, "sample", compartmentID, wafsdk.NetworkAddressListLifecycleStateCreating, []string{"10.0.0.0/24"}),
			OpcRequestId:       common.String("opc-create-1"),
		}, nil
	}
	fake.get = func(context.Context, wafsdk.GetNetworkAddressListRequest) (wafsdk.GetNetworkAddressListResponse, error) {
		return wafsdk.GetNetworkAddressListResponse{
			NetworkAddressList: networkAddressListAddresses(resourceID, "sample", compartmentID, wafsdk.NetworkAddressListLifecycleStateActive, []string{"10.0.0.0/24"}),
			OpcRequestId:       common.String("opc-get-1"),
		}, nil
	}

	response, err := newTestNetworkAddressListServiceClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if got, want := resource.Status.OsokStatus.Ocid, shared.OCID(resourceID); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
	if got, want := resource.Status.Id, resourceID; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-create-1"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	if got, want := lastNetworkAddressListCondition(resource), shared.Active; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func TestNetworkAddressListCreateOrUpdateBindsExistingFromPaginatedList(t *testing.T) {
	const (
		compartmentID = "ocid1.compartment.oc1..bind"
		resourceID    = "ocid1.networkaddresslist.oc1..bind"
	)

	resource := newNetworkAddressList("sample", wafv1beta1.NetworkAddressListSpec{
		CompartmentId: compartmentID,
		DisplayName:   "sample",
		Addresses:     []string{"10.0.1.0/24"},
	})
	fake := &fakeNetworkAddressListOCIClient{t: t}
	fake.list = func(_ context.Context, request wafsdk.ListNetworkAddressListsRequest) (wafsdk.ListNetworkAddressListsResponse, error) {
		switch stringValue(request.Page) {
		case "":
			return wafsdk.ListNetworkAddressListsResponse{
				NetworkAddressListCollection: wafsdk.NetworkAddressListCollection{
					Items: []wafsdk.NetworkAddressListSummary{
						networkAddressListAddressesSummary("ocid1.networkaddresslist.oc1..other", "other", compartmentID, wafsdk.NetworkAddressListLifecycleStateActive, []string{"10.0.0.0/24"}),
					},
				},
				OpcNextPage: common.String("next"),
			}, nil
		case "next":
			return wafsdk.ListNetworkAddressListsResponse{
				NetworkAddressListCollection: wafsdk.NetworkAddressListCollection{
					Items: []wafsdk.NetworkAddressListSummary{
						networkAddressListAddressesSummary(resourceID, "sample", compartmentID, wafsdk.NetworkAddressListLifecycleStateActive, []string{"10.0.1.0/24"}),
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected list page = %q", stringValue(request.Page))
			return wafsdk.ListNetworkAddressListsResponse{}, nil
		}
	}
	fake.get = func(context.Context, wafsdk.GetNetworkAddressListRequest) (wafsdk.GetNetworkAddressListResponse, error) {
		return wafsdk.GetNetworkAddressListResponse{
			NetworkAddressList: networkAddressListAddresses(resourceID, "sample", compartmentID, wafsdk.NetworkAddressListLifecycleStateActive, []string{"10.0.1.0/24"}),
		}, nil
	}

	response, err := newTestNetworkAddressListServiceClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if got, want := len(fake.listRequests), 2; got != want {
		t.Fatalf("list call count = %d, want %d", got, want)
	}
	if got := len(fake.createRequests); got != 0 {
		t.Fatalf("create call count = %d, want 0", got)
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("update call count = %d, want 0", got)
	}
	if got, want := resource.Status.OsokStatus.Ocid, shared.OCID(resourceID); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func TestNetworkAddressListCreateOrUpdateDoesNotBindOmittedDisplayNameByCompartmentOnly(t *testing.T) {
	const (
		compartmentID = "ocid1.compartment.oc1..omittedname"
		createdID     = "ocid1.networkaddresslist.oc1..omittedname-created"
		otherID       = "ocid1.networkaddresslist.oc1..omittedname-other"
	)

	resource := newNetworkAddressList("sample", wafv1beta1.NetworkAddressListSpec{
		CompartmentId: compartmentID,
		Addresses:     []string{"10.0.9.0/24"},
	})
	fake := &fakeNetworkAddressListOCIClient{t: t}
	fake.list = func(context.Context, wafsdk.ListNetworkAddressListsRequest) (wafsdk.ListNetworkAddressListsResponse, error) {
		return wafsdk.ListNetworkAddressListsResponse{
			NetworkAddressListCollection: wafsdk.NetworkAddressListCollection{
				Items: []wafsdk.NetworkAddressListSummary{
					networkAddressListAddressesSummary(otherID, "unrelated", compartmentID, wafsdk.NetworkAddressListLifecycleStateActive, []string{"10.99.0.0/24"}),
				},
			},
		}, nil
	}
	fake.create = func(_ context.Context, request wafsdk.CreateNetworkAddressListRequest) (wafsdk.CreateNetworkAddressListResponse, error) {
		details, ok := request.CreateNetworkAddressListDetails.(wafsdk.CreateNetworkAddressListAddressesDetails)
		if !ok {
			t.Fatalf("create body type = %T, want CreateNetworkAddressListAddressesDetails", request.CreateNetworkAddressListDetails)
		}
		if got, want := details.Addresses, []string{"10.0.9.0/24"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("create addresses = %#v, want %#v", got, want)
		}
		return wafsdk.CreateNetworkAddressListResponse{
			NetworkAddressList: networkAddressListAddresses(createdID, "", compartmentID, wafsdk.NetworkAddressListLifecycleStateCreating, []string{"10.0.9.0/24"}),
		}, nil
	}
	fake.get = func(context.Context, wafsdk.GetNetworkAddressListRequest) (wafsdk.GetNetworkAddressListResponse, error) {
		return wafsdk.GetNetworkAddressListResponse{
			NetworkAddressList: networkAddressListAddresses(createdID, "", compartmentID, wafsdk.NetworkAddressListLifecycleStateActive, []string{"10.0.9.0/24"}),
		}, nil
	}

	response, err := newTestNetworkAddressListServiceClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if got, want := len(fake.createRequests), 1; got != want {
		t.Fatalf("create call count = %d, want %d", got, want)
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("update call count = %d, want 0", got)
	}
	if got, want := resource.Status.OsokStatus.Ocid, shared.OCID(createdID); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func TestNetworkAddressListCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	const (
		compartmentID = "ocid1.compartment.oc1..noop"
		resourceID    = "ocid1.networkaddresslist.oc1..noop"
	)

	resource := newNetworkAddressList("sample", wafv1beta1.NetworkAddressListSpec{
		CompartmentId: compartmentID,
		DisplayName:   "sample",
		Addresses:     []string{"10.0.2.0/24"},
	})
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	fake := &fakeNetworkAddressListOCIClient{t: t}
	fake.get = func(context.Context, wafsdk.GetNetworkAddressListRequest) (wafsdk.GetNetworkAddressListResponse, error) {
		return wafsdk.GetNetworkAddressListResponse{
			NetworkAddressList: networkAddressListAddresses(resourceID, "sample", compartmentID, wafsdk.NetworkAddressListLifecycleStateActive, []string{"10.0.2.0/24"}),
		}, nil
	}

	response, err := newTestNetworkAddressListServiceClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("update call count = %d, want 0", got)
	}
}

func TestNetworkAddressListCreateOrUpdateNoopsWhenExplicitTypeCaseDiffersFromReadback(t *testing.T) {
	const (
		compartmentID = "ocid1.compartment.oc1..typecase"
		resourceID    = "ocid1.networkaddresslist.oc1..typecase"
	)

	resource := newNetworkAddressList("sample", wafv1beta1.NetworkAddressListSpec{
		CompartmentId: compartmentID,
		DisplayName:   "sample",
		Type:          "addresses",
		Addresses:     []string{"10.0.2.0/24"},
	})
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	fake := &fakeNetworkAddressListOCIClient{t: t}
	fake.get = func(context.Context, wafsdk.GetNetworkAddressListRequest) (wafsdk.GetNetworkAddressListResponse, error) {
		return wafsdk.GetNetworkAddressListResponse{
			NetworkAddressList: networkAddressListAddresses(resourceID, "sample", compartmentID, wafsdk.NetworkAddressListLifecycleStateActive, []string{"10.0.2.0/24"}),
		}, nil
	}

	response, err := newTestNetworkAddressListServiceClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("update call count = %d, want 0", got)
	}
	if got, want := resource.Spec.Type, networkAddressListTypeAddresses; got != want {
		t.Fatalf("spec.type = %q, want %q", got, want)
	}
}

func TestNetworkAddressListCreateOrUpdateUpdatesMutableAddressesFields(t *testing.T) {
	const (
		compartmentID = "ocid1.compartment.oc1..update"
		resourceID    = "ocid1.networkaddresslist.oc1..update"
	)

	resource := newMutableAddressesUpdateResource(compartmentID, resourceID)
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	fake := &fakeNetworkAddressListOCIClient{t: t}
	configureMutableAddressesUpdateFake(t, fake, resourceID, compartmentID)

	response, err := newTestNetworkAddressListServiceClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertMutableAddressesUpdateResult(t, resource, response, fake)
}

func newMutableAddressesUpdateResource(compartmentID, resourceID string) *wafv1beta1.NetworkAddressList {
	resource := newNetworkAddressList("sample", wafv1beta1.NetworkAddressListSpec{
		CompartmentId: compartmentID,
		DisplayName:   "sample-renamed",
		Addresses:     []string{"10.0.3.0/24", "10.0.4.0/24"},
		FreeformTags:  map[string]string{"env": "prod"},
	})
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	return resource
}

func configureMutableAddressesUpdateFake(
	t *testing.T,
	fake *fakeNetworkAddressListOCIClient,
	resourceID string,
	compartmentID string,
) {
	t.Helper()
	fake.get = func(_ context.Context, _ wafsdk.GetNetworkAddressListRequest) (wafsdk.GetNetworkAddressListResponse, error) {
		displayName := "sample"
		addresses := []string{"10.0.3.0/24"}
		if len(fake.getRequests) > 1 {
			displayName = "sample-renamed"
			addresses = []string{"10.0.3.0/24", "10.0.4.0/24"}
		}
		return wafsdk.GetNetworkAddressListResponse{
			NetworkAddressList: networkAddressListAddresses(
				resourceID,
				displayName,
				compartmentID,
				wafsdk.NetworkAddressListLifecycleStateActive,
				addresses,
			),
		}, nil
	}
	fake.update = func(_ context.Context, request wafsdk.UpdateNetworkAddressListRequest) (wafsdk.UpdateNetworkAddressListResponse, error) {
		assertMutableAddressesUpdateRequest(t, request)
		return wafsdk.UpdateNetworkAddressListResponse{OpcRequestId: common.String("opc-update-1")}, nil
	}
}

func assertMutableAddressesUpdateRequest(t *testing.T, request wafsdk.UpdateNetworkAddressListRequest) {
	t.Helper()
	details, ok := request.UpdateNetworkAddressListDetails.(wafsdk.UpdateNetworkAddressListAddressesDetails)
	if !ok {
		t.Fatalf("update body type = %T, want UpdateNetworkAddressListAddressesDetails", request.UpdateNetworkAddressListDetails)
	}
	if got, want := stringValue(details.DisplayName), "sample-renamed"; got != want {
		t.Fatalf("update displayName = %q, want %q", got, want)
	}
	if got, want := details.Addresses, []string{"10.0.3.0/24", "10.0.4.0/24"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("update addresses = %#v, want %#v", got, want)
	}
	if got, want := details.FreeformTags, map[string]string{"env": "prod"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("update freeformTags = %#v, want %#v", got, want)
	}
}

func assertMutableAddressesUpdateResult(
	t *testing.T,
	resource *wafv1beta1.NetworkAddressList,
	response servicemanager.OSOKResponse,
	fake *fakeNetworkAddressListOCIClient,
) {
	t.Helper()
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = false, want true")
	}
	if got, want := len(fake.updateRequests), 1; got != want {
		t.Fatalf("update call count = %d, want %d", got, want)
	}
	if got, want := resource.Status.DisplayName, "sample-renamed"; got != want {
		t.Fatalf("status.displayName = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-update-1"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestNetworkAddressListCreateOrUpdateRejectsCreateOnlyTypeDrift(t *testing.T) {
	const (
		compartmentID = "ocid1.compartment.oc1..drift"
		resourceID    = "ocid1.networkaddresslist.oc1..drift"
	)

	resource := newNetworkAddressList("sample", wafv1beta1.NetworkAddressListSpec{
		CompartmentId: compartmentID,
		Type:          networkAddressListTypeVcnAddresses,
		VcnAddresses: []wafv1beta1.NetworkAddressListVcnAddress{{
			VcnId:     "ocid1.vcn.oc1..drift",
			Addresses: "10.0.0.0/16",
		}},
	})
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	fake := &fakeNetworkAddressListOCIClient{t: t}
	fake.get = func(context.Context, wafsdk.GetNetworkAddressListRequest) (wafsdk.GetNetworkAddressListResponse, error) {
		return wafsdk.GetNetworkAddressListResponse{
			NetworkAddressList: networkAddressListAddresses(resourceID, "sample", compartmentID, wafsdk.NetworkAddressListLifecycleStateActive, []string{"10.0.0.0/24"}),
		}, nil
	}

	response, err := newTestNetworkAddressListServiceClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want type drift error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = true, want false")
	}
	if got := len(fake.updateRequests); got != 0 {
		t.Fatalf("update call count = %d, want 0", got)
	}
}

func TestNetworkAddressListDeleteRetainsFinalizerWhileLifecycleDeleting(t *testing.T) {
	const (
		compartmentID = "ocid1.compartment.oc1..delete"
		resourceID    = "ocid1.networkaddresslist.oc1..delete"
	)

	resource := newNetworkAddressList("sample", wafv1beta1.NetworkAddressListSpec{
		CompartmentId: compartmentID,
		DisplayName:   "sample",
		Addresses:     []string{"10.0.5.0/24"},
	})
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	fake := &fakeNetworkAddressListOCIClient{t: t}
	fake.get = func(context.Context, wafsdk.GetNetworkAddressListRequest) (wafsdk.GetNetworkAddressListResponse, error) {
		state := wafsdk.NetworkAddressListLifecycleStateActive
		if len(fake.getRequests) == 3 {
			state = wafsdk.NetworkAddressListLifecycleStateDeleting
		}
		return wafsdk.GetNetworkAddressListResponse{
			NetworkAddressList: networkAddressListAddresses(resourceID, "sample", compartmentID, state, []string{"10.0.5.0/24"}),
		}, nil
	}
	fake.delete = func(context.Context, wafsdk.DeleteNetworkAddressListRequest) (wafsdk.DeleteNetworkAddressListResponse, error) {
		return wafsdk.DeleteNetworkAddressListResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}

	deleted, err := newTestNetworkAddressListServiceClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while lifecycle is DELETING")
	}
	if got, want := lastNetworkAddressListCondition(resource), shared.Terminating; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-delete-1"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestNetworkAddressListDeleteCompletesOnTerminalLifecycle(t *testing.T) {
	const (
		compartmentID = "ocid1.compartment.oc1..deleted"
		resourceID    = "ocid1.networkaddresslist.oc1..deleted"
	)

	resource := newNetworkAddressList("sample", wafv1beta1.NetworkAddressListSpec{
		CompartmentId: compartmentID,
		DisplayName:   "sample",
		Addresses:     []string{"10.0.6.0/24"},
	})
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	fake := &fakeNetworkAddressListOCIClient{t: t}
	fake.get = func(context.Context, wafsdk.GetNetworkAddressListRequest) (wafsdk.GetNetworkAddressListResponse, error) {
		state := wafsdk.NetworkAddressListLifecycleStateActive
		if len(fake.getRequests) == 3 {
			state = wafsdk.NetworkAddressListLifecycleStateDeleted
		}
		return wafsdk.GetNetworkAddressListResponse{
			NetworkAddressList: networkAddressListAddresses(resourceID, "sample", compartmentID, state, []string{"10.0.6.0/24"}),
		}, nil
	}
	fake.delete = func(context.Context, wafsdk.DeleteNetworkAddressListRequest) (wafsdk.DeleteNetworkAddressListResponse, error) {
		return wafsdk.DeleteNetworkAddressListResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}

	deleted, err := newTestNetworkAddressListServiceClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true for lifecycle DELETED")
	}
}

func TestNetworkAddressListDeleteRejectsAuthShapedNotFound(t *testing.T) {
	const resourceID = "ocid1.networkaddresslist.oc1..auth"

	resource := newNetworkAddressList("sample", wafv1beta1.NetworkAddressListSpec{
		CompartmentId: "ocid1.compartment.oc1..auth",
		DisplayName:   "sample",
		Addresses:     []string{"10.0.7.0/24"},
	})
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	authErr.OpcRequestID = "opc-auth-1"
	fake := &fakeNetworkAddressListOCIClient{t: t}
	fake.get = func(context.Context, wafsdk.GetNetworkAddressListRequest) (wafsdk.GetNetworkAddressListResponse, error) {
		return wafsdk.GetNetworkAddressListResponse{}, authErr
	}

	deleted, err := newTestNetworkAddressListServiceClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if got := len(fake.deleteRequests); got != 0 {
		t.Fatalf("delete call count = %d, want 0", got)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-auth-1"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestNetworkAddressListDeleteWithoutTrackedIDIgnoresOmittedDisplayNameCompartmentOnlyMatch(t *testing.T) {
	const (
		compartmentID = "ocid1.compartment.oc1..delete-untracked"
		otherID       = "ocid1.networkaddresslist.oc1..delete-untracked-other"
	)

	resource := newNetworkAddressList("sample", wafv1beta1.NetworkAddressListSpec{
		CompartmentId: compartmentID,
		Addresses:     []string{"10.0.10.0/24"},
	})
	fake := &fakeNetworkAddressListOCIClient{t: t}
	fake.list = func(context.Context, wafsdk.ListNetworkAddressListsRequest) (wafsdk.ListNetworkAddressListsResponse, error) {
		return wafsdk.ListNetworkAddressListsResponse{
			NetworkAddressListCollection: wafsdk.NetworkAddressListCollection{
				Items: []wafsdk.NetworkAddressListSummary{
					networkAddressListAddressesSummary(otherID, "unrelated", compartmentID, wafsdk.NetworkAddressListLifecycleStateActive, []string{"10.88.0.0/24"}),
				},
			},
		}, nil
	}

	deleted, err := newTestNetworkAddressListServiceClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true when no matching OCI resource exists")
	}
	if got := len(fake.deleteRequests); got != 0 {
		t.Fatalf("delete call count = %d, want 0", got)
	}
	if got, want := lastNetworkAddressListCondition(resource), shared.Terminating; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func TestNetworkAddressListCreateOrUpdateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := newNetworkAddressList("sample", wafv1beta1.NetworkAddressListSpec{
		CompartmentId: "ocid1.compartment.oc1..error",
		DisplayName:   "sample",
		Addresses:     []string{"10.0.8.0/24"},
	})
	ociErr := errortest.NewServiceError(500, "InternalError", "create failed")
	ociErr.OpcRequestID = "opc-create-error"
	fake := &fakeNetworkAddressListOCIClient{t: t}
	fake.list = func(context.Context, wafsdk.ListNetworkAddressListsRequest) (wafsdk.ListNetworkAddressListsResponse, error) {
		return wafsdk.ListNetworkAddressListsResponse{}, nil
	}
	fake.create = func(context.Context, wafsdk.CreateNetworkAddressListRequest) (wafsdk.CreateNetworkAddressListResponse, error) {
		return wafsdk.CreateNetworkAddressListResponse{}, ociErr
	}

	response, err := newTestNetworkAddressListServiceClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate().IsSuccessful = true, want false")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-create-error"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	if got, want := lastNetworkAddressListCondition(resource), shared.Failed; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func TestBuildNetworkAddressListCreateBodySupportsVcnAddresses(t *testing.T) {
	resource := newNetworkAddressList("vcn", wafv1beta1.NetworkAddressListSpec{
		CompartmentId: "ocid1.compartment.oc1..vcn",
		DisplayName:   "vcn",
		Type:          networkAddressListTypeVcnAddresses,
		VcnAddresses: []wafv1beta1.NetworkAddressListVcnAddress{{
			VcnId:     "ocid1.vcn.oc1..vcn",
			Addresses: "10.0.0.0/16",
		}},
	})

	body, err := buildNetworkAddressListCreateBody(context.Background(), resource, "")
	if err != nil {
		t.Fatalf("buildNetworkAddressListCreateBody() error = %v", err)
	}
	details, ok := body.(wafsdk.CreateNetworkAddressListVcnAddressesDetails)
	if !ok {
		t.Fatalf("create body type = %T, want CreateNetworkAddressListVcnAddressesDetails", body)
	}
	if got, want := len(details.VcnAddresses), 1; got != want {
		t.Fatalf("vcnAddresses length = %d, want %d", got, want)
	}
	if got, want := stringValue(details.VcnAddresses[0].VcnId), "ocid1.vcn.oc1..vcn"; got != want {
		t.Fatalf("vcnAddresses[0].vcnId = %q, want %q", got, want)
	}
}

func newTestNetworkAddressListServiceClient(fake *fakeNetworkAddressListOCIClient) NetworkAddressListServiceClient {
	hooks := NetworkAddressListRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*wafv1beta1.NetworkAddressList]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*wafv1beta1.NetworkAddressList]{},
		StatusHooks:     generatedruntime.StatusHooks[*wafv1beta1.NetworkAddressList]{},
		ParityHooks:     generatedruntime.ParityHooks[*wafv1beta1.NetworkAddressList]{},
		Async:           generatedruntime.AsyncHooks[*wafv1beta1.NetworkAddressList]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*wafv1beta1.NetworkAddressList]{},
		Create: runtimeOperationHooks[wafsdk.CreateNetworkAddressListRequest, wafsdk.CreateNetworkAddressListResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateNetworkAddressListDetails", RequestName: "CreateNetworkAddressListDetails", Contribution: "body"}},
			Call:   fake.CreateNetworkAddressList,
		},
		Get: runtimeOperationHooks[wafsdk.GetNetworkAddressListRequest, wafsdk.GetNetworkAddressListResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "NetworkAddressListId", RequestName: "networkAddressListId", Contribution: "path", PreferResourceID: true}},
			Call:   fake.GetNetworkAddressList,
		},
		List: runtimeOperationHooks[wafsdk.ListNetworkAddressListsRequest, wafsdk.ListNetworkAddressListsResponse]{
			Fields: networkAddressListListFields(),
			Call:   fake.ListNetworkAddressLists,
		},
		Update: runtimeOperationHooks[wafsdk.UpdateNetworkAddressListRequest, wafsdk.UpdateNetworkAddressListResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "NetworkAddressListId", RequestName: "networkAddressListId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateNetworkAddressListDetails", RequestName: "UpdateNetworkAddressListDetails", Contribution: "body"},
			},
			Call: fake.UpdateNetworkAddressList,
		},
		Delete: runtimeOperationHooks[wafsdk.DeleteNetworkAddressListRequest, wafsdk.DeleteNetworkAddressListResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "NetworkAddressListId", RequestName: "networkAddressListId", Contribution: "path", PreferResourceID: true}},
			Call:   fake.DeleteNetworkAddressList,
		},
		WrapGeneratedClient: []func(NetworkAddressListServiceClient) NetworkAddressListServiceClient{},
	}
	applyNetworkAddressListRuntimeHooks(&hooks)
	manager := &NetworkAddressListServiceManager{Log: loggerutil.OSOKLogger{}}
	delegate := defaultNetworkAddressListServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*wafv1beta1.NetworkAddressList](
			buildNetworkAddressListGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapNetworkAddressListGeneratedClient(hooks, delegate)
}

func newNetworkAddressList(name string, spec wafv1beta1.NetworkAddressListSpec) *wafv1beta1.NetworkAddressList {
	return &wafv1beta1.NetworkAddressList{
		Spec: spec,
	}
}

func networkAddressListAddresses(
	id string,
	displayName string,
	compartmentID string,
	state wafsdk.NetworkAddressListLifecycleStateEnum,
	addresses []string,
) wafsdk.NetworkAddressListAddresses {
	return wafsdk.NetworkAddressListAddresses{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String(compartmentID),
		LifecycleState: state,
		Addresses:      append([]string(nil), addresses...),
		FreeformTags:   map[string]string{},
		DefinedTags:    map[string]map[string]interface{}{},
		SystemTags:     map[string]map[string]interface{}{},
	}
}

func networkAddressListAddressesSummary(
	id string,
	displayName string,
	compartmentID string,
	state wafsdk.NetworkAddressListLifecycleStateEnum,
	addresses []string,
) wafsdk.NetworkAddressListAddressesSummary {
	return wafsdk.NetworkAddressListAddressesSummary{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String(compartmentID),
		LifecycleState: state,
		Addresses:      append([]string(nil), addresses...),
		FreeformTags:   map[string]string{},
		DefinedTags:    map[string]map[string]interface{}{},
		SystemTags:     map[string]map[string]interface{}{},
	}
}

func lastNetworkAddressListCondition(resource *wafv1beta1.NetworkAddressList) shared.OSOKConditionType {
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		return ""
	}
	return resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type
}
