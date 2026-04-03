/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/oracle/oci-service-operator/internal/formal"
)

func TestBuildPackageModelDiscoversResources(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "mysql",
		SDKPackage:     "example.com/test/sdk",
		Group:          "mysql",
		PackageProfile: "controller-backed",
	}

	discoverer := &Discoverer{
		resolveDir: func(context.Context, string) (string, error) {
			return sampleSDKDir(t), nil
		},
	}

	pkg, err := discoverer.BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	if pkg.GroupDNSName != "mysql.oracle.com" {
		t.Fatalf("BuildPackageModel() group DNS name = %q, want %q", pkg.GroupDNSName, "mysql.oracle.com")
	}

	assertDiscoveredMySQLDbSystem(t, findResource(t, pkg.Resources, "DbSystem"))
	assertDiscoveredWidget(t, findResource(t, pkg.Resources, "Widget"))
	assertDiscoveredReport(t, findResource(t, pkg.Resources, "Report"))
	assertDiscoveredReportByName(t, findResource(t, pkg.Resources, "ReportByName"))
	assertDiscoveredOAuthClientCredential(t, findResource(t, pkg.Resources, "OAuthClientCredential"))
}

func TestBuildPackageModelAttachesFormalModelFromResourceOverride(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	configPath := filepath.Join(repo, "internal", "generator", "config", "services.yaml")
	writeGeneratorTestFile(t, configPath, `schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  controller-backed:
    description: runtime-integrated groups
services:
  - service: mysql
    sdkPackage: example.com/test/sdk
    group: mysql
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: explicit
      includeKinds:
        - Widget
    generation:
      resources:
        - kind: Widget
          formalSpec: widget
          serviceManager:
            strategy: generated
`)
	writeGeneratorFormalScaffold(t, repo, "mysql", "widget", "Widget")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", configPath, err)
	}
	service := cfg.Services[0]

	discoverer := &Discoverer{
		resolveDir: func(context.Context, string) (string, error) {
			return sampleSDKDir(t), nil
		},
	}

	pkg, err := discoverer.BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	assertWidgetFormalModel(t, findResource(t, pkg.Resources, "Widget"))

	report := findResource(t, pkg.Resources, "Report")
	if report.Formal != nil {
		t.Fatalf("Report formal model = %#v, want nil", report.Formal)
	}

	assertWidgetServiceManagerFormalModel(t, findServiceManagerModel(t, pkg.ServiceManagers, "Widget"))
}

func TestBuildPackageModelDerivesRuntimeSemanticsFromFormalSpec(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	configPath := filepath.Join(repo, "internal", "generator", "config", "services.yaml")
	writeGeneratorTestFile(t, configPath, `schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  controller-backed:
    description: runtime-integrated groups
services:
  - service: mysql
    sdkPackage: example.com/test/sdk
    group: mysql
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: explicit
      includeKinds:
        - Widget
    generation:
      resources:
        - kind: Widget
          formalSpec: widget
          serviceManager:
            strategy: generated
`)
	writeGeneratorFormalScaffold(t, repo, "mysql", "widget", "Widget")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", configPath, err)
	}

	discoverer := &Discoverer{
		resolveDir: func(context.Context, string) (string, error) {
			return sampleSDKDir(t), nil
		},
	}

	pkg, err := discoverer.BuildPackageModel(context.Background(), cfg, cfg.Services[0])
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	assertWidgetRuntimeSemantics(t, findResource(t, pkg.Resources, "Widget"))
	assertWidgetServiceManagerSemantics(t, findServiceManagerModel(t, pkg.ServiceManagers, "Widget"))
}

func TestBuildPackageModelSynthesizesComplexSDKFields(t *testing.T) {
	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}

	tests := []struct {
		name    string
		service ServiceConfig
		assert  func(*testing.T, *PackageModel)
	}{
		{
			name: "functions",
			service: ServiceConfig{
				Service:        "functions",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/functions",
				Group:          "functions",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: assertFunctionsSDKFields,
		},
		{
			name: "core",
			service: ServiceConfig{
				Service:        "core",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/core",
				Group:          "core",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: assertCoreSDKFields,
		},
		{
			name: "certificates",
			service: ServiceConfig{
				Service:        "certificates",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/certificates",
				Group:          "certificates",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: assertCertificatesSDKFields,
		},
		{
			name: "nosql",
			service: ServiceConfig{
				Service:        "nosql",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/nosql",
				Group:          "nosql",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: assertNoSQLSDKFields,
		},
		{
			name: "secrets",
			service: ServiceConfig{
				Service:        "secrets",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/secrets",
				Group:          "secrets",
				PackageProfile: PackageProfileCRDOnly,
				ObservedState: ObservedStateConfig{
					SDKAliases: map[string][]string{
						"SecretBundleByName": {"SecretBundle"},
					},
				},
			},
			assert: assertSecretsSDKFields,
		},
		{
			name: "vault",
			service: ServiceConfig{
				Service:        "vault",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/vault",
				Group:          "vault",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: assertVaultSDKFields,
		},
		{
			name: "artifacts",
			service: ServiceConfig{
				Service:        "artifacts",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/artifacts",
				Group:          "artifacts",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: assertArtifactsSDKFields,
		},
		{
			name: "networkloadbalancer",
			service: ServiceConfig{
				Service:        "networkloadbalancer",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/networkloadbalancer",
				Group:          "networkloadbalancer",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: assertNetworkLoadBalancerSDKFields,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			discoverer := NewDiscoverer()
			pkg, err := discoverer.BuildPackageModel(context.Background(), cfg, test.service)
			if err != nil {
				t.Fatalf("BuildPackageModel() error = %v", err)
			}
			test.assert(t, pkg)
		})
	}
}

func TestBuildPackageModelSynthesizesPSQLObservedStateFields(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "psql",
		SDKPackage:     "github.com/oracle/oci-go-sdk/v65/psql",
		Group:          "psql",
		PackageProfile: PackageProfileCRDOnly,
		ObservedState: ObservedStateConfig{
			SDKAliases: map[string][]string{
				"PrimaryDbInstance": []string{"PrimaryDbInstanceDetails"},
				"WorkRequestLog":    []string{"WorkRequestLogEntry"},
			},
		},
	}

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	assertPackageResourceStatusFields(t, pkg, map[string][]string{
		"DbSystem":          {"DisplayName", "CompartmentId", "Shape", "DbVersion"},
		"Configuration":     {"DisplayName", "Shape", "DbVersion", "InstanceOcpuCount"},
		"Backup":            {"DisplayName", "CompartmentId", "DbSystemId", "RetentionPeriod"},
		"PrimaryDbInstance": {"DbInstanceId"},
		"WorkRequestLog":    {"Message", "Timestamp"},
	})
}

func TestBuildPackageModelExcludesMySQLDbSystemSourceURLFromObservedState(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "mysql",
		SDKPackage:     "github.com/oracle/oci-go-sdk/v65/mysql",
		Group:          "mysql",
		PackageProfile: PackageProfileCRDOnly,
		ObservedState: ObservedStateConfig{
			ExcludedFieldPaths: map[string][]string{
				"DbSystem": {"Source.SourceUrl"},
			},
		},
	}

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	dbSystem := findResource(t, pkg.Resources, "DbSystem")

	specSource := findFieldModel(t, dbSystem.SpecFields, "Source")
	assertFieldType(t, "DbSystem spec Source", specSource, "DbSystemSource")
	assertHelperTypeFields(t, findHelperType(t, dbSystem.HelperTypes, "DbSystemSource"), "JsonData", "SourceType", "BackupId", "SourceUrl", "DbSystemId", "RecoveryPoint")

	statusSource := findFieldModel(t, dbSystem.StatusFields, "Source")
	assertFieldType(t, "DbSystem status Source", statusSource, "DbSystemSourceObservedState")
	statusSourceType := findHelperType(t, dbSystem.HelperTypes, "DbSystemSourceObservedState")
	assertHelperTypeFields(t, statusSourceType, "JsonData", "SourceType", "BackupId", "DbSystemId", "RecoveryPoint")
	assertFieldNamesAbsent(t, statusSourceType.Name+" fields", statusSourceType.Fields, "SourceUrl")
}

func TestBuildPackageModelSynthesizesQueueObservedStateFields(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "queue",
		SDKPackage:     "github.com/oracle/oci-go-sdk/v65/queue",
		Group:          "queue",
		PackageProfile: PackageProfileCRDOnly,
		ObservedState: ObservedStateConfig{
			SDKAliases: map[string][]string{
				"WorkRequestLog": {"WorkRequestLogEntry"},
			},
		},
	}

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	workRequestLog := findResource(t, pkg.Resources, "WorkRequestLog")
	for _, fieldName := range []string{"Message", "Timestamp"} {
		if !hasField(workRequestLog.StatusFields, fieldName) {
			t.Fatalf("WorkRequestLog status fields = %#v, want %s", workRequestLog.StatusFields, fieldName)
		}
	}
}

func TestBuildPackageModelSynthesizesNoSQLObservedStateFields(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "nosql",
		SDKPackage:     "github.com/oracle/oci-go-sdk/v65/nosql",
		Group:          "nosql",
		PackageProfile: PackageProfileCRDOnly,
		ObservedState: ObservedStateConfig{
			SDKAliases: map[string][]string{
				"WorkRequestLog": {"WorkRequestLogEntry"},
			},
		},
	}

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	workRequestLog := findResource(t, pkg.Resources, "WorkRequestLog")
	for _, fieldName := range []string{"Message", "Timestamp"} {
		if !hasField(workRequestLog.StatusFields, fieldName) {
			t.Fatalf("WorkRequestLog status fields = %#v, want %s", workRequestLog.StatusFields, fieldName)
		}
	}
}

func TestBuildPackageModelSynthesizesContainerEngineObservedStateFields(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "containerengine",
		SDKPackage:     "github.com/oracle/oci-go-sdk/v65/containerengine",
		Group:          "containerengine",
		PackageProfile: PackageProfileCRDOnly,
		ObservedState: ObservedStateConfig{
			SDKAliases: map[string][]string{
				"WorkRequestLog": {"WorkRequestLogEntry"},
			},
		},
	}

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	assertPackageResourceStatusFields(t, pkg, map[string][]string{
		"Cluster":         {"Name", "CompartmentId", "EndpointConfig", "VcnId", "KubernetesVersion", "KmsKeyId", "FreeformTags", "DefinedTags", "Options", "ImagePolicyConfig", "ClusterPodNetworkOptions", "Type"},
		"NodePool":        {"CompartmentId", "ClusterId", "Name", "KubernetesVersion", "NodeMetadata", "NodeImageName", "NodeSourceDetails", "NodeShapeConfig", "InitialNodeLabels", "SshPublicKey", "QuantityPerSubnet", "SubnetIds", "NodeConfigDetails", "FreeformTags", "DefinedTags", "NodeEvictionNodePoolSettings", "NodePoolCyclingDetails"},
		"VirtualNodePool": {"CompartmentId", "ClusterId", "DisplayName", "PlacementConfigurations", "InitialVirtualNodeLabels", "Taints", "Size", "NsgIds", "PodConfiguration", "FreeformTags", "DefinedTags", "VirtualNodeTags"},
		"Addon":           {"Version", "Configurations"},
		"WorkloadMapping": {"Namespace", "MappedCompartmentId", "FreeformTags", "DefinedTags"},
		"WorkRequestLog":  {"Message", "Timestamp"},
	})

	workRequest := findResource(t, pkg.Resources, "WorkRequest")
	workRequestStatus := findFieldModel(t, workRequest.StatusFields, "Status")
	assertFieldTag(t, "WorkRequest Status", workRequestStatus, `json:"sdkStatus,omitempty"`)

	credentialRotationStatus := findResource(t, pkg.Resources, "CredentialRotationStatus")
	credentialRotationObservedStatus := findFieldModel(t, credentialRotationStatus.StatusFields, "Status")
	assertFieldTag(t, "CredentialRotationStatus Status", credentialRotationObservedStatus, `json:"sdkStatus,omitempty"`)
}

func TestBuildPackageModelSynthesizesDNSObservedStateAliases(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "dns",
		SDKPackage:     "github.com/oracle/oci-go-sdk/v65/dns",
		Group:          "dns",
		PackageProfile: PackageProfileCRDOnly,
		ObservedState: ObservedStateConfig{
			SDKAliases: map[string][]string{
				"DomainRecord":     {"Record"},
				"ZoneFromZoneFile": {"Zone"},
				"ZoneRecord":       {"Record"},
			},
		},
	}

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	domainRecord := findResource(t, pkg.Resources, "DomainRecord")
	for _, fieldName := range []string{"Domain", "RecordHash", "IsProtected", "Rdata", "RrsetVersion", "Rtype", "Ttl"} {
		if !hasField(domainRecord.StatusFields, fieldName) {
			t.Fatalf("DomainRecord status fields = %#v, want %s", domainRecord.StatusFields, fieldName)
		}
	}

	zoneFromZoneFile := findResource(t, pkg.Resources, "ZoneFromZoneFile")
	for _, fieldName := range []string{"Name", "ZoneType", "CompartmentId", "Scope", "Id", "LifecycleState", "Nameservers"} {
		if !hasField(zoneFromZoneFile.StatusFields, fieldName) {
			t.Fatalf("ZoneFromZoneFile status fields = %#v, want %s", zoneFromZoneFile.StatusFields, fieldName)
		}
	}

	zoneRecord := findResource(t, pkg.Resources, "ZoneRecord")
	for _, fieldName := range []string{"Domain", "RecordHash", "IsProtected", "Rdata", "RrsetVersion", "Rtype", "Ttl"} {
		if !hasField(zoneRecord.StatusFields, fieldName) {
			t.Fatalf("ZoneRecord status fields = %#v, want %s", zoneRecord.StatusFields, fieldName)
		}
	}
}

func TestBuildPackageModelSynthesizesMonitoringObservedStateFields(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "monitoring",
		SDKPackage:     "github.com/oracle/oci-go-sdk/v65/monitoring",
		Group:          "monitoring",
		PackageProfile: PackageProfileCRDOnly,
		ObservedState: ObservedStateConfig{
			SDKAliases: map[string][]string{
				"AlarmHistory": {"AlarmHistoryCollection"},
			},
		},
	}

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	alarmHistory := findResource(t, pkg.Resources, "AlarmHistory")
	for _, fieldName := range []string{"AlarmId", "IsEnabled", "Entries"} {
		if !hasField(alarmHistory.StatusFields, fieldName) {
			t.Fatalf("AlarmHistory status fields = %#v, want %s", alarmHistory.StatusFields, fieldName)
		}
	}

	entryField := findFieldModel(t, alarmHistory.StatusFields, "Entries")
	if entryField.Type != "[]AlarmHistoryEntry" {
		t.Fatalf("AlarmHistory Entries type = %q, want %q", entryField.Type, "[]AlarmHistoryEntry")
	}

	entryHelper := findHelperType(t, alarmHistory.HelperTypes, "AlarmHistoryEntry")
	for _, fieldName := range []string{"Summary", "Timestamp", "TimestampTriggered"} {
		if !hasField(entryHelper.Fields, fieldName) {
			t.Fatalf("AlarmHistoryEntry helper fields = %#v, want %s", entryHelper.Fields, fieldName)
		}
	}
}

