/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package replicationschedule

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
func TestMockIntegrationReplicationScheduleCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.ReplicationSchedule](t, `
{
  "metadata": {"name": "mock-replicationschedule", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "executionRecurrences": "FREQ=DAILY;INTERVAL=1"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-replicationschedule")
	resource.Status = apiv1beta1.ReplicationScheduleStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateReplicationScheduleDetails](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "executionRecurrences": "FREQ=DAILY;INTERVAL=1"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateReplicationScheduleDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.ReplicationSchedule](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "executionRecurrences": "FREQ=DAILY;INTERVAL=1",
  "id": "<ocid:2>",
  "key": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.ReplicationSchedule](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-updated",
  "executionRecurrences": "FREQ=DAILY;INTERVAL=1",
  "id": "<ocid:2>",
  "key": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEREPLICATIONSCHEDULE",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "ReplicationSchedule", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEREPLICATIONSCHEDULE",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "ReplicationSchedule", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEREPLICATIONSCHEDULE",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "ReplicationSchedule", "identifier": "<ocid:2>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.ReplicationSchedule, sdksvc.CreateReplicationScheduleDetails, sdksvc.UpdateReplicationScheduleDetails]{
		CollectionPath: "/20220919/replicationSchedules", ItemPath: "/20220919/replicationSchedules/<ocid:2>",
		CreatePath: "/20220919/replicationSchedules", CreateMethod: http.MethodPost,
		UpdatePath: "/20220919/replicationSchedules/<ocid:2>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220919/replicationSchedules/<ocid:2>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateReplicationScheduleDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateReplicationScheduleDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateReplicationScheduleDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
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
	session, err := ocimock.Open(ocimock.Options{Host: "https://migration.us-ashburn-1.oci.oraclecloud.com", BasePath: "20220919", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.MigrationClient{BaseClient: session.BaseClient()}
	manager := &ReplicationScheduleServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newReplicationScheduleRuntimeHooks(manager, sdkClient)
	client := wrapReplicationScheduleGeneratedClient(hooks, defaultReplicationScheduleServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.ReplicationSchedule](buildReplicationScheduleGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.ReplicationSchedule]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.ReplicationSchedule) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created ReplicationSchedule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.ReplicationSchedule) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.ReplicationSchedule) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated ReplicationSchedule status = %+v", current.Status)
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
