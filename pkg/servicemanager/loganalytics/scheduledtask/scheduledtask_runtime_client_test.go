/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package scheduledtask

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testCompartmentID   = "ocid1.compartment.oc1..scheduledtask"
	testScheduledTaskID = "ocid1.scheduledtask.oc1..scheduledtask"
	testNamespaceName   = "la-namespace"
	testSavedSearchID   = "ocid1.managementsavedsearch.oc1..scheduledtask"
)

func TestScheduledTaskCreateResolvesNamespaceAndProjectsStatus(t *testing.T) {
	ctx := context.Background()
	resource := scheduledTaskFixture()
	fake := &fakeScheduledTaskOCIClient{
		listNamespacesFunc: func(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error) {
			return namespaceResponse(testNamespaceName, testCompartmentID), nil
		},
		listScheduledTasksFunc: func(context.Context, loganalyticssdk.ListScheduledTasksRequest) (loganalyticssdk.ListScheduledTasksResponse, error) {
			return loganalyticssdk.ListScheduledTasksResponse{}, nil
		},
		createScheduledTaskFunc: func(_ context.Context, request loganalyticssdk.CreateScheduledTaskRequest) (loganalyticssdk.CreateScheduledTaskResponse, error) {
			requireCreateStandardTaskRequest(t, request, testNamespaceName)
			return loganalyticssdk.CreateScheduledTaskResponse{
				ScheduledTask: standardTask(testScheduledTaskID, "daily search", testCompartmentID, loganalyticssdk.ScheduledTaskLifecycleStateActive),
				OpcRequestId:  common.String("opc-create-1"),
			}, nil
		},
		getScheduledTaskFunc: func(context.Context, loganalyticssdk.GetScheduledTaskRequest) (loganalyticssdk.GetScheduledTaskResponse, error) {
			return loganalyticssdk.GetScheduledTaskResponse{
				ScheduledTask: standardTask(testScheduledTaskID, "daily search", testCompartmentID, loganalyticssdk.ScheduledTaskLifecycleStateActive),
			}, nil
		},
	}

	response, err := newTestScheduledTaskClient(fake).CreateOrUpdate(ctx, resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() did not report success")
	}
	if resource.Namespace != "k8s-ns" {
		t.Fatalf("resource namespace = %q, want restored Kubernetes namespace", resource.Namespace)
	}
	requireScheduledTaskStatus(t, resource, testScheduledTaskID, "daily search", string(loganalyticssdk.ScheduledTaskLifecycleStateActive))
	requireScheduledTaskOpcRequestID(t, resource, "opc-create-1")
	requireTrailingScheduledTaskCondition(t, resource, shared.Active)
	if got := len(fake.listScheduledTasksRequests); got != 1 {
		t.Fatalf("ListScheduledTasks calls = %d, want 1", got)
	}
	requireListScheduledTaskRequest(t, fake.listScheduledTasksRequests[0], "", testNamespaceName)
}

func TestScheduledTaskAccelerationCreateBodyUsesPolymorphicDetails(t *testing.T) {
	resource := scheduledTaskFixture()
	resource.Spec.Kind = "ACCELERATION"
	resource.Spec.TaskType = ""
	resource.Spec.SavedSearchId = testSavedSearchID
	resource.Spec.Action = loganalyticsv1beta1.ScheduledTaskAction{}
	resource.Spec.Schedules = nil

	body, err := buildScheduledTaskCreateBody(resource)
	if err != nil {
		t.Fatalf("buildScheduledTaskCreateBody() error = %v", err)
	}
	details, ok := body.(loganalyticssdk.CreateAccelerationTaskDetails)
	if !ok {
		t.Fatalf("create body = %T, want CreateAccelerationTaskDetails", body)
	}
	if got := stringValue(details.CompartmentId); got != testCompartmentID {
		t.Fatalf("acceleration compartmentId = %q, want %q", got, testCompartmentID)
	}
	if got := stringValue(details.SavedSearchId); got != testSavedSearchID {
		t.Fatalf("acceleration savedSearchId = %q, want %q", got, testSavedSearchID)
	}
}

func TestScheduledTaskAccelerationRejectsMissingDisplayNameBeforeCreate(t *testing.T) {
	ctx := context.Background()
	resource := scheduledTaskAccelerationFixture()
	resource.Spec.DisplayName = ""
	fake := &fakeScheduledTaskOCIClient{
		listNamespacesFunc: func(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error) {
			return namespaceResponse(testNamespaceName, testCompartmentID), nil
		},
		listScheduledTasksFunc: func(context.Context, loganalyticssdk.ListScheduledTasksRequest) (loganalyticssdk.ListScheduledTasksResponse, error) {
			t.Fatal("ListScheduledTasks should not be called without safe acceleration identity")
			return loganalyticssdk.ListScheduledTasksResponse{}, nil
		},
		createScheduledTaskFunc: func(context.Context, loganalyticssdk.CreateScheduledTaskRequest) (loganalyticssdk.CreateScheduledTaskResponse, error) {
			t.Fatal("CreateScheduledTask should not be called without safe acceleration identity")
			return loganalyticssdk.CreateScheduledTaskResponse{}, nil
		},
	}

	response, err := newTestScheduledTaskClient(fake).CreateOrUpdate(ctx, resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() expected missing displayName error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() unexpectedly reported success")
	}
	if !strings.Contains(err.Error(), "spec.displayName is required for ACCELERATION ScheduledTask create-or-bind identity") {
		t.Fatalf("CreateOrUpdate() error = %q, want unsafe acceleration identity rejection", err.Error())
	}
	if got := len(fake.createScheduledTaskRequests); got != 0 {
		t.Fatalf("CreateScheduledTask calls = %d, want 0", got)
	}
}

