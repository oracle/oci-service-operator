package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	devopssdk "github.com/oracle/oci-go-sdk/v65/devops"
	devopsv1beta1 "github.com/oracle/oci-service-operator/api/devops/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestApplyRepositoryRuntimeHooksConfiguresReviewedRuntime(t *testing.T) {
	t.Parallel()

	hooks := newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{})
	applyRepositoryRuntimeHooks(&hooks, nil, nil)

	requireRepositoryRuntimeHooksConfigured(t, hooks)

	resource := newRepositoryTestResource()
	resource.Spec.Description = "updated repository description"
	body, updateNeeded, err := hooks.BuildUpdateBody(
		context.Background(),
		resource,
		resource.Namespace,
		observedRepositoryFromSpec(
			"ocid1.devopsrepository.oc1..existing",
			newRepositoryTestResource().Spec,
			devopssdk.RepositoryLifecycleStateActive,
		),
	)
	if err != nil {
		t.Fatalf("hooks.BuildUpdateBody() error = %v", err)
	}
	requireRepositoryUpdateBodyDescription(t, body, updateNeeded, resource.Spec.Description)
}

func TestRepositoryCreateStartsWorkRequestAndRecordsIdentity(t *testing.T) {
	t.Parallel()

	const createdID = "ocid1.devopsrepository.oc1..created"

	resource := newRepositoryTestResource()
	var createRequest devopssdk.CreateRepositoryRequest

	manager := newRepositoryRuntimeTestManager(generatedruntime.Config[*devopsv1beta1.Repository]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.CreateRepositoryRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				createRequest = *request.(*devopssdk.CreateRepositoryRequest)
				return devopssdk.CreateRepositoryResponse{
					Repository:       observedRepositoryFromSpec(createdID, resource.Spec, devopssdk.RepositoryLifecycleStateCreating),
					OpcRequestId:     common.String("opc-create-1"),
					OpcWorkRequestId: common.String("wr-create-1"),
				}, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Create.Fields,
		},
		Async: generatedruntime.AsyncHooks[*devopsv1beta1.Repository]{
			GetWorkRequest: func(_ context.Context, workRequestID string) (any, error) {
				return repositoryWorkRequest(
					workRequestID,
					devopssdk.OperationTypeCreateRepository,
					devopssdk.OperationStatusInProgress,
					devopssdk.ActionTypeInProgress,
					createdID,
				), nil
			},
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireRepositorySuccessfulRequeue(t, response)
	requireRepositoryCreateRequest(t, createRequest, resource)
	requireRepositoryIdentity(t, resource, createdID)
	requireRepositoryOpcRequestID(t, resource, "opc-create-1")
	requireRepositoryAsyncCurrent(t, resource, shared.OSOKAsyncPhaseCreate, "wr-create-1", "IN_PROGRESS", shared.OSOKAsyncClassPending)
}

func TestRepositoryCreateOrUpdateBindsExistingRepositoryByProjectAndName(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.devopsrepository.oc1..existing"

	resource := newRepositoryTestResource()
	createCalled := false
	listCalls := 0
	getCalls := 0

	manager := newRepositoryRuntimeTestManager(generatedruntime.Config[*devopsv1beta1.Repository]{
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.CreateRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				createCalled = true
				t.Fatal("CreateRepository() should not be called when list resolves an existing repository")
				return nil, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Create.Fields,
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.GetRepositoryRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				getCalls++
				getRequest := request.(*devopssdk.GetRepositoryRequest)
				requireStringPointer(t, getRequest.RepositoryId, existingID, "GetRepositoryRequest.RepositoryId")
				return devopssdk.GetRepositoryResponse{
					Repository: observedRepositoryFromSpec(existingID, resource.Spec, devopssdk.RepositoryLifecycleStateActive),
				}, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Get.Fields,
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.ListRepositoriesRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listCalls++
				listRequest := request.(*devopssdk.ListRepositoriesRequest)
				requireRepositoryListRequest(t, *listRequest, resource)
				return devopssdk.ListRepositoriesResponse{
					RepositoryCollection: devopssdk.RepositoryCollection{
						Items: []devopssdk.RepositorySummary{
							observedRepositorySummaryFromSpec(existingID, resource.Spec, devopssdk.RepositoryLifecycleStateActive),
						},
					},
				}, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).List.Fields,
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireRepositorySuccessfulNoRequeue(t, response)
	if createCalled {
		t.Fatal("CreateRepository() was called unexpectedly")
	}
	requireCallCount(t, "ListRepositories()", listCalls, 1)
	requireCallCount(t, "GetRepository()", getCalls, 1)
	requireRepositoryIdentity(t, resource, existingID)
}

func TestRepositoryCreateOrUpdateSkipsUpdateWhenCurrentMatches(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.devopsrepository.oc1..existing"

	resource := newExistingRepositoryTestResource(existingID)

	manager := newRepositoryRuntimeTestManager(generatedruntime.Config[*devopsv1beta1.Repository]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.GetRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				return devopssdk.GetRepositoryResponse{
					Repository: observedRepositoryFromSpec(existingID, resource.Spec, devopssdk.RepositoryLifecycleStateActive),
				}, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Get.Fields,
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.UpdateRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				t.Fatal("UpdateRepository() should not be called when desired and observed state match")
				return nil, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Update.Fields,
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for a no-op ACTIVE observe")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil for steady no-op observe", resource.Status.OsokStatus.Async.Current)
	}
}

func TestRepositoryCreateOrUpdatePreservesOmittedOptionalStrings(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.devopsrepository.oc1..existing"

	resource := newExistingRepositoryTestResource(existingID)
	resource.Spec.Description = ""
	resource.Spec.DefaultBranch = ""

	current := observedRepositoryFromSpec(existingID, resource.Spec, devopssdk.RepositoryLifecycleStateActive)
	current.Description = common.String("existing description")
	current.DefaultBranch = common.String("main")

	manager := newRepositoryRuntimeTestManager(generatedruntime.Config[*devopsv1beta1.Repository]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.GetRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				return devopssdk.GetRepositoryResponse{Repository: current}, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Get.Fields,
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.UpdateRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				t.Fatal("UpdateRepository() should not be called when optional strings are omitted from the spec")
				return nil, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Update.Fields,
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue for ACTIVE readback with omitted optional strings")
	}
	if resource.Status.Description != "existing description" {
		t.Fatalf("status.description = %q, want existing description", resource.Status.Description)
	}
	if resource.Status.DefaultBranch != "main" {
		t.Fatalf("status.defaultBranch = %q, want main", resource.Status.DefaultBranch)
	}
}

func TestRepositoryCreateOrUpdateUpdatesMutableFieldsAndCompletesWorkRequest(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.devopsrepository.oc1..existing"

	original := newRepositoryTestResource()
	resource := newExistingRepositoryTestResource(existingID)
	resource.Spec.Description = "updated repository description"
	resource.Spec.DefaultBranch = "release"
	resource.Spec.FreeformTags = map[string]string{"env": "prod"}
	resource.Spec.DefinedTags = map[string]shared.MapValue{"Operations": {"CostCenter": "84"}}

	getCalls := 0
	var updateRequest devopssdk.UpdateRepositoryRequest

	manager := newRepositoryRuntimeTestManager(generatedruntime.Config[*devopsv1beta1.Repository]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.GetRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				getCalls++
				spec := original.Spec
				if getCalls > 1 {
					spec = resource.Spec
				}
				return devopssdk.GetRepositoryResponse{
					Repository: observedRepositoryFromSpec(existingID, spec, devopssdk.RepositoryLifecycleStateActive),
				}, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Get.Fields,
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.UpdateRepositoryRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				updateRequest = *request.(*devopssdk.UpdateRepositoryRequest)
				return devopssdk.UpdateRepositoryResponse{
					Repository:       observedRepositoryFromSpec(existingID, resource.Spec, devopssdk.RepositoryLifecycleStateActive),
					OpcRequestId:     common.String("opc-update-1"),
					OpcWorkRequestId: common.String("wr-update-1"),
				}, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Update.Fields,
		},
		Async: generatedruntime.AsyncHooks[*devopsv1beta1.Repository]{
			GetWorkRequest: func(_ context.Context, workRequestID string) (any, error) {
				return repositoryWorkRequest(
					workRequestID,
					devopssdk.OperationTypeUpdateRepository,
					devopssdk.OperationStatusSucceeded,
					devopssdk.ActionTypeUpdated,
					existingID,
				), nil
			},
		},
	})

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	requireRepositorySuccessfulNoRequeue(t, response)
	requireRepositoryUpdateRequest(t, updateRequest, resource, existingID)
	requireRepositoryOpcRequestID(t, resource, "opc-update-1")
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after successful update readback", resource.Status.OsokStatus.Async.Current)
	}
	requireCallCount(t, "GetRepository()", getCalls, 2)
}

