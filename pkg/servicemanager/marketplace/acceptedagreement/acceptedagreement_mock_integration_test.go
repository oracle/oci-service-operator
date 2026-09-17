/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package acceptedagreement

import (
	"context"
	"fmt"
	marketplacesdk "github.com/oracle/oci-go-sdk/v65/marketplace"
	marketplacev1beta1 "github.com/oracle/oci-service-operator/api/marketplace/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationAcceptedAgreementLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &marketplacev1beta1.AcceptedAgreement{Spec: marketplacev1beta1.AcceptedAgreementSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", ListingId: "ocid1.appcataloglisting.oc1..mock",
		PackageVersion: "1.0", AgreementId: "ocid1.marketplaceagreement.oc1..mock", Signature: "synthetic-signature", DisplayName: "osok-mock-agreement",
	}}
	ocimock.InitializeResource(resource, "mock-acceptedagreement")
	resource.Spec = ocimock.MustJSONFixture[marketplacev1beta1.AcceptedAgreementSpec](t, `{
  "agreementId": "\u003cocid:1\u003e",
  "compartmentId": "\u003cocid:2\u003e",
  "displayName": "osok-mock-agreement",
  "listingId": "\u003cocid:3\u003e",
  "packageVersion": "1.0",
  "signature": "synthetic-signature"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "84"
    }
  },
  "displayName": "osok-mock-agreement-updated",
  "freeformTags": {
    "env": "prod"
  }
}`)
	createRequest := ocimock.MustJSONFixture[marketplacesdk.CreateAcceptedAgreementDetails](t, `{
  "agreementId": "\u003cocid:1\u003e",
  "compartmentId": "\u003cocid:2\u003e",
  "displayName": "osok-mock-agreement",
  "listingId": "\u003cocid:3\u003e",
  "packageVersion": "1.0",
  "signature": "synthetic-signature"
}`)
	createdState := ocimock.MustOCIResponseFixture[marketplacesdk.AcceptedAgreement](t, `{
  "agreementId": "\u003cocid:1\u003e",
  "compartmentId": "\u003cocid:2\u003e",
  "displayName": "osok-mock-agreement",
  "id": "\u003cocid:4\u003e",
  "lifecycleState": "ACTIVE",
  "listingId": "\u003cocid:3\u003e",
  "packageVersion": "1.0",
  "signature": "synthetic-signature"
}`)
	updateRequest := ocimock.MustJSONFixture[marketplacesdk.UpdateAcceptedAgreementDetails](t, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "84"
    }
  },
  "displayName": "osok-mock-agreement-updated",
  "freeformTags": {
    "env": "prod"
  }
}`)
	updatedState := createdState
	updatedState.DefinedTags = updateRequest.DefinedTags
	updatedState.DisplayName = updateRequest.DisplayName
	updatedState.FreeformTags = updateRequest.FreeformTags
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		marketplacesdk.AcceptedAgreement,
		marketplacesdk.CreateAcceptedAgreementDetails,
		marketplacesdk.UpdateAcceptedAgreementDetails,
	]{
		CollectionPath:    "/20181001/acceptedAgreements",
		ItemPath:          "/20181001/acceptedAgreements/<ocid:4>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeArray,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		UpdatedReadStates: []marketplacesdk.AcceptedAgreement{updatedState},
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ marketplacesdk.CreateAcceptedAgreementDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ marketplacesdk.AcceptedAgreement) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20181001", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close AcceptedAgreement OCI mock: %v", err)
		}
	})
	sdkClient := marketplacesdk.MarketplaceClient{BaseClient: session.BaseClient()}
	manager := &AcceptedAgreementServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newAcceptedAgreementRuntimeHooks(manager, sdkClient)
	client := wrapAcceptedAgreementGeneratedClient(hooks, defaultAcceptedAgreementServiceClient{ServiceClient: generatedruntime.NewServiceClient[*marketplacev1beta1.AcceptedAgreement](buildAcceptedAgreementGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*marketplacev1beta1.AcceptedAgreement]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *marketplacev1beta1.AcceptedAgreement) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				!reflect.DeepEqual(current.Status.AgreementId, current.Spec.AgreementId) ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.ListingId, current.Spec.ListingId) ||
				!reflect.DeepEqual(current.Status.PackageVersion, current.Spec.PackageVersion) {
				return fmt.Errorf("created AcceptedAgreement status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *marketplacev1beta1.AcceptedAgreement) {
			current.Spec = updatedSpec
		},
		ValidateUpdated: func(current *marketplacev1beta1.AcceptedAgreement) error {
			if current.Status.Id != "<ocid:4>" ||
				current.Status.AppliedSignature != current.Spec.Signature ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) {
				return fmt.Errorf("updated AcceptedAgreement status = %+v", current.Status)
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
