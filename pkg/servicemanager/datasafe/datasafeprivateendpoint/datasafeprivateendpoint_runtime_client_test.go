/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package datasafeprivateendpoint

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testDataSafePrivateEndpointID        = "ocid1.datasafeprivateendpoint.oc1..existing"
	testUpdatedDataSafePrivateEndpointID = "ocid1.datasafeprivateendpoint.oc1..updated"
	testDataSafeCompartmentID            = "ocid1.compartment.oc1..datasafe"
	testDataSafeVCNID                    = "ocid1.vcn.oc1..datasafe"
	testDataSafeSubnetID                 = "ocid1.subnet.oc1..datasafe"
	testDataSafeNSGID                    = "ocid1.networksecuritygroup.oc1..datasafe"
	testUpdatedDataSafeNSGID             = "ocid1.networksecuritygroup.oc1..updated"
	testDataSafeUnderlyingPrivateID      = "ocid1.privateendpoint.oc1..datasafe"
	testDataSafePrivateEndpointName      = "datasafe-endpoint"
	testDataSafePrivateEndpointIP        = "10.0.0.10"
)

type fakeDataSafePrivateEndpointOCIClient struct {
	resources map[string]datasafesdk.DataSafePrivateEndpoint

	createRequests      []datasafesdk.CreateDataSafePrivateEndpointRequest
	getRequests         []datasafesdk.GetDataSafePrivateEndpointRequest
	listRequests        []datasafesdk.ListDataSafePrivateEndpointsRequest
	updateRequests      []datasafesdk.UpdateDataSafePrivateEndpointRequest
	deleteRequests      []datasafesdk.DeleteDataSafePrivateEndpointRequest
	workRequestRequests []datasafesdk.GetWorkRequestRequest

	getResults    []dataSafePrivateEndpointGetResult
	listResponses []datasafesdk.ListDataSafePrivateEndpointsResponse
	workRequests  map[string]datasafesdk.WorkRequest

	createErr error
	updateErr error
	deleteErr error

	createWorkRequestID string
	updateWorkRequestID string
	deleteWorkRequestID string
}

type dataSafePrivateEndpointGetResult struct {
	response datasafesdk.GetDataSafePrivateEndpointResponse
	err      error
}

func (f *fakeDataSafePrivateEndpointOCIClient) CreateDataSafePrivateEndpoint(
	_ context.Context,
	request datasafesdk.CreateDataSafePrivateEndpointRequest,
) (datasafesdk.CreateDataSafePrivateEndpointResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return datasafesdk.CreateDataSafePrivateEndpointResponse{}, f.createErr
	}

	resource := dataSafePrivateEndpointFromCreateDetails(testDataSafePrivateEndpointID, request.CreateDataSafePrivateEndpointDetails)
	f.ensureResources()[testDataSafePrivateEndpointID] = resource
	return datasafesdk.CreateDataSafePrivateEndpointResponse{
		DataSafePrivateEndpoint: resource,
		OpcWorkRequestId:        common.String(f.createWorkRequestID),
		OpcRequestId:            common.String("opc-create-1"),
	}, nil
}

func (f *fakeDataSafePrivateEndpointOCIClient) GetDataSafePrivateEndpoint(
	_ context.Context,
	request datasafesdk.GetDataSafePrivateEndpointRequest,
) (datasafesdk.GetDataSafePrivateEndpointResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if len(f.getResults) > 0 {
		result := f.getResults[0]
		f.getResults = f.getResults[1:]
		return result.response, result.err
	}

	id := dataSafePrivateEndpointStringValue(request.DataSafePrivateEndpointId)
	resource, ok := f.resources[id]
	if !ok {
		return datasafesdk.GetDataSafePrivateEndpointResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
	}
	return datasafesdk.GetDataSafePrivateEndpointResponse{
		DataSafePrivateEndpoint: resource,
		OpcRequestId:            common.String("opc-get-1"),
	}, nil
}

