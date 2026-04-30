/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package iotdomaingroup

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	iotsdk "github.com/oracle/oci-go-sdk/v65/iot"
	iotv1beta1 "github.com/oracle/oci-service-operator/api/iot/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testIotDomainGroupID            = "ocid1.iotdomaingroup.oc1..group"
	testIotDomainGroupCompartmentID = "ocid1.compartment.oc1..compartment"
	testIotDomainGroupName          = "iot-domain-group"
)

type fakeIotDomainGroupOCIClient struct {
	createFn func(context.Context, iotsdk.CreateIotDomainGroupRequest) (iotsdk.CreateIotDomainGroupResponse, error)
	getFn    func(context.Context, iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error)
	listFn   func(context.Context, iotsdk.ListIotDomainGroupsRequest) (iotsdk.ListIotDomainGroupsResponse, error)
	updateFn func(context.Context, iotsdk.UpdateIotDomainGroupRequest) (iotsdk.UpdateIotDomainGroupResponse, error)
	deleteFn func(context.Context, iotsdk.DeleteIotDomainGroupRequest) (iotsdk.DeleteIotDomainGroupResponse, error)
}

func (f *fakeIotDomainGroupOCIClient) CreateIotDomainGroup(ctx context.Context, req iotsdk.CreateIotDomainGroupRequest) (iotsdk.CreateIotDomainGroupResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return iotsdk.CreateIotDomainGroupResponse{}, nil
}

func (f *fakeIotDomainGroupOCIClient) GetIotDomainGroup(ctx context.Context, req iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return iotsdk.GetIotDomainGroupResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "iot domain group is missing")
}

func (f *fakeIotDomainGroupOCIClient) ListIotDomainGroups(ctx context.Context, req iotsdk.ListIotDomainGroupsRequest) (iotsdk.ListIotDomainGroupsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return iotsdk.ListIotDomainGroupsResponse{}, nil
}

func (f *fakeIotDomainGroupOCIClient) UpdateIotDomainGroup(ctx context.Context, req iotsdk.UpdateIotDomainGroupRequest) (iotsdk.UpdateIotDomainGroupResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return iotsdk.UpdateIotDomainGroupResponse{}, nil
}

func (f *fakeIotDomainGroupOCIClient) DeleteIotDomainGroup(ctx context.Context, req iotsdk.DeleteIotDomainGroupRequest) (iotsdk.DeleteIotDomainGroupResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return iotsdk.DeleteIotDomainGroupResponse{}, nil
}

func newTestIotDomainGroupClient(client iotDomainGroupOCIClient) IotDomainGroupServiceClient {
	return newIotDomainGroupServiceClientWithOCIClient(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		client,
	)
}

func makeIotDomainGroupResource() *iotv1beta1.IotDomainGroup {
	return &iotv1beta1.IotDomainGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testIotDomainGroupName,
			Namespace: "default",
		},
		Spec: iotv1beta1.IotDomainGroupSpec{
			CompartmentId: testIotDomainGroupCompartmentID,
			Type:          string(iotsdk.IotDomainGroupTypeStandard),
			DisplayName:   testIotDomainGroupName,
			Description:   "initial description",
			FreeformTags:  map[string]string{"env": "test"},
			DefinedTags:   map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeTrackedIotDomainGroupResource() *iotv1beta1.IotDomainGroup {
	resource := makeIotDomainGroupResource()
	resource.Status.OsokStatus.Ocid = shared.OCID(testIotDomainGroupID)
	resource.Status.Id = testIotDomainGroupID
	resource.Status.CompartmentId = testIotDomainGroupCompartmentID
	resource.Status.Type = string(iotsdk.IotDomainGroupTypeStandard)
	resource.Status.DisplayName = testIotDomainGroupName
	resource.Status.LifecycleState = string(iotsdk.IotDomainGroupLifecycleStateActive)
	return resource
}

func makeIotDomainGroupRequest(resource *iotv1beta1.IotDomainGroup) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		},
	}
}

func makeSDKIotDomainGroup(
	id string,
	spec iotv1beta1.IotDomainGroupSpec,
	state iotsdk.IotDomainGroupLifecycleStateEnum,
) iotsdk.IotDomainGroup {
	return iotsdk.IotDomainGroup{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		Type:           iotsdk.IotDomainGroupTypeEnum(spec.Type),
		DisplayName:    common.String(spec.DisplayName),
		Description:    common.String(spec.Description),
		LifecycleState: state,
		FreeformTags:   cloneIotDomainGroupStringMap(spec.FreeformTags),
		DefinedTags:    iotDomainGroupDefinedTags(spec.DefinedTags),
	}
}

