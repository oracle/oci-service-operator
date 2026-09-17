/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package project

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

func TestMockIntegrationProjectCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &dataintegrationv1beta1.Project{}
	ocimock.InitializeResource(resource, "mock-project")
	resource.Spec = ocimock.MustJSONFixture[dataintegrationv1beta1.ProjectSpec](t, `{
  "workspaceId": "<ocid:1>",
  "name": "project create",
  "identifier": "PROJECT_CREATE",
  "modelType": "PROJECT",
  "objectVersion": 1,
  "description": "create"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"description":"updated"}`)
	createRequest := ocimock.MustJSONFixture[dataintegrationsdk.CreateProjectDetails](t, `{
  "name": "project create",
  "identifier": "PROJECT_CREATE",
  "description": "create"
}`)
	updateRequest := ocimock.MustJSONFixture[dataintegrationsdk.UpdateProjectDetails](t, `{
  "description": "updated",
  "modelType": "PROJECT",
  "objectVersion": 1,
  "key": "resource-key"
}`)
	createdState := ocimock.MustOCIResponseFixture[dataintegrationsdk.Project](t, `{
  "name": "project create",
  "identifier": "PROJECT_CREATE",
  "modelType": "PROJECT",
  "objectVersion": 1,
  "description": "create",
  "key": "resource-key"
}`)
	updatedState := ocimock.MustOCIResponseFixture[dataintegrationsdk.Project](t, `{
  "name": "project create",
  "identifier": "PROJECT_CREATE",
  "modelType": "PROJECT",
  "objectVersion": 1,
  "description": "updated",
  "key": "resource-key"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[dataintegrationsdk.Project, dataintegrationsdk.CreateProjectDetails, dataintegrationsdk.UpdateProjectDetails]{
		CollectionPath: "/20200430/workspaces/<ocid:1>/projects", ItemPath: "/20200430/workspaces/<ocid:1>/projects/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ dataintegrationsdk.CreateProjectDetails) error {
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
	manager := &ProjectServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newProjectRuntimeHooks(manager, sdkClient)
	client := wrapProjectGeneratedClient(hooks, defaultProjectServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dataintegrationv1beta1.Project](buildProjectGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dataintegrationv1beta1.Project]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dataintegrationv1beta1.Project) error {
			if current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created Project status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dataintegrationv1beta1.Project) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *dataintegrationv1beta1.Project) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated Project status = %+v", current.Status)
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
