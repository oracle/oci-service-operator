/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package opsiconfiguration

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	opsiConfigurationKind       = "OpsiConfiguration"
	opsiConfigurationEntityType = "opsiconfiguration"
	opsiConfigurationTypeUX     = string(opsisdk.OpsiConfigurationTypeUxConfiguration)
)

type opsiConfigurationOCIClient interface {
	CreateOpsiConfiguration(context.Context, opsisdk.CreateOpsiConfigurationRequest) (opsisdk.CreateOpsiConfigurationResponse, error)
	GetOpsiConfiguration(context.Context, opsisdk.GetOpsiConfigurationRequest) (opsisdk.GetOpsiConfigurationResponse, error)
	ListOpsiConfigurations(context.Context, opsisdk.ListOpsiConfigurationsRequest) (opsisdk.ListOpsiConfigurationsResponse, error)
	UpdateOpsiConfiguration(context.Context, opsisdk.UpdateOpsiConfigurationRequest) (opsisdk.UpdateOpsiConfigurationResponse, error)
	DeleteOpsiConfiguration(context.Context, opsisdk.DeleteOpsiConfigurationRequest) (opsisdk.DeleteOpsiConfigurationResponse, error)
	GetWorkRequest(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

type opsiConfigurationListCall func(context.Context, opsisdk.ListOpsiConfigurationsRequest) (opsisdk.ListOpsiConfigurationsResponse, error)

type opsiConfigurationDesiredState struct {
	FromJSON          bool
	JSONFields        map[string]bool
	CompartmentID     string
	DisplayName       string
	Description       string
	FreeformTags      map[string]string
	DefinedTags       map[string]map[string]interface{}
	SystemTags        map[string]map[string]interface{}
	OpsiConfigType    string
	CreateConfigItems []opsisdk.CreateConfigurationItemDetails
	UpdateConfigItems []opsisdk.UpdateConfigurationItemDetails
}

type opsiConfigurationView struct {
	ID               string
	CompartmentID    string
	DisplayName      string
	Description      string
	FreeformTags     map[string]string
	DefinedTags      map[string]map[string]interface{}
	SystemTags       map[string]map[string]interface{}
	TimeCreated      *common.SDKTime
	TimeUpdated      *common.SDKTime
	LifecycleState   opsisdk.OpsiConfigurationLifecycleStateEnum
	LifecycleDetails string
	OpsiConfigType   string
	ConfigItems      any
}

type opsiConfigurationAuthShapedNotFoundError struct {
	message      string
	opcRequestID string
}

func (e opsiConfigurationAuthShapedNotFoundError) Error() string {
	return e.message
}

func (e opsiConfigurationAuthShapedNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

var opsiConfigurationWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(opsisdk.OperationStatusAccepted),
		string(opsisdk.OperationStatusInProgress),
		string(opsisdk.OperationStatusWaiting),
		string(opsisdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(opsisdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(opsisdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(opsisdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(opsisdk.ActionTypeCreated)},
	UpdateActionTokens:    []string{string(opsisdk.ActionTypeUpdated)},
	DeleteActionTokens:    []string{string(opsisdk.ActionTypeDeleted)},
}

var (
	pendingOpsiConfigurationCreateBodies sync.Map
	pendingOpsiConfigurationUpdateBodies sync.Map
)

func init() {
	registerOpsiConfigurationRuntimeHooksMutator(func(manager *OpsiConfigurationServiceManager, hooks *OpsiConfigurationRuntimeHooks) {
		client, initErr := newOpsiConfigurationOperationsInsightsClient(manager)
		applyOpsiConfigurationRuntimeHooks(hooks, client, initErr)
	})
}

func newOpsiConfigurationOperationsInsightsClient(manager *OpsiConfigurationServiceManager) (opsiConfigurationOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", opsiConfigurationKind)
	}
	client, err := opsisdk.NewOperationsInsightsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyOpsiConfigurationRuntimeHooks(
	hooks *OpsiConfigurationRuntimeHooks,
	client opsiConfigurationOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newOpsiConfigurationRuntimeSemantics()
	hooks.BuildCreateBody = buildOpsiConfigurationCreateBody
	hooks.BuildUpdateBody = buildOpsiConfigurationUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardOpsiConfigurationExistingBeforeCreate
	hooks.Create.Fields = opsiConfigurationCreateFields()
	hooks.Get.Fields = opsiConfigurationGetFields()
	hooks.List.Fields = opsiConfigurationListFields()
	hooks.Update.Fields = opsiConfigurationUpdateFields()
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedOpsiConfigurationIdentity
	hooks.StatusHooks.ProjectStatus = opsiConfigurationStatusFromResponse
	hooks.StatusHooks.MarkDeleted = markOpsiConfigurationDeleted
	hooks.DeleteHooks.HandleError = handleOpsiConfigurationDeleteError
	hooks.Async.Adapter = opsiConfigurationWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getOpsiConfigurationWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveOpsiConfigurationWorkRequestAction
	hooks.Async.ResolvePhase = resolveOpsiConfigurationWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverOpsiConfigurationIDFromWorkRequest
	hooks.Async.Message = opsiConfigurationWorkRequestMessage
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateOpsiConfigurationCreateOnlyDrift
	if hooks.Create.Call != nil {
		call := hooks.Create.Call
		hooks.Create.Call = func(ctx context.Context, request opsisdk.CreateOpsiConfigurationRequest) (opsisdk.CreateOpsiConfigurationResponse, error) {
			var err error
			request, err = withOpsiConfigurationCreateRequestBody(request)
			if err != nil {
				return opsisdk.CreateOpsiConfigurationResponse{}, err
			}
			return call(ctx, withOpsiConfigurationCreateResponseFields(request))
		}
	}
	if hooks.Update.Call != nil {
		call := hooks.Update.Call
		hooks.Update.Call = func(ctx context.Context, request opsisdk.UpdateOpsiConfigurationRequest) (opsisdk.UpdateOpsiConfigurationResponse, error) {
			var err error
			request, err = withOpsiConfigurationUpdateRequestBody(request)
			if err != nil {
				return opsisdk.UpdateOpsiConfigurationResponse{}, err
			}
			return call(ctx, request)
		}
	}
	if hooks.Get.Call != nil {
		call := hooks.Get.Call
		hooks.Get.Call = func(ctx context.Context, request opsisdk.GetOpsiConfigurationRequest) (opsisdk.GetOpsiConfigurationResponse, error) {
			return call(ctx, withOpsiConfigurationGetResponseFields(request))
		}
	}
	if hooks.List.Call != nil {
		hooks.List.Call = listOpsiConfigurationPages(hooks.List.Call)
	}
}

func newOpsiConfigurationRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "opsi",
		FormalSlug:    "opsiconfiguration",
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
			ProvisioningStates: []string{string(opsisdk.OpsiConfigurationLifecycleStateCreating)},
			UpdatingStates:     []string{string(opsisdk.OpsiConfigurationLifecycleStateUpdating)},
			ActiveStates:       []string{string(opsisdk.OpsiConfigurationLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(opsisdk.OpsiConfigurationLifecycleStateDeleting)},
			TerminalStates: []string{string(opsisdk.OpsiConfigurationLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "opsiConfigType"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"displayName", "description", "freeformTags", "definedTags", "systemTags", "jsonData"},
			Mutable:         []string{"displayName", "description", "freeformTags", "definedTags", "systemTags", "jsonData"},
			ForceNew:        []string{"compartmentId", "opsiConfigType"},
			ConflictsWith:   map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: opsiConfigurationEntityType, Action: string(opsisdk.OperationTypeCreateOpsiConfiguration)},
			},
			Update: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: opsiConfigurationEntityType, Action: string(opsisdk.OperationTypeUpdateOpsiConfiguration)},
			},
			Delete: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: opsiConfigurationEntityType, Action: string(opsisdk.OperationTypeDeleteOpsiConfiguration)},
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetOpsiConfiguration",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: opsiConfigurationEntityType, Action: string(opsisdk.OperationTypeCreateOpsiConfiguration)}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetOpsiConfiguration",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: opsiConfigurationEntityType, Action: string(opsisdk.OperationTypeUpdateOpsiConfiguration)}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: opsiConfigurationEntityType, Action: string(opsisdk.OperationTypeDeleteOpsiConfiguration)}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func opsiConfigurationGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OpsiConfigurationId", RequestName: "opsiConfigurationId", Contribution: "path", PreferResourceID: true},
	}
}

func opsiConfigurationListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func opsiConfigurationCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OpsiConfigField", RequestName: "opsiConfigField", Contribution: "query"},
		{FieldName: "ConfigItemCustomStatus", RequestName: "configItemCustomStatus", Contribution: "query"},
		{FieldName: "ConfigItemsApplicableContext", RequestName: "configItemsApplicableContext", Contribution: "query"},
		{FieldName: "ConfigItemField", RequestName: "configItemField", Contribution: "query"},
	}
}

func opsiConfigurationUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OpsiConfigurationId", RequestName: "opsiConfigurationId", Contribution: "path", PreferResourceID: true},
	}
}

func guardOpsiConfigurationExistingBeforeCreate(
	_ context.Context,
	resource *opsiv1beta1.OpsiConfiguration,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	desired, err := opsiConfigurationDesiredStateForResource(resource)
	if err != nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, err
	}
	if strings.TrimSpace(desired.CompartmentID) == "" || strings.TrimSpace(desired.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildOpsiConfigurationCreateBody(
	_ context.Context,
	resource *opsiv1beta1.OpsiConfiguration,
	_ string,
) (any, error) {
	details, err := opsiConfigurationCreateBody(resource)
	if err != nil {
		return nil, err
	}
	rememberOpsiConfigurationCreateBody(resource, details)
	return details, nil
}

//nolint:gocyclo // OpsiConfiguration create accepts either structured spec fields or UX jsonData defaults.
func opsiConfigurationCreateBody(resource *opsiv1beta1.OpsiConfiguration) (opsisdk.CreateOpsiUxConfigurationDetails, error) {
	if resource == nil {
		return opsisdk.CreateOpsiUxConfigurationDetails{}, fmt.Errorf("%s resource is nil", opsiConfigurationKind)
	}

	if raw := strings.TrimSpace(resource.Spec.JsonData); raw != "" {
		details, jsonType, err := opsiConfigurationCreateDetailsFromJSON(raw)
		if err != nil {
			return opsisdk.CreateOpsiUxConfigurationDetails{}, err
		}
		details, err = opsiConfigurationApplyCreateSpecDefaults(details, resource.Spec, jsonType)
		if err != nil {
			return opsisdk.CreateOpsiUxConfigurationDetails{}, err
		}
		return details, nil
	}

	configType, err := normalizeOpsiConfigurationType(resource.Spec.OpsiConfigType)
	if err != nil {
		return opsisdk.CreateOpsiUxConfigurationDetails{}, err
	}
	if configType != opsiConfigurationTypeUX {
		return opsisdk.CreateOpsiUxConfigurationDetails{}, unsupportedOpsiConfigurationTypeError(configType)
	}

	details := opsisdk.CreateOpsiUxConfigurationDetails{
		CompartmentId: common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:   common.String(strings.TrimSpace(resource.Spec.DisplayName)),
	}
	if strings.TrimSpace(resource.Spec.Description) != "" {
		details.Description = common.String(strings.TrimSpace(resource.Spec.Description))
	}
	if resource.Spec.FreeformTags != nil {
		details.FreeformTags = cloneOpsiConfigurationStringMap(resource.Spec.FreeformTags)
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = opsiConfigurationDefinedTags(resource.Spec.DefinedTags)
	}
	if resource.Spec.SystemTags != nil {
		details.SystemTags = opsiConfigurationDefinedTags(resource.Spec.SystemTags)
	}
	if err := validateOpsiConfigurationCreateDetails(details, configType); err != nil {
		return opsisdk.CreateOpsiUxConfigurationDetails{}, err
	}
	return details, nil
}

//nolint:gocyclo // Mutable field shaping intentionally stays beside the SDK update model for reviewability.
func buildOpsiConfigurationUpdateBody(
	_ context.Context,
	resource *opsiv1beta1.OpsiConfiguration,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return opsisdk.UpdateOpsiUxConfigurationDetails{}, false, fmt.Errorf("%s resource is nil", opsiConfigurationKind)
	}
	desired, err := opsiConfigurationDesiredStateForResource(resource)
	if err != nil {
		return opsisdk.UpdateOpsiUxConfigurationDetails{}, false, err
	}
	current, ok := opsiConfigurationViewFromResponse(currentResponse)
	if !ok {
		return opsisdk.UpdateOpsiUxConfigurationDetails{}, false, fmt.Errorf("current %s response does not expose an OpsiConfiguration body", opsiConfigurationKind)
	}

	details := opsisdk.UpdateOpsiUxConfigurationDetails{}
	updateNeeded := false
	updateNeeded = opsiConfigurationApplyStringUpdate(&details.DisplayName, desired.DisplayName, current.DisplayName, true) || updateNeeded
	updateNeeded = opsiConfigurationApplyStringUpdate(&details.Description, desired.Description, current.Description, desired.JSONFields["description"]) || updateNeeded
	if desired.FreeformTags != nil && !reflect.DeepEqual(current.FreeformTags, desired.FreeformTags) {
		details.FreeformTags = cloneOpsiConfigurationStringMap(desired.FreeformTags)
		updateNeeded = true
	}
	if desired.DefinedTags != nil && !reflect.DeepEqual(current.DefinedTags, desired.DefinedTags) {
		details.DefinedTags = cloneOpsiConfigurationNestedMap(desired.DefinedTags)
		updateNeeded = true
	}
	if desired.SystemTags != nil && !reflect.DeepEqual(current.SystemTags, desired.SystemTags) {
		details.SystemTags = cloneOpsiConfigurationNestedMap(desired.SystemTags)
		updateNeeded = true
	}
	if desired.JSONFields["configItems"] && !opsiConfigurationJSONEqual(
		normalizeOpsiConfigurationConfigItems(desired.UpdateConfigItems),
		normalizeOpsiConfigurationConfigItems(current.ConfigItems),
	) {
		details.ConfigItems = cloneOpsiConfigurationUpdateConfigItems(desired.UpdateConfigItems)
		updateNeeded = true
	}
	if updateNeeded {
		rememberOpsiConfigurationUpdateBody(current.ID, details)
	}
	return details, updateNeeded, nil
}

