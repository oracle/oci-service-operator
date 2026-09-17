/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package oracledbgcpkeyring

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
func TestMockIntegrationOracleDbGcpKeyRingCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.OracleDbGcpKeyRing](t, `
{
  "metadata": {"name": "mock-oracledbgcpkeyring", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "gcpKeyRingId": "<ocid:9>",
  "location": "mock-location-updated",
  "oracleDbConnectorId": "<ocid:required>",
  "properties": {
    "mock": "updated"
  },
  "type": "mock-type-updated"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-oracledbgcpkeyring")
	resource.Status = apiv1beta1.OracleDbGcpKeyRingStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateOracleDbGcpKeyRingDetails](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "gcpKeyRingId": "<ocid:9>",
  "location": "mock-location-updated",
  "oracleDbConnectorId": "<ocid:required>",
  "properties": {
    "mock": "updated"
  },
  "type": "mock-type-updated"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateOracleDbGcpKeyRingDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.OracleDbGcpKeyRing](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "gcpKeyRingId": "<ocid:9>",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "location": "mock-location-updated",
  "oracleDbConnectorId": "<ocid:required>",
  "properties": {
    "mock": "updated"
  },
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "type": "mock-type-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.OracleDbGcpKeyRing](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "gcpKeyRingId": "<ocid:9>",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "location": "mock-location-updated",
  "oracleDbConnectorId": "<ocid:required>",
  "properties": {
    "mock": "updated"
  },
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "type": "mock-type-updated"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEORACLEDBGCPKEYRING",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "OracleDbGcpKeyRing", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEORACLEDBGCPKEYRING",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "OracleDbGcpKeyRing", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEORACLEDBGCPKEYRING",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "OracleDbGcpKeyRing", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.OracleDbGcpKeyRing, sdksvc.CreateOracleDbGcpKeyRingDetails, sdksvc.UpdateOracleDbGcpKeyRingDetails]{
		CollectionPath: "/20240501/oracleDbGcpKeyRing", ItemPath: "/20240501/oracleDbGcpKeyRing/<ocid:1>",
		CreatePath: "/20240501/oracleDbGcpKeyRing", CreateMethod: http.MethodPost,
		UpdatePath: "/20240501/oracleDbGcpKeyRing/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20240501/oracleDbGcpKeyRing/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateOracleDbGcpKeyRingDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateOracleDbGcpKeyRingDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateOracleDbGcpKeyRingDetails) error {
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
	sdkClient := OracleDbGcpKeyRingSDKClients{dbMulticloudGcpProviderClient: sdksvc.DbMulticloudGCPProviderClient{BaseClient: session.BaseClient()}, workRequestClient: sdksvc.WorkRequestClient{BaseClient: session.BaseClient()}}
	manager := &OracleDbGcpKeyRingServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newOracleDbGcpKeyRingRuntimeHooks(manager, sdkClient)
	client := wrapOracleDbGcpKeyRingGeneratedClient(hooks, defaultOracleDbGcpKeyRingServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.OracleDbGcpKeyRing](buildOracleDbGcpKeyRingGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.OracleDbGcpKeyRing]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.OracleDbGcpKeyRing) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created OracleDbGcpKeyRing status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.OracleDbGcpKeyRing) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.OracleDbGcpKeyRing) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated OracleDbGcpKeyRing status = %+v", current.Status)
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
