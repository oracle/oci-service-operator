/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package schedule

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	usageapisdk "github.com/oracle/oci-go-sdk/v65/usageapi"
	usageapiv1beta1 "github.com/oracle/oci-service-operator/api/usageapi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeScheduleOCIClient struct {
	createScheduleFn func(context.Context, usageapisdk.CreateScheduleRequest) (usageapisdk.CreateScheduleResponse, error)
	getScheduleFn    func(context.Context, usageapisdk.GetScheduleRequest) (usageapisdk.GetScheduleResponse, error)
	listSchedulesFn  func(context.Context, usageapisdk.ListSchedulesRequest) (usageapisdk.ListSchedulesResponse, error)
	updateScheduleFn func(context.Context, usageapisdk.UpdateScheduleRequest) (usageapisdk.UpdateScheduleResponse, error)
	deleteScheduleFn func(context.Context, usageapisdk.DeleteScheduleRequest) (usageapisdk.DeleteScheduleResponse, error)
}

func (f *fakeScheduleOCIClient) CreateSchedule(ctx context.Context, req usageapisdk.CreateScheduleRequest) (usageapisdk.CreateScheduleResponse, error) {
	if f.createScheduleFn != nil {
		return f.createScheduleFn(ctx, req)
	}
	return usageapisdk.CreateScheduleResponse{}, nil
}

func (f *fakeScheduleOCIClient) GetSchedule(ctx context.Context, req usageapisdk.GetScheduleRequest) (usageapisdk.GetScheduleResponse, error) {
	if f.getScheduleFn != nil {
		return f.getScheduleFn(ctx, req)
	}
	return usageapisdk.GetScheduleResponse{}, nil
}

func (f *fakeScheduleOCIClient) ListSchedules(ctx context.Context, req usageapisdk.ListSchedulesRequest) (usageapisdk.ListSchedulesResponse, error) {
	if f.listSchedulesFn != nil {
		return f.listSchedulesFn(ctx, req)
	}
	return usageapisdk.ListSchedulesResponse{}, nil
}

func (f *fakeScheduleOCIClient) UpdateSchedule(ctx context.Context, req usageapisdk.UpdateScheduleRequest) (usageapisdk.UpdateScheduleResponse, error) {
	if f.updateScheduleFn != nil {
		return f.updateScheduleFn(ctx, req)
	}
	return usageapisdk.UpdateScheduleResponse{}, nil
}

func (f *fakeScheduleOCIClient) DeleteSchedule(ctx context.Context, req usageapisdk.DeleteScheduleRequest) (usageapisdk.DeleteScheduleResponse, error) {
	if f.deleteScheduleFn != nil {
		return f.deleteScheduleFn(ctx, req)
	}
	return usageapisdk.DeleteScheduleResponse{}, nil
}

