package apispec

import (
	"reflect"

	admv1beta1 "github.com/oracle/oci-service-operator/api/adm/v1beta1"
	aidocumentv1beta1 "github.com/oracle/oci-service-operator/api/aidocument/v1beta1"
	ailanguagev1beta1 "github.com/oracle/oci-service-operator/api/ailanguage/v1beta1"
	aispeechv1beta1 "github.com/oracle/oci-service-operator/api/aispeech/v1beta1"
	aivisionv1beta1 "github.com/oracle/oci-service-operator/api/aivision/v1beta1"
	analyticsv1beta1 "github.com/oracle/oci-service-operator/api/analytics/v1beta1"
	bdsv1beta1 "github.com/oracle/oci-service-operator/api/bds/v1beta1"
	budgetv1beta1 "github.com/oracle/oci-service-operator/api/budget/v1beta1"
	clusterplacementgroupsv1beta1 "github.com/oracle/oci-service-operator/api/clusterplacementgroups/v1beta1"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	containerinstancesv1beta1 "github.com/oracle/oci-service-operator/api/containerinstances/v1beta1"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	databasetoolsv1beta1 "github.com/oracle/oci-service-operator/api/databasetools/v1beta1"
	dataflowv1beta1 "github.com/oracle/oci-service-operator/api/dataflow/v1beta1"
	datasciencev1beta1 "github.com/oracle/oci-service-operator/api/datascience/v1beta1"
	emailv1beta1 "github.com/oracle/oci-service-operator/api/email/v1beta1"
	functionsv1beta1 "github.com/oracle/oci-service-operator/api/functions/v1beta1"
	generativeaiv1beta1 "github.com/oracle/oci-service-operator/api/generativeai/v1beta1"
	identityv1beta1 "github.com/oracle/oci-service-operator/api/identity/v1beta1"
	keymanagementv1beta1 "github.com/oracle/oci-service-operator/api/keymanagement/v1beta1"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	loggingv1beta1 "github.com/oracle/oci-service-operator/api/logging/v1beta1"
	marketplacev1beta1 "github.com/oracle/oci-service-operator/api/marketplace/v1beta1"
	monitoringv1beta1 "github.com/oracle/oci-service-operator/api/monitoring/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	nosqlv1beta1 "github.com/oracle/oci-service-operator/api/nosql/v1beta1"
	objectstoragev1beta1 "github.com/oracle/oci-service-operator/api/objectstorage/v1beta1"
	ocvpv1beta1 "github.com/oracle/oci-service-operator/api/ocvp/v1beta1"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	opensearchv1beta1 "github.com/oracle/oci-service-operator/api/opensearch/v1beta1"
	psqlv1beta1 "github.com/oracle/oci-service-operator/api/psql/v1beta1"
	queuev1beta1 "github.com/oracle/oci-service-operator/api/queue/v1beta1"
	redisv1beta1 "github.com/oracle/oci-service-operator/api/redis/v1beta1"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	usageapiv1beta1 "github.com/oracle/oci-service-operator/api/usageapi/v1beta1"
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
		Name:       "EmailDkim",
		SpecType:   reflect.TypeOf(emailv1beta1.DkimSpec{}),
		StatusType: reflect.TypeOf(emailv1beta1.DkimStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "email.CreateDkimDetails",
			},
			{
				SDKStruct: "email.UpdateDkimDetails",
			},
			{
				SDKStruct: "email.Dkim",
			},
			{
				SDKStruct: "email.DkimCollection",
			},
			{
				SDKStruct: "email.DkimSummary",
			},
		},
	},
	{
		Name:       "EmailEmailDomain",
		SpecType:   reflect.TypeOf(emailv1beta1.EmailDomainSpec{}),
		StatusType: reflect.TypeOf(emailv1beta1.EmailDomainStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "email.CreateEmailDomainDetails",
			},
			{
				SDKStruct: "email.UpdateEmailDomainDetails",
			},
			{
				SDKStruct: "email.EmailDomain",
			},
			{
				SDKStruct: "email.EmailDomainCollection",
			},
			{
				SDKStruct: "email.EmailDomainSummary",
			},
		},
	},
	{
		Name:       "EmailSender",
		SpecType:   reflect.TypeOf(emailv1beta1.SenderSpec{}),
		StatusType: reflect.TypeOf(emailv1beta1.SenderStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "email.CreateSenderDetails",
			},
			{
				SDKStruct: "email.UpdateSenderDetails",
			},
			{
				SDKStruct: "email.Sender",
			},
			{
				SDKStruct: "email.SenderSummary",
			},
		},
	},
	{
		Name:       "EmailSuppression",
		SpecType:   reflect.TypeOf(emailv1beta1.SuppressionSpec{}),
		StatusType: reflect.TypeOf(emailv1beta1.SuppressionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "email.CreateSuppressionDetails",
			},
			{
				SDKStruct: "email.Suppression",
			},
			{
				SDKStruct: "email.SuppressionSummary",
			},
		},
	},
	{
		Name:       "GenerativeAIDedicatedAiCluster",
		SpecType:   reflect.TypeOf(generativeaiv1beta1.DedicatedAiClusterSpec{}),
		StatusType: reflect.TypeOf(generativeaiv1beta1.DedicatedAiClusterStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "generativeai.CreateDedicatedAiClusterDetails",
			},
			{
				SDKStruct: "generativeai.UpdateDedicatedAiClusterDetails",
			},
			{
				SDKStruct: "generativeai.DedicatedAiCluster",
			},
			{
				SDKStruct: "generativeai.DedicatedAiClusterCollection",
			},
			{
				SDKStruct: "generativeai.DedicatedAiClusterSummary",
			},
		},
	},
	{
		Name:       "GenerativeAIEndpoint",
		SpecType:   reflect.TypeOf(generativeaiv1beta1.EndpointSpec{}),
		StatusType: reflect.TypeOf(generativeaiv1beta1.EndpointStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "generativeai.CreateEndpointDetails",
			},
			{
				SDKStruct: "generativeai.UpdateEndpointDetails",
			},
			{
				SDKStruct: "generativeai.Endpoint",
			},
			{
				SDKStruct: "generativeai.EndpointCollection",
			},
			{
				SDKStruct: "generativeai.EndpointSummary",
			},
		},
	},
	{
		Name:       "GenerativeAIModel",
		SpecType:   reflect.TypeOf(generativeaiv1beta1.ModelSpec{}),
		StatusType: reflect.TypeOf(generativeaiv1beta1.ModelStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "generativeai.CreateModelDetails",
			},
			{
				SDKStruct: "generativeai.UpdateModelDetails",
			},
			{
				SDKStruct: "generativeai.Model",
			},
			{
				SDKStruct: "generativeai.ModelCollection",
			},
			{
				SDKStruct: "generativeai.ModelSummary",
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
		Name:       "MarketplaceAcceptedAgreement",
		SpecType:   reflect.TypeOf(marketplacev1beta1.AcceptedAgreementSpec{}),
		StatusType: reflect.TypeOf(marketplacev1beta1.AcceptedAgreementStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "marketplace.CreateAcceptedAgreementDetails",
			},
			{
				SDKStruct: "marketplace.UpdateAcceptedAgreementDetails",
			},
			{
				SDKStruct: "marketplace.AcceptedAgreement",
			},
			{
				SDKStruct: "marketplace.AcceptedAgreementSummary",
			},
		},
	},
	{
		Name:       "MarketplacePublication",
		SpecType:   reflect.TypeOf(marketplacev1beta1.PublicationSpec{}),
		StatusType: reflect.TypeOf(marketplacev1beta1.PublicationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "marketplace.CreatePublicationDetails",
			},
			{
				SDKStruct: "marketplace.UpdatePublicationDetails",
			},
			{
				SDKStruct: "marketplace.Publication",
			},
			{
				SDKStruct: "marketplace.PublicationSummary",
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
		Name:       "OCVPCluster",
		SpecType:   reflect.TypeOf(ocvpv1beta1.ClusterSpec{}),
		StatusType: reflect.TypeOf(ocvpv1beta1.ClusterStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "ocvp.CreateClusterDetails",
			},
			{
				SDKStruct: "ocvp.UpdateClusterDetails",
			},
			{
				SDKStruct: "ocvp.Cluster",
			},
			{
				SDKStruct: "ocvp.ClusterCollection",
			},
			{
				SDKStruct: "ocvp.ClusterSummary",
			},
		},
	},
	{
		Name:       "OCVPEsxiHost",
		SpecType:   reflect.TypeOf(ocvpv1beta1.EsxiHostSpec{}),
		StatusType: reflect.TypeOf(ocvpv1beta1.EsxiHostStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "ocvp.CreateEsxiHostDetails",
			},
			{
				SDKStruct: "ocvp.UpdateEsxiHostDetails",
			},
			{
				SDKStruct: "ocvp.EsxiHost",
			},
			{
				SDKStruct: "ocvp.EsxiHostCollection",
			},
			{
				SDKStruct: "ocvp.EsxiHostSummary",
			},
		},
	},
	{
		Name:       "OCVPSddc",
		SpecType:   reflect.TypeOf(ocvpv1beta1.SddcSpec{}),
		StatusType: reflect.TypeOf(ocvpv1beta1.SddcStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "ocvp.CreateSddcDetails",
			},
			{
				SDKStruct: "ocvp.UpdateSddcDetails",
			},
			{
				SDKStruct: "ocvp.Sddc",
			},
			{
				SDKStruct: "ocvp.SddcCollection",
			},
			{
				SDKStruct: "ocvp.SddcSummary",
			},
		},
	},
	{
		Name:       "ODAAuthenticationProvider",
		SpecType:   reflect.TypeOf(odav1beta1.AuthenticationProviderSpec{}),
		StatusType: reflect.TypeOf(odav1beta1.AuthenticationProviderStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "oda.CreateAuthenticationProviderDetails",
			},
			{
				SDKStruct: "oda.UpdateAuthenticationProviderDetails",
			},
			{
				SDKStruct: "oda.AuthenticationProvider",
			},
			{
				SDKStruct: "oda.AuthenticationProviderCollection",
			},
			{
				SDKStruct: "oda.AuthenticationProviderSummary",
			},
		},
	},
	{
		Name:       "ODAChannel",
		SpecType:   reflect.TypeOf(odav1beta1.ChannelSpec{}),
		StatusType: reflect.TypeOf(odav1beta1.ChannelStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "oda.ChannelCollection",
			},
			{
				SDKStruct: "oda.ChannelSummary",
			},
		},
	},
	{
		Name:       "ODADigitalAssistant",
		SpecType:   reflect.TypeOf(odav1beta1.DigitalAssistantSpec{}),
		StatusType: reflect.TypeOf(odav1beta1.DigitalAssistantStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "oda.UpdateDigitalAssistantDetails",
			},
			{
				SDKStruct: "oda.DigitalAssistant",
			},
			{
				SDKStruct: "oda.DigitalAssistantCollection",
			},
			{
				SDKStruct: "oda.DigitalAssistantSummary",
			},
		},
	},
	{
		Name:       "ODAImportedPackage",
		SpecType:   reflect.TypeOf(odav1beta1.ImportedPackageSpec{}),
		StatusType: reflect.TypeOf(odav1beta1.ImportedPackageStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "oda.CreateImportedPackageDetails",
			},
			{
				SDKStruct: "oda.UpdateImportedPackageDetails",
			},
			{
				SDKStruct: "oda.ImportedPackage",
			},
			{
				SDKStruct: "oda.ImportedPackageSummary",
			},
		},
	},
	{
		Name:       "ODAOdaInstance",
		SpecType:   reflect.TypeOf(odav1beta1.OdaInstanceSpec{}),
		StatusType: reflect.TypeOf(odav1beta1.OdaInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "oda.CreateOdaInstanceDetails",
			},
			{
				SDKStruct: "oda.UpdateOdaInstanceDetails",
			},
			{
				SDKStruct: "oda.OdaInstance",
			},
			{
				SDKStruct: "oda.OdaInstanceSummary",
			},
		},
	},
	{
		Name:       "ODAOdaInstanceAttachment",
		SpecType:   reflect.TypeOf(odav1beta1.OdaInstanceAttachmentSpec{}),
		StatusType: reflect.TypeOf(odav1beta1.OdaInstanceAttachmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "oda.CreateOdaInstanceAttachmentDetails",
			},
			{
				SDKStruct: "oda.UpdateOdaInstanceAttachmentDetails",
			},
			{
				SDKStruct: "oda.OdaInstanceAttachment",
			},
			{
				SDKStruct: "oda.OdaInstanceAttachmentCollection",
			},
			{
				SDKStruct: "oda.OdaInstanceAttachmentSummary",
			},
		},
	},
	{
		Name:       "ODAOdaPrivateEndpoint",
		SpecType:   reflect.TypeOf(odav1beta1.OdaPrivateEndpointSpec{}),
		StatusType: reflect.TypeOf(odav1beta1.OdaPrivateEndpointStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "oda.CreateOdaPrivateEndpointDetails",
			},
			{
				SDKStruct: "oda.UpdateOdaPrivateEndpointDetails",
			},
			{
				SDKStruct: "oda.OdaPrivateEndpoint",
			},
			{
				SDKStruct: "oda.OdaPrivateEndpointCollection",
			},
			{
				SDKStruct: "oda.OdaPrivateEndpointSummary",
			},
		},
	},
	{
		Name:       "ODAOdaPrivateEndpointAttachment",
		SpecType:   reflect.TypeOf(odav1beta1.OdaPrivateEndpointAttachmentSpec{}),
		StatusType: reflect.TypeOf(odav1beta1.OdaPrivateEndpointAttachmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "oda.CreateOdaPrivateEndpointAttachmentDetails",
			},
			{
				SDKStruct: "oda.OdaPrivateEndpointAttachment",
			},
			{
				SDKStruct: "oda.OdaPrivateEndpointAttachmentCollection",
			},
			{
				SDKStruct: "oda.OdaPrivateEndpointAttachmentSummary",
			},
		},
	},
	{
		Name:       "ODAOdaPrivateEndpointScanProxy",
		SpecType:   reflect.TypeOf(odav1beta1.OdaPrivateEndpointScanProxySpec{}),
		StatusType: reflect.TypeOf(odav1beta1.OdaPrivateEndpointScanProxyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "oda.CreateOdaPrivateEndpointScanProxyDetails",
			},
			{
				SDKStruct: "oda.OdaPrivateEndpointScanProxy",
			},
			{
				SDKStruct: "oda.OdaPrivateEndpointScanProxyCollection",
			},
			{
				SDKStruct: "oda.OdaPrivateEndpointScanProxySummary",
			},
		},
	},
	{
		Name:       "ODASkill",
		SpecType:   reflect.TypeOf(odav1beta1.SkillSpec{}),
		StatusType: reflect.TypeOf(odav1beta1.SkillStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "oda.UpdateSkillDetails",
			},
			{
				SDKStruct: "oda.Skill",
			},
			{
				SDKStruct: "oda.SkillCollection",
			},
			{
				SDKStruct: "oda.SkillSummary",
			},
		},
	},
	{
		Name:       "ODASkillParameter",
		SpecType:   reflect.TypeOf(odav1beta1.SkillParameterSpec{}),
		StatusType: reflect.TypeOf(odav1beta1.SkillParameterStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "oda.CreateSkillParameterDetails",
			},
			{
				SDKStruct: "oda.UpdateSkillParameterDetails",
			},
			{
				SDKStruct: "oda.SkillParameter",
			},
			{
				SDKStruct: "oda.SkillParameterCollection",
			},
			{
				SDKStruct: "oda.SkillParameterSummary",
			},
		},
	},
	{
		Name:       "ODATranslator",
		SpecType:   reflect.TypeOf(odav1beta1.TranslatorSpec{}),
		StatusType: reflect.TypeOf(odav1beta1.TranslatorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "oda.CreateTranslatorDetails",
			},
			{
				SDKStruct: "oda.UpdateTranslatorDetails",
			},
			{
				SDKStruct: "oda.Translator",
			},
			{
				SDKStruct: "oda.TranslatorCollection",
			},
			{
				SDKStruct: "oda.TranslatorSummary",
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
		Name:       "LoggingLogGroup",
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
			},
			{
				SDKStruct: "logging.UnifiedAgentConfigurationSummary",
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
		Name:       "UsageAPICustomTable",
		SpecType:   reflect.TypeOf(usageapiv1beta1.CustomTableSpec{}),
		StatusType: reflect.TypeOf(usageapiv1beta1.CustomTableStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "usageapi.CreateCustomTableDetails",
			},
			{
				SDKStruct: "usageapi.UpdateCustomTableDetails",
			},
			{
				SDKStruct: "usageapi.CustomTable",
			},
			{
				SDKStruct: "usageapi.CustomTableCollection",
			},
			{
				SDKStruct: "usageapi.CustomTableSummary",
			},
		},
	},
	{
		Name:       "UsageAPIQuery",
		SpecType:   reflect.TypeOf(usageapiv1beta1.QuerySpec{}),
		StatusType: reflect.TypeOf(usageapiv1beta1.QueryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "usageapi.CreateQueryDetails",
			},
			{
				SDKStruct: "usageapi.UpdateQueryDetails",
			},
			{
				SDKStruct: "usageapi.Query",
			},
			{
				SDKStruct: "usageapi.QueryCollection",
			},
			{
				SDKStruct: "usageapi.QuerySummary",
			},
		},
	},
	{
		Name:       "UsageAPISchedule",
		SpecType:   reflect.TypeOf(usageapiv1beta1.ScheduleSpec{}),
		StatusType: reflect.TypeOf(usageapiv1beta1.ScheduleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "usageapi.CreateScheduleDetails",
			},
			{
				SDKStruct: "usageapi.UpdateScheduleDetails",
			},
			{
				SDKStruct: "usageapi.Schedule",
			},
			{
				SDKStruct: "usageapi.ScheduleCollection",
			},
			{
				SDKStruct: "usageapi.ScheduleSummary",
			},
		},
	},
	{
		Name:       "UsageAPIUsageCarbonEmissionsQuery",
		SpecType:   reflect.TypeOf(usageapiv1beta1.UsageCarbonEmissionsQuerySpec{}),
		StatusType: reflect.TypeOf(usageapiv1beta1.UsageCarbonEmissionsQueryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "usageapi.CreateUsageCarbonEmissionsQueryDetails",
			},
			{
				SDKStruct: "usageapi.UpdateUsageCarbonEmissionsQueryDetails",
			},
			{
				SDKStruct: "usageapi.UsageCarbonEmissionsQuery",
			},
			{
				SDKStruct: "usageapi.UsageCarbonEmissionsQueryCollection",
			},
			{
				SDKStruct: "usageapi.UsageCarbonEmissionsQuerySummary",
			},
		},
	},
	{
		Name:       "MonitoringAlarm",
		SpecType:   reflect.TypeOf(monitoringv1beta1.AlarmSpec{}),
		StatusType: reflect.TypeOf(monitoringv1beta1.AlarmStatus{}),
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
		Name:       "LoadBalancerLoadBalancer",
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
		Name:       "AdmKnowledgeBase",
		SpecType:   reflect.TypeOf(admv1beta1.KnowledgeBaseSpec{}),
		StatusType: reflect.TypeOf(admv1beta1.KnowledgeBaseStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "adm.CreateKnowledgeBaseDetails",
			},
			{
				SDKStruct: "adm.UpdateKnowledgeBaseDetails",
			},
			{
				SDKStruct: "adm.KnowledgeBase",
			},
			{
				SDKStruct: "adm.KnowledgeBaseCollection",
			},
			{
				SDKStruct: "adm.KnowledgeBaseSummary",
			},
		},
	},
	{
		Name:       "AidocumentProject",
		SpecType:   reflect.TypeOf(aidocumentv1beta1.ProjectSpec{}),
		StatusType: reflect.TypeOf(aidocumentv1beta1.ProjectStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "aidocument.CreateProjectDetails",
			},
			{
				SDKStruct: "aidocument.UpdateProjectDetails",
			},
			{
				SDKStruct: "aidocument.Project",
			},
			{
				SDKStruct: "aidocument.ProjectCollection",
			},
			{
				SDKStruct: "aidocument.ProjectSummary",
			},
		},
	},
	{
		Name:       "AilanguageProject",
		SpecType:   reflect.TypeOf(ailanguagev1beta1.ProjectSpec{}),
		StatusType: reflect.TypeOf(ailanguagev1beta1.ProjectStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "ailanguage.CreateProjectDetails",
			},
			{
				SDKStruct: "ailanguage.UpdateProjectDetails",
			},
			{
				SDKStruct: "ailanguage.Project",
			},
			{
				SDKStruct: "ailanguage.ProjectCollection",
			},
			{
				SDKStruct: "ailanguage.ProjectSummary",
			},
		},
	},
	{
		Name:       "AispeechTranscriptionJob",
		SpecType:   reflect.TypeOf(aispeechv1beta1.TranscriptionJobSpec{}),
		StatusType: reflect.TypeOf(aispeechv1beta1.TranscriptionJobStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "aispeech.CreateTranscriptionJobDetails",
			},
			{
				SDKStruct: "aispeech.UpdateTranscriptionJobDetails",
			},
			{
				SDKStruct: "aispeech.TranscriptionJob",
			},
			{
				SDKStruct: "aispeech.TranscriptionJobCollection",
			},
			{
				SDKStruct: "aispeech.TranscriptionJobSummary",
			},
		},
	},
	{
		Name:       "AivisionProject",
		SpecType:   reflect.TypeOf(aivisionv1beta1.ProjectSpec{}),
		StatusType: reflect.TypeOf(aivisionv1beta1.ProjectStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "aivision.CreateProjectDetails",
			},
			{
				SDKStruct: "aivision.UpdateProjectDetails",
			},
			{
				SDKStruct: "aivision.Project",
			},
			{
				SDKStruct: "aivision.ProjectCollection",
			},
			{
				SDKStruct: "aivision.ProjectSummary",
			},
		},
	},
	{
		Name:       "AnalyticsAnalyticsInstance",
		SpecType:   reflect.TypeOf(analyticsv1beta1.AnalyticsInstanceSpec{}),
		StatusType: reflect.TypeOf(analyticsv1beta1.AnalyticsInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "analytics.CreateAnalyticsInstanceDetails",
			},
			{
				SDKStruct: "analytics.UpdateAnalyticsInstanceDetails",
			},
			{
				SDKStruct: "analytics.AnalyticsInstance",
			},
			{
				SDKStruct: "analytics.AnalyticsInstanceSummary",
			},
		},
	},
	{
		Name:       "BdsBdsInstance",
		SpecType:   reflect.TypeOf(bdsv1beta1.BdsInstanceSpec{}),
		StatusType: reflect.TypeOf(bdsv1beta1.BdsInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "bds.CreateBdsInstanceDetails",
			},
			{
				SDKStruct: "bds.UpdateBdsInstanceDetails",
			},
			{
				SDKStruct: "bds.BdsInstance",
			},
			{
				SDKStruct: "bds.BdsInstanceSummary",
			},
		},
	},
	{
		Name:       "BudgetBudget",
		SpecType:   reflect.TypeOf(budgetv1beta1.BudgetSpec{}),
		StatusType: reflect.TypeOf(budgetv1beta1.BudgetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "budget.CreateBudgetDetails",
			},
			{
				SDKStruct: "budget.UpdateBudgetDetails",
			},
			{
				SDKStruct: "budget.Budget",
			},
			{
				SDKStruct: "budget.BudgetSummary",
			},
		},
	},
	{
		Name:       "ClusterplacementgroupsClusterPlacementGroup",
		SpecType:   reflect.TypeOf(clusterplacementgroupsv1beta1.ClusterPlacementGroupSpec{}),
		StatusType: reflect.TypeOf(clusterplacementgroupsv1beta1.ClusterPlacementGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "clusterplacementgroups.CreateClusterPlacementGroupDetails",
			},
			{
				SDKStruct: "clusterplacementgroups.UpdateClusterPlacementGroupDetails",
			},
			{
				SDKStruct: "clusterplacementgroups.ClusterPlacementGroup",
			},
			{
				SDKStruct: "clusterplacementgroups.ClusterPlacementGroupCollection",
			},
			{
				SDKStruct: "clusterplacementgroups.ClusterPlacementGroupSummary",
			},
		},
	},
	{
		Name:       "ContainerinstancesContainerInstance",
		SpecType:   reflect.TypeOf(containerinstancesv1beta1.ContainerInstanceSpec{}),
		StatusType: reflect.TypeOf(containerinstancesv1beta1.ContainerInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "containerinstances.CreateContainerInstanceDetails",
			},
			{
				SDKStruct: "containerinstances.UpdateContainerInstanceDetails",
			},
			{
				SDKStruct: "containerinstances.ContainerInstance",
			},
			{
				SDKStruct: "containerinstances.ContainerInstanceCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct: "containerinstances.ContainerInstanceSummary",
			},
		},
	},
	{
		Name:       "DatabasetoolsDatabaseToolsConnection",
		SpecType:   reflect.TypeOf(databasetoolsv1beta1.DatabaseToolsConnectionSpec{}),
		StatusType: reflect.TypeOf(databasetoolsv1beta1.DatabaseToolsConnectionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "databasetools.CreateDatabaseToolsConnectionGenericJdbcDetails",
			},
			{
				SDKStruct: "databasetools.CreateDatabaseToolsConnectionMySqlDetails",
			},
			{
				SDKStruct: "databasetools.CreateDatabaseToolsConnectionOracleDatabaseDetails",
			},
			{
				SDKStruct: "databasetools.CreateDatabaseToolsConnectionPostgresqlDetails",
			},
			{
				SDKStruct: "databasetools.UpdateDatabaseToolsConnectionGenericJdbcDetails",
			},
			{
				SDKStruct: "databasetools.UpdateDatabaseToolsConnectionMySqlDetails",
			},
			{
				SDKStruct: "databasetools.UpdateDatabaseToolsConnectionOracleDatabaseDetails",
			},
			{
				SDKStruct: "databasetools.UpdateDatabaseToolsConnectionPostgresqlDetails",
			},
			{
				SDKStruct: "databasetools.DatabaseToolsConnectionCollection",
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
			},
			{
				SDKStruct:  "databasetools.DatabaseToolsConnectionGenericJdbc",
				APISurface: "status",
			},
			{
				SDKStruct:  "databasetools.DatabaseToolsConnectionMySql",
				APISurface: "status",
			},
			{
				SDKStruct:  "databasetools.DatabaseToolsConnectionOracleDatabase",
				APISurface: "status",
			},
			{
				SDKStruct:  "databasetools.DatabaseToolsConnectionPostgresql",
				APISurface: "status",
			},
			{
				SDKStruct:  "databasetools.DatabaseToolsConnectionGenericJdbcSummary",
				APISurface: "status",
			},
			{
				SDKStruct:  "databasetools.DatabaseToolsConnectionMySqlSummary",
				APISurface: "status",
			},
			{
				SDKStruct:  "databasetools.DatabaseToolsConnectionOracleDatabaseSummary",
				APISurface: "status",
			},
			{
				SDKStruct:  "databasetools.DatabaseToolsConnectionPostgresqlSummary",
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
		Name:       "DatascienceProject",
		SpecType:   reflect.TypeOf(datasciencev1beta1.ProjectSpec{}),
		StatusType: reflect.TypeOf(datasciencev1beta1.ProjectStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datascience.CreateProjectDetails",
			},
			{
				SDKStruct: "datascience.UpdateProjectDetails",
			},
			{
				SDKStruct: "datascience.Project",
			},
			{
				SDKStruct: "datascience.ProjectSummary",
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
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
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
				Exclude:   true,
				Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
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