func TestRepositoryCreateOrUpdateRejectsCreateOnlyDriftBeforeUpdate(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.devopsrepository.oc1..existing"

	testCases := []struct {
		name      string
		mutate    func(*devopsv1beta1.Repository)
		current   devopssdk.Repository
		wantError string
	}{
		{
			name: "project drift",
			mutate: func(resource *devopsv1beta1.Repository) {
				resource.Spec.ProjectId = "ocid1.devopsproject.oc1..other"
			},
			current:   observedRepositoryFromSpec(existingID, newRepositoryTestResource().Spec, devopssdk.RepositoryLifecycleStateActive),
			wantError: "projectId",
		},
		{
			name: "parent repository removed",
			mutate: func(resource *devopsv1beta1.Repository) {
				resource.Spec.RepositoryType = string(devopssdk.RepositoryRepositoryTypeForked)
				resource.Spec.ParentRepositoryId = ""
			},
			current: func() devopssdk.Repository {
				spec := newRepositoryTestResource().Spec
				spec.RepositoryType = string(devopssdk.RepositoryRepositoryTypeForked)
				spec.ParentRepositoryId = "ocid1.devopsrepository.oc1..parent"
				return observedRepositoryFromSpec(existingID, spec, devopssdk.RepositoryLifecycleStateActive)
			}(),
			wantError: "parentRepositoryId",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource := newExistingRepositoryTestResource(existingID)
			tc.mutate(resource)

			manager := newRepositoryRuntimeTestManager(generatedruntime.Config[*devopsv1beta1.Repository]{
				Get: &generatedruntime.Operation{
					NewRequest: func() any { return &devopssdk.GetRepositoryRequest{} },
					Call: func(context.Context, any) (any, error) {
						return devopssdk.GetRepositoryResponse{Repository: tc.current}, nil
					},
					Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Get.Fields,
				},
				Update: &generatedruntime.Operation{
					NewRequest: func() any { return &devopssdk.UpdateRepositoryRequest{} },
					Call: func(context.Context, any) (any, error) {
						t.Fatal("UpdateRepository() should not be called for create-only drift")
						return nil, nil
					},
					Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Update.Fields,
				},
			})

			_, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("CreateOrUpdate() error = %v, want create-only drift error containing %q", err, tc.wantError)
			}
		})
	}
}

