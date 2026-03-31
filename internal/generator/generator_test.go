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

	assertDbSystemDiscovery(t, findResource(t, pkg.Resources, "DbSystem"))
	assertWidgetDiscovery(t, findResource(t, pkg.Resources, "Widget"))
	assertReportDiscovery(t, findResource(t, pkg.Resources, "Report"))
	assertReportByNameDiscovery(t, findResource(t, pkg.Resources, "ReportByName"))
	assertOAuthClientCredentialDiscovery(t, findResource(t, pkg.Resources, "OAuthClientCredential"))
}

func TestBuildPackageModelAttachesFormalModelFromResourceOverride(t *testing.T) {
	t.Parallel()

	pkg := buildWidgetFormalPackageModel(t)
	assertWidgetFormalAttachment(t, findResource(t, pkg.Resources, "Widget"))
	assertNoFormalAttachment(t, findResource(t, pkg.Resources, "Report"))
	assertWidgetServiceManagerFormalAttachment(t, findServiceManagerModel(t, pkg.ServiceManagers, "Widget"))
}

func TestBuildPackageModelDerivesRuntimeSemanticsFromFormalSpec(t *testing.T) {
	t.Parallel()

	pkg := buildWidgetFormalPackageModel(t)
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

	assertPSQLObservedStateFields(t, pkg)
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

	assertContainerEngineObservedStateAliases(t, pkg)
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

	assertIdentityObservedStateAliases(t, pkg)
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

	assertCoreObservedStateAliases(t, pkg)
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
	result := generateServices(t, pipeline, cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	})
	assertServiceResultCount(t, "generated", result.Generated, 1)
	assertFileContentContains(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "groupversion_info.go"), []string{
		"// Code generated by generator. DO NOT EDIT.",
		`GroupVersion = schema.GroupVersion{Group: "mysql.oracle.com", Version: "v1beta1"}`,
	})
	assertFileContentContains(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "dbsystem_types.go"), []string{
		"type DbSystemSpec struct",
		"Port",
		`json:"port,omitempty"`,
	})

	result = generateServices(t, pipeline, cfg, []ServiceConfig{service}, Options{
		OutputRoot:   outputRoot,
		SkipExisting: true,
	})
	assertServiceResultCount(t, "skipped", result.Skipped, 1)
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
	assertApplicationFieldDocumentation(t, application)

	content := renderResourceFileForTest(t, pkg, application)
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
				"- ../../../config/rbac/role_binding.yaml",
				"- ../../../config/rbac/leader_election_role.yaml",
				"- ../../../config/rbac/leader_election_role_binding.yaml",
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
				"role_binding.yaml",
				"leader_election_role.yaml",
				"leader_election_role_binding.yaml",
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
	service.Generation.Resources = []ResourceGenerationOverride{
		{
			Kind: "DbSystem",
			ServiceManager: ServiceManagerGenerationOverride{
				NeedsCredentialClient: true,
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

	serviceClientPath := filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "dbsystem", "dbsystem_serviceclient.go")
	serviceManagerPath := filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "dbsystem", "dbsystem_servicemanager.go")

	serviceClientContent := readFile(t, serviceClientPath)
	assertContains(t, serviceClientContent, []string{
		"package dbsystem",
		"type DbSystemServiceClient interface {",
		"var newDbSystemServiceClient = func(manager *DbSystemServiceManager) DbSystemServiceClient {",
		`Kind:`,
		`"DbSystem"`,
		`CredentialClient: manager.CredentialClient`,
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

	serviceClientContent := readFile(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "dbsystem", "dbsystem_serviceclient.go"))
	serviceManagerContent := readFile(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "dbsystem", "dbsystem_servicemanager.go"))

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

func TestCurrentServiceParityMatchesCheckedInArtifacts(t *testing.T) {
	cfg := loadCheckedInConfig(t)
	services := selectServicesByName(t, cfg, "database", "streaming")
	outputRoot := prepareGeneratedOutputRoot(t)
	pipeline := New()
	result := generateServices(t, pipeline, cfg, services, Options{
		OutputRoot:                      outputRoot,
		PreserveExistingSpecSurfaceRoot: repoRoot(t),
	})
	assertGeneratedServiceCounts(t, result.Generated, map[string]int{
		"database":  79,
		"streaming": 7,
	})

	apiFiles := []string{
		"api/database/v1beta1/groupversion_info.go",
		"api/database/v1beta1/autonomousdatabase_types.go",
		"api/database/v1beta1/autonomousdatabasebackup_types.go",
		"api/streaming/v1beta1/groupversion_info.go",
		"api/streaming/v1beta1/stream_types.go",
		"api/streaming/v1beta1/streampool_types.go",
	}
	assertGoParityFiles(t, outputRoot, apiFiles)

	exactFiles := []string{
		"config/samples/database_v1beta1_autonomousdatabase.yaml",
		"config/samples/streaming_v1beta1_stream.yaml",
		"packages/database/metadata.env",
		"packages/database/install/kustomization.yaml",
		"packages/streaming/metadata.env",
		"packages/streaming/install/kustomization.yaml",
	}
	assertExactFileMatches(t, outputRoot, exactFiles)

	runtimeFiles := []string{
		"controllers/database/autonomousdatabase_controller.go",
		"controllers/database/autonomousdatabasebackup_controller.go",
		"controllers/streaming/stream_controller.go",
		"controllers/streaming/streampool_controller.go",
		"pkg/servicemanager/database/autonomousdatabase/autonomousdatabase_serviceclient.go",
		"pkg/servicemanager/database/autonomousdatabase/autonomousdatabase_servicemanager.go",
		"pkg/servicemanager/database/autonomousdatabasebackup/autonomousdatabasebackup_serviceclient.go",
		"pkg/servicemanager/database/autonomousdatabasebackup/autonomousdatabasebackup_servicemanager.go",
		"pkg/servicemanager/streaming/stream/stream_serviceclient.go",
		"pkg/servicemanager/streaming/stream/stream_servicemanager.go",
		"pkg/servicemanager/streaming/streampool/streampool_serviceclient.go",
		"pkg/servicemanager/streaming/streampool/streampool_servicemanager.go",
		"internal/registrations/database_generated.go",
		"internal/registrations/streaming_generated.go",
	}
	assertGoParityFiles(t, outputRoot, runtimeFiles)

	assertResourceOrderContainsSubset(
		t,
		filepath.Join(repoRoot(t), "config", "samples", "kustomization.yaml"),
		filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"),
	)
}

func TestCheckedInGeneratedMySQLArtifactsMatchGenerator(t *testing.T) {
	cfg := loadCheckedInConfig(t)
	mysqlService := requireServiceConfig(t, cfg, "mysql")
	outputRoot := prepareGeneratedOutputRoot(t)

	pipeline := New()
	result := generateServices(t, pipeline, cfg, []ServiceConfig{mysqlService}, Options{
		OutputRoot:                      outputRoot,
		PreserveExistingSpecSurfaceRoot: repoRoot(t),
	})
	assertGeneratedServiceCounts(t, result.Generated, map[string]int{"mysql": 12})

	apiFiles := []string{
		"api/mysql/v1beta1/groupversion_info.go",
		"api/mysql/v1beta1/dbsystem_types.go",
		"api/mysql/v1beta1/backup_types.go",
	}
	assertGoParityFiles(t, outputRoot, apiFiles)

	exactFiles := []string{
		"config/samples/mysql_v1beta1_dbsystem.yaml",
		"packages/mysql/metadata.env",
		"packages/mysql/install/kustomization.yaml",
	}
	assertExactFileMatches(t, outputRoot, exactFiles)

	runtimeFiles := []string{
		"controllers/mysql/dbsystem_controller.go",
		"controllers/mysql/backup_controller.go",
		"pkg/servicemanager/mysql/dbsystem/dbsystem_serviceclient.go",
		"pkg/servicemanager/mysql/dbsystem/dbsystem_servicemanager.go",
		"pkg/servicemanager/mysql/backup/backup_serviceclient.go",
		"pkg/servicemanager/mysql/backup/backup_servicemanager.go",
		"internal/registrations/mysql_generated.go",
	}
	assertGoParityFiles(t, outputRoot, runtimeFiles)

	assertResourceOrderContainsSubset(
		t,
		filepath.Join(repoRoot(t), "config", "samples", "kustomization.yaml"),
		filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"),
	)
}

func TestCheckedInPromotedCoreRuntimeArtifactsMatchGenerator(t *testing.T) {
	cfg := loadCheckedInConfig(t)
	coreService := requireServiceConfig(t, cfg, "core")
	outputRoot := prepareGeneratedOutputRoot(t)

	pipeline := New()
	result := generateServices(t, pipeline, cfg, []ServiceConfig{coreService}, Options{
		OutputRoot:                      outputRoot,
		PreserveExistingSpecSurfaceRoot: repoRoot(t),
	})
	assertServiceResultCount(t, "generated", result.Generated, 1)

	runtimeFiles := []string{
		"controllers/core/vcn_controller.go",
		"pkg/servicemanager/core/vcn/vcn_serviceclient.go",
		"pkg/servicemanager/core/vcn/vcn_servicemanager.go",
		"internal/registrations/core_generated.go",
	}
	assertGoParityFiles(t, outputRoot, runtimeFiles)

	exactFiles := []string{
		"packages/core/metadata.env",
		"packages/core/install/kustomization.yaml",
	}
	assertExactFileMatches(t, outputRoot, exactFiles)
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
		OutputRoot:                      outputRoot,
		PreserveExistingSpecSurfaceRoot: repoRoot(t),
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
		"AdminUsername *shared.UsernameSource `json:\"adminUsername,omitempty\"`",
		"AdminPassword *shared.PasswordSource `json:\"adminPassword,omitempty\"`",
		"DeletionPolicy DbSystemDeletionPolicy `json:\"deletionPolicy,omitempty\"`",
		"CrashRecovery string `json:\"crashRecovery,omitempty\"`",
		"DatabaseManagement string `json:\"databaseManagement,omitempty\"`",
		"SecureConnections DbSystemSecureConnections `json:\"secureConnections,omitempty\"`",
	})
	if count := strings.Count(normalized, "AdminUsername *shared.UsernameSource `json:\"adminUsername,omitempty\"`"); count != 2 {
		t.Fatalf("generated mysql DbSystem adminUsername pointer field count = %d, want spec and status entries", count)
	}
	if count := strings.Count(normalized, "AdminPassword *shared.PasswordSource `json:\"adminPassword,omitempty\"`"); count != 2 {
		t.Fatalf("generated mysql DbSystem adminPassword pointer field count = %d, want spec and status entries", count)
	}
}

