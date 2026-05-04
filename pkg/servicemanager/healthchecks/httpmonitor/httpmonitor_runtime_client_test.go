/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package httpmonitor

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
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeHttpMonitorOCIClient struct {
	createFn func(context.Context, healthcheckssdk.CreateHttpMonitorRequest) (healthcheckssdk.CreateHttpMonitorResponse, error)
	getFn    func(context.Context, healthcheckssdk.GetHttpMonitorRequest) (healthcheckssdk.GetHttpMonitorResponse, error)
	listFn   func(context.Context, healthcheckssdk.ListHttpMonitorsRequest) (healthcheckssdk.ListHttpMonitorsResponse, error)
	updateFn func(context.Context, healthcheckssdk.UpdateHttpMonitorRequest) (healthcheckssdk.UpdateHttpMonitorResponse, error)
	deleteFn func(context.Context, healthcheckssdk.DeleteHttpMonitorRequest) (healthcheckssdk.DeleteHttpMonitorResponse, error)
}

func (f *fakeHttpMonitorOCIClient) CreateHttpMonitor(ctx context.Context, req healthcheckssdk.CreateHttpMonitorRequest) (healthcheckssdk.CreateHttpMonitorResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return healthcheckssdk.CreateHttpMonitorResponse{}, nil
}

func (f *fakeHttpMonitorOCIClient) GetHttpMonitor(ctx context.Context, req healthcheckssdk.GetHttpMonitorRequest) (healthcheckssdk.GetHttpMonitorResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return healthcheckssdk.GetHttpMonitorResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
}

func (f *fakeHttpMonitorOCIClient) ListHttpMonitors(ctx context.Context, req healthcheckssdk.ListHttpMonitorsRequest) (healthcheckssdk.ListHttpMonitorsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return healthcheckssdk.ListHttpMonitorsResponse{}, nil
}

func (f *fakeHttpMonitorOCIClient) UpdateHttpMonitor(ctx context.Context, req healthcheckssdk.UpdateHttpMonitorRequest) (healthcheckssdk.UpdateHttpMonitorResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return healthcheckssdk.UpdateHttpMonitorResponse{}, nil
}

func (f *fakeHttpMonitorOCIClient) DeleteHttpMonitor(ctx context.Context, req healthcheckssdk.DeleteHttpMonitorRequest) (healthcheckssdk.DeleteHttpMonitorResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return healthcheckssdk.DeleteHttpMonitorResponse{}, nil
}

func testHttpMonitorClient(fake *fakeHttpMonitorOCIClient) HttpMonitorServiceClient {
	return newHttpMonitorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: logr.Discard()}, fake)
}

