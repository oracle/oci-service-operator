/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package digitaltwinrelationship

import (
	"context"
	"fmt"
	iotsdk "github.com/oracle/oci-go-sdk/v65/iot"
	iotv1beta1 "github.com/oracle/oci-service-operator/api/iot/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationDigitalTwinRelationshipLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeDigitalTwinRelationshipResource()
	ocimock.InitializeResource(resource, "mock-digitaltwinrelationship")
	resource.Spec = ocimock.MustJSONFixture[iotv1beta1.DigitalTwinRelationshipSpec](t, `{
  "content": {
    "enabled": true,
    "temperature": 72.5,
    "units": "fahrenheit"
  },
  "contentPath": "contains",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial relationship",
  "displayName": "source-contains-target",
  "freeformTags": {
    "env": "test"
  },
  "iotDomainId": "\u003cocid:1\u003e",
  "sourceDigitalTwinInstanceId": "\u003cocid:2\u003e",
  "targetDigitalTwinInstanceId": "\u003cocid:3\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "content": {
    "enabled": true,
    "temperature": 72.5,
    "units": "fahrenheit"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial relationship-updated",
  "displayName": "source-contains-target",
  "freeformTags": {
    "env": "test"
  }
}`)
	createRequest := ocimock.MustJSONFixture[iotsdk.CreateDigitalTwinRelationshipDetails](t, `{
  "content": {
    "enabled": true,
    "temperature": 72.5,
    "units": "fahrenheit"
  },
  "contentPath": "contains",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial relationship",
  "displayName": "source-contains-target",
  "freeformTags": {
    "env": "test"
  },
  "iotDomainId": "\u003cocid:1\u003e",
  "sourceDigitalTwinInstanceId": "\u003cocid:2\u003e",
  "targetDigitalTwinInstanceId": "\u003cocid:3\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[iotsdk.DigitalTwinRelationship](t, `{
  "content": {
    "enabled": true,
    "temperature": 72.5,
    "units": "fahrenheit"
  },
  "contentPath": "contains",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial relationship",
  "displayName": "source-contains-target",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:4\u003e",
  "iotDomainId": "\u003cocid:1\u003e",
  "lifecycleState": "ACTIVE",
  "sourceDigitalTwinInstanceId": "\u003cocid:2\u003e",
  "targetDigitalTwinInstanceId": "\u003cocid:3\u003e"
}`)
	updateRequest := ocimock.MustJSONFixture[iotsdk.UpdateDigitalTwinRelationshipDetails](t, `{
  "content": {
    "enabled": true,
    "temperature": 72.5,
    "units": "fahrenheit"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial relationship-updated",
  "displayName": "source-contains-target",
  "freeformTags": {
    "env": "test"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[iotsdk.DigitalTwinRelationship](t, `{
  "content": {
    "enabled": true,
    "temperature": 72.5,
    "units": "fahrenheit"
  },
  "contentPath": "contains",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial relationship-updated",
  "displayName": "source-contains-target",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:4\u003e",
  "iotDomainId": "\u003cocid:1\u003e",
  "lifecycleState": "ACTIVE",
  "sourceDigitalTwinInstanceId": "\u003cocid:2\u003e",
  "targetDigitalTwinInstanceId": "\u003cocid:3\u003e"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		iotsdk.DigitalTwinRelationship,
		iotsdk.CreateDigitalTwinRelationshipDetails,
		iotsdk.UpdateDigitalTwinRelationshipDetails,
	]{
		CollectionPath:    "/20250531/digitalTwinRelationships",
		ItemPath:          "/20250531/digitalTwinRelationships/<ocid:4>",
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
		ValidateCreate: func(request ocimock.Request, _ iotsdk.CreateDigitalTwinRelationshipDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ iotsdk.DigitalTwinRelationship) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20250531", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close DigitalTwinRelationship OCI mock: %v", err)
		}
	})
	sdkClient := iotsdk.IotClient{BaseClient: session.BaseClient()}
	client := newDigitalTwinRelationshipServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*iotv1beta1.DigitalTwinRelationship]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *iotv1beta1.DigitalTwinRelationship) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Content, current.Spec.Content) ||
				!reflect.DeepEqual(current.Status.ContentPath, current.Spec.ContentPath) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IotDomainId, current.Spec.IotDomainId) ||
				!reflect.DeepEqual(current.Status.SourceDigitalTwinInstanceId, current.Spec.SourceDigitalTwinInstanceId) ||
				!reflect.DeepEqual(current.Status.TargetDigitalTwinInstanceId, current.Spec.TargetDigitalTwinInstanceId) {
				return fmt.Errorf("created DigitalTwinRelationship status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *iotv1beta1.DigitalTwinRelationship) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *iotv1beta1.DigitalTwinRelationship) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Content, current.Spec.Content) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated DigitalTwinRelationship status = %+v", current.Status)
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
