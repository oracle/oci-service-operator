/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicemanager

import (
	"testing"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func TestProjectAsyncCondition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		class         shared.OSOKAsyncNormalizedClass
		phase         shared.OSOKAsyncPhase
		wantCondition shared.OSOKConditionType
		wantRequeue   bool
		wantMessage   string
	}{
		{
			name:          "pending create",
			class:         shared.OSOKAsyncClassPending,
			phase:         shared.OSOKAsyncPhaseCreate,
			wantCondition: shared.Provisioning,
			wantRequeue:   true,
			wantMessage:   "OCI create is in progress",
		},
		{
			name:          "pending update",
			class:         shared.OSOKAsyncClassPending,
			phase:         shared.OSOKAsyncPhaseUpdate,
			wantCondition: shared.Updating,
			wantRequeue:   true,
			wantMessage:   "OCI update is in progress",
		},
		{
			name:          "pending delete",
			class:         shared.OSOKAsyncClassPending,
			phase:         shared.OSOKAsyncPhaseDelete,
			wantCondition: shared.Terminating,
			wantRequeue:   true,
			wantMessage:   "OCI delete is in progress",
		},
		{
			name:          "delete success keeps terminating until confirmed",
			class:         shared.OSOKAsyncClassSucceeded,
			phase:         shared.OSOKAsyncPhaseDelete,
			wantCondition: shared.Terminating,
			wantRequeue:   true,
			wantMessage:   "OCI delete completed; waiting for final confirmation",
		},
		{
			name:          "attention fails closed",
			class:         shared.OSOKAsyncClassAttention,
			phase:         shared.OSOKAsyncPhaseUpdate,
			wantCondition: shared.Failed,
			wantRequeue:   false,
			wantMessage:   "OCI update requires attention",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ProjectAsyncCondition(tt.class, tt.phase)
			if got.Condition != tt.wantCondition {
				t.Fatalf("ProjectAsyncCondition(%q, %q) condition = %q, want %q", tt.class, tt.phase, got.Condition, tt.wantCondition)
			}
			if got.ShouldRequeue != tt.wantRequeue {
				t.Fatalf("ProjectAsyncCondition(%q, %q) requeue = %t, want %t", tt.class, tt.phase, got.ShouldRequeue, tt.wantRequeue)
			}
			if got.DefaultMessage != tt.wantMessage {
				t.Fatalf("ProjectAsyncCondition(%q, %q) message = %q, want %q", tt.class, tt.phase, got.DefaultMessage, tt.wantMessage)
			}
		})
	}
}

func TestApplyAsyncOperationUpdatesSharedStatus(t *testing.T) {
	t.Parallel()

	percent := float32(42)
	status := &shared.OSOKStatus{}
	projection := ApplyAsyncOperation(status, &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:    "wr-update-1",
		RawStatus:        "IN_PROGRESS",
		RawOperationType: "UPDATE_QUEUE",
		NormalizedClass:  shared.OSOKAsyncClassPending,
		PercentComplete:  &percent,
	}, loggerutil.OSOKLogger{})

	if projection.Condition != shared.Updating {
		t.Fatalf("projection condition = %q, want %q", projection.Condition, shared.Updating)
	}
	if !projection.ShouldRequeue {
		t.Fatalf("projection should requeue = false, want true")
	}
	if status.Async.Current == nil {
		t.Fatalf("status.async.current = nil, want populated tracker")
	}
	if status.Async.Current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", status.Async.Current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if status.Async.Current.Phase != shared.OSOKAsyncPhaseUpdate {
		t.Fatalf("status.async.current.phase = %q, want %q", status.Async.Current.Phase, shared.OSOKAsyncPhaseUpdate)
	}
	if status.Async.Current.WorkRequestID != "wr-update-1" {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", status.Async.Current.WorkRequestID, "wr-update-1")
	}
	if status.Async.Current.RawStatus != "IN_PROGRESS" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", status.Async.Current.RawStatus, "IN_PROGRESS")
	}
	if status.Async.Current.RawOperationType != "UPDATE_QUEUE" {
		t.Fatalf("status.async.current.rawOperationType = %q, want %q", status.Async.Current.RawOperationType, "UPDATE_QUEUE")
	}
	if status.Async.Current.PercentComplete == nil || *status.Async.Current.PercentComplete != percent {
		t.Fatalf("status.async.current.percentComplete = %v, want %v", status.Async.Current.PercentComplete, percent)
	}
	if status.Message != "OCI update is in progress" {
		t.Fatalf("status.message = %q, want %q", status.Message, "OCI update is in progress")
	}
	if status.Reason != string(shared.Updating) {
		t.Fatalf("status.reason = %q, want %q", status.Reason, shared.Updating)
	}
	if status.UpdatedAt == nil {
		t.Fatalf("status.updatedAt = nil, want timestamp")
	}
	if len(status.Conditions) == 0 {
		t.Fatalf("status.conditions = %#v, want at least one entry", status.Conditions)
	}
}

