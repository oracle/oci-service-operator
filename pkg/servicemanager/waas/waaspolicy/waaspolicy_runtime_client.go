/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package waaspolicy

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	waassdk "github.com/oracle/oci-go-sdk/v65/waas"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const waasPolicyDeletePendingMessage = "OCI WaasPolicy delete is in progress"

var waasPolicyWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(waassdk.WorkRequestStatusValuesAccepted),
		string(waassdk.WorkRequestStatusValuesInProgress),
		string(waassdk.WorkRequestStatusValuesCanceling),
	},
	SucceededStatusTokens: []string{string(waassdk.WorkRequestStatusValuesSucceeded)},
	FailedStatusTokens:    []string{string(waassdk.WorkRequestStatusValuesFailed)},
	CanceledStatusTokens:  []string{string(waassdk.WorkRequestStatusValuesCanceled)},
	CreateActionTokens:    []string{string(waassdk.WorkRequestOperationTypeCreateWaasPolicy)},
	UpdateActionTokens:    []string{string(waassdk.WorkRequestOperationTypeUpdateWaasPolicy)},
	DeleteActionTokens:    []string{string(waassdk.WorkRequestOperationTypeDeleteWaasPolicy)},
}

type waasPolicyScalarPayloadFunc func(reflect.Value, bool, bool) (any, bool)

var waasPolicyScalarPayloads = map[reflect.Kind]waasPolicyScalarPayloadFunc{
	reflect.String:  waasPolicyStringScalarPayload,
	reflect.Bool:    waasPolicyBoolScalarPayload,
	reflect.Int:     waasPolicySignedIntegerScalarPayload,
	reflect.Int8:    waasPolicySignedIntegerScalarPayload,
	reflect.Int16:   waasPolicySignedIntegerScalarPayload,
	reflect.Int32:   waasPolicySignedIntegerScalarPayload,
	reflect.Int64:   waasPolicySignedIntegerScalarPayload,
	reflect.Uint:    waasPolicyUnsignedIntegerScalarPayload,
	reflect.Uint8:   waasPolicyUnsignedIntegerScalarPayload,
	reflect.Uint16:  waasPolicyUnsignedIntegerScalarPayload,
	reflect.Uint32:  waasPolicyUnsignedIntegerScalarPayload,
	reflect.Uint64:  waasPolicyUnsignedIntegerScalarPayload,
	reflect.Float32: waasPolicyFloatScalarPayload,
	reflect.Float64: waasPolicyFloatScalarPayload,
}

func waasPolicyStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

type waasPolicyOCIClient interface {
	ChangeWaasPolicyCompartment(context.Context, waassdk.ChangeWaasPolicyCompartmentRequest) (waassdk.ChangeWaasPolicyCompartmentResponse, error)
	CreateWaasPolicy(context.Context, waassdk.CreateWaasPolicyRequest) (waassdk.CreateWaasPolicyResponse, error)
	GetWaasPolicy(context.Context, waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error)
	ListWaasPolicies(context.Context, waassdk.ListWaasPoliciesRequest) (waassdk.ListWaasPoliciesResponse, error)
	UpdateWaasPolicy(context.Context, waassdk.UpdateWaasPolicyRequest) (waassdk.UpdateWaasPolicyResponse, error)
	DeleteWaasPolicy(context.Context, waassdk.DeleteWaasPolicyRequest) (waassdk.DeleteWaasPolicyResponse, error)
	GetWorkRequest(context.Context, waassdk.GetWorkRequestRequest) (waassdk.GetWorkRequestResponse, error)
}

type waasPolicyIdentity struct {
	compartmentID string
	domain        string
}

type waasPolicyAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e waasPolicyAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e waasPolicyAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerWaasPolicyRuntimeHooksMutator(func(manager *WaasPolicyServiceManager, hooks *WaasPolicyRuntimeHooks) {
		workRequestClient, initErr := newWaasPolicyWorkRequestClient(manager)
		applyWaasPolicyRuntimeHooks(hooks, workRequestClient, initErr)
	})
}

func applyWaasPolicyRuntimeHooks(
	hooks *WaasPolicyRuntimeHooks,
	workRequestClient waasPolicyOCIClient,
	workRequestClientInitErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = waasPolicyRuntimeSemantics()
	hooks.BuildCreateBody = buildWaasPolicyCreateBody
	hooks.BuildUpdateBody = buildWaasPolicyUpdateBody
	hooks.Identity.Resolve = resolveWaasPolicyIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardWaasPolicyExistingBeforeCreate
	hooks.Identity.RecordTracked = recordTrackedWaasPolicyIdentity
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedWaasPolicyIdentity
	hooks.Async.Adapter = waasPolicyWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getWaasPolicyWorkRequest(ctx, workRequestClient, workRequestClientInitErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveWaasPolicyWorkRequestAction
	hooks.Async.RecoverResourceID = recoverWaasPolicyIDFromWorkRequest
	hooks.ParityHooks.RequiresParityHandling = waasPolicyRequiresCompartmentMove
	hooks.ParityHooks.ApplyParityUpdate = func(
		ctx context.Context,
		resource *waasv1beta1.WaasPolicy,
		currentResponse any,
	) (servicemanager.OSOKResponse, error) {
		return applyWaasPolicyCompartmentMove(ctx, resource, currentResponse, workRequestClient, workRequestClientInitErr)
	}
	hooks.List.Fields = waasPolicyListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listWaasPoliciesAllPages(hooks.List.Call)
	}
	hooks.StatusHooks.MarkTerminating = markWaasPolicyTerminating
	hooks.StatusHooks.MarkDeleted = markWaasPolicyDeleted
	hooks.DeleteHooks.HandleError = handleWaasPolicyDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyWaasPolicyDeleteOutcome
	wrapWaasPolicyDeleteGuard(hooks)
}

func newWaasPolicyServiceClientWithOCIClient(client waasPolicyOCIClient) WaasPolicyServiceClient {
	hooks := newWaasPolicyRuntimeHooksWithOCIClient(client)
	applyWaasPolicyRuntimeHooks(&hooks, client, nil)
	delegate := defaultWaasPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*waasv1beta1.WaasPolicy](
			buildWaasPolicyGeneratedRuntimeConfig(&WaasPolicyServiceManager{}, hooks),
		),
	}
	return wrapWaasPolicyGeneratedClient(hooks, delegate)
}