func TestRepositoryDeleteWaitsForWorkRequestConfirmation(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.devopsrepository.oc1..existing"

	resource := newExistingRepositoryTestResource(existingID)
	var deleteRequest devopssdk.DeleteRepositoryRequest

	manager := newRepositoryRuntimeTestManager(generatedruntime.Config[*devopsv1beta1.Repository]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.GetRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				return devopssdk.GetRepositoryResponse{
					Repository: observedRepositoryFromSpec(existingID, resource.Spec, devopssdk.RepositoryLifecycleStateActive),
				}, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Get.Fields,
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.DeleteRepositoryRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				deleteRequest = *request.(*devopssdk.DeleteRepositoryRequest)
				return devopssdk.DeleteRepositoryResponse{
					OpcRequestId:     common.String("opc-delete-1"),
					OpcWorkRequestId: common.String("wr-delete-1"),
				}, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Delete.Fields,
		},
		Async: generatedruntime.AsyncHooks[*devopsv1beta1.Repository]{
			GetWorkRequest: func(_ context.Context, workRequestID string) (any, error) {
				return repositoryWorkRequest(
					workRequestID,
					devopssdk.OperationTypeDeleteRepository,
					devopssdk.OperationStatusInProgress,
					devopssdk.ActionTypeInProgress,
					existingID,
				), nil
			},
		},
	})

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained while delete work request is pending")
	}
	if deleteRequest.RepositoryId == nil || *deleteRequest.RepositoryId != existingID {
		t.Fatalf("DeleteRepositoryRequest.RepositoryId = %v, want %s", deleteRequest.RepositoryId, existingID)
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-1" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-1", resource.Status.OsokStatus.OpcRequestID)
	}
	requireRepositoryAsyncCurrent(t, resource, shared.OSOKAsyncPhaseDelete, "wr-delete-1", "IN_PROGRESS", shared.OSOKAsyncClassPending)
}

