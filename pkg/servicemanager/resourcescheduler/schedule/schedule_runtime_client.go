/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	resourceschedulersdk "github.com/oracle/oci-go-sdk/v65/resourcescheduler"
	resourceschedulerv1beta1 "github.com/oracle/oci-service-operator/api/resourcescheduler/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type scheduleOCIClient interface {
	CreateSchedule(context.Context, resourceschedulersdk.CreateScheduleRequest) (resourceschedulersdk.CreateScheduleResponse, error)
	GetSchedule(context.Context, resourceschedulersdk.GetScheduleRequest) (resourceschedulersdk.GetScheduleResponse, error)
	ListSchedules(context.Context, resourceschedulersdk.ListSchedulesRequest) (resourceschedulersdk.ListSchedulesResponse, error)
	UpdateSchedule(context.Context, resourceschedulersdk.UpdateScheduleRequest) (resourceschedulersdk.UpdateScheduleResponse, error)
	DeleteSchedule(context.Context, resourceschedulersdk.DeleteScheduleRequest) (resourceschedulersdk.DeleteScheduleResponse, error)
}

type scheduleRuntimeClient struct {
	delegate    ScheduleServiceClient
	get         func(context.Context, resourceschedulersdk.GetScheduleRequest) (resourceschedulersdk.GetScheduleResponse, error)
	confirmRead func(context.Context, *resourceschedulerv1beta1.Schedule, string) (any, error)
}

type scheduleAuthShapedConfirmRead struct {
	err error
}

type scheduleNoMatchConfirmRead struct {
	message string
}

func (e scheduleAuthShapedConfirmRead) Error() string {
	return fmt.Sprintf("schedule delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", e.err)
}

func (e scheduleAuthShapedConfirmRead) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

func (e scheduleNoMatchConfirmRead) Error() string {
	return e.message
}

var _ ScheduleServiceClient = (*scheduleRuntimeClient)(nil)

func init() {
	registerScheduleRuntimeHooksMutator(func(_ *ScheduleServiceManager, hooks *ScheduleRuntimeHooks) {
		applyScheduleRuntimeHooks(hooks)
	})
}

func applyScheduleRuntimeHooks(hooks *ScheduleRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = scheduleRuntimeSemantics()
	hooks.BuildCreateBody = buildScheduleCreateBody
	hooks.BuildUpdateBody = buildScheduleUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardScheduleExistingBeforeCreate
	hooks.List.Fields = scheduleListFields()
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *resourceschedulerv1beta1.Schedule, currentID string) (any, error) {
		return confirmScheduleDeleteRead(ctx, hooks, resource, currentID)
	}
	hooks.DeleteHooks.HandleError = handleScheduleDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyScheduleDeleteConfirmOutcome
	wrapScheduleReadAndDeleteCalls(hooks)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ScheduleServiceClient) ScheduleServiceClient {
		return &scheduleRuntimeClient{
			delegate:    delegate,
			get:         hooks.Get.Call,
			confirmRead: hooks.DeleteHooks.ConfirmRead,
		}
	})
}

