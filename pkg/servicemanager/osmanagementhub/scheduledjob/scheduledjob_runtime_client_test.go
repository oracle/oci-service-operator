/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package scheduledjob

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	osmanagementhubsdk "github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeScheduledJobOCIClient struct {
	create func(context.Context, osmanagementhubsdk.CreateScheduledJobRequest) (osmanagementhubsdk.CreateScheduledJobResponse, error)
	get    func(context.Context, osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error)
	list   func(context.Context, osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error)
	update func(context.Context, osmanagementhubsdk.UpdateScheduledJobRequest) (osmanagementhubsdk.UpdateScheduledJobResponse, error)
	delete func(context.Context, osmanagementhubsdk.DeleteScheduledJobRequest) (osmanagementhubsdk.DeleteScheduledJobResponse, error)

	createRequests []osmanagementhubsdk.CreateScheduledJobRequest
	getRequests    []osmanagementhubsdk.GetScheduledJobRequest
	listRequests   []osmanagementhubsdk.ListScheduledJobsRequest
	updateRequests []osmanagementhubsdk.UpdateScheduledJobRequest
	deleteRequests []osmanagementhubsdk.DeleteScheduledJobRequest
}

func (f *fakeScheduledJobOCIClient) CreateScheduledJob(
	ctx context.Context,
	request osmanagementhubsdk.CreateScheduledJobRequest,
) (osmanagementhubsdk.CreateScheduledJobResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.create == nil {
		return osmanagementhubsdk.CreateScheduledJobResponse{}, nil
	}
	return f.create(ctx, request)
}

func (f *fakeScheduledJobOCIClient) GetScheduledJob(
	ctx context.Context,
	request osmanagementhubsdk.GetScheduledJobRequest,
) (osmanagementhubsdk.GetScheduledJobResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.get == nil {
		return osmanagementhubsdk.GetScheduledJobResponse{}, nil
	}
	return f.get(ctx, request)
}

func (f *fakeScheduledJobOCIClient) ListScheduledJobs(
	ctx context.Context,
	request osmanagementhubsdk.ListScheduledJobsRequest,
) (osmanagementhubsdk.ListScheduledJobsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.list == nil {
		return osmanagementhubsdk.ListScheduledJobsResponse{}, nil
	}
	return f.list(ctx, request)
}

func (f *fakeScheduledJobOCIClient) UpdateScheduledJob(
	ctx context.Context,
	request osmanagementhubsdk.UpdateScheduledJobRequest,
) (osmanagementhubsdk.UpdateScheduledJobResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.update == nil {
		return osmanagementhubsdk.UpdateScheduledJobResponse{}, nil
	}
	return f.update(ctx, request)
}

func (f *fakeScheduledJobOCIClient) DeleteScheduledJob(
	ctx context.Context,
	request osmanagementhubsdk.DeleteScheduledJobRequest,
) (osmanagementhubsdk.DeleteScheduledJobResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete == nil {
		return osmanagementhubsdk.DeleteScheduledJobResponse{}, nil
	}
	return f.delete(ctx, request)
}

func TestScheduledJobRuntimeSemantics(t *testing.T) {
	semantics := newScheduledJobRuntimeSemantics()
	if semantics == nil {
		t.Fatal("newScheduledJobRuntimeSemantics() = nil")
	}
	if got, want := semantics.FinalizerPolicy, "retain-until-confirmed-delete"; got != want {
		t.Fatalf("FinalizerPolicy = %q, want %q", got, want)
	}
	if semantics.Async == nil || semantics.Async.Strategy != "lifecycle" {
		t.Fatalf("Async = %+v, want lifecycle semantics", semantics.Async)
	}
	if got, want := semantics.Delete.Policy, "required"; got != want {
		t.Fatalf("Delete.Policy = %q, want %q", got, want)
	}
	if !containsScheduledJobString(semantics.Mutation.Mutable, "operations") {
		t.Fatalf("Mutation.Mutable = %#v, want operations mutable", semantics.Mutation.Mutable)
	}
	if !containsScheduledJobString(semantics.Mutation.ForceNew, "managedInstanceIds") {
		t.Fatalf("Mutation.ForceNew = %#v, want managedInstanceIds create-only", semantics.Mutation.ForceNew)
	}
}

