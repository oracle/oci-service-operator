package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildSDKMappingsAppliesLegacyStatusOverrides(t *testing.T) {
	t.Parallel()

	got := buildSDKMappings("core", "AllDrgAttachment", []string{"DrgAttachmentInfo"}, false, specTarget{})
	if len(got) != 1 {
		t.Fatalf("len(buildSDKMappings()) = %d, want 1", len(got))
	}
	if got[0].SDKStruct != "core.DrgAttachmentInfo" {
		t.Fatalf("buildSDKMappings()[0].SDKStruct = %q, want %q", got[0].SDKStruct, "core.DrgAttachmentInfo")
	}
	if got[0].APISurface != "status" {
		t.Fatalf("buildSDKMappings()[0].APISurface = %q, want %q", got[0].APISurface, "status")
	}
}

func TestBuildSDKMappingsSupportsExplicitSurfaceAndExclusionOverrides(t *testing.T) {
	t.Parallel()

	coreInstance := buildSDKMappings("core", "Instance", []string{
		"UpdateInstanceDetails",
		"Instance",
		"InstanceSummary",
	}, false, specTarget{})

	coreByStruct := make(map[string]sdkMapping, len(coreInstance))
	for _, mapping := range coreInstance {
		coreByStruct[mapping.SDKStruct] = mapping
	}
	if coreByStruct["core.Instance"].APISurface != "status" {
		t.Fatalf("core.Instance APISurface = %q, want status", coreByStruct["core.Instance"].APISurface)
	}
	if coreByStruct["core.InstanceSummary"].APISurface != "status" {
		t.Fatalf("core.InstanceSummary APISurface = %q, want status", coreByStruct["core.InstanceSummary"].APISurface)
	}
	if coreByStruct["core.UpdateInstanceDetails"].APISurface != "" {
		t.Fatalf("core.UpdateInstanceDetails APISurface = %q, want empty", coreByStruct["core.UpdateInstanceDetails"].APISurface)
	}

	loadBalancerShape := buildSDKMappings("loadbalancer", "Shape", []string{
		"UpdateLoadBalancerShapeDetails",
		"ShapeDetails",
		"LoadBalancerShape",
	}, false, specTarget{})

	shapeByStruct := make(map[string]sdkMapping, len(loadBalancerShape))
	for _, mapping := range loadBalancerShape {
		shapeByStruct[mapping.SDKStruct] = mapping
	}
	if !shapeByStruct["loadbalancer.UpdateLoadBalancerShapeDetails"].Exclude {
		t.Fatal("loadbalancer.UpdateLoadBalancerShapeDetails should be excluded")
	}
	if shapeByStruct["loadbalancer.UpdateLoadBalancerShapeDetails"].Reason == "" {
		t.Fatal("loadbalancer.UpdateLoadBalancerShapeDetails should carry an exclusion reason")
	}
	for _, sdkStruct := range []string{"loadbalancer.ShapeDetails", "loadbalancer.LoadBalancerShape"} {
		mapping := shapeByStruct[sdkStruct]
		if !mapping.Exclude {
			t.Fatalf("%s Exclude = false, want true", sdkStruct)
		}
		if mapping.Reason == "" {
			t.Fatalf("%s Reason = %q, want non-empty exclusion reason", sdkStruct, mapping.Reason)
		}
	}

	loadBalancerPolicy := buildSDKMappings("loadbalancer", "Policy", []string{
		"LoadBalancerPolicy",
	}, false, specTarget{})

	policyByStruct := make(map[string]sdkMapping, len(loadBalancerPolicy))
	for _, mapping := range loadBalancerPolicy {
		policyByStruct[mapping.SDKStruct] = mapping
	}
	if !policyByStruct["loadbalancer.LoadBalancerPolicy"].Exclude {
		t.Fatal("loadbalancer.LoadBalancerPolicy should be excluded")
	}
	if policyByStruct["loadbalancer.LoadBalancerPolicy"].Reason == "" {
		t.Fatal("loadbalancer.LoadBalancerPolicy exclusion should carry a reason")
	}

	loadBalancerProtocol := buildSDKMappings("loadbalancer", "Protocol", []string{
		"LoadBalancerProtocol",
	}, false, specTarget{})

	protocolByStruct := make(map[string]sdkMapping, len(loadBalancerProtocol))
	for _, mapping := range loadBalancerProtocol {
		protocolByStruct[mapping.SDKStruct] = mapping
	}
	if !protocolByStruct["loadbalancer.LoadBalancerProtocol"].Exclude {
		t.Fatal("loadbalancer.LoadBalancerProtocol should be excluded")
	}
	if protocolByStruct["loadbalancer.LoadBalancerProtocol"].Reason == "" {
		t.Fatal("loadbalancer.LoadBalancerProtocol exclusion should carry a reason")
	}

	containerEngineCluster := buildSDKMappings("containerengine", "Cluster", []string{
		"CreateClusterDetails",
		"Cluster",
		"ClusterSummary",
		"UpdateClusterDetails",
	}, false, specTarget{})

	clusterByStruct := make(map[string]sdkMapping, len(containerEngineCluster))
	for _, mapping := range containerEngineCluster {
		clusterByStruct[mapping.SDKStruct] = mapping
	}
	if clusterByStruct["containerengine.Cluster"].APISurface != "status" {
		t.Fatalf("containerengine.Cluster APISurface = %q, want status", clusterByStruct["containerengine.Cluster"].APISurface)
	}
	if clusterByStruct["containerengine.ClusterSummary"].APISurface != "status" {
		t.Fatalf("containerengine.ClusterSummary APISurface = %q, want status", clusterByStruct["containerengine.ClusterSummary"].APISurface)
	}
	if clusterByStruct["containerengine.CreateClusterDetails"].APISurface != "" {
		t.Fatalf("containerengine.CreateClusterDetails APISurface = %q, want empty", clusterByStruct["containerengine.CreateClusterDetails"].APISurface)
	}

	monitoringMetric := buildSDKMappings("monitoring", "Metric", []string{
		"Metric",
	}, false, specTarget{})

	metricByStruct := make(map[string]sdkMapping, len(monitoringMetric))
	for _, mapping := range monitoringMetric {
		metricByStruct[mapping.SDKStruct] = mapping
	}
	if metricByStruct["monitoring.Metric"].APISurface != "status" {
		t.Fatalf("monitoring.Metric APISurface = %q, want status", metricByStruct["monitoring.Metric"].APISurface)
	}

	networkLoadBalancerBackend := buildSDKMappings("networkloadbalancer", "Backend", []string{
		"CreateBackendDetails",
		"Backend",
		"BackendSummary",
		"UpdateBackendDetails",
	}, false, specTarget{})

	nlbByStruct := make(map[string]sdkMapping, len(networkLoadBalancerBackend))
	for _, mapping := range networkLoadBalancerBackend {
		nlbByStruct[mapping.SDKStruct] = mapping
	}
	if nlbByStruct["networkloadbalancer.Backend"].APISurface != "spec" {
		t.Fatalf("networkloadbalancer.Backend APISurface = %q, want spec", nlbByStruct["networkloadbalancer.Backend"].APISurface)
	}
	if nlbByStruct["networkloadbalancer.BackendSummary"].APISurface != "spec" {
		t.Fatalf("networkloadbalancer.BackendSummary APISurface = %q, want spec", nlbByStruct["networkloadbalancer.BackendSummary"].APISurface)
	}
	if nlbByStruct["networkloadbalancer.CreateBackendDetails"].APISurface != "" {
		t.Fatalf("networkloadbalancer.CreateBackendDetails APISurface = %q, want empty", nlbByStruct["networkloadbalancer.CreateBackendDetails"].APISurface)
	}

	topic := buildSDKMappings("ons", "Topic", []string{
		"CreateTopicDetails",
		"NotificationTopic",
		"NotificationTopicSummary",
		"UpdateTopicDetails",
	}, false, specTarget{})

	topicByStruct := make(map[string]sdkMapping, len(topic))
	for _, mapping := range topic {
		topicByStruct[mapping.SDKStruct] = mapping
	}
	for _, sdkStruct := range []string{"ons.NotificationTopic", "ons.NotificationTopicSummary"} {
		mapping := topicByStruct[sdkStruct]
		if !mapping.Exclude {
			t.Fatalf("%s Exclude = false, want true", sdkStruct)
		}
		if mapping.Reason == "" {
			t.Fatalf("%s Reason = %q, want non-empty exclusion reason", sdkStruct, mapping.Reason)
		}
	}
	if topicByStruct["ons.CreateTopicDetails"].Exclude {
		t.Fatal("ons.CreateTopicDetails should remain included")
	}

	oauthClientCredential := buildSDKMappings("identity", "OAuthClientCredential", []string{
		"CreateOAuth2ClientCredentialDetails",
		"OAuth2ClientCredential",
		"OAuth2ClientCredentialSummary",
		"UpdateOAuth2ClientCredentialDetails",
	}, false, specTarget{})

	oauthByStruct := make(map[string]sdkMapping, len(oauthClientCredential))
	for _, mapping := range oauthClientCredential {
		oauthByStruct[mapping.SDKStruct] = mapping
	}
	for _, sdkStruct := range []string{"identity.OAuth2ClientCredential", "identity.OAuth2ClientCredentialSummary"} {
		mapping := oauthByStruct[sdkStruct]
		if !mapping.Exclude {
			t.Fatalf("%s Exclude = false, want true", sdkStruct)
		}
		if mapping.Reason == "" {
			t.Fatalf("%s Reason = %q, want non-empty exclusion reason", sdkStruct, mapping.Reason)
		}
	}
	if oauthByStruct["identity.CreateOAuth2ClientCredentialDetails"].Exclude {
		t.Fatal("identity.CreateOAuth2ClientCredentialDetails should remain included")
	}

	containerImage := buildSDKMappings("artifacts", "ContainerImage", []string{
		"ContainerImage",
		"ContainerImageCollection",
		"ContainerImageSummary",
		"UpdateContainerImageDetails",
	}, false, specTarget{})

	containerImageByStruct := make(map[string]sdkMapping, len(containerImage))
	for _, mapping := range containerImage {
		containerImageByStruct[mapping.SDKStruct] = mapping
	}
	if !containerImageByStruct["artifacts.ContainerImageCollection"].Exclude {
		t.Fatal("artifacts.ContainerImageCollection should be excluded")
	}
	if containerImageByStruct["artifacts.ContainerImage"].Exclude {
		t.Fatal("artifacts.ContainerImage should remain included")
	}

	repository := buildSDKMappings("artifacts", "Repository", []string{
		"ContainerRepository",
		"GenericRepository",
		"RepositoryCollection",
	}, false, specTarget{})

	repositoryByStruct := make(map[string]sdkMapping, len(repository))
	for _, mapping := range repository {
		repositoryByStruct[mapping.SDKStruct] = mapping
	}
	if repositoryByStruct["artifacts.GenericRepository"].APISurface != "status" {
		t.Fatalf("artifacts.GenericRepository APISurface = %q, want status", repositoryByStruct["artifacts.GenericRepository"].APISurface)
	}
	if !repositoryByStruct["artifacts.ContainerRepository"].Exclude {
		t.Fatal("artifacts.ContainerRepository should be excluded for Repository")
	}
	if repositoryByStruct["artifacts.ContainerRepository"].Reason == "" {
		t.Fatal("artifacts.ContainerRepository exclusion should carry a reason")
	}
	if !repositoryByStruct["artifacts.RepositoryCollection"].Exclude {
		t.Fatal("artifacts.RepositoryCollection should be excluded")
	}
}