func (f *fakeDataSafePrivateEndpointOCIClient) ListDataSafePrivateEndpoints(
	_ context.Context,
	request datasafesdk.ListDataSafePrivateEndpointsRequest,
) (datasafesdk.ListDataSafePrivateEndpointsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if len(f.listResponses) > 0 {
		response := f.listResponses[0]
		f.listResponses = f.listResponses[1:]
		return response, nil
	}

	var items []datasafesdk.DataSafePrivateEndpointSummary
	for _, resource := range f.resources {
		if dataSafePrivateEndpointMatchesListRequest(resource, request) {
			items = append(items, dataSafePrivateEndpointSummaryFromSDK(resource))
		}
	}
	return datasafesdk.ListDataSafePrivateEndpointsResponse{
		Items:        items,
		OpcRequestId: common.String("opc-list-1"),
	}, nil
}

func (f *fakeDataSafePrivateEndpointOCIClient) UpdateDataSafePrivateEndpoint(
	_ context.Context,
	request datasafesdk.UpdateDataSafePrivateEndpointRequest,
) (datasafesdk.UpdateDataSafePrivateEndpointResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return datasafesdk.UpdateDataSafePrivateEndpointResponse{}, f.updateErr
	}
	return datasafesdk.UpdateDataSafePrivateEndpointResponse{
		OpcWorkRequestId: common.String(f.updateWorkRequestID),
		OpcRequestId:     common.String("opc-update-1"),
	}, nil
}

func (f *fakeDataSafePrivateEndpointOCIClient) DeleteDataSafePrivateEndpoint(
	_ context.Context,
	request datasafesdk.DeleteDataSafePrivateEndpointRequest,
) (datasafesdk.DeleteDataSafePrivateEndpointResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return datasafesdk.DeleteDataSafePrivateEndpointResponse{}, f.deleteErr
	}
	return datasafesdk.DeleteDataSafePrivateEndpointResponse{
		OpcWorkRequestId: common.String(f.deleteWorkRequestID),
		OpcRequestId:     common.String("opc-delete-1"),
	}, nil
}

func (f *fakeDataSafePrivateEndpointOCIClient) GetWorkRequest(
	_ context.Context,
	request datasafesdk.GetWorkRequestRequest,
) (datasafesdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	workRequest, ok := f.workRequests[dataSafePrivateEndpointStringValue(request.WorkRequestId)]
	if !ok {
		return datasafesdk.GetWorkRequestResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "work request not found")
	}
	return datasafesdk.GetWorkRequestResponse{
		WorkRequest:  workRequest,
		OpcRequestId: common.String("opc-work-request-1"),
	}, nil
}

func (f *fakeDataSafePrivateEndpointOCIClient) ensureResources() map[string]datasafesdk.DataSafePrivateEndpoint {
	if f.resources == nil {
		f.resources = map[string]datasafesdk.DataSafePrivateEndpoint{}
	}
	return f.resources
}

func TestDataSafePrivateEndpointRuntimeHooks(t *testing.T) {
	hooks := newDataSafePrivateEndpointRuntimeHooksWithOCIClient(&fakeDataSafePrivateEndpointOCIClient{})
	applyDataSafePrivateEndpointRuntimeHooks(&DataSafePrivateEndpointServiceManager{}, &hooks, nil, nil)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed runtime semantics")
	}
	if got := hooks.Semantics.Async.Strategy; got != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got)
	}
	if got := hooks.Semantics.FinalizerPolicy; got != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got)
	}
	assertContainsAll(t, "Mutation.Mutable", hooks.Semantics.Mutation.Mutable, "displayName", "description", "nsgIds", "freeformTags", "definedTags")
	assertContainsAll(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "compartmentId", "vcnId", "subnetId", "privateEndpointIp")
	assertContainsAll(t, "List.MatchFields", hooks.Semantics.List.MatchFields, "displayName", "compartmentId", "vcnId", "subnetId")
	if hooks.BuildCreateBody == nil || hooks.BuildUpdateBody == nil {
		t.Fatal("BuildCreateBody/BuildUpdateBody = nil, want resource-specific body builders")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("DeleteHooks.HandleError = nil, want auth-shaped not-found guard")
	}

	resource := dataSafePrivateEndpointResource()
	body, err := hooks.BuildCreateBody(context.Background(), resource, "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	createBody := body.(datasafesdk.CreateDataSafePrivateEndpointDetails)
	if got := dataSafePrivateEndpointStringValue(createBody.PrivateEndpointIp); got != testDataSafePrivateEndpointIP {
		t.Fatalf("create privateEndpointIp = %q, want %q", got, testDataSafePrivateEndpointIP)
	}
	if !reflect.DeepEqual(createBody.NsgIds, []string{testDataSafeNSGID}) {
		t.Fatalf("create nsgIds = %#v, want %#v", createBody.NsgIds, []string{testDataSafeNSGID})
	}
}

