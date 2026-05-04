/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package operationsinsightswarehouseuser

import (
	"context"
	"crypto/sha256"
	"fmt"
	"reflect"
	"strings"
	"time"

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
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	operationsInsightsWarehouseUserRequeueDuration             = time.Minute
	operationsInsightsWarehouseUserPasswordReasonPrefix        = "OperationsInsightsWarehouseUserConnectionPassword:v1:"
	operationsInsightsWarehouseUserPendingPasswordReasonPrefix = "OperationsInsightsWarehouseUserConnectionPasswordPending:v1:"
)

type operationsInsightsWarehouseUserPasswordCaptureContextKey struct{}

type operationsInsightsWarehouseUserPasswordCapture struct {
	resource *opsiv1beta1.OperationsInsightsWarehouseUser
	sent     *operationsInsightsWarehouseUserPasswordFingerprintRecord
}

type operationsInsightsWarehouseUserPasswordFingerprintRecord struct {
	phase         shared.OSOKAsyncPhase
	workRequestID string
	fingerprint   string
}

type operationsInsightsWarehouseUserOCIClient interface {
	CreateOperationsInsightsWarehouseUser(context.Context, opsisdk.CreateOperationsInsightsWarehouseUserRequest) (opsisdk.CreateOperationsInsightsWarehouseUserResponse, error)
	GetOperationsInsightsWarehouseUser(context.Context, opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error)
	ListOperationsInsightsWarehouseUsers(context.Context, opsisdk.ListOperationsInsightsWarehouseUsersRequest) (opsisdk.ListOperationsInsightsWarehouseUsersResponse, error)
	UpdateOperationsInsightsWarehouseUser(context.Context, opsisdk.UpdateOperationsInsightsWarehouseUserRequest) (opsisdk.UpdateOperationsInsightsWarehouseUserResponse, error)
	DeleteOperationsInsightsWarehouseUser(context.Context, opsisdk.DeleteOperationsInsightsWarehouseUserRequest) (opsisdk.DeleteOperationsInsightsWarehouseUserResponse, error)
	GetWorkRequest(context.Context, opsisdk.GetWorkRequestRequest) (opsisdk.GetWorkRequestResponse, error)
}

type operationsInsightsWarehouseUserRuntimeClient struct {
	delegate OperationsInsightsWarehouseUserServiceClient
	hooks    OperationsInsightsWarehouseUserRuntimeHooks
	log      loggerutil.OSOKLogger
}

type operationsInsightsWarehouseUserAmbiguousNotFoundError struct {
	message      string
	opcRequestID string
}

func (e operationsInsightsWarehouseUserAmbiguousNotFoundError) Error() string {
	return e.message
}

func (e operationsInsightsWarehouseUserAmbiguousNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

var operationsInsightsWarehouseUserWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
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
	registerOperationsInsightsWarehouseUserRuntimeHooksMutator(func(manager *OperationsInsightsWarehouseUserServiceManager, hooks *OperationsInsightsWarehouseUserRuntimeHooks) {
		client, initErr := newOperationsInsightsWarehouseUserOperationsInsightsClient(manager)
		applyOperationsInsightsWarehouseUserRuntimeHooks(hooks, client, initErr, manager.Log)
	})
}

func newOperationsInsightsWarehouseUserOperationsInsightsClient(manager *OperationsInsightsWarehouseUserServiceManager) (operationsInsightsWarehouseUserOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("operationsInsightsWarehouseUser service manager is nil")
	}
	client, err := opsisdk.NewOperationsInsightsClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyOperationsInsightsWarehouseUserRuntimeHooks(
	hooks *OperationsInsightsWarehouseUserRuntimeHooks,
	client operationsInsightsWarehouseUserOCIClient,
	initErr error,
	log loggerutil.OSOKLogger,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = operationsInsightsWarehouseUserRuntimeSemantics()
	hooks.BuildCreateBody = buildOperationsInsightsWarehouseUserCreateBody
	hooks.BuildUpdateBody = buildOperationsInsightsWarehouseUserUpdateBody
	hooks.Identity.GuardExistingBeforeCreate = guardOperationsInsightsWarehouseUserExistingBeforeCreate
	hooks.List.Fields = operationsInsightsWarehouseUserListFields()
	hooks.StatusHooks.ProjectStatus = operationsInsightsWarehouseUserStatusFromResponse
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateOperationsInsightsWarehouseUserCreateOnlyDrift
	hooks.DeleteHooks.HandleError = handleOperationsInsightsWarehouseUserDeleteError
	hooks.Async.Adapter = operationsInsightsWarehouseUserWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getOperationsInsightsWarehouseUserWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveOperationsInsightsWarehouseUserWorkRequestAction
	hooks.Async.ResolvePhase = resolveOperationsInsightsWarehouseUserWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverOperationsInsightsWarehouseUserIDFromWorkRequest
	hooks.Async.Message = operationsInsightsWarehouseUserWorkRequestMessage
	wrapOperationsInsightsWarehouseUserPasswordFingerprintCalls(hooks)
	wrapOperationsInsightsWarehouseUserListCalls(hooks)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate OperationsInsightsWarehouseUserServiceClient) OperationsInsightsWarehouseUserServiceClient {
		return operationsInsightsWarehouseUserRuntimeClient{
			delegate: delegate,
			hooks:    *hooks,
			log:      log,
		}
	})
}

func newOperationsInsightsWarehouseUserRuntimeHooksWithOCIClient(client operationsInsightsWarehouseUserOCIClient) OperationsInsightsWarehouseUserRuntimeHooks {
	return OperationsInsightsWarehouseUserRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*opsiv1beta1.OperationsInsightsWarehouseUser]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*opsiv1beta1.OperationsInsightsWarehouseUser]{},
		StatusHooks:     generatedruntime.StatusHooks[*opsiv1beta1.OperationsInsightsWarehouseUser]{},
		ParityHooks:     generatedruntime.ParityHooks[*opsiv1beta1.OperationsInsightsWarehouseUser]{},
		Async:           generatedruntime.AsyncHooks[*opsiv1beta1.OperationsInsightsWarehouseUser]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*opsiv1beta1.OperationsInsightsWarehouseUser]{},
		Create: runtimeOperationHooks[opsisdk.CreateOperationsInsightsWarehouseUserRequest, opsisdk.CreateOperationsInsightsWarehouseUserResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "CreateOperationsInsightsWarehouseUserDetails", RequestName: "CreateOperationsInsightsWarehouseUserDetails", Contribution: "body"}},
			Call: func(ctx context.Context, request opsisdk.CreateOperationsInsightsWarehouseUserRequest) (opsisdk.CreateOperationsInsightsWarehouseUserResponse, error) {
				return client.CreateOperationsInsightsWarehouseUser(ctx, request)
			},
		},
		Get: runtimeOperationHooks[opsisdk.GetOperationsInsightsWarehouseUserRequest, opsisdk.GetOperationsInsightsWarehouseUserResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OperationsInsightsWarehouseUserId", RequestName: "operationsInsightsWarehouseUserId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.GetOperationsInsightsWarehouseUserRequest) (opsisdk.GetOperationsInsightsWarehouseUserResponse, error) {
				return client.GetOperationsInsightsWarehouseUser(ctx, request)
			},
		},
		List: runtimeOperationHooks[opsisdk.ListOperationsInsightsWarehouseUsersRequest, opsisdk.ListOperationsInsightsWarehouseUsersResponse]{
			Fields: operationsInsightsWarehouseUserListFields(),
			Call: func(ctx context.Context, request opsisdk.ListOperationsInsightsWarehouseUsersRequest) (opsisdk.ListOperationsInsightsWarehouseUsersResponse, error) {
				return client.ListOperationsInsightsWarehouseUsers(ctx, request)
			},
		},
		Update: runtimeOperationHooks[opsisdk.UpdateOperationsInsightsWarehouseUserRequest, opsisdk.UpdateOperationsInsightsWarehouseUserResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "OperationsInsightsWarehouseUserId", RequestName: "operationsInsightsWarehouseUserId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateOperationsInsightsWarehouseUserDetails", RequestName: "UpdateOperationsInsightsWarehouseUserDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request opsisdk.UpdateOperationsInsightsWarehouseUserRequest) (opsisdk.UpdateOperationsInsightsWarehouseUserResponse, error) {
				return client.UpdateOperationsInsightsWarehouseUser(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[opsisdk.DeleteOperationsInsightsWarehouseUserRequest, opsisdk.DeleteOperationsInsightsWarehouseUserResponse]{
			Fields: []generatedruntime.RequestField{{FieldName: "OperationsInsightsWarehouseUserId", RequestName: "operationsInsightsWarehouseUserId", Contribution: "path", PreferResourceID: true}},
			Call: func(ctx context.Context, request opsisdk.DeleteOperationsInsightsWarehouseUserRequest) (opsisdk.DeleteOperationsInsightsWarehouseUserResponse, error) {
				return client.DeleteOperationsInsightsWarehouseUser(ctx, request)
			},
		},
		WrapGeneratedClient: []func(OperationsInsightsWarehouseUserServiceClient) OperationsInsightsWarehouseUserServiceClient{},
	}
}

func (c operationsInsightsWarehouseUserRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("OperationsInsightsWarehouseUser runtime client is not configured")
	}
	workRequestID, phase := currentOperationsInsightsWarehouseUserWorkRequest(resource)
	capture := &operationsInsightsWarehouseUserPasswordCapture{resource: resource}
	response, err := c.delegate.CreateOrUpdate(
		context.WithValue(ctx, operationsInsightsWarehouseUserPasswordCaptureContextKey{}, capture),
		resource,
		req,
	)
	if err == nil && response.IsSuccessful {
		reconcileOperationsInsightsWarehouseUserPasswordFingerprint(resource, capture, workRequestID, phase)
	}
	scrubOperationsInsightsWarehouseUserSecretStatus(resource)
	return response, err
}

func (c operationsInsightsWarehouseUserRuntimeClient) Delete(ctx context.Context, resource *opsiv1beta1.OperationsInsightsWarehouseUser) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("OperationsInsightsWarehouseUser resource is nil")
	}

	if workRequestID, phase := currentOperationsInsightsWarehouseUserWorkRequest(resource); workRequestID != "" {
		return c.resumeOperationsInsightsWarehouseUserWorkRequestBeforeDelete(ctx, resource, workRequestID, phase)
	}

	currentID := currentOperationsInsightsWarehouseUserID(resource)
	if currentID == "" {
		existing, found, err := c.lookupExistingOperationsInsightsWarehouseUser(ctx, resource)
		if err != nil {
			return false, markOperationsInsightsWarehouseUserFailure(resource, err, c.log)
		}
		if !found {
			markOperationsInsightsWarehouseUserDeleted(resource, "OCI resource no longer exists", c.log)
			return true, nil
		}
		operationsInsightsWarehouseUserProjectSDK(resource, existing)
		currentID = operationsInsightsWarehouseUserString(existing.Id)
	}

	return c.deleteCurrentOperationsInsightsWarehouseUser(ctx, resource, currentID)
}

func operationsInsightsWarehouseUserRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "opsi",
		FormalSlug:        "operationsinsightswarehouseuser",
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
			ProvisioningStates: []string{string(opsisdk.OperationsInsightsWarehouseUserLifecycleStateCreating)},
			UpdatingStates:     []string{string(opsisdk.OperationsInsightsWarehouseUserLifecycleStateUpdating)},
			ActiveStates:       []string{string(opsisdk.OperationsInsightsWarehouseUserLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(opsisdk.OperationsInsightsWarehouseUserLifecycleStateDeleting)},
			TerminalStates: []string{string(opsisdk.OperationsInsightsWarehouseUserLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"operationsInsightsWarehouseId", "compartmentId", "name", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"connectionPassword",
				"isAwrDataAccess",
				"isEmDataAccess",
				"isOpsiDataAccess",
				"freeformTags",
				"definedTags",
			},
			ForceNew:      []string{"operationsInsightsWarehouseId", "compartmentId", "name"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "OperationsInsightsWarehouseUser", Action: "CreateOperationsInsightsWarehouseUser"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "OperationsInsightsWarehouseUser", Action: "UpdateOperationsInsightsWarehouseUser"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "OperationsInsightsWarehouseUser", Action: "DeleteOperationsInsightsWarehouseUser"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetOperationsInsightsWarehouseUser",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "OperationsInsightsWarehouseUser", Action: string(opsisdk.OperationTypeCreateOpsiWarehouseUser)}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "GetWorkRequest -> GetOperationsInsightsWarehouseUser",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "OperationsInsightsWarehouseUser", Action: string(opsisdk.OperationTypeUpdateOpsiWarehouseUser)}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "OperationsInsightsWarehouseUser", Action: string(opsisdk.OperationTypeDeleteOpsiWarehouseUser)}},
		},
	}
}

func operationsInsightsWarehouseUserListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "OperationsInsightsWarehouseId",
			RequestName:  "operationsInsightsWarehouseId",
			Contribution: "query",
			LookupPaths:  []string{"status.operationsInsightsWarehouseId", "spec.operationsInsightsWarehouseId", "operationsInsightsWarehouseId"},
		},
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
			LookupPaths:  []string{"status.name", "spec.name", "name"},
		},
		{
			FieldName:        "Id",
			RequestName:      "id",
			Contribution:     "query",
			PreferResourceID: true,
			LookupPaths:      []string{"status.id", "status.ocid"},
		},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
	}
}

func guardOperationsInsightsWarehouseUserExistingBeforeCreate(_ context.Context, resource *opsiv1beta1.OperationsInsightsWarehouseUser) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("OperationsInsightsWarehouseUser resource is nil")
	}
	if strings.TrimSpace(resource.Spec.OperationsInsightsWarehouseId) == "" ||
		strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.Name) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildOperationsInsightsWarehouseUserCreateBody(
	_ context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("OperationsInsightsWarehouseUser resource is nil")
	}
	if err := validateOperationsInsightsWarehouseUserSpec(resource.Spec); err != nil {
		return nil, err
	}

	spec := resource.Spec
	body := opsisdk.CreateOperationsInsightsWarehouseUserDetails{
		OperationsInsightsWarehouseId: common.String(strings.TrimSpace(spec.OperationsInsightsWarehouseId)),
		CompartmentId:                 common.String(strings.TrimSpace(spec.CompartmentId)),
		Name:                          common.String(strings.TrimSpace(spec.Name)),
		ConnectionPassword:            common.String(spec.ConnectionPassword),
		IsAwrDataAccess:               common.Bool(spec.IsAwrDataAccess),
	}
	if spec.IsEmDataAccess {
		body.IsEmDataAccess = common.Bool(true)
	}
	if spec.IsOpsiDataAccess {
		body.IsOpsiDataAccess = common.Bool(true)
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = cloneOperationsInsightsWarehouseUserStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = operationsInsightsWarehouseUserDefinedTagsFromSpec(spec.DefinedTags)
	}
	return body, nil
}

