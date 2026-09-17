/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package batchtaskprofile

import (
	"context"
	"fmt"
	batchsdk "github.com/oracle/oci-go-sdk/v65/batch"
	batchv1beta1 "github.com/oracle/oci-service-operator/api/batch/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationBatchTaskProfileEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &batchv1beta1.BatchTaskProfile{}
	ocimock.InitializeResource(resource, "mock-batchtaskprofile")
	resource.Spec = ocimock.MustJSONFixture[batchv1beta1.BatchTaskProfileSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-batch-task-profile-v3",
  "freeformTags": {
    "osok-mock": "create"
  },
  "minMemoryInGBs": 1,
  "minOcpus": 1
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[batchsdk.CreateBatchTaskProfileDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-batch-task-profile-v3",
  "freeformTags": {
    "osok-mock": "create"
  },
  "minMemoryInGBs": 1,
  "minOcpus": 1
}`)
	createdState := ocimock.MustOCIResponseFixture[batchsdk.BatchTaskProfile](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T04:50:44.082Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-batch-task-profile-v3",
  "extendedInformation": null,
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "minDiskSizeInGBs": 0,
  "minMemoryInGBs": 1,
  "minOcpus": 1,
  "systemTags": {},
  "timeCreated": "2026-09-03T04:50:44.255Z",
  "timeUpdated": "2026-09-03T04:50:44.285Z"
}`)
	updateRequest := ocimock.MustJSONFixture[batchsdk.UpdateBatchTaskProfileDetails](t, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[batchsdk.BatchTaskProfile](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T04:50:44.082Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-batch-task-profile-v3",
  "extendedInformation": null,
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "minDiskSizeInGBs": 0,
  "minMemoryInGBs": 1,
  "minOcpus": 1,
  "systemTags": {},
  "timeCreated": "2026-09-03T04:50:44.255Z",
  "timeUpdated": "2026-09-03T04:50:44.931Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[batchsdk.BatchTaskProfile](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T04:50:44.082Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-batch-task-profile-v3",
  "extendedInformation": null,
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "DELETED",
  "minDiskSizeInGBs": 0,
  "minMemoryInGBs": 1,
  "minOcpus": 1,
  "systemTags": {},
  "timeCreated": "2026-09-03T04:50:44.255Z",
  "timeUpdated": "2026-09-03T04:50:45.736Z"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		batchsdk.BatchTaskProfile,
		batchsdk.CreateBatchTaskProfileDetails,
		batchsdk.UpdateBatchTaskProfileDetails,
	]{
		CollectionPath:    "/20251031/batchTaskProfiles",
		ItemPath:          "/20251031/batchTaskProfiles/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		DeletedState:      &deletedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ batchsdk.CreateBatchTaskProfileDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ batchsdk.BatchTaskProfile) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20251031", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close BatchTaskProfile OCI mock: %v", err)
		}
	})
	sdkClient := batchsdk.BatchComputingClient{BaseClient: session.BaseClient()}
	manager := &BatchTaskProfileServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newBatchTaskProfileRuntimeHooks(manager, sdkClient)
	client := wrapBatchTaskProfileGeneratedClient(hooks, defaultBatchTaskProfileServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*batchv1beta1.BatchTaskProfile](buildBatchTaskProfileGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*batchv1beta1.BatchTaskProfile]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *batchv1beta1.BatchTaskProfile) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.MinMemoryInGBs, current.Spec.MinMemoryInGBs) ||
				!reflect.DeepEqual(current.Status.MinOcpus, current.Spec.MinOcpus) {
				return fmt.Errorf("created BatchTaskProfile status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *batchv1beta1.BatchTaskProfile) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *batchv1beta1.BatchTaskProfile) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated BatchTaskProfile status = %+v", current.Status)
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
