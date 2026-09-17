/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package targetdetectorrecipe

import (
	"context"
	"fmt"
	"testing"

	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationTargetDetectorRecipeCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &cloudguardv1beta1.TargetDetectorRecipe{}
	ocimock.InitializeResource(resource, "mock-target-detector-recipe")
	resource.Spec = ocimock.MustJSONFixture[cloudguardv1beta1.TargetDetectorRecipeSpec](t, `{
  "targetId":"<ocid:1>","compartmentId":"<ocid:2>","detectorRecipeId":"<ocid:3>"
}`)
	updatedSpec := resource.Spec
	updatedSpec.DetectorRules = []cloudguardv1beta1.TargetDetectorRecipeDetectorRuleFields{{DetectorRuleId: "rule-1"}}
	createRequest := ocimock.MustJSONFixture[cloudguardsdk.CreateTargetDetectorRecipeDetails](t, `{"detectorRecipeId":"<ocid:3>"}`)
	createdState := ocimock.MustOCIResponseFixture[cloudguardsdk.TargetDetectorRecipe](t, `{
  "id":"<ocid:4>","displayName":"mock target detector","compartmentId":"<ocid:2>",
  "detectorRecipeId":"<ocid:3>","owner":"CUSTOMER","detector":"ACTIVITY","detectorRules":[],"lifecycleState":"ACTIVE"
}`)
	updateRequest := ocimock.MustJSONFixture[cloudguardsdk.UpdateTargetDetectorRecipeDetails](t, `{
  "isValidationOnlyQuery":false,"detectorRules":[{"detectorRuleId":"rule-1"}]
}`)
	updatedState := ocimock.MustOCIResponseFixture[cloudguardsdk.TargetDetectorRecipe](t, `{
  "id":"<ocid:4>","displayName":"mock target detector","compartmentId":"<ocid:2>",
  "detectorRecipeId":"<ocid:3>","owner":"CUSTOMER","detector":"ACTIVITY",
  "detectorRules":[{"detectorRuleId":"rule-1","details":{"conditionGroups":[]}}],"lifecycleState":"ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[cloudguardsdk.TargetDetectorRecipe, cloudguardsdk.CreateTargetDetectorRecipeDetails, cloudguardsdk.UpdateTargetDetectorRecipeDetails]{
		CollectionPath: "/20200131/targets/<ocid:1>/targetDetectorRecipes", ItemPath: "/20200131/targets/<ocid:1>/targetDetectorRecipes/<ocid:4>",
		Operations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		ValidateCreateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState,
		ListShape:         ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudguard.mock.invalid", BasePath: "20200131", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := cloudguardsdk.CloudGuardClient{BaseClient: session.BaseClient()}
	manager := &TargetDetectorRecipeServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newTargetDetectorRecipeRuntimeHooks(manager, sdkClient)
	client := wrapTargetDetectorRecipeGeneratedClient(hooks, defaultTargetDetectorRecipeServiceClient{ServiceClient: generatedruntime.NewServiceClient[*cloudguardv1beta1.TargetDetectorRecipe](buildTargetDetectorRecipeGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*cloudguardv1beta1.TargetDetectorRecipe]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *cloudguardv1beta1.TargetDetectorRecipe) error {
			if current.Status.Id != "<ocid:4>" || current.Status.DetectorRecipeId != current.Spec.DetectorRecipeId || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("created TargetDetectorRecipe status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *cloudguardv1beta1.TargetDetectorRecipe) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *cloudguardv1beta1.TargetDetectorRecipe) error {
			if len(current.Status.DetectorRules) != 1 || current.Status.DetectorRules[0].DetectorRuleId != "rule-1" {
				return fmt.Errorf("updated TargetDetectorRecipe status = %+v", current.Status)
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
