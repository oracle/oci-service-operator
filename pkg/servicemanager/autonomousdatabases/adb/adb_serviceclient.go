/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/database"
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	"reflect"
)

var additionalCreatePayloadExcludedFields = map[string]struct{}{
	"adminPassword":        {},
	"compartmentId":        {},
	"computeCount":         {},
	"computeModel":         {},
	"cpuCoreCount":         {},
	"dataStorageSizeInTBs": {},
	"dbName":               {},
	"dbVersion":            {},
	"dbWorkload":           {},
	"definedTags":          {},
	"displayName":          {},
	"freeformTags":         {},
	"id":                   {},
	"isAutoScalingEnabled": {},
	"isDedicated":          {},
	"isFreeTier":           {},
	"licenseModel":         {},
	"wallet":               {},
}

var additionalUpdatePayloadExcludedFields = map[string]struct{}{
	"adminPassword":                     {},
	"autonomousContainerDatabaseId":     {},
	"autonomousMaintenanceScheduleType": {},
	"characterSet":                      {},
	"compartmentId":                     {},
	"cpuCoreCount":                      {},
	"dataStorageSizeInTBs":              {},
	"dbName":                            {},
	"dbVersion":                         {},
	"dbWorkload":                        {},
	"definedTags":                       {},
	"displayName":                       {},
	"freeformTags":                      {},
	"id":                                {},
	"isAutoScalingEnabled":              {},
	"isDedicated":                       {},
	"isFreeTier":                        {},
	"isPreviewVersionWithServiceTermsAccepted": {},
	"kmsKeyId":            {},
	"licenseModel":        {},
	"ncharacterSet":       {},
	"secretId":            {},
	"secretVersionNumber": {},
	"vaultId":             {},
	"wallet":              {},
}

type AdbServiceClient interface {
	CreateAdb(ctx context.Context, adb databasev1beta1.AutonomousDatabases) (database.AutonomousDatabase, error)

	UpdateAdb(ctx context.Context, request database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error)

	GetAdb(ctx context.Context, request database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error)

	DeleteAdb() (string, error)

	servicemanager.OSOKServiceManager
}

// DatabaseClientInterface defines the OCI operations used by AdbServiceManager.
type DatabaseClientInterface interface {
	CreateAutonomousDatabase(ctx context.Context, request database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error)
	ListAutonomousDatabases(ctx context.Context, request database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error)
	GetAutonomousDatabase(ctx context.Context, request database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error)
	UpdateAutonomousDatabase(ctx context.Context, request database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error)
}

func getDbClient(provider common.ConfigurationProvider) database.DatabaseClient {
	dbClient, _ := database.NewDatabaseClientWithConfigurationProvider(provider)
	return dbClient
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *AdbServiceManager) getOCIClient() DatabaseClientInterface {
	if c.ociClient != nil {
		return c.ociClient
	}
	return getDbClient(c.Provider)
}

