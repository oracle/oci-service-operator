/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

func TestApplySuccessSetsLifecycleAsyncTrackerWhilePending(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Lifecycle: LifecycleSemantics{UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}}}})
	resource := &fakeResource{}
	response, err := client.applySuccess(resource, fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..pending", DisplayName: "pending-thing", LifecycleState: "UPDATING"}}, shared.Updating)
	if err != nil {
		t.Fatalf("applySuccess() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatalf("response.ShouldRequeue = false, want true")
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatalf("status.async.current = nil, want lifecycle tracker")
	}
	if resource.Status.OsokStatus.Async.Current.Source != shared.OSOKAsyncSourceLifecycle {
		t.Fatalf("status.async.current.source = %q, want %q", resource.Status.OsokStatus.Async.Current.Source, shared.OSOKAsyncSourceLifecycle)
	}
	if resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseUpdate {
		t.Fatalf("status.async.current.phase = %q, want %q", resource.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseUpdate)
	}
	if resource.Status.OsokStatus.Async.Current.RawStatus != "UPDATING" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, "UPDATING")
	}
	if resource.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", resource.Status.OsokStatus.Async.Current.NormalizedClass, shared.OSOKAsyncClassPending)
	}
}

func TestServiceClientCreateOrUpdateCapturesOpcRequestIDFromOCIError(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Create: &Operation{NewRequest: func() any {
		return &fakeCreateThingRequest{}
	}, Call: func(_ context.Context, _ any) (any, error) {
		return nil, errortest.NewServiceError(409, "IncorrectState", "update conflict")
	}}})
	resource := &fakeResource{Spec: fakeSpec{CompartmentId: "ocid1.compartment.oc1..example", DisplayName: "desired-name"}}
	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want OCI failure")
	}
	if response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() response = %#v, want unsuccessful response", response)
	}
	requireStatusOpcRequestID(t, resource, "opc-request-id")
	requireTrailingCondition(t, resource, shared.Failed)
}

func TestApplySuccessClearsLifecycleAsyncTrackerWhenActive(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Lifecycle: LifecycleSemantics{UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}}}})
	resource := &fakeResource{Status: fakeStatus{OsokStatus: shared.OSOKStatus{Async: shared.OSOKAsyncTracker{Current: &shared.OSOKAsyncOperation{Source: shared.OSOKAsyncSourceLifecycle, Phase: shared.OSOKAsyncPhaseUpdate, RawStatus: "UPDATING", NormalizedClass: shared.OSOKAsyncClassPending}}}}}
	response, err := client.applySuccess(resource, fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..active", DisplayName: "active-thing", LifecycleState: "ACTIVE"}}, shared.Active)
	if err != nil {
		t.Fatalf("applySuccess() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = false, want true")
	}
	if response.ShouldRequeue {
		t.Fatalf("response.ShouldRequeue = true, want false")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker", resource.Status.OsokStatus.Async.Current)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Active) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Active)
	}
}

func TestApplySuccessPrefersObservedLifecyclePhaseTransition(t *testing.T) {
	t.Parallel()
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Lifecycle: LifecycleSemantics{UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}}, Delete: DeleteSemantics{PendingStates: []string{"DELETING"}}}})
	resource := &fakeResource{Status: fakeStatus{OsokStatus: shared.OSOKStatus{Async: shared.OSOKAsyncTracker{Current: &shared.OSOKAsyncOperation{Source: shared.OSOKAsyncSourceLifecycle, Phase: shared.OSOKAsyncPhaseUpdate, RawStatus: "UPDATING", NormalizedClass: shared.OSOKAsyncClassPending}}}}}
	response, err := client.applySuccess(resource, fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..deleting", DisplayName: "deleting-thing", LifecycleState: "DELETING"}}, shared.Updating)
	if err != nil {
		t.Fatalf("applySuccess() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("response.IsSuccessful = false, want true")
	}
	if !response.ShouldRequeue {
		t.Fatalf("response.ShouldRequeue = false, want true")
	}
	if resource.Status.OsokStatus.Async.Current == nil {
		t.Fatalf("status.async.current = nil, want lifecycle tracker")
	}
	if resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current.phase = %q, want %q", resource.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseDelete)
	}
	if resource.Status.OsokStatus.Async.Current.RawStatus != "DELETING" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, "DELETING")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
	}
}