func makeSDKIotDomainGroupSummary(
	id string,
	spec iotv1beta1.IotDomainGroupSpec,
	state iotsdk.IotDomainGroupLifecycleStateEnum,
) iotsdk.IotDomainGroupSummary {
	return iotsdk.IotDomainGroupSummary{
		Id:             common.String(id),
		CompartmentId:  common.String(spec.CompartmentId),
		Type:           iotsdk.IotDomainGroupSummaryTypeEnum(spec.Type),
		DisplayName:    common.String(spec.DisplayName),
		Description:    common.String(spec.Description),
		LifecycleState: state,
		FreeformTags:   cloneIotDomainGroupStringMap(spec.FreeformTags),
		DefinedTags:    iotDomainGroupDefinedTags(spec.DefinedTags),
	}
}

func TestIotDomainGroupCreateOrUpdateBindsExistingGroupByPagedList(t *testing.T) {
	t.Parallel()

	resource := makeIotDomainGroupResource()
	probe := &pagedBindIotDomainGroupProbe{}
	client := newPagedBindIotDomainGroupClient(t, resource, probe)

	response, err := client.CreateOrUpdate(context.Background(), resource, makeIotDomainGroupRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	requirePagedBindIotDomainGroupCalls(t, probe)
	if got := string(resource.Status.OsokStatus.Ocid); got != testIotDomainGroupID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testIotDomainGroupID)
	}
	requireLastCondition(t, resource, shared.Active)
}

type pagedBindIotDomainGroupProbe struct {
	listCalls    int
	getCalls     int
	createCalled bool
	updateCalled bool
}

func newPagedBindIotDomainGroupClient(
	t *testing.T,
	resource *iotv1beta1.IotDomainGroup,
	probe *pagedBindIotDomainGroupProbe,
) IotDomainGroupServiceClient {
	t.Helper()

	return newTestIotDomainGroupClient(&fakeIotDomainGroupOCIClient{
		listFn:   pagedBindIotDomainGroupListFn(t, resource, probe),
		getFn:    pagedBindIotDomainGroupGetFn(t, resource, probe),
		createFn: pagedBindIotDomainGroupCreateFn(probe),
		updateFn: pagedBindIotDomainGroupUpdateFn(probe),
	})
}

func pagedBindIotDomainGroupListFn(
	t *testing.T,
	resource *iotv1beta1.IotDomainGroup,
	probe *pagedBindIotDomainGroupProbe,
) func(context.Context, iotsdk.ListIotDomainGroupsRequest) (iotsdk.ListIotDomainGroupsResponse, error) {
	t.Helper()

	return func(_ context.Context, req iotsdk.ListIotDomainGroupsRequest) (iotsdk.ListIotDomainGroupsResponse, error) {
		probe.listCalls++
		requirePagedBindIotDomainGroupListRequest(t, resource, req)
		if probe.listCalls == 1 {
			return firstPagedBindIotDomainGroupListResponse(t, resource, req), nil
		}
		requireStringPtr(t, "second ListIotDomainGroupsRequest.Page", req.Page, "page-2")
		return matchingPagedBindIotDomainGroupListResponse(resource), nil
	}
}

func requirePagedBindIotDomainGroupListRequest(
	t *testing.T,
	resource *iotv1beta1.IotDomainGroup,
	req iotsdk.ListIotDomainGroupsRequest,
) {
	t.Helper()

	requireStringPtr(t, "ListIotDomainGroupsRequest.CompartmentId", req.CompartmentId, resource.Spec.CompartmentId)
	requireStringPtr(t, "ListIotDomainGroupsRequest.DisplayName", req.DisplayName, resource.Spec.DisplayName)
	if got := req.Type; got != iotsdk.IotDomainGroupTypeStandard {
		t.Fatalf("ListIotDomainGroupsRequest.Type = %q, want %q", got, iotsdk.IotDomainGroupTypeStandard)
	}
}

