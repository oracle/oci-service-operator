/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package scheduledtask

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func TestMockIntegrationScheduledTaskCompositeCRUD(t *testing.T) {
	t.Parallel()
	resource := scheduledTaskFixture()
	ocimock.InitializeResource(resource, "mock-scheduled-task")
	updatedSpec := resource.Spec
	updatedSpec.DisplayName = "daily search updated"
	createRequest := ocimock.MustJSONFixture[loganalyticssdk.CreateStandardTaskDetails](t, `{
  "compartmentId":"ocid1.compartment.oc1..scheduledtask","displayName":"daily search","taskType":"SAVED_SEARCH",
  "action":{"type":"STREAM","savedSearchId":"ocid1.managementsavedsearch.oc1..scheduledtask","savedSearchDuration":"PT5M"},
  "schedules":[{"type":"FIXED_FREQUENCY","recurringInterval":"PT5M","repeatCount":0}]
}`)
	createdState := standardTask(testScheduledTaskID, resource.Spec.DisplayName, resource.Spec.CompartmentId, loganalyticssdk.ScheduledTaskLifecycleStateActive)
	updateRequest := ocimock.MustJSONFixture[loganalyticssdk.UpdateStandardTaskDetails](t, `{
  "displayName":"daily search updated",
  "action":{"type":"STREAM","savedSearchId":"ocid1.managementsavedsearch.oc1..scheduledtask","savedSearchDuration":"PT5M"},
  "schedules":[{"type":"FIXED_FREQUENCY","recurringInterval":"PT5M","repeatCount":0}]
}`)
	updatedState := standardTask(testScheduledTaskID, updatedSpec.DisplayName, updatedSpec.CompartmentId, loganalyticssdk.ScheduledTaskLifecycleStateActive)
	responder, err := ocimock.NewExplicitCRUDResponder(ocimock.ExplicitCRUDOptions[loganalyticssdk.StandardTask, loganalyticssdk.CreateStandardTaskDetails, loganalyticssdk.UpdateStandardTaskDetails]{
		CollectionPath: "/20200601/namespaces/mocknamespace/scheduledTasks", ItemPath: "/20200601/namespaces/mocknamespace/scheduledTasks/" + testScheduledTaskID,
		Operations:   []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		CreatedState: &createdState, UpdatedState: &updatedState, ListShape: ocimock.ListShapeItems,
		RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true, DeleteEndsNotFound: true,
		CreateStatus: 201, UpdateStatus: 200, DeleteStatus: 204, NotFoundCode: "NotFound",
		ValidateCreateRaw: func(request ocimock.Request) error {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				return err
			}
			return ocimock.ValidateDiscriminatedJSONRequest(request, "kind", "STANDARD", createRequest)
		},
		ValidateUpdateRaw: func(request ocimock.Request) error {
			return ocimock.ValidateDiscriminatedJSONRequest(request, "kind", "STANDARD", updateRequest)
		},
		ValidateDelete: func(request ocimock.Request, _ loganalyticssdk.StandardTask) error {
			if request.Method != http.MethodDelete {
				return fmt.Errorf("delete method = %s", request.Method)
			}
			return nil
		},
		AdditionalRoutes: []ocimock.Route{{
			Name: "namespace-lookup", Method: http.MethodGet, Path: "/20200601/namespaces", MinimumCalls: 1,
			Respond: func(ocimock.Request) (ocimock.Response, error) {
				return ocimock.JSONResponse(http.StatusOK, namespaceResponse("mocknamespace", testCompartmentID).NamespaceCollection)
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://loganalytics.mock.invalid", BasePath: "20200601", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	sdkClient := loganalyticssdk.LogAnalyticsClient{BaseClient: session.BaseClient()}
	manager := &ScheduledTaskServiceManager{Log: loggerutil.OSOKLogger{}}
	hooks := newScheduledTaskDefaultRuntimeHooks(sdkClient)
	applyScheduledTaskRuntimeHooks(manager, &hooks, sdkClient, nil)
	client := wrapScheduledTaskGeneratedClient(hooks, defaultScheduledTaskServiceClient{ServiceClient: generatedruntime.NewServiceClient[*loganalyticsv1beta1.ScheduledTask](buildScheduledTaskGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*loganalyticsv1beta1.ScheduledTask]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *loganalyticsv1beta1.ScheduledTask) error {
			if current.Status.Id != testScheduledTaskID || current.Status.DisplayName != current.Spec.DisplayName || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("created ScheduledTask status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *loganalyticsv1beta1.ScheduledTask) { current.Spec = updatedSpec },
		ValidateUpdated: func(current *loganalyticsv1beta1.ScheduledTask) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.LifecycleState != "ACTIVE" {
				return fmt.Errorf("updated ScheduledTask status = %+v", current.Status)
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