func TestMySQLDbSystemSourceVariantFieldsAreOptional(t *testing.T) {
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

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, *mysqlService)
	if err != nil {
		t.Fatalf("BuildPackageModel() error = %v", err)
	}

	dbSystem := findResource(t, pkg.Resources, "DbSystem")
	source := findHelperType(t, dbSystem.HelperTypes, "DbSystemSource")

	for _, fieldName := range []string{"BackupId", "DbSystemId", "SourceUrl"} {
		field := findFieldModel(t, source.Fields, fieldName)
		if !slices.Equal(field.Markers, []string{"+kubebuilder:validation:Optional"}) {
			t.Fatalf("DbSystemSource.%s markers = %#v, want optional marker", fieldName, field.Markers)
		}
		if !strings.HasSuffix(field.Tag, ",omitempty\"") {
			t.Fatalf("DbSystemSource.%s tag = %q, want omitempty json tag", fieldName, field.Tag)
		}
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
		"- mysql_v1beta1_dbsystem.yaml",
	})
	if strings.Index(sampleKustomization, "- existing.yaml") > strings.Index(sampleKustomization, "- mysql_v1beta1_dbsystem.yaml") {
		t.Fatalf("existing sample entry was not preserved ahead of the generated sample:\n%s", sampleKustomization)
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

func assertDbSystemDiscovery(t *testing.T, resource ResourceModel) {
	t.Helper()

	if resource.SDKName != "DbSystem" {
		t.Fatalf("DbSystem SDK name = %q, want %q", resource.SDKName, "DbSystem")
	}
	assertResourceHasSpecFields(t, resource, "Port")
	assertResourceLacksSpecFields(t, resource, "Id")
	if resource.PrimaryDisplayField != "DisplayName" {
		t.Fatalf("DbSystem primary display field = %q, want DisplayName", resource.PrimaryDisplayField)
	}
}

func assertWidgetDiscovery(t *testing.T, resource ResourceModel) {
	t.Helper()

	if len(resource.Operations) != 5 {
		t.Fatalf("Widget operations = %v, want 5 CRUD operations", resource.Operations)
	}
	assertResourceHasSpecFields(t, resource, "Mode", "CreatedAt")
	assertResourceLacksSpecFields(t, resource, "LifecycleState", "TimeUpdated")
	assertResourceHasStatusFields(t, resource, "LifecycleState", "TimeUpdated")

	compartmentID := findFieldModel(t, resource.SpecFields, "CompartmentId")
	assertFieldTag(t, "Widget CompartmentId", compartmentID, `json:"compartmentId"`)
	assertFieldCommentsEqual(t, "Widget CompartmentId", compartmentID, []string{"The OCID of the widget compartment."})
	assertFieldMarkersEqual(t, "Widget CompartmentId", compartmentID, []string{"+kubebuilder:validation:Required"})

	labels := findFieldModel(t, resource.SpecFields, "Labels")
	assertFieldTag(t, "Widget Labels", labels, `json:"labels,omitempty"`)
	assertFieldCommentsEqual(t, "Widget Labels", labels, []string{"Additional labels for the widget."})
	assertFieldMarkersEqual(t, "Widget Labels", labels, []string{"+kubebuilder:validation:Optional"})

	serverState := findFieldModel(t, resource.SpecFields, "ServerState")
	assertFieldTag(t, "Widget ServerState", serverState, `json:"serverState,omitempty"`)
	assertNoFieldMarkers(t, "Widget ServerState", serverState)

	lifecycleState := findFieldModel(t, resource.StatusFields, "LifecycleState")
	assertFieldCommentsEqual(t, "Widget LifecycleState", lifecycleState, []string{"The lifecycle state of the widget."})
	assertNoFieldMarkers(t, "Widget LifecycleState", lifecycleState)
}

func assertReportDiscovery(t *testing.T, resource ResourceModel) {
	t.Helper()

	if len(resource.SpecFields) != 0 {
		t.Fatalf("Report spec fields = %#v, want empty spec when no create or update payload exists", resource.SpecFields)
	}
	assertResourceHasStatusFields(t, resource, "Id", "LifecycleState", "DisplayName")
}

func assertReportByNameDiscovery(t *testing.T, resource ResourceModel) {
	t.Helper()
	assertResourceHasSpecFields(t, resource, "DisplayName")
}

func assertOAuthClientCredentialDiscovery(t *testing.T, resource ResourceModel) {
	t.Helper()
	assertResourceHasSpecFields(t, resource, "Name", "Description", "Scopes")
}

func buildWidgetFormalPackageModel(t *testing.T) *PackageModel {
	t.Helper()

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

	return pkg
}

func assertWidgetFormalAttachment(t *testing.T, resource ResourceModel) {
	t.Helper()

	if resource.Formal == nil {
		t.Fatal("Widget formal model was not attached")
	}
	if resource.Formal.Reference.Service != "mysql" {
		t.Fatalf("Widget formal service = %q, want %q", resource.Formal.Reference.Service, "mysql")
	}
	if resource.Formal.Reference.Slug != "widget" {
		t.Fatalf("Widget formal slug = %q, want %q", resource.Formal.Reference.Slug, "widget")
	}
	if resource.Formal.Binding.Import.ProviderResource != "widget_resource" {
		t.Fatalf("Widget provider resource = %q, want %q", resource.Formal.Binding.Import.ProviderResource, "widget_resource")
	}
	if resource.Formal.Binding.Spec.Kind != "Widget" {
		t.Fatalf("Widget formal kind = %q, want %q", resource.Formal.Binding.Spec.Kind, "Widget")
	}
	if resource.Formal.Diagrams.ActivitySourcePath != "controllers/mysql/widget/diagrams/activity.puml" {
		t.Fatalf("Widget activity diagram path = %q, want %q", resource.Formal.Diagrams.ActivitySourcePath, "controllers/mysql/widget/diagrams/activity.puml")
	}
}

func assertNoFormalAttachment(t *testing.T, resource ResourceModel) {
	t.Helper()

	if resource.Formal != nil {
		t.Fatalf("%s formal model = %#v, want nil", resource.Kind, resource.Formal)
	}
}

func assertWidgetServiceManagerFormalAttachment(t *testing.T, serviceManager ServiceManagerModel) {
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
	assertStringSliceEqual(t, "Widget provisioning states", semantics.Lifecycle.ProvisioningStates, []string{"PROVISIONING"})
	assertStringSliceEqual(t, "Widget active states", semantics.Lifecycle.ActiveStates, []string{"ACTIVE"})
	if semantics.Delete.Policy != "required" {
		t.Fatalf("Widget delete policy = %q, want required", semantics.Delete.Policy)
	}
	if semantics.List == nil {
		t.Fatal("Widget list semantics were not attached")
	}
	if semantics.List.ResponseItemsField != "Items" {
		t.Fatalf("Widget responseItemsField = %q, want %q", semantics.List.ResponseItemsField, "Items")
	}
	assertStringSliceEqual(t, "Widget list match fields", semantics.List.MatchFields, []string{"compartmentId", "state"})
	assertStringSliceEqual(t, "Widget forceNew", semantics.Mutation.ForceNew, []string{"compartmentId"})
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

func assertFunctionsComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	application := findResource(t, pkg.Resources, "Application")
	assertFieldType(t, "Application TraceConfig", findFieldModel(t, application.SpecFields, "TraceConfig"), "ApplicationTraceConfig")
	assertFieldType(t, "Application ImagePolicyConfig", findFieldModel(t, application.SpecFields, "ImagePolicyConfig"), "ApplicationImagePolicyConfig")
	assertFieldType(t, "Application DefinedTags", findFieldModel(t, application.SpecFields, "DefinedTags"), "map[string]shared.MapValue")
	assertHelperTypeHasFields(t, findHelperType(t, application.HelperTypes, "ApplicationTraceConfig"), "DomainId")
	assertHelperTypeHasFields(t, findHelperType(t, application.HelperTypes, "ApplicationImagePolicyConfig"), "IsPolicyEnabled")

	function := findResource(t, pkg.Resources, "Function")
	assertFieldType(t, "Function SourceDetails", findFieldModel(t, function.SpecFields, "SourceDetails"), "FunctionSourceDetails")
	assertHelperTypeHasFields(t, findHelperType(t, function.HelperTypes, "FunctionSourceDetails"), "SourceType", "PbfListingId")
	assertFieldType(t, "Function ProvisionedConcurrencyConfig", findFieldModel(t, function.SpecFields, "ProvisionedConcurrencyConfig"), "FunctionProvisionedConcurrencyConfig")
	assertHelperTypeHasFields(t, findHelperType(t, function.HelperTypes, "FunctionProvisionedConcurrencyConfig"), "Strategy", "Count")
}

func assertCoreComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	tunnel := findResource(t, pkg.Resources, "IPSecConnectionTunnel")
	assertFieldType(t, "IPSecConnectionTunnel BgpSessionConfig", findFieldModel(t, tunnel.SpecFields, "BgpSessionConfig"), "IPSecConnectionTunnelBgpSessionConfig")
	assertFieldType(t, "IPSecConnectionTunnel PhaseOneConfig", findFieldModel(t, tunnel.SpecFields, "PhaseOneConfig"), "IPSecConnectionTunnelPhaseOneConfig")
	assertFieldType(t, "IPSecConnectionTunnel PhaseTwoConfig", findFieldModel(t, tunnel.SpecFields, "PhaseTwoConfig"), "IPSecConnectionTunnelPhaseTwoConfig")
	assertHelperTypeHasFields(t, findHelperType(t, tunnel.HelperTypes, "IPSecConnectionTunnelBgpSessionConfig"), "CustomerBgpAsn")
	assertHelperTypeHasFields(t, findHelperType(t, tunnel.HelperTypes, "IPSecConnectionTunnelPhaseOneConfig"), "DiffieHelmanGroup")
}

func assertCertificatesComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	bundle := findResource(t, pkg.Resources, "CertificateBundle")
	assertFieldType(t, "CertificateBundle Validity", findFieldModel(t, bundle.StatusFields, "Validity"), "CertificateBundleValidity")
	assertFieldType(t, "CertificateBundle RevocationStatus", findFieldModel(t, bundle.StatusFields, "RevocationStatus"), "CertificateBundleRevocationStatus")
	assertHelperTypeHasFields(t, findHelperType(t, bundle.HelperTypes, "CertificateBundleValidity"), "TimeOfValidityNotBefore")
	assertHelperTypeHasFields(t, findHelperType(t, bundle.HelperTypes, "CertificateBundleRevocationStatus"), "RevocationReason")
}

func assertNoSQLComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()
	assertFieldType(t, "Row Value", findFieldModel(t, findResource(t, pkg.Resources, "Row").SpecFields, "Value"), "map[string]shared.JSONValue")
}

func assertSecretsComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	bundle := findResource(t, pkg.Resources, "SecretBundle")
	assertFieldType(t, "SecretBundle SecretBundleContent", findFieldModel(t, bundle.StatusFields, "SecretBundleContent"), "SecretBundleContent")
	assertHelperTypeHasFields(t, findHelperType(t, bundle.HelperTypes, "SecretBundleContent"), "ContentType", "Content")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "SecretBundleByName"), "SecretId", "VersionNumber", "SecretBundleContent", "Metadata")
}

func assertVaultComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()
	assertFieldType(t, "Secret Metadata", findFieldModel(t, findResource(t, pkg.Resources, "Secret").SpecFields, "Metadata"), "map[string]shared.JSONValue")
}

func assertArtifactsComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "ContainerConfiguration"), "IsRepositoryCreatedOnFirstPush")

	containerImage := findResource(t, pkg.Resources, "ContainerImage")
	assertResourceHasStatusFields(t, containerImage, "FreeformTags")
	assertFieldType(t, "ContainerImage DefinedTags", findFieldModel(t, containerImage.StatusFields, "DefinedTags"), "map[string]shared.MapValue")

	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "ContainerImageSignature"), "CompartmentId", "ImageId", "Message", "Signature", "SigningAlgorithm")

	containerRepository := findResource(t, pkg.Resources, "ContainerRepository")
	assertResourceHasStatusFields(t, containerRepository, "CompartmentId", "DisplayName", "IsImmutable", "IsPublic", "FreeformTags", "DefinedTags")
	assertFieldType(t, "ContainerRepository Readme", findFieldModel(t, containerRepository.StatusFields, "Readme"), "ContainerRepositoryReadme")

	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "GenericArtifact"), "FreeformTags")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "Repository"), "DisplayName", "Description", "CompartmentId", "IsImmutable", "FreeformTags", "DefinedTags")
}

func assertNetworkLoadBalancerComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	healthChecker := findResource(t, pkg.Resources, "HealthChecker")
	assertFieldType(t, "HealthChecker RequestData", findFieldModel(t, healthChecker.SpecFields, "RequestData"), "string")
	assertFieldType(t, "HealthChecker ResponseData", findFieldModel(t, healthChecker.SpecFields, "ResponseData"), "string")
}

func assertPSQLObservedStateFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "DbSystem"), "DisplayName", "CompartmentId", "Shape", "DbVersion")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "Configuration"), "DisplayName", "Shape", "DbVersion", "InstanceOcpuCount")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "Backup"), "DisplayName", "CompartmentId", "DbSystemId", "RetentionPeriod")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "PrimaryDbInstance"), "DbInstanceId")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "WorkRequestLog"), "Message", "Timestamp")
}

func assertContainerEngineObservedStateAliases(t *testing.T, pkg *PackageModel) {
	t.Helper()

	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "Cluster"), "Name", "CompartmentId", "EndpointConfig", "VcnId", "KubernetesVersion", "KmsKeyId", "FreeformTags", "DefinedTags", "Options", "ImagePolicyConfig", "ClusterPodNetworkOptions", "Type")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "NodePool"), "CompartmentId", "ClusterId", "Name", "KubernetesVersion", "NodeMetadata", "NodeImageName", "NodeSourceDetails", "NodeShapeConfig", "InitialNodeLabels", "SshPublicKey", "QuantityPerSubnet", "SubnetIds", "NodeConfigDetails", "FreeformTags", "DefinedTags", "NodeEvictionNodePoolSettings", "NodePoolCyclingDetails")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "VirtualNodePool"), "CompartmentId", "ClusterId", "DisplayName", "PlacementConfigurations", "InitialVirtualNodeLabels", "Taints", "Size", "NsgIds", "PodConfiguration", "FreeformTags", "DefinedTags", "VirtualNodeTags")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "Addon"), "Version", "Configurations")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "WorkloadMapping"), "Namespace", "MappedCompartmentId", "FreeformTags", "DefinedTags")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "WorkRequestLog"), "Message", "Timestamp")
	assertFieldTag(t, "WorkRequest Status", findFieldModel(t, findResource(t, pkg.Resources, "WorkRequest").StatusFields, "Status"), `json:"sdkStatus,omitempty"`)
	assertFieldTag(t, "CredentialRotationStatus Status", findFieldModel(t, findResource(t, pkg.Resources, "CredentialRotationStatus").StatusFields, "Status"), `json:"sdkStatus,omitempty"`)
}

