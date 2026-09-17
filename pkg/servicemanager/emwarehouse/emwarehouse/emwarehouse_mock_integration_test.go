/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package emwarehouse

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/emwarehouse"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/emwarehouse/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationEmWarehouseCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.EmWarehouse](t, `
{
  "metadata": {"name": "mock-emwarehouse", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "emBridgeId": "<ocid:required>",
  "freeformTags": {
    "mock": "initial"
  },
  "operationsInsightsWarehouseId": "<ocid:required>"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-emwarehouse")
	resource.Status = apiv1beta1.EmWarehouseStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateEmWarehouseDetails](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "emBridgeId": "<ocid:required>",
  "freeformTags": {
    "mock": "initial"
  },
  "operationsInsightsWarehouseId": "<ocid:required>"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateEmWarehouseDetails](t, `{
  "freeformTags": {
    "mock": "updated"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.EmWarehouse](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "emBridgeId": "<ocid:required>",
  "freeformTags": {
    "mock": "initial"
  },
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "operationsInsightsWarehouseId": "<ocid:required>",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.EmWarehouse](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "emBridgeId": "<ocid:required>",
  "freeformTags": {
    "mock": "updated"
  },
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "operationsInsightsWarehouseId": "<ocid:required>",
  "resourceId": "<ocid:1>",
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
  "operationType": "CREATEEMWAREHOUSE",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "EmWarehouse", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEEMWAREHOUSE",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "EmWarehouse", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEEMWAREHOUSE",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "EmWarehouse", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.EmWarehouse, sdksvc.CreateEmWarehouseDetails, sdksvc.UpdateEmWarehouseDetails]{
		CollectionPath: "/20180828/emWarehouses", ItemPath: "/20180828/emWarehouses/<ocid:1>",
		CreatePath: "/20180828/emWarehouses", CreateMethod: http.MethodPost,
		UpdatePath: "/20180828/emWarehouses/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20180828/emWarehouses/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateEmWarehouseDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateEmWarehouseDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateEmWarehouseDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20180828/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20180828/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20180828/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://operationsinsights.us-ashburn-1.oci.oraclecloud.com", BasePath: "20180828", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.EmWarehouseClient{BaseClient: session.BaseClient()}
	manager := &EmWarehouseServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newEmWarehouseRuntimeHooks(manager, sdkClient)
	client := wrapEmWarehouseGeneratedClient(hooks, defaultEmWarehouseServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.EmWarehouse](buildEmWarehouseGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.EmWarehouse]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.EmWarehouse) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.FreeformTags["mock"] != "initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created EmWarehouse status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.EmWarehouse) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "freeformTags": {
    "mock": "updated"
  }
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.EmWarehouse) error {
			if current.Status.FreeformTags["mock"] != "updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated EmWarehouse status = %+v", current.Status)
			}
			return nil
		},
		ValidateStable: func(*apiv1beta1.EmWarehouse) error {
			if got := responder.OperationCounts()[ocimock.OperationUpdate]; got != 1 {
				return fmt.Errorf("stable EmWarehouse update calls = %d, want 1", got)
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
