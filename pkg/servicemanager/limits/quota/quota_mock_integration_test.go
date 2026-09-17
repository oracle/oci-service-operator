/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package quota

import (
	"context"
	"fmt"
	limitssdk "github.com/oracle/oci-go-sdk/v65/limits"
	limitsv1beta1 "github.com/oracle/oci-service-operator/api/limits/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationQuotaLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newQuotaRuntimeTestResource()
	ocimock.InitializeResource(resource, "mock-quota")
	resource.Spec = ocimock.MustJSONFixture[limitsv1beta1.QuotaSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "runtime quota",
  "freeformTags": {
    "env": "dev"
  },
  "name": "quota-runtime",
  "statements": [
    "zero compute-core quotas in compartment runtime"
  ]
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "runtime quota-updated"
}`)
	createRequest := ocimock.MustJSONFixture[limitssdk.CreateQuotaDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "runtime quota",
  "freeformTags": {
    "env": "dev"
  },
  "name": "quota-runtime",
  "statements": [
    "zero compute-core quotas in compartment runtime"
  ]
}`)
	createdState := ocimock.MustOCIResponseFixture[limitssdk.Quota](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "runtime quota",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "name": "quota-runtime",
  "statements": [
    "zero compute-core quotas in compartment runtime"
  ]
}`)
	updateRequest := ocimock.MustJSONFixture[limitssdk.UpdateQuotaDetails](t, `{
  "description": "runtime quota-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[limitssdk.Quota](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "runtime quota-updated",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "name": "quota-runtime",
  "statements": [
    "zero compute-core quotas in compartment runtime"
  ]
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		limitssdk.Quota,
		limitssdk.CreateQuotaDetails,
		limitssdk.UpdateQuotaDetails,
	]{
		CollectionPath:    "/20181025/quotas",
		ItemPath:          "/20181025/quotas/<ocid:2>",
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
		ValidateCreate: func(request ocimock.Request, _ limitssdk.CreateQuotaDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ limitssdk.Quota) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20181025", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Quota OCI mock: %v", err)
		}
	})
	sdkClient := limitssdk.QuotasClient{BaseClient: session.BaseClient()}
	client := newQuotaRuntimeTestClient(&fakeQuotaOCIClient{createFunc: sdkClient.CreateQuota, getFunc: sdkClient.GetQuota, listFunc: sdkClient.ListQuotas, updateFunc: sdkClient.UpdateQuota, deleteFunc: sdkClient.DeleteQuota})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*limitsv1beta1.Quota]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *limitsv1beta1.Quota) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.Statements, current.Spec.Statements) {
				return fmt.Errorf("created Quota status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *limitsv1beta1.Quota) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *limitsv1beta1.Quota) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated Quota status = %+v", current.Status)
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