func newWaasPolicyRuntimeHooksWithOCIClient(client waasPolicyOCIClient) WaasPolicyRuntimeHooks {
	hooks := newWaasPolicyDefaultRuntimeHooks(waassdk.WaasClient{})
	hooks.Create.Call = func(ctx context.Context, request waassdk.CreateWaasPolicyRequest) (waassdk.CreateWaasPolicyResponse, error) {
		if client == nil {
			return waassdk.CreateWaasPolicyResponse{}, fmt.Errorf("WaasPolicy OCI client is nil")
		}
		return client.CreateWaasPolicy(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error) {
		if client == nil {
			return waassdk.GetWaasPolicyResponse{}, fmt.Errorf("WaasPolicy OCI client is nil")
		}
		return client.GetWaasPolicy(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request waassdk.ListWaasPoliciesRequest) (waassdk.ListWaasPoliciesResponse, error) {
		if client == nil {
			return waassdk.ListWaasPoliciesResponse{}, fmt.Errorf("WaasPolicy OCI client is nil")
		}
		return client.ListWaasPolicies(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request waassdk.UpdateWaasPolicyRequest) (waassdk.UpdateWaasPolicyResponse, error) {
		if client == nil {
			return waassdk.UpdateWaasPolicyResponse{}, fmt.Errorf("WaasPolicy OCI client is nil")
		}
		return client.UpdateWaasPolicy(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request waassdk.DeleteWaasPolicyRequest) (waassdk.DeleteWaasPolicyResponse, error) {
		if client == nil {
			return waassdk.DeleteWaasPolicyResponse{}, fmt.Errorf("WaasPolicy OCI client is nil")
		}
		return client.DeleteWaasPolicy(ctx, request)
	}
	return hooks
}

func newWaasPolicyWorkRequestClient(manager *WaasPolicyServiceManager) (waasPolicyOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("WaasPolicy service manager is nil")
	}
	client, err := waassdk.NewWaasClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, fmt.Errorf("initialize WaasPolicy work request OCI client: %w", err)
	}
	return client, nil
}

func getWaasPolicyWorkRequest(
	ctx context.Context,
	client waasPolicyOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, initErr
	}
	if client == nil {
		return nil, fmt.Errorf("WaasPolicy OCI client is nil")
	}
	response, err := client.GetWorkRequest(ctx, waassdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveWaasPolicyWorkRequestAction(workRequest any) (string, error) {
	current, err := waasPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func recoverWaasPolicyIDFromWorkRequest(
	resource *waasv1beta1.WaasPolicy,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	if policyID := trackedWaasPolicyID(resource); policyID != "" {
		return policyID, nil
	}
	current, err := waasPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return waasPolicyIDFromWorkRequestResources(current.Resources, phase), nil
}

func waasPolicyIDFromWorkRequestResources(resources []waassdk.WorkRequestResource, phase shared.OSOKAsyncPhase) string {
	for _, resource := range resources {
		if !isWaasPolicyWorkRequestResource(resource) {
			continue
		}
		if !waasPolicyWorkRequestActionMatchesPhase(resource.ActionType, phase) {
			continue
		}
		if id := strings.TrimSpace(waasPolicyStringValue(resource.Identifier)); id != "" {
			return id
		}
	}
	for _, resource := range resources {
		if !isWaasPolicyWorkRequestResource(resource) {
			continue
		}
		if id := strings.TrimSpace(waasPolicyStringValue(resource.Identifier)); id != "" {
			return id
		}
	}
	return ""
}

func waasPolicyWorkRequestActionMatchesPhase(action waassdk.WorkRequestResourceActionTypeEnum, phase shared.OSOKAsyncPhase) bool {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return action == waassdk.WorkRequestResourceActionTypeCreated || action == waassdk.WorkRequestResourceActionTypeInProgress
	case shared.OSOKAsyncPhaseUpdate:
		return action == waassdk.WorkRequestResourceActionTypeUpdated || action == waassdk.WorkRequestResourceActionTypeInProgress
	case shared.OSOKAsyncPhaseDelete:
		return action == waassdk.WorkRequestResourceActionTypeDeleted || action == waassdk.WorkRequestResourceActionTypeInProgress
	default:
		return true
	}
}

func isWaasPolicyWorkRequestResource(resource waassdk.WorkRequestResource) bool {
	token := normalizeWaasPolicyWorkRequestToken(waasPolicyStringValue(resource.EntityType))
	return token == "waaspolicy" || token == "waaspolicies"
}

func normalizeWaasPolicyWorkRequestToken(value string) string {
	replacer := strings.NewReplacer("_", "", "-", "", " ", "")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(value)))
}

func waasPolicyWorkRequestFromAny(workRequest any) (waassdk.WorkRequest, error) {
	switch typed := workRequest.(type) {
	case waassdk.WorkRequest:
		return typed, nil
	case *waassdk.WorkRequest:
		if typed == nil {
			return waassdk.WorkRequest{}, fmt.Errorf("WaasPolicy work request is nil")
		}
		return *typed, nil
	default:
		return waassdk.WorkRequest{}, fmt.Errorf("expected WaasPolicy work request, got %T", workRequest)
	}
}

func waasPolicyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "waas",
		FormalSlug:        "waaspolicy",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(waassdk.LifecycleStatesCreating)},
			UpdatingStates:     []string{string(waassdk.LifecycleStatesUpdating)},
			ActiveStates:       []string{string(waassdk.LifecycleStatesActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(waassdk.LifecycleStatesDeleting)},
			TerminalStates: []string{string(waassdk.LifecycleStatesDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "domain", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"compartmentId",
				"displayName",
				"additionalDomains",
				"origins",
				"originGroups",
				"policyConfig",
				"wafConfig",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"domain"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource", EntityType: "WaasPolicy", Action: "CreateWaasPolicy"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "WorkRequest", Action: "GetWorkRequest"},
			},
			Update: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource", EntityType: "WaasPolicy", Action: "UpdateWaasPolicy"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "WorkRequest", Action: "GetWorkRequest"},
			},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "WaasPolicy", Action: "DeleteWaasPolicy"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "workrequest",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "workrequest",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "WorkRequest", Action: "GetWorkRequest"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func waasPolicyListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func resolveWaasPolicyIdentity(resource *waasv1beta1.WaasPolicy) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("WaasPolicy resource is nil")
	}
	return waasPolicyIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		domain:        strings.TrimSpace(resource.Spec.Domain),
	}, nil
}