func TestBuildSDKMappingsAppliesDatabaseMySQLAndStreamingOverrides(t *testing.T) {
	t.Parallel()

	type expectation struct {
		exclude bool
	}

	tests := []struct {
		name       string
		service    string
		spec       string
		candidates []string
		want       map[string]expectation
	}{
		{
			name:       "database autonomous container database",
			service:    "database",
			spec:       "AutonomousContainerDatabase",
			candidates: []string{"CreateAutonomousContainerDatabaseDetails", "UpdateAutonomousContainerDatabaseDetails", "AutonomousContainerDatabase", "AutonomousContainerDatabaseSummary", "AutonomousContainerDatabaseVersionSummary"},
			want: map[string]expectation{
				"database.UpdateAutonomousContainerDatabaseDetails":  {exclude: true},
				"database.AutonomousContainerDatabaseVersionSummary": {exclude: true},
			},
		},
		{
			name:       "database autonomous database regional wallet",
			service:    "database",
			spec:       "AutonomousDatabaseRegionalWallet",
			candidates: []string{"UpdateAutonomousDatabaseWalletDetails", "AutonomousDatabaseWallet"},
			want: map[string]expectation{
				"database.AutonomousDatabaseWallet": {exclude: true},
			},
		},
		{
			name:       "database backup destination",
			service:    "database",
			spec:       "BackupDestination",
			candidates: []string{"UpdateBackupDestinationDetails", "BackupDestinationDetails", "BackupDestination", "BackupDestinationSummary"},
			want: map[string]expectation{
				"database.CreateNfsBackupDestinationDetails":               {},
				"database.CreateRecoveryApplianceBackupDestinationDetails": {},
				"database.UpdateBackupDestinationDetails":                  {exclude: true},
				"database.BackupDestinationDetails":                        {exclude: true},
			},
		},
		{
			name:       "database cloud vm cluster",
			service:    "database",
			spec:       "CloudVmCluster",
			candidates: []string{"CreateCloudVmClusterDetails", "UpdateCloudVmClusterDetails", "CloudVmCluster", "CloudVmClusterSummary"},
			want: map[string]expectation{
				"database.UpdateCloudVmClusterDetails": {exclude: true},
			},
		},
		{
			name:       "database cloud vm cluster iorm config",
			service:    "database",
			spec:       "CloudVmClusterIormConfig",
			candidates: []string{"ExadataIormConfigUpdateDetails", "ExadataIormConfig"},
			want: map[string]expectation{
				"database.ExadataIormConfig": {exclude: true},
			},
		},
		{
			name:       "database console history",
			service:    "database",
			spec:       "ConsoleHistory",
			candidates: []string{"CreateConsoleHistoryDetails", "UpdateConsoleHistoryDetails", "ConsoleHistory", "ConsoleHistoryCollection", "ConsoleHistorySummary"},
			want: map[string]expectation{
				"database.ConsoleHistoryCollection": {exclude: true},
			},
		},
		{
			name:       "database data guard association",
			service:    "database",
			spec:       "DataGuardAssociation",
			candidates: []string{"UpdateDataGuardAssociationDetails", "DataGuardAssociation", "DataGuardAssociationSummary"},
			want: map[string]expectation{
				"database.CreateDataGuardAssociationToExistingDbSystemDetails":  {},
				"database.CreateDataGuardAssociationToExistingVmClusterDetails": {},
				"database.CreateDataGuardAssociationWithNewDbSystemDetails":     {},
				"database.UpdateDataGuardAssociationDetails":                    {exclude: true},
			},
		},
		{
			name:       "database db server",
			service:    "database",
			spec:       "DbServer",
			candidates: []string{"DbServerDetails", "DbServer", "DbServerSummary"},
			want: map[string]expectation{
				"database.DbServerDetails": {exclude: true},
			},
		},
		{
			name:       "database flex component",
			service:    "database",
			spec:       "FlexComponent",
			candidates: []string{"FlexComponentCollection", "FlexComponentSummary"},
			want: map[string]expectation{
				"database.FlexComponentCollection": {exclude: true},
			},
		},
		{
			name:       "database system version",
			service:    "database",
			spec:       "SystemVersion",
			candidates: []string{"SystemVersionCollection", "SystemVersionSummary"},
			want: map[string]expectation{
				"database.SystemVersionCollection": {exclude: true},
			},
		},
		{
			name:       "database vm cluster update",
			service:    "database",
			spec:       "VmClusterUpdate",
			candidates: []string{"VmClusterUpdateDetails", "VmClusterUpdate", "VmClusterUpdateSummary"},
			want: map[string]expectation{
				"database.VmClusterUpdateDetails": {exclude: true},
			},
		},
		{
			name:       "mysql work request log",
			service:    "mysql",
			spec:       "WorkRequestLog",
			candidates: []string{"WorkRequestLogEntry"},
			want: map[string]expectation{
				"mysql.WorkRequestLogEntry": {exclude: true},
			},
		},
		{
			name:       "streaming connect harness",
			service:    "streaming",
			spec:       "ConnectHarness",
			candidates: []string{"CreateConnectHarnessDetails", "UpdateConnectHarnessDetails", "ConnectHarness", "ConnectHarnessSummary"},
			want: map[string]expectation{
				"streaming.UpdateConnectHarnessDetails": {exclude: true},
			},
		},
		{
			name:       "streaming stream pool",
			service:    "streaming",
			spec:       "StreamPool",
			candidates: []string{"CreateStreamPoolDetails", "UpdateStreamPoolDetails", "StreamPool", "StreamPoolSummary"},
			want: map[string]expectation{
				"streaming.UpdateStreamPoolDetails": {exclude: true},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildSDKMappings(tt.service, tt.spec, tt.candidates, false, specTarget{})
			byStruct := make(map[string]sdkMapping, len(got))
			for _, mapping := range got {
				byStruct[mapping.SDKStruct] = mapping
			}

			for sdkStruct, want := range tt.want {
				mapping, ok := byStruct[sdkStruct]
				if !ok {
					t.Fatalf("missing mapping for %s", sdkStruct)
				}
				if mapping.Exclude != want.exclude {
					t.Fatalf("%s Exclude = %t, want %t", sdkStruct, mapping.Exclude, want.exclude)
				}
				if want.exclude && mapping.Reason == "" {
					t.Fatalf("%s exclusion should carry a reason", sdkStruct)
				}
			}
		})
	}
}

