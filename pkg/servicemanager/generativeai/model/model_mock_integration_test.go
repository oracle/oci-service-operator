/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package model

import (
	"context"
	"fmt"
	generativeaisdk "github.com/oracle/oci-go-sdk/v65/generativeai"
	generativeaiv1beta1 "github.com/oracle/oci-service-operator/api/generativeai/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationModelLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeSpecModel()
	ocimock.InitializeResource(resource, "mock-model")
	resource.Spec = ocimock.MustJSONFixture[generativeaiv1beta1.ModelSpec](t, `{
  "baseModelId": "\u003cocid:1\u003e",
  "compartmentId": "\u003cocid:2\u003e",
  "description": "OSOK Model sample",
  "displayName": "osok-model-sample",
  "fineTuneDetails": {
    "dedicatedAiClusterId": "\u003cocid:3\u003e",
    "trainingDataset": {
      "bucketName": "generativeai-training-data",
      "datasetType": "OBJECT_STORAGE",
      "namespaceName": "object-storage-namespace",
      "objectName": "datasets/model-training.jsonl"
    }
  },
  "freeformTags": {
    "managed-by": "osok"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "OSOK Model sample-updated"
}`)
	createRequest := ocimock.MustJSONFixture[generativeaisdk.CreateModelDetails](t, `{
  "baseModelId": "\u003cocid:1\u003e",
  "compartmentId": "\u003cocid:2\u003e",
  "description": "OSOK Model sample",
  "displayName": "osok-model-sample",
  "fineTuneDetails": {
    "dedicatedAiClusterId": "\u003cocid:3\u003e",
    "trainingDataset": {
      "bucketName": "generativeai-training-data",
      "datasetType": "OBJECT_STORAGE",
      "namespaceName": "object-storage-namespace",
      "objectName": "datasets/model-training.jsonl"
    }
  },
  "freeformTags": {
    "managed-by": "osok"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[generativeaisdk.Model](t, `{
  "baseModelId": "\u003cocid:1\u003e",
  "compartmentId": "\u003cocid:2\u003e",
  "description": "OSOK Model sample",
  "displayName": "osok-model-sample",
  "fineTuneDetails": {
    "dedicatedAiClusterId": "\u003cocid:3\u003e",
    "trainingDataset": {
      "bucketName": "generativeai-training-data",
      "datasetType": "OBJECT_STORAGE",
      "namespaceName": "object-storage-namespace",
      "objectName": "datasets/model-training.jsonl"
    }
  },
  "freeformTags": {
    "managed-by": "osok"
  },
  "id": "\u003cocid:4\u003e",
  "lifecycleState": "ACTIVE"
}`)
	updateRequest := ocimock.MustJSONFixture[generativeaisdk.UpdateModelDetails](t, `{
  "description": "OSOK Model sample-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[generativeaisdk.Model](t, `{
  "baseModelId": "\u003cocid:1\u003e",
  "compartmentId": "\u003cocid:2\u003e",
  "description": "OSOK Model sample-updated",
  "displayName": "osok-model-sample",
  "fineTuneDetails": {
    "dedicatedAiClusterId": "\u003cocid:3\u003e",
    "trainingDataset": {
      "bucketName": "generativeai-training-data",
      "datasetType": "OBJECT_STORAGE",
      "namespaceName": "object-storage-namespace",
      "objectName": "datasets/model-training.jsonl"
    }
  },
  "freeformTags": {
    "managed-by": "osok"
  },
  "id": "\u003cocid:4\u003e",
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		generativeaisdk.Model,
		generativeaisdk.CreateModelDetails,
		generativeaisdk.UpdateModelDetails,
	]{
		CollectionPath:     "/20231130/models",
		ItemPath:           "/20231130/models/<ocid:4>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		CreatedReadStates:  ocimock.LifecycleStateSequence(t, createdState, "CREATING"),
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		DeletedReadStates:  ocimock.LifecycleStates(t, updatedState, "DELETING"),
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ generativeaisdk.CreateModelDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ generativeaisdk.Model) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20231130", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Model OCI mock: %v", err)
		}
	})
	sdkClient := generativeaisdk.GenerativeAiClient{BaseClient: session.BaseClient()}
	manager := &ModelServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newModelDefaultRuntimeHooks(sdkClient)
	applyModelRuntimeHooks(&hooks)
	client := wrapModelGeneratedClient(hooks, defaultModelServiceClient{ServiceClient: generatedruntime.NewServiceClient[*generativeaiv1beta1.Model](buildModelGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*generativeaiv1beta1.Model]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *generativeaiv1beta1.Model) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.BaseModelId, current.Spec.BaseModelId) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FineTuneDetails, current.Spec.FineTuneDetails) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("created Model status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *generativeaiv1beta1.Model) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *generativeaiv1beta1.Model) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated Model status = %+v", current.Status)
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
