/*
 Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
 Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package formalscaffold

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/oracle/oci-service-operator/internal/formal"
	"github.com/oracle/oci-service-operator/internal/generator"
)

const manifestHeader = "service\tslug\tkind\tstage\tsurface\timport\tspec\tlogic_gaps\tdiagram_dir\n"

// Options controls formal scaffold expansion.
type Options struct {
	Root         string
	ConfigPath   string
	ProviderPath string
}

// Report summarizes scaffold generation.
type Report struct {
	Root                  string
	ConfigPath            string
	ServicesScanned       int
	PublishedKinds        int
	ProviderKinds         int
	NewRows               int
	ExistingRowsPreserved int
	ManifestRows          int
	FilesWritten          int
}

// String renders the scaffold report.
func (r Report) String() string {
	var b strings.Builder
	b.WriteString("formal scaffold completed\n")
	fmt.Fprintf(&b, "- root: %s\n", r.Root)
	fmt.Fprintf(&b, "- config: %s\n", r.ConfigPath)
	fmt.Fprintf(&b, "- services scanned: %d\n", r.ServicesScanned)
	fmt.Fprintf(&b, "- published kinds discovered: %d\n", r.PublishedKinds)
	if r.ProviderKinds > 0 {
		fmt.Fprintf(&b, "- provider kinds discovered: %d\n", r.ProviderKinds)
	}
	fmt.Fprintf(&b, "- existing rows preserved: %d\n", r.ExistingRowsPreserved)
	fmt.Fprintf(&b, "- new scaffold rows: %d\n", r.NewRows)
	fmt.Fprintf(&b, "- manifest rows: %d\n", r.ManifestRows)
	fmt.Fprintf(&b, "- files written: %d\n", r.FilesWritten)
	return b.String()
}

type inventoryEntry struct {
	Service          string
	Group            string
	Version          string
	Slug             string
	Kind             string
	ProviderResource string
}

type scaffoldArtifacts struct {
	Spec    []byte
	Logic   []byte
	Diagram []byte
	Import  []byte
}

type generateInputs struct {
	Root       string
	ConfigPath string
	Config     *generator.Config
	Catalog    *formal.Catalog
	Template   formal.ControllerBinding
}

// Generate expands the repo-local formal scaffold to cover every published API kind.
func Generate(opts Options) (Report, error) {
	report, err := initializeGenerateReport(opts)
	if err != nil {
		return report, err
	}

	inputs, err := loadGenerateInputs(report)
	if err != nil {
		return report, err
	}
	report.Root = inputs.Root
	report.ConfigPath = inputs.ConfigPath

	entries, discoveredKeys, err := loadGenerateInventory(inputs, strings.TrimSpace(opts.ProviderPath), &report)
	if err != nil {
		return report, err
	}

	rows, err := buildManifestRows(inputs.Root, inputs.Config, inputs.Catalog, entries, &report)
	if err != nil {
		return report, err
	}
	report.ManifestRows = len(rows)

	if err := writeManifestAndPrune(inputs.Root, rows, &report); err != nil {
		return report, err
	}
	if err := writeDiscoveredScaffolds(inputs.Root, rows, discoveredKeys, inputs.Template, &report); err != nil {
		return report, err
	}
	if err := renderAndVerifyScaffold(inputs, strings.TrimSpace(opts.ProviderPath), &report); err != nil {
		return report, err
	}
	return report, nil
}

func initializeGenerateReport(opts Options) (Report, error) {
	report := Report{
		Root:       filepath.Clean(strings.TrimSpace(opts.Root)),
		ConfigPath: filepath.Clean(strings.TrimSpace(opts.ConfigPath)),
	}
	if report.Root == "" {
		return report, fmt.Errorf("formal root must not be empty")
	}
	if report.ConfigPath == "" {
		return report, fmt.Errorf("generator config path must not be empty")
	}
	return report, nil
}

func loadGenerateInputs(report Report) (generateInputs, error) {
	root, err := filepath.Abs(report.Root)
	if err != nil {
		return generateInputs{}, fmt.Errorf("resolve formal root %q: %w", report.Root, err)
	}
	configPath, err := filepath.Abs(report.ConfigPath)
	if err != nil {
		return generateInputs{}, fmt.Errorf("resolve generator config %q: %w", report.ConfigPath, err)
	}
	if err := requireDirectory(root); err != nil {
		return generateInputs{}, err
	}

	cfg, err := generator.LoadConfig(configPath)
	if err != nil {
		return generateInputs{}, err
	}
	catalog, err := formal.LoadCatalogUnchecked(root)
	if err != nil {
		return generateInputs{}, fmt.Errorf("load existing formal catalog %q: %w", root, err)
	}
	templateBinding, ok := catalog.Lookup("template", "template")
	if !ok {
		return generateInputs{}, fmt.Errorf("formal template binding template/template was not found in %s", filepath.ToSlash(filepath.Join(root, "controller_manifest.tsv")))
	}

	return generateInputs{
		Root:       root,
		ConfigPath: configPath,
		Config:     cfg,
		Catalog:    catalog,
		Template:   templateBinding,
	}, nil
}

func loadGenerateInventory(inputs generateInputs, providerPath string, report *Report) ([]inventoryEntry, map[string]inventoryEntry, error) {
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(inputs.ConfigPath), "..", "..", ".."))
	entries, selectedServices, err := discoverPublishedKinds(repoRoot, inputs.Config)
	if err != nil {
		return nil, nil, err
	}
	report.ServicesScanned = len(selectedServices)
	report.PublishedKinds = len(entries)

	if providerPath != "" {
		providerEntries, err := discoverProviderKinds(providerPath)
		if err != nil {
			return nil, nil, err
		}
		report.ProviderKinds = len(providerEntries)
		entries, err = mergeInventoryEntries(entries, providerEntries)
		if err != nil {
			return nil, nil, err
		}
	}

	return entries, indexInventoryEntries(entries), nil
}

func indexInventoryEntries(entries []inventoryEntry) map[string]inventoryEntry {
	index := make(map[string]inventoryEntry, len(entries))
	for _, entry := range entries {
		index[rowKey(entry.Service, entry.Slug)] = entry
	}
	return index
}

func buildManifestRows(formalRoot string, cfg *generator.Config, catalog *formal.Catalog, entries []inventoryEntry, report *Report) ([]formal.ManifestRow, error) {
	existingBindings, rowByKey := seedManifestRows(cfg, catalog)
	recreatedRows, err := addConfiguredFormalSpecRows(formalRoot, cfg, rowByKey)
	if err != nil {
		return nil, err
	}
	report.NewRows += recreatedRows

	for _, entry := range entries {
		if err := mergeDiscoveredManifestRow(entry, existingBindings, rowByKey, report); err != nil {
			return nil, err
		}
	}

	rows := make([]formal.ManifestRow, 0, len(rowByKey))
	for _, row := range rowByKey {
		rows = append(rows, row)
	}
	sortManifestRows(rows)
	return rows, nil
}

func seedManifestRows(cfg *generator.Config, catalog *formal.Catalog) (map[string]formal.ControllerBinding, map[string]formal.ManifestRow) {
	existingBindings := make(map[string]formal.ControllerBinding, len(catalog.Controllers))
	rowByKey := make(map[string]formal.ManifestRow, len(catalog.Controllers)+1)
	for _, binding := range catalog.Controllers {
		key := rowKey(binding.Manifest.Service, binding.Manifest.Slug)
		existingBindings[key] = binding
		if shouldPreserveManifestRow(cfg, binding.Manifest) {
			rowByKey[key] = binding.Manifest
		}
	}
	return existingBindings, rowByKey
}

func mergeDiscoveredManifestRow(entry inventoryEntry, existingBindings map[string]formal.ControllerBinding, rowByKey map[string]formal.ManifestRow, report *Report) error {
	key := rowKey(entry.Service, entry.Slug)
	if binding, ok := existingBindings[key]; ok {
		if err := ensureManifestKindMatch(entry, binding.Manifest.Kind); err != nil {
			return err
		}
		report.ExistingRowsPreserved++
		rowByKey[key] = binding.Manifest
		return nil
	}
	if row, ok := rowByKey[key]; ok {
		return ensureManifestKindMatch(entry, row.Kind)
	}

	rowByKey[key] = scaffoldManifestRow(entry)
	report.NewRows++
	return nil
}

func ensureManifestKindMatch(entry inventoryEntry, kind string) error {
	if kind == entry.Kind {
		return nil
	}
	return fmt.Errorf(
		"existing formal row %s/%s kind %q does not match published API kind %q",
		entry.Service,
		entry.Slug,
		kind,
		entry.Kind,
	)
}

func writeManifestAndPrune(root string, rows []formal.ManifestRow, report *Report) error {
	changed, err := writeFileIfChanged(filepath.Join(root, "controller_manifest.tsv"), renderManifest(rows))
	if err != nil {
		return err
	}
	if changed {
		report.FilesWritten++
	}
	return pruneStaleFormalArtifacts(root, rows)
}

func writeDiscoveredScaffolds(root string, rows []formal.ManifestRow, discoveredKeys map[string]inventoryEntry, template formal.ControllerBinding, report *Report) error {
	for _, row := range rows {
		writes, err := writeDiscoveredScaffoldRow(root, row, discoveredKeys, template)
		if err != nil {
			return err
		}
		report.FilesWritten += writes
	}
	return nil
}

func writeDiscoveredScaffoldRow(root string, row formal.ManifestRow, discoveredKeys map[string]inventoryEntry, template formal.ControllerBinding) (int, error) {
	entry, ok := discoveredKeys[rowKey(row.Service, row.Slug)]
	if !ok || row.Stage != "scaffold" {
		return 0, nil
	}

	artifacts, err := scaffoldForRow(template, row, entry)
	if err != nil {
		return 0, err
	}
	return writeScaffoldArtifacts(root, row, artifacts)
}

func renderAndVerifyScaffold(inputs generateInputs, providerPath string, report *Report) error {
	diagramReport, err := formal.RenderDiagrams(formal.RenderOptions{Root: inputs.Root})
	if err != nil {
		return err
	}
	report.FilesWritten += diagramReport.FilesWritten

	if _, err := formal.Verify(inputs.Root); err != nil {
		return err
	}
	if providerPath == "" {
		return nil
	}
	_, err = VerifyCoverage(Options{
		Root:         inputs.Root,
		ConfigPath:   inputs.ConfigPath,
		ProviderPath: providerPath,
	})
	return err
}

func discoverPublishedKinds(repoRoot string, cfg *generator.Config) ([]inventoryEntry, []generator.ServiceConfig, error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("generator config is required")
	}

	services, err := cfg.SelectServices("", true)
	if err != nil {
		return nil, nil, err
	}

	entries := make([]inventoryEntry, 0, len(services))
	seen := map[string]string{}
	for _, service := range services {
		serviceEntries, err := discoverServicePublishedKinds(repoRoot, cfg, service, seen)
		if err != nil {
			return nil, nil, err
		}
		entries = append(entries, serviceEntries...)
	}

	sortInventoryEntries(entries)
	return entries, services, nil
}

func discoverServicePublishedKinds(repoRoot string, cfg *generator.Config, service generator.ServiceConfig, seen map[string]string) ([]inventoryEntry, error) {
	version := service.VersionOrDefault(cfg.DefaultVersion)
	apiDir := filepath.Join(repoRoot, "api", service.Group, version)
	dirEntries, err := os.ReadDir(apiDir)
	if err != nil {
		return nil, fmt.Errorf("read published API directory %q for service %q: %w", apiDir, service.Service, err)
	}

	selectedKinds := selectedKindSet(service.SelectedKinds())
	remainingSelectedKinds := copyKindSet(selectedKinds)
	entries := make([]inventoryEntry, 0, len(dirEntries))
	for _, dirEntry := range dirEntries {
		entry, ok, err := publishedInventoryEntry(apiDir, service, version, dirEntry, selectedKinds, remainingSelectedKinds, seen)
		if err != nil {
			return nil, err
		}
		if ok {
			entries = append(entries, entry)
		}
	}

	if err := validateSelectedKindsFound(service.Service, apiDir, remainingSelectedKinds); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("service %q has no published API kinds under %q", service.Service, apiDir)
	}
	return entries, nil
}

func publishedInventoryEntry(apiDir string, service generator.ServiceConfig, version string, dirEntry os.DirEntry, selectedKinds map[string]struct{}, remainingSelectedKinds map[string]struct{}, seen map[string]string) (inventoryEntry, bool, error) {
	if dirEntry.IsDir() || !strings.HasSuffix(dirEntry.Name(), "_types.go") {
		return inventoryEntry{}, false, nil
	}

	path := filepath.Join(apiDir, dirEntry.Name())
	kind, err := publishedKindFromFile(path)
	if err != nil {
		return inventoryEntry{}, false, err
	}
	if !selectedKindAllowed(kind, selectedKinds, remainingSelectedKinds) {
		return inventoryEntry{}, false, nil
	}

	slug := strings.TrimSpace(service.FormalSpecFor(kind))
	if slug == "" {
		slug = strings.TrimSuffix(dirEntry.Name(), "_types.go")
	}
	key := rowKey(service.Service, slug)
	if previous, ok := seen[key]; ok {
		return inventoryEntry{}, false, fmt.Errorf("duplicate published API kind key %s from %q and %q", key, previous, path)
	}
	seen[key] = path

	return inventoryEntry{
		Service: service.Service,
		Group:   service.Group,
		Version: version,
		Slug:    slug,
		Kind:    kind,
	}, true, nil
}

func selectedKindAllowed(kind string, selectedKinds map[string]struct{}, remainingSelectedKinds map[string]struct{}) bool {
	if len(selectedKinds) == 0 {
		return true
	}
	if _, ok := selectedKinds[kind]; !ok {
		return false
	}
	delete(remainingSelectedKinds, kind)
	return true
}

func validateSelectedKindsFound(serviceName string, apiDir string, remainingSelectedKinds map[string]struct{}) error {
	if len(remainingSelectedKinds) == 0 {
		return nil
	}

	missing := sortedKindSet(remainingSelectedKinds)
	return fmt.Errorf(
		"service %q selected kinds %s were not found under %q",
		serviceName,
		strings.Join(missing, ", "),
		apiDir,
	)
}

func copyKindSet(values map[string]struct{}) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(values))
	for value := range values {
		out[value] = struct{}{}
	}
	return out
}

func sortedKindSet(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func shouldPreserveManifestRow(cfg *generator.Config, row formal.ManifestRow) bool {
	if row.Service == "template" && row.Slug == "template" {
		return true
	}
	if cfg == nil {
		return false
	}

	for _, service := range cfg.Services {
		if service.Service != row.Service {
			continue
		}
		return strings.TrimSpace(service.FormalSpecFor(row.Kind)) == row.Slug
	}
	return false
}

func addConfiguredFormalSpecRows(formalRoot string, cfg *generator.Config, rowByKey map[string]formal.ManifestRow) (int, error) {
	if cfg == nil {
		return 0, nil
	}

	added := 0
	for _, service := range cfg.Services {
		serviceAdded, err := addConfiguredFormalSpecRowsForService(formalRoot, service, rowByKey)
		if err != nil {
			return 0, err
		}
		added += serviceAdded
	}

	return added, nil
}

func addConfiguredFormalSpecRowsForService(formalRoot string, service generator.ServiceConfig, rowByKey map[string]formal.ManifestRow) (int, error) {
	added := 0
	for _, resource := range service.Generation.Resources {
		addedRow, err := addConfiguredFormalSpecRow(formalRoot, service, resource.Kind, strings.TrimSpace(resource.FormalSpec), rowByKey)
		if err != nil {
			return 0, err
		}
		if addedRow {
			added++
		}
	}
	return added, nil
}

func addConfiguredFormalSpecRow(formalRoot string, service generator.ServiceConfig, kind string, slug string, rowByKey map[string]formal.ManifestRow) (bool, error) {
	if slug == "" {
		return false, nil
	}
	key := rowKey(service.Service, slug)
	if _, ok := rowByKey[key]; ok {
		return false, nil
	}

	row, ok, err := configuredFormalSpecManifestRow(formalRoot, service.Service, kind, slug)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	rowByKey[key] = row
	return true, nil
}

func configuredFormalSpecManifestRow(formalRoot string, serviceName string, kind string, slug string) (formal.ManifestRow, bool, error) {
	specPath := path.Join("controllers", serviceName, slug, "spec.cfg")
	stage, surface, ok, err := manifestMetadataFromSpecFile(filepath.Join(formalRoot, filepath.FromSlash(specPath)))
	if err != nil {
		return formal.ManifestRow{}, false, err
	}
	if !ok {
		return formal.ManifestRow{}, false, nil
	}
	return formal.ManifestRow{
		Service:    serviceName,
		Slug:       slug,
		Kind:       kind,
		Stage:      stage,
		Surface:    surface,
		ImportPath: path.Join("imports", serviceName, slug+".json"),
		SpecPath:   specPath,
		LogicPath:  path.Join("controllers", serviceName, slug, "logic-gaps.md"),
		DiagramDir: path.Join("controllers", serviceName, slug, "diagrams"),
	}, true, nil
}

func manifestMetadataFromSpecFile(path string) (string, string, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", false, nil
		}
		return "", "", false, fmt.Errorf("read formal spec %q: %w", filepath.ToSlash(path), err)
	}

	stage := ""
	surface := "repo-authored-semantics"
	for _, line := range strings.Split(string(content), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "stage":
			stage = strings.TrimSpace(value)
		case "surface":
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				surface = trimmed
			}
		}
	}

	if stage == "" {
		return "", "", false, fmt.Errorf("formal spec %q is missing stage metadata", filepath.ToSlash(path))
	}
	return stage, surface, true, nil
}

func selectedKindSet(kinds []string) map[string]struct{} {
	if len(kinds) == 0 {
		return nil
	}
	selected := make(map[string]struct{}, len(kinds))
	for _, kind := range kinds {
		selected[kind] = struct{}{}
	}
	return selected
}

func discoverProviderKinds(providerPath string) ([]inventoryEntry, error) {
	providerEntries, err := formal.DiscoverProviderInventory(providerPath)
	if err != nil {
		return nil, err
	}

	entries := make([]inventoryEntry, 0, len(providerEntries))
	for _, entry := range providerEntries {
		entries = append(entries, inventoryEntry{
			Service:          entry.Service,
			Group:            entry.Service,
			Slug:             entry.Slug,
			Kind:             entry.Kind,
			ProviderResource: entry.TerraformName,
		})
	}
	sortInventoryEntries(entries)
	return entries, nil
}

func mergeInventoryEntries(primary, secondary []inventoryEntry) ([]inventoryEntry, error) {
	merged := map[string]inventoryEntry{}
	for _, entry := range append(append([]inventoryEntry(nil), primary...), secondary...) {
		key := rowKey(entry.Service, entry.Slug)
		if current, ok := merged[key]; ok {
			if current.Kind != entry.Kind {
				return nil, fmt.Errorf("formal inventory key %s has conflicting kinds %q and %q", key, current.Kind, entry.Kind)
			}
			if current.ProviderResource == "" && entry.ProviderResource != "" {
				current.ProviderResource = entry.ProviderResource
			}
			merged[key] = current
			continue
		}
		merged[key] = entry
	}

	out := make([]inventoryEntry, 0, len(merged))
	for _, entry := range merged {
		out = append(out, entry)
	}
	sortInventoryEntries(out)
	return out, nil
}

func publishedKindFromFile(path string) (string, error) {
	fileSet := token.NewFileSet()
	node, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("parse published API file %q: %w", path, err)
	}

	kinds := publishedRootKinds(node)
	switch len(kinds) {
	case 1:
		return kinds[0], nil
	case 0:
		return "", fmt.Errorf("published API file %q does not define a non-list kubebuilder root kind", path)
	default:
		return "", fmt.Errorf("published API file %q defines multiple non-list kubebuilder root kinds: %s", path, strings.Join(kinds, ", "))
	}
}

func publishedRootKinds(node *ast.File) []string {
	var kinds []string
	for _, decl := range node.Decls {
		kinds = append(kinds, publishedRootKindsFromDecl(decl)...)
	}
	return kinds
}

func publishedRootKindsFromDecl(decl ast.Decl) []string {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.TYPE {
		return nil
	}

	var kinds []string
	for _, spec := range genDecl.Specs {
		kind, ok := publishedRootKindFromSpec(genDecl.Doc, spec)
		if ok {
			kinds = append(kinds, kind)
		}
	}
	return kinds
}

func publishedRootKindFromSpec(genDoc *ast.CommentGroup, spec ast.Spec) (string, bool) {
	typeSpec, ok := spec.(*ast.TypeSpec)
	if !ok {
		return "", false
	}
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok || !hasRootMarker(genDoc, typeSpec.Doc) || isListContainerType(structType) {
		return "", false
	}
	return typeSpec.Name.Name, true
}

func sortInventoryEntries(entries []inventoryEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Service != entries[j].Service {
			return entries[i].Service < entries[j].Service
		}
		if entries[i].Slug != entries[j].Slug {
			return entries[i].Slug < entries[j].Slug
		}
		return entries[i].Kind < entries[j].Kind
	})
}

func isListContainerType(structType *ast.StructType) bool {
	if structType == nil || structType.Fields == nil {
		return false
	}
	hasItems := false
	hasListMeta := false
	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			if name.Name == "Items" {
				hasItems = true
				break
			}
		}
		if selector, ok := field.Type.(*ast.SelectorExpr); ok && selector.Sel != nil && selector.Sel.Name == "ListMeta" {
			hasListMeta = true
		}
	}
	return hasItems && hasListMeta
}

func hasRootMarker(groups ...*ast.CommentGroup) bool {
	for _, group := range groups {
		if group == nil {
			continue
		}
		for _, comment := range group.List {
			if strings.Contains(comment.Text, "+kubebuilder:object:root=true") {
				return true
			}
		}
	}
	return false
}

func scaffoldManifestRow(entry inventoryEntry) formal.ManifestRow {
	return formal.ManifestRow{
		Service:    entry.Service,
		Slug:       entry.Slug,
		Kind:       entry.Kind,
		Stage:      "scaffold",
		Surface:    "repo-authored-semantics",
		ImportPath: path.Join("imports", entry.Service, entry.Slug+".json"),
		SpecPath:   path.Join("controllers", entry.Service, entry.Slug, "spec.cfg"),
		LogicPath:  path.Join("controllers", entry.Service, entry.Slug, "logic-gaps.md"),
		DiagramDir: path.Join("controllers", entry.Service, entry.Slug, "diagrams"),
	}
}

func scaffoldForRow(template formal.ControllerBinding, row formal.ManifestRow, entry inventoryEntry) (scaffoldArtifacts, error) {
	importDoc, err := renderImport(template.Import, row, entry)
	if err != nil {
		return scaffoldArtifacts{}, err
	}
	return scaffoldArtifacts{
		Spec:    renderSpec(template.Spec.SharedContracts, row),
		Logic:   renderLogicGaps(row),
		Diagram: renderDiagram(row),
		Import:  importDoc,
	}, nil
}

func renderManifest(rows []formal.ManifestRow) []byte {
	var b strings.Builder
	b.WriteString(manifestHeader)
	for _, row := range rows {
		fmt.Fprintf(
			&b,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.Service,
			row.Slug,
			row.Kind,
			row.Stage,
			row.Surface,
			row.ImportPath,
			row.SpecPath,
			row.LogicPath,
			row.DiagramDir,
		)
	}
	return []byte(b.String())
}

func renderSpec(sharedContracts []string, row formal.ManifestRow) []byte {
	var b strings.Builder
	b.WriteString("# formal controller binding schema v1\n")
	b.WriteString("schema_version = 1\n")
	b.WriteString("surface = repo-authored-semantics\n")
	fmt.Fprintf(&b, "service = %s\n", row.Service)
	fmt.Fprintf(&b, "slug = %s\n", row.Slug)
	fmt.Fprintf(&b, "kind = %s\n", row.Kind)
	fmt.Fprintf(&b, "stage = %s\n", row.Stage)
	fmt.Fprintf(&b, "import = %s\n", row.ImportPath)
	fmt.Fprintf(&b, "shared_contracts = %s\n", strings.Join(sharedContracts, ","))
	b.WriteString("status_projection = required\n")
	b.WriteString("success_condition = active\n")
	b.WriteString("requeue_conditions = provisioning,updating,terminating\n")
	b.WriteString("delete_confirmation = required\n")
	b.WriteString("finalizer_policy = retain-until-confirmed-delete\n")
	b.WriteString("secret_side_effects = none\n")
	return []byte(b.String())
}

func renderLogicGaps(row formal.ManifestRow) []byte {
	return []byte(fmt.Sprintf(`---
schemaVersion: 1
surface: repo-authored-semantics
service: %s
slug: %s
gaps: []
---

# Logic Gaps

This scaffold row tracks the published %s API shape for %s. Replace this
placeholder with repo-authored semantics and explicit stop conditions before
adding formalSpec or promoting runtime ownership.
`, row.Service, row.Slug, row.Kind, row.Service))
}

func renderDiagram(row formal.ManifestRow) []byte {
	return []byte(fmt.Sprintf(`schemaVersion: 1
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
  - Scaffold metadata only; replace with controller-specific runtime states before promotion.
`, row.Service, row.Slug, row.Kind))
}

func renderImport(template formal.ImportModel, row formal.ManifestRow, entry inventoryEntry) ([]byte, error) {
	doc := template
	doc.Service = row.Service
	doc.Slug = row.Slug
	doc.Kind = row.Kind
	doc.ProviderResource = scaffoldProviderResource(row)
	if strings.TrimSpace(entry.ProviderResource) != "" {
		doc.ProviderResource = entry.ProviderResource
	}
	doc.Operations = formal.Operations{
		Create: []formal.OperationBinding{{
			Operation:    "Create" + row.Kind,
			RequestType:  row.Service + ".Create" + row.Kind + "Request",
			ResponseType: row.Service + ".Create" + row.Kind + "Response",
		}},
		Get: []formal.OperationBinding{{
			Operation:    "Get" + row.Kind,
			RequestType:  row.Service + ".Get" + row.Kind + "Request",
			ResponseType: row.Service + ".Get" + row.Kind + "Response",
		}},
		List: []formal.OperationBinding{{
			Operation:    "List" + row.Kind,
			RequestType:  row.Service + ".List" + row.Kind + "Request",
			ResponseType: row.Service + ".List" + row.Kind + "Response",
		}},
		Update: []formal.OperationBinding{{
			Operation:    "Update" + row.Kind,
			RequestType:  row.Service + ".Update" + row.Kind + "Request",
			ResponseType: row.Service + ".Update" + row.Kind + "Response",
		}},
		Delete: []formal.OperationBinding{{
			Operation:    "Delete" + row.Kind,
			RequestType:  row.Service + ".Delete" + row.Kind + "Request",
			ResponseType: row.Service + ".Delete" + row.Kind + "Response",
		}},
	}
	doc.ListLookup = &formal.ListLookup{
		Datasource:         "scaffold_" + row.Service + "_" + row.Slug,
		CollectionField:    row.Slug,
		ResponseItemsField: "Items",
		FilterFields:       []string{"compartment_id", "state"},
	}
	doc.Boundary.RepoAuthoredSpecPath = row.SpecPath
	doc.Boundary.RepoAuthoredLogicGapsPath = row.LogicPath
	doc.Notes = []string{
		fmt.Sprintf("Scaffold placeholder for %s/%s; replace imported facts before promoting this row beyond scaffold.", row.Service, row.Slug),
	}

	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal scaffold import for %s/%s: %w", row.Service, row.Slug, err)
	}
	payload = append(payload, '\n')
	return payload, nil
}

func scaffoldProviderResource(row formal.ManifestRow) string {
	return "scaffold_" + row.Service + "_" + row.Slug
}

func writeScaffoldArtifacts(root string, row formal.ManifestRow, artifacts scaffoldArtifacts) (int, error) {
	type target struct {
		path string
		data []byte
	}
	targets := []target{
		{path: filepath.Join(root, filepath.FromSlash(row.SpecPath)), data: artifacts.Spec},
		{path: filepath.Join(root, filepath.FromSlash(row.LogicPath)), data: artifacts.Logic},
		{path: filepath.Join(root, filepath.FromSlash(path.Join(row.DiagramDir, "runtime-lifecycle.yaml"))), data: artifacts.Diagram},
		{path: filepath.Join(root, filepath.FromSlash(row.ImportPath)), data: artifacts.Import},
	}

	writes := 0
	for _, target := range targets {
		changed, err := writeFileIfChanged(target.path, target.data)
		if err != nil {
			return writes, err
		}
		if changed {
			writes++
		}
	}
	return writes, nil
}

func pruneStaleFormalArtifacts(root string, rows []formal.ManifestRow) error {
	desiredControllerRoots := make(map[string]struct{}, len(rows))
	desiredImportPaths := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		desiredControllerRoots[filepath.Clean(filepath.Dir(filepath.FromSlash(row.SpecPath)))] = struct{}{}
		desiredImportPaths[filepath.Clean(filepath.FromSlash(row.ImportPath))] = struct{}{}
	}

	if err := pruneStaleControllerArtifacts(filepath.Join(root, "controllers"), root, desiredControllerRoots); err != nil {
		return err
	}
	if err := pruneStaleImportArtifacts(filepath.Join(root, "imports"), root, desiredImportPaths); err != nil {
		return err
	}
	return nil
}

func pruneStaleControllerArtifacts(controllersRoot string, formalRoot string, desired map[string]struct{}) error {
	staleRoots, err := discoverStaleControllerRoots(controllersRoot, formalRoot, desired)
	if err != nil {
		return fmt.Errorf("scan formal controller artifacts: %w", err)
	}
	return removeStaleControllerRoots(staleRoots, formalRoot, controllersRoot)
}

func discoverStaleControllerRoots(controllersRoot string, formalRoot string, desired map[string]struct{}) ([]string, error) {
	exists, err := dirExists(controllersRoot)
	if err != nil {
		return nil, fmt.Errorf("stat formal controllers root %q: %w", filepath.ToSlash(controllersRoot), err)
	}
	if !exists {
		return nil, nil
	}

	staleRoots, err := collectStaleControllerRoots(controllersRoot, formalRoot, desired)
	if err != nil {
		return nil, err
	}
	return sortStaleControllerRoots(staleRoots), nil
}

func dirExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func collectStaleControllerRoots(controllersRoot string, formalRoot string, desired map[string]struct{}) (map[string]struct{}, error) {
	staleRoots := map[string]struct{}{}
	if err := filepath.WalkDir(controllersRoot, func(path string, d fs.DirEntry, walkErr error) error {
		return markStaleControllerRoot(staleRoots, desired, formalRoot, path, d, walkErr)
	}); err != nil {
		return nil, err
	}
	return staleRoots, nil
}

func markStaleControllerRoot(staleRoots map[string]struct{}, desired map[string]struct{}, formalRoot string, path string, d fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}

	rel, ok, skipDir, err := controllerArtifactRoot(formalRoot, path, d)
	if err != nil {
		return err
	}
	if ok {
		if _, keep := desired[rel]; !keep {
			staleRoots[rel] = struct{}{}
		}
	}
	if skipDir {
		return fs.SkipDir
	}
	return nil
}

func sortStaleControllerRoots(staleRoots map[string]struct{}) []string {
	roots := make([]string, 0, len(staleRoots))
	for rel := range staleRoots {
		roots = append(roots, rel)
	}
	sort.Slice(roots, func(i, j int) bool {
		if len(roots[i]) != len(roots[j]) {
			return len(roots[i]) > len(roots[j])
		}
		return roots[i] < roots[j]
	})
	return roots
}

func controllerArtifactRoot(formalRoot string, path string, d fs.DirEntry) (string, bool, bool, error) {
	if d.IsDir() {
		if d.Name() != "diagrams" {
			return "", false, false, nil
		}
		rel, err := filepath.Rel(formalRoot, filepath.Dir(path))
		if err != nil {
			return "", false, false, err
		}
		return filepath.Clean(rel), true, true, nil
	}
	if d.Name() != "spec.cfg" && d.Name() != "logic-gaps.md" {
		return "", false, false, nil
	}
	rel, err := filepath.Rel(formalRoot, filepath.Dir(path))
	if err != nil {
		return "", false, false, err
	}
	return filepath.Clean(rel), true, false, nil
}

func removeStaleControllerRoots(roots []string, formalRoot string, controllersRoot string) error {
	for _, rel := range roots {
		path := filepath.Join(formalRoot, rel)
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove stale formal controller artifacts %q: %w", filepath.ToSlash(rel), err)
		}
		if err := pruneEmptyParents(filepath.Dir(path), controllersRoot); err != nil {
			return err
		}
	}

	return nil
}

func pruneStaleImportArtifacts(importsRoot string, formalRoot string, desired map[string]struct{}) error {
	stale, err := discoverStaleImportPaths(importsRoot, formalRoot, desired)
	if err != nil {
		return fmt.Errorf("scan formal import artifacts: %w", err)
	}
	return removeStaleImportPaths(stale, formalRoot, importsRoot)
}

func discoverStaleImportPaths(importsRoot string, formalRoot string, desired map[string]struct{}) ([]string, error) {
	if _, err := os.Stat(importsRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat formal imports root %q: %w", filepath.ToSlash(importsRoot), err)
	}

	var stale []string
	if err := filepath.WalkDir(importsRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".json" {
			return nil
		}

		rel, err := filepath.Rel(formalRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.Clean(rel)
		if _, ok := desired[rel]; !ok {
			stale = append(stale, rel)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(stale)
	return stale, nil
}

func removeStaleImportPaths(stale []string, formalRoot string, importsRoot string) error {
	for _, rel := range stale {
		path := filepath.Join(formalRoot, rel)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove stale formal import artifact %q: %w", filepath.ToSlash(rel), err)
		}
		if err := pruneEmptyParents(filepath.Dir(path), importsRoot); err != nil {
			return err
		}
	}

	return nil
}

func pruneEmptyParents(dir string, stop string) error {
	stop = filepath.Clean(stop)
	for {
		dir = filepath.Clean(dir)
		if isPruneStop(dir, stop) {
			return nil
		}

		advance, err := pruneEmptyDir(dir)
		if err != nil {
			return err
		}
		if !advance {
			return nil
		}
		dir = filepath.Dir(dir)
	}
}

func isPruneStop(dir string, stop string) bool {
	return dir == stop || dir == "." || dir == string(filepath.Separator)
}

func pruneEmptyDir(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("read %q: %w", filepath.ToSlash(dir), err)
	}
	if len(entries) != 0 {
		return false, nil
	}
	if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("remove empty formal directory %q: %w", filepath.ToSlash(dir), err)
	}
	return true, nil
}

func writeFileIfChanged(path string, contents []byte) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("create %q: %w", filepath.Dir(path), err)
	}

	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, contents) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("read %q: %w", path, err)
	}

	if err := os.WriteFile(path, contents, 0o644); err != nil {
		return false, fmt.Errorf("write %q: %w", path, err)
	}
	return true, nil
}

func sortManifestRows(rows []formal.ManifestRow) {
	sort.Slice(rows, func(i, j int) bool {
		left := rows[i]
		right := rows[j]
		if left.Service == "template" || right.Service == "template" {
			return left.Service == "template" && right.Service != "template"
		}
		if left.Service != right.Service {
			return left.Service < right.Service
		}
		if left.Slug != right.Slug {
			return left.Slug < right.Slug
		}
		return left.Kind < right.Kind
	})
}

func rowKey(service string, slug string) string {
	return strings.TrimSpace(service) + "\x00" + strings.TrimSpace(slug)
}

func requireDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%q does not exist", path)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", path)
	}
	return nil
}
