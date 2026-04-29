/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package governancerule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	governancerulescontrolplanesdk "github.com/oracle/oci-go-sdk/v65/governancerulescontrolplane"
	governancerulescontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/governancerulescontrolplane/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type governanceRuleIdentity struct {
	compartmentID string
	displayName   string
	ruleType      string
}

type governanceRuleWorkRequestClient interface {
	GetWorkRequest(context.Context, governancerulescontrolplanesdk.GetWorkRequestRequest) (governancerulescontrolplanesdk.GetWorkRequestResponse, error)
}

type ambiguousGovernanceRuleNotFoundError struct {
	message      string
	opcRequestID string
}

var governanceRuleWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(governancerulescontrolplanesdk.OperationStatusAccepted),
		string(governancerulescontrolplanesdk.OperationStatusInProgress),
		string(governancerulescontrolplanesdk.OperationStatusWaiting),
		string(governancerulescontrolplanesdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(governancerulescontrolplanesdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(governancerulescontrolplanesdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(governancerulescontrolplanesdk.OperationStatusCanceled)},
	CreateActionTokens: []string{
		string(governancerulescontrolplanesdk.OperationTypeCreateGovernanceRule),
		string(governancerulescontrolplanesdk.OperationTypeCreateEnforcedQuotaGovernanceRule),
		string(governancerulescontrolplanesdk.OperationTypeCreateEnforcedTagGovernanceRule),
	},
	UpdateActionTokens: []string{
		string(governancerulescontrolplanesdk.OperationTypeUpdateGovernanceRule),
		string(governancerulescontrolplanesdk.OperationTypeUpdateEnforcedQuotaGovernanceRule),
		string(governancerulescontrolplanesdk.OperationTypeUpdateEnforcedTagGovernanceRule),
	},
	DeleteActionTokens: []string{
		string(governancerulescontrolplanesdk.OperationTypeDeleteGovernanceRule),
		string(governancerulescontrolplanesdk.OperationTypeDeleteEnforcedQuotaGovernanceRule),
		string(governancerulescontrolplanesdk.OperationTypeDeleteEnforcedTagGovernanceRule),
	},
}

func (e ambiguousGovernanceRuleNotFoundError) Error() string {
	return e.message
}

func (e ambiguousGovernanceRuleNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerGovernanceRuleRuntimeHooksMutator(func(manager *GovernanceRuleServiceManager, hooks *GovernanceRuleRuntimeHooks) {
		workRequestClient, workRequestInitErr := newGovernanceRuleWorkRequestClient(manager)
		applyGovernanceRuleRuntimeHooks(hooks, workRequestClient, workRequestInitErr)
	})
}

func newGovernanceRuleWorkRequestClient(manager *GovernanceRuleServiceManager) (governanceRuleWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("GovernanceRule service manager is nil")
	}
	return governancerulescontrolplanesdk.NewWorkRequestClientWithConfigurationProvider(manager.Provider)
}

func applyGovernanceRuleRuntimeHooks(
	hooks *GovernanceRuleRuntimeHooks,
	workRequestClient governanceRuleWorkRequestClient,
	workRequestInitErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = governanceRuleRuntimeSemantics()
	hooks.BuildCreateBody = buildGovernanceRuleCreateBody
	hooks.BuildUpdateBody = buildGovernanceRuleUpdateBody
	hooks.Identity.Resolve = resolveGovernanceRuleIdentity
	hooks.Identity.RecordPath = recordGovernanceRulePathIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardGovernanceRuleExistingBeforeCreate
	hooks.List.Fields = governanceRuleListFields()
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedGovernanceRuleIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateGovernanceRuleCreateOnlyDriftForResponse
	hooks.DeleteHooks.HandleError = handleGovernanceRuleDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyGovernanceRuleDeleteConfirmOutcome
	hooks.Async.Adapter = governanceRuleWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getGovernanceRuleWorkRequest(ctx, workRequestClient, workRequestInitErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveGovernanceRuleWorkRequestAction
	hooks.Async.ResolvePhase = resolveGovernanceRuleWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverGovernanceRuleIDFromWorkRequest
	hooks.Async.Message = governanceRuleWorkRequestMessage
	wrapGovernanceRuleDeleteWorkRequestConfirmation(hooks)
	wrapGovernanceRuleListCalls(hooks)
}

func governanceRuleRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "governancerulescontrolplane",
		FormalSlug:    "governancerule",
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
			ActiveStates: []string{string(governancerulescontrolplanesdk.GovernanceRuleLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(governancerulescontrolplanesdk.GovernanceRuleLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "type", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"template",
				"relatedResourceId",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"compartmentId", "type", "creationOption"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "governanceRule", Action: string(governancerulescontrolplanesdk.OperationTypeCreateGovernanceRule)},
			},
			Update: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "governanceRule", Action: string(governancerulescontrolplanesdk.OperationTypeUpdateGovernanceRule)},
			},
			Delete: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "governanceRule", Action: string(governancerulescontrolplanesdk.OperationTypeDeleteGovernanceRule)},
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetGovernanceRule",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "governanceRule", Action: string(governancerulescontrolplanesdk.OperationTypeCreateGovernanceRule)},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetGovernanceRule",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "governanceRule", Action: string(governancerulescontrolplanesdk.OperationTypeUpdateGovernanceRule)},
			},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "governanceRule", Action: string(governancerulescontrolplanesdk.OperationTypeDeleteGovernanceRule)},
			},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func governanceRuleListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "GovernanceRuleType", RequestName: "governanceRuleType", Contribution: "query", LookupPaths: []string{"status.type", "spec.type", "type"}},
		{FieldName: "GovernanceRuleId", RequestName: "governanceRuleId", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func buildGovernanceRuleCreateBody(
	_ context.Context,
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("governance rule resource is nil")
	}
	return governanceRuleCreateDetailsFromSpec(resource.Spec)
}

