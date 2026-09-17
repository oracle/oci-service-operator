/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package detectorrecipedetectorrule

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

func TestMockIntegrationDetectorRecipeDetectorRuleCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &cloudguardv1beta1.DetectorRecipeDetectorRule{}
	ocimock.InitializeResource(resource, "mock-detector-rule")
	resource.Spec = ocimock.MustJSONFixture[cloudguardv1beta1.DetectorRecipeDetectorRuleSpec](t, `{
  "detectorRecipeId":"<ocid:1>","compartmentId":"<ocid:2>",
  "details":{"name":"osok-mock-rule","description":"mock create","isEnabled":true,"riskLevel":"LOW"}
}`)
	updatedSpec := resource.Spec
	updatedSpec.Details.Description = "mock update"
	updatedSpec.Details.IsEnabled = false
	createRequest := ocimock.MustJSONFixture[cloudguardsdk.CreateDetectorRecipeDetectorRuleDetails](t, `{
  "details":{"name":"osok-mock-rule","description":"mock create","isEnabled":true,"riskLevel":"LOW"}
}`)
	createdState := ocimock.MustOCIResponseFixture[cloudguardsdk.DetectorRecipeDetectorRule](t, `{
  "detectorRuleId":"rule-1","detector":"ACTIVITY","serviceType":"COMPUTE","resourceType":"INSTANCE",
  "description":"mock create","details":{"name":"osok-mock-rule","description":"mock create","isEnabled":true,"riskLevel":"LOW"},
  "lifecycleState":"ACTIVE"
}`)
	updateRequest := ocimock.MustJSONFixture[cloudguardsdk.UpdateDetectorRecipeDetectorRuleDetails](t, `{
  "details":{"description":"mock update","isEnabled":false,"riskLevel":"LOW"}
}`)
	updatedState := ocimock.MustOCIResponseFixture[cloudguardsdk.DetectorRecipeDetectorRule](t, `{
  "detectorRuleId":"rule-1","detector":"ACTIVITY","serviceType":"COMPUTE","resourceType":"INSTANCE",
  "description":"mock update","details":{"name":"osok-mock-rule","description":"mock update","isEnabled":false,"riskLevel":"LOW"},
  "lifecycleState":"ACTIVE"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[cloudguardsdk.DetectorRecipeDetectorRule, cloudguardsdk.CreateDetectorRecipeDetectorRuleDetails, cloudguardsdk.UpdateDetectorRecipeDetectorRuleDetails]{
		CollectionPath: "/20200131/detectorRecipes/<ocid:1>/detectorRules", ItemPath: "/20200131/detectorRecipes/<ocid:1>/detectorRules/rule-1",
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
	manager := &DetectorRecipeDetectorRuleServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newDetectorRecipeDetectorRuleRuntimeHooks(manager, sdkClient)
	client := wrapDetectorRecipeDetectorRuleGeneratedClient(hooks, defaultDetectorRecipeDetectorRuleServiceClient{ServiceClient: generatedruntime.NewServiceClient[*cloudguardv1beta1.DetectorRecipeDetectorRule](buildDetectorRecipeDetectorRuleGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*cloudguardv1beta1.DetectorRecipeDetectorRule]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *cloudguardv1beta1.DetectorRecipeDetectorRule) error {
			if current.Status.DetectorRuleId != "rule-1" || string(current.Status.OsokStatus.Ocid) != "rule-1" || current.Status.Details.Description != current.Spec.Details.Description {
				return fmt.Errorf("created DetectorRecipeDetectorRule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *cloudguardv1beta1.DetectorRecipeDetectorRule) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *cloudguardv1beta1.DetectorRecipeDetectorRule) error {
			if current.Status.Details.Description != current.Spec.Details.Description || current.Status.Details.IsEnabled != current.Spec.Details.IsEnabled {
				return fmt.Errorf("updated DetectorRecipeDetectorRule status = %+v", current.Status)
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
