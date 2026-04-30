/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package delegationcontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	delegateaccesscontrolsdk "github.com/oracle/oci-go-sdk/v65/delegateaccesscontrol"
	delegateaccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/delegateaccesscontrol/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

var delegationControlWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(delegateaccesscontrolsdk.OperationStatusAccepted),
		string(delegateaccesscontrolsdk.OperationStatusInProgress),
		string(delegateaccesscontrolsdk.OperationStatusWaiting),
		string(delegateaccesscontrolsdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(delegateaccesscontrolsdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(delegateaccesscontrolsdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(delegateaccesscontrolsdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(delegateaccesscontrolsdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(delegateaccesscontrolsdk.OperationTypeCreateDelegationControl)},
	UpdateActionTokens:    []string{string(delegateaccesscontrolsdk.OperationTypeUpdateDelegationControl)},
	DeleteActionTokens:    []string{string(delegateaccesscontrolsdk.OperationTypeDeleteDelegationControl)},
}

type delegationControlOCIClient interface {
	CreateDelegationControl(context.Context, delegateaccesscontrolsdk.CreateDelegationControlRequest) (delegateaccesscontrolsdk.CreateDelegationControlResponse, error)
	GetDelegationControl(context.Context, delegateaccesscontrolsdk.GetDelegationControlRequest) (delegateaccesscontrolsdk.GetDelegationControlResponse, error)
	ListDelegationControls(context.Context, delegateaccesscontrolsdk.ListDelegationControlsRequest) (delegateaccesscontrolsdk.ListDelegationControlsResponse, error)
	UpdateDelegationControl(context.Context, delegateaccesscontrolsdk.UpdateDelegationControlRequest) (delegateaccesscontrolsdk.UpdateDelegationControlResponse, error)
	DeleteDelegationControl(context.Context, delegateaccesscontrolsdk.DeleteDelegationControlRequest) (delegateaccesscontrolsdk.DeleteDelegationControlResponse, error)
}

type delegationControlWorkRequestClient interface {
	GetWorkRequest(context.Context, delegateaccesscontrolsdk.GetWorkRequestRequest) (delegateaccesscontrolsdk.GetWorkRequestResponse, error)
}

type delegationControlRuntimeClient interface {
	delegationControlOCIClient
	delegationControlWorkRequestClient
}

type delegationControlIdentity struct {
	compartmentID   string
	displayName     string
	resourceType    string
	resourceIDs     []string
	firstResourceID string
	vaultID         string
	vaultKeyID      string
}

func init() {
	registerDelegationControlRuntimeHooksMutator(func(manager *DelegationControlServiceManager, hooks *DelegationControlRuntimeHooks) {
		ociClient, clientErr := newDelegationControlSDKClient(manager)
		workRequestClient, workRequestErr := newDelegationControlWorkRequestClient(manager)
		applyDelegationControlRuntimeHooks(hooks, ociClient, clientErr, workRequestClient, workRequestErr)
	})
}

func newDelegationControlSDKClient(manager *DelegationControlServiceManager) (delegationControlOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("DelegationControl service manager is nil")
	}

	client, err := delegateaccesscontrolsdk.NewDelegateAccessControlClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func newDelegationControlWorkRequestClient(
	manager *DelegationControlServiceManager,
) (delegationControlWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("DelegationControl service manager is nil")
	}

	client, err := delegateaccesscontrolsdk.NewWorkRequestClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyDelegationControlRuntimeHooks(
	hooks *DelegationControlRuntimeHooks,
	ociClient delegationControlOCIClient,
	ociClientErr error,
	workRequestClient delegationControlWorkRequestClient,
	workRequestErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedDelegationControlRuntimeSemantics()
	hooks.BuildCreateBody = func(
		ctx context.Context,
		resource *delegateaccesscontrolv1beta1.DelegationControl,
		namespace string,
	) (any, error) {
		return buildDelegationControlCreateBody(ctx, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *delegateaccesscontrolv1beta1.DelegationControl,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildDelegationControlUpdateBody(resource, currentResponse)
	}
	hooks.Identity.Resolve = func(resource *delegateaccesscontrolv1beta1.DelegationControl) (any, error) {
		return resolveDelegationControlIdentity(resource)
	}
	hooks.Identity.LookupExisting = func(
		ctx context.Context,
		_ *delegateaccesscontrolv1beta1.DelegationControl,
		identity any,
	) (any, error) {
		delegationIdentity, ok := identity.(delegationControlIdentity)
		if !ok {
			return nil, fmt.Errorf("unexpected DelegationControl identity type %T", identity)
		}
		return lookupExistingDelegationControl(ctx, ociClient, ociClientErr, delegationIdentity)
	}
	hooks.Identity.GuardExistingBeforeCreate = guardDelegationControlExistingBeforeCreate
	hooks.ParityHooks.ValidateCreateOnlyDrift = validateDelegationControlCreateOnlyDrift
	hooks.Create.Fields = delegationControlCreateFields()
	hooks.Get.Fields = delegationControlGetFields()
	hooks.List.Fields = delegationControlListFields()
	hooks.Update.Fields = delegationControlUpdateFields()
	hooks.Delete.Fields = delegationControlDeleteFields()
	hooks.Async.Adapter = delegationControlWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getDelegationControlWorkRequest(ctx, workRequestClient, workRequestErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveDelegationControlGeneratedWorkRequestAction
	hooks.Async.ResolvePhase = resolveDelegationControlGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverDelegationControlIDFromGeneratedWorkRequest
	hooks.Async.Message = delegationControlGeneratedWorkRequestMessage
}

func newDelegationControlServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client delegationControlRuntimeClient,
) DelegationControlServiceClient {
	return defaultDelegationControlServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*delegateaccesscontrolv1beta1.DelegationControl](
			newDelegationControlRuntimeConfig(log, client),
		),
	}
}

func newDelegationControlRuntimeConfig(
	log loggerutil.OSOKLogger,
	client delegationControlRuntimeClient,
) generatedruntime.Config[*delegateaccesscontrolv1beta1.DelegationControl] {
	hooks := newDelegationControlRuntimeHooksWithOCIClient(client)
	applyDelegationControlRuntimeHooks(&hooks, client, nil, client, nil)
	return buildDelegationControlGeneratedRuntimeConfig(&DelegationControlServiceManager{Log: log}, hooks)
}

func newDelegationControlRuntimeHooksWithOCIClient(
	client delegationControlRuntimeClient,
) DelegationControlRuntimeHooks {
	return DelegationControlRuntimeHooks{
		Semantics: reviewedDelegationControlRuntimeSemantics(),
		Create: runtimeOperationHooks[delegateaccesscontrolsdk.CreateDelegationControlRequest, delegateaccesscontrolsdk.CreateDelegationControlResponse]{
			Fields: delegationControlCreateFields(),
			Call: func(ctx context.Context, request delegateaccesscontrolsdk.CreateDelegationControlRequest) (delegateaccesscontrolsdk.CreateDelegationControlResponse, error) {
				return client.CreateDelegationControl(ctx, request)
			},
		},
		Get: runtimeOperationHooks[delegateaccesscontrolsdk.GetDelegationControlRequest, delegateaccesscontrolsdk.GetDelegationControlResponse]{
			Fields: delegationControlGetFields(),
			Call: func(ctx context.Context, request delegateaccesscontrolsdk.GetDelegationControlRequest) (delegateaccesscontrolsdk.GetDelegationControlResponse, error) {
				return client.GetDelegationControl(ctx, request)
			},
		},
		List: runtimeOperationHooks[delegateaccesscontrolsdk.ListDelegationControlsRequest, delegateaccesscontrolsdk.ListDelegationControlsResponse]{
			Fields: delegationControlListFields(),
			Call: func(ctx context.Context, request delegateaccesscontrolsdk.ListDelegationControlsRequest) (delegateaccesscontrolsdk.ListDelegationControlsResponse, error) {
				return client.ListDelegationControls(ctx, request)
			},
		},
		Update: runtimeOperationHooks[delegateaccesscontrolsdk.UpdateDelegationControlRequest, delegateaccesscontrolsdk.UpdateDelegationControlResponse]{
			Fields: delegationControlUpdateFields(),
			Call: func(ctx context.Context, request delegateaccesscontrolsdk.UpdateDelegationControlRequest) (delegateaccesscontrolsdk.UpdateDelegationControlResponse, error) {
				return client.UpdateDelegationControl(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[delegateaccesscontrolsdk.DeleteDelegationControlRequest, delegateaccesscontrolsdk.DeleteDelegationControlResponse]{
			Fields: delegationControlDeleteFields(),
			Call: func(ctx context.Context, request delegateaccesscontrolsdk.DeleteDelegationControlRequest) (delegateaccesscontrolsdk.DeleteDelegationControlResponse, error) {
				return client.DeleteDelegationControl(ctx, request)
			},
		},
	}
}

func reviewedDelegationControlRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newDelegationControlRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		// DelegateAccessControl list summaries do not expose resourceIds, so generic no-ID
		// read/delete fallback must stay on summary-compatible fields.
		MatchFields: []string{"compartmentId", "displayName", "resourceType"},
	}
	semantics.Hooks = generatedruntime.HookSet{
		Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
	}
	semantics.CreateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetDelegationControl",
		Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
	}
	semantics.UpdateFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetDelegationControl",
		Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
	}
	semantics.DeleteFollowUp = generatedruntime.FollowUpSemantics{
		Strategy: "GetWorkRequest -> GetDelegationControl/ListDelegationControls confirm-delete",
		Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func delegationControlCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateDelegationControlDetails", RequestName: "CreateDelegationControlDetails", Contribution: "body"},
	}
}

func delegationControlGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DelegationControlId", RequestName: "delegationControlId", Contribution: "path", PreferResourceID: true},
	}
}

