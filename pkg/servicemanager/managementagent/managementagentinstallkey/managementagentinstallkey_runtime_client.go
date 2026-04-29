/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managementagentinstallkey

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	managementagentsdk "github.com/oracle/oci-go-sdk/v65/managementagent"
	managementagentv1beta1 "github.com/oracle/oci-service-operator/api/managementagent/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const managementAgentInstallKeyKind = "ManagementAgentInstallKey"

type managementAgentInstallKeyOCIClient interface {
	CreateManagementAgentInstallKey(context.Context, managementagentsdk.CreateManagementAgentInstallKeyRequest) (managementagentsdk.CreateManagementAgentInstallKeyResponse, error)
	GetManagementAgentInstallKey(context.Context, managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error)
	ListManagementAgentInstallKeys(context.Context, managementagentsdk.ListManagementAgentInstallKeysRequest) (managementagentsdk.ListManagementAgentInstallKeysResponse, error)
	UpdateManagementAgentInstallKey(context.Context, managementagentsdk.UpdateManagementAgentInstallKeyRequest) (managementagentsdk.UpdateManagementAgentInstallKeyResponse, error)
	DeleteManagementAgentInstallKey(context.Context, managementagentsdk.DeleteManagementAgentInstallKeyRequest) (managementagentsdk.DeleteManagementAgentInstallKeyResponse, error)
}

func init() {
	registerManagementAgentInstallKeyRuntimeHooksMutator(func(_ *ManagementAgentInstallKeyServiceManager, hooks *ManagementAgentInstallKeyRuntimeHooks) {
		applyManagementAgentInstallKeyRuntimeHooks(hooks)
	})
}

func applyManagementAgentInstallKeyRuntimeHooks(hooks *ManagementAgentInstallKeyRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = managementAgentInstallKeyRuntimeSemantics()
	hooks.BuildCreateBody = buildManagementAgentInstallKeyCreateBody
	hooks.BuildUpdateBody = buildManagementAgentInstallKeyUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardManagementAgentInstallKeyExistingBeforeCreate
	hooks.List.Fields = managementAgentInstallKeyListFields()
	hooks.List.Call = paginatedManagementAgentInstallKeyListCall(hooks.List.Call)
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedManagementAgentInstallKeyIdentity
	hooks.ParityHooks.NormalizeDesiredState = normalizeManagementAgentInstallKeyDesiredState
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateManagementAgentInstallKeyCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleManagementAgentInstallKeyDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ManagementAgentInstallKeyServiceClient) ManagementAgentInstallKeyServiceClient {
		return managementAgentInstallKeyNormalizeClient{delegate: delegate}
	})
	if hooks.Get.Call != nil {
		get := hooks.Get.Call
		hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ManagementAgentInstallKeyServiceClient) ManagementAgentInstallKeyServiceClient {
			return managementAgentInstallKeyDeleteGuardClient{delegate: delegate, get: get}
		})
	}
}

func managementAgentInstallKeyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "managementagent",
		FormalSlug:        "managementagentinstallkey",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(managementagentsdk.LifecycleStatesCreating)},
			UpdatingStates:     []string{string(managementagentsdk.LifecycleStatesUpdating)},
			ActiveStates: []string{
				string(managementagentsdk.LifecycleStatesActive),
				string(managementagentsdk.LifecycleStatesInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(managementagentsdk.LifecycleStatesDeleting),
			},
			TerminalStates: []string{
				string(managementagentsdk.LifecycleStatesDeleted),
				string(managementagentsdk.LifecycleStatesTerminated),
			},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"displayName", "isKeyActive"},
			Mutable:         []string{"displayName", "isKeyActive"},
			ForceNew:        []string{"compartmentId"},
			ConflictsWith:   map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func managementAgentInstallKeyCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateManagementAgentInstallKeyDetails", RequestName: "CreateManagementAgentInstallKeyDetails", Contribution: "body"},
	}
}

func managementAgentInstallKeyGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ManagementAgentInstallKeyId", RequestName: "managementAgentInstallKeyId", Contribution: "path", PreferResourceID: true},
	}
}

func managementAgentInstallKeyListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"spec.compartmentId", "status.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "DisplayName",
			RequestName:  "displayName",
			Contribution: "query",
			LookupPaths:  []string{"spec.displayName", "status.displayName", "displayName"},
		},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func managementAgentInstallKeyUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ManagementAgentInstallKeyId", RequestName: "managementAgentInstallKeyId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateManagementAgentInstallKeyDetails", RequestName: "UpdateManagementAgentInstallKeyDetails", Contribution: "body"},
	}
}

func managementAgentInstallKeyDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ManagementAgentInstallKeyId", RequestName: "managementAgentInstallKeyId", Contribution: "path", PreferResourceID: true},
	}
}

func buildManagementAgentInstallKeyCreateBody(
	_ context.Context,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
	_ string,
) (any, error) {
	normalizeManagementAgentInstallKeyDesiredState(resource, nil)
	if err := validateManagementAgentInstallKeySpec(resource); err != nil {
		return managementagentsdk.CreateManagementAgentInstallKeyDetails{}, err
	}

	spec := resource.Spec
	details := managementagentsdk.CreateManagementAgentInstallKeyDetails{
		DisplayName:   common.String(strings.TrimSpace(spec.DisplayName)),
		CompartmentId: common.String(strings.TrimSpace(spec.CompartmentId)),
	}
	if spec.AllowedKeyInstallCount != 0 {
		details.AllowedKeyInstallCount = common.Int(spec.AllowedKeyInstallCount)
	}
	if strings.TrimSpace(spec.TimeExpires) != "" {
		expiresAt, err := parseManagementAgentInstallKeyTime(spec.TimeExpires)
		if err != nil {
			return managementagentsdk.CreateManagementAgentInstallKeyDetails{}, err
		}
		details.TimeExpires = &expiresAt
	}
	if spec.IsUnlimited {
		details.IsUnlimited = common.Bool(true)
	}
	return details, nil
}

func buildManagementAgentInstallKeyUpdateBody(
	_ context.Context,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
	_ string,
	currentResponse any,
) (any, bool, error) {
	normalizeManagementAgentInstallKeyDesiredState(resource, currentResponse)
	if err := validateManagementAgentInstallKeySpec(resource); err != nil {
		return managementagentsdk.UpdateManagementAgentInstallKeyDetails{}, false, err
	}

	current, ok := managementAgentInstallKeyFromResponse(currentResponse)
	if !ok {
		return managementagentsdk.UpdateManagementAgentInstallKeyDetails{}, false, fmt.Errorf("current %s response does not expose a management agent install key body", managementAgentInstallKeyKind)
	}
	if err := validateManagementAgentInstallKeyCreateOnlyDrift(resource, currentResponse); err != nil {
		return managementagentsdk.UpdateManagementAgentInstallKeyDetails{}, false, err
	}

	var details managementagentsdk.UpdateManagementAgentInstallKeyDetails
	updateNeeded := false
	if desired := strings.TrimSpace(resource.Spec.DisplayName); desired != "" && desired != stringValue(current.DisplayName) {
		details.DisplayName = common.String(desired)
		updateNeeded = true
	}
	if desired, update := managementAgentInstallKeyActiveUpdate(resource.Spec.IsKeyActive, current.LifecycleState); update {
		details.IsKeyActive = common.Bool(desired)
		updateNeeded = true
	}
	return details, updateNeeded, nil
}

func validateManagementAgentInstallKeySpec(resource *managementagentv1beta1.ManagementAgentInstallKey) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", managementAgentInstallKeyKind)
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return fmt.Errorf("%s spec.displayName is required", managementAgentInstallKeyKind)
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return fmt.Errorf("%s spec.compartmentId is required", managementAgentInstallKeyKind)
	}
	if strings.TrimSpace(resource.Spec.TimeExpires) != "" {
		if _, err := parseManagementAgentInstallKeyTime(resource.Spec.TimeExpires); err != nil {
			return err
		}
	}
	return nil
}