func guardWaasPolicyExistingBeforeCreate(
	_ context.Context,
	resource *waasv1beta1.WaasPolicy,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("WaasPolicy resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.Domain) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildWaasPolicyCreateBody(
	_ context.Context,
	resource *waasv1beta1.WaasPolicy,
	_ string,
) (any, error) {
	if resource == nil {
		return waassdk.CreateWaasPolicyDetails{}, fmt.Errorf("WaasPolicy resource is nil")
	}
	if err := validateWaasPolicyRequiredSpec(resource.Spec); err != nil {
		return waassdk.CreateWaasPolicyDetails{}, err
	}

	payload, err := waasPolicySpecPayload(resource.Spec, false)
	if err != nil {
		return waassdk.CreateWaasPolicyDetails{}, err
	}
	payload["compartmentId"] = strings.TrimSpace(resource.Spec.CompartmentId)
	payload["domain"] = strings.TrimSpace(resource.Spec.Domain)

	var details waassdk.CreateWaasPolicyDetails
	if err := decodeWaasPolicyPayload(payload, &details); err != nil {
		return waassdk.CreateWaasPolicyDetails{}, fmt.Errorf("decode WaasPolicy create details: %w", err)
	}
	return details, nil
}

func buildWaasPolicyUpdateBody(
	_ context.Context,
	resource *waasv1beta1.WaasPolicy,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return waassdk.UpdateWaasPolicyDetails{}, false, fmt.Errorf("WaasPolicy resource is nil")
	}
	if err := validateWaasPolicyRequiredSpec(resource.Spec); err != nil {
		return waassdk.UpdateWaasPolicyDetails{}, false, err
	}

	current, ok := waasPolicyFromResponse(currentResponse)
	if !ok {
		return waassdk.UpdateWaasPolicyDetails{}, false, fmt.Errorf("current WaasPolicy response does not expose a WaasPolicy body")
	}

	payload, err := waasPolicySpecPayload(resource.Spec, true)
	if err != nil {
		return waassdk.UpdateWaasPolicyDetails{}, false, err
	}
	updatePayload, updateNeeded, err := waasPolicyUpdatePayload(payload, current)
	if err != nil {
		return waassdk.UpdateWaasPolicyDetails{}, false, err
	}
	if waasPolicyCompartmentNeedsMove(resource.Spec, current) {
		updateNeeded = true
	}
	if !updateNeeded {
		return waassdk.UpdateWaasPolicyDetails{}, false, nil
	}

	var details waassdk.UpdateWaasPolicyDetails
	if err := decodeWaasPolicyPayload(updatePayload, &details); err != nil {
		return waassdk.UpdateWaasPolicyDetails{}, false, fmt.Errorf("decode WaasPolicy update details: %w", err)
	}
	return details, true, nil
}

func validateWaasPolicyRequiredSpec(spec waasv1beta1.WaasPolicySpec) error {
	var problems []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		problems = append(problems, "compartmentId is required")
	}
	if strings.TrimSpace(spec.Domain) == "" {
		problems = append(problems, "domain is required")
	}
	if len(problems) > 0 {
		return fmt.Errorf("WaasPolicy spec is invalid: %s", strings.Join(problems, "; "))
	}
	return nil
}

func waasPolicySpecPayload(spec waasv1beta1.WaasPolicySpec, includeZeroBool bool) (map[string]any, error) {
	payload, include, err := waasPolicyValuePayload(reflect.ValueOf(spec), includeZeroBool, true, false)
	if err != nil {
		return nil, err
	}
	if !include {
		return map[string]any{}, nil
	}
	values, ok := payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("WaasPolicy spec payload has unexpected type %T", payload)
	}
	if err := normalizeWaasPolicyJSONHelpers(values); err != nil {
		return nil, err
	}
	return values, nil
}

func waasPolicyValuePayload(value reflect.Value, includeZeroBool bool, root bool, inCollection bool) (any, bool, error) {
	value, ok := indirectWaasPolicyValue(value)
	if !ok {
		return nil, false, nil
	}

	switch value.Kind() {
	case reflect.Struct:
		if !root && value.IsZero() && !waasPolicyIncludeZeroStruct(value.Type()) {
			return nil, false, nil
		}
		return waasPolicyStructPayload(value, includeZeroBool, root)
	case reflect.Map:
		return waasPolicyMapPayload(value, includeZeroBool)
	case reflect.Slice, reflect.Array:
		return waasPolicySlicePayload(value, includeZeroBool)
	default:
		return waasPolicyScalarPayload(value, includeZeroBool, inCollection)
	}
}

func indirectWaasPolicyValue(value reflect.Value) (reflect.Value, bool) {
	if !value.IsValid() {
		return reflect.Value{}, false
	}
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}
	return value, true
}

func waasPolicyMapPayload(value reflect.Value, includeZeroBool bool) (any, bool, error) {
	if value.IsNil() {
		return nil, false, nil
	}
	result := make(map[string]any, value.Len())
	iter := value.MapRange()
	for iter.Next() {
		key := fmt.Sprint(iter.Key().Interface())
		child, include, err := waasPolicyValuePayload(iter.Value(), includeZeroBool, false, true)
		if err != nil {
			return nil, false, err
		}
		if include {
			result[key] = child
		}
	}
	return result, true, nil
}

func waasPolicySlicePayload(value reflect.Value, includeZeroBool bool) (any, bool, error) {
	if value.Kind() == reflect.Slice && value.IsNil() {
		return nil, false, nil
	}
	result := make([]any, 0, value.Len())
	for i := 0; i < value.Len(); i++ {
		child, include, err := waasPolicyValuePayload(value.Index(i), includeZeroBool, false, true)
		if err != nil {
			return nil, false, err
		}
		if include {
			result = append(result, child)
		} else {
			result = append(result, nil)
		}
	}
	return result, true, nil
}