func TestBuildPackageModelSynthesizesONSObservedStateFields(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "ons",
		SDKPackage:     "github.com/oracle/oci-go-sdk/v65/ons",
		Group:          "ons",
		PackageProfile: PackageProfileCRDOnly,
		ObservedState: ObservedStateConfig{
			SDKAliases: map[string][]string{
				"ConfirmSubscription": {"ConfirmationResult"},
			},
		},
	}

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	confirmSubscription := findResource(t, pkg.Resources, "ConfirmSubscription")
	for _, fieldName := range []string{"Endpoint", "Message", "SubscriptionId", "TopicId", "TopicName", "UnsubscribeUrl"} {
		if !hasField(confirmSubscription.StatusFields, fieldName) {
			t.Fatalf("ConfirmSubscription status fields = %#v, want %s", confirmSubscription.StatusFields, fieldName)
		}
	}
}

func TestBuildPackageModelSynthesizesWorkRequestsObservedStateAlias(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "workrequests",
		SDKPackage:     "github.com/oracle/oci-go-sdk/v65/workrequests",
		Group:          "workrequests",
		PackageProfile: PackageProfileCRDOnly,
		ObservedState: ObservedStateConfig{
			SDKAliases: map[string][]string{
				"WorkRequestLog": {"WorkRequestLogEntry"},
			},
		},
	}

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	workRequestLog := findResource(t, pkg.Resources, "WorkRequestLog")
	for _, fieldName := range []string{"Message", "Timestamp"} {
		if !hasField(workRequestLog.StatusFields, fieldName) {
			t.Fatalf("WorkRequestLog status fields = %#v, want %s", workRequestLog.StatusFields, fieldName)
		}
	}
}

func TestBuildPackageModelSynthesizesIdentityObservedStateAliases(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "identity",
		SDKPackage:     "github.com/oracle/oci-go-sdk/v65/identity",
		Group:          "identity",
		PackageProfile: PackageProfileCRDOnly,
		ObservedState: ObservedStateConfig{
			SDKAliases: map[string][]string{
				"BulkActionResourceType":    []string{"BulkActionResourceTypeCollection"},
				"BulkEditTagsResourceType":  []string{"BulkEditTagsResourceTypeCollection"},
				"CostTrackingTag":           []string{"Tag"},
				"IdentityProvider":          []string{"Saml2IdentityProvider"},
				"NetworkSource":             []string{"NetworkSources"},
				"OrResetUIPassword":         []string{"UiPassword"},
				"StandardTagNamespace":      []string{"StandardTagNamespaceTemplate", "StandardTagNamespaceTemplateSummary"},
				"StandardTagTemplate":       []string{"StandardTagDefinitionTemplate"},
				"UserState":                 []string{"User"},
				"UserUIPasswordInformation": []string{"UiPasswordInformation"},
			},
		},
	}

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	assertPackageResourceStatusFields(t, pkg, map[string][]string{
		"BulkActionResourceType":    {"Items"},
		"BulkEditTagsResourceType":  {"Items"},
		"CostTrackingTag":           {"TagNamespaceId", "TagNamespaceName", "IsRetired", "Validator"},
		"IdentityProvider":          {"CompartmentId", "Name", "Description", "Metadata", "MetadataUrl", "ProductType"},
		"NetworkSource":             {"CompartmentId", "Name", "Description", "PublicSourceList", "Services", "VirtualSourceList"},
		"OrResetUIPassword":         {"Password", "UserId", "TimeCreated", "LifecycleState", "InactiveStatus"},
		"StandardTagNamespace":      {"Description", "StandardTagNamespaceName", "TagDefinitionTemplates"},
		"StandardTagTemplate":       {"Description", "TagDefinitionName", "Type", "IsCostTracking"},
		"UserState":                 {"Id", "CompartmentId", "Name", "LifecycleState", "Capabilities"},
		"UserUIPasswordInformation": {"UserId", "TimeCreated", "LifecycleState"},
	})

	standardTagNamespace := findResource(t, pkg.Resources, "StandardTagNamespace")
	standardTagNamespaceStatus := findFieldModel(t, standardTagNamespace.StatusFields, "Status")
	assertFieldTag(t, "StandardTagNamespace Status", standardTagNamespaceStatus, `json:"sdkStatus,omitempty"`)
}

func TestBuildPackageModelSynthesizesCoreObservedStateAliases(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "core",
		SDKPackage:     "github.com/oracle/oci-go-sdk/v65/core",
		Group:          "core",
		PackageProfile: PackageProfileCRDOnly,
		ObservedState: ObservedStateConfig{
			SDKAliases: map[string][]string{
				"AllDrgAttachment":                                {"DrgAttachmentInfo"},
				"AllowedPeerRegionsForRemotePeering":              {"PeerRegionForRemotePeering"},
				"AppCatalogListingAgreement":                      {"AppCatalogListingResourceVersionAgreements"},
				"ClusterNetworkInstance":                          {"InstanceSummary"},
				"ComputeCapacityReservationInstance":              {"CapacityReservationInstanceSummary"},
				"ComputeGlobalImageCapabilitySchema":              {"ComputeGlobalImageCapabilitySchemaVersionSummary"},
				"CrossConnectLetterOfAuthority":                   {"LetterOfAuthority"},
				"CrossConnectMapping":                             {"CrossConnectMappingDetails"},
				"CrossconnectPortSpeedShape":                      {"CrossConnectPortSpeedShape"},
				"DhcpOption":                                      {"DhcpOptions"},
				"FastConnectProviderVirtualCircuitBandwidthShape": {"VirtualCircuitBandwidthShape"},
				"IPSecConnectionTunnelError":                      {"IpSecConnectionTunnelErrorDetails"},
				"IPSecConnectionTunnelRoute":                      {"TunnelRouteSummary"},
				"IPSecConnectionTunnelSecurityAssociation":        {"TunnelSecurityAssociationSummary"},
				"InstanceDevice":                                  {"Device"},
				"NetworkSecurityGroupSecurityRule":                {"SecurityRule"},
				"VirtualCircuitAssociatedTunnel":                  {"VirtualCircuitAssociatedTunnelDetails"},
				"VolumeBackupPolicyAssetAssignment":               {"VolumeBackupPolicyAssignment"},
				"WindowsInstanceInitialCredential":                {"InstanceCredentials"},
			},
		},
	}

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	assertPackageResourceStatusFields(t, pkg, map[string][]string{
		"ClusterNetworkInstance":                          {"AvailabilityDomain", "CompartmentId", "Region", "State", "TimeCreated"},
		"ComputeCapacityReservationInstance":              {"AvailabilityDomain", "CompartmentId", "Id", "Shape"},
		"ComputeGlobalImageCapabilitySchema":              {"ComputeGlobalImageCapabilitySchemaId", "Name"},
		"NetworkSecurityGroupSecurityRule":                {"Direction", "Protocol", "Id", "TcpOptions", "UdpOptions"},
		"IPSecConnectionTunnelError":                      {"ErrorCode", "ErrorDescription", "Id", "Solution", "Timestamp"},
		"IPSecConnectionTunnelRoute":                      {"Advertiser", "AsPath", "IsBestPath", "Prefix"},
		"IPSecConnectionTunnelSecurityAssociation":        {"CpeSubnet", "OracleSubnet", "TunnelSaStatus"},
		"InstanceDevice":                                  {"IsAvailable", "Name"},
		"VolumeBackupPolicyAssetAssignment":               {"AssetId", "Id", "PolicyId", "TimeCreated"},
		"WindowsInstanceInitialCredential":                {"Password", "Username"},
		"FastConnectProviderVirtualCircuitBandwidthShape": {"BandwidthInMbps", "Name"},
		"CrossconnectPortSpeedShape":                      {"Name", "PortSpeedInGbps"},
		"AllDrgAttachment":                                {"Id"},
		"AllowedPeerRegionsForRemotePeering":              {"Name"},
		"AppCatalogListingAgreement":                      {"ListingId", "ListingResourceVersion", "OracleTermsOfUseLink", "EulaLink", "TimeRetrieved", "Signature"},
		"CrossConnectLetterOfAuthority":                   {"CrossConnectId", "FacilityLocation", "TimeExpires"},
		"CrossConnectMapping":                             {"Ipv4BgpStatus", "Ipv6BgpStatus", "OciLogicalDeviceName"},
		"DhcpOption":                                      {"CompartmentId", "DisplayName", "LifecycleState", "Options", "TimeCreated", "VcnId"},
		"VirtualCircuitAssociatedTunnel":                  {"TunnelId", "TunnelType"},
	})
}

func TestBuildPackageModelAvoidsStatusTypeCollisions(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "monitoring",
		SDKPackage:     "example.com/test/sdk",
		Group:          "monitoring",
		PackageProfile: PackageProfileCRDOnly,
	}

	pkg, err := buildPackageModel(cfg, service, []ResourceModel{
		{
			Kind:           "Alarm",
			FileStem:       "alarm",
			KindPlural:     "alarms",
			StatusTypeName: defaultStatusTypeName("Alarm"),
			StatusComments: []string{"AlarmStatus defines the observed state of Alarm."},
		},
		{
			Kind:           "AlarmStatus",
			FileStem:       "alarmstatus",
			KindPlural:     "alarmstatuses",
			StatusTypeName: defaultStatusTypeName("AlarmStatus"),
			StatusComments: []string{"AlarmStatusObservedState defines the observed state of AlarmStatus."},
		},
	})
	if err != nil {
		t.Fatalf("buildPackageModel() error = %v", err)
	}

	alarm := findResource(t, pkg.Resources, "Alarm")
	if alarm.StatusTypeName != "AlarmObservedState" {
		t.Fatalf("Alarm status type = %q, want %q", alarm.StatusTypeName, "AlarmObservedState")
	}
	if len(alarm.StatusComments) != 1 || alarm.StatusComments[0] != "AlarmObservedState defines the observed state of Alarm." {
		t.Fatalf("Alarm status comments = %#v, want updated default comment", alarm.StatusComments)
	}

	alarmStatus := findResource(t, pkg.Resources, "AlarmStatus")
	if alarmStatus.StatusTypeName != "AlarmStatusObservedState" {
		t.Fatalf("AlarmStatus status type = %q, want %q", alarmStatus.StatusTypeName, "AlarmStatusObservedState")
	}
}

func TestBuildPackageModelAvoidsHelperTypeCollisions(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "containerengine",
		SDKPackage:     "example.com/test/sdk",
		Group:          "containerengine",
		PackageProfile: PackageProfileCRDOnly,
	}

	pkg, err := buildPackageModel(cfg, service, []ResourceModel{
		{
			Kind:           "Cluster",
			FileStem:       "cluster",
			KindPlural:     "clusters",
			StatusTypeName: defaultStatusTypeName("Cluster"),
			StatusComments: []string{"ClusterStatus defines the observed state of Cluster."},
			SpecFields: []FieldModel{
				{
					Name: "EndpointConfig",
					Type: "ClusterEndpointConfig",
					Tag:  `json:"endpointConfig,omitempty"`,
				},
			},
			HelperTypes: []TypeModel{
				{
					Name:     "ClusterEndpointConfig",
					Comments: []string{"ClusterEndpointConfig defines nested fields for Cluster.EndpointConfig."},
					Fields: []FieldModel{
						{
							Name: "SubnetId",
							Type: "string",
							Tag:  `json:"subnetId,omitempty"`,
						},
					},
				},
			},
		},
		{
			Kind:           "ClusterEndpointConfig",
			FileStem:       "clusterendpointconfig",
			KindPlural:     "clusterendpointconfigs",
			StatusTypeName: defaultStatusTypeName("ClusterEndpointConfig"),
			StatusComments: []string{"ClusterEndpointConfigStatus defines the observed state of ClusterEndpointConfig."},
		},
	})
	if err != nil {
		t.Fatalf("buildPackageModel() error = %v", err)
	}

	cluster := findResource(t, pkg.Resources, "Cluster")
	endpointConfig := findFieldModel(t, cluster.SpecFields, "EndpointConfig")
	if endpointConfig.Type != "ClusterEndpointConfigFields" {
		t.Fatalf("Cluster EndpointConfig type = %q, want %q", endpointConfig.Type, "ClusterEndpointConfigFields")
	}

	helperType := findHelperType(t, cluster.HelperTypes, "ClusterEndpointConfigFields")
	if len(helperType.Comments) != 1 || helperType.Comments[0] != "ClusterEndpointConfigFields defines nested fields for Cluster.EndpointConfig." {
		t.Fatalf("helper comments = %#v, want renamed default comment", helperType.Comments)
	}

	clusterEndpointConfig := findResource(t, pkg.Resources, "ClusterEndpointConfig")
	if clusterEndpointConfig.StatusTypeName != "ClusterEndpointConfigStatus" {
		t.Fatalf("ClusterEndpointConfig status type = %q, want %q", clusterEndpointConfig.StatusTypeName, "ClusterEndpointConfigStatus")
	}
}

func TestBuildPackageModelAppliesResourceFieldAndSampleOverrides(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "mysql",
		SDKPackage:     "example.com/test/sdk",
		Group:          "mysql",
		PackageProfile: PackageProfileCRDOnly,
		Generation: GenerationConfig{
			Resources: []ResourceGenerationOverride{
				{
					Kind: "Widget",
					SpecFields: []FieldOverride{
						{Name: "DisplayName", Type: "shared.UsernameSource", Tag: `json:"displayName,omitempty"`},
					},
					StatusFields: []FieldOverride{
						{Name: "AdminUsername", Type: "shared.UsernameSource", Tag: `json:"adminUsername,omitempty"`},
					},
					Sample: SampleOverride{MetadataName: "widget-sample"},
				},
			},
		},
	}

	pkg, err := buildPackageModel(cfg, service, []ResourceModel{
		{
			SDKName:        "Widget",
			Kind:           "Widget",
			FileStem:       "widget",
			KindPlural:     "widgets",
			StatusTypeName: defaultStatusTypeName("Widget"),
			SpecComments:   []string{"WidgetSpec defines the desired state of Widget."},
			StatusComments: []string{"WidgetStatus defines the observed state of Widget."},
			SpecFields: []FieldModel{
				{
					Name:     "DisplayName",
					Type:     "string",
					Tag:      `json:"displayName,omitempty"`,
					Comments: []string{"Original display name comment."},
					Markers:  []string{"+kubebuilder:validation:Optional"},
				},
				{Name: "Enabled", Type: "bool", Tag: `json:"enabled,omitempty"`},
			},
			StatusFields: []FieldModel{
				{Name: "OsokStatus", Type: "shared.OSOKStatus", Tag: `json:"status"`},
				{Name: "Id", Type: "string", Tag: `json:"id,omitempty"`},
			},
			Sample: SampleModel{MetadataName: "widget-default"},
		},
	})
	if err != nil {
		t.Fatalf("buildPackageModel() error = %v", err)
	}

	resource := findResource(t, pkg.Resources, "Widget")
	displayName := findFieldModel(t, resource.SpecFields, "DisplayName")
	assertFieldType(t, "Widget DisplayName", displayName, "shared.UsernameSource")
	assertFieldCommentsEqual(t, "Widget DisplayName", displayName, []string{"Original display name comment."})
	assertFieldMarkers(t, "Widget DisplayName", displayName, []string{"+kubebuilder:validation:Optional"})
	assertResourceSpecFields(t, resource, "Enabled")
	assertResourceStatusFields(t, resource, "Id", "AdminUsername")
	if resource.Sample.MetadataName != "widget-sample" {
		t.Fatalf("Widget sample metadata name = %q, want %q", resource.Sample.MetadataName, "widget-sample")
	}
}

