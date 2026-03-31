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
		Generation: GenerationConfig{
			Resources: []ResourceGenerationOverride{
				{
					Kind:    "MySqlDbSystem",
					SDKName: "DbSystem",
				},
			},
		},
	}

	discoverer := &Discoverer{
		resolveDir: func(context.Context, string) (string, error) {
			return sampleSDKDir(t), nil
		},
	}

	pkg := mustBuildPackageModelWithDiscoverer(t, discoverer, cfg, service)

	if pkg.GroupDNSName != "mysql.oracle.com" {
		t.Fatalf("BuildPackageModel() group DNS name = %q, want %q", pkg.GroupDNSName, "mysql.oracle.com")
	}

	assertDiscoveredMySQLDbSystem(t, findResource(t, pkg.Resources, "MySqlDbSystem"))
	assertDiscoveredWidget(t, findResource(t, pkg.Resources, "Widget"))
	assertDiscoveredReport(t, findResource(t, pkg.Resources, "Report"))
	assertResourceSpecFields(t, findResource(t, pkg.Resources, "ReportByName"), []string{"DisplayName"})
	assertResourceSpecFields(t, findResource(t, pkg.Resources, "OAuthClientCredential"), []string{"Name", "Description", "Scopes"})
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
    generation:
      resources:
        - kind: Widget
          formalSpec: widget
          serviceManager:
            strategy: generated
`)
	writeGeneratorFormalScaffold(t, repo, "mysql", "widget", "Widget")

	pkg := buildPackageModelWithFormalWidgetScaffold(t, repo, configPath)
	assertWidgetFormalModel(t, pkg)
	if report := findResource(t, pkg.Resources, "Report"); report.Formal != nil {
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
    generation:
      resources:
        - kind: Widget
          formalSpec: widget
          serviceManager:
            strategy: generated
`)
	writeGeneratorFormalScaffold(t, repo, "mysql", "widget", "Widget")

	pkg := buildPackageModelWithFormalWidgetScaffold(t, repo, configPath)
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
			assert: assertFunctionsComplexSDKFields,
		},
		{
			name: "core",
			service: ServiceConfig{
				Service:        "core",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/core",
				Group:          "core",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: assertCoreComplexSDKFields,
		},
		{
			name: "certificates",
			service: ServiceConfig{
				Service:        "certificates",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/certificates",
				Group:          "certificates",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: assertCertificatesComplexSDKFields,
		},
		{
			name: "nosql",
			service: ServiceConfig{
				Service:        "nosql",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/nosql",
				Group:          "nosql",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: assertNoSQLComplexSDKFields,
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
			assert: assertSecretsComplexSDKFields,
		},
		{
			name: "vault",
			service: ServiceConfig{
				Service:        "vault",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/vault",
				Group:          "vault",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: assertVaultComplexSDKFields,
		},
		{
			name: "artifacts",
			service: ServiceConfig{
				Service:        "artifacts",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/artifacts",
				Group:          "artifacts",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: assertArtifactsComplexSDKFields,
		},
		{
			name: "networkloadbalancer",
			service: ServiceConfig{
				Service:        "networkloadbalancer",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/networkloadbalancer",
				Group:          "networkloadbalancer",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: assertNetworkLoadBalancerComplexSDKFields,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			test.assert(t, mustBuildPackageModel(t, cfg, test.service))
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

	pkg := mustBuildPackageModel(t, cfg, service)
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "DbSystem"), []string{"DisplayName", "CompartmentId", "Shape", "DbVersion"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "Configuration"), []string{"DisplayName", "Shape", "DbVersion", "InstanceOcpuCount"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "Backup"), []string{"DisplayName", "CompartmentId", "DbSystemId", "RetentionPeriod"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "PrimaryDbInstance"), []string{"DbInstanceId"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "WorkRequestLog"), []string{"Message", "Timestamp"})
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

	pkg := mustBuildPackageModel(t, cfg, service)
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "Cluster"), []string{"Name", "CompartmentId", "EndpointConfig", "VcnId", "KubernetesVersion", "KmsKeyId", "FreeformTags", "DefinedTags", "Options", "ImagePolicyConfig", "ClusterPodNetworkOptions", "Type"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "NodePool"), []string{"CompartmentId", "ClusterId", "Name", "KubernetesVersion", "NodeMetadata", "NodeImageName", "NodeSourceDetails", "NodeShapeConfig", "InitialNodeLabels", "SshPublicKey", "QuantityPerSubnet", "SubnetIds", "NodeConfigDetails", "FreeformTags", "DefinedTags", "NodeEvictionNodePoolSettings", "NodePoolCyclingDetails"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "VirtualNodePool"), []string{"CompartmentId", "ClusterId", "DisplayName", "PlacementConfigurations", "InitialVirtualNodeLabels", "Taints", "Size", "NsgIds", "PodConfiguration", "FreeformTags", "DefinedTags", "VirtualNodeTags"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "Addon"), []string{"Version", "Configurations"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "WorkloadMapping"), []string{"Namespace", "MappedCompartmentId", "FreeformTags", "DefinedTags"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "WorkRequestLog"), []string{"Message", "Timestamp"})
	assertFieldTag(t, findResource(t, pkg.Resources, "WorkRequest").StatusFields, "Status", `json:"sdkStatus,omitempty"`)
	assertFieldTag(t, findResource(t, pkg.Resources, "CredentialRotationStatus").StatusFields, "Status", `json:"sdkStatus,omitempty"`)
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

	pkg := mustBuildPackageModel(t, cfg, service)
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "BulkActionResourceType"), []string{"Items"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "BulkEditTagsResourceType"), []string{"Items"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "CostTrackingTag"), []string{"TagNamespaceId", "TagNamespaceName", "IsRetired", "Validator"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "IdentityProvider"), []string{"CompartmentId", "Name", "Description", "Metadata", "MetadataUrl", "ProductType"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "NetworkSource"), []string{"CompartmentId", "Name", "Description", "PublicSourceList", "Services", "VirtualSourceList"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "OrResetUIPassword"), []string{"Password", "UserId", "TimeCreated", "LifecycleState", "InactiveStatus"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "StandardTagNamespace"), []string{"Description", "StandardTagNamespaceName", "TagDefinitionTemplates"})
	assertFieldTag(t, findResource(t, pkg.Resources, "StandardTagNamespace").StatusFields, "Status", `json:"sdkStatus,omitempty"`)
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "StandardTagTemplate"), []string{"Description", "TagDefinitionName", "Type", "IsCostTracking"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "UserState"), []string{"Id", "CompartmentId", "Name", "LifecycleState", "Capabilities"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "UserUIPasswordInformation"), []string{"UserId", "TimeCreated", "LifecycleState"})
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

	pkg := mustBuildPackageModel(t, cfg, service)
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "ClusterNetworkInstance"), []string{"AvailabilityDomain", "CompartmentId", "Region", "State", "TimeCreated"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "ComputeCapacityReservationInstance"), []string{"AvailabilityDomain", "CompartmentId", "Id", "Shape"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "ComputeGlobalImageCapabilitySchema"), []string{"ComputeGlobalImageCapabilitySchemaId", "Name"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "NetworkSecurityGroupSecurityRule"), []string{"Direction", "Protocol", "Id", "TcpOptions", "UdpOptions"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "IPSecConnectionTunnelError"), []string{"ErrorCode", "ErrorDescription", "Id", "Solution", "Timestamp"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "IPSecConnectionTunnelRoute"), []string{"Advertiser", "AsPath", "IsBestPath", "Prefix"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "IPSecConnectionTunnelSecurityAssociation"), []string{"CpeSubnet", "OracleSubnet", "TunnelSaStatus"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "InstanceDevice"), []string{"IsAvailable", "Name"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "VolumeBackupPolicyAssetAssignment"), []string{"AssetId", "Id", "PolicyId", "TimeCreated"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "WindowsInstanceInitialCredential"), []string{"Password", "Username"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "FastConnectProviderVirtualCircuitBandwidthShape"), []string{"BandwidthInMbps", "Name"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "CrossconnectPortSpeedShape"), []string{"Name", "PortSpeedInGbps"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "AllDrgAttachment"), []string{"Id"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "AllowedPeerRegionsForRemotePeering"), []string{"Name"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "AppCatalogListingAgreement"), []string{"ListingId", "ListingResourceVersion", "OracleTermsOfUseLink", "EulaLink", "TimeRetrieved", "Signature"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "CrossConnectLetterOfAuthority"), []string{"CrossConnectId", "FacilityLocation", "TimeExpires"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "CrossConnectMapping"), []string{"Ipv4BgpStatus", "Ipv6BgpStatus", "OciLogicalDeviceName"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "DhcpOption"), []string{"CompartmentId", "DisplayName", "LifecycleState", "Options", "TimeCreated", "VcnId"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "VirtualCircuitAssociatedTunnel"), []string{"TunnelId", "TunnelType"})
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

