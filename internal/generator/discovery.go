/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/oracle/oci-service-operator/internal/ocisdk"
)

var operationPattern = regexp.MustCompile(`^(Create|Get|List|Update|Delete)(.+?)(Request|Response)$`)

type packageDirResolver = ocisdk.ResolveDirFunc

// Discoverer builds the intermediate generator model from OCI SDK packages.
type Discoverer struct {
	resolveDir packageDirResolver
	index      *ocisdk.Index
}

// NewDiscoverer returns the default SDK package discoverer.
func NewDiscoverer() *Discoverer {
	return &Discoverer{
		resolveDir: defaultPackageDirResolver,
		index:      ocisdk.NewIndex(defaultPackageDirResolver),
	}
}

// BuildPackageModel loads one OCI SDK package and converts it into a generator model.
func (d *Discoverer) BuildPackageModel(ctx context.Context, cfg *Config, service ServiceConfig) (*PackageModel, error) {
	index, err := d.sdkIndex().Package(ctx, service.SDKPackage)
	if err != nil {
		return nil, fmt.Errorf("load sdk package %q: %w", service.SDKPackage, err)
	}

	resources, err := resourceModels(index, service)
	if err != nil {
		return nil, fmt.Errorf("discover resources for service %q: %w", service.Service, err)
	}

	pkg, err := buildPackageModel(cfg, service, resources)
	if err != nil {
		return nil, fmt.Errorf("build package model: %w", err)
	}

	return pkg, nil
}

func (d *Discoverer) sdkIndex() *ocisdk.Index {
	if d.index != nil {
		return d.index
	}

	resolver := d.resolveDir
	if resolver == nil {
		resolver = defaultPackageDirResolver
		d.resolveDir = resolver
	}
	d.index = ocisdk.NewIndex(resolver)
	return d.index
}

func defaultPackageDirResolver(ctx context.Context, importPath string) (string, error) {
	if moduleRoot, err := findModuleRoot(); err == nil {
		vendorDir := filepath.Join(moduleRoot, "vendor", filepath.FromSlash(importPath))
		if info, statErr := os.Stat(vendorDir); statErr == nil && info.IsDir() {
			return vendorDir, nil
		}
	}

	cmd := exec.CommandContext(ctx, "go", "list", "-buildvcs=false", "-f", "{{.Dir}}", importPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go list %q: %w: %s", importPath, err, strings.TrimSpace(string(output)))
	}

	dir := strings.TrimSpace(string(output))
	if dir == "" {
		return "", fmt.Errorf("go list returned an empty directory for %q", importPath)
	}

	return dir, nil
}

func findModuleRoot() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	currentDir := workingDir
	for {
		if info, statErr := os.Stat(filepath.Join(currentDir, "go.mod")); statErr == nil && !info.IsDir() {
			return currentDir, nil
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", fmt.Errorf("go.mod not found from %q", workingDir)
		}
		currentDir = parentDir
	}
}

func resourceModels(index *ocisdk.Package, service ServiceConfig) ([]ResourceModel, error) {
	type candidate struct {
		rawName             string
		operations          map[string]struct{}
		requestBodyPayloads []string
	}

	candidates := make(map[string]*candidate)
	for _, typeName := range index.TypeNames() {
		matches := operationPattern.FindStringSubmatch(typeName)
		if len(matches) == 0 {
			continue
		}

		rawName := singularize(matches[2])
		if rawName == "" {
			continue
		}
		entry, ok := candidates[rawName]
		if !ok {
			entry = &candidate{
				rawName:    rawName,
				operations: make(map[string]struct{}),
			}
			candidates[rawName] = entry
		}
		entry.operations[matches[1]] = struct{}{}
		if matches[3] == "Request" {
			entry.requestBodyPayloads = appendUniqueStrings(entry.requestBodyPayloads, index.RequestBodyPayloads(typeName)...)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no CRUD request/response families discovered")
	}

	var resources []ResourceModel
	for _, entry := range candidates {
		operations := make([]string, 0, len(entry.operations))
		for operation := range entry.operations {
			operations = append(operations, operation)
		}
		sort.Strings(operations)

		kind := entry.rawName
		compatibilityLocked := false
		if compatibilityKindName, ok := compatibilityKind(entry.rawName, service.Compatibility); ok {
			kind = compatibilityKindName
			compatibilityLocked = true
		}

		fieldSet := synthesizeResourceFieldSet(index, service, kind, entry.rawName, desiredStateStructCandidates(entry.rawName, entry.requestBodyPayloads))
		displayField := ""
		switch {
		case hasField(fieldSet.SpecFields, "DisplayName"):
			displayField = "DisplayName"
		case hasField(fieldSet.SpecFields, "Name"):
			displayField = "Name"
		}

		resources = append(resources, ResourceModel{
			SDKName:             entry.rawName,
			Kind:                kind,
			FileStem:            fileStem(kind),
			KindPlural:          strings.ToLower(pluralize(kind)),
			Operations:          operations,
			SpecComments:        []string{fmt.Sprintf("%sSpec defines the desired state of %s.", kind, kind)},
			HelperTypes:         fieldSet.HelperTypes,
			SpecFields:          fieldSet.SpecFields,
			StatusTypeName:      defaultStatusTypeName(kind),
			StatusComments:      []string{fmt.Sprintf("%s defines the observed state of %s.", defaultStatusTypeName(kind), kind)},
			StatusFields:        fieldSet.StatusFields,
			PrintColumns:        defaultPrintColumns(kind, displayField),
			ObjectComments:      []string{fmt.Sprintf("%s is the Schema for the %s API.", kind, strings.ToLower(pluralize(kind)))},
			ListComments:        []string{fmt.Sprintf("%sList contains a list of %s.", kind, kind)},
			PrimaryDisplayField: displayField,
			CompatibilityLocked: compatibilityLocked,
		})
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Kind < resources[j].Kind
	})

	return resources, nil
}

