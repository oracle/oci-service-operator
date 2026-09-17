/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package batchcontext

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/batch"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/batch/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationBatchContextCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.BatchContext](t, `
{
  "metadata": {"name": "mock-batchcontext", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "fleets": [],
  "jobPriorityConfigurations": [],
  "network": {
    "subnetId": "<ocid:required>"
  }
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-batchcontext")
	resource.Status = apiv1beta1.BatchContextStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateBatchContextDetails](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "fleets": [],
  "jobPriorityConfigurations": [],
  "network": {
    "subnetId": "<ocid:required>"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateBatchContextDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.BatchContext](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "fleets": [],
  "id": "<ocid:2>",
  "jobPriorityConfigurations": [],
  "key": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "loggingConfiguration": {},
  "network": {
    "subnetId": "<ocid:required>"
  },
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.BatchContext](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-updated",
  "fleets": [],
  "id": "<ocid:2>",
  "jobPriorityConfigurations": [],
  "key": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "loggingConfiguration": {},
  "network": {
    "subnetId": "<ocid:required>"
  },
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEBATCHCONTEXT",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "BatchContext", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEBATCHCONTEXT",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "BatchContext", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEBATCHCONTEXT",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "BatchContext", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.BatchContext, sdksvc.CreateBatchContextDetails, sdksvc.UpdateBatchContextDetails]{
		CollectionPath: "/20251031/batchContexts", ItemPath: "/20251031/batchContexts/<ocid:2>",
		CreatePath: "/20251031/batchContexts", CreateMethod: http.MethodPost,
		UpdatePath: "/20251031/batchContexts/<ocid:2>", UpdateMethod: http.MethodPut,
		DeletePath: "/20251031/batchContexts/<ocid:2>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateBatchContextDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateBatchContextDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateBatchContextDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20251031/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20251031/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20251031/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://batch.us-ashburn-1.oci.oraclecloud.com", BasePath: "20251031", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.BatchComputingClient{BaseClient: session.BaseClient()}
	manager := &BatchContextServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newBatchContextRuntimeHooks(manager, sdkClient)
	client := wrapBatchContextGeneratedClient(hooks, defaultBatchContextServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.BatchContext](buildBatchContextGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.BatchContext]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.BatchContext) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created BatchContext status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.BatchContext) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.BatchContext) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated BatchContext status = %+v", current.Status)
			}
			return nil
		},
		ValidateStable: func(*apiv1beta1.BatchContext) error {
			if got := responder.OperationCounts()[ocimock.OperationUpdate]; got != 1 {
				return fmt.Errorf("stable BatchContext update calls = %d, want 1", got)
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
