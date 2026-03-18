package ocisdk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestIndexCachesPackageLoads(t *testing.T) {
	t.Parallel()

	var resolveCalls int
	index := NewIndex(func(context.Context, string) (string, error) {
		resolveCalls++
		return filepath.Join(moduleRoot(t), "internal", "generator", "testdata", "sdk", "sample"), nil
	})

	const importPath = "example.com/test/sdk"
	if _, err := index.Package(context.Background(), importPath); err != nil {
		t.Fatalf("Package() error = %v", err)
	}
	if _, err := index.Package(context.Background(), importPath); err != nil {
		t.Fatalf("Package() second call error = %v", err)
	}
	if _, ok, err := index.Struct(context.Background(), importPath, "CreateWidgetDetails"); err != nil {
		t.Fatalf("Struct(CreateWidgetDetails) error = %v", err)
	} else if !ok {
		t.Fatal("Struct(CreateWidgetDetails) did not find type")
	}
	if _, ok, err := index.Struct(context.Background(), importPath, "UpdateWidgetDetails"); err != nil {
		t.Fatalf("Struct(UpdateWidgetDetails) error = %v", err)
	} else if !ok {
		t.Fatal("Struct(UpdateWidgetDetails) did not find type")
	}

	if resolveCalls != 1 {
		t.Fatalf("resolveDir calls = %d, want 1", resolveCalls)
	}
}

