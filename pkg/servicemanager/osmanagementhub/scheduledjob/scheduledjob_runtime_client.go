/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package scheduledjob

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	osmanagementhubsdk "github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

type scheduledJobOCIClient interface {
	CreateScheduledJob(context.Context, osmanagementhubsdk.CreateScheduledJobRequest) (osmanagementhubsdk.CreateScheduledJobResponse, error)
	GetScheduledJob(context.Context, osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error)
	ListScheduledJobs(context.Context, osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error)
	UpdateScheduledJob(context.Context, osmanagementhubsdk.UpdateScheduledJobRequest) (osmanagementhubsdk.UpdateScheduledJobResponse, error)
	DeleteScheduledJob(context.Context, osmanagementhubsdk.DeleteScheduledJobRequest) (osmanagementhubsdk.DeleteScheduledJobResponse, error)
}

type scheduledJobRuntimeClient struct {
	delegate ScheduledJobServiceClient
	hooks    ScheduledJobRuntimeHooks
}

var _ ScheduledJobServiceClient = (*scheduledJobRuntimeClient)(nil)

type scheduledJobIdentity struct {
	CompartmentID     string
	ScheduleType      osmanagementhubsdk.ListScheduledJobsScheduleTypeEnum
	TimeNextExecution string
	OperationType     osmanagementhubsdk.ListScheduledJobsOperationTypeEnum
}

type scheduledJobServiceError struct {
	statusCode   int
	code         string
	message      string
	opcRequestID string
}

func (e scheduledJobServiceError) Error() string {
	return e.message
}

func (e scheduledJobServiceError) GetHTTPStatusCode() int {
	return e.statusCode
}

func (e scheduledJobServiceError) GetMessage() string {
	return e.message
}

func (e scheduledJobServiceError) GetCode() string {
	return e.code
}

func (e scheduledJobServiceError) GetOpcRequestID() string {
	return e.opcRequestID
}

var _ common.ServiceError = scheduledJobServiceError{}

type scheduledJobAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e scheduledJobAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e scheduledJobAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerScheduledJobRuntimeHooksMutator(func(manager *ScheduledJobServiceManager, hooks *ScheduledJobRuntimeHooks) {
		applyScheduledJobRuntimeHooks(manager, hooks)
	})
}

func applyScheduledJobRuntimeHooks(_ *ScheduledJobServiceManager, hooks *ScheduledJobRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newScheduledJobRuntimeSemantics()
	hooks.BuildCreateBody = buildScheduledJobCreateBody
	hooks.BuildUpdateBody = buildScheduledJobUpdateBody
	hooks.Identity.Resolve = resolveScheduledJobIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardScheduledJobExistingBeforeCreate
	hooks.List.Fields = scheduledJobListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listScheduledJobsAllPages(hooks.List.Call)
	}
	listScheduledJobs := hooks.List.Call
	getScheduledJob := hooks.Get.Call
	hooks.Identity.LookupExisting = func(ctx context.Context, resource *osmanagementhubv1beta1.ScheduledJob, identity any) (any, error) {
		return lookupExistingScheduledJob(ctx, resource, identity, listScheduledJobs, getScheduledJob)
	}
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *osmanagementhubv1beta1.ScheduledJob, currentID string) (any, error) {
		return confirmScheduledJobDelete(ctx, resource, currentID, getScheduledJob, listScheduledJobs)
	}
	hooks.DeleteHooks.HandleError = handleScheduledJobDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ScheduledJobServiceClient) ScheduledJobServiceClient {
		return &scheduledJobRuntimeClient{delegate: delegate, hooks: *hooks}
	})
}

func newScheduledJobRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "osmanagementhub",
		FormalSlug:        "scheduledjob",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(osmanagementhubsdk.ScheduledJobLifecycleStateCreating)},
			UpdatingStates:     []string{string(osmanagementhubsdk.ScheduledJobLifecycleStateUpdating)},
			ActiveStates: []string{
				string(osmanagementhubsdk.ScheduledJobLifecycleStateActive),
				string(osmanagementhubsdk.ScheduledJobLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(osmanagementhubsdk.ScheduledJobLifecycleStateCreating),
				string(osmanagementhubsdk.ScheduledJobLifecycleStateUpdating),
				string(osmanagementhubsdk.ScheduledJobLifecycleStateDeleting),
			},
			TerminalStates: []string{string(osmanagementhubsdk.ScheduledJobLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"scheduleType",
				"timeNextExecution",
				"operations",
				"locations",
				"managedInstanceIds",
				"managedInstanceGroupIds",
				"managedCompartmentIds",
				"lifecycleStageIds",
				"isSubcompartmentIncluded",
				"isManagedByAutonomousLinux",
				"retryIntervals",
				"workRequestId",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{
				"displayName",
				"description",
				"scheduleType",
				"timeNextExecution",
				"recurringRule",
				"operations",
				"freeformTags",
				"definedTags",
				"retryIntervals",
			},
			Mutable: []string{
				"displayName",
				"description",
				"scheduleType",
				"timeNextExecution",
				"recurringRule",
				"operations",
				"freeformTags",
				"definedTags",
				"retryIntervals",
			},
			ForceNew: []string{
				"compartmentId",
				"locations",
				"managedInstanceIds",
				"managedInstanceGroupIds",
				"managedCompartmentIds",
				"lifecycleStageIds",
				"workRequestId",
			},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func scheduledJobListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "ScheduleType", RequestName: "scheduleType", Contribution: "query", LookupPaths: []string{"status.scheduleType", "spec.scheduleType", "scheduleType"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func resolveScheduledJobIdentity(resource *osmanagementhubv1beta1.ScheduledJob) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("scheduledjob resource is nil")
	}
	if !scheduledJobSpecHasListIdentity(resource.Spec) {
		return nil, nil
	}

	scheduleType, ok := osmanagementhubsdk.GetMappingListScheduledJobsScheduleTypeEnum(strings.TrimSpace(resource.Spec.ScheduleType))
	if !ok {
		return nil, nil
	}
	operationType, ok := firstScheduledJobListOperationType(resource.Spec.Operations)
	if !ok {
		return nil, nil
	}
	return scheduledJobIdentity{
		CompartmentID:     strings.TrimSpace(resource.Spec.CompartmentId),
		ScheduleType:      scheduleType,
		TimeNextExecution: strings.TrimSpace(resource.Spec.TimeNextExecution),
		OperationType:     operationType,
	}, nil
}

func guardScheduledJobExistingBeforeCreate(
	_ context.Context,
	resource *osmanagementhubv1beta1.ScheduledJob,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("scheduledjob resource is nil")
	}
	if !scheduledJobSpecHasListIdentity(resource.Spec) {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func scheduledJobSpecHasListIdentity(spec osmanagementhubv1beta1.ScheduledJobSpec) bool {
	return strings.TrimSpace(spec.CompartmentId) != "" &&
		strings.TrimSpace(spec.ScheduleType) != "" &&
		strings.TrimSpace(spec.TimeNextExecution) != "" &&
		len(spec.Operations) > 0 &&
		strings.TrimSpace(spec.Operations[0].OperationType) != ""
}

func buildScheduledJobCreateBody(
	_ context.Context,
	resource *osmanagementhubv1beta1.ScheduledJob,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("scheduledjob resource is nil")
	}
	if err := validateScheduledJobSpec(resource.Spec); err != nil {
		return nil, err
	}

	timeNextExecution, err := scheduledJobTimeFromSpec("timeNextExecution", resource.Spec.TimeNextExecution)
	if err != nil {
		return nil, err
	}
	scheduleType, _ := osmanagementhubsdk.GetMappingScheduleTypesEnum(strings.TrimSpace(resource.Spec.ScheduleType))
	operations, err := scheduledJobOperationsFromSpec(resource.Spec.Operations)
	if err != nil {
		return nil, err
	}

	details := osmanagementhubsdk.CreateScheduledJobDetails{
		CompartmentId:     common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		ScheduleType:      scheduleType,
		TimeNextExecution: timeNextExecution,
		Operations:        operations,
	}
	if err := applyScheduledJobCreateOptionalFields(&details, resource.Spec); err != nil {
		return nil, err
	}
	applyScheduledJobCreateTargets(&details, resource.Spec)
	applyScheduledJobCreateTags(&details, resource.Spec)
	applyScheduledJobCreateFlags(&details, resource.Spec)
	applyScheduledJobCreateRetryAndWorkRequest(&details, resource.Spec)
	return details, nil
}

func applyScheduledJobCreateOptionalFields(
	details *osmanagementhubsdk.CreateScheduledJobDetails,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) error {
	if displayName := strings.TrimSpace(spec.DisplayName); displayName != "" {
		details.DisplayName = common.String(displayName)
	}
	if description := strings.TrimSpace(spec.Description); description != "" {
		details.Description = common.String(description)
	}
	locations, err := scheduledJobLocationsFromSpec(spec.Locations)
	if err != nil {
		return err
	}
	if len(locations) > 0 {
		details.Locations = locations
	}
	if recurringRule := strings.TrimSpace(spec.RecurringRule); recurringRule != "" {
		details.RecurringRule = common.String(recurringRule)
	}
	return nil
}

func applyScheduledJobCreateTargets(
	details *osmanagementhubsdk.CreateScheduledJobDetails,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) {
	details.ManagedInstanceIds = trimScheduledJobStringSlice(spec.ManagedInstanceIds)
	details.ManagedInstanceGroupIds = trimScheduledJobStringSlice(spec.ManagedInstanceGroupIds)
	details.ManagedCompartmentIds = trimScheduledJobStringSlice(spec.ManagedCompartmentIds)
	details.LifecycleStageIds = trimScheduledJobStringSlice(spec.LifecycleStageIds)
}

func applyScheduledJobCreateTags(
	details *osmanagementhubsdk.CreateScheduledJobDetails,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) {
	if spec.FreeformTags != nil {
		details.FreeformTags = cloneScheduledJobStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = scheduledJobDefinedTags(spec.DefinedTags)
	}
}

func applyScheduledJobCreateFlags(
	details *osmanagementhubsdk.CreateScheduledJobDetails,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) {
	if spec.IsSubcompartmentIncluded {
		details.IsSubcompartmentIncluded = common.Bool(true)
	}
	if spec.IsManagedByAutonomousLinux {
		details.IsManagedByAutonomousLinux = common.Bool(true)
	}
}

func applyScheduledJobCreateRetryAndWorkRequest(
	details *osmanagementhubsdk.CreateScheduledJobDetails,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) {
	details.RetryIntervals = cloneScheduledJobIntSlice(spec.RetryIntervals)
	if workRequestID := strings.TrimSpace(spec.WorkRequestId); workRequestID != "" {
		details.WorkRequestId = common.String(workRequestID)
	}
}

//nolint:gocognit,gocyclo // ScheduledJob update shaping needs per-field drift checks plus restricted-job guards.
func buildScheduledJobUpdateBody(
	_ context.Context,
	resource *osmanagementhubv1beta1.ScheduledJob,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return osmanagementhubsdk.UpdateScheduledJobDetails{}, false, fmt.Errorf("scheduledjob resource is nil")
	}
	if err := validateScheduledJobSpec(resource.Spec); err != nil {
		return osmanagementhubsdk.UpdateScheduledJobDetails{}, false, err
	}
	current, ok := scheduledJobBodyFromResponse(currentResponse)
	if !ok {
		return osmanagementhubsdk.UpdateScheduledJobDetails{}, false, fmt.Errorf("current ScheduledJob response does not expose a ScheduledJob body")
	}
	if err := validateScheduledJobCreateOnlyDrift(resource.Spec, current); err != nil {
		return osmanagementhubsdk.UpdateScheduledJobDetails{}, false, err
	}

	details := osmanagementhubsdk.UpdateScheduledJobDetails{}
	updateNeeded := false
	var restrictedDrift []string
	restricted := boolPointerValue(current.IsRestricted)

	if updated := applyScheduledJobDisplayNameUpdate(&details, resource.Spec, current); updated {
		updateNeeded = true
		restrictedDrift = appendRestrictedScheduledJobField(restrictedDrift, restricted, "displayName")
	}
	if updated := applyScheduledJobDescriptionUpdate(&details, resource.Spec, current); updated {
		updateNeeded = true
		restrictedDrift = appendRestrictedScheduledJobField(restrictedDrift, restricted, "description")
	}
	if updated := applyScheduledJobScheduleTypeUpdate(&details, resource.Spec, current); updated {
		updateNeeded = true
		restrictedDrift = appendRestrictedScheduledJobField(restrictedDrift, restricted, "scheduleType")
	}
	updated, err := applyScheduledJobTimeNextExecutionUpdate(&details, resource.Spec, current)
	if err != nil {
		return osmanagementhubsdk.UpdateScheduledJobDetails{}, false, err
	}
	updateNeeded = updated || updateNeeded
	if updated := applyScheduledJobRecurringRuleUpdate(&details, resource.Spec, current); updated {
		updateNeeded = true
	}
	updated, err = applyScheduledJobOperationsUpdate(&details, resource.Spec, current)
	if err != nil {
		return osmanagementhubsdk.UpdateScheduledJobDetails{}, false, err
	}
	if updated {
		updateNeeded = true
		restrictedDrift = appendRestrictedScheduledJobField(restrictedDrift, restricted, "operations")
	}
	if updated := applyScheduledJobFreeformTagsUpdate(&details, resource.Spec, current); updated {
		updateNeeded = true
	}
	if updated := applyScheduledJobDefinedTagsUpdate(&details, resource.Spec, current); updated {
		updateNeeded = true
	}
	if updated := applyScheduledJobRetryIntervalsUpdate(&details, resource.Spec, current); updated {
		updateNeeded = true
		restrictedDrift = appendRestrictedScheduledJobField(restrictedDrift, restricted, "retryIntervals")
	}
	if len(restrictedDrift) > 0 {
		return osmanagementhubsdk.UpdateScheduledJobDetails{}, false,
			fmt.Errorf("scheduledjob restricted update supports only timeNextExecution, recurringRule, freeformTags, and definedTags; unsupported drift: %s", strings.Join(restrictedDrift, ", "))
	}
	return details, updateNeeded, nil
}

func appendRestrictedScheduledJobField(fields []string, restricted bool, field string) []string {
	if !restricted {
		return fields
	}
	return append(fields, field)
}

func applyScheduledJobDisplayNameUpdate(
	details *osmanagementhubsdk.UpdateScheduledJobDetails,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
	current scheduledJobBody,
) bool {
	desired := strings.TrimSpace(spec.DisplayName)
	if desired == "" || desired == stringPointerValue(current.DisplayName) {
		return false
	}
	details.DisplayName = common.String(desired)
	return true
}

func applyScheduledJobDescriptionUpdate(
	details *osmanagementhubsdk.UpdateScheduledJobDetails,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
	current scheduledJobBody,
) bool {
	desired := strings.TrimSpace(spec.Description)
	if desired == "" || desired == stringPointerValue(current.Description) {
		return false
	}
	details.Description = common.String(desired)
	return true
}

func applyScheduledJobScheduleTypeUpdate(
	details *osmanagementhubsdk.UpdateScheduledJobDetails,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
	current scheduledJobBody,
) bool {
	desired, _ := osmanagementhubsdk.GetMappingScheduleTypesEnum(strings.TrimSpace(spec.ScheduleType))
	if string(desired) == strings.TrimSpace(string(current.ScheduleType)) {
		return false
	}
	details.ScheduleType = desired
	return true
}

func applyScheduledJobTimeNextExecutionUpdate(
	details *osmanagementhubsdk.UpdateScheduledJobDetails,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
	current scheduledJobBody,
) (bool, error) {
	desired, err := scheduledJobTimeFromSpec("timeNextExecution", spec.TimeNextExecution)
	if err != nil {
		return false, err
	}
	if scheduledJobTimesEqual(desired, current.TimeNextExecution) {
		return false, nil
	}
	details.TimeNextExecution = desired
	return true, nil
}

func applyScheduledJobRecurringRuleUpdate(
	details *osmanagementhubsdk.UpdateScheduledJobDetails,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
	current scheduledJobBody,
) bool {
	desired := strings.TrimSpace(spec.RecurringRule)
	if desired == "" || desired == stringPointerValue(current.RecurringRule) {
		return false
	}
	details.RecurringRule = common.String(desired)
	return true
}

func applyScheduledJobOperationsUpdate(
	details *osmanagementhubsdk.UpdateScheduledJobDetails,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
	current scheduledJobBody,
) (bool, error) {
	desired, err := scheduledJobOperationsFromSpec(spec.Operations)
	if err != nil {
		return false, err
	}
	if scheduledJobOperationsEqual(desired, current.Operations) {
		return false, nil
	}
	details.Operations = desired
	return true, nil
}

func applyScheduledJobFreeformTagsUpdate(
	details *osmanagementhubsdk.UpdateScheduledJobDetails,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
	current scheduledJobBody,
) bool {
	if spec.FreeformTags == nil || reflect.DeepEqual(spec.FreeformTags, current.FreeformTags) {
		return false
	}
	details.FreeformTags = cloneScheduledJobStringMap(spec.FreeformTags)
	return true
}

func applyScheduledJobDefinedTagsUpdate(
	details *osmanagementhubsdk.UpdateScheduledJobDetails,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
	current scheduledJobBody,
) bool {
	if spec.DefinedTags == nil {
		return false
	}
	desired := scheduledJobDefinedTags(spec.DefinedTags)
	if reflect.DeepEqual(desired, current.DefinedTags) {
		return false
	}
	details.DefinedTags = desired
	return true
}

func applyScheduledJobRetryIntervalsUpdate(
	details *osmanagementhubsdk.UpdateScheduledJobDetails,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
	current scheduledJobBody,
) bool {
	desired := cloneScheduledJobIntSlice(spec.RetryIntervals)
	if len(desired) == 0 || reflect.DeepEqual(desired, current.RetryIntervals) {
		return false
	}
	details.RetryIntervals = desired
	return true
}

func validateScheduledJobSpec(spec osmanagementhubv1beta1.ScheduledJobSpec) error {
	if err := validateScheduledJobRequiredFields(spec); err != nil {
		return err
	}
	if _, ok := osmanagementhubsdk.GetMappingScheduleTypesEnum(strings.TrimSpace(spec.ScheduleType)); !ok {
		return fmt.Errorf("scheduledjob spec is invalid: unsupported scheduleType %q", spec.ScheduleType)
	}
	if _, err := scheduledJobTimeFromSpec("timeNextExecution", spec.TimeNextExecution); err != nil {
		return err
	}
	if _, err := scheduledJobOperationsFromSpec(spec.Operations); err != nil {
		return err
	}
	if _, err := scheduledJobLocationsFromSpec(spec.Locations); err != nil {
		return err
	}
	if err := validateScheduledJobTargets(spec); err != nil {
		return err
	}
	return validateScheduledJobRetryIntervals(spec.RetryIntervals)
}

func validateScheduledJobRequiredFields(spec osmanagementhubv1beta1.ScheduledJobSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.ScheduleType) == "" {
		missing = append(missing, "scheduleType")
	}
	if strings.TrimSpace(spec.TimeNextExecution) == "" {
		missing = append(missing, "timeNextExecution")
	}
	if len(spec.Operations) == 0 {
		missing = append(missing, "operations")
	}
	if len(missing) > 0 {
		return fmt.Errorf("scheduledjob spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func validateScheduledJobRetryIntervals(intervals []int) error {
	for _, interval := range intervals {
		if interval < 0 {
			return fmt.Errorf("scheduledjob spec is invalid: retryIntervals must be non-negative")
		}
	}
	return nil
}

func validateScheduledJobTargets(spec osmanagementhubv1beta1.ScheduledJobSpec) error {
	targetGroups := scheduledJobTargetGroups(spec)
	if err := validateScheduledJobTargetGroupCount(targetGroups); err != nil {
		return err
	}
	if err := validateScheduledJobCompartmentTargetOptions(spec); err != nil {
		return err
	}
	return validateScheduledJobWorkRequestTarget(spec, targetGroups)
}

func validateScheduledJobTargetGroupCount(targetGroups []string) error {
	if len(targetGroups) > 1 {
		return fmt.Errorf("scheduledjob spec is invalid: exactly one target group is supported, got %s", strings.Join(targetGroups, ", "))
	}
	return nil
}

func validateScheduledJobCompartmentTargetOptions(spec osmanagementhubv1beta1.ScheduledJobSpec) error {
	if len(spec.Locations) > 0 && len(trimScheduledJobStringSlice(spec.ManagedCompartmentIds)) == 0 {
		return fmt.Errorf("scheduledjob spec is invalid: locations can be set only with managedCompartmentIds")
	}
	if spec.IsSubcompartmentIncluded && len(trimScheduledJobStringSlice(spec.ManagedCompartmentIds)) == 0 {
		return fmt.Errorf("scheduledjob spec is invalid: isSubcompartmentIncluded can be set only with managedCompartmentIds")
	}
	return nil
}

func validateScheduledJobWorkRequestTarget(spec osmanagementhubv1beta1.ScheduledJobSpec, targetGroups []string) error {
	includesRerun := scheduledJobOperationsIncludeRerunWorkRequest(spec.Operations)
	if strings.TrimSpace(spec.WorkRequestId) != "" && !includesRerun {
		return fmt.Errorf("scheduledjob spec is invalid: workRequestId can be set only with RERUN_WORK_REQUEST")
	}
	if len(targetGroups) == 0 && !includesRerun {
		return fmt.Errorf("scheduledjob spec is invalid: one of managedInstanceIds, managedInstanceGroupIds, managedCompartmentIds, or lifecycleStageIds is required")
	}
	if includesRerun && strings.TrimSpace(spec.WorkRequestId) == "" {
		return fmt.Errorf("scheduledjob spec is invalid: workRequestId is required with RERUN_WORK_REQUEST")
	}
	return nil
}

func scheduledJobTargetGroups(spec osmanagementhubv1beta1.ScheduledJobSpec) []string {
	var groups []string
	if len(trimScheduledJobStringSlice(spec.ManagedInstanceIds)) > 0 {
		groups = append(groups, "managedInstanceIds")
	}
	if len(trimScheduledJobStringSlice(spec.ManagedInstanceGroupIds)) > 0 {
		groups = append(groups, "managedInstanceGroupIds")
	}
	if len(trimScheduledJobStringSlice(spec.ManagedCompartmentIds)) > 0 {
		groups = append(groups, "managedCompartmentIds")
	}
	if len(trimScheduledJobStringSlice(spec.LifecycleStageIds)) > 0 {
		groups = append(groups, "lifecycleStageIds")
	}
	return groups
}

//nolint:gocyclo // Each create-only identity field needs an explicit pre-OCI drift guard.
func validateScheduledJobCreateOnlyDrift(spec osmanagementhubv1beta1.ScheduledJobSpec, current scheduledJobBody) error {
	driftFields := make([]string, 0, 9)
	if strings.TrimSpace(spec.CompartmentId) != stringPointerValue(current.CompartmentId) {
		driftFields = append(driftFields, "compartmentId")
	}
	if !scheduledJobLocationsEqualFromSpec(spec.Locations, current.Locations) {
		driftFields = append(driftFields, "locations")
	}
	if !equalScheduledJobStringSlices(trimScheduledJobStringSlice(spec.ManagedInstanceIds), current.ManagedInstanceIds) {
		driftFields = append(driftFields, "managedInstanceIds")
	}
	if !equalScheduledJobStringSlices(trimScheduledJobStringSlice(spec.ManagedInstanceGroupIds), current.ManagedInstanceGroupIds) {
		driftFields = append(driftFields, "managedInstanceGroupIds")
	}
	if !equalScheduledJobStringSlices(trimScheduledJobStringSlice(spec.ManagedCompartmentIds), current.ManagedCompartmentIds) {
		driftFields = append(driftFields, "managedCompartmentIds")
	}
	if !equalScheduledJobStringSlices(trimScheduledJobStringSlice(spec.LifecycleStageIds), current.LifecycleStageIds) {
		driftFields = append(driftFields, "lifecycleStageIds")
	}
	if spec.IsSubcompartmentIncluded != boolPointerValue(current.IsSubcompartmentIncluded) {
		driftFields = append(driftFields, "isSubcompartmentIncluded")
	}
	if spec.IsManagedByAutonomousLinux != boolPointerValue(current.IsManagedByAutonomousLinux) {
		driftFields = append(driftFields, "isManagedByAutonomousLinux")
	}
	if strings.TrimSpace(spec.WorkRequestId) != stringPointerValue(current.WorkRequestId) {
		driftFields = append(driftFields, "workRequestId")
	}
	if len(driftFields) > 0 {
		return fmt.Errorf("scheduledjob create-only field drift is not supported: %s", strings.Join(driftFields, ", "))
	}
	return nil
}

func scheduledJobTimeFromSpec(field string, value string) (*common.SDKTime, error) {
	value = strings.TrimSpace(value)
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, fmt.Errorf("scheduledjob spec is invalid: %s must be RFC3339: %w", field, err)
	}
	return &common.SDKTime{Time: parsed}, nil
}

//nolint:gocyclo // Multi-operation validation follows the service's per-operation allowlist.
func scheduledJobOperationsFromSpec(spec []osmanagementhubv1beta1.ScheduledJobOperation) ([]osmanagementhubsdk.ScheduledJobOperation, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("scheduledjob spec is invalid: operations is required")
	}
	if len(spec) > 1 {
		for _, operation := range spec {
			operationType, ok := osmanagementhubsdk.GetMappingOperationTypesEnum(strings.TrimSpace(operation.OperationType))
			if !ok {
				return nil, fmt.Errorf("scheduledjob spec is invalid: unsupported operations.operationType %q", operation.OperationType)
			}
			if !scheduledJobOperationAllowsMultiOperation(operationType) {
				return nil, fmt.Errorf("scheduledjob spec is invalid: operationType %s cannot be combined with other operations", operationType)
			}
		}
	}

	operations := make([]osmanagementhubsdk.ScheduledJobOperation, 0, len(spec))
	for index, operationSpec := range spec {
		operation, err := scheduledJobOperationFromSpec(index, operationSpec)
		if err != nil {
			return nil, err
		}
		operations = append(operations, operation)
	}
	return operations, nil
}

