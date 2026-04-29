/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package clusterplacementgroup

import (
	"context"
	"reflect"
	"strings"
	"testing"

	clusterplacementgroupssdk "github.com/oracle/oci-go-sdk/v65/clusterplacementgroups"
	"github.com/oracle/oci-go-sdk/v65/common"
	clusterplacementgroupsv1beta1 "github.com/oracle/oci-service-operator/api/clusterplacementgroups/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeClusterPlacementGroupOCIClient struct {
	createFn      func(context.Context, clusterplacementgroupssdk.CreateClusterPlacementGroupRequest) (clusterplacementgroupssdk.CreateClusterPlacementGroupResponse, error)
	getFn         func(context.Context, clusterplacementgroupssdk.GetClusterPlacementGroupRequest) (clusterplacementgroupssdk.GetClusterPlacementGroupResponse, error)
	listFn        func(context.Context, clusterplacementgroupssdk.ListClusterPlacementGroupsRequest) (clusterplacementgroupssdk.ListClusterPlacementGroupsResponse, error)
	updateFn      func(context.Context, clusterplacementgroupssdk.UpdateClusterPlacementGroupRequest) (clusterplacementgroupssdk.UpdateClusterPlacementGroupResponse, error)
	deleteFn      func(context.Context, clusterplacementgroupssdk.DeleteClusterPlacementGroupRequest) (clusterplacementgroupssdk.DeleteClusterPlacementGroupResponse, error)
	workRequestFn func(context.Context, clusterplacementgroupssdk.GetWorkRequestRequest) (clusterplacementgroupssdk.GetWorkRequestResponse, error)
}

func (f *fakeClusterPlacementGroupOCIClient) CreateClusterPlacementGroup(
	ctx context.Context,
	req clusterplacementgroupssdk.CreateClusterPlacementGroupRequest,
) (clusterplacementgroupssdk.CreateClusterPlacementGroupResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return clusterplacementgroupssdk.CreateClusterPlacementGroupResponse{}, nil
}

func (f *fakeClusterPlacementGroupOCIClient) GetClusterPlacementGroup(
	ctx context.Context,
	req clusterplacementgroupssdk.GetClusterPlacementGroupRequest,
) (clusterplacementgroupssdk.GetClusterPlacementGroupResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return clusterplacementgroupssdk.GetClusterPlacementGroupResponse{}, errortest.NewServiceError(404, "NotFound", "missing")
}

func (f *fakeClusterPlacementGroupOCIClient) ListClusterPlacementGroups(
	ctx context.Context,
	req clusterplacementgroupssdk.ListClusterPlacementGroupsRequest,
) (clusterplacementgroupssdk.ListClusterPlacementGroupsResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return clusterplacementgroupssdk.ListClusterPlacementGroupsResponse{}, nil
}

func (f *fakeClusterPlacementGroupOCIClient) UpdateClusterPlacementGroup(
	ctx context.Context,
	req clusterplacementgroupssdk.UpdateClusterPlacementGroupRequest,
) (clusterplacementgroupssdk.UpdateClusterPlacementGroupResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return clusterplacementgroupssdk.UpdateClusterPlacementGroupResponse{}, nil
}

func (f *fakeClusterPlacementGroupOCIClient) DeleteClusterPlacementGroup(
	ctx context.Context,
	req clusterplacementgroupssdk.DeleteClusterPlacementGroupRequest,
) (clusterplacementgroupssdk.DeleteClusterPlacementGroupResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return clusterplacementgroupssdk.DeleteClusterPlacementGroupResponse{}, nil
}

func (f *fakeClusterPlacementGroupOCIClient) GetWorkRequest(
	ctx context.Context,
	req clusterplacementgroupssdk.GetWorkRequestRequest,
) (clusterplacementgroupssdk.GetWorkRequestResponse, error) {
	if f.workRequestFn != nil {
		return f.workRequestFn(ctx, req)
	}
	return clusterplacementgroupssdk.GetWorkRequestResponse{}, nil
}