func TestBuildSDKMappingsAppliesPSQLStatusAndWrapperOverrides(t *testing.T) {
	t.Parallel()

	type expectation struct {
		apiSurface string
		exclude    bool
	}

	tests := []struct {
		name       string
		spec       string
		candidates []string
		want       map[string]expectation
	}{
		{
			name:       "backup",
			spec:       "Backup",
			candidates: []string{"CreateBackupDetails", "UpdateBackupDetails", "Backup", "BackupCollection", "BackupSummary"},
			want: map[string]expectation{
				"psql.Backup":           {apiSurface: "status"},
				"psql.BackupCollection": {exclude: true},
				"psql.BackupSummary":    {apiSurface: "status"},
			},
		},
		{
			name:       "configuration",
			spec:       "Configuration",
			candidates: []string{"CreateConfigurationDetails", "UpdateConfigurationDetails", "Configuration", "ConfigurationCollection", "ConfigurationDetails", "ConfigurationSummary"},
			want: map[string]expectation{
				"psql.Configuration":           {apiSurface: "status"},
				"psql.ConfigurationCollection": {exclude: true},
				"psql.ConfigurationDetails":    {exclude: true},
				"psql.ConfigurationSummary":    {apiSurface: "status"},
			},
		},
		{
			name:       "db system",
			spec:       "DbSystem",
			candidates: []string{"CreateDbSystemDetails", "UpdateDbSystemDetails", "DbSystem", "DbSystemCollection", "DbSystemSummary"},
			want: map[string]expectation{
				"psql.DbSystem":           {apiSurface: "status"},
				"psql.DbSystemCollection": {exclude: true},
				"psql.DbSystemSummary":    {apiSurface: "status"},
			},
		},
		{
			name:       "default configuration",
			spec:       "DefaultConfiguration",
			candidates: []string{"DefaultConfiguration", "DefaultConfigurationCollection", "DefaultConfigurationDetails", "DefaultConfigurationSummary"},
			want: map[string]expectation{
				"psql.DefaultConfiguration":           {apiSurface: "status"},
				"psql.DefaultConfigurationCollection": {exclude: true},
				"psql.DefaultConfigurationDetails":    {exclude: true},
				"psql.DefaultConfigurationSummary":    {apiSurface: "status"},
			},
		},
		{
			name:       "primary db instance",
			spec:       "PrimaryDbInstance",
			candidates: []string{"PrimaryDbInstanceDetails"},
			want: map[string]expectation{
				"psql.PrimaryDbInstanceDetails": {apiSurface: "status"},
			},
		},
		{
			name:       "shape",
			spec:       "Shape",
			candidates: []string{"ShapeCollection", "ShapeSummary"},
			want: map[string]expectation{
				"psql.ShapeCollection": {exclude: true},
				"psql.ShapeSummary":    {apiSurface: "status"},
			},
		},
		{
			name:       "work request",
			spec:       "WorkRequest",
			candidates: []string{"WorkRequest", "WorkRequestSummary"},
			want: map[string]expectation{
				"psql.WorkRequest":        {apiSurface: "status"},
				"psql.WorkRequestSummary": {apiSurface: "status"},
			},
		},
		{
			name:       "work request error",
			spec:       "WorkRequestError",
			candidates: []string{"WorkRequestError", "WorkRequestErrorCollection"},
			want: map[string]expectation{
				"psql.WorkRequestError":           {apiSurface: "status"},
				"psql.WorkRequestErrorCollection": {exclude: true},
			},
		},
		{
			name:       "work request log",
			spec:       "WorkRequestLog",
			candidates: []string{"WorkRequestLogEntry", "WorkRequestLogEntryCollection"},
			want: map[string]expectation{
				"psql.WorkRequestLogEntry":           {apiSurface: "status"},
				"psql.WorkRequestLogEntryCollection": {exclude: true},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildSDKMappings("psql", tt.spec, tt.candidates, false, specTarget{})
			byStruct := make(map[string]sdkMapping, len(got))
			for _, mapping := range got {
				byStruct[mapping.SDKStruct] = mapping
			}

			for sdkStruct, want := range tt.want {
				mapping, ok := byStruct[sdkStruct]
				if !ok {
					t.Fatalf("missing mapping for %s", sdkStruct)
				}
				if mapping.APISurface != want.apiSurface {
					t.Fatalf("%s APISurface = %q, want %q", sdkStruct, mapping.APISurface, want.apiSurface)
				}
				if mapping.Exclude != want.exclude {
					t.Fatalf("%s Exclude = %t, want %t", sdkStruct, mapping.Exclude, want.exclude)
				}
				if want.exclude && mapping.Reason == "" {
					t.Fatalf("%s exclusion should carry a reason", sdkStruct)
				}
			}
		})
	}
}

