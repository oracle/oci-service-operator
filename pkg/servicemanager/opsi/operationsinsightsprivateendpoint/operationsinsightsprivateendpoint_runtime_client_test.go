/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package operationsinsightsprivateendpoint

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testPrivateEndpointID          = "ocid1.opsiprivateendpoint.oc1..existing"
	testUpdatedPrivateEndpointID   = "ocid1.opsiprivateendpoint.oc1..updated"
	testCompartmentID              = "ocid1.compartment.oc1..opsi"
	testVCNID                      = "ocid1.vcn.oc1..opsi"
	testSubnetID                   = "ocid1.subnet.oc1..opsi"
	testNSGID                      = "ocid1.networksecuritygroup.oc1..opsi"
	testUpdatedNSGID               = "ocid1.networksecuritygroup.oc1..updated"
	testOperationsInsightsEndpoint = "opsi-endpoint"
)

type fakeOperationsInsightsPrivateEndpointOCIClient struct {
	resources map[string]opsisdk.OperationsInsightsPrivateEndpoint

	createRequests      []opsisdk.CreateOperationsInsightsPrivateEndpointRequest
	getRequests         []opsisdk.GetOperationsInsightsPrivateEndpointRequest
	listRequests        []opsisdk.ListOperationsInsightsPrivateEndpointsRequest
	updateRequests      []opsisdk.UpdateOperationsInsightsPrivateEndpointRequest
	deleteRequests      []opsisdk.DeleteOperationsInsightsPrivateEndpointRequest
	workRequestRequests []opsisdk.GetWorkRequestRequest

	getResults    []operationsInsightsPrivateEndpointGetResult
	listResponses []opsisdk.ListOperationsInsightsPrivateEndpointsResponse
	workRequests  map[string]opsisdk.WorkRequest

	createErr error
	updateErr error
	deleteErr error

	createWorkRequestID string
	updateWorkRequestID string
	deleteWorkRequestID string
}

type operationsInsightsPrivateEndpointGetResult struct {
	response opsisdk.GetOperationsInsightsPrivateEndpointResponse
	err      error
}

func (f *fakeOperationsInsightsPrivateEndpointOCIClient) CreateOperationsInsightsPrivateEndpoint(
	_ context.Context,
	request opsisdk.CreateOperationsInsightsPrivateEndpointRequest,
) (opsisdk.CreateOperationsInsightsPrivateEndpointResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createErr != nil {
		return opsisdk.CreateOperationsInsightsPrivateEndpointResponse{}, f.createErr
	}

	id := testPrivateEndpointID
	if f.createWorkRequestID != "" {
		id = strings.TrimSuffix(testPrivateEndpointID, "existing") + "created"
	}
	resource := operationsInsightsPrivateEndpointFromCreateDetails(id, request.CreateOperationsInsightsPrivateEndpointDetails)
	f.ensureResources()[id] = resource
	return opsisdk.CreateOperationsInsightsPrivateEndpointResponse{
		OperationsInsightsPrivateEndpoint: resource,
		OpcWorkRequestId:                  common.String(f.createWorkRequestID),
		OpcRequestId:                      common.String("opc-create-1"),
	}, nil
}

func (f *fakeOperationsInsightsPrivateEndpointOCIClient) GetOperationsInsightsPrivateEndpoint(
	_ context.Context,
	request opsisdk.GetOperationsInsightsPrivateEndpointRequest,
) (opsisdk.GetOperationsInsightsPrivateEndpointResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if len(f.getResults) > 0 {
		result := f.getResults[0]
		f.getResults = f.getResults[1:]
		return result.response, result.err
	}

	id := stringValue(request.OperationsInsightsPrivateEndpointId)
	resource, ok := f.resources[id]
	if !ok {
		return opsisdk.GetOperationsInsightsPrivateEndpointResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
	}
	return opsisdk.GetOperationsInsightsPrivateEndpointResponse{
		OperationsInsightsPrivateEndpoint: resource,
		OpcRequestId:                      common.String("opc-get-1"),
	}, nil
}