func TestDataSafePrivateEndpointCreateOrUpdateCreatesAndTracksWorkRequest(t *testing.T) {
	client := &fakeDataSafePrivateEndpointOCIClient{
		createWorkRequestID: "wr-create-1",
		workRequests: map[string]datasafesdk.WorkRequest{
			"wr-create-1": dataSafePrivateEndpointWorkRequest(
				"wr-create-1",
				datasafesdk.WorkRequestStatusSucceeded,
				datasafesdk.WorkRequestOperationTypeCreatePrivateEndpoint,
				datasafesdk.WorkRequestResourceActionTypeCreated,
				testDataSafePrivateEndpointID,
			),
		},
		getResults: []dataSafePrivateEndpointGetResult{{
			response: dataSafePrivateEndpointGetResponse(dataSafePrivateEndpointSDK(testDataSafePrivateEndpointID, datasafesdk.LifecycleStateActive)),
		}},
	}
	serviceClient := newDataSafePrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := dataSafePrivateEndpointResource()

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(client.createRequests))
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testDataSafePrivateEndpointID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testDataSafePrivateEndpointID)
	}
	if got := resource.Status.Id; got != testDataSafePrivateEndpointID {
		t.Fatalf("status.id = %q, want %q", got, testDataSafePrivateEndpointID)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after succeeded work request readback", resource.Status.OsokStatus.Async.Current)
	}
}

func TestDataSafePrivateEndpointCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	existing := dataSafePrivateEndpointSDK(testDataSafePrivateEndpointID, datasafesdk.LifecycleStateActive)
	client := &fakeDataSafePrivateEndpointOCIClient{
		resources: map[string]datasafesdk.DataSafePrivateEndpoint{
			testDataSafePrivateEndpointID: existing,
		},
		listResponses: []datasafesdk.ListDataSafePrivateEndpointsResponse{
			{
				Items:       nil,
				OpcNextPage: common.String("page-2"),
			},
			{
				Items: []datasafesdk.DataSafePrivateEndpointSummary{dataSafePrivateEndpointSummaryFromSDK(existing)},
			},
		},
	}
	serviceClient := newDataSafePrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := dataSafePrivateEndpointResource()

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for bind", len(client.createRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2 for pagination", len(client.listRequests))
	}
	if got := dataSafePrivateEndpointStringValue(client.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second list page = %q, want page-2", got)
	}
	if got := resource.Status.Id; got != testDataSafePrivateEndpointID {
		t.Fatalf("status.id = %q, want bound ID %q", got, testDataSafePrivateEndpointID)
	}
}

func TestDataSafePrivateEndpointCreateOrUpdateNoopUsesTrackedGet(t *testing.T) {
	client := &fakeDataSafePrivateEndpointOCIClient{
		resources: map[string]datasafesdk.DataSafePrivateEndpoint{
			testDataSafePrivateEndpointID: dataSafePrivateEndpointSDK(testDataSafePrivateEndpointID, datasafesdk.LifecycleStateActive),
		},
	}
	serviceClient := newDataSafePrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := dataSafePrivateEndpointResource()
	resource.Status.Id = testDataSafePrivateEndpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDataSafePrivateEndpointID)

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 for matching readback", len(client.updateRequests))
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for tracked resource", len(client.createRequests))
	}
}

