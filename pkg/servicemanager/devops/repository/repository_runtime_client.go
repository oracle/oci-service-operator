/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	devopssdk "github.com/oracle/oci-go-sdk/v65/devops"
	devopsv1beta1 "github.com/oracle/oci-service-operator/api/devops/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const repositoryAmbiguousNotFoundErrorCode = "RepositoryAmbiguousNotFound"

var repositoryWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(devopssdk.OperationStatusAccepted),
		string(devopssdk.OperationStatusInProgress),
		string(devopssdk.OperationStatusWaiting),
		string(devopssdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(devopssdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(devopssdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(devopssdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(devopssdk.OperationStatusNeedsAttention)},
	CreateActionTokens: []string{
		string(devopssdk.OperationTypeCreateRepository),
		string(devopssdk.OperationTypeForkRepository),
		string(devopssdk.OperationTypeMirrorRepository),
	},
	UpdateActionTokens: []string{
		string(devopssdk.OperationTypeUpdateRepository),
		string(devopssdk.OperationTypeSyncForkRepository),
	},
	DeleteActionTokens: []string{string(devopssdk.OperationTypeDeleteRepository)},
}

type repositoryWorkRequestClient interface {
	GetWorkRequest(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error)
}

func init() {
	registerRepositoryRuntimeHooksMutator(func(manager *RepositoryServiceManager, hooks *RepositoryRuntimeHooks) {
		workRequestClient, initErr := newRepositoryWorkRequestClient(manager)
		applyRepositoryRuntimeHooks(hooks, workRequestClient, initErr)
	})
}

func newRepositoryWorkRequestClient(manager *RepositoryServiceManager) (repositoryWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("repository service manager is nil")
	}
	client, err := devopssdk.NewDevopsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyRepositoryRuntimeHooks(
	hooks *RepositoryRuntimeHooks,
	workRequestClient repositoryWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newRepositoryRuntimeSemantics()
	hooks.Async.Adapter = repositoryWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getRepositoryWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveRepositoryWorkRequestAction
	hooks.Async.ResolvePhase = resolveRepositoryWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverRepositoryIDFromWorkRequest
	hooks.Async.Message = repositoryWorkRequestMessage
	hooks.DeleteHooks.ConfirmRead = repositoryDeleteConfirmRead(hooks.Get.Call, hooks.List.Call)
	hooks.DeleteHooks.HandleError = handleRepositoryDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyRepositoryDeleteOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapRepositoryDeleteWithoutTrackedID(hooks.List.Call))
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *devopsv1beta1.Repository,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildRepositoryUpdateBody(resource, currentResponse)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateRepositoryCreateOnlyDrift
}

func newRepositoryRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "devops",
		FormalSlug:    "repository",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(devopssdk.RepositoryLifecycleStateCreating)},
			ActiveStates:       []string{string(devopssdk.RepositoryLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(devopssdk.RepositoryLifecycleStateDeleting)},
			TerminalStates: []string{string(devopssdk.RepositoryLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"projectId", "name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"name",
				"description",
				"defaultBranch",
				"repositoryType",
				"mirrorRepositoryConfig",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"projectId", "parentRepositoryId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

type repositoryAmbiguousDeleteConfirmResponse struct {
	Repository devopssdk.Repository `presentIn:"body"`
	err        error
}

type repositoryDeleteWithoutTrackedIDClient struct {
	RepositoryServiceClient
	listRepositories func(context.Context, devopssdk.ListRepositoriesRequest) (devopssdk.ListRepositoriesResponse, error)
}

type repositoryRuntimeBodyConverter func(any) (devopssdk.Repository, bool, error)

var repositoryRuntimeBodyConverters = []repositoryRuntimeBodyConverter{
	repositoryRuntimeBodyFromRepository,
	repositoryRuntimeBodyFromSummary,
	repositoryRuntimeBodyFromCreateResponse,
	repositoryRuntimeBodyFromGetResponse,
	repositoryRuntimeBodyFromUpdateResponse,
}

func wrapRepositoryDeleteWithoutTrackedID(
	listRepositories func(context.Context, devopssdk.ListRepositoriesRequest) (devopssdk.ListRepositoriesResponse, error),
) func(RepositoryServiceClient) RepositoryServiceClient {
	return func(delegate RepositoryServiceClient) RepositoryServiceClient {
		return repositoryDeleteWithoutTrackedIDClient{
			RepositoryServiceClient: delegate,
			listRepositories:        listRepositories,
		}
	}
}

func (c repositoryDeleteWithoutTrackedIDClient) Delete(
	ctx context.Context,
	resource *devopsv1beta1.Repository,
) (bool, error) {
	if repositoryTrackedID(resource) != "" {
		return c.RepositoryServiceClient.Delete(ctx, resource)
	}

	response, found, err := repositoryDeleteResolutionByList(ctx, resource, c.listRepositories)
	if err != nil {
		return false, err
	}
	if !found {
		markRepositoryDeleted(resource, "OCI resource no longer exists")
		return true, nil
	}
	if repositoryID := repositoryStringValue(response.Id); repositoryID != "" {
		resource.Status.Id = repositoryID
		resource.Status.OsokStatus.Ocid = shared.OCID(repositoryID)
	}
	return c.RepositoryServiceClient.Delete(ctx, resource)
}

func repositoryTrackedID(resource *devopsv1beta1.Repository) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	}
	return strings.TrimSpace(resource.Status.Id)
}

func repositoryDeleteConfirmRead(
	getRepository func(context.Context, devopssdk.GetRepositoryRequest) (devopssdk.GetRepositoryResponse, error),
	listRepositories func(context.Context, devopssdk.ListRepositoriesRequest) (devopssdk.ListRepositoriesResponse, error),
) func(context.Context, *devopsv1beta1.Repository, string) (any, error) {
	return func(ctx context.Context, resource *devopsv1beta1.Repository, currentID string) (any, error) {
		currentID = strings.TrimSpace(currentID)
		if currentID == "" {
			return repositoryDeleteConfirmReadByList(ctx, resource, listRepositories)
		}
		if getRepository == nil {
			return nil, fmt.Errorf("repository delete confirmation requires a readable OCI operation")
		}

		response, err := getRepository(ctx, devopssdk.GetRepositoryRequest{
			RepositoryId: common.String(currentID),
		})
		if err == nil {
			return response, nil
		}

		if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			return nil, err
		}
		handledErr := handleRepositoryDeleteError(resource, err)
		if handledErr == nil {
			handledErr = err
		}
		return repositoryAmbiguousDeleteConfirmResponse{
			Repository: devopssdk.Repository{
				Id:             common.String(currentID),
				LifecycleState: devopssdk.RepositoryLifecycleStateActive,
			},
			err: handledErr,
		}, nil
	}
}

