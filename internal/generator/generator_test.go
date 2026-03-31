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
		Compatibility: CompatibilityConfig{
			ExistingKinds: []string{"MySqlDbSystem"},
		},
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

	assertEqual(t, "BuildPackageModel() group DNS name", pkg.GroupDNSName, "mysql.oracle.com")

	dbSystem := findResource(t, pkg.Resources, "MySqlDbSystem")
	assertEqual(t, "MySqlDbSystem SDK name", dbSystem.SDKName, "DbSystem")
	assertEqual(t, "MySqlDbSystem compatibility override", dbSystem.CompatibilityLocked, true)
	assertFieldsPresent(t, "MySqlDbSystem spec", dbSystem.SpecFields, []string{"Port"})
	assertFieldsAbsent(t, "MySqlDbSystem spec", dbSystem.SpecFields, []string{"Id"})
	assertEqual(t, "MySqlDbSystem primary display field", dbSystem.PrimaryDisplayField, "DisplayName")

	widget := findResource(t, pkg.Resources, "Widget")
	assertEqual(t, "Widget operations", len(widget.Operations), 5)
	assertFieldsPresent(t, "Widget spec", widget.SpecFields, []string{"Mode", "CreatedAt"})
	assertFieldsAbsent(t, "Widget spec", widget.SpecFields, []string{"LifecycleState", "TimeUpdated"})
	assertFieldsPresent(t, "Widget status", widget.StatusFields, []string{"LifecycleState", "TimeUpdated"})

	compartmentID := findFieldModel(t, widget.SpecFields, "CompartmentId")
	assertFieldMatches(t, "Widget CompartmentId", compartmentID, fieldExpectation{
		Tag:      `json:"compartmentId"`,
		Comments: []string{"The OCID of the widget compartment."},
		Markers:  []string{"+kubebuilder:validation:Required"},
	})

	labels := findFieldModel(t, widget.SpecFields, "Labels")
	assertFieldMatches(t, "Widget Labels", labels, fieldExpectation{
		Tag:      `json:"labels,omitempty"`,
		Comments: []string{"Additional labels for the widget."},
		Markers:  []string{"+kubebuilder:validation:Optional"},
	})

	serverState := findFieldModel(t, widget.SpecFields, "ServerState")
	assertFieldMatches(t, "Widget ServerState", serverState, fieldExpectation{
		Tag:     `json:"serverState,omitempty"`,
		Markers: []string{},
	})

	lifecycleState := findFieldModel(t, widget.StatusFields, "LifecycleState")
	assertFieldMatches(t, "Widget LifecycleState", lifecycleState, fieldExpectation{
		Comments: []string{"The lifecycle state of the widget."},
		Markers:  []string{},
	})

	report := findResource(t, pkg.Resources, "Report")
	assertNoFields(t, "Report spec", report.SpecFields)
	assertFieldsPresent(t, "Report status", report.StatusFields, []string{"Id", "LifecycleState", "DisplayName"})

	reportByName := findResource(t, pkg.Resources, "ReportByName")
	assertFieldsPresent(t, "ReportByName spec", reportByName.SpecFields, []string{"DisplayName"})

	oauthClientCredential := findResource(t, pkg.Resources, "OAuthClientCredential")
	assertFieldsPresent(t, "OAuthClientCredential spec", oauthClientCredential.SpecFields, []string{"Name", "Description", "Scopes"})
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

	widget := findResource(t, pkg.Resources, "Widget")
	if widget.Formal == nil {
		t.Fatal("Widget formal model was not attached")
	}
	assertEqual(t, "Widget formal service", widget.Formal.Reference.Service, "mysql")
	assertEqual(t, "Widget formal slug", widget.Formal.Reference.Slug, "widget")
	assertEqual(t, "Widget provider resource", widget.Formal.Binding.Import.ProviderResource, "widget_resource")
	assertEqual(t, "Widget formal kind", widget.Formal.Binding.Spec.Kind, "Widget")
	assertEqual(t, "Widget activity diagram path", widget.Formal.Diagrams.ActivitySourcePath, "controllers/mysql/widget/diagrams/activity.puml")

	report := findResource(t, pkg.Resources, "Report")
	if report.Formal != nil {
		t.Fatalf("Report formal model = %#v, want nil", report.Formal)
	}

	serviceManager := findServiceManagerModel(t, pkg.ServiceManagers, "Widget")
	if serviceManager.Formal == nil {
		t.Fatal("Widget service manager formal model was not attached")
	}
	assertEqual(t, "Widget service manager formal slug", serviceManager.Formal.Reference.Slug, "widget")
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

	widget := findResource(t, pkg.Resources, "Widget")
	if widget.Runtime == nil || widget.Runtime.Semantics == nil {
		t.Fatal("Widget runtime semantics were not attached")
	}
	assertSliceEqual(t, "Widget provisioning states", widget.Runtime.Semantics.Lifecycle.ProvisioningStates, []string{"PROVISIONING"})
	assertSliceEqual(t, "Widget active states", widget.Runtime.Semantics.Lifecycle.ActiveStates, []string{"ACTIVE"})
	assertEqual(t, "Widget delete policy", widget.Runtime.Semantics.Delete.Policy, "required")
	if widget.Runtime.Semantics.List == nil {
		t.Fatal("Widget list semantics were not attached")
	}
	assertEqual(t, "Widget list responseItemsField", widget.Runtime.Semantics.List.ResponseItemsField, "Items")
	assertSliceEqual(t, "Widget list match fields", widget.Runtime.Semantics.List.MatchFields, []string{"compartmentId", "state"})
	assertSliceEqual(t, "Widget forceNew", widget.Runtime.Semantics.Mutation.ForceNew, []string{"compartmentId"})
	assertEqual(t, "Widget create follow-up", widget.Runtime.Semantics.CreateFollowUp.Strategy, followUpStrategyReadAfterWrite)
	assertEqual(t, "Widget open gaps", len(widget.Runtime.Semantics.OpenGaps), 0)

	serviceManager := findServiceManagerModel(t, pkg.ServiceManagers, "Widget")
	if serviceManager.Semantics == nil {
		t.Fatal("Widget service manager semantics were not attached")
	}
	assertEqual(t, "Widget service manager formal slug", serviceManager.Semantics.FormalSlug, "widget")
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

func assertFunctionsComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	application := findResource(t, pkg.Resources, "Application")
	assertFieldMatches(t, "Application TraceConfig", findFieldModel(t, application.SpecFields, "TraceConfig"), fieldExpectation{Type: "ApplicationTraceConfig"})
	assertFieldMatches(t, "Application ImagePolicyConfig", findFieldModel(t, application.SpecFields, "ImagePolicyConfig"), fieldExpectation{Type: "ApplicationImagePolicyConfig"})
	assertFieldMatches(t, "Application DefinedTags", findFieldModel(t, application.SpecFields, "DefinedTags"), fieldExpectation{Type: "map[string]shared.MapValue"})
	assertFieldsPresent(t, "ApplicationTraceConfig", findHelperType(t, application.HelperTypes, "ApplicationTraceConfig").Fields, []string{"DomainId"})
	assertFieldsPresent(t, "ApplicationImagePolicyConfig", findHelperType(t, application.HelperTypes, "ApplicationImagePolicyConfig").Fields, []string{"IsPolicyEnabled"})

	function := findResource(t, pkg.Resources, "Function")
	assertFieldMatches(t, "Function SourceDetails", findFieldModel(t, function.SpecFields, "SourceDetails"), fieldExpectation{Type: "FunctionSourceDetails"})
	assertFieldsPresent(t, "FunctionSourceDetails", findHelperType(t, function.HelperTypes, "FunctionSourceDetails").Fields, []string{"SourceType", "PbfListingId"})
	assertFieldMatches(t, "Function ProvisionedConcurrencyConfig", findFieldModel(t, function.SpecFields, "ProvisionedConcurrencyConfig"), fieldExpectation{Type: "FunctionProvisionedConcurrencyConfig"})
	assertFieldsPresent(t, "FunctionProvisionedConcurrencyConfig", findHelperType(t, function.HelperTypes, "FunctionProvisionedConcurrencyConfig").Fields, []string{"Strategy", "Count"})
}

func assertCoreComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	tunnel := findResource(t, pkg.Resources, "IPSecConnectionTunnel")
	assertFieldMatches(t, "IPSecConnectionTunnel BgpSessionConfig", findFieldModel(t, tunnel.SpecFields, "BgpSessionConfig"), fieldExpectation{Type: "IPSecConnectionTunnelBgpSessionConfig"})
	assertFieldMatches(t, "IPSecConnectionTunnel PhaseOneConfig", findFieldModel(t, tunnel.SpecFields, "PhaseOneConfig"), fieldExpectation{Type: "IPSecConnectionTunnelPhaseOneConfig"})
	assertFieldMatches(t, "IPSecConnectionTunnel PhaseTwoConfig", findFieldModel(t, tunnel.SpecFields, "PhaseTwoConfig"), fieldExpectation{Type: "IPSecConnectionTunnelPhaseTwoConfig"})
	assertFieldsPresent(t, "IPSecConnectionTunnelBgpSessionConfig", findHelperType(t, tunnel.HelperTypes, "IPSecConnectionTunnelBgpSessionConfig").Fields, []string{"CustomerBgpAsn"})
	assertFieldsPresent(t, "IPSecConnectionTunnelPhaseOneConfig", findHelperType(t, tunnel.HelperTypes, "IPSecConnectionTunnelPhaseOneConfig").Fields, []string{"DiffieHelmanGroup"})
}

func assertCertificatesComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	bundle := findResource(t, pkg.Resources, "CertificateBundle")
	assertFieldMatches(t, "CertificateBundle Validity", findFieldModel(t, bundle.StatusFields, "Validity"), fieldExpectation{Type: "CertificateBundleValidity"})
	assertFieldMatches(t, "CertificateBundle RevocationStatus", findFieldModel(t, bundle.StatusFields, "RevocationStatus"), fieldExpectation{Type: "CertificateBundleRevocationStatus"})
	assertFieldsPresent(t, "CertificateBundleValidity", findHelperType(t, bundle.HelperTypes, "CertificateBundleValidity").Fields, []string{"TimeOfValidityNotBefore"})
	assertFieldsPresent(t, "CertificateBundleRevocationStatus", findHelperType(t, bundle.HelperTypes, "CertificateBundleRevocationStatus").Fields, []string{"RevocationReason"})
}

func assertNoSQLComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	row := findResource(t, pkg.Resources, "Row")
	assertFieldMatches(t, "Row Value", findFieldModel(t, row.SpecFields, "Value"), fieldExpectation{Type: "map[string]shared.JSONValue"})
}

func assertSecretsComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	bundle := findResource(t, pkg.Resources, "SecretBundle")
	assertFieldMatches(t, "SecretBundle SecretBundleContent", findFieldModel(t, bundle.StatusFields, "SecretBundleContent"), fieldExpectation{Type: "SecretBundleContent"})
	assertFieldsPresent(t, "SecretBundleContent", findHelperType(t, bundle.HelperTypes, "SecretBundleContent").Fields, []string{"ContentType", "Content"})
	assertResourceStatusFields(t, pkg, "SecretBundleByName", []string{"SecretId", "VersionNumber", "SecretBundleContent", "Metadata"})
}

func assertVaultComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	secret := findResource(t, pkg.Resources, "Secret")
	assertFieldMatches(t, "Secret Metadata", findFieldModel(t, secret.SpecFields, "Metadata"), fieldExpectation{Type: "map[string]shared.JSONValue"})
}

func assertArtifactsComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	assertResourceStatusFields(t, pkg, "ContainerConfiguration", []string{"IsRepositoryCreatedOnFirstPush"})
	containerImage := findResource(t, pkg.Resources, "ContainerImage")
	assertFieldsPresent(t, "ContainerImage status", containerImage.StatusFields, []string{"FreeformTags"})
	assertFieldMatches(t, "ContainerImage DefinedTags", findFieldModel(t, containerImage.StatusFields, "DefinedTags"), fieldExpectation{Type: "map[string]shared.MapValue"})
	assertResourceStatusFields(t, pkg, "ContainerImageSignature", []string{"CompartmentId", "ImageId", "Message", "Signature", "SigningAlgorithm"})
	containerRepository := findResource(t, pkg.Resources, "ContainerRepository")
	assertFieldsPresent(t, "ContainerRepository status", containerRepository.StatusFields, []string{"CompartmentId", "DisplayName", "IsImmutable", "IsPublic", "FreeformTags", "DefinedTags"})
	assertFieldMatches(t, "ContainerRepository Readme", findFieldModel(t, containerRepository.StatusFields, "Readme"), fieldExpectation{Type: "ContainerRepositoryReadme"})
	assertResourceStatusFields(t, pkg, "GenericArtifact", []string{"FreeformTags"})
	assertResourceStatusFields(t, pkg, "Repository", []string{"DisplayName", "Description", "CompartmentId", "IsImmutable", "FreeformTags", "DefinedTags"})
}

