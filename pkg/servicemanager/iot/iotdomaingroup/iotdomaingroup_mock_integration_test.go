/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package iotdomaingroup

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
func TestMockIntegrationIotDomainGroupEvidenceCRUD(t *testing.T) {
	t.Parallel()

	resource := &iotv1beta1.IotDomainGroup{}
	ocimock.InitializeResource(resource, "mock-iotdomaingroup")
	resource.Spec = ocimock.MustJSONFixture[iotv1beta1.IotDomainGroupSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "OSOK recorded IoT domain group",
  "displayName": "osok-mock-iot-domain-group",
  "freeformTags": {
    "osok-mock": "create"
  },
  "type": "LIGHTWEIGHT"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "OSOK recorded IoT domain group updated",
  "displayName": "osok-mock-iot-domain-group",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	createRequest := ocimock.MustJSONFixture[iotsdk.CreateIotDomainGroupDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "OSOK recorded IoT domain group",
  "displayName": "osok-mock-iot-domain-group",
  "freeformTags": {
    "osok-mock": "create"
  },
  "type": "LIGHTWEIGHT"
}`)
	createdState := ocimock.MustOCIResponseFixture[iotsdk.IotDomainGroup](t, `{
  "compartmentId": "<ocid:1>",
  "dataHost": "dsmyijd3s23eu.data.iot.us-ashburn-1.oci.oraclecloud.com",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:02:11.724Z"
    }
  },
  "description": "OSOK recorded IoT domain group",
  "displayName": "osok-mock-iot-domain-group",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:02:11.895Z",
  "timeUpdated": "2026-09-02T20:07:03.867Z",
  "type": "LIGHTWEIGHT"
}`)
	createdReadStates := []iotsdk.IotDomainGroup{
		ocimock.MustOCIResponseFixture[iotsdk.IotDomainGroup](t, `{
  "compartmentId": "<ocid:1>",
  "dataHost": "dsmyijd3s23eu.data.iot.us-ashburn-1.oci.oraclecloud.com",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:02:11.724Z"
    }
  },
  "description": "OSOK recorded IoT domain group",
  "displayName": "osok-mock-iot-domain-group",
  "freeformTags": {
    "osok-mock": "create"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:02:11.895Z",
  "timeUpdated": "2026-09-02T20:07:03.867Z",
  "type": "LIGHTWEIGHT"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[iotsdk.UpdateIotDomainGroupDetails](t, `{
  "description": "OSOK recorded IoT domain group updated",
  "displayName": "osok-mock-iot-domain-group",
  "freeformTags": {
    "osok-mock": "update"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[iotsdk.IotDomainGroup](t, `{
  "compartmentId": "<ocid:1>",
  "dataHost": "dsmyijd3s23eu.data.iot.us-ashburn-1.oci.oraclecloud.com",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:02:11.724Z"
    }
  },
  "description": "OSOK recorded IoT domain group updated",
  "displayName": "osok-mock-iot-domain-group",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:02:11.895Z",
  "timeUpdated": "2026-09-02T20:07:08.330Z",
  "type": "LIGHTWEIGHT"
}`)
	updatedReadStates := []iotsdk.IotDomainGroup{
		ocimock.MustOCIResponseFixture[iotsdk.IotDomainGroup](t, `{
  "compartmentId": "<ocid:1>",
  "dataHost": "dsmyijd3s23eu.data.iot.us-ashburn-1.oci.oraclecloud.com",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:02:11.724Z"
    }
  },
  "description": "OSOK recorded IoT domain group updated",
  "displayName": "osok-mock-iot-domain-group",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "ACTIVE",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:02:11.895Z",
  "timeUpdated": "2026-09-02T20:07:08.330Z",
  "type": "LIGHTWEIGHT"
}`),
	}
	deletedReadStates := []iotsdk.IotDomainGroup{
		ocimock.MustOCIResponseFixture[iotsdk.IotDomainGroup](t, `{
  "compartmentId": "<ocid:1>",
  "dataHost": "dsmyijd3s23eu.data.iot.us-ashburn-1.oci.oraclecloud.com",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:02:11.724Z"
    }
  },
  "description": "OSOK recorded IoT domain group updated",
  "displayName": "osok-mock-iot-domain-group",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "DELETING",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:02:11.895Z",
  "timeUpdated": "2026-09-02T20:07:11.897Z",
  "type": "LIGHTWEIGHT"
}`),
		ocimock.MustOCIResponseFixture[iotsdk.IotDomainGroup](t, `{
  "compartmentId": "<ocid:1>",
  "dataHost": "dsmyijd3s23eu.data.iot.us-ashburn-1.oci.oraclecloud.com",
  "definedTags": {
    "Oracle-Tags": {
      "CreatedBy": "<redacted>",
      "CreatedOn": "2026-09-02T20:02:11.724Z"
    }
  },
  "description": "OSOK recorded IoT domain group updated",
  "displayName": "osok-mock-iot-domain-group",
  "freeformTags": {
    "osok-mock": "update"
  },
  "id": "<ocid:2>",
  "lifecycleState": "DELETED",
  "systemTags": {},
  "timeCreated": "2026-09-02T20:02:11.895Z",
  "timeUpdated": "2026-09-02T20:10:43.680Z",
  "type": "LIGHTWEIGHT"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		iotsdk.IotDomainGroup,
		iotsdk.CreateIotDomainGroupDetails,
		iotsdk.UpdateIotDomainGroupDetails,
	]{
		CollectionPath:    "/20250531/iotDomainGroups",
		ItemPath:          "/20250531/iotDomainGroups/<ocid:2>",
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
		CreateStatus:      201,
		UpdateStatus:      202,
		DeleteStatus:      202,
		ValidateCreate: func(request ocimock.Request, _ iotsdk.CreateIotDomainGroupDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ iotsdk.IotDomainGroup) error {
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
			t.Errorf("close IotDomainGroup OCI mock: %v", err)
		}
	})
	sdkClient := iotsdk.IotClient{BaseClient: session.BaseClient()}
	client := newIotDomainGroupServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*iotv1beta1.IotDomainGroup]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *iotv1beta1.IotDomainGroup) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) {
				return fmt.Errorf("created IotDomainGroup status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *iotv1beta1.IotDomainGroup) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *iotv1beta1.IotDomainGroup) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated IotDomainGroup status = %+v", current.Status)
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
