/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package schedule

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	resourceschedulersdk "github.com/oracle/oci-go-sdk/v65/resourcescheduler"
	resourceschedulerv1beta1 "github.com/oracle/oci-service-operator/api/resourcescheduler/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testScheduleCompartmentID = "ocid1.compartment.oc1..schedulecompartment"
	testScheduleID            = "ocid1.resourceschedulerschedule.oc1..schedule"
	testScheduleOtherID       = "ocid1.resourceschedulerschedule.oc1..other"
	testScheduleDisplayName   = "runtime-schedule"
	testScheduleResourceID    = "ocid1.instance.oc1..scheduled"
)

type fakeScheduleOCIClient struct {
	createFunc func(context.Context, resourceschedulersdk.CreateScheduleRequest) (resourceschedulersdk.CreateScheduleResponse, error)
	getFunc    func(context.Context, resourceschedulersdk.GetScheduleRequest) (resourceschedulersdk.GetScheduleResponse, error)
	listFunc   func(context.Context, resourceschedulersdk.ListSchedulesRequest) (resourceschedulersdk.ListSchedulesResponse, error)
	updateFunc func(context.Context, resourceschedulersdk.UpdateScheduleRequest) (resourceschedulersdk.UpdateScheduleResponse, error)
	deleteFunc func(context.Context, resourceschedulersdk.DeleteScheduleRequest) (resourceschedulersdk.DeleteScheduleResponse, error)

	createRequests []resourceschedulersdk.CreateScheduleRequest
	getRequests    []resourceschedulersdk.GetScheduleRequest
	listRequests   []resourceschedulersdk.ListSchedulesRequest
	updateRequests []resourceschedulersdk.UpdateScheduleRequest
	deleteRequests []resourceschedulersdk.DeleteScheduleRequest
}

func (f *fakeScheduleOCIClient) CreateSchedule(
	ctx context.Context,
	request resourceschedulersdk.CreateScheduleRequest,
) (resourceschedulersdk.CreateScheduleResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.createFunc != nil {
		return f.createFunc(ctx, request)
	}
	return resourceschedulersdk.CreateScheduleResponse{}, nil
}

func (f *fakeScheduleOCIClient) GetSchedule(
	ctx context.Context,
	request resourceschedulersdk.GetScheduleRequest,
) (resourceschedulersdk.GetScheduleResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.getFunc != nil {
		return f.getFunc(ctx, request)
	}
	return resourceschedulersdk.GetScheduleResponse{}, nil
}

func (f *fakeScheduleOCIClient) ListSchedules(
	ctx context.Context,
	request resourceschedulersdk.ListSchedulesRequest,
) (resourceschedulersdk.ListSchedulesResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.listFunc != nil {
		return f.listFunc(ctx, request)
	}
	return resourceschedulersdk.ListSchedulesResponse{}, nil
}

func (f *fakeScheduleOCIClient) UpdateSchedule(
	ctx context.Context,
	request resourceschedulersdk.UpdateScheduleRequest,
) (resourceschedulersdk.UpdateScheduleResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.updateFunc != nil {
		return f.updateFunc(ctx, request)
	}
	return resourceschedulersdk.UpdateScheduleResponse{}, nil
}

func (f *fakeScheduleOCIClient) DeleteSchedule(
	ctx context.Context,
	request resourceschedulersdk.DeleteScheduleRequest,
) (resourceschedulersdk.DeleteScheduleResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.deleteFunc != nil {
		return f.deleteFunc(ctx, request)
	}
	return resourceschedulersdk.DeleteScheduleResponse{}, nil
}

func TestScheduleRuntimeHooksConfigured(t *testing.T) {
	t.Parallel()

	hooks := newScheduleDefaultRuntimeHooks(resourceschedulersdk.ScheduleClient{})
	applyScheduleRuntimeHooks(&hooks)

	assertScheduleRuntimeSemantics(t, hooks.Semantics)
	assertScheduleRuntimeHookPresence(t, hooks)
}

func assertScheduleRuntimeSemantics(t *testing.T, semantics *generatedruntime.Semantics) {
	t.Helper()

	if semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed semantics")
	}
	if semantics.Async == nil || semantics.Async.Strategy != "lifecycle" || semantics.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async = %#v, want lifecycle generatedruntime semantics", semantics.Async)
	}
	if got := semantics.FinalizerPolicy; got != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got)
	}
	if semantics.List == nil || !reflect.DeepEqual(semantics.List.MatchFields, []string{"compartmentId", "displayName", "id"}) {
		t.Fatalf("List semantics = %#v, want compartment/displayName/id matching", semantics.List)
	}
}