func TestBuildSDKMappingsAppliesNoSQLStatusAndWrapperOverrides(t *testing.T) {
	t.Parallel()

	type expectation struct {
		apiSurface string
		exclude    bool
	}

	tests := []struct {
		name       string
		spec       string
		candidates []string
		want       map[string]expectation
	}{
		{
			name:       "index",
			spec:       "Index",
			candidates: []string{"Index", "IndexCollection", "IndexSummary"},
			want: map[string]expectation{
				"nosql.Index":           {apiSurface: "status"},
				"nosql.IndexCollection": {exclude: true},
				"nosql.IndexSummary":    {apiSurface: "status"},
			},
		},
		{
			name:       "table",
			spec:       "Table",
			candidates: []string{"CreateTableDetails", "UpdateTableDetails", "Table", "TableCollection", "TableSummary"},
			want: map[string]expectation{
				"nosql.Table":           {apiSurface: "status"},
				"nosql.TableCollection": {exclude: true},
				"nosql.TableSummary":    {apiSurface: "status"},
			},
		},
		{
			name:       "table usage",
			spec:       "TableUsage",
			candidates: []string{"TableUsageCollection", "TableUsageSummary"},
			want: map[string]expectation{
				"nosql.TableUsageCollection": {exclude: true},
				"nosql.TableUsageSummary":    {apiSurface: "status"},
			},
		},
		{
			name:       "work request",
			spec:       "WorkRequest",
			candidates: []string{"WorkRequest", "WorkRequestCollection", "WorkRequestSummary"},
			want: map[string]expectation{
				"nosql.WorkRequest":           {apiSurface: "status"},
				"nosql.WorkRequestCollection": {exclude: true},
				"nosql.WorkRequestSummary":    {apiSurface: "status"},
			},
		},
		{
			name:       "work request error",
			spec:       "WorkRequestError",
			candidates: []string{"WorkRequestError", "WorkRequestErrorCollection"},
			want: map[string]expectation{
				"nosql.WorkRequestError":           {apiSurface: "status"},
				"nosql.WorkRequestErrorCollection": {exclude: true},
			},
		},
		{
			name:       "work request log",
			spec:       "WorkRequestLog",
			candidates: []string{"WorkRequestLogEntry", "WorkRequestLogEntryCollection"},
			want: map[string]expectation{
				"nosql.WorkRequestLogEntry":           {apiSurface: "status"},
				"nosql.WorkRequestLogEntryCollection": {exclude: true},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildSDKMappings("nosql", tt.spec, tt.candidates, false, specTarget{})
			byStruct := make(map[string]sdkMapping, len(got))
			for _, mapping := range got {
				byStruct[mapping.SDKStruct] = mapping
			}

			for sdkStruct, want := range tt.want {
				mapping, ok := byStruct[sdkStruct]
				if !ok {
					t.Fatalf("missing mapping for %s", sdkStruct)
				}
				if mapping.APISurface != want.apiSurface {
					t.Fatalf("%s APISurface = %q, want %q", sdkStruct, mapping.APISurface, want.apiSurface)
				}
				if mapping.Exclude != want.exclude {
					t.Fatalf("%s Exclude = %t, want %t", sdkStruct, mapping.Exclude, want.exclude)
				}
				if want.exclude && mapping.Reason == "" {
					t.Fatalf("%s exclusion should carry a reason", sdkStruct)
				}
			}
		})
	}
}