func opsiConfigurationApplyStringUpdate(target **string, desired string, current string, explicit bool) bool {
	desired = strings.TrimSpace(desired)
	current = strings.TrimSpace(current)
	if desired == "" && !explicit {
		return false
	}
	if desired == current {
		return false
	}
	*target = common.String(desired)
	return true
}

func opsiConfigurationCreateDetailsFromJSON(raw string) (opsisdk.CreateOpsiUxConfigurationDetails, string, error) {
	jsonType, err := opsiConfigurationTypeFromJSON(raw)
	if err != nil {
		return opsisdk.CreateOpsiUxConfigurationDetails{}, "", err
	}
	configType, err := normalizeOpsiConfigurationType(jsonType)
	if err != nil {
		return opsisdk.CreateOpsiUxConfigurationDetails{}, "", err
	}
	if configType != opsiConfigurationTypeUX {
		return opsisdk.CreateOpsiUxConfigurationDetails{}, "", unsupportedOpsiConfigurationTypeError(configType)
	}

	var details opsisdk.CreateOpsiUxConfigurationDetails
	if err := json.Unmarshal([]byte(raw), &details); err != nil {
		return opsisdk.CreateOpsiUxConfigurationDetails{}, "", fmt.Errorf("decode %s UX_CONFIGURATION jsonData: %w", opsiConfigurationKind, err)
	}
	return details, configType, nil
}

//nolint:gocyclo // Each branch preserves one optional spec default only when jsonData omitted it.
func opsiConfigurationApplyCreateSpecDefaults(
	details opsisdk.CreateOpsiUxConfigurationDetails,
	spec opsiv1beta1.OpsiConfigurationSpec,
	jsonType string,
) (opsisdk.CreateOpsiUxConfigurationDetails, error) {
	if err := validateOpsiConfigurationJSONSpecConflicts(spec, details, jsonType); err != nil {
		return opsisdk.CreateOpsiUxConfigurationDetails{}, err
	}

	if details.CompartmentId == nil && strings.TrimSpace(spec.CompartmentId) != "" {
		details.CompartmentId = common.String(strings.TrimSpace(spec.CompartmentId))
	}
	if details.DisplayName == nil && strings.TrimSpace(spec.DisplayName) != "" {
		details.DisplayName = common.String(strings.TrimSpace(spec.DisplayName))
	}
	if details.Description == nil && strings.TrimSpace(spec.Description) != "" {
		details.Description = common.String(strings.TrimSpace(spec.Description))
	}
	if details.FreeformTags == nil && spec.FreeformTags != nil {
		details.FreeformTags = cloneOpsiConfigurationStringMap(spec.FreeformTags)
	}
	if details.DefinedTags == nil && spec.DefinedTags != nil {
		details.DefinedTags = opsiConfigurationDefinedTags(spec.DefinedTags)
	}
	if details.SystemTags == nil && spec.SystemTags != nil {
		details.SystemTags = opsiConfigurationDefinedTags(spec.SystemTags)
	}
	if err := validateOpsiConfigurationCreateDetails(details, jsonType); err != nil {
		return opsisdk.CreateOpsiUxConfigurationDetails{}, err
	}
	return details, nil
}

func validateOpsiConfigurationJSONSpecConflicts(
	spec opsiv1beta1.OpsiConfigurationSpec,
	details opsisdk.CreateOpsiUxConfigurationDetails,
	jsonType string,
) error {
	var conflicts []string
	conflicts = appendPointerStringConflict(conflicts, "compartmentId", spec.CompartmentId, details.CompartmentId)
	conflicts = appendPointerStringConflict(conflicts, "displayName", spec.DisplayName, details.DisplayName)
	conflicts = appendPointerStringConflict(conflicts, "description", spec.Description, details.Description)
	conflicts = appendMapConflict(conflicts, "freeformTags", spec.FreeformTags, details.FreeformTags)
	conflicts = appendJSONConflict(conflicts, "definedTags", opsiConfigurationDefinedTags(spec.DefinedTags), details.DefinedTags)
	conflicts = appendJSONConflict(conflicts, "systemTags", opsiConfigurationDefinedTags(spec.SystemTags), details.SystemTags)
	if strings.TrimSpace(spec.OpsiConfigType) != "" {
		specType, err := normalizeOpsiConfigurationType(spec.OpsiConfigType)
		if err != nil {
			return err
		}
		if specType != jsonType {
			conflicts = append(conflicts, "opsiConfigType")
		}
	}
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf("%s jsonData conflicts with spec field(s): %s", opsiConfigurationKind, strings.Join(conflicts, ", "))
}

