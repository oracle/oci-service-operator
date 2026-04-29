/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func TestServiceClientDeleteErrorHookCanReplaceAndProjectError(t *testing.T) {
	t.Parallel()

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		DeleteHooks: DeleteHooks[*fakeResource]{
			HandleError: func(resource *fakeResource, err error) error {
				servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
				return errors.New("custom delete failure")
			},
		},
		Delete: &Operation{
			NewRequest: func() any { return &fakeDeleteThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return nil, errortest.NewServiceError(409, "IncorrectState", "delete conflict")
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})

	resource := &fakeResource{
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..delete"},
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err == nil || err.Error() != "custom delete failure" {
		t.Fatalf("Delete() error = %v, want custom delete failure", err)
	}
	if deleted {
		t.Fatal("Delete() should not report success when the delete hook returns an error")
	}
	requireStatusOpcRequestID(t, resource, "opc-request-id")
}

func TestServiceClientDeleteOutcomeHookCanShortCircuitAlreadyPendingDelete(t *testing.T) {
	t.Parallel()

	deleteCalls := 0
	confirmReadCalls := 0
	var gotStage DeleteConfirmStage

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Delete: DeleteSemantics{
				Policy:         "required",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
			DeleteFollowUp: FollowUpSemantics{
				Strategy: "confirm-delete",
				Hooks:    []Hook{{Helper: "tfresource.DeleteResource"}},
			},
		},
		DeleteHooks: DeleteHooks[*fakeResource]{
			ConfirmRead: func(_ context.Context, resource *fakeResource, currentID string) (any, error) {
				confirmReadCalls++
				if currentID != "ocid1.thing.oc1..delete" {
					t.Fatalf("DeleteHooks.ConfirmRead() currentID = %q, want existing OCID", currentID)
				}
				return fakeThing{
					Id:             currentID,
					LifecycleState: "CANCELING",
				}, nil
			},
			ApplyOutcome: func(resource *fakeResource, _ any, stage DeleteConfirmStage) (DeleteOutcome, error) {
				gotStage = stage
				resource.Status.OsokStatus.Message = "custom pending delete"
				return DeleteOutcome{Handled: true, Deleted: false}, nil
			},
		},
		Delete: &Operation{
			NewRequest: func() any { return &fakeDeleteThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				deleteCalls++
				return nil, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})

	resource := &fakeResource{
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..delete"},
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep waiting when the delete outcome hook reports pending")
	}
	if deleteCalls != 0 {
		t.Fatalf("Delete() calls = %d, want 0 once the outcome hook handles the already-pending read", deleteCalls)
	}
	if confirmReadCalls != 1 {
		t.Fatalf("DeleteHooks.ConfirmRead() calls = %d, want 1", confirmReadCalls)
	}
	if gotStage != DeleteConfirmStageAlreadyPending {
		t.Fatalf("DeleteHooks.ApplyOutcome() stage = %q, want %q", gotStage, DeleteConfirmStageAlreadyPending)
	}
	if resource.Status.LifecycleState != "CANCELING" {
		t.Fatalf("status.lifecycleState = %q, want CANCELING", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.Message != "custom pending delete" {
		t.Fatalf("status.message = %q, want custom pending delete", resource.Status.OsokStatus.Message)
	}
}

func TestServiceClientDeleteOutcomeHookCanHandlePostDeleteConfirmRead(t *testing.T) {
	t.Parallel()

	var deleteRequest fakeDeleteThingRequest
	confirmReadCalls := 0
	var gotStage DeleteConfirmStage

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Delete: DeleteSemantics{
				Policy:         "required",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
			DeleteFollowUp: FollowUpSemantics{
				Strategy: "confirm-delete",
				Hooks:    []Hook{{Helper: "tfresource.DeleteResource"}},
			},
		},
		DeleteHooks: DeleteHooks[*fakeResource]{
			ConfirmRead: func(_ context.Context, resource *fakeResource, currentID string) (any, error) {
				confirmReadCalls++
				if currentID != "ocid1.thing.oc1..delete" {
					t.Fatalf("DeleteHooks.ConfirmRead() currentID = %q, want existing OCID", currentID)
				}
				return fakeThing{
					Id:             currentID,
					LifecycleState: "ACTIVE",
				}, nil
			},
			ApplyOutcome: func(resource *fakeResource, _ any, stage DeleteConfirmStage) (DeleteOutcome, error) {
				gotStage = stage
				if stage == DeleteConfirmStageAlreadyPending {
					return DeleteOutcome{}, nil
				}
				resource.Status.OsokStatus.Message = "hooked delete still settling"
				return DeleteOutcome{Handled: true, Deleted: false}, nil
			},
		},
		Delete: &Operation{
			NewRequest: func() any { return &fakeDeleteThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				deleteRequest = *request.(*fakeDeleteThingRequest)
				return fakeDeleteThingResponse{OpcRequestId: stringPtr("opc-delete-1")}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})

	resource := &fakeResource{
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..delete"},
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep waiting when the post-delete outcome hook reports pending")
	}
	requireThingIDRequest(t, "delete", deleteRequest.ThingId, "ocid1.thing.oc1..delete")
	if confirmReadCalls != 2 {
		t.Fatalf("DeleteHooks.ConfirmRead() calls = %d, want 2", confirmReadCalls)
	}
	if gotStage != DeleteConfirmStageAfterRequest {
		t.Fatalf("DeleteHooks.ApplyOutcome() stage = %q, want %q", gotStage, DeleteConfirmStageAfterRequest)
	}
	if resource.Status.LifecycleState != "ACTIVE" {
		t.Fatalf("status.lifecycleState = %q, want ACTIVE", resource.Status.LifecycleState)
	}
	if resource.Status.OsokStatus.Message != "hooked delete still settling" {
		t.Fatalf("status.message = %q, want hooked delete still settling", resource.Status.OsokStatus.Message)
	}
	requireStatusOpcRequestID(t, resource, "opc-delete-1")
}

func TestServiceClientDeleteUsesTemporarySyntheticTrackedIDWhenConfigured(t *testing.T) {
	t.Parallel()

	var deleteRequest fakeDeleteThingRequest
	var seedCalls int

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Delete: &Operation{
			NewRequest: func() any { return &fakeDeleteThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				deleteRequest = *request.(*fakeDeleteThingRequest)
				return fakeDeleteThingResponse{}, nil
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
				resource.Status.CompartmentId = identity.(fakePathIdentity).parentID
			},
			SeedSyntheticTrackedID: func(resource *fakeResource, identity any) func() {
				seedCalls++
				previous := resource.Status.OsokStatus.Ocid
				resource.Status.OsokStatus.Ocid = shared.OCID(identity.(fakePathIdentity).syntheticID)
				return func() {
					resource.Status.OsokStatus.Ocid = previous
				}
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			CompartmentId: "ocid1.compartment.oc1..delete",
			Name:          "thing-delete",
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want true")
	}
	if seedCalls != 1 {
		t.Fatalf("SeedSyntheticTrackedID() calls = %d, want 1", seedCalls)
	}
	requireThingIDRequest(t, "delete", deleteRequest.ThingId, "thing/thing-delete")
	requireStringEqual(t, "status.compartmentId", resource.Status.CompartmentId, "ocid1.compartment.oc1..delete")
	requireStatusOCID(t, resource, "")
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt should be set after successful delete")
	}
}
