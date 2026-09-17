/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package digitaltwininstance

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
func TestMockIntegrationDigitalTwinInstanceLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeDigitalTwinInstanceResource()
	ocimock.InitializeResource(resource, "mock-digitaltwininstance")
	resource.Spec = ocimock.MustJSONFixture[iotv1beta1.DigitalTwinInstanceSpec](t, `{
  "authId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description",
  "digitalTwinAdapterId": "\u003cocid:2\u003e",
  "digitalTwinModelId": "\u003cocid:3\u003e",
  "digitalTwinModelSpecUri": "dtmi:com:example:Thermostat;1",
  "displayName": "instance-sample",
  "externalKey": "device-001",
  "freeformTags": {
    "env": "test"
  },
  "iotDomainId": "\u003cocid:4\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "authId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description-updated",
  "digitalTwinAdapterId": "\u003cocid:2\u003e",
  "digitalTwinModelId": "\u003cocid:3\u003e",
  "digitalTwinModelSpecUri": "dtmi:com:example:Thermostat;1",
  "displayName": "instance-sample",
  "externalKey": "device-001",
  "freeformTags": {
    "env": "test"
  }
}`)
	createRequest := ocimock.MustJSONFixture[iotsdk.CreateDigitalTwinInstanceDetails](t, `{
  "authId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description",
  "digitalTwinAdapterId": "\u003cocid:2\u003e",
  "digitalTwinModelId": "\u003cocid:3\u003e",
  "digitalTwinModelSpecUri": "dtmi:com:example:Thermostat;1",
  "displayName": "instance-sample",
  "externalKey": "device-001",
  "freeformTags": {
    "env": "test"
  },
  "iotDomainId": "\u003cocid:4\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[iotsdk.DigitalTwinInstance](t, `{
  "authId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description",
  "digitalTwinAdapterId": "\u003cocid:2\u003e",
  "digitalTwinModelId": "\u003cocid:3\u003e",
  "digitalTwinModelSpecUri": "dtmi:com:example:Thermostat;1",
  "displayName": "instance-sample",
  "externalKey": "device-001",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:5\u003e",
  "iotDomainId": "\u003cocid:4\u003e",
  "lifecycleState": "ACTIVE"
}`)
	updateRequest := ocimock.MustJSONFixture[iotsdk.UpdateDigitalTwinInstanceDetails](t, `{
  "authId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description-updated",
  "digitalTwinAdapterId": "\u003cocid:2\u003e",
  "digitalTwinModelId": "\u003cocid:3\u003e",
  "digitalTwinModelSpecUri": "dtmi:com:example:Thermostat;1",
  "displayName": "instance-sample",
  "externalKey": "device-001",
  "freeformTags": {
    "env": "test"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[iotsdk.DigitalTwinInstance](t, `{
  "authId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description-updated",
  "digitalTwinAdapterId": "\u003cocid:2\u003e",
  "digitalTwinModelId": "\u003cocid:3\u003e",
  "digitalTwinModelSpecUri": "dtmi:com:example:Thermostat;1",
  "displayName": "instance-sample",
  "externalKey": "device-001",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:5\u003e",
  "iotDomainId": "\u003cocid:4\u003e",
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		iotsdk.DigitalTwinInstance,
		iotsdk.CreateDigitalTwinInstanceDetails,
		iotsdk.UpdateDigitalTwinInstanceDetails,
	]{
		CollectionPath:    "/20250531/digitalTwinInstances",
		ItemPath:          "/20250531/digitalTwinInstances/<ocid:5>",
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
		ValidateCreate: func(request ocimock.Request, _ iotsdk.CreateDigitalTwinInstanceDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ iotsdk.DigitalTwinInstance) error {
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
			t.Errorf("close DigitalTwinInstance OCI mock: %v", err)
		}
	})
	sdkClient := iotsdk.IotClient{BaseClient: session.BaseClient()}
	client := newDigitalTwinInstanceServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*iotv1beta1.DigitalTwinInstance]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *iotv1beta1.DigitalTwinInstance) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.AuthId, current.Spec.AuthId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DigitalTwinAdapterId, current.Spec.DigitalTwinAdapterId) ||
				!reflect.DeepEqual(current.Status.DigitalTwinModelId, current.Spec.DigitalTwinModelId) ||
				!reflect.DeepEqual(current.Status.DigitalTwinModelSpecUri, current.Spec.DigitalTwinModelSpecUri) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.ExternalKey, current.Spec.ExternalKey) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IotDomainId, current.Spec.IotDomainId) {
				return fmt.Errorf("created DigitalTwinInstance status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *iotv1beta1.DigitalTwinInstance) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *iotv1beta1.DigitalTwinInstance) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.AuthId, current.Spec.AuthId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DigitalTwinAdapterId, current.Spec.DigitalTwinAdapterId) ||
				!reflect.DeepEqual(current.Status.DigitalTwinModelId, current.Spec.DigitalTwinModelId) ||
				!reflect.DeepEqual(current.Status.DigitalTwinModelSpecUri, current.Spec.DigitalTwinModelSpecUri) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.ExternalKey, current.Spec.ExternalKey) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated DigitalTwinInstance status = %+v", current.Status)
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
