/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	selfsdk "github.com/oracle/oci-go-sdk/v65/self"
	selfv1beta1 "github.com/oracle/oci-service-operator/api/self/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

const subscriptionKind = "Subscription"

var subscriptionWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(selfsdk.OperationStatusAccepted),
		string(selfsdk.OperationStatusInProgress),
		string(selfsdk.OperationStatusWaiting),
		string(selfsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(selfsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(selfsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(selfsdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(selfsdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(selfsdk.OperationTypeCreateSubscription)},
	UpdateActionTokens:    []string{string(selfsdk.OperationTypeUpdateSubscription)},
	DeleteActionTokens:    []string{string(selfsdk.OperationTypeDeleteSubscription)},
}

type subscriptionOCIClient interface {
	CreateSubscription(context.Context, selfsdk.CreateSubscriptionRequest) (selfsdk.CreateSubscriptionResponse, error)
	GetSubscription(context.Context, selfsdk.GetSubscriptionRequest) (selfsdk.GetSubscriptionResponse, error)
	ListSubscriptions(context.Context, selfsdk.ListSubscriptionsRequest) (selfsdk.ListSubscriptionsResponse, error)
	UpdateSubscription(context.Context, selfsdk.UpdateSubscriptionRequest) (selfsdk.UpdateSubscriptionResponse, error)
	DeleteSubscription(context.Context, selfsdk.DeleteSubscriptionRequest) (selfsdk.DeleteSubscriptionResponse, error)
	GetWorkRequest(context.Context, selfsdk.GetWorkRequestRequest) (selfsdk.GetWorkRequestResponse, error)
}

type subscriptionIdentity struct {
	compartmentID string
	displayName   string
	tenantID      string
	sellerID      string
	productID     string
	createDetails selfsdk.CreateSubscriptionDetails
}

type observedSubscriptionState struct {
	displayName  string
	freeformTags map[string]string
	definedTags  map[string]map[string]interface{}
}

func init() {
	registerSubscriptionRuntimeHooksMutator(func(manager *SubscriptionServiceManager, hooks *SubscriptionRuntimeHooks) {
		client, initErr := newSubscriptionSDKClient(manager)
		applySubscriptionRuntimeHooks(hooks, client, initErr)
	})
}

func newSubscriptionSDKClient(manager *SubscriptionServiceManager) (subscriptionOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", subscriptionKind)
	}
	client, err := selfsdk.NewSubscriptionClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applySubscriptionRuntimeHooks(
	hooks *SubscriptionRuntimeHooks,
	client subscriptionOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedSubscriptionRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *selfv1beta1.Subscription, _ string) (any, error) {
		return buildSubscriptionCreateBody(resource)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *selfv1beta1.Subscription,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildSubscriptionUpdateBody(resource, currentResponse)
	}

	if client != nil {
		hooks.Create.Call = func(ctx context.Context, request selfsdk.CreateSubscriptionRequest) (selfsdk.CreateSubscriptionResponse, error) {
			return client.CreateSubscription(ctx, request)
		}
		hooks.Get.Call = func(ctx context.Context, request selfsdk.GetSubscriptionRequest) (selfsdk.GetSubscriptionResponse, error) {
			return client.GetSubscription(ctx, request)
		}
		hooks.List.Call = func(ctx context.Context, request selfsdk.ListSubscriptionsRequest) (selfsdk.ListSubscriptionsResponse, error) {
			return client.ListSubscriptions(ctx, request)
		}
		hooks.Update.Call = func(ctx context.Context, request selfsdk.UpdateSubscriptionRequest) (selfsdk.UpdateSubscriptionResponse, error) {
			return client.UpdateSubscription(ctx, request)
		}
		hooks.Delete.Call = func(ctx context.Context, request selfsdk.DeleteSubscriptionRequest) (selfsdk.DeleteSubscriptionResponse, error) {
			return client.DeleteSubscription(ctx, request)
		}
	}

	if hooks.Create.Call != nil {
		hooks.Create.Call = normalizeSubscriptionCreateCall(hooks.Create.Call)
	}
	if hooks.Get.Call != nil {
		hooks.Get.Call = normalizeSubscriptionGetCall(hooks.Get.Call)
	}

	hooks.List.Fields = subscriptionListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = normalizeSubscriptionListCall(listSubscriptionsAllPages(hooks.List.Call))
	}

	hooks.Identity.Resolve = resolveSubscriptionIdentity
	hooks.Identity.GuardExistingBeforeCreate = guardSubscriptionExistingBeforeCreate
	hooks.Identity.LookupExisting = func(
		ctx context.Context,
		resource *selfv1beta1.Subscription,
		identity any,
	) (any, error) {
		return lookupExistingSubscription(ctx, resource, identity, hooks.List.Call, hooks.Get.Call)
	}

	hooks.Async.Adapter = subscriptionWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getSubscriptionWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveSubscriptionGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveSubscriptionGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverSubscriptionIDFromGeneratedWorkRequest
	hooks.Async.Message = subscriptionGeneratedWorkRequestMessage
}

func reviewedSubscriptionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "self",
		FormalSlug:    "subscription",
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
			ProvisioningStates: []string{
				string(selfsdk.LifecycleStateEnumInactive),
				string(selfsdk.LifecycleDetailsEnumCreated),
				string(selfsdk.LifecycleDetailsEnumPendingActivation),
				string(selfsdk.LifecycleDetailsEnumProvisioningStarted),
				string(selfsdk.LifecycleDetailsEnumProvisioningCompleted),
			},
			UpdatingStates: []string{string(selfsdk.LifecycleDetailsEnumUpdating)},
			ActiveStates:   []string{string(selfsdk.LifecycleStateEnumActive), string(selfsdk.LifecycleDetailsEnumActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(selfsdk.LifecycleDetailsEnumDeleting)},
			TerminalStates: []string{string(selfsdk.LifecycleStateEnumDeleted), string(selfsdk.LifecycleDetailsEnumDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"displayName", "freeformTags", "definedTags"},
			Mutable:         []string{"displayName", "freeformTags", "definedTags"},
			ForceNew: []string{
				"compartmentId",
				"tenantId",
				"subscriptionDetails",
				"sellerId",
				"productId",
				"sourceType",
				"additionalDetails",
				"realm",
				"region",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetSubscription",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetSubscription",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetSubscription/ListSubscriptions confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func buildSubscriptionCreateBody(resource *selfv1beta1.Subscription) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("subscription resource is nil")
	}
	return convertThroughJSON[selfsdk.CreateSubscriptionDetails](resource.Spec)
}

func buildSubscriptionUpdateBody(resource *selfv1beta1.Subscription, currentResponse any) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("subscription resource is nil")
	}

	desired, err := convertThroughJSON[selfsdk.UpdateSubscriptionDetails](resource.Spec)
	if err != nil {
		return nil, false, err
	}
	current, err := observedSubscriptionStateFromAny(currentResponse)
	if err != nil {
		return nil, false, err
	}

	update := selfsdk.UpdateSubscriptionDetails{}
	updateNeeded := false

	if desired.DisplayName != nil && strings.TrimSpace(*desired.DisplayName) != "" && strings.TrimSpace(*desired.DisplayName) != current.displayName {
		update.DisplayName = desired.DisplayName
		updateNeeded = true
	}
	if resource.Spec.FreeformTags != nil && !jsonEqual(desired.FreeformTags, current.freeformTags) {
		update.FreeformTags = desired.FreeformTags
		updateNeeded = true
	}
	if resource.Spec.DefinedTags != nil && !jsonEqual(desired.DefinedTags, current.definedTags) {
		update.DefinedTags = desired.DefinedTags
		updateNeeded = true
	}

	return update, updateNeeded, nil
}