func TestGenerateRendersAndSkipsExisting(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileControllerBacked)
	pipeline := newTestGenerator(t)

	outputRoot := t.TempDir()
	result, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Generated) != 1 {
		t.Fatalf("Generate() generated = %d services, want 1", len(result.Generated))
	}

	groupVersionPath := filepath.Join(outputRoot, "api", "mysql", "v1beta1", "groupversion_info.go")
	assertFileContains(t, groupVersionPath, []string{
		"// Code generated by generator. DO NOT EDIT.",
		`GroupVersion = schema.GroupVersion{Group: "mysql.oracle.com", Version: "v1beta1"}`,
	})

	resourcePath := filepath.Join(outputRoot, "api", "mysql", "v1beta1", "dbsystem_types.go")
	resourceContent, err := os.ReadFile(resourcePath)
	if err != nil {
		t.Fatalf("read %s: %v", resourcePath, err)
	}
	if !strings.Contains(string(resourceContent), "type DbSystemSpec struct") {
		t.Fatalf("dbsystem_types.go did not render the expected kind: %s", string(resourceContent))
	}
	if !strings.Contains(string(resourceContent), "Port") || !strings.Contains(string(resourceContent), `json:"port,omitempty"`) {
		t.Fatalf("dbsystem_types.go did not render the expected Port field: %s", string(resourceContent))
	}

	result, err = pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot:   outputRoot,
		SkipExisting: true,
	})
	if err != nil {
		t.Fatalf("Generate() second run error = %v", err)
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("Generate() skipped = %d services, want 1", len(result.Skipped))
	}
}

func TestRenderResourceFileIncludesSDKDocumentationAndRequiredness(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "functions",
		SDKPackage:     "github.com/oracle/oci-go-sdk/v65/functions",
		Group:          "functions",
		PackageProfile: PackageProfileCRDOnly,
	}

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	application := findResource(t, pkg.Resources, "Application")

	compartmentID := findFieldModel(t, application.SpecFields, "CompartmentId")
	assertFieldMarkers(t, "Application CompartmentId", compartmentID, []string{"+kubebuilder:validation:Required"})
	assertFieldTag(t, "Application CompartmentId", compartmentID, `json:"compartmentId"`)
	assertFieldCommentsContain(t, "Application CompartmentId", compartmentID, "compartment to create the application within")

	config := findFieldModel(t, application.SpecFields, "Config")
	assertFieldMarkers(t, "Application Config", config, []string{"+kubebuilder:validation:Optional"})
	assertFieldTag(t, "Application Config", config, `json:"config,omitempty"`)
	assertFieldCommentsContain(t, "Application Config", config, "Application configuration")

	lifecycleState := findFieldModel(t, application.StatusFields, "LifecycleState")
	assertFieldHasNoMarkers(t, "Application LifecycleState", lifecycleState)
	assertFieldCommentsContain(t, "Application LifecycleState", lifecycleState, "current state of the application")

	content, err := renderResourceFile(pkg, application)
	if err != nil {
		t.Fatalf("renderResourceFile() error = %v", err)
	}

	assertContains(t, content, []string{
		"// The OCID of the compartment to create the application within.",
		"// +kubebuilder:validation:Required",
		"CompartmentId string `json:\"compartmentId\"`",
		"// Application configuration. These values are passed on to the function as environment variables, functions may override application configuration.",
		"// +kubebuilder:validation:Optional",
		"Config map[string]string `json:\"config,omitempty\"`",
		"// The current state of the application.",
		"LifecycleState string `json:\"lifecycleState,omitempty\"`",
	})
}

func TestGenerateRendersPackageOutputsByProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		profile                 string
		wantMetadataContains    []string
		wantMetadataNotContains []string
		wantInstallContains     []string
		wantInstallNotContains  []string
	}{
		{
			name:    "controller-backed",
			profile: PackageProfileControllerBacked,
			wantMetadataContains: []string{
				"PACKAGE_NAME=oci-service-operator-mysql",
				"CRD_PATHS=./api/mysql/...",
				"RBAC_PATHS=./controllers/mysql/...",
			},
			wantInstallContains: []string{
				"- generated/crd",
				"- generated/rbac",
				"- ../../../config/manager",
			},
		},
		{
			name:    "crd-only",
			profile: PackageProfileCRDOnly,
			wantMetadataContains: []string{
				"PACKAGE_NAME=oci-service-operator-mysql",
				"CRD_PATHS=./api/mysql/...",
			},
			wantMetadataNotContains: []string{
				"RBAC_PATHS=",
			},
			wantInstallContains: []string{
				"- generated/crd",
			},
			wantInstallNotContains: []string{
				"generated/rbac",
				"../../../config/manager",
				"dbsystem_editor_role.yaml",
				"dbsystem_viewer_role.yaml",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{
				Domain:         "oracle.com",
				DefaultVersion: "v1beta1",
			}
			service := testServiceConfig(test.profile)
			pipeline := newTestGenerator(t)

			outputRoot := t.TempDir()
			if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
				OutputRoot: outputRoot,
			}); err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			metadataContent := readFile(t, filepath.Join(outputRoot, "packages", "mysql", "metadata.env"))
			assertContains(t, metadataContent, test.wantMetadataContains)
			assertNotContains(t, metadataContent, test.wantMetadataNotContains)

			installContent := readFile(t, filepath.Join(outputRoot, "packages", "mysql", "install", "kustomization.yaml"))
			assertContains(t, installContent, test.wantInstallContains)
			assertNotContains(t, installContent, test.wantInstallNotContains)

			sampleContent := readFile(t, filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_dbsystem.yaml"))
			assertContains(t, sampleContent, []string{
				"apiVersion: mysql.oracle.com/v1beta1",
				"kind: DbSystem",
				"name: dbsystem-sample",
			})

			sampleKustomization := readFile(t, filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"))
			assertContains(t, sampleKustomization, []string{
				"- mysql_v1beta1_dbsystem.yaml",
				"# +kubebuilder:scaffold:manifestskustomizesamples",
			})
		})
	}
}

func TestGenerateRendersControllerOutputs(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileControllerBacked)
	service.Generation.Controller.Strategy = GenerationStrategyGenerated
	service.Generation.Resources = []ResourceGenerationOverride{
		{
			Kind: "DbSystem",
			Controller: ControllerGenerationOverride{
				MaxConcurrentReconciles: 3,
				ExtraRBACMarkers: []string{
					`groups="",resources=secrets,verbs=get;list;watch`,
					`groups="",resources=events,verbs=create;patch`,
				},
			},
		},
	}
	pipeline := newTestGenerator(t)

	outputRoot := t.TempDir()
	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	content := readFile(t, filepath.Join(outputRoot, "controllers", "mysql", "dbsystem_controller.go"))
	assertContains(t, content, []string{
		"package mysql",
		`mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"`,
		"type DbSystemReconciler struct {",
		"// +kubebuilder:rbac:groups=mysql.oracle.com,resources=dbsystems,verbs=get;list;watch;create;update;patch;delete",
		"// +kubebuilder:rbac:groups=mysql.oracle.com,resources=dbsystems/status,verbs=get;update;patch",
		"// +kubebuilder:rbac:groups=mysql.oracle.com,resources=dbsystems/finalizers,verbs=update",
		`// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch`,
		`// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch`,
		"builder = builder.WithOptions(controller.Options{MaxConcurrentReconciles: 3})",
		"dbSystem := &mysqlv1beta1.DbSystem{}",
		"return r.Reconciler.Reconcile(ctx, req, dbSystem)",
	})
}

func TestGenerateDoesNotOverwriteExistingControllerOutput(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileControllerBacked)
	service.Generation.Controller.Strategy = GenerationStrategyGenerated
	pipeline := newTestGenerator(t)

	outputRoot := t.TempDir()
	controllerDir := filepath.Join(outputRoot, "controllers", "mysql")
	if err := os.MkdirAll(controllerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", controllerDir, err)
	}
	controllerPath := filepath.Join(controllerDir, "dbsystem_controller.go")
	if err := os.WriteFile(controllerPath, []byte("package mysql\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", controllerPath, err)
	}

	_, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	})
	if err == nil {
		t.Fatal("Generate() unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), controllerPath) {
		t.Fatalf("Generate() error = %v, want controller path %q", err, controllerPath)
	}
}

func TestGenerateRendersRegistrationOutputs(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileControllerBacked)
	service.Generation.Controller.Strategy = GenerationStrategyGenerated
	service.Generation.ServiceManager.Strategy = GenerationStrategyGenerated
	service.Generation.Registration.Strategy = GenerationStrategyGenerated
	pipeline := newTestGenerator(t)

	outputRoot := t.TempDir()
	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	content := readFile(t, filepath.Join(outputRoot, "internal", "registrations", "mysql_generated.go"))
	assertContains(t, content, []string{
		"package registrations",
		`mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"`,
		`mysqlcontrollers "github.com/oracle/oci-service-operator/controllers/mysql"`,
		`mysqldbsystemservicemanager "github.com/oracle/oci-service-operator/pkg/servicemanager/mysql/dbsystem"`,
		"registerGeneratedGroup(GroupRegistration{",
		`Group:       "mysql",`,
		"AddToScheme: mysqlv1beta1.AddToScheme,",
		"(&mysqlcontrollers.DbSystemReconciler{",
		`ctx,`,
		`"DbSystem",`,
		"return mysqldbsystemservicemanager.NewDbSystemServiceManagerWithDeps(deps)",
	})
}

func TestBuildRegistrationOutputSkipsKindsOptedOutOfGeneratedRuntime(t *testing.T) {
	t.Parallel()

	service := ServiceConfig{
		Service:        "streaming",
		Group:          "streaming",
		PackageProfile: PackageProfileControllerBacked,
		Generation: GenerationConfig{
			Controller:     GenerationSurfaceConfig{Strategy: GenerationStrategyGenerated},
			ServiceManager: GenerationSurfaceConfig{Strategy: GenerationStrategyGenerated},
			Registration:   GenerationSurfaceConfig{Strategy: GenerationStrategyGenerated},
			Resources: []ResourceGenerationOverride{
				{
					Kind: "Cursor",
					Controller: ControllerGenerationOverride{
						Strategy: GenerationStrategyNone,
					},
					ServiceManager: ServiceManagerGenerationOverride{
						Strategy: GenerationStrategyNone,
					},
				},
			},
		},
	}

	registration, err := buildRegistrationOutputModel(
		service,
		"v1beta1",
		[]ResourceModel{
			{Kind: "Stream"},
			{Kind: "Cursor"},
		},
		ControllerOutputModel{
			Resources: []ControllerModel{
				{Kind: "Stream", ReconcilerType: "StreamReconciler"},
			},
		},
		[]ServiceManagerModel{
			{
				Kind:                "Stream",
				FileStem:            "stream",
				PackagePath:         "streaming/stream",
				WithDepsConstructor: "NewStreamServiceManagerWithDeps",
			},
		},
		"streaming",
	)
	if err != nil {
		t.Fatalf("buildRegistrationOutputModel() error = %v", err)
	}
	if len(registration.Resources) != 1 {
		t.Fatalf("len(registration.Resources) = %d, want 1", len(registration.Resources))
	}
	if registration.Resources[0].Kind != "Stream" {
		t.Fatalf("registration.Resources[0].Kind = %q, want %q", registration.Resources[0].Kind, "Stream")
	}
}

func TestGenerateRejectsGeneratedRegistrationWithoutGeneratedRuntimeSurfaces(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileControllerBacked)
	service.Generation.ServiceManager.Strategy = GenerationStrategyGenerated
	service.Generation.Registration.Strategy = GenerationStrategyGenerated
	pipeline := newTestGenerator(t)

	_, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: t.TempDir(),
	})
	if err == nil {
		t.Fatal("Generate() unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), `registration strategy "generated" requires generated controller output for kind "DbSystem"`) {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestGenerateControllerBackedPackagesDoNotReferenceEditorViewerRoles(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "events",
		SDKPackage:     "example.com/test/sdk",
		Group:          "events",
		PackageProfile: PackageProfileControllerBacked,
	}
	pipeline := newTestGenerator(t)

	outputRoot := t.TempDir()
	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	installContent := readFile(t, filepath.Join(outputRoot, "packages", "events", "install", "kustomization.yaml"))
	assertContains(t, installContent, []string{
		"- generated/crd",
		"- generated/rbac",
		"- ../../../config/manager",
	})
	assertNotContains(t, installContent, []string{
		"_editor_role.yaml",
		"_viewer_role.yaml",
	})
}

func TestGeneratedControllerCompiles(t *testing.T) {
	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileControllerBacked)
	service.Generation.Controller.Strategy = GenerationStrategyManual
	service.Generation.Resources = []ResourceGenerationOverride{
		{
			Kind: "DbSystem",
			Controller: ControllerGenerationOverride{
				Strategy:                GenerationStrategyGenerated,
				MaxConcurrentReconciles: 3,
				ExtraRBACMarkers: []string{
					`groups="",resources=secrets,verbs=get;list;watch`,
				},
			},
		},
	}
	pipeline := newTestGenerator(t)

	moduleRoot := repoRoot(t)
	outputRoot, err := os.MkdirTemp(moduleRoot, "generated-controller-")
	if err != nil {
		t.Fatalf("MkdirTemp(%s) error = %v", moduleRoot, err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(outputRoot)
	})

	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	relativePackagePath, err := filepath.Rel(moduleRoot, filepath.Join(outputRoot, "controllers", "mysql"))
	if err != nil {
		t.Fatalf("Rel() error = %v", err)
	}

	cmd := exec.Command("go", "test", "./"+filepath.ToSlash(relativePackagePath))
	cmd.Dir = moduleRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test %s error = %v\n%s", relativePackagePath, err, output)
	}
}