func assertIdentityObservedStateAliases(t *testing.T, pkg *PackageModel) {
	t.Helper()

	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "BulkActionResourceType"), "Items")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "BulkEditTagsResourceType"), "Items")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "CostTrackingTag"), "TagNamespaceId", "TagNamespaceName", "IsRetired", "Validator")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "IdentityProvider"), "CompartmentId", "Name", "Description", "Metadata", "MetadataUrl", "ProductType")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "NetworkSource"), "CompartmentId", "Name", "Description", "PublicSourceList", "Services", "VirtualSourceList")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "OrResetUIPassword"), "Password", "UserId", "TimeCreated", "LifecycleState", "InactiveStatus")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "StandardTagNamespace"), "Description", "StandardTagNamespaceName", "TagDefinitionTemplates")
	assertFieldTag(t, "StandardTagNamespace Status", findFieldModel(t, findResource(t, pkg.Resources, "StandardTagNamespace").StatusFields, "Status"), `json:"sdkStatus,omitempty"`)
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "StandardTagTemplate"), "Description", "TagDefinitionName", "Type", "IsCostTracking")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "UserState"), "Id", "CompartmentId", "Name", "LifecycleState", "Capabilities")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "UserUIPasswordInformation"), "UserId", "TimeCreated", "LifecycleState")
}