func TestReviewedClusterPlacementGroupRuntimeSemanticsEncodesWorkRequestContract(t *testing.T) {
	t.Parallel()

	got := reviewedClusterPlacementGroupRuntimeSemantics()
	if got == nil {
		t.Fatal("reviewedClusterPlacementGroupRuntimeSemantics() = nil")
	}

	if got.FormalService != "clusterplacementgroups" {
		t.Fatalf("FormalService = %q, want clusterplacementgroups", got.FormalService)
	}
	if got.FormalSlug != "clusterplacementgroup" {
		t.Fatalf("FormalSlug = %q, want clusterplacementgroup", got.FormalSlug)
	}
	if got.Async == nil {
		t.Fatal("Async = nil, want workrequest semantics")
	}
	if got.Async.Strategy != "workrequest" {
		t.Fatalf("Async.Strategy = %q, want workrequest", got.Async.Strategy)
	}
	if got.Async.Runtime != "generatedruntime" {
		t.Fatalf("Async.Runtime = %q, want generatedruntime", got.Async.Runtime)
	}
	if got.Async.WorkRequest == nil {
		t.Fatal("Async.WorkRequest = nil")
	}
	assertClusterPlacementGroupStringSliceEqual(t, "Async.WorkRequest.Phases", got.Async.WorkRequest.Phases, []string{"create", "update", "delete"})
	assertClusterPlacementGroupStringSliceEqual(t, "Lifecycle.ProvisioningStates", got.Lifecycle.ProvisioningStates, []string{"CREATING"})
	assertClusterPlacementGroupStringSliceEqual(t, "Lifecycle.UpdatingStates", got.Lifecycle.UpdatingStates, []string{"UPDATING"})
	assertClusterPlacementGroupStringSliceEqual(t, "Lifecycle.ActiveStates", got.Lifecycle.ActiveStates, []string{"ACTIVE", "INACTIVE"})
	assertClusterPlacementGroupStringSliceEqual(t, "Delete.PendingStates", got.Delete.PendingStates, []string{"DELETING"})
	assertClusterPlacementGroupStringSliceEqual(t, "Delete.TerminalStates", got.Delete.TerminalStates, []string{"DELETED"})
	assertClusterPlacementGroupStringSliceEqual(t, "List.MatchFields", got.List.MatchFields, []string{"availabilityDomain", "clusterPlacementGroupType", "compartmentId", "name"})
	assertClusterPlacementGroupStringSliceEqual(t, "Mutation.Mutable", got.Mutation.Mutable, []string{"definedTags", "description", "freeformTags"})
	assertClusterPlacementGroupStringSliceEqual(t, "Mutation.ForceNew", got.Mutation.ForceNew, []string{"availabilityDomain", "capabilities", "clusterPlacementGroupType", "compartmentId", "name", "placementInstruction"})
	if got.FinalizerPolicy != "retain-until-confirmed-delete" {
		t.Fatalf("FinalizerPolicy = %q, want retain-until-confirmed-delete", got.FinalizerPolicy)
	}
	if got.Delete.Policy != "required" {
		t.Fatalf("Delete.Policy = %q, want required", got.Delete.Policy)
	}
	if got.CreateFollowUp.Strategy != "GetWorkRequest -> GetClusterPlacementGroup" {
		t.Fatalf("CreateFollowUp.Strategy = %q, want GetWorkRequest -> GetClusterPlacementGroup", got.CreateFollowUp.Strategy)
	}
	if got.UpdateFollowUp.Strategy != "GetWorkRequest -> GetClusterPlacementGroup" {
		t.Fatalf("UpdateFollowUp.Strategy = %q, want GetWorkRequest -> GetClusterPlacementGroup", got.UpdateFollowUp.Strategy)
	}
	if got.DeleteFollowUp.Strategy != "GetWorkRequest -> GetClusterPlacementGroup confirm-delete" {
		t.Fatalf("DeleteFollowUp.Strategy = %q, want GetWorkRequest -> GetClusterPlacementGroup confirm-delete", got.DeleteFollowUp.Strategy)
	}
	if len(got.AuxiliaryOperations) != 0 {
		t.Fatalf("AuxiliaryOperations = %#v, want none for published runtime", got.AuxiliaryOperations)
	}
}

