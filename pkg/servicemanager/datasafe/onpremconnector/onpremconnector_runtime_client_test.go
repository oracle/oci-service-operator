/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package onpremconnector

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
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testOnPremConnectorID            = "ocid1.datasafeonpremconnector.oc1..connector"
	testOnPremConnectorCompartmentID = "ocid1.compartment.oc1..datasafe"
	testOnPremConnectorDisplayName   = "customer-connector"
	testOnPremConnectorDescription   = "customer on-prem connector"
)

type fakeOnPremConnectorOCIClient struct {
	createFn func(context.Context, datasafesdk.CreateOnPremConnectorRequest) (datasafesdk.CreateOnPremConnectorResponse, error)
	getFn    func(context.Context, datasafesdk.GetOnPremConnectorRequest) (datasafesdk.GetOnPremConnectorResponse, error)
	listFn   func(context.Context, datasafesdk.ListOnPremConnectorsRequest) (datasafesdk.ListOnPremConnectorsResponse, error)
	updateFn func(context.Context, datasafesdk.UpdateOnPremConnectorRequest) (datasafesdk.UpdateOnPremConnectorResponse, error)
	deleteFn func(context.Context, datasafesdk.DeleteOnPremConnectorRequest) (datasafesdk.DeleteOnPremConnectorResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int
}

func (f *fakeOnPremConnectorOCIClient) CreateOnPremConnector(
	ctx context.Context,
	request datasafesdk.CreateOnPremConnectorRequest,
) (datasafesdk.CreateOnPremConnectorResponse, error) {
	f.createCalls++
	if f.createFn != nil {
		return f.createFn(ctx, request)
	}
	return datasafesdk.CreateOnPremConnectorResponse{}, nil
}

func (f *fakeOnPremConnectorOCIClient) GetOnPremConnector(
	ctx context.Context,
	request datasafesdk.GetOnPremConnectorRequest,
) (datasafesdk.GetOnPremConnectorResponse, error) {
	f.getCalls++
	if f.getFn != nil {
		return f.getFn(ctx, request)
	}
	return datasafesdk.GetOnPremConnectorResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "OnPremConnector is missing")
}

func (f *fakeOnPremConnectorOCIClient) ListOnPremConnectors(
	ctx context.Context,
	request datasafesdk.ListOnPremConnectorsRequest,
) (datasafesdk.ListOnPremConnectorsResponse, error) {
	f.listCalls++
	if f.listFn != nil {
		return f.listFn(ctx, request)
	}
	return datasafesdk.ListOnPremConnectorsResponse{}, nil
}

func (f *fakeOnPremConnectorOCIClient) UpdateOnPremConnector(
	ctx context.Context,
	request datasafesdk.UpdateOnPremConnectorRequest,
) (datasafesdk.UpdateOnPremConnectorResponse, error) {
	f.updateCalls++
	if f.updateFn != nil {
		return f.updateFn(ctx, request)
	}
	return datasafesdk.UpdateOnPremConnectorResponse{}, nil
}

func (f *fakeOnPremConnectorOCIClient) DeleteOnPremConnector(
	ctx context.Context,
	request datasafesdk.DeleteOnPremConnectorRequest,
) (datasafesdk.DeleteOnPremConnectorResponse, error) {
	f.deleteCalls++
	if f.deleteFn != nil {
		return f.deleteFn(ctx, request)
	}
	return datasafesdk.DeleteOnPremConnectorResponse{}, nil
}

