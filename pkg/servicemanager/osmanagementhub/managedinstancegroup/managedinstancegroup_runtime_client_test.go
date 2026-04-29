/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managedinstancegroup

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

const (
	testManagedInstanceGroupID            = "ocid1.managedinstancegroup.oc1..test"
	testManagedInstanceGroupCompartmentID = "ocid1.compartment.oc1..test"
	testManagedInstanceGroupDisplayName   = "test-managed-instance-group"
	testManagedInstanceGroupDescription   = "test description"
	testSoftwareSourceID                  = "ocid1.softwaresource.oc1..test"
	testManagedInstanceID                 = "ocid1.instance.oc1..test"
)

type fakeManagedInstanceGroupOCIClient struct {
	t *testing.T

	create                 func(context.Context, osmanagementhubsdk.CreateManagedInstanceGroupRequest) (osmanagementhubsdk.CreateManagedInstanceGroupResponse, error)
	get                    func(context.Context, osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error)
	list                   func(context.Context, osmanagementhubsdk.ListManagedInstanceGroupsRequest) (osmanagementhubsdk.ListManagedInstanceGroupsResponse, error)
	update                 func(context.Context, osmanagementhubsdk.UpdateManagedInstanceGroupRequest) (osmanagementhubsdk.UpdateManagedInstanceGroupResponse, error)
	delete                 func(context.Context, osmanagementhubsdk.DeleteManagedInstanceGroupRequest) (osmanagementhubsdk.DeleteManagedInstanceGroupResponse, error)
	attachSoftwareSources  func(context.Context, osmanagementhubsdk.AttachSoftwareSourcesToManagedInstanceGroupRequest) (osmanagementhubsdk.AttachSoftwareSourcesToManagedInstanceGroupResponse, error)
	detachSoftwareSources  func(context.Context, osmanagementhubsdk.DetachSoftwareSourcesFromManagedInstanceGroupRequest) (osmanagementhubsdk.DetachSoftwareSourcesFromManagedInstanceGroupResponse, error)
	attachManagedInstances func(context.Context, osmanagementhubsdk.AttachManagedInstancesToManagedInstanceGroupRequest) (osmanagementhubsdk.AttachManagedInstancesToManagedInstanceGroupResponse, error)
	detachManagedInstances func(context.Context, osmanagementhubsdk.DetachManagedInstancesFromManagedInstanceGroupRequest) (osmanagementhubsdk.DetachManagedInstancesFromManagedInstanceGroupResponse, error)

	createRequests                []osmanagementhubsdk.CreateManagedInstanceGroupRequest
	getRequests                   []osmanagementhubsdk.GetManagedInstanceGroupRequest
	listRequests                  []osmanagementhubsdk.ListManagedInstanceGroupsRequest
	updateRequests                []osmanagementhubsdk.UpdateManagedInstanceGroupRequest
	deleteRequests                []osmanagementhubsdk.DeleteManagedInstanceGroupRequest
	attachSoftwareSourceRequests  []osmanagementhubsdk.AttachSoftwareSourcesToManagedInstanceGroupRequest
	detachSoftwareSourceRequests  []osmanagementhubsdk.DetachSoftwareSourcesFromManagedInstanceGroupRequest
	attachManagedInstanceRequests []osmanagementhubsdk.AttachManagedInstancesToManagedInstanceGroupRequest
	detachManagedInstanceRequests []osmanagementhubsdk.DetachManagedInstancesFromManagedInstanceGroupRequest
}

func (f *fakeManagedInstanceGroupOCIClient) CreateManagedInstanceGroup(
	ctx context.Context,
	request osmanagementhubsdk.CreateManagedInstanceGroupRequest,
) (osmanagementhubsdk.CreateManagedInstanceGroupResponse, error) {
	f.t.Helper()
	f.createRequests = append(f.createRequests, request)
	if f.create == nil {
		f.t.Fatalf("unexpected CreateManagedInstanceGroup request: %#v", request)
	}
	return f.create(ctx, request)
}

func (f *fakeManagedInstanceGroupOCIClient) GetManagedInstanceGroup(
	ctx context.Context,
	request osmanagementhubsdk.GetManagedInstanceGroupRequest,
) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error) {
	f.t.Helper()
	f.getRequests = append(f.getRequests, request)
	if f.get == nil {
		f.t.Fatalf("unexpected GetManagedInstanceGroup request: %#v", request)
	}
	return f.get(ctx, request)
}

func (f *fakeManagedInstanceGroupOCIClient) ListManagedInstanceGroups(
	ctx context.Context,
	request osmanagementhubsdk.ListManagedInstanceGroupsRequest,
) (osmanagementhubsdk.ListManagedInstanceGroupsResponse, error) {
	f.t.Helper()
	f.listRequests = append(f.listRequests, request)
	if f.list == nil {
		f.t.Fatalf("unexpected ListManagedInstanceGroups request: %#v", request)
	}
	return f.list(ctx, request)
}