func buildGovernanceRuleUpdateBody(
	_ context.Context,
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return governancerulescontrolplanesdk.UpdateGovernanceRuleDetails{}, false, fmt.Errorf("governance rule resource is nil")
	}
	current, ok := governanceRuleFromResponse(currentResponse)
	if !ok {
		return governancerulescontrolplanesdk.UpdateGovernanceRuleDetails{}, false, fmt.Errorf("current governance rule response does not expose a governance rule body")
	}
	if err := validateGovernanceRuleCreateOnlyDrift(resource.Spec, current); err != nil {
		return governancerulescontrolplanesdk.UpdateGovernanceRuleDetails{}, false, err
	}

	desired, err := governanceRuleUpdateDetailsFromSpec(resource.Spec)
	if err != nil {
		return governancerulescontrolplanesdk.UpdateGovernanceRuleDetails{}, false, err
	}

	return governanceRuleUpdateDetailsFromCurrent(desired, current)
}

func governanceRuleUpdateDetailsFromCurrent(
	desired governancerulescontrolplanesdk.UpdateGovernanceRuleDetails,
	current governancerulescontrolplanesdk.GovernanceRule,
) (governancerulescontrolplanesdk.UpdateGovernanceRuleDetails, bool, error) {
	update := governancerulescontrolplanesdk.UpdateGovernanceRuleDetails{}
	updateNeeded := applyGovernanceRuleStringUpdates(&update, desired, current)
	updateNeeded = applyGovernanceRuleStructuredUpdates(&update, desired, current) || updateNeeded
	if !updateNeeded {
		return governancerulescontrolplanesdk.UpdateGovernanceRuleDetails{}, false, nil
	}
	return update, true, nil
}

func applyGovernanceRuleStringUpdates(
	update *governancerulescontrolplanesdk.UpdateGovernanceRuleDetails,
	desired governancerulescontrolplanesdk.UpdateGovernanceRuleDetails,
	current governancerulescontrolplanesdk.GovernanceRule,
) bool {
	updateNeeded := false
	updateNeeded = applyGovernanceRuleStringUpdate(&update.DisplayName, desired.DisplayName, current.DisplayName) || updateNeeded
	updateNeeded = applyGovernanceRuleStringUpdate(&update.Description, desired.Description, current.Description) || updateNeeded
	updateNeeded = applyGovernanceRuleStringUpdate(&update.RelatedResourceId, desired.RelatedResourceId, current.RelatedResourceId) || updateNeeded
	return updateNeeded
}

func applyGovernanceRuleStringUpdate(target **string, desired *string, current *string) bool {
	value, ok := governanceRuleDesiredStringUpdate(desired, current)
	if !ok {
		return false
	}
	*target = value
	return true
}

func applyGovernanceRuleStructuredUpdates(
	update *governancerulescontrolplanesdk.UpdateGovernanceRuleDetails,
	desired governancerulescontrolplanesdk.UpdateGovernanceRuleDetails,
	current governancerulescontrolplanesdk.GovernanceRule,
) bool {
	updateNeeded := false
	if desired.Template != nil && !governanceRuleSemanticEqual(desired.Template, current.Template) {
		update.Template = desired.Template
		updateNeeded = true
	}
	if desired.FreeformTags != nil && !reflect.DeepEqual(current.FreeformTags, desired.FreeformTags) {
		update.FreeformTags = cloneGovernanceRuleStringMap(desired.FreeformTags)
		updateNeeded = true
	}
	if desired.DefinedTags != nil && !reflect.DeepEqual(current.DefinedTags, desired.DefinedTags) {
		update.DefinedTags = cloneGovernanceRuleDefinedTags(desired.DefinedTags)
		updateNeeded = true
	}
	return updateNeeded
}

func governanceRuleDesiredStringUpdate(desired *string, current *string) (*string, bool) {
	if desired == nil || governanceRuleStringPtrEqual(current, *desired) {
		return nil, false
	}
	return common.String(strings.TrimSpace(*desired)), true
}

func governanceRuleCreateDetailsFromSpec(
	spec governancerulescontrolplanev1beta1.GovernanceRuleSpec,
) (governancerulescontrolplanesdk.CreateGovernanceRuleDetails, error) {
	ruleType, err := normalizeGovernanceRuleType(spec.Type)
	if err != nil {
		return governancerulescontrolplanesdk.CreateGovernanceRuleDetails{}, err
	}
	creationOption, err := normalizeGovernanceRuleCreationOption(spec.CreationOption)
	if err != nil {
		return governancerulescontrolplanesdk.CreateGovernanceRuleDetails{}, err
	}
	template, err := governanceRuleTemplateFromSpecForCreation(string(ruleType), creationOption, spec)
	if err != nil {
		return governancerulescontrolplanesdk.CreateGovernanceRuleDetails{}, err
	}
	if err := validateGovernanceRuleRequiredSpec(spec, creationOption, template); err != nil {
		return governancerulescontrolplanesdk.CreateGovernanceRuleDetails{}, err
	}

	body := governancerulescontrolplanesdk.CreateGovernanceRuleDetails{
		CompartmentId:  common.String(strings.TrimSpace(spec.CompartmentId)),
		DisplayName:    common.String(strings.TrimSpace(spec.DisplayName)),
		Type:           ruleType,
		CreationOption: creationOption,
		Template:       template,
	}
	applyGovernanceRuleMutableFields(&body.Description, &body.RelatedResourceId, &body.FreeformTags, &body.DefinedTags, spec)
	return body, nil
}

