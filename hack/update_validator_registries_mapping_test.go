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
	if shapeByStruct["loadbalancer.ShapeDetails"].Exclude {
		t.Fatal("loadbalancer.ShapeDetails should remain included")
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