func (c *AdbServiceManager) CreateAdb(ctx context.Context, adb databasev1beta1.AutonomousDatabases, adminPwd string) (database.CreateAutonomousDatabaseResponse, error) {

	dbClient := c.getOCIClient()

	c.Log.DebugLog("Creating Autonomous Database ", "name", adb.Spec.DisplayName)

	createAutonomousDatabaseDetails := database.CreateAutonomousDatabaseDetails{
		CompartmentId:        common.String(string(adb.Spec.CompartmentId)),
		DisplayName:          common.String(adb.Spec.DisplayName),
		DbName:               common.String(adb.Spec.DbName),
		DataStorageSizeInTBs: common.Int(adb.Spec.DataStorageSizeInTBs),
		IsAutoScalingEnabled: common.Bool(adb.Spec.IsAutoScalingEnabled),
		IsDedicated:          common.Bool(adb.Spec.IsDedicated),
		DbWorkload:           database.CreateAutonomousDatabaseBaseDbWorkloadEnum(adb.Spec.DbWorkload),
		IsFreeTier:           common.Bool(adb.Spec.IsFreeTier),
		FreeformTags:         adb.Spec.FreeFormTags,
		DefinedTags:          *util.ConvertToOciDefinedTags(&adb.Spec.DefinedTags),
	}

	if adb.Spec.ComputeModel != "" {
		createAutonomousDatabaseDetails.ComputeModel = database.CreateAutonomousDatabaseBaseComputeModelEnum(adb.Spec.ComputeModel)
		createAutonomousDatabaseDetails.ComputeCount = common.Float32(adb.Spec.ComputeCount)
	} else {
		createAutonomousDatabaseDetails.CpuCoreCount = common.Int(adb.Spec.CpuCoreCount)
	}

	if adb.Spec.DbVersion != "" {
		createAutonomousDatabaseDetails.DbVersion = common.String(adb.Spec.DbVersion)
	}

	if adb.Spec.LicenseModel != "" {
		createAutonomousDatabaseDetails.LicenseModel = database.CreateAutonomousDatabaseBaseLicenseModelEnum(adb.Spec.LicenseModel)
	}

	if adminPwd != "" {
		createAutonomousDatabaseDetails.AdminPassword = common.String(adminPwd)
	}

	additionalPayload, err := buildAdditionalSpecPayload(adb.Spec, additionalCreatePayloadExcludedFields)
	if err != nil {
		return database.CreateAutonomousDatabaseResponse{}, err
	}
	if err := decodeAdditionalSpecPayload(additionalPayload, &createAutonomousDatabaseDetails); err != nil {
		return database.CreateAutonomousDatabaseResponse{}, err
	}

	createAutonomousDatabaseRequest := database.CreateAutonomousDatabaseRequest{
		CreateAutonomousDatabaseDetails: createAutonomousDatabaseDetails,
	}

	return dbClient.CreateAutonomousDatabase(ctx, createAutonomousDatabaseRequest)
}

