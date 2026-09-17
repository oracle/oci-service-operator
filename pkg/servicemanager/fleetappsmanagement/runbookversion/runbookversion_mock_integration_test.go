/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package runbookversion

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/fleetappsmanagement"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/fleetappsmanagement/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationRunbookVersionCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.RunbookVersion](t, `
{
  "metadata": {"name": "mock-runbookversion", "namespace": "default"},
  "spec": {
  "executionWorkflowDetails": {
    "workflow": [
      {
        "groupName": "mock-groupname",
        "steps": [
          {
            "groupName": "mock-groupname",
            "type": "PARALLEL_TASK_GROUP"
          }
        ],
        "type": "PARALLEL_RESOURCE_GROUP"
      }
    ]
  },
  "freeformTags": {
    "mock": "updated"
  },
  "groups": [],
  "rollbackWorkflowDetails": {
    "scope": "ACTION_GROUP",
    "workflow": []
  },
  "runbookId": "<ocid:required>",
  "tasks": []
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-runbookversion")
	resource.Status = apiv1beta1.RunbookVersionStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateRunbookVersionDetails](t, `{
  "executionWorkflowDetails": {
    "workflow": [
      {
        "groupName": "mock-groupname",
        "steps": [
          {
            "groupName": "mock-groupname",
            "type": "PARALLEL_TASK_GROUP"
          }
        ],
        "type": "PARALLEL_RESOURCE_GROUP"
      }
    ]
  },
  "freeformTags": {
    "mock": "updated"
  },
  "groups": [],
  "rollbackWorkflowDetails": {
    "scope": "ACTION_GROUP",
    "workflow": []
  },
  "runbookId": "<ocid:required>",
  "tasks": []
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateRunbookVersionDetails](t, `{
  "rollbackWorkflowDetails": {
    "scope": "TARGET",
    "workflow": []
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.RunbookVersion](t, `{
  "executionWorkflowDetails": {
    "workflow": [
      {
        "groupName": "mock-groupname",
        "steps": [
          {
            "groupName": "mock-groupname",
            "type": "PARALLEL_TASK_GROUP"
          }
        ],
        "type": "PARALLEL_RESOURCE_GROUP"
      }
    ]
  },
  "freeformTags": {
    "mock": "updated"
  },
  "groups": [],
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "rollbackWorkflowDetails": {
    "scope": "ACTION_GROUP",
    "workflow": []
  },
  "runbookId": "<ocid:required>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "tasks": [],
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.RunbookVersion](t, `{
  "executionWorkflowDetails": {
    "workflow": [
      {
        "groupName": "mock-groupname",
        "steps": [
          {
            "groupName": "mock-groupname",
            "type": "PARALLEL_TASK_GROUP"
          }
        ],
        "type": "PARALLEL_RESOURCE_GROUP"
      }
    ]
  },
  "freeformTags": {
    "mock": "updated"
  },
  "groups": [],
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "rollbackWorkflowDetails": {
    "scope": "TARGET",
    "workflow": []
  },
  "runbookId": "<ocid:required>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "tasks": [],
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATERUNBOOKVERSION",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "RunbookVersion", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATERUNBOOKVERSION",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "RunbookVersion", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETERUNBOOKVERSION",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "RunbookVersion", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.RunbookVersion, sdksvc.CreateRunbookVersionDetails, sdksvc.UpdateRunbookVersionDetails]{
		CollectionPath: "/20250228/runbookVersions", ItemPath: "/20250228/runbookVersions/<ocid:1>",
		CreatePath: "/20250228/runbookVersions", CreateMethod: http.MethodPost,
		UpdatePath: "/20250228/runbookVersions/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20250228/runbookVersions/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateRunbookVersionDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateRunbookVersionDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateRunbookVersionDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://fams.us-ashburn-1.oci.oraclecloud.com", BasePath: "20250228", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := RunbookVersionSDKClients{fleetAppsManagementRunbooksClient: sdksvc.FleetAppsManagementRunbooksClient{BaseClient: session.BaseClient()}, fleetAppsManagementWorkRequestClient: sdksvc.FleetAppsManagementWorkRequestClient{BaseClient: session.BaseClient()}}
	manager := &RunbookVersionServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newRunbookVersionRuntimeHooks(manager, sdkClient)
	client := wrapRunbookVersionGeneratedClient(hooks, defaultRunbookVersionServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.RunbookVersion](buildRunbookVersionGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.RunbookVersion]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.RunbookVersion) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.RollbackWorkflowDetails.Scope != "ACTION_GROUP" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created RunbookVersion status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.RunbookVersion) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "rollbackWorkflowDetails": {
    "scope": "TARGET",
    "workflow": []
  }
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.RunbookVersion) error {
			if current.Status.RollbackWorkflowDetails.Scope != "TARGET" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated RunbookVersion status = %+v", current.Status)
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
