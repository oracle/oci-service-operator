/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package awrhub

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	opsisdk "github.com/oracle/oci-go-sdk/v65/opsi"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type awrHubOCIClient interface {
	CreateAwrHub(context.Context, opsisdk.CreateAwrHubRequest) (opsisdk.CreateAwrHubResponse, error)
	GetAwrHub(context.Context, opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error)
	ListAwrHubs(context.Context, opsisdk.ListAwrHubsRequest) (opsisdk.ListAwrHubsResponse, error)
	UpdateAwrHub(context.Context, opsisdk.UpdateAwrHubRequest) (opsisdk.UpdateAwrHubResponse, error)
	DeleteAwrHub(context.Context, opsisdk.DeleteAwrHubRequest) (opsisdk.DeleteAwrHubResponse, error)
	GetWorkRequest(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

type ambiguousAwrHubNotFoundError struct {
	message      string
	opcRequestID string
}

type awrHubAmbiguousDeleteConfirmResponse struct {
	AwrHub opsisdk.AwrHub
	err    error
}

func (e ambiguousAwrHubNotFoundError) Error() string {
	return e.message
}

func (e ambiguousAwrHubNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

var awrHubWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
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

func init() {
	registerAwrHubRuntimeHooksMutator(func(manager *AwrHubServiceManager, hooks *AwrHubRuntimeHooks) {
		client, initErr := newAwrHubOperationsInsightsClient(manager)
		applyAwrHubRuntimeHooks(hooks, client, initErr)
	})
}

func newAwrHubOperationsInsightsClient(manager *AwrHubServiceManager) (awrHubOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("awrhub service manager is nil")
	}
	client, err := opsisdk.NewOperationsInsightsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyAwrHubRuntimeHooks(
	hooks *AwrHubRuntimeHooks,
	client awrHubOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = awrHubRuntimeSemantics()
	hooks.BuildCreateBody = buildAwrHubCreateBody
	hooks.BuildUpdateBody = buildAwrHubUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardAwrHubExistingBeforeCreate
	hooks.List.Fields = awrHubListFields()
	wrapAwrHubListCalls(hooks)
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedAwrHubIdentity
	hooks.StatusHooks.ProjectStatus = awrHubStatusFromResponse
	hooks.DeleteHooks.ConfirmRead = awrHubDeleteConfirmRead(hooks.Get.Call, hooks.List.Call)
	hooks.DeleteHooks.HandleError = handleAwrHubDeleteError
	hooks.DeleteHooks.ApplyOutcome = applyAwrHubDeleteConfirmOutcome
	hooks.Async.Adapter = awrHubWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getAwrHubWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveAwrHubWorkRequestAction
	hooks.Async.ResolvePhase = resolveAwrHubWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverAwrHubIDFromWorkRequest
	hooks.Async.Message = awrHubWorkRequestMessage
	wrapAwrHubDeleteGuardClient(hooks)
}

func newAwrHubRuntimeHooksWithOCIClient(client awrHubOCIClient) AwrHubRuntimeHooks {
	return AwrHubRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*opsiv1beta1.AwrHub]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*opsiv1beta1.AwrHub]{},
		StatusHooks:     generatedruntime.StatusHooks[*opsiv1beta1.AwrHub]{},
		ParityHooks:     generatedruntime.ParityHooks[*opsiv1beta1.AwrHub]{},
		Async:           generatedruntime.AsyncHooks[*opsiv1beta1.AwrHub]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*opsiv1beta1.AwrHub]{},
		Create: runtimeOperationHooks[opsisdk.CreateAwrHubRequest, opsisdk.CreateAwrHubResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateAwrHubDetails", RequestName: "CreateAwrHubDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request opsisdk.CreateAwrHubRequest) (opsisdk.CreateAwrHubResponse, error) {
				return client.CreateAwrHub(ctx, request)
			},
		},
		Get: runtimeOperationHooks[opsisdk.GetAwrHubRequest, opsisdk.GetAwrHubResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "AwrHubId", RequestName: "awrHubId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error) {
				return client.GetAwrHub(ctx, request)
			},
		},
		List: runtimeOperationHooks[opsisdk.ListAwrHubsRequest, opsisdk.ListAwrHubsResponse]{
			Fields: awrHubListFields(),
			Call: func(ctx context.Context, request opsisdk.ListAwrHubsRequest) (opsisdk.ListAwrHubsResponse, error) {
				return client.ListAwrHubs(ctx, request)
			},
		},
		Update: runtimeOperationHooks[opsisdk.UpdateAwrHubRequest, opsisdk.UpdateAwrHubResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "AwrHubId", RequestName: "awrHubId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateAwrHubDetails", RequestName: "UpdateAwrHubDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request opsisdk.UpdateAwrHubRequest) (opsisdk.UpdateAwrHubResponse, error) {
				return client.UpdateAwrHub(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[opsisdk.DeleteAwrHubRequest, opsisdk.DeleteAwrHubResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "AwrHubId", RequestName: "awrHubId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.DeleteAwrHubRequest) (opsisdk.DeleteAwrHubResponse, error) {
				return client.DeleteAwrHub(ctx, request)
			},
		},
		WrapGeneratedClient: []func(AwrHubServiceClient) AwrHubServiceClient{},
	}
}

func awrHubRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "opsi",
		FormalSlug:    "awrhub",
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
			ProvisioningStates: []string{string(opsisdk.AwrHubLifecycleStateCreating)},
			UpdatingStates:     []string{string(opsisdk.AwrHubLifecycleStateUpdating)},
			ActiveStates:       []string{string(opsisdk.AwrHubLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(opsisdk.AwrHubLifecycleStateDeleting)},
			TerminalStates: []string{string(opsisdk.AwrHubLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"operationsInsightsWarehouseId",
				"compartmentId",
				"displayName",
				"objectStorageBucketName",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"displayName", "freeformTags", "definedTags"},
			Mutable:         []string{"displayName", "freeformTags", "definedTags"},
			ForceNew: []string{
				"operationsInsightsWarehouseId",
				"compartmentId",
				"objectStorageBucketName",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "awrHub", Action: string(opsisdk.OperationTypeCreateAwrhub)},
			},
			Update: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "awrHub", Action: string(opsisdk.OperationTypeUpdateAwrhub)},
			},
			Delete: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "awrHub", Action: string(opsisdk.OperationTypeDeleteAwrhub)},
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetAwrHub",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "awrHub", Action: string(opsisdk.OperationTypeCreateAwrhub)},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetAwrHub",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "awrHub", Action: string(opsisdk.OperationTypeUpdateAwrhub)},
			},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "awrHub", Action: string(opsisdk.OperationTypeDeleteAwrhub)},
			},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func awrHubListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "OperationsInsightsWarehouseId", RequestName: "operationsInsightsWarehouseId", Contribution: "query", LookupPaths: []string{"status.operationsInsightsWarehouseId", "spec.operationsInsightsWarehouseId", "operationsInsightsWarehouseId"}},
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", LookupPaths: []string{"status.displayName", "spec.displayName", "displayName"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true, LookupPaths: []string{"status.id", "status.ocid"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func guardAwrHubExistingBeforeCreate(_ context.Context, resource *opsiv1beta1.AwrHub) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("awrhub resource is nil")
	}
	if strings.TrimSpace(resource.Spec.OperationsInsightsWarehouseId) == "" ||
		strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildAwrHubCreateBody(
	_ context.Context,
	resource *opsiv1beta1.AwrHub,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("awrhub resource is nil")
	}
	if err := validateAwrHubSpec(resource.Spec); err != nil {
		return nil, err
	}

	spec := resource.Spec
	body := opsisdk.CreateAwrHubDetails{
		OperationsInsightsWarehouseId: common.String(strings.TrimSpace(spec.OperationsInsightsWarehouseId)),
		CompartmentId:                 common.String(strings.TrimSpace(spec.CompartmentId)),
		DisplayName:                   common.String(strings.TrimSpace(spec.DisplayName)),
	}
	if strings.TrimSpace(spec.ObjectStorageBucketName) != "" {
		body.ObjectStorageBucketName = common.String(strings.TrimSpace(spec.ObjectStorageBucketName))
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneAwrHubStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = awrHubDefinedTagsFromSpec(spec.DefinedTags)
	}
	return body, nil
}

func buildAwrHubUpdateBody(
	_ context.Context,
	resource *opsiv1beta1.AwrHub,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return opsisdk.UpdateAwrHubDetails{}, false, fmt.Errorf("awrhub resource is nil")
	}
	if err := validateAwrHubSpec(resource.Spec); err != nil {
		return opsisdk.UpdateAwrHubDetails{}, false, err
	}

	current, ok := awrHubFromResponse(currentResponse)
	if !ok {
		return opsisdk.UpdateAwrHubDetails{}, false, fmt.Errorf("current AwrHub response does not expose an AwrHub body")
	}

	details := opsisdk.UpdateAwrHubDetails{}
	updateNeeded := false
	if displayName, ok := desiredAwrHubStringForUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = displayName
		updateNeeded = true
	}
	if freeformTags, ok := desiredAwrHubFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = freeformTags
		updateNeeded = true
	}
	if definedTags, ok := desiredAwrHubDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = definedTags
		updateNeeded = true
	}
	if !updateNeeded {
		return opsisdk.UpdateAwrHubDetails{}, false, nil
	}
	return details, true, nil
}

func validateAwrHubSpec(spec opsiv1beta1.AwrHubSpec) error {
	if strings.TrimSpace(spec.OperationsInsightsWarehouseId) == "" {
		return fmt.Errorf("awrhub spec.operationsInsightsWarehouseId is required")
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		return fmt.Errorf("awrhub spec.compartmentId is required")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		return fmt.Errorf("awrhub spec.displayName is required")
	}
	return nil
}

func wrapAwrHubListCalls(hooks *AwrHubRuntimeHooks) {
	if hooks == nil || hooks.List.Call == nil {
		return
	}
	call := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request opsisdk.ListAwrHubsRequest) (opsisdk.ListAwrHubsResponse, error) {
		return listAwrHubsAllPages(ctx, call, request)
	}
}

