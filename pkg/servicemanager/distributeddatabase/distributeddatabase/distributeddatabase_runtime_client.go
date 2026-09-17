/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package distributeddatabase

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	distributeddatabasesdk "github.com/oracle/oci-go-sdk/v65/distributeddatabase"
	distributeddatabasev1beta1 "github.com/oracle/oci-service-operator/api/distributeddatabase/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
)

func init() {
	registerDistributedDatabaseRuntimeHooksMutator(func(manager *DistributedDatabaseServiceManager, hooks *DistributedDatabaseRuntimeHooks) {
		applyDistributedDatabaseRuntimeHooks(manager, hooks)
	})
}

func applyDistributedDatabaseRuntimeHooks(
	manager *DistributedDatabaseServiceManager,
	hooks *DistributedDatabaseRuntimeHooks,
) {
	if hooks == nil {
		return
	}

	var credentialClient credhelper.CredentialClient
	if manager != nil {
		credentialClient = manager.CredentialClient
	}

	hooks.Semantics = reviewedDistributedDatabaseRuntimeSemantics()
	hooks.Get.Fields = reviewedDistributedDatabaseGetFields()
	hooks.List.Fields = reviewedDistributedDatabaseListFields()
	hooks.BuildCreateBody = func(
		ctx context.Context,
		resource *distributeddatabasev1beta1.DistributedDatabase,
		namespace string,
	) (any, error) {
		return buildDistributedDatabaseCreateBody(ctx, credentialClient, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *distributeddatabasev1beta1.DistributedDatabase,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildDistributedDatabaseUpdateBody(resource, currentResponse)
	}
	hooks.ParityHooks.UnsupportedDriftEquivalent = distributedDatabaseUnsupportedDriftEquivalent
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedDistributedDatabaseIdentity
}

func distributedDatabaseUnsupportedDriftEquivalent(path string, desired, observed any) (bool, bool) {
	switch path {
	case "catalogDetails", "shardDetails":
		return true, distributedDatabaseDesiredSubsetMatches(path, desired, observed)
	default:
		return false, false
	}
}

func distributedDatabaseDesiredSubsetMatches(rootPath string, desired, observed any) bool {
	switch desiredValue := desired.(type) {
	case map[string]any:
		observedValue, ok := observed.(map[string]any)
		if !ok {
			return false
		}
		for key, desiredChild := range desiredValue {
			if distributedDatabaseUnobservableCreateField(rootPath, key) {
				continue
			}
			observedChild, found := distributedDatabaseMapValue(observedValue, key)
			if !found || !distributedDatabaseDesiredSubsetMatches(rootPath, desiredChild, observedChild) {
				return false
			}
		}
		return true
	case []any:
		observedValue, ok := observed.([]any)
		if !ok || len(desiredValue) != len(observedValue) {
			return false
		}
		for index := range desiredValue {
			if !distributedDatabaseDesiredSubsetMatches(rootPath, desiredValue[index], observedValue[index]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(desired, observed)
	}
}

func distributedDatabaseUnobservableCreateField(rootPath, field string) bool {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "adminpassword", "peervmclusterids":
		return true
	case "shardspace":
		return rootPath == "catalogDetails"
	default:
		return false
	}
}

func distributedDatabaseMapValue(values map[string]any, key string) (any, bool) {
	if value, ok := values[key]; ok {
		return value, true
	}
	normalized := strings.ToLower(strings.TrimSpace(key))
	for candidate, value := range values {
		if strings.ToLower(strings.TrimSpace(candidate)) == normalized {
			return value, true
		}
	}
	return nil, false
}

func reviewedDistributedDatabaseRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newDistributedDatabaseRuntimeSemantics()
	semantics.Lifecycle = generatedruntime.LifecycleSemantics{
		ProvisioningStates: []string{"CREATING"},
		UpdatingStates:     []string{"UPDATING"},
		ActiveStates:       []string{"ACTIVE", "INACTIVE"},
	}
	semantics.Delete = generatedruntime.DeleteSemantics{
		Policy:         "required",
		PendingStates:  []string{"DELETING"},
		TerminalStates: []string{"DELETED", "NOT_FOUND"},
	}
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName", "prefix", "dbDeploymentType", "lifecycleState"},
	}
	semantics.Mutation = generatedruntime.MutationSemantics{
		Mutable:       []string{"definedTags", "displayName", "freeformTags"},
		ForceNew:      []string{"compartmentId"},
		ConflictsWith: map[string][]string{},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func reviewedDistributedDatabaseGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "DistributedDatabaseId", RequestName: "distributedDatabaseId", Contribution: "path", PreferResourceID: true},
	}
}

func reviewedDistributedDatabaseListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "DbDeploymentType", RequestName: "dbDeploymentType", Contribution: "query"},
	}
}

func buildDistributedDatabaseCreateBody(
	ctx context.Context,
	credentialClient credhelper.CredentialClient,
	resource *distributeddatabasev1beta1.DistributedDatabase,
	namespace string,
) (distributeddatabasesdk.CreateDistributedDatabaseDetails, error) {
	if resource == nil {
		return distributeddatabasesdk.CreateDistributedDatabaseDetails{}, fmt.Errorf("DistributedDatabase resource is nil")
	}

	resolvedSpec, err := generatedruntime.ResolveSpecValueWithBoolFields(resource, ctx, credentialClient, namespace)
	if err != nil {
		return distributeddatabasesdk.CreateDistributedDatabaseDetails{}, err
	}

	createMap, ok := resolvedSpec.(map[string]any)
	if !ok {
		return distributeddatabasesdk.CreateDistributedDatabaseDetails{}, fmt.Errorf("resolved DistributedDatabase spec is %T, want map[string]any", resolvedSpec)
	}
	if err := normalizeDistributedDatabaseCreateBody(createMap); err != nil {
		return distributeddatabasesdk.CreateDistributedDatabaseDetails{}, err
	}

	payload, err := json.Marshal(createMap)
	if err != nil {
		return distributeddatabasesdk.CreateDistributedDatabaseDetails{}, fmt.Errorf("marshal resolved DistributedDatabase create body: %w", err)
	}

	var details distributeddatabasesdk.CreateDistributedDatabaseDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return distributeddatabasesdk.CreateDistributedDatabaseDetails{}, fmt.Errorf("decode DistributedDatabase create request body: %w", err)
	}
	normalizeDistributedDatabaseCreateDetails(resource.Spec, &details)

	return details, nil
}

func normalizeDistributedDatabaseCreateBody(body map[string]any) error {
	if err := normalizeDistributedDatabaseCreateSource(body, "shardDetails"); err != nil {
		return err
	}
	if err := normalizeDistributedDatabaseCreateSource(body, "catalogDetails"); err != nil {
		return err
	}
	return nil
}

func normalizeDistributedDatabaseCreateSource(body map[string]any, field string) error {
	rawItems, ok := body[field]
	if !ok || rawItems == nil {
		return nil
	}

	items, ok := rawItems.([]any)
	if !ok {
		return fmt.Errorf("resolved DistributedDatabase %s is %T, want []any", field, rawItems)
	}

	for index, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return fmt.Errorf("resolved DistributedDatabase %s[%d] is %T, want map[string]any", field, index, rawItem)
		}
		delete(item, "jsonData")

		source, err := normalizedDistributedDatabaseSource(item["source"])
		if err != nil {
			return fmt.Errorf("normalize DistributedDatabase %s[%d] source: %w", field, index, err)
		}
		item["source"] = source
	}

	return nil
}

