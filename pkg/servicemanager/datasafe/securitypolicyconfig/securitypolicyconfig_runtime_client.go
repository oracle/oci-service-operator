/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package securitypolicyconfig

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
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

const securityPolicyConfigKind = "SecurityPolicyConfig"

var securityPolicyConfigWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(datasafesdk.WorkRequestStatusAccepted),
		string(datasafesdk.WorkRequestStatusInProgress),
		string(datasafesdk.WorkRequestStatusCanceling),
		string(datasafesdk.WorkRequestStatusSuspending),
	},
	SucceededStatusTokens: []string{string(datasafesdk.WorkRequestStatusSucceeded)},
	FailedStatusTokens:    []string{string(datasafesdk.WorkRequestStatusFailed)},
	CanceledStatusTokens:  []string{string(datasafesdk.WorkRequestStatusCanceled)},
	AttentionStatusTokens: []string{string(datasafesdk.WorkRequestStatusSuspended)},
	CreateActionTokens: []string{
		string(datasafesdk.WorkRequestOperationTypeCreateSecurityPolicyConfig),
		string(datasafesdk.WorkRequestResourceActionTypeCreated),
	},
	UpdateActionTokens: []string{
		string(datasafesdk.WorkRequestOperationTypeUpdateSecurityPolicyConfig),
		string(datasafesdk.WorkRequestOperationTypeChangeSecurityPolicyConfigCompartment),
		string(datasafesdk.WorkRequestResourceActionTypeUpdated),
	},
	DeleteActionTokens: []string{
		string(datasafesdk.WorkRequestOperationTypeDeleteSecurityPolicyConfig),
		string(datasafesdk.WorkRequestResourceActionTypeDeleted),
	},
}

type securityPolicyConfigOCIClient interface {
	CreateSecurityPolicyConfig(context.Context, datasafesdk.CreateSecurityPolicyConfigRequest) (datasafesdk.CreateSecurityPolicyConfigResponse, error)
	GetSecurityPolicyConfig(context.Context, datasafesdk.GetSecurityPolicyConfigRequest) (datasafesdk.GetSecurityPolicyConfigResponse, error)
	ListSecurityPolicyConfigs(context.Context, datasafesdk.ListSecurityPolicyConfigsRequest) (datasafesdk.ListSecurityPolicyConfigsResponse, error)
	UpdateSecurityPolicyConfig(context.Context, datasafesdk.UpdateSecurityPolicyConfigRequest) (datasafesdk.UpdateSecurityPolicyConfigResponse, error)
	DeleteSecurityPolicyConfig(context.Context, datasafesdk.DeleteSecurityPolicyConfigRequest) (datasafesdk.DeleteSecurityPolicyConfigResponse, error)
	ChangeSecurityPolicyConfigCompartment(context.Context, datasafesdk.ChangeSecurityPolicyConfigCompartmentRequest) (datasafesdk.ChangeSecurityPolicyConfigCompartmentResponse, error)
	GetWorkRequest(context.Context, datasafesdk.GetWorkRequestRequest) (datasafesdk.GetWorkRequestResponse, error)
}

type securityPolicyConfigRuntimeClient struct {
	delegate SecurityPolicyConfigServiceClient
	client   securityPolicyConfigOCIClient
	initErr  error
	log      loggerutil.OSOKLogger
}

type securityPolicyConfigAuthShapedNotFound struct {
	err error
}

func (e securityPolicyConfigAuthShapedNotFound) Error() string {
	return fmt.Sprintf("SecurityPolicyConfig delete path returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", e.err)
}

func (e securityPolicyConfigAuthShapedNotFound) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

type securityPolicyConfigAuthShapedConfirmRead struct {
	err error
}

func (e securityPolicyConfigAuthShapedConfirmRead) Error() string {
	return fmt.Sprintf("SecurityPolicyConfig delete confirmation returned ambiguous 404 NotAuthorizedOrNotFound; refusing to treat it as deleted: %v", e.err)
}

func (e securityPolicyConfigAuthShapedConfirmRead) GetOpcRequestID() string {
	return errorutil.OpcRequestID(e.err)
}

var _ SecurityPolicyConfigServiceClient = (*securityPolicyConfigRuntimeClient)(nil)

func init() {
	registerSecurityPolicyConfigRuntimeHooksMutator(func(manager *SecurityPolicyConfigServiceManager, hooks *SecurityPolicyConfigRuntimeHooks) {
		client, initErr := newSecurityPolicyConfigOCIClient(manager)
		applySecurityPolicyConfigRuntimeHooks(manager, hooks, client, initErr)
	})
}

func newSecurityPolicyConfigOCIClient(manager *SecurityPolicyConfigServiceManager) (securityPolicyConfigOCIClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("%s service manager is nil", securityPolicyConfigKind)
	}
	client, err := datasafesdk.NewDataSafeClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applySecurityPolicyConfigRuntimeHooks(
	manager *SecurityPolicyConfigServiceManager,
	hooks *SecurityPolicyConfigRuntimeHooks,
	client securityPolicyConfigOCIClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.Semantics = securityPolicyConfigRuntimeSemantics()
	hooks.BuildCreateBody = buildSecurityPolicyConfigCreateBody
	hooks.BuildUpdateBody = buildSecurityPolicyConfigUpdateBody
	applySecurityPolicyConfigOperationHooks(hooks, client, initErr)
	hooks.List.Fields = securityPolicyConfigListFields()
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedSecurityPolicyConfigIdentity
	hooks.ParityHooks.RequiresParityHandling = securityPolicyConfigRequiresCompartmentMove
	hooks.ParityHooks.ApplyParityUpdate = func(
		ctx context.Context,
		resource *datasafev1beta1.SecurityPolicyConfig,
		currentResponse any,
	) (servicemanager.OSOKResponse, error) {
		return applySecurityPolicyConfigCompartmentMove(ctx, resource, currentResponse, client, initErr)
	}
	hooks.StatusHooks.ProjectStatus = projectSecurityPolicyConfigStatus
	hooks.Async.Adapter = securityPolicyConfigWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getSecurityPolicyConfigWorkRequest(ctx, client, initErr, workRequestID)
	}
	hooks.Async.ResolveAction = resolveSecurityPolicyConfigWorkRequestAction
	hooks.Async.RecoverResourceID = recoverSecurityPolicyConfigIDFromWorkRequest
	hooks.Async.Message = securityPolicyConfigWorkRequestMessage
	hooks.DeleteHooks.ConfirmRead = func(ctx context.Context, _ *datasafev1beta1.SecurityPolicyConfig, currentID string) (any, error) {
		return confirmSecurityPolicyConfigDeleteRead(ctx, hooks.Get.Call, currentID)
	}
	hooks.DeleteHooks.HandleError = handleSecurityPolicyConfigDeleteError
	hooks.DeleteHooks.ApplyOutcome = handleSecurityPolicyConfigDeleteConfirmReadOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate SecurityPolicyConfigServiceClient) SecurityPolicyConfigServiceClient {
		runtimeClient := &securityPolicyConfigRuntimeClient{
			delegate: delegate,
			client:   client,
			initErr:  initErr,
		}
		if manager != nil {
			runtimeClient.log = manager.Log
		}
		return runtimeClient
	})
}

