/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loggroup

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
func TestMockIntegrationLogGroupWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &loggingv1beta1.LogGroup{}
	ocimock.InitializeResource(resource, "mock-loggroup")
	resource.Spec = ocimock.MustJSONFixture[loggingv1beta1.LogGroupSpec](t, `{
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-log-group-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)

	createRequest := ocimock.MustJSONFixture[loggingsdk.CreateLogGroupDetails](t, `{
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-log-group-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[loggingsdk.UpdateLogGroupDetails](t, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[loggingsdk.LogGroup](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T01:22:56.329Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-log-group-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "systemTags": {
  },
  "timeCreated": "2026-09-01T01:22:56.373Z",
  "timeLastModified": "2026-09-01T01:22:56.373Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[loggingsdk.LogGroup](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T01:22:56.329Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-log-group-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "systemTags": {
  },
  "timeCreated": "2026-09-01T01:22:56.373Z",
  "timeLastModified": "2026-09-01T01:22:57.105Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[loggingsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "CREATE_LOG_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "loggroup",
      "entityUri": "/logGroups/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T01:22:56.379Z",
  "timeFinished": "2026-09-01T01:22:56.379Z",
  "timeStarted": "2026-09-01T01:22:56.379Z"
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[loggingsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_LOG_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "loggroup",
      "entityUri": "/logGroups/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T01:22:57.106Z",
  "timeFinished": "2026-09-01T01:22:57.106Z",
  "timeStarted": "2026-09-01T01:22:57.106Z"
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[loggingsdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "DELETE_LOG_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "loggroup",
      "entityUri": "/logGroups/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T01:22:57.757Z",
  "timeFinished": "2026-09-01T01:22:57.757Z",
  "timeStarted": "2026-09-01T01:22:57.757Z"
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[loggingsdk.LogGroup, loggingsdk.CreateLogGroupDetails, loggingsdk.UpdateLogGroupDetails]{
		CollectionPath:     "/20200531/logGroups",
		ItemPath:           "/20200531/logGroups/<ocid:3>",
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
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}}, UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}}, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ loggingsdk.CreateLogGroupDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20200531/workRequests/<ocid:2>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "update-work-request", Method: http.MethodGet, Path: "/20200531/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20200531/workRequests/<ocid:5>", MinimumCalls: 1,
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
	manager := &LogGroupServiceManager{Log: log}
	hooks := newLogGroupDefaultRuntimeHooks(sdkClient)
	applyLogGroupRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapLogGroupGeneratedClient(hooks, defaultLogGroupServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loggingv1beta1.LogGroup](buildLogGroupGeneratedRuntimeConfig(manager, hooks)),
	})

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loggingv1beta1.LogGroup]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loggingv1beta1.LogGroup) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created LogGroup status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loggingv1beta1.LogGroup) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loggingv1beta1.LogGroup) error {
			if current.Status.Description != current.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated LogGroup status = %+v", current.Status)
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