func testScheduleClient(fake *fakeScheduleOCIClient) ScheduleServiceClient {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := &ScheduleServiceManager{Log: log}
	hooks := newScheduleDefaultRuntimeHooks(usageapisdk.UsageapiClient{})
	hooks.Create.Call = func(ctx context.Context, request usageapisdk.CreateScheduleRequest) (usageapisdk.CreateScheduleResponse, error) {
		return fake.CreateSchedule(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request usageapisdk.GetScheduleRequest) (usageapisdk.GetScheduleResponse, error) {
		return fake.GetSchedule(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request usageapisdk.ListSchedulesRequest) (usageapisdk.ListSchedulesResponse, error) {
		return fake.ListSchedules(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request usageapisdk.UpdateScheduleRequest) (usageapisdk.UpdateScheduleResponse, error) {
		return fake.UpdateSchedule(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request usageapisdk.DeleteScheduleRequest) (usageapisdk.DeleteScheduleResponse, error) {
		return fake.DeleteSchedule(ctx, request)
	}
	applyScheduleRuntimeHooks(&hooks)
	return defaultScheduleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*usageapiv1beta1.Schedule](
			buildScheduleGeneratedRuntimeConfig(manager, hooks),
		),
	}
}

func makeScheduleResource() *usageapiv1beta1.Schedule {
	return &usageapiv1beta1.Schedule{
		Spec: usageapiv1beta1.ScheduleSpec{
			Name:                "schedule-sample",
			CompartmentId:       "ocid1.compartment.oc1..scheduleexample",
			ScheduleRecurrences: "FREQ=DAILY;INTERVAL=1",
			TimeScheduled:       "2026-01-15T15:00:00Z",
			Description:         "Daily usage export",
			OutputFileFormat:    "CSV",
			SavedReportId:       "ocid1.savedreport.oc1..scheduleexample",
			ResultLocation: usageapiv1beta1.ScheduleResultLocation{
				LocationType: "OBJECT_STORAGE",
				Region:       "us-ashburn-1",
				Namespace:    "example-namespace",
				BucketName:   "usage-reports",
			},
			FreeformTags: map[string]string{
				"workload": "finops",
			},
		},
	}
}

func makeSDKScheduleResultLocation(bucketName string) usageapisdk.ObjectStorageLocation {
	return usageapisdk.ObjectStorageLocation{
		Region:     common.String("us-ashburn-1"),
		Namespace:  common.String("example-namespace"),
		BucketName: common.String(bucketName),
	}
}

func makeSDKSchedule(
	id string,
	name string,
	description string,
	outputFormat usageapisdk.ScheduleOutputFileFormatEnum,
	lifecycle usageapisdk.ScheduleLifecycleStateEnum,
	bucketName string,
) usageapisdk.Schedule {
	return usageapisdk.Schedule{
		Id:                  common.String(id),
		Name:                common.String(name),
		CompartmentId:       common.String("ocid1.compartment.oc1..scheduleexample"),
		ResultLocation:      makeSDKScheduleResultLocation(bucketName),
		ScheduleRecurrences: common.String("FREQ=DAILY;INTERVAL=1"),
		TimeScheduled:       &common.SDKTime{Time: time.Date(2026, 1, 15, 15, 0, 0, 0, time.UTC)},
		TimeCreated:         &common.SDKTime{Time: time.Date(2026, 1, 15, 14, 50, 0, 0, time.UTC)},
		LifecycleState:      lifecycle,
		Description:         common.String(description),
		OutputFileFormat:    outputFormat,
		SavedReportId:       common.String("ocid1.savedreport.oc1..scheduleexample"),
		FreeformTags: map[string]string{
			"workload": "finops",
		},
	}
}

func makeSDKScheduleSummary(id string, name string) usageapisdk.ScheduleSummary {
	return usageapisdk.ScheduleSummary{
		Id:                  common.String(id),
		Name:                common.String(name),
		ScheduleRecurrences: common.String("FREQ=DAILY;INTERVAL=1"),
		TimeScheduled:       &common.SDKTime{Time: time.Date(2026, 1, 15, 15, 0, 0, 0, time.UTC)},
		LifecycleState:      usageapisdk.ScheduleLifecycleStateActive,
		Description:         common.String("Daily usage export"),
	}
}

func TestScheduleServiceClientCreatesAndProjectsStatus(t *testing.T) {
	t.Parallel()

	var createRequest usageapisdk.CreateScheduleRequest
	getCalls := 0
	client := testScheduleClient(&fakeScheduleOCIClient{
		listSchedulesFn: func(_ context.Context, req usageapisdk.ListSchedulesRequest) (usageapisdk.ListSchedulesResponse, error) {
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..scheduleexample" {
				t.Fatalf("list compartmentId = %v, want schedule compartment", req.CompartmentId)
			}
			if req.Name == nil || *req.Name != "schedule-sample" {
				t.Fatalf("list name = %v, want schedule name", req.Name)
			}
			return usageapisdk.ListSchedulesResponse{}, nil
		},
		createScheduleFn: func(_ context.Context, req usageapisdk.CreateScheduleRequest) (usageapisdk.CreateScheduleResponse, error) {
			createRequest = req
			return usageapisdk.CreateScheduleResponse{
				Schedule:     makeSDKSchedule("ocid1.schedule.oc1..created", "schedule-sample", "Daily usage export", usageapisdk.ScheduleOutputFileFormatCsv, usageapisdk.ScheduleLifecycleStateActive, "usage-reports"),
				OpcRequestId: common.String("opc-create-1"),
			}, nil
		},
		getScheduleFn: func(_ context.Context, req usageapisdk.GetScheduleRequest) (usageapisdk.GetScheduleResponse, error) {
			getCalls++
			if req.ScheduleId == nil || *req.ScheduleId != "ocid1.schedule.oc1..created" {
				t.Fatalf("get scheduleId = %v, want created schedule OCID", req.ScheduleId)
			}
			return usageapisdk.GetScheduleResponse{
				Schedule: makeSDKSchedule("ocid1.schedule.oc1..created", "schedule-sample", "Daily usage export", usageapisdk.ScheduleOutputFileFormatCsv, usageapisdk.ScheduleLifecycleStateActive, "usage-reports"),
			}, nil
		},
	})

	resource := makeScheduleResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after read-after-write GetSchedule confirmation")
	}
	if createRequest.CreateScheduleDetails.Name == nil || *createRequest.CreateScheduleDetails.Name != resource.Spec.Name {
		t.Fatalf("create name = %v, want %q", createRequest.CreateScheduleDetails.Name, resource.Spec.Name)
	}
	if createRequest.CreateScheduleDetails.CompartmentId == nil || *createRequest.CreateScheduleDetails.CompartmentId != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %v, want %q", createRequest.CreateScheduleDetails.CompartmentId, resource.Spec.CompartmentId)
	}
	location, ok := createRequest.CreateScheduleDetails.ResultLocation.(usageapisdk.ObjectStorageLocation)
	if !ok || location.BucketName == nil || *location.BucketName != "usage-reports" {
		t.Fatalf("create resultLocation = %#v, want object storage bucket %q", createRequest.CreateScheduleDetails.ResultLocation, "usage-reports")
	}
	if createRequest.CreateScheduleDetails.SavedReportId == nil || *createRequest.CreateScheduleDetails.SavedReportId != resource.Spec.SavedReportId {
		t.Fatalf("create savedReportId = %v, want %q", createRequest.CreateScheduleDetails.SavedReportId, resource.Spec.SavedReportId)
	}
	if getCalls != 1 {
		t.Fatalf("GetSchedule() calls = %d, want 1 follow-up read", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.schedule.oc1..created" {
		t.Fatalf("status.ocid = %q, want created schedule OCID", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-create-1")
	}
	if got := resource.Status.Id; got != "ocid1.schedule.oc1..created" {
		t.Fatalf("status.id = %q, want created schedule OCID", got)
	}
	if got := resource.Status.Name; got != resource.Spec.Name {
		t.Fatalf("status.name = %q, want %q", got, resource.Spec.Name)
	}
	if got := resource.Status.CompartmentId; got != resource.Spec.CompartmentId {
		t.Fatalf("status.compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
	}
	if got := resource.Status.ResultLocation.BucketName; got != "usage-reports" {
		t.Fatalf("status.resultLocation.bucketName = %q, want %q", got, "usage-reports")
	}
	if got := resource.Status.LifecycleState; got != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want %q", got, "ACTIVE")
	}
}

func TestScheduleServiceClientBindsExistingScheduleWithoutCreate(t *testing.T) {
	t.Parallel()

	createCalled := false
	getCalls := 0
	client := testScheduleClient(&fakeScheduleOCIClient{
		listSchedulesFn: func(_ context.Context, req usageapisdk.ListSchedulesRequest) (usageapisdk.ListSchedulesResponse, error) {
			if req.CompartmentId == nil || *req.CompartmentId != "ocid1.compartment.oc1..scheduleexample" {
				t.Fatalf("list compartmentId = %v, want schedule compartment", req.CompartmentId)
			}
			if req.Name == nil || *req.Name != "schedule-sample" {
				t.Fatalf("list name = %v, want schedule name", req.Name)
			}
			return usageapisdk.ListSchedulesResponse{
				ScheduleCollection: usageapisdk.ScheduleCollection{
					Items: []usageapisdk.ScheduleSummary{
						makeSDKScheduleSummary("ocid1.schedule.oc1..mismatch", "another-schedule"),
						makeSDKScheduleSummary("ocid1.schedule.oc1..existing", "schedule-sample"),
					},
				},
			}, nil
		},
		getScheduleFn: func(_ context.Context, req usageapisdk.GetScheduleRequest) (usageapisdk.GetScheduleResponse, error) {
			getCalls++
			if req.ScheduleId == nil || *req.ScheduleId != "ocid1.schedule.oc1..existing" {
				t.Fatalf("get scheduleId = %v, want existing schedule OCID", req.ScheduleId)
			}
			return usageapisdk.GetScheduleResponse{
				Schedule: makeSDKSchedule("ocid1.schedule.oc1..existing", "schedule-sample", "Daily usage export", usageapisdk.ScheduleOutputFileFormatCsv, usageapisdk.ScheduleLifecycleStateActive, "usage-reports"),
			}, nil
		},
		createScheduleFn: func(_ context.Context, _ usageapisdk.CreateScheduleRequest) (usageapisdk.CreateScheduleResponse, error) {
			createCalled = true
			return usageapisdk.CreateScheduleResponse{}, nil
		},
	})

	resource := makeScheduleResource()
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success when list lookup reuses an existing schedule")
	}
	if createCalled {
		t.Fatal("CreateSchedule() should not be called when ListSchedules finds a matching schedule")
	}
	if getCalls != 1 {
		t.Fatalf("GetSchedule() calls = %d, want 1 read of the bound schedule", getCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != "ocid1.schedule.oc1..existing" {
		t.Fatalf("status.ocid = %q, want existing schedule OCID", got)
	}
}

func TestScheduleServiceClientUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	var updateRequest usageapisdk.UpdateScheduleRequest
	getCalls := 0
	updateCalls := 0
	client := testScheduleClient(&fakeScheduleOCIClient{
		getScheduleFn: func(_ context.Context, req usageapisdk.GetScheduleRequest) (usageapisdk.GetScheduleResponse, error) {
			getCalls++
			if req.ScheduleId == nil || *req.ScheduleId != "ocid1.schedule.oc1..existing" {
				t.Fatalf("get scheduleId = %v, want existing schedule OCID", req.ScheduleId)
			}
			description := "Daily usage export"
			outputFormat := usageapisdk.ScheduleOutputFileFormatCsv
			lifecycle := usageapisdk.ScheduleLifecycleStateActive
			bucket := "usage-reports"
			if getCalls > 1 {
				description = "Updated usage export"
				outputFormat = usageapisdk.ScheduleOutputFileFormatPdf
				lifecycle = usageapisdk.ScheduleLifecycleStateInactive
				bucket = "usage-reports-updated"
			}
			return usageapisdk.GetScheduleResponse{
				Schedule: makeSDKSchedule("ocid1.schedule.oc1..existing", "schedule-sample", description, outputFormat, lifecycle, bucket),
			}, nil
		},
		updateScheduleFn: func(_ context.Context, req usageapisdk.UpdateScheduleRequest) (usageapisdk.UpdateScheduleResponse, error) {
			updateCalls++
			updateRequest = req
			return usageapisdk.UpdateScheduleResponse{
				Schedule:     makeSDKSchedule("ocid1.schedule.oc1..existing", "schedule-sample", "Updated usage export", usageapisdk.ScheduleOutputFileFormatPdf, usageapisdk.ScheduleLifecycleStateInactive, "usage-reports-updated"),
				OpcRequestId: common.String("opc-update-1"),
			}, nil
		},
	})

	resource := makeScheduleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.schedule.oc1..existing")
	resource.Spec.Description = "Updated usage export"
	resource.Spec.OutputFileFormat = "PDF"
	resource.Spec.ResultLocation.BucketName = "usage-reports-updated"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success after updating mutable schedule fields")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after update follow-up GetSchedule confirmation")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateSchedule() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetSchedule() calls = %d, want 2 (observe + follow-up)", getCalls)
	}
	if updateRequest.ScheduleId == nil || *updateRequest.ScheduleId != "ocid1.schedule.oc1..existing" {
		t.Fatalf("update scheduleId = %v, want existing schedule OCID", updateRequest.ScheduleId)
	}
	if updateRequest.UpdateScheduleDetails.Description == nil || *updateRequest.UpdateScheduleDetails.Description != "Updated usage export" {
		t.Fatalf("update description = %v, want %q", updateRequest.UpdateScheduleDetails.Description, "Updated usage export")
	}
	if got := updateRequest.UpdateScheduleDetails.OutputFileFormat; got != usageapisdk.UpdateScheduleDetailsOutputFileFormatPdf {
		t.Fatalf("update outputFileFormat = %q, want %q", got, usageapisdk.UpdateScheduleDetailsOutputFileFormatPdf)
	}
	location, ok := updateRequest.UpdateScheduleDetails.ResultLocation.(usageapisdk.ObjectStorageLocation)
	if !ok || location.BucketName == nil || *location.BucketName != "usage-reports-updated" {
		t.Fatalf("update resultLocation = %#v, want updated object storage bucket", updateRequest.UpdateScheduleDetails.ResultLocation)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-update-1")
	}
	if got := resource.Status.Description; got != "Updated usage export" {
		t.Fatalf("status.description = %q, want %q", got, "Updated usage export")
	}
	if got := resource.Status.OutputFileFormat; got != "PDF" {
		t.Fatalf("status.outputFileFormat = %q, want %q", got, "PDF")
	}
	if got := resource.Status.ResultLocation.BucketName; got != "usage-reports-updated" {
		t.Fatalf("status.resultLocation.bucketName = %q, want %q", got, "usage-reports-updated")
	}
	if got := resource.Status.LifecycleState; got != "INACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want %q", got, "INACTIVE")
	}
}

