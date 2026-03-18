/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"context"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
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

func TestCurrentServiceParityMatchesCheckedInArtifacts(t *testing.T) {
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
		t.Fatalf("selected %d parity services, want 3", len(services))
	}

	outputRoot := t.TempDir()
	pipeline := New()
	result, err := pipeline.Generate(context.Background(), cfg, services, Options{
		OutputRoot: outputRoot,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Generated) != 3 {
		t.Fatalf("Generate() generated %d services, want 3", len(result.Generated))
	}
	for _, generated := range result.Generated {
		if generated.ResourceCount != 1 {
			t.Fatalf("service %s generated %d resources, want 1", generated.Service, generated.ResourceCount)
		}
	}

	apiFiles := []string{
		"api/database/v1beta1/groupversion_info.go",
		"api/database/v1beta1/autonomousdatabases_types.go",
		"api/mysql/v1beta1/groupversion_info.go",
		"api/mysql/v1beta1/mysqldbsystem_types.go",
		"api/streaming/v1beta1/groupversion_info.go",
		"api/streaming/v1beta1/stream_types.go",
	}
	for _, relativePath := range apiFiles {
		assertGoParity(t, filepath.Join(repoRoot(t), relativePath), filepath.Join(outputRoot, relativePath))
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

	assertResourceOrderContainsSubset(
		t,
		filepath.Join(repoRoot(t), "config", "samples", "kustomization.yaml"),
		filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"),
	)
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
		SDKPackage:     "example.com/test/sdk",
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

func assertGoParity(t *testing.T, wantPath string, gotPath string) {
	t.Helper()

	want := normalizeGoForParity(t, readFile(t, wantPath))
	got := normalizeGoForParity(t, readFile(t, gotPath))
	if want != got {
		t.Fatalf("Go parity mismatch for %s\nwant:\n%s\n\ngot:\n%s", wantPath, want, got)
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
