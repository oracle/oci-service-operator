package sdk

import (
	"path"
	"reflect"

	"github.com/oracle/oci-go-sdk/v65/accessgovernancecp"
	"github.com/oracle/oci-go-sdk/v65/adm"
	"github.com/oracle/oci-go-sdk/v65/aidataplatform"
	"github.com/oracle/oci-go-sdk/v65/aidocument"
	"github.com/oracle/oci-go-sdk/v65/ailanguage"
	"github.com/oracle/oci-go-sdk/v65/aispeech"
	"github.com/oracle/oci-go-sdk/v65/aivision"
	"github.com/oracle/oci-go-sdk/v65/analytics"
	"github.com/oracle/oci-go-sdk/v65/announcementsservice"
	"github.com/oracle/oci-go-sdk/v65/apiaccesscontrol"
	"github.com/oracle/oci-go-sdk/v65/apiplatform"
	"github.com/oracle/oci-go-sdk/v65/apmconfig"
	"github.com/oracle/oci-go-sdk/v65/apmcontrolplane"
	"github.com/oracle/oci-go-sdk/v65/apmsynthetics"
	"github.com/oracle/oci-go-sdk/v65/apmtraces"
	"github.com/oracle/oci-go-sdk/v65/appmgmtcontrol"
	"github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/autoscaling"
	"github.com/oracle/oci-go-sdk/v65/bastion"
	"github.com/oracle/oci-go-sdk/v65/batch"
	"github.com/oracle/oci-go-sdk/v65/bds"
	"github.com/oracle/oci-go-sdk/v65/blockchain"
	"github.com/oracle/oci-go-sdk/v65/budget"
	"github.com/oracle/oci-go-sdk/v65/capacitymanagement"
	"github.com/oracle/oci-go-sdk/v65/certificatesmanagement"
	"github.com/oracle/oci-go-sdk/v65/cloudbridge"
	"github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/cloudmigrations"
	"github.com/oracle/oci-go-sdk/v65/clusterplacementgroups"
	"github.com/oracle/oci-go-sdk/v65/computecloudatcustomer"
	"github.com/oracle/oci-go-sdk/v65/computeinstanceagent"
	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/oracle/oci-go-sdk/v65/containerinstances"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/dashboardservice"
	"github.com/oracle/oci-go-sdk/v65/database"
	"github.com/oracle/oci-go-sdk/v65/databasemigration"
	"github.com/oracle/oci-go-sdk/v65/databasetools"
	"github.com/oracle/oci-go-sdk/v65/datacatalog"
	"github.com/oracle/oci-go-sdk/v65/dataflow"
	"github.com/oracle/oci-go-sdk/v65/dataintegration"
	"github.com/oracle/oci-go-sdk/v65/datalabelingservice"
	"github.com/oracle/oci-go-sdk/v65/datalabelingservicedataplane"
	"github.com/oracle/oci-go-sdk/v65/datasafe"
	"github.com/oracle/oci-go-sdk/v65/datascience"
	"github.com/oracle/oci-go-sdk/v65/dbmulticloud"
	"github.com/oracle/oci-go-sdk/v65/delegateaccesscontrol"
	"github.com/oracle/oci-go-sdk/v65/demandsignal"
	"github.com/oracle/oci-go-sdk/v65/desktops"
	"github.com/oracle/oci-go-sdk/v65/devops"
	"github.com/oracle/oci-go-sdk/v65/dif"
	"github.com/oracle/oci-go-sdk/v65/disasterrecovery"
	"github.com/oracle/oci-go-sdk/v65/distributeddatabase"
	"github.com/oracle/oci-go-sdk/v65/dns"
	"github.com/oracle/oci-go-sdk/v65/email"
	"github.com/oracle/oci-go-sdk/v65/emwarehouse"
	"github.com/oracle/oci-go-sdk/v65/events"
	"github.com/oracle/oci-go-sdk/v65/filestorage"
	"github.com/oracle/oci-go-sdk/v65/fleetappsmanagement"
	"github.com/oracle/oci-go-sdk/v65/fleetsoftwareupdate"
	"github.com/oracle/oci-go-sdk/v65/functions"
	"github.com/oracle/oci-go-sdk/v65/fusionapps"
	"github.com/oracle/oci-go-sdk/v65/gdp"
	"github.com/oracle/oci-go-sdk/v65/generativeai"
	"github.com/oracle/oci-go-sdk/v65/generativeaiagent"
	"github.com/oracle/oci-go-sdk/v65/generativeaiagentruntime"
	"github.com/oracle/oci-go-sdk/v65/generativeaidata"
	"github.com/oracle/oci-go-sdk/v65/goldengate"
	"github.com/oracle/oci-go-sdk/v65/governancerulescontrolplane"
	"github.com/oracle/oci-go-sdk/v65/healthchecks"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/oracle/oci-go-sdk/v65/integration"
	"github.com/oracle/oci-go-sdk/v65/iot"
	"github.com/oracle/oci-go-sdk/v65/jms"
	"github.com/oracle/oci-go-sdk/v65/jmsjavadownloads"
	"github.com/oracle/oci-go-sdk/v65/keymanagement"
	"github.com/oracle/oci-go-sdk/v65/licensemanager"
	"github.com/oracle/oci-go-sdk/v65/limits"
	"github.com/oracle/oci-go-sdk/v65/limitsincrease"
	"github.com/oracle/oci-go-sdk/v65/loadbalancer"
	"github.com/oracle/oci-go-sdk/v65/lockbox"
	"github.com/oracle/oci-go-sdk/v65/loganalytics"
	"github.com/oracle/oci-go-sdk/v65/logging"
	"github.com/oracle/oci-go-sdk/v65/lustrefilestorage"
	"github.com/oracle/oci-go-sdk/v65/managedkafka"
	"github.com/oracle/oci-go-sdk/v65/managementagent"
	"github.com/oracle/oci-go-sdk/v65/managementdashboard"
	"github.com/oracle/oci-go-sdk/v65/marketplace"
	"github.com/oracle/oci-go-sdk/v65/marketplaceprivateoffer"
	"github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	"github.com/oracle/oci-go-sdk/v65/mediaservices"
	"github.com/oracle/oci-go-sdk/v65/mngdmac"
	"github.com/oracle/oci-go-sdk/v65/monitoring"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	"github.com/oracle/oci-go-sdk/v65/networkfirewall"
	"github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	"github.com/oracle/oci-go-sdk/v65/nosql"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
	"github.com/oracle/oci-go-sdk/v65/oce"
	"github.com/oracle/oci-go-sdk/v65/ocvp"
	"github.com/oracle/oci-go-sdk/v65/oda"
	"github.com/oracle/oci-go-sdk/v65/onesubscription"
	"github.com/oracle/oci-go-sdk/v65/ons"
	"github.com/oracle/oci-go-sdk/v65/opa"
	"github.com/oracle/oci-go-sdk/v65/opensearch"
	"github.com/oracle/oci-go-sdk/v65/operatoraccesscontrol"
	"github.com/oracle/oci-go-sdk/v65/opsi"
	"github.com/oracle/oci-go-sdk/v65/optimizer"
	"github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	"github.com/oracle/oci-go-sdk/v65/osubsubscription"
	"github.com/oracle/oci-go-sdk/v65/psa"
	"github.com/oracle/oci-go-sdk/v65/psql"
	"github.com/oracle/oci-go-sdk/v65/queue"
	"github.com/oracle/oci-go-sdk/v65/recovery"
	"github.com/oracle/oci-go-sdk/v65/redis"
	"github.com/oracle/oci-go-sdk/v65/resourceanalytics"
	"github.com/oracle/oci-go-sdk/v65/resourcemanager"
	"github.com/oracle/oci-go-sdk/v65/resourcescheduler"
	"github.com/oracle/oci-go-sdk/v65/rover"
	"github.com/oracle/oci-go-sdk/v65/sch"
	"github.com/oracle/oci-go-sdk/v65/securityattribute"
	"github.com/oracle/oci-go-sdk/v65/self"
	"github.com/oracle/oci-go-sdk/v65/servicecatalog"
	"github.com/oracle/oci-go-sdk/v65/servicemanagerproxy"
	"github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	"github.com/oracle/oci-go-sdk/v65/streaming"
	"github.com/oracle/oci-go-sdk/v65/tenantmanagercontrolplane"
	"github.com/oracle/oci-go-sdk/v65/usageapi"
	"github.com/oracle/oci-go-sdk/v65/vbsinst"
	"github.com/oracle/oci-go-sdk/v65/visualbuilder"
	"github.com/oracle/oci-go-sdk/v65/vnmonitoring"
	"github.com/oracle/oci-go-sdk/v65/vulnerabilityscanning"
	"github.com/oracle/oci-go-sdk/v65/waa"
	"github.com/oracle/oci-go-sdk/v65/waas"
	"github.com/oracle/oci-go-sdk/v65/waf"
	"github.com/oracle/oci-go-sdk/v65/wlms"
	"github.com/oracle/oci-go-sdk/v65/zpr"
)

const (
	modulePath    = "github.com/oracle/oci-go-sdk/v65"
	moduleVersion = "v65.110.0"
)