func buildOperationsInsightsWarehouseUserUpdateBody(
	_ context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return opsisdk.UpdateOperationsInsightsWarehouseUserDetails{}, false, fmt.Errorf("OperationsInsightsWarehouseUser resource is nil")
	}
	if err := validateOperationsInsightsWarehouseUserSpec(resource.Spec); err != nil {
		return opsisdk.UpdateOperationsInsightsWarehouseUserDetails{}, false, err
	}

	current, ok := operationsInsightsWarehouseUserFromResponse(currentResponse)
	if !ok {
		return opsisdk.UpdateOperationsInsightsWarehouseUserDetails{}, false, fmt.Errorf("current OperationsInsightsWarehouseUser response does not expose an OperationsInsightsWarehouseUser body")
	}
	if err := validateOperationsInsightsWarehouseUserCreateOnlyDrift(resource, current); err != nil {
		return opsisdk.UpdateOperationsInsightsWarehouseUserDetails{}, false, err
	}

	details, updateNeeded := buildOperationsInsightsWarehouseUserUpdateDetails(resource, current)
	if !updateNeeded {
		return opsisdk.UpdateOperationsInsightsWarehouseUserDetails{}, false, nil
	}
	return details, true, nil
}

func buildOperationsInsightsWarehouseUserUpdateDetails(
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	current opsisdk.OperationsInsightsWarehouseUser,
) (opsisdk.UpdateOperationsInsightsWarehouseUserDetails, bool) {
	spec := resource.Spec
	details := opsisdk.UpdateOperationsInsightsWarehouseUserDetails{}
	updateNeeded := false
	if password, ok := desiredOperationsInsightsWarehouseUserPasswordForUpdate(resource, current.ConnectionPassword); ok {
		details.ConnectionPassword = password
		updateNeeded = true
	}
	if isAwrDataAccess, ok := desiredOperationsInsightsWarehouseUserRequiredBoolForUpdate(spec.IsAwrDataAccess, current.IsAwrDataAccess); ok {
		details.IsAwrDataAccess = isAwrDataAccess
		updateNeeded = true
	}
	if isEmDataAccess, ok := desiredOperationsInsightsWarehouseUserOptionalBoolForUpdate(spec.IsEmDataAccess, current.IsEmDataAccess); ok {
		details.IsEmDataAccess = isEmDataAccess
		updateNeeded = true
	}
	if isOpsiDataAccess, ok := desiredOperationsInsightsWarehouseUserOptionalBoolForUpdate(spec.IsOpsiDataAccess, current.IsOpsiDataAccess); ok {
		details.IsOpsiDataAccess = isOpsiDataAccess
		updateNeeded = true
	}
	if freeformTags, ok := desiredOperationsInsightsWarehouseUserFreeformTagsForUpdate(spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = freeformTags
		updateNeeded = true
	}
	if definedTags, ok := desiredOperationsInsightsWarehouseUserDefinedTagsForUpdate(spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = definedTags
		updateNeeded = true
	}
	return details, updateNeeded
}

func validateOperationsInsightsWarehouseUserSpec(spec opsiv1beta1.OperationsInsightsWarehouseUserSpec) error {
	if strings.TrimSpace(spec.OperationsInsightsWarehouseId) == "" {
		return fmt.Errorf("OperationsInsightsWarehouseUser spec.operationsInsightsWarehouseId is required")
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		return fmt.Errorf("OperationsInsightsWarehouseUser spec.compartmentId is required")
	}
	if strings.TrimSpace(spec.Name) == "" {
		return fmt.Errorf("OperationsInsightsWarehouseUser spec.name is required")
	}
	if spec.ConnectionPassword == "" {
		return fmt.Errorf("OperationsInsightsWarehouseUser spec.connectionPassword is required")
	}
	return nil
}

func validateOperationsInsightsWarehouseUserCreateOnlyDrift(resource *opsiv1beta1.OperationsInsightsWarehouseUser, currentResponse any) error {
	current, ok := operationsInsightsWarehouseUserFromResponse(currentResponse)
	if !ok {
		return nil
	}
	return validateOperationsInsightsWarehouseUserImmutableFields(resource, current)
}

func validateOperationsInsightsWarehouseUserImmutableFields(
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	current opsisdk.OperationsInsightsWarehouseUser,
) error {
	if resource == nil {
		return fmt.Errorf("OperationsInsightsWarehouseUser resource is nil")
	}
	checks := []struct {
		field string
		want  string
		have  string
	}{
		{field: "spec.operationsInsightsWarehouseId", want: strings.TrimSpace(resource.Spec.OperationsInsightsWarehouseId), have: operationsInsightsWarehouseUserString(current.OperationsInsightsWarehouseId)},
		{field: "spec.compartmentId", want: strings.TrimSpace(resource.Spec.CompartmentId), have: operationsInsightsWarehouseUserString(current.CompartmentId)},
		{field: "spec.name", want: strings.TrimSpace(resource.Spec.Name), have: operationsInsightsWarehouseUserString(current.Name)},
	}
	for _, check := range checks {
		if check.want != "" && check.have != "" && check.want != check.have {
			return fmt.Errorf("OperationsInsightsWarehouseUser formal semantics require replacement when %s changes", check.field)
		}
	}
	return nil
}

func wrapOperationsInsightsWarehouseUserListCalls(hooks *OperationsInsightsWarehouseUserRuntimeHooks) {
	if hooks == nil || hooks.List.Call == nil {
		return
	}
	call := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request opsisdk.ListOperationsInsightsWarehouseUsersRequest) (opsisdk.ListOperationsInsightsWarehouseUsersResponse, error) {
		return listOperationsInsightsWarehouseUserAllPages(ctx, call, request)
	}
}

func wrapOperationsInsightsWarehouseUserPasswordFingerprintCalls(hooks *OperationsInsightsWarehouseUserRuntimeHooks) {
	if hooks == nil {
		return
	}
	if hooks.Create.Call != nil {
		call := hooks.Create.Call
		hooks.Create.Call = func(ctx context.Context, request opsisdk.CreateOperationsInsightsWarehouseUserRequest) (opsisdk.CreateOperationsInsightsWarehouseUserResponse, error) {
			response, err := call(ctx, request)
			if err == nil {
				captureOperationsInsightsWarehouseUserSentPassword(ctx, shared.OSOKAsyncPhaseCreate, request.ConnectionPassword, operationsInsightsWarehouseUserString(response.OpcWorkRequestId))
			}
			return response, err
		}
	}
	if hooks.Update.Call != nil {
		call := hooks.Update.Call
		hooks.Update.Call = func(ctx context.Context, request opsisdk.UpdateOperationsInsightsWarehouseUserRequest) (opsisdk.UpdateOperationsInsightsWarehouseUserResponse, error) {
			response, err := call(ctx, request)
			if err == nil {
				captureOperationsInsightsWarehouseUserSentPassword(ctx, shared.OSOKAsyncPhaseUpdate, request.ConnectionPassword, operationsInsightsWarehouseUserString(response.OpcWorkRequestId))
			}
			return response, err
		}
	}
}

