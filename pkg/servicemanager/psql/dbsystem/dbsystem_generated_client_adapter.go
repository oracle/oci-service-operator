/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	psqlsdk "github.com/oracle/oci-go-sdk/v65/psql"
	psqlv1beta1 "github.com/oracle/oci-service-operator/api/psql/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	dbSystemDefaultRequeueDuration = time.Minute
	dbSystemDeleteInProgress       = "DbSystem delete is in progress"
)

func init() {
	registerDbSystemRuntimeHooksMutator(applyDbSystemRuntimeHooks)
}

func applyDbSystemRuntimeHooks(manager *DbSystemServiceManager, hooks *DbSystemRuntimeHooks) {
	if hooks == nil {
		return
	}

	alignDbSystemRuntimeSemantics(hooks)
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(DbSystemServiceClient) DbSystemServiceClient {
		return newManualDbSystemServiceClient(manager)
	})
}

func alignDbSystemRuntimeSemantics(hooks *DbSystemRuntimeHooks) {
	if hooks == nil || hooks.Semantics == nil {
		return
	}

	hooks.Semantics.Lifecycle.ProvisioningStates = []string{"CREATING"}
	hooks.Semantics.Lifecycle.UpdatingStates = []string{"UPDATING"}
	hooks.Semantics.Lifecycle.ActiveStates = []string{"ACTIVE", "INACTIVE", "NEEDS_ATTENTION"}
}

func newManualDbSystemServiceClient(manager *DbSystemServiceManager) DbSystemServiceClient {
	client := manualDbSystemServiceClient{}
	if manager == nil {
		client.initErr = fmt.Errorf("initialize DbSystem OCI client: service manager is nil")
		return client
	}

	sdkClient, err := psqlsdk.NewPostgresqlClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		client.initErr = fmt.Errorf("initialize DbSystem OCI client: %w", err)
	}
	client.sdk = sdkClient
	client.log = manager.Log
	client.credentialClient = manager.CredentialClient
	return client
}

type dbSystemOCIClient interface {
	CreateDbSystem(context.Context, psqlsdk.CreateDbSystemRequest) (psqlsdk.CreateDbSystemResponse, error)
	GetDbSystem(context.Context, psqlsdk.GetDbSystemRequest) (psqlsdk.GetDbSystemResponse, error)
	ListDbSystems(context.Context, psqlsdk.ListDbSystemsRequest) (psqlsdk.ListDbSystemsResponse, error)
	UpdateDbSystem(context.Context, psqlsdk.UpdateDbSystemRequest) (psqlsdk.UpdateDbSystemResponse, error)
	DeleteDbSystem(context.Context, psqlsdk.DeleteDbSystemRequest) (psqlsdk.DeleteDbSystemResponse, error)
}

type manualDbSystemServiceClient struct {
	sdk              dbSystemOCIClient
	log              loggerutil.OSOKLogger
	credentialClient credhelper.CredentialClient
	initErr          error
}

var _ DbSystemServiceClient = manualDbSystemServiceClient{}

func (c manualDbSystemServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *psqlv1beta1.DbSystem,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, c.initErr)
	}
	if err := validateDbSystemSpec(resource.Spec); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
	}

	current, found, err := c.lookupExistingDbSystem(ctx, resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
	}
	if !found {
		createDetails, err := buildCreateDbSystemDetails(ctx, resource, c.credentialClient)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
		}

		response, err := c.sdk.CreateDbSystem(ctx, psqlsdk.CreateDbSystemRequest{
			CreateDbSystemDetails: createDetails,
		})
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
		}

		osokResponse, err := c.projectDbSystem(resource, response.DbSystem, shared.Provisioning)
		recordDbSystemResponseRequestID(resource, response)
		return osokResponse, err
	}

	switch strings.ToUpper(string(current.LifecycleState)) {
	case "CREATING", "UPDATING", "DELETING":
		return c.projectDbSystem(resource, current, shared.Active)
	}

	observed, err := observedDbSystemStatus(current)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
	}
	if err := validateImmutableDrift(resource.Spec, observed); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
	}
	if err := validateAdminSecretDrift(resource.Spec, resource.Status); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
	}

	updateDetails, needsUpdate, err := buildUpdateDbSystemDetails(resource.Spec, observed)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
	}
	if !needsUpdate {
		return c.projectDbSystem(resource, current, shared.Active)
	}

	dbSystemID := stringValue(current.Id)
	if dbSystemID == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, fmt.Errorf("DbSystem bind did not resolve an OCI identifier"))
	}
	updateResponse, err := c.sdk.UpdateDbSystem(ctx, psqlsdk.UpdateDbSystemRequest{
		DbSystemId:            common.String(dbSystemID),
		UpdateDbSystemDetails: updateDetails,
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
	}

	refreshed, found, err := c.getDbSystem(ctx, dbSystemID)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
	}
	if !found {
		return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, fmt.Errorf("DbSystem %q disappeared immediately after update", dbSystemID))
	}

	osokResponse, err := c.projectDbSystem(resource, refreshed, shared.Updating)
	recordDbSystemResponseRequestID(resource, updateResponse)
	return osokResponse, err
}

