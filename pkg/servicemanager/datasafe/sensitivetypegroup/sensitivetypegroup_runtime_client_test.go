/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sensitivetypegroup

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testSensitiveTypeGroupID            = "ocid1.datasafesensitivetypegroup.oc1..group"
	testSensitiveTypeGroupCompartmentID = "ocid1.compartment.oc1..datasafe"
	testSensitiveTypeGroupDisplayName   = "customer-sensitive-types"
	testSensitiveTypeGroupDescription   = "customer sensitive type group"
)

type fakeSensitiveTypeGroupOCI struct {
	createRequests []datasafesdk.CreateSensitiveTypeGroupRequest
	getRequests    []datasafesdk.GetSensitiveTypeGroupRequest
	listRequests   []datasafesdk.ListSensitiveTypeGroupsRequest
	updateRequests []datasafesdk.UpdateSensitiveTypeGroupRequest
	deleteRequests []datasafesdk.DeleteSensitiveTypeGroupRequest
	workRequests   []datasafesdk.GetWorkRequestRequest

	create         func(context.Context, datasafesdk.CreateSensitiveTypeGroupRequest) (datasafesdk.CreateSensitiveTypeGroupResponse, error)
	get            func(context.Context, datasafesdk.GetSensitiveTypeGroupRequest) (datasafesdk.GetSensitiveTypeGroupResponse, error)
	list           func(context.Context, datasafesdk.ListSensitiveTypeGroupsRequest) (datasafesdk.ListSensitiveTypeGroupsResponse, error)
	update         func(context.Context, datasafesdk.UpdateSensitiveTypeGroupRequest) (datasafesdk.UpdateSensitiveTypeGroupResponse, error)
	delete         func(context.Context, datasafesdk.DeleteSensitiveTypeGroupRequest) (datasafesdk.DeleteSensitiveTypeGroupResponse, error)
	getWorkRequest func(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
}

func (f *fakeSensitiveTypeGroupOCI) createSensitiveTypeGroup(
	ctx context.Context,
	request datasafesdk.CreateSensitiveTypeGroupRequest,
) (datasafesdk.CreateSensitiveTypeGroupResponse, error) {
	f.createRequests = append(f.createRequests, request)
	if f.create != nil {
		return f.create(ctx, request)
	}
	return datasafesdk.CreateSensitiveTypeGroupResponse{}, nil
}

func (f *fakeSensitiveTypeGroupOCI) getSensitiveTypeGroup(
	ctx context.Context,
	request datasafesdk.GetSensitiveTypeGroupRequest,
) (datasafesdk.GetSensitiveTypeGroupResponse, error) {
	f.getRequests = append(f.getRequests, request)
	if f.get != nil {
		return f.get(ctx, request)
	}
	return datasafesdk.GetSensitiveTypeGroupResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing")
}

func (f *fakeSensitiveTypeGroupOCI) listSensitiveTypeGroups(
	ctx context.Context,
	request datasafesdk.ListSensitiveTypeGroupsRequest,
) (datasafesdk.ListSensitiveTypeGroupsResponse, error) {
	f.listRequests = append(f.listRequests, request)
	if f.list != nil {
		return f.list(ctx, request)
	}
	return datasafesdk.ListSensitiveTypeGroupsResponse{}, nil
}

func (f *fakeSensitiveTypeGroupOCI) updateSensitiveTypeGroup(
	ctx context.Context,
	request datasafesdk.UpdateSensitiveTypeGroupRequest,
) (datasafesdk.UpdateSensitiveTypeGroupResponse, error) {
	f.updateRequests = append(f.updateRequests, request)
	if f.update != nil {
		return f.update(ctx, request)
	}
	return datasafesdk.UpdateSensitiveTypeGroupResponse{}, nil
}

func (f *fakeSensitiveTypeGroupOCI) deleteSensitiveTypeGroup(
	ctx context.Context,
	request datasafesdk.DeleteSensitiveTypeGroupRequest,
) (datasafesdk.DeleteSensitiveTypeGroupResponse, error) {
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete != nil {
		return f.delete(ctx, request)
	}
	return datasafesdk.DeleteSensitiveTypeGroupResponse{}, nil
}

func (f *fakeSensitiveTypeGroupOCI) GetWorkRequest(
	ctx context.Context,
	request datasafesdk.GetWorkRequestRequest,
) (datasafesdk.GetWorkRequestResponse, error) {
	f.workRequests = append(f.workRequests, request)
	if f.getWorkRequest != nil {
		return f.getWorkRequest(ctx, request)
	}
	workRequestID := sensitiveTypeGroupStringValue(request.WorkRequestId)
	phase := shared.OSOKAsyncPhaseCreate
	switch {
	case strings.Contains(workRequestID, "update"):
		phase = shared.OSOKAsyncPhaseUpdate
	case strings.Contains(workRequestID, "delete"):
		phase = shared.OSOKAsyncPhaseDelete
	}
	return datasafesdk.GetWorkRequestResponse{
		WorkRequest: sensitiveTypeGroupWorkRequest(workRequestID, phase, datasafesdk.WorkRequestStatusInProgress),
	}, nil
}

func newTestSensitiveTypeGroupClient(fake *fakeSensitiveTypeGroupOCI) SensitiveTypeGroupServiceClient {
	hooks := newSensitiveTypeGroupDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	hooks.Create.Call = fake.createSensitiveTypeGroup
	hooks.Get.Call = fake.getSensitiveTypeGroup
	hooks.List.Call = fake.listSensitiveTypeGroups
	hooks.Update.Call = fake.updateSensitiveTypeGroup
	hooks.Delete.Call = fake.deleteSensitiveTypeGroup
	applySensitiveTypeGroupRuntimeHooks(&hooks, fake, nil)

	manager := &SensitiveTypeGroupServiceManager{}
	delegate := defaultSensitiveTypeGroupServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.SensitiveTypeGroup](
			buildSensitiveTypeGroupGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapSensitiveTypeGroupGeneratedClient(hooks, delegate)
}

func TestSensitiveTypeGroupRuntimeHooksConfigured(t *testing.T) {
	t.Parallel()

	hooks := newSensitiveTypeGroupDefaultRuntimeHooks(datasafesdk.DataSafeClient{})
	applySensitiveTypeGroupRuntimeHooks(&hooks, &fakeSensitiveTypeGroupOCI{}, nil)

	checks := []struct {
		name string
		ok   bool
	}{
		{name: "Semantics", ok: hooks.Semantics != nil},
		{name: "BuildCreateBody", ok: hooks.BuildCreateBody != nil},
		{name: "BuildUpdateBody", ok: hooks.BuildUpdateBody != nil},
		{name: "Identity.Resolve", ok: hooks.Identity.Resolve != nil},
		{name: "Identity.RecordPath", ok: hooks.Identity.RecordPath != nil},
		{name: "Identity.GuardExistingBeforeCreate", ok: hooks.Identity.GuardExistingBeforeCreate != nil},
		{name: "DeleteHooks.HandleError", ok: hooks.DeleteHooks.HandleError != nil},
		{name: "StatusHooks.ProjectStatus", ok: hooks.StatusHooks.ProjectStatus != nil},
		{name: "StatusHooks.MarkTerminating", ok: hooks.StatusHooks.MarkTerminating != nil},
		{name: "Async.GetWorkRequest", ok: hooks.Async.GetWorkRequest != nil},
		{name: "WrapGeneratedClient", ok: len(hooks.WrapGeneratedClient) > 0},
	}
	for _, check := range checks {
		if !check.ok {
			t.Fatalf("hooks.%s not configured", check.name)
		}
	}

	body, err := hooks.BuildCreateBody(context.Background(), newSensitiveTypeGroupResource(), "default")
	if err != nil {
		t.Fatalf("BuildCreateBody() error = %v", err)
	}
	details, ok := body.(datasafesdk.CreateSensitiveTypeGroupDetails)
	if !ok {
		t.Fatalf("BuildCreateBody() type = %T, want CreateSensitiveTypeGroupDetails", body)
	}
	requireSensitiveTypeGroupStringPtr(t, "create compartmentId", details.CompartmentId, testSensitiveTypeGroupCompartmentID)
	requireSensitiveTypeGroupStringPtr(t, "create displayName", details.DisplayName, testSensitiveTypeGroupDisplayName)
	requireSensitiveTypeGroupStringPtr(t, "create description", details.Description, testSensitiveTypeGroupDescription)
}

func TestSensitiveTypeGroupCreateProjectsLifecycleAndRequestID(t *testing.T) {
	t.Parallel()

	resource := newSensitiveTypeGroupResource()
	fake := &fakeSensitiveTypeGroupOCI{}
	fake.create = func(_ context.Context, request datasafesdk.CreateSensitiveTypeGroupRequest) (datasafesdk.CreateSensitiveTypeGroupResponse, error) {
		requireSensitiveTypeGroupCreateRequest(t, request, resource)
		return datasafesdk.CreateSensitiveTypeGroupResponse{
			SensitiveTypeGroup: sensitiveTypeGroupBody(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateCreating),
			OpcRequestId:       common.String("opc-create"),
			OpcWorkRequestId:   common.String("wr-create"),
		}, nil
	}
	fake.get = func(_ context.Context, request datasafesdk.GetSensitiveTypeGroupRequest) (datasafesdk.GetSensitiveTypeGroupResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "get sensitiveTypeGroupId", request.SensitiveTypeGroupId, testSensitiveTypeGroupID)
		return datasafesdk.GetSensitiveTypeGroupResponse{
			SensitiveTypeGroup: sensitiveTypeGroupBody(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateCreating),
		}, nil
	}

	response, err := newTestSensitiveTypeGroupClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() ShouldRequeue = false, want lifecycle requeue")
	}
	requireSensitiveTypeGroupCallCount(t, "ListSensitiveTypeGroups", len(fake.listRequests), 1)
	requireSensitiveTypeGroupCallCount(t, "CreateSensitiveTypeGroup", len(fake.createRequests), 1)
	requireSensitiveTypeGroupCallCount(t, "GetSensitiveTypeGroup", len(fake.getRequests), 0)
	requireSensitiveTypeGroupCallCount(t, "GetWorkRequest", len(fake.workRequests), 1)
	requireSensitiveTypeGroupRecordedID(t, resource, testSensitiveTypeGroupID)
	requireSensitiveTypeGroupString(t, "status.lifecycleState", resource.Status.LifecycleState, "CREATING")
	requireSensitiveTypeGroupString(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-create")
	requireSensitiveTypeGroupWorkRequestAsync(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create", "IN_PROGRESS")
}

func TestSensitiveTypeGroupBindsExistingFromLaterListPage(t *testing.T) {
	t.Parallel()

	resource := newSensitiveTypeGroupResource()
	fake := &fakeSensitiveTypeGroupOCI{}
	fake.list = func(_ context.Context, request datasafesdk.ListSensitiveTypeGroupsRequest) (datasafesdk.ListSensitiveTypeGroupsResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "list compartmentId", request.CompartmentId, testSensitiveTypeGroupCompartmentID)
		requireSensitiveTypeGroupStringPtr(t, "list displayName", request.DisplayName, testSensitiveTypeGroupDisplayName)
		if request.Page == nil {
			return datasafesdk.ListSensitiveTypeGroupsResponse{
				SensitiveTypeGroupCollection: datasafesdk.SensitiveTypeGroupCollection{
					Items: []datasafesdk.SensitiveTypeGroupSummary{
						sensitiveTypeGroupSummary(resource, "ocid1.datasafesensitivetypegroup.oc1..other", "other", datasafesdk.SensitiveTypeGroupLifecycleStateActive),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		}
		requireSensitiveTypeGroupStringPtr(t, "list page", request.Page, "page-2")
		return datasafesdk.ListSensitiveTypeGroupsResponse{
			SensitiveTypeGroupCollection: datasafesdk.SensitiveTypeGroupCollection{
				Items: []datasafesdk.SensitiveTypeGroupSummary{
					sensitiveTypeGroupSummary(resource, testSensitiveTypeGroupID, testSensitiveTypeGroupDisplayName, datasafesdk.SensitiveTypeGroupLifecycleStateActive),
				},
			},
		}, nil
	}
	fake.create = func(context.Context, datasafesdk.CreateSensitiveTypeGroupRequest) (datasafesdk.CreateSensitiveTypeGroupResponse, error) {
		t.Fatal("CreateSensitiveTypeGroup called despite reusable list match")
		return datasafesdk.CreateSensitiveTypeGroupResponse{}, nil
	}
	fake.get = func(_ context.Context, request datasafesdk.GetSensitiveTypeGroupRequest) (datasafesdk.GetSensitiveTypeGroupResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "get sensitiveTypeGroupId", request.SensitiveTypeGroupId, testSensitiveTypeGroupID)
		return datasafesdk.GetSensitiveTypeGroupResponse{
			SensitiveTypeGroup: sensitiveTypeGroupBody(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive),
		}, nil
	}

	response, err := newTestSensitiveTypeGroupClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	requireSensitiveTypeGroupCallCount(t, "ListSensitiveTypeGroups", len(fake.listRequests), 2)
	requireSensitiveTypeGroupCallCount(t, "CreateSensitiveTypeGroup", len(fake.createRequests), 0)
	requireSensitiveTypeGroupRecordedID(t, resource, testSensitiveTypeGroupID)
}

func TestSensitiveTypeGroupNoopReconcileSkipsUpdate(t *testing.T) {
	t.Parallel()

	resource := newSensitiveTypeGroupResource()
	seedSensitiveTypeGroupStatus(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive)
	fake := &fakeSensitiveTypeGroupOCI{}
	fake.get = func(_ context.Context, request datasafesdk.GetSensitiveTypeGroupRequest) (datasafesdk.GetSensitiveTypeGroupResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "get sensitiveTypeGroupId", request.SensitiveTypeGroupId, testSensitiveTypeGroupID)
		return datasafesdk.GetSensitiveTypeGroupResponse{
			SensitiveTypeGroup: sensitiveTypeGroupBody(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive),
		}, nil
	}
	fake.update = func(context.Context, datasafesdk.UpdateSensitiveTypeGroupRequest) (datasafesdk.UpdateSensitiveTypeGroupResponse, error) {
		t.Fatal("UpdateSensitiveTypeGroup called during no-op reconcile")
		return datasafesdk.UpdateSensitiveTypeGroupResponse{}, nil
	}

	response, err := newTestSensitiveTypeGroupClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	requireSensitiveTypeGroupCallCount(t, "UpdateSensitiveTypeGroup", len(fake.updateRequests), 0)
	requireSensitiveTypeGroupCondition(t, resource, shared.Active)
}

func TestSensitiveTypeGroupMutableUpdateRefreshesObservedState(t *testing.T) {
	t.Parallel()

	resource := newSensitiveTypeGroupResource()
	seedSensitiveTypeGroupStatus(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive)
	resource.Spec.DisplayName = "updated-sensitive-types"
	resource.Spec.FreeformTags = map[string]string{}
	getCalls := 0
	fake := &fakeSensitiveTypeGroupOCI{}
	fake.get = func(_ context.Context, request datasafesdk.GetSensitiveTypeGroupRequest) (datasafesdk.GetSensitiveTypeGroupResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "get sensitiveTypeGroupId", request.SensitiveTypeGroupId, testSensitiveTypeGroupID)
		getCalls++
		if getCalls == 1 {
			current := sensitiveTypeGroupBody(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive)
			current.DisplayName = common.String(testSensitiveTypeGroupDisplayName)
			current.FreeformTags = map[string]string{"keep": "old"}
			return datasafesdk.GetSensitiveTypeGroupResponse{SensitiveTypeGroup: current}, nil
		}
		return datasafesdk.GetSensitiveTypeGroupResponse{
			SensitiveTypeGroup: sensitiveTypeGroupBody(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive),
		}, nil
	}
	fake.update = func(_ context.Context, request datasafesdk.UpdateSensitiveTypeGroupRequest) (datasafesdk.UpdateSensitiveTypeGroupResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "update sensitiveTypeGroupId", request.SensitiveTypeGroupId, testSensitiveTypeGroupID)
		requireSensitiveTypeGroupStringPtr(t, "update displayName", request.DisplayName, resource.Spec.DisplayName)
		if request.FreeformTags == nil || len(request.FreeformTags) != 0 {
			t.Fatalf("update freeformTags = %#v, want explicit empty map", request.FreeformTags)
		}
		return datasafesdk.UpdateSensitiveTypeGroupResponse{
			OpcRequestId:     common.String("opc-update"),
			OpcWorkRequestId: common.String("wr-update"),
		}, nil
	}
	fake.getWorkRequest = func(_ context.Context, request datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "workRequestId", request.WorkRequestId, "wr-update")
		return datasafesdk.GetWorkRequestResponse{
			WorkRequest: sensitiveTypeGroupWorkRequest("wr-update", shared.OSOKAsyncPhaseUpdate, datasafesdk.WorkRequestStatusSucceeded),
		}, nil
	}

	response, err := newTestSensitiveTypeGroupClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = false, want true")
	}
	requireSensitiveTypeGroupCallCount(t, "UpdateSensitiveTypeGroup", len(fake.updateRequests), 1)
	requireSensitiveTypeGroupCallCount(t, "GetWorkRequest", len(fake.workRequests), 1)
	requireSensitiveTypeGroupString(t, "status.displayName", resource.Status.DisplayName, resource.Spec.DisplayName)
	requireSensitiveTypeGroupString(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-update")
}

