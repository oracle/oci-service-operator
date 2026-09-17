/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package listingrevisionpackage

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
func TestMockIntegrationListingRevisionPackageLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := testListingRevisionPackageResource()
	ocimock.InitializeResource(resource, "mock-listingrevisionpackage")
	resource.Spec = ocimock.MustJSONFixture[marketplacepublisherv1beta1.ListingRevisionPackageSpec](t, `{
  "areSecurityUpgradesProvided": false,
  "artifactId": "artifact-1",
  "definedTags": {
    "ns": {
      "team": "osok"
    }
  },
  "description": "description",
  "displayName": "package",
  "freeformTags": {
    "env": "dev"
  },
  "isDefault": true,
  "listingRevisionId": "revision-1",
  "packageVersion": "1.0.0",
  "termId": "term-1"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "description-updated"
}`)
	createRequest := ocimock.MustJSONFixture[marketplacepublishersdk.CreateListingRevisionPackageDetails](t, `{
  "areSecurityUpgradesProvided": false,
  "artifactId": "artifact-1",
  "definedTags": {
    "ns": {
      "team": "osok"
    }
  },
  "description": "description",
  "displayName": "package",
  "freeformTags": {
    "env": "dev"
  },
  "isDefault": true,
  "listingRevisionId": "revision-1",
  "packageVersion": "1.0.0",
  "termId": "term-1"
}`)
	createdState := ocimock.MustOCIResponseFixture[marketplacepublishersdk.StackPackage](t, `{
  "areSecurityUpgradesProvided": false,
  "artifactId": "artifact-1",
  "definedTags": {
    "ns": {
      "team": "osok"
    }
  },
  "description": "description",
  "displayName": "package",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:1\u003e",
  "isDefault": true,
  "lifecycleState": "ACTIVE",
  "listingRevisionId": "revision-1",
  "packageVersion": "1.0.0",
  "termId": "term-1"
}`)
	createdReadStates := []marketplacepublishersdk.StackPackage{
		ocimock.MustOCIResponseFixture[marketplacepublishersdk.StackPackage](t, `{
  "areSecurityUpgradesProvided": false,
  "artifactId": "artifact-1",
  "definedTags": {
    "ns": {
      "team": "osok"
    }
  },
  "description": "description",
  "displayName": "package",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:1\u003e",
  "isDefault": true,
  "lifecycleState": "ACTIVE",
  "listingRevisionId": "revision-1",
  "packageVersion": "1.0.0",
  "termId": "term-1"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[marketplacepublishersdk.UpdateListingRevisionPackageDetails](t, `{
  "description": "description-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[marketplacepublishersdk.StackPackage](t, `{
  "areSecurityUpgradesProvided": false,
  "artifactId": "artifact-1",
  "definedTags": {
    "ns": {
      "team": "osok"
    }
  },
  "description": "description-updated",
  "displayName": "package",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:1\u003e",
  "isDefault": true,
  "lifecycleState": "ACTIVE",
  "listingRevisionId": "revision-1",
  "packageVersion": "1.0.0",
  "termId": "term-1"
}`)
	updatedReadStates := []marketplacepublishersdk.StackPackage{
		ocimock.MustOCIResponseFixture[marketplacepublishersdk.StackPackage](t, `{
  "areSecurityUpgradesProvided": false,
  "artifactId": "artifact-1",
  "definedTags": {
    "ns": {
      "team": "osok"
    }
  },
  "description": "description-updated",
  "displayName": "package",
  "freeformTags": {
    "env": "dev"
  },
  "id": "\u003cocid:1\u003e",
  "isDefault": true,
  "lifecycleState": "ACTIVE",
  "listingRevisionId": "revision-1",
  "packageVersion": "1.0.0",
  "termId": "term-1"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		marketplacepublishersdk.StackPackage,
		marketplacepublishersdk.CreateListingRevisionPackageDetails,
		marketplacepublishersdk.UpdateListingRevisionPackageDetails,
	]{
		CollectionPath:     "/20241201/listingRevisionPackages",
		ItemPath:           "/20241201/listingRevisionPackages/<ocid:1>",
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
		ValidateCreate: func(request ocimock.Request, _ marketplacepublishersdk.CreateListingRevisionPackageDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ marketplacepublishersdk.StackPackage) error {
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
			t.Errorf("close ListingRevisionPackage OCI mock: %v", err)
		}
	})
	sdkClient := marketplacepublishersdk.MarketplacePublisherClient{BaseClient: session.BaseClient()}
	client := newListingRevisionPackageServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*marketplacepublisherv1beta1.ListingRevisionPackage]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *marketplacepublisherv1beta1.ListingRevisionPackage) error {
			if current.Status.Id != "<ocid:1>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:1>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.AreSecurityUpgradesProvided, current.Spec.AreSecurityUpgradesProvided) ||
				!reflect.DeepEqual(current.Status.ArtifactId, current.Spec.ArtifactId) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.IsDefault, current.Spec.IsDefault) ||
				!reflect.DeepEqual(current.Status.ListingRevisionId, current.Spec.ListingRevisionId) ||
				!reflect.DeepEqual(current.Status.PackageVersion, current.Spec.PackageVersion) ||
				!reflect.DeepEqual(current.Status.TermId, current.Spec.TermId) {
				return fmt.Errorf("created ListingRevisionPackage status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *marketplacepublisherv1beta1.ListingRevisionPackage) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *marketplacepublisherv1beta1.ListingRevisionPackage) error {
			if current.Status.Id != "<ocid:1>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:1>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated ListingRevisionPackage status = %+v", current.Status)
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
