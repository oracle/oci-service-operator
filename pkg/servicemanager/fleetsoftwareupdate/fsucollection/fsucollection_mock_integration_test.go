/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package fsucollection

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/fleetsoftwareupdate"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/fleetsoftwareupdate/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationFsuCollectionCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.FsuCollection](t, `
{
  "metadata": {"name": "mock-fsucollection", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "serviceType": "EXACS",
  "sourceMajorVersion": "DB_11204",
  "type": "DB"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-fsucollection")
	resource.Status = apiv1beta1.FsuCollectionStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateDbFsuCollectionDetails](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "serviceType": "EXACS",
  "sourceMajorVersion": "DB_11204"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateFsuCollectionDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.DbCollection](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "S_ACTIVE",
  "resourceId": "<ocid:1>",
  "serviceType": "EXACS",
  "sourceMajorVersion": "DB_11204",
  "state": "S_ACTIVE",
  "status": "S_ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.DbCollection](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "S_ACTIVE",
  "resourceId": "<ocid:1>",
  "serviceType": "EXACS",
  "sourceMajorVersion": "DB_11204",
  "state": "S_ACTIVE",
  "status": "S_ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEFSUCOLLECTION",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "FsuCollection", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEFSUCOLLECTION",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "FsuCollection", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEFSUCOLLECTION",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "FsuCollection", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.DbCollection, sdksvc.CreateDbFsuCollectionDetails, sdksvc.UpdateFsuCollectionDetails]{
		CollectionPath: "/20220528/fsuCollections", ItemPath: "/20220528/fsuCollections/<ocid:1>",
		CreatePath: "/20220528/fsuCollections", CreateMethod: http.MethodPost,
		UpdatePath: "/20220528/fsuCollections/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220528/fsuCollections/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},

		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateFsuCollectionDetails],
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
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "type", "DB", createRequest)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20220528/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20220528/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20220528/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://fleet-software-update.us-ashburn-1.oci.oraclecloud.com", BasePath: "20220528", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.FleetSoftwareUpdateClient{BaseClient: session.BaseClient()}
	manager := &FsuCollectionServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newFsuCollectionRuntimeHooks(manager, sdkClient)
	client := wrapFsuCollectionGeneratedClient(hooks, defaultFsuCollectionServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.FsuCollection](buildFsuCollectionGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.FsuCollection]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.FsuCollection) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "S_ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created FsuCollection status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.FsuCollection) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.FsuCollection) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated FsuCollection status = %+v", current.Status)
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
