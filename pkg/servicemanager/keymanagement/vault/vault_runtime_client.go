/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	keymanagementsdk "github.com/oracle/oci-go-sdk/v65/keymanagement"
	keymanagementv1beta1 "github.com/oracle/oci-service-operator/api/keymanagement/v1beta1"
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

const (
	defaultVaultDeletionScheduleDays int32 = 30
	vaultRequeueDuration                   = time.Minute
)

type vaultOCIClient interface {
	CreateVault(context.Context, keymanagementsdk.CreateVaultRequest) (keymanagementsdk.CreateVaultResponse, error)
	GetVault(context.Context, keymanagementsdk.GetVaultRequest) (keymanagementsdk.GetVaultResponse, error)
	ListVaults(context.Context, keymanagementsdk.ListVaultsRequest) (keymanagementsdk.ListVaultsResponse, error)
	UpdateVault(context.Context, keymanagementsdk.UpdateVaultRequest) (keymanagementsdk.UpdateVaultResponse, error)
	ScheduleVaultDeletion(context.Context, keymanagementsdk.ScheduleVaultDeletionRequest) (keymanagementsdk.ScheduleVaultDeletionResponse, error)
}

type vaultRuntimeClient struct {
	delegate VaultServiceClient
	sdk      vaultOCIClient
	log      loggerutil.OSOKLogger
	initErr  error
}

func init() {
	registerVaultRuntimeHooksMutator(func(manager *VaultServiceManager, hooks *VaultRuntimeHooks) {
		applyVaultRuntimeHooks(manager, hooks)
		appendVaultRuntimeWrapper(manager, hooks)
	})
}

func applyVaultRuntimeHooks(manager *VaultServiceManager, hooks *VaultRuntimeHooks) {
	if manager == nil || hooks == nil {
		return
	}

	hooks.Semantics = newVaultRuntimeSemantics()
	hooks.BuildCreateBody = func(ctx context.Context, resource *keymanagementv1beta1.Vault, namespace string) (any, error) {
		return buildVaultCreateDetails(ctx, resource, namespace)
	}
}

func appendVaultRuntimeWrapper(manager *VaultServiceManager, hooks *VaultRuntimeHooks) {
	if manager == nil || hooks == nil {
		return
	}

	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate VaultServiceClient) VaultServiceClient {
		return newVaultRuntimeClient(manager, delegate)
	})
}

func newVaultRuntimeClient(manager *VaultServiceManager, delegate VaultServiceClient) *vaultRuntimeClient {
	runtimeClient := &vaultRuntimeClient{delegate: delegate}
	if manager == nil {
		return runtimeClient
	}

	runtimeClient.log = manager.Log
	if manager.Provider == nil {
		return runtimeClient
	}
	sdkClient, err := keymanagementsdk.NewKmsVaultClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		runtimeClient.initErr = fmt.Errorf("initialize Vault OCI client: %w", err)
		return runtimeClient
	}

	runtimeClient.sdk = sdkClient
	return runtimeClient
}