func (f *fakeManagedInstanceGroupOCIClient) UpdateManagedInstanceGroup(
	ctx context.Context,
	request osmanagementhubsdk.UpdateManagedInstanceGroupRequest,
) (osmanagementhubsdk.UpdateManagedInstanceGroupResponse, error) {
	f.t.Helper()
	f.updateRequests = append(f.updateRequests, request)
	if f.update == nil {
		f.t.Fatalf("unexpected UpdateManagedInstanceGroup request: %#v", request)
	}
	return f.update(ctx, request)
}

func (f *fakeManagedInstanceGroupOCIClient) DeleteManagedInstanceGroup(
	ctx context.Context,
	request osmanagementhubsdk.DeleteManagedInstanceGroupRequest,
) (osmanagementhubsdk.DeleteManagedInstanceGroupResponse, error) {
	f.t.Helper()
	f.deleteRequests = append(f.deleteRequests, request)
	if f.delete == nil {
		f.t.Fatalf("unexpected DeleteManagedInstanceGroup request: %#v", request)
	}
	return f.delete(ctx, request)
}

func (f *fakeManagedInstanceGroupOCIClient) AttachSoftwareSourcesToManagedInstanceGroup(
	ctx context.Context,
	request osmanagementhubsdk.AttachSoftwareSourcesToManagedInstanceGroupRequest,
) (osmanagementhubsdk.AttachSoftwareSourcesToManagedInstanceGroupResponse, error) {
	f.t.Helper()
	f.attachSoftwareSourceRequests = append(f.attachSoftwareSourceRequests, request)
	if f.attachSoftwareSources == nil {
		f.t.Fatalf("unexpected AttachSoftwareSourcesToManagedInstanceGroup request: %#v", request)
	}
	return f.attachSoftwareSources(ctx, request)
}

func (f *fakeManagedInstanceGroupOCIClient) DetachSoftwareSourcesFromManagedInstanceGroup(
	ctx context.Context,
	request osmanagementhubsdk.DetachSoftwareSourcesFromManagedInstanceGroupRequest,
) (osmanagementhubsdk.DetachSoftwareSourcesFromManagedInstanceGroupResponse, error) {
	f.t.Helper()
	f.detachSoftwareSourceRequests = append(f.detachSoftwareSourceRequests, request)
	if f.detachSoftwareSources == nil {
		f.t.Fatalf("unexpected DetachSoftwareSourcesFromManagedInstanceGroup request: %#v", request)
	}
	return f.detachSoftwareSources(ctx, request)
}

func (f *fakeManagedInstanceGroupOCIClient) AttachManagedInstancesToManagedInstanceGroup(
	ctx context.Context,
	request osmanagementhubsdk.AttachManagedInstancesToManagedInstanceGroupRequest,
) (osmanagementhubsdk.AttachManagedInstancesToManagedInstanceGroupResponse, error) {
	f.t.Helper()
	f.attachManagedInstanceRequests = append(f.attachManagedInstanceRequests, request)
	if f.attachManagedInstances == nil {
		f.t.Fatalf("unexpected AttachManagedInstancesToManagedInstanceGroup request: %#v", request)
	}
	return f.attachManagedInstances(ctx, request)
}

func (f *fakeManagedInstanceGroupOCIClient) DetachManagedInstancesFromManagedInstanceGroup(
	ctx context.Context,
	request osmanagementhubsdk.DetachManagedInstancesFromManagedInstanceGroupRequest,
) (osmanagementhubsdk.DetachManagedInstancesFromManagedInstanceGroupResponse, error) {
	f.t.Helper()
	f.detachManagedInstanceRequests = append(f.detachManagedInstanceRequests, request)
	if f.detachManagedInstances == nil {
		f.t.Fatalf("unexpected DetachManagedInstancesFromManagedInstanceGroup request: %#v", request)
	}
	return f.detachManagedInstances(ctx, request)
}