func makeHttpMonitorResource() *healthchecksv1beta1.HttpMonitor {
	return &healthchecksv1beta1.HttpMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "http-monitor-sample",
			Namespace: "default",
			UID:       "http-monitor-uid",
		},
		Spec: healthchecksv1beta1.HttpMonitorSpec{
			CompartmentId:     "ocid1.compartment.oc1..example",
			Targets:           []string{"example.com"},
			Protocol:          string(healthcheckssdk.HttpProbeProtocolHttps),
			DisplayName:       "http-monitor-alpha",
			IntervalInSeconds: 30,
			VantagePointNames: []string{"aws-iad"},
			Port:              443,
			TimeoutInSeconds:  10,
			Method:            string(healthcheckssdk.HttpProbeMethodGet),
			Path:              "/health",
			Headers:           map[string]string{"User-Agent": "osok"},
			IsEnabled:         true,
			FreeformTags:      map[string]string{"managed-by": "osok"},
			DefinedTags:       map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeSDKHttpMonitor(id string) healthcheckssdk.HttpMonitor {
	return healthcheckssdk.HttpMonitor{
		Id:                common.String(id),
		ResultsUrl:        common.String("https://healthchecks.example/results"),
		HomeRegion:        common.String("us-ashburn-1"),
		CompartmentId:     common.String("ocid1.compartment.oc1..example"),
		Targets:           []string{"example.com"},
		VantagePointNames: []string{"aws-iad"},
		Port:              common.Int(443),
		TimeoutInSeconds:  common.Int(10),
		Protocol:          healthcheckssdk.HttpProbeProtocolHttps,
		Method:            healthcheckssdk.HttpProbeMethodGet,
		Path:              common.String("/health"),
		Headers:           map[string]string{"User-Agent": "osok"},
		DisplayName:       common.String("http-monitor-alpha"),
		IntervalInSeconds: common.Int(30),
		IsEnabled:         common.Bool(true),
		FreeformTags:      map[string]string{"managed-by": "osok"},
		DefinedTags:       map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
	}
}

func makeSDKHttpMonitorSummary(id string) healthcheckssdk.HttpMonitorSummary {
	return healthcheckssdk.HttpMonitorSummary{
		Id:                common.String(id),
		ResultsUrl:        common.String("https://healthchecks.example/results"),
		HomeRegion:        common.String("us-ashburn-1"),
		CompartmentId:     common.String("ocid1.compartment.oc1..example"),
		DisplayName:       common.String("http-monitor-alpha"),
		IntervalInSeconds: common.Int(30),
		IsEnabled:         common.Bool(true),
		FreeformTags:      map[string]string{"managed-by": "osok"},
		DefinedTags:       map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		Protocol:          healthcheckssdk.HttpProbeProtocolHttps,
	}
}

func assertHttpMonitorCreateRequest(t *testing.T, request healthcheckssdk.CreateHttpMonitorRequest, resource *healthchecksv1beta1.HttpMonitor) {
	t.Helper()
	if request.CompartmentId == nil || *request.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %v, want %q", request.CompartmentId, resource.Spec.CompartmentId)
	}
	if request.DisplayName == nil || *request.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("create displayName = %v, want %q", request.DisplayName, resource.Spec.DisplayName)
	}
	if request.Protocol != healthcheckssdk.HttpProbeProtocolHttps {
		t.Fatalf("create protocol = %q, want %q", request.Protocol, healthcheckssdk.HttpProbeProtocolHttps)
	}
	if request.OpcRetryToken == nil || *request.OpcRetryToken == "" {
		t.Fatal("create opcRetryToken is empty, want deterministic retry token")
	}
}

func assertHttpMonitorCreatedStatus(t *testing.T, resource *healthchecksv1beta1.HttpMonitor, getRequest healthcheckssdk.GetHttpMonitorRequest) {
	t.Helper()
	if getRequest.MonitorId == nil || *getRequest.MonitorId != "ocid1.httpmonitor.oc1..created" {
		t.Fatalf("get monitorId = %v, want created monitor ID", getRequest.MonitorId)
	}
	if resource.Status.Id != "ocid1.httpmonitor.oc1..created" {
		t.Fatalf("status.id = %q, want created monitor ID", resource.Status.Id)
	}
	if string(resource.Status.OsokStatus.Ocid) != "ocid1.httpmonitor.oc1..created" {
		t.Fatalf("status.ocid = %q, want created monitor ID", resource.Status.OsokStatus.Ocid)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-create-1")
	}
}

func assertTrackedHttpMonitorGetRequest(t *testing.T, request healthcheckssdk.GetHttpMonitorRequest, want string) {
	t.Helper()
	if request.MonitorId == nil || *request.MonitorId != want {
		t.Fatalf("get monitorId = %v, want %s", request.MonitorId, want)
	}
}

func assertTrackedHttpMonitorDeleteRequest(t *testing.T, request healthcheckssdk.DeleteHttpMonitorRequest, want string) {
	t.Helper()
	if request.MonitorId == nil || *request.MonitorId != want {
		t.Fatalf("delete monitorId = %v, want %s", request.MonitorId, want)
	}
}

