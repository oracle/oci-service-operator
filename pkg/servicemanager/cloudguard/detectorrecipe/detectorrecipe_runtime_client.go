/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package detectorrecipe

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	registerDetectorRecipeRuntimeHooksMutator(func(_ *DetectorRecipeServiceManager, hooks *DetectorRecipeRuntimeHooks) {
		applyDetectorRecipeRuntimeHooks(hooks)
	})
}

func applyDetectorRecipeRuntimeHooks(hooks *DetectorRecipeRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newDetectorRecipeRuntimeSemantics()
	hooks.BuildCreateBody = buildDetectorRecipeCreateBody
	hooks.BuildUpdateBody = buildDetectorRecipeUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardDetectorRecipeExistingBeforeCreate
	hooks.List.Fields = detectorRecipeListFields()
	wrapDetectorRecipeResponseTagNormalization(hooks)
	hooks.Read.List = detectorRecipePaginatedListReadOperation(hooks)
	hooks.StatusHooks.ClearProjectedStatus = clearDetectorRecipeProjectedStatus
	hooks.StatusHooks.RestoreStatus = restoreDetectorRecipeProjectedStatus
	hooks.StatusHooks.ProjectStatus = projectDetectorRecipeStatus
	hooks.DeleteHooks.HandleError = handleDetectorRecipeDeleteError
	wrapDetectorRecipeDeleteConfirmation(hooks)
}

func newDetectorRecipeRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "cloudguard",
		FormalSlug:    "detectorrecipe",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(cloudguardsdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(cloudguardsdk.LifecycleStateUpdating)},
			ActiveStates: []string{
				string(cloudguardsdk.LifecycleStateActive),
				string(cloudguardsdk.LifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(cloudguardsdk.LifecycleStateCreating),
				string(cloudguardsdk.LifecycleStateUpdating),
				string(cloudguardsdk.LifecycleStateDeleting),
			},
			TerminalStates: []string{string(cloudguardsdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"displayName",
				"detector",
				"sourceDetectorRecipeId",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{
				"definedTags",
				"description",
				"detectorRules",
				"displayName",
				"freeformTags",
			},
			Mutable: []string{
				"definedTags",
				"description",
				"detectorRules",
				"displayName",
				"freeformTags",
			},
			ForceNew: []string{
				"compartmentId",
				"detector",
				"sourceDetectorRecipeId",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "DetectorRecipe", Action: "CreateDetectorRecipe"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "DetectorRecipe", Action: "UpdateDetectorRecipe"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "DetectorRecipe", Action: "DeleteDetectorRecipe"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "DetectorRecipe", Action: "GetDetectorRecipe"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "DetectorRecipe", Action: "GetDetectorRecipe"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "DetectorRecipe", Action: "GetDetectorRecipe"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func detectorRecipeListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"status.displayName", "spec.displayName", "displayName"},
		},
		{
			FieldName:    "LifecycleState",
			RequestName:  "lifecycleState",
			Contribution: "query",
			LookupPaths:  []string{"status.lifecycleState", "lifecycleState"},
		},
		{FieldName: "ResourceMetadataOnly", RequestName: "resourceMetadataOnly", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "CompartmentIdInSubtree", RequestName: "compartmentIdInSubtree", Contribution: "query"},
		{FieldName: "AccessLevel", RequestName: "accessLevel", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func guardDetectorRecipeExistingBeforeCreate(
	_ context.Context,
	resource *cloudguardv1beta1.DetectorRecipe,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("detectorRecipe resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildDetectorRecipeCreateBody(
	_ context.Context,
	resource *cloudguardv1beta1.DetectorRecipe,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("detectorRecipe resource is nil")
	}
	if err := validateDetectorRecipeSpec(resource.Spec); err != nil {
		return nil, err
	}

	rules, err := detectorRecipeRulesFromSpec(resource.Spec.DetectorRules)
	if err != nil {
		return nil, err
	}

	spec := resource.Spec
	details := cloudguardsdk.CreateDetectorRecipeDetails{
		DisplayName:   common.String(spec.DisplayName),
		CompartmentId: common.String(spec.CompartmentId),
	}
	if strings.TrimSpace(spec.Description) != "" {
		details.Description = common.String(spec.Description)
	}
	if strings.TrimSpace(spec.Detector) != "" {
		details.Detector = cloudguardsdk.DetectorEnumEnum(spec.Detector)
	}
	if strings.TrimSpace(spec.SourceDetectorRecipeId) != "" {
		details.SourceDetectorRecipeId = common.String(spec.SourceDetectorRecipeId)
	}
	if len(rules) > 0 {
		details.DetectorRules = rules
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = detectorRecipeDefinedTagsFromSpec(spec.DefinedTags)
	}
	return details, nil
}

func buildDetectorRecipeUpdateBody(
	_ context.Context,
	resource *cloudguardv1beta1.DetectorRecipe,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return cloudguardsdk.UpdateDetectorRecipeDetails{}, false, fmt.Errorf("detectorRecipe resource is nil")
	}
	if err := validateDetectorRecipeSpec(resource.Spec); err != nil {
		return cloudguardsdk.UpdateDetectorRecipeDetails{}, false, err
	}

	current, err := detectorRecipeResponseMap(currentResponse)
	if err != nil {
		return cloudguardsdk.UpdateDetectorRecipeDetails{}, false, err
	}

	details := cloudguardsdk.UpdateDetectorRecipeDetails{}
	updateNeeded := detectorRecipeApplyStringUpdates(&details, resource.Spec, current)
	if changed, err := detectorRecipeApplyDetectorRulesUpdate(&details, resource.Spec, current); err != nil {
		return cloudguardsdk.UpdateDetectorRecipeDetails{}, false, err
	} else if changed {
		updateNeeded = true
	}
	if detectorRecipeApplyTagUpdates(&details, resource.Spec, current) {
		updateNeeded = true
	}
	return details, updateNeeded, nil
}

func detectorRecipeApplyStringUpdates(
	details *cloudguardsdk.UpdateDetectorRecipeDetails,
	spec cloudguardv1beta1.DetectorRecipeSpec,
	current map[string]any,
) bool {
	updateNeeded := detectorRecipeApplyDisplayNameUpdate(details, spec, current)
	if detectorRecipeApplyDescriptionUpdate(details, spec, current) {
		updateNeeded = true
	}
	return updateNeeded
}

func detectorRecipeApplyDisplayNameUpdate(
	details *cloudguardsdk.UpdateDetectorRecipeDetails,
	spec cloudguardv1beta1.DetectorRecipeSpec,
	current map[string]any,
) bool {
	desired, ok := detectorRecipeStringUpdate(spec.DisplayName, detectorRecipeStringField(current, "displayName"))
	if !ok {
		return false
	}
	details.DisplayName = desired
	return true
}

func detectorRecipeApplyDescriptionUpdate(
	details *cloudguardsdk.UpdateDetectorRecipeDetails,
	spec cloudguardv1beta1.DetectorRecipeSpec,
	current map[string]any,
) bool {
	desired := strings.TrimSpace(spec.Description)
	if desired == detectorRecipeStringField(current, "description") {
		return false
	}
	details.Description = common.String(desired)
	return true
}

func detectorRecipeApplyDetectorRulesUpdate(
	details *cloudguardsdk.UpdateDetectorRecipeDetails,
	spec cloudguardv1beta1.DetectorRecipeSpec,
	current map[string]any,
) (bool, error) {
	if spec.DetectorRules == nil {
		return false, nil
	}

	desired, err := detectorRecipeRulesFromSpec(spec.DetectorRules)
	if err != nil {
		return false, err
	}
	currentRules, err := detectorRecipeRulesFromValue(current["detectorRules"])
	if err != nil {
		return false, err
	}
	if detectorRecipeJSONEqual(desired, currentRules) {
		return false, nil
	}
	details.DetectorRules = desired
	return true, nil
}

func detectorRecipeApplyTagUpdates(
	details *cloudguardsdk.UpdateDetectorRecipeDetails,
	spec cloudguardv1beta1.DetectorRecipeSpec,
	current map[string]any,
) bool {
	updateNeeded := false
	if spec.FreeformTags != nil {
		currentFreeformTags := detectorRecipeStringMap(current["freeformTags"])
		if !maps.Equal(spec.FreeformTags, currentFreeformTags) {
			details.FreeformTags = maps.Clone(spec.FreeformTags)
			updateNeeded = true
		}
	}
	if spec.DefinedTags != nil {
		desiredDefinedTags := detectorRecipeDefinedTagsFromSpec(spec.DefinedTags)
		currentDefinedTags := detectorRecipeDefinedTagsFromValue(current["definedTags"])
		if !detectorRecipeDefinedTagsEqual(desiredDefinedTags, currentDefinedTags) {
			details.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}
	return updateNeeded
}

func validateDetectorRecipeSpec(spec cloudguardv1beta1.DetectorRecipeSpec) error {
	var problems []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		problems = append(problems, "compartmentId is required")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		problems = append(problems, "displayName is required")
	}
	if strings.TrimSpace(spec.Detector) != "" {
		if _, ok := cloudguardsdk.GetMappingDetectorEnumEnum(spec.Detector); !ok {
			problems = append(problems, "detector must be a supported Cloud Guard detector enum")
		}
	}
	for index, rule := range spec.DetectorRules {
		if strings.TrimSpace(rule.DetectorRuleId) == "" {
			problems = append(problems, fmt.Sprintf("detectorRules[%d].detectorRuleId is required", index))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("invalid DetectorRecipe spec: %s", strings.Join(problems, "; "))
}

func detectorRecipeRulesFromSpec(
	rules []cloudguardv1beta1.DetectorRecipeDetectorRuleFields,
) ([]cloudguardsdk.UpdateDetectorRecipeDetectorRule, error) {
	if rules == nil {
		return nil, nil
	}
	payload, err := json.Marshal(rules)
	if err != nil {
		return nil, fmt.Errorf("marshal detector rules: %w", err)
	}
	payload, err = detectorRecipePruneEmptyRuleConditions(payload)
	if err != nil {
		return nil, err
	}
	var converted []cloudguardsdk.UpdateDetectorRecipeDetectorRule
	if err := json.Unmarshal(payload, &converted); err != nil {
		return nil, fmt.Errorf("convert detector rules to OCI request body: %w", err)
	}
	for index, rule := range converted {
		if rule.DetectorRuleId == nil || strings.TrimSpace(*rule.DetectorRuleId) == "" {
			return nil, fmt.Errorf("detectorRules[%d].detectorRuleId is required", index)
		}
		if rule.Details == nil {
			return nil, fmt.Errorf("detectorRules[%d].details is required", index)
		}
	}
	return converted, nil
}

func detectorRecipeRulesFromValue(value any) ([]cloudguardsdk.UpdateDetectorRecipeDetectorRule, error) {
	if value == nil {
		return nil, nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal observed detector rules: %w", err)
	}
	payload, err = detectorRecipePruneEmptyRuleConditions(payload)
	if err != nil {
		return nil, err
	}
	var rules []cloudguardsdk.UpdateDetectorRecipeDetectorRule
	if err := json.Unmarshal(payload, &rules); err != nil {
		return nil, fmt.Errorf("convert observed detector rules: %w", err)
	}
	return rules, nil
}

func detectorRecipePruneEmptyRuleConditions(payload []byte) ([]byte, error) {
	var rules []map[string]any
	if err := json.Unmarshal(payload, &rules); err != nil {
		return nil, fmt.Errorf("decode detector rules for normalization: %w", err)
	}
	for _, rule := range rules {
		details, ok := rule["details"].(map[string]any)
		if !ok {
			continue
		}
		condition, ok := details["condition"].(map[string]any)
		if ok && detectorRecipeEmptyCondition(condition) {
			delete(details, "condition")
		}
	}
	normalized, err := json.Marshal(rules)
	if err != nil {
		return nil, fmt.Errorf("marshal normalized detector rules: %w", err)
	}
	return normalized, nil
}

func detectorRecipeEmptyCondition(condition map[string]any) bool {
	if len(condition) == 0 {
		return true
	}
	if kind := detectorRecipeStringValue(condition["kind"]); kind != "" {
		return false
	}
	if kind := detectorRecipeStringValue(condition["Kind"]); kind != "" {
		return false
	}
	for key, value := range condition {
		switch strings.ToLower(key) {
		case "jsondata", "json_data", "kind":
			continue
		default:
			if detectorRecipeMeaningfulValue(value) {
				return false
			}
		}
	}
	return true
}

func detectorRecipeMeaningfulValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		return detectorRecipeSliceHasMeaningfulValue(typed)
	case map[string]any:
		return detectorRecipeMapHasMeaningfulValue(typed)
	case bool:
		return true
	case float64:
		return typed != 0
	default:
		return true
	}
}

func detectorRecipeSliceHasMeaningfulValue(values []any) bool {
	for _, item := range values {
		if detectorRecipeMeaningfulValue(item) {
			return true
		}
	}
	return false
}

func detectorRecipeMapHasMeaningfulValue(values map[string]any) bool {
	for _, item := range values {
		if detectorRecipeMeaningfulValue(item) {
			return true
		}
	}
	return false
}

func wrapDetectorRecipeResponseTagNormalization(hooks *DetectorRecipeRuntimeHooks) {
	if hooks == nil {
		return
	}
	if hooks.Create.Call != nil {
		createCall := hooks.Create.Call
		hooks.Create.Call = func(ctx context.Context, request cloudguardsdk.CreateDetectorRecipeRequest) (cloudguardsdk.CreateDetectorRecipeResponse, error) {
			response, err := createCall(ctx, request)
			normalizeDetectorRecipeSDKTags(&response.DetectorRecipe)
			return response, err
		}
	}
	if hooks.Get.Call != nil {
		getCall := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request cloudguardsdk.GetDetectorRecipeRequest) (cloudguardsdk.GetDetectorRecipeResponse, error) {
			response, err := getCall(ctx, request)
			normalizeDetectorRecipeSDKTags(&response.DetectorRecipe)
			return response, err
		}
	}
	if hooks.Update.Call != nil {
		updateCall := hooks.Update.Call
		hooks.Update.Call = func(ctx context.Context, request cloudguardsdk.UpdateDetectorRecipeRequest) (cloudguardsdk.UpdateDetectorRecipeResponse, error) {
			response, err := updateCall(ctx, request)
			normalizeDetectorRecipeSDKTags(&response.DetectorRecipe)
			return response, err
		}
	}
	if hooks.List.Call != nil {
		listCall := hooks.List.Call
		hooks.List.Call = func(ctx context.Context, request cloudguardsdk.ListDetectorRecipesRequest) (cloudguardsdk.ListDetectorRecipesResponse, error) {
			response, err := listCall(ctx, request)
			for index := range response.Items {
				normalizeDetectorRecipeSDKSummaryTags(&response.Items[index])
			}
			return response, err
		}
	}
}

func normalizeDetectorRecipeSDKTags(recipe *cloudguardsdk.DetectorRecipe) {
	if recipe == nil {
		return
	}
	normalizeDetectorRecipeSDKTagValues(recipe.DefinedTags)
	normalizeDetectorRecipeSDKTagValues(recipe.SystemTags)
}

func normalizeDetectorRecipeSDKSummaryTags(summary *cloudguardsdk.DetectorRecipeSummary) {
	if summary == nil {
		return
	}
	normalizeDetectorRecipeSDKTagValues(summary.DefinedTags)
	normalizeDetectorRecipeSDKTagValues(summary.SystemTags)
}

func normalizeDetectorRecipeSDKTagValues(tags map[string]map[string]interface{}) {
	for namespace, values := range tags {
		if values == nil {
			continue
		}
		normalized := make(map[string]interface{}, len(values))
		for key, value := range values {
			normalized[key] = fmt.Sprint(value)
		}
		tags[namespace] = normalized
	}
}

func detectorRecipePaginatedListReadOperation(hooks *DetectorRecipeRuntimeHooks) *generatedruntime.Operation {
	if hooks == nil || hooks.List.Call == nil {
		return nil
	}

	listCall := hooks.List.Call
	fields := append([]generatedruntime.RequestField(nil), hooks.List.Fields...)
	return &generatedruntime.Operation{
		NewRequest: func() any { return &cloudguardsdk.ListDetectorRecipesRequest{} },
		Fields:     fields,
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*cloudguardsdk.ListDetectorRecipesRequest)
			if !ok {
				return nil, fmt.Errorf("expected *cloudguard.ListDetectorRecipesRequest, got %T", request)
			}
			return listDetectorRecipePages(ctx, listCall, *typed)
		},
	}
}