func (f *fakeOperationsInsightsPrivateEndpointOCIClient) ListOperationsInsightsPrivateEndpoints(
	_ context.Context,
	request opsisdk.ListOperationsInsightsPrivateEndpointsRequest,
) (opsisdk.ListOperationsInsightsPrivateEndpointsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if len(f.listResponses) > 0 {
		response := f.listResponses[0]
		f.listResponses = f.listResponses[1:]
		return response, nil
	}

	var items []opsisdk.OperationsInsightsPrivateEndpointSummary
	for _, resource := range f.resources {
		if operationsInsightsPrivateEndpointMatchesListRequest(resource, request) {
			items = append(items, operationsInsightsPrivateEndpointSummaryFromSDK(resource))
		}
	}
	return opsisdk.ListOperationsInsightsPrivateEndpointsResponse{
		OperationsInsightsPrivateEndpointCollection: opsisdk.OperationsInsightsPrivateEndpointCollection{Items: items},
		OpcRequestId: common.String("opc-list-1"),
	}, nil
}

func (f *fakeOperationsInsightsPrivateEndpointOCIClient) UpdateOperationsInsightsPrivateEndpoint(
	_ context.Context,
	request opsisdk.UpdateOperationsInsightsPrivateEndpointRequest,
) (opsisdk.UpdateOperationsInsightsPrivateEndpointResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateErr != nil {
		return opsisdk.UpdateOperationsInsightsPrivateEndpointResponse{}, f.updateErr
	}
	return opsisdk.UpdateOperationsInsightsPrivateEndpointResponse{
		OpcWorkRequestId: common.String(f.updateWorkRequestID),
		OpcRequestId:     common.String("opc-update-1"),
	}, nil
}

func (f *fakeOperationsInsightsPrivateEndpointOCIClient) DeleteOperationsInsightsPrivateEndpoint(
	_ context.Context,
	request opsisdk.DeleteOperationsInsightsPrivateEndpointRequest,
) (opsisdk.DeleteOperationsInsightsPrivateEndpointResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteErr != nil {
		return opsisdk.DeleteOperationsInsightsPrivateEndpointResponse{}, f.deleteErr
	}
	return opsisdk.DeleteOperationsInsightsPrivateEndpointResponse{
		OpcWorkRequestId: common.String(f.deleteWorkRequestID),
		OpcRequestId:     common.String("opc-delete-1"),
	}, nil
}

func (f *fakeOperationsInsightsPrivateEndpointOCIClient) GetWorkRequest(
	_ context.Context,
	request opsisdk.GetWorkRequestRequest,
) (opsisdk.GetWorkRequestResponse, error) {
	f.workRequestRequests = append(f.workRequestRequests, request)
	workRequest, ok := f.workRequests[stringValue(request.WorkRequestId)]
	if !ok {
		return opsisdk.GetWorkRequestResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "work request not found")
	}
	return opsisdk.GetWorkRequestResponse{
		WorkRequest:  workRequest,
		OpcRequestId: common.String("opc-work-request-1"),
	}, nil
}

func (f *fakeOperationsInsightsPrivateEndpointOCIClient) ensureResources() map[string]opsisdk.OperationsInsightsPrivateEndpoint {
	if f.resources == nil {
		f.resources = map[string]opsisdk.OperationsInsightsPrivateEndpoint{}
	}
	return f.resources
}