func TestBuildSDKMappingsAppliesQueueStatusAndWrapperOverrides(t *testing.T) {
	t.Parallel()

	type expectation struct {
		apiSurface string
		exclude    bool
	}

	tests := []struct {
		name       string
		spec       string
		candidates []string
		want       map[string]expectation
	}{
		{
			name:       "channel",
			spec:       "Channel",
			candidates: []string{"ChannelCollection"},
			want: map[string]expectation{
				"queue.ChannelCollection": {exclude: true},
			},
		},
		{
			name:       "queue",
			spec:       "Queue",
			candidates: []string{"CreateQueueDetails", "UpdateQueueDetails", "Queue", "QueueCollection", "QueueSummary"},
			want: map[string]expectation{
				"queue.Queue":           {apiSurface: "status"},
				"queue.QueueCollection": {exclude: true},
				"queue.QueueSummary":    {apiSurface: "status"},
			},
		},
		{
			name:       "work request error",
			spec:       "WorkRequestError",
			candidates: []string{"WorkRequestError", "WorkRequestErrorCollection"},
			want: map[string]expectation{
				"queue.WorkRequestError":           {apiSurface: "status"},
				"queue.WorkRequestErrorCollection": {exclude: true},
			},
		},
		{
			name:       "work request log",
			spec:       "WorkRequestLog",
			candidates: []string{"WorkRequestLogEntry", "WorkRequestLogEntryCollection"},
			want: map[string]expectation{
				"queue.WorkRequestLogEntry":           {apiSurface: "status"},
				"queue.WorkRequestLogEntryCollection": {exclude: true},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildSDKMappings("queue", tt.spec, tt.candidates, false, specTarget{})
			byStruct := make(map[string]sdkMapping, len(got))
			for _, mapping := range got {
				byStruct[mapping.SDKStruct] = mapping
			}

			for sdkStruct, want := range tt.want {
				mapping, ok := byStruct[sdkStruct]
				if !ok {
					t.Fatalf("missing mapping for %s", sdkStruct)
				}
				if mapping.APISurface != want.apiSurface {
					t.Fatalf("%s APISurface = %q, want %q", sdkStruct, mapping.APISurface, want.apiSurface)
				}
				if mapping.Exclude != want.exclude {
					t.Fatalf("%s Exclude = %t, want %t", sdkStruct, mapping.Exclude, want.exclude)
				}
				if want.exclude && mapping.Reason == "" {
					t.Fatalf("%s exclusion should carry a reason", sdkStruct)
				}
			}
		})
	}
}