//nolint:gocognit,gocyclo // The create path assertions keep request shaping and status projection together.
func TestScheduledJobCreateOrUpdateCreatesAndProjectsStatus(t *testing.T) {
	resource := newTestScheduledJob()
	client := &fakeScheduledJobOCIClient{}
	client.list = func(_ context.Context, request osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error) {
		assertStringPtr(t, "list compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
		if request.ScheduleType != osmanagementhubsdk.ListScheduledJobsScheduleTypeOnetime {
			t.Fatalf("list scheduleType = %q, want ONETIME", request.ScheduleType)
		}
		if len(client.listRequests) == 1 && request.OperationType != osmanagementhubsdk.ListScheduledJobsOperationTypeUpdateAll {
			t.Fatalf("list operationType = %q, want UPDATE_ALL", request.OperationType)
		}
		if len(client.listRequests) == 1 {
			assertStringPtr(t, "list managedInstanceId", request.ManagedInstanceId, resource.Spec.ManagedInstanceIds[0])
		}
		return osmanagementhubsdk.ListScheduledJobsResponse{}, nil
	}
	client.create = func(_ context.Context, request osmanagementhubsdk.CreateScheduledJobRequest) (osmanagementhubsdk.CreateScheduledJobResponse, error) {
		assertScheduledJobCreateRequest(t, request, resource.Spec)
		return osmanagementhubsdk.CreateScheduledJobResponse{
			ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..created", resource.Spec, osmanagementhubsdk.ScheduledJobLifecycleStateCreating),
			OpcRequestId: common.String("opc-create"),
		}, nil
	}
	client.get = func(_ context.Context, request osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
		assertStringPtr(t, "get scheduledJobId", request.ScheduledJobId, "ocid1.scheduledjob.oc1..created")
		return osmanagementhubsdk.GetScheduledJobResponse{
			ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..created", resource.Spec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
			OpcRequestId: common.String("opc-get"),
		}, nil
	}

	response, err := newTestScheduledJobClient(client).CreateOrUpdate(context.Background(), resource, testScheduledJobRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful without requeue", response)
	}
	assertScheduledJobCreatedStatus(t, resource, "ocid1.scheduledjob.oc1..created")
	if len(client.createRequests) != 1 {
		t.Fatalf("CreateScheduledJob() calls = %d, want 1", len(client.createRequests))
	}
}

//nolint:gocognit,gocyclo // The paginated bind regression is clearer with the staged list responses inline.
func TestScheduledJobCreateOrUpdateBindsExistingFromSecondListPage(t *testing.T) {
	resource := newTestScheduledJob()
	client := &fakeScheduledJobOCIClient{}
	client.list = func(_ context.Context, request osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error) {
		otherSpec := resource.Spec
		otherSpec.TimeNextExecution = "2026-05-03T10:00:00Z"
		if request.Page == nil {
			return osmanagementhubsdk.ListScheduledJobsResponse{
				ScheduledJobCollection: osmanagementhubsdk.ScheduledJobCollection{
					Items: []osmanagementhubsdk.ScheduledJobSummary{
						scheduledJobSummaryFromSpec("ocid1.scheduledjob.oc1..other", otherSpec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		}
		assertStringPtr(t, "second page", request.Page, "page-2")
		return osmanagementhubsdk.ListScheduledJobsResponse{
			ScheduledJobCollection: osmanagementhubsdk.ScheduledJobCollection{
				Items: []osmanagementhubsdk.ScheduledJobSummary{
					scheduledJobSummaryFromSpec("ocid1.scheduledjob.oc1..existing", resource.Spec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
				},
			},
		}, nil
	}
	client.get = func(_ context.Context, request osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
		assertStringPtr(t, "get scheduledJobId", request.ScheduledJobId, "ocid1.scheduledjob.oc1..existing")
		return osmanagementhubsdk.GetScheduledJobResponse{
			ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..existing", resource.Spec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
		}, nil
	}

	response, err := newTestScheduledJobClient(client).CreateOrUpdate(context.Background(), resource, testScheduledJobRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful bind without requeue", response)
	}
	if len(client.createRequests) != 0 {
		t.Fatalf("CreateScheduledJob() calls = %d, want 0 for bind", len(client.createRequests))
	}
	if len(client.listRequests) != 2 {
		t.Fatalf("ListScheduledJobs() calls = %d, want 2 pages", len(client.listRequests))
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), "ocid1.scheduledjob.oc1..existing"; got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func TestScheduledJobCreateOrUpdateDoesNotBindSummaryWithSubcompartmentDrift(t *testing.T) {
	resource := newTestScheduledJob()
	resource.Spec.ManagedInstanceIds = nil
	resource.Spec.ManagedCompartmentIds = []string{"ocid1.compartment.oc1..managed"}
	resource.Spec.IsSubcompartmentIncluded = true

	candidateSpec := resource.Spec
	candidateSpec.IsSubcompartmentIncluded = false

	client := &fakeScheduledJobOCIClient{}
	client.list = func(_ context.Context, _ osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error) {
		return osmanagementhubsdk.ListScheduledJobsResponse{
			ScheduledJobCollection: osmanagementhubsdk.ScheduledJobCollection{
				Items: []osmanagementhubsdk.ScheduledJobSummary{
					scheduledJobSummaryFromSpec("ocid1.scheduledjob.oc1..candidate", candidateSpec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
				},
			},
		}, nil
	}
	client.get = func(_ context.Context, request osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
		switch stringPointerValue(request.ScheduledJobId) {
		case "ocid1.scheduledjob.oc1..candidate":
			return osmanagementhubsdk.GetScheduledJobResponse{
				ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..candidate", candidateSpec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
			}, nil
		case "ocid1.scheduledjob.oc1..created":
			return osmanagementhubsdk.GetScheduledJobResponse{
				ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..created", resource.Spec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
			}, nil
		default:
			t.Fatalf("GetScheduledJob() scheduledJobId = %q, want candidate or created", stringPointerValue(request.ScheduledJobId))
			return osmanagementhubsdk.GetScheduledJobResponse{}, nil
		}
	}
	client.create = func(_ context.Context, request osmanagementhubsdk.CreateScheduledJobRequest) (osmanagementhubsdk.CreateScheduledJobResponse, error) {
		assertScheduledJobCreateRequest(t, request, resource.Spec)
		if request.IsSubcompartmentIncluded == nil || !*request.IsSubcompartmentIncluded {
			t.Fatalf("create isSubcompartmentIncluded = %v, want true", request.IsSubcompartmentIncluded)
		}
		return osmanagementhubsdk.CreateScheduledJobResponse{
			ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..created", resource.Spec, osmanagementhubsdk.ScheduledJobLifecycleStateCreating),
			OpcRequestId: common.String("opc-create"),
		}, nil
	}

	response, err := newTestScheduledJobClient(client).CreateOrUpdate(context.Background(), resource, testScheduledJobRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful create without requeue", response)
	}
	if len(client.createRequests) != 1 {
		t.Fatalf("CreateScheduledJob() calls = %d, want 1 when full body has isSubcompartmentIncluded drift", len(client.createRequests))
	}
	if got, want := string(resource.Status.OsokStatus.Ocid), "ocid1.scheduledjob.oc1..created"; got != want {
		t.Fatalf("status.status.ocid = %q, want %q", got, want)
	}
}

func TestScheduledJobCreateOrUpdateNoopsWithoutMutableDrift(t *testing.T) {
	resource := newTestScheduledJob()
	recordScheduledJobID(resource, "ocid1.scheduledjob.oc1..existing")
	client := &fakeScheduledJobOCIClient{}
	client.get = func(_ context.Context, request osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
		assertStringPtr(t, "get scheduledJobId", request.ScheduledJobId, "ocid1.scheduledjob.oc1..existing")
		return osmanagementhubsdk.GetScheduledJobResponse{
			ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..existing", resource.Spec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
		}, nil
	}

	response, err := newTestScheduledJobClient(client).CreateOrUpdate(context.Background(), resource, testScheduledJobRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful no-op", response)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateScheduledJob() calls = %d, want 0", len(client.updateRequests))
	}
}

//nolint:gocognit,gocyclo // The two-read mutable update sequence is the behavior under test.
func TestScheduledJobCreateOrUpdateSendsMutableUpdate(t *testing.T) {
	resource := newTestScheduledJob()
	recordScheduledJobID(resource, "ocid1.scheduledjob.oc1..existing")
	resource.Spec.DisplayName = "scheduled-job-new"
	resource.Spec.TimeNextExecution = "2026-05-02T10:00:00Z"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"ops": {"owner": "platform"}}
	resource.Spec.RetryIntervals = []int{3, 9}

	currentSpec := resource.Spec
	currentSpec.DisplayName = "scheduled-job-old"
	currentSpec.TimeNextExecution = "2026-05-01T10:00:00Z"
	currentSpec.FreeformTags = map[string]string{"env": "dev"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"ops": {"owner": "old"}}
	currentSpec.RetryIntervals = []int{2, 5}

	client := &fakeScheduledJobOCIClient{}
	getCalls := 0
	client.get = func(_ context.Context, request osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
		assertStringPtr(t, "get scheduledJobId", request.ScheduledJobId, "ocid1.scheduledjob.oc1..existing")
		getCalls++
		if getCalls == 2 {
			return osmanagementhubsdk.GetScheduledJobResponse{
				ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..existing", resource.Spec, osmanagementhubsdk.ScheduledJobLifecycleStateUpdating),
			}, nil
		}
		return osmanagementhubsdk.GetScheduledJobResponse{
			ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..existing", currentSpec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
		}, nil
	}
	client.update = func(_ context.Context, request osmanagementhubsdk.UpdateScheduledJobRequest) (osmanagementhubsdk.UpdateScheduledJobResponse, error) {
		assertScheduledJobMutableUpdateRequest(t, request, resource.Spec)
		return osmanagementhubsdk.UpdateScheduledJobResponse{
			ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..existing", resource.Spec, osmanagementhubsdk.ScheduledJobLifecycleStateUpdating),
			OpcRequestId: common.String("opc-update"),
		}, nil
	}

	response, err := newTestScheduledJobClient(client).CreateOrUpdate(context.Background(), resource, testScheduledJobRequest())
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %+v, want successful updating requeue", response)
	}
	if len(client.updateRequests) != 1 {
		t.Fatalf("UpdateScheduledJob() calls = %d, want 1", len(client.updateRequests))
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-update"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func TestScheduledJobCreateOrUpdateRejectsCreateOnlyDrift(t *testing.T) {
	resource := newTestScheduledJob()
	recordScheduledJobID(resource, "ocid1.scheduledjob.oc1..existing")
	resource.Spec.ManagedInstanceIds = []string{"ocid1.instance.oc1..new"}

	currentSpec := resource.Spec
	currentSpec.ManagedInstanceIds = []string{"ocid1.instance.oc1..old"}
	client := &fakeScheduledJobOCIClient{}
	client.get = func(_ context.Context, _ osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
		return osmanagementhubsdk.GetScheduledJobResponse{
			ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..existing", currentSpec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
		}, nil
	}

	_, err := newTestScheduledJobClient(client).CreateOrUpdate(context.Background(), resource, testScheduledJobRequest())
	if err == nil || !strings.Contains(err.Error(), "managedInstanceIds") {
		t.Fatalf("CreateOrUpdate() error = %v, want create-only drift rejection", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateScheduledJob() calls = %d, want 0 after drift rejection", len(client.updateRequests))
	}
}

func TestScheduledJobCreateOrUpdateRejectsOmittedCreateOnlyBoolReadback(t *testing.T) {
	tests := []struct {
		name       string
		mutateSpec func(*osmanagementhubv1beta1.ScheduledJobSpec)
		omitField  func(*osmanagementhubsdk.ScheduledJob)
		wantField  string
	}{
		{
			name: "isSubcompartmentIncluded",
			mutateSpec: func(spec *osmanagementhubv1beta1.ScheduledJobSpec) {
				spec.ManagedInstanceIds = nil
				spec.ManagedCompartmentIds = []string{"ocid1.compartment.oc1..managed"}
				spec.IsSubcompartmentIncluded = true
			},
			omitField: func(job *osmanagementhubsdk.ScheduledJob) {
				job.IsSubcompartmentIncluded = nil
			},
			wantField: "isSubcompartmentIncluded",
		},
		{
			name: "isManagedByAutonomousLinux",
			mutateSpec: func(spec *osmanagementhubv1beta1.ScheduledJobSpec) {
				spec.IsManagedByAutonomousLinux = true
			},
			omitField: func(job *osmanagementhubsdk.ScheduledJob) {
				job.IsManagedByAutonomousLinux = nil
			},
			wantField: "isManagedByAutonomousLinux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := newTestScheduledJob()
			tt.mutateSpec(&resource.Spec)
			recordScheduledJobID(resource, "ocid1.scheduledjob.oc1..existing")

			client := &fakeScheduledJobOCIClient{}
			client.get = func(_ context.Context, _ osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
				current := scheduledJobFromSpec("ocid1.scheduledjob.oc1..existing", resource.Spec, osmanagementhubsdk.ScheduledJobLifecycleStateActive)
				tt.omitField(&current)
				return osmanagementhubsdk.GetScheduledJobResponse{ScheduledJob: current}, nil
			}

			response, err := newTestScheduledJobClient(client).CreateOrUpdate(context.Background(), resource, testScheduledJobRequest())
			if err == nil || !strings.Contains(err.Error(), tt.wantField) {
				t.Fatalf("CreateOrUpdate() error = %v, want create-only drift rejection for %s", err, tt.wantField)
			}
			if response.IsSuccessful {
				t.Fatalf("CreateOrUpdate() response.IsSuccessful = true, want false after %s drift rejection", tt.wantField)
			}
			if len(client.updateRequests) != 0 {
				t.Fatalf("UpdateScheduledJob() calls = %d, want 0 after %s drift rejection", len(client.updateRequests), tt.wantField)
			}
		})
	}
}

func TestScheduledJobCreateOrUpdateRejectsRestrictedUnsupportedDrift(t *testing.T) {
	resource := newTestScheduledJob()
	recordScheduledJobID(resource, "ocid1.scheduledjob.oc1..existing")
	resource.Spec.DisplayName = "scheduled-job-new"

	currentSpec := resource.Spec
	currentSpec.DisplayName = "scheduled-job-old"
	client := &fakeScheduledJobOCIClient{}
	client.get = func(_ context.Context, _ osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
		current := scheduledJobFromSpec("ocid1.scheduledjob.oc1..existing", currentSpec, osmanagementhubsdk.ScheduledJobLifecycleStateActive)
		current.IsRestricted = common.Bool(true)
		return osmanagementhubsdk.GetScheduledJobResponse{ScheduledJob: current}, nil
	}

	_, err := newTestScheduledJobClient(client).CreateOrUpdate(context.Background(), resource, testScheduledJobRequest())
	if err == nil || !strings.Contains(err.Error(), "restricted update") {
		t.Fatalf("CreateOrUpdate() error = %v, want restricted update rejection", err)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("UpdateScheduledJob() calls = %d, want 0 after restricted rejection", len(client.updateRequests))
	}
}

//nolint:gocognit,gocyclo // The delete confirmation sequence keeps pre-read, delete, and readback assertions together.
func TestScheduledJobDeleteRetainsFinalizerWhileDeleting(t *testing.T) {
	resource := newTestScheduledJob()
	recordScheduledJobID(resource, "ocid1.scheduledjob.oc1..existing")
	client := &fakeScheduledJobOCIClient{}
	getCalls := 0
	client.get = func(_ context.Context, request osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
		assertStringPtr(t, "get scheduledJobId", request.ScheduledJobId, "ocid1.scheduledjob.oc1..existing")
		getCalls++
		state := osmanagementhubsdk.ScheduledJobLifecycleStateActive
		if getCalls == 3 {
			state = osmanagementhubsdk.ScheduledJobLifecycleStateDeleting
		}
		return osmanagementhubsdk.GetScheduledJobResponse{
			ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..existing", resource.Spec, state),
		}, nil
	}
	client.delete = func(_ context.Context, request osmanagementhubsdk.DeleteScheduledJobRequest) (osmanagementhubsdk.DeleteScheduledJobResponse, error) {
		assertStringPtr(t, "delete scheduledJobId", request.ScheduledJobId, "ocid1.scheduledjob.oc1..existing")
		return osmanagementhubsdk.DeleteScheduledJobResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := newTestScheduledJobClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want pending delete confirmation")
	}
	if got, want := resource.Status.LifecycleState, string(osmanagementhubsdk.ScheduledJobLifecycleStateDeleting); got != want {
		t.Fatalf("status.lifecycleState = %q, want %q", got, want)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want delete lifecycle tracker")
	}
	if got, want := resource.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseDelete; got != want {
		t.Fatalf("status.status.async.current.phase = %q, want %q", got, want)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil while delete is pending", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestScheduledJobDeleteConfirmsNotFoundAfterDelete(t *testing.T) {
	resource := newTestScheduledJob()
	recordScheduledJobID(resource, "ocid1.scheduledjob.oc1..existing")
	client := &fakeScheduledJobOCIClient{}
	getCalls := 0
	client.get = func(_ context.Context, _ osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
		getCalls++
		if getCalls == 3 {
			return osmanagementhubsdk.GetScheduledJobResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
		}
		return osmanagementhubsdk.GetScheduledJobResponse{
			ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..existing", resource.Spec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
		}, nil
	}
	client.delete = func(_ context.Context, _ osmanagementhubsdk.DeleteScheduledJobRequest) (osmanagementhubsdk.DeleteScheduledJobResponse, error) {
		return osmanagementhubsdk.DeleteScheduledJobResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := newTestScheduledJobClient(client).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous not-found confirmation")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete completion timestamp")
	}
}

func TestScheduledJobDeleteRetainsFinalizerOnAuthShapedConfirmRead(t *testing.T) {
	resource := newTestScheduledJob()
	recordScheduledJobID(resource, "ocid1.scheduledjob.oc1..existing")
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	authErr.OpcRequestID = "opc-auth-confirm"

	client := &fakeScheduledJobOCIClient{}
	getCalls := 0
	client.get = func(_ context.Context, _ osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
		getCalls++
		if getCalls == 3 {
			return osmanagementhubsdk.GetScheduledJobResponse{}, authErr
		}
		return osmanagementhubsdk.GetScheduledJobResponse{
			ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..existing", resource.Spec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
		}, nil
	}
	client.delete = func(_ context.Context, _ osmanagementhubsdk.DeleteScheduledJobRequest) (osmanagementhubsdk.DeleteScheduledJobResponse, error) {
		return osmanagementhubsdk.DeleteScheduledJobResponse{OpcRequestId: common.String("opc-delete")}, nil
	}

	deleted, err := newTestScheduledJobClient(client).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped confirm-read error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped confirm read")
	}
	if len(client.deleteRequests) != 1 {
		t.Fatalf("DeleteScheduledJob() calls = %d, want 1 before auth-shaped confirm read", len(client.deleteRequests))
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-auth-confirm"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil after auth-shaped confirm read", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestScheduledJobDeleteRetainsFinalizerOnNoTrackedIDAuthShapedListConfirm(t *testing.T) {
	resource := newTestScheduledJob()
	authErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	authErr.OpcRequestID = "opc-auth-list"

	client := &fakeScheduledJobOCIClient{}
	listCalls := 0
	client.list = func(_ context.Context, _ osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error) {
		listCalls++
		if listCalls == 2 {
			return osmanagementhubsdk.ListScheduledJobsResponse{}, authErr
		}
		return osmanagementhubsdk.ListScheduledJobsResponse{
			ScheduledJobCollection: osmanagementhubsdk.ScheduledJobCollection{
				Items: []osmanagementhubsdk.ScheduledJobSummary{
					scheduledJobSummaryFromSpec("ocid1.scheduledjob.oc1..existing", resource.Spec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
				},
			},
		}, nil
	}
	client.get = func(_ context.Context, request osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
		assertStringPtr(t, "pre-delete get scheduledJobId", request.ScheduledJobId, "ocid1.scheduledjob.oc1..existing")
		return osmanagementhubsdk.GetScheduledJobResponse{
			ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..existing", resource.Spec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
		}, nil
	}
	client.delete = func(context.Context, osmanagementhubsdk.DeleteScheduledJobRequest) (osmanagementhubsdk.DeleteScheduledJobResponse, error) {
		t.Fatal("DeleteScheduledJob() called after auth-shaped no-tracked-ID list confirmation")
		return osmanagementhubsdk.DeleteScheduledJobResponse{}, nil
	}

	deleted, err := newTestScheduledJobClient(client).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want conservative auth-shaped list-confirmation error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped list confirmation")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteScheduledJob() calls = %d, want 0 after auth-shaped list confirmation", len(client.deleteRequests))
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-auth-list"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil after auth-shaped list confirmation", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestScheduledJobDeleteDoesNotDeleteSummaryWithSubcompartmentDrift(t *testing.T) {
	resource := newTestScheduledJob()
	resource.Spec.ManagedInstanceIds = nil
	resource.Spec.ManagedCompartmentIds = []string{"ocid1.compartment.oc1..managed"}
	resource.Spec.IsSubcompartmentIncluded = true

	candidateSpec := resource.Spec
	candidateSpec.IsSubcompartmentIncluded = false

	client := &fakeScheduledJobOCIClient{}
	client.list = func(_ context.Context, _ osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error) {
		return osmanagementhubsdk.ListScheduledJobsResponse{
			ScheduledJobCollection: osmanagementhubsdk.ScheduledJobCollection{
				Items: []osmanagementhubsdk.ScheduledJobSummary{
					scheduledJobSummaryFromSpec("ocid1.scheduledjob.oc1..candidate", candidateSpec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
				},
			},
		}, nil
	}
	client.get = func(_ context.Context, request osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
		assertStringPtr(t, "get scheduledJobId", request.ScheduledJobId, "ocid1.scheduledjob.oc1..candidate")
		return osmanagementhubsdk.GetScheduledJobResponse{
			ScheduledJob: scheduledJobFromSpec("ocid1.scheduledjob.oc1..candidate", candidateSpec, osmanagementhubsdk.ScheduledJobLifecycleStateActive),
		}, nil
	}
	client.delete = func(context.Context, osmanagementhubsdk.DeleteScheduledJobRequest) (osmanagementhubsdk.DeleteScheduledJobResponse, error) {
		t.Fatal("DeleteScheduledJob() called for summary candidate with isSubcompartmentIncluded drift")
		return osmanagementhubsdk.DeleteScheduledJobResponse{}, nil
	}

	deleted, err := newTestScheduledJobClient(client).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "delete confirmation did not find") {
		t.Fatalf("Delete() error = %v, want no verified full-body match", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained when candidate full body does not match")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteScheduledJob() calls = %d, want 0 for mismatched summary candidate", len(client.deleteRequests))
	}
	if len(client.getRequests) == 0 {
		t.Fatal("GetScheduledJob() calls = 0, want full-body candidate verification")
	}
}

func TestScheduledJobDeleteRejectsAuthShapedPreRead(t *testing.T) {
	resource := newTestScheduledJob()
	recordScheduledJobID(resource, "ocid1.scheduledjob.oc1..existing")
	client := &fakeScheduledJobOCIClient{}
	client.get = func(_ context.Context, _ osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
		return osmanagementhubsdk.GetScheduledJobResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	}

	deleted, err := newTestScheduledJobClient(client).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want auth-shaped 404 rejection", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-read")
	}
	if len(client.deleteRequests) != 0 {
		t.Fatalf("DeleteScheduledJob() calls = %d, want 0 after auth-shaped pre-read", len(client.deleteRequests))
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatalf("status.status.deletedAt = %v, want nil after auth-shaped pre-read", resource.Status.OsokStatus.DeletedAt)
	}
}

func TestScheduledJobCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	resource := newTestScheduledJob()
	client := &fakeScheduledJobOCIClient{}
	client.list = func(_ context.Context, _ osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error) {
		return osmanagementhubsdk.ListScheduledJobsResponse{}, nil
	}
	client.create = func(_ context.Context, _ osmanagementhubsdk.CreateScheduledJobRequest) (osmanagementhubsdk.CreateScheduledJobResponse, error) {
		return osmanagementhubsdk.CreateScheduledJobResponse{}, errortest.NewServiceError(500, "InternalError", "create failed")
	}

	response, err := newTestScheduledJobClient(client).CreateOrUpdate(context.Background(), resource, testScheduledJobRequest())
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI create failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response.IsSuccessful = true, want false")
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-request-id"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
	if !strings.Contains(resource.Status.OsokStatus.Message, "create failed") {
		t.Fatalf("status.status.message = %q, want OCI error message", resource.Status.OsokStatus.Message)
	}
}

func assertScheduledJobCreateRequest(
	t *testing.T,
	request osmanagementhubsdk.CreateScheduledJobRequest,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) {
	t.Helper()
	assertStringPtr(t, "create compartmentId", request.CompartmentId, spec.CompartmentId)
	assertStringPtr(t, "create displayName", request.DisplayName, spec.DisplayName)
	assertStringPtr(t, "create description", request.Description, spec.Description)
	if request.ScheduleType != osmanagementhubsdk.ScheduleTypesOnetime {
		t.Fatalf("create scheduleType = %q, want ONETIME", request.ScheduleType)
	}
	assertSDKTimePtr(t, "create timeNextExecution", request.TimeNextExecution, spec.TimeNextExecution)
	if len(request.Operations) != 1 || request.Operations[0].OperationType != osmanagementhubsdk.OperationTypesUpdateAll {
		t.Fatalf("create operations = %#v, want UPDATE_ALL", request.Operations)
	}
	if !reflect.DeepEqual(request.ManagedInstanceIds, spec.ManagedInstanceIds) {
		t.Fatalf("create managedInstanceIds = %#v, want %#v", request.ManagedInstanceIds, spec.ManagedInstanceIds)
	}
	if request.OpcRetryToken == nil || strings.TrimSpace(*request.OpcRetryToken) == "" {
		t.Fatal("create opcRetryToken = empty, want deterministic retry token")
	}
	if !reflect.DeepEqual(request.FreeformTags, spec.FreeformTags) {
		t.Fatalf("create freeformTags = %#v, want %#v", request.FreeformTags, spec.FreeformTags)
	}
	wantDefinedTags := map[string]map[string]interface{}{"ops": {"owner": "old"}}
	if !reflect.DeepEqual(request.DefinedTags, wantDefinedTags) {
		t.Fatalf("create definedTags = %#v, want %#v", request.DefinedTags, wantDefinedTags)
	}
	if !reflect.DeepEqual(request.RetryIntervals, spec.RetryIntervals) {
		t.Fatalf("create retryIntervals = %#v, want %#v", request.RetryIntervals, spec.RetryIntervals)
	}
}

func assertScheduledJobMutableUpdateRequest(
	t *testing.T,
	request osmanagementhubsdk.UpdateScheduledJobRequest,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) {
	t.Helper()
	assertStringPtr(t, "update scheduledJobId", request.ScheduledJobId, "ocid1.scheduledjob.oc1..existing")
	assertStringPtr(t, "update displayName", request.DisplayName, "scheduled-job-new")
	assertSDKTimePtr(t, "update timeNextExecution", request.TimeNextExecution, spec.TimeNextExecution)
	if !reflect.DeepEqual(request.FreeformTags, spec.FreeformTags) {
		t.Fatalf("update freeformTags = %#v, want %#v", request.FreeformTags, spec.FreeformTags)
	}
	wantDefinedTags := map[string]map[string]interface{}{"ops": {"owner": "platform"}}
	if !reflect.DeepEqual(request.DefinedTags, wantDefinedTags) {
		t.Fatalf("update definedTags = %#v, want %#v", request.DefinedTags, wantDefinedTags)
	}
	if !reflect.DeepEqual(request.RetryIntervals, spec.RetryIntervals) {
		t.Fatalf("update retryIntervals = %#v, want %#v", request.RetryIntervals, spec.RetryIntervals)
	}
}

func assertScheduledJobCreatedStatus(t *testing.T, resource *osmanagementhubv1beta1.ScheduledJob, id string) {
	t.Helper()
	if got := string(resource.Status.OsokStatus.Ocid); got != id {
		t.Fatalf("status.status.ocid = %q, want %q", got, id)
	}
	if got := resource.Status.Id; got != id {
		t.Fatalf("status.id = %q, want %q", got, id)
	}
	if got, want := resource.Status.LifecycleState, string(osmanagementhubsdk.ScheduledJobLifecycleStateActive); got != want {
		t.Fatalf("status.lifecycleState = %q, want %q", got, want)
	}
	if got, want := resource.Status.OsokStatus.OpcRequestID, "opc-create"; got != want {
		t.Fatalf("status.status.opcRequestId = %q, want %q", got, want)
	}
}

func newTestScheduledJobClient(client *fakeScheduledJobOCIClient) ScheduledJobServiceClient {
	return newScheduledJobServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
}

func newTestScheduledJob() *osmanagementhubv1beta1.ScheduledJob {
	return &osmanagementhubv1beta1.ScheduledJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scheduled-job",
			Namespace: "default",
			UID:       types.UID("scheduled-job-uid"),
		},
		Spec: osmanagementhubv1beta1.ScheduledJobSpec{
			CompartmentId:     "ocid1.compartment.oc1..example",
			DisplayName:       "scheduled-job",
			Description:       "patch window",
			ScheduleType:      string(osmanagementhubsdk.ScheduleTypesOnetime),
			TimeNextExecution: "2026-05-01T10:00:00Z",
			Operations: []osmanagementhubv1beta1.ScheduledJobOperation{
				{OperationType: string(osmanagementhubsdk.OperationTypesUpdateAll)},
			},
			ManagedInstanceIds: []string{"ocid1.instance.oc1..example"},
			FreeformTags:       map[string]string{"env": "dev"},
			DefinedTags:        map[string]shared.MapValue{"ops": {"owner": "old"}},
			RetryIntervals:     []int{2, 5},
		},
	}
}

func testScheduledJobRequest() ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "scheduled-job"}}
}