func TestGenerateRendersAndSkipsExisting(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileControllerBacked)
	pipeline := newTestGenerator(t)

	outputRoot := t.TempDir()
	result := mustGenerateRun(t, pipeline, cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	})
	if len(result.Generated) != 1 {
		t.Fatalf("Generate() generated = %d services, want 1", len(result.Generated))
	}

	assertContains(t, readFile(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "groupversion_info.go")), []string{
		"// Code generated by generator. DO NOT EDIT.",
		`GroupVersion = schema.GroupVersion{Group: "mysql.oracle.com", Version: "v1beta1"}`,
	})
	assertContains(t, readFile(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "mysqldbsystem_types.go")), []string{
		"type MySqlDbSystemSpec struct",
		"Port",
		`json:"port,omitempty"`,
	})

	result = mustGenerateRun(t, pipeline, cfg, []ServiceConfig{service}, Options{
		OutputRoot:   outputRoot,
		SkipExisting: true,
	})
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

	pkg := mustBuildPackageModel(t, cfg, service)

	application := findResource(t, pkg.Resources, "Application")
	assertFieldMarkers(t, application.SpecFields, "CompartmentId", []string{"+kubebuilder:validation:Required"})
	assertFieldTag(t, application.SpecFields, "CompartmentId", `json:"compartmentId"`)
	assertFieldCommentContains(t, application.SpecFields, "CompartmentId", "compartment to create the application within")
	assertFieldMarkers(t, application.SpecFields, "Config", []string{"+kubebuilder:validation:Optional"})
	assertFieldTag(t, application.SpecFields, "Config", `json:"config,omitempty"`)
	assertFieldCommentContains(t, application.SpecFields, "Config", "Application configuration")
	assertFieldHasNoMarkers(t, application.StatusFields, "LifecycleState")
	assertFieldCommentContains(t, application.StatusFields, "LifecycleState", "current state of the application")

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
			wantInstallNotContains: []string{
				"mysqldbsystem_editor_role.yaml",
				"mysqldbsystem_viewer_role.yaml",
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
				"mysqldbsystem_editor_role.yaml",
				"mysqldbsystem_viewer_role.yaml",
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

			sampleContent := readFile(t, filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_mysqldbsystem.yaml"))
			assertContains(t, sampleContent, []string{
				"apiVersion: mysql.oracle.com/v1beta1",
				"kind: MySqlDbSystem",
				"name: mysqldbsystem-sample",
			})

			sampleKustomization := readFile(t, filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"))
			assertContains(t, sampleKustomization, []string{
				"- mysql_v1beta1_mysqldbsystem.yaml",
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
			Kind:    "MySqlDbSystem",
			SDKName: "DbSystem",
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

	content := readFile(t, filepath.Join(outputRoot, "controllers", "mysql", "mysqldbsystem_controller.go"))
	assertContains(t, content, []string{
		"package mysql",
		`mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"`,
		"type MySqlDbSystemReconciler struct {",
		"// +kubebuilder:rbac:groups=mysql.oracle.com,resources=mysqldbsystems,verbs=get;list;watch;create;update;patch;delete",
		"// +kubebuilder:rbac:groups=mysql.oracle.com,resources=mysqldbsystems/status,verbs=get;update;patch",
		"// +kubebuilder:rbac:groups=mysql.oracle.com,resources=mysqldbsystems/finalizers,verbs=update",
		`// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch`,
		`// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch`,
		"builder = builder.WithOptions(controller.Options{MaxConcurrentReconciles: 3})",
		"mySqlDbSystem := &mysqlv1beta1.MySqlDbSystem{}",
		"return r.Reconciler.Reconcile(ctx, req, mySqlDbSystem)",
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
	controllerPath := filepath.Join(controllerDir, "mysqldbsystem_controller.go")
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
		`mysqlmysqldbsystemservicemanager "github.com/oracle/oci-service-operator/pkg/servicemanager/mysql/mysqldbsystem"`,
		"registerGeneratedGroup(GroupRegistration{",
		`Group:       "mysql",`,
		"AddToScheme: mysqlv1beta1.AddToScheme,",
		"(&mysqlcontrollers.MySqlDbSystemReconciler{",
		`ctx,`,
		`"MySqlDbSystem",`,
		"return mysqlmysqldbsystemservicemanager.NewMySqlDbSystemServiceManagerWithDeps(deps)",
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
	if !strings.Contains(err.Error(), `registration strategy "generated" requires generated controller output for kind "MySqlDbSystem"`) {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestGenerateControllerBackedPackagesDoNotReferenceEditorViewerRolesByDefault(t *testing.T) {
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

func TestGenerateControllerBackedPackagesIncludeServicePackageExtraResources(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "database",
		SDKPackage:     "example.com/test/sdk",
		Group:          "database",
		PackageProfile: PackageProfileControllerBacked,
		Package: PackageConfig{
			ExtraResources: []string{
				"../../../config/rbac/autonomousdatabases_editor_role.yaml",
				"../../../config/rbac/autonomousdatabases_viewer_role.yaml",
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

	installContent := readFile(t, filepath.Join(outputRoot, "packages", "database", "install", "kustomization.yaml"))
	assertContains(t, installContent, []string{
		"- generated/crd",
		"- generated/rbac",
		"- ../../../config/manager",
		"- ../../../config/rbac/autonomousdatabases_editor_role.yaml",
		"- ../../../config/rbac/autonomousdatabases_viewer_role.yaml",
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
			Kind:    "MySqlDbSystem",
			SDKName: "DbSystem",
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

	serviceClientPath := filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "mysqldbsystem", "mysqldbsystem_serviceclient.go")
	serviceManagerPath := filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "mysqldbsystem", "mysqldbsystem_servicemanager.go")

	serviceClientContent := readFile(t, serviceClientPath)
	assertContains(t, serviceClientContent, []string{
		"package mysqldbsystem",
		"type MySqlDbSystemServiceClient interface {",
		"var newMySqlDbSystemServiceClient = func(manager *MySqlDbSystemServiceManager) MySqlDbSystemServiceClient {",
		`Kind:    "MySqlDbSystem"`,
		"github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime",
		"generatedruntime.NewServiceClient[*mysqlv1beta1.MySqlDbSystem]",
		"mysqlsdk.NewSampleClientWithConfigurationProvider(manager.Provider)",
	})

	serviceManagerContent := readFile(t, serviceManagerPath)
	assertContains(t, serviceManagerContent, []string{
		"type MySqlDbSystemServiceManager struct {",
		"func NewMySqlDbSystemServiceManagerWithDeps(deps servicemanager.RuntimeDeps) *MySqlDbSystemServiceManager {",
		"func (c *MySqlDbSystemServiceManager) WithClient(client MySqlDbSystemServiceClient) *MySqlDbSystemServiceManager {",
		"func (c *MySqlDbSystemServiceManager) GetCrdStatus(obj runtime.Object) (*shared.OSOKStatus, error) {",
		"return &resource.Status.OsokStatus",
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
			Kind:    "MySqlDbSystem",
			SDKName: "DbSystem",
			ServiceManager: ServiceManagerGenerationOverride{
				PackagePath: "mysql/dbsystem",
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

	serviceClientContent := readFile(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "dbsystem", "mysqldbsystem_serviceclient.go"))
	serviceManagerContent := readFile(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "dbsystem", "mysqldbsystem_servicemanager.go"))

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

	relativePackagePath, err := filepath.Rel(moduleRoot, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "mysqldbsystem"))
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

func TestCurrentServiceParityMatchesCheckedInArtifacts(t *testing.T) {
	cfg := loadCheckedInGeneratorConfig(t)

	var services []ServiceConfig
	for _, service := range cfg.Services {
		if slices.Contains([]string{"database", "mysql", "streaming"}, service.Service) {
			services = append(services, service)
		}
	}
	if len(services) != 3 {
		t.Fatalf("selected %d parity services, want 3", len(services))
	}

	outputRoot := t.TempDir()
	seedSampleKustomization(t, outputRoot)
	pipeline := New()
	result := mustGenerateRun(t, pipeline, cfg, services, Options{
		OutputRoot:                      outputRoot,
		PreserveExistingSpecSurfaceRoot: repoRoot(t),
	})
	if len(result.Generated) != 3 {
		t.Fatalf("Generate() generated %d services, want 3", len(result.Generated))
	}
	assertServiceResourceCounts(t, result.Generated, map[string]int{
		"database":  79,
		"mysql":     12,
		"streaming": 7,
	})

	apiFiles := []string{
		"api/database/v1beta1/groupversion_info.go",
		"api/database/v1beta1/autonomousdatabase_types.go",
		"api/database/v1beta1/autonomousdatabasebackup_types.go",
		"api/mysql/v1beta1/groupversion_info.go",
		"api/mysql/v1beta1/mysqldbsystem_types.go",
		"api/mysql/v1beta1/backup_types.go",
		"api/streaming/v1beta1/groupversion_info.go",
		"api/streaming/v1beta1/stream_types.go",
		"api/streaming/v1beta1/streampool_types.go",
	}
	assertGoParityFiles(t, repoRoot(t), outputRoot, apiFiles)

	exactFiles := []string{
		"config/samples/database_v1beta1_autonomousdatabase.yaml",
		"config/samples/mysql_v1beta1_mysqldbsystem.yaml",
		"config/samples/streaming_v1beta1_stream.yaml",
		"packages/database/metadata.env",
		"packages/database/install/kustomization.yaml",
		"packages/mysql/metadata.env",
		"packages/mysql/install/kustomization.yaml",
		"packages/streaming/metadata.env",
		"packages/streaming/install/kustomization.yaml",
	}
	assertExactFileMatches(t, repoRoot(t), outputRoot, exactFiles)

	runtimeFiles := []string{
		"controllers/database/autonomousdatabase_controller.go",
		"controllers/database/autonomousdatabasebackup_controller.go",
		"controllers/mysql/mysqldbsystem_controller.go",
		"controllers/mysql/backup_controller.go",
		"controllers/streaming/stream_controller.go",
		"controllers/streaming/streampool_controller.go",
		"pkg/servicemanager/database/autonomousdatabase/autonomousdatabase_serviceclient.go",
		"pkg/servicemanager/database/autonomousdatabase/autonomousdatabase_servicemanager.go",
		"pkg/servicemanager/database/autonomousdatabasebackup/autonomousdatabasebackup_serviceclient.go",
		"pkg/servicemanager/database/autonomousdatabasebackup/autonomousdatabasebackup_servicemanager.go",
		"pkg/servicemanager/mysql/dbsystem/mysqldbsystem_serviceclient.go",
		"pkg/servicemanager/mysql/dbsystem/mysqldbsystem_servicemanager.go",
		"pkg/servicemanager/mysql/backup/backup_serviceclient.go",
		"pkg/servicemanager/mysql/backup/backup_servicemanager.go",
		"pkg/servicemanager/streaming/stream/stream_serviceclient.go",
		"pkg/servicemanager/streaming/stream/stream_servicemanager.go",
		"pkg/servicemanager/streaming/streampool/streampool_serviceclient.go",
		"pkg/servicemanager/streaming/streampool/streampool_servicemanager.go",
		"internal/registrations/database_generated.go",
		"internal/registrations/mysql_generated.go",
		"internal/registrations/streaming_generated.go",
	}
	assertGoParityFiles(t, repoRoot(t), outputRoot, runtimeFiles)

	assertResourceOrderContainsSubset(
		t,
		filepath.Join(repoRoot(t), "config", "samples", "kustomization.yaml"),
		filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"),
	)
}

func TestCheckedInPromotedCoreRuntimeArtifactsMatchGenerator(t *testing.T) {
	cfg := loadCheckedInGeneratorConfig(t)
	coreService := mustFindGeneratorService(t, cfg, "core")

	outputRoot := t.TempDir()
	seedSampleKustomization(t, outputRoot)

	pipeline := New()
	result := mustGenerateRun(t, pipeline, cfg, []ServiceConfig{*coreService}, Options{
		OutputRoot:                      outputRoot,
		PreserveExistingSpecSurfaceRoot: repoRoot(t),
	})
	if len(result.Generated) != 1 {
		t.Fatalf("Generate() generated %d services, want 1", len(result.Generated))
	}

	runtimeFiles := []string{
		"controllers/core/vcn_controller.go",
		"pkg/servicemanager/core/vcn/vcn_serviceclient.go",
		"pkg/servicemanager/core/vcn/vcn_servicemanager.go",
		"internal/registrations/core_generated.go",
	}
	assertGoParityFiles(t, repoRoot(t), outputRoot, runtimeFiles)

	exactFiles := []string{
		"packages/core/metadata.env",
		"packages/core/install/kustomization.yaml",
	}
	assertExactFileMatches(t, repoRoot(t), outputRoot, exactFiles)
}

func TestCheckedInIdentityUserRuntimeArtifactsMatchGenerator(t *testing.T) {
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
		OutputRoot:                      outputRoot,
		PreserveExistingSpecSurfaceRoot: repoRoot(t),
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Generated) != 1 {
		t.Fatalf("Generate() generated %d services, want 1", len(result.Generated))
	}

	serviceClientPath := "pkg/servicemanager/identity/user/user_serviceclient.go"
	serviceManagerPath := "pkg/servicemanager/identity/user/user_servicemanager.go"
	assertGeneratedGoMatches(t, filepath.Join(repoRoot(t), serviceClientPath), filepath.Join(outputRoot, serviceClientPath))
	assertGeneratedGoMatches(t, filepath.Join(repoRoot(t), serviceManagerPath), filepath.Join(outputRoot, serviceManagerPath))

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

func TestMySQLPublishedKindIncludesOptionalDesiredStateFields(t *testing.T) {
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

	content := readFile(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "mysqldbsystem_types.go"))
	normalized := strings.Join(strings.Fields(content), " ")
	assertContains(t, normalized, []string{
		"type MySqlDbSystemDeletionPolicy struct {",
		"AutomaticBackupRetention string `json:\"automaticBackupRetention,omitempty\"`",
		"FinalBackup string `json:\"finalBackup,omitempty\"`",
		"IsDeleteProtected bool `json:\"isDeleteProtected,omitempty\"`",
		"type MySqlDbSystemSecureConnections struct {",
		"CertificateGenerationType string `json:\"certificateGenerationType\"`",
		"CertificateId string `json:\"certificateId,omitempty\"`",
		"DeletionPolicy MySqlDbSystemDeletionPolicy `json:\"deletionPolicy,omitempty\"`",
		"CrashRecovery string `json:\"crashRecovery,omitempty\"`",
		"DatabaseManagement string `json:\"databaseManagement,omitempty\"`",
		"SecureConnections MySqlDbSystemSecureConnections `json:\"secureConnections,omitempty\"`",
		"type MySqlDbSystemSourceObservedState struct {",
		"Source MySqlDbSystemSourceObservedState `json:\"source,omitempty\"`",
		"BackupId string `json:\"backupId\"`",
		"DbSystemId string `json:\"dbSystemId\"`",
	})
	if slices.Contains(structFieldNames(t, content, "MySqlDbSystemSourceObservedState"), "SourceUrl") {
		t.Fatalf("MySqlDbSystemSourceObservedState unexpectedly contains SourceUrl:\n%s", content)
	}
}

func TestGenerateMergesExistingSampleKustomizationEntries(t *testing.T) {
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
		"- mysql_v1beta1_mysqldbsystem.yaml",
	})
	if strings.Index(sampleKustomization, "- existing.yaml") > strings.Index(sampleKustomization, "- mysql_v1beta1_mysqldbsystem.yaml") {
		t.Fatalf("existing sample entry was not preserved ahead of the generated sample:\n%s", sampleKustomization)
	}
}

func mustBuildPackageModel(t *testing.T, cfg *Config, service ServiceConfig) *PackageModel {
	t.Helper()

	return mustBuildPackageModelWithDiscoverer(t, NewDiscoverer(), cfg, service)
}

func mustBuildPackageModelWithDiscoverer(t *testing.T, discoverer *Discoverer, cfg *Config, service ServiceConfig) *PackageModel {
	t.Helper()

	pkg, err := discoverer.BuildPackageModel(context.Background(), cfg, service)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	return pkg
}

func buildPackageModelWithFormalWidgetScaffold(t *testing.T, _ string, configPath string) *PackageModel {
	t.Helper()

	cfg := mustLoadGeneratorConfig(t, configPath)
	discoverer := &Discoverer{
		resolveDir: func(context.Context, string) (string, error) {
			return sampleSDKDir(t), nil
		},
	}

	return mustBuildPackageModelWithDiscoverer(t, discoverer, cfg, cfg.Services[0])
}

func mustGenerateRun(t *testing.T, pipeline *Generator, cfg *Config, services []ServiceConfig, options Options) RunResult {
	t.Helper()

	result, err := pipeline.Generate(context.Background(), cfg, services, options)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	return result
}

func seedSampleKustomization(t *testing.T, outputRoot string) {
	t.Helper()

	samplesDir := filepath.Join(outputRoot, "config", "samples")
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", samplesDir, err)
	}
	mustWriteGeneratorFile(t, filepath.Join(samplesDir, "kustomization.yaml"), readFile(t, filepath.Join(repoRoot(t), "config", "samples", "kustomization.yaml")))
}

func mustWriteGeneratorFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func assertServiceResourceCounts(t *testing.T, generated []ServiceResult, want map[string]int) {
	t.Helper()

	for _, service := range generated {
		if service.ResourceCount != want[service.Service] {
			t.Fatalf("service %s generated %d resources, want %d", service.Service, service.ResourceCount, want[service.Service])
		}
	}
}

func assertGoParityFiles(t *testing.T, root string, outputRoot string, files []string) {
	t.Helper()

	for _, relativePath := range files {
		assertGoParity(t, filepath.Join(root, relativePath), filepath.Join(outputRoot, relativePath))
	}
}

func assertGoParity(t *testing.T, wantPath string, gotPath string) {
	t.Helper()

	assertGeneratedGoMatches(t, wantPath, gotPath)
}

func assertExactFileMatches(t *testing.T, root string, outputRoot string, files []string) {
	t.Helper()

	for _, relativePath := range files {
		assertExactFileMatch(t, filepath.Join(root, relativePath), filepath.Join(outputRoot, relativePath))
	}
}

func assertResourceSpecFields(t *testing.T, resource ResourceModel, want []string) {
	t.Helper()

	assertResourceFields(t, resource.Kind, "spec", resource.SpecFields, want)
}

func assertResourceStatusFields(t *testing.T, resource ResourceModel, want []string) {
	t.Helper()

	assertResourceFields(t, resource.Kind, "status", resource.StatusFields, want)
}

func assertResourceFields(t *testing.T, kind string, fieldSet string, fields []FieldModel, want []string) {
	t.Helper()

	for _, fieldName := range want {
		if !hasField(fields, fieldName) {
			t.Fatalf("%s %s fields = %#v, want %s", kind, fieldSet, fields, fieldName)
		}
	}
}

func assertHelperFields(t *testing.T, helper TypeModel, want []string) {
	t.Helper()

	for _, fieldName := range want {
		if !hasField(helper.Fields, fieldName) {
			t.Fatalf("%s fields = %#v, want %s", helper.Name, helper.Fields, fieldName)
		}
	}
}

func assertFieldType(t *testing.T, fields []FieldModel, name string, want string) {
	t.Helper()

	field := findFieldModel(t, fields, name)
	if field.Type != want {
		t.Fatalf("%s type = %q, want %q", name, field.Type, want)
	}
}

func assertFieldTag(t *testing.T, fields []FieldModel, name string, want string) {
	t.Helper()

	field := findFieldModel(t, fields, name)
	if field.Tag != want {
		t.Fatalf("%s tag = %q, want %q", name, field.Tag, want)
	}
}

func assertFieldComments(t *testing.T, fields []FieldModel, name string, want []string) {
	t.Helper()

	field := findFieldModel(t, fields, name)
	if !slices.Equal(field.Comments, want) {
		t.Fatalf("%s comments = %#v, want %#v", name, field.Comments, want)
	}
}

func assertFieldCommentContains(t *testing.T, fields []FieldModel, name string, want string) {
	t.Helper()

	field := findFieldModel(t, fields, name)
	if !strings.Contains(strings.Join(field.Comments, "\n"), want) {
		t.Fatalf("%s comments = %#v, want substring %q", name, field.Comments, want)
	}
}

func assertFieldMarkers(t *testing.T, fields []FieldModel, name string, want []string) {
	t.Helper()

	field := findFieldModel(t, fields, name)
	if !slices.Equal(field.Markers, want) {
		t.Fatalf("%s markers = %#v, want %#v", name, field.Markers, want)
	}
}

func assertFieldHasNoMarkers(t *testing.T, fields []FieldModel, name string) {
	t.Helper()

	field := findFieldModel(t, fields, name)
	if len(field.Markers) != 0 {
		t.Fatalf("%s markers = %#v, want none", name, field.Markers)
	}
}

func assertNoField(t *testing.T, fields []FieldModel, name string, context string) {
	t.Helper()

	if hasField(fields, name) {
		t.Fatalf("%s fields = %#v, want no %s", context, fields, name)
	}
}

func assertDiscoveredMySQLDbSystem(t *testing.T, resource ResourceModel) {
	t.Helper()

	if resource.SDKName != "DbSystem" {
		t.Fatalf("MySqlDbSystem SDK name = %q, want %q", resource.SDKName, "DbSystem")
	}
	assertResourceSpecFields(t, resource, []string{"Port"})
	assertNoField(t, resource.SpecFields, "Id", "MySqlDbSystem")
	if resource.PrimaryDisplayField != "DisplayName" {
		t.Fatalf("MySqlDbSystem primary display field = %q, want DisplayName", resource.PrimaryDisplayField)
	}
}

func assertDiscoveredWidget(t *testing.T, resource ResourceModel) {
	t.Helper()

	if len(resource.Operations) != 5 {
		t.Fatalf("Widget operations = %v, want 5 CRUD operations", resource.Operations)
	}
	assertResourceSpecFields(t, resource, []string{"Mode", "CreatedAt"})
	assertNoField(t, resource.SpecFields, "LifecycleState", "Widget spec")
	assertNoField(t, resource.SpecFields, "TimeUpdated", "Widget spec")
	assertResourceStatusFields(t, resource, []string{"LifecycleState", "TimeUpdated"})
	assertFieldTag(t, resource.SpecFields, "CompartmentId", `json:"compartmentId"`)
	assertFieldComments(t, resource.SpecFields, "CompartmentId", []string{"The OCID of the widget compartment."})
	assertFieldMarkers(t, resource.SpecFields, "CompartmentId", []string{"+kubebuilder:validation:Required"})
	assertFieldTag(t, resource.SpecFields, "Labels", `json:"labels,omitempty"`)
	assertFieldComments(t, resource.SpecFields, "Labels", []string{"Additional labels for the widget."})
	assertFieldMarkers(t, resource.SpecFields, "Labels", []string{"+kubebuilder:validation:Optional"})
	assertFieldTag(t, resource.SpecFields, "ServerState", `json:"serverState,omitempty"`)
	assertFieldHasNoMarkers(t, resource.SpecFields, "ServerState")
	assertFieldComments(t, resource.StatusFields, "LifecycleState", []string{"The lifecycle state of the widget."})
	assertFieldHasNoMarkers(t, resource.StatusFields, "LifecycleState")
}

func assertDiscoveredReport(t *testing.T, resource ResourceModel) {
	t.Helper()

	if len(resource.SpecFields) != 0 {
		t.Fatalf("Report spec fields = %#v, want empty spec when no create or update payload exists", resource.SpecFields)
	}
	assertResourceStatusFields(t, resource, []string{"Id", "LifecycleState", "DisplayName"})
}

func assertWidgetFormalModel(t *testing.T, pkg *PackageModel) {
	t.Helper()

	widget := findResource(t, pkg.Resources, "Widget")
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

func assertWidgetRuntimeSemantics(t *testing.T, resource ResourceModel) {
	t.Helper()

	if resource.Runtime == nil || resource.Runtime.Semantics == nil {
		t.Fatal("Widget runtime semantics were not attached")
	}
	semantics := resource.Runtime.Semantics
	assertWidgetLifecycleSemantics(t, semantics)
	assertWidgetListSemantics(t, semantics)
	assertWidgetMutationSemantics(t, semantics)
	assertWidgetFollowUpSemantics(t, semantics)
	assertWidgetOpenGaps(t, semantics)
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

func assertFunctionsComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	application := findResource(t, pkg.Resources, "Application")
	assertFieldType(t, application.SpecFields, "TraceConfig", "ApplicationTraceConfig")
	assertFieldType(t, application.SpecFields, "ImagePolicyConfig", "ApplicationImagePolicyConfig")
	assertFieldType(t, application.SpecFields, "DefinedTags", "map[string]shared.MapValue")
	assertHelperFields(t, findHelperType(t, application.HelperTypes, "ApplicationTraceConfig"), []string{"DomainId"})
	assertHelperFields(t, findHelperType(t, application.HelperTypes, "ApplicationImagePolicyConfig"), []string{"IsPolicyEnabled"})

	function := findResource(t, pkg.Resources, "Function")
	assertFieldType(t, function.SpecFields, "SourceDetails", "FunctionSourceDetails")
	assertHelperFields(t, findHelperType(t, function.HelperTypes, "FunctionSourceDetails"), []string{"SourceType", "PbfListingId"})
	assertFieldType(t, function.SpecFields, "ProvisionedConcurrencyConfig", "FunctionProvisionedConcurrencyConfig")
	assertHelperFields(t, findHelperType(t, function.HelperTypes, "FunctionProvisionedConcurrencyConfig"), []string{"Strategy", "Count"})
}

func assertCoreComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	tunnel := findResource(t, pkg.Resources, "IPSecConnectionTunnel")
	assertFieldType(t, tunnel.SpecFields, "BgpSessionConfig", "IPSecConnectionTunnelBgpSessionConfig")
	assertFieldType(t, tunnel.SpecFields, "PhaseOneConfig", "IPSecConnectionTunnelPhaseOneConfig")
	assertFieldType(t, tunnel.SpecFields, "PhaseTwoConfig", "IPSecConnectionTunnelPhaseTwoConfig")
	assertHelperFields(t, findHelperType(t, tunnel.HelperTypes, "IPSecConnectionTunnelBgpSessionConfig"), []string{"CustomerBgpAsn"})
	assertHelperFields(t, findHelperType(t, tunnel.HelperTypes, "IPSecConnectionTunnelPhaseOneConfig"), []string{"DiffieHelmanGroup"})
}

func assertCertificatesComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	bundle := findResource(t, pkg.Resources, "CertificateBundle")
	assertFieldType(t, bundle.StatusFields, "Validity", "CertificateBundleValidity")
	assertFieldType(t, bundle.StatusFields, "RevocationStatus", "CertificateBundleRevocationStatus")
	assertHelperFields(t, findHelperType(t, bundle.HelperTypes, "CertificateBundleValidity"), []string{"TimeOfValidityNotBefore"})
	assertHelperFields(t, findHelperType(t, bundle.HelperTypes, "CertificateBundleRevocationStatus"), []string{"RevocationReason"})
}

func assertNoSQLComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	assertFieldType(t, findResource(t, pkg.Resources, "Row").SpecFields, "Value", "map[string]shared.JSONValue")
}

func assertSecretsComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	bundle := findResource(t, pkg.Resources, "SecretBundle")
	assertFieldType(t, bundle.StatusFields, "SecretBundleContent", "SecretBundleContent")
	assertHelperFields(t, findHelperType(t, bundle.HelperTypes, "SecretBundleContent"), []string{"ContentType", "Content"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "SecretBundleByName"), []string{"SecretId", "VersionNumber", "SecretBundleContent", "Metadata"})
}

func assertVaultComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	assertFieldType(t, findResource(t, pkg.Resources, "Secret").SpecFields, "Metadata", "map[string]shared.JSONValue")
}

func assertArtifactsComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	assertResourceStatusFields(t, findResource(t, pkg.Resources, "ContainerConfiguration"), []string{"IsRepositoryCreatedOnFirstPush"})
	containerImage := findResource(t, pkg.Resources, "ContainerImage")
	assertResourceStatusFields(t, containerImage, []string{"FreeformTags"})
	assertFieldType(t, containerImage.StatusFields, "DefinedTags", "map[string]shared.MapValue")
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "ContainerImageSignature"), []string{"CompartmentId", "ImageId", "Message", "Signature", "SigningAlgorithm"})
	containerRepository := findResource(t, pkg.Resources, "ContainerRepository")
	assertResourceStatusFields(t, containerRepository, []string{"CompartmentId", "DisplayName", "IsImmutable", "IsPublic", "FreeformTags", "DefinedTags"})
	assertFieldType(t, containerRepository.StatusFields, "Readme", "ContainerRepositoryReadme")
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "GenericArtifact"), []string{"FreeformTags"})
	assertResourceStatusFields(t, findResource(t, pkg.Resources, "Repository"), []string{"DisplayName", "Description", "CompartmentId", "IsImmutable", "FreeformTags", "DefinedTags"})
}

func assertNetworkLoadBalancerComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	healthChecker := findResource(t, pkg.Resources, "HealthChecker")
	assertFieldType(t, healthChecker.SpecFields, "RequestData", "string")
	assertFieldType(t, healthChecker.SpecFields, "ResponseData", "string")
}

func assertWidgetLifecycleSemantics(t *testing.T, semantics *RuntimeSemanticsModel) {
	t.Helper()

	assertStringSliceEqual(t, "Widget provisioning states", semantics.Lifecycle.ProvisioningStates, []string{"PROVISIONING"})
	assertStringSliceEqual(t, "Widget active states", semantics.Lifecycle.ActiveStates, []string{"ACTIVE"})
	if semantics.Delete.Policy != "required" {
		t.Fatalf("Widget delete policy = %q, want required", semantics.Delete.Policy)
	}
}

func assertWidgetListSemantics(t *testing.T, semantics *RuntimeSemanticsModel) {
	t.Helper()

	if semantics.List == nil || semantics.List.ResponseItemsField != "Items" {
		t.Fatalf("Widget list semantics = %#v, want responseItemsField Items", semantics.List)
	}
	assertStringSliceEqual(t, "Widget list match fields", semantics.List.MatchFields, []string{"compartmentId", "state"})
}

func assertWidgetMutationSemantics(t *testing.T, semantics *RuntimeSemanticsModel) {
	t.Helper()

	assertStringSliceEqual(t, "Widget forceNew", semantics.Mutation.ForceNew, []string{"compartmentId"})
}

