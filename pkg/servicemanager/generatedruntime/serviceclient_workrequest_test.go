package generatedruntime

import (
	"context"
	"fmt"
	"testing"

	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeWorkRequest struct {
	Id              string
	Status          string
	OperationType   string
	Action          string
	ResourceID      string
	PercentComplete *float32
}

func newFakeWorkRequestConfig(workRequests map[string]fakeWorkRequest) Config[*fakeResource] {
	return Config[*fakeResource]{
		Kind:    "Queue",
		SDKName: "Queue",
		Semantics: &Semantics{
			Async: &AsyncSemantics{
				Strategy:             "workrequest",
				Runtime:              "generatedruntime",
				FormalClassification: "workrequest",
				WorkRequest: &WorkRequestSemantics{
					Source: "service-sdk",
					Phases: []string{"create", "update", "delete"},
					LegacyFieldBridge: &WorkRequestLegacyFieldBridge{
						Create: "CreateWorkRequestId",
						Update: "UpdateWorkRequestId",
						Delete: "DeleteWorkRequestId",
					},
				},
			},
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				UpdatingStates:     []string{"UPDATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Delete: DeleteSemantics{
				Policy:         "required",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
		},
		Async: AsyncHooks[*fakeResource]{
			Adapter: servicemanager.WorkRequestAsyncAdapter{
				PendingStatusTokens:   []string{"IN_PROGRESS"},
				SucceededStatusTokens: []string{"SUCCEEDED"},
				FailedStatusTokens:    []string{"FAILED"},
				CreateActionTokens:    []string{"CREATED"},
				UpdateActionTokens:    []string{"UPDATED"},
				DeleteActionTokens:    []string{"DELETED"},
			},
			GetWorkRequest: func(_ context.Context, workRequestID string) (any, error) {
				workRequest, ok := workRequests[workRequestID]
				if !ok {
					return nil, fmt.Errorf("missing fake work request %s", workRequestID)
				}
				return workRequest, nil
			},
			ResolveAction: func(workRequest any) (string, error) {
				return workRequest.(fakeWorkRequest).Action, nil
			},
			RecoverResourceID: func(_ *fakeResource, workRequest any, _ shared.OSOKAsyncPhase) (string, error) {
				return workRequest.(fakeWorkRequest).ResourceID, nil
			},
		},
	}
}

func TestServiceClientCreateStartsGeneratedWorkRequestTracking(t *testing.T) {
	t.Parallel()

	workRequests := map[string]fakeWorkRequest{
		"wr-create-1": {
			Id:         "wr-create-1",
			Status:     "IN_PROGRESS",
			Action:     "CREATED",
			ResourceID: "ocid1.thing.oc1..created",
		},
	}
	config := newFakeWorkRequestConfig(workRequests)
	config.Create = &Operation{
		NewRequest: func() any { return &fakeCreateThingRequest{} },
		Call: func(_ context.Context, _ any) (any, error) {
			return fakeCreateThingResponse{
				OpcWorkRequestId: stringPtr("wr-create-1"),
				Thing: fakeThing{
					Id:             "ocid1.thing.oc1..created",
					DisplayName:    "created-name",
					LifecycleState: "CREATING",
				},
			}, nil
		},
	}
	config.Get = &Operation{
		NewRequest: func() any { return &fakeGetThingRequest{} },
		Call: func(_ context.Context, _ any) (any, error) {
			t.Fatal("Get() should not run while the work request is still pending")
			return nil, nil
		},
		Fields: []RequestField{
			{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
		},
	}

	client := NewServiceClient[*fakeResource](config)
	resource := &fakeResource{
		Spec: fakeSpec{
			DisplayName: "created-name",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireRequeueState(t, response, true, "CreateOrUpdate() should requeue while the create work request stays pending")
	requireCurrentAsyncSource(t, resource, shared.OSOKAsyncSourceWorkRequest)
	requireCurrentAsyncPhase(t, resource, shared.OSOKAsyncPhaseCreate)
	requireCurrentWorkRequestID(t, resource, "wr-create-1")
	if got := resource.Status.CreateWorkRequestId; got != "wr-create-1" {
		t.Fatalf("status.createWorkRequestId = %q, want %q", got, "wr-create-1")
	}
	if resource.Status.UpdateWorkRequestId != "" || resource.Status.DeleteWorkRequestId != "" {
		t.Fatalf("legacy bridge fields = %#v, want only create tracking", resource.Status)
	}
}

func TestServiceClientCreateResumesGeneratedWorkRequestFromLegacyBridge(t *testing.T) {
	t.Parallel()

	workRequests := map[string]fakeWorkRequest{
		"wr-create-legacy": {
			Id:         "wr-create-legacy",
			Status:     "SUCCEEDED",
			Action:     "CREATED",
			ResourceID: "ocid1.thing.oc1..created",
		},
	}
	config := newFakeWorkRequestConfig(workRequests)
	config.Get = &Operation{
		NewRequest: func() any { return &fakeGetThingRequest{} },
		Call: func(_ context.Context, request any) (any, error) {
			if got := request.(*fakeGetThingRequest).ThingId; got == nil || *got != "ocid1.thing.oc1..created" {
				t.Fatalf("Get() thingId = %v, want recovered created resource id", got)
			}
			return fakeGetThingResponse{
				Thing: fakeThing{
					Id:             "ocid1.thing.oc1..created",
					DisplayName:    "created-name",
					LifecycleState: "ACTIVE",
				},
			}, nil
		},
		Fields: []RequestField{
			{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
		},
	}

	client := NewServiceClient[*fakeResource](config)
	resource := &fakeResource{
		Spec: fakeSpec{
			DisplayName: "created-name",
		},
		Status: fakeStatus{
			CreateWorkRequestId: "wr-create-legacy",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireRequeueState(t, response, false, "CreateOrUpdate() should converge once the recovered resource is ACTIVE")
	requireStatusOCID(t, resource, "ocid1.thing.oc1..created")
	requireTrailingCondition(t, resource, shared.Active)
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker after convergence", resource.Status.OsokStatus.Async.Current)
	}
	if resource.Status.CreateWorkRequestId != "" || resource.Status.UpdateWorkRequestId != "" || resource.Status.DeleteWorkRequestId != "" {
		t.Fatalf("legacy bridge fields = %#v, want cleared values after convergence", resource.Status)
	}
}

func TestServiceClientDeleteResumesGeneratedWorkRequestAndMarksDeleted(t *testing.T) {
	t.Parallel()

	workRequests := map[string]fakeWorkRequest{
		"wr-delete-1": {
			Id:     "wr-delete-1",
			Status: "SUCCEEDED",
			Action: "DELETED",
		},
	}
	config := newFakeWorkRequestConfig(workRequests)
	config.Get = &Operation{
		NewRequest: func() any { return &fakeGetThingRequest{} },
		Call: func(_ context.Context, _ any) (any, error) {
			return nil, errResourceNotFound
		},
		Fields: []RequestField{
			{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
		},
	}

	client := NewServiceClient[*fakeResource](config)
	resource := &fakeResource{
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{
				Ocid: shared.OCID("ocid1.thing.oc1..delete"),
				Async: shared.OSOKAsyncTracker{
					Current: &shared.OSOKAsyncOperation{
						Source:          shared.OSOKAsyncSourceWorkRequest,
						Phase:           shared.OSOKAsyncPhaseDelete,
						WorkRequestID:   "wr-delete-1",
						NormalizedClass: shared.OSOKAsyncClassPending,
					},
				},
			},
			DeleteWorkRequestId: "wr-delete-1",
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() = false, want deleted after the delete work request succeeds and the resource is gone")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want cleared tracker after delete", resource.Status.OsokStatus.Async.Current)
	}
	if resource.Status.DeleteWorkRequestId != "" || resource.Status.CreateWorkRequestId != "" || resource.Status.UpdateWorkRequestId != "" {
		t.Fatalf("legacy bridge fields = %#v, want cleared delete tracking", resource.Status)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete timestamp")
	}
}
