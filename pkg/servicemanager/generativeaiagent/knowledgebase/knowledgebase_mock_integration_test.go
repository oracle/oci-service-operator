/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package knowledgebase

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
func TestMockIntegrationKnowledgeBaseWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &generativeaiagentv1beta1.KnowledgeBase{}
	ocimock.InitializeResource(resource, "mock-knowledgebase")
	resource.Spec = ocimock.MustJSONFixture[generativeaiagentv1beta1.KnowledgeBaseSpec](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "knowledge-base description",
  "displayName": "knowledge-base-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "indexConfig": {
    "indexConfigType": "DEFAULT_INDEX_CONFIG",
    "shouldEnableHybridSearch": false
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "knowledge base description updated"
}`)

	createRequest := ocimock.MustJSONFixture[generativeaiagentsdk.CreateKnowledgeBaseDetails](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "knowledge-base description",
  "displayName": "knowledge-base-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "indexConfig": {
    "indexConfigType": "DEFAULT_INDEX_CONFIG",
    "shouldEnableHybridSearch": false
  }
}`)
	updateRequest := ocimock.MustJSONFixture[generativeaiagentsdk.UpdateKnowledgeBaseDetails](t, `{
  "description": "knowledge base description updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[generativeaiagentsdk.KnowledgeBase](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "knowledge-base description",
  "displayName": "knowledge-base-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "id": "<ocid:3>",
  "indexConfig": {
    "indexConfigType": "DEFAULT_INDEX_CONFIG",
    "shouldEnableHybridSearch": false
  },
  "knowledgeBaseStatistics": {
    "sizeInBytes": 128,
    "totalIngestedFiles": 4
  },
  "lifecycleDetails": "lifecycle detail",
  "lifecycleState": "ACTIVE",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "1970-01-01T00:00:00Z",
  "timeUpdated": "1970-01-01T00:00:00Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[generativeaiagentsdk.KnowledgeBase](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "knowledge base description updated",
  "displayName": "knowledge-base-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "id": "<ocid:3>",
  "indexConfig": {
    "indexConfigType": "DEFAULT_INDEX_CONFIG",
    "shouldEnableHybridSearch": false
  },
  "knowledgeBaseStatistics": {
    "sizeInBytes": 128,
    "totalIngestedFiles": 4
  },
  "lifecycleDetails": "lifecycle detail",
  "lifecycleState": "ACTIVE",
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
  "id": "<ocid:2>",
  "operationType": "CREATE_KNOWLEDGE_BASE",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "KnowledgeBase",
      "entityUri": null,
      "identifier": "<ocid:3>",
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
  "operationType": "UPDATE_KNOWLEDGE_BASE",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "KnowledgeBase",
      "entityUri": null,
      "identifier": "<ocid:3>",
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
  "id": "<ocid:4>",
  "operationType": "DELETE_KNOWLEDGE_BASE",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "KnowledgeBase",
      "entityUri": null,
      "identifier": "<ocid:3>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "1970-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[generativeaiagentsdk.KnowledgeBase, generativeaiagentsdk.CreateKnowledgeBaseDetails, generativeaiagentsdk.UpdateKnowledgeBaseDetails]{
		CollectionPath:     "/20240531/knowledgeBases",
		ItemPath:           "/20240531/knowledgeBases/<ocid:3>",
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
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		ValidateCreate: func(request ocimock.Request, _ generativeaiagentsdk.CreateKnowledgeBaseDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20240531/workRequests/<ocid:2>", MinimumCalls: 1,
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
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20240531/workRequests/<ocid:4>", MinimumCalls: 1,
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
	client := newKnowledgeBaseServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*generativeaiagentv1beta1.KnowledgeBase]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *generativeaiagentv1beta1.KnowledgeBase) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created KnowledgeBase status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *generativeaiagentv1beta1.KnowledgeBase) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *generativeaiagentv1beta1.KnowledgeBase) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated KnowledgeBase status = %+v", current.Status)
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
