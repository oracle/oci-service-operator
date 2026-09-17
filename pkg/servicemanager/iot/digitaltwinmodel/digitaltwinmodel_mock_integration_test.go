/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package digitaltwinmodel

import (
	"context"
	"fmt"
	iotsdk "github.com/oracle/oci-go-sdk/v65/iot"
	iotv1beta1 "github.com/oracle/oci-service-operator/api/iot/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"net/http"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationDigitalTwinModelEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &iotv1beta1.DigitalTwinModel{}
	ocimock.InitializeResource(resource, "mock-digitaltwinmodel")
	resource.Spec = ocimock.MustJSONFixture[iotv1beta1.DigitalTwinModelSpec](t, `{
  "description": "OSOK recorded digital twin model",
  "displayName": "osok-mock-digital-twin-model",
  "freeformTags": {
    "osok-mock": "create"
  },
  "iotDomainId": "\u003cocid:1\u003e",
  "spec": {
    "@context": "dtmi:dtdl:context;3",
    "@id": "dtmi:com:oracle:osok:MockLifecycleThermostat;1",
    "@type": "Interface",
    "contents": [
      {
        "@type": "Property",
        "name": "temperature",
        "schema": "double"
      }
    ],
    "displayName": "OSOK Mock Thermostat"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "OSOK recorded digital twin model updated",
  "displayName": "osok-mock-digital-twin-model",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[iotsdk.CreateDigitalTwinModelDetails](t, `{
  "description": "OSOK recorded digital twin model",
  "displayName": "osok-mock-digital-twin-model",
  "freeformTags": {
    "osok-mock": "create"
  },
  "iotDomainId": "\u003cocid:1\u003e",
  "spec": {
    "@context": "dtmi:dtdl:context;3",
    "@id": "dtmi:com:oracle:osok:MockLifecycleThermostat;1",
    "@type": "Interface",
    "contents": [
      {
        "@type": "Property",
        "name": "temperature",
        "schema": "double"
      }
    ],
    "displayName": "OSOK Mock Thermostat"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[iotsdk.DigitalTwinModel](t, `{
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:52:20.926Z"
    }
  },
  "description": "OSOK recorded digital twin model",
  "displayName": "osok-mock-digital-twin-model",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "iotDomainId": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "specUri": "dtmi:com:oracle:osok:MockLifecycleThermostat;1",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:52:21.154Z",
  "timeUpdated": "2026-09-02T20:52:21.154Z"
}`)
	createdReadStates := []iotsdk.DigitalTwinModel{
		ocimock.MustOCIResponseFixture[iotsdk.DigitalTwinModel](t, `{
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:52:20.926Z"
    }
  },
  "description": "OSOK recorded digital twin model",
  "displayName": "osok-mock-digital-twin-model",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "iotDomainId": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "specUri": "dtmi:com:oracle:osok:MockLifecycleThermostat;1",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:52:21.154Z",
  "timeUpdated": "2026-09-02T20:52:21.154Z"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[iotsdk.UpdateDigitalTwinModelDetails](t, `{
  "description": "OSOK recorded digital twin model updated",
  "displayName": "osok-mock-digital-twin-model",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[iotsdk.DigitalTwinModel](t, `{
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:52:20.926Z"
    }
  },
  "description": "OSOK recorded digital twin model updated",
  "displayName": "osok-mock-digital-twin-model",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "iotDomainId": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "specUri": "dtmi:com:oracle:osok:MockLifecycleThermostat;1",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:52:21.154Z",
  "timeUpdated": "2026-09-02T20:52:25.360Z"
}`)
	updatedReadStates := []iotsdk.DigitalTwinModel{
		ocimock.MustOCIResponseFixture[iotsdk.DigitalTwinModel](t, `{
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:52:20.926Z"
    }
  },
  "description": "OSOK recorded digital twin model updated",
  "displayName": "osok-mock-digital-twin-model",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "iotDomainId": "<ocid:1>",
  "lifecycleState": "ACTIVE",
  "specUri": "dtmi:com:oracle:osok:MockLifecycleThermostat;1",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:52:21.154Z",
  "timeUpdated": "2026-09-02T20:52:25.360Z"
}`),
	}
	deletedReadStates := []iotsdk.DigitalTwinModel{
		ocimock.MustOCIResponseFixture[iotsdk.DigitalTwinModel](t, `{
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:52:20.926Z"
    }
  },
  "description": "OSOK recorded digital twin model updated",
  "displayName": "osok-mock-digital-twin-model",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "iotDomainId": "<ocid:1>",
  "lifecycleState": "DELETED",
  "specUri": "dtmi:com:oracle:osok:MockLifecycleThermostat;1",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:52:21.154Z",
  "timeUpdated": "2026-09-02T20:52:28.938Z"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		iotsdk.DigitalTwinModel,
		iotsdk.CreateDigitalTwinModelDetails,
		iotsdk.UpdateDigitalTwinModelDetails,
	]{
		CollectionPath:    "/20250531/digitalTwinModels",
		ItemPath:          "/20250531/digitalTwinModels/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		CreatedReadStates: createdReadStates,
		UpdatedReadStates: updatedReadStates,
		DeletedReadStates: deletedReadStates,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      204,
		ValidateCreate: func(request ocimock.Request, _ iotsdk.CreateDigitalTwinModelDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ iotsdk.DigitalTwinModel) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
		AdditionalRoutes: []ocimock.Route{{
			Name:         "read model specification",
			Method:       http.MethodGet,
			Path:         "/20250531/digitalTwinModels/<ocid:2>/spec",
			MinimumCalls: 2,
			Respond: func(request ocimock.Request) (ocimock.Response, error) {
				if len(request.Body) != 0 {
					return ocimock.Response{}, fmt.Errorf("model specification request body = %s", request.Body)
				}
				return ocimock.Response{
					StatusCode: http.StatusOK,
					Body:       []byte(`{"@context":"dtmi:dtdl:context;3","@id":"dtmi:com:oracle:osok:MockLifecycleThermostat;1","@type":"Interface","contents":[{"@id":"dtmi:com:oracle:osok:MockLifecycleThermostat:_contents:__temperature;1","@type":"Property","name":"temperature","schema":"double"}],"displayName":"OSOK Mock Thermostat"}`),
				}, nil
			},
		}},
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
			t.Errorf("close DigitalTwinModel OCI mock: %v", err)
		}
	})
	sdkClient := iotsdk.IotClient{BaseClient: session.BaseClient()}
	client := newDigitalTwinModelServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*iotv1beta1.DigitalTwinModel]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *iotv1beta1.DigitalTwinModel) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IotDomainId, current.Spec.IotDomainId) {
				return fmt.Errorf("created DigitalTwinModel status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *iotv1beta1.DigitalTwinModel) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *iotv1beta1.DigitalTwinModel) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated DigitalTwinModel status = %+v", current.Status)
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