func listOperationsInsightsWarehouseUserAllPages(
	ctx context.Context,
	call func(context.Context, opsisdk.ListOperationsInsightsWarehouseUsersRequest) (opsisdk.ListOperationsInsightsWarehouseUsersResponse, error),
	request opsisdk.ListOperationsInsightsWarehouseUsersRequest,
) (opsisdk.ListOperationsInsightsWarehouseUsersResponse, error) {
	var combined opsisdk.ListOperationsInsightsWarehouseUsersResponse
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

func (c operationsInsightsWarehouseUserRuntimeClient) deleteCurrentOperationsInsightsWarehouseUser(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	currentID string,
) (bool, error) {
	current, found, err := c.getOperationsInsightsWarehouseUser(ctx, resource, currentID)
	if err != nil {
		return false, markOperationsInsightsWarehouseUserFailure(resource, err, c.log)
	}
	if !found {
		markOperationsInsightsWarehouseUserDeleted(resource, "OCI resource no longer exists", c.log)
		return true, nil
	}
	return c.deleteReadableOperationsInsightsWarehouseUser(ctx, resource, currentID, current)
}

func (c operationsInsightsWarehouseUserRuntimeClient) deleteReadableOperationsInsightsWarehouseUser(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	currentID string,
	current opsisdk.OperationsInsightsWarehouseUser,
) (bool, error) {
	if operationsInsightsWarehouseUserLifecycleDeleted(current.LifecycleState) {
		operationsInsightsWarehouseUserProjectSDK(resource, current)
		markOperationsInsightsWarehouseUserDeleted(resource, "OCI resource deleted", c.log)
		return true, nil
	}
	if operationsInsightsWarehouseUserLifecycleDeleting(current.LifecycleState) {
		operationsInsightsWarehouseUserProjectSDK(resource, current)
		markOperationsInsightsWarehouseUserTerminating(resource, "OCI resource delete is in progress", c.log)
		return false, nil
	}

	if c.hooks.Delete.Call == nil {
		err := fmt.Errorf("OperationsInsightsWarehouseUser delete OCI call is not configured")
		return false, markOperationsInsightsWarehouseUserFailure(resource, err, c.log)
	}
	response, err := c.hooks.Delete.Call(ctx, opsisdk.DeleteOperationsInsightsWarehouseUserRequest{
		OperationsInsightsWarehouseUserId: common.String(currentID),
	})
	if err != nil {
		err = handleOperationsInsightsWarehouseUserDeleteError(resource, err)
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			markOperationsInsightsWarehouseUserDeleted(resource, "OCI resource no longer exists", c.log)
			return true, nil
		}
		return false, markOperationsInsightsWarehouseUserFailure(resource, err, c.log)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	if workRequestID := strings.TrimSpace(operationsInsightsWarehouseUserString(response.OpcWorkRequestId)); workRequestID != "" {
		markOperationsInsightsWarehouseUserWorkRequest(resource, &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceWorkRequest,
			Phase:           shared.OSOKAsyncPhaseDelete,
			WorkRequestID:   workRequestID,
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         fmt.Sprintf("OperationsInsightsWarehouseUser delete work request %s is pending", workRequestID),
		}, c.log)
		return false, nil
	}
	return c.confirmOperationsInsightsWarehouseUserDeleted(ctx, resource, currentID)
}

func (c operationsInsightsWarehouseUserRuntimeClient) confirmOperationsInsightsWarehouseUserDeleted(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	currentID string,
) (bool, error) {
	current, found, err := c.getOperationsInsightsWarehouseUser(ctx, resource, currentID)
	if err != nil {
		return false, markOperationsInsightsWarehouseUserFailure(resource, err, c.log)
	}
	if !found || operationsInsightsWarehouseUserLifecycleDeleted(current.LifecycleState) {
		markOperationsInsightsWarehouseUserDeleted(resource, "OCI resource deleted", c.log)
		return true, nil
	}
	operationsInsightsWarehouseUserProjectSDK(resource, current)
	markOperationsInsightsWarehouseUserTerminating(resource, "OCI resource delete is in progress", c.log)
	return false, nil
}

func (c operationsInsightsWarehouseUserRuntimeClient) getOperationsInsightsWarehouseUser(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	currentID string,
) (opsisdk.OperationsInsightsWarehouseUser, bool, error) {
	if strings.TrimSpace(currentID) == "" {
		return opsisdk.OperationsInsightsWarehouseUser{}, false, nil
	}
	if c.hooks.Get.Call == nil {
		return opsisdk.OperationsInsightsWarehouseUser{}, false, fmt.Errorf("OperationsInsightsWarehouseUser get OCI call is not configured")
	}
	response, err := c.hooks.Get.Call(ctx, opsisdk.GetOperationsInsightsWarehouseUserRequest{
		OperationsInsightsWarehouseUserId: common.String(strings.TrimSpace(currentID)),
	})
	if err != nil {
		err = handleOperationsInsightsWarehouseUserDeleteError(resource, err)
		if errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return opsisdk.OperationsInsightsWarehouseUser{}, false, nil
		}
		return opsisdk.OperationsInsightsWarehouseUser{}, false, err
	}
	return response.OperationsInsightsWarehouseUser, true, nil
}

func (c operationsInsightsWarehouseUserRuntimeClient) lookupExistingOperationsInsightsWarehouseUser(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
) (opsisdk.OperationsInsightsWarehouseUser, bool, error) {
	if c.hooks.List.Call == nil {
		return opsisdk.OperationsInsightsWarehouseUser{}, false, nil
	}
	response, err := c.hooks.List.Call(ctx, opsisdk.ListOperationsInsightsWarehouseUsersRequest{
		OperationsInsightsWarehouseId: common.String(strings.TrimSpace(resource.Spec.OperationsInsightsWarehouseId)),
		CompartmentId:                 common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		DisplayName:                   common.String(strings.TrimSpace(resource.Spec.Name)),
	})
	if err != nil {
		err = handleOperationsInsightsWarehouseUserDeleteError(resource, err)
		return opsisdk.OperationsInsightsWarehouseUser{}, false, err
	}

	var matches []opsisdk.OperationsInsightsWarehouseUser
	for _, item := range response.Items {
		current := operationsInsightsWarehouseUserFromSummary(item)
		if !operationsInsightsWarehouseUserCanBindExisting(current) || !operationsInsightsWarehouseUserMatchesSpec(current, resource.Spec) {
			continue
		}
		matches = append(matches, current)
	}
	switch len(matches) {
	case 0:
		return opsisdk.OperationsInsightsWarehouseUser{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return opsisdk.OperationsInsightsWarehouseUser{}, false, fmt.Errorf("OperationsInsightsWarehouseUser list response returned multiple matching resources for operationsInsightsWarehouseId %q and name %q", resource.Spec.OperationsInsightsWarehouseId, resource.Spec.Name)
	}
}

func (c operationsInsightsWarehouseUserRuntimeClient) resumeOperationsInsightsWarehouseUserWorkRequestBeforeDelete(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (bool, error) {
	workRequest, current, err := c.pollOperationsInsightsWarehouseUserWorkRequest(ctx, resource, workRequestID, phase)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, markOperationsInsightsWarehouseUserFailure(resource, err, c.log)
	}

	switch current.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		markOperationsInsightsWarehouseUserWorkRequest(resource, current, c.log)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		if current.Phase == shared.OSOKAsyncPhaseDelete {
			return c.confirmSucceededOperationsInsightsWarehouseUserDeleteWorkRequest(ctx, resource, workRequest, current)
		}
		return c.deleteAfterSucceededOperationsInsightsWarehouseUserWriteWorkRequest(ctx, resource, workRequest, current)
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		err := fmt.Errorf("OperationsInsightsWarehouseUser %s work request %s finished with status %s", current.Phase, workRequestID, current.RawStatus)
		return false, markOperationsInsightsWarehouseUserFailure(resource, err, c.log)
	default:
		err := fmt.Errorf("OperationsInsightsWarehouseUser %s work request %s projected unsupported async class %s", current.Phase, workRequestID, current.NormalizedClass)
		return false, markOperationsInsightsWarehouseUserFailure(resource, err, c.log)
	}
}

func (c operationsInsightsWarehouseUserRuntimeClient) pollOperationsInsightsWarehouseUserWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (any, *shared.OSOKAsyncOperation, error) {
	if c.hooks.Async.GetWorkRequest == nil {
		return nil, nil, fmt.Errorf("OperationsInsightsWarehouseUser work request polling is not configured")
	}
	workRequest, err := c.hooks.Async.GetWorkRequest(ctx, workRequestID)
	if err != nil {
		return nil, nil, err
	}
	current, err := buildOperationsInsightsWarehouseUserAsyncOperation(&resource.Status.OsokStatus, workRequest, phase)
	if err != nil {
		return nil, nil, err
	}
	return workRequest, current, nil
}