func validateOpsiConfigurationCreateDetails(details opsisdk.CreateOpsiUxConfigurationDetails, configType string) error {
	var missing []string
	if strings.TrimSpace(configType) == "" {
		missing = append(missing, "opsiConfigType")
	}
	if stringValue(details.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if stringValue(details.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%s create details missing required field(s): %s", opsiConfigurationKind, strings.Join(missing, ", "))
}

//nolint:gocyclo // Desired state normalization has to keep jsonData and structured spec precedence in one path.
func opsiConfigurationDesiredStateForResource(resource *opsiv1beta1.OpsiConfiguration) (opsiConfigurationDesiredState, error) {
	if resource == nil {
		return opsiConfigurationDesiredState{}, fmt.Errorf("%s resource is nil", opsiConfigurationKind)
	}

	rawJSON := strings.TrimSpace(resource.Spec.JsonData)
	jsonFields, err := opsiConfigurationJSONFields(rawJSON)
	if err != nil {
		return opsiConfigurationDesiredState{}, err
	}
	details, err := opsiConfigurationCreateBody(resource)
	if err != nil {
		return opsiConfigurationDesiredState{}, err
	}
	configType := opsiConfigurationTypeUX
	if rawJSON != "" {
		configType, err = opsiConfigurationTypeFromJSON(rawJSON)
		if err != nil {
			return opsiConfigurationDesiredState{}, err
		}
		configType, err = normalizeOpsiConfigurationType(firstString(configType, resource.Spec.OpsiConfigType))
		if err != nil {
			return opsiConfigurationDesiredState{}, err
		}
	} else if strings.TrimSpace(resource.Spec.OpsiConfigType) != "" {
		configType, err = normalizeOpsiConfigurationType(resource.Spec.OpsiConfigType)
		if err != nil {
			return opsiConfigurationDesiredState{}, err
		}
	}

	updateConfigItems, err := updateOpsiConfigurationConfigItemsFromJSON(rawJSON, jsonFields["configItems"])
	if err != nil {
		return opsiConfigurationDesiredState{}, err
	}
	return opsiConfigurationDesiredState{
		FromJSON:          rawJSON != "",
		JSONFields:        jsonFields,
		CompartmentID:     stringValue(details.CompartmentId),
		DisplayName:       stringValue(details.DisplayName),
		Description:       stringValue(details.Description),
		FreeformTags:      cloneOpsiConfigurationStringMap(details.FreeformTags),
		DefinedTags:       cloneOpsiConfigurationNestedMap(details.DefinedTags),
		SystemTags:        cloneOpsiConfigurationNestedMap(details.SystemTags),
		OpsiConfigType:    configType,
		CreateConfigItems: cloneOpsiConfigurationCreateConfigItems(details.ConfigItems),
		UpdateConfigItems: updateConfigItems,
	}, nil
}

func updateOpsiConfigurationConfigItemsFromJSON(raw string, explicit bool) ([]opsisdk.UpdateConfigurationItemDetails, error) {
	if !explicit {
		return nil, nil
	}
	var details opsisdk.UpdateOpsiUxConfigurationDetails
	if err := json.Unmarshal([]byte(raw), &details); err != nil {
		return nil, fmt.Errorf("decode %s update configItems from jsonData: %w", opsiConfigurationKind, err)
	}
	return cloneOpsiConfigurationUpdateConfigItems(details.ConfigItems), nil
}

func validateOpsiConfigurationCreateOnlyDrift(resource *opsiv1beta1.OpsiConfiguration, currentResponse any) error {
	if resource == nil {
		return nil
	}
	current, ok := opsiConfigurationViewFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current %s response does not expose an OpsiConfiguration body", opsiConfigurationKind)
	}
	desired, err := opsiConfigurationDesiredStateForResource(resource)
	if err != nil {
		return err
	}

	var drift []string
	drift = appendStringDrift(drift, desired.fieldPath("compartmentId"), desired.CompartmentID, current.CompartmentID)
	drift = appendStringDrift(drift, desired.fieldPath("opsiConfigType"), desired.OpsiConfigType, current.OpsiConfigType)
	if len(drift) == 0 {
		return nil
	}
	return fmt.Errorf("%s create-only field drift detected for %s; recreate the resource instead of updating immutable fields", opsiConfigurationKind, strings.Join(drift, ", "))
}

func (d opsiConfigurationDesiredState) fieldPath(field string) string {
	if d.FromJSON && d.JSONFields[field] {
		return "jsonData." + field
	}
	return field
}

func opsiConfigurationStatusFromResponse(resource *opsiv1beta1.OpsiConfiguration, response any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", opsiConfigurationKind)
	}
	current, ok := opsiConfigurationViewFromResponse(response)
	if !ok {
		return nil
	}
	jsonData, err := opsiConfigurationStatusJSONData(current)
	if err != nil {
		return err
	}

	status := resource.Status.OsokStatus
	resource.Status = opsiv1beta1.OpsiConfigurationStatus{
		OsokStatus:       status,
		JsonData:         jsonData,
		Id:               current.ID,
		CompartmentId:    current.CompartmentID,
		DisplayName:      current.DisplayName,
		Description:      current.Description,
		FreeformTags:     cloneOpsiConfigurationStringMap(current.FreeformTags),
		DefinedTags:      opsiConfigurationStatusDefinedTags(current.DefinedTags),
		SystemTags:       opsiConfigurationStatusDefinedTags(current.SystemTags),
		LifecycleState:   string(current.LifecycleState),
		LifecycleDetails: current.LifecycleDetails,
		OpsiConfigType:   current.OpsiConfigType,
	}
	if current.TimeCreated != nil {
		resource.Status.TimeCreated = current.TimeCreated.String()
	}
	if current.TimeUpdated != nil {
		resource.Status.TimeUpdated = current.TimeUpdated.String()
	}
	if current.ID != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(current.ID)
	}
	return nil
}