func repositoryDeleteConfirmReadByList(
	ctx context.Context,
	resource *devopsv1beta1.Repository,
	listRepositories func(context.Context, devopssdk.ListRepositoriesRequest) (devopssdk.ListRepositoriesResponse, error),
) (any, error) {
	response, found, err := repositoryDeleteResolutionByList(ctx, resource, listRepositories)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("repository delete confirmation found no repository matching projectId %q and name %q", strings.TrimSpace(resource.Spec.ProjectId), strings.TrimSpace(resource.Spec.Name))
	}
	return response, nil
}

type repositoryDeleteListIdentity struct {
	projectID string
	name      string
}

func repositoryDeleteResolutionByList(
	ctx context.Context,
	resource *devopsv1beta1.Repository,
	listRepositories func(context.Context, devopssdk.ListRepositoriesRequest) (devopssdk.ListRepositoriesResponse, error),
) (devopssdk.GetRepositoryResponse, bool, error) {
	identity, err := repositoryDeleteListIdentityFor(resource, listRepositories)
	if err != nil {
		return devopssdk.GetRepositoryResponse{}, false, err
	}

	matches, opcRequestID, err := repositoryDeleteListMatches(ctx, identity, listRepositories)
	if err != nil {
		return devopssdk.GetRepositoryResponse{}, false, err
	}
	return repositoryDeleteResolutionFromMatches(identity, matches, opcRequestID)
}

