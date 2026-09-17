/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package taskschedule

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

func TestMockIntegrationTaskScheduleCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := &dataintegrationv1beta1.TaskSchedule{}
	ocimock.InitializeResource(resource, "mock-taskschedule")
	resource.Spec = ocimock.MustJSONFixture[dataintegrationv1beta1.TaskScheduleSpec](t, `{
  "workspaceId": "<ocid:1>",
  "applicationKey": "application-key",
  "name": "task schedule create",
  "identifier": "TASK_SCHEDULE_CREATE",
  "description": "create",
  "objectVersion": 1
}`)
	updatedSpec := resource.Spec
	ocimock.MustMergeJSONFixture(t, &updatedSpec, `{"description":"updated"}`)
	createRequest := ocimock.MustJSONFixture[dataintegrationsdk.CreateTaskScheduleDetails](t, `{
  "name": "task schedule create",
  "identifier": "TASK_SCHEDULE_CREATE",
  "description": "create",
  "objectVersion": 1
}`)
	updateRequest := ocimock.MustJSONFixture[dataintegrationsdk.UpdateTaskScheduleDetails](t, `{
  "description": "updated",
  "key": "resource-key",
  "objectVersion": 1
}`)
	createdState := ocimock.MustOCIResponseFixture[dataintegrationsdk.TaskSchedule](t, `{
  "name": "task schedule create",
  "identifier": "TASK_SCHEDULE_CREATE",
  "description": "create",
  "key": "resource-key",
  "objectVersion": 1
}`)
	updatedState := ocimock.MustOCIResponseFixture[dataintegrationsdk.TaskSchedule](t, `{
  "name": "task schedule create",
  "identifier": "TASK_SCHEDULE_CREATE",
  "description": "updated",
  "key": "resource-key",
  "objectVersion": 1
}`)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[dataintegrationsdk.TaskSchedule, dataintegrationsdk.CreateTaskScheduleDetails, dataintegrationsdk.UpdateTaskScheduleDetails]{
		CollectionPath: "/20200430/workspaces/<ocid:1>/applications/application-key/taskSchedules", ItemPath: "/20200430/workspaces/<ocid:1>/applications/application-key/taskSchedules/resource-key",
		Operations:    []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreateRequest: &createRequest, CreatedState: &createdState, UpdateRequest: &updateRequest, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreate: func(request ocimock.Request, _ dataintegrationsdk.CreateTaskScheduleDetails) error {
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
	manager := &TaskScheduleServiceManager{Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}}
	hooks := newTaskScheduleRuntimeHooks(manager, sdkClient)
	client := wrapTaskScheduleGeneratedClient(hooks, defaultTaskScheduleServiceClient{ServiceClient: generatedruntime.NewServiceClient[*dataintegrationv1beta1.TaskSchedule](buildTaskScheduleGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*dataintegrationv1beta1.TaskSchedule]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *dataintegrationv1beta1.TaskSchedule) error {
			if current.Status.Description != resource.Spec.Description || current.Status.OsokStatus.Ocid == "" {
				return fmt.Errorf("created TaskSchedule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *dataintegrationv1beta1.TaskSchedule) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *dataintegrationv1beta1.TaskSchedule) error {
			if current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated TaskSchedule status = %+v", current.Status)
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
