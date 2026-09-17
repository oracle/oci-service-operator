/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package mediaworkflowconfiguration

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
func TestMockIntegrationMediaWorkflowConfigurationLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newMediaWorkflowConfigurationTestResource()
	ocimock.InitializeResource(resource, "mock-mediaworkflowconfiguration")
	resource.Spec = ocimock.MustJSONFixture[mediaservicesv1beta1.MediaWorkflowConfigurationSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "configuration-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "parameters": {
    "thumbnails": {
      "enabled": true
    },
    "transcode": {
      "bitrateKbps": 6000,
      "preset": "HD"
    }
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "configuration-alpha-updated"
}`)
	createRequest := ocimock.MustJSONFixture[mediaservicessdk.CreateMediaWorkflowConfigurationDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "configuration-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "parameters": {
    "thumbnails": {
      "enabled": true
    },
    "transcode": {
      "bitrateKbps": 6000,
      "preset": "HD"
    }
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[mediaservicessdk.MediaWorkflowConfiguration](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "configuration-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "parameters": {
    "thumbnails": {
      "enabled": true
    },
    "transcode": {
      "bitrateKbps": 6000,
      "preset": "HD"
    }
  }
}`)
	createdReadStates := []mediaservicessdk.MediaWorkflowConfiguration{
		ocimock.MustOCIResponseFixture[mediaservicessdk.MediaWorkflowConfiguration](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "configuration-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "parameters": {
    "thumbnails": {
      "enabled": true
    },
    "transcode": {
      "bitrateKbps": 6000,
      "preset": "HD"
    }
  }
}`),
	}
	updateRequest := ocimock.MustJSONFixture[mediaservicessdk.UpdateMediaWorkflowConfigurationDetails](t, `{
  "displayName": "configuration-alpha-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[mediaservicessdk.MediaWorkflowConfiguration](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "configuration-alpha-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "parameters": {
    "thumbnails": {
      "enabled": true
    },
    "transcode": {
      "bitrateKbps": 6000,
      "preset": "HD"
    }
  }
}`)
	updatedReadStates := []mediaservicessdk.MediaWorkflowConfiguration{
		ocimock.MustOCIResponseFixture[mediaservicessdk.MediaWorkflowConfiguration](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "configuration-alpha-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "parameters": {
    "thumbnails": {
      "enabled": true
    },
    "transcode": {
      "bitrateKbps": 6000,
      "preset": "HD"
    }
  }
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		mediaservicessdk.MediaWorkflowConfiguration,
		mediaservicessdk.CreateMediaWorkflowConfigurationDetails,
		mediaservicessdk.UpdateMediaWorkflowConfigurationDetails,
	]{
		CollectionPath:     "/20211101/mediaWorkflowConfigurations",
		ItemPath:           "/20211101/mediaWorkflowConfigurations/<ocid:2>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		CreatedReadStates:  createdReadStates,
		UpdatedReadStates:  updatedReadStates,
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ mediaservicessdk.CreateMediaWorkflowConfigurationDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ mediaservicessdk.MediaWorkflowConfiguration) error {
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
			t.Errorf("close MediaWorkflowConfiguration OCI mock: %v", err)
		}
	})
	sdkClient := mediaservicessdk.MediaServicesClient{BaseClient: session.BaseClient()}
	client := newMediaWorkflowConfigurationServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*mediaservicesv1beta1.MediaWorkflowConfiguration]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *mediaservicesv1beta1.MediaWorkflowConfiguration) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				len(current.Status.Parameters) != 2 ||
				current.Status.Parameters["thumbnails"].Raw == nil ||
				current.Status.Parameters["transcode"].Raw == nil {
				return fmt.Errorf("created MediaWorkflowConfiguration status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *mediaservicesv1beta1.MediaWorkflowConfiguration) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *mediaservicesv1beta1.MediaWorkflowConfiguration) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("updated MediaWorkflowConfiguration status = %+v", current.Status)
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
