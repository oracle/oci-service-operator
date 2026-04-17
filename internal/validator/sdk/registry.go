package sdk

import (
	"path"
	"reflect"

	"github.com/oracle/oci-go-sdk/v65/aidocument"
	"github.com/oracle/oci-go-sdk/v65/ailanguage"
	"github.com/oracle/oci-go-sdk/v65/aivision"
	"github.com/oracle/oci-go-sdk/v65/analytics"
	"github.com/oracle/oci-go-sdk/v65/bds"
	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/oracle/oci-go-sdk/v65/containerinstances"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/database"
	"github.com/oracle/oci-go-sdk/v65/databasetools"
	"github.com/oracle/oci-go-sdk/v65/dataflow"
	"github.com/oracle/oci-go-sdk/v65/datascience"
	"github.com/oracle/oci-go-sdk/v65/functions"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/oracle/oci-go-sdk/v65/keymanagement"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	"github.com/oracle/oci-go-sdk/v65/nosql"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
	"github.com/oracle/oci-go-sdk/v65/opensearch"
	"github.com/oracle/oci-go-sdk/v65/psql"
	"github.com/oracle/oci-go-sdk/v65/queue"
	"github.com/oracle/oci-go-sdk/v65/redis"
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
	newTarget("mysql", "CreateDbSystemDetails", reflect.TypeOf(mysql.CreateDbSystemDetails{})),
	newTarget("mysql", "UpdateDbSystemDetails", reflect.TypeOf(mysql.UpdateDbSystemDetails{})),
	newTarget("mysql", "DbSystem", reflect.TypeOf(mysql.DbSystem{})),
	newTarget("mysql", "DbSystemSummary", reflect.TypeOf(mysql.DbSystemSummary{})),

	// Streaming CRD support
	newTarget("streaming", "CreateStreamDetails", reflect.TypeOf(streaming.CreateStreamDetails{})),
	newTarget("streaming", "UpdateStreamDetails", reflect.TypeOf(streaming.UpdateStreamDetails{})),
	newTarget("streaming", "Stream", reflect.TypeOf(streaming.Stream{})),
	newTarget("streaming", "StreamSummary", reflect.TypeOf(streaming.StreamSummary{})),

	// Queue CRD support
	newTarget("queue", "CreateQueueDetails", reflect.TypeOf(queue.CreateQueueDetails{})),
	newTarget("queue", "UpdateQueueDetails", reflect.TypeOf(queue.UpdateQueueDetails{})),
	newTarget("queue", "Queue", reflect.TypeOf(queue.Queue{})),
	newTarget("queue", "QueueCollection", reflect.TypeOf(queue.QueueCollection{})),
	newTarget("queue", "QueueSummary", reflect.TypeOf(queue.QueueSummary{})),

	// Functions CRD support
	newTarget("functions", "CreateApplicationDetails", reflect.TypeOf(functions.CreateApplicationDetails{})),
	newTarget("functions", "CreateFunctionDetails", reflect.TypeOf(functions.CreateFunctionDetails{})),
	newTarget("functions", "UpdateApplicationDetails", reflect.TypeOf(functions.UpdateApplicationDetails{})),
	newTarget("functions", "UpdateFunctionDetails", reflect.TypeOf(functions.UpdateFunctionDetails{})),
	newTarget("functions", "Application", reflect.TypeOf(functions.Application{})),
	newTarget("functions", "Function", reflect.TypeOf(functions.Function{})),
	newTarget("functions", "ApplicationSummary", reflect.TypeOf(functions.ApplicationSummary{})),
	newTarget("functions", "FunctionSummary", reflect.TypeOf(functions.FunctionSummary{})),

	// NoSQL CRD support
	newTarget("nosql", "CreateTableDetails", reflect.TypeOf(nosql.CreateTableDetails{})),
	newTarget("nosql", "UpdateTableDetails", reflect.TypeOf(nosql.UpdateTableDetails{})),
	newTarget("nosql", "Table", reflect.TypeOf(nosql.Table{})),
	newTarget("nosql", "TableCollection", reflect.TypeOf(nosql.TableCollection{})),
	newTarget("nosql", "TableSummary", reflect.TypeOf(nosql.TableSummary{})),

	// Object Storage CRD support
	newTarget("objectstorage", "CreateBucketDetails", reflect.TypeOf(objectstorage.CreateBucketDetails{})),
	newTarget("objectstorage", "UpdateBucketDetails", reflect.TypeOf(objectstorage.UpdateBucketDetails{})),
	newTarget("objectstorage", "Bucket", reflect.TypeOf(objectstorage.Bucket{})),
	newTarget("objectstorage", "BucketSummary", reflect.TypeOf(objectstorage.BucketSummary{})),

	// PostgreSQL CRD support
	newTarget("psql", "CreateDbSystemDetails", reflect.TypeOf(psql.CreateDbSystemDetails{})),
	newTarget("psql", "UpdateDbSystemDetails", reflect.TypeOf(psql.UpdateDbSystemDetails{})),
	newTarget("psql", "DbSystemDetails", reflect.TypeOf(psql.DbSystemDetails{})),
	newTarget("psql", "DbSystem", reflect.TypeOf(psql.DbSystem{})),
	newTarget("psql", "DbSystemCollection", reflect.TypeOf(psql.DbSystemCollection{})),
	newTarget("psql", "DbSystemSummary", reflect.TypeOf(psql.DbSystemSummary{})),

	// Container Engine CRD support
	newTarget("containerengine", "CreateClusterDetails", reflect.TypeOf(containerengine.CreateClusterDetails{})),
	newTarget("containerengine", "CreateNodePoolDetails", reflect.TypeOf(containerengine.CreateNodePoolDetails{})),
	newTarget("containerengine", "UpdateClusterDetails", reflect.TypeOf(containerengine.UpdateClusterDetails{})),
	newTarget("containerengine", "UpdateNodePoolDetails", reflect.TypeOf(containerengine.UpdateNodePoolDetails{})),
	newTarget("containerengine", "Cluster", reflect.TypeOf(containerengine.Cluster{})),
	newTarget("containerengine", "NodePool", reflect.TypeOf(containerengine.NodePool{})),
	newTarget("containerengine", "ClusterSummary", reflect.TypeOf(containerengine.ClusterSummary{})),
	newTarget("containerengine", "NodePoolSummary", reflect.TypeOf(containerengine.NodePoolSummary{})),

	// Identity CRD support
	newTarget("identity", "CreateCompartmentDetails", reflect.TypeOf(identity.CreateCompartmentDetails{})),
	newTarget("identity", "UpdateCompartmentDetails", reflect.TypeOf(identity.UpdateCompartmentDetails{})),
	newTarget("identity", "Compartment", reflect.TypeOf(identity.Compartment{})),

	// Key Management CRD support
	newTarget("keymanagement", "CreateVaultDetails", reflect.TypeOf(keymanagement.CreateVaultDetails{})),
	newTarget("keymanagement", "UpdateVaultDetails", reflect.TypeOf(keymanagement.UpdateVaultDetails{})),
	newTarget("keymanagement", "Vault", reflect.TypeOf(keymanagement.Vault{})),
	newTarget("keymanagement", "VaultSummary", reflect.TypeOf(keymanagement.VaultSummary{})),

	// Core VCN CRD support
	newTarget("core", "CreateDrgDetails", reflect.TypeOf(core.CreateDrgDetails{})),
	newTarget("core", "CreateInternetGatewayDetails", reflect.TypeOf(core.CreateInternetGatewayDetails{})),
	newTarget("core", "CreateNatGatewayDetails", reflect.TypeOf(core.CreateNatGatewayDetails{})),
	newTarget("core", "CreateNetworkSecurityGroupDetails", reflect.TypeOf(core.CreateNetworkSecurityGroupDetails{})),
	newTarget("core", "CreateRouteTableDetails", reflect.TypeOf(core.CreateRouteTableDetails{})),
	newTarget("core", "CreateSecurityListDetails", reflect.TypeOf(core.CreateSecurityListDetails{})),
	newTarget("core", "CreateServiceGatewayDetails", reflect.TypeOf(core.CreateServiceGatewayDetails{})),
	newTarget("core", "CreateSubnetDetails", reflect.TypeOf(core.CreateSubnetDetails{})),
	newTarget("core", "CreateVcnDetails", reflect.TypeOf(core.CreateVcnDetails{})),
	newTarget("core", "UpdateDrgDetails", reflect.TypeOf(core.UpdateDrgDetails{})),
	newTarget("core", "UpdateInstanceDetails", reflect.TypeOf(core.UpdateInstanceDetails{})),
	newTarget("core", "UpdateInternetGatewayDetails", reflect.TypeOf(core.UpdateInternetGatewayDetails{})),
	newTarget("core", "UpdateNatGatewayDetails", reflect.TypeOf(core.UpdateNatGatewayDetails{})),
	newTarget("core", "UpdateNetworkSecurityGroupDetails", reflect.TypeOf(core.UpdateNetworkSecurityGroupDetails{})),
	newTarget("core", "UpdateRouteTableDetails", reflect.TypeOf(core.UpdateRouteTableDetails{})),
	newTarget("core", "UpdateSecurityListDetails", reflect.TypeOf(core.UpdateSecurityListDetails{})),
	newTarget("core", "UpdateServiceGatewayDetails", reflect.TypeOf(core.UpdateServiceGatewayDetails{})),
	newTarget("core", "UpdateSubnetDetails", reflect.TypeOf(core.UpdateSubnetDetails{})),
	newTarget("core", "UpdateVcnDetails", reflect.TypeOf(core.UpdateVcnDetails{})),
	newTarget("core", "Drg", reflect.TypeOf(core.Drg{})),
	newTarget("core", "Instance", reflect.TypeOf(core.Instance{})),
	newTarget("core", "InternetGateway", reflect.TypeOf(core.InternetGateway{})),
	newTarget("core", "NatGateway", reflect.TypeOf(core.NatGateway{})),
	newTarget("core", "NetworkSecurityGroup", reflect.TypeOf(core.NetworkSecurityGroup{})),
	newTarget("core", "RouteTable", reflect.TypeOf(core.RouteTable{})),
	newTarget("core", "SecurityList", reflect.TypeOf(core.SecurityList{})),
	newTarget("core", "ServiceGateway", reflect.TypeOf(core.ServiceGateway{})),
	newTarget("core", "Subnet", reflect.TypeOf(core.Subnet{})),
	newTarget("core", "Vcn", reflect.TypeOf(core.Vcn{})),
	newTarget("core", "InstanceSummary", reflect.TypeOf(core.InstanceSummary{})),

	// Aidocument CRD support
	newTarget("aidocument", "CreateProjectDetails", reflect.TypeOf(aidocument.CreateProjectDetails{})),
	newTarget("aidocument", "UpdateProjectDetails", reflect.TypeOf(aidocument.UpdateProjectDetails{})),
	newTarget("aidocument", "Project", reflect.TypeOf(aidocument.Project{})),
	newTarget("aidocument", "ProjectCollection", reflect.TypeOf(aidocument.ProjectCollection{})),
	newTarget("aidocument", "ProjectSummary", reflect.TypeOf(aidocument.ProjectSummary{})),

	// Ailanguage CRD support
	newTarget("ailanguage", "CreateProjectDetails", reflect.TypeOf(ailanguage.CreateProjectDetails{})),
	newTarget("ailanguage", "UpdateProjectDetails", reflect.TypeOf(ailanguage.UpdateProjectDetails{})),
	newTarget("ailanguage", "Project", reflect.TypeOf(ailanguage.Project{})),
	newTarget("ailanguage", "ProjectCollection", reflect.TypeOf(ailanguage.ProjectCollection{})),
	newTarget("ailanguage", "ProjectSummary", reflect.TypeOf(ailanguage.ProjectSummary{})),

	// Aivision CRD support
	newTarget("aivision", "CreateProjectDetails", reflect.TypeOf(aivision.CreateProjectDetails{})),
	newTarget("aivision", "UpdateProjectDetails", reflect.TypeOf(aivision.UpdateProjectDetails{})),
	newTarget("aivision", "Project", reflect.TypeOf(aivision.Project{})),
	newTarget("aivision", "ProjectCollection", reflect.TypeOf(aivision.ProjectCollection{})),
	newTarget("aivision", "ProjectSummary", reflect.TypeOf(aivision.ProjectSummary{})),

	// Analytics CRD support
	newTarget("analytics", "CreateAnalyticsInstanceDetails", reflect.TypeOf(analytics.CreateAnalyticsInstanceDetails{})),
	newTarget("analytics", "UpdateAnalyticsInstanceDetails", reflect.TypeOf(analytics.UpdateAnalyticsInstanceDetails{})),
	newTarget("analytics", "AnalyticsInstance", reflect.TypeOf(analytics.AnalyticsInstance{})),
	newTarget("analytics", "AnalyticsInstanceSummary", reflect.TypeOf(analytics.AnalyticsInstanceSummary{})),

	// Bds CRD support
	newTarget("bds", "CreateBdsInstanceDetails", reflect.TypeOf(bds.CreateBdsInstanceDetails{})),
	newTarget("bds", "UpdateBdsInstanceDetails", reflect.TypeOf(bds.UpdateBdsInstanceDetails{})),
	newTarget("bds", "BdsInstance", reflect.TypeOf(bds.BdsInstance{})),
	newTarget("bds", "BdsInstanceSummary", reflect.TypeOf(bds.BdsInstanceSummary{})),

	// Containerinstances CRD support
	newTarget("containerinstances", "CreateContainerInstanceDetails", reflect.TypeOf(containerinstances.CreateContainerInstanceDetails{})),
	newTarget("containerinstances", "UpdateContainerInstanceDetails", reflect.TypeOf(containerinstances.UpdateContainerInstanceDetails{})),
	newTarget("containerinstances", "ContainerInstance", reflect.TypeOf(containerinstances.ContainerInstance{})),
	newTarget("containerinstances", "ContainerInstanceCollection", reflect.TypeOf(containerinstances.ContainerInstanceCollection{})),
	newTarget("containerinstances", "ContainerInstanceSummary", reflect.TypeOf(containerinstances.ContainerInstanceSummary{})),

	// Databasetools CRD support
	newTarget("databasetools", "CreateDatabaseToolsConnectionGenericJdbcDetails", reflect.TypeOf(databasetools.CreateDatabaseToolsConnectionGenericJdbcDetails{})),
	newTarget("databasetools", "CreateDatabaseToolsConnectionMySqlDetails", reflect.TypeOf(databasetools.CreateDatabaseToolsConnectionMySqlDetails{})),
	newTarget("databasetools", "CreateDatabaseToolsConnectionOracleDatabaseDetails", reflect.TypeOf(databasetools.CreateDatabaseToolsConnectionOracleDatabaseDetails{})),
	newTarget("databasetools", "CreateDatabaseToolsConnectionPostgresqlDetails", reflect.TypeOf(databasetools.CreateDatabaseToolsConnectionPostgresqlDetails{})),
	newTarget("databasetools", "UpdateDatabaseToolsConnectionGenericJdbcDetails", reflect.TypeOf(databasetools.UpdateDatabaseToolsConnectionGenericJdbcDetails{})),
	newTarget("databasetools", "UpdateDatabaseToolsConnectionMySqlDetails", reflect.TypeOf(databasetools.UpdateDatabaseToolsConnectionMySqlDetails{})),
	newTarget("databasetools", "UpdateDatabaseToolsConnectionOracleDatabaseDetails", reflect.TypeOf(databasetools.UpdateDatabaseToolsConnectionOracleDatabaseDetails{})),
	newTarget("databasetools", "UpdateDatabaseToolsConnectionPostgresqlDetails", reflect.TypeOf(databasetools.UpdateDatabaseToolsConnectionPostgresqlDetails{})),
	newTarget("databasetools", "DatabaseToolsConnectionCollection", reflect.TypeOf(databasetools.DatabaseToolsConnectionCollection{})),
	newTarget("databasetools", "DatabaseToolsConnectionGenericJdbc", reflect.TypeOf(databasetools.DatabaseToolsConnectionGenericJdbc{})),
	newTarget("databasetools", "DatabaseToolsConnectionMySql", reflect.TypeOf(databasetools.DatabaseToolsConnectionMySql{})),
	newTarget("databasetools", "DatabaseToolsConnectionOracleDatabase", reflect.TypeOf(databasetools.DatabaseToolsConnectionOracleDatabase{})),
	newTarget("databasetools", "DatabaseToolsConnectionPostgresql", reflect.TypeOf(databasetools.DatabaseToolsConnectionPostgresql{})),
	newTarget("databasetools", "DatabaseToolsConnectionGenericJdbcSummary", reflect.TypeOf(databasetools.DatabaseToolsConnectionGenericJdbcSummary{})),
	newTarget("databasetools", "DatabaseToolsConnectionMySqlSummary", reflect.TypeOf(databasetools.DatabaseToolsConnectionMySqlSummary{})),
	newTarget("databasetools", "DatabaseToolsConnectionOracleDatabaseSummary", reflect.TypeOf(databasetools.DatabaseToolsConnectionOracleDatabaseSummary{})),
	newTarget("databasetools", "DatabaseToolsConnectionPostgresqlSummary", reflect.TypeOf(databasetools.DatabaseToolsConnectionPostgresqlSummary{})),

	// Dataflow CRD support
	newTarget("dataflow", "CreateApplicationDetails", reflect.TypeOf(dataflow.CreateApplicationDetails{})),
	newTarget("dataflow", "UpdateApplicationDetails", reflect.TypeOf(dataflow.UpdateApplicationDetails{})),
	newTarget("dataflow", "Application", reflect.TypeOf(dataflow.Application{})),
	newTarget("dataflow", "ApplicationSummary", reflect.TypeOf(dataflow.ApplicationSummary{})),

	// Datascience CRD support
	newTarget("datascience", "CreateProjectDetails", reflect.TypeOf(datascience.CreateProjectDetails{})),
	newTarget("datascience", "UpdateProjectDetails", reflect.TypeOf(datascience.UpdateProjectDetails{})),
	newTarget("datascience", "Project", reflect.TypeOf(datascience.Project{})),
	newTarget("datascience", "ProjectSummary", reflect.TypeOf(datascience.ProjectSummary{})),

	// Opensearch CRD support
	newTarget("opensearch", "CreateOpensearchClusterDetails", reflect.TypeOf(opensearch.CreateOpensearchClusterDetails{})),
	newTarget("opensearch", "UpdateOpensearchClusterDetails", reflect.TypeOf(opensearch.UpdateOpensearchClusterDetails{})),
	newTarget("opensearch", "OpensearchCluster", reflect.TypeOf(opensearch.OpensearchCluster{})),
	newTarget("opensearch", "OpensearchClusterCollection", reflect.TypeOf(opensearch.OpensearchClusterCollection{})),
	newTarget("opensearch", "OpensearchClusterSummary", reflect.TypeOf(opensearch.OpensearchClusterSummary{})),

	// Redis CRD support
	newTarget("redis", "CreateRedisClusterDetails", reflect.TypeOf(redis.CreateRedisClusterDetails{})),
	newTarget("redis", "UpdateRedisClusterDetails", reflect.TypeOf(redis.UpdateRedisClusterDetails{})),
	newTarget("redis", "RedisCluster", reflect.TypeOf(redis.RedisCluster{})),
	newTarget("redis", "RedisClusterCollection", reflect.TypeOf(redis.RedisClusterCollection{})),
	newTarget("redis", "RedisClusterSummary", reflect.TypeOf(redis.RedisClusterSummary{})),
}

