/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package alarm

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	monitoringsdk "github.com/oracle/oci-go-sdk/v65/monitoring"
	monitoringv1beta1 "github.com/oracle/oci-service-operator/api/monitoring/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type alarmOCIClient interface {
	CreateAlarm(context.Context, monitoringsdk.CreateAlarmRequest) (monitoringsdk.CreateAlarmResponse, error)
	GetAlarm(context.Context, monitoringsdk.GetAlarmRequest) (monitoringsdk.GetAlarmResponse, error)
	ListAlarms(context.Context, monitoringsdk.ListAlarmsRequest) (monitoringsdk.ListAlarmsResponse, error)
	UpdateAlarm(context.Context, monitoringsdk.UpdateAlarmRequest) (monitoringsdk.UpdateAlarmResponse, error)
	DeleteAlarm(context.Context, monitoringsdk.DeleteAlarmRequest) (monitoringsdk.DeleteAlarmResponse, error)
}

type fakeAlarmOCIClient struct {
	createFn func(context.Context, monitoringsdk.CreateAlarmRequest) (monitoringsdk.CreateAlarmResponse, error)
	getFn    func(context.Context, monitoringsdk.GetAlarmRequest) (monitoringsdk.GetAlarmResponse, error)
	listFn   func(context.Context, monitoringsdk.ListAlarmsRequest) (monitoringsdk.ListAlarmsResponse, error)
	updateFn func(context.Context, monitoringsdk.UpdateAlarmRequest) (monitoringsdk.UpdateAlarmResponse, error)
	deleteFn func(context.Context, monitoringsdk.DeleteAlarmRequest) (monitoringsdk.DeleteAlarmResponse, error)
}

func (f *fakeAlarmOCIClient) CreateAlarm(ctx context.Context, req monitoringsdk.CreateAlarmRequest) (monitoringsdk.CreateAlarmResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return monitoringsdk.CreateAlarmResponse{}, nil
}

func (f *fakeAlarmOCIClient) GetAlarm(ctx context.Context, req monitoringsdk.GetAlarmRequest) (monitoringsdk.GetAlarmResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return monitoringsdk.GetAlarmResponse{}, nil
}

func (f *fakeAlarmOCIClient) ListAlarms(ctx context.Context, req monitoringsdk.ListAlarmsRequest) (monitoringsdk.ListAlarmsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return monitoringsdk.ListAlarmsResponse{}, nil
}

func (f *fakeAlarmOCIClient) UpdateAlarm(ctx context.Context, req monitoringsdk.UpdateAlarmRequest) (monitoringsdk.UpdateAlarmResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return monitoringsdk.UpdateAlarmResponse{}, nil
}

func (f *fakeAlarmOCIClient) DeleteAlarm(ctx context.Context, req monitoringsdk.DeleteAlarmRequest) (monitoringsdk.DeleteAlarmResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return monitoringsdk.DeleteAlarmResponse{}, nil
}

