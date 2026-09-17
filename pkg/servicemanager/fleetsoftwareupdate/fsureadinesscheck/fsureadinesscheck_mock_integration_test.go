/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package fsureadinesscheck

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
func TestMockIntegrationFsuReadinessCheckCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.FsuReadinessCheck](t, `
{
  "metadata": {"name": "mock-fsureadinesscheck", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "type": "TARGET"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-fsureadinesscheck")
	resource.Status = apiv1beta1.FsuReadinessCheckStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateTargetFsuReadinessCheckDetails](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateFsuReadinessCheckDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.TargetFsuReadinessCheck](t, `{
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
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.TargetFsuReadinessCheck](t, `{
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
  "operationType": "CREATEFSUREADINESSCHECK",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "FsuReadinessCheck", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEFSUREADINESSCHECK",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "FsuReadinessCheck", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.TargetFsuReadinessCheck, sdksvc.CreateTargetFsuReadinessCheckDetails, sdksvc.UpdateFsuReadinessCheckDetails]{
		CollectionPath: "/20220528/fsuReadinessChecks", ItemPath: "/20220528/fsuReadinessChecks/<ocid:1>",
		CreatePath: "/20220528/fsuReadinessChecks", CreateMethod: http.MethodPost,
		UpdatePath: "/20220528/fsuReadinessChecks/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220528/fsuReadinessChecks/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},

		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateFsuReadinessCheckDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 200, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequestSubset(request, "type", "TARGET", createRequest)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20220528/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
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
	manager := &FsuReadinessCheckServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newFsuReadinessCheckRuntimeHooks(manager, sdkClient)
	client := wrapFsuReadinessCheckGeneratedClient(hooks, defaultFsuReadinessCheckServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.FsuReadinessCheck](buildFsuReadinessCheckGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.FsuReadinessCheck]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.FsuReadinessCheck) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created FsuReadinessCheck status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.FsuReadinessCheck) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.FsuReadinessCheck) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated FsuReadinessCheck status = %+v", current.Status)
			}
			return nil
		},
		ValidateStable: func(*apiv1beta1.FsuReadinessCheck) error {
			if got := responder.OperationCounts()[ocimock.OperationUpdate]; got != 1 {
				return fmt.Errorf("stable FsuReadinessCheck update calls = %d, want 1", got)
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