func (c operationsInsightsWarehouseUserRuntimeClient) deleteAfterSucceededOperationsInsightsWarehouseUserWriteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	workRequest any,
	current *shared.OSOKAsyncOperation,
) (bool, error) {
	resourceID, err := recoverOperationsInsightsWarehouseUserIDFromWorkRequest(resource, workRequest, current.Phase)
	if err != nil {
		return false, markOperationsInsightsWarehouseUserFailure(resource, err, c.log)
	}
	if resourceID == "" {
		err := fmt.Errorf("OperationsInsightsWarehouseUser %s work request %s did not expose a resource identifier before delete", current.Phase, current.WorkRequestID)
		return false, markOperationsInsightsWarehouseUserFailure(resource, err, c.log)
	}
	resource.Status.Id = resourceID
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	readback, found, err := c.getOperationsInsightsWarehouseUser(ctx, resource, resourceID)
	if err != nil {
		return false, markOperationsInsightsWarehouseUserFailure(resource, err, c.log)
	}
	if !found {
		markOperationsInsightsWarehouseUserWriteReadbackPending(resource, current, resourceID, c.log)
		return false, nil
	}
	operationsInsightsWarehouseUserProjectSDK(resource, readback)
	servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
	return c.deleteReadableOperationsInsightsWarehouseUser(ctx, resource, resourceID, readback)
}

func (c operationsInsightsWarehouseUserRuntimeClient) confirmSucceededOperationsInsightsWarehouseUserDeleteWorkRequest(
	ctx context.Context,
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	workRequest any,
	current *shared.OSOKAsyncOperation,
) (bool, error) {
	resourceID := currentOperationsInsightsWarehouseUserID(resource)
	if resourceID == "" {
		var err error
		resourceID, err = recoverOperationsInsightsWarehouseUserIDFromWorkRequest(resource, workRequest, shared.OSOKAsyncPhaseDelete)
		if err != nil {
			return false, markOperationsInsightsWarehouseUserFailure(resource, err, c.log)
		}
	}
	if resourceID == "" {
		markOperationsInsightsWarehouseUserDeleted(resource, "OCI delete work request completed", c.log)
		return true, nil
	}
	return c.confirmOperationsInsightsWarehouseUserDeleted(ctx, resource, resourceID)
}

func getOperationsInsightsWarehouseUserWorkRequest(
	ctx context.Context,
	client operationsInsightsWarehouseUserOCIClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize OperationsInsightsWarehouseUser OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("OperationsInsightsWarehouseUser OCI client is not configured")
	}
	response, err := client.GetWorkRequest(ctx, opsisdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func buildOperationsInsightsWarehouseUserAsyncOperation(
	status *shared.OSOKStatus,
	workRequest any,
	fallbackPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	current, err := operationsInsightsWarehouseUserWorkRequestFromAny(workRequest)
	if err != nil {
		return nil, err
	}
	rawAction, err := resolveOperationsInsightsWarehouseUserWorkRequestAction(workRequest)
	if err != nil {
		return nil, err
	}
	if derivedPhase, ok, err := resolveOperationsInsightsWarehouseUserWorkRequestPhase(workRequest); err != nil {
		return nil, err
	} else if ok {
		if fallbackPhase != "" && fallbackPhase != derivedPhase {
			return nil, fmt.Errorf("OperationsInsightsWarehouseUser work request exposes phase %q while reconcile expected %q", derivedPhase, fallbackPhase)
		}
		fallbackPhase = derivedPhase
	}
	asyncOperation, err := servicemanager.BuildWorkRequestAsyncOperation(status, operationsInsightsWarehouseUserWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(current.Status),
		RawAction:        rawAction,
		RawOperationType: string(current.OperationType),
		WorkRequestID:    operationsInsightsWarehouseUserString(current.Id),
		PercentComplete:  current.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
	if err != nil {
		return nil, err
	}
	if message := strings.TrimSpace(operationsInsightsWarehouseUserWorkRequestMessage(asyncOperation.Phase, workRequest)); message != "" {
		asyncOperation.Message = message
	}
	return asyncOperation, nil
}

func resolveOperationsInsightsWarehouseUserWorkRequestAction(workRequest any) (string, error) {
	current, err := operationsInsightsWarehouseUserWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	for _, resource := range current.Resources {
		if !isOperationsInsightsWarehouseUserWorkRequestResource(resource) {
			continue
		}
		switch resource.ActionType {
		case opsisdk.ActionTypeCreated, opsisdk.ActionTypeUpdated, opsisdk.ActionTypeDeleted:
			return string(resource.ActionType), nil
		}
	}
	switch current.OperationType {
	case opsisdk.OperationTypeCreateOpsiWarehouseUser:
		return string(opsisdk.ActionTypeCreated), nil
	case opsisdk.OperationTypeUpdateOpsiWarehouseUser:
		return string(opsisdk.ActionTypeUpdated), nil
	case opsisdk.OperationTypeDeleteOpsiWarehouseUser:
		return string(opsisdk.ActionTypeDeleted), nil
	default:
		return "", nil
	}
}

func resolveOperationsInsightsWarehouseUserWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	current, err := operationsInsightsWarehouseUserWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	switch current.OperationType {
	case opsisdk.OperationTypeCreateOpsiWarehouseUser:
		return shared.OSOKAsyncPhaseCreate, true, nil
	case opsisdk.OperationTypeUpdateOpsiWarehouseUser:
		return shared.OSOKAsyncPhaseUpdate, true, nil
	case opsisdk.OperationTypeDeleteOpsiWarehouseUser:
		return shared.OSOKAsyncPhaseDelete, true, nil
	}
	for _, resource := range current.Resources {
		if !isOperationsInsightsWarehouseUserWorkRequestResource(resource) {
			continue
		}
		switch resource.ActionType {
		case opsisdk.ActionTypeCreated:
			return shared.OSOKAsyncPhaseCreate, true, nil
		case opsisdk.ActionTypeUpdated:
			return shared.OSOKAsyncPhaseUpdate, true, nil
		case opsisdk.ActionTypeDeleted:
			return shared.OSOKAsyncPhaseDelete, true, nil
		}
	}
	return "", false, nil
}

func recoverOperationsInsightsWarehouseUserIDFromWorkRequest(
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	if currentID := currentOperationsInsightsWarehouseUserID(resource); currentID != "" {
		return currentID, nil
	}
	current, err := operationsInsightsWarehouseUserWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	for _, candidate := range current.Resources {
		if !isOperationsInsightsWarehouseUserWorkRequestResource(candidate) {
			continue
		}
		if !operationsInsightsWarehouseUserWorkRequestActionMatchesPhase(candidate.ActionType, phase) {
			continue
		}
		if id := operationsInsightsWarehouseUserString(candidate.Identifier); id != "" {
			return id, nil
		}
	}
	for _, candidate := range current.Resources {
		if !isOperationsInsightsWarehouseUserWorkRequestResource(candidate) {
			continue
		}
		if id := operationsInsightsWarehouseUserString(candidate.Identifier); id != "" {
			return id, nil
		}
	}
	return "", nil
}

func operationsInsightsWarehouseUserWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := operationsInsightsWarehouseUserWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	workRequestID := operationsInsightsWarehouseUserString(current.Id)
	rawStatus := strings.TrimSpace(string(current.Status))
	switch {
	case phase != "" && workRequestID != "" && rawStatus != "":
		return fmt.Sprintf("OperationsInsightsWarehouseUser %s work request %s is %s", phase, workRequestID, rawStatus)
	case phase != "" && rawStatus != "":
		return fmt.Sprintf("OperationsInsightsWarehouseUser %s work request is %s", phase, rawStatus)
	default:
		return ""
	}
}