func assertScheduleRuntimeHookPresence(t *testing.T, hooks ScheduleRuntimeHooks) {
	t.Helper()

	if hooks.BuildCreateBody == nil {
		t.Fatal("BuildCreateBody = nil, want resource-local create shaping")
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("BuildUpdateBody = nil, want resource-local update shaping")
	}
	if hooks.Identity.GuardExistingBeforeCreate == nil {
		t.Fatal("Identity.GuardExistingBeforeCreate = nil, want bounded pre-create guard")
	}
	if hooks.DeleteHooks.HandleError == nil {
		t.Fatal("DeleteHooks.HandleError = nil, want conservative delete error handling")
	}
	if len(hooks.WrapGeneratedClient) == 0 {
		t.Fatal("WrapGeneratedClient is empty, want pre-delete confirmation wrapper")
	}
}

func TestScheduleServiceClientCreateOrUpdateCreatesScheduleAndRefreshesStatus(t *testing.T) {
	t.Parallel()

	resource := newScheduleRuntimeTestResource()
	fake := &fakeScheduleOCIClient{}
	fake.listFunc = func(context.Context, resourceschedulersdk.ListSchedulesRequest) (resourceschedulersdk.ListSchedulesResponse, error) {
		return resourceschedulersdk.ListSchedulesResponse{}, nil
	}
	fake.createFunc = func(_ context.Context, request resourceschedulersdk.CreateScheduleRequest) (resourceschedulersdk.CreateScheduleResponse, error) {
		assertScheduleCreateRequest(t, request, resource)
		return resourceschedulersdk.CreateScheduleResponse{
			OpcRequestId:     common.String("opc-create-1"),
			OpcWorkRequestId: common.String("wr-create-1"),
			Schedule:         scheduleFromSpec(testScheduleID, resource.Spec, resourceschedulersdk.ScheduleLifecycleStateCreating),
		}, nil
	}
	fake.getFunc = func(_ context.Context, request resourceschedulersdk.GetScheduleRequest) (resourceschedulersdk.GetScheduleResponse, error) {
		if got := scheduleStringValue(request.ScheduleId); got != testScheduleID {
			t.Fatalf("get scheduleId = %q, want %q", got, testScheduleID)
		}
		return resourceschedulersdk.GetScheduleResponse{
			Schedule: scheduleFromSpec(testScheduleID, resource.Spec, resourceschedulersdk.ScheduleLifecycleStateActive),
		}, nil
	}
	client := newScheduleRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, scheduleReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	assertScheduleCreateOrUpdateSucceeded(t, response)
	assertScheduleCreateCallCounts(t, fake)
	assertScheduleCreatedStatus(t, resource)
}

func assertScheduleCreateOrUpdateSucceeded(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()

	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue after ACTIVE follow-up")
	}
}

func assertScheduleCreateCallCounts(t *testing.T, fake *fakeScheduleOCIClient) {
	t.Helper()

	if len(fake.listRequests) != 1 {
		t.Fatalf("ListSchedules() calls = %d, want 1 pre-create lookup", len(fake.listRequests))
	}
	if len(fake.createRequests) != 1 {
		t.Fatalf("CreateSchedule() calls = %d, want 1", len(fake.createRequests))
	}
}

func assertScheduleCreatedStatus(t *testing.T, resource *resourceschedulerv1beta1.Schedule) {
	t.Helper()

	if got := resource.Status.Id; got != testScheduleID {
		t.Fatalf("status.id = %q, want %q", got, testScheduleID)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testScheduleID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testScheduleID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-1", got)
	}
	if got := resource.Status.LifecycleState; got != string(resourceschedulersdk.ScheduleLifecycleStateActive) {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", got)
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.status.async.current = %#v, want nil after ACTIVE follow-up", resource.Status.OsokStatus.Async.Current)
	}
	if len(resource.Status.ResourceFilters) != 1 || strings.TrimSpace(resource.Status.ResourceFilters[0].JsonData) == "" {
		t.Fatalf("status.resourceFilters = %#v, want projected jsonData for polymorphic filter", resource.Status.ResourceFilters)
	}
}

func TestScheduleServiceClientCreateOrUpdateBindsExistingScheduleFromLaterListPage(t *testing.T) {
	t.Parallel()

	resource := newScheduleRuntimeTestResource()
	fake := &fakeScheduleOCIClient{}
	fake.listFunc = schedulePaginatedBindListFunc(t, fake, resource)
	fake.getFunc = func(_ context.Context, request resourceschedulersdk.GetScheduleRequest) (resourceschedulersdk.GetScheduleResponse, error) {
		if got := scheduleStringValue(request.ScheduleId); got != testScheduleID {
			t.Fatalf("get scheduleId = %q, want %q", got, testScheduleID)
		}
		return resourceschedulersdk.GetScheduleResponse{
			Schedule: scheduleFromSpec(testScheduleID, resource.Spec, resourceschedulersdk.ScheduleLifecycleStateActive),
		}, nil
	}
	fake.createFunc = func(context.Context, resourceschedulersdk.CreateScheduleRequest) (resourceschedulersdk.CreateScheduleResponse, error) {
		t.Fatal("CreateSchedule() called; want existing schedule bind")
		return resourceschedulersdk.CreateScheduleResponse{}, nil
	}
	client := newScheduleRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, scheduleReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.listRequests) != 2 {
		t.Fatalf("ListSchedules() calls = %d, want 2 paginated calls", len(fake.listRequests))
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("CreateSchedule() calls = %d, want 0", len(fake.createRequests))
	}
	if got := resource.Status.Id; got != testScheduleID {
		t.Fatalf("status.id = %q, want bound schedule ID", got)
	}
}

