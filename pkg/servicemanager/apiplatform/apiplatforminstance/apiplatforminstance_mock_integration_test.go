/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package apiplatforminstance

import (
	"context"
	"fmt"
	apiplatformsdk "github.com/oracle/oci-go-sdk/v65/apiplatform"
	apiplatformv1beta1 "github.com/oracle/oci-service-operator/api/apiplatform/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

// Explicit typed service-manager lifecycle; package-owned typed fixtures define the exercised behavior.
func TestMockIntegrationApiPlatformInstanceLifecycleCRUD(t *testing.T) {
	t.Parallel()

	resource := &apiplatformv1beta1.ApiPlatformInstance{Spec: apiplatformv1beta1.ApiPlatformInstanceSpec{
		Name: "osok-mock-api-platform", CompartmentId: "ocid1.compartment.oc1..mock", Description: "synthetic API Platform instance",
	}}
	ocimock.InitializeResource(resource, "mock-apiplatforminstance")
	resource.Spec = ocimock.MustJSONFixture[apiplatformv1beta1.ApiPlatformInstanceSpec](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "synthetic API Platform instance",
  "name": "osok-mock-api-platform"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{
  "description": "synthetic API Platform instance-updated"
}`)
	createRequest := ocimock.MustJSONFixture[apiplatformsdk.CreateApiPlatformInstanceDetails](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "synthetic API Platform instance",
  "name": "osok-mock-api-platform"
}`)
	createdState := ocimock.MustOCIResponseFixture[apiplatformsdk.ApiPlatformInstance](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "synthetic API Platform instance",
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "name": "osok-mock-api-platform"
}`)
	updateRequest := ocimock.MustJSONFixture[apiplatformsdk.UpdateApiPlatformInstanceDetails](t, `{
  "description": "synthetic API Platform instance-updated"
}`)
	updatedState := ocimock.MustOCIResponseFixture[apiplatformsdk.ApiPlatformInstance](t, `{
  "compartmentId": "\u003cocid:1\u003e",
  "description": "synthetic API Platform instance-updated",
  "id": "\u003cocid:2\u003e",
  "lifecycleState": "ACTIVE",
  "name": "osok-mock-api-platform"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[
		apiplatformsdk.ApiPlatformInstance,
		apiplatformsdk.CreateApiPlatformInstanceDetails,
		apiplatformsdk.UpdateApiPlatformInstanceDetails,
	]{
		CollectionPath:     "/20240829/apiPlatformInstances",
		ItemPath:           "/20240829/apiPlatformInstances/<ocid:2>",
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
		ValidateCreate: func(request ocimock.Request, _ apiplatformsdk.CreateApiPlatformInstanceDetails) error {
			if request.Header.Get("opc-retry-token") == "" {
				return fmt.Errorf("create retry token is empty")
			}
			return nil
		},
		ValidateDelete: func(request ocimock.Request, _ apiplatformsdk.ApiPlatformInstance) error {
			if len(request.Body) != 0 {
				return fmt.Errorf("delete body = %s", request.Body)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://oci.mock.invalid", BasePath: "20240829", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close ApiPlatformInstance OCI mock: %v", err)
		}
	})
	sdkClient := apiplatformsdk.ApiPlatformClient{BaseClient: session.BaseClient()}
	manager := &ApiPlatformInstanceServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("synthetic-integration")}}
	hooks := newApiPlatformInstanceRuntimeHooks(manager, sdkClient)
	client := wrapApiPlatformInstanceGeneratedClient(hooks, defaultApiPlatformInstanceServiceClient{ServiceClient: generatedruntime.NewServiceClient[*apiplatformv1beta1.ApiPlatformInstance](buildApiPlatformInstanceGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*apiplatformv1beta1.ApiPlatformInstance]{
		Resource:      resource,
		Client:        client,
		CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *apiplatformv1beta1.ApiPlatformInstance) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.CompartmentId, current.Spec.CompartmentId) ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) ||
				!reflect.DeepEqual(current.Status.Name, current.Spec.Name) {
				return fmt.Errorf("created ApiPlatformInstance status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *apiplatformv1beta1.ApiPlatformInstance) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *apiplatformv1beta1.ApiPlatformInstance) error {
			if current.Status.Id != "<ocid:2>" ||
				string(current.Status.OsokStatus.Ocid) != "<ocid:2>" ||
				current.Status.LifecycleState != "ACTIVE" ||
				!reflect.DeepEqual(current.Status.Description, current.Spec.Description) {
				return fmt.Errorf("updated ApiPlatformInstance status = %+v", current.Status)
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