func TestManagedInstanceGroupServiceClientCreateProjectsStatus(t *testing.T) {
	resource := newManagedInstanceGroupTestResource()
	fake := &fakeManagedInstanceGroupOCIClient{t: t}
	fake.list = func(_ context.Context, request osmanagementhubsdk.ListManagedInstanceGroupsRequest) (osmanagementhubsdk.ListManagedInstanceGroupsResponse, error) {
		requireStringPtr(t, "list compartmentId", request.CompartmentId, testManagedInstanceGroupCompartmentID)
		if request.DisplayName != nil {
			t.Fatalf("list displayName = %#v, want nil because SDK expects []string", request.DisplayName)
		}
		return osmanagementhubsdk.ListManagedInstanceGroupsResponse{}, nil
	}
	fake.create = func(_ context.Context, request osmanagementhubsdk.CreateManagedInstanceGroupRequest) (osmanagementhubsdk.CreateManagedInstanceGroupResponse, error) {
		body := request.CreateManagedInstanceGroupDetails
		requireStringPtr(t, "create displayName", body.DisplayName, testManagedInstanceGroupDisplayName)
		requireStringPtr(t, "create compartmentId", body.CompartmentId, testManagedInstanceGroupCompartmentID)
		if got := body.OsFamily; got != osmanagementhubsdk.OsFamilyOracleLinux8 {
			t.Fatalf("create osFamily = %q, want %q", got, osmanagementhubsdk.OsFamilyOracleLinux8)
		}
		if body.AutonomousSettings == nil || body.AutonomousSettings.IsDataCollectionAuthorized == nil ||
			*body.AutonomousSettings.IsDataCollectionAuthorized {
			t.Fatalf("create autonomousSettings.isDataCollectionAuthorized = %#v, want explicit false", body.AutonomousSettings)
		}
		return osmanagementhubsdk.CreateManagedInstanceGroupResponse{
			ManagedInstanceGroup: managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateCreating),
			OpcRequestId:         common.String("opc-create-1"),
		}, nil
	}
	fake.get = func(_ context.Context, request osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error) {
		requireStringPtr(t, "get managedInstanceGroupId", request.ManagedInstanceGroupId, testManagedInstanceGroupID)
		return osmanagementhubsdk.GetManagedInstanceGroupResponse{
			ManagedInstanceGroup: managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive),
			OpcRequestId:         common.String("opc-get-1"),
		}, nil
	}

	response, err := newManagedInstanceGroupTestClient(fake).CreateOrUpdate(context.Background(), resource, managedInstanceGroupRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful non-requeue", response)
	}
	requireManagedInstanceGroupStatus(t, resource, testManagedInstanceGroupID, "opc-create-1")
}

func TestManagedInstanceGroupServiceClientBindsFromPaginatedList(t *testing.T) {
	resource := newManagedInstanceGroupTestResource()
	fake := &fakeManagedInstanceGroupOCIClient{t: t}
	fake.list = func(_ context.Context, request osmanagementhubsdk.ListManagedInstanceGroupsRequest) (osmanagementhubsdk.ListManagedInstanceGroupsResponse, error) {
		switch len(fake.listRequests) {
		case 1:
			if request.Page != nil {
				t.Fatalf("first list page = %q, want nil", *request.Page)
			}
			return osmanagementhubsdk.ListManagedInstanceGroupsResponse{
				ManagedInstanceGroupCollection: osmanagementhubsdk.ManagedInstanceGroupCollection{
					Items: []osmanagementhubsdk.ManagedInstanceGroupSummary{
						managedInstanceGroupSummary("ocid1.managedinstancegroup.oc1..other", "other"),
					},
				},
				OpcNextPage: common.String("page-2"),
			}, nil
		case 2:
			requireStringPtr(t, "second list page", request.Page, "page-2")
			return osmanagementhubsdk.ListManagedInstanceGroupsResponse{
				ManagedInstanceGroupCollection: osmanagementhubsdk.ManagedInstanceGroupCollection{
					Items: []osmanagementhubsdk.ManagedInstanceGroupSummary{
						managedInstanceGroupSummary(testManagedInstanceGroupID, testManagedInstanceGroupDisplayName),
					},
				},
			}, nil
		default:
			t.Fatalf("unexpected list call %d", len(fake.listRequests))
			return osmanagementhubsdk.ListManagedInstanceGroupsResponse{}, nil
		}
	}
	fake.get = func(_ context.Context, request osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error) {
		requireStringPtr(t, "get managedInstanceGroupId", request.ManagedInstanceGroupId, testManagedInstanceGroupID)
		return osmanagementhubsdk.GetManagedInstanceGroupResponse{
			ManagedInstanceGroup: managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive),
			OpcRequestId:         common.String("opc-get-1"),
		}, nil
	}

	response, err := newManagedInstanceGroupTestClient(fake).CreateOrUpdate(context.Background(), resource, managedInstanceGroupRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful bind", response)
	}
	if len(fake.createRequests) != 0 {
		t.Fatalf("create requests = %d, want 0 after bind", len(fake.createRequests))
	}
	requireManagedInstanceGroupStatus(t, resource, testManagedInstanceGroupID, "")
}

func TestManagedInstanceGroupServiceClientNoOpObservedReadDoesNotUpdate(t *testing.T) {
	resource := newManagedInstanceGroupTestResource()
	resource.Status.Id = testManagedInstanceGroupID
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagedInstanceGroupID)
	fake := &fakeManagedInstanceGroupOCIClient{t: t}
	fake.get = func(_ context.Context, request osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error) {
		requireStringPtr(t, "get managedInstanceGroupId", request.ManagedInstanceGroupId, testManagedInstanceGroupID)
		return osmanagementhubsdk.GetManagedInstanceGroupResponse{
			ManagedInstanceGroup: managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive),
			OpcRequestId:         common.String("opc-get-1"),
		}, nil
	}

	response, err := newManagedInstanceGroupTestClient(fake).CreateOrUpdate(context.Background(), resource, managedInstanceGroupRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful no-op", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0", len(fake.updateRequests))
	}
	requireManagedInstanceGroupStatus(t, resource, testManagedInstanceGroupID, "")
}