func resolveSubscriptionIdentity(resource *selfv1beta1.Subscription) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("subscription resource is nil")
	}
	createDetails, err := convertThroughJSON[selfsdk.CreateSubscriptionDetails](resource.Spec)
	if err != nil {
		return nil, err
	}
	return subscriptionIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		displayName:   strings.TrimSpace(resource.Spec.DisplayName),
		tenantID:      strings.TrimSpace(resource.Spec.TenantId),
		sellerID:      strings.TrimSpace(resource.Spec.SellerId),
		productID:     strings.TrimSpace(resource.Spec.ProductId),
		createDetails: createDetails,
	}, nil
}

func guardSubscriptionExistingBeforeCreate(_ context.Context, resource *selfv1beta1.Subscription) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("subscription resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func lookupExistingSubscription(
	ctx context.Context,
	_ *selfv1beta1.Subscription,
	identity any,
	listCall func(context.Context, selfsdk.ListSubscriptionsRequest) (selfsdk.ListSubscriptionsResponse, error),
	getCall func(context.Context, selfsdk.GetSubscriptionRequest) (selfsdk.GetSubscriptionResponse, error),
) (any, error) {
	resolved, ok := identity.(subscriptionIdentity)
	if !ok {
		return nil, fmt.Errorf("unexpected subscription identity type %T", identity)
	}
	if listCall == nil || getCall == nil {
		return nil, nil
	}

	listRequest := selfsdk.ListSubscriptionsRequest{
		CompartmentId: common.String(resolved.compartmentID),
		DisplayName:   common.String(resolved.displayName),
	}
	listResponse, err := listCall(ctx, listRequest)
	if err != nil {
		return nil, err
	}

	var matched *selfsdk.GetSubscriptionResponse
	for _, candidate := range listResponse.Items {
		if !subscriptionSummaryMatchesIdentity(candidate, resolved) {
			continue
		}
		if candidate.Id == nil || strings.TrimSpace(*candidate.Id) == "" {
			continue
		}
		currentResponse, err := getCall(ctx, selfsdk.GetSubscriptionRequest{SubscriptionId: candidate.Id})
		if err != nil {
			if isSubscriptionReadNotFound(err) {
				continue
			}
			return nil, err
		}
		if !subscriptionMatchesIdentity(currentResponse.Subscription, resolved) {
			continue
		}
		if matched != nil && !strings.EqualFold(stringValue(matched.Subscription.Id), stringValue(currentResponse.Subscription.Id)) {
			return nil, fmt.Errorf("%s existing-before-create lookup returned multiple matching resources", subscriptionKind)
		}
		currentCopy := currentResponse
		matched = &currentCopy
	}

	if matched == nil {
		return nil, nil
	}
	return *matched, nil
}

