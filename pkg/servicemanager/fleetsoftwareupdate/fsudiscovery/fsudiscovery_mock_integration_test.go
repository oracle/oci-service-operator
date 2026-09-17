/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package fsudiscovery

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
func TestMockIntegrationFsuDiscoveryCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.FsuDiscovery](t, `
{
  "metadata": {"name": "mock-fsudiscovery", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "details": {
    "criteria": {
      "filters": [
        {
          "tags": [
            {
              "key": "mock-key",
              "namespace": "mock-namespace",
              "value": "mock-value"
            }
          ],
          "type": "DEFINED_TAG"
        }
      ],
      "strategy": "FILTERS"
    },
    "serviceType": "EXACS",
    "sourceMajorVersion": "GI_18",
    "type": "GI"
  },
  "displayName": "mock-displayname-initial"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-fsudiscovery")
	resource.Status = apiv1beta1.FsuDiscoveryStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateFsuDiscoveryDetails](t, `{
  "compartmentId": "<ocid:required>",
  "details": {
    "criteria": {
      "filters": [
        {
          "tags": [
            {
              "key": "mock-key",
              "namespace": "mock-namespace",
              "value": "mock-value"
            }
          ],
          "type": "DEFINED_TAG"
        }
      ],
      "strategy": "FILTERS"
    },
    "serviceType": "EXACS",
    "sourceMajorVersion": "GI_18",
    "type": "GI"
  },
  "displayName": "mock-displayname-initial"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateFsuDiscoveryDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.FsuDiscovery](t, `{
  "compartmentId": "<ocid:required>",
  "details": {
    "criteria": {
      "filters": [
        {
          "tags": [
            {
              "key": "mock-key",
              "namespace": "mock-namespace",
              "value": "mock-value"
            }
          ],
          "type": "DEFINED_TAG"
        }
      ],
      "strategy": "FILTERS"
    },
    "serviceType": "EXACS",
    "sourceMajorVersion": "GI_18",
    "type": "GI"
  },
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
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.FsuDiscovery](t, `{
  "compartmentId": "<ocid:required>",
  "details": {
    "criteria": {
      "filters": [
        {
          "tags": [
            {
              "key": "mock-key",
              "namespace": "mock-namespace",
              "value": "mock-value"
            }
          ],
          "type": "DEFINED_TAG"
        }
      ],
      "strategy": "FILTERS"
    },
    "serviceType": "EXACS",
    "sourceMajorVersion": "GI_18",
    "type": "GI"
  },
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
  "operationType": "CREATEFSUDISCOVERY",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "FsuDiscovery", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEFSUDISCOVERY",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "FsuDiscovery", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.FsuDiscovery, sdksvc.CreateFsuDiscoveryDetails, sdksvc.UpdateFsuDiscoveryDetails]{
		CollectionPath: "/20220528/fsuDiscoveries", ItemPath: "/20220528/fsuDiscoveries/<ocid:1>",
		CreatePath: "/20220528/fsuDiscoveries", CreateMethod: http.MethodPost,
		UpdatePath: "/20220528/fsuDiscoveries/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220528/fsuDiscoveries/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateFsuDiscoveryDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateFsuDiscoveryDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 200, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateFsuDiscoveryDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
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
	manager := &FsuDiscoveryServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newFsuDiscoveryRuntimeHooks(manager, sdkClient)
	client := wrapFsuDiscoveryGeneratedClient(hooks, defaultFsuDiscoveryServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.FsuDiscovery](buildFsuDiscoveryGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.FsuDiscovery]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.FsuDiscovery) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created FsuDiscovery status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.FsuDiscovery) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.FsuDiscovery) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated FsuDiscovery status = %+v", current.Status)
			}
			return nil
		},
		ValidateStable: func(*apiv1beta1.FsuDiscovery) error {
			if got := responder.OperationCounts()[ocimock.OperationUpdate]; got != 1 {
				return fmt.Errorf("stable FsuDiscovery update calls = %d, want 1", got)
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
