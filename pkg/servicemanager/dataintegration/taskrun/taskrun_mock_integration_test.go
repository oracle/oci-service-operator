/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package taskrun

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

func TestMockIntegrationTaskRunCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &dataintegrationv1beta1.TaskRun{}
	ocimock.InitializeResource(resource, "mock-taskrun")
	resource.Spec = ocimock.MustJSONFixture[dataintegrationv1beta1.TaskRunSpec](t, `{
  "workspaceId": "<ocid:1>",
  "applicationKey": "application-key",
  "aggregatorKey": "aggregator-key",
  "name": "task run create",
  "description": "create"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"description":"updated"}`)
	createRequest := ocimock.MustJSONFixture[dataintegrationsdk.CreateTaskRunDetails](t, `{
  "name": "task run create",
  "description": "create"
}`)
	updateRequest := ocimock.MustJSONFixture[dataintegrationsdk.UpdateTaskRunDetails](t, `{
  "description": "updated"
}`)
	createdState := ocimock.MustOCIResponseFixture[dataintegrationsdk.TaskRun](t, `{
  "aggregatorKey": "aggregator-key",
  "name": "task run create",
  "description": "create",
  "key": "resource-key",
  "status": "SUCCESS"
}`)
	updatedState := ocimock.MustOCIResponseFixture[dataintegrationsdk.TaskRun](t, `{
  "aggregatorKey": "aggregator-key",
  "name": "task run create",
  "description": "updated",
  "key": "resource-key",
  "status": "SUCCESS"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[dataintegrationsdk.TaskRun, dataintegrationsdk.CreateTaskRunDetails, dataintegrationsdk.UpdateTaskRunDetails]{
		CollectionPath: "/20200430/workspaces/<ocid:1>/applications/application-key/taskRuns", ItemPath: "/20200430/workspaces/<ocid:1>/applications/application-key/taskRuns/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ dataintegrationsdk.CreateTaskRunDetails) error {
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
	manager := &TaskRunServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newTaskRunRuntimeHooks(manager, sdkClient)
	client := wrapTaskRunGeneratedClient(hooks, defaultTaskRunServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dataintegrationv1beta1.TaskRun](buildTaskRunGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dataintegrationv1beta1.TaskRun]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dataintegrationv1beta1.TaskRun) error {
			if current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created TaskRun status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dataintegrationv1beta1.TaskRun) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *dataintegrationv1beta1.TaskRun) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated TaskRun status = %+v", current.Status)
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