func TestOperationsInsightsPrivateEndpointRuntimeHooks(t *testing.T) {
	hooks := newOperationsInsightsPrivateEndpointRuntimeHooksWithOCIClient(&fakeOperationsInsightsPrivateEndpointOCIClient{})
	applyOperationsInsightsPrivateEndpointRuntimeHooks(&OperationsInsightsPrivateEndpointServiceManager{}, &hooks, nil, nil)

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
	assertContainsAll(t, "Mutation.ForceNew", hooks.Semantics.Mutation.ForceNew, "compartmentId", "vcnId", "subnetId", "isUsedForRacDbs")
	assertContainsAll(t, "List.MatchFields", hooks.Semantics.List.MatchFields, "displayName", "compartmentId", "vcnId", "subnetId", "isUsedForRacDbs")
	if hooks.BuildCreateBody == nil || hooks.BuildUpdateBody == nil {
		t.Fatal("BuildCreateBody/BuildUpdateBody = nil, want resource-specific body builders")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("DeleteHooks.HandleError = nil, want auth-shaped not-found guard")
	}

	resource := operationsInsightsPrivateEndpointResource()
	resource.Spec.IsUsedForRacDbs = false
	body, err := hooks.BuildCreateBody(context.Background(), resource, "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	createBody := body.(opsisdk.CreateOperationsInsightsPrivateEndpointDetails)
	if createBody.IsUsedForRacDbs == nil {
		t.Fatal("Create body IsUsedForRacDbs = nil, want explicit false preserved")
	}
	if *createBody.IsUsedForRacDbs {
		t.Fatal("Create body IsUsedForRacDbs = true, want false")
	}
}

func TestOperationsInsightsPrivateEndpointCreateOrUpdateCreatesAndTracksWorkRequest(t *testing.T) {
	client := &fakeOperationsInsightsPrivateEndpointOCIClient{
		createWorkRequestID: "wr-create-1",
		workRequests: map[string]opsisdk.WorkRequest{
			"wr-create-1": operationsInsightsPrivateEndpointWorkRequest(
				"wr-create-1",
				opsisdk.OperationStatusSucceeded,
				opsisdk.OperationTypeCreatePrivateEndpoint,
				opsisdk.ActionTypeCreated,
				testPrivateEndpointID,
			),
		},
		getResults: []operationsInsightsPrivateEndpointGetResult{{
			response: operationsInsightsPrivateEndpointGetResponse(operationsInsightsPrivateEndpointSDK(testPrivateEndpointID, opsisdk.OperationsInsightsPrivateEndpointLifecycleStateActive)),
		}},
	}
	serviceClient := newOperationsInsightsPrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := operationsInsightsPrivateEndpointResource()

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
	if got := string(resource.Status.OsokStatus.Ocid); got != testPrivateEndpointID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testPrivateEndpointID)
	}
	if got := resource.Status.Id; got != testPrivateEndpointID {
		t.Fatalf("status.id = %q, want %q", got, testPrivateEndpointID)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after succeeded work request readback", resource.Status.OsokStatus.Async.Current)
	}
}

func TestOperationsInsightsPrivateEndpointCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	existing := operationsInsightsPrivateEndpointSDK(testPrivateEndpointID, opsisdk.OperationsInsightsPrivateEndpointLifecycleStateActive)
	client := &fakeOperationsInsightsPrivateEndpointOCIClient{
		resources: map[string]opsisdk.OperationsInsightsPrivateEndpoint{
			testPrivateEndpointID: existing,
		},
		listResponses: []opsisdk.ListOperationsInsightsPrivateEndpointsResponse{
			{
				OperationsInsightsPrivateEndpointCollection: opsisdk.OperationsInsightsPrivateEndpointCollection{},
				OpcNextPage: common.String("page-2"),
			},
			{
				OperationsInsightsPrivateEndpointCollection: opsisdk.OperationsInsightsPrivateEndpointCollection{
					Items: []opsisdk.OperationsInsightsPrivateEndpointSummary{operationsInsightsPrivateEndpointSummaryFromSDK(existing)},
				},
			},
		},
	}
	serviceClient := newOperationsInsightsPrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := operationsInsightsPrivateEndpointResource()

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
	if got := stringValue(client.listRequests[1].Page); got != "page-2" {
		t.Fatalf("second list page = %q, want page-2", got)
	}
	if client.listRequests[0].IsUsedForRacDbs == nil || *client.listRequests[0].IsUsedForRacDbs {
		t.Fatalf("first list IsUsedForRacDbs = %#v, want explicit false filter", client.listRequests[0].IsUsedForRacDbs)
	}
	if got := resource.Status.Id; got != testPrivateEndpointID {
		t.Fatalf("status.id = %q, want bound ID %q", got, testPrivateEndpointID)
	}
}

func TestOperationsInsightsPrivateEndpointCreateOrUpdateBindsWhenListOmitsDeprecatedRacFlag(t *testing.T) {
	existing := operationsInsightsPrivateEndpointSDK(testPrivateEndpointID, opsisdk.OperationsInsightsPrivateEndpointLifecycleStateActive)
	existing.IsUsedForRacDbs = nil
	summary := operationsInsightsPrivateEndpointSummaryFromSDK(existing)
	summary.IsUsedForRacDbs = nil
	client := &fakeOperationsInsightsPrivateEndpointOCIClient{
		resources: map[string]opsisdk.OperationsInsightsPrivateEndpoint{
			testPrivateEndpointID: existing,
		},
		listResponses: []opsisdk.ListOperationsInsightsPrivateEndpointsResponse{{
			OperationsInsightsPrivateEndpointCollection: opsisdk.OperationsInsightsPrivateEndpointCollection{
				Items: []opsisdk.OperationsInsightsPrivateEndpointSummary{summary},
			},
		}},
	}
	serviceClient := newOperationsInsightsPrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := operationsInsightsPrivateEndpointResource()
	resource.Spec.IsUsedForRacDbs = false

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 for bind with omitted deprecated flag", len(client.createRequests))
	}
	if got := resource.Status.Id; got != testPrivateEndpointID {
		t.Fatalf("status.id = %q, want bound ID %q", got, testPrivateEndpointID)
	}
}

