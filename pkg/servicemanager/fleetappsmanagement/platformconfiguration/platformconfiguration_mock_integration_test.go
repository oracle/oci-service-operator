/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package platformconfiguration

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
func TestMockIntegrationPlatformConfigurationCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.PlatformConfiguration](t, `
{
  "metadata": {"name": "mock-platformconfiguration", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:required>",
  "configCategoryDetails": {
    "configCategory": "PRODUCT_STACK",
    "products": [
      {
        "id": "<ocid:required>"
      }
    ]
  },
  "displayName": "mock-displayname-initial"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-platformconfiguration")
	resource.Status = apiv1beta1.PlatformConfigurationStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreatePlatformConfigurationDetails](t, `{
  "compartmentId": "<ocid:required>",
  "configCategoryDetails": {
    "configCategory": "PRODUCT_STACK",
    "products": [
      {
        "id": "<ocid:required>"
      }
    ]
  },
  "displayName": "mock-displayname-initial"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdatePlatformConfigurationDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.PlatformConfiguration](t, `{
  "compartmentId": "<ocid:required>",
  "configCategoryDetails": {
    "configCategory": "PRODUCT_STACK",
    "products": [
      {
        "id": "<ocid:required>"
      }
    ]
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
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.PlatformConfiguration](t, `{
  "compartmentId": "<ocid:required>",
  "configCategoryDetails": {
    "configCategory": "PRODUCT_STACK",
    "products": [
      {
        "id": "<ocid:required>"
      }
    ]
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
  "operationType": "CREATEPLATFORMCONFIGURATION",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "PlatformConfiguration", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEPLATFORMCONFIGURATION",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "PlatformConfiguration", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEPLATFORMCONFIGURATION",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "PlatformConfiguration", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.PlatformConfiguration, sdksvc.CreatePlatformConfigurationDetails, sdksvc.UpdatePlatformConfigurationDetails]{
		CollectionPath: "/20250228/platformConfigurations", ItemPath: "/20250228/platformConfigurations/<ocid:1>",
		CreatePath: "/20250228/platformConfigurations", CreateMethod: http.MethodPost,
		UpdatePath: "/20250228/platformConfigurations/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20250228/platformConfigurations/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreatePlatformConfigurationDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdatePlatformConfigurationDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreatePlatformConfigurationDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20250228/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
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
	sdkClient := PlatformConfigurationSDKClients{fleetAppsManagementAdminClient: sdksvc.FleetAppsManagementAdminClient{BaseClient: session.BaseClient()}, fleetAppsManagementWorkRequestClient: sdksvc.FleetAppsManagementWorkRequestClient{BaseClient: session.BaseClient()}}
	manager := &PlatformConfigurationServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newPlatformConfigurationRuntimeHooks(manager, sdkClient)
	client := wrapPlatformConfigurationGeneratedClient(hooks, defaultPlatformConfigurationServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.PlatformConfiguration](buildPlatformConfigurationGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.PlatformConfiguration]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.PlatformConfiguration) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created PlatformConfiguration status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.PlatformConfiguration) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.PlatformConfiguration) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated PlatformConfiguration status = %+v", current.Status)
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
