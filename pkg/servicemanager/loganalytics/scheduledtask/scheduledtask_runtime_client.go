/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package scheduledtask

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	scheduledTaskKindAcceleration  = "ACCELERATION"
	scheduledTaskKindStandard      = "STANDARD"
	scheduledTaskJSONSavedSearchID = "savedSearchId"
)

type scheduledTaskOCIClient interface {
	CreateScheduledTask(context.Context, loganalyticssdk.CreateScheduledTaskRequest) (loganalyticssdk.CreateScheduledTaskResponse, error)
	GetScheduledTask(context.Context, loganalyticssdk.GetScheduledTaskRequest) (loganalyticssdk.GetScheduledTaskResponse, error)
	ListScheduledTasks(context.Context, loganalyticssdk.ListScheduledTasksRequest) (loganalyticssdk.ListScheduledTasksResponse, error)
	UpdateScheduledTask(context.Context, loganalyticssdk.UpdateScheduledTaskRequest) (loganalyticssdk.UpdateScheduledTaskResponse, error)
	DeleteScheduledTask(context.Context, loganalyticssdk.DeleteScheduledTaskRequest) (loganalyticssdk.DeleteScheduledTaskResponse, error)
	ListNamespaces(context.Context, loganalyticssdk.ListNamespacesRequest) (loganalyticssdk.ListNamespacesResponse, error)
}

type namespaceResolvingScheduledTaskClient struct {
	delegate ScheduledTaskServiceClient
	client   scheduledTaskOCIClient
	initErr  error
}

type scheduledTaskResourceContextKey struct{}

type updateAccelerationTaskDetails struct {
	DisplayName  *string                           `json:"displayName,omitempty"`
	Description  *string                           `json:"description,omitempty"`
	FreeformTags map[string]string                 `json:"freeformTags,omitempty"`
	DefinedTags  map[string]map[string]interface{} `json:"definedTags,omitempty"`
}

