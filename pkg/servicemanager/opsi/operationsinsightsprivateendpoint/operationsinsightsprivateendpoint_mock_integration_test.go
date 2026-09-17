/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package operationsinsightsprivateendpoint

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
func TestMockIntegrationOperationsInsightsPrivateEndpointWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[opsiv1beta1.OperationsInsightsPrivateEndpoint](t, `
{
  "metadata": {
    "creationTimestamp": null
  },
  "spec": {
    "compartmentId": "ocid1.compartment.oc1..opsi",
    "definedTags": {
      "Operations": {
        "CostCenter": "42"
      }
    },
    "description": "test private endpoint",
    "displayName": "opsi-endpoint",
    "freeformTags": {
      "env": "test"
    },
    "isUsedForRacDbs": false,
    "nsgIds": [
      "ocid1.networksecuritygroup.oc1..opsi"
    ],
    "subnetId": "ocid1.subnet.oc1..opsi",
    "vcnId": "ocid1.vcn.oc1..opsi"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-operationsinsightsprivateendpoint")
	resource.Status = opsiv1beta1.OperationsInsightsPrivateEndpointStatus{}
	createRequest := ocimock.MustJSONFixture[opsisdk.CreateOperationsInsightsPrivateEndpointDetails](t, `
{
  "compartmentId": "ocid1.compartment.oc1..opsi",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "test private endpoint",
  "displayName": "opsi-endpoint",
  "freeformTags": {
    "env": "test"
  },
  "isUsedForRacDbs": false,
  "nsgIds": [
    "ocid1.networksecuritygroup.oc1..opsi"
  ],
  "subnetId": "ocid1.subnet.oc1..opsi",
  "vcnId": "ocid1.vcn.oc1..opsi"
}
`)
	updateRequest := ocimock.MustJSONFixture[opsisdk.UpdateOperationsInsightsPrivateEndpointDetails](t, `
{
  "description": "updated private endpoint"
}
`)
	createdState := ocimock.MustOCIResponseFixture[opsisdk.OperationsInsightsPrivateEndpoint](t, `
{
  "compartmentId": "ocid1.compartment.oc1..opsi",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "test private endpoint",
  "displayName": "opsi-endpoint",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:1>",
  "isUsedForRacDbs": false,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "nsgIds": [
    "ocid1.networksecuritygroup.oc1..opsi"
  ],
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "subnetId": "ocid1.subnet.oc1..opsi",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "vcnId": "ocid1.vcn.oc1..opsi"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[opsisdk.OperationsInsightsPrivateEndpoint](t, `
{
  "compartmentId": "ocid1.compartment.oc1..opsi",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated private endpoint",
  "displayName": "opsi-endpoint",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:1>",
  "isUsedForRacDbs": false,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "nsgIds": [
    "ocid1.networksecuritygroup.oc1..opsi"
  ],
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "subnetId": "ocid1.subnet.oc1..opsi",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "vcnId": "ocid1.vcn.oc1..opsi"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[opsisdk.OperationsInsightsPrivateEndpoint](t, `
{
  "compartmentId": "ocid1.compartment.oc1..opsi",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "updated private endpoint",
  "displayName": "opsi-endpoint",
  "freeformTags": {
    "env": "test"
  },
  "id": "<ocid:1>",
  "isUsedForRacDbs": false,
  "key": "<ocid:1>",
  "lifecycleState": "DELETED",
  "nsgIds": [
    "ocid1.networksecuritygroup.oc1..opsi"
  ],
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "subnetId": "ocid1.subnet.oc1..opsi",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "vcnId": "ocid1.vcn.oc1..opsi"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE_PRIVATE_ENDPOINT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "OperationsInsightsPrivateEndpoint",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE_PRIVATE_ENDPOINT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "OperationsInsightsPrivateEndpoint",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETE_PRIVATE_ENDPOINT",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "OperationsInsightsPrivateEndpoint",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[opsisdk.OperationsInsightsPrivateEndpoint, opsisdk.CreateOperationsInsightsPrivateEndpointDetails, opsisdk.UpdateOperationsInsightsPrivateEndpointDetails]{
		CollectionPath: "/20200630/operationsInsightsPrivateEndpoints", ItemPath: "/20200630/operationsInsightsPrivateEndpoints/<ocid:1>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ opsisdk.CreateOperationsInsightsPrivateEndpointDetails) error {
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
	client := newOperationsInsightsPrivateEndpointServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opsiv1beta1.OperationsInsightsPrivateEndpoint]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opsiv1beta1.OperationsInsightsPrivateEndpoint) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created OperationsInsightsPrivateEndpoint status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opsiv1beta1.OperationsInsightsPrivateEndpoint) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "description": "updated private endpoint"
}`)
		},
		ValidateUpdated: func(current *opsiv1beta1.OperationsInsightsPrivateEndpoint) error {
			if !(current.Status.Description == "updated private endpoint") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated OperationsInsightsPrivateEndpoint status = %+v", current.Status)
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