func schedulePaginatedBindListFunc(
	t *testing.T,
	fake *fakeScheduleOCIClient,
	resource *resourceschedulerv1beta1.Schedule,
) func(context.Context, resourceschedulersdk.ListSchedulesRequest) (resourceschedulersdk.ListSchedulesResponse, error) {
	t.Helper()

	return func(_ context.Context, request resourceschedulersdk.ListSchedulesRequest) (resourceschedulersdk.ListSchedulesResponse, error) {
		assertScheduleBindListRequest(t, request)
		return schedulePaginatedBindListResponse(t, len(fake.listRequests), request, resource)
	}
}

func assertScheduleBindListRequest(t *testing.T, request resourceschedulersdk.ListSchedulesRequest) {
	t.Helper()

	if got := scheduleStringValue(request.CompartmentId); got != testScheduleCompartmentID {
		t.Fatalf("list compartmentId = %q, want %q", got, testScheduleCompartmentID)
	}
	if got := scheduleStringValue(request.DisplayName); got != testScheduleDisplayName {
		t.Fatalf("list displayName = %q, want %q", got, testScheduleDisplayName)
	}
}

func schedulePaginatedBindListResponse(
	t *testing.T,
	callCount int,
	request resourceschedulersdk.ListSchedulesRequest,
	resource *resourceschedulerv1beta1.Schedule,
) (resourceschedulersdk.ListSchedulesResponse, error) {
	t.Helper()

	switch callCount {
	case 1:
		return firstScheduleBindListPage(t, request), nil
	case 2:
		return secondScheduleBindListPage(t, request, resource), nil
	default:
		t.Fatalf("unexpected ListSchedules() call %d", callCount)
		return resourceschedulersdk.ListSchedulesResponse{}, nil
	}
}

func firstScheduleBindListPage(
	t *testing.T,
	request resourceschedulersdk.ListSchedulesRequest,
) resourceschedulersdk.ListSchedulesResponse {
	t.Helper()

	if request.Page != nil {
		t.Fatalf("first list page = %q, want nil", scheduleStringValue(request.Page))
	}
	otherSpec := newScheduleRuntimeTestResource().Spec
	otherSpec.DisplayName = "other-schedule"
	return resourceschedulersdk.ListSchedulesResponse{
		ScheduleCollection: resourceschedulersdk.ScheduleCollection{
			Items: []resourceschedulersdk.ScheduleSummary{
				scheduleSummaryFromSpec(testScheduleOtherID, otherSpec, resourceschedulersdk.ScheduleLifecycleStateActive),
			},
		},
		OpcNextPage: common.String("page-2"),
	}
}

func secondScheduleBindListPage(
	t *testing.T,
	request resourceschedulersdk.ListSchedulesRequest,
	resource *resourceschedulerv1beta1.Schedule,
) resourceschedulersdk.ListSchedulesResponse {
	t.Helper()

	if got := scheduleStringValue(request.Page); got != "page-2" {
		t.Fatalf("second list page = %q, want page-2", got)
	}
	return resourceschedulersdk.ListSchedulesResponse{
		ScheduleCollection: resourceschedulersdk.ScheduleCollection{
			Items: []resourceschedulersdk.ScheduleSummary{
				scheduleSummaryFromSpec(testScheduleID, resource.Spec, resourceschedulersdk.ScheduleLifecycleStateActive),
			},
		},
	}
}