func assertWidgetFollowUpSemantics(t *testing.T, semantics *RuntimeSemanticsModel) {
	t.Helper()

	if semantics.CreateFollowUp.Strategy != followUpStrategyReadAfterWrite {
		t.Fatalf("Widget create follow-up = %q, want %q", semantics.CreateFollowUp.Strategy, followUpStrategyReadAfterWrite)
	}
}

func assertWidgetOpenGaps(t *testing.T, semantics *RuntimeSemanticsModel) {
	t.Helper()

	if len(semantics.OpenGaps) != 0 {
		t.Fatalf("Widget open gaps = %#v, want none", semantics.OpenGaps)
	}
}

func assertStringSliceEqual(t *testing.T, label string, got []string, want []string) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
}

func sampleSDKDir(t *testing.T) string {
	t.Helper()

	return filepath.Join(generatorTestDir(t), "testdata", "sdk", "sample")
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
		Generation: GenerationConfig{
			Resources: []ResourceGenerationOverride{
				{
					Kind:    "MySqlDbSystem",
					SDKName: "DbSystem",
				},
			},
		},
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

func assertGeneratedGoMatches(t *testing.T, wantPath string, gotPath string) {
	t.Helper()

	want := normalizeGeneratedGo(t, readFile(t, wantPath))
	got := normalizeGeneratedGo(t, readFile(t, gotPath))
	if want != got {
		t.Fatalf("Go output mismatch for %s\nwant:\n%s\n\ngot:\n%s", wantPath, want, got)
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

func structFieldNames(t *testing.T, source string, typeName string) []string {
	t.Helper()

	_, file := parseTestGoFile(t, source)
	structType := findStructType(t, file, source, typeName)
	return structFieldNamesFromStruct(structType)
}

func parseTestGoFile(t *testing.T, source string) (*token.FileSet, *ast.File) {
	t.Helper()

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "", source, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse Go source error = %v\n%s", err, source)
	}

	return fileSet, file
}

func findStructType(t *testing.T, file *ast.File, source string, typeName string) *ast.StructType {
	t.Helper()

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != typeName {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				t.Fatalf("%s in source is %T, want struct\n%s", typeName, typeSpec.Type, source)
			}
			return structType
		}
	}

	t.Fatalf("struct type %q was not found in source:\n%s", typeName, source)
	return nil
}