func TestSensitiveTypeGroupUpdateWorkRequestKeepsPendingOnStaleActiveReadback(t *testing.T) {
	t.Parallel()

	resource := newSensitiveTypeGroupResource()
	seedSensitiveTypeGroupStatus(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive)
	resource.Spec.DisplayName = "updated-sensitive-types"
	fake := &fakeSensitiveTypeGroupOCI{}
	fake.get = func(_ context.Context, request datasafesdk.GetSensitiveTypeGroupRequest) (datasafesdk.GetSensitiveTypeGroupResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "get sensitiveTypeGroupId", request.SensitiveTypeGroupId, testSensitiveTypeGroupID)
		current := sensitiveTypeGroupBody(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive)
		current.DisplayName = common.String(testSensitiveTypeGroupDisplayName)
		return datasafesdk.GetSensitiveTypeGroupResponse{SensitiveTypeGroup: current}, nil
	}
	fake.update = func(_ context.Context, request datasafesdk.UpdateSensitiveTypeGroupRequest) (datasafesdk.UpdateSensitiveTypeGroupResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "update sensitiveTypeGroupId", request.SensitiveTypeGroupId, testSensitiveTypeGroupID)
		requireSensitiveTypeGroupStringPtr(t, "update displayName", request.DisplayName, resource.Spec.DisplayName)
		return datasafesdk.UpdateSensitiveTypeGroupResponse{
			OpcRequestId:     common.String("opc-update"),
			OpcWorkRequestId: common.String("wr-update-stale"),
		}, nil
	}
	fake.getWorkRequest = func(_ context.Context, request datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "workRequestId", request.WorkRequestId, "wr-update-stale")
		return datasafesdk.GetWorkRequestResponse{
			WorkRequest: sensitiveTypeGroupWorkRequest("wr-update-stale", shared.OSOKAsyncPhaseUpdate, datasafesdk.WorkRequestStatusSucceeded),
		}, nil
	}

	client := newTestSensitiveTypeGroupClient(fake)
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	requireSensitiveTypeGroupCallCount(t, "UpdateSensitiveTypeGroup", len(fake.updateRequests), 1)
	requireSensitiveTypeGroupWorkRequestAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update-stale", "ACCEPTED")

	fake.update = func(context.Context, datasafesdk.UpdateSensitiveTypeGroupRequest) (datasafesdk.UpdateSensitiveTypeGroupResponse, error) {
		t.Fatal("UpdateSensitiveTypeGroup reissued while stale update work request was still pending")
		return datasafesdk.UpdateSensitiveTypeGroupResponse{}, nil
	}
	response, err = client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("second CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("second CreateOrUpdate() response = %#v, want successful requeue", response)
	}
	requireSensitiveTypeGroupCallCount(t, "UpdateSensitiveTypeGroup", len(fake.updateRequests), 1)
	requireSensitiveTypeGroupWorkRequestAsync(t, resource, shared.OSOKAsyncPhaseUpdate, "wr-update-stale", "ACCEPTED")
}

