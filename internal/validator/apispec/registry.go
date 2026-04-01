package apispec

import (
	"reflect"

	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	nosqlv1beta1 "github.com/oracle/oci-service-operator/api/nosql/v1beta1"
	psqlv1beta1 "github.com/oracle/oci-service-operator/api/psql/v1beta1"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
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
		Name:       "DbSystem",
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
		Name:       "PostgreSQLDbSystem",
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
