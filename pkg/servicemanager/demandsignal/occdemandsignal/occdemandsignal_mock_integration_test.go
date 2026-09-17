/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package occdemandsignal

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/demandsignal"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/demandsignal/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationOccDemandSignalCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.OccDemandSignal](t, `
{
  "metadata": {"name": "mock-occdemandsignal", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "isActive": false,
  "occDemandSignals": [
    {
      "resourceType": "mock-resourcetype",
      "units": "mock-units",
      "values": [
        {
          "timeExpected": "2026-01-02T03:04:05Z",
          "value": 1
        }
      ]
    }
  ]
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-occdemandsignal")
	resource.Status = apiv1beta1.OccDemandSignalStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateOccDemandSignalDetails](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "isActive": false,
  "occDemandSignals": [
    {
      "resourceType": "mock-resourcetype",
      "units": "mock-units",
      "values": [
        {
          "timeExpected": "2026-01-02T03:04:05Z",
          "value": 1
        }
      ]
    }
  ]
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateOccDemandSignalDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.OccDemandSignal](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:1>",
  "isActive": false,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "occDemandSignals": [
    {
      "resourceType": "mock-resourcetype",
      "units": "mock-units",
      "values": [
        {
          "timeExpected": "2026-01-02T03:04:05Z",
          "value": 1
        }
      ]
    }
  ],
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.OccDemandSignal](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:1>",
  "isActive": false,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "occDemandSignals": [
    {
      "resourceType": "mock-resourcetype",
      "units": "mock-units",
      "values": [
        {
          "timeExpected": "2026-01-02T03:04:05Z",
          "value": 1
        }
      ]
    }
  ],
  "resourceId": "<ocid:1>",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)

	creatingState := createdState
	creatingState.LifecycleState = "CREATING"
	deletingState := updatedState
	deletingState.LifecycleState = "DELETING"

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.OccDemandSignal, sdksvc.CreateOccDemandSignalDetails, sdksvc.UpdateOccDemandSignalDetails]{
		CollectionPath: "/20240430/occDemandSignals", ItemPath: "/20240430/occDemandSignals/<ocid:1>",
		CreatePath: "/20240430/occDemandSignals", CreateMethod: http.MethodPost,
		UpdatePath: "/20240430/occDemandSignals/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20240430/occDemandSignals/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateOccDemandSignalDetails],
		CreatedState:      &creatingState,
		CreatedReadStates: ocimock.StateSequence(creatingState, createdState),
		UpdateRequest:     &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateOccDemandSignalDetails],
		UpdatedState:      &updatedState,
		DeletedState:      &deletingState,
		DeletedReadStates: ocimock.StateSequence(deletingState),
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",

		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateOccDemandSignalDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://control-center-ds.us-ashburn-1.oci.oraclecloud.com", BasePath: "20240430", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.OccDemandSignalClient{BaseClient: session.BaseClient()}
	manager := &OccDemandSignalServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newOccDemandSignalRuntimeHooks(manager, sdkClient)
	client := wrapOccDemandSignalGeneratedClient(hooks, defaultOccDemandSignalServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.OccDemandSignal](buildOccDemandSignalGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.OccDemandSignal]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.OccDemandSignal) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created OccDemandSignal status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.OccDemandSignal) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.OccDemandSignal) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated OccDemandSignal status = %+v", current.Status)
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