func listAwrHubsAllPages(
	ctx context.Context,
	call func(context.Context, opsisdk.ListAwrHubsRequest) (opsisdk.ListAwrHubsResponse, error),
	request opsisdk.ListAwrHubsRequest,
) (opsisdk.ListAwrHubsResponse, error) {
	var combined opsisdk.ListAwrHubsResponse
	for {
		response, err := call(ctx, request)
		if err != nil {
			return response, err
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.RawResponse = response.RawResponse
		combined.Items = append(combined.Items, response.Items...)
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func wrapAwrHubDeleteGuardClient(hooks *AwrHubRuntimeHooks) {
	if hooks == nil {
		return
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate AwrHubServiceClient) AwrHubServiceClient {
		return awrHubDeleteGuardClient{
			delegate:       delegate,
			getWorkRequest: hooks.Async.GetWorkRequest,
			getAwrHub:      hooks.Get.Call,
			listAwrHubs:    hooks.List.Call,
		}
	})
}

type awrHubDeleteGuardClient struct {
	delegate       AwrHubServiceClient
	getWorkRequest func(context.Context, string) (any, error)
	getAwrHub      func(context.Context, opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error)
	listAwrHubs    func(context.Context, opsisdk.ListAwrHubsRequest) (opsisdk.ListAwrHubsResponse, error)
}

func (c awrHubDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.AwrHub,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("awrhub runtime client is not configured")
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c awrHubDeleteGuardClient) Delete(ctx context.Context, resource *opsiv1beta1.AwrHub) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("awrhub runtime client is not configured")
	}
	if resource == nil {
		return false, fmt.Errorf("awrhub resource is nil")
	}

	workRequestID, phase := currentAwrHubWorkRequest(resource)
	if workRequestID == "" {
		if currentAwrHubID(resource) == "" {
			deleted, err := c.confirmAwrHubDeleteWithoutTrackedID(ctx, resource)
			if deleted || err != nil {
				return deleted, err
			}
		}
		return c.delegate.Delete(ctx, resource)
	}

	switch phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
		return c.resumeAwrHubWriteWorkRequestBeforeDelete(ctx, resource, workRequestID, phase)
	case shared.OSOKAsyncPhaseDelete:
		return c.resumeAwrHubDeleteWorkRequest(ctx, resource, workRequestID)
	default:
		return false, fmt.Errorf("awrhub delete cannot resume unsupported work request phase %q", phase)
	}
}

func (c awrHubDeleteGuardClient) confirmAwrHubDeleteWithoutTrackedID(
	ctx context.Context,
	resource *opsiv1beta1.AwrHub,
) (bool, error) {
	response, err := awrHubDeleteConfirmReadByList(ctx, resource, c.listAwrHubs)
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			markAwrHubDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, handleAwrHubDeleteError(resource, err)
	}
	if err := projectAwrHubStatusForDelete(resource, response); err != nil {
		return false, err
	}
	return false, nil
}

func currentAwrHubWorkRequest(resource *opsiv1beta1.AwrHub) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest || strings.TrimSpace(current.WorkRequestID) == "" {
		return "", ""
	}
	return strings.TrimSpace(current.WorkRequestID), current.Phase
}

func (c awrHubDeleteGuardClient) resumeAwrHubWriteWorkRequestBeforeDelete(
	ctx context.Context,
	resource *opsiv1beta1.AwrHub,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (bool, error) {
	workRequest, current, err := c.pollAwrHubWorkRequest(ctx, resource, workRequestID, phase)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.markAwrHubWorkRequestPendingBeforeDelete(resource, current, workRequestID)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		return c.deleteAfterSucceededAwrHubWriteWorkRequest(ctx, resource, workRequest, current, workRequestID)
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("awrhub %s work request %s finished with status %s before delete", current.Phase, workRequestID, current.RawStatus)
		c.markAwrHubWorkRequestFailure(resource, current, err)
		return false, err
	default:
		err := fmt.Errorf("awrhub %s work request %s projected unsupported async class %s before delete", current.Phase, workRequestID, current.NormalizedClass)
		c.markAwrHubWorkRequestFailure(resource, current, err)
		return false, err
	}
}

func (c awrHubDeleteGuardClient) resumeAwrHubDeleteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.AwrHub,
	workRequestID string,
) (bool, error) {
	workRequest, current, err := c.pollAwrHubWorkRequest(ctx, resource, workRequestID, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		c.markAwrHubWorkRequestOperation(resource, current, awrHubWorkRequestMessage(current.Phase, workRequest))
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		return c.confirmSucceededAwrHubDeleteWorkRequest(ctx, resource, workRequest, current)
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("awrhub delete work request %s finished with status %s", workRequestID, current.RawStatus)
		c.markAwrHubWorkRequestFailure(resource, current, err)
		return false, err
	default:
		err := fmt.Errorf("awrhub delete work request %s projected unsupported async class %s", workRequestID, current.NormalizedClass)
		c.markAwrHubWorkRequestFailure(resource, current, err)
		return false, err
	}
}

