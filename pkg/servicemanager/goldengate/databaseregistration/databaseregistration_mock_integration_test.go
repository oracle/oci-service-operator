/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package databaseregistration

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
func TestMockIntegrationDatabaseRegistrationCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.DatabaseRegistration](t, `
{
  "metadata": {"name": "mock-databaseregistration", "namespace": "default"},
  "spec": {
  "aliasName": "mock-aliasname",
  "compartmentId": "<ocid:required>",
  "databaseId": "<ocid:9>",
  "displayName": "mock-displayname-initial",
  "fqdn": "mock-fqdn",
  "ipAddress": "mock-ipaddress-updated",
  "keyId": "<ocid:9>",
  "password": "mock-password",
  "secretCompartmentId": "<ocid:9>",
  "subnetId": "<ocid:9>",
  "username": "mock-username",
  "vaultId": "<ocid:9>"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-databaseregistration")
	resource.Status = apiv1beta1.DatabaseRegistrationStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateDatabaseRegistrationDetails](t, `{
  "aliasName": "mock-aliasname",
  "compartmentId": "<ocid:required>",
  "databaseId": "<ocid:9>",
  "displayName": "mock-displayname-initial",
  "fqdn": "mock-fqdn",
  "ipAddress": "mock-ipaddress-updated",
  "keyId": "<ocid:9>",
  "password": "mock-password",
  "secretCompartmentId": "<ocid:9>",
  "subnetId": "<ocid:9>",
  "username": "mock-username",
  "vaultId": "<ocid:9>"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateDatabaseRegistrationDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.DatabaseRegistration](t, `{
  "aliasName": "mock-aliasname",
  "compartmentId": "<ocid:required>",
  "databaseId": "<ocid:9>",
  "displayName": "mock-displayname-initial",
  "fqdn": "mock-fqdn",
  "id": "<ocid:1>",
  "ipAddress": "mock-ipaddress-updated",
  "key": "<ocid:1>",
  "keyId": "<ocid:9>",
  "lifecycleState": "ACTIVE",
  "password": "mock-password",
  "resourceId": "<ocid:1>",
  "secretCompartmentId": "<ocid:9>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "subnetId": "<ocid:9>",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "username": "mock-username",
  "vaultId": "<ocid:9>"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.DatabaseRegistration](t, `{
  "aliasName": "mock-aliasname",
  "compartmentId": "<ocid:required>",
  "databaseId": "<ocid:9>",
  "displayName": "mock-displayname-updated",
  "fqdn": "mock-fqdn",
  "id": "<ocid:1>",
  "ipAddress": "mock-ipaddress-updated",
  "key": "<ocid:1>",
  "keyId": "<ocid:9>",
  "lifecycleState": "ACTIVE",
  "password": "mock-password",
  "resourceId": "<ocid:1>",
  "secretCompartmentId": "<ocid:9>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "subnetId": "<ocid:9>",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "username": "mock-username",
  "vaultId": "<ocid:9>"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEDATABASEREGISTRATION",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "DatabaseRegistration", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEDATABASEREGISTRATION",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "DatabaseRegistration", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEDATABASEREGISTRATION",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "DatabaseRegistration", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.DatabaseRegistration, sdksvc.CreateDatabaseRegistrationDetails, sdksvc.UpdateDatabaseRegistrationDetails]{
		CollectionPath: "/20200407/databaseRegistrations", ItemPath: "/20200407/databaseRegistrations/<ocid:1>",
		CreatePath: "/20200407/databaseRegistrations", CreateMethod: http.MethodPost,
		UpdatePath: "/20200407/databaseRegistrations/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20200407/databaseRegistrations/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateDatabaseRegistrationDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateDatabaseRegistrationDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateDatabaseRegistrationDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
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
	manager := &DatabaseRegistrationServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newDatabaseRegistrationRuntimeHooks(manager, sdkClient)
	client := wrapDatabaseRegistrationGeneratedClient(hooks, defaultDatabaseRegistrationServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.DatabaseRegistration](buildDatabaseRegistrationGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.DatabaseRegistration]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.DatabaseRegistration) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created DatabaseRegistration status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.DatabaseRegistration) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.DatabaseRegistration) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated DatabaseRegistration status = %+v", current.Status)
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