func applySecurityPolicyConfigOperationHooks(
	hooks *SecurityPolicyConfigRuntimeHooks,
	client securityPolicyConfigOCIClient,
	initErr error,
) {
	hooks.Create.Fields = securityPolicyConfigCreateFields()
	hooks.Create.Call = func(ctx context.Context, request datasafesdk.CreateSecurityPolicyConfigRequest) (datasafesdk.CreateSecurityPolicyConfigResponse, error) {
		if err := requireSecurityPolicyConfigOCIClient(client, initErr); err != nil {
			return datasafesdk.CreateSecurityPolicyConfigResponse{}, err
		}
		return client.CreateSecurityPolicyConfig(ctx, request)
	}
	hooks.Get.Fields = securityPolicyConfigGetFields()
	hooks.Get.Call = func(ctx context.Context, request datasafesdk.GetSecurityPolicyConfigRequest) (datasafesdk.GetSecurityPolicyConfigResponse, error) {
		if err := requireSecurityPolicyConfigOCIClient(client, initErr); err != nil {
			return datasafesdk.GetSecurityPolicyConfigResponse{}, err
		}
		return client.GetSecurityPolicyConfig(ctx, request)
	}
	hooks.List.Fields = securityPolicyConfigListFields()
	hooks.List.Call = func(ctx context.Context, request datasafesdk.ListSecurityPolicyConfigsRequest) (datasafesdk.ListSecurityPolicyConfigsResponse, error) {
		return listSecurityPolicyConfigsAllPages(ctx, client, initErr, request)
	}
	hooks.Update.Fields = securityPolicyConfigUpdateFields()
	hooks.Update.Call = func(ctx context.Context, request datasafesdk.UpdateSecurityPolicyConfigRequest) (datasafesdk.UpdateSecurityPolicyConfigResponse, error) {
		if err := requireSecurityPolicyConfigOCIClient(client, initErr); err != nil {
			return datasafesdk.UpdateSecurityPolicyConfigResponse{}, err
		}
		return client.UpdateSecurityPolicyConfig(ctx, request)
	}
	hooks.Delete.Fields = securityPolicyConfigDeleteFields()
	hooks.Delete.Call = func(ctx context.Context, request datasafesdk.DeleteSecurityPolicyConfigRequest) (datasafesdk.DeleteSecurityPolicyConfigResponse, error) {
		if err := requireSecurityPolicyConfigOCIClient(client, initErr); err != nil {
			return datasafesdk.DeleteSecurityPolicyConfigResponse{}, err
		}
		return client.DeleteSecurityPolicyConfig(ctx, request)
	}
}

func requireSecurityPolicyConfigOCIClient(client securityPolicyConfigOCIClient, initErr error) error {
	if initErr != nil {
		return fmt.Errorf("initialize %s OCI client: %w", securityPolicyConfigKind, initErr)
	}
	if client == nil {
		return fmt.Errorf("%s OCI client is not configured", securityPolicyConfigKind)
	}
	return nil
}

func securityPolicyConfigRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "datasafe",
		FormalSlug:        "securitypolicyconfig",
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
			ProvisioningStates: []string{string(datasafesdk.SecurityPolicyConfigLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.SecurityPolicyConfigLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.SecurityPolicyConfigLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.SecurityPolicyConfigLifecycleStateDeleting)},
			TerminalStates: []string{string(datasafesdk.SecurityPolicyConfigLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields: []string{
				"compartmentId",
				"securityPolicyId",
				"id",
			},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"compartmentId",
				"displayName",
				"description",
				"firewallConfig",
				"unifiedAuditPolicyConfig",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"securityPolicyId",
			},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func newSecurityPolicyConfigServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client securityPolicyConfigOCIClient,
) SecurityPolicyConfigServiceClient {
	manager := &SecurityPolicyConfigServiceManager{Log: log}
	hooks := newSecurityPolicyConfigRuntimeHooksWithOCIClient(client)
	applySecurityPolicyConfigRuntimeHooks(manager, &hooks, client, nil)
	delegate := defaultSecurityPolicyConfigServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*datasafev1beta1.SecurityPolicyConfig](
			buildSecurityPolicyConfigGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapSecurityPolicyConfigGeneratedClient(hooks, delegate)
}

func newSecurityPolicyConfigRuntimeHooksWithOCIClient(
	client securityPolicyConfigOCIClient,
) SecurityPolicyConfigRuntimeHooks {
	return SecurityPolicyConfigRuntimeHooks{
		Identity:        generatedruntime.IdentityHooks[*datasafev1beta1.SecurityPolicyConfig]{},
		Read:            generatedruntime.ReadHooks{},
		TrackedRecreate: generatedruntime.TrackedRecreateHooks[*datasafev1beta1.SecurityPolicyConfig]{},
		StatusHooks:     generatedruntime.StatusHooks[*datasafev1beta1.SecurityPolicyConfig]{},
		ParityHooks:     generatedruntime.ParityHooks[*datasafev1beta1.SecurityPolicyConfig]{},
		Async:           generatedruntime.AsyncHooks[*datasafev1beta1.SecurityPolicyConfig]{},
		DeleteHooks:     generatedruntime.DeleteHooks[*datasafev1beta1.SecurityPolicyConfig]{},
		Create: runtimeOperationHooks[datasafesdk.CreateSecurityPolicyConfigRequest, datasafesdk.CreateSecurityPolicyConfigResponse]{
			Fields: securityPolicyConfigCreateFields(),
			Call: func(ctx context.Context, request datasafesdk.CreateSecurityPolicyConfigRequest) (datasafesdk.CreateSecurityPolicyConfigResponse, error) {
				if client == nil {
					return datasafesdk.CreateSecurityPolicyConfigResponse{}, fmt.Errorf("%s OCI client is nil", securityPolicyConfigKind)
				}
				return client.CreateSecurityPolicyConfig(ctx, request)
			},
		},
		Get: runtimeOperationHooks[datasafesdk.GetSecurityPolicyConfigRequest, datasafesdk.GetSecurityPolicyConfigResponse]{
			Fields: securityPolicyConfigGetFields(),
			Call: func(ctx context.Context, request datasafesdk.GetSecurityPolicyConfigRequest) (datasafesdk.GetSecurityPolicyConfigResponse, error) {
				if client == nil {
					return datasafesdk.GetSecurityPolicyConfigResponse{}, fmt.Errorf("%s OCI client is nil", securityPolicyConfigKind)
				}
				return client.GetSecurityPolicyConfig(ctx, request)
			},
		},
		List: runtimeOperationHooks[datasafesdk.ListSecurityPolicyConfigsRequest, datasafesdk.ListSecurityPolicyConfigsResponse]{
			Fields: securityPolicyConfigListFields(),
			Call: func(ctx context.Context, request datasafesdk.ListSecurityPolicyConfigsRequest) (datasafesdk.ListSecurityPolicyConfigsResponse, error) {
				return listSecurityPolicyConfigsAllPages(ctx, client, nil, request)
			},
		},
		Update: runtimeOperationHooks[datasafesdk.UpdateSecurityPolicyConfigRequest, datasafesdk.UpdateSecurityPolicyConfigResponse]{
			Fields: securityPolicyConfigUpdateFields(),
			Call: func(ctx context.Context, request datasafesdk.UpdateSecurityPolicyConfigRequest) (datasafesdk.UpdateSecurityPolicyConfigResponse, error) {
				if client == nil {
					return datasafesdk.UpdateSecurityPolicyConfigResponse{}, fmt.Errorf("%s OCI client is nil", securityPolicyConfigKind)
				}
				return client.UpdateSecurityPolicyConfig(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[datasafesdk.DeleteSecurityPolicyConfigRequest, datasafesdk.DeleteSecurityPolicyConfigResponse]{
			Fields: securityPolicyConfigDeleteFields(),
			Call: func(ctx context.Context, request datasafesdk.DeleteSecurityPolicyConfigRequest) (datasafesdk.DeleteSecurityPolicyConfigResponse, error) {
				if client == nil {
					return datasafesdk.DeleteSecurityPolicyConfigResponse{}, fmt.Errorf("%s OCI client is nil", securityPolicyConfigKind)
				}
				return client.DeleteSecurityPolicyConfig(ctx, request)
			},
		},
		WrapGeneratedClient: []func(SecurityPolicyConfigServiceClient) SecurityPolicyConfigServiceClient{},
	}
}

func securityPolicyConfigCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateSecurityPolicyConfigDetails", RequestName: "CreateSecurityPolicyConfigDetails", Contribution: "body"},
	}
}

func securityPolicyConfigGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SecurityPolicyConfigId", RequestName: "securityPolicyConfigId", Contribution: "path", PreferResourceID: true},
	}
}

func securityPolicyConfigListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", LookupPaths: []string{"compartmentId"}},
		{FieldName: "SecurityPolicyConfigId", RequestName: "securityPolicyConfigId", Contribution: "query", PreferResourceID: true},
		{FieldName: "SecurityPolicyId", RequestName: "securityPolicyId", Contribution: "query", LookupPaths: []string{"securityPolicyId"}},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
	}
}

func securityPolicyConfigUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SecurityPolicyConfigId", RequestName: "securityPolicyConfigId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateSecurityPolicyConfigDetails", RequestName: "UpdateSecurityPolicyConfigDetails", Contribution: "body"},
	}
}

func securityPolicyConfigDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "SecurityPolicyConfigId", RequestName: "securityPolicyConfigId", Contribution: "path", PreferResourceID: true},
	}
}

func buildSecurityPolicyConfigCreateBody(
	_ context.Context,
	resource *datasafev1beta1.SecurityPolicyConfig,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("%s resource is nil", securityPolicyConfigKind)
	}
	if err := validateSecurityPolicyConfigSpec(resource.Spec); err != nil {
		return nil, err
	}

	spec := resource.Spec
	body := datasafesdk.CreateSecurityPolicyConfigDetails{
		CompartmentId:    common.String(strings.TrimSpace(spec.CompartmentId)),
		SecurityPolicyId: common.String(strings.TrimSpace(spec.SecurityPolicyId)),
	}
	if strings.TrimSpace(spec.DisplayName) != "" {
		body.DisplayName = common.String(strings.TrimSpace(spec.DisplayName))
	}
	if spec.Description != "" {
		body.Description = common.String(spec.Description)
	}
	if firewall, ok := securityPolicyConfigFirewallDetails(spec.FirewallConfig); ok {
		body.FirewallConfig = firewall
	}
	if audit, ok := securityPolicyConfigUnifiedAuditDetails(spec.UnifiedAuditPolicyConfig); ok {
		body.UnifiedAuditPolicyConfig = audit
	}
	if spec.FreeformTags != nil {
		body.FreeformTags = maps.Clone(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		body.DefinedTags = securityPolicyConfigDefinedTagsFromSpec(spec.DefinedTags)
	}
	return body, nil
}

func buildSecurityPolicyConfigUpdateBody(
	_ context.Context,
	resource *datasafev1beta1.SecurityPolicyConfig,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("%s resource is nil", securityPolicyConfigKind)
	}
	if err := validateSecurityPolicyConfigSpec(resource.Spec); err != nil {
		return nil, false, err
	}
	current, ok := securityPolicyConfigFromResponse(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current %s response does not expose a %s body", securityPolicyConfigKind, securityPolicyConfigKind)
	}
	if err := validateSecurityPolicyConfigCreateOnlyDrift(resource.Spec, current); err != nil {
		return nil, false, err
	}

	body, updateNeeded := securityPolicyConfigMutableUpdateBody(resource.Spec, current)
	if securityPolicyConfigCompartmentIDDrift(resource, current) {
		updateNeeded = true
	}
	return body, updateNeeded, nil
}

func securityPolicyConfigMutableUpdateBody(
	spec datasafev1beta1.SecurityPolicyConfigSpec,
	current datasafesdk.SecurityPolicyConfig,
) (datasafesdk.UpdateSecurityPolicyConfigDetails, bool) {
	body := datasafesdk.UpdateSecurityPolicyConfigDetails{}
	updateNeeded := false
	updateNeeded = setSecurityPolicyConfigDisplayName(&body, spec, current) || updateNeeded
	updateNeeded = setSecurityPolicyConfigDescription(&body, spec, current) || updateNeeded
	updateNeeded = setSecurityPolicyConfigFirewall(&body, spec, current) || updateNeeded
	updateNeeded = setSecurityPolicyConfigUnifiedAudit(&body, spec, current) || updateNeeded
	updateNeeded = setSecurityPolicyConfigFreeformTags(&body, spec, current) || updateNeeded
	updateNeeded = setSecurityPolicyConfigDefinedTags(&body, spec, current) || updateNeeded
	return body, updateNeeded
}

func setSecurityPolicyConfigDisplayName(
	body *datasafesdk.UpdateSecurityPolicyConfigDetails,
	spec datasafev1beta1.SecurityPolicyConfigSpec,
	current datasafesdk.SecurityPolicyConfig,
) bool {
	desired := strings.TrimSpace(spec.DisplayName)
	if desired == "" || securityPolicyConfigStringValue(current.DisplayName) == desired {
		return false
	}
	body.DisplayName = common.String(desired)
	return true
}

func setSecurityPolicyConfigDescription(
	body *datasafesdk.UpdateSecurityPolicyConfigDetails,
	spec datasafev1beta1.SecurityPolicyConfigSpec,
	current datasafesdk.SecurityPolicyConfig,
) bool {
	if securityPolicyConfigStringValue(current.Description) == spec.Description {
		return false
	}
	body.Description = common.String(spec.Description)
	return true
}

func setSecurityPolicyConfigFirewall(
	body *datasafesdk.UpdateSecurityPolicyConfigDetails,
	spec datasafev1beta1.SecurityPolicyConfigSpec,
	current datasafesdk.SecurityPolicyConfig,
) bool {
	details, ok := securityPolicyConfigFirewallDetails(spec.FirewallConfig)
	if !ok || securityPolicyConfigFirewallMatches(spec.FirewallConfig, current.FirewallConfig) {
		return false
	}
	body.FirewallConfig = details
	return true
}

func setSecurityPolicyConfigUnifiedAudit(
	body *datasafesdk.UpdateSecurityPolicyConfigDetails,
	spec datasafev1beta1.SecurityPolicyConfigSpec,
	current datasafesdk.SecurityPolicyConfig,
) bool {
	details, ok := securityPolicyConfigUnifiedAuditDetails(spec.UnifiedAuditPolicyConfig)
	if !ok || securityPolicyConfigUnifiedAuditMatches(spec.UnifiedAuditPolicyConfig, current.UnifiedAuditPolicyConfig) {
		return false
	}
	body.UnifiedAuditPolicyConfig = details
	return true
}

func setSecurityPolicyConfigFreeformTags(
	body *datasafesdk.UpdateSecurityPolicyConfigDetails,
	spec datasafev1beta1.SecurityPolicyConfigSpec,
	current datasafesdk.SecurityPolicyConfig,
) bool {
	if spec.FreeformTags == nil || maps.Equal(spec.FreeformTags, current.FreeformTags) {
		return false
	}
	body.FreeformTags = maps.Clone(spec.FreeformTags)
	return true
}

func setSecurityPolicyConfigDefinedTags(
	body *datasafesdk.UpdateSecurityPolicyConfigDetails,
	spec datasafev1beta1.SecurityPolicyConfigSpec,
	current datasafesdk.SecurityPolicyConfig,
) bool {
	if spec.DefinedTags == nil {
		return false
	}
	desired := securityPolicyConfigDefinedTagsFromSpec(spec.DefinedTags)
	if reflect.DeepEqual(desired, current.DefinedTags) {
		return false
	}
	body.DefinedTags = desired
	return true
}