func TestRepositoryDeleteCompletesAfterWorkRequestAndReadbackNotFound(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.devopsrepository.oc1..existing"

	resource := newExistingRepositoryTestResource(existingID)
	getCalls := 0

	manager := newRepositoryRuntimeTestManager(generatedruntime.Config[*devopsv1beta1.Repository]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.GetRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				getCalls++
				if getCalls == 1 {
					return devopssdk.GetRepositoryResponse{
						Repository: observedRepositoryFromSpec(existingID, resource.Spec, devopssdk.RepositoryLifecycleStateActive),
					}, nil
				}
				notFound := errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "repository not found")
				notFound.OpcRequestID = "opc-get-after-delete"
				return devopssdk.GetRepositoryResponse{}, notFound
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Get.Fields,
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.DeleteRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				return devopssdk.DeleteRepositoryResponse{
					OpcRequestId:     common.String("opc-delete-1"),
					OpcWorkRequestId: common.String("wr-delete-1"),
				}, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Delete.Fields,
		},
		Async: generatedruntime.AsyncHooks[*devopsv1beta1.Repository]{
			GetWorkRequest: func(_ context.Context, workRequestID string) (any, error) {
				return repositoryWorkRequest(
					workRequestID,
					devopssdk.OperationTypeDeleteRepository,
					devopssdk.OperationStatusSucceeded,
					devopssdk.ActionTypeDeleted,
					existingID,
				), nil
			},
		},
	})

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after work request succeeds and readback is not found")
	}
	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after delete confirmation", resource.Status.OsokStatus.Async.Current)
	}
}

func TestRepositoryDeleteTreatsAuthShapedNotFoundAsAmbiguous(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.devopsrepository.oc1..existing"

	resource := newExistingRepositoryTestResource(existingID)

	manager := newRepositoryRuntimeTestManager(generatedruntime.Config[*devopsv1beta1.Repository]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.GetRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				return devopssdk.GetRepositoryResponse{
					Repository: observedRepositoryFromSpec(existingID, resource.Spec, devopssdk.RepositoryLifecycleStateActive),
				}, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Get.Fields,
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.DeleteRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				err := errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "not authorized or repository not found")
				err.OpcRequestID = "opc-delete-auth"
				return devopssdk.DeleteRepositoryResponse{}, err
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Delete.Fields,
		},
	})

	deleted, err := manager.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped not-found error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained for ambiguous auth-shaped not-found")
	}
	if !strings.Contains(err.Error(), "ambiguous not-found") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt was set for ambiguous auth-shaped not-found")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-delete-auth" {
		t.Fatalf("status.opcRequestId = %q, want opc-delete-auth", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestRepositoryDeleteTreatsPreReadAuthShapedNotFoundAsAmbiguous(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.devopsrepository.oc1..existing"

	resource := newExistingRepositoryTestResource(existingID)
	deleteCalled := false

	manager := newRepositoryRuntimeTestManager(generatedruntime.Config[*devopsv1beta1.Repository]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.GetRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				err := errortest.NewServiceError(404, "NotAuthorizedOrNotFound", "not authorized or repository not found")
				err.OpcRequestID = "opc-get-auth"
				return devopssdk.GetRepositoryResponse{}, err
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Get.Fields,
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.DeleteRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				deleteCalled = true
				t.Fatal("DeleteRepository() should not be called when pre-delete read is auth-shaped not found")
				return devopssdk.DeleteRepositoryResponse{}, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Delete.Fields,
		},
	})

	deleted, err := manager.Delete(context.Background(), resource)
	if err == nil {
		t.Fatal("Delete() error = nil, want ambiguous auth-shaped pre-read error")
	}
	if deleted {
		t.Fatal("Delete() deleted = true, want finalizer retained for ambiguous auth-shaped pre-read")
	}
	if deleteCalled {
		t.Fatal("DeleteRepository() was called unexpectedly")
	}
	if !strings.Contains(err.Error(), "ambiguous not-found") {
		t.Fatalf("Delete() error = %v, want ambiguous not-found classification", err)
	}
	if resource.Status.OsokStatus.DeletedAt != nil {
		t.Fatal("status.deletedAt was set for ambiguous auth-shaped pre-read")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-get-auth" {
		t.Fatalf("status.opcRequestId = %q, want opc-get-auth", resource.Status.OsokStatus.OpcRequestID)
	}
}

func TestRepositoryDeleteWithoutTrackedIDListMissConfirmsDeleted(t *testing.T) {
	t.Parallel()

	resource := newRepositoryTestResource()
	listCalls := 0
	deleteCalled := false

	manager := newRepositoryRuntimeTestManager(generatedruntime.Config[*devopsv1beta1.Repository]{
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.ListRepositoriesRequest{} },
			Call: func(_ context.Context, request any) (any, error) {
				listCalls++
				listRequest := request.(*devopssdk.ListRepositoriesRequest)
				requireRepositoryListRequest(t, *listRequest, resource)
				return devopssdk.ListRepositoriesResponse{
					RepositoryCollection: devopssdk.RepositoryCollection{},
					OpcRequestId:         common.String("opc-list-absent"),
				}, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).List.Fields,
		},
		Delete: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.DeleteRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				deleteCalled = true
				t.Fatal("DeleteRepository() should not be called when list confirms the repository is absent")
				return nil, nil
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Delete.Fields,
		},
	})

	deleted, err := manager.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() deleted = false, want finalizer release after list confirms absence")
	}
	if deleteCalled {
		t.Fatal("DeleteRepository() was called unexpectedly")
	}
	requireCallCount(t, "ListRepositories()", listCalls, 1)
	requireRepositoryDeletedAt(t, resource)
	if resource.Status.OsokStatus.Ocid != "" {
		t.Fatalf("status.ocid = %q, want empty for absent untracked repository", resource.Status.OsokStatus.Ocid)
	}
}