func TestOperationsInsightsPrivateEndpointCreateOrUpdateNoopUsesTrackedGet(t *testing.T) {
	client := &fakeOperationsInsightsPrivateEndpointOCIClient{
		resources: map[string]opsisdk.OperationsInsightsPrivateEndpoint{
			testPrivateEndpointID: operationsInsightsPrivateEndpointSDK(testPrivateEndpointID, opsisdk.OperationsInsightsPrivateEndpointLifecycleStateActive),
		},
	}
	serviceClient := newOperationsInsightsPrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := operationsInsightsPrivateEndpointResource()
	resource.Status.Id = testPrivateEndpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(testPrivateEndpointID)

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

func TestOperationsInsightsPrivateEndpointCreateOrUpdateNoopAllowsOmittedDeprecatedRacFlag(t *testing.T) {
	current := operationsInsightsPrivateEndpointSDK(testPrivateEndpointID, opsisdk.OperationsInsightsPrivateEndpointLifecycleStateActive)
	current.IsUsedForRacDbs = nil
	client := &fakeOperationsInsightsPrivateEndpointOCIClient{
		resources: map[string]opsisdk.OperationsInsightsPrivateEndpoint{
			testPrivateEndpointID: current,
		},
	}
	serviceClient := newOperationsInsightsPrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := operationsInsightsPrivateEndpointResource()
	resource.Spec.IsUsedForRacDbs = false
	resource.Status.Id = testPrivateEndpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(testPrivateEndpointID)

	response, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 for omitted deprecated false readback", len(client.updateRequests))
	}
}

func TestOperationsInsightsPrivateEndpointCreateOrUpdateMutableUpdate(t *testing.T) {
	current := operationsInsightsPrivateEndpointSDK(testPrivateEndpointID, opsisdk.OperationsInsightsPrivateEndpointLifecycleStateActive)
	current.DisplayName = common.String("old-name")
	current.Description = common.String("old description")
	current.NsgIds = []string{testNSGID}

	updated := operationsInsightsPrivateEndpointSDK(testPrivateEndpointID, opsisdk.OperationsInsightsPrivateEndpointLifecycleStateActive)
	updated.DisplayName = common.String("updated-name")
	updated.Description = common.String("")
	updated.NsgIds = []string{testUpdatedNSGID}
	updated.FreeformTags = map[string]string{"env": "prod"}

	client := &fakeOperationsInsightsPrivateEndpointOCIClient{
		updateWorkRequestID: "wr-update-1",
		getResults: []operationsInsightsPrivateEndpointGetResult{
			{response: operationsInsightsPrivateEndpointGetResponse(current)},
			{response: operationsInsightsPrivateEndpointGetResponse(updated)},
		},
		workRequests: map[string]opsisdk.WorkRequest{
			"wr-update-1": operationsInsightsPrivateEndpointWorkRequest(
				"wr-update-1",
				opsisdk.OperationStatusSucceeded,
				opsisdk.OperationTypeUpdatePrivateEndpoint,
				opsisdk.ActionTypeUpdated,
				testPrivateEndpointID,
			),
		},
	}
	serviceClient := newOperationsInsightsPrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := operationsInsightsPrivateEndpointResource()
	resource.Status.Id = testPrivateEndpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(testPrivateEndpointID)
	resource.Spec.DisplayName = "updated-name"
	resource.Spec.Description = ""
	resource.Spec.NsgIds = []string{testUpdatedNSGID}
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
	updateBody := client.updateRequests[0].UpdateOperationsInsightsPrivateEndpointDetails
	if got := stringValue(updateBody.DisplayName); got != "updated-name" {
		t.Fatalf("update displayName = %q, want updated-name", got)
	}
	if updateBody.Description == nil || *updateBody.Description != "" {
		t.Fatalf("update description = %#v, want explicit empty string", updateBody.Description)
	}
	if !reflect.DeepEqual(updateBody.NsgIds, []string{testUpdatedNSGID}) {
		t.Fatalf("update nsgIds = %#v, want %#v", updateBody.NsgIds, []string{testUpdatedNSGID})
	}
	if got := resource.Status.DisplayName; got != "updated-name" {
		t.Fatalf("status.displayName = %q, want updated-name", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after update work request", resource.Status.OsokStatus.Async.Current)
	}
}

func TestOperationsInsightsPrivateEndpointCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	current := operationsInsightsPrivateEndpointSDK(testPrivateEndpointID, opsisdk.OperationsInsightsPrivateEndpointLifecycleStateActive)
	current.IsUsedForRacDbs = common.Bool(true)
	client := &fakeOperationsInsightsPrivateEndpointOCIClient{
		resources: map[string]opsisdk.OperationsInsightsPrivateEndpoint{testPrivateEndpointID: current},
	}
	serviceClient := newOperationsInsightsPrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := operationsInsightsPrivateEndpointResource()
	resource.Status.Id = testPrivateEndpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(testPrivateEndpointID)
	resource.Spec.DisplayName = "updated-name"
	resource.Spec.IsUsedForRacDbs = false

	_, err := serviceClient.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift")
	}
	if !strings.Contains(err.Error(), "isUsedForRacDbs") {
		t.Fatalf("CreateOrUpdate() error = %v, want isUsedForRacDbs drift", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 after create-only drift", len(client.updateRequests))
	}
}