func init() {
	registerScheduledTaskRuntimeHooksMutator(func(manager *ScheduledTaskServiceManager, hooks *ScheduledTaskRuntimeHooks) {
		client, initErr := newScheduledTaskSDKClient(manager)
		applyScheduledTaskRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newScheduledTaskSDKClient(manager *ScheduledTaskServiceManager) (scheduledTaskOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("ScheduledTask service manager is nil")
	}
	client, err := loganalyticssdk.NewLogAnalyticsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyScheduledTaskRuntimeHooks(
	manager *ScheduledTaskServiceManager,
	hooks *ScheduledTaskRuntimeHooks,
	client scheduledTaskOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedScheduledTaskRuntimeSemantics()
	applyScheduledTaskBodyHooks(hooks)
	hooks.Identity.GuardExistingBeforeCreate = guardScheduledTaskExistingBeforeCreate
	applyScheduledTaskCreateHook(hooks, client, initErr)
	applyScheduledTaskGetHook(hooks, client, initErr)
	applyScheduledTaskListHook(hooks, client, initErr)
	applyScheduledTaskUpdateHook(hooks, client, initErr)
	applyScheduledTaskDeleteHook(hooks, client, initErr)
	hooks.StatusHooks.ProjectStatus = projectScheduledTaskStatusFromResponse
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateScheduledTaskCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleScheduledTaskDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ScheduledTaskServiceClient) ScheduledTaskServiceClient {
		return &namespaceResolvingScheduledTaskClient{
			delegate: delegate,
			client:   client,
			initErr:  initErr,
		}
	})
}

func applyScheduledTaskBodyHooks(hooks *ScheduledTaskRuntimeHooks) {
	hooks.BuildCreateBody = func(_ context.Context, resource *loganalyticsv1beta1.ScheduledTask, _ string) (any, error) {
		return buildScheduledTaskCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *loganalyticsv1beta1.ScheduledTask,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildScheduledTaskUpdateBody(resource, currentResponse)
	}
}

func applyScheduledTaskCreateHook(
	hooks *ScheduledTaskRuntimeHooks,
	client scheduledTaskOCIClient,
	initErr error,
) {
	hooks.Create.Fields = scheduledTaskCreateFields()
	hooks.Create.Call = func(ctx context.Context, request loganalyticssdk.CreateScheduledTaskRequest) (loganalyticssdk.CreateScheduledTaskResponse, error) {
		if err := scheduledTaskInitError(initErr); err != nil {
			return loganalyticssdk.CreateScheduledTaskResponse{}, err
		}
		if client == nil {
			return loganalyticssdk.CreateScheduledTaskResponse{}, fmt.Errorf("ScheduledTask OCI client is not configured")
		}
		resource, err := scheduledTaskResourceFromContext(ctx)
		if err != nil {
			return loganalyticssdk.CreateScheduledTaskResponse{}, err
		}
		body, err := buildScheduledTaskCreateBody(resource)
		if err != nil {
			return loganalyticssdk.CreateScheduledTaskResponse{}, fmt.Errorf("build ScheduledTask create body: %w", err)
		}
		details, ok := body.(loganalyticssdk.CreateScheduledTaskDetails)
		if !ok {
			return loganalyticssdk.CreateScheduledTaskResponse{}, fmt.Errorf("build ScheduledTask create body: %T does not implement CreateScheduledTaskDetails", body)
		}
		request.CreateScheduledTaskDetails = details
		return client.CreateScheduledTask(ctx, request)
	}
}

func applyScheduledTaskGetHook(
	hooks *ScheduledTaskRuntimeHooks,
	client scheduledTaskOCIClient,
	initErr error,
) {
	hooks.Get.Fields = scheduledTaskGetFields()
	hooks.Get.Call = func(ctx context.Context, request loganalyticssdk.GetScheduledTaskRequest) (loganalyticssdk.GetScheduledTaskResponse, error) {
		if err := scheduledTaskInitError(initErr); err != nil {
			return loganalyticssdk.GetScheduledTaskResponse{}, err
		}
		if client == nil {
			return loganalyticssdk.GetScheduledTaskResponse{}, fmt.Errorf("ScheduledTask OCI client is not configured")
		}
		return client.GetScheduledTask(ctx, request)
	}
}

func applyScheduledTaskListHook(
	hooks *ScheduledTaskRuntimeHooks,
	client scheduledTaskOCIClient,
	initErr error,
) {
	hooks.List.Fields = scheduledTaskListFields()
	hooks.List.Call = func(ctx context.Context, request loganalyticssdk.ListScheduledTasksRequest) (loganalyticssdk.ListScheduledTasksResponse, error) {
		if request.TaskType == "" {
			if resource, err := scheduledTaskResourceFromContext(ctx); err == nil {
				if taskType, err := scheduledTaskTaskType(resource); err == nil {
					request.TaskType = loganalyticssdk.ListScheduledTasksTaskTypeEnum(taskType)
				}
			}
		}
		return listScheduledTaskPages(ctx, client, initErr, request)
	}
}

func applyScheduledTaskUpdateHook(
	hooks *ScheduledTaskRuntimeHooks,
	client scheduledTaskOCIClient,
	initErr error,
) {
	hooks.Update.Fields = scheduledTaskUpdateFields()
	hooks.Update.Call = func(ctx context.Context, request loganalyticssdk.UpdateScheduledTaskRequest) (loganalyticssdk.UpdateScheduledTaskResponse, error) {
		if err := scheduledTaskInitError(initErr); err != nil {
			return loganalyticssdk.UpdateScheduledTaskResponse{}, err
		}
		if client == nil {
			return loganalyticssdk.UpdateScheduledTaskResponse{}, fmt.Errorf("ScheduledTask OCI client is not configured")
		}
		resource, err := scheduledTaskResourceFromContext(ctx)
		if err != nil {
			return loganalyticssdk.UpdateScheduledTaskResponse{}, err
		}
		body, _, err := buildScheduledTaskUpdateBody(resource, nil)
		if err != nil {
			return loganalyticssdk.UpdateScheduledTaskResponse{}, fmt.Errorf("build ScheduledTask update body: %w", err)
		}
		if body == nil {
			return loganalyticssdk.UpdateScheduledTaskResponse{}, fmt.Errorf("build ScheduledTask update body: no mutable update body produced")
		}
		details, ok := body.(loganalyticssdk.UpdateScheduledTaskDetails)
		if !ok {
			return loganalyticssdk.UpdateScheduledTaskResponse{}, fmt.Errorf("build ScheduledTask update body: %T does not implement UpdateScheduledTaskDetails", body)
		}
		request.UpdateScheduledTaskDetails = details
		return client.UpdateScheduledTask(ctx, request)
	}
}

func applyScheduledTaskDeleteHook(
	hooks *ScheduledTaskRuntimeHooks,
	client scheduledTaskOCIClient,
	initErr error,
) {
	hooks.Delete.Fields = scheduledTaskDeleteFields()
	hooks.Delete.Call = func(ctx context.Context, request loganalyticssdk.DeleteScheduledTaskRequest) (loganalyticssdk.DeleteScheduledTaskResponse, error) {
		if err := scheduledTaskInitError(initErr); err != nil {
			return loganalyticssdk.DeleteScheduledTaskResponse{}, err
		}
		if client == nil {
			return loganalyticssdk.DeleteScheduledTaskResponse{}, fmt.Errorf("ScheduledTask OCI client is not configured")
		}
		return client.DeleteScheduledTask(ctx, request)
	}
}

func scheduledTaskInitError(initErr error) error {
	if initErr == nil {
		return nil
	}
	return fmt.Errorf("initialize ScheduledTask OCI client: %w", initErr)
}

func scheduledTaskResourceFromContext(ctx context.Context) (*loganalyticsv1beta1.ScheduledTask, error) {
	if ctx == nil {
		return nil, fmt.Errorf("ScheduledTask runtime context is nil")
	}
	resource, ok := ctx.Value(scheduledTaskResourceContextKey{}).(*loganalyticsv1beta1.ScheduledTask)
	if !ok || resource == nil {
		return nil, fmt.Errorf("ScheduledTask runtime resource missing from context")
	}
	return resource, nil
}

func reviewedScheduledTaskRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "loganalytics",
		FormalSlug:    "scheduledtask",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(loganalyticssdk.ScheduledTaskLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "best-effort",
			TerminalStates: []string{string(loganalyticssdk.ScheduledTaskLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "taskType", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"action",
				"definedTags",
				"description",
				"displayName",
				"freeformTags",
				"schedules",
			},
			ForceNew: []string{
				"compartmentId",
				"kind",
				"savedSearchId",
				"taskType",
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
		},
	}
}

func scheduledTaskCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		scheduledTaskNamespaceField(),
	}
}

func scheduledTaskGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		scheduledTaskNamespaceField(),
		scheduledTaskIDField(),
	}
}

func scheduledTaskListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		scheduledTaskNamespaceField(),
		{FieldName: "TaskType", RequestName: "taskType", Contribution: "query"},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "TemplateId", RequestName: "templateId", Contribution: "query", LookupPaths: []string{"action.templateDetails.templateId"}},
		{FieldName: "SavedSearchId", RequestName: "savedSearchId", Contribution: "query", LookupPaths: []string{"savedSearchId", "action.savedSearchId"}},
		{FieldName: "DisplayNameContains", RequestName: "displayNameContains", Contribution: "query"},
		{FieldName: "TargetService", RequestName: "targetService", Contribution: "query"},
	}
}

func scheduledTaskUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		scheduledTaskNamespaceField(),
		scheduledTaskIDField(),
	}
}

func scheduledTaskDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		scheduledTaskNamespaceField(),
		scheduledTaskIDField(),
	}
}

func scheduledTaskNamespaceField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "NamespaceName",
		RequestName:  "namespaceName",
		Contribution: "path",
	}
}

func scheduledTaskIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:        "ScheduledTaskId",
		RequestName:      "scheduledTaskId",
		Contribution:     "path",
		PreferResourceID: true,
	}
}

func guardScheduledTaskExistingBeforeCreate(
	_ context.Context,
	resource *loganalyticsv1beta1.ScheduledTask,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("ScheduledTask resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}

	if scheduledTaskKind(resource) == scheduledTaskKindAcceleration {
		if strings.TrimSpace(resource.Spec.DisplayName) == "" {
			return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("spec.displayName is required for ACCELERATION ScheduledTask create-or-bind identity")
		}
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}

	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	if _, err := scheduledTaskTaskType(resource); err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func (c *namespaceResolvingScheduledTaskClient) CreateOrUpdate(
	ctx context.Context,
	resource *loganalyticsv1beta1.ScheduledTask,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("ScheduledTask runtime client is not configured")
	}
	restore, err := c.applyLogAnalyticsNamespace(ctx, resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	defer restore()
	return c.delegate.CreateOrUpdate(context.WithValue(ctx, scheduledTaskResourceContextKey{}, resource), resource, req)
}

func (c *namespaceResolvingScheduledTaskClient) Delete(ctx context.Context, resource *loganalyticsv1beta1.ScheduledTask) (bool, error) {
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("ScheduledTask runtime client is not configured")
	}
	restore, err := c.applyLogAnalyticsNamespace(ctx, resource)
	if err != nil {
		return false, err
	}
	defer restore()
	if err := c.rejectAuthShapedPreDeleteConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *namespaceResolvingScheduledTaskClient) applyLogAnalyticsNamespace(
	ctx context.Context,
	resource *loganalyticsv1beta1.ScheduledTask,
) (func(), error) {
	if resource == nil {
		return nil, fmt.Errorf("ScheduledTask resource is nil")
	}
	namespace, err := c.resolveLogAnalyticsNamespace(ctx, resource)
	if err != nil {
		return nil, err
	}
	original := resource.Namespace
	resource.Namespace = namespace
	return func() {
		resource.Namespace = original
	}, nil
}

func (c *namespaceResolvingScheduledTaskClient) resolveLogAnalyticsNamespace(
	ctx context.Context,
	resource *loganalyticsv1beta1.ScheduledTask,
) (string, error) {
	if namespace := scheduledTaskNamespaceFromJSON(resource.Spec.JsonData); namespace != "" {
		return namespace, nil
	}
	if c.initErr != nil {
		return "", fmt.Errorf("initialize ScheduledTask OCI client: %w", c.initErr)
	}
	if c.client == nil {
		return "", fmt.Errorf("ScheduledTask OCI client is not configured")
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		return "", fmt.Errorf("lookup ScheduledTask namespace: spec.compartmentId is required")
	}
	response, err := c.client.ListNamespaces(ctx, loganalyticssdk.ListNamespacesRequest{
		CompartmentId: common.String(compartmentID),
	})
	if err != nil {
		return "", fmt.Errorf("lookup ScheduledTask namespace: %w", err)
	}

	namespace, err := scheduledTaskNamespaceFromCollection(compartmentID, response.NamespaceCollection)
	if err != nil {
		return "", fmt.Errorf("lookup ScheduledTask namespace: %w", err)
	}
	return namespace, nil
}

func (c *namespaceResolvingScheduledTaskClient) rejectAuthShapedPreDeleteConfirmRead(
	ctx context.Context,
	resource *loganalyticsv1beta1.ScheduledTask,
) error {
	currentID := scheduledTaskCurrentID(resource)
	if currentID == "" || c.initErr != nil || c.client == nil {
		return nil
	}
	_, err := c.client.GetScheduledTask(ctx, loganalyticssdk.GetScheduledTaskRequest{
		NamespaceName:   common.String(resource.Namespace),
		ScheduledTaskId: common.String(currentID),
	})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("ScheduledTask delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound before delete; refusing to send OCI delete until identity is confirmed: %v", err)
}

func scheduledTaskCurrentID(resource *loganalyticsv1beta1.ScheduledTask) string {
	if resource == nil {
		return ""
	}
	if currentID := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); currentID != "" {
		return currentID
	}
	return strings.TrimSpace(resource.Status.Id)
}

func scheduledTaskNamespaceFromJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var values map[string]any
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return ""
	}
	for _, key := range []string{"namespaceName", "namespace"} {
		if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func scheduledTaskNamespaceFromCollection(compartmentID string, collection loganalyticssdk.NamespaceCollection) (string, error) {
	items := collection.Items
	if len(items) == 0 {
		return "", fmt.Errorf("OCI returned no Log Analytics namespace for compartment %q", compartmentID)
	}
	if len(items) > 1 {
		matches := make([]loganalyticssdk.NamespaceSummary, 0, len(items))
		for _, item := range items {
			if stringValue(item.CompartmentId) == compartmentID {
				matches = append(matches, item)
			}
		}
		items = matches
	}
	if len(items) != 1 {
		return "", fmt.Errorf("OCI returned %d Log Analytics namespaces for compartment %q", len(items), compartmentID)
	}
	item := items[0]
	namespace := strings.TrimSpace(stringValue(item.NamespaceName))
	if namespace == "" {
		return "", fmt.Errorf("OCI returned an empty Log Analytics namespace")
	}
	if item.IsOnboarded != nil && !*item.IsOnboarded {
		return "", fmt.Errorf("namespace %q is not onboarded for Log Analytics", namespace)
	}
	return namespace, nil
}

func listScheduledTaskPages(
	ctx context.Context,
	client scheduledTaskOCIClient,
	initErr error,
	request loganalyticssdk.ListScheduledTasksRequest,
) (loganalyticssdk.ListScheduledTasksResponse, error) {
	if initErr != nil {
		return loganalyticssdk.ListScheduledTasksResponse{}, fmt.Errorf("initialize ScheduledTask OCI client: %w", initErr)
	}
	if client == nil {
		return loganalyticssdk.ListScheduledTasksResponse{}, fmt.Errorf("ScheduledTask OCI client is not configured")
	}

	var merged loganalyticssdk.ListScheduledTasksResponse
	for {
		response, err := client.ListScheduledTasks(ctx, request)
		if err != nil {
			return response, err
		}
		if merged.RawResponse == nil {
			merged.RawResponse = response.RawResponse
		}
		if merged.OpcRequestId == nil {
			merged.OpcRequestId = response.OpcRequestId
		}
		merged.Items = append(merged.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			merged.OpcNextPage = response.OpcNextPage
			merged.OpcPrevPage = response.OpcPrevPage
			return merged, nil
		}
		request.Page = response.OpcNextPage
	}
}

func buildScheduledTaskCreateBody(resource *loganalyticsv1beta1.ScheduledTask) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("ScheduledTask resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return nil, fmt.Errorf("spec.compartmentId is required")
	}
	kind := scheduledTaskKind(resource)
	switch kind {
	case scheduledTaskKindAcceleration:
		if strings.TrimSpace(resource.Spec.SavedSearchId) == "" {
			return nil, fmt.Errorf("spec.savedSearchId is required when ScheduledTask kind is ACCELERATION")
		}
		if strings.TrimSpace(resource.Spec.DisplayName) == "" {
			return nil, fmt.Errorf("spec.displayName is required when ScheduledTask kind is ACCELERATION")
		}
		return loganalyticssdk.CreateAccelerationTaskDetails{
			CompartmentId: common.String(resource.Spec.CompartmentId),
			SavedSearchId: common.String(resource.Spec.SavedSearchId),
			DisplayName:   optionalString(resource.Spec.DisplayName),
			Description:   optionalString(resource.Spec.Description),
			FreeformTags:  maps.Clone(resource.Spec.FreeformTags),
			DefinedTags:   definedTagsToSDK(resource.Spec.DefinedTags),
		}, nil
	case scheduledTaskKindStandard:
		taskType, err := scheduledTaskTaskType(resource)
		if err != nil {
			return nil, err
		}
		action, err := scheduledTaskActionToSDK(resource.Spec.Action, taskType)
		if err != nil {
			return nil, err
		}
		return loganalyticssdk.CreateStandardTaskDetails{
			CompartmentId: common.String(resource.Spec.CompartmentId),
			Action:        action,
			DisplayName:   optionalString(resource.Spec.DisplayName),
			Description:   optionalString(resource.Spec.Description),
			FreeformTags:  maps.Clone(resource.Spec.FreeformTags),
			DefinedTags:   definedTagsToSDK(resource.Spec.DefinedTags),
			Schedules:     scheduledTaskSchedulesToSDK(resource.Spec.Schedules),
			TaskType:      taskType,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported ScheduledTask kind %q", kind)
	}
}

func buildScheduledTaskUpdateBody(
	resource *loganalyticsv1beta1.ScheduledTask,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("ScheduledTask resource is nil")
	}
	current, _ := scheduledTaskStatusFromResponse(currentResponse)
	if !scheduledTaskMutableDrift(resource, current) {
		return nil, false, nil
	}

	kind := scheduledTaskKind(resource)
	switch kind {
	case scheduledTaskKindAcceleration:
		return updateAccelerationTaskDetails{
			DisplayName:  optionalString(resource.Spec.DisplayName),
			Description:  optionalString(resource.Spec.Description),
			FreeformTags: maps.Clone(resource.Spec.FreeformTags),
			DefinedTags:  definedTagsToSDK(resource.Spec.DefinedTags),
		}, true, nil
	case scheduledTaskKindStandard:
		taskType, err := scheduledTaskTaskType(resource)
		if err != nil {
			return nil, false, err
		}
		action, err := optionalScheduledTaskActionToSDK(resource.Spec.Action, taskType)
		if err != nil {
			return nil, false, err
		}
		return loganalyticssdk.UpdateStandardTaskDetails{
			DisplayName:  optionalString(resource.Spec.DisplayName),
			Description:  optionalString(resource.Spec.Description),
			FreeformTags: maps.Clone(resource.Spec.FreeformTags),
			DefinedTags:  definedTagsToSDK(resource.Spec.DefinedTags),
			Schedules:    scheduledTaskSchedulesToSDK(resource.Spec.Schedules),
			Action:       action,
		}, true, nil
	default:
		return nil, false, fmt.Errorf("unsupported ScheduledTask kind %q", kind)
	}
}