func TestGenerateRendersServiceManagerScaffolds(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileControllerBacked)
	service.Generation.ServiceManager.Strategy = GenerationStrategyGenerated
	pipeline := newTestGenerator(t)

	outputRoot := t.TempDir()
	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	serviceClientPath := filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "dbsystem", "dbsystem_serviceclient.go")
	serviceManagerPath := filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "dbsystem", "dbsystem_servicemanager.go")

	serviceClientContent := readFile(t, serviceClientPath)
	assertContains(t, serviceClientContent, []string{
		"package dbsystem",
		"type DbSystemServiceClient interface {",
		"var newDbSystemServiceClient = func(manager *DbSystemServiceManager) DbSystemServiceClient {",
		`Kind:    "DbSystem"`,
		"github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime",
		"generatedruntime.NewServiceClient[*mysqlv1beta1.DbSystem]",
		"mysqlsdk.NewSampleClientWithConfigurationProvider(manager.Provider)",
	})

	serviceManagerContent := readFile(t, serviceManagerPath)
	assertContains(t, serviceManagerContent, []string{
		"type DbSystemServiceManager struct {",
		"func NewDbSystemServiceManagerWithDeps(deps servicemanager.RuntimeDeps) *DbSystemServiceManager {",
		"func (c *DbSystemServiceManager) WithClient(client DbSystemServiceClient) *DbSystemServiceManager {",
		"func (c *DbSystemServiceManager) GetCrdStatus(obj runtime.Object) (*shared.OSOKStatus, error) {",
		"return &resource.Status.OsokStatus",
	})
}

func TestGenerateRendersPerServiceManagerOutputs(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileControllerBacked)
	pipeline := newTestGenerator(t)

	outputRoot := t.TempDir()
	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	managerMainContent := readFile(t, filepath.Join(outputRoot, "cmd", "manager", "mysql", "main.go"))
	assertContains(t, managerMainContent, []string{
		"package main",
		`mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"`,
		`MetricsServiceName: "mysql"`,
		`LeaderElectionID:   "40558063.oci.mysql"`,
		`}, managerservices.ForGroup("mysql")); err != nil {`,
	})

	controllerConfigContent := readFile(t, filepath.Join(outputRoot, "config", "manager", "mysql", "controller_manager_config.yaml"))
	assertContains(t, controllerConfigContent, []string{
		"apiVersion: controller-runtime.sigs.k8s.io/v1alpha1",
		"kind: ControllerManagerConfiguration",
		"resourceName: 40558063.oci.mysql",
	})

	kustomizationContent := readFile(t, filepath.Join(outputRoot, "config", "manager", "mysql", "kustomization.yaml"))
	assertContains(t, kustomizationContent, []string{
		"kind: Kustomization",
		"- manager.yaml",
		"- controller_manager_config.yaml",
		"name: manager-config",
	})

	managerDeploymentContent := readFile(t, filepath.Join(outputRoot, "config", "manager", "mysql", "manager.yaml"))
	assertContains(t, managerDeploymentContent, []string{
		"kind: Deployment",
		"name: controller-manager",
		"image: controller:latest",
	})
}

func TestGenerateRendersPackageSplitOutputs(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileControllerBacked)
	service.PackageSplits = []PackageSplitConfig{
		{
			Name:                   "mysql-split",
			DefaultControllerImage: "iad.ocir.io/oracle/oci-service-operator-mysql-split:latest",
			IncludeKinds:           []string{"DbSystem"},
		},
	}
	service.Generation.Controller.Strategy = GenerationStrategyGenerated
	service.Generation.ServiceManager.Strategy = GenerationStrategyGenerated
	service.Generation.Registration.Strategy = GenerationStrategyGenerated
	pipeline := newTestGenerator(t)

	outputRoot := t.TempDir()
	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	metadataContent := readFile(t, filepath.Join(outputRoot, "packages", "mysql-split", "metadata.env"))
	assertContains(t, metadataContent, []string{
		"PACKAGE_NAME=oci-service-operator-mysql-split",
		"PACKAGE_NAMESPACE=oci-service-operator-mysql-split-system",
		"PACKAGE_NAME_PREFIX=oci-service-operator-mysql-split-",
		"CRD_PATHS=./api/mysql/...",
		"CRD_KIND_FILTER=DbSystem",
		"RBAC_PATHS=./controllers/mysql/...",
		"DEFAULT_CONTROLLER_IMAGE=iad.ocir.io/oracle/oci-service-operator-mysql-split:latest",
	})

	registrationContent := readFile(t, filepath.Join(outputRoot, "internal", "registrations", "mysql-split_generated.go"))
	assertContains(t, registrationContent, []string{
		`Group:       "mysql-split",`,
		`mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"`,
		`mysqlcontrollers "github.com/oracle/oci-service-operator/controllers/mysql"`,
	})

	managerMainContent := readFile(t, filepath.Join(outputRoot, "cmd", "manager", "mysql-split", "main.go"))
	assertContains(t, managerMainContent, []string{
		`mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"`,
		`MetricsServiceName: "mysql-split"`,
		`LeaderElectionID:   "40558063.oci.mysql-split"`,
		`}, managerservices.ForGroup("mysql-split")); err != nil {`,
	})
}

func TestRenderServiceClientFileHandlesEndpointConstructors(t *testing.T) {
	t.Parallel()

	content, err := renderServiceClientFile(ServiceManagerModel{
		Kind:                     "Cursor",
		SDKName:                  "Cursor",
		PackageName:              "cursor",
		APIImportPath:            "github.com/oracle/oci-service-operator/api/streaming/v1beta1",
		APIImportAlias:           "streamingv1beta1",
		SDKImportPath:            "github.com/oracle/oci-go-sdk/v65/streaming",
		SDKImportAlias:           "streamingsdk",
		ManagerTypeName:          "CursorServiceManager",
		ClientInterfaceName:      "CursorServiceClient",
		DefaultClientTypeName:    "defaultCursorServiceClient",
		SDKClientTypeName:        "StreamClient",
		SDKClientConstructor:     "NewStreamClientWithConfigurationProvider",
		SDKClientConstructorKind: "provider_endpoint",
		CreateOperation: &RuntimeOperationModel{
			MethodName:      "CreateCursor",
			RequestTypeName: "CreateCursorRequest",
			UsesRequest:     true,
		},
	})
	if err != nil {
		t.Fatalf("renderServiceClientFile() error = %v", err)
	}

	assertContains(t, content, []string{
		"sdkClient streamingsdk.StreamClient",
		`err = fmt.Errorf("streamingsdk.NewStreamClientWithConfigurationProvider requires an explicit service endpoint")`,
		"return sdkClient.CreateCursor(ctx, *request.(*streamingsdk.CreateCursorRequest))",
	})
	assertNotContains(t, content, []string{
		"NewStreamClientWithConfigurationProvider(manager.Provider)",
	})
}

func TestRenderServiceClientFileRendersFormalSemanticsAndRequestFields(t *testing.T) {
	t.Parallel()

	content, err := renderServiceClientFile(ServiceManagerModel{
		Kind:                     "Thing",
		SDKName:                  "Thing",
		PackageName:              "thing",
		APIImportPath:            "github.com/oracle/oci-service-operator/api/example/v1beta1",
		APIImportAlias:           "examplev1beta1",
		SDKImportPath:            "github.com/oracle/oci-go-sdk/v65/example",
		SDKImportAlias:           "examplesdk",
		ManagerTypeName:          "ThingServiceManager",
		ClientInterfaceName:      "ThingServiceClient",
		DefaultClientTypeName:    "defaultThingServiceClient",
		SDKClientTypeName:        "ExampleClient",
		SDKClientConstructor:     "NewExampleClientWithConfigurationProvider",
		SDKClientConstructorKind: "provider",
		Semantics: &RuntimeSemanticsModel{
			FormalService:     "identity",
			FormalSlug:        "user",
			StatusProjection:  "required",
			SecretSideEffects: "none",
			FinalizerPolicy:   "retain-until-confirmed-delete",
			Lifecycle: RuntimeLifecycleModel{
				ProvisioningStates: []string{"CREATING"},
				ActiveStates:       []string{"ACTIVE"},
			},
			Delete: RuntimeDeleteSemanticsModel{
				Policy:         "required",
				PendingStates:  []string{"DELETING"},
				TerminalStates: []string{"DELETED"},
			},
			List: &RuntimeListLookupModel{
				ResponseItemsField: "Items",
				MatchFields:        []string{"name"},
			},
			Mutation: RuntimeMutationModel{
				Mutable:       []string{"displayName"},
				ForceNew:      []string{"compartmentId"},
				ConflictsWith: map[string][]string{"name": {"displayName"}},
			},
			Hooks: RuntimeHookSetModel{
				Create: []RuntimeHookModel{{Helper: "tfresource.CreateResource"}},
			},
			CreateFollowUp: RuntimeFollowUpModel{
				Strategy: "read-after-write",
				Hooks:    []RuntimeHookModel{{Helper: "tfresource.CreateResource"}},
			},
		},
		CreateOperation: &RuntimeOperationModel{
			MethodName:      "CreateThing",
			RequestTypeName: "CreateThingRequest",
			UsesRequest:     true,
			RequestFields: []RuntimeRequestFieldModel{
				{FieldName: "CreateThingDetails", Contribution: "body"},
			},
		},
		GetOperation: &RuntimeOperationModel{
			MethodName:      "GetThing",
			RequestTypeName: "GetThingRequest",
			UsesRequest:     true,
			RequestFields: []RuntimeRequestFieldModel{
				{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true},
			},
		},
	})
	if err != nil {
		t.Fatalf("renderServiceClientFile() error = %v", err)
	}

	assertContains(t, content, []string{
		"Semantics: &generatedruntime.Semantics{",
		`FormalService:     "identity"`,
		`FormalSlug:        "user"`,
		`Fields: []generatedruntime.RequestField{{FieldName: "CreateThingDetails", RequestName: "", Contribution: "body", PreferResourceID: false}},`,
		`Fields: []generatedruntime.RequestField{{FieldName: "ThingId", RequestName: "thingId", Contribution: "path", PreferResourceID: true}},`,
		`CreateFollowUp: generatedruntime.FollowUpSemantics{`,
	})
}

func TestGenerateRendersServiceManagerScaffoldAtOverridePath(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileControllerBacked)
	service.Generation.ServiceManager.Strategy = GenerationStrategyGenerated
	service.Generation.Resources = []ResourceGenerationOverride{
		{
			Kind: "DbSystem",
			ServiceManager: ServiceManagerGenerationOverride{
				PackagePath: "mysql/runtime/dbsystem",
			},
		},
	}
	pipeline := newTestGenerator(t)

	outputRoot := t.TempDir()
	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	serviceClientContent := readFile(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "runtime", "dbsystem", "dbsystem_serviceclient.go"))
	serviceManagerContent := readFile(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "runtime", "dbsystem", "dbsystem_servicemanager.go"))

	assertContains(t, serviceClientContent, []string{"package dbsystem"})
	assertContains(t, serviceManagerContent, []string{"package dbsystem"})
}

func TestGeneratedServiceManagerScaffoldCompiles(t *testing.T) {
	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileControllerBacked)
	service.Generation.ServiceManager.Strategy = GenerationStrategyGenerated
	pipeline := newTestGenerator(t)

	moduleRoot := repoRoot(t)
	outputRoot, err := os.MkdirTemp(moduleRoot, "generated-servicemanager-")
	if err != nil {
		t.Fatalf("MkdirTemp(%s) error = %v", moduleRoot, err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(outputRoot)
	})

	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	relativePackagePath, err := filepath.Rel(moduleRoot, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "dbsystem"))
	if err != nil {
		t.Fatalf("Rel() error = %v", err)
	}

	cmd := exec.Command("go", "test", "./"+filepath.ToSlash(relativePackagePath))
	cmd.Dir = moduleRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test %s error = %v\n%s", relativePackagePath, err, output)
	}
}

