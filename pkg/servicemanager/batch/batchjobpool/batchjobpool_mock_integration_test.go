/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package batchjobpool

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
func TestMockIntegrationBatchJobPoolCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.BatchJobPool](t, `
{
  "metadata": {"name": "mock-batchjobpool", "namespace": "default"},
  "spec": {
  "batchContextId": "<ocid:1>",
  "compartmentId": "<ocid:2>",
  "displayName": "mock-displayname-initial"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-batchjobpool")
	resource.Status = apiv1beta1.BatchJobPoolStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateBatchJobPoolDetails](t, `{
  "batchContextId": "<ocid:1>",
  "compartmentId": "<ocid:2>",
  "displayName": "mock-displayname-initial"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateBatchJobPoolDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.BatchJobPool](t, `{
  "batchContextId": "<ocid:1>",
  "compartmentId": "<ocid:2>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.BatchJobPool](t, `{
  "batchContextId": "<ocid:1>",
  "compartmentId": "<ocid:2>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEBATCHJOBPOOL",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "BatchJobPool", "identifier": "<ocid:3>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.BatchJobPool, sdksvc.CreateBatchJobPoolDetails, sdksvc.UpdateBatchJobPoolDetails]{
		CollectionPath: "/20251031/batchJobPools", ItemPath: "/20251031/batchJobPools/<ocid:3>",
		CreatePath: "/20251031/batchJobPools", CreateMethod: http.MethodPost,
		UpdatePath: "/20251031/batchJobPools/<ocid:3>", UpdateMethod: http.MethodPut,
		DeletePath: "/20251031/batchJobPools/<ocid:3>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateBatchJobPoolDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateBatchJobPoolDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateBatchJobPoolDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20251031/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
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
	manager := &BatchJobPoolServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newBatchJobPoolRuntimeHooks(manager, sdkClient)
	client := wrapBatchJobPoolGeneratedClient(hooks, defaultBatchJobPoolServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.BatchJobPool](buildBatchJobPoolGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.BatchJobPool]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationUpdate},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.BatchJobPool) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created BatchJobPool status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.BatchJobPool) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.BatchJobPool) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated BatchJobPool status = %+v", current.Status)
			}
			return nil
		},
		ValidateStable: func(*apiv1beta1.BatchJobPool) error {
			if got := responder.OperationCounts()[ocimock.OperationUpdate]; got != 1 {
				return fmt.Errorf("stable BatchJobPool update calls = %d, want 1", got)
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