func delegationControlListFields() []generatedruntime.RequestField {
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
			FieldName:    "ResourceType",
			RequestName:  "resourceType",
			Contribution: "query",
			LookupPaths:  []string{"status.resourceType", "spec.resourceType", "resourceType"},
		},
	}
}

func delegationControlUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DelegationControlId", RequestName: "delegationControlId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateDelegationControlDetails", RequestName: "UpdateDelegationControlDetails", Contribution: "body"},
	}
}

func delegationControlDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DelegationControlId", RequestName: "delegationControlId", Contribution: "path", PreferResourceID: true},
	}
}

func resolveDelegationControlIdentity(
	resource *delegateaccesscontrolv1beta1.DelegationControl,
) (delegationControlIdentity, error) {
	if resource == nil {
		return delegationControlIdentity{}, fmt.Errorf("DelegationControl resource is nil")
	}

	resourceIDs := canonicalStringSlice(resource.Spec.ResourceIds)
	if len(resourceIDs) == 0 {
		resourceIDs = canonicalStringSlice(resource.Status.ResourceIds)
	}

	identity := delegationControlIdentity{
		compartmentID: firstNonEmptyTrim(resource.Spec.CompartmentId, resource.Status.CompartmentId),
		displayName:   firstNonEmptyTrim(resource.Spec.DisplayName, resource.Status.DisplayName),
		resourceType:  firstNonEmptyTrim(resource.Spec.ResourceType, resource.Status.ResourceType),
		resourceIDs:   resourceIDs,
		vaultID:       firstNonEmptyTrim(resource.Spec.VaultId, resource.Status.VaultId),
		vaultKeyID:    firstNonEmptyTrim(resource.Spec.VaultKeyId, resource.Status.VaultKeyId),
	}
	if len(identity.resourceIDs) != 0 {
		identity.firstResourceID = identity.resourceIDs[0]
	}
	return identity, nil
}

