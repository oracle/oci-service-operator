/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package oracledbazurevault

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
func TestMockIntegrationOracleDbAzureVaultCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.OracleDbAzureVault](t, `
{
  "metadata": {"name": "mock-oracledbazurevault", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "oracleDbConnectorId": "<ocid:required>"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-oracledbazurevault")
	resource.Status = apiv1beta1.OracleDbAzureVaultStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateOracleDbAzureVaultDetails](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "oracleDbConnectorId": "<ocid:required>"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateOracleDbAzureVaultDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.OracleDbAzureVault](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "oracleDbConnectorId": "<ocid:required>",
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.OracleDbAzureVault](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "oracleDbConnectorId": "<ocid:required>",
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
  "operationType": "CREATEORACLEDBAZUREVAULT",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "OracleDbAzureVault", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEORACLEDBAZUREVAULT",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "OracleDbAzureVault", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEORACLEDBAZUREVAULT",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "OracleDbAzureVault", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.OracleDbAzureVault, sdksvc.CreateOracleDbAzureVaultDetails, sdksvc.UpdateOracleDbAzureVaultDetails]{
		CollectionPath: "/20240501/oracleDbAzureVault", ItemPath: "/20240501/oracleDbAzureVault/<ocid:1>",
		CreatePath: "/20240501/oracleDbAzureVault", CreateMethod: http.MethodPost,
		UpdatePath: "/20240501/oracleDbAzureVault/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20240501/oracleDbAzureVault/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateOracleDbAzureVaultDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateOracleDbAzureVaultDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateOracleDbAzureVaultDetails) error {
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
	sdkClient := OracleDbAzureVaultSDKClients{oracleDbAzureVaultClient: sdksvc.OracleDbAzureVaultClient{BaseClient: session.BaseClient()}, workRequestClient: sdksvc.WorkRequestClient{BaseClient: session.BaseClient()}}
	manager := &OracleDbAzureVaultServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newOracleDbAzureVaultRuntimeHooks(manager, sdkClient)
	client := wrapOracleDbAzureVaultGeneratedClient(hooks, defaultOracleDbAzureVaultServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.OracleDbAzureVault](buildOracleDbAzureVaultGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.OracleDbAzureVault]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.OracleDbAzureVault) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created OracleDbAzureVault status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.OracleDbAzureVault) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.OracleDbAzureVault) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated OracleDbAzureVault status = %+v", current.Status)
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