func waasPolicyScalarPayload(value reflect.Value, includeZeroBool bool, inCollection bool) (any, bool, error) {
	if payload, ok := waasPolicyScalarPayloads[value.Kind()]; ok {
		scalar, include := payload(value, includeZeroBool, inCollection)
		return scalar, include, nil
	}
	return value.Interface(), inCollection || !value.IsZero(), nil
}

func waasPolicyStringScalarPayload(value reflect.Value, _ bool, inCollection bool) (any, bool) {
	text := value.String()
	return text, inCollection || strings.TrimSpace(text) != ""
}

func waasPolicyBoolScalarPayload(value reflect.Value, includeZeroBool bool, inCollection bool) (any, bool) {
	boolean := value.Bool()
	return boolean, includeZeroBool || inCollection || boolean
}

func waasPolicySignedIntegerScalarPayload(value reflect.Value, _ bool, inCollection bool) (any, bool) {
	integer := value.Int()
	return integer, inCollection || integer != 0
}

func waasPolicyUnsignedIntegerScalarPayload(value reflect.Value, _ bool, inCollection bool) (any, bool) {
	integer := value.Uint()
	return integer, inCollection || integer != 0
}

func waasPolicyFloatScalarPayload(value reflect.Value, _ bool, inCollection bool) (any, bool) {
	float := value.Float()
	return float, inCollection || float != 0
}

func waasPolicyStructPayload(value reflect.Value, includeZeroBool bool, root bool) (any, bool, error) {
	result := map[string]any{}
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := typ.Field(i)
		if !fieldType.IsExported() {
			continue
		}
		jsonName, ok := waasPolicyJSONName(fieldType)
		if !ok {
			continue
		}
		childIncludeZeroBool := includeZeroBool || waasPolicyJSONFieldRequired(fieldType)
		child, include, err := waasPolicyValuePayload(value.Field(i), childIncludeZeroBool, false, false)
		if err != nil {
			return nil, false, fmt.Errorf("%s: %w", jsonName, err)
		}
		if include {
			result[jsonName] = child
		}
	}
	if len(result) == 0 && !root {
		return nil, false, nil
	}
	return result, true, nil
}

func waasPolicyIncludeZeroStruct(typ reflect.Type) bool {
	if typ.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		if waasPolicyJSONFieldRequired(field) {
			return true
		}
	}
	return false
}

func waasPolicyJSONFieldRequired(field reflect.StructField) bool {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return false
	}
	parts := strings.Split(tag, ",")
	for _, option := range parts[1:] {
		if option == "omitempty" {
			return false
		}
	}
	return parts[0] != ""
}

func waasPolicyJSONName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return "", false
	}
	name := strings.Split(tag, ",")[0]
	return name, name != ""
}

func normalizeWaasPolicyJSONHelpers(payload map[string]any) error {
	if err := normalizeWaasPolicyPolicyConfigJSON(payload); err != nil {
		return err
	}
	return normalizeWaasPolicyWafConfigJSON(payload)
}

func normalizeWaasPolicyPolicyConfigJSON(payload map[string]any) error {
	policyConfig, ok := payload["policyConfig"].(map[string]any)
	if !ok {
		return nil
	}
	if err := normalizeWaasPolicyNestedJSONData(policyConfig, "loadBalancingMethod", "policyConfig.loadBalancingMethod"); err != nil {
		return err
	}
	if len(policyConfig) == 0 {
		delete(payload, "policyConfig")
	}
	return nil
}

func normalizeWaasPolicyWafConfigJSON(payload map[string]any) error {
	wafConfig, ok := payload["wafConfig"].(map[string]any)
	if !ok {
		return nil
	}
	accessRules, _ := wafConfig["accessRules"].([]any)
	for ruleIndex, rawRule := range accessRules {
		rule, ok := rawRule.(map[string]any)
		if !ok {
			continue
		}
		actions, _ := rule["responseHeaderManipulation"].([]any)
		for actionIndex, rawAction := range actions {
			action, ok := rawAction.(map[string]any)
			if !ok {
				continue
			}
			path := fmt.Sprintf("wafConfig.accessRules[%d].responseHeaderManipulation[%d]", ruleIndex, actionIndex)
			if err := normalizeWaasPolicyHeaderActionJSON(actions, actionIndex, action, path); err != nil {
				return err
			}
		}
	}
	if len(wafConfig) == 0 {
		delete(payload, "wafConfig")
	}
	return nil
}

func normalizeWaasPolicyHeaderActionJSON(actions []any, index int, action map[string]any, path string) error {
	normalized, err := normalizeWaasPolicySelfJSONData(action, path)
	if err != nil {
		return err
	}
	actions[index] = normalized
	return nil
}

func normalizeWaasPolicyNestedJSONData(parent map[string]any, key string, path string) error {
	child, ok := parent[key].(map[string]any)
	if !ok {
		return nil
	}
	normalized, err := normalizeWaasPolicySelfJSONData(child, path)
	if err != nil {
		return err
	}
	if normalizedMap, ok := normalized.(map[string]any); ok && len(normalizedMap) == 0 {
		delete(parent, key)
		return nil
	}
	parent[key] = normalized
	return nil
}

func normalizeWaasPolicySelfJSONData(values map[string]any, path string) (any, error) {
	raw := strings.TrimSpace(waasPolicyString(values["jsonData"]))
	if raw == "" {
		delete(values, "jsonData")
		return values, nil
	}

	conflicts := waasPolicyJSONDataConflicts(values)
	if len(conflicts) > 0 {
		return nil, fmt.Errorf("%s.jsonData conflicts with typed field(s): %s", path, strings.Join(conflicts, ", "))
	}

	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("decode %s.jsonData: %w", path, err)
	}
	decodedMap, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s.jsonData must be a JSON object", path)
	}
	return decodedMap, nil
}

func waasPolicyJSONDataConflicts(values map[string]any) []string {
	conflicts := make([]string, 0, len(values))
	for key, value := range values {
		if key == "jsonData" || !waasPolicyMeaningfulValue(value) {
			continue
		}
		conflicts = append(conflicts, key)
	}
	sort.Strings(conflicts)
	return conflicts
}

func waasPolicyMeaningfulValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case bool:
		return typed
	case float64:
		return typed != 0
	case map[string]any:
		return waasPolicyMapHasMeaningfulValue(typed)
	case []any:
		return waasPolicySliceHasMeaningfulValue(typed)
	default:
		return true
	}
}

