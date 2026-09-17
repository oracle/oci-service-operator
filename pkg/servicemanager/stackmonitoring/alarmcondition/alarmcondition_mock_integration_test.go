/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package alarmcondition

import (
	"context"
	"fmt"
	"testing"

	stackmonitoringsdk "github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationAlarmConditionCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := makeAlarmConditionResource()
	ocimock.InitializeResource(resource, "mock-alarm-condition")
	updatedSpec := resource.Spec
	updatedSpec.MetricName = "MemoryUtilization"
	createRequest := ocimock.MustJSONFixture[stackmonitoringsdk.CreateAlarmConditionDetails](t, `{
  "namespace":"oracle_oci_database","resourceType":"ocid1.stackmonitoringresourcetype.oc1..db",
  "metricName":"CpuUtilization","conditionType":"FIXED",
  "conditions":[{"severity":"CRITICAL","query":"CpuUtilization[1m].mean() > 90","body":"CPU too high","shouldAppendNote":false,"shouldAppendUrl":true,"triggerDelay":"PT5M"}],
  "freeformTags":{"env":"dev"},"definedTags":{"Operations":{"CostCenter":"42"}}
}`)
	createdState := makeSDKAlarmCondition(testAlarmConditionID, testAlarmConditionMonitoringTemplate, resource.Spec, stackmonitoringsdk.AlarmConditionLifeCycleStatesActive)
	updateRequest := ocimock.MustJSONFixture[stackmonitoringsdk.UpdateAlarmConditionDetails](t, `{"metricName":"MemoryUtilization"}`)
	updatedState := makeSDKAlarmCondition(testAlarmConditionID, testAlarmConditionMonitoringTemplate, updatedSpec, stackmonitoringsdk.AlarmConditionLifeCycleStatesActive)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[stackmonitoringsdk.AlarmCondition, stackmonitoringsdk.CreateAlarmConditionDetails, stackmonitoringsdk.UpdateAlarmConditionDetails]{
		CollectionPath: "/20210330/monitoringTemplates/" + testAlarmConditionMonitoringTemplate + "/alarmConditions",
		ItemPath:       "/20210330/monitoringTemplates/" + testAlarmConditionMonitoringTemplate + "/alarmConditions/" + testAlarmConditionID,
		Operations:     []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		ValidateCreateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://stack-monitoring.mock.invalid", BasePath: "20210330", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newAlarmConditionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, stackmonitoringsdk.StackMonitoringClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*stackmonitoringv1beta1.AlarmCondition]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *stackmonitoringv1beta1.AlarmCondition) error {
			if current.Status.Id != testAlarmConditionID || current.Status.MonitoringTemplateId != testAlarmConditionMonitoringTemplate || current.Status.MetricName != current.Spec.MetricName || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("created AlarmCondition status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *stackmonitoringv1beta1.AlarmCondition) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *stackmonitoringv1beta1.AlarmCondition) error {
			if current.Status.MetricName != current.Spec.MetricName || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("updated AlarmCondition status = %+v", current.Status)
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
