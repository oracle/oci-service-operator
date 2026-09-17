/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package productlicense

import (
	"context"
	"fmt"
	licensemanagersdk "github.com/oracle/oci-go-sdk/v65/licensemanager"
	licensemanagerv1beta1 "github.com/oracle/oci-service-operator/api/licensemanager/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationProductLicenseLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := productLicenseResource()
	ocimock.InitializeResource(resource, "mock-productlicense")
	resource.Spec = ocimock.MustJSONFixture[licensemanagerv1beta1.ProductLicenseSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "ops": {
      "owner": "team"
    }
  },
  "displayName": "product-license",
  "freeformTags": {
    "env": "dev"
  },
  "images": [
    {
      "listingId": "listing-1",
      "packageVersion": "1.0"
    }
  ],
  "isVendorOracle": true,
  "licenseUnit": "OCPU",
  "vendorName": "Oracle"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "freeformTags": {
    "env": "dev",
    "osok-mock-update": "true"
  },
  "images": [
    {
      "listingId": "listing-1",
      "packageVersion": "1.0"
    }
  ]
}`)
	createRequest := ocimock.MustJSONFixture[licensemanagersdk.CreateProductLicenseDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "ops": {
      "owner": "team"
    }
  },
  "displayName": "product-license",
  "freeformTags": {
    "env": "dev"
  },
  "images": [
    {
      "listingId": "listing-1",
      "packageVersion": "1.0"
    }
  ],
  "isVendorOracle": true,
  "licenseUnit": "OCPU",
  "vendorName": "Oracle"
}`)
	createdState := ocimock.MustOCIResponseFixture[licensemanagersdk.ProductLicense](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "ops": {
      "owner": "team"
    }
  },
  "displayName": "product-license",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:2\u003e",
  "images": [
    {
      "listingId": "listing-1",
      "packageVersion": "1.0"
    }
  ],
  "isVendorOracle": true,
  "licenseUnit": "OCPU",
  "lifecycleState": "ACTIVE",
  "vendorName": "Oracle"
}`)
	updateRequest := ocimock.MustJSONFixture[licensemanagersdk.UpdateProductLicenseDetails](t, `{
  "freeformTags": {
    "env": "dev",
    "osok-mock-update": "true"
  },
  "images": [
    {
      "listingId": "listing-1",
      "packageVersion": "1.0"
    }
  ]
}`)
	updatedState := ocimock.MustOCIResponseFixture[licensemanagersdk.ProductLicense](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "definedTags": {
    "ops": {
      "owner": "team"
    }
  },
  "displayName": "product-license",
  "freeformTags": {
    "env": "dev",
    "osok-mock-update": "true"
  },
  "id": "\u003cocid:2\u003e",
  "images": [
    {
      "listingId": "listing-1",
      "packageVersion": "1.0"
    }
  ],
  "isVendorOracle": true,
  "licenseUnit": "OCPU",
  "lifecycleState": "ACTIVE",
  "vendorName": "Oracle"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		licensemanagersdk.ProductLicense,
		licensemanagersdk.CreateProductLicenseDetails,
		licensemanagersdk.UpdateProductLicenseDetails,
	]{
		CollectionPath:    "/20220430/productLicenses",
		ItemPath:          "/20220430/productLicenses/<ocid:2>",
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
		ValidateCreate: func(request ocimock.Request, _ licensemanagersdk.CreateProductLicenseDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ licensemanagersdk.ProductLicense) error {
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
			t.Errorf("close ProductLicense OCI mock: %v", err)
		}
	})
	sdkClient := licensemanagersdk.LicenseManagerClient{BaseClient: session.BaseClient()}
	client := newProductLicenseServiceClientWithOCIClient(sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*licensemanagerv1beta1.ProductLicense]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *licensemanagerv1beta1.ProductLicense) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Images, current.Spec.Images) ||
				!reflect.DeepEqual(current.Status.IsVendorOracle, current.Spec.IsVendorOracle) ||
				!reflect.DeepEqual(current.Status.LicenseUnit, current.Spec.LicenseUnit) ||
				!reflect.DeepEqual(current.Status.VendorName, current.Spec.VendorName) {
				return fmt.Errorf("created ProductLicense status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *licensemanagerv1beta1.ProductLicense) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *licensemanagerv1beta1.ProductLicense) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Images, current.Spec.Images) {
				return fmt.Errorf("updated ProductLicense status = %+v", current.Status)
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
