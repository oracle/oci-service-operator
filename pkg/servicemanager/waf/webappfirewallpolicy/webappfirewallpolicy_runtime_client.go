/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package webappfirewallpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	wafsdk "github.com/oracle/oci-go-sdk/v65/waf"
	wafv1beta1 "github.com/oracle/oci-service-operator/api/waf/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

const webAppFirewallPolicyWorkRequestEntityType = "webAppFirewallPolicy"

var webAppFirewallPolicyWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(wafsdk.WorkRequestStatusAccepted),
		string(wafsdk.WorkRequestStatusInProgress),
		string(wafsdk.WorkRequestStatusCanceling),
	},
	SucceededStatusTokens: []string{string(wafsdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(wafsdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(wafsdk.WorkRequestStatusCanceled)},
	CreateActionTokens: []string{
		string(wafsdk.WorkRequestResourceActionTypeCreated),
		string(wafsdk.WorkRequestOperationTypeCreateWafPolicy),
	},
	UpdateActionTokens: []string{
		string(wafsdk.WorkRequestResourceActionTypeUpdated),
		string(wafsdk.WorkRequestOperationTypeUpdateWafPolicy),
		string(wafsdk.WorkRequestOperationTypeMoveWafPolicy),
	},
	DeleteActionTokens: []string{
		string(wafsdk.WorkRequestResourceActionTypeDeleted),
		string(wafsdk.WorkRequestOperationTypeDeleteWafPolicy),
	},
}

type webAppFirewallPolicyOCIClient interface {
	ChangeWebAppFirewallPolicyCompartment(context.Context, wafsdk.ChangeWebAppFirewallPolicyCompartmentRequest) (wafsdk.ChangeWebAppFirewallPolicyCompartmentResponse, error)
	CreateWebAppFirewallPolicy(context.Context, wafsdk.CreateWebAppFirewallPolicyRequest) (wafsdk.CreateWebAppFirewallPolicyResponse, error)
	GetWebAppFirewallPolicy(context.Context, wafsdk.GetWebAppFirewallPolicyRequest) (wafsdk.GetWebAppFirewallPolicyResponse, error)
	ListWebAppFirewallPolicies(context.Context, wafsdk.ListWebAppFirewallPoliciesRequest) (wafsdk.ListWebAppFirewallPoliciesResponse, error)
	UpdateWebAppFirewallPolicy(context.Context, wafsdk.UpdateWebAppFirewallPolicyRequest) (wafsdk.UpdateWebAppFirewallPolicyResponse, error)
	DeleteWebAppFirewallPolicy(context.Context, wafsdk.DeleteWebAppFirewallPolicyRequest) (wafsdk.DeleteWebAppFirewallPolicyResponse, error)
	GetWorkRequest(context.Context, wafsdk.GetWorkRequestRequest) (wafsdk.GetWorkRequestResponse, error)
}

type webAppFirewallPolicyPendingWriteDeleteClient struct {
	delegate                WebAppFirewallPolicyServiceClient
	getWebAppFirewallPolicy func(context.Context, wafsdk.GetWebAppFirewallPolicyRequest) (wafsdk.GetWebAppFirewallPolicyResponse, error)
}

type ambiguousWebAppFirewallPolicyNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousWebAppFirewallPolicyNotFoundError) Error() string {
	return e.message
}

func (e ambiguousWebAppFirewallPolicyNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerWebAppFirewallPolicyRuntimeHooksMutator(func(manager *WebAppFirewallPolicyServiceManager, hooks *WebAppFirewallPolicyRuntimeHooks) {
		client, initErr := newWebAppFirewallPolicySDKClient(manager)
		applyWebAppFirewallPolicyRuntimeHooks(hooks, client, initErr)
	})
}

func newWebAppFirewallPolicySDKClient(manager *WebAppFirewallPolicyServiceManager) (webAppFirewallPolicyOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("WebAppFirewallPolicy service manager is nil")
	}
	client, err := wafsdk.NewWafClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyWebAppFirewallPolicyRuntimeHooks(
	hooks *WebAppFirewallPolicyRuntimeHooks,
	client webAppFirewallPolicyOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = webAppFirewallPolicyRuntimeSemantics()
	hooks.BuildCreateBody = buildWebAppFirewallPolicyCreateBody
	hooks.BuildUpdateBody = buildWebAppFirewallPolicyUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardWebAppFirewallPolicyExistingBeforeCreate
	hooks.Create.Fields = webAppFirewallPolicyCreateFields()
	hooks.Create.Call = func(ctx context.Context, request wafsdk.CreateWebAppFirewallPolicyRequest) (wafsdk.CreateWebAppFirewallPolicyResponse, error) {
		if err := webAppFirewallPolicyClientReady(client, initErr); err != nil {
			return wafsdk.CreateWebAppFirewallPolicyResponse{}, err
		}
		return client.CreateWebAppFirewallPolicy(ctx, request)
	}
	hooks.Get.Fields = webAppFirewallPolicyGetFields()
	hooks.Get.Call = func(ctx context.Context, request wafsdk.GetWebAppFirewallPolicyRequest) (wafsdk.GetWebAppFirewallPolicyResponse, error) {
		if err := webAppFirewallPolicyClientReady(client, initErr); err != nil {
			return wafsdk.GetWebAppFirewallPolicyResponse{}, err
		}
		response, err := client.GetWebAppFirewallPolicy(ctx, request)
		return response, conservativeWebAppFirewallPolicyNotFoundError(err, "read")
	}
	hooks.List.Fields = webAppFirewallPolicyListFields()
	hooks.List.Call = func(ctx context.Context, request wafsdk.ListWebAppFirewallPoliciesRequest) (wafsdk.ListWebAppFirewallPoliciesResponse, error) {
		return listWebAppFirewallPolicyPages(ctx, client, initErr, request)
	}
	hooks.Update.Fields = webAppFirewallPolicyUpdateFields()
	hooks.Update.Call = func(ctx context.Context, request wafsdk.UpdateWebAppFirewallPolicyRequest) (wafsdk.UpdateWebAppFirewallPolicyResponse, error) {
		if err := webAppFirewallPolicyClientReady(client, initErr); err != nil {
			return wafsdk.UpdateWebAppFirewallPolicyResponse{}, err
		}
		return client.UpdateWebAppFirewallPolicy(ctx, request)
	}
	hooks.Delete.Fields = webAppFirewallPolicyDeleteFields()
	hooks.Delete.Call = func(ctx context.Context, request wafsdk.DeleteWebAppFirewallPolicyRequest) (wafsdk.DeleteWebAppFirewallPolicyResponse, error) {
		if err := webAppFirewallPolicyClientReady(client, initErr); err != nil {
			return wafsdk.DeleteWebAppFirewallPolicyResponse{}, err
		}
		response, err := client.DeleteWebAppFirewallPolicy(ctx, request)
		return response, conservativeWebAppFirewallPolicyNotFoundError(err, "delete")
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedWebAppFirewallPolicyIdentity
	hooks.StatusHooks.ProjectStatus = projectWebAppFirewallPolicyStatus
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateWebAppFirewallPolicyCreateOnlyDriftForResponse
	hooks.ParityHooks.RequiresParityHandling = webAppFirewallPolicyRequiresCompartmentMove
	hooks.ParityHooks.ApplyParityUpdate = func(
		ctx context.Context,
		resource *wafv1beta1.WebAppFirewallPolicy,
		currentResponse any,
	) (servicemanager.OSOKResponse, error) {
		return applyWebAppFirewallPolicyCompartmentMove(ctx, resource, currentResponse, client, initErr)
	}
	hooks.DeleteHooks.HandleError = handleWebAppFirewallPolicyDeleteError
	hooks.Async.Adapter = webAppFirewallPolicyWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getWebAppFirewallPolicyWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveWebAppFirewallPolicyWorkRequestAction
	hooks.Async.ResolvePhase = resolveWebAppFirewallPolicyWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverWebAppFirewallPolicyIDFromWorkRequest
	hooks.Async.Message = webAppFirewallPolicyWorkRequestMessage
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate WebAppFirewallPolicyServiceClient) WebAppFirewallPolicyServiceClient {
		return webAppFirewallPolicyPendingWriteDeleteClient{
			delegate:                delegate,
			getWebAppFirewallPolicy: hooks.Get.Call,
		}
	})
}

func newWebAppFirewallPolicyServiceClientWithOCIClient(client webAppFirewallPolicyOCIClient) WebAppFirewallPolicyServiceClient {
	manager := &WebAppFirewallPolicyServiceManager{}
	hooks := newWebAppFirewallPolicyRuntimeHooksWithOCIClient(client)
	applyWebAppFirewallPolicyRuntimeHooks(&hooks, client, nil)
	delegate := defaultWebAppFirewallPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*wafv1beta1.WebAppFirewallPolicy](
			buildWebAppFirewallPolicyGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapWebAppFirewallPolicyGeneratedClient(hooks, delegate)
}