func firstPagedBindIotDomainGroupListResponse(
	t *testing.T,
	resource *iotv1beta1.IotDomainGroup,
	req iotsdk.ListIotDomainGroupsRequest,
) iotsdk.ListIotDomainGroupsResponse {
	t.Helper()

	if req.Page != nil {
		t.Fatalf("first ListIotDomainGroupsRequest.Page = %q, want nil", *req.Page)
	}
	otherSpec := resource.Spec
	otherSpec.DisplayName = "other-group"
	return iotsdk.ListIotDomainGroupsResponse{
		IotDomainGroupCollection: iotsdk.IotDomainGroupCollection{
			Items: []iotsdk.IotDomainGroupSummary{
				makeSDKIotDomainGroupSummary("ocid1.iotdomaingroup.oc1..other", otherSpec, iotsdk.IotDomainGroupLifecycleStateActive),
			},
		},
		OpcNextPage: common.String("page-2"),
	}
}

func matchingPagedBindIotDomainGroupListResponse(
	resource *iotv1beta1.IotDomainGroup,
) iotsdk.ListIotDomainGroupsResponse {
	return iotsdk.ListIotDomainGroupsResponse{
		IotDomainGroupCollection: iotsdk.IotDomainGroupCollection{
			Items: []iotsdk.IotDomainGroupSummary{
				makeSDKIotDomainGroupSummary(testIotDomainGroupID, resource.Spec, iotsdk.IotDomainGroupLifecycleStateActive),
			},
		},
	}
}

func pagedBindIotDomainGroupGetFn(
	t *testing.T,
	resource *iotv1beta1.IotDomainGroup,
	probe *pagedBindIotDomainGroupProbe,
) func(context.Context, iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error) {
	t.Helper()

	return func(_ context.Context, req iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error) {
		probe.getCalls++
		requireStringPtr(t, "GetIotDomainGroupRequest.IotDomainGroupId", req.IotDomainGroupId, testIotDomainGroupID)
		return iotsdk.GetIotDomainGroupResponse{
			IotDomainGroup: makeSDKIotDomainGroup(testIotDomainGroupID, resource.Spec, iotsdk.IotDomainGroupLifecycleStateActive),
		}, nil
	}
}

func pagedBindIotDomainGroupCreateFn(
	probe *pagedBindIotDomainGroupProbe,
) func(context.Context, iotsdk.CreateIotDomainGroupRequest) (iotsdk.CreateIotDomainGroupResponse, error) {
	return func(context.Context, iotsdk.CreateIotDomainGroupRequest) (iotsdk.CreateIotDomainGroupResponse, error) {
		probe.createCalled = true
		return iotsdk.CreateIotDomainGroupResponse{}, nil
	}
}

func pagedBindIotDomainGroupUpdateFn(
	probe *pagedBindIotDomainGroupProbe,
) func(context.Context, iotsdk.UpdateIotDomainGroupRequest) (iotsdk.UpdateIotDomainGroupResponse, error) {
	return func(context.Context, iotsdk.UpdateIotDomainGroupRequest) (iotsdk.UpdateIotDomainGroupResponse, error) {
		probe.updateCalled = true
		return iotsdk.UpdateIotDomainGroupResponse{}, nil
	}
}

func requirePagedBindIotDomainGroupCalls(t *testing.T, probe *pagedBindIotDomainGroupProbe) {
	t.Helper()

	if probe.createCalled {
		t.Fatal("CreateIotDomainGroup() called for existing group")
	}
	if probe.updateCalled {
		t.Fatal("UpdateIotDomainGroup() called for matching group")
	}
	if probe.listCalls != 2 {
		t.Fatalf("ListIotDomainGroups() calls = %d, want 2 paginated calls", probe.listCalls)
	}
	if probe.getCalls != 1 {
		t.Fatalf("GetIotDomainGroup() calls = %d, want 1", probe.getCalls)
	}
}

