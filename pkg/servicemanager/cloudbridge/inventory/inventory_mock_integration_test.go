/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package inventory

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/cloudbridge"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/cloudbridge/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationInventoryCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.Inventory](t, `
{
  "metadata": {"name": "mock-inventory", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-inventory")
	resource.Status = apiv1beta1.InventoryStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateInventoryDetails](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateInventoryDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.Inventory](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:2>",
  "key": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.Inventory](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:2>",
  "key": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEINVENTORY",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "Inventory", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEINVENTORY",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "Inventory", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.Inventory, sdksvc.CreateInventoryDetails, sdksvc.UpdateInventoryDetails]{
		CollectionPath: "/20220509/inventories", ItemPath: "/20220509/inventories/<ocid:2>",
		CreatePath: "/20220509/inventories", CreateMethod: http.MethodPost,
		UpdatePath: "/20220509/inventories/<ocid:2>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220509/inventories/<ocid:2>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateInventoryDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateInventoryDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 200, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateInventoryDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20220509/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20220509/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudbridge.us-ashburn-1.oci.oraclecloud.com", BasePath: "20220509", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := InventorySDKClients{inventoryClient: sdksvc.InventoryClient{BaseClient: session.BaseClient()}, commonClient: sdksvc.CommonClient{BaseClient: session.BaseClient()}}
	manager := &InventoryServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newInventoryRuntimeHooks(manager, sdkClient)
	client := wrapInventoryGeneratedClient(hooks, defaultInventoryServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.Inventory](buildInventoryGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.Inventory]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.Inventory) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Inventory status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.Inventory) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.Inventory) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Inventory status = %+v", current.Status)
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
