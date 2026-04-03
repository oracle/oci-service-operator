package apispec

import (
	"reflect"

	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	dataflowv1beta1 "github.com/oracle/oci-service-operator/api/dataflow/v1beta1"
	functionsv1beta1 "github.com/oracle/oci-service-operator/api/functions/v1beta1"
	identityv1beta1 "github.com/oracle/oci-service-operator/api/identity/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	nosqlv1beta1 "github.com/oracle/oci-service-operator/api/nosql/v1beta1"
	objectstoragev1beta1 "github.com/oracle/oci-service-operator/api/objectstorage/v1beta1"
	opensearchv1beta1 "github.com/oracle/oci-service-operator/api/opensearch/v1beta1"
	psqlv1beta1 "github.com/oracle/oci-service-operator/api/psql/v1beta1"
	queuev1beta1 "github.com/oracle/oci-service-operator/api/queue/v1beta1"
	redisv1beta1 "github.com/oracle/oci-service-operator/api/redis/v1beta1"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	vaultv1beta1 "github.com/oracle/oci-service-operator/api/vault/v1beta1"
)

type SDKMapping struct {
	SDKStruct  string
	APISurface string
	Exclude    bool
	Reason     string
}

type Target struct {
	Name        string
	SpecType    reflect.Type
	StatusType  reflect.Type
	SDKMappings []SDKMapping
}

