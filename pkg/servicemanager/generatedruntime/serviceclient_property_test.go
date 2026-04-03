/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type lifecycleQuickCase struct {
	ProvisioningState   string
	UpdatingState       string
	ActiveState         string
	DeletePendingState  string
	DeleteTerminalState string
	UnknownState        string
	FallbackBucket      uint8
	InputBucket         uint8
	LowercaseInput      bool
}

func (lifecycleQuickCase) Generate(r *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(lifecycleQuickCase{
		ProvisioningState:   quickStateToken(r, "PROVISIONING"),
		UpdatingState:       quickStateToken(r, "UPDATING"),
		ActiveState:         quickStateToken(r, "ACTIVE"),
		DeletePendingState:  quickStateToken(r, "DELETING"),
		DeleteTerminalState: quickStateToken(r, "DELETED"),
		UnknownState:        quickStateToken(r, "UNKNOWN"),
		FallbackBucket:      uint8(r.Intn(5)),
		InputBucket:         uint8(r.Intn(7)),
		LowercaseInput:      r.Intn(2) == 0,
	})
}

func (c lifecycleQuickCase) fallback() shared.OSOKConditionType {
	switch c.FallbackBucket % 5 {
	case 0:
		return shared.Provisioning
	case 1:
		return shared.Updating
	case 2:
		return shared.Terminating
	case 3:
		return shared.Failed
	default:
		return shared.Active
	}
}

func (c lifecycleQuickCase) inputState() string {
	var state string
	switch c.InputBucket % 7 {
	case 1:
		state = c.ProvisioningState
	case 2:
		state = c.UpdatingState
	case 3:
		state = c.DeletePendingState
	case 4:
		state = c.ActiveState
	case 5:
		state = c.DeleteTerminalState
	case 6:
		state = c.UnknownState
	}
	if c.LowercaseInput {
		return strings.ToLower(state)
	}
	return state
}

func (c lifecycleQuickCase) expectedClassification() (shared.OSOKConditionType, bool) {
	switch c.InputBucket % 7 {
	case 0:
		fallback := c.fallback()
		return fallback, shouldRequeueForCondition(fallback)
	case 1:
		return shared.Provisioning, true
	case 2:
		return shared.Updating, true
	case 3:
		return shared.Terminating, true
	case 4, 5:
		return shared.Active, false
	default:
		return shared.Failed, false
	}
}

type deleteQuickCase struct {
	PendingState    string
	TerminalState   string
	UnexpectedState string
	PolicyBucket    uint8
	InputBucket     uint8
	LowercaseInput  bool
}

func (deleteQuickCase) Generate(r *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(deleteQuickCase{
		PendingState:    quickStateToken(r, "DELETING"),
		TerminalState:   quickStateToken(r, "DELETED"),
		UnexpectedState: quickStateToken(r, "ACTIVE"),
		PolicyBucket:    uint8(r.Intn(2)),
		InputBucket:     uint8(r.Intn(4)),
		LowercaseInput:  r.Intn(2) == 0,
	})
}

func (c deleteQuickCase) policy() string {
	if c.PolicyBucket%2 == 0 {
		return "best-effort"
	}
	return "required"
}

func (c deleteQuickCase) inputState() string {
	var state string
	switch c.InputBucket % 4 {
	case 1:
		state = c.PendingState
	case 2:
		state = c.TerminalState
	case 3:
		state = c.UnexpectedState
	}
	if c.LowercaseInput {
		return strings.ToLower(state)
	}
	return state
}

func (c deleteQuickCase) expectDeleted() bool {
	switch c.policy() {
	case "best-effort":
		return c.InputBucket%4 != 3
	default:
		return c.InputBucket%4 == 2
	}
}

func (c deleteQuickCase) expectError() bool {
	return c.policy() == "required" && c.InputBucket%4 == 3
}

type deleteQuickOutcome struct {
	deleted    bool
	err        error
	inputState string
	request    fakeDeleteThingRequest
	resource   *fakeResource
}

func TestClassifyLifecycleSemanticsQuick(t *testing.T) {
	t.Parallel()

	property := func(tc lifecycleQuickCase) bool {
		t.Helper()

		semantics := &Semantics{
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{tc.ProvisioningState},
				UpdatingStates:     []string{tc.UpdatingState},
				ActiveStates:       []string{tc.ActiveState},
			},
			Delete: DeleteSemantics{
				PendingStates:  []string{tc.DeletePendingState},
				TerminalStates: []string{tc.DeleteTerminalState},
			},
		}
		response := fakeGetThingResponse{
			Thing: fakeThing{
				LifecycleState: tc.inputState(),
				DisplayName:    "generated-state",
			},
		}

		gotCondition, gotRequeue, gotMessage := classifyLifecycleSemantics(response, tc.fallback(), semantics)
		wantCondition, wantRequeue := tc.expectedClassification()
		if gotCondition != wantCondition || gotRequeue != wantRequeue {
			t.Logf("classifyLifecycleSemantics(%q) = (%s, %t), want (%s, %t)", response.Thing.LifecycleState, gotCondition, gotRequeue, wantCondition, wantRequeue)
			return false
		}
		if gotMessage == "" {
			t.Logf("classifyLifecycleSemantics(%q) returned an empty message", response.Thing.LifecycleState)
			return false
		}
		return true
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 256}); err != nil {
		t.Fatalf("quick.Check() error = %v", err)
	}
}

func TestServiceClientDeleteWithFormalSemanticsQuick(t *testing.T) {
	t.Parallel()

	property := func(tc deleteQuickCase) bool {
		t.Helper()
		return checkDeleteQuickCase(t, tc)
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 256}); err != nil {
		t.Fatalf("quick.Check() error = %v", err)
	}
}