func listDetectorRecipePages(
	ctx context.Context,
	call func(context.Context, cloudguardsdk.ListDetectorRecipesRequest) (cloudguardsdk.ListDetectorRecipesResponse, error),
	request cloudguardsdk.ListDetectorRecipesRequest,
) (cloudguardsdk.ListDetectorRecipesResponse, error) {
	if call == nil {
		return cloudguardsdk.ListDetectorRecipesResponse{}, fmt.Errorf("detectorRecipe list operation is not configured")
	}
	if request.LifecycleState != "" {
		return listDetectorRecipePagesForState(ctx, call, request)
	}

	var combined cloudguardsdk.ListDetectorRecipesResponse
	for _, state := range detectorRecipeListLifecycleStates() {
		stateRequest := request
		stateRequest.Page = nil
		stateRequest.LifecycleState = state
		response, err := listDetectorRecipePagesForState(ctx, call, stateRequest)
		if err != nil {
			return cloudguardsdk.ListDetectorRecipesResponse{}, err
		}
		mergeDetectorRecipeListResponse(&combined, response)
	}
	return combined, nil
}

func detectorRecipeListLifecycleStates() []cloudguardsdk.ListDetectorRecipesLifecycleStateEnum {
	return []cloudguardsdk.ListDetectorRecipesLifecycleStateEnum{
		cloudguardsdk.ListDetectorRecipesLifecycleStateCreating,
		cloudguardsdk.ListDetectorRecipesLifecycleStateUpdating,
		cloudguardsdk.ListDetectorRecipesLifecycleStateActive,
		cloudguardsdk.ListDetectorRecipesLifecycleStateInactive,
		cloudguardsdk.ListDetectorRecipesLifecycleStateDeleting,
		cloudguardsdk.ListDetectorRecipesLifecycleStateDeleted,
		cloudguardsdk.ListDetectorRecipesLifecycleStateFailed,
	}
}