func TestGuardClusterPlacementGroupExistingBeforeCreate(t *testing.T) {
	t.Parallel()

	resource := makeClusterPlacementGroupResource()
	resource.Spec.AvailabilityDomain = ""

	decision, err := guardClusterPlacementGroupExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardClusterPlacementGroupExistingBeforeCreate(empty availabilityDomain) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionSkip {
		t.Fatalf("guardClusterPlacementGroupExistingBeforeCreate(empty availabilityDomain) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionSkip)
	}

	resource.Spec.AvailabilityDomain = "AD-1"
	decision, err = guardClusterPlacementGroupExistingBeforeCreate(context.Background(), resource)
	if err != nil {
		t.Fatalf("guardClusterPlacementGroupExistingBeforeCreate(complete criteria) error = %v", err)
	}
	if decision != generatedruntime.ExistingBeforeCreateDecisionAllow {
		t.Fatalf("guardClusterPlacementGroupExistingBeforeCreate(complete criteria) = %q, want %q", decision, generatedruntime.ExistingBeforeCreateDecisionAllow)
	}
}

func TestBuildClusterPlacementGroupUpdateBodyPreservesClears(t *testing.T) {
	t.Parallel()

	currentResource := makeClusterPlacementGroupResource()
	desired := makeClusterPlacementGroupResource()
	desired.Spec.Description = ""
	desired.Spec.FreeformTags = map[string]string{}
	desired.Spec.DefinedTags = map[string]shared.MapValue{}

	body, updateNeeded, err := buildClusterPlacementGroupUpdateBody(
		desired,
		clusterplacementgroupssdk.GetClusterPlacementGroupResponse{
			ClusterPlacementGroup: makeSDKClusterPlacementGroup("ocid1.clusterplacementgroup.oc1..existing", currentResource, clusterplacementgroupssdk.ClusterPlacementGroupLifecycleStateActive),
		},
	)
	if err != nil {
		t.Fatalf("buildClusterPlacementGroupUpdateBody() error = %v", err)
	}
	if !updateNeeded {
		t.Fatal("buildClusterPlacementGroupUpdateBody() updateNeeded = false, want true")
	}

	requireClusterPlacementGroupStringPtr(t, "details.description", body.Description, "")
	if len(body.FreeformTags) != 0 {
		t.Fatalf("details.FreeformTags = %#v, want empty map for clear", body.FreeformTags)
	}
	if len(body.DefinedTags) != 0 {
		t.Fatalf("details.DefinedTags = %#v, want empty map for clear", body.DefinedTags)
	}
}

func TestClusterPlacementGroupCreateOrUpdateSkipsReuseWhenCriteriaMissing(t *testing.T) {
	t.Parallel()

	resource := makeClusterPlacementGroupResource()
	resource.Spec.Name = ""

	listCalls := 0
	createCalls := 0

	client := newTestClusterPlacementGroupClient(&fakeClusterPlacementGroupOCIClient{
		listFn: func(_ context.Context, _ clusterplacementgroupssdk.ListClusterPlacementGroupsRequest) (clusterplacementgroupssdk.ListClusterPlacementGroupsResponse, error) {
			listCalls++
			return clusterplacementgroupssdk.ListClusterPlacementGroupsResponse{}, nil
		},
		createFn: func(_ context.Context, req clusterplacementgroupssdk.CreateClusterPlacementGroupRequest) (clusterplacementgroupssdk.CreateClusterPlacementGroupResponse, error) {
			createCalls++
			requireClusterPlacementGroupStringPtr(t, "create compartmentId", req.CreateClusterPlacementGroupDetails.CompartmentId, resource.Spec.CompartmentId)
			requireClusterPlacementGroupStringPtr(t, "create name", req.CreateClusterPlacementGroupDetails.Name, "")
			if req.CreateClusterPlacementGroupDetails.PlacementInstruction != nil {
				t.Fatalf("create placementInstruction = %#v, want nil when spec.placementInstruction is empty", req.CreateClusterPlacementGroupDetails.PlacementInstruction)
			}
			if req.CreateClusterPlacementGroupDetails.Capabilities != nil {
				t.Fatalf("create capabilities = %#v, want nil when spec.capabilities is empty", req.CreateClusterPlacementGroupDetails.Capabilities)
			}
			return clusterplacementgroupssdk.CreateClusterPlacementGroupResponse{
				ClusterPlacementGroup: makeSDKClusterPlacementGroup("ocid1.clusterplacementgroup.oc1..created", resource, clusterplacementgroupssdk.ClusterPlacementGroupLifecycleStateCreating),
				OpcWorkRequestId:      common.String("wr-create-missing-name"),
				OpcRequestId:          common.String("opc-create-missing-name"),
			}, nil
		},
		workRequestFn: func(_ context.Context, req clusterplacementgroupssdk.GetWorkRequestRequest) (clusterplacementgroupssdk.GetWorkRequestResponse, error) {
			requireClusterPlacementGroupStringPtr(t, "workRequestId", req.WorkRequestId, "wr-create-missing-name")
			return clusterplacementgroupssdk.GetWorkRequestResponse{
				WorkRequest: makeClusterPlacementGroupWorkRequest(
					"wr-create-missing-name",
					clusterplacementgroupssdk.OperationTypeCreateClusterPlacementGroup,
					clusterplacementgroupssdk.OperationStatusInProgress,
					clusterplacementgroupssdk.ActionTypeCreated,
					"ocid1.clusterplacementgroup.oc1..created",
				),
			}, nil
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful pending create", response)
	}
	if listCalls != 0 {
		t.Fatalf("ListClusterPlacementGroups() calls = %d, want 0 when pre-create criteria are incomplete", listCalls)
	}
	if createCalls != 1 {
		t.Fatalf("CreateClusterPlacementGroup() calls = %d, want 1", createCalls)
	}
	requireClusterPlacementGroupAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-missing-name", shared.OSOKAsyncClassPending)
}

func TestClusterPlacementGroupCreateOrUpdateFailsOnDuplicateExactListMatches(t *testing.T) {
	t.Parallel()

	resource := makeClusterPlacementGroupResource()
	createCalls := 0

	client := newTestClusterPlacementGroupClient(&fakeClusterPlacementGroupOCIClient{
		listFn: func(_ context.Context, req clusterplacementgroupssdk.ListClusterPlacementGroupsRequest) (clusterplacementgroupssdk.ListClusterPlacementGroupsResponse, error) {
			requireClusterPlacementGroupStringPtr(t, "list compartmentId", req.CompartmentId, resource.Spec.CompartmentId)
			requireClusterPlacementGroupStringPtr(t, "list name", req.Name, resource.Spec.Name)
			requireClusterPlacementGroupStringPtr(t, "list ad", req.Ad, resource.Spec.AvailabilityDomain)
			return clusterplacementgroupssdk.ListClusterPlacementGroupsResponse{
				ClusterPlacementGroupCollection: clusterplacementgroupssdk.ClusterPlacementGroupCollection{
					Items: []clusterplacementgroupssdk.ClusterPlacementGroupSummary{
						makeSDKClusterPlacementGroupSummary("ocid1.clusterplacementgroup.oc1..one", resource, clusterplacementgroupssdk.ClusterPlacementGroupLifecycleStateActive),
						makeSDKClusterPlacementGroupSummary("ocid1.clusterplacementgroup.oc1..two", resource, clusterplacementgroupssdk.ClusterPlacementGroupLifecycleStateInactive),
					},
				},
			}, nil
		},
		createFn: func(context.Context, clusterplacementgroupssdk.CreateClusterPlacementGroupRequest) (clusterplacementgroupssdk.CreateClusterPlacementGroupResponse, error) {
			createCalls++
			return clusterplacementgroupssdk.CreateClusterPlacementGroupResponse{}, nil
		},
	})

	_, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want duplicate list match failure")
	}
	if !strings.Contains(err.Error(), "multiple matching resources") {
		t.Fatalf("CreateOrUpdate() error = %v, want duplicate match failure", err)
	}
	if createCalls != 0 {
		t.Fatalf("CreateClusterPlacementGroup() calls = %d, want 0 after duplicate list match", createCalls)
	}
}

func newTestClusterPlacementGroupClient(client clusterPlacementGroupOCIClient) ClusterPlacementGroupServiceClient {
	return newClusterPlacementGroupServiceClientWithOCIClient(loggerutil.OSOKLogger{}, client)
}

func makeClusterPlacementGroupResource() *clusterplacementgroupsv1beta1.ClusterPlacementGroup {
	return &clusterplacementgroupsv1beta1.ClusterPlacementGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "clusterplacementgroup-sample",
			Namespace: "default",
		},
		Spec: clusterplacementgroupsv1beta1.ClusterPlacementGroupSpec{
			Name:                      "clusterplacementgroup-sample",
			ClusterPlacementGroupType: string(clusterplacementgroupssdk.ClusterPlacementGroupTypeStandard),
			Description:               "cluster placement group",
			AvailabilityDomain:        "AD-1",
			CompartmentId:             "ocid1.compartment.oc1..exampleuniqueID",
			FreeformTags: map[string]string{
				"env": "dev",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"costCenter": "42",
				},
			},
		},
	}
}

