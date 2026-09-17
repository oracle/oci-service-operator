/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package servicecatalog

import (
	"context"
	"fmt"
	servicecatalogsdk "github.com/oracle/oci-go-sdk/v65/servicecatalog"
	servicecatalogv1beta1 "github.com/oracle/oci-service-operator/api/servicecatalog/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationServiceCatalogLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeServiceCatalogResource()
	ocimock.InitializeResource(resource, "mock-servicecatalog")
	resource.Spec = ocimock.MustJSONFixture[servicecatalogv1beta1.ServiceCatalogSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "catalog-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "status": "ACTIVE"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "catalog-alpha-updated",
  "status": "ACTIVE"
}`)
	createRequest := ocimock.MustJSONFixture[servicecatalogsdk.CreateServiceCatalogDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "catalog-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "status": "ACTIVE"
}`)
	createdState := ocimock.MustOCIResponseFixture[servicecatalogsdk.ServiceCatalog](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "catalog-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "status": "ACTIVE"
}`)
	createdReadStates := []servicecatalogsdk.ServiceCatalog{
		ocimock.MustOCIResponseFixture[servicecatalogsdk.ServiceCatalog](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "catalog-alpha",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "status": "ACTIVE"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[servicecatalogsdk.UpdateServiceCatalogDetails](t, `{
  "displayName": "catalog-alpha-updated",
  "status": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[servicecatalogsdk.ServiceCatalog](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "catalog-alpha-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "status": "ACTIVE"
}`)
	updatedReadStates := []servicecatalogsdk.ServiceCatalog{
		ocimock.MustOCIResponseFixture[servicecatalogsdk.ServiceCatalog](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "catalog-alpha-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "status": "ACTIVE"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		servicecatalogsdk.ServiceCatalog,
		servicecatalogsdk.CreateServiceCatalogDetails,
		servicecatalogsdk.UpdateServiceCatalogDetails,
	]{
		CollectionPath:     "/20210527/serviceCatalogs",
		ItemPath:           "/20210527/serviceCatalogs/<ocid:2>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
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
		ValidateCreate: func(request ocimock.Request, _ servicecatalogsdk.CreateServiceCatalogDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ servicecatalogsdk.ServiceCatalog) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20210527", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close ServiceCatalog OCI mock: %v", err)
		}
	})
	sdkClient := servicecatalogsdk.ServiceCatalogClient{BaseClient: session.BaseClient()}
	client := newServiceCatalogServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*servicecatalogv1beta1.ServiceCatalog]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *servicecatalogv1beta1.ServiceCatalog) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Status, current.Spec.Status) {
				return fmt.Errorf("created ServiceCatalog status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *servicecatalogv1beta1.ServiceCatalog) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *servicecatalogv1beta1.ServiceCatalog) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.Status, current.Spec.Status) {
				return fmt.Errorf("updated ServiceCatalog status = %+v", current.Status)
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