func (c awrHubDeleteGuardClient) pollAwrHubWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.AwrHub,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (any, *shared.OSOKAsyncOperation, error) {
	if c.getWorkRequest == nil {
		return nil, nil, fmt.Errorf("awrhub work request polling is not configured")
	}
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		return nil, nil, err
	}
	current, err := buildAwrHubAsyncOperation(&resource.Status.OsokStatus, workRequest, phase)
	if err != nil {
		return nil, nil, err
	}
	return workRequest, current, nil
}

func buildAwrHubAsyncOperation(
	status *shared.OSOKStatus,
	workRequest any,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	rawAction, err := resolveAwrHubWorkRequestAction(workRequest)
	if err != nil {
		return nil, err
	}
	if derivedPhase, ok, err := resolveAwrHubWorkRequestPhase(workRequest); err != nil {
		return nil, err
	} else if ok {
		if fallbackPhase != "" && fallbackPhase != derivedPhase {
			return nil, fmt.Errorf("awrhub work request exposes phase %q while reconcile expected %q", derivedPhase, fallbackPhase)
		}
		fallbackPhase = derivedPhase
	}

	current, err := awrHubWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}
	asyncOperation, err := servicemanager.BuildWorkRequestAsyncOperation(status, awrHubWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        rawAction,
		RawOperationType: string(current.OperationType),
		WorkRequestID:    awrHubStringValue(current.Id),
		PercentComplete:  current.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
	if err != nil {
		return nil, err
	}
	if message := strings.TrimSpace(awrHubWorkRequestMessage(asyncOperation.Phase, workRequest)); message != "" {
		asyncOperation.Message = message
	}
	return asyncOperation, nil
}

func (c awrHubDeleteGuardClient) deleteAfterSucceededAwrHubWriteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.AwrHub,
	workRequest any,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
) (bool, error) {
	resourceID, err := recoverAwrHubIDFromWorkRequest(resource, workRequest, current.Phase)
	if err != nil {
		c.markAwrHubWorkRequestFailure(resource, current, err)
		return false, err
	}
	if resourceID == "" {
		resourceID = currentAwrHubID(resource)
	}
	if resourceID == "" {
		err := fmt.Errorf("awrhub %s work request %s did not expose an AwrHub identifier before delete", current.Phase, workRequestID)
		c.markAwrHubWorkRequestFailure(resource, current, err)
		return false, err
	}
	response, err := c.readAwrHubForDelete(ctx, resourceID)
	if err != nil {
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			c.markAwrHubWorkRequestReadbackPending(resource, current, workRequestID, resourceID)
			return false, nil
		}
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		c.markAwrHubWorkRequestFailure(resource, current, err)
		return false, err
	}
	if err := projectAwrHubStatusForDelete(resource, response); err != nil {
		return false, err
	}
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	return c.delegate.Delete(ctx, resource)
}

func (c awrHubDeleteGuardClient) confirmSucceededAwrHubDeleteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.AwrHub,
	workRequest any,
	current *shared.OSOKAsyncOperation,
) (bool, error) {
	currentID, err := resolveSucceededAwrHubDeleteID(resource, workRequest, current)
	if err != nil {
		c.markAwrHubWorkRequestFailure(resource, current, err)
		return false, err
	}

	response, err := awrHubDeleteConfirmRead(c.getAwrHub, c.listAwrHubs)(ctx, resource, currentID)
	if err != nil {
		return c.handleSucceededAwrHubDeleteConfirmError(resource, current, err)
	}

	return applySucceededAwrHubDeleteConfirmResponse(resource, response)
}

func resolveSucceededAwrHubDeleteID(
	resource *opsiv1beta1.AwrHub,
	workRequest any,
	current *shared.OSOKAsyncOperation,
) (string, error) {
	currentID := currentAwrHubID(resource)
	if currentID == "" {
		recoveredID, err := recoverAwrHubIDFromWorkRequest(resource, workRequest, shared.OSOKAsyncPhaseDelete)
		if err != nil {
			return "", err
		}
		currentID = recoveredID
	}
	currentID = strings.TrimSpace(currentID)
	if currentID == "" {
		return "", fmt.Errorf("awrhub delete work request %s did not expose an AwrHub identifier", current.WorkRequestID)
	}
	return currentID, nil
}

func (c awrHubDeleteGuardClient) handleSucceededAwrHubDeleteConfirmError(
	resource *opsiv1beta1.AwrHub,
	current *shared.OSOKAsyncOperation,
	err error,
) (bool, error) {
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		markAwrHubDeleted(resource, "OCI resource deleted")
		return true, nil
	}
	c.markAwrHubWorkRequestFailure(resource, current, err)
	return false, err
}