func TestScheduledTaskAccelerationProjectionStoresSavedSearchID(t *testing.T) {
	resource := scheduledTaskAccelerationFixture()

	err := projectScheduledTaskStatusFromResponse(resource, loganalyticssdk.GetScheduledTaskResponse{
		ScheduledTask: accelerationTaskReadback(testScheduledTaskID, "accelerated search", testCompartmentID),
	})
	if err != nil {
		t.Fatalf("projectScheduledTaskStatusFromResponse() error = %v", err)
	}
	if got := scheduledTaskSavedSearchIDFromJSON(resource.Status.JsonData); got != testSavedSearchID {
		t.Fatalf("status.jsonData savedSearchId = %q, want %q", got, testSavedSearchID)
	}
}

func TestScheduledTaskBindUsesAllListPagesAndSkipsCreate(t *testing.T) {
	ctx := context.Background()
	resource := scheduledTaskFixture()
	page := "page-2"
	fake := &fakeScheduledTaskOCIClient{
		listNamespacesFunc: func(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error) {
			return namespaceResponse(testNamespaceName, testCompartmentID), nil
		},
		listScheduledTasksFunc: func(_ context.Context, request loganalyticssdk.ListScheduledTasksRequest) (loganalyticssdk.ListScheduledTasksResponse, error) {
			if request.Page == nil {
				return loganalyticssdk.ListScheduledTasksResponse{OpcNextPage: common.String(page)}, nil
			}
			if *request.Page != page {
				t.Fatalf("list page = %q, want %q", *request.Page, page)
			}
			return loganalyticssdk.ListScheduledTasksResponse{ScheduledTaskCollection: loganalyticssdk.ScheduledTaskCollection{
				Items: []loganalyticssdk.ScheduledTaskSummary{scheduledTaskSummary(testScheduledTaskID, "daily search", testCompartmentID)},
			}}, nil
		},
		getScheduledTaskFunc: func(context.Context, loganalyticssdk.GetScheduledTaskRequest) (loganalyticssdk.GetScheduledTaskResponse, error) {
			return loganalyticssdk.GetScheduledTaskResponse{
				ScheduledTask: standardTask(testScheduledTaskID, "daily search", testCompartmentID, loganalyticssdk.ScheduledTaskLifecycleStateActive),
			}, nil
		},
		createScheduledTaskFunc: func(context.Context, loganalyticssdk.CreateScheduledTaskRequest) (loganalyticssdk.CreateScheduledTaskResponse, error) {
			t.Fatal("CreateScheduledTask should not be called when list binds an existing task")
			return loganalyticssdk.CreateScheduledTaskResponse{}, nil
		},
		updateScheduledTaskFunc: func(context.Context, loganalyticssdk.UpdateScheduledTaskRequest) (loganalyticssdk.UpdateScheduledTaskResponse, error) {
			t.Fatal("UpdateScheduledTask should not be called for matching observed state")
			return loganalyticssdk.UpdateScheduledTaskResponse{}, nil
		},
	}

	response, err := newTestScheduledTaskClient(fake).CreateOrUpdate(ctx, resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() did not report success")
	}
	if got := len(fake.listScheduledTasksRequests); got != 2 {
		t.Fatalf("ListScheduledTasks calls = %d, want 2", got)
	}
	requireListScheduledTaskRequest(t, fake.listScheduledTasksRequests[0], "", testNamespaceName)
	requireListScheduledTaskRequest(t, fake.listScheduledTasksRequests[1], page, testNamespaceName)
	requireScheduledTaskStatus(t, resource, testScheduledTaskID, "daily search", string(loganalyticssdk.ScheduledTaskLifecycleStateActive))
}