func TestSensitiveTypeGroupImmutableCompartmentDriftRejectedBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := newSensitiveTypeGroupResource()
	seedSensitiveTypeGroupStatus(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive)
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	fake := &fakeSensitiveTypeGroupOCI{}
	fake.get = func(context.Context, datasafesdk.GetSensitiveTypeGroupRequest) (datasafesdk.GetSensitiveTypeGroupResponse, error) {
		current := sensitiveTypeGroupBody(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive)
		current.CompartmentId = common.String(testSensitiveTypeGroupCompartmentID)
		return datasafesdk.GetSensitiveTypeGroupResponse{SensitiveTypeGroup: current}, nil
	}
	fake.update = func(context.Context, datasafesdk.UpdateSensitiveTypeGroupRequest) (datasafesdk.UpdateSensitiveTypeGroupResponse, error) {
		t.Fatal("UpdateSensitiveTypeGroup called after immutable compartment drift")
		return datasafesdk.UpdateSensitiveTypeGroupResponse{}, nil
	}

	response, err := newTestSensitiveTypeGroupClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want immutable drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false")
	}
	if !strings.Contains(err.Error(), "compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %q, want compartmentId replacement detail", err.Error())
	}
	requireSensitiveTypeGroupCallCount(t, "UpdateSensitiveTypeGroup", len(fake.updateRequests), 0)
}

