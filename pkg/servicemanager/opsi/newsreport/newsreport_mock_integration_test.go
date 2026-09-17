/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package newsreport

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
func TestMockIntegrationNewsReportWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[opsiv1beta1.NewsReport](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "weekly-news",
    "namespace": "default",
    "uid": "newsreport-uid"
  },
  "spec": {
    "areChildCompartmentsIncluded": false,
    "compartmentId": "compartment-1",
    "contentTypes": {
      "actionableInsightsResources": [
        "NEW_HIGHS"
      ],
      "capacityPlanningResources": [
        "DATABASE"
      ]
    },
    "dayOfWeek": "MONDAY",
    "definedTags": {
      "ns": {
        "key": "value"
      }
    },
    "description": "weekly report",
    "freeformTags": {
      "env": "dev"
    },
    "locale": "EN",
    "matchRule": "MATCH_ANY",
    "name": "weekly-news",
    "newsFrequency": "WEEKLY",
    "onsTopicId": "ons-topic-1",
    "status": "ENABLED",
    "tagFilters": [
      "department=finance"
    ]
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-newsreport")
	resource.Status = opsiv1beta1.NewsReportStatus{}
	createRequest := ocimock.MustJSONFixture[opsisdk.CreateNewsReportDetails](t, `
{
  "areChildCompartmentsIncluded": false,
  "compartmentId": "compartment-1",
  "contentTypes": {
    "actionableInsightsResources": [
      "NEW_HIGHS"
    ],
    "capacityPlanningResources": [
      "DATABASE"
    ]
  },
  "dayOfWeek": "MONDAY",
  "definedTags": {
    "ns": {
      "key": "value"
    }
  },
  "description": "weekly report",
  "freeformTags": {
    "env": "dev"
  },
  "locale": "EN",
  "matchRule": "MATCH_ANY",
  "name": "weekly-news",
  "newsFrequency": "WEEKLY",
  "onsTopicId": "ons-topic-1",
  "status": "ENABLED",
  "tagFilters": [
    "department=finance"
  ]
}
`)
	updateRequest := ocimock.MustJSONFixture[opsisdk.UpdateNewsReportDetails](t, `
{
  "description": "updated news report"
}
`)
	createdState := ocimock.MustOCIResponseFixture[opsisdk.NewsReport](t, `
{
  "areChildCompartmentsIncluded": false,
  "compartmentId": "compartment-1",
  "contentTypes": {
    "actionableInsightsResources": [
      "NEW_HIGHS"
    ],
    "capacityPlanningResources": [
      "DATABASE"
    ],
    "sqlInsightsFleetAnalysisResources": null,
    "sqlInsightsPerformanceDegradationResources": null,
    "sqlInsightsPlanChangesResources": null,
    "sqlInsightsTopDatabasesResources": null,
    "sqlInsightsTopSqlByInsightsResources": null,
    "sqlInsightsTopSqlResources": null
  },
  "dayOfWeek": "MONDAY",
  "definedTags": {
    "ns": {
      "key": "value"
    }
  },
  "description": "weekly report",
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "locale": "EN",
  "matchRule": "MATCH_ANY",
  "name": "weekly-news",
  "newsFrequency": "WEEKLY",
  "onsTopicId": "ons-topic-1",
  "status": "ENABLED",
  "systemTags": null,
  "tagFilters": [
    "department=finance"
  ],
  "timeCreated": null,
  "timeUpdated": null
}
`)
	updatedState := ocimock.MustOCIResponseFixture[opsisdk.NewsReport](t, `
{
  "areChildCompartmentsIncluded": false,
  "compartmentId": "compartment-1",
  "contentTypes": {
    "actionableInsightsResources": [
      "NEW_HIGHS"
    ],
    "capacityPlanningResources": [
      "DATABASE"
    ],
    "sqlInsightsFleetAnalysisResources": null,
    "sqlInsightsPerformanceDegradationResources": null,
    "sqlInsightsPlanChangesResources": null,
    "sqlInsightsTopDatabasesResources": null,
    "sqlInsightsTopSqlByInsightsResources": null,
    "sqlInsightsTopSqlResources": null
  },
  "dayOfWeek": "MONDAY",
  "definedTags": {
    "ns": {
      "key": "value"
    }
  },
  "description": "updated news report",
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "locale": "EN",
  "matchRule": "MATCH_ANY",
  "name": "weekly-news",
  "newsFrequency": "WEEKLY",
  "onsTopicId": "ons-topic-1",
  "status": "ENABLED",
  "systemTags": null,
  "tagFilters": [
    "department=finance"
  ],
  "timeCreated": null,
  "timeUpdated": null
}
`)
	deletedState := ocimock.MustOCIResponseFixture[opsisdk.NewsReport](t, `
{
  "areChildCompartmentsIncluded": false,
  "compartmentId": "compartment-1",
  "contentTypes": {
    "actionableInsightsResources": [
      "NEW_HIGHS"
    ],
    "capacityPlanningResources": [
      "DATABASE"
    ],
    "sqlInsightsFleetAnalysisResources": null,
    "sqlInsightsPerformanceDegradationResources": null,
    "sqlInsightsPlanChangesResources": null,
    "sqlInsightsTopDatabasesResources": null,
    "sqlInsightsTopSqlByInsightsResources": null,
    "sqlInsightsTopSqlResources": null
  },
  "dayOfWeek": "MONDAY",
  "definedTags": {
    "ns": {
      "key": "value"
    }
  },
  "description": "updated news report",
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "locale": "EN",
  "matchRule": "MATCH_ANY",
  "name": "weekly-news",
  "newsFrequency": "WEEKLY",
  "onsTopicId": "ons-topic-1",
  "status": "ENABLED",
  "systemTags": null,
  "tagFilters": [
    "department=finance"
  ],
  "timeCreated": null,
  "timeUpdated": null
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE_NEWS_REPORT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "NewsReport",
      "identifier": "<ocid:2>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE_NEWS_REPORT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "NewsReport",
      "identifier": "<ocid:2>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETE_NEWS_REPORT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "NewsReport",
      "identifier": "<ocid:2>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[opsisdk.NewsReport, opsisdk.CreateNewsReportDetails, opsisdk.UpdateNewsReportDetails]{
		CollectionPath: "/20200630/newsReports", ItemPath: "/20200630/newsReports/<ocid:2>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ opsisdk.CreateNewsReportDetails) error {
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
	client := newNewsReportServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opsiv1beta1.NewsReport]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opsiv1beta1.NewsReport) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created NewsReport status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opsiv1beta1.NewsReport) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "description": "updated news report"
}`)
		},
		ValidateUpdated: func(current *opsiv1beta1.NewsReport) error {
			if !(current.Status.Description == "updated news report") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated NewsReport status = %+v", current.Status)
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