func TestDataSafePrivateEndpointCreateOrUpdateMutableUpdate(t *testing.T) {
	current := dataSafePrivateEndpointSDK(testDataSafePrivateEndpointID, datasafesdk.LifecycleStateActive)
	current.DisplayName = common.String("old-name")
	current.Description = common.String("old description")
	current.NsgIds = []string{testDataSafeNSGID}

	updated := dataSafePrivateEndpointSDK(testDataSafePrivateEndpointID, datasafesdk.LifecycleStateActive)
	updated.DisplayName = common.String("updated-name")
	updated.Description = common.String("")
	updated.NsgIds = []string{testUpdatedDataSafeNSGID}
	updated.FreeformTags = map[string]string{"env": "prod"}

	client := &fakeDataSafePrivateEndpointOCIClient{
		updateWorkRequestID: "wr-update-1",
		getResults: []dataSafePrivateEndpointGetResult{
			{response: dataSafePrivateEndpointGetResponse(current)},
			{response: dataSafePrivateEndpointGetResponse(updated)},
		},
		workRequests: map[string]datasafesdk.WorkRequest{
			"wr-update-1": dataSafePrivateEndpointWorkRequest(
				"wr-update-1",
				datasafesdk.WorkRequestStatusSucceeded,
				datasafesdk.WorkRequestOperationTypeUpdatePrivateEndpoint,
				datasafesdk.WorkRequestResourceActionTypeUpdated,
				testDataSafePrivateEndpointID,
			),
		},
	}
	serviceClient := newDataSafePrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := dataSafePrivateEndpointResource()
	resource.Status.Id = testDataSafePrivateEndpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDataSafePrivateEndpointID)
	resource.Spec.DisplayName = "updated-name"
	resource.Spec.Description = ""
	resource.Spec.NsgIds = []string{testUpdatedDataSafeNSGID}
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(client.updateRequests))
	}
	updateBody := client.updateRequests[0].UpdateDataSafePrivateEndpointDetails
	if got := dataSafePrivateEndpointStringValue(updateBody.DisplayName); got != "updated-name" {
		t.Fatalf("update displayName = %q, want updated-name", got)
	}
	if updateBody.Description == nil || *updateBody.Description != "" {
		t.Fatalf("update description = %#v, want explicit empty string", updateBody.Description)
	}
	if !reflect.DeepEqual(updateBody.NsgIds, []string{testUpdatedDataSafeNSGID}) {
		t.Fatalf("update nsgIds = %#v, want %#v", updateBody.NsgIds, []string{testUpdatedDataSafeNSGID})
	}
	if got := resource.Status.DisplayName; got != "updated-name" {
		t.Fatalf("status.displayName = %q, want updated-name", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after update work request", resource.Status.OsokStatus.Async.Current)
	}
}

func TestDataSafePrivateEndpointCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	current := dataSafePrivateEndpointSDK(testDataSafePrivateEndpointID, datasafesdk.LifecycleStateActive)
	current.PrivateEndpointIp = common.String("10.0.0.20")
	client := &fakeDataSafePrivateEndpointOCIClient{
		resources: map[string]datasafesdk.DataSafePrivateEndpoint{testDataSafePrivateEndpointID: current},
	}
	serviceClient := newDataSafePrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := dataSafePrivateEndpointResource()
	resource.Status.Id = testDataSafePrivateEndpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDataSafePrivateEndpointID)
	resource.Spec.DisplayName = "updated-name"

	_, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift")
	}
	if !strings.Contains(err.Error(), "privateEndpointIp") {
		t.Fatalf("CreateOrUpdate() error = %v, want privateEndpointIp drift", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 after create-only drift", len(client.updateRequests))
	}
}

func TestDataSafePrivateEndpointDeleteWaitsForWorkRequestAndConfirmsNotFound(t *testing.T) {
	active := dataSafePrivateEndpointSDK(testDataSafePrivateEndpointID, datasafesdk.LifecycleStateActive)
	client := &fakeDataSafePrivateEndpointOCIClient{
		deleteWorkRequestID: "wr-delete-1",
		getResults: []dataSafePrivateEndpointGetResult{
			{response: dataSafePrivateEndpointGetResponse(active)},
			{response: dataSafePrivateEndpointGetResponse(active)},
			{err: errortest.NewServiceError(404, errorutil.NotFound, "deleted")},
		},
		workRequests: map[string]datasafesdk.WorkRequest{
			"wr-delete-1": dataSafePrivateEndpointWorkRequest(
				"wr-delete-1",
				datasafesdk.WorkRequestStatusSucceeded,
				datasafesdk.WorkRequestOperationTypeDeletePrivateEndpoint,
				datasafesdk.WorkRequestResourceActionTypeDeleted,
				testDataSafePrivateEndpointID,
			),
		},
	}
	serviceClient := newDataSafePrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := dataSafePrivateEndpointResource()
	resource.Status.Id = testDataSafePrivateEndpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDataSafePrivateEndpointID)

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after confirmed not found")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("delete requests = %d, want 1", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete confirmation timestamp")
	}
}

