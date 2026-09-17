/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package listingrevision

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
func TestMockIntegrationListingRevisionLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionResource()
	ocimock.InitializeResource(resource, "mock-listingrevision")
	resource.Spec = ocimock.MustJSONFixture[marketplacepublisherv1beta1.ListingRevisionSpec](t, `{
  "displayName": "Partner service",
  "freeformTags": {
    "env": "dev"
  },
  "headline": "Partner service headline",
  "industries": [
    "Technology"
  ],
  "listingId": "\u003cocid:1\u003e",
  "listingType": "SERVICE",
  "productCodes": [
    "COMPUTE"
  ],
  "tagline": "Initial tagline"
}`)
	updatedSpec := resource.Spec
	updatedSpec.Tagline = "Updated tagline"
	createRequest := ocimock.MustJSONFixture[marketplacepublishersdk.CreateServiceListingRevisionDetails](t, `{
  "displayName": "Partner service",
  "freeformTags": {
    "env": "dev"
  },
  "headline": "Partner service headline",
  "industries": [
    "Technology"
  ],
  "listingId": "\u003cocid:1\u003e",
  "productCodes": [
    "COMPUTE"
  ],
  "tagline": "Initial tagline"
}`)
	createdState := ocimock.MustOCIResponseFixture[marketplacepublishersdk.ServiceListingRevision](t, `{
  "displayName": "Partner service",
  "freeformTags": {
    "env": "dev"
  },
  "headline": "Partner service headline",
  "id": "\u003cocid:2\u003e",
  "industries": [
    "Technology"
  ],
  "lifecycleState": "ACTIVE",
  "listingId": "\u003cocid:1\u003e",
  "listingType": "SERVICE",
  "productCodes": [
    "COMPUTE"
  ],
  "tagline": "Initial tagline"
}`)
	createdReadStates := []marketplacepublishersdk.ServiceListingRevision{
		ocimock.MustOCIResponseFixture[marketplacepublishersdk.ServiceListingRevision](t, `{
  "displayName": "Partner service",
  "freeformTags": {
    "env": "dev"
  },
  "headline": "Partner service headline",
  "id": "\u003cocid:2\u003e",
  "industries": [
    "Technology"
  ],
  "lifecycleState": "ACTIVE",
  "listingId": "\u003cocid:1\u003e",
  "listingType": "SERVICE",
  "productCodes": [
    "COMPUTE"
  ],
  "tagline": "Initial tagline"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[marketplacepublishersdk.UpdateServiceListingRevisionDetails](t, `{
  "displayName": "Partner service",
  "freeformTags": {
    "env": "dev"
  },
  "headline": "Partner service headline",
  "industries": [
    "Technology"
  ],
  "productCodes": [
    "COMPUTE"
  ],
  "tagline": "Updated tagline"
}`)
	updatedState := createdState
	updatedState.Tagline = updateRequest.Tagline
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		marketplacepublishersdk.ServiceListingRevision,
		marketplacepublishersdk.CreateServiceListingRevisionDetails,
		marketplacepublishersdk.UpdateServiceListingRevisionDetails,
	]{
		CollectionPath:     "/20241201/listingRevisions",
		ItemPath:           "/20241201/listingRevisions/<ocid:2>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState:       &createdState,
		CreatedReadStates:  createdReadStates,
		UpdatedState:       &updatedState,
		UpdatedReadStates:  []marketplacepublishersdk.ServiceListingRevision{updatedState},
		DeleteEndsNotFound: true,
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateDiscriminatedJSONRequest(request, "listingType", "SERVICE", createRequest); err != nil {
				return err
			}
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "listingType", "SERVICE", updateRequest)
		},
		ValidateDelete: func(request ocimock.Request, _ marketplacepublishersdk.ServiceListingRevision) error {
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
			t.Errorf("close ListingRevision OCI mock: %v", err)
		}
	})
	sdkClient := marketplacepublishersdk.MarketplacePublisherClient{BaseClient: session.BaseClient()}
	client := newListingRevisionServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*marketplacepublisherv1beta1.ListingRevision]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *marketplacepublisherv1beta1.ListingRevision) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.Headline, current.Spec.Headline) ||
				!reflect.DeepEqual(current.Status.Industries, current.Spec.Industries) ||
				!reflect.DeepEqual(current.Status.ListingId, current.Spec.ListingId) ||
				!reflect.DeepEqual(current.Status.ListingType, current.Spec.ListingType) ||
				!reflect.DeepEqual(current.Status.ProductCodes, current.Spec.ProductCodes) ||
				!reflect.DeepEqual(current.Status.Tagline, current.Spec.Tagline) {
				return fmt.Errorf("created ListingRevision status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *marketplacepublisherv1beta1.ListingRevision) {
			current.Spec = updatedSpec
		},
		ValidateUpdated: func(current *marketplacepublisherv1beta1.ListingRevision) error {
			if current.Status.Id != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.ListingType, current.Spec.ListingType) ||
				!reflect.DeepEqual(current.Status.Tagline, current.Spec.Tagline) {
				return fmt.Errorf("updated ListingRevision status = %+v", current.Status)
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