func TestManagedInstanceGroupServiceClientRejectsForceNewDriftBeforeUpdateOrMembership(t *testing.T) {
	resource := newManagedInstanceGroupTestResource()
	resource.Status.Id = testManagedInstanceGroupID
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagedInstanceGroupID)
	resource.Spec.CompartmentId = "ocid1.compartment.oc1..moved"
	resource.Status.CompartmentId = resource.Spec.CompartmentId
	resource.Spec.SoftwareSourceIds = []string{"ocid1.softwaresource.oc1..would-attach"}
	resource.Spec.ManagedInstanceIds = []string{"ocid1.instance.oc1..would-attach"}

	fake := &fakeManagedInstanceGroupOCIClient{t: t}
	fake.get = func(_ context.Context, request osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error) {
		requireStringPtr(t, "get managedInstanceGroupId", request.ManagedInstanceGroupId, testManagedInstanceGroupID)
		return osmanagementhubsdk.GetManagedInstanceGroupResponse{
			ManagedInstanceGroup: managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive),
			OpcRequestId:         common.String("opc-get-1"),
		}, nil
	}

	response, err := newManagedInstanceGroupTestClient(fake).CreateOrUpdate(context.Background(), resource, managedInstanceGroupRequest(resource))
	if err == nil || !strings.Contains(err.Error(), "require replacement when compartmentId changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want force-new compartmentId drift rejection", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if len(fake.getRequests) != 1 {
		t.Fatalf("get requests = %d, want 1 live readback before drift rejection", len(fake.getRequests))
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 after force-new drift rejection", len(fake.updateRequests))
	}
	requireActionRequestCount(t, "detach software sources", len(fake.detachSoftwareSourceRequests), 0)
	requireActionRequestCount(t, "attach software sources", len(fake.attachSoftwareSourceRequests), 0)
	requireActionRequestCount(t, "detach managed instances", len(fake.detachManagedInstanceRequests), 0)
	requireActionRequestCount(t, "attach managed instances", len(fake.attachManagedInstanceRequests), 0)
}

func TestManagedInstanceGroupServiceClientMutableScalarUpdate(t *testing.T) {
	resource := newManagedInstanceGroupTestResource()
	resource.Status.Id = testManagedInstanceGroupID
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagedInstanceGroupID)
	resource.Spec.DisplayName = "renamed"
	resource.Spec.Description = "updated description"
	resource.Spec.NotificationTopicId = "ocid1.onstopic.oc1..updated"
	resource.Spec.AutonomousSettings.IsDataCollectionAuthorized = true

	updated := managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive)
	updated.DisplayName = common.String(resource.Spec.DisplayName)
	updated.Description = common.String(resource.Spec.Description)
	updated.NotificationTopicId = common.String(resource.Spec.NotificationTopicId)
	updated.AutonomousSettings = &osmanagementhubsdk.AutonomousSettings{IsDataCollectionAuthorized: common.Bool(true)}

	runManagedInstanceGroupMutableUpdateTest(t, resource, updated, func(t *testing.T, request osmanagementhubsdk.UpdateManagedInstanceGroupRequest) {
		t.Helper()
		requireStringPtr(t, "update displayName", request.DisplayName, resource.Spec.DisplayName)
		requireStringPtr(t, "update description", request.Description, resource.Spec.Description)
		requireStringPtr(t, "update notificationTopicId", request.NotificationTopicId, resource.Spec.NotificationTopicId)
		if request.AutonomousSettings == nil || request.AutonomousSettings.IsDataCollectionAuthorized == nil ||
			!*request.AutonomousSettings.IsDataCollectionAuthorized {
			t.Fatalf("update autonomousSettings.isDataCollectionAuthorized = %#v, want true", request.AutonomousSettings)
		}
	})
	if resource.Status.DisplayName != resource.Spec.DisplayName {
		t.Fatalf("status.displayName = %q, want %q", resource.Status.DisplayName, resource.Spec.DisplayName)
	}
}

func TestManagedInstanceGroupServiceClientMutableTagUpdate(t *testing.T) {
	resource := newManagedInstanceGroupTestResource()
	resource.Status.Id = testManagedInstanceGroupID
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagedInstanceGroupID)
	resource.Spec.FreeformTags = map[string]string{"env": "test"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}

	updated := managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive)
	updated.FreeformTags = map[string]string{"env": "test"}
	updated.DefinedTags = map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}}

	runManagedInstanceGroupMutableUpdateTest(t, resource, updated, func(t *testing.T, request osmanagementhubsdk.UpdateManagedInstanceGroupRequest) {
		t.Helper()
		if request.FreeformTags["env"] != "test" {
			t.Fatalf("update freeformTags = %#v, want env tag", request.FreeformTags)
		}
		if request.DefinedTags["Operations"]["CostCenter"] != "42" {
			t.Fatalf("update definedTags = %#v, want Operations.CostCenter", request.DefinedTags)
		}
	})
	if resource.Status.FreeformTags["env"] != "test" {
		t.Fatalf("status.freeformTags = %#v, want env tag", resource.Status.FreeformTags)
	}
}