func (c manualDbSystemServiceClient) Delete(
	ctx context.Context,
	resource *psqlv1beta1.DbSystem,
) (bool, error) {
	if c.initErr != nil {
		return false, c.markFailure(resource, c.initErr)
	}

	current, found, err := c.lookupExistingDbSystem(ctx, resource)
	if err != nil {
		return false, c.markFailure(resource, err)
	}
	if !found {
		c.markDeleted(resource, "DbSystem no longer exists in OCI")
		return true, nil
	}
	switch strings.ToUpper(string(current.LifecycleState)) {
	case "DELETED":
		c.markDeleted(resource, "DbSystem deleted from OCI")
		return true, nil
	case "DELETING":
		return c.markDeleteInProgress(resource, current)
	}

	dbSystemID := stringValue(current.Id)
	if dbSystemID == "" {
		c.markDeleted(resource, "DbSystem identity was never established; assuming no OCI DbSystem remains")
		return true, nil
	}
	deleteResponse, err := c.sdk.DeleteDbSystem(ctx, psqlsdk.DeleteDbSystemRequest{
		DbSystemId: common.String(dbSystemID),
	})
	if err != nil {
		if isDbSystemNotFound(err) {
			c.markDeleted(resource, "DbSystem no longer exists in OCI")
			recordDbSystemErrorRequestID(resource, err)
			return true, nil
		}
		return false, c.markFailure(resource, err)
	}
	recordDbSystemResponseRequestID(resource, deleteResponse)

	refreshed, found, err := c.getDbSystem(ctx, dbSystemID)
	if err != nil {
		if isDbSystemNotFound(err) {
			c.markDeleted(resource, "DbSystem deleted from OCI")
			recordDbSystemErrorRequestID(resource, err)
			return true, nil
		}
		return false, c.markFailure(resource, err)
	}
	if !found || strings.EqualFold(string(refreshed.LifecycleState), "DELETED") {
		c.markDeleted(resource, "DbSystem deleted from OCI")
		recordDbSystemResponseRequestID(resource, deleteResponse)
		return true, nil
	}

	deleted, err := c.markDeleteInProgress(resource, refreshed)
	recordDbSystemResponseRequestID(resource, deleteResponse)
	return deleted, err
}

func (c manualDbSystemServiceClient) markDeleteInProgress(
	resource *psqlv1beta1.DbSystem,
	current psqlsdk.DbSystem,
) (bool, error) {
	if _, err := c.projectDbSystem(resource, current, shared.Terminating); err != nil {
		return false, c.markFailure(resource, err)
	}
	resource.Status.OsokStatus.Message = dbSystemDeleteInProgress
	resource.Status.OsokStatus.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(
		resource.Status.OsokStatus,
		shared.Terminating,
		v1.ConditionTrue,
		"",
		dbSystemDeleteInProgress,
		c.log,
	)
	return false, nil
}

func (c manualDbSystemServiceClient) lookupExistingDbSystem(
	ctx context.Context,
	resource *psqlv1beta1.DbSystem,
) (psqlsdk.DbSystem, bool, error) {
	if currentID := currentDbSystemID(resource); currentID != "" {
		current, found, err := c.getDbSystem(ctx, currentID)
		if err == nil && found {
			return current, true, nil
		}
		if err != nil && !isDbSystemNotFound(err) {
			return psqlsdk.DbSystem{}, false, err
		}
	}

	return c.lookupBindableDbSystem(ctx, resource.Spec)
}