func newVaultRuntimeConfig(log loggerutil.OSOKLogger, sdkClient vaultOCIClient) generatedruntime.Config[*keymanagementv1beta1.Vault] {
	return generatedruntime.Config[*keymanagementv1beta1.Vault]{
		Kind:    "Vault",
		SDKName: "Vault",
		Log:     log,
		BuildCreateBody: func(ctx context.Context, resource *keymanagementv1beta1.Vault, namespace string) (any, error) {
			return buildVaultCreateDetails(ctx, resource, namespace)
		},
		Semantics: newVaultRuntimeSemantics(),
		Create: &generatedruntime.Operation{
			NewRequest: func() any { return &keymanagementsdk.CreateVaultRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.CreateVault(ctx, *request.(*keymanagementsdk.CreateVaultRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "CreateVaultDetails", RequestName: "CreateVaultDetails", Contribution: "body", PreferResourceID: false},
			},
		},
		Get: &generatedruntime.Operation{
			NewRequest: func() any { return &keymanagementsdk.GetVaultRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.GetVault(ctx, *request.(*keymanagementsdk.GetVaultRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "VaultId", RequestName: "vaultId", Contribution: "path", PreferResourceID: true},
			},
		},
		List: &generatedruntime.Operation{
			NewRequest: func() any { return &keymanagementsdk.ListVaultsRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.ListVaults(ctx, *request.(*keymanagementsdk.ListVaultsRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
				{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
				{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
				{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
			},
		},
		Update: &generatedruntime.Operation{
			NewRequest: func() any { return &keymanagementsdk.UpdateVaultRequest{} },
			Call: func(ctx context.Context, request any) (any, error) {
				return sdkClient.UpdateVault(ctx, *request.(*keymanagementsdk.UpdateVaultRequest))
			},
			Fields: []generatedruntime.RequestField{
				{FieldName: "VaultId", RequestName: "vaultId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateVaultDetails", RequestName: "UpdateVaultDetails", Contribution: "body", PreferResourceID: false},
			},
		},
	}
}

func newVaultRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"BACKUP_IN_PROGRESS", "CREATING", "RESTORING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"CANCELLING_DELETION", "DELETING", "PENDING_DELETION", "SCHEDULING_DELETION"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "vaultType"},
		},
		Mutation: generatedruntime.MutationSemantics{
			UpdateCandidate: []string{"definedTags", "displayName", "freeformTags"},
			Mutable:         []string{"definedTags", "displayName", "freeformTags"},
			ForceNew:        []string{"compartmentId", "externalKeyManagerMetadata", "vaultType"},
			ConflictsWith:   map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
	}
}

func buildVaultCreateDetails(ctx context.Context, resource *keymanagementv1beta1.Vault, namespace string) (keymanagementsdk.CreateVaultDetails, error) {
	resolvedSpec, err := generatedruntime.ResolveSpecValue(resource, ctx, nil, namespace)
	if err != nil {
		return keymanagementsdk.CreateVaultDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return keymanagementsdk.CreateVaultDetails{}, fmt.Errorf("marshal resolved Vault spec: %w", err)
	}

	var details keymanagementsdk.CreateVaultDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return keymanagementsdk.CreateVaultDetails{}, fmt.Errorf("decode Vault create body: %w", err)
	}
	return details, nil
}

func (c *vaultRuntimeClient) CreateOrUpdate(ctx context.Context, resource *keymanagementv1beta1.Vault, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Vault delegate is not configured")
	}

	working := cloneVaultForGeneratedRuntime(resource)
	response, err := c.delegate.CreateOrUpdate(ctx, &working, req)
	resource.Status = working.Status
	if err != nil || !response.IsSuccessful {
		return response, err
	}

	action := desiredVaultDeletionAction(resource)
	switch action {
	case vaultDeletionActionNone:
		return response, nil
	case vaultDeletionActionWait:
		return ensureVaultRequeue(response), nil
	case vaultDeletionActionSchedule:
		return c.scheduleVaultDeletion(ctx, resource)
	default:
		return response, nil
	}
}

func (c *vaultRuntimeClient) Delete(ctx context.Context, resource *keymanagementv1beta1.Vault) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}
	if c.sdk == nil {
		return false, fmt.Errorf("Vault OCI client is not configured")
	}

	current, found, err := c.resolveVault(ctx, resource)
	if err != nil {
		return false, normalizeVaultOCIError(err)
	}
	if !found {
		c.markVaultDeleted(resource, "OCI Vault no longer exists")
		clearVaultDeletionScheduleStatus(resource)
		return true, nil
	}

	c.syncVaultStatus(resource, current)

	switch lifecycleState := strings.ToUpper(string(current.LifecycleState)); lifecycleState {
	case "DELETED":
		c.markVaultDeleted(resource, "OCI Vault deleted")
		clearVaultDeletionScheduleStatus(resource)
		return true, nil
	case "PENDING_DELETION", "SCHEDULING_DELETION", "DELETING":
		c.markVaultCondition(resource, shared.Terminating, vaultLifecycleMessage(lifecycleState))
		return true, nil
	case "CANCELLING_DELETION":
		c.markVaultCondition(resource, shared.Terminating, vaultLifecycleMessage(lifecycleState))
		return false, nil
	}

	vaultID := currentVaultID(resource)
	if vaultID == "" {
		return false, fmt.Errorf("Vault identity is not recorded")
	}

	days, err := effectiveVaultDeletionScheduleDays(resource.Spec.DeletionScheduleDays, true)
	if err != nil {
		return false, err
	}

	response, err := c.sdk.ScheduleVaultDeletion(ctx, keymanagementsdk.ScheduleVaultDeletionRequest{
		VaultId: common.String(vaultID),
		ScheduleVaultDeletionDetails: keymanagementsdk.ScheduleVaultDeletionDetails{
			TimeOfDeletion: sdkTimeForDeletionDays(days),
		},
	})
	if err != nil {
		if vaultConflict(err) {
			return c.confirmVaultDeleteAccepted(ctx, resource)
		}
		if vaultNotFound(err) {
			c.markVaultDeleted(resource, "OCI Vault no longer exists")
			clearVaultDeletionScheduleStatus(resource)
			return true, nil
		}
		return false, normalizeVaultOCIError(err)
	}

	c.syncVaultStatus(resource, response.Vault)
	resource.Status.RequestedDeletionScheduleDays = days
	c.markVaultCondition(resource, shared.Terminating, vaultLifecycleMessage(strings.ToUpper(string(response.Vault.LifecycleState))))
	return true, nil
}

type vaultDeletionAction string

const (
	vaultDeletionActionNone     vaultDeletionAction = "none"
	vaultDeletionActionWait     vaultDeletionAction = "wait"
	vaultDeletionActionSchedule vaultDeletionAction = "schedule"
)

func desiredVaultDeletionAction(resource *keymanagementv1beta1.Vault) vaultDeletionAction {
	requested := resource.Spec.DeletionScheduleDays
	lifecycleState := strings.ToUpper(strings.TrimSpace(resource.Status.LifecycleState))
	scheduled := strings.TrimSpace(resource.Status.TimeOfDeletion) != ""

	switch lifecycleState {
	case "BACKUP_IN_PROGRESS", "CREATING", "RESTORING", "UPDATING":
		return vaultDeletionActionWait
	case "CANCELLING_DELETION", "DELETING", "SCHEDULING_DELETION":
		return vaultDeletionActionWait
	}

	if requested <= 0 {
		return vaultDeletionActionNone
	}

	if scheduled || lifecycleState == "PENDING_DELETION" {
		return vaultDeletionActionNone
	}

	return vaultDeletionActionSchedule
}

func (c *vaultRuntimeClient) scheduleVaultDeletion(ctx context.Context, resource *keymanagementv1beta1.Vault) (servicemanager.OSOKResponse, error) {
	vaultID := currentVaultID(resource)
	if vaultID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("Vault identity is not recorded")
	}

	days, err := effectiveVaultDeletionScheduleDays(resource.Spec.DeletionScheduleDays, false)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	response, err := c.sdk.ScheduleVaultDeletion(ctx, keymanagementsdk.ScheduleVaultDeletionRequest{
		VaultId: common.String(vaultID),
		ScheduleVaultDeletionDetails: keymanagementsdk.ScheduleVaultDeletionDetails{
			TimeOfDeletion: sdkTimeForDeletionDays(days),
		},
	})
	if err != nil {
		if vaultConflict(err) {
			return c.observeVaultAfterControlPlaneConflict(ctx, resource)
		}
		return servicemanager.OSOKResponse{IsSuccessful: false}, normalizeVaultOCIError(err)
	}

	c.syncVaultStatus(resource, response.Vault)
	resource.Status.RequestedDeletionScheduleDays = days
	return c.markVaultReconcileSuccess(resource, strings.ToUpper(string(response.Vault.LifecycleState)))
}