func operationsInsightsWarehouseUserWorkRequestFromAny(workRequest any) (opsisdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case opsisdk.WorkRequest:
		return current, nil
	case *opsisdk.WorkRequest:
		if current == nil {
			return opsisdk.WorkRequest{}, fmt.Errorf("OperationsInsightsWarehouseUser work request is nil")
		}
		return *current, nil
	default:
		return opsisdk.WorkRequest{}, fmt.Errorf("OperationsInsightsWarehouseUser work request has unexpected type %T", workRequest)
	}
}

func isOperationsInsightsWarehouseUserWorkRequestResource(resource opsisdk.WorkRequestResource) bool {
	token := normalizeOperationsInsightsWarehouseUserWorkRequestToken(operationsInsightsWarehouseUserString(resource.EntityType))
	return token == "operationsinsightswarehouseuser" || token == "opsiwarehouseuser" || strings.Contains(token, "warehouseuser")
}

func normalizeOperationsInsightsWarehouseUserWorkRequestToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func operationsInsightsWarehouseUserWorkRequestActionMatchesPhase(action opsisdk.ActionTypeEnum, phase shared.OSOKAsyncPhase) bool {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return action == opsisdk.ActionTypeCreated || action == opsisdk.ActionTypeInProgress
	case shared.OSOKAsyncPhaseUpdate:
		return action == opsisdk.ActionTypeUpdated || action == opsisdk.ActionTypeInProgress
	case shared.OSOKAsyncPhaseDelete:
		return action == opsisdk.ActionTypeDeleted || action == opsisdk.ActionTypeInProgress
	default:
		return true
	}
}

func handleOperationsInsightsWarehouseUserDeleteError(resource *opsiv1beta1.OperationsInsightsWarehouseUser, err error) error {
	if err == nil {
		return nil
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	if classification := errorutil.ClassifyDeleteError(err); classification.IsAuthShapedNotFound() {
		return operationsInsightsWarehouseUserAmbiguousNotFoundError{
			message:      fmt.Sprintf("OperationsInsightsWarehouseUser delete returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", err),
			opcRequestID: errorutil.OpcRequestID(err),
		}
	}
	return err
}

func operationsInsightsWarehouseUserStatusFromResponse(resource *opsiv1beta1.OperationsInsightsWarehouseUser, response any) error {
	current, ok := operationsInsightsWarehouseUserFromResponse(response)
	if !ok {
		return nil
	}
	operationsInsightsWarehouseUserProjectSDK(resource, current)
	return nil
}

func operationsInsightsWarehouseUserProjectSDK(resource *opsiv1beta1.OperationsInsightsWarehouseUser, current opsisdk.OperationsInsightsWarehouseUser) {
	if resource == nil {
		return
	}
	resource.Status.OperationsInsightsWarehouseId = operationsInsightsWarehouseUserString(current.OperationsInsightsWarehouseId)
	resource.Status.Id = operationsInsightsWarehouseUserString(current.Id)
	resource.Status.CompartmentId = operationsInsightsWarehouseUserString(current.CompartmentId)
	resource.Status.Name = operationsInsightsWarehouseUserString(current.Name)
	if current.IsAwrDataAccess != nil {
		resource.Status.IsAwrDataAccess = *current.IsAwrDataAccess
	}
	if current.IsEmDataAccess != nil {
		resource.Status.IsEmDataAccess = *current.IsEmDataAccess
	}
	if current.IsOpsiDataAccess != nil {
		resource.Status.IsOpsiDataAccess = *current.IsOpsiDataAccess
	}
	resource.Status.LifecycleState = string(current.LifecycleState)
	resource.Status.LifecycleDetails = operationsInsightsWarehouseUserString(current.LifecycleDetails)
	resource.Status.ConnectionPassword = ""
	resource.Status.FreeformTags = cloneOperationsInsightsWarehouseUserStringMap(current.FreeformTags)
	resource.Status.DefinedTags = operationsInsightsWarehouseUserStatusDefinedTags(current.DefinedTags)
	resource.Status.SystemTags = operationsInsightsWarehouseUserStatusDefinedTags(current.SystemTags)
	if current.TimeCreated != nil {
		resource.Status.TimeCreated = current.TimeCreated.String()
	}
	if current.TimeUpdated != nil {
		resource.Status.TimeUpdated = current.TimeUpdated.String()
	}
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
}

func operationsInsightsWarehouseUserFromResponse(response any) (opsisdk.OperationsInsightsWarehouseUser, bool) {
	if current, ok := operationsInsightsWarehouseUserFromDirectResponse(response); ok {
		return current, true
	}
	if current, ok := operationsInsightsWarehouseUserFromSummaryResponse(response); ok {
		return current, true
	}
	return operationsInsightsWarehouseUserFromOperationResponse(response)
}

func operationsInsightsWarehouseUserFromDirectResponse(response any) (opsisdk.OperationsInsightsWarehouseUser, bool) {
	switch current := response.(type) {
	case opsisdk.OperationsInsightsWarehouseUser:
		return current, true
	case *opsisdk.OperationsInsightsWarehouseUser:
		if current == nil {
			return opsisdk.OperationsInsightsWarehouseUser{}, false
		}
		return *current, true
	default:
		return opsisdk.OperationsInsightsWarehouseUser{}, false
	}
}

func operationsInsightsWarehouseUserFromSummaryResponse(response any) (opsisdk.OperationsInsightsWarehouseUser, bool) {
	switch current := response.(type) {
	case opsisdk.OperationsInsightsWarehouseUserSummary:
		return operationsInsightsWarehouseUserFromSummary(current), true
	case *opsisdk.OperationsInsightsWarehouseUserSummary:
		if current == nil {
			return opsisdk.OperationsInsightsWarehouseUser{}, false
		}
		return operationsInsightsWarehouseUserFromSummary(*current), true
	default:
		return opsisdk.OperationsInsightsWarehouseUser{}, false
	}
}

func operationsInsightsWarehouseUserFromOperationResponse(response any) (opsisdk.OperationsInsightsWarehouseUser, bool) {
	switch current := response.(type) {
	case opsisdk.CreateOperationsInsightsWarehouseUserResponse:
		return current.OperationsInsightsWarehouseUser, operationsInsightsWarehouseUserHasIdentity(current.OperationsInsightsWarehouseUser)
	case *opsisdk.CreateOperationsInsightsWarehouseUserResponse:
		if current == nil {
			return opsisdk.OperationsInsightsWarehouseUser{}, false
		}
		return current.OperationsInsightsWarehouseUser, operationsInsightsWarehouseUserHasIdentity(current.OperationsInsightsWarehouseUser)
	case opsisdk.GetOperationsInsightsWarehouseUserResponse:
		return current.OperationsInsightsWarehouseUser, operationsInsightsWarehouseUserHasIdentity(current.OperationsInsightsWarehouseUser)
	case *opsisdk.GetOperationsInsightsWarehouseUserResponse:
		if current == nil {
			return opsisdk.OperationsInsightsWarehouseUser{}, false
		}
		return current.OperationsInsightsWarehouseUser, operationsInsightsWarehouseUserHasIdentity(current.OperationsInsightsWarehouseUser)
	default:
		return opsisdk.OperationsInsightsWarehouseUser{}, false
	}
}

func operationsInsightsWarehouseUserFromSummary(summary opsisdk.OperationsInsightsWarehouseUserSummary) opsisdk.OperationsInsightsWarehouseUser {
	return opsisdk.OperationsInsightsWarehouseUser{
		OperationsInsightsWarehouseId: summary.OperationsInsightsWarehouseId,
		Id:                            summary.Id,
		CompartmentId:                 summary.CompartmentId,
		Name:                          summary.Name,
		IsAwrDataAccess:               summary.IsAwrDataAccess,
		TimeCreated:                   summary.TimeCreated,
		LifecycleState:                summary.LifecycleState,
		ConnectionPassword:            summary.ConnectionPassword,
		IsEmDataAccess:                summary.IsEmDataAccess,
		IsOpsiDataAccess:              summary.IsOpsiDataAccess,
		FreeformTags:                  cloneOperationsInsightsWarehouseUserStringMap(summary.FreeformTags),
		DefinedTags:                   cloneOperationsInsightsWarehouseUserNestedMap(summary.DefinedTags),
		SystemTags:                    cloneOperationsInsightsWarehouseUserNestedMap(summary.SystemTags),
		TimeUpdated:                   summary.TimeUpdated,
		LifecycleDetails:              summary.LifecycleDetails,
	}
}

func operationsInsightsWarehouseUserHasIdentity(current opsisdk.OperationsInsightsWarehouseUser) bool {
	return operationsInsightsWarehouseUserString(current.Id) != ""
}

func operationsInsightsWarehouseUserMatchesSpec(current opsisdk.OperationsInsightsWarehouseUser, spec opsiv1beta1.OperationsInsightsWarehouseUserSpec) bool {
	return operationsInsightsWarehouseUserString(current.OperationsInsightsWarehouseId) == strings.TrimSpace(spec.OperationsInsightsWarehouseId) &&
		operationsInsightsWarehouseUserString(current.CompartmentId) == strings.TrimSpace(spec.CompartmentId) &&
		operationsInsightsWarehouseUserString(current.Name) == strings.TrimSpace(spec.Name)
}

func operationsInsightsWarehouseUserCanBindExisting(current opsisdk.OperationsInsightsWarehouseUser) bool {
	return !operationsInsightsWarehouseUserLifecycleDeleting(current.LifecycleState) && !operationsInsightsWarehouseUserLifecycleDeleted(current.LifecycleState)
}

func operationsInsightsWarehouseUserLifecycleDeleting(state opsisdk.OperationsInsightsWarehouseUserLifecycleStateEnum) bool {
	return state == opsisdk.OperationsInsightsWarehouseUserLifecycleStateDeleting
}

func operationsInsightsWarehouseUserLifecycleDeleted(state opsisdk.OperationsInsightsWarehouseUserLifecycleStateEnum) bool {
	return state == opsisdk.OperationsInsightsWarehouseUserLifecycleStateDeleted
}

func desiredOperationsInsightsWarehouseUserPasswordForUpdate(resource *opsiv1beta1.OperationsInsightsWarehouseUser, current *string) (*string, bool) {
	if resource == nil || resource.Spec.ConnectionPassword == "" {
		return nil, false
	}
	if current != nil {
		currentValue := strings.TrimSpace(*current)
		if !operationsInsightsWarehouseUserPasswordReadbackRedacted(currentValue) {
			if currentValue == resource.Spec.ConnectionPassword {
				return nil, false
			}
			return common.String(resource.Spec.ConnectionPassword), true
		}
	}
	desiredFingerprint, ok := operationsInsightsWarehouseUserDesiredPasswordFingerprint(resource)
	if !ok {
		return nil, false
	}
	appliedFingerprint := operationsInsightsWarehouseUserAppliedPasswordFingerprint(resource)
	if appliedFingerprint == "" || appliedFingerprint == desiredFingerprint {
		return nil, false
	}
	return common.String(resource.Spec.ConnectionPassword), true
}

func operationsInsightsWarehouseUserPasswordReadbackRedacted(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return true
	}
	switch normalized {
	case "redacted", "<redacted>", "[redacted]", "(redacted)", "********", "*****":
		return true
	default:
		return strings.Trim(normalized, "*") == ""
	}
}

