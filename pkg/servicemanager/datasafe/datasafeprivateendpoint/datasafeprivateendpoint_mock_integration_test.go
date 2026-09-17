/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package datasafeprivateendpoint

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

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationDataSafePrivateEndpointWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[datasafev1beta1.DataSafePrivateEndpoint](t, `
{
  "metadata": {
    "creationTimestamp": null
  },
  "spec": {
    "compartmentId": "ocid1.compartment.oc1..datasafe",
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "description": "test private endpoint",
    "displayName": "datasafe-endpoint",
    "freeformTags": {
      "env": "test"
    },
    "nsgIds": [
      "ocid1.networksecuritygroup.oc1..datasafe"
    ],
    "privateEndpointIp": "10.0.0.10",
    "subnetId": "ocid1.subnet.oc1..datasafe",
    "vcnId": "ocid1.vcn.oc1..datasafe"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-datasafeprivateendpoint")
	resource.Status = datasafev1beta1.DataSafePrivateEndpointStatus{}
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateDataSafePrivateEndpointDetails](t, `
{
  "compartmentId": "ocid1.compartment.oc1..datasafe",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "test private endpoint",
  "displayName": "datasafe-endpoint",
  "freeformTags": {
    "env": "test"
  },
  "nsgIds": [
    "ocid1.networksecuritygroup.oc1..datasafe"
  ],
  "privateEndpointIp": "10.0.0.10",
  "subnetId": "ocid1.subnet.oc1..datasafe",
  "vcnId": "ocid1.vcn.oc1..datasafe"
}
`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateDataSafePrivateEndpointDetails](t, `
{
  "description": "updated private endpoint"
}
`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.DataSafePrivateEndpoint](t, `
{
  "compartmentId": "ocid1.compartment.oc1..datasafe",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "test private endpoint",
  "displayName": "datasafe-endpoint",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "nsgIds": [
    "ocid1.networksecuritygroup.oc1..datasafe"
  ],
  "privateEndpointIp": "10.0.0.10",
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "subnetId": "ocid1.subnet.oc1..datasafe",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "vcnId": "ocid1.vcn.oc1..datasafe"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.DataSafePrivateEndpoint](t, `
{
  "compartmentId": "ocid1.compartment.oc1..datasafe",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated private endpoint",
  "displayName": "datasafe-endpoint",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "nsgIds": [
    "ocid1.networksecuritygroup.oc1..datasafe"
  ],
  "privateEndpointIp": "10.0.0.10",
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "subnetId": "ocid1.subnet.oc1..datasafe",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "vcnId": "ocid1.vcn.oc1..datasafe"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[datasafesdk.DataSafePrivateEndpoint](t, `
{
  "compartmentId": "ocid1.compartment.oc1..datasafe",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated private endpoint",
  "displayName": "datasafe-endpoint",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "DELETED",
  "nsgIds": [
    "ocid1.networksecuritygroup.oc1..datasafe"
  ],
  "privateEndpointIp": "10.0.0.10",
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "subnetId": "ocid1.subnet.oc1..datasafe",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "vcnId": "ocid1.vcn.oc1..datasafe"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE_PRIVATE_ENDPOINT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "DataSafePrivateEndpoint",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE_PRIVATE_ENDPOINT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "DataSafePrivateEndpoint",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[datasafesdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETE_PRIVATE_ENDPOINT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "DataSafePrivateEndpoint",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[datasafesdk.DataSafePrivateEndpoint, datasafesdk.CreateDataSafePrivateEndpointDetails, datasafesdk.UpdateDataSafePrivateEndpointDetails]{
		CollectionPath: "/20181201/dataSafePrivateEndpoints", ItemPath: "/20181201/dataSafePrivateEndpoints/<ocid:1>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateDataSafePrivateEndpointDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://datasafe.mock.invalid", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	client := newDataSafePrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.DataSafePrivateEndpoint]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.DataSafePrivateEndpoint) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created DataSafePrivateEndpoint status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.DataSafePrivateEndpoint) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "description": "updated private endpoint"
}`)
		},
		ValidateUpdated: func(current *datasafev1beta1.DataSafePrivateEndpoint) error {
			if !(current.Status.Description == "updated private endpoint") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated DataSafePrivateEndpoint status = %+v", current.Status)
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
