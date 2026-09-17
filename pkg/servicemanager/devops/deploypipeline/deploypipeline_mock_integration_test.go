/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package deploypipeline

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

const mockDeployPipelineName = "osok-mock-deploy-pipeline-v1"

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationDeployPipelineWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &devopsv1beta1.DeployPipeline{}
	ocimock.InitializeResource(resource, "mock-deploypipeline")
	resource.Spec = ocimock.MustJSONFixture[devopsv1beta1.DeployPipelineSpec](t, `
{
  "description": "recorded create",
  "displayName": "osok-mock-deploy-pipeline-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "projectId": "<ocid:1>"
}
`)
	createRequest := ocimock.MustJSONFixture[devopssdk.CreateDeployPipelineDetails](t, `
{
  "description": "recorded create",
  "displayName": "osok-mock-deploy-pipeline-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "projectId": "<ocid:1>"
}
`)
	updateRequest := ocimock.MustJSONFixture[devopssdk.UpdateDeployPipelineDetails](t, `
{
  "definedTags": {},
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[devopssdk.DeployPipeline](t, `
{
  "compartmentId": "<ocid:4>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T22:07:27.486Z"
    }
  },
  "deployPipelineArtifacts": {
    "items": []
  },
  "deployPipelineEnvironments": {
    "items": []
  },
  "deployPipelineParameters": {
    "items": []
  },
  "description": "recorded create",
  "displayName": "osok-mock-deploy-pipeline-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "",
  "lifecycleState": "ACTIVE",
  "projectId": "<ocid:1>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "2026-09-01T22:07:27.660Z",
  "timeUpdated": "2026-09-01T22:07:27.660Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[devopssdk.DeployPipeline](t, `
{
  "compartmentId": "<ocid:4>",
  "definedTags": {},
  "deployPipelineArtifacts": {
    "items": []
  },
  "deployPipelineEnvironments": {
    "items": []
  },
  "deployPipelineParameters": {
    "items": []
  },
  "description": "recorded update",
  "displayName": "osok-mock-deploy-pipeline-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "",
  "lifecycleState": "ACTIVE",
  "projectId": "<ocid:1>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "2026-09-01T22:07:27.660Z",
  "timeUpdated": "2026-09-01T22:07:51.208Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:4>",
  "id": "<ocid:2>",
  "operationType": "CREATE_DEPLOY_PIPELINE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "deployPipeline",
      "entityUri": "/20210630/deployPipeline/<ocid:3>",
      "identifier": "<ocid:3>"
    },
    {
      "actionType": "CREATED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:1>",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T22:07:27.660Z",
  "timeFinished": "2026-09-01T22:07:48.024Z",
  "timeStarted": "2026-09-01T22:07:47.653Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:4>",
  "id": "<ocid:5>",
  "operationType": "UPDATE_DEPLOY_PIPELINE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "deployPipeline",
      "entityUri": "/20210630/deployPipeline/<ocid:3>",
      "identifier": "<ocid:3>"
    },
    {
      "actionType": "UPDATED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:1>",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T22:07:51.208Z",
  "timeFinished": "2026-09-01T22:07:57.033Z",
  "timeStarted": "2026-09-01T22:07:56.756Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:4>",
  "id": "<ocid:6>",
  "operationType": "DELETE_DEPLOY_PIPELINE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "deployPipeline",
      "entityUri": "/20210630/deployPipeline/<ocid:3>",
      "identifier": "<ocid:3>"
    },
    {
      "actionType": "DELETED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:1>",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T22:07:58.195Z",
  "timeFinished": "2026-09-01T22:08:38.448Z",
  "timeStarted": "2026-09-01T22:08:38.052Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[devopssdk.DeployPipeline, devopssdk.CreateDeployPipelineDetails, devopssdk.UpdateDeployPipelineDetails]{
		CollectionPath: "/20210630/deployPipelines", ItemPath: "/20210630/deployPipelines/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		ValidateCreate: func(request ocimock.Request, _ devopssdk.CreateDeployPipelineDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20210630/workRequests/<ocid:2>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
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
	client := newDeployPipelineServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	mockValidateCreated := func(current *devopsv1beta1.DeployPipeline) error {
		if current.Status.DisplayName != mockDeployPipelineName || current.Status.ProjectId != resource.Spec.ProjectId {
			return fmt.Errorf("created DeployPipeline status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *devopsv1beta1.DeployPipeline) error {
		if current.Status.Description != "recorded update" || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated DeployPipeline status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*devopsv1beta1.DeployPipeline]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *devopsv1beta1.DeployPipeline) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *devopsv1beta1.DeployPipeline) {
			current.Spec.Description = "recorded update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *devopsv1beta1.DeployPipeline) error {
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