func httpMonitorPagedListResponse(t *testing.T, call int, request healthcheckssdk.ListHttpMonitorsRequest) healthcheckssdk.ListHttpMonitorsResponse {
	t.Helper()
	if request.CompartmentId == nil || *request.CompartmentId != "ocid1.compartment.oc1..example" {
		t.Fatalf("list compartmentId = %v, want spec compartment", request.CompartmentId)
	}
	if request.DisplayName == nil || *request.DisplayName != "http-monitor-alpha" {
		t.Fatalf("list displayName = %v, want spec displayName", request.DisplayName)
	}
	switch call {
	case 1:
		if request.Page != nil {
			t.Fatalf("first list page = %v, want nil", request.Page)
		}
		return healthcheckssdk.ListHttpMonitorsResponse{
			Items:       []healthcheckssdk.HttpMonitorSummary{},
			OpcNextPage: common.String("page-2"),
		}
	case 2:
		if request.Page == nil || *request.Page != "page-2" {
			t.Fatalf("second list page = %v, want page-2", request.Page)
		}
		return healthcheckssdk.ListHttpMonitorsResponse{
			Items: []healthcheckssdk.HttpMonitorSummary{
				makeSDKHttpMonitorSummary("ocid1.httpmonitor.oc1..existing"),
			},
		}
	default:
		t.Fatalf("unexpected ListHttpMonitors() call %d", call)
		return healthcheckssdk.ListHttpMonitorsResponse{}
	}
}

func httpMonitorMutableReadback(t *testing.T, call int, request healthcheckssdk.GetHttpMonitorRequest) healthcheckssdk.GetHttpMonitorResponse {
	t.Helper()
	assertTrackedHttpMonitorGetRequest(t, request, "ocid1.httpmonitor.oc1..existing")
	monitor := makeSDKHttpMonitor("ocid1.httpmonitor.oc1..existing")
	switch call {
	case 1:
		monitor.Path = common.String("/old")
		monitor.IsEnabled = common.Bool(true)
	case 2:
		monitor.Path = common.String("/desired")
		monitor.IsEnabled = common.Bool(false)
	default:
		t.Fatalf("unexpected GetHttpMonitor() call %d", call)
	}
	return healthcheckssdk.GetHttpMonitorResponse{HttpMonitor: monitor}
}

func assertHttpMonitorUpdateRequest(t *testing.T, request healthcheckssdk.UpdateHttpMonitorRequest) {
	t.Helper()
	if request.MonitorId == nil || *request.MonitorId != "ocid1.httpmonitor.oc1..existing" {
		t.Fatalf("update monitorId = %v, want tracked monitor ID", request.MonitorId)
	}
	if request.Path == nil || *request.Path != "/desired" {
		t.Fatalf("update path = %v, want /desired", request.Path)
	}
	if request.IsEnabled == nil || *request.IsEnabled {
		t.Fatalf("update isEnabled = %v, want explicit false", request.IsEnabled)
	}
	if request.DisplayName != nil {
		t.Fatalf("update displayName = %v, want nil when displayName did not drift", request.DisplayName)
	}
}

func assertHttpMonitorDeletePendingStatus(t *testing.T, resource *healthchecksv1beta1.HttpMonitor) {
	t.Helper()
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-delete-1")
	}
	if resource.Status.OsokStatus.Async.Current == nil || resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want delete tracker", resource.Status.OsokStatus.Async.Current)
	}
}

func TestHttpMonitorServiceClientCreateOrUpdateCreatesMonitorAndRefreshesStatus(t *testing.T) {
	t.Parallel()

	var createRequest healthcheckssdk.CreateHttpMonitorRequest
	var getRequest healthcheckssdk.GetHttpMonitorRequest

	client := testHttpMonitorClient(&fakeHttpMonitorOCIClient{
		createFn: func(_ context.Context, req healthcheckssdk.CreateHttpMonitorRequest) (healthcheckssdk.CreateHttpMonitorResponse, error) {
			createRequest = req
			return healthcheckssdk.CreateHttpMonitorResponse{
				OpcRequestId: common.String("opc-create-1"),
				HttpMonitor:  makeSDKHttpMonitor("ocid1.httpmonitor.oc1..created"),
			}, nil
		},
		getFn: func(_ context.Context, req healthcheckssdk.GetHttpMonitorRequest) (healthcheckssdk.GetHttpMonitorResponse, error) {
			getRequest = req
			return healthcheckssdk.GetHttpMonitorResponse{
				HttpMonitor: makeSDKHttpMonitor("ocid1.httpmonitor.oc1..created"),
			}, nil
		},
	})

	resource := makeHttpMonitorResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	assertHttpMonitorCreateRequest(t, createRequest, resource)
	assertHttpMonitorCreatedStatus(t, resource, getRequest)
}

