/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"testing"

	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestServiceClientCreateOrUpdateClearsTrackedIdentityThroughHookWhenIdentityGuardSkipsStaleTrackedIDReuse(t *testing.T) {
	t.Parallel()

	clearTrackedIdentityCalled := false
	guardCalls := 0
	listCalled := false

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"id", "name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				UpdatingStates:     []string{"UPDATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "best-effort",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
		},
		TrackedRecreate: TrackedRecreateHooks[*fakeResource]{
			ClearTrackedIdentity: func(resource *fakeResource) {
				clearTrackedIdentityCalled = true
				resource.Status = fakeStatus{}
			},
		},
		Identity: IdentityHooks[*fakeResource]{
			GuardExistingBeforeCreate: func(context.Context, *fakeResource) (ExistingBeforeCreateDecision, error) {
				guardCalls++
				return ExistingBeforeCreateDecisionSkip, nil
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createRequest := *request.(*fakeCreateThingRequest)
				if createRequest.DisplayName != "created-name" {
					t.Fatalf("create request displayName = %q, want created-name", createRequest.DisplayName)
				}
				return fakeCreateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..created",
						DisplayName:    "created-name",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				requireThingIDRequest(t, "get", request.(*fakeGetThingRequest).ThingId, "ocid1.thing.oc1..stale")
				return nil, errortest.NewServiceError(404, "NotFound", "thing not found")
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listCalled = true
				if request.(*fakeListThingRequest).Id != "" {
					t.Fatalf("list request id = %q, want empty after stale tracked ID fallback", request.(*fakeListThingRequest).Id)
				}
				return fakeListThingResponse{Collection: fakeThingCollection{}}, nil
			},
			Fields: []RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "Id", RequestName: "id", Contribution: "query"},
				{FieldName: "Name", RequestName: "name", Contribution: "query"},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..match",
			DisplayName:   "created-name",
			Name:          "wanted",
		},
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..stale"},
			Id:         "ocid1.thing.oc1..stale",
			Partitions: 7,
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	if !clearTrackedIdentityCalled {
		t.Fatal("CreateOrUpdate() should clear tracked identity when the tracked ID is stale")
	}
	if guardCalls != 1 {
		t.Fatalf("GuardExistingBeforeCreate() calls = %d, want 1", guardCalls)
	}
	if listCalled {
		t.Fatal("CreateOrUpdate() should skip list reuse when the identity guard rejects stale tracked-ID fallback reuse")
	}
	requireStatusOCID(t, resource, "ocid1.thing.oc1..created")
	requireStringEqual(t, "status.displayName", resource.Status.DisplayName, "created-name")
	if resource.Status.Partitions != 0 {
		t.Fatalf("status.partitions = %d, want 0 after stale tracked status is cleared", resource.Status.Partitions)
	}
}