func runManagedInstanceGroupMutableUpdateTest(
	t *testing.T,
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
	updated osmanagementhubsdk.ManagedInstanceGroup,
	assertUpdate func(*testing.T, osmanagementhubsdk.UpdateManagedInstanceGroupRequest),
) {
	t.Helper()
	getResponses := []osmanagementhubsdk.ManagedInstanceGroup{
		managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive),
		updated,
	}

	fake := &fakeManagedInstanceGroupOCIClient{t: t}
	fake.get = func(_ context.Context, request osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error) {
		requireStringPtr(t, "get managedInstanceGroupId", request.ManagedInstanceGroupId, testManagedInstanceGroupID)
		if len(getResponses) == 0 {
			t.Fatal("unexpected GetManagedInstanceGroup call")
		}
		next := getResponses[0]
		getResponses = getResponses[1:]
		return osmanagementhubsdk.GetManagedInstanceGroupResponse{ManagedInstanceGroup: next}, nil
	}
	fake.update = func(_ context.Context, request osmanagementhubsdk.UpdateManagedInstanceGroupRequest) (osmanagementhubsdk.UpdateManagedInstanceGroupResponse, error) {
		requireStringPtr(t, "update managedInstanceGroupId", request.ManagedInstanceGroupId, testManagedInstanceGroupID)
		assertUpdate(t, request)
		return osmanagementhubsdk.UpdateManagedInstanceGroupResponse{
			ManagedInstanceGroup: updated,
			OpcRequestId:         common.String("opc-update-1"),
		}, nil
	}

	response, err := newManagedInstanceGroupTestClient(fake).CreateOrUpdate(context.Background(), resource, managedInstanceGroupRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful update", response)
	}
	if len(fake.updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(fake.updateRequests))
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-update-1" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update-1", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestManagedInstanceGroupServiceClientReconcilesMembershipWithCustomActions(t *testing.T) {
	resource := newManagedInstanceGroupTestResource()
	resource.Status.Id = testManagedInstanceGroupID
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagedInstanceGroupID)
	resource.Spec.SoftwareSourceIds = []string{testSoftwareSourceID, "ocid1.softwaresource.oc1..attach"}
	resource.Spec.ManagedInstanceIds = []string{"ocid1.instance.oc1..attach"}

	current := managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive)
	current.SoftwareSourceIds = []osmanagementhubsdk.SoftwareSourceDetails{
		{Id: common.String(testSoftwareSourceID)},
		{Id: common.String("ocid1.softwaresource.oc1..detach")},
	}
	current.ManagedInstanceIds = []string{"ocid1.instance.oc1..detach"}

	fake := &fakeManagedInstanceGroupOCIClient{t: t}
	fake.get = func(_ context.Context, request osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error) {
		requireStringPtr(t, "get managedInstanceGroupId", request.ManagedInstanceGroupId, testManagedInstanceGroupID)
		return osmanagementhubsdk.GetManagedInstanceGroupResponse{ManagedInstanceGroup: current}, nil
	}
	fake.detachSoftwareSources = func(_ context.Context, request osmanagementhubsdk.DetachSoftwareSourcesFromManagedInstanceGroupRequest) (osmanagementhubsdk.DetachSoftwareSourcesFromManagedInstanceGroupResponse, error) {
		requireStringPtr(t, "detach software managedInstanceGroupId", request.ManagedInstanceGroupId, testManagedInstanceGroupID)
		requireStringSlice(t, "detach software sources", request.SoftwareSources, []string{"ocid1.softwaresource.oc1..detach"})
		requireRetryToken(t, "detach software sources", request.OpcRetryToken)
		return osmanagementhubsdk.DetachSoftwareSourcesFromManagedInstanceGroupResponse{OpcRequestId: common.String("opc-detach-software")}, nil
	}
	fake.attachSoftwareSources = func(_ context.Context, request osmanagementhubsdk.AttachSoftwareSourcesToManagedInstanceGroupRequest) (osmanagementhubsdk.AttachSoftwareSourcesToManagedInstanceGroupResponse, error) {
		requireStringPtr(t, "attach software managedInstanceGroupId", request.ManagedInstanceGroupId, testManagedInstanceGroupID)
		requireStringSlice(t, "attach software sources", request.SoftwareSources, []string{"ocid1.softwaresource.oc1..attach"})
		requireRetryToken(t, "attach software sources", request.OpcRetryToken)
		return osmanagementhubsdk.AttachSoftwareSourcesToManagedInstanceGroupResponse{
			OpcRequestId:     common.String("opc-attach-software"),
			OpcWorkRequestId: common.String("wr-attach-software"),
		}, nil
	}
	fake.detachManagedInstances = func(_ context.Context, request osmanagementhubsdk.DetachManagedInstancesFromManagedInstanceGroupRequest) (osmanagementhubsdk.DetachManagedInstancesFromManagedInstanceGroupResponse, error) {
		requireStringPtr(t, "detach instances managedInstanceGroupId", request.ManagedInstanceGroupId, testManagedInstanceGroupID)
		requireStringSlice(t, "detach managed instances", request.ManagedInstances, []string{"ocid1.instance.oc1..detach"})
		requireRetryToken(t, "detach managed instances", request.OpcRetryToken)
		return osmanagementhubsdk.DetachManagedInstancesFromManagedInstanceGroupResponse{OpcRequestId: common.String("opc-detach-instances")}, nil
	}
	fake.attachManagedInstances = func(_ context.Context, request osmanagementhubsdk.AttachManagedInstancesToManagedInstanceGroupRequest) (osmanagementhubsdk.AttachManagedInstancesToManagedInstanceGroupResponse, error) {
		requireStringPtr(t, "attach instances managedInstanceGroupId", request.ManagedInstanceGroupId, testManagedInstanceGroupID)
		requireStringSlice(t, "attach managed instances", request.ManagedInstances, []string{"ocid1.instance.oc1..attach"})
		requireRetryToken(t, "attach managed instances", request.OpcRetryToken)
		return osmanagementhubsdk.AttachManagedInstancesToManagedInstanceGroupResponse{
			OpcRequestId:     common.String("opc-attach-instances"),
			OpcWorkRequestId: common.String("wr-attach-instances"),
		}, nil
	}

	response, err := newManagedInstanceGroupTestClient(fake).CreateOrUpdate(context.Background(), resource, managedInstanceGroupRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue after membership action", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 for membership-only change", len(fake.updateRequests))
	}
	requireActionRequestCount(t, "detach software sources", len(fake.detachSoftwareSourceRequests), 1)
	requireActionRequestCount(t, "attach software sources", len(fake.attachSoftwareSourceRequests), 1)
	requireActionRequestCount(t, "detach managed instances", len(fake.detachManagedInstanceRequests), 1)
	requireActionRequestCount(t, "attach managed instances", len(fake.attachManagedInstanceRequests), 1)
	if resource.Status.OsokStatus.OpcRequestID != "opc-attach-instances" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-attach-instances", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseUpdate ||
		resource.Status.OsokStatus.Async.Current.WorkRequestID != "wr-attach-instances" {
		t.Fatalf("status.async.current = %#v, want update work request wr-attach-instances", resource.Status.OsokStatus.Async.Current)
	}
}

