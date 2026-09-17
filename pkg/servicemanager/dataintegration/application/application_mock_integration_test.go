/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package application

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

func TestMockIntegrationApplicationCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &dataintegrationv1beta1.Application{}
	ocimock.InitializeResource(resource, "mock-application")
	resource.Spec = ocimock.MustJSONFixture[dataintegrationv1beta1.ApplicationSpec](t, `{
  "workspaceId": "<ocid:1>",
  "name": "application create",
  "identifier": "APPLICATION_CREATE",
  "objectVersion": 1,
  "description": "create",
  "modelType": "INTEGRATION_APPLICATION"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"description":"updated"}`)
	createRequest := ocimock.MustJSONFixture[dataintegrationsdk.CreateApplicationDetails](t, `{
  "name": "application create",
  "identifier": "APPLICATION_CREATE",
  "description": "create",
  "modelType": "INTEGRATION_APPLICATION"
}`)
	updateRequest := ocimock.MustJSONFixture[dataintegrationsdk.UpdateApplicationDetails](t, `{
  "description": "updated",
  "objectVersion": 1,
  "key": "resource-key",
  "modelType": "INTEGRATION_APPLICATION"
}`)
	createdState := ocimock.MustOCIResponseFixture[dataintegrationsdk.Application](t, `{
  "name": "application create",
  "identifier": "APPLICATION_CREATE",
  "objectVersion": 1,
  "description": "create",
  "key": "resource-key",
  "lifecycleState": "ACTIVE",
  "modelType": "INTEGRATION_APPLICATION"
}`)
	updatedState := ocimock.MustOCIResponseFixture[dataintegrationsdk.Application](t, `{
  "name": "application create",
  "identifier": "APPLICATION_CREATE",
  "objectVersion": 1,
  "description": "updated",
  "key": "resource-key",
  "lifecycleState": "ACTIVE",
  "modelType": "INTEGRATION_APPLICATION"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[dataintegrationsdk.Application, dataintegrationsdk.CreateApplicationDetails, dataintegrationsdk.UpdateApplicationDetails]{
		CollectionPath: "/20200430/workspaces/<ocid:1>/applications", ItemPath: "/20200430/workspaces/<ocid:1>/applications/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ dataintegrationsdk.CreateApplicationDetails) error {
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
	manager := &ApplicationServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newApplicationRuntimeHooks(manager, sdkClient)
	client := wrapApplicationGeneratedClient(hooks, defaultApplicationServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dataintegrationv1beta1.Application](buildApplicationGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dataintegrationv1beta1.Application]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dataintegrationv1beta1.Application) error {
			if current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created Application status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dataintegrationv1beta1.Application) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *dataintegrationv1beta1.Application) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated Application status = %+v", current.Status)
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