func makeSDKClusterPlacementGroup(
	id string,
	resource *clusterplacementgroupsv1beta1.ClusterPlacementGroup,
	lifecycleState clusterplacementgroupssdk.ClusterPlacementGroupLifecycleStateEnum,
) clusterplacementgroupssdk.ClusterPlacementGroup {
	if resource == nil {
		resource = makeClusterPlacementGroupResource()
	}

	return clusterplacementgroupssdk.ClusterPlacementGroup{
		Id:                        common.String(id),
		Name:                      common.String(resource.Spec.Name),
		Description:               common.String(resource.Spec.Description),
		CompartmentId:             common.String(resource.Spec.CompartmentId),
		AvailabilityDomain:        common.String(resource.Spec.AvailabilityDomain),
		ClusterPlacementGroupType: clusterplacementgroupssdk.ClusterPlacementGroupTypeEnum(resource.Spec.ClusterPlacementGroupType),
		LifecycleState:            lifecycleState,
		FreeformTags:              mapsCloneString(resource.Spec.FreeformTags),
		DefinedTags:               clusterPlacementGroupDefinedTagsFromSpec(resource.Spec.DefinedTags),
	}
}

func makeSDKClusterPlacementGroupSummary(
	id string,
	resource *clusterplacementgroupsv1beta1.ClusterPlacementGroup,
	lifecycleState clusterplacementgroupssdk.ClusterPlacementGroupLifecycleStateEnum,
) clusterplacementgroupssdk.ClusterPlacementGroupSummary {
	if resource == nil {
		resource = makeClusterPlacementGroupResource()
	}

	return clusterplacementgroupssdk.ClusterPlacementGroupSummary{
		Id:                        common.String(id),
		Name:                      common.String(resource.Spec.Name),
		CompartmentId:             common.String(resource.Spec.CompartmentId),
		AvailabilityDomain:        common.String(resource.Spec.AvailabilityDomain),
		ClusterPlacementGroupType: clusterplacementgroupssdk.ClusterPlacementGroupTypeEnum(resource.Spec.ClusterPlacementGroupType),
		LifecycleState:            lifecycleState,
		FreeformTags:              mapsCloneString(resource.Spec.FreeformTags),
		DefinedTags:               clusterPlacementGroupDefinedTagsFromSpec(resource.Spec.DefinedTags),
	}
}