func TestOnPremConnectorRuntimeHooksConfigured(t *testing.T) {
	hooks := newOnPremConnectorDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	applyOnPremConnectorRuntimeHooks(&hooks)

	checks := []struct {
		name string
		ok   bool
	}{
		{name: "Semantics", ok: hooks.Semantics != nil},
		{name: "BuildCreateBody", ok: hooks.BuildCreateBody != nil},
		{name: "BuildUpdateBody", ok: hooks.BuildUpdateBody != nil},
		{name: "Identity.Resolve", ok: hooks.Identity.Resolve != nil},
		{name: "Identity.RecordPath", ok: hooks.Identity.RecordPath != nil},
		{name: "Identity.GuardExistingBeforeCreate", ok: hooks.Identity.GuardExistingBeforeCreate != nil},
		{name: "DeleteHooks.HandleError", ok: hooks.DeleteHooks.HandleError != nil},
		{name: "DeleteHooks.ApplyOutcome", ok: hooks.DeleteHooks.ApplyOutcome != nil},
		{name: "StatusHooks.ProjectStatus", ok: hooks.StatusHooks.ProjectStatus != nil},
		{name: "StatusHooks.MarkTerminating", ok: hooks.StatusHooks.MarkTerminating != nil},
		{name: "WrapGeneratedClient", ok: len(hooks.WrapGeneratedClient) > 0},
	}
	for _, check := range checks {
		if !check.ok {
			t.Fatalf("hooks.%s not configured", check.name)
		}
	}

	body, err := hooks.BuildCreateBody(context.Background(), makeOnPremConnectorResource(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(datasafesdk.CreateOnPremConnectorDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want CreateOnPremConnectorDetails", body)
	}
	requireOnPremConnectorStringPtr(t, "CreateOnPremConnectorDetails.CompartmentId", details.CompartmentId, testOnPremConnectorCompartmentID)
	requireOnPremConnectorStringPtr(t, "CreateOnPremConnectorDetails.DisplayName", details.DisplayName, testOnPremConnectorDisplayName)
	requireOnPremConnectorStringPtr(t, "CreateOnPremConnectorDetails.Description", details.Description, testOnPremConnectorDescription)
}

func TestOnPremConnectorCreateRecordsIdentityRequestIDAndLifecycle(t *testing.T) {
	resource := makeOnPremConnectorResource()
	created := sdkOnPremConnector(resource, testOnPremConnectorID, datasafesdk.OnPremConnectorLifecycleStateCreating)
	client := &fakeOnPremConnectorOCIClient{
		createFn: func(_ context.Context, request datasafesdk.CreateOnPremConnectorRequest) (datasafesdk.CreateOnPremConnectorResponse, error) {
			requireOnPremConnectorCreateRequest(t, request, resource)
			return datasafesdk.CreateOnPremConnectorResponse{
				OnPremConnector: created,
				OpcRequestId:    common.String("opc-create"),
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetOnPremConnectorRequest) (datasafesdk.GetOnPremConnectorResponse, error) {
			requireOnPremConnectorStringPtr(t, "GetOnPremConnectorRequest.OnPremConnectorId", request.OnPremConnectorId, testOnPremConnectorID)
			return datasafesdk.GetOnPremConnectorResponse{OnPremConnector: created}, nil
		},
	}

	response, err := newTestOnPremConnectorClient(client).CreateOrUpdate(context.Background(), resource, onPremConnectorRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want requeue while CREATING")
	}
	assertOnPremConnectorCallCount(t, "ListOnPremConnectors()", client.listCalls, 1)
	assertOnPremConnectorCallCount(t, "CreateOnPremConnector()", client.createCalls, 1)
	assertOnPremConnectorCallCount(t, "GetOnPremConnector()", client.getCalls, 1)
	assertOnPremConnectorRecordedID(t, resource, testOnPremConnectorID)
	assertOnPremConnectorOpcRequestID(t, resource, "opc-create")
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseCreate {
		t.Fatalf("status.status.async.current = %#v, want create lifecycle tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestOnPremConnectorCreateOrUpdateBindsExistingFromLaterListPage(t *testing.T) {
	resource := makeOnPremConnectorResource()
	existing := sdkOnPremConnector(resource, testOnPremConnectorID, datasafesdk.OnPremConnectorLifecycleStateActive)
	var pages []string
	client := &fakeOnPremConnectorOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListOnPremConnectorsRequest) (datasafesdk.ListOnPremConnectorsResponse, error) {
			pages = append(pages, onPremConnectorStringValue(request.Page))
			requireOnPremConnectorStringPtr(t, "ListOnPremConnectorsRequest.CompartmentId", request.CompartmentId, testOnPremConnectorCompartmentID)
			requireOnPremConnectorStringPtr(t, "ListOnPremConnectorsRequest.DisplayName", request.DisplayName, testOnPremConnectorDisplayName)
			if request.Page == nil {
				return datasafesdk.ListOnPremConnectorsResponse{
					Items: []datasafesdk.OnPremConnectorSummary{
						sdkOnPremConnectorSummary(resource, "ocid1.datasafeonpremconnector.oc1..other", "other-connector", datasafesdk.OnPremConnectorLifecycleStateActive),
					},
					OpcNextPage: common.String("page-2"),
				}, nil
			}
			return datasafesdk.ListOnPremConnectorsResponse{
				Items: []datasafesdk.OnPremConnectorSummary{
					sdkOnPremConnectorSummary(resource, testOnPremConnectorID, testOnPremConnectorDisplayName, datasafesdk.OnPremConnectorLifecycleStateActive),
				},
			}, nil
		},
		getFn: func(_ context.Context, request datasafesdk.GetOnPremConnectorRequest) (datasafesdk.GetOnPremConnectorResponse, error) {
			requireOnPremConnectorStringPtr(t, "GetOnPremConnectorRequest.OnPremConnectorId", request.OnPremConnectorId, testOnPremConnectorID)
			return datasafesdk.GetOnPremConnectorResponse{OnPremConnector: existing}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateOnPremConnectorRequest) (datasafesdk.CreateOnPremConnectorResponse, error) {
			t.Fatal("CreateOnPremConnector() called despite existing list match")
			return datasafesdk.CreateOnPremConnectorResponse{}, nil
		},
	}

	response, err := newTestOnPremConnectorClient(client).CreateOrUpdate(context.Background(), resource, onPremConnectorRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if got := strings.Join(pages, ","); got != ",page-2" {
		t.Fatalf("ListOnPremConnectors() pages = %q, want \",page-2\"", got)
	}
	assertOnPremConnectorRecordedID(t, resource, testOnPremConnectorID)
	assertOnPremConnectorCallCount(t, "CreateOnPremConnector()", client.createCalls, 0)
}

func TestOnPremConnectorCreateOrUpdateNoopsWhenObservedStateMatches(t *testing.T) {
	resource := makeOnPremConnectorResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOnPremConnectorID)
	current := sdkOnPremConnector(resource, testOnPremConnectorID, datasafesdk.OnPremConnectorLifecycleStateActive)
	client := &fakeOnPremConnectorOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetOnPremConnectorRequest) (datasafesdk.GetOnPremConnectorResponse, error) {
			requireOnPremConnectorStringPtr(t, "GetOnPremConnectorRequest.OnPremConnectorId", request.OnPremConnectorId, testOnPremConnectorID)
			return datasafesdk.GetOnPremConnectorResponse{OnPremConnector: current}, nil
		},
		createFn: func(context.Context, datasafesdk.CreateOnPremConnectorRequest) (datasafesdk.CreateOnPremConnectorResponse, error) {
			t.Fatal("CreateOnPremConnector() called during no-op reconcile")
			return datasafesdk.CreateOnPremConnectorResponse{}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateOnPremConnectorRequest) (datasafesdk.UpdateOnPremConnectorResponse, error) {
			t.Fatal("UpdateOnPremConnector() called during no-op reconcile")
			return datasafesdk.UpdateOnPremConnectorResponse{}, nil
		},
	}

	response, err := newTestOnPremConnectorClient(client).CreateOrUpdate(context.Background(), resource, onPremConnectorRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	assertOnPremConnectorCallCount(t, "GetOnPremConnector()", client.getCalls, 1)
	assertOnPremConnectorCallCount(t, "CreateOnPremConnector()", client.createCalls, 0)
	assertOnPremConnectorCallCount(t, "UpdateOnPremConnector()", client.updateCalls, 0)
	requireOnPremConnectorLastCondition(t, resource, shared.Active)
}

func TestOnPremConnectorCreateOrUpdateMutableUpdateRefreshesObservedState(t *testing.T) {
	resource := makeOnPremConnectorResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOnPremConnectorID)
	currentResource := makeOnPremConnectorResource()
	currentResource.Spec.DisplayName = "old-connector"
	currentResource.Spec.Description = "old description"
	currentResource.Spec.FreeformTags = map[string]string{"owner": "old"}
	currentResource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "41"}}
	current := sdkOnPremConnector(currentResource, testOnPremConnectorID, datasafesdk.OnPremConnectorLifecycleStateActive)
	updated := sdkOnPremConnector(resource, testOnPremConnectorID, datasafesdk.OnPremConnectorLifecycleStateActive)
	getResponses := []datasafesdk.GetOnPremConnectorResponse{
		{OnPremConnector: current},
		{OnPremConnector: updated},
	}
	client := &fakeOnPremConnectorOCIClient{
		getFn: getOnPremConnectorResponses(t, &getResponses),
		updateFn: func(_ context.Context, request datasafesdk.UpdateOnPremConnectorRequest) (datasafesdk.UpdateOnPremConnectorResponse, error) {
			requireOnPremConnectorStringPtr(t, "UpdateOnPremConnectorRequest.OnPremConnectorId", request.OnPremConnectorId, testOnPremConnectorID)
			requireOnPremConnectorStringPtr(t, "UpdateOnPremConnectorDetails.DisplayName", request.DisplayName, testOnPremConnectorDisplayName)
			requireOnPremConnectorStringPtr(t, "UpdateOnPremConnectorDetails.Description", request.Description, testOnPremConnectorDescription)
			if !reflect.DeepEqual(request.FreeformTags, resource.Spec.FreeformTags) {
				t.Fatalf("UpdateOnPremConnectorDetails.FreeformTags = %#v, want %#v", request.FreeformTags, resource.Spec.FreeformTags)
			}
			if got := request.DefinedTags["Operations"]["CostCenter"]; got != "42" {
				t.Fatalf("UpdateOnPremConnectorDetails.DefinedTags[Operations][CostCenter] = %#v, want 42", got)
			}
			return datasafesdk.UpdateOnPremConnectorResponse{OpcRequestId: common.String("opc-update")}, nil
		},
	}

	response, err := newTestOnPremConnectorClient(client).CreateOrUpdate(context.Background(), resource, onPremConnectorRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = true, want false after ACTIVE readback")
	}
	assertOnPremConnectorCallCount(t, "GetOnPremConnector()", client.getCalls, 2)
	assertOnPremConnectorCallCount(t, "UpdateOnPremConnector()", client.updateCalls, 1)
	assertOnPremConnectorOpcRequestID(t, resource, "opc-update")
	if resource.Status.DisplayName != testOnPremConnectorDisplayName {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, testOnPremConnectorDisplayName)
	}
	requireOnPremConnectorLastCondition(t, resource, shared.Active)
}