func listDetectorRecipePagesForState(
	ctx context.Context,
	call func(context.Context, cloudguardsdk.ListDetectorRecipesRequest) (cloudguardsdk.ListDetectorRecipesResponse, error),
	request cloudguardsdk.ListDetectorRecipesRequest,
) (cloudguardsdk.ListDetectorRecipesResponse, error) {
	seenPages := map[string]struct{}{}
	var combined cloudguardsdk.ListDetectorRecipesResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return cloudguardsdk.ListDetectorRecipesResponse{}, err
		}
		mergeDetectorRecipeListResponse(&combined, response)

		nextPage := detectorRecipeStringValue(response.OpcNextPage)
		if nextPage == "" {
			return combined, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return cloudguardsdk.ListDetectorRecipesResponse{}, fmt.Errorf("detectorRecipe list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = common.String(nextPage)
	}
}

func mergeDetectorRecipeListResponse(
	combined *cloudguardsdk.ListDetectorRecipesResponse,
	response cloudguardsdk.ListDetectorRecipesResponse,
) {
	if combined.RawResponse == nil {
		combined.RawResponse = response.RawResponse
	}
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	combined.Items = append(combined.Items, response.Items...)
	combined.Locks = append(combined.Locks, response.Locks...)
	combined.OpcNextPage = response.OpcNextPage
}