func normalizedDistributedDatabaseSource(raw any) (string, error) {
	source, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("expected source string, got %T", raw)
	}

	switch normalized := strings.ToUpper(strings.TrimSpace(source)); normalized {
	case "NEW_VAULT_AND_CLUSTER", "EXADB_XS":
		return normalized, nil
	case "EXISTING_CLUSTER":
		// The provider treats EXISTING_CLUSTER as the EXADB_XS create family.
		return "EXADB_XS", nil
	case "":
		return "", fmt.Errorf("source is required")
	default:
		return "", fmt.Errorf("unsupported source %q", source)
	}
}

func normalizeDistributedDatabaseCreateDetails(
	spec distributeddatabasev1beta1.DistributedDatabaseSpec,
	details *distributeddatabasesdk.CreateDistributedDatabaseDetails,
) {
	if details == nil {
		return
	}

	if reflect.ValueOf(spec.DbBackupConfig).IsZero() {
		details.DbBackupConfig = nil
	} else {
		applyDistributedDatabaseBackupConfigBoolPointers(spec.DbBackupConfig, details.DbBackupConfig)
	}

	for index := range spec.ShardDetails {
		if index >= len(details.ShardDetails) {
			break
		}
		current, ok := details.ShardDetails[index].(distributeddatabasesdk.CreateDistributedDatabaseShardWithExadbXsNewVaultAndClusterDetails)
		if !ok {
			continue
		}
		applyDistributedDatabaseVmClusterBoolPointers(
			spec.ShardDetails[index].VmClusterDetails.IsDiagnosticsEventsEnabled,
			spec.ShardDetails[index].VmClusterDetails.IsHealthMonitoringEnabled,
			spec.ShardDetails[index].VmClusterDetails.IsIncidentLogsEnabled,
			current.VmClusterDetails,
		)
		applyDistributedDatabaseShardPeerBoolPointers(spec.ShardDetails[index].PeerDetails, current.PeerDetails)
		details.ShardDetails[index] = current
	}

	for index := range spec.CatalogDetails {
		if index >= len(details.CatalogDetails) {
			break
		}
		current, ok := details.CatalogDetails[index].(distributeddatabasesdk.CreateDistributedDatabaseCatalogWithExadbXsNewVaultAndClusterDetails)
		if !ok {
			continue
		}
		applyDistributedDatabaseVmClusterBoolPointers(
			spec.CatalogDetails[index].VmClusterDetails.IsDiagnosticsEventsEnabled,
			spec.CatalogDetails[index].VmClusterDetails.IsHealthMonitoringEnabled,
			spec.CatalogDetails[index].VmClusterDetails.IsIncidentLogsEnabled,
			current.VmClusterDetails,
		)
		applyDistributedDatabaseCatalogPeerBoolPointers(spec.CatalogDetails[index].PeerDetails, current.PeerDetails)
		details.CatalogDetails[index] = current
	}
}

func applyDistributedDatabaseVmClusterBoolPointers(
	isDiagnosticsEventsEnabled bool,
	isHealthMonitoringEnabled bool,
	isIncidentLogsEnabled bool,
	details *distributeddatabasesdk.VmClusterDetails,
) {
	if details == nil {
		return
	}
	details.IsDiagnosticsEventsEnabled = common.Bool(isDiagnosticsEventsEnabled)
	details.IsHealthMonitoringEnabled = common.Bool(isHealthMonitoringEnabled)
	details.IsIncidentLogsEnabled = common.Bool(isIncidentLogsEnabled)
}

func applyDistributedDatabaseShardPeerBoolPointers(
	specPeers []distributeddatabasev1beta1.DistributedDatabaseShardDetailPeerDetail,
	details []distributeddatabasesdk.CreateShardPeerWithExadbXsNewVaultAndClusterDetails,
) {
	for index := range specPeers {
		if index >= len(details) {
			break
		}
		applyDistributedDatabaseVmClusterBoolPointers(
			specPeers[index].VmClusterDetails.IsDiagnosticsEventsEnabled,
			specPeers[index].VmClusterDetails.IsHealthMonitoringEnabled,
			specPeers[index].VmClusterDetails.IsIncidentLogsEnabled,
			details[index].VmClusterDetails,
		)
	}
}

