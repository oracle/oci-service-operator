/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"reflect"

	cloudbridgesdk "github.com/oracle/oci-go-sdk/v65/cloudbridge"
	cloudmigrationssdk "github.com/oracle/oci-go-sdk/v65/cloudmigrations"
	filestoragesdk "github.com/oracle/oci-go-sdk/v65/filestorage"
	fleetsoftwareupdatesdk "github.com/oracle/oci-go-sdk/v65/fleetsoftwareupdate"
	goldengatesdk "github.com/oracle/oci-go-sdk/v65/goldengate"
	resourcemanagersdk "github.com/oracle/oci-go-sdk/v65/resourcemanager"
)

var (
	cloudBridgeAssetCreateType                   = reflect.TypeOf((*cloudbridgesdk.CreateAssetDetails)(nil)).Elem()
	cloudBridgeAssetUpdateType                   = reflect.TypeOf((*cloudbridgesdk.UpdateAssetDetails)(nil)).Elem()
	cloudBridgeAssetSourceCreateType             = reflect.TypeOf((*cloudbridgesdk.CreateAssetSourceDetails)(nil)).Elem()
	cloudBridgeAssetSourceUpdateType             = reflect.TypeOf((*cloudbridgesdk.UpdateAssetSourceDetails)(nil)).Elem()
	cloudMigrationsTargetAssetCreateType         = reflect.TypeOf((*cloudmigrationssdk.CreateTargetAssetDetails)(nil)).Elem()
	cloudMigrationsTargetAssetUpdateType         = reflect.TypeOf((*cloudmigrationssdk.UpdateTargetAssetDetails)(nil)).Elem()
	fileStorageOutboundConnectorCreateType       = reflect.TypeOf((*filestoragesdk.CreateOutboundConnectorDetails)(nil)).Elem()
	fleetSoftwareUpdateActionCreateType          = reflect.TypeOf((*fleetsoftwareupdatesdk.CreateFsuActionDetails)(nil)).Elem()
	fleetSoftwareUpdateActionUpdateType          = reflect.TypeOf((*fleetsoftwareupdatesdk.UpdateFsuActionDetails)(nil)).Elem()
	fleetSoftwareUpdateCollectionCreateType      = reflect.TypeOf((*fleetsoftwareupdatesdk.CreateFsuCollectionDetails)(nil)).Elem()
	fleetSoftwareUpdateCycleCreateType           = reflect.TypeOf((*fleetsoftwareupdatesdk.CreateFsuCycleDetails)(nil)).Elem()
	fleetSoftwareUpdateCycleUpdateType           = reflect.TypeOf((*fleetsoftwareupdatesdk.UpdateFsuCycleDetails)(nil)).Elem()
	fleetSoftwareUpdateReadinessCreateType       = reflect.TypeOf((*fleetsoftwareupdatesdk.CreateFsuReadinessCheckDetails)(nil)).Elem()
	goldenGateConnectionCreateType               = reflect.TypeOf((*goldengatesdk.CreateConnectionDetails)(nil)).Elem()
	goldenGateConnectionUpdateType               = reflect.TypeOf((*goldengatesdk.UpdateConnectionDetails)(nil)).Elem()
	goldenGatePipelineCreateType                 = reflect.TypeOf((*goldengatesdk.CreatePipelineDetails)(nil)).Elem()
	goldenGatePipelineUpdateType                 = reflect.TypeOf((*goldengatesdk.UpdatePipelineDetails)(nil)).Elem()
	resourceManagerConfigurationSourceCreateType = reflect.TypeOf((*resourcemanagersdk.CreateConfigurationSourceProviderDetails)(nil)).Elem()
	resourceManagerConfigurationSourceUpdateType = reflect.TypeOf((*resourcemanagersdk.UpdateConfigurationSourceProviderDetails)(nil)).Elem()
)