func newWebAppFirewallPolicyRuntimeHooksWithOCIClient(client webAppFirewallPolicyOCIClient) WebAppFirewallPolicyRuntimeHooks {
	return WebAppFirewallPolicyRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*wafv1beta1.WebAppFirewallPolicy]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*wafv1beta1.WebAppFirewallPolicy]{},
		StatusHooks:     generatedruntime.StatusHooks[*wafv1beta1.WebAppFirewallPolicy]{},
		ParityHooks:     generatedruntime.ParityHooks[*wafv1beta1.WebAppFirewallPolicy]{},
		Async:           generatedruntime.AsyncHooks[*wafv1beta1.WebAppFirewallPolicy]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*wafv1beta1.WebAppFirewallPolicy]{},
		Create: runtimeOperationHooks[wafsdk.CreateWebAppFirewallPolicyRequest, wafsdk.CreateWebAppFirewallPolicyResponse]{
			Fields: webAppFirewallPolicyCreateFields(),
			Call: func(ctx context.Context, request wafsdk.CreateWebAppFirewallPolicyRequest) (wafsdk.CreateWebAppFirewallPolicyResponse, error) {
				if client == nil {
					return wafsdk.CreateWebAppFirewallPolicyResponse{}, fmt.Errorf("WebAppFirewallPolicy OCI client is not configured")
				}
				return client.CreateWebAppFirewallPolicy(ctx, request)
			},
		},
		Get: runtimeOperationHooks[wafsdk.GetWebAppFirewallPolicyRequest, wafsdk.GetWebAppFirewallPolicyResponse]{
			Fields: webAppFirewallPolicyGetFields(),
			Call: func(ctx context.Context, request wafsdk.GetWebAppFirewallPolicyRequest) (wafsdk.GetWebAppFirewallPolicyResponse, error) {
				if client == nil {
					return wafsdk.GetWebAppFirewallPolicyResponse{}, fmt.Errorf("WebAppFirewallPolicy OCI client is not configured")
				}
				return client.GetWebAppFirewallPolicy(ctx, request)
			},
		},
		List: runtimeOperationHooks[wafsdk.ListWebAppFirewallPoliciesRequest, wafsdk.ListWebAppFirewallPoliciesResponse]{
			Fields: webAppFirewallPolicyListFields(),
			Call: func(ctx context.Context, request wafsdk.ListWebAppFirewallPoliciesRequest) (wafsdk.ListWebAppFirewallPoliciesResponse, error) {
				if client == nil {
					return wafsdk.ListWebAppFirewallPoliciesResponse{}, fmt.Errorf("WebAppFirewallPolicy OCI client is not configured")
				}
				return client.ListWebAppFirewallPolicies(ctx, request)
			},
		},
		Update: runtimeOperationHooks[wafsdk.UpdateWebAppFirewallPolicyRequest, wafsdk.UpdateWebAppFirewallPolicyResponse]{
			Fields: webAppFirewallPolicyUpdateFields(),
			Call: func(ctx context.Context, request wafsdk.UpdateWebAppFirewallPolicyRequest) (wafsdk.UpdateWebAppFirewallPolicyResponse, error) {
				if client == nil {
					return wafsdk.UpdateWebAppFirewallPolicyResponse{}, fmt.Errorf("WebAppFirewallPolicy OCI client is not configured")
				}
				return client.UpdateWebAppFirewallPolicy(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[wafsdk.DeleteWebAppFirewallPolicyRequest, wafsdk.DeleteWebAppFirewallPolicyResponse]{
			Fields: webAppFirewallPolicyDeleteFields(),
			Call: func(ctx context.Context, request wafsdk.DeleteWebAppFirewallPolicyRequest) (wafsdk.DeleteWebAppFirewallPolicyResponse, error) {
				if client == nil {
					return wafsdk.DeleteWebAppFirewallPolicyResponse{}, fmt.Errorf("WebAppFirewallPolicy OCI client is not configured")
				}
				return client.DeleteWebAppFirewallPolicy(ctx, request)
			},
		},
		WrapGeneratedClient: []func(WebAppFirewallPolicyServiceClient) WebAppFirewallPolicyServiceClient{},
	}
}

func webAppFirewallPolicyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "waf",
		FormalSlug:    "webappfirewallpolicy",
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
			ProvisioningStates: []string{string(wafsdk.WebAppFirewallPolicyLifecycleStateCreating)},
			UpdatingStates:     []string{string(wafsdk.WebAppFirewallPolicyLifecycleStateUpdating)},
			ActiveStates:       []string{string(wafsdk.WebAppFirewallPolicyLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(wafsdk.WebAppFirewallPolicyLifecycleStateDeleting),
			},
			TerminalStates: []string{string(wafsdk.WebAppFirewallPolicyLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"id", "compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"compartmentId",
				"displayName",
				"actions",
				"requestAccessControl",
				"requestRateLimiting",
				"requestProtection",
				"responseAccessControl",
				"responseProtection",
				"freeformTags",
				"definedTags",
				"systemTags",
			},
			ForceNew:      []string{},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: webAppFirewallPolicyWorkRequestEntityType, Action: "CREATED"},
			},
			Update: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: webAppFirewallPolicyWorkRequestEntityType, Action: "UPDATED"},
			},
			Delete: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: webAppFirewallPolicyWorkRequestEntityType, Action: "DELETED"},
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: webAppFirewallPolicyWorkRequestEntityType, Action: "CREATED"},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: webAppFirewallPolicyWorkRequestEntityType, Action: "UPDATED"},
			},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: webAppFirewallPolicyWorkRequestEntityType, Action: "DELETED"},
			},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func webAppFirewallPolicyCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateWebAppFirewallPolicyDetails", RequestName: "CreateWebAppFirewallPolicyDetails", Contribution: "body"},
	}
}

func webAppFirewallPolicyGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "WebAppFirewallPolicyId", RequestName: "webAppFirewallPolicyId", Contribution: "path", PreferResourceID: true},
	}
}

func webAppFirewallPolicyListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName", "metadataName"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
	}
}

func webAppFirewallPolicyUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "WebAppFirewallPolicyId", RequestName: "webAppFirewallPolicyId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateWebAppFirewallPolicyDetails", RequestName: "UpdateWebAppFirewallPolicyDetails", Contribution: "body"},
	}
}

func webAppFirewallPolicyDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "WebAppFirewallPolicyId", RequestName: "webAppFirewallPolicyId", Contribution: "path", PreferResourceID: true},
	}
}

func buildWebAppFirewallPolicyCreateBody(
	_ context.Context,
	resource *wafv1beta1.WebAppFirewallPolicy,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("WebAppFirewallPolicy resource is nil")
	}
	if err := validateWebAppFirewallPolicySpec(resource.Spec); err != nil {
		return nil, err
	}

	details := webAppFirewallPolicyCreateDetails(resource.Spec)
	if err := applyWebAppFirewallPolicyCreateActions(&details, resource.Spec.Actions); err != nil {
		return nil, err
	}
	if err := applyWebAppFirewallPolicyCreateRequestAccessControl(&details, resource.Spec.RequestAccessControl); err != nil {
		return nil, err
	}
	if err := applyWebAppFirewallPolicyCreateRequestRateLimiting(&details, resource.Spec.RequestRateLimiting); err != nil {
		return nil, err
	}
	if err := applyWebAppFirewallPolicyCreateRequestProtection(&details, resource.Spec.RequestProtection); err != nil {
		return nil, err
	}
	if err := applyWebAppFirewallPolicyCreateResponseAccessControl(&details, resource.Spec.ResponseAccessControl); err != nil {
		return nil, err
	}
	if err := applyWebAppFirewallPolicyCreateResponseProtection(&details, resource.Spec.ResponseProtection); err != nil {
		return nil, err
	}
	applyWebAppFirewallPolicyCreateTags(&details, resource.Spec)
	return details, nil
}

func webAppFirewallPolicyCreateDetails(spec wafv1beta1.WebAppFirewallPolicySpec) wafsdk.CreateWebAppFirewallPolicyDetails {
	details := wafsdk.CreateWebAppFirewallPolicyDetails{
		CompartmentId: common.String(strings.TrimSpace(spec.CompartmentId)),
	}
	if displayName := strings.TrimSpace(spec.DisplayName); displayName != "" {
		details.DisplayName = common.String(displayName)
	}
	return details
}

