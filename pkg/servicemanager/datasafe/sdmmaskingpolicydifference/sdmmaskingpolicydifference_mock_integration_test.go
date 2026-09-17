/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package sdmmaskingpolicydifference

import (
	"context"
	"fmt"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationSdmMaskingPolicyDifferenceLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newSdmMaskingPolicyDifferenceTestResource()
	ocimock.InitializeResource(resource, "mock-sdmmaskingpolicydifference")
	resource.Spec = ocimock.MustJSONFixture[datasafev1beta1.SdmMaskingPolicyDifferenceSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "differenceType": "ALL",
  "displayName": "sdm-masking-policy-difference",
  "freeformTags": {
    "team": "security"
  },
  "maskingPolicyId": "\u003cocid:2\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "displayName": "sdm-masking-policy-difference-updated"
}`)
	createRequest := ocimock.MustJSONFixture[datasafesdk.CreateSdmMaskingPolicyDifferenceDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "differenceType": "ALL",
  "displayName": "sdm-masking-policy-difference",
  "freeformTags": {
    "team": "security"
  },
  "maskingPolicyId": "\u003cocid:2\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[datasafesdk.SdmMaskingPolicyDifference](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "differenceType": "ALL",
  "displayName": "sdm-masking-policy-difference",
  "freeformTags": {
    "team": "security"
  },
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "maskingPolicyId": "\u003cocid:2\u003e"
}`)
	updateRequest := ocimock.MustJSONFixture[datasafesdk.UpdateSdmMaskingPolicyDifferenceDetails](t, `{
  "displayName": "sdm-masking-policy-difference-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[datasafesdk.SdmMaskingPolicyDifference](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "differenceType": "ALL",
  "displayName": "sdm-masking-policy-difference-updated",
  "freeformTags": {
    "team": "security"
  },
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "maskingPolicyId": "\u003cocid:2\u003e"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		datasafesdk.SdmMaskingPolicyDifference,
		datasafesdk.CreateSdmMaskingPolicyDifferenceDetails,
		datasafesdk.UpdateSdmMaskingPolicyDifferenceDetails,
	]{
		CollectionPath:    "/20181201/sdmMaskingPolicyDifferences",
		ItemPath:          "/20181201/sdmMaskingPolicyDifferences/<ocid:3>",
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
		ValidateCreate: func(request ocimock.Request, _ datasafesdk.CreateSdmMaskingPolicyDifferenceDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ datasafesdk.SdmMaskingPolicyDifference) error {
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
			t.Errorf("close SdmMaskingPolicyDifference OCI mock: %v", err)
		}
	})
	sdkClient := datasafesdk.DataSafeClient{BaseClient: session.BaseClient()}
	hooks := newSdmMaskingPolicyDifferenceDefaultRuntimeHooks(sdkClient)
	applySdmMaskingPolicyDifferenceRuntimeHooks(&hooks)
	client := newSdmMaskingPolicyDifferenceRuntimeTestClient(hooks)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*datasafev1beta1.SdmMaskingPolicyDifference]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *datasafev1beta1.SdmMaskingPolicyDifference) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DifferenceType, current.Spec.DifferenceType) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.MaskingPolicyId, current.Spec.MaskingPolicyId) {
				return fmt.Errorf("created SdmMaskingPolicyDifference status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *datasafev1beta1.SdmMaskingPolicyDifference) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *datasafev1beta1.SdmMaskingPolicyDifference) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("updated SdmMaskingPolicyDifference status = %+v", current.Status)
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