func (c manualDbSystemServiceClient) getDbSystem(
	ctx context.Context,
	dbSystemID string,
) (psqlsdk.DbSystem, bool, error) {
	if strings.TrimSpace(dbSystemID) == "" {
		return psqlsdk.DbSystem{}, false, nil
	}

	response, err := c.sdk.GetDbSystem(ctx, psqlsdk.GetDbSystemRequest{
		DbSystemId: common.String(dbSystemID),
	})
	if err != nil {
		if isDbSystemNotFound(err) {
			return psqlsdk.DbSystem{}, false, err
		}
		return psqlsdk.DbSystem{}, false, err
	}

	return response.DbSystem, true, nil
}

func (c manualDbSystemServiceClient) lookupBindableDbSystem(
	ctx context.Context,
	spec psqlv1beta1.DbSystemSpec,
) (psqlsdk.DbSystem, bool, error) {
	request := psqlsdk.ListDbSystemsRequest{
		CompartmentId: common.String(spec.CompartmentId),
		DisplayName:   common.String(spec.DisplayName),
		Limit:         common.Int(100),
	}

	var matches []psqlsdk.DbSystemSummary
	for {
		response, err := c.sdk.ListDbSystems(ctx, request)
		if err != nil {
			return psqlsdk.DbSystem{}, false, err
		}

		for _, item := range response.Items {
			if !strings.EqualFold(stringValue(item.CompartmentId), spec.CompartmentId) {
				continue
			}
			if !strings.EqualFold(stringValue(item.DisplayName), spec.DisplayName) {
				continue
			}
			if !isBindableDbSystemState(item.LifecycleState) {
				continue
			}
			matches = append(matches, item)
		}

		if response.OpcNextPage == nil || strings.TrimSpace(stringValue(response.OpcNextPage)) == "" {
			break
		}
		request.Page = response.OpcNextPage
	}

	switch len(matches) {
	case 0:
		return psqlsdk.DbSystem{}, false, nil
	case 1:
		return c.getDbSystem(ctx, stringValue(matches[0].Id))
	default:
		return psqlsdk.DbSystem{}, false, fmt.Errorf("DbSystem list lookup returned multiple bind candidates for compartment %q and displayName %q", spec.CompartmentId, spec.DisplayName)
	}
}

func buildCreateDbSystemDetails(
	ctx context.Context,
	resource *psqlv1beta1.DbSystem,
	credentialClient credhelper.CredentialClient,
) (psqlsdk.CreateDbSystemDetails, error) {
	spec := resource.Spec
	var details psqlsdk.CreateDbSystemDetails
	if err := decodeViaJSON(spec, &details); err != nil {
		return psqlsdk.CreateDbSystemDetails{}, fmt.Errorf("build CreateDbSystemDetails: %w", err)
	}
	credentials, err := resolveCreateDbSystemCredentials(ctx, resource, credentialClient)
	if err != nil {
		return psqlsdk.CreateDbSystemDetails{}, err
	}
	if isZeroValue(spec.ManagementPolicy) {
		details.ManagementPolicy = nil
	}
	if credentials == nil {
		details.Credentials = nil
	} else {
		details.Credentials = credentials
	}
	if isZeroValue(spec.Source) {
		details.Source = nil
	}
	if len(spec.InstancesDetails) == 0 {
		details.InstancesDetails = nil
	}
	if len(spec.FreeformTags) == 0 {
		details.FreeformTags = nil
	}
	if len(spec.DefinedTags) == 0 {
		details.DefinedTags = nil
	}
	if strings.TrimSpace(spec.Description) == "" {
		details.Description = nil
	}
	if strings.TrimSpace(spec.ConfigId) == "" {
		details.ConfigId = nil
	}
	if spec.InstanceOcpuCount == 0 {
		details.InstanceOcpuCount = nil
	}
	if spec.InstanceMemorySizeInGBs == 0 {
		details.InstanceMemorySizeInGBs = nil
	}
	if spec.InstanceCount == 0 {
		details.InstanceCount = nil
	}

	return details, nil
}