func applyDistributedDatabaseCatalogPeerBoolPointers(
	specPeers []distributeddatabasev1beta1.DistributedDatabaseCatalogDetailPeerDetail,
	details []distributeddatabasesdk.CreateCatalogPeerWithExadbXsNewVaultAndClusterDetails,
) {
	for index := range specPeers {
		if index >= len(details) {
			break
		}
		applyDistributedDatabaseVmClusterBoolPointers(
			specPeers[index].VmClusterDetails.IsDiagnosticsEventsEnabled,
			specPeers[index].VmClusterDetails.IsHealthMonitoringEnabled,
			specPeers[index].VmClusterDetails.IsIncidentLogsEnabled,
			details[index].VmClusterDetails,
		)
	}
}

func applyDistributedDatabaseBackupConfigBoolPointers(
	spec distributeddatabasev1beta1.DistributedDatabaseDbBackupConfig,
	details *distributeddatabasesdk.DistributedDbBackupConfig,
) {
	if details == nil {
		return
	}
	details.IsAutoBackupEnabled = common.Bool(spec.IsAutoBackupEnabled)
	details.CanRunImmediateFullBackup = common.Bool(spec.CanRunImmediateFullBackup)
	details.IsRemoteBackupEnabled = common.Bool(spec.IsRemoteBackupEnabled)

	for index := range spec.BackupDestinationDetails {
		if index >= len(details.BackupDestinationDetails) {
			break
		}
		details.BackupDestinationDetails[index].IsZeroDataLossEnabled = common.Bool(spec.BackupDestinationDetails[index].IsZeroDataLossEnabled)
		details.BackupDestinationDetails[index].IsRemote = common.Bool(spec.BackupDestinationDetails[index].IsRemote)
	}
}

