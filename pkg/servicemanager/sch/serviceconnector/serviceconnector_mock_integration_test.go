/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package serviceconnector

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	schsdk "github.com/oracle/oci-go-sdk/v65/sch"
	schv1beta1 "github.com/oracle/oci-service-operator/api/sch/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type throttledServiceConnectorResponder struct {
	delegate        ocimock.Responder
	updateThrottles int
}

func (r *throttledServiceConnectorResponder) Respond(request ocimock.Request) (ocimock.Response, error) {
	if request.Method == http.MethodPut && request.URL.Path == "/20200909/serviceConnectors/<ocid:4>" && r.updateThrottles > 0 {
		r.updateThrottles--
		return ocimock.JSONResponse(http.StatusTooManyRequests, map[string]string{
			"code":    "TooManyRequests",
			"message": "recorded Service Connector update throttle",
		})
	}
	return r.delegate.Respond(request)
}

func (r *throttledServiceConnectorResponder) Verify() error {
	if r.updateThrottles != 0 {
		return fmt.Errorf("Service Connector update throttles remaining = %d", r.updateThrottles)
	}
	return r.delegate.Verify()
}

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationServiceConnectorWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &schv1beta1.ServiceConnector{}
	ocimock.InitializeResource(resource, "mock-serviceconnector")
	resource.Spec = ocimock.MustJSONFixture[schv1beta1.ServiceConnectorSpec](t, `
{
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-service-connector-recorded-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "source": {
    "kind": "logging",
    "logSources": [
      {
        "compartmentId": "<ocid:1>",
        "logGroupId": "_Audit"
      }
    ]
  },
  "target": {
    "enableFormattedMessaging": false,
    "kind": "notifications",
    "topicId": "<ocid:2>"
  },
  "tasks": [
    {
      "condition": "logContent = 'OSOK_REPLAY_NEVER_MATCH'",
      "kind": "logRule"
    }
  ]
}
`)
	createRequest := ocimock.MustJSONFixture[schsdk.CreateServiceConnectorDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-service-connector-recorded-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "source": {
    "kind": "logging",
    "logSources": [
      {
        "compartmentId": "<ocid:1>",
        "logGroupId": "_Audit"
      }
    ]
  },
  "target": {
    "enableFormattedMessaging": false,
    "kind": "notifications",
    "topicId": "<ocid:2>"
  },
  "tasks": [
    {
      "condition": "logContent = 'OSOK_REPLAY_NEVER_MATCH'",
      "kind": "logRule"
    }
  ]
}
`)
	updateRequest := ocimock.MustJSONFixture[schsdk.UpdateServiceConnectorDetails](t, `
{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  },
  "tasks": []
}
`)
	createdState := ocimock.MustOCIResponseFixture[schsdk.ServiceConnector](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T02:34:26.680Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-service-connector-recorded-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:4>",
  "lifecycleDetails": "",
  "lifecycleState": "ACTIVE",
  "lifecyleDetails": "",
  "source": {
    "kind": "logging",
    "logSources": [
      {
        "compartmentId": "<ocid:1>",
        "logGroupId": "_Audit",
        "logId": null
      }
    ],
    "privateEndpointMetadata": null
  },
  "systemTags": {},
  "target": {
    "enableFormattedMessaging": false,
    "kind": "notifications",
    "privateEndpointMetadata": null,
    "topicId": "<ocid:2>"
  },
  "tasks": [
    {
      "condition": "logContent = 'OSOK_REPLAY_NEVER_MATCH'",
      "kind": "logRule",
      "privateEndpointMetadata": null
    }
  ],
  "timeCreated": "2026-09-04T02:34:26.873Z",
  "timeUpdated": "2026-09-04T02:34:37.149Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[schsdk.ServiceConnector](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T02:34:26.680Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-service-connector-recorded-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:4>",
  "lifecycleDetails": "",
  "lifecycleState": "ACTIVE",
  "lifecyleDetails": "",
  "source": {
    "kind": "logging",
    "logSources": [
      {
        "compartmentId": "<ocid:1>",
        "logGroupId": "_Audit",
        "logId": null
      }
    ],
    "privateEndpointMetadata": null
  },
  "systemTags": {},
  "target": {
    "enableFormattedMessaging": false,
    "kind": "notifications",
    "privateEndpointMetadata": null,
    "topicId": "<ocid:2>"
  },
  "tasks": [],
  "timeCreated": "2026-09-04T02:34:26.873Z",
  "timeUpdated": "2026-09-04T02:35:40.806Z"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[schsdk.ServiceConnector](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-04T02:34:26.680Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-service-connector-recorded-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:4>",
  "lifecycleDetails": "",
  "lifecycleState": "DELETED",
  "lifecyleDetails": "",
  "source": {
    "kind": "logging",
    "logSources": [
      {
        "compartmentId": "<ocid:1>",
        "logGroupId": "_Audit",
        "logId": null
      }
    ],
    "privateEndpointMetadata": null
  },
  "systemTags": {},
  "target": {
    "enableFormattedMessaging": false,
    "kind": "notifications",
    "privateEndpointMetadata": null,
    "topicId": "<ocid:2>"
  },
  "tasks": [],
  "timeCreated": "2026-09-04T02:34:26.873Z",
  "timeUpdated": "2026-09-04T02:35:41.969Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[schsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:3>",
  "operationType": "CREATE_SERVICE_CONNECTOR",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "serviceConnector",
      "entityUri": "/serviceConnectors/<ocid:4>",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-04T02:34:26.881Z",
  "timeFinished": "2026-09-04T02:34:37.144Z",
  "timeStarted": "2026-09-04T02:34:36.989Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[schsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "UPDATE_SERVICE_CONNECTOR",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "serviceConnector",
      "entityUri": "/serviceConnectors/<ocid:4>",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-04T02:35:40.785Z",
  "timeFinished": "2026-09-04T02:35:40.806Z",
  "timeStarted": "2026-09-04T02:35:40.806Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[schsdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:6>",
  "operationType": "DELETE_SERVICE_CONNECTOR",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "serviceConnector",
      "entityUri": "/serviceConnectors/<ocid:4>",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-04T02:35:41.954Z",
  "timeFinished": "2026-09-04T02:36:13.869Z",
  "timeStarted": "2026-09-04T02:36:13.665Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[schsdk.ServiceConnector, schsdk.CreateServiceConnectorDetails, schsdk.UpdateServiceConnectorDetails]{
		CollectionPath: "/20200909/serviceConnectors", ItemPath: "/20200909/serviceConnectors/<ocid:4>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		DeletedState: &deletedState,
		ListShape:    ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:3>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		ValidateCreate: func(request ocimock.Request, _ schsdk.CreateServiceConnectorDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200909/workRequests/<ocid:3>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200909/workRequests/<ocid:5>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200909/workRequests/<ocid:6>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	throttledResponder := &throttledServiceConnectorResponder{delegate: responder, updateThrottles: 1}
	session, err := ocimock.Open(ocimock.Options{Host: "https://service-connector-hub.us-ashburn-1.oci.oraclecloud.com", BasePath: "20200909", Responder: throttledResponder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := schsdk.ServiceConnectorClient{BaseClient: session.BaseClient()}
	client := newServiceConnectorServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	mockValidateCreated := func(current *schv1beta1.ServiceConnector) error {
		if current.Status.Id == "" || current.Status.LifecycleState != string(schsdk.LifecycleStateActive) {
			return fmt.Errorf("created ServiceConnector status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *schv1beta1.ServiceConnector) error {
		if current.Status.Description != "recorded update" || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated ServiceConnector status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*schv1beta1.ServiceConnector]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		RetryError: func(err error) bool {
			var throttled errorutil.TooManyRequestsOciError
			return errors.As(err, &throttled) && throttled.HTTPStatusCode == http.StatusTooManyRequests
		},
		ValidateCreated: func(current *schv1beta1.ServiceConnector) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *schv1beta1.ServiceConnector) {
			current.Spec.Description = "recorded update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *schv1beta1.ServiceConnector) error {
			if err := mockValidateUpdated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
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