func TestCurrentDefaultActiveGeneratedArtifactsMatchCheckedInOutputs(t *testing.T) {
	cfg := loadCheckedInConfig(t)
	services, err := cfg.SelectServices("", true)
	if err != nil {
		t.Fatalf("SelectServices(--all) error = %v", err)
	}
	if len(services) != 13 {
		t.Fatalf("selected %d default-active services, want 13", len(services))
	}

	outputRoot := t.TempDir()
	seedSamplesKustomization(t, outputRoot)
	pipeline := New()
	result, err := pipeline.Generate(context.Background(), cfg, services, Options{
		OutputRoot: outputRoot,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	assertGeneratedServiceCounts(t, result.Generated, map[string]int{
		"containerengine": 18,
		"core":            123,
		"database":        1,
		"dataflow":        10,
		"identity":        1,
		"mysql":           12,
		"nosql":           8,
		"opensearch":      1,
		"psql":            11,
		"queue":           7,
		"redis":           1,
		"streaming":       1,
		"vault":           2,
	})

	apiFiles := []string{
		"api/containerengine/v1beta1/groupversion_info.go",
		"api/containerengine/v1beta1/cluster_types.go",
		"api/core/v1beta1/groupversion_info.go",
		"api/core/v1beta1/vcn_types.go",
		"api/database/v1beta1/groupversion_info.go",
		"api/database/v1beta1/autonomousdatabase_types.go",
		"api/dataflow/v1beta1/groupversion_info.go",
		"api/dataflow/v1beta1/application_types.go",
		"api/identity/v1beta1/groupversion_info.go",
		"api/identity/v1beta1/compartment_types.go",
		"api/mysql/v1beta1/groupversion_info.go",
		"api/mysql/v1beta1/dbsystem_types.go",
		"api/nosql/v1beta1/groupversion_info.go",
		"api/nosql/v1beta1/table_types.go",
		"api/opensearch/v1beta1/groupversion_info.go",
		"api/opensearch/v1beta1/opensearchcluster_types.go",
		"api/psql/v1beta1/groupversion_info.go",
		"api/psql/v1beta1/dbsystem_types.go",
		"api/queue/v1beta1/groupversion_info.go",
		"api/queue/v1beta1/queue_types.go",
		"api/redis/v1beta1/groupversion_info.go",
		"api/redis/v1beta1/rediscluster_types.go",
		"api/streaming/v1beta1/groupversion_info.go",
		"api/streaming/v1beta1/stream_types.go",
		"api/vault/v1beta1/groupversion_info.go",
		"api/vault/v1beta1/secret_types.go",
	}
	for _, relativePath := range apiFiles {
		assertGoEquivalent(t, filepath.Join(repoRoot(t), relativePath), filepath.Join(outputRoot, relativePath))
	}

	exactFiles := []string{
		"config/samples/containerengine_v1beta1_cluster.yaml",
		"config/samples/core_v1beta1_vcn.yaml",
		"config/samples/database_v1beta1_autonomousdatabase.yaml",
		"config/samples/dataflow_v1beta1_application.yaml",
		"config/samples/identity_v1beta1_compartment.yaml",
		"config/samples/mysql_v1beta1_dbsystem.yaml",
		"config/samples/nosql_v1beta1_table.yaml",
		"config/samples/opensearch_v1beta1_opensearchcluster.yaml",
		"config/samples/psql_v1beta1_dbsystem.yaml",
		"config/samples/queue_v1beta1_queue.yaml",
		"config/samples/redis_v1beta1_rediscluster.yaml",
		"config/samples/streaming_v1beta1_stream.yaml",
		"config/samples/vault_v1beta1_secret.yaml",
		"packages/containerengine/metadata.env",
		"packages/containerengine/install/kustomization.yaml",
		"packages/core/metadata.env",
		"packages/core/install/kustomization.yaml",
		"packages/core-network/metadata.env",
		"packages/core-network/install/kustomization.yaml",
		"packages/database/metadata.env",
		"packages/database/install/kustomization.yaml",
		"packages/dataflow/metadata.env",
		"packages/dataflow/install/kustomization.yaml",
		"packages/identity/metadata.env",
		"packages/identity/install/kustomization.yaml",
		"packages/mysql/metadata.env",
		"packages/mysql/install/kustomization.yaml",
		"packages/nosql/metadata.env",
		"packages/nosql/install/kustomization.yaml",
		"packages/opensearch/metadata.env",
		"packages/opensearch/install/kustomization.yaml",
		"packages/psql/metadata.env",
		"packages/psql/install/kustomization.yaml",
		"packages/queue/metadata.env",
		"packages/queue/install/kustomization.yaml",
		"packages/redis/metadata.env",
		"packages/redis/install/kustomization.yaml",
		"packages/streaming/metadata.env",
		"packages/streaming/install/kustomization.yaml",
		"packages/vault/metadata.env",
		"packages/vault/install/kustomization.yaml",
	}
	assertExactFileMatchesAll(t, repoRoot(t), outputRoot, exactFiles)

	runtimeFiles := []string{
		"controllers/containerengine/cluster_controller.go",
		"pkg/servicemanager/containerengine/cluster/cluster_serviceclient.go",
		"pkg/servicemanager/containerengine/cluster/cluster_servicemanager.go",
		"controllers/core/instance_controller.go",
		"controllers/core/vcn_controller.go",
		"controllers/database/autonomousdatabase_controller.go",
		"controllers/dataflow/application_controller.go",
		"controllers/identity/compartment_controller.go",
		"controllers/mysql/dbsystem_controller.go",
		"controllers/nosql/table_controller.go",
		"controllers/opensearch/opensearchcluster_controller.go",
		"controllers/psql/dbsystem_controller.go",
		"controllers/queue/queue_controller.go",
		"controllers/redis/rediscluster_controller.go",
		"controllers/streaming/stream_controller.go",
		"controllers/vault/secret_controller.go",
		"pkg/servicemanager/core/instance/instance_serviceclient.go",
		"pkg/servicemanager/core/instance/instance_servicemanager.go",
		"pkg/servicemanager/core/vcn/vcn_serviceclient.go",
		"pkg/servicemanager/core/vcn/vcn_servicemanager.go",
		"pkg/servicemanager/database/autonomousdatabase/autonomousdatabase_serviceclient.go",
		"pkg/servicemanager/database/autonomousdatabase/autonomousdatabase_servicemanager.go",
		"pkg/servicemanager/dataflow/application/application_serviceclient.go",
		"pkg/servicemanager/dataflow/application/application_servicemanager.go",
		"pkg/servicemanager/identity/compartment/compartment_serviceclient.go",
		"pkg/servicemanager/identity/compartment/compartment_servicemanager.go",
		"pkg/servicemanager/mysql/dbsystem/dbsystem_serviceclient.go",
		"pkg/servicemanager/mysql/dbsystem/dbsystem_servicemanager.go",
		"pkg/servicemanager/nosql/table/table_serviceclient.go",
		"pkg/servicemanager/nosql/table/table_servicemanager.go",
		"pkg/servicemanager/opensearch/opensearchcluster/opensearchcluster_serviceclient.go",
		"pkg/servicemanager/opensearch/opensearchcluster/opensearchcluster_servicemanager.go",
		"pkg/servicemanager/psql/dbsystem/dbsystem_serviceclient.go",
		"pkg/servicemanager/psql/dbsystem/dbsystem_servicemanager.go",
		"pkg/servicemanager/queue/queue/queue_serviceclient.go",
		"pkg/servicemanager/queue/queue/queue_servicemanager.go",
		"pkg/servicemanager/redis/rediscluster/rediscluster_serviceclient.go",
		"pkg/servicemanager/redis/rediscluster/rediscluster_servicemanager.go",
		"pkg/servicemanager/streaming/stream/stream_serviceclient.go",
		"pkg/servicemanager/streaming/stream/stream_servicemanager.go",
		"pkg/servicemanager/vault/secret/secret_serviceclient.go",
		"pkg/servicemanager/vault/secret/secret_servicemanager.go",
		"internal/registrations/containerengine_generated.go",
		"internal/registrations/core-network_generated.go",
		"internal/registrations/core_generated.go",
		"internal/registrations/database_generated.go",
		"internal/registrations/dataflow_generated.go",
		"internal/registrations/identity_generated.go",
		"internal/registrations/mysql_generated.go",
		"internal/registrations/nosql_generated.go",
		"internal/registrations/opensearch_generated.go",
		"internal/registrations/psql_generated.go",
		"internal/registrations/queue_generated.go",
		"internal/registrations/redis_generated.go",
		"internal/registrations/streaming_generated.go",
		"internal/registrations/vault_generated.go",
	}
	for _, relativePath := range runtimeFiles {
		assertGoEquivalent(t, filepath.Join(repoRoot(t), relativePath), filepath.Join(outputRoot, relativePath))
	}

	assertResourceOrderContainsSubset(
		t,
		filepath.Join(repoRoot(t), "config", "samples", "kustomization.yaml"),
		filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"),
	)
}

func TestExplicitCoreRuntimeArtifactsGenerateFromConfig(t *testing.T) {
	cfg := loadCheckedInConfig(t)
	coreService := serviceConfigsByName(t, cfg, "core")["core"]

	outputRoot := t.TempDir()
	seedSamplesKustomization(t, outputRoot)

	pipeline := New()
	result, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{*coreService}, Options{
		OutputRoot: outputRoot,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Generated) != 1 {
		t.Fatalf("Generate() generated %d services, want 1", len(result.Generated))
	}

	assertPathsExist(t, []string{
		filepath.Join(outputRoot, "api", "core", "v1beta1", "vcn_types.go"),
		filepath.Join(outputRoot, "controllers", "core", "vcn_controller.go"),
		filepath.Join(outputRoot, "pkg", "servicemanager", "core", "vcn", "vcn_serviceclient.go"),
		filepath.Join(outputRoot, "pkg", "servicemanager", "core", "vcn", "vcn_servicemanager.go"),
		filepath.Join(outputRoot, "internal", "registrations", "core_generated.go"),
		filepath.Join(outputRoot, "packages", "core", "metadata.env"),
		filepath.Join(outputRoot, "packages", "core", "install", "kustomization.yaml"),
		filepath.Join(outputRoot, "config", "samples", "core_v1beta1_vcn.yaml"),
	})
}

func TestCheckedInStreamingPreservesStreamNamePrinterColumn(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	streamingService := serviceConfigsByName(t, cfg, "streaming")["streaming"]

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, *streamingService)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	stream := findResource(t, pkg.Resources, "Stream")
	if len(stream.PrintColumns) == 0 {
		t.Fatal("Stream print columns = none, want Name column first")
	}

	nameColumn := stream.PrintColumns[0]
	if nameColumn.Name != "Name" {
		t.Fatalf("Stream first print column name = %q, want %q", nameColumn.Name, "Name")
	}
	if nameColumn.JSONPath != ".spec.name" {
		t.Fatalf("Stream first print column JSONPath = %q, want %q", nameColumn.JSONPath, ".spec.name")
	}
	if nameColumn.Type != "string" {
		t.Fatalf("Stream first print column type = %q, want %q", nameColumn.Type, "string")
	}
}

func TestExplicitIdentityUserRuntimeArtifactsGenerateFromConfig(t *testing.T) {
	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	var identityService *ServiceConfig
	for i := range cfg.Services {
		if cfg.Services[i].Service == "identity" {
			identityService = &cfg.Services[i]
			break
		}
	}
	if identityService == nil {
		t.Fatal("identity service was not found in services.yaml")
	}
	if got := identityService.FormalSpecFor("User"); got != "user" {
		t.Fatalf("identity User formalSpec = %q, want %q", got, "user")
	}

	outputRoot := t.TempDir()
	samplesDir := filepath.Join(outputRoot, "config", "samples")
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", samplesDir, err)
	}
	checkedInSampleKustomization := readFile(t, filepath.Join(repoRoot(t), "config", "samples", "kustomization.yaml"))
	if err := os.WriteFile(filepath.Join(samplesDir, "kustomization.yaml"), []byte(checkedInSampleKustomization), 0o644); err != nil {
		t.Fatalf("write seeded samples kustomization: %v", err)
	}

	pipeline := New()
	result, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{*identityService}, Options{
		OutputRoot: outputRoot,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Generated) != 1 {
		t.Fatalf("Generate() generated %d services, want 1", len(result.Generated))
	}

	serviceClientPath := "pkg/servicemanager/identity/user/user_serviceclient.go"
	serviceManagerPath := "pkg/servicemanager/identity/user/user_servicemanager.go"
	assertPathsExist(t, []string{
		filepath.Join(outputRoot, "api", "identity", "v1beta1", "user_types.go"),
		filepath.Join(outputRoot, "controllers", "identity", "user_controller.go"),
		filepath.Join(outputRoot, serviceClientPath),
		filepath.Join(outputRoot, serviceManagerPath),
		filepath.Join(outputRoot, "internal", "registrations", "identity_generated.go"),
		filepath.Join(outputRoot, "packages", "identity", "metadata.env"),
		filepath.Join(outputRoot, "packages", "identity", "install", "kustomization.yaml"),
		filepath.Join(outputRoot, "config", "samples", "identity_v1beta1_user.yaml"),
	})

	content := readFile(t, filepath.Join(outputRoot, serviceClientPath))
	assertContains(t, content, []string{
		"Semantics: &generatedruntime.Semantics{",
		`FormalService:     "identity"`,
		`FormalSlug:        "user"`,
		`ResponseItemsField: "Items"`,
		`CreateFollowUp: generatedruntime.FollowUpSemantics{`,
		`Strategy: "read-after-write"`,
		`DeleteFollowUp: generatedruntime.FollowUpSemantics{`,
		`Strategy: "confirm-delete"`,
		`Fields: []generatedruntime.RequestField{{FieldName: "CreateUserDetails", RequestName: "CreateUserDetails", Contribution: "body", PreferResourceID: false}},`,
		`Fields: []generatedruntime.RequestField{{FieldName: "UserId", RequestName: "userId", Contribution: "path", PreferResourceID: true}},`,
	})
}

func TestCheckedInPromotedRuntimeArtifactsMatchGenerator(t *testing.T) {
	cfg := loadCheckedInConfig(t)
	services := serviceConfigsByName(t, cfg, "database", "mysql", "streaming")

	tests := []promotedRuntimeArtifactExpectation{
		{
			serviceName:       "database",
			kind:              "AutonomousDatabase",
			formalSlug:        "databaseautonomousdatabase",
			serviceClientPath: "pkg/servicemanager/database/autonomousdatabase/autonomousdatabase_serviceclient.go",
			controllerPath:    "controllers/database/autonomousdatabase_controller.go",
			controllerContains: []string{
				`// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch`,
				`// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch`,
			},
			serviceClientChecks: []string{
				"Semantics: &generatedruntime.Semantics{",
				`FormalService:     "database"`,
				`FormalSlug:        "databaseautonomousdatabase"`,
				`SecretSideEffects: "none"`,
				`StatusProjection:  "required"`,
				`CredentialClient: manager.CredentialClient,`,
			},
		},
		{
			serviceName:       "mysql",
			kind:              "DbSystem",
			formalSlug:        "dbsystem",
			serviceClientPath: "pkg/servicemanager/mysql/dbsystem/dbsystem_serviceclient.go",
			controllerPath:    "controllers/mysql/dbsystem_controller.go",
			controllerContains: []string{
				`// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch`,
			},
			serviceClientChecks: []string{
				"Semantics: &generatedruntime.Semantics{",
				`FormalService:     "mysql"`,
				`FormalSlug:        "dbsystem"`,
				`SecretSideEffects: "none"`,
				`StatusProjection:  "required"`,
				`CredentialClient: manager.CredentialClient,`,
			},
		},
		{
			serviceName:       "streaming",
			kind:              "Stream",
			formalSlug:        "stream",
			serviceClientPath: "pkg/servicemanager/streaming/stream/stream_serviceclient.go",
			controllerPath:    "controllers/streaming/stream_controller.go",
			controllerContains: []string{
				`// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;delete`,
			},
			serviceClientChecks: []string{
				"Semantics: &generatedruntime.Semantics{",
				`FormalService:     "streaming"`,
				`FormalSlug:        "stream"`,
				`SecretSideEffects: "ready-only"`,
				`StatusProjection:  "required"`,
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.serviceName, func(t *testing.T) {
			service := services[test.serviceName]
			assertServiceFormalSpec(t, service, test.kind, test.formalSlug)
			assertPromotedRuntimeArtifactsCase(t, cfg, service, test)
		})
	}
}

func TestCheckedInDatabaseAutonomousDatabaseUsesSecretBackedAdminPassword(t *testing.T) {
	cfg := loadCheckedInConfig(t)
	databaseService := serviceConfigsByName(t, cfg, "database")["database"]

	outputRoot := t.TempDir()
	seedSamplesKustomization(t, outputRoot)

	pipeline := New()
	result, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{*databaseService}, Options{
		OutputRoot: outputRoot,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Generated) != 1 {
		t.Fatalf("Generate() generated %d services, want 1", len(result.Generated))
	}

	apiContent := readFile(t, filepath.Join(outputRoot, "api", "database", "v1beta1", "autonomousdatabase_types.go"))
	assertContains(t, apiContent, []string{
		`AdminPassword shared.PasswordSource ` + "`json:\"adminPassword,omitempty,omitzero\"`",
		"// The administrative password sourced from a Kubernetes Secret in the same namespace.\n\t// The referenced Secret must contain a `password` key. Use `secretId` and `secretVersionNumber` instead to reference an OCI Vault secret.\n\t// +kubebuilder:validation:Optional\n\tAdminPassword shared.PasswordSource `json:\"adminPassword,omitempty,omitzero\"`",
	})
	assertNotContains(t, apiContent, []string{
		`AdminPassword string ` + "`json:\"adminPassword,omitempty\"`",
	})

	serviceClientContent := readFile(t, filepath.Join(outputRoot, "pkg", "servicemanager", "database", "autonomousdatabase", "autonomousdatabase_serviceclient.go"))
	assertContains(t, serviceClientContent, []string{
		`CredentialClient: manager.CredentialClient,`,
	})
}

func TestCheckedInMySQLDbSystemUsesSecretBackedAdminCredentials(t *testing.T) {
	cfg := loadCheckedInConfig(t)
	mysqlService := serviceConfigsByName(t, cfg, "mysql")["mysql"]

	outputRoot := t.TempDir()
	seedSamplesKustomization(t, outputRoot)

	pipeline := New()
	result, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{*mysqlService}, Options{
		OutputRoot: outputRoot,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Generated) != 1 {
		t.Fatalf("Generate() generated %d services, want 1", len(result.Generated))
	}

	apiContent := readFile(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "dbsystem_types.go"))
	assertContains(t, apiContent, []string{
		`AdminUsername shared.UsernameSource ` + "`json:\"adminUsername,omitempty,omitzero\"`",
		`AdminPassword shared.PasswordSource ` + "`json:\"adminPassword,omitempty,omitzero\"`",
		"// The username for the administrative user sourced from a Kubernetes Secret in the same namespace.\n\t// The referenced Secret must contain a `username` key.\n\t// +kubebuilder:validation:Optional\n\tAdminUsername shared.UsernameSource `json:\"adminUsername,omitempty,omitzero\"`",
		"// The password for the administrative user sourced from a Kubernetes Secret in the same namespace.\n\t// The referenced Secret must contain a `password` key.\n\t// +kubebuilder:validation:Optional\n\tAdminPassword shared.PasswordSource `json:\"adminPassword,omitempty,omitzero\"`",
		"// The last applied secret reference for the administrative username.\n\tAdminUsername shared.UsernameSource `json:\"adminUsername,omitempty,omitzero\"`",
		"// The last applied secret reference for the administrative password.\n\tAdminPassword shared.PasswordSource `json:\"adminPassword,omitempty,omitzero\"`",
	})
	assertNotContains(t, apiContent, []string{
		`AdminUsername string ` + "`json:\"adminUsername,omitempty\"`",
		`AdminPassword string ` + "`json:\"adminPassword,omitempty\"`",
	})

	sampleContent := readFile(t, filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_dbsystem.yaml"))
	assertContains(t, sampleContent, []string{
		"adminUsername:",
		"adminPassword:",
		"secretName: admin-secret",
	})

	serviceClientContent := readFile(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "dbsystem", "dbsystem_serviceclient.go"))
	assertContains(t, serviceClientContent, []string{
		`CredentialClient: manager.CredentialClient,`,
	})
}

