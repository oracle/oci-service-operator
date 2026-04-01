package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestValidatorRegistriesAreInSync(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot() error = %v", err)
	}

	apispecPath := filepath.Join(root, "internal", "validator", "apispec", "registry.go")
	sdkPath := filepath.Join(root, "internal", "validator", "sdk", "registry.go")

	existingAPI, err := parseExistingAPITargets(apispecPath)
	if err != nil {
		t.Fatalf("parseExistingAPITargets(%q) error = %v", apispecPath, err)
	}
	existingSDK, err := parseExistingSDKTargets(sdkPath)
	if err != nil {
		t.Fatalf("parseExistingSDKTargets(%q) error = %v", sdkPath, err)
	}

	apiOut, sdkOut, err := generateRegistryOutputs(root, "", false, existingAPI, existingSDK)
	if err != nil {
		t.Fatalf("generateRegistryOutputs() error = %v", err)
	}

	currentAPI, err := os.ReadFile(apispecPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", apispecPath, err)
	}
	if !bytes.Equal(currentAPI, apiOut) {
		t.Fatalf("%s is out of sync; run `go run ./hack/update_validator_registries.go --write`", rel(root, apispecPath))
	}

	currentSDK, err := os.ReadFile(sdkPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", sdkPath, err)
	}
	if !bytes.Equal(currentSDK, sdkOut) {
		t.Fatalf("%s is out of sync; run `go run ./hack/update_validator_registries.go --write`", rel(root, sdkPath))
	}
}

func TestDeriveSDKTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		service string
		group   string
		spec    string
		structs []string
		want    []string
	}{
		{
			name:    "service prefixed SDK type",
			service: "loadbalancer",
			group:   "loadbalancer",
			spec:    "Shape",
			structs: []string{"LoadBalancerShape"},
			want:    []string{"LoadBalancerShape"},
		},
		{
			name:    "entry fallback",
			service: "queue",
			group:   "queue",
			spec:    "WorkRequestLog",
			structs: []string{"WorkRequestLogEntry", "WorkRequestLogEntryCollection"},
			want:    []string{"WorkRequestLogEntry", "WorkRequestLogEntryCollection"},
		},
		{
			name:    "get details fallback",
			service: "core",
			group:   "core",
			spec:    "PublicIpByIpAddress",
			structs: []string{"GetPublicIpByIpAddressDetails"},
			want:    []string{"GetPublicIpByIpAddressDetails"},
		},
		{
			name:    "plural alias",
			service: "containerengine",
			group:   "containerengine",
			spec:    "ClusterOption",
			structs: []string{"ClusterOptions"},
			want:    []string{"ClusterOptions"},
		},
		{
			name:    "acronym casing variant",
			service: "loadbalancer",
			group:   "loadbalancer",
			spec:    "SSLCipherSuite",
			structs: []string{"SslCipherSuite"},
			want:    []string{"SslCipherSuite"},
		},
		{
			name:    "explicit override",
			service: "identity",
			group:   "identity",
			spec:    "IdentityProvider",
			structs: []string{"Saml2IdentityProvider"},
			want:    []string{"Saml2IdentityProvider"},
		},
		{
			name:    "metadata override",
			service: "objectstorage",
			group:   "objectstorage",
			spec:    "Namespace",
			structs: []string{"NamespaceMetadata"},
			want:    []string{"NamespaceMetadata"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			structs := make(map[string]bool, len(tt.structs))
			for _, name := range tt.structs {
				structs[name] = true
			}

			got := deriveSDKTypes(tt.service, tt.spec, makeTargetName(tt.group, tt.spec), structs)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("deriveSDKTypes(%q, %q) = %v, want %v", tt.service, tt.spec, got, tt.want)
			}
		})
	}
}