func TestDataSafePrivateEndpointDeleteKeepsFinalizerForAuthShapedNotFound(t *testing.T) {
	active := dataSafePrivateEndpointSDK(testDataSafePrivateEndpointID, datasafesdk.LifecycleStateActive)
	client := &fakeDataSafePrivateEndpointOCIClient{
		getResults: []dataSafePrivateEndpointGetResult{
			{response: dataSafePrivateEndpointGetResponse(active)},
			{response: dataSafePrivateEndpointGetResponse(active)},
		},
		deleteErr: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
	}
	serviceClient := newDataSafePrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := dataSafePrivateEndpointResource()
	resource.Status.Id = testDataSafePrivateEndpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDataSafePrivateEndpointID)

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped 404")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 NotAuthorizedOrNotFound", err)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set for auth-shaped 404")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestDataSafePrivateEndpointDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	client := &fakeDataSafePrivateEndpointOCIClient{
		getResults: []dataSafePrivateEndpointGetResult{{
			err: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
		}},
	}
	serviceClient := newDataSafePrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := dataSafePrivateEndpointResource()
	resource.Finalizers = []string{"osok.oracle.com/finalizer"}
	resource.Status.Id = testDataSafePrivateEndpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDataSafePrivateEndpointID)

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped pre-delete read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete read")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 NotAuthorizedOrNotFound", err)
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1 pre-delete read", len(client.getRequests))
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 after auth-shaped pre-delete read", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set for auth-shaped pre-delete read")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	if got, want := resource.Finalizers, []string{"osok.oracle.com/finalizer"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("finalizers = %#v, want %#v", got, want)
	}
}

func TestDataSafePrivateEndpointDeleteWorkRequestAuthShapedReadbackKeepsFinalizer(t *testing.T) {
	client := &fakeDataSafePrivateEndpointOCIClient{
		getResults: []dataSafePrivateEndpointGetResult{{
			err: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
		}},
		workRequests: map[string]datasafesdk.WorkRequest{
			"wr-delete-done": dataSafePrivateEndpointWorkRequest(
				"wr-delete-done",
				datasafesdk.WorkRequestStatusSucceeded,
				datasafesdk.WorkRequestOperationTypeDeletePrivateEndpoint,
				datasafesdk.WorkRequestResourceActionTypeDeleted,
				testDataSafePrivateEndpointID,
			),
		},
	}
	serviceClient := newDataSafePrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := dataSafePrivateEndpointResource()
	resource.Status.Id = testDataSafePrivateEndpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(testDataSafePrivateEndpointID)
	resource.Status.OsokStatus.OpcRequestID = "opc-delete-1"
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseDelete,
		WorkRequestID:    "wr-delete-done",
		NormalizedClass:  shared.OSOKAsyncClassPending,
		RawOperationType: string(datasafesdk.WorkRequestOperationTypeDeletePrivateEndpoint),
	}

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped follow-up readback error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped delete work request readback")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous 404 NotAuthorizedOrNotFound", err)
	}
	if len(client.workRequestRequests) != 1 {
		t.Fatalf("work request calls = %d, want 1", len(client.workRequestRequests))
	}
	if len(client.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1 follow-up readback", len(client.getRequests))
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 while resuming tracked delete work request", len(client.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set for auth-shaped delete work request readback")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	assertCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending, "wr-delete-done")
}