func guardDelegationControlExistingBeforeCreate(
	_ context.Context,
	resource *delegateaccesscontrolv1beta1.DelegationControl,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("DelegationControl resource is nil")
	}

	if strings.TrimSpace(resource.Spec.CompartmentId) == "" ||
		strings.TrimSpace(resource.Spec.DisplayName) == "" ||
		strings.TrimSpace(resource.Spec.ResourceType) == "" ||
		len(canonicalStringSlice(resource.Spec.ResourceIds)) == 0 {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}

	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func lookupExistingDelegationControl(
	ctx context.Context,
	client delegationControlOCIClient,
	initErr error,
	identity delegationControlIdentity,
) (any, error) {
	if initErr != nil {
		return nil, initErr
	}
	if client == nil {
		return nil, fmt.Errorf("DelegationControl OCI client is nil")
	}

	request := delegateaccesscontrolsdk.ListDelegationControlsRequest{
		CompartmentId: common.String(identity.compartmentID),
		DisplayName:   common.String(identity.displayName),
		ResourceType:  delegateaccesscontrolsdk.ListDelegationControlsResourceTypeEnum(identity.resourceType),
	}
	if identity.firstResourceID != "" {
		request.ResourceId = common.String(identity.firstResourceID)
	}

	response, err := client.ListDelegationControls(ctx, request)
	if err != nil {
		return nil, err
	}

	matches := make([]delegateaccesscontrolsdk.GetDelegationControlResponse, 0, len(response.Items))
	for _, item := range response.Items {
		if !delegationControlSummaryMatchesIdentity(item, identity) {
			continue
		}

		resourceID := stringValue(item.Id)
		if resourceID == "" {
			continue
		}

		candidate, err := client.GetDelegationControl(ctx, delegateaccesscontrolsdk.GetDelegationControlRequest{
			DelegationControlId: common.String(resourceID),
		})
		if err != nil {
			return nil, err
		}
		if !delegationControlMatchesIdentity(candidate.DelegationControl, identity) {
			continue
		}
		if !delegationControlLifecycleReusable(candidate.DelegationControl.LifecycleState) {
			continue
		}

		matches = append(matches, candidate)
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf(
			"DelegationControl lookup returned multiple exact matches for compartmentId=%q displayName=%q resourceType=%q resourceIds=%v",
			identity.compartmentID,
			identity.displayName,
			identity.resourceType,
			identity.resourceIDs,
		)
	}
}