func scheduledJobOperationFromSpec(
	index int,
	spec osmanagementhubv1beta1.ScheduledJobOperation,
) (osmanagementhubsdk.ScheduledJobOperation, error) {
	operationType, ok := osmanagementhubsdk.GetMappingOperationTypesEnum(strings.TrimSpace(spec.OperationType))
	if !ok {
		return osmanagementhubsdk.ScheduledJobOperation{}, fmt.Errorf("scheduledjob spec is invalid: unsupported operations[%d].operationType %q", index, spec.OperationType)
	}
	if spec.RebootTimeoutInMins < 0 {
		return osmanagementhubsdk.ScheduledJobOperation{}, fmt.Errorf("scheduledjob spec is invalid: operations[%d].rebootTimeoutInMins must be non-negative", index)
	}

	operation := osmanagementhubsdk.ScheduledJobOperation{
		OperationType:      operationType,
		PackageNames:       trimScheduledJobStringSlice(spec.PackageNames),
		WindowsUpdateNames: trimScheduledJobStringSlice(spec.WindowsUpdateNames),
		SoftwareSourceIds:  trimScheduledJobStringSlice(spec.SoftwareSourceIds),
	}
	if manageDetails, ok, err := scheduledJobManageModuleStreamsFromSpec(index, spec.ManageModuleStreamsDetails); err != nil {
		return osmanagementhubsdk.ScheduledJobOperation{}, err
	} else if ok {
		operation.ManageModuleStreamsDetails = manageDetails
	}
	if switchDetails, ok, err := scheduledJobSwitchModuleStreamFromSpec(index, spec.SwitchModuleStreamsDetails); err != nil {
		return osmanagementhubsdk.ScheduledJobOperation{}, err
	} else if ok {
		operation.SwitchModuleStreamsDetails = switchDetails
	}
	if spec.RebootTimeoutInMins > 0 {
		operation.RebootTimeoutInMins = common.Int(spec.RebootTimeoutInMins)
	}
	return operation, nil
}

