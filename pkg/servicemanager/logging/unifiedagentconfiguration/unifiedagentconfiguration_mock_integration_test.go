/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package unifiedagentconfiguration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	loggingsdk "github.com/oracle/oci-go-sdk/v65/logging"
	loggingv1beta1 "github.com/oracle/oci-service-operator/api/logging/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationUnifiedAgentConfigurationWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &loggingv1beta1.UnifiedAgentConfiguration{}
	ocimock.InitializeResource(resource, "mock-unifiedagentconfiguration")
	resource.Spec = ocimock.MustJSONFixture[loggingv1beta1.UnifiedAgentConfigurationSpec](t, `{
  "compartmentId": "<ocid:1>",
  "description": "OSOK recorded unified agent configuration",
  "displayName": "osok-mock-unified-agent-config",
  "groupAssociation": {
    "groupList": [
      "<ocid:2>"
    ]
  },
  "isEnabled": true,
  "serviceConfiguration": {
    "configurationType": "LOGGING",
    "destination": {
      "logObjectId": "<ocid:3>"
    },
    "filter": [

    ],
    "sources": [
      {
        "name": "osok-mock-application",
        "parser": {
          "parserType": "NONE"
        },
        "paths": [
          "/var/log/osok-mock.log"
        ],
        "sourceType": "LOG_TAIL"
      }
    ]
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T03:41:14.890Z"
    }
  },
  "description": "OSOK recorded unified agent configuration",
  "displayName": "osok-mock-unified-agent-config-updated",
  "freeformTags": {
  },
  "groupAssociation": {
    "groupList": [
      "<ocid:2>"
    ]
  },
  "isEnabled": true,
  "serviceConfiguration": {
    "configurationType": "LOGGING",
    "destination": {
      "logObjectId": "<ocid:3>"
    },
    "filter": [

    ],
    "sources": [
      {
        "name": "osok-mock-application",
        "parser": {
          "parserType": "NONE"
        },
        "paths": [
          "/var/log/osok-mock.log"
        ],
        "sourceType": "LOG_TAIL"
      }
    ]
  }
}`)

	createRequest := ocimock.MustJSONFixture[loggingsdk.CreateUnifiedAgentConfigurationDetails](t, `{
  "compartmentId": "<ocid:1>",
  "description": "OSOK recorded unified agent configuration",
  "displayName": "osok-mock-unified-agent-config",
  "groupAssociation": {
    "groupList": [
      "<ocid:2>"
    ]
  },
  "isEnabled": true,
  "serviceConfiguration": {
    "configurationType": "LOGGING",
    "destination": {
      "logObjectId": "<ocid:3>"
    },
    "filter": [

    ],
    "sources": [
      {
        "name": "osok-mock-application",
        "parser": {
          "parserType": "NONE"
        },
        "paths": [
          "/var/log/osok-mock.log"
        ],
        "sourceType": "LOG_TAIL"
      }
    ]
  }
}`)
	updateRequest := ocimock.MustJSONFixture[loggingsdk.UpdateUnifiedAgentConfigurationDetails](t, `{
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T03:41:14.890Z"
    }
  },
  "description": "OSOK recorded unified agent configuration",
  "displayName": "osok-mock-unified-agent-config-updated",
  "freeformTags": {
  },
  "groupAssociation": {
    "groupList": [
      "<ocid:2>"
    ]
  },
  "isEnabled": true,
  "serviceConfiguration": {
    "configurationType": "LOGGING",
    "destination": {
      "logObjectId": "<ocid:3>"
    },
    "filter": [

    ],
    "sources": [
      {
        "name": "osok-mock-application",
        "parser": {
          "parserType": "NONE"
        },
        "paths": [
          "/var/log/osok-mock.log"
        ],
        "sourceType": "LOG_TAIL"
      }
    ]
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[loggingsdk.UnifiedAgentConfiguration](t, `{
  "compartmentId": "<ocid:1>",
  "configurationState": "VALID",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T03:41:14.890Z"
    }
  },
  "description": "OSOK recorded unified agent configuration",
  "displayName": "osok-mock-unified-agent-config",
  "freeformTags": {
  },
  "groupAssociation": {
    "groupList": [
      "<ocid:2>"
    ]
  },
  "id": "<ocid:5>",
  "isEnabled": true,
  "lifecycleState": "ACTIVE",
  "serviceConfiguration": {
    "configurationType": "LOGGING",
    "destination": {
      "isEnrichmentEnabled": null,
      "isMetadataHidden": null,
      "logObjectId": "<ocid:3>",
      "operationalMetricsConfiguration": null
    },
    "filter": [

    ],
    "isSourceDecoration": null,
    "sources": [
      {
        "advancedOptions": null,
        "name": "osok-mock-application",
        "parser": {
          "fieldTimeKey": null,
          "isEstimateCurrentEvent": null,
          "isKeepTimeKey": null,
          "isNullEmptyString": null,
          "messageKey": null,
          "nullValuePattern": null,
          "parserType": "NONE",
          "timeoutInMilliseconds": null,
          "types": null
        },
        "paths": [
          "/var/log/osok-mock.log"
        ],
        "sourceType": "LOG_TAIL"
      }
    ]
  },
  "systemTags": {
  },
  "timeCreated": "2026-09-02T03:41:14.959Z",
  "timeLastModified": "2026-09-02T03:41:14.959Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[loggingsdk.UnifiedAgentConfiguration](t, `{
  "compartmentId": "<ocid:1>",
  "configurationState": "VALID",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T03:41:14.890Z"
    }
  },
  "description": "OSOK recorded unified agent configuration",
  "displayName": "osok-mock-unified-agent-config-updated",
  "freeformTags": {
  },
  "groupAssociation": {
    "groupList": [
      "<ocid:2>"
    ]
  },
  "id": "<ocid:5>",
  "isEnabled": true,
  "lifecycleState": "ACTIVE",
  "serviceConfiguration": {
    "configurationType": "LOGGING",
    "destination": {
      "isEnrichmentEnabled": null,
      "isMetadataHidden": null,
      "logObjectId": "<ocid:3>",
      "operationalMetricsConfiguration": null
    },
    "filter": [

    ],
    "isSourceDecoration": null,
    "sources": [
      {
        "advancedOptions": null,
        "name": "osok-mock-application",
        "parser": {
          "fieldTimeKey": null,
          "isEstimateCurrentEvent": null,
          "isKeepTimeKey": null,
          "isNullEmptyString": null,
          "messageKey": null,
          "nullValuePattern": null,
          "parserType": "NONE",
          "timeoutInMilliseconds": null,
          "types": null
        },
        "paths": [
          "/var/log/osok-mock.log"
        ],
        "sourceType": "LOG_TAIL"
      }
    ]
  },
  "systemTags": {
  },
  "timeCreated": "2026-09-02T03:41:14.959Z",
  "timeLastModified": "2026-09-02T03:41:21.067Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[loggingsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "CREATE_CONFIGURATION",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "unifiedagentconfiguration",
      "entityUri": "/unifiedAgentConfigurations/<ocid:5>",
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T03:41:14.987Z",
  "timeFinished": "2026-09-02T03:41:17.823Z",
  "timeStarted": "2026-09-02T03:41:14.987Z"
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[loggingsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:6>",
  "operationType": "UPDATE_CONFIGURATION",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "unifiedagentconfiguration",
      "entityUri": "/unifiedAgentConfigurations/<ocid:5>",
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T03:41:21.067Z",
  "timeFinished": "2026-09-02T03:41:24.432Z",
  "timeStarted": "2026-09-02T03:41:21.067Z"
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[loggingsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:7>",
  "operationType": "DELETE_CONFIGURATION",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "unifiedagentconfiguration",
      "entityUri": "/unifiedAgentConfigurations/<ocid:5>",
      "identifier": "<ocid:5>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T03:41:27.061Z",
  "timeFinished": "2026-09-02T03:41:30.468Z",
  "timeStarted": "2026-09-02T03:41:27.061Z"
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[loggingsdk.UnifiedAgentConfiguration, loggingsdk.CreateUnifiedAgentConfigurationDetails, loggingsdk.UpdateUnifiedAgentConfigurationDetails]{
		CollectionPath:     "/20200531/unifiedAgentConfigurations",
		ItemPath:           "/20200531/unifiedAgentConfigurations/<ocid:5>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		ListShape:          ocimock.ListShapeItems,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		DeleteEndsNotFound: true, RequireDeleteRead: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:7>"}},
		ValidateCreate: func(request ocimock.Request, _ loggingsdk.CreateUnifiedAgentConfigurationDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20200531/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20200531/workRequests/<ocid:6>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20200531/workRequests/<ocid:7>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://logging.us-ashburn-1.oci.oraclecloud.com", BasePath: "20200531", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := loggingsdk.LoggingManagementClient{BaseClient: session.BaseClient()}
	manager := &UnifiedAgentConfigurationServiceManager{Log: log}
	hooks := newUnifiedAgentConfigurationRuntimeHooksWithOCIClient(sdkClient)
	applyUnifiedAgentConfigurationRuntimeHooks(manager, &hooks, sdkClient, nil)
	client := defaultUnifiedAgentConfigurationServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loggingv1beta1.UnifiedAgentConfiguration](buildUnifiedAgentConfigurationGeneratedRuntimeConfig(manager, hooks)),
	}

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loggingv1beta1.UnifiedAgentConfiguration]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loggingv1beta1.UnifiedAgentConfiguration) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created UnifiedAgentConfiguration status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loggingv1beta1.UnifiedAgentConfiguration) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loggingv1beta1.UnifiedAgentConfiguration) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated UnifiedAgentConfiguration status = %+v", current.Status)
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
