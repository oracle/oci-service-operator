/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package rule

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	eventssdk "github.com/oracle/oci-go-sdk/v65/events"
	eventsv1beta1 "github.com/oracle/oci-service-operator/api/events/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ruleIdentity struct {
	compartmentID string
	displayName   string
}

type ruleListCall func(context.Context, eventssdk.ListRulesRequest) (eventssdk.ListRulesResponse, error)

func init() {
	registerRuleRuntimeHooksMutator(func(_ *RuleServiceManager, hooks *RuleRuntimeHooks) {
		applyRuleRuntimeHooks(hooks)
	})
}

func applyRuleRuntimeHooks(hooks *RuleRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newRuleRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *eventsv1beta1.Rule, _ string) (any, error) {
		return buildRuleCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(_ context.Context, resource *eventsv1beta1.Rule, _ string, currentResponse any) (any, bool, error) {
		return buildRuleUpdateBody(resource, currentResponse)
	}
	hooks.Identity = generatedruntime.IdentityHooks[*eventsv1beta1.Rule]{
		Resolve: func(resource *eventsv1beta1.Rule) (any, error) {
			return resolveRuleIdentity(resource)
		},
		LookupExisting: func(ctx context.Context, resource *eventsv1beta1.Rule, identity any) (any, error) {
			return lookupExistingRule(ctx, hooks, resource, identity)
		},
	}
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, resource *eventsv1beta1.Rule, currentID string) (any, error) {
		return confirmRuleDeleteRead(ctx, hooks, resource, currentID)
	}
	hooks.DeleteHooks.HandleError = handleRuleDeleteError
	hooks.DeleteHooks.ApplyOutcome = handleRuleDeleteConfirmReadOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate RuleServiceClient) RuleServiceClient {
		return ruleDeleteFallbackClient{
			delegate: delegate,
			list:     hooks.List.Call,
		}
	})
}

type ruleDeleteFallbackClient struct {
	delegate RuleServiceClient
	list     ruleListCall
}

func (c ruleDeleteFallbackClient) CreateOrUpdate(
	ctx context.Context,
	resource *eventsv1beta1.Rule,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c ruleDeleteFallbackClient) Delete(ctx context.Context, resource *eventsv1beta1.Rule) (bool, error) {
	if !ruleDeleteFallbackEnabled(resource, c.list) {
		return c.delegate.Delete(ctx, resource)
	}

	summary, found, err := resolveRuleDeleteFallbackSummary(ctx, c.list, resource)
	if err != nil {
		return false, err
	}
	if !found {
		markRuleDeleted(resource, "OCI resource no longer exists")
		return true, nil
	}
	return c.deleteRuleFallbackSummary(ctx, resource, summary)
}

func ruleDeleteFallbackEnabled(resource *eventsv1beta1.Rule, list ruleListCall) bool {
	return resource != nil &&
		strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) == "" &&
		list != nil
}

func resolveRuleDeleteFallbackSummary(
	ctx context.Context,
	list ruleListCall,
	resource *eventsv1beta1.Rule,
) (eventssdk.RuleSummary, bool, error) {
	identity, err := resolveRuleIdentity(resource)
	if err != nil {
		return eventssdk.RuleSummary{}, false, err
	}
	matches, err := listMatchingRulesForDelete(ctx, list, identity)
	if err != nil {
		return eventssdk.RuleSummary{}, false, ruleDeleteFallbackListError(resource, err)
	}
	return singleRuleSummaryMatch(matches, identity)
}

func (c ruleDeleteFallbackClient) deleteRuleFallbackSummary(
	ctx context.Context,
	resource *eventsv1beta1.Rule,
	summary eventssdk.RuleSummary,
) (bool, error) {
	if ruleSummaryDeleted(summary) {
		markRuleDeleted(resource, "OCI resource deleted")
		return true, nil
	}
	currentID := stringPointerValue(summary.Id)
	if currentID == "" {
		return false, fmt.Errorf("rule delete fallback could not resolve a resource OCID")
	}
	resource.Status.OsokStatus.Ocid = shared.OCID(currentID)
	if ruleSummaryDeleting(summary) {
		markRuleDeletePending(resource, summary)
		return false, nil
	}
	return c.delegate.Delete(ctx, resource)
}