func assertNetworkLoadBalancerComplexSDKFields(t *testing.T, pkg *PackageModel) {
	t.Helper()

	healthChecker := findResource(t, pkg.Resources, "HealthChecker")
	assertFieldMatches(t, "HealthChecker RequestData", findFieldModel(t, healthChecker.SpecFields, "RequestData"), fieldExpectation{Type: "string"})
	assertFieldMatches(t, "HealthChecker ResponseData", findFieldModel(t, healthChecker.SpecFields, "ResponseData"), fieldExpectation{Type: "string"})
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

	assertResourceStatusFields(t, pkg, "DbSystem", []string{"DisplayName", "CompartmentId", "Shape", "DbVersion"})
	assertResourceStatusFields(t, pkg, "Configuration", []string{"DisplayName", "Shape", "DbVersion", "InstanceOcpuCount"})
	assertResourceStatusFields(t, pkg, "Backup", []string{"DisplayName", "CompartmentId", "DbSystemId", "RetentionPeriod"})
	assertResourceStatusFields(t, pkg, "PrimaryDbInstance", []string{"DbInstanceId"})
	assertResourceStatusFields(t, pkg, "WorkRequestLog", []string{"Message", "Timestamp"})
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

	assertResourceStatusFields(t, pkg, "Cluster", []string{"Name", "CompartmentId", "EndpointConfig", "VcnId", "KubernetesVersion", "KmsKeyId", "FreeformTags", "DefinedTags", "Options", "ImagePolicyConfig", "ClusterPodNetworkOptions", "Type"})
	assertResourceStatusFields(t, pkg, "NodePool", []string{"CompartmentId", "ClusterId", "Name", "KubernetesVersion", "NodeMetadata", "NodeImageName", "NodeSourceDetails", "NodeShapeConfig", "InitialNodeLabels", "SshPublicKey", "QuantityPerSubnet", "SubnetIds", "NodeConfigDetails", "FreeformTags", "DefinedTags", "NodeEvictionNodePoolSettings", "NodePoolCyclingDetails"})
	assertResourceStatusFields(t, pkg, "VirtualNodePool", []string{"CompartmentId", "ClusterId", "DisplayName", "PlacementConfigurations", "InitialVirtualNodeLabels", "Taints", "Size", "NsgIds", "PodConfiguration", "FreeformTags", "DefinedTags", "VirtualNodeTags"})
	assertResourceStatusFields(t, pkg, "Addon", []string{"Version", "Configurations"})
	assertResourceStatusFields(t, pkg, "WorkloadMapping", []string{"Namespace", "MappedCompartmentId", "FreeformTags", "DefinedTags"})
	assertResourceStatusFields(t, pkg, "WorkRequestLog", []string{"Message", "Timestamp"})

	workRequestStatus := findFieldModel(t, findResource(t, pkg.Resources, "WorkRequest").StatusFields, "Status")
	assertFieldMatches(t, "WorkRequest Status", workRequestStatus, fieldExpectation{Tag: `json:"sdkStatus,omitempty"`})

	credentialRotationObservedStatus := findFieldModel(t, findResource(t, pkg.Resources, "CredentialRotationStatus").StatusFields, "Status")
	assertFieldMatches(t, "CredentialRotationStatus Status", credentialRotationObservedStatus, fieldExpectation{Tag: `json:"sdkStatus,omitempty"`})
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

	assertResourceStatusFields(t, pkg, "BulkActionResourceType", []string{"Items"})
	assertResourceStatusFields(t, pkg, "BulkEditTagsResourceType", []string{"Items"})
	assertResourceStatusFields(t, pkg, "CostTrackingTag", []string{"TagNamespaceId", "TagNamespaceName", "IsRetired", "Validator"})
	assertResourceStatusFields(t, pkg, "IdentityProvider", []string{"CompartmentId", "Name", "Description", "Metadata", "MetadataUrl", "ProductType"})
	assertResourceStatusFields(t, pkg, "NetworkSource", []string{"CompartmentId", "Name", "Description", "PublicSourceList", "Services", "VirtualSourceList"})
	assertResourceStatusFields(t, pkg, "OrResetUIPassword", []string{"Password", "UserId", "TimeCreated", "LifecycleState", "InactiveStatus"})
	assertResourceStatusFields(t, pkg, "StandardTagNamespace", []string{"Description", "StandardTagNamespaceName", "TagDefinitionTemplates"})
	assertFieldMatches(t, "StandardTagNamespace Status", findFieldModel(t, findResource(t, pkg.Resources, "StandardTagNamespace").StatusFields, "Status"), fieldExpectation{
		Tag: `json:"sdkStatus,omitempty"`,
	})
	assertResourceStatusFields(t, pkg, "StandardTagTemplate", []string{"Description", "TagDefinitionName", "Type", "IsCostTracking"})
	assertResourceStatusFields(t, pkg, "UserState", []string{"Id", "CompartmentId", "Name", "LifecycleState", "Capabilities"})
	assertResourceStatusFields(t, pkg, "UserUIPasswordInformation", []string{"UserId", "TimeCreated", "LifecycleState"})
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

	assertResourceStatusFields(t, pkg, "ClusterNetworkInstance", []string{"AvailabilityDomain", "CompartmentId", "Region", "State", "TimeCreated"})
	assertResourceStatusFields(t, pkg, "ComputeCapacityReservationInstance", []string{"AvailabilityDomain", "CompartmentId", "Id", "Shape"})
	assertResourceStatusFields(t, pkg, "ComputeGlobalImageCapabilitySchema", []string{"ComputeGlobalImageCapabilitySchemaId", "Name"})
	assertResourceStatusFields(t, pkg, "NetworkSecurityGroupSecurityRule", []string{"Direction", "Protocol", "Id", "TcpOptions", "UdpOptions"})
	assertResourceStatusFields(t, pkg, "IPSecConnectionTunnelError", []string{"ErrorCode", "ErrorDescription", "Id", "Solution", "Timestamp"})
	assertResourceStatusFields(t, pkg, "IPSecConnectionTunnelRoute", []string{"Advertiser", "AsPath", "IsBestPath", "Prefix"})
	assertResourceStatusFields(t, pkg, "IPSecConnectionTunnelSecurityAssociation", []string{"CpeSubnet", "OracleSubnet", "TunnelSaStatus"})
	assertResourceStatusFields(t, pkg, "InstanceDevice", []string{"IsAvailable", "Name"})
	assertResourceStatusFields(t, pkg, "VolumeBackupPolicyAssetAssignment", []string{"AssetId", "Id", "PolicyId", "TimeCreated"})
	assertResourceStatusFields(t, pkg, "WindowsInstanceInitialCredential", []string{"Password", "Username"})
	assertResourceStatusFields(t, pkg, "FastConnectProviderVirtualCircuitBandwidthShape", []string{"BandwidthInMbps", "Name"})
	assertResourceStatusFields(t, pkg, "CrossconnectPortSpeedShape", []string{"Name", "PortSpeedInGbps"})
	assertResourceStatusFields(t, pkg, "AllDrgAttachment", []string{"Id"})
	assertResourceStatusFields(t, pkg, "AllowedPeerRegionsForRemotePeering", []string{"Name"})
	assertResourceStatusFields(t, pkg, "AppCatalogListingAgreement", []string{"ListingId", "ListingResourceVersion", "OracleTermsOfUseLink", "EulaLink", "TimeRetrieved", "Signature"})
	assertResourceStatusFields(t, pkg, "CrossConnectLetterOfAuthority", []string{"CrossConnectId", "FacilityLocation", "TimeExpires"})
	assertResourceStatusFields(t, pkg, "CrossConnectMapping", []string{"Ipv4BgpStatus", "Ipv6BgpStatus", "OciLogicalDeviceName"})
	assertResourceStatusFields(t, pkg, "DhcpOption", []string{"CompartmentId", "DisplayName", "LifecycleState", "Options", "TimeCreated", "VcnId"})
	assertResourceStatusFields(t, pkg, "VirtualCircuitAssociatedTunnel", []string{"TunnelId", "TunnelType"})
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
	result, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	assertEqual(t, "Generate() generated services", len(result.Generated), 1)

	groupVersionPath := filepath.Join(outputRoot, "api", "mysql", "v1beta1", "groupversion_info.go")
	groupVersionContent := readFile(t, groupVersionPath)
	assertStringContains(t, "groupversion_info.go generator banner", groupVersionContent, "// Code generated by generator. DO NOT EDIT.")
	assertStringContains(t, "groupversion_info.go GroupVersion", groupVersionContent, `GroupVersion = schema.GroupVersion{Group: "mysql.oracle.com", Version: "v1beta1"}`)

	resourcePath := filepath.Join(outputRoot, "api", "mysql", "v1beta1", "mysqldbsystem_types.go")
	resourceContent := readFile(t, resourcePath)
	assertStringContains(t, "mysqldbsystem_types.go kind", resourceContent, "type MySqlDbSystemSpec struct")
	assertStringContains(t, "mysqldbsystem_types.go Port field name", resourceContent, "Port")
	assertStringContains(t, "mysqldbsystem_types.go Port field tag", resourceContent, `json:"port,omitempty"`)

	result, err = pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot:   outputRoot,
		SkipExisting: true,
	})
	if err != nil {
		t.Fatalf("Generate() second run error = %v", err)
	}
	assertEqual(t, "Generate() skipped services", len(result.Skipped), 1)
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
	assertFieldMatches(t, "Application CompartmentId", compartmentID, fieldExpectation{
		Tag:     `json:"compartmentId"`,
		Markers: []string{"+kubebuilder:validation:Required"},
	})
	assertStringContains(t, "Application CompartmentId comments", strings.Join(compartmentID.Comments, "\n"), "compartment to create the application within")

	config := findFieldModel(t, application.SpecFields, "Config")
	assertFieldMatches(t, "Application Config", config, fieldExpectation{
		Tag:     `json:"config,omitempty"`,
		Markers: []string{"+kubebuilder:validation:Optional"},
	})
	assertStringContains(t, "Application Config comments", strings.Join(config.Comments, "\n"), "Application configuration")

	lifecycleState := findFieldModel(t, application.StatusFields, "LifecycleState")
	assertFieldMatches(t, "Application LifecycleState", lifecycleState, fieldExpectation{Markers: []string{}})
	assertStringContains(t, "Application LifecycleState comments", strings.Join(lifecycleState.Comments, "\n"), "current state of the application")

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
				"- ../../../config/rbac/mysqldbsystem_editor_role.yaml",
				"- ../../../config/rbac/mysqldbsystem_viewer_role.yaml",
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
			Kind: "MySqlDbSystem",
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