func waasPolicyMapHasMeaningfulValue(values map[string]any) bool {
	for _, child := range values {
		if waasPolicyMeaningfulValue(child) {
			return true
		}
	}
	return false
}

func waasPolicySliceHasMeaningfulValue(values []any) bool {
	for _, child := range values {
		if waasPolicyMeaningfulValue(child) {
			return true
		}
	}
	return false
}

func waasPolicyString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func decodeWaasPolicyPayload(payload map[string]any, target any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal WaasPolicy payload: %w", err)
	}
	if err := json.Unmarshal(encoded, target); err != nil {
		return err
	}
	return nil
}

func waasPolicyUpdatePayload(
	desired map[string]any,
	current waassdk.WaasPolicy,
) (map[string]any, bool, error) {
	currentPayload, err := waasPolicySDKPayload(current)
	if err != nil {
		return nil, false, err
	}

	update := map[string]any{}
	for _, field := range waasPolicyMutableFields() {
		desiredValue, ok := desired[field]
		if !ok {
			continue
		}
		currentValue := currentPayload[field]
		if !waasPolicyFieldNeedsUpdate(field, desiredValue, currentValue) {
			continue
		}
		update[field] = desiredValue
	}
	return update, len(update) > 0, nil
}

func waasPolicyRequiresCompartmentMove(resource *waasv1beta1.WaasPolicy, currentResponse any) bool {
	if resource == nil {
		return false
	}
	current, ok := waasPolicyFromResponse(currentResponse)
	if !ok {
		return false
	}
	return waasPolicyCompartmentNeedsMove(resource.Spec, current)
}

func waasPolicyCompartmentNeedsMove(spec waasv1beta1.WaasPolicySpec, current waassdk.WaasPolicy) bool {
	desired := strings.TrimSpace(spec.CompartmentId)
	observed := strings.TrimSpace(waasPolicyStringValue(current.CompartmentId))
	return desired != "" && observed != "" && desired != observed
}

func applyWaasPolicyCompartmentMove(
	ctx context.Context,
	resource *waasv1beta1.WaasPolicy,
	currentResponse any,
	client waasPolicyOCIClient,
	initErr error,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WaasPolicy resource is nil")
	}
	if initErr != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("initialize WaasPolicy OCI client: %w", initErr)
	}
	if client == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WaasPolicy OCI client is nil")
	}

	current, ok := waasPolicyFromResponse(currentResponse)
	if !ok {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("current WaasPolicy response does not expose a WaasPolicy body")
	}
	resourceID := strings.TrimSpace(waasPolicyStringValue(current.Id))
	if resourceID == "" {
		resourceID = trackedWaasPolicyID(resource)
	}
	if resourceID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WaasPolicy compartment move requires a tracked WaasPolicy id")
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WaasPolicy compartment move requires spec.compartmentId")
	}

	response, err := client.ChangeWaasPolicyCompartment(ctx, waassdk.ChangeWaasPolicyCompartmentRequest{
		WaasPolicyId: common.String(resourceID),
		ChangeWaasPolicyCompartmentDetails: waassdk.ChangeWaasPolicyCompartmentDetails{
			CompartmentId: common.String(compartmentID),
		},
		OpcRetryToken: common.String(waasPolicyCompartmentMoveRetryToken(resource, compartmentID)),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	recordTrackedWaasPolicyIdentity(resource, nil, resourceID)

	message := "OCI WaasPolicy compartment move is in progress"
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceLifecycle,
		Phase:            shared.OSOKAsyncPhaseUpdate,
		RawOperationType: "CHANGE_WAAS_POLICY_COMPARTMENT",
		NormalizedClass:  shared.OSOKAsyncClassPending,
		Message:          message,
	}, loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{
		IsSuccessful:  projection.Condition != shared.Failed,
		ShouldRequeue: projection.ShouldRequeue,
	}, nil
}

func waasPolicyCompartmentMoveRetryToken(resource *waasv1beta1.WaasPolicy, compartmentID string) string {
	if resource == nil {
		return ""
	}
	seed := strings.Join([]string{
		"waaspolicy",
		"compartment-move",
		strings.TrimSpace(string(resource.UID)),
		strings.TrimSpace(resource.Namespace),
		strings.TrimSpace(resource.Name),
		strings.TrimSpace(compartmentID),
	}, "/")
	sum := sha256.Sum256([]byte(seed))
	return fmt.Sprintf("waaspolicy-%x", sum[:16])
}

func waasPolicyMutableFields() []string {
	return []string{
		"displayName",
		"additionalDomains",
		"origins",
		"originGroups",
		"policyConfig",
		"wafConfig",
		"freeformTags",
		"definedTags",
	}
}

func waasPolicyFieldNeedsUpdate(field string, desired any, current any) bool {
	switch field {
	case "policyConfig", "wafConfig":
		return waasPolicySubsetDiffers(desired, current)
	default:
		return !waasPolicyComparableEqual(desired, current)
	}
}

func waasPolicySubsetDiffers(desired any, current any) bool {
	desiredMap, ok := desired.(map[string]any)
	if !ok {
		return !waasPolicyComparableEqual(desired, current)
	}
	currentMap, _ := current.(map[string]any)
	for key, desiredValue := range desiredMap {
		if waasPolicySubsetDiffers(desiredValue, currentMap[key]) {
			return true
		}
	}
	return false
}

func waasPolicyComparableEqual(left any, right any) bool {
	left = waasPolicyNormalizeComparable(left)
	right = waasPolicyNormalizeComparable(right)
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}
	return string(leftPayload) == string(rightPayload)
}

func waasPolicyNormalizeComparable(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case bool:
		if !typed {
			return nil
		}
		return typed
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, child := range typed {
			value := waasPolicyNormalizeComparable(child)
			if value == nil {
				continue
			}
			normalized[key] = value
		}
		return normalized
	case []any:
		normalized := make([]any, 0, len(typed))
		for _, child := range typed {
			normalized = append(normalized, waasPolicyNormalizeComparable(child))
		}
		return normalized
	default:
		return value
	}
}

func waasPolicySDKPayload(value any) (map[string]any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal WaasPolicy SDK value: %w", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("decode WaasPolicy SDK value: %w", err)
	}
	return decoded, nil
}

