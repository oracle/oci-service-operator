/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package listing

import (
	"context"
	"fmt"
	marketplacepublishersdk "github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationListingLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &marketplacepublisherv1beta1.Listing{Spec: marketplacepublisherv1beta1.ListingSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", Name: "osok-mock-marketplace-listing",
		ListingType: string(marketplacepublishersdk.ListingTypeOciApplication), PackageType: string(marketplacepublishersdk.PackageTypeStack),
	}}
	ocimock.InitializeResource(resource, "mock-listing")
	resource.Spec = ocimock.MustJSONFixture[marketplacepublisherv1beta1.ListingSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "listingType": "OCI_APPLICATION",
  "name": "osok-mock-marketplace-listing",
  "packageType": "STACK"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "freeformTags": {
    "osok-mock-update": "true"
  }
}`)
	createRequest := ocimock.MustJSONFixture[marketplacepublishersdk.CreateListingDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "listingType": "OCI_APPLICATION",
  "name": "osok-mock-marketplace-listing",
  "packageType": "STACK"
}`)
	createdState := ocimock.MustOCIResponseFixture[marketplacepublishersdk.Listing](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "listingType": "OCI_APPLICATION",
  "name": "osok-mock-marketplace-listing",
  "packageType": "STACK"
}`)
	updateRequest := ocimock.MustJSONFixture[marketplacepublishersdk.UpdateListingDetails](t, `{
  "freeformTags": {
    "osok-mock-update": "true"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[marketplacepublishersdk.Listing](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "freeformTags": {
    "osok-mock-update": "true"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "listingType": "OCI_APPLICATION",
  "name": "osok-mock-marketplace-listing",
  "packageType": "STACK"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		marketplacepublishersdk.Listing,
		marketplacepublishersdk.CreateListingDetails,
		marketplacepublishersdk.UpdateListingDetails,
	]{
		CollectionPath:    "/20241201/listings",
		ItemPath:          "/20241201/listings/<ocid:2>",
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
		ValidateCreate: func(request ocimock.Request, _ marketplacepublishersdk.CreateListingDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ marketplacepublishersdk.Listing) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20241201", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Listing OCI mock: %v", err)
		}
	})
	sdkClient := marketplacepublishersdk.MarketplacePublisherClient{BaseClient: session.BaseClient()}
	manager := &ListingServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newListingRuntimeHooks(manager, sdkClient)
	client := wrapListingGeneratedClient(hooks, defaultListingServiceClient{ServiceClient: generatedruntime.NewServiceClient[*marketplacepublisherv1beta1.Listing](buildListingGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*marketplacepublisherv1beta1.Listing]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *marketplacepublisherv1beta1.Listing) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.ListingType, current.Spec.ListingType) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) ||
				!reflect.DeepEqual(current.Status.PackageType, current.Spec.PackageType) {
				return fmt.Errorf("created Listing status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *marketplacepublisherv1beta1.Listing) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *marketplacepublisherv1beta1.Listing) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated Listing status = %+v", current.Status)
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