func TestGenerateControllerBackedPackagesWithoutParityDoNotReferenceEditorViewerRoles(t *testing.T) {
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
			Kind: "MySqlDbSystem",
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
			Kind: "MySqlDbSystem",
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
	cfg := loadCheckedInConfig(t)
	services := selectServices(t, cfg, []string{"database", "mysql", "streaming"})
	assertEqual(t, "selected parity services", len(services), 3)

	outputRoot, result := generateCheckedInArtifacts(t, cfg, services)
	assertEqual(t, "Generate() generated services", len(result.Generated), 3)
	assertGeneratedServiceCounts(t, result.Generated, map[string]int{
		"database":  79,
		"mysql":     12,
		"streaming": 7,
	})

	apiFiles := []string{
		"api/database/v1beta1/groupversion_info.go",
		"api/database/v1beta1/autonomousdatabases_types.go",
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
		"config/samples/database_v1beta1_autonomousdatabases.yaml",
		"config/samples/mysql_v1beta1_mysqldbsystem.yaml",
		"config/samples/streaming_v1beta1_stream.yaml",
		"packages/database/metadata.env",
		"packages/database/install/kustomization.yaml",
		"packages/mysql/metadata.env",
		"packages/mysql/install/kustomization.yaml",
		"packages/streaming/metadata.env",
		"packages/streaming/install/kustomization.yaml",
	}
	assertExactFileMatchesSet(t, repoRoot(t), outputRoot, exactFiles)

	runtimeFiles := []string{
		"controllers/database/autonomousdatabases_controller.go",
		"controllers/database/autonomousdatabasebackup_controller.go",
		"controllers/mysql/mysqldbsystem_controller.go",
		"controllers/mysql/backup_controller.go",
		"controllers/streaming/stream_controller.go",
		"controllers/streaming/streampool_controller.go",
		"pkg/servicemanager/autonomousdatabases/adb/autonomousdatabases_serviceclient.go",
		"pkg/servicemanager/autonomousdatabases/adb/autonomousdatabases_servicemanager.go",
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

func TestCheckedInStreamingParityPreservesStreamNamePrinterColumn(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	service := requireService(t, cfg, "streaming")

	pkg, err := NewDiscoverer().BuildPackageModel(context.Background(), cfg, *service)
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

func TestCheckedInPromotedCoreRuntimeArtifactsMatchGenerator(t *testing.T) {
	cfg := loadCheckedInConfig(t)
	coreService := requireService(t, cfg, "core")
	outputRoot, result := generateCheckedInArtifacts(t, cfg, []ServiceConfig{*coreService})
	assertEqual(t, "Generate() generated services", len(result.Generated), 1)

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
	assertExactFileMatchesSet(t, repoRoot(t), outputRoot, exactFiles)
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
	assertGoParity(t, filepath.Join(repoRoot(t), serviceClientPath), filepath.Join(outputRoot, serviceClientPath))
	assertGoParity(t, filepath.Join(repoRoot(t), serviceManagerPath), filepath.Join(outputRoot, serviceManagerPath))

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

func TestMySQLParityIncludesOptionalDesiredStateFields(t *testing.T) {
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
		"type DeletionPolicyDetails struct {",
		"AutomaticBackupRetention string `json:\"automaticBackupRetention,omitempty\"`",
		"FinalBackup string `json:\"finalBackup,omitempty\"`",
		"IsDeleteProtected bool `json:\"isDeleteProtected,omitempty\"`",
		"type SecureConnectionDetails struct {",
		"CertificateGenerationType string `json:\"certificateGenerationType,omitempty\"`",
		"CertificateId shared.OCID `json:\"certificateId,omitempty\"`",
		"DeletionPolicy DeletionPolicyDetails `json:\"deletionPolicy,omitempty\"`",
		"CrashRecovery string `json:\"crashRecovery,omitempty\"`",
		"DatabaseManagement string `json:\"databaseManagement,omitempty\"`",
		"SecureConnections SecureConnectionDetails `json:\"secureConnections,omitempty\"`",
	})
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
	parity := &ParityConfig{
		Resources: []ParityResource{
			{
				SourceResource: "DbSystem",
				Kind:           "MySqlDbSystem",
				FileStem:       "mysqldbsystem",
				SpecFields: []FieldOverride{
					{Name: "MySqlDbSystemId", Type: "shared.OCID", Tag: `json:"id,omitempty"`},
					{Name: "CompartmentId", Type: "shared.OCID", Tag: `json:"compartmentId,omitempty"`},
					{Name: "DisplayName", Type: "string", Tag: `json:"displayName,omitempty"`},
					{Name: "Port", Type: "int", Tag: `json:"port,omitempty"`},
				},
				Sample: SampleOverride{
					MetadataName: "mysqldbsystem-sample",
					Spec: `  compartmentId: ocid1.compartment.oc1..aaaaaaaaXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
  displayName: SampleDB
  port: 3306`,
				},
			},
		},
	}
	if profile == PackageProfileControllerBacked {
		parity.Package.ExtraResources = []string{
			"../../../config/rbac/mysqldbsystem_editor_role.yaml",
			"../../../config/rbac/mysqldbsystem_viewer_role.yaml",
		}
	}

	return ServiceConfig{
		Service:        "mysql",
		SDKPackage:     "github.com/oracle/oci-service-operator/internal/generator/testdata/sdk/sample",
		Group:          "mysql",
		PackageProfile: profile,
		Parity:         parity,
		Compatibility: CompatibilityConfig{
			ExistingKinds: []string{"MySqlDbSystem"},
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

type fieldExpectation struct {
	Type     string
	Tag      string
	Comments []string
	Markers  []string
}

func assertExactFileMatch(t *testing.T, wantPath string, gotPath string) {
	t.Helper()

	want := readFile(t, wantPath)
	got := readFile(t, gotPath)
	if want != got {
		t.Fatalf("file mismatch for %s", wantPath)
	}
}

func assertEqual[T comparable](t *testing.T, label string, got T, want T) {
	t.Helper()

	if got != want {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
}

func assertSliceEqual[T comparable](t *testing.T, label string, got []T, want []T) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
}

func assertStringContains(t *testing.T, label string, content string, want string) {
	t.Helper()

	if !strings.Contains(content, want) {
		t.Fatalf("%s did not contain %q:\n%s", label, want, content)
	}
}

func assertFieldsPresent(t *testing.T, label string, fields []FieldModel, want []string) {
	t.Helper()

	for _, fieldName := range want {
		if !hasField(fields, fieldName) {
			t.Fatalf("%s fields = %#v, want %s", label, fields, fieldName)
		}
	}
}

func assertFieldsAbsent(t *testing.T, label string, fields []FieldModel, want []string) {
	t.Helper()

	for _, fieldName := range want {
		if hasField(fields, fieldName) {
			t.Fatalf("%s fields = %#v, want no %s", label, fields, fieldName)
		}
	}
}

func assertNoFields(t *testing.T, label string, fields []FieldModel) {
	t.Helper()

	if len(fields) != 0 {
		t.Fatalf("%s fields = %#v, want none", label, fields)
	}
}

func assertFieldMatches(t *testing.T, label string, field FieldModel, want fieldExpectation) {
	t.Helper()

	if want.Type != "" {
		assertEqual(t, label+" type", field.Type, want.Type)
	}
	if want.Tag != "" {
		assertEqual(t, label+" tag", field.Tag, want.Tag)
	}
	if want.Comments != nil {
		assertSliceEqual(t, label+" comments", field.Comments, want.Comments)
	}
	if want.Markers != nil {
		assertSliceEqual(t, label+" markers", field.Markers, want.Markers)
	}
}

func assertResourceStatusFields(t *testing.T, pkg *PackageModel, kind string, want []string) {
	t.Helper()

	assertFieldsPresent(t, kind+" status", findResource(t, pkg.Resources, kind).StatusFields, want)
}

func selectServices(t *testing.T, cfg *Config, names []string) []ServiceConfig {
	t.Helper()

	selected := make([]ServiceConfig, 0, len(names))
	for _, service := range cfg.Services {
		if slices.Contains(names, service.Service) {
			selected = append(selected, service)
		}
	}
	return selected
}

func seedCheckedInSamplesKustomization(t *testing.T, outputRoot string) {
	t.Helper()

	samplesDir := filepath.Join(outputRoot, "config", "samples")
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", samplesDir, err)
	}
	checkedInSampleKustomization := readFile(t, filepath.Join(repoRoot(t), "config", "samples", "kustomization.yaml"))
	if err := os.WriteFile(filepath.Join(samplesDir, "kustomization.yaml"), []byte(checkedInSampleKustomization), 0o644); err != nil {
		t.Fatalf("write seeded samples kustomization: %v", err)
	}
}

func generateCheckedInArtifacts(t *testing.T, cfg *Config, services []ServiceConfig) (string, RunResult) {
	t.Helper()

	outputRoot := t.TempDir()
	seedCheckedInSamplesKustomization(t, outputRoot)
	result, err := New().Generate(context.Background(), cfg, services, Options{
		OutputRoot:                      outputRoot,
		PreserveExistingSpecSurfaceRoot: repoRoot(t),
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	return outputRoot, result
}

func assertGeneratedServiceCounts(t *testing.T, generated []ServiceResult, want map[string]int) {
	t.Helper()

	for _, service := range generated {
		wantCount, ok := want[service.Service]
		if !ok {
			t.Fatalf("generated unexpected service %q", service.Service)
		}
		assertEqual(t, fmt.Sprintf("service %s generated resources", service.Service), service.ResourceCount, wantCount)
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

func assertGoParity(t *testing.T, wantPath string, gotPath string) {
	t.Helper()

	want := normalizeGoForParity(t, readFile(t, wantPath))
	got := normalizeGoForParity(t, readFile(t, gotPath))
	if want != got {
		t.Fatalf("Go parity mismatch for %s\nwant:\n%s\n\ngot:\n%s", wantPath, want, got)
	}
}

func assertGoParityFiles(t *testing.T, wantRoot string, gotRoot string, relativePaths []string) {
	t.Helper()

	for _, relativePath := range relativePaths {
		assertGoParity(t, filepath.Join(wantRoot, relativePath), filepath.Join(gotRoot, relativePath))
	}
}

func assertExactFileMatchesSet(t *testing.T, wantRoot string, gotRoot string, relativePaths []string) {
	t.Helper()

	for _, relativePath := range relativePaths {
		assertExactFileMatch(t, filepath.Join(wantRoot, relativePath), filepath.Join(gotRoot, relativePath))
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

func normalizeGoForParity(t *testing.T, source string) string {
	t.Helper()

	source = strings.ReplaceAll(source, "// Code generated by generator. DO NOT EDIT.\n\n", "")
	source = strings.ReplaceAll(source, "// Code generated by osok-api-generator. DO NOT EDIT.\n\n", "")

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "parity.go", source, parser.SkipObjectResolution)
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
	if !ok || structType.Fields == nil {
		return
	}

	for _, field := range structType.Fields.List {
		field.Doc = nil
		field.Comment = nil
	}
}