func normalizeManagementAgentInstallKeyDesiredState(resource *managementagentv1beta1.ManagementAgentInstallKey, _ any) {
	if resource == nil {
		return
	}
	resource.Spec.DisplayName = strings.TrimSpace(resource.Spec.DisplayName)
	resource.Spec.CompartmentId = strings.TrimSpace(resource.Spec.CompartmentId)
	if !resource.Spec.IsUnlimited {
		return
	}
	resource.Spec.AllowedKeyInstallCount = 0
	resource.Spec.TimeExpires = ""
}

func parseManagementAgentInstallKeyTime(value string) (common.SDKTime, error) {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return common.SDKTime{}, fmt.Errorf("%s spec.timeExpires must be RFC3339: %w", managementAgentInstallKeyKind, err)
	}
	return common.SDKTime{Time: parsed}, nil
}

func guardManagementAgentInstallKeyExistingBeforeCreate(
	_ context.Context,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("%s resource is nil", managementAgentInstallKeyKind)
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func validateManagementAgentInstallKeyCreateOnlyDrift(
	resource *managementagentv1beta1.ManagementAgentInstallKey,
	currentResponse any,
) error {
	if resource == nil || currentResponse == nil {
		return nil
	}
	current, ok := managementAgentInstallKeyFromResponse(currentResponse)
	if !ok {
		return nil
	}

	checks := []func(managementagentv1beta1.ManagementAgentInstallKeySpec, managementagentsdk.ManagementAgentInstallKey) error{
		validateManagementAgentInstallKeyCompartmentDrift,
		validateManagementAgentInstallKeyAllowedCountDrift,
		validateManagementAgentInstallKeyExpiresDrift,
		validateManagementAgentInstallKeyUnlimitedDrift,
	}
	for _, check := range checks {
		if err := check(resource.Spec, current); err != nil {
			return err
		}
	}
	return nil
}

func validateManagementAgentInstallKeyCompartmentDrift(
	spec managementagentv1beta1.ManagementAgentInstallKeySpec,
	current managementagentsdk.ManagementAgentInstallKey,
) error {
	desired := strings.TrimSpace(spec.CompartmentId)
	if desired == "" || desired == stringValue(current.CompartmentId) {
		return nil
	}
	return fmt.Errorf("%s create-only field compartmentId changed from %q to %q", managementAgentInstallKeyKind, stringValue(current.CompartmentId), desired)
}

func validateManagementAgentInstallKeyAllowedCountDrift(
	spec managementagentv1beta1.ManagementAgentInstallKeySpec,
	current managementagentsdk.ManagementAgentInstallKey,
) error {
	if current.AllowedKeyInstallCount == nil && spec.AllowedKeyInstallCount == 0 {
		return nil
	}
	if current.AllowedKeyInstallCount != nil && *current.AllowedKeyInstallCount == spec.AllowedKeyInstallCount {
		return nil
	}
	return fmt.Errorf("%s create-only field allowedKeyInstallCount changed", managementAgentInstallKeyKind)
}

func validateManagementAgentInstallKeyExpiresDrift(
	spec managementagentv1beta1.ManagementAgentInstallKeySpec,
	current managementagentsdk.ManagementAgentInstallKey,
) error {
	desiredValue := strings.TrimSpace(spec.TimeExpires)
	if current.TimeExpires == nil && desiredValue == "" {
		return nil
	}
	if current.TimeExpires == nil || desiredValue == "" {
		return fmt.Errorf("%s create-only field timeExpires changed", managementAgentInstallKeyKind)
	}
	desired, err := parseManagementAgentInstallKeyTime(desiredValue)
	if err != nil {
		return err
	}
	if !current.TimeExpires.Equal(desired.Time) {
		return fmt.Errorf("%s create-only field timeExpires changed", managementAgentInstallKeyKind)
	}
	return nil
}

func validateManagementAgentInstallKeyUnlimitedDrift(
	spec managementagentv1beta1.ManagementAgentInstallKeySpec,
	current managementagentsdk.ManagementAgentInstallKey,
) error {
	if current.IsUnlimited == nil && !spec.IsUnlimited {
		return nil
	}
	if current.IsUnlimited != nil && *current.IsUnlimited == spec.IsUnlimited {
		return nil
	}
	return fmt.Errorf("%s create-only field isUnlimited changed", managementAgentInstallKeyKind)
}

func managementAgentInstallKeyActiveUpdate(desired bool, current managementagentsdk.LifecycleStatesEnum) (bool, bool) {
	switch current {
	case managementagentsdk.LifecycleStatesActive:
		return desired, !desired
	case managementagentsdk.LifecycleStatesInactive:
		return desired, desired
	default:
		return false, false
	}
}

func paginatedManagementAgentInstallKeyListCall(
	call func(context.Context, managementagentsdk.ListManagementAgentInstallKeysRequest) (managementagentsdk.ListManagementAgentInstallKeysResponse, error),
) func(context.Context, managementagentsdk.ListManagementAgentInstallKeysRequest) (managementagentsdk.ListManagementAgentInstallKeysResponse, error) {
	if call == nil {
		return nil
	}
	return func(ctx context.Context, request managementagentsdk.ListManagementAgentInstallKeysRequest) (managementagentsdk.ListManagementAgentInstallKeysResponse, error) {
		var combined managementagentsdk.ListManagementAgentInstallKeysResponse
		nextPage := request.Page
		for {
			response, err := call(ctx, managementAgentInstallKeyListPageRequest(request, nextPage))
			if err != nil {
				return response, err
			}
			mergeManagementAgentInstallKeyListPage(&combined, response)
			if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
				return combined, nil
			}
			nextPage = response.OpcNextPage
		}
	}
}

