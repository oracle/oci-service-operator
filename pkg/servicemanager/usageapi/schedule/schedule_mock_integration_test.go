/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package schedule

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	usageapisdk "github.com/oracle/oci-go-sdk/v65/usageapi"
	usageapiv1beta1 "github.com/oracle/oci-service-operator/api/usageapi/v1beta1"
	"github.com/oracle/oci-service-operator/internal/integration/ocimock"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const mockScheduleID = "ocid1.usageschedule.oc1..mock"

// Contract evidence: the package-owned typed OCI fixtures, formal schedule contract, and vendored OCI SDK.
func TestMockIntegrationScheduleLifecycleCRUD(t *testing.T) {
	t.Parallel()
	const tenancyID = "ocid1.tenancy.oc1..mock"
	resource := &usageapiv1beta1.Schedule{ObjectMeta: metav1.ObjectMeta{Name: "mock-schedule", Namespace: "default", UID: types.UID("mock-schedule-uid")}, Spec: usageapiv1beta1.ScheduleSpec{
		Name: "mock-schedule", CompartmentId: tenancyID, ScheduleRecurrences: "DAILY", TimeScheduled: "2030-01-15T10:00:00Z", Description: "mock create", OutputFileFormat: "CSV",
		ResultLocation:  usageapiv1beta1.ScheduleResultLocation{LocationType: "OBJECT_STORAGE", Region: "us-ashburn-1", Namespace: "mocknamespace", BucketName: "mock-bucket"},
		QueryProperties: usageapiv1beta1.ScheduleQueryProperties{Granularity: "DAILY", DateRange: usageapiv1beta1.ScheduleQueryPropertiesDateRange{DateRangeType: "DYNAMIC", DynamicDateRangeType: "LAST_7_DAYS"}},
		FreeformTags:    map[string]string{"osok-mock": "create"},
	}}
	responder, err := newScheduleMockResponder(resource)
	if err != nil {
		t.Fatal(err)
	}
	session, err := ocimock.Open(ocimock.Options{Host: "https://usage.mock.invalid", BasePath: "20200107", Responder: responder})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	manager := &ScheduleServiceManager{}
	hooks := newScheduleRuntimeHooks(manager, usageapisdk.UsageapiClient{BaseClient: session.BaseClient()})
	client := wrapScheduleGeneratedClient(hooks, defaultScheduleServiceClient{ServiceClient: generatedruntime.NewServiceClient[*usageapiv1beta1.Schedule](buildScheduleGeneratedRuntimeConfig(manager, hooks))})
	err = ocimock.RunLifecycle(context.Background(), ocimock.LifecycleScenario[*usageapiv1beta1.Schedule]{
		Resource: resource, Client: client, CreateContext: generatedruntime.WithSkipExistingBeforeCreate,
		ValidateCreated: func(current *usageapiv1beta1.Schedule) error {
			if current.Status.Id != mockScheduleID || current.Status.Name != resource.Spec.Name || current.Status.LifecycleState != string(usageapisdk.ScheduleLifecycleStateActive) {
				return fmt.Errorf("created Schedule status = %+v", current.Status)
			}
			return nil
		},
		Mutate: func(current *usageapiv1beta1.Schedule) {
			current.Spec.Description = "mock update"
			current.Spec.OutputFileFormat = "PDF"
			current.Spec.FreeformTags = map[string]string{"osok-mock": "update"}
		},
		ValidateUpdated: func(current *usageapiv1beta1.Schedule) error {
			if current.Status.Description != current.Spec.Description || current.Status.OutputFileFormat != "PDF" || current.Status.FreeformTags["osok-mock"] != "update" {
				return fmt.Errorf("updated Schedule status = %+v", current.Status)
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

func newScheduleMockResponder(resource *usageapiv1beta1.Schedule) (*ocimock.CRUDResponder[usageapisdk.Schedule], error) {
	now := common.SDKTime{Time: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	return ocimock.NewCRUDResponder(ocimock.CRUDOptions[usageapisdk.Schedule]{
		CollectionPath: "/20200107/schedules", ItemPath: "/20200107/schedules/" + mockScheduleID,
		ExpectedOperations: []ocimock.Operation{ocimock.OperationCreate, ocimock.OperationRead, ocimock.OperationUpdate, ocimock.OperationDelete},
		RequireCreateRead:  true, RequireUpdateRead: true, RequireDeleteRead: true,
		List: func(_ ocimock.Request, present bool, state usageapisdk.Schedule) (ocimock.Response, error) {
			if !present {
				return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []usageapisdk.Schedule{}})
			}
			return ocimock.JSONResponse(http.StatusOK, map[string]any{"items": []usageapisdk.Schedule{state}})
		},
		Create: func(request ocimock.Request) (usageapisdk.Schedule, ocimock.Response, error) {
			if err := ocimock.ValidateRetryToken(request, resource); err != nil {
				var zero usageapisdk.Schedule
				return zero, ocimock.Response{}, err
			}
			var details usageapisdk.CreateScheduleDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return usageapisdk.Schedule{}, ocimock.Response{}, err
			}
			if err := ocimock.ValidateMandatoryFields(details); err != nil {
				return usageapisdk.Schedule{}, ocimock.Response{}, err
			}
			if details.Name == nil || *details.Name != resource.Spec.Name || details.CompartmentId == nil || *details.CompartmentId != resource.Spec.CompartmentId || details.ScheduleRecurrences == nil || *details.ScheduleRecurrences != "DAILY" || details.ResultLocation == nil {
				return usageapisdk.Schedule{}, ocimock.Response{}, fmt.Errorf("unexpected CreateSchedule details: %+v", details)
			}
			state := usageapisdk.Schedule{Id: common.String(mockScheduleID), Name: details.Name, CompartmentId: details.CompartmentId, ResultLocation: details.ResultLocation,
				ScheduleRecurrences: details.ScheduleRecurrences, TimeScheduled: details.TimeScheduled, TimeCreated: &now, LifecycleState: usageapisdk.ScheduleLifecycleStateActive,
				Description: details.Description, OutputFileFormat: usageapisdk.ScheduleOutputFileFormatEnum(details.OutputFileFormat), QueryProperties: details.QueryProperties, FreeformTags: details.FreeformTags}
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Read: func(_ ocimock.Request, state usageapisdk.Schedule) (ocimock.Response, error) {
			return ocimock.JSONResponse(http.StatusOK, state)
		},
		Update: func(request ocimock.Request, state usageapisdk.Schedule) (usageapisdk.Schedule, ocimock.Response, error) {
			var details usageapisdk.UpdateScheduleDetails
			if err := ocimock.DecodeJSONRequest(request, &details); err != nil {
				return usageapisdk.Schedule{}, ocimock.Response{}, err
			}
			if details.Description == nil || *details.Description != "mock update" || details.OutputFileFormat != usageapisdk.UpdateScheduleDetailsOutputFileFormatPdf || details.FreeformTags["osok-mock"] != "update" {
				return usageapisdk.Schedule{}, ocimock.Response{}, fmt.Errorf("unexpected UpdateSchedule details: %+v", details)
			}
			state.Description, state.OutputFileFormat, state.FreeformTags = details.Description, usageapisdk.ScheduleOutputFileFormatPdf, details.FreeformTags
			response, err := ocimock.JSONResponse(http.StatusOK, state)
			return state, response, err
		},
		Delete: func(_ ocimock.Request, _ usageapisdk.Schedule) (ocimock.Response, error) {
			return ocimock.EmptyResponse(http.StatusNoContent), nil
		},
	})
}
