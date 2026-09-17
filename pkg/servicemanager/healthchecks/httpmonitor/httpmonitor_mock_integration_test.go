/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package httpmonitor

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	healthcheckssdk "github.com/oracle/oci-go-sdk/v65/healthchecks"
	healthchecksv1beta1 "github.com/oracle/oci-service-operator/api/healthchecks/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockHTTPMonitorID = "ocid1.httpmonitor.oc1..mock"

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal provider facts: formal/imports/healthchecks/httpmonitor.json
//   - repo-authored runtime: formal/controllers/healthchecks/httpmonitor/diagrams/runtime-lifecycle.yaml
//   - custom delete/pagination runtime: httpmonitor_runtime_client.go
//   - Terraform provider: terraform-provider-oci@eb653febb1ba internal/service/health_checks/health_checks_http_monitor_resource.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/healthchecks
func TestMockIntegrationHttpMonitorSynchronousCRUD(t *testing.T) {
	t.Parallel()

	resource := &healthchecksv1beta1.HttpMonitor{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-http-monitor", Namespace: "default", UID: types.UID("mock-http-monitor-uid")},
		Spec: healthchecksv1beta1.HttpMonitorSpec{
			CompartmentId:     "ocid1.compartment.oc1..mock",
			Targets:           []string{"example.com"},
			Protocol:          string(healthcheckssdk.HttpProbeProtocolHttps),
			DisplayName:       "mock-http-monitor",
			IntervalInSeconds: 60,
			Port:              443,
			TimeoutInSeconds:  10,
			Method:            string(healthcheckssdk.HttpProbeMethodGet),
			Path:              "/",
			IsEnabled:         false,
			FreeformTags:      map[string]string{"osok-mock": "create"},
		},
	}
	responder, err := newHTTPMonitorMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{
		Host:      "https://healthchecks.mock.invalid",
		BasePath:  "20180501",
		Responder: responder,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close HttpMonitor OCI mock: %v", err)
		}
	})

	sdkClient := healthcheckssdk.HealthChecksClient{BaseClient: session.BaseClient()}
	manager := &HttpMonitorServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newHttpMonitorRuntimeHooks(manager, sdkClient)
	client := wrapHttpMonitorGeneratedClient(hooks, defaultHttpMonitorServiceClient{ServiceClient: generatedruntime.NewServiceClient[*healthchecksv1beta1.HttpMonitor](
		buildHttpMonitorGeneratedRuntimeConfig(manager, hooks),
	)})

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*healthchecksv1beta1.HttpMonitor]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *healthchecksv1beta1.HttpMonitor) error {
			if current.Status.Id != mockHTTPMonitorID ||
				current.Status.DisplayName != "mock-http-monitor" ||
				current.Status.IntervalInSeconds != 60 ||
				current.Status.IsEnabled ||
				current.Status.FreeformTags["osok-mock"] != "create" {
				return fmt.Errorf("created HttpMonitor status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *healthchecksv1beta1.HttpMonitor) {
			current.Spec.DisplayName = "mock-http-monitor-updated"
			current.Spec.IntervalInSeconds = 30
			current.Spec.Path = "/health"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *healthchecksv1beta1.HttpMonitor) error {
			if current.Status.DisplayName != current.Spec.DisplayName ||
				current.Status.IntervalInSeconds != current.Spec.IntervalInSeconds ||
				current.Status.Path != current.Spec.Path ||
				current.Status.IsEnabled ||
				current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated HttpMonitor status = %+v", current.Status)
			}
			return nil
		},
		RetryDeleteError: func(err error) bool {
			return strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if counts := responder.OperationCounts(); counts[ocimock.OperationUpdate] != 2 {
		t.Fatalf("update operations = %d, want server-default correction plus requested update", counts[ocimock.OperationUpdate])
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func newHTTPMonitorMockResponder(resource *healthchecksv1beta1.HttpMonitor) (*ocimock.CRUDResponder[healthcheckssdk.HttpMonitor], error) {
	createdAt := common.SDKTime{Time: time.Date(2026, time.September, 1, 5, 43, 32, 0, time.UTC)}
	updateCount := 0
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[healthcheckssdk.HttpMonitor]{
		CollectionPath:     "/20180501/httpMonitors",
		ItemPath:           "/20180501/httpMonitors/" + mockHTTPMonitorID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		List: func(request ocimock.Request, present bool, state healthcheckssdk.HttpMonitor) (ocimock.Response, error) {
			if present {
				return ocimock.JSONResponse(http.StatusOK, []healthcheckssdk.HttpMonitor{state})
			}
			if got := request.URL.Query().Get("compartmentId"); got != resource.Spec.CompartmentId {
				return ocimock.Response{}, fmt.Errorf("list compartmentId = %q", got)
			}
			return ocimock.JSONResponse(http.StatusOK, []healthcheckssdk.HttpMonitor{})
		},
		Create: func(request ocimock.Request) (healthcheckssdk.HttpMonitor, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero healthcheckssdk.HttpMonitor
				return zero, ocimock.Response{}, err
			}
			var details healthcheckssdk.CreateHttpMonitorDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return healthcheckssdk.HttpMonitor{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return healthcheckssdk.HttpMonitor{}, ocimock.Response{}, err
			}
			expected := healthcheckssdk.CreateHttpMonitorDetails{
				CompartmentId:     common.String(resource.Spec.CompartmentId),
				Targets:           append([]string(nil), resource.Spec.Targets...),
				Protocol:          healthcheckssdk.HttpProbeProtocolEnum(resource.Spec.Protocol),
				DisplayName:       common.String(resource.Spec.DisplayName),
				IntervalInSeconds: common.Int(resource.Spec.IntervalInSeconds),
				Port:              common.Int(resource.Spec.Port),
				TimeoutInSeconds:  common.Int(resource.Spec.TimeoutInSeconds),
				Method:            healthcheckssdk.HttpProbeMethodEnum(resource.Spec.Method),
				Path:              common.String(resource.Spec.Path),
				FreeformTags:      map[string]string{"osok-mock": "create"},
			}
			if !reflect.DeepEqual(details, expected) {
				return healthcheckssdk.HttpMonitor{}, ocimock.Response{}, fmt.Errorf("create HttpMonitor details = %+v, want %+v", details, expected)
			}
			state := healthcheckssdk.HttpMonitor{
				Id:                common.String(mockHTTPMonitorID),
				ResultsUrl:        common.String("20180501/httpProbeResults/" + mockHTTPMonitorID),
				HomeRegion:        common.String("us-ashburn-1"),
				TimeCreated:       &createdAt,
				CompartmentId:     details.CompartmentId,
				Targets:           append([]string(nil), details.Targets...),
				Port:              details.Port,
				TimeoutInSeconds:  details.TimeoutInSeconds,
				Protocol:          details.Protocol,
				Method:            details.Method,
				Path:              details.Path,
				Headers:           map[string]string{},
				DisplayName:       details.DisplayName,
				IntervalInSeconds: details.IntervalInSeconds,
				IsEnabled:         common.Bool(true),
				FreeformTags:      details.FreeformTags,
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state healthcheckssdk.HttpMonitor) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state healthcheckssdk.HttpMonitor) (healthcheckssdk.HttpMonitor, ocimock.Response, error) {
			var details healthcheckssdk.UpdateHttpMonitorDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return healthcheckssdk.HttpMonitor{}, ocimock.Response{}, err
			}
			updateCount++
			switch updateCount {
			case 1:
				expected := healthcheckssdk.UpdateHttpMonitorDetails{IsEnabled: common.Bool(false)}
				if !reflect.DeepEqual(details, expected) {
					return healthcheckssdk.HttpMonitor{}, ocimock.Response{}, fmt.Errorf("default-correction details = %+v, want %+v", details, expected)
				}
				state.IsEnabled = details.IsEnabled
			case 2:
				expected := healthcheckssdk.UpdateHttpMonitorDetails{
					DisplayName:       common.String("mock-http-monitor-updated"),
					IntervalInSeconds: common.Int(30),
					Path:              common.String("/health"),
					FreeformTags:      map[string]string{"osok-mock": "update"},
				}
				if !reflect.DeepEqual(details, expected) {
					return healthcheckssdk.HttpMonitor{}, ocimock.Response{}, fmt.Errorf("requested update details = %+v, want %+v", details, expected)
				}
				state.DisplayName = details.DisplayName
				state.IntervalInSeconds = details.IntervalInSeconds
				state.Path = details.Path
				state.FreeformTags = details.FreeformTags
			default:
				return healthcheckssdk.HttpMonitor{}, ocimock.Response{}, fmt.Errorf("unexpected HttpMonitor update %d: %+v", updateCount, details)
			}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(request ocimock.Request, _ healthcheckssdk.HttpMonitor) (ocimock.Response, error) {
			if len(request.Body) != 0 {
				return ocimock.Response{}, fmt.Errorf("delete HttpMonitor body = %s", request.Body)
			}
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