func managementAgentInstallKeyListPageRequest(
	request managementagentsdk.ListManagementAgentInstallKeysRequest,
	nextPage *string,
) managementagentsdk.ListManagementAgentInstallKeysRequest {
	pageRequest := request
	pageRequest.Page = nextPage
	return pageRequest
}

func mergeManagementAgentInstallKeyListPage(
	combined *managementagentsdk.ListManagementAgentInstallKeysResponse,
	response managementagentsdk.ListManagementAgentInstallKeysResponse,
) {
	if combined.RawResponse == nil {
		combined.RawResponse = response.RawResponse
	}
	if combined.OpcRequestId == nil {
		combined.OpcRequestId = response.OpcRequestId
	}
	combined.Items = append(combined.Items, response.Items...)
}

func handleManagementAgentInstallKeyDeleteError(resource *managementagentv1beta1.ManagementAgentInstallKey, err error) error {
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
		managementAgentInstallKeyKind,
		classification.HTTPStatusCodeString(),
		classification.ErrorCodeString())
}

func clearTrackedManagementAgentInstallKeyIdentity(resource *managementagentv1beta1.ManagementAgentInstallKey) {
	if resource == nil {
		return
	}
	resource.Status = managementagentv1beta1.ManagementAgentInstallKeyStatus{}
}

type managementAgentInstallKeyDeleteGuardClient struct {
	delegate ManagementAgentInstallKeyServiceClient
	get      func(context.Context, managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error)
}

type managementAgentInstallKeyNormalizeClient struct {
	delegate ManagementAgentInstallKeyServiceClient
}

func (c managementAgentInstallKeyNormalizeClient) CreateOrUpdate(
	ctx context.Context,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	normalizeManagementAgentInstallKeyDesiredState(resource, nil)
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c managementAgentInstallKeyNormalizeClient) Delete(ctx context.Context, resource *managementagentv1beta1.ManagementAgentInstallKey) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func (c managementAgentInstallKeyDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *managementagentv1beta1.ManagementAgentInstallKey,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c managementAgentInstallKeyDeleteGuardClient) Delete(ctx context.Context, resource *managementagentv1beta1.ManagementAgentInstallKey) (bool, error) {
	currentID := managementAgentInstallKeyTrackedID(resource)
	if currentID == "" || c.get == nil {
		return c.delegate.Delete(ctx, resource)
	}

	_, err := c.get(ctx, managementagentsdk.GetManagementAgentInstallKeyRequest{ManagementAgentInstallKeyId: common.String(currentID)})
	if err != nil && errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return false, handleManagementAgentInstallKeyDeleteError(resource, err)
	}
	return c.delegate.Delete(ctx, resource)
}

