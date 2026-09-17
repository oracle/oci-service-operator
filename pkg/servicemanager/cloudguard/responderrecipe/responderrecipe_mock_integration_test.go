/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package responderrecipe

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
func TestMockIntegrationResponderRecipeCRUD(t *testing.T) {
	t.Parallel()
	resourceValue := ocimock.MustJSONFixture[apiv1beta1.ResponderRecipe](t, `
{
  "metadata": {"name": "mock-responderrecipe", "namespace": "default"},
  "spec": {
  "compartmentId": "<ocid:1>",
  "description": "OSOK synthetic responder recipe",
  "displayName": "mock-displayname-initial",
  "sourceResponderRecipeId": "<ocid:2>"
}
}
`)
	resource := &resourceValue
	ocimock.InitializeResource(resource, "mock-responderrecipe")
	resource.Status = apiv1beta1.ResponderRecipeStatus{}
	createRequest := ocimock.MustJSONFixture[sdksvc.CreateResponderRecipeDetails](t, `{
  "compartmentId": "<ocid:1>",
  "description": "OSOK synthetic responder recipe",
  "displayName": "mock-displayname-initial",
  "sourceResponderRecipeId": "<ocid:2>"
}`)
	updateRequest := ocimock.MustJSONFixture[sdksvc.UpdateResponderRecipeDetails](t, `{
  "displayName": "mock-displayname-updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[sdksvc.ResponderRecipe](t, `{
  "compartmentId": "<ocid:1>",
  "description": "OSOK synthetic responder recipe",
  "displayName": "mock-displayname-initial",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "sourceResponderRecipeId": "<ocid:2>",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)
	updatedState := ocimock.MustOCIResponseFixture[sdksvc.ResponderRecipe](t, `{
  "compartmentId": "<ocid:1>",
  "description": "OSOK synthetic responder recipe",
  "displayName": "mock-displayname-updated",
  "id": "<ocid:3>",
  "key": "<ocid:3>",
  "lifecycleState": "ACTIVE",
  "sourceResponderRecipeId": "<ocid:2>",
  "state": "ACTIVE",
  "status": "ACTIVE"
}`)

	creatingState := createdState
	creatingState.LifecycleState = "CREATING"
	updatingState := updatedState
	updatingState.LifecycleState = "UPDATING"
	deletingState := updatedState
	deletingState.LifecycleState = "DELETING"

	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[sdksvc.ResponderRecipe, sdksvc.CreateResponderRecipeDetails, sdksvc.UpdateResponderRecipeDetails]{
		CollectionPath: "/20200131/responderRecipes", ItemPath: "/20200131/responderRecipes/<ocid:3>",
		CreatePath: "/20200131/responderRecipes", CreateMethod: http.MethodPost,
		UpdatePath: "/20200131/responderRecipes/<ocid:3>", UpdateMethod: http.MethodPut,
		DeletePath: "/20200131/responderRecipes/<ocid:3>", DeleteMethod: http.MethodDelete,
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CompareCreate: ocimock.CompareJSONSubset[sdksvc.CreateResponderRecipeDetails],
		CreatedState:      &creatingState,
		CreatedReadStates: ocimock.StateSequence(creatingState, createdState),
		UpdateRequest:     &updateRequest, CompareUpdate: ocimock.CompareJSONSubset[sdksvc.UpdateResponderRecipeDetails],
		UpdatedState:      &updatingState,
		UpdatedReadStates: ocimock.StateSequence(updatingState, updatedState),
		DeletedState:      &deletingState,
		DeletedReadStates: ocimock.StateSequence(deletingState),
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotAuthorizedOrNotFound",

		ValidateCreate: func(request ocimock.Request, _ sdksvc.CreateResponderRecipeDetails) error {
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
	manager := &ResponderRecipeServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newResponderRecipeRuntimeHooks(manager, sdkClient)
	client := wrapResponderRecipeGeneratedClient(hooks, defaultResponderRecipeServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiv1beta1.ResponderRecipe](buildResponderRecipeGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiv1beta1.ResponderRecipe]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiv1beta1.ResponderRecipe) error {
			if current.Status.OsokStatus.Ocid == "" || current.Status.LifecycleState != "ACTIVE" || current.Status.DisplayName != "mock-displayname-initial" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("created ResponderRecipe status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiv1beta1.ResponderRecipe) {
			ocimock.MustMergeJSONFixture(t, &current.Spec, `{
  "displayName": "mock-displayname-updated"
}`)
		},
		ValidateUpdated: func(current *apiv1beta1.ResponderRecipe) error {
			if current.Status.DisplayName != "mock-displayname-updated" || current.Status.OsokStatus.Async.Current != nil {
				return fmt.Errorf("updated ResponderRecipe status = %+v", current.Status)
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