func buildDistributedDatabaseUpdateBody(
	resource *distributeddatabasev1beta1.DistributedDatabase,
	currentResponse any,
) (distributeddatabasesdk.UpdateDistributedDatabaseDetails, bool, error) {
	if resource == nil {
		return distributeddatabasesdk.UpdateDistributedDatabaseDetails{}, false, fmt.Errorf("DistributedDatabase resource is nil")
	}

	current, ok := distributedDatabaseFromResponse(currentResponse)
	if !ok {
		return distributeddatabasesdk.UpdateDistributedDatabaseDetails{}, false, fmt.Errorf("current DistributedDatabase response does not expose a DistributedDatabase body")
	}

	updateDetails := distributeddatabasesdk.UpdateDistributedDatabaseDetails{}
	updateNeeded := false

	if desired, ok := desiredDistributedDatabaseStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		updateDetails.DisplayName = desired
		updateNeeded = true
	}

	desiredFreeformTags := desiredDistributedDatabaseFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}

	desiredDefinedTags := desiredDistributedDatabaseDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}

	if !updateNeeded {
		return distributeddatabasesdk.UpdateDistributedDatabaseDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func distributedDatabaseFromResponse(response any) (distributeddatabasesdk.DistributedDatabase, bool) {
	switch current := response.(type) {
	case distributeddatabasesdk.CreateDistributedDatabaseResponse:
		return current.DistributedDatabase, true
	case *distributeddatabasesdk.CreateDistributedDatabaseResponse:
		if current == nil {
			return distributeddatabasesdk.DistributedDatabase{}, false
		}
		return current.DistributedDatabase, true
	case distributeddatabasesdk.GetDistributedDatabaseResponse:
		return current.DistributedDatabase, true
	case *distributeddatabasesdk.GetDistributedDatabaseResponse:
		if current == nil {
			return distributeddatabasesdk.DistributedDatabase{}, false
		}
		return current.DistributedDatabase, true
	case distributeddatabasesdk.UpdateDistributedDatabaseResponse:
		return current.DistributedDatabase, true
	case *distributeddatabasesdk.UpdateDistributedDatabaseResponse:
		if current == nil {
			return distributeddatabasesdk.DistributedDatabase{}, false
		}
		return current.DistributedDatabase, true
	case distributeddatabasesdk.DistributedDatabase:
		return current, true
	case *distributeddatabasesdk.DistributedDatabase:
		if current == nil {
			return distributeddatabasesdk.DistributedDatabase{}, false
		}
		return *current, true
	case distributeddatabasesdk.DistributedDatabaseSummary:
		return distributedDatabaseFromSummary(current), true
	case *distributeddatabasesdk.DistributedDatabaseSummary:
		if current == nil {
			return distributeddatabasesdk.DistributedDatabase{}, false
		}
		return distributedDatabaseFromSummary(*current), true
	default:
		return distributeddatabasesdk.DistributedDatabase{}, false
	}
}

func distributedDatabaseFromSummary(
	summary distributeddatabasesdk.DistributedDatabaseSummary,
) distributeddatabasesdk.DistributedDatabase {
	return distributeddatabasesdk.DistributedDatabase{
		Id:                 summary.Id,
		CompartmentId:      summary.CompartmentId,
		DisplayName:        summary.DisplayName,
		TimeCreated:        summary.TimeCreated,
		TimeUpdated:        summary.TimeUpdated,
		DatabaseVersion:    summary.DatabaseVersion,
		LifecycleState:     summary.LifecycleState,
		LifecycleDetails:   summary.LifecycleDetails,
		Prefix:             summary.Prefix,
		PrivateEndpointIds: append([]string(nil), summary.PrivateEndpointIds...),
		ShardingMethod:     summary.ShardingMethod,
		CharacterSet:       summary.CharacterSet,
		NcharacterSet:      summary.NcharacterSet,
		ListenerPort:       summary.ListenerPort,
		OnsPortLocal:       summary.OnsPortLocal,
		OnsPortRemote:      summary.OnsPortRemote,
		DbDeploymentType:   distributeddatabasesdk.DistributedDatabaseDbDeploymentTypeEnum(summary.DbDeploymentType),
		ConnectionStrings:  summary.ConnectionStrings,
		Chunks:             summary.Chunks,
		ListenerPortTls:    summary.ListenerPortTls,
		ReplicationMethod:  summary.ReplicationMethod,
		ReplicationFactor:  summary.ReplicationFactor,
		ReplicationUnit:    summary.ReplicationUnit,
		Metadata:           summary.Metadata,
		FreeformTags:       cloneDistributedDatabaseStringMap(summary.FreeformTags),
		DefinedTags:        cloneDistributedDatabaseDefinedTags(summary.DefinedTags),
		SystemTags:         cloneDistributedDatabaseDefinedTags(summary.SystemTags),
	}
}

func clearTrackedDistributedDatabaseIdentity(resource *distributeddatabasev1beta1.DistributedDatabase) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus = shared.OSOKStatus{}
}

func desiredDistributedDatabaseStringUpdate(spec string, current *string) (*string, bool) {
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

func desiredDistributedDatabaseFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneDistributedDatabaseStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredDistributedDatabaseDefinedTagsForUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) map[string]map[string]interface{} {
	if spec != nil {
		return *util.ConvertToOciDefinedTags(&spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func cloneDistributedDatabaseStringMap(spec map[string]string) map[string]string {
	if spec == nil {
		return nil
	}
	out := make(map[string]string, len(spec))
	for key, value := range spec {
		out[key] = value
	}
	return out
}

func cloneDistributedDatabaseDefinedTags(spec map[string]map[string]interface{}) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}
	out := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		if values == nil {
			out[namespace] = nil
			continue
		}
		clonedValues := make(map[string]interface{}, len(values))
		for key, value := range values {
			clonedValues[key] = value
		}
		out[namespace] = clonedValues
	}
	return out
}