func managementAgentInstallKeyTrackedID(resource *managementagentv1beta1.ManagementAgentInstallKey) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func managementAgentInstallKeyFromResponse(response any) (managementagentsdk.ManagementAgentInstallKey, bool) {
	switch typed := response.(type) {
	case managementagentsdk.ManagementAgentInstallKey:
		return typed, true
	case *managementagentsdk.ManagementAgentInstallKey:
		if typed == nil {
			return managementagentsdk.ManagementAgentInstallKey{}, false
		}
		return *typed, true
	default:
		return managementAgentInstallKeyFromDerivedResponse(response)
	}
}

func managementAgentInstallKeyFromDerivedResponse(response any) (managementagentsdk.ManagementAgentInstallKey, bool) {
	switch typed := response.(type) {
	case managementagentsdk.ManagementAgentInstallKeySummary:
		return managementAgentInstallKeyFromSummary(typed), true
	case *managementagentsdk.ManagementAgentInstallKeySummary:
		if typed == nil {
			return managementagentsdk.ManagementAgentInstallKey{}, false
		}
		return managementAgentInstallKeyFromSummary(*typed), true
	default:
		return managementAgentInstallKeyFromOperationResponse(response)
	}
}

func managementAgentInstallKeyFromOperationResponse(response any) (managementagentsdk.ManagementAgentInstallKey, bool) {
	switch typed := response.(type) {
	case managementagentsdk.CreateManagementAgentInstallKeyResponse:
		return typed.ManagementAgentInstallKey, true
	case *managementagentsdk.CreateManagementAgentInstallKeyResponse:
		if typed == nil {
			return managementagentsdk.ManagementAgentInstallKey{}, false
		}
		return typed.ManagementAgentInstallKey, true
	case managementagentsdk.GetManagementAgentInstallKeyResponse:
		return typed.ManagementAgentInstallKey, true
	case *managementagentsdk.GetManagementAgentInstallKeyResponse:
		if typed == nil {
			return managementagentsdk.ManagementAgentInstallKey{}, false
		}
		return typed.ManagementAgentInstallKey, true
	case managementagentsdk.UpdateManagementAgentInstallKeyResponse:
		return typed.ManagementAgentInstallKey, true
	case *managementagentsdk.UpdateManagementAgentInstallKeyResponse:
		if typed == nil {
			return managementagentsdk.ManagementAgentInstallKey{}, false
		}
		return typed.ManagementAgentInstallKey, true
	default:
		return managementagentsdk.ManagementAgentInstallKey{}, false
	}
}

func managementAgentInstallKeyFromSummary(summary managementagentsdk.ManagementAgentInstallKeySummary) managementagentsdk.ManagementAgentInstallKey {
	return managementagentsdk.ManagementAgentInstallKey{
		Id:                     summary.Id,
		CompartmentId:          summary.CompartmentId,
		DisplayName:            summary.DisplayName,
		CreatedByPrincipalId:   summary.CreatedByPrincipalId,
		AllowedKeyInstallCount: summary.AllowedKeyInstallCount,
		CurrentKeyInstallCount: summary.CurrentKeyInstallCount,
		LifecycleState:         summary.LifecycleState,
		LifecycleDetails:       summary.LifecycleDetails,
		TimeCreated:            summary.TimeCreated,
		TimeExpires:            summary.TimeExpires,
		IsUnlimited:            summary.IsUnlimited,
		FreeformTags:           summary.FreeformTags,
		DefinedTags:            summary.DefinedTags,
		SystemTags:             summary.SystemTags,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func newManagementAgentInstallKeyServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client managementAgentInstallKeyOCIClient,
) ManagementAgentInstallKeyServiceClient {
	hooks := newManagementAgentInstallKeyRuntimeHooksWithOCIClient(client)
	applyManagementAgentInstallKeyRuntimeHooks(&hooks)
	delegate := defaultManagementAgentInstallKeyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*managementagentv1beta1.ManagementAgentInstallKey](
			buildManagementAgentInstallKeyGeneratedRuntimeConfig(&ManagementAgentInstallKeyServiceManager{Log: log}, hooks),
		),
	}
	return wrapManagementAgentInstallKeyGeneratedClient(hooks, delegate)
}