func TestSensitiveTypeGroupDeleteRetainsFinalizerUntilConfirmed(t *testing.T) {
	t.Parallel()

	resource := newSensitiveTypeGroupResource()
	seedSensitiveTypeGroupStatus(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive)
	getCalls := 0
	fake := &fakeSensitiveTypeGroupOCI{}
	fake.get = func(_ context.Context, request datasafesdk.GetSensitiveTypeGroupRequest) (datasafesdk.GetSensitiveTypeGroupResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "get sensitiveTypeGroupId", request.SensitiveTypeGroupId, testSensitiveTypeGroupID)
		getCalls++
		state := datasafesdk.SensitiveTypeGroupLifecycleStateActive
		if getCalls >= 3 {
			state = datasafesdk.SensitiveTypeGroupLifecycleStateDeleting
		}
		return datasafesdk.GetSensitiveTypeGroupResponse{SensitiveTypeGroup: sensitiveTypeGroupBody(resource, testSensitiveTypeGroupID, state)}, nil
	}
	fake.delete = func(_ context.Context, request datasafesdk.DeleteSensitiveTypeGroupRequest) (datasafesdk.DeleteSensitiveTypeGroupResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "delete sensitiveTypeGroupId", request.SensitiveTypeGroupId, testSensitiveTypeGroupID)
		return datasafesdk.DeleteSensitiveTypeGroupResponse{
			OpcRequestId:     common.String("opc-delete"),
			OpcWorkRequestId: common.String("wr-delete"),
		}, nil
	}

	deleted, err := newTestSensitiveTypeGroupClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while OCI lifecycle is DELETING")
	}
	requireSensitiveTypeGroupCallCount(t, "DeleteSensitiveTypeGroup", len(fake.deleteRequests), 1)
	requireSensitiveTypeGroupCallCount(t, "GetWorkRequest", len(fake.workRequests), 1)
	requireSensitiveTypeGroupString(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-delete")
	requireSensitiveTypeGroupWorkRequestAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete", "IN_PROGRESS")
}