func TestRepositoryCreateOrUpdateRecordsOCIErrorRequestID(t *testing.T) {
	t.Parallel()

	const existingID = "ocid1.devopsrepository.oc1..existing"

	resource := newExistingRepositoryTestResource(existingID)

	manager := newRepositoryRuntimeTestManager(generatedruntime.Config[*devopsv1beta1.Repository]{
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &devopssdk.GetRepositoryRequest{} },
			Call: func(context.Context, any) (any, error) {
				serviceErr := errortest.NewServiceError(500, "InternalError", "devops read failed")
				serviceErr.OpcRequestID = "opc-read-failed"
				return devopssdk.GetRepositoryResponse{}, serviceErr
			},
			Fields: newRepositoryDefaultRuntimeHooks(devopssdk.DevopsClient{}).Get.Fields,
		},
	})

	_, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil {
		t.Fatal("CreateOrUpdate() error = nil, want surfaced OCI read error")
	}
	if resource.Status.OsokStatus.OpcRequestID != "opc-read-failed" {
		t.Fatalf("status.opcRequestId = %q, want opc-read-failed", resource.Status.OsokStatus.OpcRequestID)
	}
	if resource.Status.OsokStatus.Reason != string(shared.Failed) {
		t.Fatalf("status.reason = %q, want Failed", resource.Status.OsokStatus.Reason)
	}
}

func requireRepositoryRuntimeHooksConfigured(t *testing.T, hooks RepositoryRuntimeHooks) {
	t.Helper()

	if hooks.Semantics == nil {
		t.Fatal("hooks.Semantics = nil, want reviewed Repository semantics")
	}
	if hooks.Semantics.Async == nil || hooks.Semantics.Async.Strategy != "workrequest" {
		t.Fatalf("hooks.Semantics.Async = %#v, want generatedruntime workrequest semantics", hooks.Semantics.Async)
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("hooks.BuildUpdateBody = nil, want reviewed mutable update builder")
	}
	if hooks.ParityHooks.ValidateCreateOnlyDrift == nil {
		t.Fatal("hooks.ParityHooks.ValidateCreateOnlyDrift = nil, want create-only drift guard")
	}
	if hooks.Async.GetWorkRequest == nil {
		t.Fatal("hooks.Async.GetWorkRequest = nil, want DevOps work request observer")
	}
}

func requireRepositoryUpdateBodyDescription(t *testing.T, body any, updateNeeded bool, want string) {
	t.Helper()

	if !updateNeeded {
		t.Fatal("hooks.BuildUpdateBody() updateNeeded = false, want true for description drift")
	}
	details, ok := body.(devopssdk.UpdateRepositoryDetails)
	if !ok {
		t.Fatalf("hooks.BuildUpdateBody() body type = %T, want devops.UpdateRepositoryDetails", body)
	}
	requireStringPointer(t, details.Description, want, "UpdateRepositoryDetails.Description")
}

func requireRepositorySuccessfulRequeue(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()

	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if !response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should requeue")
	}
}

func requireRepositorySuccessfulNoRequeue(t *testing.T, response servicemanager.OSOKResponse) {
	t.Helper()

	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should report success")
	}
	if response.ShouldRequeue {
		t.Fatal("CreateOrUpdate() should not requeue")
	}
}

func requireRepositoryCreateRequest(
	t *testing.T,
	request devopssdk.CreateRepositoryRequest,
	resource *devopsv1beta1.Repository,
) {
	t.Helper()

	requireStringPointer(t, request.Name, resource.Spec.Name, "CreateRepositoryDetails.Name")
	requireStringPointer(t, request.ProjectId, resource.Spec.ProjectId, "CreateRepositoryDetails.ProjectId")
	if request.RepositoryType != devopssdk.RepositoryRepositoryTypeHosted {
		t.Fatalf("CreateRepositoryDetails.RepositoryType = %q, want HOSTED", request.RepositoryType)
	}
}