func recordScheduledJobID(resource *osmanagementhubv1beta1.ScheduledJob, id string) {
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	resource.Status.Id = id
}

func scheduledJobFromSpec(
	id string,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
	state osmanagementhubsdk.ScheduledJobLifecycleStateEnum,
) osmanagementhubsdk.ScheduledJob {
	timeNextExecution, err := scheduledJobTimeFromSpec("timeNextExecution", spec.TimeNextExecution)
	if err != nil {
		panic(err)
	}
	operations, err := scheduledJobOperationsFromSpec(spec.Operations)
	if err != nil {
		panic(err)
	}
	locations, err := scheduledJobLocationsFromSpec(spec.Locations)
	if err != nil {
		panic(err)
	}
	scheduleType, _ := osmanagementhubsdk.GetMappingScheduleTypesEnum(spec.ScheduleType)
	job := osmanagementhubsdk.ScheduledJob{
		Id:                         common.String(id),
		DisplayName:                common.String(spec.DisplayName),
		CompartmentId:              common.String(spec.CompartmentId),
		ScheduleType:               scheduleType,
		TimeNextExecution:          timeNextExecution,
		Operations:                 operations,
		TimeCreated:                sdkTime("2026-04-01T10:00:00Z"),
		TimeUpdated:                sdkTime("2026-04-01T10:00:00Z"),
		LifecycleState:             state,
		FreeformTags:               spec.FreeformTags,
		DefinedTags:                scheduledJobDefinedTags(spec.DefinedTags),
		Description:                common.String(spec.Description),
		Locations:                  locations,
		RecurringRule:              common.String(spec.RecurringRule),
		ManagedInstanceIds:         spec.ManagedInstanceIds,
		ManagedInstanceGroupIds:    spec.ManagedInstanceGroupIds,
		ManagedCompartmentIds:      spec.ManagedCompartmentIds,
		LifecycleStageIds:          spec.LifecycleStageIds,
		IsManagedByAutonomousLinux: common.Bool(spec.IsManagedByAutonomousLinux),
		RetryIntervals:             spec.RetryIntervals,
	}
	if spec.IsSubcompartmentIncluded {
		job.IsSubcompartmentIncluded = common.Bool(true)
	}
	if strings.TrimSpace(spec.WorkRequestId) != "" {
		job.WorkRequestId = common.String(spec.WorkRequestId)
	}
	return job
}

