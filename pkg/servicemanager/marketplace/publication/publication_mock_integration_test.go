/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package publication

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	marketplacesdk "github.com/oracle/oci-go-sdk/v65/marketplace"
	marketplacev1beta1 "github.com/oracle/oci-service-operator/api/marketplace/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence:
//   - package-owned typed OCI fixtures declared below
//   - formal contract: formal/controllers/marketplace/publication and formal/imports/marketplace/publication.json
//   - resource runtime: publication_runtime_client.go
//   - OCI SDK: vendor/github.com/oracle/oci-go-sdk/v65/marketplace
func TestMockIntegrationPublicationLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := testPublicationResource()
	ocimock.InitializeResource(resource, "mock-publication")
	const updatedName = "publication-renamed"
	responder, err := newPublicationMockResponder(resource, updatedName)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{
		Host:      "https://marketplace.mock.invalid",
		BasePath:  "20181001",
		Responder: responder,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Publication OCI mock: %v", err)
		}
	})

	sdkClient := marketplacesdk.MarketplaceClient{BaseClient: session.BaseClient()}
	client := newPublicationServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")},
		sdkClient,
	)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*marketplacev1beta1.Publication]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *marketplacev1beta1.Publication) error {
			if current.Status.Id != testPublicationID ||
				current.Status.Name != testPublicationName ||
				current.Status.LifecycleState != string(marketplacesdk.PublicationLifecycleStateActive) ||
				current.Status.PackageType != string(marketplacesdk.PackageTypeEnumImage) {
				return fmt.Errorf("created Publication status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *marketplacev1beta1.Publication) {
			current.Spec.Name = updatedName
		},
		ValidateUpdated: func(current *marketplacev1beta1.Publication) error {
			if current.Status.Id != testPublicationID ||
				current.Status.Name != current.Spec.Name ||
				current.Status.LifecycleState != string(marketplacesdk.PublicationLifecycleStateActive) {
				return fmt.Errorf("updated Publication status = %+v", current.Status)
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

func newPublicationMockResponder(resource *marketplacev1beta1.Publication, updatedName string) (*ocimock.CRUDResponder[marketplacesdk.Publication], error) {
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[marketplacesdk.Publication]{
		CollectionPath:     "/20181001/publications",
		ItemPath:           "/20181001/publications/" + testPublicationID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		List: func(request ocimock.Request, present bool, state marketplacesdk.Publication) (ocimock.Response, error) {
			if got := request.URL.Query().Get("compartmentId"); got != resource.Spec.CompartmentId {
				return ocimock.Response{}, fmt.Errorf("ListPublications compartmentId = %q", got)
			}
			if got := request.URL.Query().Get("listingType"); got != resource.Spec.ListingType {
				return ocimock.Response{}, fmt.Errorf("ListPublications listingType = %q", got)
			}
			if !present {
				return ocimock.JSONResponse(http.StatusOK, []marketplacesdk.PublicationSummary{})
			}
			return ocimock.JSONResponse(http.StatusOK, []marketplacesdk.PublicationSummary{
				sdkPublicationSummary(*state.Id, *state.Name, state.LifecycleState),
			})
		},
		Create: func(request ocimock.Request) (marketplacesdk.Publication, ocimock.Response, error) {
			var details marketplacesdk.CreatePublicationDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return marketplacesdk.Publication{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return marketplacesdk.Publication{}, ocimock.Response{}, err
			}
			if details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId ||
				details.Name == nil || *details.Name != resource.Spec.Name ||
				details.ListingType != marketplacesdk.ListingTypePartner ||
				!boolPointerValue(details.IsAgreementAcknowledged) {
				return marketplacesdk.Publication{}, ocimock.Response{}, fmt.Errorf("CreatePublication details = %+v", details)
			}
			pkg, ok := details.PackageDetails.(marketplacesdk.CreateImagePublicationPackage)
			if !ok || pkg.ImageId == nil || *pkg.ImageId != resource.Spec.PackageDetails.ImageId ||
				pkg.PackageVersion == nil || *pkg.PackageVersion != resource.Spec.PackageDetails.PackageVersion {
				return marketplacesdk.Publication{}, ocimock.Response{}, fmt.Errorf("CreatePublication package details = %#v", details.PackageDetails)
			}
			if request.Header.Get("opc-retry-token") == "" {
				return marketplacesdk.Publication{}, ocimock.Response{}, fmt.Errorf("CreatePublication opc-retry-token is empty")
			}
			state := sdkPublication(testPublicationID, testPublicationName, marketplacesdk.PublicationLifecycleStateActive)
			response, err := ocimock.JSONResponse(http.StatusCreated, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state marketplacesdk.Publication) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state marketplacesdk.Publication) (marketplacesdk.Publication, ocimock.Response, error) {
			var details marketplacesdk.UpdatePublicationDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return state, ocimock.Response{}, err
			}
			if details.Name == nil || *details.Name != updatedName ||
				details.ShortDescription != nil || details.LongDescription != nil ||
				details.SupportContacts != nil || details.DefinedTags != nil || details.FreeformTags != nil {
				return state, ocimock.Response{}, fmt.Errorf("UpdatePublication details = %+v", details)
			}
			state.Name = details.Name
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(request ocimock.Request, _ marketplacesdk.Publication) (ocimock.Response, error) {
			if len(request.Body) != 0 {
				return ocimock.Response{}, fmt.Errorf("DeletePublication body = %s", request.Body)
			}
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
