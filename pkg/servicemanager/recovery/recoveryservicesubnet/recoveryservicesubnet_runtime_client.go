/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package recoveryservicesubnet

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	recoverysdk "github.com/oracle/oci-go-sdk/v65/recovery"
	recoveryv1beta1 "github.com/oracle/oci-service-operator/api/recovery/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

const recoveryServiceSubnetWorkRequestEntityType = "recoveryServiceSubnet"

var recoveryServiceSubnetWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(recoverysdk.OperationStatusAccepted),
		string(recoverysdk.OperationStatusWaiting),
		string(recoverysdk.OperationStatusInProgress),
		string(recoverysdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(recoverysdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(recoverysdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(recoverysdk.OperationStatusCanceled)},
	CreateActionTokens:    []string{string(recoverysdk.ActionTypeCreated)},
	UpdateActionTokens:    []string{string(recoverysdk.ActionTypeUpdated)},
	DeleteActionTokens:    []string{string(recoverysdk.ActionTypeDeleted)},
}

type recoveryServiceSubnetOCIClient interface {
	ChangeRecoveryServiceSubnetCompartment(context.Context, recoverysdk.ChangeRecoveryServiceSubnetCompartmentRequest) (recoverysdk.ChangeRecoveryServiceSubnetCompartmentResponse, error)
	CreateRecoveryServiceSubnet(context.Context, recoverysdk.CreateRecoveryServiceSubnetRequest) (recoverysdk.CreateRecoveryServiceSubnetResponse, error)
	GetRecoveryServiceSubnet(context.Context, recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error)
	ListRecoveryServiceSubnets(context.Context, recoverysdk.ListRecoveryServiceSubnetsRequest) (recoverysdk.ListRecoveryServiceSubnetsResponse, error)
	UpdateRecoveryServiceSubnet(context.Context, recoverysdk.UpdateRecoveryServiceSubnetRequest) (recoverysdk.UpdateRecoveryServiceSubnetResponse, error)
	DeleteRecoveryServiceSubnet(context.Context, recoverysdk.DeleteRecoveryServiceSubnetRequest) (recoverysdk.DeleteRecoveryServiceSubnetResponse, error)
	GetWorkRequest(context.Context, recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error)
}

type recoveryServiceSubnetCompartmentMoveClient interface {
	ChangeRecoveryServiceSubnetCompartment(context.Context, recoverysdk.ChangeRecoveryServiceSubnetCompartmentRequest) (recoverysdk.ChangeRecoveryServiceSubnetCompartmentResponse, error)
	GetWorkRequest(context.Context, recoverysdk.GetWorkRequestRequest) (recoverysdk.GetWorkRequestResponse, error)
}

type recoveryServiceSubnetPendingWriteDeleteClient struct {
	delegate                 RecoveryServiceSubnetServiceClient
	getRecoveryServiceSubnet func(context.Context, recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error)
}

type ambiguousRecoveryServiceSubnetNotFoundError struct {
	message      string
	opcRequestID string
}

func (e ambiguousRecoveryServiceSubnetNotFoundError) Error() string {
	return e.message
}

func (e ambiguousRecoveryServiceSubnetNotFoundError) GetOpcRequestID() string {
	return e.opcRequestID
}

func init() {
	registerRecoveryServiceSubnetRuntimeHooksMutator(func(manager *RecoveryServiceSubnetServiceManager, hooks *RecoveryServiceSubnetRuntimeHooks) {
		workRequestClient, initErr := newRecoveryServiceSubnetWorkRequestClient(manager)
		applyRecoveryServiceSubnetRuntimeHooks(hooks, workRequestClient, initErr)
	})
}

func newRecoveryServiceSubnetWorkRequestClient(manager *RecoveryServiceSubnetServiceManager) (recoveryServiceSubnetCompartmentMoveClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("recovery service subnet manager is nil")
	}
	client, err := recoverysdk.NewDatabaseRecoveryClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyRecoveryServiceSubnetRuntimeHooks(
	hooks *RecoveryServiceSubnetRuntimeHooks,
	workRequestClient recoveryServiceSubnetCompartmentMoveClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = recoveryServiceSubnetRuntimeSemantics()
	hooks.BuildCreateBody = buildRecoveryServiceSubnetCreateBody
	hooks.BuildUpdateBody = buildRecoveryServiceSubnetUpdateBody
	hooks.Create.Fields = recoveryServiceSubnetCreateFields()
	hooks.Get.Fields = recoveryServiceSubnetGetFields()
	hooks.List.Fields = recoveryServiceSubnetListFields()
	hooks.Update.Fields = recoveryServiceSubnetUpdateFields()
	hooks.Delete.Fields = recoveryServiceSubnetDeleteFields()

	getCall := hooks.Get.Call
	hooks.Get.Call = func(ctx context.Context, request recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error) {
		if getCall == nil {
			return recoverysdk.GetRecoveryServiceSubnetResponse{}, fmt.Errorf("recovery service subnet GetRecoveryServiceSubnet call is not configured")
		}
		response, err := getCall(ctx, request)
		return response, conservativeRecoveryServiceSubnetNotFoundError(err, "get")
	}
	listCall := hooks.List.Call
	hooks.List.Call = func(ctx context.Context, request recoverysdk.ListRecoveryServiceSubnetsRequest) (recoverysdk.ListRecoveryServiceSubnetsResponse, error) {
		return listRecoveryServiceSubnetsAllPages(ctx, listCall, request)
	}

	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedRecoveryServiceSubnetIdentity
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateRecoveryServiceSubnetCreateOnlyDriftForResponse
	hooks.ParityHooks.RequiresParityHandling = recoveryServiceSubnetRequiresCompartmentMove
	hooks.ParityHooks.ApplyParityUpdate = func(
		ctx context.Context,
		resource *recoveryv1beta1.RecoveryServiceSubnet,
		currentResponse any,
	) (servicemanager.OSOKResponse, error) {
		return applyRecoveryServiceSubnetCompartmentMove(ctx, resource, currentResponse, workRequestClient, initErr)
	}
	hooks.DeleteHooks.HandleError = handleRecoveryServiceSubnetDeleteError
	hooks.Async.Adapter = recoveryServiceSubnetWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getRecoveryServiceSubnetWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveRecoveryServiceSubnetWorkRequestAction
	hooks.Async.ResolvePhase = resolveRecoveryServiceSubnetWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverRecoveryServiceSubnetIDFromWorkRequest
	hooks.Async.Message = recoveryServiceSubnetWorkRequestMessage
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate RecoveryServiceSubnetServiceClient) RecoveryServiceSubnetServiceClient {
		return recoveryServiceSubnetPendingWriteDeleteClient{delegate: delegate, getRecoveryServiceSubnet: hooks.Get.Call}
	})
}