func delegationControlSummaryMatchesIdentity(
	current delegateaccesscontrolsdk.DelegationControlSummary,
	identity delegationControlIdentity,
) bool {
	if !delegationControlLifecycleReusable(current.LifecycleState) {
		return false
	}

	return stringValue(current.CompartmentId) == identity.compartmentID &&
		stringValue(current.DisplayName) == identity.displayName &&
		string(current.ResourceType) == identity.resourceType
}

func delegationControlMatchesIdentity(
	current delegateaccesscontrolsdk.DelegationControl,
	identity delegationControlIdentity,
) bool {
	return stringValue(current.CompartmentId) == identity.compartmentID &&
		stringValue(current.DisplayName) == identity.displayName &&
		string(current.ResourceType) == identity.resourceType &&
		slices.Equal(canonicalStringSlice(current.ResourceIds), identity.resourceIDs) &&
		stringValue(current.VaultId) == identity.vaultID &&
		stringValue(current.VaultKeyId) == identity.vaultKeyID
}

func delegationControlLifecycleReusable(state delegateaccesscontrolsdk.DelegationControlLifecycleStateEnum) bool {
	switch state {
	case delegateaccesscontrolsdk.DelegationControlLifecycleStateActive,
		delegateaccesscontrolsdk.DelegationControlLifecycleStateCreating,
		delegateaccesscontrolsdk.DelegationControlLifecycleStateUpdating:
		return true
	default:
		return false
	}
}