func newTestAlarmClient(client alarmOCIClient) defaultAlarmServiceClient {
	if client == nil {
		client = &fakeAlarmOCIClient{}
	}

	return defaultAlarmServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*monitoringv1beta1.Alarm](generatedruntime.Config[*monitoringv1beta1.Alarm]{
			Kind:      "Alarm",
			SDKName:   "Alarm",
			Log:       loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
			Semantics: newAlarmRuntimeSemantics(),
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &monitoringsdk.CreateAlarmRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return client.CreateAlarm(ctx, *request.(*monitoringsdk.CreateAlarmRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CreateAlarmDetails", RequestName: "CreateAlarmDetails", Contribution: "body", PreferResourceID: false},
				},
			},
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &monitoringsdk.GetAlarmRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return client.GetAlarm(ctx, *request.(*monitoringsdk.GetAlarmRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "AlarmId", RequestName: "alarmId", Contribution: "path", PreferResourceID: true},
				},
			},
			List: &generatedruntime.Operation{
				NewRequest: func() any { return &monitoringsdk.ListAlarmsRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return client.ListAlarms(ctx, *request.(*monitoringsdk.ListAlarmsRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
					{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
					{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
					{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", PreferResourceID: false},
					{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query", PreferResourceID: false},
					{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
					{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
					{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query", PreferResourceID: false},
				},
			},
			Update: &generatedruntime.Operation{
				NewRequest: func() any { return &monitoringsdk.UpdateAlarmRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return client.UpdateAlarm(ctx, *request.(*monitoringsdk.UpdateAlarmRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "AlarmId", RequestName: "alarmId", Contribution: "path", PreferResourceID: true},
					{FieldName: "UpdateAlarmDetails", RequestName: "UpdateAlarmDetails", Contribution: "body", PreferResourceID: false},
				},
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &monitoringsdk.DeleteAlarmRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return client.DeleteAlarm(ctx, *request.(*monitoringsdk.DeleteAlarmRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "AlarmId", RequestName: "alarmId", Contribution: "path", PreferResourceID: true},
				},
			},
		}),
	}
}

func makeAlarmResource() *monitoringv1beta1.Alarm {
	return &monitoringv1beta1.Alarm{
		Spec: monitoringv1beta1.AlarmSpec{
			DisplayName:         "osok-alarm-sample",
			CompartmentId:       "ocid1.compartment.oc1..alarmcompartment",
			MetricCompartmentId: "ocid1.compartment.oc1..metriccompartment",
			Namespace:           "oci_computeagent",
			Query:               "CpuUtilization[1m].mean() > 85",
			Severity:            "CRITICAL",
			Destinations:        []string{"ocid1.onstopic.oc1..exampleuniqueID"},
			IsEnabled:           true,
			Body:                "Follow the alarm runbook.",
		},
	}
}

func makeSDKAlarm(id string, resource *monitoringv1beta1.Alarm, lifecycleState monitoringsdk.AlarmLifecycleStateEnum, body string) monitoringsdk.Alarm {
	isEnabled := resource.Spec.IsEnabled
	metricCompartmentIDInSubtree := resource.Spec.MetricCompartmentIdInSubtree
	notificationsPerMetricDimension := resource.Spec.IsNotificationsPerMetricDimensionEnabled
	return monitoringsdk.Alarm{
		Id:                                       common.String(id),
		DisplayName:                              common.String(resource.Spec.DisplayName),
		CompartmentId:                            common.String(resource.Spec.CompartmentId),
		MetricCompartmentId:                      common.String(resource.Spec.MetricCompartmentId),
		Namespace:                                common.String(resource.Spec.Namespace),
		Query:                                    common.String(resource.Spec.Query),
		Severity:                                 monitoringsdk.AlarmSeverityEnum(resource.Spec.Severity),
		Destinations:                             append([]string(nil), resource.Spec.Destinations...),
		IsEnabled:                                common.Bool(isEnabled),
		LifecycleState:                           lifecycleState,
		MetricCompartmentIdInSubtree:             common.Bool(metricCompartmentIDInSubtree),
		Body:                                     common.String(body),
		IsNotificationsPerMetricDimensionEnabled: common.Bool(notificationsPerMetricDimension),
	}
}

func makeSDKAlarmSummary(id string, resource *monitoringv1beta1.Alarm, lifecycleState monitoringsdk.AlarmLifecycleStateEnum) monitoringsdk.AlarmSummary {
	isEnabled := resource.Spec.IsEnabled
	notificationsPerMetricDimension := resource.Spec.IsNotificationsPerMetricDimensionEnabled
	return monitoringsdk.AlarmSummary{
		Id:                                       common.String(id),
		DisplayName:                              common.String(resource.Spec.DisplayName),
		CompartmentId:                            common.String(resource.Spec.CompartmentId),
		MetricCompartmentId:                      common.String(resource.Spec.MetricCompartmentId),
		Namespace:                                common.String(resource.Spec.Namespace),
		Query:                                    common.String(resource.Spec.Query),
		Severity:                                 monitoringsdk.AlarmSummarySeverityEnum(resource.Spec.Severity),
		Destinations:                             append([]string(nil), resource.Spec.Destinations...),
		IsEnabled:                                common.Bool(isEnabled),
		LifecycleState:                           lifecycleState,
		IsNotificationsPerMetricDimensionEnabled: common.Bool(notificationsPerMetricDimension),
	}
}

func TestAlarmServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.alarm.oc1..created"

	resource := makeAlarmResource()
	listCalls := 0
	getCalls := 0
	var createRequest monitoringsdk.CreateAlarmRequest

	client := newTestAlarmClient(&fakeAlarmOCIClient{
		listFn: func(_ context.Context, req monitoringsdk.ListAlarmsRequest) (monitoringsdk.ListAlarmsResponse, error) {
			listCalls++
			if req.CompartmentId == nil || *req.CompartmentId != resource.Spec.CompartmentId {
				t.Fatalf("ListAlarmsRequest.CompartmentId = %v, want %q", req.CompartmentId, resource.Spec.CompartmentId)
			}
			if req.DisplayName == nil || *req.DisplayName != resource.Spec.DisplayName {
				t.Fatalf("ListAlarmsRequest.DisplayName = %v, want %q", req.DisplayName, resource.Spec.DisplayName)
			}
			return monitoringsdk.ListAlarmsResponse{}, nil
		},
		createFn: func(_ context.Context, req monitoringsdk.CreateAlarmRequest) (monitoringsdk.CreateAlarmResponse, error) {
			createRequest = req
			return monitoringsdk.CreateAlarmResponse{
				Alarm:        makeSDKAlarm(createdID, resource, monitoringsdk.AlarmLifecycleStateActive, resource.Spec.Body),
				OpcRequestId: common.String("opc-create-1"),
			}, nil
		},
		getFn: func(_ context.Context, req monitoringsdk.GetAlarmRequest) (monitoringsdk.GetAlarmResponse, error) {
			getCalls++
			if req.AlarmId == nil || *req.AlarmId != createdID {
				t.Fatalf("GetAlarmRequest.AlarmId = %v, want %q", req.AlarmId, createdID)
			}
			return monitoringsdk.GetAlarmResponse{
				Alarm: makeSDKAlarm(createdID, resource, monitoringsdk.AlarmLifecycleStateActive, resource.Spec.Body),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want no requeue after ACTIVE follow-up", response)
	}
	if listCalls != 1 {
		t.Fatalf("ListAlarms() calls = %d, want 1", listCalls)
	}
	if getCalls != 1 {
		t.Fatalf("GetAlarm() calls = %d, want 1 follow-up read", getCalls)
	}
	if createRequest.CreateAlarmDetails.DisplayName == nil || *createRequest.CreateAlarmDetails.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("CreateAlarmRequest.DisplayName = %v, want %q", createRequest.CreateAlarmDetails.DisplayName, resource.Spec.DisplayName)
	}
	if createRequest.CreateAlarmDetails.CompartmentId == nil || *createRequest.CreateAlarmDetails.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("CreateAlarmRequest.CompartmentId = %v, want %q", createRequest.CreateAlarmDetails.CompartmentId, resource.Spec.CompartmentId)
	}
	if createRequest.CreateAlarmDetails.MetricCompartmentId == nil || *createRequest.CreateAlarmDetails.MetricCompartmentId != resource.Spec.MetricCompartmentId {
		t.Fatalf("CreateAlarmRequest.MetricCompartmentId = %v, want %q", createRequest.CreateAlarmDetails.MetricCompartmentId, resource.Spec.MetricCompartmentId)
	}
	if createRequest.CreateAlarmDetails.Query == nil || *createRequest.CreateAlarmDetails.Query != resource.Spec.Query {
		t.Fatalf("CreateAlarmRequest.Query = %v, want %q", createRequest.CreateAlarmDetails.Query, resource.Spec.Query)
	}
	if got := string(createRequest.CreateAlarmDetails.Severity); got != resource.Spec.Severity {
		t.Fatalf("CreateAlarmRequest.Severity = %q, want %q", got, resource.Spec.Severity)
	}
	if len(createRequest.CreateAlarmDetails.Destinations) != len(resource.Spec.Destinations) || createRequest.CreateAlarmDetails.Destinations[0] != resource.Spec.Destinations[0] {
		t.Fatalf("CreateAlarmRequest.Destinations = %#v, want %#v", createRequest.CreateAlarmDetails.Destinations, resource.Spec.Destinations)
	}
	if createRequest.CreateAlarmDetails.IsEnabled == nil || *createRequest.CreateAlarmDetails.IsEnabled != resource.Spec.IsEnabled {
		t.Fatalf("CreateAlarmRequest.IsEnabled = %v, want %t", createRequest.CreateAlarmDetails.IsEnabled, resource.Spec.IsEnabled)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != createdID {
		t.Fatalf("status.ocid = %q, want %q", got, createdID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-create-1")
	}
	if got := resource.Status.Id; got != createdID {
		t.Fatalf("status.id = %q, want %q", got, createdID)
	}
	if got := resource.Status.LifecycleState; got != string(monitoringsdk.AlarmLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
}

func TestAlarmServiceClientBindsExistingAlarmByList(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.alarm.oc1..existing"

	resource := makeAlarmResource()
	createCalled := false
	updateCalled := false
	var listRequest monitoringsdk.ListAlarmsRequest
	getCalls := 0

	client := newTestAlarmClient(&fakeAlarmOCIClient{
		listFn: func(_ context.Context, req monitoringsdk.ListAlarmsRequest) (monitoringsdk.ListAlarmsResponse, error) {
			listRequest = req
			other := makeAlarmResource()
			other.Spec.DisplayName = "other-alarm"
			return monitoringsdk.ListAlarmsResponse{
				Items: []monitoringsdk.AlarmSummary{
					makeSDKAlarmSummary("ocid1.alarm.oc1..other", other, monitoringsdk.AlarmLifecycleStateActive),
					makeSDKAlarmSummary(existingID, resource, monitoringsdk.AlarmLifecycleStateActive),
				},
			}, nil
		},
		getFn: func(_ context.Context, req monitoringsdk.GetAlarmRequest) (monitoringsdk.GetAlarmResponse, error) {
			getCalls++
			if req.AlarmId == nil || *req.AlarmId != existingID {
				t.Fatalf("GetAlarmRequest.AlarmId = %v, want %q", req.AlarmId, existingID)
			}
			return monitoringsdk.GetAlarmResponse{
				Alarm: makeSDKAlarm(existingID, resource, monitoringsdk.AlarmLifecycleStateActive, resource.Spec.Body),
			}, nil
		},
		createFn: func(context.Context, monitoringsdk.CreateAlarmRequest) (monitoringsdk.CreateAlarmResponse, error) {
			createCalled = true
			return monitoringsdk.CreateAlarmResponse{}, nil
		},
		updateFn: func(context.Context, monitoringsdk.UpdateAlarmRequest) (monitoringsdk.UpdateAlarmResponse, error) {
			updateCalled = true
			return monitoringsdk.UpdateAlarmResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want no requeue for ACTIVE bind", response)
	}
	if listRequest.CompartmentId == nil || *listRequest.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("ListAlarmsRequest.CompartmentId = %v, want %q", listRequest.CompartmentId, resource.Spec.CompartmentId)
	}
	if listRequest.DisplayName == nil || *listRequest.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("ListAlarmsRequest.DisplayName = %v, want %q", listRequest.DisplayName, resource.Spec.DisplayName)
	}
	if getCalls != 1 {
		t.Fatalf("GetAlarm() calls = %d, want 1 on the current bind path", getCalls)
	}
	if createCalled {
		t.Fatal("CreateAlarm() called, want list-bind path")
	}
	if updateCalled {
		t.Fatal("UpdateAlarm() called, want observe-only bind path")
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != existingID {
		t.Fatalf("status.ocid = %q, want %q", got, existingID)
	}
}

func TestAlarmServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.alarm.oc1..existing"

	resource := makeAlarmResource()
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Spec.Body = "Updated runbook instructions."

	getCalls := 0
	updateCalls := 0
	var updateRequest monitoringsdk.UpdateAlarmRequest

	client := newTestAlarmClient(&fakeAlarmOCIClient{
		getFn: func(_ context.Context, req monitoringsdk.GetAlarmRequest) (monitoringsdk.GetAlarmResponse, error) {
			getCalls++
			if req.AlarmId == nil || *req.AlarmId != existingID {
				t.Fatalf("GetAlarmRequest.AlarmId = %v, want %q", req.AlarmId, existingID)
			}
			current := makeAlarmResource()
			body := "Follow the old runbook."
			if getCalls > 1 {
				current.Spec.Body = resource.Spec.Body
				body = resource.Spec.Body
			}
			return monitoringsdk.GetAlarmResponse{
				Alarm: makeSDKAlarm(existingID, current, monitoringsdk.AlarmLifecycleStateActive, body),
			}, nil
		},
		updateFn: func(_ context.Context, req monitoringsdk.UpdateAlarmRequest) (monitoringsdk.UpdateAlarmResponse, error) {
			updateCalls++
			updateRequest = req
			return monitoringsdk.UpdateAlarmResponse{
				Alarm:        makeSDKAlarm(existingID, resource, monitoringsdk.AlarmLifecycleStateActive, resource.Spec.Body),
				OpcRequestId: common.String("opc-update-1"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want success", response)
	}
	if response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want no requeue after ACTIVE update follow-up", response)
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateAlarm() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetAlarm() calls = %d, want 2", getCalls)
	}
	if updateRequest.AlarmId == nil || *updateRequest.AlarmId != existingID {
		t.Fatalf("UpdateAlarmRequest.AlarmId = %v, want %q", updateRequest.AlarmId, existingID)
	}
	if updateRequest.UpdateAlarmDetails.Body == nil || *updateRequest.UpdateAlarmDetails.Body != resource.Spec.Body {
		t.Fatalf("UpdateAlarmRequest.Body = %v, want %q", updateRequest.UpdateAlarmDetails.Body, resource.Spec.Body)
	}
	if updateRequest.UpdateAlarmDetails.CompartmentId != nil {
		t.Fatalf("UpdateAlarmRequest.CompartmentId = %v, want nil for unchanged replacement-only field", updateRequest.UpdateAlarmDetails.CompartmentId)
	}
	if updateRequest.UpdateAlarmDetails.Query != nil {
		t.Fatalf("UpdateAlarmRequest.Query = %v, want nil when unchanged", updateRequest.UpdateAlarmDetails.Query)
	}
	if len(updateRequest.UpdateAlarmDetails.Destinations) != 0 {
		t.Fatalf("UpdateAlarmRequest.Destinations = %#v, want nil when unchanged", updateRequest.UpdateAlarmDetails.Destinations)
	}
	if got := resource.Status.Body; got != resource.Spec.Body {
		t.Fatalf("status.body = %q, want %q", got, resource.Spec.Body)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-update-1")
	}
}

func TestAlarmServiceClientRejectsReplacementOnlyCompartmentDrift(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.alarm.oc1..existing"

	resource := makeAlarmResource()
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..replacement"

	updateCalled := false

	client := newTestAlarmClient(&fakeAlarmOCIClient{
		getFn: func(_ context.Context, req monitoringsdk.GetAlarmRequest) (monitoringsdk.GetAlarmResponse, error) {
			if req.AlarmId == nil || *req.AlarmId != existingID {
				t.Fatalf("GetAlarmRequest.AlarmId = %v, want %q", req.AlarmId, existingID)
			}
			current := makeAlarmResource()
			return monitoringsdk.GetAlarmResponse{
				Alarm: makeSDKAlarm(existingID, current, monitoringsdk.AlarmLifecycleStateActive, current.Spec.Body),
			}, nil
		},
		updateFn: func(context.Context, monitoringsdk.UpdateAlarmRequest) (monitoringsdk.UpdateAlarmResponse, error) {
			updateCalled = true
			return monitoringsdk.UpdateAlarmResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-required compartment drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if updateCalled {
		t.Fatal("UpdateAlarm() should not be called when replacement-only drift is detected")
	}
}

func TestAlarmServiceClientDeleteConfirmsDeletion(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.alarm.oc1..existing"

	resource := makeAlarmResource()
	resource.Status.Id = existingID
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)

	getCalls := 0
	deleteCalls := 0

	client := newTestAlarmClient(&fakeAlarmOCIClient{
		getFn: func(_ context.Context, req monitoringsdk.GetAlarmRequest) (monitoringsdk.GetAlarmResponse, error) {
			getCalls++
			if req.AlarmId == nil || *req.AlarmId != existingID {
				t.Fatalf("GetAlarmRequest.AlarmId = %v, want %q", req.AlarmId, existingID)
			}
			state := monitoringsdk.AlarmLifecycleStateActive
			if getCalls == 2 {
				state = monitoringsdk.AlarmLifecycleStateDeleting
			}
			if getCalls > 2 {
				state = monitoringsdk.AlarmLifecycleStateDeleted
			}
			return monitoringsdk.GetAlarmResponse{
				Alarm: makeSDKAlarm(existingID, resource, state, resource.Spec.Body),
			}, nil
		},
		deleteFn: func(_ context.Context, req monitoringsdk.DeleteAlarmRequest) (monitoringsdk.DeleteAlarmResponse, error) {
			deleteCalls++
			if req.AlarmId == nil || *req.AlarmId != existingID {
				t.Fatalf("DeleteAlarmRequest.AlarmId = %v, want %q", req.AlarmId, existingID)
			}
			return monitoringsdk.DeleteAlarmResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first call = true, want delete confirmation requeue while DELETING")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteAlarm() calls after first delete = %d, want 1", deleteCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetAlarm() calls after first delete = %d, want 2", getCalls)
	}
	if got := resource.Status.LifecycleState; got != string(monitoringsdk.AlarmLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState after first delete = %q, want DELETING", got)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second call = false, want terminal delete confirmation")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteAlarm() calls after second delete = %d, want 1", deleteCalls)
	}
	if getCalls != 3 {
		t.Fatalf("GetAlarm() calls = %d, want 3", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if got := resource.Status.LifecycleState; got != string(monitoringsdk.AlarmLifecycleStateDeleted) {
		t.Fatalf("status.lifecycleState = %q, want DELETED", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-delete-1")
	}
}