func governanceRuleUpdateDetailsFromSpec(
	spec governancerulescontrolplanev1beta1.GovernanceRuleSpec,
) (governancerulescontrolplanesdk.UpdateGovernanceRuleDetails, error) {
	ruleType, err := normalizeGovernanceRuleType(spec.Type)
	if err != nil {
		return governancerulescontrolplanesdk.UpdateGovernanceRuleDetails{}, err
	}
	creationOption, err := normalizeGovernanceRuleCreationOption(spec.CreationOption)
	if err != nil {
		return governancerulescontrolplanesdk.UpdateGovernanceRuleDetails{}, err
	}
	template, err := governanceRuleTemplateFromSpecForCreation(string(ruleType), creationOption, spec)
	if err != nil {
		return governancerulescontrolplanesdk.UpdateGovernanceRuleDetails{}, err
	}
	if err := validateGovernanceRuleRequiredSpec(spec, creationOption, template); err != nil {
		return governancerulescontrolplanesdk.UpdateGovernanceRuleDetails{}, err
	}

	body := governancerulescontrolplanesdk.UpdateGovernanceRuleDetails{
		DisplayName: common.String(strings.TrimSpace(spec.DisplayName)),
		Template:    template,
	}
	applyGovernanceRuleMutableFields(&body.Description, &body.RelatedResourceId, &body.FreeformTags, &body.DefinedTags, spec)
	return body, nil
}

func applyGovernanceRuleMutableFields(
	description **string,
	relatedResourceID **string,
	freeformTags *map[string]string,
	definedTags *map[string]map[string]interface{},
	spec governancerulescontrolplanev1beta1.GovernanceRuleSpec,
) {
	if strings.TrimSpace(spec.Description) != "" {
		*description = common.String(strings.TrimSpace(spec.Description))
	}
	if strings.TrimSpace(spec.RelatedResourceId) != "" {
		*relatedResourceID = common.String(strings.TrimSpace(spec.RelatedResourceId))
	}
	if spec.FreeformTags != nil {
		*freeformTags = cloneGovernanceRuleStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		*definedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
}

func validateGovernanceRuleRequiredSpec(
	spec governancerulescontrolplanev1beta1.GovernanceRuleSpec,
	creationOption governancerulescontrolplanesdk.CreationOptionEnum,
	template governancerulescontrolplanesdk.Template,
) error {
	missing := governanceRuleMissingRequiredSpecFields(spec)
	missing = append(missing, governanceRuleMissingTemplateField(spec, creationOption, template)...)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("governance rule spec is missing required field(s): %s", strings.Join(missing, ", "))
}

func governanceRuleMissingRequiredSpecFields(spec governancerulescontrolplanev1beta1.GovernanceRuleSpec) []string {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if strings.TrimSpace(spec.Type) == "" {
		missing = append(missing, "type")
	}
	if strings.TrimSpace(spec.CreationOption) == "" {
		missing = append(missing, "creationOption")
	}
	return missing
}

func governanceRuleMissingTemplateField(
	spec governancerulescontrolplanev1beta1.GovernanceRuleSpec,
	creationOption governancerulescontrolplanesdk.CreationOptionEnum,
	template governancerulescontrolplanesdk.Template,
) []string {
	if template == nil && creationOption == governancerulescontrolplanesdk.CreationOptionClone && strings.TrimSpace(spec.RelatedResourceId) == "" {
		return []string{"template or relatedResourceId"}
	}
	if template == nil && creationOption != governancerulescontrolplanesdk.CreationOptionClone {
		return []string{"template"}
	}
	return nil
}

func normalizeGovernanceRuleType(value string) (governancerulescontrolplanesdk.GovernanceRuleTypeEnum, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", fmt.Errorf("governance rule type is required")
	}
	ruleType, ok := governancerulescontrolplanesdk.GetMappingGovernanceRuleTypeEnum(normalized)
	if !ok {
		return "", fmt.Errorf("unsupported governance rule type %q", value)
	}
	return ruleType, nil
}

func normalizeGovernanceRuleCreationOption(value string) (governancerulescontrolplanesdk.CreationOptionEnum, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", fmt.Errorf("governance rule creationOption is required")
	}
	creationOption, ok := governancerulescontrolplanesdk.GetMappingCreationOptionEnum(normalized)
	if !ok {
		return "", fmt.Errorf("unsupported governance rule creationOption %q", value)
	}
	return creationOption, nil
}

func governanceRuleTemplateFromSpecForCreation(
	ruleType string,
	creationOption governancerulescontrolplanesdk.CreationOptionEnum,
	spec governancerulescontrolplanev1beta1.GovernanceRuleSpec,
) (governancerulescontrolplanesdk.Template, error) {
	if creationOption == governancerulescontrolplanesdk.CreationOptionClone &&
		strings.TrimSpace(spec.RelatedResourceId) != "" &&
		governanceRuleTemplateSpecIsZero(spec.Template) {
		return nil, nil
	}
	return governanceRuleTemplateFromSpec(ruleType, spec.Template)
}

func governanceRuleTemplateSpecIsZero(template governancerulescontrolplanev1beta1.GovernanceRuleTemplate) bool {
	return strings.TrimSpace(template.JsonData) == "" &&
		strings.TrimSpace(template.Type) == "" &&
		strings.TrimSpace(template.Name) == "" &&
		strings.TrimSpace(template.Description) == "" &&
		strings.TrimSpace(template.DisplayName) == "" &&
		len(template.Tags) == 0 &&
		len(template.TagDefaults) == 0 &&
		len(template.Statements) == 0 &&
		len(template.Regions) == 0
}