func TestBuildSDKMappingsFunctionsPbfListingExcludesVersionSummary(t *testing.T) {
	t.Parallel()

	got := buildSDKMappings("functions", "PbfListing", []string{
		"PbfListing",
		"PbfListingSummary",
		"PbfListingVersionSummary",
	}, false, specTarget{})

	if len(got) != 3 {
		t.Fatalf("len(buildSDKMappings(...)) = %d, want 3", len(got))
	}

	want := map[string]sdkMapping{
		"functions.PbfListing": {
			SDKStruct:  "functions.PbfListing",
			APISurface: "status",
		},
		"functions.PbfListingSummary": {
			SDKStruct:  "functions.PbfListingSummary",
			APISurface: "status",
		},
		"functions.PbfListingVersionSummary": {
			SDKStruct: "functions.PbfListingVersionSummary",
			Exclude:   true,
			Reason:    "Intentionally untracked: version summaries belong to the dedicated PbfListingVersion status surface.",
		},
	}

	assertExactMappings(t, got, want)
}

func TestFilterConfiguredAPISpecsAppliesSelectedKinds(t *testing.T) {
	t.Parallel()

	got, err := filterConfiguredAPISpecs(
		configuredService{
			Service:       "database",
			Group:         "database",
			Version:       "v1beta1",
			SelectedKinds: []string{"Widget"},
		},
		[]apiTypeInfo{
			{Spec: "Widget", Status: "WidgetStatus"},
			{Spec: "Report", Status: "ReportStatus"},
		},
	)
	if err != nil {
		t.Fatalf("filterConfiguredAPISpecs() error = %v", err)
	}
	if len(got) != 1 || got[0].Spec != "Widget" {
		t.Fatalf("filterConfiguredAPISpecs() = %#v, want Widget only", got)
	}
}

func TestFilterConfiguredAPISpecsRejectsMissingSelectedKinds(t *testing.T) {
	t.Parallel()

	_, err := filterConfiguredAPISpecs(
		configuredService{
			Service:       "database",
			Group:         "database",
			Version:       "v1beta1",
			SelectedKinds: []string{"Widget"},
		},
		[]apiTypeInfo{
			{Spec: "Report", Status: "ReportStatus"},
		},
	)
	if err == nil || !strings.Contains(err.Error(), "selected kinds Widget were not found") {
		t.Fatalf("filterConfiguredAPISpecs() error = %v, want missing selected kind failure", err)
	}
}

