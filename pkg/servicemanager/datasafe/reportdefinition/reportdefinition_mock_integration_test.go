/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package reportdefinition

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
func TestMockIntegrationReportDefinitionLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeReportDefinitionResource()
	ocimock.InitializeResource(resource, "mock-reportdefinition")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.ReportDefinitionSpec](t, `{
  "columnFilters": [
    {
      "expressions": [
        "app"
      ],
      "fieldName": "userName",
      "isEnabled": true,
      "isHidden": false,
      "operator": "EQ"
    }
  ],
  "columnInfo": [
    {
      "applicableOperators": [
        "EQ",
        "IN"
      ],
      "dataType": "STRING",
      "displayName": "User",
      "displayOrder": 1,
      "fieldName": "userName",
      "isHidden": false,
      "isVirtual": false
    }
  ],
  "columnSortings": [
    {
      "fieldName": "userName",
      "isAscending": true,
      "sortingOrder": 1
    }
  ],
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial report",
  "displayName": "monthly-audit",
  "freeformTags": {
    "env": "dev"
  },
  "parentId": "\u003cocid:2\u003e",
  "summary": [
    {
      "countOf": "id",
      "displayOrder": 1,
      "groupByFieldName": "userName",
      "isHidden": false,
      "name": "Users"
    }
  ]
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "columnFilters": [
    {
      "expressions": [
        "app"
      ],
      "fieldName": "userName",
      "isEnabled": true,
      "isHidden": false,
      "operator": "EQ"
    }
  ],
  "columnInfo": [
    {
      "applicableOperators": [
        "EQ",
        "IN"
      ],
      "dataType": "STRING",
      "displayName": "User",
      "displayOrder": 1,
      "fieldName": "userName",
      "isHidden": false,
      "isVirtual": false
    }
  ],
  "columnSortings": [
    {
      "fieldName": "userName",
      "isAscending": true,
      "sortingOrder": 1
    }
  ],
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial report-updated",
  "displayName": "monthly-audit",
  "freeformTags": {
    "env": "dev"
  },
  "summary": [
    {
      "countOf": "id",
      "displayOrder": 1,
      "groupByFieldName": "userName",
      "isHidden": false,
      "name": "Users"
    }
  ]
}`)
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateReportDefinitionDetails](t, `{
  "columnFilters": [
    {
      "expressions": [
        "app"
      ],
      "fieldName": "userName",
      "isEnabled": true,
      "isHidden": false,
      "operator": "EQ"
    }
  ],
  "columnInfo": [
    {
      "applicableOperators": [
        "EQ",
        "IN"
      ],
      "dataType": "STRING",
      "displayName": "User",
      "displayOrder": 1,
      "fieldName": "userName",
      "isHidden": false,
      "isVirtual": false
    }
  ],
  "columnSortings": [
    {
      "fieldName": "userName",
      "isAscending": true,
      "sortingOrder": 1
    }
  ],
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial report",
  "displayName": "monthly-audit",
  "freeformTags": {
    "env": "dev"
  },
  "parentId": "\u003cocid:2\u003e",
  "summary": [
    {
      "countOf": "id",
      "displayOrder": 1,
      "groupByFieldName": "userName",
      "isHidden": false,
      "name": "Users"
    }
  ]
}`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.ReportDefinition](t, `{
  "columnFilters": [
    {
      "expressions": [
        "app"
      ],
      "fieldName": "userName",
      "isEnabled": true,
      "isHidden": false,
      "operator": "EQ"
    }
  ],
  "columnInfo": [
    {
      "applicableOperators": [
        "EQ",
        "IN"
      ],
      "dataType": "STRING",
      "displayName": "User",
      "displayOrder": 1,
      "fieldName": "userName",
      "isHidden": false
    }
  ],
  "columnSortings": [
    {
      "fieldName": "userName",
      "isAscending": true,
      "sortingOrder": 1
    }
  ],
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial report",
  "displayName": "monthly-audit",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "parentId": "\u003cocid:2\u003e",
  "summary": [
    {
      "countOf": "id",
      "displayOrder": 1,
      "groupByFieldName": "userName",
      "name": "Users"
    }
  ]
}`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateReportDefinitionDetails](t, `{
  "columnFilters": [
    {
      "expressions": [
        "app"
      ],
      "fieldName": "userName",
      "isEnabled": true,
      "isHidden": false,
      "operator": "EQ"
    }
  ],
  "columnInfo": [
    {
      "applicableOperators": [
        "EQ",
        "IN"
      ],
      "dataType": "STRING",
      "displayName": "User",
      "displayOrder": 1,
      "fieldName": "userName",
      "isHidden": false,
      "isVirtual": false
    }
  ],
  "columnSortings": [
    {
      "fieldName": "userName",
      "isAscending": true,
      "sortingOrder": 1
    }
  ],
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial report-updated",
  "displayName": "monthly-audit",
  "freeformTags": {
    "env": "dev"
  },
  "summary": [
    {
      "countOf": "id",
      "displayOrder": 1,
      "groupByFieldName": "userName",
      "isHidden": false,
      "name": "Users"
    }
  ]
}`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.ReportDefinition](t, `{
  "columnFilters": [
    {
      "expressions": [
        "app"
      ],
      "fieldName": "userName",
      "isEnabled": true,
      "isHidden": false,
      "operator": "EQ"
    }
  ],
  "columnInfo": [
    {
      "applicableOperators": [
        "EQ",
        "IN"
      ],
      "dataType": "STRING",
      "displayName": "User",
      "displayOrder": 1,
      "fieldName": "userName",
      "isHidden": false,
      "isVirtual": false
    }
  ],
  "columnSortings": [
    {
      "fieldName": "userName",
      "isAscending": true,
      "sortingOrder": 1
    }
  ],
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial report-updated",
  "displayName": "monthly-audit",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "parentId": "\u003cocid:2\u003e",
  "summary": [
    {
      "countOf": "id",
      "displayOrder": 1,
      "groupByFieldName": "userName",
      "isHidden": false,
      "name": "Users"
    }
  ]
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		datasafesdk.ReportDefinition,
		datasafesdk.CreateReportDefinitionDetails,
		datasafesdk.UpdateReportDefinitionDetails,
	]{
		CollectionPath:    "/20181201/reportDefinitions",
		ItemPath:          "/20181201/reportDefinitions/<ocid:3>",
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
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateReportDefinitionDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ datasafesdk.ReportDefinition) error {
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
			t.Errorf("close ReportDefinition OCI mock: %v", err)
		}
	})
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	client := newReportDefinitionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.ReportDefinition]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.ReportDefinition) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.ColumnFilters, current.Spec.ColumnFilters) ||
				!reflect.DeepEqual(current.Status.ColumnInfo, current.Spec.ColumnInfo) ||
				!reflect.DeepEqual(current.Status.ColumnSortings, current.Spec.ColumnSortings) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.ParentId, current.Spec.ParentId) ||
				!reflect.DeepEqual(current.Status.Summary, current.Spec.Summary) {
				return fmt.Errorf("created ReportDefinition status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.ReportDefinition) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datasafev1beta1.ReportDefinition) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.ColumnFilters, current.Spec.ColumnFilters) ||
				!reflect.DeepEqual(current.Status.ColumnInfo, current.Spec.ColumnInfo) ||
				!reflect.DeepEqual(current.Status.ColumnSortings, current.Spec.ColumnSortings) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Summary, current.Spec.Summary) {
				return fmt.Errorf("updated ReportDefinition status = %+v", current.Status)
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
