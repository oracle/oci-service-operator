/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package connection

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/goldengate"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/goldengate/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationConnectionCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.Connection](t, `
{
  "metadata": {"name": "mock-connection", "namespace": "default"},
  "spec": {
  "clusterPlacementGroupId": "<ocid:9>",
  "compartmentId": "<ocid:required>",
  "connectionType": "POSTGRESQL",
  "databaseName": "mock-databasename",
  "dbSystemId": "<ocid:9>",
  "displayName": "mock-displayname-initial",
  "host": "mock-host-updated",
  "port": 2,
  "privateIp": "mock-privateip-updated",
  "securityProtocol": "PLAIN",
  "sslCa": "mock-sslca-updated",
  "sslCert": "mock-sslcert-updated",
  "sslCrl": "mock-sslcrl-updated",
  "sslKeySecretId": "<ocid:9>",
  "sslMode": "PREFER",
  "subscriptionId": "<ocid:9>",
  "technologyType": "OCI_POSTGRESQL",
  "username": "mock-username"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-connection")
	resource.Status = apiv1beta1.ConnectionStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreatePostgresqlConnectionDetails](t, `{
  "clusterPlacementGroupId": "<ocid:9>",
  "compartmentId": "<ocid:required>",
  "databaseName": "mock-databasename",
  "dbSystemId": "<ocid:9>",
  "displayName": "mock-displayname-initial",
  "host": "mock-host-updated",
  "port": 2,
  "privateIp": "mock-privateip-updated",
  "securityProtocol": "PLAIN",
  "sslCa": "mock-sslca-updated",
  "sslCert": "mock-sslcert-updated",
  "sslCrl": "mock-sslcrl-updated",
  "sslKeySecretId": "<ocid:9>",
  "sslMode": "PREFER",
  "subscriptionId": "<ocid:9>",
  "technologyType": "OCI_POSTGRESQL",
  "username": "mock-username"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdatePostgresqlConnectionDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.PostgresqlConnection](t, `{
  "clusterPlacementGroupId": "<ocid:9>",
  "compartmentId": "<ocid:required>",
  "databaseName": "mock-databasename",
  "dbSystemId": "<ocid:9>",
  "displayName": "mock-displayname-initial",
  "host": "mock-host-updated",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "port": 2,
  "privateIp": "mock-privateip-updated",
  "resourceId": "<ocid:1>",
  "securityProtocol": "PLAIN",
  "sslCa": "mock-sslca-updated",
  "sslCert": "mock-sslcert-updated",
  "sslCrl": "mock-sslcrl-updated",
  "sslKeySecretId": "<ocid:9>",
  "sslMode": "PREFER",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "subscriptionId": "<ocid:9>",
  "technologyType": "OCI_POSTGRESQL",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "username": "mock-username"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.PostgresqlConnection](t, `{
  "clusterPlacementGroupId": "<ocid:9>",
  "compartmentId": "<ocid:required>",
  "databaseName": "mock-databasename",
  "dbSystemId": "<ocid:9>",
  "displayName": "mock-displayname-updated",
  "host": "mock-host-updated",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "port": 2,
  "privateIp": "mock-privateip-updated",
  "resourceId": "<ocid:1>",
  "securityProtocol": "PLAIN",
  "sslCa": "mock-sslca-updated",
  "sslCert": "mock-sslcert-updated",
  "sslCrl": "mock-sslcrl-updated",
  "sslKeySecretId": "<ocid:9>",
  "sslMode": "PREFER",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "subscriptionId": "<ocid:9>",
  "technologyType": "OCI_POSTGRESQL",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "username": "mock-username"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATECONNECTION",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "Connection", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATECONNECTION",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "Connection", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETECONNECTION",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "Connection", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.PostgresqlConnection, sdksvc.CreatePostgresqlConnectionDetails, sdksvc.UpdatePostgresqlConnectionDetails]{
		CollectionPath: "/20200407/connections", ItemPath: "/20200407/connections/<ocid:1>",
		CreatePath: "/20200407/connections", CreateMethod: http.MethodPost,
		UpdatePath: "/20200407/connections/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20200407/connections/<ocid:1>", DeleteMethod: http.MethodDelete,
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
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "connectionType", "POSTGRESQL", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "connectionType", "POSTGRESQL", updateRequest)
		},
		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20200407/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20200407/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20200407/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://goldengate.us-ashburn-1.oci.oraclecloud.com", BasePath: "20200407", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.GoldenGateClient{BaseClient: session.BaseClient()}
	manager := &ConnectionServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newConnectionRuntimeHooks(manager, sdkClient)
	client := wrapConnectionGeneratedClient(hooks, defaultConnectionServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.Connection](buildConnectionGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.Connection]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.Connection) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Connection status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.Connection) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.Connection) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Connection status = %+v", current.Status)
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