func operationsInsightsWarehouseUserAppliedPasswordFingerprint(resource *opsiv1beta1.OperationsInsightsWarehouseUser) string {
	if resource == nil {
		return ""
	}
	conditions := resource.Status.OsokStatus.Conditions
	for i := len(conditions) - 1; i >= 0; i-- {
		reason := strings.TrimSpace(conditions[i].Reason)
		if strings.HasPrefix(reason, operationsInsightsWarehouseUserPasswordReasonPrefix) {
			return strings.TrimPrefix(reason, operationsInsightsWarehouseUserPasswordReasonPrefix)
		}
	}
	return ""
}

func captureOperationsInsightsWarehouseUserSentPassword(
	ctx context.Context,
	phase shared.OSOKAsyncPhase,
	password *string,
	workRequestID string,
) {
	capture, _ := ctx.Value(operationsInsightsWarehouseUserPasswordCaptureContextKey{}).(*operationsInsightsWarehouseUserPasswordCapture)
	if capture == nil || capture.resource == nil || password == nil || strings.TrimSpace(*password) == "" {
		return
	}
	fingerprint, ok := operationsInsightsWarehouseUserPasswordFingerprintForValue(capture.resource, *password)
	if !ok {
		return
	}
	capture.sent = &operationsInsightsWarehouseUserPasswordFingerprintRecord{
		phase:         phase,
		workRequestID: strings.TrimSpace(workRequestID),
		fingerprint:   fingerprint,
	}
}

func reconcileOperationsInsightsWarehouseUserPasswordFingerprint(
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	capture *operationsInsightsWarehouseUserPasswordCapture,
	previousWorkRequestID string,
	previousPhase shared.OSOKAsyncPhase,
) {
	if resource == nil {
		return
	}
	if capture != nil && capture.sent != nil {
		record := *capture.sent
		if current := resource.Status.OsokStatus.Async.Current; current != nil && current.Phase == record.phase && current.WorkRequestID == record.workRequestID {
			recordOperationsInsightsWarehouseUserPendingPasswordFingerprint(resource, record)
			return
		}
		recordOperationsInsightsWarehouseUserAppliedPasswordFingerprint(resource, record.fingerprint)
		return
	}
	if resource.Status.OsokStatus.Async.Current != nil {
		return
	}
	record, ok := operationsInsightsWarehouseUserPendingPasswordFingerprint(resource, previousWorkRequestID, previousPhase)
	if !ok {
		return
	}
	recordOperationsInsightsWarehouseUserAppliedPasswordFingerprint(resource, record.fingerprint)
}

func recordOperationsInsightsWarehouseUserAppliedPasswordFingerprint(resource *opsiv1beta1.OperationsInsightsWarehouseUser, fingerprint string) {
	if resource == nil {
		return
	}
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" || len(resource.Status.OsokStatus.Conditions) == 0 {
		return
	}
	conditions := resource.Status.OsokStatus.Conditions
	conditions[len(conditions)-1].Reason = operationsInsightsWarehouseUserPasswordReasonPrefix + fingerprint
	resource.Status.OsokStatus.Conditions = conditions
}

func operationsInsightsWarehouseUserDesiredPasswordFingerprint(resource *opsiv1beta1.OperationsInsightsWarehouseUser) (string, bool) {
	if resource == nil || resource.Spec.ConnectionPassword == "" {
		return "", false
	}
	return operationsInsightsWarehouseUserPasswordFingerprintForValue(resource, resource.Spec.ConnectionPassword)
}

func operationsInsightsWarehouseUserPasswordFingerprintForValue(resource *opsiv1beta1.OperationsInsightsWarehouseUser, password string) (string, bool) {
	if resource == nil || password == "" {
		return "", false
	}
	hash := sha256.New()
	for _, value := range []string{
		string(resource.UID),
		strings.TrimSpace(resource.Namespace),
		strings.TrimSpace(resource.Name),
		strings.TrimSpace(resource.Spec.OperationsInsightsWarehouseId),
		strings.TrimSpace(resource.Spec.CompartmentId),
		strings.TrimSpace(resource.Spec.Name),
		password,
	} {
		_, _ = hash.Write([]byte(value))
		_, _ = hash.Write([]byte{0})
	}
	sum := hash.Sum(nil)
	return fmt.Sprintf("%x", sum[:16]), true
}

