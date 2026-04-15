package generatedruntime

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestServiceClientCreateFollowUpReadAfterWriteUsesMatrix(t *testing.T) {
	t.Parallel()

	registration := errortest.ReviewedRegistrationForFamily(
		t,
		"opensearch",
		"OpensearchCluster",
		errortest.APIErrorCoverageFamilyGeneratedRuntimeFollowUp,
	)
	if !strings.Contains(registration.Deviation, "read-after-write") {
		t.Fatalf("reviewed registration = %s, want explicit read-after-write note", errortest.DescribeReviewedRegistration(registration))
	}

	focused := errortest.FocusedAsyncFollowUpCases(t)
	cases := []errortest.AsyncFollowUpMatrixCase{
		{Candidate: focused["notfound"], WantSuccessful: true},
		{Candidate: focused["auth404"], WantErrorType: focused["auth404"].NormalizedType},
		{Candidate: focused["conflict"], WantErrorType: focused["conflict"].NormalizedType},
		{Candidate: focused["internal"], WantErrorType: focused["internal"].NormalizedType},
		{Candidate: focused["unavailable"], WantErrorType: focused["unavailable"].NormalizedType},
	}

	errortest.RunAsyncFollowUpMatrix(t, cases, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.AsyncFollowUpResult {
		client := NewServiceClient[*fakeResource](Config[*fakeResource]{
			Kind:    "OpensearchCluster",
			SDKName: "OpensearchCluster",
			Semantics: &Semantics{
				Lifecycle: LifecycleSemantics{
					ActiveStates: []string{"ACTIVE"},
				},
				Delete: DeleteSemantics{
					Policy:         "required",
					PendingStates:  []string{"DELETING"},
					TerminalStates: []string{"DELETED"},
				},
				List: &ListSemantics{
					ResponseItemsField: "Items",
					MatchFields:        []string{"displayName"},
				},
				CreateFollowUp: FollowUpSemantics{
					Strategy: "read-after-write",
					Hooks:    []Hook{{Helper: "tfresource.CreateResource"}},
				},
				DeleteFollowUp: FollowUpSemantics{
					Strategy: "confirm-delete",
					Hooks:    []Hook{{Helper: "tfresource.DeleteResource"}},
				},
			},
			Create: &Operation{
				NewRequest: func() any { return &fakeCreateThingRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
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
					return fakeListThingResponse{
						Collection: fakeThingCollection{Items: nil},
					}, nil
				},
				Fields: []RequestField{
					{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
				},
			},
		})

		resource := &fakeResource{
			Spec: fakeSpec{
				DisplayName: "created-name",
			},
		}

		response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		if err == nil {
			requireStatusOCID(t, resource, "ocid1.thing.oc1..created")
			requireStringEqual(t, "status.displayName", resource.Status.DisplayName, "created-name")
			requireTrailingCondition(t, resource, shared.Active)
		}
		return errortest.AsyncFollowUpResult{
			Err:        err,
			Successful: response.IsSuccessful,
			Requeue:    response.ShouldRequeue,
		}
	})
}

func TestServiceClientCreateFollowUpPreservesSeededWorkRequestID(t *testing.T) {
	t.Parallel()

	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "OpensearchCluster",
		SDKName: "OpensearchCluster",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				ProvisioningStates: []string{"CREATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			CreateFollowUp: FollowUpSemantics{
				Strategy: "read-after-write",
				Hooks:    []Hook{{Helper: "tfresource.CreateResource"}},
			},
		},
		Create: &Operation{
			NewRequest: func() any { return &fakeCreateThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return fakeCreateThingResponse{
					OpcWorkRequestId: stringPtr("wr-create-followup"),
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..created",
						DisplayName:    "created-name",
						LifecycleState: "CREATING",
					},
				}, nil
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.thing.oc1..created",
						DisplayName:    "created-name",
						LifecycleState: "CREATING",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			DisplayName: "created-name",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireRequeueState(t, response, true, "CreateOrUpdate() should keep requeueing while create follow-up stays CREATING")
	requireCurrentWorkRequestID(t, resource, "wr-create-followup")
	requireCurrentAsyncSource(t, resource, shared.OSOKAsyncSourceLifecycle)
	requireCurrentAsyncPhase(t, resource, shared.OSOKAsyncPhaseCreate)
	requireTrailingCondition(t, resource, shared.Provisioning)
}

func TestServiceClientCreateWaitForWorkRequestFollowUpUsesMatrix(t *testing.T) {
	t.Parallel()

	focused := errortest.FocusedAsyncFollowUpCases(t)
	cases := []errortest.AsyncFollowUpMatrixCase{
		{Candidate: focused["notfound"], WantSuccessful: true},
		{Candidate: focused["auth404"], WantErrorType: focused["auth404"].NormalizedType},
		{Candidate: focused["conflict"], WantErrorType: focused["conflict"].NormalizedType},
		{Candidate: focused["internal"], WantErrorType: focused["internal"].NormalizedType},
		{Candidate: focused["unavailable"], WantErrorType: focused["unavailable"].NormalizedType},
	}

	errortest.RunAsyncFollowUpMatrix(t, cases, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.AsyncFollowUpResult {
		// Queue-style work-request semantics are routed through handwritten runtimes,
		// so construct the client directly to keep the helper-labeled follow-up path covered.
		client := ServiceClient[*fakeResource]{config: Config[*fakeResource]{
			Kind:    "QueueLikeThing",
			SDKName: "QueueLikeThing",
			Semantics: &Semantics{
				Async: &AsyncSemantics{
					Strategy:             "workrequest",
					Runtime:              "handwritten",
					FormalClassification: "workrequest",
					WorkRequest: &WorkRequestSemantics{
						Source: "service-sdk",
						Phases: []string{"create"},
					},
				},
				Lifecycle: LifecycleSemantics{
					ActiveStates: []string{"ACTIVE"},
				},
				Delete: DeleteSemantics{
					Policy:         "required",
					PendingStates:  []string{"DELETING"},
					TerminalStates: []string{"DELETED"},
				},
				List: &ListSemantics{
					ResponseItemsField: "Items",
					MatchFields:        []string{"displayName"},
				},
				Hooks: HookSet{
					Create: []Hook{
						{Helper: "tfresource.CreateResource"},
						{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "queue", Action: "CREATED"},
					},
				},
				CreateFollowUp: FollowUpSemantics{
					Strategy: "read-after-write",
					Hooks: []Hook{
						{Helper: "tfresource.CreateResource"},
						{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "queue", Action: "CREATED"},
					},
				},
				DeleteFollowUp: FollowUpSemantics{
					Strategy: "confirm-delete",
					Hooks:    []Hook{{Helper: "tfresource.DeleteResource"}},
				},
			},
			Create: &Operation{
				NewRequest: func() any { return &fakeCreateThingRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
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
					return fakeListThingResponse{
						Collection: fakeThingCollection{Items: nil},
					}, nil
				},
				Fields: []RequestField{
					{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
				},
			},
		}}

		resource := &fakeResource{
			Spec: fakeSpec{
				DisplayName: "created-name",
			},
		}

		response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		if err == nil {
			requireStatusOCID(t, resource, "ocid1.thing.oc1..created")
			requireStringEqual(t, "status.displayName", resource.Status.DisplayName, "created-name")
			requireTrailingCondition(t, resource, shared.Active)
		}
		return errortest.AsyncFollowUpResult{
			Err:        err,
			Successful: response.IsSuccessful,
			Requeue:    response.ShouldRequeue,
		}
	})
}

func TestServiceClientUpdateWaitForUpdatedStateFollowUpUsesMatrix(t *testing.T) {
	t.Parallel()

	registration := errortest.ReviewedRegistrationForFamily(
		t,
		"streaming",
		"Stream",
		errortest.APIErrorCoverageFamilyGeneratedRuntimeFollowUp,
	)
	if !strings.Contains(registration.Deviation, "WaitForUpdatedState") {
		t.Fatalf("reviewed registration = %s, want explicit WaitForUpdatedState note", errortest.DescribeReviewedRegistration(registration))
	}

	focused := errortest.FocusedAsyncFollowUpCases(t)
	cases := []errortest.AsyncFollowUpMatrixCase{
		{Candidate: focused["notfound"], WantErrorSubstring: errResourceNotFound.Error()},
		{Candidate: focused["auth404"], WantErrorType: focused["auth404"].NormalizedType},
		{Candidate: focused["conflict"], WantErrorType: focused["conflict"].NormalizedType},
		{Candidate: focused["internal"], WantErrorType: focused["internal"].NormalizedType},
		{Candidate: focused["unavailable"], WantErrorType: focused["unavailable"].NormalizedType},
	}

	errortest.RunAsyncFollowUpMatrix(t, cases, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.AsyncFollowUpResult {
		getCalls := 0
		client := NewServiceClient[*fakeResource](Config[*fakeResource]{
			Kind:    "Stream",
			SDKName: "Stream",
			Semantics: &Semantics{
				Lifecycle: LifecycleSemantics{
					ActiveStates: []string{"ACTIVE"},
				},
				Delete: DeleteSemantics{
					Policy:         "required",
					PendingStates:  []string{"DELETING"},
					TerminalStates: []string{"DELETED"},
				},
				List: &ListSemantics{
					ResponseItemsField: "Items",
					MatchFields:        []string{"displayName"},
				},
				Mutation: MutationSemantics{
					Mutable: []string{"displayName"},
				},
				UpdateFollowUp: FollowUpSemantics{
					Strategy: "read-after-write",
					Hooks: []Hook{
						{Helper: "tfresource.UpdateResource"},
						{Helper: "tfresource.WaitForUpdatedState"},
					},
				},
				DeleteFollowUp: FollowUpSemantics{
					Strategy: "confirm-delete",
					Hooks:    []Hook{{Helper: "tfresource.DeleteResource"}},
				},
			},
			Get: &Operation{
				NewRequest: func() any { return &fakeGetThingRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					getCalls++
					if getCalls == 1 {
						return fakeGetThingResponse{
							Thing: fakeThing{
								Id:             "ocid1.stream.oc1..existing",
								DisplayName:    "old-name",
								LifecycleState: "ACTIVE",
							},
						}, nil
					}
					return nil, errortest.NewServiceErrorFromCase(candidate)
				},
				Fields: []RequestField{
					{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
				},
			},
			List: &Operation{
				NewRequest: func() any { return &fakeListThingRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					return fakeListThingResponse{
						Collection: fakeThingCollection{Items: nil},
					}, nil
				},
				Fields: []RequestField{
					{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
				},
			},
			Update: &Operation{
				NewRequest: func() any { return &fakeUpdateThingRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					return fakeUpdateThingResponse{
						Thing: fakeThing{
							Id:             "ocid1.stream.oc1..existing",
							DisplayName:    "new-name",
							LifecycleState: "UPDATING",
						},
					}, nil
				},
				Fields: []RequestField{
					{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
					{FieldName: "FakeUpdateThingDetails", RequestName: "FakeUpdateThingDetails", Contribution: "body"},
				},
			},
		})

		resource := &fakeResource{
			Spec: fakeSpec{
				DisplayName: "new-name",
			},
			Status: fakeStatus{
				OsokStatus: shared.OSOKStatus{Ocid: "ocid1.stream.oc1..existing"},
				Id:         "ocid1.stream.oc1..existing",
			},
		}

		response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
		return errortest.AsyncFollowUpResult{
			Err:        err,
			Successful: response.IsSuccessful,
			Requeue:    response.ShouldRequeue,
		}
	})
}

func TestServiceClientUpdateFollowUpPreservesSeededWorkRequestID(t *testing.T) {
	t.Parallel()

	getCalls := 0
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "Stream",
		SDKName: "Stream",
		Semantics: &Semantics{
			Lifecycle: LifecycleSemantics{
				UpdatingStates: []string{"UPDATING"},
				ActiveStates:   []string{"ACTIVE"},
			},
			Mutation: MutationSemantics{
				Mutable: []string{"displayName"},
			},
			UpdateFollowUp: FollowUpSemantics{
				Strategy: "read-after-write",
				Hooks: []Hook{
					{Helper: "tfresource.UpdateResource"},
					{Helper: "tfresource.WaitForUpdatedState"},
				},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				getCalls++
				displayName := "old-name"
				lifecycleState := "ACTIVE"
				if getCalls > 1 {
					displayName = "new-name"
					lifecycleState = "UPDATING"
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.stream.oc1..existing",
						DisplayName:    displayName,
						LifecycleState: lifecycleState,
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
				return fakeUpdateThingResponse{
					OpcWorkRequestId: stringPtr("wr-update-followup"),
					Thing: fakeThing{
						Id:             "ocid1.stream.oc1..existing",
						DisplayName:    "new-name",
						LifecycleState: "UPDATING",
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
				{FieldName: "FakeUpdateThingDetails", RequestName: "FakeUpdateThingDetails", Contribution: "body"},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			DisplayName: "new-name",
		},
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: "ocid1.stream.oc1..existing"},
			Id:         "ocid1.stream.oc1..existing",
		},
	}

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err)
	requireRequeueState(t, response, true, "CreateOrUpdate() should keep requeueing while update follow-up stays UPDATING")
	requireCurrentWorkRequestID(t, resource, "wr-update-followup")
	requireCurrentAsyncSource(t, resource, shared.OSOKAsyncSourceLifecycle)
	requireCurrentAsyncPhase(t, resource, shared.OSOKAsyncPhaseUpdate)
	requireTrailingCondition(t, resource, shared.Updating)
}

func TestServiceClientDeleteConfirmDeleteReadUsesMatrix(t *testing.T) {
	t.Parallel()

	registration := errortest.ReviewedRegistrationForFamily(
		t,
		"opensearch",
		"OpensearchCluster",
		errortest.APIErrorCoverageFamilyGeneratedRuntimeFollowUp,
	)
	if !strings.Contains(registration.Deviation, "confirm-delete") {
		t.Fatalf("reviewed registration = %s, want explicit confirm-delete note", errortest.DescribeReviewedRegistration(registration))
	}

	focused := errortest.FocusedAsyncFollowUpCases(t)
	cases := []errortest.AsyncFollowUpMatrixCase{
		{Candidate: focused["notfound"], WantDeleted: true},
		{Candidate: focused["auth404"], WantDeleted: true},
		{Candidate: focused["conflict"], WantErrorType: focused["conflict"].NormalizedType},
		{Candidate: focused["internal"], WantErrorType: focused["internal"].NormalizedType},
		{Candidate: focused["unavailable"], WantErrorType: focused["unavailable"].NormalizedType},
	}

	errortest.RunAsyncFollowUpMatrix(t, cases, func(t *testing.T, candidate errortest.CommonErrorCase) errortest.AsyncFollowUpResult {
		getCalls := 0
		client := NewServiceClient[*fakeResource](Config[*fakeResource]{
			Kind:    "OpensearchCluster",
			SDKName: "OpensearchCluster",
			Semantics: &Semantics{
				Delete: DeleteSemantics{
					Policy:         "required",
					PendingStates:  []string{"DELETING"},
					TerminalStates: []string{"DELETED"},
				},
				List: &ListSemantics{
					ResponseItemsField: "Items",
					MatchFields:        []string{"displayName"},
				},
				DeleteFollowUp: FollowUpSemantics{
					Strategy: "confirm-delete",
					Hooks:    []Hook{{Helper: "tfresource.DeleteResource"}},
				},
			},
			Delete: &Operation{
				NewRequest: func() any { return &fakeDeleteThingRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					return fakeDeleteThingResponse{}, nil
				},
				Fields: []RequestField{
					{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
				},
			},
			Get: &Operation{
				NewRequest: func() any { return &fakeGetThingRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					getCalls++
					if getCalls == 1 {
						return fakeGetThingResponse{
							Thing: fakeThing{
								Id:             "ocid1.opensearchcluster.oc1..existing",
								DisplayName:    "cluster-sample",
								LifecycleState: "ACTIVE",
							},
						}, nil
					}
					return nil, errortest.NewServiceErrorFromCase(candidate)
				},
				Fields: []RequestField{
					{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
				},
			},
			List: &Operation{
				NewRequest: func() any { return &fakeListThingRequest{} },
				Call: func(_ context.Context, _ any) (any, error) {
					return fakeListThingResponse{
						Collection: fakeThingCollection{Items: nil},
					}, nil
				},
				Fields: []RequestField{
					{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
				},
			},
		})

		resource := &fakeResource{
			Spec: fakeSpec{
				DisplayName: "cluster-sample",
			},
			Status: fakeStatus{
				OsokStatus: shared.OSOKStatus{Ocid: "ocid1.opensearchcluster.oc1..existing"},
				Id:         "ocid1.opensearchcluster.oc1..existing",
			},
		}

		deleted, err := client.Delete(context.Background(), resource)
		if err == nil && deleted && resource.Status.OsokStatus.DeletedAt == nil {
			t.Fatal("status.deletedAt should be set after confirmed delete")
		}
		return errortest.AsyncFollowUpResult{
			Err:     err,
			Deleted: deleted,
		}
	})
}

func TestServiceClientDeleteFollowUpPreservesSeededWorkRequestID(t *testing.T) {
	t.Parallel()

	getCalls := 0
	client := NewServiceClient[*fakeResource](Config[*fakeResource]{
		Kind:    "OpensearchCluster",
		SDKName: "OpensearchCluster",
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
		Delete: &Operation{
			NewRequest: func() any { return &fakeDeleteThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				return fakeDeleteThingResponse{
					OpcWorkRequestId: stringPtr("wr-delete-followup"),
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
		Get: &Operation{
			NewRequest: func() any { return &fakeGetThingRequest{} },
			Call: func(_ context.Context, _ any) (any, error) {
				getCalls++
				lifecycleState := "ACTIVE"
				if getCalls > 1 {
					lifecycleState = "DELETING"
				}
				return fakeGetThingResponse{
					Thing: fakeThing{
						Id:             "ocid1.opensearchcluster.oc1..existing",
						DisplayName:    "cluster-sample",
						LifecycleState: lifecycleState,
					},
				}, nil
			},
			Fields: []RequestField{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})

	resource := &fakeResource{
		Spec: fakeSpec{
			DisplayName: "cluster-sample",
		},
		Status: fakeStatus{
			OsokStatus: shared.OSOKStatus{Ocid: "ocid1.opensearchcluster.oc1..existing"},
			Id:         "ocid1.opensearchcluster.oc1..existing",
		},
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() should keep waiting while confirm-delete follow-up stays DELETING")
	}
	requireCurrentWorkRequestID(t, resource, "wr-delete-followup")
	requireCurrentAsyncSource(t, resource, shared.OSOKAsyncSourceLifecycle)
	requireCurrentAsyncPhase(t, resource, shared.OSOKAsyncPhaseDelete)
	requireTrailingCondition(t, resource, shared.Terminating)
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt should stay empty while delete follow-up remains pending")
	}
}