func scheduledJobOperationAllowsMultiOperation(operationType osmanagementhubsdk.OperationTypesEnum) bool {
	switch operationType {
	case osmanagementhubsdk.OperationTypesUpdatePackages,
		osmanagementhubsdk.OperationTypesUpdateAll,
		osmanagementhubsdk.OperationTypesUpdateSecurity,
		osmanagementhubsdk.OperationTypesUpdateBugfix,
		osmanagementhubsdk.OperationTypesUpdateEnhancement,
		osmanagementhubsdk.OperationTypesUpdateOther,
		osmanagementhubsdk.OperationTypesUpdateKspliceUserspace,
		osmanagementhubsdk.OperationTypesUpdateKspliceKernel:
		return true
	default:
		return false
	}
}

func scheduledJobOperationsIncludeRerunWorkRequest(spec []osmanagementhubv1beta1.ScheduledJobOperation) bool {
	for _, operation := range spec {
		operationType, ok := osmanagementhubsdk.GetMappingOperationTypesEnum(strings.TrimSpace(operation.OperationType))
		if ok && operationType == osmanagementhubsdk.OperationTypesRerunWorkRequest {
			return true
		}
	}
	return false
}

func scheduledJobManageModuleStreamsFromSpec(
	operationIndex int,
	spec osmanagementhubv1beta1.ScheduledJobOperationManageModuleStreamsDetails,
) (*osmanagementhubsdk.ManageModuleStreamsInScheduledJobDetails, bool, error) {
	specified := len(spec.Enable) > 0 || len(spec.Disable) > 0 || len(spec.Install) > 0 || len(spec.Remove) > 0
	if !specified {
		return nil, false, nil
	}
	details := &osmanagementhubsdk.ManageModuleStreamsInScheduledJobDetails{}
	var err error
	details.Enable, err = scheduledJobModuleStreamDetailsFromEnableSpec(operationIndex, "enable", spec.Enable)
	if err != nil {
		return nil, false, err
	}
	details.Disable, err = scheduledJobModuleStreamDetailsFromDisableSpec(operationIndex, "disable", spec.Disable)
	if err != nil {
		return nil, false, err
	}
	details.Install, err = scheduledJobModuleStreamProfileDetailsFromInstallSpec(operationIndex, "install", spec.Install)
	if err != nil {
		return nil, false, err
	}
	details.Remove, err = scheduledJobModuleStreamProfileDetailsFromRemoveSpec(operationIndex, "remove", spec.Remove)
	if err != nil {
		return nil, false, err
	}
	return details, true, nil
}

func scheduledJobModuleStreamDetailsFromEnableSpec(
	operationIndex int,
	field string,
	spec []osmanagementhubv1beta1.ScheduledJobOperationManageModuleStreamsDetailsEnable,
) ([]osmanagementhubsdk.ModuleStreamDetails, error) {
	details := make([]osmanagementhubsdk.ModuleStreamDetails, 0, len(spec))
	for index, item := range spec {
		detail, err := scheduledJobModuleStreamDetails(operationIndex, field, index, item.ModuleName, item.StreamName, item.SoftwareSourceId)
		if err != nil {
			return nil, err
		}
		details = append(details, detail)
	}
	return details, nil
}

func scheduledJobModuleStreamDetailsFromDisableSpec(
	operationIndex int,
	field string,
	spec []osmanagementhubv1beta1.ScheduledJobOperationManageModuleStreamsDetailsDisable,
) ([]osmanagementhubsdk.ModuleStreamDetails, error) {
	details := make([]osmanagementhubsdk.ModuleStreamDetails, 0, len(spec))
	for index, item := range spec {
		detail, err := scheduledJobModuleStreamDetails(operationIndex, field, index, item.ModuleName, item.StreamName, item.SoftwareSourceId)
		if err != nil {
			return nil, err
		}
		details = append(details, detail)
	}
	return details, nil
}

func scheduledJobModuleStreamDetails(
	operationIndex int,
	field string,
	index int,
	moduleName string,
	streamName string,
	softwareSourceID string,
) (osmanagementhubsdk.ModuleStreamDetails, error) {
	moduleName = strings.TrimSpace(moduleName)
	streamName = strings.TrimSpace(streamName)
	if moduleName == "" || streamName == "" {
		return osmanagementhubsdk.ModuleStreamDetails{}, fmt.Errorf("scheduledjob spec is invalid: operations[%d].manageModuleStreamsDetails.%s[%d].moduleName and streamName are required", operationIndex, field, index)
	}
	detail := osmanagementhubsdk.ModuleStreamDetails{
		ModuleName: common.String(moduleName),
		StreamName: common.String(streamName),
	}
	if softwareSourceID = strings.TrimSpace(softwareSourceID); softwareSourceID != "" {
		detail.SoftwareSourceId = common.String(softwareSourceID)
	}
	return detail, nil
}

func scheduledJobModuleStreamProfileDetailsFromInstallSpec(
	operationIndex int,
	field string,
	spec []osmanagementhubv1beta1.ScheduledJobOperationManageModuleStreamsDetailsInstall,
) ([]osmanagementhubsdk.ModuleStreamProfileDetails, error) {
	details := make([]osmanagementhubsdk.ModuleStreamProfileDetails, 0, len(spec))
	for index, item := range spec {
		detail, err := scheduledJobModuleStreamProfileDetails(operationIndex, field, index, item.ModuleName, item.StreamName, item.ProfileName, item.SoftwareSourceId)
		if err != nil {
			return nil, err
		}
		details = append(details, detail)
	}
	return details, nil
}

func scheduledJobModuleStreamProfileDetailsFromRemoveSpec(
	operationIndex int,
	field string,
	spec []osmanagementhubv1beta1.ScheduledJobOperationManageModuleStreamsDetailsRemove,
) ([]osmanagementhubsdk.ModuleStreamProfileDetails, error) {
	details := make([]osmanagementhubsdk.ModuleStreamProfileDetails, 0, len(spec))
	for index, item := range spec {
		detail, err := scheduledJobModuleStreamProfileDetails(operationIndex, field, index, item.ModuleName, item.StreamName, item.ProfileName, item.SoftwareSourceId)
		if err != nil {
			return nil, err
		}
		details = append(details, detail)
	}
	return details, nil
}

func scheduledJobModuleStreamProfileDetails(
	operationIndex int,
	field string,
	index int,
	moduleName string,
	streamName string,
	profileName string,
	softwareSourceID string,
) (osmanagementhubsdk.ModuleStreamProfileDetails, error) {
	moduleName = strings.TrimSpace(moduleName)
	streamName = strings.TrimSpace(streamName)
	profileName = strings.TrimSpace(profileName)
	if moduleName == "" || streamName == "" || profileName == "" {
		return osmanagementhubsdk.ModuleStreamProfileDetails{}, fmt.Errorf("scheduledjob spec is invalid: operations[%d].manageModuleStreamsDetails.%s[%d].moduleName, streamName, and profileName are required", operationIndex, field, index)
	}
	detail := osmanagementhubsdk.ModuleStreamProfileDetails{
		ModuleName:  common.String(moduleName),
		StreamName:  common.String(streamName),
		ProfileName: common.String(profileName),
	}
	if softwareSourceID = strings.TrimSpace(softwareSourceID); softwareSourceID != "" {
		detail.SoftwareSourceId = common.String(softwareSourceID)
	}
	return detail, nil
}