func recordOperationsInsightsWarehouseUserPendingPasswordFingerprint(
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	record operationsInsightsWarehouseUserPasswordFingerprintRecord,
) {
	if resource == nil || len(resource.Status.OsokStatus.Conditions) == 0 {
		return
	}
	if strings.TrimSpace(record.fingerprint) == "" || strings.TrimSpace(record.workRequestID) == "" || record.phase == "" {
		return
	}
	conditions := resource.Status.OsokStatus.Conditions
	conditions[len(conditions)-1].Reason = operationsInsightsWarehouseUserPendingPasswordReasonPrefix +
		string(record.phase) + ":" + strings.TrimSpace(record.workRequestID) + ":" + strings.TrimSpace(record.fingerprint)
	resource.Status.OsokStatus.Conditions = conditions
}

func operationsInsightsWarehouseUserPendingPasswordFingerprint(
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (operationsInsightsWarehouseUserPasswordFingerprintRecord, bool) {
	if resource == nil || strings.TrimSpace(workRequestID) == "" || phase == "" {
		return operationsInsightsWarehouseUserPasswordFingerprintRecord{}, false
	}
	conditions := resource.Status.OsokStatus.Conditions
	for i := len(conditions) - 1; i >= 0; i-- {
		record, ok := parseOperationsInsightsWarehouseUserPendingPasswordFingerprint(conditions[i].Reason)
		if !ok {
			continue
		}
		if record.workRequestID == strings.TrimSpace(workRequestID) && record.phase == phase {
			return record, true
		}
	}
	return operationsInsightsWarehouseUserPasswordFingerprintRecord{}, false
}

func parseOperationsInsightsWarehouseUserPendingPasswordFingerprint(reason string) (operationsInsightsWarehouseUserPasswordFingerprintRecord, bool) {
	raw := strings.TrimSpace(reason)
	if !strings.HasPrefix(raw, operationsInsightsWarehouseUserPendingPasswordReasonPrefix) {
		return operationsInsightsWarehouseUserPasswordFingerprintRecord{}, false
	}
	parts := strings.SplitN(strings.TrimPrefix(raw, operationsInsightsWarehouseUserPendingPasswordReasonPrefix), ":", 3)
	if len(parts) != 3 {
		return operationsInsightsWarehouseUserPasswordFingerprintRecord{}, false
	}
	phase := shared.OSOKAsyncPhase(strings.TrimSpace(parts[0]))
	workRequestID := strings.TrimSpace(parts[1])
	fingerprint := strings.TrimSpace(parts[2])
	if phase == "" || workRequestID == "" || fingerprint == "" {
		return operationsInsightsWarehouseUserPasswordFingerprintRecord{}, false
	}
	return operationsInsightsWarehouseUserPasswordFingerprintRecord{
		phase:         phase,
		workRequestID: workRequestID,
		fingerprint:   fingerprint,
	}, true
}

func desiredOperationsInsightsWarehouseUserRequiredBoolForUpdate(spec bool, current *bool) (*bool, bool) {
	if current != nil && *current == spec {
		return nil, false
	}
	return common.Bool(spec), true
}

func desiredOperationsInsightsWarehouseUserOptionalBoolForUpdate(spec bool, current *bool) (*bool, bool) {
	if current == nil && !spec {
		return nil, false
	}
	if current != nil && *current == spec {
		return nil, false
	}
	return common.Bool(spec), true
}

func desiredOperationsInsightsWarehouseUserFreeformTagsForUpdate(spec map[string]string, current map[string]string) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	desired := cloneOperationsInsightsWarehouseUserStringMap(spec)
	if reflect.DeepEqual(current, desired) {
		return nil, false
	}
	return desired, true
}

func desiredOperationsInsightsWarehouseUserDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}
	desired := operationsInsightsWarehouseUserDefinedTagsFromSpec(spec)
	if reflect.DeepEqual(current, desired) {
		return nil, false
	}
	return desired, true
}

func operationsInsightsWarehouseUserDefinedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&tags)
}

func operationsInsightsWarehouseUserStatusDefinedTags(tags map[string]map[string]interface{}) map[string]shared.MapValue {
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

func cloneOperationsInsightsWarehouseUserStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneOperationsInsightsWarehouseUserNestedMap(source map[string]map[string]interface{}) map[string]map[string]interface{} {
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

func currentOperationsInsightsWarehouseUserID(resource *opsiv1beta1.OperationsInsightsWarehouseUser) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func currentOperationsInsightsWarehouseUserWorkRequest(resource *opsiv1beta1.OperationsInsightsWarehouseUser) (string, shared.OSOKAsyncPhase) {
	if resource == nil {
		return "", ""
	}
	return servicemanager.ResolveTrackedWorkRequest(&resource.Status.OsokStatus, resource, servicemanager.WorkRequestLegacyBridge{}, "")
}

func markOperationsInsightsWarehouseUserWorkRequest(resource *opsiv1beta1.OperationsInsightsWarehouseUser, current *shared.OSOKAsyncOperation, log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: operationsInsightsWarehouseUserRequeueDuration,
	}
}

func markOperationsInsightsWarehouseUserWriteReadbackPending(
	resource *opsiv1beta1.OperationsInsightsWarehouseUser,
	current *shared.OSOKAsyncOperation,
	resourceID string,
	log loggerutil.OSOKLogger,
) {
	if resource == nil || current == nil {
		return
	}
	now := metav1.Now()
	next := *current
	next.NormalizedClass = shared.OSOKAsyncClassPending
	next.Message = operationsInsightsWarehouseUserWriteReadbackPendingMessage(next.Phase, next.WorkRequestID, resourceID)
	next.UpdatedAt = &now
	if strings.TrimSpace(resourceID) != "" {
		resource.Status.Id = strings.TrimSpace(resourceID)
		resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(resourceID))
	}
	markOperationsInsightsWarehouseUserWorkRequest(resource, &next, log)
}

func operationsInsightsWarehouseUserWriteReadbackPendingMessage(
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	resourceID string,
) string {
	if strings.TrimSpace(resourceID) == "" {
		return fmt.Sprintf("OperationsInsightsWarehouseUser %s work request %s succeeded; waiting for readback before delete", phase, workRequestID)
	}
	return fmt.Sprintf("OperationsInsightsWarehouseUser %s work request %s succeeded for %s; waiting for readback before delete", phase, workRequestID, resourceID)
}

func markOperationsInsightsWarehouseUserTerminating(resource *opsiv1beta1.OperationsInsightsWarehouseUser, message string, log loggerutil.OSOKLogger) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, log)
	scrubOperationsInsightsWarehouseUserSecretStatus(resource)
}

func markOperationsInsightsWarehouseUserDeleted(resource *opsiv1beta1.OperationsInsightsWarehouseUser, message string, log loggerutil.OSOKLogger) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, log)
	scrubOperationsInsightsWarehouseUserSecretStatus(resource)
}

func markOperationsInsightsWarehouseUserFailure(resource *opsiv1beta1.OperationsInsightsWarehouseUser, err error, log loggerutil.OSOKLogger) error {
	if resource == nil || err == nil {
		return err
	}
	status := &resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(status, err)
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, corev1.ConditionFalse, "", err.Error(), log)
	scrubOperationsInsightsWarehouseUserSecretStatus(resource)
	return err
}

func scrubOperationsInsightsWarehouseUserSecretStatus(resource *opsiv1beta1.OperationsInsightsWarehouseUser) {
	if resource == nil {
		return
	}
	resource.Status.ConnectionPassword = ""
}

func operationsInsightsWarehouseUserString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