func buildDelegationControlCreateBody(
	ctx context.Context,
	resource *delegateaccesscontrolv1beta1.DelegationControl,
	namespace string,
) (delegateaccesscontrolsdk.CreateDelegationControlDetails, error) {
	if resource == nil {
		return delegateaccesscontrolsdk.CreateDelegationControlDetails{}, fmt.Errorf("DelegationControl resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, nil, namespace)
	if err != nil {
		return delegateaccesscontrolsdk.CreateDelegationControlDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return delegateaccesscontrolsdk.CreateDelegationControlDetails{}, fmt.Errorf("marshal resolved DelegationControl spec: %w", err)
	}

	var details delegateaccesscontrolsdk.CreateDelegationControlDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return delegateaccesscontrolsdk.CreateDelegationControlDetails{}, fmt.Errorf("decode DelegationControl create request body: %w", err)
	}
	if err := validateDelegationControlVaultConstraint(resource.Spec.ResourceType, resource.Spec.VaultId, resource.Spec.VaultKeyId); err != nil {
		return delegateaccesscontrolsdk.CreateDelegationControlDetails{}, err
	}

	return details, nil
}

func buildDelegationControlUpdateBody(
	resource *delegateaccesscontrolv1beta1.DelegationControl,
	currentResponse any,
) (delegateaccesscontrolsdk.UpdateDelegationControlDetails, bool, error) {
	if resource == nil {
		return delegateaccesscontrolsdk.UpdateDelegationControlDetails{}, false, fmt.Errorf("DelegationControl resource is nil")
	}
	if err := validateDelegationControlVaultConstraint(resource.Spec.ResourceType, resource.Spec.VaultId, resource.Spec.VaultKeyId); err != nil {
		return delegateaccesscontrolsdk.UpdateDelegationControlDetails{}, false, err
	}

	current, err := delegationControlFromResponse(currentResponse)
	if err != nil {
		return delegateaccesscontrolsdk.UpdateDelegationControlDetails{}, false, err
	}

	details := delegateaccesscontrolsdk.UpdateDelegationControlDetails{}
	updateNeeded := false

	if desired, ok := delegationControlDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := delegationControlDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := delegationControlDesiredIntUpdate(resource.Spec.NumApprovalsRequired, current.NumApprovalsRequired); ok {
		details.NumApprovalsRequired = desired
		updateNeeded = true
	}
	if desired, ok := delegationControlDesiredStringSliceUpdate(resource.Spec.DelegationSubscriptionIds, current.DelegationSubscriptionIds); ok {
		details.DelegationSubscriptionIds = desired
		updateNeeded = true
	}
	if desired, ok := delegationControlDesiredBoolUpdate(resource.Spec.IsAutoApproveDuringMaintenance, current.IsAutoApproveDuringMaintenance); ok {
		details.IsAutoApproveDuringMaintenance = desired
		updateNeeded = true
	}
	if desired, ok := delegationControlDesiredStringSliceUpdate(resource.Spec.ResourceIds, current.ResourceIds); ok {
		details.ResourceIds = desired
		updateNeeded = true
	}
	if desired, ok := delegationControlDesiredStringSliceUpdate(resource.Spec.PreApprovedServiceProviderActionNames, current.PreApprovedServiceProviderActionNames); ok {
		details.PreApprovedServiceProviderActionNames = desired
		updateNeeded = true
	}
	if desired, ok := delegationControlDesiredStringUpdate(resource.Spec.NotificationTopicId, current.NotificationTopicId); ok {
		details.NotificationTopicId = desired
		updateNeeded = true
	}
	if desired, ok := delegationControlDesiredNotificationMessageFormatUpdate(resource.Spec.NotificationMessageFormat, current.NotificationMessageFormat); ok {
		details.NotificationMessageFormat = desired
		updateNeeded = true
	}
	if desired, ok := delegationControlDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := delegationControlDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func validateDelegationControlCreateOnlyDrift(
	resource *delegateaccesscontrolv1beta1.DelegationControl,
	currentResponse any,
) error {
	if resource == nil || currentResponse == nil {
		return nil
	}

	current, err := delegationControlFromResponse(currentResponse)
	if err != nil {
		return err
	}
	if delegationControlForceNewStringChanged(resource.Spec.VaultId, current.VaultId) {
		return fmt.Errorf("DelegationControl formal semantics require replacement when vaultId changes")
	}
	if delegationControlForceNewStringChanged(resource.Spec.VaultKeyId, current.VaultKeyId) {
		return fmt.Errorf("DelegationControl formal semantics require replacement when vaultKeyId changes")
	}
	return nil
}

func validateDelegationControlVaultConstraint(resourceType string, vaultID string, vaultKeyID string) error {
	resourceType = strings.TrimSpace(resourceType)
	vaultID = strings.TrimSpace(vaultID)
	vaultKeyID = strings.TrimSpace(vaultKeyID)

	if resourceType == string(delegateaccesscontrolsdk.DelegationControlResourceTypeCloudvmcluster) {
		if vaultID == "" || vaultKeyID == "" {
			return fmt.Errorf("DelegationControl resourceType CLOUDVMCLUSTER requires both vaultId and vaultKeyId")
		}
		return nil
	}

	if vaultID != "" || vaultKeyID != "" {
		return fmt.Errorf("DelegationControl vaultId and vaultKeyId are only supported when resourceType is CLOUDVMCLUSTER")
	}
	return nil
}

func delegationControlFromResponse(currentResponse any) (delegateaccesscontrolsdk.DelegationControl, error) {
	switch current := currentResponse.(type) {
	case delegateaccesscontrolsdk.DelegationControl:
		return current, nil
	case *delegateaccesscontrolsdk.DelegationControl:
		if current == nil {
			return delegateaccesscontrolsdk.DelegationControl{}, fmt.Errorf("current DelegationControl response is nil")
		}
		return *current, nil
	case delegateaccesscontrolsdk.DelegationControlSummary:
		return delegateaccesscontrolsdk.DelegationControl{
			Id:                    current.Id,
			DisplayName:           current.DisplayName,
			CompartmentId:         current.CompartmentId,
			ResourceType:          current.ResourceType,
			TimeCreated:           current.TimeCreated,
			TimeUpdated:           current.TimeUpdated,
			TimeDeleted:           current.TimeDeleted,
			LifecycleState:        current.LifecycleState,
			LifecycleStateDetails: current.LifecycleStateDetails,
			FreeformTags:          current.FreeformTags,
			DefinedTags:           current.DefinedTags,
			SystemTags:            current.SystemTags,
		}, nil
	case *delegateaccesscontrolsdk.DelegationControlSummary:
		if current == nil {
			return delegateaccesscontrolsdk.DelegationControl{}, fmt.Errorf("current DelegationControl response is nil")
		}
		return delegationControlFromResponse(*current)
	case delegateaccesscontrolsdk.CreateDelegationControlResponse:
		return current.DelegationControl, nil
	case *delegateaccesscontrolsdk.CreateDelegationControlResponse:
		if current == nil {
			return delegateaccesscontrolsdk.DelegationControl{}, fmt.Errorf("current DelegationControl response is nil")
		}
		return current.DelegationControl, nil
	case delegateaccesscontrolsdk.GetDelegationControlResponse:
		return current.DelegationControl, nil
	case *delegateaccesscontrolsdk.GetDelegationControlResponse:
		if current == nil {
			return delegateaccesscontrolsdk.DelegationControl{}, fmt.Errorf("current DelegationControl response is nil")
		}
		return current.DelegationControl, nil
	case delegateaccesscontrolsdk.UpdateDelegationControlResponse:
		return current.DelegationControl, nil
	case *delegateaccesscontrolsdk.UpdateDelegationControlResponse:
		if current == nil {
			return delegateaccesscontrolsdk.DelegationControl{}, fmt.Errorf("current DelegationControl response is nil")
		}
		return current.DelegationControl, nil
	default:
		return delegateaccesscontrolsdk.DelegationControl{}, fmt.Errorf("unexpected current DelegationControl response type %T", currentResponse)
	}
}

func delegationControlDesiredStringUpdate(spec string, current *string) (*string, bool) {
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

func delegationControlDesiredIntUpdate(spec int, current *int) (*int, bool) {
	if spec == 0 {
		return nil, false
	}
	if current != nil && *current == spec {
		return nil, false
	}
	return common.Int(spec), true
}

func delegationControlDesiredBoolUpdate(spec bool, current *bool) (*bool, bool) {
	currentValue := false
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	return common.Bool(spec), true
}

func delegationControlDesiredStringSliceUpdate(spec []string, current []string) ([]string, bool) {
	if spec == nil {
		return nil, false
	}

	desired := nonEmptyTrimmedStringSlice(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if stringSlicesEqualCanonical(desired, current) {
		return nil, false
	}
	return desired, true
}

func delegationControlDesiredNotificationMessageFormatUpdate(
	spec string,
	current delegateaccesscontrolsdk.DelegationControlNotificationMessageFormatEnum,
) (delegateaccesscontrolsdk.DelegationControlNotificationMessageFormatEnum, bool) {
	if spec == "" || spec == string(current) {
		return "", false
	}
	return delegateaccesscontrolsdk.DelegationControlNotificationMessageFormatEnum(spec), true
}

func delegationControlDesiredFreeformTagsUpdate(
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

func delegationControlDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := delegationControlDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if delegationControlJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func delegationControlDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}

	desired := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		desired[namespace] = converted
	}
	return desired
}

func delegationControlJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftPayload) == string(rightPayload)
}