func TestScheduleServiceClientCreateOrUpdateSkipsUpdateWhenObservedStateMatches(t *testing.T) {
	t.Parallel()

	resource := newScheduleRuntimeTestResource()
	trackScheduleID(resource, testScheduleID)
	fake := &fakeScheduleOCIClient{}
	fake.getFunc = func(context.Context, resourceschedulersdk.GetScheduleRequest) (resourceschedulersdk.GetScheduleResponse, error) {
		return resourceschedulersdk.GetScheduleResponse{
			Schedule: scheduleFromSpec(testScheduleID, resource.Spec, resourceschedulersdk.ScheduleLifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, resourceschedulersdk.UpdateScheduleRequest) (resourceschedulersdk.UpdateScheduleResponse, error) {
		t.Fatal("UpdateSchedule() called; want no-op reconcile")
		return resourceschedulersdk.UpdateScheduleResponse{}, nil
	}
	client := newScheduleRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, scheduleReconcileRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateSchedule() calls = %d, want 0", len(fake.updateRequests))
	}
}

func TestScheduleServiceClientCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	desired := newScheduleRuntimeTestResource()
	currentSpec := desired.Spec
	currentSpec.DisplayName = "old-schedule"
	currentSpec.Description = "old description"
	currentSpec.Action = string(resourceschedulersdk.ScheduleActionStopResource)
	currentSpec.RecurrenceDetails = "FREQ=WEEKLY;INTERVAL=1"
	currentSpec.FreeformTags = map[string]string{"env": "old"}

	fake := &fakeScheduleOCIClient{}
	fake.getFunc = scheduleMutableUpdateGetFunc(t, fake, desired, currentSpec)
	fake.updateFunc = func(_ context.Context, request resourceschedulersdk.UpdateScheduleRequest) (resourceschedulersdk.UpdateScheduleResponse, error) {
		assertScheduleUpdateRequest(t, request, desired)
		return resourceschedulersdk.UpdateScheduleResponse{
			OpcRequestId:     common.String("opc-update-1"),
			OpcWorkRequestId: common.String("wr-update-1"),
		}, nil
	}
	client := newScheduleRuntimeTestClient(fake)

	trackScheduleID(desired, testScheduleID)
	response, err := client.CreateOrUpdate(context.Background(), desired, scheduleReconcileRequest(desired))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("UpdateSchedule() calls = %d, want 1", len(fake.updateRequests))
	}
	if got := desired.Status.OsokStatus.OpcRequestID; got != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", got)
	}
}

func scheduleMutableUpdateGetFunc(
	t *testing.T,
	fake *fakeScheduleOCIClient,
	desired *resourceschedulerv1beta1.Schedule,
	currentSpec resourceschedulerv1beta1.ScheduleSpec,
) func(context.Context, resourceschedulersdk.GetScheduleRequest) (resourceschedulersdk.GetScheduleResponse, error) {
	t.Helper()

	return func(_ context.Context, request resourceschedulersdk.GetScheduleRequest) (resourceschedulersdk.GetScheduleResponse, error) {
		assertScheduleGetRequest(t, request)
		return scheduleMutableUpdateGetResponse(t, len(fake.getRequests), desired, currentSpec)
	}
}

func assertScheduleGetRequest(t *testing.T, request resourceschedulersdk.GetScheduleRequest) {
	t.Helper()

	if got := scheduleStringValue(request.ScheduleId); got != testScheduleID {
		t.Fatalf("get scheduleId = %q, want %q", got, testScheduleID)
	}
}

func scheduleMutableUpdateGetResponse(
	t *testing.T,
	callCount int,
	desired *resourceschedulerv1beta1.Schedule,
	currentSpec resourceschedulerv1beta1.ScheduleSpec,
) (resourceschedulersdk.GetScheduleResponse, error) {
	t.Helper()

	switch callCount {
	case 1:
		return resourceschedulersdk.GetScheduleResponse{
			Schedule: scheduleFromSpec(testScheduleID, currentSpec, resourceschedulersdk.ScheduleLifecycleStateActive),
		}, nil
	case 2:
		return resourceschedulersdk.GetScheduleResponse{
			Schedule: scheduleFromSpec(testScheduleID, desired.Spec, resourceschedulersdk.ScheduleLifecycleStateActive),
		}, nil
	default:
		t.Fatalf("unexpected GetSchedule() call %d", callCount)
		return resourceschedulersdk.GetScheduleResponse{}, nil
	}
}

func assertScheduleUpdateRequest(
	t *testing.T,
	request resourceschedulersdk.UpdateScheduleRequest,
	desired *resourceschedulerv1beta1.Schedule,
) {
	t.Helper()

	if got := scheduleStringValue(request.ScheduleId); got != testScheduleID {
		t.Fatalf("update scheduleId = %q, want %q", got, testScheduleID)
	}
	if got := scheduleStringValue(request.DisplayName); got != testScheduleDisplayName {
		t.Fatalf("update displayName = %q, want %q", got, testScheduleDisplayName)
	}
	if got := scheduleStringValue(request.Description); got != desired.Spec.Description {
		t.Fatalf("update description = %q, want desired description", got)
	}
	if got := request.Action; got != resourceschedulersdk.UpdateScheduleDetailsActionStartResource {
		t.Fatalf("update action = %q, want START_RESOURCE", got)
	}
	if got := scheduleStringValue(request.RecurrenceDetails); got != desired.Spec.RecurrenceDetails {
		t.Fatalf("update recurrenceDetails = %q, want desired recurrence", got)
	}
	if got := request.FreeformTags; !reflect.DeepEqual(got, desired.Spec.FreeformTags) {
		t.Fatalf("update freeformTags = %#v, want %#v", got, desired.Spec.FreeformTags)
	}
}