func newScheduleRuntimeHooksWithOCIClient(client scheduleOCIClient) ScheduleRuntimeHooks {
	return ScheduleRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*resourceschedulerv1beta1.Schedule]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*resourceschedulerv1beta1.Schedule]{},
		StatusHooks:     generatedruntime.StatusHooks[*resourceschedulerv1beta1.Schedule]{},
		ParityHooks:     generatedruntime.ParityHooks[*resourceschedulerv1beta1.Schedule]{},
		Async:           generatedruntime.AsyncHooks[*resourceschedulerv1beta1.Schedule]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*resourceschedulerv1beta1.Schedule]{},
		Create: runtimeOperationHooks[resourceschedulersdk.CreateScheduleRequest, resourceschedulersdk.CreateScheduleResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateScheduleDetails", RequestName: "CreateScheduleDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request resourceschedulersdk.CreateScheduleRequest) (resourceschedulersdk.CreateScheduleResponse, error) {
				if client == nil {
					return resourceschedulersdk.CreateScheduleResponse{}, fmt.Errorf("schedule OCI client is nil")
				}
				return client.CreateSchedule(ctx, request)
			},
		},
		Get: runtimeOperationHooks[resourceschedulersdk.GetScheduleRequest, resourceschedulersdk.GetScheduleResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ScheduleId", RequestName: "scheduleId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request resourceschedulersdk.GetScheduleRequest) (resourceschedulersdk.GetScheduleResponse, error) {
				if client == nil {
					return resourceschedulersdk.GetScheduleResponse{}, fmt.Errorf("schedule OCI client is nil")
				}
				return client.GetSchedule(ctx, request)
			},
		},
		List: runtimeOperationHooks[resourceschedulersdk.ListSchedulesRequest, resourceschedulersdk.ListSchedulesResponse]{
			Fields: scheduleListFields(),
			Call: func(ctx context.Context, request resourceschedulersdk.ListSchedulesRequest) (resourceschedulersdk.ListSchedulesResponse, error) {
				if client == nil {
					return resourceschedulersdk.ListSchedulesResponse{}, fmt.Errorf("schedule OCI client is nil")
				}
				return client.ListSchedules(ctx, request)
			},
		},
		Update: runtimeOperationHooks[resourceschedulersdk.UpdateScheduleRequest, resourceschedulersdk.UpdateScheduleResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "ScheduleId", RequestName: "scheduleId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateScheduleDetails", RequestName: "UpdateScheduleDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request resourceschedulersdk.UpdateScheduleRequest) (resourceschedulersdk.UpdateScheduleResponse, error) {
				if client == nil {
					return resourceschedulersdk.UpdateScheduleResponse{}, fmt.Errorf("schedule OCI client is nil")
				}
				return client.UpdateSchedule(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[resourceschedulersdk.DeleteScheduleRequest, resourceschedulersdk.DeleteScheduleResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "ScheduleId", RequestName: "scheduleId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request resourceschedulersdk.DeleteScheduleRequest) (resourceschedulersdk.DeleteScheduleResponse, error) {
				if client == nil {
					return resourceschedulersdk.DeleteScheduleResponse{}, fmt.Errorf("schedule OCI client is nil")
				}
				return client.DeleteSchedule(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ScheduleServiceClient) ScheduleServiceClient{},
	}
}

func scheduleRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "resourcescheduler",
		FormalSlug:        "schedule",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(resourceschedulersdk.ScheduleLifecycleStateCreating)},
			UpdatingStates:     []string{string(resourceschedulersdk.ScheduleLifecycleStateUpdating)},
			ActiveStates: []string{
				string(resourceschedulersdk.ScheduleLifecycleStateActive),
				string(resourceschedulersdk.ScheduleLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(resourceschedulersdk.ScheduleLifecycleStateCreating),
				string(resourceschedulersdk.ScheduleLifecycleStateUpdating),
				string(resourceschedulersdk.ScheduleLifecycleStateDeleting),
			},
			TerminalStates: []string{string(resourceschedulersdk.ScheduleLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{
				"action",
				"definedTags",
				"description",
				"displayName",
				"freeformTags",
				"recurrenceDetails",
				"recurrenceType",
				"resourceFilters",
				"resources",
				"timeEnds",
				"timeStarts",
			},
			Mutable: []string{
				"action",
				"definedTags",
				"description",
				"displayName",
				"freeformTags",
				"recurrenceDetails",
				"recurrenceType",
				"resourceFilters",
				"resources",
				"timeEnds",
				"timeStarts",
			},
			ForceNew:      []string{"compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "Schedule", Action: "CreateSchedule"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "Schedule", Action: "UpdateSchedule"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "Schedule", Action: "DeleteSchedule"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func scheduleListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "ScheduleId", RequestName: "scheduleId", Contribution: "query", PreferResourceID: true},
		{FieldName: "ResourceId", RequestName: "resourceId", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
	}
}

func guardScheduleExistingBeforeCreate(
	_ context.Context,
	resource *resourceschedulerv1beta1.Schedule,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("schedule resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildScheduleCreateBody(
	_ context.Context,
	resource *resourceschedulerv1beta1.Schedule,
	_ string,
) (any, error) {
	if resource == nil {
		return resourceschedulersdk.CreateScheduleDetails{}, fmt.Errorf("schedule resource is nil")
	}
	if err := validateScheduleSpec(resource.Spec); err != nil {
		return resourceschedulersdk.CreateScheduleDetails{}, err
	}

	spec := resource.Spec
	filters, err := scheduleResourceFiltersFromSpec(spec.ResourceFilters)
	if err != nil {
		return resourceschedulersdk.CreateScheduleDetails{}, err
	}
	resources, err := scheduleResourcesFromSpec(spec.Resources)
	if err != nil {
		return resourceschedulersdk.CreateScheduleDetails{}, err
	}
	timeStarts, err := scheduleSDKTimeFromSpec("timeStarts", spec.TimeStarts)
	if err != nil {
		return resourceschedulersdk.CreateScheduleDetails{}, err
	}
	timeEnds, err := scheduleSDKTimeFromSpec("timeEnds", spec.TimeEnds)
	if err != nil {
		return resourceschedulersdk.CreateScheduleDetails{}, err
	}

	return resourceschedulersdk.CreateScheduleDetails{
		CompartmentId:     common.String(spec.CompartmentId),
		Action:            resourceschedulersdk.CreateScheduleDetailsActionEnum(scheduleNormalizeEnum(spec.Action)),
		RecurrenceDetails: common.String(spec.RecurrenceDetails),
		RecurrenceType:    resourceschedulersdk.CreateScheduleDetailsRecurrenceTypeEnum(scheduleNormalizeEnum(spec.RecurrenceType)),
		DisplayName:       scheduleOptionalString(spec.DisplayName),
		Description:       scheduleOptionalString(spec.Description),
		ResourceFilters:   filters,
		Resources:         resources,
		TimeStarts:        timeStarts,
		TimeEnds:          timeEnds,
		FreeformTags:      maps.Clone(spec.FreeformTags),
		DefinedTags:       scheduleDefinedTagsFromSpec(spec.DefinedTags),
	}, nil
}

func buildScheduleUpdateBody(
	_ context.Context,
	resource *resourceschedulerv1beta1.Schedule,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return resourceschedulersdk.UpdateScheduleDetails{}, false, fmt.Errorf("schedule resource is nil")
	}
	if err := validateScheduleSpec(resource.Spec); err != nil {
		return resourceschedulersdk.UpdateScheduleDetails{}, false, err
	}

	current, ok := scheduleFromResponse(currentResponse)
	if !ok {
		return resourceschedulersdk.UpdateScheduleDetails{}, false, fmt.Errorf("current Schedule response does not expose a Schedule body")
	}
	if currentCompartment := scheduleStringValue(current.CompartmentId); currentCompartment != "" &&
		resource.Spec.CompartmentId != currentCompartment {
		return resourceschedulersdk.UpdateScheduleDetails{}, false,
			fmt.Errorf("schedule formal semantics require replacement when compartmentId changes")
	}

	details, updateNeeded, err := scheduleUpdateDetailsFromSpec(resource.Spec, current)
	if err != nil {
		return resourceschedulersdk.UpdateScheduleDetails{}, false, err
	}
	return details, updateNeeded, nil
}

func scheduleUpdateDetailsFromSpec(
	spec resourceschedulerv1beta1.ScheduleSpec,
	current resourceschedulersdk.Schedule,
) (resourceschedulersdk.UpdateScheduleDetails, bool, error) {
	details := resourceschedulersdk.UpdateScheduleDetails{}
	updateNeeded := scheduleApplyStringUpdates(&details, spec, current)
	updateNeeded = scheduleApplyEnumUpdates(&details, spec, current) || updateNeeded
	updateNeeded = scheduleApplyTagUpdates(&details, spec, current) || updateNeeded

	collectionUpdates, err := scheduleApplyCollectionUpdates(&details, spec, current)
	if err != nil {
		return resourceschedulersdk.UpdateScheduleDetails{}, false, err
	}
	updateNeeded = collectionUpdates || updateNeeded

	timeUpdates, err := scheduleApplyTimeUpdates(&details, spec, current)
	if err != nil {
		return resourceschedulersdk.UpdateScheduleDetails{}, false, err
	}
	updateNeeded = timeUpdates || updateNeeded
	return details, updateNeeded, nil
}

func scheduleApplyStringUpdates(
	details *resourceschedulersdk.UpdateScheduleDetails,
	spec resourceschedulerv1beta1.ScheduleSpec,
	current resourceschedulersdk.Schedule,
) bool {
	updateNeeded := false
	if desired, ok := scheduleStringUpdate(spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := scheduleStringUpdate(spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := scheduleStringUpdate(spec.RecurrenceDetails, current.RecurrenceDetails); ok {
		details.RecurrenceDetails = desired
		updateNeeded = true
	}
	return updateNeeded
}

func scheduleApplyEnumUpdates(
	details *resourceschedulersdk.UpdateScheduleDetails,
	spec resourceschedulerv1beta1.ScheduleSpec,
	current resourceschedulersdk.Schedule,
) bool {
	updateNeeded := false
	if desired := scheduleNormalizeEnum(spec.Action); desired != "" && desired != string(current.Action) {
		details.Action = resourceschedulersdk.UpdateScheduleDetailsActionEnum(desired)
		updateNeeded = true
	}
	if desired := scheduleNormalizeEnum(spec.RecurrenceType); desired != "" && desired != string(current.RecurrenceType) {
		details.RecurrenceType = resourceschedulersdk.UpdateScheduleDetailsRecurrenceTypeEnum(desired)
		updateNeeded = true
	}
	return updateNeeded
}

func scheduleApplyTagUpdates(
	details *resourceschedulersdk.UpdateScheduleDetails,
	spec resourceschedulerv1beta1.ScheduleSpec,
	current resourceschedulersdk.Schedule,
) bool {
	updateNeeded := false
	if spec.FreeformTags != nil && !maps.Equal(spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
		updateNeeded = true
	}
	if spec.DefinedTags != nil {
		desired := scheduleDefinedTagsFromSpec(spec.DefinedTags)
		if !reflect.DeepEqual(desired, current.DefinedTags) {
			details.DefinedTags = desired
			updateNeeded = true
		}
	}
	return updateNeeded
}

func scheduleApplyCollectionUpdates(
	details *resourceschedulersdk.UpdateScheduleDetails,
	spec resourceschedulerv1beta1.ScheduleSpec,
	current resourceschedulersdk.Schedule,
) (bool, error) {
	updateNeeded := false
	if spec.ResourceFilters != nil {
		desired, err := scheduleResourceFiltersFromSpec(spec.ResourceFilters)
		if err != nil {
			return false, err
		}
		if !scheduleJSONEqual(scheduleCanonicalResourceFilters(desired), scheduleCanonicalResourceFilters(current.ResourceFilters)) {
			details.ResourceFilters = desired
			updateNeeded = true
		}
	}
	if spec.Resources != nil {
		desired, err := scheduleResourcesFromSpec(spec.Resources)
		if err != nil {
			return false, err
		}
		if !scheduleJSONEqual(scheduleCanonicalResources(desired), scheduleCanonicalResources(current.Resources)) {
			details.Resources = desired
			updateNeeded = true
		}
	}
	return updateNeeded, nil
}

func scheduleApplyTimeUpdates(
	details *resourceschedulersdk.UpdateScheduleDetails,
	spec resourceschedulerv1beta1.ScheduleSpec,
	current resourceschedulersdk.Schedule,
) (bool, error) {
	updateNeeded := false
	if strings.TrimSpace(spec.TimeStarts) != "" {
		desired, err := scheduleSDKTimeFromSpec("timeStarts", spec.TimeStarts)
		if err != nil {
			return false, err
		}
		if current.TimeStarts == nil || !current.TimeStarts.Equal(desired.Time) {
			details.TimeStarts = desired
			updateNeeded = true
		}
	}
	if strings.TrimSpace(spec.TimeEnds) != "" {
		desired, err := scheduleSDKTimeFromSpec("timeEnds", spec.TimeEnds)
		if err != nil {
			return false, err
		}
		if current.TimeEnds == nil || !current.TimeEnds.Equal(desired.Time) {
			details.TimeEnds = desired
			updateNeeded = true
		}
	}
	return updateNeeded, nil
}

func validateScheduleSpec(spec resourceschedulerv1beta1.ScheduleSpec) error {
	problems := scheduleRequiredFieldProblems(spec)
	problems = append(problems, scheduleEnumProblems(spec)...)
	problems = append(problems, scheduleTimeProblems(spec)...)
	problems = append(problems, scheduleNestedSpecProblems(spec)...)
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("schedule spec is invalid: %s", strings.Join(problems, "; "))
}

func scheduleRequiredFieldProblems(spec resourceschedulerv1beta1.ScheduleSpec) []string {
	requiredFields := []struct {
		name  string
		value string
	}{
		{name: "compartmentId", value: spec.CompartmentId},
		{name: "action", value: spec.Action},
		{name: "recurrenceDetails", value: spec.RecurrenceDetails},
		{name: "recurrenceType", value: spec.RecurrenceType},
	}

	problems := make([]string, 0, len(requiredFields)+1)
	for _, field := range requiredFields {
		if strings.TrimSpace(field.value) == "" {
			problems = append(problems, fmt.Sprintf("%s is required", field.name))
		}
	}
	if len(spec.ResourceFilters) == 0 && len(spec.Resources) == 0 {
		problems = append(problems, "either resourceFilters or resources is required")
	}
	return problems
}

func scheduleEnumProblems(spec resourceschedulerv1beta1.ScheduleSpec) []string {
	var problems []string
	if scheduleUnsupportedAction(spec.Action) {
		problems = append(problems, fmt.Sprintf("action %q is unsupported", spec.Action))
	}
	if scheduleUnsupportedRecurrenceType(spec.RecurrenceType) {
		problems = append(problems, fmt.Sprintf("recurrenceType %q is unsupported", spec.RecurrenceType))
	}
	return problems
}

func scheduleUnsupportedAction(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	_, ok := resourceschedulersdk.GetMappingCreateScheduleDetailsActionEnum(value)
	return !ok
}

func scheduleUnsupportedRecurrenceType(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	_, ok := resourceschedulersdk.GetMappingCreateScheduleDetailsRecurrenceTypeEnum(value)
	return !ok
}

func scheduleTimeProblems(spec resourceschedulerv1beta1.ScheduleSpec) []string {
	timeFields := []struct {
		name  string
		value string
	}{
		{name: "timeStarts", value: spec.TimeStarts},
		{name: "timeEnds", value: spec.TimeEnds},
	}

	var problems []string
	for _, field := range timeFields {
		if _, err := scheduleSDKTimeFromSpec(field.name, field.value); err != nil {
			problems = append(problems, err.Error())
		}
	}
	return problems
}

func scheduleNestedSpecProblems(spec resourceschedulerv1beta1.ScheduleSpec) []string {
	var problems []string
	if _, err := scheduleResourceFiltersFromSpec(spec.ResourceFilters); err != nil {
		problems = append(problems, err.Error())
	}
	if _, err := scheduleResourcesFromSpec(spec.Resources); err != nil {
		problems = append(problems, err.Error())
	}
	return problems
}

func scheduleResourceFiltersFromSpec(filters []resourceschedulerv1beta1.ScheduleResourceFilter) ([]resourceschedulersdk.ResourceFilter, error) {
	if filters == nil {
		return nil, nil
	}
	converted := make([]resourceschedulersdk.ResourceFilter, 0, len(filters))
	for index, filter := range filters {
		sdkFilter, err := scheduleResourceFilterFromSpec(filter)
		if err != nil {
			return nil, fmt.Errorf("resourceFilters[%d]: %w", index, err)
		}
		converted = append(converted, sdkFilter)
	}
	return converted, nil
}

func scheduleResourceFilterFromSpec(filter resourceschedulerv1beta1.ScheduleResourceFilter) (resourceschedulersdk.ResourceFilter, error) {
	if raw := strings.TrimSpace(filter.JsonData); raw != "" {
		return scheduleResourceFilterFromJSON(raw)
	}

	attribute := scheduleNormalizeEnum(filter.Attribute)
	if attribute == "" {
		return nil, fmt.Errorf("attribute is required")
	}
	return scheduleResourceFilterForAttribute(attribute, filter)
}

func scheduleResourceFilterForAttribute(
	attribute string,
	filter resourceschedulerv1beta1.ScheduleResourceFilter,
) (resourceschedulersdk.ResourceFilter, error) {
	switch attribute {
	case string(resourceschedulersdk.ResourceFilterAttributeCompartmentId):
		return resourceschedulersdk.CompartmentIdResourceFilter{
			Value:                          scheduleOptionalString(filter.Value),
			ShouldIncludeChildCompartments: common.Bool(filter.ShouldIncludeChildCompartments),
		}, nil
	case string(resourceschedulersdk.ResourceFilterAttributeTimeCreated):
		condition, err := scheduleTimeCreatedCondition(filter.Condition)
		if err != nil {
			return nil, err
		}
		return resourceschedulersdk.TimeCreatedResourceFilter{
			Value:     scheduleOptionalString(filter.Value),
			Condition: condition,
		}, nil
	case string(resourceschedulersdk.ResourceFilterAttributeResourceType):
		return scheduleResourceTypeFilter(filter.Value)
	case string(resourceschedulersdk.ResourceFilterAttributeLifecycleState):
		return scheduleLifecycleStateFilter(filter.Value)
	case string(resourceschedulersdk.ResourceFilterAttributeDefinedTags):
		return nil, fmt.Errorf("defined tags filters require jsonData")
	default:
		return nil, fmt.Errorf("attribute %q is unsupported", filter.Attribute)
	}
}

func scheduleResourceTypeFilter(value string) (resourceschedulersdk.ResourceFilter, error) {
	values := scheduleStringListFromValue(value)
	if len(values) == 0 {
		return nil, fmt.Errorf("resource type value is required")
	}
	return resourceschedulersdk.ResourceTypeResourceFilter{Value: values}, nil
}

func scheduleLifecycleStateFilter(value string) (resourceschedulersdk.ResourceFilter, error) {
	values := scheduleStringListFromValue(value)
	if len(values) == 0 {
		return nil, fmt.Errorf("lifecycle state value is required")
	}
	return resourceschedulersdk.LifecycleStateResourceFilter{Value: values}, nil
}

func scheduleResourceFilterFromJSON(raw string) (resourceschedulersdk.ResourceFilter, error) {
	var probe struct {
		Attribute string `json:"attribute"`
	}
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return nil, fmt.Errorf("decode jsonData discriminator: %w", err)
	}
	switch scheduleNormalizeEnum(probe.Attribute) {
	case string(resourceschedulersdk.ResourceFilterAttributeCompartmentId):
		var filter resourceschedulersdk.CompartmentIdResourceFilter
		return filter, json.Unmarshal([]byte(raw), &filter)
	case string(resourceschedulersdk.ResourceFilterAttributeTimeCreated):
		var filter resourceschedulersdk.TimeCreatedResourceFilter
		return filter, json.Unmarshal([]byte(raw), &filter)
	case string(resourceschedulersdk.ResourceFilterAttributeResourceType):
		var filter resourceschedulersdk.ResourceTypeResourceFilter
		return filter, json.Unmarshal([]byte(raw), &filter)
	case string(resourceschedulersdk.ResourceFilterAttributeLifecycleState):
		var filter resourceschedulersdk.LifecycleStateResourceFilter
		return filter, json.Unmarshal([]byte(raw), &filter)
	case string(resourceschedulersdk.ResourceFilterAttributeDefinedTags):
		var filter resourceschedulersdk.DefinedTagsResourceFilter
		return filter, json.Unmarshal([]byte(raw), &filter)
	case "":
		return nil, fmt.Errorf("jsonData attribute is required")
	default:
		return nil, fmt.Errorf("jsonData attribute %q is unsupported", probe.Attribute)
	}
}

func scheduleResourcesFromSpec(resources []resourceschedulerv1beta1.ScheduleResource) ([]resourceschedulersdk.Resource, error) {
	if resources == nil {
		return nil, nil
	}
	converted := make([]resourceschedulersdk.Resource, 0, len(resources))
	for index, resource := range resources {
		if strings.TrimSpace(resource.Id) == "" {
			return nil, fmt.Errorf("resources[%d].id is required", index)
		}
		parameters, err := scheduleParametersFromSpec(resource.Parameters)
		if err != nil {
			return nil, fmt.Errorf("resources[%d]: %w", index, err)
		}
		converted = append(converted, resourceschedulersdk.Resource{
			Id:         common.String(resource.Id),
			Metadata:   maps.Clone(resource.Metadata),
			Parameters: parameters,
		})
	}
	return converted, nil
}

func scheduleParametersFromSpec(parameters []resourceschedulerv1beta1.ScheduleResourceParameter) ([]resourceschedulersdk.Parameter, error) {
	if parameters == nil {
		return nil, nil
	}
	converted := make([]resourceschedulersdk.Parameter, 0, len(parameters))
	for index, parameter := range parameters {
		sdkParameter, err := scheduleParameterFromSpec(parameter)
		if err != nil {
			return nil, fmt.Errorf("parameters[%d]: %w", index, err)
		}
		converted = append(converted, sdkParameter)
	}
	return converted, nil
}

func scheduleParameterFromSpec(parameter resourceschedulerv1beta1.ScheduleResourceParameter) (resourceschedulersdk.Parameter, error) {
	if raw := strings.TrimSpace(parameter.JsonData); raw != "" {
		return scheduleParameterFromJSON(raw)
	}

	value := maps.Clone(parameter.Value)
	switch scheduleNormalizeEnum(parameter.ParameterType) {
	case string(resourceschedulersdk.ParameterParameterTypeBody):
		var body any = value
		return resourceschedulersdk.BodyParameter{Value: &body}, nil
	case string(resourceschedulersdk.ParameterParameterTypeHeader):
		return resourceschedulersdk.HeaderParameter{Value: value}, nil
	case string(resourceschedulersdk.ParameterParameterTypePath):
		return resourceschedulersdk.PathParameter{Value: value}, nil
	case string(resourceschedulersdk.ParameterParameterTypeQuery):
		return resourceschedulersdk.QueryParameter{Value: value}, nil
	case "":
		return nil, fmt.Errorf("parameterType is required")
	default:
		return nil, fmt.Errorf("parameterType %q is unsupported", parameter.ParameterType)
	}
}

func scheduleParameterFromJSON(raw string) (resourceschedulersdk.Parameter, error) {
	var probe struct {
		ParameterType string `json:"parameterType"`
	}
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return nil, fmt.Errorf("decode jsonData discriminator: %w", err)
	}
	switch scheduleNormalizeEnum(probe.ParameterType) {
	case string(resourceschedulersdk.ParameterParameterTypeBody):
		var parameter resourceschedulersdk.BodyParameter
		return parameter, json.Unmarshal([]byte(raw), &parameter)
	case string(resourceschedulersdk.ParameterParameterTypeHeader):
		var parameter resourceschedulersdk.HeaderParameter
		return parameter, json.Unmarshal([]byte(raw), &parameter)
	case string(resourceschedulersdk.ParameterParameterTypePath):
		var parameter resourceschedulersdk.PathParameter
		return parameter, json.Unmarshal([]byte(raw), &parameter)
	case string(resourceschedulersdk.ParameterParameterTypeQuery):
		var parameter resourceschedulersdk.QueryParameter
		return parameter, json.Unmarshal([]byte(raw), &parameter)
	case "":
		return nil, fmt.Errorf("jsonData parameterType is required")
	default:
		return nil, fmt.Errorf("jsonData parameterType %q is unsupported", probe.ParameterType)
	}
}

func scheduleTimeCreatedCondition(value string) (resourceschedulersdk.TimeCreatedResourceFilterConditionEnum, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	condition, ok := resourceschedulersdk.GetMappingTimeCreatedResourceFilterConditionEnum(trimmed)
	if !ok {
		return "", fmt.Errorf("time created condition %q is unsupported", value)
	}
	return condition, nil
}

func scheduleSDKTimeFromSpec(field string, value string) (*common.SDKTime, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s must be RFC3339: %w", field, err)
	}
	return &common.SDKTime{Time: parsed}, nil
}

func scheduleDefinedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		converted[namespace] = make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[namespace][key] = value
		}
	}
	return converted
}