func applyWebAppFirewallPolicyCreateActions(
	details *wafsdk.CreateWebAppFirewallPolicyDetails,
	spec []wafv1beta1.WebAppFirewallPolicyAction,
) error {
	if spec == nil {
		return nil
	}
	actions, err := webAppFirewallPolicyActions(spec)
	if err != nil {
		return err
	}
	details.Actions = actions
	return nil
}

func applyWebAppFirewallPolicyCreateRequestAccessControl(
	details *wafsdk.CreateWebAppFirewallPolicyDetails,
	spec wafv1beta1.WebAppFirewallPolicyRequestAccessControl,
) error {
	if !webAppFirewallPolicyRequestAccessControlSpecified(spec) {
		return nil
	}
	module, err := webAppFirewallPolicyRequestAccessControl(spec)
	if err != nil {
		return err
	}
	details.RequestAccessControl = module
	return nil
}

func applyWebAppFirewallPolicyCreateRequestRateLimiting(
	details *wafsdk.CreateWebAppFirewallPolicyDetails,
	spec wafv1beta1.WebAppFirewallPolicyRequestRateLimiting,
) error {
	if spec.Rules == nil {
		return nil
	}
	module, err := webAppFirewallPolicyRequestRateLimiting(spec)
	if err != nil {
		return err
	}
	details.RequestRateLimiting = module
	return nil
}

func applyWebAppFirewallPolicyCreateRequestProtection(
	details *wafsdk.CreateWebAppFirewallPolicyDetails,
	spec wafv1beta1.WebAppFirewallPolicyRequestProtection,
) error {
	if !webAppFirewallPolicyRequestProtectionSpecified(spec) {
		return nil
	}
	module, err := webAppFirewallPolicyRequestProtection(spec)
	if err != nil {
		return err
	}
	details.RequestProtection = module
	return nil
}

func applyWebAppFirewallPolicyCreateResponseAccessControl(
	details *wafsdk.CreateWebAppFirewallPolicyDetails,
	spec wafv1beta1.WebAppFirewallPolicyResponseAccessControl,
) error {
	if spec.Rules == nil {
		return nil
	}
	module, err := webAppFirewallPolicyResponseAccessControl(spec)
	if err != nil {
		return err
	}
	details.ResponseAccessControl = module
	return nil
}

func applyWebAppFirewallPolicyCreateResponseProtection(
	details *wafsdk.CreateWebAppFirewallPolicyDetails,
	spec wafv1beta1.WebAppFirewallPolicyResponseProtection,
) error {
	if spec.Rules == nil {
		return nil
	}
	module, err := webAppFirewallPolicyResponseProtection(spec)
	if err != nil {
		return err
	}
	details.ResponseProtection = module
	return nil
}

func applyWebAppFirewallPolicyCreateTags(
	details *wafsdk.CreateWebAppFirewallPolicyDetails,
	spec wafv1beta1.WebAppFirewallPolicySpec,
) {
	if spec.FreeformTags != nil {
		details.FreeformTags = cloneWebAppFirewallPolicyStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = webAppFirewallPolicyDefinedTagsFromSpec(spec.DefinedTags)
	}
	if spec.SystemTags != nil {
		details.SystemTags = webAppFirewallPolicyDefinedTagsFromSpec(spec.SystemTags)
	}
}

type webAppFirewallPolicyUpdateBuilder struct {
	details wafsdk.UpdateWebAppFirewallPolicyDetails
	needed  bool
}

func buildWebAppFirewallPolicyUpdateBody(
	_ context.Context,
	resource *wafv1beta1.WebAppFirewallPolicy,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return wafsdk.UpdateWebAppFirewallPolicyDetails{}, false, fmt.Errorf("WebAppFirewallPolicy resource is nil")
	}
	if err := validateWebAppFirewallPolicySpec(resource.Spec); err != nil {
		return wafsdk.UpdateWebAppFirewallPolicyDetails{}, false, err
	}
	current, ok := webAppFirewallPolicyFromResponse(currentResponse)
	if !ok {
		return wafsdk.UpdateWebAppFirewallPolicyDetails{}, false, fmt.Errorf("current WebAppFirewallPolicy response does not expose a WebAppFirewallPolicy body")
	}
	if err := validateWebAppFirewallPolicyCreateOnlyDrift(resource, current); err != nil {
		return wafsdk.UpdateWebAppFirewallPolicyDetails{}, false, err
	}

	builder := webAppFirewallPolicyUpdateBuilder{
		needed: webAppFirewallPolicyCompartmentNeedsMove(resource.Spec, current),
	}
	if err := applyWebAppFirewallPolicyUpdateSpec(&builder, resource.Spec, current); err != nil {
		return wafsdk.UpdateWebAppFirewallPolicyDetails{}, false, err
	}
	if !builder.needed {
		return wafsdk.UpdateWebAppFirewallPolicyDetails{}, false, nil
	}
	return builder.details, true, nil
}

func applyWebAppFirewallPolicyUpdateSpec(
	builder *webAppFirewallPolicyUpdateBuilder,
	spec wafv1beta1.WebAppFirewallPolicySpec,
	current wafsdk.WebAppFirewallPolicy,
) error {
	builder.applyDisplayName(spec, current)
	if err := builder.applyActions(spec, current); err != nil {
		return err
	}
	if err := builder.applyRequestAccessControl(spec, current); err != nil {
		return err
	}
	if err := builder.applyRequestRateLimiting(spec, current); err != nil {
		return err
	}
	if err := builder.applyRequestProtection(spec, current); err != nil {
		return err
	}
	if err := builder.applyResponseAccessControl(spec, current); err != nil {
		return err
	}
	if err := builder.applyResponseProtection(spec, current); err != nil {
		return err
	}
	builder.applyTags(spec, current)
	return nil
}

func (b *webAppFirewallPolicyUpdateBuilder) applyDisplayName(
	spec wafv1beta1.WebAppFirewallPolicySpec,
	current wafsdk.WebAppFirewallPolicy,
) {
	displayName := strings.TrimSpace(spec.DisplayName)
	if displayName == "" || webAppFirewallPolicyStringPtrEqual(current.DisplayName, displayName) {
		return
	}
	b.details.DisplayName = common.String(displayName)
	b.needed = true
}

func (b *webAppFirewallPolicyUpdateBuilder) applyActions(
	spec wafv1beta1.WebAppFirewallPolicySpec,
	current wafsdk.WebAppFirewallPolicy,
) error {
	if spec.Actions == nil {
		return nil
	}
	desired, err := webAppFirewallPolicyActions(spec.Actions)
	if err != nil {
		return err
	}
	if webAppFirewallPolicyJSONEqual(desired, current.Actions) {
		return nil
	}
	b.details.Actions = desired
	b.needed = true
	return nil
}

func (b *webAppFirewallPolicyUpdateBuilder) applyRequestAccessControl(
	spec wafv1beta1.WebAppFirewallPolicySpec,
	current wafsdk.WebAppFirewallPolicy,
) error {
	if !webAppFirewallPolicyRequestAccessControlSpecified(spec.RequestAccessControl) {
		return nil
	}
	desired, err := webAppFirewallPolicyRequestAccessControl(spec.RequestAccessControl)
	if err != nil {
		return err
	}
	if webAppFirewallPolicyJSONEqual(desired, current.RequestAccessControl) {
		return nil
	}
	b.details.RequestAccessControl = desired
	b.needed = true
	return nil
}

func (b *webAppFirewallPolicyUpdateBuilder) applyRequestRateLimiting(
	spec wafv1beta1.WebAppFirewallPolicySpec,
	current wafsdk.WebAppFirewallPolicy,
) error {
	if spec.RequestRateLimiting.Rules == nil {
		return nil
	}
	desired, err := webAppFirewallPolicyRequestRateLimiting(spec.RequestRateLimiting)
	if err != nil {
		return err
	}
	if webAppFirewallPolicyJSONEqual(desired, current.RequestRateLimiting) {
		return nil
	}
	b.details.RequestRateLimiting = desired
	b.needed = true
	return nil
}

func (b *webAppFirewallPolicyUpdateBuilder) applyRequestProtection(
	spec wafv1beta1.WebAppFirewallPolicySpec,
	current wafsdk.WebAppFirewallPolicy,
) error {
	if !webAppFirewallPolicyRequestProtectionSpecified(spec.RequestProtection) {
		return nil
	}
	desired, err := webAppFirewallPolicyRequestProtection(spec.RequestProtection)
	if err != nil {
		return err
	}
	if webAppFirewallPolicyJSONEqual(desired, current.RequestProtection) {
		return nil
	}
	b.details.RequestProtection = desired
	b.needed = true
	return nil
}

