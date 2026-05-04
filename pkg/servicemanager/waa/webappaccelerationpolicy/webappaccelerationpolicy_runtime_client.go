/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package webappaccelerationpolicy

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	waasdk "github.com/oracle/oci-go-sdk/v65/waa"
	waav1beta1 "github.com/oracle/oci-service-operator/api/waa/v1beta1"
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

var webAppAccelerationPolicyWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(waasdk.WorkRequestStatusAccepted),
		string(waasdk.WorkRequestStatusInProgress),
		string(waasdk.WorkRequestStatusCanceling),
	},
	SucceededStatusTokens: []string{string(waasdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(waasdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(waasdk.WorkRequestStatusCanceled)},
	CreateActionTokens: []string{
		string(waasdk.WorkRequestOperationTypeCreateWaaPolicy),
		string(waasdk.WorkRequestResourceActionTypeCreated),
	},
	UpdateActionTokens: []string{
		string(waasdk.WorkRequestOperationTypeUpdateWaaPolicy),
		string(waasdk.WorkRequestOperationTypeMoveWaaPolicy),
		string(waasdk.WorkRequestResourceActionTypeUpdated),
	},
	DeleteActionTokens: []string{
		string(waasdk.WorkRequestOperationTypeDeleteWaaPolicy),
		string(waasdk.WorkRequestResourceActionTypeDeleted),
	},
}

type webAppAccelerationPolicyOCIClient interface {
	ChangeWebAppAccelerationPolicyCompartment(context.Context, waasdk.ChangeWebAppAccelerationPolicyCompartmentRequest) (waasdk.ChangeWebAppAccelerationPolicyCompartmentResponse, error)
	CreateWebAppAccelerationPolicy(context.Context, waasdk.CreateWebAppAccelerationPolicyRequest) (waasdk.CreateWebAppAccelerationPolicyResponse, error)
	GetWebAppAccelerationPolicy(context.Context, waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error)
	ListWebAppAccelerationPolicies(context.Context, waasdk.ListWebAppAccelerationPoliciesRequest) (waasdk.ListWebAppAccelerationPoliciesResponse, error)
	UpdateWebAppAccelerationPolicy(context.Context, waasdk.UpdateWebAppAccelerationPolicyRequest) (waasdk.UpdateWebAppAccelerationPolicyResponse, error)
	DeleteWebAppAccelerationPolicy(context.Context, waasdk.DeleteWebAppAccelerationPolicyRequest) (waasdk.DeleteWebAppAccelerationPolicyResponse, error)
	GetWorkRequest(context.Context, waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error)
}

type webAppAccelerationPolicyCompartmentMoveClient interface {
	ChangeWebAppAccelerationPolicyCompartment(context.Context, waasdk.ChangeWebAppAccelerationPolicyCompartmentRequest) (waasdk.ChangeWebAppAccelerationPolicyCompartmentResponse, error)
	GetWorkRequest(context.Context, waasdk.GetWorkRequestRequest) (waasdk.GetWorkRequestResponse, error)
}

type webAppAccelerationPolicySDKCompartmentMoveClient struct {
	policy      waasdk.WaaClient
	workRequest waasdk.WorkRequestClient
}

type ambiguousWebAppAccelerationPolicyNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousWebAppAccelerationPolicyNotFoundError) Error() string {
	return e.message
}

func (e ambiguousWebAppAccelerationPolicyNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerWebAppAccelerationPolicyRuntimeHooksMutator(func(manager *WebAppAccelerationPolicyServiceManager, hooks *WebAppAccelerationPolicyRuntimeHooks) {
		moveClient, initErr := newWebAppAccelerationPolicyCompartmentMoveClient(manager)
		applyWebAppAccelerationPolicyRuntimeHooks(hooks, moveClient, initErr)
	})
}

func newWebAppAccelerationPolicyCompartmentMoveClient(
	manager *WebAppAccelerationPolicyServiceManager,
) (webAppAccelerationPolicyCompartmentMoveClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("WebAppAccelerationPolicy manager is nil")
	}
	policyClient, err := waasdk.NewWaaClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	workRequestClient, err := waasdk.NewWorkRequestClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return webAppAccelerationPolicySDKCompartmentMoveClient{policy: policyClient, workRequest: workRequestClient}, nil
}

func (c webAppAccelerationPolicySDKCompartmentMoveClient) ChangeWebAppAccelerationPolicyCompartment(
	ctx context.Context,
	request waasdk.ChangeWebAppAccelerationPolicyCompartmentRequest,
) (waasdk.ChangeWebAppAccelerationPolicyCompartmentResponse, error) {
	return c.policy.ChangeWebAppAccelerationPolicyCompartment(ctx, request)
}

func (c webAppAccelerationPolicySDKCompartmentMoveClient) GetWorkRequest(
	ctx context.Context,
	request waasdk.GetWorkRequestRequest,
) (waasdk.GetWorkRequestResponse, error) {
	return c.workRequest.GetWorkRequest(ctx, request)
}

