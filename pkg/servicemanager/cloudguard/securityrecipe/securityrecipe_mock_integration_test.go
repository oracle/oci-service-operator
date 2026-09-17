/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package securityrecipe

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	sdksvc "github.com/oracle/oci-go-sdk/v65/cloudguard"
	apiv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contract evidence: vendored OCI SDK, generated production service manager,
// reviewed formal metadata, and package-owned typed OCI fixtures.
func TestMockIntegrationSecurityRecipeCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.SecurityRecipe](t, `
{
  "metadata": {"name": "mock-securityrecipe", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:1>",
  "description": "OSOK synthetic security recipe",
  "displayName": "mock-displayname-initial",
  "securityPolicies": [
    "<ocid:2>"
  ]
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-securityrecipe")
	resource.Status = apiv1beta1.SecurityRecipeStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateSecurityRecipeDetails](t, `{
  "compartmentId": "<ocid:1>",
  "description": "OSOK synthetic security recipe",
  "displayName": "mock-displayname-initial",
  "securityPolicies": [
    "<ocid:2>"
  ]
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateSecurityRecipeDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.SecurityRecipe](t, `{
  "compartmentId": "<ocid:1>",
  "description": "OSOK synthetic security recipe",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "securityPolicies": [
    "<ocid:2>"
  ],
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.SecurityRecipe](t, `{
  "compartmentId": "<ocid:1>",
  "description": "OSOK synthetic security recipe",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "securityPolicies": [
    "<ocid:2>"
  ],
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)

	creatingState := createdState
	creatingState.LifecycleState = "CREATING"
	updatingState := updatedState
	updatingState.LifecycleState = "UPDATING"
	deletingState := updatedState
	deletingState.LifecycleState = "DELETING"

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.SecurityRecipe, sdksvc.CreateSecurityRecipeDetails, sdksvc.UpdateSecurityRecipeDetails]{
		CollectionPath: "/20200131/securityRecipes", ItemPath: "/20200131/securityRecipes/<ocid:3>",
		CreatePath: "/20200131/securityRecipes", CreateMethod: http.MethodPost,
		UpdatePath: "/20200131/securityRecipes/<ocid:3>", UpdateMethod: http.MethodPut,
		DeletePath: "/20200131/securityRecipes/<ocid:3>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateSecurityRecipeDetails],
		CreatedState:      &creatingState,
		CreatedReadStates: ocimock.StateSequence(creatingState, createdState),
		UpdateRequest:     &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateSecurityRecipeDetails],
		UpdatedState:      &updatingState,
		UpdatedReadStates: ocimock.StateSequence(updatingState, updatedState),
		DeletedState:      &deletingState,
		DeletedReadStates: ocimock.StateSequence(deletingState),
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",

		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateSecurityRecipeDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},

		AdditionalRoutes: []ocimock.Route{},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://cloudguard-cp-api.us-ashburn-1.oci.oraclecloud.com", BasePath: "20200131", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := sdksvc.CloudGuardClient{BaseClient: session.BaseClient()}
	manager := &SecurityRecipeServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newSecurityRecipeRuntimeHooks(manager, sdkClient)
	client := wrapSecurityRecipeGeneratedClient(hooks, defaultSecurityRecipeServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.SecurityRecipe](buildSecurityRecipeGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.SecurityRecipe]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.SecurityRecipe) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created SecurityRecipe status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.SecurityRecipe) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.SecurityRecipe) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated SecurityRecipe status = %+v", current.Status)
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