func getDelegationControlWorkRequest(
	ctx context.Context,
	client delegationControlWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize DelegationControl OCI work request client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("DelegationControl OCI work request client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, delegateaccesscontrolsdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveDelegationControlGeneratedWorkRequestAction(workRequest any) (string, error) {
	current, err := delegationControlWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return string(current.OperationType), nil
}

func resolveDelegationControlGeneratedWorkRequestPhase(
	workRequest any,
) (shared.OSOKAsyncPhase, bool, error) {
	current, err := delegationControlWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := delegationControlWorkRequestPhaseFromOperationType(current.OperationType)
	return phase, ok, nil
}

func recoverDelegationControlIDFromGeneratedWorkRequest(
	_ *delegateaccesscontrolv1beta1.DelegationControl,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, err := delegationControlWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}

	action := delegationControlWorkRequestActionForPhase(phase)
	if id, ok := resolveDelegationControlIDFromResources(current.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveDelegationControlIDFromResources(current.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("DelegationControl work request %s does not expose a delegation control identifier", stringValue(current.Id))
}

func delegationControlGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, err := delegationControlWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("DelegationControl %s work request %s is %s", phase, stringValue(current.Id), current.Status)
}

func delegationControlWorkRequestFromAny(workRequest any) (delegateaccesscontrolsdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case delegateaccesscontrolsdk.WorkRequest:
		return current, nil
	case *delegateaccesscontrolsdk.WorkRequest:
		if current == nil {
			return delegateaccesscontrolsdk.WorkRequest{}, fmt.Errorf("DelegationControl work request is nil")
		}
		return *current, nil
	default:
		return delegateaccesscontrolsdk.WorkRequest{}, fmt.Errorf("unexpected DelegationControl work request type %T", workRequest)
	}
}