func TestHttpMonitorServiceClientCreateOrUpdateBindsExistingMonitorAcrossListPages(t *testing.T) {
	t.Parallel()

	createCalls := 0
	listCalls := 0
	getCalls := 0

	client := testHttpMonitorClient(&fakeHttpMonitorOCIClient{
		createFn: func(context.Context, healthcheckssdk.CreateHttpMonitorRequest) (healthcheckssdk.CreateHttpMonitorResponse, error) {
			createCalls++
			return healthcheckssdk.CreateHttpMonitorResponse{}, nil
		},
		listFn: func(_ context.Context, req healthcheckssdk.ListHttpMonitorsRequest) (healthcheckssdk.ListHttpMonitorsResponse, error) {
			listCalls++
			return httpMonitorPagedListResponse(t, listCalls, req), nil
		},
		getFn: func(_ context.Context, req healthcheckssdk.GetHttpMonitorRequest) (healthcheckssdk.GetHttpMonitorResponse, error) {
			getCalls++
			assertTrackedHttpMonitorGetRequest(t, req, "ocid1.httpmonitor.oc1..existing")
			return healthcheckssdk.GetHttpMonitorResponse{
				HttpMonitor: makeSDKHttpMonitor("ocid1.httpmonitor.oc1..existing"),
			}, nil
		},
	})

	resource := makeHttpMonitorResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue when bind readback is current")
	}
	if createCalls != 0 {
		t.Fatalf("CreateHttpMonitor() calls = %d, want 0", createCalls)
	}
	if listCalls != 2 {
		t.Fatalf("ListHttpMonitors() calls = %d, want 2", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetHttpMonitor() calls = %d, want 1", getCalls)
	}
	if resource.Status.Id != "ocid1.httpmonitor.oc1..existing" {
		t.Fatalf("status.id = %q, want existing monitor ID", resource.Status.Id)
	}
}

func TestHttpMonitorServiceClientCreateOrUpdateDoesNotUpdateMatchingMonitor(t *testing.T) {
	t.Parallel()

	updateCalls := 0
	getCalls := 0

	client := testHttpMonitorClient(&fakeHttpMonitorOCIClient{
		getFn: func(_ context.Context, req healthcheckssdk.GetHttpMonitorRequest) (healthcheckssdk.GetHttpMonitorResponse, error) {
			getCalls++
			assertTrackedHttpMonitorGetRequest(t, req, "ocid1.httpmonitor.oc1..existing")
			return healthcheckssdk.GetHttpMonitorResponse{
				HttpMonitor: makeSDKHttpMonitor("ocid1.httpmonitor.oc1..existing"),
			}, nil
		},
		updateFn: func(context.Context, healthcheckssdk.UpdateHttpMonitorRequest) (healthcheckssdk.UpdateHttpMonitorResponse, error) {
			updateCalls++
			return healthcheckssdk.UpdateHttpMonitorResponse{}, nil
		},
	})

	resource := makeHttpMonitorResource()
	resource.Status.Id = "ocid1.httpmonitor.oc1..existing"
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue when no mutable field drifted")
	}
	if getCalls != 1 {
		t.Fatalf("GetHttpMonitor() calls = %d, want 1", getCalls)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateHttpMonitor() calls = %d, want 0", updateCalls)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil for synchronous no-op", resource.Status.OsokStatus.Async.Current)
	}
}

