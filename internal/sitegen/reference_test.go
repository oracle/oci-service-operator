/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sitegen_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/oracle/oci-service-operator/internal/sitegen"
)

func TestBuildReferenceSiteUsesReleaseAndPackageFallbackData(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	site, err := sitegen.BuildReferenceSite(root)
	if err != nil {
		t.Fatalf("BuildReferenceSite() error = %v", err)
	}

	identity := findPackage(t, site.Packages, "identity")
	if !identity.CustomerVisible {
		t.Fatalf("identity customerVisible = false, want true")
	}
	if identity.LatestReleaseVersion != "v2.0.0-alpha" {
		t.Fatalf("identity latest release = %q, want %q", identity.LatestReleaseVersion, "v2.0.0-alpha")
	}
	if len(identity.Resources) != 1 {
		t.Fatalf("identity resources = %d, want 1", len(identity.Resources))
	}
	if got := identity.Resources[0].SampleSourcePath; got != "config/samples/identity_v1beta1_compartment.yaml" {
		t.Fatalf("identity sample path = %q, want %q", got, "config/samples/identity_v1beta1_compartment.yaml")
	}

	core := findPackage(t, site.Packages, "core")
	if !core.CustomerVisible {
		t.Fatalf("core customerVisible = false, want true")
	}
	if core.LatestReleaseVersion != "" {
		t.Fatalf("core latest release = %q, want empty", core.LatestReleaseVersion)
	}
	if len(core.Resources) != 1 {
		t.Fatalf("core resources = %d, want 1", len(core.Resources))
	}
	if !hasKind(core.Resources, "Instance") {
		t.Fatalf("core resources do not include Instance")
	}
	if hasKind(core.Resources, "SecurityList") {
		t.Fatalf("core resources unexpectedly include SecurityList")
	}

	findPackage(t, site.PublicPackages, "core")
}

func TestGenerateReferenceDocsMatchesCheckedInOutputs(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	tempDir := t.TempDir()

	result, err := sitegen.GenerateReferenceDocs(sitegen.GenerateOptions{
		Root:       root,
		OutputRoot: tempDir,
	})
	if err != nil {
		t.Fatalf("GenerateReferenceDocs() error = %v", err)
	}
	if len(result.Written) == 0 {
		t.Fatalf("GenerateReferenceDocs() wrote no files")
	}

	expected := append([]string{}, result.Written...)
	sort.Strings(expected)
	checkedIn, err := checkedInGeneratedPaths(root)
	if err != nil {
		t.Fatalf("checkedInGeneratedPaths() error = %v", err)
	}
	sort.Strings(checkedIn)
	if strings.Join(expected, "\n") != strings.Join(checkedIn, "\n") {
		t.Fatalf("generated file set = %v, want %v", expected, checkedIn)
	}

	for _, relPath := range result.Written {
		got, err := os.ReadFile(filepath.Join(tempDir, filepath.FromSlash(relPath)))
		if err != nil {
			t.Fatalf("read generated file %q: %v", relPath, err)
		}
		want, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
		if err != nil {
			t.Fatalf("read checked-in file %q: %v", relPath, err)
		}
		if string(got) != string(want) {
			t.Fatalf("%s does not match checked-in output", relPath)
		}
	}
}

func checkedInGeneratedPaths(root string) ([]string, error) {
	var paths []string

	referenceRoot := filepath.Join(root, "docs", "reference")
	if err := filepath.WalkDir(referenceRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		return nil, err
	}
	paths = append(paths, "mkdocs.yml")

	return paths, nil
}

func findPackage(t *testing.T, packages []sitegen.ReferencePackage, name string) sitegen.ReferencePackage {
	t.Helper()

	for _, pkg := range packages {
		if pkg.Package == name {
			return pkg
		}
	}
	t.Fatalf("package %q not found", name)
	return sitegen.ReferencePackage{}
}

func hasKind(resources []sitegen.ReferenceResource, want string) bool {
	for _, resource := range resources {
		if resource.Kind == want {
			return true
		}
	}
	return false
}