func TestBuildSDKMappingsAppliesMonitoringStatusOverrides(t *testing.T) {
	t.Parallel()

	alarmHistory := buildSDKMappings("monitoring", "AlarmHistory", []string{
		"AlarmHistoryCollection",
		"AlarmHistoryEntry",
	}, false, specTarget{})

	alarmHistoryByStruct := make(map[string]sdkMapping, len(alarmHistory))
	for _, mapping := range alarmHistory {
		alarmHistoryByStruct[mapping.SDKStruct] = mapping
	}
	if alarmHistoryByStruct["monitoring.AlarmHistoryCollection"].APISurface != "status" {
		t.Fatalf("monitoring.AlarmHistoryCollection APISurface = %q, want status", alarmHistoryByStruct["monitoring.AlarmHistoryCollection"].APISurface)
	}
	if !alarmHistoryByStruct["monitoring.AlarmHistoryEntry"].Exclude {
		t.Fatal("monitoring.AlarmHistoryEntry should be excluded")
	}
	if alarmHistoryByStruct["monitoring.AlarmHistoryEntry"].Reason == "" {
		t.Fatal("monitoring.AlarmHistoryEntry exclusion should carry a reason")
	}

	alarmSuppression := buildSDKMappings("monitoring", "AlarmSuppression", []string{
		"CreateAlarmSuppressionDetails",
		"AlarmSuppression",
		"AlarmSuppressionCollection",
		"AlarmSuppressionSummary",
	}, false, specTarget{})

	alarmSuppressionByStruct := make(map[string]sdkMapping, len(alarmSuppression))
	for _, mapping := range alarmSuppression {
		alarmSuppressionByStruct[mapping.SDKStruct] = mapping
	}
	if alarmSuppressionByStruct["monitoring.AlarmSuppression"].APISurface != "status" {
		t.Fatalf("monitoring.AlarmSuppression APISurface = %q, want status", alarmSuppressionByStruct["monitoring.AlarmSuppression"].APISurface)
	}
	if alarmSuppressionByStruct["monitoring.AlarmSuppressionSummary"].APISurface != "status" {
		t.Fatalf("monitoring.AlarmSuppressionSummary APISurface = %q, want status", alarmSuppressionByStruct["monitoring.AlarmSuppressionSummary"].APISurface)
	}
	if !alarmSuppressionByStruct["monitoring.AlarmSuppressionCollection"].Exclude {
		t.Fatal("monitoring.AlarmSuppressionCollection should be excluded")
	}
	if alarmSuppressionByStruct["monitoring.AlarmSuppressionCollection"].Reason == "" {
		t.Fatal("monitoring.AlarmSuppressionCollection exclusion should carry a reason")
	}
	if alarmSuppressionByStruct["monitoring.CreateAlarmSuppressionDetails"].Exclude {
		t.Fatal("monitoring.CreateAlarmSuppressionDetails should remain included")
	}
}