func (c *vaultRuntimeClient) observeVaultAfterControlPlaneConflict(ctx context.Context, resource *keymanagementv1beta1.Vault) (servicemanager.OSOKResponse, error) {
	current, found, err := c.resolveVault(ctx, resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, normalizeVaultOCIError(err)
	}
	if !found {
		c.markVaultDeleted(resource, "OCI Vault no longer exists")
		clearVaultDeletionScheduleStatus(resource)
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	}

	c.syncVaultStatus(resource, current)
	return c.markVaultReconcileSuccess(resource, strings.ToUpper(string(current.LifecycleState)))
}

func (c *vaultRuntimeClient) confirmVaultDeleteAccepted(ctx context.Context, resource *keymanagementv1beta1.Vault) (bool, error) {
	current, found, err := c.resolveVault(ctx, resource)
	if err != nil {
		return false, normalizeVaultOCIError(err)
	}
	if !found {
		c.markVaultDeleted(resource, "OCI Vault no longer exists")
		clearVaultDeletionScheduleStatus(resource)
		return true, nil
	}

	c.syncVaultStatus(resource, current)
	switch lifecycleState := strings.ToUpper(string(current.LifecycleState)); lifecycleState {
	case "DELETED":
		c.markVaultDeleted(resource, "OCI Vault deleted")
		clearVaultDeletionScheduleStatus(resource)
		return true, nil
	case "PENDING_DELETION", "SCHEDULING_DELETION", "DELETING":
		c.markVaultCondition(resource, shared.Terminating, vaultLifecycleMessage(lifecycleState))
		return true, nil
	case "CANCELLING_DELETION":
		c.markVaultCondition(resource, shared.Terminating, vaultLifecycleMessage(lifecycleState))
		return false, nil
	default:
		return false, nil
	}
}