func scheduledTaskKind(resource *loganalyticsv1beta1.ScheduledTask) string {
	if resource == nil {
		return ""
	}
	kind := strings.ToUpper(strings.TrimSpace(resource.Spec.Kind))
	if kind != "" {
		return kind
	}
	if strings.EqualFold(resource.Spec.TaskType, string(loganalyticssdk.TaskTypeAcceleration)) ||
		strings.TrimSpace(resource.Spec.SavedSearchId) != "" {
		return scheduledTaskKindAcceleration
	}
	return scheduledTaskKindStandard
}

func scheduledTaskTaskType(resource *loganalyticsv1beta1.ScheduledTask) (loganalyticssdk.TaskTypeEnum, error) {
	if resource == nil {
		return "", fmt.Errorf("ScheduledTask resource is nil")
	}
	taskType := strings.TrimSpace(resource.Spec.TaskType)
	if taskType == "" && scheduledTaskKind(resource) == scheduledTaskKindAcceleration {
		return loganalyticssdk.TaskTypeAcceleration, nil
	}
	if taskType == "" {
		return "", fmt.Errorf("spec.taskType is required when ScheduledTask kind is STANDARD")
	}
	if mapped, ok := loganalyticssdk.GetMappingTaskTypeEnum(taskType); ok {
		return mapped, nil
	}
	return "", fmt.Errorf("unsupported spec.taskType %q", taskType)
}

func scheduledTaskActionToSDK(
	action loganalyticsv1beta1.ScheduledTaskAction,
	taskType loganalyticssdk.TaskTypeEnum,
) (loganalyticssdk.Action, error) {
	sdkAction, err := optionalScheduledTaskActionToSDK(action, taskType)
	if err != nil {
		return nil, err
	}
	if sdkAction == nil {
		return nil, fmt.Errorf("spec.action is required when ScheduledTask kind is STANDARD")
	}
	return sdkAction, nil
}

