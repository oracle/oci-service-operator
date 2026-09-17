/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package agentendpoint

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
func TestMockIntegrationAgentEndpointWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &generativeaiagentv1beta1.AgentEndpoint{}
	ocimock.InitializeResource(resource, "mock-agentendpoint")
	resource.Spec = ocimock.MustJSONFixture[generativeaiagentv1beta1.AgentEndpointSpec](t, `{
  "agentId": "<ocid:1>",
  "compartmentId": "<ocid:2>",
  "contentModerationConfig": {
    "shouldEnableOnInput": true,
    "shouldEnableOnOutput": false
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "agent endpoint description",
  "displayName": "agentendpoint-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "guardrailConfig": {
    "contentModerationConfig": {
      "inputGuardrailMode": "BLOCK",
      "outputGuardrailMode": "REDACT"
    },
    "personallyIdentifiableInformationConfig": {
      "inputGuardrailMode": "REDACT",
      "outputGuardrailMode": "REDACT"
    },
    "promptInjectionConfig": {
      "inputGuardrailMode": "BLOCK"
    }
  },
  "humanInputConfig": {
    "shouldEnableHumanInput": false
  },
  "metadata": {
    "environment": "dev"
  },
  "outputConfig": {
    "outputLocation": {
      "bucketName": "bucket",
      "namespaceName": "namespace",
      "outputLocationType": "OBJECT_STORAGE_PREFIX",
      "prefix": "results/"
    },
    "retentionPeriodInMinutes": 60
  },
  "provisionedCapacityConfig": {
    "platformRuntimeConfig": {
      "platformRuntimeConfigType": "AGENT_PLATFORM",
      "version": "2024.05.31"
    },
    "provisionedCapacityId": "<ocid:3>",
    "toolRuntimeConfigs": [
      {
        "toolRuntimeConfigType": "RAG",
        "version": "1.1"
      }
    ]
  },
  "sessionConfig": {
    "idleTimeoutInSeconds": 600
  },
  "shouldEnableCitation": true,
  "shouldEnableMultiLanguage": true,
  "shouldEnableSession": true,
  "shouldEnableTrace": false
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "agent endpoint description updated"
}`)

	createRequest := ocimock.MustJSONFixture[generativeaiagentsdk.CreateAgentEndpointDetails](t, `{
  "agentId": "<ocid:1>",
  "compartmentId": "<ocid:2>",
  "contentModerationConfig": {
    "shouldEnableOnInput": true,
    "shouldEnableOnOutput": false
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "agent endpoint description",
  "displayName": "agentendpoint-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "guardrailConfig": {
    "contentModerationConfig": {
      "inputGuardrailMode": "BLOCK",
      "outputGuardrailMode": "REDACT"
    },
    "personallyIdentifiableInformationConfig": {
      "inputGuardrailMode": "REDACT",
      "outputGuardrailMode": "REDACT"
    },
    "promptInjectionConfig": {
      "inputGuardrailMode": "BLOCK"
    }
  },
  "humanInputConfig": {
    "shouldEnableHumanInput": false
  },
  "metadata": {
    "environment": "dev"
  },
  "outputConfig": {
    "outputLocation": {
      "bucketName": "bucket",
      "namespaceName": "namespace",
      "outputLocationType": "OBJECT_STORAGE_PREFIX",
      "prefix": "results/"
    },
    "retentionPeriodInMinutes": 60
  },
  "provisionedCapacityConfig": {
    "platformRuntimeConfig": {
      "platformRuntimeConfigType": "AGENT_PLATFORM",
      "version": "2024.05.31"
    },
    "provisionedCapacityId": "<ocid:3>",
    "toolRuntimeConfigs": [
      {
        "toolRuntimeConfigType": "RAG",
        "version": "1.1"
      }
    ]
  },
  "sessionConfig": {
    "idleTimeoutInSeconds": 600
  },
  "shouldEnableCitation": true,
  "shouldEnableMultiLanguage": true,
  "shouldEnableSession": true,
  "shouldEnableTrace": false
}`)
	updateRequest := ocimock.MustJSONFixture[generativeaiagentsdk.UpdateAgentEndpointDetails](t, `{
  "description": "agent endpoint description updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[generativeaiagentsdk.AgentEndpoint](t, `{
  "agentId": "<ocid:1>",
  "compartmentId": "<ocid:2>",
  "contentModerationConfig": {
    "shouldEnableOnInput": true,
    "shouldEnableOnOutput": false
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "agent endpoint description",
  "displayName": "agentendpoint-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "guardrailConfig": {
    "contentModerationConfig": {
      "inputGuardrailMode": "BLOCK",
      "outputGuardrailMode": "REDACT"
    },
    "personallyIdentifiableInformationConfig": {
      "inputGuardrailMode": "REDACT",
      "outputGuardrailMode": "REDACT"
    },
    "promptInjectionConfig": {
      "inputGuardrailMode": "BLOCK"
    }
  },
  "humanInputConfig": {
    "shouldEnableHumanInput": false
  },
  "id": "<ocid:5>",
  "lifecycleDetails": "lifecycle detail",
  "lifecycleState": "ACTIVE",
  "metadata": {
    "environment": "dev"
  },
  "outputConfig": {
    "outputLocation": {
      "bucketName": "bucket",
      "namespaceName": "namespace",
      "outputLocationType": "OBJECT_STORAGE_PREFIX",
      "prefix": "results/"
    },
    "retentionPeriodInMinutes": 60
  },
  "provisionedCapacityConfig": {
    "platformRuntimeConfig": {
      "platformRuntimeConfigType": "AGENT_PLATFORM",
      "version": "2024.05.31"
    },
    "provisionedCapacityId": "<ocid:3>",
    "toolRuntimeConfigs": [
      {
        "toolRuntimeConfigType": "RAG",
        "version": "1.1"
      }
    ]
  },
  "sessionConfig": {
    "idleTimeoutInSeconds": 600
  },
  "shouldEnableCitation": true,
  "shouldEnableMultiLanguage": true,
  "shouldEnableSession": true,
  "shouldEnableTrace": false,
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "1970-01-01T00:00:00Z",
  "timeUpdated": "1970-01-01T00:00:00Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[generativeaiagentsdk.AgentEndpoint](t, `{
  "agentId": "<ocid:1>",
  "compartmentId": "<ocid:2>",
  "contentModerationConfig": {
    "shouldEnableOnInput": true,
    "shouldEnableOnOutput": false
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "agent endpoint description updated",
  "displayName": "agentendpoint-sample",
  "freeformTags": {
    "environment": "dev"
  },
  "guardrailConfig": {
    "contentModerationConfig": {
      "inputGuardrailMode": "BLOCK",
      "outputGuardrailMode": "REDACT"
    },
    "personallyIdentifiableInformationConfig": {
      "inputGuardrailMode": "REDACT",
      "outputGuardrailMode": "REDACT"
    },
    "promptInjectionConfig": {
      "inputGuardrailMode": "BLOCK"
    }
  },
  "humanInputConfig": {
    "shouldEnableHumanInput": false
  },
  "id": "<ocid:5>",
  "lifecycleDetails": "lifecycle detail",
  "lifecycleState": "ACTIVE",
  "metadata": {
    "environment": "dev"
  },
  "outputConfig": {
    "outputLocation": {
      "bucketName": "bucket",
      "namespaceName": "namespace",
      "outputLocationType": "OBJECT_STORAGE_PREFIX",
      "prefix": "results/"
    },
    "retentionPeriodInMinutes": 60
  },
  "provisionedCapacityConfig": {
    "platformRuntimeConfig": {
      "platformRuntimeConfigType": "AGENT_PLATFORM",
      "version": "2024.05.31"
    },
    "provisionedCapacityId": "<ocid:3>",
    "toolRuntimeConfigs": [
      {
        "toolRuntimeConfigType": "RAG",
        "version": "1.1"
      }
    ]
  },
  "sessionConfig": {
    "idleTimeoutInSeconds": 600
  },
  "shouldEnableCitation": true,
  "shouldEnableMultiLanguage": true,
  "shouldEnableSession": true,
  "shouldEnableTrace": false,
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "1970-01-01T00:00:00Z",
  "timeUpdated": "1970-01-01T00:00:00Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[generativeaiagentsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:2>",
  "id": "<ocid:4>",
  "operationType": "CREATE_AGENT_ENDPOINT",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "AgentEndpoint",
      "entityUri": null,
      "identifier": "<ocid:5>",
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
  "compartmentId": "<ocid:2>",
  "id": "wr-update",
  "operationType": "UPDATE_AGENT_ENDPOINT",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "AgentEndpoint",
      "entityUri": null,
      "identifier": "<ocid:5>",
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
  "compartmentId": "<ocid:2>",
  "id": "<ocid:6>",
  "operationType": "DELETE_AGENT_ENDPOINT",
  "percentComplete": 50,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "AgentEndpoint",
      "entityUri": null,
      "identifier": "<ocid:5>",
      "metadata": null
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "1970-01-01T00:00:00Z",
  "timeFinished": null,
  "timeStarted": null,
  "timeUpdated": null
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[generativeaiagentsdk.AgentEndpoint, generativeaiagentsdk.CreateAgentEndpointDetails, generativeaiagentsdk.UpdateAgentEndpointDetails]{
		CollectionPath:     "/20240531/agentEndpoints",
		ItemPath:           "/20240531/agentEndpoints/<ocid:5>",
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
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"wr-update"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		ValidateCreate: func(request ocimock.Request, _ generativeaiagentsdk.CreateAgentEndpointDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20240531/workRequests/<ocid:4>", MinimumCalls: 1,
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
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20240531/workRequests/<ocid:6>", MinimumCalls: 1,
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
	client := newAgentEndpointServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*generativeaiagentv1beta1.AgentEndpoint]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *generativeaiagentv1beta1.AgentEndpoint) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created AgentEndpoint status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *generativeaiagentv1beta1.AgentEndpoint) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *generativeaiagentv1beta1.AgentEndpoint) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated AgentEndpoint status = %+v", current.Status)
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