func opsiConfigurationStatusJSONData(current opsiConfigurationView) (string, error) {
	if strings.TrimSpace(current.OpsiConfigType) == "" && isNil(current.ConfigItems) {
		return "", nil
	}

	payload := struct {
		OpsiConfigType string `json:"opsiConfigType,omitempty"`
		ConfigItems    any    `json:"configItems,omitempty"`
	}{
		OpsiConfigType: strings.TrimSpace(current.OpsiConfigType),
	}
	if !isNil(current.ConfigItems) {
		payload.ConfigItems = current.ConfigItems
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("project %s status jsonData: %w", opsiConfigurationKind, err)
	}
	return string(encoded), nil
}

func opsiConfigurationViewFromResponse(response any) (opsiConfigurationView, bool) {
	if current, ok := opsiConfigurationViewFromDirectResponse(response); ok {
		return current, true
	}
	return opsiConfigurationViewFromPointerResponse(response)
}

func opsiConfigurationViewFromDirectResponse(response any) (opsiConfigurationView, bool) {
	switch current := response.(type) {
	case opsisdk.CreateOpsiConfigurationResponse:
		return opsiConfigurationViewFromConfiguration(current.OpsiConfiguration), current.OpsiConfiguration != nil
	case opsisdk.GetOpsiConfigurationResponse:
		return opsiConfigurationViewFromConfiguration(current.OpsiConfiguration), current.OpsiConfiguration != nil
	case opsisdk.OpsiConfiguration:
		return opsiConfigurationViewFromConfiguration(current), true
	case opsisdk.OpsiConfigurationSummary:
		return opsiConfigurationViewFromSummary(current), true
	default:
		return opsiConfigurationView{}, false
	}
}

func opsiConfigurationViewFromPointerResponse(response any) (opsiConfigurationView, bool) {
	switch current := response.(type) {
	case *opsisdk.CreateOpsiConfigurationResponse:
		if current == nil || current.OpsiConfiguration == nil {
			return opsiConfigurationView{}, false
		}
		return opsiConfigurationViewFromConfiguration(current.OpsiConfiguration), true
	case *opsisdk.GetOpsiConfigurationResponse:
		if current == nil || current.OpsiConfiguration == nil {
			return opsiConfigurationView{}, false
		}
		return opsiConfigurationViewFromConfiguration(current.OpsiConfiguration), true
	default:
		return opsiConfigurationView{}, false
	}
}

func opsiConfigurationViewFromConfiguration(current opsisdk.OpsiConfiguration) opsiConfigurationView {
	return opsiConfigurationView{
		ID:               stringValue(current.GetId()),
		CompartmentID:    stringValue(current.GetCompartmentId()),
		DisplayName:      stringValue(current.GetDisplayName()),
		Description:      stringValue(current.GetDescription()),
		FreeformTags:     cloneOpsiConfigurationStringMap(current.GetFreeformTags()),
		DefinedTags:      cloneOpsiConfigurationNestedMap(current.GetDefinedTags()),
		SystemTags:       cloneOpsiConfigurationNestedMap(current.GetSystemTags()),
		TimeCreated:      current.GetTimeCreated(),
		TimeUpdated:      current.GetTimeUpdated(),
		LifecycleState:   current.GetLifecycleState(),
		LifecycleDetails: stringValue(current.GetLifecycleDetails()),
		OpsiConfigType:   opsiConfigurationTypeFromSDK(current),
		ConfigItems:      current.GetConfigItems(),
	}
}

func opsiConfigurationViewFromSummary(current opsisdk.OpsiConfigurationSummary) opsiConfigurationView {
	return opsiConfigurationView{
		ID:               stringValue(current.GetId()),
		CompartmentID:    stringValue(current.GetCompartmentId()),
		DisplayName:      stringValue(current.GetDisplayName()),
		Description:      stringValue(current.GetDescription()),
		FreeformTags:     cloneOpsiConfigurationStringMap(current.GetFreeformTags()),
		DefinedTags:      cloneOpsiConfigurationNestedMap(current.GetDefinedTags()),
		SystemTags:       cloneOpsiConfigurationNestedMap(current.GetSystemTags()),
		TimeCreated:      current.GetTimeCreated(),
		TimeUpdated:      current.GetTimeUpdated(),
		LifecycleState:   current.GetLifecycleState(),
		LifecycleDetails: stringValue(current.GetLifecycleDetails()),
		OpsiConfigType:   opsiConfigurationTypeFromSDK(current),
	}
}

func opsiConfigurationTypeFromSDK(value any) string {
	switch value.(type) {
	case opsisdk.OpsiUxConfiguration, *opsisdk.OpsiUxConfiguration,
		opsisdk.OpsiUxConfigurationSummary, *opsisdk.OpsiUxConfigurationSummary,
		opsisdk.CreateOpsiUxConfigurationDetails, *opsisdk.CreateOpsiUxConfigurationDetails,
		opsisdk.UpdateOpsiUxConfigurationDetails, *opsisdk.UpdateOpsiUxConfigurationDetails:
		return opsiConfigurationTypeUX
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	configType, err := opsiConfigurationTypeFromJSON(string(payload))
	if err != nil {
		return ""
	}
	return configType
}

func getOpsiConfigurationWorkRequest(
	ctx context.Context,
	client opsiConfigurationOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s OCI client: %w", opsiConfigurationKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s OCI client is not configured", opsiConfigurationKind)
	}
	response, err := client.GetWorkRequest(ctx, opsisdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveOpsiConfigurationWorkRequestAction(workRequest any) (string, error) {
	current, err := opsiConfigurationWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	var action string
	for _, resource := range current.Resources {
		if !isOpsiConfigurationWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		switch opsisdk.ActionTypeEnum(candidate) {
		case "", opsisdk.ActionTypeInProgress, opsisdk.ActionTypeRelated, opsisdk.ActionTypeFailed:
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf("%s work request %s exposes conflicting action types %q and %q", opsiConfigurationKind, stringValue(current.Id), action, candidate)
		}
	}
	return action, nil
}

func resolveOpsiConfigurationWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := opsiConfigurationWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case opsisdk.OperationTypeCreateOpsiConfiguration:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case opsisdk.OperationTypeUpdateOpsiConfiguration:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case opsisdk.OperationTypeDeleteOpsiConfiguration:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func recoverOpsiConfigurationIDFromWorkRequest(
	_ *opsiv1beta1.OpsiConfiguration,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := opsiConfigurationWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	expectedAction := opsiConfigurationWorkRequestActionForPhase(phase)
	for _, resource := range current.Resources {
		if !isOpsiConfigurationWorkRequestResource(resource) {
			continue
		}
		if expectedAction != "" && resource.ActionType != expectedAction && resource.ActionType != opsisdk.ActionTypeInProgress {
			continue
		}
		if id := strings.TrimSpace(stringValue(resource.Identifier)); id != "" {
			return id, nil
		}
	}
	return "", nil
}

func opsiConfigurationWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) opsisdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return opsisdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return opsisdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return opsisdk.ActionTypeDeleted
	default:
		return ""
	}
}

func opsiConfigurationWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := opsiConfigurationWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", opsiConfigurationKind, phase, stringValue(current.Id), current.Status)
}

func opsiConfigurationWorkRequestFromAny(workRequest any) (opsisdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case opsisdk.WorkRequest:
		return current, nil
	case *opsisdk.WorkRequest:
		if current == nil {
			return opsisdk.WorkRequest{}, fmt.Errorf("%s work request is nil", opsiConfigurationKind)
		}
		return *current, nil
	default:
		return opsisdk.WorkRequest{}, fmt.Errorf("%s work request has unexpected type %T", opsiConfigurationKind, workRequest)
	}
}

func isOpsiConfigurationWorkRequestResource(resource opsisdk.WorkRequestResource) bool {
	entityType := normalizeOpsiConfigurationToken(stringValue(resource.EntityType))
	if entityType == opsiConfigurationEntityType || strings.Contains(entityType, opsiConfigurationEntityType) {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/opsiconfigurations/")
}

func normalizeOpsiConfigurationToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func listOpsiConfigurationPages(call opsiConfigurationListCall) opsiConfigurationListCall {
	return func(ctx context.Context, request opsisdk.ListOpsiConfigurationsRequest) (opsisdk.ListOpsiConfigurationsResponse, error) {
		seenPages := map[string]struct{}{}
		var combined opsisdk.ListOpsiConfigurationsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return opsisdk.ListOpsiConfigurationsResponse{}, err
			}
			if combined.OpcRequestId == nil {
				combined.OpcRequestId = response.OpcRequestId
			}
			combined.RawResponse = response.RawResponse
			combined.Items = append(combined.Items, response.Items...)

			nextPage := stringValue(response.OpcNextPage)
			if nextPage == "" {
				return combined, nil
			}
			if _, ok := seenPages[nextPage]; ok {
				return opsisdk.ListOpsiConfigurationsResponse{}, fmt.Errorf("%s list pagination repeated page token %q", opsiConfigurationKind, nextPage)
			}
			seenPages[nextPage] = struct{}{}
			request.Page = response.OpcNextPage
			combined.OpcNextPage = response.OpcNextPage
		}
	}
}