func subscriptionSummaryMatchesIdentity(summary selfsdk.SubscriptionSummary, identity subscriptionIdentity) bool {
	if identity.compartmentID != "" && stringValue(summary.CompartmentId) != identity.compartmentID {
		return false
	}
	if identity.displayName != "" && stringValue(summary.DisplayName) != identity.displayName {
		return false
	}
	if identity.tenantID != "" && stringValue(summary.TenantId) != identity.tenantID {
		return false
	}
	if identity.sellerID != "" && stringValue(summary.SellerId) != identity.sellerID {
		return false
	}
	if identity.productID != "" && stringValue(summary.ProductId) != identity.productID {
		return false
	}
	return subscriptionLifecycleReusable(normalizedSubscriptionLifecycleToken(string(summary.LifecycleDetails), string(summary.LifecycleState)))
}

func subscriptionMatchesIdentity(current selfsdk.Subscription, identity subscriptionIdentity) bool {
	if identity.compartmentID != "" && stringValue(current.CompartmentId) != identity.compartmentID {
		return false
	}
	if identity.displayName != "" && stringValue(current.DisplayName) != identity.displayName {
		return false
	}
	if identity.tenantID != "" && stringValue(current.TenantId) != identity.tenantID {
		return false
	}
	if identity.sellerID != "" && stringValue(current.SellerId) != identity.sellerID {
		return false
	}
	if identity.productID != "" && stringValue(current.ProductId) != identity.productID {
		return false
	}
	if identity.createDetails.SubscriptionDetails != nil && !jsonEqual(identity.createDetails.SubscriptionDetails, current.SubscriptionDetails) {
		return false
	}
	if identity.createDetails.SourceType != "" && current.SourceType != identity.createDetails.SourceType {
		return false
	}
	if identity.createDetails.Realm != nil && stringValue(current.Realm) != stringValue(identity.createDetails.Realm) {
		return false
	}
	if identity.createDetails.Region != nil && stringValue(current.Region) != stringValue(identity.createDetails.Region) {
		return false
	}
	if len(identity.createDetails.AdditionalDetails) > 0 && !jsonEqual(identity.createDetails.AdditionalDetails, current.AdditionalDetails) {
		return false
	}
	return subscriptionLifecycleReusable(normalizedSubscriptionLifecycleToken(string(current.LifecycleDetails), string(current.LifecycleState)))
}

