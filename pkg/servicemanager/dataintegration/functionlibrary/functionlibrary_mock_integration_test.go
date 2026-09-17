/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package functionlibrary

import (
	"context"
	"fmt"
	"testing"

	dataintegrationsdk "github.com/oracle/oci-go-sdk/v65/dataintegration"
	dataintegrationv1beta1 "github.com/oracle/oci-service-operator/api/dataintegration/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestMockIntegrationFunctionLibraryCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &dataintegrationv1beta1.FunctionLibrary{}
	ocimock.InitializeResource(resource, "mock-functionlibrary")
	resource.Spec = ocimock.MustJSONFixture[dataintegrationv1beta1.FunctionLibrarySpec](t, `{
  "workspaceId": "<ocid:1>",
  "aggregatorKey": "aggregator-key",
  "name": "function library create",
  "identifier": "FUNCTION_LIBRARY_CREATE",
  "registryMetadata": {
    "aggregatorKey": "aggregator-key"
  },
  "description": "create"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"description":"updated"}`)
	createRequest := ocimock.MustJSONFixture[dataintegrationsdk.CreateFunctionLibraryDetails](t, `{
  "name": "function library create",
  "identifier": "FUNCTION_LIBRARY_CREATE",
  "description": "create",
  "registryMetadata": {
    "aggregatorKey": "aggregator-key"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[dataintegrationsdk.UpdateFunctionLibraryDetails](t, `{
  "description": "updated",
  "registryMetadata": {
    "aggregatorKey": "aggregator-key",
    "isFavorite": false
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[dataintegrationsdk.FunctionLibrary](t, `{
  "aggregatorKey": "aggregator-key",
  "name": "function library create",
  "identifier": "FUNCTION_LIBRARY_CREATE",
  "description": "create",
  "key": "resource-key"
}`)
	updatedState := ocimock.MustOCIResponseFixture[dataintegrationsdk.FunctionLibrary](t, `{
  "aggregatorKey": "aggregator-key",
  "name": "function library create",
  "identifier": "FUNCTION_LIBRARY_CREATE",
  "description": "updated",
  "key": "resource-key"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[dataintegrationsdk.FunctionLibrary, dataintegrationsdk.CreateFunctionLibraryDetails, dataintegrationsdk.UpdateFunctionLibraryDetails]{
		CollectionPath: "/20200430/workspaces/<ocid:1>/functionLibraries", ItemPath: "/20200430/workspaces/<ocid:1>/functionLibraries/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ dataintegrationsdk.CreateFunctionLibraryDetails) error {
			return ocimock.ValidateRetryToken(request, resource)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://dataintegration.mock.invalid", BasePath: "20200430", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := dataintegrationsdk.DataIntegrationClient{BaseClient: session.BaseClient()}
	manager := &FunctionLibraryServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newFunctionLibraryRuntimeHooks(manager, sdkClient)
	client := wrapFunctionLibraryGeneratedClient(hooks, defaultFunctionLibraryServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dataintegrationv1beta1.FunctionLibrary](buildFunctionLibraryGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dataintegrationv1beta1.FunctionLibrary]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dataintegrationv1beta1.FunctionLibrary) error {
			if current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created FunctionLibrary status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dataintegrationv1beta1.FunctionLibrary) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *dataintegrationv1beta1.FunctionLibrary) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated FunctionLibrary status = %+v", current.Status)
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