func requireRepositoryUpdateRequest(
	t *testing.T,
	request devopssdk.UpdateRepositoryRequest,
	resource *devopsv1beta1.Repository,
	wantID string,
) {
	t.Helper()

	requireStringPointer(t, request.RepositoryId, wantID, "UpdateRepositoryRequest.RepositoryId")
	requireStringPointer(t, request.Description, resource.Spec.Description, "UpdateRepositoryDetails.Description")
	requireStringPointer(t, request.DefaultBranch, resource.Spec.DefaultBranch, "UpdateRepositoryDetails.DefaultBranch")
	if got := request.FreeformTags["env"]; got != "prod" {
		t.Fatalf("UpdateRepositoryDetails.FreeformTags[env] = %q, want prod", got)
	}
	if got := request.DefinedTags["Operations"]["CostCenter"]; got != "84" {
		t.Fatalf("UpdateRepositoryDetails.DefinedTags Operations/CostCenter = %#v, want 84", got)
	}
}

func requireRepositoryListRequest(
	t *testing.T,
	request devopssdk.ListRepositoriesRequest,
	resource *devopsv1beta1.Repository,
) {
	t.Helper()

	requireStringPointer(t, request.ProjectId, resource.Spec.ProjectId, "ListRepositoriesRequest.ProjectId")
	requireStringPointer(t, request.Name, resource.Spec.Name, "ListRepositoriesRequest.Name")
}

func requireRepositoryIdentity(t *testing.T, resource *devopsv1beta1.Repository, wantID string) {
	t.Helper()

	if got := string(resource.Status.OsokStatus.Ocid); got != wantID {
		t.Fatalf("status.ocid = %q, want %q", got, wantID)
	}
	if resource.Status.Id != wantID {
		t.Fatalf("status.id = %q, want %q", resource.Status.Id, wantID)
	}
}

func requireRepositoryOpcRequestID(t *testing.T, resource *devopsv1beta1.Repository, want string) {
	t.Helper()

	if resource.Status.OsokStatus.OpcRequestID != want {
		t.Fatalf("status.opcRequestId = %q, want %s", resource.Status.OsokStatus.OpcRequestID, want)
	}
}

func requireRepositoryDeletedAt(t *testing.T, resource *devopsv1beta1.Repository) {
	t.Helper()

	if resource.Status.OsokStatus.DeletedAt == nil {
		t.Fatal("status.deletedAt = nil, want delete confirmation timestamp")
	}
}

func requireCallCount(t *testing.T, operation string, got int, want int) {
	t.Helper()

	if got != want {
		t.Fatalf("%s calls = %d, want %d", operation, got, want)
	}
}

func requireStringPointer(t *testing.T, got *string, want string, field string) {
	t.Helper()

	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", field, got, want)
	}
}

func newRepositoryRuntimeTestManager(
	cfg generatedruntime.Config[*devopsv1beta1.Repository],
) *RepositoryServiceManager {
	applyRepositoryRuntimeTestDefaults(&cfg)
	getRepository := repositoryRuntimeTestGetOperation(cfg.Get)
	listRepositories := repositoryRuntimeTestListOperation(cfg.List)
	if cfg.DeleteHooks.ConfirmRead == nil && (getRepository != nil || listRepositories != nil) {
		cfg.DeleteHooks.ConfirmRead = repositoryDeleteConfirmRead(getRepository, listRepositories)
	}
	if cfg.DeleteHooks.ApplyOutcome == nil {
		cfg.DeleteHooks.ApplyOutcome = applyRepositoryDeleteOutcome
	}

	client := defaultRepositoryServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*devopsv1beta1.Repository](cfg),
	}
	var wrappedClient RepositoryServiceClient = client
	if listRepositories != nil {
		wrappedClient = wrapRepositoryDeleteWithoutTrackedID(listRepositories)(wrappedClient)
	}

	return &RepositoryServiceManager{
		client: wrappedClient,
	}
}

func applyRepositoryRuntimeTestDefaults(cfg *generatedruntime.Config[*devopsv1beta1.Repository]) {
	if cfg.Kind == "" {
		cfg.Kind = "Repository"
	}
	if cfg.SDKName == "" {
		cfg.SDKName = "Repository"
	}
	if cfg.Semantics == nil {
		cfg.Semantics = newRepositoryRuntimeSemantics()
	}
	applyRepositoryRuntimeTestAsyncDefaults(cfg)
	applyRepositoryRuntimeTestMutationDefaults(cfg)
	applyRepositoryRuntimeTestDeleteDefaults(cfg)
}