func TestDataSafePrivateEndpointDeleteWaitsForPendingWriteWorkRequest(t *testing.T) {
	client := &fakeDataSafePrivateEndpointOCIClient{
		workRequests: map[string]datasafesdk.WorkRequest{
			"wr-create-pending": dataSafePrivateEndpointWorkRequest(
				"wr-create-pending",
				datasafesdk.WorkRequestStatusInProgress,
				datasafesdk.WorkRequestOperationTypeCreatePrivateEndpoint,
				datasafesdk.WorkRequestResourceActionTypeInProgress,
				testDataSafePrivateEndpointID,
			),
		},
	}
	serviceClient := newDataSafePrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := dataSafePrivateEndpointResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:        shared.OSOKAsyncSourceWorkRequest,
		Phase:         shared.OSOKAsyncPhaseCreate,
		WorkRequestID: "wr-create-pending",
	}

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while create work request is pending")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 while create work request is pending", len(client.deleteRequests))
	}
	assertCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create-pending")
}

func TestDataSafePrivateEndpointDeleteWaitsForSucceededWriteReadback(t *testing.T) {
	client := &fakeDataSafePrivateEndpointOCIClient{
		getResults: []dataSafePrivateEndpointGetResult{{
			err: errortest.NewServiceError(404, errorutil.NotFound, "readback lag"),
		}},
		workRequests: map[string]datasafesdk.WorkRequest{
			"wr-create-done": dataSafePrivateEndpointWorkRequest(
				"wr-create-done",
				datasafesdk.WorkRequestStatusSucceeded,
				datasafesdk.WorkRequestOperationTypeCreatePrivateEndpoint,
				datasafesdk.WorkRequestResourceActionTypeCreated,
				testUpdatedDataSafePrivateEndpointID,
			),
		},
	}
	serviceClient := newDataSafePrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := dataSafePrivateEndpointResource()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:        shared.OSOKAsyncSourceWorkRequest,
		Phase:         shared.OSOKAsyncPhaseCreate,
		WorkRequestID: "wr-create-done",
	}

	deleted, err := serviceClient.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while succeeded create readback is not visible")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 until write readback is visible", len(client.deleteRequests))
	}
	if got := resource.Status.Id; got != testUpdatedDataSafePrivateEndpointID {
		t.Fatalf("status.id = %q, want recovered work request ID %q", got, testUpdatedDataSafePrivateEndpointID)
	}
	assertCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create-done")
}

