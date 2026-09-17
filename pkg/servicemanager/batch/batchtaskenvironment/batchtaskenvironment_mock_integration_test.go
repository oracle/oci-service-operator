/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package batchtaskenvironment

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
func TestMockIntegrationBatchTaskEnvironmentEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &batchv1beta1.BatchTaskEnvironment{}
	ocimock.InitializeResource(resource, "mock-batchtaskenvironment")
	resource.Spec = ocimock.MustJSONFixture[batchv1beta1.BatchTaskEnvironmentSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-batch-task-environment-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "imageUrl": "\u003cbinding:batch-image-url\u003e",
  "volumes": []
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[batchsdk.CreateBatchTaskEnvironmentDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "recorded create",
  "displayName": "osok-mock-batch-task-environment-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "imageUrl": "\u003cbinding:batch-image-url\u003e",
  "volumes": []
}`)
	createdState := ocimock.MustOCIResponseFixture[batchsdk.BatchTaskEnvironment](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T04:50:55.020Z"
    }
  },
  "description": "recorded create",
  "displayName": "osok-mock-batch-task-environment-v1",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "imageUrl": "<binding:batch-image-url>",
  "lifecycleState": "ACTIVE",
  "securityContext": null,
  "systemTags": {},
  "timeCreated": "2026-09-03T04:50:55.138Z",
  "timeUpdated": "2026-09-03T04:50:55.162Z",
  "volumes": [],
  "workingDirectory": null
}`)
	updateRequest := ocimock.MustJSONFixture[batchsdk.UpdateBatchTaskEnvironmentDetails](t, `{
  "description": "recorded update",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[batchsdk.BatchTaskEnvironment](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T04:50:55.020Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-batch-task-environment-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "imageUrl": "<binding:batch-image-url>",
  "lifecycleState": "ACTIVE",
  "securityContext": null,
  "systemTags": {},
  "timeCreated": "2026-09-03T04:50:55.138Z",
  "timeUpdated": "2026-09-03T04:50:55.789Z",
  "volumes": [],
  "workingDirectory": null
}`)
	deletedState := ocimock.MustOCIResponseFixture[batchsdk.BatchTaskEnvironment](t, `{
  "compartmentId": "<ocid:1>",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-03T04:50:55.020Z"
    }
  },
  "description": "recorded update",
  "displayName": "osok-mock-batch-task-environment-v1",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "imageUrl": "<binding:batch-image-url>",
  "lifecycleState": "DELETED",
  "securityContext": null,
  "systemTags": {},
  "timeCreated": "2026-09-03T04:50:55.138Z",
  "timeUpdated": "2026-09-03T04:50:56.412Z",
  "volumes": [],
  "workingDirectory": null
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		batchsdk.BatchTaskEnvironment,
		batchsdk.CreateBatchTaskEnvironmentDetails,
		batchsdk.UpdateBatchTaskEnvironmentDetails,
	]{
		CollectionPath:    "/20251031/batchTaskEnvironments",
		ItemPath:          "/20251031/batchTaskEnvironments/<ocid:2>",
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
		ValidateCreate: func(request ocimock.Request, _ batchsdk.CreateBatchTaskEnvironmentDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ batchsdk.BatchTaskEnvironment) error {
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
			t.Errorf("close BatchTaskEnvironment OCI mock: %v", err)
		}
	})
	sdkClient := batchsdk.BatchComputingClient{BaseClient: session.BaseClient()}
	manager := &BatchTaskEnvironmentServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newBatchTaskEnvironmentRuntimeHooks(manager, sdkClient)
	client := wrapBatchTaskEnvironmentGeneratedClient(hooks, defaultBatchTaskEnvironmentServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*batchv1beta1.BatchTaskEnvironment](buildBatchTaskEnvironmentGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*batchv1beta1.BatchTaskEnvironment]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *batchv1beta1.BatchTaskEnvironment) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.ImageUrl, current.Spec.ImageUrl) ||
				!reflect.DeepEqual(current.Status.Volumes, current.Spec.Volumes) {
				return fmt.Errorf("created BatchTaskEnvironment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *batchv1beta1.BatchTaskEnvironment) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *batchv1beta1.BatchTaskEnvironment) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated BatchTaskEnvironment status = %+v", current.Status)
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