func delegationControlWorkRequestPhaseFromOperationType(
	operationType delegateaccesscontrolsdk.OperationTypeEnum,
) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case delegateaccesscontrolsdk.OperationTypeCreateDelegationControl:
		return shared.OSOKAsyncPhaseCreate, true
	case delegateaccesscontrolsdk.OperationTypeUpdateDelegationControl:
		return shared.OSOKAsyncPhaseUpdate, true
	case delegateaccesscontrolsdk.OperationTypeDeleteDelegationControl:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func delegationControlWorkRequestActionForPhase(
	phase shared.OSOKAsyncPhase,
) delegateaccesscontrolsdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return delegateaccesscontrolsdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return delegateaccesscontrolsdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return delegateaccesscontrolsdk.ActionTypeDeleted
	default:
		return ""
	}
}

func resolveDelegationControlIDFromResources(
	resources []delegateaccesscontrolsdk.WorkRequestResource,
	action delegateaccesscontrolsdk.ActionTypeEnum,
	preferDelegationControlOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferDelegationControlOnly && !isDelegationControlWorkRequestResource(resource) {
			continue
		}

		id := strings.TrimSpace(stringValue(resource.Identifier))
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

func isDelegationControlWorkRequestResource(resource delegateaccesscontrolsdk.WorkRequestResource) bool {
	return normalizeDelegationControlToken(stringValue(resource.EntityType)) == "delegationcontrol"
}

func normalizeDelegationControlToken(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(value))
}

func delegationControlForceNewStringChanged(spec string, current *string) bool {
	spec = strings.TrimSpace(spec)
	currentValue := stringValue(current)
	if spec == "" && currentValue == "" {
		return false
	}
	return spec != currentValue
}

func canonicalStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := nonEmptyTrimmedStringSlice(values)
	if len(normalized) == 0 {
		return nil
	}
	slices.Sort(normalized)
	return normalized
}

func nonEmptyTrimmedStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func stringSlicesEqualCanonical(left []string, right []string) bool {
	return slices.Equal(canonicalStringSlice(left), canonicalStringSlice(right))
}

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
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