func TestScheduledTaskMutableUpdateShapesUpdateBody(t *testing.T) {
	ctx := context.Background()
	resource := scheduledTaskFixture()
	resource.Status.OsokStatus.Ocid = shared.OCID(testScheduledTaskID)
	resource.Spec.DisplayName = "daily search renamed"
	getCalls := 0
	fake := &fakeScheduledTaskOCIClient{
		listNamespacesFunc: func(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error) {
			return namespaceResponse(testNamespaceName, testCompartmentID), nil
		},
		getScheduledTaskFunc: func(_ context.Context, request loganalyticssdk.GetScheduledTaskRequest) (loganalyticssdk.GetScheduledTaskResponse, error) {
			if got := stringValue(request.NamespaceName); got != testNamespaceName {
				t.Fatalf("get namespaceName = %q, want %q", got, testNamespaceName)
			}
			getCalls++
			displayName := "daily search"
			if getCalls > 1 {
				displayName = "daily search renamed"
			}
			return loganalyticssdk.GetScheduledTaskResponse{
				ScheduledTask: standardTask(testScheduledTaskID, displayName, testCompartmentID, loganalyticssdk.ScheduledTaskLifecycleStateActive),
			}, nil
		},
		updateScheduledTaskFunc: func(_ context.Context, request loganalyticssdk.UpdateScheduledTaskRequest) (loganalyticssdk.UpdateScheduledTaskResponse, error) {
			requireUpdateStandardTaskRequest(t, request, testNamespaceName, testScheduledTaskID, "daily search renamed")
			return loganalyticssdk.UpdateScheduledTaskResponse{
				ScheduledTask: standardTask(testScheduledTaskID, "daily search renamed", testCompartmentID, loganalyticssdk.ScheduledTaskLifecycleStateActive),
				OpcRequestId:  common.String("opc-update-1"),
			}, nil
		},
	}

	response, err := newTestScheduledTaskClient(fake).CreateOrUpdate(ctx, resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() did not report success")
	}
	if got := len(fake.updateScheduledTaskRequests); got != 1 {
		t.Fatalf("UpdateScheduledTask calls = %d, want 1", got)
	}
	requireScheduledTaskStatus(t, resource, testScheduledTaskID, "daily search renamed", string(loganalyticssdk.ScheduledTaskLifecycleStateActive))
	requireScheduledTaskOpcRequestID(t, resource, "opc-update-1")
}

func TestScheduledTaskCreateOnlyDriftRejectsUnsafeUpdate(t *testing.T) {
	ctx := context.Background()
	resource := scheduledTaskFixture()
	resource.Status.OsokStatus.Ocid = shared.OCID(testScheduledTaskID)
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..changed"
	fake := &fakeScheduledTaskOCIClient{
		listNamespacesFunc: func(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error) {
			return namespaceResponse(testNamespaceName, "ocid1.compartment.oc1..changed"), nil
		},
		getScheduledTaskFunc: func(context.Context, loganalyticssdk.GetScheduledTaskRequest) (loganalyticssdk.GetScheduledTaskResponse, error) {
			return loganalyticssdk.GetScheduledTaskResponse{
				ScheduledTask: standardTask(testScheduledTaskID, "daily search", testCompartmentID, loganalyticssdk.ScheduledTaskLifecycleStateActive),
			}, nil
		},
		updateScheduledTaskFunc: func(context.Context, loganalyticssdk.UpdateScheduledTaskRequest) (loganalyticssdk.UpdateScheduledTaskResponse, error) {
			t.Fatal("UpdateScheduledTask should not be called for create-only drift")
			return loganalyticssdk.UpdateScheduledTaskResponse{}, nil
		},
	}

	response, err := newTestScheduledTaskClient(fake).CreateOrUpdate(ctx, resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() expected create-only drift error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() unexpectedly reported success")
	}
	if !strings.Contains(err.Error(), "compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want compartmentId drift", err.Error())
	}
	if got := len(fake.updateScheduledTaskRequests); got != 0 {
		t.Fatalf("UpdateScheduledTask calls = %d, want 0", got)
	}
	requireTrailingScheduledTaskCondition(t, resource, shared.Failed)
}

func TestScheduledTaskAccelerationSavedSearchDriftRejectsUnsafeUpdate(t *testing.T) {
	ctx := context.Background()
	resource := scheduledTaskAccelerationFixture()
	resource.Status.OsokStatus.Ocid = shared.OCID(testScheduledTaskID)
	resource.Status.Id = testScheduledTaskID
	resource.Status.JsonData = fmt.Sprintf(`{"%s":"%s"}`, scheduledTaskJSONSavedSearchID, testSavedSearchID)
	resource.Spec.SavedSearchId = "ocid1.managementsavedsearch.oc1..changed"
	fake := &fakeScheduledTaskOCIClient{
		listNamespacesFunc: func(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error) {
			return namespaceResponse(testNamespaceName, testCompartmentID), nil
		},
		getScheduledTaskFunc: func(context.Context, loganalyticssdk.GetScheduledTaskRequest) (loganalyticssdk.GetScheduledTaskResponse, error) {
			return loganalyticssdk.GetScheduledTaskResponse{
				ScheduledTask: accelerationTaskReadback(testScheduledTaskID, "accelerated search", testCompartmentID),
			}, nil
		},
		updateScheduledTaskFunc: func(context.Context, loganalyticssdk.UpdateScheduledTaskRequest) (loganalyticssdk.UpdateScheduledTaskResponse, error) {
			t.Fatal("UpdateScheduledTask should not be called for savedSearchId drift")
			return loganalyticssdk.UpdateScheduledTaskResponse{}, nil
		},
	}

	response, err := newTestScheduledTaskClient(fake).CreateOrUpdate(ctx, resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() expected savedSearchId drift error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() unexpectedly reported success")
	}
	if !strings.Contains(err.Error(), "savedSearchId changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want savedSearchId drift", err.Error())
	}
	if got := len(fake.updateScheduledTaskRequests); got != 0 {
		t.Fatalf("UpdateScheduledTask calls = %d, want 0", got)
	}
}

