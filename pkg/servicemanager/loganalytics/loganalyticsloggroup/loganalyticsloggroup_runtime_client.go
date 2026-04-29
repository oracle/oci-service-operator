/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package loganalyticsloggroup

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	loganalyticssdk "github.com/oracle/oci-go-sdk/v65/loganalytics"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const logAnalyticsLogGroupKind = "LogAnalyticsLogGroup"

type logAnalyticsLogGroupOCIClient interface {
	CreateLogAnalyticsLogGroup(context.Context, loganalyticssdk.CreateLogAnalyticsLogGroupRequest) (loganalyticssdk.CreateLogAnalyticsLogGroupResponse, error)
	GetLogAnalyticsLogGroup(context.Context, loganalyticssdk.GetLogAnalyticsLogGroupRequest) (loganalyticssdk.GetLogAnalyticsLogGroupResponse, error)
	ListLogAnalyticsLogGroups(context.Context, loganalyticssdk.ListLogAnalyticsLogGroupsRequest) (loganalyticssdk.ListLogAnalyticsLogGroupsResponse, error)
	UpdateLogAnalyticsLogGroup(context.Context, loganalyticssdk.UpdateLogAnalyticsLogGroupRequest) (loganalyticssdk.UpdateLogAnalyticsLogGroupResponse, error)
	DeleteLogAnalyticsLogGroup(context.Context, loganalyticssdk.DeleteLogAnalyticsLogGroupRequest) (loganalyticssdk.DeleteLogAnalyticsLogGroupResponse, error)
}

func init() {
	registerLogAnalyticsLogGroupRuntimeHooksMutator(func(_ *LogAnalyticsLogGroupServiceManager, hooks *LogAnalyticsLogGroupRuntimeHooks) {
		applyLogAnalyticsLogGroupRuntimeHooks(hooks)
	})
}

func applyLogAnalyticsLogGroupRuntimeHooks(hooks *LogAnalyticsLogGroupRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newLogAnalyticsLogGroupRuntimeSemantics()
	hooks.BuildCreateBody = buildLogAnalyticsLogGroupCreateBody
	hooks.BuildUpdateBody = buildLogAnalyticsLogGroupUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardLogAnalyticsLogGroupExistingBeforeCreate
	hooks.List.Fields = logAnalyticsLogGroupListFields()
	hooks.List.Call = paginatedLogAnalyticsLogGroupListCall(hooks.List.Call)
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedLogAnalyticsLogGroupIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateLogAnalyticsLogGroupCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleLogAnalyticsLogGroupDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyLogAnalyticsLogGroupDeleteOutcome
	if hooks.Get.Call != nil {
		get := hooks.Get.Call
		hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate LogAnalyticsLogGroupServiceClient) LogAnalyticsLogGroupServiceClient {
			return logAnalyticsLogGroupDeleteGuardClient{delegate: delegate, get: get}
		})
	}
}

func newLogAnalyticsLogGroupRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "loganalytics",
		FormalSlug:        "loganalyticsloggroup",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"displayName", "description", "freeformTags", "definedTags"},
			Mutable:         []string{"displayName", "description", "freeformTags", "definedTags"},
			ForceNew:        []string{"compartmentId"},
			ConflictsWith:   map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "read-after-write", Hooks: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}}},
		UpdateFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "read-after-write", Hooks: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}}},
		DeleteFollowUp:      generatedruntime.FollowUpSemantics{Strategy: "confirm-delete", Hooks: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}}},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func logAnalyticsLogGroupCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path"},
		{FieldName: "CreateLogAnalyticsLogGroupDetails", RequestName: "CreateLogAnalyticsLogGroupDetails", Contribution: "body"},
	}
}

func logAnalyticsLogGroupGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path"},
		{FieldName: "LogAnalyticsLogGroupId", RequestName: "logAnalyticsLogGroupId", Contribution: "path", PreferResourceID: true},
	}
}

func logAnalyticsLogGroupListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path"},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func logAnalyticsLogGroupUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path"},
		{FieldName: "LogAnalyticsLogGroupId", RequestName: "logAnalyticsLogGroupId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateLogAnalyticsLogGroupDetails", RequestName: "UpdateLogAnalyticsLogGroupDetails", Contribution: "body"},
	}
}

func logAnalyticsLogGroupDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "NamespaceName", RequestName: "namespaceName", Contribution: "path"},
		{FieldName: "LogAnalyticsLogGroupId", RequestName: "logAnalyticsLogGroupId", Contribution: "path", PreferResourceID: true},
	}
}