func TestOnPremConnectorCompartmentDriftRejectedBeforeOCI(t *testing.T) {
	resource := makeOnPremConnectorResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOnPremConnectorID)
	resource.Status.Id = testOnPremConnectorID
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..replacement"
	client := &fakeOnPremConnectorOCIClient{
		getFn: func(context.Context, datasafesdk.GetOnPremConnectorRequest) (datasafesdk.GetOnPremConnectorResponse, error) {
			t.Fatal("GetOnPremConnector() called despite compartment drift")
			return datasafesdk.GetOnPremConnectorResponse{}, nil
		},
		updateFn: func(context.Context, datasafesdk.UpdateOnPremConnectorRequest) (datasafesdk.UpdateOnPremConnectorResponse, error) {
			t.Fatal("UpdateOnPremConnector() called despite compartment drift")
			return datasafesdk.UpdateOnPremConnectorResponse{}, nil
		},
	}

	_, err := newTestOnPremConnectorClient(client).CreateOrUpdate(context.Background(), resource, onPremConnectorRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want compartment drift rejection")
	}
	if !strings.Contains(err.Error(), "compartmentId") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId drift detail", err)
	}
	assertOnPremConnectorCallCount(t, "GetOnPremConnector()", client.getCalls, 0)
	assertOnPremConnectorCallCount(t, "UpdateOnPremConnector()", client.updateCalls, 0)
	requireOnPremConnectorLastCondition(t, resource, shared.Failed)
}