func newRecoveryServiceSubnetServiceClientWithOCIClient(log loggerutil.OSOKLogger, client recoveryServiceSubnetOCIClient) RecoveryServiceSubnetServiceClient {
	manager := &RecoveryServiceSubnetServiceManager{Log: log}
	hooks := newRecoveryServiceSubnetRuntimeHooksWithOCIClient(client)
	applyRecoveryServiceSubnetRuntimeHooks(&hooks, client, nil)
	config := buildRecoveryServiceSubnetGeneratedRuntimeConfig(manager, hooks)
	delegate := defaultRecoveryServiceSubnetServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*recoveryv1beta1.RecoveryServiceSubnet](config),
	}
	return wrapRecoveryServiceSubnetGeneratedClient(hooks, delegate)
}

func newRecoveryServiceSubnetRuntimeHooksWithOCIClient(client recoveryServiceSubnetOCIClient) RecoveryServiceSubnetRuntimeHooks {
	return RecoveryServiceSubnetRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*recoveryv1beta1.RecoveryServiceSubnet]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*recoveryv1beta1.RecoveryServiceSubnet]{},
		StatusHooks:     generatedruntime.StatusHooks[*recoveryv1beta1.RecoveryServiceSubnet]{},
		ParityHooks:     generatedruntime.ParityHooks[*recoveryv1beta1.RecoveryServiceSubnet]{},
		Async:           generatedruntime.AsyncHooks[*recoveryv1beta1.RecoveryServiceSubnet]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*recoveryv1beta1.RecoveryServiceSubnet]{},
		Create: runtimeOperationHooks[recoverysdk.CreateRecoveryServiceSubnetRequest, recoverysdk.CreateRecoveryServiceSubnetResponse]{
			Fields: recoveryServiceSubnetCreateFields(),
			Call: func(ctx context.Context, request recoverysdk.CreateRecoveryServiceSubnetRequest) (recoverysdk.CreateRecoveryServiceSubnetResponse, error) {
				if client == nil {
					return recoverysdk.CreateRecoveryServiceSubnetResponse{}, fmt.Errorf("recovery service subnet OCI client is nil")
				}
				return client.CreateRecoveryServiceSubnet(ctx, request)
			},
		},
		Get: runtimeOperationHooks[recoverysdk.GetRecoveryServiceSubnetRequest, recoverysdk.GetRecoveryServiceSubnetResponse]{
			Fields: recoveryServiceSubnetGetFields(),
			Call: func(ctx context.Context, request recoverysdk.GetRecoveryServiceSubnetRequest) (recoverysdk.GetRecoveryServiceSubnetResponse, error) {
				if client == nil {
					return recoverysdk.GetRecoveryServiceSubnetResponse{}, fmt.Errorf("recovery service subnet OCI client is nil")
				}
				return client.GetRecoveryServiceSubnet(ctx, request)
			},
		},
		List: runtimeOperationHooks[recoverysdk.ListRecoveryServiceSubnetsRequest, recoverysdk.ListRecoveryServiceSubnetsResponse]{
			Fields: recoveryServiceSubnetListFields(),
			Call: func(ctx context.Context, request recoverysdk.ListRecoveryServiceSubnetsRequest) (recoverysdk.ListRecoveryServiceSubnetsResponse, error) {
				if client == nil {
					return recoverysdk.ListRecoveryServiceSubnetsResponse{}, fmt.Errorf("recovery service subnet OCI client is nil")
				}
				return client.ListRecoveryServiceSubnets(ctx, request)
			},
		},
		Update: runtimeOperationHooks[recoverysdk.UpdateRecoveryServiceSubnetRequest, recoverysdk.UpdateRecoveryServiceSubnetResponse]{
			Fields: recoveryServiceSubnetUpdateFields(),
			Call: func(ctx context.Context, request recoverysdk.UpdateRecoveryServiceSubnetRequest) (recoverysdk.UpdateRecoveryServiceSubnetResponse, error) {
				if client == nil {
					return recoverysdk.UpdateRecoveryServiceSubnetResponse{}, fmt.Errorf("recovery service subnet OCI client is nil")
				}
				return client.UpdateRecoveryServiceSubnet(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[recoverysdk.DeleteRecoveryServiceSubnetRequest, recoverysdk.DeleteRecoveryServiceSubnetResponse]{
			Fields: recoveryServiceSubnetDeleteFields(),
			Call: func(ctx context.Context, request recoverysdk.DeleteRecoveryServiceSubnetRequest) (recoverysdk.DeleteRecoveryServiceSubnetResponse, error) {
				if client == nil {
					return recoverysdk.DeleteRecoveryServiceSubnetResponse{}, fmt.Errorf("recovery service subnet OCI client is nil")
				}
				return client.DeleteRecoveryServiceSubnet(ctx, request)
			},
		},
		WrapGeneratedClient: []func(RecoveryServiceSubnetServiceClient) RecoveryServiceSubnetServiceClient{},
	}
}

func recoveryServiceSubnetRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "recovery",
		FormalSlug:    "recoveryservicesubnet",
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
			ProvisioningStates: []string{string(recoverysdk.LifecycleStateCreating)},
			UpdatingStates:     []string{string(recoverysdk.LifecycleStateUpdating)},
			ActiveStates:       []string{string(recoverysdk.LifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required",
			PendingStates: []string{
				string(recoverysdk.LifecycleStateDeleteScheduled),
				string(recoverysdk.LifecycleStateDeleting),
			},
			TerminalStates: []string{string(recoverysdk.LifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "vcnId", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"compartmentId",
				"displayName",
				"subnets",
				"nsgIds",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"vcnId",
				"subnetId",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: recoveryServiceSubnetWorkRequestEntityType, Action: "CREATED"},
			},
			Update: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: recoveryServiceSubnetWorkRequestEntityType, Action: "UPDATED"},
			},
			Delete: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: recoveryServiceSubnetWorkRequestEntityType, Action: "DELETED"},
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.CreateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: recoveryServiceSubnetWorkRequestEntityType, Action: "CREATED"},
			},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.UpdateResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: recoveryServiceSubnetWorkRequestEntityType, Action: "UPDATED"},
			},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks: []generatedruntime.Hook{
				{Helper: "tfresource.DeleteResource"},
				{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: recoveryServiceSubnetWorkRequestEntityType, Action: "DELETED"},
			},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func recoveryServiceSubnetCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateRecoveryServiceSubnetDetails", RequestName: "CreateRecoveryServiceSubnetDetails", Contribution: "body"},
	}
}

func recoveryServiceSubnetGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "RecoveryServiceSubnetId", RequestName: "recoveryServiceSubnetId", Contribution: "path", PreferResourceID: true},
	}
}

func recoveryServiceSubnetListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"status.compartmentId", "spec.compartmentId", "compartmentId"}},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
		{FieldName: "VcnId", RequestName: "vcnId", Contribution: "query", LookupPaths: []string{"status.vcnId", "spec.vcnId", "vcnId"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
	}
}

func recoveryServiceSubnetUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "RecoveryServiceSubnetId", RequestName: "recoveryServiceSubnetId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateRecoveryServiceSubnetDetails", RequestName: "UpdateRecoveryServiceSubnetDetails", Contribution: "body"},
	}
}

func recoveryServiceSubnetDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "RecoveryServiceSubnetId", RequestName: "recoveryServiceSubnetId", Contribution: "path", PreferResourceID: true},
	}
}

func buildRecoveryServiceSubnetCreateBody(
	_ context.Context,
	resource *recoveryv1beta1.RecoveryServiceSubnet,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("recovery service subnet resource is nil")
	}
	if err := validateRecoveryServiceSubnetSpec(resource.Spec); err != nil {
		return nil, err
	}

	spec := resource.Spec
	details := recoverysdk.CreateRecoveryServiceSubnetDetails{
		DisplayName:   common.String(strings.TrimSpace(spec.DisplayName)),
		VcnId:         common.String(strings.TrimSpace(spec.VcnId)),
		CompartmentId: common.String(strings.TrimSpace(spec.CompartmentId)),
	}
	if subnetID := strings.TrimSpace(spec.SubnetId); subnetID != "" {
		details.SubnetId = common.String(subnetID)
	}
	if spec.Subnets != nil {
		details.Subnets = cloneRecoveryServiceSubnetStringSlice(spec.Subnets)
	}
	if spec.NsgIds != nil {
		details.NsgIds = cloneRecoveryServiceSubnetStringSlice(spec.NsgIds)
	}
	if spec.FreeformTags != nil {
		details.FreeformTags = cloneRecoveryServiceSubnetStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		details.DefinedTags = recoveryServiceSubnetDefinedTagsFromSpec(spec.DefinedTags)
	}
	return details, nil
}