func guardLogAnalyticsLogGroupExistingBeforeCreate(
	_ context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsLogGroup,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("%s resource is nil", logAnalyticsLogGroupKind)
	}
	if strings.TrimSpace(resource.Namespace) == "" ||
		strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildLogAnalyticsLogGroupCreateBody(
	_ context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsLogGroup,
	namespace string,
) (any, error) {
	if err := validateLogAnalyticsLogGroupSpec(resource, namespace); err != nil {
		return nil, err
	}

	spec := resource.Spec
	details := loganalyticssdk.CreateLogAnalyticsLogGroupDetails{
		DisplayName:   common.String(strings.TrimSpace(spec.DisplayName)),
		CompartmentId: common.String(strings.TrimSpace(spec.CompartmentId)),
	}
	if spec.Description != "" {
		details.Description = common.String(spec.Description)
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = logAnalyticsLogGroupDefinedTags(spec.DefinedTags)
	}
	return details, nil
}

func buildLogAnalyticsLogGroupUpdateBody(
	_ context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsLogGroup,
	namespace string,
	currentResponse any,
) (any, bool, error) {
	if err := validateLogAnalyticsLogGroupSpec(resource, namespace); err != nil {
		return loganalyticssdk.UpdateLogAnalyticsLogGroupDetails{}, false, err
	}
	current, ok := logAnalyticsLogGroupFromResponse(currentResponse)
	if !ok {
		return loganalyticssdk.UpdateLogAnalyticsLogGroupDetails{}, false, fmt.Errorf("current %s response does not expose a log group body", logAnalyticsLogGroupKind)
	}
	if err := validateLogAnalyticsLogGroupCreateOnlyDrift(resource.Spec, current); err != nil {
		return loganalyticssdk.UpdateLogAnalyticsLogGroupDetails{}, false, err
	}

	details := loganalyticssdk.UpdateLogAnalyticsLogGroupDetails{}
	updateNeeded := false
	if desired, ok := logAnalyticsLogGroupDisplayNameUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := logAnalyticsLogGroupOptionalStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := logAnalyticsLogGroupFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := logAnalyticsLogGroupDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}
	return details, updateNeeded, nil
}

func validateLogAnalyticsLogGroupSpec(resource *loganalyticsv1beta1.LogAnalyticsLogGroup, namespace string) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", logAnalyticsLogGroupKind)
	}
	var problems []string
	if strings.TrimSpace(namespace) == "" {
		problems = append(problems, "namespaceName is required")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		problems = append(problems, "displayName is required")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		problems = append(problems, "compartmentId is required")
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%s spec is invalid: %s", logAnalyticsLogGroupKind, strings.Join(problems, "; "))
}

func validateLogAnalyticsLogGroupCreateOnlyDriftForResponse(
	resource *loganalyticsv1beta1.LogAnalyticsLogGroup,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", logAnalyticsLogGroupKind)
	}
	current, ok := logAnalyticsLogGroupFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current %s response does not expose a log group body", logAnalyticsLogGroupKind)
	}
	return validateLogAnalyticsLogGroupCreateOnlyDrift(resource.Spec, current)
}

func validateLogAnalyticsLogGroupCreateOnlyDrift(
	spec loganalyticsv1beta1.LogAnalyticsLogGroupSpec,
	current loganalyticssdk.LogAnalyticsLogGroup,
) error {
	if logAnalyticsLogGroupStringValue(current.CompartmentId) == strings.TrimSpace(spec.CompartmentId) {
		return nil
	}
	return fmt.Errorf("%s create-only field drift is not supported: compartmentId", logAnalyticsLogGroupKind)
}

func logAnalyticsLogGroupDisplayNameUpdate(spec string, current *string) (*string, bool) {
	desired := strings.TrimSpace(spec)
	if desired == "" || desired == logAnalyticsLogGroupStringValue(current) {
		return nil, false
	}
	return common.String(desired), true
}

func logAnalyticsLogGroupOptionalStringUpdate(spec string, current *string) (*string, bool) {
	currentValue := ""
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	if spec == "" && current == nil {
		return nil, false
	}
	return common.String(spec), true
}

func logAnalyticsLogGroupFreeformTagsUpdate(spec map[string]string, current map[string]string) (map[string]string, bool) {
	if spec == nil || maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func logAnalyticsLogGroupDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := logAnalyticsLogGroupDefinedTags(spec)
	if reflect.DeepEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func logAnalyticsLogGroupDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	return *util.ConvertToOciDefinedTags(&tags)
}