func applyWebAppAccelerationPolicyRuntimeHooks(
	hooks *WebAppAccelerationPolicyRuntimeHooks,
	moveClient webAppAccelerationPolicyCompartmentMoveClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = webAppAccelerationPolicyRuntimeSemantics()
	hooks.BuildCreateBody = buildWebAppAccelerationPolicyCreateBody
	hooks.BuildUpdateBody = buildWebAppAccelerationPolicyUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardWebAppAccelerationPolicyExistingBeforeCreate
	hooks.List.Fields = webAppAccelerationPolicyListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = listWebAppAccelerationPoliciesAllPages(hooks.List.Call)
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedWebAppAccelerationPolicyIdentity
	hooks.ParityHooks.RequiresParityHandling = webAppAccelerationPolicyRequiresCompartmentMove
	hooks.ParityHooks.ApplyParityUpdate = func(
		ctx context.Context,
		resource *waav1beta1.WebAppAccelerationPolicy,
		currentResponse any,
	) (servicemanager.OSOKResponse, error) {
		return applyWebAppAccelerationPolicyCompartmentMove(ctx, resource, currentResponse, hooks.Get.Call, moveClient, initErr)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapWebAppAccelerationPolicyPendingMoveResume(
		hooks.Get.Call,
		moveClient,
		initErr,
	))
	hooks.DeleteHooks.HandleError = handleWebAppAccelerationPolicyDeleteError
	hooks.Async.Adapter = webAppAccelerationPolicyWorkRequestAsyncAdapter
	wrapWebAppAccelerationPolicyDeleteConfirmation(hooks)
}

func newWebAppAccelerationPolicyServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client webAppAccelerationPolicyOCIClient,
) WebAppAccelerationPolicyServiceClient {
	manager := &WebAppAccelerationPolicyServiceManager{Log: log}
	hooks := newWebAppAccelerationPolicyRuntimeHooksWithOCIClient(client)
	applyWebAppAccelerationPolicyRuntimeHooks(&hooks, client, nil)
	delegate := defaultWebAppAccelerationPolicyServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*waav1beta1.WebAppAccelerationPolicy](
			buildWebAppAccelerationPolicyGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapWebAppAccelerationPolicyGeneratedClient(hooks, delegate)
}

func newWebAppAccelerationPolicyRuntimeHooksWithOCIClient(client webAppAccelerationPolicyOCIClient) WebAppAccelerationPolicyRuntimeHooks {
	hooks := newWebAppAccelerationPolicyDefaultRuntimeHooks(waasdk.WaaClient{})
	hooks.Create.Call = func(ctx context.Context, request waasdk.CreateWebAppAccelerationPolicyRequest) (waasdk.CreateWebAppAccelerationPolicyResponse, error) {
		if client == nil {
			return waasdk.CreateWebAppAccelerationPolicyResponse{}, fmt.Errorf("WebAppAccelerationPolicy OCI client is not configured")
		}
		return client.CreateWebAppAccelerationPolicy(ctx, request)
	}
	hooks.Get.Call = func(ctx context.Context, request waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error) {
		if client == nil {
			return waasdk.GetWebAppAccelerationPolicyResponse{}, fmt.Errorf("WebAppAccelerationPolicy OCI client is not configured")
		}
		return client.GetWebAppAccelerationPolicy(ctx, request)
	}
	hooks.List.Call = func(ctx context.Context, request waasdk.ListWebAppAccelerationPoliciesRequest) (waasdk.ListWebAppAccelerationPoliciesResponse, error) {
		if client == nil {
			return waasdk.ListWebAppAccelerationPoliciesResponse{}, fmt.Errorf("WebAppAccelerationPolicy OCI client is not configured")
		}
		return client.ListWebAppAccelerationPolicies(ctx, request)
	}
	hooks.Update.Call = func(ctx context.Context, request waasdk.UpdateWebAppAccelerationPolicyRequest) (waasdk.UpdateWebAppAccelerationPolicyResponse, error) {
		if client == nil {
			return waasdk.UpdateWebAppAccelerationPolicyResponse{}, fmt.Errorf("WebAppAccelerationPolicy OCI client is not configured")
		}
		return client.UpdateWebAppAccelerationPolicy(ctx, request)
	}
	hooks.Delete.Call = func(ctx context.Context, request waasdk.DeleteWebAppAccelerationPolicyRequest) (waasdk.DeleteWebAppAccelerationPolicyResponse, error) {
		if client == nil {
			return waasdk.DeleteWebAppAccelerationPolicyResponse{}, fmt.Errorf("WebAppAccelerationPolicy OCI client is not configured")
		}
		return client.DeleteWebAppAccelerationPolicy(ctx, request)
	}
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		if client == nil {
			return nil, fmt.Errorf("WebAppAccelerationPolicy OCI client is not configured")
		}
		response, err := client.GetWorkRequest(ctx, waasdk.GetWorkRequestRequest{
			WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
		})
		if err != nil {
			return nil, err
		}
		return response.WorkRequest, nil
	}
	return hooks
}

func webAppAccelerationPolicyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "waa",
		FormalSlug:    "webappaccelerationpolicy",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(waasdk.WebAppAccelerationPolicyLifecycleStateCreating)},
			UpdatingStates:     []string{string(waasdk.WebAppAccelerationPolicyLifecycleStateUpdating)},
			ActiveStates:       []string{string(waasdk.WebAppAccelerationPolicyLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(waasdk.WebAppAccelerationPolicyLifecycleStateCreating),
				string(waasdk.WebAppAccelerationPolicyLifecycleStateUpdating),
				string(waasdk.WebAppAccelerationPolicyLifecycleStateDeleting),
			},
			TerminalStates: []string{string(waasdk.WebAppAccelerationPolicyLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{
				"compartmentId",
				"displayName",
				"responseCachingPolicy",
				"responseCompressionPolicy",
				"freeformTags",
				"definedTags",
				"systemTags",
			},
			Mutable: []string{
				"compartmentId",
				"displayName",
				"responseCachingPolicy",
				"responseCompressionPolicy",
				"freeformTags",
				"definedTags",
				"systemTags",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "WebAppAccelerationPolicy", Action: "CreateWebAppAccelerationPolicy"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "WebAppAccelerationPolicy", Action: "UpdateWebAppAccelerationPolicy"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "WebAppAccelerationPolicy", Action: "DeleteWebAppAccelerationPolicy"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "WebAppAccelerationPolicy", Action: "GetWebAppAccelerationPolicy"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "WebAppAccelerationPolicy", Action: "GetWebAppAccelerationPolicy"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "WebAppAccelerationPolicy", Action: "GetWebAppAccelerationPolicy"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func webAppAccelerationPolicyListFields() []generatedruntime.RequestField {
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
			FieldName:        "Id",
			RequestName:      "id",
			Contribution:     "query",
			PreferResourceID: true,
			LookupPaths:      []string{"status.id", "status.status.ocid", "id", "ocid"},
		},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func guardWebAppAccelerationPolicyExistingBeforeCreate(
	_ context.Context,
	resource *waav1beta1.WebAppAccelerationPolicy,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("WebAppAccelerationPolicy resource is nil")
	}
	if trackedWebAppAccelerationPolicyID(resource) != "" {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildWebAppAccelerationPolicyCreateBody(
	_ context.Context,
	resource *waav1beta1.WebAppAccelerationPolicy,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("WebAppAccelerationPolicy resource is nil")
	}
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return nil, fmt.Errorf("compartmentId is required")
	}

	spec := resource.Spec
	body := waasdk.CreateWebAppAccelerationPolicyDetails{
		CompartmentId: common.String(strings.TrimSpace(spec.CompartmentId)),
	}
	if value := strings.TrimSpace(spec.DisplayName); value != "" {
		body.DisplayName = common.String(value)
	}
	if policy := webAppAccelerationPolicyCreateCachingPolicy(spec.ResponseCachingPolicy); policy != nil {
		body.ResponseCachingPolicy = policy
	}
	if policy := webAppAccelerationPolicyCreateCompressionPolicy(spec.ResponseCompressionPolicy); policy != nil {
		body.ResponseCompressionPolicy = policy
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneWebAppAccelerationPolicyStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = webAppAccelerationPolicyDefinedTags(spec.DefinedTags)
	}
	if spec.SystemTags != nil {
		body.SystemTags = webAppAccelerationPolicyDefinedTags(spec.SystemTags)
	}
	return body, nil
}

func buildWebAppAccelerationPolicyUpdateBody(
	_ context.Context,
	resource *waav1beta1.WebAppAccelerationPolicy,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return waasdk.UpdateWebAppAccelerationPolicyDetails{}, false, fmt.Errorf("WebAppAccelerationPolicy resource is nil")
	}
	current, ok := webAppAccelerationPolicyFromResponse(currentResponse)
	if !ok {
		return waasdk.UpdateWebAppAccelerationPolicyDetails{}, false, fmt.Errorf("current WebAppAccelerationPolicy response does not expose a WebAppAccelerationPolicy body")
	}

	body := waasdk.UpdateWebAppAccelerationPolicyDetails{}
	updateNeeded := webAppAccelerationPolicyCompartmentNeedsMove(resource.Spec, current)
	if displayName, ok := webAppAccelerationPolicyDisplayNameUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		body.DisplayName = displayName
		updateNeeded = true
	}
	if policy, ok := webAppAccelerationPolicyCachingPolicyUpdate(resource.Spec.ResponseCachingPolicy, current.ResponseCachingPolicy); ok {
		body.ResponseCachingPolicy = policy
		updateNeeded = true
	}
	if policy, ok := webAppAccelerationPolicyCompressionPolicyUpdate(resource.Spec.ResponseCompressionPolicy, current.ResponseCompressionPolicy); ok {
		body.ResponseCompressionPolicy = policy
		updateNeeded = true
	}
	if webAppAccelerationPolicyApplyTagUpdates(&body, resource.Spec, current) {
		updateNeeded = true
	}
	return body, updateNeeded, nil
}

func webAppAccelerationPolicyDisplayNameUpdate(desired string, current *string) (*string, bool) {
	desired = strings.TrimSpace(desired)
	if desired == "" || webAppAccelerationPolicyStringPtrEqual(current, desired) {
		return nil, false
	}
	return common.String(desired), true
}

func webAppAccelerationPolicyCreateCachingPolicy(
	spec waav1beta1.WebAppAccelerationPolicyResponseCachingPolicy,
) *waasdk.ResponseCachingPolicy {
	if !spec.IsResponseHeaderBasedCachingEnabled {
		return nil
	}
	return &waasdk.ResponseCachingPolicy{
		IsResponseHeaderBasedCachingEnabled: common.Bool(spec.IsResponseHeaderBasedCachingEnabled),
	}
}

func webAppAccelerationPolicyCachingPolicyUpdate(
	spec waav1beta1.WebAppAccelerationPolicyResponseCachingPolicy,
	current *waasdk.ResponseCachingPolicy,
) (*waasdk.ResponseCachingPolicy, bool) {
	desired := spec.IsResponseHeaderBasedCachingEnabled
	if current == nil || current.IsResponseHeaderBasedCachingEnabled == nil {
		if !desired {
			return nil, false
		}
		return &waasdk.ResponseCachingPolicy{IsResponseHeaderBasedCachingEnabled: common.Bool(desired)}, true
	}
	if *current.IsResponseHeaderBasedCachingEnabled == desired {
		return nil, false
	}
	return &waasdk.ResponseCachingPolicy{IsResponseHeaderBasedCachingEnabled: common.Bool(desired)}, true
}

func webAppAccelerationPolicyCreateCompressionPolicy(
	spec waav1beta1.WebAppAccelerationPolicyResponseCompressionPolicy,
) *waasdk.ResponseCompressionPolicy {
	if !spec.GzipCompression.IsEnabled {
		return nil
	}
	return &waasdk.ResponseCompressionPolicy{
		GzipCompression: &waasdk.GzipCompressionPolicy{IsEnabled: common.Bool(spec.GzipCompression.IsEnabled)},
	}
}

func webAppAccelerationPolicyCompressionPolicyUpdate(
	spec waav1beta1.WebAppAccelerationPolicyResponseCompressionPolicy,
	current *waasdk.ResponseCompressionPolicy,
) (*waasdk.ResponseCompressionPolicy, bool) {
	desired := spec.GzipCompression.IsEnabled
	if current == nil || current.GzipCompression == nil || current.GzipCompression.IsEnabled == nil {
		if !desired {
			return nil, false
		}
		return &waasdk.ResponseCompressionPolicy{
			GzipCompression: &waasdk.GzipCompressionPolicy{IsEnabled: common.Bool(desired)},
		}, true
	}
	if *current.GzipCompression.IsEnabled == desired {
		return nil, false
	}
	return &waasdk.ResponseCompressionPolicy{
		GzipCompression: &waasdk.GzipCompressionPolicy{IsEnabled: common.Bool(desired)},
	}, true
}

func webAppAccelerationPolicyApplyTagUpdates(
	body *waasdk.UpdateWebAppAccelerationPolicyDetails,
	spec waav1beta1.WebAppAccelerationPolicySpec,
	current waasdk.WebAppAccelerationPolicy,
) bool {
	updateNeeded := false

	if spec.FreeformTags != nil {
		desiredFreeformTags := cloneWebAppAccelerationPolicyStringMap(spec.FreeformTags)
		if !webAppAccelerationPolicyStringMapsEqual(current.FreeformTags, desiredFreeformTags) {
			body.FreeformTags = desiredFreeformTags
			updateNeeded = true
		}
	}

	if spec.DefinedTags != nil {
		desiredDefinedTags := webAppAccelerationPolicyDefinedTags(spec.DefinedTags)
		if !webAppAccelerationPolicyDefinedTagMapsEqual(current.DefinedTags, desiredDefinedTags) {
			body.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}

	if spec.SystemTags != nil {
		desiredSystemTags := webAppAccelerationPolicyDefinedTags(spec.SystemTags)
		if !webAppAccelerationPolicyDefinedTagMapsEqual(current.SystemTags, desiredSystemTags) {
			body.SystemTags = desiredSystemTags
			updateNeeded = true
		}
	}
	return updateNeeded
}

func webAppAccelerationPolicyStringMapsEqual(current, desired map[string]string) bool {
	if len(current) == 0 && len(desired) == 0 {
		return true
	}
	return reflect.DeepEqual(current, desired)
}

func webAppAccelerationPolicyDefinedTagMapsEqual(
	current map[string]map[string]interface{},
	desired map[string]map[string]interface{},
) bool {
	if len(current) == 0 && len(desired) == 0 {
		return true
	}
	return reflect.DeepEqual(current, desired)
}

func webAppAccelerationPolicyRequiresCompartmentMove(
	resource *waav1beta1.WebAppAccelerationPolicy,
	currentResponse any,
) bool {
	if resource == nil {
		return false
	}
	current, ok := webAppAccelerationPolicyFromResponse(currentResponse)
	if !ok {
		return false
	}
	return webAppAccelerationPolicyCompartmentNeedsMove(resource.Spec, current)
}

func webAppAccelerationPolicyCompartmentNeedsMove(
	spec waav1beta1.WebAppAccelerationPolicySpec,
	current waasdk.WebAppAccelerationPolicy,
) bool {
	desired := strings.TrimSpace(spec.CompartmentId)
	observed := strings.TrimSpace(webAppAccelerationPolicyStringValue(current.CompartmentId))
	return desired != "" && observed != "" && desired != observed
}

func applyWebAppAccelerationPolicyCompartmentMove(
	ctx context.Context,
	resource *waav1beta1.WebAppAccelerationPolicy,
	currentResponse any,
	getPolicy func(context.Context, waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error),
	client webAppAccelerationPolicyCompartmentMoveClient,
	initErr error,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppAccelerationPolicy resource is nil")
	}
	if initErr != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("initialize WebAppAccelerationPolicy compartment move client: %w", initErr)
	}
	if client == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppAccelerationPolicy compartment move client is not configured")
	}
	policyID, compartmentID, err := webAppAccelerationPolicyCompartmentMoveInputs(resource, currentResponse)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	if workRequestID := pendingWebAppAccelerationPolicyMoveWorkRequest(resource); workRequestID != "" {
		return followWebAppAccelerationPolicyCompartmentMoveWorkRequest(ctx, resource, policyID, workRequestID, getPolicy, client)
	}

	response, err := client.ChangeWebAppAccelerationPolicyCompartment(ctx, waasdk.ChangeWebAppAccelerationPolicyCompartmentRequest{
		WebAppAccelerationPolicyId: common.String(policyID),
		ChangeWebAppAccelerationPolicyCompartmentDetails: waasdk.ChangeWebAppAccelerationPolicyCompartmentDetails{
			CompartmentId: common.String(compartmentID),
		},
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("change WebAppAccelerationPolicy compartment: %w", err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	workRequestID := strings.TrimSpace(webAppAccelerationPolicyStringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppAccelerationPolicy compartment move did not return an opc-work-request-id")
	}

	resource.Status.Id = policyID
	resource.Status.OsokStatus.Ocid = shared.OCID(policyID)
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:    workRequestID,
		RawOperationType: string(waasdk.WorkRequestOperationTypeMoveWaaPolicy),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		Message:          fmt.Sprintf("WebAppAccelerationPolicy compartment move work request %s is pending", workRequestID),
	}, loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{
		IsSuccessful:  projection.Condition != shared.Failed,
		ShouldRequeue: projection.ShouldRequeue,
	}, nil
}

func webAppAccelerationPolicyCompartmentMoveInputs(
	resource *waav1beta1.WebAppAccelerationPolicy,
	currentResponse any,
) (string, string, error) {
	current, ok := webAppAccelerationPolicyFromResponse(currentResponse)
	if !ok {
		return "", "", fmt.Errorf("current WebAppAccelerationPolicy response does not expose a WebAppAccelerationPolicy body")
	}
	policyID := strings.TrimSpace(webAppAccelerationPolicyStringValue(current.Id))
	if policyID == "" {
		policyID = trackedWebAppAccelerationPolicyID(resource)
	}
	if policyID == "" {
		return "", "", fmt.Errorf("WebAppAccelerationPolicy compartment move requires a tracked policy OCID")
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		return "", "", fmt.Errorf("WebAppAccelerationPolicy compartment move requires spec.compartmentId")
	}
	return policyID, compartmentID, nil
}

func pendingWebAppAccelerationPolicyMoveWorkRequest(resource *waav1beta1.WebAppAccelerationPolicy) string {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest ||
		current.Phase != shared.OSOKAsyncPhaseUpdate ||
		current.NormalizedClass != shared.OSOKAsyncClassPending {
		return ""
	}
	return strings.TrimSpace(current.WorkRequestID)
}

func followWebAppAccelerationPolicyCompartmentMoveWorkRequest(
	ctx context.Context,
	resource *waav1beta1.WebAppAccelerationPolicy,
	policyID string,
	workRequestID string,
	getPolicy func(context.Context, waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error),
	client webAppAccelerationPolicyCompartmentMoveClient,
) (servicemanager.OSOKResponse, error) {
	response, err := client.GetWorkRequest(ctx, waasdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("get WebAppAccelerationPolicy compartment move work request: %w", err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	current, err := webAppAccelerationPolicyWorkRequestAsyncOperation(&resource.Status.OsokStatus, response.WorkRequest, shared.OSOKAsyncPhaseUpdate)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, loggerutil.OSOKLogger{})

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: projection.ShouldRequeue}, nil
	case shared.OSOKAsyncClassSucceeded:
		return readWebAppAccelerationPolicyAfterCompartmentMove(ctx, resource, policyID, workRequestID, getPolicy)
	default:
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf(
			"WebAppAccelerationPolicy compartment move work request %s finished with status %s",
			strings.TrimSpace(workRequestID),
			current.RawStatus,
		)
	}
}

