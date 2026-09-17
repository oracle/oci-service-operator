/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package assessment

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	databasemigrationsdk "github.com/oracle/oci-go-sdk/v65/databasemigration"
	databasemigrationv1beta1 "github.com/oracle/oci-service-operator/api/databasemigration/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: explicit MYSQL subtype selected from the CR discriminator,
// vendored SDK, production runtime, formal lifecycle, and sanitized evidence.
func TestMockIntegrationAssessmentWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &databasemigrationv1beta1.Assessment{}
	ocimock.InitializeResource(resource, "mock-assessment")
	resource.Spec = ocimock.MustJSONFixture[databasemigrationv1beta1.AssessmentSpec](t, `{
  "acceptableDowntime": "LESS_THAN_1_HOUR",
  "compartmentId": "<ocid:1>",
  "creationType": "CREATE_ONLY",
  "databaseCombination": "MYSQL",
  "databaseDataSize": "GB_10_50",
  "ddlExpectation": "DDL_NOT_EXPECTED",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "mysql assessment",
  "displayName": "mysql-assessment",
  "freeformTags": {
    "env": "test"
  },
  "networkSpeedMegabitPerSecond": "MBPS_100",
  "sourceDatabaseConnection": {
    "id": "<ocid:2>"
  },
  "targetDatabaseConnection": {
    "id": "<ocid:3>"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "assessment updated"
}`)

	createRequest := ocimock.MustJSONFixture[databasemigrationsdk.CreateMySqlAssessmentDetails](t, `{
  "acceptableDowntime": "LESS_THAN_1_HOUR",
  "compartmentId": "<ocid:1>",
  "creationType": "CREATE_ONLY",
  "databaseDataSize": "GB_10_50",
  "ddlExpectation": "DDL_NOT_EXPECTED",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "mysql assessment",
  "displayName": "mysql-assessment",
  "freeformTags": {
    "env": "test"
  },
  "networkSpeedMegabitPerSecond": "MBPS_100",
  "sourceDatabaseConnection": {
    "id": "<ocid:2>"
  },
  "targetDatabaseConnection": {
    "id": "<ocid:3>"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[databasemigrationsdk.UpdateMySqlAssessmentDetails](t, `{
	"description": "assessment updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[databasemigrationsdk.MySqlAssessment](t, `{
  "acceptableDowntime": "LESS_THAN_1_HOUR",
  "compartmentId": "<ocid:1>",
  "creationType": "CREATE_ONLY",
  "databaseCombination": "MYSQL",
  "databaseDataSize": "GB_10_50",
  "ddlExpectation": "DDL_NOT_EXPECTED",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "mysql assessment",
  "displayName": "mysql-assessment",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:5>",
  "lifecycleState": "ACTIVE",
  "networkSpeedMegabitPerSecond": "MBPS_100",
  "sourceDatabaseConnection": {
    "id": "<ocid:2>"
  },
  "targetDatabaseConnection": {
    "id": "<ocid:3>"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[databasemigrationsdk.MySqlAssessment](t, `{
  "acceptableDowntime": "LESS_THAN_1_HOUR",
  "compartmentId": "<ocid:1>",
  "creationType": "CREATE_ONLY",
  "databaseCombination": "MYSQL",
  "databaseDataSize": "GB_10_50",
  "ddlExpectation": "DDL_NOT_EXPECTED",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "assessment updated",
  "displayName": "mysql-assessment",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:5>",
  "lifecycleState": "ACTIVE",
  "networkSpeedMegabitPerSecond": "MBPS_100",
  "sourceDatabaseConnection": {
    "id": "<ocid:2>"
  },
  "targetDatabaseConnection": {
    "id": "<ocid:3>"
  }
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[databasemigrationsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:6>",
  "id": "<ocid:4>",
  "operationType": "CREATE_ASSESSMENT",
  "percentComplete": 25,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "Assessment",
      "entityUri": null,
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2024-04-16T04:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[databasemigrationsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:6>",
  "id": "wr-update",
  "operationType": "UPDATE_ASSESSMENT",
  "percentComplete": 25,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "Assessment",
      "entityUri": null,
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2024-04-16T04:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[databasemigrationsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:6>",
  "id": "<ocid:7>",
  "operationType": "DELETE_ASSESSMENT",
  "percentComplete": 25,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "Assessment",
      "entityUri": null,
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2024-04-16T04:00:00Z",
  "timeFinished": null,
  "timeStarted": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[databasemigrationsdk.MySqlAssessment, databasemigrationsdk.CreateMySqlAssessmentDetails, databasemigrationsdk.UpdateMySqlAssessmentDetails]{
		CollectionPath: "/20230518/assessments", ItemPath: "/20230518/assessments/<ocid:5>",
		Operations:   []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState: &createdState, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: http.StatusAccepted, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:7>"}},
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "databaseCombination", "MYSQL", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "databaseCombination", "MYSQL", updateRequest)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20230518/workRequests/<ocid:4>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20230518/workRequests/wr-update", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20230518/workRequests/<ocid:7>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://database-migration.us-ashburn-1.oci.oraclecloud.com", BasePath: "20230518", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	client := newAssessmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, databasemigrationsdk.DatabaseMigrationClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*databasemigrationv1beta1.Assessment]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *databasemigrationv1beta1.Assessment) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Assessment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *databasemigrationv1beta1.Assessment) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *databasemigrationv1beta1.Assessment) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Assessment status = %+v", current.Status)
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