func TestScheduledTaskDeleteKeepsFinalizerUntilDeletionConfirmed(t *testing.T) {
	ctx := context.Background()
	resource := scheduledTaskFixture()
	resource.Status.OsokStatus.Ocid = shared.OCID(testScheduledTaskID)
	fake := &fakeScheduledTaskOCIClient{
		listNamespacesFunc: func(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error) {
			return namespaceResponse(testNamespaceName, testCompartmentID), nil
		},
		getScheduledTaskFunc: func(context.Context, loganalyticssdk.GetScheduledTaskRequest) (loganalyticssdk.GetScheduledTaskResponse, error) {
			return loganalyticssdk.GetScheduledTaskResponse{
				ScheduledTask: standardTask(testScheduledTaskID, "daily search", testCompartmentID, loganalyticssdk.ScheduledTaskLifecycleStateActive),
			}, nil
		},
		deleteScheduledTaskFunc: func(_ context.Context, request loganalyticssdk.DeleteScheduledTaskRequest) (loganalyticssdk.DeleteScheduledTaskResponse, error) {
			if got := stringValue(request.NamespaceName); got != testNamespaceName {
				t.Fatalf("delete namespaceName = %q, want %q", got, testNamespaceName)
			}
			return loganalyticssdk.DeleteScheduledTaskResponse{OpcRequestId: common.String("opc-delete-1")}, nil
		},
	}

	deleted, err := newTestScheduledTaskClient(fake).Delete(ctx, resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() = deleted, want finalizer retained while lifecycle is ACTIVE")
	}
	requireScheduledTaskOpcRequestID(t, resource, "opc-delete-1")
	requireTrailingScheduledTaskCondition(t, resource, shared.Terminating)
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("DeletedAt was set before OCI deletion confirmation")
	}
}

func TestScheduledTaskDeleteRejectsAuthShapedNotFound(t *testing.T) {
	ctx := context.Background()
	resource := scheduledTaskFixture()
	resource.Status.OsokStatus.Ocid = shared.OCID(testScheduledTaskID)
	fake := &fakeScheduledTaskOCIClient{
		listNamespacesFunc: func(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error) {
			return namespaceResponse(testNamespaceName, testCompartmentID), nil
		},
		getScheduledTaskFunc: func(context.Context, loganalyticssdk.GetScheduledTaskRequest) (loganalyticssdk.GetScheduledTaskResponse, error) {
			return loganalyticssdk.GetScheduledTaskResponse{
				ScheduledTask: standardTask(testScheduledTaskID, "daily search", testCompartmentID, loganalyticssdk.ScheduledTaskLifecycleStateActive),
			}, nil
		},
		deleteScheduledTaskFunc: func(context.Context, loganalyticssdk.DeleteScheduledTaskRequest) (loganalyticssdk.DeleteScheduledTaskResponse, error) {
			return loganalyticssdk.DeleteScheduledTaskResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
	}

	deleted, err := newTestScheduledTaskClient(fake).Delete(ctx, resource)
	if err == nil {
		t.Fatal("Delete() expected auth-shaped not-found error")
	}
	if deleted {
		t.Fatal("Delete() = deleted, want finalizer retained for auth-shaped not-found")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want auth-shaped ambiguity", err.Error())
	}
	requireScheduledTaskOpcRequestID(t, resource, "opc-request-id")
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("DeletedAt was set for auth-shaped not-found")
	}
}

func TestScheduledTaskDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	ctx := context.Background()
	resource := scheduledTaskFixture()
	resource.Status.OsokStatus.Ocid = shared.OCID(testScheduledTaskID)
	fake := &fakeScheduledTaskOCIClient{
		listNamespacesFunc: func(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error) {
			return namespaceResponse(testNamespaceName, testCompartmentID), nil
		},
		getScheduledTaskFunc: func(context.Context, loganalyticssdk.GetScheduledTaskRequest) (loganalyticssdk.GetScheduledTaskResponse, error) {
			return loganalyticssdk.GetScheduledTaskResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
		},
		deleteScheduledTaskFunc: func(context.Context, loganalyticssdk.DeleteScheduledTaskRequest) (loganalyticssdk.DeleteScheduledTaskResponse, error) {
			t.Fatal("DeleteScheduledTask should not be called after auth-shaped pre-delete confirm read")
			return loganalyticssdk.DeleteScheduledTaskResponse{}, nil
		},
	}

	deleted, err := newTestScheduledTaskClient(fake).Delete(ctx, resource)
	if err == nil {
		t.Fatal("Delete() expected auth-shaped pre-delete confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() = deleted, want finalizer retained for auth-shaped confirm read")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound before delete") {
		t.Fatalf("Delete() error = %q, want pre-delete auth-shaped ambiguity", err.Error())
	}
	requireScheduledTaskOpcRequestID(t, resource, "opc-request-id")
	if got := len(fake.deleteScheduledTaskRequests); got != 0 {
		t.Fatalf("DeleteScheduledTask calls = %d, want 0", got)
	}
}