func (c *vaultRuntimeClient) resolveVault(ctx context.Context, resource *keymanagementv1beta1.Vault) (keymanagementsdk.Vault, bool, error) {
	if vaultID := currentVaultID(resource); vaultID != "" {
		current, found, err := c.getVault(ctx, vaultID)
		if err == nil || !vaultNotFound(err) {
			return current, found, err
		}
	}
	return c.lookupVaultBySpec(ctx, resource)
}

func (c *vaultRuntimeClient) getVault(ctx context.Context, vaultID string) (keymanagementsdk.Vault, bool, error) {
	response, err := c.sdk.GetVault(ctx, keymanagementsdk.GetVaultRequest{
		VaultId: common.String(vaultID),
	})
	if err != nil {
		if vaultNotFound(err) {
			return keymanagementsdk.Vault{}, false, nil
		}
		return keymanagementsdk.Vault{}, false, err
	}
	return response.Vault, true, nil
}

func (c *vaultRuntimeClient) lookupVaultBySpec(ctx context.Context, resource *keymanagementv1beta1.Vault) (keymanagementsdk.Vault, bool, error) {
	compartmentID := strings.TrimSpace(resource.Spec.CompartmentId)
	displayName := strings.TrimSpace(resource.Spec.DisplayName)
	vaultType := strings.TrimSpace(resource.Spec.VaultType)
	if compartmentID == "" || displayName == "" {
		return keymanagementsdk.Vault{}, false, nil
	}

	response, err := c.sdk.ListVaults(ctx, keymanagementsdk.ListVaultsRequest{
		CompartmentId: common.String(compartmentID),
	})
	if err != nil {
		return keymanagementsdk.Vault{}, false, err
	}

	var matchedIDs []string
	for _, item := range response.Items {
		if strings.TrimSpace(stringValue(item.DisplayName)) != displayName {
			continue
		}
		if vaultType != "" && strings.TrimSpace(string(item.VaultType)) != vaultType {
			continue
		}
		if id := strings.TrimSpace(stringValue(item.Id)); id != "" {
			matchedIDs = append(matchedIDs, id)
		}
	}

	switch len(matchedIDs) {
	case 0:
		return keymanagementsdk.Vault{}, false, nil
	case 1:
		return c.getVault(ctx, matchedIDs[0])
	default:
		return keymanagementsdk.Vault{}, false, fmt.Errorf("Vault list returned multiple matching resources for displayName %q", displayName)
	}
}

func effectiveVaultDeletionScheduleDays(days int32, allowDefault bool) (int32, error) {
	switch {
	case days == 0 && allowDefault:
		return defaultVaultDeletionScheduleDays, nil
	case days >= 7 && days <= 30:
		return days, nil
	case days == 0:
		return 0, nil
	default:
		return 0, fmt.Errorf("Vault deletionScheduleDays must be between 7 and 30")
	}
}

