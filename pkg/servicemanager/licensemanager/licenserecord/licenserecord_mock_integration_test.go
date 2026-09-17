/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package licenserecord

import (
	"context"
	"fmt"
	licensemanagersdk "github.com/oracle/oci-go-sdk/v65/licensemanager"
	licensemanagerv1beta1 "github.com/oracle/oci-service-operator/api/licensemanager/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationLicenseRecordLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeLicenseRecordResource()
	ocimock.InitializeResource(resource, "mock-licenserecord")
	resource.Spec = ocimock.MustJSONFixture[licensemanagerv1beta1.LicenseRecordSpec](t, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "license-record-sample",
  "expirationDate": "2027-01-02T00:00:00Z",
  "freeformTags": {
    "env": "test"
  },
  "isPerpetual": false,
  "isUnlimited": true,
  "licenseCount": 3,
  "productId": "license-product",
  "supportEndDate": "2027-06-03T04:05:06Z"
}`)
	resource.Annotations[LicenseRecordProductLicenseIDAnnotation] = "<ocid:1>"
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "license-record-sample-updated",
  "expirationDate": "2027-01-02T00:00:00Z",
  "freeformTags": {
    "env": "test"
  },
  "isPerpetual": false,
  "isUnlimited": true,
  "licenseCount": 3,
  "productId": "license-product",
  "supportEndDate": "2027-06-03T04:05:06Z"
}`)
	createRequest := ocimock.MustJSONFixture[licensemanagersdk.CreateLicenseRecordDetails](t, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "license-record-sample",
  "expirationDate": "2027-01-02T00:00:00Z",
  "freeformTags": {
    "env": "test"
  },
  "isPerpetual": false,
  "isUnlimited": true,
  "licenseCount": 3,
  "productId": "license-product",
  "supportEndDate": "2027-06-03T04:05:06Z"
}`)
	createdState := ocimock.MustOCIResponseFixture[licensemanagersdk.LicenseRecord](t, `{
  "compartmentId": "\u003cocid:2\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "license-record-sample",
  "expirationDate": "2027-01-02T00:00:00Z",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:3\u003e",
  "isPerpetual": false,
  "isUnlimited": true,
  "licenseCount": 3,
  "lifecycleState": "ACTIVE",
  "productId": "license-product",
  "productLicense": null,
  "productLicenseId": "\u003cocid:1\u003e",
  "supportEndDate": "2027-06-03T04:05:06Z",
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null
}`)
	updateRequest := ocimock.MustJSONFixture[licensemanagersdk.UpdateLicenseRecordDetails](t, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "license-record-sample-updated",
  "expirationDate": "2027-01-02T00:00:00Z",
  "freeformTags": {
    "env": "test"
  },
  "isPerpetual": false,
  "isUnlimited": true,
  "licenseCount": 3,
  "productId": "license-product",
  "supportEndDate": "2027-06-03T04:05:06Z"
}`)
	updatedState := ocimock.MustOCIResponseFixture[licensemanagersdk.LicenseRecord](t, `{
  "compartmentId": "\u003cocid:2\u003e",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "displayName": "license-record-sample-updated",
  "expirationDate": "2027-01-02T00:00:00Z",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:3\u003e",
  "isPerpetual": false,
  "isUnlimited": true,
  "licenseCount": 3,
  "lifecycleState": "ACTIVE",
  "productId": "license-product",
  "productLicense": null,
  "productLicenseId": "\u003cocid:1\u003e",
  "supportEndDate": "2027-06-03T04:05:06Z",
  "systemTags": null,
  "timeCreated": null,
  "timeUpdated": null
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		licensemanagersdk.LicenseRecord,
		licensemanagersdk.CreateLicenseRecordDetails,
		licensemanagersdk.UpdateLicenseRecordDetails,
	]{
		CollectionPath:    "/20220430/licenseRecords",
		ItemPath:          "/20220430/licenseRecords/<ocid:3>",
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
		ValidateCreate: func(request ocimock.Request, _ licensemanagersdk.CreateLicenseRecordDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ licensemanagersdk.LicenseRecord) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20220430", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close LicenseRecord OCI mock: %v", err)
		}
	})
	sdkClient := licensemanagersdk.LicenseManagerClient{BaseClient: session.BaseClient()}
	client := newLicenseRecordServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*licensemanagerv1beta1.LicenseRecord]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *licensemanagerv1beta1.LicenseRecord) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.ExpirationDate, current.Spec.ExpirationDate) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsPerpetual, current.Spec.IsPerpetual) ||
				!reflect.DeepEqual(current.Status.IsUnlimited, current.Spec.IsUnlimited) ||
				!reflect.DeepEqual(current.Status.LicenseCount, current.Spec.LicenseCount) ||
				!reflect.DeepEqual(current.Status.ProductId, current.Spec.ProductId) ||
				!reflect.DeepEqual(current.Status.SupportEndDate, current.Spec.SupportEndDate) {
				return fmt.Errorf("created LicenseRecord status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *licensemanagerv1beta1.LicenseRecord) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *licensemanagerv1beta1.LicenseRecord) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.ExpirationDate, current.Spec.ExpirationDate) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsPerpetual, current.Spec.IsPerpetual) ||
				!reflect.DeepEqual(current.Status.IsUnlimited, current.Spec.IsUnlimited) ||
				!reflect.DeepEqual(current.Status.LicenseCount, current.Spec.LicenseCount) ||
				!reflect.DeepEqual(current.Status.ProductId, current.Spec.ProductId) ||
				!reflect.DeepEqual(current.Status.SupportEndDate, current.Spec.SupportEndDate) {
				return fmt.Errorf("updated LicenseRecord status = %+v", current.Status)
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