var interfaceImplementations = map[string][]reflect.Type{
	qualifiedTypeName(reflect.TypeOf((*mysql.CreateDbSystemSourceDetails)(nil)).Elem()): {
		reflect.TypeOf(mysql.CreateDbSystemSourceFromBackupDetails{}),
		reflect.TypeOf(mysql.CreateDbSystemSourceFromNoneDetails{}),
		reflect.TypeOf(mysql.CreateDbSystemSourceFromPitrDetails{}),
		reflect.TypeOf(mysql.CreateDbSystemSourceImportFromUrlDetails{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.CreateDatabaseToolsConnectionDetails)(nil)).Elem()): {
		reflect.TypeOf(databasetools.CreateDatabaseToolsConnectionGenericJdbcDetails{}),
		reflect.TypeOf(databasetools.CreateDatabaseToolsConnectionMySqlDetails{}),
		reflect.TypeOf(databasetools.CreateDatabaseToolsConnectionOracleDatabaseDetails{}),
		reflect.TypeOf(databasetools.CreateDatabaseToolsConnectionPostgresqlDetails{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.UpdateDatabaseToolsConnectionDetails)(nil)).Elem()): {
		reflect.TypeOf(databasetools.UpdateDatabaseToolsConnectionGenericJdbcDetails{}),
		reflect.TypeOf(databasetools.UpdateDatabaseToolsConnectionMySqlDetails{}),
		reflect.TypeOf(databasetools.UpdateDatabaseToolsConnectionOracleDatabaseDetails{}),
		reflect.TypeOf(databasetools.UpdateDatabaseToolsConnectionPostgresqlDetails{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsConnection)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsConnectionGenericJdbc{}),
		reflect.TypeOf(databasetools.DatabaseToolsConnectionMySql{}),
		reflect.TypeOf(databasetools.DatabaseToolsConnectionOracleDatabase{}),
		reflect.TypeOf(databasetools.DatabaseToolsConnectionPostgresql{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsConnectionSummary)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsConnectionGenericJdbcSummary{}),
		reflect.TypeOf(databasetools.DatabaseToolsConnectionMySqlSummary{}),
		reflect.TypeOf(databasetools.DatabaseToolsConnectionOracleDatabaseSummary{}),
		reflect.TypeOf(databasetools.DatabaseToolsConnectionPostgresqlSummary{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsConnectionOracleDatabaseProxyClient)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsConnectionOracleDatabaseProxyClientNoProxy{}),
		reflect.TypeOf(databasetools.DatabaseToolsConnectionOracleDatabaseProxyClientUserName{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsConnectionOracleDatabaseProxyClientDetails)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsConnectionOracleDatabaseProxyClientNoProxyDetails{}),
		reflect.TypeOf(databasetools.DatabaseToolsConnectionOracleDatabaseProxyClientUserNameDetails{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsConnectionOracleDatabaseProxyClientSummary)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsConnectionOracleDatabaseProxyClientNoProxySummary{}),
		reflect.TypeOf(databasetools.DatabaseToolsConnectionOracleDatabaseProxyClientUserNameSummary{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStoreContent)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStoreContentSecretId{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStoreContentDetails)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStoreContentSecretIdDetails{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStoreContentSummary)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStoreContentSecretIdSummary{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStoreContentGenericJdbc)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStoreContentSecretIdGenericJdbc{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStoreContentGenericJdbcDetails)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStoreContentSecretIdGenericJdbcDetails{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStoreContentGenericJdbcSummary)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStoreContentSecretIdGenericJdbcSummary{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStoreContentMySql)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStoreContentSecretIdMySql{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStoreContentMySqlDetails)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStoreContentSecretIdMySqlDetails{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStoreContentMySqlSummary)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStoreContentSecretIdMySqlSummary{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStoreContentPostgresql)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStoreContentSecretIdPostgresql{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStoreContentPostgresqlDetails)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStoreContentSecretIdPostgresqlDetails{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStoreContentPostgresqlSummary)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStoreContentSecretIdPostgresqlSummary{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStorePassword)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStorePasswordSecretId{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStorePasswordDetails)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStorePasswordSecretIdDetails{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStorePasswordSummary)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStorePasswordSecretIdSummary{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStorePasswordGenericJdbc)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStorePasswordSecretIdGenericJdbc{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStorePasswordGenericJdbcDetails)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStorePasswordSecretIdGenericJdbcDetails{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStorePasswordGenericJdbcSummary)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStorePasswordSecretIdGenericJdbcSummary{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStorePasswordMySql)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStorePasswordSecretIdMySql{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStorePasswordMySqlDetails)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStorePasswordSecretIdMySqlDetails{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStorePasswordMySqlSummary)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStorePasswordSecretIdMySqlSummary{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStorePasswordPostgresql)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStorePasswordSecretIdPostgresql{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStorePasswordPostgresqlDetails)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStorePasswordSecretIdPostgresqlDetails{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsKeyStorePasswordPostgresqlSummary)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsKeyStorePasswordSecretIdPostgresqlSummary{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsUserPassword)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsUserPasswordSecretId{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsUserPasswordDetails)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsUserPasswordSecretIdDetails{}),
	},
	qualifiedTypeName(reflect.TypeOf((*databasetools.DatabaseToolsUserPasswordSummary)(nil)).Elem()): {
		reflect.TypeOf(databasetools.DatabaseToolsUserPasswordSecretIdSummary{}),
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
