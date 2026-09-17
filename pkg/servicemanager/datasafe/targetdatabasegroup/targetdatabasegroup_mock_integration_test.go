/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package targetdatabasegroup

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockTargetDatabaseGroupName = "osok-mock-target-database-group-v1"

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationTargetDatabaseGroupWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resource := &datasafev1beta1.TargetDatabaseGroup{}
	ocimock.InitializeResource(resource, "mock-targetdatabasegroup")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.TargetDatabaseGroupSpec](t, `
{
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-target-database-group-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "matchingCriteria": {
    "include": {
      "compartments": [
        {
          "id": "<ocid:1>"
        }
      ]
    }
  }
}
`)
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateTargetDatabaseGroupDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "description": "recorded create",
  "displayName": "osok-mock-target-database-group-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "matchingCriteria": {
    "include": {
      "compartments": [
        {
          "id": "<ocid:1>"
        }
      ]
    }
  }
}
`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateTargetDatabaseGroupDetails](t, `
{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  },
  "matchingCriteria": {
    "include": {
      "compartments": [
        {
          "id": "<ocid:1>"
        }
      ]
    }
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.TargetDatabaseGroup](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T03:37:00.991Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-target-database-group-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "Target database group is refreshed",
  "lifecycleState": "ACTIVE",
  "matchingCriteria": {
    "include": {
      "compartments": [
        {
          "id": "<ocid:1>"
        }
      ]
    }
  },
  "membershipCount": 0,
  "membershipUpdateTime": "2026-09-03T03:37:03.804Z",
  "systemTags": {},
  "timeCreated": "2026-09-03T03:37:01.099Z",
  "timeUpdated": "2026-09-03T03:37:01.099Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.TargetDatabaseGroup](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T03:37:00.991Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-target-database-group-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "Target database group is refreshed",
  "lifecycleState": "ACTIVE",
  "matchingCriteria": {
    "include": {
      "compartments": [
        {
          "id": "<ocid:1>"
        }
      ]
    }
  },
  "membershipCount": 0,
  "membershipUpdateTime": "2026-09-03T03:37:10.445Z",
  "systemTags": {},
  "timeCreated": "2026-09-03T03:37:01.099Z",
  "timeUpdated": "2026-09-03T03:37:07.344Z"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[datasafesdk.TargetDatabaseGroup](t, `
{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T03:37:00.991Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-target-database-group-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "lifecycleDetails": "Target database group is deleted",
  "lifecycleState": "DELETED",
  "matchingCriteria": {
    "include": {
      "compartments": [
        {
          "id": "<ocid:1>"
        }
      ]
    }
  },
  "membershipCount": 0,
  "membershipUpdateTime": "2026-09-03T03:37:10.445Z",
  "systemTags": {},
  "timeCreated": "2026-09-03T03:37:01.099Z",
  "timeUpdated": "2026-09-03T03:37:17.413Z"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:2>",
  "operationType": "CREATE_TARGET_DATABASE_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "targetdatabasegroup",
      "entityUri": "/targetDatabaseGroups/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T03:37:01.161Z",
  "timeFinished": "2026-09-03T03:37:03.889Z",
  "timeStarted": "2026-09-03T03:37:03.285Z"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:4>",
  "operationType": "UPDATE_TARGET_DATABASE_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "targetdatabasegroup",
      "entityUri": "/targetDatabaseGroups/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T03:37:07.387Z",
  "timeFinished": "2026-09-03T03:37:10.489Z",
  "timeStarted": "2026-09-03T03:37:10.367Z"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "compartmentId": "<ocid:1>",
  "id": "<ocid:5>",
  "operationType": "DELETE_TARGET_DATABASE_GROUP",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "targetdatabasegroup",
      "entityUri": "/targetDatabaseGroups/<ocid:3>",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED",
  "timeAccepted": "2026-09-03T03:37:13.586Z",
  "timeFinished": "2026-09-03T03:37:17.865Z",
  "timeStarted": "2026-09-03T03:37:17.043Z"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[datasafesdk.TargetDatabaseGroup, datasafesdk.CreateTargetDatabaseGroupDetails, datasafesdk.UpdateTargetDatabaseGroupDetails]{
		CollectionPath: "/20181201/targetDatabaseGroups", ItemPath: "/20181201/targetDatabaseGroups/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		DeletedState: &deletedState,
		ListShape:    ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:2>"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:4>"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:5>"}},
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateTargetDatabaseGroupDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:2>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, createWorkRequest)
			}},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:4>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, updateWorkRequest)
			}},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:5>", MinimumCalls: 1, Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, deleteWorkRequest)
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://datasafe.us-ashburn-1.oci.oraclecloud.com", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	manager := &TargetDatabaseGroupServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newTargetDatabaseGroupDefaultRuntimeHooks(sdkClient)
	applyTargetDatabaseGroupRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapTargetDatabaseGroupGeneratedClient(hooks, defaultTargetDatabaseGroupServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.TargetDatabaseGroup](buildTargetDatabaseGroupGeneratedRuntimeConfig(manager, hooks)),
	})
	mockValidateCreated := func(current *datasafev1beta1.TargetDatabaseGroup) error {
		if current.Status.OsokStatus.Ocid == "" || current.Status.DisplayName != mockTargetDatabaseGroupName {
			return fmt.Errorf("created TargetDatabaseGroup status = %+v", current.Status)
		}
		return nil
	}
	mockValidateUpdated := func(current *datasafev1beta1.TargetDatabaseGroup) error {
		if current.Status.Description != "recorded update" || current.Status.FreeformTags["osok-mock"] != "update" {
			return fmt.Errorf("updated TargetDatabaseGroup status = %+v", current.Status)
		}
		return nil
	}
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.TargetDatabaseGroup]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.TargetDatabaseGroup) error {
			if err := mockValidateCreated(current); err != nil {
				return err
			}
			if current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created async state = %+v, want cleared", current.Status.OsokStatus.Async.Current)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.TargetDatabaseGroup) {
			current.Spec.Description = "recorded update"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *datasafev1beta1.TargetDatabaseGroup) error {
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