func paginatedLogAnalyticsLogGroupListCall(
	call func(context.Context, loganalyticssdk.ListLogAnalyticsLogGroupsRequest) (loganalyticssdk.ListLogAnalyticsLogGroupsResponse, error),
) func(context.Context, loganalyticssdk.ListLogAnalyticsLogGroupsRequest) (loganalyticssdk.ListLogAnalyticsLogGroupsResponse, error) {
	if call == nil {
		return nil
	}
	return func(ctx context.Context, request loganalyticssdk.ListLogAnalyticsLogGroupsRequest) (loganalyticssdk.ListLogAnalyticsLogGroupsResponse, error) {
		var combined loganalyticssdk.ListLogAnalyticsLogGroupsResponse
		nextPage := request.Page
		for {
			response, err := call(ctx, logAnalyticsLogGroupListPageRequest(request, nextPage))
			if err != nil {
				return response, err
			}
			mergeLogAnalyticsLogGroupListPage(&combined, response)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				return combined, nil
			}
			nextPage = response.OpcNextPage
		}
	}
}

func logAnalyticsLogGroupListPageRequest(
	request loganalyticssdk.ListLogAnalyticsLogGroupsRequest,
	nextPage *string,
) loganalyticssdk.ListLogAnalyticsLogGroupsRequest {
	pageRequest := request
	pageRequest.Page = nextPage
	return pageRequest
}

func mergeLogAnalyticsLogGroupListPage(
	combined *loganalyticssdk.ListLogAnalyticsLogGroupsResponse,
	response loganalyticssdk.ListLogAnalyticsLogGroupsResponse,
) {
	if combined.RawResponse == nil {
		combined.RawResponse = response.RawResponse
	}
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	combined.Items = append(combined.Items, response.Items...)
}

func handleLogAnalyticsLogGroupDeleteError(resource *loganalyticsv1beta1.LogAnalyticsLogGroup, err error) error {
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
	return fmt.Errorf("%s delete returned ambiguous %s %s; retaining finalizer",
		logAnalyticsLogGroupKind,
		classification.HTTPStatusCodeString(),
		classification.ErrorCodeString())
}