func governanceRuleTemplateFromSpec(
	ruleType string,
	template governancerulescontrolplanev1beta1.GovernanceRuleTemplate,
) (governancerulescontrolplanesdk.Template, error) {
	if raw := strings.TrimSpace(template.JsonData); raw != "" {
		return governanceRuleTemplateFromJSON(raw, ruleType)
	}
	templateType, err := normalizeGovernanceRuleTemplateType(ruleType, template.Type)
	if err != nil {
		return nil, err
	}

	switch templateType {
	case string(governancerulescontrolplanesdk.GovernanceRuleTypeQuota):
		return governanceRuleQuotaTemplateFromSpec(template)
	case string(governancerulescontrolplanesdk.GovernanceRuleTypeTag):
		return governanceRuleTagTemplateFromSpec(template)
	case string(governancerulescontrolplanesdk.GovernanceRuleTypeAllowedRegions):
		return governanceRuleAllowedRegionsTemplateFromSpec(template)
	default:
		return nil, fmt.Errorf("unsupported governance rule template type %q", templateType)
	}
}

func governanceRuleQuotaTemplateFromSpec(
	template governancerulescontrolplanev1beta1.GovernanceRuleTemplate,
) (governancerulescontrolplanesdk.QuotaTemplate, error) {
	if strings.TrimSpace(template.DisplayName) == "" || len(template.Statements) == 0 {
		return governancerulescontrolplanesdk.QuotaTemplate{}, fmt.Errorf("QUOTA governance rule template requires displayName and statements")
	}
	return governancerulescontrolplanesdk.QuotaTemplate{
		DisplayName: common.String(strings.TrimSpace(template.DisplayName)),
		Description: optionalGovernanceRuleString(template.Description),
		Statements:  append([]string(nil), template.Statements...),
	}, nil
}

func governanceRuleTagTemplateFromSpec(
	template governancerulescontrolplanev1beta1.GovernanceRuleTemplate,
) (governancerulescontrolplanesdk.TagTemplate, error) {
	if strings.TrimSpace(template.Name) == "" {
		return governancerulescontrolplanesdk.TagTemplate{}, fmt.Errorf("TAG governance rule template requires name")
	}
	tags, err := governanceRuleTemplateTags(template.Tags)
	if err != nil {
		return governancerulescontrolplanesdk.TagTemplate{}, err
	}
	tagDefaults, err := governanceRuleTemplateTagDefaults(template.TagDefaults)
	if err != nil {
		return governancerulescontrolplanesdk.TagTemplate{}, err
	}
	return governancerulescontrolplanesdk.TagTemplate{
		Name:        common.String(strings.TrimSpace(template.Name)),
		Description: optionalGovernanceRuleString(template.Description),
		Tags:        tags,
		TagDefaults: tagDefaults,
	}, nil
}

func governanceRuleAllowedRegionsTemplateFromSpec(
	template governancerulescontrolplanev1beta1.GovernanceRuleTemplate,
) (governancerulescontrolplanesdk.AllowedRegionsTemplate, error) {
	if strings.TrimSpace(template.DisplayName) == "" || len(template.Regions) == 0 {
		return governancerulescontrolplanesdk.AllowedRegionsTemplate{}, fmt.Errorf("ALLOWED_REGIONS governance rule template requires displayName and regions")
	}
	return governancerulescontrolplanesdk.AllowedRegionsTemplate{
		DisplayName: common.String(strings.TrimSpace(template.DisplayName)),
		Description: optionalGovernanceRuleString(template.Description),
		Regions:     append([]string(nil), template.Regions...),
	}, nil
}

func governanceRuleTemplateFromJSON(raw string, fallbackType string) (governancerulescontrolplanesdk.Template, error) {
	var discriminator struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(raw), &discriminator); err != nil {
		return nil, fmt.Errorf("decode governance rule template discriminator: %w", err)
	}
	templateType, err := normalizeGovernanceRuleTemplateType(fallbackType, discriminator.Type)
	if err != nil {
		return nil, err
	}

	switch templateType {
	case string(governancerulescontrolplanesdk.GovernanceRuleTypeQuota):
		var body governancerulescontrolplanesdk.QuotaTemplate
		if err := json.Unmarshal([]byte(raw), &body); err != nil {
			return nil, fmt.Errorf("decode QUOTA governance rule template: %w", err)
		}
		return body, nil
	case string(governancerulescontrolplanesdk.GovernanceRuleTypeTag):
		var body governancerulescontrolplanesdk.TagTemplate
		if err := json.Unmarshal([]byte(raw), &body); err != nil {
			return nil, fmt.Errorf("decode TAG governance rule template: %w", err)
		}
		return body, nil
	case string(governancerulescontrolplanesdk.GovernanceRuleTypeAllowedRegions):
		var body governancerulescontrolplanesdk.AllowedRegionsTemplate
		if err := json.Unmarshal([]byte(raw), &body); err != nil {
			return nil, fmt.Errorf("decode ALLOWED_REGIONS governance rule template: %w", err)
		}
		return body, nil
	default:
		return nil, fmt.Errorf("unsupported governance rule template type %q", templateType)
	}
}

func normalizeGovernanceRuleTemplateType(ruleType string, templateType string) (string, error) {
	normalizedRuleType := strings.ToUpper(strings.TrimSpace(ruleType))
	normalizedTemplateType := strings.ToUpper(strings.TrimSpace(templateType))
	if normalizedTemplateType == "" {
		normalizedTemplateType = normalizedRuleType
	}
	if normalizedRuleType != "" && normalizedTemplateType != normalizedRuleType {
		return "", fmt.Errorf("governance rule template type %q conflicts with spec type %q", templateType, ruleType)
	}
	if _, err := normalizeGovernanceRuleType(normalizedTemplateType); err != nil {
		return "", err
	}
	return normalizedTemplateType, nil
}

