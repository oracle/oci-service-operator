/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package pingmonitor

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	healthcheckssdk "github.com/oracle/oci-go-sdk/v65/healthchecks"
	healthchecksv1beta1 "github.com/oracle/oci-service-operator/api/healthchecks/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testPingMonitorCompartmentID = "ocid1.compartment.oc1..ping"
	testPingMonitorExistingID    = "ocid1.healthcheckpingmonitor.oc1..existing"
	testPingMonitorCreatedID     = "ocid1.healthcheckpingmonitor.oc1..created"
	testPingMonitorDisplayName   = "ping-monitor-sample"
	testPingMonitorProtocol      = string(healthcheckssdk.PingProbeProtocolTcp)
)

func TestPingMonitorRuntimeSemantics(t *testing.T) {
	t.Parallel()

	hooks := PingMonitorRuntimeHooks{}
	applyPingMonitorRuntimeHooks(&hooks)

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed PingMonitor semantics")
	}
	if hooks.Semantics.Async == nil || hooks.Semantics.Async.Strategy != "lifecycle" {
		t.Fatalf("Async = %#v, want lifecycle semantics", hooks.Semantics.Async)
	}
	if hooks.Semantics.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", hooks.Semantics.FinalizerPolicy)
	}
	if hooks.Semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want confirm-delete", hooks.Semantics.DeleteFollowUp.Strategy)
	}
	if hooks.Semantics.List == nil {
		t.Fatal("List semantics = nil, want create-or-bind list matching")
	}
	assertContainsAll(t, hooks.Semantics.List.MatchFields, "compartmentId", "displayName", "protocol")
	assertContainsAll(t, hooks.Semantics.Mutation.Mutable, "targets", "displayName", "intervalInSeconds", "isEnabled", "freeformTags", "definedTags")
	assertContainsAll(t, hooks.Semantics.Mutation.ForceNew, "compartmentId")
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("DeleteHooks.HandleError = nil, want auth-shaped not-found guard")
	}
}

func TestPingMonitorCreateOrUpdateCreatesThenObservesExisting(t *testing.T) {
	t.Parallel()

	resource := newPingMonitorResource()
	fake := newCreateThenObserveFake(t)
	client := newTestPingMonitorClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertCreatePingMonitorResponse(t, response, resource)

	response, err = client.CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("second CreateOrUpdate() error = %v", err)
	}
	assertNoOpPingMonitorResponse(t, response, resource, fake)
}

func TestPingMonitorCreateOrUpdateBindsExistingMonitor(t *testing.T) {
	t.Parallel()

	resource := newPingMonitorResource()
	fake := &fakePingMonitorOCIClient{}
	fake.listPingMonitors = func(_ context.Context, _ healthcheckssdk.ListPingMonitorsRequest) (healthcheckssdk.ListPingMonitorsResponse, error) {
		return healthcheckssdk.ListPingMonitorsResponse{
			Items: []healthcheckssdk.PingMonitorSummary{
				pingMonitorSummary(testPingMonitorExistingID, testPingMonitorCompartmentID, testPingMonitorDisplayName),
			},
		}, nil
	}
	fake.getPingMonitor = func(_ context.Context, request healthcheckssdk.GetPingMonitorRequest) (healthcheckssdk.GetPingMonitorResponse, error) {
		if got, want := stringValue(request.MonitorId), testPingMonitorExistingID; got != want {
			t.Fatalf("GetPingMonitor monitorId = %q, want %q", got, want)
		}
		return healthcheckssdk.GetPingMonitorResponse{
			PingMonitor: pingMonitorBody(testPingMonitorExistingID, testPingMonitorCompartmentID, testPingMonitorDisplayName, true),
		}, nil
	}

	response, err := newTestPingMonitorClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active bind", response)
	}
	if fake.createCalls != 0 {
		t.Fatalf("CreatePingMonitor calls = %d, want 0", fake.createCalls)
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), testPingMonitorExistingID; got != want {
		t.Fatalf("status.ocid = %q, want %q", got, want)
	}
	assertTrailingCondition(t, resource, shared.Active)
}

