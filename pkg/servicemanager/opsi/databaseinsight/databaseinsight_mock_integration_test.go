/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package databaseinsight

import (
	"context"
	"fmt"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationDatabaseInsightLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeDatabaseInsightResource()
	ocimock.InitializeResource(resource, "mock-databaseinsight")
	resource.Spec = ocimock.MustJSONFixture[opsiv1beta1.DatabaseInsightSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "databaseId": "\u003cocid:2\u003e",
  "databaseResourceType": "AUTONOMOUS_DATABASE",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "entitySource": "AUTONOMOUS_DATABASE",
  "freeformTags": {
    "env": "test"
  },
  "isAdvancedFeaturesEnabled": false
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "entitySource": "AUTONOMOUS_DATABASE",
  "freeformTags": {
    "env": "test",
    "osok-mock-update": "true"
  }
}`)
	createRequest := ocimock.MustJSONFixture[opsisdk.CreateAutonomousDatabaseInsightDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "databaseId": "\u003cocid:2\u003e",
  "databaseResourceType": "AUTONOMOUS_DATABASE",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "entitySource": "AUTONOMOUS_DATABASE",
  "freeformTags": {
    "env": "test"
  },
  "isAdvancedFeaturesEnabled": false
}`)
	createdState := ocimock.MustOCIResponseFixture[opsisdk.AutonomousDatabaseInsight](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "databaseId": "\u003cocid:2\u003e",
  "databaseResourceType": "AUTONOMOUS_DATABASE",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "entitySource": "AUTONOMOUS_DATABASE",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:3\u003e",
  "isAdvancedFeaturesEnabled": false,
  "lifecycleState": "ACTIVE"
}`)
	createdReadStates := []opsisdk.AutonomousDatabaseInsight{
		ocimock.MustOCIResponseFixture[opsisdk.AutonomousDatabaseInsight](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "databaseId": "\u003cocid:2\u003e",
  "databaseResourceType": "AUTONOMOUS_DATABASE",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "entitySource": "AUTONOMOUS_DATABASE",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:3\u003e",
  "isAdvancedFeaturesEnabled": false,
  "lifecycleState": "ACTIVE"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[opsisdk.UpdateAutonomousDatabaseInsightDetails](t, `{
  "freeformTags": {
    "env": "test",
    "osok-mock-update": "true"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[opsisdk.AutonomousDatabaseInsight](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "databaseId": "\u003cocid:2\u003e",
  "databaseResourceType": "AUTONOMOUS_DATABASE",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "entitySource": "AUTONOMOUS_DATABASE",
  "freeformTags": {
    "env": "test",
    "osok-mock-update": "true"
  },
  "id": "\u003cocid:3\u003e",
  "isAdvancedFeaturesEnabled": false,
  "lifecycleState": "ACTIVE"
}`)
	updatedReadStates := []opsisdk.AutonomousDatabaseInsight{
		ocimock.MustOCIResponseFixture[opsisdk.AutonomousDatabaseInsight](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "databaseId": "\u003cocid:2\u003e",
  "databaseResourceType": "AUTONOMOUS_DATABASE",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "entitySource": "AUTONOMOUS_DATABASE",
  "freeformTags": {
    "env": "test",
    "osok-mock-update": "true"
  },
  "id": "\u003cocid:3\u003e",
  "isAdvancedFeaturesEnabled": false,
  "lifecycleState": "ACTIVE"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		opsisdk.AutonomousDatabaseInsight,
		opsisdk.CreateAutonomousDatabaseInsightDetails,
		opsisdk.UpdateAutonomousDatabaseInsightDetails,
	]{
		CollectionPath:     "/20200630/databaseInsights",
		ItemPath:           "/20200630/databaseInsights/<ocid:3>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		UpdatedState:       &updatedState,
		CreatedReadStates:  createdReadStates,
		UpdatedReadStates:  updatedReadStates,
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ opsisdk.CreateAutonomousDatabaseInsightDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "entitySource", "AUTONOMOUS_DATABASE", updateRequest)
		},
		ValidateDelete: func(request ocimock.Request, _ opsisdk.AutonomousDatabaseInsight) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20200630", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close DatabaseInsight OCI mock: %v", err)
		}
	})
	sdkClient := opsisdk.OperationsInsightsClient{BaseClient: session.BaseClient()}
	client := newDatabaseInsightServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*opsiv1beta1.DatabaseInsight]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *opsiv1beta1.DatabaseInsight) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DatabaseId, current.Spec.DatabaseId) ||
				!reflect.DeepEqual(current.Status.DatabaseResourceType, current.Spec.DatabaseResourceType) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.EntitySource, current.Spec.EntitySource) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsAdvancedFeaturesEnabled, current.Spec.IsAdvancedFeaturesEnabled) {
				return fmt.Errorf("created DatabaseInsight status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *opsiv1beta1.DatabaseInsight) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *opsiv1beta1.DatabaseInsight) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.EntitySource, current.Spec.EntitySource) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated DatabaseInsight status = %+v", current.Status)
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