func makeClusterPlacementGroupWorkRequest(
	workRequestID string,
	operationType clusterplacementgroupssdk.OperationTypeEnum,
	status clusterplacementgroupssdk.OperationStatusEnum,
	action clusterplacementgroupssdk.ActionTypeEnum,
	resourceID string,
) clusterplacementgroupssdk.WorkRequest {
	resources := []clusterplacementgroupssdk.WorkRequestResource{}
	if strings.TrimSpace(resourceID) != "" {
		resources = append(resources, clusterplacementgroupssdk.WorkRequestResource{
			EntityType: common.String("ClusterPlacementGroup"),
			ActionType: action,
			Identifier: common.String(resourceID),
			EntityUri:  common.String("/20230801/clusterPlacementGroups/" + resourceID),
		})
	}
	return clusterplacementgroupssdk.WorkRequest{
		Id:            common.String(workRequestID),
		OperationType: operationType,
		Status:        status,
		Resources:     resources,
	}
}

func requireClusterPlacementGroupStringPtr(t *testing.T, field string, actual *string, want string) {
	t.Helper()
	if actual == nil {
		t.Fatalf("%s = nil, want %q", field, want)
	}
	if *actual != want {
		t.Fatalf("%s = %q, want %q", field, *actual, want)
	}
}

func requireClusterPlacementGroupAsyncCurrent(
	t *testing.T,
	resource *clusterplacementgroupsv1beta1.ClusterPlacementGroup,
	wantPhase shared.OSOKAsyncPhase,
	wantWorkRequestID string,
	wantClass shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()
	if resource == nil {
		t.Fatal("resource = nil")
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatal("status.async.current = nil")
	}
	if current.Phase != wantPhase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, wantPhase)
	}
	if current.WorkRequestID != wantWorkRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, wantWorkRequestID)
	}
	if current.NormalizedClass != wantClass {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, wantClass)
	}
}

func assertClusterPlacementGroupStringSliceEqual(t *testing.T, field string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
}

func mapsCloneString(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