func TestCheckedInPSQLDbSystemUsesSecretBackedAdminCredentials(t *testing.T) {
	cfg := loadCheckedInConfig(t)
	psqlService := serviceConfigsByName(t, cfg, "psql")["psql"]

	outputRoot := t.TempDir()
	seedSamplesKustomization(t, outputRoot)

	pipeline := New()
	result, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{*psqlService}, Options{
		OutputRoot: outputRoot,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Generated) != 1 {
		t.Fatalf("Generate() generated %d services, want 1", len(result.Generated))
	}

	apiContent := readFile(t, filepath.Join(outputRoot, "api", "psql", "v1beta1", "dbsystem_types.go"))
	assertContains(t, apiContent, []string{
		`AdminUsername shared.UsernameSource ` + "`json:\"adminUsername,omitempty,omitzero\"`",
		`AdminPassword shared.PasswordSource ` + "`json:\"adminPassword,omitempty,omitzero\"`",
		`AdminUsernameSource shared.UsernameSource ` + "`json:\"adminUsernameSource,omitempty,omitzero\"`",
		`AdminPasswordSource shared.PasswordSource ` + "`json:\"adminPasswordSource,omitempty,omitzero\"`",
		"// The administrative username sourced from a Kubernetes Secret in the same namespace.\n\t// The referenced Secret must contain a `username` key. If omitted, `spec.credentials.username` remains available for direct credential input.\n\t// +kubebuilder:validation:Optional\n\tAdminUsername shared.UsernameSource `json:\"adminUsername,omitempty,omitzero\"`",
		"// The administrative password sourced from a Kubernetes Secret in the same namespace.\n\t// The referenced Secret must contain a `password` key. If omitted, `spec.credentials.passwordDetails` remains available for plaintext or OCI Vault secret input.\n\t// +kubebuilder:validation:Optional\n\tAdminPassword shared.PasswordSource `json:\"adminPassword,omitempty,omitzero\"`",
		"// The last applied secret reference for the administrative username.\n\tAdminUsernameSource shared.UsernameSource `json:\"adminUsernameSource,omitempty,omitzero\"`",
		"// The last applied secret reference for the administrative password.\n\tAdminPasswordSource shared.PasswordSource `json:\"adminPasswordSource,omitempty,omitzero\"`",
		`AdminUsername string ` + "`json:\"adminUsername,omitempty\"`",
	})
	assertNotContains(t, apiContent, []string{
		`AdminPassword string ` + "`json:\"adminPassword,omitempty\"`",
	})

	sampleContent := readFile(t, filepath.Join(outputRoot, "config", "samples", "psql_v1beta1_dbsystem.yaml"))
	assertContains(t, sampleContent, []string{
		"adminUsername:",
		"adminPassword:",
		"secretName: admin-secret",
	})
	assertNotContains(t, sampleContent, []string{
		"credentials:",
	})

	serviceClientContent := readFile(t, filepath.Join(outputRoot, "pkg", "servicemanager", "psql", "dbsystem", "dbsystem_serviceclient.go"))
	assertContains(t, serviceClientContent, []string{
		`CredentialClient: manager.CredentialClient,`,
	})
}

func TestCheckedInConfigIncludesNetworkLoadBalancerObservedStateAlias(t *testing.T) {
	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	var networkLoadBalancerService *ServiceConfig
	for i := range cfg.Services {
		if cfg.Services[i].Service == "networkloadbalancer" {
			networkLoadBalancerService = &cfg.Services[i]
			break
		}
	}
	if networkLoadBalancerService == nil {
		t.Fatal("networkloadbalancer service was not found in services.yaml")
	}

	if !slices.Equal(networkLoadBalancerService.ObservedState.SDKAliases["WorkRequestLog"], []string{"WorkRequestLogEntry"}) {
		t.Fatalf("networkloadbalancer WorkRequestLog aliases = %v, want WorkRequestLogEntry", networkLoadBalancerService.ObservedState.SDKAliases["WorkRequestLog"])
	}
}

func TestCheckedInConfigIncludesNoSQLObservedStateAlias(t *testing.T) {
	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	var nosqlService *ServiceConfig
	for i := range cfg.Services {
		if cfg.Services[i].Service == "nosql" {
			nosqlService = &cfg.Services[i]
			break
		}
	}
	if nosqlService == nil {
		t.Fatal("nosql service was not found in services.yaml")
	}

	if !slices.Equal(nosqlService.ObservedState.SDKAliases["WorkRequestLog"], []string{"WorkRequestLogEntry"}) {
		t.Fatalf("nosql WorkRequestLog aliases = %v, want WorkRequestLogEntry", nosqlService.ObservedState.SDKAliases["WorkRequestLog"])
	}
}

func TestCheckedInConfigIncludesObjectStorageObservedStateAlias(t *testing.T) {
	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	var objectStorageService *ServiceConfig
	for i := range cfg.Services {
		if cfg.Services[i].Service == "objectstorage" {
			objectStorageService = &cfg.Services[i]
			break
		}
	}
	if objectStorageService == nil {
		t.Fatal("objectstorage service was not found in services.yaml")
	}

	if !slices.Equal(objectStorageService.ObservedState.SDKAliases["WorkRequestLog"], []string{"WorkRequestLogEntry"}) {
		t.Fatalf("objectstorage WorkRequestLog aliases = %v, want WorkRequestLogEntry", objectStorageService.ObservedState.SDKAliases["WorkRequestLog"])
	}
}

func TestCheckedInConfigIncludesQueueObservedStateAlias(t *testing.T) {
	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	var queueService *ServiceConfig
	for i := range cfg.Services {
		if cfg.Services[i].Service == "queue" {
			queueService = &cfg.Services[i]
			break
		}
	}
	if queueService == nil {
		t.Fatal("queue service was not found in services.yaml")
	}

	if !slices.Equal(queueService.ObservedState.SDKAliases["WorkRequestLog"], []string{"WorkRequestLogEntry"}) {
		t.Fatalf("queue WorkRequestLog aliases = %v, want WorkRequestLogEntry", queueService.ObservedState.SDKAliases["WorkRequestLog"])
	}
}

func TestCheckedInConfigIncludesSecretsObservedStateAlias(t *testing.T) {
	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	var secretsService *ServiceConfig
	for i := range cfg.Services {
		if cfg.Services[i].Service == "secrets" {
			secretsService = &cfg.Services[i]
			break
		}
	}
	if secretsService == nil {
		t.Fatal("secrets service was not found in services.yaml")
	}

	if !slices.Equal(secretsService.ObservedState.SDKAliases["SecretBundleByName"], []string{"SecretBundle"}) {
		t.Fatalf("secrets SecretBundleByName aliases = %v, want SecretBundle", secretsService.ObservedState.SDKAliases["SecretBundleByName"])
	}
}

func TestCheckedInConfigIncludesONSObservedStateAlias(t *testing.T) {
	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	var onsService *ServiceConfig
	for i := range cfg.Services {
		if cfg.Services[i].Service == "ons" {
			onsService = &cfg.Services[i]
			break
		}
	}
	if onsService == nil {
		t.Fatal("ons service was not found in services.yaml")
	}

	if !slices.Equal(onsService.ObservedState.SDKAliases["ConfirmSubscription"], []string{"ConfirmationResult"}) {
		t.Fatalf("ons ConfirmSubscription aliases = %v, want ConfirmationResult", onsService.ObservedState.SDKAliases["ConfirmSubscription"])
	}
}

func TestMySQLDbSystemIncludesOptionalDesiredStateFields(t *testing.T) {
	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	var mysqlService *ServiceConfig
	for i := range cfg.Services {
		if cfg.Services[i].Service == "mysql" {
			mysqlService = &cfg.Services[i]
			break
		}
	}
	if mysqlService == nil {
		t.Fatal("mysql service was not found in services.yaml")
	}

	outputRoot := t.TempDir()
	pipeline := New()
	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{*mysqlService}, Options{
		OutputRoot: outputRoot,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	content := readFile(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "dbsystem_types.go"))
	normalized := strings.Join(strings.Fields(content), " ")
	assertContains(t, normalized, []string{
		"type DbSystemDeletionPolicy struct {",
		"AutomaticBackupRetention string `json:\"automaticBackupRetention,omitempty\"`",
		"FinalBackup string `json:\"finalBackup,omitempty\"`",
		"IsDeleteProtected bool `json:\"isDeleteProtected,omitempty\"`",
		"type DbSystemSecureConnections struct {",
		"CertificateGenerationType string `json:\"certificateGenerationType\"`",
		"CertificateId string `json:\"certificateId,omitempty\"`",
		"DeletionPolicy DbSystemDeletionPolicy `json:\"deletionPolicy,omitempty\"`",
		"CrashRecovery string `json:\"crashRecovery,omitempty\"`",
		"DatabaseManagement string `json:\"databaseManagement,omitempty\"`",
		"SecureConnections DbSystemSecureConnections `json:\"secureConnections,omitempty\"`",
	})
}

func TestCheckedInRedisClusterFormalBindingMatchesDiscovery(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	var redisService *ServiceConfig
	for i := range cfg.Services {
		if cfg.Services[i].Service == "redis" {
			redisService = &cfg.Services[i]
			break
		}
	}
	if redisService == nil {
		t.Fatal("redis service was not found in services.yaml")
	}
	if got := redisService.FormalSpecFor("RedisCluster"); got != "rediscluster" {
		t.Fatalf("redis RedisCluster formalSpec = %q, want %q", got, "rediscluster")
	}
	if got := redisService.FormalSpecFor("RedisRedisCluster"); got != "" {
		t.Fatalf("redis RedisRedisCluster formalSpec = %q, want empty", got)
	}

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, *redisService)
	if err != nil {
		if strings.Contains(err.Error(), "github.com/oracle/oci-go-sdk/v65/redis") &&
			strings.Contains(err.Error(), "cannot find module providing package") {
			t.Skip("redis SDK is not vendored in this checkout yet")
		}
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	redisCluster := findResource(t, pkg.Resources, "RedisCluster")
	if redisCluster.SDKName != "RedisCluster" {
		t.Fatalf("RedisCluster SDK name = %q, want %q", redisCluster.SDKName, "RedisCluster")
	}
	if redisCluster.Formal == nil {
		t.Fatal("RedisCluster formal model was not attached")
	}
	if redisCluster.Formal.Reference.Service != "redis" {
		t.Fatalf("RedisCluster formal service = %q, want %q", redisCluster.Formal.Reference.Service, "redis")
	}
	if redisCluster.Formal.Reference.Slug != "rediscluster" {
		t.Fatalf("RedisCluster formal slug = %q, want %q", redisCluster.Formal.Reference.Slug, "rediscluster")
	}
	if redisCluster.Formal.Binding.Spec.Kind != "RedisCluster" {
		t.Fatalf("RedisCluster formal kind = %q, want %q", redisCluster.Formal.Binding.Spec.Kind, "RedisCluster")
	}
	if redisCluster.Formal.Binding.Import.ProviderResource != "oci_redis_redis_cluster" {
		t.Fatalf("RedisCluster provider resource = %q, want %q", redisCluster.Formal.Binding.Import.ProviderResource, "oci_redis_redis_cluster")
	}
	if redisCluster.Runtime == nil {
		t.Fatal("RedisCluster runtime model was not attached")
	}
	if redisCluster.Runtime.Create == nil || redisCluster.Runtime.Create.MethodName != "CreateRedisCluster" {
		t.Fatalf("RedisCluster create method = %#v, want CreateRedisCluster", redisCluster.Runtime.Create)
	}
	if redisCluster.Runtime.Get == nil || redisCluster.Runtime.Get.MethodName != "GetRedisCluster" {
		t.Fatalf("RedisCluster get method = %#v, want GetRedisCluster", redisCluster.Runtime.Get)
	}
	if redisCluster.Runtime.List == nil || redisCluster.Runtime.List.MethodName != "ListRedisClusters" {
		t.Fatalf("RedisCluster list method = %#v, want ListRedisClusters", redisCluster.Runtime.List)
	}
	if redisCluster.Runtime.Update == nil || redisCluster.Runtime.Update.MethodName != "UpdateRedisCluster" {
		t.Fatalf("RedisCluster update method = %#v, want UpdateRedisCluster", redisCluster.Runtime.Update)
	}
	if redisCluster.Runtime.Delete == nil || redisCluster.Runtime.Delete.MethodName != "DeleteRedisCluster" {
		t.Fatalf("RedisCluster delete method = %#v, want DeleteRedisCluster", redisCluster.Runtime.Delete)
	}
	if redisCluster.Runtime.Semantics == nil {
		t.Fatal("RedisCluster runtime semantics were not attached")
	}
	if got := redisCluster.Runtime.Semantics.List; got == nil {
		t.Fatal("RedisCluster list semantics were not attached")
	} else if !slices.Equal(got.MatchFields, []string{"compartmentId", "displayName"}) {
		t.Fatalf("RedisCluster list match fields = %v, want [compartmentId displayName]", got.MatchFields)
	}
	if got := redisCluster.Runtime.Semantics.Mutation.Mutable; !slices.Equal(got, []string{"definedTags", "displayName", "freeformTags", "nodeCount", "nodeMemoryInGbs"}) {
		t.Fatalf("RedisCluster mutable fields = %v, want reviewed mutable surface", got)
	}
	if got := redisCluster.Runtime.Semantics.Mutation.ForceNew; !slices.Equal(got, []string{"compartmentId", "softwareVersion", "subnetId"}) {
		t.Fatalf("RedisCluster force-new fields = %v, want [compartmentId softwareVersion subnetId]", got)
	}
	if len(redisCluster.Runtime.Semantics.AuxiliaryOperations) != 0 {
		t.Fatalf("RedisCluster auxiliary operations = %v, want none", redisCluster.Runtime.Semantics.AuxiliaryOperations)
	}
	if len(redisCluster.Runtime.Semantics.OpenGaps) != 0 {
		t.Fatalf("RedisCluster open gaps = %v, want none", redisCluster.Runtime.Semantics.OpenGaps)
	}

	for _, resource := range pkg.Resources {
		if resource.Kind == "RedisRedisCluster" {
			t.Fatal("discovered RedisRedisCluster resource kind, want published RedisCluster only")
		}
	}
}

