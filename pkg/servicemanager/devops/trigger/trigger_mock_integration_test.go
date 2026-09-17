/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package trigger

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

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationTriggerWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &devopsv1beta1.Trigger{}
	ocimock.InitializeResource(resource, "mock-trigger")
	resource.Spec = devopsv1beta1.TriggerSpec{
		JsonData:      `{"actions":[{"buildPipelineId":"<ocid:2>","type":"TRIGGER_BUILD_PIPELINE"}]}`,
		Description:   "OSOK recorded DevOps trigger",
		DisplayName:   "osok-mock-devops-trigger",
		FreeformTags:  map[string]string{"osok-mock": "create"},
		ProjectId:     "<ocid:1>",
		RepositoryId:  "<ocid:3>",
		TriggerSource: string(devopssdk.TriggerTriggerSourceDevopsCodeRepository),
	}
	createRequest := ocimock.MustJSONFixture[devopssdk.CreateDevopsCodeRepositoryTriggerDetails](t, `
{
  "actions": [
    {
      "buildPipelineId": "<ocid:2>",
      "type": "TRIGGER_BUILD_PIPELINE"
    }
  ],
  "description": "OSOK recorded DevOps trigger",
  "displayName": "osok-mock-devops-trigger",
  "freeformTags": {
    "osok-mock": "create"
  },
  "projectId": "<ocid:1>",
  "repositoryId": "<ocid:3>",
  "triggerSource": "DEVOPS_CODE_REPOSITORY"
}
`)
	updateRequest := ocimock.MustJSONFixture[devopssdk.UpdateDevopsCodeRepositoryTriggerDetails](t, `
{
  "actions": [
    {
      "buildPipelineId": "<ocid:2>",
      "type": "TRIGGER_BUILD_PIPELINE"
    }
  ],
  "description": "OSOK recorded DevOps trigger updated",
  "displayName": "osok-mock-devops-trigger",
  "freeformTags": {
    "osok-mock": "update"
  },
  "repositoryId": "<ocid:3>",
  "triggerSource": "DEVOPS_CODE_REPOSITORY"
}
`)
	createdState := ocimock.MustOCIResponseFixture[devopssdk.DevopsCodeRepositoryTrigger](t, `
{
  "actions": [
    {
      "buildPipelineId": "<ocid:2>",
      "filter": null,
      "type": "TRIGGER_BUILD_PIPELINE"
    }
  ],
  "compartmentId": "<ocid:5>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:35:04.354Z"
    }
  },
  "description": "OSOK recorded DevOps trigger",
  "displayName": "osok-mock-devops-trigger",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:6>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "projectId": "<ocid:1>",
  "repositoryId": "<ocid:3>",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:35:04.559Z",
  "timeUpdated": "2026-09-02T20:35:04.559Z",
  "triggerSource": "DEVOPS_CODE_REPOSITORY"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[devopssdk.DevopsCodeRepositoryTrigger](t, `
{
  "actions": [
    {
      "buildPipelineId": "<ocid:2>",
      "filter": null,
      "type": "TRIGGER_BUILD_PIPELINE"
    }
  ],
  "compartmentId": "<ocid:5>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:35:04.354Z"
    }
  },
  "description": "OSOK recorded DevOps trigger updated",
  "displayName": "osok-mock-devops-trigger",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:6>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "projectId": "<ocid:1>",
  "repositoryId": "<ocid:3>",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:35:04.559Z",
  "timeUpdated": "2026-09-02T20:35:17.603Z",
  "triggerSource": "DEVOPS_CODE_REPOSITORY"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:5>",
  "id": "<ocid:4>",
  "operationType": "CREATE_TRIGGER",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "trigger",
      "entityUri": "/20210630/triggers/<ocid:6>",
      "identifier": "<ocid:6>"
    },
    {
      "actionType": "CREATED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:1>",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T20:35:04.566Z",
  "timeFinished": "2026-09-02T20:35:10.873Z",
  "timeStarted": "2026-09-02T20:35:10.304Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:5>",
  "id": "<ocid:7>",
  "operationType": "UPDATE_TRIGGER",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "trigger",
      "entityUri": "/20210630/triggers/<ocid:6>",
      "identifier": "<ocid:6>"
    },
    {
      "actionType": "UPDATED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:1>",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T20:35:17.669Z",
  "timeFinished": "2026-09-02T20:35:39.369Z",
  "timeStarted": "2026-09-02T20:35:39.112Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:5>",
  "id": "<ocid:8>",
  "operationType": "DELETE_TRIGGER",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "trigger",
      "entityUri": "/20210630/triggers/<ocid:6>",
      "identifier": "<ocid:6>"
    },
    {
      "actionType": "DELETED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:1>",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T20:35:51.633Z",
  "timeFinished": "2026-09-02T20:36:30.044Z",
  "timeStarted": "2026-09-02T20:36:29.586Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[devopssdk.DevopsCodeRepositoryTrigger, devopssdk.CreateDevopsCodeRepositoryTriggerDetails, devopssdk.UpdateDevopsCodeRepositoryTriggerDetails]{
		CollectionPath: "/20210630/triggers", ItemPath: "/20210630/triggers/<ocid:6>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:7>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:8>"}},
		ValidateCreate: func(request ocimock.Request, _ devopssdk.CreateDevopsCodeRepositoryTriggerDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20210630/workRequests/<ocid:4>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20210630/workRequests/<ocid:7>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20210630/workRequests/<ocid:8>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
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
	client := newTriggerServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
		sdkClient,
	)
	mockValidateCreated := func(current *devopsv1beta1.Trigger) error {
		if current.Status.DisplayName != "osok-mock-devops-trigger" || current.Status.LifecycleState != string(devopssdk.TriggerLifecycleStateActive) {
			return fmt.Errorf("created Trigger status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *devopsv1beta1.Trigger) error {
		if current.Status.Description != "OSOK recorded DevOps trigger updated" || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated Trigger status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*devopsv1beta1.Trigger]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *devopsv1beta1.Trigger) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *devopsv1beta1.Trigger) {
			current.Spec.Description = "OSOK recorded DevOps trigger updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *devopsv1beta1.Trigger) error {
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