func assertCoreObservedStateAliases(t *testing.T, pkg *PackageModel) {
	t.Helper()

	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "ClusterNetworkInstance"), "AvailabilityDomain", "CompartmentId", "Region", "State", "TimeCreated")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "ComputeCapacityReservationInstance"), "AvailabilityDomain", "CompartmentId", "Id", "Shape")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "ComputeGlobalImageCapabilitySchema"), "ComputeGlobalImageCapabilitySchemaId", "Name")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "NetworkSecurityGroupSecurityRule"), "Direction", "Protocol", "Id", "TcpOptions", "UdpOptions")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "IPSecConnectionTunnelError"), "ErrorCode", "ErrorDescription", "Id", "Solution", "Timestamp")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "IPSecConnectionTunnelRoute"), "Advertiser", "AsPath", "IsBestPath", "Prefix")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "IPSecConnectionTunnelSecurityAssociation"), "CpeSubnet", "OracleSubnet", "TunnelSaStatus")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "InstanceDevice"), "IsAvailable", "Name")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "VolumeBackupPolicyAssetAssignment"), "AssetId", "Id", "PolicyId", "TimeCreated")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "WindowsInstanceInitialCredential"), "Password", "Username")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "FastConnectProviderVirtualCircuitBandwidthShape"), "BandwidthInMbps", "Name")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "CrossconnectPortSpeedShape"), "Name", "PortSpeedInGbps")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "AllDrgAttachment"), "Id")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "AllowedPeerRegionsForRemotePeering"), "Name")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "AppCatalogListingAgreement"), "ListingId", "ListingResourceVersion", "OracleTermsOfUseLink", "EulaLink", "TimeRetrieved", "Signature")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "CrossConnectLetterOfAuthority"), "CrossConnectId", "FacilityLocation", "TimeExpires")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "CrossConnectMapping"), "Ipv4BgpStatus", "Ipv6BgpStatus", "OciLogicalDeviceName")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "DhcpOption"), "CompartmentId", "DisplayName", "LifecycleState", "Options", "TimeCreated", "VcnId")
	assertResourceHasStatusFields(t, findResource(t, pkg.Resources, "VirtualCircuitAssociatedTunnel"), "TunnelId", "TunnelType")
}

func assertApplicationFieldDocumentation(t *testing.T, resource ResourceModel) {
	t.Helper()

	compartmentID := findFieldModel(t, resource.SpecFields, "CompartmentId")
	assertFieldMarkersEqual(t, "Application CompartmentId", compartmentID, []string{"+kubebuilder:validation:Required"})
	assertFieldTag(t, "Application CompartmentId", compartmentID, `json:"compartmentId"`)
	assertFieldCommentsContain(t, "Application CompartmentId", compartmentID, "compartment to create the application within")

	config := findFieldModel(t, resource.SpecFields, "Config")
	assertFieldMarkersEqual(t, "Application Config", config, []string{"+kubebuilder:validation:Optional"})
	assertFieldTag(t, "Application Config", config, `json:"config,omitempty"`)
	assertFieldCommentsContain(t, "Application Config", config, "Application configuration")

	lifecycleState := findFieldModel(t, resource.StatusFields, "LifecycleState")
	assertNoFieldMarkers(t, "Application LifecycleState", lifecycleState)
	assertFieldCommentsContain(t, "Application LifecycleState", lifecycleState, "current state of the application")
}

func renderResourceFileForTest(t *testing.T, pkg *PackageModel, resource ResourceModel) string {
	t.Helper()

	content, err := renderResourceFile(pkg, resource)
	if err != nil {
		t.Fatalf("renderResourceFile() error = %v", err)
	}

	return content
}

func generateServices(t *testing.T, pipeline *Generator, cfg *Config, services []ServiceConfig, options Options) RunResult {
	t.Helper()

	result, err := pipeline.Generate(context.Background(), cfg, services, options)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	return result
}

func assertServiceResultCount(t *testing.T, label string, results []ServiceResult, want int) {
	t.Helper()

	if len(results) != want {
		t.Fatalf("%s results = %d services, want %d", label, len(results), want)
	}
}

func assertGeneratedServiceCounts(t *testing.T, results []ServiceResult, want map[string]int) {
	t.Helper()

	if len(results) != len(want) {
		t.Fatalf("generated %d services, want %d", len(results), len(want))
	}
	for _, result := range results {
		wantCount, ok := want[result.Service]
		if !ok {
			t.Fatalf("unexpected generated service %q", result.Service)
		}
		if result.ResourceCount != wantCount {
			t.Fatalf("service %s generated %d resources, want %d", result.Service, result.ResourceCount, wantCount)
		}
	}
}

func assertFileContentContains(t *testing.T, path string, want []string) {
	t.Helper()
	assertContains(t, readFile(t, path), want)
}

func loadCheckedInConfig(t *testing.T) *Config {
	t.Helper()

	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	return cfg
}

func requireServiceConfig(t *testing.T, cfg *Config, name string) ServiceConfig {
	t.Helper()

	for _, service := range cfg.Services {
		if service.Service == name {
			return service
		}
	}

	t.Fatalf("%s service was not found in services.yaml", name)
	return ServiceConfig{}
}

func selectServicesByName(t *testing.T, cfg *Config, names ...string) []ServiceConfig {
	t.Helper()

	services := make([]ServiceConfig, 0, len(names))
	for _, name := range names {
		services = append(services, requireServiceConfig(t, cfg, name))
	}

	return services
}

func prepareGeneratedOutputRoot(t *testing.T) string {
	t.Helper()

	outputRoot := t.TempDir()
	seedSampleKustomization(t, outputRoot)

	return outputRoot
}