func TestIndexLoadsRepresentativeOCISDKMetadata(t *testing.T) {
	t.Parallel()

	index := NewIndex(vendorResolver(t))

	tests := []struct {
		name       string
		importPath string
		typeName   string
		assert     func(*testing.T, Struct)
	}{
		{
			name:       "functions.CreateApplicationDetails",
			importPath: "github.com/oracle/oci-go-sdk/v65/functions",
			typeName:   "CreateApplicationDetails",
			assert: func(t *testing.T, structMeta Struct) {
				compartmentID := findField(t, structMeta.Fields, "CompartmentId")
				if compartmentID.JSONName != "compartmentId" {
					t.Fatalf("CompartmentId JSON name = %q, want compartmentId", compartmentID.JSONName)
				}
				if !compartmentID.Mandatory {
					t.Fatal("CompartmentId should be mandatory")
				}
				if compartmentID.Kind != FieldKindScalar {
					t.Fatalf("CompartmentId kind = %q, want %q", compartmentID.Kind, FieldKindScalar)
				}
				if !strings.Contains(compartmentID.Documentation, "compartment to create the application within") {
					t.Fatalf("CompartmentId documentation = %q, want compartment text", compartmentID.Documentation)
				}

				config := findField(t, structMeta.Fields, "Config")
				if config.RenderableType != "map[string]string" {
					t.Fatalf("Config renderable type = %q, want map[string]string", config.RenderableType)
				}

				traceConfig := findField(t, structMeta.Fields, "TraceConfig")
				if traceConfig.Kind != FieldKindStruct {
					t.Fatalf("TraceConfig kind = %q, want %q", traceConfig.Kind, FieldKindStruct)
				}
				if traceConfig.Type != "*ApplicationTraceConfig" {
					t.Fatalf("TraceConfig type = %q, want *ApplicationTraceConfig", traceConfig.Type)
				}
				domainID := findField(t, traceConfig.NestedFields, "DomainId")
				if domainID.JSONName != "domainId" {
					t.Fatalf("TraceConfig.DomainId JSON name = %q, want domainId", domainID.JSONName)
				}
				if !strings.Contains(domainID.Documentation, "collector") {
					t.Fatalf("TraceConfig.DomainId documentation = %q, want collector text", domainID.Documentation)
				}

				imagePolicy := findField(t, structMeta.Fields, "ImagePolicyConfig")
				if imagePolicy.Kind != FieldKindStruct {
					t.Fatalf("ImagePolicyConfig kind = %q, want %q", imagePolicy.Kind, FieldKindStruct)
				}
				isPolicyEnabled := findField(t, imagePolicy.NestedFields, "IsPolicyEnabled")
				if !isPolicyEnabled.Mandatory {
					t.Fatal("ImagePolicyConfig.IsPolicyEnabled should be mandatory")
				}
			},
		},
		{
			name:       "core.CreateIpSecConnectionTunnelDetails",
			importPath: "github.com/oracle/oci-go-sdk/v65/core",
			typeName:   "CreateIpSecConnectionTunnelDetails",
			assert: func(t *testing.T, structMeta Struct) {
				displayName := findField(t, structMeta.Fields, "DisplayName")
				if displayName.JSONName != "displayName" {
					t.Fatalf("DisplayName JSON name = %q, want displayName", displayName.JSONName)
				}
				if !strings.Contains(displayName.Documentation, "A user-friendly name") {
					t.Fatalf("DisplayName documentation = %q, want display-name text", displayName.Documentation)
				}

				associatedVirtualCircuits := findField(t, structMeta.Fields, "AssociatedVirtualCircuits")
				if associatedVirtualCircuits.RenderableType != "[]string" {
					t.Fatalf("AssociatedVirtualCircuits renderable type = %q, want []string", associatedVirtualCircuits.RenderableType)
				}

				bgpSession := findField(t, structMeta.Fields, "BgpSessionConfig")
				if bgpSession.Kind != FieldKindStruct {
					t.Fatalf("BgpSessionConfig kind = %q, want %q", bgpSession.Kind, FieldKindStruct)
				}
				if bgpSession.Type != "*CreateIpSecTunnelBgpSessionDetails" {
					t.Fatalf("BgpSessionConfig type = %q, want *CreateIpSecTunnelBgpSessionDetails", bgpSession.Type)
				}
				customerBgpASN := findField(t, bgpSession.NestedFields, "CustomerBgpAsn")
				if customerBgpASN.JSONName != "customerBgpAsn" {
					t.Fatalf("BgpSessionConfig.CustomerBgpAsn JSON name = %q, want customerBgpAsn", customerBgpASN.JSONName)
				}
				if !strings.Contains(customerBgpASN.Documentation, "BGP session") {
					t.Fatalf("BgpSessionConfig.CustomerBgpAsn documentation = %q, want BGP session text", customerBgpASN.Documentation)
				}
			},
		},
		{
			name:       "certificates.CertificateBundlePublicOnly",
			importPath: "github.com/oracle/oci-go-sdk/v65/certificates",
			typeName:   "CertificateBundlePublicOnly",
			assert: func(t *testing.T, structMeta Struct) {
				certificateID := findField(t, structMeta.Fields, "CertificateId")
				if certificateID.JSONName != "certificateId" {
					t.Fatalf("CertificateId JSON name = %q, want certificateId", certificateID.JSONName)
				}
				if !certificateID.Mandatory {
					t.Fatal("CertificateId should be mandatory")
				}
				if !strings.Contains(certificateID.Documentation, "OCID of the certificate") {
					t.Fatalf("CertificateId documentation = %q, want OCID text", certificateID.Documentation)
				}

				timeCreated := findField(t, structMeta.Fields, "TimeCreated")
				if timeCreated.RenderableType != "string" {
					t.Fatalf("TimeCreated renderable type = %q, want string", timeCreated.RenderableType)
				}
				if !timeCreated.Mandatory {
					t.Fatal("TimeCreated should be mandatory")
				}

				validity := findField(t, structMeta.Fields, "Validity")
				if validity.Kind != FieldKindStruct {
					t.Fatalf("Validity kind = %q, want %q", validity.Kind, FieldKindStruct)
				}
				notBefore := findField(t, validity.NestedFields, "TimeOfValidityNotBefore")
				if !notBefore.Mandatory {
					t.Fatal("Validity.TimeOfValidityNotBefore should be mandatory")
				}
				if notBefore.RenderableType != "string" {
					t.Fatalf("Validity.TimeOfValidityNotBefore renderable type = %q, want string", notBefore.RenderableType)
				}

				stages := findField(t, structMeta.Fields, "Stages")
				if stages.RenderableType != "[]string" {
					t.Fatalf("Stages renderable type = %q, want []string", stages.RenderableType)
				}

				revocationStatus := findField(t, structMeta.Fields, "RevocationStatus")
				if revocationStatus.Kind != FieldKindStruct {
					t.Fatalf("RevocationStatus kind = %q, want %q", revocationStatus.Kind, FieldKindStruct)
				}
				revocationReason := findField(t, revocationStatus.NestedFields, "RevocationReason")
				if revocationReason.RenderableType != "string" {
					t.Fatalf("RevocationStatus.RevocationReason renderable type = %q, want string", revocationReason.RenderableType)
				}
			},
		},
		{
			name:       "networkloadbalancer.HealthChecker",
			importPath: "github.com/oracle/oci-go-sdk/v65/networkloadbalancer",
			typeName:   "HealthChecker",
			assert: func(t *testing.T, structMeta Struct) {
				requestData := findField(t, structMeta.Fields, "RequestData")
				if requestData.RenderableType != "string" {
					t.Fatalf("RequestData renderable type = %q, want string", requestData.RenderableType)
				}

				responseData := findField(t, structMeta.Fields, "ResponseData")
				if responseData.RenderableType != "string" {
					t.Fatalf("ResponseData renderable type = %q, want string", responseData.RenderableType)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			structMeta, ok, err := index.Struct(context.Background(), tt.importPath, tt.typeName)
			if err != nil {
				t.Fatalf("Struct(%s) error = %v", tt.name, err)
			}
			if !ok {
				t.Fatalf("Struct(%s) did not find type", tt.name)
			}
			tt.assert(t, structMeta)
		})
	}
}

func TestIndexLoadsPolymorphicInterfaceFamilies(t *testing.T) {
	t.Parallel()

	index := NewIndex(vendorResolver(t))

	tests := []struct {
		name       string
		importPath string
		typeName   string
		assert     func(*testing.T, InterfaceFamily)
	}{
		{
			name:       "functions.FunctionSourceDetails",
			importPath: "github.com/oracle/oci-go-sdk/v65/functions",
			typeName:   "FunctionSourceDetails",
			assert: func(t *testing.T, family InterfaceFamily) {
				sourceType := findField(t, family.Base.Fields, "SourceType")
				if sourceType.JSONName != "sourceType" {
					t.Fatalf("SourceType JSON name = %q, want sourceType", sourceType.JSONName)
				}

				preBuilt := findStruct(t, family.Implementations, "PreBuiltFunctionSourceDetails")
				pbfListingID := findField(t, preBuilt.Fields, "PbfListingId")
				if pbfListingID.JSONName != "pbfListingId" {
					t.Fatalf("PreBuiltFunctionSourceDetails.PbfListingId JSON name = %q, want pbfListingId", pbfListingID.JSONName)
				}
				if !pbfListingID.Mandatory {
					t.Fatal("PreBuiltFunctionSourceDetails.PbfListingId should be mandatory")
				}
			},
		},
		{
			name:       "functions.FunctionProvisionedConcurrencyConfig",
			importPath: "github.com/oracle/oci-go-sdk/v65/functions",
			typeName:   "FunctionProvisionedConcurrencyConfig",
			assert: func(t *testing.T, family InterfaceFamily) {
				strategy := findField(t, family.Base.Fields, "Strategy")
				if strategy.JSONName != "strategy" {
					t.Fatalf("Strategy JSON name = %q, want strategy", strategy.JSONName)
				}

				constant := findStruct(t, family.Implementations, "ConstantProvisionedConcurrencyConfig")
				count := findField(t, constant.Fields, "Count")
				if !count.Mandatory {
					t.Fatal("ConstantProvisionedConcurrencyConfig.Count should be mandatory")
				}
			},
		},
		{
			name:       "vault.SecretContentDetails",
			importPath: "github.com/oracle/oci-go-sdk/v65/vault",
			typeName:   "SecretContentDetails",
			assert: func(t *testing.T, family InterfaceFamily) {
				contentType := findField(t, family.Base.Fields, "ContentType")
				if contentType.JSONName != "contentType" {
					t.Fatalf("ContentType JSON name = %q, want contentType", contentType.JSONName)
				}

				base64Content := findStruct(t, family.Implementations, "Base64SecretContentDetails")
				content := findField(t, base64Content.Fields, "Content")
				if content.JSONName != "content" {
					t.Fatalf("Base64SecretContentDetails.Content JSON name = %q, want content", content.JSONName)
				}
			},
		},
		{
			name:       "secrets.SecretBundleContentDetails",
			importPath: "github.com/oracle/oci-go-sdk/v65/secrets",
			typeName:   "SecretBundleContentDetails",
			assert: func(t *testing.T, family InterfaceFamily) {
				contentType := findField(t, family.Base.Fields, "ContentType")
				if contentType.JSONName != "contentType" {
					t.Fatalf("SecretBundleContentDetails.ContentType JSON name = %q, want contentType", contentType.JSONName)
				}

				base64Content := findStruct(t, family.Implementations, "Base64SecretBundleContentDetails")
				content := findField(t, base64Content.Fields, "Content")
				if content.JSONName != "content" {
					t.Fatalf("Base64SecretBundleContentDetails.Content JSON name = %q, want content", content.JSONName)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pkg, err := index.Package(context.Background(), tt.importPath)
			if err != nil {
				t.Fatalf("Package(%s) error = %v", tt.importPath, err)
			}

			family, ok := pkg.InterfaceFamily(tt.typeName)
			if !ok {
				t.Fatalf("InterfaceFamily(%s) did not find a family", tt.typeName)
			}
			tt.assert(t, family)
		})
	}
}

func TestPackageRequestBodyPayloads(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		importPath string
		request    string
		want       []string
	}{
		{
			name:       "sample alias payload",
			importPath: "example.com/test/sdk",
			request:    "CreateOAuthClientCredentialRequest",
			want:       []string{"CreateOAuth2ClientCredentialDetails"},
		},
		{
			name:       "sample non-crud payload",
			importPath: "example.com/test/sdk",
			request:    "GetReportByNameRequest",
			want:       []string{"GetReportByNameDetails"},
		},
		{
			name:       "core create plural alias",
			importPath: "github.com/oracle/oci-go-sdk/v65/core",
			request:    "CreateDhcpOptionsRequest",
			want:       []string{"CreateDhcpDetails"},
		},
		{
			name:       "core get by ip address",
			importPath: "github.com/oracle/oci-go-sdk/v65/core",
			request:    "GetPublicIpByIpAddressRequest",
			want:       []string{"GetPublicIpByIpAddressDetails"},
		},
		{
			name:       "identity oauth alias",
			importPath: "github.com/oracle/oci-go-sdk/v65/identity",
			request:    "UpdateOAuthClientCredentialRequest",
			want:       []string{"UpdateOAuth2ClientCredentialDetails"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := vendorResolver(t)
			if strings.HasPrefix(tt.importPath, "example.com/") {
				resolver = func(context.Context, string) (string, error) {
					return filepath.Join(moduleRoot(t), "internal", "generator", "testdata", "sdk", "sample"), nil
				}
			}

			index := NewIndex(resolver)
			pkg, err := index.Package(context.Background(), tt.importPath)
			if err != nil {
				t.Fatalf("Package(%s) error = %v", tt.importPath, err)
			}

			got := pkg.RequestBodyPayloads(tt.request)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("RequestBodyPayloads(%s) = %v, want %v", tt.request, got, tt.want)
			}
		})
	}
}