func clearDetectorRecipeProjectedStatus(resource *cloudguardv1beta1.DetectorRecipe) any {
	if resource == nil {
		return nil
	}
	baseline := resource.Status
	resource.Status = cloudguardv1beta1.DetectorRecipeStatus{OsokStatus: baseline.OsokStatus}
	return baseline
}

func restoreDetectorRecipeProjectedStatus(resource *cloudguardv1beta1.DetectorRecipe, baseline any) {
	if resource == nil {
		return
	}
	if status, ok := baseline.(cloudguardv1beta1.DetectorRecipeStatus); ok {
		osokStatus := resource.Status.OsokStatus
		resource.Status = status
		resource.Status.OsokStatus = osokStatus
	}
}

func projectDetectorRecipeStatus(resource *cloudguardv1beta1.DetectorRecipe, response any) error {
	if resource == nil {
		return fmt.Errorf("detectorRecipe resource is nil")
	}
	body, err := detectorRecipeResponseBody(response)
	if err != nil {
		return err
	}
	values, err := detectorRecipeBodyMap(body)
	if err != nil {
		return err
	}
	normalizeDetectorRecipeStatusTags(values, "definedTags")
	normalizeDetectorRecipeStatusTags(values, "systemTags")

	status := cloudguardv1beta1.DetectorRecipeStatus{OsokStatus: resource.Status.OsokStatus}
	payload, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshal DetectorRecipe status projection: %w", err)
	}
	if err := json.Unmarshal(payload, &status); err != nil {
		return fmt.Errorf("project DetectorRecipe status: %w", err)
	}
	resource.Status = status
	return nil
}