func structFieldNamesFromStruct(structType *ast.StructType) []string {
	if structType.Fields == nil {
		return nil
	}

	names := make([]string, 0, len(structType.Fields.List))
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			if name := embeddedFieldName(field.Type); name != "" {
				names = append(names, name)
			}
			continue
		}
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}

	return names
}

func embeddedFieldName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return typed.Sel.Name
	case *ast.StarExpr:
		return embeddedFieldName(typed.X)
	default:
		return ""
	}
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

func normalizeGeneratedGo(t *testing.T, source string) string {
	t.Helper()

	source = strings.ReplaceAll(source, "// Code generated by generator. DO NOT EDIT.\n\n", "")

	fileSet, file := parseTestGoFile(t, source)

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
		stripGoDeclComments(decl)
	}
}

func stripGoDeclComments(decl ast.Decl) {
	switch concrete := decl.(type) {
	case *ast.GenDecl:
		stripGoGenDeclComments(concrete)
	case *ast.FuncDecl:
		concrete.Doc = nil
	}
}

func stripGoGenDeclComments(decl *ast.GenDecl) {
	decl.Doc = nil
	for _, spec := range decl.Specs {
		stripGoSpecComments(spec)
	}
}

func stripGoSpecComments(spec ast.Spec) {
	switch typed := spec.(type) {
	case *ast.TypeSpec:
		stripGoTypeSpecComments(typed)
	case *ast.ValueSpec:
		typed.Doc = nil
		typed.Comment = nil
	}
}

func stripGoTypeSpecComments(spec *ast.TypeSpec) {
	spec.Doc = nil
	spec.Comment = nil

	structType, ok := spec.Type.(*ast.StructType)
	if !ok {
		return
	}
	stripGoStructFieldComments(structType.Fields)
}

func stripGoStructFieldComments(fields *ast.FieldList) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		field.Doc = nil
		field.Comment = nil
	}
}