func TestScheduledTaskCreateErrorCapturesOpcRequestID(t *testing.T) {
	ctx := context.Background()
	resource := scheduledTaskFixture()
	fake := &fakeScheduledTaskOCIClient{
		listNamespacesFunc: func(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error) {
			return namespaceResponse(testNamespaceName, testCompartmentID), nil
		},
		listScheduledTasksFunc: func(context.Context, loganalyticssdk.ListScheduledTasksRequest) (loganalyticssdk.ListScheduledTasksResponse, error) {
			return loganalyticssdk.ListScheduledTasksResponse{}, nil
		},
		createScheduledTaskFunc: func(context.Context, loganalyticssdk.CreateScheduledTaskRequest) (loganalyticssdk.CreateScheduledTaskResponse, error) {
			return loganalyticssdk.CreateScheduledTaskResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
		},
	}

	response, err := newTestScheduledTaskClient(fake).CreateOrUpdate(ctx, resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() expected create error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() unexpectedly reported success")
	}
	requireScheduledTaskOpcRequestID(t, resource, "opc-request-id")
	requireTrailingScheduledTaskCondition(t, resource, shared.Failed)
}

type fakeScheduledTaskOCIClient struct {
	createScheduledTaskFunc func(context.Context, loganalyticssdk.CreateScheduledTaskRequest) (loganalyticssdk.CreateScheduledTaskResponse, error)
	getScheduledTaskFunc    func(context.Context, loganalyticssdk.GetScheduledTaskRequest) (loganalyticssdk.GetScheduledTaskResponse, error)
	listScheduledTasksFunc  func(context.Context, loganalyticssdk.ListScheduledTasksRequest) (loganalyticssdk.ListScheduledTasksResponse, error)
	updateScheduledTaskFunc func(context.Context, loganalyticssdk.UpdateScheduledTaskRequest) (loganalyticssdk.UpdateScheduledTaskResponse, error)
	deleteScheduledTaskFunc func(context.Context, loganalyticssdk.DeleteScheduledTaskRequest) (loganalyticssdk.DeleteScheduledTaskResponse, error)
	listNamespacesFunc      func(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error)

	createScheduledTaskRequests []loganalyticssdk.CreateScheduledTaskRequest
	getScheduledTaskRequests    []loganalyticssdk.GetScheduledTaskRequest
	listScheduledTasksRequests  []loganalyticssdk.ListScheduledTasksRequest
	updateScheduledTaskRequests []loganalyticssdk.UpdateScheduledTaskRequest
	deleteScheduledTaskRequests []loganalyticssdk.DeleteScheduledTaskRequest
	listNamespacesRequests      []loganalyticssdk.ListNamespacesRequest
}

func (f *fakeScheduledTaskOCIClient) CreateScheduledTask(
	ctx context.Context,
	request loganalyticssdk.CreateScheduledTaskRequest,
) (loganalyticssdk.CreateScheduledTaskResponse, error) {
	f.createScheduledTaskRequests = append(f.createScheduledTaskRequests, request)
	if f.createScheduledTaskFunc == nil {
		return loganalyticssdk.CreateScheduledTaskResponse{}, fmt.Errorf("unexpected CreateScheduledTask")
	}
	return f.createScheduledTaskFunc(ctx, request)
}

func (f *fakeScheduledTaskOCIClient) GetScheduledTask(
	ctx context.Context,
	request loganalyticssdk.GetScheduledTaskRequest,
) (loganalyticssdk.GetScheduledTaskResponse, error) {
	f.getScheduledTaskRequests = append(f.getScheduledTaskRequests, request)
	if f.getScheduledTaskFunc == nil {
		return loganalyticssdk.GetScheduledTaskResponse{}, fmt.Errorf("unexpected GetScheduledTask")
	}
	return f.getScheduledTaskFunc(ctx, request)
}

func (f *fakeScheduledTaskOCIClient) ListScheduledTasks(
	ctx context.Context,
	request loganalyticssdk.ListScheduledTasksRequest,
) (loganalyticssdk.ListScheduledTasksResponse, error) {
	f.listScheduledTasksRequests = append(f.listScheduledTasksRequests, request)
	if f.listScheduledTasksFunc == nil {
		return loganalyticssdk.ListScheduledTasksResponse{}, fmt.Errorf("unexpected ListScheduledTasks")
	}
	return f.listScheduledTasksFunc(ctx, request)
}

