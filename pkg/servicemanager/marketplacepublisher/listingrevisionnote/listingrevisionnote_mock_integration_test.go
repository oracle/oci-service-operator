/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package listingrevisionnote

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
func TestMockIntegrationListingRevisionNoteLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newListingRevisionNoteResource()
	ocimock.InitializeResource(resource, "mock-listingrevisionnote")
	resource.Spec = ocimock.MustJSONFixture[marketplacepublisherv1beta1.ListingRevisionNoteSpec](t, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "test"
  },
  "listingRevisionId": "\u003cocid:1\u003e",
  "noteDetails": "review note"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "freeformTags": {
    "env": "test",
    "osok-mock-update": "true"
  }
}`)
	createRequest := ocimock.MustJSONFixture[marketplacepublishersdk.CreateListingRevisionNoteDetails](t, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "test"
  },
  "listingRevisionId": "\u003cocid:1\u003e",
  "noteDetails": "review note"
}`)
	createdState := ocimock.MustOCIResponseFixture[marketplacepublishersdk.ListingRevisionNote](t, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "listingRevisionId": "\u003cocid:1\u003e",
  "noteDetails": "review note"
}`)
	updateRequest := ocimock.MustJSONFixture[marketplacepublishersdk.UpdateListingRevisionNoteDetails](t, `{
  "freeformTags": {
    "env": "test",
    "osok-mock-update": "true"
  }
}`)
	updatedState := ocimock.MustOCIResponseFixture[marketplacepublishersdk.ListingRevisionNote](t, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "freeformTags": {
    "env": "test",
    "osok-mock-update": "true"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "listingRevisionId": "\u003cocid:1\u003e",
  "noteDetails": "review note"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		marketplacepublishersdk.ListingRevisionNote,
		marketplacepublishersdk.CreateListingRevisionNoteDetails,
		marketplacepublishersdk.UpdateListingRevisionNoteDetails,
	]{
		CollectionPath:    "/20241201/listingRevisionNotes",
		ItemPath:          "/20241201/listingRevisionNotes/<ocid:2>",
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
		ValidateCreate: func(request ocimock.Request, _ marketplacepublishersdk.CreateListingRevisionNoteDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ marketplacepublishersdk.ListingRevisionNote) error {
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
			t.Errorf("close ListingRevisionNote OCI mock: %v", err)
		}
	})
	sdkClient := marketplacepublishersdk.MarketplacePublisherClient{BaseClient: session.BaseClient()}
	manager := &ListingRevisionNoteServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newListingRevisionNoteDefaultRuntimeHooks(sdkClient)
	applyListingRevisionNoteRuntimeHooks(&hooks)
	client := wrapListingRevisionNoteGeneratedClient(hooks, defaultListingRevisionNoteServiceClient{ServiceClient: generatedruntime.NewServiceClient[*marketplacepublisherv1beta1.ListingRevisionNote](buildListingRevisionNoteGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*marketplacepublisherv1beta1.ListingRevisionNote]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *marketplacepublisherv1beta1.ListingRevisionNote) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.ListingRevisionId, current.Spec.ListingRevisionId) ||
				!reflect.DeepEqual(current.Status.NoteDetails, current.Spec.NoteDetails) {
				return fmt.Errorf("created ListingRevisionNote status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *marketplacepublisherv1beta1.ListingRevisionNote) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *marketplacepublisherv1beta1.ListingRevisionNote) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated ListingRevisionNote status = %+v", current.Status)
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
