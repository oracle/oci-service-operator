package sdk

import (
	"path"
	"reflect"

	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/oracle/oci-go-sdk/v65/database"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	"github.com/oracle/oci-go-sdk/v65/nosql"
	"github.com/oracle/oci-go-sdk/v65/psql"
	"github.com/oracle/oci-go-sdk/v65/streaming"
)

const (
	modulePath    = "github.com/oracle/oci-go-sdk/v65"
	moduleVersion = "v65.61.1"
)

var seedTargets = []Target{
	// Autonomous Database CRD support
	newTarget("database", "CreateAutonomousDatabaseDetails", reflect.TypeOf(database.CreateAutonomousDatabaseDetails{})),
	newTarget("database", "UpdateAutonomousDatabaseDetails", reflect.TypeOf(database.UpdateAutonomousDatabaseDetails{})),
	newTarget("database", "AutonomousDatabase", reflect.TypeOf(database.AutonomousDatabase{})),
	newTarget("database", "AutonomousDatabaseSummary", reflect.TypeOf(database.AutonomousDatabaseSummary{})),

	// MySQL DB System CRD support
	newTarget("mysql", "CreateBackupDetails", reflect.TypeOf(mysql.CreateBackupDetails{})),
	newTarget("mysql", "CreateChannelDetails", reflect.TypeOf(mysql.CreateChannelDetails{})),
	newTarget("mysql", "CreateConfigurationDetails", reflect.TypeOf(mysql.CreateConfigurationDetails{})),
	newTarget("mysql", "CreateDbSystemDetails", reflect.TypeOf(mysql.CreateDbSystemDetails{})),
	newTarget("mysql", "CreateReplicaDetails", reflect.TypeOf(mysql.CreateReplicaDetails{})),
	newTarget("mysql", "UpdateBackupDetails", reflect.TypeOf(mysql.UpdateBackupDetails{})),
	newTarget("mysql", "UpdateChannelDetails", reflect.TypeOf(mysql.UpdateChannelDetails{})),
	newTarget("mysql", "UpdateConfigurationDetails", reflect.TypeOf(mysql.UpdateConfigurationDetails{})),
	newTarget("mysql", "UpdateDbSystemDetails", reflect.TypeOf(mysql.UpdateDbSystemDetails{})),
	newTarget("mysql", "UpdateHeatWaveClusterDetails", reflect.TypeOf(mysql.UpdateHeatWaveClusterDetails{})),
	newTarget("mysql", "UpdateReplicaDetails", reflect.TypeOf(mysql.UpdateReplicaDetails{})),
	newTarget("mysql", "Backup", reflect.TypeOf(mysql.Backup{})),
	newTarget("mysql", "Channel", reflect.TypeOf(mysql.Channel{})),
	newTarget("mysql", "Configuration", reflect.TypeOf(mysql.Configuration{})),
	newTarget("mysql", "DbSystem", reflect.TypeOf(mysql.DbSystem{})),
	newTarget("mysql", "HeatWaveCluster", reflect.TypeOf(mysql.HeatWaveCluster{})),
	newTarget("mysql", "HeatWaveClusterMemoryEstimate", reflect.TypeOf(mysql.HeatWaveClusterMemoryEstimate{})),
	newTarget("mysql", "Replica", reflect.TypeOf(mysql.Replica{})),
	newTarget("mysql", "Version", reflect.TypeOf(mysql.Version{})),
	newTarget("mysql", "WorkRequest", reflect.TypeOf(mysql.WorkRequest{})),
	newTarget("mysql", "WorkRequestError", reflect.TypeOf(mysql.WorkRequestError{})),
	newTarget("mysql", "WorkRequestLogEntry", reflect.TypeOf(mysql.WorkRequestLogEntry{})),
	newTarget("mysql", "VersionSummary", reflect.TypeOf(mysql.VersionSummary{})),
	newTarget("mysql", "BackupSummary", reflect.TypeOf(mysql.BackupSummary{})),
	newTarget("mysql", "ChannelSummary", reflect.TypeOf(mysql.ChannelSummary{})),
	newTarget("mysql", "ConfigurationSummary", reflect.TypeOf(mysql.ConfigurationSummary{})),
	newTarget("mysql", "DbSystemSummary", reflect.TypeOf(mysql.DbSystemSummary{})),
	newTarget("mysql", "HeatWaveClusterSummary", reflect.TypeOf(mysql.HeatWaveClusterSummary{})),
	newTarget("mysql", "ReplicaSummary", reflect.TypeOf(mysql.ReplicaSummary{})),
	newTarget("mysql", "ShapeSummary", reflect.TypeOf(mysql.ShapeSummary{})),
	newTarget("mysql", "WorkRequestSummary", reflect.TypeOf(mysql.WorkRequestSummary{})),

	// Streaming CRD support
	newTarget("streaming", "CreateStreamDetails", reflect.TypeOf(streaming.CreateStreamDetails{})),
	newTarget("streaming", "UpdateStreamDetails", reflect.TypeOf(streaming.UpdateStreamDetails{})),
	newTarget("streaming", "Stream", reflect.TypeOf(streaming.Stream{})),
	newTarget("streaming", "StreamSummary", reflect.TypeOf(streaming.StreamSummary{})),

	// NoSQL CRD support
	newTarget("nosql", "CreateIndexDetails", reflect.TypeOf(nosql.CreateIndexDetails{})),
	newTarget("nosql", "CreateReplicaDetails", reflect.TypeOf(nosql.CreateReplicaDetails{})),
	newTarget("nosql", "CreateTableDetails", reflect.TypeOf(nosql.CreateTableDetails{})),
	newTarget("nosql", "UpdateRowDetails", reflect.TypeOf(nosql.UpdateRowDetails{})),
	newTarget("nosql", "UpdateTableDetails", reflect.TypeOf(nosql.UpdateTableDetails{})),
	newTarget("nosql", "Index", reflect.TypeOf(nosql.Index{})),
	newTarget("nosql", "IndexCollection", reflect.TypeOf(nosql.IndexCollection{})),
	newTarget("nosql", "Replica", reflect.TypeOf(nosql.Replica{})),
	newTarget("nosql", "Row", reflect.TypeOf(nosql.Row{})),
	newTarget("nosql", "Table", reflect.TypeOf(nosql.Table{})),
	newTarget("nosql", "TableCollection", reflect.TypeOf(nosql.TableCollection{})),
	newTarget("nosql", "TableUsageCollection", reflect.TypeOf(nosql.TableUsageCollection{})),
	newTarget("nosql", "WorkRequest", reflect.TypeOf(nosql.WorkRequest{})),
	newTarget("nosql", "WorkRequestCollection", reflect.TypeOf(nosql.WorkRequestCollection{})),
	newTarget("nosql", "WorkRequestError", reflect.TypeOf(nosql.WorkRequestError{})),
	newTarget("nosql", "WorkRequestErrorCollection", reflect.TypeOf(nosql.WorkRequestErrorCollection{})),
	newTarget("nosql", "WorkRequestLogEntry", reflect.TypeOf(nosql.WorkRequestLogEntry{})),
	newTarget("nosql", "WorkRequestLogEntryCollection", reflect.TypeOf(nosql.WorkRequestLogEntryCollection{})),
	newTarget("nosql", "IndexSummary", reflect.TypeOf(nosql.IndexSummary{})),
	newTarget("nosql", "TableSummary", reflect.TypeOf(nosql.TableSummary{})),
	newTarget("nosql", "TableUsageSummary", reflect.TypeOf(nosql.TableUsageSummary{})),
	newTarget("nosql", "WorkRequestSummary", reflect.TypeOf(nosql.WorkRequestSummary{})),

	// PostgreSQL CRD support
	newTarget("psql", "CreateBackupDetails", reflect.TypeOf(psql.CreateBackupDetails{})),
	newTarget("psql", "CreateConfigurationDetails", reflect.TypeOf(psql.CreateConfigurationDetails{})),
	newTarget("psql", "CreateDbSystemDetails", reflect.TypeOf(psql.CreateDbSystemDetails{})),
	newTarget("psql", "UpdateBackupDetails", reflect.TypeOf(psql.UpdateBackupDetails{})),
	newTarget("psql", "UpdateConfigurationDetails", reflect.TypeOf(psql.UpdateConfigurationDetails{})),
	newTarget("psql", "UpdateDbSystemDbInstanceDetails", reflect.TypeOf(psql.UpdateDbSystemDbInstanceDetails{})),
	newTarget("psql", "UpdateDbSystemDetails", reflect.TypeOf(psql.UpdateDbSystemDetails{})),
	newTarget("psql", "ConfigurationDetails", reflect.TypeOf(psql.ConfigurationDetails{})),
	newTarget("psql", "ConnectionDetails", reflect.TypeOf(psql.ConnectionDetails{})),
	newTarget("psql", "DbSystemDetails", reflect.TypeOf(psql.DbSystemDetails{})),
	newTarget("psql", "DefaultConfigurationDetails", reflect.TypeOf(psql.DefaultConfigurationDetails{})),
	newTarget("psql", "PrimaryDbInstanceDetails", reflect.TypeOf(psql.PrimaryDbInstanceDetails{})),
	newTarget("psql", "Backup", reflect.TypeOf(psql.Backup{})),
	newTarget("psql", "BackupCollection", reflect.TypeOf(psql.BackupCollection{})),
	newTarget("psql", "Configuration", reflect.TypeOf(psql.Configuration{})),
	newTarget("psql", "ConfigurationCollection", reflect.TypeOf(psql.ConfigurationCollection{})),
	newTarget("psql", "DbSystem", reflect.TypeOf(psql.DbSystem{})),
	newTarget("psql", "DbSystemCollection", reflect.TypeOf(psql.DbSystemCollection{})),
	newTarget("psql", "DefaultConfiguration", reflect.TypeOf(psql.DefaultConfiguration{})),
	newTarget("psql", "DefaultConfigurationCollection", reflect.TypeOf(psql.DefaultConfigurationCollection{})),
	newTarget("psql", "ShapeCollection", reflect.TypeOf(psql.ShapeCollection{})),
	newTarget("psql", "WorkRequest", reflect.TypeOf(psql.WorkRequest{})),
	newTarget("psql", "WorkRequestError", reflect.TypeOf(psql.WorkRequestError{})),
	newTarget("psql", "WorkRequestErrorCollection", reflect.TypeOf(psql.WorkRequestErrorCollection{})),
	newTarget("psql", "WorkRequestLogEntry", reflect.TypeOf(psql.WorkRequestLogEntry{})),
	newTarget("psql", "WorkRequestLogEntryCollection", reflect.TypeOf(psql.WorkRequestLogEntryCollection{})),
	newTarget("psql", "BackupSummary", reflect.TypeOf(psql.BackupSummary{})),
	newTarget("psql", "ConfigurationSummary", reflect.TypeOf(psql.ConfigurationSummary{})),
	newTarget("psql", "DbSystemSummary", reflect.TypeOf(psql.DbSystemSummary{})),
	newTarget("psql", "DefaultConfigurationSummary", reflect.TypeOf(psql.DefaultConfigurationSummary{})),
	newTarget("psql", "ShapeSummary", reflect.TypeOf(psql.ShapeSummary{})),
	newTarget("psql", "WorkRequestSummary", reflect.TypeOf(psql.WorkRequestSummary{})),

	// Container Engine CRD support
	newTarget("containerengine", "CreateClusterDetails", reflect.TypeOf(containerengine.CreateClusterDetails{})),
	newTarget("containerengine", "CreateClusterEndpointConfigDetails", reflect.TypeOf(containerengine.CreateClusterEndpointConfigDetails{})),
	newTarget("containerengine", "CreateClusterKubeconfigContentDetails", reflect.TypeOf(containerengine.CreateClusterKubeconfigContentDetails{})),
	newTarget("containerengine", "CreateNodePoolDetails", reflect.TypeOf(containerengine.CreateNodePoolDetails{})),
	newTarget("containerengine", "CreateVirtualNodePoolDetails", reflect.TypeOf(containerengine.CreateVirtualNodePoolDetails{})),
	newTarget("containerengine", "CreateWorkloadMappingDetails", reflect.TypeOf(containerengine.CreateWorkloadMappingDetails{})),
	newTarget("containerengine", "UpdateAddonDetails", reflect.TypeOf(containerengine.UpdateAddonDetails{})),
	newTarget("containerengine", "UpdateClusterDetails", reflect.TypeOf(containerengine.UpdateClusterDetails{})),
	newTarget("containerengine", "UpdateClusterEndpointConfigDetails", reflect.TypeOf(containerengine.UpdateClusterEndpointConfigDetails{})),
	newTarget("containerengine", "UpdateNodePoolDetails", reflect.TypeOf(containerengine.UpdateNodePoolDetails{})),
	newTarget("containerengine", "UpdateVirtualNodePoolDetails", reflect.TypeOf(containerengine.UpdateVirtualNodePoolDetails{})),
	newTarget("containerengine", "UpdateWorkloadMappingDetails", reflect.TypeOf(containerengine.UpdateWorkloadMappingDetails{})),
	newTarget("containerengine", "Addon", reflect.TypeOf(containerengine.Addon{})),
	newTarget("containerengine", "Cluster", reflect.TypeOf(containerengine.Cluster{})),
	newTarget("containerengine", "ClusterEndpointConfig", reflect.TypeOf(containerengine.ClusterEndpointConfig{})),
	newTarget("containerengine", "ClusterMigrateToNativeVcnStatus", reflect.TypeOf(containerengine.ClusterMigrateToNativeVcnStatus{})),
	newTarget("containerengine", "ClusterOptions", reflect.TypeOf(containerengine.ClusterOptions{})),
	newTarget("containerengine", "CredentialRotationStatus", reflect.TypeOf(containerengine.CredentialRotationStatus{})),
	newTarget("containerengine", "Node", reflect.TypeOf(containerengine.Node{})),
	newTarget("containerengine", "NodePool", reflect.TypeOf(containerengine.NodePool{})),
	newTarget("containerengine", "NodePoolOptions", reflect.TypeOf(containerengine.NodePoolOptions{})),
	newTarget("containerengine", "PodShape", reflect.TypeOf(containerengine.PodShape{})),
	newTarget("containerengine", "VirtualNode", reflect.TypeOf(containerengine.VirtualNode{})),
	newTarget("containerengine", "VirtualNodePool", reflect.TypeOf(containerengine.VirtualNodePool{})),
	newTarget("containerengine", "WorkRequest", reflect.TypeOf(containerengine.WorkRequest{})),
	newTarget("containerengine", "WorkRequestError", reflect.TypeOf(containerengine.WorkRequestError{})),
	newTarget("containerengine", "WorkRequestLogEntry", reflect.TypeOf(containerengine.WorkRequestLogEntry{})),
	newTarget("containerengine", "WorkloadMapping", reflect.TypeOf(containerengine.WorkloadMapping{})),
	newTarget("containerengine", "AddonOptionSummary", reflect.TypeOf(containerengine.AddonOptionSummary{})),
	newTarget("containerengine", "AddonSummary", reflect.TypeOf(containerengine.AddonSummary{})),
	newTarget("containerengine", "ClusterSummary", reflect.TypeOf(containerengine.ClusterSummary{})),
	newTarget("containerengine", "NodePoolSummary", reflect.TypeOf(containerengine.NodePoolSummary{})),
	newTarget("containerengine", "PodShapeSummary", reflect.TypeOf(containerengine.PodShapeSummary{})),
	newTarget("containerengine", "VirtualNodePoolSummary", reflect.TypeOf(containerengine.VirtualNodePoolSummary{})),
	newTarget("containerengine", "VirtualNodeSummary", reflect.TypeOf(containerengine.VirtualNodeSummary{})),
	newTarget("containerengine", "WorkRequestSummary", reflect.TypeOf(containerengine.WorkRequestSummary{})),
	newTarget("containerengine", "WorkloadMappingSummary", reflect.TypeOf(containerengine.WorkloadMappingSummary{})),
}