func (b *webAppFirewallPolicyUpdateBuilder) applyResponseAccessControl(
	spec wafv1beta1.WebAppFirewallPolicySpec,
	current wafsdk.WebAppFirewallPolicy,
) error {
	if spec.ResponseAccessControl.Rules == nil {
		return nil
	}
	desired, err := webAppFirewallPolicyResponseAccessControl(spec.ResponseAccessControl)
	if err != nil {
		return err
	}
	if webAppFirewallPolicyJSONEqual(desired, current.ResponseAccessControl) {
		return nil
	}
	b.details.ResponseAccessControl = desired
	b.needed = true
	return nil
}

func (b *webAppFirewallPolicyUpdateBuilder) applyResponseProtection(
	spec wafv1beta1.WebAppFirewallPolicySpec,
	current wafsdk.WebAppFirewallPolicy,
) error {
	if spec.ResponseProtection.Rules == nil {
		return nil
	}
	desired, err := webAppFirewallPolicyResponseProtection(spec.ResponseProtection)
	if err != nil {
		return err
	}
	if webAppFirewallPolicyJSONEqual(desired, current.ResponseProtection) {
		return nil
	}
	b.details.ResponseProtection = desired
	b.needed = true
	return nil
}

func (b *webAppFirewallPolicyUpdateBuilder) applyTags(
	spec wafv1beta1.WebAppFirewallPolicySpec,
	current wafsdk.WebAppFirewallPolicy,
) {
	if spec.FreeformTags != nil && !webAppFirewallPolicyStringMapEqual(spec.FreeformTags, current.FreeformTags) {
		b.details.FreeformTags = cloneWebAppFirewallPolicyStringMap(spec.FreeformTags)
		b.needed = true
	}
	if spec.DefinedTags != nil {
		desired := webAppFirewallPolicyDefinedTagsFromSpec(spec.DefinedTags)
		if !webAppFirewallPolicyJSONEqual(desired, current.DefinedTags) {
			b.details.DefinedTags = desired
			b.needed = true
		}
	}
	if spec.SystemTags != nil {
		desired := webAppFirewallPolicyDefinedTagsFromSpec(spec.SystemTags)
		if !webAppFirewallPolicyJSONEqual(desired, current.SystemTags) {
			b.details.SystemTags = desired
			b.needed = true
		}
	}
}

