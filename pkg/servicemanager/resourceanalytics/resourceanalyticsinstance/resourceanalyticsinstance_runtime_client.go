/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package resourceanalyticsinstance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	resourceanalyticssdk "github.com/oracle/oci-go-sdk/v65/resourceanalytics"
	resourceanalyticsv1beta1 "github.com/oracle/oci-service-operator/api/resourceanalytics/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	resourceAnalyticsInstanceKind                     = "ResourceAnalyticsInstance"
	resourceAnalyticsInstanceCreateOnlyFingerprintKey = "resourceAnalyticsInstanceCreateOnlySHA256="
)

var resourceAnalyticsInstanceWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(resourceanalyticssdk.OperationStatusAccepted),
		string(resourceanalyticssdk.OperationStatusInProgress),
		string(resourceanalyticssdk.OperationStatusWaiting),
		string(resourceanalyticssdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(resourceanalyticssdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(resourceanalyticssdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(resourceanalyticssdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(resourceanalyticssdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(resourceanalyticssdk.OperationTypeCreateResourceAnalyticsInstance)},
	UpdateActionTokens:    []string{string(resourceanalyticssdk.OperationTypeUpdateResourceAnalyticsInstance)},
	DeleteActionTokens:    []string{string(resourceanalyticssdk.OperationTypeDeleteResourceAnalyticsInstance)},
}

type resourceAnalyticsInstanceOCIClient interface {
	CreateResourceAnalyticsInstance(context.Context, resourceanalyticssdk.CreateResourceAnalyticsInstanceRequest) (resourceanalyticssdk.CreateResourceAnalyticsInstanceResponse, error)
	GetResourceAnalyticsInstance(context.Context, resourceanalyticssdk.GetResourceAnalyticsInstanceRequest) (resourceanalyticssdk.GetResourceAnalyticsInstanceResponse, error)
	ListResourceAnalyticsInstances(context.Context, resourceanalyticssdk.ListResourceAnalyticsInstancesRequest) (resourceanalyticssdk.ListResourceAnalyticsInstancesResponse, error)
	UpdateResourceAnalyticsInstance(context.Context, resourceanalyticssdk.UpdateResourceAnalyticsInstanceRequest) (resourceanalyticssdk.UpdateResourceAnalyticsInstanceResponse, error)
	DeleteResourceAnalyticsInstance(context.Context, resourceanalyticssdk.DeleteResourceAnalyticsInstanceRequest) (resourceanalyticssdk.DeleteResourceAnalyticsInstanceResponse, error)
	GetWorkRequest(context.Context, resourceanalyticssdk.GetWorkRequestRequest) (resourceanalyticssdk.GetWorkRequestResponse, error)
}

type resourceAnalyticsInstanceListCall func(context.Context, resourceanalyticssdk.ListResourceAnalyticsInstancesRequest) (resourceanalyticssdk.ListResourceAnalyticsInstancesResponse, error)

type resourceAnalyticsInstanceAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e resourceAnalyticsInstanceAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e resourceAnalyticsInstanceAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerResourceAnalyticsInstanceRuntimeHooksMutator(func(manager *ResourceAnalyticsInstanceServiceManager, hooks *ResourceAnalyticsInstanceRuntimeHooks) {
		client, initErr := newResourceAnalyticsInstanceOCIClient(manager)
		applyResourceAnalyticsInstanceRuntimeHooks(hooks, client, initErr)
	})
}

func newResourceAnalyticsInstanceOCIClient(manager *ResourceAnalyticsInstanceServiceManager) (resourceAnalyticsInstanceOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", resourceAnalyticsInstanceKind)
	}
	client, err := resourceanalyticssdk.NewResourceAnalyticsInstanceClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyResourceAnalyticsInstanceRuntimeHooks(
	hooks *ResourceAnalyticsInstanceRuntimeHooks,
	client resourceAnalyticsInstanceOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = resourceAnalyticsInstanceRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance, _ string) (any, error) {
		return buildResourceAnalyticsInstanceCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildResourceAnalyticsInstanceUpdateBody(resource, currentResponse)
	}
	hooks.List.Fields = resourceAnalyticsInstanceListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listResourceAnalyticsInstancesAllPages(hooks.List.Call)
	}
	hooks.Identity.Resolve = resolveResourceAnalyticsInstanceIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardResourceAnalyticsInstanceExistingBeforeCreate
	hooks.Identity.LookupExisting = lookupExistingResourceAnalyticsInstance(client, initErr)
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateResourceAnalyticsInstanceCreateOnlyDrift
	hooks.Async.Adapter = resourceAnalyticsInstanceWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getResourceAnalyticsInstanceWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveResourceAnalyticsInstanceWorkRequestAction
	hooks.Async.ResolvePhase = resolveResourceAnalyticsInstanceWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverResourceAnalyticsInstanceIDFromWorkRequest
	hooks.Async.Message = resourceAnalyticsInstanceWorkRequestMessage
	hooks.DeleteHooks.HandleError = handleResourceAnalyticsInstanceDeleteError
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ResourceAnalyticsInstanceServiceClient) ResourceAnalyticsInstanceServiceClient {
		return resourceAnalyticsInstanceCreateOnlyTrackingClient{delegate: delegate}
	})
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapResourceAnalyticsInstanceDeleteSafety(client, initErr))
}

func resourceAnalyticsInstanceRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "resourceanalytics",
		FormalSlug:    "resourceanalyticsinstance",
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
			ProvisioningStates: []string{string(resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateCreating)},
			UpdatingStates:     []string{string(resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateUpdating)},
			ActiveStates:       []string{string(resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateDeleting)},
			TerminalStates: []string{string(resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
				"adwAdminPassword",
				"subnetId",
				"isMutualTlsRequired",
				"nsgIds",
				"licenseModel",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: resourceAnalyticsInstanceKind, Action: "CreateResourceAnalyticsInstance"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: resourceAnalyticsInstanceKind, Action: "UpdateResourceAnalyticsInstance"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: resourceAnalyticsInstanceKind, Action: "DeleteResourceAnalyticsInstance"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: resourceAnalyticsInstanceKind, Action: "GetResourceAnalyticsInstance"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: resourceAnalyticsInstanceKind, Action: "GetResourceAnalyticsInstance"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: resourceAnalyticsInstanceKind, Action: "GetResourceAnalyticsInstance"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func resourceAnalyticsInstanceListFields() []generatedruntime.RequestField {
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
			LookupPaths:  []string{"spec.displayName", "status.displayName", "displayName"},
		},
		{
			FieldName:        "Id",
			RequestName:      "id",
			Contribution:     "query",
			PreferResourceID: true,
			LookupPaths:      []string{"status.id", "status.status.ocid", "id", "ocid"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func guardResourceAnalyticsInstanceExistingBeforeCreate(
	_ context.Context,
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("%s resource is nil", resourceAnalyticsInstanceKind)
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

type resourceAnalyticsInstanceIdentity struct {
	compartmentID string
	displayName   string
}

func resolveResourceAnalyticsInstanceIdentity(
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", resourceAnalyticsInstanceKind)
	}
	displayName := strings.TrimSpace(resource.Spec.DisplayName)
	if displayName == "" {
		return nil, nil
	}
	return resourceAnalyticsInstanceIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		displayName:   displayName,
	}, nil
}

func lookupExistingResourceAnalyticsInstance(
	client resourceAnalyticsInstanceOCIClient,
	initErr error,
) func(context.Context, *resourceanalyticsv1beta1.ResourceAnalyticsInstance, any) (any, error) {
	return func(ctx context.Context, _ *resourceanalyticsv1beta1.ResourceAnalyticsInstance, identity any) (any, error) {
		if initErr != nil {
			return nil, initErr
		}
		if client == nil || identity == nil {
			return nil, nil
		}
		resolved, ok := identity.(resourceAnalyticsInstanceIdentity)
		if !ok {
			return nil, fmt.Errorf("unexpected %s identity type %T", resourceAnalyticsInstanceKind, identity)
		}
		if resolved.compartmentID == "" || resolved.displayName == "" {
			return nil, nil
		}
		match, found, err := lookupResourceAnalyticsInstanceByDisplayName(ctx, client, resolved)
		if err != nil || !found {
			return nil, err
		}
		return match, nil
	}
}

func lookupResourceAnalyticsInstanceByDisplayName(
	ctx context.Context,
	client resourceAnalyticsInstanceOCIClient,
	identity resourceAnalyticsInstanceIdentity,
) (resourceanalyticssdk.ResourceAnalyticsInstanceSummary, bool, error) {
	request := resourceanalyticssdk.ListResourceAnalyticsInstancesRequest{
		CompartmentId: common.String(identity.compartmentID),
		DisplayName:   common.String(identity.displayName),
	}
	var match *resourceanalyticssdk.ResourceAnalyticsInstanceSummary
	seenPages := map[string]struct{}{}
	for {
		response, err := client.ListResourceAnalyticsInstances(ctx, request)
		if err != nil {
			return resourceanalyticssdk.ResourceAnalyticsInstanceSummary{}, false, err
		}

		match, err = mergeResourceAnalyticsInstanceListMatch(match, response.Items, identity)
		if err != nil {
			return resourceanalyticssdk.ResourceAnalyticsInstanceSummary{}, false, err
		}
		advanced, err := advanceResourceAnalyticsInstanceLookupPage(&request, response.OpcNextPage, seenPages)
		if err != nil {
			return resourceanalyticssdk.ResourceAnalyticsInstanceSummary{}, false, err
		}
		if !advanced {
			break
		}
	}
	if match == nil {
		return resourceanalyticssdk.ResourceAnalyticsInstanceSummary{}, false, nil
	}
	return *match, true, nil
}

func mergeResourceAnalyticsInstanceListMatch(
	match *resourceanalyticssdk.ResourceAnalyticsInstanceSummary,
	items []resourceanalyticssdk.ResourceAnalyticsInstanceSummary,
	identity resourceAnalyticsInstanceIdentity,
) (*resourceanalyticssdk.ResourceAnalyticsInstanceSummary, error) {
	for _, item := range items {
		if !resourceAnalyticsInstanceSummaryMatchesIdentity(item, identity) {
			continue
		}
		if resourceAnalyticsInstanceConflictsWithMatch(match, item) {
			return nil, fmt.Errorf("%s list returned multiple matches for displayName %q in compartment %q", resourceAnalyticsInstanceKind, identity.displayName, identity.compartmentID)
		}
		current := item
		match = &current
	}
	return match, nil
}

func resourceAnalyticsInstanceConflictsWithMatch(
	match *resourceanalyticssdk.ResourceAnalyticsInstanceSummary,
	item resourceanalyticssdk.ResourceAnalyticsInstanceSummary,
) bool {
	return match != nil && resourceAnalyticsInstanceStringValue(match.Id) != resourceAnalyticsInstanceStringValue(item.Id)
}

func advanceResourceAnalyticsInstanceLookupPage(
	request *resourceanalyticssdk.ListResourceAnalyticsInstancesRequest,
	opcNextPage *string,
	seenPages map[string]struct{},
) (bool, error) {
	nextPage := resourceAnalyticsInstanceStringValue(opcNextPage)
	if nextPage == "" {
		return false, nil
	}
	if _, ok := seenPages[nextPage]; ok {
		return false, fmt.Errorf("%s list pagination repeated page token %q", resourceAnalyticsInstanceKind, nextPage)
	}
	seenPages[nextPage] = struct{}{}
	request.Page = common.String(nextPage)
	return true, nil
}

func resourceAnalyticsInstanceSummaryMatchesIdentity(
	summary resourceanalyticssdk.ResourceAnalyticsInstanceSummary,
	identity resourceAnalyticsInstanceIdentity,
) bool {
	return resourceAnalyticsInstanceSummaryAllowsBind(summary) &&
		resourceAnalyticsInstanceStringValue(summary.CompartmentId) == identity.compartmentID &&
		resourceAnalyticsInstanceStringValue(summary.DisplayName) == identity.displayName
}

func resourceAnalyticsInstanceSummaryAllowsBind(summary resourceanalyticssdk.ResourceAnalyticsInstanceSummary) bool {
	switch summary.LifecycleState {
	case resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateDeleting,
		resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateDeleted,
		resourceanalyticssdk.ResourceAnalyticsInstanceLifecycleStateFailed:
		return false
	default:
		return true
	}
}

func buildResourceAnalyticsInstanceCreateBody(
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
) (resourceanalyticssdk.CreateResourceAnalyticsInstanceDetails, error) {
	if resource == nil {
		return resourceanalyticssdk.CreateResourceAnalyticsInstanceDetails{}, fmt.Errorf("%s resource is nil", resourceAnalyticsInstanceKind)
	}
	password, err := resourceAnalyticsInstancePasswordDetails(resource.Spec.AdwAdminPassword)
	if err != nil {
		return resourceanalyticssdk.CreateResourceAnalyticsInstanceDetails{}, err
	}
	if err := validateResourceAnalyticsInstanceCreateSpec(resource.Spec); err != nil {
		return resourceanalyticssdk.CreateResourceAnalyticsInstanceDetails{}, err
	}

	details := resourceanalyticssdk.CreateResourceAnalyticsInstanceDetails{
		CompartmentId:       common.String(resource.Spec.CompartmentId),
		AdwAdminPassword:    password,
		SubnetId:            common.String(resource.Spec.SubnetId),
		IsMutualTlsRequired: common.Bool(resource.Spec.IsMutualTlsRequired),
	}
	if err := applyResourceAnalyticsInstanceOptionalCreateFields(&details, resource.Spec); err != nil {
		return resourceanalyticssdk.CreateResourceAnalyticsInstanceDetails{}, err
	}
	return details, nil
}

func validateResourceAnalyticsInstanceCreateSpec(spec resourceanalyticsv1beta1.ResourceAnalyticsInstanceSpec) error {
	if strings.TrimSpace(spec.CompartmentId) == "" {
		return fmt.Errorf("%s spec.compartmentId is required", resourceAnalyticsInstanceKind)
	}
	if strings.TrimSpace(spec.SubnetId) == "" {
		return fmt.Errorf("%s spec.subnetId is required", resourceAnalyticsInstanceKind)
	}
	return nil
}

func applyResourceAnalyticsInstanceOptionalCreateFields(
	details *resourceanalyticssdk.CreateResourceAnalyticsInstanceDetails,
	spec resourceanalyticsv1beta1.ResourceAnalyticsInstanceSpec,
) error {
	if strings.TrimSpace(spec.DisplayName) != "" {
		details.DisplayName = common.String(spec.DisplayName)
	}
	if spec.Description != "" {
		details.Description = common.String(spec.Description)
	}
	if len(spec.NsgIds) > 0 {
		details.NsgIds = slices.Clone(spec.NsgIds)
	}
	if err := setResourceAnalyticsInstanceCreateLicenseModel(details, spec.LicenseModel); err != nil {
		return err
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = resourceAnalyticsInstanceDefinedTagsFromSpec(spec.DefinedTags)
	}
	return nil
}

func setResourceAnalyticsInstanceCreateLicenseModel(
	details *resourceanalyticssdk.CreateResourceAnalyticsInstanceDetails,
	licenseModel string,
) error {
	if strings.TrimSpace(licenseModel) == "" {
		return nil
	}
	model, ok := resourceanalyticssdk.GetMappingCreateResourceAnalyticsInstanceDetailsLicenseModelEnum(licenseModel)
	if !ok {
		return fmt.Errorf("%s spec.licenseModel %q is not supported", resourceAnalyticsInstanceKind, licenseModel)
	}
	details.LicenseModel = model
	return nil
}

func resourceAnalyticsInstancePasswordDetails(
	spec resourceanalyticsv1beta1.ResourceAnalyticsInstanceAdwAdminPassword,
) (resourceanalyticssdk.AdwAdminPasswordDetails, error) {
	if strings.TrimSpace(spec.JsonData) != "" {
		return resourceAnalyticsInstancePasswordDetailsFromJSON(spec.JsonData)
	}

	passwordType := strings.ToUpper(strings.TrimSpace(spec.PasswordType))
	switch {
	case passwordType == "" && spec.Password != "":
		passwordType = string(resourceanalyticssdk.AdwAdminPasswordDetailsPasswordTypePlainText)
	case passwordType == "" && spec.SecretId != "":
		passwordType = string(resourceanalyticssdk.AdwAdminPasswordDetailsPasswordTypeVaultSecret)
	}

	switch resourceanalyticssdk.AdwAdminPasswordDetailsPasswordTypeEnum(passwordType) {
	case resourceanalyticssdk.AdwAdminPasswordDetailsPasswordTypePlainText:
		if spec.Password == "" {
			return nil, fmt.Errorf("%s spec.adwAdminPassword.password is required for PLAIN_TEXT", resourceAnalyticsInstanceKind)
		}
		return resourceanalyticssdk.PlainTextPasswordDetails{Password: common.String(spec.Password)}, nil
	case resourceanalyticssdk.AdwAdminPasswordDetailsPasswordTypeVaultSecret:
		if strings.TrimSpace(spec.SecretId) == "" {
			return nil, fmt.Errorf("%s spec.adwAdminPassword.secretId is required for VAULT_SECRET", resourceAnalyticsInstanceKind)
		}
		return resourceanalyticssdk.VaultSecretPasswordDetails{SecretId: common.String(spec.SecretId)}, nil
	default:
		return nil, fmt.Errorf("%s spec.adwAdminPassword.passwordType %q is not supported", resourceAnalyticsInstanceKind, spec.PasswordType)
	}
}

func resourceAnalyticsInstancePasswordDetailsFromJSON(raw string) (resourceanalyticssdk.AdwAdminPasswordDetails, error) {
	var decoded resourceanalyticsv1beta1.ResourceAnalyticsInstanceAdwAdminPassword
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("decode %s spec.adwAdminPassword.jsonData: %w", resourceAnalyticsInstanceKind, err)
	}
	return resourceAnalyticsInstancePasswordDetails(decoded)
}

func buildResourceAnalyticsInstanceUpdateBody(
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
	currentResponse any,
) (resourceanalyticssdk.UpdateResourceAnalyticsInstanceDetails, bool, error) {
	if resource == nil {
		return resourceanalyticssdk.UpdateResourceAnalyticsInstanceDetails{}, false, fmt.Errorf("%s resource is nil", resourceAnalyticsInstanceKind)
	}

	current, err := observedResourceAnalyticsInstanceState(resource, currentResponse)
	if err != nil {
		return resourceanalyticssdk.UpdateResourceAnalyticsInstanceDetails{}, false, err
	}

	details := resourceanalyticssdk.UpdateResourceAnalyticsInstanceDetails{}
	updateNeeded := false
	if desired, ok := resourceAnalyticsInstanceDesiredDisplayNameUpdate(resource.Spec.DisplayName, current.displayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := resourceAnalyticsInstanceDesiredDescriptionUpdate(resource.Spec.Description, current.description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := resourceAnalyticsInstanceDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.freeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := resourceAnalyticsInstanceDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.definedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}
	return details, updateNeeded, nil
}

type resourceAnalyticsInstanceObservedState struct {
	displayName  *string
	description  *string
	freeformTags map[string]string
	definedTags  map[string]map[string]interface{}
}

func observedResourceAnalyticsInstanceState(
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
	response any,
) (resourceAnalyticsInstanceObservedState, error) {
	state := resourceAnalyticsInstanceObservedState{}
	if resource != nil {
		if resource.Status.DisplayName != "" {
			state.displayName = common.String(resource.Status.DisplayName)
		}
		if resource.Status.Description != "" {
			state.description = common.String(resource.Status.Description)
		}
		state.freeformTags = maps.Clone(resource.Status.FreeformTags)
		state.definedTags = resourceAnalyticsInstanceDefinedTagsFromStatus(resource.Status.DefinedTags)
	}
	if response == nil {
		return state, nil
	}
	body, ok, err := resourceAnalyticsInstanceBody(response)
	if err != nil || !ok {
		return state, err
	}
	state.apply(body)
	return state, nil
}

func (s *resourceAnalyticsInstanceObservedState) apply(body resourceanalyticssdk.ResourceAnalyticsInstance) {
	s.displayName = body.DisplayName
	s.description = body.Description
	s.freeformTags = maps.Clone(body.FreeformTags)
	s.definedTags = cloneResourceAnalyticsInstanceDefinedTags(body.DefinedTags)
}

func resourceAnalyticsInstanceBody(response any) (resourceanalyticssdk.ResourceAnalyticsInstance, bool, error) {
	if body, matched, err := resourceAnalyticsInstanceDirectBody(response); matched || err != nil {
		return body, matched, err
	}
	if body, matched, err := resourceAnalyticsInstanceSummaryBody(response); matched || err != nil {
		return body, matched, err
	}
	if body, matched, err := resourceAnalyticsInstanceResponseBody(response); matched || err != nil {
		return body, matched, err
	}
	return resourceanalyticssdk.ResourceAnalyticsInstance{}, false, fmt.Errorf("unexpected current %s response type %T", resourceAnalyticsInstanceKind, response)
}

func resourceAnalyticsInstanceDirectBody(response any) (resourceanalyticssdk.ResourceAnalyticsInstance, bool, error) {
	switch current := response.(type) {
	case resourceanalyticssdk.ResourceAnalyticsInstance:
		return current, true, nil
	case *resourceanalyticssdk.ResourceAnalyticsInstance:
		if current == nil {
			return resourceanalyticssdk.ResourceAnalyticsInstance{}, false, fmt.Errorf("current %s response is nil", resourceAnalyticsInstanceKind)
		}
		return *current, true, nil
	default:
		return resourceanalyticssdk.ResourceAnalyticsInstance{}, false, nil
	}
}

func resourceAnalyticsInstanceSummaryBody(response any) (resourceanalyticssdk.ResourceAnalyticsInstance, bool, error) {
	switch current := response.(type) {
	case resourceanalyticssdk.ResourceAnalyticsInstanceSummary:
		return resourceAnalyticsInstanceFromSummary(current), true, nil
	case *resourceanalyticssdk.ResourceAnalyticsInstanceSummary:
		if current == nil {
			return resourceanalyticssdk.ResourceAnalyticsInstance{}, false, fmt.Errorf("current %s response is nil", resourceAnalyticsInstanceKind)
		}
		return resourceAnalyticsInstanceFromSummary(*current), true, nil
	default:
		return resourceanalyticssdk.ResourceAnalyticsInstance{}, false, nil
	}
}

func resourceAnalyticsInstanceResponseBody(response any) (resourceanalyticssdk.ResourceAnalyticsInstance, bool, error) {
	switch current := response.(type) {
	case resourceanalyticssdk.CreateResourceAnalyticsInstanceResponse:
		return current.ResourceAnalyticsInstance, true, nil
	case *resourceanalyticssdk.CreateResourceAnalyticsInstanceResponse:
		if current == nil {
			return resourceanalyticssdk.ResourceAnalyticsInstance{}, false, fmt.Errorf("current %s response is nil", resourceAnalyticsInstanceKind)
		}
		return current.ResourceAnalyticsInstance, true, nil
	case resourceanalyticssdk.GetResourceAnalyticsInstanceResponse:
		return current.ResourceAnalyticsInstance, true, nil
	case *resourceanalyticssdk.GetResourceAnalyticsInstanceResponse:
		if current == nil {
			return resourceanalyticssdk.ResourceAnalyticsInstance{}, false, fmt.Errorf("current %s response is nil", resourceAnalyticsInstanceKind)
		}
		return current.ResourceAnalyticsInstance, true, nil
	default:
		return resourceanalyticssdk.ResourceAnalyticsInstance{}, false, nil
	}
}

func resourceAnalyticsInstanceFromSummary(summary resourceanalyticssdk.ResourceAnalyticsInstanceSummary) resourceanalyticssdk.ResourceAnalyticsInstance {
	return resourceanalyticssdk.ResourceAnalyticsInstance{
		Id:               summary.Id,
		DisplayName:      summary.DisplayName,
		CompartmentId:    summary.CompartmentId,
		TimeCreated:      summary.TimeCreated,
		LifecycleState:   summary.LifecycleState,
		FreeformTags:     summary.FreeformTags,
		DefinedTags:      summary.DefinedTags,
		Description:      summary.Description,
		TimeUpdated:      summary.TimeUpdated,
		LifecycleDetails: summary.LifecycleDetails,
		SystemTags:       summary.SystemTags,
	}
}

func resourceAnalyticsInstanceDesiredDisplayNameUpdate(spec string, current *string) (*string, bool) {
	if strings.TrimSpace(spec) == "" {
		return nil, false
	}
	return resourceAnalyticsInstanceDesiredStringUpdate(spec, current)
}

func resourceAnalyticsInstanceDesiredDescriptionUpdate(spec string, current *string) (*string, bool) {
	if strings.TrimSpace(spec) == "" {
		return nil, false
	}
	return resourceAnalyticsInstanceDesiredStringUpdate(spec, current)
}

func resourceAnalyticsInstanceDesiredStringUpdate(spec string, current *string) (*string, bool) {
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

func resourceAnalyticsInstanceDesiredFreeformTagsUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func resourceAnalyticsInstanceDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := resourceAnalyticsInstanceDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if resourceAnalyticsInstanceJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func getResourceAnalyticsInstanceWorkRequest(
	ctx context.Context,
	client resourceAnalyticsInstanceOCIClient,
	initErr error,
	workRequestID string,
) (resourceanalyticssdk.WorkRequest, error) {
	if initErr != nil {
		return resourceanalyticssdk.WorkRequest{}, initErr
	}
	if client == nil {
		return resourceanalyticssdk.WorkRequest{}, fmt.Errorf("%s work request client is not configured", resourceAnalyticsInstanceKind)
	}
	workRequestID = strings.TrimSpace(workRequestID)
	if workRequestID == "" {
		return resourceanalyticssdk.WorkRequest{}, fmt.Errorf("%s work request ID is required", resourceAnalyticsInstanceKind)
	}
	response, err := client.GetWorkRequest(ctx, resourceanalyticssdk.GetWorkRequestRequest{WorkRequestId: common.String(workRequestID)})
	if err != nil {
		return resourceanalyticssdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func resolveResourceAnalyticsInstanceWorkRequestAction(workRequest any) (string, error) {
	current, err := resourceAnalyticsInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveResourceAnalyticsInstanceWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := resourceAnalyticsInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case resourceanalyticssdk.OperationTypeCreateResourceAnalyticsInstance:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case resourceanalyticssdk.OperationTypeUpdateResourceAnalyticsInstance:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case resourceanalyticssdk.OperationTypeDeleteResourceAnalyticsInstance:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func recoverResourceAnalyticsInstanceIDFromWorkRequest(
	_ *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := resourceAnalyticsInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resourceAnalyticsInstanceIDFromWorkRequest(current, resourceAnalyticsInstanceActionForPhase(phase))
}

func resourceAnalyticsInstanceWorkRequestFromAny(workRequest any) (resourceanalyticssdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case resourceanalyticssdk.WorkRequest:
		return current, nil
	case *resourceanalyticssdk.WorkRequest:
		if current == nil {
			return resourceanalyticssdk.WorkRequest{}, fmt.Errorf("%s work request is nil", resourceAnalyticsInstanceKind)
		}
		return *current, nil
	default:
		return resourceanalyticssdk.WorkRequest{}, fmt.Errorf("unexpected %s work request type %T", resourceAnalyticsInstanceKind, workRequest)
	}
}

func resourceAnalyticsInstanceActionForPhase(phase shared.OSOKAsyncPhase) resourceanalyticssdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return resourceanalyticssdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return resourceanalyticssdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return resourceanalyticssdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resourceAnalyticsInstanceIDFromWorkRequest(workRequest resourceanalyticssdk.WorkRequest, action resourceanalyticssdk.ActionTypeEnum) (string, error) {
	if id, ok := resourceAnalyticsInstanceIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resourceAnalyticsInstanceIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("%s work request %s does not expose a resource analytics instance identifier", resourceAnalyticsInstanceKind, resourceAnalyticsInstanceStringValue(workRequest.Id))
}

func resourceAnalyticsInstanceIDFromResources(resources []resourceanalyticssdk.WorkRequestResource, action resourceanalyticssdk.ActionTypeEnum, preferResourceOnly bool) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferResourceOnly && !isResourceAnalyticsInstanceWorkRequestResource(resource) {
			continue
		}
		id := strings.TrimSpace(resourceAnalyticsInstanceStringValue(resource.Identifier))
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

func isResourceAnalyticsInstanceWorkRequestResource(resource resourceanalyticssdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(resourceAnalyticsInstanceStringValue(resource.EntityType)))
	switch entityType {
	case "resourceanalyticsinstance", "resourceanalyticsinstances", "resource_analytics_instance", "resource_analytics_instances":
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(resourceAnalyticsInstanceStringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "resourceanalyticsinstance") ||
		strings.Contains(entityURI, "resource-analytics-instance")
}

func resourceAnalyticsInstanceWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := resourceAnalyticsInstanceWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", resourceAnalyticsInstanceKind, phase, resourceAnalyticsInstanceStringValue(current.Id), current.Status)
}

func listResourceAnalyticsInstancesAllPages(call resourceAnalyticsInstanceListCall) resourceAnalyticsInstanceListCall {
	return func(ctx context.Context, request resourceanalyticssdk.ListResourceAnalyticsInstancesRequest) (resourceanalyticssdk.ListResourceAnalyticsInstancesResponse, error) {
		if call == nil {
			return resourceanalyticssdk.ListResourceAnalyticsInstancesResponse{}, fmt.Errorf("%s list operation is not configured", resourceAnalyticsInstanceKind)
		}
		if !resourceAnalyticsInstanceHasBoundedListRequest(request) {
			return resourceanalyticssdk.ListResourceAnalyticsInstancesResponse{}, nil
		}

		accumulator := newResourceAnalyticsInstanceListAccumulator()
		for {
			response, err := call(ctx, request)
			if err != nil {
				return resourceanalyticssdk.ListResourceAnalyticsInstancesResponse{}, err
			}
			accumulator.append(response)

			nextPage := resourceAnalyticsInstanceStringValue(response.OpcNextPage)
			if nextPage == "" {
				return accumulator.response, nil
			}
			if err := accumulator.advance(&request, nextPage); err != nil {
				return resourceanalyticssdk.ListResourceAnalyticsInstancesResponse{}, err
			}
		}
	}
}

func resourceAnalyticsInstanceHasBoundedListRequest(request resourceanalyticssdk.ListResourceAnalyticsInstancesRequest) bool {
	return strings.TrimSpace(resourceAnalyticsInstanceStringValue(request.Id)) != "" ||
		strings.TrimSpace(resourceAnalyticsInstanceStringValue(request.DisplayName)) != ""
}

type resourceAnalyticsInstanceListAccumulator struct {
	response  resourceanalyticssdk.ListResourceAnalyticsInstancesResponse
	seenPages map[string]struct{}
}

func newResourceAnalyticsInstanceListAccumulator() resourceAnalyticsInstanceListAccumulator {
	return resourceAnalyticsInstanceListAccumulator{seenPages: map[string]struct{}{}}
}

func (a *resourceAnalyticsInstanceListAccumulator) append(response resourceanalyticssdk.ListResourceAnalyticsInstancesResponse) {
	if a.response.RawResponse == nil {
		a.response.RawResponse = response.RawResponse
	}
	if a.response.OpcRequestId == nil {
		a.response.OpcRequestId = response.OpcRequestId
	}
	a.response.OpcNextPage = response.OpcNextPage
	a.response.Items = append(a.response.Items, response.Items...)
}

func (a *resourceAnalyticsInstanceListAccumulator) advance(request *resourceanalyticssdk.ListResourceAnalyticsInstancesRequest, nextPage string) error {
	if _, ok := a.seenPages[nextPage]; ok {
		return fmt.Errorf("%s list pagination repeated page token %q", resourceAnalyticsInstanceKind, nextPage)
	}
	a.seenPages[nextPage] = struct{}{}
	request.Page = common.String(nextPage)
	return nil
}

type resourceAnalyticsInstanceCreateOnlyTrackingClient struct {
	delegate ResourceAnalyticsInstanceServiceClient
}

func (c resourceAnalyticsInstanceCreateOnlyTrackingClient) CreateOrUpdate(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	wasTracked := resourceAnalyticsInstanceHasTrackedIdentity(resource)
	recorded, hasRecorded := resourceAnalyticsInstanceRecordedCreateOnlyFingerprint(resource)
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	switch {
	case hasRecorded && resourceAnalyticsInstanceHasTrackedIdentity(resource):
		setResourceAnalyticsInstanceCreateOnlyFingerprint(resource, recorded)
	case err == nil && response.IsSuccessful && !wasTracked && resourceAnalyticsInstanceHasTrackedIdentity(resource):
		recordResourceAnalyticsInstanceCreateOnlyFingerprint(resource)
	}
	return response, err
}

func (c resourceAnalyticsInstanceCreateOnlyTrackingClient) Delete(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func validateResourceAnalyticsInstanceCreateOnlyDrift(resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance, _ any) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", resourceAnalyticsInstanceKind)
	}
	recorded, ok := resourceAnalyticsInstanceRecordedCreateOnlyFingerprint(resource)
	if !ok {
		if resourceAnalyticsInstanceHasEstablishedTrackedIdentity(resource) {
			return fmt.Errorf("%s create-only fingerprint is missing for tracked resource; recreate the resource before changing create-only fields", resourceAnalyticsInstanceKind)
		}
		return nil
	}
	desired, err := resourceAnalyticsInstanceCreateOnlyFingerprint(resource.Spec)
	if err != nil {
		return err
	}
	if desired != recorded {
		return fmt.Errorf("%s formal semantics require replacement when create-only fields change", resourceAnalyticsInstanceKind)
	}
	return nil
}

func recordResourceAnalyticsInstanceCreateOnlyFingerprint(resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance) {
	if resource == nil {
		return
	}
	fingerprint, err := resourceAnalyticsInstanceCreateOnlyFingerprint(resource.Spec)
	if err != nil {
		return
	}
	setResourceAnalyticsInstanceCreateOnlyFingerprint(resource, fingerprint)
}

func setResourceAnalyticsInstanceCreateOnlyFingerprint(resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance, fingerprint string) {
	if resource == nil {
		return
	}
	base := stripResourceAnalyticsInstanceCreateOnlyFingerprint(resource.Status.OsokStatus.Message)
	marker := resourceAnalyticsInstanceCreateOnlyFingerprintKey + fingerprint
	if base == "" {
		resource.Status.OsokStatus.Message = marker
		return
	}
	resource.Status.OsokStatus.Message = base + "; " + marker
}

func resourceAnalyticsInstanceCreateOnlyFingerprint(spec resourceanalyticsv1beta1.ResourceAnalyticsInstanceSpec) (string, error) {
	password, err := resourceAnalyticsInstancePasswordSnapshot(spec.AdwAdminPassword)
	if err != nil {
		return "", err
	}
	nsgIDs := slices.Clone(spec.NsgIds)
	slices.Sort(nsgIDs)
	payload := struct {
		CompartmentId       string   `json:"compartmentId"`
		AdwAdminPassword    any      `json:"adwAdminPassword"`
		SubnetId            string   `json:"subnetId"`
		IsMutualTlsRequired bool     `json:"isMutualTlsRequired"`
		NsgIds              []string `json:"nsgIds,omitempty"`
		LicenseModel        string   `json:"licenseModel,omitempty"`
	}{
		CompartmentId:       strings.TrimSpace(spec.CompartmentId),
		AdwAdminPassword:    password,
		SubnetId:            strings.TrimSpace(spec.SubnetId),
		IsMutualTlsRequired: spec.IsMutualTlsRequired,
		NsgIds:              nsgIDs,
		LicenseModel:        strings.TrimSpace(spec.LicenseModel),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal %s create-only fingerprint: %w", resourceAnalyticsInstanceKind, err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func resourceAnalyticsInstancePasswordSnapshot(
	spec resourceanalyticsv1beta1.ResourceAnalyticsInstanceAdwAdminPassword,
) (any, error) {
	if strings.TrimSpace(spec.JsonData) != "" {
		var decoded resourceanalyticsv1beta1.ResourceAnalyticsInstanceAdwAdminPassword
		if err := json.Unmarshal([]byte(spec.JsonData), &decoded); err != nil {
			return nil, fmt.Errorf("decode %s spec.adwAdminPassword.jsonData: %w", resourceAnalyticsInstanceKind, err)
		}
		return resourceAnalyticsInstancePasswordSnapshot(decoded)
	}
	passwordType := strings.ToUpper(strings.TrimSpace(spec.PasswordType))
	switch {
	case passwordType == "" && spec.Password != "":
		passwordType = string(resourceanalyticssdk.AdwAdminPasswordDetailsPasswordTypePlainText)
	case passwordType == "" && spec.SecretId != "":
		passwordType = string(resourceanalyticssdk.AdwAdminPasswordDetailsPasswordTypeVaultSecret)
	}
	return struct {
		PasswordType string `json:"passwordType"`
		Password     string `json:"password,omitempty"`
		SecretId     string `json:"secretId,omitempty"`
	}{
		PasswordType: passwordType,
		Password:     spec.Password,
		SecretId:     strings.TrimSpace(spec.SecretId),
	}, nil
}

func resourceAnalyticsInstanceRecordedCreateOnlyFingerprint(resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance) (string, bool) {
	if resource == nil {
		return "", false
	}
	raw := resource.Status.OsokStatus.Message
	index := strings.LastIndex(raw, resourceAnalyticsInstanceCreateOnlyFingerprintKey)
	if index < 0 {
		return "", false
	}
	start := index + len(resourceAnalyticsInstanceCreateOnlyFingerprintKey)
	end := start
	for end < len(raw) && resourceAnalyticsInstanceIsHexDigit(raw[end]) {
		end++
	}
	fingerprint := raw[start:end]
	if len(fingerprint) != sha256.Size*2 {
		return "", false
	}
	if _, err := hex.DecodeString(fingerprint); err != nil {
		return "", false
	}
	return fingerprint, true
}

func stripResourceAnalyticsInstanceCreateOnlyFingerprint(raw string) string {
	raw = strings.TrimSpace(raw)
	index := strings.LastIndex(raw, resourceAnalyticsInstanceCreateOnlyFingerprintKey)
	if index < 0 {
		return raw
	}
	prefix := strings.TrimSpace(strings.TrimRight(raw[:index], "; "))
	start := index + len(resourceAnalyticsInstanceCreateOnlyFingerprintKey)
	end := start
	for end < len(raw) && resourceAnalyticsInstanceIsHexDigit(raw[end]) {
		end++
	}
	suffix := strings.TrimSpace(strings.TrimLeft(raw[end:], "; "))
	switch {
	case prefix == "":
		return suffix
	case suffix == "":
		return prefix
	default:
		return prefix + "; " + suffix
	}
}

func resourceAnalyticsInstanceIsHexDigit(value byte) bool {
	return ('0' <= value && value <= '9') ||
		('a' <= value && value <= 'f') ||
		('A' <= value && value <= 'F')
}

func wrapResourceAnalyticsInstanceDeleteSafety(
	client resourceAnalyticsInstanceOCIClient,
	initErr error,
) func(ResourceAnalyticsInstanceServiceClient) ResourceAnalyticsInstanceServiceClient {
	return func(delegate ResourceAnalyticsInstanceServiceClient) ResourceAnalyticsInstanceServiceClient {
		return resourceAnalyticsInstanceDeleteSafetyClient{
			delegate: delegate,
			client:   client,
			initErr:  initErr,
		}
	}
}

type resourceAnalyticsInstanceDeleteSafetyClient struct {
	delegate ResourceAnalyticsInstanceServiceClient
	client   resourceAnalyticsInstanceOCIClient
	initErr  error
}

func (c resourceAnalyticsInstanceDeleteSafetyClient) CreateOrUpdate(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c resourceAnalyticsInstanceDeleteSafetyClient) Delete(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
) (bool, error) {
	if deleted, handled, err := c.handlePendingWriteWorkRequest(ctx, resource); handled {
		return deleted, err
	}
	if err := c.rejectAuthShapedCompletedDelete(ctx, resource); err != nil {
		return false, err
	}
	if err := c.rejectAuthShapedPreDeleteConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c resourceAnalyticsInstanceDeleteSafetyClient) handlePendingWriteWorkRequest(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
) (bool, bool, error) {
	current, ok := c.pendingWriteWorkRequest(resource)
	if !ok {
		return false, false, nil
	}
	workRequest, err := c.fetchDeleteSafetyWorkRequest(ctx, current.WorkRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, true, err
	}
	return resourceAnalyticsInstancePendingWriteDecision(current, workRequest)
}

func (c resourceAnalyticsInstanceDeleteSafetyClient) pendingWriteWorkRequest(
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
) (*shared.OSOKAsyncOperation, bool) {
	if c.initErr != nil || c.client == nil || resource == nil {
		return nil, false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Source != shared.OSOKAsyncSourceWorkRequest {
		return nil, false
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
		return current, true
	default:
		return nil, false
	}
}

func resourceAnalyticsInstancePendingWriteDecision(
	current *shared.OSOKAsyncOperation,
	workRequest resourceanalyticssdk.WorkRequest,
) (bool, bool, error) {
	class, err := resourceAnalyticsInstanceWorkRequestAsyncAdapter.Normalize(string(workRequest.Status))
	if err != nil {
		return false, true, err
	}
	if class == shared.OSOKAsyncClassPending {
		return false, true, nil
	}
	if class != shared.OSOKAsyncClassSucceeded {
		return false, true, fmt.Errorf("%s %s work request %s finished with status %s; refusing delete until the write is resolved", resourceAnalyticsInstanceKind, current.Phase, current.WorkRequestID, workRequest.Status)
	}
	return false, false, nil
}

func (c resourceAnalyticsInstanceDeleteSafetyClient) rejectAuthShapedCompletedDelete(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
) error {
	if c.initErr != nil || c.client == nil || resource == nil {
		return nil
	}
	current := resource.Status.OsokStatus.Async.Current
	if current == nil || current.Source != shared.OSOKAsyncSourceWorkRequest || current.Phase != shared.OSOKAsyncPhaseDelete {
		return nil
	}
	workRequest, err := c.fetchDeleteSafetyWorkRequest(ctx, current.WorkRequestID)
	if err != nil || !resourceAnalyticsInstanceDeleteWorkRequestSucceeded(workRequest) {
		return nil
	}
	return c.rejectAuthShapedConfirmRead(
		ctx,
		resource,
		"delete work request completed but confirmation returned authorization-shaped not found; refusing to confirm deletion",
	)
}

func (c resourceAnalyticsInstanceDeleteSafetyClient) rejectAuthShapedPreDeleteConfirmRead(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
) error {
	return c.rejectAuthShapedConfirmRead(
		ctx,
		resource,
		"pre-delete confirmation returned authorization-shaped not found; refusing to call delete",
	)
}

func (c resourceAnalyticsInstanceDeleteSafetyClient) fetchDeleteSafetyWorkRequest(
	ctx context.Context,
	workRequestID string,
) (resourceanalyticssdk.WorkRequest, error) {
	return getResourceAnalyticsInstanceWorkRequest(ctx, c.client, c.initErr, workRequestID)
}

func (c resourceAnalyticsInstanceDeleteSafetyClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance,
	reason string,
) error {
	if c.initErr != nil || c.client == nil || resource == nil {
		return nil
	}
	resourceID := trackedResourceAnalyticsInstanceID(resource)
	if resourceID == "" {
		return nil
	}
	_, err := c.client.GetResourceAnalyticsInstance(ctx, resourceanalyticssdk.GetResourceAnalyticsInstanceRequest{
		ResourceAnalyticsInstanceId: common.String(resourceID),
	})
	if err == nil || !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return fmt.Errorf("%s %s: %w", resourceAnalyticsInstanceKind, reason, err)
}

func resourceAnalyticsInstanceDeleteWorkRequestSucceeded(workRequest resourceanalyticssdk.WorkRequest) bool {
	if workRequest.OperationType != resourceanalyticssdk.OperationTypeDeleteResourceAnalyticsInstance {
		return false
	}
	class, err := resourceAnalyticsInstanceWorkRequestAsyncAdapter.Normalize(string(workRequest.Status))
	return err == nil && class == shared.OSOKAsyncClassSucceeded
}

func handleResourceAnalyticsInstanceDeleteError(resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance, err error) error {
	if err == nil {
		return nil
	}
	classification := errorutil.ClassifyDeleteError(err)
	if !classification.IsAuthShapedNotFound() {
		return err
	}
	requestID := errorutil.OpcRequestID(err)
	if resource != nil {
		servicemanager.SetOpcRequestID(&resource.Status.OsokStatus, requestID)
	}
	return resourceAnalyticsInstanceAmbiguousNotFoundError{
		message:      "resource analytics instance delete returned NotAuthorizedOrNotFound; keeping the finalizer until deletion is unambiguously confirmed",
		opcRequestID: requestID,
	}
}

func trackedResourceAnalyticsInstanceID(resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func resourceAnalyticsInstanceHasTrackedIdentity(resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance) bool {
	return trackedResourceAnalyticsInstanceID(resource) != ""
}

func resourceAnalyticsInstanceHasEstablishedTrackedIdentity(resource *resourceanalyticsv1beta1.ResourceAnalyticsInstance) bool {
	return resourceAnalyticsInstanceHasTrackedIdentity(resource) &&
		resource != nil &&
		resource.Status.OsokStatus.CreatedAt != nil
}

func resourceAnalyticsInstanceDefinedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		tagValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			tagValues[key] = value
		}
		converted[namespace] = tagValues
	}
	return converted
}

func resourceAnalyticsInstanceDefinedTagsFromStatus(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	return resourceAnalyticsInstanceDefinedTagsFromSpec(tags)
}

func cloneResourceAnalyticsInstanceDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	clone := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		tagValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			tagValues[key] = value
		}
		clone[namespace] = tagValues
	}
	return clone
}

func resourceAnalyticsInstanceJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func resourceAnalyticsInstanceStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
