/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package task

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

func TestMockIntegrationTaskCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &dataintegrationv1beta1.Task{}
	ocimock.InitializeResource(resource, "mock-task")
	resource.Spec = ocimock.MustJSONFixture[dataintegrationv1beta1.TaskSpec](t, `{
  "workspaceId": "<ocid:1>",
  "name": "task create",
  "identifier": "TASK_CREATE",
  "registryMetadata": {
  },
  "modelType": "REST_TASK",
  "objectVersion": 1,
  "description": "create"
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"description":"updated"}`)
	createRequest := ocimock.MustJSONFixture[dataintegrationsdk.CreateTaskFromRestTask](t, `{
  "name": "task create",
  "identifier": "TASK_CREATE",
  "registryMetadata": {},
  "description": "create"
}`)
	updateRequest := ocimock.MustJSONFixture[dataintegrationsdk.UpdateTaskFromRestTask](t, `{
  "description": "updated",
  "key": "resource-key",
  "objectVersion": 1
}`)
	createdState := ocimock.MustOCIResponseFixture[dataintegrationsdk.TaskFromRestTaskDetails](t, `{
  "name": "task create",
  "identifier": "TASK_CREATE",
  "registryMetadata": {
  },
  "modelType": "REST_TASK",
  "objectVersion": 1,
  "description": "create",
  "key": "resource-key"
}`)
	updatedState := ocimock.MustOCIResponseFixture[dataintegrationsdk.TaskFromRestTaskDetails](t, `{
  "name": "task create",
  "identifier": "TASK_CREATE",
  "registryMetadata": {
  },
  "modelType": "REST_TASK",
  "objectVersion": 1,
  "description": "updated",
  "key": "resource-key"
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[dataintegrationsdk.TaskFromRestTaskDetails, dataintegrationsdk.CreateTaskFromRestTask, dataintegrationsdk.UpdateTaskFromRestTask]{
		CollectionPath: "/20200430/workspaces/<ocid:1>/tasks", ItemPath: "/20200430/workspaces/<ocid:1>/tasks/resource-key",
		Operations:   []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState: &createdState, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "modelType", "REST_TASK", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "modelType", "REST_TASK", updateRequest)
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
	manager := &TaskServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newTaskRuntimeHooks(manager, sdkClient)
	client := wrapTaskGeneratedClient(hooks, defaultTaskServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dataintegrationv1beta1.Task](buildTaskGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dataintegrationv1beta1.Task]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dataintegrationv1beta1.Task) error {
			if current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created Task status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dataintegrationv1beta1.Task) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *dataintegrationv1beta1.Task) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated Task status = %+v", current.Status)
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
