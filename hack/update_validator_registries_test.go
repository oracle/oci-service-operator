package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
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

	apiOut, sdkOut, err := generateRegistryOutputs(root, existingAPI, existingSDK)
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
