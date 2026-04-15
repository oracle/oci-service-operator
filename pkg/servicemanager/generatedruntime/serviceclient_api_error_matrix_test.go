package generatedruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const generatedRuntimeMatrixCurrentID = "ocid1.thing.oc1..matrix"

func TestServiceClientGeneratedRuntimePlainCreateErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainCreateMatrix(t, func(_ *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainMutationResult {
		resource := &fakeResource{
			Spec: fakeSpec{
				CompartmentId: "ocid1.compartment.oc1..matrix",
				DisplayName:   "matrix-create",
			},
		}

		response, err := newGeneratedRuntimePlainCreateErrorClient(candidate).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		return errortest.GeneratedRuntimePlainMutationResult{
			Response:     response,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

func TestServiceClientGeneratedRuntimePlainReadErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainReadMatrix(t, func(_ *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainReadResult {
		resource := &fakeResource{
			Spec: fakeSpec{
				CompartmentId: "ocid1.compartment.oc1..matrix",
				Name:          "matrix-read",
			},
			Status: fakeStatus{
				OsokStatus: shared.OSOKStatus{Ocid: generatedRuntimeMatrixCurrentID},
			},
		}

		_, err := newGeneratedRuntimePlainReadErrorClient(candidate).readResource(context.Background(), resource, generatedRuntimeMatrixCurrentID, readPhaseObserve)
		if errors.Is(err, errResourceNotFound) {
			return errortest.GeneratedRuntimePlainReadResult{Missing: true}
		}
		return errortest.GeneratedRuntimePlainReadResult{
			Err:     err,
			Missing: false,
		}
	})
}

func TestServiceClientGeneratedRuntimePlainUpdateErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainUpdateMatrix(t, func(_ *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainMutationResult {
		resource := &fakeResource{
			Spec: fakeSpec{
				DisplayName: "matrix-update",
			},
			Status: fakeStatus{
				OsokStatus: shared.OSOKStatus{Ocid: generatedRuntimeMatrixCurrentID},
			},
		}

		response, err := newGeneratedRuntimePlainUpdateErrorClient(candidate).CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		return errortest.GeneratedRuntimePlainMutationResult{
			Response:     response,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

func TestServiceClientGeneratedRuntimePlainDeleteErrorMatrix(t *testing.T) {
	t.Parallel()

	errortest.RunGeneratedRuntimePlainDeleteMatrix(t, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.GeneratedRuntimePlainDeleteResult {
		resource := &fakeResource{
			Status: fakeStatus{
				OsokStatus: shared.OSOKStatus{Ocid: generatedRuntimeMatrixCurrentID},
			},
		}

		deleted, err := newGeneratedRuntimePlainDeleteErrorClient(t, candidate).Delete(context.Background(), resource)
		return errortest.GeneratedRuntimePlainDeleteResult{
			Deleted:      deleted,
			Err:          err,
			StatusReason: resource.Status.OsokStatus.Reason,
		}
	})
}

func TestServiceClientGeneratedRuntimePlainDeleteRetryableConflictMayResolveDeletedAfterConfirmRead(t *testing.T) {
	t.Parallel()

	candidate, ok := errortest.LookupCommonErrorCase(409, "IncorrectState")
	if !ok {
		t.Fatal("LookupCommonErrorCase(409, IncorrectState) = missing, want matrix row")
	}

	resource := &fakeResource{
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: generatedRuntimeMatrixCurrentID},
		},
	}

	deleted, err := newGeneratedRuntimePlainDeleteErrorClientWithConfirmRead(t, candidate, func(getCalls int) (any, error) {
		if getCalls == 1 {
			return fakeGetThingResponse{
				Thing: fakeThing{
					Id:             generatedRuntimeMatrixCurrentID,
					LifecycleState: "ACTIVE",
				},
			}, nil
		}
		return nil, errortest.NewServiceError(404, "NotFound", "delete confirm reread reports disappearance")
	}).Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v, want deleted outcome after confirm reread reports NotFound", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want true after confirm reread reports NotFound")
	}
	if resource.Status.OsokStatus.Reason != string(shared.Terminating) {
		t.Fatalf("status reason = %q, want %q", resource.Status.OsokStatus.Reason, shared.Terminating)
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want timestamp after confirm reread reports NotFound")
	}
}

func newGeneratedRuntimePlainCreateErrorClient(candidate errortest.CommonErrorCase) ServiceClient[*fakeResource] {
	return NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{ActiveStates: []string{"ACTIVE"}},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return nil, errortest.NewServiceErrorFromCase(candidate)
			},
			Fields: []RequestField{
				{FieldName: "FakeCreateThingDetails", Contribution: "body"},
			},
		},
	})
}

func newGeneratedRuntimePlainReadErrorClient(candidate errortest.CommonErrorCase) ServiceClient[*fakeResource] {
	return NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			List: &ListSemantics{
				ResponseItemsField: "Items",
				MatchFields:        []string{"name", "compartmentId"},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return nil, errortest.NewServiceErrorFromCase(candidate)
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &Operation{
			NewRequest: func() any { return &fakeListThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return fakeListThingResponse{Collection: fakeThingCollection{}}, nil
			},
			Fields: []RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "Name", RequestName: "name", Contribution: "query"},
			},
		},
	})
}

func newGeneratedRuntimePlainUpdateErrorClient(candidate errortest.CommonErrorCase) ServiceClient[*fakeResource] {
	return NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{ActiveStates: []string{"ACTIVE"}},
		},
		BuildUpdateBody: func(context.Context, *fakeResource, string, any) (any, bool, error) {
			return FakeUpdateThingDetails{DisplayName: "matrix-update"}, true, nil
		},
		Update: &Operation{
			NewRequest: func() any { return &fakeUpdateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return nil, errortest.NewServiceErrorFromCase(candidate)
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
				{FieldName: "FakeUpdateThingDetails", Contribution: "body"},
			},
		},
	})
}

func newGeneratedRuntimePlainDeleteErrorClient(t *testing.T, candidate errortest.CommonErrorCase) ServiceClient[*fakeResource] {
	t.Helper()

	return newGeneratedRuntimePlainDeleteErrorClientWithConfirmRead(t, candidate, nil)
}

func newGeneratedRuntimePlainDeleteErrorClientWithConfirmRead(
	t *testing.T,
	candidate errortest.CommonErrorCase,
	confirmRead func(getCalls int) (any, error),
) ServiceClient[*fakeResource] {
	t.Helper()

	getCalls := 0

	return NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Thing",
		SDKName: "Thing",
		Semantics: &Semantics{
			Delete: DeleteSemantics{
				Policy:         "required",
				PendingStates:  []string{"TERMINATING"},
				TerminalStates: []string{"TERMINATED"},
			},
			DeleteFollowUp: FollowUpSemantics{
				Strategy: "confirm-delete",
			},
		},
		Delete: &Operation{
			NewRequest: func() any { return &fakeDeleteThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return nil, errortest.NewServiceErrorFromCase(candidate)
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				getCalls++
				if confirmRead != nil {
					return confirmRead(getCalls)
				}
				lifecycleState := "ACTIVE"
				if getCalls > 1 && errortest.GeneratedRuntimePlainDeleteRequiresConfirmRead(candidate) {
					lifecycleState = "TERMINATING"
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             generatedRuntimeMatrixCurrentID,
						LifecycleState: lifecycleState,
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})
}