func subscriptionLifecycleReusable(token string) bool {
	switch token {
	case string(selfsdk.LifecycleStateEnumActive),
		string(selfsdk.LifecycleDetailsEnumCreated),
		string(selfsdk.LifecycleDetailsEnumPendingActivation),
		string(selfsdk.LifecycleDetailsEnumProvisioningStarted),
		string(selfsdk.LifecycleDetailsEnumProvisioningCompleted),
		string(selfsdk.LifecycleDetailsEnumUpdating):
		return true
	default:
		return false
	}
}

func subscriptionListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func normalizeSubscriptionCreateCall(
	call func(context.Context, selfsdk.CreateSubscriptionRequest) (selfsdk.CreateSubscriptionResponse, error),
) func(context.Context, selfsdk.CreateSubscriptionRequest) (selfsdk.CreateSubscriptionResponse, error) {
	return func(ctx context.Context, request selfsdk.CreateSubscriptionRequest) (selfsdk.CreateSubscriptionResponse, error) {
		response, err := call(ctx, request)
		if err != nil {
			return selfsdk.CreateSubscriptionResponse{}, err
		}
		response.Subscription = normalizeSDKSubscription(response.Subscription)
		return response, nil
	}
}

func normalizeSubscriptionGetCall(
	call func(context.Context, selfsdk.GetSubscriptionRequest) (selfsdk.GetSubscriptionResponse, error),
) func(context.Context, selfsdk.GetSubscriptionRequest) (selfsdk.GetSubscriptionResponse, error) {
	return func(ctx context.Context, request selfsdk.GetSubscriptionRequest) (selfsdk.GetSubscriptionResponse, error) {
		response, err := call(ctx, request)
		if err != nil {
			return selfsdk.GetSubscriptionResponse{}, err
		}
		response.Subscription = normalizeSDKSubscription(response.Subscription)
		return response, nil
	}
}

func normalizeSubscriptionListCall(
	call func(context.Context, selfsdk.ListSubscriptionsRequest) (selfsdk.ListSubscriptionsResponse, error),
) func(context.Context, selfsdk.ListSubscriptionsRequest) (selfsdk.ListSubscriptionsResponse, error) {
	return func(ctx context.Context, request selfsdk.ListSubscriptionsRequest) (selfsdk.ListSubscriptionsResponse, error) {
		response, err := call(ctx, request)
		if err != nil {
			return selfsdk.ListSubscriptionsResponse{}, err
		}
		for i := range response.Items {
			response.Items[i] = normalizeSDKSubscriptionSummary(response.Items[i])
		}
		return response, nil
	}
}

func normalizeSDKSubscription(current selfsdk.Subscription) selfsdk.Subscription {
	if effective := normalizedSubscriptionLifecycleToken(string(current.LifecycleDetails), string(current.LifecycleState)); effective != "" {
		current.LifecycleState = selfsdk.LifecycleStateEnumEnum(effective)
	}
	return current
}

func normalizeSDKSubscriptionSummary(current selfsdk.SubscriptionSummary) selfsdk.SubscriptionSummary {
	if effective := normalizedSubscriptionLifecycleToken(string(current.LifecycleDetails), string(current.LifecycleState)); effective != "" {
		current.LifecycleState = selfsdk.LifecycleStateEnumEnum(effective)
	}
	return current
}