func resolveCreateDbSystemCredentials(
	ctx context.Context,
	resource *psqlv1beta1.DbSystem,
	credentialClient credhelper.CredentialClient,
) (*psqlsdk.Credentials, error) {
	spec := resource.Spec
	username := strings.TrimSpace(spec.Credentials.Username)
	passwordType := strings.ToUpper(strings.TrimSpace(spec.Credentials.PasswordDetails.PasswordType))
	password := strings.TrimSpace(spec.Credentials.PasswordDetails.Password)
	secretID := strings.TrimSpace(spec.Credentials.PasswordDetails.SecretId)
	secretVersion := strings.TrimSpace(spec.Credentials.PasswordDetails.SecretVersion)

	if secretName := strings.TrimSpace(spec.AdminUsername.Secret.SecretName); secretName != "" {
		resolved, err := resolveDbSystemSecretValue(ctx, credentialClient, resource.Namespace, secretName, "username")
		if err != nil {
			return nil, err
		}
		username = resolved
	}
	if secretName := strings.TrimSpace(spec.AdminPassword.Secret.SecretName); secretName != "" {
		resolved, err := resolveDbSystemSecretValue(ctx, credentialClient, resource.Namespace, secretName, "password")
		if err != nil {
			return nil, err
		}
		passwordType = "PLAIN_TEXT"
		password = resolved
		secretID = ""
		secretVersion = ""
	}

	if username == "" && password == "" && secretID == "" && secretVersion == "" && passwordType == "" {
		return nil, fmt.Errorf("DbSystem create requires either admin secret references or spec.credentials")
	}
	if username == "" {
		return nil, fmt.Errorf("DbSystem create requires a non-empty username from adminUsername or credentials.username")
	}

	passwordDetails, err := buildDbSystemPasswordDetails(passwordType, password, secretID, secretVersion)
	if err != nil {
		return nil, err
	}
	if passwordDetails == nil {
		return nil, fmt.Errorf("DbSystem create requires password details from adminPassword or credentials.passwordDetails")
	}

	return &psqlsdk.Credentials{
		Username:        common.String(username),
		PasswordDetails: passwordDetails,
	}, nil
}

func buildDbSystemPasswordDetails(
	passwordType string,
	password string,
	secretID string,
	secretVersion string,
) (psqlsdk.PasswordDetails, error) {
	switch {
	case secretID != "" || secretVersion != "" || passwordType == "VAULT_SECRET":
		if secretID == "" || secretVersion == "" {
			return nil, fmt.Errorf("DbSystem Vault secret password details require both secretId and secretVersion")
		}
		return psqlsdk.VaultSecretPasswordDetails{
			SecretId:      common.String(secretID),
			SecretVersion: common.String(secretVersion),
		}, nil
	case password != "" || passwordType == "" || passwordType == "PLAIN_TEXT":
		if password == "" {
			return nil, fmt.Errorf("DbSystem plain-text password details require a non-empty password")
		}
		return psqlsdk.PlainTextPasswordDetails{
			Password: common.String(password),
		}, nil
	default:
		return nil, fmt.Errorf("DbSystem credentials.passwordDetails.passwordType %q is not supported", passwordType)
	}
}

func buildUpdateDbSystemDetails(
	spec psqlv1beta1.DbSystemSpec,
	observed psqlv1beta1.DbSystemStatus,
) (psqlsdk.UpdateDbSystemDetails, bool, error) {
	var details psqlsdk.UpdateDbSystemDetails
	needsUpdate := false

	if spec.DisplayName != observed.DisplayName {
		details.DisplayName = common.String(spec.DisplayName)
		needsUpdate = true
	}
	if spec.Description != observed.Description {
		details.Description = common.String(spec.Description)
		needsUpdate = true
	}
	if spec.StorageDetails.Iops != 0 && spec.StorageDetails.Iops != observed.StorageDetails.Iops {
		details.StorageDetails = &psqlsdk.UpdateStorageDetailsParams{
			Iops: common.Int64(spec.StorageDetails.Iops),
		}
		needsUpdate = true
	}
	if hasDbConfigurationParams(spec.DbConfigurationParams) &&
		(spec.DbConfigurationParams.ConfigId != observed.ConfigId || strings.TrimSpace(spec.DbConfigurationParams.ApplyConfig) != "") {
		updateParams := &psqlsdk.UpdateDbConfigParams{
			ConfigId: common.String(spec.DbConfigurationParams.ConfigId),
		}
		if applyConfig := strings.TrimSpace(spec.DbConfigurationParams.ApplyConfig); applyConfig != "" {
			updateParams.ApplyConfig = psqlsdk.UpdateDbConfigParamsApplyConfigEnum(applyConfig)
		}
		details.DbConfigurationParams = updateParams
		needsUpdate = true
	}
	if !isZeroValue(spec.ManagementPolicy) && !jsonEqual(spec.ManagementPolicy, observed.ManagementPolicy) {
		var policy psqlsdk.ManagementPolicyDetails
		if err := decodeViaJSON(spec.ManagementPolicy, &policy); err != nil {
			return psqlsdk.UpdateDbSystemDetails{}, false, fmt.Errorf("build ManagementPolicyDetails: %w", err)
		}
		details.ManagementPolicy = &policy
		needsUpdate = true
	}
	if !equalStringMaps(spec.FreeformTags, observed.FreeformTags) {
		details.FreeformTags = desiredFreeformTags(spec.FreeformTags)
		needsUpdate = true
	}
	if !equalDefinedTags(spec.DefinedTags, observed.DefinedTags) {
		details.DefinedTags = desiredDefinedTags(spec.DefinedTags)
		needsUpdate = true
	}

	return details, needsUpdate, nil
}