func detectorRecipeResponseMap(response any) (map[string]any, error) {
	body, err := detectorRecipeResponseBody(response)
	if err != nil {
		return nil, err
	}
	return detectorRecipeBodyMap(body)
}

func detectorRecipeResponseBody(response any) (any, error) {
	switch typed := response.(type) {
	case cloudguardsdk.CreateDetectorRecipeResponse:
		return typed.DetectorRecipe, nil
	case *cloudguardsdk.CreateDetectorRecipeResponse:
		if typed == nil {
			return nil, fmt.Errorf("DetectorRecipe response is nil")
		}
		return typed.DetectorRecipe, nil
	case cloudguardsdk.GetDetectorRecipeResponse:
		return typed.DetectorRecipe, nil
	case *cloudguardsdk.GetDetectorRecipeResponse:
		if typed == nil {
			return nil, fmt.Errorf("DetectorRecipe response is nil")
		}
		return typed.DetectorRecipe, nil
	case cloudguardsdk.UpdateDetectorRecipeResponse:
		return typed.DetectorRecipe, nil
	case *cloudguardsdk.UpdateDetectorRecipeResponse:
		if typed == nil {
			return nil, fmt.Errorf("DetectorRecipe response is nil")
		}
		return typed.DetectorRecipe, nil
	default:
		return detectorRecipeDirectResponseBody(response)
	}
}

func detectorRecipeDirectResponseBody(response any) (any, error) {
	switch typed := response.(type) {
	case cloudguardsdk.DetectorRecipe:
		return typed, nil
	case *cloudguardsdk.DetectorRecipe:
		if typed == nil {
			return nil, fmt.Errorf("DetectorRecipe body is nil")
		}
		return *typed, nil
	case cloudguardsdk.DetectorRecipeSummary:
		return typed, nil
	case *cloudguardsdk.DetectorRecipeSummary:
		if typed == nil {
			return nil, fmt.Errorf("DetectorRecipe summary is nil")
		}
		return *typed, nil
	case map[string]any:
		return typed, nil
	default:
		return nil, fmt.Errorf("DetectorRecipe response body has unsupported type %T", response)
	}
}

func detectorRecipeBodyMap(body any) (map[string]any, error) {
	if body == nil {
		return nil, fmt.Errorf("DetectorRecipe response body is nil")
	}
	if values, ok := body.(map[string]any); ok {
		return maps.Clone(values), nil
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal DetectorRecipe response body: %w", err)
	}
	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, fmt.Errorf("decode DetectorRecipe response body: %w", err)
	}
	return values, nil
}

func normalizeDetectorRecipeStatusTags(values map[string]any, field string) {
	if values == nil {
		return
	}
	raw, ok := values[field]
	if !ok {
		return
	}
	values[field] = detectorRecipeStatusTagsFromValue(raw)
}

