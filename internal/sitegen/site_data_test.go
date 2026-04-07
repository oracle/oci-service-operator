/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sitegen_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/oracle/oci-service-operator/internal/sitegen"
)

func TestCatalogMatchesCheckedInPackageDirectories(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	catalog, err := sitegen.LoadCatalog(filepath.Join(root, "docs", "site-data", "catalog.yaml"))
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}

	metadataByPackage, err := sitegen.LoadPackageMetadataDir(filepath.Join(root, "packages"))
	if err != nil {
		t.Fatalf("LoadPackageMetadataDir() error = %v", err)
	}

	got := make([]string, 0, len(catalog.Packages))
	visible := make([]string, 0, len(catalog.Packages))
	for _, pkg := range catalog.Packages {
		got = append(got, pkg.Package)
		if _, err := os.Stat(filepath.Join(root, pkg.GuidePath)); err != nil {
			t.Fatalf("guidePath %q for package %q: %v", pkg.GuidePath, pkg.Package, err)
		}
		if pkg.CustomerVisible {
			visible = append(visible, pkg.Package)
		}
	}

	want := make([]string, 0, len(metadataByPackage))
	for pkg := range metadataByPackage {
		want = append(want, pkg)
	}
	sort.Strings(want)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog packages = %v, want %v", got, want)
	}

	if !contains(visible, "core") {
		t.Fatalf("catalog omitted core from customerVisible packages")
	}
}

func TestKindOverridesReferToCheckedInKinds(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	catalog, err := sitegen.LoadCatalog(filepath.Join(root, "docs", "site-data", "catalog.yaml"))
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}

	apiKinds, err := loadAPIKinds(filepath.Join(root, "api"))
	if err != nil {
		t.Fatalf("loadAPIKinds() error = %v", err)
	}

	for _, pkg := range catalog.Packages {
		for _, override := range pkg.KindOverrides {
			kindsByVersion, ok := apiKinds[override.Group]
			if !ok {
				t.Fatalf("package %q kind override references unknown group %q", pkg.Package, override.Group)
			}
			if !containsAnyVersion(kindsByVersion, override.Kind) {
				t.Fatalf("package %q kind override references unknown kind %q for group %q", pkg.Package, override.Kind, override.Group)
			}
		}
	}
}

func TestReleaseManifestsMatchCheckedInPackageScope(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	releases, err := sitegen.LoadReleaseManifests(filepath.Join(root, "docs", "site-data", "releases"))
	if err != nil {
		t.Fatalf("LoadReleaseManifests() error = %v", err)
	}
	catalog, err := sitegen.LoadCatalog(filepath.Join(root, "docs", "site-data", "catalog.yaml"))
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	metadataByPackage, err := sitegen.LoadPackageMetadataDir(filepath.Join(root, "packages"))
	if err != nil {
		t.Fatalf("LoadPackageMetadataDir() error = %v", err)
	}
	apiKinds, err := loadAPIKinds(filepath.Join(root, "api"))
	if err != nil {
		t.Fatalf("loadAPIKinds() error = %v", err)
	}

	catalogByPackage := make(map[string]sitegen.CatalogPackage, len(catalog.Packages))
	for _, pkg := range catalog.Packages {
		catalogByPackage[pkg.Package] = pkg
	}

	releasedPackages := make(map[string]struct{})
	for _, release := range releases {
		for _, pkg := range release.Packages {
			releasedPackages[pkg.Package] = struct{}{}

			metadata, ok := metadataByPackage[pkg.Package]
			if !ok {
				t.Fatalf("release %q references package %q without checked-in metadata", release.Version, pkg.Package)
			}
			catalogEntry, ok := catalogByPackage[pkg.Package]
			if !ok {
				t.Fatalf("release %q references package %q without catalog metadata", release.Version, pkg.Package)
			}
			if !catalogEntry.CustomerVisible {
				t.Fatalf("release %q references customer-hidden package %q", release.Version, pkg.Package)
			}

			wantControllerImage := "ghcr.io/<REPOSITORY_OWNER>/" + metadata.PackageName + ":" + release.Version
			if pkg.ControllerImage != wantControllerImage {
				t.Fatalf("release %q package %q controllerImage = %q, want %q", release.Version, pkg.Package, pkg.ControllerImage, wantControllerImage)
			}

			wantBundleImage := "ghcr.io/<REPOSITORY_OWNER>/" + metadata.PackageName + "-bundle:" + release.Version
			if pkg.BundleImage != wantBundleImage {
				t.Fatalf("release %q package %q bundleImage = %q, want %q", release.Version, pkg.Package, pkg.BundleImage, wantBundleImage)
			}

			wantGroups, err := expectedReleaseGroups(pkg.Package, metadata, apiKinds)
			if err != nil {
				t.Fatalf("expectedReleaseGroups(%q) error = %v", pkg.Package, err)
			}
			if !reflect.DeepEqual(pkg.Groups, wantGroups) {
				t.Fatalf("release %q package %q groups = %#v, want %#v", release.Version, pkg.Package, pkg.Groups, wantGroups)
			}
		}
	}

	for _, pkg := range catalog.Packages {
		if !pkg.CustomerVisible {
			if _, ok := releasedPackages[pkg.Package]; ok {
				t.Fatalf("customer-hidden package %q unexpectedly appears in release history", pkg.Package)
			}
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %q", dir)
		}
		dir = parent
	}
}