func ruleDeleteFallbackListError(resource *eventsv1beta1.Rule, err error) error {
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		}
		return ruleDeleteAuthShapedConfirmRead{err: err}
	}
	return err
}

func markRuleDeleted(resource *eventsv1beta1.Rule, message string) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = nil
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, servicemanager.RuntimeDeps{}.Log)
}

func markRuleDeletePending(resource *eventsv1beta1.Rule, summary eventssdk.RuleSummary) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	message := "OCI resource delete is in progress"
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       string(summary.LifecycleState),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	if currentID := stringPointerValue(summary.Id); currentID != "" {
		status.Ocid = shared.OCID(currentID)
		resource.Status.Id = currentID
	}
	resource.Status.LifecycleState = string(summary.LifecycleState)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, servicemanager.RuntimeDeps{}.Log)
}

func newRuleRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:       "events",
		FormalSlug:          "rule",
		StatusProjection:    "required",
		SecretSideEffects:   "none",
		FinalizerPolicy:     "retain-until-confirmed-delete",
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE", "INACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"isEnabled",
				"condition",
				"actions",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"compartmentId"},
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
	}
}

func buildRuleCreateBody(resource *eventsv1beta1.Rule) (eventssdk.CreateRuleDetails, error) {
	if resource == nil {
		return eventssdk.CreateRuleDetails{}, fmt.Errorf("rule resource is nil")
	}
	actions, err := ruleSDKActionDetailsList(resource.Spec.Actions)
	if err != nil {
		return eventssdk.CreateRuleDetails{}, err
	}
	return eventssdk.CreateRuleDetails{
		DisplayName:   stringPointer(resource.Spec.DisplayName),
		IsEnabled:     boolPointer(resource.Spec.IsEnabled),
		Condition:     stringPointer(resource.Spec.Condition),
		CompartmentId: stringPointer(resource.Spec.CompartmentId),
		Actions:       actions,
		Description:   stringPointer(resource.Spec.Description),
		FreeformTags:  cloneStringMap(resource.Spec.FreeformTags),
		DefinedTags:   ruleDefinedTags(resource.Spec.DefinedTags),
	}, nil
}

func buildRuleUpdateBody(resource *eventsv1beta1.Rule, currentResponse any) (eventssdk.UpdateRuleDetails, bool, error) {
	if resource == nil {
		return eventssdk.UpdateRuleDetails{}, false, fmt.Errorf("rule resource is nil")
	}
	current, err := ruleRuntimeBody(currentResponse)
	if err != nil {
		return eventssdk.UpdateRuleDetails{}, false, err
	}

	details := eventssdk.UpdateRuleDetails{}
	updateNeeded := applyRuleScalarUpdates(&details, resource, current)
	actionsUpdated, err := applyRuleActionsUpdate(&details, resource, current)
	if err != nil {
		return eventssdk.UpdateRuleDetails{}, false, err
	}
	tagsUpdated, err := applyRuleTagsUpdate(&details, resource, current)
	if err != nil {
		return eventssdk.UpdateRuleDetails{}, false, err
	}
	return details, updateNeeded || actionsUpdated || tagsUpdated, nil
}

func applyRuleScalarUpdates(
	details *eventssdk.UpdateRuleDetails,
	resource *eventsv1beta1.Rule,
	current eventssdk.Rule,
) bool {
	updateNeeded := false
	if desired, ok := desiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := desiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := desiredBoolUpdate(resource.Spec.IsEnabled, current.IsEnabled); ok {
		details.IsEnabled = desired
		updateNeeded = true
	}
	if desired, ok := desiredStringUpdate(resource.Spec.Condition, current.Condition); ok {
		details.Condition = desired
		updateNeeded = true
	}
	return updateNeeded
}