func (c *AdbServiceManager) GetAdbOcid(ctx context.Context, adb databasev1beta1.AutonomousDatabases) (*shared.OCID, error) {
	dbClient := c.getOCIClient()

	// List ADBs based on compartmentId and displayName and lifecycle-state as Active
	listAdbRequest := database.ListAutonomousDatabasesRequest{
		CompartmentId: common.String(string(adb.Spec.CompartmentId)),
		DisplayName:   common.String(adb.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	listAdbResponse, err := dbClient.ListAutonomousDatabases(ctx, listAdbRequest)
	if err != nil {
		c.Log.ErrorLog(err, "Error while listing Autonomous Database")
		return nil, err
	}

	if len(listAdbResponse.Items) > 0 {
		status := listAdbResponse.Items[0].LifecycleState
		if status == "AVAILABLE" || status == "PROVISIONING" {

			c.Log.DebugLog(fmt.Sprintf("Autonomous Database %s exists.", adb.Spec.DisplayName))

			return (*shared.OCID)(listAdbResponse.Items[0].Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("Autonomous Database %s does not exist.", adb.Spec.DisplayName))
	return nil, nil
}

func (c *AdbServiceManager) DeleteAdb() (string, error) {
	return "", nil
}

// Sync the Autonomous Database details
func (c *AdbServiceManager) GetAdb(ctx context.Context, adbId shared.OCID, retryPolicy *common.RetryPolicy) (*database.AutonomousDatabase, error) {

	dbClient := c.getOCIClient()

	getAutonomousDatabaseRequest := database.GetAutonomousDatabaseRequest{
		AutonomousDatabaseId: common.String(string(adbId)),
	}

	if retryPolicy != nil {
		getAutonomousDatabaseRequest.RequestMetadata.RetryPolicy = retryPolicy
	}

	response, err := dbClient.GetAutonomousDatabase(ctx, getAutonomousDatabaseRequest)
	if err != nil {
		return nil, err
	}

	return &response.AutonomousDatabase, nil
}

func (c *AdbServiceManager) UpdateAdb(ctx context.Context, adb *databasev1beta1.AutonomousDatabases) error {

	dbClient := c.getOCIClient()

	existingAdb, err := c.GetAdb(ctx, adb.Spec.AdbId, nil)
	if err != nil {
		return err
	}

	updateAutonomousDatabaseDetails := database.UpdateAutonomousDatabaseDetails{}

	updateNeeded := false
	if adb.Spec.DisplayName != "" && *existingAdb.DisplayName != adb.Spec.DisplayName {
		updateAutonomousDatabaseDetails.DisplayName = common.String(adb.Spec.DisplayName)
		updateNeeded = true
	}

	if adb.Spec.DbName != "" && adb.Spec.DbName != *existingAdb.DbName {
		updateAutonomousDatabaseDetails.DbName = common.String(adb.Spec.DbName)
	}

	if adb.Spec.DbWorkload != "" && string(existingAdb.DbWorkload) != adb.Spec.DbWorkload {
		updateAutonomousDatabaseDetails.DbWorkload = database.UpdateAutonomousDatabaseDetailsDbWorkloadEnum(
			adb.Spec.DbWorkload)
		updateNeeded = true
	}

	if adb.Spec.DbVersion != "" && adb.Spec.DbVersion != *existingAdb.DbVersion {
		updateAutonomousDatabaseDetails.DbVersion = common.String(adb.Spec.DbVersion)
		updateNeeded = true
	}

	if adb.Spec.DataStorageSizeInTBs != 0 && adb.Spec.DataStorageSizeInTBs != *existingAdb.DataStorageSizeInTBs {
		updateAutonomousDatabaseDetails.DataStorageSizeInTBs = common.Int(adb.Spec.DataStorageSizeInTBs)
		updateNeeded = true
	}

	if adb.Spec.CpuCoreCount != 0 && adb.Spec.CpuCoreCount != *existingAdb.CpuCoreCount {
		updateAutonomousDatabaseDetails.CpuCoreCount = common.Int(adb.Spec.CpuCoreCount)
		updateNeeded = true
	}

	if adb.Spec.IsAutoScalingEnabled != false && adb.Spec.IsAutoScalingEnabled != *existingAdb.IsAutoScalingEnabled {
		updateAutonomousDatabaseDetails.IsAutoScalingEnabled = common.Bool(adb.Spec.IsAutoScalingEnabled)
		updateNeeded = true
	}

	if adb.Spec.IsFreeTier != false && adb.Spec.IsFreeTier != *existingAdb.IsFreeTier {
		updateAutonomousDatabaseDetails.IsFreeTier = common.Bool(adb.Spec.IsFreeTier)
		updateNeeded = true
	}

	if adb.Spec.LicenseModel != "" && string(existingAdb.LicenseModel) != adb.Spec.LicenseModel {
		updateAutonomousDatabaseDetails.LicenseModel = database.UpdateAutonomousDatabaseDetailsLicenseModelEnum(adb.Spec.LicenseModel)
		updateNeeded = true
	}

	if adb.Spec.FreeFormTags != nil && !reflect.DeepEqual(existingAdb.FreeformTags, adb.Spec.FreeFormTags) {
		updateAutonomousDatabaseDetails.FreeformTags = adb.Spec.FreeFormTags
		updateNeeded = true
	}

	if adb.Spec.DefinedTags != nil {
		if defTag := *util.ConvertToOciDefinedTags(&adb.Spec.DefinedTags); !reflect.DeepEqual(existingAdb.DefinedTags, defTag) {
			updateAutonomousDatabaseDetails.DefinedTags = defTag
			updateNeeded = true
		}
	}

	if secretReferenceUpdateNeeded(adb.Spec, adb.Status) {
		secretID, _, _ := desiredAutonomousDatabaseSecretReference(adb.Spec)
		updateAutonomousDatabaseDetails.SecretId = common.String(secretID)
		if adb.Spec.SecretVersionNumber != 0 {
			updateAutonomousDatabaseDetails.SecretVersionNumber = common.Int(adb.Spec.SecretVersionNumber)
		}
		updateNeeded = true
	}

	additionalPayload, err := buildAdditionalSpecPayload(adb.Spec, additionalUpdatePayloadExcludedFields)
	if err != nil {
		return err
	}
	if len(additionalPayload) > 0 {
		additionalNeeded, err := additionalAutonomousDatabaseUpdateNeeded(additionalPayload, *existingAdb)
		if err != nil {
			return err
		}
		if additionalNeeded {
			if err := decodeAdditionalSpecPayload(additionalPayload, &updateAutonomousDatabaseDetails); err != nil {
				return err
			}
			updateNeeded = true
		}
	}

	if updateNeeded {
		updateAutonomousDatabaseRequest := database.UpdateAutonomousDatabaseRequest{
			AutonomousDatabaseId:            common.String(string(adb.Spec.AdbId)),
			UpdateAutonomousDatabaseDetails: updateAutonomousDatabaseDetails,
		}

		if _, err := dbClient.UpdateAutonomousDatabase(ctx, updateAutonomousDatabaseRequest); err != nil {
			return err
		}
	}

	return nil
}

func buildAdditionalSpecPayload(spec databasev1beta1.AutonomousDatabasesSpec, excluded map[string]struct{}) (map[string]json.RawMessage, error) {
	payloadBytes, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal autonomous database spec: %w", err)
	}

	payload := map[string]json.RawMessage{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("decode autonomous database spec payload: %w", err)
	}

	for fieldName := range excluded {
		delete(payload, fieldName)
	}

	for fieldName, rawValue := range payload {
		var decoded any
		if err := json.Unmarshal(rawValue, &decoded); err != nil {
			return nil, fmt.Errorf("decode autonomous database field %q: %w", fieldName, err)
		}

		normalizedValue, empty := pruneEmptyJSONValue(decoded)
		if empty {
			delete(payload, fieldName)
			continue
		}

		normalizedRaw, err := json.Marshal(normalizedValue)
		if err != nil {
			return nil, fmt.Errorf("marshal normalized autonomous database field %q: %w", fieldName, err)
		}
		payload[fieldName] = normalizedRaw
	}

	return payload, nil
}

func decodeAdditionalSpecPayload(payload map[string]json.RawMessage, out any) error {
	if len(payload) == 0 {
		return nil
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal autonomous database payload: %w", err)
	}
	if err := json.Unmarshal(payloadBytes, out); err != nil {
		return fmt.Errorf("decode autonomous database payload into SDK details: %w", err)
	}
	return nil
}

func additionalAutonomousDatabaseUpdateNeeded(payload map[string]json.RawMessage, existing database.AutonomousDatabase) (bool, error) {
	if len(payload) == 0 {
		return false, nil
	}

	existingBytes, err := json.Marshal(existing)
	if err != nil {
		return false, fmt.Errorf("marshal existing autonomous database: %w", err)
	}

	existingPayload := map[string]json.RawMessage{}
	if err := json.Unmarshal(existingBytes, &existingPayload); err != nil {
		return false, fmt.Errorf("decode existing autonomous database payload: %w", err)
	}

	for fieldName, desiredValue := range payload {
		currentValue, ok := existingPayload[fieldName]
		if !ok {
			return true, nil
		}
		equal, err := jsonPayloadEqual(desiredValue, currentValue)
		if err != nil {
			return false, fmt.Errorf("compare autonomous database field %q: %w", fieldName, err)
		}
		if !equal {
			return true, nil
		}
	}

	return false, nil
}

func additionalAutonomousDatabaseUpdateNeededForSpec(spec databasev1beta1.AutonomousDatabasesSpec, existing database.AutonomousDatabase) (bool, error) {
	payload, err := buildAdditionalSpecPayload(spec, additionalUpdatePayloadExcludedFields)
	if err != nil {
		return false, err
	}
	return additionalAutonomousDatabaseUpdateNeeded(payload, existing)
}

func jsonPayloadEqual(leftRaw, rightRaw json.RawMessage) (bool, error) {
	var left any
	if err := json.Unmarshal(leftRaw, &left); err != nil {
		return false, err
	}

	var right any
	if err := json.Unmarshal(rightRaw, &right); err != nil {
		return false, err
	}

	return jsonPayloadSubsetEqual(left, right), nil
}

func jsonPayloadSubsetEqual(desired, current any) bool {
	switch desiredTyped := desired.(type) {
	case map[string]any:
		currentTyped, ok := current.(map[string]any)
		if !ok {
			return false
		}
		for key, desiredValue := range desiredTyped {
			currentValue, ok := currentTyped[key]
			if !ok {
				return false
			}
			if !jsonPayloadSubsetEqual(desiredValue, currentValue) {
				return false
			}
		}
		return true
	case []any:
		currentTyped, ok := current.([]any)
		if !ok || len(desiredTyped) != len(currentTyped) {
			return false
		}
		for i := range desiredTyped {
			if !jsonPayloadSubsetEqual(desiredTyped[i], currentTyped[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(desired, current)
	}
}

func pruneEmptyJSONValue(value any) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, child := range typed {
			normalizedChild, empty := pruneEmptyJSONValue(child)
			if empty {
				continue
			}
			normalized[key] = normalizedChild
		}
		if len(normalized) == 0 {
			return nil, true
		}
		return normalized, false
	case []any:
		if len(typed) == 0 {
			return nil, true
		}
		return typed, false
	case nil:
		return nil, true
	default:
		return value, false
	}
}
