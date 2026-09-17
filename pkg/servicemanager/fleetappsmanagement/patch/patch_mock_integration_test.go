/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package patch

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
func TestMockIntegrationPatchCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.Patch](t, `
{
  "metadata": {"name": "mock-patch", "namespace": "default"},
  "spec": {
  "artifactDetails": {
    "artifacts": [
      {
        "architecture": "ARM_64",
        "content": {
          "bucketName": "mock-bucketname",
          "checksum": "mock-checksum",
          "namespaceName": "mock-namespacename",
          "objectName": "mock-objectname",
          "sourceType": "OBJECT_STORAGE_BUCKET"
        },
        "osType": "WINDOWS"
      }
    ],
    "category": "PLATFORM_SPECIFIC"
  },
  "compartmentId": "<ocid:required>",
  "description": "mock-description-initial",
  "name": "mock-name",
  "patchType": {
    "platformConfigurationId": "<ocid:required>"
  },
  "product": {
    "platformConfigurationId": "<ocid:required>",
    "version": "mock-version"
  },
  "severity": "CRITICAL",
  "timeReleased": "2026-01-02T03:04:05Z"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-patch")
	resource.Status = apiv1beta1.PatchStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreatePatchDetails](t, `{
  "artifactDetails": {
    "artifacts": [
      {
        "architecture": "ARM_64",
        "content": {
          "bucketName": "mock-bucketname",
          "checksum": "mock-checksum",
          "namespaceName": "mock-namespacename",
          "objectName": "mock-objectname",
          "sourceType": "OBJECT_STORAGE_BUCKET"
        },
        "osType": "WINDOWS"
      }
    ],
    "category": "PLATFORM_SPECIFIC"
  },
  "compartmentId": "<ocid:required>",
  "description": "mock-description-initial",
  "name": "mock-name",
  "patchType": {
    "platformConfigurationId": "<ocid:required>"
  },
  "product": {
    "platformConfigurationId": "<ocid:required>",
    "version": "mock-version"
  },
  "severity": "CRITICAL",
  "timeReleased": "2026-01-02T03:04:05Z"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdatePatchDetails](t, `{
  "description": "mock-description-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.Patch](t, `{
  "artifactDetails": {
    "artifacts": [
      {
        "architecture": "ARM_64",
        "content": {
          "bucketName": "mock-bucketname",
          "checksum": "mock-checksum",
          "namespaceName": "mock-namespacename",
          "objectName": "mock-objectname",
          "sourceType": "OBJECT_STORAGE_BUCKET"
        },
        "osType": "WINDOWS"
      }
    ],
    "category": "PLATFORM_SPECIFIC"
  },
  "compartmentId": "<ocid:required>",
  "description": "mock-description-initial",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "name": "mock-name",
  "patchType": {
    "platformConfigurationId": "<ocid:required>"
  },
  "product": {
    "platformConfigurationId": "<ocid:required>",
    "version": "mock-version"
  },
  "resourceId": "<ocid:1>",
  "severity": "CRITICAL",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.Patch](t, `{
  "artifactDetails": {
    "artifacts": [
      {
        "architecture": "ARM_64",
        "content": {
          "bucketName": "mock-bucketname",
          "checksum": "mock-checksum",
          "namespaceName": "mock-namespacename",
          "objectName": "mock-objectname",
          "sourceType": "OBJECT_STORAGE_BUCKET"
        },
        "osType": "WINDOWS"
      }
    ],
    "category": "PLATFORM_SPECIFIC"
  },
  "compartmentId": "<ocid:required>",
  "description": "mock-description-updated",
  "id": "<ocid:1>",
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "name": "mock-name",
  "patchType": {
    "platformConfigurationId": "<ocid:required>"
  },
  "product": {
    "platformConfigurationId": "<ocid:required>",
    "version": "mock-version"
  },
  "resourceId": "<ocid:1>",
  "severity": "CRITICAL",
  "state": "ACTIVE",
  "status": "ACTIVE",
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z"
}`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEPATCH",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "Patch", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEPATCH",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "Patch", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.Patch, sdksvc.CreatePatchDetails, sdksvc.UpdatePatchDetails]{
		CollectionPath: "/20250228/patches", ItemPath: "/20250228/patches/<ocid:1>",
		CreatePath: "/20250228/patches", CreateMethod: http.MethodPost,
		UpdatePath: "/20250228/patches/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20250228/patches/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreatePatchDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdatePatchDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreatePatchDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
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
	sdkClient := PatchSDKClients{fleetAppsManagementOperationsClient: sdksvc.FleetAppsManagementOperationsClient{BaseClient: session.BaseClient()}, fleetAppsManagementWorkRequestClient: sdksvc.FleetAppsManagementWorkRequestClient{BaseClient: session.BaseClient()}}
	manager := &PatchServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newPatchRuntimeHooks(manager, sdkClient)
	client := wrapPatchGeneratedClient(hooks, defaultPatchServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.Patch](buildPatchGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.Patch]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.Patch) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.Description != "mock-description-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created Patch status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.Patch) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "description": "mock-description-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.Patch) error {
			if current.Status.Description != "mock-description-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated Patch status = %+v", current.Status)
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