func TestOnPremConnectorDeleteRetainsFinalizerWhileReadbackStillActive(t *testing.T) {
	resource := makeOnPremConnectorResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOnPremConnectorID)
	active := sdkOnPremConnector(resource, testOnPremConnectorID, datasafesdk.OnPremConnectorLifecycleStateActive)
	getResponses := []datasafesdk.GetOnPremConnectorResponse{
		{OnPremConnector: active},
		{OnPremConnector: active},
		{OnPremConnector: active},
	}
	client := &fakeOnPremConnectorOCIClient{
		getFn: getOnPremConnectorResponses(t, &getResponses),
		deleteFn: func(_ context.Context, request datasafesdk.DeleteOnPremConnectorRequest) (datasafesdk.DeleteOnPremConnectorResponse, error) {
			requireOnPremConnectorStringPtr(t, "DeleteOnPremConnectorRequest.OnPremConnectorId", request.OnPremConnectorId, testOnPremConnectorID)
			return datasafesdk.DeleteOnPremConnectorResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestOnPremConnectorClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	assertOnPremConnectorCallCount(t, "DeleteOnPremConnector()", client.deleteCalls, 1)
	assertOnPremConnectorOpcRequestID(t, resource, "opc-delete")
	requireOnPremConnectorLastCondition(t, resource, shared.Terminating)
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current = %#v, want delete lifecycle tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestOnPremConnectorDeleteWaitsForPendingWriteLifecycle(t *testing.T) {
	tests := []struct {
		name  string
		state datasafesdk.OnPremConnectorLifecycleStateEnum
	}{
		{name: "creating", state: datasafesdk.OnPremConnectorLifecycleStateCreating},
		{name: "updating", state: datasafesdk.OnPremConnectorLifecycleStateUpdating},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireOnPremConnectorDeleteWaitsForPendingWriteLifecycle(t, tt.state)
		})
	}
}