func scheduledJobSwitchModuleStreamFromSpec(
	operationIndex int,
	spec osmanagementhubv1beta1.ScheduledJobOperationSwitchModuleStreamsDetails,
) (*osmanagementhubsdk.ModuleStreamDetails, bool, error) {
	specified := strings.TrimSpace(spec.ModuleName) != "" ||
		strings.TrimSpace(spec.StreamName) != "" ||
		strings.TrimSpace(spec.SoftwareSourceId) != ""
	if !specified {
		return nil, false, nil
	}
	detail, err := scheduledJobModuleStreamDetails(operationIndex, "switchModuleStreamsDetails", 0, spec.ModuleName, spec.StreamName, spec.SoftwareSourceId)
	if err != nil {
		return nil, false, err
	}
	return &detail, true, nil
}

func scheduledJobLocationsFromSpec(spec []string) ([]osmanagementhubsdk.ManagedInstanceLocationEnum, error) {
	locations := make([]osmanagementhubsdk.ManagedInstanceLocationEnum, 0, len(spec))
	for _, location := range spec {
		location = strings.TrimSpace(location)
		if location == "" {
			continue
		}
		locationEnum, ok := osmanagementhubsdk.GetMappingManagedInstanceLocationEnum(location)
		if !ok {
			return nil, fmt.Errorf("scheduledjob spec is invalid: unsupported location %q", location)
		}
		locations = append(locations, locationEnum)
	}
	return locations, nil
}

func firstScheduledJobListOperationType(
	operations []osmanagementhubv1beta1.ScheduledJobOperation,
) (osmanagementhubsdk.ListScheduledJobsOperationTypeEnum, bool) {
	if len(operations) == 0 {
		return "", false
	}
	return osmanagementhubsdk.GetMappingListScheduledJobsOperationTypeEnum(strings.TrimSpace(operations[0].OperationType))
}

func listScheduledJobsAllPages(
	call func(context.Context, osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error),
) func(context.Context, osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error) {
	return func(ctx context.Context, request osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error) {
		var combined osmanagementhubsdk.ListScheduledJobsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return response, err
			}
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			combined.RawResponse = response.RawResponse
			combined.Items = append(combined.Items, response.Items...)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				return combined, nil
			}
			request.Page = response.OpcNextPage
		}
	}
}

func lookupExistingScheduledJob(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ScheduledJob,
	identity any,
	listScheduledJobs func(context.Context, osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error),
	getScheduledJob func(context.Context, osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error),
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("scheduledjob resource is nil")
	}
	if identity == nil || listScheduledJobs == nil {
		return nil, nil
	}
	request, err := scheduledJobListRequestFromSpec(resource.Spec)
	if err != nil {
		return nil, err
	}
	response, err := listScheduledJobs(ctx, request)
	if err != nil {
		return nil, err
	}
	matches, err := matchingScheduledJobFullBodies(ctx, resource.Spec, response.Items, getScheduledJob)
	if err != nil {
		return nil, err
	}
	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("multiple OCI ScheduledJobs matched compartmentId %q, scheduleType %q, timeNextExecution %q, and operationType %q",
			resource.Spec.CompartmentId, resource.Spec.ScheduleType, resource.Spec.TimeNextExecution, resource.Spec.Operations[0].OperationType)
	}
}

func scheduledJobListRequestFromSpec(spec osmanagementhubv1beta1.ScheduledJobSpec) (osmanagementhubsdk.ListScheduledJobsRequest, error) {
	request := osmanagementhubsdk.ListScheduledJobsRequest{
		CompartmentId: common.String(strings.TrimSpace(spec.CompartmentId)),
	}
	if scheduleType, ok := osmanagementhubsdk.GetMappingListScheduledJobsScheduleTypeEnum(strings.TrimSpace(spec.ScheduleType)); ok {
		request.ScheduleType = scheduleType
	} else {
		return request, fmt.Errorf("scheduledjob spec is invalid: unsupported scheduleType %q", spec.ScheduleType)
	}
	if operationType, ok := firstScheduledJobListOperationType(spec.Operations); ok {
		request.OperationType = operationType
	} else {
		return request, fmt.Errorf("scheduledjob spec is invalid: unsupported operations[0].operationType")
	}
	if ids := trimScheduledJobStringSlice(spec.ManagedInstanceIds); len(ids) > 0 {
		request.ManagedInstanceId = common.String(ids[0])
	}
	if ids := trimScheduledJobStringSlice(spec.ManagedInstanceGroupIds); len(ids) > 0 {
		request.ManagedInstanceGroupId = common.String(ids[0])
	}
	if ids := trimScheduledJobStringSlice(spec.ManagedCompartmentIds); len(ids) > 0 {
		request.ManagedCompartmentId = common.String(ids[0])
	}
	if ids := trimScheduledJobStringSlice(spec.LifecycleStageIds); len(ids) > 0 {
		request.LifecycleStageId = common.String(ids[0])
	}
	if locations, err := scheduledJobLocationsFromSpec(spec.Locations); err != nil {
		return request, err
	} else if len(locations) > 0 {
		request.Location = locations
	}
	if spec.IsManagedByAutonomousLinux {
		request.IsManagedByAutonomousLinux = common.Bool(true)
	}
	return request, nil
}

func matchingScheduledJobSummaries(
	items []osmanagementhubsdk.ScheduledJobSummary,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) []osmanagementhubsdk.ScheduledJobSummary {
	var matches []osmanagementhubsdk.ScheduledJobSummary
	for _, item := range items {
		if scheduledJobSummaryMatchesSpec(item, spec) {
			matches = append(matches, item)
		}
	}
	return matches
}

func scheduledJobSummaryMatchesSpec(
	summary osmanagementhubsdk.ScheduledJobSummary,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) bool {
	if !scheduledJobSummaryMatchesSchedule(summary, spec) {
		return false
	}
	if !scheduledJobSummaryMatchesOperations(summary, spec) {
		return false
	}
	if !scheduledJobSummaryMatchesTargets(summary, spec) {
		return false
	}
	return scheduledJobSummaryMatchesOptions(summary, spec)
}

func matchingScheduledJobFullBodies(
	ctx context.Context,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
	items []osmanagementhubsdk.ScheduledJobSummary,
	getScheduledJob func(context.Context, osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error),
) ([]osmanagementhubsdk.GetScheduledJobResponse, error) {
	summaryMatches := matchingScheduledJobSummaries(items, spec)
	if len(summaryMatches) == 0 {
		return nil, nil
	}
	if getScheduledJob == nil {
		return nil, fmt.Errorf("scheduledjob candidate validation requires GetScheduledJob")
	}

	fullMatches := make([]osmanagementhubsdk.GetScheduledJobResponse, 0, len(summaryMatches))
	for _, summary := range summaryMatches {
		response, found, err := getScheduledJobCandidate(ctx, summary, getScheduledJob)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		body := scheduledJobBodyFromSDKScheduledJob(response.ScheduledJob)
		if scheduledJobBodyMatchesSpec(body, spec) {
			fullMatches = append(fullMatches, response)
		}
	}
	return fullMatches, nil
}

func getScheduledJobCandidate(
	ctx context.Context,
	summary osmanagementhubsdk.ScheduledJobSummary,
	getScheduledJob func(context.Context, osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error),
) (osmanagementhubsdk.GetScheduledJobResponse, bool, error) {
	id := stringPointerValue(summary.Id)
	if id == "" {
		return osmanagementhubsdk.GetScheduledJobResponse{}, false, nil
	}
	response, err := getScheduledJob(ctx, osmanagementhubsdk.GetScheduledJobRequest{ScheduledJobId: common.String(id)})
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			return osmanagementhubsdk.GetScheduledJobResponse{}, false, nil
		}
		return osmanagementhubsdk.GetScheduledJobResponse{}, false, err
	}
	return response, true, nil
}

func scheduledJobBodyMatchesSpec(
	body scheduledJobBody,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) bool {
	if !scheduledJobBodyMatchesSchedule(body, spec) {
		return false
	}
	if !scheduledJobBodyMatchesOperations(body, spec) {
		return false
	}
	if !scheduledJobBodyMatchesTargets(body, spec) {
		return false
	}
	return scheduledJobBodyMatchesOptions(body, spec)
}

func scheduledJobBodyMatchesSchedule(
	body scheduledJobBody,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) bool {
	scheduleType, ok := osmanagementhubsdk.GetMappingScheduleTypesEnum(strings.TrimSpace(spec.ScheduleType))
	if !ok || scheduleType != body.ScheduleType {
		return false
	}
	timeNextExecution, err := scheduledJobTimeFromSpec("timeNextExecution", spec.TimeNextExecution)
	return err == nil &&
		scheduledJobTimesEqual(timeNextExecution, body.TimeNextExecution) &&
		stringPointerValue(body.CompartmentId) == strings.TrimSpace(spec.CompartmentId)
}

func scheduledJobBodyMatchesOperations(
	body scheduledJobBody,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) bool {
	operations, err := scheduledJobOperationsFromSpec(spec.Operations)
	return err == nil && scheduledJobOperationsEqual(operations, body.Operations)
}