func applyRuleActionsUpdate(
	details *eventssdk.UpdateRuleDetails,
	resource *eventsv1beta1.Rule,
	current eventssdk.Rule,
) (bool, error) {
	desiredActions, err := ruleSDKActionDetailsList(resource.Spec.Actions)
	if err != nil {
		return false, err
	}
	currentActions, err := ruleSDKActionDetailsListFromCurrent(current.Actions)
	if err != nil {
		return false, err
	}
	actionsEqual, err := ruleJSONEqual(desiredActions, currentActions)
	if err != nil {
		return false, err
	}
	if actionsEqual {
		return false, nil
	}
	details.Actions = desiredActions
	return true, nil
}

func applyRuleTagsUpdate(
	details *eventssdk.UpdateRuleDetails,
	resource *eventsv1beta1.Rule,
	current eventssdk.Rule,
) (bool, error) {
	updateNeeded := false
	if resource.Spec.FreeformTags != nil && !maps.Equal(resource.Spec.FreeformTags, current.FreeformTags) {
		details.FreeformTags = cloneStringMap(resource.Spec.FreeformTags)
		updateNeeded = true
	}
	if resource.Spec.DefinedTags != nil {
		desired := ruleDefinedTags(resource.Spec.DefinedTags)
		tagsEqual, err := ruleJSONEqual(desired, current.DefinedTags)
		if err != nil {
			return false, err
		}
		if !tagsEqual {
			details.DefinedTags = desired
			updateNeeded = true
		}
	}
	return updateNeeded, nil
}

func ruleRuntimeBody(currentResponse any) (eventssdk.Rule, error) {
	switch current := currentResponse.(type) {
	case eventssdk.Rule:
		return current, nil
	case *eventssdk.Rule:
		if current == nil {
			return eventssdk.Rule{}, fmt.Errorf("current Rule response is nil")
		}
		return *current, nil
	case eventssdk.RuleSummary:
		return ruleFromSummary(current), nil
	case eventssdk.CreateRuleResponse:
		return current.Rule, nil
	case eventssdk.GetRuleResponse:
		return current.Rule, nil
	case eventssdk.UpdateRuleResponse:
		return current.Rule, nil
	default:
		return ruleRuntimeBodyPointer(currentResponse)
	}
}

func ruleRuntimeBodyPointer(currentResponse any) (eventssdk.Rule, error) {
	switch current := currentResponse.(type) {
	case *eventssdk.RuleSummary:
		if current == nil {
			return eventssdk.Rule{}, fmt.Errorf("current rule response is nil")
		}
		return ruleFromSummary(*current), nil
	case *eventssdk.CreateRuleResponse:
		if current == nil {
			return eventssdk.Rule{}, fmt.Errorf("current rule response is nil")
		}
		return current.Rule, nil
	case *eventssdk.GetRuleResponse:
		if current == nil {
			return eventssdk.Rule{}, fmt.Errorf("current rule response is nil")
		}
		return current.Rule, nil
	case *eventssdk.UpdateRuleResponse:
		if current == nil {
			return eventssdk.Rule{}, fmt.Errorf("current rule response is nil")
		}
		return current.Rule, nil
	default:
		return eventssdk.Rule{}, fmt.Errorf("unexpected current rule response type %T", currentResponse)
	}
}

func ruleFromSummary(summary eventssdk.RuleSummary) eventssdk.Rule {
	return eventssdk.Rule{
		Id:             summary.Id,
		DisplayName:    summary.DisplayName,
		LifecycleState: summary.LifecycleState,
		Condition:      summary.Condition,
		CompartmentId:  summary.CompartmentId,
		IsEnabled:      summary.IsEnabled,
		TimeCreated:    summary.TimeCreated,
		Description:    summary.Description,
		FreeformTags:   cloneStringMap(summary.FreeformTags),
		DefinedTags:    cloneDefinedTags(summary.DefinedTags),
	}
}