func TestSensitiveTypeGroupDeleteWorkRequestKeepsFinalizerOnStaleActiveReadback(t *testing.T) {
	t.Parallel()

	resource := newSensitiveTypeGroupResource()
	seedSensitiveTypeGroupStatus(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive)
	fake := &fakeSensitiveTypeGroupOCI{}
	fake.get = func(_ context.Context, request datasafesdk.GetSensitiveTypeGroupRequest) (datasafesdk.GetSensitiveTypeGroupResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "get sensitiveTypeGroupId", request.SensitiveTypeGroupId, testSensitiveTypeGroupID)
		return datasafesdk.GetSensitiveTypeGroupResponse{
			SensitiveTypeGroup: sensitiveTypeGroupBody(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive),
		}, nil
	}
	fake.delete = func(_ context.Context, request datasafesdk.DeleteSensitiveTypeGroupRequest) (datasafesdk.DeleteSensitiveTypeGroupResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "delete sensitiveTypeGroupId", request.SensitiveTypeGroupId, testSensitiveTypeGroupID)
		return datasafesdk.DeleteSensitiveTypeGroupResponse{
			OpcRequestId:     common.String("opc-delete"),
			OpcWorkRequestId: common.String("wr-delete-stale"),
		}, nil
	}
	fake.getWorkRequest = func(_ context.Context, request datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error) {
		requireSensitiveTypeGroupStringPtr(t, "workRequestId", request.WorkRequestId, "wr-delete-stale")
		return datasafesdk.GetWorkRequestResponse{
			WorkRequest: sensitiveTypeGroupWorkRequest("wr-delete-stale", shared.OSOKAsyncPhaseDelete, datasafesdk.WorkRequestStatusSucceeded),
		}, nil
	}

	deleted, err := newTestSensitiveTypeGroupClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while delete work request readback is stale ACTIVE")
	}
	requireSensitiveTypeGroupCallCount(t, "DeleteSensitiveTypeGroup", len(fake.deleteRequests), 1)
	requireSensitiveTypeGroupCallCount(t, "GetWorkRequest", len(fake.workRequests), 1)
	requireSensitiveTypeGroupString(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-delete")
	requireSensitiveTypeGroupWorkRequestAsync(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete-stale", "ACCEPTED")
}

func TestSensitiveTypeGroupDeleteRejectsAuthShapedPreDeleteRead(t *testing.T) {
	t.Parallel()

	resource := newSensitiveTypeGroupResource()
	seedSensitiveTypeGroupStatus(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive)
	fake := &fakeSensitiveTypeGroupOCI{}
	fake.get = func(context.Context, datasafesdk.GetSensitiveTypeGroupRequest) (datasafesdk.GetSensitiveTypeGroupResponse, error) {
		return datasafesdk.GetSensitiveTypeGroupResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
	}
	fake.delete = func(context.Context, datasafesdk.DeleteSensitiveTypeGroupRequest) (datasafesdk.DeleteSensitiveTypeGroupResponse, error) {
		t.Fatal("DeleteSensitiveTypeGroup called after ambiguous pre-delete read")
		return datasafesdk.DeleteSensitiveTypeGroupResponse{}, nil
	}

	deleted, err := newTestSensitiveTypeGroupClient(fake).Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous NotAuthorizedOrNotFound rejection")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained")
	}
	if !strings.Contains(err.Error(), "NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %q, want NotAuthorizedOrNotFound detail", err.Error())
	}
	requireSensitiveTypeGroupString(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-request-id")
	requireSensitiveTypeGroupCallCount(t, "DeleteSensitiveTypeGroup", len(fake.deleteRequests), 0)
}