func TestScheduleServiceClientCreateOrUpdateRejectsCompartmentDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newScheduleRuntimeTestResource()
	trackScheduleID(resource, testScheduleID)
	fake := &fakeScheduleOCIClient{}
	fake.getFunc = func(context.Context, resourceschedulersdk.GetScheduleRequest) (resourceschedulersdk.GetScheduleResponse, error) {
		currentSpec := resource.Spec
		currentSpec.CompartmentId = "ocid1.compartment.oc1..observed"
		return resourceschedulersdk.GetScheduleResponse{
			Schedule: scheduleFromSpec(testScheduleID, currentSpec, resourceschedulersdk.ScheduleLifecycleStateActive),
		}, nil
	}
	fake.updateFunc = func(context.Context, resourceschedulersdk.UpdateScheduleRequest) (resourceschedulersdk.UpdateScheduleResponse, error) {
		t.Fatal("UpdateSchedule() called; want create-only drift rejection")
		return resourceschedulersdk.UpdateScheduleResponse{}, nil
	}
	client := newScheduleRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, scheduleReconcileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only drift rejection")
	}
	if !strings.Contains(err.Error(), "replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want compartmentId replacement rejection", err)
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success for create-only drift")
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("UpdateSchedule() calls = %d, want 0", len(fake.updateRequests))
	}
}

//nolint:gocognit,gocyclo // Delete lifecycle sequencing is the contract under test.
func TestScheduleServiceClientDeleteRetainsFinalizerUntilLifecycleDeleteConfirmed(t *testing.T) {
	t.Parallel()

	resource := newScheduleRuntimeTestResource()
	trackScheduleID(resource, testScheduleID)
	fake := &fakeScheduleOCIClient{}
	fake.getFunc = func(_ context.Context, request resourceschedulersdk.GetScheduleRequest) (resourceschedulersdk.GetScheduleResponse, error) {
		if got := scheduleStringValue(request.ScheduleId); got != testScheduleID {
			t.Fatalf("get scheduleId = %q, want %q", got, testScheduleID)
		}
		switch len(fake.getRequests) {
		case 1, 2:
			return resourceschedulersdk.GetScheduleResponse{
				Schedule: scheduleFromSpec(testScheduleID, resource.Spec, resourceschedulersdk.ScheduleLifecycleStateActive),
			}, nil
		case 3:
			return resourceschedulersdk.GetScheduleResponse{
				Schedule: scheduleFromSpec(testScheduleID, resource.Spec, resourceschedulersdk.ScheduleLifecycleStateDeleting),
			}, nil
		case 4, 5:
			return resourceschedulersdk.GetScheduleResponse{
				Schedule: scheduleFromSpec(testScheduleID, resource.Spec, resourceschedulersdk.ScheduleLifecycleStateDeleted),
			}, nil
		default:
			t.Fatalf("unexpected GetSchedule() call %d", len(fake.getRequests))
			return resourceschedulersdk.GetScheduleResponse{}, nil
		}
	}
	fake.deleteFunc = func(_ context.Context, request resourceschedulersdk.DeleteScheduleRequest) (resourceschedulersdk.DeleteScheduleResponse, error) {
		if got := scheduleStringValue(request.ScheduleId); got != testScheduleID {
			t.Fatalf("delete scheduleId = %q, want %q", got, testScheduleID)
		}
		return resourceschedulersdk.DeleteScheduleResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	client := newScheduleRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first call error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first call deleted = true, want finalizer retained while DELETING")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteSchedule() calls after first delete = %d, want 1", len(fake.deleteRequests))
	}
	if got := resource.Status.LifecycleState; got != string(resourceschedulersdk.ScheduleLifecycleStateDeleting) {
		t.Fatalf("status.lifecycleState after first delete = %q, want DELETING", got)
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.status.async.current = nil, want delete lifecycle tracker")
	}
	if got := resource.Status.OsokStatus.Async.Current.Phase; got != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.status.async.current.phase = %q, want delete", got)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete-1", got)
	}

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second call error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second call deleted = false, want finalizer release after DELETED")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteSchedule() calls after second delete = %d, want no reissue", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete timestamp")
	}
}

func TestScheduleServiceClientDeleteWithoutTrackedIDReleasesFinalizerWhenListFindsNoMatch(t *testing.T) {
	t.Parallel()

	resource := newScheduleRuntimeTestResource()
	fake := &fakeScheduleOCIClient{}
	fake.listFunc = func(_ context.Context, request resourceschedulersdk.ListSchedulesRequest) (resourceschedulersdk.ListSchedulesResponse, error) {
		assertScheduleBindListRequest(t, request)
		return resourceschedulersdk.ListSchedulesResponse{}, nil
	}
	fake.deleteFunc = func(context.Context, resourceschedulersdk.DeleteScheduleRequest) (resourceschedulersdk.DeleteScheduleResponse, error) {
		t.Fatal("DeleteSchedule() called; want no-match list confirmation to release finalizer without OCI delete")
		return resourceschedulersdk.DeleteScheduleResponse{}, nil
	}
	client := newScheduleRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release for unambiguous no-match confirmation")
	}
	if len(fake.listRequests) != 1 {
		t.Fatalf("ListSchedules() calls = %d, want 1", len(fake.listRequests))
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteSchedule() calls = %d, want 0", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want delete timestamp")
	}
	if got := resource.Status.OsokStatus.Reason; got != string(shared.Terminating) {
		t.Fatalf("status.status.reason = %q, want Terminating", got)
	}
}