func TestManagedInstanceGroupServiceClientOmittedMembershipFieldsDoNotDetach(t *testing.T) {
	resource := newManagedInstanceGroupTestResource()
	resource.Status.Id = testManagedInstanceGroupID
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagedInstanceGroupID)
	resource.Spec.SoftwareSourceIds = nil
	resource.Spec.ManagedInstanceIds = nil

	current := managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive)
	current.SoftwareSourceIds = []osmanagementhubsdk.SoftwareSourceDetails{{Id: common.String("ocid1.softwaresource.oc1..existing")}}
	current.ManagedInstanceIds = []string{"ocid1.instance.oc1..existing"}
	fake := &fakeManagedInstanceGroupOCIClient{t: t}
	fake.get = func(_ context.Context, _ osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error) {
		return osmanagementhubsdk.GetManagedInstanceGroupResponse{ManagedInstanceGroup: current}, nil
	}

	response, err := newManagedInstanceGroupTestClient(fake).CreateOrUpdate(context.Background(), resource, managedInstanceGroupRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful observe", response)
	}
	requireActionRequestCount(t, "detach software sources", len(fake.detachSoftwareSourceRequests), 0)
	requireActionRequestCount(t, "detach managed instances", len(fake.detachManagedInstanceRequests), 0)
}

func TestManagedInstanceGroupServiceClientDeleteWaitsForLifecycleConfirmation(t *testing.T) {
	resource := newManagedInstanceGroupTestResource()
	resource.Status.Id = testManagedInstanceGroupID
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagedInstanceGroupID)

	getResponses := []osmanagementhubsdk.ManagedInstanceGroup{
		managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive),
		managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive),
		managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateDeleting),
	}
	fake := &fakeManagedInstanceGroupOCIClient{t: t}
	fake.get = func(_ context.Context, _ osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error) {
		if len(getResponses) == 0 {
			t.Fatal("unexpected GetManagedInstanceGroup call")
		}
		next := getResponses[0]
		getResponses = getResponses[1:]
		return osmanagementhubsdk.GetManagedInstanceGroupResponse{ManagedInstanceGroup: next}, nil
	}
	fake.delete = func(_ context.Context, request osmanagementhubsdk.DeleteManagedInstanceGroupRequest) (osmanagementhubsdk.DeleteManagedInstanceGroupResponse, error) {
		requireStringPtr(t, "delete managedInstanceGroupId", request.ManagedInstanceGroupId, testManagedInstanceGroupID)
		return osmanagementhubsdk.DeleteManagedInstanceGroupResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}

	deleted, err := newManagedInstanceGroupTestClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false while lifecycle is DELETING")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-1", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.Async.Current == nil ||
		resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current = %#v, want delete phase", resource.Status.OsokStatus.Async.Current)
	}
}

