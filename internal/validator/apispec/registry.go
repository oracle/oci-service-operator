package apispec

import (
	"reflect"

	aidocumentv1beta1 "github.com/oracle/oci-service-operator/api/aidocument/v1beta1"
	ailanguagev1beta1 "github.com/oracle/oci-service-operator/api/ailanguage/v1beta1"
	aispeechv1beta1 "github.com/oracle/oci-service-operator/api/aispeech/v1beta1"
	aivisionv1beta1 "github.com/oracle/oci-service-operator/api/aivision/v1beta1"
	analyticsv1beta1 "github.com/oracle/oci-service-operator/api/analytics/v1beta1"
	artifactsv1beta1 "github.com/oracle/oci-service-operator/api/artifacts/v1beta1"
	bastionv1beta1 "github.com/oracle/oci-service-operator/api/bastion/v1beta1"
	bdsv1beta1 "github.com/oracle/oci-service-operator/api/bds/v1beta1"
	certificatesmanagementv1beta1 "github.com/oracle/oci-service-operator/api/certificatesmanagement/v1beta1"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	containerinstancesv1beta1 "github.com/oracle/oci-service-operator/api/containerinstances/v1beta1"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	databasetoolsv1beta1 "github.com/oracle/oci-service-operator/api/databasetools/v1beta1"
	dataflowv1beta1 "github.com/oracle/oci-service-operator/api/dataflow/v1beta1"
	datasciencev1beta1 "github.com/oracle/oci-service-operator/api/datascience/v1beta1"
	devopsv1beta1 "github.com/oracle/oci-service-operator/api/devops/v1beta1"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	emailv1beta1 "github.com/oracle/oci-service-operator/api/email/v1beta1"
	eventsv1beta1 "github.com/oracle/oci-service-operator/api/events/v1beta1"
	functionsv1beta1 "github.com/oracle/oci-service-operator/api/functions/v1beta1"
	generativeaiv1beta1 "github.com/oracle/oci-service-operator/api/generativeai/v1beta1"
	governancerulescontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/governancerulescontrolplane/v1beta1"
	healthchecksv1beta1 "github.com/oracle/oci-service-operator/api/healthchecks/v1beta1"
	identityv1beta1 "github.com/oracle/oci-service-operator/api/identity/v1beta1"
	integrationv1beta1 "github.com/oracle/oci-service-operator/api/integration/v1beta1"
	iotv1beta1 "github.com/oracle/oci-service-operator/api/iot/v1beta1"
	keymanagementv1beta1 "github.com/oracle/oci-service-operator/api/keymanagement/v1beta1"
	licensemanagerv1beta1 "github.com/oracle/oci-service-operator/api/licensemanager/v1beta1"
	limitsv1beta1 "github.com/oracle/oci-service-operator/api/limits/v1beta1"
	limitsincreasev1beta1 "github.com/oracle/oci-service-operator/api/limitsincrease/v1beta1"
	loadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/loadbalancer/v1beta1"
	lockboxv1beta1 "github.com/oracle/oci-service-operator/api/lockbox/v1beta1"
	loganalyticsv1beta1 "github.com/oracle/oci-service-operator/api/loganalytics/v1beta1"
	loggingv1beta1 "github.com/oracle/oci-service-operator/api/logging/v1beta1"
	managedkafkav1beta1 "github.com/oracle/oci-service-operator/api/managedkafka/v1beta1"
	managementagentv1beta1 "github.com/oracle/oci-service-operator/api/managementagent/v1beta1"
	managementdashboardv1beta1 "github.com/oracle/oci-service-operator/api/managementdashboard/v1beta1"
	marketplacev1beta1 "github.com/oracle/oci-service-operator/api/marketplace/v1beta1"
	marketplaceprivateofferv1beta1 "github.com/oracle/oci-service-operator/api/marketplaceprivateoffer/v1beta1"
	marketplacepublisherv1beta1 "github.com/oracle/oci-service-operator/api/marketplacepublisher/v1beta1"
	monitoringv1beta1 "github.com/oracle/oci-service-operator/api/monitoring/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	networkloadbalancerv1beta1 "github.com/oracle/oci-service-operator/api/networkloadbalancer/v1beta1"
	nosqlv1beta1 "github.com/oracle/oci-service-operator/api/nosql/v1beta1"
	objectstoragev1beta1 "github.com/oracle/oci-service-operator/api/objectstorage/v1beta1"
	ocvpv1beta1 "github.com/oracle/oci-service-operator/api/ocvp/v1beta1"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	onsv1beta1 "github.com/oracle/oci-service-operator/api/ons/v1beta1"
	opensearchv1beta1 "github.com/oracle/oci-service-operator/api/opensearch/v1beta1"
	operatoraccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/operatoraccesscontrol/v1beta1"
	opsiv1beta1 "github.com/oracle/oci-service-operator/api/opsi/v1beta1"
	optimizerv1beta1 "github.com/oracle/oci-service-operator/api/optimizer/v1beta1"
	osmanagementhubv1beta1 "github.com/oracle/oci-service-operator/api/osmanagementhub/v1beta1"
	psqlv1beta1 "github.com/oracle/oci-service-operator/api/psql/v1beta1"
	queuev1beta1 "github.com/oracle/oci-service-operator/api/queue/v1beta1"
	recoveryv1beta1 "github.com/oracle/oci-service-operator/api/recovery/v1beta1"
	redisv1beta1 "github.com/oracle/oci-service-operator/api/redis/v1beta1"
	resourceanalyticsv1beta1 "github.com/oracle/oci-service-operator/api/resourceanalytics/v1beta1"
	resourceschedulerv1beta1 "github.com/oracle/oci-service-operator/api/resourcescheduler/v1beta1"
	schv1beta1 "github.com/oracle/oci-service-operator/api/sch/v1beta1"
	securityattributev1beta1 "github.com/oracle/oci-service-operator/api/securityattribute/v1beta1"
	servicecatalogv1beta1 "github.com/oracle/oci-service-operator/api/servicecatalog/v1beta1"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	usageapiv1beta1 "github.com/oracle/oci-service-operator/api/usageapi/v1beta1"
	waav1beta1 "github.com/oracle/oci-service-operator/api/waa/v1beta1"
	waasv1beta1 "github.com/oracle/oci-service-operator/api/waas/v1beta1"
	wafv1beta1 "github.com/oracle/oci-service-operator/api/waf/v1beta1"
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
		Name:       "NotificationSubscription",
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
		Name:       "NetworkLoadBalancerNetworkLoadBalancer",
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
		Name:       "ArtifactsRepository",
		SpecType:   reflect.TypeOf(artifactsv1beta1.RepositorySpec{}),
		StatusType: reflect.TypeOf(artifactsv1beta1.RepositoryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "artifacts.ContainerRepository",
				Exclude:   true,
				Reason:    "Intentionally untracked: ArtifactsRepository status represents generic repositories; container repository parity is tracked on ArtifactsContainerRepository.",
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
		Name:       "BastionBastion",
		SpecType:   reflect.TypeOf(bastionv1beta1.BastionSpec{}),
		StatusType: reflect.TypeOf(bastionv1beta1.BastionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "bastion.CreateBastionDetails",
			},
			{
				SDKStruct: "bastion.UpdateBastionDetails",
			},
			{
				SDKStruct: "bastion.Bastion",
			},
			{
				SDKStruct: "bastion.BastionSummary",
			},
		},
	},
	{
		Name:       "BastionSession",
		SpecType:   reflect.TypeOf(bastionv1beta1.SessionSpec{}),
		StatusType: reflect.TypeOf(bastionv1beta1.SessionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "bastion.CreateSessionDetails",
			},
			{
				SDKStruct: "bastion.UpdateSessionDetails",
			},
			{
				SDKStruct: "bastion.Session",
			},
			{
				SDKStruct: "bastion.SessionSummary",
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
		Name:       "DevopsBuildPipeline",
		SpecType:   reflect.TypeOf(devopsv1beta1.BuildPipelineSpec{}),
		StatusType: reflect.TypeOf(devopsv1beta1.BuildPipelineStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "devops.CreateBuildPipelineDetails",
			},
			{
				SDKStruct: "devops.UpdateBuildPipelineDetails",
			},
			{
				SDKStruct: "devops.BuildPipeline",
			},
			{
				SDKStruct: "devops.BuildPipelineCollection",
			},
			{
				SDKStruct: "devops.BuildPipelineSummary",
			},
		},
	},
	{
		Name:       "DevopsDeployArtifact",
		SpecType:   reflect.TypeOf(devopsv1beta1.DeployArtifactSpec{}),
		StatusType: reflect.TypeOf(devopsv1beta1.DeployArtifactStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "devops.CreateDeployArtifactDetails",
			},
			{
				SDKStruct: "devops.UpdateDeployArtifactDetails",
			},
			{
				SDKStruct: "devops.DeployArtifact",
			},
			{
				SDKStruct: "devops.DeployArtifactCollection",
			},
			{
				SDKStruct: "devops.DeployArtifactSummary",
			},
		},
	},
	{
		Name:       "DevopsDeployPipeline",
		SpecType:   reflect.TypeOf(devopsv1beta1.DeployPipelineSpec{}),
		StatusType: reflect.TypeOf(devopsv1beta1.DeployPipelineStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "devops.CreateDeployPipelineDetails",
			},
			{
				SDKStruct: "devops.UpdateDeployPipelineDetails",
			},
			{
				SDKStruct: "devops.DeployPipeline",
			},
			{
				SDKStruct: "devops.DeployPipelineCollection",
			},
			{
				SDKStruct: "devops.DeployPipelineSummary",
			},
		},
	},
	{
		Name:       "DevopsProject",
		SpecType:   reflect.TypeOf(devopsv1beta1.ProjectSpec{}),
		StatusType: reflect.TypeOf(devopsv1beta1.ProjectStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "devops.CreateProjectDetails",
			},
			{
				SDKStruct: "devops.UpdateProjectDetails",
			},
			{
				SDKStruct: "devops.Project",
			},
			{
				SDKStruct: "devops.ProjectCollection",
			},
			{
				SDKStruct: "devops.ProjectSummary",
			},
		},
	},
	{
		Name:       "DevopsRepository",
		SpecType:   reflect.TypeOf(devopsv1beta1.RepositorySpec{}),
		StatusType: reflect.TypeOf(devopsv1beta1.RepositoryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "devops.CreateRepositoryDetails",
			},
			{
				SDKStruct: "devops.UpdateRepositoryDetails",
			},
			{
				SDKStruct: "devops.Repository",
			},
			{
				SDKStruct: "devops.RepositoryCollection",
			},
			{
				SDKStruct: "devops.RepositorySummary",
			},
		},
	},
	{
		Name:       "DevopsTrigger",
		SpecType:   reflect.TypeOf(devopsv1beta1.TriggerSpec{}),
		StatusType: reflect.TypeOf(devopsv1beta1.TriggerStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "devops.TriggerCollection",
			},
		},
	},
	{
		Name:       "GovernancerulescontrolplaneGovernanceRule",
		SpecType:   reflect.TypeOf(governancerulescontrolplanev1beta1.GovernanceRuleSpec{}),
		StatusType: reflect.TypeOf(governancerulescontrolplanev1beta1.GovernanceRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "governancerulescontrolplane.CreateGovernanceRuleDetails",
			},
			{
				SDKStruct: "governancerulescontrolplane.UpdateGovernanceRuleDetails",
			},
			{
				SDKStruct: "governancerulescontrolplane.GovernanceRule",
			},
			{
				SDKStruct: "governancerulescontrolplane.GovernanceRuleCollection",
			},
			{
				SDKStruct: "governancerulescontrolplane.GovernanceRuleSummary",
			},
		},
	},
	{
		Name:       "HealthchecksHttpMonitor",
		SpecType:   reflect.TypeOf(healthchecksv1beta1.HttpMonitorSpec{}),
		StatusType: reflect.TypeOf(healthchecksv1beta1.HttpMonitorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "healthchecks.CreateHttpMonitorDetails",
			},
			{
				SDKStruct: "healthchecks.UpdateHttpMonitorDetails",
			},
			{
				SDKStruct: "healthchecks.HttpMonitor",
			},
			{
				SDKStruct: "healthchecks.HttpMonitorSummary",
			},
		},
	},
	{
		Name:       "HealthchecksPingMonitor",
		SpecType:   reflect.TypeOf(healthchecksv1beta1.PingMonitorSpec{}),
		StatusType: reflect.TypeOf(healthchecksv1beta1.PingMonitorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "healthchecks.CreatePingMonitorDetails",
			},
			{
				SDKStruct: "healthchecks.UpdatePingMonitorDetails",
			},
			{
				SDKStruct: "healthchecks.PingMonitor",
			},
			{
				SDKStruct: "healthchecks.PingMonitorSummary",
			},
		},
	},
	{
		Name:       "IntegrationIntegrationInstance",
		SpecType:   reflect.TypeOf(integrationv1beta1.IntegrationInstanceSpec{}),
		StatusType: reflect.TypeOf(integrationv1beta1.IntegrationInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "integration.CreateIntegrationInstanceDetails",
			},
			{
				SDKStruct: "integration.UpdateIntegrationInstanceDetails",
			},
			{
				SDKStruct: "integration.IntegrationInstance",
			},
			{
				SDKStruct: "integration.IntegrationInstanceSummary",
			},
		},
	},
	{
		Name:       "IotDigitalTwinAdapter",
		SpecType:   reflect.TypeOf(iotv1beta1.DigitalTwinAdapterSpec{}),
		StatusType: reflect.TypeOf(iotv1beta1.DigitalTwinAdapterStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "iot.CreateDigitalTwinAdapterDetails",
			},
			{
				SDKStruct: "iot.UpdateDigitalTwinAdapterDetails",
			},
			{
				SDKStruct: "iot.DigitalTwinAdapter",
			},
			{
				SDKStruct: "iot.DigitalTwinAdapterCollection",
			},
			{
				SDKStruct: "iot.DigitalTwinAdapterSummary",
			},
		},
	},
	{
		Name:       "IotDigitalTwinInstance",
		SpecType:   reflect.TypeOf(iotv1beta1.DigitalTwinInstanceSpec{}),
		StatusType: reflect.TypeOf(iotv1beta1.DigitalTwinInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "iot.CreateDigitalTwinInstanceDetails",
			},
			{
				SDKStruct: "iot.UpdateDigitalTwinInstanceDetails",
			},
			{
				SDKStruct: "iot.DigitalTwinInstance",
			},
			{
				SDKStruct: "iot.DigitalTwinInstanceCollection",
			},
			{
				SDKStruct: "iot.DigitalTwinInstanceSummary",
			},
		},
	},
	{
		Name:       "IotDigitalTwinModel",
		SpecType:   reflect.TypeOf(iotv1beta1.DigitalTwinModelSpec{}),
		StatusType: reflect.TypeOf(iotv1beta1.DigitalTwinModelStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "iot.CreateDigitalTwinModelDetails",
			},
			{
				SDKStruct: "iot.UpdateDigitalTwinModelDetails",
			},
			{
				SDKStruct: "iot.DigitalTwinModel",
			},
			{
				SDKStruct: "iot.DigitalTwinModelCollection",
			},
			{
				SDKStruct: "iot.DigitalTwinModelSummary",
			},
		},
	},
	{
		Name:       "IotDigitalTwinRelationship",
		SpecType:   reflect.TypeOf(iotv1beta1.DigitalTwinRelationshipSpec{}),
		StatusType: reflect.TypeOf(iotv1beta1.DigitalTwinRelationshipStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "iot.CreateDigitalTwinRelationshipDetails",
			},
			{
				SDKStruct: "iot.UpdateDigitalTwinRelationshipDetails",
			},
			{
				SDKStruct: "iot.DigitalTwinRelationship",
			},
			{
				SDKStruct: "iot.DigitalTwinRelationshipCollection",
			},
			{
				SDKStruct: "iot.DigitalTwinRelationshipSummary",
			},
		},
	},
	{
		Name:       "IotIotDomain",
		SpecType:   reflect.TypeOf(iotv1beta1.IotDomainSpec{}),
		StatusType: reflect.TypeOf(iotv1beta1.IotDomainStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "iot.CreateIotDomainDetails",
			},
			{
				SDKStruct: "iot.UpdateIotDomainDetails",
			},
			{
				SDKStruct: "iot.IotDomain",
			},
			{
				SDKStruct: "iot.IotDomainCollection",
			},
			{
				SDKStruct: "iot.IotDomainSummary",
			},
		},
	},
	{
		Name:       "IotIotDomainGroup",
		SpecType:   reflect.TypeOf(iotv1beta1.IotDomainGroupSpec{}),
		StatusType: reflect.TypeOf(iotv1beta1.IotDomainGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "iot.CreateIotDomainGroupDetails",
			},
			{
				SDKStruct: "iot.UpdateIotDomainGroupDetails",
			},
			{
				SDKStruct: "iot.IotDomainGroup",
			},
			{
				SDKStruct: "iot.IotDomainGroupCollection",
			},
			{
				SDKStruct: "iot.IotDomainGroupSummary",
			},
		},
	},
	{
		Name:       "LicensemanagerLicenseRecord",
		SpecType:   reflect.TypeOf(licensemanagerv1beta1.LicenseRecordSpec{}),
		StatusType: reflect.TypeOf(licensemanagerv1beta1.LicenseRecordStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "licensemanager.CreateLicenseRecordDetails",
			},
			{
				SDKStruct: "licensemanager.UpdateLicenseRecordDetails",
			},
			{
				SDKStruct: "licensemanager.LicenseRecord",
			},
			{
				SDKStruct: "licensemanager.LicenseRecordCollection",
			},
			{
				SDKStruct: "licensemanager.LicenseRecordSummary",
			},
		},
	},
	{
		Name:       "LicensemanagerProductLicense",
		SpecType:   reflect.TypeOf(licensemanagerv1beta1.ProductLicenseSpec{}),
		StatusType: reflect.TypeOf(licensemanagerv1beta1.ProductLicenseStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "licensemanager.CreateProductLicenseDetails",
			},
			{
				SDKStruct: "licensemanager.UpdateProductLicenseDetails",
			},
			{
				SDKStruct: "licensemanager.ProductLicense",
			},
			{
				SDKStruct: "licensemanager.ProductLicenseCollection",
			},
			{
				SDKStruct: "licensemanager.ProductLicenseSummary",
			},
		},
	},
	{
		Name:       "LimitsincreaseLimitsIncreaseRequest",
		SpecType:   reflect.TypeOf(limitsincreasev1beta1.LimitsIncreaseRequestSpec{}),
		StatusType: reflect.TypeOf(limitsincreasev1beta1.LimitsIncreaseRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "limitsincrease.CreateLimitsIncreaseRequestDetails",
			},
			{
				SDKStruct: "limitsincrease.UpdateLimitsIncreaseRequestDetails",
			},
			{
				SDKStruct: "limitsincrease.LimitsIncreaseRequest",
			},
			{
				SDKStruct: "limitsincrease.LimitsIncreaseRequestCollection",
			},
			{
				SDKStruct: "limitsincrease.LimitsIncreaseRequestSummary",
			},
		},
	},
	{
		Name:       "LockboxApprovalTemplate",
		SpecType:   reflect.TypeOf(lockboxv1beta1.ApprovalTemplateSpec{}),
		StatusType: reflect.TypeOf(lockboxv1beta1.ApprovalTemplateStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "lockbox.CreateApprovalTemplateDetails",
			},
			{
				SDKStruct: "lockbox.UpdateApprovalTemplateDetails",
			},
			{
				SDKStruct: "lockbox.ApprovalTemplate",
			},
			{
				SDKStruct: "lockbox.ApprovalTemplateCollection",
			},
			{
				SDKStruct: "lockbox.ApprovalTemplateSummary",
			},
		},
	},
	{
		Name:       "LockboxLockbox",
		SpecType:   reflect.TypeOf(lockboxv1beta1.LockboxSpec{}),
		StatusType: reflect.TypeOf(lockboxv1beta1.LockboxStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "lockbox.CreateLockboxDetails",
			},
			{
				SDKStruct: "lockbox.UpdateLockboxDetails",
			},
			{
				SDKStruct: "lockbox.Lockbox",
			},
			{
				SDKStruct: "lockbox.LockboxCollection",
			},
			{
				SDKStruct: "lockbox.LockboxSummary",
			},
		},
	},
	{
		Name:       "LoganalyticsIngestTimeRule",
		SpecType:   reflect.TypeOf(loganalyticsv1beta1.IngestTimeRuleSpec{}),
		StatusType: reflect.TypeOf(loganalyticsv1beta1.IngestTimeRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loganalytics.CreateIngestTimeRuleDetails",
			},
			{
				SDKStruct: "loganalytics.IngestTimeRule",
			},
			{
				SDKStruct: "loganalytics.IngestTimeRuleSummary",
			},
		},
	},
	{
		Name:       "LoganalyticsLogAnalyticsEmBridge",
		SpecType:   reflect.TypeOf(loganalyticsv1beta1.LogAnalyticsEmBridgeSpec{}),
		StatusType: reflect.TypeOf(loganalyticsv1beta1.LogAnalyticsEmBridgeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loganalytics.CreateLogAnalyticsEmBridgeDetails",
			},
			{
				SDKStruct: "loganalytics.UpdateLogAnalyticsEmBridgeDetails",
			},
			{
				SDKStruct: "loganalytics.LogAnalyticsEmBridge",
			},
			{
				SDKStruct: "loganalytics.LogAnalyticsEmBridgeCollection",
			},
			{
				SDKStruct: "loganalytics.LogAnalyticsEmBridgeSummary",
			},
		},
	},
	{
		Name:       "LoganalyticsLogAnalyticsEntity",
		SpecType:   reflect.TypeOf(loganalyticsv1beta1.LogAnalyticsEntitySpec{}),
		StatusType: reflect.TypeOf(loganalyticsv1beta1.LogAnalyticsEntityStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loganalytics.CreateLogAnalyticsEntityDetails",
			},
			{
				SDKStruct: "loganalytics.UpdateLogAnalyticsEntityDetails",
			},
			{
				SDKStruct: "loganalytics.LogAnalyticsEntity",
			},
			{
				SDKStruct: "loganalytics.LogAnalyticsEntityCollection",
			},
			{
				SDKStruct: "loganalytics.LogAnalyticsEntitySummary",
			},
		},
	},
	{
		Name:       "LoganalyticsLogAnalyticsEntityType",
		SpecType:   reflect.TypeOf(loganalyticsv1beta1.LogAnalyticsEntityTypeSpec{}),
		StatusType: reflect.TypeOf(loganalyticsv1beta1.LogAnalyticsEntityTypeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loganalytics.CreateLogAnalyticsEntityTypeDetails",
			},
			{
				SDKStruct: "loganalytics.UpdateLogAnalyticsEntityTypeDetails",
			},
			{
				SDKStruct: "loganalytics.LogAnalyticsEntityType",
			},
			{
				SDKStruct: "loganalytics.LogAnalyticsEntityTypeCollection",
			},
			{
				SDKStruct: "loganalytics.LogAnalyticsEntityTypeSummary",
			},
		},
	},
	{
		Name:       "LoganalyticsLogAnalyticsLogGroup",
		SpecType:   reflect.TypeOf(loganalyticsv1beta1.LogAnalyticsLogGroupSpec{}),
		StatusType: reflect.TypeOf(loganalyticsv1beta1.LogAnalyticsLogGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loganalytics.CreateLogAnalyticsLogGroupDetails",
			},
			{
				SDKStruct: "loganalytics.UpdateLogAnalyticsLogGroupDetails",
			},
			{
				SDKStruct: "loganalytics.LogAnalyticsLogGroup",
			},
			{
				SDKStruct: "loganalytics.LogAnalyticsLogGroupSummary",
			},
		},
	},
	{
		Name:       "LoganalyticsLogAnalyticsObjectCollectionRule",
		SpecType:   reflect.TypeOf(loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleSpec{}),
		StatusType: reflect.TypeOf(loganalyticsv1beta1.LogAnalyticsObjectCollectionRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loganalytics.CreateLogAnalyticsObjectCollectionRuleDetails",
			},
			{
				SDKStruct: "loganalytics.UpdateLogAnalyticsObjectCollectionRuleDetails",
			},
			{
				SDKStruct: "loganalytics.LogAnalyticsObjectCollectionRule",
			},
			{
				SDKStruct: "loganalytics.LogAnalyticsObjectCollectionRuleCollection",
			},
			{
				SDKStruct: "loganalytics.LogAnalyticsObjectCollectionRuleSummary",
			},
		},
	},
	{
		Name:       "LoganalyticsScheduledTask",
		SpecType:   reflect.TypeOf(loganalyticsv1beta1.ScheduledTaskSpec{}),
		StatusType: reflect.TypeOf(loganalyticsv1beta1.ScheduledTaskStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "loganalytics.ScheduledTaskCollection",
			},
			{
				SDKStruct: "loganalytics.ScheduledTaskSummary",
			},
		},
	},
	{
		Name:       "ManagedkafkaKafkaCluster",
		SpecType:   reflect.TypeOf(managedkafkav1beta1.KafkaClusterSpec{}),
		StatusType: reflect.TypeOf(managedkafkav1beta1.KafkaClusterStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "managedkafka.CreateKafkaClusterDetails",
			},
			{
				SDKStruct: "managedkafka.UpdateKafkaClusterDetails",
			},
			{
				SDKStruct: "managedkafka.KafkaCluster",
			},
			{
				SDKStruct: "managedkafka.KafkaClusterCollection",
			},
			{
				SDKStruct: "managedkafka.KafkaClusterSummary",
			},
		},
	},
	{
		Name:       "ManagedkafkaKafkaClusterConfig",
		SpecType:   reflect.TypeOf(managedkafkav1beta1.KafkaClusterConfigSpec{}),
		StatusType: reflect.TypeOf(managedkafkav1beta1.KafkaClusterConfigStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "managedkafka.CreateKafkaClusterConfigDetails",
			},
			{
				SDKStruct: "managedkafka.UpdateKafkaClusterConfigDetails",
			},
			{
				SDKStruct: "managedkafka.KafkaClusterConfig",
			},
			{
				SDKStruct: "managedkafka.KafkaClusterConfigCollection",
			},
			{
				SDKStruct: "managedkafka.KafkaClusterConfigVersionSummary",
			},
			{
				SDKStruct: "managedkafka.KafkaClusterConfigSummary",
			},
		},
	},
	{
		Name:        "ManagementagentDataSource",
		SpecType:    reflect.TypeOf(managementagentv1beta1.DataSourceSpec{}),
		StatusType:  reflect.TypeOf(managementagentv1beta1.DataSourceStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:       "ManagementagentManagementAgentInstallKey",
		SpecType:   reflect.TypeOf(managementagentv1beta1.ManagementAgentInstallKeySpec{}),
		StatusType: reflect.TypeOf(managementagentv1beta1.ManagementAgentInstallKeyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "managementagent.CreateManagementAgentInstallKeyDetails",
			},
			{
				SDKStruct: "managementagent.UpdateManagementAgentInstallKeyDetails",
			},
			{
				SDKStruct: "managementagent.ManagementAgentInstallKey",
			},
			{
				SDKStruct: "managementagent.ManagementAgentInstallKeySummary",
			},
		},
	},
	{
		Name:       "ManagementagentNamedCredential",
		SpecType:   reflect.TypeOf(managementagentv1beta1.NamedCredentialSpec{}),
		StatusType: reflect.TypeOf(managementagentv1beta1.NamedCredentialStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "managementagent.CreateNamedCredentialDetails",
			},
			{
				SDKStruct: "managementagent.UpdateNamedCredentialDetails",
			},
			{
				SDKStruct: "managementagent.NamedCredential",
			},
			{
				SDKStruct: "managementagent.NamedCredentialCollection",
			},
			{
				SDKStruct: "managementagent.NamedCredentialSummary",
			},
		},
	},
	{
		Name:       "ManagementdashboardManagementDashboard",
		SpecType:   reflect.TypeOf(managementdashboardv1beta1.ManagementDashboardSpec{}),
		StatusType: reflect.TypeOf(managementdashboardv1beta1.ManagementDashboardStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "managementdashboard.CreateManagementDashboardDetails",
			},
			{
				SDKStruct: "managementdashboard.UpdateManagementDashboardDetails",
			},
			{
				SDKStruct: "managementdashboard.ManagementDashboard",
			},
			{
				SDKStruct: "managementdashboard.ManagementDashboardCollection",
			},
			{
				SDKStruct: "managementdashboard.ManagementDashboardSummary",
			},
		},
	},
	{
		Name:       "ManagementdashboardManagementSavedSearch",
		SpecType:   reflect.TypeOf(managementdashboardv1beta1.ManagementSavedSearchSpec{}),
		StatusType: reflect.TypeOf(managementdashboardv1beta1.ManagementSavedSearchStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "managementdashboard.CreateManagementSavedSearchDetails",
			},
			{
				SDKStruct: "managementdashboard.UpdateManagementSavedSearchDetails",
			},
			{
				SDKStruct: "managementdashboard.ManagementSavedSearch",
			},
			{
				SDKStruct: "managementdashboard.ManagementSavedSearchCollection",
			},
			{
				SDKStruct: "managementdashboard.ManagementSavedSearchSummary",
			},
		},
	},
	{
		Name:       "MarketplaceprivateofferOffer",
		SpecType:   reflect.TypeOf(marketplaceprivateofferv1beta1.OfferSpec{}),
		StatusType: reflect.TypeOf(marketplaceprivateofferv1beta1.OfferStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "marketplaceprivateoffer.CreateOfferDetails",
			},
			{
				SDKStruct: "marketplaceprivateoffer.UpdateOfferDetails",
			},
			{
				SDKStruct: "marketplaceprivateoffer.Offer",
			},
			{
				SDKStruct: "marketplaceprivateoffer.OfferCollection",
			},
			{
				SDKStruct: "marketplaceprivateoffer.OfferSummary",
			},
		},
	},
	{
		Name:       "MarketplacepublisherArtifact",
		SpecType:   reflect.TypeOf(marketplacepublisherv1beta1.ArtifactSpec{}),
		StatusType: reflect.TypeOf(marketplacepublisherv1beta1.ArtifactStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "marketplacepublisher.ArtifactCollection",
			},
			{
				SDKStruct: "marketplacepublisher.ArtifactSummary",
			},
		},
	},
	{
		Name:       "MarketplacepublisherListing",
		SpecType:   reflect.TypeOf(marketplacepublisherv1beta1.ListingSpec{}),
		StatusType: reflect.TypeOf(marketplacepublisherv1beta1.ListingStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "marketplacepublisher.CreateListingDetails",
			},
			{
				SDKStruct: "marketplacepublisher.UpdateListingDetails",
			},
			{
				SDKStruct: "marketplacepublisher.Listing",
			},
			{
				SDKStruct: "marketplacepublisher.ListingCollection",
			},
			{
				SDKStruct: "marketplacepublisher.ListingSummary",
			},
		},
	},
	{
		Name:       "MarketplacepublisherListingRevision",
		SpecType:   reflect.TypeOf(marketplacepublisherv1beta1.ListingRevisionSpec{}),
		StatusType: reflect.TypeOf(marketplacepublisherv1beta1.ListingRevisionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "marketplacepublisher.ListingRevisionCollection",
			},
			{
				SDKStruct: "marketplacepublisher.ListingRevisionSummary",
			},
		},
	},
	{
		Name:       "MarketplacepublisherListingRevisionAttachment",
		SpecType:   reflect.TypeOf(marketplacepublisherv1beta1.ListingRevisionAttachmentSpec{}),
		StatusType: reflect.TypeOf(marketplacepublisherv1beta1.ListingRevisionAttachmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "marketplacepublisher.ListingRevisionAttachmentCollection",
			},
			{
				SDKStruct: "marketplacepublisher.ListingRevisionAttachmentSummary",
			},
		},
	},
	{
		Name:       "MarketplacepublisherListingRevisionNote",
		SpecType:   reflect.TypeOf(marketplacepublisherv1beta1.ListingRevisionNoteSpec{}),
		StatusType: reflect.TypeOf(marketplacepublisherv1beta1.ListingRevisionNoteStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "marketplacepublisher.CreateListingRevisionNoteDetails",
			},
			{
				SDKStruct: "marketplacepublisher.UpdateListingRevisionNoteDetails",
			},
			{
				SDKStruct: "marketplacepublisher.ListingRevisionNote",
			},
			{
				SDKStruct: "marketplacepublisher.ListingRevisionNoteCollection",
			},
			{
				SDKStruct: "marketplacepublisher.ListingRevisionNoteSummary",
			},
		},
	},
	{
		Name:       "MarketplacepublisherListingRevisionPackage",
		SpecType:   reflect.TypeOf(marketplacepublisherv1beta1.ListingRevisionPackageSpec{}),
		StatusType: reflect.TypeOf(marketplacepublisherv1beta1.ListingRevisionPackageStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "marketplacepublisher.CreateListingRevisionPackageDetails",
			},
			{
				SDKStruct: "marketplacepublisher.UpdateListingRevisionPackageDetails",
			},
			{
				SDKStruct: "marketplacepublisher.ListingRevisionPackageCollection",
			},
			{
				SDKStruct: "marketplacepublisher.ListingRevisionPackageSummary",
			},
		},
	},
	{
		Name:       "MarketplacepublisherTerm",
		SpecType:   reflect.TypeOf(marketplacepublisherv1beta1.TermSpec{}),
		StatusType: reflect.TypeOf(marketplacepublisherv1beta1.TermStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "marketplacepublisher.CreateTermDetails",
			},
			{
				SDKStruct: "marketplacepublisher.UpdateTermDetails",
			},
			{
				SDKStruct: "marketplacepublisher.Term",
			},
			{
				SDKStruct: "marketplacepublisher.TermCollection",
			},
			{
				SDKStruct: "marketplacepublisher.TermVersionSummary",
			},
			{
				SDKStruct: "marketplacepublisher.TermSummary",
			},
		},
	},
	{
		Name:       "MarketplacepublisherTermVersion",
		SpecType:   reflect.TypeOf(marketplacepublisherv1beta1.TermVersionSpec{}),
		StatusType: reflect.TypeOf(marketplacepublisherv1beta1.TermVersionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "marketplacepublisher.UpdateTermVersionDetails",
			},
			{
				SDKStruct: "marketplacepublisher.TermVersion",
			},
			{
				SDKStruct: "marketplacepublisher.TermVersionCollection",
			},
			{
				SDKStruct: "marketplacepublisher.TermVersionSummary",
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
		Name:       "OperatoraccesscontrolOperatorControl",
		SpecType:   reflect.TypeOf(operatoraccesscontrolv1beta1.OperatorControlSpec{}),
		StatusType: reflect.TypeOf(operatoraccesscontrolv1beta1.OperatorControlStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "operatoraccesscontrol.CreateOperatorControlDetails",
			},
			{
				SDKStruct: "operatoraccesscontrol.UpdateOperatorControlDetails",
			},
			{
				SDKStruct: "operatoraccesscontrol.OperatorControl",
			},
			{
				SDKStruct: "operatoraccesscontrol.OperatorControlCollection",
			},
			{
				SDKStruct: "operatoraccesscontrol.OperatorControlSummary",
			},
		},
	},
	{
		Name:       "OperatoraccesscontrolOperatorControlAssignment",
		SpecType:   reflect.TypeOf(operatoraccesscontrolv1beta1.OperatorControlAssignmentSpec{}),
		StatusType: reflect.TypeOf(operatoraccesscontrolv1beta1.OperatorControlAssignmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "operatoraccesscontrol.CreateOperatorControlAssignmentDetails",
			},
			{
				SDKStruct: "operatoraccesscontrol.UpdateOperatorControlAssignmentDetails",
			},
			{
				SDKStruct: "operatoraccesscontrol.OperatorControlAssignment",
			},
			{
				SDKStruct: "operatoraccesscontrol.OperatorControlAssignmentCollection",
			},
			{
				SDKStruct: "operatoraccesscontrol.OperatorControlAssignmentSummary",
			},
		},
	},
	{
		Name:       "OpsiAwrHub",
		SpecType:   reflect.TypeOf(opsiv1beta1.AwrHubSpec{}),
		StatusType: reflect.TypeOf(opsiv1beta1.AwrHubStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "opsi.CreateAwrHubDetails",
			},
			{
				SDKStruct: "opsi.UpdateAwrHubDetails",
			},
			{
				SDKStruct: "opsi.AwrHub",
			},
			{
				SDKStruct: "opsi.AwrHubs",
			},
			{
				SDKStruct: "opsi.AwrHubSummary",
			},
		},
	},
	{
		Name:       "OpsiAwrHubSource",
		SpecType:   reflect.TypeOf(opsiv1beta1.AwrHubSourceSpec{}),
		StatusType: reflect.TypeOf(opsiv1beta1.AwrHubSourceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "opsi.CreateAwrHubSourceDetails",
			},
			{
				SDKStruct: "opsi.UpdateAwrHubSourceDetails",
			},
			{
				SDKStruct: "opsi.AwrHubSource",
			},
			{
				SDKStruct: "opsi.AwrHubSources",
			},
			{
				SDKStruct: "opsi.AwrHubSourceSummary",
			},
		},
	},
	{
		Name:       "OpsiChargebackPlan",
		SpecType:   reflect.TypeOf(opsiv1beta1.ChargebackPlanSpec{}),
		StatusType: reflect.TypeOf(opsiv1beta1.ChargebackPlanStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "opsi.UpdateChargebackPlanDetails",
			},
			{
				SDKStruct: "opsi.ChargebackPlanDetails",
			},
			{
				SDKStruct: "opsi.ChargebackPlan",
			},
			{
				SDKStruct: "opsi.ChargebackPlanCollection",
			},
			{
				SDKStruct: "opsi.ChargebackPlanSummary",
			},
		},
	},
	{
		Name:       "OpsiChargebackPlanReport",
		SpecType:   reflect.TypeOf(opsiv1beta1.ChargebackPlanReportSpec{}),
		StatusType: reflect.TypeOf(opsiv1beta1.ChargebackPlanReportStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "opsi.CreateChargebackPlanReportDetails",
			},
			{
				SDKStruct: "opsi.UpdateChargebackPlanReportDetails",
			},
			{
				SDKStruct: "opsi.ChargebackPlanReport",
			},
			{
				SDKStruct: "opsi.ChargebackPlanReportCollection",
			},
			{
				SDKStruct: "opsi.ChargebackPlanReportSummary",
			},
		},
	},
	{
		Name:       "OpsiDatabaseInsight",
		SpecType:   reflect.TypeOf(opsiv1beta1.DatabaseInsightSpec{}),
		StatusType: reflect.TypeOf(opsiv1beta1.DatabaseInsightStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "opsi.DatabaseInsights",
			},
		},
	},
	{
		Name:       "OpsiEnterpriseManagerBridge",
		SpecType:   reflect.TypeOf(opsiv1beta1.EnterpriseManagerBridgeSpec{}),
		StatusType: reflect.TypeOf(opsiv1beta1.EnterpriseManagerBridgeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "opsi.CreateEnterpriseManagerBridgeDetails",
			},
			{
				SDKStruct: "opsi.UpdateEnterpriseManagerBridgeDetails",
			},
			{
				SDKStruct: "opsi.EnterpriseManagerBridge",
			},
			{
				SDKStruct: "opsi.EnterpriseManagerBridgeCollection",
			},
			{
				SDKStruct: "opsi.EnterpriseManagerBridges",
			},
			{
				SDKStruct: "opsi.EnterpriseManagerBridgeSummary",
			},
		},
	},
	{
		Name:       "OpsiExadataInsight",
		SpecType:   reflect.TypeOf(opsiv1beta1.ExadataInsightSpec{}),
		StatusType: reflect.TypeOf(opsiv1beta1.ExadataInsightStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "opsi.ExadataInsights",
			},
		},
	},
	{
		Name:       "OpsiHostInsight",
		SpecType:   reflect.TypeOf(opsiv1beta1.HostInsightSpec{}),
		StatusType: reflect.TypeOf(opsiv1beta1.HostInsightStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "opsi.HostInsights",
			},
		},
	},
	{
		Name:       "OpsiNewsReport",
		SpecType:   reflect.TypeOf(opsiv1beta1.NewsReportSpec{}),
		StatusType: reflect.TypeOf(opsiv1beta1.NewsReportStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "opsi.CreateNewsReportDetails",
			},
			{
				SDKStruct: "opsi.UpdateNewsReportDetails",
			},
			{
				SDKStruct: "opsi.NewsReport",
			},
			{
				SDKStruct: "opsi.NewsReportCollection",
			},
			{
				SDKStruct: "opsi.NewsReports",
			},
			{
				SDKStruct: "opsi.NewsReportSummary",
			},
		},
	},
	{
		Name:       "OpsiOperationsInsightsPrivateEndpoint",
		SpecType:   reflect.TypeOf(opsiv1beta1.OperationsInsightsPrivateEndpointSpec{}),
		StatusType: reflect.TypeOf(opsiv1beta1.OperationsInsightsPrivateEndpointStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "opsi.CreateOperationsInsightsPrivateEndpointDetails",
			},
			{
				SDKStruct: "opsi.UpdateOperationsInsightsPrivateEndpointDetails",
			},
			{
				SDKStruct: "opsi.OperationsInsightsPrivateEndpoint",
			},
			{
				SDKStruct: "opsi.OperationsInsightsPrivateEndpointCollection",
			},
			{
				SDKStruct: "opsi.OperationsInsightsPrivateEndpointSummary",
			},
		},
	},
	{
		Name:       "OpsiOperationsInsightsWarehouse",
		SpecType:   reflect.TypeOf(opsiv1beta1.OperationsInsightsWarehouseSpec{}),
		StatusType: reflect.TypeOf(opsiv1beta1.OperationsInsightsWarehouseStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "opsi.CreateOperationsInsightsWarehouseDetails",
			},
			{
				SDKStruct: "opsi.UpdateOperationsInsightsWarehouseDetails",
			},
			{
				SDKStruct: "opsi.OperationsInsightsWarehouse",
			},
			{
				SDKStruct: "opsi.OperationsInsightsWarehouses",
			},
			{
				SDKStruct: "opsi.OperationsInsightsWarehouseSummary",
			},
		},
	},
	{
		Name:       "OpsiOperationsInsightsWarehouseUser",
		SpecType:   reflect.TypeOf(opsiv1beta1.OperationsInsightsWarehouseUserSpec{}),
		StatusType: reflect.TypeOf(opsiv1beta1.OperationsInsightsWarehouseUserStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "opsi.CreateOperationsInsightsWarehouseUserDetails",
			},
			{
				SDKStruct: "opsi.UpdateOperationsInsightsWarehouseUserDetails",
			},
			{
				SDKStruct: "opsi.OperationsInsightsWarehouseUser",
			},
			{
				SDKStruct: "opsi.OperationsInsightsWarehouseUsers",
			},
			{
				SDKStruct: "opsi.OperationsInsightsWarehouseUserSummary",
			},
		},
	},
	{
		Name:       "OpsiOpsiConfiguration",
		SpecType:   reflect.TypeOf(opsiv1beta1.OpsiConfigurationSpec{}),
		StatusType: reflect.TypeOf(opsiv1beta1.OpsiConfigurationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "opsi.OpsiConfigurations",
			},
		},
	},
	{
		Name:       "OptimizerProfile",
		SpecType:   reflect.TypeOf(optimizerv1beta1.ProfileSpec{}),
		StatusType: reflect.TypeOf(optimizerv1beta1.ProfileStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "optimizer.CreateProfileDetails",
			},
			{
				SDKStruct: "optimizer.UpdateProfileDetails",
			},
			{
				SDKStruct: "optimizer.Profile",
			},
			{
				SDKStruct: "optimizer.ProfileCollection",
			},
			{
				SDKStruct: "optimizer.ProfileSummary",
			},
		},
	},
	{
		Name:       "OsmanagementhubLifecycleEnvironment",
		SpecType:   reflect.TypeOf(osmanagementhubv1beta1.LifecycleEnvironmentSpec{}),
		StatusType: reflect.TypeOf(osmanagementhubv1beta1.LifecycleEnvironmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "osmanagementhub.CreateLifecycleEnvironmentDetails",
			},
			{
				SDKStruct: "osmanagementhub.UpdateLifecycleEnvironmentDetails",
			},
			{
				SDKStruct: "osmanagementhub.LifecycleEnvironmentDetails",
			},
			{
				SDKStruct: "osmanagementhub.LifecycleEnvironment",
			},
			{
				SDKStruct: "osmanagementhub.LifecycleEnvironmentCollection",
			},
			{
				SDKStruct: "osmanagementhub.LifecycleEnvironmentSummary",
			},
		},
	},
	{
		Name:       "OsmanagementhubManagedInstanceGroup",
		SpecType:   reflect.TypeOf(osmanagementhubv1beta1.ManagedInstanceGroupSpec{}),
		StatusType: reflect.TypeOf(osmanagementhubv1beta1.ManagedInstanceGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "osmanagementhub.CreateManagedInstanceGroupDetails",
			},
			{
				SDKStruct: "osmanagementhub.UpdateManagedInstanceGroupDetails",
			},
			{
				SDKStruct: "osmanagementhub.ManagedInstanceGroupDetails",
			},
			{
				SDKStruct: "osmanagementhub.ManagedInstanceGroup",
			},
			{
				SDKStruct: "osmanagementhub.ManagedInstanceGroupCollection",
			},
			{
				SDKStruct: "osmanagementhub.ManagedInstanceGroupSummary",
			},
		},
	},
	{
		Name:       "OsmanagementhubManagementStation",
		SpecType:   reflect.TypeOf(osmanagementhubv1beta1.ManagementStationSpec{}),
		StatusType: reflect.TypeOf(osmanagementhubv1beta1.ManagementStationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "osmanagementhub.CreateManagementStationDetails",
			},
			{
				SDKStruct: "osmanagementhub.UpdateManagementStationDetails",
			},
			{
				SDKStruct: "osmanagementhub.ManagementStationDetails",
			},
			{
				SDKStruct: "osmanagementhub.ManagementStation",
			},
			{
				SDKStruct: "osmanagementhub.ManagementStationCollection",
			},
			{
				SDKStruct: "osmanagementhub.ManagementStationSummary",
			},
		},
	},
	{
		Name:       "OsmanagementhubProfile",
		SpecType:   reflect.TypeOf(osmanagementhubv1beta1.ProfileSpec{}),
		StatusType: reflect.TypeOf(osmanagementhubv1beta1.ProfileStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "osmanagementhub.UpdateProfileDetails",
			},
			{
				SDKStruct: "osmanagementhub.ProfileCollection",
			},
			{
				SDKStruct: "osmanagementhub.ProfileSummary",
			},
		},
	},
	{
		Name:       "OsmanagementhubScheduledJob",
		SpecType:   reflect.TypeOf(osmanagementhubv1beta1.ScheduledJobSpec{}),
		StatusType: reflect.TypeOf(osmanagementhubv1beta1.ScheduledJobStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "osmanagementhub.CreateScheduledJobDetails",
			},
			{
				SDKStruct: "osmanagementhub.UpdateScheduledJobDetails",
			},
			{
				SDKStruct: "osmanagementhub.ScheduledJob",
			},
			{
				SDKStruct: "osmanagementhub.ScheduledJobCollection",
			},
			{
				SDKStruct: "osmanagementhub.ScheduledJobSummary",
			},
		},
	},
	{
		Name:       "OsmanagementhubSoftwareSource",
		SpecType:   reflect.TypeOf(osmanagementhubv1beta1.SoftwareSourceSpec{}),
		StatusType: reflect.TypeOf(osmanagementhubv1beta1.SoftwareSourceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "osmanagementhub.SoftwareSourceDetails",
			},
			{
				SDKStruct: "osmanagementhub.SoftwareSourceCollection",
			},
		},
	},
	{
		Name:       "RecoveryProtectedDatabase",
		SpecType:   reflect.TypeOf(recoveryv1beta1.ProtectedDatabaseSpec{}),
		StatusType: reflect.TypeOf(recoveryv1beta1.ProtectedDatabaseStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "recovery.CreateProtectedDatabaseDetails",
			},
			{
				SDKStruct: "recovery.UpdateProtectedDatabaseDetails",
			},
			{
				SDKStruct: "recovery.ProtectedDatabase",
			},
			{
				SDKStruct: "recovery.ProtectedDatabaseCollection",
			},
			{
				SDKStruct: "recovery.ProtectedDatabaseSummary",
			},
		},
	},
	{
		Name:       "RecoveryProtectionPolicy",
		SpecType:   reflect.TypeOf(recoveryv1beta1.ProtectionPolicySpec{}),
		StatusType: reflect.TypeOf(recoveryv1beta1.ProtectionPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "recovery.CreateProtectionPolicyDetails",
			},
			{
				SDKStruct: "recovery.UpdateProtectionPolicyDetails",
			},
			{
				SDKStruct: "recovery.ProtectionPolicy",
			},
			{
				SDKStruct: "recovery.ProtectionPolicyCollection",
			},
			{
				SDKStruct: "recovery.ProtectionPolicySummary",
			},
		},
	},
	{
		Name:       "RecoveryRecoveryServiceSubnet",
		SpecType:   reflect.TypeOf(recoveryv1beta1.RecoveryServiceSubnetSpec{}),
		StatusType: reflect.TypeOf(recoveryv1beta1.RecoveryServiceSubnetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "recovery.CreateRecoveryServiceSubnetDetails",
			},
			{
				SDKStruct: "recovery.UpdateRecoveryServiceSubnetDetails",
			},
			{
				SDKStruct: "recovery.RecoveryServiceSubnetDetails",
			},
			{
				SDKStruct: "recovery.RecoveryServiceSubnet",
			},
			{
				SDKStruct: "recovery.RecoveryServiceSubnetCollection",
			},
			{
				SDKStruct: "recovery.RecoveryServiceSubnetSummary",
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
	{
		Name:       "ResourceanalyticsResourceAnalyticsInstance",
		SpecType:   reflect.TypeOf(resourceanalyticsv1beta1.ResourceAnalyticsInstanceSpec{}),
		StatusType: reflect.TypeOf(resourceanalyticsv1beta1.ResourceAnalyticsInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "resourceanalytics.CreateResourceAnalyticsInstanceDetails",
			},
			{
				SDKStruct: "resourceanalytics.UpdateResourceAnalyticsInstanceDetails",
			},
			{
				SDKStruct: "resourceanalytics.ResourceAnalyticsInstance",
			},
			{
				SDKStruct: "resourceanalytics.ResourceAnalyticsInstanceCollection",
			},
			{
				SDKStruct: "resourceanalytics.ResourceAnalyticsInstanceSummary",
			},
		},
	},
	{
		Name:       "ResourceanalyticsTenancyAttachment",
		SpecType:   reflect.TypeOf(resourceanalyticsv1beta1.TenancyAttachmentSpec{}),
		StatusType: reflect.TypeOf(resourceanalyticsv1beta1.TenancyAttachmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "resourceanalytics.CreateTenancyAttachmentDetails",
			},
			{
				SDKStruct: "resourceanalytics.UpdateTenancyAttachmentDetails",
			},
			{
				SDKStruct: "resourceanalytics.TenancyAttachment",
			},
			{
				SDKStruct: "resourceanalytics.TenancyAttachmentCollection",
			},
			{
				SDKStruct: "resourceanalytics.TenancyAttachmentSummary",
			},
		},
	},
	{
		Name:       "ResourceschedulerSchedule",
		SpecType:   reflect.TypeOf(resourceschedulerv1beta1.ScheduleSpec{}),
		StatusType: reflect.TypeOf(resourceschedulerv1beta1.ScheduleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "resourcescheduler.CreateScheduleDetails",
			},
			{
				SDKStruct: "resourcescheduler.UpdateScheduleDetails",
			},
			{
				SDKStruct: "resourcescheduler.Schedule",
			},
			{
				SDKStruct: "resourcescheduler.ScheduleCollection",
			},
			{
				SDKStruct: "resourcescheduler.ScheduleSummary",
			},
		},
	},
	{
		Name:       "SchServiceConnector",
		SpecType:   reflect.TypeOf(schv1beta1.ServiceConnectorSpec{}),
		StatusType: reflect.TypeOf(schv1beta1.ServiceConnectorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "sch.CreateServiceConnectorDetails",
			},
			{
				SDKStruct: "sch.UpdateServiceConnectorDetails",
			},
			{
				SDKStruct: "sch.ServiceConnector",
			},
			{
				SDKStruct: "sch.ServiceConnectorCollection",
			},
			{
				SDKStruct: "sch.ServiceConnectorSummary",
			},
		},
	},
	{
		Name:       "SecurityattributeSecurityAttribute",
		SpecType:   reflect.TypeOf(securityattributev1beta1.SecurityAttributeSpec{}),
		StatusType: reflect.TypeOf(securityattributev1beta1.SecurityAttributeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "securityattribute.CreateSecurityAttributeDetails",
			},
			{
				SDKStruct: "securityattribute.UpdateSecurityAttributeDetails",
			},
			{
				SDKStruct: "securityattribute.SecurityAttribute",
			},
			{
				SDKStruct: "securityattribute.SecurityAttributeSummary",
			},
		},
	},
	{
		Name:       "SecurityattributeSecurityAttributeNamespace",
		SpecType:   reflect.TypeOf(securityattributev1beta1.SecurityAttributeNamespaceSpec{}),
		StatusType: reflect.TypeOf(securityattributev1beta1.SecurityAttributeNamespaceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "securityattribute.CreateSecurityAttributeNamespaceDetails",
			},
			{
				SDKStruct: "securityattribute.UpdateSecurityAttributeNamespaceDetails",
			},
			{
				SDKStruct: "securityattribute.SecurityAttributeNamespace",
			},
			{
				SDKStruct: "securityattribute.SecurityAttributeNamespaceSummary",
			},
		},
	},
	{
		Name:       "ServicecatalogPrivateApplication",
		SpecType:   reflect.TypeOf(servicecatalogv1beta1.PrivateApplicationSpec{}),
		StatusType: reflect.TypeOf(servicecatalogv1beta1.PrivateApplicationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "servicecatalog.CreatePrivateApplicationDetails",
			},
			{
				SDKStruct: "servicecatalog.UpdatePrivateApplicationDetails",
			},
			{
				SDKStruct: "servicecatalog.PrivateApplication",
			},
			{
				SDKStruct: "servicecatalog.PrivateApplicationCollection",
			},
			{
				SDKStruct: "servicecatalog.PrivateApplicationSummary",
			},
		},
	},
	{
		Name:       "ServicecatalogServiceCatalog",
		SpecType:   reflect.TypeOf(servicecatalogv1beta1.ServiceCatalogSpec{}),
		StatusType: reflect.TypeOf(servicecatalogv1beta1.ServiceCatalogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "servicecatalog.CreateServiceCatalogDetails",
			},
			{
				SDKStruct: "servicecatalog.UpdateServiceCatalogDetails",
			},
			{
				SDKStruct: "servicecatalog.ServiceCatalog",
			},
			{
				SDKStruct: "servicecatalog.ServiceCatalogCollection",
			},
			{
				SDKStruct: "servicecatalog.ServiceCatalogSummary",
			},
		},
	},
	{
		Name:       "StackmonitoringAlarmCondition",
		SpecType:   reflect.TypeOf(stackmonitoringv1beta1.AlarmConditionSpec{}),
		StatusType: reflect.TypeOf(stackmonitoringv1beta1.AlarmConditionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "stackmonitoring.CreateAlarmConditionDetails",
			},
			{
				SDKStruct: "stackmonitoring.UpdateAlarmConditionDetails",
			},
			{
				SDKStruct: "stackmonitoring.AlarmCondition",
			},
			{
				SDKStruct: "stackmonitoring.AlarmConditionCollection",
			},
			{
				SDKStruct: "stackmonitoring.AlarmConditionSummary",
			},
		},
	},
	{
		Name:       "StackmonitoringBaselineableMetric",
		SpecType:   reflect.TypeOf(stackmonitoringv1beta1.BaselineableMetricSpec{}),
		StatusType: reflect.TypeOf(stackmonitoringv1beta1.BaselineableMetricStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "stackmonitoring.CreateBaselineableMetricDetails",
			},
			{
				SDKStruct: "stackmonitoring.UpdateBaselineableMetricDetails",
			},
			{
				SDKStruct: "stackmonitoring.BaselineableMetric",
			},
			{
				SDKStruct: "stackmonitoring.BaselineableMetricSummary",
			},
		},
	},
	{
		Name:       "StackmonitoringConfig",
		SpecType:   reflect.TypeOf(stackmonitoringv1beta1.ConfigSpec{}),
		StatusType: reflect.TypeOf(stackmonitoringv1beta1.ConfigStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "stackmonitoring.ConfigCollection",
			},
		},
	},
	{
		Name:       "StackmonitoringMaintenanceWindow",
		SpecType:   reflect.TypeOf(stackmonitoringv1beta1.MaintenanceWindowSpec{}),
		StatusType: reflect.TypeOf(stackmonitoringv1beta1.MaintenanceWindowStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "stackmonitoring.CreateMaintenanceWindowDetails",
			},
			{
				SDKStruct: "stackmonitoring.UpdateMaintenanceWindowDetails",
			},
			{
				SDKStruct: "stackmonitoring.MaintenanceWindow",
			},
			{
				SDKStruct: "stackmonitoring.MaintenanceWindowCollection",
			},
			{
				SDKStruct: "stackmonitoring.MaintenanceWindowSummary",
			},
		},
	},
	{
		Name:       "StackmonitoringMetricExtension",
		SpecType:   reflect.TypeOf(stackmonitoringv1beta1.MetricExtensionSpec{}),
		StatusType: reflect.TypeOf(stackmonitoringv1beta1.MetricExtensionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "stackmonitoring.CreateMetricExtensionDetails",
			},
			{
				SDKStruct: "stackmonitoring.UpdateMetricExtensionDetails",
			},
			{
				SDKStruct: "stackmonitoring.MetricExtension",
			},
			{
				SDKStruct: "stackmonitoring.MetricExtensionCollection",
			},
			{
				SDKStruct: "stackmonitoring.MetricExtensionSummary",
			},
		},
	},
	{
		Name:       "StackmonitoringMonitoredResource",
		SpecType:   reflect.TypeOf(stackmonitoringv1beta1.MonitoredResourceSpec{}),
		StatusType: reflect.TypeOf(stackmonitoringv1beta1.MonitoredResourceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "stackmonitoring.CreateMonitoredResourceDetails",
			},
			{
				SDKStruct: "stackmonitoring.UpdateMonitoredResourceDetails",
			},
			{
				SDKStruct: "stackmonitoring.MonitoredResourceDetails",
			},
			{
				SDKStruct: "stackmonitoring.MonitoredResource",
			},
			{
				SDKStruct: "stackmonitoring.MonitoredResourceCollection",
			},
			{
				SDKStruct: "stackmonitoring.MonitoredResourceSummary",
			},
		},
	},
	{
		Name:       "StackmonitoringMonitoredResourceType",
		SpecType:   reflect.TypeOf(stackmonitoringv1beta1.MonitoredResourceTypeSpec{}),
		StatusType: reflect.TypeOf(stackmonitoringv1beta1.MonitoredResourceTypeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "stackmonitoring.CreateMonitoredResourceTypeDetails",
			},
			{
				SDKStruct: "stackmonitoring.UpdateMonitoredResourceTypeDetails",
			},
			{
				SDKStruct: "stackmonitoring.MonitoredResourceType",
			},
			{
				SDKStruct: "stackmonitoring.MonitoredResourceTypeSummary",
			},
		},
	},
	{
		Name:       "StackmonitoringMonitoringTemplate",
		SpecType:   reflect.TypeOf(stackmonitoringv1beta1.MonitoringTemplateSpec{}),
		StatusType: reflect.TypeOf(stackmonitoringv1beta1.MonitoringTemplateStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "stackmonitoring.CreateMonitoringTemplateDetails",
			},
			{
				SDKStruct: "stackmonitoring.UpdateMonitoringTemplateDetails",
			},
			{
				SDKStruct: "stackmonitoring.MonitoringTemplate",
			},
			{
				SDKStruct: "stackmonitoring.MonitoringTemplateCollection",
			},
			{
				SDKStruct: "stackmonitoring.MonitoringTemplateSummary",
			},
		},
	},
	{
		Name:       "StackmonitoringProcessSet",
		SpecType:   reflect.TypeOf(stackmonitoringv1beta1.ProcessSetSpec{}),
		StatusType: reflect.TypeOf(stackmonitoringv1beta1.ProcessSetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "stackmonitoring.CreateProcessSetDetails",
			},
			{
				SDKStruct: "stackmonitoring.UpdateProcessSetDetails",
			},
			{
				SDKStruct: "stackmonitoring.ProcessSet",
			},
			{
				SDKStruct: "stackmonitoring.ProcessSetCollection",
			},
			{
				SDKStruct: "stackmonitoring.ProcessSetSummary",
			},
		},
	},
	{
		Name:       "WaaWebAppAcceleration",
		SpecType:   reflect.TypeOf(waav1beta1.WebAppAccelerationSpec{}),
		StatusType: reflect.TypeOf(waav1beta1.WebAppAccelerationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "waa.UpdateWebAppAccelerationDetails",
			},
			{
				SDKStruct: "waa.WebAppAccelerationCollection",
			},
		},
	},
	{
		Name:       "WaaWebAppAccelerationPolicy",
		SpecType:   reflect.TypeOf(waav1beta1.WebAppAccelerationPolicySpec{}),
		StatusType: reflect.TypeOf(waav1beta1.WebAppAccelerationPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "waa.CreateWebAppAccelerationPolicyDetails",
			},
			{
				SDKStruct: "waa.UpdateWebAppAccelerationPolicyDetails",
			},
			{
				SDKStruct: "waa.WebAppAccelerationPolicy",
			},
			{
				SDKStruct: "waa.WebAppAccelerationPolicyCollection",
			},
			{
				SDKStruct: "waa.WebAppAccelerationPolicySummary",
			},
		},
	},
	{
		Name:       "WaasAddressList",
		SpecType:   reflect.TypeOf(waasv1beta1.AddressListSpec{}),
		StatusType: reflect.TypeOf(waasv1beta1.AddressListStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "waas.CreateAddressListDetails",
			},
			{
				SDKStruct: "waas.UpdateAddressListDetails",
			},
			{
				SDKStruct: "waas.AddressList",
			},
			{
				SDKStruct: "waas.AddressListSummary",
			},
		},
	},
	{
		Name:       "WaasCertificate",
		SpecType:   reflect.TypeOf(waasv1beta1.CertificateSpec{}),
		StatusType: reflect.TypeOf(waasv1beta1.CertificateStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "waas.CreateCertificateDetails",
			},
			{
				SDKStruct: "waas.UpdateCertificateDetails",
			},
			{
				SDKStruct: "waas.Certificate",
			},
			{
				SDKStruct: "waas.CertificateSummary",
			},
		},
	},
	{
		Name:       "WaasCustomProtectionRule",
		SpecType:   reflect.TypeOf(waasv1beta1.CustomProtectionRuleSpec{}),
		StatusType: reflect.TypeOf(waasv1beta1.CustomProtectionRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "waas.CreateCustomProtectionRuleDetails",
			},
			{
				SDKStruct: "waas.UpdateCustomProtectionRuleDetails",
			},
			{
				SDKStruct: "waas.CustomProtectionRule",
			},
			{
				SDKStruct: "waas.CustomProtectionRuleSummary",
			},
		},
	},
	{
		Name:       "WaasHttpRedirect",
		SpecType:   reflect.TypeOf(waasv1beta1.HttpRedirectSpec{}),
		StatusType: reflect.TypeOf(waasv1beta1.HttpRedirectStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "waas.CreateHttpRedirectDetails",
			},
			{
				SDKStruct: "waas.UpdateHttpRedirectDetails",
			},
			{
				SDKStruct: "waas.HttpRedirect",
			},
			{
				SDKStruct: "waas.HttpRedirectSummary",
			},
		},
	},
	{
		Name:       "WaasWaasPolicy",
		SpecType:   reflect.TypeOf(waasv1beta1.WaasPolicySpec{}),
		StatusType: reflect.TypeOf(waasv1beta1.WaasPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "waas.CreateWaasPolicyDetails",
			},
			{
				SDKStruct: "waas.UpdateWaasPolicyDetails",
			},
			{
				SDKStruct: "waas.WaasPolicy",
			},
			{
				SDKStruct: "waas.WaasPolicySummary",
			},
		},
	},
	{
		Name:       "WafNetworkAddressList",
		SpecType:   reflect.TypeOf(wafv1beta1.NetworkAddressListSpec{}),
		StatusType: reflect.TypeOf(wafv1beta1.NetworkAddressListStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "waf.NetworkAddressListCollection",
			},
		},
	},
	{
		Name:       "WafWebAppFirewall",
		SpecType:   reflect.TypeOf(wafv1beta1.WebAppFirewallSpec{}),
		StatusType: reflect.TypeOf(wafv1beta1.WebAppFirewallStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "waf.UpdateWebAppFirewallDetails",
			},
			{
				SDKStruct: "waf.WebAppFirewallCollection",
			},
		},
	},
	{
		Name:       "WafWebAppFirewallPolicy",
		SpecType:   reflect.TypeOf(wafv1beta1.WebAppFirewallPolicySpec{}),
		StatusType: reflect.TypeOf(wafv1beta1.WebAppFirewallPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "waf.CreateWebAppFirewallPolicyDetails",
			},
			{
				SDKStruct: "waf.UpdateWebAppFirewallPolicyDetails",
			},
			{
				SDKStruct: "waf.WebAppFirewallPolicy",
			},
			{
				SDKStruct: "waf.WebAppFirewallPolicyCollection",
			},
			{
				SDKStruct: "waf.WebAppFirewallPolicySummary",
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