func validateDbSystemSpec(spec psqlv1beta1.DbSystemSpec) error {
	if spec.ConfigId != "" && spec.DbConfigurationParams.ConfigId != "" && spec.ConfigId != spec.DbConfigurationParams.ConfigId {
		return fmt.Errorf("DbSystem spec cannot set configId %q and dbConfigurationParams.configId %q to different values", spec.ConfigId, spec.DbConfigurationParams.ConfigId)
	}
	return nil
}

func validateImmutableDrift(
	spec psqlv1beta1.DbSystemSpec,
	observed psqlv1beta1.DbSystemStatus,
) error {
	checks := []struct {
		field string
		equal bool
	}{
		{field: "compartmentId", equal: spec.CompartmentId == observed.CompartmentId},
		{field: "dbVersion", equal: spec.DbVersion == observed.DbVersion},
		{field: "shape", equal: spec.Shape == observed.Shape},
		{field: "systemType", equal: spec.SystemType == "" || spec.SystemType == observed.SystemType},
		{field: "configId", equal: spec.ConfigId == "" || spec.ConfigId == observed.ConfigId},
		{field: "instanceOcpuCount", equal: spec.InstanceOcpuCount == 0 || spec.InstanceOcpuCount == observed.InstanceOcpuCount},
		{field: "instanceMemorySizeInGBs", equal: spec.InstanceMemorySizeInGBs == 0 || spec.InstanceMemorySizeInGBs == observed.InstanceMemorySizeInGBs},
		{field: "instanceCount", equal: spec.InstanceCount == 0 || spec.InstanceCount == observed.InstanceCount},
		{field: "networkDetails", equal: reflect.DeepEqual(spec.NetworkDetails, observed.NetworkDetails)},
		{field: "storageDetails.availabilityDomain", equal: spec.StorageDetails.AvailabilityDomain == observed.StorageDetails.AvailabilityDomain},
		{field: "storageDetails.isRegionallyDurable", equal: spec.StorageDetails.IsRegionallyDurable == observed.StorageDetails.IsRegionallyDurable},
		{field: "storageDetails.systemType", equal: spec.StorageDetails.SystemType == observed.StorageDetails.SystemType},
	}
	for _, check := range checks {
		if !check.equal {
			return fmt.Errorf("DbSystem update does not support changing %s after create or bind", check.field)
		}
	}

	if spec.Credentials.Username != "" && spec.Credentials.Username != observed.AdminUsername {
		return fmt.Errorf("DbSystem update does not support changing credentials.username after create or bind")
	}
	if !isZeroValue(spec.Source) && !jsonEqual(spec.Source, observed.Source) {
		return fmt.Errorf("DbSystem update does not support changing source after create or bind")
	}

	return nil
}

func validateAdminSecretDrift(
	spec psqlv1beta1.DbSystemSpec,
	current psqlv1beta1.DbSystemStatus,
) error {
	if hasSecretSourceName(current.AdminUsernameSource) && !reflect.DeepEqual(spec.AdminUsername, current.AdminUsernameSource) {
		return fmt.Errorf("DbSystem update does not support changing adminUsername secret reference after create or bind")
	}
	if hasSecretSourceName(current.AdminPasswordSource) && !reflect.DeepEqual(spec.AdminPassword, current.AdminPasswordSource) {
		return fmt.Errorf("DbSystem update does not support changing adminPassword secret reference after create or bind")
	}
	return nil
}

