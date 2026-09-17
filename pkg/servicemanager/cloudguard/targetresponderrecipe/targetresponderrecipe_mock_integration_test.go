/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package targetresponderrecipe

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

func TestMockIntegrationTargetResponderRecipeCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &cloudguardv1beta1.TargetResponderRecipe{}
	ocimock.InitializeResource(resource, "mock-target-responder-recipe")
	resource.Spec = ocimock.MustJSONFixture[cloudguardv1beta1.TargetResponderRecipeSpec](t, `{
  "targetId":"<ocid:1>","compartmentId":"<ocid:2>","responderRecipeId":"<ocid:3>","responderRules":[]
}`)
	updatedSpec := resource.Spec
	updatedSpec.ResponderRules = []cloudguardv1beta1.TargetResponderRecipeResponderRuleFields{{ResponderRuleId: "rule-1"}}
	createRequest := ocimock.MustJSONFixture[cloudguardsdk.CreateTargetResponderRecipeDetails](t, `{"responderRecipeId":"<ocid:3>"}`)
	createdState := ocimock.MustOCIResponseFixture[cloudguardsdk.TargetResponderRecipe](t, `{
  "id":"<ocid:4>","responderRecipeId":"<ocid:3>","compartmentId":"<ocid:2>",
  "displayName":"mock target responder","description":"mock responder","owner":"CUSTOMER","responderRules":[]
}`)
	updateRequest := ocimock.MustJSONFixture[cloudguardsdk.UpdateTargetResponderRecipeDetails](t, `{
  "responderRules":[{"responderRuleId":"rule-1"}]
}`)
	updatedState := ocimock.MustOCIResponseFixture[cloudguardsdk.TargetResponderRecipe](t, `{
  "id":"<ocid:4>","responderRecipeId":"<ocid:3>","compartmentId":"<ocid:2>",
  "displayName":"mock target responder","description":"mock responder","owner":"CUSTOMER",
  "responderRules":[{"responderRuleId":"rule-1","details":{"configurations":[]}}]
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[cloudguardsdk.TargetResponderRecipe, cloudguardsdk.CreateTargetResponderRecipeDetails, cloudguardsdk.UpdateTargetResponderRecipeDetails]{
		CollectionPath: "/20200131/targets/<ocid:1>/targetResponderRecipes", ItemPath: "/20200131/targets/<ocid:1>/targetResponderRecipes/<ocid:4>",
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
	manager := &TargetResponderRecipeServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newTargetResponderRecipeRuntimeHooks(manager, sdkClient)
	client := wrapTargetResponderRecipeGeneratedClient(hooks, defaultTargetResponderRecipeServiceClient{ServiceClient: generatedruntime.NewServiceClient[*cloudguardv1beta1.TargetResponderRecipe](buildTargetResponderRecipeGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*cloudguardv1beta1.TargetResponderRecipe]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *cloudguardv1beta1.TargetResponderRecipe) error {
			if current.Status.Id != "<ocid:4>" || current.Status.ResponderRecipeId != current.Spec.ResponderRecipeId || string(current.Status.OsokStatus.Reason) != "Active" {
				return fmt.Errorf("created TargetResponderRecipe status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *cloudguardv1beta1.TargetResponderRecipe) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *cloudguardv1beta1.TargetResponderRecipe) error {
			if len(current.Status.ResponderRules) != 1 || current.Status.ResponderRules[0].ResponderRuleId != "rule-1" {
				return fmt.Errorf("updated TargetResponderRecipe status = %+v", current.Status)
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
