package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type checkedInFormalImport struct {
	Service          string `json:"service"`
	Slug             string `json:"slug"`
	Kind             string `json:"kind"`
	SourceRef        string `json:"sourceRef"`
	ProviderResource string `json:"providerResource"`
}

// TestCheckedInMutabilityProvenanceMatchesFormalImports prevents checked-in
// generator outputs from retaining a provider identity that no longer matches
// the repo-authored formal provider facts.
func TestCheckedInMutabilityProvenanceMatchesFormalImports(t *testing.T) {
	root := repoRoot(t)
	overlayPaths, err := filepath.Glob(filepath.Join(
		root,
		"internal", "generator", "generated", "mutability_overlay", "*", "*.json",
	))
	if err != nil {
		t.Fatalf("glob checked-in mutability overlays: %v", err)
	}
	if len(overlayPaths) == 0 {
		t.Fatal("no checked-in mutability overlays found")
	}

	for _, overlayPath := range overlayPaths {
		service := filepath.Base(filepath.Dir(overlayPath))
		slug := strings.TrimSuffix(filepath.Base(overlayPath), filepath.Ext(overlayPath))
		t.Run(service+"/"+slug, func(t *testing.T) {
			formalPath := filepath.Join(root, "formal", "imports", service, slug+".json")
			formalImport := readCheckedInJSON[checkedInFormalImport](t, formalPath)
			overlay := readCheckedInJSON[mutabilityOverlayDocument](t, overlayPath)
			vapPath := filepath.Join(
				root,
				"internal", "generator", "generated", "vap_update_policy", service, slug+".json",
			)
			vap := readCheckedInJSON[vapUpdatePolicyDocument](t, vapPath)

			assertCheckedInIdentity(t, "formal service", formalImport.Service, service)
			assertCheckedInIdentity(t, "formal slug", formalImport.Slug, slug)
			assertCheckedInIdentity(t, "overlay service", overlay.Resource.Service, service)
			assertCheckedInIdentity(t, "overlay kind", overlay.Resource.Kind, formalImport.Kind)
			assertCheckedInIdentity(t, "overlay formal slug", overlay.Resource.FormalSlug, slug)
			assertCheckedInIdentity(
				t,
				"overlay formal import path",
				overlay.Resource.FormalImportPath,
				filepath.ToSlash(filepath.Join("imports", service, slug+".json")),
			)
			assertCheckedInIdentity(
				t,
				"overlay provider source",
				overlay.Metadata.ProviderSourceRef,
				formalImport.SourceRef,
			)
			assertCheckedInIdentity(
				t,
				"overlay provider resource",
				overlay.Resource.ProviderResource,
				formalImport.ProviderResource,
			)

			assertCheckedInIdentity(t, "VAP service", vap.Target.Service, service)
			assertCheckedInIdentity(t, "VAP kind", vap.Target.Kind, formalImport.Kind)
			assertCheckedInIdentity(t, "VAP formal slug", vap.Target.FormalSlug, slug)
			assertCheckedInIdentity(
				t,
				"VAP provider source",
				vap.Metadata.ProviderSourceRef,
				formalImport.SourceRef,
			)
			assertCheckedInIdentity(
				t,
				"VAP provider resource",
				vap.Target.ProviderResource,
				formalImport.ProviderResource,
			)

			pageName, err := mutabilityOverlayRegistryPageName(formalImport.ProviderResource)
			if err != nil {
				t.Fatalf("derive Terraform docs page from formal provider resource: %v", err)
			}
			for index, field := range overlay.Fields {
				gotPage, _, _ := strings.Cut(field.Provenance.TerraformDocsPage, "#")
				assertCheckedInIdentity(
					t,
					"overlay field "+field.ASTFieldPath+" Terraform docs page",
					gotPage,
					pageName,
				)
				if strings.TrimSpace(field.ASTFieldPath) == "" {
					t.Errorf("overlay field %d has an empty AST field path", index)
				}
			}

			metadataPattern := filepath.Join(
				root,
				"internal", "generator", "testdata", "mutability_overlay", "docs",
				"*", service, slug, "metadata.json",
			)
			metadataPaths, err := filepath.Glob(metadataPattern)
			if err != nil {
				t.Fatalf("glob checked-in docs metadata: %v", err)
			}
			for _, metadataPath := range metadataPaths {
				metadata := readCheckedInJSON[mutabilityOverlayDocsInputMetadata](t, metadataPath)
				assertCheckedInIdentity(t, "docs service", metadata.Service, service)
				assertCheckedInIdentity(t, "docs kind", metadata.Kind, formalImport.Kind)
				assertCheckedInIdentity(t, "docs formal slug", metadata.FormalSlug, slug)
				assertCheckedInIdentity(
					t,
					"docs provider resource",
					metadata.ProviderResource,
					formalImport.ProviderResource,
				)

				contract := mutabilityOverlayDocsContract{
					TerraformDocsVersion: metadata.TerraformDocsVersion,
					RegistryHost:         mutabilityOverlayTerraformRegistryHost,
					ProviderNamespace:    mutabilityOverlayTerraformRegistryNamespace,
					ProviderName:         mutabilityOverlayTerraformRegistryProvider,
				}
				wantPath, wantURL, err := contract.versionedResourcePage(formalImport.ProviderResource)
				if err != nil {
					t.Fatalf("derive versioned Terraform docs page: %v", err)
				}
				assertCheckedInIdentity(t, "docs registry path", metadata.RegistryPath, wantPath)
				assertCheckedInIdentity(t, "docs registry URL", metadata.RegistryURL, wantURL)
			}
		})
	}
}

func readCheckedInJSON[T any](t *testing.T, path string) T {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var value T
	if err := json.Unmarshal(content, &value); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return value
}

func assertCheckedInIdentity(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}