func buildRecoveryServiceSubnetUpdateBody(
	_ context.Context,
	resource *recoveryv1beta1.RecoveryServiceSubnet,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return recoverysdk.UpdateRecoveryServiceSubnetDetails{}, false, fmt.Errorf("recovery service subnet resource is nil")
	}
	if err := validateRecoveryServiceSubnetSpec(resource.Spec); err != nil {
		return recoverysdk.UpdateRecoveryServiceSubnetDetails{}, false, err
	}
	current, ok := recoveryServiceSubnetFromResponse(currentResponse)
	if !ok {
		return recoverysdk.UpdateRecoveryServiceSubnetDetails{}, false, fmt.Errorf("current recovery service subnet response does not expose a resource body")
	}
	if err := validateRecoveryServiceSubnetCreateOnlyDrift(resource.Spec, current); err != nil {
		return recoverysdk.UpdateRecoveryServiceSubnetDetails{}, false, err
	}

	details, updateNeeded := buildRecoveryServiceSubnetUpdateDetails(resource.Spec, current)
	if !updateNeeded {
		return recoverysdk.UpdateRecoveryServiceSubnetDetails{}, false, nil
	}
	return details, true, nil
}

func buildRecoveryServiceSubnetUpdateDetails(
	spec recoveryv1beta1.RecoveryServiceSubnetSpec,
	current recoverysdk.RecoveryServiceSubnet,
) (recoverysdk.UpdateRecoveryServiceSubnetDetails, bool) {
	details := recoverysdk.UpdateRecoveryServiceSubnetDetails{}
	updateNeeded := recoveryServiceSubnetCompartmentNeedsMove(spec, current)

	if addRecoveryServiceSubnetDisplayNameUpdate(&details, spec, current) {
		updateNeeded = true
	}
	if addRecoveryServiceSubnetSubnetsUpdate(&details, spec, current) {
		updateNeeded = true
	}
	if addRecoveryServiceSubnetNsgIdsUpdate(&details, spec, current) {
		updateNeeded = true
	}
	if addRecoveryServiceSubnetFreeformTagsUpdate(&details, spec, current) {
		updateNeeded = true
	}
	if addRecoveryServiceSubnetDefinedTagsUpdate(&details, spec, current) {
		updateNeeded = true
	}

	return details, updateNeeded
}

func addRecoveryServiceSubnetDisplayNameUpdate(
	details *recoverysdk.UpdateRecoveryServiceSubnetDetails,
	spec recoveryv1beta1.RecoveryServiceSubnetSpec,
	current recoverysdk.RecoveryServiceSubnet,
) bool {
	if recoveryServiceSubnetStringPtrEqual(current.DisplayName, spec.DisplayName) {
		return false
	}
	details.DisplayName = common.String(strings.TrimSpace(spec.DisplayName))
	return true
}

func addRecoveryServiceSubnetSubnetsUpdate(
	details *recoverysdk.UpdateRecoveryServiceSubnetDetails,
	spec recoveryv1beta1.RecoveryServiceSubnetSpec,
	current recoverysdk.RecoveryServiceSubnet,
) bool {
	if spec.Subnets == nil || recoveryServiceSubnetStringSliceSetEqual(current.Subnets, spec.Subnets) {
		return false
	}
	details.Subnets = cloneRecoveryServiceSubnetStringSlice(spec.Subnets)
	return true
}

func addRecoveryServiceSubnetNsgIdsUpdate(
	details *recoverysdk.UpdateRecoveryServiceSubnetDetails,
	spec recoveryv1beta1.RecoveryServiceSubnetSpec,
	current recoverysdk.RecoveryServiceSubnet,
) bool {
	if spec.NsgIds == nil || recoveryServiceSubnetStringSliceSetEqual(current.NsgIds, spec.NsgIds) {
		return false
	}
	details.NsgIds = cloneRecoveryServiceSubnetStringSlice(spec.NsgIds)
	return true
}

func addRecoveryServiceSubnetFreeformTagsUpdate(
	details *recoverysdk.UpdateRecoveryServiceSubnetDetails,
	spec recoveryv1beta1.RecoveryServiceSubnetSpec,
	current recoverysdk.RecoveryServiceSubnet,
) bool {
	if spec.FreeformTags == nil || reflect.DeepEqual(current.FreeformTags, spec.FreeformTags) {
		return false
	}
	details.FreeformTags = cloneRecoveryServiceSubnetStringMap(spec.FreeformTags)
	return true
}

func addRecoveryServiceSubnetDefinedTagsUpdate(
	details *recoverysdk.UpdateRecoveryServiceSubnetDetails,
	spec recoveryv1beta1.RecoveryServiceSubnetSpec,
	current recoverysdk.RecoveryServiceSubnet,
) bool {
	if spec.DefinedTags == nil {
		return false
	}
	desired := recoveryServiceSubnetDefinedTagsFromSpec(spec.DefinedTags)
	if reflect.DeepEqual(current.DefinedTags, desired) {
		return false
	}
	details.DefinedTags = desired
	return true
}