func waasPolicyFromResponse(response any) (waassdk.WaasPolicy, bool) {
	switch typed := response.(type) {
	case waassdk.GetWaasPolicyResponse:
		return typed.WaasPolicy, true
	case *waassdk.GetWaasPolicyResponse:
		if typed == nil {
			return waassdk.WaasPolicy{}, false
		}
		return typed.WaasPolicy, true
	case waassdk.WaasPolicy:
		return typed, true
	case *waassdk.WaasPolicy:
		if typed == nil {
			return waassdk.WaasPolicy{}, false
		}
		return *typed, true
	case waassdk.WaasPolicySummary:
		return waasPolicyFromSummary(typed), true
	case *waassdk.WaasPolicySummary:
		if typed == nil {
			return waassdk.WaasPolicy{}, false
		}
		return waasPolicyFromSummary(*typed), true
	default:
		return waassdk.WaasPolicy{}, false
	}
}

func waasPolicyFromSummary(summary waassdk.WaasPolicySummary) waassdk.WaasPolicy {
	return waassdk.WaasPolicy{
		Id:             summary.Id,
		CompartmentId:  summary.CompartmentId,
		DisplayName:    summary.DisplayName,
		Domain:         summary.Domain,
		LifecycleState: summary.LifecycleState,
		TimeCreated:    summary.TimeCreated,
		FreeformTags:   summary.FreeformTags,
		DefinedTags:    summary.DefinedTags,
	}
}

func recordTrackedWaasPolicyIdentity(resource *waasv1beta1.WaasPolicy, _ any, resourceID string) {
	if resource == nil || strings.TrimSpace(resourceID) == "" {
		return
	}
	resource.Status.Id = strings.TrimSpace(resourceID)
	resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(resourceID))
}

func clearTrackedWaasPolicyIdentity(resource *waasv1beta1.WaasPolicy) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus.Ocid = ""
}

func listWaasPoliciesAllPages(
	call func(context.Context, waassdk.ListWaasPoliciesRequest) (waassdk.ListWaasPoliciesResponse, error),
) func(context.Context, waassdk.ListWaasPoliciesRequest) (waassdk.ListWaasPoliciesResponse, error) {
	return func(ctx context.Context, request waassdk.ListWaasPoliciesRequest) (waassdk.ListWaasPoliciesResponse, error) {
		if call == nil {
			return waassdk.ListWaasPoliciesResponse{}, fmt.Errorf("WaasPolicy list operation is not configured")
		}

		var accumulated waassdk.ListWaasPoliciesResponse
		seenPages := map[string]struct{}{}
		for {
			response, err := call(ctx, request)
			if err != nil {
				return waassdk.ListWaasPoliciesResponse{}, err
			}
			appendWaasPolicyListPage(&accumulated, response)

			nextPage, err := nextWaasPolicyListPage(response, seenPages)
			if err != nil {
				return waassdk.ListWaasPoliciesResponse{}, err
			}
			if nextPage == nil {
				accumulated.OpcNextPage = nil
				return accumulated, nil
			}
			request.Page = nextPage
		}
	}
}

func appendWaasPolicyListPage(accumulated *waassdk.ListWaasPoliciesResponse, page waassdk.ListWaasPoliciesResponse) {
	if accumulated.RawResponse == nil {
		accumulated.RawResponse = page.RawResponse
	}
	if accumulated.OpcRequestId == nil {
		accumulated.OpcRequestId = page.OpcRequestId
	}
	accumulated.Items = append(accumulated.Items, page.Items...)
}

func nextWaasPolicyListPage(response waassdk.ListWaasPoliciesResponse, seenPages map[string]struct{}) (*string, error) {
	nextPage := strings.TrimSpace(waasPolicyStringValue(response.OpcNextPage))
	if nextPage == "" {
		return nil, nil
	}
	if _, seen := seenPages[nextPage]; seen {
		return nil, fmt.Errorf("WaasPolicy list pagination repeated page token %q", nextPage)
	}
	seenPages[nextPage] = struct{}{}
	return common.String(nextPage), nil
}

func handleWaasPolicyDeleteError(resource *waasv1beta1.WaasPolicy, err error) error {
	if err == nil {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return waasPolicyAmbiguousNotFoundError{
		message:      "WaasPolicy delete returned ambiguous 404 NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func applyWaasPolicyDeleteOutcome(
	resource *waasv1beta1.WaasPolicy,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	switch strings.ToUpper(waasPolicyLifecycleState(response)) {
	case "", string(waassdk.LifecycleStatesDeleting), string(waassdk.LifecycleStatesDeleted):
		return generatedruntime.DeleteOutcome{}, nil
	case string(waassdk.LifecycleStatesActive),
		string(waassdk.LifecycleStatesCreating),
		string(waassdk.LifecycleStatesUpdating),
		string(waassdk.LifecycleStatesFailed):
		if stage == generatedruntime.DeleteConfirmStageAlreadyPending && !waasPolicyDeleteAlreadyPending(resource) {
			return generatedruntime.DeleteOutcome{}, nil
		}
		markWaasPolicyTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	default:
		return generatedruntime.DeleteOutcome{}, nil
	}
}

func waasPolicyLifecycleState(response any) string {
	current, ok := waasPolicyFromResponse(response)
	if !ok {
		return ""
	}
	return string(current.LifecycleState)
}

func waasPolicyDeleteAlreadyPending(resource *waasv1beta1.WaasPolicy) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markWaasPolicyTerminating(resource *waasv1beta1.WaasPolicy, _ any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = waasPolicyDeletePendingMessage
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         waasPolicyDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		waasPolicyDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}

func wrapWaasPolicyDeleteGuard(hooks *WaasPolicyRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getWaasPolicy := hooks.Get.Call
	getWorkRequest := hooks.Async.GetWorkRequest
	deleteWaasPolicy := hooks.Delete.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate WaasPolicyServiceClient) WaasPolicyServiceClient {
		return waasPolicyDeleteGuardClient{
			delegate:         delegate,
			getWaasPolicy:    getWaasPolicy,
			getWorkRequest:   getWorkRequest,
			deleteWaasPolicy: deleteWaasPolicy,
		}
	})
}