func TestPingMonitorCreateOrUpdateBindsExistingMonitorFromLaterListPage(t *testing.T) {
	t.Parallel()

	resource := newPingMonitorResource()
	fake := &fakePingMonitorOCIClient{}
	fake.listPingMonitors = pagedPingMonitorList(t, []pingMonitorListPage{
		{
			items: []healthcheckssdk.PingMonitorSummary{
				pingMonitorSummary("ocid1.healthcheckpingmonitor.oc1..other", testPingMonitorCompartmentID, "other-monitor"),
			},
			nextPage: "page-2",
		},
		{
			wantPage: "page-2",
			items: []healthcheckssdk.PingMonitorSummary{
				pingMonitorSummary(testPingMonitorExistingID, testPingMonitorCompartmentID, testPingMonitorDisplayName),
			},
		},
	})
	fake.getPingMonitor = func(_ context.Context, request healthcheckssdk.GetPingMonitorRequest) (healthcheckssdk.GetPingMonitorResponse, error) {
		if got, want := stringValue(request.MonitorId), testPingMonitorExistingID; got != want {
			t.Fatalf("GetPingMonitor monitorId = %q, want %q", got, want)
		}
		return healthcheckssdk.GetPingMonitorResponse{
			PingMonitor: pingMonitorBody(testPingMonitorExistingID, testPingMonitorCompartmentID, testPingMonitorDisplayName, true),
		}, nil
	}

	response, err := newTestPingMonitorClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want active bind", response)
	}
	if fake.listCalls != 2 {
		t.Fatalf("ListPingMonitors calls = %d, want 2", fake.listCalls)
	}
	if fake.createCalls != 0 {
		t.Fatalf("CreatePingMonitor calls = %d, want 0", fake.createCalls)
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), testPingMonitorExistingID; got != want {
		t.Fatalf("status.ocid = %q, want %q", got, want)
	}
	assertTrailingCondition(t, resource, shared.Active)
}

func TestPingMonitorCreateOrUpdateRejectsDuplicateMatchesAcrossListPages(t *testing.T) {
	t.Parallel()

	resource := newPingMonitorResource()
	fake := &fakePingMonitorOCIClient{}
	fake.listPingMonitors = pagedPingMonitorList(t, []pingMonitorListPage{
		{
			items: []healthcheckssdk.PingMonitorSummary{
				pingMonitorSummary(testPingMonitorExistingID, testPingMonitorCompartmentID, testPingMonitorDisplayName),
			},
			nextPage: "page-2",
		},
		{
			wantPage: "page-2",
			items: []healthcheckssdk.PingMonitorSummary{
				pingMonitorSummary("ocid1.healthcheckpingmonitor.oc1..duplicate", testPingMonitorCompartmentID, testPingMonitorDisplayName),
			},
		},
	})
	fake.getPingMonitor = func(context.Context, healthcheckssdk.GetPingMonitorRequest) (healthcheckssdk.GetPingMonitorResponse, error) {
		t.Fatal("GetPingMonitor should not be called when list resolution finds duplicate matches")
		return healthcheckssdk.GetPingMonitorResponse{}, nil
	}

	response, err := newTestPingMonitorClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err == nil || !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate match error", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if fake.listCalls != 2 {
		t.Fatalf("ListPingMonitors calls = %d, want 2", fake.listCalls)
	}
	if fake.createCalls != 0 {
		t.Fatalf("CreatePingMonitor calls = %d, want 0", fake.createCalls)
	}
}

func TestPingMonitorMutableUpdatePreservesExplicitFalse(t *testing.T) {
	t.Parallel()

	resource := newPingMonitorResource()
	resource.Spec.IsEnabled = false
	resource.Status.OsokStatus.Ocid = shared.OCID(testPingMonitorExistingID)
	resource.Status.Id = testPingMonitorExistingID
	fake := newMutableUpdateFake(t)

	response, err := newTestPingMonitorClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertMutableUpdateResponse(t, response, resource, fake)
}

func TestPingMonitorCreateOnlyDriftRejectedBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newPingMonitorResource()
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..different"
	resource.Status.OsokStatus.Ocid = shared.OCID(testPingMonitorExistingID)
	resource.Status.Id = testPingMonitorExistingID
	fake := &fakePingMonitorOCIClient{}
	fake.getPingMonitor = func(_ context.Context, _ healthcheckssdk.GetPingMonitorRequest) (healthcheckssdk.GetPingMonitorResponse, error) {
		return healthcheckssdk.GetPingMonitorResponse{
			PingMonitor: pingMonitorBody(testPingMonitorExistingID, testPingMonitorCompartmentID, testPingMonitorDisplayName, true),
		}, nil
	}

	response, err := newTestPingMonitorClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want compartmentId replacement rejection", err.Error())
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdatePingMonitor calls = %d, want 0", fake.updateCalls)
	}
	assertTrailingCondition(t, resource, shared.Failed)
}