func applySucceededAwrHubDeleteConfirmResponse(
	resource *opsiv1beta1.AwrHub,
	response any,
) (bool, error) {
	outcome, err := applyAwrHubDeleteConfirmOutcome(resource, response, generatedruntime.DeleteConfirmStageAfterRequest)
	if err != nil || outcome.Handled {
		return outcome.Deleted, err
	}
	if err := projectAwrHubStatusForDelete(resource, response); err != nil {
		return false, err
	}

	currentHub, ok := awrHubFromResponse(response)
	if !ok {
		return false, fmt.Errorf("awrhub delete confirmation response %T does not expose an AwrHub body", response)
	}
	switch currentHub.LifecycleState {
	case opsisdk.AwrHubLifecycleStateDeleted:
		markAwrHubDeleted(resource, "OCI resource deleted")
		return true, nil
	case "", opsisdk.AwrHubLifecycleStateDeleting:
		markAwrHubTerminating(resource, "OCI resource delete is in progress")
		return false, nil
	default:
		return false, fmt.Errorf("awrhub delete confirmation returned unexpected lifecycle state %q", currentHub.LifecycleState)
	}
}

func (c awrHubDeleteGuardClient) readAwrHubForDelete(
	ctx context.Context,
	resourceID string,
) (opsisdk.GetAwrHubResponse, error) {
	if c.getAwrHub == nil {
		return opsisdk.GetAwrHubResponse{}, fmt.Errorf("awrhub delete requires a readable OCI operation")
	}
	return c.getAwrHub(ctx, opsisdk.GetAwrHubRequest{
		AwrHubId: common.String(strings.TrimSpace(resourceID)),
	})
}

func (c awrHubDeleteGuardClient) markAwrHubWorkRequestPendingBeforeDelete(
	resource *opsiv1beta1.AwrHub,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
) {
	message := fmt.Sprintf("AwrHub %s work request %s is still in progress; waiting before delete", current.Phase, strings.TrimSpace(workRequestID))
	c.markAwrHubWorkRequestOperation(resource, current, message)
}

func (c awrHubDeleteGuardClient) markAwrHubWorkRequestReadbackPending(
	resource *opsiv1beta1.AwrHub,
	current *shared.OSOKAsyncOperation,
	workRequestID string,
	resourceID string,
) {
	if strings.TrimSpace(resourceID) != "" {
		resource.Status.Id = strings.TrimSpace(resourceID)
		resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(resourceID))
	}
	message := fmt.Sprintf(
		"AwrHub %s work request %s succeeded; waiting for AwrHub %s to become readable before delete",
		current.Phase,
		strings.TrimSpace(workRequestID),
		strings.TrimSpace(resourceID),
	)
	next := *current
	next.NormalizedClass = shared.OSOKAsyncClassPending
	c.markAwrHubWorkRequestOperation(resource, &next, message)
}

func (c awrHubDeleteGuardClient) markAwrHubWorkRequestFailure(
	resource *opsiv1beta1.AwrHub,
	current *shared.OSOKAsyncOperation,
	err error,
) {
	if err == nil {
		return
	}
	next := *current
	next.NormalizedClass = shared.OSOKAsyncClassFailed
	c.markAwrHubWorkRequestOperation(resource, &next, err.Error())
}

func (c awrHubDeleteGuardClient) markAwrHubWorkRequestOperation(
	resource *opsiv1beta1.AwrHub,
	current *shared.OSOKAsyncOperation,
	message string,
) {
	if resource == nil || current == nil {
		return
	}
	next := *current
	if strings.TrimSpace(message) != "" {
		next.Message = strings.TrimSpace(message)
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &next, loggerutil.OSOKLogger{})
}

func projectAwrHubStatusForDelete(resource *opsiv1beta1.AwrHub, response any) error {
	if err := awrHubStatusFromResponse(resource, response); err != nil {
		return err
	}
	if current, ok := awrHubFromResponse(response); ok {
		if id := awrHubStringValue(current.Id); strings.TrimSpace(id) != "" {
			resource.Status.Id = strings.TrimSpace(id)
			resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(id))
		}
	}
	return nil
}

func currentAwrHubID(resource *opsiv1beta1.AwrHub) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func markAwrHubDeleted(resource *opsiv1beta1.AwrHub, message string) {
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
		v1.ConditionTrue,
		"",
		resource.Status.OsokStatus.Message,
		loggerutil.OSOKLogger{},
	)
}

func markAwrHubTerminating(resource *opsiv1beta1.AwrHub, message string) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	resource.Status.OsokStatus.UpdatedAt = &now
	resource.Status.OsokStatus.Message = strings.TrimSpace(message)
	resource.Status.OsokStatus.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(
		resource.Status.OsokStatus,
		shared.Terminating,
		v1.ConditionTrue,
		"",
		resource.Status.OsokStatus.Message,
		loggerutil.OSOKLogger{},
	)
}