func TestApplySuccessHeuristicDeleteTerminalKeepsAsyncTrackerWithoutSemantics(t *testing.T) {
	t.Parallel()
	for _, lifecycleState := range []string{"DELETED", "TERMINATED"} {
		lifecycleState := lifecycleState
		t.Run(lifecycleState, func(t *testing.T) {
			t.Parallel()
			client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing"})
			resource := &fakeResource{Status: fakeStatus{OsokStatus: shared.OSOKStatus{Async: shared.OSOKAsyncTracker{Current: &shared.OSOKAsyncOperation{Source: shared.OSOKAsyncSourceLifecycle, Phase: shared.OSOKAsyncPhaseUpdate, RawStatus: "UPDATING", NormalizedClass: shared.OSOKAsyncClassPending}}}}}
			response, err := client.applySuccess(resource, fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..deleted", DisplayName: "deleted-thing", LifecycleState: lifecycleState}}, shared.Active)
			if err != nil {
				t.Fatalf("applySuccess() error = %v", err)
			}
			if !response.IsSuccessful {
				t.Fatalf("response.IsSuccessful = false, want true")
			}
			if !response.ShouldRequeue {
				t.Fatalf("response.ShouldRequeue = false, want true")
			}
			if resource.Status.OsokStatus.Async.Current == nil {
				t.Fatalf("status.async.current = nil, want delete tracker")
			}
			if resource.Status.OsokStatus.Async.Current.Source != shared.OSOKAsyncSourceLifecycle {
				t.Fatalf("status.async.current.source = %q, want %q", resource.Status.OsokStatus.Async.Current.Source, shared.OSOKAsyncSourceLifecycle)
			}
			if resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
				t.Fatalf("status.async.current.phase = %q, want %q", resource.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseDelete)
			}
			if resource.Status.OsokStatus.Async.Current.RawStatus != lifecycleState {
				t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, lifecycleState)
			}
			if resource.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassSucceeded {
				t.Fatalf("status.async.current.normalizedClass = %q, want %q", resource.Status.OsokStatus.Async.Current.NormalizedClass, shared.OSOKAsyncClassSucceeded)
			}
			if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
				t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
			}
		})
	}
}

func TestApplySuccessFormalDeleteTerminalKeepsAsyncTrackerWithSemantics(t *testing.T) {
	t.Parallel()
	for _, lifecycleState := range []string{"DELETED", "TERMINATED"} {
		lifecycleState := lifecycleState
		t.Run(lifecycleState, func(t *testing.T) {
			t.Parallel()
			client := NewServiceClient[*fakeResource](Config[*fakeResource]{Kind: "Thing", SDKName: "Thing", Semantics: &Semantics{Lifecycle: LifecycleSemantics{UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}}, Delete: DeleteSemantics{TerminalStates: []string{"DELETED", "TERMINATED"}}}})
			resource := &fakeResource{Status: fakeStatus{OsokStatus: shared.OSOKStatus{Async: shared.OSOKAsyncTracker{Current: &shared.OSOKAsyncOperation{Source: shared.OSOKAsyncSourceLifecycle, Phase: shared.OSOKAsyncPhaseUpdate, RawStatus: "UPDATING", NormalizedClass: shared.OSOKAsyncClassPending}}}}}
			response, err := client.applySuccess(resource, fakeGetThingResponse{Thing: fakeThing{Id: "ocid1.thing.oc1..deleted", DisplayName: "deleted-thing", LifecycleState: lifecycleState}}, shared.Active)
			if err != nil {
				t.Fatalf("applySuccess() error = %v", err)
			}
			if !response.IsSuccessful {
				t.Fatalf("response.IsSuccessful = false, want true")
			}
			if !response.ShouldRequeue {
				t.Fatalf("response.ShouldRequeue = false, want true")
			}
			if resource.Status.OsokStatus.Async.Current == nil {
				t.Fatalf("status.async.current = nil, want delete tracker")
			}
			if resource.Status.OsokStatus.Async.Current.Source != shared.OSOKAsyncSourceLifecycle {
				t.Fatalf("status.async.current.source = %q, want %q", resource.Status.OsokStatus.Async.Current.Source, shared.OSOKAsyncSourceLifecycle)
			}
			if resource.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
				t.Fatalf("status.async.current.phase = %q, want %q", resource.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseDelete)
			}
			if resource.Status.OsokStatus.Async.Current.RawStatus != lifecycleState {
				t.Fatalf("status.async.current.rawStatus = %q, want %q", resource.Status.OsokStatus.Async.Current.RawStatus, lifecycleState)
			}
			if resource.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassSucceeded {
				t.Fatalf("status.async.current.normalizedClass = %q, want %q", resource.Status.OsokStatus.Async.Current.NormalizedClass, shared.OSOKAsyncClassSucceeded)
			}
			if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
				t.Fatalf("status.reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
			}
		})
	}
}

func TestResponseWorkRequestIDReadsOCIHeader(t *testing.T) {
	t.Parallel()
	if got := responseWorkRequestID(fakeCreateThingResponse{OpcWorkRequestId: stringPtr("wr-create-1")}); got != "wr-create-1" {
		t.Fatalf("responseWorkRequestID(create) = %q, want %q", got, "wr-create-1")
	}
	if got := responseWorkRequestID(fakeDeleteThingResponse{}); got != "" {
		t.Fatalf("responseWorkRequestID(delete) = %q, want empty string", got)
	}
}

func TestResponseRequestIDReadsOCIHeader(t *testing.T) {
	t.Parallel()
	if got := responseRequestID(fakeCreateThingResponse{OpcRequestId: stringPtr("opc-create-1")}); got != "opc-create-1" {
		t.Fatalf("responseRequestID(create) = %q, want %q", got, "opc-create-1")
	}
	if got := responseRequestID(fakeDeleteThingResponse{}); got != "" {
		t.Fatalf("responseRequestID(delete) = %q, want empty string", got)
	}
}
