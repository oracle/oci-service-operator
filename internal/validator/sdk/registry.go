package sdk

import (
	"path"
	"reflect"

	"github.com/oracle/oci-go-sdk/v65/accessgovernancecp"
	"github.com/oracle/oci-go-sdk/v65/adm"
	"github.com/oracle/oci-go-sdk/v65/aidocument"
	"github.com/oracle/oci-go-sdk/v65/ailanguage"
	"github.com/oracle/oci-go-sdk/v65/aispeech"
	"github.com/oracle/oci-go-sdk/v65/aivision"
	"github.com/oracle/oci-go-sdk/v65/analytics"
	"github.com/oracle/oci-go-sdk/v65/apiaccesscontrol"
	"github.com/oracle/oci-go-sdk/v65/apiplatform"
	"github.com/oracle/oci-go-sdk/v65/apmconfig"
	"github.com/oracle/oci-go-sdk/v65/apmcontrolplane"
	"github.com/oracle/oci-go-sdk/v65/apmsynthetics"
	"github.com/oracle/oci-go-sdk/v65/apmtraces"
	"github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/bastion"
	"github.com/oracle/oci-go-sdk/v65/bds"
	"github.com/oracle/oci-go-sdk/v65/budget"
	"github.com/oracle/oci-go-sdk/v65/capacitymanagement"
	"github.com/oracle/oci-go-sdk/v65/certificatesmanagement"
	"github.com/oracle/oci-go-sdk/v65/clusterplacementgroups"
	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/oracle/oci-go-sdk/v65/containerinstances"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/dashboardservice"
	"github.com/oracle/oci-go-sdk/v65/database"
	"github.com/oracle/oci-go-sdk/v65/databasemigration"
	"github.com/oracle/oci-go-sdk/v65/databasetools"
	"github.com/oracle/oci-go-sdk/v65/dataflow"
	"github.com/oracle/oci-go-sdk/v65/datalabelingservice"
	"github.com/oracle/oci-go-sdk/v65/datascience"
	"github.com/oracle/oci-go-sdk/v65/delegateaccesscontrol"
	"github.com/oracle/oci-go-sdk/v65/devops"
	"github.com/oracle/oci-go-sdk/v65/dns"
	"github.com/oracle/oci-go-sdk/v65/email"
	"github.com/oracle/oci-go-sdk/v65/events"
	"github.com/oracle/oci-go-sdk/v65/functions"
	"github.com/oracle/oci-go-sdk/v65/generativeai"
	"github.com/oracle/oci-go-sdk/v65/governancerulescontrolplane"
	"github.com/oracle/oci-go-sdk/v65/healthchecks"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/oracle/oci-go-sdk/v65/integration"
	"github.com/oracle/oci-go-sdk/v65/iot"
	"github.com/oracle/oci-go-sdk/v65/keymanagement"
	"github.com/oracle/oci-go-sdk/v65/licensemanager"
	"github.com/oracle/oci-go-sdk/v65/limits"
	"github.com/oracle/oci-go-sdk/v65/limitsincrease"
	"github.com/oracle/oci-go-sdk/v65/loadbalancer"
	"github.com/oracle/oci-go-sdk/v65/lockbox"
	"github.com/oracle/oci-go-sdk/v65/loganalytics"
	"github.com/oracle/oci-go-sdk/v65/logging"
	"github.com/oracle/oci-go-sdk/v65/managedkafka"
	"github.com/oracle/oci-go-sdk/v65/managementagent"
	"github.com/oracle/oci-go-sdk/v65/managementdashboard"
	"github.com/oracle/oci-go-sdk/v65/marketplace"
	"github.com/oracle/oci-go-sdk/v65/marketplaceprivateoffer"
	"github.com/oracle/oci-go-sdk/v65/marketplacepublisher"
	"github.com/oracle/oci-go-sdk/v65/monitoring"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	"github.com/oracle/oci-go-sdk/v65/networkloadbalancer"
	"github.com/oracle/oci-go-sdk/v65/nosql"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
	"github.com/oracle/oci-go-sdk/v65/ocvp"
	"github.com/oracle/oci-go-sdk/v65/oda"
	"github.com/oracle/oci-go-sdk/v65/ons"
	"github.com/oracle/oci-go-sdk/v65/opensearch"
	"github.com/oracle/oci-go-sdk/v65/operatoraccesscontrol"
	"github.com/oracle/oci-go-sdk/v65/opsi"
	"github.com/oracle/oci-go-sdk/v65/optimizer"
	"github.com/oracle/oci-go-sdk/v65/osmanagementhub"
	"github.com/oracle/oci-go-sdk/v65/psql"
	"github.com/oracle/oci-go-sdk/v65/queue"
	"github.com/oracle/oci-go-sdk/v65/recovery"
	"github.com/oracle/oci-go-sdk/v65/redis"
	"github.com/oracle/oci-go-sdk/v65/resourceanalytics"
	"github.com/oracle/oci-go-sdk/v65/resourcescheduler"
	"github.com/oracle/oci-go-sdk/v65/sch"
	"github.com/oracle/oci-go-sdk/v65/securityattribute"
	"github.com/oracle/oci-go-sdk/v65/servicecatalog"
	"github.com/oracle/oci-go-sdk/v65/stackmonitoring"
	"github.com/oracle/oci-go-sdk/v65/streaming"
	"github.com/oracle/oci-go-sdk/v65/usageapi"
	"github.com/oracle/oci-go-sdk/v65/waa"
	"github.com/oracle/oci-go-sdk/v65/waas"
	"github.com/oracle/oci-go-sdk/v65/waf"
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

	// Bastion CRD support
	newTarget("bastion", "CreateBastionDetails", reflect.TypeOf(bastion.CreateBastionDetails{})),
	newTarget("bastion", "CreateSessionDetails", reflect.TypeOf(bastion.CreateSessionDetails{})),
	newTarget("bastion", "UpdateBastionDetails", reflect.TypeOf(bastion.UpdateBastionDetails{})),
	newTarget("bastion", "UpdateSessionDetails", reflect.TypeOf(bastion.UpdateSessionDetails{})),
	newTarget("bastion", "Bastion", reflect.TypeOf(bastion.Bastion{})),
	newTarget("bastion", "Session", reflect.TypeOf(bastion.Session{})),
	newTarget("bastion", "BastionSummary", reflect.TypeOf(bastion.BastionSummary{})),
	newTarget("bastion", "SessionSummary", reflect.TypeOf(bastion.SessionSummary{})),

	// Bds CRD support
	newTarget("bds", "CreateBdsInstanceDetails", reflect.TypeOf(bds.CreateBdsInstanceDetails{})),
	newTarget("bds", "UpdateBdsInstanceDetails", reflect.TypeOf(bds.UpdateBdsInstanceDetails{})),
	newTarget("bds", "BdsInstance", reflect.TypeOf(bds.BdsInstance{})),
	newTarget("bds", "BdsInstanceSummary", reflect.TypeOf(bds.BdsInstanceSummary{})),

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

	// Clusterplacementgroups CRD support
	newTarget("clusterplacementgroups", "CreateClusterPlacementGroupDetails", reflect.TypeOf(clusterplacementgroups.CreateClusterPlacementGroupDetails{})),
	newTarget("clusterplacementgroups", "UpdateClusterPlacementGroupDetails", reflect.TypeOf(clusterplacementgroups.UpdateClusterPlacementGroupDetails{})),
	newTarget("clusterplacementgroups", "ClusterPlacementGroup", reflect.TypeOf(clusterplacementgroups.ClusterPlacementGroup{})),
	newTarget("clusterplacementgroups", "ClusterPlacementGroupCollection", reflect.TypeOf(clusterplacementgroups.ClusterPlacementGroupCollection{})),
	newTarget("clusterplacementgroups", "ClusterPlacementGroupSummary", reflect.TypeOf(clusterplacementgroups.ClusterPlacementGroupSummary{})),

	// Containerinstances CRD support
	newTarget("containerinstances", "CreateContainerInstanceDetails", reflect.TypeOf(containerinstances.CreateContainerInstanceDetails{})),
	newTarget("containerinstances", "UpdateContainerInstanceDetails", reflect.TypeOf(containerinstances.UpdateContainerInstanceDetails{})),
	newTarget("containerinstances", "ContainerInstance", reflect.TypeOf(containerinstances.ContainerInstance{})),
	newTarget("containerinstances", "ContainerInstanceCollection", reflect.TypeOf(containerinstances.ContainerInstanceCollection{})),
	newTarget("containerinstances", "ContainerInstanceSummary", reflect.TypeOf(containerinstances.ContainerInstanceSummary{})),

	// Dashboardservice CRD support
	newTarget("dashboardservice", "CreateDashboardGroupDetails", reflect.TypeOf(dashboardservice.CreateDashboardGroupDetails{})),
	newTarget("dashboardservice", "UpdateDashboardGroupDetails", reflect.TypeOf(dashboardservice.UpdateDashboardGroupDetails{})),
	newTarget("dashboardservice", "DashboardGroup", reflect.TypeOf(dashboardservice.DashboardGroup{})),
	newTarget("dashboardservice", "DashboardGroupCollection", reflect.TypeOf(dashboardservice.DashboardGroupCollection{})),
	newTarget("dashboardservice", "DashboardGroupSummary", reflect.TypeOf(dashboardservice.DashboardGroupSummary{})),

	// Databasemigration CRD support
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

	// Dataflow CRD support
	newTarget("dataflow", "CreateApplicationDetails", reflect.TypeOf(dataflow.CreateApplicationDetails{})),
	newTarget("dataflow", "UpdateApplicationDetails", reflect.TypeOf(dataflow.UpdateApplicationDetails{})),
	newTarget("dataflow", "Application", reflect.TypeOf(dataflow.Application{})),
	newTarget("dataflow", "ApplicationSummary", reflect.TypeOf(dataflow.ApplicationSummary{})),

	// Datalabelingservice CRD support
	newTarget("datalabelingservice", "CreateDatasetDetails", reflect.TypeOf(datalabelingservice.CreateDatasetDetails{})),
	newTarget("datalabelingservice", "UpdateDatasetDetails", reflect.TypeOf(datalabelingservice.UpdateDatasetDetails{})),
	newTarget("datalabelingservice", "Dataset", reflect.TypeOf(datalabelingservice.Dataset{})),
	newTarget("datalabelingservice", "DatasetCollection", reflect.TypeOf(datalabelingservice.DatasetCollection{})),
	newTarget("datalabelingservice", "DatasetSummary", reflect.TypeOf(datalabelingservice.DatasetSummary{})),

	// Datascience CRD support
	newTarget("datascience", "CreateProjectDetails", reflect.TypeOf(datascience.CreateProjectDetails{})),
	newTarget("datascience", "UpdateProjectDetails", reflect.TypeOf(datascience.UpdateProjectDetails{})),
	newTarget("datascience", "Project", reflect.TypeOf(datascience.Project{})),
	newTarget("datascience", "ProjectSummary", reflect.TypeOf(datascience.ProjectSummary{})),

	// Delegateaccesscontrol CRD support
	newTarget("delegateaccesscontrol", "CreateDelegationControlDetails", reflect.TypeOf(delegateaccesscontrol.CreateDelegationControlDetails{})),
	newTarget("delegateaccesscontrol", "UpdateDelegationControlDetails", reflect.TypeOf(delegateaccesscontrol.UpdateDelegationControlDetails{})),
	newTarget("delegateaccesscontrol", "DelegationControl", reflect.TypeOf(delegateaccesscontrol.DelegationControl{})),
	newTarget("delegateaccesscontrol", "DelegationControlSummary", reflect.TypeOf(delegateaccesscontrol.DelegationControlSummary{})),

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

	// Resourcescheduler CRD support
	newTarget("resourcescheduler", "CreateScheduleDetails", reflect.TypeOf(resourcescheduler.CreateScheduleDetails{})),
	newTarget("resourcescheduler", "UpdateScheduleDetails", reflect.TypeOf(resourcescheduler.UpdateScheduleDetails{})),
	newTarget("resourcescheduler", "Schedule", reflect.TypeOf(resourcescheduler.Schedule{})),
	newTarget("resourcescheduler", "ScheduleCollection", reflect.TypeOf(resourcescheduler.ScheduleCollection{})),
	newTarget("resourcescheduler", "ScheduleSummary", reflect.TypeOf(resourcescheduler.ScheduleSummary{})),

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