func TestBuildWorkRequestAsyncOperationUsesNormalizedMappings(t *testing.T) {
	t.Parallel()

	percent := float32(12)
	current, err := BuildWorkRequestAsyncOperation(&shared.OSOKStatus{}, WorkRequestAsyncAdapter{
		PendingStatusTokens:   []string{"ACCEPTED", "IN_PROGRESS"},
		SucceededStatusTokens: []string{"SUCCEEDED"},
		FailedStatusTokens:    []string{"FAILED"},
		CanceledStatusTokens:  []string{"CANCELED"},
		CreateActionTokens:    []string{"CREATED"},
		UpdateActionTokens:    []string{"UPDATED"},
		DeleteActionTokens:    []string{"DELETED"},
	}, WorkRequestAsyncInput{
		RawStatus:        "IN_PROGRESS",
		RawAction:        "UPDATED",
		RawOperationType: "UPDATE_QUEUE",
		WorkRequestID:    "wr-update-1",
		PercentComplete:  &percent,
	})
	if err != nil {
		t.Fatalf("BuildWorkRequestAsyncOperation() error = %v", err)
	}
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("current.source = %q, want %q", current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if current.Phase != shared.OSOKAsyncPhaseUpdate {
		t.Fatalf("current.phase = %q, want %q", current.Phase, shared.OSOKAsyncPhaseUpdate)
	}
	if current.WorkRequestID != "wr-update-1" {
		t.Fatalf("current.workRequestId = %q, want %q", current.WorkRequestID, "wr-update-1")
	}
	if current.RawStatus != "IN_PROGRESS" {
		t.Fatalf("current.rawStatus = %q, want %q", current.RawStatus, "IN_PROGRESS")
	}
	if current.RawOperationType != "UPDATE_QUEUE" {
		t.Fatalf("current.rawOperationType = %q, want %q", current.RawOperationType, "UPDATE_QUEUE")
	}
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("current.normalizedClass = %q, want %q", current.NormalizedClass, shared.OSOKAsyncClassPending)
	}
	if current.PercentComplete == nil || *current.PercentComplete != percent {
		t.Fatalf("current.percentComplete = %v, want %v", current.PercentComplete, percent)
	}
	if current.Message != "OCI update is in progress" {
		t.Fatalf("current.message = %q, want %q", current.Message, "OCI update is in progress")
	}
}

func TestBuildWorkRequestAsyncOperationRejectsUnknownStatus(t *testing.T) {
	t.Parallel()

	_, err := BuildWorkRequestAsyncOperation(&shared.OSOKStatus{}, WorkRequestAsyncAdapter{
		PendingStatusTokens: []string{"ACCEPTED"},
		CreateActionTokens:  []string{"CREATED"},
	}, WorkRequestAsyncInput{
		RawStatus:     "WAITING",
		RawAction:     "CREATED",
		FallbackPhase: shared.OSOKAsyncPhaseCreate,
	})
	if err == nil {
		t.Fatalf("BuildWorkRequestAsyncOperation() error = nil, want unknown status failure")
	}
	if err.Error() != `unmodeled async status "WAITING"` {
		t.Fatalf("BuildWorkRequestAsyncOperation() error = %q, want %q", err.Error(), `unmodeled async status "WAITING"`)
	}
}

func TestBuildWorkRequestAsyncOperationRejectsActionPhaseConflict(t *testing.T) {
	t.Parallel()

	_, err := BuildWorkRequestAsyncOperation(&shared.OSOKStatus{}, WorkRequestAsyncAdapter{
		PendingStatusTokens: []string{"ACCEPTED"},
		CreateActionTokens:  []string{"CREATED"},
		DeleteActionTokens:  []string{"DELETED"},
	}, WorkRequestAsyncInput{
		RawStatus:     "ACCEPTED",
		RawAction:     "DELETED",
		FallbackPhase: shared.OSOKAsyncPhaseCreate,
	})
	if err == nil {
		t.Fatalf("BuildWorkRequestAsyncOperation() error = nil, want phase conflict")
	}
	if err.Error() != `async phase "delete" derived from action "DELETED" conflicts with expected phase "create"` {
		t.Fatalf("BuildWorkRequestAsyncOperation() error = %q, want %q", err.Error(), `async phase "delete" derived from action "DELETED" conflicts with expected phase "create"`)
	}
}

func TestResolveAsyncPhasePrefersExplicitPhase(t *testing.T) {
	t.Parallel()

	status := &shared.OSOKStatus{
		Async: shared.OSOKAsyncTracker{
			Current: &shared.OSOKAsyncOperation{
				Source:          shared.OSOKAsyncSourceLifecycle,
				Phase:           shared.OSOKAsyncPhaseDelete,
				NormalizedClass: shared.OSOKAsyncClassPending,
			},
		},
	}

	if got := ResolveAsyncPhase(status, shared.OSOKAsyncPhaseCreate); got != shared.OSOKAsyncPhaseCreate {
		t.Fatalf("ResolveAsyncPhase() = %q, want %q", got, shared.OSOKAsyncPhaseCreate)
	}
}

func TestResolveAsyncPhaseFallsBackToPersistedCurrent(t *testing.T) {
	t.Parallel()

	status := &shared.OSOKStatus{
		Async: shared.OSOKAsyncTracker{
			Current: &shared.OSOKAsyncOperation{
				Source:          shared.OSOKAsyncSourceLifecycle,
				Phase:           shared.OSOKAsyncPhaseDelete,
				NormalizedClass: shared.OSOKAsyncClassPending,
			},
		},
	}

	if got := ResolveAsyncPhase(status, ""); got != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("ResolveAsyncPhase() = %q, want %q", got, shared.OSOKAsyncPhaseDelete)
	}
}