func repositoryDeleteListIdentityFor(
	resource *devopsv1beta1.Repository,
	listRepositories func(context.Context, devopssdk.ListRepositoriesRequest) (devopssdk.ListRepositoriesResponse, error),
) (repositoryDeleteListIdentity, error) {
	if resource == nil {
		return repositoryDeleteListIdentity{}, fmt.Errorf("repository resource is nil")
	}
	if listRepositories == nil {
		return repositoryDeleteListIdentity{}, fmt.Errorf("repository delete confirmation requires a tracked repository OCID")
	}

	identity := repositoryDeleteListIdentity{
		projectID: strings.TrimSpace(resource.Spec.ProjectId),
		name:      strings.TrimSpace(resource.Spec.Name),
	}
	if identity.projectID == "" || identity.name == "" {
		return repositoryDeleteListIdentity{}, fmt.Errorf("repository delete confirmation requires projectId and name when no repository OCID is tracked")
	}
	return identity, nil
}

func repositoryDeleteListMatches(
	ctx context.Context,
	identity repositoryDeleteListIdentity,
	listRepositories func(context.Context, devopssdk.ListRepositoriesRequest) (devopssdk.ListRepositoriesResponse, error),
) ([]devopssdk.RepositorySummary, *string, error) {
	var matches []devopssdk.RepositorySummary
	var opcRequestID *string
	var page *string
	for {
		response, err := listRepositories(ctx, devopssdk.ListRepositoriesRequest{
			ProjectId: common.String(identity.projectID),
			Name:      common.String(identity.name),
			Page:      page,
		})
		if err != nil {
			return nil, nil, err
		}
		if opcRequestID == nil {
			opcRequestID = response.OpcRequestId
		}
		for _, item := range response.Items {
			if repositorySummaryMatchesDeleteIdentity(item, identity) {
				matches = append(matches, item)
			}
		}
		page = repositoryNextPage(response.OpcNextPage)
		if page == nil {
			break
		}
	}
	return matches, opcRequestID, nil
}

func repositorySummaryMatchesDeleteIdentity(
	item devopssdk.RepositorySummary,
	identity repositoryDeleteListIdentity,
) bool {
	return repositoryStringValue(item.ProjectId) == identity.projectID &&
		repositoryStringValue(item.Name) == identity.name
}

func repositoryNextPage(page *string) *string {
	if repositoryStringValue(page) == "" {
		return nil
	}
	return page
}

func repositoryDeleteResolutionFromMatches(
	identity repositoryDeleteListIdentity,
	matches []devopssdk.RepositorySummary,
	opcRequestID *string,
) (devopssdk.GetRepositoryResponse, bool, error) {
	switch len(matches) {
	case 0:
		return devopssdk.GetRepositoryResponse{OpcRequestId: opcRequestID}, false, nil
	case 1:
		return devopssdk.GetRepositoryResponse{
			Repository:   repositoryFromSummary(matches[0]),
			OpcRequestId: opcRequestID,
		}, true, nil
	default:
		return devopssdk.GetRepositoryResponse{}, false, fmt.Errorf("repository delete confirmation found %d repositories matching projectId %q and name %q", len(matches), identity.projectID, identity.name)
	}
}

