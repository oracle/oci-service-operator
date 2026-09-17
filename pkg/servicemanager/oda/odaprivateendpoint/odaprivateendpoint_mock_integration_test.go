/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package odaprivateendpoint

import (
	"context"
	"fmt"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationOdaPrivateEndpointLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := newOdaPrivateEndpointResource("osok-mock-oda-endpoint")
	ocimock.InitializeResource(resource, "mock-odaprivateendpoint")
	resource.Spec = ocimock.MustJSONFixture[odav1beta1.OdaPrivateEndpointSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-oda-endpoint",
  "subnetId": "\u003cocid:2\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "84"
    }
  },
  "description": "updated private endpoint",
  "displayName": "osok-mock-oda-endpoint-updated",
  "freeformTags": {
    "env": "prod"
  },
  "nsgIds": [
    "\u003cocid:4\u003e"
  ]
}`)
	createRequest := ocimock.MustJSONFixture[odasdk.CreateOdaPrivateEndpointDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-oda-endpoint",
  "subnetId": "\u003cocid:2\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[odasdk.OdaPrivateEndpoint](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "displayName": "osok-mock-oda-endpoint",
  "id": "\u003cocid:3\u003e",
  "lifecycleState": "ACTIVE",
  "subnetId": "\u003cocid:2\u003e"
}`)
	updateRequest := ocimock.MustJSONFixture[odasdk.UpdateOdaPrivateEndpointDetails](t, `{
  "definedTags": {
    "Operations": {
      "CostCenter": "84"
    }
  },
  "description": "updated private endpoint",
  "displayName": "osok-mock-oda-endpoint-updated",
  "freeformTags": {
    "env": "prod"
  },
  "nsgIds": [
    "\u003cocid:4\u003e"
  ]
}`)
	updatedState := createdState
	updatedState.DefinedTags = updateRequest.DefinedTags
	updatedState.Description = updateRequest.Description
	updatedState.DisplayName = updateRequest.DisplayName
	updatedState.FreeformTags = updateRequest.FreeformTags
	updatedState.NsgIds = updateRequest.NsgIds
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		odasdk.OdaPrivateEndpoint,
		odasdk.CreateOdaPrivateEndpointDetails,
		odasdk.UpdateOdaPrivateEndpointDetails,
	]{
		CollectionPath:    "/20190506/odaPrivateEndpoints",
		ItemPath:          "/20190506/odaPrivateEndpoints/<ocid:3>",
		Operations:        []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:     &createRequest,
		CreatedState:      &createdState,
		ListShape:         ocimock.ListShapeItems,
		UpdateRequest:     &updateRequest,
		UpdatedState:      &updatedState,
		UpdatedReadStates: []odasdk.OdaPrivateEndpoint{updatedState},
		RequireCreateRead: true,
		RequireUpdateRead: true,
		RequireDeleteRead: true,
		CreateStatus:      201,
		UpdateStatus:      202,
		DeleteStatus:      204,
		NotFoundCode:      "NotFound",
		ValidateCreate: func(request ocimock.Request, _ odasdk.CreateOdaPrivateEndpointDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ odasdk.OdaPrivateEndpoint) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20190506", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close OdaPrivateEndpoint OCI mock: %v", err)
		}
	})
	sdkClient := odasdk.ManagementClient{BaseClient: session.BaseClient()}
	client := newOdaPrivateEndpointServiceClientWithOCIClient(sdkClient)
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*odav1beta1.OdaPrivateEndpoint]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *odav1beta1.OdaPrivateEndpoint) error {
			if current.Status.Id != "<ocid:3>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.SubnetId, current.Spec.SubnetId) {
				return fmt.Errorf("created OdaPrivateEndpoint status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *odav1beta1.OdaPrivateEndpoint) {
			current.Spec = updatedSpec
		},
		ValidateUpdated: func(current *odav1beta1.OdaPrivateEndpoint) error {
			if current.Status.Id != "<ocid:3>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.DefinedTags, current.Spec.DefinedTags) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.FreeformTags, current.Spec.FreeformTags) ||
				!reflect.DeepEqual(current.Status.NsgIds, current.Spec.NsgIds) ||
				!reflect.DeepEqual(current.Status.SubnetId, current.Spec.SubnetId) {
				return fmt.Errorf("updated OdaPrivateEndpoint status = %+v", current.Status)
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
