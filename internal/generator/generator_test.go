/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

//nolint:gocognit,gocyclo // These generator contract tests use large end-to-end fixtures to lock emitted surfaces.
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

	if pkg.GroupDNSName != "mysql.oracle.com" {
		t.Fatalf("BuildPackageModel() group DNS name = %q, want %q", pkg.GroupDNSName, "mysql.oracle.com")
	}

	dbSystem := findResource(t, pkg.Resources, "MySqlDbSystem")
	if dbSystem.SDKName != "DbSystem" {
		t.Fatalf("MySqlDbSystem SDK name = %q, want %q", dbSystem.SDKName, "DbSystem")
	}
	if !dbSystem.CompatibilityLocked {
		t.Fatal("MySqlDbSystem compatibility override was not applied")
	}
	if !hasField(dbSystem.SpecFields, "Port") {
		t.Fatalf("MySqlDbSystem fields = %#v, want Port", dbSystem.SpecFields)
	}
	if hasField(dbSystem.SpecFields, "Id") {
		t.Fatalf("MySqlDbSystem fields = %#v, want no implicit Id field", dbSystem.SpecFields)
	}
	if dbSystem.PrimaryDisplayField != "DisplayName" {
		t.Fatalf("MySqlDbSystem primary display field = %q, want DisplayName", dbSystem.PrimaryDisplayField)
	}

	widget := findResource(t, pkg.Resources, "Widget")
	if len(widget.Operations) != 5 {
		t.Fatalf("Widget operations = %v, want 5 CRUD operations", widget.Operations)
	}
	if !hasField(widget.SpecFields, "Mode") {
		t.Fatalf("Widget fields = %#v, want Mode alias field", widget.SpecFields)
	}
	if !hasField(widget.SpecFields, "CreatedAt") {
		t.Fatalf("Widget fields = %#v, want CreatedAt selector field", widget.SpecFields)
	}
	if hasField(widget.SpecFields, "LifecycleState") {
		t.Fatalf("Widget spec fields = %#v, want read-model fields moved out of spec", widget.SpecFields)
	}
	if hasField(widget.SpecFields, "TimeUpdated") {
		t.Fatalf("Widget spec fields = %#v, want summary fields moved out of spec", widget.SpecFields)
	}
	if !hasField(widget.StatusFields, "LifecycleState") {
		t.Fatalf("Widget status fields = %#v, want LifecycleState from the read model", widget.StatusFields)
	}
	if !hasField(widget.StatusFields, "TimeUpdated") {
		t.Fatalf("Widget status fields = %#v, want TimeUpdated from the summary model", widget.StatusFields)
	}

	compartmentID := findFieldModel(t, widget.SpecFields, "CompartmentId")
	if compartmentID.Tag != `json:"compartmentId"` {
		t.Fatalf("Widget CompartmentId tag = %q, want required json tag", compartmentID.Tag)
	}
	if !slices.Equal(compartmentID.Comments, []string{"The OCID of the widget compartment."}) {
		t.Fatalf("Widget CompartmentId comments = %#v, want SDK documentation", compartmentID.Comments)
	}
	if !slices.Equal(compartmentID.Markers, []string{"+kubebuilder:validation:Required"}) {
		t.Fatalf("Widget CompartmentId markers = %#v, want required marker", compartmentID.Markers)
	}

	labels := findFieldModel(t, widget.SpecFields, "Labels")
	if labels.Tag != `json:"labels,omitempty"` {
		t.Fatalf("Widget Labels tag = %q, want optional json tag", labels.Tag)
	}
	if !slices.Equal(labels.Comments, []string{"Additional labels for the widget."}) {
		t.Fatalf("Widget Labels comments = %#v, want SDK documentation", labels.Comments)
	}
	if !slices.Equal(labels.Markers, []string{"+kubebuilder:validation:Optional"}) {
		t.Fatalf("Widget Labels markers = %#v, want optional marker", labels.Markers)
	}

	serverState := findFieldModel(t, widget.SpecFields, "ServerState")
	if serverState.Tag != `json:"serverState,omitempty"` {
		t.Fatalf("Widget ServerState tag = %q, want read-only field to keep omitempty", serverState.Tag)
	}
	if len(serverState.Markers) != 0 {
		t.Fatalf("Widget ServerState markers = %#v, want read-only field to suppress requiredness markers", serverState.Markers)
	}

	lifecycleState := findFieldModel(t, widget.StatusFields, "LifecycleState")
	if !slices.Equal(lifecycleState.Comments, []string{"The lifecycle state of the widget."}) {
		t.Fatalf("Widget LifecycleState comments = %#v, want SDK documentation on status fields", lifecycleState.Comments)
	}
	if len(lifecycleState.Markers) != 0 {
		t.Fatalf("Widget LifecycleState markers = %#v, want no requiredness markers on status fields", lifecycleState.Markers)
	}

	report := findResource(t, pkg.Resources, "Report")
	if len(report.SpecFields) != 0 {
		t.Fatalf("Report spec fields = %#v, want empty spec when no create or update payload exists", report.SpecFields)
	}
	if !hasField(report.StatusFields, "Id") {
		t.Fatalf("Report status fields = %#v, want Id from the read model", report.StatusFields)
	}
	if !hasField(report.StatusFields, "LifecycleState") {
		t.Fatalf("Report status fields = %#v, want LifecycleState from the read model", report.StatusFields)
	}
	if !hasField(report.StatusFields, "DisplayName") {
		t.Fatalf("Report status fields = %#v, want DisplayName from the summary model", report.StatusFields)
	}

	reportByName := findResource(t, pkg.Resources, "ReportByName")
	if !hasField(reportByName.SpecFields, "DisplayName") {
		t.Fatalf("ReportByName spec fields = %#v, want DisplayName from the non-CRUD request payload", reportByName.SpecFields)
	}

	oauthClientCredential := findResource(t, pkg.Resources, "OAuthClientCredential")
	if !hasField(oauthClientCredential.SpecFields, "Name") {
		t.Fatalf("OAuthClientCredential spec fields = %#v, want Name from the aliased create payload", oauthClientCredential.SpecFields)
	}
	if !hasField(oauthClientCredential.SpecFields, "Description") {
		t.Fatalf("OAuthClientCredential spec fields = %#v, want Description from the aliased create/update payloads", oauthClientCredential.SpecFields)
	}
	if !hasField(oauthClientCredential.SpecFields, "Scopes") {
		t.Fatalf("OAuthClientCredential spec fields = %#v, want Scopes from the aliased create/update payloads", oauthClientCredential.SpecFields)
	}
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

	report := findResource(t, pkg.Resources, "Report")
	if report.Formal != nil {
		t.Fatalf("Report formal model = %#v, want nil", report.Formal)
	}

	serviceManager := findServiceManagerModel(t, pkg.ServiceManagers, "Widget")
	if serviceManager.Formal == nil {
		t.Fatal("Widget service manager formal model was not attached")
	}
	if serviceManager.Formal.Reference.Slug != "widget" {
		t.Fatalf("Widget service manager formal slug = %q, want %q", serviceManager.Formal.Reference.Slug, "widget")
	}
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
	if got := widget.Runtime.Semantics.Lifecycle.ProvisioningStates; !slices.Equal(got, []string{"PROVISIONING"}) {
		t.Fatalf("Widget provisioning states = %v, want [PROVISIONING]", got)
	}
	if got := widget.Runtime.Semantics.Lifecycle.ActiveStates; !slices.Equal(got, []string{"ACTIVE"}) {
		t.Fatalf("Widget active states = %v, want [ACTIVE]", got)
	}
	if got := widget.Runtime.Semantics.Delete.Policy; got != "required" {
		t.Fatalf("Widget delete policy = %q, want required", got)
	}
	if got := widget.Runtime.Semantics.List; got == nil || got.ResponseItemsField != "Items" {
		t.Fatalf("Widget list semantics = %#v, want responseItemsField Items", got)
	}
	if got := widget.Runtime.Semantics.List.MatchFields; !slices.Equal(got, []string{"compartmentId", "state"}) {
		t.Fatalf("Widget list match fields = %v, want [compartmentId state]", got)
	}
	if got := widget.Runtime.Semantics.Mutation.ForceNew; !slices.Equal(got, []string{"compartmentId"}) {
		t.Fatalf("Widget forceNew = %v, want [compartmentId]", got)
	}
	if got := widget.Runtime.Semantics.CreateFollowUp.Strategy; got != followUpStrategyReadAfterWrite {
		t.Fatalf("Widget create follow-up = %q, want %q", got, followUpStrategyReadAfterWrite)
	}
	if len(widget.Runtime.Semantics.OpenGaps) != 0 {
		t.Fatalf("Widget open gaps = %#v, want none", widget.Runtime.Semantics.OpenGaps)
	}

	serviceManager := findServiceManagerModel(t, pkg.ServiceManagers, "Widget")
	if serviceManager.Semantics == nil {
		t.Fatal("Widget service manager semantics were not attached")
	}
	if serviceManager.Semantics.FormalSlug != "widget" {
		t.Fatalf("Widget service manager formal slug = %q, want widget", serviceManager.Semantics.FormalSlug)
	}
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
			assert: func(t *testing.T, pkg *PackageModel) {
				application := findResource(t, pkg.Resources, "Application")
				traceConfig := findFieldModel(t, application.SpecFields, "TraceConfig")
				if traceConfig.Type != "ApplicationTraceConfig" {
					t.Fatalf("Application TraceConfig type = %q, want %q", traceConfig.Type, "ApplicationTraceConfig")
				}
				imagePolicy := findFieldModel(t, application.SpecFields, "ImagePolicyConfig")
				if imagePolicy.Type != "ApplicationImagePolicyConfig" {
					t.Fatalf("Application ImagePolicyConfig type = %q, want %q", imagePolicy.Type, "ApplicationImagePolicyConfig")
				}
				definedTags := findFieldModel(t, application.SpecFields, "DefinedTags")
				if definedTags.Type != "map[string]shared.MapValue" {
					t.Fatalf("Application DefinedTags type = %q, want %q", definedTags.Type, "map[string]shared.MapValue")
				}

				traceHelper := findHelperType(t, application.HelperTypes, "ApplicationTraceConfig")
				if !hasField(traceHelper.Fields, "DomainId") {
					t.Fatalf("ApplicationTraceConfig fields = %#v, want DomainId", traceHelper.Fields)
				}
				imagePolicyHelper := findHelperType(t, application.HelperTypes, "ApplicationImagePolicyConfig")
				if !hasField(imagePolicyHelper.Fields, "IsPolicyEnabled") {
					t.Fatalf("ApplicationImagePolicyConfig fields = %#v, want IsPolicyEnabled", imagePolicyHelper.Fields)
				}

				function := findResource(t, pkg.Resources, "Function")
				sourceDetails := findFieldModel(t, function.SpecFields, "SourceDetails")
				if sourceDetails.Type != "FunctionSourceDetails" {
					t.Fatalf("Function SourceDetails type = %q, want %q", sourceDetails.Type, "FunctionSourceDetails")
				}
				sourceHelper := findHelperType(t, function.HelperTypes, "FunctionSourceDetails")
				if !hasField(sourceHelper.Fields, "SourceType") {
					t.Fatalf("FunctionSourceDetails fields = %#v, want SourceType", sourceHelper.Fields)
				}
				if !hasField(sourceHelper.Fields, "PbfListingId") {
					t.Fatalf("FunctionSourceDetails fields = %#v, want PbfListingId", sourceHelper.Fields)
				}

				provisionedConcurrency := findFieldModel(t, function.SpecFields, "ProvisionedConcurrencyConfig")
				if provisionedConcurrency.Type != "FunctionProvisionedConcurrencyConfig" {
					t.Fatalf("Function ProvisionedConcurrencyConfig type = %q, want %q", provisionedConcurrency.Type, "FunctionProvisionedConcurrencyConfig")
				}
				provisionedHelper := findHelperType(t, function.HelperTypes, "FunctionProvisionedConcurrencyConfig")
				if !hasField(provisionedHelper.Fields, "Strategy") {
					t.Fatalf("FunctionProvisionedConcurrencyConfig fields = %#v, want Strategy", provisionedHelper.Fields)
				}
				if !hasField(provisionedHelper.Fields, "Count") {
					t.Fatalf("FunctionProvisionedConcurrencyConfig fields = %#v, want Count", provisionedHelper.Fields)
				}
			},
		},
		{
			name: "core",
			service: ServiceConfig{
				Service:        "core",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/core",
				Group:          "core",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: func(t *testing.T, pkg *PackageModel) {
				tunnel := findResource(t, pkg.Resources, "IPSecConnectionTunnel")
				bgpSession := findFieldModel(t, tunnel.SpecFields, "BgpSessionConfig")
				if bgpSession.Type != "IPSecConnectionTunnelBgpSessionConfig" {
					t.Fatalf("IPSecConnectionTunnel BgpSessionConfig type = %q, want %q", bgpSession.Type, "IPSecConnectionTunnelBgpSessionConfig")
				}
				phaseOne := findFieldModel(t, tunnel.SpecFields, "PhaseOneConfig")
				if phaseOne.Type != "IPSecConnectionTunnelPhaseOneConfig" {
					t.Fatalf("IPSecConnectionTunnel PhaseOneConfig type = %q, want %q", phaseOne.Type, "IPSecConnectionTunnelPhaseOneConfig")
				}
				phaseTwo := findFieldModel(t, tunnel.SpecFields, "PhaseTwoConfig")
				if phaseTwo.Type != "IPSecConnectionTunnelPhaseTwoConfig" {
					t.Fatalf("IPSecConnectionTunnel PhaseTwoConfig type = %q, want %q", phaseTwo.Type, "IPSecConnectionTunnelPhaseTwoConfig")
				}

				bgpHelper := findHelperType(t, tunnel.HelperTypes, "IPSecConnectionTunnelBgpSessionConfig")
				if !hasField(bgpHelper.Fields, "CustomerBgpAsn") {
					t.Fatalf("IPSecConnectionTunnelBgpSessionConfig fields = %#v, want CustomerBgpAsn", bgpHelper.Fields)
				}
				phaseOneHelper := findHelperType(t, tunnel.HelperTypes, "IPSecConnectionTunnelPhaseOneConfig")
				if !hasField(phaseOneHelper.Fields, "DiffieHelmanGroup") {
					t.Fatalf("IPSecConnectionTunnelPhaseOneConfig fields = %#v, want DiffieHelmanGroup", phaseOneHelper.Fields)
				}
			},
		},
		{
			name: "certificates",
			service: ServiceConfig{
				Service:        "certificates",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/certificates",
				Group:          "certificates",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: func(t *testing.T, pkg *PackageModel) {
				bundle := findResource(t, pkg.Resources, "CertificateBundle")
				validity := findFieldModel(t, bundle.StatusFields, "Validity")
				if validity.Type != "CertificateBundleValidity" {
					t.Fatalf("CertificateBundle Validity type = %q, want %q", validity.Type, "CertificateBundleValidity")
				}
				revocationStatus := findFieldModel(t, bundle.StatusFields, "RevocationStatus")
				if revocationStatus.Type != "CertificateBundleRevocationStatus" {
					t.Fatalf("CertificateBundle RevocationStatus type = %q, want %q", revocationStatus.Type, "CertificateBundleRevocationStatus")
				}

				validityHelper := findHelperType(t, bundle.HelperTypes, "CertificateBundleValidity")
				if !hasField(validityHelper.Fields, "TimeOfValidityNotBefore") {
					t.Fatalf("CertificateBundleValidity fields = %#v, want TimeOfValidityNotBefore", validityHelper.Fields)
				}
				revocationHelper := findHelperType(t, bundle.HelperTypes, "CertificateBundleRevocationStatus")
				if !hasField(revocationHelper.Fields, "RevocationReason") {
					t.Fatalf("CertificateBundleRevocationStatus fields = %#v, want RevocationReason", revocationHelper.Fields)
				}
			},
		},
		{
			name: "nosql",
			service: ServiceConfig{
				Service:        "nosql",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/nosql",
				Group:          "nosql",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: func(t *testing.T, pkg *PackageModel) {
				row := findResource(t, pkg.Resources, "Row")
				value := findFieldModel(t, row.SpecFields, "Value")
				if value.Type != "map[string]shared.JSONValue" {
					t.Fatalf("Row Value type = %q, want %q", value.Type, "map[string]shared.JSONValue")
				}
			},
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
			assert: func(t *testing.T, pkg *PackageModel) {
				bundle := findResource(t, pkg.Resources, "SecretBundle")
				secretBundleContent := findFieldModel(t, bundle.StatusFields, "SecretBundleContent")
				if secretBundleContent.Type != "SecretBundleContent" {
					t.Fatalf("SecretBundle SecretBundleContent type = %q, want %q", secretBundleContent.Type, "SecretBundleContent")
				}

				contentHelper := findHelperType(t, bundle.HelperTypes, "SecretBundleContent")
				if !hasField(contentHelper.Fields, "ContentType") {
					t.Fatalf("SecretBundleContent fields = %#v, want ContentType", contentHelper.Fields)
				}
				if !hasField(contentHelper.Fields, "Content") {
					t.Fatalf("SecretBundleContent fields = %#v, want Content", contentHelper.Fields)
				}

				bundleByName := findResource(t, pkg.Resources, "SecretBundleByName")
				for _, fieldName := range []string{"SecretId", "VersionNumber", "SecretBundleContent", "Metadata"} {
					if !hasField(bundleByName.StatusFields, fieldName) {
						t.Fatalf("SecretBundleByName status fields = %#v, want %s", bundleByName.StatusFields, fieldName)
					}
				}
			},
		},
		{
			name: "vault",
			service: ServiceConfig{
				Service:        "vault",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/vault",
				Group:          "vault",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: func(t *testing.T, pkg *PackageModel) {
				secret := findResource(t, pkg.Resources, "Secret")
				metadata := findFieldModel(t, secret.SpecFields, "Metadata")
				if metadata.Type != "map[string]shared.JSONValue" {
					t.Fatalf("Secret Metadata type = %q, want %q", metadata.Type, "map[string]shared.JSONValue")
				}
			},
		},
		{
			name: "artifacts",
			service: ServiceConfig{
				Service:        "artifacts",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/artifacts",
				Group:          "artifacts",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: func(t *testing.T, pkg *PackageModel) {
				containerConfiguration := findResource(t, pkg.Resources, "ContainerConfiguration")
				if !hasField(containerConfiguration.StatusFields, "IsRepositoryCreatedOnFirstPush") {
					t.Fatalf("ContainerConfiguration status fields = %#v, want IsRepositoryCreatedOnFirstPush", containerConfiguration.StatusFields)
				}

				containerImage := findResource(t, pkg.Resources, "ContainerImage")
				if !hasField(containerImage.StatusFields, "FreeformTags") {
					t.Fatalf("ContainerImage status fields = %#v, want FreeformTags", containerImage.StatusFields)
				}
				definedTags := findFieldModel(t, containerImage.StatusFields, "DefinedTags")
				if definedTags.Type != "map[string]shared.MapValue" {
					t.Fatalf("ContainerImage DefinedTags type = %q, want %q", definedTags.Type, "map[string]shared.MapValue")
				}

				containerImageSignature := findResource(t, pkg.Resources, "ContainerImageSignature")
				for _, fieldName := range []string{"CompartmentId", "ImageId", "Message", "Signature", "SigningAlgorithm"} {
					if !hasField(containerImageSignature.StatusFields, fieldName) {
						t.Fatalf("ContainerImageSignature status fields = %#v, want %s", containerImageSignature.StatusFields, fieldName)
					}
				}

				containerRepository := findResource(t, pkg.Resources, "ContainerRepository")
				for _, fieldName := range []string{"CompartmentId", "DisplayName", "IsImmutable", "IsPublic", "FreeformTags", "DefinedTags"} {
					if !hasField(containerRepository.StatusFields, fieldName) {
						t.Fatalf("ContainerRepository status fields = %#v, want %s", containerRepository.StatusFields, fieldName)
					}
				}
				readme := findFieldModel(t, containerRepository.StatusFields, "Readme")
				if readme.Type != "ContainerRepositoryReadme" {
					t.Fatalf("ContainerRepository Readme type = %q, want %q", readme.Type, "ContainerRepositoryReadme")
				}

				genericArtifact := findResource(t, pkg.Resources, "GenericArtifact")
				if !hasField(genericArtifact.StatusFields, "FreeformTags") {
					t.Fatalf("GenericArtifact status fields = %#v, want FreeformTags", genericArtifact.StatusFields)
				}

				repository := findResource(t, pkg.Resources, "Repository")
				for _, fieldName := range []string{"DisplayName", "Description", "CompartmentId", "IsImmutable", "FreeformTags", "DefinedTags"} {
					if !hasField(repository.StatusFields, fieldName) {
						t.Fatalf("Repository status fields = %#v, want %s", repository.StatusFields, fieldName)
					}
				}
			},
		},
		{
			name: "networkloadbalancer",
			service: ServiceConfig{
				Service:        "networkloadbalancer",
				SDKPackage:     "github.com/oracle/oci-go-sdk/v65/networkloadbalancer",
				Group:          "networkloadbalancer",
				PackageProfile: PackageProfileCRDOnly,
			},
			assert: func(t *testing.T, pkg *PackageModel) {
				healthChecker := findResource(t, pkg.Resources, "HealthChecker")
				requestData := findFieldModel(t, healthChecker.SpecFields, "RequestData")
				if requestData.Type != "string" {
					t.Fatalf("HealthChecker RequestData type = %q, want string", requestData.Type)
				}

				responseData := findFieldModel(t, healthChecker.SpecFields, "ResponseData")
				if responseData.Type != "string" {
					t.Fatalf("HealthChecker ResponseData type = %q, want string", responseData.Type)
				}
			},
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

	dbSystem := findResource(t, pkg.Resources, "DbSystem")
	for _, fieldName := range []string{"DisplayName", "CompartmentId", "Shape", "DbVersion"} {
		if !hasField(dbSystem.StatusFields, fieldName) {
			t.Fatalf("DbSystem status fields = %#v, want %s", dbSystem.StatusFields, fieldName)
		}
	}

	configuration := findResource(t, pkg.Resources, "Configuration")
	for _, fieldName := range []string{"DisplayName", "Shape", "DbVersion", "InstanceOcpuCount"} {
		if !hasField(configuration.StatusFields, fieldName) {
			t.Fatalf("Configuration status fields = %#v, want %s", configuration.StatusFields, fieldName)
		}
	}

	backup := findResource(t, pkg.Resources, "Backup")
	for _, fieldName := range []string{"DisplayName", "CompartmentId", "DbSystemId", "RetentionPeriod"} {
		if !hasField(backup.StatusFields, fieldName) {
			t.Fatalf("Backup status fields = %#v, want %s", backup.StatusFields, fieldName)
		}
	}

	primaryDbInstance := findResource(t, pkg.Resources, "PrimaryDbInstance")
	if !hasField(primaryDbInstance.StatusFields, "DbInstanceId") {
		t.Fatalf("PrimaryDbInstance status fields = %#v, want DbInstanceId", primaryDbInstance.StatusFields)
	}

	workRequestLog := findResource(t, pkg.Resources, "WorkRequestLog")
	for _, fieldName := range []string{"Message", "Timestamp"} {
		if !hasField(workRequestLog.StatusFields, fieldName) {
			t.Fatalf("WorkRequestLog status fields = %#v, want %s", workRequestLog.StatusFields, fieldName)
		}
	}
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

	cluster := findResource(t, pkg.Resources, "Cluster")
	for _, fieldName := range []string{"Name", "CompartmentId", "EndpointConfig", "VcnId", "KubernetesVersion", "KmsKeyId", "FreeformTags", "DefinedTags", "Options", "ImagePolicyConfig", "ClusterPodNetworkOptions", "Type"} {
		if !hasField(cluster.StatusFields, fieldName) {
			t.Fatalf("Cluster status fields = %#v, want %s", cluster.StatusFields, fieldName)
		}
	}

	nodePool := findResource(t, pkg.Resources, "NodePool")
	for _, fieldName := range []string{"CompartmentId", "ClusterId", "Name", "KubernetesVersion", "NodeMetadata", "NodeImageName", "NodeSourceDetails", "NodeShapeConfig", "InitialNodeLabels", "SshPublicKey", "QuantityPerSubnet", "SubnetIds", "NodeConfigDetails", "FreeformTags", "DefinedTags", "NodeEvictionNodePoolSettings", "NodePoolCyclingDetails"} {
		if !hasField(nodePool.StatusFields, fieldName) {
			t.Fatalf("NodePool status fields = %#v, want %s", nodePool.StatusFields, fieldName)
		}
	}

	virtualNodePool := findResource(t, pkg.Resources, "VirtualNodePool")
	for _, fieldName := range []string{"CompartmentId", "ClusterId", "DisplayName", "PlacementConfigurations", "InitialVirtualNodeLabels", "Taints", "Size", "NsgIds", "PodConfiguration", "FreeformTags", "DefinedTags", "VirtualNodeTags"} {
		if !hasField(virtualNodePool.StatusFields, fieldName) {
			t.Fatalf("VirtualNodePool status fields = %#v, want %s", virtualNodePool.StatusFields, fieldName)
		}
	}

	addon := findResource(t, pkg.Resources, "Addon")
	for _, fieldName := range []string{"Version", "Configurations"} {
		if !hasField(addon.StatusFields, fieldName) {
			t.Fatalf("Addon status fields = %#v, want %s", addon.StatusFields, fieldName)
		}
	}

	workloadMapping := findResource(t, pkg.Resources, "WorkloadMapping")
	for _, fieldName := range []string{"Namespace", "MappedCompartmentId", "FreeformTags", "DefinedTags"} {
		if !hasField(workloadMapping.StatusFields, fieldName) {
			t.Fatalf("WorkloadMapping status fields = %#v, want %s", workloadMapping.StatusFields, fieldName)
		}
	}

	workRequestLog := findResource(t, pkg.Resources, "WorkRequestLog")
	for _, fieldName := range []string{"Message", "Timestamp"} {
		if !hasField(workRequestLog.StatusFields, fieldName) {
			t.Fatalf("WorkRequestLog status fields = %#v, want %s", workRequestLog.StatusFields, fieldName)
		}
	}

	workRequest := findResource(t, pkg.Resources, "WorkRequest")
	workRequestStatus := findFieldModel(t, workRequest.StatusFields, "Status")
	if workRequestStatus.Tag != `json:"sdkStatus,omitempty"` {
		t.Fatalf("WorkRequest Status tag = %q, want sdkStatus collision escape", workRequestStatus.Tag)
	}

	credentialRotationStatus := findResource(t, pkg.Resources, "CredentialRotationStatus")
	credentialRotationObservedStatus := findFieldModel(t, credentialRotationStatus.StatusFields, "Status")
	if credentialRotationObservedStatus.Tag != `json:"sdkStatus,omitempty"` {
		t.Fatalf("CredentialRotationStatus Status tag = %q, want sdkStatus collision escape", credentialRotationObservedStatus.Tag)
	}
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

	bulkAction := findResource(t, pkg.Resources, "BulkActionResourceType")
	if !hasField(bulkAction.StatusFields, "Items") {
		t.Fatalf("BulkActionResourceType status fields = %#v, want Items", bulkAction.StatusFields)
	}

	bulkEdit := findResource(t, pkg.Resources, "BulkEditTagsResourceType")
	if !hasField(bulkEdit.StatusFields, "Items") {
		t.Fatalf("BulkEditTagsResourceType status fields = %#v, want Items", bulkEdit.StatusFields)
	}

	costTrackingTag := findResource(t, pkg.Resources, "CostTrackingTag")
	for _, fieldName := range []string{"TagNamespaceId", "TagNamespaceName", "IsRetired", "Validator"} {
		if !hasField(costTrackingTag.StatusFields, fieldName) {
			t.Fatalf("CostTrackingTag status fields = %#v, want %s", costTrackingTag.StatusFields, fieldName)
		}
	}

	identityProvider := findResource(t, pkg.Resources, "IdentityProvider")
	for _, fieldName := range []string{"CompartmentId", "Name", "Description", "Metadata", "MetadataUrl", "ProductType"} {
		if !hasField(identityProvider.StatusFields, fieldName) {
			t.Fatalf("IdentityProvider status fields = %#v, want %s", identityProvider.StatusFields, fieldName)
		}
	}

	networkSource := findResource(t, pkg.Resources, "NetworkSource")
	for _, fieldName := range []string{"CompartmentId", "Name", "Description", "PublicSourceList", "Services", "VirtualSourceList"} {
		if !hasField(networkSource.StatusFields, fieldName) {
			t.Fatalf("NetworkSource status fields = %#v, want %s", networkSource.StatusFields, fieldName)
		}
	}

	orResetUIPassword := findResource(t, pkg.Resources, "OrResetUIPassword")
	for _, fieldName := range []string{"Password", "UserId", "TimeCreated", "LifecycleState", "InactiveStatus"} {
		if !hasField(orResetUIPassword.StatusFields, fieldName) {
			t.Fatalf("OrResetUIPassword status fields = %#v, want %s", orResetUIPassword.StatusFields, fieldName)
		}
	}

	standardTagNamespace := findResource(t, pkg.Resources, "StandardTagNamespace")
	for _, fieldName := range []string{"Description", "StandardTagNamespaceName", "TagDefinitionTemplates"} {
		if !hasField(standardTagNamespace.StatusFields, fieldName) {
			t.Fatalf("StandardTagNamespace status fields = %#v, want %s", standardTagNamespace.StatusFields, fieldName)
		}
	}
	standardTagNamespaceStatus := findFieldModel(t, standardTagNamespace.StatusFields, "Status")
	if standardTagNamespaceStatus.Tag != `json:"sdkStatus,omitempty"` {
		t.Fatalf("StandardTagNamespace Status tag = %q, want sdkStatus collision escape", standardTagNamespaceStatus.Tag)
	}

	standardTagTemplate := findResource(t, pkg.Resources, "StandardTagTemplate")
	for _, fieldName := range []string{"Description", "TagDefinitionName", "Type", "IsCostTracking"} {
		if !hasField(standardTagTemplate.StatusFields, fieldName) {
			t.Fatalf("StandardTagTemplate status fields = %#v, want %s", standardTagTemplate.StatusFields, fieldName)
		}
	}

	userState := findResource(t, pkg.Resources, "UserState")
	for _, fieldName := range []string{"Id", "CompartmentId", "Name", "LifecycleState", "Capabilities"} {
		if !hasField(userState.StatusFields, fieldName) {
			t.Fatalf("UserState status fields = %#v, want %s", userState.StatusFields, fieldName)
		}
	}

	userUIPasswordInformation := findResource(t, pkg.Resources, "UserUIPasswordInformation")
	for _, fieldName := range []string{"UserId", "TimeCreated", "LifecycleState"} {
		if !hasField(userUIPasswordInformation.StatusFields, fieldName) {
			t.Fatalf("UserUIPasswordInformation status fields = %#v, want %s", userUIPasswordInformation.StatusFields, fieldName)
		}
	}
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

	clusterNetworkInstance := findResource(t, pkg.Resources, "ClusterNetworkInstance")
	for _, fieldName := range []string{"AvailabilityDomain", "CompartmentId", "Region", "State", "TimeCreated"} {
		if !hasField(clusterNetworkInstance.StatusFields, fieldName) {
			t.Fatalf("ClusterNetworkInstance status fields = %#v, want %s", clusterNetworkInstance.StatusFields, fieldName)
		}
	}

	computeCapacityReservationInstance := findResource(t, pkg.Resources, "ComputeCapacityReservationInstance")
	for _, fieldName := range []string{"AvailabilityDomain", "CompartmentId", "Id", "Shape"} {
		if !hasField(computeCapacityReservationInstance.StatusFields, fieldName) {
			t.Fatalf("ComputeCapacityReservationInstance status fields = %#v, want %s", computeCapacityReservationInstance.StatusFields, fieldName)
		}
	}

	computeGlobalImageCapabilitySchema := findResource(t, pkg.Resources, "ComputeGlobalImageCapabilitySchema")
	for _, fieldName := range []string{"ComputeGlobalImageCapabilitySchemaId", "Name"} {
		if !hasField(computeGlobalImageCapabilitySchema.StatusFields, fieldName) {
			t.Fatalf("ComputeGlobalImageCapabilitySchema status fields = %#v, want %s", computeGlobalImageCapabilitySchema.StatusFields, fieldName)
		}
	}

	networkSecurityGroupSecurityRule := findResource(t, pkg.Resources, "NetworkSecurityGroupSecurityRule")
	for _, fieldName := range []string{"Direction", "Protocol", "Id", "TcpOptions", "UdpOptions"} {
		if !hasField(networkSecurityGroupSecurityRule.StatusFields, fieldName) {
			t.Fatalf("NetworkSecurityGroupSecurityRule status fields = %#v, want %s", networkSecurityGroupSecurityRule.StatusFields, fieldName)
		}
	}

	ipSecConnectionTunnelError := findResource(t, pkg.Resources, "IPSecConnectionTunnelError")
	for _, fieldName := range []string{"ErrorCode", "ErrorDescription", "Id", "Solution", "Timestamp"} {
		if !hasField(ipSecConnectionTunnelError.StatusFields, fieldName) {
			t.Fatalf("IPSecConnectionTunnelError status fields = %#v, want %s", ipSecConnectionTunnelError.StatusFields, fieldName)
		}
	}

	ipSecConnectionTunnelRoute := findResource(t, pkg.Resources, "IPSecConnectionTunnelRoute")
	for _, fieldName := range []string{"Advertiser", "AsPath", "IsBestPath", "Prefix"} {
		if !hasField(ipSecConnectionTunnelRoute.StatusFields, fieldName) {
			t.Fatalf("IPSecConnectionTunnelRoute status fields = %#v, want %s", ipSecConnectionTunnelRoute.StatusFields, fieldName)
		}
	}

	ipSecConnectionTunnelSecurityAssociation := findResource(t, pkg.Resources, "IPSecConnectionTunnelSecurityAssociation")
	for _, fieldName := range []string{"CpeSubnet", "OracleSubnet", "TunnelSaStatus"} {
		if !hasField(ipSecConnectionTunnelSecurityAssociation.StatusFields, fieldName) {
			t.Fatalf("IPSecConnectionTunnelSecurityAssociation status fields = %#v, want %s", ipSecConnectionTunnelSecurityAssociation.StatusFields, fieldName)
		}
	}

	instanceDevice := findResource(t, pkg.Resources, "InstanceDevice")
	for _, fieldName := range []string{"IsAvailable", "Name"} {
		if !hasField(instanceDevice.StatusFields, fieldName) {
			t.Fatalf("InstanceDevice status fields = %#v, want %s", instanceDevice.StatusFields, fieldName)
		}
	}

	volumeBackupPolicyAssetAssignment := findResource(t, pkg.Resources, "VolumeBackupPolicyAssetAssignment")
	for _, fieldName := range []string{"AssetId", "Id", "PolicyId", "TimeCreated"} {
		if !hasField(volumeBackupPolicyAssetAssignment.StatusFields, fieldName) {
			t.Fatalf("VolumeBackupPolicyAssetAssignment status fields = %#v, want %s", volumeBackupPolicyAssetAssignment.StatusFields, fieldName)
		}
	}

	windowsInstanceInitialCredential := findResource(t, pkg.Resources, "WindowsInstanceInitialCredential")
	for _, fieldName := range []string{"Password", "Username"} {
		if !hasField(windowsInstanceInitialCredential.StatusFields, fieldName) {
			t.Fatalf("WindowsInstanceInitialCredential status fields = %#v, want %s", windowsInstanceInitialCredential.StatusFields, fieldName)
		}
	}

	fastConnectProviderVirtualCircuitBandwidthShape := findResource(t, pkg.Resources, "FastConnectProviderVirtualCircuitBandwidthShape")
	for _, fieldName := range []string{"BandwidthInMbps", "Name"} {
		if !hasField(fastConnectProviderVirtualCircuitBandwidthShape.StatusFields, fieldName) {
			t.Fatalf("FastConnectProviderVirtualCircuitBandwidthShape status fields = %#v, want %s", fastConnectProviderVirtualCircuitBandwidthShape.StatusFields, fieldName)
		}
	}

	crossconnectPortSpeedShape := findResource(t, pkg.Resources, "CrossconnectPortSpeedShape")
	for _, fieldName := range []string{"Name", "PortSpeedInGbps"} {
		if !hasField(crossconnectPortSpeedShape.StatusFields, fieldName) {
			t.Fatalf("CrossconnectPortSpeedShape status fields = %#v, want %s", crossconnectPortSpeedShape.StatusFields, fieldName)
		}
	}

	allDrgAttachment := findResource(t, pkg.Resources, "AllDrgAttachment")
	if !hasField(allDrgAttachment.StatusFields, "Id") {
		t.Fatalf("AllDrgAttachment status fields = %#v, want Id", allDrgAttachment.StatusFields)
	}

	allowedPeerRegions := findResource(t, pkg.Resources, "AllowedPeerRegionsForRemotePeering")
	if !hasField(allowedPeerRegions.StatusFields, "Name") {
		t.Fatalf("AllowedPeerRegionsForRemotePeering status fields = %#v, want Name", allowedPeerRegions.StatusFields)
	}

	appCatalogListingAgreement := findResource(t, pkg.Resources, "AppCatalogListingAgreement")
	for _, fieldName := range []string{"ListingId", "ListingResourceVersion", "OracleTermsOfUseLink", "EulaLink", "TimeRetrieved", "Signature"} {
		if !hasField(appCatalogListingAgreement.StatusFields, fieldName) {
			t.Fatalf("AppCatalogListingAgreement status fields = %#v, want %s", appCatalogListingAgreement.StatusFields, fieldName)
		}
	}

	crossConnectLetterOfAuthority := findResource(t, pkg.Resources, "CrossConnectLetterOfAuthority")
	for _, fieldName := range []string{"CrossConnectId", "FacilityLocation", "TimeExpires"} {
		if !hasField(crossConnectLetterOfAuthority.StatusFields, fieldName) {
			t.Fatalf("CrossConnectLetterOfAuthority status fields = %#v, want %s", crossConnectLetterOfAuthority.StatusFields, fieldName)
		}
	}

	crossConnectMapping := findResource(t, pkg.Resources, "CrossConnectMapping")
	for _, fieldName := range []string{"Ipv4BgpStatus", "Ipv6BgpStatus", "OciLogicalDeviceName"} {
		if !hasField(crossConnectMapping.StatusFields, fieldName) {
			t.Fatalf("CrossConnectMapping status fields = %#v, want %s", crossConnectMapping.StatusFields, fieldName)
		}
	}

	dhcpOption := findResource(t, pkg.Resources, "DhcpOption")
	for _, fieldName := range []string{"CompartmentId", "DisplayName", "LifecycleState", "Options", "TimeCreated", "VcnId"} {
		if !hasField(dhcpOption.StatusFields, fieldName) {
			t.Fatalf("DhcpOption status fields = %#v, want %s", dhcpOption.StatusFields, fieldName)
		}
	}

	virtualCircuitAssociatedTunnel := findResource(t, pkg.Resources, "VirtualCircuitAssociatedTunnel")
	for _, fieldName := range []string{"TunnelId", "TunnelType"} {
		if !hasField(virtualCircuitAssociatedTunnel.StatusFields, fieldName) {
			t.Fatalf("VirtualCircuitAssociatedTunnel status fields = %#v, want %s", virtualCircuitAssociatedTunnel.StatusFields, fieldName)
		}
	}
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
	if len(result.Generated) != 1 {
		t.Fatalf("Generate() generated = %d services, want 1", len(result.Generated))
	}

	groupVersionPath := filepath.Join(outputRoot, "api", "mysql", "v1beta1", "groupversion_info.go")
	groupVersionContent, err := os.ReadFile(groupVersionPath)
	if err != nil {
		t.Fatalf("read %s: %v", groupVersionPath, err)
	}
	if !strings.Contains(string(groupVersionContent), "// Code generated by generator. DO NOT EDIT.") {
		t.Fatalf("groupversion_info.go did not include the canonical generator banner: %s", string(groupVersionContent))
	}
	if !strings.Contains(string(groupVersionContent), `GroupVersion = schema.GroupVersion{Group: "mysql.oracle.com", Version: "v1beta1"}`) {
		t.Fatalf("groupversion_info.go did not contain the expected GroupVersion: %s", string(groupVersionContent))
	}

	resourcePath := filepath.Join(outputRoot, "api", "mysql", "v1beta1", "mysqldbsystem_types.go")
	resourceContent, err := os.ReadFile(resourcePath)
	if err != nil {
		t.Fatalf("read %s: %v", resourcePath, err)
	}
	if !strings.Contains(string(resourceContent), "type MySqlDbSystemSpec struct") {
		t.Fatalf("mysqldbsystem_types.go did not render the expected kind: %s", string(resourceContent))
	}
	if !strings.Contains(string(resourceContent), "Port") || !strings.Contains(string(resourceContent), `json:"port,omitempty"`) {
		t.Fatalf("mysqldbsystem_types.go did not render the expected Port field: %s", string(resourceContent))
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
	if !slices.Equal(compartmentID.Markers, []string{"+kubebuilder:validation:Required"}) {
		t.Fatalf("Application CompartmentId markers = %#v, want required marker", compartmentID.Markers)
	}
	if compartmentID.Tag != `json:"compartmentId"` {
		t.Fatalf("Application CompartmentId tag = %q, want required json tag", compartmentID.Tag)
	}
	if !strings.Contains(strings.Join(compartmentID.Comments, "\n"), "compartment to create the application within") {
		t.Fatalf("Application CompartmentId comments = %#v, want SDK documentation", compartmentID.Comments)
	}

	config := findFieldModel(t, application.SpecFields, "Config")
	if !slices.Equal(config.Markers, []string{"+kubebuilder:validation:Optional"}) {
		t.Fatalf("Application Config markers = %#v, want optional marker", config.Markers)
	}
	if config.Tag != `json:"config,omitempty"` {
		t.Fatalf("Application Config tag = %q, want optional json tag", config.Tag)
	}
	if !strings.Contains(strings.Join(config.Comments, "\n"), "Application configuration") {
		t.Fatalf("Application Config comments = %#v, want SDK documentation", config.Comments)
	}

	lifecycleState := findFieldModel(t, application.StatusFields, "LifecycleState")
	if len(lifecycleState.Markers) != 0 {
		t.Fatalf("Application LifecycleState markers = %#v, want no requiredness markers on status fields", lifecycleState.Markers)
	}
	if !strings.Contains(strings.Join(lifecycleState.Comments, "\n"), "current state of the application") {
		t.Fatalf("Application LifecycleState comments = %#v, want SDK documentation", lifecycleState.Comments)
	}

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

func TestCheckedInCompatibilityLockedServicesMatchGenerator(t *testing.T) {
	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	var services []ServiceConfig
	for _, service := range cfg.Services {
		if slices.Contains([]string{"database", "mysql", "streaming"}, service.Service) {
			services = append(services, service)
		}
	}
	if len(services) != 3 {
		t.Fatalf("selected %d compatibility-locked services, want 3", len(services))
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
	result, err := pipeline.Generate(context.Background(), cfg, services, Options{
		OutputRoot:                      outputRoot,
		PreserveExistingSpecSurfaceRoot: repoRoot(t),
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Generated) != 3 {
		t.Fatalf("Generate() generated %d services, want 3", len(result.Generated))
	}
	wantResourceCounts := map[string]int{
		"database":  79,
		"mysql":     12,
		"streaming": 7,
	}
	for _, generated := range result.Generated {
		if generated.ResourceCount != wantResourceCounts[generated.Service] {
			t.Fatalf("service %s generated %d resources, want %d", generated.Service, generated.ResourceCount, wantResourceCounts[generated.Service])
		}
	}

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
	for _, relativePath := range apiFiles {
		assertGeneratedGoMatches(t, filepath.Join(repoRoot(t), relativePath), filepath.Join(outputRoot, relativePath))
	}

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
	for _, relativePath := range exactFiles {
		assertExactFileMatch(t, filepath.Join(repoRoot(t), relativePath), filepath.Join(outputRoot, relativePath))
	}

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
	for _, relativePath := range runtimeFiles {
		assertGeneratedGoMatches(t, filepath.Join(repoRoot(t), relativePath), filepath.Join(outputRoot, relativePath))
	}

	assertResourceOrderContainsSubset(
		t,
		filepath.Join(repoRoot(t), "config", "samples", "kustomization.yaml"),
		filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"),
	)
}

func TestCheckedInPromotedCoreRuntimeArtifactsMatchGenerator(t *testing.T) {
	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	var coreService *ServiceConfig
	for i := range cfg.Services {
		if cfg.Services[i].Service == "core" {
			coreService = &cfg.Services[i]
			break
		}
	}
	if coreService == nil {
		t.Fatal("core service was not found in services.yaml")
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
	result, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{*coreService}, Options{
		OutputRoot:                      outputRoot,
		PreserveExistingSpecSurfaceRoot: repoRoot(t),
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Generated) != 1 {
		t.Fatalf("Generate() generated %d services, want 1", len(result.Generated))
	}

	runtimeFiles := []string{
		"controllers/core/vcn_controller.go",
		"pkg/servicemanager/core/vcn/vcn_serviceclient.go",
		"pkg/servicemanager/core/vcn/vcn_servicemanager.go",
		"internal/registrations/core_generated.go",
	}
	for _, relativePath := range runtimeFiles {
		assertGeneratedGoMatches(t, filepath.Join(repoRoot(t), relativePath), filepath.Join(outputRoot, relativePath))
	}

	exactFiles := []string{
		"packages/core/metadata.env",
		"packages/core/install/kustomization.yaml",
	}
	for _, relativePath := range exactFiles {
		assertExactFileMatch(t, filepath.Join(repoRoot(t), relativePath), filepath.Join(outputRoot, relativePath))
	}
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

func TestMySQLCompatibilityLockedKindIncludesOptionalDesiredStateFields(t *testing.T) {
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

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "generated.go", source, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("ParseFile() error = %v\nsource:\n%s", err, source)
	}

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
			if !ok || structType.Fields == nil {
				t.Fatalf("type %q was not a struct in source:\n%s", typeName, source)
			}
			names := make([]string, 0, len(structType.Fields.List))
			for _, field := range structType.Fields.List {
				for _, name := range field.Names {
					names = append(names, name.Name)
				}
			}
			return names
		}
	}

	t.Fatalf("type %q was not found in source:\n%s", typeName, source)
	return nil
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

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "generated.go", source, parser.SkipObjectResolution)
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
		switch concrete := decl.(type) {
		case *ast.GenDecl:
			concrete.Doc = nil
			for _, spec := range concrete.Specs {
				switch typed := spec.(type) {
				case *ast.TypeSpec:
					typed.Doc = nil
					typed.Comment = nil
					if structType, ok := typed.Type.(*ast.StructType); ok && structType.Fields != nil {
						for _, field := range structType.Fields.List {
							field.Doc = nil
							field.Comment = nil
						}
					}
				case *ast.ValueSpec:
					typed.Doc = nil
					typed.Comment = nil
				}
			}
		case *ast.FuncDecl:
			concrete.Doc = nil
		}
	}
}
