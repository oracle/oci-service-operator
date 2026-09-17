/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package operationsinsightswarehouse

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationOperationsInsightsWarehouseWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[opsiv1beta1.OperationsInsightsWarehouse](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "opsi-warehouse",
    "namespace": "default",
    "uid": "uid-opsi-warehouse"
  },
  "spec": {
    "compartmentId": "<ocid:1>",
    "computeModel": "OCPU",
    "cpuAllocated": 2,
    "displayName": "opsi-warehouse",
    "storageAllocatedInGBs": 256
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-operationsinsightswarehouse")
	resource.Status = opsiv1beta1.OperationsInsightsWarehouseStatus{}
	createRequest := ocimock.MustJSONFixture[opsisdk.CreateOperationsInsightsWarehouseDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "computeModel": "OCPU",
  "cpuAllocated": 2,
  "displayName": "opsi-warehouse",
  "storageAllocatedInGBs": 256
}
`)
	updateRequest := ocimock.MustJSONFixture[opsisdk.UpdateOperationsInsightsWarehouseDetails](t, `
{
  "displayName": "opsi-warehouse-updated"
}
`)
	createdState := ocimock.MustOCIResponseFixture[opsisdk.OperationsInsightsWarehouse](t, `
{
  "compartmentId": "<ocid:1>",
  "computeModel": "OCPU",
  "cpuAllocated": 2,
  "cpuUsed": null,
  "definedTags": {},
  "displayName": "opsi-warehouse",
  "dynamicGroupId": null,
  "freeformTags": null,
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "operationsInsightsTenancyId": null,
  "storageAllocatedInGBs": 256,
  "storageUsedInGBs": null,
  "systemTags": null,
  "timeCreated": null,
  "timeLastWalletRotated": null,
  "timeUpdated": null
}
`)
	updatedState := ocimock.MustOCIResponseFixture[opsisdk.OperationsInsightsWarehouse](t, `
{
  "compartmentId": "<ocid:1>",
  "computeModel": "OCPU",
  "cpuAllocated": 2,
  "cpuUsed": null,
  "definedTags": {},
  "displayName": "opsi-warehouse-updated",
  "dynamicGroupId": null,
  "freeformTags": null,
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "ACTIVE",
  "operationsInsightsTenancyId": null,
  "storageAllocatedInGBs": 256,
  "storageUsedInGBs": null,
  "systemTags": null,
  "timeCreated": null,
  "timeLastWalletRotated": null,
  "timeUpdated": null
}
`)
	deletedState := ocimock.MustOCIResponseFixture[opsisdk.OperationsInsightsWarehouse](t, `
{
  "compartmentId": "<ocid:1>",
  "computeModel": "OCPU",
  "cpuAllocated": 2,
  "cpuUsed": null,
  "definedTags": {},
  "displayName": "opsi-warehouse-updated",
  "dynamicGroupId": null,
  "freeformTags": null,
  "id": "<ocid:3>",
  "lifecycleDetails": null,
  "lifecycleState": "DELETED",
  "operationsInsightsTenancyId": null,
  "storageAllocatedInGBs": 256,
  "storageUsedInGBs": null,
  "systemTags": null,
  "timeCreated": null,
  "timeLastWalletRotated": null,
  "timeUpdated": null
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "OperationsInsightsWarehouse",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "OperationsInsightsWarehouse",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETED",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "OperationsInsightsWarehouse",
      "identifier": "<ocid:3>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[opsisdk.OperationsInsightsWarehouse, opsisdk.CreateOperationsInsightsWarehouseDetails, opsisdk.UpdateOperationsInsightsWarehouseDetails]{
		CollectionPath: "/20200630/operationsInsightsWarehouses", ItemPath: "/20200630/operationsInsightsWarehouses/<ocid:3>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ opsisdk.CreateOperationsInsightsWarehouseDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200630/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://opsi.mock.invalid", BasePath: "20200630", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := opsisdk.OperationsInsightsClient{BaseClient: session.BaseClient()}
	client := newOperationsInsightsWarehouseServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opsiv1beta1.OperationsInsightsWarehouse]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opsiv1beta1.OperationsInsightsWarehouse) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created OperationsInsightsWarehouse status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opsiv1beta1.OperationsInsightsWarehouse) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "opsi-warehouse-updated"
}`)
		},
		ValidateUpdated: func(current *opsiv1beta1.OperationsInsightsWarehouse) error {
			if !(current.Status.DisplayName == "opsi-warehouse-updated") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated OperationsInsightsWarehouse status = %+v", current.Status)
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
