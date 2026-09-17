/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package oracledbazureblobcontainer

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/dbmulticloud"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/dbmulticloud/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationOracleDbAzureBlobContainerCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.OracleDbAzureBlobContainer](t, `
{
  "metadata": {"name": "mock-oracledbazureblobcontainer", "namespace": "default"},
  "spec": {
  "azureStorageAccountName": "mock-azurestorageaccountname",
  "azureStorageContainerName": "mock-azurestoragecontainername",
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-oracledbazureblobcontainer")
	resource.Status = apiv1beta1.OracleDbAzureBlobContainerStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateOracleDbAzureBlobContainerDetails](t, `{
  "azureStorageAccountName": "mock-azurestorageaccountname",
  "azureStorageContainerName": "mock-azurestoragecontainername",
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateOracleDbAzureBlobContainerDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.OracleDbAzureBlobContainer](t, `{
  "azureStorageAccountName": "mock-azurestorageaccountname",
  "azureStorageContainerName": "mock-azurestoragecontainername",
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.OracleDbAzureBlobContainer](t, `{
  "azureStorageAccountName": "mock-azurestorageaccountname",
  "azureStorageContainerName": "mock-azurestoragecontainername",
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
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
  "operationType": "CREATEORACLEDBAZUREBLOBCONTAINER",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "OracleDbAzureBlobContainer", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEORACLEDBAZUREBLOBCONTAINER",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "OracleDbAzureBlobContainer", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEORACLEDBAZUREBLOBCONTAINER",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "OracleDbAzureBlobContainer", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.OracleDbAzureBlobContainer, sdksvc.CreateOracleDbAzureBlobContainerDetails, sdksvc.UpdateOracleDbAzureBlobContainerDetails]{
		CollectionPath: "/20240501/oracleDbAzureBlobContainer", ItemPath: "/20240501/oracleDbAzureBlobContainer/<ocid:1>",
		CreatePath: "/20240501/oracleDbAzureBlobContainer", CreateMethod: http.MethodPost,
		UpdatePath: "/20240501/oracleDbAzureBlobContainer/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20240501/oracleDbAzureBlobContainer/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateOracleDbAzureBlobContainerDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateOracleDbAzureBlobContainerDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateOracleDbAzureBlobContainerDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20240501/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20240501/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20240501/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dbmulticloud.us-ashburn-1.oci.oraclecloud.com", BasePath: "20240501", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := OracleDbAzureBlobContainerSDKClients{oracleDbAzureBlobContainerClient: sdksvc.OracleDBAzureBlobContainerClient{BaseClient: session.BaseClient()}, workRequestClient: sdksvc.WorkRequestClient{BaseClient: session.BaseClient()}}
	manager := &OracleDbAzureBlobContainerServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newOracleDbAzureBlobContainerRuntimeHooks(manager, sdkClient)
	client := wrapOracleDbAzureBlobContainerGeneratedClient(hooks, defaultOracleDbAzureBlobContainerServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.OracleDbAzureBlobContainer](buildOracleDbAzureBlobContainerGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.OracleDbAzureBlobContainer]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.OracleDbAzureBlobContainer) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created OracleDbAzureBlobContainer status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.OracleDbAzureBlobContainer) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.OracleDbAzureBlobContainer) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated OracleDbAzureBlobContainer status = %+v", current.Status)
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