func convertAdditionalPolymorphicInterfaceValue(payload []byte, targetType reflect.Type) (reflect.Value, bool, error) {
	switch targetType {
	case cloudBridgeAssetCreateType:
		body, err := convertDiscriminatedInterface[cloudbridgesdk.CreateAssetDetails](payload, "Cloud Bridge asset", "assetType", map[string]reflect.Type{
			"AWS_EBS":   reflect.TypeOf(cloudbridgesdk.CreateAwsEbsAssetDetails{}),
			"VMWARE_VM": reflect.TypeOf(cloudbridgesdk.CreateVmwareVmAssetDetails{}),
			"AWS_EC2":   reflect.TypeOf(cloudbridgesdk.CreateAwsEc2AssetDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case cloudBridgeAssetUpdateType:
		body, err := convertDiscriminatedInterface[cloudbridgesdk.UpdateAssetDetails](payload, "Cloud Bridge asset", "assetType", map[string]reflect.Type{
			"VM":        reflect.TypeOf(cloudbridgesdk.UpdateVmAssetDetails{}),
			"AWS_EBS":   reflect.TypeOf(cloudbridgesdk.UpdateAwsEbsAssetDetails{}),
			"VMWARE_VM": reflect.TypeOf(cloudbridgesdk.UpdateVmwareVmAssetDetails{}),
			"AWS_EC2":   reflect.TypeOf(cloudbridgesdk.UpdateAwsEc2AssetDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case cloudBridgeAssetSourceCreateType:
		body, err := convertDiscriminatedInterface[cloudbridgesdk.CreateAssetSourceDetails](payload, "Cloud Bridge asset source", "type", map[string]reflect.Type{
			"VMWARE": reflect.TypeOf(cloudbridgesdk.CreateVmWareAssetSourceDetails{}),
			"AWS":    reflect.TypeOf(cloudbridgesdk.CreateAwsAssetSourceDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case cloudBridgeAssetSourceUpdateType:
		body, err := convertDiscriminatedInterface[cloudbridgesdk.UpdateAssetSourceDetails](payload, "Cloud Bridge asset source", "type", map[string]reflect.Type{
			"VMWARE": reflect.TypeOf(cloudbridgesdk.UpdateVmWareAssetSourceDetails{}),
			"AWS":    reflect.TypeOf(cloudbridgesdk.UpdateAwsAssetSourceDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case cloudMigrationsTargetAssetCreateType:
		body, err := convertDiscriminatedInterface[cloudmigrationssdk.CreateTargetAssetDetails](payload, "Cloud Migrations target asset", "type", map[string]reflect.Type{
			"OLVM_INSTANCE": reflect.TypeOf(cloudmigrationssdk.CreateOlvmTargetAssetDetails{}),
			"INSTANCE":      reflect.TypeOf(cloudmigrationssdk.CreateVmTargetAssetDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case cloudMigrationsTargetAssetUpdateType:
		body, err := convertDiscriminatedInterface[cloudmigrationssdk.UpdateTargetAssetDetails](payload, "Cloud Migrations target asset", "type", map[string]reflect.Type{
			"INSTANCE":      reflect.TypeOf(cloudmigrationssdk.UpdateVmTargetAssetDetails{}),
			"OLVM_INSTANCE": reflect.TypeOf(cloudmigrationssdk.UpdateOlvmTargetAssetDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case fileStorageOutboundConnectorCreateType:
		body, err := convertDiscriminatedInterface[filestoragesdk.CreateOutboundConnectorDetails](payload, "File Storage outbound connector", "connectorType", map[string]reflect.Type{
			"LDAPBIND": reflect.TypeOf(filestoragesdk.CreateLdapBindAccountDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case fleetSoftwareUpdateActionCreateType:
		body, err := convertDiscriminatedInterface[fleetsoftwareupdatesdk.CreateFsuActionDetails](payload, "Fleet Software Update action", "type", map[string]reflect.Type{
			"ROLLBACK_MAINTENANCE_CYCLE": reflect.TypeOf(fleetsoftwareupdatesdk.CreateRollbackCycleApplyActionDetails{}),
			"APPLY":                      reflect.TypeOf(fleetsoftwareupdatesdk.CreateApplyActionDetails{}),
			"STAGE":                      reflect.TypeOf(fleetsoftwareupdatesdk.CreateStageActionDetails{}),
			"ROLLBACK_AND_REMOVE_TARGET": reflect.TypeOf(fleetsoftwareupdatesdk.CreateRollbackActionDetails{}),
			"CLEANUP":                    reflect.TypeOf(fleetsoftwareupdatesdk.CreateCleanupActionDetails{}),
			"PRECHECK":                   reflect.TypeOf(fleetsoftwareupdatesdk.CreatePrecheckActionDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case fleetSoftwareUpdateActionUpdateType:
		body, err := convertDiscriminatedInterface[fleetsoftwareupdatesdk.UpdateFsuActionDetails](payload, "Fleet Software Update action", "type", map[string]reflect.Type{
			"ROLLBACK_MAINTENANCE_CYCLE": reflect.TypeOf(fleetsoftwareupdatesdk.UpdateRollbackCycleActionDetails{}),
			"STAGE":                      reflect.TypeOf(fleetsoftwareupdatesdk.UpdateStageActionDetails{}),
			"APPLY":                      reflect.TypeOf(fleetsoftwareupdatesdk.UpdateApplyActionDetails{}),
			"ROLLBACK_AND_REMOVE_TARGET": reflect.TypeOf(fleetsoftwareupdatesdk.UpdateRollbackActionDetails{}),
			"PRECHECK":                   reflect.TypeOf(fleetsoftwareupdatesdk.UpdatePrecheckActionDetails{}),
			"CLEANUP":                    reflect.TypeOf(fleetsoftwareupdatesdk.UpdateCleanupActionDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case fleetSoftwareUpdateCollectionCreateType:
		body, err := convertDiscriminatedInterface[fleetsoftwareupdatesdk.CreateFsuCollectionDetails](payload, "Fleet Software Update collection", "type", map[string]reflect.Type{
			"DB":          reflect.TypeOf(fleetsoftwareupdatesdk.CreateDbFsuCollectionDetails{}),
			"GI":          reflect.TypeOf(fleetsoftwareupdatesdk.CreateGiFsuCollectionDetails{}),
			"GUEST_OS":    reflect.TypeOf(fleetsoftwareupdatesdk.CreateGuestOsFsuCollectionDetails{}),
			"EXADB_STACK": reflect.TypeOf(fleetsoftwareupdatesdk.CreateExadbStackFsuCollectionDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case fleetSoftwareUpdateCycleCreateType:
		body, err := convertDiscriminatedInterface[fleetsoftwareupdatesdk.CreateFsuCycleDetails](payload, "Fleet Software Update cycle", "type", map[string]reflect.Type{
			"PATCH":   reflect.TypeOf(fleetsoftwareupdatesdk.CreatePatchFsuCycle{}),
			"UPGRADE": reflect.TypeOf(fleetsoftwareupdatesdk.CreateUpgradeFsuCycle{}),
		})
		return interfaceValue(targetType, body, err)
	case fleetSoftwareUpdateCycleUpdateType:
		body, err := convertDiscriminatedInterface[fleetsoftwareupdatesdk.UpdateFsuCycleDetails](payload, "Fleet Software Update cycle", "type", map[string]reflect.Type{
			"PATCH":   reflect.TypeOf(fleetsoftwareupdatesdk.UpdatePatchFsuCycle{}),
			"UPGRADE": reflect.TypeOf(fleetsoftwareupdatesdk.UpdateUpgradeFsuCycle{}),
		})
		return interfaceValue(targetType, body, err)
	case fleetSoftwareUpdateReadinessCreateType:
		body, err := convertDiscriminatedInterface[fleetsoftwareupdatesdk.CreateFsuReadinessCheckDetails](payload, "Fleet Software Update readiness check", "type", map[string]reflect.Type{
			"TARGET": reflect.TypeOf(fleetsoftwareupdatesdk.CreateTargetFsuReadinessCheckDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case goldenGateConnectionCreateType:
		body, err := convertDiscriminatedInterface[goldengatesdk.CreateConnectionDetails](payload, "GoldenGate connection", "connectionType", goldenGateCreateConnectionTypes())
		return interfaceValue(targetType, body, err)
	case goldenGateConnectionUpdateType:
		body, err := convertDiscriminatedInterface[goldengatesdk.UpdateConnectionDetails](payload, "GoldenGate connection", "connectionType", goldenGateUpdateConnectionTypes())
		return interfaceValue(targetType, body, err)
	case goldenGatePipelineCreateType:
		body, err := convertDiscriminatedInterface[goldengatesdk.CreatePipelineDetails](payload, "GoldenGate pipeline", "recipeType", map[string]reflect.Type{
			"ZERO_ETL": reflect.TypeOf(goldengatesdk.CreateZeroEtlPipelineDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case goldenGatePipelineUpdateType:
		body, err := convertDiscriminatedInterface[goldengatesdk.UpdatePipelineDetails](payload, "GoldenGate pipeline", "recipeType", map[string]reflect.Type{
			"ZERO_ETL": reflect.TypeOf(goldengatesdk.UpdateZeroEtlPipelineDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case resourceManagerConfigurationSourceCreateType:
		body, err := convertDiscriminatedInterface[resourcemanagersdk.CreateConfigurationSourceProviderDetails](payload, "Resource Manager configuration source provider", "configSourceProviderType", map[string]reflect.Type{
			"GITLAB_ACCESS_TOKEN":                  reflect.TypeOf(resourcemanagersdk.CreateGitlabAccessTokenConfigurationSourceProviderDetails{}),
			"BITBUCKET_CLOUD_USERNAME_APPPASSWORD": reflect.TypeOf(resourcemanagersdk.CreateBitbucketCloudUsernameAppPasswordConfigurationSourceProviderDetails{}),
			"GITHUB_ACCESS_TOKEN":                  reflect.TypeOf(resourcemanagersdk.CreateGithubAccessTokenConfigurationSourceProviderDetails{}),
			"BITBUCKET_SERVER_ACCESS_TOKEN":        reflect.TypeOf(resourcemanagersdk.CreateBitbucketServerAccessTokenConfigurationSourceProviderDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case resourceManagerConfigurationSourceUpdateType:
		body, err := convertDiscriminatedInterface[resourcemanagersdk.UpdateConfigurationSourceProviderDetails](payload, "Resource Manager configuration source provider", "configSourceProviderType", map[string]reflect.Type{
			"BITBUCKET_CLOUD_USERNAME_APPPASSWORD": reflect.TypeOf(resourcemanagersdk.UpdateBitbucketCloudUsernameAppPasswordConfigurationSourceProviderDetails{}),
			"BITBUCKET_SERVER_ACCESS_TOKEN":        reflect.TypeOf(resourcemanagersdk.UpdateBitbucketServerAccessTokenConfigurationSourceProviderDetails{}),
			"GITLAB_ACCESS_TOKEN":                  reflect.TypeOf(resourcemanagersdk.UpdateGitlabAccessTokenConfigurationSourceProviderDetails{}),
			"GITHUB_ACCESS_TOKEN":                  reflect.TypeOf(resourcemanagersdk.UpdateGithubAccessTokenConfigurationSourceProviderDetails{}),
		})
		return interfaceValue(targetType, body, err)
	default:
		return reflect.Value{}, false, nil
	}
}

func additionalPolymorphicUpdateDiscriminatorField(targetType reflect.Type) (string, bool) {
	switch targetType {
	case cloudBridgeAssetUpdateType:
		return "assetType", true
	case cloudBridgeAssetSourceUpdateType,
		cloudMigrationsTargetAssetUpdateType,
		fleetSoftwareUpdateActionUpdateType,
		fleetSoftwareUpdateCycleUpdateType:
		return "type", true
	case goldenGateConnectionUpdateType:
		return "connectionType", true
	case goldenGatePipelineUpdateType:
		return "recipeType", true
	case resourceManagerConfigurationSourceUpdateType:
		return "configSourceProviderType", true
	default:
		return "", false
	}
}

func goldenGateCreateConnectionTypes() map[string]reflect.Type {
	return map[string]reflect.Type{
		"POSTGRESQL":              reflect.TypeOf(goldengatesdk.CreatePostgresqlConnectionDetails{}),
		"KAFKA_SCHEMA_REGISTRY":   reflect.TypeOf(goldengatesdk.CreateKafkaSchemaRegistryConnectionDetails{}),
		"MICROSOFT_SQLSERVER":     reflect.TypeOf(goldengatesdk.CreateMicrosoftSqlserverConnectionDetails{}),
		"AMAZON_KINESIS":          reflect.TypeOf(goldengatesdk.CreateAmazonKinesisConnectionDetails{}),
		"AZURE_DATA_LAKE_STORAGE": reflect.TypeOf(goldengatesdk.CreateAzureDataLakeStorageConnectionDetails{}),
		"GOOGLE_PUBSUB":           reflect.TypeOf(goldengatesdk.CreateGooglePubSubConnectionDetails{}),
		"HDFS":                    reflect.TypeOf(goldengatesdk.CreateHdfsConnectionDetails{}),
		"OCI_OBJECT_STORAGE":      reflect.TypeOf(goldengatesdk.CreateOciObjectStorageConnectionDetails{}),
		"REDIS":                   reflect.TypeOf(goldengatesdk.CreateRedisConnectionDetails{}),
		"MICROSOFT_FABRIC":        reflect.TypeOf(goldengatesdk.CreateMicrosoftFabricConnectionDetails{}),
		"GOOGLE_CLOUD_STORAGE":    reflect.TypeOf(goldengatesdk.CreateGoogleCloudStorageConnectionDetails{}),
		"KAFKA":                   reflect.TypeOf(goldengatesdk.CreateKafkaConnectionDetails{}),
		"ORACLE_NOSQL":            reflect.TypeOf(goldengatesdk.CreateOracleNosqlConnectionDetails{}),
		"JAVA_MESSAGE_SERVICE":    reflect.TypeOf(goldengatesdk.CreateJavaMessageServiceConnectionDetails{}),
		"GOOGLE_BIGQUERY":         reflect.TypeOf(goldengatesdk.CreateGoogleBigQueryConnectionDetails{}),
		"SNOWFLAKE":               reflect.TypeOf(goldengatesdk.CreateSnowflakeConnectionDetails{}),
		"MONGODB":                 reflect.TypeOf(goldengatesdk.CreateMongoDbConnectionDetails{}),
		"ORACLE_AI_DATA_PLATFORM": reflect.TypeOf(goldengatesdk.CreateOracleAiDataPlatformConnectionDetails{}),
		"AMAZON_S3":               reflect.TypeOf(goldengatesdk.CreateAmazonS3ConnectionDetails{}),
		"DATABRICKS":              reflect.TypeOf(goldengatesdk.CreateDatabricksConnectionDetails{}),
		"DB2":                     reflect.TypeOf(goldengatesdk.CreateDb2ConnectionDetails{}),
		"ELASTICSEARCH":           reflect.TypeOf(goldengatesdk.CreateElasticsearchConnectionDetails{}),
		"AZURE_SYNAPSE_ANALYTICS": reflect.TypeOf(goldengatesdk.CreateAzureSynapseConnectionDetails{}),
		"ICEBERG":                 reflect.TypeOf(goldengatesdk.CreateIcebergConnectionDetails{}),
		"MYSQL":                   reflect.TypeOf(goldengatesdk.CreateMysqlConnectionDetails{}),
		"GENERIC":                 reflect.TypeOf(goldengatesdk.CreateGenericConnectionDetails{}),
		"ORACLE":                  reflect.TypeOf(goldengatesdk.CreateOracleConnectionDetails{}),
		"GOLDENGATE":              reflect.TypeOf(goldengatesdk.CreateGoldenGateConnectionDetails{}),
		"AMAZON_REDSHIFT":         reflect.TypeOf(goldengatesdk.CreateAmazonRedshiftConnectionDetails{}),
	}
}

func goldenGateUpdateConnectionTypes() map[string]reflect.Type {
	return map[string]reflect.Type{
		"ELASTICSEARCH":           reflect.TypeOf(goldengatesdk.UpdateElasticsearchConnectionDetails{}),
		"GOOGLE_BIGQUERY":         reflect.TypeOf(goldengatesdk.UpdateGoogleBigQueryConnectionDetails{}),
		"ORACLE":                  reflect.TypeOf(goldengatesdk.UpdateOracleConnectionDetails{}),
		"AMAZON_REDSHIFT":         reflect.TypeOf(goldengatesdk.UpdateAmazonRedshiftConnectionDetails{}),
		"OCI_OBJECT_STORAGE":      reflect.TypeOf(goldengatesdk.UpdateOciObjectStorageConnectionDetails{}),
		"REDIS":                   reflect.TypeOf(goldengatesdk.UpdateRedisConnectionDetails{}),
		"MONGODB":                 reflect.TypeOf(goldengatesdk.UpdateMongoDbConnectionDetails{}),
		"GOOGLE_CLOUD_STORAGE":    reflect.TypeOf(goldengatesdk.UpdateGoogleCloudStorageConnectionDetails{}),
		"ORACLE_AI_DATA_PLATFORM": reflect.TypeOf(goldengatesdk.UpdateOracleAiDataPlatformConnectionDetails{}),
		"MICROSOFT_FABRIC":        reflect.TypeOf(goldengatesdk.UpdateMicrosoftFabricConnectionDetails{}),
		"POSTGRESQL":              reflect.TypeOf(goldengatesdk.UpdatePostgresqlConnectionDetails{}),
		"MICROSOFT_SQLSERVER":     reflect.TypeOf(goldengatesdk.UpdateMicrosoftSqlserverConnectionDetails{}),
		"SNOWFLAKE":               reflect.TypeOf(goldengatesdk.UpdateSnowflakeConnectionDetails{}),
		"HDFS":                    reflect.TypeOf(goldengatesdk.UpdateHdfsConnectionDetails{}),
		"DATABRICKS":              reflect.TypeOf(goldengatesdk.UpdateDatabricksConnectionDetails{}),
		"KAFKA":                   reflect.TypeOf(goldengatesdk.UpdateKafkaConnectionDetails{}),
		"AZURE_DATA_LAKE_STORAGE": reflect.TypeOf(goldengatesdk.UpdateAzureDataLakeStorageConnectionDetails{}),
		"AMAZON_KINESIS":          reflect.TypeOf(goldengatesdk.UpdateAmazonKinesisConnectionDetails{}),
		"JAVA_MESSAGE_SERVICE":    reflect.TypeOf(goldengatesdk.UpdateJavaMessageServiceConnectionDetails{}),
		"GOLDENGATE":              reflect.TypeOf(goldengatesdk.UpdateGoldenGateConnectionDetails{}),
		"GOOGLE_PUBSUB":           reflect.TypeOf(goldengatesdk.UpdateGooglePubSubConnectionDetails{}),
		"ORACLE_NOSQL":            reflect.TypeOf(goldengatesdk.UpdateOracleNosqlConnectionDetails{}),
		"KAFKA_SCHEMA_REGISTRY":   reflect.TypeOf(goldengatesdk.UpdateKafkaSchemaRegistryConnectionDetails{}),
		"AMAZON_S3":               reflect.TypeOf(goldengatesdk.UpdateAmazonS3ConnectionDetails{}),
		"MYSQL":                   reflect.TypeOf(goldengatesdk.UpdateMysqlConnectionDetails{}),
		"DB2":                     reflect.TypeOf(goldengatesdk.UpdateDb2ConnectionDetails{}),
		"ICEBERG":                 reflect.TypeOf(goldengatesdk.UpdateIcebergConnectionDetails{}),
		"GENERIC":                 reflect.TypeOf(goldengatesdk.UpdateGenericConnectionDetails{}),
		"AZURE_SYNAPSE_ANALYTICS": reflect.TypeOf(goldengatesdk.UpdateAzureSynapseConnectionDetails{}),
	}
}