func resolveRuleIdentity(resource *eventsv1beta1.Rule) (ruleIdentity, error) {
	if resource == nil {
		return ruleIdentity{}, fmt.Errorf("resolve Rule identity: resource is nil")
	}
	identity := ruleIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		displayName:   strings.TrimSpace(resource.Spec.DisplayName),
	}
	if identity.compartmentID == "" {
		return ruleIdentity{}, fmt.Errorf("resolve Rule identity: compartmentId is required")
	}
	if identity.displayName == "" {
		return ruleIdentity{}, fmt.Errorf("resolve Rule identity: displayName is required")
	}
	return identity, nil
}

func lookupExistingRule(
	ctx context.Context,
	hooks *RuleRuntimeHooks,
	_ *eventsv1beta1.Rule,
	identity any,
) (any, error) {
	typedIdentity, err := ruleIdentityFromValue(identity)
	if err != nil {
		return nil, err
	}
	if hooks == nil || hooks.List.Call == nil {
		return nil, nil
	}

	matches, err := listMatchingRules(ctx, hooks.List.Call, typedIdentity)
	if err != nil {
		return nil, err
	}
	return singleRuleMatch(matches, typedIdentity)
}

func ruleIdentityFromValue(identity any) (ruleIdentity, error) {
	typedIdentity, ok := identity.(ruleIdentity)
	if !ok {
		return ruleIdentity{}, fmt.Errorf("resolve Rule identity: expected ruleIdentity, got %T", identity)
	}
	return typedIdentity, nil
}

func listMatchingRules(ctx context.Context, list ruleListCall, identity ruleIdentity) ([]eventssdk.RuleSummary, error) {
	return listMatchingRulesWithPredicate(ctx, list, identity, ruleSummaryMatchesIdentity)
}

func listMatchingRulesForDelete(ctx context.Context, list ruleListCall, identity ruleIdentity) ([]eventssdk.RuleSummary, error) {
	return listMatchingRulesWithPredicate(ctx, list, identity, ruleSummaryMatchesDeleteIdentity)
}

