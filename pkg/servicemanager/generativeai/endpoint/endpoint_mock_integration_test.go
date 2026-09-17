/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package endpoint

import (
	"context"
	"fmt"
	generativeaisdk "github.com/oracle/oci-go-sdk/v65/generativeai"
	generativeaiv1beta1 "github.com/oracle/oci-service-operator/api/generativeai/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationEndpointLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &generativeaiv1beta1.Endpoint{Spec: generativeaiv1beta1.EndpointSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", ModelId: "ocid1.generativeaimodel.oc1..mock",
		DedicatedAiClusterId: "ocid1.generativeaidedicatedaicluster.oc1..mock", DisplayName: "osok-mock-generative-ai-endpoint",
	}}
	ocimock.InitializeResource(resource, "mock-endpoint")
	resource.Spec = ocimock.MustJSONFixture[generativeaiv1beta1.EndpointSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "dedicatedAiClusterId": "\u003cocid:2\u003e",
  "displayName": "osok-mock-generative-ai-endpoint",
  "modelId": "\u003cocid:3\u003e"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "contentModerationConfig": {
    "isEnabled": false
  },
  "description": "mock-updated"
}`)
	createRequest := ocimock.MustJSONFixture[generativeaisdk.CreateEndpointDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "dedicatedAiClusterId": "\u003cocid:2\u003e",
  "displayName": "osok-mock-generative-ai-endpoint",
  "modelId": "\u003cocid:3\u003e"
}`)
	createdState := ocimock.MustOCIResponseFixture[generativeaisdk.Endpoint](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "contentModerationConfig": {
    "isEnabled": false
  },
  "dedicatedAiClusterId": "\u003cocid:2\u003e",
  "displayName": "osok-mock-generative-ai-endpoint",
  "id": "\u003cocid:4\u003e",
  "lifecycleState": "ACTIVE",
  "modelId": "\u003cocid:3\u003e"
}`)
	updateRequest := ocimock.MustJSONFixture[generativeaisdk.UpdateEndpointDetails](t, `{
  "contentModerationConfig": {
    "isEnabled": false
  },
  "description": "mock-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[generativeaisdk.Endpoint](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "contentModerationConfig": {
    "isEnabled": false
  },
  "dedicatedAiClusterId": "\u003cocid:2\u003e",
  "description": "mock-updated",
  "displayName": "osok-mock-generative-ai-endpoint",
  "id": "\u003cocid:4\u003e",
  "lifecycleState": "ACTIVE",
  "modelId": "\u003cocid:3\u003e"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		generativeaisdk.Endpoint,
		generativeaisdk.CreateEndpointDetails,
		generativeaisdk.UpdateEndpointDetails,
	]{
		CollectionPath:     "/20231130/endpoints",
		ItemPath:           "/20231130/endpoints/<ocid:4>",
		Operations:         []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest:      &createRequest,
		CreatedState:       &createdState,
		CreatedReadStates:  ocimock.LifecycleStateSequence(t, createdState, "CREATING"),
		ListShape:          ocimock.ListShapeItems,
		UpdateRequest:      &updateRequest,
		UpdatedState:       &updatedState,
		DeletedReadStates:  ocimock.LifecycleStates(t, updatedState, "DELETING"),
		DeleteEndsNotFound: true,
		UpdatedReadStates:  ocimock.LifecycleStateSequence(t, updatedState, "UPDATING"),
		RequireCreateRead:  true,
		RequireUpdateRead:  true,
		RequireDeleteRead:  true,
		CreateStatus:       201,
		UpdateStatus:       200,
		DeleteStatus:       204,
		NotFoundCode:       "NotFound",
		ValidateCreate: func(request ocimock.Request, _ generativeaisdk.CreateEndpointDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ generativeaisdk.Endpoint) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20231130", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close Endpoint OCI mock: %v", err)
		}
	})
	sdkClient := generativeaisdk.GenerativeAiClient{BaseClient: session.BaseClient()}
	manager := &EndpointServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newEndpointRuntimeHooks(manager, sdkClient)
	client := wrapEndpointGeneratedClient(hooks, defaultEndpointServiceClient{ServiceClient: generatedruntime.NewServiceClient[*generativeaiv1beta1.Endpoint](buildEndpointGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*generativeaiv1beta1.Endpoint]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *generativeaiv1beta1.Endpoint) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.DedicatedAiClusterId, current.Spec.DedicatedAiClusterId) ||
				!reflect.DeepEqual(current.Status.DisplayName, current.Spec.DisplayName) ||
				!reflect.DeepEqual(current.Status.ModelId, current.Spec.ModelId) {
				return fmt.Errorf("created Endpoint status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *generativeaiv1beta1.Endpoint) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *generativeaiv1beta1.Endpoint) error {
			if current.Status.Id != "<ocid:4>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:4>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.ContentModerationConfig, current.Spec.ContentModerationConfig) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated Endpoint status = %+v", current.Status)
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