func TestServiceClientCreateOrUpdateSkipsPreCreateReuseWhenIdentityGuardSkips(t *testing.T) {
	t.Parallel()

	var createCalled bool
	var lookupCalls int
	var listCalls int

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"name", "compartmentId"},
			},
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				UpdatingStates:     []string{"UPDATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "best-effort",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				return fakeCreateThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..created",
						Name:           "thing-a",
						DisplayName:    "created-name",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				listCalls++
				return fakeListThingResponse{}, nil
			},
			Fields: []RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "Name", RequestName: "name", Contribution: "query"},
			},
		},
		Identity: IdentityHooks[*fakeResource]{
			Resolve: func(resource *fakeResource) (any, error) {
				return fakePathIdentity{
					parentID:    resource.Spec.CompartmentId,
					thingName:   resource.Spec.Name,
					syntheticID: "thing/" + resource.Spec.Name,
				}, nil
			},
			GuardExistingBeforeCreate: func(context.Context, *fakeResource) (ExistingBeforeCreateDecision, error) {
				return ExistingBeforeCreateDecisionSkip, nil
			},
			LookupExisting: func(context.Context, *fakeResource, any) (any, error) {
				lookupCalls++
				return fakeGetThingResponse{}, nil
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..parent",
			Name:          "thing-a",
			DisplayName:   "created-name",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	if !createCalled {
		t.Fatal("CreateOrUpdate() should call Create() when the identity guard skips pre-create reuse")
	}
	if lookupCalls != 0 {
		t.Fatalf("LookupExisting() calls = %d, want 0 when pre-create reuse is skipped", lookupCalls)
	}
	if listCalls != 0 {
		t.Fatalf("List() calls = %d, want 0 when pre-create reuse is skipped", listCalls)
	}
	requireStatusOCID(t, resource, "ocid1.thing.oc1..created")
}

func TestServiceClientCreateOrUpdateUsesIdentityLookupAndPersistsTrackedIdentity(t *testing.T) {
	t.Parallel()

	var createCalled bool
	var guardCalls []string
	var lookupCalls int
	var seedCalls int
	var recordTrackedCalls int

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				createCalled = true
				return fakeCreateThingResponse{}, nil
			},
		},
		Identity: IdentityHooks[*fakeResource]{
			Resolve: func(resource *fakeResource) (any, error) {
				return fakePathIdentity{
					parentID:    resource.Spec.CompartmentId,
					thingName:   resource.Spec.Name,
					syntheticID: "thing/" + resource.Spec.Name,
				}, nil
			},
			RecordPath: func(resource *fakeResource, identity any) {
				resolved := identity.(fakePathIdentity)
				resource.Status.CompartmentId = resolved.parentID
				resource.Status.Name = resolved.thingName
			},
			GuardExistingBeforeCreate: func(context.Context, *fakeResource) (ExistingBeforeCreateDecision, error) {
				guardCalls = append(guardCalls, "guard")
				return ExistingBeforeCreateDecisionAllow, nil
			},
			LookupExisting: func(context.Context, *fakeResource, any) (any, error) {
				guardCalls = append(guardCalls, "lookup")
				lookupCalls++
				return fakeGetThingResponse{
					Thing: fakeThing{
						Name:           "thing-a",
						DisplayName:    "existing-display",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
			SeedSyntheticTrackedID: func(resource *fakeResource, identity any) func() {
				seedCalls++
				previous := resource.Status.OsokStatus.Ocid
				resource.Status.OsokStatus.Ocid = shared.OCID(identity.(fakePathIdentity).syntheticID)
				return func() {
					resource.Status.OsokStatus.Ocid = previous
				}
			},
			RecordTracked: func(resource *fakeResource, _ any, resourceID string) {
				recordTrackedCalls++
				resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..parent",
			Name:          "thing-a",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireRequeueState(t, response, false, "CreateOrUpdate() should not requeue for ACTIVE lifecycle")
	if createCalled {
		t.Fatal("CreateOrUpdate() invoked Create instead of reusing the looked-up resource")
	}
	if lookupCalls != 1 {
		t.Fatalf("LookupExisting() calls = %d, want 1", lookupCalls)
	}
	if len(guardCalls) != 2 || guardCalls[0] != "guard" || guardCalls[1] != "lookup" {
		t.Fatalf("guard/lookup call order = %v, want [guard lookup]", guardCalls)
	}
	if seedCalls != 1 {
		t.Fatalf("SeedSyntheticTrackedID() calls = %d, want 1", seedCalls)
	}
	if recordTrackedCalls != 1 {
		t.Fatalf("RecordTracked() calls = %d, want 1", recordTrackedCalls)
	}
	requireStatusOCID(t, resource, "thing/thing-a")
	requireStringEqual(t, "status.compartmentId", resource.Status.CompartmentId, "ocid1.compartment.oc1..parent")
	requireStringEqual(t, "status.name", resource.Status.Name, "thing-a")
	requireStringEqual(t, "status.displayName", resource.Status.DisplayName, "existing-display")
	requireTrailingCondition(t, resource, shared.Active)
}

func TestServiceClientCreateOrUpdateUsesReadGetAdapterWhenConfigured(t *testing.T) {
	t.Parallel()

	var getRequest fakeNestedGetThingRequest

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Read: ReadHooks{
			Get: &Operation{
				NewRequest: func() any { return &fakeNestedGetThingRequest{} },
				Fields: []RequestField{
					{FieldName: "ParentId", RequestName: "parentId", Contribution: "path", LookupPaths: []string{"status.compartmentId", "spec.compartmentId"}},
					{FieldName: "ThingName", RequestName: "thingName", Contribution: "path", LookupPaths: []string{"status.name", "spec.name", "name"}},
				},
				Call: func(_ context.Context, request any) (any, error) {
					getRequest = *request.(*fakeNestedGetThingRequest)
					return fakeGetThingResponse{
						Thing: fakeThing{
							Id:             "ocid1.thing.oc1..nested",
							Name:           "thing-read",
							DisplayName:    "nested-display",
							LifecycleState: "ACTIVE",
						},
					}, nil
				},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..read",
			Name:          "thing-read",
		},
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: "thing/thing-read",
			},
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireThingIDRequest(t, "read parent", getRequest.ParentId, "ocid1.compartment.oc1..read")
	requireThingIDRequest(t, "read name", getRequest.ThingName, "thing-read")
	requireStatusOCID(t, resource, "ocid1.thing.oc1..nested")
	requireStringEqual(t, "status.displayName", resource.Status.DisplayName, "nested-display")
}

func TestServiceClientCreateOrUpdateUsesReadListAdapterWhenConfigured(t *testing.T) {
	t.Parallel()

	var listRequest fakeNestedListThingRequest

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy: "best-effort",
			},
			List: &ListSemantics{
				ResponseItemsField: "Resources",
				MatchFields:        []string{"name"},
			},
		},
		Read: ReadHooks{
			List: &Operation{
				NewRequest: func() any { return &fakeNestedListThingRequest{} },
				Fields: []RequestField{
					{FieldName: "ParentId", RequestName: "parentId", Contribution: "path", LookupPaths: []string{"status.compartmentId", "spec.compartmentId"}},
				},
				Call: func(_ context.Context, request any) (any, error) {
					listRequest = *request.(*fakeNestedListThingRequest)
					return fakeNamedListThingResponse{
						Collection: fakeNamedThingCollection{
							Resources: []fakeThingSummary{
								{Id: "ocid1.thing.oc1..other", Name: "other", LifecycleState: "ACTIVE"},
								{Id: "ocid1.thing.oc1..list", Name: "thing-list", LifecycleState: "ACTIVE"},
							},
						},
					}, nil
				},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..list",
			Name:          "thing-list",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireThingIDRequest(t, "list parent", listRequest.ParentId, "ocid1.compartment.oc1..list")
	requireStatusOCID(t, resource, "ocid1.thing.oc1..list")
	requireStringEqual(t, "status.name", resource.Status.Name, "thing-list")
}
