package apispec

import (
	"reflect"

	artifactsv1beta1 "github.com/oracle/oci-service-operator/api/artifacts/v1beta1"
	certificatesv1beta1 "github.com/oracle/oci-service-operator/api/certificates/v1beta1"
	certificatesmanagementv1beta1 "github.com/oracle/oci-service-operator/api/certificatesmanagement/v1beta1"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	eventsv1beta1 "github.com/oracle/oci-service-operator/api/events/v1beta1"
	functionsv1beta1 "github.com/oracle/oci-service-operator/api/functions/v1beta1"
	identityv1beta1 "github.com/oracle/oci-service-operator/api/identity/v1beta1"
	keymanagementv1beta1 "github.com/oracle/oci-service-operator/api/keymanagement/v1beta1"
	limitsv1beta1 "github.com/oracle/oci-service-operator/api/limits/v1beta1"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	loggingv1beta1 "github.com/oracle/oci-service-operator/api/logging/v1beta1"
	monitoringv1beta1 "github.com/oracle/oci-service-operator/api/monitoring/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	networkloadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/networkloadbalancer/v1beta1"
	nosqlv1beta1 "github.com/oracle/oci-service-operator/api/nosql/v1beta1"
	objectstoragev1beta1 "github.com/oracle/oci-service-operator/api/objectstorage/v1beta1"
	onsv1beta1 "github.com/oracle/oci-service-operator/api/ons/v1beta1"
	psqlv1beta1 "github.com/oracle/oci-service-operator/api/psql/v1beta1"
	queuev1beta1 "github.com/oracle/oci-service-operator/api/queue/v1beta1"
	secretsv1beta1 "github.com/oracle/oci-service-operator/api/secrets/v1beta1"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	vaultv1beta1 "github.com/oracle/oci-service-operator/api/vault/v1beta1"
	workrequestsv1beta1 "github.com/oracle/oci-service-operator/api/workrequests/v1beta1"
)

type Target struct {
	Name       string
	SpecType   reflect.Type
	SDKStructs []string
}