func TestBuildSDKMappingsAppliesVaultStatusOverrides(t *testing.T) {
	t.Parallel()

	secret := buildSDKMappings("vault", "Secret", []string{
		"CreateSecretDetails",
		"Secret",
		"SecretSummary",
		"SecretVersionSummary",
		"UpdateSecretDetails",
	}, false, specTarget{})

	secretByStruct := make(map[string]sdkMapping, len(secret))
	for _, mapping := range secret {
		secretByStruct[mapping.SDKStruct] = mapping
	}
	if secretByStruct["vault.Secret"].APISurface != "status" {
		t.Fatalf("vault.Secret APISurface = %q, want status", secretByStruct["vault.Secret"].APISurface)
	}
	if secretByStruct["vault.SecretSummary"].APISurface != "status" {
		t.Fatalf("vault.SecretSummary APISurface = %q, want status", secretByStruct["vault.SecretSummary"].APISurface)
	}
	if !secretByStruct["vault.SecretVersionSummary"].Exclude {
		t.Fatal("vault.SecretVersionSummary should be excluded")
	}
	if secretByStruct["vault.SecretVersionSummary"].Reason == "" {
		t.Fatal("vault.SecretVersionSummary exclusion should carry a reason")
	}
	if secretByStruct["vault.CreateSecretDetails"].Exclude {
		t.Fatal("vault.CreateSecretDetails should remain included")
	}
}

