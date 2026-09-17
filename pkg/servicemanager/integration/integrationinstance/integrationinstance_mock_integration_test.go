/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package integrationinstance

import (
	"context"
	"fmt"
	integrationsdk "github.com/oracle/oci-go-sdk/v65/integration"
	integrationv1beta1 "github.com/oracle/oci-service-operator/api/integration/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationIntegrationInstanceLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newIntegrationInstanceTestResource()
	ocimock.InitializeResource(resource, "mock-integrationinstance")
	resource.Spec = ocimock.MustJSONFixture[integrationv1beta1.IntegrationInstanceSpec](t, `{
  "alternateCustomEndpoints": [],
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "integration-sample",
  "freeformTags": {
    "env": "test"
  },
  "integrationInstanceType": "ENTERPRISE",
  "isByol": false,
  "isFileServerEnabled": true,
  "isVisualBuilderEnabled": true,
  "messagePacks": 1,
  "shape": "DEVELOPMENT"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "integration-updated"
}`)
	createRequest := ocimock.MustJSONFixture[integrationsdk.CreateIntegrationInstanceDetails](t, `{
  "alternateCustomEndpoints": [],
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "integration-sample",
  "freeformTags": {
    "env": "test"
  },
  "integrationInstanceType": "ENTERPRISE",
  "isByol": false,
  "isFileServerEnabled": true,
  "isVisualBuilderEnabled": true,
  "messagePacks": 1,
  "shape": "DEVELOPMENT"
}`)
	createdState := ocimock.MustOCIResponseFixture[integrationsdk.IntegrationInstance](t, `{
  "id": "<ocid:2>",
  "displayName": "integration-sample",
  "compartmentId": "<ocid:1>",
  "integrationInstanceType": "ENTERPRISE",
  "isByol": false,
  "isDisasterRecoveryEnabled": false,
  "messagePacks": 1,
  "lifecycleState": "ACTIVE",
  "freeformTags": {
    "env": "test"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "isVisualBuilderEnabled": true,
  "isFileServerEnabled": true,
  "shape": "DEVELOPMENT"
}`)
	updateRequest := ocimock.MustJSONFixture[integrationsdk.UpdateIntegrationInstanceDetails](t, `{
  "displayName": "integration-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[integrationsdk.IntegrationInstance](t, `{
  "id": "<ocid:2>",
  "displayName": "integration-updated",
  "compartmentId": "<ocid:1>",
  "integrationInstanceType": "ENTERPRISE",
  "isByol": false,
  "isDisasterRecoveryEnabled": false,
  "messagePacks": 1,
  "lifecycleState": "ACTIVE",
  "freeformTags": {
    "env": "test"
  },
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "isVisualBuilderEnabled": true,
  "isFileServerEnabled": true,
  "shape": "DEVELOPMENT"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		integrationsdk.IntegrationInstance,
		integrationsdk.CreateIntegrationInstanceDetails,
		integrationsdk.UpdateIntegrationInstanceDetails,
	]{
		CollectionPath:    "/20190131/integrationInstances",
		ItemPath:          "/20190131/integrationInstances/<ocid:2>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeArray,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      202,
		UpdateStatus:      202,
		DeleteStatus:      202,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ integrationsdk.CreateIntegrationInstanceDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ integrationsdk.IntegrationInstance) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20190131", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close IntegrationInstance OCI mock: %v", err)
		}
	})
	sdkClient := integrationsdk.IntegrationInstanceClient{BaseClient: session.BaseClient()}
	hooks := newIntegrationInstanceDefaultRuntimeHooks(sdkClient)
	applyIntegrationInstanceRuntimeHooks(&hooks)
	client := newIntegrationInstanceRuntimeTestClient(hooks)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*integrationv1beta1.IntegrationInstance]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *integrationv1beta1.IntegrationInstance) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IntegrationInstanceType, current.Spec.IntegrationInstanceType) ||
				!reflect.DeepEqual(current.Status.IsByol, current.Spec.IsByol) ||
				!reflect.DeepEqual(current.Status.IsFileServerEnabled, current.Spec.IsFileServerEnabled) ||
				!reflect.DeepEqual(current.Status.IsVisualBuilderEnabled, current.Spec.IsVisualBuilderEnabled) ||
				!reflect.DeepEqual(current.Status.MessagePacks, current.Spec.MessagePacks) ||
				!reflect.DeepEqual(current.Status.Shape, current.Spec.Shape) {
				return fmt.Errorf("created IntegrationInstance status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *integrationv1beta1.IntegrationInstance) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *integrationv1beta1.IntegrationInstance) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("updated IntegrationInstance status = %+v", current.Status)
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