func (f *fakeScheduledTaskOCIClient) UpdateScheduledTask(
	ctx context.Context,
	request loganalyticssdk.UpdateScheduledTaskRequest,
) (loganalyticssdk.UpdateScheduledTaskResponse, error) {
	f.updateScheduledTaskRequests = append(f.updateScheduledTaskRequests, request)
	if f.updateScheduledTaskFunc == nil {
		return loganalyticssdk.UpdateScheduledTaskResponse{}, fmt.Errorf("unexpected UpdateScheduledTask")
	}
	return f.updateScheduledTaskFunc(ctx, request)
}

func (f *fakeScheduledTaskOCIClient) DeleteScheduledTask(
	ctx context.Context,
	request loganalyticssdk.DeleteScheduledTaskRequest,
) (loganalyticssdk.DeleteScheduledTaskResponse, error) {
	f.deleteScheduledTaskRequests = append(f.deleteScheduledTaskRequests, request)
	if f.deleteScheduledTaskFunc == nil {
		return loganalyticssdk.DeleteScheduledTaskResponse{}, fmt.Errorf("unexpected DeleteScheduledTask")
	}
	return f.deleteScheduledTaskFunc(ctx, request)
}

func (f *fakeScheduledTaskOCIClient) ListNamespaces(
	ctx context.Context,
	request loganalyticssdk.ListNamespacesRequest,
) (loganalyticssdk.ListNamespacesResponse, error) {
	f.listNamespacesRequests = append(f.listNamespacesRequests, request)
	if f.listNamespacesFunc == nil {
		return loganalyticssdk.ListNamespacesResponse{}, fmt.Errorf("unexpected ListNamespaces")
	}
	return f.listNamespacesFunc(ctx, request)
}

func newTestScheduledTaskClient(fake *fakeScheduledTaskOCIClient) ScheduledTaskServiceClient {
	manager := &ScheduledTaskServiceManager{}
	hooks := newScheduledTaskDefaultRuntimeHooks(loganalyticssdk.LogAnalyticsClient{})
	applyScheduledTaskRuntimeHooks(manager, &hooks, fake, nil)
	config := buildScheduledTaskGeneratedRuntimeConfig(manager, hooks)
	delegate := defaultScheduledTaskServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loganalyticsv1beta1.ScheduledTask](config),
	}
	return wrapScheduledTaskGeneratedClient(hooks, delegate)
}

func scheduledTaskFixture() *loganalyticsv1beta1.ScheduledTask {
	return &loganalyticsv1beta1.ScheduledTask{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scheduled-task",
			Namespace: "k8s-ns",
			UID:       types.UID("scheduled-task-uid"),
		},
		Spec: loganalyticsv1beta1.ScheduledTaskSpec{
			CompartmentId: testCompartmentID,
			DisplayName:   "daily search",
			Kind:          "STANDARD",
			TaskType:      string(loganalyticssdk.TaskTypeSavedSearch),
			Action: loganalyticsv1beta1.ScheduledTaskAction{
				Type:                string(loganalyticssdk.ActionTypeStream),
				SavedSearchId:       testSavedSearchID,
				SavedSearchDuration: "PT5M",
			},
			Schedules: []loganalyticsv1beta1.ScheduledTaskSchedule{{
				Type:              string(loganalyticssdk.ScheduleTypeFixedFrequency),
				RecurringInterval: "PT5M",
				RepeatCount:       0,
			}},
		},
	}
}

func scheduledTaskAccelerationFixture() *loganalyticsv1beta1.ScheduledTask {
	resource := scheduledTaskFixture()
	resource.Spec.Kind = scheduledTaskKindAcceleration
	resource.Spec.TaskType = ""
	resource.Spec.SavedSearchId = testSavedSearchID
	resource.Spec.Action = loganalyticsv1beta1.ScheduledTaskAction{}
	resource.Spec.Schedules = nil
	resource.Spec.DisplayName = "accelerated search"
	return resource
}

func namespaceResponse(namespace string, compartmentID string) loganalyticssdk.ListNamespacesResponse {
	return loganalyticssdk.ListNamespacesResponse{NamespaceCollection: loganalyticssdk.NamespaceCollection{
		Items: []loganalyticssdk.NamespaceSummary{{
			NamespaceName: common.String(namespace),
			CompartmentId: common.String(compartmentID),
			IsOnboarded:   common.Bool(true),
		}},
	}}
}

func scheduledTaskSummary(id string, displayName string, compartmentID string) loganalyticssdk.ScheduledTaskSummary {
	return loganalyticssdk.ScheduledTaskSummary{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String(compartmentID),
		TaskType:       loganalyticssdk.TaskTypeSavedSearch,
		LifecycleState: loganalyticssdk.ScheduledTaskLifecycleStateActive,
		TimeCreated:    sdkTestTime(),
		TimeUpdated:    sdkTestTime(),
	}
}