func validateRecoveryServiceSubnetSpec(spec recoveryv1beta1.RecoveryServiceSubnetSpec) error {
	var missing []string
	if strings.TrimSpace(spec.DisplayName) == "" {
		missing = append(missing, "displayName")
	}
	if strings.TrimSpace(spec.VcnId) == "" {
		missing = append(missing, "vcnId")
	}
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if len(missing) > 0 {
		return fmt.Errorf("recovery service subnet spec is missing required field(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func validateRecoveryServiceSubnetCreateOnlyDriftForResponse(
	resource *recoveryv1beta1.RecoveryServiceSubnet,
	currentResponse any,
) error {
	if resource == nil {
		return fmt.Errorf("recovery service subnet resource is nil")
	}
	current, ok := recoveryServiceSubnetFromResponse(currentResponse)
	if !ok {
		return fmt.Errorf("current recovery service subnet response does not expose a resource body")
	}
	return validateRecoveryServiceSubnetCreateOnlyDrift(resource.Spec, current)
}

func validateRecoveryServiceSubnetCreateOnlyDrift(
	spec recoveryv1beta1.RecoveryServiceSubnetSpec,
	current recoverysdk.RecoveryServiceSubnet,
) error {
	var unsupported []string
	if !recoveryServiceSubnetStringPtrEqual(current.VcnId, spec.VcnId) {
		unsupported = append(unsupported, "vcnId")
	}
	if strings.TrimSpace(spec.SubnetId) != "" && !recoveryServiceSubnetStringPtrEqual(current.SubnetId, spec.SubnetId) {
		unsupported = append(unsupported, "subnetId")
	}
	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("recovery service subnet create-only field drift is not supported: %s", strings.Join(unsupported, ", "))
}

func recoveryServiceSubnetRequiresCompartmentMove(resource *recoveryv1beta1.RecoveryServiceSubnet, currentResponse any) bool {
	if resource == nil {
		return false
	}
	current, ok := recoveryServiceSubnetFromResponse(currentResponse)
	if !ok {
		return false
	}
	return recoveryServiceSubnetCompartmentNeedsMove(resource.Spec, current)
}

func recoveryServiceSubnetCompartmentNeedsMove(
	spec recoveryv1beta1.RecoveryServiceSubnetSpec,
	current recoverysdk.RecoveryServiceSubnet,
) bool {
	desired := strings.TrimSpace(spec.CompartmentId)
	observed := strings.TrimSpace(recoveryServiceSubnetStringValue(current.CompartmentId))
	return desired != "" && observed != "" && desired != observed
}

func applyRecoveryServiceSubnetCompartmentMove(
	ctx context.Context,
	resource *recoveryv1beta1.RecoveryServiceSubnet,
	currentResponse any,
	client recoveryServiceSubnetCompartmentMoveClient,
	initErr error,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("recovery service subnet resource is nil")
	}
	if initErr != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("initialize recovery service subnet OCI client: %w", initErr)
	}
	if client == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("recovery service subnet OCI client is not configured")
	}

	current, ok := recoveryServiceSubnetFromResponse(currentResponse)
	if !ok {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("current recovery service subnet response does not expose a resource body")
	}
	resourceID := strings.TrimSpace(recoveryServiceSubnetStringValue(current.Id))
	if resourceID == "" {
		resourceID = recoveryServiceSubnetCurrentID(resource)
	}
	if resourceID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("recovery service subnet compartment move requires a tracked recovery service subnet id")
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("recovery service subnet compartment move requires spec.compartmentId")
	}

	response, err := client.ChangeRecoveryServiceSubnetCompartment(ctx, recoverysdk.ChangeRecoveryServiceSubnetCompartmentRequest{
		RecoveryServiceSubnetId: common.String(resourceID),
		ChangeRecoveryServiceSubnetCompartmentDetails: recoverysdk.ChangeRecoveryServiceSubnetCompartmentDetails{
			CompartmentId: common.String(compartmentID),
		},
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	workRequestID := strings.TrimSpace(recoveryServiceSubnetStringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("recovery service subnet compartment move did not return an opc-work-request-id")
	}

	resource.Status.Id = resourceID
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, &shared.OSOKAsyncOperation{
		Source:           shared.OSOKAsyncSourceWorkRequest,
		Phase:            shared.OSOKAsyncPhaseUpdate,
		WorkRequestID:    workRequestID,
		RawOperationType: string(recoverysdk.OperationTypeMoveRecoveryServiceSubnet),
		NormalizedClass:  shared.OSOKAsyncClassPending,
		Message:          fmt.Sprintf("RecoveryServiceSubnet compartment move work request %s is pending", workRequestID),
	}, loggerutil.OSOKLogger{})
	return servicemanager.OSOKResponse{
		IsSuccessful:  projection.Condition != shared.Failed,
		ShouldRequeue: projection.ShouldRequeue,
	}, nil
}

