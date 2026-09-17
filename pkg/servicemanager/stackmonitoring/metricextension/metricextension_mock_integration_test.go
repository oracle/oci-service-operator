/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package metricextension

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockMetricExtensionID = "ocid1.metricextension.oc1..mock"

// Contract evidence: recorded metric-extension CRUD, resource-local semantics, OCI SDK, and pinned provider.
func TestMockIntegrationMetricExtensionLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &stackmonitoringv1beta1.MetricExtension{Spec: stackmonitoringv1beta1.MetricExtensionSpec{
		Name: "ME_OSOK_MOCK_GC_TIME", DisplayName: "OSOK mock garbage collection time", ResourceType: "weblogic_j2eeserver", CompartmentId: "ocid1.compartment.oc1..mock", CollectionRecurrences: "FREQ=DAILY;INTERVAL=1",
		MetricList:      []stackmonitoringv1beta1.MetricExtensionMetricList{{Name: "ServerName", DataType: "STRING", DisplayName: "Server name", IsDimension: true}, {Name: "TotalGCExecTime", DataType: "NUMBER", DisplayName: "Total GC time", MetricCategory: "UTILIZATION", Unit: "milliseconds"}},
		QueryProperties: stackmonitoringv1beta1.MetricExtensionQueryProperties{CollectionMethod: "JMX", ManagedBeanQuery: "java.lang:Location=%name%,type=GarbageCollector,*", JmxAttributes: "CollectionTime", IdentityMetric: "name;Location"},
		Description:     "mock create",
	}}
	ocimock.InitializeResource(resource, "mock-metric-extension")
	responder, err := newMetricExtensionMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://stack-monitoring.mock.invalid", BasePath: "20210330", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close MetricExtension OCI mock: %v", err)
		}
	})
	client := newMetricExtensionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, stackmonitoringsdk.StackMonitoringClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*stackmonitoringv1beta1.MetricExtension]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *stackmonitoringv1beta1.MetricExtension) error {
			if current.Status.Id != mockMetricExtensionID || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.LifecycleState != string(stackmonitoringsdk.MetricExtensionLifeCycleStatesActive) {
				return fmt.Errorf("created MetricExtension status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *stackmonitoringv1beta1.MetricExtension) {
			current.Spec.DisplayName = "OSOK mock metric extension updated"
			current.Spec.Description = "mock update"
		},
		ValidateUpdated: func(current *stackmonitoringv1beta1.MetricExtension) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated MetricExtension status = %+v", current.Status)
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

func newMetricExtensionMockResponder(resource *stackmonitoringv1beta1.MetricExtension) (*ocimock.CRUDResponder[stackmonitoringsdk.MetricExtension], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[stackmonitoringsdk.MetricExtension]{
		CollectionPath: "/20210330/metricExtensions", ItemPath: "/20210330/metricExtensions/" + mockMetricExtensionID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete}, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, RetainStateAfterDelete: true,
		List: func(_ ocimock.Request, present bool, state stackmonitoringsdk.MetricExtension) (ocimock.Response, error) {
			if !present {
				return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []stackmonitoringsdk.MetricExtension{}})
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []stackmonitoringsdk.MetricExtension{state}})
		},
		Create: func(request ocimock.Request) (stackmonitoringsdk.MetricExtension, ocimock.Response, error) {
			if got := request.Header.Get("opc-retry-token"); got != string(resource.UID) {
				return stackmonitoringsdk.MetricExtension{}, ocimock.Response{}, fmt.Errorf("CreateMetricExtension retry token = %q, want resource UID %q", got, resource.UID)
			}
			var details stackmonitoringsdk.CreateMetricExtensionDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return stackmonitoringsdk.MetricExtension{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return stackmonitoringsdk.MetricExtension{}, ocimock.Response{}, err
			}
			if details.Name == nil || *details.Name != resource.Spec.Name || details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName || len(details.MetricList) != 2 {
				return stackmonitoringsdk.MetricExtension{}, ocimock.Response{}, fmt.Errorf("unexpected CreateMetricExtension details: %+v", details)
			}
			state := stackmonitoringsdk.MetricExtension{Id: common.String(mockMetricExtensionID), Name: details.Name, DisplayName: details.DisplayName, ResourceType: details.ResourceType, CompartmentId: details.CompartmentId,
				TenantId: common.String("ocid1.tenancy.oc1..mock"), CollectionMethod: common.String("JMX"), Status: stackmonitoringsdk.MetricExtensionLifeCycleDetailsDraft, CollectionRecurrences: details.CollectionRecurrences,
				MetricList: details.MetricList, QueryProperties: details.QueryProperties, Description: details.Description, LifecycleState: stackmonitoringsdk.MetricExtensionLifeCycleStatesActive, TimeCreated: &now, TimeUpdated: &now}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state stackmonitoringsdk.MetricExtension) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state stackmonitoringsdk.MetricExtension) (stackmonitoringsdk.MetricExtension, ocimock.Response, error) {
			var details stackmonitoringsdk.UpdateMetricExtensionDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return state, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "OSOK mock metric extension updated" || details.Description == nil || *details.Description != "mock update" {
				return state, ocimock.Response{}, fmt.Errorf("unexpected UpdateMetricExtension details: %+v", details)
			}
			state.DisplayName, state.Description, state.CollectionRecurrences, state.MetricList, state.TimeUpdated = details.DisplayName, details.Description, details.CollectionRecurrences, details.MetricList, &now
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		DeleteTransition: func(_ ocimock.Request, state stackmonitoringsdk.MetricExtension) (stackmonitoringsdk.MetricExtension, ocimock.Response, error) {
			state.LifecycleState = stackmonitoringsdk.MetricExtensionLifeCycleStatesDeleted
			return state, ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
