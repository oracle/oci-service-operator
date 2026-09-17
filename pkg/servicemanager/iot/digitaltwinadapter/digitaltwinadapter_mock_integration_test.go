/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package digitaltwinadapter

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
func TestMockIntegrationDigitalTwinAdapterEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &iotv1beta1.DigitalTwinAdapter{}
	ocimock.InitializeResource(resource, "mock-digitaltwinadapter")
	resource.Spec = ocimock.MustJSONFixture[iotv1beta1.DigitalTwinAdapterSpec](t, `{
  "description": "OSOK recorded digital twin adapter",
  "digitalTwinModelId": "\u003cocid:1\u003e",
  "digitalTwinModelSpecUri": "dtmi:com:oracle:osok:MockThermostat;1",
  "displayName": "osok-mock-digital-twin-adapter",
  "freeformTags": {
    "osok-mock": "create"
  },
  "inboundEnvelope": {
    "envelopeMapping": {
      "timeObserved": "$.time"
    },
    "referenceEndpoint": "device/temperature",
    "referencePayload": {
      "data": {
        "temperature": 72,
        "time": "2026-01-01T00:00:00Z"
      },
      "dataFormat": "JSON"
    }
  },
  "inboundRoutes": [
    {
      "condition": "*",
      "description": "default mock route",
      "payloadMapping": {
        "$.temperature": "$.temperature"
      },
      "referencePayload": {
        "data": {
          "temperature": 72
        },
        "dataFormat": "JSON"
      }
    }
  ],
  "iotDomainId": "\u003cocid:2\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "OSOK recorded digital twin adapter updated",
  "displayName": "osok-mock-digital-twin-adapter",
  "freeformTags": {
    "osok-mock": "update"
  },
  "inboundEnvelope": {
    "envelopeMapping": {
      "timeObserved": "$.time"
    },
    "referenceEndpoint": "device/temperature",
    "referencePayload": {
      "data": {
        "temperature": 72,
        "time": "2026-01-01T00:00:00Z"
      },
      "dataFormat": "JSON"
    }
  },
  "inboundRoutes": [
    {
      "condition": "*",
      "description": "default mock route",
      "payloadMapping": {
        "$.temperature": "$.temperature"
      },
      "referencePayload": {
        "data": {
          "temperature": 72
        },
        "dataFormat": "JSON"
      }
    }
  ]
}`)
	createRequest := ocimock.MustJSONFixture[iotsdk.CreateDigitalTwinAdapterDetails](t, `{
  "description": "OSOK recorded digital twin adapter",
  "digitalTwinModelId": "\u003cocid:1\u003e",
  "digitalTwinModelSpecUri": "dtmi:com:oracle:osok:MockThermostat;1",
  "displayName": "osok-mock-digital-twin-adapter",
  "freeformTags": {
    "osok-mock": "create"
  },
  "inboundEnvelope": {
    "envelopeMapping": {
      "timeObserved": "$.time"
    },
    "referenceEndpoint": "device/temperature",
    "referencePayload": {
      "data": {
        "temperature": 72,
        "time": "2026-01-01T00:00:00Z"
      },
      "dataFormat": "JSON"
    }
  },
  "inboundRoutes": [
    {
      "condition": "*",
      "description": "default mock route",
      "payloadMapping": {
        "$.temperature": "$.temperature"
      },
      "referencePayload": {
        "data": {
          "temperature": 72
        },
        "dataFormat": "JSON"
      }
    }
  ],
  "iotDomainId": "\u003cocid:2\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[iotsdk.DigitalTwinAdapter](t, `{
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T21:11:45.036Z"
    }
  },
  "description": "OSOK recorded digital twin adapter",
  "digitalTwinModelId": "<ocid:1>",
  "digitalTwinModelSpecUri": "dtmi:com:oracle:osok:MockThermostat;1",
  "displayName": "osok-mock-digital-twin-adapter",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:3>",
  "inboundEnvelope": {
    "envelopeMapping": {
      "timeObserved": "$.time"
    },
    "referenceEndpoint": "device/temperature",
    "referencePayload": {
      "data": {
        "temperature": 72,
        "time": "2026-01-01T00:00:00Z"
      },
      "dataFormat": "JSON"
    }
  },
  "inboundRoutes": [
    {
      "condition": "*",
      "description": "default mock route",
      "payloadMapping": {
        "$.temperature": "$.temperature"
      },
      "referencePayload": {
        "data": {
          "temperature": 72
        },
        "dataFormat": "JSON"
      }
    }
  ],
  "iotDomainId": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-02T21:11:45.793Z",
  "timeUpdated": "2026-09-02T21:11:45.793Z"
}`)
	updateRequest := ocimock.MustJSONFixture[iotsdk.UpdateDigitalTwinAdapterDetails](t, `{
  "description": "OSOK recorded digital twin adapter updated",
  "displayName": "osok-mock-digital-twin-adapter",
  "freeformTags": {
    "osok-mock": "update"
  },
  "inboundEnvelope": {
    "envelopeMapping": {
      "timeObserved": "$.time"
    },
    "referenceEndpoint": "device/temperature",
    "referencePayload": {
      "data": {
        "temperature": 72,
        "time": "2026-01-01T00:00:00Z"
      },
      "dataFormat": "JSON"
    }
  },
  "inboundRoutes": [
    {
      "condition": "*",
      "description": "default mock route",
      "payloadMapping": {
        "$.temperature": "$.temperature"
      },
      "referencePayload": {
        "data": {
          "temperature": 72
        },
        "dataFormat": "JSON"
      }
    }
  ]
}`)
	updatedState := ocimock.MustOCIResponseFixture[iotsdk.DigitalTwinAdapter](t, `{
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T21:11:45.036Z"
    }
  },
  "description": "OSOK recorded digital twin adapter updated",
  "digitalTwinModelId": "<ocid:1>",
  "digitalTwinModelSpecUri": "dtmi:com:oracle:osok:MockThermostat;1",
  "displayName": "osok-mock-digital-twin-adapter",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "inboundEnvelope": {
    "envelopeMapping": {
      "timeObserved": "$.time"
    },
    "referenceEndpoint": "device/temperature",
    "referencePayload": {
      "data": {
        "temperature": 72,
        "time": "2026-01-01T00:00:00Z"
      },
      "dataFormat": "JSON"
    }
  },
  "inboundRoutes": [
    {
      "condition": "*",
      "description": "default mock route",
      "payloadMapping": {
        "$.temperature": "$.temperature"
      },
      "referencePayload": {
        "data": {
          "temperature": 72
        },
        "dataFormat": "JSON"
      }
    }
  ],
  "iotDomainId": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-02T21:11:45.793Z",
  "timeUpdated": "2026-09-02T21:11:48.908Z"
}`)
	deletedState := ocimock.MustOCIResponseFixture[iotsdk.DigitalTwinAdapter](t, `{
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T21:11:45.036Z"
    }
  },
  "description": "OSOK recorded digital twin adapter updated",
  "digitalTwinModelId": "<ocid:1>",
  "digitalTwinModelSpecUri": "dtmi:com:oracle:osok:MockThermostat;1",
  "displayName": "osok-mock-digital-twin-adapter",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:3>",
  "inboundEnvelope": {
    "envelopeMapping": {
      "timeObserved": "$.time"
    },
    "referenceEndpoint": "device/temperature",
    "referencePayload": {
      "data": {
        "temperature": 72,
        "time": "2026-01-01T00:00:00Z"
      },
      "dataFormat": "JSON"
    }
  },
  "inboundRoutes": [
    {
      "condition": "*",
      "description": "default mock route",
      "payloadMapping": {
        "$.temperature": "$.temperature"
      },
      "referencePayload": {
        "data": {
          "temperature": 72
        },
        "dataFormat": "JSON"
      }
    }
  ],
  "iotDomainId": "<ocid:2>",
  "lifecycleState": "DELETED",
  "systemTags": {},
  "timeCreated": "2026-09-02T21:11:45.793Z",
  "timeUpdated": "2026-09-02T21:11:52.208Z"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		iotsdk.DigitalTwinAdapter,
		iotsdk.CreateDigitalTwinAdapterDetails,
		iotsdk.UpdateDigitalTwinAdapterDetails,
	]{
		CollectionPath:    "/20250531/digitalTwinAdapters",
		ItemPath:          "/20250531/digitalTwinAdapters/<ocid:3>",
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
		ValidateCreate: func(request ocimock.Request, _ iotsdk.CreateDigitalTwinAdapterDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ iotsdk.DigitalTwinAdapter) error {
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
			t.Errorf("close DigitalTwinAdapter OCI mock: %v", err)
		}
	})
	sdkClient := iotsdk.IotClient{BaseClient: session.BaseClient()}
	client := newDigitalTwinAdapterServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*iotv1beta1.DigitalTwinAdapter]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *iotv1beta1.DigitalTwinAdapter) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DigitalTwinModelId, current.Spec.DigitalTwinModelId) ||
				!reflect.DeepEqual(current.Status.DigitalTwinModelSpecUri, current.Spec.DigitalTwinModelSpecUri) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.InboundEnvelope, current.Spec.InboundEnvelope) ||
				!reflect.DeepEqual(current.Status.InboundRoutes, current.Spec.InboundRoutes) ||
				!reflect.DeepEqual(current.Status.IotDomainId, current.Spec.IotDomainId) {
				return fmt.Errorf("created DigitalTwinAdapter status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *iotv1beta1.DigitalTwinAdapter) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *iotv1beta1.DigitalTwinAdapter) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.InboundEnvelope, current.Spec.InboundEnvelope) ||
				!reflect.DeepEqual(current.Status.InboundRoutes, current.Spec.InboundRoutes) {
				return fmt.Errorf("updated DigitalTwinAdapter status = %+v", current.Status)
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