func TestOnPremConnectorDeleteReleasesFinalizerAfterDeletedReadback(t *testing.T) {
	resource := makeOnPremConnectorResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOnPremConnectorID)
	active := sdkOnPremConnector(resource, testOnPremConnectorID, datasafesdk.OnPremConnectorLifecycleStateActive)
	deletedConnector := sdkOnPremConnector(resource, testOnPremConnectorID, datasafesdk.OnPremConnectorLifecycleStateDeleted)
	getResponses := []datasafesdk.GetOnPremConnectorResponse{
		{OnPremConnector: active},
		{OnPremConnector: active},
		{OnPremConnector: deletedConnector},
	}
	client := &fakeOnPremConnectorOCIClient{
		getFn: getOnPremConnectorResponses(t, &getResponses),
		deleteFn: func(_ context.Context, request datasafesdk.DeleteOnPremConnectorRequest) (datasafesdk.DeleteOnPremConnectorResponse, error) {
			requireOnPremConnectorStringPtr(t, "DeleteOnPremConnectorRequest.OnPremConnectorId", request.OnPremConnectorId, testOnPremConnectorID)
			return datasafesdk.DeleteOnPremConnectorResponse{OpcRequestId: common.String("opc-delete")}, nil
		},
	}

	deleted, err := newTestOnPremConnectorClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after deleted readback")
	}
	assertOnPremConnectorCallCount(t, "DeleteOnPremConnector()", client.deleteCalls, 1)
	assertOnPremConnectorOpcRequestID(t, resource, "opc-delete")
}

func TestOnPremConnectorDeleteReleasesFinalizerAfterPreDeleteGetNotFound(t *testing.T) {
	resource := makeOnPremConnectorResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOnPremConnectorID)
	notFoundErr := errortest.NewServiceError(404, errorutil.NotFound, "missing")
	notFoundErr.OpcRequestID = "opc-not-found"
	client := &fakeOnPremConnectorOCIClient{
		getFn: func(context.Context, datasafesdk.GetOnPremConnectorRequest) (datasafesdk.GetOnPremConnectorResponse, error) {
			return datasafesdk.GetOnPremConnectorResponse{}, notFoundErr
		},
		deleteFn: func(context.Context, datasafesdk.DeleteOnPremConnectorRequest) (datasafesdk.DeleteOnPremConnectorResponse, error) {
			t.Fatal("DeleteOnPremConnector() called after unambiguous pre-delete NotFound")
			return datasafesdk.DeleteOnPremConnectorResponse{}, nil
		},
	}

	deleted, err := newTestOnPremConnectorClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after unambiguous pre-delete NotFound")
	}
	assertOnPremConnectorCallCount(t, "GetOnPremConnector()", client.getCalls, 1)
	assertOnPremConnectorCallCount(t, "DeleteOnPremConnector()", client.deleteCalls, 0)
	assertOnPremConnectorOpcRequestID(t, resource, "opc-not-found")
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp after unambiguous pre-delete NotFound")
	}
}

func TestOnPremConnectorDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	resource := makeOnPremConnectorResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOnPremConnectorID)
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
	authErr.OpcRequestID = "opc-auth"
	client := &fakeOnPremConnectorOCIClient{
		getFn: func(context.Context, datasafesdk.GetOnPremConnectorRequest) (datasafesdk.GetOnPremConnectorResponse, error) {
			return datasafesdk.GetOnPremConnectorResponse{}, authErr
		},
		deleteFn: func(context.Context, datasafesdk.DeleteOnPremConnectorRequest) (datasafesdk.DeleteOnPremConnectorResponse, error) {
			t.Fatal("DeleteOnPremConnector() called after ambiguous pre-delete read")
			return datasafesdk.DeleteOnPremConnectorResponse{}, nil
		},
	}

	deleted, err := newTestOnPremConnectorClient(client).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not found")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want auth-shaped detail", err)
	}
	assertOnPremConnectorCallCount(t, "DeleteOnPremConnector()", client.deleteCalls, 0)
	assertOnPremConnectorOpcRequestID(t, resource, "opc-auth")
}