func TestIotDomainGroupCreateRecordsRetryTokenRequestIDAndProvisioningStatus(t *testing.T) {
	t.Parallel()

	resource := makeIotDomainGroupResource()
	listCalls := 0
	createCalls := 0

	client := newTestIotDomainGroupClient(&fakeIotDomainGroupOCIClient{
		listFn: func(context.Context, iotsdk.ListIotDomainGroupsRequest) (iotsdk.ListIotDomainGroupsResponse, error) {
			listCalls++
			return iotsdk.ListIotDomainGroupsResponse{}, nil
		},
		createFn: func(_ context.Context, req iotsdk.CreateIotDomainGroupRequest) (iotsdk.CreateIotDomainGroupResponse, error) {
			createCalls++
			requireIotDomainGroupCreateRequest(t, req, resource)
			return iotsdk.CreateIotDomainGroupResponse{
				IotDomainGroup:   makeSDKIotDomainGroup(testIotDomainGroupID, resource.Spec, iotsdk.IotDomainGroupLifecycleStateCreating),
				OpcRequestId:     common.String("opc-create"),
				OpcWorkRequestId: common.String("wr-create"),
			}, nil
		},
		getFn: func(_ context.Context, req iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error) {
			requireStringPtr(t, "GetIotDomainGroupRequest.IotDomainGroupId", req.IotDomainGroupId, testIotDomainGroupID)
			return iotsdk.GetIotDomainGroupResponse{
				IotDomainGroup: makeSDKIotDomainGroup(testIotDomainGroupID, resource.Spec, iotsdk.IotDomainGroupLifecycleStateCreating),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeIotDomainGroupRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() shouldRequeue = false, want true for CREATING lifecycle")
	}
	if listCalls != 1 {
		t.Fatalf("ListIotDomainGroups() calls = %d, want 1 pre-create lookup", listCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreateIotDomainGroup() calls = %d, want 1", createCalls)
	}
	if got := string(resource.Status.OsokStatus.Ocid); got != testIotDomainGroupID {
		t.Fatalf("status.status.ocid = %q, want %q", got, testIotDomainGroupID)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-create" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-create", got)
	}
	requireCurrentAsync(t, resource, shared.OSOKAsyncPhaseCreate, iotsdk.IotDomainGroupLifecycleStateCreating)
	requireLastCondition(t, resource, shared.Provisioning)
}

func TestIotDomainGroupCreateOrUpdateNoopsWhenReadbackMatches(t *testing.T) {
	t.Parallel()

	resource := makeTrackedIotDomainGroupResource()
	updateCalled := false

	client := newTestIotDomainGroupClient(&fakeIotDomainGroupOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error) {
			requireStringPtr(t, "GetIotDomainGroupRequest.IotDomainGroupId", req.IotDomainGroupId, testIotDomainGroupID)
			return iotsdk.GetIotDomainGroupResponse{
				IotDomainGroup: makeSDKIotDomainGroup(testIotDomainGroupID, resource.Spec, iotsdk.IotDomainGroupLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateIotDomainGroupRequest) (iotsdk.UpdateIotDomainGroupResponse, error) {
			updateCalled = true
			return iotsdk.UpdateIotDomainGroupResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeIotDomainGroupRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalled {
		t.Fatal("UpdateIotDomainGroup() called for matching readback")
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestIotDomainGroupCreateOrUpdateUpdatesMutableFields(t *testing.T) {
	t.Parallel()

	resource := makeTrackedIotDomainGroupResource()
	resource.Spec.DisplayName = "updated-group"
	resource.Spec.Description = "updated description"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "99"}}
	currentSpec := resource.Spec
	currentSpec.DisplayName = testIotDomainGroupName
	currentSpec.Description = "initial description"
	currentSpec.FreeformTags = map[string]string{"env": "test"}
	currentSpec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "42"}}
	getCalls := 0
	updateCalls := 0

	client := newTestIotDomainGroupClient(&fakeIotDomainGroupOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error) {
			getCalls++
			requireStringPtr(t, "GetIotDomainGroupRequest.IotDomainGroupId", req.IotDomainGroupId, testIotDomainGroupID)
			if getCalls == 1 {
				return iotsdk.GetIotDomainGroupResponse{
					IotDomainGroup: makeSDKIotDomainGroup(testIotDomainGroupID, currentSpec, iotsdk.IotDomainGroupLifecycleStateActive),
				}, nil
			}
			return iotsdk.GetIotDomainGroupResponse{
				IotDomainGroup: makeSDKIotDomainGroup(testIotDomainGroupID, resource.Spec, iotsdk.IotDomainGroupLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req iotsdk.UpdateIotDomainGroupRequest) (iotsdk.UpdateIotDomainGroupResponse, error) {
			updateCalls++
			requireStringPtr(t, "UpdateIotDomainGroupRequest.IotDomainGroupId", req.IotDomainGroupId, testIotDomainGroupID)
			requireStringPtr(t, "UpdateIotDomainGroupDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
			requireStringPtr(t, "UpdateIotDomainGroupDetails.Description", req.Description, resource.Spec.Description)
			if got := req.FreeformTags["env"]; got != "prod" {
				t.Fatalf("UpdateIotDomainGroupDetails.FreeformTags[env] = %q, want prod", got)
			}
			if got := req.DefinedTags["Operations"]["CostCenter"]; got != "99" {
				t.Fatalf("UpdateIotDomainGroupDetails.DefinedTags[Operations][CostCenter] = %v, want 99", got)
			}
			return iotsdk.UpdateIotDomainGroupResponse{
				OpcRequestId:     common.String("opc-update"),
				OpcWorkRequestId: common.String("wr-update"),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeIotDomainGroupRequest(resource))
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = false, want true")
	}
	if updateCalls != 1 {
		t.Fatalf("UpdateIotDomainGroup() calls = %d, want 1", updateCalls)
	}
	if getCalls != 2 {
		t.Fatalf("GetIotDomainGroup() calls = %d, want current read and update follow-up", getCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-update" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-update", got)
	}
	requireLastCondition(t, resource, shared.Active)
}

func TestIotDomainGroupCreateOrUpdateRejectsCreateOnlyTypeDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	resource := makeTrackedIotDomainGroupResource()
	resource.Spec.Type = string(iotsdk.IotDomainGroupTypeLightweight)
	currentSpec := resource.Spec
	currentSpec.Type = string(iotsdk.IotDomainGroupTypeStandard)
	updateCalled := false

	client := newTestIotDomainGroupClient(&fakeIotDomainGroupOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error) {
			requireStringPtr(t, "GetIotDomainGroupRequest.IotDomainGroupId", req.IotDomainGroupId, testIotDomainGroupID)
			return iotsdk.GetIotDomainGroupResponse{
				IotDomainGroup: makeSDKIotDomainGroup(testIotDomainGroupID, currentSpec, iotsdk.IotDomainGroupLifecycleStateActive),
			}, nil
		},
		updateFn: func(context.Context, iotsdk.UpdateIotDomainGroupRequest) (iotsdk.UpdateIotDomainGroupResponse, error) {
			updateCalled = true
			return iotsdk.UpdateIotDomainGroupResponse{}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeIotDomainGroupRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want create-only type drift rejection")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if updateCalled {
		t.Fatal("UpdateIotDomainGroup() called after create-only type drift")
	}
	if !strings.Contains(err.Error(), "type changes") {
		t.Fatalf("CreateOrUpdate() error = %v, want type force-new rejection", err)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func TestIotDomainGroupDeleteKeepsFinalizerUntilReadConfirmsNotFound(t *testing.T) {
	t.Parallel()

	resource := makeTrackedIotDomainGroupResource()
	getCalls := 0
	deleteCalls := 0

	client := newTestIotDomainGroupClient(&fakeIotDomainGroupOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error) {
			getCalls++
			requireStringPtr(t, "GetIotDomainGroupRequest.IotDomainGroupId", req.IotDomainGroupId, testIotDomainGroupID)
			switch getCalls {
			case 1, 2, 3:
				return iotsdk.GetIotDomainGroupResponse{
					IotDomainGroup: makeSDKIotDomainGroup(testIotDomainGroupID, resource.Spec, iotsdk.IotDomainGroupLifecycleStateActive),
				}, nil
			default:
				return iotsdk.GetIotDomainGroupResponse{}, errortest.NewServiceError(404, errorutil.NotFound, "iot domain group is gone")
			}
		},
		deleteFn: func(_ context.Context, req iotsdk.DeleteIotDomainGroupRequest) (iotsdk.DeleteIotDomainGroupResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteIotDomainGroupRequest.IotDomainGroupId", req.IotDomainGroupId, testIotDomainGroupID)
			return iotsdk.DeleteIotDomainGroupResponse{
				OpcRequestId:     common.String("opc-delete"),
				OpcWorkRequestId: common.String("wr-delete"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() first error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() first deleted = true, want false while readback remains ACTIVE")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteIotDomainGroup() calls after first delete = %d, want 1", deleteCalls)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-delete" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-delete", got)
	}
	requireDeletePendingStatus(t, resource)
	requireLastCondition(t, resource, shared.Terminating)

	deleted, err = client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() second error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() second deleted = false, want true after unambiguous NotFound")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteIotDomainGroup() calls after confirmed delete = %d, want still 1", deleteCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.status.deletedAt = nil, want deletion timestamp")
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func TestIotDomainGroupDeleteRetainsFinalizerForPendingWriteLifecycle(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name      string
		state     iotsdk.IotDomainGroupLifecycleStateEnum
		wantPhase shared.OSOKAsyncPhase
	}{
		{name: "creating", state: iotsdk.IotDomainGroupLifecycleStateCreating, wantPhase: shared.OSOKAsyncPhaseCreate},
		{name: "updating", state: iotsdk.IotDomainGroupLifecycleStateUpdating, wantPhase: shared.OSOKAsyncPhaseUpdate},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runIotDomainGroupPendingWriteDeleteCase(t, tt.state, tt.wantPhase)
		})
	}
}

func runIotDomainGroupPendingWriteDeleteCase(
	t *testing.T,
	state iotsdk.IotDomainGroupLifecycleStateEnum,
	wantPhase shared.OSOKAsyncPhase,
) {
	t.Helper()

	resource := makeTrackedIotDomainGroupResource()
	getCalls := 0
	deleteCalled := false
	client := newPendingWriteDeleteIotDomainGroupClient(t, resource, state, &getCalls, &deleteCalled)

	deleted, err := client.Delete(context.Background(), resource)
	requireIotDomainGroupPendingWriteDeleteResult(
		t,
		resource,
		deleted,
		err,
		getCalls,
		deleteCalled,
		state,
		wantPhase,
	)
}

func newPendingWriteDeleteIotDomainGroupClient(
	t *testing.T,
	resource *iotv1beta1.IotDomainGroup,
	state iotsdk.IotDomainGroupLifecycleStateEnum,
	getCalls *int,
	deleteCalled *bool,
) IotDomainGroupServiceClient {
	t.Helper()

	return newTestIotDomainGroupClient(&fakeIotDomainGroupOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error) {
			(*getCalls)++
			requireStringPtr(t, "GetIotDomainGroupRequest.IotDomainGroupId", req.IotDomainGroupId, testIotDomainGroupID)
			return iotsdk.GetIotDomainGroupResponse{
				IotDomainGroup: makeSDKIotDomainGroup(testIotDomainGroupID, resource.Spec, state),
			}, nil
		},
		deleteFn: func(context.Context, iotsdk.DeleteIotDomainGroupRequest) (iotsdk.DeleteIotDomainGroupResponse, error) {
			*deleteCalled = true
			return iotsdk.DeleteIotDomainGroupResponse{}, nil
		},
	})
}

func requireIotDomainGroupPendingWriteDeleteResult(
	t *testing.T,
	resource *iotv1beta1.IotDomainGroup,
	deleted bool,
	err error,
	getCalls int,
	deleteCalled bool,
	state iotsdk.IotDomainGroupLifecycleStateEnum,
	wantPhase shared.OSOKAsyncPhase,
) {
	t.Helper()

	requireIotDomainGroupPendingWriteDeleteCallResult(t, deleted, err, getCalls, deleteCalled)
	requireIotDomainGroupPendingWriteStatus(t, resource, state, wantPhase)
}

func requireIotDomainGroupPendingWriteDeleteCallResult(
	t *testing.T,
	deleted bool,
	err error,
	getCalls int,
	deleteCalled bool,
) {
	t.Helper()

	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while write is pending")
	}
	if deleteCalled {
		t.Fatal("DeleteIotDomainGroup() called while write lifecycle state is still pending")
	}
	if getCalls != 2 {
		t.Fatalf("GetIotDomainGroup() calls = %d, want pre-delete and generated confirmation reads", getCalls)
	}
}

func requireIotDomainGroupPendingWriteStatus(
	t *testing.T,
	resource *iotv1beta1.IotDomainGroup,
	state iotsdk.IotDomainGroupLifecycleStateEnum,
	wantPhase shared.OSOKAsyncPhase,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want pending write tracker")
	}
	if !iotDomainGroupPendingWriteStatusMatches(current, state, wantPhase) {
		t.Fatalf("status.status.async.current = %#v, want lifecycle %s %s", current, wantPhase, state)
	}
	if got := resource.Status.OsokStatus.Message; got != iotDomainGroupPendingWriteDeleteMessage {
		t.Fatalf("status.status.message = %q, want %q", got, iotDomainGroupPendingWriteDeleteMessage)
	}
	requireLastCondition(t, resource, shared.Terminating)
}

func iotDomainGroupPendingWriteStatusMatches(
	current *shared.OSOKAsyncOperation,
	state iotsdk.IotDomainGroupLifecycleStateEnum,
	wantPhase shared.OSOKAsyncPhase,
) bool {
	return current.Phase == wantPhase &&
		current.Source == shared.OSOKAsyncSourceLifecycle &&
		current.NormalizedClass == shared.OSOKAsyncClassPending &&
		current.RawStatus == string(state)
}

func TestIotDomainGroupDeleteTreatsAuthShapedNotFoundConservatively(t *testing.T) {
	t.Parallel()

	resource := makeTrackedIotDomainGroupResource()

	client := newTestIotDomainGroupClient(&fakeIotDomainGroupOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error) {
			requireStringPtr(t, "GetIotDomainGroupRequest.IotDomainGroupId", req.IotDomainGroupId, testIotDomainGroupID)
			return iotsdk.GetIotDomainGroupResponse{
				IotDomainGroup: makeSDKIotDomainGroup(testIotDomainGroupID, resource.Spec, iotsdk.IotDomainGroupLifecycleStateActive),
			}, nil
		},
		deleteFn: func(_ context.Context, req iotsdk.DeleteIotDomainGroupRequest) (iotsdk.DeleteIotDomainGroupResponse, error) {
			requireStringPtr(t, "DeleteIotDomainGroupRequest.IotDomainGroupId", req.IotDomainGroupId, testIotDomainGroupID)
			return iotsdk.DeleteIotDomainGroupResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped 404")
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestIotDomainGroupDeleteRejectsAuthShapedPreDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeTrackedIotDomainGroupResource()
	deleteCalled := false

	client := newTestIotDomainGroupClient(&fakeIotDomainGroupOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error) {
			requireStringPtr(t, "GetIotDomainGroupRequest.IotDomainGroupId", req.IotDomainGroupId, testIotDomainGroupID)
			return iotsdk.GetIotDomainGroupResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
		deleteFn: func(context.Context, iotsdk.DeleteIotDomainGroupRequest) (iotsdk.DeleteIotDomainGroupResponse, error) {
			deleteCalled = true
			return iotsdk.DeleteIotDomainGroupResponse{}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous pre-delete confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped pre-delete confirm read")
	}
	if deleteCalled {
		t.Fatal("DeleteIotDomainGroup() called after auth-shaped pre-delete confirm read")
	}
	if !strings.Contains(err.Error(), "delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestIotDomainGroupDeleteRejectsAuthShapedPostDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	resource := makeTrackedIotDomainGroupResource()
	getCalls := 0
	deleteCalls := 0

	client := newTestIotDomainGroupClient(&fakeIotDomainGroupOCIClient{
		getFn: func(_ context.Context, req iotsdk.GetIotDomainGroupRequest) (iotsdk.GetIotDomainGroupResponse, error) {
			getCalls++
			requireStringPtr(t, "GetIotDomainGroupRequest.IotDomainGroupId", req.IotDomainGroupId, testIotDomainGroupID)
			if getCalls < 3 {
				return iotsdk.GetIotDomainGroupResponse{
					IotDomainGroup: makeSDKIotDomainGroup(testIotDomainGroupID, resource.Spec, iotsdk.IotDomainGroupLifecycleStateActive),
				}, nil
			}
			return iotsdk.GetIotDomainGroupResponse{}, errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "authorization or existence is ambiguous")
		},
		deleteFn: func(_ context.Context, req iotsdk.DeleteIotDomainGroupRequest) (iotsdk.DeleteIotDomainGroupResponse, error) {
			deleteCalls++
			requireStringPtr(t, "DeleteIotDomainGroupRequest.IotDomainGroupId", req.IotDomainGroupId, testIotDomainGroupID)
			return iotsdk.DeleteIotDomainGroupResponse{
				OpcRequestId: common.String("opc-delete"),
			}, nil
		},
	})

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous post-delete confirm-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want false for auth-shaped post-delete confirm read")
	}
	if deleteCalls != 1 {
		t.Fatalf("DeleteIotDomainGroup() calls = %d, want 1", deleteCalls)
	}
	if !strings.Contains(err.Error(), "ambiguous 404 NotAuthorizedOrNotFound") {
		t.Fatalf("Delete() error = %v, want ambiguous confirm-read classification", err)
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
}

func TestIotDomainGroupCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	resource := makeIotDomainGroupResource()

	client := newTestIotDomainGroupClient(&fakeIotDomainGroupOCIClient{
		listFn: func(context.Context, iotsdk.ListIotDomainGroupsRequest) (iotsdk.ListIotDomainGroupsResponse, error) {
			return iotsdk.ListIotDomainGroupsResponse{}, nil
		},
		createFn: func(context.Context, iotsdk.CreateIotDomainGroupRequest) (iotsdk.CreateIotDomainGroupResponse, error) {
			return iotsdk.CreateIotDomainGroupResponse{}, errortest.NewServiceError(500, errorutil.InternalServerError, "create failed")
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, makeIotDomainGroupRequest(resource))
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI service error")
	}
	if response.IsSuccessful {
		t.Fatal("CreateOrUpdate() successful = true, want false")
	}
	if got := resource.Status.OsokStatus.OpcRequestID; got != "opc-request-id" {
		t.Fatalf("status.status.opcRequestId = %q, want opc-request-id", got)
	}
	requireLastCondition(t, resource, shared.Failed)
}

func requireIotDomainGroupCreateRequest(
	t *testing.T,
	req iotsdk.CreateIotDomainGroupRequest,
	resource *iotv1beta1.IotDomainGroup,
) {
	t.Helper()
	requireStringPtr(t, "CreateIotDomainGroupDetails.CompartmentId", req.CompartmentId, resource.Spec.CompartmentId)
	if got := req.Type; got != iotsdk.CreateIotDomainGroupDetailsTypeStandard {
		t.Fatalf("CreateIotDomainGroupDetails.Type = %q, want %q", got, iotsdk.CreateIotDomainGroupDetailsTypeStandard)
	}
	requireStringPtr(t, "CreateIotDomainGroupDetails.DisplayName", req.DisplayName, resource.Spec.DisplayName)
	requireStringPtr(t, "CreateIotDomainGroupDetails.Description", req.Description, resource.Spec.Description)
	if got := req.FreeformTags["env"]; got != "test" {
		t.Fatalf("CreateIotDomainGroupDetails.FreeformTags[env] = %q, want test", got)
	}
	if got := req.DefinedTags["Operations"]["CostCenter"]; got != "42" {
		t.Fatalf("CreateIotDomainGroupDetails.DefinedTags[Operations][CostCenter] = %v, want 42", got)
	}
	if req.OpcRetryToken == nil || strings.TrimSpace(*req.OpcRetryToken) == "" {
		t.Fatal("CreateIotDomainGroupRequest.OpcRetryToken is empty, want deterministic retry token")
	}
}

func requireDeletePendingStatus(t *testing.T, resource *iotv1beta1.IotDomainGroup) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want delete pending tracker")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle ||
		current.Phase != shared.OSOKAsyncPhaseDelete ||
		current.NormalizedClass != shared.OSOKAsyncClassPending ||
		current.RawStatus != string(iotsdk.IotDomainGroupLifecycleStateActive) {
		t.Fatalf("status.status.async.current = %#v, want lifecycle delete pending ACTIVE", current)
	}
}

func requireCurrentAsync(
	t *testing.T,
	resource *iotv1beta1.IotDomainGroup,
	wantPhase shared.OSOKAsyncPhase,
	wantRawStatus iotsdk.IotDomainGroupLifecycleStateEnum,
) {
	t.Helper()
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.status.async.current = nil, want current async operation")
	}
	if current.Source != shared.OSOKAsyncSourceLifecycle ||
		current.Phase != wantPhase ||
		current.NormalizedClass != shared.OSOKAsyncClassPending ||
		current.RawStatus != string(wantRawStatus) {
		t.Fatalf("status.status.async.current = %#v, want lifecycle %s pending %s", current, wantPhase, wantRawStatus)
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

func requireLastCondition(t *testing.T, resource *iotv1beta1.IotDomainGroup, want shared.OSOKConditionType) {
	t.Helper()
	conditions := resource.Status.OsokStatus.Conditions
	if len(conditions) == 0 {
		t.Fatalf("status.status.conditions is empty, want trailing %s", want)
	}
	if got := conditions[len(conditions)-1].Type; got != want {
		t.Fatalf("last condition = %s, want %s", got, want)
	}
}