func observedDbSystemStatus(current psqlsdk.DbSystem) (psqlv1beta1.DbSystemStatus, error) {
	var status psqlv1beta1.DbSystemStatus
	if err := decodeViaJSON(current, &status); err != nil {
		return psqlv1beta1.DbSystemStatus{}, fmt.Errorf("project DbSystem into observed status: %w", err)
	}
	return status, nil
}

func (c manualDbSystemServiceClient) projectDbSystem(
	resource *psqlv1beta1.DbSystem,
	current psqlsdk.DbSystem,
	fallback shared.OSOKConditionType,
) (servicemanager.OSOKResponse, error) {
	if err := decodeViaJSON(current, &resource.Status); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("project OCI DbSystem into status: %w", err)
	}
	stampAdminSecretSourceStatus(resource)

	status := resource.Status.OsokStatus
	if currentID := stringValue(current.Id); currentID != "" {
		status.Ocid = shared.OCID(currentID)
	}

	condition, shouldRequeue, message := classifyDbSystemLifecycle(current, fallback)
	status.Message = message
	status.Reason = string(condition)
	now := metav1.Now()
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status = util.UpdateOSOKStatusCondition(status, condition, v1.ConditionTrue, "", message, c.log)
	resource.Status.OsokStatus = status

	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: dbSystemDefaultRequeueDuration,
	}, nil
}

func classifyDbSystemLifecycle(
	current psqlsdk.DbSystem,
	fallback shared.OSOKConditionType,
) (shared.OSOKConditionType, bool, string) {
	message := dbSystemLifecycleMessage(current, fallback)
	switch strings.ToUpper(string(current.LifecycleState)) {
	case "":
		return fallback, shouldRequeueForCondition(fallback), message
	case "CREATING":
		return shared.Provisioning, true, message
	case "UPDATING":
		return shared.Updating, true, message
	case "DELETING":
		return shared.Terminating, true, message
	case "ACTIVE", "INACTIVE", "NEEDS_ATTENTION":
		if fallback == shared.Updating {
			return shared.Updating, true, message
		}
		if fallback == shared.Terminating {
			return shared.Terminating, true, message
		}
		return shared.Active, false, message
	case "FAILED":
		return shared.Failed, false, message
	default:
		return shared.Failed, false, fmt.Sprintf("DbSystem lifecycle state %q is not modeled: %s", current.LifecycleState, message)
	}
}

func dbSystemLifecycleMessage(current psqlsdk.DbSystem, fallback shared.OSOKConditionType) string {
	if message := strings.TrimSpace(stringValue(current.LifecycleDetails)); message != "" {
		return message
	}
	if displayName := strings.TrimSpace(stringValue(current.DisplayName)); displayName != "" {
		return displayName
	}
	switch fallback {
	case shared.Provisioning:
		return "OCI resource provisioning is in progress"
	case shared.Updating:
		return "OCI resource update is in progress"
	case shared.Terminating:
		return dbSystemDeleteInProgress
	case shared.Failed:
		return "OCI resource reconcile failed"
	default:
		return "OCI resource is active"
	}
}

func (c manualDbSystemServiceClient) markFailure(resource *psqlv1beta1.DbSystem, err error) error {
	status := resource.Status.OsokStatus
	servicemanager.RecordErrorOpcRequestID(&status, err)
	now := metav1.Now()
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	status.UpdatedAt = &now
	status = util.UpdateOSOKStatusCondition(status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.log)
	resource.Status.OsokStatus = status
	return err
}

func recordDbSystemResponseRequestID(resource *psqlv1beta1.DbSystem, response any) {
	if resource == nil {
		return
	}
	servicemanager.RecordResponseOpcRequestID(&resource.Status.OsokStatus, response)
}

func recordDbSystemErrorRequestID(resource *psqlv1beta1.DbSystem, err error) {
	if resource == nil {
		return
	}
	servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
}

func (c manualDbSystemServiceClient) markDeleted(resource *psqlv1beta1.DbSystem, message string) {
	status := resource.Status.OsokStatus
	now := metav1.Now()
	status.Message = message
	status.Reason = string(shared.Terminating)
	status.UpdatedAt = &now
	status.DeletedAt = &now
	status = util.UpdateOSOKStatusCondition(status, shared.Terminating, v1.ConditionTrue, "", message, c.log)
	resource.Status.OsokStatus = status
}