func governanceRuleTemplateTags(
	tags []governancerulescontrolplanev1beta1.GovernanceRuleTemplateTag,
) ([]governancerulescontrolplanesdk.Tag, error) {
	if len(tags) == 0 {
		return nil, nil
	}
	result := make([]governancerulescontrolplanesdk.Tag, 0, len(tags))
	for _, tag := range tags {
		if strings.TrimSpace(tag.Name) == "" {
			return nil, fmt.Errorf("TAG governance rule template tag requires name")
		}
		validator, err := governanceRuleTagValidator(tag.Validator)
		if err != nil {
			return nil, err
		}
		result = append(result, governancerulescontrolplanesdk.Tag{
			Name:           common.String(strings.TrimSpace(tag.Name)),
			Description:    optionalGovernanceRuleString(tag.Description),
			IsCostTracking: common.Bool(tag.IsCostTracking),
			Validator:      validator,
		})
	}
	return result, nil
}

func governanceRuleTemplateTagDefaults(
	defaults []governancerulescontrolplanev1beta1.GovernanceRuleTemplateTagDefault,
) ([]governancerulescontrolplanesdk.TagDefault, error) {
	if len(defaults) == 0 {
		return nil, nil
	}
	result := make([]governancerulescontrolplanesdk.TagDefault, 0, len(defaults))
	for _, tagDefault := range defaults {
		if strings.TrimSpace(tagDefault.TagName) == "" {
			return nil, fmt.Errorf("TAG governance rule template tag default requires tagName")
		}
		result = append(result, governancerulescontrolplanesdk.TagDefault{
			TagName:    common.String(strings.TrimSpace(tagDefault.TagName)),
			Value:      common.String(tagDefault.Value),
			IsRequired: common.Bool(tagDefault.IsRequired),
		})
	}
	return result, nil
}

func governanceRuleTagValidator(
	validator governancerulescontrolplanev1beta1.GovernanceRuleTemplateTagValidator,
) (governancerulescontrolplanesdk.BaseTagDefinitionValidator, error) {
	if raw := strings.TrimSpace(validator.JsonData); raw != "" {
		body, err := governanceRuleTagValidatorFromJSON(raw, validator.ValidatorType)
		if err != nil {
			return nil, err
		}
		return body, nil
	}
	validatorType := strings.ToUpper(strings.TrimSpace(validator.ValidatorType))
	if validatorType == "" && len(validator.Values) != 0 {
		validatorType = string(governancerulescontrolplanesdk.BaseTagDefinitionValidatorValidatorTypeEnumvalue)
	}

	switch validatorType {
	case "":
		return nil, nil
	case string(governancerulescontrolplanesdk.BaseTagDefinitionValidatorValidatorTypeDefault):
		return governancerulescontrolplanesdk.DefaultTagDefinitionValidator{}, nil
	case string(governancerulescontrolplanesdk.BaseTagDefinitionValidatorValidatorTypeEnumvalue):
		return governancerulescontrolplanesdk.EnumTagDefinitionValidator{Values: append([]string(nil), validator.Values...)}, nil
	default:
		return nil, fmt.Errorf("unsupported governance rule tag validator type %q", validator.ValidatorType)
	}
}

func governanceRuleTagValidatorFromJSON(
	raw string,
	fallbackType string,
) (governancerulescontrolplanesdk.BaseTagDefinitionValidator, error) {
	var discriminator struct {
		ValidatorType string `json:"validatorType"`
	}
	if err := json.Unmarshal([]byte(raw), &discriminator); err != nil {
		return nil, fmt.Errorf("decode governance rule tag validator discriminator: %w", err)
	}
	validatorType := strings.ToUpper(strings.TrimSpace(discriminator.ValidatorType))
	if validatorType == "" {
		validatorType = strings.ToUpper(strings.TrimSpace(fallbackType))
	}
	switch validatorType {
	case string(governancerulescontrolplanesdk.BaseTagDefinitionValidatorValidatorTypeDefault):
		var body governancerulescontrolplanesdk.DefaultTagDefinitionValidator
		if err := json.Unmarshal([]byte(raw), &body); err != nil {
			return nil, err
		}
		return body, nil
	case string(governancerulescontrolplanesdk.BaseTagDefinitionValidatorValidatorTypeEnumvalue):
		var body governancerulescontrolplanesdk.EnumTagDefinitionValidator
		if err := json.Unmarshal([]byte(raw), &body); err != nil {
			return nil, err
		}
		return body, nil
	default:
		return nil, fmt.Errorf("unsupported governance rule tag validator type %q", validatorType)
	}
}

func optionalGovernanceRuleString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return common.String(strings.TrimSpace(value))
}

func guardGovernanceRuleExistingBeforeCreate(
	_ context.Context,
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	identity, err := resolveGovernanceRuleIdentityValue(resource)
	if err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}
	if identity.compartmentID == "" || identity.displayName == "" || identity.ruleType == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func resolveGovernanceRuleIdentity(
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
) (any, error) {
	return resolveGovernanceRuleIdentityValue(resource)
}

func resolveGovernanceRuleIdentityValue(
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
) (governanceRuleIdentity, error) {
	if resource == nil {
		return governanceRuleIdentity{}, fmt.Errorf("governance rule resource is nil")
	}
	createDetails, err := governanceRuleCreateDetailsFromSpec(resource.Spec)
	if err != nil {
		return governanceRuleIdentity{}, err
	}
	return governanceRuleIdentity{
		compartmentID: strings.TrimSpace(governanceRuleStringValue(createDetails.CompartmentId)),
		displayName:   strings.TrimSpace(governanceRuleStringValue(createDetails.DisplayName)),
		ruleType:      string(createDetails.Type),
	}, nil
}

func recordGovernanceRulePathIdentity(
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	identity any,
) {
	if resource == nil || strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" {
		return
	}
	typed, ok := identity.(governanceRuleIdentity)
	if !ok {
		return
	}
	resource.Status.CompartmentId = typed.compartmentID
	resource.Status.DisplayName = typed.displayName
	resource.Status.Type = typed.ruleType
}

