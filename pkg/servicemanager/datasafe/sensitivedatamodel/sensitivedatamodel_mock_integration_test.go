/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package sensitivedatamodel

import (
	"context"
	"fmt"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationSensitiveDataModelLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeSensitiveDataModelResource()
	ocimock.InitializeResource(resource, "mock-sensitivedatamodel")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.SensitiveDataModelSpec](t, `{
  "appSuiteName": "GENERIC",
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "customer data model",
  "displayName": "customer-model",
  "freeformTags": {
    "env": "test"
  },
  "isAppDefinedRelationDiscoveryEnabled": true,
  "isIncludeAllSchemas": false,
  "isIncludeAllSensitiveTypes": true,
  "isSampleDataCollectionEnabled": false,
  "schemasForDiscovery": [
    "HR",
    "OE"
  ],
  "sensitiveTypeGroupIdsForDiscovery": [
    "\u003cocid:2\u003e"
  ],
  "sensitiveTypeIdsForDiscovery": [
    "\u003cocid:3\u003e"
  ],
  "tablesForDiscovery": [
    {
      "schemaName": "HR",
      "tableNames": [
        "EMPLOYEES",
        "JOBS"
      ]
    }
  ],
  "targetId": "\u003cocid:4\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "customer data model-updated"
}`)
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateSensitiveDataModelDetails](t, `{
  "appSuiteName": "GENERIC",
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "customer data model",
  "displayName": "customer-model",
  "freeformTags": {
    "env": "test"
  },
  "isAppDefinedRelationDiscoveryEnabled": true,
  "isIncludeAllSchemas": false,
  "isIncludeAllSensitiveTypes": true,
  "isSampleDataCollectionEnabled": false,
  "schemasForDiscovery": [
    "HR",
    "OE"
  ],
  "sensitiveTypeGroupIdsForDiscovery": [
    "\u003cocid:2\u003e"
  ],
  "sensitiveTypeIdsForDiscovery": [
    "\u003cocid:3\u003e"
  ],
  "tablesForDiscovery": [
    {
      "schemaName": "HR",
      "tableNames": [
        "EMPLOYEES",
        "JOBS"
      ]
    }
  ],
  "targetId": "\u003cocid:4\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.SensitiveDataModel](t, `{
  "appSuiteName": "GENERIC",
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "customer data model",
  "displayName": "customer-model",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:5\u003e",
  "isAppDefinedRelationDiscoveryEnabled": true,
  "isIncludeAllSchemas": false,
  "isIncludeAllSensitiveTypes": true,
  "isSampleDataCollectionEnabled": false,
  "lifecycleState": "ACTIVE",
  "schemasForDiscovery": [
    "HR",
    "OE"
  ],
  "sensitiveTypeGroupIdsForDiscovery": [
    "\u003cocid:2\u003e"
  ],
  "sensitiveTypeIdsForDiscovery": [
    "\u003cocid:3\u003e"
  ],
  "tablesForDiscovery": [
    {
      "schemaName": "HR",
      "tableNames": [
        "EMPLOYEES",
        "JOBS"
      ]
    }
  ],
  "targetId": "\u003cocid:4\u003e"
}`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateSensitiveDataModelDetails](t, `{
  "description": "customer data model-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.SensitiveDataModel](t, `{
  "appSuiteName": "GENERIC",
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "customer data model-updated",
  "displayName": "customer-model",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:5\u003e",
  "isAppDefinedRelationDiscoveryEnabled": true,
  "isIncludeAllSchemas": false,
  "isIncludeAllSensitiveTypes": true,
  "isSampleDataCollectionEnabled": false,
  "lifecycleState": "ACTIVE",
  "schemasForDiscovery": [
    "HR",
    "OE"
  ],
  "sensitiveTypeGroupIdsForDiscovery": [
    "\u003cocid:2\u003e"
  ],
  "sensitiveTypeIdsForDiscovery": [
    "\u003cocid:3\u003e"
  ],
  "tablesForDiscovery": [
    {
      "schemaName": "HR",
      "tableNames": [
        "EMPLOYEES",
        "JOBS"
      ]
    }
  ],
  "targetId": "\u003cocid:4\u003e"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		datasafesdk.SensitiveDataModel,
		datasafesdk.CreateSensitiveDataModelDetails,
		datasafesdk.UpdateSensitiveDataModelDetails,
	]{
		CollectionPath:    "/20181201/sensitiveDataModels",
		ItemPath:          "/20181201/sensitiveDataModels/<ocid:5>",
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
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateSensitiveDataModelDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ datasafesdk.SensitiveDataModel) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20181201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close SensitiveDataModel OCI mock: %v", err)
		}
	})
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	client := newSensitiveDataModelServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.SensitiveDataModel]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.SensitiveDataModel) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.AppSuiteName, current.Spec.AppSuiteName) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsAppDefinedRelationDiscoveryEnabled, current.Spec.IsAppDefinedRelationDiscoveryEnabled) ||
				!reflect.DeepEqual(current.Status.IsIncludeAllSchemas, current.Spec.IsIncludeAllSchemas) ||
				!reflect.DeepEqual(current.Status.IsIncludeAllSensitiveTypes, current.Spec.IsIncludeAllSensitiveTypes) ||
				!reflect.DeepEqual(current.Status.IsSampleDataCollectionEnabled, current.Spec.IsSampleDataCollectionEnabled) ||
				!reflect.DeepEqual(current.Status.SchemasForDiscovery, current.Spec.SchemasForDiscovery) ||
				!reflect.DeepEqual(current.Status.SensitiveTypeGroupIdsForDiscovery, current.Spec.SensitiveTypeGroupIdsForDiscovery) ||
				!reflect.DeepEqual(current.Status.SensitiveTypeIdsForDiscovery, current.Spec.SensitiveTypeIdsForDiscovery) ||
				!reflect.DeepEqual(current.Status.TablesForDiscovery, current.Spec.TablesForDiscovery) ||
				!reflect.DeepEqual(current.Status.TargetId, current.Spec.TargetId) {
				return fmt.Errorf("created SensitiveDataModel status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.SensitiveDataModel) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datasafev1beta1.SensitiveDataModel) error {
			if current.Status.Id != "<ocid:5>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:5>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated SensitiveDataModel status = %+v", current.Status)
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
