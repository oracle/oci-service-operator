/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package scheduledjob

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	osmanagementhubsdk "github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const mockScheduledJobID = "ocid1.osmhscheduledjob.oc1..mock"

// Contract evidence: recorded scheduled-job CRUD, resource-local semantics, OCI SDK, and pinned provider.
func TestMockIntegrationScheduledJobLifecycleCRUD(t *testing.T) {
	t.Parallel()
	resource := &osmanagementhubv1beta1.ScheduledJob{ObjectMeta: metav1.ObjectMeta{Name: "mock-scheduled-job", Namespace: "default", UID: types.UID("mock-scheduled-job-uid")}, Spec: osmanagementhubv1beta1.ScheduledJobSpec{
		CompartmentId: "ocid1.compartment.oc1..mock", DisplayName: "mock-scheduled-job", Description: "mock create", ScheduleType: string(osmanagementhubsdk.ScheduleTypesOnetime), TimeNextExecution: "2030-01-15T10:00:00Z",
		Operations: []osmanagementhubv1beta1.ScheduledJobOperation{{OperationType: string(osmanagementhubsdk.OperationTypesUpdateAll)}}, ManagedCompartmentIds: []string{"ocid1.compartment.oc1..mock"}, FreeformTags: map[string]string{"mock": "create"},
	}}
	responder, err := newScheduledJobMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://osmh.mock.invalid", BasePath: "20220901", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("close ScheduledJob OCI mock: %v", err)
		}
	})
	client := newScheduledJobServiceClientWithOCIClient(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("mock-integration")}, osmanagementhubsdk.ScheduledJobClient{BaseClient: session.BaseClient()})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*osmanagementhubv1beta1.ScheduledJob]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *osmanagementhubv1beta1.ScheduledJob) error {
			if current.Status.Id != mockScheduledJobID || current.Status.DisplayName != resource.Spec.DisplayName {
				return fmt.Errorf("created ScheduledJob status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *osmanagementhubv1beta1.ScheduledJob) {
			current.Spec.DisplayName = "mock-scheduled-job-updated"
			current.Spec.Description = "mock update"
			current.Spec.FreeformTags = map[string]string{"mock": "update"}
		},
		ValidateUpdated: func(current *osmanagementhubv1beta1.ScheduledJob) error {
			if current.Status.DisplayName != current.Spec.DisplayName || current.Status.Description != current.Spec.Description {
				return fmt.Errorf("updated ScheduledJob status = %+v", current.Status)
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

func newScheduledJobMockResponder(resource *osmanagementhubv1beta1.ScheduledJob) (*ocimock.CRUDResponder[osmanagementhubsdk.ScheduledJob], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[osmanagementhubsdk.ScheduledJob]{
		CollectionPath: "/20220901/scheduledJobs", ItemPath: "/20220901/scheduledJobs/" + mockScheduledJobID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete}, RequireCreateRead: true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(_ ocimock.Request, present bool, state osmanagementhubsdk.ScheduledJob) (ocimock.Response, error) {
			if !present {
				return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []osmanagementhubsdk.ScheduledJob{}})
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []osmanagementhubsdk.ScheduledJob{state}})
		},
		Create: func(request ocimock.Request) (osmanagementhubsdk.ScheduledJob, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero osmanagementhubsdk.ScheduledJob
				return zero, ocimock.Response{}, err
			}
			var details osmanagementhubsdk.CreateScheduledJobDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return osmanagementhubsdk.ScheduledJob{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return osmanagementhubsdk.ScheduledJob{}, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != resource.Spec.DisplayName || len(details.Operations) != 1 || len(details.ManagedCompartmentIds) != 1 {
				return osmanagementhubsdk.ScheduledJob{}, ocimock.Response{}, fmt.Errorf("unexpected CreateScheduledJob details: %+v", details)
			}
			state := osmanagementhubsdk.ScheduledJob{Id: common.String(mockScheduledJobID), DisplayName: details.DisplayName, CompartmentId: details.CompartmentId, ScheduleType: details.ScheduleType, TimeNextExecution: details.TimeNextExecution, Operations: details.Operations,
				TimeCreated: &now, TimeUpdated: &now, LifecycleState: osmanagementhubsdk.ScheduledJobLifecycleStateActive, FreeformTags: details.FreeformTags, DefinedTags: details.DefinedTags, Description: details.Description,
				ManagedCompartmentIds: details.ManagedCompartmentIds, IsSubcompartmentIncluded: common.Bool(false), IsManagedByAutonomousLinux: common.Bool(false), IsRestricted: common.Bool(false)}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state osmanagementhubsdk.ScheduledJob) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state osmanagementhubsdk.ScheduledJob) (osmanagementhubsdk.ScheduledJob, ocimock.Response, error) {
			var details osmanagementhubsdk.UpdateScheduledJobDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return state, ocimock.Response{}, err
			}
			if details.DisplayName == nil || *details.DisplayName != "mock-scheduled-job-updated" || details.Description == nil || *details.Description != "mock update" {
				return state, ocimock.Response{}, fmt.Errorf("unexpected UpdateScheduledJob details: %+v", details)
			}
			state.DisplayName, state.Description, state.FreeformTags, state.TimeUpdated = details.DisplayName, details.Description, details.FreeformTags, &now
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ osmanagementhubsdk.ScheduledJob) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