func applyRepositoryRuntimeTestAsyncDefaults(cfg *generatedruntime.Config[*devopsv1beta1.Repository]) {
	if cfg.Async.GetWorkRequest == nil {
		cfg.Async.GetWorkRequest = func(_ context.Context, workRequestID string) (any, error) {
			return repositoryWorkRequest(
				workRequestID,
				devopssdk.OperationTypeUpdateRepository,
				devopssdk.OperationStatusSucceeded,
				devopssdk.ActionTypeUpdated,
				"ocid1.devopsrepository.oc1..unused",
			), nil
		}
	}
	cfg.Async.Adapter = repositoryWorkRequestAsyncAdapter
	if cfg.Async.ResolveAction == nil {
		cfg.Async.ResolveAction = resolveRepositoryWorkRequestAction
	}
	if cfg.Async.ResolvePhase == nil {
		cfg.Async.ResolvePhase = resolveRepositoryWorkRequestPhase
	}
	if cfg.Async.RecoverResourceID == nil {
		cfg.Async.RecoverResourceID = recoverRepositoryIDFromWorkRequest
	}
	if cfg.Async.Message == nil {
		cfg.Async.Message = repositoryWorkRequestMessage
	}
}

func applyRepositoryRuntimeTestMutationDefaults(cfg *generatedruntime.Config[*devopsv1beta1.Repository]) {
	if cfg.BuildUpdateBody == nil {
		cfg.BuildUpdateBody = func(
			_ context.Context,
			resource *devopsv1beta1.Repository,
			_ string,
			currentResponse any,
		) (any, bool, error) {
			return buildRepositoryUpdateBody(resource, currentResponse)
		}
	}
	if cfg.ParityHooks.ValidateCreateOnlyDrift == nil {
		cfg.ParityHooks.ValidateCreateOnlyDrift = validateRepositoryCreateOnlyDrift
	}
}

func applyRepositoryRuntimeTestDeleteDefaults(cfg *generatedruntime.Config[*devopsv1beta1.Repository]) {
	if cfg.DeleteHooks.HandleError == nil {
		cfg.DeleteHooks.HandleError = handleRepositoryDeleteError
	}
}

func repositoryRuntimeTestGetOperation(
	operation *generatedruntime.Operation,
) func(context.Context, devopssdk.GetRepositoryRequest) (devopssdk.GetRepositoryResponse, error) {
	if operation == nil {
		return nil
	}
	return func(ctx context.Context, request devopssdk.GetRepositoryRequest) (devopssdk.GetRepositoryResponse, error) {
		response, err := operation.Call(ctx, &request)
		if err != nil {
			return devopssdk.GetRepositoryResponse{}, err
		}
		return repositoryRuntimeTestGetResponse(response), nil
	}
}

func repositoryRuntimeTestGetResponse(response any) devopssdk.GetRepositoryResponse {
	switch typed := response.(type) {
	case devopssdk.GetRepositoryResponse:
		return typed
	case *devopssdk.GetRepositoryResponse:
		if typed == nil {
			return devopssdk.GetRepositoryResponse{}
		}
		return *typed
	default:
		return devopssdk.GetRepositoryResponse{}
	}
}

func repositoryRuntimeTestListOperation(
	operation *generatedruntime.Operation,
) func(context.Context, devopssdk.ListRepositoriesRequest) (devopssdk.ListRepositoriesResponse, error) {
	if operation == nil {
		return nil
	}
	return func(ctx context.Context, request devopssdk.ListRepositoriesRequest) (devopssdk.ListRepositoriesResponse, error) {
		response, err := operation.Call(ctx, &request)
		if err != nil {
			return devopssdk.ListRepositoriesResponse{}, err
		}
		return repositoryRuntimeTestListResponse(response), nil
	}
}

func repositoryRuntimeTestListResponse(response any) devopssdk.ListRepositoriesResponse {
	switch typed := response.(type) {
	case devopssdk.ListRepositoriesResponse:
		return typed
	case *devopssdk.ListRepositoriesResponse:
		if typed == nil {
			return devopssdk.ListRepositoriesResponse{}
		}
		return *typed
	default:
		return devopssdk.ListRepositoriesResponse{}
	}
}

