/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package project

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	devopssdk "github.com/oracle/oci-go-sdk/v65/devops"
	devopsv1beta1 "github.com/oracle/oci-service-operator/api/devops/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockDevopsProjectName = "osok-mock-common-devops-project-v1"

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationProjectWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &devopsv1beta1.Project{}
	ocimock.InitializeResource(resource, "mock-project")
	resource.Spec = ocimock.MustJSONFixture[devopsv1beta1.ProjectSpec](t, `
{
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "freeformTags": {
    "osok-mock": "create"
  },
  "name": "osok-mock-common-devops-project-v1",
  "notificationConfig": {
    "topicId": "<ocid:2>"
  }
}
`)
	createRequest := ocimock.MustJSONFixture[devopssdk.CreateProjectDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "freeformTags": {
    "osok-mock": "create"
  },
  "name": "osok-mock-common-devops-project-v1",
  "notificationConfig": {
    "topicId": "<ocid:2>"
  }
}
`)
	updateRequest := ocimock.MustJSONFixture[devopssdk.UpdateProjectDetails](t, `
{
  "definedTags": {},
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[devopssdk.Project](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T00:15:41.081Z"
    }
  },
  "description": "recorded create",
  "disasterRecoveryProjectDetails": null,
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "",
  "lifecycleState": "ACTIVE",
  "name": "osok-mock-common-devops-project-v1",
  "namespace": "iddevjmhjw0n",
  "notificationConfig": {
    "topicId": "<ocid:2>"
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "2026-09-01T00:15:41.303Z",
  "timeUpdated": "2026-09-01T00:15:41.303Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[devopssdk.Project](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "description": "recorded update",
  "disasterRecoveryProjectDetails": null,
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "",
  "lifecycleState": "ACTIVE",
  "name": "osok-mock-common-devops-project-v1",
  "namespace": "iddevjmhjw0n",
  "notificationConfig": {
    "topicId": "<ocid:2>"
  },
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "2026-09-01T00:15:41.303Z",
  "timeUpdated": "2026-09-01T00:15:42.529Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "CREATE_PROJECT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T00:15:41.303Z",
  "timeFinished": "2026-09-01T00:15:41.311Z",
  "timeStarted": "2026-09-01T00:15:41.311Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "UPDATE_PROJECT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T00:15:42.645Z",
  "timeFinished": "2026-09-01T00:15:42.648Z",
  "timeStarted": "2026-09-01T00:15:42.648Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:6>",
  "operationType": "DELETE_PROJECT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T00:15:44.056Z",
  "timeFinished": "2026-09-01T00:15:44.066Z",
  "timeStarted": "2026-09-01T00:15:44.066Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[devopssdk.Project, devopssdk.CreateProjectDetails, devopssdk.UpdateProjectDetails]{
		CollectionPath: "/20210630/projects", ItemPath: "/20210630/projects/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		ValidateCreate: func(request ocimock.Request, _ devopssdk.CreateProjectDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20210630/workRequests/<ocid:4>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20210630/workRequests/<ocid:5>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20210630/workRequests/<ocid:6>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://devops.us-ashburn-1.oci.oraclecloud.com", BasePath: "20210630", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := devopssdk.DevopsClient{BaseClient: session.BaseClient()}
	client := newProjectServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
		sdkClient,
	)
	mockValidateCreated := func(current *devopsv1beta1.Project) error {
		if current.Status.LifecycleState != string(devopssdk.ProjectLifecycleStateActive) || current.Status.Name != mockDevopsProjectName {
			return fmt.Errorf("created DevOps Project status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *devopsv1beta1.Project) error {
		if current.Status.Description != current.Spec.Description || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated DevOps Project status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*devopsv1beta1.Project]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *devopsv1beta1.Project) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *devopsv1beta1.Project) {
			current.Spec.Description = "recorded update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *devopsv1beta1.Project) error {
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
