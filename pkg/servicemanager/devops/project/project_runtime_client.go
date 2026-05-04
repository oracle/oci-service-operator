/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package project

import (
	"context"
	"errors"
	"fmt"
	"reflect"
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
)

var projectWorkRequestAsyncAdapter = servicemanagerWorkRequestAdapter()

type projectOCIClient interface {
	CreateProject(context.Context, devopssdk.CreateProjectRequest) (devopssdk.CreateProjectResponse, error)
	GetProject(context.Context, devopssdk.GetProjectRequest) (devopssdk.GetProjectResponse, error)
	ListProjects(context.Context, devopssdk.ListProjectsRequest) (devopssdk.ListProjectsResponse, error)
	UpdateProject(context.Context, devopssdk.UpdateProjectRequest) (devopssdk.UpdateProjectResponse, error)
	DeleteProject(context.Context, devopssdk.DeleteProjectRequest) (devopssdk.DeleteProjectResponse, error)
	GetWorkRequest(context.Context, devopssdk.GetWorkRequestRequest) (devopssdk.GetWorkRequestResponse, error)
}

type ambiguousProjectNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousProjectNotFoundError) Error() string {
	return e.message
}

func (e ambiguousProjectNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerProjectRuntimeHooksMutator(func(manager *ProjectServiceManager, hooks *ProjectRuntimeHooks) {
		client, initErr := newProjectSDKClient(manager)
		applyProjectRuntimeHooks(manager, hooks, client, initErr)
	})
}

func servicemanagerWorkRequestAdapter() servicemanager.WorkRequestAsyncAdapter {
	return servicemanager.WorkRequestAsyncAdapter{
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
		CreateActionTokens:    []string{string(devopssdk.ActionTypeCreated)},
		UpdateActionTokens:    []string{string(devopssdk.ActionTypeUpdated)},
		DeleteActionTokens:    []string{string(devopssdk.ActionTypeDeleted)},
	}
}

func newProjectSDKClient(manager *ProjectServiceManager) (projectOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("Project service manager is nil")
	}
	client, err := devopssdk.NewDevopsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyProjectRuntimeHooks(
	_ *ProjectServiceManager,
	hooks *ProjectRuntimeHooks,
	client projectOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedProjectRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *devopsv1beta1.Project, _ string) (any, error) {
		if resource == nil {
			return nil, fmt.Errorf("Project resource is nil")
		}
		return buildProjectCreateBody(resource.Spec)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *devopsv1beta1.Project,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildProjectUpdateBody(resource, currentResponse)
	}
	hooks.Get.Call = func(ctx context.Context, request devopssdk.GetProjectRequest) (devopssdk.GetProjectResponse, error) {
		if initErr != nil {
			return devopssdk.GetProjectResponse{}, fmt.Errorf("initialize Project OCI client: %w", initErr)
		}
		if client == nil {
			return devopssdk.GetProjectResponse{}, fmt.Errorf("Project OCI client is not configured")
		}
		response, err := client.GetProject(ctx, request)
		return response, conservativeProjectNotFoundError(err, "read")
	}
	hooks.List.Fields = projectListFields()
	hooks.List.Call = func(ctx context.Context, request devopssdk.ListProjectsRequest) (devopssdk.ListProjectsResponse, error) {
		return listProjectsAllPages(ctx, client, initErr, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request devopssdk.DeleteProjectRequest) (devopssdk.DeleteProjectResponse, error) {
		if initErr != nil {
			return devopssdk.DeleteProjectResponse{}, fmt.Errorf("initialize Project OCI client: %w", initErr)
		}
		if client == nil {
			return devopssdk.DeleteProjectResponse{}, fmt.Errorf("Project OCI client is not configured")
		}
		response, err := client.DeleteProject(ctx, request)
		return response, conservativeProjectNotFoundError(err, "delete")
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedProjectIdentity
	hooks.StatusHooks.ProjectStatus = projectStatusFromResponse
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateProjectCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleProjectDeleteError
	hooks.Async.Adapter = projectWorkRequestAsyncAdapter
	hooks.Async.ResolveAction = resolveProjectGeneratedWorkRequestAction
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getProjectWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolvePhase = resolveProjectGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverProjectIDFromGeneratedWorkRequest
	hooks.Async.Message = projectGeneratedWorkRequestMessage
}

func newProjectServiceClientWithOCIClient(log loggerutil.OSOKLogger, client projectOCIClient) ProjectServiceClient {
	manager := &ProjectServiceManager{Log: log}
	hooks := newProjectRuntimeHooksWithOCIClient(client)
	applyProjectRuntimeHooks(manager, &hooks, client, nil)
	return defaultProjectServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*devopsv1beta1.Project](
			buildProjectGeneratedRuntimeConfig(manager, hooks),
		),
	}
}