func awrHubStatusFromResponse(resource *opsiv1beta1.AwrHub, response any) error {
	if resource == nil {
		return fmt.Errorf("awrhub resource is nil")
	}
	current, ok := awrHubFromResponse(response)
	if !ok {
		return nil
	}

	resource.Status = opsiv1beta1.AwrHubStatus{
		OsokStatus:                    resource.Status.OsokStatus,
		OperationsInsightsWarehouseId: awrHubStringValue(current.OperationsInsightsWarehouseId),
		Id:                            awrHubStringValue(current.Id),
		CompartmentId:                 awrHubStringValue(current.CompartmentId),
		DisplayName:                   awrHubStringValue(current.DisplayName),
		ObjectStorageBucketName:       awrHubStringValue(current.ObjectStorageBucketName),
		TimeCreated:                   awrHubSDKTimeString(current.TimeCreated),
		LifecycleState:                string(current.LifecycleState),
		AwrMailboxUrl:                 awrHubStringValue(current.AwrMailboxUrl),
		FreeformTags:                  cloneAwrHubStringMap(current.FreeformTags),
		DefinedTags:                   awrHubStatusDefinedTags(current.DefinedTags),
		SystemTags:                    awrHubStatusDefinedTags(current.SystemTags),
		TimeUpdated:                   awrHubSDKTimeString(current.TimeUpdated),
		LifecycleDetails:              awrHubStringValue(current.LifecycleDetails),
		HubDstTimezoneVersion:         awrHubStringValue(current.HubDstTimezoneVersion),
	}
	return nil
}

func awrHubFromResponse(response any) (opsisdk.AwrHub, bool) {
	if current, ok := awrHubFromSDKResponse(response); ok {
		return current, true
	}
	if current, ok := awrHubFromOperationResponse(response); ok {
		return current, true
	}
	return awrHubFromDeleteConfirmResponse(response)
}

func awrHubFromSDKResponse(response any) (opsisdk.AwrHub, bool) {
	switch current := response.(type) {
	case opsisdk.AwrHub:
		return current, true
	case *opsisdk.AwrHub:
		if current == nil {
			return opsisdk.AwrHub{}, false
		}
		return *current, true
	case opsisdk.AwrHubSummary:
		return awrHubFromSummary(current), true
	case *opsisdk.AwrHubSummary:
		if current == nil {
			return opsisdk.AwrHub{}, false
		}
		return awrHubFromSummary(*current), true
	default:
		return opsisdk.AwrHub{}, false
	}
}

func awrHubFromOperationResponse(response any) (opsisdk.AwrHub, bool) {
	switch current := response.(type) {
	case opsisdk.CreateAwrHubResponse:
		return current.AwrHub, true
	case *opsisdk.CreateAwrHubResponse:
		if current == nil {
			return opsisdk.AwrHub{}, false
		}
		return current.AwrHub, true
	case opsisdk.GetAwrHubResponse:
		return current.AwrHub, true
	case *opsisdk.GetAwrHubResponse:
		if current == nil {
			return opsisdk.AwrHub{}, false
		}
		return current.AwrHub, true
	default:
		return opsisdk.AwrHub{}, false
	}
}

func awrHubFromDeleteConfirmResponse(response any) (opsisdk.AwrHub, bool) {
	switch current := response.(type) {
	case awrHubAmbiguousDeleteConfirmResponse:
		return current.AwrHub, true
	case *awrHubAmbiguousDeleteConfirmResponse:
		if current == nil {
			return opsisdk.AwrHub{}, false
		}
		return current.AwrHub, true
	default:
		return opsisdk.AwrHub{}, false
	}
}

func awrHubFromSummary(summary opsisdk.AwrHubSummary) opsisdk.AwrHub {
	return opsisdk.AwrHub{
		OperationsInsightsWarehouseId: summary.OperationsInsightsWarehouseId,
		Id:                            summary.Id,
		CompartmentId:                 summary.CompartmentId,
		DisplayName:                   summary.DisplayName,
		ObjectStorageBucketName:       summary.ObjectStorageBucketName,
		TimeCreated:                   summary.TimeCreated,
		LifecycleState:                summary.LifecycleState,
		AwrMailboxUrl:                 summary.AwrMailboxUrl,
		FreeformTags:                  summary.FreeformTags,
		DefinedTags:                   summary.DefinedTags,
		SystemTags:                    summary.SystemTags,
		TimeUpdated:                   summary.TimeUpdated,
		LifecycleDetails:              summary.LifecycleDetails,
	}
}

func clearTrackedAwrHubIdentity(resource *opsiv1beta1.AwrHub) {
	if resource == nil {
		return
	}
	resource.Status = opsiv1beta1.AwrHubStatus{}
}

func handleAwrHubDeleteError(resource *opsiv1beta1.AwrHub, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return conservativeAwrHubNotFoundError(err, "delete")
}