var interfaceImplementations = map[string][]reflect.Type{
	qualifiedTypeName(reflect.TypeOf((*mysql.CreateDbSystemSourceDetails)(nil)).Elem()): {
		reflect.TypeOf(mysql.CreateDbSystemSourceFromBackupDetails{}),
		reflect.TypeOf(mysql.CreateDbSystemSourceFromNoneDetails{}),
		reflect.TypeOf(mysql.CreateDbSystemSourceFromPitrDetails{}),
		reflect.TypeOf(mysql.CreateDbSystemSourceImportFromUrlDetails{}),
	},
}

func SeedTargets() []Target {
	result := make([]Target, len(seedTargets))
	copy(result, seedTargets)
	return result
}

func TargetByName(qualifiedName string) (Target, bool) {
	for _, target := range seedTargets {
		if target.QualifiedName == qualifiedName {
			return target, true
		}
	}
	return Target{}, false
}

func knownInterfaceImplementations(interfaceType reflect.Type) []reflect.Type {
	known := interfaceImplementations[qualifiedTypeName(interfaceType)]
	result := make([]reflect.Type, len(known))
	copy(result, known)
	return result
}

func newTarget(packageName string, typeName string, typeRef reflect.Type) Target {
	return Target{
		QualifiedName: packageName + "." + typeName,
		PackageName:   packageName,
		TypeName:      typeName,
		ImportPath:    typeRef.PkgPath(),
		ReflectType:   typeRef,
	}
}

func qualifiedTypeName(typeRef reflect.Type) string {
	return path.Base(typeRef.PkgPath()) + "." + typeRef.Name()
}