func TestParseExistingAPITargetsPreservesSDKMappingMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	registryPath := filepath.Join(dir, "registry.go")
	source := `package apispec

import (
	"reflect"

	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
)

var targets = []Target{
	{
		Name:       "CoreInstance",
		SpecType:   reflect.TypeOf(corev1beta1.InstanceSpec{}),
		StatusType: reflect.TypeOf(corev1beta1.InstanceStatus{}),
		SDKMappings: []SDKMapping{
			{
				SDKStruct:  "core.Instance",
				APISurface: "status",
			},
			{
				SDKStruct: "core.InstanceSummary",
				Exclude:   true,
				Reason:    "Intentionally untracked: summary mapping excluded from desired-state coverage.",
			},
		},
	},
}
`
	if err := os.WriteFile(registryPath, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := parseExistingAPITargets(registryPath)
	if err != nil {
		t.Fatalf("parseExistingAPITargets() error = %v", err)
	}

	target, ok := got["core.Instance"]
	if !ok {
		t.Fatalf("parsed targets missing %q", "core.Instance")
	}
	if len(target.SDKMappings) != 2 {
		t.Fatalf("len(target.SDKMappings) = %d, want 2", len(target.SDKMappings))
	}
	if target.SDKMappings[0].APISurface != "status" {
		t.Fatalf("target.SDKMappings[0].APISurface = %q, want status", target.SDKMappings[0].APISurface)
	}
	if !target.SDKMappings[1].Exclude {
		t.Fatal("target.SDKMappings[1].Exclude = false, want true")
	}
}