func TestOnPremConnectorDeleteReturnsPreDeleteGetFailure(t *testing.T) {
	resource := makeOnPremConnectorResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOnPremConnectorID)
	getErr := errortest.NewServiceError(500, "InternalError", "get failed")
	getErr.OpcRequestID = "opc-get-failure"
	client := &fakeOnPremConnectorOCIClient{
		getFn: func(context.Context, datasafesdk.GetOnPremConnectorRequest) (datasafesdk.GetOnPremConnectorResponse, error) {
			return datasafesdk.GetOnPremConnectorResponse{}, getErr
		},
		deleteFn: func(context.Context, datasafesdk.DeleteOnPremConnectorRequest) (datasafesdk.DeleteOnPremConnectorResponse, error) {
			t.Fatal("DeleteOnPremConnector() called after failed pre-delete get")
			return datasafesdk.DeleteOnPremConnectorResponse{}, nil
		},
	}

	deleted, err := newTestOnPremConnectorClient(client).Delete(context.Background(), resource)
	requireOnPremConnectorPreDeleteFailure(t, resource, deleted, err, "pre-delete get", "opc-get-failure")
	assertOnPremConnectorCallCount(t, "GetOnPremConnector()", client.getCalls, 1)
	assertOnPremConnectorCallCount(t, "DeleteOnPremConnector()", client.deleteCalls, 0)
}

func TestOnPremConnectorDeleteReturnsPreDeleteListFailure(t *testing.T) {
	resource := makeOnPremConnectorResource()
	listErr := errortest.NewServiceError(500, "InternalError", "list failed")
	listErr.OpcRequestID = "opc-list-failure"
	client := &fakeOnPremConnectorOCIClient{
		listFn: func(_ context.Context, request datasafesdk.ListOnPremConnectorsRequest) (datasafesdk.ListOnPremConnectorsResponse, error) {
			requireOnPremConnectorStringPtr(t, "ListOnPremConnectorsRequest.CompartmentId", request.CompartmentId, testOnPremConnectorCompartmentID)
			requireOnPremConnectorStringPtr(t, "ListOnPremConnectorsRequest.DisplayName", request.DisplayName, testOnPremConnectorDisplayName)
			return datasafesdk.ListOnPremConnectorsResponse{}, listErr
		},
		deleteFn: func(context.Context, datasafesdk.DeleteOnPremConnectorRequest) (datasafesdk.DeleteOnPremConnectorResponse, error) {
			t.Fatal("DeleteOnPremConnector() called after failed pre-delete list")
			return datasafesdk.DeleteOnPremConnectorResponse{}, nil
		},
	}

	deleted, err := newTestOnPremConnectorClient(client).Delete(context.Background(), resource)
	requireOnPremConnectorPreDeleteFailure(t, resource, deleted, err, "pre-delete list", "opc-list-failure")
	assertOnPremConnectorCallCount(t, "ListOnPremConnectors()", client.listCalls, 1)
	assertOnPremConnectorCallCount(t, "DeleteOnPremConnector()", client.deleteCalls, 0)
}

func TestOnPremConnectorCreateOrUpdateRecordsOpcRequestIDFromOCIError(t *testing.T) {
	resource := makeOnPremConnectorResource()
	createErr := errortest.NewServiceError(500, "InternalError", "create failed")
	createErr.OpcRequestID = "opc-create-error"
	client := &fakeOnPremConnectorOCIClient{
		createFn: func(context.Context, datasafesdk.CreateOnPremConnectorRequest) (datasafesdk.CreateOnPremConnectorResponse, error) {
			return datasafesdk.CreateOnPremConnectorResponse{}, createErr
		},
	}

	_, err := newTestOnPremConnectorClient(client).CreateOrUpdate(context.Background(), resource, onPremConnectorRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI error")
	}
	assertOnPremConnectorOpcRequestID(t, resource, "opc-create-error")
	requireOnPremConnectorLastCondition(t, resource, shared.Failed)
}

