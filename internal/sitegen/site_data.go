/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sitegen

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const SchemaVersionV1Alpha1 = "v1alpha1"

// Catalog is the hand-maintained customer-facing metadata consumed by site generation.
type Catalog struct {
	SchemaVersion string           `yaml:"schemaVersion"`
	Packages      []CatalogPackage `yaml:"packages"`
}

// CatalogPackage stores customer-facing package labels and optional per-kind copy overrides.
type CatalogPackage struct {
	Package         string         `yaml:"package"`
	DisplayName     string         `yaml:"displayName"`
	Summary         string         `yaml:"summary"`
	SupportStatus   string         `yaml:"supportStatus"`
	CustomerVisible bool           `yaml:"customerVisible"`
	GuidePath       string         `yaml:"guidePath"`
	PackageNotes    []string       `yaml:"packageNotes,omitempty"`
	KindOverrides   []KindOverride `yaml:"kindOverrides,omitempty"`
}

// KindOverride stores package-scoped copy overrides for one published kind.
type KindOverride struct {
	Group       string `yaml:"group"`
	Kind        string `yaml:"kind"`
	Summary     string `yaml:"summary,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// ReleaseManifest records one checked-in package release history entry.
type ReleaseManifest struct {
	SchemaVersion string           `yaml:"schemaVersion"`
	Version       string           `yaml:"version"`
	PublishedAt   time.Time        `yaml:"publishedAt"`
	Commit        string           `yaml:"commit"`
	Notes         []string         `yaml:"notes,omitempty"`
	Packages      []ReleasePackage `yaml:"packages"`
}

// ReleasePackage records one package released for a specific version.
type ReleasePackage struct {
	Package         string         `yaml:"package"`
	ControllerImage string         `yaml:"controllerImage"`
	BundleImage     string         `yaml:"bundleImage"`
	Groups          []ReleaseGroup `yaml:"groups"`
}

// ReleaseGroup records one group/version surface exposed by a released package.
type ReleaseGroup struct {
	Group   string   `yaml:"group"`
	Version string   `yaml:"version"`
	Kinds   []string `yaml:"kinds"`
}

// PackageMetadata is the checked-in package identity data consumed by packaging and docs generation.
type PackageMetadata struct {
	PackageName            string
	PackageNamespace       string
	PackageNamePrefix      string
	CRDPaths               string
	CRDKindFilter          string
	RBACPaths              string
	DefaultControllerImage string
}

// LoadCatalog reads and validates the checked-in site metadata catalog.
func LoadCatalog(path string) (*Catalog, error) {
	var catalog Catalog
	if err := decodeYAMLFile(path, &catalog); err != nil {
		return nil, fmt.Errorf("load catalog %q: %w", path, err)
	}
	if err := catalog.Validate(); err != nil {
		return nil, fmt.Errorf("validate catalog %q: %w", path, err)
	}
	return &catalog, nil
}

// LoadReleaseManifest reads and validates one checked-in release manifest.
func LoadReleaseManifest(path string) (*ReleaseManifest, error) {
	var manifest ReleaseManifest
	if err := decodeYAMLFile(path, &manifest); err != nil {
		return nil, fmt.Errorf("load release manifest %q: %w", path, err)
	}
	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("validate release manifest %q: %w", path, err)
	}
	return &manifest, nil
}

// LoadReleaseManifests reads all checked-in release manifests in deterministic filename order.
func LoadReleaseManifests(dir string) ([]*ReleaseManifest, error) {
	pattern := filepath.Join(dir, "*.yaml")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob release manifests %q: %w", pattern, err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("no release manifests found in %q", dir)
	}

	manifests := make([]*ReleaseManifest, 0, len(paths))
	versions := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		manifest, err := LoadReleaseManifest(path)
		if err != nil {
			return nil, err
		}
		if _, exists := versions[manifest.Version]; exists {
			return nil, fmt.Errorf("duplicate release version %q in %q", manifest.Version, path)
		}
		versions[manifest.Version] = struct{}{}
		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

// LoadPackageMetadata reads one checked-in packages/<group>/metadata.env file.
func LoadPackageMetadata(path string) (*PackageMetadata, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read package metadata %q: %w", path, err)
	}

	fields := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("parse package metadata %q: malformed line %q", path, line)
		}
		fields[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan package metadata %q: %w", path, err)
	}

	metadata := &PackageMetadata{
		PackageName:            fields["PACKAGE_NAME"],
		PackageNamespace:       fields["PACKAGE_NAMESPACE"],
		PackageNamePrefix:      fields["PACKAGE_NAME_PREFIX"],
		CRDPaths:               fields["CRD_PATHS"],
		CRDKindFilter:          fields["CRD_KIND_FILTER"],
		RBACPaths:              fields["RBAC_PATHS"],
		DefaultControllerImage: fields["DEFAULT_CONTROLLER_IMAGE"],
	}
	if err := metadata.Validate(); err != nil {
		return nil, fmt.Errorf("validate package metadata %q: %w", path, err)
	}

	return metadata, nil
}

// LoadPackageMetadataDir reads all checked-in package metadata files keyed by package directory name.
func LoadPackageMetadataDir(dir string) (map[string]*PackageMetadata, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read package metadata directory %q: %w", dir, err)
	}

	packages := make(map[string]*PackageMetadata)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metadataPath := filepath.Join(dir, entry.Name(), "metadata.env")
		if _, err := os.Stat(metadataPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat package metadata %q: %w", metadataPath, err)
		}
		metadata, err := LoadPackageMetadata(metadataPath)
		if err != nil {
			return nil, err
		}
		packages[entry.Name()] = metadata
	}

	if len(packages) == 0 {
		return nil, fmt.Errorf("no package metadata files found in %q", dir)
	}

	return packages, nil
}

// Validate ensures the catalog is internally consistent.
func (c *Catalog) Validate() error {
	if c == nil {
		return fmt.Errorf("catalog is required")
	}
	if c.SchemaVersion != SchemaVersionV1Alpha1 {
		return fmt.Errorf("schemaVersion must be %q", SchemaVersionV1Alpha1)
	}
	if len(c.Packages) == 0 {
		return fmt.Errorf("at least one catalog package is required")
	}

	seen := make(map[string]struct{}, len(c.Packages))
	var previous string
	for _, pkg := range c.Packages {
		if err := pkg.Validate(); err != nil {
			return err
		}
		if previous != "" && pkg.Package < previous {
			return fmt.Errorf("catalog packages must be sorted by package name")
		}
		previous = pkg.Package
		if _, exists := seen[pkg.Package]; exists {
			return fmt.Errorf("duplicate catalog package %q", pkg.Package)
		}
		seen[pkg.Package] = struct{}{}
	}

	return nil
}

// Validate ensures the release manifest is internally consistent.
func (m *ReleaseManifest) Validate() error {
	if m == nil {
		return fmt.Errorf("release manifest is required")
	}
	if m.SchemaVersion != SchemaVersionV1Alpha1 {
		return fmt.Errorf("schemaVersion must be %q", SchemaVersionV1Alpha1)
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("version is required")
	}
	if m.PublishedAt.IsZero() {
		return fmt.Errorf("publishedAt is required")
	}
	if strings.TrimSpace(m.Commit) == "" {
		return fmt.Errorf("commit is required")
	}
	if len(m.Packages) == 0 {
		return fmt.Errorf("at least one released package is required")
	}
	for _, note := range m.Notes {
		if strings.TrimSpace(note) == "" {
			return fmt.Errorf("release notes must not be blank")
		}
	}

	seen := make(map[string]struct{}, len(m.Packages))
	var previous string
	for _, pkg := range m.Packages {
		if err := pkg.Validate(); err != nil {
			return err
		}
		if previous != "" && pkg.Package < previous {
			return fmt.Errorf("release packages must be sorted by package name")
		}
		previous = pkg.Package
		if _, exists := seen[pkg.Package]; exists {
			return fmt.Errorf("duplicate released package %q", pkg.Package)
		}
		seen[pkg.Package] = struct{}{}
	}

	return nil
}

// Validate ensures the checked-in package metadata includes the required fields.
func (m *PackageMetadata) Validate() error {
	if m == nil {
		return fmt.Errorf("package metadata is required")
	}
	if strings.TrimSpace(m.PackageName) == "" {
		return fmt.Errorf("PACKAGE_NAME is required")
	}
	if strings.TrimSpace(m.PackageNamespace) == "" {
		return fmt.Errorf("PACKAGE_NAMESPACE is required")
	}
	if strings.TrimSpace(m.PackageNamePrefix) == "" {
		return fmt.Errorf("PACKAGE_NAME_PREFIX is required")
	}
	if strings.TrimSpace(m.CRDPaths) == "" {
		return fmt.Errorf("CRD_PATHS is required")
	}
	if strings.TrimSpace(m.DefaultControllerImage) == "" {
		return fmt.Errorf("DEFAULT_CONTROLLER_IMAGE is required")
	}
	return nil
}

// Validate ensures a catalog package has the required customer-facing fields.
func (p *CatalogPackage) Validate() error {
	if strings.TrimSpace(p.Package) == "" {
		return fmt.Errorf("catalog package name is required")
	}
	if strings.TrimSpace(p.DisplayName) == "" {
		return fmt.Errorf("catalog package %q displayName is required", p.Package)
	}
	if strings.TrimSpace(p.Summary) == "" {
		return fmt.Errorf("catalog package %q summary is required", p.Package)
	}
	if strings.TrimSpace(p.SupportStatus) == "" {
		return fmt.Errorf("catalog package %q supportStatus is required", p.Package)
	}
	if strings.TrimSpace(p.GuidePath) == "" {
		return fmt.Errorf("catalog package %q guidePath is required", p.Package)
	}
	for _, note := range p.PackageNotes {
		if strings.TrimSpace(note) == "" {
			return fmt.Errorf("catalog package %q has a blank package note", p.Package)
		}
	}

	seen := make(map[string]struct{}, len(p.KindOverrides))
	var previous string
	for _, override := range p.KindOverrides {
		if err := override.Validate(); err != nil {
			return fmt.Errorf("catalog package %q: %w", p.Package, err)
		}
		key := override.Group + "/" + override.Kind
		if previous != "" && key < previous {
			return fmt.Errorf("catalog package %q kindOverrides must be sorted by group/kind", p.Package)
		}
		previous = key
		if _, exists := seen[key]; exists {
			return fmt.Errorf("catalog package %q has duplicate kind override %q", p.Package, key)
		}
		seen[key] = struct{}{}
	}

	return nil
}

// Validate ensures a kind override references a specific kind and includes copy.
func (o *KindOverride) Validate() error {
	if strings.TrimSpace(o.Group) == "" {
		return fmt.Errorf("kind override group is required")
	}
	if strings.TrimSpace(o.Kind) == "" {
		return fmt.Errorf("kind override kind is required")
	}
	if strings.TrimSpace(o.Summary) == "" && strings.TrimSpace(o.Description) == "" {
		return fmt.Errorf("kind override %s/%s requires summary or description", o.Group, o.Kind)
	}
	return nil
}

// Validate ensures a released package has the required image and group data.
func (p *ReleasePackage) Validate() error {
	if strings.TrimSpace(p.Package) == "" {
		return fmt.Errorf("released package name is required")
	}
	if strings.TrimSpace(p.ControllerImage) == "" {
		return fmt.Errorf("released package %q controllerImage is required", p.Package)
	}
	if strings.TrimSpace(p.BundleImage) == "" {
		return fmt.Errorf("released package %q bundleImage is required", p.Package)
	}
	if len(p.Groups) == 0 {
		return fmt.Errorf("released package %q requires at least one group", p.Package)
	}

	seen := make(map[string]struct{}, len(p.Groups))
	var previous string
	for _, group := range p.Groups {
		if err := group.Validate(); err != nil {
			return fmt.Errorf("released package %q: %w", p.Package, err)
		}
		key := group.Group + "/" + group.Version
		if previous != "" && key < previous {
			return fmt.Errorf("released package %q groups must be sorted by group/version", p.Package)
		}
		previous = key
		if _, exists := seen[key]; exists {
			return fmt.Errorf("released package %q has duplicate group %q", p.Package, key)
		}
		seen[key] = struct{}{}
	}

	return nil
}

// Validate ensures a released group includes a concrete version and kind list.
func (g *ReleaseGroup) Validate() error {
	if strings.TrimSpace(g.Group) == "" {
		return fmt.Errorf("released group name is required")
	}
	if strings.TrimSpace(g.Version) == "" {
		return fmt.Errorf("released group %q version is required", g.Group)
	}
	if len(g.Kinds) == 0 {
		return fmt.Errorf("released group %q requires at least one kind", g.Group)
	}

	seen := make(map[string]struct{}, len(g.Kinds))
	var previous string
	for _, kind := range g.Kinds {
		if strings.TrimSpace(kind) == "" {
			return fmt.Errorf("released group %q has a blank kind", g.Group)
		}
		if previous != "" && kind < previous {
			return fmt.Errorf("released group %q kinds must be sorted", g.Group)
		}
		previous = kind
		if _, exists := seen[kind]; exists {
			return fmt.Errorf("released group %q has duplicate kind %q", g.Group, kind)
		}
		seen[kind] = struct{}{}
	}

	return nil
}

func decodeYAMLFile(path string, out any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read yaml file %q: %w", path, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode yaml file %q: %w", path, err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return fmt.Errorf("decode yaml file %q: %w", path, err)
		}
		return fmt.Errorf("yaml file %q contains more than one document", path)
	}

	return nil
}
