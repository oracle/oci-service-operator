/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package deployartifact

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
func TestMockIntegrationDeployArtifactWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &devopsv1beta1.DeployArtifact{}
	ocimock.InitializeResource(resource, "mock-deployartifact")
	resource.Spec = ocimock.MustJSONFixture[devopsv1beta1.DeployArtifactSpec](t, `
{
  "argumentSubstitutionMode": "NONE",
  "deployArtifactSource": {
    "deployArtifactPath": "manifests/app.yaml",
    "deployArtifactSourceType": "GENERIC_ARTIFACT",
    "deployArtifactVersion": "1.0.0",
    "repositoryId": "<ocid:1>"
  },
  "deployArtifactType": "KUBERNETES_MANIFEST",
  "description": "OSOK recorded deploy artifact",
  "displayName": "osok-mock-deploy-artifact",
  "freeformTags": {
    "osok-mock": "create"
  },
  "projectId": "<ocid:2>"
}
`)
	createRequest := ocimock.MustJSONFixture[devopssdk.CreateDeployArtifactDetails](t, `
{
  "argumentSubstitutionMode": "NONE",
  "deployArtifactSource": {
    "deployArtifactPath": "manifests/app.yaml",
    "deployArtifactSourceType": "GENERIC_ARTIFACT",
    "deployArtifactVersion": "1.0.0",
    "repositoryId": "<ocid:1>"
  },
  "deployArtifactType": "KUBERNETES_MANIFEST",
  "description": "OSOK recorded deploy artifact",
  "displayName": "osok-mock-deploy-artifact",
  "freeformTags": {
    "osok-mock": "create"
  },
  "projectId": "<ocid:2>"
}
`)
	updateRequest := ocimock.MustJSONFixture[devopssdk.UpdateDeployArtifactDetails](t, `
{
  "description": "OSOK recorded deploy artifact updated",
  "freeformTags": {
    "osok-mock": "update"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[devopssdk.DeployArtifact](t, `
{
  "argumentSubstitutionMode": "NONE",
  "compartmentId": "<ocid:5>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:23:00.616Z"
    }
  },
  "deployArtifactSource": {
    "deployArtifactPath": "manifests/app.yaml",
    "deployArtifactSourceType": "GENERIC_ARTIFACT",
    "deployArtifactVersion": "1.0.0",
    "repositoryId": "<ocid:1>"
  },
  "deployArtifactType": "KUBERNETES_MANIFEST",
  "description": "OSOK recorded deploy artifact",
  "displayName": "osok-mock-deploy-artifact",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:4>",
  "lifecycleDetails": "",
  "lifecycleState": "ACTIVE",
  "projectId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "2026-09-02T20:23:00.839Z",
  "timeUpdated": "2026-09-02T20:23:00.839Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[devopssdk.DeployArtifact](t, `
{
  "argumentSubstitutionMode": "NONE",
  "compartmentId": "<ocid:5>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:23:00.616Z"
    }
  },
  "deployArtifactSource": {
    "deployArtifactPath": "manifests/app.yaml",
    "deployArtifactSourceType": "GENERIC_ARTIFACT",
    "deployArtifactVersion": "1.0.0",
    "repositoryId": "<ocid:1>"
  },
  "deployArtifactType": "KUBERNETES_MANIFEST",
  "description": "OSOK recorded deploy artifact updated",
  "displayName": "osok-mock-deploy-artifact",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:4>",
  "lifecycleDetails": "",
  "lifecycleState": "ACTIVE",
  "projectId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "true"
    }
  },
  "timeCreated": "2026-09-02T20:23:00.839Z",
  "timeUpdated": "2026-09-02T20:23:01.957Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:5>",
  "id": "<ocid:3>",
  "operationType": "CREATE_DEPLOY_ARTIFACT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "deployArtifact",
      "entityUri": "/20210630/deployArtifact/<ocid:4>",
      "identifier": "<ocid:4>"
    },
    {
      "actionType": "CREATED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:2>",
      "identifier": "<ocid:2>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T20:23:00.819Z",
  "timeFinished": "2026-09-02T20:23:00.830Z",
  "timeStarted": "2026-09-02T20:23:00.830Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:5>",
  "id": "<ocid:6>",
  "operationType": "UPDATE_DEPLOY_ARTIFACT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "deployArtifact",
      "entityUri": "/20210630/deployArtifact/<ocid:4>",
      "identifier": "<ocid:4>"
    },
    {
      "actionType": "UPDATED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:2>",
      "identifier": "<ocid:2>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T20:23:01.958Z",
  "timeFinished": "2026-09-02T20:23:34.459Z",
  "timeStarted": "2026-09-02T20:23:34.218Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:5>",
  "id": "<ocid:7>",
  "operationType": "DELETE_DEPLOY_ARTIFACT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "deployArtifact",
      "entityUri": "/20210630/deployArtifact/<ocid:4>",
      "identifier": "<ocid:4>"
    },
    {
      "actionType": "DELETED",
      "entityType": "project",
      "entityUri": "/20210630/projects/<ocid:2>",
      "identifier": "<ocid:2>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-02T20:23:46.403Z",
  "timeFinished": "2026-09-02T20:23:46.411Z",
  "timeStarted": "2026-09-02T20:23:46.411Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[devopssdk.DeployArtifact, devopssdk.CreateDeployArtifactDetails, devopssdk.UpdateDeployArtifactDetails]{
		CollectionPath: "/20210630/deployArtifacts", ItemPath: "/20210630/deployArtifacts/<ocid:4>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:3>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:7>"}},
		ValidateCreate: func(request ocimock.Request, _ devopssdk.CreateDeployArtifactDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20210630/workRequests/<ocid:3>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20210630/workRequests/<ocid:6>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20210630/workRequests/<ocid:7>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
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
	client := newDeployArtifactServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
		sdkClient,
	)
	mockValidateCreated := func(current *devopsv1beta1.DeployArtifact) error {
		if current.Status.DisplayName != "osok-mock-deploy-artifact" {
			return fmt.Errorf("created DeployArtifact status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *devopsv1beta1.DeployArtifact) error {
		if current.Status.Description != "OSOK recorded deploy artifact updated" || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated DeployArtifact status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*devopsv1beta1.DeployArtifact]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *devopsv1beta1.DeployArtifact) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *devopsv1beta1.DeployArtifact) {
			current.Spec.Description = "OSOK recorded deploy artifact updated"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *devopsv1beta1.DeployArtifact) error {
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