func listRecoveryServiceSubnetsAllPages(
	ctx context.Context,
	listCall func(context.Context, recoverysdk.ListRecoveryServiceSubnetsRequest) (recoverysdk.ListRecoveryServiceSubnetsResponse, error),
	request recoverysdk.ListRecoveryServiceSubnetsRequest,
) (recoverysdk.ListRecoveryServiceSubnetsResponse, error) {
	if listCall == nil {
		return recoverysdk.ListRecoveryServiceSubnetsResponse{}, fmt.Errorf("recovery service subnet ListRecoveryServiceSubnets call is not configured")
	}

	var combined recoverysdk.ListRecoveryServiceSubnetsResponse
	for {
		response, err := listCall(ctx, request)
		if err != nil {
			return recoverysdk.ListRecoveryServiceSubnetsResponse{}, err
		}
		combined.RawResponse = response.RawResponse
		combined.OpcRequestId = response.OpcRequestId
		for _, item := range response.Items {
			if item.LifecycleState == recoverysdk.LifecycleStateDeleted {
				continue
			}
			combined.Items = append(combined.Items, item)
		}
		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			combined.OpcNextPage = nil
			return combined, nil
		}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func clearTrackedRecoveryServiceSubnetIdentity(resource *recoveryv1beta1.RecoveryServiceSubnet) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus = shared.OSOKStatus{}
}

func handleRecoveryServiceSubnetDeleteError(resource *recoveryv1beta1.RecoveryServiceSubnet, err error) error {
	err = conservativeRecoveryServiceSubnetNotFoundError(err, "delete")
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func conservativeRecoveryServiceSubnetNotFoundError(err error, operation string) error {
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
	return ambiguousRecoveryServiceSubnetNotFoundError{
		message: fmt.Sprintf(
			"recovery service subnet %s returned ambiguous 404 NotAuthorizedOrNotFound: %s",
			operation,
			err.Error(),
		),
		opcRequestID: errorutil.OpcRequestID(err),
	}
}

func (c recoveryServiceSubnetPendingWriteDeleteClient) CreateOrUpdate(
	ctx context.Context,
	resource *recoveryv1beta1.RecoveryServiceSubnet,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c recoveryServiceSubnetPendingWriteDeleteClient) Delete(
	ctx context.Context,
	resource *recoveryv1beta1.RecoveryServiceSubnet,
) (bool, error) {
	if recoveryServiceSubnetHasPendingDeleteWorkRequest(resource) {
		return c.delegate.Delete(ctx, resource)
	}
	if recoveryServiceSubnetHasPendingWrite(resource) {
		response, err := c.delegate.CreateOrUpdate(ctx, resource, ctrl.Request{})
		if err != nil {
			return false, err
		}
		if recoveryServiceSubnetHasPendingWrite(resource) || response.ShouldRequeue {
			return false, nil
		}
	}
	if err := c.guardRecoveryServiceSubnetDeleteRead(ctx, resource); err != nil {
		return false, err
	}
	return c.delegate.Delete(ctx, resource)
}

func (c recoveryServiceSubnetPendingWriteDeleteClient) guardRecoveryServiceSubnetDeleteRead(
	ctx context.Context,
	resource *recoveryv1beta1.RecoveryServiceSubnet,
) error {
	currentID := recoveryServiceSubnetCurrentID(resource)
	if currentID == "" || c.getRecoveryServiceSubnet == nil {
		return nil
	}

	_, err := c.getRecoveryServiceSubnet(ctx, recoverysdk.GetRecoveryServiceSubnetRequest{
		RecoveryServiceSubnetId: common.String(currentID),
	})
	if err == nil || errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound() {
		return nil
	}
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	return err
}

func recoveryServiceSubnetHasPendingWrite(resource *recoveryv1beta1.RecoveryServiceSubnet) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.NormalizedClass != shared.OSOKAsyncClassPending {
		return false
	}
	return current.Phase == shared.OSOKAsyncPhaseCreate || current.Phase == shared.OSOKAsyncPhaseUpdate
}

func recoveryServiceSubnetHasPendingDeleteWorkRequest(resource *recoveryv1beta1.RecoveryServiceSubnet) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Source == shared.OSOKAsyncSourceWorkRequest &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.WorkRequestID != "" &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func recoveryServiceSubnetCurrentID(resource *recoveryv1beta1.RecoveryServiceSubnet) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func getRecoveryServiceSubnetWorkRequest(
	ctx context.Context,
	client recoveryServiceSubnetCompartmentMoveClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize recovery service subnet OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("recovery service subnet OCI client is not configured")
	}
	response, err := client.GetWorkRequest(ctx, recoverysdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveRecoveryServiceSubnetWorkRequestAction(workRequest any) (string, error) {
	recoveryServiceSubnetWorkRequest, err := recoveryServiceSubnetWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveRecoveryServiceSubnetWorkRequestResourceAction(recoveryServiceSubnetWorkRequest)
}

func resolveRecoveryServiceSubnetWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	recoveryServiceSubnetWorkRequest, err := recoveryServiceSubnetWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := recoveryServiceSubnetWorkRequestPhaseFromOperationType(recoveryServiceSubnetWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverRecoveryServiceSubnetIDFromWorkRequest(
	_ *recoveryv1beta1.RecoveryServiceSubnet,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	recoveryServiceSubnetWorkRequest, err := recoveryServiceSubnetWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveRecoveryServiceSubnetIDFromWorkRequest(
		recoveryServiceSubnetWorkRequest,
		recoveryServiceSubnetWorkRequestActionForPhase(phase),
	)
}

func recoveryServiceSubnetWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	recoveryServiceSubnetWorkRequest, err := recoveryServiceSubnetWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf(
		"RecoveryServiceSubnet %s work request %s is %s",
		phase,
		recoveryServiceSubnetStringValue(recoveryServiceSubnetWorkRequest.Id),
		recoveryServiceSubnetWorkRequest.Status,
	)
}

func recoveryServiceSubnetWorkRequestFromAny(workRequest any) (recoverysdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case recoverysdk.WorkRequest:
		return current, nil
	case *recoverysdk.WorkRequest:
		if current == nil {
			return recoverysdk.WorkRequest{}, fmt.Errorf("recovery service subnet work request is nil")
		}
		return *current, nil
	default:
		return recoverysdk.WorkRequest{}, fmt.Errorf("unexpected recovery service subnet work request type %T", workRequest)
	}
}