func TestPingMonitorDeleteWaitsForConfirmedNotFound(t *testing.T) {
	t.Parallel()

	resource := newPingMonitorResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testPingMonitorExistingID)
	resource.Status.Id = testPingMonitorExistingID
	fake := &fakePingMonitorOCIClient{}
	fake.getPingMonitor = func(_ context.Context, _ healthcheckssdk.GetPingMonitorRequest) (healthcheckssdk.GetPingMonitorResponse, error) {
		if fake.getCalls == 1 {
			return healthcheckssdk.GetPingMonitorResponse{
				PingMonitor: pingMonitorBody(testPingMonitorExistingID, testPingMonitorCompartmentID, testPingMonitorDisplayName, true),
			}, nil
		}
		return healthcheckssdk.GetPingMonitorResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "ping monitor deleted")
	}
	fake.deletePingMonitor = func(_ context.Context, request healthcheckssdk.DeletePingMonitorRequest) (healthcheckssdk.DeletePingMonitorResponse, error) {
		if got, want := stringValue(request.MonitorId), testPingMonitorExistingID; got != want {
			t.Fatalf("DeletePingMonitor monitorId = %q, want %q", got, want)
		}
		return healthcheckssdk.DeletePingMonitorResponse{OpcRequestId: common.String("delete-opc")}, nil
	}

	deleted, err := newTestPingMonitorClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not-found confirmation")
	}
	if fake.deleteCalls != 1 {
		t.Fatalf("DeletePingMonitor calls = %d, want 1", fake.deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want confirmed delete timestamp")
	}
	assertTrailingCondition(t, resource, shared.Terminating)
}

func TestPingMonitorDeleteKeepsFinalizerOnAuthShapedNotFound(t *testing.T) {
	t.Parallel()

	resource := newPingMonitorResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testPingMonitorExistingID)
	resource.Status.Id = testPingMonitorExistingID
	fake := &fakePingMonitorOCIClient{}
	fake.getPingMonitor = func(_ context.Context, _ healthcheckssdk.GetPingMonitorRequest) (healthcheckssdk.GetPingMonitorResponse, error) {
		return healthcheckssdk.GetPingMonitorResponse{
			PingMonitor: pingMonitorBody(testPingMonitorExistingID, testPingMonitorCompartmentID, testPingMonitorDisplayName, true),
		}, nil
	}
	fake.deletePingMonitor = func(_ context.Context, _ healthcheckssdk.DeletePingMonitorRequest) (healthcheckssdk.DeletePingMonitorResponse, error) {
		return healthcheckssdk.DeletePingMonitorResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous ping monitor miss")
	}

	deleted, err := newTestPingMonitorClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped not-found to remain fatal")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), "keeping the finalizer") {
		t.Fatalf("Delete() error = %q, want conservative finalizer message", err.Error())
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt set, want finalizer retained")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
}