func validateGovernanceRuleCreateOnlyDriftForResponse(
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("governance rule resource is nil")
	}
	current, ok := governanceRuleFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current governance rule response does not expose a governance rule body")
	}
	return validateGovernanceRuleCreateOnlyDrift(resource.Spec, current)
}

func validateGovernanceRuleCreateOnlyDrift(
	spec governancerulescontrolplanev1beta1.GovernanceRuleSpec,
	current governancerulescontrolplanesdk.GovernanceRule,
) error {
	desired, err := governanceRuleCreateDetailsFromSpec(spec)
	if err != nil {
		return err
	}

	var drift []string
	if !governanceRuleStringPtrEqual(current.CompartmentId, governanceRuleStringValue(desired.CompartmentId)) {
		drift = append(drift, "compartmentId")
	}
	if current.Type != "" && current.Type != desired.Type {
		drift = append(drift, "type")
	}
	if current.CreationOption != "" && current.CreationOption != desired.CreationOption {
		drift = append(drift, "creationOption")
	}
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("governance rule create-only field drift is not supported: %s", strings.Join(drift, ", "))
}

func wrapGovernanceRuleListCalls(hooks *GovernanceRuleRuntimeHooks) {
	if hooks == nil || hooks.List.Call == nil {
		return
	}
	list := hooks.List.Call
	hooks.List.Call = func(
		ctx context.Context,
		request governancerulescontrolplanesdk.ListGovernanceRulesRequest,
	) (governancerulescontrolplanesdk.ListGovernanceRulesResponse, error) {
		return listGovernanceRulePages(ctx, list, request)
	}
}

func listGovernanceRulePages(
	ctx context.Context,
	list func(context.Context, governancerulescontrolplanesdk.ListGovernanceRulesRequest) (governancerulescontrolplanesdk.ListGovernanceRulesResponse, error),
	request governancerulescontrolplanesdk.ListGovernanceRulesRequest,
) (governancerulescontrolplanesdk.ListGovernanceRulesResponse, error) {
	var combined governancerulescontrolplanesdk.ListGovernanceRulesResponse
	for {
		response, err := list(ctx, request)
		if err != nil {
			return governancerulescontrolplanesdk.ListGovernanceRulesResponse{}, conservativeGovernanceRuleNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func handleGovernanceRuleDeleteError(
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	err error,
) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return conservativeGovernanceRuleNotFoundError(err, "delete")
}

func applyGovernanceRuleDeleteConfirmOutcome(
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if ambiguous, ok := asAmbiguousGovernanceRuleNotFoundError(response); ok {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, ambiguous)
		}
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, ambiguous
	}
	current, ok := governanceRuleFromResponse(response)
	if !ok || current.LifecycleState != governancerulescontrolplanesdk.GovernanceRuleLifecycleStateActive {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage != generatedruntime.DeleteConfirmStageAfterRequest {
		return generatedruntime.DeleteOutcome{}, nil
	}
	markGovernanceRuleDeletePending(resource, governanceRuleStringValue(current.Id))
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
}

func markGovernanceRuleDeletePending(
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	resourceID string,
) {
	if resource == nil {
		return
	}
	if strings.TrimSpace(resourceID) != "" {
		resource.Status.Id = strings.TrimSpace(resourceID)
		resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(resourceID))
	}
	now := metav1.Now()
	message := "OCI GovernanceRule delete is in progress"
	status := &resource.Status.OsokStatus
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.UpdatedAt = &now
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, loggerutil.OSOKLogger{})
}

func wrapGovernanceRuleDeleteWorkRequestConfirmation(hooks *GovernanceRuleRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getGovernanceRule := hooks.Get.Call
	getWorkRequest := hooks.Async.GetWorkRequest
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate GovernanceRuleServiceClient) GovernanceRuleServiceClient {
		return governanceRuleDeleteWorkRequestConfirmationClient{
			GovernanceRuleServiceClient: delegate,
			getGovernanceRule:           getGovernanceRule,
			getWorkRequest:              getWorkRequest,
		}
	})
}

type governanceRuleDeleteWorkRequestConfirmationClient struct {
	GovernanceRuleServiceClient
	getGovernanceRule func(context.Context, governancerulescontrolplanesdk.GetGovernanceRuleRequest) (governancerulescontrolplanesdk.GetGovernanceRuleResponse, error)
	getWorkRequest    func(context.Context, string) (any, error)
}

func (c governanceRuleDeleteWorkRequestConfirmationClient) Delete(
	ctx context.Context,
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
) (bool, error) {
	if deleted, handled, err := c.resumeDeleteWorkRequest(ctx, resource); handled {
		return deleted, err
	}
	return c.GovernanceRuleServiceClient.Delete(ctx, resource)
}

func (c governanceRuleDeleteWorkRequestConfirmationClient) resumeDeleteWorkRequest(
	ctx context.Context,
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
) (bool, bool, error) {
	workRequestID := currentGovernanceRuleDeleteWorkRequest(resource)
	if workRequestID == "" {
		return false, false, nil
	}
	if c.getWorkRequest == nil {
		return false, true, fmt.Errorf("GovernanceRule work request polling is not configured")
	}

	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		if resource != nil {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		}
		return false, true, err
	}
	var status *shared.OSOKStatus
	if resource != nil {
		status = &resource.Status.OsokStatus
	}
	current, err := buildGovernanceRuleWorkRequestAsyncOperation(status, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return false, true, c.failWorkRequestForDelete(resource, nil, err)
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.markWorkRequestForDelete(resource, current, shared.OSOKAsyncClassPending, governanceRuleWorkRequestMessage(current.Phase, workRequest))
		return false, true, nil
	case shared.OSOKAsyncClassSucceeded:
		deleted, err := c.confirmSucceededDeleteWorkRequest(ctx, resource, workRequest, current, workRequestID)
		return deleted, true, err
	case shared.OSOKAsyncClassFailed,
		shared.OSOKAsyncClassCanceled,
		shared.OSOKAsyncClassAttention,
		shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("GovernanceRule delete work request %s finished with status %s", workRequestID, current.RawStatus)
		return false, true, c.failWorkRequestForDelete(resource, current, err)
	default:
		err := fmt.Errorf("GovernanceRule delete work request %s projected unsupported async class %s", workRequestID, current.NormalizedClass)
		return false, true, c.failWorkRequestForDelete(resource, current, err)
	}
}