func markRepositoryDeleted(resource *devopsv1beta1.Repository, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func applyRepositoryDeleteOutcome(
	_ *devopsv1beta1.Repository,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if err, ok := repositoryAmbiguousDeleteConfirmError(response); ok {
		if err != nil {
			return generatedruntime.DeleteOutcome{Handled: true}, err
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func repositoryAmbiguousDeleteConfirmError(response any) (error, bool) {
	switch typed := response.(type) {
	case repositoryAmbiguousDeleteConfirmResponse:
		return typed.err, true
	case *repositoryAmbiguousDeleteConfirmResponse:
		if typed == nil {
			return nil, false
		}
		return typed.err, true
	default:
		return nil, false
	}
}

type repositoryAmbiguousNotFoundError struct {
	HTTPStatusCode int
	ErrorCode      string
	OpcRequestID   string
	message        string
}

func (e repositoryAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e repositoryAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.OpcRequestID
}

func handleRepositoryDeleteError(resource *devopsv1beta1.Repository, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}

	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	return repositoryAmbiguousNotFoundError{
		HTTPStatusCode: classification.HTTPStatusCode,
		ErrorCode:      repositoryAmbiguousNotFoundErrorCode,
		OpcRequestID:   servicemanager.ErrorOpcRequestID(err),
		message: fmt.Sprintf(
			"Repository delete returned ambiguous not-found response (HTTP %s, code %s); retaining finalizer until OCI deletion is confirmed",
			classification.HTTPStatusCodeString(),
			classification.ErrorCodeString(),
		),
	}
}

func getRepositoryWorkRequest(
	ctx context.Context,
	client repositoryWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize Repository OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("repository OCI client is not configured")
	}
	response, err := client.GetWorkRequest(ctx, devopssdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func buildRepositoryUpdateBody(
	resource *devopsv1beta1.Repository,
	currentResponse any,
) (devopssdk.UpdateRepositoryDetails, bool, error) {
	if resource == nil {
		return devopssdk.UpdateRepositoryDetails{}, false, fmt.Errorf("repository resource is nil")
	}

	current, err := repositoryRuntimeBody(currentResponse)
	if err != nil {
		return devopssdk.UpdateRepositoryDetails{}, false, err
	}

	details := devopssdk.UpdateRepositoryDetails{}
	updateNeeded := false

	if desired, ok := repositoryDesiredRequiredStringUpdate(resource.Spec.Name, current.Name); ok {
		details.Name = desired
		updateNeeded = true
	}
	if desired, ok := repositoryDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := repositoryDesiredStringUpdate(resource.Spec.DefaultBranch, current.DefaultBranch); ok {
		details.DefaultBranch = desired
		updateNeeded = true
	}
	if desired, ok := repositoryDesiredRepositoryTypeUpdate(resource.Spec.RepositoryType, current.RepositoryType); ok {
		details.RepositoryType = desired
		updateNeeded = true
	}
	if desired, ok := repositoryDesiredMirrorConfigUpdate(resource.Spec.MirrorRepositoryConfig, current.MirrorRepositoryConfig); ok {
		details.MirrorRepositoryConfig = desired
		updateNeeded = true
	}
	if desired, ok := repositoryDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := repositoryDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func validateRepositoryCreateOnlyDrift(resource *devopsv1beta1.Repository, currentResponse any) error {
	if resource == nil || currentResponse == nil {
		return nil
	}

	current, err := repositoryRuntimeBody(currentResponse)
	if err != nil {
		return err
	}

	if desired := strings.TrimSpace(resource.Spec.ProjectId); desired != "" && desired != repositoryStringValue(current.ProjectId) {
		return fmt.Errorf("repository create-only field projectId requires replacement when changed")
	}

	desiredParentID := strings.TrimSpace(resource.Spec.ParentRepositoryId)
	currentParentID := repositoryStringValue(current.ParentRepositoryId)
	if desiredParentID != currentParentID && (desiredParentID != "" || currentParentID != "") {
		return fmt.Errorf("repository create-only field parentRepositoryId requires replacement when changed")
	}
	return nil
}

func repositoryRuntimeBody(currentResponse any) (devopssdk.Repository, error) {
	for _, convert := range repositoryRuntimeBodyConverters {
		repository, ok, err := convert(currentResponse)
		if err != nil {
			return repository, err
		}
		if ok {
			return repository, nil
		}
	}
	return devopssdk.Repository{}, fmt.Errorf("unexpected current repository response type %T", currentResponse)
}

func repositoryRuntimeBodyFromRepository(currentResponse any) (devopssdk.Repository, bool, error) {
	switch current := currentResponse.(type) {
	case devopssdk.Repository:
		return current, true, nil
	case *devopssdk.Repository:
		if current == nil {
			return devopssdk.Repository{}, true, fmt.Errorf("current repository response is nil")
		}
		return *current, true, nil
	default:
		return devopssdk.Repository{}, false, nil
	}
}

func repositoryRuntimeBodyFromSummary(currentResponse any) (devopssdk.Repository, bool, error) {
	switch current := currentResponse.(type) {
	case devopssdk.RepositorySummary:
		return repositoryFromSummary(current), true, nil
	case *devopssdk.RepositorySummary:
		if current == nil {
			return devopssdk.Repository{}, true, fmt.Errorf("current repository response is nil")
		}
		return repositoryFromSummary(*current), true, nil
	default:
		return devopssdk.Repository{}, false, nil
	}
}

func repositoryRuntimeBodyFromCreateResponse(currentResponse any) (devopssdk.Repository, bool, error) {
	switch current := currentResponse.(type) {
	case devopssdk.CreateRepositoryResponse:
		return current.Repository, true, nil
	case *devopssdk.CreateRepositoryResponse:
		if current == nil {
			return devopssdk.Repository{}, true, fmt.Errorf("current repository response is nil")
		}
		return current.Repository, true, nil
	default:
		return devopssdk.Repository{}, false, nil
	}
}

func repositoryRuntimeBodyFromGetResponse(currentResponse any) (devopssdk.Repository, bool, error) {
	switch current := currentResponse.(type) {
	case devopssdk.GetRepositoryResponse:
		return current.Repository, true, nil
	case *devopssdk.GetRepositoryResponse:
		if current == nil {
			return devopssdk.Repository{}, true, fmt.Errorf("current repository response is nil")
		}
		return current.Repository, true, nil
	default:
		return devopssdk.Repository{}, false, nil
	}
}

func repositoryRuntimeBodyFromUpdateResponse(currentResponse any) (devopssdk.Repository, bool, error) {
	switch current := currentResponse.(type) {
	case devopssdk.UpdateRepositoryResponse:
		return current.Repository, true, nil
	case *devopssdk.UpdateRepositoryResponse:
		if current == nil {
			return devopssdk.Repository{}, true, fmt.Errorf("current repository response is nil")
		}
		return current.Repository, true, nil
	default:
		return devopssdk.Repository{}, false, nil
	}
}

func repositoryFromSummary(summary devopssdk.RepositorySummary) devopssdk.Repository {
	return devopssdk.Repository{
		Id:                     summary.Id,
		CompartmentId:          summary.CompartmentId,
		ProjectId:              summary.ProjectId,
		Name:                   summary.Name,
		Namespace:              summary.Namespace,
		ParentRepositoryId:     summary.ParentRepositoryId,
		ProjectName:            summary.ProjectName,
		SshUrl:                 summary.SshUrl,
		HttpUrl:                summary.HttpUrl,
		Description:            summary.Description,
		DefaultBranch:          summary.DefaultBranch,
		RepositoryType:         summary.RepositoryType,
		MirrorRepositoryConfig: summary.MirrorRepositoryConfig,
		TimeCreated:            summary.TimeCreated,
		TimeUpdated:            summary.TimeUpdated,
		LifecycleState:         summary.LifecycleState,
		LifecyleDetails:        summary.LifecycleDetails,
		FreeformTags:           summary.FreeformTags,
		DefinedTags:            summary.DefinedTags,
		SystemTags:             summary.SystemTags,
	}
}

func repositoryDesiredRequiredStringUpdate(spec string, current *string) (*string, bool) {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == repositoryStringValue(current) {
		return nil, false
	}
	return common.String(spec), true
}

func repositoryDesiredStringUpdate(spec string, current *string) (*string, bool) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, false
	}
	currentValue := repositoryStringValue(current)
	if spec == currentValue {
		return nil, false
	}
	return common.String(spec), true
}

func repositoryDesiredRepositoryTypeUpdate(
	spec string,
	current devopssdk.RepositoryRepositoryTypeEnum,
) (devopssdk.RepositoryRepositoryTypeEnum, bool) {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == string(current) {
		return "", false
	}
	return devopssdk.RepositoryRepositoryTypeEnum(spec), true
}

func repositoryDesiredMirrorConfigUpdate(
	spec devopsv1beta1.RepositoryMirrorRepositoryConfig,
	current *devopssdk.MirrorRepositoryConfig,
) (*devopssdk.MirrorRepositoryConfig, bool) {
	desired, ok := repositoryMirrorConfigFromSpec(spec)
	if !ok {
		return nil, false
	}
	if current != nil && repositoryJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func repositoryMirrorConfigFromSpec(
	spec devopsv1beta1.RepositoryMirrorRepositoryConfig,
) (*devopssdk.MirrorRepositoryConfig, bool) {
	config := devopssdk.MirrorRepositoryConfig{}
	meaningful := false

	if strings.TrimSpace(spec.ConnectorId) != "" {
		config.ConnectorId = common.String(strings.TrimSpace(spec.ConnectorId))
		meaningful = true
	}
	if strings.TrimSpace(spec.RepositoryUrl) != "" {
		config.RepositoryUrl = common.String(strings.TrimSpace(spec.RepositoryUrl))
		meaningful = true
	}
	if triggerSchedule, ok := repositoryTriggerScheduleFromSpec(spec.TriggerSchedule); ok {
		config.TriggerSchedule = triggerSchedule
		meaningful = true
	}
	if !meaningful {
		return nil, false
	}
	return &config, true
}

func repositoryTriggerScheduleFromSpec(
	spec devopsv1beta1.RepositoryMirrorRepositoryConfigTriggerSchedule,
) (*devopssdk.TriggerSchedule, bool) {
	scheduleType := strings.TrimSpace(spec.ScheduleType)
	customSchedule := strings.TrimSpace(spec.CustomSchedule)
	if scheduleType == "" && customSchedule == "" {
		return nil, false
	}

	schedule := devopssdk.TriggerSchedule{
		ScheduleType: devopssdk.TriggerScheduleScheduleTypeEnum(scheduleType),
	}
	if customSchedule != "" {
		schedule.CustomSchedule = common.String(customSchedule)
	}
	return &schedule, true
}

func repositoryDesiredFreeformTagsUpdate(spec map[string]string, current map[string]string) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if repositoryJSONEqual(spec, current) {
		return nil, false
	}
	return repositoryCloneStringMap(spec), true
}

func repositoryDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := repositoryDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if repositoryJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func repositoryDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}

	desired := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		desired[namespace] = converted
	}
	return desired
}