func newProjectRuntimeHooksWithOCIClient(client projectOCIClient) ProjectRuntimeHooks {
	return ProjectRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*devopsv1beta1.Project]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*devopsv1beta1.Project]{},
		StatusHooks:     generatedruntime.StatusHooks[*devopsv1beta1.Project]{},
		ParityHooks:     generatedruntime.ParityHooks[*devopsv1beta1.Project]{},
		Async:           generatedruntime.AsyncHooks[*devopsv1beta1.Project]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*devopsv1beta1.Project]{},
		Create: runtimeOperationHooks[devopssdk.CreateProjectRequest, devopssdk.CreateProjectResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateProjectDetails", RequestName: "CreateProjectDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request devopssdk.CreateProjectRequest) (devopssdk.CreateProjectResponse, error) {
				return client.CreateProject(ctx, request)
			},
		},
		Get: runtimeOperationHooks[devopssdk.GetProjectRequest, devopssdk.GetProjectResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ProjectId", RequestName: "projectId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request devopssdk.GetProjectRequest) (devopssdk.GetProjectResponse, error) {
				return client.GetProject(ctx, request)
			},
		},
		List: runtimeOperationHooks[devopssdk.ListProjectsRequest, devopssdk.ListProjectsResponse]{
			Fields: projectListFields(),
			Call: func(ctx context.Context, request devopssdk.ListProjectsRequest) (devopssdk.ListProjectsResponse, error) {
				return client.ListProjects(ctx, request)
			},
		},
		Update: runtimeOperationHooks[devopssdk.UpdateProjectRequest, devopssdk.UpdateProjectResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ProjectId", RequestName: "projectId", Contribution: "path", PreferResourceID: true}, {FieldName: "UpdateProjectDetails", RequestName: "UpdateProjectDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request devopssdk.UpdateProjectRequest) (devopssdk.UpdateProjectResponse, error) {
				return client.UpdateProject(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[devopssdk.DeleteProjectRequest, devopssdk.DeleteProjectResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ProjectId", RequestName: "projectId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request devopssdk.DeleteProjectRequest) (devopssdk.DeleteProjectResponse, error) {
				return client.DeleteProject(ctx, request)
			},
		},
	}
}

func reviewedProjectRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "devops",
		FormalSlug:    "project",
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
			ProvisioningStates: []string{string(devopssdk.ProjectLifecycleStateCreating)},
			UpdatingStates:     []string{string(devopssdk.ProjectLifecycleStateUpdating)},
			ActiveStates:       []string{string(devopssdk.ProjectLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(devopssdk.ProjectLifecycleStateDeleting)},
			TerminalStates: []string{string(devopssdk.ProjectLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"description", "notificationConfig", "freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId", "name"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "project", Action: "CREATED"},
			},
			Update: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "project", Action: "UPDATED"},
			},
			Delete: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "project", Action: "DELETED"},
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetProject",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "project", Action: "CREATED"},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetProject",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "project", Action: "UPDATED"},
			},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "project", Action: "DELETED"},
			},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func projectListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "Name", RequestName: "name", Contribution: "query", LookupPaths: []string{"status.name", "spec.name", "name"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
	}
}

