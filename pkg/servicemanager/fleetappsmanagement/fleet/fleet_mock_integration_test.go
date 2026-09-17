/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package fleet

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/fleetappsmanagement"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/fleetappsmanagement/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationFleetCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.Fleet](t, `
{
  "metadata": {"name": "mock-fleet", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "parentFleetId": "<ocid:9>",
  "resourceSelection": {
    "resourceSelectionType": "DYNAMIC",
    "ruleSelectionCriteria": {
      "matchCondition": "MATCH_ALL",
      "rules": [
        {
          "compartmentId": "<ocid:required>",
          "conditions": [
            {
              "attrGroup": "mock-attrgroup",
              "attrKey": "mock-attrkey",
              "attrValue": "mock-attrvalue"
            }
          ],
          "resourceCompartmentId": "<ocid:required>"
        }
      ]
    }
  }
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-fleet")
	resource.Status = apiv1beta1.FleetStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateFleetDetails](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "parentFleetId": "<ocid:9>",
  "resourceSelection": {
    "resourceSelectionType": "DYNAMIC",
    "ruleSelectionCriteria": {
      "matchCondition": "MATCH_ALL",
      "rules": [
        {
          "compartmentId": "<ocid:required>",
          "conditions": [
            {
              "attrGroup": "mock-attrgroup",
              "attrKey": "mock-attrkey",
              "attrValue": "mock-attrvalue"
            }
          ],
          "resourceCompartmentId": "<ocid:required>"
        }
      ]
    }
  }
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateFleetDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.Fleet](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "parentFleetId": "<ocid:9>",
  "resourceId": "<ocid:1>",
  "resourceSelection": {
    "resourceSelectionType": "DYNAMIC",
    "ruleSelectionCriteria": {
      "matchCondition": "MATCH_ALL",
      "rules": [
        {
          "compartmentId": "<ocid:required>",
          "conditions": [
            {
              "attrGroup": "mock-attrgroup",
              "attrKey": "mock-attrkey",
              "attrValue": "mock-attrvalue"
            }
          ],
          "resourceCompartmentId": "<ocid:required>"
        }
      ]
    }
  },
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.Fleet](t, `{
  "compartmentId": "<ocid:required>",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "parentFleetId": "<ocid:9>",
  "resourceId": "<ocid:1>",
  "resourceSelection": {
    "resourceSelectionType": "DYNAMIC",
    "ruleSelectionCriteria": {
      "matchCondition": "MATCH_ALL",
      "rules": [
        {
          "compartmentId": "<ocid:required>",
          "conditions": [
            {
              "attrGroup": "mock-attrgroup",
              "attrKey": "mock-attrkey",
              "attrValue": "mock-attrvalue"
            }
          ],
          "resourceCompartmentId": "<ocid:required>"
        }
      ]
    }
  },
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
  "operationType": "CREATEFLEET",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "Fleet", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEFLEET",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "Fleet", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.Fleet, sdksvc.CreateFleetDetails, sdksvc.UpdateFleetDetails]{
		CollectionPath: "/20250228/fleets", ItemPath: "/20250228/fleets/<ocid:1>",
		CreatePath: "/20250228/fleets", CreateMethod: http.MethodPost,
		UpdatePath: "/20250228/fleets/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20250228/fleets/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateFleetDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateFleetDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 200, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateFleetDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://fams.us-ashburn-1.oci.oraclecloud.com", BasePath: "20250228", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := FleetSDKClients{fleetAppsManagementClient: sdksvc.FleetAppsManagementClient{BaseClient: session.BaseClient()}, fleetAppsManagementWorkRequestClient: sdksvc.FleetAppsManagementWorkRequestClient{BaseClient: session.BaseClient()}}
	manager := &FleetServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newFleetRuntimeHooks(manager, sdkClient)
	client := wrapFleetGeneratedClient(hooks, defaultFleetServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.Fleet](buildFleetGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.Fleet]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.Fleet) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Fleet status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.Fleet) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.Fleet) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Fleet status = %+v", current.Status)
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