func dataSafePrivateEndpointResource() *datasafev1beta1.DataSafePrivateEndpoint {
	return &datasafev1beta1.DataSafePrivateEndpoint{
		Spec: datasafev1beta1.DataSafePrivateEndpointSpec{
			DisplayName:       testDataSafePrivateEndpointName,
			CompartmentId:     testDataSafeCompartmentID,
			VcnId:             testDataSafeVCNID,
			SubnetId:          testDataSafeSubnetID,
			PrivateEndpointIp: testDataSafePrivateEndpointIP,
			Description:       "test private endpoint",
			NsgIds:            []string{testDataSafeNSGID},
			FreeformTags:      map[string]string{"env": "test"},
			DefinedTags:       map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func dataSafePrivateEndpointSDK(
	id string,
	lifecycle datasafesdk.LifecycleStateEnum,
) datasafesdk.DataSafePrivateEndpoint {
	now := common.SDKTime{Time: time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)}
	return datasafesdk.DataSafePrivateEndpoint{
		Id:                common.String(id),
		DisplayName:       common.String(testDataSafePrivateEndpointName),
		CompartmentId:     common.String(testDataSafeCompartmentID),
		VcnId:             common.String(testDataSafeVCNID),
		SubnetId:          common.String(testDataSafeSubnetID),
		PrivateEndpointId: common.String(testDataSafeUnderlyingPrivateID),
		PrivateEndpointIp: common.String(testDataSafePrivateEndpointIP),
		EndpointFqdn:      common.String("datasafe-endpoint.privatesubnet.oraclevcn.com"),
		Description:       common.String("test private endpoint"),
		TimeCreated:       &now,
		LifecycleState:    lifecycle,
		NsgIds:            []string{testDataSafeNSGID},
		FreeformTags:      map[string]string{"env": "test"},
		DefinedTags:       map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:        map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func dataSafePrivateEndpointFromCreateDetails(
	id string,
	details datasafesdk.CreateDataSafePrivateEndpointDetails,
) datasafesdk.DataSafePrivateEndpoint {
	return datasafesdk.DataSafePrivateEndpoint{
		Id:                common.String(id),
		DisplayName:       details.DisplayName,
		CompartmentId:     details.CompartmentId,
		VcnId:             details.VcnId,
		SubnetId:          details.SubnetId,
		PrivateEndpointId: common.String(testDataSafeUnderlyingPrivateID),
		PrivateEndpointIp: details.PrivateEndpointIp,
		LifecycleState:    datasafesdk.LifecycleStateCreating,
		Description:       details.Description,
		NsgIds:            append([]string(nil), details.NsgIds...),
		FreeformTags:      details.FreeformTags,
		DefinedTags:       details.DefinedTags,
	}
}

func dataSafePrivateEndpointGetResponse(
	resource datasafesdk.DataSafePrivateEndpoint,
) datasafesdk.GetDataSafePrivateEndpointResponse {
	return datasafesdk.GetDataSafePrivateEndpointResponse{
		DataSafePrivateEndpoint: resource,
		OpcRequestId:            common.String("opc-get-1"),
	}
}

func dataSafePrivateEndpointSummaryFromSDK(
	resource datasafesdk.DataSafePrivateEndpoint,
) datasafesdk.DataSafePrivateEndpointSummary {
	return datasafesdk.DataSafePrivateEndpointSummary{
		Id:                resource.Id,
		DisplayName:       resource.DisplayName,
		CompartmentId:     resource.CompartmentId,
		VcnId:             resource.VcnId,
		SubnetId:          resource.SubnetId,
		PrivateEndpointId: resource.PrivateEndpointId,
		Description:       resource.Description,
		TimeCreated:       resource.TimeCreated,
		LifecycleState:    resource.LifecycleState,
		FreeformTags:      resource.FreeformTags,
		DefinedTags:       resource.DefinedTags,
		SystemTags:        resource.SystemTags,
	}
}

func dataSafePrivateEndpointWorkRequest(
	id string,
	status datasafesdk.WorkRequestStatusEnum,
	operationType datasafesdk.WorkRequestOperationTypeEnum,
	actionType datasafesdk.WorkRequestResourceActionTypeEnum,
	resourceID string,
) datasafesdk.WorkRequest {
	return datasafesdk.WorkRequest{
		Id:              common.String(id),
		Status:          status,
		OperationType:   operationType,
		CompartmentId:   common.String(testDataSafeCompartmentID),
		PercentComplete: common.Float32(100),
		Resources: []datasafesdk.WorkRequestResource{{
			EntityType: common.String("dataSafePrivateEndpoint"),
			ActionType: actionType,
			Identifier: common.String(resourceID),
		}},
	}
}

func dataSafePrivateEndpointMatchesListRequest(
	resource datasafesdk.DataSafePrivateEndpoint,
	request datasafesdk.ListDataSafePrivateEndpointsRequest,
) bool {
	return dataSafePrivateEndpointRequestStringMatches(request.CompartmentId, resource.CompartmentId) &&
		dataSafePrivateEndpointRequestStringMatches(request.DisplayName, resource.DisplayName) &&
		dataSafePrivateEndpointRequestStringMatches(request.VcnId, resource.VcnId)
}

func dataSafePrivateEndpointRequestStringMatches(requestValue *string, resourceValue *string) bool {
	return requestValue == nil || dataSafePrivateEndpointStringValue(resourceValue) == dataSafePrivateEndpointStringValue(requestValue)
}

func assertCurrentWorkRequest(
	t *testing.T,
	resource *datasafev1beta1.DataSafePrivateEndpoint,
	phase shared.OSOKAsyncPhase,
	class shared.OSOKAsyncNormalizedClass,
	workRequestID string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil, want work request")
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
}

func assertContainsAll(t *testing.T, field string, got []string, want ...string) {
	t.Helper()
	values := map[string]bool{}
	for _, value := range got {
		values[value] = true
	}
	for _, value := range want {
		if !values[value] {
			t.Fatalf("%s = %#v, want to contain %q", field, got, value)
		}
	}
}