func validateSecurityPolicyConfigSpec(spec datasafev1beta1.SecurityPolicyConfigSpec) error {
	var missing []string
	if strings.TrimSpace(spec.CompartmentId) == "" {
		missing = append(missing, "compartmentId")
	}
	if strings.TrimSpace(spec.SecurityPolicyId) == "" {
		missing = append(missing, "securityPolicyId")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%s spec is missing required field(s): %s", securityPolicyConfigKind, strings.Join(missing, ", "))
}

func validateSecurityPolicyConfigCreateOnlyDrift(
	spec datasafev1beta1.SecurityPolicyConfigSpec,
	current datasafesdk.SecurityPolicyConfig,
) error {
	desired := strings.TrimSpace(spec.SecurityPolicyId)
	observed := securityPolicyConfigStringValue(current.SecurityPolicyId)
	if desired != "" && observed != "" && desired != observed {
		return fmt.Errorf("%s securityPolicyId is immutable after creation", securityPolicyConfigKind)
	}
	return nil
}

func securityPolicyConfigFirewallDetails(
	config datasafev1beta1.SecurityPolicyConfigFirewallConfig,
) (*datasafesdk.FirewallConfigDetails, bool) {
	details := &datasafesdk.FirewallConfigDetails{}
	meaningful := false
	if strings.TrimSpace(config.Status) != "" {
		details.Status = datasafesdk.FirewallConfigDetailsStatusEnum(strings.TrimSpace(config.Status))
		meaningful = true
	}
	if strings.TrimSpace(config.ViolationLogAutoPurge) != "" {
		details.ViolationLogAutoPurge = datasafesdk.FirewallConfigDetailsViolationLogAutoPurgeEnum(strings.TrimSpace(config.ViolationLogAutoPurge))
		meaningful = true
	}
	if strings.TrimSpace(config.ExcludeJob) != "" {
		details.ExcludeJob = datasafesdk.FirewallConfigDetailsExcludeJobEnum(strings.TrimSpace(config.ExcludeJob))
		meaningful = true
	}
	return details, meaningful
}

func securityPolicyConfigUnifiedAuditDetails(
	config datasafev1beta1.SecurityPolicyConfigUnifiedAuditPolicyConfig,
) (*datasafesdk.UnifiedAuditPolicyConfigDetails, bool) {
	details := &datasafesdk.UnifiedAuditPolicyConfigDetails{}
	if strings.TrimSpace(config.ExcludeDatasafeUser) == "" {
		return details, false
	}
	details.ExcludeDatasafeUser = datasafesdk.UnifiedAuditPolicyConfigDetailsExcludeDatasafeUserEnum(strings.TrimSpace(config.ExcludeDatasafeUser))
	return details, true
}

func securityPolicyConfigFirewallMatches(
	desired datasafev1beta1.SecurityPolicyConfigFirewallConfig,
	current *datasafesdk.FirewallConfig,
) bool {
	if current == nil {
		return !securityPolicyConfigFirewallMeaningful(desired)
	}
	if strings.TrimSpace(desired.Status) != "" &&
		strings.TrimSpace(desired.Status) != string(current.Status) {
		return false
	}
	if strings.TrimSpace(desired.ViolationLogAutoPurge) != "" &&
		strings.TrimSpace(desired.ViolationLogAutoPurge) != string(current.ViolationLogAutoPurge) {
		return false
	}
	if strings.TrimSpace(desired.ExcludeJob) != "" &&
		strings.TrimSpace(desired.ExcludeJob) != string(current.ExcludeJob) {
		return false
	}
	return true
}

func securityPolicyConfigUnifiedAuditMatches(
	desired datasafev1beta1.SecurityPolicyConfigUnifiedAuditPolicyConfig,
	current *datasafesdk.UnifiedAuditPolicyConfig,
) bool {
	desiredUser := strings.TrimSpace(desired.ExcludeDatasafeUser)
	if current == nil {
		return desiredUser == ""
	}
	return desiredUser == "" || desiredUser == string(current.ExcludeDatasafeUser)
}

func securityPolicyConfigFirewallMeaningful(config datasafev1beta1.SecurityPolicyConfigFirewallConfig) bool {
	return strings.TrimSpace(config.Status) != "" ||
		strings.TrimSpace(config.ViolationLogAutoPurge) != "" ||
		strings.TrimSpace(config.ExcludeJob) != ""
}

func listSecurityPolicyConfigsAllPages(
	ctx context.Context,
	client securityPolicyConfigOCIClient,
	initErr error,
	request datasafesdk.ListSecurityPolicyConfigsRequest,
) (datasafesdk.ListSecurityPolicyConfigsResponse, error) {
	if err := requireSecurityPolicyConfigOCIClient(client, initErr); err != nil {
		return datasafesdk.ListSecurityPolicyConfigsResponse{}, err
	}

	seenPages := map[string]struct{}{}
	var combined datasafesdk.ListSecurityPolicyConfigsResponse
	for {
		response, err := client.ListSecurityPolicyConfigs(ctx, request)
		if err != nil {
			return datasafesdk.ListSecurityPolicyConfigsResponse{}, err
		}
		if combined.RawResponse == nil {
			combined.RawResponse = response.RawResponse
		}
		if combined.OpcRequestId == nil {
			combined.OpcRequestId = response.OpcRequestId
		}
		combined.Items = append(combined.Items, response.Items...)

		nextPage := securityPolicyConfigStringValue(response.OpcNextPage)
		if nextPage == "" {
			return combined, nil
		}
		if _, ok := seenPages[nextPage]; ok {
			return datasafesdk.ListSecurityPolicyConfigsResponse{}, fmt.Errorf("%s list pagination repeated page token %q", securityPolicyConfigKind, nextPage)
		}
		seenPages[nextPage] = struct{}{}
		request.Page = response.OpcNextPage
		combined.OpcNextPage = response.OpcNextPage
	}
}

func securityPolicyConfigRequiresCompartmentMove(
	resource *datasafev1beta1.SecurityPolicyConfig,
	currentResponse any,
) bool {
	if resource == nil {
		return false
	}
	current, ok := securityPolicyConfigFromResponse(currentResponse)
	return ok && securityPolicyConfigCompartmentIDDrift(resource, current)
}

func securityPolicyConfigCompartmentIDDrift(
	resource *datasafev1beta1.SecurityPolicyConfig,
	current datasafesdk.SecurityPolicyConfig,
) bool {
	if resource == nil {
		return false
	}
	desired := strings.TrimSpace(resource.Spec.CompartmentId)
	observed := securityPolicyConfigStringValue(current.CompartmentId)
	return desired != "" && observed != "" && desired != observed
}

func applySecurityPolicyConfigCompartmentMove(
	ctx context.Context,
	resource *datasafev1beta1.SecurityPolicyConfig,
	currentResponse any,
	client securityPolicyConfigOCIClient,
	initErr error,
) (servicemanager.OSOKResponse, error) {
	if resource == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s resource is nil", securityPolicyConfigKind)
	}
	if err := requireSecurityPolicyConfigOCIClient(client, initErr); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	current, ok := securityPolicyConfigFromResponse(currentResponse)
	if !ok {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("current %s response does not expose a %s body", securityPolicyConfigKind, securityPolicyConfigKind)
	}
	resourceID := securityPolicyConfigStringValue(current.Id)
	if resourceID == "" {
		resourceID = currentSecurityPolicyConfigID(resource)
	}
	if resourceID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s compartment move requires a tracked OCID", securityPolicyConfigKind)
	}
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	if compartmentID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s compartment move requires spec.compartmentId", securityPolicyConfigKind)
	}

	response, err := client.ChangeSecurityPolicyConfigCompartment(ctx, datasafesdk.ChangeSecurityPolicyConfigCompartmentRequest{
		SecurityPolicyConfigId: common.String(resourceID),
		ChangeSecurityPolicyConfigCompartmentDetails: datasafesdk.ChangeSecurityPolicyConfigCompartmentDetails{
			CompartmentId: common.String(compartmentID),
		},
		OpcRetryToken: securityPolicyConfigCompartmentMoveRetryToken(resource, compartmentID),
	})
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	workRequestID := securityPolicyConfigStringValue(response.OpcWorkRequestId)
	if workRequestID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s compartment move did not return an opc-work-request-id", securityPolicyConfigKind)
	}

	resource.Status.Id = resourceID
	resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	currentAsync, err := servicemanager.BuildWorkRequestAsyncOperation(&resource.Status.OsokStatus, securityPolicyConfigWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(datasafesdk.WorkRequestStatusAccepted),
		RawAction:        string(datasafesdk.WorkRequestOperationTypeChangeSecurityPolicyConfigCompartment),
		RawOperationType: string(datasafesdk.WorkRequestOperationTypeChangeSecurityPolicyConfigCompartment),
		WorkRequestID:    workRequestID,
		FallbackPhase:    shared.OSOKAsyncPhaseUpdate,
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	projection := servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, currentAsync, nilSecurityPolicyConfigLogger())
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: 0,
	}, nil
}