func detectorRecipeStatusTagsFromValue(value any) map[string]shared.MapValue {
	namespaces, ok := value.(map[string]any)
	if !ok || len(namespaces) == 0 {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(namespaces))
	for namespace, rawChildren := range namespaces {
		children := detectorRecipeStringMap(rawChildren)
		if children == nil {
			converted[namespace] = nil
			continue
		}
		converted[namespace] = shared.MapValue(children)
	}
	return converted
}

func detectorRecipeDefinedTagsFromSpec(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&source)
}

func detectorRecipeDefinedTagsFromValue(value any) map[string]map[string]interface{} {
	namespaces, ok := value.(map[string]any)
	if !ok || len(namespaces) == 0 {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(namespaces))
	for namespace, rawChildren := range namespaces {
		children := detectorRecipeStringMap(rawChildren)
		if children == nil {
			converted[namespace] = nil
			continue
		}
		convertedChildren := make(map[string]interface{}, len(children))
		for key, child := range children {
			convertedChildren[key] = child
		}
		converted[namespace] = convertedChildren
	}
	return converted
}

func detectorRecipeStringMap(value any) map[string]string {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case map[string]string:
		return maps.Clone(typed)
	case map[string]any:
		if len(typed) == 0 {
			return map[string]string{}
		}
		converted := make(map[string]string, len(typed))
		for key, child := range typed {
			converted[key] = fmt.Sprint(child)
		}
		return converted
	default:
		return nil
	}
}

func detectorRecipeStringField(values map[string]any, field string) string {
	if values == nil {
		return ""
	}
	return detectorRecipeStringValue(values[field])
}

func detectorRecipeStringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case *string:
		if typed == nil {
			return ""
		}
		return strings.TrimSpace(*typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func detectorRecipeStringUpdate(desired string, current string) (*string, bool) {
	desired = strings.TrimSpace(desired)
	if desired == "" || desired == strings.TrimSpace(current) {
		return nil, false
	}
	return common.String(desired), true
}

func detectorRecipeDefinedTagsEqual(
	left map[string]map[string]interface{},
	right map[string]map[string]interface{},
) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}
	return detectorRecipeJSONEqual(left, right)
}

func detectorRecipeJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return fmt.Sprint(left) == fmt.Sprint(right)
	}
	return string(leftPayload) == string(rightPayload)
}

func handleDetectorRecipeDeleteError(resource *cloudguardv1beta1.DetectorRecipe, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return fmt.Errorf("detectorRecipe delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
	}
	return err
}

func wrapDetectorRecipeDeleteConfirmation(hooks *DetectorRecipeRuntimeHooks) {
	if hooks.Get.Call == nil {
		return
	}
	getDetectorRecipe := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate DetectorRecipeServiceClient) DetectorRecipeServiceClient {
		return detectorRecipeDeleteConfirmationClient{
			delegate:          delegate,
			getDetectorRecipe: getDetectorRecipe,
		}
	})
}

type detectorRecipeDeleteConfirmationClient struct {
	delegate          DetectorRecipeServiceClient
	getDetectorRecipe func(context.Context, cloudguardsdk.GetDetectorRecipeRequest) (cloudguardsdk.GetDetectorRecipeResponse, error)
}

func (c detectorRecipeDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *cloudguardv1beta1.DetectorRecipe,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c detectorRecipeDeleteConfirmationClient) Delete(
	ctx context.Context,
	resource *cloudguardv1beta1.DetectorRecipe,
) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c detectorRecipeDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *cloudguardv1beta1.DetectorRecipe,
) error {
	if c.getDetectorRecipe == nil || resource == nil {
		return nil
	}
	detectorRecipeID := trackedDetectorRecipeID(resource)
	if detectorRecipeID == "" {
		return nil
	}
	_, err := c.getDetectorRecipe(ctx, cloudguardsdk.GetDetectorRecipeRequest{DetectorRecipeId: common.String(detectorRecipeID)})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("detectorRecipe delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
}

func trackedDetectorRecipeID(resource *cloudguardv1beta1.DetectorRecipe) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); id != "" {
		return id
	}
	return strings.TrimSpace(resource.Status.Id)
}