func currentGovernanceRuleDeleteWorkRequest(resource *governancerulescontrolplanev1beta1.GovernanceRule) string {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete {
		return ""
	}
	return strings.TrimSpace(current.WorkRequestID)
}

func buildGovernanceRuleWorkRequestAsyncOperation(
	status *shared.OSOKStatus,
	workRequest any,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	current, err := governanceRuleWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}
	derivedPhase, ok, err := resolveGovernanceRuleWorkRequestPhase(current)
	if err != nil {
		return nil, err
	}
	if ok {
		if fallbackPhase != "" && fallbackPhase != derivedPhase {
			return nil, fmt.Errorf(
				"GovernanceRule work request %s exposes phase %q while delete expected %q",
				governanceRuleStringValue(current.Id),
				derivedPhase,
				fallbackPhase,
			)
		}
		fallbackPhase = derivedPhase
	}
	action, err := resolveGovernanceRuleWorkRequestAction(current)
	if err != nil {
		return nil, err
	}
	operation, err := servicemanager.BuildWorkRequestAsyncOperation(status, governanceRuleWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        action,
		RawOperationType: string(current.OperationType),
		WorkRequestID:    governanceRuleStringValue(current.Id),
		PercentComplete:  current.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
	if err != nil {
		return nil, err
	}
	if message := governanceRuleWorkRequestMessage(operation.Phase, current); message != "" {
		operation.Message = message
	}
	return operation, nil
}

func (c governanceRuleDeleteWorkRequestConfirmationClient) confirmSucceededDeleteWorkRequest(
	ctx context.Context,
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	workRequest any,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
) (bool, error) {
	resourceID := trackedGovernanceRuleID(resource)
	if resourceID == "" {
		recoveredID, err := recoverGovernanceRuleIDFromWorkRequest(resource, workRequest, shared.OSOKAsyncPhaseDelete)
		if err != nil {
			return false, c.failWorkRequestForDelete(resource, current, err)
		}
		resourceID = strings.TrimSpace(recoveredID)
	}
	if resourceID == "" || c.getGovernanceRule == nil {
		return c.GovernanceRuleServiceClient.Delete(ctx, resource)
	}

	_, err := c.getGovernanceRule(ctx, governancerulescontrolplanesdk.GetGovernanceRuleRequest{
		GovernanceRuleId: common.String(resourceID),
	})
	if err == nil || errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		return c.GovernanceRuleServiceClient.Delete(ctx, resource)
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		err = fmt.Errorf(
			"GovernanceRule delete work request %s succeeded but confirmation read returned ambiguous 404 NotAuthorizedOrNotFound: %w",
			strings.TrimSpace(workRequestID),
			err,
		)
		return false, c.failWorkRequestForDelete(resource, current, err)
	}
	return false, c.failWorkRequestForDelete(resource, current, err)
}

func trackedGovernanceRuleID(resource *governancerulescontrolplanev1beta1.GovernanceRule) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func (c governanceRuleDeleteWorkRequestConfirmationClient) markWorkRequestForDelete(
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	current *shared.OSOKAsyncOperation,
	class shared.OSOKAsyncNormalizedClass,
	message string,
) {
	if resource == nil || current == nil {
		return
	}
	next := *current
	next.NormalizedClass = class
	next.Message = strings.TrimSpace(message)
	next.UpdatedAt = nil
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, loggerutil.OSOKLogger{})
}