func awrHubDeleteConfirmRead(
	getAwrHub func(context.Context, opsisdk.GetAwrHubRequest) (opsisdk.GetAwrHubResponse, error),
	listAwrHubs func(context.Context, opsisdk.ListAwrHubsRequest) (opsisdk.ListAwrHubsResponse, error),
) func(context.Context, *opsiv1beta1.AwrHub, string) (any, error) {
	return func(ctx context.Context, resource *opsiv1beta1.AwrHub, currentID string) (any, error) {
		currentID = strings.TrimSpace(currentID)
		if currentID == "" {
			return awrHubDeleteConfirmReadByList(ctx, resource, listAwrHubs)
		}
		if getAwrHub == nil {
			return nil, fmt.Errorf("awrhub delete confirmation requires a readable OCI operation")
		}
		response, err := getAwrHub(ctx, opsisdk.GetAwrHubRequest{
			AwrHubId: common.String(currentID),
		})
		if err == nil {
			return response, nil
		}
		if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
			return nil, err
		}
		handledErr := handleAwrHubDeleteError(resource, err)
		if handledErr == nil {
			handledErr = err
		}
		return awrHubAmbiguousDeleteConfirmResponse{
			AwrHub: awrHubSyntheticCurrent(resource, currentID, opsisdk.AwrHubLifecycleStateActive),
			err:    handledErr,
		}, nil
	}
}

func awrHubDeleteConfirmReadByList(
	ctx context.Context,
	resource *opsiv1beta1.AwrHub,
	listAwrHubs func(context.Context, opsisdk.ListAwrHubsRequest) (opsisdk.ListAwrHubsResponse, error),
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("awrhub resource is nil")
	}
	if listAwrHubs == nil {
		return nil, fmt.Errorf("awrhub delete confirmation requires a readable OCI operation")
	}
	warehouseID := firstAwrHubString(resource.Status.OperationsInsightsWarehouseId, resource.Spec.OperationsInsightsWarehouseId)
	if warehouseID == "" {
		return nil, fmt.Errorf("awrhub delete confirmation requires operationsInsightsWarehouseId")
	}

	request := opsisdk.ListAwrHubsRequest{
		OperationsInsightsWarehouseId: common.String(warehouseID),
	}
	if compartmentID := firstAwrHubString(resource.Status.CompartmentId, resource.Spec.CompartmentId); compartmentID != "" {
		request.CompartmentId = common.String(compartmentID)
	}
	if displayName := firstAwrHubString(resource.Status.DisplayName, resource.Spec.DisplayName); displayName != "" {
		request.DisplayName = common.String(displayName)
	}

	response, err := listAwrHubs(ctx, request)
	if err != nil {
		return nil, err
	}
	matches := matchingAwrHubSummaries(resource, response.Items)
	switch len(matches) {
	case 0:
		return nil, errorutil.NotFoundOciError(errorutil.OciErrors{
			HTTPStatusCode: 404,
			ErrorCode:      errorutil.NotFound,
			OpcRequestID:   awrHubStringValue(response.OpcRequestId),
			Description:    "awrhub not found during delete confirmation",
		})
	case 1:
		return opsisdk.GetAwrHubResponse{
			AwrHub:       awrHubFromSummary(matches[0]),
			OpcRequestId: response.OpcRequestId,
		}, nil
	default:
		return nil, fmt.Errorf("awrhub delete confirmation found %d matching AwrHubs", len(matches))
	}
}

func applyAwrHubDeleteConfirmOutcome(
	resource *opsiv1beta1.AwrHub,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	err, ok := awrHubAmbiguousDeleteConfirmError(response)
	if !ok {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, err
}

func awrHubAmbiguousDeleteConfirmError(response any) (error, bool) {
	switch typed := response.(type) {
	case awrHubAmbiguousDeleteConfirmResponse:
		return typed.err, true
	case *awrHubAmbiguousDeleteConfirmResponse:
		if typed == nil {
			return nil, false
		}
		return typed.err, true
	default:
		return nil, false
	}
}

func conservativeAwrHubNotFoundError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	message := fmt.Sprintf("awrhub %s returned ambiguous 404 NotAuthorizedOrNotFound: %s", strings.TrimSpace(operation), err.Error())
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return ambiguousAwrHubNotFoundError{message: message, opcRequestID: serviceErr.GetOpcRequestID()}
	}
	return ambiguousAwrHubNotFoundError{message: message}
}

func matchingAwrHubSummaries(resource *opsiv1beta1.AwrHub, items []opsisdk.AwrHubSummary) []opsisdk.AwrHubSummary {
	if resource == nil {
		return nil
	}
	var matches []opsisdk.AwrHubSummary
	for _, item := range items {
		if awrHubSummaryMatches(resource, item) {
			matches = append(matches, item)
		}
	}
	return matches
}

func awrHubSummaryMatches(resource *opsiv1beta1.AwrHub, item opsisdk.AwrHubSummary) bool {
	if item.LifecycleState == opsisdk.AwrHubLifecycleStateDeleted {
		return false
	}
	if want := firstAwrHubString(resource.Status.OperationsInsightsWarehouseId, resource.Spec.OperationsInsightsWarehouseId); want != "" && awrHubStringValue(item.OperationsInsightsWarehouseId) != want {
		return false
	}
	if want := firstAwrHubString(resource.Status.CompartmentId, resource.Spec.CompartmentId); want != "" && awrHubStringValue(item.CompartmentId) != want {
		return false
	}
	if want := firstAwrHubString(resource.Status.DisplayName, resource.Spec.DisplayName); want != "" && awrHubStringValue(item.DisplayName) != want {
		return false
	}
	if want := firstAwrHubString(resource.Status.ObjectStorageBucketName, resource.Spec.ObjectStorageBucketName); want != "" && awrHubStringValue(item.ObjectStorageBucketName) != want {
		return false
	}
	return true
}