var seedTargets = []Target{
	// Autonomous Database CRD support
	newTarget("database", "CreateAutonomousDatabaseDetails", reflect.TypeOf(database.CreateAutonomousDatabaseDetails{})),
	newTarget("database", "UpdateAutonomousDatabaseDetails", reflect.TypeOf(database.UpdateAutonomousDatabaseDetails{})),
	newTarget("database", "AutonomousDatabase", reflect.TypeOf(database.AutonomousDatabase{})),
	newTarget("database", "AutonomousDatabaseSummary", reflect.TypeOf(database.AutonomousDatabaseSummary{})),

	// Email CRD support
	newTarget("email", "CreateDkimDetails", reflect.TypeOf(email.CreateDkimDetails{})),
	newTarget("email", "CreateEmailDomainDetails", reflect.TypeOf(email.CreateEmailDomainDetails{})),
	newTarget("email", "CreateSenderDetails", reflect.TypeOf(email.CreateSenderDetails{})),
	newTarget("email", "CreateSuppressionDetails", reflect.TypeOf(email.CreateSuppressionDetails{})),
	newTarget("email", "UpdateDkimDetails", reflect.TypeOf(email.UpdateDkimDetails{})),
	newTarget("email", "UpdateEmailDomainDetails", reflect.TypeOf(email.UpdateEmailDomainDetails{})),
	newTarget("email", "UpdateSenderDetails", reflect.TypeOf(email.UpdateSenderDetails{})),
	newTarget("email", "Dkim", reflect.TypeOf(email.Dkim{})),
	newTarget("email", "DkimCollection", reflect.TypeOf(email.DkimCollection{})),
	newTarget("email", "EmailDomain", reflect.TypeOf(email.EmailDomain{})),
	newTarget("email", "EmailDomainCollection", reflect.TypeOf(email.EmailDomainCollection{})),
	newTarget("email", "Sender", reflect.TypeOf(email.Sender{})),
	newTarget("email", "Suppression", reflect.TypeOf(email.Suppression{})),
	newTarget("email", "DkimSummary", reflect.TypeOf(email.DkimSummary{})),
	newTarget("email", "EmailDomainSummary", reflect.TypeOf(email.EmailDomainSummary{})),
	newTarget("email", "SenderSummary", reflect.TypeOf(email.SenderSummary{})),
	newTarget("email", "SuppressionSummary", reflect.TypeOf(email.SuppressionSummary{})),

	// Generative AI CRD support
	newTarget("generativeai", "CreateDedicatedAiClusterDetails", reflect.TypeOf(generativeai.CreateDedicatedAiClusterDetails{})),
	newTarget("generativeai", "CreateEndpointDetails", reflect.TypeOf(generativeai.CreateEndpointDetails{})),
	newTarget("generativeai", "CreateModelDetails", reflect.TypeOf(generativeai.CreateModelDetails{})),
	newTarget("generativeai", "UpdateDedicatedAiClusterDetails", reflect.TypeOf(generativeai.UpdateDedicatedAiClusterDetails{})),
	newTarget("generativeai", "UpdateEndpointDetails", reflect.TypeOf(generativeai.UpdateEndpointDetails{})),
	newTarget("generativeai", "UpdateModelDetails", reflect.TypeOf(generativeai.UpdateModelDetails{})),
	newTarget("generativeai", "DedicatedAiCluster", reflect.TypeOf(generativeai.DedicatedAiCluster{})),
	newTarget("generativeai", "DedicatedAiClusterCollection", reflect.TypeOf(generativeai.DedicatedAiClusterCollection{})),
	newTarget("generativeai", "Endpoint", reflect.TypeOf(generativeai.Endpoint{})),
	newTarget("generativeai", "EndpointCollection", reflect.TypeOf(generativeai.EndpointCollection{})),
	newTarget("generativeai", "Model", reflect.TypeOf(generativeai.Model{})),
	newTarget("generativeai", "ModelCollection", reflect.TypeOf(generativeai.ModelCollection{})),
	newTarget("generativeai", "DedicatedAiClusterSummary", reflect.TypeOf(generativeai.DedicatedAiClusterSummary{})),
	newTarget("generativeai", "EndpointSummary", reflect.TypeOf(generativeai.EndpointSummary{})),
	newTarget("generativeai", "ModelSummary", reflect.TypeOf(generativeai.ModelSummary{})),

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

	// Marketplace CRD support
	newTarget("marketplace", "CreateAcceptedAgreementDetails", reflect.TypeOf(marketplace.CreateAcceptedAgreementDetails{})),
	newTarget("marketplace", "CreatePublicationDetails", reflect.TypeOf(marketplace.CreatePublicationDetails{})),
	newTarget("marketplace", "UpdateAcceptedAgreementDetails", reflect.TypeOf(marketplace.UpdateAcceptedAgreementDetails{})),
	newTarget("marketplace", "UpdatePublicationDetails", reflect.TypeOf(marketplace.UpdatePublicationDetails{})),
	newTarget("marketplace", "AcceptedAgreement", reflect.TypeOf(marketplace.AcceptedAgreement{})),
	newTarget("marketplace", "Publication", reflect.TypeOf(marketplace.Publication{})),
	newTarget("marketplace", "AcceptedAgreementSummary", reflect.TypeOf(marketplace.AcceptedAgreementSummary{})),
	newTarget("marketplace", "PublicationSummary", reflect.TypeOf(marketplace.PublicationSummary{})),

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

	// OCVP CRD support
	newTarget("ocvp", "CreateClusterDetails", reflect.TypeOf(ocvp.CreateClusterDetails{})),
	newTarget("ocvp", "CreateEsxiHostDetails", reflect.TypeOf(ocvp.CreateEsxiHostDetails{})),
	newTarget("ocvp", "CreateSddcDetails", reflect.TypeOf(ocvp.CreateSddcDetails{})),
	newTarget("ocvp", "UpdateClusterDetails", reflect.TypeOf(ocvp.UpdateClusterDetails{})),
	newTarget("ocvp", "UpdateEsxiHostDetails", reflect.TypeOf(ocvp.UpdateEsxiHostDetails{})),
	newTarget("ocvp", "UpdateSddcDetails", reflect.TypeOf(ocvp.UpdateSddcDetails{})),
	newTarget("ocvp", "Cluster", reflect.TypeOf(ocvp.Cluster{})),
	newTarget("ocvp", "ClusterCollection", reflect.TypeOf(ocvp.ClusterCollection{})),
	newTarget("ocvp", "EsxiHost", reflect.TypeOf(ocvp.EsxiHost{})),
	newTarget("ocvp", "EsxiHostCollection", reflect.TypeOf(ocvp.EsxiHostCollection{})),
	newTarget("ocvp", "Sddc", reflect.TypeOf(ocvp.Sddc{})),
	newTarget("ocvp", "SddcCollection", reflect.TypeOf(ocvp.SddcCollection{})),
	newTarget("ocvp", "ClusterSummary", reflect.TypeOf(ocvp.ClusterSummary{})),
	newTarget("ocvp", "EsxiHostSummary", reflect.TypeOf(ocvp.EsxiHostSummary{})),
	newTarget("ocvp", "SddcSummary", reflect.TypeOf(ocvp.SddcSummary{})),

	// ODA CRD support
	newTarget("oda", "CreateAuthenticationProviderDetails", reflect.TypeOf(oda.CreateAuthenticationProviderDetails{})),
	newTarget("oda", "CreateImportedPackageDetails", reflect.TypeOf(oda.CreateImportedPackageDetails{})),
	newTarget("oda", "CreateOdaInstanceAttachmentDetails", reflect.TypeOf(oda.CreateOdaInstanceAttachmentDetails{})),
	newTarget("oda", "CreateOdaInstanceDetails", reflect.TypeOf(oda.CreateOdaInstanceDetails{})),
	newTarget("oda", "CreateOdaPrivateEndpointAttachmentDetails", reflect.TypeOf(oda.CreateOdaPrivateEndpointAttachmentDetails{})),
	newTarget("oda", "CreateOdaPrivateEndpointDetails", reflect.TypeOf(oda.CreateOdaPrivateEndpointDetails{})),
	newTarget("oda", "CreateOdaPrivateEndpointScanProxyDetails", reflect.TypeOf(oda.CreateOdaPrivateEndpointScanProxyDetails{})),
	newTarget("oda", "CreateSkillParameterDetails", reflect.TypeOf(oda.CreateSkillParameterDetails{})),
	newTarget("oda", "CreateTranslatorDetails", reflect.TypeOf(oda.CreateTranslatorDetails{})),
	newTarget("oda", "UpdateAuthenticationProviderDetails", reflect.TypeOf(oda.UpdateAuthenticationProviderDetails{})),
	newTarget("oda", "UpdateDigitalAssistantDetails", reflect.TypeOf(oda.UpdateDigitalAssistantDetails{})),
	newTarget("oda", "UpdateImportedPackageDetails", reflect.TypeOf(oda.UpdateImportedPackageDetails{})),
	newTarget("oda", "UpdateOdaInstanceAttachmentDetails", reflect.TypeOf(oda.UpdateOdaInstanceAttachmentDetails{})),
	newTarget("oda", "UpdateOdaInstanceDetails", reflect.TypeOf(oda.UpdateOdaInstanceDetails{})),
	newTarget("oda", "UpdateOdaPrivateEndpointDetails", reflect.TypeOf(oda.UpdateOdaPrivateEndpointDetails{})),
	newTarget("oda", "UpdateSkillDetails", reflect.TypeOf(oda.UpdateSkillDetails{})),
	newTarget("oda", "UpdateSkillParameterDetails", reflect.TypeOf(oda.UpdateSkillParameterDetails{})),
	newTarget("oda", "UpdateTranslatorDetails", reflect.TypeOf(oda.UpdateTranslatorDetails{})),
	newTarget("oda", "AuthenticationProvider", reflect.TypeOf(oda.AuthenticationProvider{})),
	newTarget("oda", "AuthenticationProviderCollection", reflect.TypeOf(oda.AuthenticationProviderCollection{})),
	newTarget("oda", "ChannelCollection", reflect.TypeOf(oda.ChannelCollection{})),
	newTarget("oda", "DigitalAssistant", reflect.TypeOf(oda.DigitalAssistant{})),
	newTarget("oda", "DigitalAssistantCollection", reflect.TypeOf(oda.DigitalAssistantCollection{})),
	newTarget("oda", "ImportedPackage", reflect.TypeOf(oda.ImportedPackage{})),
	newTarget("oda", "OdaInstance", reflect.TypeOf(oda.OdaInstance{})),
	newTarget("oda", "OdaInstanceAttachment", reflect.TypeOf(oda.OdaInstanceAttachment{})),
	newTarget("oda", "OdaInstanceAttachmentCollection", reflect.TypeOf(oda.OdaInstanceAttachmentCollection{})),
	newTarget("oda", "OdaPrivateEndpoint", reflect.TypeOf(oda.OdaPrivateEndpoint{})),
	newTarget("oda", "OdaPrivateEndpointAttachment", reflect.TypeOf(oda.OdaPrivateEndpointAttachment{})),
	newTarget("oda", "OdaPrivateEndpointAttachmentCollection", reflect.TypeOf(oda.OdaPrivateEndpointAttachmentCollection{})),
	newTarget("oda", "OdaPrivateEndpointCollection", reflect.TypeOf(oda.OdaPrivateEndpointCollection{})),
	newTarget("oda", "OdaPrivateEndpointScanProxy", reflect.TypeOf(oda.OdaPrivateEndpointScanProxy{})),
	newTarget("oda", "OdaPrivateEndpointScanProxyCollection", reflect.TypeOf(oda.OdaPrivateEndpointScanProxyCollection{})),
	newTarget("oda", "Skill", reflect.TypeOf(oda.Skill{})),
	newTarget("oda", "SkillCollection", reflect.TypeOf(oda.SkillCollection{})),
	newTarget("oda", "SkillParameter", reflect.TypeOf(oda.SkillParameter{})),
	newTarget("oda", "SkillParameterCollection", reflect.TypeOf(oda.SkillParameterCollection{})),
	newTarget("oda", "Translator", reflect.TypeOf(oda.Translator{})),
	newTarget("oda", "TranslatorCollection", reflect.TypeOf(oda.TranslatorCollection{})),
	newTarget("oda", "AuthenticationProviderSummary", reflect.TypeOf(oda.AuthenticationProviderSummary{})),
	newTarget("oda", "ChannelSummary", reflect.TypeOf(oda.ChannelSummary{})),
	newTarget("oda", "DigitalAssistantSummary", reflect.TypeOf(oda.DigitalAssistantSummary{})),
	newTarget("oda", "ImportedPackageSummary", reflect.TypeOf(oda.ImportedPackageSummary{})),
	newTarget("oda", "OdaInstanceAttachmentSummary", reflect.TypeOf(oda.OdaInstanceAttachmentSummary{})),
	newTarget("oda", "OdaInstanceSummary", reflect.TypeOf(oda.OdaInstanceSummary{})),
	newTarget("oda", "OdaPrivateEndpointAttachmentSummary", reflect.TypeOf(oda.OdaPrivateEndpointAttachmentSummary{})),
	newTarget("oda", "OdaPrivateEndpointScanProxySummary", reflect.TypeOf(oda.OdaPrivateEndpointScanProxySummary{})),
	newTarget("oda", "OdaPrivateEndpointSummary", reflect.TypeOf(oda.OdaPrivateEndpointSummary{})),
	newTarget("oda", "SkillParameterSummary", reflect.TypeOf(oda.SkillParameterSummary{})),
	newTarget("oda", "SkillSummary", reflect.TypeOf(oda.SkillSummary{})),
	newTarget("oda", "TranslatorSummary", reflect.TypeOf(oda.TranslatorSummary{})),

	// Notifications (ONS) CRD support
	newTarget("ons", "CreateSubscriptionDetails", reflect.TypeOf(ons.CreateSubscriptionDetails{})),
	newTarget("ons", "CreateTopicDetails", reflect.TypeOf(ons.CreateTopicDetails{})),
	newTarget("ons", "UpdateSubscriptionDetails", reflect.TypeOf(ons.UpdateSubscriptionDetails{})),
	newTarget("ons", "NotificationTopic", reflect.TypeOf(ons.NotificationTopic{})),
	newTarget("ons", "Subscription", reflect.TypeOf(ons.Subscription{})),
	newTarget("ons", "NotificationTopicSummary", reflect.TypeOf(ons.NotificationTopicSummary{})),
	newTarget("ons", "SubscriptionSummary", reflect.TypeOf(ons.SubscriptionSummary{})),

	// Logging CRD support
	newTarget("logging", "CreateLogDetails", reflect.TypeOf(logging.CreateLogDetails{})),
	newTarget("logging", "CreateLogGroupDetails", reflect.TypeOf(logging.CreateLogGroupDetails{})),
	newTarget("logging", "CreateLogSavedSearchDetails", reflect.TypeOf(logging.CreateLogSavedSearchDetails{})),
	newTarget("logging", "CreateUnifiedAgentConfigurationDetails", reflect.TypeOf(logging.CreateUnifiedAgentConfigurationDetails{})),
	newTarget("logging", "UpdateLogDetails", reflect.TypeOf(logging.UpdateLogDetails{})),
	newTarget("logging", "UpdateLogGroupDetails", reflect.TypeOf(logging.UpdateLogGroupDetails{})),
	newTarget("logging", "UpdateLogSavedSearchDetails", reflect.TypeOf(logging.UpdateLogSavedSearchDetails{})),
	newTarget("logging", "UpdateUnifiedAgentConfigurationDetails", reflect.TypeOf(logging.UpdateUnifiedAgentConfigurationDetails{})),
	newTarget("logging", "Log", reflect.TypeOf(logging.Log{})),
	newTarget("logging", "LogGroup", reflect.TypeOf(logging.LogGroup{})),
	newTarget("logging", "LogSavedSearch", reflect.TypeOf(logging.LogSavedSearch{})),
	newTarget("logging", "UnifiedAgentConfiguration", reflect.TypeOf(logging.UnifiedAgentConfiguration{})),
	newTarget("logging", "UnifiedAgentConfigurationCollection", reflect.TypeOf(logging.UnifiedAgentConfigurationCollection{})),
	newTarget("logging", "LogGroupSummary", reflect.TypeOf(logging.LogGroupSummary{})),
	newTarget("logging", "LogSavedSearchSummary", reflect.TypeOf(logging.LogSavedSearchSummary{})),
	newTarget("logging", "LogSummary", reflect.TypeOf(logging.LogSummary{})),
	newTarget("logging", "UnifiedAgentConfigurationSummary", reflect.TypeOf(logging.UnifiedAgentConfigurationSummary{})),

	// PostgreSQL CRD support
	newTarget("psql", "CreateDbSystemDetails", reflect.TypeOf(psql.CreateDbSystemDetails{})),
	newTarget("psql", "UpdateDbSystemDetails", reflect.TypeOf(psql.UpdateDbSystemDetails{})),
	newTarget("psql", "DbSystemDetails", reflect.TypeOf(psql.DbSystemDetails{})),
	newTarget("psql", "DbSystem", reflect.TypeOf(psql.DbSystem{})),
	newTarget("psql", "DbSystemCollection", reflect.TypeOf(psql.DbSystemCollection{})),
	newTarget("psql", "DbSystemSummary", reflect.TypeOf(psql.DbSystemSummary{})),

	// Usage API CRD support
	newTarget("usageapi", "CreateCustomTableDetails", reflect.TypeOf(usageapi.CreateCustomTableDetails{})),
	newTarget("usageapi", "CreateQueryDetails", reflect.TypeOf(usageapi.CreateQueryDetails{})),
	newTarget("usageapi", "CreateScheduleDetails", reflect.TypeOf(usageapi.CreateScheduleDetails{})),
	newTarget("usageapi", "CreateUsageCarbonEmissionsQueryDetails", reflect.TypeOf(usageapi.CreateUsageCarbonEmissionsQueryDetails{})),
	newTarget("usageapi", "UpdateCustomTableDetails", reflect.TypeOf(usageapi.UpdateCustomTableDetails{})),
	newTarget("usageapi", "UpdateQueryDetails", reflect.TypeOf(usageapi.UpdateQueryDetails{})),
	newTarget("usageapi", "UpdateScheduleDetails", reflect.TypeOf(usageapi.UpdateScheduleDetails{})),
	newTarget("usageapi", "UpdateUsageCarbonEmissionsQueryDetails", reflect.TypeOf(usageapi.UpdateUsageCarbonEmissionsQueryDetails{})),
	newTarget("usageapi", "CustomTable", reflect.TypeOf(usageapi.CustomTable{})),
	newTarget("usageapi", "CustomTableCollection", reflect.TypeOf(usageapi.CustomTableCollection{})),
	newTarget("usageapi", "Query", reflect.TypeOf(usageapi.Query{})),
	newTarget("usageapi", "QueryCollection", reflect.TypeOf(usageapi.QueryCollection{})),
	newTarget("usageapi", "Schedule", reflect.TypeOf(usageapi.Schedule{})),
	newTarget("usageapi", "ScheduleCollection", reflect.TypeOf(usageapi.ScheduleCollection{})),
	newTarget("usageapi", "UsageCarbonEmissionsQuery", reflect.TypeOf(usageapi.UsageCarbonEmissionsQuery{})),
	newTarget("usageapi", "UsageCarbonEmissionsQueryCollection", reflect.TypeOf(usageapi.UsageCarbonEmissionsQueryCollection{})),
	newTarget("usageapi", "CustomTableSummary", reflect.TypeOf(usageapi.CustomTableSummary{})),
	newTarget("usageapi", "QuerySummary", reflect.TypeOf(usageapi.QuerySummary{})),
	newTarget("usageapi", "ScheduleSummary", reflect.TypeOf(usageapi.ScheduleSummary{})),
	newTarget("usageapi", "UsageCarbonEmissionsQuerySummary", reflect.TypeOf(usageapi.UsageCarbonEmissionsQuerySummary{})),

	// Events CRD support
	newTarget("events", "CreateRuleDetails", reflect.TypeOf(events.CreateRuleDetails{})),
	newTarget("events", "UpdateRuleDetails", reflect.TypeOf(events.UpdateRuleDetails{})),
	newTarget("events", "Rule", reflect.TypeOf(events.Rule{})),
	newTarget("events", "RuleSummary", reflect.TypeOf(events.RuleSummary{})),

	// Monitoring CRD support
	newTarget("monitoring", "CreateAlarmDetails", reflect.TypeOf(monitoring.CreateAlarmDetails{})),
	newTarget("monitoring", "CreateAlarmSuppressionDetails", reflect.TypeOf(monitoring.CreateAlarmSuppressionDetails{})),
	newTarget("monitoring", "UpdateAlarmDetails", reflect.TypeOf(monitoring.UpdateAlarmDetails{})),
	newTarget("monitoring", "Alarm", reflect.TypeOf(monitoring.Alarm{})),
	newTarget("monitoring", "AlarmSuppression", reflect.TypeOf(monitoring.AlarmSuppression{})),
	newTarget("monitoring", "AlarmSuppressionCollection", reflect.TypeOf(monitoring.AlarmSuppressionCollection{})),
	newTarget("monitoring", "AlarmSummary", reflect.TypeOf(monitoring.AlarmSummary{})),
	newTarget("monitoring", "AlarmSuppressionSummary", reflect.TypeOf(monitoring.AlarmSuppressionSummary{})),

	// DNS CRD support
	newTarget("dns", "CreateSteeringPolicyAttachmentDetails", reflect.TypeOf(dns.CreateSteeringPolicyAttachmentDetails{})),
	newTarget("dns", "CreateSteeringPolicyDetails", reflect.TypeOf(dns.CreateSteeringPolicyDetails{})),
	newTarget("dns", "CreateTsigKeyDetails", reflect.TypeOf(dns.CreateTsigKeyDetails{})),
	newTarget("dns", "CreateViewDetails", reflect.TypeOf(dns.CreateViewDetails{})),
	newTarget("dns", "CreateZoneDetails", reflect.TypeOf(dns.CreateZoneDetails{})),
	newTarget("dns", "UpdateSteeringPolicyAttachmentDetails", reflect.TypeOf(dns.UpdateSteeringPolicyAttachmentDetails{})),
	newTarget("dns", "UpdateSteeringPolicyDetails", reflect.TypeOf(dns.UpdateSteeringPolicyDetails{})),
	newTarget("dns", "UpdateTsigKeyDetails", reflect.TypeOf(dns.UpdateTsigKeyDetails{})),
	newTarget("dns", "UpdateViewDetails", reflect.TypeOf(dns.UpdateViewDetails{})),
	newTarget("dns", "UpdateZoneDetails", reflect.TypeOf(dns.UpdateZoneDetails{})),
	newTarget("dns", "SteeringPolicy", reflect.TypeOf(dns.SteeringPolicy{})),
	newTarget("dns", "SteeringPolicyAttachment", reflect.TypeOf(dns.SteeringPolicyAttachment{})),
	newTarget("dns", "TsigKey", reflect.TypeOf(dns.TsigKey{})),
	newTarget("dns", "View", reflect.TypeOf(dns.View{})),
	newTarget("dns", "Zone", reflect.TypeOf(dns.Zone{})),
	newTarget("dns", "SteeringPolicyAttachmentSummary", reflect.TypeOf(dns.SteeringPolicyAttachmentSummary{})),
	newTarget("dns", "SteeringPolicySummary", reflect.TypeOf(dns.SteeringPolicySummary{})),
	newTarget("dns", "TsigKeySummary", reflect.TypeOf(dns.TsigKeySummary{})),
	newTarget("dns", "ViewSummary", reflect.TypeOf(dns.ViewSummary{})),
	newTarget("dns", "ZoneSummary", reflect.TypeOf(dns.ZoneSummary{})),

	// Load Balancer CRD support
	newTarget("loadbalancer", "CreateBackendDetails", reflect.TypeOf(loadbalancer.CreateBackendDetails{})),
	newTarget("loadbalancer", "CreateBackendSetDetails", reflect.TypeOf(loadbalancer.CreateBackendSetDetails{})),
	newTarget("loadbalancer", "CreateCertificateDetails", reflect.TypeOf(loadbalancer.CreateCertificateDetails{})),
	newTarget("loadbalancer", "CreateHostnameDetails", reflect.TypeOf(loadbalancer.CreateHostnameDetails{})),
	newTarget("loadbalancer", "CreateListenerDetails", reflect.TypeOf(loadbalancer.CreateListenerDetails{})),
	newTarget("loadbalancer", "CreateLoadBalancerDetails", reflect.TypeOf(loadbalancer.CreateLoadBalancerDetails{})),
	newTarget("loadbalancer", "CreatePathRouteSetDetails", reflect.TypeOf(loadbalancer.CreatePathRouteSetDetails{})),
	newTarget("loadbalancer", "CreateRoutingPolicyDetails", reflect.TypeOf(loadbalancer.CreateRoutingPolicyDetails{})),
	newTarget("loadbalancer", "CreateRuleSetDetails", reflect.TypeOf(loadbalancer.CreateRuleSetDetails{})),
	newTarget("loadbalancer", "CreateSslCipherSuiteDetails", reflect.TypeOf(loadbalancer.CreateSslCipherSuiteDetails{})),
	newTarget("loadbalancer", "UpdateBackendDetails", reflect.TypeOf(loadbalancer.UpdateBackendDetails{})),
	newTarget("loadbalancer", "UpdateBackendSetDetails", reflect.TypeOf(loadbalancer.UpdateBackendSetDetails{})),
	newTarget("loadbalancer", "UpdateHostnameDetails", reflect.TypeOf(loadbalancer.UpdateHostnameDetails{})),
	newTarget("loadbalancer", "UpdateListenerDetails", reflect.TypeOf(loadbalancer.UpdateListenerDetails{})),
	newTarget("loadbalancer", "UpdateLoadBalancerDetails", reflect.TypeOf(loadbalancer.UpdateLoadBalancerDetails{})),
	newTarget("loadbalancer", "UpdatePathRouteSetDetails", reflect.TypeOf(loadbalancer.UpdatePathRouteSetDetails{})),
	newTarget("loadbalancer", "UpdateRoutingPolicyDetails", reflect.TypeOf(loadbalancer.UpdateRoutingPolicyDetails{})),
	newTarget("loadbalancer", "UpdateRuleSetDetails", reflect.TypeOf(loadbalancer.UpdateRuleSetDetails{})),
	newTarget("loadbalancer", "UpdateSslCipherSuiteDetails", reflect.TypeOf(loadbalancer.UpdateSslCipherSuiteDetails{})),
	newTarget("loadbalancer", "BackendDetails", reflect.TypeOf(loadbalancer.BackendDetails{})),
	newTarget("loadbalancer", "BackendSetDetails", reflect.TypeOf(loadbalancer.BackendSetDetails{})),
	newTarget("loadbalancer", "CertificateDetails", reflect.TypeOf(loadbalancer.CertificateDetails{})),
	newTarget("loadbalancer", "HostnameDetails", reflect.TypeOf(loadbalancer.HostnameDetails{})),
	newTarget("loadbalancer", "ListenerDetails", reflect.TypeOf(loadbalancer.ListenerDetails{})),
	newTarget("loadbalancer", "PathRouteSetDetails", reflect.TypeOf(loadbalancer.PathRouteSetDetails{})),
	newTarget("loadbalancer", "RoutingPolicyDetails", reflect.TypeOf(loadbalancer.RoutingPolicyDetails{})),
	newTarget("loadbalancer", "RuleSetDetails", reflect.TypeOf(loadbalancer.RuleSetDetails{})),
	newTarget("loadbalancer", "SslCipherSuiteDetails", reflect.TypeOf(loadbalancer.SslCipherSuiteDetails{})),
	newTarget("loadbalancer", "Backend", reflect.TypeOf(loadbalancer.Backend{})),
	newTarget("loadbalancer", "BackendSet", reflect.TypeOf(loadbalancer.BackendSet{})),
	newTarget("loadbalancer", "Certificate", reflect.TypeOf(loadbalancer.Certificate{})),
	newTarget("loadbalancer", "Hostname", reflect.TypeOf(loadbalancer.Hostname{})),
	newTarget("loadbalancer", "Listener", reflect.TypeOf(loadbalancer.Listener{})),
	newTarget("loadbalancer", "LoadBalancer", reflect.TypeOf(loadbalancer.LoadBalancer{})),
	newTarget("loadbalancer", "PathRouteSet", reflect.TypeOf(loadbalancer.PathRouteSet{})),
	newTarget("loadbalancer", "RoutingPolicy", reflect.TypeOf(loadbalancer.RoutingPolicy{})),
	newTarget("loadbalancer", "RuleSet", reflect.TypeOf(loadbalancer.RuleSet{})),
	newTarget("loadbalancer", "SslCipherSuite", reflect.TypeOf(loadbalancer.SslCipherSuite{})),

	// Network Load Balancer CRD support
	newTarget("networkloadbalancer", "CreateBackendDetails", reflect.TypeOf(networkloadbalancer.CreateBackendDetails{})),
	newTarget("networkloadbalancer", "CreateBackendSetDetails", reflect.TypeOf(networkloadbalancer.CreateBackendSetDetails{})),
	newTarget("networkloadbalancer", "CreateListenerDetails", reflect.TypeOf(networkloadbalancer.CreateListenerDetails{})),
	newTarget("networkloadbalancer", "CreateNetworkLoadBalancerDetails", reflect.TypeOf(networkloadbalancer.CreateNetworkLoadBalancerDetails{})),
	newTarget("networkloadbalancer", "UpdateBackendDetails", reflect.TypeOf(networkloadbalancer.UpdateBackendDetails{})),
	newTarget("networkloadbalancer", "UpdateBackendSetDetails", reflect.TypeOf(networkloadbalancer.UpdateBackendSetDetails{})),
	newTarget("networkloadbalancer", "UpdateListenerDetails", reflect.TypeOf(networkloadbalancer.UpdateListenerDetails{})),
	newTarget("networkloadbalancer", "UpdateNetworkLoadBalancerDetails", reflect.TypeOf(networkloadbalancer.UpdateNetworkLoadBalancerDetails{})),
	newTarget("networkloadbalancer", "BackendDetails", reflect.TypeOf(networkloadbalancer.BackendDetails{})),
	newTarget("networkloadbalancer", "BackendSetDetails", reflect.TypeOf(networkloadbalancer.BackendSetDetails{})),
	newTarget("networkloadbalancer", "ListenerDetails", reflect.TypeOf(networkloadbalancer.ListenerDetails{})),
	newTarget("networkloadbalancer", "Backend", reflect.TypeOf(networkloadbalancer.Backend{})),
	newTarget("networkloadbalancer", "BackendCollection", reflect.TypeOf(networkloadbalancer.BackendCollection{})),
	newTarget("networkloadbalancer", "BackendSet", reflect.TypeOf(networkloadbalancer.BackendSet{})),
	newTarget("networkloadbalancer", "BackendSetCollection", reflect.TypeOf(networkloadbalancer.BackendSetCollection{})),
	newTarget("networkloadbalancer", "Listener", reflect.TypeOf(networkloadbalancer.Listener{})),
	newTarget("networkloadbalancer", "ListenerCollection", reflect.TypeOf(networkloadbalancer.ListenerCollection{})),
	newTarget("networkloadbalancer", "NetworkLoadBalancer", reflect.TypeOf(networkloadbalancer.NetworkLoadBalancer{})),
	newTarget("networkloadbalancer", "NetworkLoadBalancerCollection", reflect.TypeOf(networkloadbalancer.NetworkLoadBalancerCollection{})),
	newTarget("networkloadbalancer", "BackendSetSummary", reflect.TypeOf(networkloadbalancer.BackendSetSummary{})),
	newTarget("networkloadbalancer", "BackendSummary", reflect.TypeOf(networkloadbalancer.BackendSummary{})),
	newTarget("networkloadbalancer", "ListenerSummary", reflect.TypeOf(networkloadbalancer.ListenerSummary{})),
	newTarget("networkloadbalancer", "NetworkLoadBalancerSummary", reflect.TypeOf(networkloadbalancer.NetworkLoadBalancerSummary{})),

	// Artifacts CRD support
	newTarget("artifacts", "CreateContainerImageSignatureDetails", reflect.TypeOf(artifacts.CreateContainerImageSignatureDetails{})),
	newTarget("artifacts", "CreateContainerRepositoryDetails", reflect.TypeOf(artifacts.CreateContainerRepositoryDetails{})),
	newTarget("artifacts", "UpdateContainerImageSignatureDetails", reflect.TypeOf(artifacts.UpdateContainerImageSignatureDetails{})),
	newTarget("artifacts", "UpdateContainerRepositoryDetails", reflect.TypeOf(artifacts.UpdateContainerRepositoryDetails{})),
	newTarget("artifacts", "ContainerImageSignature", reflect.TypeOf(artifacts.ContainerImageSignature{})),
	newTarget("artifacts", "ContainerImageSignatureCollection", reflect.TypeOf(artifacts.ContainerImageSignatureCollection{})),
	newTarget("artifacts", "ContainerRepository", reflect.TypeOf(artifacts.ContainerRepository{})),
	newTarget("artifacts", "ContainerRepositoryCollection", reflect.TypeOf(artifacts.ContainerRepositoryCollection{})),
	newTarget("artifacts", "GenericRepository", reflect.TypeOf(artifacts.GenericRepository{})),
	newTarget("artifacts", "RepositoryCollection", reflect.TypeOf(artifacts.RepositoryCollection{})),
	newTarget("artifacts", "ContainerImageSignatureSummary", reflect.TypeOf(artifacts.ContainerImageSignatureSummary{})),
	newTarget("artifacts", "ContainerRepositorySummary", reflect.TypeOf(artifacts.ContainerRepositorySummary{})),

	// Certificates Management CRD support
	newTarget("certificatesmanagement", "CreateCaBundleDetails", reflect.TypeOf(certificatesmanagement.CreateCaBundleDetails{})),
	newTarget("certificatesmanagement", "UpdateCaBundleDetails", reflect.TypeOf(certificatesmanagement.UpdateCaBundleDetails{})),
	newTarget("certificatesmanagement", "CaBundle", reflect.TypeOf(certificatesmanagement.CaBundle{})),
	newTarget("certificatesmanagement", "CaBundleCollection", reflect.TypeOf(certificatesmanagement.CaBundleCollection{})),
	newTarget("certificatesmanagement", "CaBundleSummary", reflect.TypeOf(certificatesmanagement.CaBundleSummary{})),

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
	newTarget("keymanagement", "CreateEkmsPrivateEndpointDetails", reflect.TypeOf(keymanagement.CreateEkmsPrivateEndpointDetails{})),
	newTarget("keymanagement", "CreateVaultDetails", reflect.TypeOf(keymanagement.CreateVaultDetails{})),
	newTarget("keymanagement", "UpdateEkmsPrivateEndpointDetails", reflect.TypeOf(keymanagement.UpdateEkmsPrivateEndpointDetails{})),
	newTarget("keymanagement", "UpdateVaultDetails", reflect.TypeOf(keymanagement.UpdateVaultDetails{})),
	newTarget("keymanagement", "EkmsPrivateEndpoint", reflect.TypeOf(keymanagement.EkmsPrivateEndpoint{})),
	newTarget("keymanagement", "Vault", reflect.TypeOf(keymanagement.Vault{})),
	newTarget("keymanagement", "EkmsPrivateEndpointSummary", reflect.TypeOf(keymanagement.EkmsPrivateEndpointSummary{})),
	newTarget("keymanagement", "VaultSummary", reflect.TypeOf(keymanagement.VaultSummary{})),

	// Limits CRD support
	newTarget("limits", "CreateQuotaDetails", reflect.TypeOf(limits.CreateQuotaDetails{})),
	newTarget("limits", "UpdateQuotaDetails", reflect.TypeOf(limits.UpdateQuotaDetails{})),
	newTarget("limits", "Quota", reflect.TypeOf(limits.Quota{})),
	newTarget("limits", "QuotaSummary", reflect.TypeOf(limits.QuotaSummary{})),

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

	// Accessgovernancecp CRD support
	newTarget("accessgovernancecp", "CreateGovernanceInstanceDetails", reflect.TypeOf(accessgovernancecp.CreateGovernanceInstanceDetails{})),
	newTarget("accessgovernancecp", "UpdateGovernanceInstanceDetails", reflect.TypeOf(accessgovernancecp.UpdateGovernanceInstanceDetails{})),
	newTarget("accessgovernancecp", "GovernanceInstance", reflect.TypeOf(accessgovernancecp.GovernanceInstance{})),
	newTarget("accessgovernancecp", "GovernanceInstanceCollection", reflect.TypeOf(accessgovernancecp.GovernanceInstanceCollection{})),
	newTarget("accessgovernancecp", "GovernanceInstanceSummary", reflect.TypeOf(accessgovernancecp.GovernanceInstanceSummary{})),

	// Adm CRD support
	newTarget("adm", "CreateKnowledgeBaseDetails", reflect.TypeOf(adm.CreateKnowledgeBaseDetails{})),
	newTarget("adm", "UpdateKnowledgeBaseDetails", reflect.TypeOf(adm.UpdateKnowledgeBaseDetails{})),
	newTarget("adm", "KnowledgeBase", reflect.TypeOf(adm.KnowledgeBase{})),
	newTarget("adm", "KnowledgeBaseCollection", reflect.TypeOf(adm.KnowledgeBaseCollection{})),
	newTarget("adm", "KnowledgeBaseSummary", reflect.TypeOf(adm.KnowledgeBaseSummary{})),

	// Aidataplatform CRD support
	newTarget("aidataplatform", "CreateAiDataPlatformDetails", reflect.TypeOf(aidataplatform.CreateAiDataPlatformDetails{})),
	newTarget("aidataplatform", "UpdateAiDataPlatformDetails", reflect.TypeOf(aidataplatform.UpdateAiDataPlatformDetails{})),
	newTarget("aidataplatform", "AiDataPlatform", reflect.TypeOf(aidataplatform.AiDataPlatform{})),
	newTarget("aidataplatform", "AiDataPlatformCollection", reflect.TypeOf(aidataplatform.AiDataPlatformCollection{})),
	newTarget("aidataplatform", "AiDataPlatformSummary", reflect.TypeOf(aidataplatform.AiDataPlatformSummary{})),

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

	// Aispeech CRD support
	newTarget("aispeech", "CreateTranscriptionJobDetails", reflect.TypeOf(aispeech.CreateTranscriptionJobDetails{})),
	newTarget("aispeech", "UpdateTranscriptionJobDetails", reflect.TypeOf(aispeech.UpdateTranscriptionJobDetails{})),
	newTarget("aispeech", "TranscriptionJob", reflect.TypeOf(aispeech.TranscriptionJob{})),
	newTarget("aispeech", "TranscriptionJobCollection", reflect.TypeOf(aispeech.TranscriptionJobCollection{})),
	newTarget("aispeech", "TranscriptionJobSummary", reflect.TypeOf(aispeech.TranscriptionJobSummary{})),

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

	// Announcementsservice CRD support
	newTarget("announcementsservice", "CreateAnnouncementSubscriptionDetails", reflect.TypeOf(announcementsservice.CreateAnnouncementSubscriptionDetails{})),
	newTarget("announcementsservice", "UpdateAnnouncementSubscriptionDetails", reflect.TypeOf(announcementsservice.UpdateAnnouncementSubscriptionDetails{})),
	newTarget("announcementsservice", "AnnouncementSubscription", reflect.TypeOf(announcementsservice.AnnouncementSubscription{})),
	newTarget("announcementsservice", "AnnouncementSubscriptionCollection", reflect.TypeOf(announcementsservice.AnnouncementSubscriptionCollection{})),
	newTarget("announcementsservice", "AnnouncementSubscriptionSummary", reflect.TypeOf(announcementsservice.AnnouncementSubscriptionSummary{})),

	// Apiaccesscontrol CRD support
	newTarget("apiaccesscontrol", "CreatePrivilegedApiControlDetails", reflect.TypeOf(apiaccesscontrol.CreatePrivilegedApiControlDetails{})),
	newTarget("apiaccesscontrol", "UpdatePrivilegedApiControlDetails", reflect.TypeOf(apiaccesscontrol.UpdatePrivilegedApiControlDetails{})),
	newTarget("apiaccesscontrol", "PrivilegedApiControl", reflect.TypeOf(apiaccesscontrol.PrivilegedApiControl{})),
	newTarget("apiaccesscontrol", "PrivilegedApiControlCollection", reflect.TypeOf(apiaccesscontrol.PrivilegedApiControlCollection{})),
	newTarget("apiaccesscontrol", "PrivilegedApiControlSummary", reflect.TypeOf(apiaccesscontrol.PrivilegedApiControlSummary{})),

	// Apiplatform CRD support
	newTarget("apiplatform", "CreateApiPlatformInstanceDetails", reflect.TypeOf(apiplatform.CreateApiPlatformInstanceDetails{})),
	newTarget("apiplatform", "UpdateApiPlatformInstanceDetails", reflect.TypeOf(apiplatform.UpdateApiPlatformInstanceDetails{})),
	newTarget("apiplatform", "ApiPlatformInstance", reflect.TypeOf(apiplatform.ApiPlatformInstance{})),
	newTarget("apiplatform", "ApiPlatformInstanceCollection", reflect.TypeOf(apiplatform.ApiPlatformInstanceCollection{})),
	newTarget("apiplatform", "ApiPlatformInstanceSummary", reflect.TypeOf(apiplatform.ApiPlatformInstanceSummary{})),

	// Apmconfig CRD support
	newTarget("apmconfig", "ConfigCollection", reflect.TypeOf(apmconfig.ConfigCollection{})),

	// Apmcontrolplane CRD support
	newTarget("apmcontrolplane", "CreateApmDomainDetails", reflect.TypeOf(apmcontrolplane.CreateApmDomainDetails{})),
	newTarget("apmcontrolplane", "UpdateApmDomainDetails", reflect.TypeOf(apmcontrolplane.UpdateApmDomainDetails{})),
	newTarget("apmcontrolplane", "ApmDomain", reflect.TypeOf(apmcontrolplane.ApmDomain{})),
	newTarget("apmcontrolplane", "ApmDomainSummary", reflect.TypeOf(apmcontrolplane.ApmDomainSummary{})),

	// Apmsynthetics CRD support
	newTarget("apmsynthetics", "CreateScriptDetails", reflect.TypeOf(apmsynthetics.CreateScriptDetails{})),
	newTarget("apmsynthetics", "UpdateScriptDetails", reflect.TypeOf(apmsynthetics.UpdateScriptDetails{})),
	newTarget("apmsynthetics", "Script", reflect.TypeOf(apmsynthetics.Script{})),
	newTarget("apmsynthetics", "ScriptCollection", reflect.TypeOf(apmsynthetics.ScriptCollection{})),
	newTarget("apmsynthetics", "ScriptSummary", reflect.TypeOf(apmsynthetics.ScriptSummary{})),

	// Apmtraces CRD support
	newTarget("apmtraces", "CreateScheduledQueryDetails", reflect.TypeOf(apmtraces.CreateScheduledQueryDetails{})),
	newTarget("apmtraces", "UpdateScheduledQueryDetails", reflect.TypeOf(apmtraces.UpdateScheduledQueryDetails{})),
	newTarget("apmtraces", "ScheduledQuery", reflect.TypeOf(apmtraces.ScheduledQuery{})),
	newTarget("apmtraces", "ScheduledQueryCollection", reflect.TypeOf(apmtraces.ScheduledQueryCollection{})),
	newTarget("apmtraces", "ScheduledQuerySummary", reflect.TypeOf(apmtraces.ScheduledQuerySummary{})),

	// Appmgmtcontrol CRD support
	newTarget("appmgmtcontrol", "MonitoredInstance", reflect.TypeOf(appmgmtcontrol.MonitoredInstance{})),
	newTarget("appmgmtcontrol", "MonitoredInstanceCollection", reflect.TypeOf(appmgmtcontrol.MonitoredInstanceCollection{})),
	newTarget("appmgmtcontrol", "MonitoredInstanceSummary", reflect.TypeOf(appmgmtcontrol.MonitoredInstanceSummary{})),

	// Autoscaling CRD support
	newTarget("autoscaling", "CreateAutoScalingConfigurationDetails", reflect.TypeOf(autoscaling.CreateAutoScalingConfigurationDetails{})),
	newTarget("autoscaling", "UpdateAutoScalingConfigurationDetails", reflect.TypeOf(autoscaling.UpdateAutoScalingConfigurationDetails{})),
	newTarget("autoscaling", "AutoScalingConfiguration", reflect.TypeOf(autoscaling.AutoScalingConfiguration{})),
	newTarget("autoscaling", "AutoScalingConfigurationSummary", reflect.TypeOf(autoscaling.AutoScalingConfigurationSummary{})),
	newTarget("autoscaling", "AutoScalingPolicySummary", reflect.TypeOf(autoscaling.AutoScalingPolicySummary{})),

	// Bastion CRD support
	newTarget("bastion", "CreateBastionDetails", reflect.TypeOf(bastion.CreateBastionDetails{})),
	newTarget("bastion", "CreateSessionDetails", reflect.TypeOf(bastion.CreateSessionDetails{})),
	newTarget("bastion", "UpdateBastionDetails", reflect.TypeOf(bastion.UpdateBastionDetails{})),
	newTarget("bastion", "UpdateSessionDetails", reflect.TypeOf(bastion.UpdateSessionDetails{})),
	newTarget("bastion", "Bastion", reflect.TypeOf(bastion.Bastion{})),
	newTarget("bastion", "Session", reflect.TypeOf(bastion.Session{})),
	newTarget("bastion", "BastionSummary", reflect.TypeOf(bastion.BastionSummary{})),
	newTarget("bastion", "SessionSummary", reflect.TypeOf(bastion.SessionSummary{})),

	// Batch CRD support
	newTarget("batch", "CreateBatchContextDetails", reflect.TypeOf(batch.CreateBatchContextDetails{})),
	newTarget("batch", "CreateBatchJobPoolDetails", reflect.TypeOf(batch.CreateBatchJobPoolDetails{})),
	newTarget("batch", "CreateBatchTaskEnvironmentDetails", reflect.TypeOf(batch.CreateBatchTaskEnvironmentDetails{})),
	newTarget("batch", "CreateBatchTaskProfileDetails", reflect.TypeOf(batch.CreateBatchTaskProfileDetails{})),
	newTarget("batch", "UpdateBatchContextDetails", reflect.TypeOf(batch.UpdateBatchContextDetails{})),
	newTarget("batch", "UpdateBatchJobPoolDetails", reflect.TypeOf(batch.UpdateBatchJobPoolDetails{})),
	newTarget("batch", "UpdateBatchTaskEnvironmentDetails", reflect.TypeOf(batch.UpdateBatchTaskEnvironmentDetails{})),
	newTarget("batch", "UpdateBatchTaskProfileDetails", reflect.TypeOf(batch.UpdateBatchTaskProfileDetails{})),
	newTarget("batch", "BatchContext", reflect.TypeOf(batch.BatchContext{})),
	newTarget("batch", "BatchContextCollection", reflect.TypeOf(batch.BatchContextCollection{})),
	newTarget("batch", "BatchJobPool", reflect.TypeOf(batch.BatchJobPool{})),
	newTarget("batch", "BatchJobPoolCollection", reflect.TypeOf(batch.BatchJobPoolCollection{})),
	newTarget("batch", "BatchTaskEnvironment", reflect.TypeOf(batch.BatchTaskEnvironment{})),
	newTarget("batch", "BatchTaskEnvironmentCollection", reflect.TypeOf(batch.BatchTaskEnvironmentCollection{})),
	newTarget("batch", "BatchTaskProfile", reflect.TypeOf(batch.BatchTaskProfile{})),
	newTarget("batch", "BatchTaskProfileCollection", reflect.TypeOf(batch.BatchTaskProfileCollection{})),
	newTarget("batch", "BatchContextSummary", reflect.TypeOf(batch.BatchContextSummary{})),
	newTarget("batch", "BatchJobPoolSummary", reflect.TypeOf(batch.BatchJobPoolSummary{})),
	newTarget("batch", "BatchTaskEnvironmentSummary", reflect.TypeOf(batch.BatchTaskEnvironmentSummary{})),
	newTarget("batch", "BatchTaskProfileSummary", reflect.TypeOf(batch.BatchTaskProfileSummary{})),

	// Bds CRD support
	newTarget("bds", "CreateBdsInstanceDetails", reflect.TypeOf(bds.CreateBdsInstanceDetails{})),
	newTarget("bds", "UpdateBdsInstanceDetails", reflect.TypeOf(bds.UpdateBdsInstanceDetails{})),
	newTarget("bds", "BdsInstance", reflect.TypeOf(bds.BdsInstance{})),
	newTarget("bds", "BdsInstanceSummary", reflect.TypeOf(bds.BdsInstanceSummary{})),

	// Blockchain CRD support
	newTarget("blockchain", "CreateBlockchainPlatformDetails", reflect.TypeOf(blockchain.CreateBlockchainPlatformDetails{})),
	newTarget("blockchain", "CreateOsnDetails", reflect.TypeOf(blockchain.CreateOsnDetails{})),
	newTarget("blockchain", "CreatePeerDetails", reflect.TypeOf(blockchain.CreatePeerDetails{})),
	newTarget("blockchain", "UpdateBlockchainPlatformDetails", reflect.TypeOf(blockchain.UpdateBlockchainPlatformDetails{})),
	newTarget("blockchain", "UpdateOsnDetails", reflect.TypeOf(blockchain.UpdateOsnDetails{})),
	newTarget("blockchain", "UpdatePeerDetails", reflect.TypeOf(blockchain.UpdatePeerDetails{})),
	newTarget("blockchain", "BlockchainPlatform", reflect.TypeOf(blockchain.BlockchainPlatform{})),
	newTarget("blockchain", "BlockchainPlatformCollection", reflect.TypeOf(blockchain.BlockchainPlatformCollection{})),
	newTarget("blockchain", "Osn", reflect.TypeOf(blockchain.Osn{})),
	newTarget("blockchain", "OsnCollection", reflect.TypeOf(blockchain.OsnCollection{})),
	newTarget("blockchain", "Peer", reflect.TypeOf(blockchain.Peer{})),
	newTarget("blockchain", "PeerCollection", reflect.TypeOf(blockchain.PeerCollection{})),
	newTarget("blockchain", "BlockchainPlatformSummary", reflect.TypeOf(blockchain.BlockchainPlatformSummary{})),
	newTarget("blockchain", "OsnSummary", reflect.TypeOf(blockchain.OsnSummary{})),
	newTarget("blockchain", "PeerSummary", reflect.TypeOf(blockchain.PeerSummary{})),

	// Budget CRD support
	newTarget("budget", "CreateBudgetDetails", reflect.TypeOf(budget.CreateBudgetDetails{})),
	newTarget("budget", "UpdateBudgetDetails", reflect.TypeOf(budget.UpdateBudgetDetails{})),
	newTarget("budget", "Budget", reflect.TypeOf(budget.Budget{})),
	newTarget("budget", "BudgetSummary", reflect.TypeOf(budget.BudgetSummary{})),

	// Capacitymanagement CRD support
	newTarget("capacitymanagement", "CreateOccCapacityRequestDetails", reflect.TypeOf(capacitymanagement.CreateOccCapacityRequestDetails{})),
	newTarget("capacitymanagement", "UpdateOccCapacityRequestDetails", reflect.TypeOf(capacitymanagement.UpdateOccCapacityRequestDetails{})),
	newTarget("capacitymanagement", "OccCapacityRequest", reflect.TypeOf(capacitymanagement.OccCapacityRequest{})),
	newTarget("capacitymanagement", "OccCapacityRequestCollection", reflect.TypeOf(capacitymanagement.OccCapacityRequestCollection{})),
	newTarget("capacitymanagement", "OccCapacityRequestSummary", reflect.TypeOf(capacitymanagement.OccCapacityRequestSummary{})),

	// Cloudbridge CRD support
	newTarget("cloudbridge", "CreateAgentDependencyDetails", reflect.TypeOf(cloudbridge.CreateAgentDependencyDetails{})),
	newTarget("cloudbridge", "CreateAgentDetails", reflect.TypeOf(cloudbridge.CreateAgentDetails{})),
	newTarget("cloudbridge", "CreateDiscoveryScheduleDetails", reflect.TypeOf(cloudbridge.CreateDiscoveryScheduleDetails{})),
	newTarget("cloudbridge", "CreateEnvironmentDetails", reflect.TypeOf(cloudbridge.CreateEnvironmentDetails{})),
	newTarget("cloudbridge", "CreateInventoryDetails", reflect.TypeOf(cloudbridge.CreateInventoryDetails{})),
	newTarget("cloudbridge", "UpdateAgentDependencyDetails", reflect.TypeOf(cloudbridge.UpdateAgentDependencyDetails{})),
	newTarget("cloudbridge", "UpdateAgentDetails", reflect.TypeOf(cloudbridge.UpdateAgentDetails{})),
	newTarget("cloudbridge", "UpdateDiscoveryScheduleDetails", reflect.TypeOf(cloudbridge.UpdateDiscoveryScheduleDetails{})),
	newTarget("cloudbridge", "UpdateEnvironmentDetails", reflect.TypeOf(cloudbridge.UpdateEnvironmentDetails{})),
	newTarget("cloudbridge", "UpdateInventoryDetails", reflect.TypeOf(cloudbridge.UpdateInventoryDetails{})),
	newTarget("cloudbridge", "Agent", reflect.TypeOf(cloudbridge.Agent{})),
	newTarget("cloudbridge", "AgentCollection", reflect.TypeOf(cloudbridge.AgentCollection{})),
	newTarget("cloudbridge", "AgentDependency", reflect.TypeOf(cloudbridge.AgentDependency{})),
	newTarget("cloudbridge", "AgentDependencyCollection", reflect.TypeOf(cloudbridge.AgentDependencyCollection{})),
	newTarget("cloudbridge", "AssetCollection", reflect.TypeOf(cloudbridge.AssetCollection{})),
	newTarget("cloudbridge", "AssetSourceCollection", reflect.TypeOf(cloudbridge.AssetSourceCollection{})),
	newTarget("cloudbridge", "DiscoverySchedule", reflect.TypeOf(cloudbridge.DiscoverySchedule{})),
	newTarget("cloudbridge", "DiscoveryScheduleCollection", reflect.TypeOf(cloudbridge.DiscoveryScheduleCollection{})),
	newTarget("cloudbridge", "Environment", reflect.TypeOf(cloudbridge.Environment{})),
	newTarget("cloudbridge", "EnvironmentCollection", reflect.TypeOf(cloudbridge.EnvironmentCollection{})),
	newTarget("cloudbridge", "Inventory", reflect.TypeOf(cloudbridge.Inventory{})),
	newTarget("cloudbridge", "InventoryCollection", reflect.TypeOf(cloudbridge.InventoryCollection{})),
	newTarget("cloudbridge", "AgentDependencySummary", reflect.TypeOf(cloudbridge.AgentDependencySummary{})),
	newTarget("cloudbridge", "AgentSummary", reflect.TypeOf(cloudbridge.AgentSummary{})),
	newTarget("cloudbridge", "AssetSummary", reflect.TypeOf(cloudbridge.AssetSummary{})),
	newTarget("cloudbridge", "DiscoveryScheduleSummary", reflect.TypeOf(cloudbridge.DiscoveryScheduleSummary{})),
	newTarget("cloudbridge", "EnvironmentSummary", reflect.TypeOf(cloudbridge.EnvironmentSummary{})),
	newTarget("cloudbridge", "InventorySummary", reflect.TypeOf(cloudbridge.InventorySummary{})),

	// Cloudguard CRD support
	newTarget("cloudguard", "CreateAdhocQueryDetails", reflect.TypeOf(cloudguard.CreateAdhocQueryDetails{})),
	newTarget("cloudguard", "CreateDataMaskRuleDetails", reflect.TypeOf(cloudguard.CreateDataMaskRuleDetails{})),
	newTarget("cloudguard", "CreateDataSourceDetails", reflect.TypeOf(cloudguard.CreateDataSourceDetails{})),
	newTarget("cloudguard", "CreateDetectorRecipeDetails", reflect.TypeOf(cloudguard.CreateDetectorRecipeDetails{})),
	newTarget("cloudguard", "CreateDetectorRecipeDetectorRuleDetails", reflect.TypeOf(cloudguard.CreateDetectorRecipeDetectorRuleDetails{})),
	newTarget("cloudguard", "CreateManagedListDetails", reflect.TypeOf(cloudguard.CreateManagedListDetails{})),
	newTarget("cloudguard", "CreateResponderRecipeDetails", reflect.TypeOf(cloudguard.CreateResponderRecipeDetails{})),
	newTarget("cloudguard", "CreateSavedQueryDetails", reflect.TypeOf(cloudguard.CreateSavedQueryDetails{})),
	newTarget("cloudguard", "CreateSecurityRecipeDetails", reflect.TypeOf(cloudguard.CreateSecurityRecipeDetails{})),
	newTarget("cloudguard", "CreateSecurityZoneDetails", reflect.TypeOf(cloudguard.CreateSecurityZoneDetails{})),
	newTarget("cloudguard", "CreateTargetDetails", reflect.TypeOf(cloudguard.CreateTargetDetails{})),
	newTarget("cloudguard", "CreateTargetDetectorRecipeDetails", reflect.TypeOf(cloudguard.CreateTargetDetectorRecipeDetails{})),
	newTarget("cloudguard", "CreateTargetResponderRecipeDetails", reflect.TypeOf(cloudguard.CreateTargetResponderRecipeDetails{})),
	newTarget("cloudguard", "CreateWlpAgentDetails", reflect.TypeOf(cloudguard.CreateWlpAgentDetails{})),
	newTarget("cloudguard", "UpdateDataMaskRuleDetails", reflect.TypeOf(cloudguard.UpdateDataMaskRuleDetails{})),
	newTarget("cloudguard", "UpdateDataSourceDetails", reflect.TypeOf(cloudguard.UpdateDataSourceDetails{})),
	newTarget("cloudguard", "UpdateDetectorRecipeDetails", reflect.TypeOf(cloudguard.UpdateDetectorRecipeDetails{})),
	newTarget("cloudguard", "UpdateDetectorRecipeDetectorRuleDetails", reflect.TypeOf(cloudguard.UpdateDetectorRecipeDetectorRuleDetails{})),
	newTarget("cloudguard", "UpdateManagedListDetails", reflect.TypeOf(cloudguard.UpdateManagedListDetails{})),
	newTarget("cloudguard", "UpdateResponderRecipeDetails", reflect.TypeOf(cloudguard.UpdateResponderRecipeDetails{})),
	newTarget("cloudguard", "UpdateSavedQueryDetails", reflect.TypeOf(cloudguard.UpdateSavedQueryDetails{})),
	newTarget("cloudguard", "UpdateSecurityRecipeDetails", reflect.TypeOf(cloudguard.UpdateSecurityRecipeDetails{})),
	newTarget("cloudguard", "UpdateSecurityZoneDetails", reflect.TypeOf(cloudguard.UpdateSecurityZoneDetails{})),
	newTarget("cloudguard", "UpdateTargetDetails", reflect.TypeOf(cloudguard.UpdateTargetDetails{})),
	newTarget("cloudguard", "UpdateTargetDetectorRecipeDetails", reflect.TypeOf(cloudguard.UpdateTargetDetectorRecipeDetails{})),
	newTarget("cloudguard", "UpdateTargetResponderRecipeDetails", reflect.TypeOf(cloudguard.UpdateTargetResponderRecipeDetails{})),
	newTarget("cloudguard", "UpdateWlpAgentDetails", reflect.TypeOf(cloudguard.UpdateWlpAgentDetails{})),
	newTarget("cloudguard", "AdhocQueryDetails", reflect.TypeOf(cloudguard.AdhocQueryDetails{})),
	newTarget("cloudguard", "AdhocQuery", reflect.TypeOf(cloudguard.AdhocQuery{})),
	newTarget("cloudguard", "AdhocQueryCollection", reflect.TypeOf(cloudguard.AdhocQueryCollection{})),
	newTarget("cloudguard", "DataMaskRule", reflect.TypeOf(cloudguard.DataMaskRule{})),
	newTarget("cloudguard", "DataMaskRuleCollection", reflect.TypeOf(cloudguard.DataMaskRuleCollection{})),
	newTarget("cloudguard", "DataSource", reflect.TypeOf(cloudguard.DataSource{})),
	newTarget("cloudguard", "DataSourceCollection", reflect.TypeOf(cloudguard.DataSourceCollection{})),
	newTarget("cloudguard", "DetectorRecipe", reflect.TypeOf(cloudguard.DetectorRecipe{})),
	newTarget("cloudguard", "DetectorRecipeCollection", reflect.TypeOf(cloudguard.DetectorRecipeCollection{})),
	newTarget("cloudguard", "DetectorRecipeDetectorRule", reflect.TypeOf(cloudguard.DetectorRecipeDetectorRule{})),
	newTarget("cloudguard", "DetectorRecipeDetectorRuleCollection", reflect.TypeOf(cloudguard.DetectorRecipeDetectorRuleCollection{})),
	newTarget("cloudguard", "ManagedList", reflect.TypeOf(cloudguard.ManagedList{})),
	newTarget("cloudguard", "ManagedListCollection", reflect.TypeOf(cloudguard.ManagedListCollection{})),
	newTarget("cloudguard", "ResponderRecipe", reflect.TypeOf(cloudguard.ResponderRecipe{})),
	newTarget("cloudguard", "ResponderRecipeCollection", reflect.TypeOf(cloudguard.ResponderRecipeCollection{})),
	newTarget("cloudguard", "SavedQuery", reflect.TypeOf(cloudguard.SavedQuery{})),
	newTarget("cloudguard", "SavedQueryCollection", reflect.TypeOf(cloudguard.SavedQueryCollection{})),
	newTarget("cloudguard", "SecurityRecipe", reflect.TypeOf(cloudguard.SecurityRecipe{})),
	newTarget("cloudguard", "SecurityRecipeCollection", reflect.TypeOf(cloudguard.SecurityRecipeCollection{})),
	newTarget("cloudguard", "SecurityZone", reflect.TypeOf(cloudguard.SecurityZone{})),
	newTarget("cloudguard", "SecurityZoneCollection", reflect.TypeOf(cloudguard.SecurityZoneCollection{})),
	newTarget("cloudguard", "Target", reflect.TypeOf(cloudguard.Target{})),
	newTarget("cloudguard", "TargetCollection", reflect.TypeOf(cloudguard.TargetCollection{})),
	newTarget("cloudguard", "TargetDetectorRecipe", reflect.TypeOf(cloudguard.TargetDetectorRecipe{})),
	newTarget("cloudguard", "TargetDetectorRecipeCollection", reflect.TypeOf(cloudguard.TargetDetectorRecipeCollection{})),
	newTarget("cloudguard", "TargetResponderRecipe", reflect.TypeOf(cloudguard.TargetResponderRecipe{})),
	newTarget("cloudguard", "TargetResponderRecipeCollection", reflect.TypeOf(cloudguard.TargetResponderRecipeCollection{})),
	newTarget("cloudguard", "WlpAgent", reflect.TypeOf(cloudguard.WlpAgent{})),
	newTarget("cloudguard", "WlpAgentCollection", reflect.TypeOf(cloudguard.WlpAgentCollection{})),
	newTarget("cloudguard", "AdhocQuerySummary", reflect.TypeOf(cloudguard.AdhocQuerySummary{})),
	newTarget("cloudguard", "DataMaskRuleSummary", reflect.TypeOf(cloudguard.DataMaskRuleSummary{})),
	newTarget("cloudguard", "DataSourceSummary", reflect.TypeOf(cloudguard.DataSourceSummary{})),
	newTarget("cloudguard", "DetectorRecipeDetectorRuleSummary", reflect.TypeOf(cloudguard.DetectorRecipeDetectorRuleSummary{})),
	newTarget("cloudguard", "DetectorRecipeSummary", reflect.TypeOf(cloudguard.DetectorRecipeSummary{})),
	newTarget("cloudguard", "ManagedListSummary", reflect.TypeOf(cloudguard.ManagedListSummary{})),
	newTarget("cloudguard", "ResponderRecipeSummary", reflect.TypeOf(cloudguard.ResponderRecipeSummary{})),
	newTarget("cloudguard", "SavedQuerySummary", reflect.TypeOf(cloudguard.SavedQuerySummary{})),
	newTarget("cloudguard", "SecurityRecipeSummary", reflect.TypeOf(cloudguard.SecurityRecipeSummary{})),
	newTarget("cloudguard", "SecurityZoneSummary", reflect.TypeOf(cloudguard.SecurityZoneSummary{})),
	newTarget("cloudguard", "TargetDetectorRecipeSummary", reflect.TypeOf(cloudguard.TargetDetectorRecipeSummary{})),
	newTarget("cloudguard", "TargetResponderRecipeSummary", reflect.TypeOf(cloudguard.TargetResponderRecipeSummary{})),
	newTarget("cloudguard", "TargetSummary", reflect.TypeOf(cloudguard.TargetSummary{})),
	newTarget("cloudguard", "WlpAgentSummary", reflect.TypeOf(cloudguard.WlpAgentSummary{})),

	// Cloudmigrations CRD support
	newTarget("cloudmigrations", "CreateMigrationAssetDetails", reflect.TypeOf(cloudmigrations.CreateMigrationAssetDetails{})),
	newTarget("cloudmigrations", "CreateMigrationDetails", reflect.TypeOf(cloudmigrations.CreateMigrationDetails{})),
	newTarget("cloudmigrations", "CreateMigrationPlanDetails", reflect.TypeOf(cloudmigrations.CreateMigrationPlanDetails{})),
	newTarget("cloudmigrations", "CreateReplicationScheduleDetails", reflect.TypeOf(cloudmigrations.CreateReplicationScheduleDetails{})),
	newTarget("cloudmigrations", "UpdateMigrationAssetDetails", reflect.TypeOf(cloudmigrations.UpdateMigrationAssetDetails{})),
	newTarget("cloudmigrations", "UpdateMigrationDetails", reflect.TypeOf(cloudmigrations.UpdateMigrationDetails{})),
	newTarget("cloudmigrations", "UpdateMigrationPlanDetails", reflect.TypeOf(cloudmigrations.UpdateMigrationPlanDetails{})),
	newTarget("cloudmigrations", "UpdateReplicationScheduleDetails", reflect.TypeOf(cloudmigrations.UpdateReplicationScheduleDetails{})),
	newTarget("cloudmigrations", "Migration", reflect.TypeOf(cloudmigrations.Migration{})),
	newTarget("cloudmigrations", "MigrationAsset", reflect.TypeOf(cloudmigrations.MigrationAsset{})),
	newTarget("cloudmigrations", "MigrationAssetCollection", reflect.TypeOf(cloudmigrations.MigrationAssetCollection{})),
	newTarget("cloudmigrations", "MigrationCollection", reflect.TypeOf(cloudmigrations.MigrationCollection{})),
	newTarget("cloudmigrations", "MigrationPlan", reflect.TypeOf(cloudmigrations.MigrationPlan{})),
	newTarget("cloudmigrations", "MigrationPlanCollection", reflect.TypeOf(cloudmigrations.MigrationPlanCollection{})),
	newTarget("cloudmigrations", "ReplicationSchedule", reflect.TypeOf(cloudmigrations.ReplicationSchedule{})),
	newTarget("cloudmigrations", "ReplicationScheduleCollection", reflect.TypeOf(cloudmigrations.ReplicationScheduleCollection{})),
	newTarget("cloudmigrations", "TargetAssetCollection", reflect.TypeOf(cloudmigrations.TargetAssetCollection{})),
	newTarget("cloudmigrations", "MigrationAssetSummary", reflect.TypeOf(cloudmigrations.MigrationAssetSummary{})),
	newTarget("cloudmigrations", "MigrationPlanSummary", reflect.TypeOf(cloudmigrations.MigrationPlanSummary{})),
	newTarget("cloudmigrations", "MigrationSummary", reflect.TypeOf(cloudmigrations.MigrationSummary{})),
	newTarget("cloudmigrations", "ReplicationScheduleSummary", reflect.TypeOf(cloudmigrations.ReplicationScheduleSummary{})),

	// Clusterplacementgroups CRD support
	newTarget("clusterplacementgroups", "CreateClusterPlacementGroupDetails", reflect.TypeOf(clusterplacementgroups.CreateClusterPlacementGroupDetails{})),
	newTarget("clusterplacementgroups", "UpdateClusterPlacementGroupDetails", reflect.TypeOf(clusterplacementgroups.UpdateClusterPlacementGroupDetails{})),
	newTarget("clusterplacementgroups", "ClusterPlacementGroup", reflect.TypeOf(clusterplacementgroups.ClusterPlacementGroup{})),
	newTarget("clusterplacementgroups", "ClusterPlacementGroupCollection", reflect.TypeOf(clusterplacementgroups.ClusterPlacementGroupCollection{})),
	newTarget("clusterplacementgroups", "ClusterPlacementGroupSummary", reflect.TypeOf(clusterplacementgroups.ClusterPlacementGroupSummary{})),

	// Computecloudatcustomer CRD support
	newTarget("computecloudatcustomer", "CreateCccInfrastructureDetails", reflect.TypeOf(computecloudatcustomer.CreateCccInfrastructureDetails{})),
	newTarget("computecloudatcustomer", "CreateCccUpgradeScheduleDetails", reflect.TypeOf(computecloudatcustomer.CreateCccUpgradeScheduleDetails{})),
	newTarget("computecloudatcustomer", "UpdateCccInfrastructureDetails", reflect.TypeOf(computecloudatcustomer.UpdateCccInfrastructureDetails{})),
	newTarget("computecloudatcustomer", "UpdateCccUpgradeScheduleDetails", reflect.TypeOf(computecloudatcustomer.UpdateCccUpgradeScheduleDetails{})),
	newTarget("computecloudatcustomer", "CccInfrastructure", reflect.TypeOf(computecloudatcustomer.CccInfrastructure{})),
	newTarget("computecloudatcustomer", "CccInfrastructureCollection", reflect.TypeOf(computecloudatcustomer.CccInfrastructureCollection{})),
	newTarget("computecloudatcustomer", "CccUpgradeSchedule", reflect.TypeOf(computecloudatcustomer.CccUpgradeSchedule{})),
	newTarget("computecloudatcustomer", "CccUpgradeScheduleCollection", reflect.TypeOf(computecloudatcustomer.CccUpgradeScheduleCollection{})),
	newTarget("computecloudatcustomer", "CccInfrastructureSummary", reflect.TypeOf(computecloudatcustomer.CccInfrastructureSummary{})),
	newTarget("computecloudatcustomer", "CccUpgradeScheduleSummary", reflect.TypeOf(computecloudatcustomer.CccUpgradeScheduleSummary{})),

	// Computeinstanceagent CRD support
	newTarget("computeinstanceagent", "InstanceAgentPlugin", reflect.TypeOf(computeinstanceagent.InstanceAgentPlugin{})),
	newTarget("computeinstanceagent", "InstanceAgentPluginSummary", reflect.TypeOf(computeinstanceagent.InstanceAgentPluginSummary{})),

	// Containerinstances CRD support
	newTarget("containerinstances", "CreateContainerInstanceDetails", reflect.TypeOf(containerinstances.CreateContainerInstanceDetails{})),
	newTarget("containerinstances", "UpdateContainerInstanceDetails", reflect.TypeOf(containerinstances.UpdateContainerInstanceDetails{})),
	newTarget("containerinstances", "ContainerInstance", reflect.TypeOf(containerinstances.ContainerInstance{})),
	newTarget("containerinstances", "ContainerInstanceCollection", reflect.TypeOf(containerinstances.ContainerInstanceCollection{})),
	newTarget("containerinstances", "ContainerInstanceSummary", reflect.TypeOf(containerinstances.ContainerInstanceSummary{})),

	// Dashboardservice CRD support
	newTarget("dashboardservice", "CreateDashboardGroupDetails", reflect.TypeOf(dashboardservice.CreateDashboardGroupDetails{})),
	newTarget("dashboardservice", "UpdateDashboardGroupDetails", reflect.TypeOf(dashboardservice.UpdateDashboardGroupDetails{})),
	newTarget("dashboardservice", "DashboardCollection", reflect.TypeOf(dashboardservice.DashboardCollection{})),
	newTarget("dashboardservice", "DashboardGroup", reflect.TypeOf(dashboardservice.DashboardGroup{})),
	newTarget("dashboardservice", "DashboardGroupCollection", reflect.TypeOf(dashboardservice.DashboardGroupCollection{})),
	newTarget("dashboardservice", "DashboardGroupSummary", reflect.TypeOf(dashboardservice.DashboardGroupSummary{})),
	newTarget("dashboardservice", "DashboardSummary", reflect.TypeOf(dashboardservice.DashboardSummary{})),

	// Databasemigration CRD support
	newTarget("databasemigration", "AssessmentCollection", reflect.TypeOf(databasemigration.AssessmentCollection{})),
	newTarget("databasemigration", "ConnectionCollection", reflect.TypeOf(databasemigration.ConnectionCollection{})),

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

	// Datacatalog CRD support
	newTarget("datacatalog", "CreateAttributeDetails", reflect.TypeOf(datacatalog.CreateAttributeDetails{})),
	newTarget("datacatalog", "CreateCatalogDetails", reflect.TypeOf(datacatalog.CreateCatalogDetails{})),
	newTarget("datacatalog", "CreateCatalogPrivateEndpointDetails", reflect.TypeOf(datacatalog.CreateCatalogPrivateEndpointDetails{})),
	newTarget("datacatalog", "CreateConnectionDetails", reflect.TypeOf(datacatalog.CreateConnectionDetails{})),
	newTarget("datacatalog", "CreateCustomPropertyDetails", reflect.TypeOf(datacatalog.CreateCustomPropertyDetails{})),
	newTarget("datacatalog", "CreateDataAssetDetails", reflect.TypeOf(datacatalog.CreateDataAssetDetails{})),
	newTarget("datacatalog", "CreateEntityDetails", reflect.TypeOf(datacatalog.CreateEntityDetails{})),
	newTarget("datacatalog", "CreateFolderDetails", reflect.TypeOf(datacatalog.CreateFolderDetails{})),
	newTarget("datacatalog", "CreateGlossaryDetails", reflect.TypeOf(datacatalog.CreateGlossaryDetails{})),
	newTarget("datacatalog", "CreateJobDefinitionDetails", reflect.TypeOf(datacatalog.CreateJobDefinitionDetails{})),
	newTarget("datacatalog", "CreateJobDetails", reflect.TypeOf(datacatalog.CreateJobDetails{})),
	newTarget("datacatalog", "CreateMetastoreDetails", reflect.TypeOf(datacatalog.CreateMetastoreDetails{})),
	newTarget("datacatalog", "CreateNamespaceDetails", reflect.TypeOf(datacatalog.CreateNamespaceDetails{})),
	newTarget("datacatalog", "CreatePatternDetails", reflect.TypeOf(datacatalog.CreatePatternDetails{})),
	newTarget("datacatalog", "CreateTermDetails", reflect.TypeOf(datacatalog.CreateTermDetails{})),
	newTarget("datacatalog", "CreateTermRelationshipDetails", reflect.TypeOf(datacatalog.CreateTermRelationshipDetails{})),
	newTarget("datacatalog", "UpdateAttributeDetails", reflect.TypeOf(datacatalog.UpdateAttributeDetails{})),
	newTarget("datacatalog", "UpdateCatalogDetails", reflect.TypeOf(datacatalog.UpdateCatalogDetails{})),
	newTarget("datacatalog", "UpdateCatalogPrivateEndpointDetails", reflect.TypeOf(datacatalog.UpdateCatalogPrivateEndpointDetails{})),
	newTarget("datacatalog", "UpdateConnectionDetails", reflect.TypeOf(datacatalog.UpdateConnectionDetails{})),
	newTarget("datacatalog", "UpdateCustomPropertyDetails", reflect.TypeOf(datacatalog.UpdateCustomPropertyDetails{})),
	newTarget("datacatalog", "UpdateDataAssetDetails", reflect.TypeOf(datacatalog.UpdateDataAssetDetails{})),
	newTarget("datacatalog", "UpdateEntityDetails", reflect.TypeOf(datacatalog.UpdateEntityDetails{})),
	newTarget("datacatalog", "UpdateFolderDetails", reflect.TypeOf(datacatalog.UpdateFolderDetails{})),
	newTarget("datacatalog", "UpdateGlossaryDetails", reflect.TypeOf(datacatalog.UpdateGlossaryDetails{})),
	newTarget("datacatalog", "UpdateJobDefinitionDetails", reflect.TypeOf(datacatalog.UpdateJobDefinitionDetails{})),
	newTarget("datacatalog", "UpdateJobDetails", reflect.TypeOf(datacatalog.UpdateJobDetails{})),
	newTarget("datacatalog", "UpdateMetastoreDetails", reflect.TypeOf(datacatalog.UpdateMetastoreDetails{})),
	newTarget("datacatalog", "UpdateNamespaceDetails", reflect.TypeOf(datacatalog.UpdateNamespaceDetails{})),
	newTarget("datacatalog", "UpdatePatternDetails", reflect.TypeOf(datacatalog.UpdatePatternDetails{})),
	newTarget("datacatalog", "UpdateTermDetails", reflect.TypeOf(datacatalog.UpdateTermDetails{})),
	newTarget("datacatalog", "UpdateTermRelationshipDetails", reflect.TypeOf(datacatalog.UpdateTermRelationshipDetails{})),
	newTarget("datacatalog", "Attribute", reflect.TypeOf(datacatalog.Attribute{})),
	newTarget("datacatalog", "AttributeCollection", reflect.TypeOf(datacatalog.AttributeCollection{})),
	newTarget("datacatalog", "AttributeTag", reflect.TypeOf(datacatalog.AttributeTag{})),
	newTarget("datacatalog", "AttributeTagCollection", reflect.TypeOf(datacatalog.AttributeTagCollection{})),
	newTarget("datacatalog", "Catalog", reflect.TypeOf(datacatalog.Catalog{})),
	newTarget("datacatalog", "CatalogPrivateEndpoint", reflect.TypeOf(datacatalog.CatalogPrivateEndpoint{})),
	newTarget("datacatalog", "Connection", reflect.TypeOf(datacatalog.Connection{})),
	newTarget("datacatalog", "ConnectionCollection", reflect.TypeOf(datacatalog.ConnectionCollection{})),
	newTarget("datacatalog", "CustomProperty", reflect.TypeOf(datacatalog.CustomProperty{})),
	newTarget("datacatalog", "CustomPropertyCollection", reflect.TypeOf(datacatalog.CustomPropertyCollection{})),
	newTarget("datacatalog", "DataAsset", reflect.TypeOf(datacatalog.DataAsset{})),
	newTarget("datacatalog", "DataAssetCollection", reflect.TypeOf(datacatalog.DataAssetCollection{})),
	newTarget("datacatalog", "DataAssetTag", reflect.TypeOf(datacatalog.DataAssetTag{})),
	newTarget("datacatalog", "DataAssetTagCollection", reflect.TypeOf(datacatalog.DataAssetTagCollection{})),
	newTarget("datacatalog", "Entity", reflect.TypeOf(datacatalog.Entity{})),
	newTarget("datacatalog", "EntityCollection", reflect.TypeOf(datacatalog.EntityCollection{})),
	newTarget("datacatalog", "EntityTag", reflect.TypeOf(datacatalog.EntityTag{})),
	newTarget("datacatalog", "EntityTagCollection", reflect.TypeOf(datacatalog.EntityTagCollection{})),
	newTarget("datacatalog", "Folder", reflect.TypeOf(datacatalog.Folder{})),
	newTarget("datacatalog", "FolderCollection", reflect.TypeOf(datacatalog.FolderCollection{})),
	newTarget("datacatalog", "FolderTag", reflect.TypeOf(datacatalog.FolderTag{})),
	newTarget("datacatalog", "FolderTagCollection", reflect.TypeOf(datacatalog.FolderTagCollection{})),
	newTarget("datacatalog", "Glossary", reflect.TypeOf(datacatalog.Glossary{})),
	newTarget("datacatalog", "GlossaryCollection", reflect.TypeOf(datacatalog.GlossaryCollection{})),
	newTarget("datacatalog", "Job", reflect.TypeOf(datacatalog.Job{})),
	newTarget("datacatalog", "JobCollection", reflect.TypeOf(datacatalog.JobCollection{})),
	newTarget("datacatalog", "JobDefinition", reflect.TypeOf(datacatalog.JobDefinition{})),
	newTarget("datacatalog", "JobDefinitionCollection", reflect.TypeOf(datacatalog.JobDefinitionCollection{})),
	newTarget("datacatalog", "Metastore", reflect.TypeOf(datacatalog.Metastore{})),
	newTarget("datacatalog", "Namespace", reflect.TypeOf(datacatalog.Namespace{})),
	newTarget("datacatalog", "NamespaceCollection", reflect.TypeOf(datacatalog.NamespaceCollection{})),
	newTarget("datacatalog", "Pattern", reflect.TypeOf(datacatalog.Pattern{})),
	newTarget("datacatalog", "PatternCollection", reflect.TypeOf(datacatalog.PatternCollection{})),
	newTarget("datacatalog", "Term", reflect.TypeOf(datacatalog.Term{})),
	newTarget("datacatalog", "TermCollection", reflect.TypeOf(datacatalog.TermCollection{})),
	newTarget("datacatalog", "TermRelationship", reflect.TypeOf(datacatalog.TermRelationship{})),
	newTarget("datacatalog", "TermRelationshipCollection", reflect.TypeOf(datacatalog.TermRelationshipCollection{})),
	newTarget("datacatalog", "AttributeSummary", reflect.TypeOf(datacatalog.AttributeSummary{})),
	newTarget("datacatalog", "AttributeTagSummary", reflect.TypeOf(datacatalog.AttributeTagSummary{})),
	newTarget("datacatalog", "CatalogPrivateEndpointSummary", reflect.TypeOf(datacatalog.CatalogPrivateEndpointSummary{})),
	newTarget("datacatalog", "CatalogSummary", reflect.TypeOf(datacatalog.CatalogSummary{})),
	newTarget("datacatalog", "ConnectionSummary", reflect.TypeOf(datacatalog.ConnectionSummary{})),
	newTarget("datacatalog", "CustomPropertySummary", reflect.TypeOf(datacatalog.CustomPropertySummary{})),
	newTarget("datacatalog", "DataAssetSummary", reflect.TypeOf(datacatalog.DataAssetSummary{})),
	newTarget("datacatalog", "DataAssetTagSummary", reflect.TypeOf(datacatalog.DataAssetTagSummary{})),
	newTarget("datacatalog", "EntitySummary", reflect.TypeOf(datacatalog.EntitySummary{})),
	newTarget("datacatalog", "EntityTagSummary", reflect.TypeOf(datacatalog.EntityTagSummary{})),
	newTarget("datacatalog", "FolderSummary", reflect.TypeOf(datacatalog.FolderSummary{})),
	newTarget("datacatalog", "FolderTagSummary", reflect.TypeOf(datacatalog.FolderTagSummary{})),
	newTarget("datacatalog", "GlossarySummary", reflect.TypeOf(datacatalog.GlossarySummary{})),
	newTarget("datacatalog", "JobDefinitionSummary", reflect.TypeOf(datacatalog.JobDefinitionSummary{})),
	newTarget("datacatalog", "JobSummary", reflect.TypeOf(datacatalog.JobSummary{})),
	newTarget("datacatalog", "MetastoreSummary", reflect.TypeOf(datacatalog.MetastoreSummary{})),
	newTarget("datacatalog", "NamespaceSummary", reflect.TypeOf(datacatalog.NamespaceSummary{})),
	newTarget("datacatalog", "PatternSummary", reflect.TypeOf(datacatalog.PatternSummary{})),
	newTarget("datacatalog", "TermRelationshipSummary", reflect.TypeOf(datacatalog.TermRelationshipSummary{})),
	newTarget("datacatalog", "TermSummary", reflect.TypeOf(datacatalog.TermSummary{})),

	// Dataflow CRD support
	newTarget("dataflow", "CreateApplicationDetails", reflect.TypeOf(dataflow.CreateApplicationDetails{})),
	newTarget("dataflow", "UpdateApplicationDetails", reflect.TypeOf(dataflow.UpdateApplicationDetails{})),
	newTarget("dataflow", "Application", reflect.TypeOf(dataflow.Application{})),
	newTarget("dataflow", "ApplicationSummary", reflect.TypeOf(dataflow.ApplicationSummary{})),

	// Dataintegration CRD support
	newTarget("dataintegration", "CreateApplicationDetails", reflect.TypeOf(dataintegration.CreateApplicationDetails{})),
	newTarget("dataintegration", "CreateConnectionValidationDetails", reflect.TypeOf(dataintegration.CreateConnectionValidationDetails{})),
	newTarget("dataintegration", "CreateCopyObjectRequestDetails", reflect.TypeOf(dataintegration.CreateCopyObjectRequestDetails{})),
	newTarget("dataintegration", "CreateDataFlowDetails", reflect.TypeOf(dataintegration.CreateDataFlowDetails{})),
	newTarget("dataintegration", "CreateDataFlowValidationDetails", reflect.TypeOf(dataintegration.CreateDataFlowValidationDetails{})),
	newTarget("dataintegration", "CreateDisApplicationDetails", reflect.TypeOf(dataintegration.CreateDisApplicationDetails{})),
	newTarget("dataintegration", "CreateExportRequestDetails", reflect.TypeOf(dataintegration.CreateExportRequestDetails{})),
	newTarget("dataintegration", "CreateExternalPublicationDetails", reflect.TypeOf(dataintegration.CreateExternalPublicationDetails{})),
	newTarget("dataintegration", "CreateExternalPublicationValidationDetails", reflect.TypeOf(dataintegration.CreateExternalPublicationValidationDetails{})),
	newTarget("dataintegration", "CreateFolderDetails", reflect.TypeOf(dataintegration.CreateFolderDetails{})),
	newTarget("dataintegration", "CreateFunctionLibraryDetails", reflect.TypeOf(dataintegration.CreateFunctionLibraryDetails{})),
	newTarget("dataintegration", "CreateImportRequestDetails", reflect.TypeOf(dataintegration.CreateImportRequestDetails{})),
	newTarget("dataintegration", "CreatePatchDetails", reflect.TypeOf(dataintegration.CreatePatchDetails{})),
	newTarget("dataintegration", "CreatePipelineDetails", reflect.TypeOf(dataintegration.CreatePipelineDetails{})),
	newTarget("dataintegration", "CreatePipelineValidationDetails", reflect.TypeOf(dataintegration.CreatePipelineValidationDetails{})),
	newTarget("dataintegration", "CreateProjectDetails", reflect.TypeOf(dataintegration.CreateProjectDetails{})),
	newTarget("dataintegration", "CreateScheduleDetails", reflect.TypeOf(dataintegration.CreateScheduleDetails{})),
	newTarget("dataintegration", "CreateTaskRunDetails", reflect.TypeOf(dataintegration.CreateTaskRunDetails{})),
	newTarget("dataintegration", "CreateTaskScheduleDetails", reflect.TypeOf(dataintegration.CreateTaskScheduleDetails{})),
	newTarget("dataintegration", "CreateUserDefinedFunctionDetails", reflect.TypeOf(dataintegration.CreateUserDefinedFunctionDetails{})),
	newTarget("dataintegration", "CreateUserDefinedFunctionValidationDetails", reflect.TypeOf(dataintegration.CreateUserDefinedFunctionValidationDetails{})),
	newTarget("dataintegration", "CreateWorkspaceDetails", reflect.TypeOf(dataintegration.CreateWorkspaceDetails{})),
	newTarget("dataintegration", "UpdateApplicationDetails", reflect.TypeOf(dataintegration.UpdateApplicationDetails{})),
	newTarget("dataintegration", "UpdateCopyObjectRequestDetails", reflect.TypeOf(dataintegration.UpdateCopyObjectRequestDetails{})),
	newTarget("dataintegration", "UpdateDataFlowDetails", reflect.TypeOf(dataintegration.UpdateDataFlowDetails{})),
	newTarget("dataintegration", "UpdateDisApplicationDetails", reflect.TypeOf(dataintegration.UpdateDisApplicationDetails{})),
	newTarget("dataintegration", "UpdateExportRequestDetails", reflect.TypeOf(dataintegration.UpdateExportRequestDetails{})),
	newTarget("dataintegration", "UpdateExternalPublicationDetails", reflect.TypeOf(dataintegration.UpdateExternalPublicationDetails{})),
	newTarget("dataintegration", "UpdateFolderDetails", reflect.TypeOf(dataintegration.UpdateFolderDetails{})),
	newTarget("dataintegration", "UpdateFunctionLibraryDetails", reflect.TypeOf(dataintegration.UpdateFunctionLibraryDetails{})),
	newTarget("dataintegration", "UpdateImportRequestDetails", reflect.TypeOf(dataintegration.UpdateImportRequestDetails{})),
	newTarget("dataintegration", "UpdatePipelineDetails", reflect.TypeOf(dataintegration.UpdatePipelineDetails{})),
	newTarget("dataintegration", "UpdateProjectDetails", reflect.TypeOf(dataintegration.UpdateProjectDetails{})),
	newTarget("dataintegration", "UpdateScheduleDetails", reflect.TypeOf(dataintegration.UpdateScheduleDetails{})),
	newTarget("dataintegration", "UpdateTaskRunDetails", reflect.TypeOf(dataintegration.UpdateTaskRunDetails{})),
	newTarget("dataintegration", "UpdateTaskScheduleDetails", reflect.TypeOf(dataintegration.UpdateTaskScheduleDetails{})),
	newTarget("dataintegration", "UpdateUserDefinedFunctionDetails", reflect.TypeOf(dataintegration.UpdateUserDefinedFunctionDetails{})),
	newTarget("dataintegration", "UpdateWorkspaceDetails", reflect.TypeOf(dataintegration.UpdateWorkspaceDetails{})),
	newTarget("dataintegration", "ApplicationDetails", reflect.TypeOf(dataintegration.ApplicationDetails{})),
	newTarget("dataintegration", "DataFlowDetails", reflect.TypeOf(dataintegration.DataFlowDetails{})),
	newTarget("dataintegration", "FolderDetails", reflect.TypeOf(dataintegration.FolderDetails{})),
	newTarget("dataintegration", "FunctionLibraryDetails", reflect.TypeOf(dataintegration.FunctionLibraryDetails{})),
	newTarget("dataintegration", "ProjectDetails", reflect.TypeOf(dataintegration.ProjectDetails{})),
	newTarget("dataintegration", "TaskRunDetails", reflect.TypeOf(dataintegration.TaskRunDetails{})),
	newTarget("dataintegration", "UserDefinedFunctionDetails", reflect.TypeOf(dataintegration.UserDefinedFunctionDetails{})),
	newTarget("dataintegration", "Application", reflect.TypeOf(dataintegration.Application{})),
	newTarget("dataintegration", "ConnectionValidation", reflect.TypeOf(dataintegration.ConnectionValidation{})),
	newTarget("dataintegration", "CopyObjectRequest", reflect.TypeOf(dataintegration.CopyObjectRequest{})),
	newTarget("dataintegration", "DataFlow", reflect.TypeOf(dataintegration.DataFlow{})),
	newTarget("dataintegration", "DataFlowValidation", reflect.TypeOf(dataintegration.DataFlowValidation{})),
	newTarget("dataintegration", "DisApplication", reflect.TypeOf(dataintegration.DisApplication{})),
	newTarget("dataintegration", "ExportRequest", reflect.TypeOf(dataintegration.ExportRequest{})),
	newTarget("dataintegration", "ExternalPublication", reflect.TypeOf(dataintegration.ExternalPublication{})),
	newTarget("dataintegration", "ExternalPublicationValidation", reflect.TypeOf(dataintegration.ExternalPublicationValidation{})),
	newTarget("dataintegration", "Folder", reflect.TypeOf(dataintegration.Folder{})),
	newTarget("dataintegration", "FunctionLibrary", reflect.TypeOf(dataintegration.FunctionLibrary{})),
	newTarget("dataintegration", "ImportRequest", reflect.TypeOf(dataintegration.ImportRequest{})),
	newTarget("dataintegration", "Patch", reflect.TypeOf(dataintegration.Patch{})),
	newTarget("dataintegration", "Pipeline", reflect.TypeOf(dataintegration.Pipeline{})),
	newTarget("dataintegration", "PipelineValidation", reflect.TypeOf(dataintegration.PipelineValidation{})),
	newTarget("dataintegration", "Project", reflect.TypeOf(dataintegration.Project{})),
	newTarget("dataintegration", "Schedule", reflect.TypeOf(dataintegration.Schedule{})),
	newTarget("dataintegration", "TaskRun", reflect.TypeOf(dataintegration.TaskRun{})),
	newTarget("dataintegration", "TaskSchedule", reflect.TypeOf(dataintegration.TaskSchedule{})),
	newTarget("dataintegration", "TaskValidation", reflect.TypeOf(dataintegration.TaskValidation{})),
	newTarget("dataintegration", "UserDefinedFunction", reflect.TypeOf(dataintegration.UserDefinedFunction{})),
	newTarget("dataintegration", "UserDefinedFunctionValidation", reflect.TypeOf(dataintegration.UserDefinedFunctionValidation{})),
	newTarget("dataintegration", "Workspace", reflect.TypeOf(dataintegration.Workspace{})),
	newTarget("dataintegration", "ApplicationSummary", reflect.TypeOf(dataintegration.ApplicationSummary{})),
	newTarget("dataintegration", "ConnectionValidationSummary", reflect.TypeOf(dataintegration.ConnectionValidationSummary{})),
	newTarget("dataintegration", "CopyObjectRequestSummary", reflect.TypeOf(dataintegration.CopyObjectRequestSummary{})),
	newTarget("dataintegration", "DataFlowSummary", reflect.TypeOf(dataintegration.DataFlowSummary{})),
	newTarget("dataintegration", "DataFlowValidationSummary", reflect.TypeOf(dataintegration.DataFlowValidationSummary{})),
	newTarget("dataintegration", "DisApplicationSummary", reflect.TypeOf(dataintegration.DisApplicationSummary{})),
	newTarget("dataintegration", "ExportRequestSummary", reflect.TypeOf(dataintegration.ExportRequestSummary{})),
	newTarget("dataintegration", "ExternalPublicationSummary", reflect.TypeOf(dataintegration.ExternalPublicationSummary{})),
	newTarget("dataintegration", "ExternalPublicationValidationSummary", reflect.TypeOf(dataintegration.ExternalPublicationValidationSummary{})),
	newTarget("dataintegration", "FolderSummary", reflect.TypeOf(dataintegration.FolderSummary{})),
	newTarget("dataintegration", "FunctionLibrarySummary", reflect.TypeOf(dataintegration.FunctionLibrarySummary{})),
	newTarget("dataintegration", "ImportRequestSummary", reflect.TypeOf(dataintegration.ImportRequestSummary{})),
	newTarget("dataintegration", "PatchSummary", reflect.TypeOf(dataintegration.PatchSummary{})),
	newTarget("dataintegration", "PipelineSummary", reflect.TypeOf(dataintegration.PipelineSummary{})),
	newTarget("dataintegration", "PipelineValidationSummary", reflect.TypeOf(dataintegration.PipelineValidationSummary{})),
	newTarget("dataintegration", "ProjectSummary", reflect.TypeOf(dataintegration.ProjectSummary{})),
	newTarget("dataintegration", "ScheduleSummary", reflect.TypeOf(dataintegration.ScheduleSummary{})),
	newTarget("dataintegration", "TaskRunSummary", reflect.TypeOf(dataintegration.TaskRunSummary{})),
	newTarget("dataintegration", "TaskScheduleSummary", reflect.TypeOf(dataintegration.TaskScheduleSummary{})),
	newTarget("dataintegration", "TaskValidationSummary", reflect.TypeOf(dataintegration.TaskValidationSummary{})),
	newTarget("dataintegration", "UserDefinedFunctionSummary", reflect.TypeOf(dataintegration.UserDefinedFunctionSummary{})),
	newTarget("dataintegration", "UserDefinedFunctionValidationSummary", reflect.TypeOf(dataintegration.UserDefinedFunctionValidationSummary{})),
	newTarget("dataintegration", "WorkspaceSummary", reflect.TypeOf(dataintegration.WorkspaceSummary{})),

	// Datalabelingservice CRD support
	newTarget("datalabelingservice", "CreateDatasetDetails", reflect.TypeOf(datalabelingservice.CreateDatasetDetails{})),
	newTarget("datalabelingservice", "UpdateDatasetDetails", reflect.TypeOf(datalabelingservice.UpdateDatasetDetails{})),
	newTarget("datalabelingservice", "Dataset", reflect.TypeOf(datalabelingservice.Dataset{})),
	newTarget("datalabelingservice", "DatasetCollection", reflect.TypeOf(datalabelingservice.DatasetCollection{})),
	newTarget("datalabelingservice", "DatasetSummary", reflect.TypeOf(datalabelingservice.DatasetSummary{})),

	// Datalabelingservicedataplane CRD support
	newTarget("datalabelingservicedataplane", "CreateAnnotationDetails", reflect.TypeOf(datalabelingservicedataplane.CreateAnnotationDetails{})),
	newTarget("datalabelingservicedataplane", "CreateRecordDetails", reflect.TypeOf(datalabelingservicedataplane.CreateRecordDetails{})),
	newTarget("datalabelingservicedataplane", "UpdateAnnotationDetails", reflect.TypeOf(datalabelingservicedataplane.UpdateAnnotationDetails{})),
	newTarget("datalabelingservicedataplane", "UpdateRecordDetails", reflect.TypeOf(datalabelingservicedataplane.UpdateRecordDetails{})),
	newTarget("datalabelingservicedataplane", "Annotation", reflect.TypeOf(datalabelingservicedataplane.Annotation{})),
	newTarget("datalabelingservicedataplane", "AnnotationCollection", reflect.TypeOf(datalabelingservicedataplane.AnnotationCollection{})),
	newTarget("datalabelingservicedataplane", "Record", reflect.TypeOf(datalabelingservicedataplane.Record{})),
	newTarget("datalabelingservicedataplane", "RecordCollection", reflect.TypeOf(datalabelingservicedataplane.RecordCollection{})),
	newTarget("datalabelingservicedataplane", "AnnotationSummary", reflect.TypeOf(datalabelingservicedataplane.AnnotationSummary{})),
	newTarget("datalabelingservicedataplane", "RecordSummary", reflect.TypeOf(datalabelingservicedataplane.RecordSummary{})),

	// Datasafe CRD support
	newTarget("datasafe", "CreateAlertPolicyDetails", reflect.TypeOf(datasafe.CreateAlertPolicyDetails{})),
	newTarget("datasafe", "CreateAlertPolicyRuleDetails", reflect.TypeOf(datasafe.CreateAlertPolicyRuleDetails{})),
	newTarget("datasafe", "CreateAttributeSetDetails", reflect.TypeOf(datasafe.CreateAttributeSetDetails{})),
	newTarget("datasafe", "CreateAuditArchiveRetrievalDetails", reflect.TypeOf(datasafe.CreateAuditArchiveRetrievalDetails{})),
	newTarget("datasafe", "CreateAuditProfileDetails", reflect.TypeOf(datasafe.CreateAuditProfileDetails{})),
	newTarget("datasafe", "CreateDataSafePrivateEndpointDetails", reflect.TypeOf(datasafe.CreateDataSafePrivateEndpointDetails{})),
	newTarget("datasafe", "CreateDiscoveryJobDetails", reflect.TypeOf(datasafe.CreateDiscoveryJobDetails{})),
	newTarget("datasafe", "CreateLibraryMaskingFormatDetails", reflect.TypeOf(datasafe.CreateLibraryMaskingFormatDetails{})),
	newTarget("datasafe", "CreateMaskingColumnDetails", reflect.TypeOf(datasafe.CreateMaskingColumnDetails{})),
	newTarget("datasafe", "CreateMaskingPolicyDetails", reflect.TypeOf(datasafe.CreateMaskingPolicyDetails{})),
	newTarget("datasafe", "CreateOnPremConnectorDetails", reflect.TypeOf(datasafe.CreateOnPremConnectorDetails{})),
	newTarget("datasafe", "CreatePeerTargetDatabaseDetails", reflect.TypeOf(datasafe.CreatePeerTargetDatabaseDetails{})),
	newTarget("datasafe", "CreateReferentialRelationDetails", reflect.TypeOf(datasafe.CreateReferentialRelationDetails{})),
	newTarget("datasafe", "CreateReportDefinitionDetails", reflect.TypeOf(datasafe.CreateReportDefinitionDetails{})),
	newTarget("datasafe", "CreateSdmMaskingPolicyDifferenceDetails", reflect.TypeOf(datasafe.CreateSdmMaskingPolicyDifferenceDetails{})),
	newTarget("datasafe", "CreateSecurityAssessmentDetails", reflect.TypeOf(datasafe.CreateSecurityAssessmentDetails{})),
	newTarget("datasafe", "CreateSecurityPolicyConfigDetails", reflect.TypeOf(datasafe.CreateSecurityPolicyConfigDetails{})),
	newTarget("datasafe", "CreateSecurityPolicyDeploymentDetails", reflect.TypeOf(datasafe.CreateSecurityPolicyDeploymentDetails{})),
	newTarget("datasafe", "CreateSecurityPolicyDetails", reflect.TypeOf(datasafe.CreateSecurityPolicyDetails{})),
	newTarget("datasafe", "CreateSensitiveColumnDetails", reflect.TypeOf(datasafe.CreateSensitiveColumnDetails{})),
	newTarget("datasafe", "CreateSensitiveDataModelDetails", reflect.TypeOf(datasafe.CreateSensitiveDataModelDetails{})),
	newTarget("datasafe", "CreateSensitiveTypeGroupDetails", reflect.TypeOf(datasafe.CreateSensitiveTypeGroupDetails{})),
	newTarget("datasafe", "CreateSensitiveTypesExportDetails", reflect.TypeOf(datasafe.CreateSensitiveTypesExportDetails{})),
	newTarget("datasafe", "CreateSqlCollectionDetails", reflect.TypeOf(datasafe.CreateSqlCollectionDetails{})),
	newTarget("datasafe", "CreateTargetAlertPolicyAssociationDetails", reflect.TypeOf(datasafe.CreateTargetAlertPolicyAssociationDetails{})),
	newTarget("datasafe", "CreateTargetDatabaseDetails", reflect.TypeOf(datasafe.CreateTargetDatabaseDetails{})),
	newTarget("datasafe", "CreateTargetDatabaseGroupDetails", reflect.TypeOf(datasafe.CreateTargetDatabaseGroupDetails{})),
	newTarget("datasafe", "CreateUnifiedAuditPolicyDetails", reflect.TypeOf(datasafe.CreateUnifiedAuditPolicyDetails{})),
	newTarget("datasafe", "CreateUserAssessmentDetails", reflect.TypeOf(datasafe.CreateUserAssessmentDetails{})),
	newTarget("datasafe", "UpdateAlertPolicyDetails", reflect.TypeOf(datasafe.UpdateAlertPolicyDetails{})),
	newTarget("datasafe", "UpdateAlertPolicyRuleDetails", reflect.TypeOf(datasafe.UpdateAlertPolicyRuleDetails{})),
	newTarget("datasafe", "UpdateAttributeSetDetails", reflect.TypeOf(datasafe.UpdateAttributeSetDetails{})),
	newTarget("datasafe", "UpdateAuditArchiveRetrievalDetails", reflect.TypeOf(datasafe.UpdateAuditArchiveRetrievalDetails{})),
	newTarget("datasafe", "UpdateAuditProfileDetails", reflect.TypeOf(datasafe.UpdateAuditProfileDetails{})),
	newTarget("datasafe", "UpdateDataSafePrivateEndpointDetails", reflect.TypeOf(datasafe.UpdateDataSafePrivateEndpointDetails{})),
	newTarget("datasafe", "UpdateLibraryMaskingFormatDetails", reflect.TypeOf(datasafe.UpdateLibraryMaskingFormatDetails{})),
	newTarget("datasafe", "UpdateMaskingColumnDetails", reflect.TypeOf(datasafe.UpdateMaskingColumnDetails{})),
	newTarget("datasafe", "UpdateMaskingPolicyDetails", reflect.TypeOf(datasafe.UpdateMaskingPolicyDetails{})),
	newTarget("datasafe", "UpdateOnPremConnectorDetails", reflect.TypeOf(datasafe.UpdateOnPremConnectorDetails{})),
	newTarget("datasafe", "UpdatePeerTargetDatabaseDetails", reflect.TypeOf(datasafe.UpdatePeerTargetDatabaseDetails{})),
	newTarget("datasafe", "UpdateReportDefinitionDetails", reflect.TypeOf(datasafe.UpdateReportDefinitionDetails{})),
	newTarget("datasafe", "UpdateSdmMaskingPolicyDifferenceDetails", reflect.TypeOf(datasafe.UpdateSdmMaskingPolicyDifferenceDetails{})),
	newTarget("datasafe", "UpdateSecurityAssessmentDetails", reflect.TypeOf(datasafe.UpdateSecurityAssessmentDetails{})),
	newTarget("datasafe", "UpdateSecurityPolicyConfigDetails", reflect.TypeOf(datasafe.UpdateSecurityPolicyConfigDetails{})),
	newTarget("datasafe", "UpdateSecurityPolicyDeploymentDetails", reflect.TypeOf(datasafe.UpdateSecurityPolicyDeploymentDetails{})),
	newTarget("datasafe", "UpdateSecurityPolicyDetails", reflect.TypeOf(datasafe.UpdateSecurityPolicyDetails{})),
	newTarget("datasafe", "UpdateSensitiveColumnDetails", reflect.TypeOf(datasafe.UpdateSensitiveColumnDetails{})),
	newTarget("datasafe", "UpdateSensitiveDataModelDetails", reflect.TypeOf(datasafe.UpdateSensitiveDataModelDetails{})),
	newTarget("datasafe", "UpdateSensitiveTypeGroupDetails", reflect.TypeOf(datasafe.UpdateSensitiveTypeGroupDetails{})),
	newTarget("datasafe", "UpdateSensitiveTypesExportDetails", reflect.TypeOf(datasafe.UpdateSensitiveTypesExportDetails{})),
	newTarget("datasafe", "UpdateSqlCollectionDetails", reflect.TypeOf(datasafe.UpdateSqlCollectionDetails{})),
	newTarget("datasafe", "UpdateTargetAlertPolicyAssociationDetails", reflect.TypeOf(datasafe.UpdateTargetAlertPolicyAssociationDetails{})),
	newTarget("datasafe", "UpdateTargetDatabaseDetails", reflect.TypeOf(datasafe.UpdateTargetDatabaseDetails{})),
	newTarget("datasafe", "UpdateTargetDatabaseGroupDetails", reflect.TypeOf(datasafe.UpdateTargetDatabaseGroupDetails{})),
	newTarget("datasafe", "UpdateUnifiedAuditPolicyDetails", reflect.TypeOf(datasafe.UpdateUnifiedAuditPolicyDetails{})),
	newTarget("datasafe", "UpdateUserAssessmentDetails", reflect.TypeOf(datasafe.UpdateUserAssessmentDetails{})),
	newTarget("datasafe", "AlertPolicy", reflect.TypeOf(datasafe.AlertPolicy{})),
	newTarget("datasafe", "AlertPolicyCollection", reflect.TypeOf(datasafe.AlertPolicyCollection{})),
	newTarget("datasafe", "AlertPolicyRule", reflect.TypeOf(datasafe.AlertPolicyRule{})),
	newTarget("datasafe", "AlertPolicyRuleCollection", reflect.TypeOf(datasafe.AlertPolicyRuleCollection{})),
	newTarget("datasafe", "AttributeSet", reflect.TypeOf(datasafe.AttributeSet{})),
	newTarget("datasafe", "AttributeSetCollection", reflect.TypeOf(datasafe.AttributeSetCollection{})),
	newTarget("datasafe", "AuditArchiveRetrieval", reflect.TypeOf(datasafe.AuditArchiveRetrieval{})),
	newTarget("datasafe", "AuditArchiveRetrievalCollection", reflect.TypeOf(datasafe.AuditArchiveRetrievalCollection{})),
	newTarget("datasafe", "AuditProfile", reflect.TypeOf(datasafe.AuditProfile{})),
	newTarget("datasafe", "AuditProfileCollection", reflect.TypeOf(datasafe.AuditProfileCollection{})),
	newTarget("datasafe", "DataSafePrivateEndpoint", reflect.TypeOf(datasafe.DataSafePrivateEndpoint{})),
	newTarget("datasafe", "DiscoveryJob", reflect.TypeOf(datasafe.DiscoveryJob{})),
	newTarget("datasafe", "DiscoveryJobCollection", reflect.TypeOf(datasafe.DiscoveryJobCollection{})),
	newTarget("datasafe", "LibraryMaskingFormat", reflect.TypeOf(datasafe.LibraryMaskingFormat{})),
	newTarget("datasafe", "LibraryMaskingFormatCollection", reflect.TypeOf(datasafe.LibraryMaskingFormatCollection{})),
	newTarget("datasafe", "LibraryMaskingFormatEntry", reflect.TypeOf(datasafe.LibraryMaskingFormatEntry{})),
	newTarget("datasafe", "MaskingColumn", reflect.TypeOf(datasafe.MaskingColumn{})),
	newTarget("datasafe", "MaskingColumnCollection", reflect.TypeOf(datasafe.MaskingColumnCollection{})),
	newTarget("datasafe", "MaskingPolicy", reflect.TypeOf(datasafe.MaskingPolicy{})),
	newTarget("datasafe", "MaskingPolicyCollection", reflect.TypeOf(datasafe.MaskingPolicyCollection{})),
	newTarget("datasafe", "OnPremConnector", reflect.TypeOf(datasafe.OnPremConnector{})),
	newTarget("datasafe", "PeerTargetDatabase", reflect.TypeOf(datasafe.PeerTargetDatabase{})),
	newTarget("datasafe", "PeerTargetDatabaseCollection", reflect.TypeOf(datasafe.PeerTargetDatabaseCollection{})),
	newTarget("datasafe", "ReferentialRelation", reflect.TypeOf(datasafe.ReferentialRelation{})),
	newTarget("datasafe", "ReferentialRelationCollection", reflect.TypeOf(datasafe.ReferentialRelationCollection{})),
	newTarget("datasafe", "ReportDefinition", reflect.TypeOf(datasafe.ReportDefinition{})),
	newTarget("datasafe", "ReportDefinitionCollection", reflect.TypeOf(datasafe.ReportDefinitionCollection{})),
	newTarget("datasafe", "SdmMaskingPolicyDifference", reflect.TypeOf(datasafe.SdmMaskingPolicyDifference{})),
	newTarget("datasafe", "SdmMaskingPolicyDifferenceCollection", reflect.TypeOf(datasafe.SdmMaskingPolicyDifferenceCollection{})),
	newTarget("datasafe", "SecurityAssessment", reflect.TypeOf(datasafe.SecurityAssessment{})),
	newTarget("datasafe", "SecurityPolicy", reflect.TypeOf(datasafe.SecurityPolicy{})),
	newTarget("datasafe", "SecurityPolicyCollection", reflect.TypeOf(datasafe.SecurityPolicyCollection{})),
	newTarget("datasafe", "SecurityPolicyConfig", reflect.TypeOf(datasafe.SecurityPolicyConfig{})),
	newTarget("datasafe", "SecurityPolicyConfigCollection", reflect.TypeOf(datasafe.SecurityPolicyConfigCollection{})),
	newTarget("datasafe", "SecurityPolicyDeployment", reflect.TypeOf(datasafe.SecurityPolicyDeployment{})),
	newTarget("datasafe", "SecurityPolicyDeploymentCollection", reflect.TypeOf(datasafe.SecurityPolicyDeploymentCollection{})),
	newTarget("datasafe", "SensitiveColumn", reflect.TypeOf(datasafe.SensitiveColumn{})),
	newTarget("datasafe", "SensitiveColumnCollection", reflect.TypeOf(datasafe.SensitiveColumnCollection{})),
	newTarget("datasafe", "SensitiveDataModel", reflect.TypeOf(datasafe.SensitiveDataModel{})),
	newTarget("datasafe", "SensitiveDataModelCollection", reflect.TypeOf(datasafe.SensitiveDataModelCollection{})),
	newTarget("datasafe", "SensitiveTypeCollection", reflect.TypeOf(datasafe.SensitiveTypeCollection{})),
	newTarget("datasafe", "SensitiveTypeGroup", reflect.TypeOf(datasafe.SensitiveTypeGroup{})),
	newTarget("datasafe", "SensitiveTypeGroupCollection", reflect.TypeOf(datasafe.SensitiveTypeGroupCollection{})),
	newTarget("datasafe", "SensitiveTypesExport", reflect.TypeOf(datasafe.SensitiveTypesExport{})),
	newTarget("datasafe", "SensitiveTypesExportCollection", reflect.TypeOf(datasafe.SensitiveTypesExportCollection{})),
	newTarget("datasafe", "SqlCollection", reflect.TypeOf(datasafe.SqlCollection{})),
	newTarget("datasafe", "SqlCollectionCollection", reflect.TypeOf(datasafe.SqlCollectionCollection{})),
	newTarget("datasafe", "TargetAlertPolicyAssociation", reflect.TypeOf(datasafe.TargetAlertPolicyAssociation{})),
	newTarget("datasafe", "TargetAlertPolicyAssociationCollection", reflect.TypeOf(datasafe.TargetAlertPolicyAssociationCollection{})),
	newTarget("datasafe", "TargetDatabase", reflect.TypeOf(datasafe.TargetDatabase{})),
	newTarget("datasafe", "TargetDatabaseGroup", reflect.TypeOf(datasafe.TargetDatabaseGroup{})),
	newTarget("datasafe", "TargetDatabaseGroupCollection", reflect.TypeOf(datasafe.TargetDatabaseGroupCollection{})),
	newTarget("datasafe", "UnifiedAuditPolicy", reflect.TypeOf(datasafe.UnifiedAuditPolicy{})),
	newTarget("datasafe", "UnifiedAuditPolicyCollection", reflect.TypeOf(datasafe.UnifiedAuditPolicyCollection{})),
	newTarget("datasafe", "UserAssessment", reflect.TypeOf(datasafe.UserAssessment{})),
	newTarget("datasafe", "AlertPolicyRuleSummary", reflect.TypeOf(datasafe.AlertPolicyRuleSummary{})),
	newTarget("datasafe", "AlertPolicySummary", reflect.TypeOf(datasafe.AlertPolicySummary{})),
	newTarget("datasafe", "AttributeSetSummary", reflect.TypeOf(datasafe.AttributeSetSummary{})),
	newTarget("datasafe", "AuditArchiveRetrievalSummary", reflect.TypeOf(datasafe.AuditArchiveRetrievalSummary{})),
	newTarget("datasafe", "AuditProfileSummary", reflect.TypeOf(datasafe.AuditProfileSummary{})),
	newTarget("datasafe", "DataSafePrivateEndpointSummary", reflect.TypeOf(datasafe.DataSafePrivateEndpointSummary{})),
	newTarget("datasafe", "DiscoveryJobSummary", reflect.TypeOf(datasafe.DiscoveryJobSummary{})),
	newTarget("datasafe", "LibraryMaskingFormatSummary", reflect.TypeOf(datasafe.LibraryMaskingFormatSummary{})),
	newTarget("datasafe", "MaskingColumnSummary", reflect.TypeOf(datasafe.MaskingColumnSummary{})),
	newTarget("datasafe", "MaskingPolicySummary", reflect.TypeOf(datasafe.MaskingPolicySummary{})),
	newTarget("datasafe", "OnPremConnectorSummary", reflect.TypeOf(datasafe.OnPremConnectorSummary{})),
	newTarget("datasafe", "PeerTargetDatabaseSummary", reflect.TypeOf(datasafe.PeerTargetDatabaseSummary{})),
	newTarget("datasafe", "ReferentialRelationSummary", reflect.TypeOf(datasafe.ReferentialRelationSummary{})),
	newTarget("datasafe", "ReportDefinitionSummary", reflect.TypeOf(datasafe.ReportDefinitionSummary{})),
	newTarget("datasafe", "SdmMaskingPolicyDifferenceSummary", reflect.TypeOf(datasafe.SdmMaskingPolicyDifferenceSummary{})),
	newTarget("datasafe", "SecurityAssessmentSummary", reflect.TypeOf(datasafe.SecurityAssessmentSummary{})),
	newTarget("datasafe", "SecurityPolicyConfigSummary", reflect.TypeOf(datasafe.SecurityPolicyConfigSummary{})),
	newTarget("datasafe", "SecurityPolicyDeploymentSummary", reflect.TypeOf(datasafe.SecurityPolicyDeploymentSummary{})),
	newTarget("datasafe", "SecurityPolicySummary", reflect.TypeOf(datasafe.SecurityPolicySummary{})),
	newTarget("datasafe", "SensitiveColumnSummary", reflect.TypeOf(datasafe.SensitiveColumnSummary{})),
	newTarget("datasafe", "SensitiveDataModelSummary", reflect.TypeOf(datasafe.SensitiveDataModelSummary{})),
	newTarget("datasafe", "SensitiveTypeGroupSummary", reflect.TypeOf(datasafe.SensitiveTypeGroupSummary{})),
	newTarget("datasafe", "SensitiveTypeSummary", reflect.TypeOf(datasafe.SensitiveTypeSummary{})),
	newTarget("datasafe", "SensitiveTypesExportSummary", reflect.TypeOf(datasafe.SensitiveTypesExportSummary{})),
	newTarget("datasafe", "SqlCollectionSummary", reflect.TypeOf(datasafe.SqlCollectionSummary{})),
	newTarget("datasafe", "TargetAlertPolicyAssociationSummary", reflect.TypeOf(datasafe.TargetAlertPolicyAssociationSummary{})),
	newTarget("datasafe", "TargetDatabaseGroupSummary", reflect.TypeOf(datasafe.TargetDatabaseGroupSummary{})),
	newTarget("datasafe", "TargetDatabaseSummary", reflect.TypeOf(datasafe.TargetDatabaseSummary{})),
	newTarget("datasafe", "UnifiedAuditPolicySummary", reflect.TypeOf(datasafe.UnifiedAuditPolicySummary{})),
	newTarget("datasafe", "UserAssessmentSummary", reflect.TypeOf(datasafe.UserAssessmentSummary{})),

	// Datascience CRD support
	newTarget("datascience", "CreateProjectDetails", reflect.TypeOf(datascience.CreateProjectDetails{})),
	newTarget("datascience", "UpdateProjectDetails", reflect.TypeOf(datascience.UpdateProjectDetails{})),
	newTarget("datascience", "Project", reflect.TypeOf(datascience.Project{})),
	newTarget("datascience", "ProjectSummary", reflect.TypeOf(datascience.ProjectSummary{})),

	// Dbmulticloud CRD support
	newTarget("dbmulticloud", "CreateMultiCloudResourceDiscoveryDetails", reflect.TypeOf(dbmulticloud.CreateMultiCloudResourceDiscoveryDetails{})),
	newTarget("dbmulticloud", "CreateOracleDbAwsIdentityConnectorDetails", reflect.TypeOf(dbmulticloud.CreateOracleDbAwsIdentityConnectorDetails{})),
	newTarget("dbmulticloud", "CreateOracleDbAwsKeyDetails", reflect.TypeOf(dbmulticloud.CreateOracleDbAwsKeyDetails{})),
	newTarget("dbmulticloud", "CreateOracleDbAzureBlobContainerDetails", reflect.TypeOf(dbmulticloud.CreateOracleDbAzureBlobContainerDetails{})),
	newTarget("dbmulticloud", "CreateOracleDbAzureBlobMountDetails", reflect.TypeOf(dbmulticloud.CreateOracleDbAzureBlobMountDetails{})),
	newTarget("dbmulticloud", "CreateOracleDbAzureConnectorDetails", reflect.TypeOf(dbmulticloud.CreateOracleDbAzureConnectorDetails{})),
	newTarget("dbmulticloud", "CreateOracleDbAzureVaultAssociationDetails", reflect.TypeOf(dbmulticloud.CreateOracleDbAzureVaultAssociationDetails{})),
	newTarget("dbmulticloud", "CreateOracleDbAzureVaultDetails", reflect.TypeOf(dbmulticloud.CreateOracleDbAzureVaultDetails{})),
	newTarget("dbmulticloud", "CreateOracleDbGcpIdentityConnectorDetails", reflect.TypeOf(dbmulticloud.CreateOracleDbGcpIdentityConnectorDetails{})),
	newTarget("dbmulticloud", "CreateOracleDbGcpKeyRingDetails", reflect.TypeOf(dbmulticloud.CreateOracleDbGcpKeyRingDetails{})),
	newTarget("dbmulticloud", "UpdateMultiCloudResourceDiscoveryDetails", reflect.TypeOf(dbmulticloud.UpdateMultiCloudResourceDiscoveryDetails{})),
	newTarget("dbmulticloud", "UpdateOracleDbAwsIdentityConnectorDetails", reflect.TypeOf(dbmulticloud.UpdateOracleDbAwsIdentityConnectorDetails{})),
	newTarget("dbmulticloud", "UpdateOracleDbAwsKeyDetails", reflect.TypeOf(dbmulticloud.UpdateOracleDbAwsKeyDetails{})),
	newTarget("dbmulticloud", "UpdateOracleDbAzureBlobContainerDetails", reflect.TypeOf(dbmulticloud.UpdateOracleDbAzureBlobContainerDetails{})),
	newTarget("dbmulticloud", "UpdateOracleDbAzureBlobMountDetails", reflect.TypeOf(dbmulticloud.UpdateOracleDbAzureBlobMountDetails{})),
	newTarget("dbmulticloud", "UpdateOracleDbAzureConnectorDetails", reflect.TypeOf(dbmulticloud.UpdateOracleDbAzureConnectorDetails{})),
	newTarget("dbmulticloud", "UpdateOracleDbAzureVaultAssociationDetails", reflect.TypeOf(dbmulticloud.UpdateOracleDbAzureVaultAssociationDetails{})),
	newTarget("dbmulticloud", "UpdateOracleDbAzureVaultDetails", reflect.TypeOf(dbmulticloud.UpdateOracleDbAzureVaultDetails{})),
	newTarget("dbmulticloud", "UpdateOracleDbGcpIdentityConnectorDetails", reflect.TypeOf(dbmulticloud.UpdateOracleDbGcpIdentityConnectorDetails{})),
	newTarget("dbmulticloud", "UpdateOracleDbGcpKeyRingDetails", reflect.TypeOf(dbmulticloud.UpdateOracleDbGcpKeyRingDetails{})),
	newTarget("dbmulticloud", "MultiCloudResourceDiscovery", reflect.TypeOf(dbmulticloud.MultiCloudResourceDiscovery{})),
	newTarget("dbmulticloud", "OracleDbAwsIdentityConnector", reflect.TypeOf(dbmulticloud.OracleDbAwsIdentityConnector{})),
	newTarget("dbmulticloud", "OracleDbAwsKey", reflect.TypeOf(dbmulticloud.OracleDbAwsKey{})),
	newTarget("dbmulticloud", "OracleDbAzureBlobContainer", reflect.TypeOf(dbmulticloud.OracleDbAzureBlobContainer{})),
	newTarget("dbmulticloud", "OracleDbAzureBlobMount", reflect.TypeOf(dbmulticloud.OracleDbAzureBlobMount{})),
	newTarget("dbmulticloud", "OracleDbAzureConnector", reflect.TypeOf(dbmulticloud.OracleDbAzureConnector{})),
	newTarget("dbmulticloud", "OracleDbAzureVault", reflect.TypeOf(dbmulticloud.OracleDbAzureVault{})),
	newTarget("dbmulticloud", "OracleDbAzureVaultAssociation", reflect.TypeOf(dbmulticloud.OracleDbAzureVaultAssociation{})),
	newTarget("dbmulticloud", "OracleDbGcpIdentityConnector", reflect.TypeOf(dbmulticloud.OracleDbGcpIdentityConnector{})),
	newTarget("dbmulticloud", "OracleDbGcpKeyRing", reflect.TypeOf(dbmulticloud.OracleDbGcpKeyRing{})),
	newTarget("dbmulticloud", "MultiCloudResourceDiscoverySummary", reflect.TypeOf(dbmulticloud.MultiCloudResourceDiscoverySummary{})),
	newTarget("dbmulticloud", "OracleDbAwsIdentityConnectorSummary", reflect.TypeOf(dbmulticloud.OracleDbAwsIdentityConnectorSummary{})),
	newTarget("dbmulticloud", "OracleDbAwsKeySummary", reflect.TypeOf(dbmulticloud.OracleDbAwsKeySummary{})),
	newTarget("dbmulticloud", "OracleDbAzureBlobContainerSummary", reflect.TypeOf(dbmulticloud.OracleDbAzureBlobContainerSummary{})),
	newTarget("dbmulticloud", "OracleDbAzureBlobMountSummary", reflect.TypeOf(dbmulticloud.OracleDbAzureBlobMountSummary{})),
	newTarget("dbmulticloud", "OracleDbAzureConnectorSummary", reflect.TypeOf(dbmulticloud.OracleDbAzureConnectorSummary{})),
	newTarget("dbmulticloud", "OracleDbAzureVaultAssociationSummary", reflect.TypeOf(dbmulticloud.OracleDbAzureVaultAssociationSummary{})),
	newTarget("dbmulticloud", "OracleDbAzureVaultSummary", reflect.TypeOf(dbmulticloud.OracleDbAzureVaultSummary{})),
	newTarget("dbmulticloud", "OracleDbGcpIdentityConnectorSummary", reflect.TypeOf(dbmulticloud.OracleDbGcpIdentityConnectorSummary{})),
	newTarget("dbmulticloud", "OracleDbGcpKeyRingSummary", reflect.TypeOf(dbmulticloud.OracleDbGcpKeyRingSummary{})),

	// Delegateaccesscontrol CRD support
	newTarget("delegateaccesscontrol", "CreateDelegationControlDetails", reflect.TypeOf(delegateaccesscontrol.CreateDelegationControlDetails{})),
	newTarget("delegateaccesscontrol", "UpdateDelegationControlDetails", reflect.TypeOf(delegateaccesscontrol.UpdateDelegationControlDetails{})),
	newTarget("delegateaccesscontrol", "DelegationControl", reflect.TypeOf(delegateaccesscontrol.DelegationControl{})),
	newTarget("delegateaccesscontrol", "DelegationControlSummary", reflect.TypeOf(delegateaccesscontrol.DelegationControlSummary{})),

	// Demandsignal CRD support
	newTarget("demandsignal", "CreateOccDemandSignalDetails", reflect.TypeOf(demandsignal.CreateOccDemandSignalDetails{})),
	newTarget("demandsignal", "UpdateOccDemandSignalDetails", reflect.TypeOf(demandsignal.UpdateOccDemandSignalDetails{})),
	newTarget("demandsignal", "OccDemandSignal", reflect.TypeOf(demandsignal.OccDemandSignal{})),
	newTarget("demandsignal", "OccDemandSignalCollection", reflect.TypeOf(demandsignal.OccDemandSignalCollection{})),
	newTarget("demandsignal", "OccDemandSignalSummary", reflect.TypeOf(demandsignal.OccDemandSignalSummary{})),

	// Desktops CRD support
	newTarget("desktops", "CreateDesktopPoolDetails", reflect.TypeOf(desktops.CreateDesktopPoolDetails{})),
	newTarget("desktops", "UpdateDesktopPoolDetails", reflect.TypeOf(desktops.UpdateDesktopPoolDetails{})),
	newTarget("desktops", "DesktopPool", reflect.TypeOf(desktops.DesktopPool{})),
	newTarget("desktops", "DesktopPoolCollection", reflect.TypeOf(desktops.DesktopPoolCollection{})),
	newTarget("desktops", "DesktopPoolSummary", reflect.TypeOf(desktops.DesktopPoolSummary{})),

	// Devops CRD support
	newTarget("devops", "CreateBuildPipelineDetails", reflect.TypeOf(devops.CreateBuildPipelineDetails{})),
	newTarget("devops", "CreateDeployArtifactDetails", reflect.TypeOf(devops.CreateDeployArtifactDetails{})),
	newTarget("devops", "CreateDeployPipelineDetails", reflect.TypeOf(devops.CreateDeployPipelineDetails{})),
	newTarget("devops", "CreateProjectDetails", reflect.TypeOf(devops.CreateProjectDetails{})),
	newTarget("devops", "CreateRepositoryDetails", reflect.TypeOf(devops.CreateRepositoryDetails{})),
	newTarget("devops", "UpdateBuildPipelineDetails", reflect.TypeOf(devops.UpdateBuildPipelineDetails{})),
	newTarget("devops", "UpdateDeployArtifactDetails", reflect.TypeOf(devops.UpdateDeployArtifactDetails{})),
	newTarget("devops", "UpdateDeployPipelineDetails", reflect.TypeOf(devops.UpdateDeployPipelineDetails{})),
	newTarget("devops", "UpdateProjectDetails", reflect.TypeOf(devops.UpdateProjectDetails{})),
	newTarget("devops", "UpdateRepositoryDetails", reflect.TypeOf(devops.UpdateRepositoryDetails{})),
	newTarget("devops", "BuildPipeline", reflect.TypeOf(devops.BuildPipeline{})),
	newTarget("devops", "BuildPipelineCollection", reflect.TypeOf(devops.BuildPipelineCollection{})),
	newTarget("devops", "DeployArtifact", reflect.TypeOf(devops.DeployArtifact{})),
	newTarget("devops", "DeployArtifactCollection", reflect.TypeOf(devops.DeployArtifactCollection{})),
	newTarget("devops", "DeployPipeline", reflect.TypeOf(devops.DeployPipeline{})),
	newTarget("devops", "DeployPipelineCollection", reflect.TypeOf(devops.DeployPipelineCollection{})),
	newTarget("devops", "Project", reflect.TypeOf(devops.Project{})),
	newTarget("devops", "ProjectCollection", reflect.TypeOf(devops.ProjectCollection{})),
	newTarget("devops", "Repository", reflect.TypeOf(devops.Repository{})),
	newTarget("devops", "RepositoryCollection", reflect.TypeOf(devops.RepositoryCollection{})),
	newTarget("devops", "TriggerCollection", reflect.TypeOf(devops.TriggerCollection{})),
	newTarget("devops", "BuildPipelineSummary", reflect.TypeOf(devops.BuildPipelineSummary{})),
	newTarget("devops", "DeployArtifactSummary", reflect.TypeOf(devops.DeployArtifactSummary{})),
	newTarget("devops", "DeployPipelineSummary", reflect.TypeOf(devops.DeployPipelineSummary{})),
	newTarget("devops", "ProjectSummary", reflect.TypeOf(devops.ProjectSummary{})),
	newTarget("devops", "RepositorySummary", reflect.TypeOf(devops.RepositorySummary{})),

	// Dif CRD support
	newTarget("dif", "CreateStackDetails", reflect.TypeOf(dif.CreateStackDetails{})),
	newTarget("dif", "UpdateStackDetails", reflect.TypeOf(dif.UpdateStackDetails{})),
	newTarget("dif", "Stack", reflect.TypeOf(dif.Stack{})),
	newTarget("dif", "StackCollection", reflect.TypeOf(dif.StackCollection{})),
	newTarget("dif", "StackSummary", reflect.TypeOf(dif.StackSummary{})),

	// Disasterrecovery CRD support
	newTarget("disasterrecovery", "CreateDrProtectionGroupDetails", reflect.TypeOf(disasterrecovery.CreateDrProtectionGroupDetails{})),
	newTarget("disasterrecovery", "UpdateDrProtectionGroupDetails", reflect.TypeOf(disasterrecovery.UpdateDrProtectionGroupDetails{})),
	newTarget("disasterrecovery", "DrProtectionGroup", reflect.TypeOf(disasterrecovery.DrProtectionGroup{})),
	newTarget("disasterrecovery", "DrProtectionGroupCollection", reflect.TypeOf(disasterrecovery.DrProtectionGroupCollection{})),
	newTarget("disasterrecovery", "DrProtectionGroupSummary", reflect.TypeOf(disasterrecovery.DrProtectionGroupSummary{})),

	// Distributeddatabase CRD support
	newTarget("distributeddatabase", "CreateDistributedDatabasePrivateEndpointDetails", reflect.TypeOf(distributeddatabase.CreateDistributedDatabasePrivateEndpointDetails{})),
	newTarget("distributeddatabase", "UpdateDistributedDatabasePrivateEndpointDetails", reflect.TypeOf(distributeddatabase.UpdateDistributedDatabasePrivateEndpointDetails{})),
	newTarget("distributeddatabase", "DistributedDatabasePrivateEndpoint", reflect.TypeOf(distributeddatabase.DistributedDatabasePrivateEndpoint{})),
	newTarget("distributeddatabase", "DistributedDatabasePrivateEndpointCollection", reflect.TypeOf(distributeddatabase.DistributedDatabasePrivateEndpointCollection{})),
	newTarget("distributeddatabase", "DistributedDatabasePrivateEndpointSummary", reflect.TypeOf(distributeddatabase.DistributedDatabasePrivateEndpointSummary{})),

	// Emwarehouse CRD support
	newTarget("emwarehouse", "CreateEmWarehouseDetails", reflect.TypeOf(emwarehouse.CreateEmWarehouseDetails{})),
	newTarget("emwarehouse", "UpdateEmWarehouseDetails", reflect.TypeOf(emwarehouse.UpdateEmWarehouseDetails{})),
	newTarget("emwarehouse", "EmWarehouse", reflect.TypeOf(emwarehouse.EmWarehouse{})),
	newTarget("emwarehouse", "EmWarehouseCollection", reflect.TypeOf(emwarehouse.EmWarehouseCollection{})),
	newTarget("emwarehouse", "EmWarehouseSummary", reflect.TypeOf(emwarehouse.EmWarehouseSummary{})),

	// Filestorage CRD support
	newTarget("filestorage", "CreateExportDetails", reflect.TypeOf(filestorage.CreateExportDetails{})),
	newTarget("filestorage", "CreateFileSystemDetails", reflect.TypeOf(filestorage.CreateFileSystemDetails{})),
	newTarget("filestorage", "CreateFilesystemSnapshotPolicyDetails", reflect.TypeOf(filestorage.CreateFilesystemSnapshotPolicyDetails{})),
	newTarget("filestorage", "CreateMountTargetDetails", reflect.TypeOf(filestorage.CreateMountTargetDetails{})),
	newTarget("filestorage", "CreateQuotaRuleDetails", reflect.TypeOf(filestorage.CreateQuotaRuleDetails{})),
	newTarget("filestorage", "CreateReplicationDetails", reflect.TypeOf(filestorage.CreateReplicationDetails{})),
	newTarget("filestorage", "CreateSnapshotDetails", reflect.TypeOf(filestorage.CreateSnapshotDetails{})),
	newTarget("filestorage", "UpdateExportDetails", reflect.TypeOf(filestorage.UpdateExportDetails{})),
	newTarget("filestorage", "UpdateFileSystemDetails", reflect.TypeOf(filestorage.UpdateFileSystemDetails{})),
	newTarget("filestorage", "UpdateFilesystemSnapshotPolicyDetails", reflect.TypeOf(filestorage.UpdateFilesystemSnapshotPolicyDetails{})),
	newTarget("filestorage", "UpdateMountTargetDetails", reflect.TypeOf(filestorage.UpdateMountTargetDetails{})),
	newTarget("filestorage", "UpdateOutboundConnectorDetails", reflect.TypeOf(filestorage.UpdateOutboundConnectorDetails{})),
	newTarget("filestorage", "UpdateQuotaRuleDetails", reflect.TypeOf(filestorage.UpdateQuotaRuleDetails{})),
	newTarget("filestorage", "UpdateReplicationDetails", reflect.TypeOf(filestorage.UpdateReplicationDetails{})),
	newTarget("filestorage", "UpdateSnapshotDetails", reflect.TypeOf(filestorage.UpdateSnapshotDetails{})),
	newTarget("filestorage", "Export", reflect.TypeOf(filestorage.Export{})),
	newTarget("filestorage", "FileSystem", reflect.TypeOf(filestorage.FileSystem{})),
	newTarget("filestorage", "FilesystemSnapshotPolicy", reflect.TypeOf(filestorage.FilesystemSnapshotPolicy{})),
	newTarget("filestorage", "MountTarget", reflect.TypeOf(filestorage.MountTarget{})),
	newTarget("filestorage", "QuotaRule", reflect.TypeOf(filestorage.QuotaRule{})),
	newTarget("filestorage", "Replication", reflect.TypeOf(filestorage.Replication{})),
	newTarget("filestorage", "Snapshot", reflect.TypeOf(filestorage.Snapshot{})),
	newTarget("filestorage", "ExportSummary", reflect.TypeOf(filestorage.ExportSummary{})),
	newTarget("filestorage", "FileSystemSummary", reflect.TypeOf(filestorage.FileSystemSummary{})),
	newTarget("filestorage", "FilesystemSnapshotPolicySummary", reflect.TypeOf(filestorage.FilesystemSnapshotPolicySummary{})),
	newTarget("filestorage", "MountTargetSummary", reflect.TypeOf(filestorage.MountTargetSummary{})),
	newTarget("filestorage", "QuotaRuleSummary", reflect.TypeOf(filestorage.QuotaRuleSummary{})),
	newTarget("filestorage", "ReplicationSummary", reflect.TypeOf(filestorage.ReplicationSummary{})),
	newTarget("filestorage", "SnapshotSummary", reflect.TypeOf(filestorage.SnapshotSummary{})),

	// Fleetappsmanagement CRD support
	newTarget("fleetappsmanagement", "CreateCatalogItemDetails", reflect.TypeOf(fleetappsmanagement.CreateCatalogItemDetails{})),
	newTarget("fleetappsmanagement", "CreateCompliancePolicyRuleDetails", reflect.TypeOf(fleetappsmanagement.CreateCompliancePolicyRuleDetails{})),
	newTarget("fleetappsmanagement", "CreateFleetCredentialDetails", reflect.TypeOf(fleetappsmanagement.CreateFleetCredentialDetails{})),
	newTarget("fleetappsmanagement", "CreateFleetDetails", reflect.TypeOf(fleetappsmanagement.CreateFleetDetails{})),
	newTarget("fleetappsmanagement", "CreateFleetPropertyDetails", reflect.TypeOf(fleetappsmanagement.CreateFleetPropertyDetails{})),
	newTarget("fleetappsmanagement", "CreateFleetResourceDetails", reflect.TypeOf(fleetappsmanagement.CreateFleetResourceDetails{})),
	newTarget("fleetappsmanagement", "CreateMaintenanceWindowDetails", reflect.TypeOf(fleetappsmanagement.CreateMaintenanceWindowDetails{})),
	newTarget("fleetappsmanagement", "CreateOnboardingDetails", reflect.TypeOf(fleetappsmanagement.CreateOnboardingDetails{})),
	newTarget("fleetappsmanagement", "CreatePatchDetails", reflect.TypeOf(fleetappsmanagement.CreatePatchDetails{})),
	newTarget("fleetappsmanagement", "CreatePlatformConfigurationDetails", reflect.TypeOf(fleetappsmanagement.CreatePlatformConfigurationDetails{})),
	newTarget("fleetappsmanagement", "CreatePropertyDetails", reflect.TypeOf(fleetappsmanagement.CreatePropertyDetails{})),
	newTarget("fleetappsmanagement", "CreateProvisionDetails", reflect.TypeOf(fleetappsmanagement.CreateProvisionDetails{})),
	newTarget("fleetappsmanagement", "CreateRunbookDetails", reflect.TypeOf(fleetappsmanagement.CreateRunbookDetails{})),
	newTarget("fleetappsmanagement", "CreateRunbookVersionDetails", reflect.TypeOf(fleetappsmanagement.CreateRunbookVersionDetails{})),
	newTarget("fleetappsmanagement", "CreateSchedulerDefinitionDetails", reflect.TypeOf(fleetappsmanagement.CreateSchedulerDefinitionDetails{})),
	newTarget("fleetappsmanagement", "CreateTaskRecordDetails", reflect.TypeOf(fleetappsmanagement.CreateTaskRecordDetails{})),
	newTarget("fleetappsmanagement", "UpdateCatalogItemDetails", reflect.TypeOf(fleetappsmanagement.UpdateCatalogItemDetails{})),
	newTarget("fleetappsmanagement", "UpdateCompliancePolicyRuleDetails", reflect.TypeOf(fleetappsmanagement.UpdateCompliancePolicyRuleDetails{})),
	newTarget("fleetappsmanagement", "UpdateFleetCredentialDetails", reflect.TypeOf(fleetappsmanagement.UpdateFleetCredentialDetails{})),
	newTarget("fleetappsmanagement", "UpdateFleetDetails", reflect.TypeOf(fleetappsmanagement.UpdateFleetDetails{})),
	newTarget("fleetappsmanagement", "UpdateFleetPropertyDetails", reflect.TypeOf(fleetappsmanagement.UpdateFleetPropertyDetails{})),
	newTarget("fleetappsmanagement", "UpdateFleetResourceDetails", reflect.TypeOf(fleetappsmanagement.UpdateFleetResourceDetails{})),
	newTarget("fleetappsmanagement", "UpdateMaintenanceWindowDetails", reflect.TypeOf(fleetappsmanagement.UpdateMaintenanceWindowDetails{})),
	newTarget("fleetappsmanagement", "UpdateOnboardingDetails", reflect.TypeOf(fleetappsmanagement.UpdateOnboardingDetails{})),
	newTarget("fleetappsmanagement", "UpdatePatchDetails", reflect.TypeOf(fleetappsmanagement.UpdatePatchDetails{})),
	newTarget("fleetappsmanagement", "UpdatePlatformConfigurationDetails", reflect.TypeOf(fleetappsmanagement.UpdatePlatformConfigurationDetails{})),
	newTarget("fleetappsmanagement", "UpdatePropertyDetails", reflect.TypeOf(fleetappsmanagement.UpdatePropertyDetails{})),
	newTarget("fleetappsmanagement", "UpdateProvisionDetails", reflect.TypeOf(fleetappsmanagement.UpdateProvisionDetails{})),
	newTarget("fleetappsmanagement", "UpdateRunbookDetails", reflect.TypeOf(fleetappsmanagement.UpdateRunbookDetails{})),
	newTarget("fleetappsmanagement", "UpdateRunbookVersionDetails", reflect.TypeOf(fleetappsmanagement.UpdateRunbookVersionDetails{})),
	newTarget("fleetappsmanagement", "UpdateSchedulerDefinitionDetails", reflect.TypeOf(fleetappsmanagement.UpdateSchedulerDefinitionDetails{})),
	newTarget("fleetappsmanagement", "UpdateTaskRecordDetails", reflect.TypeOf(fleetappsmanagement.UpdateTaskRecordDetails{})),
	newTarget("fleetappsmanagement", "CatalogItem", reflect.TypeOf(fleetappsmanagement.CatalogItem{})),
	newTarget("fleetappsmanagement", "CatalogItemCollection", reflect.TypeOf(fleetappsmanagement.CatalogItemCollection{})),
	newTarget("fleetappsmanagement", "CompliancePolicyRule", reflect.TypeOf(fleetappsmanagement.CompliancePolicyRule{})),
	newTarget("fleetappsmanagement", "CompliancePolicyRuleCollection", reflect.TypeOf(fleetappsmanagement.CompliancePolicyRuleCollection{})),
	newTarget("fleetappsmanagement", "Fleet", reflect.TypeOf(fleetappsmanagement.Fleet{})),
	newTarget("fleetappsmanagement", "FleetCollection", reflect.TypeOf(fleetappsmanagement.FleetCollection{})),
	newTarget("fleetappsmanagement", "FleetCredential", reflect.TypeOf(fleetappsmanagement.FleetCredential{})),
	newTarget("fleetappsmanagement", "FleetCredentialCollection", reflect.TypeOf(fleetappsmanagement.FleetCredentialCollection{})),
	newTarget("fleetappsmanagement", "FleetProperty", reflect.TypeOf(fleetappsmanagement.FleetProperty{})),
	newTarget("fleetappsmanagement", "FleetPropertyCollection", reflect.TypeOf(fleetappsmanagement.FleetPropertyCollection{})),
	newTarget("fleetappsmanagement", "FleetResource", reflect.TypeOf(fleetappsmanagement.FleetResource{})),
	newTarget("fleetappsmanagement", "FleetResourceCollection", reflect.TypeOf(fleetappsmanagement.FleetResourceCollection{})),
	newTarget("fleetappsmanagement", "MaintenanceWindow", reflect.TypeOf(fleetappsmanagement.MaintenanceWindow{})),
	newTarget("fleetappsmanagement", "MaintenanceWindowCollection", reflect.TypeOf(fleetappsmanagement.MaintenanceWindowCollection{})),
	newTarget("fleetappsmanagement", "Onboarding", reflect.TypeOf(fleetappsmanagement.Onboarding{})),
	newTarget("fleetappsmanagement", "OnboardingCollection", reflect.TypeOf(fleetappsmanagement.OnboardingCollection{})),
	newTarget("fleetappsmanagement", "Patch", reflect.TypeOf(fleetappsmanagement.Patch{})),
	newTarget("fleetappsmanagement", "PatchCollection", reflect.TypeOf(fleetappsmanagement.PatchCollection{})),
	newTarget("fleetappsmanagement", "PlatformConfiguration", reflect.TypeOf(fleetappsmanagement.PlatformConfiguration{})),
	newTarget("fleetappsmanagement", "PlatformConfigurationCollection", reflect.TypeOf(fleetappsmanagement.PlatformConfigurationCollection{})),
	newTarget("fleetappsmanagement", "Properties", reflect.TypeOf(fleetappsmanagement.Properties{})),
	newTarget("fleetappsmanagement", "Property", reflect.TypeOf(fleetappsmanagement.Property{})),
	newTarget("fleetappsmanagement", "PropertyCollection", reflect.TypeOf(fleetappsmanagement.PropertyCollection{})),
	newTarget("fleetappsmanagement", "Provision", reflect.TypeOf(fleetappsmanagement.Provision{})),
	newTarget("fleetappsmanagement", "ProvisionCollection", reflect.TypeOf(fleetappsmanagement.ProvisionCollection{})),
	newTarget("fleetappsmanagement", "Runbook", reflect.TypeOf(fleetappsmanagement.Runbook{})),
	newTarget("fleetappsmanagement", "RunbookCollection", reflect.TypeOf(fleetappsmanagement.RunbookCollection{})),
	newTarget("fleetappsmanagement", "RunbookVersion", reflect.TypeOf(fleetappsmanagement.RunbookVersion{})),
	newTarget("fleetappsmanagement", "RunbookVersionCollection", reflect.TypeOf(fleetappsmanagement.RunbookVersionCollection{})),
	newTarget("fleetappsmanagement", "SchedulerDefinition", reflect.TypeOf(fleetappsmanagement.SchedulerDefinition{})),
	newTarget("fleetappsmanagement", "SchedulerDefinitionCollection", reflect.TypeOf(fleetappsmanagement.SchedulerDefinitionCollection{})),
	newTarget("fleetappsmanagement", "TaskRecord", reflect.TypeOf(fleetappsmanagement.TaskRecord{})),
	newTarget("fleetappsmanagement", "TaskRecordCollection", reflect.TypeOf(fleetappsmanagement.TaskRecordCollection{})),
	newTarget("fleetappsmanagement", "RunbookVersionSummary", reflect.TypeOf(fleetappsmanagement.RunbookVersionSummary{})),
	newTarget("fleetappsmanagement", "CatalogItemSummary", reflect.TypeOf(fleetappsmanagement.CatalogItemSummary{})),
	newTarget("fleetappsmanagement", "CompliancePolicyRuleSummary", reflect.TypeOf(fleetappsmanagement.CompliancePolicyRuleSummary{})),
	newTarget("fleetappsmanagement", "FleetCredentialSummary", reflect.TypeOf(fleetappsmanagement.FleetCredentialSummary{})),
	newTarget("fleetappsmanagement", "FleetPropertySummary", reflect.TypeOf(fleetappsmanagement.FleetPropertySummary{})),
	newTarget("fleetappsmanagement", "FleetResourceSummary", reflect.TypeOf(fleetappsmanagement.FleetResourceSummary{})),
	newTarget("fleetappsmanagement", "FleetSummary", reflect.TypeOf(fleetappsmanagement.FleetSummary{})),
	newTarget("fleetappsmanagement", "MaintenanceWindowSummary", reflect.TypeOf(fleetappsmanagement.MaintenanceWindowSummary{})),
	newTarget("fleetappsmanagement", "OnboardingSummary", reflect.TypeOf(fleetappsmanagement.OnboardingSummary{})),
	newTarget("fleetappsmanagement", "PatchSummary", reflect.TypeOf(fleetappsmanagement.PatchSummary{})),
	newTarget("fleetappsmanagement", "PlatformConfigurationSummary", reflect.TypeOf(fleetappsmanagement.PlatformConfigurationSummary{})),
	newTarget("fleetappsmanagement", "PropertySummary", reflect.TypeOf(fleetappsmanagement.PropertySummary{})),
	newTarget("fleetappsmanagement", "ProvisionSummary", reflect.TypeOf(fleetappsmanagement.ProvisionSummary{})),
	newTarget("fleetappsmanagement", "RunbookSummary", reflect.TypeOf(fleetappsmanagement.RunbookSummary{})),
	newTarget("fleetappsmanagement", "SchedulerDefinitionSummary", reflect.TypeOf(fleetappsmanagement.SchedulerDefinitionSummary{})),
	newTarget("fleetappsmanagement", "TaskRecordSummary", reflect.TypeOf(fleetappsmanagement.TaskRecordSummary{})),

	// Fleetsoftwareupdate CRD support
	newTarget("fleetsoftwareupdate", "CreateFsuDiscoveryDetails", reflect.TypeOf(fleetsoftwareupdate.CreateFsuDiscoveryDetails{})),
	newTarget("fleetsoftwareupdate", "UpdateFsuCollectionDetails", reflect.TypeOf(fleetsoftwareupdate.UpdateFsuCollectionDetails{})),
	newTarget("fleetsoftwareupdate", "UpdateFsuDiscoveryDetails", reflect.TypeOf(fleetsoftwareupdate.UpdateFsuDiscoveryDetails{})),
	newTarget("fleetsoftwareupdate", "UpdateFsuReadinessCheckDetails", reflect.TypeOf(fleetsoftwareupdate.UpdateFsuReadinessCheckDetails{})),
	newTarget("fleetsoftwareupdate", "FsuDiscovery", reflect.TypeOf(fleetsoftwareupdate.FsuDiscovery{})),
	newTarget("fleetsoftwareupdate", "FsuReadinessCheckCollection", reflect.TypeOf(fleetsoftwareupdate.FsuReadinessCheckCollection{})),
	newTarget("fleetsoftwareupdate", "FsuCycleSummary", reflect.TypeOf(fleetsoftwareupdate.FsuCycleSummary{})),
	newTarget("fleetsoftwareupdate", "FsuDiscoverySummary", reflect.TypeOf(fleetsoftwareupdate.FsuDiscoverySummary{})),
	newTarget("fleetsoftwareupdate", "FsuReadinessCheckSummary", reflect.TypeOf(fleetsoftwareupdate.FsuReadinessCheckSummary{})),

	// Fusionapps CRD support
	newTarget("fusionapps", "CreateFusionEnvironmentDetails", reflect.TypeOf(fusionapps.CreateFusionEnvironmentDetails{})),
	newTarget("fusionapps", "CreateFusionEnvironmentFamilyDetails", reflect.TypeOf(fusionapps.CreateFusionEnvironmentFamilyDetails{})),
	newTarget("fusionapps", "CreateRefreshActivityDetails", reflect.TypeOf(fusionapps.CreateRefreshActivityDetails{})),
	newTarget("fusionapps", "CreateServiceAttachmentDetails", reflect.TypeOf(fusionapps.CreateServiceAttachmentDetails{})),
	newTarget("fusionapps", "UpdateFusionEnvironmentDetails", reflect.TypeOf(fusionapps.UpdateFusionEnvironmentDetails{})),
	newTarget("fusionapps", "UpdateFusionEnvironmentFamilyDetails", reflect.TypeOf(fusionapps.UpdateFusionEnvironmentFamilyDetails{})),
	newTarget("fusionapps", "UpdateRefreshActivityDetails", reflect.TypeOf(fusionapps.UpdateRefreshActivityDetails{})),
	newTarget("fusionapps", "FusionEnvironment", reflect.TypeOf(fusionapps.FusionEnvironment{})),
	newTarget("fusionapps", "FusionEnvironmentCollection", reflect.TypeOf(fusionapps.FusionEnvironmentCollection{})),
	newTarget("fusionapps", "FusionEnvironmentFamily", reflect.TypeOf(fusionapps.FusionEnvironmentFamily{})),
	newTarget("fusionapps", "FusionEnvironmentFamilyCollection", reflect.TypeOf(fusionapps.FusionEnvironmentFamilyCollection{})),
	newTarget("fusionapps", "RefreshActivity", reflect.TypeOf(fusionapps.RefreshActivity{})),
	newTarget("fusionapps", "RefreshActivityCollection", reflect.TypeOf(fusionapps.RefreshActivityCollection{})),
	newTarget("fusionapps", "ServiceAttachment", reflect.TypeOf(fusionapps.ServiceAttachment{})),
	newTarget("fusionapps", "ServiceAttachmentCollection", reflect.TypeOf(fusionapps.ServiceAttachmentCollection{})),
	newTarget("fusionapps", "FusionEnvironmentFamilySummary", reflect.TypeOf(fusionapps.FusionEnvironmentFamilySummary{})),
	newTarget("fusionapps", "FusionEnvironmentSummary", reflect.TypeOf(fusionapps.FusionEnvironmentSummary{})),
	newTarget("fusionapps", "RefreshActivitySummary", reflect.TypeOf(fusionapps.RefreshActivitySummary{})),
	newTarget("fusionapps", "ServiceAttachmentSummary", reflect.TypeOf(fusionapps.ServiceAttachmentSummary{})),

	// Gdp CRD support
	newTarget("gdp", "CreateGdpPipelineDetails", reflect.TypeOf(gdp.CreateGdpPipelineDetails{})),
	newTarget("gdp", "UpdateGdpPipelineDetails", reflect.TypeOf(gdp.UpdateGdpPipelineDetails{})),
	newTarget("gdp", "GdpPipeline", reflect.TypeOf(gdp.GdpPipeline{})),
	newTarget("gdp", "GdpPipelineCollection", reflect.TypeOf(gdp.GdpPipelineCollection{})),
	newTarget("gdp", "GdpPipelineSummary", reflect.TypeOf(gdp.GdpPipelineSummary{})),

	// Generativeaiagent CRD support
	newTarget("generativeaiagent", "CreateAgentDetails", reflect.TypeOf(generativeaiagent.CreateAgentDetails{})),
	newTarget("generativeaiagent", "CreateKnowledgeBaseDetails", reflect.TypeOf(generativeaiagent.CreateKnowledgeBaseDetails{})),
	newTarget("generativeaiagent", "UpdateAgentDetails", reflect.TypeOf(generativeaiagent.UpdateAgentDetails{})),
	newTarget("generativeaiagent", "UpdateKnowledgeBaseDetails", reflect.TypeOf(generativeaiagent.UpdateKnowledgeBaseDetails{})),
	newTarget("generativeaiagent", "Agent", reflect.TypeOf(generativeaiagent.Agent{})),
	newTarget("generativeaiagent", "AgentCollection", reflect.TypeOf(generativeaiagent.AgentCollection{})),
	newTarget("generativeaiagent", "KnowledgeBase", reflect.TypeOf(generativeaiagent.KnowledgeBase{})),
	newTarget("generativeaiagent", "KnowledgeBaseCollection", reflect.TypeOf(generativeaiagent.KnowledgeBaseCollection{})),
	newTarget("generativeaiagent", "AgentSummary", reflect.TypeOf(generativeaiagent.AgentSummary{})),
	newTarget("generativeaiagent", "KnowledgeBaseSummary", reflect.TypeOf(generativeaiagent.KnowledgeBaseSummary{})),

	// Generativeaiagentruntime CRD support
	newTarget("generativeaiagentruntime", "CreateSessionDetails", reflect.TypeOf(generativeaiagentruntime.CreateSessionDetails{})),
	newTarget("generativeaiagentruntime", "UpdateSessionDetails", reflect.TypeOf(generativeaiagentruntime.UpdateSessionDetails{})),
	newTarget("generativeaiagentruntime", "Session", reflect.TypeOf(generativeaiagentruntime.Session{})),

	// Generativeaidata CRD support
	newTarget("generativeaidata", "EnrichmentJob", reflect.TypeOf(generativeaidata.EnrichmentJob{})),
	newTarget("generativeaidata", "EnrichmentJobCollection", reflect.TypeOf(generativeaidata.EnrichmentJobCollection{})),
	newTarget("generativeaidata", "EnrichmentJobSummary", reflect.TypeOf(generativeaidata.EnrichmentJobSummary{})),

	// Goldengate CRD support
	newTarget("goldengate", "CreateCertificateDetails", reflect.TypeOf(goldengate.CreateCertificateDetails{})),
	newTarget("goldengate", "CreateConnectionAssignmentDetails", reflect.TypeOf(goldengate.CreateConnectionAssignmentDetails{})),
	newTarget("goldengate", "CreateDatabaseRegistrationDetails", reflect.TypeOf(goldengate.CreateDatabaseRegistrationDetails{})),
	newTarget("goldengate", "CreateDeploymentBackupDetails", reflect.TypeOf(goldengate.CreateDeploymentBackupDetails{})),
	newTarget("goldengate", "CreateDeploymentDetails", reflect.TypeOf(goldengate.CreateDeploymentDetails{})),
	newTarget("goldengate", "UpdateDatabaseRegistrationDetails", reflect.TypeOf(goldengate.UpdateDatabaseRegistrationDetails{})),
	newTarget("goldengate", "UpdateDeploymentBackupDetails", reflect.TypeOf(goldengate.UpdateDeploymentBackupDetails{})),
	newTarget("goldengate", "UpdateDeploymentDetails", reflect.TypeOf(goldengate.UpdateDeploymentDetails{})),
	newTarget("goldengate", "Certificate", reflect.TypeOf(goldengate.Certificate{})),
	newTarget("goldengate", "CertificateCollection", reflect.TypeOf(goldengate.CertificateCollection{})),
	newTarget("goldengate", "ConnectionAssignment", reflect.TypeOf(goldengate.ConnectionAssignment{})),
	newTarget("goldengate", "ConnectionAssignmentCollection", reflect.TypeOf(goldengate.ConnectionAssignmentCollection{})),
	newTarget("goldengate", "ConnectionCollection", reflect.TypeOf(goldengate.ConnectionCollection{})),
	newTarget("goldengate", "DatabaseRegistration", reflect.TypeOf(goldengate.DatabaseRegistration{})),
	newTarget("goldengate", "DatabaseRegistrationCollection", reflect.TypeOf(goldengate.DatabaseRegistrationCollection{})),
	newTarget("goldengate", "Deployment", reflect.TypeOf(goldengate.Deployment{})),
	newTarget("goldengate", "DeploymentBackup", reflect.TypeOf(goldengate.DeploymentBackup{})),
	newTarget("goldengate", "DeploymentBackupCollection", reflect.TypeOf(goldengate.DeploymentBackupCollection{})),
	newTarget("goldengate", "DeploymentCollection", reflect.TypeOf(goldengate.DeploymentCollection{})),
	newTarget("goldengate", "PipelineCollection", reflect.TypeOf(goldengate.PipelineCollection{})),
	newTarget("goldengate", "DeploymentVersionSummary", reflect.TypeOf(goldengate.DeploymentVersionSummary{})),
	newTarget("goldengate", "CertificateSummary", reflect.TypeOf(goldengate.CertificateSummary{})),
	newTarget("goldengate", "ConnectionAssignmentSummary", reflect.TypeOf(goldengate.ConnectionAssignmentSummary{})),
	newTarget("goldengate", "DatabaseRegistrationSummary", reflect.TypeOf(goldengate.DatabaseRegistrationSummary{})),
	newTarget("goldengate", "DeploymentBackupSummary", reflect.TypeOf(goldengate.DeploymentBackupSummary{})),
	newTarget("goldengate", "DeploymentSummary", reflect.TypeOf(goldengate.DeploymentSummary{})),

	// Governancerulescontrolplane CRD support
	newTarget("governancerulescontrolplane", "CreateGovernanceRuleDetails", reflect.TypeOf(governancerulescontrolplane.CreateGovernanceRuleDetails{})),
	newTarget("governancerulescontrolplane", "CreateInclusionCriterionDetails", reflect.TypeOf(governancerulescontrolplane.CreateInclusionCriterionDetails{})),
	newTarget("governancerulescontrolplane", "UpdateGovernanceRuleDetails", reflect.TypeOf(governancerulescontrolplane.UpdateGovernanceRuleDetails{})),
	newTarget("governancerulescontrolplane", "GovernanceRule", reflect.TypeOf(governancerulescontrolplane.GovernanceRule{})),
	newTarget("governancerulescontrolplane", "GovernanceRuleCollection", reflect.TypeOf(governancerulescontrolplane.GovernanceRuleCollection{})),
	newTarget("governancerulescontrolplane", "InclusionCriterion", reflect.TypeOf(governancerulescontrolplane.InclusionCriterion{})),
	newTarget("governancerulescontrolplane", "InclusionCriterionCollection", reflect.TypeOf(governancerulescontrolplane.InclusionCriterionCollection{})),
	newTarget("governancerulescontrolplane", "GovernanceRuleSummary", reflect.TypeOf(governancerulescontrolplane.GovernanceRuleSummary{})),
	newTarget("governancerulescontrolplane", "InclusionCriterionSummary", reflect.TypeOf(governancerulescontrolplane.InclusionCriterionSummary{})),

	// Healthchecks CRD support
	newTarget("healthchecks", "CreateHttpMonitorDetails", reflect.TypeOf(healthchecks.CreateHttpMonitorDetails{})),
	newTarget("healthchecks", "CreatePingMonitorDetails", reflect.TypeOf(healthchecks.CreatePingMonitorDetails{})),
	newTarget("healthchecks", "UpdateHttpMonitorDetails", reflect.TypeOf(healthchecks.UpdateHttpMonitorDetails{})),
	newTarget("healthchecks", "UpdatePingMonitorDetails", reflect.TypeOf(healthchecks.UpdatePingMonitorDetails{})),
	newTarget("healthchecks", "HttpMonitor", reflect.TypeOf(healthchecks.HttpMonitor{})),
	newTarget("healthchecks", "PingMonitor", reflect.TypeOf(healthchecks.PingMonitor{})),
	newTarget("healthchecks", "HttpMonitorSummary", reflect.TypeOf(healthchecks.HttpMonitorSummary{})),
	newTarget("healthchecks", "PingMonitorSummary", reflect.TypeOf(healthchecks.PingMonitorSummary{})),

	// Integration CRD support
	newTarget("integration", "CreateIntegrationInstanceDetails", reflect.TypeOf(integration.CreateIntegrationInstanceDetails{})),
	newTarget("integration", "UpdateIntegrationInstanceDetails", reflect.TypeOf(integration.UpdateIntegrationInstanceDetails{})),
	newTarget("integration", "IntegrationInstance", reflect.TypeOf(integration.IntegrationInstance{})),
	newTarget("integration", "IntegrationInstanceSummary", reflect.TypeOf(integration.IntegrationInstanceSummary{})),

	// Iot CRD support
	newTarget("iot", "CreateDigitalTwinAdapterDetails", reflect.TypeOf(iot.CreateDigitalTwinAdapterDetails{})),
	newTarget("iot", "CreateDigitalTwinInstanceDetails", reflect.TypeOf(iot.CreateDigitalTwinInstanceDetails{})),
	newTarget("iot", "CreateDigitalTwinModelDetails", reflect.TypeOf(iot.CreateDigitalTwinModelDetails{})),
	newTarget("iot", "CreateDigitalTwinRelationshipDetails", reflect.TypeOf(iot.CreateDigitalTwinRelationshipDetails{})),
	newTarget("iot", "CreateIotDomainDetails", reflect.TypeOf(iot.CreateIotDomainDetails{})),
	newTarget("iot", "CreateIotDomainGroupDetails", reflect.TypeOf(iot.CreateIotDomainGroupDetails{})),
	newTarget("iot", "UpdateDigitalTwinAdapterDetails", reflect.TypeOf(iot.UpdateDigitalTwinAdapterDetails{})),
	newTarget("iot", "UpdateDigitalTwinInstanceDetails", reflect.TypeOf(iot.UpdateDigitalTwinInstanceDetails{})),
	newTarget("iot", "UpdateDigitalTwinModelDetails", reflect.TypeOf(iot.UpdateDigitalTwinModelDetails{})),
	newTarget("iot", "UpdateDigitalTwinRelationshipDetails", reflect.TypeOf(iot.UpdateDigitalTwinRelationshipDetails{})),
	newTarget("iot", "UpdateIotDomainDetails", reflect.TypeOf(iot.UpdateIotDomainDetails{})),
	newTarget("iot", "UpdateIotDomainGroupDetails", reflect.TypeOf(iot.UpdateIotDomainGroupDetails{})),
	newTarget("iot", "DigitalTwinAdapter", reflect.TypeOf(iot.DigitalTwinAdapter{})),
	newTarget("iot", "DigitalTwinAdapterCollection", reflect.TypeOf(iot.DigitalTwinAdapterCollection{})),
	newTarget("iot", "DigitalTwinInstance", reflect.TypeOf(iot.DigitalTwinInstance{})),
	newTarget("iot", "DigitalTwinInstanceCollection", reflect.TypeOf(iot.DigitalTwinInstanceCollection{})),
	newTarget("iot", "DigitalTwinModel", reflect.TypeOf(iot.DigitalTwinModel{})),
	newTarget("iot", "DigitalTwinModelCollection", reflect.TypeOf(iot.DigitalTwinModelCollection{})),
	newTarget("iot", "DigitalTwinRelationship", reflect.TypeOf(iot.DigitalTwinRelationship{})),
	newTarget("iot", "DigitalTwinRelationshipCollection", reflect.TypeOf(iot.DigitalTwinRelationshipCollection{})),
	newTarget("iot", "IotDomain", reflect.TypeOf(iot.IotDomain{})),
	newTarget("iot", "IotDomainCollection", reflect.TypeOf(iot.IotDomainCollection{})),
	newTarget("iot", "IotDomainGroup", reflect.TypeOf(iot.IotDomainGroup{})),
	newTarget("iot", "IotDomainGroupCollection", reflect.TypeOf(iot.IotDomainGroupCollection{})),
	newTarget("iot", "DigitalTwinAdapterSummary", reflect.TypeOf(iot.DigitalTwinAdapterSummary{})),
	newTarget("iot", "DigitalTwinInstanceSummary", reflect.TypeOf(iot.DigitalTwinInstanceSummary{})),
	newTarget("iot", "DigitalTwinModelSummary", reflect.TypeOf(iot.DigitalTwinModelSummary{})),
	newTarget("iot", "DigitalTwinRelationshipSummary", reflect.TypeOf(iot.DigitalTwinRelationshipSummary{})),
	newTarget("iot", "IotDomainGroupSummary", reflect.TypeOf(iot.IotDomainGroupSummary{})),
	newTarget("iot", "IotDomainSummary", reflect.TypeOf(iot.IotDomainSummary{})),

	// Jms CRD support
	newTarget("jms", "CreateFleetDetails", reflect.TypeOf(jms.CreateFleetDetails{})),
	newTarget("jms", "UpdateFleetDetails", reflect.TypeOf(jms.UpdateFleetDetails{})),
	newTarget("jms", "Fleet", reflect.TypeOf(jms.Fleet{})),
	newTarget("jms", "FleetCollection", reflect.TypeOf(jms.FleetCollection{})),
	newTarget("jms", "FleetSummary", reflect.TypeOf(jms.FleetSummary{})),

	// Jmsjavadownloads CRD support
	newTarget("jmsjavadownloads", "CreateJavaDownloadTokenDetails", reflect.TypeOf(jmsjavadownloads.CreateJavaDownloadTokenDetails{})),
	newTarget("jmsjavadownloads", "UpdateJavaDownloadTokenDetails", reflect.TypeOf(jmsjavadownloads.UpdateJavaDownloadTokenDetails{})),
	newTarget("jmsjavadownloads", "JavaDownloadToken", reflect.TypeOf(jmsjavadownloads.JavaDownloadToken{})),
	newTarget("jmsjavadownloads", "JavaDownloadTokenCollection", reflect.TypeOf(jmsjavadownloads.JavaDownloadTokenCollection{})),
	newTarget("jmsjavadownloads", "JavaDownloadTokenSummary", reflect.TypeOf(jmsjavadownloads.JavaDownloadTokenSummary{})),

	// Licensemanager CRD support
	newTarget("licensemanager", "CreateLicenseRecordDetails", reflect.TypeOf(licensemanager.CreateLicenseRecordDetails{})),
	newTarget("licensemanager", "CreateProductLicenseDetails", reflect.TypeOf(licensemanager.CreateProductLicenseDetails{})),
	newTarget("licensemanager", "UpdateLicenseRecordDetails", reflect.TypeOf(licensemanager.UpdateLicenseRecordDetails{})),
	newTarget("licensemanager", "UpdateProductLicenseDetails", reflect.TypeOf(licensemanager.UpdateProductLicenseDetails{})),
	newTarget("licensemanager", "LicenseRecord", reflect.TypeOf(licensemanager.LicenseRecord{})),
	newTarget("licensemanager", "LicenseRecordCollection", reflect.TypeOf(licensemanager.LicenseRecordCollection{})),
	newTarget("licensemanager", "ProductLicense", reflect.TypeOf(licensemanager.ProductLicense{})),
	newTarget("licensemanager", "ProductLicenseCollection", reflect.TypeOf(licensemanager.ProductLicenseCollection{})),
	newTarget("licensemanager", "LicenseRecordSummary", reflect.TypeOf(licensemanager.LicenseRecordSummary{})),
	newTarget("licensemanager", "ProductLicenseSummary", reflect.TypeOf(licensemanager.ProductLicenseSummary{})),

	// Limitsincrease CRD support
	newTarget("limitsincrease", "CreateLimitsIncreaseRequestDetails", reflect.TypeOf(limitsincrease.CreateLimitsIncreaseRequestDetails{})),
	newTarget("limitsincrease", "UpdateLimitsIncreaseRequestDetails", reflect.TypeOf(limitsincrease.UpdateLimitsIncreaseRequestDetails{})),
	newTarget("limitsincrease", "LimitsIncreaseRequest", reflect.TypeOf(limitsincrease.LimitsIncreaseRequest{})),
	newTarget("limitsincrease", "LimitsIncreaseRequestCollection", reflect.TypeOf(limitsincrease.LimitsIncreaseRequestCollection{})),
	newTarget("limitsincrease", "LimitsIncreaseRequestSummary", reflect.TypeOf(limitsincrease.LimitsIncreaseRequestSummary{})),

	// Lockbox CRD support
	newTarget("lockbox", "CreateApprovalTemplateDetails", reflect.TypeOf(lockbox.CreateApprovalTemplateDetails{})),
	newTarget("lockbox", "CreateLockboxDetails", reflect.TypeOf(lockbox.CreateLockboxDetails{})),
	newTarget("lockbox", "UpdateApprovalTemplateDetails", reflect.TypeOf(lockbox.UpdateApprovalTemplateDetails{})),
	newTarget("lockbox", "UpdateLockboxDetails", reflect.TypeOf(lockbox.UpdateLockboxDetails{})),
	newTarget("lockbox", "ApprovalTemplate", reflect.TypeOf(lockbox.ApprovalTemplate{})),
	newTarget("lockbox", "ApprovalTemplateCollection", reflect.TypeOf(lockbox.ApprovalTemplateCollection{})),
	newTarget("lockbox", "Lockbox", reflect.TypeOf(lockbox.Lockbox{})),
	newTarget("lockbox", "LockboxCollection", reflect.TypeOf(lockbox.LockboxCollection{})),
	newTarget("lockbox", "ApprovalTemplateSummary", reflect.TypeOf(lockbox.ApprovalTemplateSummary{})),
	newTarget("lockbox", "LockboxSummary", reflect.TypeOf(lockbox.LockboxSummary{})),

	// Loganalytics CRD support
	newTarget("loganalytics", "CreateIngestTimeRuleDetails", reflect.TypeOf(loganalytics.CreateIngestTimeRuleDetails{})),
	newTarget("loganalytics", "CreateLogAnalyticsEmBridgeDetails", reflect.TypeOf(loganalytics.CreateLogAnalyticsEmBridgeDetails{})),
	newTarget("loganalytics", "CreateLogAnalyticsEntityDetails", reflect.TypeOf(loganalytics.CreateLogAnalyticsEntityDetails{})),
	newTarget("loganalytics", "CreateLogAnalyticsEntityTypeDetails", reflect.TypeOf(loganalytics.CreateLogAnalyticsEntityTypeDetails{})),
	newTarget("loganalytics", "CreateLogAnalyticsLogGroupDetails", reflect.TypeOf(loganalytics.CreateLogAnalyticsLogGroupDetails{})),
	newTarget("loganalytics", "CreateLogAnalyticsObjectCollectionRuleDetails", reflect.TypeOf(loganalytics.CreateLogAnalyticsObjectCollectionRuleDetails{})),
	newTarget("loganalytics", "UpdateLogAnalyticsEmBridgeDetails", reflect.TypeOf(loganalytics.UpdateLogAnalyticsEmBridgeDetails{})),
	newTarget("loganalytics", "UpdateLogAnalyticsEntityDetails", reflect.TypeOf(loganalytics.UpdateLogAnalyticsEntityDetails{})),
	newTarget("loganalytics", "UpdateLogAnalyticsEntityTypeDetails", reflect.TypeOf(loganalytics.UpdateLogAnalyticsEntityTypeDetails{})),
	newTarget("loganalytics", "UpdateLogAnalyticsLogGroupDetails", reflect.TypeOf(loganalytics.UpdateLogAnalyticsLogGroupDetails{})),
	newTarget("loganalytics", "UpdateLogAnalyticsObjectCollectionRuleDetails", reflect.TypeOf(loganalytics.UpdateLogAnalyticsObjectCollectionRuleDetails{})),
	newTarget("loganalytics", "IngestTimeRule", reflect.TypeOf(loganalytics.IngestTimeRule{})),
	newTarget("loganalytics", "LogAnalyticsEmBridge", reflect.TypeOf(loganalytics.LogAnalyticsEmBridge{})),
	newTarget("loganalytics", "LogAnalyticsEmBridgeCollection", reflect.TypeOf(loganalytics.LogAnalyticsEmBridgeCollection{})),
	newTarget("loganalytics", "LogAnalyticsEntity", reflect.TypeOf(loganalytics.LogAnalyticsEntity{})),
	newTarget("loganalytics", "LogAnalyticsEntityCollection", reflect.TypeOf(loganalytics.LogAnalyticsEntityCollection{})),
	newTarget("loganalytics", "LogAnalyticsEntityType", reflect.TypeOf(loganalytics.LogAnalyticsEntityType{})),
	newTarget("loganalytics", "LogAnalyticsEntityTypeCollection", reflect.TypeOf(loganalytics.LogAnalyticsEntityTypeCollection{})),
	newTarget("loganalytics", "LogAnalyticsLogGroup", reflect.TypeOf(loganalytics.LogAnalyticsLogGroup{})),
	newTarget("loganalytics", "LogAnalyticsObjectCollectionRule", reflect.TypeOf(loganalytics.LogAnalyticsObjectCollectionRule{})),
	newTarget("loganalytics", "LogAnalyticsObjectCollectionRuleCollection", reflect.TypeOf(loganalytics.LogAnalyticsObjectCollectionRuleCollection{})),
	newTarget("loganalytics", "ScheduledTaskCollection", reflect.TypeOf(loganalytics.ScheduledTaskCollection{})),
	newTarget("loganalytics", "IngestTimeRuleSummary", reflect.TypeOf(loganalytics.IngestTimeRuleSummary{})),
	newTarget("loganalytics", "LogAnalyticsEmBridgeSummary", reflect.TypeOf(loganalytics.LogAnalyticsEmBridgeSummary{})),
	newTarget("loganalytics", "LogAnalyticsEntitySummary", reflect.TypeOf(loganalytics.LogAnalyticsEntitySummary{})),
	newTarget("loganalytics", "LogAnalyticsEntityTypeSummary", reflect.TypeOf(loganalytics.LogAnalyticsEntityTypeSummary{})),
	newTarget("loganalytics", "LogAnalyticsLogGroupSummary", reflect.TypeOf(loganalytics.LogAnalyticsLogGroupSummary{})),
	newTarget("loganalytics", "LogAnalyticsObjectCollectionRuleSummary", reflect.TypeOf(loganalytics.LogAnalyticsObjectCollectionRuleSummary{})),
	newTarget("loganalytics", "ScheduledTaskSummary", reflect.TypeOf(loganalytics.ScheduledTaskSummary{})),

	// Lustrefilestorage CRD support
	newTarget("lustrefilestorage", "CreateLustreFileSystemDetails", reflect.TypeOf(lustrefilestorage.CreateLustreFileSystemDetails{})),
	newTarget("lustrefilestorage", "CreateObjectStorageLinkDetails", reflect.TypeOf(lustrefilestorage.CreateObjectStorageLinkDetails{})),
	newTarget("lustrefilestorage", "UpdateLustreFileSystemDetails", reflect.TypeOf(lustrefilestorage.UpdateLustreFileSystemDetails{})),
	newTarget("lustrefilestorage", "UpdateObjectStorageLinkDetails", reflect.TypeOf(lustrefilestorage.UpdateObjectStorageLinkDetails{})),
	newTarget("lustrefilestorage", "LustreFileSystem", reflect.TypeOf(lustrefilestorage.LustreFileSystem{})),
	newTarget("lustrefilestorage", "LustreFileSystemCollection", reflect.TypeOf(lustrefilestorage.LustreFileSystemCollection{})),
	newTarget("lustrefilestorage", "ObjectStorageLink", reflect.TypeOf(lustrefilestorage.ObjectStorageLink{})),
	newTarget("lustrefilestorage", "ObjectStorageLinkCollection", reflect.TypeOf(lustrefilestorage.ObjectStorageLinkCollection{})),
	newTarget("lustrefilestorage", "LustreFileSystemSummary", reflect.TypeOf(lustrefilestorage.LustreFileSystemSummary{})),
	newTarget("lustrefilestorage", "ObjectStorageLinkSummary", reflect.TypeOf(lustrefilestorage.ObjectStorageLinkSummary{})),

	// Managedkafka CRD support
	newTarget("managedkafka", "CreateKafkaClusterConfigDetails", reflect.TypeOf(managedkafka.CreateKafkaClusterConfigDetails{})),
	newTarget("managedkafka", "CreateKafkaClusterDetails", reflect.TypeOf(managedkafka.CreateKafkaClusterDetails{})),
	newTarget("managedkafka", "UpdateKafkaClusterConfigDetails", reflect.TypeOf(managedkafka.UpdateKafkaClusterConfigDetails{})),
	newTarget("managedkafka", "UpdateKafkaClusterDetails", reflect.TypeOf(managedkafka.UpdateKafkaClusterDetails{})),
	newTarget("managedkafka", "KafkaCluster", reflect.TypeOf(managedkafka.KafkaCluster{})),
	newTarget("managedkafka", "KafkaClusterCollection", reflect.TypeOf(managedkafka.KafkaClusterCollection{})),
	newTarget("managedkafka", "KafkaClusterConfig", reflect.TypeOf(managedkafka.KafkaClusterConfig{})),
	newTarget("managedkafka", "KafkaClusterConfigCollection", reflect.TypeOf(managedkafka.KafkaClusterConfigCollection{})),
	newTarget("managedkafka", "KafkaClusterConfigVersionSummary", reflect.TypeOf(managedkafka.KafkaClusterConfigVersionSummary{})),
	newTarget("managedkafka", "KafkaClusterConfigSummary", reflect.TypeOf(managedkafka.KafkaClusterConfigSummary{})),
	newTarget("managedkafka", "KafkaClusterSummary", reflect.TypeOf(managedkafka.KafkaClusterSummary{})),

	// Managementagent CRD support
	newTarget("managementagent", "CreateManagementAgentInstallKeyDetails", reflect.TypeOf(managementagent.CreateManagementAgentInstallKeyDetails{})),
	newTarget("managementagent", "CreateNamedCredentialDetails", reflect.TypeOf(managementagent.CreateNamedCredentialDetails{})),
	newTarget("managementagent", "UpdateManagementAgentInstallKeyDetails", reflect.TypeOf(managementagent.UpdateManagementAgentInstallKeyDetails{})),
	newTarget("managementagent", "UpdateNamedCredentialDetails", reflect.TypeOf(managementagent.UpdateNamedCredentialDetails{})),
	newTarget("managementagent", "ManagementAgentInstallKey", reflect.TypeOf(managementagent.ManagementAgentInstallKey{})),
	newTarget("managementagent", "NamedCredential", reflect.TypeOf(managementagent.NamedCredential{})),
	newTarget("managementagent", "NamedCredentialCollection", reflect.TypeOf(managementagent.NamedCredentialCollection{})),
	newTarget("managementagent", "ManagementAgentInstallKeySummary", reflect.TypeOf(managementagent.ManagementAgentInstallKeySummary{})),
	newTarget("managementagent", "NamedCredentialSummary", reflect.TypeOf(managementagent.NamedCredentialSummary{})),

	// Managementdashboard CRD support
	newTarget("managementdashboard", "CreateManagementDashboardDetails", reflect.TypeOf(managementdashboard.CreateManagementDashboardDetails{})),
	newTarget("managementdashboard", "CreateManagementSavedSearchDetails", reflect.TypeOf(managementdashboard.CreateManagementSavedSearchDetails{})),
	newTarget("managementdashboard", "UpdateManagementDashboardDetails", reflect.TypeOf(managementdashboard.UpdateManagementDashboardDetails{})),
	newTarget("managementdashboard", "UpdateManagementSavedSearchDetails", reflect.TypeOf(managementdashboard.UpdateManagementSavedSearchDetails{})),
	newTarget("managementdashboard", "ManagementDashboard", reflect.TypeOf(managementdashboard.ManagementDashboard{})),
	newTarget("managementdashboard", "ManagementDashboardCollection", reflect.TypeOf(managementdashboard.ManagementDashboardCollection{})),
	newTarget("managementdashboard", "ManagementSavedSearch", reflect.TypeOf(managementdashboard.ManagementSavedSearch{})),
	newTarget("managementdashboard", "ManagementSavedSearchCollection", reflect.TypeOf(managementdashboard.ManagementSavedSearchCollection{})),
	newTarget("managementdashboard", "ManagementDashboardSummary", reflect.TypeOf(managementdashboard.ManagementDashboardSummary{})),
	newTarget("managementdashboard", "ManagementSavedSearchSummary", reflect.TypeOf(managementdashboard.ManagementSavedSearchSummary{})),

	// Marketplaceprivateoffer CRD support
	newTarget("marketplaceprivateoffer", "CreateAttachmentDetails", reflect.TypeOf(marketplaceprivateoffer.CreateAttachmentDetails{})),
	newTarget("marketplaceprivateoffer", "CreateOfferDetails", reflect.TypeOf(marketplaceprivateoffer.CreateOfferDetails{})),
	newTarget("marketplaceprivateoffer", "UpdateOfferDetails", reflect.TypeOf(marketplaceprivateoffer.UpdateOfferDetails{})),
	newTarget("marketplaceprivateoffer", "Attachment", reflect.TypeOf(marketplaceprivateoffer.Attachment{})),
	newTarget("marketplaceprivateoffer", "AttachmentCollection", reflect.TypeOf(marketplaceprivateoffer.AttachmentCollection{})),
	newTarget("marketplaceprivateoffer", "Offer", reflect.TypeOf(marketplaceprivateoffer.Offer{})),
	newTarget("marketplaceprivateoffer", "OfferCollection", reflect.TypeOf(marketplaceprivateoffer.OfferCollection{})),
	newTarget("marketplaceprivateoffer", "AttachmentSummary", reflect.TypeOf(marketplaceprivateoffer.AttachmentSummary{})),
	newTarget("marketplaceprivateoffer", "OfferSummary", reflect.TypeOf(marketplaceprivateoffer.OfferSummary{})),

	// Marketplacepublisher CRD support
	newTarget("marketplacepublisher", "CreateListingDetails", reflect.TypeOf(marketplacepublisher.CreateListingDetails{})),
	newTarget("marketplacepublisher", "CreateListingRevisionNoteDetails", reflect.TypeOf(marketplacepublisher.CreateListingRevisionNoteDetails{})),
	newTarget("marketplacepublisher", "CreateListingRevisionPackageDetails", reflect.TypeOf(marketplacepublisher.CreateListingRevisionPackageDetails{})),
	newTarget("marketplacepublisher", "CreateTermDetails", reflect.TypeOf(marketplacepublisher.CreateTermDetails{})),
	newTarget("marketplacepublisher", "UpdateListingDetails", reflect.TypeOf(marketplacepublisher.UpdateListingDetails{})),
	newTarget("marketplacepublisher", "UpdateListingRevisionNoteDetails", reflect.TypeOf(marketplacepublisher.UpdateListingRevisionNoteDetails{})),
	newTarget("marketplacepublisher", "UpdateListingRevisionPackageDetails", reflect.TypeOf(marketplacepublisher.UpdateListingRevisionPackageDetails{})),
	newTarget("marketplacepublisher", "UpdateTermDetails", reflect.TypeOf(marketplacepublisher.UpdateTermDetails{})),
	newTarget("marketplacepublisher", "UpdateTermVersionDetails", reflect.TypeOf(marketplacepublisher.UpdateTermVersionDetails{})),
	newTarget("marketplacepublisher", "ArtifactCollection", reflect.TypeOf(marketplacepublisher.ArtifactCollection{})),
	newTarget("marketplacepublisher", "Listing", reflect.TypeOf(marketplacepublisher.Listing{})),
	newTarget("marketplacepublisher", "ListingCollection", reflect.TypeOf(marketplacepublisher.ListingCollection{})),
	newTarget("marketplacepublisher", "ListingRevisionAttachmentCollection", reflect.TypeOf(marketplacepublisher.ListingRevisionAttachmentCollection{})),
	newTarget("marketplacepublisher", "ListingRevisionCollection", reflect.TypeOf(marketplacepublisher.ListingRevisionCollection{})),
	newTarget("marketplacepublisher", "ListingRevisionNote", reflect.TypeOf(marketplacepublisher.ListingRevisionNote{})),
	newTarget("marketplacepublisher", "ListingRevisionNoteCollection", reflect.TypeOf(marketplacepublisher.ListingRevisionNoteCollection{})),
	newTarget("marketplacepublisher", "ListingRevisionPackageCollection", reflect.TypeOf(marketplacepublisher.ListingRevisionPackageCollection{})),
	newTarget("marketplacepublisher", "Term", reflect.TypeOf(marketplacepublisher.Term{})),
	newTarget("marketplacepublisher", "TermCollection", reflect.TypeOf(marketplacepublisher.TermCollection{})),
	newTarget("marketplacepublisher", "TermVersion", reflect.TypeOf(marketplacepublisher.TermVersion{})),
	newTarget("marketplacepublisher", "TermVersionCollection", reflect.TypeOf(marketplacepublisher.TermVersionCollection{})),
	newTarget("marketplacepublisher", "TermVersionSummary", reflect.TypeOf(marketplacepublisher.TermVersionSummary{})),
	newTarget("marketplacepublisher", "ArtifactSummary", reflect.TypeOf(marketplacepublisher.ArtifactSummary{})),
	newTarget("marketplacepublisher", "ListingRevisionAttachmentSummary", reflect.TypeOf(marketplacepublisher.ListingRevisionAttachmentSummary{})),
	newTarget("marketplacepublisher", "ListingRevisionNoteSummary", reflect.TypeOf(marketplacepublisher.ListingRevisionNoteSummary{})),
	newTarget("marketplacepublisher", "ListingRevisionPackageSummary", reflect.TypeOf(marketplacepublisher.ListingRevisionPackageSummary{})),
	newTarget("marketplacepublisher", "ListingRevisionSummary", reflect.TypeOf(marketplacepublisher.ListingRevisionSummary{})),
	newTarget("marketplacepublisher", "ListingSummary", reflect.TypeOf(marketplacepublisher.ListingSummary{})),
	newTarget("marketplacepublisher", "TermSummary", reflect.TypeOf(marketplacepublisher.TermSummary{})),

	// Mediaservices CRD support
	newTarget("mediaservices", "CreateMediaAssetDetails", reflect.TypeOf(mediaservices.CreateMediaAssetDetails{})),
	newTarget("mediaservices", "CreateMediaWorkflowDetails", reflect.TypeOf(mediaservices.CreateMediaWorkflowDetails{})),
	newTarget("mediaservices", "UpdateMediaAssetDetails", reflect.TypeOf(mediaservices.UpdateMediaAssetDetails{})),
	newTarget("mediaservices", "UpdateMediaWorkflowDetails", reflect.TypeOf(mediaservices.UpdateMediaWorkflowDetails{})),
	newTarget("mediaservices", "MediaAsset", reflect.TypeOf(mediaservices.MediaAsset{})),
	newTarget("mediaservices", "MediaAssetCollection", reflect.TypeOf(mediaservices.MediaAssetCollection{})),
	newTarget("mediaservices", "MediaWorkflow", reflect.TypeOf(mediaservices.MediaWorkflow{})),
	newTarget("mediaservices", "MediaWorkflowCollection", reflect.TypeOf(mediaservices.MediaWorkflowCollection{})),
	newTarget("mediaservices", "MediaAssetSummary", reflect.TypeOf(mediaservices.MediaAssetSummary{})),
	newTarget("mediaservices", "MediaWorkflowSummary", reflect.TypeOf(mediaservices.MediaWorkflowSummary{})),

	// Mngdmac CRD support
	newTarget("mngdmac", "CreateMacOrderDetails", reflect.TypeOf(mngdmac.CreateMacOrderDetails{})),
	newTarget("mngdmac", "UpdateMacOrderDetails", reflect.TypeOf(mngdmac.UpdateMacOrderDetails{})),
	newTarget("mngdmac", "MacDevice", reflect.TypeOf(mngdmac.MacDevice{})),
	newTarget("mngdmac", "MacDeviceCollection", reflect.TypeOf(mngdmac.MacDeviceCollection{})),
	newTarget("mngdmac", "MacOrder", reflect.TypeOf(mngdmac.MacOrder{})),
	newTarget("mngdmac", "MacOrderCollection", reflect.TypeOf(mngdmac.MacOrderCollection{})),
	newTarget("mngdmac", "MacDeviceSummary", reflect.TypeOf(mngdmac.MacDeviceSummary{})),
	newTarget("mngdmac", "MacOrderSummary", reflect.TypeOf(mngdmac.MacOrderSummary{})),

	// Networkfirewall CRD support
	newTarget("networkfirewall", "CreateAddressListDetails", reflect.TypeOf(networkfirewall.CreateAddressListDetails{})),
	newTarget("networkfirewall", "CreateApplicationGroupDetails", reflect.TypeOf(networkfirewall.CreateApplicationGroupDetails{})),
	newTarget("networkfirewall", "CreateDecryptionRuleDetails", reflect.TypeOf(networkfirewall.CreateDecryptionRuleDetails{})),
	newTarget("networkfirewall", "CreateNetworkFirewallDetails", reflect.TypeOf(networkfirewall.CreateNetworkFirewallDetails{})),
	newTarget("networkfirewall", "CreateNetworkFirewallPolicyDetails", reflect.TypeOf(networkfirewall.CreateNetworkFirewallPolicyDetails{})),
	newTarget("networkfirewall", "CreateSecurityRuleDetails", reflect.TypeOf(networkfirewall.CreateSecurityRuleDetails{})),
	newTarget("networkfirewall", "CreateServiceListDetails", reflect.TypeOf(networkfirewall.CreateServiceListDetails{})),
	newTarget("networkfirewall", "CreateUrlListDetails", reflect.TypeOf(networkfirewall.CreateUrlListDetails{})),
	newTarget("networkfirewall", "UpdateApplicationGroupDetails", reflect.TypeOf(networkfirewall.UpdateApplicationGroupDetails{})),
	newTarget("networkfirewall", "UpdateDecryptionRuleDetails", reflect.TypeOf(networkfirewall.UpdateDecryptionRuleDetails{})),
	newTarget("networkfirewall", "UpdateNetworkFirewallDetails", reflect.TypeOf(networkfirewall.UpdateNetworkFirewallDetails{})),
	newTarget("networkfirewall", "UpdateNetworkFirewallPolicyDetails", reflect.TypeOf(networkfirewall.UpdateNetworkFirewallPolicyDetails{})),
	newTarget("networkfirewall", "UpdateSecurityRuleDetails", reflect.TypeOf(networkfirewall.UpdateSecurityRuleDetails{})),
	newTarget("networkfirewall", "UpdateServiceListDetails", reflect.TypeOf(networkfirewall.UpdateServiceListDetails{})),
	newTarget("networkfirewall", "UpdateUrlListDetails", reflect.TypeOf(networkfirewall.UpdateUrlListDetails{})),
	newTarget("networkfirewall", "AddressList", reflect.TypeOf(networkfirewall.AddressList{})),
	newTarget("networkfirewall", "ApplicationGroup", reflect.TypeOf(networkfirewall.ApplicationGroup{})),
	newTarget("networkfirewall", "DecryptionRule", reflect.TypeOf(networkfirewall.DecryptionRule{})),
	newTarget("networkfirewall", "NatRuleCollection", reflect.TypeOf(networkfirewall.NatRuleCollection{})),
	newTarget("networkfirewall", "NetworkFirewall", reflect.TypeOf(networkfirewall.NetworkFirewall{})),
	newTarget("networkfirewall", "NetworkFirewallCollection", reflect.TypeOf(networkfirewall.NetworkFirewallCollection{})),
	newTarget("networkfirewall", "NetworkFirewallPolicy", reflect.TypeOf(networkfirewall.NetworkFirewallPolicy{})),
	newTarget("networkfirewall", "SecurityRule", reflect.TypeOf(networkfirewall.SecurityRule{})),
	newTarget("networkfirewall", "ServiceList", reflect.TypeOf(networkfirewall.ServiceList{})),
	newTarget("networkfirewall", "UrlList", reflect.TypeOf(networkfirewall.UrlList{})),
	newTarget("networkfirewall", "AddressListSummary", reflect.TypeOf(networkfirewall.AddressListSummary{})),
	newTarget("networkfirewall", "ApplicationGroupSummary", reflect.TypeOf(networkfirewall.ApplicationGroupSummary{})),
	newTarget("networkfirewall", "DecryptionProfileSummary", reflect.TypeOf(networkfirewall.DecryptionProfileSummary{})),
	newTarget("networkfirewall", "DecryptionRuleSummary", reflect.TypeOf(networkfirewall.DecryptionRuleSummary{})),
	newTarget("networkfirewall", "MappedSecretSummary", reflect.TypeOf(networkfirewall.MappedSecretSummary{})),
	newTarget("networkfirewall", "NetworkFirewallPolicySummary", reflect.TypeOf(networkfirewall.NetworkFirewallPolicySummary{})),
	newTarget("networkfirewall", "NetworkFirewallSummary", reflect.TypeOf(networkfirewall.NetworkFirewallSummary{})),
	newTarget("networkfirewall", "SecurityRuleSummary", reflect.TypeOf(networkfirewall.SecurityRuleSummary{})),
	newTarget("networkfirewall", "ServiceListSummary", reflect.TypeOf(networkfirewall.ServiceListSummary{})),
	newTarget("networkfirewall", "ServiceSummary", reflect.TypeOf(networkfirewall.ServiceSummary{})),
	newTarget("networkfirewall", "UrlListSummary", reflect.TypeOf(networkfirewall.UrlListSummary{})),

	// Oce CRD support
	newTarget("oce", "CreateOceInstanceDetails", reflect.TypeOf(oce.CreateOceInstanceDetails{})),
	newTarget("oce", "UpdateOceInstanceDetails", reflect.TypeOf(oce.UpdateOceInstanceDetails{})),
	newTarget("oce", "OceInstance", reflect.TypeOf(oce.OceInstance{})),
	newTarget("oce", "OceInstanceSummary", reflect.TypeOf(oce.OceInstanceSummary{})),

	// Onesubscription CRD support
	newTarget("onesubscription", "SubscriptionSummary", reflect.TypeOf(onesubscription.SubscriptionSummary{})),

	// Opa CRD support
	newTarget("opa", "CreateOpaInstanceDetails", reflect.TypeOf(opa.CreateOpaInstanceDetails{})),
	newTarget("opa", "UpdateOpaInstanceDetails", reflect.TypeOf(opa.UpdateOpaInstanceDetails{})),
	newTarget("opa", "OpaInstance", reflect.TypeOf(opa.OpaInstance{})),
	newTarget("opa", "OpaInstanceCollection", reflect.TypeOf(opa.OpaInstanceCollection{})),
	newTarget("opa", "OpaInstanceSummary", reflect.TypeOf(opa.OpaInstanceSummary{})),

	// Opensearch CRD support
	newTarget("opensearch", "CreateOpensearchClusterDetails", reflect.TypeOf(opensearch.CreateOpensearchClusterDetails{})),
	newTarget("opensearch", "UpdateOpensearchClusterDetails", reflect.TypeOf(opensearch.UpdateOpensearchClusterDetails{})),
	newTarget("opensearch", "OpensearchCluster", reflect.TypeOf(opensearch.OpensearchCluster{})),
	newTarget("opensearch", "OpensearchClusterCollection", reflect.TypeOf(opensearch.OpensearchClusterCollection{})),
	newTarget("opensearch", "OpensearchClusterSummary", reflect.TypeOf(opensearch.OpensearchClusterSummary{})),

	// Operatoraccesscontrol CRD support
	newTarget("operatoraccesscontrol", "CreateOperatorControlAssignmentDetails", reflect.TypeOf(operatoraccesscontrol.CreateOperatorControlAssignmentDetails{})),
	newTarget("operatoraccesscontrol", "CreateOperatorControlDetails", reflect.TypeOf(operatoraccesscontrol.CreateOperatorControlDetails{})),
	newTarget("operatoraccesscontrol", "UpdateOperatorControlAssignmentDetails", reflect.TypeOf(operatoraccesscontrol.UpdateOperatorControlAssignmentDetails{})),
	newTarget("operatoraccesscontrol", "UpdateOperatorControlDetails", reflect.TypeOf(operatoraccesscontrol.UpdateOperatorControlDetails{})),
	newTarget("operatoraccesscontrol", "OperatorControl", reflect.TypeOf(operatoraccesscontrol.OperatorControl{})),
	newTarget("operatoraccesscontrol", "OperatorControlAssignment", reflect.TypeOf(operatoraccesscontrol.OperatorControlAssignment{})),
	newTarget("operatoraccesscontrol", "OperatorControlAssignmentCollection", reflect.TypeOf(operatoraccesscontrol.OperatorControlAssignmentCollection{})),
	newTarget("operatoraccesscontrol", "OperatorControlCollection", reflect.TypeOf(operatoraccesscontrol.OperatorControlCollection{})),
	newTarget("operatoraccesscontrol", "OperatorControlAssignmentSummary", reflect.TypeOf(operatoraccesscontrol.OperatorControlAssignmentSummary{})),
	newTarget("operatoraccesscontrol", "OperatorControlSummary", reflect.TypeOf(operatoraccesscontrol.OperatorControlSummary{})),

	// Opsi CRD support
	newTarget("opsi", "CreateAwrHubDetails", reflect.TypeOf(opsi.CreateAwrHubDetails{})),
	newTarget("opsi", "CreateAwrHubSourceDetails", reflect.TypeOf(opsi.CreateAwrHubSourceDetails{})),
	newTarget("opsi", "CreateChargebackPlanReportDetails", reflect.TypeOf(opsi.CreateChargebackPlanReportDetails{})),
	newTarget("opsi", "CreateEnterpriseManagerBridgeDetails", reflect.TypeOf(opsi.CreateEnterpriseManagerBridgeDetails{})),
	newTarget("opsi", "CreateNewsReportDetails", reflect.TypeOf(opsi.CreateNewsReportDetails{})),
	newTarget("opsi", "CreateOperationsInsightsPrivateEndpointDetails", reflect.TypeOf(opsi.CreateOperationsInsightsPrivateEndpointDetails{})),
	newTarget("opsi", "CreateOperationsInsightsWarehouseDetails", reflect.TypeOf(opsi.CreateOperationsInsightsWarehouseDetails{})),
	newTarget("opsi", "CreateOperationsInsightsWarehouseUserDetails", reflect.TypeOf(opsi.CreateOperationsInsightsWarehouseUserDetails{})),
	newTarget("opsi", "UpdateAwrHubDetails", reflect.TypeOf(opsi.UpdateAwrHubDetails{})),
	newTarget("opsi", "UpdateAwrHubSourceDetails", reflect.TypeOf(opsi.UpdateAwrHubSourceDetails{})),
	newTarget("opsi", "UpdateChargebackPlanDetails", reflect.TypeOf(opsi.UpdateChargebackPlanDetails{})),
	newTarget("opsi", "UpdateChargebackPlanReportDetails", reflect.TypeOf(opsi.UpdateChargebackPlanReportDetails{})),
	newTarget("opsi", "UpdateEnterpriseManagerBridgeDetails", reflect.TypeOf(opsi.UpdateEnterpriseManagerBridgeDetails{})),
	newTarget("opsi", "UpdateNewsReportDetails", reflect.TypeOf(opsi.UpdateNewsReportDetails{})),
	newTarget("opsi", "UpdateOperationsInsightsPrivateEndpointDetails", reflect.TypeOf(opsi.UpdateOperationsInsightsPrivateEndpointDetails{})),
	newTarget("opsi", "UpdateOperationsInsightsWarehouseDetails", reflect.TypeOf(opsi.UpdateOperationsInsightsWarehouseDetails{})),
	newTarget("opsi", "UpdateOperationsInsightsWarehouseUserDetails", reflect.TypeOf(opsi.UpdateOperationsInsightsWarehouseUserDetails{})),
	newTarget("opsi", "ChargebackPlanDetails", reflect.TypeOf(opsi.ChargebackPlanDetails{})),
	newTarget("opsi", "AwrHub", reflect.TypeOf(opsi.AwrHub{})),
	newTarget("opsi", "AwrHubSource", reflect.TypeOf(opsi.AwrHubSource{})),
	newTarget("opsi", "AwrHubSources", reflect.TypeOf(opsi.AwrHubSources{})),
	newTarget("opsi", "AwrHubs", reflect.TypeOf(opsi.AwrHubs{})),
	newTarget("opsi", "ChargebackPlan", reflect.TypeOf(opsi.ChargebackPlan{})),
	newTarget("opsi", "ChargebackPlanCollection", reflect.TypeOf(opsi.ChargebackPlanCollection{})),
	newTarget("opsi", "ChargebackPlanReport", reflect.TypeOf(opsi.ChargebackPlanReport{})),
	newTarget("opsi", "ChargebackPlanReportCollection", reflect.TypeOf(opsi.ChargebackPlanReportCollection{})),
	newTarget("opsi", "DatabaseInsights", reflect.TypeOf(opsi.DatabaseInsights{})),
	newTarget("opsi", "EnterpriseManagerBridge", reflect.TypeOf(opsi.EnterpriseManagerBridge{})),
	newTarget("opsi", "EnterpriseManagerBridgeCollection", reflect.TypeOf(opsi.EnterpriseManagerBridgeCollection{})),
	newTarget("opsi", "EnterpriseManagerBridges", reflect.TypeOf(opsi.EnterpriseManagerBridges{})),
	newTarget("opsi", "ExadataInsights", reflect.TypeOf(opsi.ExadataInsights{})),
	newTarget("opsi", "HostInsights", reflect.TypeOf(opsi.HostInsights{})),
	newTarget("opsi", "NewsReport", reflect.TypeOf(opsi.NewsReport{})),
	newTarget("opsi", "NewsReportCollection", reflect.TypeOf(opsi.NewsReportCollection{})),
	newTarget("opsi", "NewsReports", reflect.TypeOf(opsi.NewsReports{})),
	newTarget("opsi", "OperationsInsightsPrivateEndpoint", reflect.TypeOf(opsi.OperationsInsightsPrivateEndpoint{})),
	newTarget("opsi", "OperationsInsightsPrivateEndpointCollection", reflect.TypeOf(opsi.OperationsInsightsPrivateEndpointCollection{})),
	newTarget("opsi", "OperationsInsightsWarehouse", reflect.TypeOf(opsi.OperationsInsightsWarehouse{})),
	newTarget("opsi", "OperationsInsightsWarehouseUser", reflect.TypeOf(opsi.OperationsInsightsWarehouseUser{})),
	newTarget("opsi", "OperationsInsightsWarehouseUsers", reflect.TypeOf(opsi.OperationsInsightsWarehouseUsers{})),
	newTarget("opsi", "OperationsInsightsWarehouses", reflect.TypeOf(opsi.OperationsInsightsWarehouses{})),
	newTarget("opsi", "OpsiConfigurations", reflect.TypeOf(opsi.OpsiConfigurations{})),
	newTarget("opsi", "AwrHubSourceSummary", reflect.TypeOf(opsi.AwrHubSourceSummary{})),
	newTarget("opsi", "AwrHubSummary", reflect.TypeOf(opsi.AwrHubSummary{})),
	newTarget("opsi", "ChargebackPlanReportSummary", reflect.TypeOf(opsi.ChargebackPlanReportSummary{})),
	newTarget("opsi", "ChargebackPlanSummary", reflect.TypeOf(opsi.ChargebackPlanSummary{})),
	newTarget("opsi", "EnterpriseManagerBridgeSummary", reflect.TypeOf(opsi.EnterpriseManagerBridgeSummary{})),
	newTarget("opsi", "NewsReportSummary", reflect.TypeOf(opsi.NewsReportSummary{})),
	newTarget("opsi", "OperationsInsightsPrivateEndpointSummary", reflect.TypeOf(opsi.OperationsInsightsPrivateEndpointSummary{})),
	newTarget("opsi", "OperationsInsightsWarehouseSummary", reflect.TypeOf(opsi.OperationsInsightsWarehouseSummary{})),
	newTarget("opsi", "OperationsInsightsWarehouseUserSummary", reflect.TypeOf(opsi.OperationsInsightsWarehouseUserSummary{})),

	// Optimizer CRD support
	newTarget("optimizer", "CreateProfileDetails", reflect.TypeOf(optimizer.CreateProfileDetails{})),
	newTarget("optimizer", "UpdateProfileDetails", reflect.TypeOf(optimizer.UpdateProfileDetails{})),
	newTarget("optimizer", "Profile", reflect.TypeOf(optimizer.Profile{})),
	newTarget("optimizer", "ProfileCollection", reflect.TypeOf(optimizer.ProfileCollection{})),
	newTarget("optimizer", "ProfileSummary", reflect.TypeOf(optimizer.ProfileSummary{})),

	// Osmanagementhub CRD support
	newTarget("osmanagementhub", "CreateLifecycleEnvironmentDetails", reflect.TypeOf(osmanagementhub.CreateLifecycleEnvironmentDetails{})),
	newTarget("osmanagementhub", "CreateManagedInstanceGroupDetails", reflect.TypeOf(osmanagementhub.CreateManagedInstanceGroupDetails{})),
	newTarget("osmanagementhub", "CreateManagementStationDetails", reflect.TypeOf(osmanagementhub.CreateManagementStationDetails{})),
	newTarget("osmanagementhub", "CreateScheduledJobDetails", reflect.TypeOf(osmanagementhub.CreateScheduledJobDetails{})),
	newTarget("osmanagementhub", "UpdateLifecycleEnvironmentDetails", reflect.TypeOf(osmanagementhub.UpdateLifecycleEnvironmentDetails{})),
	newTarget("osmanagementhub", "UpdateManagedInstanceGroupDetails", reflect.TypeOf(osmanagementhub.UpdateManagedInstanceGroupDetails{})),
	newTarget("osmanagementhub", "UpdateManagementStationDetails", reflect.TypeOf(osmanagementhub.UpdateManagementStationDetails{})),
	newTarget("osmanagementhub", "UpdateProfileDetails", reflect.TypeOf(osmanagementhub.UpdateProfileDetails{})),
	newTarget("osmanagementhub", "UpdateScheduledJobDetails", reflect.TypeOf(osmanagementhub.UpdateScheduledJobDetails{})),
	newTarget("osmanagementhub", "LifecycleEnvironmentDetails", reflect.TypeOf(osmanagementhub.LifecycleEnvironmentDetails{})),
	newTarget("osmanagementhub", "ManagedInstanceGroupDetails", reflect.TypeOf(osmanagementhub.ManagedInstanceGroupDetails{})),
	newTarget("osmanagementhub", "ManagementStationDetails", reflect.TypeOf(osmanagementhub.ManagementStationDetails{})),
	newTarget("osmanagementhub", "SoftwareSourceDetails", reflect.TypeOf(osmanagementhub.SoftwareSourceDetails{})),
	newTarget("osmanagementhub", "LifecycleEnvironment", reflect.TypeOf(osmanagementhub.LifecycleEnvironment{})),
	newTarget("osmanagementhub", "LifecycleEnvironmentCollection", reflect.TypeOf(osmanagementhub.LifecycleEnvironmentCollection{})),
	newTarget("osmanagementhub", "ManagedInstanceGroup", reflect.TypeOf(osmanagementhub.ManagedInstanceGroup{})),
	newTarget("osmanagementhub", "ManagedInstanceGroupCollection", reflect.TypeOf(osmanagementhub.ManagedInstanceGroupCollection{})),
	newTarget("osmanagementhub", "ManagementStation", reflect.TypeOf(osmanagementhub.ManagementStation{})),
	newTarget("osmanagementhub", "ManagementStationCollection", reflect.TypeOf(osmanagementhub.ManagementStationCollection{})),
	newTarget("osmanagementhub", "ProfileCollection", reflect.TypeOf(osmanagementhub.ProfileCollection{})),
	newTarget("osmanagementhub", "ScheduledJob", reflect.TypeOf(osmanagementhub.ScheduledJob{})),
	newTarget("osmanagementhub", "ScheduledJobCollection", reflect.TypeOf(osmanagementhub.ScheduledJobCollection{})),
	newTarget("osmanagementhub", "SoftwareSourceCollection", reflect.TypeOf(osmanagementhub.SoftwareSourceCollection{})),
	newTarget("osmanagementhub", "LifecycleEnvironmentSummary", reflect.TypeOf(osmanagementhub.LifecycleEnvironmentSummary{})),
	newTarget("osmanagementhub", "ManagedInstanceGroupSummary", reflect.TypeOf(osmanagementhub.ManagedInstanceGroupSummary{})),
	newTarget("osmanagementhub", "ManagementStationSummary", reflect.TypeOf(osmanagementhub.ManagementStationSummary{})),
	newTarget("osmanagementhub", "ProfileSummary", reflect.TypeOf(osmanagementhub.ProfileSummary{})),
	newTarget("osmanagementhub", "ScheduledJobSummary", reflect.TypeOf(osmanagementhub.ScheduledJobSummary{})),

	// Osubsubscription CRD support
	newTarget("osubsubscription", "SubscriptionSummary", reflect.TypeOf(osubsubscription.SubscriptionSummary{})),

	// Psa CRD support
	newTarget("psa", "CreatePrivateServiceAccessDetails", reflect.TypeOf(psa.CreatePrivateServiceAccessDetails{})),
	newTarget("psa", "UpdatePrivateServiceAccessDetails", reflect.TypeOf(psa.UpdatePrivateServiceAccessDetails{})),
	newTarget("psa", "PrivateServiceAccess", reflect.TypeOf(psa.PrivateServiceAccess{})),
	newTarget("psa", "PrivateServiceAccessCollection", reflect.TypeOf(psa.PrivateServiceAccessCollection{})),
	newTarget("psa", "PrivateServiceAccessSummary", reflect.TypeOf(psa.PrivateServiceAccessSummary{})),

	// Recovery CRD support
	newTarget("recovery", "CreateProtectedDatabaseDetails", reflect.TypeOf(recovery.CreateProtectedDatabaseDetails{})),
	newTarget("recovery", "CreateProtectionPolicyDetails", reflect.TypeOf(recovery.CreateProtectionPolicyDetails{})),
	newTarget("recovery", "CreateRecoveryServiceSubnetDetails", reflect.TypeOf(recovery.CreateRecoveryServiceSubnetDetails{})),
	newTarget("recovery", "UpdateProtectedDatabaseDetails", reflect.TypeOf(recovery.UpdateProtectedDatabaseDetails{})),
	newTarget("recovery", "UpdateProtectionPolicyDetails", reflect.TypeOf(recovery.UpdateProtectionPolicyDetails{})),
	newTarget("recovery", "UpdateRecoveryServiceSubnetDetails", reflect.TypeOf(recovery.UpdateRecoveryServiceSubnetDetails{})),
	newTarget("recovery", "RecoveryServiceSubnetDetails", reflect.TypeOf(recovery.RecoveryServiceSubnetDetails{})),
	newTarget("recovery", "ProtectedDatabase", reflect.TypeOf(recovery.ProtectedDatabase{})),
	newTarget("recovery", "ProtectedDatabaseCollection", reflect.TypeOf(recovery.ProtectedDatabaseCollection{})),
	newTarget("recovery", "ProtectionPolicy", reflect.TypeOf(recovery.ProtectionPolicy{})),
	newTarget("recovery", "ProtectionPolicyCollection", reflect.TypeOf(recovery.ProtectionPolicyCollection{})),
	newTarget("recovery", "RecoveryServiceSubnet", reflect.TypeOf(recovery.RecoveryServiceSubnet{})),
	newTarget("recovery", "RecoveryServiceSubnetCollection", reflect.TypeOf(recovery.RecoveryServiceSubnetCollection{})),
	newTarget("recovery", "ProtectedDatabaseSummary", reflect.TypeOf(recovery.ProtectedDatabaseSummary{})),
	newTarget("recovery", "ProtectionPolicySummary", reflect.TypeOf(recovery.ProtectionPolicySummary{})),
	newTarget("recovery", "RecoveryServiceSubnetSummary", reflect.TypeOf(recovery.RecoveryServiceSubnetSummary{})),

	// Redis CRD support
	newTarget("redis", "CreateRedisClusterDetails", reflect.TypeOf(redis.CreateRedisClusterDetails{})),
	newTarget("redis", "UpdateRedisClusterDetails", reflect.TypeOf(redis.UpdateRedisClusterDetails{})),
	newTarget("redis", "RedisCluster", reflect.TypeOf(redis.RedisCluster{})),
	newTarget("redis", "RedisClusterCollection", reflect.TypeOf(redis.RedisClusterCollection{})),
	newTarget("redis", "RedisClusterSummary", reflect.TypeOf(redis.RedisClusterSummary{})),

	// Resourceanalytics CRD support
	newTarget("resourceanalytics", "CreateMonitoredRegionDetails", reflect.TypeOf(resourceanalytics.CreateMonitoredRegionDetails{})),
	newTarget("resourceanalytics", "CreateResourceAnalyticsInstanceDetails", reflect.TypeOf(resourceanalytics.CreateResourceAnalyticsInstanceDetails{})),
	newTarget("resourceanalytics", "CreateTenancyAttachmentDetails", reflect.TypeOf(resourceanalytics.CreateTenancyAttachmentDetails{})),
	newTarget("resourceanalytics", "UpdateResourceAnalyticsInstanceDetails", reflect.TypeOf(resourceanalytics.UpdateResourceAnalyticsInstanceDetails{})),
	newTarget("resourceanalytics", "UpdateTenancyAttachmentDetails", reflect.TypeOf(resourceanalytics.UpdateTenancyAttachmentDetails{})),
	newTarget("resourceanalytics", "MonitoredRegion", reflect.TypeOf(resourceanalytics.MonitoredRegion{})),
	newTarget("resourceanalytics", "MonitoredRegionCollection", reflect.TypeOf(resourceanalytics.MonitoredRegionCollection{})),
	newTarget("resourceanalytics", "ResourceAnalyticsInstance", reflect.TypeOf(resourceanalytics.ResourceAnalyticsInstance{})),
	newTarget("resourceanalytics", "ResourceAnalyticsInstanceCollection", reflect.TypeOf(resourceanalytics.ResourceAnalyticsInstanceCollection{})),
	newTarget("resourceanalytics", "TenancyAttachment", reflect.TypeOf(resourceanalytics.TenancyAttachment{})),
	newTarget("resourceanalytics", "TenancyAttachmentCollection", reflect.TypeOf(resourceanalytics.TenancyAttachmentCollection{})),
	newTarget("resourceanalytics", "MonitoredRegionSummary", reflect.TypeOf(resourceanalytics.MonitoredRegionSummary{})),
	newTarget("resourceanalytics", "ResourceAnalyticsInstanceSummary", reflect.TypeOf(resourceanalytics.ResourceAnalyticsInstanceSummary{})),
	newTarget("resourceanalytics", "TenancyAttachmentSummary", reflect.TypeOf(resourceanalytics.TenancyAttachmentSummary{})),

	// Resourcemanager CRD support
	newTarget("resourcemanager", "CreatePrivateEndpointDetails", reflect.TypeOf(resourcemanager.CreatePrivateEndpointDetails{})),
	newTarget("resourcemanager", "CreateStackDetails", reflect.TypeOf(resourcemanager.CreateStackDetails{})),
	newTarget("resourcemanager", "CreateTemplateDetails", reflect.TypeOf(resourcemanager.CreateTemplateDetails{})),
	newTarget("resourcemanager", "UpdatePrivateEndpointDetails", reflect.TypeOf(resourcemanager.UpdatePrivateEndpointDetails{})),
	newTarget("resourcemanager", "UpdateStackDetails", reflect.TypeOf(resourcemanager.UpdateStackDetails{})),
	newTarget("resourcemanager", "UpdateTemplateDetails", reflect.TypeOf(resourcemanager.UpdateTemplateDetails{})),
	newTarget("resourcemanager", "ConfigurationSourceProviderCollection", reflect.TypeOf(resourcemanager.ConfigurationSourceProviderCollection{})),
	newTarget("resourcemanager", "PrivateEndpoint", reflect.TypeOf(resourcemanager.PrivateEndpoint{})),
	newTarget("resourcemanager", "PrivateEndpointCollection", reflect.TypeOf(resourcemanager.PrivateEndpointCollection{})),
	newTarget("resourcemanager", "Stack", reflect.TypeOf(resourcemanager.Stack{})),
	newTarget("resourcemanager", "Template", reflect.TypeOf(resourcemanager.Template{})),
	newTarget("resourcemanager", "PrivateEndpointSummary", reflect.TypeOf(resourcemanager.PrivateEndpointSummary{})),
	newTarget("resourcemanager", "StackSummary", reflect.TypeOf(resourcemanager.StackSummary{})),
	newTarget("resourcemanager", "TemplateSummary", reflect.TypeOf(resourcemanager.TemplateSummary{})),

	// Resourcescheduler CRD support
	newTarget("resourcescheduler", "CreateScheduleDetails", reflect.TypeOf(resourcescheduler.CreateScheduleDetails{})),
	newTarget("resourcescheduler", "UpdateScheduleDetails", reflect.TypeOf(resourcescheduler.UpdateScheduleDetails{})),
	newTarget("resourcescheduler", "Schedule", reflect.TypeOf(resourcescheduler.Schedule{})),
	newTarget("resourcescheduler", "ScheduleCollection", reflect.TypeOf(resourcescheduler.ScheduleCollection{})),
	newTarget("resourcescheduler", "ScheduleSummary", reflect.TypeOf(resourcescheduler.ScheduleSummary{})),

	// Rover CRD support
	newTarget("rover", "CreateRoverClusterDetails", reflect.TypeOf(rover.CreateRoverClusterDetails{})),
	newTarget("rover", "CreateRoverNodeDetails", reflect.TypeOf(rover.CreateRoverNodeDetails{})),
	newTarget("rover", "UpdateRoverClusterDetails", reflect.TypeOf(rover.UpdateRoverClusterDetails{})),
	newTarget("rover", "UpdateRoverNodeDetails", reflect.TypeOf(rover.UpdateRoverNodeDetails{})),
	newTarget("rover", "RoverCluster", reflect.TypeOf(rover.RoverCluster{})),
	newTarget("rover", "RoverClusterCollection", reflect.TypeOf(rover.RoverClusterCollection{})),
	newTarget("rover", "RoverNode", reflect.TypeOf(rover.RoverNode{})),
	newTarget("rover", "RoverNodeCollection", reflect.TypeOf(rover.RoverNodeCollection{})),
	newTarget("rover", "RoverClusterSummary", reflect.TypeOf(rover.RoverClusterSummary{})),
	newTarget("rover", "RoverNodeSummary", reflect.TypeOf(rover.RoverNodeSummary{})),

	// Sch CRD support
	newTarget("sch", "CreateServiceConnectorDetails", reflect.TypeOf(sch.CreateServiceConnectorDetails{})),
	newTarget("sch", "UpdateServiceConnectorDetails", reflect.TypeOf(sch.UpdateServiceConnectorDetails{})),
	newTarget("sch", "ServiceConnector", reflect.TypeOf(sch.ServiceConnector{})),
	newTarget("sch", "ServiceConnectorCollection", reflect.TypeOf(sch.ServiceConnectorCollection{})),
	newTarget("sch", "ServiceConnectorSummary", reflect.TypeOf(sch.ServiceConnectorSummary{})),

	// Securityattribute CRD support
	newTarget("securityattribute", "CreateSecurityAttributeDetails", reflect.TypeOf(securityattribute.CreateSecurityAttributeDetails{})),
	newTarget("securityattribute", "CreateSecurityAttributeNamespaceDetails", reflect.TypeOf(securityattribute.CreateSecurityAttributeNamespaceDetails{})),
	newTarget("securityattribute", "UpdateSecurityAttributeDetails", reflect.TypeOf(securityattribute.UpdateSecurityAttributeDetails{})),
	newTarget("securityattribute", "UpdateSecurityAttributeNamespaceDetails", reflect.TypeOf(securityattribute.UpdateSecurityAttributeNamespaceDetails{})),
	newTarget("securityattribute", "SecurityAttribute", reflect.TypeOf(securityattribute.SecurityAttribute{})),
	newTarget("securityattribute", "SecurityAttributeNamespace", reflect.TypeOf(securityattribute.SecurityAttributeNamespace{})),
	newTarget("securityattribute", "SecurityAttributeNamespaceSummary", reflect.TypeOf(securityattribute.SecurityAttributeNamespaceSummary{})),
	newTarget("securityattribute", "SecurityAttributeSummary", reflect.TypeOf(securityattribute.SecurityAttributeSummary{})),

	// Self CRD support
	newTarget("self", "CreateSubscriptionDetails", reflect.TypeOf(self.CreateSubscriptionDetails{})),
	newTarget("self", "UpdateSubscriptionDetails", reflect.TypeOf(self.UpdateSubscriptionDetails{})),
	newTarget("self", "SubscriptionDetails", reflect.TypeOf(self.SubscriptionDetails{})),
	newTarget("self", "Subscription", reflect.TypeOf(self.Subscription{})),
	newTarget("self", "SubscriptionCollection", reflect.TypeOf(self.SubscriptionCollection{})),
	newTarget("self", "SubscriptionSummary", reflect.TypeOf(self.SubscriptionSummary{})),

	// Servicecatalog CRD support
	newTarget("servicecatalog", "CreatePrivateApplicationDetails", reflect.TypeOf(servicecatalog.CreatePrivateApplicationDetails{})),
	newTarget("servicecatalog", "CreateServiceCatalogDetails", reflect.TypeOf(servicecatalog.CreateServiceCatalogDetails{})),
	newTarget("servicecatalog", "UpdatePrivateApplicationDetails", reflect.TypeOf(servicecatalog.UpdatePrivateApplicationDetails{})),
	newTarget("servicecatalog", "UpdateServiceCatalogDetails", reflect.TypeOf(servicecatalog.UpdateServiceCatalogDetails{})),
	newTarget("servicecatalog", "PrivateApplication", reflect.TypeOf(servicecatalog.PrivateApplication{})),
	newTarget("servicecatalog", "PrivateApplicationCollection", reflect.TypeOf(servicecatalog.PrivateApplicationCollection{})),
	newTarget("servicecatalog", "ServiceCatalog", reflect.TypeOf(servicecatalog.ServiceCatalog{})),
	newTarget("servicecatalog", "ServiceCatalogCollection", reflect.TypeOf(servicecatalog.ServiceCatalogCollection{})),
	newTarget("servicecatalog", "PrivateApplicationSummary", reflect.TypeOf(servicecatalog.PrivateApplicationSummary{})),
	newTarget("servicecatalog", "ServiceCatalogSummary", reflect.TypeOf(servicecatalog.ServiceCatalogSummary{})),

	// Servicemanagerproxy CRD support
	newTarget("servicemanagerproxy", "ServiceEnvironment", reflect.TypeOf(servicemanagerproxy.ServiceEnvironment{})),
	newTarget("servicemanagerproxy", "ServiceEnvironmentCollection", reflect.TypeOf(servicemanagerproxy.ServiceEnvironmentCollection{})),
	newTarget("servicemanagerproxy", "ServiceEnvironmentSummary", reflect.TypeOf(servicemanagerproxy.ServiceEnvironmentSummary{})),

	// Stackmonitoring CRD support
	newTarget("stackmonitoring", "CreateAlarmConditionDetails", reflect.TypeOf(stackmonitoring.CreateAlarmConditionDetails{})),
	newTarget("stackmonitoring", "CreateBaselineableMetricDetails", reflect.TypeOf(stackmonitoring.CreateBaselineableMetricDetails{})),
	newTarget("stackmonitoring", "CreateDiscoveryJobDetails", reflect.TypeOf(stackmonitoring.CreateDiscoveryJobDetails{})),
	newTarget("stackmonitoring", "CreateMaintenanceWindowDetails", reflect.TypeOf(stackmonitoring.CreateMaintenanceWindowDetails{})),
	newTarget("stackmonitoring", "CreateMetricExtensionDetails", reflect.TypeOf(stackmonitoring.CreateMetricExtensionDetails{})),
	newTarget("stackmonitoring", "CreateMonitoredResourceDetails", reflect.TypeOf(stackmonitoring.CreateMonitoredResourceDetails{})),
	newTarget("stackmonitoring", "CreateMonitoredResourceTypeDetails", reflect.TypeOf(stackmonitoring.CreateMonitoredResourceTypeDetails{})),
	newTarget("stackmonitoring", "CreateMonitoringTemplateDetails", reflect.TypeOf(stackmonitoring.CreateMonitoringTemplateDetails{})),
	newTarget("stackmonitoring", "CreateProcessSetDetails", reflect.TypeOf(stackmonitoring.CreateProcessSetDetails{})),
	newTarget("stackmonitoring", "UpdateAlarmConditionDetails", reflect.TypeOf(stackmonitoring.UpdateAlarmConditionDetails{})),
	newTarget("stackmonitoring", "UpdateBaselineableMetricDetails", reflect.TypeOf(stackmonitoring.UpdateBaselineableMetricDetails{})),
	newTarget("stackmonitoring", "UpdateMaintenanceWindowDetails", reflect.TypeOf(stackmonitoring.UpdateMaintenanceWindowDetails{})),
	newTarget("stackmonitoring", "UpdateMetricExtensionDetails", reflect.TypeOf(stackmonitoring.UpdateMetricExtensionDetails{})),
	newTarget("stackmonitoring", "UpdateMonitoredResourceDetails", reflect.TypeOf(stackmonitoring.UpdateMonitoredResourceDetails{})),
	newTarget("stackmonitoring", "UpdateMonitoredResourceTypeDetails", reflect.TypeOf(stackmonitoring.UpdateMonitoredResourceTypeDetails{})),
	newTarget("stackmonitoring", "UpdateMonitoringTemplateDetails", reflect.TypeOf(stackmonitoring.UpdateMonitoringTemplateDetails{})),
	newTarget("stackmonitoring", "UpdateProcessSetDetails", reflect.TypeOf(stackmonitoring.UpdateProcessSetDetails{})),
	newTarget("stackmonitoring", "MonitoredResourceDetails", reflect.TypeOf(stackmonitoring.MonitoredResourceDetails{})),
	newTarget("stackmonitoring", "AlarmCondition", reflect.TypeOf(stackmonitoring.AlarmCondition{})),
	newTarget("stackmonitoring", "AlarmConditionCollection", reflect.TypeOf(stackmonitoring.AlarmConditionCollection{})),
	newTarget("stackmonitoring", "BaselineableMetric", reflect.TypeOf(stackmonitoring.BaselineableMetric{})),
	newTarget("stackmonitoring", "ConfigCollection", reflect.TypeOf(stackmonitoring.ConfigCollection{})),
	newTarget("stackmonitoring", "DiscoveryJob", reflect.TypeOf(stackmonitoring.DiscoveryJob{})),
	newTarget("stackmonitoring", "DiscoveryJobCollection", reflect.TypeOf(stackmonitoring.DiscoveryJobCollection{})),
	newTarget("stackmonitoring", "MaintenanceWindow", reflect.TypeOf(stackmonitoring.MaintenanceWindow{})),
	newTarget("stackmonitoring", "MaintenanceWindowCollection", reflect.TypeOf(stackmonitoring.MaintenanceWindowCollection{})),
	newTarget("stackmonitoring", "MetricExtension", reflect.TypeOf(stackmonitoring.MetricExtension{})),
	newTarget("stackmonitoring", "MetricExtensionCollection", reflect.TypeOf(stackmonitoring.MetricExtensionCollection{})),
	newTarget("stackmonitoring", "MonitoredResource", reflect.TypeOf(stackmonitoring.MonitoredResource{})),
	newTarget("stackmonitoring", "MonitoredResourceCollection", reflect.TypeOf(stackmonitoring.MonitoredResourceCollection{})),
	newTarget("stackmonitoring", "MonitoredResourceType", reflect.TypeOf(stackmonitoring.MonitoredResourceType{})),
	newTarget("stackmonitoring", "MonitoringTemplate", reflect.TypeOf(stackmonitoring.MonitoringTemplate{})),
	newTarget("stackmonitoring", "MonitoringTemplateCollection", reflect.TypeOf(stackmonitoring.MonitoringTemplateCollection{})),
	newTarget("stackmonitoring", "ProcessSet", reflect.TypeOf(stackmonitoring.ProcessSet{})),
	newTarget("stackmonitoring", "ProcessSetCollection", reflect.TypeOf(stackmonitoring.ProcessSetCollection{})),
	newTarget("stackmonitoring", "AlarmConditionSummary", reflect.TypeOf(stackmonitoring.AlarmConditionSummary{})),
	newTarget("stackmonitoring", "BaselineableMetricSummary", reflect.TypeOf(stackmonitoring.BaselineableMetricSummary{})),
	newTarget("stackmonitoring", "DiscoveryJobSummary", reflect.TypeOf(stackmonitoring.DiscoveryJobSummary{})),
	newTarget("stackmonitoring", "MaintenanceWindowSummary", reflect.TypeOf(stackmonitoring.MaintenanceWindowSummary{})),
	newTarget("stackmonitoring", "MetricExtensionSummary", reflect.TypeOf(stackmonitoring.MetricExtensionSummary{})),
	newTarget("stackmonitoring", "MonitoredResourceSummary", reflect.TypeOf(stackmonitoring.MonitoredResourceSummary{})),
	newTarget("stackmonitoring", "MonitoredResourceTypeSummary", reflect.TypeOf(stackmonitoring.MonitoredResourceTypeSummary{})),
	newTarget("stackmonitoring", "MonitoringTemplateSummary", reflect.TypeOf(stackmonitoring.MonitoringTemplateSummary{})),
	newTarget("stackmonitoring", "ProcessSetSummary", reflect.TypeOf(stackmonitoring.ProcessSetSummary{})),

	// Tenantmanagercontrolplane CRD support
	newTarget("tenantmanagercontrolplane", "CreateDomainDetails", reflect.TypeOf(tenantmanagercontrolplane.CreateDomainDetails{})),
	newTarget("tenantmanagercontrolplane", "UpdateDomainDetails", reflect.TypeOf(tenantmanagercontrolplane.UpdateDomainDetails{})),
	newTarget("tenantmanagercontrolplane", "UpdateOrganizationDetails", reflect.TypeOf(tenantmanagercontrolplane.UpdateOrganizationDetails{})),
	newTarget("tenantmanagercontrolplane", "Domain", reflect.TypeOf(tenantmanagercontrolplane.Domain{})),
	newTarget("tenantmanagercontrolplane", "DomainCollection", reflect.TypeOf(tenantmanagercontrolplane.DomainCollection{})),
	newTarget("tenantmanagercontrolplane", "Organization", reflect.TypeOf(tenantmanagercontrolplane.Organization{})),
	newTarget("tenantmanagercontrolplane", "OrganizationCollection", reflect.TypeOf(tenantmanagercontrolplane.OrganizationCollection{})),
	newTarget("tenantmanagercontrolplane", "DomainSummary", reflect.TypeOf(tenantmanagercontrolplane.DomainSummary{})),
	newTarget("tenantmanagercontrolplane", "OrganizationSummary", reflect.TypeOf(tenantmanagercontrolplane.OrganizationSummary{})),

	// Vbsinst CRD support
	newTarget("vbsinst", "CreateVbsInstanceDetails", reflect.TypeOf(vbsinst.CreateVbsInstanceDetails{})),
	newTarget("vbsinst", "UpdateVbsInstanceDetails", reflect.TypeOf(vbsinst.UpdateVbsInstanceDetails{})),
	newTarget("vbsinst", "VbsInstance", reflect.TypeOf(vbsinst.VbsInstance{})),
	newTarget("vbsinst", "VbsInstanceSummary", reflect.TypeOf(vbsinst.VbsInstanceSummary{})),

	// Visualbuilder CRD support
	newTarget("visualbuilder", "CreateVbInstanceDetails", reflect.TypeOf(visualbuilder.CreateVbInstanceDetails{})),
	newTarget("visualbuilder", "UpdateVbInstanceDetails", reflect.TypeOf(visualbuilder.UpdateVbInstanceDetails{})),
	newTarget("visualbuilder", "VbInstance", reflect.TypeOf(visualbuilder.VbInstance{})),
	newTarget("visualbuilder", "VbInstanceSummary", reflect.TypeOf(visualbuilder.VbInstanceSummary{})),

	// Vnmonitoring CRD support
	newTarget("vnmonitoring", "CreatePathAnalyzerTestDetails", reflect.TypeOf(vnmonitoring.CreatePathAnalyzerTestDetails{})),
	newTarget("vnmonitoring", "UpdatePathAnalyzerTestDetails", reflect.TypeOf(vnmonitoring.UpdatePathAnalyzerTestDetails{})),
	newTarget("vnmonitoring", "PathAnalyzerTest", reflect.TypeOf(vnmonitoring.PathAnalyzerTest{})),
	newTarget("vnmonitoring", "PathAnalyzerTestCollection", reflect.TypeOf(vnmonitoring.PathAnalyzerTestCollection{})),
	newTarget("vnmonitoring", "PathAnalyzerTestSummary", reflect.TypeOf(vnmonitoring.PathAnalyzerTestSummary{})),

	// Vulnerabilityscanning CRD support
	newTarget("vulnerabilityscanning", "CreateContainerScanRecipeDetails", reflect.TypeOf(vulnerabilityscanning.CreateContainerScanRecipeDetails{})),
	newTarget("vulnerabilityscanning", "CreateContainerScanTargetDetails", reflect.TypeOf(vulnerabilityscanning.CreateContainerScanTargetDetails{})),
	newTarget("vulnerabilityscanning", "CreateHostScanRecipeDetails", reflect.TypeOf(vulnerabilityscanning.CreateHostScanRecipeDetails{})),
	newTarget("vulnerabilityscanning", "CreateHostScanTargetDetails", reflect.TypeOf(vulnerabilityscanning.CreateHostScanTargetDetails{})),
	newTarget("vulnerabilityscanning", "UpdateContainerScanRecipeDetails", reflect.TypeOf(vulnerabilityscanning.UpdateContainerScanRecipeDetails{})),
	newTarget("vulnerabilityscanning", "UpdateContainerScanTargetDetails", reflect.TypeOf(vulnerabilityscanning.UpdateContainerScanTargetDetails{})),
	newTarget("vulnerabilityscanning", "UpdateHostScanRecipeDetails", reflect.TypeOf(vulnerabilityscanning.UpdateHostScanRecipeDetails{})),
	newTarget("vulnerabilityscanning", "UpdateHostScanTargetDetails", reflect.TypeOf(vulnerabilityscanning.UpdateHostScanTargetDetails{})),
	newTarget("vulnerabilityscanning", "ContainerScanRecipe", reflect.TypeOf(vulnerabilityscanning.ContainerScanRecipe{})),
	newTarget("vulnerabilityscanning", "ContainerScanTarget", reflect.TypeOf(vulnerabilityscanning.ContainerScanTarget{})),
	newTarget("vulnerabilityscanning", "HostScanRecipe", reflect.TypeOf(vulnerabilityscanning.HostScanRecipe{})),
	newTarget("vulnerabilityscanning", "HostScanTarget", reflect.TypeOf(vulnerabilityscanning.HostScanTarget{})),
	newTarget("vulnerabilityscanning", "ContainerScanRecipeSummary", reflect.TypeOf(vulnerabilityscanning.ContainerScanRecipeSummary{})),
	newTarget("vulnerabilityscanning", "ContainerScanTargetSummary", reflect.TypeOf(vulnerabilityscanning.ContainerScanTargetSummary{})),
	newTarget("vulnerabilityscanning", "HostScanRecipeSummary", reflect.TypeOf(vulnerabilityscanning.HostScanRecipeSummary{})),
	newTarget("vulnerabilityscanning", "HostScanTargetSummary", reflect.TypeOf(vulnerabilityscanning.HostScanTargetSummary{})),

	// Waa CRD support
	newTarget("waa", "CreateWebAppAccelerationPolicyDetails", reflect.TypeOf(waa.CreateWebAppAccelerationPolicyDetails{})),
	newTarget("waa", "UpdateWebAppAccelerationDetails", reflect.TypeOf(waa.UpdateWebAppAccelerationDetails{})),
	newTarget("waa", "UpdateWebAppAccelerationPolicyDetails", reflect.TypeOf(waa.UpdateWebAppAccelerationPolicyDetails{})),
	newTarget("waa", "WebAppAccelerationCollection", reflect.TypeOf(waa.WebAppAccelerationCollection{})),
	newTarget("waa", "WebAppAccelerationPolicy", reflect.TypeOf(waa.WebAppAccelerationPolicy{})),
	newTarget("waa", "WebAppAccelerationPolicyCollection", reflect.TypeOf(waa.WebAppAccelerationPolicyCollection{})),
	newTarget("waa", "WebAppAccelerationPolicySummary", reflect.TypeOf(waa.WebAppAccelerationPolicySummary{})),

	// Waas CRD support
	newTarget("waas", "CreateAddressListDetails", reflect.TypeOf(waas.CreateAddressListDetails{})),
	newTarget("waas", "CreateCertificateDetails", reflect.TypeOf(waas.CreateCertificateDetails{})),
	newTarget("waas", "CreateCustomProtectionRuleDetails", reflect.TypeOf(waas.CreateCustomProtectionRuleDetails{})),
	newTarget("waas", "CreateHttpRedirectDetails", reflect.TypeOf(waas.CreateHttpRedirectDetails{})),
	newTarget("waas", "CreateWaasPolicyDetails", reflect.TypeOf(waas.CreateWaasPolicyDetails{})),
	newTarget("waas", "UpdateAddressListDetails", reflect.TypeOf(waas.UpdateAddressListDetails{})),
	newTarget("waas", "UpdateCertificateDetails", reflect.TypeOf(waas.UpdateCertificateDetails{})),
	newTarget("waas", "UpdateCustomProtectionRuleDetails", reflect.TypeOf(waas.UpdateCustomProtectionRuleDetails{})),
	newTarget("waas", "UpdateHttpRedirectDetails", reflect.TypeOf(waas.UpdateHttpRedirectDetails{})),
	newTarget("waas", "UpdateWaasPolicyDetails", reflect.TypeOf(waas.UpdateWaasPolicyDetails{})),
	newTarget("waas", "AddressList", reflect.TypeOf(waas.AddressList{})),
	newTarget("waas", "Certificate", reflect.TypeOf(waas.Certificate{})),
	newTarget("waas", "CustomProtectionRule", reflect.TypeOf(waas.CustomProtectionRule{})),
	newTarget("waas", "HttpRedirect", reflect.TypeOf(waas.HttpRedirect{})),
	newTarget("waas", "WaasPolicy", reflect.TypeOf(waas.WaasPolicy{})),
	newTarget("waas", "AddressListSummary", reflect.TypeOf(waas.AddressListSummary{})),
	newTarget("waas", "CertificateSummary", reflect.TypeOf(waas.CertificateSummary{})),
	newTarget("waas", "CustomProtectionRuleSummary", reflect.TypeOf(waas.CustomProtectionRuleSummary{})),
	newTarget("waas", "HttpRedirectSummary", reflect.TypeOf(waas.HttpRedirectSummary{})),
	newTarget("waas", "WaasPolicySummary", reflect.TypeOf(waas.WaasPolicySummary{})),

	// Waf CRD support
	newTarget("waf", "CreateWebAppFirewallPolicyDetails", reflect.TypeOf(waf.CreateWebAppFirewallPolicyDetails{})),
	newTarget("waf", "UpdateWebAppFirewallDetails", reflect.TypeOf(waf.UpdateWebAppFirewallDetails{})),
	newTarget("waf", "UpdateWebAppFirewallPolicyDetails", reflect.TypeOf(waf.UpdateWebAppFirewallPolicyDetails{})),
	newTarget("waf", "NetworkAddressListCollection", reflect.TypeOf(waf.NetworkAddressListCollection{})),
	newTarget("waf", "WebAppFirewallCollection", reflect.TypeOf(waf.WebAppFirewallCollection{})),
	newTarget("waf", "WebAppFirewallPolicy", reflect.TypeOf(waf.WebAppFirewallPolicy{})),
	newTarget("waf", "WebAppFirewallPolicyCollection", reflect.TypeOf(waf.WebAppFirewallPolicyCollection{})),
	newTarget("waf", "WebAppFirewallPolicySummary", reflect.TypeOf(waf.WebAppFirewallPolicySummary{})),

	// Wlms CRD support
	newTarget("wlms", "UpdateManagedInstanceDetails", reflect.TypeOf(wlms.UpdateManagedInstanceDetails{})),
	newTarget("wlms", "UpdateWlsDomainDetails", reflect.TypeOf(wlms.UpdateWlsDomainDetails{})),
	newTarget("wlms", "ManagedInstance", reflect.TypeOf(wlms.ManagedInstance{})),
	newTarget("wlms", "ManagedInstanceCollection", reflect.TypeOf(wlms.ManagedInstanceCollection{})),
	newTarget("wlms", "WlsDomain", reflect.TypeOf(wlms.WlsDomain{})),
	newTarget("wlms", "WlsDomainCollection", reflect.TypeOf(wlms.WlsDomainCollection{})),
	newTarget("wlms", "ManagedInstanceSummary", reflect.TypeOf(wlms.ManagedInstanceSummary{})),
	newTarget("wlms", "WlsDomainSummary", reflect.TypeOf(wlms.WlsDomainSummary{})),

	// Zpr CRD support
	newTarget("zpr", "CreateConfigurationDetails", reflect.TypeOf(zpr.CreateConfigurationDetails{})),
	newTarget("zpr", "CreateZprPolicyDetails", reflect.TypeOf(zpr.CreateZprPolicyDetails{})),
	newTarget("zpr", "UpdateZprPolicyDetails", reflect.TypeOf(zpr.UpdateZprPolicyDetails{})),
	newTarget("zpr", "Configuration", reflect.TypeOf(zpr.Configuration{})),
	newTarget("zpr", "ZprPolicy", reflect.TypeOf(zpr.ZprPolicy{})),
	newTarget("zpr", "ZprPolicyCollection", reflect.TypeOf(zpr.ZprPolicyCollection{})),
	newTarget("zpr", "ZprPolicySummary", reflect.TypeOf(zpr.ZprPolicySummary{})),
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