func normalizedSubscriptionLifecycleToken(lifecycleDetails string, lifecycleState string) string {
	detail := strings.ToUpper(strings.TrimSpace(lifecycleDetails))
	switch detail {
	case string(selfsdk.LifecycleDetailsEnumCreated),
		string(selfsdk.LifecycleDetailsEnumPendingActivation),
		string(selfsdk.LifecycleDetailsEnumProvisioningStarted),
		string(selfsdk.LifecycleDetailsEnumProvisioningCompleted),
		string(selfsdk.LifecycleDetailsEnumProvisioningFailed),
		string(selfsdk.LifecycleDetailsEnumActive),
		string(selfsdk.LifecycleDetailsEnumExpired),
		string(selfsdk.LifecycleDetailsEnumTerminated),
		string(selfsdk.LifecycleDetailsEnumFailed),
		string(selfsdk.LifecycleDetailsEnumDeleting),
		string(selfsdk.LifecycleDetailsEnumUpdating),
		string(selfsdk.LifecycleDetailsEnumDeleted):
		return detail
	}
	return strings.ToUpper(strings.TrimSpace(lifecycleState))
}

func listSubscriptionsAllPages(
	call func(context.Context, selfsdk.ListSubscriptionsRequest) (selfsdk.ListSubscriptionsResponse, error),
) func(context.Context, selfsdk.ListSubscriptionsRequest) (selfsdk.ListSubscriptionsResponse, error) {
	return func(ctx context.Context, request selfsdk.ListSubscriptionsRequest) (selfsdk.ListSubscriptionsResponse, error) {
		var combined selfsdk.ListSubscriptionsResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return selfsdk.ListSubscriptionsResponse{}, err
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
}

func getSubscriptionWorkRequest(
	ctx context.Context,
	client subscriptionOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize %s OCI client: %w", subscriptionKind, initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("%s OCI client is not configured", subscriptionKind)
	}

	response, err := client.GetWorkRequest(ctx, selfsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveSubscriptionGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := subscriptionWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveSubscriptionGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := subscriptionWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case selfsdk.OperationTypeCreateSubscription:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case selfsdk.OperationTypeUpdateSubscription:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case selfsdk.OperationTypeDeleteSubscription:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func recoverSubscriptionIDFromGeneratedWorkRequest(
	_ *selfv1beta1.Subscription,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := subscriptionWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveSubscriptionIDFromWorkRequest(current, subscriptionWorkRequestActionForPhase(phase))
}

func subscriptionGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := subscriptionWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	workRequestID := stringValue(current.Id)
	rawStatus := string(current.Status)
	switch {
	case phase != "" && workRequestID != "" && rawStatus != "":
		return fmt.Sprintf("%s %s work request %s is %s", subscriptionKind, phase, workRequestID, rawStatus)
	case phase != "" && rawStatus != "":
		return fmt.Sprintf("%s %s work request is %s", subscriptionKind, phase, rawStatus)
	default:
		return ""
	}
}

func subscriptionWorkRequestFromAny(workRequest any) (selfsdk.WorkRequest, error) {
	switch typed := workRequest.(type) {
	case selfsdk.WorkRequest:
		return typed, nil
	case *selfsdk.WorkRequest:
		if typed == nil {
			return selfsdk.WorkRequest{}, fmt.Errorf("%s work request is nil", subscriptionKind)
		}
		return *typed, nil
	case selfsdk.GetWorkRequestResponse:
		return typed.WorkRequest, nil
	case *selfsdk.GetWorkRequestResponse:
		if typed == nil {
			return selfsdk.WorkRequest{}, fmt.Errorf("%s work request response is nil", subscriptionKind)
		}
		return typed.WorkRequest, nil
	default:
		return selfsdk.WorkRequest{}, fmt.Errorf("unexpected %s work request type %T", subscriptionKind, workRequest)
	}
}

func subscriptionWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) selfsdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return selfsdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return selfsdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return selfsdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveSubscriptionIDFromWorkRequest(workRequest selfsdk.WorkRequest, action selfsdk.ActionTypeEnum) (string, error) {
	if id, ok := resolveSubscriptionIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveSubscriptionIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("%s work request %s does not expose a subscription identifier", subscriptionKind, stringValue(workRequest.Id))
}