func (c *securityPolicyConfigRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *datasafev1beta1.SecurityPolicyConfig,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c == nil || c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("%s runtime client is not configured", securityPolicyConfigKind)
	}
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c *securityPolicyConfigRuntimeClient) Delete(
	ctx context.Context,
	resource *datasafev1beta1.SecurityPolicyConfig,
) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("%s resource is nil", securityPolicyConfigKind)
	}
	if c == nil || c.delegate == nil {
		return false, fmt.Errorf("%s runtime client is not configured", securityPolicyConfigKind)
	}
	if workRequestID, phase := pendingSecurityPolicyConfigWriteWorkRequest(resource); workRequestID != "" {
		resolved, err := c.resumeWriteWorkRequestBeforeDelete(ctx, resource, workRequestID, phase)
		if err != nil || !resolved {
			return false, err
		}
	}
	if workRequestID := pendingSecurityPolicyConfigDeleteWorkRequestID(resource); workRequestID != "" {
		return c.resumeDeleteWorkRequest(ctx, resource, workRequestID)
	}
	return c.deleteCurrent(ctx, resource)
}

func (c *securityPolicyConfigRuntimeClient) resumeWriteWorkRequestBeforeDelete(
	ctx context.Context,
	resource *datasafev1beta1.SecurityPolicyConfig,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (bool, error) {
	workRequest, err := getSecurityPolicyConfigWorkRequest(ctx, c.client, c.initErr, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}
	currentAsync, err := buildSecurityPolicyConfigWorkRequestAsyncOperation(resource, workRequest, workRequestID, phase)
	if err != nil {
		return false, err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, currentAsync, c.log)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		recordSecurityPolicyConfigIDFromWorkRequest(resource, workRequest, phase)
		servicemanager.ClearAsyncOperation(&resource.Status.OsokStatus)
		return true, nil
	default:
		servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, currentAsync, c.log)
		return false, fmt.Errorf("%s %s work request %s finished with status %s", securityPolicyConfigKind, phase, workRequestID, currentAsync.RawStatus)
	}
}

func (c *securityPolicyConfigRuntimeClient) deleteCurrent(
	ctx context.Context,
	resource *datasafev1beta1.SecurityPolicyConfig,
) (bool, error) {
	if err := requireSecurityPolicyConfigOCIClient(c.client, c.initErr); err != nil {
		return false, err
	}
	currentID, err := c.resolveDeleteID(ctx, resource)
	if err != nil {
		return false, err
	}
	if currentID == "" {
		markSecurityPolicyConfigDeleted(resource, c.log, "OCI SecurityPolicyConfig no longer exists")
		return true, nil
	}
	if handled, deleted, err := c.confirmPreDeleteRead(ctx, resource, currentID); handled || err != nil {
		return deleted, err
	}
	return c.startDeleteWorkRequest(ctx, resource, currentID)
}

func (c *securityPolicyConfigRuntimeClient) resolveDeleteID(
	ctx context.Context,
	resource *datasafev1beta1.SecurityPolicyConfig,
) (string, error) {
	currentID := currentSecurityPolicyConfigID(resource)
	if currentID != "" {
		return currentID, nil
	}
	current, found, err := c.lookupExistingForDelete(ctx, resource)
	if err != nil || !found {
		return "", err
	}
	if err := projectSecurityPolicyConfigSDKStatus(resource, current); err != nil {
		return "", err
	}
	return securityPolicyConfigStringValue(current.Id), nil
}

func (c *securityPolicyConfigRuntimeClient) lookupExistingForDelete(
	ctx context.Context,
	resource *datasafev1beta1.SecurityPolicyConfig,
) (datasafesdk.SecurityPolicyConfig, bool, error) {
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" || strings.TrimSpace(resource.Spec.SecurityPolicyId) == "" {
		return datasafesdk.SecurityPolicyConfig{}, false, nil
	}
	response, err := listSecurityPolicyConfigsAllPages(ctx, c.client, c.initErr, datasafesdk.ListSecurityPolicyConfigsRequest{
		CompartmentId:    common.String(strings.TrimSpace(resource.Spec.CompartmentId)),
		SecurityPolicyId: common.String(strings.TrimSpace(resource.Spec.SecurityPolicyId)),
	})
	if err != nil {
		return datasafesdk.SecurityPolicyConfig{}, false, err
	}
	var matches []datasafesdk.SecurityPolicyConfig
	for _, item := range response.Items {
		current := securityPolicyConfigFromSummary(item)
		if securityPolicyConfigMatchesDesired(resource, current) {
			matches = append(matches, current)
		}
	}
	switch len(matches) {
	case 0:
		return datasafesdk.SecurityPolicyConfig{}, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return datasafesdk.SecurityPolicyConfig{}, false, fmt.Errorf("%s list response returned multiple matching resources", securityPolicyConfigKind)
	}
}