func newRepositoryTestResource() *devopsv1beta1.Repository {
	return &devopsv1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repository-sample",
			Namespace: "default",
			UID:       types.UID("repository-sample-uid"),
		},
		Spec: devopsv1beta1.RepositorySpec{
			Name:           "repository-sample",
			ProjectId:      "ocid1.devopsproject.oc1..example",
			RepositoryType: string(devopssdk.RepositoryRepositoryTypeHosted),
			DefaultBranch:  "main",
			Description:    "repository description",
			FreeformTags: map[string]string{
				"env": "test",
			},
			DefinedTags: map[string]shared.MapValue{
				"Operations": {
					"CostCenter": "42",
				},
			},
		},
	}
}

func newExistingRepositoryTestResource(existingID string) *devopsv1beta1.Repository {
	resource := newRepositoryTestResource()
	resource.Status.Id = existingID
	resource.Status.ProjectId = resource.Spec.ProjectId
	resource.Status.Name = resource.Spec.Name
	resource.Status.RepositoryType = resource.Spec.RepositoryType
	resource.Status.DefaultBranch = resource.Spec.DefaultBranch
	resource.Status.Description = resource.Spec.Description
	resource.Status.OsokStatus.Ocid = shared.OCID(existingID)
	return resource
}

func observedRepositoryFromSpec(
	id string,
	spec devopsv1beta1.RepositorySpec,
	lifecycleState devopssdk.RepositoryLifecycleStateEnum,
) devopssdk.Repository {
	repository := devopssdk.Repository{
		Id:                 common.String(id),
		CompartmentId:      common.String("ocid1.compartment.oc1..example"),
		ProjectId:          common.String(spec.ProjectId),
		Name:               common.String(spec.Name),
		ParentRepositoryId: repositoryOptionalString(spec.ParentRepositoryId),
		Description:        repositoryOptionalString(spec.Description),
		DefaultBranch:      repositoryOptionalString(spec.DefaultBranch),
		RepositoryType:     devopssdk.RepositoryRepositoryTypeEnum(spec.RepositoryType),
		LifecycleState:     lifecycleState,
		FreeformTags:       repositoryCloneStringMap(spec.FreeformTags),
		DefinedTags:        repositoryDefinedTagsFromSpec(spec.DefinedTags),
	}
	if mirrorConfig, ok := repositoryMirrorConfigFromSpec(spec.MirrorRepositoryConfig); ok {
		repository.MirrorRepositoryConfig = mirrorConfig
	}
	return repository
}

func observedRepositorySummaryFromSpec(
	id string,
	spec devopsv1beta1.RepositorySpec,
	lifecycleState devopssdk.RepositoryLifecycleStateEnum,
) devopssdk.RepositorySummary {
	repository := observedRepositoryFromSpec(id, spec, lifecycleState)
	return devopssdk.RepositorySummary{
		Id:                     repository.Id,
		CompartmentId:          repository.CompartmentId,
		ProjectId:              repository.ProjectId,
		Name:                   repository.Name,
		ParentRepositoryId:     repository.ParentRepositoryId,
		Description:            repository.Description,
		DefaultBranch:          repository.DefaultBranch,
		RepositoryType:         repository.RepositoryType,
		MirrorRepositoryConfig: repository.MirrorRepositoryConfig,
		LifecycleState:         repository.LifecycleState,
		FreeformTags:           repository.FreeformTags,
		DefinedTags:            repository.DefinedTags,
	}
}

func repositoryWorkRequest(
	id string,
	operationType devopssdk.OperationTypeEnum,
	status devopssdk.OperationStatusEnum,
	actionType devopssdk.ActionTypeEnum,
	resourceID string,
) devopssdk.WorkRequest {
	return devopssdk.WorkRequest{
		Id:              common.String(id),
		OperationType:   operationType,
		Status:          status,
		CompartmentId:   common.String("ocid1.compartment.oc1..example"),
		PercentComplete: common.Float32(50),
		Resources: []devopssdk.WorkRequestResource{
			{
				EntityType: common.String("repository"),
				ActionType: actionType,
				Identifier: common.String(resourceID),
			},
		},
	}
}

func repositoryOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func requireRepositoryAsyncCurrent(
	t *testing.T,
	resource *devopsv1beta1.Repository,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	rawStatus string,
	class shared.OSOKAsyncNormalizedClass,
) {
	t.Helper()

	current := resource.Status.OsokStatus.Async.Current
	if current == nil {
		t.Fatalf("status.async.current = nil, want phase %s work request %s", phase, workRequestID)
	}
	if current.Phase != phase {
		t.Fatalf("status.async.current.phase = %q, want %q", current.Phase, phase)
	}
	if current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", current.WorkRequestID, workRequestID)
	}
	if current.RawStatus != rawStatus {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", current.RawStatus, rawStatus)
	}
	if current.NormalizedClass != class {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", current.NormalizedClass, class)
	}
}