func scheduledJobBodyMatchesTargets(
	body scheduledJobBody,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) bool {
	if !scheduledJobLocationsEqualFromSpec(spec.Locations, body.Locations) {
		return false
	}
	if !equalScheduledJobStringSlices(trimScheduledJobStringSlice(spec.ManagedInstanceIds), body.ManagedInstanceIds) {
		return false
	}
	if !equalScheduledJobStringSlices(trimScheduledJobStringSlice(spec.ManagedInstanceGroupIds), body.ManagedInstanceGroupIds) {
		return false
	}
	return equalScheduledJobStringSlices(trimScheduledJobStringSlice(spec.ManagedCompartmentIds), body.ManagedCompartmentIds) &&
		equalScheduledJobStringSlices(trimScheduledJobStringSlice(spec.LifecycleStageIds), body.LifecycleStageIds)
}

func scheduledJobBodyMatchesOptions(
	body scheduledJobBody,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) bool {
	return spec.IsSubcompartmentIncluded == boolPointerValue(body.IsSubcompartmentIncluded) &&
		spec.IsManagedByAutonomousLinux == boolPointerValue(body.IsManagedByAutonomousLinux) &&
		equalScheduledJobIntSlices(cloneScheduledJobIntSlice(spec.RetryIntervals), body.RetryIntervals) &&
		strings.TrimSpace(spec.WorkRequestId) == stringPointerValue(body.WorkRequestId)
}

func scheduledJobSummaryMatchesSchedule(
	summary osmanagementhubsdk.ScheduledJobSummary,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) bool {
	scheduleType, ok := osmanagementhubsdk.GetMappingScheduleTypesEnum(strings.TrimSpace(spec.ScheduleType))
	if !ok || scheduleType != summary.ScheduleType {
		return false
	}
	timeNextExecution, err := scheduledJobTimeFromSpec("timeNextExecution", spec.TimeNextExecution)
	return err == nil &&
		scheduledJobTimesEqual(timeNextExecution, summary.TimeNextExecution) &&
		stringPointerValue(summary.CompartmentId) == strings.TrimSpace(spec.CompartmentId)
}

func scheduledJobSummaryMatchesOperations(
	summary osmanagementhubsdk.ScheduledJobSummary,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) bool {
	operations, err := scheduledJobOperationsFromSpec(spec.Operations)
	return err == nil && scheduledJobOperationsEqual(operations, summary.Operations)
}

func scheduledJobSummaryMatchesTargets(
	summary osmanagementhubsdk.ScheduledJobSummary,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) bool {
	if !scheduledJobLocationsEqualFromSpec(spec.Locations, summary.Locations) {
		return false
	}
	if !equalScheduledJobStringSlices(trimScheduledJobStringSlice(spec.ManagedInstanceIds), summary.ManagedInstanceIds) {
		return false
	}
	if !equalScheduledJobStringSlices(trimScheduledJobStringSlice(spec.ManagedInstanceGroupIds), summary.ManagedInstanceGroupIds) {
		return false
	}
	return equalScheduledJobStringSlices(trimScheduledJobStringSlice(spec.ManagedCompartmentIds), summary.ManagedCompartmentIds) &&
		equalScheduledJobStringSlices(trimScheduledJobStringSlice(spec.LifecycleStageIds), summary.LifecycleStageIds)
}

func scheduledJobSummaryMatchesOptions(
	summary osmanagementhubsdk.ScheduledJobSummary,
	spec osmanagementhubv1beta1.ScheduledJobSpec,
) bool {
	return spec.IsManagedByAutonomousLinux == boolPointerValue(summary.IsManagedByAutonomousLinux) &&
		equalScheduledJobIntSlices(cloneScheduledJobIntSlice(spec.RetryIntervals), summary.RetryIntervals) &&
		strings.TrimSpace(spec.WorkRequestId) == stringPointerValue(summary.WorkRequestId)
}

func confirmScheduledJobDelete(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ScheduledJob,
	currentID string,
	getScheduledJob func(context.Context, osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error),
	listScheduledJobs func(context.Context, osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error),
) (any, error) {
	if strings.TrimSpace(currentID) != "" {
		return confirmScheduledJobDeleteByID(ctx, resource, currentID, getScheduledJob)
	}
	if resource == nil {
		return nil, fmt.Errorf("scheduledjob resource is nil")
	}
	return confirmScheduledJobDeleteByList(ctx, resource, listScheduledJobs, getScheduledJob)
}

func confirmScheduledJobDeleteByID(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ScheduledJob,
	currentID string,
	getScheduledJob func(context.Context, osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error),
) (any, error) {
	if getScheduledJob == nil {
		return nil, fmt.Errorf("scheduledjob delete confirmation cannot read OCI resource by id")
	}
	response, err := getScheduledJob(ctx, osmanagementhubsdk.GetScheduledJobRequest{ScheduledJobId: common.String(strings.TrimSpace(currentID))})
	if err != nil {
		return nil, conservativeScheduledJobNotFoundError(resource, err, "delete confirmation read")
	}
	return response, nil
}

func confirmScheduledJobDeleteByList(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ScheduledJob,
	listScheduledJobs func(context.Context, osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error),
	getScheduledJob func(context.Context, osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error),
) (any, error) {
	if listScheduledJobs == nil || !scheduledJobSpecHasListIdentity(resource.Spec) {
		return nil, newScheduledJobNotFoundError("scheduledjob delete confirmation has no recorded OCI identity")
	}
	request, err := scheduledJobListRequestFromSpec(resource.Spec)
	if err != nil {
		return nil, err
	}
	response, err := listScheduledJobs(ctx, request)
	if err != nil {
		return nil, conservativeScheduledJobNotFoundError(resource, err, "delete confirmation list")
	}
	matches, err := matchingScheduledJobFullBodies(ctx, resource.Spec, response.Items, getScheduledJob)
	if err != nil {
		return nil, conservativeScheduledJobNotFoundError(resource, err, "delete confirmation get")
	}
	switch len(matches) {
	case 0:
		return nil, newScheduledJobNotFoundError("scheduledjob delete confirmation did not find an OCI resource")
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("multiple OCI ScheduledJobs matched delete confirmation identity for compartmentId %q", resource.Spec.CompartmentId)
	}
}

func handleScheduledJobDeleteError(resource *osmanagementhubv1beta1.ScheduledJob, err error) error {
	return conservativeScheduledJobNotFoundError(resource, err, "delete")
}

func conservativeScheduledJobNotFoundError(resource *osmanagementhubv1beta1.ScheduledJob, err error, operation string) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return scheduledJobAmbiguousNotFoundError{
		message:      fmt.Sprintf("scheduledjob %s returned ambiguous 404 NotAuthorizedOrNotFound; retaining finalizer until deletion is unambiguously confirmed: %v", strings.TrimSpace(operation), err),
		opcRequestID: servicemanager.ErrorOpcRequestID(err),
	}
}

func newScheduledJobNotFoundError(message string) error {
	return scheduledJobServiceError{
		statusCode: 404,
		code:       errorutil.NotFound,
		message:    message,
	}
}

func (c *scheduledJobRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ScheduledJob,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("scheduledjob runtime client is not configured")
	}
	if currentScheduledJobID(resource) == "" {
		if response, found, err := c.lookupVerifiedScheduledJobBeforeCreate(ctx, resource); err != nil {
			return failScheduledJobCreateOrUpdate(resource, err)
		} else if found {
			recordScheduledJobRuntimeID(resource, stringPointerValue(response.Id))
		}
		ctx = generatedruntime.WithSkipExistingBeforeCreate(ctx)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *scheduledJobRuntimeClient) Delete(ctx context.Context, resource *osmanagementhubv1beta1.ScheduledJob) (bool, error) {
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("scheduledjob runtime client is not configured")
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *scheduledJobRuntimeClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ScheduledJob,
) error {
	if resource == nil {
		return nil
	}
	if currentID := currentScheduledJobID(resource); currentID != "" {
		return c.rejectAuthShapedPreDeleteGet(ctx, resource, currentID)
	}
	return c.rejectAuthShapedPreDeleteList(ctx, resource)
}

func (c *scheduledJobRuntimeClient) rejectAuthShapedPreDeleteGet(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ScheduledJob,
	currentID string,
) error {
	if c.hooks.Get.Call == nil {
		return nil
	}
	_, err := c.hooks.Get.Call(ctx, osmanagementhubsdk.GetScheduledJobRequest{ScheduledJobId: common.String(strings.TrimSpace(currentID))})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	return handleScheduledJobDeleteError(resource, err)
}

