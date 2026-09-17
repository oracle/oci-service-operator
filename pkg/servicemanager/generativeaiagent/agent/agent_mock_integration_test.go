/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package agent

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
func TestMockIntegrationAgentWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &generativeaiagentv1beta1.Agent{}
	ocimock.InitializeResource(resource, "mock-agent")
	resource.Spec = ocimock.MustJSONFixture[generativeaiagentv1beta1.AgentSpec](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "agent description",
  "displayName": "agent-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "knowledgeBaseIds": [
    "<ocid:2>"
  ],
  "llmConfig": {
    "routingLlmCustomization": {
      "instruction": "Answer from the attached knowledge base first.",
      "llmHyperParameters": {
        "temperature": 0.25,
        "useKnowledgeBase": false
      },
      "llmSelection": {
        "llmSelectionType": "DEFAULT"
      }
    },
    "runtimeVersion": "2024.05.31"
  },
  "welcomeMessage": "Hello from the published agent runtime."
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "agent description updated"
}`)

	createRequest := ocimock.MustJSONFixture[generativeaiagentsdk.CreateAgentDetails](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "agent description",
  "displayName": "agent-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "knowledgeBaseIds": [
    "<ocid:2>"
  ],
  "llmConfig": {
    "routingLlmCustomization": {
      "instruction": "Answer from the attached knowledge base first.",
      "llmHyperParameters": {
        "temperature": 0.25,
        "useKnowledgeBase": false
      },
      "llmSelection": {
        "llmSelectionType": "DEFAULT"
      }
    },
    "runtimeVersion": "2024.05.31"
  },
  "welcomeMessage": "Hello from the published agent runtime."
}`)
	updateRequest := ocimock.MustJSONFixture[generativeaiagentsdk.UpdateAgentDetails](t, `{
  "description": "agent description updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[generativeaiagentsdk.Agent](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "agent description",
  "displayName": "agent-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "id": "<ocid:4>",
  "knowledgeBaseIds": [
    "<ocid:2>"
  ],
  "lifecycleDetails": "lifecycle detail",
  "lifecycleState": "ACTIVE",
  "llmConfig": {
    "routingLlmCustomization": {
      "instruction": "Answer from the attached knowledge base first.",
      "llmHyperParameters": {
        "temperature": 0.25,
        "useKnowledgeBase": false
      },
      "llmSelection": {
        "llmSelectionType": "DEFAULT"
      }
    },
    "runtimeVersion": "2024.05.31"
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "1970-01-01T00:00:00Z",
  "timeUpdated": "1970-01-01T00:00:00Z",
  "welcomeMessage": "Hello from the published agent runtime."
}`)
	updatedState := ocimock.MustOCIResponseFixture[generativeaiagentsdk.Agent](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "agent description updated",
  "displayName": "agent-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "id": "<ocid:4>",
  "knowledgeBaseIds": [
    "<ocid:2>"
  ],
  "lifecycleDetails": "lifecycle detail",
  "lifecycleState": "ACTIVE",
  "llmConfig": {
    "routingLlmCustomization": {
      "instruction": "Answer from the attached knowledge base first.",
      "llmHyperParameters": {
        "temperature": 0.25,
        "useKnowledgeBase": false
      },
      "llmSelection": {
        "llmSelectionType": "DEFAULT"
      }
    },
    "runtimeVersion": "2024.05.31"
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "1970-01-01T00:00:00Z",
  "timeUpdated": "1970-01-01T00:00:00Z",
  "welcomeMessage": "Hello from the published agent runtime."
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[generativeaiagentsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:3>",
  "operationType": "CREATE_AGENT",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "Agent",
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
  "operationType": "UPDATE_AGENT",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "Agent",
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
  "operationType": "DELETE_AGENT",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "Agent",
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

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[generativeaiagentsdk.Agent, generativeaiagentsdk.CreateAgentDetails, generativeaiagentsdk.UpdateAgentDetails]{
		CollectionPath:     "/20240531/agents",
		ItemPath:           "/20240531/agents/<ocid:4>",
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
		ValidateCreate: func(request ocimock.Request, _ generativeaiagentsdk.CreateAgentDetails) error {
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
	client := newAgentServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*generativeaiagentv1beta1.Agent]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *generativeaiagentv1beta1.Agent) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Agent status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *generativeaiagentv1beta1.Agent) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *generativeaiagentv1beta1.Agent) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Agent status = %+v", current.Status)
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
