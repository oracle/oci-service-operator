/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package detectorrecipe

import (
	"context"
	"fmt"
	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationDetectorRecipeLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &cloudguardv1beta1.DetectorRecipe{}
	ocimock.InitializeResource(resource, "mock-detectorrecipe")
	resource.Spec = ocimock.MustJSONFixture[cloudguardv1beta1.DetectorRecipeSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "OSOK synthetic detector recipe",
  "displayName": "osok-mock-detector-recipe"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "OSOK synthetic detector recipe-updated"
}`)
	createRequest := ocimock.MustJSONFixture[cloudguardsdk.CreateDetectorRecipeDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "OSOK synthetic detector recipe",
  "displayName": "osok-mock-detector-recipe"
}`)
	createdState := ocimock.MustOCIResponseFixture[cloudguardsdk.DetectorRecipe](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "OSOK synthetic detector recipe",
  "displayName": "osok-mock-detector-recipe",
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE"
}`)
	updateRequest := ocimock.MustJSONFixture[cloudguardsdk.UpdateDetectorRecipeDetails](t, `{
  "description": "OSOK synthetic detector recipe-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[cloudguardsdk.DetectorRecipe](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "OSOK synthetic detector recipe-updated",
  "displayName": "osok-mock-detector-recipe",
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		cloudguardsdk.DetectorRecipe,
		cloudguardsdk.CreateDetectorRecipeDetails,
		cloudguardsdk.UpdateDetectorRecipeDetails,
	]{
		CollectionPath:    "/20200131/detectorRecipes",
		ItemPath:          "/20200131/detectorRecipes/<ocid:2>",
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
		ValidateCreate: func(request ocimock.Request, _ cloudguardsdk.CreateDetectorRecipeDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ cloudguardsdk.DetectorRecipe) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20200131", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close DetectorRecipe OCI mock: %v", err)
		}
	})
	sdkClient := cloudguardsdk.CloudGuardClient{BaseClient: session.BaseClient()}
	manager := &DetectorRecipeServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newDetectorRecipeRuntimeHooks(manager, sdkClient)
	client := wrapDetectorRecipeGeneratedClient(hooks, defaultDetectorRecipeServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*cloudguardv1beta1.DetectorRecipe](buildDetectorRecipeGeneratedRuntimeConfig(manager, hooks)),
	})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*cloudguardv1beta1.DetectorRecipe]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *cloudguardv1beta1.DetectorRecipe) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) {
				return fmt.Errorf("created DetectorRecipe status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *cloudguardv1beta1.DetectorRecipe) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *cloudguardv1beta1.DetectorRecipe) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated DetectorRecipe status = %+v", current.Status)
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