var targets = []Target{
	{
		Name:       "AutonomousDatabase",
		SpecType:   reflect.TypeOf(databasev1beta1.AutonomousDatabaseSpec{}),
		StatusType: reflect.TypeOf(databasev1beta1.AutonomousDatabaseStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "database.CreateAutonomousDatabaseDetails",
			},
			{
				SDKStruct: "database.UpdateAutonomousDatabaseDetails",
			},
			{
				SDKStruct: "database.AutonomousDatabase",
			},
			{
				SDKStruct: "database.AutonomousDatabaseSummary",
			},
		},
	},
	{
		Name:       "MySqlBackup",
		SpecType:   reflect.TypeOf(mysqlv1beta1.BackupSpec{}),
		StatusType: reflect.TypeOf(mysqlv1beta1.BackupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "mysql.CreateBackupDetails",
			},
			{
				SDKStruct: "mysql.UpdateBackupDetails",
			},
			{
				SDKStruct: "mysql.Backup",
			},
			{
				SDKStruct: "mysql.BackupSummary",
			},
		},
	},
	{
		Name:       "MySqlChannel",
		SpecType:   reflect.TypeOf(mysqlv1beta1.ChannelSpec{}),
		StatusType: reflect.TypeOf(mysqlv1beta1.ChannelStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "mysql.CreateChannelDetails",
			},
			{
				SDKStruct: "mysql.UpdateChannelDetails",
			},
			{
				SDKStruct: "mysql.Channel",
			},
			{
				SDKStruct: "mysql.ChannelSummary",
			},
		},
	},
	{
		Name:       "MySqlConfiguration",
		SpecType:   reflect.TypeOf(mysqlv1beta1.ConfigurationSpec{}),
		StatusType: reflect.TypeOf(mysqlv1beta1.ConfigurationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "mysql.CreateConfigurationDetails",
			},
			{
				SDKStruct: "mysql.UpdateConfigurationDetails",
			},
			{
				SDKStruct: "mysql.Configuration",
			},
			{
				SDKStruct: "mysql.ConfigurationSummary",
			},
		},
	},
	{
		Name:       "MySqlDbSystem",
		SpecType:   reflect.TypeOf(mysqlv1beta1.DbSystemSpec{}),
		StatusType: reflect.TypeOf(mysqlv1beta1.DbSystemStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "mysql.CreateDbSystemDetails",
			},
			{
				SDKStruct: "mysql.UpdateDbSystemDetails",
			},
			{
				SDKStruct: "mysql.DbSystem",
			},
			{
				SDKStruct: "mysql.DbSystemSummary",
			},
		},
	},
	{
		Name:       "MySqlHeatWaveCluster",
		SpecType:   reflect.TypeOf(mysqlv1beta1.HeatWaveClusterSpec{}),
		StatusType: reflect.TypeOf(mysqlv1beta1.HeatWaveClusterStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "mysql.UpdateHeatWaveClusterDetails",
			},
			{
				SDKStruct: "mysql.HeatWaveCluster",
			},
			{
				SDKStruct: "mysql.HeatWaveClusterSummary",
			},
		},
	},
	{
		Name:       "MySqlHeatWaveClusterMemoryEstimate",
		SpecType:   reflect.TypeOf(mysqlv1beta1.HeatWaveClusterMemoryEstimateSpec{}),
		StatusType: reflect.TypeOf(mysqlv1beta1.HeatWaveClusterMemoryEstimateStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "mysql.HeatWaveClusterMemoryEstimate",
			},
		},
	},
	{
		Name:       "MySqlReplica",
		SpecType:   reflect.TypeOf(mysqlv1beta1.ReplicaSpec{}),
		StatusType: reflect.TypeOf(mysqlv1beta1.ReplicaStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "mysql.CreateReplicaDetails",
			},
			{
				SDKStruct: "mysql.UpdateReplicaDetails",
			},
			{
				SDKStruct: "mysql.Replica",
			},
			{
				SDKStruct: "mysql.ReplicaSummary",
			},
		},
	},
	{
		Name:       "MySqlShape",
		SpecType:   reflect.TypeOf(mysqlv1beta1.ShapeSpec{}),
		StatusType: reflect.TypeOf(mysqlv1beta1.ShapeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "mysql.ShapeSummary",
			},
		},
	},
	{
		Name:       "MySqlVersion",
		SpecType:   reflect.TypeOf(mysqlv1beta1.VersionSpec{}),
		StatusType: reflect.TypeOf(mysqlv1beta1.VersionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "mysql.Version",
			},
			{
				SDKStruct: "mysql.VersionSummary",
			},
		},
	},
	{
		Name:       "MySqlWorkRequest",
		SpecType:   reflect.TypeOf(mysqlv1beta1.WorkRequestSpec{}),
		StatusType: reflect.TypeOf(mysqlv1beta1.WorkRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "mysql.WorkRequest",
			},
			{
				SDKStruct: "mysql.WorkRequestSummary",
			},
		},
	},
	{
		Name:       "MySqlWorkRequestError",
		SpecType:   reflect.TypeOf(mysqlv1beta1.WorkRequestErrorSpec{}),
		StatusType: reflect.TypeOf(mysqlv1beta1.WorkRequestErrorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "mysql.WorkRequestError",
			},
		},
	},
	{
		Name:       "MySqlWorkRequestLog",
		SpecType:   reflect.TypeOf(mysqlv1beta1.WorkRequestLogSpec{}),
		StatusType: reflect.TypeOf(mysqlv1beta1.WorkRequestLogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "mysql.WorkRequestLogEntry",
				Exclude:   true,
				Reason:    "Intentionally untracked: OCI read-model mappings broaden desired-state coverage, and this CRD does not expose a meaningful status surface for parity tracking.",
			},
		},
	},
	{
		Name:       "Stream",
		SpecType:   reflect.TypeOf(streamingv1beta1.StreamSpec{}),
		StatusType: reflect.TypeOf(streamingv1beta1.StreamStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "streaming.CreateStreamDetails",
			},
			{
				SDKStruct: "streaming.UpdateStreamDetails",
			},
			{
				SDKStruct: "streaming.Stream",
				Exclude:   true,
				Reason:    "Intentionally untracked: OCI read-model mappings broaden desired-state coverage, and this CRD does not expose a meaningful status surface for parity tracking.",
			},
			{
				SDKStruct: "streaming.StreamSummary",
				Exclude:   true,
				Reason:    "Intentionally untracked: OCI read-model mappings broaden desired-state coverage, and this CRD does not expose a meaningful status surface for parity tracking.",
			},
		},
	},
	{
		Name:       "Channel",
		SpecType:   reflect.TypeOf(queuev1beta1.ChannelSpec{}),
		StatusType: reflect.TypeOf(queuev1beta1.ChannelStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "queue.ChannelCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "Message",
		SpecType:   reflect.TypeOf(queuev1beta1.MessageSpec{}),
		StatusType: reflect.TypeOf(queuev1beta1.MessageStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "queue.UpdateMessageDetails",
			},
		},
	},
	{
		Name:       "Queue",
		SpecType:   reflect.TypeOf(queuev1beta1.QueueSpec{}),
		StatusType: reflect.TypeOf(queuev1beta1.QueueStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "queue.CreateQueueDetails",
			},
			{
				SDKStruct: "queue.UpdateQueueDetails",
			},
			{
				SDKStruct:  "queue.Queue",
				APISurface: "status",
			},
			{
				SDKStruct: "queue.QueueCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "queue.QueueSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "Stats",
		SpecType:   reflect.TypeOf(queuev1beta1.StatsSpec{}),
		StatusType: reflect.TypeOf(queuev1beta1.StatsObservedState{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "queue.Stats",
			},
		},
	},
	{
		Name:       "WorkRequest",
		SpecType:   reflect.TypeOf(queuev1beta1.WorkRequestSpec{}),
		StatusType: reflect.TypeOf(queuev1beta1.WorkRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "queue.WorkRequest",
			},
			{
				SDKStruct: "queue.WorkRequestSummary",
			},
		},
	},
	{
		Name:       "WorkRequestError",
		SpecType:   reflect.TypeOf(queuev1beta1.WorkRequestErrorSpec{}),
		StatusType: reflect.TypeOf(queuev1beta1.WorkRequestErrorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "queue.WorkRequestError",
				APISurface: "status",
			},
			{
				SDKStruct: "queue.WorkRequestErrorCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "WorkRequestLog",
		SpecType:   reflect.TypeOf(queuev1beta1.WorkRequestLogSpec{}),
		StatusType: reflect.TypeOf(queuev1beta1.WorkRequestLogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "queue.WorkRequestLogEntry",
				APISurface: "status",
			},
			{
				SDKStruct: "queue.WorkRequestLogEntryCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "FunctionsApplication",
		SpecType:   reflect.TypeOf(functionsv1beta1.ApplicationSpec{}),
		StatusType: reflect.TypeOf(functionsv1beta1.ApplicationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "functions.CreateApplicationDetails",
			},
			{
				SDKStruct: "functions.UpdateApplicationDetails",
			},
			{
				SDKStruct:  "functions.Application",
				APISurface: "status",
			},
			{
				SDKStruct:  "functions.ApplicationSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "FunctionsFunction",
		SpecType:   reflect.TypeOf(functionsv1beta1.FunctionSpec{}),
		StatusType: reflect.TypeOf(functionsv1beta1.FunctionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "functions.CreateFunctionDetails",
			},
			{
				SDKStruct: "functions.UpdateFunctionDetails",
			},
			{
				SDKStruct:  "functions.Function",
				APISurface: "status",
			},
			{
				SDKStruct:  "functions.FunctionSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "FunctionsPbfListing",
		SpecType:   reflect.TypeOf(functionsv1beta1.PbfListingSpec{}),
		StatusType: reflect.TypeOf(functionsv1beta1.PbfListingStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "functions.PbfListing",
				APISurface: "status",
			},
			{
				SDKStruct: "functions.PbfListingVersionSummary",
				Exclude:   true,
				Reason:    "Intentionally untracked: version summaries belong to the dedicated PbfListingVersion status surface.",
			},
			{
				SDKStruct:  "functions.PbfListingSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "FunctionsPbfListingVersion",
		SpecType:   reflect.TypeOf(functionsv1beta1.PbfListingVersionSpec{}),
		StatusType: reflect.TypeOf(functionsv1beta1.PbfListingVersionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "functions.PbfListingVersion",
			},
			{
				SDKStruct: "functions.PbfListingVersionSummary",
			},
		},
	},
	{
		Name:       "FunctionsTrigger",
		SpecType:   reflect.TypeOf(functionsv1beta1.TriggerSpec{}),
		StatusType: reflect.TypeOf(functionsv1beta1.TriggerStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "functions.Trigger",
			},
			{
				SDKStruct: "functions.TriggerSummary",
			},
		},
	},
	{
		Name:       "NoSQLIndex",
		SpecType:   reflect.TypeOf(nosqlv1beta1.IndexSpec{}),
		StatusType: reflect.TypeOf(nosqlv1beta1.IndexStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "nosql.CreateIndexDetails",
			},
			{
				SDKStruct:  "nosql.Index",
				APISurface: "status",
			},
			{
				SDKStruct: "nosql.IndexCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "nosql.IndexSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "NoSQLReplica",
		SpecType:   reflect.TypeOf(nosqlv1beta1.ReplicaSpec{}),
		StatusType: reflect.TypeOf(nosqlv1beta1.ReplicaStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "nosql.CreateReplicaDetails",
			},
			{
				SDKStruct: "nosql.Replica",
			},
		},
	},
	{
		Name:       "NoSQLRow",
		SpecType:   reflect.TypeOf(nosqlv1beta1.RowSpec{}),
		StatusType: reflect.TypeOf(nosqlv1beta1.RowStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "nosql.UpdateRowDetails",
			},
			{
				SDKStruct: "nosql.Row",
			},
		},
	},
	{
		Name:       "NoSQLTable",
		SpecType:   reflect.TypeOf(nosqlv1beta1.TableSpec{}),
		StatusType: reflect.TypeOf(nosqlv1beta1.TableStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "nosql.CreateTableDetails",
			},
			{
				SDKStruct: "nosql.UpdateTableDetails",
			},
			{
				SDKStruct:  "nosql.Table",
				APISurface: "status",
			},
			{
				SDKStruct: "nosql.TableCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "nosql.TableSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "NoSQLTableUsage",
		SpecType:   reflect.TypeOf(nosqlv1beta1.TableUsageSpec{}),
		StatusType: reflect.TypeOf(nosqlv1beta1.TableUsageStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "nosql.TableUsageCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "nosql.TableUsageSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "NoSQLWorkRequest",
		SpecType:   reflect.TypeOf(nosqlv1beta1.WorkRequestSpec{}),
		StatusType: reflect.TypeOf(nosqlv1beta1.WorkRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "nosql.WorkRequest",
				APISurface: "status",
			},
			{
				SDKStruct: "nosql.WorkRequestCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "nosql.WorkRequestSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "NoSQLWorkRequestError",
		SpecType:   reflect.TypeOf(nosqlv1beta1.WorkRequestErrorSpec{}),
		StatusType: reflect.TypeOf(nosqlv1beta1.WorkRequestErrorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "nosql.WorkRequestError",
				APISurface: "status",
			},
			{
				SDKStruct: "nosql.WorkRequestErrorCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "NoSQLWorkRequestLog",
		SpecType:   reflect.TypeOf(nosqlv1beta1.WorkRequestLogSpec{}),
		StatusType: reflect.TypeOf(nosqlv1beta1.WorkRequestLogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "nosql.WorkRequestLogEntry",
				APISurface: "status",
			},
			{
				SDKStruct: "nosql.WorkRequestLogEntryCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "ObjectStorageBucket",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.BucketSpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.BucketStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.CreateBucketDetails",
			},
			{
				SDKStruct: "objectstorage.UpdateBucketDetails",
			},
			{
				SDKStruct:  "objectstorage.Bucket",
				APISurface: "status",
			},
			{
				SDKStruct:  "objectstorage.BucketSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "PSQLBackup",
		SpecType:   reflect.TypeOf(psqlv1beta1.BackupSpec{}),
		StatusType: reflect.TypeOf(psqlv1beta1.BackupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "psql.CreateBackupDetails",
			},
			{
				SDKStruct: "psql.UpdateBackupDetails",
			},
			{
				SDKStruct:  "psql.Backup",
				APISurface: "status",
			},
			{
				SDKStruct: "psql.BackupCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: transport wrapper type does not correspond to a top-level CRD spec or status surface.",
			},
			{
				SDKStruct:  "psql.BackupSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "PSQLConfiguration",
		SpecType:   reflect.TypeOf(psqlv1beta1.ConfigurationSpec{}),
		StatusType: reflect.TypeOf(psqlv1beta1.ConfigurationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "psql.CreateConfigurationDetails",
			},
			{
				SDKStruct: "psql.UpdateConfigurationDetails",
			},
			{
				SDKStruct: "psql.ConfigurationDetails",
				Exclude:   true,
				Reason:    "Intentionally untracked: transport wrapper type does not correspond to a top-level CRD spec or status surface.",
			},
			{
				SDKStruct:  "psql.Configuration",
				APISurface: "status",
			},
			{
				SDKStruct: "psql.ConfigurationCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: transport wrapper type does not correspond to a top-level CRD spec or status surface.",
			},
			{
				SDKStruct:  "psql.ConfigurationSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "PSQLConnectionDetail",
		SpecType:   reflect.TypeOf(psqlv1beta1.ConnectionDetailSpec{}),
		StatusType: reflect.TypeOf(psqlv1beta1.ConnectionDetailStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "psql.ConnectionDetails",
			},
		},
	},
	{
		Name:       "PSQLDbSystem",
		SpecType:   reflect.TypeOf(psqlv1beta1.DbSystemSpec{}),
		StatusType: reflect.TypeOf(psqlv1beta1.DbSystemStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "psql.CreateDbSystemDetails",
			},
			{
				SDKStruct: "psql.UpdateDbSystemDetails",
			},
			{
				SDKStruct: "psql.DbSystemDetails",
			},
			{
				SDKStruct:  "psql.DbSystem",
				APISurface: "status",
			},
			{
				SDKStruct: "psql.DbSystemCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: transport wrapper type does not correspond to a top-level CRD spec or status surface.",
			},
			{
				SDKStruct:  "psql.DbSystemSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "PSQLDbSystemDbInstance",
		SpecType:   reflect.TypeOf(psqlv1beta1.DbSystemDbInstanceSpec{}),
		StatusType: reflect.TypeOf(psqlv1beta1.DbSystemDbInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "psql.UpdateDbSystemDbInstanceDetails",
			},
		},
	},
	{
		Name:       "PSQLDefaultConfiguration",
		SpecType:   reflect.TypeOf(psqlv1beta1.DefaultConfigurationSpec{}),
		StatusType: reflect.TypeOf(psqlv1beta1.DefaultConfigurationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "psql.DefaultConfigurationDetails",
				Exclude:   true,
				Reason:    "Intentionally untracked: transport wrapper type does not correspond to a top-level CRD spec or status surface.",
			},
			{
				SDKStruct:  "psql.DefaultConfiguration",
				APISurface: "status",
			},
			{
				SDKStruct: "psql.DefaultConfigurationCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: transport wrapper type does not correspond to a top-level CRD spec or status surface.",
			},
			{
				SDKStruct:  "psql.DefaultConfigurationSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "PSQLPrimaryDbInstance",
		SpecType:   reflect.TypeOf(psqlv1beta1.PrimaryDbInstanceSpec{}),
		StatusType: reflect.TypeOf(psqlv1beta1.PrimaryDbInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "psql.PrimaryDbInstanceDetails",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "PSQLShape",
		SpecType:   reflect.TypeOf(psqlv1beta1.ShapeSpec{}),
		StatusType: reflect.TypeOf(psqlv1beta1.ShapeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "psql.ShapeCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: transport wrapper type does not correspond to a top-level CRD spec or status surface.",
			},
			{
				SDKStruct:  "psql.ShapeSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "PSQLWorkRequest",
		SpecType:   reflect.TypeOf(psqlv1beta1.WorkRequestSpec{}),
		StatusType: reflect.TypeOf(psqlv1beta1.WorkRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "psql.WorkRequest",
				APISurface: "status",
			},
			{
				SDKStruct:  "psql.WorkRequestSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "PSQLWorkRequestError",
		SpecType:   reflect.TypeOf(psqlv1beta1.WorkRequestErrorSpec{}),
		StatusType: reflect.TypeOf(psqlv1beta1.WorkRequestErrorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "psql.WorkRequestError",
				APISurface: "status",
			},
			{
				SDKStruct: "psql.WorkRequestErrorCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: transport wrapper type does not correspond to a top-level CRD spec or status surface.",
			},
		},
	},
	{
		Name:       "PSQLWorkRequestLog",
		SpecType:   reflect.TypeOf(psqlv1beta1.WorkRequestLogSpec{}),
		StatusType: reflect.TypeOf(psqlv1beta1.WorkRequestLogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "psql.WorkRequestLogEntry",
				APISurface: "status",
			},
			{
				SDKStruct: "psql.WorkRequestLogEntryCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: transport wrapper type does not correspond to a top-level CRD spec or status surface.",
			},
		},
	},
	{
		Name:       "ContainerEngineAddon",
		SpecType:   reflect.TypeOf(containerenginev1beta1.AddonSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.AddonStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.UpdateAddonDetails",
			},
			{
				SDKStruct: "containerengine.Addon",
			},
			{
				SDKStruct: "containerengine.AddonSummary",
			},
		},
	},
	{
		Name:       "ContainerEngineAddonOption",
		SpecType:   reflect.TypeOf(containerenginev1beta1.AddonOptionSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.AddonOptionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.AddonOptionSummary",
			},
		},
	},
	{
		Name:       "ContainerEngineCluster",
		SpecType:   reflect.TypeOf(containerenginev1beta1.ClusterSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.ClusterStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.CreateClusterDetails",
			},
			{
				SDKStruct: "containerengine.UpdateClusterDetails",
			},
			{
				SDKStruct:  "containerengine.Cluster",
				APISurface: "status",
			},
			{
				SDKStruct:  "containerengine.ClusterSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "ContainerEngineClusterEndpointConfig",
		SpecType:   reflect.TypeOf(containerenginev1beta1.ClusterEndpointConfigSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.ClusterEndpointConfigStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.CreateClusterEndpointConfigDetails",
			},
			{
				SDKStruct: "containerengine.UpdateClusterEndpointConfigDetails",
			},
			{
				SDKStruct:  "containerengine.ClusterEndpointConfig",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "ContainerEngineClusterMigrateToNativeVcnStatus",
		SpecType:   reflect.TypeOf(containerenginev1beta1.ClusterMigrateToNativeVcnStatusSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.ClusterMigrateToNativeVcnStatusObservedState{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.ClusterMigrateToNativeVcnStatus",
			},
		},
	},
	{
		Name:       "ContainerEngineClusterOption",
		SpecType:   reflect.TypeOf(containerenginev1beta1.ClusterOptionSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.ClusterOptionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.ClusterOptions",
			},
		},
	},
	{
		Name:       "ContainerEngineCredentialRotationStatus",
		SpecType:   reflect.TypeOf(containerenginev1beta1.CredentialRotationStatusSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.CredentialRotationStatusObservedState{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.CredentialRotationStatus",
			},
		},
	},
	{
		Name:       "ContainerEngineKubeconfig",
		SpecType:   reflect.TypeOf(containerenginev1beta1.KubeconfigSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.KubeconfigStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.CreateClusterKubeconfigContentDetails",
			},
		},
	},
	{
		Name:       "ContainerEngineNode",
		SpecType:   reflect.TypeOf(containerenginev1beta1.NodeSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.NodeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.Node",
			},
		},
	},
	{
		Name:       "ContainerEngineNodePool",
		SpecType:   reflect.TypeOf(containerenginev1beta1.NodePoolSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.NodePoolStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.CreateNodePoolDetails",
			},
			{
				SDKStruct: "containerengine.UpdateNodePoolDetails",
			},
			{
				SDKStruct:  "containerengine.NodePool",
				APISurface: "status",
			},
			{
				SDKStruct:  "containerengine.NodePoolSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "ContainerEngineNodePoolOption",
		SpecType:   reflect.TypeOf(containerenginev1beta1.NodePoolOptionSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.NodePoolOptionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.NodePoolOptions",
			},
		},
	},
	{
		Name:       "ContainerEnginePodShape",
		SpecType:   reflect.TypeOf(containerenginev1beta1.PodShapeSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.PodShapeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.PodShape",
			},
			{
				SDKStruct: "containerengine.PodShapeSummary",
			},
		},
	},
	{
		Name:       "ContainerEngineVirtualNode",
		SpecType:   reflect.TypeOf(containerenginev1beta1.VirtualNodeSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.VirtualNodeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.VirtualNode",
			},
			{
				SDKStruct: "containerengine.VirtualNodeSummary",
			},
		},
	},
	{
		Name:       "ContainerEngineVirtualNodePool",
		SpecType:   reflect.TypeOf(containerenginev1beta1.VirtualNodePoolSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.VirtualNodePoolStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.CreateVirtualNodePoolDetails",
			},
			{
				SDKStruct: "containerengine.UpdateVirtualNodePoolDetails",
			},
			{
				SDKStruct:  "containerengine.VirtualNodePool",
				APISurface: "status",
			},
			{
				SDKStruct:  "containerengine.VirtualNodePoolSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "ContainerEngineWorkRequest",
		SpecType:   reflect.TypeOf(containerenginev1beta1.WorkRequestSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.WorkRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.WorkRequest",
			},
			{
				SDKStruct: "containerengine.WorkRequestSummary",
			},
		},
	},
	{
		Name:       "ContainerEngineWorkRequestError",
		SpecType:   reflect.TypeOf(containerenginev1beta1.WorkRequestErrorSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.WorkRequestErrorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.WorkRequestError",
			},
		},
	},
	{
		Name:       "ContainerEngineWorkRequestLog",
		SpecType:   reflect.TypeOf(containerenginev1beta1.WorkRequestLogSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.WorkRequestLogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.WorkRequestLogEntry",
			},
		},
	},
	{
		Name:       "ContainerEngineWorkloadMapping",
		SpecType:   reflect.TypeOf(containerenginev1beta1.WorkloadMappingSpec{}),
		StatusType: reflect.TypeOf(containerenginev1beta1.WorkloadMappingStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerengine.CreateWorkloadMappingDetails",
			},
			{
				SDKStruct: "containerengine.UpdateWorkloadMappingDetails",
			},
			{
				SDKStruct: "containerengine.WorkloadMapping",
			},
			{
				SDKStruct: "containerengine.WorkloadMappingSummary",
			},
		},
	},
	{
		Name:       "IdentityCompartment",
		SpecType:   reflect.TypeOf(identityv1beta1.CompartmentSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.CompartmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateCompartmentDetails",
			},
			{
				SDKStruct: "identity.UpdateCompartmentDetails",
			},
			{
				SDKStruct: "identity.Compartment",
			},
		},
	},
	{
		Name:       "VaultSecret",
		SpecType:   reflect.TypeOf(vaultv1beta1.SecretSpec{}),
		StatusType: reflect.TypeOf(vaultv1beta1.SecretStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "vault.CreateSecretDetails",
			},
			{
				SDKStruct: "vault.UpdateSecretDetails",
			},
			{
				SDKStruct:  "vault.Secret",
				APISurface: "status",
			},
			{
				SDKStruct: "vault.SecretVersionSummary",
				Exclude:   true,
				Reason:    "Intentionally untracked: secret version summaries belong to the dedicated VaultSecretVersion status surface.",
			},
			{
				SDKStruct:  "vault.SecretSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "VaultSecretVersion",
		SpecType:   reflect.TypeOf(vaultv1beta1.SecretVersionSpec{}),
		StatusType: reflect.TypeOf(vaultv1beta1.SecretVersionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "vault.SecretVersion",
			},
			{
				SDKStruct: "vault.SecretVersionSummary",
			},
		},
	},
	{
		Name:       "CoreAllDrgAttachment",
		SpecType:   reflect.TypeOf(corev1beta1.AllDrgAttachmentSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.AllDrgAttachmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.DrgAttachmentInfo",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreAllowedIkeIPSecParameter",
		SpecType:   reflect.TypeOf(corev1beta1.AllowedIkeIPSecParameterSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.AllowedIkeIPSecParameterStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.AllowedIkeIpSecParameters",
			},
		},
	},
	{
		Name:       "CoreAllowedPeerRegionsForRemotePeering",
		SpecType:   reflect.TypeOf(corev1beta1.AllowedPeerRegionsForRemotePeeringSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.AllowedPeerRegionsForRemotePeeringStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.PeerRegionForRemotePeering",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreAppCatalogListing",
		SpecType:   reflect.TypeOf(corev1beta1.AppCatalogListingSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.AppCatalogListingStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.AppCatalogListing",
			},
			{
				SDKStruct: "core.AppCatalogListingSummary",
			},
		},
	},
	{
		Name:       "CoreAppCatalogListingAgreement",
		SpecType:   reflect.TypeOf(corev1beta1.AppCatalogListingAgreementSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.AppCatalogListingAgreementStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.AppCatalogListingResourceVersionAgreements",
			},
		},
	},
	{
		Name:       "CoreAppCatalogListingResourceVersion",
		SpecType:   reflect.TypeOf(corev1beta1.AppCatalogListingResourceVersionSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.AppCatalogListingResourceVersionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.AppCatalogListingResourceVersion",
			},
			{
				SDKStruct: "core.AppCatalogListingResourceVersionSummary",
			},
		},
	},
	{
		Name:       "CoreAppCatalogSubscription",
		SpecType:   reflect.TypeOf(corev1beta1.AppCatalogSubscriptionSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.AppCatalogSubscriptionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateAppCatalogSubscriptionDetails",
			},
			{
				SDKStruct: "core.AppCatalogSubscription",
			},
			{
				SDKStruct: "core.AppCatalogSubscriptionSummary",
			},
		},
	},
	{
		Name:       "CoreBlockVolumeReplica",
		SpecType:   reflect.TypeOf(corev1beta1.BlockVolumeReplicaSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.BlockVolumeReplicaStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.BlockVolumeReplicaDetails",
			},
			{
				SDKStruct: "core.BlockVolumeReplica",
			},
		},
	},
	{
		Name:       "CoreBootVolume",
		SpecType:   reflect.TypeOf(corev1beta1.BootVolumeSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.BootVolumeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateBootVolumeDetails",
			},
			{
				SDKStruct: "core.UpdateBootVolumeDetails",
			},
			{
				SDKStruct:  "core.BootVolume",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreBootVolumeAttachment",
		SpecType:   reflect.TypeOf(corev1beta1.BootVolumeAttachmentSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.BootVolumeAttachmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.BootVolumeAttachment",
			},
		},
	},
	{
		Name:       "CoreBootVolumeBackup",
		SpecType:   reflect.TypeOf(corev1beta1.BootVolumeBackupSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.BootVolumeBackupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateBootVolumeBackupDetails",
			},
			{
				SDKStruct: "core.UpdateBootVolumeBackupDetails",
			},
			{
				SDKStruct: "core.BootVolumeBackup",
			},
		},
	},
	{
		Name:       "CoreBootVolumeKmsKey",
		SpecType:   reflect.TypeOf(corev1beta1.BootVolumeKmsKeySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.BootVolumeKmsKeyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.UpdateBootVolumeKmsKeyDetails",
			},
			{
				SDKStruct:  "core.BootVolumeKmsKey",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "CoreBootVolumeReplica",
		SpecType:   reflect.TypeOf(corev1beta1.BootVolumeReplicaSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.BootVolumeReplicaStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.BootVolumeReplicaDetails",
			},
			{
				SDKStruct: "core.BootVolumeReplica",
			},
		},
	},
	{
		Name:       "CoreByoipAllocatedRange",
		SpecType:   reflect.TypeOf(corev1beta1.ByoipAllocatedRangeSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ByoipAllocatedRangeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.ByoipAllocatedRangeCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "core.ByoipAllocatedRangeSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreByoipRange",
		SpecType:   reflect.TypeOf(corev1beta1.ByoipRangeSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ByoipRangeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateByoipRangeDetails",
			},
			{
				SDKStruct: "core.UpdateByoipRangeDetails",
			},
			{
				SDKStruct:  "core.ByoipRange",
				APISurface: "status",
			},
			{
				SDKStruct: "core.ByoipRangeCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "core.ByoipRangeSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreCaptureFilter",
		SpecType:   reflect.TypeOf(corev1beta1.CaptureFilterSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.CaptureFilterStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateCaptureFilterDetails",
			},
			{
				SDKStruct: "core.UpdateCaptureFilterDetails",
			},
			{
				SDKStruct:  "core.CaptureFilter",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreClusterNetwork",
		SpecType:   reflect.TypeOf(corev1beta1.ClusterNetworkSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ClusterNetworkStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateClusterNetworkDetails",
			},
			{
				SDKStruct: "core.UpdateClusterNetworkDetails",
			},
			{
				SDKStruct:  "core.ClusterNetwork",
				APISurface: "status",
			},
			{
				SDKStruct:  "core.ClusterNetworkSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreClusterNetworkInstance",
		SpecType:   reflect.TypeOf(corev1beta1.ClusterNetworkInstanceSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ClusterNetworkInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.InstanceSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreComputeCapacityReport",
		SpecType:   reflect.TypeOf(corev1beta1.ComputeCapacityReportSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ComputeCapacityReportStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateComputeCapacityReportDetails",
			},
			{
				SDKStruct:  "core.ComputeCapacityReport",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreComputeCapacityReservation",
		SpecType:   reflect.TypeOf(corev1beta1.ComputeCapacityReservationSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ComputeCapacityReservationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateComputeCapacityReservationDetails",
			},
			{
				SDKStruct: "core.UpdateComputeCapacityReservationDetails",
			},
			{
				SDKStruct:  "core.ComputeCapacityReservation",
				APISurface: "status",
			},
			{
				SDKStruct:  "core.ComputeCapacityReservationSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreComputeCapacityReservationInstance",
		SpecType:   reflect.TypeOf(corev1beta1.ComputeCapacityReservationInstanceSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ComputeCapacityReservationInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.CapacityReservationInstanceSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreComputeCapacityReservationInstanceShape",
		SpecType:   reflect.TypeOf(corev1beta1.ComputeCapacityReservationInstanceShapeSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ComputeCapacityReservationInstanceShapeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.ComputeCapacityReservationInstanceShapeSummary",
			},
		},
	},
	{
		Name:       "CoreComputeCapacityTopology",
		SpecType:   reflect.TypeOf(corev1beta1.ComputeCapacityTopologySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ComputeCapacityTopologyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateComputeCapacityTopologyDetails",
			},
			{
				SDKStruct: "core.UpdateComputeCapacityTopologyDetails",
			},
			{
				SDKStruct:  "core.ComputeCapacityTopology",
				APISurface: "status",
			},
			{
				SDKStruct: "core.ComputeCapacityTopologyCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "core.ComputeCapacityTopologySummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreComputeCapacityTopologyComputeBareMetalHost",
		SpecType:   reflect.TypeOf(corev1beta1.ComputeCapacityTopologyComputeBareMetalHostSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ComputeCapacityTopologyComputeBareMetalHostStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.ComputeBareMetalHostCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "CoreComputeCapacityTopologyComputeHpcIsland",
		SpecType:   reflect.TypeOf(corev1beta1.ComputeCapacityTopologyComputeHpcIslandSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ComputeCapacityTopologyComputeHpcIslandStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.ComputeHpcIslandCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "CoreComputeCapacityTopologyComputeNetworkBlock",
		SpecType:   reflect.TypeOf(corev1beta1.ComputeCapacityTopologyComputeNetworkBlockSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ComputeCapacityTopologyComputeNetworkBlockStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.ComputeNetworkBlockCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "CoreComputeCluster",
		SpecType:   reflect.TypeOf(corev1beta1.ComputeClusterSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ComputeClusterStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateComputeClusterDetails",
			},
			{
				SDKStruct: "core.UpdateComputeClusterDetails",
			},
			{
				SDKStruct:  "core.ComputeCluster",
				APISurface: "status",
			},
			{
				SDKStruct: "core.ComputeClusterCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "core.ComputeClusterSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreComputeGlobalImageCapabilitySchema",
		SpecType:   reflect.TypeOf(corev1beta1.ComputeGlobalImageCapabilitySchemaSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ComputeGlobalImageCapabilitySchemaStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.ComputeGlobalImageCapabilitySchema",
			},
			{
				SDKStruct: "core.ComputeGlobalImageCapabilitySchemaVersionSummary",
			},
			{
				SDKStruct: "core.ComputeGlobalImageCapabilitySchemaSummary",
			},
		},
	},
	{
		Name:       "CoreComputeGlobalImageCapabilitySchemaVersion",
		SpecType:   reflect.TypeOf(corev1beta1.ComputeGlobalImageCapabilitySchemaVersionSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ComputeGlobalImageCapabilitySchemaVersionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.ComputeGlobalImageCapabilitySchemaVersion",
			},
			{
				SDKStruct: "core.ComputeGlobalImageCapabilitySchemaVersionSummary",
			},
		},
	},
	{
		Name:       "CoreComputeImageCapabilitySchema",
		SpecType:   reflect.TypeOf(corev1beta1.ComputeImageCapabilitySchemaSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ComputeImageCapabilitySchemaStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateComputeImageCapabilitySchemaDetails",
			},
			{
				SDKStruct: "core.UpdateComputeImageCapabilitySchemaDetails",
			},
			{
				SDKStruct:  "core.ComputeImageCapabilitySchema",
				APISurface: "status",
			},
			{
				SDKStruct:  "core.ComputeImageCapabilitySchemaSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreConsoleHistory",
		SpecType:   reflect.TypeOf(corev1beta1.ConsoleHistorySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ConsoleHistoryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.UpdateConsoleHistoryDetails",
			},
			{
				SDKStruct: "core.ConsoleHistory",
			},
		},
	},
	{
		Name:        "CoreConsoleHistoryContent",
		SpecType:    reflect.TypeOf(corev1beta1.ConsoleHistoryContentSpec{}),
		StatusType:  reflect.TypeOf(corev1beta1.ConsoleHistoryContentStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:       "CoreCpe",
		SpecType:   reflect.TypeOf(corev1beta1.CpeSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.CpeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateCpeDetails",
			},
			{
				SDKStruct: "core.UpdateCpeDetails",
			},
			{
				SDKStruct:  "core.Cpe",
				APISurface: "status",
			},
		},
	},
	{
		Name:        "CoreCpeDeviceConfigContent",
		SpecType:    reflect.TypeOf(corev1beta1.CpeDeviceConfigContentSpec{}),
		StatusType:  reflect.TypeOf(corev1beta1.CpeDeviceConfigContentStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:       "CoreCpeDeviceShape",
		SpecType:   reflect.TypeOf(corev1beta1.CpeDeviceShapeSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.CpeDeviceShapeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CpeDeviceShapeSummary",
			},
		},
	},
	{
		Name:       "CoreCrossConnect",
		SpecType:   reflect.TypeOf(corev1beta1.CrossConnectSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.CrossConnectObservedState{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateCrossConnectDetails",
			},
			{
				SDKStruct: "core.UpdateCrossConnectDetails",
			},
			{
				SDKStruct:  "core.CrossConnect",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreCrossConnectGroup",
		SpecType:   reflect.TypeOf(corev1beta1.CrossConnectGroupSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.CrossConnectGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateCrossConnectGroupDetails",
			},
			{
				SDKStruct: "core.UpdateCrossConnectGroupDetails",
			},
			{
				SDKStruct:  "core.CrossConnectGroup",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreCrossConnectLetterOfAuthority",
		SpecType:   reflect.TypeOf(corev1beta1.CrossConnectLetterOfAuthoritySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.CrossConnectLetterOfAuthorityStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.LetterOfAuthority",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreCrossConnectLocation",
		SpecType:   reflect.TypeOf(corev1beta1.CrossConnectLocationSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.CrossConnectLocationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CrossConnectLocation",
			},
		},
	},
	{
		Name:       "CoreCrossConnectMapping",
		SpecType:   reflect.TypeOf(corev1beta1.CrossConnectMappingSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.CrossConnectMappingStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CrossConnectMappingDetails",
			},
			{
				SDKStruct: "core.CrossConnectMapping",
			},
		},
	},
	{
		Name:       "CoreCrossConnectStatus",
		SpecType:   reflect.TypeOf(corev1beta1.CrossConnectStatusSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.CrossConnectStatusObservedState{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CrossConnectStatus",
			},
		},
	},
	{
		Name:       "CoreCrossconnectPortSpeedShape",
		SpecType:   reflect.TypeOf(corev1beta1.CrossconnectPortSpeedShapeSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.CrossconnectPortSpeedShapeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CrossConnectPortSpeedShape",
			},
		},
	},
	{
		Name:       "CoreDedicatedVmHost",
		SpecType:   reflect.TypeOf(corev1beta1.DedicatedVmHostSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.DedicatedVmHostStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateDedicatedVmHostDetails",
			},
			{
				SDKStruct: "core.UpdateDedicatedVmHostDetails",
			},
			{
				SDKStruct: "core.DedicatedVmHost",
			},
			{
				SDKStruct: "core.DedicatedVmHostSummary",
			},
		},
	},
	{
		Name:       "CoreDedicatedVmHostInstance",
		SpecType:   reflect.TypeOf(corev1beta1.DedicatedVmHostInstanceSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.DedicatedVmHostInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.DedicatedVmHostInstanceSummary",
			},
		},
	},
	{
		Name:       "CoreDedicatedVmHostInstanceShape",
		SpecType:   reflect.TypeOf(corev1beta1.DedicatedVmHostInstanceShapeSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.DedicatedVmHostInstanceShapeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.DedicatedVmHostInstanceShapeSummary",
			},
		},
	},
	{
		Name:       "CoreDedicatedVmHostShape",
		SpecType:   reflect.TypeOf(corev1beta1.DedicatedVmHostShapeSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.DedicatedVmHostShapeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.DedicatedVmHostShapeSummary",
			},
		},
	},
	{
		Name:       "CoreDhcpOption",
		SpecType:   reflect.TypeOf(corev1beta1.DhcpOptionSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.DhcpOptionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.DhcpOptions",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreDrg",
		SpecType:   reflect.TypeOf(corev1beta1.DrgSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.DrgStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateDrgDetails",
			},
			{
				SDKStruct: "core.UpdateDrgDetails",
			},
			{
				SDKStruct:  "core.Drg",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreDrgAttachment",
		SpecType:   reflect.TypeOf(corev1beta1.DrgAttachmentSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.DrgAttachmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateDrgAttachmentDetails",
			},
			{
				SDKStruct: "core.UpdateDrgAttachmentDetails",
			},
			{
				SDKStruct:  "core.DrgAttachment",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreDrgRedundancyStatus",
		SpecType:   reflect.TypeOf(corev1beta1.DrgRedundancyStatusSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.DrgRedundancyStatusObservedState{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.DrgRedundancyStatus",
			},
		},
	},
	{
		Name:       "CoreDrgRouteDistribution",
		SpecType:   reflect.TypeOf(corev1beta1.DrgRouteDistributionSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.DrgRouteDistributionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateDrgRouteDistributionDetails",
			},
			{
				SDKStruct: "core.UpdateDrgRouteDistributionDetails",
			},
			{
				SDKStruct:  "core.DrgRouteDistribution",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreDrgRouteDistributionStatement",
		SpecType:   reflect.TypeOf(corev1beta1.DrgRouteDistributionStatementSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.DrgRouteDistributionStatementStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.UpdateDrgRouteDistributionStatementDetails",
			},
			{
				SDKStruct:  "core.DrgRouteDistributionStatement",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreDrgRouteRule",
		SpecType:   reflect.TypeOf(corev1beta1.DrgRouteRuleSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.DrgRouteRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.UpdateDrgRouteRuleDetails",
			},
			{
				SDKStruct: "core.DrgRouteRule",
			},
		},
	},
	{
		Name:       "CoreDrgRouteTable",
		SpecType:   reflect.TypeOf(corev1beta1.DrgRouteTableSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.DrgRouteTableStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateDrgRouteTableDetails",
			},
			{
				SDKStruct: "core.UpdateDrgRouteTableDetails",
			},
			{
				SDKStruct:  "core.DrgRouteTable",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreFastConnectProviderService",
		SpecType:   reflect.TypeOf(corev1beta1.FastConnectProviderServiceSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.FastConnectProviderServiceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.FastConnectProviderService",
			},
		},
	},
	{
		Name:       "CoreFastConnectProviderServiceKey",
		SpecType:   reflect.TypeOf(corev1beta1.FastConnectProviderServiceKeySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.FastConnectProviderServiceKeyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.FastConnectProviderServiceKey",
			},
		},
	},
	{
		Name:       "CoreFastConnectProviderVirtualCircuitBandwidthShape",
		SpecType:   reflect.TypeOf(corev1beta1.FastConnectProviderVirtualCircuitBandwidthShapeSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.FastConnectProviderVirtualCircuitBandwidthShapeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.VirtualCircuitBandwidthShape",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreIPSecConnection",
		SpecType:   reflect.TypeOf(corev1beta1.IPSecConnectionSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.IPSecConnectionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateIpSecConnectionDetails",
			},
			{
				SDKStruct: "core.UpdateIpSecConnectionDetails",
			},
			{
				SDKStruct:  "core.IpSecConnection",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreIPSecConnectionDeviceConfig",
		SpecType:   reflect.TypeOf(corev1beta1.IPSecConnectionDeviceConfigSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.IPSecConnectionDeviceConfigStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.IpSecConnectionDeviceConfig",
			},
		},
	},
	{
		Name:       "CoreIPSecConnectionDeviceStatus",
		SpecType:   reflect.TypeOf(corev1beta1.IPSecConnectionDeviceStatusSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.IPSecConnectionDeviceStatusObservedState{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.IpSecConnectionDeviceStatus",
			},
		},
	},
	{
		Name:       "CoreIPSecConnectionTunnel",
		SpecType:   reflect.TypeOf(corev1beta1.IPSecConnectionTunnelSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.IPSecConnectionTunnelStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateIpSecConnectionTunnelDetails",
			},
			{
				SDKStruct: "core.UpdateIpSecConnectionTunnelDetails",
			},
			{
				SDKStruct: "core.IpSecConnectionTunnel",
			},
		},
	},
	{
		Name:       "CoreIPSecConnectionTunnelError",
		SpecType:   reflect.TypeOf(corev1beta1.IPSecConnectionTunnelErrorSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.IPSecConnectionTunnelErrorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.IpSecConnectionTunnelErrorDetails",
			},
		},
	},
	{
		Name:       "CoreIPSecConnectionTunnelRoute",
		SpecType:   reflect.TypeOf(corev1beta1.IPSecConnectionTunnelRouteSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.IPSecConnectionTunnelRouteStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.TunnelRouteSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreIPSecConnectionTunnelSecurityAssociation",
		SpecType:   reflect.TypeOf(corev1beta1.IPSecConnectionTunnelSecurityAssociationSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.IPSecConnectionTunnelSecurityAssociationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.TunnelSecurityAssociationSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreIPSecConnectionTunnelSharedSecret",
		SpecType:   reflect.TypeOf(corev1beta1.IPSecConnectionTunnelSharedSecretSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.IPSecConnectionTunnelSharedSecretStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.UpdateIpSecConnectionTunnelSharedSecretDetails",
			},
			{
				SDKStruct:  "core.IpSecConnectionTunnelSharedSecret",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "CoreImage",
		SpecType:   reflect.TypeOf(corev1beta1.ImageSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ImageStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateImageDetails",
			},
			{
				SDKStruct: "core.UpdateImageDetails",
			},
			{
				SDKStruct: "core.Image",
			},
		},
	},
	{
		Name:       "CoreImageShapeCompatibilityEntry",
		SpecType:   reflect.TypeOf(corev1beta1.ImageShapeCompatibilityEntrySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ImageShapeCompatibilityEntryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.ImageShapeCompatibilityEntry",
			},
		},
	},
	{
		Name:       "CoreInstance",
		SpecType:   reflect.TypeOf(corev1beta1.InstanceSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.InstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.UpdateInstanceDetails",
			},
			{
				SDKStruct:  "core.Instance",
				APISurface: "status",
			},
			{
				SDKStruct:  "core.InstanceSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreInstanceConfiguration",
		SpecType:   reflect.TypeOf(corev1beta1.InstanceConfigurationSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.InstanceConfigurationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateInstanceConfigurationDetails",
			},
			{
				SDKStruct: "core.UpdateInstanceConfigurationDetails",
			},
			{
				SDKStruct:  "core.InstanceConfiguration",
				APISurface: "status",
			},
			{
				SDKStruct:  "core.InstanceConfigurationSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreInstanceConsoleConnection",
		SpecType:   reflect.TypeOf(corev1beta1.InstanceConsoleConnectionSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.InstanceConsoleConnectionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateInstanceConsoleConnectionDetails",
			},
			{
				SDKStruct: "core.UpdateInstanceConsoleConnectionDetails",
			},
			{
				SDKStruct: "core.InstanceConsoleConnection",
			},
		},
	},
	{
		Name:       "CoreInstanceDevice",
		SpecType:   reflect.TypeOf(corev1beta1.InstanceDeviceSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.InstanceDeviceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.Device",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreInstanceMaintenanceReboot",
		SpecType:   reflect.TypeOf(corev1beta1.InstanceMaintenanceRebootSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.InstanceMaintenanceRebootStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.InstanceMaintenanceReboot",
			},
		},
	},
	{
		Name:       "CoreInstancePool",
		SpecType:   reflect.TypeOf(corev1beta1.InstancePoolSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.InstancePoolStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateInstancePoolDetails",
			},
			{
				SDKStruct: "core.UpdateInstancePoolDetails",
			},
			{
				SDKStruct:  "core.InstancePool",
				APISurface: "status",
			},
			{
				SDKStruct:  "core.InstancePoolSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreInstancePoolInstance",
		SpecType:   reflect.TypeOf(corev1beta1.InstancePoolInstanceSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.InstancePoolInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.InstancePoolInstance",
			},
		},
	},
	{
		Name:       "CoreInstancePoolLoadBalancerAttachment",
		SpecType:   reflect.TypeOf(corev1beta1.InstancePoolLoadBalancerAttachmentSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.InstancePoolLoadBalancerAttachmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.InstancePoolLoadBalancerAttachment",
			},
		},
	},
	{
		Name:       "CoreInternetGateway",
		SpecType:   reflect.TypeOf(corev1beta1.InternetGatewaySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.InternetGatewayStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateInternetGatewayDetails",
			},
			{
				SDKStruct: "core.UpdateInternetGatewayDetails",
			},
			{
				SDKStruct:  "core.InternetGateway",
				APISurface: "status",
			},
		},
	},
	{
		Name:        "CoreIpsecCpeDeviceConfigContent",
		SpecType:    reflect.TypeOf(corev1beta1.IpsecCpeDeviceConfigContentSpec{}),
		StatusType:  reflect.TypeOf(corev1beta1.IpsecCpeDeviceConfigContentStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:       "CoreIpv6",
		SpecType:   reflect.TypeOf(corev1beta1.Ipv6Spec{}),
		StatusType: reflect.TypeOf(corev1beta1.Ipv6Status{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateIpv6Details",
			},
			{
				SDKStruct: "core.UpdateIpv6Details",
			},
			{
				SDKStruct: "core.Ipv6",
			},
		},
	},
	{
		Name:       "CoreLocalPeeringGateway",
		SpecType:   reflect.TypeOf(corev1beta1.LocalPeeringGatewaySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.LocalPeeringGatewayStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateLocalPeeringGatewayDetails",
			},
			{
				SDKStruct: "core.UpdateLocalPeeringGatewayDetails",
			},
			{
				SDKStruct: "core.LocalPeeringGateway",
			},
		},
	},
	{
		Name:       "CoreMeasuredBootReport",
		SpecType:   reflect.TypeOf(corev1beta1.MeasuredBootReportSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.MeasuredBootReportStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.MeasuredBootReport",
			},
		},
	},
	{
		Name:       "CoreNatGateway",
		SpecType:   reflect.TypeOf(corev1beta1.NatGatewaySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.NatGatewayStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateNatGatewayDetails",
			},
			{
				SDKStruct: "core.UpdateNatGatewayDetails",
			},
			{
				SDKStruct:  "core.NatGateway",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreNetworkSecurityGroup",
		SpecType:   reflect.TypeOf(corev1beta1.NetworkSecurityGroupSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.NetworkSecurityGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateNetworkSecurityGroupDetails",
			},
			{
				SDKStruct: "core.UpdateNetworkSecurityGroupDetails",
			},
			{
				SDKStruct:  "core.NetworkSecurityGroup",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreNetworkSecurityGroupSecurityRule",
		SpecType:   reflect.TypeOf(corev1beta1.NetworkSecurityGroupSecurityRuleSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.NetworkSecurityGroupSecurityRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.SecurityRule",
			},
		},
	},
	{
		Name:       "CoreNetworkSecurityGroupVnic",
		SpecType:   reflect.TypeOf(corev1beta1.NetworkSecurityGroupVnicSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.NetworkSecurityGroupVnicStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.NetworkSecurityGroupVnic",
			},
		},
	},
	{
		Name:       "CoreNetworkingTopology",
		SpecType:   reflect.TypeOf(corev1beta1.NetworkingTopologySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.NetworkingTopologyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.NetworkingTopology",
			},
		},
	},
	{
		Name:       "CorePrivateIp",
		SpecType:   reflect.TypeOf(corev1beta1.PrivateIpSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.PrivateIpStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreatePrivateIpDetails",
			},
			{
				SDKStruct: "core.UpdatePrivateIpDetails",
			},
			{
				SDKStruct:  "core.PrivateIp",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CorePublicIp",
		SpecType:   reflect.TypeOf(corev1beta1.PublicIpSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.PublicIpStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreatePublicIpDetails",
			},
			{
				SDKStruct: "core.UpdatePublicIpDetails",
			},
			{
				SDKStruct: "core.PublicIp",
			},
		},
	},
	{
		Name:       "CorePublicIpByIpAddress",
		SpecType:   reflect.TypeOf(corev1beta1.PublicIpByIpAddressSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.PublicIpByIpAddressStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.GetPublicIpByIpAddressDetails",
			},
		},
	},
	{
		Name:       "CorePublicIpByPrivateIpId",
		SpecType:   reflect.TypeOf(corev1beta1.PublicIpByPrivateIpIdSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.PublicIpByPrivateIpIdStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.GetPublicIpByPrivateIpIdDetails",
			},
		},
	},
	{
		Name:       "CorePublicIpPool",
		SpecType:   reflect.TypeOf(corev1beta1.PublicIpPoolSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.PublicIpPoolStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreatePublicIpPoolDetails",
			},
			{
				SDKStruct: "core.UpdatePublicIpPoolDetails",
			},
			{
				SDKStruct:  "core.PublicIpPool",
				APISurface: "status",
			},
			{
				SDKStruct: "core.PublicIpPoolCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "core.PublicIpPoolSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreRemotePeeringConnection",
		SpecType:   reflect.TypeOf(corev1beta1.RemotePeeringConnectionSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.RemotePeeringConnectionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateRemotePeeringConnectionDetails",
			},
			{
				SDKStruct: "core.UpdateRemotePeeringConnectionDetails",
			},
			{
				SDKStruct: "core.RemotePeeringConnection",
			},
		},
	},
	{
		Name:       "CoreRouteTable",
		SpecType:   reflect.TypeOf(corev1beta1.RouteTableSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.RouteTableStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateRouteTableDetails",
			},
			{
				SDKStruct: "core.UpdateRouteTableDetails",
			},
			{
				SDKStruct:  "core.RouteTable",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreSecurityList",
		SpecType:   reflect.TypeOf(corev1beta1.SecurityListSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.SecurityListStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateSecurityListDetails",
			},
			{
				SDKStruct: "core.UpdateSecurityListDetails",
			},
			{
				SDKStruct:  "core.SecurityList",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreService",
		SpecType:   reflect.TypeOf(corev1beta1.ServiceSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ServiceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.Service",
			},
		},
	},
	{
		Name:       "CoreServiceGateway",
		SpecType:   reflect.TypeOf(corev1beta1.ServiceGatewaySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ServiceGatewayStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateServiceGatewayDetails",
			},
			{
				SDKStruct: "core.UpdateServiceGatewayDetails",
			},
			{
				SDKStruct:  "core.ServiceGateway",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreShape",
		SpecType:   reflect.TypeOf(corev1beta1.ShapeSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ShapeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.Shape",
			},
		},
	},
	{
		Name:       "CoreSubnet",
		SpecType:   reflect.TypeOf(corev1beta1.SubnetSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.SubnetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateSubnetDetails",
			},
			{
				SDKStruct: "core.UpdateSubnetDetails",
			},
			{
				SDKStruct:  "core.Subnet",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreSubnetTopology",
		SpecType:   reflect.TypeOf(corev1beta1.SubnetTopologySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.SubnetTopologyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.SubnetTopology",
			},
		},
	},
	{
		Name:       "CoreTunnelCpeDeviceConfig",
		SpecType:   reflect.TypeOf(corev1beta1.TunnelCpeDeviceConfigSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.TunnelCpeDeviceConfigStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.UpdateTunnelCpeDeviceConfigDetails",
			},
			{
				SDKStruct: "core.TunnelCpeDeviceConfig",
			},
		},
	},
	{
		Name:        "CoreTunnelCpeDeviceConfigContent",
		SpecType:    reflect.TypeOf(corev1beta1.TunnelCpeDeviceConfigContentSpec{}),
		StatusType:  reflect.TypeOf(corev1beta1.TunnelCpeDeviceConfigContentStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:       "CoreUpgradeStatus",
		SpecType:   reflect.TypeOf(corev1beta1.UpgradeStatusSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.UpgradeStatusObservedState{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.UpgradeStatus",
			},
		},
	},
	{
		Name:       "CoreVcn",
		SpecType:   reflect.TypeOf(corev1beta1.VcnSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VcnStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateVcnDetails",
			},
			{
				SDKStruct: "core.UpdateVcnDetails",
			},
			{
				SDKStruct: "core.Vcn",
			},
		},
	},
	{
		Name:       "CoreVcnDnsResolverAssociation",
		SpecType:   reflect.TypeOf(corev1beta1.VcnDnsResolverAssociationSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VcnDnsResolverAssociationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.VcnDnsResolverAssociation",
			},
		},
	},
	{
		Name:       "CoreVcnTopology",
		SpecType:   reflect.TypeOf(corev1beta1.VcnTopologySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VcnTopologyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.VcnTopology",
			},
		},
	},
	{
		Name:       "CoreVirtualCircuit",
		SpecType:   reflect.TypeOf(corev1beta1.VirtualCircuitSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VirtualCircuitStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateVirtualCircuitDetails",
			},
			{
				SDKStruct: "core.UpdateVirtualCircuitDetails",
			},
			{
				SDKStruct:  "core.VirtualCircuit",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreVirtualCircuitAssociatedTunnel",
		SpecType:   reflect.TypeOf(corev1beta1.VirtualCircuitAssociatedTunnelSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VirtualCircuitAssociatedTunnelStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.VirtualCircuitAssociatedTunnelDetails",
			},
		},
	},
	{
		Name:       "CoreVirtualCircuitBandwidthShape",
		SpecType:   reflect.TypeOf(corev1beta1.VirtualCircuitBandwidthShapeSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VirtualCircuitBandwidthShapeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.VirtualCircuitBandwidthShape",
			},
		},
	},
	{
		Name:       "CoreVirtualCircuitPublicPrefix",
		SpecType:   reflect.TypeOf(corev1beta1.VirtualCircuitPublicPrefixSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VirtualCircuitPublicPrefixStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateVirtualCircuitPublicPrefixDetails",
			},
			{
				SDKStruct: "core.VirtualCircuitPublicPrefix",
			},
		},
	},
	{
		Name:       "CoreVlan",
		SpecType:   reflect.TypeOf(corev1beta1.VlanSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VlanStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateVlanDetails",
			},
			{
				SDKStruct: "core.UpdateVlanDetails",
			},
			{
				SDKStruct:  "core.Vlan",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreVnic",
		SpecType:   reflect.TypeOf(corev1beta1.VnicSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VnicStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateVnicDetails",
			},
			{
				SDKStruct: "core.UpdateVnicDetails",
			},
			{
				SDKStruct: "core.Vnic",
			},
		},
	},
	{
		Name:       "CoreVnicAttachment",
		SpecType:   reflect.TypeOf(corev1beta1.VnicAttachmentSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VnicAttachmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.VnicAttachment",
			},
		},
	},
	{
		Name:       "CoreVolume",
		SpecType:   reflect.TypeOf(corev1beta1.VolumeSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VolumeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateVolumeDetails",
			},
			{
				SDKStruct: "core.UpdateVolumeDetails",
			},
			{
				SDKStruct:  "core.Volume",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreVolumeAttachment",
		SpecType:   reflect.TypeOf(corev1beta1.VolumeAttachmentSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VolumeAttachmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.UpdateVolumeAttachmentDetails",
			},
		},
	},
	{
		Name:       "CoreVolumeBackup",
		SpecType:   reflect.TypeOf(corev1beta1.VolumeBackupSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VolumeBackupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateVolumeBackupDetails",
			},
			{
				SDKStruct: "core.UpdateVolumeBackupDetails",
			},
			{
				SDKStruct: "core.VolumeBackup",
			},
		},
	},
	{
		Name:       "CoreVolumeBackupPolicy",
		SpecType:   reflect.TypeOf(corev1beta1.VolumeBackupPolicySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VolumeBackupPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateVolumeBackupPolicyDetails",
			},
			{
				SDKStruct: "core.UpdateVolumeBackupPolicyDetails",
			},
			{
				SDKStruct:  "core.VolumeBackupPolicy",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreVolumeBackupPolicyAssetAssignment",
		SpecType:   reflect.TypeOf(corev1beta1.VolumeBackupPolicyAssetAssignmentSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VolumeBackupPolicyAssetAssignmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.VolumeBackupPolicyAssignment",
			},
		},
	},
	{
		Name:       "CoreVolumeBackupPolicyAssignment",
		SpecType:   reflect.TypeOf(corev1beta1.VolumeBackupPolicyAssignmentSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VolumeBackupPolicyAssignmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateVolumeBackupPolicyAssignmentDetails",
			},
			{
				SDKStruct: "core.VolumeBackupPolicyAssignment",
			},
		},
	},
	{
		Name:       "CoreVolumeGroup",
		SpecType:   reflect.TypeOf(corev1beta1.VolumeGroupSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VolumeGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateVolumeGroupDetails",
			},
			{
				SDKStruct: "core.UpdateVolumeGroupDetails",
			},
			{
				SDKStruct:  "core.VolumeGroup",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreVolumeGroupBackup",
		SpecType:   reflect.TypeOf(corev1beta1.VolumeGroupBackupSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VolumeGroupBackupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateVolumeGroupBackupDetails",
			},
			{
				SDKStruct: "core.UpdateVolumeGroupBackupDetails",
			},
			{
				SDKStruct: "core.VolumeGroupBackup",
			},
		},
	},
	{
		Name:       "CoreVolumeGroupReplica",
		SpecType:   reflect.TypeOf(corev1beta1.VolumeGroupReplicaSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VolumeGroupReplicaStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.VolumeGroupReplicaDetails",
			},
			{
				SDKStruct: "core.VolumeGroupReplica",
			},
		},
	},
	{
		Name:       "CoreVolumeKmsKey",
		SpecType:   reflect.TypeOf(corev1beta1.VolumeKmsKeySpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VolumeKmsKeyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.UpdateVolumeKmsKeyDetails",
			},
			{
				SDKStruct:  "core.VolumeKmsKey",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "CoreVtap",
		SpecType:   reflect.TypeOf(corev1beta1.VtapSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.VtapStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "core.CreateVtapDetails",
			},
			{
				SDKStruct: "core.UpdateVtapDetails",
			},
			{
				SDKStruct:  "core.Vtap",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CoreWindowsInstanceInitialCredential",
		SpecType:   reflect.TypeOf(corev1beta1.WindowsInstanceInitialCredentialSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.WindowsInstanceInitialCredentialStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.InstanceCredentials",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "DataflowApplication",
		SpecType:   reflect.TypeOf(dataflowv1beta1.ApplicationSpec{}),
		StatusType: reflect.TypeOf(dataflowv1beta1.ApplicationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataflow.CreateApplicationDetails",
			},
			{
				SDKStruct: "dataflow.UpdateApplicationDetails",
			},
			{
				SDKStruct: "dataflow.Application",
			},
			{
				SDKStruct: "dataflow.ApplicationSummary",
			},
		},
	},
	{
		Name:       "DataflowPool",
		SpecType:   reflect.TypeOf(dataflowv1beta1.PoolSpec{}),
		StatusType: reflect.TypeOf(dataflowv1beta1.PoolStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataflow.CreatePoolDetails",
			},
			{
				SDKStruct: "dataflow.UpdatePoolDetails",
			},
			{
				SDKStruct: "dataflow.Pool",
			},
			{
				SDKStruct: "dataflow.PoolCollection",
			},
			{
				SDKStruct: "dataflow.PoolSummary",
			},
		},
	},
	{
		Name:       "DataflowPrivateEndpoint",
		SpecType:   reflect.TypeOf(dataflowv1beta1.PrivateEndpointSpec{}),
		StatusType: reflect.TypeOf(dataflowv1beta1.PrivateEndpointStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataflow.CreatePrivateEndpointDetails",
			},
			{
				SDKStruct: "dataflow.UpdatePrivateEndpointDetails",
			},
			{
				SDKStruct: "dataflow.PrivateEndpoint",
			},
			{
				SDKStruct: "dataflow.PrivateEndpointCollection",
			},
			{
				SDKStruct: "dataflow.PrivateEndpointSummary",
			},
		},
	},
	{
		Name:       "DataflowRun",
		SpecType:   reflect.TypeOf(dataflowv1beta1.RunSpec{}),
		StatusType: reflect.TypeOf(dataflowv1beta1.RunStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataflow.CreateRunDetails",
			},
			{
				SDKStruct: "dataflow.UpdateRunDetails",
			},
			{
				SDKStruct: "dataflow.Run",
			},
			{
				SDKStruct: "dataflow.RunSummary",
			},
		},
	},
	{
		Name:       "DataflowRunLog",
		SpecType:   reflect.TypeOf(dataflowv1beta1.RunLogSpec{}),
		StatusType: reflect.TypeOf(dataflowv1beta1.RunLogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataflow.RunLogSummary",
			},
		},
	},
	{
		Name:       "DataflowSqlEndpoint",
		SpecType:   reflect.TypeOf(dataflowv1beta1.SqlEndpointSpec{}),
		StatusType: reflect.TypeOf(dataflowv1beta1.SqlEndpointStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataflow.CreateSqlEndpointDetails",
			},
			{
				SDKStruct: "dataflow.UpdateSqlEndpointDetails",
			},
			{
				SDKStruct: "dataflow.SqlEndpoint",
			},
			{
				SDKStruct: "dataflow.SqlEndpointCollection",
			},
			{
				SDKStruct: "dataflow.SqlEndpointSummary",
			},
		},
	},
	{
		Name:       "DataflowStatement",
		SpecType:   reflect.TypeOf(dataflowv1beta1.StatementSpec{}),
		StatusType: reflect.TypeOf(dataflowv1beta1.StatementStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataflow.CreateStatementDetails",
			},
			{
				SDKStruct: "dataflow.Statement",
			},
			{
				SDKStruct: "dataflow.StatementCollection",
			},
			{
				SDKStruct: "dataflow.StatementSummary",
			},
		},
	},
	{
		Name:       "DataflowWorkRequest",
		SpecType:   reflect.TypeOf(dataflowv1beta1.WorkRequestSpec{}),
		StatusType: reflect.TypeOf(dataflowv1beta1.WorkRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataflow.WorkRequest",
			},
			{
				SDKStruct: "dataflow.WorkRequestCollection",
			},
			{
				SDKStruct: "dataflow.WorkRequestSummary",
			},
		},
	},
	{
		Name:       "DataflowWorkRequestError",
		SpecType:   reflect.TypeOf(dataflowv1beta1.WorkRequestErrorSpec{}),
		StatusType: reflect.TypeOf(dataflowv1beta1.WorkRequestErrorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataflow.WorkRequestError",
			},
			{
				SDKStruct: "dataflow.WorkRequestErrorCollection",
			},
		},
	},
	{
		Name:       "DataflowWorkRequestLog",
		SpecType:   reflect.TypeOf(dataflowv1beta1.WorkRequestLogSpec{}),
		StatusType: reflect.TypeOf(dataflowv1beta1.WorkRequestLogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataflow.WorkRequestLog",
			},
			{
				SDKStruct: "dataflow.WorkRequestLogCollection",
			},
		},
	},
	{
		Name:       "OpensearchOpensearchCluster",
		SpecType:   reflect.TypeOf(opensearchv1beta1.OpensearchClusterSpec{}),
		StatusType: reflect.TypeOf(opensearchv1beta1.OpensearchClusterStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "opensearch.CreateOpensearchClusterDetails",
			},
			{
				SDKStruct: "opensearch.UpdateOpensearchClusterDetails",
			},
			{
				SDKStruct: "opensearch.OpensearchCluster",
			},
			{
				SDKStruct: "opensearch.OpensearchClusterCollection",
			},
			{
				SDKStruct: "opensearch.OpensearchClusterSummary",
			},
		},
	},
	{
		Name:       "RedisRedisCluster",
		SpecType:   reflect.TypeOf(redisv1beta1.RedisClusterSpec{}),
		StatusType: reflect.TypeOf(redisv1beta1.RedisClusterStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "redis.CreateRedisClusterDetails",
			},
			{
				SDKStruct: "redis.UpdateRedisClusterDetails",
			},
			{
				SDKStruct: "redis.RedisCluster",
			},
			{
				SDKStruct: "redis.RedisClusterCollection",
			},
			{
				SDKStruct: "redis.RedisClusterSummary",
			},
		},
	},
}

func Targets() []Target {
	result := make([]Target, len(targets))
	for i := range targets {
		result[i] = targets[i]
		if len(targets[i].SDKMappings) > 0 {
			result[i].SDKMappings = append([]SDKMapping(nil), targets[i].SDKMappings...)
		}
	}
	return result
}
