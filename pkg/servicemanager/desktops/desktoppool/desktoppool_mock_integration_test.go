/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package desktoppool

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/desktops"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/desktops/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationDesktopPoolCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.DesktopPool](t, `
{
  "metadata": {"name": "mock-desktoppool", "namespace": "default"},
  "spec": {
  "arePrivilegedUsers": false,
  "availabilityDomain": "mock-availabilitydomain",
  "availabilityPolicy": {
    "startSchedule": {
      "cronExpression": "mock-cronexpression",
      "timezone": "mock-timezone"
    },
    "stopSchedule": {
      "cronExpression": "mock-cronexpression",
      "timezone": "mock-timezone"
    }
  },
  "compartmentId": "<ocid:required>",
  "contactDetails": "mock-contactdetails",
  "devicePolicy": {
    "audioMode": "NONE",
    "cdmMode": "NONE",
    "clipboardMode": "NONE",
    "isDisplayEnabled": false,
    "isKeyboardEnabled": false,
    "isPointerEnabled": false,
    "isPrintingEnabled": false
  },
  "displayName": "mock-displayname-initial",
  "image": {
    "imageId": "<ocid:required>",
    "imageName": "mock-imagename"
  },
  "isStorageEnabled": false,
  "maximumSize": 1,
  "networkConfiguration": {
    "subnetId": "<ocid:required>",
    "vcnId": "<ocid:required>"
  },
  "shapeName": "mock-shapename",
  "standbySize": 1,
  "storageBackupPolicyId": "<ocid:required>",
  "storageSizeInGBs": 1,
  "useDedicatedVmHost": "TRUE"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-desktoppool")
	resource.Status = apiv1beta1.DesktopPoolStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateDesktopPoolDetails](t, `{
  "arePrivilegedUsers": false,
  "availabilityDomain": "mock-availabilitydomain",
  "availabilityPolicy": {
    "startSchedule": {
      "cronExpression": "mock-cronexpression",
      "timezone": "mock-timezone"
    },
    "stopSchedule": {
      "cronExpression": "mock-cronexpression",
      "timezone": "mock-timezone"
    }
  },
  "compartmentId": "<ocid:required>",
  "contactDetails": "mock-contactdetails",
  "devicePolicy": {
    "audioMode": "NONE",
    "cdmMode": "NONE",
    "clipboardMode": "NONE",
    "isDisplayEnabled": false,
    "isKeyboardEnabled": false,
    "isPointerEnabled": false,
    "isPrintingEnabled": false
  },
  "displayName": "mock-displayname-initial",
  "image": {
    "imageId": "<ocid:required>",
    "imageName": "mock-imagename"
  },
  "isStorageEnabled": false,
  "maximumSize": 1,
  "networkConfiguration": {
    "subnetId": "<ocid:required>",
    "vcnId": "<ocid:required>"
  },
  "shapeName": "mock-shapename",
  "standbySize": 1,
  "storageBackupPolicyId": "<ocid:required>",
  "storageSizeInGBs": 1,
  "useDedicatedVmHost": "TRUE"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateDesktopPoolDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.DesktopPool](t, `{
  "arePrivilegedUsers": false,
  "availabilityDomain": "mock-availabilitydomain",
  "availabilityPolicy": {
    "startSchedule": {
      "cronExpression": "mock-cronexpression",
      "timezone": "mock-timezone"
    },
    "stopSchedule": {
      "cronExpression": "mock-cronexpression",
      "timezone": "mock-timezone"
    }
  },
  "compartmentId": "<ocid:required>",
  "contactDetails": "mock-contactdetails",
  "devicePolicy": {
    "audioMode": "NONE",
    "cdmMode": "NONE",
    "clipboardMode": "NONE",
    "isDisplayEnabled": false,
    "isKeyboardEnabled": false,
    "isPointerEnabled": false,
    "isPrintingEnabled": false
  },
  "displayName": "mock-displayname-initial",
  "id": "<ocid:1>",
  "image": {
    "imageId": "<ocid:required>",
    "imageName": "mock-imagename"
  },
  "isStorageEnabled": false,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "maximumSize": 1,
  "networkConfiguration": {
    "subnetId": "<ocid:required>",
    "vcnId": "<ocid:required>"
  },
  "resourceId": "<ocid:1>",
  "shapeName": "mock-shapename",
  "standbySize": 1,
  "state": "ACTIVE",
  "status": "ACTIVE",
  "storageBackupPolicyId": "<ocid:required>",
  "storageSizeInGBs": 1,
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "useDedicatedVmHost": "TRUE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.DesktopPool](t, `{
  "arePrivilegedUsers": false,
  "availabilityDomain": "mock-availabilitydomain",
  "availabilityPolicy": {
    "startSchedule": {
      "cronExpression": "mock-cronexpression",
      "timezone": "mock-timezone"
    },
    "stopSchedule": {
      "cronExpression": "mock-cronexpression",
      "timezone": "mock-timezone"
    }
  },
  "compartmentId": "<ocid:required>",
  "contactDetails": "mock-contactdetails",
  "devicePolicy": {
    "audioMode": "NONE",
    "cdmMode": "NONE",
    "clipboardMode": "NONE",
    "isDisplayEnabled": false,
    "isKeyboardEnabled": false,
    "isPointerEnabled": false,
    "isPrintingEnabled": false
  },
  "displayName": "mock-displayname-updated",
  "id": "<ocid:1>",
  "image": {
    "imageId": "<ocid:required>",
    "imageName": "mock-imagename"
  },
  "isStorageEnabled": false,
  "key": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "maximumSize": 1,
  "networkConfiguration": {
    "subnetId": "<ocid:required>",
    "vcnId": "<ocid:required>"
  },
  "resourceId": "<ocid:1>",
  "shapeName": "mock-shapename",
  "standbySize": 1,
  "state": "ACTIVE",
  "status": "ACTIVE",
  "storageBackupPolicyId": "<ocid:required>",
  "storageSizeInGBs": 1,
  "timeCreated": "2026-01-02T03:04:05Z",
  "timeReleased": "2026-01-02T03:04:05Z",
  "timeScheduleStart": "2026-01-02T03:04:05Z",
  "timeUpdated": "2026-01-03T03:04:05Z",
  "useDedicatedVmHost": "TRUE"
}`)
	createWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:create-work-request>",
  "operationType": "CREATEDESKTOPPOOL",
  "percentComplete": 100,
  "resources": [{"actionType": "CREATED", "entityType": "DesktopPool", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	updateWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:update-work-request>",
  "operationType": "UPDATEDESKTOPPOOL",
  "percentComplete": 100,
  "resources": [{"actionType": "UPDATED", "entityType": "DesktopPool", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	deleteWorkRequest := ocimock.MustOCIResponseFixture[sdksvc.WorkRequest](t, `
{
  "id": "<ocid:delete-work-request>",
  "operationType": "DELETEDESKTOPPOOL",
  "percentComplete": 100,
  "resources": [{"actionType": "DELETED", "entityType": "DesktopPool", "identifier": "<ocid:1>"}],
  "status": "SUCCEEDED"
}
`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.DesktopPool, sdksvc.CreateDesktopPoolDetails, sdksvc.UpdateDesktopPoolDetails]{
		CollectionPath: "/20220618/desktopPools", ItemPath: "/20220618/desktopPools/<ocid:1>",
		CreatePath: "/20220618/desktopPools", CreateMethod: http.MethodPost,
		UpdatePath: "/20220618/desktopPools/<ocid:1>", UpdateMethod: http.MethodPut,
		DeletePath: "/20220618/desktopPools/<ocid:1>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateDesktopPoolDetails],
		CreatedState:  &createdState,
		UpdateRequest: &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateDesktopPoolDetails],
		UpdatedState:      &updatedState,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 202, UpdateStatus: 202, DeleteStatus: 202, NotFoundCode: "NotAuthorizedOrNotFound",
		CreateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:create-work-request>"}, "Opc-Request-Id": []string{"mock-create-request"}},
		UpdateHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:update-work-request>"}, "Opc-Request-Id": []string{"mock-update-request"}},
		DeleteHeaders: http.Header{"Opc-Work-Request-Id": []string{"<ocid:delete-work-request>"}, "Opc-Request-Id": []string{"mock-delete-request"}},
		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateDesktopPoolDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{
			{Name: "create-work-request", Method: http.MethodGet, Path: "/20220618/workRequests/<ocid:create-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", createWorkRequest)},
			{Name: "update-work-request", Method: http.MethodGet, Path: "/20220618/workRequests/<ocid:update-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", updateWorkRequest)},
			{Name: "delete-work-request", Method: http.MethodGet, Path: "/20220618/workRequests/<ocid:delete-work-request>", MinimumCalls: 1, Respond: ocimock.NewWorkRequestResponseSequence(t, "IN_PROGRESS", deleteWorkRequest)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://api.desktops.us-ashburn-1.oci.oraclecloud.com", BasePath: "20220618", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.DesktopServiceClient{BaseClient: session.BaseClient()}
	manager := &DesktopPoolServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newDesktopPoolRuntimeHooks(manager, sdkClient)
	client := wrapDesktopPoolGeneratedClient(hooks, defaultDesktopPoolServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.DesktopPool](buildDesktopPoolGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.DesktopPool]{
		RequireAsyncPending: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationUpdate, ocimock.OperationDelete},
		Resource:            resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.DesktopPool) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created DesktopPool status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.DesktopPool) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.DesktopPool) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated DesktopPool status = %+v", current.Status)
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