func newTestOnPremConnectorClient(client onPremConnectorOCIClient) OnPremConnectorServiceClient {
	return newOnPremConnectorServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeOnPremConnectorResource() *datasafev1beta1.OnPremConnector {
	return &datasafev1beta1.OnPremConnector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "on-prem-connector",
			Namespace: "default",
		},
		Spec: datasafev1beta1.OnPremConnectorSpec{
			CompartmentId: testOnPremConnectorCompartmentID,
			DisplayName:   testOnPremConnectorDisplayName,
			Description:   testOnPremConnectorDescription,
			FreeformTags:  map[string]string{"owner": "runtime"},
			DefinedTags:   map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func onPremConnectorRequest(resource *datasafev1beta1.OnPremConnector) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: resource.Name, Namespace: resource.Namespace}}
}

func sdkOnPremConnector(
	resource *datasafev1beta1.OnPremConnector,
	id string,
	lifecycleState datasafesdk.OnPremConnectorLifecycleStateEnum,
) datasafesdk.OnPremConnector {
	created := common.SDKTime{Time: time.Date(2026, time.May, 5, 10, 0, 0, 0, time.UTC)}
	return datasafesdk.OnPremConnector{
		Id:               common.String(id),
		DisplayName:      common.String(resource.Spec.DisplayName),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		TimeCreated:      &created,
		LifecycleState:   lifecycleState,
		Description:      common.String(resource.Spec.Description),
		LifecycleDetails: common.String("connector lifecycle details"),
		FreeformTags:     onPremConnectorStringMap(resource.Spec.FreeformTags),
		DefinedTags:      onPremConnectorDefinedTags(resource.Spec.DefinedTags),
		SystemTags:       map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "false"}},
		AvailableVersion: common.String("2.0"),
		CreatedVersion:   common.String("1.0"),
	}
}

func sdkOnPremConnectorSummary(
	resource *datasafev1beta1.OnPremConnector,
	id string,
	displayName string,
	lifecycleState datasafesdk.OnPremConnectorLifecycleStateEnum,
) datasafesdk.OnPremConnectorSummary {
	created := common.SDKTime{Time: time.Date(2026, time.May, 5, 10, 0, 0, 0, time.UTC)}
	return datasafesdk.OnPremConnectorSummary{
		Id:               common.String(id),
		DisplayName:      common.String(displayName),
		CompartmentId:    common.String(resource.Spec.CompartmentId),
		TimeCreated:      &created,
		LifecycleState:   lifecycleState,
		Description:      common.String(resource.Spec.Description),
		LifecycleDetails: common.String("connector lifecycle details"),
		FreeformTags:     onPremConnectorStringMap(resource.Spec.FreeformTags),
		DefinedTags:      onPremConnectorDefinedTags(resource.Spec.DefinedTags),
		SystemTags:       map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "false"}},
		CreatedVersion:   common.String("1.0"),
	}
}

func getOnPremConnectorResponses(
	t *testing.T,
	responses *[]datasafesdk.GetOnPremConnectorResponse,
) func(context.Context, datasafesdk.GetOnPremConnectorRequest) (datasafesdk.GetOnPremConnectorResponse, error) {
	t.Helper()
	return func(_ context.Context, request datasafesdk.GetOnPremConnectorRequest) (datasafesdk.GetOnPremConnectorResponse, error) {
		t.Helper()
		requireOnPremConnectorStringPtr(t, "GetOnPremConnectorRequest.OnPremConnectorId", request.OnPremConnectorId, testOnPremConnectorID)
		if len(*responses) == 0 {
			t.Fatal("GetOnPremConnector() called with no prepared response")
		}
		response := (*responses)[0]
		*responses = (*responses)[1:]
		return response, nil
	}
}

