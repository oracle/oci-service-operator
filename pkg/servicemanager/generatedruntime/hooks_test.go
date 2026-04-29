/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestServiceClientCreateOrUpdateFailsBeforePreCreateReuseWhenIdentityGuardFails(t *testing.T) {
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
				return fakeCreateThingResponse{}, nil
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
				return ExistingBeforeCreateDecisionFail, errors.New("Thing spec.displayName is required before pre-create reuse")
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
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want identity guard failure")
	}
	if err.Error() != "Thing spec.displayName is required before pre-create reuse" {
		t.Fatalf("CreateOrUpdate() error = %v, want identity guard failure", err)
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful guard failure", response)
	}
	if createCalled {
		t.Fatal("CreateOrUpdate() should not call Create() when the identity guard fails")
	}
	if lookupCalls != 0 {
		t.Fatalf("LookupExisting() calls = %d, want 0 when the identity guard fails", lookupCalls)
	}
	if listCalls != 0 {
		t.Fatalf("List() calls = %d, want 0 when the identity guard fails", listCalls)
	}
	requireTrailingCondition(t, resource, shared.Failed)
}

func TestServiceClientCreateOrUpdateRestoresStatusAfterDelegateFailure(t *testing.T) {
	t.Parallel()

	type statusRestoreState struct {
		previous fakeStatus
		cleared  fakeStatus
	}

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		StatusHooks: StatusHooks[*fakeResource]{
			ClearProjectedStatus: func(resource *fakeResource) any {
				state := statusRestoreState{previous: resource.Status}
				resource.Status = fakeStatus{
					OsokStatus: resource.Status.OsokStatus,
					Id:         resource.Status.Id,
				}
				state.cleared = resource.Status
				return state
			},
			RestoreStatus: func(resource *fakeResource, baseline any) {
				state := baseline.(statusRestoreState)
				failedStatus := resource.Status.OsokStatus
				resource.Status = state.previous
				resource.Status.OsokStatus = failedStatus
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return nil, io.ErrUnexpectedEOF
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			DisplayName: "desired-name",
		},
		Status: fakeStatus{
			DisplayName: "persisted-name",
			Partitions:  9,
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want delegate failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful failure response", response)
	}
	requireStringEqual(t, "status.displayName", resource.Status.DisplayName, "persisted-name")
	if resource.Status.Partitions != 9 {
		t.Fatalf("status.partitions = %d, want restored value 9", resource.Status.Partitions)
	}
	requireStringEqual(t, "status.reason", resource.Status.OsokStatus.Reason, string(shared.Failed))
}

func TestServiceClientCreateOrUpdateSkipsStatusRestoreWhenDelegateProjectsFreshStatus(t *testing.T) {
	t.Parallel()

	type statusRestoreState struct {
		previous fakeStatus
		cleared  fakeStatus
	}

	normalizeStatus := func(status fakeStatus) string {
		status.OsokStatus = shared.OSOKStatus{}
		payload, err := json.Marshal(status)
		if err != nil {
			t.Fatalf("Marshal(status) error = %v", err)
		}
		return string(payload)
	}

	resource := &fakeResource{
		Spec: fakeSpec{
			DisplayName: "desired-name",
		},
		Status: fakeStatus{
			DisplayName: "persisted-name",
		},
	}

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		StatusHooks: StatusHooks[*fakeResource]{
			ClearProjectedStatus: func(resource *fakeResource) any {
				state := statusRestoreState{previous: resource.Status}
				resource.Status = fakeStatus{
					OsokStatus: resource.Status.OsokStatus,
					Id:         resource.Status.Id,
				}
				state.cleared = resource.Status
				return state
			},
			ShouldRestoreOnFailure: func(resource *fakeResource, baseline any) bool {
				state := baseline.(statusRestoreState)
				return normalizeStatus(resource.Status) == normalizeStatus(state.cleared)
			},
			RestoreStatus: func(resource *fakeResource, baseline any) {
				state := baseline.(statusRestoreState)
				failedStatus := resource.Status.OsokStatus
				resource.Status = state.previous
				resource.Status.OsokStatus = failedStatus
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				resource.Status.DisplayName = "fresh-projection"
				return nil, io.ErrUnexpectedEOF
			},
		},
	})

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want delegate failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful failure response", response)
	}
	requireStringEqual(t, "status.displayName", resource.Status.DisplayName, "fresh-projection")
	requireStringEqual(t, "status.reason", resource.Status.OsokStatus.Reason, string(shared.Failed))
}

func TestServiceClientCreateOrUpdateAppliesParityUpdateHooks(t *testing.T) {
	t.Parallel()

	normalized := false
	validated := false
	parityUpdateCalled := false
	defaultUpdateCalled := false

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ActiveStates: []string{"ACTIVE"},
			},
			Mutation: MutationSemantics{
				Mutable: []string{"displayName"},
			},
		},
		ParityHooks: ParityHooks[*fakeResource]{
			NormalizeDesiredState: func(resource *fakeResource, _ any) {
				normalized = true
				resource.Spec.DisplayName = "normalized-name"
			},
			ValidateCreateOnlyDrift: func(resource *fakeResource, currentResponse any) error {
				validated = true
				if _, ok := currentResponse.(fakeGetThingResponse); !ok {
					t.Fatalf("ValidateCreateOnlyDrift() currentResponse = %T, want fakeGetThingResponse", currentResponse)
				}
				return nil
			},
			RequiresParityHandling: func(resource *fakeResource, currentResponse any) bool {
				if !normalized {
					t.Fatal("RequiresParityHandling() ran before NormalizeDesiredState()")
				}
				if _, ok := currentResponse.(fakeGetThingResponse); !ok {
					t.Fatalf("RequiresParityHandling() currentResponse = %T, want fakeGetThingResponse", currentResponse)
				}
				return true
			},
			ApplyParityUpdate: func(_ context.Context, resource *fakeResource, currentResponse any) (servicemanager.OSOKResponse, error) {
				parityUpdateCalled = true
				if !validated {
					t.Fatal("ApplyParityUpdate() ran before ValidateCreateOnlyDrift()")
				}
				current := currentResponse.(fakeGetThingResponse)
				requireStringEqual(t, "current thing id", current.Thing.Id, "ocid1.thing.oc1..existing")
				requireStringEqual(t, "normalized spec displayName", resource.Spec.DisplayName, "normalized-name")
				resource.Status.DisplayName = "parity-updated"
				resource.Status.OsokStatus.Ocid = shared.OCID(current.Thing.Id)
				return servicemanager.OSOKResponse{IsSuccessful: true}, nil
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				requireThingIDRequest(t, "get", request.(*fakeGetThingRequest).ThingId, "ocid1.thing.oc1..existing")
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..existing",
						DisplayName:    "current-name",
						LifecycleState: "ACTIVE",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				defaultUpdateCalled = true
				t.Fatal("Update() should not be called when parity hooks handle the update path")
				return nil, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			DisplayName: "desired-name",
		},
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..existing"},
			Id:         "ocid1.thing.oc1..existing",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	if !parityUpdateCalled {
		t.Fatal("CreateOrUpdate() should route mutable drift through ApplyParityUpdate() when parity hooks require handling")
	}
	if defaultUpdateCalled {
		t.Fatal("CreateOrUpdate() should not invoke the default Update() operation when parity hooks handle the update path")
	}
	requireStatusOCID(t, resource, "ocid1.thing.oc1..existing")
	requireStringEqual(t, "status.displayName", resource.Status.DisplayName, "parity-updated")
}