func TestIndexFlagsDeprecatedAndReadOnlyDocumentation(t *testing.T) {
	t.Parallel()

	index := NewIndex(vendorResolver(t))

	coreVolumeDetails, ok, err := index.Struct(context.Background(), "github.com/oracle/oci-go-sdk/v65/core", "CreateVolumeDetails")
	if err != nil {
		t.Fatalf("Struct(CreateVolumeDetails) error = %v", err)
	}
	if !ok {
		t.Fatal("Struct(CreateVolumeDetails) did not find type")
	}
	sizeInMBs := findField(t, coreVolumeDetails.Fields, "SizeInMBs")
	if !sizeInMBs.Deprecated {
		t.Fatal("CreateVolumeDetails.SizeInMBs should be marked deprecated")
	}

	attachISCSIVolumeDetails, ok, err := index.Struct(context.Background(), "github.com/oracle/oci-go-sdk/v65/core", "AttachIScsiVolumeDetails")
	if err != nil {
		t.Fatalf("Struct(AttachIScsiVolumeDetails) error = %v", err)
	}
	if !ok {
		t.Fatal("Struct(AttachIScsiVolumeDetails) did not find type")
	}
	isReadOnly := findField(t, attachISCSIVolumeDetails.Fields, "IsReadOnly")
	if !isReadOnly.ReadOnly {
		t.Fatal("AttachIScsiVolumeDetails.IsReadOnly should be marked readOnly")
	}
}

func vendorResolver(t *testing.T) ResolveDirFunc {
	t.Helper()

	root := moduleRoot(t)
	return func(_ context.Context, importPath string) (string, error) {
		dir := filepath.Join(root, "vendor", filepath.FromSlash(importPath))
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, nil
		}
		return "", fmt.Errorf("package %q is not vendored", importPath)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("unable to locate module root from %s", dir)
		}
		dir = parent
	}
}

func findField(t *testing.T, fields []Field, name string) Field {
	t.Helper()

	for _, field := range fields {
		if field.Name == name {
			return field
		}
	}
	t.Fatalf("field %q not found", name)
	return Field{}
}

func findStruct(t *testing.T, structs []Struct, name string) Struct {
	t.Helper()

	for _, structMeta := range structs {
		if structMeta.Name == name {
			return structMeta
		}
	}
	t.Fatalf("struct %q not found", name)
	return Struct{}
}