func TestPingMonitorCreateErrorCapturesOpcRequestID(t *testing.T) {
	t.Parallel()

	resource := newPingMonitorResource()
	fake := &fakePingMonitorOCIClient{}
	fake.listPingMonitors = func(_ context.Context, _ healthcheckssdk.ListPingMonitorsRequest) (healthcheckssdk.ListPingMonitorsResponse, error) {
		return healthcheckssdk.ListPingMonitorsResponse{}, nil
	}
	fake.createPingMonitor = func(_ context.Context, _ healthcheckssdk.CreatePingMonitorRequest) (healthcheckssdk.CreatePingMonitorResponse, error) {
		return healthcheckssdk.CreatePingMonitorResponse{}, errortest.NewServiceError(409, errorutil.IncorrectState, "create is still settling")
	}

	response, err := newTestPingMonitorClient(fake).CreateOrUpdate(context.Background(), resource, reconcileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want surfaced OCI error")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
	assertTrailingCondition(t, resource, shared.Failed)
}

func newCreateThenObserveFake(t *testing.T) *fakePingMonitorOCIClient {
	t.Helper()
	fake := &fakePingMonitorOCIClient{}
	fake.listPingMonitors = func(_ context.Context, request healthcheckssdk.ListPingMonitorsRequest) (healthcheckssdk.ListPingMonitorsResponse, error) {
		assertPingMonitorListRequest(t, request)
		return healthcheckssdk.ListPingMonitorsResponse{}, nil
	}
	fake.createPingMonitor = func(_ context.Context, request healthcheckssdk.CreatePingMonitorRequest) (healthcheckssdk.CreatePingMonitorResponse, error) {
		assertPingMonitorCreateRequest(t, request)
		return healthcheckssdk.CreatePingMonitorResponse{
			PingMonitor:  pingMonitorBody(testPingMonitorCreatedID, testPingMonitorCompartmentID, testPingMonitorDisplayName, true),
			OpcRequestId: common.String("create-opc"),
		}, nil
	}
	fake.getPingMonitor = func(_ context.Context, request healthcheckssdk.GetPingMonitorRequest) (healthcheckssdk.GetPingMonitorResponse, error) {
		assertPingMonitorGetRequest(t, request, testPingMonitorCreatedID)
		return healthcheckssdk.GetPingMonitorResponse{
			PingMonitor: pingMonitorBody(testPingMonitorCreatedID, testPingMonitorCompartmentID, testPingMonitorDisplayName, true),
		}, nil
	}
	return fake
}

func newMutableUpdateFake(t *testing.T) *fakePingMonitorOCIClient {
	t.Helper()
	fake := &fakePingMonitorOCIClient{}
	fake.getPingMonitor = func(_ context.Context, _ healthcheckssdk.GetPingMonitorRequest) (healthcheckssdk.GetPingMonitorResponse, error) {
		enabled := fake.getCalls <= 1
		return healthcheckssdk.GetPingMonitorResponse{
			PingMonitor: pingMonitorBody(testPingMonitorExistingID, testPingMonitorCompartmentID, testPingMonitorDisplayName, enabled),
		}, nil
	}
	fake.updatePingMonitor = func(_ context.Context, request healthcheckssdk.UpdatePingMonitorRequest) (healthcheckssdk.UpdatePingMonitorResponse, error) {
		assertPingMonitorUpdateRequest(t, request)
		return healthcheckssdk.UpdatePingMonitorResponse{
			PingMonitor:  pingMonitorBody(testPingMonitorExistingID, testPingMonitorCompartmentID, testPingMonitorDisplayName, false),
			OpcRequestId: common.String("update-opc"),
		}, nil
	}
	return fake
}

func assertCreatePingMonitorResponse(t *testing.T, response servicemanager.OSOKResponse, resource *healthchecksv1beta1.PingMonitor) {
	t.Helper()
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful provisioning requeue", response)
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), testPingMonitorCreatedID; got != want {
		t.Fatalf("status.ocid = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "create-opc"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
	assertTrailingCondition(t, resource, shared.Provisioning)
}

func assertNoOpPingMonitorResponse(t *testing.T, response servicemanager.OSOKResponse, resource *healthchecksv1beta1.PingMonitor, fake *fakePingMonitorOCIClient) {
	t.Helper()
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("second CreateOrUpdate() response = %#v, want active no-op", response)
	}
	if fake.createCalls != 1 {
		t.Fatalf("CreatePingMonitor calls = %d, want 1", fake.createCalls)
	}
	if fake.updateCalls != 0 {
		t.Fatalf("UpdatePingMonitor calls = %d, want 0", fake.updateCalls)
	}
	assertTrailingCondition(t, resource, shared.Active)
}

func assertMutableUpdateResponse(t *testing.T, response servicemanager.OSOKResponse, resource *healthchecksv1beta1.PingMonitor, fake *fakePingMonitorOCIClient) {
	t.Helper()
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful update requeue", response)
	}
	if fake.updateCalls != 1 {
		t.Fatalf("UpdatePingMonitor calls = %d, want 1", fake.updateCalls)
	}
	if resource.Status.IsEnabled {
		t.Fatal("status.isEnabled = true, want false after update readback")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "update-opc"; got != want {
		t.Fatalf("status.opcRequestId = %q, want %q", got, want)
	}
	assertTrailingCondition(t, resource, shared.Updating)
}

func assertPingMonitorListRequest(t *testing.T, request healthcheckssdk.ListPingMonitorsRequest) {
	t.Helper()
	if got, want := stringValue(request.CompartmentId), testPingMonitorCompartmentID; got != want {
		t.Fatalf("ListPingMonitors compartmentId = %q, want %q", got, want)
	}
	if got, want := stringValue(request.DisplayName), testPingMonitorDisplayName; got != want {
		t.Fatalf("ListPingMonitors displayName = %q, want %q", got, want)
	}
}

func assertPingMonitorCreateRequest(t *testing.T, request healthcheckssdk.CreatePingMonitorRequest) {
	t.Helper()
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("CreatePingMonitor OpcRetryToken is empty")
	}
	if request.IsEnabled == nil || !*request.IsEnabled {
		t.Fatalf("CreatePingMonitor IsEnabled = %v, want true", request.IsEnabled)
	}
}

