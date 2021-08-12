/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb

import (
	"context"
	"fmt"
	"github.com/oracle/oci-go-sdk/v41/common"
	"github.com/oracle/oci-go-sdk/v41/database"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	"reflect"
)

type AdbServiceClient interface {
	CreateAdb(ctx context.Context, adb ociv1beta1.AutonomousDatabases) (database.AutonomousDatabase, error)

	UpdateAdb(ctx context.Context, request database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error)

	GetAdb(ctx context.Context, request database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error)

	DeleteAdb() (string, error)

	servicemanager.OSOKServiceManager
}

func getDbClient(provider common.ConfigurationProvider) database.DatabaseClient {
	dbClient, _ := database.NewDatabaseClientWithConfigurationProvider(provider)
	return dbClient
}

func (c *AdbServiceManager) CreateAdb(ctx context.Context, adb ociv1beta1.AutonomousDatabases, adminPwd string) (database.CreateAutonomousDatabaseResponse, error) {

	dbClient := getDbClient(c.Provider)

	c.Log.DebugLog("Creating Autonomous Database ", "name", adb.Spec.DisplayName)

	createAutonomousDatabaseDetails := database.CreateAutonomousDatabaseDetails{
		CompartmentId:        common.String(string(adb.Spec.CompartmentId)),
		DisplayName:          common.String(adb.Spec.DisplayName),
		DbName:               common.String(adb.Spec.DbName),
		CpuCoreCount:         common.Int(adb.Spec.CpuCoreCount),
		DataStorageSizeInTBs: common.Int(adb.Spec.DataStorageSizeInTBs),
		AdminPassword:        common.String(adminPwd),
		IsAutoScalingEnabled: common.Bool(adb.Spec.IsAutoScalingEnabled),
		IsDedicated:          common.Bool(adb.Spec.IsDedicated),
		DbVersion:            common.String(adb.Spec.DbVersion),
		DbWorkload:           database.CreateAutonomousDatabaseBaseDbWorkloadEnum(adb.Spec.DbWorkload),
		IsFreeTier:           common.Bool(adb.Spec.IsFreeTier),
		LicenseModel:         database.CreateAutonomousDatabaseBaseLicenseModelEnum(adb.Spec.LicenseModel),
		FreeformTags:         adb.Spec.FreeFormTags,
		DefinedTags:          *util.ConvertToOciDefinedTags(&adb.Spec.DefinedTags),
	}

	createAutonomousDatabaseRequest := database.CreateAutonomousDatabaseRequest{
		CreateAutonomousDatabaseDetails: createAutonomousDatabaseDetails,
	}

	return dbClient.CreateAutonomousDatabase(ctx, createAutonomousDatabaseRequest)
}

func (c *AdbServiceManager) GetAdbOcid(ctx context.Context, adb ociv1beta1.AutonomousDatabases) (*ociv1beta1.OCID, error) {
	dbClient := getDbClient(c.Provider)

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

			return (*ociv1beta1.OCID)(listAdbResponse.Items[0].Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("Autonomous Database %s does not exist.", adb.Spec.DisplayName))
	return nil, nil
}

func (c *AdbServiceManager) DeleteAdb() (string, error) {
	return "", nil
}

// Sync the Autonomous Database details
func (c *AdbServiceManager) GetAdb(ctx context.Context, adbId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*database.AutonomousDatabase, error) {

	dbClient := getDbClient(c.Provider)

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

func (c *AdbServiceManager) UpdateAdb(ctx context.Context, adb *ociv1beta1.AutonomousDatabases) error {

	dbClient := getDbClient(c.Provider)

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