func applyLogAnalyticsLogGroupDeleteOutcome(
	resource *loganalyticsv1beta1.LogAnalyticsLogGroup,
	_ any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if stage != generatedruntime.DeleteConfirmStageAlreadyPending || !logAnalyticsLogGroupDeleteIsPending(resource) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	markLogAnalyticsLogGroupTerminating(resource, "OCI resource delete is in progress")
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func logAnalyticsLogGroupDeleteIsPending(resource *loganalyticsv1beta1.LogAnalyticsLogGroup) bool {
	return resource != nil &&
		resource.Status.OsokStatus.Async.Current != nil &&
		resource.Status.OsokStatus.Async.Current.Phase == shared.OSOKAsyncPhaseDelete
}

func clearTrackedLogAnalyticsLogGroupIdentity(resource *loganalyticsv1beta1.LogAnalyticsLogGroup) {
	if resource == nil {
		return
	}
	resource.Status = loganalyticsv1beta1.LogAnalyticsLogGroupStatus{}
}

func logAnalyticsLogGroupFromResponse(response any) (loganalyticssdk.LogAnalyticsLogGroup, bool) {
	if current, ok := logAnalyticsLogGroupFromDirectResponse(response); ok {
		return current, true
	}
	return logAnalyticsLogGroupFromOperationResponse(response)
}

func logAnalyticsLogGroupFromDirectResponse(response any) (loganalyticssdk.LogAnalyticsLogGroup, bool) {
	switch current := response.(type) {
	case loganalyticssdk.LogAnalyticsLogGroup:
		return current, true
	case *loganalyticssdk.LogAnalyticsLogGroup:
		return logAnalyticsLogGroupFromOptional(current, func(group loganalyticssdk.LogAnalyticsLogGroup) loganalyticssdk.LogAnalyticsLogGroup {
			return group
		})
	case loganalyticssdk.LogAnalyticsLogGroupSummary:
		return logAnalyticsLogGroupFromSummary(current), true
	case *loganalyticssdk.LogAnalyticsLogGroupSummary:
		return logAnalyticsLogGroupFromOptional(current, logAnalyticsLogGroupFromSummary)
	default:
		return loganalyticssdk.LogAnalyticsLogGroup{}, false
	}
}

func logAnalyticsLogGroupFromOperationResponse(response any) (loganalyticssdk.LogAnalyticsLogGroup, bool) {
	switch current := response.(type) {
	case loganalyticssdk.CreateLogAnalyticsLogGroupResponse:
		return current.LogAnalyticsLogGroup, true
	case *loganalyticssdk.CreateLogAnalyticsLogGroupResponse:
		return logAnalyticsLogGroupFromOptional(current, func(response loganalyticssdk.CreateLogAnalyticsLogGroupResponse) loganalyticssdk.LogAnalyticsLogGroup {
			return response.LogAnalyticsLogGroup
		})
	case loganalyticssdk.GetLogAnalyticsLogGroupResponse:
		return current.LogAnalyticsLogGroup, true
	case *loganalyticssdk.GetLogAnalyticsLogGroupResponse:
		return logAnalyticsLogGroupFromOptional(current, func(response loganalyticssdk.GetLogAnalyticsLogGroupResponse) loganalyticssdk.LogAnalyticsLogGroup {
			return response.LogAnalyticsLogGroup
		})
	case loganalyticssdk.UpdateLogAnalyticsLogGroupResponse:
		return current.LogAnalyticsLogGroup, true
	case *loganalyticssdk.UpdateLogAnalyticsLogGroupResponse:
		return logAnalyticsLogGroupFromOptional(current, func(response loganalyticssdk.UpdateLogAnalyticsLogGroupResponse) loganalyticssdk.LogAnalyticsLogGroup {
			return response.LogAnalyticsLogGroup
		})
	default:
		return loganalyticssdk.LogAnalyticsLogGroup{}, false
	}
}

func logAnalyticsLogGroupFromOptional[T any](
	current *T,
	convert func(T) loganalyticssdk.LogAnalyticsLogGroup,
) (loganalyticssdk.LogAnalyticsLogGroup, bool) {
	if current == nil {
		return loganalyticssdk.LogAnalyticsLogGroup{}, false
	}
	return convert(*current), true
}

func logAnalyticsLogGroupFromSummary(summary loganalyticssdk.LogAnalyticsLogGroupSummary) loganalyticssdk.LogAnalyticsLogGroup {
	return loganalyticssdk.LogAnalyticsLogGroup{
		Id:            summary.Id,
		CompartmentId: summary.CompartmentId,
		DisplayName:   summary.DisplayName,
		Description:   summary.Description,
		TimeCreated:   summary.TimeCreated,
		TimeUpdated:   summary.TimeUpdated,
		FreeformTags:  summary.FreeformTags,
		DefinedTags:   summary.DefinedTags,
	}
}

type logAnalyticsLogGroupDeleteGuardClient struct {
	delegate LogAnalyticsLogGroupServiceClient
	get      func(context.Context, loganalyticssdk.GetLogAnalyticsLogGroupRequest) (loganalyticssdk.GetLogAnalyticsLogGroupResponse, error)
}

func (c logAnalyticsLogGroupDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsLogGroup,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c logAnalyticsLogGroupDeleteGuardClient) Delete(
	ctx context.Context,
	resource *loganalyticsv1beta1.LogAnalyticsLogGroup,
) (bool, error) {
	currentID := logAnalyticsLogGroupTrackedID(resource)
	namespace := logAnalyticsLogGroupNamespace(resource)
	if currentID == "" || namespace == "" || c.get == nil {
		return c.delegate.Delete(ctx, resource)
	}

	_, err := c.get(ctx, loganalyticssdk.GetLogAnalyticsLogGroupRequest{
		NamespaceName:          common.String(namespace),
		LogAnalyticsLogGroupId: common.String(currentID),
	})
	if err != nil && errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return false, handleLogAnalyticsLogGroupDeleteError(resource, err)
	}
	return c.delegate.Delete(ctx, resource)
}

func logAnalyticsLogGroupTrackedID(resource *loganalyticsv1beta1.LogAnalyticsLogGroup) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func logAnalyticsLogGroupNamespace(resource *loganalyticsv1beta1.LogAnalyticsLogGroup) string {
	if resource == nil {
		return ""
	}
	return strings.TrimSpace(resource.Namespace)
}

func markLogAnalyticsLogGroupTerminating(resource *loganalyticsv1beta1.LogAnalyticsLogGroup, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func logAnalyticsLogGroupStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func newLogAnalyticsLogGroupServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client logAnalyticsLogGroupOCIClient,
) LogAnalyticsLogGroupServiceClient {
	hooks := newLogAnalyticsLogGroupRuntimeHooksWithOCIClient(client)
	applyLogAnalyticsLogGroupRuntimeHooks(&hooks)
	manager := &LogAnalyticsLogGroupServiceManager{Log: log}
	delegate := defaultLogAnalyticsLogGroupServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*loganalyticsv1beta1.LogAnalyticsLogGroup](
			buildLogAnalyticsLogGroupGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapLogAnalyticsLogGroupGeneratedClient(hooks, delegate)
}

