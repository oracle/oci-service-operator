/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package formal

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Catalog is the typed formal controller catalog derived from the repo-local scaffold.
type Catalog struct {
	Root        string
	Controllers []ControllerBinding

	byKey map[string]ControllerBinding
}

// ControllerBinding joins one manifest row with its typed repo-authored and provider-fact inputs.
type ControllerBinding struct {
	Manifest  ManifestRow
	Spec      SpecModel
	LogicGaps []LogicGap
	Import    ImportModel
}

// ManifestRow is one controller row from formal/controller_manifest.tsv.
type ManifestRow = manifestRow

// LogicGap is one repo-authored stop-condition entry from logic-gaps front matter.
type LogicGap = logicGap

// ImportModel is the normalized provider-fact JSON committed under formal/imports/.
type ImportModel = importFile

// Operations groups provider-backed CRUD operation bindings.
type Operations = operations

// OperationBinding is one provider-backed request/response binding.
type OperationBinding = operationBinding

// Lifecycle captures the provider lifecycle states extracted for a resource.
type Lifecycle = lifecycle

// LifecyclePhase captures the pending and target states for one lifecycle phase.
type LifecyclePhase = lifecyclePhase

// Mutation describes mutable, force-new, and conflictsWith provider facts.
type Mutation = mutation

// Hooks describes provider-backed waiter or helper hooks discovered during import.
type Hooks = hooks

// Hook is one imported helper hook binding.
type Hook = hook

// ListLookup describes the imported provider list lookup hints.
type ListLookup = listLookup

// Boundary describes the provider-facts versus repo-authored semantics split.
type Boundary = importBoundary

// SpecModel is the typed repo-authored spec.cfg binding for one controller row.
type SpecModel struct {
	SchemaVersion      int
	Surface            string
	Service            string
	Slug               string
	Kind               string
	Stage              string
	ImportPath         string
	SharedContracts    []string
	StatusProjection   string
	SuccessCondition   string
	RequeueConditions  []string
	DeleteConfirmation string
	FinalizerPolicy    string
	SecretSideEffects  string
}

// LoadCatalog verifies and loads the repo-local formal catalog into typed controller bindings.
func LoadCatalog(root string) (*Catalog, error) {
	return loadCatalog(root, true)
}

// LoadCatalogUnchecked loads the repo-local formal catalog without running the
// strict verifier first. This is used by scaffold regeneration so older formal
// trees can be upgraded into a newer contract.
func LoadCatalogUnchecked(root string) (*Catalog, error) {
	return loadCatalog(root, false)
}

func loadCatalog(root string, verify bool) (*Catalog, error) {
	reportRoot := filepath.Clean(root)
	if verify {
		report, err := Verify(root)
		if err != nil {
			return nil, err
		}
		reportRoot = report.Root
	} else {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		reportRoot = absRoot
	}

	rows, err := loadManifest(filepath.Join(reportRoot, "controller_manifest.tsv"))
	if err != nil {
		return nil, fmt.Errorf("load controller manifest: %w", err)
	}

	catalog := &Catalog{
		Root:        reportRoot,
		Controllers: make([]ControllerBinding, 0, len(rows)),
		byKey:       make(map[string]ControllerBinding, len(rows)),
	}
	for _, row := range rows {
		specValues, err := loadSpec(filepath.Join(reportRoot, row.SpecPath))
		if err != nil {
			return nil, fmt.Errorf("load spec %q: %w", filepath.ToSlash(row.SpecPath), err)
		}
		spec, err := parseSpecModel(row.SpecPath, specValues)
		if err != nil {
			return nil, err
		}

		logicGaps, err := loadLogicGaps(filepath.Join(reportRoot, row.LogicPath))
		if err != nil {
			return nil, fmt.Errorf("load logic gaps %q: %w", filepath.ToSlash(row.LogicPath), err)
		}
		importDoc, err := loadImport(filepath.Join(reportRoot, row.ImportPath))
		if err != nil {
			return nil, fmt.Errorf("load import %q: %w", filepath.ToSlash(row.ImportPath), err)
		}

		binding := ControllerBinding{
			Manifest:  row,
			Spec:      spec,
			LogicGaps: append([]LogicGap(nil), logicGaps.Gaps...),
			Import:    importDoc,
		}
		key := catalogKey(row.Service, row.Slug)
		if _, exists := catalog.byKey[key]; exists {
			return nil, fmt.Errorf("duplicate formal controller binding for %s/%s", row.Service, row.Slug)
		}
		catalog.byKey[key] = binding
		catalog.Controllers = append(catalog.Controllers, binding)
	}

	sort.Slice(catalog.Controllers, func(i, j int) bool {
		left := catalog.Controllers[i].Manifest
		right := catalog.Controllers[j].Manifest
		if left.Service != right.Service {
			return left.Service < right.Service
		}
		if left.Slug != right.Slug {
			return left.Slug < right.Slug
		}
		return left.Kind < right.Kind
	})

	return catalog, nil
}

// Lookup returns one formal controller binding by service and slug.
func (c *Catalog) Lookup(service string, slug string) (ControllerBinding, bool) {
	if c == nil {
		return ControllerBinding{}, false
	}
	binding, ok := c.byKey[catalogKey(strings.TrimSpace(service), strings.TrimSpace(slug))]
	return binding, ok
}

func parseSpecModel(path string, values map[string]string) (SpecModel, error) {
	schemaVersion, err := strconv.Atoi(strings.TrimSpace(values["schema_version"]))
	if err != nil {
		return SpecModel{}, fmt.Errorf("parse spec %q schema_version: %w", filepath.ToSlash(path), err)
	}

	return SpecModel{
		SchemaVersion:      schemaVersion,
		Surface:            strings.TrimSpace(values["surface"]),
		Service:            strings.TrimSpace(values["service"]),
		Slug:               strings.TrimSpace(values["slug"]),
		Kind:               strings.TrimSpace(values["kind"]),
		Stage:              strings.TrimSpace(values["stage"]),
		ImportPath:         strings.TrimSpace(values["import"]),
		SharedContracts:    append([]string(nil), splitList(values["shared_contracts"])...),
		StatusProjection:   strings.TrimSpace(values["status_projection"]),
		SuccessCondition:   strings.TrimSpace(values["success_condition"]),
		RequeueConditions:  append([]string(nil), splitList(values["requeue_conditions"])...),
		DeleteConfirmation: strings.TrimSpace(values["delete_confirmation"]),
		FinalizerPolicy:    strings.TrimSpace(values["finalizer_policy"]),
		SecretSideEffects:  strings.TrimSpace(values["secret_side_effects"]),
	}, nil
}

func catalogKey(service string, slug string) string {
	return service + "\x00" + slug
}
