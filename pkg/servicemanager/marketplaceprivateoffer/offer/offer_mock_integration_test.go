/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package offer

import (
	"context"
	"fmt"
	marketplaceprivateoffersdk "github.com/oracle/oci-go-sdk/v65/marketplaceprivateoffer"
	marketplaceprivateofferv1beta1 "github.com/oracle/oci-service-operator/api/marketplaceprivateoffer/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationOfferLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newOfferRuntimeTestResource()
	ocimock.InitializeResource(resource, "mock-offer")
	resource.Spec = ocimock.MustJSONFixture[marketplaceprivateofferv1beta1.OfferSpec](t, `{
  "buyerCompartmentId": "\u003cocid:1\u003e",
  "description": "runtime offer",
  "displayName": "offer-runtime",
  "sellerCompartmentId": "\u003cocid:2\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "runtime offer-updated"
}`)
	createRequest := ocimock.MustJSONFixture[marketplaceprivateoffersdk.CreateOfferDetails](t, `{
  "buyerCompartmentId": "\u003cocid:1\u003e",
  "description": "runtime offer",
  "displayName": "offer-runtime",
  "sellerCompartmentId": "\u003cocid:2\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[marketplaceprivateoffersdk.Offer](t, `{
  "buyerCompartmentId": "\u003cocid:1\u003e",
  "description": "runtime offer",
  "displayName": "offer-runtime",
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "sellerCompartmentId": "\u003cocid:2\u003e"
}`)
	createdReadStates := []marketplaceprivateoffersdk.Offer{
		ocimock.MustOCIResponseFixture[marketplaceprivateoffersdk.Offer](t, `{
  "buyerCompartmentId": "\u003cocid:1\u003e",
  "description": "runtime offer",
  "displayName": "offer-runtime",
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "sellerCompartmentId": "\u003cocid:2\u003e"
}`),
	}
	updateRequest := ocimock.MustJSONFixture[marketplaceprivateoffersdk.UpdateOfferDetails](t, `{
  "description": "runtime offer-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[marketplaceprivateoffersdk.Offer](t, `{
  "buyerCompartmentId": "\u003cocid:1\u003e",
  "description": "runtime offer-updated",
  "displayName": "offer-runtime",
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "sellerCompartmentId": "\u003cocid:2\u003e"
}`)
	updatedReadStates := []marketplaceprivateoffersdk.Offer{
		ocimock.MustOCIResponseFixture[marketplaceprivateoffersdk.Offer](t, `{
  "buyerCompartmentId": "\u003cocid:1\u003e",
  "description": "runtime offer-updated",
  "displayName": "offer-runtime",
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "sellerCompartmentId": "\u003cocid:2\u003e"
}`),
	}
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		marketplaceprivateoffersdk.Offer,
		marketplaceprivateoffersdk.CreateOfferDetails,
		marketplaceprivateoffersdk.UpdateOfferDetails,
	]{
		CollectionPath:     "/20220901/offers",
		ItemPath:           "/20220901/offers/<ocid:3>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
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
		ValidateCreate: func(request ocimock.Request, _ marketplaceprivateoffersdk.CreateOfferDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ marketplaceprivateoffersdk.Offer) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20220901", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Offer OCI mock: %v", err)
		}
	})
	sdkClient := marketplaceprivateoffersdk.OfferClient{BaseClient: session.BaseClient()}
	manager := &OfferServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newOfferRuntimeHooksWithOCIClient(sdkClient)
	applyOfferRuntimeHooks(&hooks, sdkClient, nil)
	client := wrapOfferGeneratedClient(hooks, defaultOfferServiceClient{ServiceClient: generatedruntime.NewServiceClient[*marketplaceprivateofferv1beta1.Offer](buildOfferGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*marketplaceprivateofferv1beta1.Offer]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *marketplaceprivateofferv1beta1.Offer) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.BuyerCompartmentId, current.Spec.BuyerCompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.SellerCompartmentId, current.Spec.SellerCompartmentId) {
				return fmt.Errorf("created Offer status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *marketplaceprivateofferv1beta1.Offer) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *marketplaceprivateofferv1beta1.Offer) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated Offer status = %+v", current.Status)
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