func recoveryServiceSubnetWorkRequestPhaseFromOperationType(operationType recoverysdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case recoverysdk.OperationTypeCreateRecoveryServiceSubnet:
		return shared.OSOKAsyncPhaseCreate, true
	case recoverysdk.OperationTypeUpdateRecoveryServiceSubnet:
		return shared.OSOKAsyncPhaseUpdate, true
	case recoverysdk.OperationTypeMoveRecoveryServiceSubnet:
		return shared.OSOKAsyncPhaseUpdate, true
	case recoverysdk.OperationTypeDeleteRecoveryServiceSubnet:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func recoveryServiceSubnetWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) recoverysdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return recoverysdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return recoverysdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return recoverysdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveRecoveryServiceSubnetIDFromWorkRequest(
	workRequest recoverysdk.WorkRequest,
	action recoverysdk.ActionTypeEnum,
) (string, error) {
	if id, ok := recoveryServiceSubnetIDFromWorkRequestResources(workRequest.Resources, action); ok {
		return id, nil
	}
	if id, ok := recoveryServiceSubnetIDFromWorkRequestResources(workRequest.Resources, ""); ok {
		return id, nil
	}
	return "", fmt.Errorf("recovery service subnet work request %s does not expose a resource identifier", recoveryServiceSubnetStringValue(workRequest.Id))
}

func recoveryServiceSubnetIDFromWorkRequestResources(
	resources []recoverysdk.WorkRequestResource,
	action recoverysdk.ActionTypeEnum,
) (string, bool) {
	for _, resource := range resources {
		id, ok := recoveryServiceSubnetIDFromWorkRequestResource(resource, action)
		if ok {
			return id, true
		}
	}
	return "", false
}

func recoveryServiceSubnetIDFromWorkRequestResource(
	resource recoverysdk.WorkRequestResource,
	action recoverysdk.ActionTypeEnum,
) (string, bool) {
	if !isRecoveryServiceSubnetWorkRequestResource(resource) {
		return "", false
	}
	if isRecoveryServiceSubnetIgnorableWorkRequestAction(resource.ActionType) {
		return "", false
	}
	if action != "" && resource.ActionType != action {
		return "", false
	}
	id := strings.TrimSpace(recoveryServiceSubnetStringValue(resource.Identifier))
	return id, id != ""
}

func resolveRecoveryServiceSubnetWorkRequestResourceAction(workRequest recoverysdk.WorkRequest) (string, error) {
	var action string
	for _, resource := range workRequest.Resources {
		if !isRecoveryServiceSubnetWorkRequestResource(resource) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" || isRecoveryServiceSubnetIgnorableWorkRequestAction(resource.ActionType) {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf(
				"recovery service subnet work request %s exposes conflicting action types %q and %q",
				recoveryServiceSubnetStringValue(workRequest.Id),
				action,
				candidate,
			)
		}
	}
	return action, nil
}

func isRecoveryServiceSubnetIgnorableWorkRequestAction(action recoverysdk.ActionTypeEnum) bool {
	return action == recoverysdk.ActionTypeInProgress || action == recoverysdk.ActionTypeRelated
}

func isRecoveryServiceSubnetWorkRequestResource(resource recoverysdk.WorkRequestResource) bool {
	return normalizeRecoveryServiceSubnetWorkRequestEntity(recoveryServiceSubnetStringValue(resource.EntityType)) == "recoveryservicesubnet"
}