func TestScheduleServiceClientRejectsReplacementOnlyNameDrift(t *testing.T) {
	t.Parallel()

	updateCalled := false
	client := testScheduleClient(&fakeScheduleOCIClient{
		getScheduleFn: func(_ context.Context, _ usageapisdk.GetScheduleRequest) (usageapisdk.GetScheduleResponse, error) {
			return usageapisdk.GetScheduleResponse{
				Schedule: makeSDKSchedule("ocid1.schedule.oc1..existing", "schedule-sample", "Daily usage export", usageapisdk.ScheduleOutputFileFormatCsv, usageapisdk.ScheduleLifecycleStateActive, "usage-reports"),
			}, nil
		},
		updateScheduleFn: func(_ context.Context, _ usageapisdk.UpdateScheduleRequest) (usageapisdk.UpdateScheduleResponse, error) {
			updateCalled = true
			return usageapisdk.UpdateScheduleResponse{}, nil
		},
	})

	resource := makeScheduleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.schedule.oc1..existing")
	resource.Spec.Name = "schedule-renamed"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "require replacement when name changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want replacement-required drift failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	if updateCalled {
		t.Fatal("UpdateSchedule() should not be called when replacement-only drift is detected")
	}
}

func TestScheduleServiceClientDeleteConfirmsNotFound(t *testing.T) {
	t.Parallel()

	var deleteRequest usageapisdk.DeleteScheduleRequest
	getCalls := 0
	client := testScheduleClient(&fakeScheduleOCIClient{
		getScheduleFn: func(_ context.Context, req usageapisdk.GetScheduleRequest) (usageapisdk.GetScheduleResponse, error) {
			getCalls++
			if req.ScheduleId == nil || *req.ScheduleId != "ocid1.schedule.oc1..existing" {
				t.Fatalf("get scheduleId = %v, want existing schedule OCID", req.ScheduleId)
			}
			if getCalls > 1 {
				return usageapisdk.GetScheduleResponse{}, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "schedule not found")
			}
			return usageapisdk.GetScheduleResponse{
				Schedule: makeSDKSchedule("ocid1.schedule.oc1..existing", "schedule-sample", "Daily usage export", usageapisdk.ScheduleOutputFileFormatCsv, usageapisdk.ScheduleLifecycleStateActive, "usage-reports"),
			}, nil
		},
		deleteScheduleFn: func(_ context.Context, req usageapisdk.DeleteScheduleRequest) (usageapisdk.DeleteScheduleResponse, error) {
			deleteRequest = req
			return usageapisdk.DeleteScheduleResponse{
				OpcRequestId: common.String("opc-delete-1"),
			}, nil
		},
	})

	resource := makeScheduleResource()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.schedule.oc1..existing")

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should report success once follow-up GetSchedule confirms not found")
	}
	if getCalls != 2 {
		t.Fatalf("GetSchedule() calls = %d, want 2 (preflight + confirmation)", getCalls)
	}
	if deleteRequest.ScheduleId == nil || *deleteRequest.ScheduleId != "ocid1.schedule.oc1..existing" {
		t.Fatalf("delete scheduleId = %v, want existing schedule OCID", deleteRequest.ScheduleId)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want %q", got, "opc-request-id")
	}
}
