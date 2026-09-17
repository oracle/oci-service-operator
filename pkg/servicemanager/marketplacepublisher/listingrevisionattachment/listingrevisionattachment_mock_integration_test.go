/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package listingrevisionattachment

import (
	"context"
	"encoding/json"
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
func TestMockIntegrationListingRevisionAttachmentLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := makeListingRevisionAttachmentResource()
	ocimock.InitializeResource(resource, "mock-listingrevisionattachment")
	resource.Spec = ocimock.MustJSONFixture[marketplacepublisherv1beta1.ListingRevisionAttachmentSpec](t, `{
  "attachmentType": "SUPPORTED_SERVICES",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description",
  "displayName": "listing-revision-attachment",
  "freeformTags": {
    "env": "test"
  },
  "listingRevisionId": "\u003cocid:1\u003e",
  "serviceName": "implementation services",
  "type": "IMPLEMENTATION_SERVICE",
  "url": "https://example.com/service"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "attachmentType": "SUPPORTED_SERVICES",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description-updated",
  "displayName": "listing-revision-attachment",
  "freeformTags": {
    "env": "test"
  },
  "serviceName": "implementation services",
  "type": "IMPLEMENTATION_SERVICE",
  "url": "https://example.com/service"
}`)
	createRequest := ocimock.MustJSONFixture[marketplacepublishersdk.CreateSupportedServiceAttachment](t, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description",
  "displayName": "listing-revision-attachment",
  "freeformTags": {
    "env": "test"
  },
  "listingRevisionId": "\u003cocid:1\u003e",
  "serviceName": "implementation services",
  "type": "IMPLEMENTATION_SERVICE",
  "url": "https://example.com/service"
}`)
	createdState := ocimock.MustOCIResponseFixture[marketplacepublishersdk.SupportedServiceAttachment](t, `{
  "attachmentType": "SUPPORTED_SERVICES",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description",
  "displayName": "listing-revision-attachment",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "listingRevisionId": "\u003cocid:1\u003e",
  "serviceName": "implementation services",
  "type": "IMPLEMENTATION_SERVICE",
  "url": "https://example.com/service"
}`)
	createdReadStates := []marketplacepublishersdk.SupportedServiceAttachment{
		ocimock.MustOCIResponseFixture[marketplacepublishersdk.SupportedServiceAttachment](t, `{
  "attachmentType": "SUPPORTED_SERVICES",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description",
  "displayName": "listing-revision-attachment",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "listingRevisionId": "\u003cocid:1\u003e",
  "serviceName": "implementation services",
  "type": "IMPLEMENTATION_SERVICE",
  "url": "https://example.com/service"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[marketplacepublishersdk.UpdateSupportedServiceAttachment](t, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description-updated",
  "displayName": "listing-revision-attachment",
  "freeformTags": {
    "env": "test"
  },
  "serviceName": "implementation services",
  "type": "IMPLEMENTATION_SERVICE",
  "url": "https://example.com/service"
}`)
	updatedState := ocimock.MustOCIResponseFixture[marketplacepublishersdk.SupportedServiceAttachment](t, `{
  "attachmentType": "SUPPORTED_SERVICES",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description-updated",
  "displayName": "listing-revision-attachment",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "listingRevisionId": "\u003cocid:1\u003e",
  "serviceName": "implementation services",
  "type": "IMPLEMENTATION_SERVICE",
  "url": "https://example.com/service"
}`)
	updatedReadStates := []marketplacepublishersdk.SupportedServiceAttachment{
		ocimock.MustOCIResponseFixture[marketplacepublishersdk.SupportedServiceAttachment](t, `{
  "attachmentType": "SUPPORTED_SERVICES",
  "definedTags": {
    "Operations": {
      "CostCenter": "42"
    }
  },
  "description": "initial description-updated",
  "displayName": "listing-revision-attachment",
  "freeformTags": {
    "env": "test"
  },
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "listingRevisionId": "\u003cocid:1\u003e",
  "serviceName": "implementation services",
  "type": "IMPLEMENTATION_SERVICE",
  "url": "https://example.com/service"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		marketplacepublishersdk.SupportedServiceAttachment,
		marketplacepublishersdk.CreateSupportedServiceAttachment,
		marketplacepublishersdk.UpdateSupportedServiceAttachment,
	]{
		CollectionPath:     "/20241201/listingRevisionAttachments",
		ItemPath:           "/20241201/listingRevisionAttachments/<ocid:2>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState:       &createdState,
		ListShape:          ocimock.ListShapeItems,
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
		ValidateCreateRaw: func(request ocimock.Request) error {
			normalized, err := listingRevisionAttachmentRequestWithoutType(request)
			if err != nil {
				return err
			}
			if err := ocimock.ValidateJSONRequest(normalized, createRequest); err != nil {
				return err
			}
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			normalized, err := listingRevisionAttachmentRequestWithoutType(request)
			if err != nil {
				return err
			}
			return ocimock.ValidateJSONRequest(normalized, updateRequest)
		},
		ValidateDelete: func(request ocimock.Request, _ marketplacepublishersdk.SupportedServiceAttachment) error {
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
			t.Errorf("close ListingRevisionAttachment OCI mock: %v", err)
		}
	})
	sdkClient := marketplacepublishersdk.MarketplacePublisherClient{BaseClient: session.BaseClient()}
	client := newListingRevisionAttachmentServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}, sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*marketplacepublisherv1beta1.ListingRevisionAttachment]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *marketplacepublisherv1beta1.ListingRevisionAttachment) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.AttachmentType, current.Spec.AttachmentType) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.ListingRevisionId, current.Spec.ListingRevisionId) ||
				!reflect.DeepEqual(current.Status.ServiceName, current.Spec.ServiceName) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) ||
				!reflect.DeepEqual(current.Status.Url, current.Spec.Url) {
				return fmt.Errorf("created ListingRevisionAttachment status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *marketplacepublisherv1beta1.ListingRevisionAttachment) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *marketplacepublisherv1beta1.ListingRevisionAttachment) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.AttachmentType, current.Spec.AttachmentType) ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.ServiceName, current.Spec.ServiceName) ||
				!reflect.DeepEqual(current.Status.Type, current.Spec.Type) ||
				!reflect.DeepEqual(current.Status.Url, current.Spec.Url) {
				return fmt.Errorf("updated ListingRevisionAttachment status = %+v", current.Status)
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

func listingRevisionAttachmentRequestWithoutType(request ocimock.Request) (ocimock.Request, error) {
	var body map[string]any
	if err := json.Unmarshal(request.Body, &body); err != nil {
		return request, err
	}
	if body["attachmentType"] != "SUPPORTED_SERVICES" {
		return request, fmt.Errorf("attachmentType = %#v", body["attachmentType"])
	}
	delete(body, "attachmentType")
	normalized, err := json.Marshal(body)
	if err != nil {
		return request, err
	}
	request.Body = normalized
	return request, nil
}
