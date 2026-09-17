/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package mediaworkflow

import (
	"context"
	"fmt"
	mediaservicessdk "github.com/oracle/oci-go-sdk/v65/mediaservices"
	mediaservicesv1beta1 "github.com/oracle/oci-service-operator/api/mediaservices/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationMediaWorkflowLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newMediaWorkflowTestResource()
	ocimock.InitializeResource(resource, "mock-mediaworkflow")
	resource.Spec = ocimock.MustJSONFixture[mediaservicesv1beta1.MediaWorkflowSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "workflow-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "mediaWorkflowConfigurationIds": [
    "\u003cocid:2\u003e"
  ],
  "parameters": {
    "bucket": "media-bucket",
    "retryCount": 3
  },
  "tasks": [
    {
      "enableParameterReference": "/flags/transcodeEnabled",
      "enableWhenReferencedParameterEquals": {
        "value": true
      },
      "key": "transcode",
      "parameters": {
        "bitrateKbps": 6000,
        "preset": "HD"
      },
      "type": "TRANSCODE_VIDEO",
      "version": 1
    }
  ]
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "workflow-alpha-updated"
}`)
	createRequest := ocimock.MustJSONFixture[mediaservicessdk.CreateMediaWorkflowDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "workflow-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "mediaWorkflowConfigurationIds": [
    "\u003cocid:2\u003e"
  ],
  "parameters": {
    "bucket": "media-bucket",
    "retryCount": 3
  },
  "tasks": [
    {
      "enableParameterReference": "/flags/transcodeEnabled",
      "enableWhenReferencedParameterEquals": {
        "value": true
      },
      "key": "transcode",
      "parameters": {
        "bitrateKbps": 6000,
        "preset": "HD"
      },
      "type": "TRANSCODE_VIDEO",
      "version": 1
    }
  ]
}`)
	createdState := ocimock.MustOCIResponseFixture[mediaservicessdk.MediaWorkflow](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "workflow-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "mediaWorkflowConfigurationIds": [
    "\u003cocid:2\u003e"
  ],
  "parameters": {
    "bucket": "media-bucket",
    "retryCount": 3
  },
  "tasks": [
    {
      "enableParameterReference": "/flags/transcodeEnabled",
      "enableWhenReferencedParameterEquals": {
        "value": true
      },
      "key": "transcode",
      "parameters": {
        "bitrateKbps": 6000,
        "preset": "HD"
      },
      "type": "TRANSCODE_VIDEO",
      "version": 1
    }
  ]
}`)
	updateRequest := ocimock.MustJSONFixture[mediaservicessdk.UpdateMediaWorkflowDetails](t, `{
  "displayName": "workflow-alpha-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[mediaservicessdk.MediaWorkflow](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "workflow-alpha-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "mediaWorkflowConfigurationIds": [
    "\u003cocid:2\u003e"
  ],
  "parameters": {
    "bucket": "media-bucket",
    "retryCount": 3
  },
  "tasks": [
    {
      "enableParameterReference": "/flags/transcodeEnabled",
      "enableWhenReferencedParameterEquals": {
        "value": true
      },
      "key": "transcode",
      "parameters": {
        "bitrateKbps": 6000,
        "preset": "HD"
      },
      "type": "TRANSCODE_VIDEO",
      "version": 1
    }
  ]
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		mediaservicessdk.MediaWorkflow,
		mediaservicessdk.CreateMediaWorkflowDetails,
		mediaservicessdk.UpdateMediaWorkflowDetails,
	]{
		CollectionPath:    "/20211101/mediaWorkflows",
		ItemPath:          "/20211101/mediaWorkflows/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeItems,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ mediaservicessdk.CreateMediaWorkflowDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ mediaservicessdk.MediaWorkflow) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20211101", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close MediaWorkflow OCI mock: %v", err)
		}
	})
	sdkClient := mediaservicessdk.MediaServicesClient{BaseClient: session.BaseClient()}
	client := newMediaWorkflowServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*mediaservicesv1beta1.MediaWorkflow]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *mediaservicesv1beta1.MediaWorkflow) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.MediaWorkflowConfigurationIds, current.Spec.MediaWorkflowConfigurationIds) ||
				!reflect.DeepEqual(current.Status.Parameters, current.Spec.Parameters) ||
				!reflect.DeepEqual(current.Status.Tasks, current.Spec.Tasks) {
				return fmt.Errorf("created MediaWorkflow status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *mediaservicesv1beta1.MediaWorkflow) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *mediaservicesv1beta1.MediaWorkflow) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("updated MediaWorkflow status = %+v", current.Status)
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
