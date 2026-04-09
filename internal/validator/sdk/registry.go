package sdk

import (
	"path"
	"reflect"

	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/oracle/oci-go-sdk/v65/containerinstances"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/database"
	"github.com/oracle/oci-go-sdk/v65/dataflow"
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
	newTarget("containerengine", "UpdateClusterDetails", reflect.TypeOf(containerengine.UpdateClusterDetails{})),
	newTarget("containerengine", "Cluster", reflect.TypeOf(containerengine.Cluster{})),
	newTarget("containerengine", "ClusterSummary", reflect.TypeOf(containerengine.ClusterSummary{})),

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

	// Containerinstances CRD support
	newTarget("containerinstances", "CreateContainerInstanceDetails", reflect.TypeOf(containerinstances.CreateContainerInstanceDetails{})),
	newTarget("containerinstances", "UpdateContainerInstanceDetails", reflect.TypeOf(containerinstances.UpdateContainerInstanceDetails{})),
	newTarget("containerinstances", "ContainerInstance", reflect.TypeOf(containerinstances.ContainerInstance{})),
	newTarget("containerinstances", "ContainerInstanceCollection", reflect.TypeOf(containerinstances.ContainerInstanceCollection{})),
	newTarget("containerinstances", "ContainerInstanceSummary", reflect.TypeOf(containerinstances.ContainerInstanceSummary{})),

	// Dataflow CRD support
	newTarget("dataflow", "CreateApplicationDetails", reflect.TypeOf(dataflow.CreateApplicationDetails{})),
	newTarget("dataflow", "UpdateApplicationDetails", reflect.TypeOf(dataflow.UpdateApplicationDetails{})),
	newTarget("dataflow", "Application", reflect.TypeOf(dataflow.Application{})),
	newTarget("dataflow", "ApplicationSummary", reflect.TypeOf(dataflow.ApplicationSummary{})),

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
