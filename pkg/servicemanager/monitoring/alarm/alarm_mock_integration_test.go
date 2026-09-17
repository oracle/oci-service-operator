/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package alarm

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	monitoringsdk "github.com/oracle/oci-go-sdk/v65/monitoring"
	monitoringv1beta1 "github.com/oracle/oci-service-operator/api/monitoring/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockAlarmID = "ocid1.alarm.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/monitoring/alarm and formal/imports/monitoring/alarm.json
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/monitoring
func TestMockIntegrationAlarmLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &monitoringv1beta1.Alarm{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-alarm", Namespace: "default", UID: types.UID("mock-alarm-uid")},
		Spec: monitoringv1beta1.AlarmSpec{
			DisplayName:         "mock-alarm",
			CompartmentId:       "ocid1.compartment.oc1..mock",
			MetricCompartmentId: "ocid1.compartment.oc1..mock",
			Namespace:           "oci_computeagent",
			Query:               "CpuUtilization[1m].mean() > 100",
			Severity:            "CRITICAL",
			Destinations:        []string{"ocid1.onstopic.oc1..mock"},
			IsEnabled:           false,
			Body:                "Disabled mock integration alarm.",
			FreeformTags:        map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newAlarmMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://telemetry.mock.invalid", BasePath: "20180401", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Alarm OCI mock: %v", err)
		}
	})

	client := newMockAlarmClient(monitoringsdk.MonitoringClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*monitoringv1beta1.Alarm]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *monitoringv1beta1.Alarm) error {
			if current.Status.Id != mockAlarmID ||
				current.Status.DisplayName != resource.Spec.DisplayName ||
				current.Status.LifecycleState != string(monitoringsdk.AlarmLifecycleStateActive) ||
				current.Status.IsEnabled {
				return fmt.Errorf("created Alarm status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *monitoringv1beta1.Alarm) {
			current.Spec.DisplayName = "mock-alarm-updated"
			current.Spec.Query = "CpuUtilization[1m].mean() > 99"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *monitoringv1beta1.Alarm) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.Query != current.Spec.Query ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Alarm status = %+v", current.Status)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func newAlarmMockResponder(resource *monitoringv1beta1.Alarm) (*ocimock.CRUDResponder[monitoringsdk.Alarm], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	deleteReadObserved := false
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[monitoringsdk.Alarm]{
		CollectionPath:         "/20180401/alarms",
		ItemPath:               "/20180401/alarms/" + mockAlarmID,
		ExpectedOperations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:      true,
		RequireUpdateRead:      true,
		RequireDeleteRead:      true,
		RetainStateAfterDelete: true,
		Create: func(request ocimock.Request) (monitoringsdk.Alarm, ocimock.Response, error) {
			var details monitoringsdk.CreateAlarmDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return monitoringsdk.Alarm{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return monitoringsdk.Alarm{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName ||
				details.Query == nil || *details.Query != resource.Spec.Query ||
				details.IsEnabled == nil || *details.IsEnabled || len(details.Destinations) != 1 {
				return monitoringsdk.Alarm{}, ocimock.Response{}, fmt.Errorf("unexpected CreateAlarm details: %+v", details)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return monitoringsdk.Alarm{}, ocimock.Response{}, fmt.Errorf("CreateAlarm opc-retry-token is empty")
			}
			state := monitoringsdk.Alarm{
				Id: common.String(mockAlarmID), DisplayName: details.DisplayName,
				CompartmentId: details.CompartmentId, MetricCompartmentId: details.MetricCompartmentId,
				Namespace: details.Namespace, Query: details.Query, Severity: monitoringsdk.AlarmSeverityEnum(details.Severity),
				Destinations: append([]string(nil), details.Destinations...), IsEnabled: details.IsEnabled,
				Body: details.Body, FreeformTags: details.FreeformTags,
				LifecycleState: monitoringsdk.AlarmLifecycleStateActive, TimeCreated: &now, TimeUpdated: &now,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		ReadTransition: func(_ ocimock.Request, state monitoringsdk.Alarm) (monitoringsdk.Alarm, ocimock.Response, error) {
			if state.LifecycleState == monitoringsdk.AlarmLifecycleStateDeleting {
				if deleteReadObserved {
					state.LifecycleState = monitoringsdk.AlarmLifecycleStateDeleted
				} else {
					deleteReadObserved = true
				}
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Update: func(request ocimock.Request, state monitoringsdk.Alarm) (monitoringsdk.Alarm, ocimock.Response, error) {
			var details monitoringsdk.UpdateAlarmDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return monitoringsdk.Alarm{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-alarm-updated" ||
				details.Query == nil || *details.Query != "CpuUtilization[1m].mean() > 99" ||
				details.FreeformTags["osok-mock"] != "update" {
				return monitoringsdk.Alarm{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateAlarm details: %+v", details)
			}
			state.DisplayName = details.DisplayName
			state.Query = details.Query
			state.FreeformTags = details.FreeformTags
			state.TimeUpdated = &now
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state monitoringsdk.Alarm) (monitoringsdk.Alarm, ocimock.Response, error) {
			state.LifecycleState = monitoringsdk.AlarmLifecycleStateDeleting
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