func checkDeleteQuickCase(t *testing.T, tc deleteQuickCase) bool {
	t.Helper()

	outcome := runDeleteQuickCase(tc)
	if !checkDeleteQuickRequest(t, tc, outcome.request) {
		return false
	}
	return checkDeleteQuickOutcome(t, tc, outcome)
}

func runDeleteQuickCase(tc deleteQuickCase) deleteQuickOutcome {
	inputState := tc.inputState()
	var deleteRequest fakeDeleteThingRequest

	resource := &fakeResource{
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: "ocid1.thing.oc1..delete"},
		},
	}

	deleted, err := newDeleteQuickClient(tc, inputState, &deleteRequest).Delete(context.Background(), resource)
	return deleteQuickOutcome{
		deleted:    deleted,
		err:        err,
		inputState: inputState,
		request:    deleteRequest,
		resource:   resource,
	}
}

func newDeleteQuickClient(tc deleteQuickCase, inputState string, deleteRequest *fakeDeleteThingRequest) ServiceClient[*fakeResource] {
	return NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Delete: DeleteSemantics{
				Policy:         tc.policy(),
				PendingStates:  []string{tc.PendingState},
				TerminalStates: []string{tc.TerminalState},
			},
			DeleteFollowUp: FollowUpSemantics{
				Strategy: "confirm-delete",
			},
		},
		Delete: &Operation{
			NewRequest: func() any { return &fakeDeleteThingRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				*deleteRequest = *request.(*fakeDeleteThingRequest)
				return fakeDeleteThingResponse{}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..delete",
						LifecycleState: inputState,
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})
}

func checkDeleteQuickRequest(t *testing.T, tc deleteQuickCase, request fakeDeleteThingRequest) bool {
	t.Helper()

	if tc.InputBucket%4 == 1 || tc.InputBucket%4 == 2 {
		if request.ThingId != nil {
			t.Logf("delete request thingId = %v, want no delete request once lifecycle is already pending or terminal", request.ThingId)
			return false
		}
		return true
	}

	if request.ThingId == nil || *request.ThingId != "ocid1.thing.oc1..delete" {
		t.Logf("delete request thingId = %v, want ocid1.thing.oc1..delete", request.ThingId)
		return false
	}
	return true
}

func checkDeleteQuickOutcome(t *testing.T, tc deleteQuickCase, outcome deleteQuickOutcome) bool {
	t.Helper()

	if tc.expectError() {
		return checkDeleteQuickError(t, outcome)
	}
	return checkDeleteQuickSuccess(t, tc, outcome)
}

func checkDeleteQuickError(t *testing.T, outcome deleteQuickOutcome) bool {
	t.Helper()

	if outcome.err == nil || !strings.Contains(outcome.err.Error(), "unexpected lifecycle state") {
		t.Logf("Delete() error = %v, want unexpected lifecycle state failure for %q", outcome.err, outcome.inputState)
		return false
	}
	if outcome.deleted {
		t.Logf("Delete() unexpectedly reported deletion for error case %q", outcome.inputState)
		return false
	}
	if outcome.resource.Status.OsokStatus.DeletedAt != nil {
		t.Logf("Delete() set deletedAt for error case %q", outcome.inputState)
		return false
	}
	if len(outcome.resource.Status.OsokStatus.Conditions) != 0 {
		t.Logf("Delete() recorded conditions for error case %q: %#v", outcome.inputState, outcome.resource.Status.OsokStatus.Conditions)
		return false
	}
	return true
}

func checkDeleteQuickSuccess(t *testing.T, tc deleteQuickCase, outcome deleteQuickOutcome) bool {
	t.Helper()

	if outcome.err != nil {
		t.Logf("Delete() error = %v for policy=%s state=%q", outcome.err, tc.policy(), outcome.inputState)
		return false
	}
	if outcome.deleted != tc.expectDeleted() {
		t.Logf("Delete() deleted = %t, want %t for policy=%s state=%q", outcome.deleted, tc.expectDeleted(), tc.policy(), outcome.inputState)
		return false
	}
	if outcome.resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Logf("status.reason = %q, want %q", outcome.resource.Status.OsokStatus.Reason, shared.Terminating)
		return false
	}
	if !hasTrailingCondition(outcome.resource.Status.OsokStatus.Conditions, shared.Terminating) {
		t.Logf("status conditions = %#v, want trailing Terminating condition", outcome.resource.Status.OsokStatus.Conditions)
		return false
	}
	return checkDeleteQuickDeletedAt(t, tc, outcome)
}

func hasTrailingCondition(conditions []shared.OSOKCondition, want shared.OSOKConditionType) bool {
	if len(conditions) == 0 {
		return false
	}
	return conditions[len(conditions)-1].Type == want
}

func checkDeleteQuickDeletedAt(t *testing.T, tc deleteQuickCase, outcome deleteQuickOutcome) bool {
	t.Helper()

	if tc.expectDeleted() {
		if outcome.resource.Status.OsokStatus.DeletedAt == nil {
			t.Logf("Delete() did not set deletedAt for policy=%s state=%q", tc.policy(), outcome.inputState)
			return false
		}
		return true
	}
	if outcome.resource.Status.OsokStatus.DeletedAt != nil {
		t.Logf("Delete() set deletedAt unexpectedly for policy=%s state=%q", tc.policy(), outcome.inputState)
		return false
	}
	return true
}

func quickStateToken(r *rand.Rand, prefix string) string {
	return fmt.Sprintf("%s_%08X", prefix, r.Uint32())
}