func wrapScheduleReadAndDeleteCalls(hooks *ScheduleRuntimeHooks) {
	if hooks.Get.Call != nil {
		call := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request resourceschedulersdk.GetScheduleRequest) (resourceschedulersdk.GetScheduleResponse, error) {
			response, err := call(ctx, request)
			if err == nil {
				normalizeScheduleResponse(&response.Schedule)
			}
			return response, err
		}
	}
	if hooks.Create.Call != nil {
		call := hooks.Create.Call
		hooks.Create.Call = func(ctx context.Context, request resourceschedulersdk.CreateScheduleRequest) (resourceschedulersdk.CreateScheduleResponse, error) {
			response, err := call(ctx, request)
			if err == nil {
				normalizeScheduleResponse(&response.Schedule)
			}
			return response, err
		}
	}
	if hooks.List.Call != nil {
		call := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request resourceschedulersdk.ListSchedulesRequest) (resourceschedulersdk.ListSchedulesResponse, error) {
			response, err := listSchedulePages(ctx, call, request)
			if err != nil {
				return resourceschedulersdk.ListSchedulesResponse{}, err
			}
			for index := range response.Items {
				normalizeScheduleSummaryResponse(&response.Items[index])
			}
			return response, nil
		}
	}
}

func listSchedulePages(
	ctx context.Context,
	call func(context.Context, resourceschedulersdk.ListSchedulesRequest) (resourceschedulersdk.ListSchedulesResponse, error),
	request resourceschedulersdk.ListSchedulesRequest,
) (resourceschedulersdk.ListSchedulesResponse, error) {
	var combined resourceschedulersdk.ListSchedulesResponse
	seenPages := map[string]bool{}
	for {
		response, err := call(ctx, request)
		if err != nil {
			return resourceschedulersdk.ListSchedulesResponse{}, err
		}
		if combined.RawResponse == nil {
			combined.RawResponse = response.RawResponse
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)
		nextPage := strings.TrimSpace(scheduleStringValue(response.OpcNextPage))
		if nextPage == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		if seenPages[nextPage] {
			return resourceschedulersdk.ListSchedulesResponse{}, fmt.Errorf("schedule list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = true
		combined.OpcNextPage = response.OpcNextPage
		request.Page = response.OpcNextPage
	}
}

func (c *scheduleRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *resourceschedulerv1beta1.Schedule,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("schedule runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *scheduleRuntimeClient) Delete(ctx context.Context, resource *resourceschedulerv1beta1.Schedule) (bool, error) {
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("schedule runtime client is not configured")
	}
	if deleted, err, handled := c.deleteUntrackedNoMatch(ctx, resource); handled {
		return deleted, err
	}
	if err := c.rejectAuthShapedPreDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *scheduleRuntimeClient) deleteUntrackedNoMatch(
	ctx context.Context,
	resource *resourceschedulerv1beta1.Schedule,
) (bool, error, bool) {
	if currentScheduleID(resource) != "" || c.confirmRead == nil {
		return false, nil, false
	}
	_, err := c.confirmRead(ctx, resource, "")
	if err == nil {
		return false, nil, false
	}
	var noMatch scheduleNoMatchConfirmRead
	if !errors.As(err, &noMatch) {
		return false, err, true
	}
	markScheduleDeleted(resource, noMatch.message)
	return true, nil, true
}

func (c *scheduleRuntimeClient) rejectAuthShapedPreDeleteRead(
	ctx context.Context,
	resource *resourceschedulerv1beta1.Schedule,
) error {
	currentID := currentScheduleID(resource)
	if currentID == "" || c.get == nil {
		return nil
	}
	_, err := c.get(ctx, resourceschedulersdk.GetScheduleRequest{ScheduleId: common.String(currentID)})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return fmt.Errorf("schedule delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %v", err)
}

func currentScheduleID(resource *resourceschedulerv1beta1.Schedule) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func handleScheduleDeleteError(resource *resourceschedulerv1beta1.Schedule, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return fmt.Errorf("schedule delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
	}
	return err
}

func confirmScheduleDeleteRead(
	ctx context.Context,
	hooks *ScheduleRuntimeHooks,
	resource *resourceschedulerv1beta1.Schedule,
	currentID string,
) (any, error) {
	if hooks == nil {
		return nil, fmt.Errorf("confirm Schedule delete: runtime hooks are nil")
	}
	if currentID = strings.TrimSpace(currentID); currentID != "" {
		return confirmScheduleDeleteReadByID(ctx, hooks, currentID)
	}
	return confirmScheduleDeleteReadByIdentity(ctx, hooks, resource)
}

func confirmScheduleDeleteReadByID(
	ctx context.Context,
	hooks *ScheduleRuntimeHooks,
	currentID string,
) (any, error) {
	if hooks.Get.Call == nil {
		return nil, fmt.Errorf("confirm Schedule delete: get hook is not configured")
	}
	response, err := hooks.Get.Call(ctx, resourceschedulersdk.GetScheduleRequest{
		ScheduleId: common.String(currentID),
	})
	return scheduleDeleteConfirmReadResponse(response, err)
}

func confirmScheduleDeleteReadByIdentity(
	ctx context.Context,
	hooks *ScheduleRuntimeHooks,
	resource *resourceschedulerv1beta1.Schedule,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("confirm Schedule delete: resource is nil")
	}
	if hooks.List.Call == nil {
		return nil, fmt.Errorf("confirm Schedule delete: list hook is not configured")
	}

	response, err := hooks.List.Call(ctx, resourceschedulersdk.ListSchedulesRequest{
		CompartmentId: common.String(resource.Spec.CompartmentId),
		DisplayName:   scheduleOptionalString(resource.Spec.DisplayName),
	})
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return nil, scheduleAuthShapedConfirmRead{err: err}
		}
		return nil, err
	}

	matches := make([]resourceschedulersdk.ScheduleSummary, 0, len(response.Items))
	for _, item := range response.Items {
		if scheduleSummaryMatchesSpec(item, resource.Spec) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return nil, scheduleNoMatchConfirmRead{message: "Schedule delete confirmation did not find a matching OCI schedule"}
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("schedule list response returned multiple matching resources for compartmentId %q and displayName %q", resource.Spec.CompartmentId, resource.Spec.DisplayName)
	}
}