func assertPingMonitorGetRequest(t *testing.T, request healthcheckssdk.GetPingMonitorRequest, want string) {
	t.Helper()
	if got := stringValue(request.MonitorId); got != want {
		t.Fatalf("GetPingMonitor monitorId = %q, want %q", got, want)
	}
}

func assertPingMonitorUpdateRequest(t *testing.T, request healthcheckssdk.UpdatePingMonitorRequest) {
	t.Helper()
	if got, want := stringValue(request.MonitorId), testPingMonitorExistingID; got != want {
		t.Fatalf("UpdatePingMonitor monitorId = %q, want %q", got, want)
	}
	if request.IsEnabled == nil {
		t.Fatal("UpdatePingMonitor IsEnabled = nil, want explicit false")
	}
	if *request.IsEnabled {
		t.Fatal("UpdatePingMonitor IsEnabled = true, want false")
	}
}

type fakePingMonitorOCIClient struct {
	createPingMonitor func(context.Context, healthcheckssdk.CreatePingMonitorRequest) (healthcheckssdk.CreatePingMonitorResponse, error)
	getPingMonitor    func(context.Context, healthcheckssdk.GetPingMonitorRequest) (healthcheckssdk.GetPingMonitorResponse, error)
	listPingMonitors  func(context.Context, healthcheckssdk.ListPingMonitorsRequest) (healthcheckssdk.ListPingMonitorsResponse, error)
	updatePingMonitor func(context.Context, healthcheckssdk.UpdatePingMonitorRequest) (healthcheckssdk.UpdatePingMonitorResponse, error)
	deletePingMonitor func(context.Context, healthcheckssdk.DeletePingMonitorRequest) (healthcheckssdk.DeletePingMonitorResponse, error)

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int
}

type pingMonitorListPage struct {
	wantPage string
	nextPage string
	items    []healthcheckssdk.PingMonitorSummary
}

func pagedPingMonitorList(t *testing.T, pages []pingMonitorListPage) func(context.Context, healthcheckssdk.ListPingMonitorsRequest) (healthcheckssdk.ListPingMonitorsResponse, error) {
	t.Helper()
	remaining := append([]pingMonitorListPage(nil), pages...)
	return func(_ context.Context, request healthcheckssdk.ListPingMonitorsRequest) (healthcheckssdk.ListPingMonitorsResponse, error) {
		t.Helper()
		assertPingMonitorListRequest(t, request)
		if len(remaining) == 0 {
			t.Fatalf("unexpected ListPingMonitors page request %q", stringValue(request.Page))
		}
		page := remaining[0]
		remaining = remaining[1:]
		if got := stringValue(request.Page); got != page.wantPage {
			t.Fatalf("ListPingMonitors page = %q, want %q", got, page.wantPage)
		}
		response := healthcheckssdk.ListPingMonitorsResponse{Items: page.items}
		if page.nextPage != "" {
			response.OpcNextPage = common.String(page.nextPage)
		}
		return response, nil
	}
}

func (f *fakePingMonitorOCIClient) CreatePingMonitor(ctx context.Context, request healthcheckssdk.CreatePingMonitorRequest) (healthcheckssdk.CreatePingMonitorResponse, error) {
	f.createCalls++
	if f.createPingMonitor == nil {
		return healthcheckssdk.CreatePingMonitorResponse{}, nil
	}
	return f.createPingMonitor(ctx, request)
}

func (f *fakePingMonitorOCIClient) GetPingMonitor(ctx context.Context, request healthcheckssdk.GetPingMonitorRequest) (healthcheckssdk.GetPingMonitorResponse, error) {
	f.getCalls++
	if f.getPingMonitor == nil {
		return healthcheckssdk.GetPingMonitorResponse{}, nil
	}
	return f.getPingMonitor(ctx, request)
}