func handleOpsiConfigurationDeleteError(resource *opsiv1beta1.OpsiConfiguration, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return opsiConfigurationAuthShapedNotFoundError{
		message:      fmt.Sprintf("%s delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed: %v", opsiConfigurationKind, err),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func markOpsiConfigurationDeleted(resource *opsiv1beta1.OpsiConfiguration, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	resource.Status.OsokStatus.DeletedAt = &now
	resource.Status.OsokStatus.UpdatedAt = &now
	if strings.TrimSpace(message) != "" {
		resource.Status.OsokStatus.Message = strings.TrimSpace(message)
	}
	resource.Status.OsokStatus.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(
		resource.Status.OsokStatus,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		resource.Status.OsokStatus.Message,
		loggerutil.OSOKLogger{},
	)
}

func clearTrackedOpsiConfigurationIdentity(resource *opsiv1beta1.OpsiConfiguration) {
	if resource == nil {
		return
	}
	resource.Status = opsiv1beta1.OpsiConfigurationStatus{}
}

func withOpsiConfigurationCreateResponseFields(request opsisdk.CreateOpsiConfigurationRequest) opsisdk.CreateOpsiConfigurationRequest {
	if len(request.OpsiConfigField) == 0 {
		request.OpsiConfigField = []opsisdk.CreateOpsiConfigurationOpsiConfigFieldEnum{
			opsisdk.CreateOpsiConfigurationOpsiConfigFieldConfigitems,
		}
	}
	if len(request.ConfigItemCustomStatus) == 0 {
		request.ConfigItemCustomStatus = []opsisdk.CreateOpsiConfigurationConfigItemCustomStatusEnum{
			opsisdk.CreateOpsiConfigurationConfigItemCustomStatusCustomized,
			opsisdk.CreateOpsiConfigurationConfigItemCustomStatusNoncustomized,
		}
	}
	if len(request.ConfigItemField) == 0 {
		request.ConfigItemField = []opsisdk.CreateOpsiConfigurationConfigItemFieldEnum{
			opsisdk.CreateOpsiConfigurationConfigItemFieldName,
			opsisdk.CreateOpsiConfigurationConfigItemFieldValue,
			opsisdk.CreateOpsiConfigurationConfigItemFieldDefaultvalue,
			opsisdk.CreateOpsiConfigurationConfigItemFieldMetadata,
			opsisdk.CreateOpsiConfigurationConfigItemFieldApplicablecontexts,
		}
	}
	return request
}

func withOpsiConfigurationGetResponseFields(request opsisdk.GetOpsiConfigurationRequest) opsisdk.GetOpsiConfigurationRequest {
	if len(request.OpsiConfigField) == 0 {
		request.OpsiConfigField = []opsisdk.GetOpsiConfigurationOpsiConfigFieldEnum{
			opsisdk.GetOpsiConfigurationOpsiConfigFieldConfigitems,
		}
	}
	if len(request.ConfigItemCustomStatus) == 0 {
		request.ConfigItemCustomStatus = []opsisdk.GetOpsiConfigurationConfigItemCustomStatusEnum{
			opsisdk.GetOpsiConfigurationConfigItemCustomStatusCustomized,
			opsisdk.GetOpsiConfigurationConfigItemCustomStatusNoncustomized,
		}
	}
	if len(request.ConfigItemField) == 0 {
		request.ConfigItemField = []opsisdk.GetOpsiConfigurationConfigItemFieldEnum{
			opsisdk.GetOpsiConfigurationConfigItemFieldName,
			opsisdk.GetOpsiConfigurationConfigItemFieldValue,
			opsisdk.GetOpsiConfigurationConfigItemFieldDefaultvalue,
			opsisdk.GetOpsiConfigurationConfigItemFieldMetadata,
			opsisdk.GetOpsiConfigurationConfigItemFieldApplicablecontexts,
		}
	}
	return request
}

func rememberOpsiConfigurationCreateBody(resource *opsiv1beta1.OpsiConfiguration, details opsisdk.CreateOpsiConfigurationDetails) {
	key := opsiConfigurationResourceRetryToken(resource)
	if key == "" || details == nil {
		return
	}
	pendingOpsiConfigurationCreateBodies.Store(key, details)
}

func rememberOpsiConfigurationUpdateBody(id string, details opsisdk.UpdateOpsiConfigurationDetails) {
	id = strings.TrimSpace(id)
	if id == "" || details == nil {
		return
	}
	pendingOpsiConfigurationUpdateBodies.Store(id, details)
}

func withOpsiConfigurationCreateRequestBody(request opsisdk.CreateOpsiConfigurationRequest) (opsisdk.CreateOpsiConfigurationRequest, error) {
	if request.CreateOpsiConfigurationDetails != nil {
		return request, nil
	}

	key := stringValue(request.OpcRetryToken)
	if key == "" {
		return request, fmt.Errorf("%s create request is missing opc retry token for resource-local body staging", opsiConfigurationKind)
	}
	body, ok := pendingOpsiConfigurationCreateBodies.LoadAndDelete(key)
	if !ok {
		return request, fmt.Errorf("%s create request body was not staged for retry token %q", opsiConfigurationKind, key)
	}
	details, ok := body.(opsisdk.CreateOpsiConfigurationDetails)
	if !ok {
		return request, fmt.Errorf("%s staged create request body has unexpected type %T", opsiConfigurationKind, body)
	}
	request.CreateOpsiConfigurationDetails = details
	return request, nil
}

func withOpsiConfigurationUpdateRequestBody(request opsisdk.UpdateOpsiConfigurationRequest) (opsisdk.UpdateOpsiConfigurationRequest, error) {
	if request.UpdateOpsiConfigurationDetails != nil {
		return request, nil
	}

	id := stringValue(request.OpsiConfigurationId)
	if id == "" {
		return request, fmt.Errorf("%s update request is missing resource ID for resource-local body staging", opsiConfigurationKind)
	}
	body, ok := pendingOpsiConfigurationUpdateBodies.LoadAndDelete(id)
	if !ok {
		return request, fmt.Errorf("%s update request body was not staged for resource %q", opsiConfigurationKind, id)
	}
	details, ok := body.(opsisdk.UpdateOpsiConfigurationDetails)
	if !ok {
		return request, fmt.Errorf("%s staged update request body has unexpected type %T", opsiConfigurationKind, body)
	}
	request.UpdateOpsiConfigurationDetails = details
	return request, nil
}

func opsiConfigurationResourceRetryToken(resource *opsiv1beta1.OpsiConfiguration) string {
	if resource == nil {
		return ""
	}
	if uid := strings.TrimSpace(string(resource.UID)); uid != "" {
		return uid
	}

	namespace := strings.TrimSpace(resource.Namespace)
	name := strings.TrimSpace(resource.Name)
	if namespace == "" && name == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(namespace + "/" + name))
	return fmt.Sprintf("%x", sum[:16])
}

func opsiConfigurationTypeFromJSON(raw string) (string, error) {
	var discriminator struct {
		OpsiConfigType string `json:"opsiConfigType"`
	}
	if err := json.Unmarshal([]byte(raw), &discriminator); err != nil {
		return "", fmt.Errorf("decode %s jsonData discriminator: %w", opsiConfigurationKind, err)
	}
	return normalizeOpsiConfigurationType(discriminator.OpsiConfigType)
}

func normalizeOpsiConfigurationType(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return opsiConfigurationTypeUX, nil
	}
	configType, ok := opsisdk.GetMappingOpsiConfigurationTypeEnum(raw)
	if !ok {
		return "", fmt.Errorf("unsupported %s opsiConfigType %q", opsiConfigurationKind, raw)
	}
	return string(configType), nil
}

func unsupportedOpsiConfigurationTypeError(configType string) error {
	return fmt.Errorf("unsupported %s opsiConfigType %q; SDK create/update details are available only for %s", opsiConfigurationKind, configType, opsiConfigurationTypeUX)
}

func opsiConfigurationJSONFields(raw string) (map[string]bool, error) {
	fields := map[string]bool{}
	if strings.TrimSpace(raw) == "" {
		return fields, nil
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("decode %s jsonData: %w", opsiConfigurationKind, err)
	}
	for field := range values {
		fields[field] = true
	}
	return fields, nil
}

func normalizeOpsiConfigurationConfigItems(items any) any {
	if isNil(items) {
		return nil
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return items
	}
	var decoded []map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return items
	}
	normalized := make([]map[string]any, 0, len(decoded))
	for _, item := range decoded {
		next := map[string]any{}
		for _, field := range []string{"configItemType", "name", "value"} {
			if value, ok := item[field]; ok && !isNil(value) {
				next[field] = value
			}
		}
		if len(next) != 0 {
			normalized = append(normalized, next)
		}
	}
	sort.Slice(normalized, func(i, j int) bool {
		return configItemSortKey(normalized[i]) < configItemSortKey(normalized[j])
	})
	return normalized
}

func configItemSortKey(item map[string]any) string {
	return fmt.Sprintf("%v/%v/%v", item["configItemType"], item["name"], item["value"])
}

func opsiConfigurationDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		child := make(map[string]interface{}, len(values))
		for key, value := range values {
			child[key] = value
		}
		converted[namespace] = child
	}
	return converted
}

func opsiConfigurationStatusDefinedTags(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if tags == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(tags))
	for namespace, values := range tags {
		child := make(shared.MapValue, len(values))
		for key, value := range values {
			child[key] = fmt.Sprint(value)
		}
		converted[namespace] = child
	}
	return converted
}

func cloneOpsiConfigurationStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneOpsiConfigurationNestedMap(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		child := make(map[string]interface{}, len(values))
		for key, value := range values {
			child[key] = value
		}
		cloned[namespace] = child
	}
	return cloned
}

func cloneOpsiConfigurationCreateConfigItems(source []opsisdk.CreateConfigurationItemDetails) []opsisdk.CreateConfigurationItemDetails {
	if source == nil {
		return nil
	}
	cloned := make([]opsisdk.CreateConfigurationItemDetails, len(source))
	copy(cloned, source)
	return cloned
}

func cloneOpsiConfigurationUpdateConfigItems(source []opsisdk.UpdateConfigurationItemDetails) []opsisdk.UpdateConfigurationItemDetails {
	if source == nil {
		return nil
	}
	cloned := make([]opsisdk.UpdateConfigurationItemDetails, len(source))
	copy(cloned, source)
	return cloned
}

func appendPointerStringConflict(conflicts []string, field string, spec string, current *string) []string {
	if current != nil && strings.TrimSpace(spec) != "" && stringValue(current) != strings.TrimSpace(spec) {
		return append(conflicts, field)
	}
	return conflicts
}

func appendMapConflict(conflicts []string, field string, spec map[string]string, current map[string]string) []string {
	if current != nil && spec != nil && !reflect.DeepEqual(current, spec) {
		return append(conflicts, field)
	}
	return conflicts
}

func appendJSONConflict(conflicts []string, field string, spec any, current any) []string {
	if !isNil(current) && !isNil(spec) && !opsiConfigurationJSONEqual(current, spec) {
		return append(conflicts, field)
	}
	return conflicts
}

func appendStringDrift(drift []string, field string, desired string, current string) []string {
	if strings.TrimSpace(desired) != "" && strings.TrimSpace(current) != "" && strings.TrimSpace(desired) != strings.TrimSpace(current) {
		return append(drift, field)
	}
	return drift
}

func opsiConfigurationJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}
	var leftDecoded any
	var rightDecoded any
	if err := json.Unmarshal(leftPayload, &leftDecoded); err != nil {
		return reflect.DeepEqual(left, right)
	}
	if err := json.Unmarshal(rightPayload, &rightDecoded); err != nil {
		return reflect.DeepEqual(left, right)
	}
	return reflect.DeepEqual(leftDecoded, rightDecoded)
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

func firstString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func newOpsiConfigurationRuntimeHooksWithOCIClient(client opsiConfigurationOCIClient) OpsiConfigurationRuntimeHooks {
	call := func(name string) error {
		return fmt.Errorf("%s OCI client is not configured for %s", opsiConfigurationKind, name)
	}
	return OpsiConfigurationRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*opsiv1beta1.OpsiConfiguration]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*opsiv1beta1.OpsiConfiguration]{},
		StatusHooks:     generatedruntime.StatusHooks[*opsiv1beta1.OpsiConfiguration]{},
		ParityHooks:     generatedruntime.ParityHooks[*opsiv1beta1.OpsiConfiguration]{},
		Async:           generatedruntime.AsyncHooks[*opsiv1beta1.OpsiConfiguration]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*opsiv1beta1.OpsiConfiguration]{},
		Create: runtimeOperationHooks[opsisdk.CreateOpsiConfigurationRequest, opsisdk.CreateOpsiConfigurationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateOpsiConfigurationDetails", RequestName: "CreateOpsiConfigurationDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request opsisdk.CreateOpsiConfigurationRequest) (opsisdk.CreateOpsiConfigurationResponse, error) {
				if client == nil {
					return opsisdk.CreateOpsiConfigurationResponse{}, call("CreateOpsiConfiguration")
				}
				return client.CreateOpsiConfiguration(ctx, request)
			},
		},
		Get: runtimeOperationHooks[opsisdk.GetOpsiConfigurationRequest, opsisdk.GetOpsiConfigurationResponse]{
			Fields: opsiConfigurationGetFields(),
			Call: func(ctx context.Context, request opsisdk.GetOpsiConfigurationRequest) (opsisdk.GetOpsiConfigurationResponse, error) {
				if client == nil {
					return opsisdk.GetOpsiConfigurationResponse{}, call("GetOpsiConfiguration")
				}
				return client.GetOpsiConfiguration(ctx, request)
			},
		},
		List: runtimeOperationHooks[opsisdk.ListOpsiConfigurationsRequest, opsisdk.ListOpsiConfigurationsResponse]{
			Fields: opsiConfigurationListFields(),
			Call: func(ctx context.Context, request opsisdk.ListOpsiConfigurationsRequest) (opsisdk.ListOpsiConfigurationsResponse, error) {
				if client == nil {
					return opsisdk.ListOpsiConfigurationsResponse{}, call("ListOpsiConfigurations")
				}
				return client.ListOpsiConfigurations(ctx, request)
			},
		},
		Update: runtimeOperationHooks[opsisdk.UpdateOpsiConfigurationRequest, opsisdk.UpdateOpsiConfigurationResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "OpsiConfigurationId", RequestName: "opsiConfigurationId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateOpsiConfigurationDetails", RequestName: "UpdateOpsiConfigurationDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request opsisdk.UpdateOpsiConfigurationRequest) (opsisdk.UpdateOpsiConfigurationResponse, error) {
				if client == nil {
					return opsisdk.UpdateOpsiConfigurationResponse{}, call("UpdateOpsiConfiguration")
				}
				return client.UpdateOpsiConfiguration(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[opsisdk.DeleteOpsiConfigurationRequest, opsisdk.DeleteOpsiConfigurationResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OpsiConfigurationId", RequestName: "opsiConfigurationId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.DeleteOpsiConfigurationRequest) (opsisdk.DeleteOpsiConfigurationResponse, error) {
				if client == nil {
					return opsisdk.DeleteOpsiConfigurationResponse{}, call("DeleteOpsiConfiguration")
				}
				return client.DeleteOpsiConfiguration(ctx, request)
			},
		},
		WrapGeneratedClient: []func(OpsiConfigurationServiceClient) OpsiConfigurationServiceClient{},
	}
}