func TestHttpMonitorServiceClientCreateOrUpdateUpdatesMutableDriftAndPreservesFalseBool(t *testing.T) {
	t.Parallel()

	getCalls := 0
	var updateRequest healthcheckssdk.UpdateHttpMonitorRequest

	client := testHttpMonitorClient(&fakeHttpMonitorOCIClient{
		getFn: func(_ context.Context, req healthcheckssdk.GetHttpMonitorRequest) (healthcheckssdk.GetHttpMonitorResponse, error) {
			getCalls++
			return httpMonitorMutableReadback(t, getCalls, req), nil
		},
		updateFn: func(_ context.Context, req healthcheckssdk.UpdateHttpMonitorRequest) (healthcheckssdk.UpdateHttpMonitorResponse, error) {
			updateRequest = req
			return healthcheckssdk.UpdateHttpMonitorResponse{
				OpcRequestId: common.String("opc-update-1"),
				HttpMonitor:  makeSDKHttpMonitor("ocid1.httpmonitor.oc1..existing"),
			}, nil
		},
	})

	resource := makeHttpMonitorResource()
	resource.Spec.Path = "/desired"
	resource.Spec.IsEnabled = false
	resource.Status.Id = "ocid1.httpmonitor.oc1..existing"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	assertHttpMonitorUpdateRequest(t, updateRequest)
	if resource.Status.Path != "/desired" {
		t.Fatalf("status.path = %q, want /desired", resource.Status.Path)
	}
	if resource.Status.IsEnabled {
		t.Fatal("status.isEnabled = true, want false")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, "opc-update-1")
	}
}

func TestHttpMonitorServiceClientCreateOrUpdateRejectsCompartmentDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	updateCalls := 0

	client := testHttpMonitorClient(&fakeHttpMonitorOCIClient{
		getFn: func(context.Context, healthcheckssdk.GetHttpMonitorRequest) (healthcheckssdk.GetHttpMonitorResponse, error) {
			monitor := makeSDKHttpMonitor("ocid1.httpmonitor.oc1..existing")
			monitor.CompartmentId = common.String("ocid1.compartment.oc1..observed")
			return healthcheckssdk.GetHttpMonitorResponse{HttpMonitor: monitor}, nil
		},
		updateFn: func(context.Context, healthcheckssdk.UpdateHttpMonitorRequest) (healthcheckssdk.UpdateHttpMonitorResponse, error) {
			updateCalls++
			return healthcheckssdk.UpdateHttpMonitorResponse{}, nil
		},
	})

	resource := makeHttpMonitorResource()
	resource.Status.Id = "ocid1.httpmonitor.oc1..existing"
	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartment replacement failure", err)
	}
	if updateCalls != 0 {
		t.Fatalf("UpdateHttpMonitor() calls = %d, want 0", updateCalls)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Failed)
	}
}

func TestHttpMonitorServiceClientDeleteKeepsFinalizerUntilReadbackIsGone(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0

	client := testHttpMonitorClient(&fakeHttpMonitorOCIClient{
		getFn: func(_ context.Context, req healthcheckssdk.GetHttpMonitorRequest) (healthcheckssdk.GetHttpMonitorResponse, error) {
			getCalls++
			assertTrackedHttpMonitorGetRequest(t, req, "ocid1.httpmonitor.oc1..existing")
			return healthcheckssdk.GetHttpMonitorResponse{
				HttpMonitor: makeSDKHttpMonitor("ocid1.httpmonitor.oc1..existing"),
			}, nil
		},
		deleteFn: func(_ context.Context, req healthcheckssdk.DeleteHttpMonitorRequest) (healthcheckssdk.DeleteHttpMonitorResponse, error) {
			deleteCalls++
			assertTrackedHttpMonitorDeleteRequest(t, req, "ocid1.httpmonitor.oc1..existing")
			return healthcheckssdk.DeleteHttpMonitorResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	})

	resource := makeHttpMonitorResource()
	resource.Status.Id = "ocid1.httpmonitor.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.httpmonitor.oc1..existing"

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep finalizer while readback still finds the monitor")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteHttpMonitor() calls = %d, want 1", deleteCalls)
	}
	if getCalls != 3 {
		t.Fatalf("GetHttpMonitor() calls = %d, want 3", getCalls)
	}
	assertHttpMonitorDeletePendingStatus(t, resource)
}

func TestHttpMonitorServiceClientDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	t.Parallel()

	getCalls := 0
	client := testHttpMonitorClient(&fakeHttpMonitorOCIClient{
		getFn: func(context.Context, healthcheckssdk.GetHttpMonitorRequest) (healthcheckssdk.GetHttpMonitorResponse, error) {
			getCalls++
			if getCalls <= 2 {
				return healthcheckssdk.GetHttpMonitorResponse{
					HttpMonitor: makeSDKHttpMonitor("ocid1.httpmonitor.oc1..existing"),
				}, nil
			}
			return healthcheckssdk.GetHttpMonitorResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "deleted")
		},
		deleteFn: func(context.Context, healthcheckssdk.DeleteHttpMonitorRequest) (healthcheckssdk.DeleteHttpMonitorResponse, error) {
			return healthcheckssdk.DeleteHttpMonitorResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	resource := makeHttpMonitorResource()
	resource.Status.Id = "ocid1.httpmonitor.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.httpmonitor.oc1..existing"

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want confirmed deletion")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want deletion timestamp")
	}
}

func TestHttpMonitorServiceClientDeleteTreatsAuthShapedNotFoundAsAmbiguous(t *testing.T) {
	t.Parallel()

	getCalls := 0
	client := testHttpMonitorClient(&fakeHttpMonitorOCIClient{
		getFn: func(context.Context, healthcheckssdk.GetHttpMonitorRequest) (healthcheckssdk.GetHttpMonitorResponse, error) {
			getCalls++
			if getCalls <= 2 {
				return healthcheckssdk.GetHttpMonitorResponse{
					HttpMonitor: makeSDKHttpMonitor("ocid1.httpmonitor.oc1..existing"),
				}, nil
			}
			return healthcheckssdk.GetHttpMonitorResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, healthcheckssdk.DeleteHttpMonitorRequest) (healthcheckssdk.DeleteHttpMonitorResponse, error) {
			return healthcheckssdk.DeleteHttpMonitorResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	})

	resource := makeHttpMonitorResource()
	resource.Status.Id = "ocid1.httpmonitor.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.httpmonitor.oc1..existing"

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous auth-shaped 404 failure", err)
	}
	if deleted {
		t.Fatal("Delete() should not report deleted for auth-shaped 404")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want surfaced auth error request ID", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.deletedAt = %v, want nil for ambiguous delete", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestHttpMonitorServiceClientDeleteFailsFastOnPreDeleteAuthShapedNotFound(t *testing.T) {
	t.Parallel()

	getCalls := 0
	deleteCalls := 0
	client := testHttpMonitorClient(&fakeHttpMonitorOCIClient{
		getFn: func(context.Context, healthcheckssdk.GetHttpMonitorRequest) (healthcheckssdk.GetHttpMonitorResponse, error) {
			getCalls++
			return healthcheckssdk.GetHttpMonitorResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteFn: func(context.Context, healthcheckssdk.DeleteHttpMonitorRequest) (healthcheckssdk.DeleteHttpMonitorResponse, error) {
			deleteCalls++
			return healthcheckssdk.DeleteHttpMonitorResponse{}, nil
		},
	})

	resource := makeHttpMonitorResource()
	resource.Status.Id = "ocid1.httpmonitor.oc1..existing"
	resource.Status.OsokStatus.Ocid = "ocid1.httpmonitor.oc1..existing"

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want pre-delete auth-shaped 404 failure", err)
	}
	if deleted {
		t.Fatal("Delete() should not report deleted for pre-delete auth-shaped 404")
	}
	if getCalls != 1 {
		t.Fatalf("GetHttpMonitor() calls = %d, want 1", getCalls)
	}
	if deleteCalls != 0 {
		t.Fatalf("DeleteHttpMonitor() calls = %d, want 0", deleteCalls)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want surfaced auth error request ID", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestHttpMonitorServiceClientCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	client := testHttpMonitorClient(&fakeHttpMonitorOCIClient{
		createFn: func(context.Context, healthcheckssdk.CreateHttpMonitorRequest) (healthcheckssdk.CreateHttpMonitorResponse, error) {
			return healthcheckssdk.CreateHttpMonitorResponse{}, errortest.NewServiceError(500, "InternalError", "create failed")
		},
	})

	resource := makeHttpMonitorResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want error request ID", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Failed)
	}
}
