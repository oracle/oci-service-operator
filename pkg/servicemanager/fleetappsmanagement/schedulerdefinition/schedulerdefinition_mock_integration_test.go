/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package schedulerdefinition

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
func TestMockIntegrationSchedulerDefinitionCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.SchedulerDefinition](t, `
{
  "metadata": {"name": "mock-schedulerdefinition", "namespace": "default"},
  "spec": {
  "actionGroups": [
    {
      "fleetId": "<ocid:required>",
      "kind": "FLEET_USING_RUNBOOK",
      "runbookId": "<ocid:required>",
      "runbookVersionName": "mock-runbookversionname"
    }
  ],
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "schedule": {
    "duration": "mock-duration",
    "executionStartdate": "2026-01-02T03:04:05Z",
    "type": "CUSTOM"
  }
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-schedulerdefinition")
	resource.Status = apiv1beta1.SchedulerDefinitionStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateSchedulerDefinitionDetails](t, `{
  "actionGroups": [
    {
      "fleetId": "<ocid:required>",
      "kind": "FLEET_USING_RUNBOOK",
      "runbookId": "<ocid:required>",
      "runbookVersionName": "mock-runbookversionname"
    }
  ],
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "schedule": {
    "duration": "mock-duration",
    "executionStartdate": "2026-01-02T03:04:05Z",
    "type": "CUSTOM"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateSchedulerDefinitionDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.SchedulerDefinition](t, `{
  "actionGroups": [
    {
      "fleetId": "<ocid:required>",
      "kind": "FLEET_USING_RUNBOOK",
      "runbookId": "<ocid:required>",
      "runbookVersionName": "mock-runbookversionname"
    }
  ],
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "schedule": {
    "duration": "mock-duration",
    "executionStartdate": "2026-01-02T03:04:05Z",
    "type": "CUSTOM"
  },
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.SchedulerDefinition](t, `{
  "actionGroups": [
    {
      "fleetId": "<ocid:required>",
      "kind": "FLEET_USING_RUNBOOK",
      "runbookId": "<ocid:required>",
      "runbookVersionName": "mock-runbookversionname"
    }
  ],
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "schedule": {
    "duration": "mock-duration",
    "executionStartdate": "2026-01-02T03:04:05Z",
    "type": "CUSTOM"
  },
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATESCHEDULERDEFINITION",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "SchedulerDefinition", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATESCHEDULERDEFINITION",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "SchedulerDefinition", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.SchedulerDefinition, sdksvc.CreateSchedulerDefinitionDetails, sdksvc.UpdateSchedulerDefinitionDetails]{
		CollectionPath: "/20250228/schedulerDefinitions", ItemPath: "/20250228/schedulerDefinitions/<ocid:1>",
		CreatePath: "/20250228/schedulerDefinitions", CreateMethod: http.MethodPost,
		UpdatePath: "/20250228/schedulerDefinitions/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20250228/schedulerDefinitions/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateSchedulerDefinitionDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateSchedulerDefinitionDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateSchedulerDefinitionDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
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
	sdkClient := SchedulerDefinitionSDKClients{fleetAppsManagementOperationsClient: sdksvc.FleetAppsManagementOperationsClient{BaseClient: session.BaseClient()}, fleetAppsManagementWorkRequestClient: sdksvc.FleetAppsManagementWorkRequestClient{BaseClient: session.BaseClient()}}
	manager := &SchedulerDefinitionServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newSchedulerDefinitionRuntimeHooks(manager, sdkClient)
	client := wrapSchedulerDefinitionGeneratedClient(hooks, defaultSchedulerDefinitionServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.SchedulerDefinition](buildSchedulerDefinitionGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.SchedulerDefinition]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.SchedulerDefinition) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created SchedulerDefinition status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.SchedulerDefinition) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.SchedulerDefinition) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated SchedulerDefinition status = %+v", current.Status)
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