func normalizeRecoveryServiceSubnetWorkRequestEntity(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func recoveryServiceSubnetFromResponse(response any) (recoverysdk.RecoveryServiceSubnet, bool) {
	if current, ok := recoveryServiceSubnetFromDirectResponse(response); ok {
		return current, true
	}
	if current, ok := recoveryServiceSubnetFromSummaryResponse(response); ok {
		return current, true
	}
	if current, ok := recoveryServiceSubnetFromCreateResponse(response); ok {
		return current, true
	}
	if current, ok := recoveryServiceSubnetFromGetResponse(response); ok {
		return current, true
	}
	return recoveryServiceSubnetFromResponseField(response)
}

func recoveryServiceSubnetFromDirectResponse(response any) (recoverysdk.RecoveryServiceSubnet, bool) {
	switch current := response.(type) {
	case recoverysdk.RecoveryServiceSubnet:
		return current, true
	case *recoverysdk.RecoveryServiceSubnet:
		if current == nil {
			return recoverysdk.RecoveryServiceSubnet{}, false
		}
		return *current, true
	default:
		return recoverysdk.RecoveryServiceSubnet{}, false
	}
}

func recoveryServiceSubnetFromSummaryResponse(response any) (recoverysdk.RecoveryServiceSubnet, bool) {
	switch current := response.(type) {
	case recoverysdk.RecoveryServiceSubnetSummary:
		return recoveryServiceSubnetFromSummary(current), true
	case *recoverysdk.RecoveryServiceSubnetSummary:
		if current == nil {
			return recoverysdk.RecoveryServiceSubnet{}, false
		}
		return recoveryServiceSubnetFromSummary(*current), true
	default:
		return recoverysdk.RecoveryServiceSubnet{}, false
	}
}

func recoveryServiceSubnetFromCreateResponse(response any) (recoverysdk.RecoveryServiceSubnet, bool) {
	switch current := response.(type) {
	case recoverysdk.CreateRecoveryServiceSubnetResponse:
		return current.RecoveryServiceSubnet, true
	case *recoverysdk.CreateRecoveryServiceSubnetResponse:
		if current == nil {
			return recoverysdk.RecoveryServiceSubnet{}, false
		}
		return current.RecoveryServiceSubnet, true
	default:
		return recoverysdk.RecoveryServiceSubnet{}, false
	}
}

func recoveryServiceSubnetFromGetResponse(response any) (recoverysdk.RecoveryServiceSubnet, bool) {
	switch current := response.(type) {
	case recoverysdk.GetRecoveryServiceSubnetResponse:
		return current.RecoveryServiceSubnet, true
	case *recoverysdk.GetRecoveryServiceSubnetResponse:
		if current == nil {
			return recoverysdk.RecoveryServiceSubnet{}, false
		}
		return current.RecoveryServiceSubnet, true
	default:
		return recoverysdk.RecoveryServiceSubnet{}, false
	}
}

func recoveryServiceSubnetFromResponseField(response any) (recoverysdk.RecoveryServiceSubnet, bool) {
	value := reflect.ValueOf(response)
	if !value.IsValid() {
		return recoverysdk.RecoveryServiceSubnet{}, false
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return recoverysdk.RecoveryServiceSubnet{}, false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return recoverysdk.RecoveryServiceSubnet{}, false
	}

	field := value.FieldByName("RecoveryServiceSubnet")
	if !field.IsValid() || !field.CanInterface() {
		return recoverysdk.RecoveryServiceSubnet{}, false
	}
	current, ok := field.Interface().(recoverysdk.RecoveryServiceSubnet)
	return current, ok
}

func recoveryServiceSubnetFromSummary(summary recoverysdk.RecoveryServiceSubnetSummary) recoverysdk.RecoveryServiceSubnet {
	return recoverysdk.RecoveryServiceSubnet{
		Id:               summary.Id,
		CompartmentId:    summary.CompartmentId,
		VcnId:            summary.VcnId,
		SubnetId:         summary.SubnetId,
		DisplayName:      summary.DisplayName,
		Subnets:          cloneRecoveryServiceSubnetStringSlice(summary.Subnets),
		NsgIds:           cloneRecoveryServiceSubnetStringSlice(summary.NsgIds),
		TimeCreated:      summary.TimeCreated,
		TimeUpdated:      summary.TimeUpdated,
		LifecycleState:   summary.LifecycleState,
		LifecycleDetails: summary.LifecycleDetails,
		FreeformTags:     cloneRecoveryServiceSubnetStringMap(summary.FreeformTags),
		DefinedTags:      cloneRecoveryServiceSubnetDefinedTagMap(summary.DefinedTags),
		SystemTags:       cloneRecoveryServiceSubnetDefinedTagMap(summary.SystemTags),
	}
}

func recoveryServiceSubnetStringPtrEqual(current *string, desired string) bool {
	return strings.TrimSpace(recoveryServiceSubnetStringValue(current)) == strings.TrimSpace(desired)
}

func recoveryServiceSubnetStringSliceSetEqual(current []string, desired []string) bool {
	currentCounts := recoveryServiceSubnetStringCounts(current)
	desiredCounts := recoveryServiceSubnetStringCounts(desired)
	if len(currentCounts) != len(desiredCounts) {
		return false
	}
	for value, currentCount := range currentCounts {
		if desiredCounts[value] != currentCount {
			return false
		}
	}
	return true
}

func recoveryServiceSubnetStringCounts(values []string) map[string]int {
	counts := make(map[string]int, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		counts[trimmed]++
	}
	return counts
}

func recoveryServiceSubnetStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func cloneRecoveryServiceSubnetStringSlice(source []string) []string {
	if source == nil {
		return nil
	}
	clone := make([]string, len(source))
	copy(clone, source)
	return clone
}

func cloneRecoveryServiceSubnetStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneRecoveryServiceSubnetDefinedTagMap(source map[string]map[string]interface{}) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	clone := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		inner := make(map[string]interface{}, len(values))
		for key, value := range values {
			inner[key] = value
		}
		clone[namespace] = inner
	}
	return clone
}

func recoveryServiceSubnetDefinedTagsFromSpec(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if source == nil {
		return nil
	}
	return *util.ConvertToOciDefinedTags(&source)
}