func stampAdminSecretSourceStatus(resource *psqlv1beta1.DbSystem) {
	if resource == nil {
		return
	}
	if hasSecretSourceName(resource.Spec.AdminUsername) {
		resource.Status.AdminUsernameSource = resource.Spec.AdminUsername
	} else {
		resource.Status.AdminUsernameSource = shared.UsernameSource{}
	}
	if hasSecretSourceName(resource.Spec.AdminPassword) {
		resource.Status.AdminPasswordSource = resource.Spec.AdminPassword
	} else {
		resource.Status.AdminPasswordSource = shared.PasswordSource{}
	}
}

func currentDbSystemID(resource *psqlv1beta1.DbSystem) string {
	if resource.Status.OsokStatus.Ocid != "" {
		return string(resource.Status.OsokStatus.Ocid)
	}
	return strings.TrimSpace(resource.Status.Id)
}

func isBindableDbSystemState(state psqlsdk.DbSystemLifecycleStateEnum) bool {
	switch strings.ToUpper(string(state)) {
	case "ACTIVE", "CREATING", "UPDATING", "INACTIVE", "NEEDS_ATTENTION":
		return true
	default:
		return false
	}
}

func shouldRequeueForCondition(condition shared.OSOKConditionType) bool {
	return condition == shared.Provisioning || condition == shared.Updating || condition == shared.Terminating
}

func hasDbConfigurationParams(params psqlv1beta1.DbSystemDbConfigurationParams) bool {
	return strings.TrimSpace(params.ConfigId) != "" || strings.TrimSpace(params.ApplyConfig) != ""
}

func desiredFreeformTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(tags))
	for key, value := range tags {
		out[key] = value
	}
	return out
}

func desiredDefinedTags(tags map[string]shared.MapValue) map[string]map[string]interface{} {
	if len(tags) == 0 {
		return map[string]map[string]interface{}{}
	}
	return *util.ConvertToOciDefinedTags(&tags)
}

func decodeViaJSON(source any, target any) error {
	payload, err := json.Marshal(source)
	if err != nil {
		return fmt.Errorf("marshal source value: %w", err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("unmarshal target value: %w", err)
	}
	return nil
}

func jsonEqual(left any, right any) bool {
	normalize := func(value any) any {
		payload, err := json.Marshal(value)
		if err != nil {
			return nil
		}
		var decoded any
		if err := json.Unmarshal(payload, &decoded); err != nil {
			return nil
		}
		return decoded
	}
	return reflect.DeepEqual(normalize(left), normalize(right))
}

func isZeroValue(value any) bool {
	if value == nil {
		return true
	}
	return reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface())
}

func isDbSystemNotFound(err error) bool {
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

func resolveDbSystemSecretValue(
	ctx context.Context,
	credentialClient credhelper.CredentialClient,
	namespace string,
	secretName string,
	dataKey string,
) (string, error) {
	if credentialClient == nil {
		return "", fmt.Errorf("resolve %s secret %q: credential client is nil", dataKey, secretName)
	}
	if strings.TrimSpace(namespace) == "" {
		return "", fmt.Errorf("resolve %s secret %q: namespace is empty", dataKey, secretName)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	secretData, err := credentialClient.GetSecret(ctx, secretName, namespace)
	if err != nil {
		return "", fmt.Errorf("get %s secret %q: %w", dataKey, secretName, err)
	}
	rawValue, ok := secretData[dataKey]
	if !ok {
		return "", fmt.Errorf("%s key in secret %q is not found", dataKey, secretName)
	}
	return string(rawValue), nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func equalStringMaps(left map[string]string, right map[string]string) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}
	return reflect.DeepEqual(left, right)
}

func equalDefinedTags(left map[string]shared.MapValue, right map[string]shared.MapValue) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}
	return reflect.DeepEqual(left, right)
}

func hasSecretSourceName(source any) bool {
	value := reflect.ValueOf(source)
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return false
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return false
	}
	secretField := value.FieldByName("Secret")
	if !secretField.IsValid() {
		return false
	}
	for secretField.IsValid() && secretField.Kind() == reflect.Pointer {
		if secretField.IsNil() {
			return false
		}
		secretField = secretField.Elem()
	}
	if !secretField.IsValid() || secretField.Kind() != reflect.Struct {
		return false
	}
	nameField := secretField.FieldByName("SecretName")
	return nameField.IsValid() && nameField.Kind() == reflect.String && strings.TrimSpace(nameField.String()) != ""
}
