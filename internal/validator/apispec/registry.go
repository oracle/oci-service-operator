package apispec

import (
	"reflect"

	accessgovernancecpv1beta1 "github.com/oracle/oci-service-operator/api/accessgovernancecp/v1beta1"
	admv1beta1 "github.com/oracle/oci-service-operator/api/adm/v1beta1"
	aidataplatformv1beta1 "github.com/oracle/oci-service-operator/api/aidataplatform/v1beta1"
	aidocumentv1beta1 "github.com/oracle/oci-service-operator/api/aidocument/v1beta1"
	ailanguagev1beta1 "github.com/oracle/oci-service-operator/api/ailanguage/v1beta1"
	aispeechv1beta1 "github.com/oracle/oci-service-operator/api/aispeech/v1beta1"
	aivisionv1beta1 "github.com/oracle/oci-service-operator/api/aivision/v1beta1"
	analyticsv1beta1 "github.com/oracle/oci-service-operator/api/analytics/v1beta1"
	announcementsservicev1beta1 "github.com/oracle/oci-service-operator/api/announcementsservice/v1beta1"
	apiaccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/apiaccesscontrol/v1beta1"
	apiplatformv1beta1 "github.com/oracle/oci-service-operator/api/apiplatform/v1beta1"
	apmconfigv1beta1 "github.com/oracle/oci-service-operator/api/apmconfig/v1beta1"
	apmcontrolplanev1beta1 "github.com/oracle/oci-service-operator/api/apmcontrolplane/v1beta1"
	apmsyntheticsv1beta1 "github.com/oracle/oci-service-operator/api/apmsynthetics/v1beta1"
	apmtracesv1beta1 "github.com/oracle/oci-service-operator/api/apmtraces/v1beta1"
	artifactsv1beta1 "github.com/oracle/oci-service-operator/api/artifacts/v1beta1"
	autoscalingv1beta1 "github.com/oracle/oci-service-operator/api/autoscaling/v1beta1"
	bastionv1beta1 "github.com/oracle/oci-service-operator/api/bastion/v1beta1"
	batchv1beta1 "github.com/oracle/oci-service-operator/api/batch/v1beta1"
	bdsv1beta1 "github.com/oracle/oci-service-operator/api/bds/v1beta1"
	blockchainv1beta1 "github.com/oracle/oci-service-operator/api/blockchain/v1beta1"
	budgetv1beta1 "github.com/oracle/oci-service-operator/api/budget/v1beta1"
	capacitymanagementv1beta1 "github.com/oracle/oci-service-operator/api/capacitymanagement/v1beta1"
	certificatesmanagementv1beta1 "github.com/oracle/oci-service-operator/api/certificatesmanagement/v1beta1"
	cloudbridgev1beta1 "github.com/oracle/oci-service-operator/api/cloudbridge/v1beta1"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	cloudmigrationsv1beta1 "github.com/oracle/oci-service-operator/api/cloudmigrations/v1beta1"
	clusterplacementgroupsv1beta1 "github.com/oracle/oci-service-operator/api/clusterplacementgroups/v1beta1"
	computecloudatcustomerv1beta1 "github.com/oracle/oci-service-operator/api/computecloudatcustomer/v1beta1"
	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	containerinstancesv1beta1 "github.com/oracle/oci-service-operator/api/containerinstances/v1beta1"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	dashboardservicev1beta1 "github.com/oracle/oci-service-operator/api/dashboardservice/v1beta1"
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	databasemigrationv1beta1 "github.com/oracle/oci-service-operator/api/databasemigration/v1beta1"
	databasetoolsv1beta1 "github.com/oracle/oci-service-operator/api/databasetools/v1beta1"
	datacatalogv1beta1 "github.com/oracle/oci-service-operator/api/datacatalog/v1beta1"
	dataflowv1beta1 "github.com/oracle/oci-service-operator/api/dataflow/v1beta1"
	dataintegrationv1beta1 "github.com/oracle/oci-service-operator/api/dataintegration/v1beta1"
	datalabelingservicev1beta1 "github.com/oracle/oci-service-operator/api/datalabelingservice/v1beta1"
	datalabelingservicedataplanev1beta1 "github.com/oracle/oci-service-operator/api/datalabelingservicedataplane/v1beta1"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	datasciencev1beta1 "github.com/oracle/oci-service-operator/api/datascience/v1beta1"
	dbmulticloudv1beta1 "github.com/oracle/oci-service-operator/api/dbmulticloud/v1beta1"
	delegateaccesscontrolv1beta1 "github.com/oracle/oci-service-operator/api/delegateaccesscontrol/v1beta1"
	demandsignalv1beta1 "github.com/oracle/oci-service-operator/api/demandsignal/v1beta1"
	desktopsv1beta1 "github.com/oracle/oci-service-operator/api/desktops/v1beta1"
	devopsv1beta1 "github.com/oracle/oci-service-operator/api/devops/v1beta1"
	difv1beta1 "github.com/oracle/oci-service-operator/api/dif/v1beta1"
	dnsv1beta1 "github.com/oracle/oci-service-operator/api/dns/v1beta1"
	emailv1beta1 "github.com/oracle/oci-service-operator/api/email/v1beta1"
	emwarehousev1beta1 "github.com/oracle/oci-service-operator/api/emwarehouse/v1beta1"
	eventsv1beta1 "github.com/oracle/oci-service-operator/api/events/v1beta1"
	filestoragev1beta1 "github.com/oracle/oci-service-operator/api/filestorage/v1beta1"
	fleetappsmanagementv1beta1 "github.com/oracle/oci-service-operator/api/fleetappsmanagement/v1beta1"
	fleetsoftwareupdatev1beta1 "github.com/oracle/oci-service-operator/api/fleetsoftwareupdate/v1beta1"
	functionsv1beta1 "github.com/oracle/oci-service-operator/api/functions/v1beta1"
	fusionappsv1beta1 "github.com/oracle/oci-service-operator/api/fusionapps/v1beta1"
	generativeaiv1beta1 "github.com/oracle/oci-service-operator/api/generativeai/v1beta1"
	goldengatev1beta1 "github.com/oracle/oci-service-operator/api/goldengate/v1beta1"
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
	networkfirewallv1beta1 "github.com/oracle/oci-service-operator/api/networkfirewall/v1beta1"
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
	resourcemanagerv1beta1 "github.com/oracle/oci-service-operator/api/resourcemanager/v1beta1"
	resourceschedulerv1beta1 "github.com/oracle/oci-service-operator/api/resourcescheduler/v1beta1"
	schv1beta1 "github.com/oracle/oci-service-operator/api/sch/v1beta1"
	securityattributev1beta1 "github.com/oracle/oci-service-operator/api/securityattribute/v1beta1"
	servicecatalogv1beta1 "github.com/oracle/oci-service-operator/api/servicecatalog/v1beta1"
	stackmonitoringv1beta1 "github.com/oracle/oci-service-operator/api/stackmonitoring/v1beta1"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	usageapiv1beta1 "github.com/oracle/oci-service-operator/api/usageapi/v1beta1"
	vulnerabilityscanningv1beta1 "github.com/oracle/oci-service-operator/api/vulnerabilityscanning/v1beta1"
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
		Name:       "AccessgovernancecpGovernanceInstance",
		SpecType:   reflect.TypeOf(accessgovernancecpv1beta1.GovernanceInstanceSpec{}),
		StatusType: reflect.TypeOf(accessgovernancecpv1beta1.GovernanceInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "accessgovernancecp.CreateGovernanceInstanceDetails",
			},
			{
				SDKStruct: "accessgovernancecp.UpdateGovernanceInstanceDetails",
			},
			{
				SDKStruct: "accessgovernancecp.GovernanceInstance",
			},
			{
				SDKStruct: "accessgovernancecp.GovernanceInstanceCollection",
			},
			{
				SDKStruct: "accessgovernancecp.GovernanceInstanceSummary",
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
		Name:       "AidataplatformAiDataPlatform",
		SpecType:   reflect.TypeOf(aidataplatformv1beta1.AiDataPlatformSpec{}),
		StatusType: reflect.TypeOf(aidataplatformv1beta1.AiDataPlatformStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "aidataplatform.CreateAiDataPlatformDetails",
			},
			{
				SDKStruct: "aidataplatform.UpdateAiDataPlatformDetails",
			},
			{
				SDKStruct: "aidataplatform.AiDataPlatform",
			},
			{
				SDKStruct: "aidataplatform.AiDataPlatformCollection",
			},
			{
				SDKStruct: "aidataplatform.AiDataPlatformSummary",
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
		Name:       "AnnouncementsserviceAnnouncementSubscription",
		SpecType:   reflect.TypeOf(announcementsservicev1beta1.AnnouncementSubscriptionSpec{}),
		StatusType: reflect.TypeOf(announcementsservicev1beta1.AnnouncementSubscriptionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "announcementsservice.CreateAnnouncementSubscriptionDetails",
			},
			{
				SDKStruct: "announcementsservice.UpdateAnnouncementSubscriptionDetails",
			},
			{
				SDKStruct: "announcementsservice.AnnouncementSubscription",
			},
			{
				SDKStruct: "announcementsservice.AnnouncementSubscriptionCollection",
			},
			{
				SDKStruct: "announcementsservice.AnnouncementSubscriptionSummary",
			},
		},
	},
	{
		Name:       "ApiaccesscontrolPrivilegedApiControl",
		SpecType:   reflect.TypeOf(apiaccesscontrolv1beta1.PrivilegedApiControlSpec{}),
		StatusType: reflect.TypeOf(apiaccesscontrolv1beta1.PrivilegedApiControlStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "apiaccesscontrol.CreatePrivilegedApiControlDetails",
			},
			{
				SDKStruct: "apiaccesscontrol.UpdatePrivilegedApiControlDetails",
			},
			{
				SDKStruct: "apiaccesscontrol.PrivilegedApiControl",
			},
			{
				SDKStruct: "apiaccesscontrol.PrivilegedApiControlCollection",
			},
			{
				SDKStruct: "apiaccesscontrol.PrivilegedApiControlSummary",
			},
		},
	},
	{
		Name:       "ApiplatformApiPlatformInstance",
		SpecType:   reflect.TypeOf(apiplatformv1beta1.ApiPlatformInstanceSpec{}),
		StatusType: reflect.TypeOf(apiplatformv1beta1.ApiPlatformInstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "apiplatform.CreateApiPlatformInstanceDetails",
			},
			{
				SDKStruct: "apiplatform.UpdateApiPlatformInstanceDetails",
			},
			{
				SDKStruct: "apiplatform.ApiPlatformInstance",
			},
			{
				SDKStruct: "apiplatform.ApiPlatformInstanceCollection",
			},
			{
				SDKStruct: "apiplatform.ApiPlatformInstanceSummary",
			},
		},
	},
	{
		Name:       "ApmconfigConfig",
		SpecType:   reflect.TypeOf(apmconfigv1beta1.ConfigSpec{}),
		StatusType: reflect.TypeOf(apmconfigv1beta1.ConfigStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "apmconfig.ConfigCollection",
			},
		},
	},
	{
		Name:       "ApmcontrolplaneApmDomain",
		SpecType:   reflect.TypeOf(apmcontrolplanev1beta1.ApmDomainSpec{}),
		StatusType: reflect.TypeOf(apmcontrolplanev1beta1.ApmDomainStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "apmcontrolplane.CreateApmDomainDetails",
			},
			{
				SDKStruct: "apmcontrolplane.UpdateApmDomainDetails",
			},
			{
				SDKStruct: "apmcontrolplane.ApmDomain",
			},
			{
				SDKStruct: "apmcontrolplane.ApmDomainSummary",
			},
		},
	},
	{
		Name:       "ApmsyntheticsScript",
		SpecType:   reflect.TypeOf(apmsyntheticsv1beta1.ScriptSpec{}),
		StatusType: reflect.TypeOf(apmsyntheticsv1beta1.ScriptStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "apmsynthetics.CreateScriptDetails",
			},
			{
				SDKStruct: "apmsynthetics.UpdateScriptDetails",
			},
			{
				SDKStruct: "apmsynthetics.Script",
			},
			{
				SDKStruct: "apmsynthetics.ScriptCollection",
			},
			{
				SDKStruct: "apmsynthetics.ScriptSummary",
			},
		},
	},
	{
		Name:       "ApmtracesScheduledQuery",
		SpecType:   reflect.TypeOf(apmtracesv1beta1.ScheduledQuerySpec{}),
		StatusType: reflect.TypeOf(apmtracesv1beta1.ScheduledQueryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "apmtraces.CreateScheduledQueryDetails",
			},
			{
				SDKStruct: "apmtraces.UpdateScheduledQueryDetails",
			},
			{
				SDKStruct: "apmtraces.ScheduledQuery",
			},
			{
				SDKStruct: "apmtraces.ScheduledQueryCollection",
			},
			{
				SDKStruct: "apmtraces.ScheduledQuerySummary",
			},
		},
	},
	{
		Name:       "AutoscalingAutoScalingConfiguration",
		SpecType:   reflect.TypeOf(autoscalingv1beta1.AutoScalingConfigurationSpec{}),
		StatusType: reflect.TypeOf(autoscalingv1beta1.AutoScalingConfigurationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "autoscaling.CreateAutoScalingConfigurationDetails",
			},
			{
				SDKStruct: "autoscaling.UpdateAutoScalingConfigurationDetails",
			},
			{
				SDKStruct: "autoscaling.AutoScalingConfiguration",
			},
			{
				SDKStruct: "autoscaling.AutoScalingConfigurationSummary",
			},
		},
	},
	{
		Name:       "AutoscalingAutoScalingPolicy",
		SpecType:   reflect.TypeOf(autoscalingv1beta1.AutoScalingPolicySpec{}),
		StatusType: reflect.TypeOf(autoscalingv1beta1.AutoScalingPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "autoscaling.AutoScalingPolicySummary",
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
		Name:       "BatchBatchContext",
		SpecType:   reflect.TypeOf(batchv1beta1.BatchContextSpec{}),
		StatusType: reflect.TypeOf(batchv1beta1.BatchContextStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "batch.CreateBatchContextDetails",
			},
			{
				SDKStruct: "batch.UpdateBatchContextDetails",
			},
			{
				SDKStruct: "batch.BatchContext",
			},
			{
				SDKStruct: "batch.BatchContextCollection",
			},
			{
				SDKStruct: "batch.BatchContextSummary",
			},
		},
	},
	{
		Name:       "BatchBatchJobPool",
		SpecType:   reflect.TypeOf(batchv1beta1.BatchJobPoolSpec{}),
		StatusType: reflect.TypeOf(batchv1beta1.BatchJobPoolStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "batch.CreateBatchJobPoolDetails",
			},
			{
				SDKStruct: "batch.UpdateBatchJobPoolDetails",
			},
			{
				SDKStruct: "batch.BatchJobPool",
			},
			{
				SDKStruct: "batch.BatchJobPoolCollection",
			},
			{
				SDKStruct: "batch.BatchJobPoolSummary",
			},
		},
	},
	{
		Name:       "BatchBatchTaskEnvironment",
		SpecType:   reflect.TypeOf(batchv1beta1.BatchTaskEnvironmentSpec{}),
		StatusType: reflect.TypeOf(batchv1beta1.BatchTaskEnvironmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "batch.CreateBatchTaskEnvironmentDetails",
			},
			{
				SDKStruct: "batch.UpdateBatchTaskEnvironmentDetails",
			},
			{
				SDKStruct: "batch.BatchTaskEnvironment",
			},
			{
				SDKStruct: "batch.BatchTaskEnvironmentCollection",
			},
			{
				SDKStruct: "batch.BatchTaskEnvironmentSummary",
			},
		},
	},
	{
		Name:       "BatchBatchTaskProfile",
		SpecType:   reflect.TypeOf(batchv1beta1.BatchTaskProfileSpec{}),
		StatusType: reflect.TypeOf(batchv1beta1.BatchTaskProfileStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "batch.CreateBatchTaskProfileDetails",
			},
			{
				SDKStruct: "batch.UpdateBatchTaskProfileDetails",
			},
			{
				SDKStruct: "batch.BatchTaskProfile",
			},
			{
				SDKStruct: "batch.BatchTaskProfileCollection",
			},
			{
				SDKStruct: "batch.BatchTaskProfileSummary",
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
		Name:       "BlockchainBlockchainPlatform",
		SpecType:   reflect.TypeOf(blockchainv1beta1.BlockchainPlatformSpec{}),
		StatusType: reflect.TypeOf(blockchainv1beta1.BlockchainPlatformStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "blockchain.CreateBlockchainPlatformDetails",
			},
			{
				SDKStruct: "blockchain.UpdateBlockchainPlatformDetails",
			},
			{
				SDKStruct: "blockchain.BlockchainPlatform",
			},
			{
				SDKStruct: "blockchain.BlockchainPlatformCollection",
			},
			{
				SDKStruct: "blockchain.BlockchainPlatformSummary",
			},
		},
	},
	{
		Name:       "BlockchainOsn",
		SpecType:   reflect.TypeOf(blockchainv1beta1.OsnSpec{}),
		StatusType: reflect.TypeOf(blockchainv1beta1.OsnStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "blockchain.CreateOsnDetails",
			},
			{
				SDKStruct: "blockchain.UpdateOsnDetails",
			},
			{
				SDKStruct: "blockchain.Osn",
			},
			{
				SDKStruct: "blockchain.OsnCollection",
			},
			{
				SDKStruct: "blockchain.OsnSummary",
			},
		},
	},
	{
		Name:       "BlockchainPeer",
		SpecType:   reflect.TypeOf(blockchainv1beta1.PeerSpec{}),
		StatusType: reflect.TypeOf(blockchainv1beta1.PeerStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "blockchain.CreatePeerDetails",
			},
			{
				SDKStruct: "blockchain.UpdatePeerDetails",
			},
			{
				SDKStruct: "blockchain.Peer",
			},
			{
				SDKStruct: "blockchain.PeerCollection",
			},
			{
				SDKStruct: "blockchain.PeerSummary",
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
		Name:       "CapacitymanagementOccCapacityRequest",
		SpecType:   reflect.TypeOf(capacitymanagementv1beta1.OccCapacityRequestSpec{}),
		StatusType: reflect.TypeOf(capacitymanagementv1beta1.OccCapacityRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "capacitymanagement.CreateOccCapacityRequestDetails",
			},
			{
				SDKStruct: "capacitymanagement.UpdateOccCapacityRequestDetails",
			},
			{
				SDKStruct: "capacitymanagement.OccCapacityRequest",
			},
			{
				SDKStruct: "capacitymanagement.OccCapacityRequestCollection",
			},
			{
				SDKStruct: "capacitymanagement.OccCapacityRequestSummary",
			},
		},
	},
	{
		Name:       "CloudbridgeAgent",
		SpecType:   reflect.TypeOf(cloudbridgev1beta1.AgentSpec{}),
		StatusType: reflect.TypeOf(cloudbridgev1beta1.AgentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudbridge.CreateAgentDetails",
			},
			{
				SDKStruct: "cloudbridge.UpdateAgentDetails",
			},
			{
				SDKStruct: "cloudbridge.Agent",
			},
			{
				SDKStruct: "cloudbridge.AgentCollection",
			},
			{
				SDKStruct: "cloudbridge.AgentSummary",
			},
		},
	},
	{
		Name:       "CloudbridgeAgentDependency",
		SpecType:   reflect.TypeOf(cloudbridgev1beta1.AgentDependencySpec{}),
		StatusType: reflect.TypeOf(cloudbridgev1beta1.AgentDependencyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudbridge.CreateAgentDependencyDetails",
			},
			{
				SDKStruct: "cloudbridge.UpdateAgentDependencyDetails",
			},
			{
				SDKStruct: "cloudbridge.AgentDependency",
			},
			{
				SDKStruct: "cloudbridge.AgentDependencyCollection",
			},
			{
				SDKStruct: "cloudbridge.AgentDependencySummary",
			},
		},
	},
	{
		Name:       "CloudbridgeAsset",
		SpecType:   reflect.TypeOf(cloudbridgev1beta1.AssetSpec{}),
		StatusType: reflect.TypeOf(cloudbridgev1beta1.AssetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudbridge.AssetCollection",
			},
			{
				SDKStruct: "cloudbridge.AssetSummary",
			},
		},
	},
	{
		Name:       "CloudbridgeAssetSource",
		SpecType:   reflect.TypeOf(cloudbridgev1beta1.AssetSourceSpec{}),
		StatusType: reflect.TypeOf(cloudbridgev1beta1.AssetSourceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudbridge.AssetSourceCollection",
			},
		},
	},
	{
		Name:       "CloudbridgeDiscoverySchedule",
		SpecType:   reflect.TypeOf(cloudbridgev1beta1.DiscoveryScheduleSpec{}),
		StatusType: reflect.TypeOf(cloudbridgev1beta1.DiscoveryScheduleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudbridge.CreateDiscoveryScheduleDetails",
			},
			{
				SDKStruct: "cloudbridge.UpdateDiscoveryScheduleDetails",
			},
			{
				SDKStruct: "cloudbridge.DiscoverySchedule",
			},
			{
				SDKStruct: "cloudbridge.DiscoveryScheduleCollection",
			},
			{
				SDKStruct: "cloudbridge.DiscoveryScheduleSummary",
			},
		},
	},
	{
		Name:       "CloudbridgeEnvironment",
		SpecType:   reflect.TypeOf(cloudbridgev1beta1.EnvironmentSpec{}),
		StatusType: reflect.TypeOf(cloudbridgev1beta1.EnvironmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudbridge.CreateEnvironmentDetails",
			},
			{
				SDKStruct: "cloudbridge.UpdateEnvironmentDetails",
			},
			{
				SDKStruct: "cloudbridge.Environment",
			},
			{
				SDKStruct: "cloudbridge.EnvironmentCollection",
			},
			{
				SDKStruct: "cloudbridge.EnvironmentSummary",
			},
		},
	},
	{
		Name:       "CloudbridgeInventory",
		SpecType:   reflect.TypeOf(cloudbridgev1beta1.InventorySpec{}),
		StatusType: reflect.TypeOf(cloudbridgev1beta1.InventoryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudbridge.CreateInventoryDetails",
			},
			{
				SDKStruct: "cloudbridge.UpdateInventoryDetails",
			},
			{
				SDKStruct: "cloudbridge.Inventory",
			},
			{
				SDKStruct: "cloudbridge.InventoryCollection",
			},
			{
				SDKStruct: "cloudbridge.InventorySummary",
			},
		},
	},
	{
		Name:       "CloudguardAdhocQuery",
		SpecType:   reflect.TypeOf(cloudguardv1beta1.AdhocQuerySpec{}),
		StatusType: reflect.TypeOf(cloudguardv1beta1.AdhocQueryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudguard.CreateAdhocQueryDetails",
			},
			{
				SDKStruct: "cloudguard.AdhocQueryDetails",
			},
			{
				SDKStruct: "cloudguard.AdhocQuery",
			},
			{
				SDKStruct: "cloudguard.AdhocQueryCollection",
			},
			{
				SDKStruct: "cloudguard.AdhocQuerySummary",
			},
		},
	},
	{
		Name:       "CloudguardDataMaskRule",
		SpecType:   reflect.TypeOf(cloudguardv1beta1.DataMaskRuleSpec{}),
		StatusType: reflect.TypeOf(cloudguardv1beta1.DataMaskRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudguard.CreateDataMaskRuleDetails",
			},
			{
				SDKStruct: "cloudguard.UpdateDataMaskRuleDetails",
			},
			{
				SDKStruct: "cloudguard.DataMaskRule",
			},
			{
				SDKStruct: "cloudguard.DataMaskRuleCollection",
			},
			{
				SDKStruct: "cloudguard.DataMaskRuleSummary",
			},
		},
	},
	{
		Name:       "CloudguardDataSource",
		SpecType:   reflect.TypeOf(cloudguardv1beta1.DataSourceSpec{}),
		StatusType: reflect.TypeOf(cloudguardv1beta1.DataSourceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudguard.CreateDataSourceDetails",
			},
			{
				SDKStruct: "cloudguard.UpdateDataSourceDetails",
			},
			{
				SDKStruct: "cloudguard.DataSource",
			},
			{
				SDKStruct: "cloudguard.DataSourceCollection",
			},
			{
				SDKStruct: "cloudguard.DataSourceSummary",
			},
		},
	},
	{
		Name:       "CloudguardDetectorRecipe",
		SpecType:   reflect.TypeOf(cloudguardv1beta1.DetectorRecipeSpec{}),
		StatusType: reflect.TypeOf(cloudguardv1beta1.DetectorRecipeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudguard.CreateDetectorRecipeDetails",
			},
			{
				SDKStruct: "cloudguard.UpdateDetectorRecipeDetails",
			},
			{
				SDKStruct: "cloudguard.DetectorRecipe",
			},
			{
				SDKStruct: "cloudguard.DetectorRecipeCollection",
			},
			{
				SDKStruct: "cloudguard.DetectorRecipeSummary",
			},
		},
	},
	{
		Name:       "CloudguardDetectorRecipeDetectorRule",
		SpecType:   reflect.TypeOf(cloudguardv1beta1.DetectorRecipeDetectorRuleSpec{}),
		StatusType: reflect.TypeOf(cloudguardv1beta1.DetectorRecipeDetectorRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudguard.CreateDetectorRecipeDetectorRuleDetails",
			},
			{
				SDKStruct: "cloudguard.UpdateDetectorRecipeDetectorRuleDetails",
			},
			{
				SDKStruct: "cloudguard.DetectorRecipeDetectorRule",
			},
			{
				SDKStruct: "cloudguard.DetectorRecipeDetectorRuleCollection",
			},
			{
				SDKStruct: "cloudguard.DetectorRecipeDetectorRuleSummary",
			},
		},
	},
	{
		Name:       "CloudguardManagedList",
		SpecType:   reflect.TypeOf(cloudguardv1beta1.ManagedListSpec{}),
		StatusType: reflect.TypeOf(cloudguardv1beta1.ManagedListStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudguard.CreateManagedListDetails",
			},
			{
				SDKStruct: "cloudguard.UpdateManagedListDetails",
			},
			{
				SDKStruct: "cloudguard.ManagedList",
			},
			{
				SDKStruct: "cloudguard.ManagedListCollection",
			},
			{
				SDKStruct: "cloudguard.ManagedListSummary",
			},
		},
	},
	{
		Name:       "CloudguardResponderRecipe",
		SpecType:   reflect.TypeOf(cloudguardv1beta1.ResponderRecipeSpec{}),
		StatusType: reflect.TypeOf(cloudguardv1beta1.ResponderRecipeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudguard.CreateResponderRecipeDetails",
			},
			{
				SDKStruct: "cloudguard.UpdateResponderRecipeDetails",
			},
			{
				SDKStruct: "cloudguard.ResponderRecipe",
			},
			{
				SDKStruct: "cloudguard.ResponderRecipeCollection",
			},
			{
				SDKStruct: "cloudguard.ResponderRecipeSummary",
			},
		},
	},
	{
		Name:       "CloudguardSavedQuery",
		SpecType:   reflect.TypeOf(cloudguardv1beta1.SavedQuerySpec{}),
		StatusType: reflect.TypeOf(cloudguardv1beta1.SavedQueryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudguard.CreateSavedQueryDetails",
			},
			{
				SDKStruct: "cloudguard.UpdateSavedQueryDetails",
			},
			{
				SDKStruct: "cloudguard.SavedQuery",
			},
			{
				SDKStruct: "cloudguard.SavedQueryCollection",
			},
			{
				SDKStruct: "cloudguard.SavedQuerySummary",
			},
		},
	},
	{
		Name:       "CloudguardSecurityRecipe",
		SpecType:   reflect.TypeOf(cloudguardv1beta1.SecurityRecipeSpec{}),
		StatusType: reflect.TypeOf(cloudguardv1beta1.SecurityRecipeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudguard.CreateSecurityRecipeDetails",
			},
			{
				SDKStruct: "cloudguard.UpdateSecurityRecipeDetails",
			},
			{
				SDKStruct: "cloudguard.SecurityRecipe",
			},
			{
				SDKStruct: "cloudguard.SecurityRecipeCollection",
			},
			{
				SDKStruct: "cloudguard.SecurityRecipeSummary",
			},
		},
	},
	{
		Name:       "CloudguardSecurityZone",
		SpecType:   reflect.TypeOf(cloudguardv1beta1.SecurityZoneSpec{}),
		StatusType: reflect.TypeOf(cloudguardv1beta1.SecurityZoneStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudguard.CreateSecurityZoneDetails",
			},
			{
				SDKStruct: "cloudguard.UpdateSecurityZoneDetails",
			},
			{
				SDKStruct: "cloudguard.SecurityZone",
			},
			{
				SDKStruct: "cloudguard.SecurityZoneCollection",
			},
			{
				SDKStruct: "cloudguard.SecurityZoneSummary",
			},
		},
	},
	{
		Name:       "CloudguardTarget",
		SpecType:   reflect.TypeOf(cloudguardv1beta1.TargetSpec{}),
		StatusType: reflect.TypeOf(cloudguardv1beta1.TargetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudguard.CreateTargetDetails",
			},
			{
				SDKStruct: "cloudguard.UpdateTargetDetails",
			},
			{
				SDKStruct: "cloudguard.Target",
			},
			{
				SDKStruct: "cloudguard.TargetCollection",
			},
			{
				SDKStruct: "cloudguard.TargetSummary",
			},
		},
	},
	{
		Name:       "CloudguardTargetDetectorRecipe",
		SpecType:   reflect.TypeOf(cloudguardv1beta1.TargetDetectorRecipeSpec{}),
		StatusType: reflect.TypeOf(cloudguardv1beta1.TargetDetectorRecipeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudguard.CreateTargetDetectorRecipeDetails",
			},
			{
				SDKStruct: "cloudguard.UpdateTargetDetectorRecipeDetails",
			},
			{
				SDKStruct: "cloudguard.TargetDetectorRecipe",
			},
			{
				SDKStruct: "cloudguard.TargetDetectorRecipeCollection",
			},
			{
				SDKStruct: "cloudguard.TargetDetectorRecipeSummary",
			},
		},
	},
	{
		Name:       "CloudguardTargetResponderRecipe",
		SpecType:   reflect.TypeOf(cloudguardv1beta1.TargetResponderRecipeSpec{}),
		StatusType: reflect.TypeOf(cloudguardv1beta1.TargetResponderRecipeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudguard.CreateTargetResponderRecipeDetails",
			},
			{
				SDKStruct: "cloudguard.UpdateTargetResponderRecipeDetails",
			},
			{
				SDKStruct: "cloudguard.TargetResponderRecipe",
			},
			{
				SDKStruct: "cloudguard.TargetResponderRecipeCollection",
			},
			{
				SDKStruct: "cloudguard.TargetResponderRecipeSummary",
			},
		},
	},
	{
		Name:       "CloudguardWlpAgent",
		SpecType:   reflect.TypeOf(cloudguardv1beta1.WlpAgentSpec{}),
		StatusType: reflect.TypeOf(cloudguardv1beta1.WlpAgentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudguard.CreateWlpAgentDetails",
			},
			{
				SDKStruct: "cloudguard.UpdateWlpAgentDetails",
			},
			{
				SDKStruct: "cloudguard.WlpAgent",
			},
			{
				SDKStruct: "cloudguard.WlpAgentCollection",
			},
			{
				SDKStruct: "cloudguard.WlpAgentSummary",
			},
		},
	},
	{
		Name:       "CloudmigrationsMigration",
		SpecType:   reflect.TypeOf(cloudmigrationsv1beta1.MigrationSpec{}),
		StatusType: reflect.TypeOf(cloudmigrationsv1beta1.MigrationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudmigrations.CreateMigrationDetails",
			},
			{
				SDKStruct: "cloudmigrations.UpdateMigrationDetails",
			},
			{
				SDKStruct: "cloudmigrations.Migration",
			},
			{
				SDKStruct: "cloudmigrations.MigrationCollection",
			},
			{
				SDKStruct: "cloudmigrations.MigrationSummary",
			},
		},
	},
	{
		Name:       "CloudmigrationsMigrationAsset",
		SpecType:   reflect.TypeOf(cloudmigrationsv1beta1.MigrationAssetSpec{}),
		StatusType: reflect.TypeOf(cloudmigrationsv1beta1.MigrationAssetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudmigrations.CreateMigrationAssetDetails",
			},
			{
				SDKStruct: "cloudmigrations.UpdateMigrationAssetDetails",
			},
			{
				SDKStruct: "cloudmigrations.MigrationAsset",
			},
			{
				SDKStruct: "cloudmigrations.MigrationAssetCollection",
			},
			{
				SDKStruct: "cloudmigrations.MigrationAssetSummary",
			},
		},
	},
	{
		Name:       "CloudmigrationsMigrationPlan",
		SpecType:   reflect.TypeOf(cloudmigrationsv1beta1.MigrationPlanSpec{}),
		StatusType: reflect.TypeOf(cloudmigrationsv1beta1.MigrationPlanStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudmigrations.CreateMigrationPlanDetails",
			},
			{
				SDKStruct: "cloudmigrations.UpdateMigrationPlanDetails",
			},
			{
				SDKStruct: "cloudmigrations.MigrationPlan",
			},
			{
				SDKStruct: "cloudmigrations.MigrationPlanCollection",
			},
			{
				SDKStruct: "cloudmigrations.MigrationPlanSummary",
			},
		},
	},
	{
		Name:       "CloudmigrationsReplicationSchedule",
		SpecType:   reflect.TypeOf(cloudmigrationsv1beta1.ReplicationScheduleSpec{}),
		StatusType: reflect.TypeOf(cloudmigrationsv1beta1.ReplicationScheduleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudmigrations.CreateReplicationScheduleDetails",
			},
			{
				SDKStruct: "cloudmigrations.UpdateReplicationScheduleDetails",
			},
			{
				SDKStruct: "cloudmigrations.ReplicationSchedule",
			},
			{
				SDKStruct: "cloudmigrations.ReplicationScheduleCollection",
			},
			{
				SDKStruct: "cloudmigrations.ReplicationScheduleSummary",
			},
		},
	},
	{
		Name:       "CloudmigrationsTargetAsset",
		SpecType:   reflect.TypeOf(cloudmigrationsv1beta1.TargetAssetSpec{}),
		StatusType: reflect.TypeOf(cloudmigrationsv1beta1.TargetAssetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "cloudmigrations.TargetAssetCollection",
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
		Name:       "ComputecloudatcustomerCccInfrastructure",
		SpecType:   reflect.TypeOf(computecloudatcustomerv1beta1.CccInfrastructureSpec{}),
		StatusType: reflect.TypeOf(computecloudatcustomerv1beta1.CccInfrastructureStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "computecloudatcustomer.CreateCccInfrastructureDetails",
			},
			{
				SDKStruct: "computecloudatcustomer.UpdateCccInfrastructureDetails",
			},
			{
				SDKStruct: "computecloudatcustomer.CccInfrastructure",
			},
			{
				SDKStruct: "computecloudatcustomer.CccInfrastructureCollection",
			},
			{
				SDKStruct: "computecloudatcustomer.CccInfrastructureSummary",
			},
		},
	},
	{
		Name:       "ComputecloudatcustomerCccUpgradeSchedule",
		SpecType:   reflect.TypeOf(computecloudatcustomerv1beta1.CccUpgradeScheduleSpec{}),
		StatusType: reflect.TypeOf(computecloudatcustomerv1beta1.CccUpgradeScheduleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "computecloudatcustomer.CreateCccUpgradeScheduleDetails",
			},
			{
				SDKStruct: "computecloudatcustomer.UpdateCccUpgradeScheduleDetails",
			},
			{
				SDKStruct: "computecloudatcustomer.CccUpgradeSchedule",
			},
			{
				SDKStruct: "computecloudatcustomer.CccUpgradeScheduleCollection",
			},
			{
				SDKStruct: "computecloudatcustomer.CccUpgradeScheduleSummary",
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
		Name:       "DashboardserviceDashboardGroup",
		SpecType:   reflect.TypeOf(dashboardservicev1beta1.DashboardGroupSpec{}),
		StatusType: reflect.TypeOf(dashboardservicev1beta1.DashboardGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dashboardservice.CreateDashboardGroupDetails",
			},
			{
				SDKStruct: "dashboardservice.UpdateDashboardGroupDetails",
			},
			{
				SDKStruct: "dashboardservice.DashboardGroup",
			},
			{
				SDKStruct: "dashboardservice.DashboardGroupCollection",
			},
			{
				SDKStruct: "dashboardservice.DashboardGroupSummary",
			},
		},
	},
	{
		Name:       "DatabasemigrationConnection",
		SpecType:   reflect.TypeOf(databasemigrationv1beta1.ConnectionSpec{}),
		StatusType: reflect.TypeOf(databasemigrationv1beta1.ConnectionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "databasemigration.ConnectionCollection",
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
		Name:       "DatacatalogAttribute",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.AttributeSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.AttributeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateAttributeDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateAttributeDetails",
			},
			{
				SDKStruct: "datacatalog.Attribute",
			},
			{
				SDKStruct: "datacatalog.AttributeCollection",
			},
			{
				SDKStruct: "datacatalog.AttributeSummary",
			},
		},
	},
	{
		Name:       "DatacatalogAttributeTag",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.AttributeTagSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.AttributeTagStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.AttributeTag",
			},
			{
				SDKStruct: "datacatalog.AttributeTagCollection",
			},
			{
				SDKStruct: "datacatalog.AttributeTagSummary",
			},
		},
	},
	{
		Name:       "DatacatalogCatalog",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.CatalogSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.CatalogStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateCatalogDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateCatalogDetails",
			},
			{
				SDKStruct: "datacatalog.Catalog",
			},
			{
				SDKStruct: "datacatalog.CatalogSummary",
			},
		},
	},
	{
		Name:       "DatacatalogCatalogPrivateEndpoint",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.CatalogPrivateEndpointSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.CatalogPrivateEndpointStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateCatalogPrivateEndpointDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateCatalogPrivateEndpointDetails",
			},
			{
				SDKStruct: "datacatalog.CatalogPrivateEndpoint",
			},
			{
				SDKStruct: "datacatalog.CatalogPrivateEndpointSummary",
			},
		},
	},
	{
		Name:       "DatacatalogConnection",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.ConnectionSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.ConnectionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateConnectionDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateConnectionDetails",
			},
			{
				SDKStruct: "datacatalog.Connection",
			},
			{
				SDKStruct: "datacatalog.ConnectionCollection",
			},
			{
				SDKStruct: "datacatalog.ConnectionSummary",
			},
		},
	},
	{
		Name:       "DatacatalogCustomProperty",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.CustomPropertySpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.CustomPropertyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateCustomPropertyDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateCustomPropertyDetails",
			},
			{
				SDKStruct: "datacatalog.CustomProperty",
			},
			{
				SDKStruct: "datacatalog.CustomPropertyCollection",
			},
			{
				SDKStruct: "datacatalog.CustomPropertySummary",
			},
		},
	},
	{
		Name:       "DatacatalogDataAsset",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.DataAssetSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.DataAssetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateDataAssetDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateDataAssetDetails",
			},
			{
				SDKStruct: "datacatalog.DataAsset",
			},
			{
				SDKStruct: "datacatalog.DataAssetCollection",
			},
			{
				SDKStruct: "datacatalog.DataAssetSummary",
			},
		},
	},
	{
		Name:       "DatacatalogDataAssetTag",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.DataAssetTagSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.DataAssetTagStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.DataAssetTag",
			},
			{
				SDKStruct: "datacatalog.DataAssetTagCollection",
			},
			{
				SDKStruct: "datacatalog.DataAssetTagSummary",
			},
		},
	},
	{
		Name:       "DatacatalogEntity",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.EntitySpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.EntityStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateEntityDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateEntityDetails",
			},
			{
				SDKStruct: "datacatalog.Entity",
			},
			{
				SDKStruct: "datacatalog.EntityCollection",
			},
			{
				SDKStruct: "datacatalog.EntitySummary",
			},
		},
	},
	{
		Name:       "DatacatalogEntityTag",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.EntityTagSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.EntityTagStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.EntityTag",
			},
			{
				SDKStruct: "datacatalog.EntityTagCollection",
			},
			{
				SDKStruct: "datacatalog.EntityTagSummary",
			},
		},
	},
	{
		Name:       "DatacatalogFolder",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.FolderSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.FolderStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateFolderDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateFolderDetails",
			},
			{
				SDKStruct: "datacatalog.Folder",
			},
			{
				SDKStruct: "datacatalog.FolderCollection",
			},
			{
				SDKStruct: "datacatalog.FolderSummary",
			},
		},
	},
	{
		Name:       "DatacatalogFolderTag",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.FolderTagSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.FolderTagStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.FolderTag",
			},
			{
				SDKStruct: "datacatalog.FolderTagCollection",
			},
			{
				SDKStruct: "datacatalog.FolderTagSummary",
			},
		},
	},
	{
		Name:       "DatacatalogGlossary",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.GlossarySpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.GlossaryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateGlossaryDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateGlossaryDetails",
			},
			{
				SDKStruct: "datacatalog.Glossary",
			},
			{
				SDKStruct: "datacatalog.GlossaryCollection",
			},
			{
				SDKStruct: "datacatalog.GlossarySummary",
			},
		},
	},
	{
		Name:       "DatacatalogJob",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.JobSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.JobStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateJobDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateJobDetails",
			},
			{
				SDKStruct: "datacatalog.Job",
			},
			{
				SDKStruct: "datacatalog.JobCollection",
			},
			{
				SDKStruct: "datacatalog.JobSummary",
			},
		},
	},
	{
		Name:       "DatacatalogJobDefinition",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.JobDefinitionSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.JobDefinitionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateJobDefinitionDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateJobDefinitionDetails",
			},
			{
				SDKStruct: "datacatalog.JobDefinition",
			},
			{
				SDKStruct: "datacatalog.JobDefinitionCollection",
			},
			{
				SDKStruct: "datacatalog.JobDefinitionSummary",
			},
		},
	},
	{
		Name:       "DatacatalogMetastore",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.MetastoreSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.MetastoreStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateMetastoreDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateMetastoreDetails",
			},
			{
				SDKStruct: "datacatalog.Metastore",
			},
			{
				SDKStruct: "datacatalog.MetastoreSummary",
			},
		},
	},
	{
		Name:       "DatacatalogNamespace",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.NamespaceSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.NamespaceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateNamespaceDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateNamespaceDetails",
			},
			{
				SDKStruct: "datacatalog.Namespace",
			},
			{
				SDKStruct: "datacatalog.NamespaceCollection",
			},
			{
				SDKStruct: "datacatalog.NamespaceSummary",
			},
		},
	},
	{
		Name:       "DatacatalogPattern",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.PatternSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.PatternStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreatePatternDetails",
			},
			{
				SDKStruct: "datacatalog.UpdatePatternDetails",
			},
			{
				SDKStruct: "datacatalog.Pattern",
			},
			{
				SDKStruct: "datacatalog.PatternCollection",
			},
			{
				SDKStruct: "datacatalog.PatternSummary",
			},
		},
	},
	{
		Name:       "DatacatalogTerm",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.TermSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.TermStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateTermDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateTermDetails",
			},
			{
				SDKStruct: "datacatalog.Term",
			},
			{
				SDKStruct: "datacatalog.TermCollection",
			},
			{
				SDKStruct: "datacatalog.TermSummary",
			},
		},
	},
	{
		Name:       "DatacatalogTermRelationship",
		SpecType:   reflect.TypeOf(datacatalogv1beta1.TermRelationshipSpec{}),
		StatusType: reflect.TypeOf(datacatalogv1beta1.TermRelationshipStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datacatalog.CreateTermRelationshipDetails",
			},
			{
				SDKStruct: "datacatalog.UpdateTermRelationshipDetails",
			},
			{
				SDKStruct: "datacatalog.TermRelationship",
			},
			{
				SDKStruct: "datacatalog.TermRelationshipCollection",
			},
			{
				SDKStruct: "datacatalog.TermRelationshipSummary",
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
		Name:       "DataintegrationApplication",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.ApplicationSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.ApplicationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateApplicationDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateApplicationDetails",
			},
			{
				SDKStruct: "dataintegration.ApplicationDetails",
			},
			{
				SDKStruct: "dataintegration.Application",
			},
			{
				SDKStruct: "dataintegration.ApplicationSummary",
			},
		},
	},
	{
		Name:        "DataintegrationApplicationDetailedDescription",
		SpecType:    reflect.TypeOf(dataintegrationv1beta1.ApplicationDetailedDescriptionSpec{}),
		StatusType:  reflect.TypeOf(dataintegrationv1beta1.ApplicationDetailedDescriptionStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:        "DataintegrationConnection",
		SpecType:    reflect.TypeOf(dataintegrationv1beta1.ConnectionSpec{}),
		StatusType:  reflect.TypeOf(dataintegrationv1beta1.ConnectionStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:       "DataintegrationConnectionValidation",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.ConnectionValidationSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.ConnectionValidationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateConnectionValidationDetails",
			},
			{
				SDKStruct: "dataintegration.ConnectionValidation",
			},
			{
				SDKStruct: "dataintegration.ConnectionValidationSummary",
			},
		},
	},
	{
		Name:       "DataintegrationCopyObjectRequest",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.CopyObjectRequestSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.CopyObjectRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateCopyObjectRequestDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateCopyObjectRequestDetails",
			},
			{
				SDKStruct: "dataintegration.CopyObjectRequest",
			},
			{
				SDKStruct: "dataintegration.CopyObjectRequestSummary",
			},
		},
	},
	{
		Name:        "DataintegrationDataAsset",
		SpecType:    reflect.TypeOf(dataintegrationv1beta1.DataAssetSpec{}),
		StatusType:  reflect.TypeOf(dataintegrationv1beta1.DataAssetStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:       "DataintegrationDataFlow",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.DataFlowSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.DataFlowStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateDataFlowDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateDataFlowDetails",
			},
			{
				SDKStruct: "dataintegration.DataFlowDetails",
			},
			{
				SDKStruct: "dataintegration.DataFlow",
			},
			{
				SDKStruct: "dataintegration.DataFlowSummary",
			},
		},
	},
	{
		Name:       "DataintegrationDataFlowValidation",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.DataFlowValidationSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.DataFlowValidationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateDataFlowValidationDetails",
			},
			{
				SDKStruct: "dataintegration.DataFlowValidation",
			},
			{
				SDKStruct: "dataintegration.DataFlowValidationSummary",
			},
		},
	},
	{
		Name:       "DataintegrationDisApplication",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.DisApplicationSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.DisApplicationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateDisApplicationDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateDisApplicationDetails",
			},
			{
				SDKStruct: "dataintegration.DisApplication",
			},
			{
				SDKStruct: "dataintegration.DisApplicationSummary",
			},
		},
	},
	{
		Name:        "DataintegrationDisApplicationDetailedDescription",
		SpecType:    reflect.TypeOf(dataintegrationv1beta1.DisApplicationDetailedDescriptionSpec{}),
		StatusType:  reflect.TypeOf(dataintegrationv1beta1.DisApplicationDetailedDescriptionStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:       "DataintegrationExportRequest",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.ExportRequestSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.ExportRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateExportRequestDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateExportRequestDetails",
			},
			{
				SDKStruct: "dataintegration.ExportRequest",
			},
			{
				SDKStruct: "dataintegration.ExportRequestSummary",
			},
		},
	},
	{
		Name:       "DataintegrationExternalPublication",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.ExternalPublicationSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.ExternalPublicationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateExternalPublicationDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateExternalPublicationDetails",
			},
			{
				SDKStruct: "dataintegration.ExternalPublication",
			},
			{
				SDKStruct: "dataintegration.ExternalPublicationSummary",
			},
		},
	},
	{
		Name:       "DataintegrationExternalPublicationValidation",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.ExternalPublicationValidationSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.ExternalPublicationValidationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateExternalPublicationValidationDetails",
			},
			{
				SDKStruct: "dataintegration.ExternalPublicationValidation",
			},
			{
				SDKStruct: "dataintegration.ExternalPublicationValidationSummary",
			},
		},
	},
	{
		Name:       "DataintegrationFolder",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.FolderSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.FolderStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateFolderDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateFolderDetails",
			},
			{
				SDKStruct: "dataintegration.FolderDetails",
			},
			{
				SDKStruct: "dataintegration.Folder",
			},
			{
				SDKStruct: "dataintegration.FolderSummary",
			},
		},
	},
	{
		Name:       "DataintegrationFunctionLibrary",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.FunctionLibrarySpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.FunctionLibraryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateFunctionLibraryDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateFunctionLibraryDetails",
			},
			{
				SDKStruct: "dataintegration.FunctionLibraryDetails",
			},
			{
				SDKStruct: "dataintegration.FunctionLibrary",
			},
			{
				SDKStruct: "dataintegration.FunctionLibrarySummary",
			},
		},
	},
	{
		Name:       "DataintegrationImportRequest",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.ImportRequestSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.ImportRequestStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateImportRequestDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateImportRequestDetails",
			},
			{
				SDKStruct: "dataintegration.ImportRequest",
			},
			{
				SDKStruct: "dataintegration.ImportRequestSummary",
			},
		},
	},
	{
		Name:       "DataintegrationPatch",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.PatchSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.PatchStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreatePatchDetails",
			},
			{
				SDKStruct: "dataintegration.Patch",
			},
			{
				SDKStruct: "dataintegration.PatchSummary",
			},
		},
	},
	{
		Name:       "DataintegrationPipeline",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.PipelineSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.PipelineStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreatePipelineDetails",
			},
			{
				SDKStruct: "dataintegration.UpdatePipelineDetails",
			},
			{
				SDKStruct: "dataintegration.Pipeline",
			},
			{
				SDKStruct: "dataintegration.PipelineSummary",
			},
		},
	},
	{
		Name:       "DataintegrationPipelineValidation",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.PipelineValidationSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.PipelineValidationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreatePipelineValidationDetails",
			},
			{
				SDKStruct: "dataintegration.PipelineValidation",
			},
			{
				SDKStruct: "dataintegration.PipelineValidationSummary",
			},
		},
	},
	{
		Name:       "DataintegrationProject",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.ProjectSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.ProjectStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateProjectDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateProjectDetails",
			},
			{
				SDKStruct: "dataintegration.ProjectDetails",
			},
			{
				SDKStruct: "dataintegration.Project",
			},
			{
				SDKStruct: "dataintegration.ProjectSummary",
			},
		},
	},
	{
		Name:       "DataintegrationSchedule",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.ScheduleSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.ScheduleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateScheduleDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateScheduleDetails",
			},
			{
				SDKStruct: "dataintegration.Schedule",
			},
			{
				SDKStruct: "dataintegration.ScheduleSummary",
			},
		},
	},
	{
		Name:        "DataintegrationTask",
		SpecType:    reflect.TypeOf(dataintegrationv1beta1.TaskSpec{}),
		StatusType:  reflect.TypeOf(dataintegrationv1beta1.TaskStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:       "DataintegrationTaskRun",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.TaskRunSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.TaskRunStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateTaskRunDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateTaskRunDetails",
			},
			{
				SDKStruct: "dataintegration.TaskRunDetails",
			},
			{
				SDKStruct: "dataintegration.TaskRun",
			},
			{
				SDKStruct: "dataintegration.TaskRunSummary",
			},
		},
	},
	{
		Name:       "DataintegrationTaskSchedule",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.TaskScheduleSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.TaskScheduleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateTaskScheduleDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateTaskScheduleDetails",
			},
			{
				SDKStruct: "dataintegration.TaskSchedule",
			},
			{
				SDKStruct: "dataintegration.TaskScheduleSummary",
			},
		},
	},
	{
		Name:       "DataintegrationTaskValidation",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.TaskValidationSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.TaskValidationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.TaskValidation",
			},
			{
				SDKStruct: "dataintegration.TaskValidationSummary",
			},
		},
	},
	{
		Name:       "DataintegrationUserDefinedFunction",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.UserDefinedFunctionSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.UserDefinedFunctionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateUserDefinedFunctionDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateUserDefinedFunctionDetails",
			},
			{
				SDKStruct: "dataintegration.UserDefinedFunctionDetails",
			},
			{
				SDKStruct: "dataintegration.UserDefinedFunction",
			},
			{
				SDKStruct: "dataintegration.UserDefinedFunctionSummary",
			},
		},
	},
	{
		Name:       "DataintegrationUserDefinedFunctionValidation",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.UserDefinedFunctionValidationSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.UserDefinedFunctionValidationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateUserDefinedFunctionValidationDetails",
			},
			{
				SDKStruct: "dataintegration.UserDefinedFunctionValidation",
			},
			{
				SDKStruct: "dataintegration.UserDefinedFunctionValidationSummary",
			},
		},
	},
	{
		Name:       "DataintegrationWorkspace",
		SpecType:   reflect.TypeOf(dataintegrationv1beta1.WorkspaceSpec{}),
		StatusType: reflect.TypeOf(dataintegrationv1beta1.WorkspaceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dataintegration.CreateWorkspaceDetails",
			},
			{
				SDKStruct: "dataintegration.UpdateWorkspaceDetails",
			},
			{
				SDKStruct: "dataintegration.Workspace",
			},
			{
				SDKStruct: "dataintegration.WorkspaceSummary",
			},
		},
	},
	{
		Name:       "DatalabelingserviceDataset",
		SpecType:   reflect.TypeOf(datalabelingservicev1beta1.DatasetSpec{}),
		StatusType: reflect.TypeOf(datalabelingservicev1beta1.DatasetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datalabelingservice.CreateDatasetDetails",
			},
			{
				SDKStruct: "datalabelingservice.UpdateDatasetDetails",
			},
			{
				SDKStruct: "datalabelingservice.Dataset",
			},
			{
				SDKStruct: "datalabelingservice.DatasetCollection",
			},
			{
				SDKStruct: "datalabelingservice.DatasetSummary",
			},
		},
	},
	{
		Name:       "DatalabelingservicedataplaneAnnotation",
		SpecType:   reflect.TypeOf(datalabelingservicedataplanev1beta1.AnnotationSpec{}),
		StatusType: reflect.TypeOf(datalabelingservicedataplanev1beta1.AnnotationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datalabelingservicedataplane.CreateAnnotationDetails",
			},
			{
				SDKStruct: "datalabelingservicedataplane.UpdateAnnotationDetails",
			},
			{
				SDKStruct: "datalabelingservicedataplane.Annotation",
			},
			{
				SDKStruct: "datalabelingservicedataplane.AnnotationCollection",
			},
			{
				SDKStruct: "datalabelingservicedataplane.AnnotationSummary",
			},
		},
	},
	{
		Name:       "DatalabelingservicedataplaneRecord",
		SpecType:   reflect.TypeOf(datalabelingservicedataplanev1beta1.RecordSpec{}),
		StatusType: reflect.TypeOf(datalabelingservicedataplanev1beta1.RecordStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datalabelingservicedataplane.CreateRecordDetails",
			},
			{
				SDKStruct: "datalabelingservicedataplane.UpdateRecordDetails",
			},
			{
				SDKStruct: "datalabelingservicedataplane.Record",
			},
			{
				SDKStruct: "datalabelingservicedataplane.RecordCollection",
			},
			{
				SDKStruct: "datalabelingservicedataplane.RecordSummary",
			},
		},
	},
	{
		Name:       "DatasafeAlertPolicy",
		SpecType:   reflect.TypeOf(datasafev1beta1.AlertPolicySpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.AlertPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateAlertPolicyDetails",
			},
			{
				SDKStruct: "datasafe.UpdateAlertPolicyDetails",
			},
			{
				SDKStruct: "datasafe.AlertPolicy",
			},
			{
				SDKStruct: "datasafe.AlertPolicyCollection",
			},
			{
				SDKStruct: "datasafe.AlertPolicySummary",
			},
		},
	},
	{
		Name:       "DatasafeAlertPolicyRule",
		SpecType:   reflect.TypeOf(datasafev1beta1.AlertPolicyRuleSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.AlertPolicyRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateAlertPolicyRuleDetails",
			},
			{
				SDKStruct: "datasafe.UpdateAlertPolicyRuleDetails",
			},
			{
				SDKStruct: "datasafe.AlertPolicyRule",
			},
			{
				SDKStruct: "datasafe.AlertPolicyRuleCollection",
			},
			{
				SDKStruct: "datasafe.AlertPolicyRuleSummary",
			},
		},
	},
	{
		Name:       "DatasafeAttributeSet",
		SpecType:   reflect.TypeOf(datasafev1beta1.AttributeSetSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.AttributeSetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateAttributeSetDetails",
			},
			{
				SDKStruct: "datasafe.UpdateAttributeSetDetails",
			},
			{
				SDKStruct: "datasafe.AttributeSet",
			},
			{
				SDKStruct: "datasafe.AttributeSetCollection",
			},
			{
				SDKStruct: "datasafe.AttributeSetSummary",
			},
		},
	},
	{
		Name:       "DatasafeAuditArchiveRetrieval",
		SpecType:   reflect.TypeOf(datasafev1beta1.AuditArchiveRetrievalSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.AuditArchiveRetrievalStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateAuditArchiveRetrievalDetails",
			},
			{
				SDKStruct: "datasafe.UpdateAuditArchiveRetrievalDetails",
			},
			{
				SDKStruct: "datasafe.AuditArchiveRetrieval",
			},
			{
				SDKStruct: "datasafe.AuditArchiveRetrievalCollection",
			},
			{
				SDKStruct: "datasafe.AuditArchiveRetrievalSummary",
			},
		},
	},
	{
		Name:       "DatasafeAuditProfile",
		SpecType:   reflect.TypeOf(datasafev1beta1.AuditProfileSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.AuditProfileStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateAuditProfileDetails",
			},
			{
				SDKStruct: "datasafe.UpdateAuditProfileDetails",
			},
			{
				SDKStruct: "datasafe.AuditProfile",
			},
			{
				SDKStruct: "datasafe.AuditProfileCollection",
			},
			{
				SDKStruct: "datasafe.AuditProfileSummary",
			},
		},
	},
	{
		Name:       "DatasafeDataSafePrivateEndpoint",
		SpecType:   reflect.TypeOf(datasafev1beta1.DataSafePrivateEndpointSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.DataSafePrivateEndpointStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateDataSafePrivateEndpointDetails",
			},
			{
				SDKStruct: "datasafe.UpdateDataSafePrivateEndpointDetails",
			},
			{
				SDKStruct: "datasafe.DataSafePrivateEndpoint",
			},
			{
				SDKStruct: "datasafe.DataSafePrivateEndpointSummary",
			},
		},
	},
	{
		Name:       "DatasafeDiscoveryJob",
		SpecType:   reflect.TypeOf(datasafev1beta1.DiscoveryJobSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.DiscoveryJobStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateDiscoveryJobDetails",
			},
			{
				SDKStruct: "datasafe.DiscoveryJob",
			},
			{
				SDKStruct: "datasafe.DiscoveryJobCollection",
			},
			{
				SDKStruct: "datasafe.DiscoveryJobSummary",
			},
		},
	},
	{
		Name:       "DatasafeLibraryMaskingFormat",
		SpecType:   reflect.TypeOf(datasafev1beta1.LibraryMaskingFormatSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.LibraryMaskingFormatStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateLibraryMaskingFormatDetails",
			},
			{
				SDKStruct: "datasafe.UpdateLibraryMaskingFormatDetails",
			},
			{
				SDKStruct: "datasafe.LibraryMaskingFormat",
			},
			{
				SDKStruct: "datasafe.LibraryMaskingFormatCollection",
			},
			{
				SDKStruct: "datasafe.LibraryMaskingFormatEntry",
			},
			{
				SDKStruct: "datasafe.LibraryMaskingFormatSummary",
			},
		},
	},
	{
		Name:       "DatasafeMaskingColumn",
		SpecType:   reflect.TypeOf(datasafev1beta1.MaskingColumnSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.MaskingColumnStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateMaskingColumnDetails",
			},
			{
				SDKStruct: "datasafe.UpdateMaskingColumnDetails",
			},
			{
				SDKStruct: "datasafe.MaskingColumn",
			},
			{
				SDKStruct: "datasafe.MaskingColumnCollection",
			},
			{
				SDKStruct: "datasafe.MaskingColumnSummary",
			},
		},
	},
	{
		Name:       "DatasafeMaskingPolicy",
		SpecType:   reflect.TypeOf(datasafev1beta1.MaskingPolicySpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.MaskingPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateMaskingPolicyDetails",
			},
			{
				SDKStruct: "datasafe.UpdateMaskingPolicyDetails",
			},
			{
				SDKStruct: "datasafe.MaskingPolicy",
			},
			{
				SDKStruct: "datasafe.MaskingPolicyCollection",
			},
			{
				SDKStruct: "datasafe.MaskingPolicySummary",
			},
		},
	},
	{
		Name:       "DatasafeOnPremConnector",
		SpecType:   reflect.TypeOf(datasafev1beta1.OnPremConnectorSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.OnPremConnectorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateOnPremConnectorDetails",
			},
			{
				SDKStruct: "datasafe.UpdateOnPremConnectorDetails",
			},
			{
				SDKStruct: "datasafe.OnPremConnector",
			},
			{
				SDKStruct: "datasafe.OnPremConnectorSummary",
			},
		},
	},
	{
		Name:       "DatasafePeerTargetDatabase",
		SpecType:   reflect.TypeOf(datasafev1beta1.PeerTargetDatabaseSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.PeerTargetDatabaseStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreatePeerTargetDatabaseDetails",
			},
			{
				SDKStruct: "datasafe.UpdatePeerTargetDatabaseDetails",
			},
			{
				SDKStruct: "datasafe.PeerTargetDatabase",
			},
			{
				SDKStruct: "datasafe.PeerTargetDatabaseCollection",
			},
			{
				SDKStruct: "datasafe.PeerTargetDatabaseSummary",
			},
		},
	},
	{
		Name:       "DatasafeReferentialRelation",
		SpecType:   reflect.TypeOf(datasafev1beta1.ReferentialRelationSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.ReferentialRelationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateReferentialRelationDetails",
			},
			{
				SDKStruct: "datasafe.ReferentialRelation",
			},
			{
				SDKStruct: "datasafe.ReferentialRelationCollection",
			},
			{
				SDKStruct: "datasafe.ReferentialRelationSummary",
			},
		},
	},
	{
		Name:       "DatasafeReportDefinition",
		SpecType:   reflect.TypeOf(datasafev1beta1.ReportDefinitionSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.ReportDefinitionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateReportDefinitionDetails",
			},
			{
				SDKStruct: "datasafe.UpdateReportDefinitionDetails",
			},
			{
				SDKStruct: "datasafe.ReportDefinition",
			},
			{
				SDKStruct: "datasafe.ReportDefinitionCollection",
			},
			{
				SDKStruct: "datasafe.ReportDefinitionSummary",
			},
		},
	},
	{
		Name:       "DatasafeSdmMaskingPolicyDifference",
		SpecType:   reflect.TypeOf(datasafev1beta1.SdmMaskingPolicyDifferenceSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.SdmMaskingPolicyDifferenceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateSdmMaskingPolicyDifferenceDetails",
			},
			{
				SDKStruct: "datasafe.UpdateSdmMaskingPolicyDifferenceDetails",
			},
			{
				SDKStruct: "datasafe.SdmMaskingPolicyDifference",
			},
			{
				SDKStruct: "datasafe.SdmMaskingPolicyDifferenceCollection",
			},
			{
				SDKStruct: "datasafe.SdmMaskingPolicyDifferenceSummary",
			},
		},
	},
	{
		Name:       "DatasafeSecurityAssessment",
		SpecType:   reflect.TypeOf(datasafev1beta1.SecurityAssessmentSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.SecurityAssessmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateSecurityAssessmentDetails",
			},
			{
				SDKStruct: "datasafe.UpdateSecurityAssessmentDetails",
			},
			{
				SDKStruct: "datasafe.SecurityAssessment",
			},
			{
				SDKStruct: "datasafe.SecurityAssessmentSummary",
			},
		},
	},
	{
		Name:       "DatasafeSecurityPolicy",
		SpecType:   reflect.TypeOf(datasafev1beta1.SecurityPolicySpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.SecurityPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateSecurityPolicyDetails",
			},
			{
				SDKStruct: "datasafe.UpdateSecurityPolicyDetails",
			},
			{
				SDKStruct: "datasafe.SecurityPolicy",
			},
			{
				SDKStruct: "datasafe.SecurityPolicyCollection",
			},
			{
				SDKStruct: "datasafe.SecurityPolicySummary",
			},
		},
	},
	{
		Name:       "DatasafeSecurityPolicyConfig",
		SpecType:   reflect.TypeOf(datasafev1beta1.SecurityPolicyConfigSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.SecurityPolicyConfigStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateSecurityPolicyConfigDetails",
			},
			{
				SDKStruct: "datasafe.UpdateSecurityPolicyConfigDetails",
			},
			{
				SDKStruct: "datasafe.SecurityPolicyConfig",
			},
			{
				SDKStruct: "datasafe.SecurityPolicyConfigCollection",
			},
			{
				SDKStruct: "datasafe.SecurityPolicyConfigSummary",
			},
		},
	},
	{
		Name:       "DatasafeSecurityPolicyDeployment",
		SpecType:   reflect.TypeOf(datasafev1beta1.SecurityPolicyDeploymentSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.SecurityPolicyDeploymentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateSecurityPolicyDeploymentDetails",
			},
			{
				SDKStruct: "datasafe.UpdateSecurityPolicyDeploymentDetails",
			},
			{
				SDKStruct: "datasafe.SecurityPolicyDeployment",
			},
			{
				SDKStruct: "datasafe.SecurityPolicyDeploymentCollection",
			},
			{
				SDKStruct: "datasafe.SecurityPolicyDeploymentSummary",
			},
		},
	},
	{
		Name:       "DatasafeSensitiveColumn",
		SpecType:   reflect.TypeOf(datasafev1beta1.SensitiveColumnSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.SensitiveColumnStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateSensitiveColumnDetails",
			},
			{
				SDKStruct: "datasafe.UpdateSensitiveColumnDetails",
			},
			{
				SDKStruct: "datasafe.SensitiveColumn",
			},
			{
				SDKStruct: "datasafe.SensitiveColumnCollection",
			},
			{
				SDKStruct: "datasafe.SensitiveColumnSummary",
			},
		},
	},
	{
		Name:       "DatasafeSensitiveDataModel",
		SpecType:   reflect.TypeOf(datasafev1beta1.SensitiveDataModelSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.SensitiveDataModelStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateSensitiveDataModelDetails",
			},
			{
				SDKStruct: "datasafe.UpdateSensitiveDataModelDetails",
			},
			{
				SDKStruct: "datasafe.SensitiveDataModel",
			},
			{
				SDKStruct: "datasafe.SensitiveDataModelCollection",
			},
			{
				SDKStruct: "datasafe.SensitiveDataModelSummary",
			},
		},
	},
	{
		Name:       "DatasafeSensitiveType",
		SpecType:   reflect.TypeOf(datasafev1beta1.SensitiveTypeSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.SensitiveTypeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.SensitiveTypeCollection",
			},
			{
				SDKStruct: "datasafe.SensitiveTypeSummary",
			},
		},
	},
	{
		Name:       "DatasafeSensitiveTypeGroup",
		SpecType:   reflect.TypeOf(datasafev1beta1.SensitiveTypeGroupSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.SensitiveTypeGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateSensitiveTypeGroupDetails",
			},
			{
				SDKStruct: "datasafe.UpdateSensitiveTypeGroupDetails",
			},
			{
				SDKStruct: "datasafe.SensitiveTypeGroup",
			},
			{
				SDKStruct: "datasafe.SensitiveTypeGroupCollection",
			},
			{
				SDKStruct: "datasafe.SensitiveTypeGroupSummary",
			},
		},
	},
	{
		Name:       "DatasafeSensitiveTypesExport",
		SpecType:   reflect.TypeOf(datasafev1beta1.SensitiveTypesExportSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.SensitiveTypesExportStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateSensitiveTypesExportDetails",
			},
			{
				SDKStruct: "datasafe.UpdateSensitiveTypesExportDetails",
			},
			{
				SDKStruct: "datasafe.SensitiveTypesExport",
			},
			{
				SDKStruct: "datasafe.SensitiveTypesExportCollection",
			},
			{
				SDKStruct: "datasafe.SensitiveTypesExportSummary",
			},
		},
	},
	{
		Name:       "DatasafeSqlCollection",
		SpecType:   reflect.TypeOf(datasafev1beta1.SqlCollectionSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.SqlCollectionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateSqlCollectionDetails",
			},
			{
				SDKStruct: "datasafe.UpdateSqlCollectionDetails",
			},
			{
				SDKStruct: "datasafe.SqlCollection",
			},
			{
				SDKStruct: "datasafe.SqlCollectionCollection",
			},
			{
				SDKStruct: "datasafe.SqlCollectionSummary",
			},
		},
	},
	{
		Name:       "DatasafeTargetAlertPolicyAssociation",
		SpecType:   reflect.TypeOf(datasafev1beta1.TargetAlertPolicyAssociationSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.TargetAlertPolicyAssociationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateTargetAlertPolicyAssociationDetails",
			},
			{
				SDKStruct: "datasafe.UpdateTargetAlertPolicyAssociationDetails",
			},
			{
				SDKStruct: "datasafe.TargetAlertPolicyAssociation",
			},
			{
				SDKStruct: "datasafe.TargetAlertPolicyAssociationCollection",
			},
			{
				SDKStruct: "datasafe.TargetAlertPolicyAssociationSummary",
			},
		},
	},
	{
		Name:       "DatasafeTargetDatabase",
		SpecType:   reflect.TypeOf(datasafev1beta1.TargetDatabaseSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.TargetDatabaseStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateTargetDatabaseDetails",
			},
			{
				SDKStruct: "datasafe.UpdateTargetDatabaseDetails",
			},
			{
				SDKStruct: "datasafe.TargetDatabase",
			},
			{
				SDKStruct: "datasafe.TargetDatabaseSummary",
			},
		},
	},
	{
		Name:       "DatasafeTargetDatabaseGroup",
		SpecType:   reflect.TypeOf(datasafev1beta1.TargetDatabaseGroupSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.TargetDatabaseGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateTargetDatabaseGroupDetails",
			},
			{
				SDKStruct: "datasafe.UpdateTargetDatabaseGroupDetails",
			},
			{
				SDKStruct: "datasafe.TargetDatabaseGroup",
			},
			{
				SDKStruct: "datasafe.TargetDatabaseGroupCollection",
			},
			{
				SDKStruct: "datasafe.TargetDatabaseGroupSummary",
			},
		},
	},
	{
		Name:       "DatasafeUnifiedAuditPolicy",
		SpecType:   reflect.TypeOf(datasafev1beta1.UnifiedAuditPolicySpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.UnifiedAuditPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateUnifiedAuditPolicyDetails",
			},
			{
				SDKStruct: "datasafe.UpdateUnifiedAuditPolicyDetails",
			},
			{
				SDKStruct: "datasafe.UnifiedAuditPolicy",
			},
			{
				SDKStruct: "datasafe.UnifiedAuditPolicyCollection",
			},
			{
				SDKStruct: "datasafe.UnifiedAuditPolicySummary",
			},
		},
	},
	{
		Name:       "DatasafeUserAssessment",
		SpecType:   reflect.TypeOf(datasafev1beta1.UserAssessmentSpec{}),
		StatusType: reflect.TypeOf(datasafev1beta1.UserAssessmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "datasafe.CreateUserAssessmentDetails",
			},
			{
				SDKStruct: "datasafe.UpdateUserAssessmentDetails",
			},
			{
				SDKStruct: "datasafe.UserAssessment",
			},
			{
				SDKStruct: "datasafe.UserAssessmentSummary",
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
		Name:       "DbmulticloudMultiCloudResourceDiscovery",
		SpecType:   reflect.TypeOf(dbmulticloudv1beta1.MultiCloudResourceDiscoverySpec{}),
		StatusType: reflect.TypeOf(dbmulticloudv1beta1.MultiCloudResourceDiscoveryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dbmulticloud.CreateMultiCloudResourceDiscoveryDetails",
			},
			{
				SDKStruct: "dbmulticloud.UpdateMultiCloudResourceDiscoveryDetails",
			},
			{
				SDKStruct: "dbmulticloud.MultiCloudResourceDiscovery",
			},
			{
				SDKStruct: "dbmulticloud.MultiCloudResourceDiscoverySummary",
			},
		},
	},
	{
		Name:       "DbmulticloudOracleDbAwsIdentityConnector",
		SpecType:   reflect.TypeOf(dbmulticloudv1beta1.OracleDbAwsIdentityConnectorSpec{}),
		StatusType: reflect.TypeOf(dbmulticloudv1beta1.OracleDbAwsIdentityConnectorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dbmulticloud.CreateOracleDbAwsIdentityConnectorDetails",
			},
			{
				SDKStruct: "dbmulticloud.UpdateOracleDbAwsIdentityConnectorDetails",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbAwsIdentityConnector",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbAwsIdentityConnectorSummary",
			},
		},
	},
	{
		Name:       "DbmulticloudOracleDbAwsKey",
		SpecType:   reflect.TypeOf(dbmulticloudv1beta1.OracleDbAwsKeySpec{}),
		StatusType: reflect.TypeOf(dbmulticloudv1beta1.OracleDbAwsKeyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dbmulticloud.CreateOracleDbAwsKeyDetails",
			},
			{
				SDKStruct: "dbmulticloud.UpdateOracleDbAwsKeyDetails",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbAwsKey",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbAwsKeySummary",
			},
		},
	},
	{
		Name:       "DbmulticloudOracleDbAzureBlobContainer",
		SpecType:   reflect.TypeOf(dbmulticloudv1beta1.OracleDbAzureBlobContainerSpec{}),
		StatusType: reflect.TypeOf(dbmulticloudv1beta1.OracleDbAzureBlobContainerStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dbmulticloud.CreateOracleDbAzureBlobContainerDetails",
			},
			{
				SDKStruct: "dbmulticloud.UpdateOracleDbAzureBlobContainerDetails",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbAzureBlobContainer",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbAzureBlobContainerSummary",
			},
		},
	},
	{
		Name:       "DbmulticloudOracleDbAzureBlobMount",
		SpecType:   reflect.TypeOf(dbmulticloudv1beta1.OracleDbAzureBlobMountSpec{}),
		StatusType: reflect.TypeOf(dbmulticloudv1beta1.OracleDbAzureBlobMountStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dbmulticloud.CreateOracleDbAzureBlobMountDetails",
			},
			{
				SDKStruct: "dbmulticloud.UpdateOracleDbAzureBlobMountDetails",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbAzureBlobMount",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbAzureBlobMountSummary",
			},
		},
	},
	{
		Name:       "DbmulticloudOracleDbAzureConnector",
		SpecType:   reflect.TypeOf(dbmulticloudv1beta1.OracleDbAzureConnectorSpec{}),
		StatusType: reflect.TypeOf(dbmulticloudv1beta1.OracleDbAzureConnectorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dbmulticloud.CreateOracleDbAzureConnectorDetails",
			},
			{
				SDKStruct: "dbmulticloud.UpdateOracleDbAzureConnectorDetails",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbAzureConnector",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbAzureConnectorSummary",
			},
		},
	},
	{
		Name:       "DbmulticloudOracleDbAzureVault",
		SpecType:   reflect.TypeOf(dbmulticloudv1beta1.OracleDbAzureVaultSpec{}),
		StatusType: reflect.TypeOf(dbmulticloudv1beta1.OracleDbAzureVaultStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dbmulticloud.CreateOracleDbAzureVaultDetails",
			},
			{
				SDKStruct: "dbmulticloud.UpdateOracleDbAzureVaultDetails",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbAzureVault",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbAzureVaultSummary",
			},
		},
	},
	{
		Name:       "DbmulticloudOracleDbAzureVaultAssociation",
		SpecType:   reflect.TypeOf(dbmulticloudv1beta1.OracleDbAzureVaultAssociationSpec{}),
		StatusType: reflect.TypeOf(dbmulticloudv1beta1.OracleDbAzureVaultAssociationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dbmulticloud.CreateOracleDbAzureVaultAssociationDetails",
			},
			{
				SDKStruct: "dbmulticloud.UpdateOracleDbAzureVaultAssociationDetails",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbAzureVaultAssociation",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbAzureVaultAssociationSummary",
			},
		},
	},
	{
		Name:       "DbmulticloudOracleDbGcpIdentityConnector",
		SpecType:   reflect.TypeOf(dbmulticloudv1beta1.OracleDbGcpIdentityConnectorSpec{}),
		StatusType: reflect.TypeOf(dbmulticloudv1beta1.OracleDbGcpIdentityConnectorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dbmulticloud.CreateOracleDbGcpIdentityConnectorDetails",
			},
			{
				SDKStruct: "dbmulticloud.UpdateOracleDbGcpIdentityConnectorDetails",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbGcpIdentityConnector",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbGcpIdentityConnectorSummary",
			},
		},
	},
	{
		Name:       "DbmulticloudOracleDbGcpKeyRing",
		SpecType:   reflect.TypeOf(dbmulticloudv1beta1.OracleDbGcpKeyRingSpec{}),
		StatusType: reflect.TypeOf(dbmulticloudv1beta1.OracleDbGcpKeyRingStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dbmulticloud.CreateOracleDbGcpKeyRingDetails",
			},
			{
				SDKStruct: "dbmulticloud.UpdateOracleDbGcpKeyRingDetails",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbGcpKeyRing",
			},
			{
				SDKStruct: "dbmulticloud.OracleDbGcpKeyRingSummary",
			},
		},
	},
	{
		Name:       "DelegateaccesscontrolDelegationControl",
		SpecType:   reflect.TypeOf(delegateaccesscontrolv1beta1.DelegationControlSpec{}),
		StatusType: reflect.TypeOf(delegateaccesscontrolv1beta1.DelegationControlStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "delegateaccesscontrol.CreateDelegationControlDetails",
			},
			{
				SDKStruct: "delegateaccesscontrol.UpdateDelegationControlDetails",
			},
			{
				SDKStruct: "delegateaccesscontrol.DelegationControl",
			},
			{
				SDKStruct: "delegateaccesscontrol.DelegationControlSummary",
			},
		},
	},
	{
		Name:       "DemandsignalOccDemandSignal",
		SpecType:   reflect.TypeOf(demandsignalv1beta1.OccDemandSignalSpec{}),
		StatusType: reflect.TypeOf(demandsignalv1beta1.OccDemandSignalStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "demandsignal.CreateOccDemandSignalDetails",
			},
			{
				SDKStruct: "demandsignal.UpdateOccDemandSignalDetails",
			},
			{
				SDKStruct: "demandsignal.OccDemandSignal",
			},
			{
				SDKStruct: "demandsignal.OccDemandSignalCollection",
			},
			{
				SDKStruct: "demandsignal.OccDemandSignalSummary",
			},
		},
	},
	{
		Name:       "DesktopsDesktopPool",
		SpecType:   reflect.TypeOf(desktopsv1beta1.DesktopPoolSpec{}),
		StatusType: reflect.TypeOf(desktopsv1beta1.DesktopPoolStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "desktops.CreateDesktopPoolDetails",
			},
			{
				SDKStruct: "desktops.UpdateDesktopPoolDetails",
			},
			{
				SDKStruct: "desktops.DesktopPool",
			},
			{
				SDKStruct: "desktops.DesktopPoolCollection",
			},
			{
				SDKStruct: "desktops.DesktopPoolSummary",
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
		Name:       "DifStack",
		SpecType:   reflect.TypeOf(difv1beta1.StackSpec{}),
		StatusType: reflect.TypeOf(difv1beta1.StackStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "dif.CreateStackDetails",
			},
			{
				SDKStruct: "dif.UpdateStackDetails",
			},
			{
				SDKStruct: "dif.Stack",
			},
			{
				SDKStruct: "dif.StackCollection",
			},
			{
				SDKStruct: "dif.StackSummary",
			},
		},
	},
	{
		Name:       "EmwarehouseEmWarehouse",
		SpecType:   reflect.TypeOf(emwarehousev1beta1.EmWarehouseSpec{}),
		StatusType: reflect.TypeOf(emwarehousev1beta1.EmWarehouseStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "emwarehouse.CreateEmWarehouseDetails",
			},
			{
				SDKStruct: "emwarehouse.UpdateEmWarehouseDetails",
			},
			{
				SDKStruct: "emwarehouse.EmWarehouse",
			},
			{
				SDKStruct: "emwarehouse.EmWarehouseCollection",
			},
			{
				SDKStruct: "emwarehouse.EmWarehouseSummary",
			},
		},
	},
	{
		Name:       "FilestorageExport",
		SpecType:   reflect.TypeOf(filestoragev1beta1.ExportSpec{}),
		StatusType: reflect.TypeOf(filestoragev1beta1.ExportStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "filestorage.CreateExportDetails",
			},
			{
				SDKStruct: "filestorage.UpdateExportDetails",
			},
			{
				SDKStruct: "filestorage.Export",
			},
			{
				SDKStruct: "filestorage.ExportSummary",
			},
		},
	},
	{
		Name:       "FilestorageFileSystem",
		SpecType:   reflect.TypeOf(filestoragev1beta1.FileSystemSpec{}),
		StatusType: reflect.TypeOf(filestoragev1beta1.FileSystemStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "filestorage.CreateFileSystemDetails",
			},
			{
				SDKStruct: "filestorage.UpdateFileSystemDetails",
			},
			{
				SDKStruct: "filestorage.FileSystem",
			},
			{
				SDKStruct: "filestorage.FileSystemSummary",
			},
		},
	},
	{
		Name:       "FilestorageFilesystemSnapshotPolicy",
		SpecType:   reflect.TypeOf(filestoragev1beta1.FilesystemSnapshotPolicySpec{}),
		StatusType: reflect.TypeOf(filestoragev1beta1.FilesystemSnapshotPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "filestorage.CreateFilesystemSnapshotPolicyDetails",
			},
			{
				SDKStruct: "filestorage.UpdateFilesystemSnapshotPolicyDetails",
			},
			{
				SDKStruct: "filestorage.FilesystemSnapshotPolicy",
			},
			{
				SDKStruct: "filestorage.FilesystemSnapshotPolicySummary",
			},
		},
	},
	{
		Name:       "FilestorageMountTarget",
		SpecType:   reflect.TypeOf(filestoragev1beta1.MountTargetSpec{}),
		StatusType: reflect.TypeOf(filestoragev1beta1.MountTargetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "filestorage.CreateMountTargetDetails",
			},
			{
				SDKStruct: "filestorage.UpdateMountTargetDetails",
			},
			{
				SDKStruct: "filestorage.MountTarget",
			},
			{
				SDKStruct: "filestorage.MountTargetSummary",
			},
		},
	},
	{
		Name:       "FilestorageOutboundConnector",
		SpecType:   reflect.TypeOf(filestoragev1beta1.OutboundConnectorSpec{}),
		StatusType: reflect.TypeOf(filestoragev1beta1.OutboundConnectorStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "filestorage.UpdateOutboundConnectorDetails",
			},
		},
	},
	{
		Name:       "FilestorageQuotaRule",
		SpecType:   reflect.TypeOf(filestoragev1beta1.QuotaRuleSpec{}),
		StatusType: reflect.TypeOf(filestoragev1beta1.QuotaRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "filestorage.CreateQuotaRuleDetails",
			},
			{
				SDKStruct: "filestorage.UpdateQuotaRuleDetails",
			},
			{
				SDKStruct: "filestorage.QuotaRule",
			},
			{
				SDKStruct: "filestorage.QuotaRuleSummary",
			},
		},
	},
	{
		Name:       "FilestorageReplication",
		SpecType:   reflect.TypeOf(filestoragev1beta1.ReplicationSpec{}),
		StatusType: reflect.TypeOf(filestoragev1beta1.ReplicationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "filestorage.CreateReplicationDetails",
			},
			{
				SDKStruct: "filestorage.UpdateReplicationDetails",
			},
			{
				SDKStruct: "filestorage.Replication",
			},
			{
				SDKStruct: "filestorage.ReplicationSummary",
			},
		},
	},
	{
		Name:       "FilestorageSnapshot",
		SpecType:   reflect.TypeOf(filestoragev1beta1.SnapshotSpec{}),
		StatusType: reflect.TypeOf(filestoragev1beta1.SnapshotStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "filestorage.CreateSnapshotDetails",
			},
			{
				SDKStruct: "filestorage.UpdateSnapshotDetails",
			},
			{
				SDKStruct: "filestorage.Snapshot",
			},
			{
				SDKStruct: "filestorage.SnapshotSummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementCatalogItem",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.CatalogItemSpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.CatalogItemStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreateCatalogItemDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdateCatalogItemDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.CatalogItem",
			},
			{
				SDKStruct: "fleetappsmanagement.CatalogItemCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.CatalogItemSummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementCompliancePolicyRule",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.CompliancePolicyRuleSpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.CompliancePolicyRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreateCompliancePolicyRuleDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdateCompliancePolicyRuleDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.CompliancePolicyRule",
			},
			{
				SDKStruct: "fleetappsmanagement.CompliancePolicyRuleCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.CompliancePolicyRuleSummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementFleet",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.FleetSpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.FleetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreateFleetDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdateFleetDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.Fleet",
			},
			{
				SDKStruct: "fleetappsmanagement.FleetCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.FleetSummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementFleetCredential",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.FleetCredentialSpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.FleetCredentialStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreateFleetCredentialDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdateFleetCredentialDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.FleetCredential",
			},
			{
				SDKStruct: "fleetappsmanagement.FleetCredentialCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.FleetCredentialSummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementFleetProperty",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.FleetPropertySpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.FleetPropertyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreateFleetPropertyDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdateFleetPropertyDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.FleetProperty",
			},
			{
				SDKStruct: "fleetappsmanagement.FleetPropertyCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.FleetPropertySummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementFleetResource",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.FleetResourceSpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.FleetResourceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreateFleetResourceDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdateFleetResourceDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.FleetResource",
			},
			{
				SDKStruct: "fleetappsmanagement.FleetResourceCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.FleetResourceSummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementMaintenanceWindow",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.MaintenanceWindowSpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.MaintenanceWindowStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreateMaintenanceWindowDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdateMaintenanceWindowDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.MaintenanceWindow",
			},
			{
				SDKStruct: "fleetappsmanagement.MaintenanceWindowCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.MaintenanceWindowSummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementOnboarding",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.OnboardingSpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.OnboardingStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreateOnboardingDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdateOnboardingDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.Onboarding",
			},
			{
				SDKStruct: "fleetappsmanagement.OnboardingCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.OnboardingSummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementPatch",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.PatchSpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.PatchStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreatePatchDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdatePatchDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.Patch",
			},
			{
				SDKStruct: "fleetappsmanagement.PatchCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.PatchSummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementPlatformConfiguration",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.PlatformConfigurationSpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.PlatformConfigurationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreatePlatformConfigurationDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdatePlatformConfigurationDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.PlatformConfiguration",
			},
			{
				SDKStruct: "fleetappsmanagement.PlatformConfigurationCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.PlatformConfigurationSummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementProperty",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.PropertySpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.PropertyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreatePropertyDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdatePropertyDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.Properties",
			},
			{
				SDKStruct: "fleetappsmanagement.Property",
			},
			{
				SDKStruct: "fleetappsmanagement.PropertyCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.PropertySummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementProvision",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.ProvisionSpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.ProvisionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreateProvisionDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdateProvisionDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.Provision",
			},
			{
				SDKStruct: "fleetappsmanagement.ProvisionCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.ProvisionSummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementRunbook",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.RunbookSpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.RunbookStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreateRunbookDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdateRunbookDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.Runbook",
			},
			{
				SDKStruct: "fleetappsmanagement.RunbookCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.RunbookVersionSummary",
			},
			{
				SDKStruct: "fleetappsmanagement.RunbookSummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementRunbookVersion",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.RunbookVersionSpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.RunbookVersionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreateRunbookVersionDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdateRunbookVersionDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.RunbookVersion",
			},
			{
				SDKStruct: "fleetappsmanagement.RunbookVersionCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.RunbookVersionSummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementSchedulerDefinition",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.SchedulerDefinitionSpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.SchedulerDefinitionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreateSchedulerDefinitionDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdateSchedulerDefinitionDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.SchedulerDefinition",
			},
			{
				SDKStruct: "fleetappsmanagement.SchedulerDefinitionCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.SchedulerDefinitionSummary",
			},
		},
	},
	{
		Name:       "FleetappsmanagementTaskRecord",
		SpecType:   reflect.TypeOf(fleetappsmanagementv1beta1.TaskRecordSpec{}),
		StatusType: reflect.TypeOf(fleetappsmanagementv1beta1.TaskRecordStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetappsmanagement.CreateTaskRecordDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.UpdateTaskRecordDetails",
			},
			{
				SDKStruct: "fleetappsmanagement.TaskRecord",
			},
			{
				SDKStruct: "fleetappsmanagement.TaskRecordCollection",
			},
			{
				SDKStruct: "fleetappsmanagement.TaskRecordSummary",
			},
		},
	},
	{
		Name:        "FleetsoftwareupdateFsuAction",
		SpecType:    reflect.TypeOf(fleetsoftwareupdatev1beta1.FsuActionSpec{}),
		StatusType:  reflect.TypeOf(fleetsoftwareupdatev1beta1.FsuActionStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:       "FleetsoftwareupdateFsuCollection",
		SpecType:   reflect.TypeOf(fleetsoftwareupdatev1beta1.FsuCollectionSpec{}),
		StatusType: reflect.TypeOf(fleetsoftwareupdatev1beta1.FsuCollectionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetsoftwareupdate.UpdateFsuCollectionDetails",
			},
		},
	},
	{
		Name:       "FleetsoftwareupdateFsuCycle",
		SpecType:   reflect.TypeOf(fleetsoftwareupdatev1beta1.FsuCycleSpec{}),
		StatusType: reflect.TypeOf(fleetsoftwareupdatev1beta1.FsuCycleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetsoftwareupdate.FsuCycleSummary",
			},
		},
	},
	{
		Name:       "FleetsoftwareupdateFsuDiscovery",
		SpecType:   reflect.TypeOf(fleetsoftwareupdatev1beta1.FsuDiscoverySpec{}),
		StatusType: reflect.TypeOf(fleetsoftwareupdatev1beta1.FsuDiscoveryStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetsoftwareupdate.CreateFsuDiscoveryDetails",
			},
			{
				SDKStruct: "fleetsoftwareupdate.UpdateFsuDiscoveryDetails",
			},
			{
				SDKStruct: "fleetsoftwareupdate.FsuDiscovery",
			},
			{
				SDKStruct: "fleetsoftwareupdate.FsuDiscoverySummary",
			},
		},
	},
	{
		Name:       "FleetsoftwareupdateFsuReadinessCheck",
		SpecType:   reflect.TypeOf(fleetsoftwareupdatev1beta1.FsuReadinessCheckSpec{}),
		StatusType: reflect.TypeOf(fleetsoftwareupdatev1beta1.FsuReadinessCheckStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fleetsoftwareupdate.UpdateFsuReadinessCheckDetails",
			},
			{
				SDKStruct: "fleetsoftwareupdate.FsuReadinessCheckCollection",
			},
			{
				SDKStruct: "fleetsoftwareupdate.FsuReadinessCheckSummary",
			},
		},
	},
	{
		Name:       "FusionappsFusionEnvironment",
		SpecType:   reflect.TypeOf(fusionappsv1beta1.FusionEnvironmentSpec{}),
		StatusType: reflect.TypeOf(fusionappsv1beta1.FusionEnvironmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fusionapps.CreateFusionEnvironmentDetails",
			},
			{
				SDKStruct: "fusionapps.UpdateFusionEnvironmentDetails",
			},
			{
				SDKStruct: "fusionapps.FusionEnvironment",
			},
			{
				SDKStruct: "fusionapps.FusionEnvironmentCollection",
			},
			{
				SDKStruct: "fusionapps.FusionEnvironmentSummary",
			},
		},
	},
	{
		Name:       "FusionappsFusionEnvironmentFamily",
		SpecType:   reflect.TypeOf(fusionappsv1beta1.FusionEnvironmentFamilySpec{}),
		StatusType: reflect.TypeOf(fusionappsv1beta1.FusionEnvironmentFamilyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fusionapps.CreateFusionEnvironmentFamilyDetails",
			},
			{
				SDKStruct: "fusionapps.UpdateFusionEnvironmentFamilyDetails",
			},
			{
				SDKStruct: "fusionapps.FusionEnvironmentFamily",
			},
			{
				SDKStruct: "fusionapps.FusionEnvironmentFamilyCollection",
			},
			{
				SDKStruct: "fusionapps.FusionEnvironmentFamilySummary",
			},
		},
	},
	{
		Name:       "FusionappsRefreshActivity",
		SpecType:   reflect.TypeOf(fusionappsv1beta1.RefreshActivitySpec{}),
		StatusType: reflect.TypeOf(fusionappsv1beta1.RefreshActivityStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fusionapps.CreateRefreshActivityDetails",
			},
			{
				SDKStruct: "fusionapps.UpdateRefreshActivityDetails",
			},
			{
				SDKStruct: "fusionapps.RefreshActivity",
			},
			{
				SDKStruct: "fusionapps.RefreshActivityCollection",
			},
			{
				SDKStruct: "fusionapps.RefreshActivitySummary",
			},
		},
	},
	{
		Name:       "FusionappsServiceAttachment",
		SpecType:   reflect.TypeOf(fusionappsv1beta1.ServiceAttachmentSpec{}),
		StatusType: reflect.TypeOf(fusionappsv1beta1.ServiceAttachmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "fusionapps.CreateServiceAttachmentDetails",
			},
			{
				SDKStruct: "fusionapps.ServiceAttachment",
			},
			{
				SDKStruct: "fusionapps.ServiceAttachmentCollection",
			},
			{
				SDKStruct: "fusionapps.ServiceAttachmentSummary",
			},
		},
	},
	{
		Name:       "GoldengateCertificate",
		SpecType:   reflect.TypeOf(goldengatev1beta1.CertificateSpec{}),
		StatusType: reflect.TypeOf(goldengatev1beta1.CertificateStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "goldengate.CreateCertificateDetails",
			},
			{
				SDKStruct: "goldengate.Certificate",
			},
			{
				SDKStruct: "goldengate.CertificateCollection",
			},
			{
				SDKStruct: "goldengate.CertificateSummary",
			},
		},
	},
	{
		Name:       "GoldengateConnection",
		SpecType:   reflect.TypeOf(goldengatev1beta1.ConnectionSpec{}),
		StatusType: reflect.TypeOf(goldengatev1beta1.ConnectionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "goldengate.ConnectionCollection",
			},
		},
	},
	{
		Name:       "GoldengateConnectionAssignment",
		SpecType:   reflect.TypeOf(goldengatev1beta1.ConnectionAssignmentSpec{}),
		StatusType: reflect.TypeOf(goldengatev1beta1.ConnectionAssignmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "goldengate.CreateConnectionAssignmentDetails",
			},
			{
				SDKStruct: "goldengate.ConnectionAssignment",
			},
			{
				SDKStruct: "goldengate.ConnectionAssignmentCollection",
			},
			{
				SDKStruct: "goldengate.ConnectionAssignmentSummary",
			},
		},
	},
	{
		Name:       "GoldengateDatabaseRegistration",
		SpecType:   reflect.TypeOf(goldengatev1beta1.DatabaseRegistrationSpec{}),
		StatusType: reflect.TypeOf(goldengatev1beta1.DatabaseRegistrationStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "goldengate.CreateDatabaseRegistrationDetails",
			},
			{
				SDKStruct: "goldengate.UpdateDatabaseRegistrationDetails",
			},
			{
				SDKStruct: "goldengate.DatabaseRegistration",
			},
			{
				SDKStruct: "goldengate.DatabaseRegistrationCollection",
			},
			{
				SDKStruct: "goldengate.DatabaseRegistrationSummary",
			},
		},
	},
	{
		Name:       "GoldengateDeployment",
		SpecType:   reflect.TypeOf(goldengatev1beta1.DeploymentSpec{}),
		StatusType: reflect.TypeOf(goldengatev1beta1.DeploymentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "goldengate.CreateDeploymentDetails",
			},
			{
				SDKStruct: "goldengate.UpdateDeploymentDetails",
			},
			{
				SDKStruct: "goldengate.Deployment",
			},
			{
				SDKStruct: "goldengate.DeploymentCollection",
			},
			{
				SDKStruct: "goldengate.DeploymentVersionSummary",
			},
			{
				SDKStruct: "goldengate.DeploymentSummary",
			},
		},
	},
	{
		Name:       "GoldengateDeploymentBackup",
		SpecType:   reflect.TypeOf(goldengatev1beta1.DeploymentBackupSpec{}),
		StatusType: reflect.TypeOf(goldengatev1beta1.DeploymentBackupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "goldengate.CreateDeploymentBackupDetails",
			},
			{
				SDKStruct: "goldengate.UpdateDeploymentBackupDetails",
			},
			{
				SDKStruct: "goldengate.DeploymentBackup",
			},
			{
				SDKStruct: "goldengate.DeploymentBackupCollection",
			},
			{
				SDKStruct: "goldengate.DeploymentBackupSummary",
			},
		},
	},
	{
		Name:       "GoldengatePipeline",
		SpecType:   reflect.TypeOf(goldengatev1beta1.PipelineSpec{}),
		StatusType: reflect.TypeOf(goldengatev1beta1.PipelineStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "goldengate.PipelineCollection",
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
		Name:       "GovernancerulescontrolplaneInclusionCriterion",
		SpecType:   reflect.TypeOf(governancerulescontrolplanev1beta1.InclusionCriterionSpec{}),
		StatusType: reflect.TypeOf(governancerulescontrolplanev1beta1.InclusionCriterionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "governancerulescontrolplane.CreateInclusionCriterionDetails",
			},
			{
				SDKStruct: "governancerulescontrolplane.InclusionCriterion",
			},
			{
				SDKStruct: "governancerulescontrolplane.InclusionCriterionCollection",
			},
			{
				SDKStruct: "governancerulescontrolplane.InclusionCriterionSummary",
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
		Name:       "MarketplaceprivateofferAttachment",
		SpecType:   reflect.TypeOf(marketplaceprivateofferv1beta1.AttachmentSpec{}),
		StatusType: reflect.TypeOf(marketplaceprivateofferv1beta1.AttachmentStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "marketplaceprivateoffer.CreateAttachmentDetails",
			},
			{
				SDKStruct: "marketplaceprivateoffer.Attachment",
			},
			{
				SDKStruct: "marketplaceprivateoffer.AttachmentCollection",
			},
			{
				SDKStruct: "marketplaceprivateoffer.AttachmentSummary",
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
		Name:       "NetworkfirewallAddressList",
		SpecType:   reflect.TypeOf(networkfirewallv1beta1.AddressListSpec{}),
		StatusType: reflect.TypeOf(networkfirewallv1beta1.AddressListStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkfirewall.CreateAddressListDetails",
			},
			{
				SDKStruct: "networkfirewall.AddressList",
			},
			{
				SDKStruct: "networkfirewall.AddressListSummary",
			},
		},
	},
	{
		Name:        "NetworkfirewallApplication",
		SpecType:    reflect.TypeOf(networkfirewallv1beta1.ApplicationSpec{}),
		StatusType:  reflect.TypeOf(networkfirewallv1beta1.ApplicationStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:       "NetworkfirewallApplicationGroup",
		SpecType:   reflect.TypeOf(networkfirewallv1beta1.ApplicationGroupSpec{}),
		StatusType: reflect.TypeOf(networkfirewallv1beta1.ApplicationGroupStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkfirewall.CreateApplicationGroupDetails",
			},
			{
				SDKStruct: "networkfirewall.UpdateApplicationGroupDetails",
			},
			{
				SDKStruct: "networkfirewall.ApplicationGroup",
			},
			{
				SDKStruct: "networkfirewall.ApplicationGroupSummary",
			},
		},
	},
	{
		Name:       "NetworkfirewallDecryptionProfile",
		SpecType:   reflect.TypeOf(networkfirewallv1beta1.DecryptionProfileSpec{}),
		StatusType: reflect.TypeOf(networkfirewallv1beta1.DecryptionProfileStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkfirewall.DecryptionProfileSummary",
			},
		},
	},
	{
		Name:       "NetworkfirewallDecryptionRule",
		SpecType:   reflect.TypeOf(networkfirewallv1beta1.DecryptionRuleSpec{}),
		StatusType: reflect.TypeOf(networkfirewallv1beta1.DecryptionRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkfirewall.CreateDecryptionRuleDetails",
			},
			{
				SDKStruct: "networkfirewall.UpdateDecryptionRuleDetails",
			},
			{
				SDKStruct: "networkfirewall.DecryptionRule",
			},
			{
				SDKStruct: "networkfirewall.DecryptionRuleSummary",
			},
		},
	},
	{
		Name:       "NetworkfirewallMappedSecret",
		SpecType:   reflect.TypeOf(networkfirewallv1beta1.MappedSecretSpec{}),
		StatusType: reflect.TypeOf(networkfirewallv1beta1.MappedSecretStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkfirewall.MappedSecretSummary",
			},
		},
	},
	{
		Name:       "NetworkfirewallNatRule",
		SpecType:   reflect.TypeOf(networkfirewallv1beta1.NatRuleSpec{}),
		StatusType: reflect.TypeOf(networkfirewallv1beta1.NatRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkfirewall.NatRuleCollection",
			},
		},
	},
	{
		Name:       "NetworkfirewallNetworkFirewall",
		SpecType:   reflect.TypeOf(networkfirewallv1beta1.NetworkFirewallSpec{}),
		StatusType: reflect.TypeOf(networkfirewallv1beta1.NetworkFirewallStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkfirewall.CreateNetworkFirewallDetails",
			},
			{
				SDKStruct: "networkfirewall.UpdateNetworkFirewallDetails",
			},
			{
				SDKStruct: "networkfirewall.NetworkFirewall",
			},
			{
				SDKStruct: "networkfirewall.NetworkFirewallCollection",
			},
			{
				SDKStruct: "networkfirewall.NetworkFirewallSummary",
			},
		},
	},
	{
		Name:       "NetworkfirewallNetworkFirewallPolicy",
		SpecType:   reflect.TypeOf(networkfirewallv1beta1.NetworkFirewallPolicySpec{}),
		StatusType: reflect.TypeOf(networkfirewallv1beta1.NetworkFirewallPolicyStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkfirewall.CreateNetworkFirewallPolicyDetails",
			},
			{
				SDKStruct: "networkfirewall.UpdateNetworkFirewallPolicyDetails",
			},
			{
				SDKStruct: "networkfirewall.NetworkFirewallPolicy",
			},
			{
				SDKStruct: "networkfirewall.NetworkFirewallPolicySummary",
			},
		},
	},
	{
		Name:       "NetworkfirewallSecurityRule",
		SpecType:   reflect.TypeOf(networkfirewallv1beta1.SecurityRuleSpec{}),
		StatusType: reflect.TypeOf(networkfirewallv1beta1.SecurityRuleStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkfirewall.CreateSecurityRuleDetails",
			},
			{
				SDKStruct: "networkfirewall.UpdateSecurityRuleDetails",
			},
			{
				SDKStruct: "networkfirewall.SecurityRule",
			},
			{
				SDKStruct: "networkfirewall.SecurityRuleSummary",
			},
		},
	},
	{
		Name:       "NetworkfirewallService",
		SpecType:   reflect.TypeOf(networkfirewallv1beta1.ServiceSpec{}),
		StatusType: reflect.TypeOf(networkfirewallv1beta1.ServiceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkfirewall.ServiceSummary",
			},
		},
	},
	{
		Name:       "NetworkfirewallServiceList",
		SpecType:   reflect.TypeOf(networkfirewallv1beta1.ServiceListSpec{}),
		StatusType: reflect.TypeOf(networkfirewallv1beta1.ServiceListStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkfirewall.CreateServiceListDetails",
			},
			{
				SDKStruct: "networkfirewall.UpdateServiceListDetails",
			},
			{
				SDKStruct: "networkfirewall.ServiceList",
			},
			{
				SDKStruct: "networkfirewall.ServiceListSummary",
			},
		},
	},
	{
		Name:        "NetworkfirewallTunnelInspectionRule",
		SpecType:    reflect.TypeOf(networkfirewallv1beta1.TunnelInspectionRuleSpec{}),
		StatusType:  reflect.TypeOf(networkfirewallv1beta1.TunnelInspectionRuleStatus{}),
		SDKMappings: []SDKMapping{},
	},
	{
		Name:       "NetworkfirewallUrlList",
		SpecType:   reflect.TypeOf(networkfirewallv1beta1.UrlListSpec{}),
		StatusType: reflect.TypeOf(networkfirewallv1beta1.UrlListStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "networkfirewall.CreateUrlListDetails",
			},
			{
				SDKStruct: "networkfirewall.UpdateUrlListDetails",
			},
			{
				SDKStruct: "networkfirewall.UrlList",
			},
			{
				SDKStruct: "networkfirewall.UrlListSummary",
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
		Name:       "ResourceanalyticsMonitoredRegion",
		SpecType:   reflect.TypeOf(resourceanalyticsv1beta1.MonitoredRegionSpec{}),
		StatusType: reflect.TypeOf(resourceanalyticsv1beta1.MonitoredRegionStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "resourceanalytics.CreateMonitoredRegionDetails",
			},
			{
				SDKStruct: "resourceanalytics.MonitoredRegion",
			},
			{
				SDKStruct: "resourceanalytics.MonitoredRegionCollection",
			},
			{
				SDKStruct: "resourceanalytics.MonitoredRegionSummary",
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
		Name:       "ResourcemanagerConfigurationSourceProvider",
		SpecType:   reflect.TypeOf(resourcemanagerv1beta1.ConfigurationSourceProviderSpec{}),
		StatusType: reflect.TypeOf(resourcemanagerv1beta1.ConfigurationSourceProviderStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "resourcemanager.ConfigurationSourceProviderCollection",
			},
		},
	},
	{
		Name:       "ResourcemanagerPrivateEndpoint",
		SpecType:   reflect.TypeOf(resourcemanagerv1beta1.PrivateEndpointSpec{}),
		StatusType: reflect.TypeOf(resourcemanagerv1beta1.PrivateEndpointStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "resourcemanager.CreatePrivateEndpointDetails",
			},
			{
				SDKStruct: "resourcemanager.UpdatePrivateEndpointDetails",
			},
			{
				SDKStruct: "resourcemanager.PrivateEndpoint",
			},
			{
				SDKStruct: "resourcemanager.PrivateEndpointCollection",
			},
			{
				SDKStruct: "resourcemanager.PrivateEndpointSummary",
			},
		},
	},
	{
		Name:       "ResourcemanagerStack",
		SpecType:   reflect.TypeOf(resourcemanagerv1beta1.StackSpec{}),
		StatusType: reflect.TypeOf(resourcemanagerv1beta1.StackStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "resourcemanager.CreateStackDetails",
			},
			{
				SDKStruct: "resourcemanager.UpdateStackDetails",
			},
			{
				SDKStruct: "resourcemanager.Stack",
			},
			{
				SDKStruct: "resourcemanager.StackSummary",
			},
		},
	},
	{
		Name:       "ResourcemanagerTemplate",
		SpecType:   reflect.TypeOf(resourcemanagerv1beta1.TemplateSpec{}),
		StatusType: reflect.TypeOf(resourcemanagerv1beta1.TemplateStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "resourcemanager.CreateTemplateDetails",
			},
			{
				SDKStruct: "resourcemanager.UpdateTemplateDetails",
			},
			{
				SDKStruct: "resourcemanager.Template",
			},
			{
				SDKStruct: "resourcemanager.TemplateSummary",
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
		Name:       "StackmonitoringDiscoveryJob",
		SpecType:   reflect.TypeOf(stackmonitoringv1beta1.DiscoveryJobSpec{}),
		StatusType: reflect.TypeOf(stackmonitoringv1beta1.DiscoveryJobStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "stackmonitoring.CreateDiscoveryJobDetails",
			},
			{
				SDKStruct: "stackmonitoring.DiscoveryJob",
			},
			{
				SDKStruct: "stackmonitoring.DiscoveryJobCollection",
			},
			{
				SDKStruct: "stackmonitoring.DiscoveryJobSummary",
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
		Name:       "VulnerabilityscanningContainerScanRecipe",
		SpecType:   reflect.TypeOf(vulnerabilityscanningv1beta1.ContainerScanRecipeSpec{}),
		StatusType: reflect.TypeOf(vulnerabilityscanningv1beta1.ContainerScanRecipeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "vulnerabilityscanning.CreateContainerScanRecipeDetails",
			},
			{
				SDKStruct: "vulnerabilityscanning.UpdateContainerScanRecipeDetails",
			},
			{
				SDKStruct: "vulnerabilityscanning.ContainerScanRecipe",
			},
			{
				SDKStruct: "vulnerabilityscanning.ContainerScanRecipeSummary",
			},
		},
	},
	{
		Name:       "VulnerabilityscanningContainerScanTarget",
		SpecType:   reflect.TypeOf(vulnerabilityscanningv1beta1.ContainerScanTargetSpec{}),
		StatusType: reflect.TypeOf(vulnerabilityscanningv1beta1.ContainerScanTargetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "vulnerabilityscanning.CreateContainerScanTargetDetails",
			},
			{
				SDKStruct: "vulnerabilityscanning.UpdateContainerScanTargetDetails",
			},
			{
				SDKStruct: "vulnerabilityscanning.ContainerScanTarget",
			},
			{
				SDKStruct: "vulnerabilityscanning.ContainerScanTargetSummary",
			},
		},
	},
	{
		Name:       "VulnerabilityscanningHostScanRecipe",
		SpecType:   reflect.TypeOf(vulnerabilityscanningv1beta1.HostScanRecipeSpec{}),
		StatusType: reflect.TypeOf(vulnerabilityscanningv1beta1.HostScanRecipeStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "vulnerabilityscanning.CreateHostScanRecipeDetails",
			},
			{
				SDKStruct: "vulnerabilityscanning.UpdateHostScanRecipeDetails",
			},
			{
				SDKStruct: "vulnerabilityscanning.HostScanRecipe",
			},
			{
				SDKStruct: "vulnerabilityscanning.HostScanRecipeSummary",
			},
		},
	},
	{
		Name:       "VulnerabilityscanningHostScanTarget",
		SpecType:   reflect.TypeOf(vulnerabilityscanningv1beta1.HostScanTargetSpec{}),
		StatusType: reflect.TypeOf(vulnerabilityscanningv1beta1.HostScanTargetStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct: "vulnerabilityscanning.CreateHostScanTargetDetails",
			},
			{
				SDKStruct: "vulnerabilityscanning.UpdateHostScanTargetDetails",
			},
			{
				SDKStruct: "vulnerabilityscanning.HostScanTarget",
			},
			{
				SDKStruct: "vulnerabilityscanning.HostScanTargetSummary",
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