func listMatchingRulesWithPredicate(
	ctx context.Context,
	list ruleListCall,
	identity ruleIdentity,
	matchesIdentity func(eventssdk.RuleSummary, ruleIdentity) bool,
) ([]eventssdk.RuleSummary, error) {
	var matches []eventssdk.RuleSummary
	var page *string
	for {
		response, err := list(ctx, eventssdk.ListRulesRequest{
			CompartmentId: stringPointer(identity.compartmentID),
			DisplayName:   stringPointer(identity.displayName),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range response.Items {
			if matchesIdentity(item, identity) {
				matches = append(matches, item)
			}
		}
		if ruleListPageDone(response.OpcNextPage) {
			break
		}
		page = response.OpcNextPage
	}
	return matches, nil
}

func ruleSummaryMatchesIdentity(summary eventssdk.RuleSummary, identity ruleIdentity) bool {
	return stringPointerValue(summary.CompartmentId) == identity.compartmentID &&
		stringPointerValue(summary.DisplayName) == identity.displayName &&
		ruleLifecycleUsableForBind(summary.LifecycleState)
}

func ruleSummaryMatchesDeleteIdentity(summary eventssdk.RuleSummary, identity ruleIdentity) bool {
	return stringPointerValue(summary.CompartmentId) == identity.compartmentID &&
		stringPointerValue(summary.DisplayName) == identity.displayName
}

func ruleLifecycleUsableForBind(state eventssdk.RuleLifecycleStateEnum) bool {
	switch state {
	case eventssdk.RuleLifecycleStateDeleting, eventssdk.RuleLifecycleStateDeleted, eventssdk.RuleLifecycleStateFailed:
		return false
	default:
		return true
	}
}

func ruleSummaryDeleted(summary eventssdk.RuleSummary) bool {
	return summary.LifecycleState == eventssdk.RuleLifecycleStateDeleted
}

func ruleSummaryDeleting(summary eventssdk.RuleSummary) bool {
	return summary.LifecycleState == eventssdk.RuleLifecycleStateDeleting
}

func ruleListPageDone(nextPage *string) bool {
	return nextPage == nil || strings.TrimSpace(*nextPage) == ""
}

func singleRuleMatch(matches []eventssdk.RuleSummary, identity ruleIdentity) (any, error) {
	summary, found, err := singleRuleSummaryMatch(matches, identity)
	if err != nil || !found {
		return nil, err
	}
	return summary, nil
}

func singleRuleSummaryMatch(matches []eventssdk.RuleSummary, identity ruleIdentity) (eventssdk.RuleSummary, bool, error) {
	switch len(matches) {
	case 0:
		return eventssdk.RuleSummary{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return eventssdk.RuleSummary{}, false, fmt.Errorf("rule list response returned multiple matching resources for compartmentId %q and displayName %q", identity.compartmentID, identity.displayName)
	}
}

func confirmRuleDeleteRead(
	ctx context.Context,
	hooks *RuleRuntimeHooks,
	resource *eventsv1beta1.Rule,
	currentID string,
) (any, error) {
	if hooks == nil {
		return nil, fmt.Errorf("confirm Rule delete: runtime hooks are nil")
	}
	if currentID = strings.TrimSpace(currentID); currentID != "" {
		return confirmRuleDeleteReadByID(ctx, hooks, currentID)
	}
	return confirmRuleDeleteReadByIdentity(ctx, hooks, resource)
}

func confirmRuleDeleteReadByID(ctx context.Context, hooks *RuleRuntimeHooks, currentID string) (any, error) {
	if hooks.Get.Call == nil {
		return nil, fmt.Errorf("confirm Rule delete: get hook is not configured")
	}
	response, err := hooks.Get.Call(ctx, eventssdk.GetRuleRequest{RuleId: rawStringPointer(currentID)})
	return ruleDeleteConfirmReadResponse(response, err)
}

func confirmRuleDeleteReadByIdentity(
	ctx context.Context,
	hooks *RuleRuntimeHooks,
	resource *eventsv1beta1.Rule,
) (any, error) {
	identity, err := resolveRuleIdentity(resource)
	if err != nil {
		return nil, err
	}
	if hooks.List.Call == nil {
		return nil, fmt.Errorf("confirm Rule delete: list hook is not configured")
	}
	matches, err := listMatchingRulesForDelete(ctx, hooks.List.Call, identity)
	if err != nil {
		return nil, ruleDeleteConfirmReadError(err)
	}
	summary, found, err := singleRuleSummaryMatch(matches, identity)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ruleDeleteConfirmReadNotFoundError()
	}
	if ruleSummaryDeleted(summary) {
		return nil, ruleDeleteConfirmReadNotFoundError()
	}
	return summary, nil
}

func ruleDeleteConfirmReadResponse(response any, err error) (any, error) {
	if err == nil {
		return response, nil
	}
	convertedErr := ruleDeleteConfirmReadError(err)
	if typed, ok := convertedErr.(ruleDeleteAuthShapedConfirmRead); ok {
		return typed, nil
	}
	return nil, convertedErr
}

func ruleDeleteConfirmReadError(err error) error {
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return ruleDeleteAuthShapedConfirmRead{err: err}
	}
	return err
}

func ruleDeleteConfirmReadNotFoundError() error {
	return errorutil.NotFoundOciError{
		HTTPStatusCode: 404,
		ErrorCode:      errorutil.NotFound,
		Description:    "Rule delete confirmation did not find a matching OCI rule",
	}
}

func ruleSDKActionDetailsList(actions eventsv1beta1.RuleActions) (*eventssdk.ActionDetailsList, error) {
	converted, err := ruleSDKActionDetails(actions.Actions)
	if err != nil {
		return nil, err
	}
	return &eventssdk.ActionDetailsList{Actions: converted}, nil
}

func ruleSDKActionDetails(actions []eventsv1beta1.RuleActionsAction) ([]eventssdk.ActionDetails, error) {
	if actions == nil {
		return nil, nil
	}
	converted := make([]eventssdk.ActionDetails, 0, len(actions))
	for index, action := range actions {
		details, err := ruleSDKActionDetailsItem(action)
		if err != nil {
			return nil, fmt.Errorf("convert rule action %d: %w", index, err)
		}
		converted = append(converted, details)
	}
	return converted, nil
}

func ruleSDKActionDetailsItem(action eventsv1beta1.RuleActionsAction) (eventssdk.ActionDetails, error) {
	if raw := strings.TrimSpace(action.JsonData); raw != "" {
		return ruleSDKActionDetailsFromJSON(raw)
	}

	actionType, err := ruleActionType(action)
	if err != nil {
		return nil, err
	}
	switch actionType {
	case string(eventssdk.ActionDetailsActionTypeOns):
		return eventssdk.CreateNotificationServiceActionDetails{
			IsEnabled:   boolPointer(action.IsEnabled),
			Description: stringPointer(action.Description),
			TopicId:     stringPointer(action.TopicId),
		}, nil
	case string(eventssdk.ActionDetailsActionTypeOss):
		return eventssdk.CreateStreamingServiceActionDetails{
			IsEnabled:   boolPointer(action.IsEnabled),
			Description: stringPointer(action.Description),
			StreamId:    stringPointer(action.StreamId),
		}, nil
	case string(eventssdk.ActionDetailsActionTypeFaas):
		return eventssdk.CreateFaaSActionDetails{
			IsEnabled:   boolPointer(action.IsEnabled),
			Description: stringPointer(action.Description),
			FunctionId:  stringPointer(action.FunctionId),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported actionType %q", action.ActionType)
	}
}

func ruleSDKActionDetailsFromJSON(raw string) (eventssdk.ActionDetails, error) {
	var discriminator struct {
		ActionType string `json:"actionType"`
	}
	if err := json.Unmarshal([]byte(raw), &discriminator); err != nil {
		return nil, fmt.Errorf("decode jsonData discriminator: %w", err)
	}
	actionType := strings.ToUpper(strings.TrimSpace(discriminator.ActionType))
	switch actionType {
	case string(eventssdk.ActionDetailsActionTypeOns):
		var details eventssdk.CreateNotificationServiceActionDetails
		if err := json.Unmarshal([]byte(raw), &details); err != nil {
			return nil, fmt.Errorf("decode ONS jsonData: %w", err)
		}
		return details, nil
	case string(eventssdk.ActionDetailsActionTypeOss):
		var details eventssdk.CreateStreamingServiceActionDetails
		if err := json.Unmarshal([]byte(raw), &details); err != nil {
			return nil, fmt.Errorf("decode OSS jsonData: %w", err)
		}
		return details, nil
	case string(eventssdk.ActionDetailsActionTypeFaas):
		var details eventssdk.CreateFaaSActionDetails
		if err := json.Unmarshal([]byte(raw), &details); err != nil {
			return nil, fmt.Errorf("decode FAAS jsonData: %w", err)
		}
		return details, nil
	case "":
		return nil, fmt.Errorf("jsonData actionType is required")
	default:
		return nil, fmt.Errorf("unsupported jsonData actionType %q", discriminator.ActionType)
	}
}

func ruleActionType(action eventsv1beta1.RuleActionsAction) (string, error) {
	actionType := strings.ToUpper(strings.TrimSpace(action.ActionType))
	if actionType != "" {
		return actionType, nil
	}

	var inferred []string
	if strings.TrimSpace(action.TopicId) != "" {
		inferred = append(inferred, string(eventssdk.ActionDetailsActionTypeOns))
	}
	if strings.TrimSpace(action.StreamId) != "" {
		inferred = append(inferred, string(eventssdk.ActionDetailsActionTypeOss))
	}
	if strings.TrimSpace(action.FunctionId) != "" {
		inferred = append(inferred, string(eventssdk.ActionDetailsActionTypeFaas))
	}
	switch len(inferred) {
	case 1:
		return inferred[0], nil
	case 0:
		return "", fmt.Errorf("actionType is required when no target OCID identifies the action type")
	default:
		return "", fmt.Errorf("actionType is required when multiple action target OCIDs are set")
	}
}

func ruleSDKActionDetailsListFromCurrent(actions *eventssdk.ActionList) (*eventssdk.ActionDetailsList, error) {
	if actions == nil {
		return nil, nil
	}
	converted := make([]eventssdk.ActionDetails, 0, len(actions.Actions))
	for index, action := range actions.Actions {
		details, err := ruleSDKActionDetailsFromCurrent(action)
		if err != nil {
			return nil, fmt.Errorf("convert current rule action %d: %w", index, err)
		}
		converted = append(converted, details)
	}
	return &eventssdk.ActionDetailsList{Actions: converted}, nil
}

func ruleSDKActionDetailsFromCurrent(action eventssdk.Action) (eventssdk.ActionDetails, error) {
	switch typed := action.(type) {
	case eventssdk.NotificationServiceAction:
		return eventssdk.CreateNotificationServiceActionDetails{
			IsEnabled:   typed.IsEnabled,
			Description: typed.Description,
			TopicId:     typed.TopicId,
		}, nil
	case eventssdk.StreamingServiceAction:
		return eventssdk.CreateStreamingServiceActionDetails{
			IsEnabled:   typed.IsEnabled,
			Description: typed.Description,
			StreamId:    typed.StreamId,
		}, nil
	case eventssdk.FaaSAction:
		return eventssdk.CreateFaaSActionDetails{
			IsEnabled:   typed.IsEnabled,
			Description: typed.Description,
			FunctionId:  typed.FunctionId,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported current action type %T", action)
	}
}

func handleRuleDeleteError(resource *eventsv1beta1.Rule, err error) error {
	if err == nil {
		return nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		}
		return fmt.Errorf("rule delete returned authorization-shaped not found; refusing to confirm deletion: %s", err)
	}
	return err
}

type ruleDeleteAuthShapedConfirmRead struct {
	err error
}

func (e ruleDeleteAuthShapedConfirmRead) Error() string {
	return fmt.Sprintf("rule delete confirmation returned authorization-shaped not found; refusing to confirm deletion: %s", e.err)
}

func handleRuleDeleteConfirmReadOutcome(
	_ *eventsv1beta1.Rule,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	switch typed := response.(type) {
	case ruleDeleteAuthShapedConfirmRead:
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, typed
	case *ruleDeleteAuthShapedConfirmRead:
		if typed != nil {
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, *typed
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func desiredStringUpdate(spec string, current *string) (*string, bool) {
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
	return rawStringPointer(spec), true
}

func desiredBoolUpdate(spec bool, current *bool) (*bool, bool) {
	if current != nil && spec == *current {
		return nil, false
	}
	return boolPointer(spec), true
}

func ruleDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		copied := make(map[string]interface{}, len(values))
		for key, value := range values {
			copied[key] = value
		}
		converted[namespace] = copied
	}
	return converted
}

func cloneDefinedTags(tags map[string]map[string]interface{}) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		copied := make(map[string]interface{}, len(values))
		for key, value := range values {
			copied[key] = value
		}
		cloned[namespace] = copied
	}
	return cloned
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func ruleJSONEqual(left any, right any) (bool, error) {
	leftPayload, err := json.Marshal(left)
	if err != nil {
		return false, fmt.Errorf("marshal desired Rule value: %w", err)
	}
	rightPayload, err := json.Marshal(right)
	if err != nil {
		return false, fmt.Errorf("marshal current Rule value: %w", err)
	}
	return string(leftPayload) == string(rightPayload), nil
}

func stringPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return rawStringPointer(value)
}

func rawStringPointer(value string) *string {
	return &value
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func boolPointer(value bool) *bool {
	return &value
}