func TestGeneratePreservesExistingSampleKustomizationEntries(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileCRDOnly)
	pipeline := newTestGenerator(t)

	outputRoot := t.TempDir()
	samplesDir := filepath.Join(outputRoot, "config", "samples")
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", samplesDir, err)
	}
	if err := os.WriteFile(filepath.Join(samplesDir, "existing.yaml"), []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: existing\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing.yaml) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(samplesDir, "kustomization.yaml"), []byte("resources:\n- existing.yaml\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(kustomization.yaml) error = %v", err)
	}

	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	sampleKustomization := readFile(t, filepath.Join(samplesDir, "kustomization.yaml"))
	assertContains(t, sampleKustomization, []string{
		"- existing.yaml",
		"- mysql_v1beta1_dbsystem.yaml",
	})
	if strings.Index(sampleKustomization, "- existing.yaml") > strings.Index(sampleKustomization, "- mysql_v1beta1_dbsystem.yaml") {
		t.Fatalf("existing sample entry was not preserved ahead of the generated sample:\n%s", sampleKustomization)
	}
}

func TestGenerateIncrementalSampleKustomizationKeepsExistingGeneratedServices(t *testing.T) {
	t.Parallel()

	databaseService := testServiceConfig(PackageProfileCRDOnly)
	databaseService.Service = "database"
	databaseService.Group = "database"
	databaseService.SampleOrder = 10

	mysqlService := testServiceConfig(PackageProfileCRDOnly)
	mysqlService.SampleOrder = 20

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		Services:       []ServiceConfig{databaseService, mysqlService},
	}
	pipeline := newTestGenerator(t)
	outputRoot := t.TempDir()

	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{databaseService}, Options{
		OutputRoot: outputRoot,
	}); err != nil {
		t.Fatalf("Generate(database) error = %v", err)
	}

	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{mysqlService}, Options{
		OutputRoot: outputRoot,
	}); err != nil {
		t.Fatalf("Generate(mysql) error = %v", err)
	}

	order, err := readSampleKustomizationOrder(filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"))
	if err != nil {
		t.Fatalf("readSampleKustomizationOrder(kustomization.yaml) error = %v", err)
	}

	want := []string{
		"database_v1beta1_dbsystem.yaml",
		"database_v1beta1_oauthclientcredential.yaml",
		"database_v1beta1_report.yaml",
		"database_v1beta1_reportbyname.yaml",
		"database_v1beta1_widget.yaml",
		"mysql_v1beta1_dbsystem.yaml",
		"mysql_v1beta1_oauthclientcredential.yaml",
		"mysql_v1beta1_report.yaml",
		"mysql_v1beta1_reportbyname.yaml",
		"mysql_v1beta1_widget.yaml",
	}
	if !slices.Equal(order, want) {
		t.Fatalf("sample kustomization resources = %#v, want %#v", order, want)
	}
}

func sampleSDKDir(t *testing.T) string {
	t.Helper()

	return filepath.Join(generatorTestDir(t), "testdata", "sdk", "sample")
}

func assertDiscoveredMySQLDbSystem(t *testing.T, dbSystem ResourceModel) {
	t.Helper()

	if dbSystem.SDKName != "DbSystem" {
		t.Fatalf("MySqlDbSystem SDK name = %q, want %q", dbSystem.SDKName, "DbSystem")
	}
	assertResourceSpecFields(t, dbSystem, "Port")
	assertResourceSpecFieldsAbsent(t, dbSystem, "Id")
	if dbSystem.PrimaryDisplayField != "DisplayName" {
		t.Fatalf("MySqlDbSystem primary display field = %q, want DisplayName", dbSystem.PrimaryDisplayField)
	}
}

func assertDiscoveredWidget(t *testing.T, widget ResourceModel) {
	t.Helper()

	if len(widget.Operations) != 5 {
		t.Fatalf("Widget operations = %v, want 5 CRUD operations", widget.Operations)
	}
	assertResourceSpecFields(t, widget, "Mode", "CreatedAt")
	assertResourceSpecFieldsAbsent(t, widget, "LifecycleState", "TimeUpdated")
	assertResourceStatusFields(t, widget, "LifecycleState", "TimeUpdated")

	compartmentID := findFieldModel(t, widget.SpecFields, "CompartmentId")
	assertFieldTag(t, "Widget CompartmentId", compartmentID, `json:"compartmentId"`)
	assertFieldCommentsEqual(t, "Widget CompartmentId", compartmentID, []string{"The OCID of the widget compartment."})
	assertFieldMarkers(t, "Widget CompartmentId", compartmentID, []string{"+kubebuilder:validation:Required"})

	labels := findFieldModel(t, widget.SpecFields, "Labels")
	assertFieldTag(t, "Widget Labels", labels, `json:"labels,omitempty"`)
	assertFieldCommentsEqual(t, "Widget Labels", labels, []string{"Additional labels for the widget."})
	assertFieldMarkers(t, "Widget Labels", labels, []string{"+kubebuilder:validation:Optional"})

	serverState := findFieldModel(t, widget.SpecFields, "ServerState")
	assertFieldTag(t, "Widget ServerState", serverState, `json:"serverState,omitempty"`)
	assertFieldHasNoMarkers(t, "Widget ServerState", serverState)

	lifecycleState := findFieldModel(t, widget.StatusFields, "LifecycleState")
	assertFieldCommentsEqual(t, "Widget LifecycleState", lifecycleState, []string{"The lifecycle state of the widget."})
	assertFieldHasNoMarkers(t, "Widget LifecycleState", lifecycleState)
}

func assertDiscoveredReport(t *testing.T, report ResourceModel) {
	t.Helper()

	if len(report.SpecFields) != 0 {
		t.Fatalf("Report spec fields = %#v, want empty spec when no create or update payload exists", report.SpecFields)
	}
	assertResourceStatusFields(t, report, "Id", "LifecycleState", "DisplayName")
}

func assertDiscoveredReportByName(t *testing.T, report ResourceModel) {
	t.Helper()

	assertResourceSpecFields(t, report, "DisplayName")
}

func assertDiscoveredOAuthClientCredential(t *testing.T, credential ResourceModel) {
	t.Helper()

	assertResourceSpecFields(t, credential, "Name", "Description", "Scopes")
}

func assertWidgetFormalModel(t *testing.T, widget ResourceModel) {
	t.Helper()

	if widget.Formal == nil {
		t.Fatal("Widget formal model was not attached")
	}
	if widget.Formal.Reference.Service != "mysql" {
		t.Fatalf("Widget formal service = %q, want %q", widget.Formal.Reference.Service, "mysql")
	}
	if widget.Formal.Reference.Slug != "widget" {
		t.Fatalf("Widget formal slug = %q, want %q", widget.Formal.Reference.Slug, "widget")
	}
	if widget.Formal.Binding.Import.ProviderResource != "widget_resource" {
		t.Fatalf("Widget provider resource = %q, want %q", widget.Formal.Binding.Import.ProviderResource, "widget_resource")
	}
	if widget.Formal.Binding.Spec.Kind != "Widget" {
		t.Fatalf("Widget formal kind = %q, want %q", widget.Formal.Binding.Spec.Kind, "Widget")
	}
	if widget.Formal.Diagrams.ActivitySourcePath != "controllers/mysql/widget/diagrams/activity.puml" {
		t.Fatalf("Widget activity diagram path = %q, want %q", widget.Formal.Diagrams.ActivitySourcePath, "controllers/mysql/widget/diagrams/activity.puml")
	}
}

func assertWidgetServiceManagerFormalModel(t *testing.T, serviceManager ServiceManagerModel) {
	t.Helper()

	if serviceManager.Formal == nil {
		t.Fatal("Widget service manager formal model was not attached")
	}
	if serviceManager.Formal.Reference.Slug != "widget" {
		t.Fatalf("Widget service manager formal slug = %q, want %q", serviceManager.Formal.Reference.Slug, "widget")
	}
}

func assertWidgetRuntimeSemantics(t *testing.T, widget ResourceModel) {
	t.Helper()

	if widget.Runtime == nil || widget.Runtime.Semantics == nil {
		t.Fatal("Widget runtime semantics were not attached")
	}

	semantics := widget.Runtime.Semantics
	assertWidgetLifecycleSemantics(t, semantics)
	assertWidgetListAndMutationSemantics(t, semantics)
}

func assertWidgetLifecycleSemantics(t *testing.T, semantics *RuntimeSemanticsModel) {
	t.Helper()

	if !slices.Equal(semantics.Lifecycle.ProvisioningStates, []string{"PROVISIONING"}) {
		t.Fatalf("Widget provisioning states = %v, want [PROVISIONING]", semantics.Lifecycle.ProvisioningStates)
	}
	if !slices.Equal(semantics.Lifecycle.ActiveStates, []string{"ACTIVE"}) {
		t.Fatalf("Widget active states = %v, want [ACTIVE]", semantics.Lifecycle.ActiveStates)
	}
	if semantics.Delete.Policy != "required" {
		t.Fatalf("Widget delete policy = %q, want required", semantics.Delete.Policy)
	}
	if semantics.List == nil || semantics.List.ResponseItemsField != "Items" {
		t.Fatalf("Widget list semantics = %#v, want responseItemsField Items", semantics.List)
	}
}

func assertWidgetListAndMutationSemantics(t *testing.T, semantics *RuntimeSemanticsModel) {
	t.Helper()

	if !slices.Equal(semantics.List.MatchFields, []string{"compartmentId", "state"}) {
		t.Fatalf("Widget list match fields = %v, want [compartmentId state]", semantics.List.MatchFields)
	}
	if !slices.Equal(semantics.Mutation.ForceNew, []string{"compartmentId"}) {
		t.Fatalf("Widget forceNew = %v, want [compartmentId]", semantics.Mutation.ForceNew)
	}
	if semantics.CreateFollowUp.Strategy != followUpStrategyReadAfterWrite {
		t.Fatalf("Widget create follow-up = %q, want %q", semantics.CreateFollowUp.Strategy, followUpStrategyReadAfterWrite)
	}
	if len(semantics.OpenGaps) != 0 {
		t.Fatalf("Widget open gaps = %#v, want none", semantics.OpenGaps)
	}
}

func assertWidgetServiceManagerSemantics(t *testing.T, serviceManager ServiceManagerModel) {
	t.Helper()

	if serviceManager.Semantics == nil {
		t.Fatal("Widget service manager semantics were not attached")
	}
	if serviceManager.Semantics.FormalSlug != "widget" {
		t.Fatalf("Widget service manager formal slug = %q, want widget", serviceManager.Semantics.FormalSlug)
	}
}

func assertFunctionsSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	application := findResource(t, pkg.Resources, "Application")
	assertFieldType(t, "Application TraceConfig", findFieldModel(t, application.SpecFields, "TraceConfig"), "ApplicationTraceConfig")
	assertFieldType(t, "Application ImagePolicyConfig", findFieldModel(t, application.SpecFields, "ImagePolicyConfig"), "ApplicationImagePolicyConfig")
	assertFieldType(t, "Application DefinedTags", findFieldModel(t, application.SpecFields, "DefinedTags"), "map[string]shared.MapValue")
	assertHelperTypeFields(t, findHelperType(t, application.HelperTypes, "ApplicationTraceConfig"), "DomainId")
	assertHelperTypeFields(t, findHelperType(t, application.HelperTypes, "ApplicationImagePolicyConfig"), "IsPolicyEnabled")

	function := findResource(t, pkg.Resources, "Function")
	assertFieldType(t, "Function SourceDetails", findFieldModel(t, function.SpecFields, "SourceDetails"), "FunctionSourceDetails")
	assertHelperTypeFields(t, findHelperType(t, function.HelperTypes, "FunctionSourceDetails"), "SourceType", "PbfListingId")
	assertFieldType(t, "Function ProvisionedConcurrencyConfig", findFieldModel(t, function.SpecFields, "ProvisionedConcurrencyConfig"), "FunctionProvisionedConcurrencyConfig")
	assertHelperTypeFields(t, findHelperType(t, function.HelperTypes, "FunctionProvisionedConcurrencyConfig"), "Strategy", "Count")
}

func assertCoreSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	tunnel := findResource(t, pkg.Resources, "IPSecConnectionTunnel")
	assertFieldType(t, "IPSecConnectionTunnel BgpSessionConfig", findFieldModel(t, tunnel.SpecFields, "BgpSessionConfig"), "IPSecConnectionTunnelBgpSessionConfig")
	assertFieldType(t, "IPSecConnectionTunnel PhaseOneConfig", findFieldModel(t, tunnel.SpecFields, "PhaseOneConfig"), "IPSecConnectionTunnelPhaseOneConfig")
	assertFieldType(t, "IPSecConnectionTunnel PhaseTwoConfig", findFieldModel(t, tunnel.SpecFields, "PhaseTwoConfig"), "IPSecConnectionTunnelPhaseTwoConfig")
	assertHelperTypeFields(t, findHelperType(t, tunnel.HelperTypes, "IPSecConnectionTunnelBgpSessionConfig"), "CustomerBgpAsn")
	assertHelperTypeFields(t, findHelperType(t, tunnel.HelperTypes, "IPSecConnectionTunnelPhaseOneConfig"), "DiffieHelmanGroup")
}

func assertCertificatesSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	bundle := findResource(t, pkg.Resources, "CertificateBundle")
	assertFieldType(t, "CertificateBundle Validity", findFieldModel(t, bundle.StatusFields, "Validity"), "CertificateBundleValidity")
	assertFieldType(t, "CertificateBundle RevocationStatus", findFieldModel(t, bundle.StatusFields, "RevocationStatus"), "CertificateBundleRevocationStatus")
	assertHelperTypeFields(t, findHelperType(t, bundle.HelperTypes, "CertificateBundleValidity"), "TimeOfValidityNotBefore")
	assertHelperTypeFields(t, findHelperType(t, bundle.HelperTypes, "CertificateBundleRevocationStatus"), "RevocationReason")
}

func assertNoSQLSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	row := findResource(t, pkg.Resources, "Row")
	assertFieldType(t, "Row Value", findFieldModel(t, row.SpecFields, "Value"), "map[string]shared.JSONValue")
}

func assertSecretsSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	bundle := findResource(t, pkg.Resources, "SecretBundle")
	assertFieldType(t, "SecretBundle SecretBundleContent", findFieldModel(t, bundle.StatusFields, "SecretBundleContent"), "SecretBundleContent")
	assertHelperTypeFields(t, findHelperType(t, bundle.HelperTypes, "SecretBundleContent"), "ContentType", "Content")
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "SecretBundleByName"), "SecretId", "VersionNumber", "SecretBundleContent", "Metadata")
}

func assertVaultSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	secret := findResource(t, pkg.Resources, "Secret")
	assertFieldType(t, "Secret Metadata", findFieldModel(t, secret.SpecFields, "Metadata"), "map[string]shared.JSONValue")
}

func assertArtifactsSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	assertPackageResourceStatusFields(t, pkg, map[string][]string{
		"ContainerConfiguration":  {"IsRepositoryCreatedOnFirstPush"},
		"ContainerImage":          {"FreeformTags"},
		"ContainerImageSignature": {"CompartmentId", "ImageId", "Message", "Signature", "SigningAlgorithm"},
		"ContainerRepository":     {"CompartmentId", "DisplayName", "IsImmutable", "IsPublic", "FreeformTags", "DefinedTags"},
		"GenericArtifact":         {"FreeformTags"},
		"Repository":              {"DisplayName", "Description", "CompartmentId", "IsImmutable", "FreeformTags", "DefinedTags"},
	})

	containerImage := findResource(t, pkg.Resources, "ContainerImage")
	assertFieldType(t, "ContainerImage DefinedTags", findFieldModel(t, containerImage.StatusFields, "DefinedTags"), "map[string]shared.MapValue")

	containerRepository := findResource(t, pkg.Resources, "ContainerRepository")
	assertFieldType(t, "ContainerRepository Readme", findFieldModel(t, containerRepository.StatusFields, "Readme"), "ContainerRepositoryReadme")
}

func assertNetworkLoadBalancerSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	healthChecker := findResource(t, pkg.Resources, "HealthChecker")
	assertFieldType(t, "HealthChecker RequestData", findFieldModel(t, healthChecker.SpecFields, "RequestData"), "string")
	assertFieldType(t, "HealthChecker ResponseData", findFieldModel(t, healthChecker.SpecFields, "ResponseData"), "string")
}

type promotedRuntimeArtifactExpectation struct {
	serviceName         string
	kind                string
	formalSlug          string
	serviceClientPath   string
	controllerPath      string
	controllerContains  []string
	controllerExcludes  []string
	serviceClientChecks []string
}

func assertPromotedRuntimeArtifactsCase(t *testing.T, cfg *Config, service *ServiceConfig, want promotedRuntimeArtifactExpectation) {
	t.Helper()

	outputRoot := t.TempDir()
	seedSamplesKustomization(t, outputRoot)

	pipeline := New()
	result, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{*service}, Options{
		OutputRoot: outputRoot,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Generated) != 1 {
		t.Fatalf("Generate() generated %d services, want 1", len(result.Generated))
	}

	assertGeneratedGoMatchesAll(t, repoRoot(t), outputRoot, []string{
		want.serviceClientPath,
		want.controllerPath,
	})
	assertFileContains(t, filepath.Join(outputRoot, want.serviceClientPath), want.serviceClientChecks)
	assertFileContains(t, filepath.Join(outputRoot, want.controllerPath), want.controllerContains)
	assertFileDoesNotContain(t, filepath.Join(outputRoot, want.controllerPath), want.controllerExcludes)
}

func newTestGenerator(t *testing.T) *Generator {
	t.Helper()

	return &Generator{
		discoverer: &Discoverer{
			resolveDir: func(context.Context, string) (string, error) {
				return sampleSDKDir(t), nil
			},
		},
		renderer: NewRenderer(),
	}
}

func testServiceConfig(profile string) ServiceConfig {
	return ServiceConfig{
		Service:        "mysql",
		SDKPackage:     "github.com/oracle/oci-service-operator/internal/generator/testdata/sdk/sample",
		Group:          "mysql",
		PackageProfile: profile,
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(content)
}

func assertExactFileMatch(t *testing.T, wantPath string, gotPath string) {
	t.Helper()

	want := readFile(t, wantPath)
	got := readFile(t, gotPath)
	if want != got {
		t.Fatalf("file mismatch for %s", wantPath)
	}
}

func assertResourceOrderContainsSubset(t *testing.T, fullPath string, subsetPath string) {
	t.Helper()

	full, err := readSampleKustomizationOrder(fullPath)
	if err != nil {
		t.Fatalf("readSampleKustomizationOrder(%s) error = %v", fullPath, err)
	}
	subset, err := readSampleKustomizationOrder(subsetPath)
	if err != nil {
		t.Fatalf("readSampleKustomizationOrder(%s) error = %v", subsetPath, err)
	}

	position := 0
	for _, name := range subset {
		offset := slices.Index(full[position:], name)
		if offset < 0 {
			t.Fatalf("resource %q from %s was not found in %s", name, subsetPath, fullPath)
		}
		position += offset + 1
	}
}

func assertGoEquivalent(t *testing.T, wantPath string, gotPath string) {
	t.Helper()

	want := normalizeGoForComparison(t, readFile(t, wantPath))
	got := normalizeGoForComparison(t, readFile(t, gotPath))
	if want != got {
		t.Fatalf("Go mismatch for %s\nwant:\n%s\n\ngot:\n%s", wantPath, want, got)
	}
}

func assertContains(t *testing.T, content string, want []string) {
	t.Helper()

	for _, needle := range want {
		if !strings.Contains(content, needle) {
			t.Fatalf("content did not contain %q:\n%s", needle, content)
		}
	}
}

func assertNotContains(t *testing.T, content string, want []string) {
	t.Helper()

	for _, needle := range want {
		if strings.Contains(content, needle) {
			t.Fatalf("content unexpectedly contained %q:\n%s", needle, content)
		}
	}
}

func findResource(t *testing.T, resources []ResourceModel, kind string) ResourceModel {
	t.Helper()

	for _, resource := range resources {
		if resource.Kind == kind {
			return resource
		}
	}

	t.Fatalf("resource kind %q was not found in %#v", kind, resources)
	return ResourceModel{}
}

func findFieldModel(t *testing.T, fields []FieldModel, name string) FieldModel {
	t.Helper()

	for _, field := range fields {
		if field.Name == name {
			return field
		}
	}

	t.Fatalf("field %q was not found in %#v", name, fields)
	return FieldModel{}
}

func findHelperType(t *testing.T, helperTypes []TypeModel, name string) TypeModel {
	t.Helper()

	for _, helperType := range helperTypes {
		if helperType.Name == name {
			return helperType
		}
	}

	t.Fatalf("helper type %q was not found in %#v", name, helperTypes)
	return TypeModel{}
}

func findServiceManagerModel(t *testing.T, serviceManagers []ServiceManagerModel, kind string) ServiceManagerModel {
	t.Helper()

	for _, serviceManager := range serviceManagers {
		if serviceManager.Kind == kind {
			return serviceManager
		}
	}

	t.Fatalf("service manager kind %q was not found in %#v", kind, serviceManagers)
	return ServiceManagerModel{}
}

func generatorTestDir(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}

	return filepath.Dir(currentFile)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Clean(filepath.Join(generatorTestDir(t), "..", ".."))
}

func normalizeGoForComparison(t *testing.T, source string) string {
	t.Helper()

	source = strings.ReplaceAll(source, "// Code generated by generator. DO NOT EDIT.\n\n", "")

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "comparison.go", source, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("ParseFile() error = %v\nsource:\n%s", err, source)
	}

	stripGoComments(file)

	var builder strings.Builder
	if err := format.Node(&builder, fileSet, file); err != nil {
		t.Fatalf("format.Node() error = %v", err)
	}

	lines := strings.Split(builder.String(), "\n")
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, strings.Join(strings.Fields(trimmed), " "))
	}

	return strings.Join(normalized, "\n")
}

func writeGeneratorTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func writeGeneratorFormalScaffold(t *testing.T, repoRoot string, service string, slug string, kind string) {
	t.Helper()

	formalRoot := filepath.Join(repoRoot, "formal")
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "controller_manifest.tsv"), "service\tslug\tkind\tstage\tsurface\timport\tspec\tlogic_gaps\tdiagram_dir\n"+
		fmt.Sprintf("%s\t%s\t%s\tscaffold\trepo-authored-semantics\timports/%s/%s.json\tcontrollers/%s/%s/spec.cfg\tcontrollers/%s/%s/logic-gaps.md\tcontrollers/%s/%s/diagrams\n", service, slug, kind, service, slug, service, slug, service, slug, service, slug))
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "sources.lock"), `{
  "schemaVersion": 1,
  "sources": [
    {
      "name": "terraform-provider-oci",
      "surface": "provider-facts",
      "status": "scaffold",
      "notes": [
        "formal-import will pin a provider revision here."
      ]
    }
  ]
}
`)
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "shared", "BaseReconcilerContract.tla"), `------------------------------ MODULE BaseReconcilerContract ------------------------------
EXTENDS ControllerLifecycleSpec

VARIABLES deletionRequested, deleteConfirmed, finalizerPresent, lifecycleCondition, shouldRequeue, requestedAtStamped

FinalizerRetention == deletionRequested /\ ~deleteConfirmed => finalizerPresent
RetryableConditionsRequeue == lifecycleCondition \in {"Provisioning", "Updating", "Terminating"} => shouldRequeue
StatusProjectionStampsRequestedAt == requestedAtStamped

=============================================================================
`)
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "shared", "ControllerLifecycleSpec.tla"), `------------------------------ MODULE ControllerLifecycleSpec ------------------------------

RetryableConditions == {"Provisioning", "Updating", "Terminating"}
ShouldRequeue(condition) == condition \in RetryableConditions

=============================================================================
`)
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "shared", "OSOKServiceManagerContract.tla"), `------------------------------ MODULE OSOKServiceManagerContract ------------------------------
EXTENDS Naturals

ResponseShape(response) == response \in [IsSuccessful : BOOLEAN, ShouldRequeue : BOOLEAN, RequeueDuration : Nat]

=============================================================================
`)
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "shared", "SecretSideEffectsContract.tla"), `------------------------------ MODULE SecretSideEffectsContract ------------------------------

SecretWritePolicies == {"none", "ready-only"}
SecretWritesAllowed(policy, condition) == IF policy = "none" THEN FALSE ELSE condition = "Active"

=============================================================================
`)
	writeGeneratorDiagramStrategyFixtures(t, formalRoot)
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "controllers", service, slug, "spec.cfg"), fmt.Sprintf(`# formal controller binding schema v1
schema_version = 1
surface = repo-authored-semantics
service = %s
slug = %s
kind = %s
stage = scaffold
import = imports/%s/%s.json
shared_contracts = shared/BaseReconcilerContract.tla,shared/ControllerLifecycleSpec.tla,shared/OSOKServiceManagerContract.tla,shared/SecretSideEffectsContract.tla
status_projection = required
success_condition = active
requeue_conditions = provisioning,updating,terminating
delete_confirmation = required
finalizer_policy = retain-until-confirmed-delete
secret_side_effects = none
`, service, slug, kind, service, slug))
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "controllers", service, slug, "logic-gaps.md"), fmt.Sprintf(`---
schemaVersion: 1
surface: repo-authored-semantics
service: %s
slug: %s
gaps: []
---

# Logic Gaps
`, service, slug))
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "controllers", service, slug, "diagrams", "runtime-lifecycle.yaml"), fmt.Sprintf(`schemaVersion: 1
surface: repo-authored-semantics
service: %s
slug: %s
kind: %s
archetype: generated-service-manager
states:
  - provisioning
  - active
  - updating
  - terminating
notes:
  - Scaffold metadata only.
`, service, slug, kind))
	writeGeneratorTestFile(t, filepath.Join(formalRoot, "imports", service, slug+".json"), fmt.Sprintf(`{
  "schemaVersion": 1,
  "surface": "provider-facts",
  "service": %q,
  "slug": %q,
  "kind": %q,
  "sourceRef": "terraform-provider-oci",
  "providerResource": "widget_resource",
  "operations": {
    "create": [
      {
        "operation": "CreateWidget",
        "requestType": "CreateWidgetRequest",
        "responseType": "CreateWidgetResponse"
      }
    ],
    "get": [
      {
        "operation": "GetWidget",
        "requestType": "GetWidgetRequest",
        "responseType": "GetWidgetResponse"
      }
    ],
    "list": [
      {
        "operation": "ListWidgets",
        "requestType": "ListWidgetsRequest",
        "responseType": "ListWidgetsResponse"
      }
    ],
    "update": [
      {
        "operation": "UpdateWidget",
        "requestType": "UpdateWidgetRequest",
        "responseType": "UpdateWidgetResponse"
      }
    ],
    "delete": [
      {
        "operation": "DeleteWidget",
        "requestType": "DeleteWidgetRequest",
        "responseType": "DeleteWidgetResponse"
      }
    ]
  },
  "lifecycle": {
    "create": {
      "pending": ["PROVISIONING"],
      "target": ["ACTIVE"]
    },
    "update": {
      "pending": ["UPDATING"],
      "target": ["ACTIVE"]
    }
  },
  "mutation": {
    "mutable": ["display_name"],
    "forceNew": ["compartment_id"],
    "conflictsWith": {}
  },
  "hooks": {
    "create": [
      {
        "helper": "tfresource.CreateResource"
      }
    ]
  },
  "deleteConfirmation": {
    "pending": ["DELETING"],
    "target": ["DELETED"]
  },
  "listLookup": {
    "datasource": "oci_widget_widgets",
    "collectionField": "widgets",
    "responseItemsField": "Items",
    "filterFields": ["compartment_id", "state"]
  },
  "boundary": {
    "providerFactsOnly": true,
    "repoAuthoredSpecPath": "controllers/%s/%s/spec.cfg",
    "repoAuthoredLogicGapsPath": "controllers/%s/%s/logic-gaps.md",
    "excludedSemantics": [
      "bind-versus-create",
      "secret-output",
      "delete-confirmation"
    ]
  }
}
`, service, slug, kind, service, slug, service, slug))
	if _, err := formal.RenderDiagrams(formal.RenderOptions{Root: formalRoot}); err != nil {
		t.Fatalf("RenderDiagrams(%q) error = %v", formalRoot, err)
	}
}

func stripGoComments(file *ast.File) {
	file.Comments = nil

	for _, decl := range file.Decls {
		stripDeclComments(decl)
	}
}

func stripDeclComments(decl ast.Decl) {
	switch concrete := decl.(type) {
	case *ast.GenDecl:
		stripGenDeclComments(concrete)
	case *ast.FuncDecl:
		concrete.Doc = nil
	}
}

func stripGenDeclComments(decl *ast.GenDecl) {
	decl.Doc = nil
	for _, spec := range decl.Specs {
		stripSpecComments(spec)
	}
}

func stripSpecComments(spec ast.Spec) {
	switch typed := spec.(type) {
	case *ast.TypeSpec:
		stripTypeSpecComments(typed)
	case *ast.ValueSpec:
		typed.Doc = nil
		typed.Comment = nil
	}
}

func stripTypeSpecComments(typeSpec *ast.TypeSpec) {
	typeSpec.Doc = nil
	typeSpec.Comment = nil

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok || structType.Fields == nil {
		return
	}

	for _, field := range structType.Fields.List {
		field.Doc = nil
		field.Comment = nil
	}
}