func TestSensitiveTypeGroupDeleteConfirmsUnambiguousNotFound(t *testing.T) {
	t.Parallel()

	resource := newSensitiveTypeGroupResource()
	seedSensitiveTypeGroupStatus(resource, testSensitiveTypeGroupID, datasafesdk.SensitiveTypeGroupLifecycleStateActive)
	fake := &fakeSensitiveTypeGroupOCI{}
	fake.get = func(context.Context, datasafesdk.GetSensitiveTypeGroupRequest) (datasafesdk.GetSensitiveTypeGroupResponse, error) {
		return datasafesdk.GetSensitiveTypeGroupResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "missing")
	}
	fake.delete = func(context.Context, datasafesdk.DeleteSensitiveTypeGroupRequest) (datasafesdk.DeleteSensitiveTypeGroupResponse, error) {
		t.Fatal("DeleteSensitiveTypeGroup called after confirmed NotFound")
		return datasafesdk.DeleteSensitiveTypeGroupResponse{}, nil
	}

	deleted, err := newTestSensitiveTypeGroupClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion confirmation timestamp")
	}
	requireSensitiveTypeGroupString(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-request-id")
	requireSensitiveTypeGroupCallCount(t, "DeleteSensitiveTypeGroup", len(fake.deleteRequests), 0)
}

