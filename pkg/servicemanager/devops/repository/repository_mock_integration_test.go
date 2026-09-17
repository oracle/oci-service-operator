/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package repository

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

const mockRepositoryName = "osok-mock-repository-v1"

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationRepositoryWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &devopsv1beta1.Repository{}
	ocimock.InitializeResource(resource, "mock-repository")
	resource.Spec = ocimock.MustJSONFixture[devopsv1beta1.RepositorySpec](t, `
{
  "defaultBranch": "main",
  "description": "recorded create",
  "freeformTags": {
    "osok-mock": "create"
  },
  "name": "osok-mock-repository-v1",
  "projectId": "<ocid:1>",
  "repositoryType": "HOSTED"
}
`)
	createRequest := ocimock.MustJSONFixture[devopssdk.CreateRepositoryDetails](t, `
{
  "defaultBranch": "main",
  "description": "recorded create",
  "freeformTags": {
    "osok-mock": "create"
  },
  "name": "osok-mock-repository-v1",
  "projectId": "<ocid:1>",
  "repositoryType": "HOSTED"
}
`)
	updateRequest := ocimock.MustJSONFixture[devopssdk.UpdateRepositoryDetails](t, `
{
  "defaultBranch": "main",
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[devopssdk.Repository](t, `
{
  "branchCount": 0,
  "commitCount": 0,
  "compartmentId": "<ocid:3>",
  "defaultBranch": "refs/heads/main",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T03:10:34.175Z"
    }
  },
  "description": "recorded create",
  "disasterRecoveryDetails": null,
  "freeformTags": {
    "osok-mock": "create"
  },
  "httpUrl": "https://devops.scmservice.us-ashburn-1.oci.oraclecloud.com/namespaces/iddevjmhjw0n/projects/osok-mock-repository-project-v1/repositories/osok-mock-repository-v1",
  "id": "<ocid:4>",
  "lifecycleState": "ACTIVE",
  "lifecyleDetails": "",
  "mirrorRepositoryConfig": null,
  "name": "osok-mock-repository-v1",
  "namespace": "iddevjmhjw0n",
  "parentRepositoryId": "",
  "projectId": "<ocid:1>",
  "projectName": "osok-mock-repository-project-v1",
  "repositoryType": "HOSTED",
  "sizeInBytes": 0,
  "sshUrl": "ssh://devops.scmservice.us-ashburn-1.oci.oraclecloud.com/namespaces/iddevjmhjw0n/projects/osok-mock-repository-project-v1/repositories/osok-mock-repository-v1",
  "systemTags": {},
  "timeCreated": "2026-09-01T03:10:34.750Z",
  "timeUpdated": "2026-09-01T03:10:34.750Z",
  "triggerBuildEvents": [
    "PUSH"
  ]
}
`)
	updatedState := ocimock.MustOCIResponseFixture[devopssdk.Repository](t, `
{
  "branchCount": 0,
  "commitCount": 0,
  "compartmentId": "<ocid:3>",
  "defaultBranch": "refs/heads/main",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-01T03:10:34.175Z"
    }
  },
  "description": "recorded update",
  "disasterRecoveryDetails": null,
  "freeformTags": {
    "osok-mock": "update"
  },
  "httpUrl": "https://devops.scmservice.us-ashburn-1.oci.oraclecloud.com/namespaces/iddevjmhjw0n/projects/osok-mock-repository-project-v1/repositories/osok-mock-repository-v1",
  "id": "<ocid:4>",
  "lifecycleState": "ACTIVE",
  "lifecyleDetails": "",
  "mirrorRepositoryConfig": null,
  "name": "osok-mock-repository-v1",
  "namespace": "iddevjmhjw0n",
  "parentRepositoryId": "",
  "projectId": "<ocid:1>",
  "projectName": "osok-mock-repository-project-v1",
  "repositoryType": "HOSTED",
  "sizeInBytes": 0,
  "sshUrl": "ssh://devops.scmservice.us-ashburn-1.oci.oraclecloud.com/namespaces/iddevjmhjw0n/projects/osok-mock-repository-project-v1/repositories/osok-mock-repository-v1",
  "systemTags": {},
  "timeCreated": "2026-09-01T03:10:34.750Z",
  "timeUpdated": "2026-09-01T03:10:56.704Z",
  "triggerBuildEvents": [
    "PUSH"
  ]
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:2>",
  "operationType": "CREATE_REPOSITORY",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "repository",
      "entityUri": "/repositories/<ocid:4>",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T03:10:34.741Z",
  "timeFinished": "2026-09-01T03:10:55.462Z",
  "timeStarted": "2026-09-01T03:10:53.880Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:5>",
  "operationType": "UPDATE_REPOSITORY",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "repository",
      "entityUri": "/repositories/<ocid:4>",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T03:10:56.704Z",
  "timeFinished": "2026-09-01T03:10:56.707Z",
  "timeStarted": "2026-09-01T03:10:56.707Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[devopssdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:3>",
  "id": "<ocid:6>",
  "operationType": "DELETE_REPOSITORY",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "repository",
      "entityUri": "/repositories/<ocid:4>",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-01T03:10:58.468Z",
  "timeFinished": "2026-09-01T03:11:14.182Z",
  "timeStarted": "2026-09-01T03:11:12.968Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[devopssdk.Repository, devopssdk.CreateRepositoryDetails, devopssdk.UpdateRepositoryDetails]{
		CollectionPath: "/20210630/repositories", ItemPath: "/20210630/repositories/<ocid:4>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:6>"}},
		ValidateCreate: func(request ocimock.Request, _ devopssdk.CreateRepositoryDetails) error {
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
	manager := &RepositoryServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newRepositoryDefaultRuntimeHooks(sdkClient)
	applyRepositoryRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapRepositoryGeneratedClient(hooks, defaultRepositoryServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*devopsv1beta1.Repository](
			buildRepositoryGeneratedRuntimeConfig(manager, hooks),
		),
	})
	mockValidateCreated := func(current *devopsv1beta1.Repository) error {
		if current.Status.LifecycleState != string(devopssdk.RepositoryLifecycleStateActive) || current.Status.Name != mockRepositoryName || current.Status.ProjectId != current.Spec.ProjectId {
			return fmt.Errorf("created Repository status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *devopsv1beta1.Repository) error {
		if current.Status.Description != current.Spec.Description || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated Repository status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*devopsv1beta1.Repository]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *devopsv1beta1.Repository) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *devopsv1beta1.Repository) {
			current.Spec.Description = "recorded update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *devopsv1beta1.Repository) error {
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