func TestScheduleServiceClientDeleteWithoutTrackedIDRejectsAuthShapedListConfirmRead(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-list-confirm-error-1"

	resource := newScheduleRuntimeTestResource()
	fake := &fakeScheduleOCIClient{}
	fake.listFunc = func(context.Context, resourceschedulersdk.ListSchedulesRequest) (resourceschedulersdk.ListSchedulesResponse, error) {
		return resourceschedulersdk.ListSchedulesResponse{}, serviceErr
	}
	fake.deleteFunc = func(context.Context, resourceschedulersdk.DeleteScheduleRequest) (resourceschedulersdk.DeleteScheduleResponse, error) {
		t.Fatal("DeleteSchedule() called; want auth-shaped list confirmation to retain finalizer")
		return resourceschedulersdk.DeleteScheduleResponse{}, nil
	}
	client := newScheduleRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped list confirm-read to stay fatal")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous list confirm-read 404", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteSchedule() calls = %d, want 0", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-list-confirm-error-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-list-confirm-error-1", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set, want finalizer-retaining delete status")
	}
}

func TestScheduleServiceClientDeleteWithoutTrackedIDRejectsMultipleListMatches(t *testing.T) {
	t.Parallel()

	resource := newScheduleRuntimeTestResource()
	fake := &fakeScheduleOCIClient{}
	fake.listFunc = func(_ context.Context, request resourceschedulersdk.ListSchedulesRequest) (resourceschedulersdk.ListSchedulesResponse, error) {
		assertScheduleBindListRequest(t, request)
		return resourceschedulersdk.ListSchedulesResponse{
			ScheduleCollection: resourceschedulersdk.ScheduleCollection{
				Items: []resourceschedulersdk.ScheduleSummary{
					scheduleSummaryFromSpec(testScheduleID, resource.Spec, resourceschedulersdk.ScheduleLifecycleStateActive),
					scheduleSummaryFromSpec(testScheduleOtherID, resource.Spec, resourceschedulersdk.ScheduleLifecycleStateActive),
				},
			},
		}, nil
	}
	fake.deleteFunc = func(context.Context, resourceschedulersdk.DeleteScheduleRequest) (resourceschedulersdk.DeleteScheduleResponse, error) {
		t.Fatal("DeleteSchedule() called; want duplicate list matches to retain finalizer")
		return resourceschedulersdk.DeleteScheduleResponse{}, nil
	}
	client := newScheduleRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want duplicate list matches to stay fatal")
	}
	if !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("Delete() error = %v, want duplicate-match error", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteSchedule() calls = %d, want 0", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set, want finalizer-retaining delete status")
	}
}

func TestScheduleServiceClientDeleteRejectsAuthShapedAfterRequestConfirmRead(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-confirm-after-error-1"

	resource := newScheduleRuntimeTestResource()
	trackScheduleID(resource, testScheduleID)
	fake := &fakeScheduleOCIClient{}
	fake.getFunc = func(_ context.Context, request resourceschedulersdk.GetScheduleRequest) (resourceschedulersdk.GetScheduleResponse, error) {
		assertScheduleGetRequest(t, request)
		switch len(fake.getRequests) {
		case 1, 2:
			return resourceschedulersdk.GetScheduleResponse{
				Schedule: scheduleFromSpec(testScheduleID, resource.Spec, resourceschedulersdk.ScheduleLifecycleStateActive),
			}, nil
		case 3:
			return resourceschedulersdk.GetScheduleResponse{}, serviceErr
		default:
			t.Fatalf("unexpected GetSchedule() call %d", len(fake.getRequests))
			return resourceschedulersdk.GetScheduleResponse{}, nil
		}
	}
	fake.deleteFunc = func(_ context.Context, request resourceschedulersdk.DeleteScheduleRequest) (resourceschedulersdk.DeleteScheduleResponse, error) {
		if got := scheduleStringValue(request.ScheduleId); got != testScheduleID {
			t.Fatalf("delete scheduleId = %q, want %q", got, testScheduleID)
		}
		return resourceschedulersdk.DeleteScheduleResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	client := newScheduleRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped after-request confirm-read to stay fatal")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous after-request confirm-read 404", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if len(fake.deleteRequests) != 1 {
		t.Fatalf("DeleteSchedule() calls = %d, want 1", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-confirm-after-error-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-confirm-after-error-1", got)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.status.deletedAt set, want finalizer-retaining delete status")
	}
}

func TestScheduleServiceClientDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found")
	serviceErr.OpcRequestID = "opc-confirm-pre-error-1"

	resource := newScheduleRuntimeTestResource()
	trackScheduleID(resource, testScheduleID)
	fake := &fakeScheduleOCIClient{}
	fake.getFunc = func(context.Context, resourceschedulersdk.GetScheduleRequest) (resourceschedulersdk.GetScheduleResponse, error) {
		return resourceschedulersdk.GetScheduleResponse{}, serviceErr
	}
	fake.deleteFunc = func(context.Context, resourceschedulersdk.DeleteScheduleRequest) (resourceschedulersdk.DeleteScheduleResponse, error) {
		t.Fatal("DeleteSchedule() called; want pre-delete auth-shaped read to stop delete")
		return resourceschedulersdk.DeleteScheduleResponse{}, nil
	}
	client := newScheduleRuntimeTestClient(fake)

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want auth-shaped pre-delete read to stay fatal")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read 404", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("DeleteSchedule() calls = %d, want 0", len(fake.deleteRequests))
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-confirm-pre-error-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-confirm-pre-error-1", got)
	}
}

func TestScheduleServiceClientCreateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	serviceErr := errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
	serviceErr.OpcRequestID = "opc-create-error-1"

	resource := newScheduleRuntimeTestResource()
	fake := &fakeScheduleOCIClient{}
	fake.listFunc = func(context.Context, resourceschedulersdk.ListSchedulesRequest) (resourceschedulersdk.ListSchedulesResponse, error) {
		return resourceschedulersdk.ListSchedulesResponse{}, nil
	}
	fake.createFunc = func(context.Context, resourceschedulersdk.CreateScheduleRequest) (resourceschedulersdk.CreateScheduleResponse, error) {
		return resourceschedulersdk.CreateScheduleResponse{}, serviceErr
	}
	client := newScheduleRuntimeTestClient(fake)

	response, err := client.CreateOrUpdate(context.Background(), resource, scheduleReconcileRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want surfaced OCI create error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should not report success")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create-error-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create-error-1", got)
	}
}

func newScheduleRuntimeTestClient(fake *fakeScheduleOCIClient) ScheduleServiceClient {
	hooks := newScheduleRuntimeHooksWithOCIClient(fake)
	applyScheduleRuntimeHooks(&hooks)
	manager := &ScheduleServiceManager{
		Log: loggerutil.OSOKLogger{Logger: logr.Discard()},
	}
	delegate := defaultScheduleServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*resourceschedulerv1beta1.Schedule](
			buildScheduleGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapScheduleGeneratedClient(hooks, delegate)
}

func newScheduleRuntimeTestResource() *resourceschedulerv1beta1.Schedule {
	return &resourceschedulerv1beta1.Schedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "schedule-sample",
			Namespace: "default",
			UID:       types.UID("schedule-uid"),
		},
		Spec: resourceschedulerv1beta1.ScheduleSpec{
			CompartmentId:     testScheduleCompartmentID,
			DisplayName:       testScheduleDisplayName,
			Description:       "runtime schedule",
			Action:            string(resourceschedulersdk.ScheduleActionStartResource),
			RecurrenceDetails: "FREQ=DAILY;INTERVAL=1",
			RecurrenceType:    string(resourceschedulersdk.ScheduleRecurrenceTypeIcal),
			ResourceFilters: []resourceschedulerv1beta1.ScheduleResourceFilter{
				{
					Attribute: string(resourceschedulersdk.ResourceFilterAttributeResourceType),
					Value:     "instance",
				},
			},
			Resources: []resourceschedulerv1beta1.ScheduleResource{
				{
					Id:       testScheduleResourceID,
					Metadata: map[string]string{"compartmentId": testScheduleCompartmentID},
					Parameters: []resourceschedulerv1beta1.ScheduleResourceParameter{
						{
							ParameterType: string(resourceschedulersdk.ParameterParameterTypeBody),
							Value:         map[string]string{"action": "softstop"},
						},
					},
				},
			},
			TimeStarts:   "2026-01-15T15:00:00Z",
			FreeformTags: map[string]string{"env": "dev"},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {"CostCenter": "42"},
			},
		},
	}
}

func scheduleReconcileRequest(resource *resourceschedulerv1beta1.Schedule) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}}
}

func trackScheduleID(resource *resourceschedulerv1beta1.Schedule, id string) {
	resource.Status.Id = id
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
}

func scheduleFromSpec(
	id string,
	spec resourceschedulerv1beta1.ScheduleSpec,
	state resourceschedulersdk.ScheduleLifecycleStateEnum,
) resourceschedulersdk.Schedule {
	filters, _ := scheduleResourceFiltersFromSpec(spec.ResourceFilters)
	resources, _ := scheduleResourcesFromSpec(spec.Resources)
	timeStarts, _ := scheduleSDKTimeFromSpec("timeStarts", spec.TimeStarts)
	timeEnds, _ := scheduleSDKTimeFromSpec("timeEnds", spec.TimeEnds)
	return resourceschedulersdk.Schedule{
		Id:                common.String(id),
		CompartmentId:     common.String(spec.CompartmentId),
		DisplayName:       common.String(spec.DisplayName),
		Action:            resourceschedulersdk.ScheduleActionEnum(scheduleNormalizeEnum(spec.Action)),
		RecurrenceDetails: common.String(spec.RecurrenceDetails),
		RecurrenceType:    resourceschedulersdk.ScheduleRecurrenceTypeEnum(scheduleNormalizeEnum(spec.RecurrenceType)),
		TimeCreated:       &common.SDKTime{Time: metav1.Now().Time},
		LifecycleState:    state,
		Description:       common.String(spec.Description),
		ResourceFilters:   filters,
		Resources:         resources,
		TimeStarts:        timeStarts,
		TimeEnds:          timeEnds,
		FreeformTags:      cloneScheduleStringMap(spec.FreeformTags),
		DefinedTags:       scheduleDefinedTagsFromSpec(spec.DefinedTags),
	}
}