type waasPolicyDeleteGuardClient struct {
	delegate         WaasPolicyServiceClient
	getWaasPolicy    func(context.Context, waassdk.GetWaasPolicyRequest) (waassdk.GetWaasPolicyResponse, error)
	getWorkRequest   func(context.Context, string) (any, error)
	deleteWaasPolicy func(context.Context, waassdk.DeleteWaasPolicyRequest) (waassdk.DeleteWaasPolicyResponse, error)
}

func (c waasPolicyDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *waasv1beta1.WaasPolicy,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c waasPolicyDeleteGuardClient) Delete(ctx context.Context, resource *waasv1beta1.WaasPolicy) (bool, error) {
	if handled, deleted, err := c.handlePendingDeleteWorkRequest(ctx, resource); handled || err != nil {
		return deleted, err
	}
	policyID := trackedWaasPolicyID(resource)
	if policyID == "" {
		return c.delegate.Delete(ctx, resource)
	}

	if handled, deleted, err := c.guardPreDeleteRead(ctx, resource, policyID); handled || err != nil {
		return deleted, err
	}
	return c.deleteTrackedWaasPolicy(ctx, resource, policyID)
}

func (c waasPolicyDeleteGuardClient) handlePendingDeleteWorkRequest(
	ctx context.Context,
	resource *waasv1beta1.WaasPolicy,
) (bool, bool, error) {
	workRequestID := waasPolicyPendingDeleteWorkRequestID(resource)
	if workRequestID == "" {
		return false, false, nil
	}
	workRequest, currentAsync, err := c.currentDeleteWorkRequest(ctx, resource, workRequestID)
	if err != nil {
		return true, false, err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		message := fmt.Sprintf("WaasPolicy delete work request %s is still in progress", workRequestID)
		applyWaasPolicyWorkRequestOperationAs(resource, currentAsync, shared.OSOKAsyncClassPending, message)
		return true, false, nil
	case shared.OSOKAsyncClassSucceeded:
		deleted, err := c.confirmSucceededDeleteWorkRequest(ctx, resource, workRequest, currentAsync)
		return true, deleted, err
	default:
		err := fmt.Errorf("WaasPolicy delete work request %s finished with status %s", workRequestID, currentAsync.RawStatus)
		applyWaasPolicyWorkRequestOperation(resource, currentAsync)
		return true, false, err
	}
}

func (c waasPolicyDeleteGuardClient) currentDeleteWorkRequest(
	ctx context.Context,
	resource *waasv1beta1.WaasPolicy,
	workRequestID string,
) (any, *shared.OSOKAsyncOperation, error) {
	if c.getWorkRequest == nil {
		return nil, nil, fmt.Errorf("WaasPolicy work request operation is not configured")
	}
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return nil, nil, err
	}
	currentAsync, err := buildWaasPolicyWorkRequestOperation(&resource.Status.OsokStatus, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return nil, nil, err
	}
	return workRequest, currentAsync, nil
}

func (c waasPolicyDeleteGuardClient) confirmSucceededDeleteWorkRequest(
	ctx context.Context,
	resource *waasv1beta1.WaasPolicy,
	workRequest any,
	currentAsync *shared.OSOKAsyncOperation,
) (bool, error) {
	policyID, err := recoverWaasPolicyIDFromWorkRequest(resource, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return false, err
	}
	if policyID == "" {
		markWaasPolicyDeleted(resource, "OCI WaasPolicy delete work request completed")
		return true, nil
	}

	response, err := c.readWaasPolicyForDeleteConfirmation(ctx, policyID)
	if err != nil {
		return handleWaasPolicyDeleteConfirmationReadError(resource, err)
	}
	if waasPolicyLifecycleState(response) == string(waassdk.LifecycleStatesDeleted) {
		markWaasPolicyDeleted(resource, "OCI WaasPolicy deleted")
		return true, nil
	}
	recordWaasPolicyDeleteReadback(resource, response)
	applyWaasPolicyWorkRequestOperationAs(resource, currentAsync, shared.OSOKAsyncClassPending, waasPolicyDeletePendingMessage)
	return false, nil
}