func webAppAccelerationPolicyWorkRequestAsyncOperation(
	status *shared.OSOKStatus,
	workRequest waasdk.WorkRequest,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	return servicemanager.BuildWorkRequestAsyncOperation(status, webAppAccelerationPolicyWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.Status),
		RawAction:        string(workRequest.OperationType),
		RawOperationType: string(workRequest.OperationType),
		WorkRequestID:    webAppAccelerationPolicyStringValue(workRequest.Id),
		PercentComplete:  workRequest.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
}

func readWebAppAccelerationPolicyAfterCompartmentMove(
	ctx context.Context,
	resource *waav1beta1.WebAppAccelerationPolicy,
	policyID string,
	workRequestID string,
	getPolicy func(context.Context, waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error),
) (servicemanager.OSOKResponse, error) {
	if getPolicy == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppAccelerationPolicy Get call is not configured")
	}
	response, err := getPolicy(ctx, waasdk.GetWebAppAccelerationPolicyRequest{
		WebAppAccelerationPolicyId: common.String(policyID),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("read WebAppAccelerationPolicy after compartment move: %w", err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	projectWebAppAccelerationPolicyStatus(resource, response.WebAppAccelerationPolicy)
	if webAppAccelerationPolicyCompartmentNeedsMove(resource.Spec, response.WebAppAccelerationPolicy) {
		return webAppAccelerationPolicyMoveReadbackPending(resource, workRequestID), nil
	}
	return webAppAccelerationPolicyLifecycleResponse(resource, response.WebAppAccelerationPolicy, "WebAppAccelerationPolicy compartment move completed"), nil
}

func webAppAccelerationPolicyMoveReadbackPending(
	resource *waav1beta1.WebAppAccelerationPolicy,
	workRequestID string,
) servicemanager.OSOKResponse {
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:    strings.TrimSpace(workRequestID),
		RawStatus:        string(waasdk.WorkRequestStatusSucceeded),
		RawOperationType: string(waasdk.WorkRequestOperationTypeMoveWaaPolicy),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		Message:          fmt.Sprintf("WebAppAccelerationPolicy compartment move work request %s succeeded; waiting for target compartment readback", strings.TrimSpace(workRequestID)),
	}, loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{
		IsSuccessful:  projection.Condition != shared.Failed,
		ShouldRequeue: projection.ShouldRequeue,
	}
}

func wrapWebAppAccelerationPolicyPendingMoveResume(
	getPolicy func(context.Context, waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error),
	client webAppAccelerationPolicyCompartmentMoveClient,
	initErr error,
) func(WebAppAccelerationPolicyServiceClient) WebAppAccelerationPolicyServiceClient {
	return func(delegate WebAppAccelerationPolicyServiceClient) WebAppAccelerationPolicyServiceClient {
		return webAppAccelerationPolicyPendingMoveResumeClient{
			delegate:  delegate,
			getPolicy: getPolicy,
			client:    client,
			initErr:   initErr,
		}
	}
}

type webAppAccelerationPolicyPendingMoveResumeClient struct {
	delegate  WebAppAccelerationPolicyServiceClient
	getPolicy func(context.Context, waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error)
	client    webAppAccelerationPolicyCompartmentMoveClient
	initErr   error
}

func (c webAppAccelerationPolicyPendingMoveResumeClient) CreateOrUpdate(
	ctx context.Context,
	resource *waav1beta1.WebAppAccelerationPolicy,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppAccelerationPolicy runtime client is not configured")
	}
	if workRequestID := pendingWebAppAccelerationPolicyMoveWorkRequest(resource); workRequestID != "" {
		if c.initErr != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("initialize WebAppAccelerationPolicy compartment move client: %w", c.initErr)
		}
		if c.client == nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppAccelerationPolicy compartment move client is not configured")
		}
		policyID := trackedWebAppAccelerationPolicyID(resource)
		if policyID == "" {
			return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppAccelerationPolicy pending compartment move requires a tracked policy OCID")
		}
		return followWebAppAccelerationPolicyCompartmentMoveWorkRequest(ctx, resource, policyID, workRequestID, c.getPolicy, c.client)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c webAppAccelerationPolicyPendingMoveResumeClient) Delete(
	ctx context.Context,
	resource *waav1beta1.WebAppAccelerationPolicy,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("WebAppAccelerationPolicy runtime client is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func listWebAppAccelerationPoliciesAllPages(
	call func(context.Context, waasdk.ListWebAppAccelerationPoliciesRequest) (waasdk.ListWebAppAccelerationPoliciesResponse, error),
) func(context.Context, waasdk.ListWebAppAccelerationPoliciesRequest) (waasdk.ListWebAppAccelerationPoliciesResponse, error) {
	return func(ctx context.Context, request waasdk.ListWebAppAccelerationPoliciesRequest) (waasdk.ListWebAppAccelerationPoliciesResponse, error) {
		if !webAppAccelerationPolicyListHasIdentityFilter(request) {
			return waasdk.ListWebAppAccelerationPoliciesResponse{}, nil
		}

		var combined waasdk.ListWebAppAccelerationPoliciesResponse
		for {
			response, err := call(ctx, request)
			if err != nil {
				return waasdk.ListWebAppAccelerationPoliciesResponse{}, err
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

func webAppAccelerationPolicyListHasIdentityFilter(request waasdk.ListWebAppAccelerationPoliciesRequest) bool {
	return request.Id != nil && strings.TrimSpace(*request.Id) != "" ||
		request.DisplayName != nil && strings.TrimSpace(*request.DisplayName) != ""
}

func handleWebAppAccelerationPolicyDeleteError(resource *waav1beta1.WebAppAccelerationPolicy, err error) error {
	err = conservativeWebAppAccelerationPolicyNotFoundError(err, "delete")
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func conservativeWebAppAccelerationPolicyNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("WebAppAccelerationPolicy %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousWebAppAccelerationPolicyNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousWebAppAccelerationPolicyNotFoundError{message: message, opcRequestID: errorutil.OpcRequestID(err)}
}

func isAmbiguousWebAppAccelerationPolicyNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ambiguous ambiguousWebAppAccelerationPolicyNotFoundError
	if errors.As(err, &ambiguous) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound()
}

func wrapWebAppAccelerationPolicyDeleteConfirmation(hooks *WebAppAccelerationPolicyRuntimeHooks) {
	if hooks == nil || hooks.Get.Call == nil {
		return
	}
	getWebAppAccelerationPolicy := hooks.Get.Call
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate WebAppAccelerationPolicyServiceClient) WebAppAccelerationPolicyServiceClient {
		return webAppAccelerationPolicyDeleteGuardClient{
			delegate:                    delegate,
			getWebAppAccelerationPolicy: getWebAppAccelerationPolicy,
		}
	})
}

type webAppAccelerationPolicyDeleteGuardClient struct {
	delegate                    WebAppAccelerationPolicyServiceClient
	getWebAppAccelerationPolicy func(context.Context, waasdk.GetWebAppAccelerationPolicyRequest) (waasdk.GetWebAppAccelerationPolicyResponse, error)
}

func (c webAppAccelerationPolicyDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *waav1beta1.WebAppAccelerationPolicy,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("WebAppAccelerationPolicy runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c webAppAccelerationPolicyDeleteGuardClient) Delete(
	ctx context.Context,
	resource *waav1beta1.WebAppAccelerationPolicy,
) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("WebAppAccelerationPolicy runtime client is not configured")
	}
	if err := c.rejectAuthShapedConfirmRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c webAppAccelerationPolicyDeleteGuardClient) rejectAuthShapedConfirmRead(
	ctx context.Context,
	resource *waav1beta1.WebAppAccelerationPolicy,
) error {
	if c.getWebAppAccelerationPolicy == nil || resource == nil {
		return nil
	}
	policyID := trackedWebAppAccelerationPolicyID(resource)
	if policyID == "" {
		return nil
	}
	_, err := c.getWebAppAccelerationPolicy(ctx, waasdk.GetWebAppAccelerationPolicyRequest{
		WebAppAccelerationPolicyId: common.String(policyID),
	})
	if err == nil || !isAmbiguousWebAppAccelerationPolicyNotFound(err) {
		return nil
	}
	err = conservativeWebAppAccelerationPolicyNotFoundError(err, "pre-delete read")
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	return err
}

func clearTrackedWebAppAccelerationPolicyIdentity(resource *waav1beta1.WebAppAccelerationPolicy) {
	if resource == nil {
		return
	}
	resource.Status = waav1beta1.WebAppAccelerationPolicyStatus{}
}

func trackedWebAppAccelerationPolicyID(resource *waav1beta1.WebAppAccelerationPolicy) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func webAppAccelerationPolicyFromResponse(response any) (waasdk.WebAppAccelerationPolicy, bool) {
	if current, ok := webAppAccelerationPolicyMutationResponse(response); ok {
		return current, true
	}
	if current, ok := webAppAccelerationPolicyReadResponse(response); ok {
		return current, true
	}
	return webAppAccelerationPolicySummaryResponse(response)
}

func webAppAccelerationPolicyMutationResponse(response any) (waasdk.WebAppAccelerationPolicy, bool) {
	switch current := response.(type) {
	case waasdk.CreateWebAppAccelerationPolicyResponse:
		return current.WebAppAccelerationPolicy, true
	case *waasdk.CreateWebAppAccelerationPolicyResponse:
		if current == nil {
			return waasdk.WebAppAccelerationPolicy{}, false
		}
		return current.WebAppAccelerationPolicy, true
	default:
		return waasdk.WebAppAccelerationPolicy{}, false
	}
}

func webAppAccelerationPolicyReadResponse(response any) (waasdk.WebAppAccelerationPolicy, bool) {
	switch current := response.(type) {
	case waasdk.GetWebAppAccelerationPolicyResponse:
		return current.WebAppAccelerationPolicy, true
	case *waasdk.GetWebAppAccelerationPolicyResponse:
		if current == nil {
			return waasdk.WebAppAccelerationPolicy{}, false
		}
		return current.WebAppAccelerationPolicy, true
	case waasdk.WebAppAccelerationPolicy:
		return current, true
	case *waasdk.WebAppAccelerationPolicy:
		if current == nil {
			return waasdk.WebAppAccelerationPolicy{}, false
		}
		return *current, true
	default:
		return waasdk.WebAppAccelerationPolicy{}, false
	}
}

func webAppAccelerationPolicySummaryResponse(response any) (waasdk.WebAppAccelerationPolicy, bool) {
	switch current := response.(type) {
	case waasdk.WebAppAccelerationPolicySummary:
		return webAppAccelerationPolicyFromSummary(current), true
	case *waasdk.WebAppAccelerationPolicySummary:
		if current == nil {
			return waasdk.WebAppAccelerationPolicy{}, false
		}
		return webAppAccelerationPolicyFromSummary(*current), true
	default:
		return waasdk.WebAppAccelerationPolicy{}, false
	}
}

func webAppAccelerationPolicyFromSummary(summary waasdk.WebAppAccelerationPolicySummary) waasdk.WebAppAccelerationPolicy {
	return waasdk.WebAppAccelerationPolicy{
		Id:               summary.Id,
		DisplayName:      summary.DisplayName,
		CompartmentId:    summary.CompartmentId,
		TimeCreated:      summary.TimeCreated,
		LifecycleState:   summary.LifecycleState,
		FreeformTags:     cloneWebAppAccelerationPolicyStringMap(summary.FreeformTags),
		DefinedTags:      cloneWebAppAccelerationPolicyDefinedTags(summary.DefinedTags),
		SystemTags:       cloneWebAppAccelerationPolicyDefinedTags(summary.SystemTags),
		TimeUpdated:      summary.TimeUpdated,
		LifecycleDetails: summary.LifecycleDetails,
	}
}

func projectWebAppAccelerationPolicyStatus(
	resource *waav1beta1.WebAppAccelerationPolicy,
	current waasdk.WebAppAccelerationPolicy,
) {
	if resource == nil {
		return
	}
	resource.Status.Id = webAppAccelerationPolicyStringValue(current.Id)
	resource.Status.DisplayName = webAppAccelerationPolicyStringValue(current.DisplayName)
	resource.Status.CompartmentId = webAppAccelerationPolicyStringValue(current.CompartmentId)
	resource.Status.TimeCreated = webAppAccelerationPolicySDKTimeString(current.TimeCreated)
	resource.Status.LifecycleState = string(current.LifecycleState)
	resource.Status.FreeformTags = cloneWebAppAccelerationPolicyStringMap(current.FreeformTags)
	resource.Status.DefinedTags = webAppAccelerationPolicyStatusDefinedTags(current.DefinedTags)
	resource.Status.SystemTags = webAppAccelerationPolicyStatusDefinedTags(current.SystemTags)
	resource.Status.TimeUpdated = webAppAccelerationPolicySDKTimeString(current.TimeUpdated)
	resource.Status.LifecycleDetails = webAppAccelerationPolicyStringValue(current.LifecycleDetails)
	resource.Status.ResponseCachingPolicy = webAppAccelerationPolicyStatusCachingPolicy(current.ResponseCachingPolicy)
	resource.Status.ResponseCompressionPolicy = webAppAccelerationPolicyStatusCompressionPolicy(current.ResponseCompressionPolicy)
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
}

func webAppAccelerationPolicyLifecycleResponse(
	resource *waav1beta1.WebAppAccelerationPolicy,
	current waasdk.WebAppAccelerationPolicy,
	activeMessage string,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	state := strings.ToUpper(string(current.LifecycleState))
	switch state {
	case string(waasdk.WebAppAccelerationPolicyLifecycleStateCreating), string(waasdk.WebAppAccelerationPolicyLifecycleStateUpdating):
		projection := servicemanager.ApplyAsyncOperation(status, &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseUpdate,
			RawStatus:       state,
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         fmt.Sprintf("WebAppAccelerationPolicy lifecycle state %s is pending", state),
		}, loggerutil.OSOKLogger{})
		return servicemanager.OSOKResponse{IsSuccessful: projection.Condition != shared.Failed, ShouldRequeue: projection.ShouldRequeue}
	case string(waasdk.WebAppAccelerationPolicyLifecycleStateDeleted):
		projection := servicemanager.ApplyAsyncOperation(status, &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseDelete,
			RawStatus:       state,
			NormalizedClass: shared.OSOKAsyncClassSucceeded,
			Message:         "WebAppAccelerationPolicy lifecycle state DELETED",
		}, loggerutil.OSOKLogger{})
		return servicemanager.OSOKResponse{IsSuccessful: projection.Condition != shared.Failed, ShouldRequeue: projection.ShouldRequeue}
	case string(waasdk.WebAppAccelerationPolicyLifecycleStateFailed):
		projection := servicemanager.ApplyAsyncOperation(status, &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseUpdate,
			RawStatus:       state,
			NormalizedClass: shared.OSOKAsyncClassFailed,
			Message:         webAppAccelerationPolicyFailureMessage(current),
		}, loggerutil.OSOKLogger{})
		return servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: projection.ShouldRequeue}
	default:
		servicemanager.ClearAsyncOperation(status)
		now := metav1.Now()
		status.UpdatedAt = &now
		status.Message = activeMessage
		status.Reason = string(shared.Active)
		*status = util.UpdateOSOKStatusCondition(*status, shared.Active, corev1.ConditionTrue, "", activeMessage, loggerutil.OSOKLogger{})
		return servicemanager.OSOKResponse{IsSuccessful: true}
	}
}

