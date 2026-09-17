/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package migrationasset

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
func TestMockIntegrationMigrationAssetCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.MigrationAsset](t, `
{
  "metadata": {"name": "mock-migrationasset", "namespace": "default"},
  "spec": {
  "availabilityDomain": "AD-1",
  "displayName": "mock-displayname-initial",
  "inventoryAssetId": "<ocid:1>",
  "migrationId": "<ocid:2>",
  "replicationCompartmentId": "<ocid:3>",
  "snapShotBucketName": "osok-mock"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-migrationasset")
	resource.Status = apiv1beta1.MigrationAssetStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateMigrationAssetDetails](t, `{
  "availabilityDomain": "AD-1",
  "displayName": "mock-displayname-initial",
  "inventoryAssetId": "<ocid:1>",
  "migrationId": "<ocid:2>",
  "replicationCompartmentId": "<ocid:3>",
  "snapShotBucketName": "osok-mock"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateMigrationAssetDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.MigrationAsset](t, `{
  "availabilityDomain": "AD-1",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:4>",
  "inventoryAssetId": "<ocid:1>",
  "key": "<ocid:4>",
  "lifecycleState": "ACTIVE",
  "migrationId": "<ocid:2>",
  "replicationCompartmentId": "<ocid:3>",
  "snapShotBucketName": "osok-mock",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.MigrationAsset](t, `{
  "availabilityDomain": "AD-1",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:4>",
  "inventoryAssetId": "<ocid:1>",
  "key": "<ocid:4>",
  "lifecycleState": "ACTIVE",
  "migrationId": "<ocid:2>",
  "replicationCompartmentId": "<ocid:3>",
  "snapShotBucketName": "osok-mock",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEMIGRATIONASSET",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "MigrationAsset", "identifier": "<ocid:4>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEMIGRATIONASSET",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "MigrationAsset", "identifier": "<ocid:4>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.MigrationAsset, sdksvc.CreateMigrationAssetDetails, sdksvc.UpdateMigrationAssetDetails]{
		CollectionPath: "/20220919/migrationAssets", ItemPath: "/20220919/migrationAssets/<ocid:4>",
		CreatePath: "/20220919/migrationAssets", CreateMethod: http.MethodPost,
		UpdatePath: "/20220919/migrationAssets/<ocid:4>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220919/migrationAssets/<ocid:4>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateMigrationAssetDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateMigrationAssetDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 200, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateMigrationAssetDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20220919/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20220919/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://migration.us-ashburn-1.oci.oraclecloud.com", BasePath: "20220919", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.MigrationClient{BaseClient: session.BaseClient()}
	manager := &MigrationAssetServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newMigrationAssetRuntimeHooks(manager, sdkClient)
	client := wrapMigrationAssetGeneratedClient(hooks, defaultMigrationAssetServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.MigrationAsset](buildMigrationAssetGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.MigrationAsset]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.MigrationAsset) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created MigrationAsset status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.MigrationAsset) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.MigrationAsset) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated MigrationAsset status = %+v", current.Status)
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
