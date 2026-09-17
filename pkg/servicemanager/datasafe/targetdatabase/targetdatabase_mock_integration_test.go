/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package targetdatabase

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/datasafe"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationTargetDatabaseCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.TargetDatabase](t, `
{
  "metadata": {"name": "mock-targetdatabase", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:1>",
  "databaseDetails": {
    "JsonData": "eyJhdXRvbm9tb3VzRGF0YWJhc2VJZCI6Im9jaWQxLmF1dG9ub21vdXNkYXRhYmFzZS5vYzEuLnN5bnRoZXRpYyIsImRhdGFiYXNlVHlwZSI6bnVsbCwiZGJTeXN0ZW1JZCI6bnVsbCwiaW5mcmFzdHJ1Y3R1cmVUeXBlIjoiQVVUT05PTU9VU19EQVRBQkFTRSIsImluc3RhbmNlSWQiOm51bGwsImlwQWRkcmVzc2VzIjpudWxsLCJqc29uRGF0YSI6bnVsbCwibGlzdGVuZXJQb3J0IjpudWxsLCJwbHVnZ2FibGVEYXRhYmFzZUlkIjpudWxsLCJzZXJ2aWNlTmFtZSI6bnVsbCwidm1DbHVzdGVySWQiOm51bGx9",
    "databaseType": "",
    "infrastructureType": "AUTONOMOUS_DATABASE"
  },
  "displayName": "mock-displayname-initial",
  "peerTargetDatabaseDetails": []
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-targetdatabase")
	resource.Status = apiv1beta1.TargetDatabaseStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateTargetDatabaseDetails](t, `{
  "compartmentId": "<ocid:1>",
  "databaseDetails": {
    "JsonData": "eyJhdXRvbm9tb3VzRGF0YWJhc2VJZCI6Im9jaWQxLmF1dG9ub21vdXNkYXRhYmFzZS5vYzEuLnN5bnRoZXRpYyIsImRhdGFiYXNlVHlwZSI6bnVsbCwiZGJTeXN0ZW1JZCI6bnVsbCwiaW5mcmFzdHJ1Y3R1cmVUeXBlIjoiQVVUT05PTU9VU19EQVRBQkFTRSIsImluc3RhbmNlSWQiOm51bGwsImlwQWRkcmVzc2VzIjpudWxsLCJqc29uRGF0YSI6bnVsbCwibGlzdGVuZXJQb3J0IjpudWxsLCJwbHVnZ2FibGVEYXRhYmFzZUlkIjpudWxsLCJzZXJ2aWNlTmFtZSI6bnVsbCwidm1DbHVzdGVySWQiOm51bGx9",
    "databaseType": "",
    "infrastructureType": "AUTONOMOUS_DATABASE"
  },
  "displayName": "mock-displayname-initial",
  "peerTargetDatabaseDetails": []
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateTargetDatabaseDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.TargetDatabase](t, `{
  "compartmentId": "<ocid:1>",
  "connectionOption": {},
  "credentials": {
    "password": "<redacted>",
    "userName": ""
  },
  "databaseDetails": {
    "JsonData": "eyJhdXRvbm9tb3VzRGF0YWJhc2VJZCI6Im9jaWQxLmF1dG9ub21vdXNkYXRhYmFzZS5vYzEuLnN5bnRoZXRpYyIsImRhdGFiYXNlVHlwZSI6bnVsbCwiZGJTeXN0ZW1JZCI6bnVsbCwiaW5mcmFzdHJ1Y3R1cmVUeXBlIjoiQVVUT05PTU9VU19EQVRBQkFTRSIsImluc3RhbmNlSWQiOm51bGwsImlwQWRkcmVzc2VzIjpudWxsLCJqc29uRGF0YSI6bnVsbCwibGlzdGVuZXJQb3J0IjpudWxsLCJwbHVnZ2FibGVEYXRhYmFzZUlkIjpudWxsLCJzZXJ2aWNlTmFtZSI6bnVsbCwidm1DbHVzdGVySWQiOm51bGx9",
    "databaseType": "",
    "infrastructureType": "AUTONOMOUS_DATABASE"
  },
  "displayName": "mock-displayname-initial",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "peerTargetDatabaseDetails": [],
  "state": "ACTIVE",
  "status": "ACTIVE",
  "tlsConfig": {
    "status": ""
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.TargetDatabase](t, `{
  "compartmentId": "<ocid:1>",
  "connectionOption": {},
  "credentials": {
    "password": "<redacted>",
    "userName": ""
  },
  "databaseDetails": {
    "JsonData": "eyJhdXRvbm9tb3VzRGF0YWJhc2VJZCI6Im9jaWQxLmF1dG9ub21vdXNkYXRhYmFzZS5vYzEuLnN5bnRoZXRpYyIsImRhdGFiYXNlVHlwZSI6bnVsbCwiZGJTeXN0ZW1JZCI6bnVsbCwiaW5mcmFzdHJ1Y3R1cmVUeXBlIjoiQVVUT05PTU9VU19EQVRBQkFTRSIsImluc3RhbmNlSWQiOm51bGwsImlwQWRkcmVzc2VzIjpudWxsLCJqc29uRGF0YSI6bnVsbCwibGlzdGVuZXJQb3J0IjpudWxsLCJwbHVnZ2FibGVEYXRhYmFzZUlkIjpudWxsLCJzZXJ2aWNlTmFtZSI6bnVsbCwidm1DbHVzdGVySWQiOm51bGx9",
    "databaseType": "",
    "infrastructureType": "AUTONOMOUS_DATABASE"
  },
  "displayName": "mock-displayname-updated",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "peerTargetDatabaseDetails": [],
  "state": "ACTIVE",
  "status": "ACTIVE",
  "tlsConfig": {
    "status": ""
  }
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATETARGETDATABASE",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "TargetDatabase", "identifier": "<ocid:3>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATETARGETDATABASE",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "TargetDatabase", "identifier": "<ocid:3>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETETARGETDATABASE",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "TargetDatabase", "identifier": "<ocid:3>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.TargetDatabase, sdksvc.CreateTargetDatabaseDetails, sdksvc.UpdateTargetDatabaseDetails]{
		CollectionPath: "/20181201/targetDatabases", ItemPath: "/20181201/targetDatabases/<ocid:3>",
		CreatePath: "/20181201/targetDatabases", CreateMethod: http.MethodPost,
		UpdatePath: "/20181201/targetDatabases/<ocid:3>", UpdateMethod: http.MethodPut,
		DeletePath: "/20181201/targetDatabases/<ocid:3>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateTargetDatabaseDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateTargetDatabaseDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateTargetDatabaseDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20181201/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://datasafe.us-ashburn-1.oci.oraclecloud.com", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.DataSafeClient{BaseClient: session.BaseClient()}
	manager := &TargetDatabaseServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newTargetDatabaseRuntimeHooks(manager, sdkClient)
	client := wrapTargetDatabaseGeneratedClient(hooks, defaultTargetDatabaseServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.TargetDatabase](buildTargetDatabaseGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.TargetDatabase]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.TargetDatabase) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created TargetDatabase status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.TargetDatabase) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.TargetDatabase) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated TargetDatabase status = %+v", current.Status)
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
