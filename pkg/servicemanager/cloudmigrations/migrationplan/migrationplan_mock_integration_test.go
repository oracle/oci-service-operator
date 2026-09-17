/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package migrationplan

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
func TestMockIntegrationMigrationPlanCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.MigrationPlan](t, `
{
  "metadata": {"name": "mock-migrationplan", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "migrationId": "<ocid:2>",
  "sourceMigrationPlanId": "<ocid:9>",
  "strategies": [],
  "targetEnvironments": []
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-migrationplan")
	resource.Status = apiv1beta1.MigrationPlanStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateMigrationPlanDetails](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "migrationId": "<ocid:2>",
  "sourceMigrationPlanId": "<ocid:9>",
  "strategies": [],
  "targetEnvironments": []
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateMigrationPlanDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.MigrationPlan](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "migrationId": "<ocid:2>",
  "sourceMigrationPlanId": "<ocid:9>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "strategies": [],
  "targetEnvironments": []
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.MigrationPlan](t, `{
  "compartmentId": "<ocid:1>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "migrationId": "<ocid:2>",
  "sourceMigrationPlanId": "<ocid:9>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "strategies": [],
  "targetEnvironments": []
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEMIGRATIONPLAN",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "MigrationPlan", "identifier": "<ocid:3>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEMIGRATIONPLAN",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "MigrationPlan", "identifier": "<ocid:3>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEMIGRATIONPLAN",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "MigrationPlan", "identifier": "<ocid:3>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.MigrationPlan, sdksvc.CreateMigrationPlanDetails, sdksvc.UpdateMigrationPlanDetails]{
		CollectionPath: "/20220919/migrationPlans", ItemPath: "/20220919/migrationPlans/<ocid:3>",
		CreatePath: "/20220919/migrationPlans", CreateMethod: http.MethodPost,
		UpdatePath: "/20220919/migrationPlans/<ocid:3>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220919/migrationPlans/<ocid:3>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateMigrationPlanDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateMigrationPlanDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateMigrationPlanDetails) error {
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
	manager := &MigrationPlanServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newMigrationPlanRuntimeHooks(manager, sdkClient)
	client := wrapMigrationPlanGeneratedClient(hooks, defaultMigrationPlanServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.MigrationPlan](buildMigrationPlanGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.MigrationPlan]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.MigrationPlan) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created MigrationPlan status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.MigrationPlan) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.MigrationPlan) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated MigrationPlan status = %+v", current.Status)
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
