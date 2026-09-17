/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package operationsinsightswarehouseuser

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
func TestMockIntegrationOperationsInsightsWarehouseUserWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[opsiv1beta1.OperationsInsightsWarehouseUser](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "warehouse-user",
    "namespace": "default",
    "uid": "warehouse-user-uid"
  },
  "spec": {
    "compartmentId": "ocid1.compartment.oc1..test",
    "connectionPassword": "secret-password",
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "freeformTags": {
      "env": "test"
    },
    "isAwrDataAccess": true,
    "isEmDataAccess": true,
    "isOpsiDataAccess": true,
    "name": "warehouse_user",
    "operationsInsightsWarehouseId": "ocid1.opsiwarehouse.oc1..test"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-operationsinsightswarehouseuser")
	resource.Status = opsiv1beta1.OperationsInsightsWarehouseUserStatus{}
	createRequest := ocimock.MustJSONFixture[opsisdk.CreateOperationsInsightsWarehouseUserDetails](t, `
{
  "compartmentId": "ocid1.compartment.oc1..test",
  "connectionPassword": "secret-password",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "test"
  },
  "isAwrDataAccess": true,
  "isEmDataAccess": true,
  "isOpsiDataAccess": true,
  "name": "warehouse_user",
  "operationsInsightsWarehouseId": "ocid1.opsiwarehouse.oc1..test"
}
`)
	updateRequest := ocimock.MustJSONFixture[opsisdk.UpdateOperationsInsightsWarehouseUserDetails](t, `
{
  "freeformTags": {
    "env": "test",
    "mock": "updated"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[opsisdk.OperationsInsightsWarehouseUser](t, `
{
  "compartmentId": "ocid1.compartment.oc1..test",
  "connectionPassword": "secret-password",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:1>",
  "isAwrDataAccess": true,
  "isEmDataAccess": true,
  "isOpsiDataAccess": true,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "name": "warehouse_user",
  "operationsInsightsWarehouseId": "ocid1.opsiwarehouse.oc1..test",
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[opsisdk.OperationsInsightsWarehouseUser](t, `
{
  "compartmentId": "ocid1.compartment.oc1..test",
  "connectionPassword": "secret-password",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "mock": "updated"
  },
  "id": "<ocid:1>",
  "isAwrDataAccess": true,
  "isEmDataAccess": true,
  "isOpsiDataAccess": true,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "name": "warehouse_user",
  "operationsInsightsWarehouseId": "ocid1.opsiwarehouse.oc1..test",
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[opsisdk.OperationsInsightsWarehouseUser](t, `
{
  "compartmentId": "ocid1.compartment.oc1..test",
  "connectionPassword": "secret-password",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "mock": "updated"
  },
  "id": "<ocid:1>",
  "isAwrDataAccess": true,
  "isEmDataAccess": true,
  "isOpsiDataAccess": true,
  "key": "<ocid:1>",
  "lifecycleState": "DELETED",
  "name": "warehouse_user",
  "operationsInsightsWarehouseId": "ocid1.opsiwarehouse.oc1..test",
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
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
      "entityType": "OperationsInsightsWarehouseUser",
      "identifier": "<ocid:1>"
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
      "entityType": "OperationsInsightsWarehouseUser",
      "identifier": "<ocid:1>"
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
      "entityType": "OperationsInsightsWarehouseUser",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[opsisdk.OperationsInsightsWarehouseUser, opsisdk.CreateOperationsInsightsWarehouseUserDetails, opsisdk.UpdateOperationsInsightsWarehouseUserDetails]{
		CollectionPath: "/20200630/operationsInsightsWarehouseUsers", ItemPath: "/20200630/operationsInsightsWarehouseUsers/<ocid:1>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ opsisdk.CreateOperationsInsightsWarehouseUserDetails) error {
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
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}
	manager := &OperationsInsightsWarehouseUserServiceManager{Log: log}
	hooks := newOperationsInsightsWarehouseUserRuntimeHooksWithOCIClient(sdkClient)
	applyOperationsInsightsWarehouseUserRuntimeHooks(&hooks, sdkClient, nil, log)
	client := wrapOperationsInsightsWarehouseUserGeneratedClient(hooks, defaultOperationsInsightsWarehouseUserServiceClient{ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.OperationsInsightsWarehouseUser](buildOperationsInsightsWarehouseUserGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opsiv1beta1.OperationsInsightsWarehouseUser]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opsiv1beta1.OperationsInsightsWarehouseUser) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.Name != resource.Spec.Name || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created OperationsInsightsWarehouseUser status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opsiv1beta1.OperationsInsightsWarehouseUser) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "freeformTags": {
    "mock": "updated"
  }
}`)
		},
		ValidateUpdated: func(current *opsiv1beta1.OperationsInsightsWarehouseUser) error {
			if !(current.Status.FreeformTags["mock"] == "updated") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated OperationsInsightsWarehouseUser status = %+v", current.Status)
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