func TestManagedInstanceGroupServiceClientDeleteReleasesAfterNotFoundConfirmation(t *testing.T) {
	resource := newManagedInstanceGroupTestResource()
	resource.Status.Id = testManagedInstanceGroupID
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagedInstanceGroupID)

	getCalls := 0
	fake := &fakeManagedInstanceGroupOCIClient{t: t}
	fake.get = func(_ context.Context, _ osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error) {
		getCalls++
		if getCalls <= 2 {
			return osmanagementhubsdk.GetManagedInstanceGroupResponse{
				ManagedInstanceGroup: managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive),
			}, nil
		}
		return osmanagementhubsdk.GetManagedInstanceGroupResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "not found")
	}
	fake.delete = func(_ context.Context, _ osmanagementhubsdk.DeleteManagedInstanceGroupRequest) (osmanagementhubsdk.DeleteManagedInstanceGroupResponse, error) {
		return osmanagementhubsdk.DeleteManagedInstanceGroupResponse{OpcRequestId: common.String("opc-delete-1")}, nil
	}
	fake.list = func(_ context.Context, _ osmanagementhubsdk.ListManagedInstanceGroupsRequest) (osmanagementhubsdk.ListManagedInstanceGroupsResponse, error) {
		return osmanagementhubsdk.ListManagedInstanceGroupsResponse{}, nil
	}

	deleted, err := newManagedInstanceGroupTestClient(fake).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after unambiguous NotFound confirmation")
	}
}