func webAppAccelerationPolicyFailureMessage(current waasdk.WebAppAccelerationPolicy) string {
	if details := strings.TrimSpace(webAppAccelerationPolicyStringValue(current.LifecycleDetails)); details != "" {
		return details
	}
	return "WebAppAccelerationPolicy lifecycle state FAILED"
}

func webAppAccelerationPolicySDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func webAppAccelerationPolicyStatusCachingPolicy(
	current *waasdk.ResponseCachingPolicy,
) waav1beta1.WebAppAccelerationPolicyResponseCachingPolicy {
	if current == nil || current.IsResponseHeaderBasedCachingEnabled == nil {
		return waav1beta1.WebAppAccelerationPolicyResponseCachingPolicy{}
	}
	return waav1beta1.WebAppAccelerationPolicyResponseCachingPolicy{
		IsResponseHeaderBasedCachingEnabled: *current.IsResponseHeaderBasedCachingEnabled,
	}
}

func webAppAccelerationPolicyStatusCompressionPolicy(
	current *waasdk.ResponseCompressionPolicy,
) waav1beta1.WebAppAccelerationPolicyResponseCompressionPolicy {
	if current == nil || current.GzipCompression == nil || current.GzipCompression.IsEnabled == nil {
		return waav1beta1.WebAppAccelerationPolicyResponseCompressionPolicy{}
	}
	return waav1beta1.WebAppAccelerationPolicyResponseCompressionPolicy{
		GzipCompression: waav1beta1.WebAppAccelerationPolicyResponseCompressionPolicyGzipCompression{
			IsEnabled: *current.GzipCompression.IsEnabled,
		},
	}
}

func webAppAccelerationPolicyStatusDefinedTags(source map[string]map[string]interface{}) map[string]shared.MapValue {
	if source == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(source))
	for namespace, values := range source {
		child := make(shared.MapValue, len(values))
		for key, value := range values {
			child[key] = fmt.Sprint(value)
		}
		converted[namespace] = child
	}
	return converted
}

func webAppAccelerationPolicyDefinedTags(source map[string]shared.MapValue) map[string]map[string]interface{} {
	return *util.ConvertToOciDefinedTags(&source)
}

func cloneWebAppAccelerationPolicyStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneWebAppAccelerationPolicyDefinedTags(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		if values == nil {
			cloned[namespace] = nil
			continue
		}
		clonedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			clonedValues[key] = value
		}
		cloned[namespace] = clonedValues
	}
	return cloned
}

func webAppAccelerationPolicyStringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(webAppAccelerationPolicyStringValue(current)) == strings.TrimSpace(desired)
}

func webAppAccelerationPolicyStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