func (f *fakePingMonitorOCIClient) ListPingMonitors(ctx context.Context, request healthcheckssdk.ListPingMonitorsRequest) (healthcheckssdk.ListPingMonitorsResponse, error) {
	f.listCalls++
	if f.listPingMonitors == nil {
		return healthcheckssdk.ListPingMonitorsResponse{}, nil
	}
	return f.listPingMonitors(ctx, request)
}

func (f *fakePingMonitorOCIClient) UpdatePingMonitor(ctx context.Context, request healthcheckssdk.UpdatePingMonitorRequest) (healthcheckssdk.UpdatePingMonitorResponse, error) {
	f.updateCalls++
	if f.updatePingMonitor == nil {
		return healthcheckssdk.UpdatePingMonitorResponse{}, nil
	}
	return f.updatePingMonitor(ctx, request)
}

func (f *fakePingMonitorOCIClient) DeletePingMonitor(ctx context.Context, request healthcheckssdk.DeletePingMonitorRequest) (healthcheckssdk.DeletePingMonitorResponse, error) {
	f.deleteCalls++
	if f.deletePingMonitor == nil {
		return healthcheckssdk.DeletePingMonitorResponse{}, nil
	}
	return f.deletePingMonitor(ctx, request)
}

func newTestPingMonitorClient(client pingMonitorOCIClient) PingMonitorServiceClient {
	return newPingMonitorServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: logr.Discard()},
		client,
	)
}

func newPingMonitorResource() *healthchecksv1beta1.PingMonitor {
	return &healthchecksv1beta1.PingMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPingMonitorDisplayName,
			Namespace: "default",
			UID:       k8stypes.UID("ping-monitor-uid"),
		},
		Spec: healthchecksv1beta1.PingMonitorSpec{
			CompartmentId:     testPingMonitorCompartmentID,
			Targets:           []string{"198.51.100.10"},
			Protocol:          testPingMonitorProtocol,
			DisplayName:       testPingMonitorDisplayName,
			IntervalInSeconds: 30,
			VantagePointNames: []string{"aws-iad"},
			Port:              443,
			TimeoutInSeconds:  10,
			IsEnabled:         true,
			FreeformTags:      map[string]string{"managed-by": "osok"},
			DefinedTags:       map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func reconcileRequest(resource *healthchecksv1beta1.PingMonitor) ctrl.Request {
	return ctrl.Request{
		NamespacedName: k8stypes.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func pingMonitorBody(id string, compartmentID string, displayName string, enabled bool) healthcheckssdk.PingMonitor {
	return healthcheckssdk.PingMonitor{
		Id:                common.String(id),
		CompartmentId:     common.String(compartmentID),
		Targets:           []string{"198.51.100.10"},
		VantagePointNames: []string{"aws-iad"},
		Port:              common.Int(443),
		TimeoutInSeconds:  common.Int(10),
		Protocol:          healthcheckssdk.PingProbeProtocolEnum(testPingMonitorProtocol),
		DisplayName:       common.String(displayName),
		IntervalInSeconds: common.Int(30),
		IsEnabled:         common.Bool(enabled),
		FreeformTags:      map[string]string{"managed-by": "osok"},
		DefinedTags:       map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func pingMonitorSummary(id string, compartmentID string, displayName string) healthcheckssdk.PingMonitorSummary {
	return healthcheckssdk.PingMonitorSummary{
		Id:                common.String(id),
		CompartmentId:     common.String(compartmentID),
		DisplayName:       common.String(displayName),
		Protocol:          healthcheckssdk.PingProbeProtocolEnum(testPingMonitorProtocol),
		IntervalInSeconds: common.Int(30),
		IsEnabled:         common.Bool(true),
		FreeformTags:      map[string]string{"managed-by": "osok"},
		DefinedTags:       map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func assertTrailingCondition(t *testing.T, resource *healthchecksv1beta1.PingMonitor, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.conditions empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("trailing condition = %s, want %s", got, want)
	}
}

func assertContainsAll(t *testing.T, got []string, want ...string) {
	t.Helper()
	seen := make(map[string]bool, len(got))
	for _, item := range got {
		seen[item] = true
	}
	for _, item := range want {
		if !seen[item] {
			t.Fatalf("slice %#v missing %q", got, item)
		}
	}
}