func requireOnPremConnectorDeleteWaitsForPendingWriteLifecycle(
	t *testing.T,
	state datasafesdk.OnPremConnectorLifecycleStateEnum,
) {
	t.Helper()
	resource := makeOnPremConnectorResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testOnPremConnectorID)
	current := sdkOnPremConnector(resource, testOnPremConnectorID, state)
	client := &fakeOnPremConnectorOCIClient{
		getFn: func(_ context.Context, request datasafesdk.GetOnPremConnectorRequest) (datasafesdk.GetOnPremConnectorResponse, error) {
			requireOnPremConnectorStringPtr(t, "GetOnPremConnectorRequest.OnPremConnectorId", request.OnPremConnectorId, testOnPremConnectorID)
			return datasafesdk.GetOnPremConnectorResponse{OnPremConnector: current}, nil
		},
		deleteFn: func(context.Context, datasafesdk.DeleteOnPremConnectorRequest) (datasafesdk.DeleteOnPremConnectorResponse, error) {
			t.Fatal("DeleteOnPremConnector() called while live OnPremConnector has pending write lifecycle state")
			return datasafesdk.DeleteOnPremConnectorResponse{}, nil
		},
	}

	deleted, err := newTestOnPremConnectorClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while pending write completes")
	}
	assertOnPremConnectorCallCount(t, "GetOnPremConnector()", client.getCalls, 1)
	assertOnPremConnectorCallCount(t, "DeleteOnPremConnector()", client.deleteCalls, 0)
	if got := resource.Status.LifecycleState; got != string(state) {
		t.Fatalf("status.lifecycleState = %q, want %q", got, state)
	}
	requireOnPremConnectorLastCondition(t, resource, shared.Terminating)
	requireOnPremConnectorPendingDeleteAsync(t, resource, state)
}

func requireOnPremConnectorCreateRequest(
	t *testing.T,
	request datasafesdk.CreateOnPremConnectorRequest,
	resource *datasafev1beta1.OnPremConnector,
) {
	t.Helper()
	requireOnPremConnectorStringPtr(t, "CreateOnPremConnectorDetails.CompartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireOnPremConnectorStringPtr(t, "CreateOnPremConnectorDetails.DisplayName", request.DisplayName, resource.Spec.DisplayName)
	requireOnPremConnectorStringPtr(t, "CreateOnPremConnectorDetails.Description", request.Description, resource.Spec.Description)
	if !reflect.DeepEqual(request.FreeformTags, resource.Spec.FreeformTags) {
		t.Fatalf("CreateOnPremConnectorDetails.FreeformTags = %#v, want %#v", request.FreeformTags, resource.Spec.FreeformTags)
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("CreateOnPremConnectorDetails.DefinedTags[Operations][CostCenter] = %#v, want 42", got)
	}
}

func assertOnPremConnectorRecordedID(t *testing.T, resource *datasafev1beta1.OnPremConnector, want string) {
	t.Helper()
	if got := resource.Status.Id; got != want {
		t.Fatalf("status.id = %q, want %q", got, want)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func assertOnPremConnectorOpcRequestID(t *testing.T, resource *datasafev1beta1.OnPremConnector, want string) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func assertOnPremConnectorCallCount(t *testing.T, operation string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s calls = %d, want %d", operation, got, want)
	}
}

func requireOnPremConnectorPreDeleteFailure(
	t *testing.T,
	resource *datasafev1beta1.OnPremConnector,
	deleted bool,
	err error,
	wantDetail string,
	wantOpcRequestID string,
) {
	t.Helper()
	if err == nil {
		t.Fatalf("Delete() error = nil, want %s failure", wantDetail)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), wantDetail) {
		t.Fatalf("Delete() error = %v, want %s detail", err, wantDetail)
	}
	assertOnPremConnectorOpcRequestID(t, resource, wantOpcRequestID)
	requireOnPremConnectorLastCondition(t, resource, shared.Failed)
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set after failed pre-delete read")
	}
}

func requireOnPremConnectorPendingDeleteAsync(
	t *testing.T,
	resource *datasafev1beta1.OnPremConnector,
	state datasafesdk.OnPremConnectorLifecycleStateEnum,
) {
	t.Helper()
	currentAsync := resource.Status.OsokStatus.Async.Current
	if currentAsync == nil ||
		currentAsync.Phase != shared.OSOKAsyncPhaseDelete ||
		currentAsync.NormalizedClass != shared.OSOKAsyncClassPending ||
		currentAsync.RawStatus != string(state) {
		t.Fatalf("status.status.async.current = %#v, want pending delete for %s", currentAsync, state)
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, string(state)) {
		t.Fatalf("status.status.message = %q, want pending state %s", resource.Status.OsokStatus.Message, state)
	}
}

func requireOnPremConnectorLastCondition(
	t *testing.T,
	resource *datasafev1beta1.OnPremConnector,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %q, want %q", got, want)
	}
}

func requireOnPremConnectorStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}