func (c governanceRuleDeleteWorkRequestConfirmationClient) failWorkRequestForDelete(
	resource *governancerulescontrolplanev1beta1.GovernanceRule,
	current *shared.OSOKAsyncOperation,
	err error,
) error {
	if resource == nil || err == nil {
		return err
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	if current == nil {
		return err
	}
	class := current.NormalizedClass
	switch class {
	case shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
	default:
		class = shared.OSOKAsyncClassFailed
	}
	c.markWorkRequestForDelete(resource, current, class, err.Error())
	return err
}

func conservativeGovernanceRuleNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("governance rule %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousGovernanceRuleNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousGovernanceRuleNotFoundError{message: message}
}

func asAmbiguousGovernanceRuleNotFoundError(value any) (ambiguousGovernanceRuleNotFoundError, bool) {
	err, _ := value.(error)
	var ambiguous ambiguousGovernanceRuleNotFoundError
	if errors.As(err, &ambiguous) {
		return ambiguous, true
	}
	return ambiguousGovernanceRuleNotFoundError{}, false
}

func getGovernanceRuleWorkRequest(
	ctx context.Context,
	client governanceRuleWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize GovernanceRule work request OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("GovernanceRule work request OCI client is not configured")
	}
	response, err := client.GetWorkRequest(ctx, governancerulescontrolplanesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveGovernanceRuleWorkRequestAction(workRequest any) (string, error) {
	current, err := governanceRuleWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveGovernanceRuleWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := governanceRuleWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case governancerulescontrolplanesdk.OperationTypeCreateGovernanceRule,
		governancerulescontrolplanesdk.OperationTypeCreateEnforcedQuotaGovernanceRule,
		governancerulescontrolplanesdk.OperationTypeCreateEnforcedTagGovernanceRule:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case governancerulescontrolplanesdk.OperationTypeUpdateGovernanceRule,
		governancerulescontrolplanesdk.OperationTypeUpdateEnforcedQuotaGovernanceRule,
		governancerulescontrolplanesdk.OperationTypeUpdateEnforcedTagGovernanceRule:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case governancerulescontrolplanesdk.OperationTypeDeleteGovernanceRule,
		governancerulescontrolplanesdk.OperationTypeDeleteEnforcedQuotaGovernanceRule,
		governancerulescontrolplanesdk.OperationTypeDeleteEnforcedTagGovernanceRule:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func recoverGovernanceRuleIDFromWorkRequest(
	_ *governancerulescontrolplanev1beta1.GovernanceRule,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := governanceRuleWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	if id, ok := resolveGovernanceRuleIDFromWorkRequestResources(current.Resources, phase, true); ok {
		return id, nil
	}
	if id, ok := resolveGovernanceRuleIDFromWorkRequestResources(current.Resources, phase, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("GovernanceRule work request %s does not expose a governance rule identifier", governanceRuleStringValue(current.Id))
}

func resolveGovernanceRuleIDFromWorkRequestResources(
	resources []governancerulescontrolplanesdk.WorkRequestResource,
	phase shared.OSOKAsyncPhase,
	requireActionMatch bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if !isGovernanceRuleWorkRequestResource(resource) {
			continue
		}
		if requireActionMatch && !governanceRuleWorkRequestActionMatchesPhase(resource.ActionType, phase) {
			continue
		}
		id := strings.TrimSpace(governanceRuleStringValue(resource.Identifier))
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

func isGovernanceRuleWorkRequestResource(resource governancerulescontrolplanesdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(governanceRuleStringValue(resource.EntityType)))
	if strings.Contains(entityType, "governance") && strings.Contains(entityType, "rule") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(governanceRuleStringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/governancerules/")
}

func governanceRuleWorkRequestActionMatchesPhase(
	action governancerulescontrolplanesdk.ActionTypeEnum,
	phase shared.OSOKAsyncPhase,
) bool {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return action == governancerulescontrolplanesdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return action == governancerulescontrolplanesdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return action == governancerulescontrolplanesdk.ActionTypeDeleted
	default:
		return false
	}
}

func governanceRuleWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := governanceRuleWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("GovernanceRule %s work request %s is %s", phase, governanceRuleStringValue(current.Id), current.Status)
}

func governanceRuleWorkRequestFromAny(workRequest any) (governancerulescontrolplanesdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case governancerulescontrolplanesdk.WorkRequest:
		return current, nil
	case *governancerulescontrolplanesdk.WorkRequest:
		if current == nil {
			return governancerulescontrolplanesdk.WorkRequest{}, fmt.Errorf("GovernanceRule work request is nil")
		}
		return *current, nil
	default:
		return governancerulescontrolplanesdk.WorkRequest{}, fmt.Errorf("unexpected GovernanceRule work request type %T", workRequest)
	}
}

func clearTrackedGovernanceRuleIdentity(resource *governancerulescontrolplanev1beta1.GovernanceRule) {
	if resource == nil {
		return
	}
	resource.Status = governancerulescontrolplanev1beta1.GovernanceRuleStatus{}
}

func governanceRuleFromResponse(response any) (governancerulescontrolplanesdk.GovernanceRule, bool) {
	switch current := response.(type) {
	case governancerulescontrolplanesdk.CreateGovernanceRuleResponse:
		return current.GovernanceRule, current.Id != nil
	case *governancerulescontrolplanesdk.CreateGovernanceRuleResponse:
		if current == nil {
			return governancerulescontrolplanesdk.GovernanceRule{}, false
		}
		return current.GovernanceRule, current.Id != nil
	case governancerulescontrolplanesdk.GetGovernanceRuleResponse:
		return current.GovernanceRule, current.Id != nil
	case *governancerulescontrolplanesdk.GetGovernanceRuleResponse:
		if current == nil {
			return governancerulescontrolplanesdk.GovernanceRule{}, false
		}
		return current.GovernanceRule, current.Id != nil
	case governancerulescontrolplanesdk.GovernanceRule:
		return current, current.Id != nil
	case *governancerulescontrolplanesdk.GovernanceRule:
		if current == nil {
			return governancerulescontrolplanesdk.GovernanceRule{}, false
		}
		return *current, current.Id != nil
	default:
		return governancerulescontrolplanesdk.GovernanceRule{}, false
	}
}

func cloneGovernanceRuleStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneGovernanceRuleDefinedTags(values map[string]map[string]interface{}) map[string]map[string]interface{} {
	if values == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(values))
	for outerKey, outerValue := range values {
		inner := make(map[string]interface{}, len(outerValue))
		for innerKey, innerValue := range outerValue {
			inner[innerKey] = innerValue
		}
		cloned[outerKey] = inner
	}
	return cloned
}

func governanceRuleSemanticEqual(left any, right any) bool {
	leftJSON, leftOK := governanceRuleSemanticJSON(left)
	rightJSON, rightOK := governanceRuleSemanticJSON(right)
	return leftOK && rightOK && reflect.DeepEqual(leftJSON, rightJSON)
}

func governanceRuleSemanticJSON(value any) (map[string]any, bool) {
	if value == nil {
		return nil, true
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, false
	}
	return decoded, true
}

func governanceRuleStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func governanceRuleStringPtrEqual(value *string, want string) bool {
	if value == nil {
		return strings.TrimSpace(want) == ""
	}
	return strings.TrimSpace(*value) == strings.TrimSpace(want)
}
