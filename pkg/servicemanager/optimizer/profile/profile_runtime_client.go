/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package profile

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	optimizersdk "github.com/oracle/oci-go-sdk/v65/optimizer"
	optimizerv1beta1 "github.com/oracle/oci-service-operator/api/optimizer/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	registerProfileRuntimeHooksMutator(func(_ *ProfileServiceManager, hooks *ProfileRuntimeHooks) {
		applyProfileRuntimeHooks(hooks)
	})
}

func applyProfileRuntimeHooks(hooks *ProfileRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newProfileRuntimeSemantics()
	hooks.BuildCreateBody = buildProfileCreateBody
	hooks.BuildUpdateBody = buildProfileUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardProfileExistingBeforeCreate
	hooks.List.Fields = profileListFields()
	hooks.Read.List = profilePaginatedListReadOperation(hooks)
	hooks.DeleteHooks.HandleError = handleProfileDeleteError
	wrapProfileDeleteConfirmation(hooks)
}

func newProfileRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "optimizer",
		FormalSlug:    "profile",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(optimizersdk.LifecycleStateCreating)},
			UpdatingStates: []string{
				string(optimizersdk.LifecycleStateUpdating),
				string(optimizersdk.LifecycleStateAttaching),
				string(optimizersdk.LifecycleStateDetaching),
			},
			ActiveStates: []string{
				string(optimizersdk.LifecycleStateActive),
				string(optimizersdk.LifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(optimizersdk.LifecycleStateCreating),
				string(optimizersdk.LifecycleStateUpdating),
				string(optimizersdk.LifecycleStateAttaching),
				string(optimizersdk.LifecycleStateDetaching),
				string(optimizersdk.LifecycleStateDeleting),
			},
			TerminalStates: []string{string(optimizersdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "name", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{
				"aggregationIntervalInDays",
				"definedTags",
				"description",
				"freeformTags",
				"levelsConfiguration",
				"name",
				"systemTags",
				"targetCompartments",
				"targetTags",
			},
			Mutable: []string{
				"aggregationIntervalInDays",
				"definedTags",
				"description",
				"freeformTags",
				"levelsConfiguration",
				"name",
				"systemTags",
				"targetCompartments",
				"targetTags",
			},
			ForceNew:      []string{"compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "Profile", Action: "CreateProfile"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "Profile", Action: "UpdateProfile"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "Profile", Action: "DeleteProfile"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "Profile", Action: "GetProfile"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "Profile", Action: "GetProfile"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "Profile", Action: "GetProfile"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func profileListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "Name", RequestName: "name", Contribution: "query", LookupPaths: []string{"status.name", "spec.name", "name"}},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func guardProfileExistingBeforeCreate(
	_ context.Context,
	resource *optimizerv1beta1.Profile,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("profile resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.Name) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildProfileCreateBody(
	_ context.Context,
	resource *optimizerv1beta1.Profile,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("profile resource is nil")
	}
	if err := validateProfileSpec(resource.Spec); err != nil {
		return nil, err
	}

	spec := resource.Spec
	details := optimizersdk.CreateProfileDetails{
		CompartmentId:       common.String(spec.CompartmentId),
		Name:                common.String(spec.Name),
		Description:         common.String(spec.Description),
		LevelsConfiguration: profileLevelsConfigurationFromSpec(spec.LevelsConfiguration),
	}
	if spec.AggregationIntervalInDays != 0 {
		details.AggregationIntervalInDays = common.Int(spec.AggregationIntervalInDays)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = profileDefinedTagsFromSpec(spec.DefinedTags)
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.SystemTags != nil {
		details.SystemTags = profileDefinedTagsFromSpec(spec.SystemTags)
	}
	if targetCompartments, ok := profileTargetCompartmentsFromSpec(spec.TargetCompartments); ok {
		details.TargetCompartments = targetCompartments
	}
	if targetTags, ok := profileTargetTagsFromSpec(spec.TargetTags); ok {
		details.TargetTags = targetTags
	}
	return details, nil
}

func buildProfileUpdateBody(
	_ context.Context,
	resource *optimizerv1beta1.Profile,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return optimizersdk.UpdateProfileDetails{}, false, fmt.Errorf("profile resource is nil")
	}
	if err := validateProfileSpec(resource.Spec); err != nil {
		return optimizersdk.UpdateProfileDetails{}, false, err
	}

	current, err := profileFromResponse(currentResponse)
	if err != nil {
		return optimizersdk.UpdateProfileDetails{}, false, err
	}
	if currentCompartment := profileStringValue(current.CompartmentId); currentCompartment != "" &&
		resource.Spec.CompartmentId != currentCompartment {
		return optimizersdk.UpdateProfileDetails{}, false,
			fmt.Errorf("profile formal semantics require replacement when compartmentId changes")
	}

	details, updateNeeded := profileUpdateDetailsFromSpec(resource.Spec, current)
	return details, updateNeeded, nil
}

func profileUpdateDetailsFromSpec(
	spec optimizerv1beta1.ProfileSpec,
	current optimizersdk.Profile,
) (optimizersdk.UpdateProfileDetails, bool) {
	details := optimizersdk.UpdateProfileDetails{}
	updateNeeded := profileApplyStringUpdates(&details, spec, current)
	if profileApplyTagUpdates(&details, spec, current) {
		updateNeeded = true
	}
	if profileApplyNestedUpdates(&details, spec, current) {
		updateNeeded = true
	}
	return details, updateNeeded
}

func profileApplyStringUpdates(
	details *optimizersdk.UpdateProfileDetails,
	spec optimizerv1beta1.ProfileSpec,
	current optimizersdk.Profile,
) bool {
	updateNeeded := false
	if desired, ok := profileStringUpdate(spec.Name, current.Name); ok {
		details.Name = desired
		updateNeeded = true
	}
	if desired, ok := profileStringUpdate(spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := profileAggregationIntervalUpdate(spec.AggregationIntervalInDays, current.AggregationIntervalInDays); ok {
		details.AggregationIntervalInDays = desired
		updateNeeded = true
	}
	return updateNeeded
}

func profileApplyTagUpdates(
	details *optimizersdk.UpdateProfileDetails,
	spec optimizerv1beta1.ProfileSpec,
	current optimizersdk.Profile,
) bool {
	updateNeeded := false
	if desired, ok := profileDefinedTagsUpdate(spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}
	if desired, ok := profileFreeformTagsUpdate(spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := profileDefinedTagsUpdate(spec.SystemTags, current.SystemTags); ok {
		details.SystemTags = desired
		updateNeeded = true
	}
	return updateNeeded
}

func profileApplyNestedUpdates(
	details *optimizersdk.UpdateProfileDetails,
	spec optimizerv1beta1.ProfileSpec,
	current optimizersdk.Profile,
) bool {
	updateNeeded := false
	if !profileLevelsConfigurationEqual(spec.LevelsConfiguration, current.LevelsConfiguration) {
		details.LevelsConfiguration = profileLevelsConfigurationFromSpec(spec.LevelsConfiguration)
		updateNeeded = true
	}
	if desired, ok := profileTargetCompartmentsUpdate(spec.TargetCompartments, current.TargetCompartments); ok {
		details.TargetCompartments = desired
		updateNeeded = true
	}
	if desired, ok := profileTargetTagsUpdate(spec.TargetTags, current.TargetTags); ok {
		details.TargetTags = desired
		updateNeeded = true
	}
	return updateNeeded
}

func validateProfileSpec(spec optimizerv1beta1.ProfileSpec) error {
	var problems []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		problems = append(problems, "compartmentId is required")
	}
	if strings.TrimSpace(spec.Name) == "" {
		problems = append(problems, "name is required")
	}
	if strings.TrimSpace(spec.Description) == "" {
		problems = append(problems, "description is required")
	}
	problems = append(problems, validateProfileTargetTags(spec.TargetTags)...)
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("invalid Profile spec: %s", strings.Join(problems, "; "))
}

func validateProfileTargetTags(targetTags optimizerv1beta1.ProfileTargetTags) []string {
	if targetTags.Items == nil {
		return nil
	}
	var problems []string
	for index, item := range targetTags.Items {
		prefix := fmt.Sprintf("targetTags.items[%d]", index)
		if strings.TrimSpace(item.TagNamespaceName) == "" {
			problems = append(problems, prefix+".tagNamespaceName is required")
		}
		if strings.TrimSpace(item.TagDefinitionName) == "" {
			problems = append(problems, prefix+".tagDefinitionName is required")
		}
		switch profileNormalizeTagValueType(item.TagValueType) {
		case string(optimizersdk.TagValueTypeAny):
			if len(item.TagValues) != 0 {
				problems = append(problems, prefix+".tagValues must be empty when tagValueType is ANY")
			}
		case string(optimizersdk.TagValueTypeValue):
			if len(item.TagValues) == 0 {
				problems = append(problems, prefix+".tagValues is required when tagValueType is VALUE")
			}
		default:
			problems = append(problems, prefix+".tagValueType must be ANY or VALUE")
		}
	}
	return problems
}

func profilePaginatedListReadOperation(hooks *ProfileRuntimeHooks) *generatedruntime.Operation {
	if hooks == nil || hooks.List.Call == nil {
		return nil
	}

	listCall := hooks.List.Call
	fields := append([]generatedruntime.RequestField(nil), hooks.List.Fields...)
	return &generatedruntime.Operation{
		NewRequest: func() any { return &optimizersdk.ListProfilesRequest{} },
		Fields:     fields,
		Call: func(ctx context.Context, request any) (any, error) {
			typed, ok := request.(*optimizersdk.ListProfilesRequest)
			if !ok {
				return nil, fmt.Errorf("expected *optimizer.ListProfilesRequest, got %T", request)
			}
			return listProfilePages(ctx, listCall, *typed)
		},
	}
}

func listProfilePages(
	ctx context.Context,
	call func(context.Context, optimizersdk.ListProfilesRequest) (optimizersdk.ListProfilesResponse, error),
	request optimizersdk.ListProfilesRequest,
) (optimizersdk.ListProfilesResponse, error) {
	if call == nil {
		return optimizersdk.ListProfilesResponse{}, fmt.Errorf("profile list operation is not configured")
	}

	seenPages := map[string]struct{}{}
	var combined optimizersdk.ListProfilesResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return optimizersdk.ListProfilesResponse{}, err
		}
		if combined.RawResponse == nil {
			combined.RawResponse = response.RawResponse
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		nextPage := profileStringValue(response.OpcNextPage)
		if nextPage == "" {
			return combined, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return optimizersdk.ListProfilesResponse{}, fmt.Errorf("profile list pagination repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = common.String(nextPage)
	}
}

func handleProfileDeleteError(resource *optimizerv1beta1.Profile, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return fmt.Errorf("profile delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
	}
	return err
}

func wrapProfileDeleteConfirmation(hooks *ProfileRuntimeHooks) {
	if hooks.Get.Call == nil {
		return
	}
	getProfile := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ProfileServiceClient) ProfileServiceClient {
		return profileDeleteConfirmationClient{
			delegate:   delegate,
			getProfile: getProfile,
		}
	})
}

type profileDeleteConfirmationClient struct {
	delegate   ProfileServiceClient
	getProfile func(context.Context, optimizersdk.GetProfileRequest) (optimizersdk.GetProfileResponse, error)
}

func (c profileDeleteConfirmationClient) CreateOrUpdate(
	ctx context.Context,
	resource *optimizerv1beta1.Profile,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c profileDeleteConfirmationClient) Delete(ctx context.Context, resource *optimizerv1beta1.Profile) (bool, error) {
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c profileDeleteConfirmationClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *optimizerv1beta1.Profile,
) error {
	if c.getProfile == nil || resource == nil {
		return nil
	}
	profileID := trackedProfileID(resource)
	if profileID == "" {
		return nil
	}
	_, err := c.getProfile(ctx, optimizersdk.GetProfileRequest{ProfileId: common.String(profileID)})
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("profile delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err)
}

func trackedProfileID(resource *optimizerv1beta1.Profile) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func profileFromResponse(currentResponse any) (optimizersdk.Profile, error) {
	switch current := currentResponse.(type) {
	case optimizersdk.Profile:
		return current, nil
	case *optimizersdk.Profile:
		return profileFromPointer(current)
	case optimizersdk.ProfileSummary:
		return profileFromSummary(current), nil
	case *optimizersdk.ProfileSummary:
		return profileFromSummaryPointer(current)
	default:
		return profileFromOperationResponse(currentResponse)
	}
}

func profileFromOperationResponse(currentResponse any) (optimizersdk.Profile, error) {
	switch current := currentResponse.(type) {
	case optimizersdk.CreateProfileResponse:
		return current.Profile, nil
	case *optimizersdk.CreateProfileResponse:
		return profileFromCreateResponsePointer(current)
	case optimizersdk.GetProfileResponse:
		return current.Profile, nil
	case *optimizersdk.GetProfileResponse:
		return profileFromGetResponsePointer(current)
	case optimizersdk.UpdateProfileResponse:
		return current.Profile, nil
	case *optimizersdk.UpdateProfileResponse:
		return profileFromUpdateResponsePointer(current)
	default:
		return optimizersdk.Profile{}, fmt.Errorf("unexpected current Profile response type %T", currentResponse)
	}
}

func profileFromPointer(current *optimizersdk.Profile) (optimizersdk.Profile, error) {
	if current == nil {
		return optimizersdk.Profile{}, fmt.Errorf("current Profile response is nil")
	}
	return *current, nil
}

func profileFromSummaryPointer(current *optimizersdk.ProfileSummary) (optimizersdk.Profile, error) {
	if current == nil {
		return optimizersdk.Profile{}, fmt.Errorf("current Profile response is nil")
	}
	return profileFromSummary(*current), nil
}

func profileFromCreateResponsePointer(current *optimizersdk.CreateProfileResponse) (optimizersdk.Profile, error) {
	if current == nil {
		return optimizersdk.Profile{}, fmt.Errorf("current Profile response is nil")
	}
	return current.Profile, nil
}

func profileFromGetResponsePointer(current *optimizersdk.GetProfileResponse) (optimizersdk.Profile, error) {
	if current == nil {
		return optimizersdk.Profile{}, fmt.Errorf("current Profile response is nil")
	}
	return current.Profile, nil
}

func profileFromUpdateResponsePointer(current *optimizersdk.UpdateProfileResponse) (optimizersdk.Profile, error) {
	if current == nil {
		return optimizersdk.Profile{}, fmt.Errorf("current Profile response is nil")
	}
	return current.Profile, nil
}

func profileFromSummary(summary optimizersdk.ProfileSummary) optimizersdk.Profile {
	return optimizersdk.Profile(summary)
}

func profileStringUpdate(spec string, current *string) (*string, bool) {
	if spec == profileStringValue(current) {
		return nil, false
	}
	return common.String(spec), true
}

func profileAggregationIntervalUpdate(spec int, current *int) (*int, bool) {
	if spec == 0 {
		return nil, false
	}
	if current != nil && *current == spec {
		return nil, false
	}
	return common.Int(spec), true
}

func profileFreeformTagsUpdate(spec map[string]string, current map[string]string) (map[string]string, bool) {
	if spec == nil || maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func profileDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := profileDefinedTagsFromSpec(spec)
	if reflect.DeepEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func profileTargetCompartmentsUpdate(
	spec optimizerv1beta1.ProfileTargetCompartments,
	current *optimizersdk.TargetCompartments,
) (*optimizersdk.TargetCompartments, bool) {
	if spec.Items == nil {
		return nil, false
	}
	if slices.Equal(spec.Items, profileTargetCompartmentsItemsFromSDK(current)) {
		return nil, false
	}
	desired, _ := profileTargetCompartmentsFromSpec(spec)
	return desired, true
}

func profileTargetTagsUpdate(
	spec optimizerv1beta1.ProfileTargetTags,
	current *optimizersdk.TargetTags,
) (*optimizersdk.TargetTags, bool) {
	if spec.Items == nil {
		return nil, false
	}
	if profileTargetTagsEqual(spec.Items, profileTargetTagsItemsFromSDK(current)) {
		return nil, false
	}
	desired, _ := profileTargetTagsFromSpec(spec)
	return desired, true
}

func profileTargetTagsEqual(
	desired []optimizerv1beta1.ProfileTargetTagsItem,
	current []optimizerv1beta1.ProfileTargetTagsItem,
) bool {
	if len(desired) != len(current) {
		return false
	}
	for index := range desired {
		if !profileTargetTagEqual(desired[index], current[index]) {
			return false
		}
	}
	return true
}

func profileTargetTagEqual(
	desired optimizerv1beta1.ProfileTargetTagsItem,
	current optimizerv1beta1.ProfileTargetTagsItem,
) bool {
	return desired.TagNamespaceName == current.TagNamespaceName &&
		desired.TagDefinitionName == current.TagDefinitionName &&
		profileNormalizeTagValueType(desired.TagValueType) == profileNormalizeTagValueType(current.TagValueType) &&
		slices.Equal(desired.TagValues, current.TagValues)
}

func profileLevelsConfigurationEqual(
	spec optimizerv1beta1.ProfileLevelsConfiguration,
	current *optimizersdk.LevelsConfiguration,
) bool {
	return reflect.DeepEqual(spec.Items, profileLevelConfigurationItemsFromSDK(current))
}

func profileLevelsConfigurationFromSpec(
	spec optimizerv1beta1.ProfileLevelsConfiguration,
) *optimizersdk.LevelsConfiguration {
	items := make([]optimizersdk.LevelConfiguration, 0, len(spec.Items))
	for _, item := range spec.Items {
		items = append(items, optimizersdk.LevelConfiguration{
			RecommendationId: profileOptionalString(item.RecommendationId),
			Level:            profileOptionalString(item.Level),
		})
	}
	return &optimizersdk.LevelsConfiguration{Items: items}
}

func profileTargetCompartmentsFromSpec(
	spec optimizerv1beta1.ProfileTargetCompartments,
) (*optimizersdk.TargetCompartments, bool) {
	if spec.Items == nil {
		return nil, false
	}
	return &optimizersdk.TargetCompartments{Items: append([]string(nil), spec.Items...)}, true
}

func profileTargetTagsFromSpec(spec optimizerv1beta1.ProfileTargetTags) (*optimizersdk.TargetTags, bool) {
	if spec.Items == nil {
		return nil, false
	}
	items := make([]optimizersdk.TargetTag, 0, len(spec.Items))
	for _, item := range spec.Items {
		items = append(items, optimizersdk.TargetTag{
			TagNamespaceName:  common.String(item.TagNamespaceName),
			TagDefinitionName: common.String(item.TagDefinitionName),
			TagValueType:      optimizersdk.TagValueTypeEnum(profileNormalizeTagValueType(item.TagValueType)),
			TagValues:         append([]string(nil), item.TagValues...),
		})
	}
	return &optimizersdk.TargetTags{Items: items}, true
}

func profileDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	converted := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		inner := make(map[string]interface{}, len(values))
		for key, value := range values {
			inner[key] = value
		}
		converted[namespace] = inner
	}
	return converted
}

func profileLevelConfigurationItemsFromSDK(
	current *optimizersdk.LevelsConfiguration,
) []optimizerv1beta1.ProfileLevelsConfigurationItem {
	if current == nil || len(current.Items) == 0 {
		return nil
	}
	items := make([]optimizerv1beta1.ProfileLevelsConfigurationItem, 0, len(current.Items))
	for _, item := range current.Items {
		items = append(items, optimizerv1beta1.ProfileLevelsConfigurationItem{
			RecommendationId: profileStringValue(item.RecommendationId),
			Level:            profileStringValue(item.Level),
		})
	}
	return items
}

func profileTargetCompartmentsItemsFromSDK(current *optimizersdk.TargetCompartments) []string {
	if current == nil || len(current.Items) == 0 {
		return nil
	}
	return append([]string(nil), current.Items...)
}

func profileTargetTagsItemsFromSDK(current *optimizersdk.TargetTags) []optimizerv1beta1.ProfileTargetTagsItem {
	if current == nil || len(current.Items) == 0 {
		return nil
	}
	items := make([]optimizerv1beta1.ProfileTargetTagsItem, 0, len(current.Items))
	for _, item := range current.Items {
		items = append(items, optimizerv1beta1.ProfileTargetTagsItem{
			TagNamespaceName:  profileStringValue(item.TagNamespaceName),
			TagDefinitionName: profileStringValue(item.TagDefinitionName),
			TagValueType:      profileNormalizeTagValueType(string(item.TagValueType)),
			TagValues:         append([]string(nil), item.TagValues...),
		})
	}
	return items
}

func profileOptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return common.String(value)
}

func profileStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func profileNormalizeTagValueType(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