var targets = []Target{
	{
		Name:     "AutonomousDatabases",
		SpecType: reflect.TypeOf(databasev1beta1.AutonomousDatabasesSpec{}),
		SDKStructs: []string{
			"database.CreateAutonomousDatabaseDetails",
			"database.UpdateAutonomousDatabaseDetails",
		},
	},
	{
		Name:     "MySqlDbSystem",
		SpecType: reflect.TypeOf(mysqlv1beta1.MySqlDbSystemSpec{}),
		SDKStructs: []string{
			"mysql.CreateDbSystemDetails",
			"mysql.UpdateDbSystemDetails",
		},
	},
	{
		Name:     "Stream",
		SpecType: reflect.TypeOf(streamingv1beta1.StreamSpec{}),
		SDKStructs: []string{
			"streaming.CreateStreamDetails",
			"streaming.UpdateStreamDetails",
			"streaming.Stream",
			"streaming.StreamSummary",
		},
	},
	{
		Name:     "Queue",
		SpecType: reflect.TypeOf(queuev1beta1.QueueSpec{}),
		SDKStructs: []string{
			"queue.CreateQueueDetails",
			"queue.UpdateQueueDetails",
			"queue.Queue",
			"queue.QueueSummary",
		},
	},
	{
		Name:     "QueueMessage",
		SpecType: reflect.TypeOf(queuev1beta1.MessageSpec{}),
		SDKStructs: []string{
			"queue.UpdateMessageDetails",
		},
	},
	{
		Name:     "QueueStats",
		SpecType: reflect.TypeOf(queuev1beta1.StatsSpec{}),
		SDKStructs: []string{
			"queue.Stats",
		},
	},
	{
		Name:     "QueueWorkRequest",
		SpecType: reflect.TypeOf(queuev1beta1.WorkRequestSpec{}),
		SDKStructs: []string{
			"queue.WorkRequest",
			"queue.WorkRequestSummary",
		},
	},
	{
		Name:     "QueueWorkRequestError",
		SpecType: reflect.TypeOf(queuev1beta1.WorkRequestErrorSpec{}),
		SDKStructs: []string{
			"queue.WorkRequestError",
		},
	},
	{
		Name:     "FunctionsApplication",
		SpecType: reflect.TypeOf(functionsv1beta1.ApplicationSpec{}),
		SDKStructs: []string{
			"functions.CreateApplicationDetails",
			"functions.UpdateApplicationDetails",
			"functions.Application",
			"functions.ApplicationSummary",
		},
	},
	{
		Name:     "FunctionsFunction",
		SpecType: reflect.TypeOf(functionsv1beta1.FunctionSpec{}),
		SDKStructs: []string{
			"functions.CreateFunctionDetails",
			"functions.UpdateFunctionDetails",
			"functions.Function",
			"functions.FunctionSummary",
		},
	},
	{
		Name:     "FunctionsPbfListing",
		SpecType: reflect.TypeOf(functionsv1beta1.PbfListingSpec{}),
		SDKStructs: []string{
			"functions.PbfListing",
			"functions.PbfListingVersionSummary",
			"functions.PbfListingSummary",
		},
	},
	{
		Name:     "FunctionsPbfListingVersion",
		SpecType: reflect.TypeOf(functionsv1beta1.PbfListingVersionSpec{}),
		SDKStructs: []string{
			"functions.PbfListingVersion",
			"functions.PbfListingVersionSummary",
		},
	},
	{
		Name:     "FunctionsTrigger",
		SpecType: reflect.TypeOf(functionsv1beta1.TriggerSpec{}),
		SDKStructs: []string{
			"functions.Trigger",
			"functions.TriggerSummary",
		},
	},
	{
		Name:     "NoSQLIndex",
		SpecType: reflect.TypeOf(nosqlv1beta1.IndexSpec{}),
		SDKStructs: []string{
			"nosql.CreateIndexDetails",
			"nosql.Index",
			"nosql.IndexSummary",
		},
	},
	{
		Name:     "NoSQLReplica",
		SpecType: reflect.TypeOf(nosqlv1beta1.ReplicaSpec{}),
		SDKStructs: []string{
			"nosql.CreateReplicaDetails",
			"nosql.Replica",
		},
	},
	{
		Name:     "NoSQLRow",
		SpecType: reflect.TypeOf(nosqlv1beta1.RowSpec{}),
		SDKStructs: []string{
			"nosql.UpdateRowDetails",
			"nosql.Row",
		},
	},
	{
		Name:     "NoSQLTable",
		SpecType: reflect.TypeOf(nosqlv1beta1.TableSpec{}),
		SDKStructs: []string{
			"nosql.CreateTableDetails",
			"nosql.UpdateTableDetails",
			"nosql.Table",
			"nosql.TableSummary",
		},
	},
	{
		Name:     "NoSQLTableUsage",
		SpecType: reflect.TypeOf(nosqlv1beta1.TableUsageSpec{}),
		SDKStructs: []string{
			"nosql.TableUsageSummary",
		},
	},
	{
		Name:     "NoSQLWorkRequest",
		SpecType: reflect.TypeOf(nosqlv1beta1.WorkRequestSpec{}),
		SDKStructs: []string{
			"nosql.WorkRequest",
			"nosql.WorkRequestSummary",
		},
	},
	{
		Name:     "NoSQLWorkRequestError",
		SpecType: reflect.TypeOf(nosqlv1beta1.WorkRequestErrorSpec{}),
		SDKStructs: []string{
			"nosql.WorkRequestError",
		},
	},
	{
		Name:     "ObjectStorageBucket",
		SpecType: reflect.TypeOf(objectstoragev1beta1.BucketSpec{}),
		SDKStructs: []string{
			"objectstorage.CreateBucketDetails",
			"objectstorage.UpdateBucketDetails",
			"objectstorage.Bucket",
			"objectstorage.BucketSummary",
		},
	},
	{
		Name:     "ObjectStorageMultipartUpload",
		SpecType: reflect.TypeOf(objectstoragev1beta1.MultipartUploadSpec{}),
		SDKStructs: []string{
			"objectstorage.CreateMultipartUploadDetails",
			"objectstorage.MultipartUpload",
		},
	},
	{
		Name:     "ObjectStorageMultipartUploadPart",
		SpecType: reflect.TypeOf(objectstoragev1beta1.MultipartUploadPartSpec{}),
		SDKStructs: []string{
			"objectstorage.MultipartUploadPartSummary",
		},
	},
	{
		Name:     "ObjectStorageNamespaceMetadata",
		SpecType: reflect.TypeOf(objectstoragev1beta1.NamespaceMetadataSpec{}),
		SDKStructs: []string{
			"objectstorage.UpdateNamespaceMetadataDetails",
			"objectstorage.NamespaceMetadata",
		},
	},
	{
		Name:     "ObjectStorageObject",
		SpecType: reflect.TypeOf(objectstoragev1beta1.ObjectSpec{}),
		SDKStructs: []string{
			"objectstorage.ObjectVersionSummary",
			"objectstorage.ObjectSummary",
		},
	},
	{
		Name:     "ObjectStorageObjectLifecyclePolicy",
		SpecType: reflect.TypeOf(objectstoragev1beta1.ObjectLifecyclePolicySpec{}),
		SDKStructs: []string{
			"objectstorage.ObjectLifecyclePolicy",
		},
	},
	{
		Name:     "ObjectStorageObjectStorageTier",
		SpecType: reflect.TypeOf(objectstoragev1beta1.ObjectStorageTierSpec{}),
		SDKStructs: []string{
			"objectstorage.UpdateObjectStorageTierDetails",
		},
	},
	{
		Name:     "ObjectStorageObjectVersion",
		SpecType: reflect.TypeOf(objectstoragev1beta1.ObjectVersionSpec{}),
		SDKStructs: []string{
			"objectstorage.ObjectVersionSummary",
		},
	},
	{
		Name:     "ObjectStoragePreauthenticatedRequest",
		SpecType: reflect.TypeOf(objectstoragev1beta1.PreauthenticatedRequestSpec{}),
		SDKStructs: []string{
			"objectstorage.CreatePreauthenticatedRequestDetails",
			"objectstorage.PreauthenticatedRequest",
			"objectstorage.PreauthenticatedRequestSummary",
		},
	},
	{
		Name:     "ObjectStorageReplicationPolicy",
		SpecType: reflect.TypeOf(objectstoragev1beta1.ReplicationPolicySpec{}),
		SDKStructs: []string{
			"objectstorage.CreateReplicationPolicyDetails",
			"objectstorage.ReplicationPolicy",
			"objectstorage.ReplicationPolicySummary",
		},
	},
	{
		Name:     "ObjectStorageReplicationSource",
		SpecType: reflect.TypeOf(objectstoragev1beta1.ReplicationSourceSpec{}),
		SDKStructs: []string{
			"objectstorage.ReplicationSource",
		},
	},
	{
		Name:     "ObjectStorageRetentionRule",
		SpecType: reflect.TypeOf(objectstoragev1beta1.RetentionRuleSpec{}),
		SDKStructs: []string{
			"objectstorage.CreateRetentionRuleDetails",
			"objectstorage.UpdateRetentionRuleDetails",
			"objectstorage.RetentionRule",
			"objectstorage.RetentionRuleSummary",
		},
	},
	{
		Name:     "ObjectStorageWorkRequest",
		SpecType: reflect.TypeOf(objectstoragev1beta1.WorkRequestSpec{}),
		SDKStructs: []string{
			"objectstorage.WorkRequest",
			"objectstorage.WorkRequestSummary",
		},
	},
	{
		Name:     "ObjectStorageWorkRequestError",
		SpecType: reflect.TypeOf(objectstoragev1beta1.WorkRequestErrorSpec{}),
		SDKStructs: []string{
			"objectstorage.WorkRequestError",
		},
	},
	{
		Name:     "NotificationTopic",
		SpecType: reflect.TypeOf(onsv1beta1.TopicSpec{}),
		SDKStructs: []string{
			"ons.CreateTopicDetails",
		},
	},
	{
		Name:     "ONSSubscription",
		SpecType: reflect.TypeOf(onsv1beta1.SubscriptionSpec{}),
		SDKStructs: []string{
			"ons.CreateSubscriptionDetails",
			"ons.UpdateSubscriptionDetails",
			"ons.Subscription",
			"ons.SubscriptionSummary",
		},
	},
	{
		Name:     "LogGroup",
		SpecType: reflect.TypeOf(loggingv1beta1.LogGroupSpec{}),
		SDKStructs: []string{
			"logging.CreateLogGroupDetails",
			"logging.UpdateLogGroupDetails",
			"logging.LogGroup",
			"logging.LogGroupSummary",
		},
	},
	{
		Name:     "LoggingLog",
		SpecType: reflect.TypeOf(loggingv1beta1.LogSpec{}),
		SDKStructs: []string{
			"logging.CreateLogDetails",
			"logging.UpdateLogDetails",
			"logging.Log",
			"logging.LogSummary",
		},
	},
	{
		Name:     "LoggingLogSavedSearch",
		SpecType: reflect.TypeOf(loggingv1beta1.LogSavedSearchSpec{}),
		SDKStructs: []string{
			"logging.CreateLogSavedSearchDetails",
			"logging.UpdateLogSavedSearchDetails",
			"logging.LogSavedSearch",
			"logging.LogSavedSearchSummary",
		},
	},
	{
		Name:     "LoggingService",
		SpecType: reflect.TypeOf(loggingv1beta1.ServiceSpec{}),
		SDKStructs: []string{
			"logging.ServiceSummary",
		},
	},
	{
		Name:     "LoggingUnifiedAgentConfiguration",
		SpecType: reflect.TypeOf(loggingv1beta1.UnifiedAgentConfigurationSpec{}),
		SDKStructs: []string{
			"logging.CreateUnifiedAgentConfigurationDetails",
			"logging.UpdateUnifiedAgentConfigurationDetails",
			"logging.UnifiedAgentConfiguration",
			"logging.UnifiedAgentConfigurationSummary",
		},
	},
	{
		Name:     "LoggingWorkRequest",
		SpecType: reflect.TypeOf(loggingv1beta1.WorkRequestSpec{}),
		SDKStructs: []string{
			"logging.WorkRequest",
			"logging.WorkRequestSummary",
		},
	},
	{
		Name:     "LoggingWorkRequestError",
		SpecType: reflect.TypeOf(loggingv1beta1.WorkRequestErrorSpec{}),
		SDKStructs: []string{
			"logging.WorkRequestError",
		},
	},
	{
		Name:     "LoggingWorkRequestLog",
		SpecType: reflect.TypeOf(loggingv1beta1.WorkRequestLogSpec{}),
		SDKStructs: []string{
			"logging.WorkRequestLog",
		},
	},
	{
		Name:     "PSQLBackup",
		SpecType: reflect.TypeOf(psqlv1beta1.BackupSpec{}),
		SDKStructs: []string{
			"psql.CreateBackupDetails",
			"psql.UpdateBackupDetails",
			"psql.Backup",
			"psql.BackupSummary",
		},
	},
	{
		Name:     "PSQLConfiguration",
		SpecType: reflect.TypeOf(psqlv1beta1.ConfigurationSpec{}),
		SDKStructs: []string{
			"psql.CreateConfigurationDetails",
			"psql.UpdateConfigurationDetails",
			"psql.Configuration",
			"psql.ConfigurationSummary",
		},
	},
	{
		Name:     "PSQLDbSystemDbInstance",
		SpecType: reflect.TypeOf(psqlv1beta1.DbSystemDbInstanceSpec{}),
		SDKStructs: []string{
			"psql.UpdateDbSystemDbInstanceDetails",
		},
	},
	{
		Name:     "PSQLDefaultConfiguration",
		SpecType: reflect.TypeOf(psqlv1beta1.DefaultConfigurationSpec{}),
		SDKStructs: []string{
			"psql.DefaultConfiguration",
			"psql.DefaultConfigurationSummary",
		},
	},
	{
		Name:     "PSQLShape",
		SpecType: reflect.TypeOf(psqlv1beta1.ShapeSpec{}),
		SDKStructs: []string{
			"psql.ShapeSummary",
		},
	},
	{
		Name:     "PSQLWorkRequest",
		SpecType: reflect.TypeOf(psqlv1beta1.WorkRequestSpec{}),
		SDKStructs: []string{
			"psql.WorkRequest",
			"psql.WorkRequestSummary",
		},
	},
	{
		Name:     "PSQLWorkRequestError",
		SpecType: reflect.TypeOf(psqlv1beta1.WorkRequestErrorSpec{}),
		SDKStructs: []string{
			"psql.WorkRequestError",
		},
	},
	{
		Name:     "PostgreSQLDbSystem",
		SpecType: reflect.TypeOf(psqlv1beta1.DbSystemSpec{}),
		SDKStructs: []string{
			"psql.CreateDbSystemDetails",
			"psql.UpdateDbSystemDetails",
			"psql.DbSystem",
			"psql.DbSystemSummary",
		},
	},
	{
		Name:     "EventsRule",
		SpecType: reflect.TypeOf(eventsv1beta1.RuleSpec{}),
		SDKStructs: []string{
			"events.CreateRuleDetails",
			"events.UpdateRuleDetails",
			"events.Rule",
			"events.RuleSummary",
		},
	},
	{
		Name:     "MonitoringAlarm",
		SpecType: reflect.TypeOf(monitoringv1beta1.AlarmSpec{}),
		SDKStructs: []string{
			"monitoring.CreateAlarmDetails",
			"monitoring.UpdateAlarmDetails",
			"monitoring.Alarm",
			"monitoring.AlarmSummary",
		},
	},
	{
		Name:     "MonitoringAlarmStatus",
		SpecType: reflect.TypeOf(monitoringv1beta1.AlarmStatusSpec{}),
		SDKStructs: []string{
			"monitoring.AlarmStatusSummary",
		},
	},
	{
		Name:     "MonitoringAlarmSuppression",
		SpecType: reflect.TypeOf(monitoringv1beta1.AlarmSuppressionSpec{}),
		SDKStructs: []string{
			"monitoring.CreateAlarmSuppressionDetails",
			"monitoring.AlarmSuppression",
			"monitoring.AlarmSuppressionSummary",
		},
	},
	{
		Name:     "MonitoringMetric",
		SpecType: reflect.TypeOf(monitoringv1beta1.MetricSpec{}),
		SDKStructs: []string{
			"monitoring.Metric",
		},
	},
	{
		Name:     "DNSResolver",
		SpecType: reflect.TypeOf(dnsv1beta1.ResolverSpec{}),
		SDKStructs: []string{
			"dns.UpdateResolverDetails",
			"dns.Resolver",
			"dns.ResolverSummary",
		},
	},
	{
		Name:     "DNSSteeringPolicy",
		SpecType: reflect.TypeOf(dnsv1beta1.SteeringPolicySpec{}),
		SDKStructs: []string{
			"dns.CreateSteeringPolicyDetails",
			"dns.UpdateSteeringPolicyDetails",
			"dns.SteeringPolicy",
			"dns.SteeringPolicySummary",
		},
	},
	{
		Name:     "DNSSteeringPolicyAttachment",
		SpecType: reflect.TypeOf(dnsv1beta1.SteeringPolicyAttachmentSpec{}),
		SDKStructs: []string{
			"dns.CreateSteeringPolicyAttachmentDetails",
			"dns.UpdateSteeringPolicyAttachmentDetails",
			"dns.SteeringPolicyAttachment",
			"dns.SteeringPolicyAttachmentSummary",
		},
	},
	{
		Name:     "DNSTsigKey",
		SpecType: reflect.TypeOf(dnsv1beta1.TsigKeySpec{}),
		SDKStructs: []string{
			"dns.CreateTsigKeyDetails",
			"dns.UpdateTsigKeyDetails",
			"dns.TsigKey",
			"dns.TsigKeySummary",
		},
	},
	{
		Name:     "DNSView",
		SpecType: reflect.TypeOf(dnsv1beta1.ViewSpec{}),
		SDKStructs: []string{
			"dns.CreateViewDetails",
			"dns.UpdateViewDetails",
			"dns.View",
			"dns.ViewSummary",
		},
	},
	{
		Name:     "DNSZone",
		SpecType: reflect.TypeOf(dnsv1beta1.ZoneSpec{}),
		SDKStructs: []string{
			"dns.CreateZoneDetails",
			"dns.UpdateZoneDetails",
			"dns.Zone",
			"dns.ZoneSummary",
		},
	},
	{
		Name:     "DNSZoneTransferServer",
		SpecType: reflect.TypeOf(dnsv1beta1.ZoneTransferServerSpec{}),
		SDKStructs: []string{
			"dns.ZoneTransferServer",
		},
	},
	{
		Name:     "LoadBalancer",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.LoadBalancerSpec{}),
		SDKStructs: []string{
			"loadbalancer.CreateLoadBalancerDetails",
			"loadbalancer.UpdateLoadBalancerDetails",
			"loadbalancer.LoadBalancer",
		},
	},
	{
		Name:     "LoadBalancerBackend",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.BackendSpec{}),
		SDKStructs: []string{
			"loadbalancer.CreateBackendDetails",
			"loadbalancer.UpdateBackendDetails",
			"loadbalancer.Backend",
		},
	},
	{
		Name:     "LoadBalancerBackendHealth",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.BackendHealthSpec{}),
		SDKStructs: []string{
			"loadbalancer.BackendHealth",
		},
	},
	{
		Name:     "LoadBalancerBackendSet",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.BackendSetSpec{}),
		SDKStructs: []string{
			"loadbalancer.CreateBackendSetDetails",
			"loadbalancer.UpdateBackendSetDetails",
			"loadbalancer.BackendSet",
		},
	},
	{
		Name:     "LoadBalancerBackendSetHealth",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.BackendSetHealthSpec{}),
		SDKStructs: []string{
			"loadbalancer.BackendSetHealth",
		},
	},
	{
		Name:     "LoadBalancerCertificate",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.CertificateSpec{}),
		SDKStructs: []string{
			"loadbalancer.CreateCertificateDetails",
			"loadbalancer.Certificate",
		},
	},
	{
		Name:     "LoadBalancerHealthChecker",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.HealthCheckerSpec{}),
		SDKStructs: []string{
			"loadbalancer.UpdateHealthCheckerDetails",
			"loadbalancer.HealthChecker",
		},
	},
	{
		Name:     "LoadBalancerHostname",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.HostnameSpec{}),
		SDKStructs: []string{
			"loadbalancer.CreateHostnameDetails",
			"loadbalancer.UpdateHostnameDetails",
			"loadbalancer.Hostname",
		},
	},
	{
		Name:     "LoadBalancerListener",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.ListenerSpec{}),
		SDKStructs: []string{
			"loadbalancer.CreateListenerDetails",
			"loadbalancer.UpdateListenerDetails",
			"loadbalancer.Listener",
		},
	},
	{
		Name:     "LoadBalancerListenerRule",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.ListenerRuleSpec{}),
		SDKStructs: []string{
			"loadbalancer.ListenerRuleSummary",
		},
	},
	{
		Name:     "LoadBalancerLoadBalancerHealth",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.LoadBalancerHealthSpec{}),
		SDKStructs: []string{
			"loadbalancer.LoadBalancerHealth",
			"loadbalancer.LoadBalancerHealthSummary",
		},
	},
	{
		Name:     "LoadBalancerLoadBalancerShape",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.LoadBalancerShapeSpec{}),
		SDKStructs: []string{
			"loadbalancer.UpdateLoadBalancerShapeDetails",
			"loadbalancer.LoadBalancerShape",
		},
	},
	{
		Name:     "LoadBalancerPathRouteSet",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.PathRouteSetSpec{}),
		SDKStructs: []string{
			"loadbalancer.CreatePathRouteSetDetails",
			"loadbalancer.UpdatePathRouteSetDetails",
			"loadbalancer.PathRouteSet",
		},
	},
	{
		Name:     "LoadBalancerRoutingPolicy",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.RoutingPolicySpec{}),
		SDKStructs: []string{
			"loadbalancer.CreateRoutingPolicyDetails",
			"loadbalancer.UpdateRoutingPolicyDetails",
			"loadbalancer.RoutingPolicy",
		},
	},
	{
		Name:     "LoadBalancerRuleSet",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.RuleSetSpec{}),
		SDKStructs: []string{
			"loadbalancer.CreateRuleSetDetails",
			"loadbalancer.UpdateRuleSetDetails",
			"loadbalancer.RuleSet",
		},
	},
	{
		Name:     "LoadBalancerWorkRequest",
		SpecType: reflect.TypeOf(loadbalancerv1beta1.WorkRequestSpec{}),
		SDKStructs: []string{
			"loadbalancer.WorkRequest",
		},
	},
	{
		Name:     "NetworkLoadBalancer",
		SpecType: reflect.TypeOf(networkloadbalancerv1beta1.NetworkLoadBalancerSpec{}),
		SDKStructs: []string{
			"networkloadbalancer.CreateNetworkLoadBalancerDetails",
			"networkloadbalancer.UpdateNetworkLoadBalancerDetails",
			"networkloadbalancer.NetworkLoadBalancer",
			"networkloadbalancer.NetworkLoadBalancerSummary",
		},
	},
	{
		Name:     "NetworkLoadBalancerBackend",
		SpecType: reflect.TypeOf(networkloadbalancerv1beta1.BackendSpec{}),
		SDKStructs: []string{
			"networkloadbalancer.CreateBackendDetails",
			"networkloadbalancer.UpdateBackendDetails",
			"networkloadbalancer.Backend",
			"networkloadbalancer.BackendSummary",
		},
	},
	{
		Name:     "NetworkLoadBalancerBackendHealth",
		SpecType: reflect.TypeOf(networkloadbalancerv1beta1.BackendHealthSpec{}),
		SDKStructs: []string{
			"networkloadbalancer.BackendHealth",
		},
	},
	{
		Name:     "NetworkLoadBalancerBackendSet",
		SpecType: reflect.TypeOf(networkloadbalancerv1beta1.BackendSetSpec{}),
		SDKStructs: []string{
			"networkloadbalancer.CreateBackendSetDetails",
			"networkloadbalancer.UpdateBackendSetDetails",
			"networkloadbalancer.BackendSet",
			"networkloadbalancer.BackendSetSummary",
		},
	},
	{
		Name:     "NetworkLoadBalancerBackendSetHealth",
		SpecType: reflect.TypeOf(networkloadbalancerv1beta1.BackendSetHealthSpec{}),
		SDKStructs: []string{
			"networkloadbalancer.BackendSetHealth",
		},
	},
	{
		Name:     "NetworkLoadBalancerHealthChecker",
		SpecType: reflect.TypeOf(networkloadbalancerv1beta1.HealthCheckerSpec{}),
		SDKStructs: []string{
			"networkloadbalancer.UpdateHealthCheckerDetails",
			"networkloadbalancer.HealthChecker",
		},
	},
	{
		Name:     "NetworkLoadBalancerListener",
		SpecType: reflect.TypeOf(networkloadbalancerv1beta1.ListenerSpec{}),
		SDKStructs: []string{
			"networkloadbalancer.CreateListenerDetails",
			"networkloadbalancer.UpdateListenerDetails",
			"networkloadbalancer.Listener",
			"networkloadbalancer.ListenerSummary",
		},
	},
	{
		Name:     "NetworkLoadBalancerNetworkLoadBalancerHealth",
		SpecType: reflect.TypeOf(networkloadbalancerv1beta1.NetworkLoadBalancerHealthSpec{}),
		SDKStructs: []string{
			"networkloadbalancer.NetworkLoadBalancerHealth",
			"networkloadbalancer.NetworkLoadBalancerHealthSummary",
		},
	},
	{
		Name:     "NetworkLoadBalancerWorkRequest",
		SpecType: reflect.TypeOf(networkloadbalancerv1beta1.WorkRequestSpec{}),
		SDKStructs: []string{
			"networkloadbalancer.WorkRequest",
			"networkloadbalancer.WorkRequestSummary",
		},
	},
	{
		Name:     "NetworkLoadBalancerWorkRequestError",
		SpecType: reflect.TypeOf(networkloadbalancerv1beta1.WorkRequestErrorSpec{}),
		SDKStructs: []string{
			"networkloadbalancer.WorkRequestError",
		},
	},
	{
		Name:     "ArtifactsContainerConfiguration",
		SpecType: reflect.TypeOf(artifactsv1beta1.ContainerConfigurationSpec{}),
		SDKStructs: []string{
			"artifacts.UpdateContainerConfigurationDetails",
			"artifacts.ContainerConfiguration",
		},
	},
	{
		Name:     "ArtifactsContainerImage",
		SpecType: reflect.TypeOf(artifactsv1beta1.ContainerImageSpec{}),
		SDKStructs: []string{
			"artifacts.UpdateContainerImageDetails",
			"artifacts.ContainerImage",
			"artifacts.ContainerImageSummary",
		},
	},
	{
		Name:     "ArtifactsContainerImageSignature",
		SpecType: reflect.TypeOf(artifactsv1beta1.ContainerImageSignatureSpec{}),
		SDKStructs: []string{
			"artifacts.CreateContainerImageSignatureDetails",
			"artifacts.UpdateContainerImageSignatureDetails",
			"artifacts.ContainerImageSignature",
			"artifacts.ContainerImageSignatureSummary",
		},
	},
	{
		Name:     "ArtifactsContainerRepository",
		SpecType: reflect.TypeOf(artifactsv1beta1.ContainerRepositorySpec{}),
		SDKStructs: []string{
			"artifacts.CreateContainerRepositoryDetails",
			"artifacts.UpdateContainerRepositoryDetails",
			"artifacts.ContainerRepository",
			"artifacts.ContainerRepositorySummary",
		},
	},
	{
		Name:     "ArtifactsGenericArtifact",
		SpecType: reflect.TypeOf(artifactsv1beta1.GenericArtifactSpec{}),
		SDKStructs: []string{
			"artifacts.UpdateGenericArtifactDetails",
			"artifacts.GenericArtifact",
			"artifacts.GenericArtifactSummary",
		},
	},
	{
		Name:     "ArtifactsGenericArtifactByPath",
		SpecType: reflect.TypeOf(artifactsv1beta1.GenericArtifactByPathSpec{}),
		SDKStructs: []string{
			"artifacts.UpdateGenericArtifactByPathDetails",
		},
	},
	{
		Name:     "CertificatesCaBundle",
		SpecType: reflect.TypeOf(certificatesv1beta1.CaBundleSpec{}),
		SDKStructs: []string{
			"certificates.CaBundle",
		},
	},
	{
		Name:     "CertificatesCertificateAuthorityBundle",
		SpecType: reflect.TypeOf(certificatesv1beta1.CertificateAuthorityBundleSpec{}),
		SDKStructs: []string{
			"certificates.CertificateAuthorityBundle",
			"certificates.CertificateAuthorityBundleVersionSummary",
		},
	},
	{
		Name:     "CertificatesCertificateAuthorityBundleVersion",
		SpecType: reflect.TypeOf(certificatesv1beta1.CertificateAuthorityBundleVersionSpec{}),
		SDKStructs: []string{
			"certificates.CertificateAuthorityBundleVersionSummary",
		},
	},
	{
		Name:     "CertificatesCertificateBundle",
		SpecType: reflect.TypeOf(certificatesv1beta1.CertificateBundleSpec{}),
		SDKStructs: []string{
			"certificates.CertificateBundlePublicOnly",
			"certificates.CertificateBundleVersionSummary",
		},
	},
	{
		Name:     "CertificatesCertificateBundleVersion",
		SpecType: reflect.TypeOf(certificatesv1beta1.CertificateBundleVersionSpec{}),
		SDKStructs: []string{
			"certificates.CertificateBundleVersionSummary",
		},
	},
	{
		Name:     "CertificatesManagementAssociation",
		SpecType: reflect.TypeOf(certificatesmanagementv1beta1.AssociationSpec{}),
		SDKStructs: []string{
			"certificatesmanagement.Association",
			"certificatesmanagement.AssociationSummary",
		},
	},
	{
		Name:     "CertificatesManagementCaBundle",
		SpecType: reflect.TypeOf(certificatesmanagementv1beta1.CaBundleSpec{}),
		SDKStructs: []string{
			"certificatesmanagement.CreateCaBundleDetails",
			"certificatesmanagement.UpdateCaBundleDetails",
			"certificatesmanagement.CaBundle",
			"certificatesmanagement.CaBundleSummary",
		},
	},
	{
		Name:     "CertificatesManagementCertificate",
		SpecType: reflect.TypeOf(certificatesmanagementv1beta1.CertificateSpec{}),
		SDKStructs: []string{
			"certificatesmanagement.CreateCertificateDetails",
			"certificatesmanagement.UpdateCertificateDetails",
			"certificatesmanagement.Certificate",
			"certificatesmanagement.CertificateVersionSummary",
			"certificatesmanagement.CertificateSummary",
		},
	},
	{
		Name:     "CertificatesManagementCertificateAuthority",
		SpecType: reflect.TypeOf(certificatesmanagementv1beta1.CertificateAuthoritySpec{}),
		SDKStructs: []string{
			"certificatesmanagement.CreateCertificateAuthorityDetails",
			"certificatesmanagement.UpdateCertificateAuthorityDetails",
			"certificatesmanagement.CertificateAuthority",
			"certificatesmanagement.CertificateAuthorityVersionSummary",
			"certificatesmanagement.CertificateAuthoritySummary",
		},
	},
	{
		Name:     "CertificatesManagementCertificateAuthorityVersion",
		SpecType: reflect.TypeOf(certificatesmanagementv1beta1.CertificateAuthorityVersionSpec{}),
		SDKStructs: []string{
			"certificatesmanagement.CertificateAuthorityVersion",
			"certificatesmanagement.CertificateAuthorityVersionSummary",
		},
	},
	{
		Name:     "CertificatesManagementCertificateVersion",
		SpecType: reflect.TypeOf(certificatesmanagementv1beta1.CertificateVersionSpec{}),
		SDKStructs: []string{
			"certificatesmanagement.CertificateVersion",
			"certificatesmanagement.CertificateVersionSummary",
		},
	},
	{
		Name:     "ContainerEngineAddon",
		SpecType: reflect.TypeOf(containerenginev1beta1.AddonSpec{}),
		SDKStructs: []string{
			"containerengine.UpdateAddonDetails",
			"containerengine.Addon",
			"containerengine.AddonSummary",
		},
	},
	{
		Name:     "ContainerEngineAddonOption",
		SpecType: reflect.TypeOf(containerenginev1beta1.AddonOptionSpec{}),
		SDKStructs: []string{
			"containerengine.AddonOptionSummary",
		},
	},
	{
		Name:     "ContainerEngineCluster",
		SpecType: reflect.TypeOf(containerenginev1beta1.ClusterSpec{}),
		SDKStructs: []string{
			"containerengine.CreateClusterDetails",
			"containerengine.UpdateClusterDetails",
			"containerengine.Cluster",
			"containerengine.ClusterSummary",
		},
	},
	{
		Name:     "ContainerEngineClusterEndpointConfig",
		SpecType: reflect.TypeOf(containerenginev1beta1.ClusterEndpointConfigSpec{}),
		SDKStructs: []string{
			"containerengine.CreateClusterEndpointConfigDetails",
			"containerengine.UpdateClusterEndpointConfigDetails",
			"containerengine.ClusterEndpointConfig",
		},
	},
	{
		Name:     "ContainerEngineClusterMigrateToNativeVcnStatus",
		SpecType: reflect.TypeOf(containerenginev1beta1.ClusterMigrateToNativeVcnStatusSpec{}),
		SDKStructs: []string{
			"containerengine.ClusterMigrateToNativeVcnStatus",
		},
	},
	{
		Name:     "ContainerEngineCredentialRotationStatus",
		SpecType: reflect.TypeOf(containerenginev1beta1.CredentialRotationStatusSpec{}),
		SDKStructs: []string{
			"containerengine.CredentialRotationStatus",
		},
	},
	{
		Name:     "ContainerEngineNode",
		SpecType: reflect.TypeOf(containerenginev1beta1.NodeSpec{}),
		SDKStructs: []string{
			"containerengine.Node",
		},
	},
	{
		Name:     "ContainerEngineNodePool",
		SpecType: reflect.TypeOf(containerenginev1beta1.NodePoolSpec{}),
		SDKStructs: []string{
			"containerengine.CreateNodePoolDetails",
			"containerengine.UpdateNodePoolDetails",
			"containerengine.NodePool",
			"containerengine.NodePoolSummary",
		},
	},
	{
		Name:     "ContainerEnginePodShape",
		SpecType: reflect.TypeOf(containerenginev1beta1.PodShapeSpec{}),
		SDKStructs: []string{
			"containerengine.PodShape",
			"containerengine.PodShapeSummary",
		},
	},
	{
		Name:     "ContainerEngineVirtualNode",
		SpecType: reflect.TypeOf(containerenginev1beta1.VirtualNodeSpec{}),
		SDKStructs: []string{
			"containerengine.VirtualNode",
			"containerengine.VirtualNodeSummary",
		},
	},
	{
		Name:     "ContainerEngineVirtualNodePool",
		SpecType: reflect.TypeOf(containerenginev1beta1.VirtualNodePoolSpec{}),
		SDKStructs: []string{
			"containerengine.CreateVirtualNodePoolDetails",
			"containerengine.UpdateVirtualNodePoolDetails",
			"containerengine.VirtualNodePool",
			"containerengine.VirtualNodePoolSummary",
		},
	},
	{
		Name:     "ContainerEngineWorkRequest",
		SpecType: reflect.TypeOf(containerenginev1beta1.WorkRequestSpec{}),
		SDKStructs: []string{
			"containerengine.WorkRequest",
			"containerengine.WorkRequestSummary",
		},
	},
	{
		Name:     "ContainerEngineWorkRequestError",
		SpecType: reflect.TypeOf(containerenginev1beta1.WorkRequestErrorSpec{}),
		SDKStructs: []string{
			"containerengine.WorkRequestError",
		},
	},
	{
		Name:     "ContainerEngineWorkloadMapping",
		SpecType: reflect.TypeOf(containerenginev1beta1.WorkloadMappingSpec{}),
		SDKStructs: []string{
			"containerengine.CreateWorkloadMappingDetails",
			"containerengine.UpdateWorkloadMappingDetails",
			"containerengine.WorkloadMapping",
			"containerengine.WorkloadMappingSummary",
		},
	},
	{
		Name:     "IdentityAllowedDomainLicenseType",
		SpecType: reflect.TypeOf(identityv1beta1.AllowedDomainLicenseTypeSpec{}),
		SDKStructs: []string{
			"identity.AllowedDomainLicenseTypeSummary",
		},
	},
	{
		Name:     "IdentityApiKey",
		SpecType: reflect.TypeOf(identityv1beta1.ApiKeySpec{}),
		SDKStructs: []string{
			"identity.CreateApiKeyDetails",
			"identity.ApiKey",
		},
	},
	{
		Name:     "IdentityAuthToken",
		SpecType: reflect.TypeOf(identityv1beta1.AuthTokenSpec{}),
		SDKStructs: []string{
			"identity.CreateAuthTokenDetails",
			"identity.UpdateAuthTokenDetails",
			"identity.AuthToken",
		},
	},
	{
		Name:     "IdentityAuthenticationPolicy",
		SpecType: reflect.TypeOf(identityv1beta1.AuthenticationPolicySpec{}),
		SDKStructs: []string{
			"identity.UpdateAuthenticationPolicyDetails",
			"identity.AuthenticationPolicy",
		},
	},
	{
		Name:     "IdentityAvailabilityDomain",
		SpecType: reflect.TypeOf(identityv1beta1.AvailabilityDomainSpec{}),
		SDKStructs: []string{
			"identity.AvailabilityDomain",
		},
	},
	{
		Name:     "IdentityBulkActionResourceType",
		SpecType: reflect.TypeOf(identityv1beta1.BulkActionResourceTypeSpec{}),
		SDKStructs: []string{
			"identity.BulkActionResourceType",
		},
	},
	{
		Name:     "IdentityBulkEditTagsResourceType",
		SpecType: reflect.TypeOf(identityv1beta1.BulkEditTagsResourceTypeSpec{}),
		SDKStructs: []string{
			"identity.BulkEditTagsResourceType",
		},
	},
	{
		Name:     "IdentityCompartment",
		SpecType: reflect.TypeOf(identityv1beta1.CompartmentSpec{}),
		SDKStructs: []string{
			"identity.CreateCompartmentDetails",
			"identity.UpdateCompartmentDetails",
			"identity.Compartment",
		},
	},
	{
		Name:     "IdentityCustomerSecretKey",
		SpecType: reflect.TypeOf(identityv1beta1.CustomerSecretKeySpec{}),
		SDKStructs: []string{
			"identity.CreateCustomerSecretKeyDetails",
			"identity.UpdateCustomerSecretKeyDetails",
			"identity.CustomerSecretKey",
			"identity.CustomerSecretKeySummary",
		},
	},
	{
		Name:     "IdentityDbCredential",
		SpecType: reflect.TypeOf(identityv1beta1.DbCredentialSpec{}),
		SDKStructs: []string{
			"identity.CreateDbCredentialDetails",
			"identity.DbCredential",
			"identity.DbCredentialSummary",
		},
	},
	{
		Name:     "IdentityDomain",
		SpecType: reflect.TypeOf(identityv1beta1.DomainSpec{}),
		SDKStructs: []string{
			"identity.CreateDomainDetails",
			"identity.UpdateDomainDetails",
			"identity.Domain",
			"identity.DomainSummary",
		},
	},
	{
		Name:     "IdentityDynamicGroup",
		SpecType: reflect.TypeOf(identityv1beta1.DynamicGroupSpec{}),
		SDKStructs: []string{
			"identity.CreateDynamicGroupDetails",
			"identity.UpdateDynamicGroupDetails",
			"identity.DynamicGroup",
		},
	},
	{
		Name:     "IdentityFaultDomain",
		SpecType: reflect.TypeOf(identityv1beta1.FaultDomainSpec{}),
		SDKStructs: []string{
			"identity.FaultDomain",
		},
	},
	{
		Name:     "IdentityGroup",
		SpecType: reflect.TypeOf(identityv1beta1.GroupSpec{}),
		SDKStructs: []string{
			"identity.CreateGroupDetails",
			"identity.UpdateGroupDetails",
			"identity.Group",
		},
	},
	{
		Name:     "IdentityIamWorkRequest",
		SpecType: reflect.TypeOf(identityv1beta1.IamWorkRequestSpec{}),
		SDKStructs: []string{
			"identity.IamWorkRequest",
			"identity.IamWorkRequestSummary",
		},
	},
	{
		Name:     "IdentityIamWorkRequestError",
		SpecType: reflect.TypeOf(identityv1beta1.IamWorkRequestErrorSpec{}),
		SDKStructs: []string{
			"identity.IamWorkRequestErrorSummary",
		},
	},
	{
		Name:     "IdentityIamWorkRequestLog",
		SpecType: reflect.TypeOf(identityv1beta1.IamWorkRequestLogSpec{}),
		SDKStructs: []string{
			"identity.IamWorkRequestLogSummary",
		},
	},
	{
		Name:     "IdentityIdentityProviderGroup",
		SpecType: reflect.TypeOf(identityv1beta1.IdentityProviderGroupSpec{}),
		SDKStructs: []string{
			"identity.IdentityProviderGroupSummary",
		},
	},
	{
		Name:     "IdentityIdpGroupMapping",
		SpecType: reflect.TypeOf(identityv1beta1.IdpGroupMappingSpec{}),
		SDKStructs: []string{
			"identity.CreateIdpGroupMappingDetails",
			"identity.UpdateIdpGroupMappingDetails",
			"identity.IdpGroupMapping",
		},
	},
	{
		Name:     "IdentityMfaTotpDevice",
		SpecType: reflect.TypeOf(identityv1beta1.MfaTotpDeviceSpec{}),
		SDKStructs: []string{
			"identity.MfaTotpDevice",
			"identity.MfaTotpDeviceSummary",
		},
	},
	{
		Name:     "IdentityNetworkSource",
		SpecType: reflect.TypeOf(identityv1beta1.NetworkSourceSpec{}),
		SDKStructs: []string{
			"identity.CreateNetworkSourceDetails",
			"identity.UpdateNetworkSourceDetails",
		},
	},
	{
		Name:     "IdentityOAuthClientCredential",
		SpecType: reflect.TypeOf(identityv1beta1.OAuthClientCredentialSpec{}),
		SDKStructs: []string{
			"identity.CreateOAuth2ClientCredentialDetails",
			"identity.UpdateOAuth2ClientCredentialDetails",
			"identity.OAuth2ClientCredential",
			"identity.OAuth2ClientCredentialSummary",
		},
	},
	{
		Name:     "IdentityPolicy",
		SpecType: reflect.TypeOf(identityv1beta1.PolicySpec{}),
		SDKStructs: []string{
			"identity.CreatePolicyDetails",
			"identity.UpdatePolicyDetails",
			"identity.Policy",
		},
	},
	{
		Name:     "IdentityRegion",
		SpecType: reflect.TypeOf(identityv1beta1.RegionSpec{}),
		SDKStructs: []string{
			"identity.Region",
		},
	},
	{
		Name:     "IdentityRegionSubscription",
		SpecType: reflect.TypeOf(identityv1beta1.RegionSubscriptionSpec{}),
		SDKStructs: []string{
			"identity.CreateRegionSubscriptionDetails",
			"identity.RegionSubscription",
		},
	},
	{
		Name:     "IdentitySmtpCredential",
		SpecType: reflect.TypeOf(identityv1beta1.SmtpCredentialSpec{}),
		SDKStructs: []string{
			"identity.CreateSmtpCredentialDetails",
			"identity.UpdateSmtpCredentialDetails",
			"identity.SmtpCredential",
			"identity.SmtpCredentialSummary",
		},
	},
	{
		Name:     "IdentitySwiftPassword",
		SpecType: reflect.TypeOf(identityv1beta1.SwiftPasswordSpec{}),
		SDKStructs: []string{
			"identity.CreateSwiftPasswordDetails",
			"identity.UpdateSwiftPasswordDetails",
			"identity.SwiftPassword",
		},
	},
	{
		Name:     "IdentityTag",
		SpecType: reflect.TypeOf(identityv1beta1.TagSpec{}),
		SDKStructs: []string{
			"identity.CreateTagDetails",
			"identity.UpdateTagDetails",
			"identity.Tag",
			"identity.TagSummary",
		},
	},
	{
		Name:     "IdentityTagDefault",
		SpecType: reflect.TypeOf(identityv1beta1.TagDefaultSpec{}),
		SDKStructs: []string{
			"identity.CreateTagDefaultDetails",
			"identity.UpdateTagDefaultDetails",
			"identity.TagDefault",
			"identity.TagDefaultSummary",
		},
	},
	{
		Name:     "IdentityTagNamespace",
		SpecType: reflect.TypeOf(identityv1beta1.TagNamespaceSpec{}),
		SDKStructs: []string{
			"identity.CreateTagNamespaceDetails",
			"identity.UpdateTagNamespaceDetails",
			"identity.TagNamespace",
			"identity.TagNamespaceSummary",
		},
	},
	{
		Name:     "IdentityTaggingWorkRequest",
		SpecType: reflect.TypeOf(identityv1beta1.TaggingWorkRequestSpec{}),
		SDKStructs: []string{
			"identity.TaggingWorkRequest",
			"identity.TaggingWorkRequestSummary",
		},
	},
	{
		Name:     "IdentityTaggingWorkRequestError",
		SpecType: reflect.TypeOf(identityv1beta1.TaggingWorkRequestErrorSpec{}),
		SDKStructs: []string{
			"identity.TaggingWorkRequestErrorSummary",
		},
	},
	{
		Name:     "IdentityTaggingWorkRequestLog",
		SpecType: reflect.TypeOf(identityv1beta1.TaggingWorkRequestLogSpec{}),
		SDKStructs: []string{
			"identity.TaggingWorkRequestLogSummary",
		},
	},
	{
		Name:     "IdentityTenancy",
		SpecType: reflect.TypeOf(identityv1beta1.TenancySpec{}),
		SDKStructs: []string{
			"identity.Tenancy",
		},
	},
	{
		Name:     "IdentityUser",
		SpecType: reflect.TypeOf(identityv1beta1.UserSpec{}),
		SDKStructs: []string{
			"identity.CreateUserDetails",
			"identity.UpdateUserDetails",
			"identity.User",
		},
	},
	{
		Name:     "IdentityUserGroupMembership",
		SpecType: reflect.TypeOf(identityv1beta1.UserGroupMembershipSpec{}),
		SDKStructs: []string{
			"identity.UserGroupMembership",
		},
	},
	{
		Name:     "IdentityWorkRequest",
		SpecType: reflect.TypeOf(identityv1beta1.WorkRequestSpec{}),
		SDKStructs: []string{
			"identity.WorkRequest",
			"identity.WorkRequestSummary",
		},
	},
	{
		Name:     "KeyManagementEkmsPrivateEndpoint",
		SpecType: reflect.TypeOf(keymanagementv1beta1.EkmsPrivateEndpointSpec{}),
		SDKStructs: []string{
			"keymanagement.CreateEkmsPrivateEndpointDetails",
			"keymanagement.UpdateEkmsPrivateEndpointDetails",
			"keymanagement.EkmsPrivateEndpoint",
			"keymanagement.EkmsPrivateEndpointSummary",
		},
	},
	{
		Name:     "KeyManagementHsmCluster",
		SpecType: reflect.TypeOf(keymanagementv1beta1.HsmClusterSpec{}),
		SDKStructs: []string{
			"keymanagement.CreateHsmClusterDetails",
			"keymanagement.UpdateHsmClusterDetails",
			"keymanagement.HsmCluster",
			"keymanagement.HsmClusterSummary",
		},
	},
	{
		Name:     "KeyManagementHsmPartition",
		SpecType: reflect.TypeOf(keymanagementv1beta1.HsmPartitionSpec{}),
		SDKStructs: []string{
			"keymanagement.HsmPartition",
			"keymanagement.HsmPartitionSummary",
		},
	},
	{
		Name:     "KeyManagementKey",
		SpecType: reflect.TypeOf(keymanagementv1beta1.KeySpec{}),
		SDKStructs: []string{
			"keymanagement.CreateKeyDetails",
			"keymanagement.UpdateKeyDetails",
			"keymanagement.Key",
			"keymanagement.KeyVersionSummary",
			"keymanagement.KeySummary",
		},
	},
	{
		Name:     "KeyManagementKeyVersion",
		SpecType: reflect.TypeOf(keymanagementv1beta1.KeyVersionSpec{}),
		SDKStructs: []string{
			"keymanagement.KeyVersion",
			"keymanagement.KeyVersionSummary",
		},
	},
	{
		Name:     "KeyManagementVault",
		SpecType: reflect.TypeOf(keymanagementv1beta1.VaultSpec{}),
		SDKStructs: []string{
			"keymanagement.CreateVaultDetails",
			"keymanagement.UpdateVaultDetails",
			"keymanagement.Vault",
			"keymanagement.VaultSummary",
		},
	},
	{
		Name:     "KeyManagementVaultReplica",
		SpecType: reflect.TypeOf(keymanagementv1beta1.VaultReplicaSpec{}),
		SDKStructs: []string{
			"keymanagement.CreateVaultReplicaDetails",
			"keymanagement.VaultReplicaSummary",
		},
	},
	{
		Name:     "KeyManagementVaultUsage",
		SpecType: reflect.TypeOf(keymanagementv1beta1.VaultUsageSpec{}),
		SDKStructs: []string{
			"keymanagement.VaultUsage",
		},
	},
	{
		Name:     "KeyManagementWrappingKey",
		SpecType: reflect.TypeOf(keymanagementv1beta1.WrappingKeySpec{}),
		SDKStructs: []string{
			"keymanagement.WrappingKey",
		},
	},
	{
		Name:     "LimitsLimitDefinition",
		SpecType: reflect.TypeOf(limitsv1beta1.LimitDefinitionSpec{}),
		SDKStructs: []string{
			"limits.LimitDefinitionSummary",
		},
	},
	{
		Name:     "LimitsLimitValue",
		SpecType: reflect.TypeOf(limitsv1beta1.LimitValueSpec{}),
		SDKStructs: []string{
			"limits.LimitValueSummary",
		},
	},
	{
		Name:     "LimitsQuota",
		SpecType: reflect.TypeOf(limitsv1beta1.QuotaSpec{}),
		SDKStructs: []string{
			"limits.CreateQuotaDetails",
			"limits.UpdateQuotaDetails",
			"limits.Quota",
			"limits.QuotaSummary",
		},
	},
	{
		Name:     "LimitsResourceAvailability",
		SpecType: reflect.TypeOf(limitsv1beta1.ResourceAvailabilitySpec{}),
		SDKStructs: []string{
			"limits.ResourceAvailability",
		},
	},
	{
		Name:     "LimitsService",
		SpecType: reflect.TypeOf(limitsv1beta1.ServiceSpec{}),
		SDKStructs: []string{
			"limits.ServiceSummary",
		},
	},
	{
		Name:     "SecretsSecretBundle",
		SpecType: reflect.TypeOf(secretsv1beta1.SecretBundleSpec{}),
		SDKStructs: []string{
			"secrets.SecretBundle",
			"secrets.SecretBundleVersionSummary",
		},
	},
	{
		Name:     "SecretsSecretBundleByName",
		SpecType: reflect.TypeOf(secretsv1beta1.SecretBundleByNameSpec{}),
		SDKStructs: []string{
			"secrets.SecretBundle",
		},
	},
	{
		Name:     "SecretsSecretBundleVersion",
		SpecType: reflect.TypeOf(secretsv1beta1.SecretBundleVersionSpec{}),
		SDKStructs: []string{
			"secrets.SecretBundleVersionSummary",
		},
	},
	{
		Name:     "VaultSecret",
		SpecType: reflect.TypeOf(vaultv1beta1.SecretSpec{}),
		SDKStructs: []string{
			"vault.CreateSecretDetails",
			"vault.UpdateSecretDetails",
			"vault.Secret",
			"vault.SecretVersionSummary",
			"vault.SecretSummary",
		},
	},
	{
		Name:     "VaultSecretVersion",
		SpecType: reflect.TypeOf(vaultv1beta1.SecretVersionSpec{}),
		SDKStructs: []string{
			"vault.SecretVersion",
			"vault.SecretVersionSummary",
		},
	},
	{
		Name:     "CoreAppCatalogListing",
		SpecType: reflect.TypeOf(corev1beta1.AppCatalogListingSpec{}),
		SDKStructs: []string{
			"core.AppCatalogListing",
			"core.AppCatalogListingSummary",
		},
	},
	{
		Name:     "CoreAppCatalogListingResourceVersion",
		SpecType: reflect.TypeOf(corev1beta1.AppCatalogListingResourceVersionSpec{}),
		SDKStructs: []string{
			"core.AppCatalogListingResourceVersion",
			"core.AppCatalogListingResourceVersionSummary",
		},
	},
	{
		Name:     "CoreAppCatalogSubscription",
		SpecType: reflect.TypeOf(corev1beta1.AppCatalogSubscriptionSpec{}),
		SDKStructs: []string{
			"core.CreateAppCatalogSubscriptionDetails",
			"core.AppCatalogSubscription",
			"core.AppCatalogSubscriptionSummary",
		},
	},
	{
		Name:     "CoreBlockVolumeReplica",
		SpecType: reflect.TypeOf(corev1beta1.BlockVolumeReplicaSpec{}),
		SDKStructs: []string{
			"core.BlockVolumeReplica",
		},
	},
	{
		Name:     "CoreBootVolume",
		SpecType: reflect.TypeOf(corev1beta1.BootVolumeSpec{}),
		SDKStructs: []string{
			"core.CreateBootVolumeDetails",
			"core.UpdateBootVolumeDetails",
			"core.BootVolume",
		},
	},
	{
		Name:     "CoreBootVolumeAttachment",
		SpecType: reflect.TypeOf(corev1beta1.BootVolumeAttachmentSpec{}),
		SDKStructs: []string{
			"core.BootVolumeAttachment",
		},
	},
	{
		Name:     "CoreBootVolumeBackup",
		SpecType: reflect.TypeOf(corev1beta1.BootVolumeBackupSpec{}),
		SDKStructs: []string{
			"core.CreateBootVolumeBackupDetails",
			"core.UpdateBootVolumeBackupDetails",
			"core.BootVolumeBackup",
		},
	},
	{
		Name:     "CoreBootVolumeKMSKey",
		SpecType: reflect.TypeOf(corev1beta1.BootVolumeKmsKeySpec{}),
		SDKStructs: []string{
			"core.UpdateBootVolumeKmsKeyDetails",
			"core.BootVolumeKmsKey",
		},
	},
	{
		Name:     "CoreBootVolumeReplica",
		SpecType: reflect.TypeOf(corev1beta1.BootVolumeReplicaSpec{}),
		SDKStructs: []string{
			"core.BootVolumeReplica",
		},
	},
	{
		Name:     "CoreByoipAllocatedRange",
		SpecType: reflect.TypeOf(corev1beta1.ByoipAllocatedRangeSpec{}),
		SDKStructs: []string{
			"core.ByoipAllocatedRangeSummary",
		},
	},
	{
		Name:     "CoreByoipRange",
		SpecType: reflect.TypeOf(corev1beta1.ByoipRangeSpec{}),
		SDKStructs: []string{
			"core.CreateByoipRangeDetails",
			"core.UpdateByoipRangeDetails",
			"core.ByoipRange",
			"core.ByoipRangeSummary",
		},
	},
	{
		Name:     "CoreCPE",
		SpecType: reflect.TypeOf(corev1beta1.CpeSpec{}),
		SDKStructs: []string{
			"core.CreateCpeDetails",
			"core.UpdateCpeDetails",
			"core.Cpe",
		},
	},
	{
		Name:     "CoreCaptureFilter",
		SpecType: reflect.TypeOf(corev1beta1.CaptureFilterSpec{}),
		SDKStructs: []string{
			"core.CreateCaptureFilterDetails",
			"core.UpdateCaptureFilterDetails",
			"core.CaptureFilter",
		},
	},
	{
		Name:     "CoreClusterNetwork",
		SpecType: reflect.TypeOf(corev1beta1.ClusterNetworkSpec{}),
		SDKStructs: []string{
			"core.CreateClusterNetworkDetails",
			"core.UpdateClusterNetworkDetails",
			"core.ClusterNetwork",
			"core.ClusterNetworkSummary",
		},
	},
	{
		Name:     "CoreComputeCapacityReport",
		SpecType: reflect.TypeOf(corev1beta1.ComputeCapacityReportSpec{}),
		SDKStructs: []string{
			"core.CreateComputeCapacityReportDetails",
			"core.ComputeCapacityReport",
		},
	},
	{
		Name:     "CoreComputeCapacityReservation",
		SpecType: reflect.TypeOf(corev1beta1.ComputeCapacityReservationSpec{}),
		SDKStructs: []string{
			"core.CreateComputeCapacityReservationDetails",
			"core.UpdateComputeCapacityReservationDetails",
			"core.ComputeCapacityReservation",
			"core.ComputeCapacityReservationSummary",
		},
	},
	{
		Name:     "CoreComputeCapacityReservationInstanceShape",
		SpecType: reflect.TypeOf(corev1beta1.ComputeCapacityReservationInstanceShapeSpec{}),
		SDKStructs: []string{
			"core.ComputeCapacityReservationInstanceShapeSummary",
		},
	},
	{
		Name:     "CoreComputeCapacityTopology",
		SpecType: reflect.TypeOf(corev1beta1.ComputeCapacityTopologySpec{}),
		SDKStructs: []string{
			"core.CreateComputeCapacityTopologyDetails",
			"core.UpdateComputeCapacityTopologyDetails",
			"core.ComputeCapacityTopology",
			"core.ComputeCapacityTopologySummary",
		},
	},
	{
		Name:     "CoreComputeCluster",
		SpecType: reflect.TypeOf(corev1beta1.ComputeClusterSpec{}),
		SDKStructs: []string{
			"core.CreateComputeClusterDetails",
			"core.UpdateComputeClusterDetails",
			"core.ComputeCluster",
			"core.ComputeClusterSummary",
		},
	},
	{
		Name:     "CoreComputeGlobalImageCapabilitySchema",
		SpecType: reflect.TypeOf(corev1beta1.ComputeGlobalImageCapabilitySchemaSpec{}),
		SDKStructs: []string{
			"core.ComputeGlobalImageCapabilitySchema",
			"core.ComputeGlobalImageCapabilitySchemaVersionSummary",
			"core.ComputeGlobalImageCapabilitySchemaSummary",
		},
	},
	{
		Name:     "CoreComputeGlobalImageCapabilitySchemaVersion",
		SpecType: reflect.TypeOf(corev1beta1.ComputeGlobalImageCapabilitySchemaVersionSpec{}),
		SDKStructs: []string{
			"core.ComputeGlobalImageCapabilitySchemaVersion",
			"core.ComputeGlobalImageCapabilitySchemaVersionSummary",
		},
	},
	{
		Name:     "CoreComputeImageCapabilitySchema",
		SpecType: reflect.TypeOf(corev1beta1.ComputeImageCapabilitySchemaSpec{}),
		SDKStructs: []string{
			"core.CreateComputeImageCapabilitySchemaDetails",
			"core.UpdateComputeImageCapabilitySchemaDetails",
			"core.ComputeImageCapabilitySchema",
			"core.ComputeImageCapabilitySchemaSummary",
		},
	},
	{
		Name:     "CoreConsoleHistory",
		SpecType: reflect.TypeOf(corev1beta1.ConsoleHistorySpec{}),
		SDKStructs: []string{
			"core.UpdateConsoleHistoryDetails",
			"core.ConsoleHistory",
		},
	},
	{
		Name:     "CoreCpeDeviceShape",
		SpecType: reflect.TypeOf(corev1beta1.CpeDeviceShapeSpec{}),
		SDKStructs: []string{
			"core.CpeDeviceShapeSummary",
		},
	},
	{
		Name:     "CoreCrossConnect",
		SpecType: reflect.TypeOf(corev1beta1.CrossConnectSpec{}),
		SDKStructs: []string{
			"core.CreateCrossConnectDetails",
			"core.UpdateCrossConnectDetails",
			"core.CrossConnect",
		},
	},
	{
		Name:     "CoreCrossConnectGroup",
		SpecType: reflect.TypeOf(corev1beta1.CrossConnectGroupSpec{}),
		SDKStructs: []string{
			"core.CreateCrossConnectGroupDetails",
			"core.UpdateCrossConnectGroupDetails",
			"core.CrossConnectGroup",
		},
	},
	{
		Name:     "CoreCrossConnectLocation",
		SpecType: reflect.TypeOf(corev1beta1.CrossConnectLocationSpec{}),
		SDKStructs: []string{
			"core.CrossConnectLocation",
		},
	},
	{
		Name:     "CoreCrossConnectMapping",
		SpecType: reflect.TypeOf(corev1beta1.CrossConnectMappingSpec{}),
		SDKStructs: []string{
			"core.CrossConnectMapping",
		},
	},
	{
		Name:     "CoreCrossConnectStatus",
		SpecType: reflect.TypeOf(corev1beta1.CrossConnectStatusSpec{}),
		SDKStructs: []string{
			"core.CrossConnectStatus",
		},
	},
	{
		Name:     "CoreDRG",
		SpecType: reflect.TypeOf(corev1beta1.DrgSpec{}),
		SDKStructs: []string{
			"core.CreateDrgDetails",
			"core.UpdateDrgDetails",
			"core.Drg",
		},
	},
	{
		Name:     "CoreDRGAttachment",
		SpecType: reflect.TypeOf(corev1beta1.DrgAttachmentSpec{}),
		SDKStructs: []string{
			"core.CreateDrgAttachmentDetails",
			"core.UpdateDrgAttachmentDetails",
			"core.DrgAttachment",
		},
	},
	{
		Name:     "CoreDRGRouteDistribution",
		SpecType: reflect.TypeOf(corev1beta1.DrgRouteDistributionSpec{}),
		SDKStructs: []string{
			"core.CreateDrgRouteDistributionDetails",
			"core.UpdateDrgRouteDistributionDetails",
			"core.DrgRouteDistribution",
		},
	},
	{
		Name:     "CoreDRGRouteDistributionStatement",
		SpecType: reflect.TypeOf(corev1beta1.DrgRouteDistributionStatementSpec{}),
		SDKStructs: []string{
			"core.UpdateDrgRouteDistributionStatementDetails",
			"core.DrgRouteDistributionStatement",
		},
	},
	{
		Name:     "CoreDRGRouteRule",
		SpecType: reflect.TypeOf(corev1beta1.DrgRouteRuleSpec{}),
		SDKStructs: []string{
			"core.UpdateDrgRouteRuleDetails",
			"core.DrgRouteRule",
		},
	},
	{
		Name:     "CoreDRGRouteTable",
		SpecType: reflect.TypeOf(corev1beta1.DrgRouteTableSpec{}),
		SDKStructs: []string{
			"core.CreateDrgRouteTableDetails",
			"core.UpdateDrgRouteTableDetails",
			"core.DrgRouteTable",
		},
	},
	{
		Name:     "CoreDedicatedVmHost",
		SpecType: reflect.TypeOf(corev1beta1.DedicatedVmHostSpec{}),
		SDKStructs: []string{
			"core.CreateDedicatedVmHostDetails",
			"core.UpdateDedicatedVmHostDetails",
			"core.DedicatedVmHost",
			"core.DedicatedVmHostSummary",
		},
	},
	{
		Name:     "CoreDedicatedVmHostInstance",
		SpecType: reflect.TypeOf(corev1beta1.DedicatedVmHostInstanceSpec{}),
		SDKStructs: []string{
			"core.DedicatedVmHostInstanceSummary",
		},
	},
	{
		Name:     "CoreDedicatedVmHostInstanceShape",
		SpecType: reflect.TypeOf(corev1beta1.DedicatedVmHostInstanceShapeSpec{}),
		SDKStructs: []string{
			"core.DedicatedVmHostInstanceShapeSummary",
		},
	},
	{
		Name:     "CoreDedicatedVmHostShape",
		SpecType: reflect.TypeOf(corev1beta1.DedicatedVmHostShapeSpec{}),
		SDKStructs: []string{
			"core.DedicatedVmHostShapeSummary",
		},
	},
	{
		Name:     "CoreDhcpOption",
		SpecType: reflect.TypeOf(corev1beta1.DhcpOptionSpec{}),
		SDKStructs: []string{
			"core.CreateDhcpDetails",
			"core.UpdateDhcpDetails",
		},
	},
	{
		Name:     "CoreDrgRedundancyStatus",
		SpecType: reflect.TypeOf(corev1beta1.DrgRedundancyStatusSpec{}),
		SDKStructs: []string{
			"core.DrgRedundancyStatus",
		},
	},
	{
		Name:     "CoreFastConnectProviderService",
		SpecType: reflect.TypeOf(corev1beta1.FastConnectProviderServiceSpec{}),
		SDKStructs: []string{
			"core.FastConnectProviderService",
		},
	},
	{
		Name:     "CoreFastConnectProviderServiceKey",
		SpecType: reflect.TypeOf(corev1beta1.FastConnectProviderServiceKeySpec{}),
		SDKStructs: []string{
			"core.FastConnectProviderServiceKey",
		},
	},
	{
		Name:     "CoreIPSecConnection",
		SpecType: reflect.TypeOf(corev1beta1.IPSecConnectionSpec{}),
		SDKStructs: []string{
			"core.CreateIpSecConnectionDetails",
			"core.UpdateIpSecConnectionDetails",
			"core.IpSecConnection",
		},
	},
	{
		Name:     "CoreIPSecConnectionDeviceConfig",
		SpecType: reflect.TypeOf(corev1beta1.IPSecConnectionDeviceConfigSpec{}),
		SDKStructs: []string{
			"core.IpSecConnectionDeviceConfig",
		},
	},
	{
		Name:     "CoreIPSecConnectionDeviceStatus",
		SpecType: reflect.TypeOf(corev1beta1.IPSecConnectionDeviceStatusSpec{}),
		SDKStructs: []string{
			"core.IpSecConnectionDeviceStatus",
		},
	},
	{
		Name:     "CoreIPSecConnectionTunnel",
		SpecType: reflect.TypeOf(corev1beta1.IPSecConnectionTunnelSpec{}),
		SDKStructs: []string{
			"core.CreateIpSecConnectionTunnelDetails",
			"core.UpdateIpSecConnectionTunnelDetails",
			"core.IpSecConnectionTunnel",
		},
	},
	{
		Name:     "CoreIPSecConnectionTunnelSharedSecret",
		SpecType: reflect.TypeOf(corev1beta1.IPSecConnectionTunnelSharedSecretSpec{}),
		SDKStructs: []string{
			"core.UpdateIpSecConnectionTunnelSharedSecretDetails",
			"core.IpSecConnectionTunnelSharedSecret",
		},
	},
	{
		Name:     "CoreImage",
		SpecType: reflect.TypeOf(corev1beta1.ImageSpec{}),
		SDKStructs: []string{
			"core.CreateImageDetails",
			"core.UpdateImageDetails",
			"core.Image",
		},
	},
	{
		Name:     "CoreImageShapeCompatibilityEntry",
		SpecType: reflect.TypeOf(corev1beta1.ImageShapeCompatibilityEntrySpec{}),
		SDKStructs: []string{
			"core.ImageShapeCompatibilityEntry",
		},
	},
	{
		Name:     "CoreInstance",
		SpecType: reflect.TypeOf(corev1beta1.InstanceSpec{}),
		SDKStructs: []string{
			"core.UpdateInstanceDetails",
			"core.Instance",
			"core.InstanceSummary",
		},
	},
	{
		Name:     "CoreInstanceConfiguration",
		SpecType: reflect.TypeOf(corev1beta1.InstanceConfigurationSpec{}),
		SDKStructs: []string{
			"core.CreateInstanceConfigurationDetails",
			"core.UpdateInstanceConfigurationDetails",
			"core.InstanceConfiguration",
			"core.InstanceConfigurationSummary",
		},
	},
	{
		Name:     "CoreInstanceConsoleConnection",
		SpecType: reflect.TypeOf(corev1beta1.InstanceConsoleConnectionSpec{}),
		SDKStructs: []string{
			"core.CreateInstanceConsoleConnectionDetails",
			"core.UpdateInstanceConsoleConnectionDetails",
			"core.InstanceConsoleConnection",
		},
	},
	{
		Name:     "CoreInstanceMaintenanceReboot",
		SpecType: reflect.TypeOf(corev1beta1.InstanceMaintenanceRebootSpec{}),
		SDKStructs: []string{
			"core.InstanceMaintenanceReboot",
		},
	},
	{
		Name:     "CoreInstancePool",
		SpecType: reflect.TypeOf(corev1beta1.InstancePoolSpec{}),
		SDKStructs: []string{
			"core.CreateInstancePoolDetails",
			"core.UpdateInstancePoolDetails",
			"core.InstancePool",
			"core.InstancePoolSummary",
		},
	},
	{
		Name:     "CoreInstancePoolInstance",
		SpecType: reflect.TypeOf(corev1beta1.InstancePoolInstanceSpec{}),
		SDKStructs: []string{
			"core.InstancePoolInstance",
		},
	},
	{
		Name:     "CoreInstancePoolLoadBalancerAttachment",
		SpecType: reflect.TypeOf(corev1beta1.InstancePoolLoadBalancerAttachmentSpec{}),
		SDKStructs: []string{
			"core.InstancePoolLoadBalancerAttachment",
		},
	},
	{
		Name:     "CoreInternetGateway",
		SpecType: reflect.TypeOf(corev1beta1.InternetGatewaySpec{}),
		SDKStructs: []string{
			"core.CreateInternetGatewayDetails",
			"core.UpdateInternetGatewayDetails",
			"core.InternetGateway",
		},
	},
	{
		Name:     "CoreIpv6",
		SpecType: reflect.TypeOf(corev1beta1.Ipv6Spec{}),
		SDKStructs: []string{
			"core.CreateIpv6Details",
			"core.UpdateIpv6Details",
			"core.Ipv6",
		},
	},
	{
		Name:     "CoreLocalPeeringGateway",
		SpecType: reflect.TypeOf(corev1beta1.LocalPeeringGatewaySpec{}),
		SDKStructs: []string{
			"core.CreateLocalPeeringGatewayDetails",
			"core.UpdateLocalPeeringGatewayDetails",
			"core.LocalPeeringGateway",
		},
	},
	{
		Name:     "CoreMeasuredBootReport",
		SpecType: reflect.TypeOf(corev1beta1.MeasuredBootReportSpec{}),
		SDKStructs: []string{
			"core.MeasuredBootReport",
		},
	},
	{
		Name:     "CoreNATGateway",
		SpecType: reflect.TypeOf(corev1beta1.NatGatewaySpec{}),
		SDKStructs: []string{
			"core.CreateNatGatewayDetails",
			"core.UpdateNatGatewayDetails",
			"core.NatGateway",
		},
	},
	{
		Name:     "CoreNetworkSecurityGroup",
		SpecType: reflect.TypeOf(corev1beta1.NetworkSecurityGroupSpec{}),
		SDKStructs: []string{
			"core.CreateNetworkSecurityGroupDetails",
			"core.UpdateNetworkSecurityGroupDetails",
			"core.NetworkSecurityGroup",
		},
	},
	{
		Name:     "CoreNetworkSecurityGroupVnic",
		SpecType: reflect.TypeOf(corev1beta1.NetworkSecurityGroupVnicSpec{}),
		SDKStructs: []string{
			"core.NetworkSecurityGroupVnic",
		},
	},
	{
		Name:     "CoreNetworkingTopology",
		SpecType: reflect.TypeOf(corev1beta1.NetworkingTopologySpec{}),
		SDKStructs: []string{
			"core.NetworkingTopology",
		},
	},
	{
		Name:     "CorePrivateIP",
		SpecType: reflect.TypeOf(corev1beta1.PrivateIpSpec{}),
		SDKStructs: []string{
			"core.CreatePrivateIpDetails",
			"core.UpdatePrivateIpDetails",
			"core.PrivateIp",
		},
	},
	{
		Name:     "CorePublicIP",
		SpecType: reflect.TypeOf(corev1beta1.PublicIpSpec{}),
		SDKStructs: []string{
			"core.CreatePublicIpDetails",
			"core.UpdatePublicIpDetails",
			"core.PublicIp",
		},
	},
	{
		Name:     "CorePublicIPPool",
		SpecType: reflect.TypeOf(corev1beta1.PublicIpPoolSpec{}),
		SDKStructs: []string{
			"core.CreatePublicIpPoolDetails",
			"core.UpdatePublicIpPoolDetails",
			"core.PublicIpPool",
			"core.PublicIpPoolSummary",
		},
	},
	{
		Name:     "CoreRemotePeeringConnection",
		SpecType: reflect.TypeOf(corev1beta1.RemotePeeringConnectionSpec{}),
		SDKStructs: []string{
			"core.CreateRemotePeeringConnectionDetails",
			"core.UpdateRemotePeeringConnectionDetails",
			"core.RemotePeeringConnection",
		},
	},
	{
		Name:     "CoreRouteTable",
		SpecType: reflect.TypeOf(corev1beta1.RouteTableSpec{}),
		SDKStructs: []string{
			"core.CreateRouteTableDetails",
			"core.UpdateRouteTableDetails",
			"core.RouteTable",
		},
	},
	{
		Name:     "CoreSecurityList",
		SpecType: reflect.TypeOf(corev1beta1.SecurityListSpec{}),
		SDKStructs: []string{
			"core.CreateSecurityListDetails",
			"core.UpdateSecurityListDetails",
			"core.SecurityList",
		},
	},
	{
		Name:     "CoreService",
		SpecType: reflect.TypeOf(corev1beta1.ServiceSpec{}),
		SDKStructs: []string{
			"core.Service",
		},
	},
	{
		Name:     "CoreServiceGateway",
		SpecType: reflect.TypeOf(corev1beta1.ServiceGatewaySpec{}),
		SDKStructs: []string{
			"core.CreateServiceGatewayDetails",
			"core.UpdateServiceGatewayDetails",
			"core.ServiceGateway",
		},
	},
	{
		Name:     "CoreShape",
		SpecType: reflect.TypeOf(corev1beta1.ShapeSpec{}),
		SDKStructs: []string{
			"core.Shape",
		},
	},
	{
		Name:     "CoreSubnet",
		SpecType: reflect.TypeOf(corev1beta1.SubnetSpec{}),
		SDKStructs: []string{
			"core.CreateSubnetDetails",
			"core.UpdateSubnetDetails",
			"core.Subnet",
		},
	},
	{
		Name:     "CoreSubnetTopology",
		SpecType: reflect.TypeOf(corev1beta1.SubnetTopologySpec{}),
		SDKStructs: []string{
			"core.SubnetTopology",
		},
	},
	{
		Name:     "CoreTunnelCPEDeviceConfig",
		SpecType: reflect.TypeOf(corev1beta1.TunnelCpeDeviceConfigSpec{}),
		SDKStructs: []string{
			"core.UpdateTunnelCpeDeviceConfigDetails",
			"core.TunnelCpeDeviceConfig",
		},
	},
	{
		Name:     "CoreUpgradeStatus",
		SpecType: reflect.TypeOf(corev1beta1.UpgradeStatusSpec{}),
		SDKStructs: []string{
			"core.UpgradeStatus",
		},
	},
	{
		Name:     "CoreVCN",
		SpecType: reflect.TypeOf(corev1beta1.VcnSpec{}),
		SDKStructs: []string{
			"core.CreateVcnDetails",
			"core.UpdateVcnDetails",
			"core.Vcn",
		},
	},
	{
		Name:     "CoreVLAN",
		SpecType: reflect.TypeOf(corev1beta1.VlanSpec{}),
		SDKStructs: []string{
			"core.CreateVlanDetails",
			"core.UpdateVlanDetails",
			"core.Vlan",
		},
	},
	{
		Name:     "CoreVNIC",
		SpecType: reflect.TypeOf(corev1beta1.VnicSpec{}),
		SDKStructs: []string{
			"core.CreateVnicDetails",
			"core.UpdateVnicDetails",
			"core.Vnic",
		},
	},
	{
		Name:     "CoreVcnDnsResolverAssociation",
		SpecType: reflect.TypeOf(corev1beta1.VcnDnsResolverAssociationSpec{}),
		SDKStructs: []string{
			"core.VcnDnsResolverAssociation",
		},
	},
	{
		Name:     "CoreVcnTopology",
		SpecType: reflect.TypeOf(corev1beta1.VcnTopologySpec{}),
		SDKStructs: []string{
			"core.VcnTopology",
		},
	},
	{
		Name:     "CoreVirtualCircuit",
		SpecType: reflect.TypeOf(corev1beta1.VirtualCircuitSpec{}),
		SDKStructs: []string{
			"core.CreateVirtualCircuitDetails",
			"core.UpdateVirtualCircuitDetails",
			"core.VirtualCircuit",
		},
	},
	{
		Name:     "CoreVirtualCircuitBandwidthShape",
		SpecType: reflect.TypeOf(corev1beta1.VirtualCircuitBandwidthShapeSpec{}),
		SDKStructs: []string{
			"core.VirtualCircuitBandwidthShape",
		},
	},
	{
		Name:     "CoreVirtualCircuitPublicPrefix",
		SpecType: reflect.TypeOf(corev1beta1.VirtualCircuitPublicPrefixSpec{}),
		SDKStructs: []string{
			"core.CreateVirtualCircuitPublicPrefixDetails",
			"core.VirtualCircuitPublicPrefix",
		},
	},
	{
		Name:     "CoreVnicAttachment",
		SpecType: reflect.TypeOf(corev1beta1.VnicAttachmentSpec{}),
		SDKStructs: []string{
			"core.VnicAttachment",
		},
	},
	{
		Name:     "CoreVolume",
		SpecType: reflect.TypeOf(corev1beta1.VolumeSpec{}),
		SDKStructs: []string{
			"core.CreateVolumeDetails",
			"core.UpdateVolumeDetails",
			"core.Volume",
		},
	},
	{
		Name:     "CoreVolumeAttachment",
		SpecType: reflect.TypeOf(corev1beta1.VolumeAttachmentSpec{}),
		SDKStructs: []string{
			"core.UpdateVolumeAttachmentDetails",
		},
	},
	{
		Name:     "CoreVolumeBackup",
		SpecType: reflect.TypeOf(corev1beta1.VolumeBackupSpec{}),
		SDKStructs: []string{
			"core.CreateVolumeBackupDetails",
			"core.UpdateVolumeBackupDetails",
			"core.VolumeBackup",
		},
	},
	{
		Name:     "CoreVolumeBackupPolicy",
		SpecType: reflect.TypeOf(corev1beta1.VolumeBackupPolicySpec{}),
		SDKStructs: []string{
			"core.CreateVolumeBackupPolicyDetails",
			"core.UpdateVolumeBackupPolicyDetails",
			"core.VolumeBackupPolicy",
		},
	},
	{
		Name:     "CoreVolumeBackupPolicyAssignment",
		SpecType: reflect.TypeOf(corev1beta1.VolumeBackupPolicyAssignmentSpec{}),
		SDKStructs: []string{
			"core.CreateVolumeBackupPolicyAssignmentDetails",
			"core.VolumeBackupPolicyAssignment",
		},
	},
	{
		Name:     "CoreVolumeGroup",
		SpecType: reflect.TypeOf(corev1beta1.VolumeGroupSpec{}),
		SDKStructs: []string{
			"core.CreateVolumeGroupDetails",
			"core.UpdateVolumeGroupDetails",
			"core.VolumeGroup",
		},
	},
	{
		Name:     "CoreVolumeGroupBackup",
		SpecType: reflect.TypeOf(corev1beta1.VolumeGroupBackupSpec{}),
		SDKStructs: []string{
			"core.CreateVolumeGroupBackupDetails",
			"core.UpdateVolumeGroupBackupDetails",
			"core.VolumeGroupBackup",
		},
	},
	{
		Name:     "CoreVolumeGroupReplica",
		SpecType: reflect.TypeOf(corev1beta1.VolumeGroupReplicaSpec{}),
		SDKStructs: []string{
			"core.VolumeGroupReplica",
		},
	},
	{
		Name:     "CoreVolumeKMSKey",
		SpecType: reflect.TypeOf(corev1beta1.VolumeKmsKeySpec{}),
		SDKStructs: []string{
			"core.UpdateVolumeKmsKeyDetails",
			"core.VolumeKmsKey",
		},
	},
	{
		Name:     "CoreVtap",
		SpecType: reflect.TypeOf(corev1beta1.VtapSpec{}),
		SDKStructs: []string{
			"core.CreateVtapDetails",
			"core.UpdateVtapDetails",
			"core.Vtap",
		},
	},
	{
		Name:     "WorkrequestsWorkRequest",
		SpecType: reflect.TypeOf(workrequestsv1beta1.WorkRequestSpec{}),
		SDKStructs: []string{
			"workrequests.WorkRequest",
			"workrequests.WorkRequestSummary",
		},
	},
	{
		Name:     "WorkrequestsWorkRequestError",
		SpecType: reflect.TypeOf(workrequestsv1beta1.WorkRequestErrorSpec{}),
		SDKStructs: []string{
			"workrequests.WorkRequestError",
		},
	},
}

func Targets() []Target {
	result := make([]Target, len(targets))
	copy(result, targets)
	return result
}