func (c *securityPolicyConfigRuntimeClient) confirmPreDeleteRead(
	ctx context.Context,
	resource *datasafev1beta1.SecurityPolicyConfig,
	currentID string,
) (bool, bool, error) {
	response, err := c.client.GetSecurityPolicyConfig(ctx, datasafesdk.GetSecurityPolicyConfigRequest{
		SecurityPolicyConfigId: common.String(currentID),
	})
	if err != nil {
		return handleSecurityPolicyConfigPreDeleteReadError(resource, c.log, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	current := response.SecurityPolicyConfig
	if err := projectSecurityPolicyConfigSDKStatus(resource, current); err != nil {
		return true, false, err
	}

	switch current.LifecycleState {
	case datasafesdk.SecurityPolicyConfigLifecycleStateCreating,
		datasafesdk.SecurityPolicyConfigLifecycleStateUpdating:
		markSecurityPolicyConfigLifecyclePending(resource, c.log, current.LifecycleState)
		return true, false, nil
	case datasafesdk.SecurityPolicyConfigLifecycleStateDeleting:
		markSecurityPolicyConfigTerminating(resource, c.log, "OCI resource delete is in progress")
		return true, false, nil
	case datasafesdk.SecurityPolicyConfigLifecycleStateDeleted:
		markSecurityPolicyConfigDeleted(resource, c.log, "OCI resource deleted")
		return true, true, nil
	default:
		return false, false, nil
	}
}

func handleSecurityPolicyConfigPreDeleteReadError(
	resource *datasafev1beta1.SecurityPolicyConfig,
	log loggerutil.OSOKLogger,
	err error,
) (bool, bool, error) {
	classification := errorutil.ClassifyDeleteError(err)
	switch {
	case classification.IsUnambiguousNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markSecurityPolicyConfigDeleted(resource, log, "OCI resource no longer exists")
		return true, true, nil
	case classification.IsAuthShapedNotFound():
		return true, false, rejectSecurityPolicyConfigAuthShapedNotFound(resource, err)
	default:
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, false, err
	}
}

func (c *securityPolicyConfigRuntimeClient) startDeleteWorkRequest(
	ctx context.Context,
	resource *datasafev1beta1.SecurityPolicyConfig,
	currentID string,
) (bool, error) {
	response, err := c.client.DeleteSecurityPolicyConfig(ctx, datasafesdk.DeleteSecurityPolicyConfigRequest{
		SecurityPolicyConfigId: common.String(currentID),
	})
	if err != nil {
		return handleSecurityPolicyConfigDeleteRequestError(resource, c.log, err)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)

	workRequestID := securityPolicyConfigStringValue(response.OpcWorkRequestId)
	if workRequestID == "" {
		return c.confirmDeleteAfterRequest(ctx, resource, currentID)
	}
	resource.Status.Id = currentID
	resource.Status.OsokStatus.Ocid = shared.OCID(currentID)
	currentAsync, err := servicemanager.BuildWorkRequestAsyncOperation(&resource.Status.OsokStatus, securityPolicyConfigWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(datasafesdk.WorkRequestStatusAccepted),
		RawAction:        string(datasafesdk.WorkRequestOperationTypeDeleteSecurityPolicyConfig),
		RawOperationType: string(datasafesdk.WorkRequestOperationTypeDeleteSecurityPolicyConfig),
		WorkRequestID:    workRequestID,
		FallbackPhase:    shared.OSOKAsyncPhaseDelete,
	})
	if err != nil {
		return false, err
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, currentAsync, c.log)
	return c.resumeDeleteWorkRequest(ctx, resource, workRequestID)
}

func handleSecurityPolicyConfigDeleteRequestError(
	resource *datasafev1beta1.SecurityPolicyConfig,
	log loggerutil.OSOKLogger,
	err error,
) (bool, error) {
	classification := errorutil.ClassifyDeleteError(err)
	switch {
	case classification.IsUnambiguousNotFound():
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		markSecurityPolicyConfigDeleted(resource, log, "OCI resource no longer exists")
		return true, nil
	case classification.IsAuthShapedNotFound():
		return false, rejectSecurityPolicyConfigAuthShapedNotFound(resource, err)
	default:
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}
}

func (c *securityPolicyConfigRuntimeClient) confirmDeleteAfterRequest(
	ctx context.Context,
	resource *datasafev1beta1.SecurityPolicyConfig,
	currentID string,
) (bool, error) {
	response, err := c.client.GetSecurityPolicyConfig(ctx, datasafesdk.GetSecurityPolicyConfigRequest{
		SecurityPolicyConfigId: common.String(currentID),
	})
	if err != nil {
		classification := errorutil.ClassifyDeleteError(err)
		switch {
		case classification.IsUnambiguousNotFound():
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			markSecurityPolicyConfigDeleted(resource, c.log, "OCI resource deleted")
			return true, nil
		case classification.IsAuthShapedNotFound():
			return false, rejectSecurityPolicyConfigAuthShapedNotFound(resource, err)
		default:
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return false, err
		}
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	current := response.SecurityPolicyConfig
	if err := projectSecurityPolicyConfigSDKStatus(resource, current); err != nil {
		return false, err
	}
	if current.LifecycleState == datasafesdk.SecurityPolicyConfigLifecycleStateDeleted {
		markSecurityPolicyConfigDeleted(resource, c.log, "OCI resource deleted")
		return true, nil
	}
	markSecurityPolicyConfigTerminating(resource, c.log, "OCI resource delete is in progress")
	return false, nil
}

func (c *securityPolicyConfigRuntimeClient) resumeDeleteWorkRequest(
	ctx context.Context,
	resource *datasafev1beta1.SecurityPolicyConfig,
	workRequestID string,
) (bool, error) {
	workRequest, err := getSecurityPolicyConfigWorkRequest(ctx, c.client, c.initErr, workRequestID)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}
	currentAsync, err := buildSecurityPolicyConfigWorkRequestAsyncOperation(resource, workRequest, workRequestID, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return false, err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, currentAsync, c.log)
		return false, nil
	case shared.OSOKAsyncClassSucceeded:
		return c.confirmDeleteAfterSucceededWorkRequest(ctx, resource)
	default:
		servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, currentAsync, c.log)
		return false, fmt.Errorf("%s delete work request %s finished with status %s", securityPolicyConfigKind, workRequestID, currentAsync.RawStatus)
	}
}

func (c *securityPolicyConfigRuntimeClient) confirmDeleteAfterSucceededWorkRequest(
	ctx context.Context,
	resource *datasafev1beta1.SecurityPolicyConfig,
) (bool, error) {
	currentID := currentSecurityPolicyConfigID(resource)
	if currentID == "" {
		markSecurityPolicyConfigDeleted(resource, c.log, "OCI SecurityPolicyConfig delete work request completed")
		return true, nil
	}
	response, err := c.client.GetSecurityPolicyConfig(ctx, datasafesdk.GetSecurityPolicyConfigRequest{
		SecurityPolicyConfigId: common.String(currentID),
	})
	if err != nil {
		classification := errorutil.ClassifyDeleteError(err)
		switch {
		case classification.IsUnambiguousNotFound():
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			markSecurityPolicyConfigDeleted(resource, c.log, "OCI resource deleted")
			return true, nil
		case classification.IsAuthShapedNotFound():
			return false, rejectSecurityPolicyConfigAuthShapedNotFound(resource, err)
		default:
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			return false, err
		}
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	current := response.SecurityPolicyConfig
	if err := projectSecurityPolicyConfigSDKStatus(resource, current); err != nil {
		return false, err
	}
	switch current.LifecycleState {
	case datasafesdk.SecurityPolicyConfigLifecycleStateDeleted:
		markSecurityPolicyConfigDeleted(resource, c.log, "OCI resource deleted")
		return true, nil
	case datasafesdk.SecurityPolicyConfigLifecycleStateDeleting, "":
		markSecurityPolicyConfigTerminating(resource, c.log, "OCI resource delete is in progress")
		return false, nil
	default:
		return false, fmt.Errorf("%s delete work request succeeded but readback lifecycle state is %q", securityPolicyConfigKind, current.LifecycleState)
	}
}

func pendingSecurityPolicyConfigWriteWorkRequest(resource *datasafev1beta1.SecurityPolicyConfig) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	switch current.Phase {
	case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
	default:
		return "", ""
	}
	switch current.NormalizedClass {
	case shared.OSOKAsyncClassSucceeded, shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled:
		return "", ""
	default:
		return strings.TrimSpace(current.WorkRequestID), current.Phase
	}
}

func pendingSecurityPolicyConfigDeleteWorkRequestID(resource *datasafev1beta1.SecurityPolicyConfig) string {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Phase != shared.OSOKAsyncPhaseDelete {
		return ""
	}
	return strings.TrimSpace(current.WorkRequestID)
}

func confirmSecurityPolicyConfigDeleteRead(
	ctx context.Context,
	get func(context.Context, datasafesdk.GetSecurityPolicyConfigRequest) (datasafesdk.GetSecurityPolicyConfigResponse, error),
	currentID string,
) (any, error) {
	if get == nil {
		return nil, fmt.Errorf("confirm %s delete: get hook is not configured", securityPolicyConfigKind)
	}
	currentID = strings.TrimSpace(currentID)
	if currentID == "" {
		return nil, fmt.Errorf("confirm %s delete: OCID is empty", securityPolicyConfigKind)
	}
	response, err := get(ctx, datasafesdk.GetSecurityPolicyConfigRequest{
		SecurityPolicyConfigId: common.String(currentID),
	})
	if err == nil {
		return response, nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return securityPolicyConfigAuthShapedConfirmRead{err: err}, nil
	}
	return nil, err
}

