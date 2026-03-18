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
		Name:       "AutonomousDatabases",
		SpecType:   reflect.TypeOf(databasev1beta1.AutonomousDatabasesSpec{}),
		StatusType: reflect.TypeOf(databasev1beta1.AutonomousDatabasesStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "database.CreateAutonomousDatabaseDetails",
			},
			{
				SDKStruct: "database.UpdateAutonomousDatabaseDetails",
			},
		},
	},
	{
		Name:       "MySqlDbSystem",
		SpecType:   reflect.TypeOf(mysqlv1beta1.MySqlDbSystemSpec{}),
		StatusType: reflect.TypeOf(mysqlv1beta1.MySqlDbSystemStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "mysql.CreateDbSystemDetails",
			},
			{
				SDKStruct: "mysql.UpdateDbSystemDetails",
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
		Name:       "QueueMessage",
		SpecType:   reflect.TypeOf(queuev1beta1.MessageSpec{}),
		StatusType: reflect.TypeOf(queuev1beta1.MessageStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "queue.UpdateMessageDetails",
			},
		},
	},
	{
		Name:       "QueueWorkRequest",
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
		Name:       "QueueWorkRequestError",
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
		Name:       "ObjectStorageMultipartUpload",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.MultipartUploadSpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.MultipartUploadStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.CreateMultipartUploadDetails",
			},
			{
				SDKStruct: "objectstorage.MultipartUpload",
			},
		},
	},
	{
		Name:       "ObjectStorageMultipartUploadPart",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.MultipartUploadPartSpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.MultipartUploadPartStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.MultipartUploadPartSummary",
			},
		},
	},
	{
		Name:       "ObjectStorageNamespace",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.NamespaceSpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.NamespaceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.NamespaceMetadata",
				Exclude:   true,
				Reason:    "Intentionally untracked: Namespace returns the namespace string in the response body; namespace metadata parity is tracked on ObjectStorageNamespaceMetadata.",
			},
		},
	},
	{
		Name:       "ObjectStorageNamespaceMetadata",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.NamespaceMetadataSpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.NamespaceMetadataStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.UpdateNamespaceMetadataDetails",
			},
			{
				SDKStruct:  "objectstorage.NamespaceMetadata",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "ObjectStorageObject",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.ObjectSpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.ObjectStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.ObjectVersionSummary",
				Exclude:   true,
				Reason:    "Intentionally untracked: version summaries belong to the dedicated ObjectStorageObjectVersion status surface.",
			},
			{
				SDKStruct:  "objectstorage.ObjectSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "ObjectStorageObjectLifecyclePolicy",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.ObjectLifecyclePolicySpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.ObjectLifecyclePolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.ObjectLifecyclePolicy",
			},
		},
	},
	{
		Name:       "ObjectStorageObjectStorageTier",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.ObjectStorageTierSpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.ObjectStorageTierStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.UpdateObjectStorageTierDetails",
			},
		},
	},
	{
		Name:       "ObjectStorageObjectVersion",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.ObjectVersionSpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.ObjectVersionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.ObjectVersionCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "objectstorage.ObjectVersionSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "ObjectStoragePreauthenticatedRequest",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.PreauthenticatedRequestSpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.PreauthenticatedRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.CreatePreauthenticatedRequestDetails",
			},
			{
				SDKStruct:  "objectstorage.PreauthenticatedRequest",
				APISurface: "status",
			},
			{
				SDKStruct:  "objectstorage.PreauthenticatedRequestSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "ObjectStorageReplicationPolicy",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.ReplicationPolicySpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.ReplicationPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.CreateReplicationPolicyDetails",
			},
			{
				SDKStruct: "objectstorage.ReplicationPolicy",
			},
			{
				SDKStruct: "objectstorage.ReplicationPolicySummary",
			},
		},
	},
	{
		Name:       "ObjectStorageReplicationSource",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.ReplicationSourceSpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.ReplicationSourceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.ReplicationSource",
			},
		},
	},
	{
		Name:       "ObjectStorageRetentionRule",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.RetentionRuleSpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.RetentionRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.CreateRetentionRuleDetails",
			},
			{
				SDKStruct: "objectstorage.UpdateRetentionRuleDetails",
			},
			{
				SDKStruct:  "objectstorage.RetentionRuleDetails",
				APISurface: "status",
			},
			{
				SDKStruct:  "objectstorage.RetentionRule",
				APISurface: "status",
			},
			{
				SDKStruct: "objectstorage.RetentionRuleCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "objectstorage.RetentionRuleSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "ObjectStorageWorkRequest",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.WorkRequestSpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.WorkRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.WorkRequest",
			},
			{
				SDKStruct: "objectstorage.WorkRequestSummary",
			},
		},
	},
	{
		Name:       "ObjectStorageWorkRequestError",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.WorkRequestErrorSpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.WorkRequestErrorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "objectstorage.WorkRequestError",
			},
		},
	},
	{
		Name:       "ObjectStorageWorkRequestLog",
		SpecType:   reflect.TypeOf(objectstoragev1beta1.WorkRequestLogSpec{}),
		StatusType: reflect.TypeOf(objectstoragev1beta1.WorkRequestLogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "objectstorage.WorkRequestLogEntry",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "NotificationConfirmSubscription",
		SpecType:   reflect.TypeOf(onsv1beta1.ConfirmSubscriptionSpec{}),
		StatusType: reflect.TypeOf(onsv1beta1.ConfirmSubscriptionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "ons.ConfirmationResult",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "NotificationTopic",
		SpecType:   reflect.TypeOf(onsv1beta1.TopicSpec{}),
		StatusType: reflect.TypeOf(onsv1beta1.TopicStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "ons.CreateTopicDetails",
			},
			{
				SDKStruct: "ons.NotificationTopic",
				Exclude:   true,
				Reason:    "Intentionally untracked: OCI read-model mappings broaden desired-state coverage, and this CRD does not expose a meaningful status surface for parity tracking.",
			},
			{
				SDKStruct: "ons.NotificationTopicSummary",
				Exclude:   true,
				Reason:    "Intentionally untracked: OCI read-model mappings broaden desired-state coverage, and this CRD does not expose a meaningful status surface for parity tracking.",
			},
		},
	},
	{
		Name:        "NotificationUnsubscription",
		SpecType:    reflect.TypeOf(onsv1beta1.UnsubscriptionSpec{}),
		StatusType:  reflect.TypeOf(onsv1beta1.UnsubscriptionStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:       "ONSSubscription",
		SpecType:   reflect.TypeOf(onsv1beta1.SubscriptionSpec{}),
		StatusType: reflect.TypeOf(onsv1beta1.SubscriptionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "ons.CreateSubscriptionDetails",
			},
			{
				SDKStruct: "ons.UpdateSubscriptionDetails",
			},
			{
				SDKStruct:  "ons.Subscription",
				APISurface: "status",
			},
			{
				SDKStruct:  "ons.SubscriptionSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "LogGroup",
		SpecType:   reflect.TypeOf(loggingv1beta1.LogGroupSpec{}),
		StatusType: reflect.TypeOf(loggingv1beta1.LogGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "logging.CreateLogGroupDetails",
			},
			{
				SDKStruct: "logging.UpdateLogGroupDetails",
			},
			{
				SDKStruct:  "logging.LogGroup",
				APISurface: "status",
			},
			{
				SDKStruct:  "logging.LogGroupSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "LoggingLog",
		SpecType:   reflect.TypeOf(loggingv1beta1.LogSpec{}),
		StatusType: reflect.TypeOf(loggingv1beta1.LogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "logging.CreateLogDetails",
			},
			{
				SDKStruct: "logging.UpdateLogDetails",
			},
			{
				SDKStruct: "logging.Log",
			},
			{
				SDKStruct:  "logging.LogSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "LoggingLogSavedSearch",
		SpecType:   reflect.TypeOf(loggingv1beta1.LogSavedSearchSpec{}),
		StatusType: reflect.TypeOf(loggingv1beta1.LogSavedSearchStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "logging.CreateLogSavedSearchDetails",
			},
			{
				SDKStruct: "logging.UpdateLogSavedSearchDetails",
			},
			{
				SDKStruct:  "logging.LogSavedSearch",
				APISurface: "status",
			},
			{
				SDKStruct:  "logging.LogSavedSearchSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "LoggingService",
		SpecType:   reflect.TypeOf(loggingv1beta1.ServiceSpec{}),
		StatusType: reflect.TypeOf(loggingv1beta1.ServiceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "logging.ServiceSummary",
			},
		},
	},
	{
		Name:       "LoggingUnifiedAgentConfiguration",
		SpecType:   reflect.TypeOf(loggingv1beta1.UnifiedAgentConfigurationSpec{}),
		StatusType: reflect.TypeOf(loggingv1beta1.UnifiedAgentConfigurationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "logging.CreateUnifiedAgentConfigurationDetails",
			},
			{
				SDKStruct: "logging.UpdateUnifiedAgentConfigurationDetails",
			},
			{
				SDKStruct:  "logging.UnifiedAgentConfiguration",
				APISurface: "status",
			},
			{
				SDKStruct: "logging.UnifiedAgentConfigurationCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "logging.UnifiedAgentConfigurationSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "LoggingWorkRequest",
		SpecType:   reflect.TypeOf(loggingv1beta1.WorkRequestSpec{}),
		StatusType: reflect.TypeOf(loggingv1beta1.WorkRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "logging.WorkRequest",
			},
			{
				SDKStruct: "logging.WorkRequestSummary",
			},
		},
	},
	{
		Name:       "LoggingWorkRequestError",
		SpecType:   reflect.TypeOf(loggingv1beta1.WorkRequestErrorSpec{}),
		StatusType: reflect.TypeOf(loggingv1beta1.WorkRequestErrorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "logging.WorkRequestError",
			},
		},
	},
	{
		Name:       "LoggingWorkRequestLog",
		SpecType:   reflect.TypeOf(loggingv1beta1.WorkRequestLogSpec{}),
		StatusType: reflect.TypeOf(loggingv1beta1.WorkRequestLogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "logging.WorkRequestLog",
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
		Name:       "EventsRule",
		SpecType:   reflect.TypeOf(eventsv1beta1.RuleSpec{}),
		StatusType: reflect.TypeOf(eventsv1beta1.RuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "events.CreateRuleDetails",
			},
			{
				SDKStruct: "events.UpdateRuleDetails",
			},
			{
				SDKStruct:  "events.Rule",
				APISurface: "status",
			},
			{
				SDKStruct:  "events.RuleSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "MonitoringAlarm",
		SpecType:   reflect.TypeOf(monitoringv1beta1.AlarmSpec{}),
		StatusType: reflect.TypeOf(monitoringv1beta1.AlarmObservedState{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "monitoring.CreateAlarmDetails",
			},
			{
				SDKStruct: "monitoring.UpdateAlarmDetails",
			},
			{
				SDKStruct:  "monitoring.Alarm",
				APISurface: "status",
			},
			{
				SDKStruct:  "monitoring.AlarmSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "MonitoringAlarmHistory",
		SpecType:   reflect.TypeOf(monitoringv1beta1.AlarmHistorySpec{}),
		StatusType: reflect.TypeOf(monitoringv1beta1.AlarmHistoryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "monitoring.AlarmHistoryCollection",
				APISurface: "status",
			},
			{
				SDKStruct: "monitoring.AlarmHistoryEntry",
				Exclude:   true,
				Reason:    "Intentionally untracked: alarm history entries are represented as nested elements under AlarmHistory.status.entries, not a top-level reusable status surface.",
			},
		},
	},
	{
		Name:       "MonitoringAlarmStatus",
		SpecType:   reflect.TypeOf(monitoringv1beta1.AlarmStatusSpec{}),
		StatusType: reflect.TypeOf(monitoringv1beta1.AlarmStatusObservedState{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "monitoring.AlarmStatusSummary",
			},
		},
	},
	{
		Name:       "MonitoringAlarmSuppression",
		SpecType:   reflect.TypeOf(monitoringv1beta1.AlarmSuppressionSpec{}),
		StatusType: reflect.TypeOf(monitoringv1beta1.AlarmSuppressionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "monitoring.CreateAlarmSuppressionDetails",
			},
			{
				SDKStruct:  "monitoring.AlarmSuppression",
				APISurface: "status",
			},
			{
				SDKStruct: "monitoring.AlarmSuppressionCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "monitoring.AlarmSuppressionSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "MonitoringMetric",
		SpecType:   reflect.TypeOf(monitoringv1beta1.MetricSpec{}),
		StatusType: reflect.TypeOf(monitoringv1beta1.MetricStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "monitoring.Metric",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "DNSDomainRecord",
		SpecType:   reflect.TypeOf(dnsv1beta1.DomainRecordSpec{}),
		StatusType: reflect.TypeOf(dnsv1beta1.DomainRecordStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dns.Record",
			},
		},
	},
	{
		Name:       "DNSRRSet",
		SpecType:   reflect.TypeOf(dnsv1beta1.RRSetSpec{}),
		StatusType: reflect.TypeOf(dnsv1beta1.RRSetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dns.UpdateRrSetDetails",
			},
			{
				SDKStruct:  "dns.RrSet",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "DNSResolver",
		SpecType:   reflect.TypeOf(dnsv1beta1.ResolverSpec{}),
		StatusType: reflect.TypeOf(dnsv1beta1.ResolverStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dns.UpdateResolverDetails",
			},
			{
				SDKStruct: "dns.Resolver",
			},
			{
				SDKStruct: "dns.ResolverSummary",
			},
		},
	},
	{
		Name:       "DNSResolverEndpoint",
		SpecType:   reflect.TypeOf(dnsv1beta1.ResolverEndpointSpec{}),
		StatusType: reflect.TypeOf(dnsv1beta1.ResolverEndpointStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "dns.ResolverVnicEndpoint",
				APISurface: "status",
			},
			{
				SDKStruct:  "dns.ResolverVnicEndpointSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "DNSSteeringPolicy",
		SpecType:   reflect.TypeOf(dnsv1beta1.SteeringPolicySpec{}),
		StatusType: reflect.TypeOf(dnsv1beta1.SteeringPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dns.CreateSteeringPolicyDetails",
			},
			{
				SDKStruct: "dns.UpdateSteeringPolicyDetails",
			},
			{
				SDKStruct:  "dns.SteeringPolicy",
				APISurface: "status",
			},
			{
				SDKStruct:  "dns.SteeringPolicySummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "DNSSteeringPolicyAttachment",
		SpecType:   reflect.TypeOf(dnsv1beta1.SteeringPolicyAttachmentSpec{}),
		StatusType: reflect.TypeOf(dnsv1beta1.SteeringPolicyAttachmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dns.CreateSteeringPolicyAttachmentDetails",
			},
			{
				SDKStruct: "dns.UpdateSteeringPolicyAttachmentDetails",
			},
			{
				SDKStruct: "dns.SteeringPolicyAttachment",
			},
			{
				SDKStruct: "dns.SteeringPolicyAttachmentSummary",
			},
		},
	},
	{
		Name:       "DNSTsigKey",
		SpecType:   reflect.TypeOf(dnsv1beta1.TsigKeySpec{}),
		StatusType: reflect.TypeOf(dnsv1beta1.TsigKeyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dns.CreateTsigKeyDetails",
			},
			{
				SDKStruct: "dns.UpdateTsigKeyDetails",
			},
			{
				SDKStruct:  "dns.TsigKey",
				APISurface: "status",
			},
			{
				SDKStruct:  "dns.TsigKeySummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "DNSView",
		SpecType:   reflect.TypeOf(dnsv1beta1.ViewSpec{}),
		StatusType: reflect.TypeOf(dnsv1beta1.ViewStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dns.CreateViewDetails",
			},
			{
				SDKStruct: "dns.UpdateViewDetails",
			},
			{
				SDKStruct: "dns.View",
			},
			{
				SDKStruct: "dns.ViewSummary",
			},
		},
	},
	{
		Name:       "DNSZone",
		SpecType:   reflect.TypeOf(dnsv1beta1.ZoneSpec{}),
		StatusType: reflect.TypeOf(dnsv1beta1.ZoneStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dns.CreateZoneDetails",
			},
			{
				SDKStruct: "dns.UpdateZoneDetails",
			},
			{
				SDKStruct: "dns.Zone",
			},
			{
				SDKStruct: "dns.ZoneSummary",
			},
		},
	},
	{
		Name:        "DNSZoneContent",
		SpecType:    reflect.TypeOf(dnsv1beta1.ZoneContentSpec{}),
		StatusType:  reflect.TypeOf(dnsv1beta1.ZoneContentStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:       "DNSZoneFromZoneFile",
		SpecType:   reflect.TypeOf(dnsv1beta1.ZoneFromZoneFileSpec{}),
		StatusType: reflect.TypeOf(dnsv1beta1.ZoneFromZoneFileStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "dns.Zone",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "DNSZoneRecord",
		SpecType:   reflect.TypeOf(dnsv1beta1.ZoneRecordSpec{}),
		StatusType: reflect.TypeOf(dnsv1beta1.ZoneRecordStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dns.Record",
			},
		},
	},
	{
		Name:       "DNSZoneTransferServer",
		SpecType:   reflect.TypeOf(dnsv1beta1.ZoneTransferServerSpec{}),
		StatusType: reflect.TypeOf(dnsv1beta1.ZoneTransferServerStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dns.ZoneTransferServer",
			},
		},
	},
	{
		Name:       "LoadBalancer",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.LoadBalancerSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.LoadBalancerStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.CreateLoadBalancerDetails",
			},
			{
				SDKStruct: "loadbalancer.UpdateLoadBalancerDetails",
			},
			{
				SDKStruct:  "loadbalancer.LoadBalancer",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "LoadBalancerBackend",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.BackendSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.BackendStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.CreateBackendDetails",
			},
			{
				SDKStruct: "loadbalancer.UpdateBackendDetails",
			},
			{
				SDKStruct: "loadbalancer.BackendDetails",
			},
			{
				SDKStruct:  "loadbalancer.Backend",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "LoadBalancerBackendHealth",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.BackendHealthSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.BackendHealthStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.BackendHealth",
			},
		},
	},
	{
		Name:       "LoadBalancerBackendSet",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.BackendSetSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.BackendSetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.CreateBackendSetDetails",
			},
			{
				SDKStruct: "loadbalancer.UpdateBackendSetDetails",
			},
			{
				SDKStruct: "loadbalancer.BackendSetDetails",
			},
			{
				SDKStruct:  "loadbalancer.BackendSet",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "LoadBalancerBackendSetHealth",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.BackendSetHealthSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.BackendSetHealthStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.BackendSetHealth",
			},
		},
	},
	{
		Name:       "LoadBalancerCertificate",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.CertificateSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.CertificateStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.CreateCertificateDetails",
			},
			{
				SDKStruct: "loadbalancer.CertificateDetails",
			},
			{
				SDKStruct:  "loadbalancer.Certificate",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "LoadBalancerHealthChecker",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.HealthCheckerSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.HealthCheckerStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.UpdateHealthCheckerDetails",
			},
			{
				SDKStruct: "loadbalancer.HealthCheckerDetails",
			},
			{
				SDKStruct:  "loadbalancer.HealthChecker",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "LoadBalancerHostname",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.HostnameSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.HostnameStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.CreateHostnameDetails",
			},
			{
				SDKStruct: "loadbalancer.UpdateHostnameDetails",
			},
			{
				SDKStruct: "loadbalancer.HostnameDetails",
			},
			{
				SDKStruct:  "loadbalancer.Hostname",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "LoadBalancerListener",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.ListenerSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.ListenerStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.CreateListenerDetails",
			},
			{
				SDKStruct: "loadbalancer.UpdateListenerDetails",
			},
			{
				SDKStruct: "loadbalancer.ListenerDetails",
			},
			{
				SDKStruct:  "loadbalancer.Listener",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "LoadBalancerListenerRule",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.ListenerRuleSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.ListenerRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.ListenerRuleSummary",
			},
		},
	},
	{
		Name:       "LoadBalancerLoadBalancerHealth",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.LoadBalancerHealthSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.LoadBalancerHealthStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.LoadBalancerHealth",
			},
			{
				SDKStruct: "loadbalancer.LoadBalancerHealthSummary",
			},
		},
	},
	{
		Name:       "LoadBalancerLoadBalancerShape",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.LoadBalancerShapeSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.LoadBalancerShapeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.UpdateLoadBalancerShapeDetails",
			},
			{
				SDKStruct: "loadbalancer.LoadBalancerShape",
			},
		},
	},
	{
		Name:       "LoadBalancerNetworkSecurityGroup",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.NetworkSecurityGroupSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.NetworkSecurityGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.UpdateNetworkSecurityGroupsDetails",
			},
		},
	},
	{
		Name:       "LoadBalancerPathRouteSet",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.PathRouteSetSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.PathRouteSetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.CreatePathRouteSetDetails",
			},
			{
				SDKStruct: "loadbalancer.UpdatePathRouteSetDetails",
			},
			{
				SDKStruct: "loadbalancer.PathRouteSetDetails",
			},
			{
				SDKStruct:  "loadbalancer.PathRouteSet",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "LoadBalancerPolicy",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.PolicySpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.PolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.LoadBalancerPolicy",
				Exclude:   true,
				Reason:    "Intentionally untracked: policy catalog entries are read-only reference data and this CRD does not expose a meaningful singular status surface.",
			},
		},
	},
	{
		Name:       "LoadBalancerProtocol",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.ProtocolSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.ProtocolStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.LoadBalancerProtocol",
				Exclude:   true,
				Reason:    "Intentionally untracked: protocol catalog entries are read-only reference data and this CRD does not expose a meaningful singular status surface.",
			},
		},
	},
	{
		Name:       "LoadBalancerRoutingPolicy",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.RoutingPolicySpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.RoutingPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.CreateRoutingPolicyDetails",
			},
			{
				SDKStruct: "loadbalancer.UpdateRoutingPolicyDetails",
			},
			{
				SDKStruct: "loadbalancer.RoutingPolicyDetails",
			},
			{
				SDKStruct:  "loadbalancer.RoutingPolicy",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "LoadBalancerRuleSet",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.RuleSetSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.RuleSetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.CreateRuleSetDetails",
			},
			{
				SDKStruct: "loadbalancer.UpdateRuleSetDetails",
			},
			{
				SDKStruct: "loadbalancer.RuleSetDetails",
			},
			{
				SDKStruct:  "loadbalancer.RuleSet",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "LoadBalancerSSLCipherSuite",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.SSLCipherSuiteSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.SSLCipherSuiteStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.CreateSslCipherSuiteDetails",
			},
			{
				SDKStruct: "loadbalancer.UpdateSslCipherSuiteDetails",
			},
			{
				SDKStruct: "loadbalancer.SslCipherSuiteDetails",
			},
			{
				SDKStruct:  "loadbalancer.SslCipherSuite",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "LoadBalancerShape",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.ShapeSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.ShapeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.UpdateLoadBalancerShapeDetails",
				Exclude:   true,
				Reason:    "Intentionally untracked: duplicate desired-state payload is already tracked on LoadBalancerLoadBalancerShape.",
			},
			{
				SDKStruct: "loadbalancer.ShapeDetails",
				Exclude:   true,
				Reason:    "Intentionally untracked: shape catalog entries are read-only reference data; load balancer shape mutation parity is tracked on LoadBalancerLoadBalancerShape.",
			},
			{
				SDKStruct: "loadbalancer.LoadBalancerShape",
				Exclude:   true,
				Reason:    "Intentionally untracked: shape catalog entries are read-only reference data; load balancer shape mutation parity is tracked on LoadBalancerLoadBalancerShape.",
			},
		},
	},
	{
		Name:       "LoadBalancerWorkRequest",
		SpecType:   reflect.TypeOf(loadbalancerv1beta1.WorkRequestSpec{}),
		StatusType: reflect.TypeOf(loadbalancerv1beta1.WorkRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loadbalancer.WorkRequest",
			},
		},
	},
	{
		Name:       "NetworkLoadBalancer",
		SpecType:   reflect.TypeOf(networkloadbalancerv1beta1.NetworkLoadBalancerSpec{}),
		StatusType: reflect.TypeOf(networkloadbalancerv1beta1.NetworkLoadBalancerStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkloadbalancer.CreateNetworkLoadBalancerDetails",
			},
			{
				SDKStruct: "networkloadbalancer.UpdateNetworkLoadBalancerDetails",
			},
			{
				SDKStruct:  "networkloadbalancer.NetworkLoadBalancer",
				APISurface: "status",
			},
			{
				SDKStruct: "networkloadbalancer.NetworkLoadBalancerCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "networkloadbalancer.NetworkLoadBalancerSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "NetworkLoadBalancerBackend",
		SpecType:   reflect.TypeOf(networkloadbalancerv1beta1.BackendSpec{}),
		StatusType: reflect.TypeOf(networkloadbalancerv1beta1.BackendStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkloadbalancer.CreateBackendDetails",
			},
			{
				SDKStruct: "networkloadbalancer.UpdateBackendDetails",
			},
			{
				SDKStruct: "networkloadbalancer.BackendDetails",
			},
			{
				SDKStruct:  "networkloadbalancer.Backend",
				APISurface: "spec",
			},
			{
				SDKStruct: "networkloadbalancer.BackendCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "networkloadbalancer.BackendSummary",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "NetworkLoadBalancerBackendHealth",
		SpecType:   reflect.TypeOf(networkloadbalancerv1beta1.BackendHealthSpec{}),
		StatusType: reflect.TypeOf(networkloadbalancerv1beta1.BackendHealthStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkloadbalancer.BackendHealth",
			},
		},
	},
	{
		Name:       "NetworkLoadBalancerBackendSet",
		SpecType:   reflect.TypeOf(networkloadbalancerv1beta1.BackendSetSpec{}),
		StatusType: reflect.TypeOf(networkloadbalancerv1beta1.BackendSetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkloadbalancer.CreateBackendSetDetails",
			},
			{
				SDKStruct: "networkloadbalancer.UpdateBackendSetDetails",
			},
			{
				SDKStruct: "networkloadbalancer.BackendSetDetails",
			},
			{
				SDKStruct:  "networkloadbalancer.BackendSet",
				APISurface: "spec",
			},
			{
				SDKStruct: "networkloadbalancer.BackendSetCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "networkloadbalancer.BackendSetSummary",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "NetworkLoadBalancerBackendSetHealth",
		SpecType:   reflect.TypeOf(networkloadbalancerv1beta1.BackendSetHealthSpec{}),
		StatusType: reflect.TypeOf(networkloadbalancerv1beta1.BackendSetHealthStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkloadbalancer.BackendSetHealth",
			},
		},
	},
	{
		Name:       "NetworkLoadBalancerHealthChecker",
		SpecType:   reflect.TypeOf(networkloadbalancerv1beta1.HealthCheckerSpec{}),
		StatusType: reflect.TypeOf(networkloadbalancerv1beta1.HealthCheckerStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkloadbalancer.UpdateHealthCheckerDetails",
			},
			{
				SDKStruct: "networkloadbalancer.HealthCheckerDetails",
			},
			{
				SDKStruct:  "networkloadbalancer.HealthChecker",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "NetworkLoadBalancerListener",
		SpecType:   reflect.TypeOf(networkloadbalancerv1beta1.ListenerSpec{}),
		StatusType: reflect.TypeOf(networkloadbalancerv1beta1.ListenerStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkloadbalancer.CreateListenerDetails",
			},
			{
				SDKStruct: "networkloadbalancer.UpdateListenerDetails",
			},
			{
				SDKStruct: "networkloadbalancer.ListenerDetails",
			},
			{
				SDKStruct:  "networkloadbalancer.Listener",
				APISurface: "spec",
			},
			{
				SDKStruct: "networkloadbalancer.ListenerCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "networkloadbalancer.ListenerSummary",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "NetworkLoadBalancerNetworkLoadBalancerHealth",
		SpecType:   reflect.TypeOf(networkloadbalancerv1beta1.NetworkLoadBalancerHealthSpec{}),
		StatusType: reflect.TypeOf(networkloadbalancerv1beta1.NetworkLoadBalancerHealthStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "networkloadbalancer.NetworkLoadBalancerHealth",
				APISurface: "status",
			},
			{
				SDKStruct: "networkloadbalancer.NetworkLoadBalancerHealthCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "networkloadbalancer.NetworkLoadBalancerHealthSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "NetworkLoadBalancerNetworkLoadBalancersPolicy",
		SpecType:   reflect.TypeOf(networkloadbalancerv1beta1.NetworkLoadBalancersPolicySpec{}),
		StatusType: reflect.TypeOf(networkloadbalancerv1beta1.NetworkLoadBalancersPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkloadbalancer.NetworkLoadBalancersPolicyCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "NetworkLoadBalancerNetworkLoadBalancersProtocol",
		SpecType:   reflect.TypeOf(networkloadbalancerv1beta1.NetworkLoadBalancersProtocolSpec{}),
		StatusType: reflect.TypeOf(networkloadbalancerv1beta1.NetworkLoadBalancersProtocolStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkloadbalancer.NetworkLoadBalancersProtocolCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "NetworkLoadBalancerNetworkSecurityGroup",
		SpecType:   reflect.TypeOf(networkloadbalancerv1beta1.NetworkSecurityGroupSpec{}),
		StatusType: reflect.TypeOf(networkloadbalancerv1beta1.NetworkSecurityGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkloadbalancer.UpdateNetworkSecurityGroupsDetails",
			},
		},
	},
	{
		Name:       "NetworkLoadBalancerWorkRequest",
		SpecType:   reflect.TypeOf(networkloadbalancerv1beta1.WorkRequestSpec{}),
		StatusType: reflect.TypeOf(networkloadbalancerv1beta1.WorkRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "networkloadbalancer.WorkRequest",
				APISurface: "status",
			},
			{
				SDKStruct: "networkloadbalancer.WorkRequestCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "networkloadbalancer.WorkRequestSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "NetworkLoadBalancerWorkRequestError",
		SpecType:   reflect.TypeOf(networkloadbalancerv1beta1.WorkRequestErrorSpec{}),
		StatusType: reflect.TypeOf(networkloadbalancerv1beta1.WorkRequestErrorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "networkloadbalancer.WorkRequestError",
				APISurface: "status",
			},
			{
				SDKStruct: "networkloadbalancer.WorkRequestErrorCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "NetworkLoadBalancerWorkRequestLog",
		SpecType:   reflect.TypeOf(networkloadbalancerv1beta1.WorkRequestLogSpec{}),
		StatusType: reflect.TypeOf(networkloadbalancerv1beta1.WorkRequestLogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "networkloadbalancer.WorkRequestLogEntry",
				APISurface: "status",
			},
			{
				SDKStruct: "networkloadbalancer.WorkRequestLogEntryCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "ArtifactsContainerConfiguration",
		SpecType:   reflect.TypeOf(artifactsv1beta1.ContainerConfigurationSpec{}),
		StatusType: reflect.TypeOf(artifactsv1beta1.ContainerConfigurationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "artifacts.UpdateContainerConfigurationDetails",
			},
			{
				SDKStruct: "artifacts.ContainerConfiguration",
			},
		},
	},
	{
		Name:       "ArtifactsContainerImage",
		SpecType:   reflect.TypeOf(artifactsv1beta1.ContainerImageSpec{}),
		StatusType: reflect.TypeOf(artifactsv1beta1.ContainerImageStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "artifacts.UpdateContainerImageDetails",
			},
			{
				SDKStruct: "artifacts.ContainerImage",
			},
			{
				SDKStruct: "artifacts.ContainerImageCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct: "artifacts.ContainerImageSummary",
			},
		},
	},
	{
		Name:       "ArtifactsContainerImageSignature",
		SpecType:   reflect.TypeOf(artifactsv1beta1.ContainerImageSignatureSpec{}),
		StatusType: reflect.TypeOf(artifactsv1beta1.ContainerImageSignatureStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "artifacts.CreateContainerImageSignatureDetails",
			},
			{
				SDKStruct: "artifacts.UpdateContainerImageSignatureDetails",
			},
			{
				SDKStruct:  "artifacts.ContainerImageSignature",
				APISurface: "status",
			},
			{
				SDKStruct: "artifacts.ContainerImageSignatureCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "artifacts.ContainerImageSignatureSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "ArtifactsContainerRepository",
		SpecType:   reflect.TypeOf(artifactsv1beta1.ContainerRepositorySpec{}),
		StatusType: reflect.TypeOf(artifactsv1beta1.ContainerRepositoryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "artifacts.CreateContainerRepositoryDetails",
			},
			{
				SDKStruct: "artifacts.UpdateContainerRepositoryDetails",
			},
			{
				SDKStruct: "artifacts.ContainerRepository",
			},
			{
				SDKStruct: "artifacts.ContainerRepositoryCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct: "artifacts.ContainerRepositorySummary",
			},
		},
	},
	{
		Name:       "ArtifactsGenericArtifact",
		SpecType:   reflect.TypeOf(artifactsv1beta1.GenericArtifactSpec{}),
		StatusType: reflect.TypeOf(artifactsv1beta1.GenericArtifactStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "artifacts.UpdateGenericArtifactDetails",
			},
			{
				SDKStruct: "artifacts.GenericArtifact",
			},
			{
				SDKStruct: "artifacts.GenericArtifactCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct: "artifacts.GenericArtifactSummary",
			},
		},
	},
	{
		Name:       "ArtifactsGenericArtifactByPath",
		SpecType:   reflect.TypeOf(artifactsv1beta1.GenericArtifactByPathSpec{}),
		StatusType: reflect.TypeOf(artifactsv1beta1.GenericArtifactByPathStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "artifacts.UpdateGenericArtifactByPathDetails",
			},
		},
	},
	{
		Name:       "ArtifactsRepository",
		SpecType:   reflect.TypeOf(artifactsv1beta1.RepositorySpec{}),
		StatusType: reflect.TypeOf(artifactsv1beta1.RepositoryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "artifacts.ContainerRepository",
				APISurface: "status",
				Exclude:    true,
				Reason:     "Intentionally untracked: ArtifactsRepository status represents generic repositories; container repository parity is tracked on ArtifactsContainerRepository.",
			},
			{
				SDKStruct:  "artifacts.GenericRepository",
				APISurface: "status",
			},
			{
				SDKStruct: "artifacts.RepositoryCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "CertificatesCaBundle",
		SpecType:   reflect.TypeOf(certificatesv1beta1.CaBundleSpec{}),
		StatusType: reflect.TypeOf(certificatesv1beta1.CaBundleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "certificates.CaBundle",
			},
		},
	},
	{
		Name:       "CertificatesCertificateAuthorityBundle",
		SpecType:   reflect.TypeOf(certificatesv1beta1.CertificateAuthorityBundleSpec{}),
		StatusType: reflect.TypeOf(certificatesv1beta1.CertificateAuthorityBundleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "certificates.CertificateAuthorityBundle",
			},
			{
				SDKStruct: "certificates.CertificateAuthorityBundleVersionSummary",
			},
		},
	},
	{
		Name:       "CertificatesCertificateAuthorityBundleVersion",
		SpecType:   reflect.TypeOf(certificatesv1beta1.CertificateAuthorityBundleVersionSpec{}),
		StatusType: reflect.TypeOf(certificatesv1beta1.CertificateAuthorityBundleVersionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "certificates.CertificateAuthorityBundleVersionCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct: "certificates.CertificateAuthorityBundleVersionSummary",
			},
		},
	},
	{
		Name:       "CertificatesCertificateBundle",
		SpecType:   reflect.TypeOf(certificatesv1beta1.CertificateBundleSpec{}),
		StatusType: reflect.TypeOf(certificatesv1beta1.CertificateBundleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "certificates.CertificateBundlePublicOnly",
			},
			{
				SDKStruct: "certificates.CertificateBundleVersionSummary",
			},
		},
	},
	{
		Name:       "CertificatesCertificateBundleVersion",
		SpecType:   reflect.TypeOf(certificatesv1beta1.CertificateBundleVersionSpec{}),
		StatusType: reflect.TypeOf(certificatesv1beta1.CertificateBundleVersionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "certificates.CertificateBundleVersionCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct: "certificates.CertificateBundleVersionSummary",
			},
		},
	},
	{
		Name:       "CertificatesManagementAssociation",
		SpecType:   reflect.TypeOf(certificatesmanagementv1beta1.AssociationSpec{}),
		StatusType: reflect.TypeOf(certificatesmanagementv1beta1.AssociationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "certificatesmanagement.Association",
				APISurface: "status",
			},
			{
				SDKStruct: "certificatesmanagement.AssociationCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "certificatesmanagement.AssociationSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CertificatesManagementCaBundle",
		SpecType:   reflect.TypeOf(certificatesmanagementv1beta1.CaBundleSpec{}),
		StatusType: reflect.TypeOf(certificatesmanagementv1beta1.CaBundleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "certificatesmanagement.CreateCaBundleDetails",
			},
			{
				SDKStruct: "certificatesmanagement.UpdateCaBundleDetails",
			},
			{
				SDKStruct:  "certificatesmanagement.CaBundle",
				APISurface: "status",
			},
			{
				SDKStruct: "certificatesmanagement.CaBundleCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "certificatesmanagement.CaBundleSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CertificatesManagementCertificate",
		SpecType:   reflect.TypeOf(certificatesmanagementv1beta1.CertificateSpec{}),
		StatusType: reflect.TypeOf(certificatesmanagementv1beta1.CertificateStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "certificatesmanagement.CreateCertificateDetails",
			},
			{
				SDKStruct: "certificatesmanagement.UpdateCertificateDetails",
			},
			{
				SDKStruct:  "certificatesmanagement.Certificate",
				APISurface: "status",
			},
			{
				SDKStruct: "certificatesmanagement.CertificateCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct: "certificatesmanagement.CertificateVersionSummary",
				Exclude:   true,
				Reason:    "Intentionally untracked: version summaries belong to the dedicated CertificatesManagementCertificateVersion status surface.",
			},
			{
				SDKStruct:  "certificatesmanagement.CertificateSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CertificatesManagementCertificateAuthority",
		SpecType:   reflect.TypeOf(certificatesmanagementv1beta1.CertificateAuthoritySpec{}),
		StatusType: reflect.TypeOf(certificatesmanagementv1beta1.CertificateAuthorityStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "certificatesmanagement.CreateCertificateAuthorityDetails",
			},
			{
				SDKStruct: "certificatesmanagement.UpdateCertificateAuthorityDetails",
			},
			{
				SDKStruct:  "certificatesmanagement.CertificateAuthority",
				APISurface: "status",
			},
			{
				SDKStruct: "certificatesmanagement.CertificateAuthorityCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct: "certificatesmanagement.CertificateAuthorityVersionSummary",
				Exclude:   true,
				Reason:    "Intentionally untracked: version summaries belong to the dedicated CertificatesManagementCertificateAuthorityVersion status surface.",
			},
			{
				SDKStruct:  "certificatesmanagement.CertificateAuthoritySummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CertificatesManagementCertificateAuthorityVersion",
		SpecType:   reflect.TypeOf(certificatesmanagementv1beta1.CertificateAuthorityVersionSpec{}),
		StatusType: reflect.TypeOf(certificatesmanagementv1beta1.CertificateAuthorityVersionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "certificatesmanagement.CertificateAuthorityVersion",
				APISurface: "status",
			},
			{
				SDKStruct: "certificatesmanagement.CertificateAuthorityVersionCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "certificatesmanagement.CertificateAuthorityVersionSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "CertificatesManagementCertificateVersion",
		SpecType:   reflect.TypeOf(certificatesmanagementv1beta1.CertificateVersionSpec{}),
		StatusType: reflect.TypeOf(certificatesmanagementv1beta1.CertificateVersionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "certificatesmanagement.CertificateVersion",
				APISurface: "status",
			},
			{
				SDKStruct: "certificatesmanagement.CertificateVersionCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "certificatesmanagement.CertificateVersionSummary",
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
	{
		Name:       "IdentityAllowedDomainLicenseType",
		SpecType:   reflect.TypeOf(identityv1beta1.AllowedDomainLicenseTypeSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.AllowedDomainLicenseTypeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.AllowedDomainLicenseTypeSummary",
			},
		},
	},
	{
		Name:       "IdentityApiKey",
		SpecType:   reflect.TypeOf(identityv1beta1.ApiKeySpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.ApiKeyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateApiKeyDetails",
			},
			{
				SDKStruct: "identity.ApiKey",
			},
		},
	},
	{
		Name:       "IdentityAuthToken",
		SpecType:   reflect.TypeOf(identityv1beta1.AuthTokenSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.AuthTokenStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateAuthTokenDetails",
			},
			{
				SDKStruct: "identity.UpdateAuthTokenDetails",
			},
			{
				SDKStruct: "identity.AuthToken",
			},
		},
	},
	{
		Name:       "IdentityAuthenticationPolicy",
		SpecType:   reflect.TypeOf(identityv1beta1.AuthenticationPolicySpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.AuthenticationPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.UpdateAuthenticationPolicyDetails",
			},
			{
				SDKStruct:  "identity.AuthenticationPolicy",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "IdentityAvailabilityDomain",
		SpecType:   reflect.TypeOf(identityv1beta1.AvailabilityDomainSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.AvailabilityDomainStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.AvailabilityDomain",
			},
		},
	},
	{
		Name:       "IdentityBulkActionResourceType",
		SpecType:   reflect.TypeOf(identityv1beta1.BulkActionResourceTypeSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.BulkActionResourceTypeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.BulkActionResourceType",
			},
			{
				SDKStruct: "identity.BulkActionResourceTypeCollection",
			},
		},
	},
	{
		Name:       "IdentityBulkEditTagsResourceType",
		SpecType:   reflect.TypeOf(identityv1beta1.BulkEditTagsResourceTypeSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.BulkEditTagsResourceTypeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.BulkEditTagsResourceType",
			},
			{
				SDKStruct: "identity.BulkEditTagsResourceTypeCollection",
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
		Name:       "IdentityCostTrackingTag",
		SpecType:   reflect.TypeOf(identityv1beta1.CostTrackingTagSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.CostTrackingTagStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "identity.Tag",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "IdentityCustomerSecretKey",
		SpecType:   reflect.TypeOf(identityv1beta1.CustomerSecretKeySpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.CustomerSecretKeyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateCustomerSecretKeyDetails",
			},
			{
				SDKStruct: "identity.UpdateCustomerSecretKeyDetails",
			},
			{
				SDKStruct: "identity.CustomerSecretKey",
			},
			{
				SDKStruct: "identity.CustomerSecretKeySummary",
			},
		},
	},
	{
		Name:       "IdentityDbCredential",
		SpecType:   reflect.TypeOf(identityv1beta1.DbCredentialSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.DbCredentialStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateDbCredentialDetails",
			},
			{
				SDKStruct: "identity.DbCredential",
			},
			{
				SDKStruct: "identity.DbCredentialSummary",
			},
		},
	},
	{
		Name:       "IdentityDomain",
		SpecType:   reflect.TypeOf(identityv1beta1.DomainSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.DomainStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateDomainDetails",
			},
			{
				SDKStruct: "identity.UpdateDomainDetails",
			},
			{
				SDKStruct: "identity.Domain",
			},
			{
				SDKStruct: "identity.DomainSummary",
			},
		},
	},
	{
		Name:       "IdentityDynamicGroup",
		SpecType:   reflect.TypeOf(identityv1beta1.DynamicGroupSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.DynamicGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateDynamicGroupDetails",
			},
			{
				SDKStruct: "identity.UpdateDynamicGroupDetails",
			},
			{
				SDKStruct:  "identity.DynamicGroup",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "IdentityFaultDomain",
		SpecType:   reflect.TypeOf(identityv1beta1.FaultDomainSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.FaultDomainStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.FaultDomain",
			},
		},
	},
	{
		Name:       "IdentityGroup",
		SpecType:   reflect.TypeOf(identityv1beta1.GroupSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.GroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateGroupDetails",
			},
			{
				SDKStruct: "identity.UpdateGroupDetails",
			},
			{
				SDKStruct:  "identity.Group",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "IdentityIamWorkRequest",
		SpecType:   reflect.TypeOf(identityv1beta1.IamWorkRequestSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.IamWorkRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.IamWorkRequest",
			},
			{
				SDKStruct: "identity.IamWorkRequestSummary",
			},
		},
	},
	{
		Name:       "IdentityIamWorkRequestError",
		SpecType:   reflect.TypeOf(identityv1beta1.IamWorkRequestErrorSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.IamWorkRequestErrorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.IamWorkRequestErrorSummary",
			},
		},
	},
	{
		Name:       "IdentityIamWorkRequestLog",
		SpecType:   reflect.TypeOf(identityv1beta1.IamWorkRequestLogSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.IamWorkRequestLogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.IamWorkRequestLogSummary",
			},
		},
	},
	{
		Name:       "IdentityIdentityProvider",
		SpecType:   reflect.TypeOf(identityv1beta1.IdentityProviderSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.IdentityProviderStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "identity.Saml2IdentityProvider",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "IdentityIdentityProviderGroup",
		SpecType:   reflect.TypeOf(identityv1beta1.IdentityProviderGroupSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.IdentityProviderGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.IdentityProviderGroupSummary",
			},
		},
	},
	{
		Name:       "IdentityIdpGroupMapping",
		SpecType:   reflect.TypeOf(identityv1beta1.IdpGroupMappingSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.IdpGroupMappingStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateIdpGroupMappingDetails",
			},
			{
				SDKStruct: "identity.UpdateIdpGroupMappingDetails",
			},
			{
				SDKStruct: "identity.IdpGroupMapping",
			},
		},
	},
	{
		Name:       "IdentityMfaTotpDevice",
		SpecType:   reflect.TypeOf(identityv1beta1.MfaTotpDeviceSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.MfaTotpDeviceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.MfaTotpDevice",
			},
			{
				SDKStruct: "identity.MfaTotpDeviceSummary",
			},
		},
	},
	{
		Name:       "IdentityNetworkSource",
		SpecType:   reflect.TypeOf(identityv1beta1.NetworkSourceSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.NetworkSourceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateNetworkSourceDetails",
			},
			{
				SDKStruct: "identity.UpdateNetworkSourceDetails",
			},
			{
				SDKStruct:  "identity.NetworkSources",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "IdentityOAuthClientCredential",
		SpecType:   reflect.TypeOf(identityv1beta1.OAuthClientCredentialSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.OAuthClientCredentialStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateOAuth2ClientCredentialDetails",
			},
			{
				SDKStruct: "identity.UpdateOAuth2ClientCredentialDetails",
			},
			{
				SDKStruct: "identity.OAuth2ClientCredential",
				Exclude:   true,
				Reason:    "Intentionally untracked: OCI read-model mappings broaden desired-state coverage, and this CRD does not expose a meaningful status surface for parity tracking.",
			},
			{
				SDKStruct: "identity.OAuth2ClientCredentialSummary",
				Exclude:   true,
				Reason:    "Intentionally untracked: OCI read-model mappings broaden desired-state coverage, and this CRD does not expose a meaningful status surface for parity tracking.",
			},
		},
	},
	{
		Name:       "IdentityOrResetUIPassword",
		SpecType:   reflect.TypeOf(identityv1beta1.OrResetUIPasswordSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.OrResetUIPasswordStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "identity.UiPassword",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "IdentityPolicy",
		SpecType:   reflect.TypeOf(identityv1beta1.PolicySpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.PolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreatePolicyDetails",
			},
			{
				SDKStruct: "identity.UpdatePolicyDetails",
			},
			{
				SDKStruct:  "identity.Policy",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "IdentityRegion",
		SpecType:   reflect.TypeOf(identityv1beta1.RegionSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.RegionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.Region",
			},
		},
	},
	{
		Name:       "IdentityRegionSubscription",
		SpecType:   reflect.TypeOf(identityv1beta1.RegionSubscriptionSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.RegionSubscriptionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateRegionSubscriptionDetails",
			},
			{
				SDKStruct: "identity.RegionSubscription",
			},
		},
	},
	{
		Name:       "IdentitySmtpCredential",
		SpecType:   reflect.TypeOf(identityv1beta1.SmtpCredentialSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.SmtpCredentialStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateSmtpCredentialDetails",
			},
			{
				SDKStruct: "identity.UpdateSmtpCredentialDetails",
			},
			{
				SDKStruct: "identity.SmtpCredential",
			},
			{
				SDKStruct: "identity.SmtpCredentialSummary",
			},
		},
	},
	{
		Name:       "IdentityStandardTagNamespace",
		SpecType:   reflect.TypeOf(identityv1beta1.StandardTagNamespaceSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.StandardTagNamespaceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.StandardTagNamespaceTemplate",
			},
			{
				SDKStruct: "identity.StandardTagNamespaceTemplateSummary",
			},
		},
	},
	{
		Name:       "IdentityStandardTagTemplate",
		SpecType:   reflect.TypeOf(identityv1beta1.StandardTagTemplateSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.StandardTagTemplateStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.StandardTagDefinitionTemplate",
			},
		},
	},
	{
		Name:       "IdentitySwiftPassword",
		SpecType:   reflect.TypeOf(identityv1beta1.SwiftPasswordSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.SwiftPasswordStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateSwiftPasswordDetails",
			},
			{
				SDKStruct: "identity.UpdateSwiftPasswordDetails",
			},
			{
				SDKStruct: "identity.SwiftPassword",
			},
		},
	},
	{
		Name:       "IdentityTag",
		SpecType:   reflect.TypeOf(identityv1beta1.TagSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.TagStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateTagDetails",
			},
			{
				SDKStruct: "identity.UpdateTagDetails",
			},
			{
				SDKStruct:  "identity.Tag",
				APISurface: "status",
			},
			{
				SDKStruct:  "identity.TagSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "IdentityTagDefault",
		SpecType:   reflect.TypeOf(identityv1beta1.TagDefaultSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.TagDefaultStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateTagDefaultDetails",
			},
			{
				SDKStruct: "identity.UpdateTagDefaultDetails",
			},
			{
				SDKStruct: "identity.TagDefault",
			},
			{
				SDKStruct: "identity.TagDefaultSummary",
			},
		},
	},
	{
		Name:       "IdentityTagNamespace",
		SpecType:   reflect.TypeOf(identityv1beta1.TagNamespaceSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.TagNamespaceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateTagNamespaceDetails",
			},
			{
				SDKStruct: "identity.UpdateTagNamespaceDetails",
			},
			{
				SDKStruct:  "identity.TagNamespace",
				APISurface: "status",
			},
			{
				SDKStruct:  "identity.TagNamespaceSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "IdentityTaggingWorkRequest",
		SpecType:   reflect.TypeOf(identityv1beta1.TaggingWorkRequestSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.TaggingWorkRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.TaggingWorkRequest",
			},
			{
				SDKStruct: "identity.TaggingWorkRequestSummary",
			},
		},
	},
	{
		Name:       "IdentityTaggingWorkRequestError",
		SpecType:   reflect.TypeOf(identityv1beta1.TaggingWorkRequestErrorSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.TaggingWorkRequestErrorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.TaggingWorkRequestErrorSummary",
			},
		},
	},
	{
		Name:       "IdentityTaggingWorkRequestLog",
		SpecType:   reflect.TypeOf(identityv1beta1.TaggingWorkRequestLogSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.TaggingWorkRequestLogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.TaggingWorkRequestLogSummary",
			},
		},
	},
	{
		Name:       "IdentityTenancy",
		SpecType:   reflect.TypeOf(identityv1beta1.TenancySpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.TenancyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.Tenancy",
			},
		},
	},
	{
		Name:       "IdentityUser",
		SpecType:   reflect.TypeOf(identityv1beta1.UserSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.UserStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.CreateUserDetails",
			},
			{
				SDKStruct: "identity.UpdateUserDetails",
			},
			{
				SDKStruct: "identity.User",
			},
		},
	},
	{
		Name:       "IdentityUserCapability",
		SpecType:   reflect.TypeOf(identityv1beta1.UserCapabilitySpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.UserCapabilityStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "identity.UserCapabilities",
				APISurface: "spec",
			},
		},
	},
	{
		Name:       "IdentityUserGroupMembership",
		SpecType:   reflect.TypeOf(identityv1beta1.UserGroupMembershipSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.UserGroupMembershipStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.UserGroupMembership",
			},
		},
	},
	{
		Name:       "IdentityUserState",
		SpecType:   reflect.TypeOf(identityv1beta1.UserStateSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.UserStateStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "identity.User",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "IdentityUserUIPasswordInformation",
		SpecType:   reflect.TypeOf(identityv1beta1.UserUIPasswordInformationSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.UserUIPasswordInformationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.UiPasswordInformation",
			},
		},
	},
	{
		Name:       "IdentityWorkRequest",
		SpecType:   reflect.TypeOf(identityv1beta1.WorkRequestSpec{}),
		StatusType: reflect.TypeOf(identityv1beta1.WorkRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "identity.WorkRequest",
			},
			{
				SDKStruct: "identity.WorkRequestSummary",
			},
		},
	},
	{
		Name:       "KeyManagementEkmsPrivateEndpoint",
		SpecType:   reflect.TypeOf(keymanagementv1beta1.EkmsPrivateEndpointSpec{}),
		StatusType: reflect.TypeOf(keymanagementv1beta1.EkmsPrivateEndpointStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "keymanagement.CreateEkmsPrivateEndpointDetails",
			},
			{
				SDKStruct: "keymanagement.UpdateEkmsPrivateEndpointDetails",
			},
			{
				SDKStruct:  "keymanagement.EkmsPrivateEndpoint",
				APISurface: "status",
			},
			{
				SDKStruct:  "keymanagement.EkmsPrivateEndpointSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "KeyManagementHsmCluster",
		SpecType:   reflect.TypeOf(keymanagementv1beta1.HsmClusterSpec{}),
		StatusType: reflect.TypeOf(keymanagementv1beta1.HsmClusterStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "keymanagement.CreateHsmClusterDetails",
			},
			{
				SDKStruct: "keymanagement.UpdateHsmClusterDetails",
			},
			{
				SDKStruct:  "keymanagement.HsmCluster",
				APISurface: "status",
			},
			{
				SDKStruct: "keymanagement.HsmClusterCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "keymanagement.HsmClusterSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "KeyManagementHsmPartition",
		SpecType:   reflect.TypeOf(keymanagementv1beta1.HsmPartitionSpec{}),
		StatusType: reflect.TypeOf(keymanagementv1beta1.HsmPartitionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "keymanagement.HsmPartition",
				APISurface: "status",
			},
			{
				SDKStruct: "keymanagement.HsmPartitionCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "keymanagement.HsmPartitionSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "KeyManagementKey",
		SpecType:   reflect.TypeOf(keymanagementv1beta1.KeySpec{}),
		StatusType: reflect.TypeOf(keymanagementv1beta1.KeyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "keymanagement.CreateKeyDetails",
			},
			{
				SDKStruct: "keymanagement.UpdateKeyDetails",
			},
			{
				SDKStruct:  "keymanagement.Key",
				APISurface: "status",
			},
			{
				SDKStruct: "keymanagement.KeyVersionSummary",
				Exclude:   true,
				Reason:    "Intentionally untracked: key version summaries belong to the dedicated KeyManagementKeyVersion status surface.",
			},
			{
				SDKStruct:  "keymanagement.KeySummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "KeyManagementKeyVersion",
		SpecType:   reflect.TypeOf(keymanagementv1beta1.KeyVersionSpec{}),
		StatusType: reflect.TypeOf(keymanagementv1beta1.KeyVersionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "keymanagement.KeyVersion",
			},
			{
				SDKStruct: "keymanagement.KeyVersionSummary",
			},
		},
	},
	{
		Name:       "KeyManagementPreCoUserCredential",
		SpecType:   reflect.TypeOf(keymanagementv1beta1.PreCoUserCredentialSpec{}),
		StatusType: reflect.TypeOf(keymanagementv1beta1.PreCoUserCredentialStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "keymanagement.PreCoUserCredentials",
			},
		},
	},
	{
		Name:       "KeyManagementReplicationStatus",
		SpecType:   reflect.TypeOf(keymanagementv1beta1.ReplicationStatusSpec{}),
		StatusType: reflect.TypeOf(keymanagementv1beta1.ReplicationStatusObservedState{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "keymanagement.ReplicationStatusDetails",
				Exclude:   true,
				Reason:    "Intentionally untracked: OCI read-model mappings broaden desired-state coverage, and this CRD does not expose a meaningful status surface for parity tracking.",
			},
		},
	},
	{
		Name:       "KeyManagementVault",
		SpecType:   reflect.TypeOf(keymanagementv1beta1.VaultSpec{}),
		StatusType: reflect.TypeOf(keymanagementv1beta1.VaultStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "keymanagement.CreateVaultDetails",
			},
			{
				SDKStruct: "keymanagement.UpdateVaultDetails",
			},
			{
				SDKStruct: "keymanagement.Vault",
			},
			{
				SDKStruct: "keymanagement.VaultSummary",
			},
		},
	},
	{
		Name:       "KeyManagementVaultReplica",
		SpecType:   reflect.TypeOf(keymanagementv1beta1.VaultReplicaSpec{}),
		StatusType: reflect.TypeOf(keymanagementv1beta1.VaultReplicaStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "keymanagement.CreateVaultReplicaDetails",
			},
			{
				SDKStruct: "keymanagement.VaultReplicaDetails",
				Exclude:   true,
				Reason:    "Intentionally untracked: replica detail payload is nested under KeyManagementVault status via replicaDetails.",
			},
			{
				SDKStruct:  "keymanagement.VaultReplicaSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "KeyManagementVaultUsage",
		SpecType:   reflect.TypeOf(keymanagementv1beta1.VaultUsageSpec{}),
		StatusType: reflect.TypeOf(keymanagementv1beta1.VaultUsageStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "keymanagement.VaultUsage",
			},
		},
	},
	{
		Name:       "KeyManagementWrappingKey",
		SpecType:   reflect.TypeOf(keymanagementv1beta1.WrappingKeySpec{}),
		StatusType: reflect.TypeOf(keymanagementv1beta1.WrappingKeyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "keymanagement.WrappingKey",
			},
		},
	},
	{
		Name:       "LimitsLimitDefinition",
		SpecType:   reflect.TypeOf(limitsv1beta1.LimitDefinitionSpec{}),
		StatusType: reflect.TypeOf(limitsv1beta1.LimitDefinitionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "limits.LimitDefinitionSummary",
			},
		},
	},
	{
		Name:       "LimitsLimitValue",
		SpecType:   reflect.TypeOf(limitsv1beta1.LimitValueSpec{}),
		StatusType: reflect.TypeOf(limitsv1beta1.LimitValueStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "limits.LimitValueSummary",
			},
		},
	},
	{
		Name:       "LimitsQuota",
		SpecType:   reflect.TypeOf(limitsv1beta1.QuotaSpec{}),
		StatusType: reflect.TypeOf(limitsv1beta1.QuotaStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "limits.CreateQuotaDetails",
			},
			{
				SDKStruct: "limits.UpdateQuotaDetails",
			},
			{
				SDKStruct:  "limits.Quota",
				APISurface: "status",
			},
			{
				SDKStruct:  "limits.QuotaSummary",
				APISurface: "status",
			},
		},
	},
	{
		Name:       "LimitsResourceAvailability",
		SpecType:   reflect.TypeOf(limitsv1beta1.ResourceAvailabilitySpec{}),
		StatusType: reflect.TypeOf(limitsv1beta1.ResourceAvailabilityStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "limits.ResourceAvailability",
			},
		},
	},
	{
		Name:       "LimitsService",
		SpecType:   reflect.TypeOf(limitsv1beta1.ServiceSpec{}),
		StatusType: reflect.TypeOf(limitsv1beta1.ServiceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "limits.ServiceSummary",
			},
		},
	},
	{
		Name:       "SecretsSecretBundle",
		SpecType:   reflect.TypeOf(secretsv1beta1.SecretBundleSpec{}),
		StatusType: reflect.TypeOf(secretsv1beta1.SecretBundleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "secrets.SecretBundle",
			},
			{
				SDKStruct: "secrets.SecretBundleVersionSummary",
			},
		},
	},
	{
		Name:       "SecretsSecretBundleByName",
		SpecType:   reflect.TypeOf(secretsv1beta1.SecretBundleByNameSpec{}),
		StatusType: reflect.TypeOf(secretsv1beta1.SecretBundleByNameStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "secrets.SecretBundle",
			},
			{
				SDKStruct: "secrets.SecretBundleVersionSummary",
			},
		},
	},
	{
		Name:       "SecretsSecretBundleVersion",
		SpecType:   reflect.TypeOf(secretsv1beta1.SecretBundleVersionSpec{}),
		StatusType: reflect.TypeOf(secretsv1beta1.SecretBundleVersionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "secrets.SecretBundleVersionSummary",
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
		Name:       "CoreBootVolumeKMSKey",
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
		Name:       "CoreCPE",
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
				SDKStruct:  "core.ComputeBareMetalHostCollection",
				APISurface: "status",
				Exclude:    true,
				Reason:     "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "CoreComputeCapacityTopologyComputeHpcIsland",
		SpecType:   reflect.TypeOf(corev1beta1.ComputeCapacityTopologyComputeHpcIslandSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ComputeCapacityTopologyComputeHpcIslandStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.ComputeHpcIslandCollection",
				APISurface: "status",
				Exclude:    true,
				Reason:     "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
		},
	},
	{
		Name:       "CoreComputeCapacityTopologyComputeNetworkBlock",
		SpecType:   reflect.TypeOf(corev1beta1.ComputeCapacityTopologyComputeNetworkBlockSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.ComputeCapacityTopologyComputeNetworkBlockStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.ComputeNetworkBlockCollection",
				APISurface: "status",
				Exclude:    true,
				Reason:     "Intentionally untracked: collection responses do not map to a singular resource status surface.",
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
		Name:       "CoreDRG",
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
		Name:       "CoreDRGAttachment",
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
		Name:       "CoreDRGRouteDistribution",
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
		Name:       "CoreDRGRouteDistributionStatement",
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
		Name:       "CoreDRGRouteRule",
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
		Name:       "CoreDRGRouteTable",
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
				SDKStruct: "core.CreateDhcpDetails",
			},
			{
				SDKStruct: "core.UpdateDhcpDetails",
			},
			{
				SDKStruct:  "core.DhcpOptions",
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
		Name:       "CoreNATGateway",
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
		Name:       "CorePrivateIP",
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
		Name:       "CorePublicIP",
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
		Name:       "CorePublicIPPool",
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
		Name:       "CoreTunnelCPEDeviceConfig",
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
		Name:       "CoreVCN",
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
		Name:       "CoreVLAN",
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
		Name:       "CoreVNIC",
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
		Name:       "CoreVolumeKMSKey",
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
		Name:       "WorkrequestsWorkRequest",
		SpecType:   reflect.TypeOf(workrequestsv1beta1.WorkRequestSpec{}),
		StatusType: reflect.TypeOf(workrequestsv1beta1.WorkRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "workrequests.WorkRequest",
			},
			{
				SDKStruct: "workrequests.WorkRequestSummary",
			},
		},
	},
	{
		Name:       "WorkrequestsWorkRequestError",
		SpecType:   reflect.TypeOf(workrequestsv1beta1.WorkRequestErrorSpec{}),
		StatusType: reflect.TypeOf(workrequestsv1beta1.WorkRequestErrorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "workrequests.WorkRequestError",
			},
		},
	},
	{
		Name:       "WorkrequestsWorkRequestLog",
		SpecType:   reflect.TypeOf(workrequestsv1beta1.WorkRequestLogSpec{}),
		StatusType: reflect.TypeOf(workrequestsv1beta1.WorkRequestLogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "workrequests.WorkRequestLogEntry",
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
