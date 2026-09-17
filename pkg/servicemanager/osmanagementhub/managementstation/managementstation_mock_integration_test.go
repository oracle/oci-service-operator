/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package managementstation

import (
	"context"
	"fmt"
	osmanagementhubsdk "github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationManagementStationLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := testManagementStationResource()
	ocimock.InitializeResource(resource, "mock-managementstation")
	resource.Spec = ocimock.MustJSONFixture[osmanagementhubv1beta1.ManagementStationSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "desired description",
  "displayName": "station",
  "freeformTags": {
    "owner": "osok"
  },
  "hostname": "station.example.com",
  "isAutoConfigEnabled": false,
  "mirror": {
    "directory": "/var/lib/osmh",
    "isSslverifyEnabled": false,
    "port": "8080",
    "sslcert": "/etc/pki/osmh.crt",
    "sslport": "8443"
  },
  "proxy": {
    "forward": "https://updates.example.com",
    "hosts": [
      "proxy.internal"
    ],
    "isEnabled": false,
    "port": "3128"
  }
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "desired description-updated"
}`)
	createRequest := ocimock.MustJSONFixture[osmanagementhubsdk.CreateManagementStationDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "desired description",
  "displayName": "station",
  "freeformTags": {
    "owner": "osok"
  },
  "hostname": "station.example.com",
  "isAutoConfigEnabled": false,
  "mirror": {
    "directory": "/var/lib/osmh",
    "isSslverifyEnabled": false,
    "port": "8080",
    "sslcert": "/etc/pki/osmh.crt",
    "sslport": "8443"
  },
  "proxy": {
    "forward": "https://updates.example.com",
    "hosts": [
      "proxy.internal"
    ],
    "isEnabled": false,
    "port": "3128"
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[osmanagementhubsdk.ManagementStation](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "desired description",
  "displayName": "station",
  "freeformTags": {
    "owner": "osok"
  },
  "hostname": "station.example.com",
  "id": "\u003cocid:2\u003e",
  "isAutoConfigEnabled": false,
  "lifecycleState": "ACTIVE",
  "mirror": {
    "directory": "/var/lib/osmh",
    "isSslverifyEnabled": false,
    "port": "8080",
    "sslcert": "/etc/pki/osmh.crt",
    "sslport": "8443"
  },
  "proxy": {
    "forward": "https://updates.example.com",
    "hosts": [
      "proxy.internal"
    ],
    "isEnabled": false,
    "port": "3128"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[osmanagementhubsdk.UpdateManagementStationDetails](t, `{
  "description": "desired description-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[osmanagementhubsdk.ManagementStation](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "desired description-updated",
  "displayName": "station",
  "freeformTags": {
    "owner": "osok"
  },
  "hostname": "station.example.com",
  "id": "\u003cocid:2\u003e",
  "isAutoConfigEnabled": false,
  "lifecycleState": "ACTIVE",
  "mirror": {
    "directory": "/var/lib/osmh",
    "isSslverifyEnabled": false,
    "port": "8080",
    "sslcert": "/etc/pki/osmh.crt",
    "sslport": "8443"
  },
  "proxy": {
    "forward": "https://updates.example.com",
    "hosts": [
      "proxy.internal"
    ],
    "isEnabled": false,
    "port": "3128"
  }
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		osmanagementhubsdk.ManagementStation,
		osmanagementhubsdk.CreateManagementStationDetails,
		osmanagementhubsdk.UpdateManagementStationDetails,
	]{
		CollectionPath:    "/20220901/managementStations",
		ItemPath:          "/20220901/managementStations/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      200,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ osmanagementhubsdk.CreateManagementStationDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ osmanagementhubsdk.ManagementStation) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20220901", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close ManagementStation OCI mock: %v", err)
		}
	})
	sdkClient := osmanagementhubsdk.ManagementStationClient{BaseClient: session.BaseClient()}
	manager := &ManagementStationServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newManagementStationDefaultRuntimeHooks(sdkClient)
	applyManagementStationRuntimeHooks(manager, &hooks)
	client := wrapManagementStationGeneratedClient(hooks, defaultManagementStationServiceClient{ServiceClient: generatedruntime.NewServiceClient[*osmanagementhubv1beta1.ManagementStation](buildManagementStationGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*osmanagementhubv1beta1.ManagementStation]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *osmanagementhubv1beta1.ManagementStation) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Hostname, current.Spec.Hostname) ||
				!reflect.DeepEqual(current.Status.IsAutoConfigEnabled, current.Spec.IsAutoConfigEnabled) ||
				!reflect.DeepEqual(current.Status.Mirror, current.Spec.Mirror) ||
				!reflect.DeepEqual(current.Status.Proxy, current.Spec.Proxy) {
				return fmt.Errorf("created ManagementStation status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *osmanagementhubv1beta1.ManagementStation) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *osmanagementhubv1beta1.ManagementStation) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated ManagementStation status = %+v", current.Status)
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