func handleSecurityPolicyConfigDeleteError(resource *datasafev1beta1.SecurityPolicyConfig, err error) error {
	if err == nil {
		return nil
	}
	if errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return rejectSecurityPolicyConfigAuthShapedNotFound(resource, err)
	}
	return err
}

func handleSecurityPolicyConfigDeleteConfirmReadOutcome(
	resource *datasafev1beta1.SecurityPolicyConfig,
	response any,
	_ generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	switch typed := response.(type) {
	case securityPolicyConfigAuthShapedConfirmRead:
		recordSecurityPolicyConfigConfirmReadRequestID(resource, typed)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, typed
	case *securityPolicyConfigAuthShapedConfirmRead:
		if typed != nil {
			recordSecurityPolicyConfigConfirmReadRequestID(resource, *typed)
			return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, *typed
		}
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func rejectSecurityPolicyConfigAuthShapedNotFound(resource *datasafev1beta1.SecurityPolicyConfig, err error) error {
	if resource != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
	}
	if !errorutil.ClassifyDeleteError(err).IsAuthShapedNotFound() {
		return err
	}
	return securityPolicyConfigAuthShapedNotFound{err: err}
}

func recordSecurityPolicyConfigConfirmReadRequestID(
	resource *datasafev1beta1.SecurityPolicyConfig,
	err securityPolicyConfigAuthShapedConfirmRead,
) {
	if resource == nil {
		return
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
}

func getSecurityPolicyConfigWorkRequest(
	ctx context.Context,
	client securityPolicyConfigOCIClient,
	initErr error,
	workRequestID string,
) (datasafesdk.WorkRequest, error) {
	if err := requireSecurityPolicyConfigOCIClient(client, initErr); err != nil {
		return datasafesdk.WorkRequest{}, err
	}
	response, err := client.GetWorkRequest(ctx, datasafesdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return datasafesdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func buildSecurityPolicyConfigWorkRequestAsyncOperation(
	resource *datasafev1beta1.SecurityPolicyConfig,
	workRequest datasafesdk.WorkRequest,
	workRequestID string,
	fallback shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	if strings.TrimSpace(securityPolicyConfigStringValue(workRequest.Id)) == "" {
		workRequest.Id = common.String(workRequestID)
	}
	rawAction, err := resolveSecurityPolicyConfigWorkRequestAction(workRequest)
	if err != nil {
		return nil, err
	}
	return servicemanager.BuildWorkRequestAsyncOperation(&resource.Status.OsokStatus, securityPolicyConfigWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.Status),
		RawAction:        rawAction,
		RawOperationType: string(workRequest.OperationType),
		WorkRequestID:    securityPolicyConfigStringValue(workRequest.Id),
		PercentComplete:  workRequest.PercentComplete,
		FallbackPhase:    fallback,
	})
}

func resolveSecurityPolicyConfigWorkRequestAction(workRequest any) (string, error) {
	current, ok := securityPolicyConfigWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("%s work request has unexpected type %T", securityPolicyConfigKind, workRequest)
	}
	if current.OperationType != "" {
		return string(current.OperationType), nil
	}

	var action string
	for _, resource := range current.Resources {
		if !isSecurityPolicyConfigWorkRequestResource(resource) || securityPolicyConfigIgnorableAction(resource.ActionType) {
			continue
		}
		candidate := strings.TrimSpace(string(resource.ActionType))
		if candidate == "" {
			continue
		}
		if action == "" {
			action = candidate
			continue
		}
		if action != candidate {
			return "", fmt.Errorf("%s work request %s exposes conflicting action types %q and %q", securityPolicyConfigKind, securityPolicyConfigStringValue(current.Id), action, candidate)
		}
	}
	return action, nil
}

func recoverSecurityPolicyConfigIDFromWorkRequest(
	_ *datasafev1beta1.SecurityPolicyConfig,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	current, ok := securityPolicyConfigWorkRequestFromAny(workRequest)
	if !ok {
		return "", fmt.Errorf("%s work request has unexpected type %T", securityPolicyConfigKind, workRequest)
	}
	action := securityPolicyConfigActionForPhase(phase)
	if id, ok := securityPolicyConfigIDFromWorkRequestResources(current.Resources, action, true); ok {
		return id, nil
	}
	id, _ := securityPolicyConfigIDFromWorkRequestResources(current.Resources, "", false)
	return id, nil
}

func recordSecurityPolicyConfigIDFromWorkRequest(
	resource *datasafev1beta1.SecurityPolicyConfig,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) {
	if resource == nil || currentSecurityPolicyConfigID(resource) != "" {
		return
	}
	resourceID, err := recoverSecurityPolicyConfigIDFromWorkRequest(resource, workRequest, phase)
	if err != nil || strings.TrimSpace(resourceID) == "" {
		return
	}
	resource.Status.Id = strings.TrimSpace(resourceID)
	resource.Status.OsokStatus.Ocid = shared.OCID(strings.TrimSpace(resourceID))
}

func securityPolicyConfigIDFromWorkRequestResources(
	resources []datasafesdk.WorkRequestResource,
	action datasafesdk.WorkRequestResourceActionTypeEnum,
	requireUnique bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		id, ok := securityPolicyConfigIDFromWorkRequestResource(resource, action)
		if !ok {
			continue
		}
		if !requireUnique {
			return id, true
		}
		if candidate != "" && candidate != id {
			return "", false
		}
		candidate = id
	}
	return candidate, candidate != ""
}

func securityPolicyConfigIDFromWorkRequestResource(
	resource datasafesdk.WorkRequestResource,
	action datasafesdk.WorkRequestResourceActionTypeEnum,
) (string, bool) {
	if !isSecurityPolicyConfigWorkRequestResource(resource) {
		return "", false
	}
	if !securityPolicyConfigWorkRequestActionMatches(resource.ActionType, action) {
		return "", false
	}
	id := strings.TrimSpace(securityPolicyConfigStringValue(resource.Identifier))
	return id, id != ""
}

func isSecurityPolicyConfigWorkRequestResource(resource datasafesdk.WorkRequestResource) bool {
	entityType := normalizeSecurityPolicyConfigEntityType(securityPolicyConfigStringValue(resource.EntityType))
	return entityType == "securitypolicyconfig" || strings.Contains(entityType, "securitypolicyconfig")
}

func normalizeSecurityPolicyConfigEntityType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "", "-", "", " ", "", ".", "", "/", "")
	return replacer.Replace(value)
}

func securityPolicyConfigWorkRequestActionMatches(
	resourceAction datasafesdk.WorkRequestResourceActionTypeEnum,
	action datasafesdk.WorkRequestResourceActionTypeEnum,
) bool {
	if action == "" {
		return !securityPolicyConfigIgnorableAction(resourceAction)
	}
	return resourceAction == action || resourceAction == datasafesdk.WorkRequestResourceActionTypeInProgress
}

func securityPolicyConfigActionForPhase(phase shared.OSOKAsyncPhase) datasafesdk.WorkRequestResourceActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return datasafesdk.WorkRequestResourceActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return datasafesdk.WorkRequestResourceActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return datasafesdk.WorkRequestResourceActionTypeDeleted
	default:
		return ""
	}
}

func securityPolicyConfigIgnorableAction(action datasafesdk.WorkRequestResourceActionTypeEnum) bool {
	return action == "" || action == datasafesdk.WorkRequestResourceActionTypeFailed
}

func securityPolicyConfigWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	current, ok := securityPolicyConfigWorkRequestFromAny(workRequest)
	if !ok {
		return ""
	}
	workRequestID := securityPolicyConfigStringValue(current.Id)
	status := strings.TrimSpace(string(current.Status))
	if workRequestID == "" || status == "" {
		return ""
	}
	return fmt.Sprintf("%s %s work request %s is %s", securityPolicyConfigKind, phase, workRequestID, status)
}

func securityPolicyConfigWorkRequestFromAny(value any) (datasafesdk.WorkRequest, bool) {
	current, ok := securityPolicyConfigDereference(value).(datasafesdk.WorkRequest)
	return current, ok
}

func projectSecurityPolicyConfigStatus(
	resource *datasafev1beta1.SecurityPolicyConfig,
	response any,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", securityPolicyConfigKind)
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
	current, ok := securityPolicyConfigFromResponse(response)
	if !ok {
		return nil
	}
	return projectSecurityPolicyConfigSDKStatus(resource, current)
}

func projectSecurityPolicyConfigSDKStatus(
	resource *datasafev1beta1.SecurityPolicyConfig,
	current datasafesdk.SecurityPolicyConfig,
) error {
	if resource == nil {
		return fmt.Errorf("%s resource is nil", securityPolicyConfigKind)
	}
	status := &resource.Status
	status.Id = securityPolicyConfigStringValue(current.Id)
	status.CompartmentId = securityPolicyConfigStringValue(current.CompartmentId)
	status.DisplayName = securityPolicyConfigStringValue(current.DisplayName)
	status.SecurityPolicyId = securityPolicyConfigStringValue(current.SecurityPolicyId)
	status.TimeCreated = securityPolicyConfigTimeString(current.TimeCreated)
	status.LifecycleState = string(current.LifecycleState)
	status.Description = securityPolicyConfigStringValue(current.Description)
	status.TimeUpdated = securityPolicyConfigTimeString(current.TimeUpdated)
	status.LifecycleDetails = securityPolicyConfigStringValue(current.LifecycleDetails)
	status.FirewallConfig = securityPolicyConfigFirewallStatus(current.FirewallConfig)
	status.UnifiedAuditPolicyConfig = securityPolicyConfigUnifiedAuditStatus(current.UnifiedAuditPolicyConfig)
	status.FreeformTags = maps.Clone(current.FreeformTags)
	status.DefinedTags = securityPolicyConfigStatusTagsFromSDK(current.DefinedTags)
	status.SystemTags = securityPolicyConfigStatusTagsFromSDK(current.SystemTags)
	if status.Id != "" {
		status.OsokStatus.Ocid = shared.OCID(status.Id)
	}
	return nil
}

func securityPolicyConfigFromResponse(response any) (datasafesdk.SecurityPolicyConfig, bool) {
	switch current := securityPolicyConfigDereference(response).(type) {
	case datasafesdk.SecurityPolicyConfig:
		return current, true
	case datasafesdk.SecurityPolicyConfigSummary:
		return securityPolicyConfigFromSummary(current), true
	case datasafesdk.CreateSecurityPolicyConfigResponse:
		return current.SecurityPolicyConfig, true
	case datasafesdk.GetSecurityPolicyConfigResponse:
		return current.SecurityPolicyConfig, true
	default:
		return datasafesdk.SecurityPolicyConfig{}, false
	}
}

func securityPolicyConfigDereference(value any) any {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() || reflected.Kind() != reflect.Pointer {
		return value
	}
	if reflected.IsNil() {
		return nil
	}
	return reflected.Elem().Interface()
}

func securityPolicyConfigFromSummary(summary datasafesdk.SecurityPolicyConfigSummary) datasafesdk.SecurityPolicyConfig {
	return datasafesdk.SecurityPolicyConfig(summary)
}

func securityPolicyConfigMatchesDesired(
	resource *datasafev1beta1.SecurityPolicyConfig,
	current datasafesdk.SecurityPolicyConfig,
) bool {
	if resource == nil {
		return false
	}
	return strings.TrimSpace(resource.Spec.CompartmentId) == securityPolicyConfigStringValue(current.CompartmentId) &&
		strings.TrimSpace(resource.Spec.SecurityPolicyId) == securityPolicyConfigStringValue(current.SecurityPolicyId)
}

func clearTrackedSecurityPolicyConfigIdentity(resource *datasafev1beta1.SecurityPolicyConfig) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus.Ocid = ""
}

func currentSecurityPolicyConfigID(resource *datasafev1beta1.SecurityPolicyConfig) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	}
	return strings.TrimSpace(resource.Status.Id)
}

func markSecurityPolicyConfigDeleted(
	resource *datasafev1beta1.SecurityPolicyConfig,
	log loggerutil.OSOKLogger,
	message string,
) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, corev1.ConditionTrue, "", message, log)
}

func markSecurityPolicyConfigTerminating(
	resource *datasafev1beta1.SecurityPolicyConfig,
	log loggerutil.OSOKLogger,
	message string,
) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         message,
		UpdatedAt:       &now,
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, log)
}

func markSecurityPolicyConfigLifecyclePending(
	resource *datasafev1beta1.SecurityPolicyConfig,
	log loggerutil.OSOKLogger,
	state datasafesdk.SecurityPolicyConfigLifecycleStateEnum,
) {
	if resource == nil {
		return
	}
	message := fmt.Sprintf("OCI %s is %s; waiting before delete", securityPolicyConfigKind, state)
	current := servicemanager.NewLifecycleAsyncOperation(&resource.Status.OsokStatus, string(state), message, "")
	if current == nil {
		return
	}
	servicemanager.ApplyAsyncOperation(&resource.Status.OsokStatus, current, log)
}

func nilSecurityPolicyConfigLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{}
}

func securityPolicyConfigFirewallStatus(current *datasafesdk.FirewallConfig) datasafev1beta1.SecurityPolicyConfigFirewallConfig {
	if current == nil {
		return datasafev1beta1.SecurityPolicyConfigFirewallConfig{}
	}
	return datasafev1beta1.SecurityPolicyConfigFirewallConfig{
		Status:                string(current.Status),
		ViolationLogAutoPurge: string(current.ViolationLogAutoPurge),
		ExcludeJob:            string(current.ExcludeJob),
	}
}

func securityPolicyConfigUnifiedAuditStatus(
	current *datasafesdk.UnifiedAuditPolicyConfig,
) datasafev1beta1.SecurityPolicyConfigUnifiedAuditPolicyConfig {
	if current == nil {
		return datasafev1beta1.SecurityPolicyConfigUnifiedAuditPolicyConfig{}
	}
	return datasafev1beta1.SecurityPolicyConfigUnifiedAuditPolicyConfig{
		ExcludeDatasafeUser: string(current.ExcludeDatasafeUser),
	}
}

func securityPolicyConfigDefinedTagsFromSpec(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if tags == nil {
		return nil
	}
	definedTags := make(map[string]map[string]interface{}, len(tags))
	for namespace, values := range tags {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		definedTags[namespace] = converted
	}
	return definedTags
}

func securityPolicyConfigStatusTagsFromSDK(tags map[string]map[string]interface{}) map[string]shared.MapValue {
	if tags == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(tags))
	for namespace, values := range tags {
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = tagValues
	}
	return converted
}

func securityPolicyConfigCompartmentMoveRetryToken(
	resource *datasafev1beta1.SecurityPolicyConfig,
	compartmentID string,
) *string {
	if resource == nil {
		return nil
	}
	parts := []string{
		string(resource.UID),
		resource.Namespace,
		resource.Name,
		strings.TrimSpace(compartmentID),
	}
	return common.String(strings.Join(parts, ":"))
}

func securityPolicyConfigTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func securityPolicyConfigStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
