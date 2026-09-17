/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package odainstance

import (
	"context"
	"fmt"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationOdaInstanceLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newOdaInstanceTestResource()
	ocimock.InitializeResource(resource, "mock-odainstance")
	resource.Spec = ocimock.MustJSONFixture[odav1beta1.OdaInstanceSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "ODA description",
  "displayName": "oda-sample",
  "freeformTags": {
    "env": "test"
  },
  "shapeName": "DEVELOPMENT"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "ODA description updated"
}`)
	createRequest := ocimock.MustJSONFixture[odasdk.CreateOdaInstanceDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "ODA description",
  "displayName": "oda-sample",
  "freeformTags": {
    "env": "test"
  },
  "shapeName": "DEVELOPMENT"
}`)
	createdState := ocimock.MustOCIResponseFixture[odasdk.OdaInstance](t, `{
  "id": "<ocid:2>",
  "compartmentId": "<ocid:1>",
  "shapeName": "DEVELOPMENT",
  "displayName": "oda-sample",
  "description": "ODA description",
  "lifecycleState": "ACTIVE",
  "freeformTags": {
    "env": "test"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "isRoleBasedAccess": false
}`)
	updateRequest := ocimock.MustJSONFixture[odasdk.UpdateOdaInstanceDetails](t, `{
  "description": "ODA description updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[odasdk.OdaInstance](t, `{
  "id": "<ocid:2>",
  "compartmentId": "<ocid:1>",
  "shapeName": "DEVELOPMENT",
  "displayName": "oda-sample",
  "description": "ODA description updated",
  "lifecycleState": "ACTIVE",
  "freeformTags": {
    "env": "test"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "isRoleBasedAccess": false
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		odasdk.OdaInstance,
		odasdk.CreateOdaInstanceDetails,
		odasdk.UpdateOdaInstanceDetails,
	]{
		CollectionPath:    "/20190506/odaInstances",
		ItemPath:          "/20190506/odaInstances/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeArray,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      200,
		UpdateStatus:      200,
		DeleteStatus:      202,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ odasdk.CreateOdaInstanceDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ odasdk.OdaInstance) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20190506", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close OdaInstance OCI mock: %v", err)
		}
	})
	sdkClient := odasdk.OdaClient{BaseClient: session.BaseClient()}
	manager := &OdaInstanceServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newOdaInstanceRuntimeHooks(manager, sdkClient)
	client := wrapOdaInstanceGeneratedClient(hooks, defaultOdaInstanceServiceClient{ServiceClient: generatedruntime.NewServiceClient[*odav1beta1.OdaInstance](buildOdaInstanceGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*odav1beta1.OdaInstance]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *odav1beta1.OdaInstance) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.ShapeName, current.Spec.ShapeName) {
				return fmt.Errorf("created OdaInstance status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *odav1beta1.OdaInstance) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *odav1beta1.OdaInstance) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated OdaInstance status = %+v", current.Status)
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