func TestManagedInstanceGroupServiceClientDeleteRejectsAuthShapedNotFound(t *testing.T) {
	resource := newManagedInstanceGroupTestResource()
	resource.Status.Id = testManagedInstanceGroupID
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagedInstanceGroupID)

	fake := &fakeManagedInstanceGroupOCIClient{t: t}
	fake.get = func(_ context.Context, _ osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error) {
		return osmanagementhubsdk.GetManagedInstanceGroupResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "ambiguous")
	}

	deleted, err := newManagedInstanceGroupTestClient(fake).Delete(context.Background(), resource)
	if err == nil || !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want auth-shaped 404 blocker", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false on auth-shaped 404")
	}
	if len(fake.deleteRequests) != 0 {
		t.Fatalf("delete requests = %d, want 0 after pre-delete auth-shaped read", len(fake.deleteRequests))
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestManagedInstanceGroupServiceClientCreateErrorCapturesOpcRequestID(t *testing.T) {
	resource := newManagedInstanceGroupTestResource()
	fake := &fakeManagedInstanceGroupOCIClient{t: t}
	fake.list = func(_ context.Context, _ osmanagementhubsdk.ListManagedInstanceGroupsRequest) (osmanagementhubsdk.ListManagedInstanceGroupsResponse, error) {
		return osmanagementhubsdk.ListManagedInstanceGroupsResponse{}, nil
	}
	fake.create = func(_ context.Context, _ osmanagementhubsdk.CreateManagedInstanceGroupRequest) (osmanagementhubsdk.CreateManagedInstanceGroupResponse, error) {
		return osmanagementhubsdk.CreateManagedInstanceGroupResponse{}, errortest.NewServiceError(500, "InternalError", "create failed")
	}

	response, err := newManagedInstanceGroupTestClient(fake).CreateOrUpdate(context.Background(), resource, managedInstanceGroupRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful", response)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-request-id" {
		t.Fatalf("status.opcRequestId = %q, want opc-request-id", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestManagedInstanceGroupServiceClientObservedUpdatingLifecycleRequeues(t *testing.T) {
	resource := newManagedInstanceGroupTestResource()
	resource.Status.Id = testManagedInstanceGroupID
	resource.Status.OsokStatus.Ocid = shared.OCID(testManagedInstanceGroupID)
	fake := &fakeManagedInstanceGroupOCIClient{t: t}
	fake.get = func(_ context.Context, _ osmanagementhubsdk.GetManagedInstanceGroupRequest) (osmanagementhubsdk.GetManagedInstanceGroupResponse, error) {
		return osmanagementhubsdk.GetManagedInstanceGroupResponse{
			ManagedInstanceGroup: managedInstanceGroupSDK(testManagedInstanceGroupID, osmanagementhubsdk.ManagedInstanceGroupLifecycleStateUpdating),
		}, nil
	}

	response, err := newManagedInstanceGroupTestClient(fake).CreateOrUpdate(context.Background(), resource, managedInstanceGroupRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful requeue while UPDATING", response)
	}
	if len(fake.updateRequests) != 0 {
		t.Fatalf("update requests = %d, want 0 while lifecycle is UPDATING", len(fake.updateRequests))
	}
}

func newManagedInstanceGroupTestClient(fake *fakeManagedInstanceGroupOCIClient) ManagedInstanceGroupServiceClient {
	return newManagedInstanceGroupServiceClientWithOCIClient(loggerutil.OSOKLogger{}, fake)
}

func managedInstanceGroupRequest(resource *osmanagementhubv1beta1.ManagedInstanceGroup) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func newManagedInstanceGroupTestResource() *osmanagementhubv1beta1.ManagedInstanceGroup {
	return &osmanagementhubv1beta1.ManagedInstanceGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "managed-instance-group",
			Namespace: "default",
		},
		Spec: osmanagementhubv1beta1.ManagedInstanceGroupSpec{
			DisplayName:        testManagedInstanceGroupDisplayName,
			CompartmentId:      testManagedInstanceGroupCompartmentID,
			OsFamily:           string(osmanagementhubsdk.OsFamilyOracleLinux8),
			ArchType:           string(osmanagementhubsdk.ArchTypeX8664),
			VendorName:         string(osmanagementhubsdk.VendorNameOracle),
			Description:        testManagedInstanceGroupDescription,
			Location:           string(osmanagementhubsdk.ManagedInstanceLocationOciCompute),
			SoftwareSourceIds:  []string{testSoftwareSourceID},
			ManagedInstanceIds: []string{testManagedInstanceID},
		},
	}
}

func managedInstanceGroupSDK(
	id string,
	lifecycle osmanagementhubsdk.ManagedInstanceGroupLifecycleStateEnum,
) osmanagementhubsdk.ManagedInstanceGroup {
	return osmanagementhubsdk.ManagedInstanceGroup{
		Id:                 common.String(id),
		CompartmentId:      common.String(testManagedInstanceGroupCompartmentID),
		LifecycleState:     lifecycle,
		DisplayName:        common.String(testManagedInstanceGroupDisplayName),
		Description:        common.String(testManagedInstanceGroupDescription),
		OsFamily:           osmanagementhubsdk.OsFamilyOracleLinux8,
		ArchType:           osmanagementhubsdk.ArchTypeX8664,
		VendorName:         osmanagementhubsdk.VendorNameOracle,
		Location:           osmanagementhubsdk.ManagedInstanceLocationOciCompute,
		SoftwareSourceIds:  []osmanagementhubsdk.SoftwareSourceDetails{{Id: common.String(testSoftwareSourceID)}},
		ManagedInstanceIds: []string{testManagedInstanceID},
	}
}

func managedInstanceGroupSummary(id string, displayName string) osmanagementhubsdk.ManagedInstanceGroupSummary {
	return osmanagementhubsdk.ManagedInstanceGroupSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(testManagedInstanceGroupCompartmentID),
		LifecycleState: osmanagementhubsdk.ManagedInstanceGroupLifecycleStateActive,
		DisplayName:    common.String(displayName),
		Description:    common.String(testManagedInstanceGroupDescription),
		OsFamily:       osmanagementhubsdk.OsFamilyOracleLinux8,
		ArchType:       osmanagementhubsdk.ArchTypeX8664,
		VendorName:     osmanagementhubsdk.VendorNameOracle,
		Location:       osmanagementhubsdk.ManagedInstanceLocationOciCompute,
	}
}

func requireStringPtr(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func requireStringSlice(t *testing.T, name string, got []string, want []string) {
	t.Helper()
	got = managedInstanceGroupNormalizedStrings(got)
	want = managedInstanceGroupNormalizedStrings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
}

func requireRetryToken(t *testing.T, name string, got *string) {
	t.Helper()
	if got == nil || strings.TrimSpace(*got) == "" {
		t.Fatalf("%s retry token = %#v, want non-empty", name, got)
	}
}

func requireActionRequestCount(t *testing.T, name string, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s requests = %d, want %d", name, got, want)
	}
}

func requireManagedInstanceGroupStatus(
	t *testing.T,
	resource *osmanagementhubv1beta1.ManagedInstanceGroup,
	wantID string,
	wantOpcRequestID string,
) {
	t.Helper()
	if resource.Status.Id != wantID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, wantID)
	}
	if string(resource.Status.OsokStatus.Ocid) != wantID {
		t.Fatalf("status.status.ocid = %q, want %q", resource.Status.OsokStatus.Ocid, wantID)
	}
	if resource.Status.OsokStatus.OpcRequestID != wantOpcRequestID {
		t.Fatalf("status.status.opcRequestId = %q, want %q", resource.Status.OsokStatus.OpcRequestID, wantOpcRequestID)
	}
}
