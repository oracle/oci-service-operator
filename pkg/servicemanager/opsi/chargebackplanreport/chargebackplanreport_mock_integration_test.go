/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package chargebackplanreport

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationChargebackPlanReportWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[opsiv1beta1.ChargebackPlanReport](t, `
{
  "metadata": {
    "annotations": {
      "opsi.oracle.com/resource-id": "ocid1.databaseinsight.oc1..source",
      "opsi.oracle.com/resource-type": "DATABASE_INSIGHT"
    },
    "creationTimestamp": null,
    "name": "sample-report",
    "namespace": "default"
  },
  "spec": {
    "reportName": "monthly-chargeback",
    "reportProperties": {
      "analysisTimeInterval": "P30D",
      "groupBy": {
        "dimension": "databaseName"
      },
      "timeIntervalEnd": "2026-01-31T00:00:00Z",
      "timeIntervalStart": "2026-01-01T00:00:00Z"
    }
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-chargebackplanreport")
	resource.Status = opsiv1beta1.ChargebackPlanReportStatus{}
	createRequest := ocimock.MustJSONFixture[opsisdk.CreateChargebackPlanReportDetails](t, `
{
  "reportName": "monthly-chargeback",
  "reportProperties": {
    "analysisTimeInterval": "P30D",
    "groupBy": {
      "dimension": "databaseName"
    },
    "timeIntervalEnd": "2026-01-31T00:00:00Z",
    "timeIntervalStart": "2026-01-01T00:00:00Z"
  }
}
`)
	updateRequest := ocimock.MustJSONFixture[opsisdk.UpdateChargebackPlanReportDetails](t, `
{
  "reportName": "updated-chargeback",
  "reportProperties": {
    "analysisTimeInterval": "P30D",
    "groupBy": {
      "dimension": "databaseName"
    },
    "timeIntervalEnd": "2026-01-31T00:00:00Z",
    "timeIntervalStart": "2026-01-01T00:00:00Z"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[opsisdk.ChargebackPlanReport](t, `
{
  "lifecycleState": "ACTIVE",
  "reportId": "<ocid:3>",
  "reportName": "monthly-chargeback",
  "reportProperties": {
    "analysisTimeInterval": "P30D",
    "groupBy": {
      "dimension": "databaseName"
    },
    "timeIntervalEnd": "2026-01-31T00:00:00Z",
    "timeIntervalStart": "2026-01-01T00:00:00Z"
  },
  "resourceId": "ocid1.databaseinsight.oc1..source",
  "resourceType": "DATABASE_INSIGHT",
  "timeCreated": "2026-01-01T00:00:00Z",
  "timeUpdated": "2026-01-31T00:00:00Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[opsisdk.ChargebackPlanReport](t, `
{
  "lifecycleState": "ACTIVE",
  "reportId": "<ocid:3>",
  "reportName": "updated-chargeback",
  "reportProperties": {
    "analysisTimeInterval": "P30D",
    "groupBy": {
      "dimension": "databaseName"
    },
    "timeIntervalEnd": "2026-01-31T00:00:00Z",
    "timeIntervalStart": "2026-01-01T00:00:00Z"
  },
  "resourceId": "ocid1.databaseinsight.oc1..source",
  "resourceType": "DATABASE_INSIGHT",
  "timeCreated": "2026-01-01T00:00:00Z",
  "timeUpdated": "2026-01-31T00:00:00Z"
}
`)

	createWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "ChargebackPlanReport",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "ChargebackPlanReport",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "ChargebackPlanReport",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[opsisdk.ChargebackPlanReport, opsisdk.CreateChargebackPlanReportDetails, opsisdk.UpdateChargebackPlanReportDetails]{
		CollectionPath: "/20200630/chargebackPlanReport", ItemPath: "/20200630/chargebackPlanReport/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ opsisdk.CreateChargebackPlanReportDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://opsi.mock.invalid", BasePath: "20200630", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := opsisdk.OperationsInsightsClient{BaseClient: session.BaseClient()}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	hooks := newChargebackPlanReportDefaultRuntimeHooks(sdkClient)
	configureChargebackPlanReportRuntimeHooks(&hooks, sdkClient, nil, log)
	client := wrapChargebackPlanReportGeneratedClient(hooks, defaultChargebackPlanReportServiceClient{ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.ChargebackPlanReport](buildChargebackPlanReportGeneratedRuntimeConfig(&ChargebackPlanReportServiceManager{Log: log}, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opsiv1beta1.ChargebackPlanReport]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opsiv1beta1.ChargebackPlanReport) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.ReportId == "" ||
				current.Status.ReportName != resource.Spec.ReportName || current.Status.ResourceType != "DATABASE_INSIGHT" ||
				current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created ChargebackPlanReport status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opsiv1beta1.ChargebackPlanReport) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "reportName": "updated-chargeback"
}`)
		},
		ValidateUpdated: func(current *opsiv1beta1.ChargebackPlanReport) error {
			if !(current.Status.ReportName == "updated-chargeback") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated ChargebackPlanReport status = %+v", current.Status)
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