func standardTask(
	id string,
	displayName string,
	compartmentID string,
	lifecycle loganalyticssdk.ScheduledTaskLifecycleStateEnum,
) loganalyticssdk.StandardTask {
	return loganalyticssdk.StandardTask{
		Id:            common.String(id),
		DisplayName:   common.String(displayName),
		CompartmentId: common.String(compartmentID),
		TaskType:      loganalyticssdk.TaskTypeSavedSearch,
		Action: loganalyticssdk.StreamAction{
			SavedSearchId:       common.String(testSavedSearchID),
			SavedSearchDuration: common.String("PT5M"),
		},
		Schedules: []loganalyticssdk.Schedule{loganalyticssdk.FixedFrequencySchedule{
			RecurringInterval: common.String("PT5M"),
			RepeatCount:       common.Int(0),
		}},
		LifecycleState: lifecycle,
		TimeCreated:    sdkTestTime(),
		TimeUpdated:    sdkTestTime(),
	}
}

type testAccelerationTaskReadback struct {
	Id                  *string                                         `json:"id"`
	DisplayName         *string                                         `json:"displayName"`
	TaskType            loganalyticssdk.TaskTypeEnum                    `json:"taskType"`
	Schedules           []loganalyticssdk.Schedule                      `json:"schedules,omitempty"`
	Action              loganalyticssdk.Action                          `json:"action,omitempty"`
	CompartmentId       *string                                         `json:"compartmentId"`
	TimeCreated         *common.SDKTime                                 `json:"timeCreated"`
	TimeUpdated         *common.SDKTime                                 `json:"timeUpdated"`
	LifecycleState      loganalyticssdk.ScheduledTaskLifecycleStateEnum `json:"lifecycleState"`
	Description         *string                                         `json:"description,omitempty"`
	TaskStatus          loganalyticssdk.ScheduledTaskTaskStatusEnum     `json:"taskStatus,omitempty"`
	PauseReason         loganalyticssdk.ScheduledTaskPauseReasonEnum    `json:"pauseReason,omitempty"`
	WorkRequestId       *string                                         `json:"workRequestId,omitempty"`
	NumOccurrences      *int64                                          `json:"numOccurrences,omitempty"`
	TimeOfNextExecution *common.SDKTime                                 `json:"timeOfNextExecution,omitempty"`
	FreeformTags        map[string]string                               `json:"freeformTags,omitempty"`
	DefinedTags         map[string]map[string]interface{}               `json:"definedTags,omitempty"`
	Kind                string                                          `json:"kind"`
}

func accelerationTaskReadback(id string, displayName string, compartmentID string) testAccelerationTaskReadback {
	return testAccelerationTaskReadback{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		TaskType:       loganalyticssdk.TaskTypeAcceleration,
		CompartmentId:  common.String(compartmentID),
		TimeCreated:    sdkTestTime(),
		TimeUpdated:    sdkTestTime(),
		LifecycleState: loganalyticssdk.ScheduledTaskLifecycleStateActive,
		Kind:           scheduledTaskKindAcceleration,
	}
}

func (m testAccelerationTaskReadback) GetId() *string {
	return m.Id
}

func (m testAccelerationTaskReadback) GetDisplayName() *string {
	return m.DisplayName
}

func (m testAccelerationTaskReadback) GetTaskType() loganalyticssdk.TaskTypeEnum {
	return m.TaskType
}

func (m testAccelerationTaskReadback) GetSchedules() []loganalyticssdk.Schedule {
	return m.Schedules
}

func (m testAccelerationTaskReadback) GetAction() loganalyticssdk.Action {
	return m.Action
}

func (m testAccelerationTaskReadback) GetCompartmentId() *string {
	return m.CompartmentId
}

func (m testAccelerationTaskReadback) GetTimeCreated() *common.SDKTime {
	return m.TimeCreated
}

func (m testAccelerationTaskReadback) GetTimeUpdated() *common.SDKTime {
	return m.TimeUpdated
}

func (m testAccelerationTaskReadback) GetLifecycleState() loganalyticssdk.ScheduledTaskLifecycleStateEnum {
	return m.LifecycleState
}

func (m testAccelerationTaskReadback) GetDescription() *string {
	return m.Description
}

func (m testAccelerationTaskReadback) GetTaskStatus() loganalyticssdk.ScheduledTaskTaskStatusEnum {
	return m.TaskStatus
}

func (m testAccelerationTaskReadback) GetPauseReason() loganalyticssdk.ScheduledTaskPauseReasonEnum {
	return m.PauseReason
}

func (m testAccelerationTaskReadback) GetWorkRequestId() *string {
	return m.WorkRequestId
}

func (m testAccelerationTaskReadback) GetNumOccurrences() *int64 {
	return m.NumOccurrences
}

func (m testAccelerationTaskReadback) GetTimeOfNextExecution() *common.SDKTime {
	return m.TimeOfNextExecution
}

func (m testAccelerationTaskReadback) GetFreeformTags() map[string]string {
	return m.FreeformTags
}