func (c *scheduledJobRuntimeClient) rejectAuthShapedPreDeleteList(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ScheduledJob,
) error {
	if c.hooks.List.Call == nil || !scheduledJobSpecHasListIdentity(resource.Spec) {
		return nil
	}
	request, err := scheduledJobListRequestFromSpec(resource.Spec)
	if err != nil {
		return nil
	}
	response, err := c.hooks.List.Call(ctx, request)
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			return handleScheduledJobDeleteError(resource, err)
		}
		return nil
	}
	matches := matchingScheduledJobSummaries(response.Items, resource.Spec)
	if len(matches) != 1 {
		return nil
	}
	return c.rejectAuthShapedPreDeleteGet(ctx, resource, stringPointerValue(matches[0].Id))
}

func (c *scheduledJobRuntimeClient) lookupVerifiedScheduledJobBeforeCreate(
	ctx context.Context,
	resource *osmanagementhubv1beta1.ScheduledJob,
) (osmanagementhubsdk.GetScheduledJobResponse, bool, error) {
	if resource == nil || !scheduledJobSpecHasListIdentity(resource.Spec) || c.hooks.List.Call == nil {
		return osmanagementhubsdk.GetScheduledJobResponse{}, false, nil
	}
	request, err := scheduledJobListRequestFromSpec(resource.Spec)
	if err != nil {
		return osmanagementhubsdk.GetScheduledJobResponse{}, false, err
	}
	response, err := c.hooks.List.Call(ctx, request)
	if err != nil {
		return osmanagementhubsdk.GetScheduledJobResponse{}, false, err
	}
	matches, err := matchingScheduledJobFullBodies(ctx, resource.Spec, response.Items, c.hooks.Get.Call)
	if err != nil {
		return osmanagementhubsdk.GetScheduledJobResponse{}, false, err
	}
	switch len(matches) {
	case 0:
		return osmanagementhubsdk.GetScheduledJobResponse{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return osmanagementhubsdk.GetScheduledJobResponse{}, false,
			fmt.Errorf("multiple OCI ScheduledJobs matched compartmentId %q, scheduleType %q, timeNextExecution %q, and operationType %q",
				resource.Spec.CompartmentId, resource.Spec.ScheduleType, resource.Spec.TimeNextExecution, resource.Spec.Operations[0].OperationType)
	}
}

func recordScheduledJobRuntimeID(resource *osmanagementhubv1beta1.ScheduledJob, id string) {
	if resource == nil || strings.TrimSpace(id) == "" {
		return
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(id)
	resource.Status.Id = id
}

func failScheduledJobCreateOrUpdate(
	resource *osmanagementhubv1beta1.ScheduledJob,
	err error,
) (servicemanager.OSOKResponse, error) {
	if resource != nil && err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		resource.Status.OsokStatus.Message = err.Error()
		resource.Status.OsokStatus.Reason = string(shared.Failed)
	}
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func currentScheduledJobID(resource *osmanagementhubv1beta1.ScheduledJob) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}

type scheduledJobBody struct {
	Id                         *string
	DisplayName                *string
	CompartmentId              *string
	ScheduleType               osmanagementhubsdk.ScheduleTypesEnum
	TimeNextExecution          *common.SDKTime
	Operations                 []osmanagementhubsdk.ScheduledJobOperation
	LifecycleState             osmanagementhubsdk.ScheduledJobLifecycleStateEnum
	FreeformTags               map[string]string
	DefinedTags                map[string]map[string]interface{}
	Description                *string
	Locations                  []osmanagementhubsdk.ManagedInstanceLocationEnum
	RecurringRule              *string
	ManagedInstanceIds         []string
	ManagedInstanceGroupIds    []string
	ManagedCompartmentIds      []string
	LifecycleStageIds          []string
	IsSubcompartmentIncluded   *bool
	IsManagedByAutonomousLinux *bool
	IsRestricted               *bool
	RetryIntervals             []int
	WorkRequestId              *string
}

//nolint:gocyclo // Response normalization must accept each SDK wrapper shape used by generatedruntime.
func scheduledJobBodyFromResponse(response any) (scheduledJobBody, bool) {
	switch current := response.(type) {
	case osmanagementhubsdk.CreateScheduledJobResponse:
		return scheduledJobBodyFromSDKScheduledJob(current.ScheduledJob), true
	case osmanagementhubsdk.GetScheduledJobResponse:
		return scheduledJobBodyFromSDKScheduledJob(current.ScheduledJob), true
	case osmanagementhubsdk.UpdateScheduledJobResponse:
		return scheduledJobBodyFromSDKScheduledJob(current.ScheduledJob), true
	case osmanagementhubsdk.ScheduledJob:
		return scheduledJobBodyFromSDKScheduledJob(current), true
	case osmanagementhubsdk.ScheduledJobSummary:
		return scheduledJobBodyFromSDKSummary(current), true
	case *osmanagementhubsdk.ScheduledJob:
		if current == nil {
			return scheduledJobBody{}, false
		}
		return scheduledJobBodyFromSDKScheduledJob(*current), true
	case *osmanagementhubsdk.ScheduledJobSummary:
		if current == nil {
			return scheduledJobBody{}, false
		}
		return scheduledJobBodyFromSDKSummary(*current), true
	default:
		return scheduledJobBody{}, false
	}
}

func scheduledJobBodyFromSDKScheduledJob(job osmanagementhubsdk.ScheduledJob) scheduledJobBody {
	return scheduledJobBody{
		Id:                         job.Id,
		DisplayName:                job.DisplayName,
		CompartmentId:              job.CompartmentId,
		ScheduleType:               job.ScheduleType,
		TimeNextExecution:          job.TimeNextExecution,
		Operations:                 job.Operations,
		LifecycleState:             job.LifecycleState,
		FreeformTags:               job.FreeformTags,
		DefinedTags:                job.DefinedTags,
		Description:                job.Description,
		Locations:                  job.Locations,
		RecurringRule:              job.RecurringRule,
		ManagedInstanceIds:         job.ManagedInstanceIds,
		ManagedInstanceGroupIds:    job.ManagedInstanceGroupIds,
		ManagedCompartmentIds:      job.ManagedCompartmentIds,
		LifecycleStageIds:          job.LifecycleStageIds,
		IsSubcompartmentIncluded:   job.IsSubcompartmentIncluded,
		IsManagedByAutonomousLinux: job.IsManagedByAutonomousLinux,
		IsRestricted:               job.IsRestricted,
		RetryIntervals:             job.RetryIntervals,
		WorkRequestId:              job.WorkRequestId,
	}
}

func scheduledJobBodyFromSDKSummary(summary osmanagementhubsdk.ScheduledJobSummary) scheduledJobBody {
	return scheduledJobBody{
		Id:                         summary.Id,
		DisplayName:                summary.DisplayName,
		CompartmentId:              summary.CompartmentId,
		ScheduleType:               summary.ScheduleType,
		TimeNextExecution:          summary.TimeNextExecution,
		Operations:                 summary.Operations,
		LifecycleState:             summary.LifecycleState,
		FreeformTags:               summary.FreeformTags,
		DefinedTags:                summary.DefinedTags,
		Locations:                  summary.Locations,
		ManagedInstanceIds:         summary.ManagedInstanceIds,
		ManagedInstanceGroupIds:    summary.ManagedInstanceGroupIds,
		ManagedCompartmentIds:      summary.ManagedCompartmentIds,
		LifecycleStageIds:          summary.LifecycleStageIds,
		IsManagedByAutonomousLinux: summary.IsManagedByAutonomousLinux,
		IsRestricted:               summary.IsRestricted,
		RetryIntervals:             summary.RetryIntervals,
		WorkRequestId:              summary.WorkRequestId,
	}
}

func scheduledJobOperationsEqual(
	left []osmanagementhubsdk.ScheduledJobOperation,
	right []osmanagementhubsdk.ScheduledJobOperation,
) bool {
	return reflect.DeepEqual(comparableScheduledJobOperations(left), comparableScheduledJobOperations(right))
}

type comparableScheduledJobOperation struct {
	OperationType                string
	PackageNames                 []string
	WindowsUpdateNames           []string
	ManageModuleStreamsDetails   comparableScheduledJobManageModuleStreams
	SwitchModuleStreamsDetails   comparableScheduledJobModuleStream
	SoftwareSourceIds            []string
	RebootTimeoutInMinsSpecified bool
	RebootTimeoutInMins          int
}

type comparableScheduledJobManageModuleStreams struct {
	Enable  []comparableScheduledJobModuleStream
	Disable []comparableScheduledJobModuleStream
	Install []comparableScheduledJobModuleStreamProfile
	Remove  []comparableScheduledJobModuleStreamProfile
}

type comparableScheduledJobModuleStream struct {
	Specified        bool
	ModuleName       string
	StreamName       string
	SoftwareSourceID string
}

type comparableScheduledJobModuleStreamProfile struct {
	ModuleName       string
	StreamName       string
	ProfileName      string
	SoftwareSourceID string
}

func comparableScheduledJobOperations(operations []osmanagementhubsdk.ScheduledJobOperation) []comparableScheduledJobOperation {
	if len(operations) == 0 {
		return nil
	}
	comparable := make([]comparableScheduledJobOperation, 0, len(operations))
	for _, operation := range operations {
		item := comparableScheduledJobOperation{
			OperationType:              strings.TrimSpace(string(operation.OperationType)),
			PackageNames:               trimScheduledJobStringSlice(operation.PackageNames),
			WindowsUpdateNames:         trimScheduledJobStringSlice(operation.WindowsUpdateNames),
			ManageModuleStreamsDetails: comparableScheduledJobManageModuleStreamsDetails(operation.ManageModuleStreamsDetails),
			SwitchModuleStreamsDetails: comparableScheduledJobModuleStreamDetails(operation.SwitchModuleStreamsDetails),
			SoftwareSourceIds:          trimScheduledJobStringSlice(operation.SoftwareSourceIds),
		}
		if operation.RebootTimeoutInMins != nil {
			item.RebootTimeoutInMinsSpecified = true
			item.RebootTimeoutInMins = *operation.RebootTimeoutInMins
		}
		comparable = append(comparable, item)
	}
	return comparable
}

func comparableScheduledJobManageModuleStreamsDetails(
	details *osmanagementhubsdk.ManageModuleStreamsInScheduledJobDetails,
) comparableScheduledJobManageModuleStreams {
	if details == nil {
		return comparableScheduledJobManageModuleStreams{}
	}
	return comparableScheduledJobManageModuleStreams{
		Enable:  comparableScheduledJobModuleStreams(details.Enable),
		Disable: comparableScheduledJobModuleStreams(details.Disable),
		Install: comparableScheduledJobModuleStreamProfiles(details.Install),
		Remove:  comparableScheduledJobModuleStreamProfiles(details.Remove),
	}
}

func comparableScheduledJobModuleStreams(streams []osmanagementhubsdk.ModuleStreamDetails) []comparableScheduledJobModuleStream {
	if len(streams) == 0 {
		return nil
	}
	comparable := make([]comparableScheduledJobModuleStream, 0, len(streams))
	for _, stream := range streams {
		comparable = append(comparable, comparableScheduledJobModuleStreamDetails(&stream))
	}
	return comparable
}

func comparableScheduledJobModuleStreamDetails(stream *osmanagementhubsdk.ModuleStreamDetails) comparableScheduledJobModuleStream {
	if stream == nil {
		return comparableScheduledJobModuleStream{}
	}
	return comparableScheduledJobModuleStream{
		Specified:        true,
		ModuleName:       stringPointerValue(stream.ModuleName),
		StreamName:       stringPointerValue(stream.StreamName),
		SoftwareSourceID: stringPointerValue(stream.SoftwareSourceId),
	}
}

func comparableScheduledJobModuleStreamProfiles(
	profiles []osmanagementhubsdk.ModuleStreamProfileDetails,
) []comparableScheduledJobModuleStreamProfile {
	if len(profiles) == 0 {
		return nil
	}
	comparable := make([]comparableScheduledJobModuleStreamProfile, 0, len(profiles))
	for _, profile := range profiles {
		comparable = append(comparable, comparableScheduledJobModuleStreamProfile{
			ModuleName:       stringPointerValue(profile.ModuleName),
			StreamName:       stringPointerValue(profile.StreamName),
			ProfileName:      stringPointerValue(profile.ProfileName),
			SoftwareSourceID: stringPointerValue(profile.SoftwareSourceId),
		})
	}
	return comparable
}

func scheduledJobLocationsEqualFromSpec(
	spec []string,
	current []osmanagementhubsdk.ManagedInstanceLocationEnum,
) bool {
	desired, err := scheduledJobLocationsFromSpec(spec)
	if err != nil {
		return false
	}
	return reflect.DeepEqual(comparableScheduledJobLocations(desired), comparableScheduledJobLocations(current))
}

func comparableScheduledJobLocations(locations []osmanagementhubsdk.ManagedInstanceLocationEnum) []string {
	if len(locations) == 0 {
		return nil
	}
	comparable := make([]string, 0, len(locations))
	for _, location := range locations {
		if value := strings.TrimSpace(string(location)); value != "" {
			comparable = append(comparable, value)
		}
	}
	if len(comparable) == 0 {
		return nil
	}
	return comparable
}

func scheduledJobTimesEqual(left *common.SDKTime, right *common.SDKTime) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.Equal(right.Time)
	}
}

