/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package maintenancewindow

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
func TestMockIntegrationMaintenanceWindowCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.MaintenanceWindow](t, `
{
  "metadata": {"name": "mock-maintenancewindow", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "duration": "mock-duration",
  "timeScheduleStart": "2026-01-02T03:04:05Z"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-maintenancewindow")
	resource.Status = apiv1beta1.MaintenanceWindowStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateMaintenanceWindowDetails](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "duration": "mock-duration",
  "timeScheduleStart": "2026-01-02T03:04:05Z"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateMaintenanceWindowDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.MaintenanceWindow](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "duration": "mock-duration",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.MaintenanceWindow](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "duration": "mock-duration",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEMAINTENANCEWINDOW",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "MaintenanceWindow", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEMAINTENANCEWINDOW",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "MaintenanceWindow", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.MaintenanceWindow, sdksvc.CreateMaintenanceWindowDetails, sdksvc.UpdateMaintenanceWindowDetails]{
		CollectionPath: "/20250228/maintenanceWindows", ItemPath: "/20250228/maintenanceWindows/<ocid:1>",
		CreatePath: "/20250228/maintenanceWindows", CreateMethod: http.MethodPost,
		UpdatePath: "/20250228/maintenanceWindows/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20250228/maintenanceWindows/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateMaintenanceWindowDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateMaintenanceWindowDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateMaintenanceWindowDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
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
	sdkClient := MaintenanceWindowSDKClients{fleetAppsManagementMaintenanceWindowClient: sdksvc.FleetAppsManagementMaintenanceWindowClient{BaseClient: session.BaseClient()}, fleetAppsManagementWorkRequestClient: sdksvc.FleetAppsManagementWorkRequestClient{BaseClient: session.BaseClient()}}
	manager := &MaintenanceWindowServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newMaintenanceWindowRuntimeHooks(manager, sdkClient)
	client := wrapMaintenanceWindowGeneratedClient(hooks, defaultMaintenanceWindowServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.MaintenanceWindow](buildMaintenanceWindowGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.MaintenanceWindow]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.MaintenanceWindow) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created MaintenanceWindow status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.MaintenanceWindow) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.MaintenanceWindow) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated MaintenanceWindow status = %+v", current.Status)
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