func TestSensitiveTypeGroupOCIErrorRecordsRequestID(t *testing.T) {
	t.Parallel()

	resource := newSensitiveTypeGroupResource()
	fake := &fakeSensitiveTypeGroupOCI{}
	fake.create = func(context.Context, datasafesdk.CreateSensitiveTypeGroupRequest) (datasafesdk.CreateSensitiveTypeGroupResponse, error) {
		return datasafesdk.CreateSensitiveTypeGroupResponse{}, errortest.NewServiceError(500, "InternalError", "create failed")
	}

	response, err := newTestSensitiveTypeGroupClient(fake).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want service error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() IsSuccessful = true, want false")
	}
	requireSensitiveTypeGroupString(t, "status.status.opcRequestId", resource.Status.OsokStatus.OpcRequestID, "opc-request-id")
	requireSensitiveTypeGroupCondition(t, resource, shared.Failed)
}

func newSensitiveTypeGroupResource() *datasafev1beta1.SensitiveTypeGroup {
	return &datasafev1beta1.SensitiveTypeGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sensitive-type-group",
			Namespace: "default",
		},
		Spec: datasafev1beta1.SensitiveTypeGroupSpec{
			CompartmentId: testSensitiveTypeGroupCompartmentID,
			DisplayName:   testSensitiveTypeGroupDisplayName,
			Description:   testSensitiveTypeGroupDescription,
			FreeformTags: map[string]string{
				"owner": "data-safe",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func seedSensitiveTypeGroupStatus(
	resource *datasafev1beta1.SensitiveTypeGroup,
	id string,
	state datasafesdk.SensitiveTypeGroupLifecycleStateEnum,
) {
	resource.Status = datasafev1beta1.SensitiveTypeGroupStatus{
		OsokStatus: shared.OSOKStatus{Ocid: shared.OCID(id)},
		Id:         id,
	}
	_ = projectSensitiveTypeGroupStatus(resource, sensitiveTypeGroupBody(resource, id, state))
}

func sensitiveTypeGroupBody(
	resource *datasafev1beta1.SensitiveTypeGroup,
	id string,
	state datasafesdk.SensitiveTypeGroupLifecycleStateEnum,
) datasafesdk.SensitiveTypeGroup {
	count := 3
	created := common.SDKTime{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	updated := common.SDKTime{Time: time.Date(2026, 1, 3, 3, 4, 5, 0, time.UTC)}
	return datasafesdk.SensitiveTypeGroup{
		Id:                 common.String(id),
		DisplayName:        common.String(resource.Spec.DisplayName),
		CompartmentId:      common.String(resource.Spec.CompartmentId),
		LifecycleState:     state,
		TimeCreated:        &created,
		SensitiveTypeCount: common.Int(count),
		Description:        common.String(resource.Spec.Description),
		TimeUpdated:        &updated,
		FreeformTags:       sensitiveTypeGroupStringMap(resource.Spec.FreeformTags),
		DefinedTags:        sensitiveTypeGroupDefinedTags(resource.Spec.DefinedTags),
	}
}

func sensitiveTypeGroupSummary(
	resource *datasafev1beta1.SensitiveTypeGroup,
	id string,
	displayName string,
	state datasafesdk.SensitiveTypeGroupLifecycleStateEnum,
) datasafesdk.SensitiveTypeGroupSummary {
	count := 3
	created := common.SDKTime{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	return datasafesdk.SensitiveTypeGroupSummary{
		Id:                 common.String(id),
		DisplayName:        common.String(displayName),
		CompartmentId:      common.String(resource.Spec.CompartmentId),
		TimeCreated:        &created,
		LifecycleState:     state,
		SensitiveTypeCount: common.Int(count),
		Description:        common.String(resource.Spec.Description),
		FreeformTags:       sensitiveTypeGroupStringMap(resource.Spec.FreeformTags),
		DefinedTags:        sensitiveTypeGroupDefinedTags(resource.Spec.DefinedTags),
	}
}

func sensitiveTypeGroupWorkRequest(
	workRequestID string,
	phase shared.OSOKAsyncPhase,
	status datasafesdk.WorkRequestStatusEnum,
) datasafesdk.WorkRequest {
	operationType := datasafesdk.WorkRequestOperationTypeCreateSensitiveTypeGroup
	actionType := datasafesdk.WorkRequestResourceActionTypeCreated
	switch phase {
	case shared.OSOKAsyncPhaseUpdate:
		operationType = datasafesdk.WorkRequestOperationTypeUpdateSensitiveTypeGroup
		actionType = datasafesdk.WorkRequestResourceActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		operationType = datasafesdk.WorkRequestOperationTypeDeleteSensitiveTypeGroup
		actionType = datasafesdk.WorkRequestResourceActionTypeDeleted
	}
	return datasafesdk.WorkRequest{
		Id:            common.String(workRequestID),
		Status:        status,
		OperationType: operationType,
		Resources: []datasafesdk.WorkRequestResource{
			{
				EntityType: common.String("SensitiveTypeGroup"),
				ActionType: actionType,
				Identifier: common.String(testSensitiveTypeGroupID),
				EntityUri:  common.String("/sensitiveTypeGroups/" + testSensitiveTypeGroupID),
			},
		},
	}
}

func requireSensitiveTypeGroupCreateRequest(
	t *testing.T,
	request datasafesdk.CreateSensitiveTypeGroupRequest,
	resource *datasafev1beta1.SensitiveTypeGroup,
) {
	t.Helper()
	requireSensitiveTypeGroupStringPtr(t, "create compartmentId", request.CompartmentId, resource.Spec.CompartmentId)
	requireSensitiveTypeGroupStringPtr(t, "create displayName", request.DisplayName, resource.Spec.DisplayName)
	requireSensitiveTypeGroupStringPtr(t, "create description", request.Description, resource.Spec.Description)
	if got := request.FreeformTags["owner"]; got != "data-safe" {
		t.Fatalf("create freeformTags[owner] = %q, want data-safe", got)
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("create definedTags Operations.CostCenter = %#v, want 42", got)
	}
}

func requireSensitiveTypeGroupRecordedID(t *testing.T, resource *datasafev1beta1.SensitiveTypeGroup, want string) {
	t.Helper()
	requireSensitiveTypeGroupString(t, "status.id", resource.Status.Id, want)
	requireSensitiveTypeGroupString(t, "status.status.ocid", string(resource.Status.OsokStatus.Ocid), want)
}

func requireSensitiveTypeGroupStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	requireSensitiveTypeGroupString(t, name, *got, want)
}

func requireSensitiveTypeGroupString(t *testing.T, name string, got string, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %q, want %q", name, got, want)
	}
}

func requireSensitiveTypeGroupCallCount(t *testing.T, name string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s calls = %d, want %d", name, got, want)
	}
}

func requireSensitiveTypeGroupCondition(
	t *testing.T,
	resource *datasafev1beta1.SensitiveTypeGroup,
	want shared.OSOKConditionType,
) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions = empty, want %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}

func requireSensitiveTypeGroupAsync(
	t *testing.T,
	resource *datasafev1beta1.SensitiveTypeGroup,
	phase shared.OSOKAsyncPhase,
	rawStatus string,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil")
	}
	if current.Phase != phase {
		t.Fatalf("status.status.async.current.phase = %s, want %s", current.Phase, phase)
	}
	if current.RawStatus != rawStatus {
		t.Fatalf("status.status.async.current.rawStatus = %q, want %q", current.RawStatus, rawStatus)
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.status.async.current.normalizedClass = %s, want pending", current.NormalizedClass)
	}
}

func requireSensitiveTypeGroupWorkRequestAsync(
	t *testing.T,
	resource *datasafev1beta1.SensitiveTypeGroup,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	rawStatus string,
) {
	t.Helper()
	requireSensitiveTypeGroupAsync(t, resource, phase, rawStatus)
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.status.async.current.source = %s, want workrequest", current.Source)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
}
