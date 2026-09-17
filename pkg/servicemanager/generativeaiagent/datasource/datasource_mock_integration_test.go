/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package datasource

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	generativeaiagentsdk "github.com/oracle/oci-go-sdk/v65/generativeaiagent"
	generativeaiagentv1beta1 "github.com/oracle/oci-service-operator/api/generativeaiagent/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationDataSourceWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &generativeaiagentv1beta1.DataSource{}
	ocimock.InitializeResource(resource, "mock-datasource")
	resource.Spec = ocimock.MustJSONFixture[generativeaiagentv1beta1.DataSourceSpec](t, `{
  "compartmentId": "<ocid:1>",
  "dataSourceConfig": {
    "dataSourceConfigType": "OCI_OBJECT_STORAGE",
    "objectStoragePrefixes": [
      {
        "bucketName": "bucket-a",
        "namespaceName": "namespace-a",
        "prefix": "documents/"
      }
    ],
    "shouldEnableMultiModality": false
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "data-source description",
  "displayName": "data-source-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "knowledgeBaseId": "<ocid:2>",
  "metadata": {
    "source": "objectstorage"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "data source description updated"
}`)

	createRequest := ocimock.MustJSONFixture[generativeaiagentsdk.CreateDataSourceDetails](t, `{
  "compartmentId": "<ocid:1>",
  "dataSourceConfig": {
    "dataSourceConfigType": "OCI_OBJECT_STORAGE",
    "objectStoragePrefixes": [
      {
        "bucketName": "bucket-a",
        "namespaceName": "namespace-a",
        "prefix": "documents/"
      }
    ],
    "shouldEnableMultiModality": false
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "data-source description",
  "displayName": "data-source-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "knowledgeBaseId": "<ocid:2>",
  "metadata": {
    "source": "objectstorage"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[generativeaiagentsdk.UpdateDataSourceDetails](t, `{
  "description": "data source description updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[generativeaiagentsdk.DataSource](t, `{
  "compartmentId": "<ocid:1>",
  "dataSourceConfig": {
    "dataSourceConfigType": "OCI_OBJECT_STORAGE",
    "objectStoragePrefixes": [
      {
        "bucketName": "bucket-a",
        "namespaceName": "namespace-a",
        "prefix": "documents/"
      }
    ],
    "shouldEnableMultiModality": false
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "data-source description",
  "displayName": "data-source-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "id": "<ocid:4>",
  "knowledgeBaseId": "<ocid:2>",
  "lifecycleDetails": "lifecycle detail",
  "lifecycleState": "ACTIVE",
  "metadata": {
    "source": "objectstorage"
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "1970-01-01T00:00:00Z",
  "timeUpdated": "1970-01-01T00:00:00Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[generativeaiagentsdk.DataSource](t, `{
  "compartmentId": "<ocid:1>",
  "dataSourceConfig": {
    "dataSourceConfigType": "OCI_OBJECT_STORAGE",
    "objectStoragePrefixes": [
      {
        "bucketName": "bucket-a",
        "namespaceName": "namespace-a",
        "prefix": "documents/"
      }
    ],
    "shouldEnableMultiModality": false
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "data source description updated",
  "displayName": "data-source-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "id": "<ocid:4>",
  "knowledgeBaseId": "<ocid:2>",
  "lifecycleDetails": "lifecycle detail",
  "lifecycleState": "ACTIVE",
  "metadata": {
    "source": "objectstorage"
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "1970-01-01T00:00:00Z",
  "timeUpdated": "1970-01-01T00:00:00Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[generativeaiagentsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:3>",
  "operationType": "CREATE_DATA_SOURCE",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "DataSource",
      "entityUri": null,
      "identifier": "<ocid:4>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "1970-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[generativeaiagentsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "wr-update",
  "operationType": "UPDATE_DATA_SOURCE",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "DataSource",
      "entityUri": null,
      "identifier": "<ocid:4>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "1970-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[generativeaiagentsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "DELETE_DATA_SOURCE",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "DataSource",
      "entityUri": null,
      "identifier": "<ocid:4>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "1970-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[generativeaiagentsdk.DataSource, generativeaiagentsdk.CreateDataSourceDetails, generativeaiagentsdk.UpdateDataSourceDetails]{
		CollectionPath:     "/20240531/dataSources",
		ItemPath:           "/20240531/dataSources/<ocid:4>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		ListShape:          ocimock.ListShapeItems,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:3>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ generativeaiagentsdk.CreateDataSourceDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20240531/workRequests/<ocid:3>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20240531/workRequests/wr-update", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20240531/workRequests/<ocid:5>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://generative-ai-agent.us-ashburn-1.oci.oraclecloud.com", BasePath: "20240531", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := generativeaiagentsdk.GenerativeAiAgentClient{BaseClient: session.BaseClient()}
	client := newDataSourceServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*generativeaiagentv1beta1.DataSource]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *generativeaiagentv1beta1.DataSource) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created DataSource status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *generativeaiagentv1beta1.DataSource) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *generativeaiagentv1beta1.DataSource) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated DataSource status = %+v", current.Status)
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
