/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package buildpipeline

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

const mockBuildPipelineName = "osok-mock-build-pipeline-v1"

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationBuildPipelineWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &devopsv1beta1.BuildPipeline{}
	ocimock.InitializeResource(resource, "mock-buildpipeline")
	resource.Spec = ocimock.MustJSONFixture[devopsv1beta1.BuildPipelineSpec](t, `
{
  "description": "recorded create",
  "displayName": "osok-mock-build-pipeline-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "projectId": "<ocid:1>"
}
`)
	createRequest := ocimock.MustJSONFixture[devopssdk.CreateBuildPipelineDetails](t, `
{
  "description": "recorded create",
  "displayName": "osok-mock-build-pipeline-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "projectId": "<ocid:1>"
}
`)
	updateRequest := ocimock.MustJSONFixture[devopssdk.UpdateBuildPipelineDetails](t, `
{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[devopssdk.BuildPipeline](t, `
{
  "autoRetryConfigDetails": null,
  "buildPipelineParameters": {
    "items": []
  },
  "compartmentId": "<ocid:3>",
  "concurrentPipelineRun": null,
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T22:07:27.495Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-build-pipeline-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:4>",
  "importRepositorySourceDetails": null,
  "initialBuildRevisionNumber": null,
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "pipelineType": "EXTERNAL",
  "projectId": "<ocid:1>",
  "realmLevelPipelineId": null,
  "retentionPolicy": null,
  "runnability": null,
  "systemTags": {},
  "tenantId": null,
  "timeCreated": "2026-09-01T22:07:28.077Z",
  "timeUpdated": "2026-09-01T22:08:18.782Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[devopssdk.BuildPipeline](t, `
{
  "autoRetryConfigDetails": null,
  "buildPipelineParameters": {
    "items": []
  },
  "compartmentId": "<ocid:3>",
  "concurrentPipelineRun": null,
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T22:07:27.495Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-build-pipeline-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:4>",
  "importRepositorySourceDetails": null,
  "initialBuildRevisionNumber": null,
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "pipelineType": "EXTERNAL",
  "projectId": "<ocid:1>",
  "realmLevelPipelineId": null,
  "retentionPolicy": null,
  "runnability": null,
  "systemTags": {},
  "tenantId": null,
  "timeCreated": "2026-09-01T22:07:28.077Z",
  "timeUpdated": "2026-09-01T22:08:26.818Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:2>",
  "operationType": "CREATE_BUILD_PIPELINE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "buildPipeline",
      "entityUri": "/20210630/buildPipelines/<ocid:4>",
      "identifier": "<ocid:4>"
    },
    {
      "actionType": "CREATED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:1>",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T22:07:28.077Z",
  "timeFinished": "2026-09-01T22:08:18.765Z",
  "timeStarted": "2026-09-01T22:07:42.553Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:5>",
  "operationType": "UPDATE_BUILD_PIPELINE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "buildPipeline",
      "entityUri": "/20210630/buildPipelines/<ocid:4>",
      "identifier": "<ocid:4>"
    },
    {
      "actionType": "UPDATED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:1>",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T22:08:26.894Z",
  "timeFinished": "2026-09-01T22:09:08.720Z",
  "timeStarted": "2026-09-01T22:08:51.953Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:6>",
  "operationType": "DELETE_BUILD_PIPELINE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "buildPipeline",
      "entityUri": "/20210630/buildPipelines/<ocid:4>",
      "identifier": "<ocid:4>"
    },
    {
      "actionType": "DELETED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:1>",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T22:09:14.482Z",
  "timeFinished": "2026-09-01T22:09:38.756Z",
  "timeStarted": "2026-09-01T22:09:38.282Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[devopssdk.BuildPipeline, devopssdk.CreateBuildPipelineDetails, devopssdk.UpdateBuildPipelineDetails]{
		CollectionPath: "/20210630/buildPipelines", ItemPath: "/20210630/buildPipelines/<ocid:4>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		ValidateCreate: func(request ocimock.Request, _ devopssdk.CreateBuildPipelineDetails) error {
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
	client := newBuildPipelineServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	mockValidateCreated := func(current *devopsv1beta1.BuildPipeline) error {
		if current.Status.DisplayName != mockBuildPipelineName || current.Status.ProjectId != resource.Spec.ProjectId {
			return fmt.Errorf("created BuildPipeline status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *devopsv1beta1.BuildPipeline) error {
		if current.Status.Description != "recorded update" || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated BuildPipeline status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*devopsv1beta1.BuildPipeline]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *devopsv1beta1.BuildPipeline) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *devopsv1beta1.BuildPipeline) {
			current.Spec.Description = "recorded update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *devopsv1beta1.BuildPipeline) error {
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
