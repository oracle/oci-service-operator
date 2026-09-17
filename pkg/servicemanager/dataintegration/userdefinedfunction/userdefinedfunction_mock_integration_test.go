/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package userdefinedfunction

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

func TestMockIntegrationUserDefinedFunctionCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &dataintegrationv1beta1.UserDefinedFunction{}
	ocimock.InitializeResource(resource, "mock-userdefinedfunction")
	resource.Spec = ocimock.MustJSONFixture[dataintegrationv1beta1.UserDefinedFunctionSpec](t, `{
  "workspaceId": "<ocid:1>",
  "name": "function create",
  "identifier": "FUNCTION_CREATE",
  "registryMetadata": {
    "aggregatorKey": "aggregator-key"
  },
  "description": "create"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"description":"updated"}`)
	createRequest := ocimock.MustJSONFixture[dataintegrationsdk.CreateUserDefinedFunctionDetails](t, `{
  "name": "function create",
  "identifier": "FUNCTION_CREATE",
  "description": "create",
  "registryMetadata": {
    "aggregatorKey": "aggregator-key"
  }
}`)
	updateRequest := ocimock.MustJSONFixture[dataintegrationsdk.UpdateUserDefinedFunctionDetails](t, `{
  "description": "updated",
  "registryMetadata": {
    "aggregatorKey": "aggregator-key",
    "isFavorite": false
  }
}`)
	createdState := ocimock.MustOCIResponseFixture[dataintegrationsdk.UserDefinedFunction](t, `{
  "name": "function create",
  "identifier": "FUNCTION_CREATE",
  "description": "create",
  "key": "resource-key"
}`)
	updatedState := ocimock.MustOCIResponseFixture[dataintegrationsdk.UserDefinedFunction](t, `{
  "name": "function create",
  "identifier": "FUNCTION_CREATE",
  "description": "updated",
  "key": "resource-key"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[dataintegrationsdk.UserDefinedFunction, dataintegrationsdk.CreateUserDefinedFunctionDetails, dataintegrationsdk.UpdateUserDefinedFunctionDetails]{
		CollectionPath: "/20200430/workspaces/<ocid:1>/userDefinedFunctions", ItemPath: "/20200430/workspaces/<ocid:1>/userDefinedFunctions/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ dataintegrationsdk.CreateUserDefinedFunctionDetails) error {
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
	manager := &UserDefinedFunctionServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newUserDefinedFunctionRuntimeHooks(manager, sdkClient)
	client := wrapUserDefinedFunctionGeneratedClient(hooks, defaultUserDefinedFunctionServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dataintegrationv1beta1.UserDefinedFunction](buildUserDefinedFunctionGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dataintegrationv1beta1.UserDefinedFunction]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dataintegrationv1beta1.UserDefinedFunction) error {
			if current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created UserDefinedFunction status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dataintegrationv1beta1.UserDefinedFunction) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *dataintegrationv1beta1.UserDefinedFunction) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated UserDefinedFunction status = %+v", current.Status)
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