func scheduledJobSummaryFromSpec(
	id string,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
	state osmanagementhubsdk.ScheduledJobLifecycleStateEnum,
) osmanagementhubsdk.ScheduledJobSummary {
	job := scheduledJobFromSpec(id, spec, state)
	return osmanagementhubsdk.ScheduledJobSummary{
		Id:                         job.Id,
		DisplayName:                job.DisplayName,
		CompartmentId:              job.CompartmentId,
		ScheduleType:               job.ScheduleType,
		TimeCreated:                job.TimeCreated,
		TimeUpdated:                job.TimeUpdated,
		TimeNextExecution:          job.TimeNextExecution,
		Operations:                 job.Operations,
		LifecycleState:             job.LifecycleState,
		FreeformTags:               job.FreeformTags,
		DefinedTags:                job.DefinedTags,
		Locations:                  job.Locations,
		ManagedInstanceIds:         job.ManagedInstanceIds,
		ManagedInstanceGroupIds:    job.ManagedInstanceGroupIds,
		ManagedCompartmentIds:      job.ManagedCompartmentIds,
		LifecycleStageIds:          job.LifecycleStageIds,
		IsManagedByAutonomousLinux: job.IsManagedByAutonomousLinux,
		RetryIntervals:             job.RetryIntervals,
		WorkRequestId:              job.WorkRequestId,
	}
}

func sdkTime(value string) *common.SDKTime {
	result, err := scheduledJobTimeFromSpec("test", value)
	if err != nil {
		panic(err)
	}
	return result
}

func assertStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func assertSDKTimePtr(t *testing.T, name string, got *common.SDKTime, want string) {
	t.Helper()
	wantTime, err := scheduledJobTimeFromSpec(name, want)
	if err != nil {
		t.Fatalf("parse want time: %v", err)
	}
	if !scheduledJobTimesEqual(got, wantTime) {
		t.Fatalf("%s = %v, want %v", name, got, wantTime)
	}
}

func containsScheduledJobString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
