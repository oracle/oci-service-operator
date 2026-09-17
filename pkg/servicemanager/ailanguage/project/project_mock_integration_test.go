/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	ailanguagesdk "github.com/oracle/oci-go-sdk/v65/ailanguage"
	ailanguagev1beta1 "github.com/oracle/oci-service-operator/api/ailanguage/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: the vendored OCI SDK, the package service manager, the
// reviewed formal lifecycle, and the existing sanitized OCI mock fixture.
func TestMockIntegrationProjectWorkRequestCRUD(t *testing.T) {
	t.Parallel()

	resource := &ailanguagev1beta1.Project{}
	ocimock.InitializeResource(resource, "mock-project")
	resource.Spec = ocimock.MustJSONFixture[ailanguagev1beta1.ProjectSpec](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-ai-language-project-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "osok-mock-ai-language-project-v1-updated"
}`)

	createRequest := ocimock.MustJSONFixture[ailanguagesdk.CreateProjectDetails](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "osok-mock-ai-language-project-v1",
  "freeformTags": {
    "osok-mock": "create"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[ailanguagesdk.UpdateProjectDetails](t, `{
  "displayName": "osok-mock-ai-language-project-v1-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[ailanguagesdk.Project](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T19:35:15.374Z"
    }
  },
  "description": null,
  "displayName": "osok-mock-ai-language-project-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {
  },
  "timeCreated": "2026-09-01T19:35:15.578Z",
  "timeUpdated": "2026-09-01T19:35:56.016Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[ailanguagesdk.Project](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T19:35:15.374Z"
    }
  },
  "description": null,
  "displayName": "osok-mock-ai-language-project-v1-updated",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "systemTags": {
  },
  "timeCreated": "2026-09-01T19:35:15.578Z",
  "timeUpdated": "2026-09-01T19:36:46.829Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[ailanguagesdk.Project](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T19:35:15.374Z"
    }
  },
  "description": null,
  "displayName": "osok-mock-ai-language-project-v1-updated",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "systemTags": {
  },
  "timeCreated": "2026-09-01T19:35:15.578Z",
  "timeUpdated": "2026-09-01T19:37:44.479Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[ailanguagesdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:3>",
  "operationType": "CREATE_PROJECT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "projects",
      "entityUri": "/projects/<ocid:2>",
      "identifier": "<ocid:2>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T19:35:15.579Z",
  "timeFinished": "2026-09-01T19:35:56.054Z",
  "timeStarted": "2026-09-01T19:35:55.810Z"
}`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[ailanguagesdk.WorkRequest](t, `{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "DELETE_PROJECT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "projects",
      "entityUri": "/projects/<ocid:2>",
      "identifier": "<ocid:2>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T19:36:49.488Z",
  "timeFinished": "2026-09-01T19:37:44.511Z",
  "timeStarted": "2026-09-01T19:37:44.267Z"
}`)

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[ailanguagesdk.Project, ailanguagesdk.CreateProjectDetails, ailanguagesdk.UpdateProjectDetails]{
		CollectionPath:    "/20221001/projects",
		ItemPath:          "/20221001/projects/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		ListShape:         ocimock.ListShapeItems,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		DeletedState:      &deletedState, RequireDeleteRead: true,
		CreateStatus: 200, UpdateStatus: 200, DeleteStatus: 202, NotFoundCode: "NotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:3>"}}, UpdateHeaders: nil, DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		ValidateCreate: func(request ocimock.Request, _ ailanguagesdk.CreateProjectDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{
				Name: "create-work-request", Method: http.MethodGet, Path: "/20221001/workRequests/<ocid:3>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
				},
			},
			{
				Name: "delete-work-request", Method: http.MethodGet, Path: "/20221001/workRequests/<ocid:4>", MinimumCalls: 1,
				Respond: func(ocimock.Request) (ocimock.Response, error) {
					return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://language.aiservice.us-ashburn-1.oci.oraclecloud.com", BasePath: "20221001", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	sdkClient := ailanguagesdk.AIServiceLanguageClient{BaseClient: session.BaseClient()}
	client := newProjectServiceClientWithOCIClient(log, sdkClient)

	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*ailanguagev1beta1.Project]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *ailanguagev1beta1.Project) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Project status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *ailanguagev1beta1.Project) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *ailanguagev1beta1.Project) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Project status = %+v", current.Status)
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
