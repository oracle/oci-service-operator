/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"testing"
)

func TestServiceClientDeleteTreatsNotFoundAsDeleted(t *testing.T) {
	t.Parallel()
	var deleteRequest fakeDeleteThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Delete: &Operation{NewRequest: func() any {
		return &fakeDeleteThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		deleteRequest = *request.(*fakeDeleteThingRequest)
		return fakeDeleteThingResponse{}, nil
	}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		return nil, errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "thing not found")
	}}})
	resource := &fakeResource{Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..delete"}}}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should report success when OCI returns not found")
	}
	if deleteRequest.ThingId == nil || *deleteRequest.ThingId != "ocid1.thing.oc1..delete" {
		t.Fatalf("delete request thingId = %v, want existing OCID", deleteRequest.ThingId)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt should be set after confirmed deletion")
	}
}

func TestServiceClientDeleteUsesFormalRequiredConfirmation(t *testing.T) {
	t.Parallel()
	var deleteRequest fakeDeleteThingRequest
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Delete: DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}, DeleteFollowUp: FollowUpSemantics{Strategy: "confirm-delete", Hooks: []Hook{{Helper: "tfresource.DeleteResource"}}}}, Delete: &Operation{NewRequest: func() any {
		return &fakeDeleteThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		deleteRequest = *request.(*fakeDeleteThingRequest)
		return fakeDeleteThingResponse{}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..delete", LifecycleState: "DELETING"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}})
	resource := &fakeResource{Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..delete"}}}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep waiting while lifecycle is DELETING")
	}
	if deleteRequest.ThingId != nil {
		t.Fatalf("delete request thingId = %v, want no delete request once confirm-delete already reports DELETING", deleteRequest.ThingId)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Terminating {
		t.Fatalf("status conditions = %#v, want trailing Terminating condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestServiceClientDeleteConflictStillConfirmsFormalPendingState(t *testing.T) {
	t.Parallel()
	var deleteRequest fakeDeleteThingRequest
	getCalls := 0
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Delete: DeleteSemantics{Policy: "required", PendingStates: []string{"TERMINATING"}, TerminalStates: []string{"TERMINATED"}}, DeleteFollowUp: FollowUpSemantics{Strategy: "confirm-delete", Hooks: []Hook{{Helper: "tfresource.DeleteResource"}}}}, Delete: &Operation{NewRequest: func() any {
		return &fakeDeleteThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		deleteRequest = *request.(*fakeDeleteThingRequest)
		return nil, errortest.NewServiceError(409, "IncorrectState", "delete is still settling")
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getCalls++
		lifecycleState := "ACTIVE"
		if getCalls > 1 {
			lifecycleState = "TERMINATING"
		}
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..delete", LifecycleState: lifecycleState}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}})
	resource := &fakeResource{Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..delete"}}}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep waiting while the confirmed lifecycle remains TERMINATING")
	}
	if getCalls != 2 {
		t.Fatalf("Get() calls = %d, want pre-delete read plus one follow-up after conflict", getCalls)
	}
	if deleteRequest.ThingId == nil || *deleteRequest.ThingId != "ocid1.thing.oc1..delete" {
		t.Fatalf("delete request thingId = %v, want existing OCID", deleteRequest.ThingId)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay empty while delete remains pending")
	}
	if resource.Status.LifecycleState != "TERMINATING" {
		t.Fatalf("status.lifecycleState = %q, want TERMINATING", resource.Status.LifecycleState)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Terminating {
		t.Fatalf("status conditions = %#v, want trailing Terminating condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestServiceClientDeleteSkipsReissuingDeleteWhenFormalStateAlreadyPending(t *testing.T) {
	t.Parallel()
	deleteCalls := 0
	getCalls := 0
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Delete: DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}, DeleteFollowUp: FollowUpSemantics{Strategy: "confirm-delete", Hooks: []Hook{{Helper: "tfresource.DeleteResource"}}}}, Delete: &Operation{NewRequest: func() any {
		return &fakeDeleteThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		deleteCalls++
		t.Fatal("Delete() should not be called once delete confirmation already reports DELETING")
		return nil, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getCalls++
		getRequest := request.(*fakeGetThingRequest)
		if getRequest.ThingId == nil || *getRequest.ThingId != "ocid1.thing.oc1..delete" {
			t.Fatalf("get request thingId = %v, want existing OCID", getRequest.ThingId)
		}
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..delete", LifecycleState: "DELETING"}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}})
	resource := &fakeResource{Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..delete"}}}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep waiting while the confirmed lifecycle remains DELETING")
	}
	if deleteCalls != 0 {
		t.Fatalf("Delete() calls = %d, want 0 once delete is already pending", deleteCalls)
	}
	if getCalls != 1 {
		t.Fatalf("Get() calls = %d, want 1 confirmation read", getCalls)
	}
	if resource.Status.LifecycleState != "DELETING" {
		t.Fatalf("status.lifecycleState = %q, want DELETING", resource.Status.LifecycleState)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Terminating {
		t.Fatalf("status conditions = %#v, want trailing Terminating condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestServiceClientDeleteConflictStillConfirmsFormalTerminalState(t *testing.T) {
	t.Parallel()
	getCalls := 0
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Delete: DeleteSemantics{Policy: "required", PendingStates: []string{"TERMINATING"}, TerminalStates: []string{"TERMINATED"}}, DeleteFollowUp: FollowUpSemantics{Strategy: "confirm-delete", Hooks: []Hook{{Helper: "tfresource.DeleteResource"}}}}, Delete: &Operation{NewRequest: func() any {
		return &fakeDeleteThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		return nil, errortest.NewServiceError(409, "IncorrectState", "delete is still settling")
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getCalls++
		lifecycleState := "ACTIVE"
		if getCalls > 1 {
			lifecycleState = "TERMINATED"
		}
		return fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..delete", LifecycleState: lifecycleState}}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}})
	resource := &fakeResource{Status: fakeStatus{OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..delete"}}}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should succeed once the conflict follow-up confirms TERMINATED")
	}
	if getCalls != 2 {
		t.Fatalf("Get() calls = %d, want pre-delete read plus one follow-up after conflict", getCalls)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt should be set after confirmed terminal delete")
	}
	if resource.Status.LifecycleState != "TERMINATED" {
		t.Fatalf("status.lifecycleState = %q, want TERMINATED", resource.Status.LifecycleState)
	}
	if len(resource.Status.OsokStatus.Conditions) == 0 || resource.Status.OsokStatus.Conditions[len(resource.Status.OsokStatus.Conditions)-1].Type != shared.Terminating {
		t.Fatalf("status conditions = %#v, want trailing Terminating condition", resource.Status.OsokStatus.Conditions)
	}
}

func TestServiceClientDeleteResolvesDeletePhaseListMatchWithoutOcid(t *testing.T) {
	t.Parallel()
	var deleteRequest fakeDeleteThingRequest
	getCalled := false
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{List: &ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name", "compartmentId"}}, Lifecycle: LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}}, Delete: DeleteSemantics{Policy: "best-effort", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}}, DeleteFollowUp: FollowUpSemantics{Strategy: "none"}}, Get: &Operation{NewRequest: func() any {
		return &fakeGetThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		getCalled = true
		if request.(*fakeGetThingRequest).ThingId == nil {
			t.Fatal("Get() should not be called without a resource OCID")
		}
		return fakeGetThingResponse{}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}, List: &Operation{NewRequest: func() any {
		return &fakeListThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		return fakeListThingResponse{Collection: fakeThingCollection{Items: []fakeThingSummary{{Id: "ocid1.thing.oc1..deleting", Name: "wanted", CompartmentId: "ocid1.compartment.oc1..match", LifecycleState: "DELETING"}}}}, nil
	}, Fields: []RequestField{{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"}, {FieldName: "Name", RequestName: "name", Contribution: "query"}}}, Delete: &Operation{NewRequest: func() any {
		return &fakeDeleteThingRequest{}
	}, Call: func(_ context.Context, request any) (any, error) {
		deleteRequest = *request.(*fakeDeleteThingRequest)
		return fakeDeleteThingResponse{}, nil
	}, Fields: []RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..match", Name: "wanted"}}
	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should resolve delete-phase list matches without a recorded OCID")
	}
	if getCalled {
		t.Fatal("Get() should be skipped when delete resolution has no recorded OCID")
	}
	if deleteRequest.ThingId == nil || *deleteRequest.ThingId != "ocid1.thing.oc1..deleting" {
		t.Fatalf("delete request thingId = %v, want delete-phase OCID", deleteRequest.ThingId)
	}
}
