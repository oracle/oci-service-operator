/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package targetasset

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/cloudmigrations"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/cloudmigrations/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationTargetAssetCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.TargetAsset](t, `
{
  "metadata": {"name": "mock-targetasset", "namespace": "default"},
  "spec": {
  "isExcludedFromExecution": false,
  "migrationPlanId": "<ocid:required>",
  "type": "OLVM_INSTANCE"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-targetasset")
	resource.Status = apiv1beta1.TargetAssetStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateOlvmTargetAssetDetails](t, `{
  "isExcludedFromExecution": false,
  "migrationPlanId": "<ocid:required>"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateOlvmTargetAssetDetails](t, `{
  "isExcludedFromExecution": true
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.OlvmTargetAsset](t, `{
  "id": "<ocid:1>",
  "isExcludedFromExecution": false,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "migrationPlanId": "<ocid:required>",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.OlvmTargetAsset](t, `{
  "id": "<ocid:1>",
  "isExcludedFromExecution": true,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "migrationPlanId": "<ocid:required>",
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
  "operationType": "CREATETARGETASSET",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "TargetAsset", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATETARGETASSET",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "TargetAsset", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETETARGETASSET",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "TargetAsset", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.OlvmTargetAsset, sdksvc.CreateOlvmTargetAssetDetails, sdksvc.UpdateOlvmTargetAssetDetails]{
		CollectionPath: "/20220919/targetAssets", ItemPath: "/20220919/targetAssets/<ocid:1>",
		CreatePath: "/20220919/targetAssets", CreateMethod: http.MethodPost,
		UpdatePath: "/20220919/targetAssets/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220919/targetAssets/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},

		CreatedState: &createdState,

		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "type", "OLVM_INSTANCE", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "type", "OLVM_INSTANCE", updateRequest)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20220919/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20220919/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20220919/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudmigration.us-ashburn-1.oci.oraclecloud.com", BasePath: "20220919", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.MigrationClient{BaseClient: session.BaseClient()}
	manager := &TargetAssetServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newTargetAssetRuntimeHooks(manager, sdkClient)
	client := wrapTargetAssetGeneratedClient(hooks, defaultTargetAssetServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.TargetAsset](buildTargetAssetGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.TargetAsset]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.TargetAsset) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.IsExcludedFromExecution != false || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created TargetAsset status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.TargetAsset) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "isExcludedFromExecution": true
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.TargetAsset) error {
			if current.Status.IsExcludedFromExecution != true || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated TargetAsset status = %+v", current.Status)
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