func hasField(fields []FieldModel, name string) bool {
	for _, field := range fields {
		if field.Name == name {
			return true
		}
	}
	return false
}

func resourceFields(index *ocisdk.Package, rawName string) ([]FieldModel, []FieldModel) {
	specFields, _ := mergeStructFields(index, []string{
		"Create" + rawName + "Details",
		"Update" + rawName + "Details",
	}, nil, fieldRenderingOptions{scope: fieldScopeSpec})

	statusFields := defaultStatusFields()
	statusJSONNames := fieldJSONNames(statusFields)
	observedFields, _ := mergeStructFields(index, []string{
		rawName,
		rawName + "Summary",
	}, nil, fieldRenderingOptions{scope: fieldScopeStatus, escapeStatusJSONCollision: true})
	for _, field := range observedFields {
		jsonName := tagJSONName(field.Tag)
		if _, exists := statusJSONNames[jsonName]; exists {
			continue
		}
		statusFields = append(statusFields, field)
		statusJSONNames[jsonName] = struct{}{}
	}

	return specFields, statusFields
}

type fieldScope string

const (
	fieldScopeSpec   fieldScope = "spec"
	fieldScopeStatus fieldScope = "status"
)

type fieldRenderingOptions struct {
	scope                     fieldScope
	escapeStatusJSONCollision bool
}

func mergeStructFields(index *ocisdk.Package, candidates []string, initial []FieldModel, options fieldRenderingOptions) ([]FieldModel, map[string]struct{}) {
	merged := make([]FieldModel, 0, len(initial)+8)
	seenJSONNames := make(map[string]struct{}, len(initial)+8)
	for _, field := range initial {
		merged = append(merged, field)
		seenJSONNames[tagJSONName(field.Tag)] = struct{}{}
	}

	for _, candidate := range candidates {
		structDef, ok := index.Struct(candidate)
		if !ok {
			continue
		}
		for _, field := range structDef.Fields {
			if field.RenderableType == "" {
				continue
			}
			jsonName := field.JSONName
			if jsonName == "" {
				jsonName = lowerCamel(field.Name)
			}
			if _, exists := seenJSONNames[jsonName]; exists {
				continue
			}

			merged = append(merged, buildFieldModel(field, jsonName, options))
			seenJSONNames[jsonName] = struct{}{}
		}
	}

	return merged, seenJSONNames
}

func buildFieldModel(field ocisdk.Field, jsonName string, options fieldRenderingOptions) FieldModel {
	renderedJSONName := renderedFieldJSONName(jsonName, options)
	comments := fieldComments(field)
	if renderedJSONName != jsonName {
		comments = append(comments, "This uses a distinct JSON name so it can coexist with the OSOK status envelope.")
	}

	return FieldModel{
		Name:     field.Name,
		Type:     field.RenderableType,
		Tag:      jsonTag(renderedJSONName, shouldOmitEmpty(field, options)),
		Comments: comments,
		Markers:  fieldMarkers(field, options),
	}
}

func renderedFieldJSONName(jsonName string, options fieldRenderingOptions) string {
	if options.scope == fieldScopeStatus && options.escapeStatusJSONCollision && strings.TrimSpace(jsonName) == "status" {
		return "sdkStatus"
	}
	return jsonName
}

func fieldComments(field ocisdk.Field) []string {
	documentation := strings.TrimSpace(field.Documentation)
	if documentation == "" {
		return nil
	}

	lines := strings.Split(documentation, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return lines
}

func fieldMarkers(field ocisdk.Field, options fieldRenderingOptions) []string {
	if options.scope != fieldScopeSpec {
		return nil
	}
	if field.ReadOnly {
		return nil
	}
	if field.Mandatory {
		return []string{"+kubebuilder:validation:Required"}
	}
	return []string{"+kubebuilder:validation:Optional"}
}

func shouldOmitEmpty(field ocisdk.Field, options fieldRenderingOptions) bool {
	if options.scope != fieldScopeSpec {
		return true
	}
	if field.ReadOnly {
		return true
	}
	return !field.Mandatory
}

func targetOutputDir(root string, pkg *PackageModel) string {
	return filepath.Join(root, "api", pkg.Service.Group, pkg.Version)
}

func jsonTag(name string, omitEmpty bool) string {
	if omitEmpty {
		return fmt.Sprintf(`json:"%s,omitempty"`, name)
	}
	return fmt.Sprintf(`json:"%s"`, name)
}

func tagJSONName(tag string) string {
	parts := strings.Split(tag, `"`)
	if len(parts) < 2 {
		return ""
	}

	return strings.Split(parts[1], ",")[0]
}

func fieldJSONNames(fields []FieldModel) map[string]struct{} {
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		seen[tagJSONName(field.Tag)] = struct{}{}
	}
	return seen
}