func resolveSubscriptionIDFromResources(
	resources []selfsdk.WorkRequestResource,
	action selfsdk.ActionTypeEnum,
	preferSubscriptionOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		entityType := strings.ToUpper(strings.TrimSpace(stringValue(resource.EntityType)))
		if preferSubscriptionOnly && !strings.Contains(entityType, "SUBSCRIPTION") {
			continue
		}
		identifier := strings.TrimSpace(stringValue(resource.Identifier))
		if identifier == "" {
			continue
		}
		if candidate == "" {
			candidate = identifier
			continue
		}
		if candidate != identifier {
			return "", false
		}
	}
	return candidate, candidate != ""
}

func observedSubscriptionStateFromAny(currentResponse any) (observedSubscriptionState, error) {
	switch typed := currentResponse.(type) {
	case selfsdk.Subscription:
		return observedSubscriptionState{
			displayName:  stringValue(typed.DisplayName),
			freeformTags: cloneStringMap(typed.FreeformTags),
			definedTags:  cloneDefinedTags(typed.DefinedTags),
		}, nil
	case *selfsdk.Subscription:
		if typed == nil {
			return observedSubscriptionState{}, fmt.Errorf("subscription response is nil")
		}
		return observedSubscriptionStateFromAny(*typed)
	case selfsdk.GetSubscriptionResponse:
		return observedSubscriptionStateFromAny(typed.Subscription)
	case *selfsdk.GetSubscriptionResponse:
		if typed == nil {
			return observedSubscriptionState{}, fmt.Errorf("subscription get response is nil")
		}
		return observedSubscriptionStateFromAny(typed.Subscription)
	case selfsdk.SubscriptionSummary:
		return observedSubscriptionState{
			displayName:  stringValue(typed.DisplayName),
			freeformTags: cloneStringMap(typed.FreeformTags),
			definedTags:  cloneDefinedTags(typed.DefinedTags),
		}, nil
	case *selfsdk.SubscriptionSummary:
		if typed == nil {
			return observedSubscriptionState{}, fmt.Errorf("subscription summary is nil")
		}
		return observedSubscriptionStateFromAny(*typed)
	default:
		return observedSubscriptionState{}, fmt.Errorf("unexpected subscription current response type %T", currentResponse)
	}
}

func newSubscriptionServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client subscriptionOCIClient,
) SubscriptionServiceClient {
	manager := &SubscriptionServiceManager{Log: log}
	hooks := newSubscriptionDefaultRuntimeHooks(selfsdk.SubscriptionClient{})
	applySubscriptionRuntimeHooks(&hooks, client, nil)
	delegate := defaultSubscriptionServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*selfv1beta1.Subscription](
			buildSubscriptionGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapSubscriptionGeneratedClient(hooks, delegate)
}

func convertThroughJSON[To any](input any) (To, error) {
	var output To
	payload, err := json.Marshal(input)
	if err != nil {
		return output, fmt.Errorf("marshal subscription payload: %w", err)
	}
	if err := json.Unmarshal(payload, &output); err != nil {
		return output, fmt.Errorf("unmarshal subscription payload: %w", err)
	}
	return output, nil
}

func jsonEqual(left any, right any) bool {
	leftPayload, err := json.Marshal(left)
	if err != nil {
		return false
	}
	rightPayload, err := json.Marshal(right)
	if err != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		if values == nil {
			cloned[namespace] = nil
			continue
		}
		namespaceCopy := make(map[string]interface{}, len(values))
		for key, value := range values {
			namespaceCopy[key] = value
		}
		cloned[namespace] = namespaceCopy
	}
	return cloned
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func isSubscriptionReadNotFound(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}