func seedSampleKustomization(t *testing.T, outputRoot string) {
	t.Helper()

	samplesDir := filepath.Join(outputRoot, "config", "samples")
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", samplesDir, err)
	}
	if err := os.WriteFile(
		filepath.Join(samplesDir, "kustomization.yaml"),
		[]byte(readFile(t, filepath.Join(repoRoot(t), "config", "samples", "kustomization.yaml"))),
		0o644,
	); err != nil {
		t.Fatalf("write seeded samples kustomization: %v", err)
	}
}

func assertGoParityFiles(t *testing.T, outputRoot string, relativePaths []string) {
	t.Helper()

	for _, relativePath := range relativePaths {
		assertGeneratedGoMatches(t, filepath.Join(repoRoot(t), relativePath), filepath.Join(outputRoot, relativePath))
	}
}

func assertExactFileMatches(t *testing.T, outputRoot string, relativePaths []string) {
	t.Helper()

	for _, relativePath := range relativePaths {
		assertExactFileMatch(t, filepath.Join(repoRoot(t), relativePath), filepath.Join(outputRoot, relativePath))
	}
}

func assertResourceHasSpecFields(t *testing.T, resource ResourceModel, fields ...string) {
	t.Helper()
	assertFieldsPresent(t, resource.Kind+" spec", resource.SpecFields, fields...)
}

func assertResourceLacksSpecFields(t *testing.T, resource ResourceModel, fields ...string) {
	t.Helper()
	assertFieldsAbsent(t, resource.Kind+" spec", resource.SpecFields, fields...)
}

func assertResourceHasStatusFields(t *testing.T, resource ResourceModel, fields ...string) {
	t.Helper()
	assertFieldsPresent(t, resource.Kind+" status", resource.StatusFields, fields...)
}

func assertHelperTypeHasFields(t *testing.T, helperType TypeModel, fields ...string) {
	t.Helper()
	assertFieldsPresent(t, helperType.Name, helperType.Fields, fields...)
}

func assertFieldsPresent(t *testing.T, owner string, fields []FieldModel, want ...string) {
	t.Helper()

	for _, fieldName := range want {
		if !hasField(fields, fieldName) {
			t.Fatalf("%s fields = %#v, want %s", owner, fields, fieldName)
		}
	}
}

func assertFieldsAbsent(t *testing.T, owner string, fields []FieldModel, want ...string) {
	t.Helper()

	for _, fieldName := range want {
		if hasField(fields, fieldName) {
			t.Fatalf("%s fields = %#v, want no %s field", owner, fields, fieldName)
		}
	}
}

func assertFieldType(t *testing.T, owner string, field FieldModel, want string) {
	t.Helper()

	if field.Type != want {
		t.Fatalf("%s type = %q, want %q", owner, field.Type, want)
	}
}

func assertFieldTag(t *testing.T, owner string, field FieldModel, want string) {
	t.Helper()

	if field.Tag != want {
		t.Fatalf("%s tag = %q, want %q", owner, field.Tag, want)
	}
}

func assertFieldMarkersEqual(t *testing.T, owner string, field FieldModel, want []string) {
	t.Helper()

	if !slices.Equal(field.Markers, want) {
		t.Fatalf("%s markers = %#v, want %#v", owner, field.Markers, want)
	}
}

func assertNoFieldMarkers(t *testing.T, owner string, field FieldModel) {
	t.Helper()

	if len(field.Markers) != 0 {
		t.Fatalf("%s markers = %#v, want none", owner, field.Markers)
	}
}

func assertFieldCommentsEqual(t *testing.T, owner string, field FieldModel, want []string) {
	t.Helper()

	if !slices.Equal(field.Comments, want) {
		t.Fatalf("%s comments = %#v, want %#v", owner, field.Comments, want)
	}
}

func assertFieldCommentsContain(t *testing.T, owner string, field FieldModel, want string) {
	t.Helper()

	if !strings.Contains(strings.Join(field.Comments, "\n"), want) {
		t.Fatalf("%s comments = %#v, want substring %q", owner, field.Comments, want)
	}
}

func assertStringSliceEqual(t *testing.T, owner string, got []string, want []string) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("%s = %v, want %v", owner, got, want)
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

func parseTestGoFile(t *testing.T, source string) (*token.FileSet, *ast.File) {
	t.Helper()

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "generated.go", source, parser.ParseComments)
	if err != nil {
		t.Fatalf("parser.ParseFile() error = %v\nsource:\n%s", err, source)
	}

	return fileSet, file
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

func stripTypeSpecComments(spec *ast.TypeSpec) {
	spec.Doc = nil
	spec.Comment = nil
	stripStructFieldComments(spec.Type)
}

func stripStructFieldComments(expr ast.Expr) {
	structType, ok := expr.(*ast.StructType)
	if !ok || structType.Fields == nil {
		return
	}

	for _, field := range structType.Fields.List {
		field.Doc = nil
		field.Comment = nil
	}
}