func sdkTimeForDeletionDays(days int32) *common.SDKTime {
	if days <= 0 {
		return nil
	}
	deletionTime := time.Now().UTC().AddDate(0, 0, int(days))
	return &common.SDKTime{Time: deletionTime}
}

func ensureVaultRequeue(response servicemanager.OSOKResponse) servicemanager.OSOKResponse {
	response.ShouldRequeue = true
	if response.RequeueDuration == 0 {
		response.RequeueDuration = vaultRequeueDuration
	}
	return response
}

func (c *vaultRuntimeClient) markVaultReconcileSuccess(resource *keymanagementv1beta1.Vault, lifecycleState string) (servicemanager.OSOKResponse, error) {
	conditionType, shouldRequeue := vaultConditionForLifecycle(lifecycleState)
	message := vaultLifecycleMessage(lifecycleState)
	c.markVaultCondition(resource, conditionType, message)
	return servicemanager.OSOKResponse{
		IsSuccessful:    conditionType != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: vaultRequeueDuration,
	}, nil
}

func vaultConditionForLifecycle(lifecycleState string) (shared.OSOKConditionType, bool) {
	switch lifecycleState {
	case "BACKUP_IN_PROGRESS", "CREATING", "RESTORING":
		return shared.Provisioning, true
	case "UPDATING":
		return shared.Updating, true
	case "ACTIVE":
		return shared.Active, false
	case "CANCELLING_DELETION", "DELETING", "PENDING_DELETION", "SCHEDULING_DELETION":
		return shared.Terminating, true
	default:
		return shared.Failed, false
	}
}

func vaultLifecycleMessage(lifecycleState string) string {
	switch lifecycleState {
	case "BACKUP_IN_PROGRESS", "CREATING", "RESTORING":
		return "OCI Vault provisioning is in progress"
	case "UPDATING":
		return "OCI Vault update is in progress"
	case "ACTIVE":
		return "OCI Vault is active"
	case "CANCELLING_DELETION":
		return "OCI Vault deletion cancellation is in progress"
	case "DELETING", "PENDING_DELETION", "SCHEDULING_DELETION":
		return "OCI Vault deletion is in progress"
	case "DELETED":
		return "OCI Vault deleted"
	default:
		return fmt.Sprintf("Vault lifecycle state %q is not modeled", lifecycleState)
	}
}

func (c *vaultRuntimeClient) syncVaultStatus(resource *keymanagementv1beta1.Vault, vault keymanagementsdk.Vault) {
	resource.Status.CompartmentId = stringValue(vault.CompartmentId)
	resource.Status.CryptoEndpoint = stringValue(vault.CryptoEndpoint)
	resource.Status.DisplayName = stringValue(vault.DisplayName)
	resource.Status.Id = stringValue(vault.Id)
	resource.Status.LifecycleState = string(vault.LifecycleState)
	resource.Status.ManagementEndpoint = stringValue(vault.ManagementEndpoint)
	resource.Status.TimeCreated = sdkTimeString(vault.TimeCreated)
	resource.Status.VaultType = string(vault.VaultType)
	resource.Status.WrappingkeyId = stringValue(vault.WrappingkeyId)
	resource.Status.DefinedTags = convertVaultDefinedTags(vault.DefinedTags)
	resource.Status.FreeformTags = cloneStringMap(vault.FreeformTags)
	resource.Status.TimeOfDeletion = sdkTimeString(vault.TimeOfDeletion)
	resource.Status.RestoredFromVaultId = stringValue(vault.RestoredFromVaultId)
	resource.Status.ReplicaDetails = convertVaultReplicaDetails(vault.ReplicaDetails)
	resource.Status.IsPrimary = boolValue(vault.IsPrimary)
	resource.Status.ExternalKeyManagerMetadataSummary = convertVaultExternalKeyManagerMetadataSummary(vault.ExternalKeyManagerMetadataSummary)
	if resource.Status.Id != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.Id)
	}
}