func buildProjectCreateBody(spec devopsv1beta1.ProjectSpec) (devopssdk.CreateProjectDetails, error) {
	if err := validateProjectSpec(spec); err != nil {
		return devopssdk.CreateProjectDetails{}, err
	}

	body := devopssdk.CreateProjectDetails{
		Name:               common.String(strings.TrimSpace(spec.Name)),
		CompartmentId:      common.String(strings.TrimSpace(spec.CompartmentId)),
		NotificationConfig: projectNotificationConfig(spec.NotificationConfig),
	}
	if strings.TrimSpace(spec.Description) != "" {
		body.Description = common.String(spec.Description)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	return body, nil
}

func buildProjectUpdateBody(
	resource *devopsv1beta1.Project,
	currentResponse any,
) (devopssdk.UpdateProjectDetails, bool, error) {
	if resource == nil {
		return devopssdk.UpdateProjectDetails{}, false, fmt.Errorf("Project resource is nil")
	}
	if err := validateProjectSpec(resource.Spec); err != nil {
		return devopssdk.UpdateProjectDetails{}, false, err
	}

	current, ok := projectFromResponse(currentResponse)
	if !ok {
		return devopssdk.UpdateProjectDetails{}, false, fmt.Errorf("current Project response does not expose a Project body")
	}
	if err := validateProjectCreateOnlyDrift(resource.Spec, current); err != nil {
		return devopssdk.UpdateProjectDetails{}, false, err
	}

	updateDetails := devopssdk.UpdateProjectDetails{}
	updateNeeded := false

	if !stringPtrEqual(current.Description, resource.Spec.Description) {
		updateDetails.Description = common.String(resource.Spec.Description)
		updateNeeded = true
	}
	if !notificationConfigEqual(current.NotificationConfig, resource.Spec.NotificationConfig) {
		updateDetails.NotificationConfig = projectNotificationConfig(resource.Spec.NotificationConfig)
		updateNeeded = true
	}

	desiredFreeformTags := desiredProjectFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}

	desiredDefinedTags := desiredProjectDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}

	if !updateNeeded {
		return devopssdk.UpdateProjectDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func validateProjectSpec(spec devopsv1beta1.ProjectSpec) error {
	var missing []string
	if strings.TrimSpace(spec.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.NotificationConfig.TopicId) == "" {
		missing = append(missing, "notificationConfig.topicId")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("Project spec is missing required field(s): %s", strings.Join(missing, ", "))
}

func projectNotificationConfig(config devopsv1beta1.ProjectNotificationConfig) *devopssdk.NotificationConfig {
	return &devopssdk.NotificationConfig{TopicId: common.String(strings.TrimSpace(config.TopicId))}
}

func notificationConfigEqual(current *devopssdk.NotificationConfig, desired devopsv1beta1.ProjectNotificationConfig) bool {
	if current == nil {
		return strings.TrimSpace(desired.TopicId) == ""
	}
	return strings.TrimSpace(stringValue(current.TopicId)) == strings.TrimSpace(desired.TopicId)
}

func validateProjectCreateOnlyDriftForResponse(resource *devopsv1beta1.Project, currentResponse any) error {
	if resource == nil {
		return fmt.Errorf("Project resource is nil")
	}
	current, ok := projectFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current Project response does not expose a Project body")
	}
	return validateProjectCreateOnlyDrift(resource.Spec, current)
}

func validateProjectCreateOnlyDrift(spec devopsv1beta1.ProjectSpec, current devopssdk.Project) error {
	var drift []string
	if !stringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		drift = append(drift, "compartmentId")
	}
	if !stringPtrEqual(current.Name, spec.Name) {
		drift = append(drift, "name")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("Project create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func projectStatusFromResponse(resource *devopsv1beta1.Project, response any) error {
	if resource == nil {
		return fmt.Errorf("Project resource is nil")
	}

	current, ok := projectFromResponse(response)
	if !ok {
		return nil
	}
	projectStatus(resource, current)
	return nil
}

func projectStatus(resource *devopsv1beta1.Project, current devopssdk.Project) {
	resource.Status = devopsv1beta1.ProjectStatus{
		OsokStatus:         resource.Status.OsokStatus,
		Id:                 stringValue(current.Id),
		Name:               stringValue(current.Name),
		CompartmentId:      stringValue(current.CompartmentId),
		NotificationConfig: statusNotificationConfig(current.NotificationConfig),
		Description:        stringValue(current.Description),
		Namespace:          stringValue(current.Namespace),
		TimeCreated:        sdkTimeString(current.TimeCreated),
		TimeUpdated:        sdkTimeString(current.TimeUpdated),
		LifecycleState:     string(current.LifecycleState),
		LifecycleDetails:   stringValue(current.LifecycleDetails),
		FreeformTags:       cloneStringMap(current.FreeformTags),
		DefinedTags:        convertOCIToStatusDefinedTags(current.DefinedTags),
		SystemTags:         convertOCIToStatusDefinedTags(current.SystemTags),
	}
}

func statusNotificationConfig(config *devopssdk.NotificationConfig) devopsv1beta1.ProjectNotificationConfig {
	if config == nil {
		return devopsv1beta1.ProjectNotificationConfig{}
	}
	return devopsv1beta1.ProjectNotificationConfig{TopicId: stringValue(config.TopicId)}
}

func clearTrackedProjectIdentity(resource *devopsv1beta1.Project) {
	if resource == nil {
		return
	}
	resource.Status = devopsv1beta1.ProjectStatus{}
}

func desiredProjectFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredProjectDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) map[string]map[string]interface{} {
	if spec != nil {
		return *util.ConvertToOciDefinedTags(&spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func getProjectWorkRequest(
	ctx context.Context,
	client projectOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize Project OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("Project OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, devopssdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveProjectGeneratedWorkRequestAction(workRequest any) (string, error) {
	projectWorkRequest, err := projectWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveProjectWorkRequestAction(projectWorkRequest)
}

func resolveProjectGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	projectWorkRequest, err := projectWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := projectWorkRequestPhaseFromOperationType(projectWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverProjectIDFromGeneratedWorkRequest(
	_ *devopsv1beta1.Project,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	projectWorkRequest, err := projectWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveProjectIDFromWorkRequest(projectWorkRequest, projectWorkRequestActionForPhase(phase))
}

func resolveProjectWorkRequestAction(workRequest devopssdk.WorkRequest) (string, error) {
	var action string
	for _, resource := range workRequest.Resources {
		if !isProjectWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" || strings.EqualFold(candidate, string(devopssdk.ActionTypeInProgress)) {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf("Project work request %s exposes conflicting Project action types %q and %q", stringValue(workRequest.Id), action, candidate)
		}
	}
	return action, nil
}

func projectWorkRequestPhaseFromOperationType(operationType devopssdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case devopssdk.OperationTypeCreateProject:
		return shared.OSOKAsyncPhaseCreate, true
	case devopssdk.OperationTypeUpdateProject:
		return shared.OSOKAsyncPhaseUpdate, true
	case devopssdk.OperationTypeDeleteProject:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func projectWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) devopssdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return devopssdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return devopssdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return devopssdk.ActionTypeDeleted
	default:
		return ""
	}
}

func projectGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	projectWorkRequest, err := projectWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("Project %s work request %s is %s", phase, stringValue(projectWorkRequest.Id), projectWorkRequest.Status)
}

func projectWorkRequestFromAny(workRequest any) (devopssdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case devopssdk.WorkRequest:
		return current, nil
	case *devopssdk.WorkRequest:
		if current == nil {
			return devopssdk.WorkRequest{}, fmt.Errorf("Project work request is nil")
		}
		return *current, nil
	default:
		return devopssdk.WorkRequest{}, fmt.Errorf("unexpected Project work request type %T", workRequest)
	}
}

func resolveProjectIDFromWorkRequest(workRequest devopssdk.WorkRequest, action devopssdk.ActionTypeEnum) (string, error) {
	if id, ok := resolveProjectIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveProjectIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("Project work request %s does not expose a Project identifier", stringValue(workRequest.Id))
}

func resolveProjectIDFromResources(
	resources []devopssdk.WorkRequestResource,
	action devopssdk.ActionTypeEnum,
	preferProjectOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferProjectOnly && !isProjectWorkRequestResource(resource) {
			continue
		}
		id := strings.TrimSpace(stringValue(resource.Identifier))
		if id == "" {
			continue
		}
		if candidate == "" {
			candidate = id
			continue
		}
		if candidate != id {
			return "", false
		}
	}
	return candidate, candidate != ""
}