func newLogAnalyticsLogGroupRuntimeHooksWithOCIClient(client logAnalyticsLogGroupOCIClient) LogAnalyticsLogGroupRuntimeHooks {
	return LogAnalyticsLogGroupRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*loganalyticsv1beta1.LogAnalyticsLogGroup]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*loganalyticsv1beta1.LogAnalyticsLogGroup]{},
		StatusHooks:     generatedruntime.StatusHooks[*loganalyticsv1beta1.LogAnalyticsLogGroup]{},
		ParityHooks:     generatedruntime.ParityHooks[*loganalyticsv1beta1.LogAnalyticsLogGroup]{},
		Async:           generatedruntime.AsyncHooks[*loganalyticsv1beta1.LogAnalyticsLogGroup]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*loganalyticsv1beta1.LogAnalyticsLogGroup]{},
		Create: runtimeOperationHooks[loganalyticssdk.CreateLogAnalyticsLogGroupRequest, loganalyticssdk.CreateLogAnalyticsLogGroupResponse]{
			Fields: logAnalyticsLogGroupCreateFields(),
			Call: func(ctx context.Context, request loganalyticssdk.CreateLogAnalyticsLogGroupRequest) (loganalyticssdk.CreateLogAnalyticsLogGroupResponse, error) {
				if client == nil {
					return loganalyticssdk.CreateLogAnalyticsLogGroupResponse{}, fmt.Errorf("%s OCI client is nil", logAnalyticsLogGroupKind)
				}
				return client.CreateLogAnalyticsLogGroup(ctx, request)
			},
		},
		Get: runtimeOperationHooks[loganalyticssdk.GetLogAnalyticsLogGroupRequest, loganalyticssdk.GetLogAnalyticsLogGroupResponse]{
			Fields: logAnalyticsLogGroupGetFields(),
			Call: func(ctx context.Context, request loganalyticssdk.GetLogAnalyticsLogGroupRequest) (loganalyticssdk.GetLogAnalyticsLogGroupResponse, error) {
				if client == nil {
					return loganalyticssdk.GetLogAnalyticsLogGroupResponse{}, fmt.Errorf("%s OCI client is nil", logAnalyticsLogGroupKind)
				}
				return client.GetLogAnalyticsLogGroup(ctx, request)
			},
		},
		List: runtimeOperationHooks[loganalyticssdk.ListLogAnalyticsLogGroupsRequest, loganalyticssdk.ListLogAnalyticsLogGroupsResponse]{
			Fields: logAnalyticsLogGroupListFields(),
			Call: func(ctx context.Context, request loganalyticssdk.ListLogAnalyticsLogGroupsRequest) (loganalyticssdk.ListLogAnalyticsLogGroupsResponse, error) {
				if client == nil {
					return loganalyticssdk.ListLogAnalyticsLogGroupsResponse{}, fmt.Errorf("%s OCI client is nil", logAnalyticsLogGroupKind)
				}
				return client.ListLogAnalyticsLogGroups(ctx, request)
			},
		},
		Update: runtimeOperationHooks[loganalyticssdk.UpdateLogAnalyticsLogGroupRequest, loganalyticssdk.UpdateLogAnalyticsLogGroupResponse]{
			Fields: logAnalyticsLogGroupUpdateFields(),
			Call: func(ctx context.Context, request loganalyticssdk.UpdateLogAnalyticsLogGroupRequest) (loganalyticssdk.UpdateLogAnalyticsLogGroupResponse, error) {
				if client == nil {
					return loganalyticssdk.UpdateLogAnalyticsLogGroupResponse{}, fmt.Errorf("%s OCI client is nil", logAnalyticsLogGroupKind)
				}
				return client.UpdateLogAnalyticsLogGroup(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[loganalyticssdk.DeleteLogAnalyticsLogGroupRequest, loganalyticssdk.DeleteLogAnalyticsLogGroupResponse]{
			Fields: logAnalyticsLogGroupDeleteFields(),
			Call: func(ctx context.Context, request loganalyticssdk.DeleteLogAnalyticsLogGroupRequest) (loganalyticssdk.DeleteLogAnalyticsLogGroupResponse, error) {
				if client == nil {
					return loganalyticssdk.DeleteLogAnalyticsLogGroupResponse{}, fmt.Errorf("%s OCI client is nil", logAnalyticsLogGroupKind)
				}
				return client.DeleteLogAnalyticsLogGroup(ctx, request)
			},
		},
		WrapGeneratedClient: []func(LogAnalyticsLogGroupServiceClient) LogAnalyticsLogGroupServiceClient{},
	}
}