func scheduleDeleteConfirmReadResponse(response any, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return scheduleAuthShapedConfirmRead{err: err}, nil
	}
	return nil, err
}

func applyScheduleDeleteConfirmOutcome(
	resource *resourceschedulerv1beta1.Schedule,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	switch typed := response.(type) {
	case scheduleAuthShapedConfirmRead:
		recordScheduleConfirmReadRequestID(resource, typed)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, typed
	case *scheduleAuthShapedConfirmRead:
		if typed != nil {
			recordScheduleConfirmReadRequestID(resource, *typed)
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, *typed
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func recordScheduleConfirmReadRequestID(resource *resourceschedulerv1beta1.Schedule, err scheduleAuthShapedConfirmRead) {
	if resource == nil {
		return
	}
	servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, err.GetOpcRequestID())
}

func scheduleSummaryMatchesSpec(summary resourceschedulersdk.ScheduleSummary, spec resourceschedulerv1beta1.ScheduleSpec) bool {
	if strings.TrimSpace(spec.CompartmentId) != "" && scheduleStringValue(summary.CompartmentId) != strings.TrimSpace(spec.CompartmentId) {
		return false
	}
	if strings.TrimSpace(spec.DisplayName) != "" && scheduleStringValue(summary.DisplayName) != strings.TrimSpace(spec.DisplayName) {
		return false
	}
	return true
}

func markScheduleDeleted(resource *resourceschedulerv1beta1.Schedule, message string) {
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
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

type scheduleProjectedResourceFilter struct {
	JsonData                       string `json:"jsonData,omitempty"`
	Attribute                      string `json:"attribute,omitempty"`
	Value                          string `json:"value,omitempty"`
	Condition                      string `json:"condition,omitempty"`
	ShouldIncludeChildCompartments bool   `json:"shouldIncludeChildCompartments,omitempty"`
}

type scheduleProjectedParameter struct {
	JsonData      string            `json:"jsonData,omitempty"`
	ParameterType string            `json:"parameterType,omitempty"`
	Value         map[string]string `json:"value,omitempty"`
}

func normalizeScheduleResponse(schedule *resourceschedulersdk.Schedule) {
	if schedule == nil {
		return
	}
	schedule.ResourceFilters = scheduleProjectedResourceFilters(schedule.ResourceFilters)
	for index := range schedule.Resources {
		schedule.Resources[index].Parameters = scheduleProjectedParameters(schedule.Resources[index].Parameters)
	}
}

func normalizeScheduleSummaryResponse(summary *resourceschedulersdk.ScheduleSummary) {
	if summary == nil {
		return
	}
	summary.ResourceFilters = scheduleProjectedResourceFilters(summary.ResourceFilters)
	for index := range summary.Resources {
		summary.Resources[index].Parameters = scheduleProjectedParameters(summary.Resources[index].Parameters)
	}
}

func scheduleProjectedResourceFilters(filters []resourceschedulersdk.ResourceFilter) []resourceschedulersdk.ResourceFilter {
	if filters == nil {
		return nil
	}
	projected := make([]resourceschedulersdk.ResourceFilter, 0, len(filters))
	for _, filter := range filters {
		projected = append(projected, scheduleProjectedResourceFilterFromSDK(filter))
	}
	return projected
}

func scheduleProjectedResourceFilterFromSDK(filter resourceschedulersdk.ResourceFilter) resourceschedulersdk.ResourceFilter {
	switch concrete := filter.(type) {
	case resourceschedulersdk.CompartmentIdResourceFilter:
		return scheduleProjectedResourceFilter{
			Attribute:                      string(resourceschedulersdk.ResourceFilterAttributeCompartmentId),
			Value:                          scheduleStringValue(concrete.Value),
			ShouldIncludeChildCompartments: scheduleBoolValue(concrete.ShouldIncludeChildCompartments),
		}
	case resourceschedulersdk.TimeCreatedResourceFilter:
		return scheduleProjectedResourceFilter{
			Attribute: string(resourceschedulersdk.ResourceFilterAttributeTimeCreated),
			Value:     scheduleStringValue(concrete.Value),
			Condition: string(concrete.Condition),
		}
	case resourceschedulersdk.ResourceTypeResourceFilter:
		return scheduleProjectedResourceFilter{
			Attribute: string(resourceschedulersdk.ResourceFilterAttributeResourceType),
			JsonData:  scheduleRawJSON(concrete),
		}
	case resourceschedulersdk.LifecycleStateResourceFilter:
		return scheduleProjectedResourceFilter{
			Attribute: string(resourceschedulersdk.ResourceFilterAttributeLifecycleState),
			JsonData:  scheduleRawJSON(concrete),
		}
	case resourceschedulersdk.DefinedTagsResourceFilter:
		return scheduleProjectedResourceFilter{
			Attribute: string(resourceschedulersdk.ResourceFilterAttributeDefinedTags),
			JsonData:  scheduleRawJSON(concrete),
		}
	case scheduleProjectedResourceFilter:
		return concrete
	default:
		return scheduleProjectedResourceFilter{JsonData: scheduleRawJSON(filter)}
	}
}

func scheduleCanonicalResourceFilters(filters []resourceschedulersdk.ResourceFilter) []resourceschedulersdk.ResourceFilter {
	if filters == nil {
		return nil
	}
	canonical := make([]resourceschedulersdk.ResourceFilter, 0, len(filters))
	for _, filter := range filters {
		canonical = append(canonical, scheduleCanonicalResourceFilter(filter))
	}
	return canonical
}

func scheduleCanonicalResourceFilter(filter resourceschedulersdk.ResourceFilter) resourceschedulersdk.ResourceFilter {
	if projected, ok := filter.(scheduleProjectedResourceFilter); ok {
		if raw := strings.TrimSpace(projected.JsonData); raw != "" {
			if decoded, err := scheduleResourceFilterFromJSON(raw); err == nil {
				return decoded
			}
		}
		decoded, err := scheduleResourceFilterFromSpec(resourceschedulerv1beta1.ScheduleResourceFilter{
			Attribute:                      projected.Attribute,
			Value:                          projected.Value,
			Condition:                      projected.Condition,
			ShouldIncludeChildCompartments: projected.ShouldIncludeChildCompartments,
		})
		if err == nil {
			return decoded
		}
	}
	return filter
}

func scheduleProjectedParameters(parameters []resourceschedulersdk.Parameter) []resourceschedulersdk.Parameter {
	if parameters == nil {
		return nil
	}
	projected := make([]resourceschedulersdk.Parameter, 0, len(parameters))
	for _, parameter := range parameters {
		projected = append(projected, scheduleProjectedParameterFromSDK(parameter))
	}
	return projected
}

func scheduleProjectedParameterFromSDK(parameter resourceschedulersdk.Parameter) resourceschedulersdk.Parameter {
	switch concrete := parameter.(type) {
	case resourceschedulersdk.HeaderParameter:
		return scheduleProjectedParameter{
			ParameterType: string(resourceschedulersdk.ParameterParameterTypeHeader),
			Value:         maps.Clone(concrete.Value),
		}
	case resourceschedulersdk.PathParameter:
		return scheduleProjectedParameter{
			ParameterType: string(resourceschedulersdk.ParameterParameterTypePath),
			Value:         maps.Clone(concrete.Value),
		}
	case resourceschedulersdk.QueryParameter:
		return scheduleProjectedParameter{
			ParameterType: string(resourceschedulersdk.ParameterParameterTypeQuery),
			Value:         maps.Clone(concrete.Value),
		}
	case resourceschedulersdk.BodyParameter:
		if value, ok := scheduleStringMapFromBodyParameter(concrete); ok {
			return scheduleProjectedParameter{
				ParameterType: string(resourceschedulersdk.ParameterParameterTypeBody),
				Value:         value,
			}
		}
		return scheduleProjectedParameter{
			ParameterType: string(resourceschedulersdk.ParameterParameterTypeBody),
			JsonData:      scheduleRawJSON(concrete),
		}
	case scheduleProjectedParameter:
		return concrete
	default:
		return scheduleProjectedParameter{JsonData: scheduleRawJSON(parameter)}
	}
}

func scheduleCanonicalResources(resources []resourceschedulersdk.Resource) []resourceschedulersdk.Resource {
	if resources == nil {
		return nil
	}
	canonical := make([]resourceschedulersdk.Resource, 0, len(resources))
	for _, resource := range resources {
		resource.Parameters = scheduleCanonicalParameters(resource.Parameters)
		canonical = append(canonical, resource)
	}
	return canonical
}

func scheduleCanonicalParameters(parameters []resourceschedulersdk.Parameter) []resourceschedulersdk.Parameter {
	if parameters == nil {
		return nil
	}
	canonical := make([]resourceschedulersdk.Parameter, 0, len(parameters))
	for _, parameter := range parameters {
		canonical = append(canonical, scheduleCanonicalParameter(parameter))
	}
	return canonical
}

func scheduleCanonicalParameter(parameter resourceschedulersdk.Parameter) resourceschedulersdk.Parameter {
	if projected, ok := parameter.(scheduleProjectedParameter); ok {
		if raw := strings.TrimSpace(projected.JsonData); raw != "" {
			if decoded, err := scheduleParameterFromJSON(raw); err == nil {
				return decoded
			}
		}
		decoded, err := scheduleParameterFromSpec(resourceschedulerv1beta1.ScheduleResourceParameter{
			ParameterType: projected.ParameterType,
			Value:         projected.Value,
		})
		if err == nil {
			return decoded
		}
	}
	return parameter
}

func scheduleStringMapFromBodyParameter(parameter resourceschedulersdk.BodyParameter) (map[string]string, bool) {
	if parameter.Value == nil {
		return nil, false
	}
	raw, ok := (*parameter.Value).(map[string]string)
	if ok {
		return maps.Clone(raw), true
	}
	payload, err := json.Marshal(*parameter.Value)
	if err != nil {
		return nil, false
	}
	var converted map[string]string
	if err := json.Unmarshal(payload, &converted); err != nil {
		return nil, false
	}
	return converted, true
}

func scheduleFromResponse(response any) (resourceschedulersdk.Schedule, bool) {
	switch current := response.(type) {
	case resourceschedulersdk.Schedule:
		return current, true
	case *resourceschedulersdk.Schedule:
		if current == nil {
			return resourceschedulersdk.Schedule{}, false
		}
		return *current, true
	case resourceschedulersdk.ScheduleSummary:
		return scheduleFromSummary(current), true
	case *resourceschedulersdk.ScheduleSummary:
		if current == nil {
			return resourceschedulersdk.Schedule{}, false
		}
		return scheduleFromSummary(*current), true
	case resourceschedulersdk.CreateScheduleResponse:
		return current.Schedule, true
	case resourceschedulersdk.GetScheduleResponse:
		return current.Schedule, true
	case resourceschedulersdk.UpdateScheduleResponse:
		return resourceschedulersdk.Schedule{}, false
	default:
		return resourceschedulersdk.Schedule{}, false
	}
}

func scheduleFromSummary(summary resourceschedulersdk.ScheduleSummary) resourceschedulersdk.Schedule {
	return resourceschedulersdk.Schedule{
		Id:                summary.Id,
		CompartmentId:     summary.CompartmentId,
		DisplayName:       summary.DisplayName,
		Action:            resourceschedulersdk.ScheduleActionEnum(summary.Action),
		RecurrenceDetails: summary.RecurrenceDetails,
		RecurrenceType:    resourceschedulersdk.ScheduleRecurrenceTypeEnum(summary.RecurrenceType),
		TimeCreated:       summary.TimeCreated,
		LifecycleState:    summary.LifecycleState,
		FreeformTags:      summary.FreeformTags,
		DefinedTags:       summary.DefinedTags,
		Description:       summary.Description,
		ResourceFilters:   summary.ResourceFilters,
		Resources:         summary.Resources,
		TimeStarts:        summary.TimeStarts,
		TimeEnds:          summary.TimeEnds,
		TimeUpdated:       summary.TimeUpdated,
		TimeLastRun:       summary.TimeLastRun,
		TimeNextRun:       summary.TimeNextRun,
		LastRunStatus:     summary.LastRunStatus,
		SystemTags:        summary.SystemTags,
	}
}

func scheduleStringUpdate(desired string, current *string) (*string, bool) {
	trimmed := strings.TrimSpace(desired)
	if trimmed == "" || trimmed == scheduleStringValue(current) {
		return nil, false
	}
	return common.String(desired), true
}

func scheduleOptionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(value)
}

func scheduleStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func scheduleBoolValue(value *bool) bool {
	return value != nil && *value
}

func scheduleNormalizeEnum(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func scheduleStringListFromValue(value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			values = append(values, item)
		}
	}
	return values
}

func scheduleRawJSON(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(payload)
}

func scheduleJSONEqual(left any, right any) bool {
	var leftJSON any
	var rightJSON any
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}
	if err := json.Unmarshal(leftPayload, &leftJSON); err != nil {
		return false
	}
	if err := json.Unmarshal(rightPayload, &rightJSON); err != nil {
		return false
	}
	return reflect.DeepEqual(leftJSON, rightJSON)
}