func loadAPIKinds(root string) (map[string]map[string][]string, error) {
	groupEntries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	out := make(map[string]map[string][]string)
	for _, groupEntry := range groupEntries {
		if !groupEntry.IsDir() {
			continue
		}
		groupPath := filepath.Join(root, groupEntry.Name())
		versionEntries, err := os.ReadDir(groupPath)
		if err != nil {
			return nil, err
		}

		out[groupEntry.Name()] = make(map[string][]string)
		for _, versionEntry := range versionEntries {
			if !versionEntry.IsDir() {
				continue
			}
			kinds, err := loadKindsForVersion(filepath.Join(groupPath, versionEntry.Name()))
			if err != nil {
				return nil, err
			}
			out[groupEntry.Name()][versionEntry.Name()] = kinds
		}
	}

	return out, nil
}

func loadKindsForVersion(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	var kinds []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_types.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE || genDecl.Doc == nil {
				continue
			}
			if !strings.Contains(genDecl.Doc.Text(), "+kubebuilder:object:root=true") {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				if hasItemsField(structType) {
					continue
				}
				kinds = append(kinds, typeSpec.Name.Name)
			}
		}
	}

	sort.Strings(kinds)
	return kinds, nil
}

func expectedReleaseGroups(
	packageName string,
	metadata *sitegen.PackageMetadata,
	apiKinds map[string]map[string][]string,
) ([]sitegen.ReleaseGroup, error) {
	group := packageGroup(metadata.CRDPaths)
	versions, ok := apiKinds[group]
	if !ok {
		return nil, os.ErrNotExist
	}
	if len(versions) != 1 {
		return nil, os.ErrInvalid
	}

	var version string
	for candidate := range versions {
		version = candidate
	}

	kinds := versions[version]
	if metadata.CRDKindFilter != "" {
		kinds = splitKinds(metadata.CRDKindFilter)
	}

	return []sitegen.ReleaseGroup{
		{
			Group:   group,
			Version: version,
			Kinds:   kinds,
		},
	}, nil
}

func packageGroup(crdPaths string) string {
	trimmed := strings.TrimPrefix(strings.TrimSpace(crdPaths), "./api/")
	before, _, _ := strings.Cut(trimmed, "/")
	return before
}

func splitKinds(csv string) []string {
	parts := strings.Split(csv, ",")
	kinds := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kinds = append(kinds, part)
	}
	return kinds
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsAnyVersion(kindsByVersion map[string][]string, want string) bool {
	for _, kinds := range kindsByVersion {
		if contains(kinds, want) {
			return true
		}
	}
	return false
}

func hasItemsField(structType *ast.StructType) bool {
	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			if name.Name == "Items" {
				return true
			}
		}
	}
	return false
}