func (m testAccelerationTaskReadback) GetDefinedTags() map[string]map[string]interface{} {
	return m.DefinedTags
}

func sdkTestTime() *common.SDKTime {
	return &common.SDKTime{Time: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)}
}

func requireCreateStandardTaskRequest(t *testing.T, request loganalyticssdk.CreateScheduledTaskRequest, namespace string) {
	t.Helper()
	if got := stringValue(request.NamespaceName); got != namespace {
		t.Fatalf("create namespaceName = %q, want %q", got, namespace)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("create request missing deterministic retry token")
	}
	body, ok := request.CreateScheduledTaskDetails.(loganalyticssdk.CreateStandardTaskDetails)
	if !ok {
		t.Fatalf("create body = %T, want CreateStandardTaskDetails", request.CreateScheduledTaskDetails)
	}
	if got := stringValue(body.CompartmentId); got != testCompartmentID {
		t.Fatalf("create body compartmentId = %q, want %q", got, testCompartmentID)
	}
	if body.TaskType != loganalyticssdk.TaskTypeSavedSearch {
		t.Fatalf("create body taskType = %q, want SAVED_SEARCH", body.TaskType)
	}
	if _, ok := body.Action.(loganalyticssdk.StreamAction); !ok {
		t.Fatalf("create body action = %T, want StreamAction", body.Action)
	}
	requireFixedFrequencySchedule(t, body.Schedules)
}

func requireFixedFrequencySchedule(t *testing.T, schedules []loganalyticssdk.Schedule) {
	t.Helper()
	if len(schedules) != 1 {
		t.Fatalf("create body schedules len = %d, want 1", len(schedules))
	}
	schedule, ok := schedules[0].(loganalyticssdk.FixedFrequencySchedule)
	if !ok {
		t.Fatalf("create body schedule = %T, want FixedFrequencySchedule", schedules[0])
	}
	if schedule.RepeatCount == nil || *schedule.RepeatCount != 0 {
		t.Fatalf("create body repeatCount = %v, want explicit zero", schedule.RepeatCount)
	}
}

func requireUpdateStandardTaskRequest(
	t *testing.T,
	request loganalyticssdk.UpdateScheduledTaskRequest,
	namespace string,
	id string,
	displayName string,
) {
	t.Helper()
	if got := stringValue(request.NamespaceName); got != namespace {
		t.Fatalf("update namespaceName = %q, want %q", got, namespace)
	}
	if got := stringValue(request.ScheduledTaskId); got != id {
		t.Fatalf("update scheduledTaskId = %q, want %q", got, id)
	}
	body, ok := request.UpdateScheduledTaskDetails.(loganalyticssdk.UpdateStandardTaskDetails)
	if !ok {
		t.Fatalf("update body = %T, want UpdateStandardTaskDetails", request.UpdateScheduledTaskDetails)
	}
	if got := stringValue(body.DisplayName); got != displayName {
		t.Fatalf("update body displayName = %q, want %q", got, displayName)
	}
	if len(body.Schedules) != 1 {
		t.Fatalf("update body schedules len = %d, want 1", len(body.Schedules))
	}
}

func requireListScheduledTaskRequest(t *testing.T, request loganalyticssdk.ListScheduledTasksRequest, page string, namespace string) {
	t.Helper()
	if got := stringValue(request.NamespaceName); got != namespace {
		t.Fatalf("list namespaceName = %q, want %q", got, namespace)
	}
	if got := string(request.TaskType); got != string(loganalyticssdk.ListScheduledTasksTaskTypeSavedSearch) {
		t.Fatalf("list taskType = %q, want SAVED_SEARCH", got)
	}
	if got := stringValue(request.CompartmentId); got != testCompartmentID {
		t.Fatalf("list compartmentId = %q, want %q", got, testCompartmentID)
	}
	if got := stringValue(request.DisplayName); got != "daily search" {
		t.Fatalf("list displayName = %q, want daily search", got)
	}
	if got := stringValue(request.Page); got != page {
		t.Fatalf("list page = %q, want %q", got, page)
	}
}

func requireScheduledTaskStatus(t *testing.T, resource *loganalyticsv1beta1.ScheduledTask, id string, displayName string, lifecycle string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != id {
		t.Fatalf("status.status.ocid = %q, want %q", got, id)
	}
	if resource.Status.Id != id {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, id)
	}
	if resource.Status.DisplayName != displayName {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, displayName)
	}
	if resource.Status.LifecycleState != lifecycle {
		t.Fatalf("status.lifecycleState = %q, want %q", resource.Status.LifecycleState, lifecycle)
	}
}

func requireScheduledTaskOpcRequestID(t *testing.T, resource *loganalyticsv1beta1.ScheduledTask, want string) {
	t.Helper()
	if got := resource.Status.OsokStatus.OpcRequestID; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func requireTrailingScheduledTaskCondition(
	t *testing.T,
	resource *loganalyticsv1beta1.ScheduledTask,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status conditions empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("trailing condition = %s, want %s", got, want)
	}
}