func (c waasPolicyDeleteGuardClient) guardPreDeleteRead(
	ctx context.Context,
	resource *waasv1beta1.WaasPolicy,
	policyID string,
) (bool, bool, error) {
	if c.getWaasPolicy == nil {
		return false, false, nil
	}
	response, err := c.getWaasPolicy(ctx, waassdk.GetWaasPolicyRequest{WaasPolicyId: common.String(policyID)})
	if err == nil {
		if waasPolicyLifecycleState(response) == string(waassdk.LifecycleStatesDeleted) {
			recordWaasPolicyDeleteReadback(resource, response)
			markWaasPolicyDeleted(resource, "OCI WaasPolicy deleted")
			return true, true, nil
		}
		return false, false, nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	switch {
	case classification.IsUnambiguousNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markWaasPolicyDeleted(resource, "OCI WaasPolicy deleted")
		return true, true, nil
	case classification.IsAuthShapedNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return true, false, fmt.Errorf("WaasPolicy delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to call delete: %v", err)
	default:
		return false, false, nil
	}
}

func (c waasPolicyDeleteGuardClient) deleteTrackedWaasPolicy(
	ctx context.Context,
	resource *waasv1beta1.WaasPolicy,
	policyID string,
) (bool, error) {
	if c.deleteWaasPolicy == nil {
		return false, fmt.Errorf("WaasPolicy delete operation is not configured")
	}
	response, err := c.deleteWaasPolicy(ctx, waassdk.DeleteWaasPolicyRequest{WaasPolicyId: common.String(policyID)})
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			markWaasPolicyDeleted(resource, "OCI WaasPolicy no longer exists")
			return true, nil
		}
		return false, handleWaasPolicyDeleteError(resource, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	workRequestID := strings.TrimSpace(waasPolicyStringValue(response.OpcWorkRequestId))
	if workRequestID != "" {
		seedWaasPolicyDeleteWorkRequest(resource, workRequestID)
		_, currentAsync, err := c.currentDeleteWorkRequest(ctx, resource, workRequestID)
		if err != nil {
			return false, err
		}
		switch currentAsync.NormalizedClass {
		case shared.OSOKAsyncClassPending:
			message := fmt.Sprintf("WaasPolicy delete work request %s is still in progress", workRequestID)
			applyWaasPolicyWorkRequestOperationAs(resource, currentAsync, shared.OSOKAsyncClassPending, message)
			return false, nil
		case shared.OSOKAsyncClassSucceeded:
			return c.confirmSucceededDeleteWorkRequest(ctx, resource, nil, currentAsync)
		default:
			err := fmt.Errorf("WaasPolicy delete work request %s finished with status %s", workRequestID, currentAsync.RawStatus)
			applyWaasPolicyWorkRequestOperation(resource, currentAsync)
			return false, err
		}
	}
	return c.confirmDeleteRequestAccepted(ctx, resource, policyID)
}

func (c waasPolicyDeleteGuardClient) confirmDeleteRequestAccepted(
	ctx context.Context,
	resource *waasv1beta1.WaasPolicy,
	policyID string,
) (bool, error) {
	response, err := c.readWaasPolicyForDeleteConfirmation(ctx, policyID)
	if err != nil {
		return handleWaasPolicyDeleteConfirmationReadError(resource, err)
	}
	if waasPolicyLifecycleState(response) == string(waassdk.LifecycleStatesDeleted) {
		markWaasPolicyDeleted(resource, "OCI WaasPolicy deleted")
		return true, nil
	}
	recordWaasPolicyDeleteReadback(resource, response)
	markWaasPolicyTerminating(resource, response)
	return false, nil
}

func (c waasPolicyDeleteGuardClient) readWaasPolicyForDeleteConfirmation(
	ctx context.Context,
	policyID string,
) (waassdk.GetWaasPolicyResponse, error) {
	if c.getWaasPolicy == nil {
		return waassdk.GetWaasPolicyResponse{}, fmt.Errorf("WaasPolicy readback operation is not configured")
	}
	return c.getWaasPolicy(ctx, waassdk.GetWaasPolicyRequest{WaasPolicyId: common.String(policyID)})
}

func handleWaasPolicyDeleteConfirmationReadError(resource *waasv1beta1.WaasPolicy, err error) (bool, error) {
	classification := errorutil.ClassifyDeleteError(err)
	if classification.IsUnambiguousNotFound() {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markWaasPolicyDeleted(resource, "OCI WaasPolicy deleted")
		return true, nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return false, err
}

func seedWaasPolicyDeleteWorkRequest(resource *waasv1beta1.WaasPolicy, workRequestID string) {
	if resource == nil || strings.TrimSpace(workRequestID) == "" {
		return
	}
	now := metav1.Now()
	resource.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           shared.OSOKAsyncPhaseDelete,
		WorkRequestID:   strings.TrimSpace(workRequestID),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         waasPolicyDeletePendingMessage,
		UpdatedAt:       &now,
	}
}

func buildWaasPolicyWorkRequestOperation(
	status *shared.OSOKStatus,
	workRequest any,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	current, err := waasPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}
	action, err := resolveWaasPolicyWorkRequestAction(current)
	if err != nil {
		return nil, err
	}
	return servicemanager.BuildWorkRequestAsyncOperation(status, waasPolicyWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        action,
		RawOperationType: string(current.OperationType),
		WorkRequestID:    waasPolicyStringValue(current.Id),
		PercentComplete:  waasPolicyWorkRequestPercentComplete(current.PercentComplete),
		FallbackPhase:    fallbackPhase,
	})
}

func waasPolicyWorkRequestPercentComplete(percent *int) *float32 {
	if percent == nil {
		return nil
	}
	value := float32(*percent)
	return &value
}

func applyWaasPolicyWorkRequestOperation(
	resource *waasv1beta1.WaasPolicy,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	if resource == nil || current == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
	now := metav1.Now()
	if current.UpdatedAt == nil {
		current.UpdatedAt = &now
	}
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{
		IsSuccessful:  projection.Condition != shared.Failed,
		ShouldRequeue: projection.ShouldRequeue,
	}
}

func applyWaasPolicyWorkRequestOperationAs(
	resource *waasv1beta1.WaasPolicy,
	current *shared.OSOKAsyncOperation,
	class shared.OSOKAsyncNormalizedClass,
	message string,
) servicemanager.OSOKResponse {
	if current == nil {
		return applyWaasPolicyWorkRequestOperation(resource, current)
	}
	next := *current
	next.NormalizedClass = class
	next.Message = strings.TrimSpace(message)
	next.UpdatedAt = nil
	return applyWaasPolicyWorkRequestOperation(resource, &next)
}

func recordWaasPolicyDeleteReadback(resource *waasv1beta1.WaasPolicy, response waassdk.GetWaasPolicyResponse) {
	if resource == nil {
		return
	}
	current := response.WaasPolicy
	if id := waasPolicyStringValue(current.Id); id != "" {
		resource.Status.Id = id
		resource.Status.OsokStatus.Ocid = shared.OCID(id)
	}
	if compartmentID := waasPolicyStringValue(current.CompartmentId); compartmentID != "" {
		resource.Status.CompartmentId = compartmentID
	}
	if displayName := waasPolicyStringValue(current.DisplayName); displayName != "" {
		resource.Status.DisplayName = displayName
	}
	if domain := waasPolicyStringValue(current.Domain); domain != "" {
		resource.Status.Domain = domain
	}
	resource.Status.LifecycleState = strings.ToUpper(strings.TrimSpace(string(current.LifecycleState)))
}

func markWaasPolicyDeleted(resource *waasv1beta1.WaasPolicy, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	if strings.TrimSpace(message) != "" {
		status.Message = strings.TrimSpace(message)
	}
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		corev1.ConditionTrue,
		"",
		status.Message,
		loggerutil.OSOKLogger{},
	)
}

func waasPolicyPendingDeleteWorkRequestID(resource *waasv1beta1.WaasPolicy) string {
	if !waasPolicyHasPendingDeleteWorkRequest(resource) {
		return ""
	}
	return strings.TrimSpace(resource.Status.OsokStatus.Async.Current.WorkRequestID)
}

func waasPolicyHasPendingDeleteWorkRequest(resource *waasv1beta1.WaasPolicy) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Source == shared.OSOKAsyncSourceWorkRequest &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending &&
		strings.TrimSpace(current.WorkRequestID) != ""
}

func trackedWaasPolicyID(resource *waasv1beta1.WaasPolicy) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}