func TestBuildSDKTargetsPrunesExistingEntriesOutsideSelectedKinds(t *testing.T) {
	t.Parallel()

	got := buildSDKTargets(
		[]specTarget{
			{
				Service: "database",
				Group:   "database",
				Spec:    "AutonomousDatabase",
				SDKMappings: []sdkMapping{
					{SDKStruct: "database.CreateAutonomousDatabaseDetails"},
					{SDKStruct: "database.AutonomousDatabase"},
				},
			},
		},
		[]sdkTarget{
			{Group: "database", Type: "CreateAutonomousDatabaseDetails"},
			{Group: "database", Type: "AutonomousDatabase"},
			{Group: "database", Type: "CreateDbSystemDetails"},
			{Group: "database", Type: "DbSystem"},
		},
		[]configuredService{
			{
				Service:       "database",
				Group:         "database",
				Version:       "v1beta1",
				SelectedKinds: []string{"AutonomousDatabase"},
			},
		},
	)

	want := []sdkTarget{
		{Group: "database", Type: "CreateAutonomousDatabaseDetails"},
		{Group: "database", Type: "AutonomousDatabase"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("buildSDKTargets() = %#v, want %#v", got, want)
	}
}

func TestBuildSDKMappingsObjectStorageOverrides(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		spec       string
		candidates []string
		want       map[string]sdkMapping
	}{
		{
			name:       "namespace is intentionally untracked",
			spec:       "Namespace",
			candidates: []string{"NamespaceMetadata"},
			want: map[string]sdkMapping{
				"objectstorage.NamespaceMetadata": {
					SDKStruct: "objectstorage.NamespaceMetadata",
					Exclude:   true,
					Reason:    "Intentionally untracked: Namespace returns the namespace string in the response body; namespace metadata parity is tracked on ObjectStorageNamespaceMetadata.",
				},
			},
		},
		{
			name:       "object excludes version summary",
			spec:       "Object",
			candidates: []string{"ObjectSummary", "ObjectVersionSummary"},
			want: map[string]sdkMapping{
				"objectstorage.ObjectSummary": {
					SDKStruct:  "objectstorage.ObjectSummary",
					APISurface: "status",
				},
				"objectstorage.ObjectVersionSummary": {
					SDKStruct: "objectstorage.ObjectVersionSummary",
					Exclude:   true,
					Reason:    "Intentionally untracked: version summaries belong to the dedicated ObjectStorageObjectVersion status surface.",
				},
			},
		},
		{
			name:       "object version excludes collection",
			spec:       "ObjectVersion",
			candidates: []string{"ObjectVersionCollection", "ObjectVersionSummary"},
			want: map[string]sdkMapping{
				"objectstorage.ObjectVersionCollection": {
					SDKStruct: "objectstorage.ObjectVersionCollection",
					Exclude:   true,
					Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
				},
				"objectstorage.ObjectVersionSummary": {
					SDKStruct:  "objectstorage.ObjectVersionSummary",
					APISurface: "status",
				},
			},
		},
		{
			name:       "retention rule excludes collection",
			spec:       "RetentionRule",
			candidates: []string{"RetentionRule", "RetentionRuleCollection", "RetentionRuleDetails", "RetentionRuleSummary"},
			want: map[string]sdkMapping{
				"objectstorage.RetentionRule": {
					SDKStruct:  "objectstorage.RetentionRule",
					APISurface: "status",
				},
				"objectstorage.RetentionRuleCollection": {
					SDKStruct: "objectstorage.RetentionRuleCollection",
					Exclude:   true,
					Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
				},
				"objectstorage.RetentionRuleDetails": {
					SDKStruct:  "objectstorage.RetentionRuleDetails",
					APISurface: "status",
				},
				"objectstorage.RetentionRuleSummary": {
					SDKStruct:  "objectstorage.RetentionRuleSummary",
					APISurface: "status",
				},
			},
		},
		{
			name:       "work request log is status mapped",
			spec:       "WorkRequestLog",
			candidates: []string{"WorkRequestLogEntry"},
			want: map[string]sdkMapping{
				"objectstorage.WorkRequestLogEntry": {
					SDKStruct:  "objectstorage.WorkRequestLogEntry",
					APISurface: "status",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildSDKMappings("objectstorage", tt.spec, tt.candidates, false, specTarget{})
			assertExactMappings(t, got, tt.want)
		})
	}
}

func TestBuildSDKMappingsCertificatesManagementOverrides(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		spec       string
		candidates []string
		want       map[string]sdkMapping
	}{
		{
			name: "association excludes collection",
			spec: "Association",
			candidates: []string{
				"Association",
				"AssociationCollection",
				"AssociationSummary",
			},
			want: map[string]sdkMapping{
				"certificatesmanagement.Association": {
					SDKStruct:  "certificatesmanagement.Association",
					APISurface: "status",
				},
				"certificatesmanagement.AssociationCollection": {
					SDKStruct: "certificatesmanagement.AssociationCollection",
					Exclude:   true,
					Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
				},
				"certificatesmanagement.AssociationSummary": {
					SDKStruct:  "certificatesmanagement.AssociationSummary",
					APISurface: "status",
				},
			},
		},
		{
			name: "ca bundle excludes collection",
			spec: "CaBundle",
			candidates: []string{
				"CaBundle",
				"CaBundleCollection",
				"CaBundleSummary",
			},
			want: map[string]sdkMapping{
				"certificatesmanagement.CaBundle": {
					SDKStruct:  "certificatesmanagement.CaBundle",
					APISurface: "status",
				},
				"certificatesmanagement.CaBundleCollection": {
					SDKStruct: "certificatesmanagement.CaBundleCollection",
					Exclude:   true,
					Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
				},
				"certificatesmanagement.CaBundleSummary": {
					SDKStruct:  "certificatesmanagement.CaBundleSummary",
					APISurface: "status",
				},
			},
		},
		{
			name: "certificate excludes collection and nested version summary",
			spec: "Certificate",
			candidates: []string{
				"Certificate",
				"CertificateCollection",
				"CertificateSummary",
				"CertificateVersionSummary",
			},
			want: map[string]sdkMapping{
				"certificatesmanagement.Certificate": {
					SDKStruct:  "certificatesmanagement.Certificate",
					APISurface: "status",
				},
				"certificatesmanagement.CertificateCollection": {
					SDKStruct: "certificatesmanagement.CertificateCollection",
					Exclude:   true,
					Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
				},
				"certificatesmanagement.CertificateSummary": {
					SDKStruct:  "certificatesmanagement.CertificateSummary",
					APISurface: "status",
				},
				"certificatesmanagement.CertificateVersionSummary": {
					SDKStruct: "certificatesmanagement.CertificateVersionSummary",
					Exclude:   true,
					Reason:    "Intentionally untracked: version summaries belong to the dedicated CertificatesManagementCertificateVersion status surface.",
				},
			},
		},
		{
			name: "certificate authority excludes collection and nested version summary",
			spec: "CertificateAuthority",
			candidates: []string{
				"CertificateAuthority",
				"CertificateAuthorityCollection",
				"CertificateAuthoritySummary",
				"CertificateAuthorityVersionSummary",
			},
			want: map[string]sdkMapping{
				"certificatesmanagement.CertificateAuthority": {
					SDKStruct:  "certificatesmanagement.CertificateAuthority",
					APISurface: "status",
				},
				"certificatesmanagement.CertificateAuthorityCollection": {
					SDKStruct: "certificatesmanagement.CertificateAuthorityCollection",
					Exclude:   true,
					Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
				},
				"certificatesmanagement.CertificateAuthoritySummary": {
					SDKStruct:  "certificatesmanagement.CertificateAuthoritySummary",
					APISurface: "status",
				},
				"certificatesmanagement.CertificateAuthorityVersionSummary": {
					SDKStruct: "certificatesmanagement.CertificateAuthorityVersionSummary",
					Exclude:   true,
					Reason:    "Intentionally untracked: version summaries belong to the dedicated CertificatesManagementCertificateAuthorityVersion status surface.",
				},
			},
		},
		{
			name: "certificate authority version excludes collection",
			spec: "CertificateAuthorityVersion",
			candidates: []string{
				"CertificateAuthorityVersion",
				"CertificateAuthorityVersionCollection",
				"CertificateAuthorityVersionSummary",
			},
			want: map[string]sdkMapping{
				"certificatesmanagement.CertificateAuthorityVersion": {
					SDKStruct:  "certificatesmanagement.CertificateAuthorityVersion",
					APISurface: "status",
				},
				"certificatesmanagement.CertificateAuthorityVersionCollection": {
					SDKStruct: "certificatesmanagement.CertificateAuthorityVersionCollection",
					Exclude:   true,
					Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
				},
				"certificatesmanagement.CertificateAuthorityVersionSummary": {
					SDKStruct:  "certificatesmanagement.CertificateAuthorityVersionSummary",
					APISurface: "status",
				},
			},
		},
		{
			name: "certificate version excludes collection",
			spec: "CertificateVersion",
			candidates: []string{
				"CertificateVersion",
				"CertificateVersionCollection",
				"CertificateVersionSummary",
			},
			want: map[string]sdkMapping{
				"certificatesmanagement.CertificateVersion": {
					SDKStruct:  "certificatesmanagement.CertificateVersion",
					APISurface: "status",
				},
				"certificatesmanagement.CertificateVersionCollection": {
					SDKStruct: "certificatesmanagement.CertificateVersionCollection",
					Exclude:   true,
					Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
				},
				"certificatesmanagement.CertificateVersionSummary": {
					SDKStruct:  "certificatesmanagement.CertificateVersionSummary",
					APISurface: "status",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildSDKMappings("certificatesmanagement", tt.spec, tt.candidates, false, specTarget{})
			assertExactMappings(t, got, tt.want)
		})
	}
}

func TestBuildSDKMappingsKeyManagementOverrides(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		spec       string
		candidates []string
		want       map[string]sdkMapping
	}{
		{
			name: "hsm cluster excludes collection",
			spec: "HsmCluster",
			candidates: []string{
				"HsmCluster",
				"HsmClusterCollection",
				"HsmClusterSummary",
			},
			want: map[string]sdkMapping{
				"keymanagement.HsmCluster": {
					SDKStruct:  "keymanagement.HsmCluster",
					APISurface: "status",
				},
				"keymanagement.HsmClusterCollection": {
					SDKStruct: "keymanagement.HsmClusterCollection",
					Exclude:   true,
					Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
				},
				"keymanagement.HsmClusterSummary": {
					SDKStruct:  "keymanagement.HsmClusterSummary",
					APISurface: "status",
				},
			},
		},
		{
			name: "hsm partition excludes collection",
			spec: "HsmPartition",
			candidates: []string{
				"HsmPartition",
				"HsmPartitionCollection",
				"HsmPartitionSummary",
			},
			want: map[string]sdkMapping{
				"keymanagement.HsmPartition": {
					SDKStruct:  "keymanagement.HsmPartition",
					APISurface: "status",
				},
				"keymanagement.HsmPartitionCollection": {
					SDKStruct: "keymanagement.HsmPartitionCollection",
					Exclude:   true,
					Reason:    "Intentionally untracked: collection responses do not map to a singular resource status surface.",
				},
				"keymanagement.HsmPartitionSummary": {
					SDKStruct:  "keymanagement.HsmPartitionSummary",
					APISurface: "status",
				},
			},
		},
		{
			name: "key excludes version summary",
			spec: "Key",
			candidates: []string{
				"Key",
				"KeySummary",
				"KeyVersionSummary",
			},
			want: map[string]sdkMapping{
				"keymanagement.Key": {
					SDKStruct:  "keymanagement.Key",
					APISurface: "status",
				},
				"keymanagement.KeySummary": {
					SDKStruct:  "keymanagement.KeySummary",
					APISurface: "status",
				},
				"keymanagement.KeyVersionSummary": {
					SDKStruct: "keymanagement.KeyVersionSummary",
					Exclude:   true,
					Reason:    "Intentionally untracked: key version summaries belong to the dedicated KeyManagementKeyVersion status surface.",
				},
			},
		},
		{
			name: "replication status is intentionally untracked",
			spec: "ReplicationStatus",
			candidates: []string{
				"ReplicationStatusDetails",
			},
			want: map[string]sdkMapping{
				"keymanagement.ReplicationStatusDetails": {
					SDKStruct: "keymanagement.ReplicationStatusDetails",
					Exclude:   true,
					Reason:    "Intentionally untracked: OCI read-model mappings broaden desired-state coverage, and this CRD does not expose a meaningful status surface for parity tracking.",
				},
			},
		},
		{
			name: "vault replica excludes nested details",
			spec: "VaultReplica",
			candidates: []string{
				"VaultReplicaDetails",
				"VaultReplicaSummary",
			},
			want: map[string]sdkMapping{
				"keymanagement.VaultReplicaDetails": {
					SDKStruct: "keymanagement.VaultReplicaDetails",
					Exclude:   true,
					Reason:    "Intentionally untracked: replica detail payload is nested under KeyManagementVault status via replicaDetails.",
				},
				"keymanagement.VaultReplicaSummary": {
					SDKStruct:  "keymanagement.VaultReplicaSummary",
					APISurface: "status",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildSDKMappings("keymanagement", tt.spec, tt.candidates, false, specTarget{})
			assertExactMappings(t, got, tt.want)
		})
	}
}

func assertExactMappings(t *testing.T, got []sdkMapping, want map[string]sdkMapping) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}

	expected := make(map[string]sdkMapping, len(want))
	for sdkStruct, mapping := range want {
		expected[sdkStruct] = mapping
	}

	for _, mapping := range got {
		wantMapping, ok := expected[mapping.SDKStruct]
		if !ok {
			t.Fatalf("unexpected mapping %#v", mapping)
		}
		if mapping != wantMapping {
			t.Fatalf("mapping for %q = %#v, want %#v", mapping.SDKStruct, mapping, wantMapping)
		}
		delete(expected, mapping.SDKStruct)
	}

	if len(expected) != 0 {
		t.Fatalf("missing mappings: %#v", expected)
	}
}