func TestOperationsInsightsPrivateEndpointDeleteWaitsForWorkRequestAndConfirmsNotFound(t *testing.T) {
	active := operationsInsightsPrivateEndpointSDK(testPrivateEndpointID, opsisdk.OperationsInsightsPrivateEndpointLifecycleStateActive)
	client := &fakeOperationsInsightsPrivateEndpointOCIClient{
		deleteWorkRequestID: "wr-delete-1",
		getResults: []operationsInsightsPrivateEndpointGetResult{
			{response: operationsInsightsPrivateEndpointGetResponse(active)},
			{err: errortest.NewServiceError(404, errorutil.NotFound, "deleted")},
		},
		workRequests: map[string]opsisdk.WorkRequest{
			"wr-delete-1": operationsInsightsPrivateEndpointWorkRequest(
				"wr-delete-1",
				opsisdk.OperationStatusSucceeded,
				opsisdk.OperationTypeDeletePrivateEndpoint,
				opsisdk.ActionTypeDeleted,
				testPrivateEndpointID,
			),
		},
	}
	serviceClient := newOperationsInsightsPrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := operationsInsightsPrivateEndpointResource()
	resource.Status.Id = testPrivateEndpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(testPrivateEndpointID)

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

func TestOperationsInsightsPrivateEndpointDeleteKeepsFinalizerForAuthShapedNotFound(t *testing.T) {
	active := operationsInsightsPrivateEndpointSDK(testPrivateEndpointID, opsisdk.OperationsInsightsPrivateEndpointLifecycleStateActive)
	client := &fakeOperationsInsightsPrivateEndpointOCIClient{
		getResults: []operationsInsightsPrivateEndpointGetResult{
			{response: operationsInsightsPrivateEndpointGetResponse(active)},
		},
		deleteErr: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
	}
	serviceClient := newOperationsInsightsPrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := operationsInsightsPrivateEndpointResource()
	resource.Status.Id = testPrivateEndpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(testPrivateEndpointID)

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

func TestOperationsInsightsPrivateEndpointDeleteWorkRequestAuthShapedReadbackKeepsFinalizer(t *testing.T) {
	client := &fakeOperationsInsightsPrivateEndpointOCIClient{
		getResults: []operationsInsightsPrivateEndpointGetResult{{
			err: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
		}},
		workRequests: map[string]opsisdk.WorkRequest{
			"wr-delete-done": operationsInsightsPrivateEndpointWorkRequest(
				"wr-delete-done",
				opsisdk.OperationStatusSucceeded,
				opsisdk.OperationTypeDeletePrivateEndpoint,
				opsisdk.ActionTypeDeleted,
				testPrivateEndpointID,
			),
		},
	}
	serviceClient := newOperationsInsightsPrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := operationsInsightsPrivateEndpointResource()
	resource.Status.Id = testPrivateEndpointID
	resource.Status.OsokStatus.Ocid = shared.OCID(testPrivateEndpointID)
	resource.Status.OsokStatus.OpcRequestID = "opc-delete-1"
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseDelete,
		WorkRequestID:    "wr-delete-done",
		NormalizedClass:  shared.OSOKAsyncClassPending,
		RawOperationType: string(opsisdk.OperationTypeDeletePrivateEndpoint),
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

func TestOperationsInsightsPrivateEndpointDeleteWaitsForPendingWriteWorkRequest(t *testing.T) {
	client := &fakeOperationsInsightsPrivateEndpointOCIClient{
		workRequests: map[string]opsisdk.WorkRequest{
			"wr-create-pending": operationsInsightsPrivateEndpointWorkRequest(
				"wr-create-pending",
				opsisdk.OperationStatusInProgress,
				opsisdk.OperationTypeCreatePrivateEndpoint,
				opsisdk.ActionTypeInProgress,
				testPrivateEndpointID,
			),
		},
	}
	serviceClient := newOperationsInsightsPrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := operationsInsightsPrivateEndpointResource()
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

func TestOperationsInsightsPrivateEndpointDeleteWaitsForSucceededWriteReadback(t *testing.T) {
	client := &fakeOperationsInsightsPrivateEndpointOCIClient{
		getResults: []operationsInsightsPrivateEndpointGetResult{{
			err: errortest.NewServiceError(404, errorutil.NotFound, "readback lag"),
		}},
		workRequests: map[string]opsisdk.WorkRequest{
			"wr-create-done": operationsInsightsPrivateEndpointWorkRequest(
				"wr-create-done",
				opsisdk.OperationStatusSucceeded,
				opsisdk.OperationTypeCreatePrivateEndpoint,
				opsisdk.ActionTypeCreated,
				testUpdatedPrivateEndpointID,
			),
		},
	}
	serviceClient := newOperationsInsightsPrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
	resource := operationsInsightsPrivateEndpointResource()
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
	if got := resource.Status.Id; got != testUpdatedPrivateEndpointID {
		t.Fatalf("status.id = %q, want recovered work request ID %q", got, testUpdatedPrivateEndpointID)
	}
	assertCurrentWorkRequest(t, resource, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending, "wr-create-done")
}

func operationsInsightsPrivateEndpointResource() *opsiv1beta1.OperationsInsightsPrivateEndpoint {
	return &opsiv1beta1.OperationsInsightsPrivateEndpoint{
		Spec: opsiv1beta1.OperationsInsightsPrivateEndpointSpec{
			DisplayName:     testOperationsInsightsEndpoint,
			CompartmentId:   testCompartmentID,
			VcnId:           testVCNID,
			SubnetId:        testSubnetID,
			IsUsedForRacDbs: false,
			Description:     "test private endpoint",
			NsgIds:          []string{testNSGID},
			FreeformTags:    map[string]string{"env": "test"},
			DefinedTags:     map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func operationsInsightsPrivateEndpointSDK(
	id string,
	lifecycle opsisdk.OperationsInsightsPrivateEndpointLifecycleStateEnum,
) opsisdk.OperationsInsightsPrivateEndpoint {
	now := common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
	return opsisdk.OperationsInsightsPrivateEndpoint{
		Id:                           common.String(id),
		DisplayName:                  common.String(testOperationsInsightsEndpoint),
		CompartmentId:                common.String(testCompartmentID),
		VcnId:                        common.String(testVCNID),
		SubnetId:                     common.String(testSubnetID),
		LifecycleState:               lifecycle,
		PrivateIp:                    common.String("10.0.0.10"),
		Description:                  common.String("test private endpoint"),
		TimeCreated:                  &now,
		LifecycleDetails:             common.String("ready"),
		PrivateEndpointStatusDetails: common.String("connected"),
		IsUsedForRacDbs:              common.Bool(false),
		NsgIds:                       []string{testNSGID},
		FreeformTags:                 map[string]string{"env": "test"},
		DefinedTags:                  map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:                   map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func operationsInsightsPrivateEndpointFromCreateDetails(
	id string,
	details opsisdk.CreateOperationsInsightsPrivateEndpointDetails,
) opsisdk.OperationsInsightsPrivateEndpoint {
	return opsisdk.OperationsInsightsPrivateEndpoint{
		Id:              common.String(id),
		DisplayName:     details.DisplayName,
		CompartmentId:   details.CompartmentId,
		VcnId:           details.VcnId,
		SubnetId:        details.SubnetId,
		LifecycleState:  opsisdk.OperationsInsightsPrivateEndpointLifecycleStateCreating,
		Description:     details.Description,
		IsUsedForRacDbs: details.IsUsedForRacDbs,
		NsgIds:          append([]string(nil), details.NsgIds...),
		FreeformTags:    details.FreeformTags,
		DefinedTags:     details.DefinedTags,
	}
}

func operationsInsightsPrivateEndpointGetResponse(
	resource opsisdk.OperationsInsightsPrivateEndpoint,
) opsisdk.GetOperationsInsightsPrivateEndpointResponse {
	return opsisdk.GetOperationsInsightsPrivateEndpointResponse{
		OperationsInsightsPrivateEndpoint: resource,
		OpcRequestId:                      common.String("opc-get-1"),
	}
}

func operationsInsightsPrivateEndpointSummaryFromSDK(
	resource opsisdk.OperationsInsightsPrivateEndpoint,
) opsisdk.OperationsInsightsPrivateEndpointSummary {
	return opsisdk.OperationsInsightsPrivateEndpointSummary{
		Id:                           resource.Id,
		DisplayName:                  resource.DisplayName,
		CompartmentId:                resource.CompartmentId,
		VcnId:                        resource.VcnId,
		SubnetId:                     resource.SubnetId,
		TimeCreated:                  resource.TimeCreated,
		LifecycleState:               resource.LifecycleState,
		IsUsedForRacDbs:              resource.IsUsedForRacDbs,
		Description:                  resource.Description,
		FreeformTags:                 resource.FreeformTags,
		DefinedTags:                  resource.DefinedTags,
		SystemTags:                   resource.SystemTags,
		LifecycleDetails:             resource.LifecycleDetails,
		PrivateEndpointStatusDetails: resource.PrivateEndpointStatusDetails,
	}
}

func operationsInsightsPrivateEndpointWorkRequest(
	id string,
	status opsisdk.OperationStatusEnum,
	operationType opsisdk.OperationTypeEnum,
	actionType opsisdk.ActionTypeEnum,
	resourceID string,
) opsisdk.WorkRequest {
	return opsisdk.WorkRequest{
		Id:              common.String(id),
		Status:          status,
		OperationType:   operationType,
		CompartmentId:   common.String(testCompartmentID),
		PercentComplete: common.Float32(100),
		Resources: []opsisdk.WorkRequestResource{{
			EntityType: common.String("operationsInsightsPrivateEndpoint"),
			ActionType: actionType,
			Identifier: common.String(resourceID),
		}},
	}
}

func operationsInsightsPrivateEndpointMatchesListRequest(
	resource opsisdk.OperationsInsightsPrivateEndpoint,
	request opsisdk.ListOperationsInsightsPrivateEndpointsRequest,
) bool {
	return operationsInsightsPrivateEndpointRequestStringMatches(request.CompartmentId, resource.CompartmentId) &&
		operationsInsightsPrivateEndpointRequestStringMatches(request.DisplayName, resource.DisplayName) &&
		operationsInsightsPrivateEndpointRequestStringMatches(request.OpsiPrivateEndpointId, resource.Id) &&
		operationsInsightsPrivateEndpointRequestBoolMatches(request.IsUsedForRacDbs, resource.IsUsedForRacDbs) &&
		operationsInsightsPrivateEndpointRequestStringMatches(request.VcnId, resource.VcnId)
}

func operationsInsightsPrivateEndpointRequestStringMatches(requestValue *string, resourceValue *string) bool {
	return requestValue == nil || stringValue(resourceValue) == stringValue(requestValue)
}

func operationsInsightsPrivateEndpointRequestBoolMatches(requestValue *bool, resourceValue *bool) bool {
	if requestValue == nil {
		return true
	}
	if resourceValue == nil {
		return !*requestValue
	}
	return *resourceValue == *requestValue
}

func assertCurrentWorkRequest(
	t *testing.T,
	resource *opsiv1beta1.OperationsInsightsPrivateEndpoint,
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

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