func validateWebAppFirewallPolicySpec(spec wafv1beta1.WebAppFirewallPolicySpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if len(missing) > 0 {
		return fmt.Errorf("WebAppFirewallPolicy spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func validateWebAppFirewallPolicyCreateOnlyDriftForResponse(
	resource *wafv1beta1.WebAppFirewallPolicy,
	currentResponse any,
) error {
	current, ok := webAppFirewallPolicyFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current WebAppFirewallPolicy response does not expose a WebAppFirewallPolicy body")
	}
	return validateWebAppFirewallPolicyCreateOnlyDrift(resource, current)
}

func validateWebAppFirewallPolicyCreateOnlyDrift(
	resource *wafv1beta1.WebAppFirewallPolicy,
	_ wafsdk.WebAppFirewallPolicy,
) error {
	if resource == nil {
		return fmt.Errorf("WebAppFirewallPolicy resource is nil")
	}
	return nil
}

func guardWebAppFirewallPolicyExistingBeforeCreate(
	_ context.Context,
	resource *wafv1beta1.WebAppFirewallPolicy,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("WebAppFirewallPolicy resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func webAppFirewallPolicyActions(specActions []wafv1beta1.WebAppFirewallPolicyAction) ([]wafsdk.Action, error) {
	actions := make([]wafsdk.Action, 0, len(specActions))
	for index, spec := range specActions {
		name := strings.TrimSpace(spec.Name)
		if name == "" {
			return nil, fmt.Errorf("WebAppFirewallPolicy spec.actions[%d] is missing required field name", index)
		}
		actionType := strings.ToUpper(strings.TrimSpace(spec.Type))
		switch actionType {
		case string(wafsdk.ActionTypeAllow):
			actions = append(actions, wafsdk.AllowAction{Name: common.String(name)})
		case string(wafsdk.ActionTypeCheck):
			actions = append(actions, wafsdk.CheckAction{Name: common.String(name)})
		case string(wafsdk.ActionTypeReturnHttpResponse):
			if spec.Code == 0 {
				return nil, fmt.Errorf("WebAppFirewallPolicy spec.actions[%d] RETURN_HTTP_RESPONSE is missing required field code", index)
			}
			body, err := webAppFirewallPolicyActionBody(spec.Body)
			if err != nil {
				return nil, fmt.Errorf("WebAppFirewallPolicy spec.actions[%d].body: %w", index, err)
			}
			actions = append(actions, wafsdk.ReturnHttpResponseAction{
				Name:    common.String(name),
				Code:    common.Int(spec.Code),
				Headers: webAppFirewallPolicyResponseHeaders(spec.Headers),
				Body:    body,
			})
		case "":
			return nil, fmt.Errorf("WebAppFirewallPolicy spec.actions[%d] is missing required field type", index)
		default:
			return nil, fmt.Errorf("WebAppFirewallPolicy spec.actions[%d] has unsupported type %q", index, spec.Type)
		}
	}
	return actions, nil
}

func webAppFirewallPolicyActionBody(spec wafv1beta1.WebAppFirewallPolicyActionBody) (wafsdk.HttpResponseBody, error) {
	bodyType := strings.ToUpper(strings.TrimSpace(spec.Type))
	switch bodyType {
	case "":
		if strings.TrimSpace(spec.Text) == "" && strings.TrimSpace(spec.Template) == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("type is required when text or template is set")
	case string(wafsdk.HttpResponseBodyTypeStaticText):
		return wafsdk.StaticTextHttpResponseBody{Text: common.String(spec.Text)}, nil
	case string(wafsdk.HttpResponseBodyTypeDynamic):
		return wafsdk.DynamicHttpResponseBody{Template: common.String(spec.Template)}, nil
	default:
		return nil, fmt.Errorf("unsupported type %q", spec.Type)
	}
}

func webAppFirewallPolicyResponseHeaders(specHeaders []wafv1beta1.WebAppFirewallPolicyActionHeader) []wafsdk.ResponseHeader {
	if specHeaders == nil {
		return nil
	}
	headers := make([]wafsdk.ResponseHeader, 0, len(specHeaders))
	for _, spec := range specHeaders {
		headers = append(headers, wafsdk.ResponseHeader{
			Name:  common.String(strings.TrimSpace(spec.Name)),
			Value: common.String(spec.Value),
		})
	}
	return headers
}

func webAppFirewallPolicyRequestAccessControlSpecified(spec wafv1beta1.WebAppFirewallPolicyRequestAccessControl) bool {
	return strings.TrimSpace(spec.DefaultActionName) != "" || spec.Rules != nil
}

func webAppFirewallPolicyRequestAccessControl(spec wafv1beta1.WebAppFirewallPolicyRequestAccessControl) (*wafsdk.RequestAccessControl, error) {
	if strings.TrimSpace(spec.DefaultActionName) == "" {
		return nil, fmt.Errorf("WebAppFirewallPolicy spec.requestAccessControl is missing required field defaultActionName")
	}
	rules, err := webAppFirewallPolicyAccessControlRules(spec.Rules, "requestAccessControl")
	if err != nil {
		return nil, err
	}
	return &wafsdk.RequestAccessControl{
		DefaultActionName: common.String(strings.TrimSpace(spec.DefaultActionName)),
		Rules:             rules,
	}, nil
}

func webAppFirewallPolicyResponseAccessControl(spec wafv1beta1.WebAppFirewallPolicyResponseAccessControl) (*wafsdk.ResponseAccessControl, error) {
	rules, err := webAppFirewallPolicyResponseAccessControlRules(spec.Rules, "responseAccessControl")
	if err != nil {
		return nil, err
	}
	return &wafsdk.ResponseAccessControl{Rules: rules}, nil
}

func webAppFirewallPolicyAccessControlRules(
	specRules []wafv1beta1.WebAppFirewallPolicyRequestAccessControlRule,
	fieldPath string,
) ([]wafsdk.AccessControlRule, error) {
	if specRules == nil {
		return nil, nil
	}
	rules := make([]wafsdk.AccessControlRule, 0, len(specRules))
	for index, spec := range specRules {
		name := strings.TrimSpace(spec.Name)
		actionName := strings.TrimSpace(spec.ActionName)
		if name == "" {
			return nil, fmt.Errorf("WebAppFirewallPolicy spec.%s.rules[%d] is missing required field name", fieldPath, index)
		}
		if actionName == "" {
			return nil, fmt.Errorf("WebAppFirewallPolicy spec.%s.rules[%d] is missing required field actionName", fieldPath, index)
		}
		rule := wafsdk.AccessControlRule{
			Name:       common.String(name),
			ActionName: common.String(actionName),
		}
		if condition := strings.TrimSpace(spec.Condition); condition != "" {
			rule.Condition = common.String(condition)
		}
		if language := strings.TrimSpace(spec.ConditionLanguage); language != "" {
			rule.ConditionLanguage = wafsdk.WebAppFirewallPolicyRuleConditionLanguageEnum(language)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func webAppFirewallPolicyResponseAccessControlRules(
	specRules []wafv1beta1.WebAppFirewallPolicyResponseAccessControlRule,
	fieldPath string,
) ([]wafsdk.AccessControlRule, error) {
	if specRules == nil {
		return nil, nil
	}
	rules := make([]wafsdk.AccessControlRule, 0, len(specRules))
	for index, spec := range specRules {
		name := strings.TrimSpace(spec.Name)
		actionName := strings.TrimSpace(spec.ActionName)
		if name == "" {
			return nil, fmt.Errorf("WebAppFirewallPolicy spec.%s.rules[%d] is missing required field name", fieldPath, index)
		}
		if actionName == "" {
			return nil, fmt.Errorf("WebAppFirewallPolicy spec.%s.rules[%d] is missing required field actionName", fieldPath, index)
		}
		rule := wafsdk.AccessControlRule{
			Name:       common.String(name),
			ActionName: common.String(actionName),
		}
		if condition := strings.TrimSpace(spec.Condition); condition != "" {
			rule.Condition = common.String(condition)
		}
		if language := strings.TrimSpace(spec.ConditionLanguage); language != "" {
			rule.ConditionLanguage = wafsdk.WebAppFirewallPolicyRuleConditionLanguageEnum(language)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func webAppFirewallPolicyRequestRateLimiting(spec wafv1beta1.WebAppFirewallPolicyRequestRateLimiting) (*wafsdk.RequestRateLimiting, error) {
	rules, err := webAppFirewallPolicyRequestRateLimitingRules(spec.Rules)
	if err != nil {
		return nil, err
	}
	return &wafsdk.RequestRateLimiting{Rules: rules}, nil
}

func webAppFirewallPolicyRequestRateLimitingRules(
	specRules []wafv1beta1.WebAppFirewallPolicyRequestRateLimitingRule,
) ([]wafsdk.RequestRateLimitingRule, error) {
	rules := make([]wafsdk.RequestRateLimitingRule, 0, len(specRules))
	for index, spec := range specRules {
		rule, err := webAppFirewallPolicyRequestRateLimitingRule(index, spec)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func webAppFirewallPolicyRequestRateLimitingRule(
	index int,
	spec wafv1beta1.WebAppFirewallPolicyRequestRateLimitingRule,
) (wafsdk.RequestRateLimitingRule, error) {
	name := strings.TrimSpace(spec.Name)
	actionName := strings.TrimSpace(spec.ActionName)
	if name == "" {
		return wafsdk.RequestRateLimitingRule{}, fmt.Errorf("WebAppFirewallPolicy spec.requestRateLimiting.rules[%d] is missing required field name", index)
	}
	if actionName == "" {
		return wafsdk.RequestRateLimitingRule{}, fmt.Errorf("WebAppFirewallPolicy spec.requestRateLimiting.rules[%d] is missing required field actionName", index)
	}
	configurations, err := webAppFirewallPolicyRequestRateLimitingConfigurations(index, spec.Configurations)
	if err != nil {
		return wafsdk.RequestRateLimitingRule{}, err
	}
	rule := wafsdk.RequestRateLimitingRule{
		Name:           common.String(name),
		ActionName:     common.String(actionName),
		Configurations: configurations,
	}
	if condition := strings.TrimSpace(spec.Condition); condition != "" {
		rule.Condition = common.String(condition)
	}
	if language := strings.TrimSpace(spec.ConditionLanguage); language != "" {
		rule.ConditionLanguage = wafsdk.WebAppFirewallPolicyRuleConditionLanguageEnum(language)
	}
	return rule, nil
}

func webAppFirewallPolicyRequestRateLimitingConfigurations(
	ruleIndex int,
	specConfigurations []wafv1beta1.WebAppFirewallPolicyRequestRateLimitingRuleConfiguration,
) ([]wafsdk.RequestRateLimitingConfiguration, error) {
	if len(specConfigurations) == 0 {
		return nil, fmt.Errorf("WebAppFirewallPolicy spec.requestRateLimiting.rules[%d] is missing required field configurations", ruleIndex)
	}
	configurations := make([]wafsdk.RequestRateLimitingConfiguration, 0, len(specConfigurations))
	for index, spec := range specConfigurations {
		if spec.PeriodInSeconds == 0 {
			return nil, fmt.Errorf("WebAppFirewallPolicy spec.requestRateLimiting.rules[%d].configurations[%d] is missing required field periodInSeconds", ruleIndex, index)
		}
		if spec.RequestsLimit == 0 {
			return nil, fmt.Errorf("WebAppFirewallPolicy spec.requestRateLimiting.rules[%d].configurations[%d] is missing required field requestsLimit", ruleIndex, index)
		}
		configurations = append(configurations, wafsdk.RequestRateLimitingConfiguration{
			PeriodInSeconds:         common.Int(spec.PeriodInSeconds),
			RequestsLimit:           common.Int(spec.RequestsLimit),
			ActionDurationInSeconds: common.Int(spec.ActionDurationInSeconds),
		})
	}
	return configurations, nil
}

func webAppFirewallPolicyRequestProtectionSpecified(spec wafv1beta1.WebAppFirewallPolicyRequestProtection) bool {
	return spec.Rules != nil ||
		spec.BodyInspectionSizeLimitInBytes != 0 ||
		strings.TrimSpace(spec.BodyInspectionSizeLimitExceededActionName) != ""
}

func webAppFirewallPolicyRequestProtection(spec wafv1beta1.WebAppFirewallPolicyRequestProtection) (*wafsdk.RequestProtection, error) {
	rules, err := webAppFirewallPolicyRequestProtectionRules(spec.Rules, "requestProtection")
	if err != nil {
		return nil, err
	}
	module := &wafsdk.RequestProtection{Rules: rules}
	if spec.BodyInspectionSizeLimitInBytes != 0 {
		module.BodyInspectionSizeLimitInBytes = common.Int(spec.BodyInspectionSizeLimitInBytes)
	}
	if actionName := strings.TrimSpace(spec.BodyInspectionSizeLimitExceededActionName); actionName != "" {
		module.BodyInspectionSizeLimitExceededActionName = common.String(actionName)
	}
	return module, nil
}

func webAppFirewallPolicyResponseProtection(spec wafv1beta1.WebAppFirewallPolicyResponseProtection) (*wafsdk.ResponseProtection, error) {
	rules, err := webAppFirewallPolicyResponseProtectionRules(spec.Rules, "responseProtection")
	if err != nil {
		return nil, err
	}
	return &wafsdk.ResponseProtection{Rules: rules}, nil
}

func webAppFirewallPolicyRequestProtectionRules(
	specRules []wafv1beta1.WebAppFirewallPolicyRequestProtectionRule,
	fieldPath string,
) ([]wafsdk.ProtectionRule, error) {
	if specRules == nil {
		return nil, nil
	}
	rules := make([]wafsdk.ProtectionRule, 0, len(specRules))
	for index, spec := range specRules {
		rule, err := webAppFirewallPolicyProtectionRule(
			fieldPath,
			index,
			spec.Name,
			spec.ActionName,
			spec.ProtectionCapabilities,
			spec.Condition,
			spec.ConditionLanguage,
			spec.ProtectionCapabilitySettings,
			spec.IsBodyInspectionEnabled,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func webAppFirewallPolicyResponseProtectionRules(
	specRules []wafv1beta1.WebAppFirewallPolicyResponseProtectionRule,
	fieldPath string,
) ([]wafsdk.ProtectionRule, error) {
	if specRules == nil {
		return nil, nil
	}
	rules := make([]wafsdk.ProtectionRule, 0, len(specRules))
	for index, spec := range specRules {
		capabilities := make([]wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapability, 0, len(spec.ProtectionCapabilities))
		for _, capability := range spec.ProtectionCapabilities {
			capabilities = append(capabilities, wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapability{
				Key:                          capability.Key,
				Version:                      capability.Version,
				Exclusions:                   wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapabilityExclusions(capability.Exclusions),
				ActionName:                   capability.ActionName,
				CollaborativeActionThreshold: capability.CollaborativeActionThreshold,
				CollaborativeWeights:         webAppFirewallPolicyRequestCollaborativeWeights(capability.CollaborativeWeights),
			})
		}
		settings := wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapabilitySettings(spec.ProtectionCapabilitySettings)
		rule, err := webAppFirewallPolicyProtectionRule(
			fieldPath,
			index,
			spec.Name,
			spec.ActionName,
			capabilities,
			spec.Condition,
			spec.ConditionLanguage,
			settings,
			spec.IsBodyInspectionEnabled,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func webAppFirewallPolicyRequestCollaborativeWeights(
	weights []wafv1beta1.WebAppFirewallPolicyResponseProtectionRuleProtectionCapabilityCollaborativeWeight,
) []wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapabilityCollaborativeWeight {
	if weights == nil {
		return nil
	}
	converted := make([]wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapabilityCollaborativeWeight, 0, len(weights))
	for _, weight := range weights {
		converted = append(converted, wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapabilityCollaborativeWeight(weight))
	}
	return converted
}

func webAppFirewallPolicyProtectionRule(
	fieldPath string,
	ruleIndex int,
	name string,
	actionName string,
	specCapabilities []wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapability,
	condition string,
	conditionLanguage string,
	settings wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapabilitySettings,
	isBodyInspectionEnabled bool,
) (wafsdk.ProtectionRule, error) {
	name = strings.TrimSpace(name)
	actionName = strings.TrimSpace(actionName)
	if name == "" {
		return wafsdk.ProtectionRule{}, fmt.Errorf("WebAppFirewallPolicy spec.%s.rules[%d] is missing required field name", fieldPath, ruleIndex)
	}
	if actionName == "" {
		return wafsdk.ProtectionRule{}, fmt.Errorf("WebAppFirewallPolicy spec.%s.rules[%d] is missing required field actionName", fieldPath, ruleIndex)
	}
	if len(specCapabilities) == 0 {
		return wafsdk.ProtectionRule{}, fmt.Errorf("WebAppFirewallPolicy spec.%s.rules[%d] is missing required field protectionCapabilities", fieldPath, ruleIndex)
	}
	capabilities, err := webAppFirewallPolicyProtectionCapabilities(specCapabilities, fieldPath, ruleIndex)
	if err != nil {
		return wafsdk.ProtectionRule{}, err
	}
	rule := wafsdk.ProtectionRule{
		Name:                   common.String(name),
		ActionName:             common.String(actionName),
		ProtectionCapabilities: capabilities,
	}
	if trimmedCondition := strings.TrimSpace(condition); trimmedCondition != "" {
		rule.Condition = common.String(trimmedCondition)
	}
	if language := strings.TrimSpace(conditionLanguage); language != "" {
		rule.ConditionLanguage = wafsdk.WebAppFirewallPolicyRuleConditionLanguageEnum(language)
	}
	if webAppFirewallPolicyProtectionSettingsSpecified(settings) {
		rule.ProtectionCapabilitySettings = webAppFirewallPolicyProtectionSettings(settings)
	}
	if isBodyInspectionEnabled {
		rule.IsBodyInspectionEnabled = common.Bool(true)
	}
	return rule, nil
}

func webAppFirewallPolicyProtectionCapabilities(
	specCapabilities []wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapability,
	fieldPath string,
	ruleIndex int,
) ([]wafsdk.ProtectionCapability, error) {
	capabilities := make([]wafsdk.ProtectionCapability, 0, len(specCapabilities))
	for capabilityIndex, spec := range specCapabilities {
		key := strings.TrimSpace(spec.Key)
		if key == "" {
			return nil, fmt.Errorf("WebAppFirewallPolicy spec.%s.rules[%d].protectionCapabilities[%d] is missing required field key", fieldPath, ruleIndex, capabilityIndex)
		}
		if spec.Version == 0 {
			return nil, fmt.Errorf("WebAppFirewallPolicy spec.%s.rules[%d].protectionCapabilities[%d] is missing required field version", fieldPath, ruleIndex, capabilityIndex)
		}
		capability := wafsdk.ProtectionCapability{
			Key:     common.String(key),
			Version: common.Int(spec.Version),
		}
		if webAppFirewallPolicyCapabilityExclusionsSpecified(spec.Exclusions) {
			capability.Exclusions = &wafsdk.ProtectionCapabilityExclusions{
				RequestCookies: cloneWebAppFirewallPolicyStringSlice(spec.Exclusions.RequestCookies),
				Args:           cloneWebAppFirewallPolicyStringSlice(spec.Exclusions.Args),
			}
		}
		if actionName := strings.TrimSpace(spec.ActionName); actionName != "" {
			capability.ActionName = common.String(actionName)
		}
		if spec.CollaborativeActionThreshold != 0 {
			capability.CollaborativeActionThreshold = common.Int(spec.CollaborativeActionThreshold)
		}
		if spec.CollaborativeWeights != nil {
			capability.CollaborativeWeights = webAppFirewallPolicyCollaborativeWeights(spec.CollaborativeWeights)
		}
		capabilities = append(capabilities, capability)
	}
	return capabilities, nil
}

func webAppFirewallPolicyCapabilityExclusionsSpecified(spec wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapabilityExclusions) bool {
	return spec.RequestCookies != nil || spec.Args != nil
}

func webAppFirewallPolicyCollaborativeWeights(
	specWeights []wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapabilityCollaborativeWeight,
) []wafsdk.CollaborativeCapabilityWeightOverride {
	if specWeights == nil {
		return nil
	}
	weights := make([]wafsdk.CollaborativeCapabilityWeightOverride, 0, len(specWeights))
	for _, spec := range specWeights {
		weights = append(weights, wafsdk.CollaborativeCapabilityWeightOverride{
			Key:    common.String(strings.TrimSpace(spec.Key)),
			Weight: common.Int(spec.Weight),
		})
	}
	return weights
}

func webAppFirewallPolicyProtectionSettingsSpecified(spec wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapabilitySettings) bool {
	return spec.MaxNumberOfArguments != 0 ||
		spec.MaxSingleArgumentLength != 0 ||
		spec.MaxTotalArgumentLength != 0 ||
		spec.MaxHttpRequestHeaders != 0 ||
		spec.MaxHttpRequestHeaderLength != 0 ||
		spec.AllowedHttpMethods != nil
}

func webAppFirewallPolicyProtectionSettings(
	spec wafv1beta1.WebAppFirewallPolicyRequestProtectionRuleProtectionCapabilitySettings,
) *wafsdk.ProtectionCapabilitySettings {
	settings := &wafsdk.ProtectionCapabilitySettings{
		AllowedHttpMethods: cloneWebAppFirewallPolicyStringSlice(spec.AllowedHttpMethods),
	}
	if spec.MaxNumberOfArguments != 0 {
		settings.MaxNumberOfArguments = common.Int(spec.MaxNumberOfArguments)
	}
	if spec.MaxSingleArgumentLength != 0 {
		settings.MaxSingleArgumentLength = common.Int(spec.MaxSingleArgumentLength)
	}
	if spec.MaxTotalArgumentLength != 0 {
		settings.MaxTotalArgumentLength = common.Int(spec.MaxTotalArgumentLength)
	}
	if spec.MaxHttpRequestHeaders != 0 {
		settings.MaxHttpRequestHeaders = common.Int(spec.MaxHttpRequestHeaders)
	}
	if spec.MaxHttpRequestHeaderLength != 0 {
		settings.MaxHttpRequestHeaderLength = common.Int(spec.MaxHttpRequestHeaderLength)
	}
	return settings
}

func webAppFirewallPolicyRequiresCompartmentMove(resource *wafv1beta1.WebAppFirewallPolicy, currentResponse any) bool {
	if resource == nil {
		return false
	}
	current, ok := webAppFirewallPolicyFromResponse(currentResponse)
	if !ok {
		return false
	}
	return webAppFirewallPolicyCompartmentNeedsMove(resource.Spec, current)
}

func webAppFirewallPolicyCompartmentNeedsMove(
	spec wafv1beta1.WebAppFirewallPolicySpec,
	current wafsdk.WebAppFirewallPolicy,
) bool {
	desired := strings.TrimSpace(spec.CompartmentId)
	observed := strings.TrimSpace(webAppFirewallPolicyStringValue(current.CompartmentId))
	return desired != "" && observed != "" && desired != observed
}

func applyWebAppFirewallPolicyCompartmentMove(
	ctx context.Context,
	resource *wafv1beta1.WebAppFirewallPolicy,
	currentResponse any,
	client webAppFirewallPolicyOCIClient,
	initErr error,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppFirewallPolicy resource is nil")
	}
	if err := webAppFirewallPolicyClientReady(client, initErr); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	current, ok := webAppFirewallPolicyFromResponse(currentResponse)
	if !ok {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("current WebAppFirewallPolicy response does not expose a WebAppFirewallPolicy body")
	}
	resourceID := strings.TrimSpace(webAppFirewallPolicyStringValue(current.Id))
	if resourceID == "" {
		resourceID = webAppFirewallPolicyCurrentID(resource)
	}
	if resourceID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppFirewallPolicy compartment move requires a tracked WebAppFirewallPolicy id")
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppFirewallPolicy compartment move requires spec.compartmentId")
	}

	response, err := client.ChangeWebAppFirewallPolicyCompartment(ctx, wafsdk.ChangeWebAppFirewallPolicyCompartmentRequest{
		WebAppFirewallPolicyId: common.String(resourceID),
		ChangeWebAppFirewallPolicyCompartmentDetails: wafsdk.ChangeWebAppFirewallPolicyCompartmentDetails{
			CompartmentId: common.String(compartmentID),
		},
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	workRequestID := strings.TrimSpace(webAppFirewallPolicyStringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppFirewallPolicy compartment move did not return an opc-work-request-id")
	}

	resource.Status.Id = resourceID
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:    workRequestID,
		RawOperationType: string(wafsdk.WorkRequestOperationTypeMoveWafPolicy),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		Message:          fmt.Sprintf("WebAppFirewallPolicy compartment move work request %s is pending", workRequestID),
	}, loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{
		IsSuccessful:  projection.Condition != shared.Failed,
		ShouldRequeue: projection.ShouldRequeue,
	}, nil
}

func listWebAppFirewallPolicyPages(
	ctx context.Context,
	client webAppFirewallPolicyOCIClient,
	initErr error,
	request wafsdk.ListWebAppFirewallPoliciesRequest,
) (wafsdk.ListWebAppFirewallPoliciesResponse, error) {
	if err := webAppFirewallPolicyClientReady(client, initErr); err != nil {
		return wafsdk.ListWebAppFirewallPoliciesResponse{}, err
	}

	var combined wafsdk.ListWebAppFirewallPoliciesResponse
	seenPages := map[string]struct{}{}
	for {
		response, err := client.ListWebAppFirewallPolicies(ctx, request)
		if err != nil {
			return wafsdk.ListWebAppFirewallPoliciesResponse{}, conservativeWebAppFirewallPolicyNotFoundError(err, "list")
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == wafsdk.WebAppFirewallPolicyLifecycleStateDeleted {
				continue
			}
			combined.Items = append(combined.Items, item)
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		nextPage := strings.TrimSpace(*response.OpcNextPage)
		if _, ok := seenPages[nextPage]; ok {
			return wafsdk.ListWebAppFirewallPoliciesResponse{}, fmt.Errorf("WebAppFirewallPolicy list pagination returned repeated page token %q", nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = common.String(nextPage)
		combined.OpcNextPage = common.String(nextPage)
	}
}

func projectWebAppFirewallPolicyStatus(resource *wafv1beta1.WebAppFirewallPolicy, response any) error {
	if resource == nil {
		return fmt.Errorf("WebAppFirewallPolicy resource is nil")
	}
	current, ok := webAppFirewallPolicyFromResponse(response)
	if !ok {
		return nil
	}
	osokStatus := resource.Status.OsokStatus
	currentForJSON := current
	currentForJSON.DefinedTags = nil
	currentForJSON.SystemTags = nil
	payload, err := json.Marshal(currentForJSON)
	if err != nil {
		return fmt.Errorf("marshal WebAppFirewallPolicy status body: %w", err)
	}
	var status wafv1beta1.WebAppFirewallPolicyStatus
	if err := json.Unmarshal(payload, &status); err != nil {
		return fmt.Errorf("project WebAppFirewallPolicy status body: %w", err)
	}
	status.OsokStatus = osokStatus
	status.Id = webAppFirewallPolicyStringValue(current.Id)
	status.DisplayName = webAppFirewallPolicyStringValue(current.DisplayName)
	status.CompartmentId = webAppFirewallPolicyStringValue(current.CompartmentId)
	status.LifecycleState = string(current.LifecycleState)
	status.LifecycleDetails = webAppFirewallPolicyStringValue(current.LifecycleDetails)
	status.FreeformTags = cloneWebAppFirewallPolicyStringMap(current.FreeformTags)
	status.DefinedTags = webAppFirewallPolicyTagsToStatus(current.DefinedTags)
	status.SystemTags = webAppFirewallPolicyTagsToStatus(current.SystemTags)
	if current.TimeCreated != nil {
		status.TimeCreated = current.TimeCreated.String()
	}
	if current.TimeUpdated != nil {
		status.TimeUpdated = current.TimeUpdated.String()
	}
	resource.Status = status
	return nil
}

func clearTrackedWebAppFirewallPolicyIdentity(resource *wafv1beta1.WebAppFirewallPolicy) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus = shared.OSOKStatus{}
}

func handleWebAppFirewallPolicyDeleteError(resource *wafv1beta1.WebAppFirewallPolicy, err error) error {
	err = conservativeWebAppFirewallPolicyNotFoundError(err, "delete")
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func conservativeWebAppFirewallPolicyNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	operation = strings.TrimSpace(operation)
	if operation == "" {
		operation = "operation"
	}
	return ambiguousWebAppFirewallPolicyNotFoundError{
		message: fmt.Sprintf(
			"WebAppFirewallPolicy %s returned ambiguous 404 NotAuthorizedOrNotFound: %s",
			operation,
			err.Error(),
		),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func (c webAppFirewallPolicyPendingWriteDeleteClient) CreateOrUpdate(
	ctx context.Context,
	resource *wafv1beta1.WebAppFirewallPolicy,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c webAppFirewallPolicyPendingWriteDeleteClient) Delete(
	ctx context.Context,
	resource *wafv1beta1.WebAppFirewallPolicy,
) (bool, error) {
	if webAppFirewallPolicyHasPendingDeleteWorkRequest(resource) {
		return c.delegate.Delete(ctx, resource)
	}
	if webAppFirewallPolicyHasPendingWrite(resource) {
		response, err := c.delegate.CreateOrUpdate(ctx, resource, ctrl.Request{})
		if err != nil {
			return false, err
		}
		if webAppFirewallPolicyHasPendingWrite(resource) || response.ShouldRequeue {
			return false, nil
		}
	}
	if err := c.guardWebAppFirewallPolicyDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c webAppFirewallPolicyPendingWriteDeleteClient) guardWebAppFirewallPolicyDeleteRead(
	ctx context.Context,
	resource *wafv1beta1.WebAppFirewallPolicy,
) error {
	currentID := webAppFirewallPolicyCurrentID(resource)
	if currentID == "" || c.getWebAppFirewallPolicy == nil {
		return nil
	}

	_, err := c.getWebAppFirewallPolicy(ctx, wafsdk.GetWebAppFirewallPolicyRequest{
		WebAppFirewallPolicyId: common.String(currentID),
	})
	if err == nil || errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func webAppFirewallPolicyHasPendingWrite(resource *wafv1beta1.WebAppFirewallPolicy) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		return false
	}
	return current.Phase == shared.OSOKAsyncPhaseCreate || current.Phase == shared.OSOKAsyncPhaseUpdate
}

func webAppFirewallPolicyHasPendingDeleteWorkRequest(resource *wafv1beta1.WebAppFirewallPolicy) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Source == shared.OSOKAsyncSourceWorkRequest &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		strings.TrimSpace(current.WorkRequestID) != "" &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func webAppFirewallPolicyCurrentID(resource *wafv1beta1.WebAppFirewallPolicy) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func getWebAppFirewallPolicyWorkRequest(
	ctx context.Context,
	client webAppFirewallPolicyOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if err := webAppFirewallPolicyClientReady(client, initErr); err != nil {
		return nil, err
	}
	trimmedWorkRequestID := strings.TrimSpace(workRequestID)
	if trimmedWorkRequestID == "" {
		return nil, fmt.Errorf("WebAppFirewallPolicy work request id is required")
	}
	response, err := client.GetWorkRequest(ctx, wafsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(trimmedWorkRequestID),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveWebAppFirewallPolicyWorkRequestAction(workRequest any) (string, error) {
	current, err := webAppFirewallPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	if action, err := resolveWebAppFirewallPolicyWorkRequestResourceAction(current); err != nil || action != "" {
		return action, err
	}
	return string(current.OperationType), nil
}

func resolveWebAppFirewallPolicyWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := webAppFirewallPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := webAppFirewallPolicyWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverWebAppFirewallPolicyIDFromWorkRequest(
	_ *wafv1beta1.WebAppFirewallPolicy,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := webAppFirewallPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveWebAppFirewallPolicyIDFromWorkRequest(current, webAppFirewallPolicyWorkRequestActionForPhase(phase))
}

func webAppFirewallPolicyWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := webAppFirewallPolicyWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf(
		"WebAppFirewallPolicy %s work request %s is %s",
		phase,
		webAppFirewallPolicyStringValue(current.Id),
		current.Status,
	)
}

func webAppFirewallPolicyWorkRequestFromAny(workRequest any) (wafsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case wafsdk.WorkRequest:
		return current, nil
	case *wafsdk.WorkRequest:
		if current == nil {
			return wafsdk.WorkRequest{}, fmt.Errorf("WebAppFirewallPolicy work request is nil")
		}
		return *current, nil
	default:
		return wafsdk.WorkRequest{}, fmt.Errorf("unexpected WebAppFirewallPolicy work request type %T", workRequest)
	}
}

func webAppFirewallPolicyWorkRequestPhaseFromOperationType(operationType wafsdk.WorkRequestOperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case wafsdk.WorkRequestOperationTypeCreateWafPolicy:
		return shared.OSOKAsyncPhaseCreate, true
	case wafsdk.WorkRequestOperationTypeUpdateWafPolicy, wafsdk.WorkRequestOperationTypeMoveWafPolicy:
		return shared.OSOKAsyncPhaseUpdate, true
	case wafsdk.WorkRequestOperationTypeDeleteWafPolicy:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func webAppFirewallPolicyWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) wafsdk.WorkRequestResourceActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return wafsdk.WorkRequestResourceActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return wafsdk.WorkRequestResourceActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return wafsdk.WorkRequestResourceActionTypeDeleted
	default:
		return ""
	}
}

func resolveWebAppFirewallPolicyIDFromWorkRequest(
	workRequest wafsdk.WorkRequest,
	action wafsdk.WorkRequestResourceActionTypeEnum,
) (string, error) {
	if id, ok := webAppFirewallPolicyIDFromWorkRequestResources(workRequest.Resources, action); ok {
		return id, nil
	}
	if id, ok := webAppFirewallPolicyIDFromWorkRequestResources(workRequest.Resources, ""); ok {
		return id, nil
	}
	return "", fmt.Errorf("WebAppFirewallPolicy work request %s does not expose a resource identifier", webAppFirewallPolicyStringValue(workRequest.Id))
}

func webAppFirewallPolicyIDFromWorkRequestResources(
	resources []wafsdk.WorkRequestResource,
	action wafsdk.WorkRequestResourceActionTypeEnum,
) (string, bool) {
	for _, resource := range resources {
		id, ok := webAppFirewallPolicyIDFromWorkRequestResource(resource, action)
		if ok {
			return id, true
		}
	}
	return "", false
}

func webAppFirewallPolicyIDFromWorkRequestResource(
	resource wafsdk.WorkRequestResource,
	action wafsdk.WorkRequestResourceActionTypeEnum,
) (string, bool) {
	if !isWebAppFirewallPolicyWorkRequestResource(resource) {
		return "", false
	}
	if isWebAppFirewallPolicyIgnorableWorkRequestAction(resource.ActionType) {
		return "", false
	}
	if action != "" && resource.ActionType != action {
		return "", false
	}
	id := strings.TrimSpace(webAppFirewallPolicyStringValue(resource.Identifier))
	return id, id != ""
}

func resolveWebAppFirewallPolicyWorkRequestResourceAction(workRequest wafsdk.WorkRequest) (string, error) {
	var action string
	for _, resource := range workRequest.Resources {
		if !isWebAppFirewallPolicyWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" || isWebAppFirewallPolicyIgnorableWorkRequestAction(resource.ActionType) {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf(
				"WebAppFirewallPolicy work request %s exposes conflicting action types %q and %q",
				webAppFirewallPolicyStringValue(workRequest.Id),
				action,
				candidate,
			)
		}
	}
	return action, nil
}

func isWebAppFirewallPolicyIgnorableWorkRequestAction(action wafsdk.WorkRequestResourceActionTypeEnum) bool {
	return action == wafsdk.WorkRequestResourceActionTypeInProgress || action == wafsdk.WorkRequestResourceActionTypeRelated
}

func isWebAppFirewallPolicyWorkRequestResource(resource wafsdk.WorkRequestResource) bool {
	normalized := normalizeWebAppFirewallPolicyWorkRequestEntity(webAppFirewallPolicyStringValue(resource.EntityType))
	return normalized == "webappfirewallpolicy" ||
		normalized == "wafpolicy" ||
		strings.Contains(normalized, "webappfirewallpolicy") ||
		strings.Contains(normalized, "wafpolicy")
}

func normalizeWebAppFirewallPolicyWorkRequestEntity(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func webAppFirewallPolicyFromResponse(response any) (wafsdk.WebAppFirewallPolicy, bool) {
	if current, ok := webAppFirewallPolicyFromBodyOrSummary(response); ok {
		return current, true
	}
	return webAppFirewallPolicyFromOperationResponse(response)
}

func webAppFirewallPolicyFromBodyOrSummary(response any) (wafsdk.WebAppFirewallPolicy, bool) {
	switch current := response.(type) {
	case wafsdk.WebAppFirewallPolicy:
		return current, true
	case *wafsdk.WebAppFirewallPolicy:
		if current == nil {
			return wafsdk.WebAppFirewallPolicy{}, false
		}
		return *current, true
	case wafsdk.WebAppFirewallPolicySummary:
		return webAppFirewallPolicyFromSummary(current), true
	case *wafsdk.WebAppFirewallPolicySummary:
		if current == nil {
			return wafsdk.WebAppFirewallPolicy{}, false
		}
		return webAppFirewallPolicyFromSummary(*current), true
	default:
		return wafsdk.WebAppFirewallPolicy{}, false
	}
}

func webAppFirewallPolicyFromOperationResponse(response any) (wafsdk.WebAppFirewallPolicy, bool) {
	switch current := response.(type) {
	case wafsdk.CreateWebAppFirewallPolicyResponse:
		return current.WebAppFirewallPolicy, true
	case *wafsdk.CreateWebAppFirewallPolicyResponse:
		if current == nil {
			return wafsdk.WebAppFirewallPolicy{}, false
		}
		return current.WebAppFirewallPolicy, true
	case wafsdk.GetWebAppFirewallPolicyResponse:
		return current.WebAppFirewallPolicy, true
	case *wafsdk.GetWebAppFirewallPolicyResponse:
		if current == nil {
			return wafsdk.WebAppFirewallPolicy{}, false
		}
		return current.WebAppFirewallPolicy, true
	default:
		return wafsdk.WebAppFirewallPolicy{}, false
	}
}

func webAppFirewallPolicyFromSummary(summary wafsdk.WebAppFirewallPolicySummary) wafsdk.WebAppFirewallPolicy {
	return wafsdk.WebAppFirewallPolicy{
		Id:               summary.Id,
		DisplayName:      summary.DisplayName,
		CompartmentId:    summary.CompartmentId,
		TimeCreated:      summary.TimeCreated,
		LifecycleState:   summary.LifecycleState,
		FreeformTags:     cloneWebAppFirewallPolicyStringMap(summary.FreeformTags),
		DefinedTags:      cloneWebAppFirewallPolicyDefinedTags(summary.DefinedTags),
		SystemTags:       cloneWebAppFirewallPolicyDefinedTags(summary.SystemTags),
		TimeUpdated:      summary.TimeUpdated,
		LifecycleDetails: summary.LifecycleDetails,
	}
}

func webAppFirewallPolicyClientReady(client webAppFirewallPolicyOCIClient, initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initialize WebAppFirewallPolicy OCI client: %w", initErr)
	}
	if client == nil {
		return fmt.Errorf("WebAppFirewallPolicy OCI client is not configured")
	}
	return nil
}

func webAppFirewallPolicyDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		child := make(map[string]interface{}, len(values))
		for key, value := range values {
			child[key] = value
		}
		converted[namespace] = child
	}
	return converted
}

func webAppFirewallPolicyTagsToStatus(tags map[string]map[string]interface{}) map[string]shared.MapValue {
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

func cloneWebAppFirewallPolicyStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func webAppFirewallPolicyStringMapEqual(left map[string]string, right map[string]string) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}
	return reflect.DeepEqual(left, right)
}

func cloneWebAppFirewallPolicyDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	clone := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		child := make(map[string]interface{}, len(values))
		for key, value := range values {
			child[key] = value
		}
		clone[namespace] = child
	}
	return clone
}

func cloneWebAppFirewallPolicyStringSlice(source []string) []string {
	if source == nil {
		return nil
	}
	clone := make([]string, len(source))
	copy(clone, source)
	return clone
}

func webAppFirewallPolicyStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func webAppFirewallPolicyStringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(webAppFirewallPolicyStringValue(current)) == strings.TrimSpace(desired)
}

func webAppFirewallPolicyJSONEqual(left any, right any) bool {
	if webAppFirewallPolicyEmptyValue(left) && webAppFirewallPolicyEmptyValue(right) {
		return true
	}
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func webAppFirewallPolicyEmptyValue(value any) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return true
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Map:
		return rv.Len() == 0
	default:
		return false
	}
}
