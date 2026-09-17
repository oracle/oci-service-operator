/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package fsucycle

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
func TestMockIntegrationFsuCycleCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.FsuCycle](t, `
{
  "metadata": {"name": "mock-fsucycle", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "fsuCollectionId": "<ocid:required>",
  "goalVersionDetails": {
    "components": [
      {
        "componentType": "GUEST_OS",
        "goalVersionDetails": {
          "goalType": "GUEST_OS_ORACLE_IMAGE",
          "goalVersion": "mock-goalversion"
        }
      }
    ],
    "type": "EXADB_STACK"
  },
  "type": "PATCH"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-fsucycle")
	resource.Status = apiv1beta1.FsuCycleStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreatePatchFsuCycle](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "fsuCollectionId": "<ocid:required>",
  "goalVersionDetails": {
    "components": [
      {
        "componentType": "GUEST_OS",
        "goalVersionDetails": {
          "goalType": "GUEST_OS_ORACLE_IMAGE",
          "goalVersion": "mock-goalversion"
        }
      }
    ],
    "type": "EXADB_STACK"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdatePatchFsuCycle](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.PatchFsuCycle](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "fsuCollectionId": "<ocid:required>",
  "goalVersionDetails": {
    "components": [
      {
        "componentType": "GUEST_OS",
        "goalVersionDetails": {
          "goalType": "GUEST_OS_ORACLE_IMAGE",
          "goalVersion": "mock-goalversion"
        }
      }
    ],
    "type": "EXADB_STACK"
  },
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "S_ACTIVE",
  "resourceId": "<ocid:1>",
  "state": "S_ACTIVE",
  "status": "S_ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.PatchFsuCycle](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "fsuCollectionId": "<ocid:required>",
  "goalVersionDetails": {
    "components": [
      {
        "componentType": "GUEST_OS",
        "goalVersionDetails": {
          "goalType": "GUEST_OS_ORACLE_IMAGE",
          "goalVersion": "mock-goalversion"
        }
      }
    ],
    "type": "EXADB_STACK"
  },
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "S_ACTIVE",
  "resourceId": "<ocid:1>",
  "state": "S_ACTIVE",
  "status": "S_ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEFSUCYCLE",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "FsuCycle", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEFSUCYCLE",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "FsuCycle", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEFSUCYCLE",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "FsuCycle", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.PatchFsuCycle, sdksvc.CreatePatchFsuCycle, sdksvc.UpdatePatchFsuCycle]{
		CollectionPath: "/20220528/fsuCycles", ItemPath: "/20220528/fsuCycles/<ocid:1>",
		CreatePath: "/20220528/fsuCycles", CreateMethod: http.MethodPost,
		UpdatePath: "/20220528/fsuCycles/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220528/fsuCycles/<ocid:1>", DeleteMethod: http.MethodDelete,
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
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "type", "PATCH", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "type", "PATCH", updateRequest)
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
	manager := &FsuCycleServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newFsuCycleRuntimeHooks(manager, sdkClient)
	client := wrapFsuCycleGeneratedClient(hooks, defaultFsuCycleServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.FsuCycle](buildFsuCycleGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.FsuCycle]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.FsuCycle) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "S_ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created FsuCycle status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.FsuCycle) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.FsuCycle) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated FsuCycle status = %+v", current.Status)
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
