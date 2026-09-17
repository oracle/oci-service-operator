/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package runbook

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
func TestMockIntegrationRunbookCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.Runbook](t, `
{
  "metadata": {"name": "mock-runbook", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "operation": "mock-operation",
  "runbookVersion": {
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
    "groups": [
      {
        "name": "mock-name",
        "type": "PARALLEL_TASK_GROUP"
      }
    ],
    "isLatest": true,
    "tasks": [
      {
        "stepName": "mock-stepname",
        "taskRecordDetails": {
          "executionDetails": {
            "command": "mock-command",
            "executionType": "SCRIPT"
          },
          "scope": "LOCAL"
        }
      }
    ]
  }
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-runbook")
	resource.Status = apiv1beta1.RunbookStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateRunbookDetails](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "operation": "mock-operation",
  "runbookVersion": {
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
    "groups": [
      {
        "name": "mock-name",
        "type": "PARALLEL_TASK_GROUP"
      }
    ],
    "isLatest": true,
    "tasks": [
      {
        "stepName": "mock-stepname",
        "taskRecordDetails": {
          "executionDetails": {
            "command": "mock-command",
            "executionType": "SCRIPT"
          },
          "scope": "LOCAL"
        }
      }
    ]
  }
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateRunbookDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.Runbook](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "INACTIVE",
  "operation": "mock-operation",
  "resourceId": "<ocid:1>",
  "runbookVersion": {
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
    "groups": [
      {
        "name": "mock-name",
        "type": "PARALLEL_TASK_GROUP"
      }
    ],
    "isLatest": true,
    "tasks": [
      {
        "stepName": "mock-stepname",
        "taskRecordDetails": {
          "executionDetails": {
            "command": "mock-command",
            "executionType": "SCRIPT"
          },
          "scope": "LOCAL"
        }
      }
    ]
  },
  "state": "INACTIVE",
  "status": "INACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.Runbook](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "INACTIVE",
  "operation": "mock-operation",
  "resourceId": "<ocid:1>",
  "runbookVersion": {
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
    "groups": [
      {
        "name": "mock-name",
        "type": "PARALLEL_TASK_GROUP"
      }
    ],
    "isLatest": true,
    "tasks": [
      {
        "stepName": "mock-stepname",
        "taskRecordDetails": {
          "executionDetails": {
            "command": "mock-command",
            "executionType": "SCRIPT"
          },
          "scope": "LOCAL"
        }
      }
    ]
  },
  "state": "INACTIVE",
  "status": "INACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATERUNBOOK",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "Runbook", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATERUNBOOK",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "Runbook", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETERUNBOOK",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "Runbook", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.Runbook, sdksvc.CreateRunbookDetails, sdksvc.UpdateRunbookDetails]{
		CollectionPath: "/20250228/runbooks", ItemPath: "/20250228/runbooks/<ocid:1>",
		CreatePath: "/20250228/runbooks", CreateMethod: http.MethodPost,
		UpdatePath: "/20250228/runbooks/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20250228/runbooks/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateRunbookDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateRunbookDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateRunbookDetails) error {
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
	sdkClient := RunbookSDKClients{fleetAppsManagementRunbooksClient: sdksvc.FleetAppsManagementRunbooksClient{BaseClient: session.BaseClient()}, fleetAppsManagementWorkRequestClient: sdksvc.FleetAppsManagementWorkRequestClient{BaseClient: session.BaseClient()}}
	manager := &RunbookServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newRunbookRuntimeHooks(manager, sdkClient)
	client := wrapRunbookGeneratedClient(hooks, defaultRunbookServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.Runbook](buildRunbookGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.Runbook]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.Runbook) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "INACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Runbook status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.Runbook) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.Runbook) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Runbook status = %+v", current.Status)
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
