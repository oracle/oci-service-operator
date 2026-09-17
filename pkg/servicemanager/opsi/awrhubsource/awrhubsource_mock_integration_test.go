/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package awrhubsource

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
func TestMockIntegrationAwrHubSourceWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[opsiv1beta1.AwrHubSource](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "source-a",
    "namespace": "default",
    "uid": "awrhubsource-uid"
  },
  "spec": {
    "associatedResourceId": "db-1",
    "awrHubId": "awr-hub-1",
    "compartmentId": "compartment-1",
    "definedTags": {
      "ns": {
        "key": "value"
      }
    },
    "freeformTags": {
      "env": "dev"
    },
    "name": "source-a",
    "type": "EXTERNAL_PDB"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-awrhubsource")
	resource.Status = opsiv1beta1.AwrHubSourceStatus{}
	createRequest := ocimock.MustJSONFixture[opsisdk.CreateAwrHubSourceDetails](t, `
{
  "associatedResourceId": "db-1",
  "awrHubId": "awr-hub-1",
  "compartmentId": "compartment-1",
  "definedTags": {
    "ns": {
      "key": "value"
    }
  },
  "freeformTags": {
    "env": "dev"
  },
  "name": "source-a",
  "type": "EXTERNAL_PDB"
}
`)
	updateRequest := ocimock.MustJSONFixture[opsisdk.UpdateAwrHubSourceDetails](t, `
{
  "freeformTags": {
    "env": "dev",
    "mock": "updated"
  }
}
`)
	createdState := ocimock.MustOCIResponseFixture[opsisdk.AwrHubSource](t, `
{
  "associatedResourceId": "db-1",
  "awrHubId": "awr-hub-1",
  "compartmentId": "compartment-1",
  "definedTags": {
    "ns": {
      "key": "value"
    }
  },
  "freeformTags": {
    "env": "dev"
  },
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "name": "source-a",
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "type": "EXTERNAL_PDB"
}
`)
	updatedState := ocimock.MustOCIResponseFixture[opsisdk.AwrHubSource](t, `
{
  "associatedResourceId": "db-1",
  "awrHubId": "awr-hub-1",
  "compartmentId": "compartment-1",
  "definedTags": {
    "ns": {
      "key": "value"
    }
  },
  "freeformTags": {
    "mock": "updated"
  },
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "name": "source-a",
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "type": "EXTERNAL_PDB"
}
`)
	deletedState := ocimock.MustOCIResponseFixture[opsisdk.AwrHubSource](t, `
{
  "associatedResourceId": "db-1",
  "awrHubId": "awr-hub-1",
  "compartmentId": "compartment-1",
  "definedTags": {
    "ns": {
      "key": "value"
    }
  },
  "freeformTags": {
    "mock": "updated"
  },
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "DELETED",
  "name": "source-a",
  "resourceId": "<ocid:1>",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "type": "EXTERNAL_PDB"
}
`)
	createWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATE_AWRHUB_SOURCE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "CREATED",
      "entityType": "AwrHubSource",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATE_AWRHUB_SOURCE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "UPDATED",
      "entityType": "AwrHubSource",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[opsisdk.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETE_AWRHUB_SOURCE",
  "percentComplete": 100,
  "resources": [
    {
      "actionType": "DELETED",
      "entityType": "AwrHubSource",
      "identifier": "<ocid:1>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[opsisdk.AwrHubSource, opsisdk.CreateAwrHubSourceDetails, opsisdk.UpdateAwrHubSourceDetails]{
		CollectionPath: "/20200630/awrHubSources", ItemPath: "/20200630/awrHubSources/<ocid:1>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ opsisdk.CreateAwrHubSourceDetails) error {
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
	client := newAwrHubSourceServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opsiv1beta1.AwrHubSource]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opsiv1beta1.AwrHubSource) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.Name != resource.Spec.Name || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created AwrHubSource status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opsiv1beta1.AwrHubSource) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "freeformTags": {
    "mock": "updated"
  }
}`)
		},
		ValidateUpdated: func(current *opsiv1beta1.AwrHubSource) error {
			if !(current.Status.FreeformTags["mock"] == "updated") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated AwrHubSource status = %+v", current.Status)
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