func equalScheduledJobStringSlices(left []string, right []string) bool {
	return reflect.DeepEqual(trimScheduledJobStringSlice(left), trimScheduledJobStringSlice(right))
}

func equalScheduledJobIntSlices(left []int, right []int) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}
	return reflect.DeepEqual(left, right)
}

func trimScheduledJobStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			trimmed = append(trimmed, value)
		}
	}
	if len(trimmed) == 0 {
		return nil
	}
	return trimmed
}

func cloneScheduledJobStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneScheduledJobIntSlice(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	return append([]int(nil), values...)
}

func scheduledJobDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&tags)
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func boolPointerValue(value *bool) bool {
	return value != nil && *value
}

func newScheduledJobServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client scheduledJobOCIClient,
) ScheduledJobServiceClient {
	hooks := newScheduledJobRuntimeHooksWithOCIClient(client)
	applyScheduledJobRuntimeHooks(&ScheduledJobServiceManager{Log: log}, &hooks)
	manager := &ScheduledJobServiceManager{Log: log}
	delegate := defaultScheduledJobServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*osmanagementhubv1beta1.ScheduledJob](
			buildScheduledJobGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapScheduledJobGeneratedClient(hooks, delegate)
}

func newScheduledJobRuntimeHooksWithOCIClient(client scheduledJobOCIClient) ScheduledJobRuntimeHooks {
	return ScheduledJobRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*osmanagementhubv1beta1.ScheduledJob]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*osmanagementhubv1beta1.ScheduledJob]{},
		StatusHooks:     generatedruntime.StatusHooks[*osmanagementhubv1beta1.ScheduledJob]{},
		ParityHooks:     generatedruntime.ParityHooks[*osmanagementhubv1beta1.ScheduledJob]{},
		Async:           generatedruntime.AsyncHooks[*osmanagementhubv1beta1.ScheduledJob]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*osmanagementhubv1beta1.ScheduledJob]{},
		Create: runtimeOperationHooks[osmanagementhubsdk.CreateScheduledJobRequest, osmanagementhubsdk.CreateScheduledJobResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateScheduledJobDetails", RequestName: "CreateScheduledJobDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request osmanagementhubsdk.CreateScheduledJobRequest) (osmanagementhubsdk.CreateScheduledJobResponse, error) {
				if client == nil {
					return osmanagementhubsdk.CreateScheduledJobResponse{}, fmt.Errorf("scheduledjob OCI client is nil")
				}
				return client.CreateScheduledJob(ctx, request)
			},
		},
		Get: runtimeOperationHooks[osmanagementhubsdk.GetScheduledJobRequest, osmanagementhubsdk.GetScheduledJobResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ScheduledJobId", RequestName: "scheduledJobId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request osmanagementhubsdk.GetScheduledJobRequest) (osmanagementhubsdk.GetScheduledJobResponse, error) {
				if client == nil {
					return osmanagementhubsdk.GetScheduledJobResponse{}, fmt.Errorf("scheduledjob OCI client is nil")
				}
				return client.GetScheduledJob(ctx, request)
			},
		},
		List: runtimeOperationHooks[osmanagementhubsdk.ListScheduledJobsRequest, osmanagementhubsdk.ListScheduledJobsResponse]{
			Fields: scheduledJobListFields(),
			Call: func(ctx context.Context, request osmanagementhubsdk.ListScheduledJobsRequest) (osmanagementhubsdk.ListScheduledJobsResponse, error) {
				if client == nil {
					return osmanagementhubsdk.ListScheduledJobsResponse{}, fmt.Errorf("scheduledjob OCI client is nil")
				}
				return client.ListScheduledJobs(ctx, request)
			},
		},
		Update: runtimeOperationHooks[osmanagementhubsdk.UpdateScheduledJobRequest, osmanagementhubsdk.UpdateScheduledJobResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ScheduledJobId", RequestName: "scheduledJobId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateScheduledJobDetails", RequestName: "UpdateScheduledJobDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request osmanagementhubsdk.UpdateScheduledJobRequest) (osmanagementhubsdk.UpdateScheduledJobResponse, error) {
				if client == nil {
					return osmanagementhubsdk.UpdateScheduledJobResponse{}, fmt.Errorf("scheduledjob OCI client is nil")
				}
				return client.UpdateScheduledJob(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[osmanagementhubsdk.DeleteScheduledJobRequest, osmanagementhubsdk.DeleteScheduledJobResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ScheduledJobId", RequestName: "scheduledJobId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request osmanagementhubsdk.DeleteScheduledJobRequest) (osmanagementhubsdk.DeleteScheduledJobResponse, error) {
				if client == nil {
					return osmanagementhubsdk.DeleteScheduledJobResponse{}, fmt.Errorf("scheduledjob OCI client is nil")
				}
				return client.DeleteScheduledJob(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ScheduledJobServiceClient) ScheduledJobServiceClient{},
	}
}