func isProjectWorkRequestResource(resource devopssdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	switch entityType {
	case "project", "projects", "devops_project", "devopsproject":
		return true
	}
	if strings.Contains(entityType, "project") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/projects/")
}

func listProjectsAllPages(
	ctx context.Context,
	client projectOCIClient,
	initErr error,
	request devopssdk.ListProjectsRequest,
) (devopssdk.ListProjectsResponse, error) {
	if initErr != nil {
		return devopssdk.ListProjectsResponse{}, fmt.Errorf("initialize Project OCI client: %w", initErr)
	}
	if client == nil {
		return devopssdk.ListProjectsResponse{}, fmt.Errorf("Project OCI client is not configured")
	}

	var combined devopssdk.ListProjectsResponse
	for {
		response, err := client.ListProjects(ctx, request)
		if err != nil {
			return devopssdk.ListProjectsResponse{}, conservativeProjectNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == devopssdk.ProjectLifecycleStateDeleted {
				continue
			}
			combined.Items = append(combined.Items, item)
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func handleProjectDeleteError(resource *devopsv1beta1.Project, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func conservativeProjectNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("Project %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousProjectNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousProjectNotFoundError{message: message}
}

func projectFromResponse(response any) (devopssdk.Project, bool) {
	switch current := response.(type) {
	case devopssdk.CreateProjectResponse:
		return current.Project, true
	case *devopssdk.CreateProjectResponse:
		if current == nil {
			return devopssdk.Project{}, false
		}
		return current.Project, true
	case devopssdk.GetProjectResponse:
		return current.Project, true
	case *devopssdk.GetProjectResponse:
		if current == nil {
			return devopssdk.Project{}, false
		}
		return current.Project, true
	case devopssdk.UpdateProjectResponse:
		return current.Project, true
	case *devopssdk.UpdateProjectResponse:
		if current == nil {
			return devopssdk.Project{}, false
		}
		return current.Project, true
	case devopssdk.Project:
		return current, true
	case *devopssdk.Project:
		if current == nil {
			return devopssdk.Project{}, false
		}
		return *current, true
	case devopssdk.ProjectSummary:
		return projectFromSummary(current), true
	case *devopssdk.ProjectSummary:
		if current == nil {
			return devopssdk.Project{}, false
		}
		return projectFromSummary(*current), true
	default:
		return devopssdk.Project{}, false
	}
}

func projectFromSummary(summary devopssdk.ProjectSummary) devopssdk.Project {
	return devopssdk.Project{
		Id:                 summary.Id,
		Name:               summary.Name,
		CompartmentId:      summary.CompartmentId,
		NotificationConfig: summary.NotificationConfig,
		Description:        summary.Description,
		Namespace:          summary.Namespace,
		TimeCreated:        summary.TimeCreated,
		TimeUpdated:        summary.TimeUpdated,
		LifecycleState:     summary.LifecycleState,
		LifecycleDetails:   summary.LifecycleDetails,
		FreeformTags:       cloneStringMap(summary.FreeformTags),
		DefinedTags:        cloneDefinedTags(summary.DefinedTags),
		SystemTags:         cloneDefinedTags(summary.SystemTags),
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPtrEqual(actual *string, expected string) bool {
	return strings.TrimSpace(stringValue(actual)) == strings.TrimSpace(expected)
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneDefinedTags(input map[string]map[string]interface{}) map[string]map[string]interface{} {
	if input == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(input))
	for key, values := range input {
		if values == nil {
			cloned[key] = nil
			continue
		}
		inner := make(map[string]interface{}, len(values))
		for innerKey, innerValue := range values {
			inner[innerKey] = innerValue
		}
		cloned[key] = inner
	}
	return cloned
}

func convertOCIToStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for key, values := range input {
		if values == nil {
			converted[key] = nil
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for innerKey, innerValue := range values {
			tagValues[innerKey] = fmt.Sprint(innerValue)
		}
		converted[key] = tagValues
	}
	return converted
}