func resolveRepositoryWorkRequestAction(workRequest any) (string, error) {
	repositoryWorkRequest, err := repositoryWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(repositoryWorkRequest.OperationType), nil
}

func resolveRepositoryWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	repositoryWorkRequest, err := repositoryWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}

	switch repositoryWorkRequest.OperationType {
	case devopssdk.OperationTypeCreateRepository,
		devopssdk.OperationTypeForkRepository,
		devopssdk.OperationTypeMirrorRepository:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case devopssdk.OperationTypeUpdateRepository,
		devopssdk.OperationTypeSyncForkRepository:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case devopssdk.OperationTypeDeleteRepository:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func recoverRepositoryIDFromWorkRequest(
	_ *devopsv1beta1.Repository,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	repositoryWorkRequest, err := repositoryWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	var fallback string
	for _, resource := range repositoryWorkRequest.Resources {
		if !repositoryWorkRequestActionMatchesPhase(resource.ActionType, phase) {
			continue
		}
		identifier := strings.TrimSpace(repositoryStringValue(resource.Identifier))
		if identifier == "" {
			continue
		}
		if repositoryWorkRequestResourceIsRepository(resource) {
			return identifier, nil
		}
		if fallback == "" {
			fallback = identifier
		}
	}
	return fallback, nil
}

func repositoryWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	repositoryWorkRequest, err := repositoryWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}

	workRequestID := repositoryStringValue(repositoryWorkRequest.Id)
	status := string(repositoryWorkRequest.Status)
	if workRequestID == "" || status == "" {
		return ""
	}
	message := fmt.Sprintf("Repository %s work request %s is %s", phase, workRequestID, status)
	if operationType := string(repositoryWorkRequest.OperationType); operationType != "" {
		message = message + " for " + operationType
	}
	return message
}