func optionalScheduledTaskActionToSDK(
	action loganalyticsv1beta1.ScheduledTaskAction,
	taskType loganalyticssdk.TaskTypeEnum,
) (loganalyticssdk.Action, error) {
	actionType := strings.ToUpper(strings.TrimSpace(action.Type))
	if actionType == "" {
		switch taskType {
		case loganalyticssdk.TaskTypePurge:
			actionType = string(loganalyticssdk.ActionTypePurge)
		case loganalyticssdk.TaskTypeSavedSearch:
			actionType = string(loganalyticssdk.ActionTypeStream)
		}
	}

	switch actionType {
	case "":
		return nil, nil
	case string(loganalyticssdk.ActionTypePurge):
		return loganalyticssdk.PurgeAction{
			QueryString:            common.String(action.QueryString),
			PurgeDuration:          common.String(action.PurgeDuration),
			PurgeCompartmentId:     common.String(action.PurgeCompartmentId),
			CompartmentIdInSubtree: common.Bool(action.CompartmentIdInSubtree),
			DataType:               loganalyticssdk.StorageDataTypeEnum(action.DataType),
		}, nil
	case string(loganalyticssdk.ActionTypeStream):
		return loganalyticssdk.StreamAction{
			SavedSearchId:       optionalString(action.SavedSearchId),
			TemplateDetails:     scheduledTaskTemplateDetailsToSDK(action.TemplateDetails),
			MetricExtraction:    scheduledTaskMetricExtractionToSDK(action.MetricExtraction),
			SavedSearchDuration: optionalString(action.SavedSearchDuration),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported spec.action.type %q", action.Type)
	}
}

func scheduledTaskSchedulesToSDK(schedules []loganalyticsv1beta1.ScheduledTaskSchedule) []loganalyticssdk.Schedule {
	if len(schedules) == 0 {
		return nil
	}
	out := make([]loganalyticssdk.Schedule, 0, len(schedules))
	for _, schedule := range schedules {
		out = append(out, scheduledTaskScheduleToSDK(schedule))
	}
	return out
}

func scheduledTaskScheduleToSDK(schedule loganalyticsv1beta1.ScheduledTaskSchedule) loganalyticssdk.Schedule {
	scheduleType := strings.ToUpper(strings.TrimSpace(schedule.Type))
	if scheduleType == "" {
		switch {
		case strings.TrimSpace(schedule.Expression) != "":
			scheduleType = string(loganalyticssdk.ScheduleTypeCron)
		case strings.TrimSpace(schedule.RecurringInterval) != "" || schedule.RepeatCount != 0:
			scheduleType = string(loganalyticssdk.ScheduleTypeFixedFrequency)
		default:
			scheduleType = string(loganalyticssdk.ScheduleTypeAuto)
		}
	}

	base := scheduledTaskScheduleBase(schedule)
	switch scheduleType {
	case string(loganalyticssdk.ScheduleTypeCron):
		return loganalyticssdk.CronSchedule{
			TimeOfFirstExecution: base.timeOfFirstExecution,
			QueryOffsetSecs:      base.queryOffsetSecs,
			TimeEnd:              base.timeEnd,
			Expression:           optionalString(schedule.Expression),
			TimeZone:             optionalString(schedule.TimeZone),
			MisfirePolicy:        base.misfirePolicy,
		}
	case string(loganalyticssdk.ScheduleTypeFixedFrequency):
		return loganalyticssdk.FixedFrequencySchedule{
			TimeOfFirstExecution: base.timeOfFirstExecution,
			QueryOffsetSecs:      base.queryOffsetSecs,
			TimeEnd:              base.timeEnd,
			RecurringInterval:    optionalString(schedule.RecurringInterval),
			RepeatCount:          common.Int(schedule.RepeatCount),
			MisfirePolicy:        base.misfirePolicy,
		}
	default:
		return loganalyticssdk.AutoSchedule{
			TimeOfFirstExecution: base.timeOfFirstExecution,
			QueryOffsetSecs:      base.queryOffsetSecs,
			TimeEnd:              base.timeEnd,
			MisfirePolicy:        base.misfirePolicy,
		}
	}
}

type scheduledTaskScheduleBaseValues struct {
	timeOfFirstExecution *common.SDKTime
	queryOffsetSecs      *int
	timeEnd              *common.SDKTime
	misfirePolicy        loganalyticssdk.ScheduleMisfirePolicyEnum
}

func scheduledTaskScheduleBase(schedule loganalyticsv1beta1.ScheduledTaskSchedule) scheduledTaskScheduleBaseValues {
	var queryOffsetSecs *int
	if schedule.QueryOffsetSecs != 0 {
		queryOffsetSecs = common.Int(schedule.QueryOffsetSecs)
	}
	return scheduledTaskScheduleBaseValues{
		timeOfFirstExecution: sdkTimeFromString(schedule.TimeOfFirstExecution),
		queryOffsetSecs:      queryOffsetSecs,
		timeEnd:              sdkTimeFromString(schedule.TimeEnd),
		misfirePolicy:        loganalyticssdk.ScheduleMisfirePolicyEnum(schedule.MisfirePolicy),
	}
}

func scheduledTaskTemplateDetailsToSDK(
	details loganalyticsv1beta1.ScheduledTaskActionTemplateDetails,
) *loganalyticssdk.TemplateDetails {
	if strings.TrimSpace(details.TemplateId) == "" && len(details.TemplateParams) == 0 {
		return nil
	}
	out := &loganalyticssdk.TemplateDetails{
		TemplateId: optionalString(details.TemplateId),
	}
	for _, param := range details.TemplateParams {
		out.TemplateParams = append(out.TemplateParams, loganalyticssdk.TemplateParams{
			KeyField:   common.String(param.KeyField),
			ValueField: common.String(param.ValueField),
		})
	}
	return out
}

func scheduledTaskMetricExtractionToSDK(
	metric loganalyticsv1beta1.ScheduledTaskActionMetricExtraction,
) *loganalyticssdk.MetricExtraction {
	if strings.TrimSpace(metric.CompartmentId) == "" &&
		strings.TrimSpace(metric.Namespace) == "" &&
		strings.TrimSpace(metric.MetricName) == "" &&
		strings.TrimSpace(metric.ResourceGroup) == "" &&
		len(metric.MetricCollections) == 0 {
		return nil
	}
	out := &loganalyticssdk.MetricExtraction{
		CompartmentId:     optionalString(metric.CompartmentId),
		Namespace:         optionalString(metric.Namespace),
		MetricName:        optionalString(metric.MetricName),
		ResourceGroup:     optionalString(metric.ResourceGroup),
		MetricCollections: scheduledTaskMetricCollectionsToSDK(metric.MetricCollections),
	}
	return out
}

func scheduledTaskMetricCollectionsToSDK(
	collections []loganalyticsv1beta1.ScheduledTaskActionMetricExtractionMetricCollection,
) []loganalyticssdk.MetricCollection {
	if len(collections) == 0 {
		return nil
	}
	out := make([]loganalyticssdk.MetricCollection, 0, len(collections))
	for _, collection := range collections {
		out = append(out, loganalyticssdk.MetricCollection{
			MetricName:           common.String(collection.MetricName),
			MetricQueryFieldName: common.String(collection.MetricQueryFieldName),
			Dimensions:           scheduledTaskDimensionsToSDK(collection.Dimensions),
			QueryTableName:       optionalString(collection.QueryTableName),
		})
	}
	return out
}

func scheduledTaskDimensionsToSDK(
	dimensions []loganalyticsv1beta1.ScheduledTaskActionMetricExtractionMetricCollectionDimension,
) []loganalyticssdk.DimensionField {
	if len(dimensions) == 0 {
		return nil
	}
	out := make([]loganalyticssdk.DimensionField, 0, len(dimensions))
	for _, dimension := range dimensions {
		out = append(out, loganalyticssdk.DimensionField{
			QueryFieldName: common.String(dimension.QueryFieldName),
			DimensionName:  optionalString(dimension.DimensionName),
		})
	}
	return out
}

func validateScheduledTaskCreateOnlyDrift(resource *loganalyticsv1beta1.ScheduledTask, currentResponse any) error {
	current, ok := scheduledTaskStatusFromResponse(currentResponse)
	if !ok {
		return nil
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) != "" && strings.TrimSpace(current.CompartmentId) != "" &&
		resource.Spec.CompartmentId != current.CompartmentId {
		return fmt.Errorf("ScheduledTask formal semantics require replacement when compartmentId changes")
	}
	if desiredTaskType, err := scheduledTaskTaskType(resource); err == nil && current.TaskType != "" &&
		string(desiredTaskType) != current.TaskType {
		return fmt.Errorf("ScheduledTask formal semantics require replacement when taskType changes")
	}
	if err := validateScheduledTaskSavedSearchDrift(resource); err != nil {
		return err
	}
	return nil
}

func validateScheduledTaskSavedSearchDrift(resource *loganalyticsv1beta1.ScheduledTask) error {
	if scheduledTaskKind(resource) != scheduledTaskKindAcceleration {
		return nil
	}
	desired := strings.TrimSpace(resource.Spec.SavedSearchId)
	if desired == "" {
		return nil
	}
	previous := scheduledTaskSavedSearchIDFromJSON(resource.Status.JsonData)
	if previous != "" && previous != desired {
		return fmt.Errorf("ScheduledTask formal semantics require replacement when savedSearchId changes")
	}
	return nil
}

func scheduledTaskMutableDrift(resource *loganalyticsv1beta1.ScheduledTask, current loganalyticsv1beta1.ScheduledTaskStatus) bool {
	if resource == nil {
		return false
	}
	spec := resource.Spec
	return scheduledTaskScalarMutableDrift(spec, current) ||
		scheduledTaskScheduleMutableDrift(spec, current) ||
		scheduledTaskActionMutableDrift(resource, spec, current)
}

func scheduledTaskScalarMutableDrift(
	spec loganalyticsv1beta1.ScheduledTaskSpec,
	current loganalyticsv1beta1.ScheduledTaskStatus,
) bool {
	return desiredStringDiffers(spec.DisplayName, current.DisplayName) ||
		desiredStringDiffers(spec.Description, current.Description) ||
		(spec.FreeformTags != nil && !reflect.DeepEqual(spec.FreeformTags, current.FreeformTags)) ||
		(spec.DefinedTags != nil && !reflect.DeepEqual(spec.DefinedTags, current.DefinedTags))
}

func desiredStringDiffers(desired string, current string) bool {
	return strings.TrimSpace(desired) != "" && desired != current
}

func scheduledTaskScheduleMutableDrift(
	spec loganalyticsv1beta1.ScheduledTaskSpec,
	current loganalyticsv1beta1.ScheduledTaskStatus,
) bool {
	return len(spec.Schedules) > 0 &&
		!jsonEqual(scheduledTaskSchedulesToSDK(spec.Schedules), currentJSONField(current.JsonData, "schedules"))
}

func scheduledTaskActionMutableDrift(
	resource *loganalyticsv1beta1.ScheduledTask,
	spec loganalyticsv1beta1.ScheduledTaskSpec,
	current loganalyticsv1beta1.ScheduledTaskStatus,
) bool {
	if !scheduledTaskActionMeaningful(spec.Action) {
		return false
	}
	taskType, err := scheduledTaskTaskType(resource)
	if err != nil {
		return true
	}
	desiredAction, err := optionalScheduledTaskActionToSDK(spec.Action, taskType)
	if err != nil {
		return true
	}
	return !jsonEqual(desiredAction, currentJSONField(current.JsonData, "action"))
}

func scheduledTaskActionMeaningful(action loganalyticsv1beta1.ScheduledTaskAction) bool {
	return anyStringMeaningful(
		action.Type,
		action.QueryString,
		action.PurgeDuration,
		action.PurgeCompartmentId,
		action.DataType,
		action.SavedSearchId,
		action.SavedSearchDuration,
	) ||
		action.CompartmentIdInSubtree ||
		scheduledTaskTemplateDetailsMeaningful(action.TemplateDetails) ||
		scheduledTaskMetricExtractionMeaningful(action.MetricExtraction)
}

func anyStringMeaningful(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func scheduledTaskTemplateDetailsMeaningful(details loganalyticsv1beta1.ScheduledTaskActionTemplateDetails) bool {
	return strings.TrimSpace(details.TemplateId) != "" || len(details.TemplateParams) > 0
}

func scheduledTaskMetricExtractionMeaningful(metric loganalyticsv1beta1.ScheduledTaskActionMetricExtraction) bool {
	return anyStringMeaningful(metric.CompartmentId, metric.Namespace, metric.MetricName, metric.ResourceGroup) ||
		len(metric.MetricCollections) > 0
}

func projectScheduledTaskStatusFromResponse(resource *loganalyticsv1beta1.ScheduledTask, response any) error {
	projected, ok := scheduledTaskStatusFromResponse(response)
	if !ok {
		return nil
	}
	if err := preserveScheduledTaskWriteOnlyFields(resource, &projected); err != nil {
		return err
	}
	osokStatus := resource.Status.OsokStatus
	resource.Status = projected
	resource.Status.OsokStatus = osokStatus
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
	return nil
}

func preserveScheduledTaskWriteOnlyFields(
	resource *loganalyticsv1beta1.ScheduledTask,
	status *loganalyticsv1beta1.ScheduledTaskStatus,
) error {
	if resource == nil || status == nil || scheduledTaskKind(resource) != scheduledTaskKindAcceleration {
		return nil
	}
	savedSearchID := strings.TrimSpace(resource.Spec.SavedSearchId)
	if savedSearchID == "" {
		savedSearchID = scheduledTaskSavedSearchIDFromJSON(resource.Status.JsonData)
	}
	if savedSearchID == "" {
		return nil
	}

	payload, err := scheduledTaskJSONWithField(status.JsonData, scheduledTaskJSONSavedSearchID, savedSearchID)
	if err != nil {
		return fmt.Errorf("preserve ScheduledTask savedSearchId in status jsonData: %w", err)
	}
	status.JsonData = payload
	return nil
}

func scheduledTaskStatusFromResponse(response any) (loganalyticsv1beta1.ScheduledTaskStatus, bool) {
	body, ok := scheduledTaskResponseBody(response)
	if !ok || body == nil {
		return loganalyticsv1beta1.ScheduledTaskStatus{}, false
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return loganalyticsv1beta1.ScheduledTaskStatus{}, false
	}
	var status loganalyticsv1beta1.ScheduledTaskStatus
	if err := json.Unmarshal(payload, &status); err != nil {
		return loganalyticsv1beta1.ScheduledTaskStatus{}, false
	}
	status.JsonData = string(payload)
	if status.Id != "" {
		status.OsokStatus.Ocid = shared.OCID(status.Id)
	}
	return status, true
}

func scheduledTaskResponseBody(response any) (any, bool) {
	switch r := response.(type) {
	case loganalyticssdk.CreateScheduledTaskResponse:
		return scheduledTaskModelBody(r.ScheduledTask)
	case loganalyticssdk.GetScheduledTaskResponse:
		return scheduledTaskModelBody(r.ScheduledTask)
	case loganalyticssdk.UpdateScheduledTaskResponse:
		return scheduledTaskModelBody(r.ScheduledTask)
	default:
		return scheduledTaskModelBody(response)
	}
}

func scheduledTaskModelBody(model any) (any, bool) {
	switch r := model.(type) {
	case nil:
		return nil, false
	case *loganalyticssdk.StandardTask:
		return pointerBody(r)
	case loganalyticssdk.StandardTask:
		return r, true
	case *loganalyticssdk.ScheduledTaskSummary:
		return pointerBody(r)
	case loganalyticssdk.ScheduledTaskSummary:
		return r, true
	case loganalyticssdk.ScheduledTask:
		return r, r != nil
	default:
		return model, true
	}
}

func pointerBody[T any](value *T) (any, bool) {
	if value == nil {
		return nil, false
	}
	return *value, true
}

func handleScheduledTaskDeleteError(resource *loganalyticsv1beta1.ScheduledTask, err error) error {
	classification := errorutil.ClassifyDeleteError(err)
	if classification.IsAuthShapedNotFound() {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return fmt.Errorf("ScheduledTask delete returned ambiguous 404 %s; retaining finalizer until OCI identity can be confirmed: %v", classification.ErrorCodeString(), err)
	}
	return err
}

func (m updateAccelerationTaskDetails) GetDisplayName() *string {
	return m.DisplayName
}

func (m updateAccelerationTaskDetails) GetDescription() *string {
	return m.Description
}

func (m updateAccelerationTaskDetails) GetFreeformTags() map[string]string {
	return m.FreeformTags
}

func (m updateAccelerationTaskDetails) GetDefinedTags() map[string]map[string]interface{} {
	return m.DefinedTags
}

func (m updateAccelerationTaskDetails) GetSchedules() []loganalyticssdk.Schedule {
	return nil
}

func (m updateAccelerationTaskDetails) MarshalJSON() ([]byte, error) {
	type alias updateAccelerationTaskDetails
	return json.Marshal(struct {
		Kind string `json:"kind"`
		alias
	}{
		Kind:  scheduledTaskKindAcceleration,
		alias: alias(m),
	})
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return common.String(value)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func sdkTimeFromString(value string) *common.SDKTime {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil
	}
	return &common.SDKTime{Time: parsed}
}

func definedTagsToSDK(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	out := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		out[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			out[namespace][key] = value
		}
	}
	return out
}

func scheduledTaskSavedSearchIDFromJSON(raw string) string {
	value, ok := currentJSONField(raw, scheduledTaskJSONSavedSearchID).(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func scheduledTaskJSONWithField(raw string, field string, value string) (string, error) {
	values := map[string]any{}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &values); err != nil {
			return "", err
		}
	}
	values[field] = value
	payload, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func currentJSONField(raw string, field string) any {
	var values map[string]any
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}
	return values[field]
}

func jsonEqual(a any, b any) bool {
	normalizedA, okA := normalizedJSONValue(a)
	normalizedB, okB := normalizedJSONValue(b)
	if !okA || !okB {
		return false
	}
	return reflect.DeepEqual(normalizedA, normalizedB)
}

func normalizedJSONValue(value any) (any, bool) {
	if value == nil {
		return nil, true
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, false
	}
	return decoded, true
}