func (c *vaultRuntimeClient) markVaultDeleted(resource *keymanagementv1beta1.Vault, message string) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.UpdatedAt = &now
	status.DeletedAt = &now
	if status.Ocid == "" && resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
}

func (c *vaultRuntimeClient) markVaultCondition(resource *keymanagementv1beta1.Vault, conditionType shared.OSOKConditionType, message string) {
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.Message = message
	status.Reason = string(conditionType)
	status.UpdatedAt = &now
	if status.Ocid == "" && resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
	}
	if status.CreatedAt == nil && resource.Status.Id != "" {
		status.CreatedAt = &now
	}
	conditionStatus := v1.ConditionTrue
	if conditionType == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, conditionType, conditionStatus, "", message, c.log)
}

func cloneVaultForGeneratedRuntime(resource *keymanagementv1beta1.Vault) keymanagementv1beta1.Vault {
	cloned := *resource
	cloned.Spec = resource.Spec
	cloned.Status = resource.Status
	cloned.Spec.DeletionScheduleDays = 0
	return cloned
}

func currentVaultID(resource *keymanagementv1beta1.Vault) string {
	if id := strings.TrimSpace(resource.Status.Id); id != "" {
		return id
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.UTC().Format(time.RFC3339)
}

func clearVaultDeletionScheduleStatus(resource *keymanagementv1beta1.Vault) {
	resource.Status.RequestedDeletionScheduleDays = 0
	resource.Status.TimeOfDeletion = ""
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func convertVaultDefinedTags(values map[string]map[string]interface{}) map[string]shared.MapValue {
	if len(values) == 0 {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(values))
	for outerKey, inner := range values {
		innerConverted := make(shared.MapValue, len(inner))
		for key, value := range inner {
			innerConverted[key] = fmt.Sprint(value)
		}
		converted[outerKey] = innerConverted
	}
	return converted
}

func convertVaultReplicaDetails(details *keymanagementsdk.VaultReplicaDetails) keymanagementv1beta1.VaultReplicaDetails {
	if details == nil {
		return keymanagementv1beta1.VaultReplicaDetails{}
	}
	return keymanagementv1beta1.VaultReplicaDetails{
		ReplicationId: stringValue(details.ReplicationId),
	}
}

func convertVaultExternalKeyManagerMetadataSummary(summary *keymanagementsdk.ExternalKeyManagerMetadataSummary) keymanagementv1beta1.VaultExternalKeyManagerMetadataSummary {
	if summary == nil {
		return keymanagementv1beta1.VaultExternalKeyManagerMetadataSummary{}
	}
	converted := keymanagementv1beta1.VaultExternalKeyManagerMetadataSummary{
		ExternalVaultEndpointUrl: stringValue(summary.ExternalVaultEndpointUrl),
		PrivateEndpointId:        stringValue(summary.PrivateEndpointId),
		Vendor:                   stringValue(summary.Vendor),
	}
	if summary.OauthMetadataSummary != nil {
		converted.OauthMetadataSummary = keymanagementv1beta1.VaultExternalKeyManagerMetadataSummaryOauthMetadataSummary{
			IdcsAccountNameUrl: stringValue(summary.OauthMetadataSummary.IdcsAccountNameUrl),
			ClientAppId:        stringValue(summary.OauthMetadataSummary.ClientAppId),
		}
	}
	return converted
}

func normalizeVaultOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func vaultNotFound(err error) bool {
	if err == nil {
		return false
	}

	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		if serviceErr.GetHTTPStatusCode() == 404 {
			return true
		}
		switch serviceErr.GetCode() {
		case "NotFound", "NotAuthorizedOrNotFound":
			return true
		}
	}

	message := err.Error()
	return strings.Contains(message, "http status code: 404") ||
		strings.Contains(message, "NotFound") ||
		strings.Contains(message, "NotAuthorizedOrNotFound")
}

func vaultConflict(err error) bool {
	if err == nil {
		return false
	}

	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.GetHTTPStatusCode() == 409
	}

	var conflictErr errorutil.ConflictOciError
	if errors.As(err, &conflictErr) {
		return true
	}

	return strings.Contains(err.Error(), "http status code: 409")
}