func repositoryWorkRequestFromAny(workRequest any) (devopssdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case devopssdk.WorkRequest:
		return current, nil
	case *devopssdk.WorkRequest:
		if current == nil {
			return devopssdk.WorkRequest{}, fmt.Errorf("repository work request is nil")
		}
		return *current, nil
	default:
		return devopssdk.WorkRequest{}, fmt.Errorf("unexpected repository work request type %T", workRequest)
	}
}

func repositoryWorkRequestActionMatchesPhase(action devopssdk.ActionTypeEnum, phase shared.OSOKAsyncPhase) bool {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return action == devopssdk.ActionTypeCreated || action == devopssdk.ActionTypeInProgress
	case shared.OSOKAsyncPhaseUpdate:
		return action == devopssdk.ActionTypeUpdated || action == devopssdk.ActionTypeInProgress
	case shared.OSOKAsyncPhaseDelete:
		return action == devopssdk.ActionTypeDeleted || action == devopssdk.ActionTypeInProgress
	default:
		return false
	}
}

func repositoryWorkRequestResourceIsRepository(resource devopssdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(repositoryStringValue(resource.EntityType)))
	entityType = strings.ReplaceAll(entityType, "_", "")
	entityType = strings.ReplaceAll(entityType, "-", "")
	return strings.Contains(entityType, "repository")
}

func repositoryStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func repositoryCloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func repositoryJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}
