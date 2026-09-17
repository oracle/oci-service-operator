/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package awrhub

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

// Contract evidence: vendored OCI SDK, production service manager, reviewed runtime, and package-owned typed OCI fixtures.
func TestMockIntegrationAwrHubWorkRequestCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[opsiv1beta1.AwrHub](t, `
{
  "metadata": {
    "creationTimestamp": null,
    "name": "awrhub-sample",
    "namespace": "default",
    "uid": "uid-awrhub"
  },
  "spec": {
    "compartmentId": "<ocid:1>",
    "displayName": "awr-hub",
    "objectStorageBucketName": "awr-bucket",
    "operationsInsightsWarehouseId": "<ocid:2>"
  },
  "status": {}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-awrhub")
	resource.Status = opsiv1beta1.AwrHubStatus{}
	createRequest := ocimock.MustJSONFixture[opsisdk.CreateAwrHubDetails](t, `
{
  "compartmentId": "<ocid:1>",
  "displayName": "awr-hub",
  "objectStorageBucketName": "awr-bucket",
  "operationsInsightsWarehouseId": "<ocid:2>"
}
`)
	updateRequest := ocimock.MustJSONFixture[opsisdk.UpdateAwrHubDetails](t, `
{
  "displayName": "awr-hub-updated"
}
`)
	createdState := ocimock.MustOCIResponseFixture[opsisdk.AwrHub](t, `
{
  "awrMailboxUrl": "https://mailbox.example.invalid",
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "displayName": "awr-hub",
  "freeformTags": null,
  "hubDstTimezoneVersion": "42",
  "id": "<ocid:4>",
  "lifecycleDetails": "ready",
  "lifecycleState": "ACTIVE",
  "objectStorageBucketName": "awr-bucket",
  "operationsInsightsWarehouseId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": null,
  "timeUpdated": null
}
`)
	updatedState := ocimock.MustOCIResponseFixture[opsisdk.AwrHub](t, `
{
  "awrMailboxUrl": "https://mailbox.example.invalid",
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "displayName": "awr-hub-updated",
  "freeformTags": null,
  "hubDstTimezoneVersion": "42",
  "id": "<ocid:4>",
  "lifecycleDetails": "ready",
  "lifecycleState": "ACTIVE",
  "objectStorageBucketName": "awr-bucket",
  "operationsInsightsWarehouseId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": null,
  "timeUpdated": null
}
`)
	deletedState := ocimock.MustOCIResponseFixture[opsisdk.AwrHub](t, `
{
  "awrMailboxUrl": "https://mailbox.example.invalid",
  "compartmentId": "<ocid:1>",
  "definedTags": {},
  "displayName": "awr-hub-updated",
  "freeformTags": null,
  "hubDstTimezoneVersion": "42",
  "id": "<ocid:4>",
  "lifecycleDetails": "ready",
  "lifecycleState": "DELETED",
  "objectStorageBucketName": "awr-bucket",
  "operationsInsightsWarehouseId": "<ocid:2>",
  "systemTags": {
    "orcl-cloud": {
      "free-tier-retained": "false"
    }
  },
  "timeCreated": null,
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
      "entityType": "AwrHub",
      "identifier": "<ocid:4>"
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
      "entityType": "AwrHub",
      "identifier": "<ocid:4>"
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
      "entityType": "AwrHub",
      "identifier": "<ocid:4>"
    }
  ],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[opsisdk.AwrHub, opsisdk.CreateAwrHubDetails, opsisdk.UpdateAwrHubDetails]{
		CollectionPath: "/20200630/awrHubs", ItemPath: "/20200630/awrHubs/<ocid:4>",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, DeletedState: &deletedState,
		ListShape: ocimock.ListShapeItems, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: false,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ opsisdk.CreateAwrHubDetails) error {
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
	hooks := newAwrHubRuntimeHooksWithOCIClient(sdkClient)
	applyAwrHubRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapAwrHubGeneratedClient(hooks, defaultAwrHubServiceClient{ServiceClient: generatedruntime.NewServiceClient[*opsiv1beta1.AwrHub](buildAwrHubGeneratedRuntimeConfig(&AwrHubServiceManager{}, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opsiv1beta1.AwrHub]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opsiv1beta1.AwrHub) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" ||
				current.Status.DisplayName != resource.Spec.DisplayName || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created AwrHub status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opsiv1beta1.AwrHub) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "awr-hub-updated"
}`)
		},
		ValidateUpdated: func(current *opsiv1beta1.AwrHub) error {
			if !(current.Status.DisplayName == "awr-hub-updated") || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated AwrHub status = %+v", current.Status)
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