func awrHubSyntheticCurrent(
	resource *opsiv1beta1.AwrHub,
	currentID string,
	state opsisdk.AwrHubLifecycleStateEnum,
) opsisdk.AwrHub {
	current := opsisdk.AwrHub{
		Id:             common.String(strings.TrimSpace(currentID)),
		LifecycleState: state,
	}
	if resource == nil {
		return current
	}
	current.OperationsInsightsWarehouseId = common.String(firstAwrHubString(resource.Status.OperationsInsightsWarehouseId, resource.Spec.OperationsInsightsWarehouseId))
	current.CompartmentId = common.String(firstAwrHubString(resource.Status.CompartmentId, resource.Spec.CompartmentId))
	current.DisplayName = common.String(firstAwrHubString(resource.Status.DisplayName, resource.Spec.DisplayName))
	if objectStorageBucketName := firstAwrHubString(resource.Status.ObjectStorageBucketName, resource.Spec.ObjectStorageBucketName); objectStorageBucketName != "" {
		current.ObjectStorageBucketName = common.String(objectStorageBucketName)
	}
	return current
}

func getAwrHubWorkRequest(
	ctx context.Context,
	client awrHubOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize AwrHub OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("awrhub OCI client is not configured")
	}
	response, err := client.GetWorkRequest(ctx, opsisdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveAwrHubWorkRequestAction(workRequest any) (string, error) {
	current, err := awrHubWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	var action string
	for _, resource := range current.Resources {
		if !isAwrHubWorkRequestResource(resource) {
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
			return "", fmt.Errorf("awrhub work request %s exposes conflicting AwrHub action types %q and %q", awrHubStringValue(current.Id), action, candidate)
		}
	}
	return action, nil
}

func resolveAwrHubWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := awrHubWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case opsisdk.OperationTypeCreateAwrhub:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case opsisdk.OperationTypeUpdateAwrhub:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case opsisdk.OperationTypeDeleteAwrhub:
		return shared.OSOKAsyncPhaseDelete, true, nil
	default:
		return "", false, nil
	}
}

func recoverAwrHubIDFromWorkRequest(
	_ *opsiv1beta1.AwrHub,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := awrHubWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	expectedAction := awrHubWorkRequestActionForPhase(phase)
	for _, resource := range current.Resources {
		if !isAwrHubWorkRequestResource(resource) {
			continue
		}
		if expectedAction != "" && resource.ActionType != expectedAction && resource.ActionType != opsisdk.ActionTypeInProgress {
			continue
		}
		if id := strings.TrimSpace(awrHubStringValue(resource.Identifier)); id != "" {
			return id, nil
		}
	}
	return "", nil
}

func awrHubWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) opsisdk.ActionTypeEnum {
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

func awrHubWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := awrHubWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("AwrHub %s work request %s is %s", phase, awrHubStringValue(current.Id), current.Status)
}

func awrHubWorkRequestFromAny(workRequest any) (opsisdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case opsisdk.WorkRequest:
		return current, nil
	case *opsisdk.WorkRequest:
		if current == nil {
			return opsisdk.WorkRequest{}, fmt.Errorf("awrhub work request is nil")
		}
		return *current, nil
	default:
		return opsisdk.WorkRequest{}, fmt.Errorf("awrhub work request has unexpected type %T", workRequest)
	}
}

func isAwrHubWorkRequestResource(resource opsisdk.WorkRequestResource) bool {
	return normalizeAwrHubWorkRequestToken(awrHubStringValue(resource.EntityType)) == "awrhub"
}

func normalizeAwrHubWorkRequestToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func desiredAwrHubStringForUpdate(spec string, current *string) (*string, bool) {
	trimmedSpec := strings.TrimSpace(spec)
	if trimmedSpec == "" || trimmedSpec == strings.TrimSpace(awrHubStringValue(current)) {
		return nil, false
	}
	return common.String(trimmedSpec), true
}

func desiredAwrHubFreeformTagsForUpdate(spec map[string]string, current map[string]string) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	desired := cloneAwrHubStringMap(spec)
	if reflect.DeepEqual(current, desired) {
		return nil, false
	}
	return desired, true
}

func desiredAwrHubDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := awrHubDefinedTagsFromSpec(spec)
	if reflect.DeepEqual(current, desired) {
		return nil, false
	}
	return desired, true
}

func awrHubDefinedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	return *util.ConvertToOciDefinedTags(&tags)
}

func awrHubStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for key, values := range input {
		if values == nil {
			converted[key] = nil
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for innerKey, innerValue := range values {
			tagValues[innerKey] = fmt.Sprint(innerValue)
		}
		converted[key] = tagValues
	}
	return converted
}

func cloneAwrHubStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func awrHubStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstAwrHubString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func awrHubSDKTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}