func scheduleSummaryFromSpec(
	id string,
	spec resourceschedulerv1beta1.ScheduleSpec,
	state resourceschedulersdk.ScheduleLifecycleStateEnum,
) resourceschedulersdk.ScheduleSummary {
	schedule := scheduleFromSpec(id, spec, state)
	return resourceschedulersdk.ScheduleSummary{
		Id:                schedule.Id,
		CompartmentId:     schedule.CompartmentId,
		DisplayName:       schedule.DisplayName,
		Action:            resourceschedulersdk.ScheduleSummaryActionEnum(schedule.Action),
		RecurrenceDetails: schedule.RecurrenceDetails,
		RecurrenceType:    resourceschedulersdk.ScheduleSummaryRecurrenceTypeEnum(schedule.RecurrenceType),
		LifecycleState:    schedule.LifecycleState,
		Description:       schedule.Description,
		ResourceFilters:   schedule.ResourceFilters,
		Resources:         schedule.Resources,
		TimeStarts:        schedule.TimeStarts,
		TimeEnds:          schedule.TimeEnds,
		TimeCreated:       schedule.TimeCreated,
		FreeformTags:      schedule.FreeformTags,
		DefinedTags:       schedule.DefinedTags,
	}
}

func assertScheduleCreateRequest(
	t *testing.T,
	request resourceschedulersdk.CreateScheduleRequest,
	resource *resourceschedulerv1beta1.Schedule,
) {
	t.Helper()

	assertScheduleCreateScalarFields(t, request, resource)
	assertScheduleCreateResourceFilter(t, request)
	assertScheduleCreateResource(t, request)
}

func assertScheduleCreateScalarFields(
	t *testing.T,
	request resourceschedulersdk.CreateScheduleRequest,
	resource *resourceschedulerv1beta1.Schedule,
) {
	t.Helper()

	if got := scheduleStringValue(request.OpcRetryToken); got != string(resource.UID) {
		t.Fatalf("create opcRetryToken = %q, want resource UID", got)
	}
	if got := scheduleStringValue(request.CompartmentId); got != resource.Spec.CompartmentId {
		t.Fatalf("create compartmentId = %q, want %q", got, resource.Spec.CompartmentId)
	}
	if got := scheduleStringValue(request.DisplayName); got != resource.Spec.DisplayName {
		t.Fatalf("create displayName = %q, want %q", got, resource.Spec.DisplayName)
	}
	if got := request.Action; got != resourceschedulersdk.CreateScheduleDetailsActionStartResource {
		t.Fatalf("create action = %q, want START_RESOURCE", got)
	}
	if got := request.RecurrenceType; got != resourceschedulersdk.CreateScheduleDetailsRecurrenceTypeIcal {
		t.Fatalf("create recurrenceType = %q, want ICAL", got)
	}
}

func assertScheduleCreateResourceFilter(t *testing.T, request resourceschedulersdk.CreateScheduleRequest) {
	t.Helper()

	if len(request.ResourceFilters) != 1 {
		t.Fatalf("create resourceFilters length = %d, want 1", len(request.ResourceFilters))
	}
	filter, ok := request.ResourceFilters[0].(resourceschedulersdk.ResourceTypeResourceFilter)
	if !ok {
		t.Fatalf("create resourceFilters[0] = %T, want ResourceTypeResourceFilter", request.ResourceFilters[0])
	}
	if !reflect.DeepEqual(filter.Value, []string{"instance"}) {
		t.Fatalf("create resource filter value = %#v, want [instance]", filter.Value)
	}
}

func assertScheduleCreateResource(t *testing.T, request resourceschedulersdk.CreateScheduleRequest) {
	t.Helper()

	if len(request.Resources) != 1 || scheduleStringValue(request.Resources[0].Id) != testScheduleResourceID {
		t.Fatalf("create resources = %#v, want scheduled resource", request.Resources)
	}
	parameter, ok := request.Resources[0].Parameters[0].(resourceschedulersdk.BodyParameter)
	if !ok {
		t.Fatalf("create parameter = %T, want BodyParameter", request.Resources[0].Parameters[0])
	}
	body, ok := (*parameter.Value).(map[string]interface{})
	if !ok || body["action"] != "softstop" {
		t.Fatalf("create body parameter value = %#v, want action=softstop", parameter.Value)
	}
}

func cloneScheduleStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