func newManagementAgentInstallKeyRuntimeHooksWithOCIClient(client managementAgentInstallKeyOCIClient) ManagementAgentInstallKeyRuntimeHooks {
	return ManagementAgentInstallKeyRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*managementagentv1beta1.ManagementAgentInstallKey]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*managementagentv1beta1.ManagementAgentInstallKey]{},
		StatusHooks:     generatedruntime.StatusHooks[*managementagentv1beta1.ManagementAgentInstallKey]{},
		ParityHooks:     generatedruntime.ParityHooks[*managementagentv1beta1.ManagementAgentInstallKey]{},
		Async:           generatedruntime.AsyncHooks[*managementagentv1beta1.ManagementAgentInstallKey]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*managementagentv1beta1.ManagementAgentInstallKey]{},
		Create: runtimeOperationHooks[managementagentsdk.CreateManagementAgentInstallKeyRequest, managementagentsdk.CreateManagementAgentInstallKeyResponse]{
			Fields: managementAgentInstallKeyCreateFields(),
			Call: func(ctx context.Context, request managementagentsdk.CreateManagementAgentInstallKeyRequest) (managementagentsdk.CreateManagementAgentInstallKeyResponse, error) {
				return client.CreateManagementAgentInstallKey(ctx, request)
			},
		},
		Get: runtimeOperationHooks[managementagentsdk.GetManagementAgentInstallKeyRequest, managementagentsdk.GetManagementAgentInstallKeyResponse]{
			Fields: managementAgentInstallKeyGetFields(),
			Call: func(ctx context.Context, request managementagentsdk.GetManagementAgentInstallKeyRequest) (managementagentsdk.GetManagementAgentInstallKeyResponse, error) {
				return client.GetManagementAgentInstallKey(ctx, request)
			},
		},
		List: runtimeOperationHooks[managementagentsdk.ListManagementAgentInstallKeysRequest, managementagentsdk.ListManagementAgentInstallKeysResponse]{
			Fields: managementAgentInstallKeyListFields(),
			Call: func(ctx context.Context, request managementagentsdk.ListManagementAgentInstallKeysRequest) (managementagentsdk.ListManagementAgentInstallKeysResponse, error) {
				return client.ListManagementAgentInstallKeys(ctx, request)
			},
		},
		Update: runtimeOperationHooks[managementagentsdk.UpdateManagementAgentInstallKeyRequest, managementagentsdk.UpdateManagementAgentInstallKeyResponse]{
			Fields: managementAgentInstallKeyUpdateFields(),
			Call: func(ctx context.Context, request managementagentsdk.UpdateManagementAgentInstallKeyRequest) (managementagentsdk.UpdateManagementAgentInstallKeyResponse, error) {
				return client.UpdateManagementAgentInstallKey(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[managementagentsdk.DeleteManagementAgentInstallKeyRequest, managementagentsdk.DeleteManagementAgentInstallKeyResponse]{
			Fields: managementAgentInstallKeyDeleteFields(),
			Call: func(ctx context.Context, request managementagentsdk.DeleteManagementAgentInstallKeyRequest) (managementagentsdk.DeleteManagementAgentInstallKeyResponse, error) {
				return client.DeleteManagementAgentInstallKey(ctx, request)
			},
		},
		WrapGeneratedClient: []func(ManagementAgentInstallKeyServiceClient) ManagementAgentInstallKeyServiceClient{},
	}
}
