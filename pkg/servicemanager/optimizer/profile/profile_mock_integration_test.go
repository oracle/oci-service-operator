/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package profile

import (
	"context"
	"fmt"
	optimizersdk "github.com/oracle/oci-go-sdk/v65/optimizer"
	optimizerv1beta1 "github.com/oracle/oci-service-operator/api/optimizer/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationProfileLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newProfileRuntimeTestResource()
	ocimock.InitializeResource(resource, "mock-profile")
	resource.Spec = ocimock.MustJSONFixture[optimizerv1beta1.ProfileSpec](t, `{
  "aggregationIntervalInDays": 60,
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "runtime profile",
  "freeformTags": {
    "env": "dev"
  },
  "levelsConfiguration": {
    "items": [
      {
        "level": "HIGH",
        "recommendationId": "\u003cocid:2\u003e"
      }
    ]
  },
  "name": "profile-alpha",
  "targetCompartments": {
    "items": [
      "\u003cocid:3\u003e"
    ]
  },
  "targetTags": {
    "items": [
      {
        "tagDefinitionName": "CostCenter",
        "tagNamespaceName": "Operations",
        "tagValueType": "VALUE",
        "tagValues": [
          "42"
        ]
      }
    ]
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "runtime profile-updated"
}`)
	createRequest := ocimock.MustJSONFixture[optimizersdk.CreateProfileDetails](t, `{
  "aggregationIntervalInDays": 60,
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "runtime profile",
  "freeformTags": {
    "env": "dev"
  },
  "levelsConfiguration": {
    "items": [
      {
        "level": "HIGH",
        "recommendationId": "\u003cocid:2\u003e"
      }
    ]
  },
  "name": "profile-alpha",
  "targetCompartments": {
    "items": [
      "\u003cocid:3\u003e"
    ]
  },
  "targetTags": {
    "items": [
      {
        "tagDefinitionName": "CostCenter",
        "tagNamespaceName": "Operations",
        "tagValueType": "VALUE",
        "tagValues": [
          "42"
        ]
      }
    ]
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[optimizersdk.Profile](t, `{
  "aggregationIntervalInDays": 60,
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "runtime profile",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:4\u003e",
  "levelsConfiguration": {
    "items": [
      {
        "level": "HIGH",
        "recommendationId": "\u003cocid:2\u003e"
      }
    ]
  },
  "lifecycleState": "ACTIVE",
  "name": "profile-alpha",
  "targetCompartments": {
    "items": [
      "\u003cocid:3\u003e"
    ]
  },
  "targetTags": {
    "items": [
      {
        "tagDefinitionName": "CostCenter",
        "tagNamespaceName": "Operations",
        "tagValueType": "VALUE",
        "tagValues": [
          "42"
        ]
      }
    ]
  }
}`)
	updateRequest := ocimock.MustJSONFixture[optimizersdk.UpdateProfileDetails](t, `{
  "description": "runtime profile-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[optimizersdk.Profile](t, `{
  "aggregationIntervalInDays": 60,
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "runtime profile-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:4\u003e",
  "levelsConfiguration": {
    "items": [
      {
        "level": "HIGH",
        "recommendationId": "\u003cocid:2\u003e"
      }
    ]
  },
  "lifecycleState": "ACTIVE",
  "name": "profile-alpha",
  "targetCompartments": {
    "items": [
      "\u003cocid:3\u003e"
    ]
  },
  "targetTags": {
    "items": [
      {
        "tagDefinitionName": "CostCenter",
        "tagNamespaceName": "Operations",
        "tagValueType": "VALUE",
        "tagValues": [
          "42"
        ]
      }
    ]
  }
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		optimizersdk.Profile,
		optimizersdk.CreateProfileDetails,
		optimizersdk.UpdateProfileDetails,
	]{
		CollectionPath:    "/20200606/profiles",
		ItemPath:          "/20200606/profiles/<ocid:4>",
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
		ValidateCreate: func(request ocimock.Request, _ optimizersdk.CreateProfileDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ optimizersdk.Profile) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20200606", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Profile OCI mock: %v", err)
		}
	})
	sdkClient := optimizersdk.OptimizerClient{BaseClient: session.BaseClient()}
	manager := &ProfileServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newProfileDefaultRuntimeHooks(sdkClient)
	applyProfileRuntimeHooks(&hooks)
	client := wrapProfileGeneratedClient(hooks, defaultProfileServiceClient{ServiceClient: generatedruntime.NewServiceClient[*optimizerv1beta1.Profile](buildProfileGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*optimizerv1beta1.Profile]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *optimizerv1beta1.Profile) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.AggregationIntervalInDays, current.Spec.AggregationIntervalInDays) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.LevelsConfiguration, current.Spec.LevelsConfiguration) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.TargetCompartments, current.Spec.TargetCompartments) ||
				!reflect.DeepEqual(current.Status.TargetTags, current.Spec.TargetTags) {
				return fmt.Errorf("created Profile status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *optimizerv1beta1.Profile) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *optimizerv1beta1.Profile) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated Profile status = %+v", current.Status)
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
