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

	"github.com/oracle/oci-service-operator/internal/formal"
	"github.com/oracle/oci-service-operator/internal/ocisdk"
)

var operationPattern = regexp.MustCompile(`^(Create|Get|List|Update|Delete)(.+?)(Request|Response)$`)

type packageDirResolver = ocisdk.ResolveDirFunc

// Discoverer builds the intermediate generator model from OCI SDK packages.
type Discoverer struct {
	resolveDir packageDirResolver
	index      *ocisdk.Index
	formal     map[string]*formal.Catalog
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
	if err := d.attachFormalModels(cfg, service, pkg); err != nil {
		return nil, fmt.Errorf("attach formal model: %w", err)
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

type resourceCandidate struct {
	rawName             string
	operations          map[string]struct{}
	requestBodyPayloads []string
}

func resourceModels(index *ocisdk.Package, service ServiceConfig) ([]ResourceModel, error) {
	candidates, err := discoverResourceCandidates(index)
	if err != nil {
		return nil, err
	}

	resources := make([]ResourceModel, 0, len(candidates))
	for _, entry := range candidates {
		resource, buildErr := buildResourceModel(index, service, entry)
		if buildErr != nil {
			return nil, buildErr
		}
		resources = append(resources, resource)
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Kind < resources[j].Kind
	})

	return resources, nil
}

func discoverResourceCandidates(index *ocisdk.Package) ([]resourceCandidate, error) {
	candidateMap := make(map[string]*resourceCandidate)
	for _, typeName := range index.TypeNames() {
		operation, rawName, requestType, ok := operationTypeParts(typeName)
		if !ok {
			continue
		}

		entry, exists := candidateMap[rawName]
		if !exists {
			entry = &resourceCandidate{
				rawName:    rawName,
				operations: make(map[string]struct{}),
			}
			candidateMap[rawName] = entry
		}

		entry.operations[operation] = struct{}{}
		if requestType {
			entry.requestBodyPayloads = appendUniqueStrings(entry.requestBodyPayloads, index.RequestBodyPayloads(typeName)...)
		}
	}

	if len(candidateMap) == 0 {
		return nil, fmt.Errorf("no CRUD request/response families discovered")
	}

	candidates := make([]resourceCandidate, 0, len(candidateMap))
	for _, entry := range candidateMap {
		candidates = append(candidates, *entry)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].rawName < candidates[j].rawName
	})

	return candidates, nil
}

func operationTypeParts(typeName string) (operation string, rawName string, requestType bool, ok bool) {
	matches := operationPattern.FindStringSubmatch(typeName)
	if len(matches) == 0 {
		return "", "", false, false
	}

	rawName = singularize(matches[2])
	if rawName == "" {
		return "", "", false, false
	}

	return matches[1], rawName, matches[3] == "Request", true
}

func buildResourceModel(index *ocisdk.Package, service ServiceConfig, entry resourceCandidate) (ResourceModel, error) {
	operations := sortedOperations(entry.operations)
	runtimeModel, err := buildRuntimeModel(index, entry.rawName, operations, index.ResourceOperations(entry.rawName))
	if err != nil {
		return ResourceModel{}, fmt.Errorf("discover runtime metadata for %q: %w", entry.rawName, err)
	}

	kind, compatibilityLocked := resolvedResourceKind(entry.rawName, service.Compatibility)
	fieldSet := synthesizeResourceFieldSet(index, service, kind, entry.rawName, desiredStateStructCandidates(entry.rawName, entry.requestBodyPayloads))
	displayField := primaryDisplayField(fieldSet.SpecFields)
	kindPlural := strings.ToLower(pluralize(kind))
	statusTypeName := defaultStatusTypeName(kind)

	return ResourceModel{
		SDKName:             entry.rawName,
		Kind:                kind,
		FileStem:            fileStem(kind),
		KindPlural:          kindPlural,
		Operations:          operations,
		Runtime:             runtimeModel,
		SpecComments:        []string{fmt.Sprintf("%sSpec defines the desired state of %s.", kind, kind)},
		HelperTypes:         fieldSet.HelperTypes,
		SpecFields:          fieldSet.SpecFields,
		StatusTypeName:      statusTypeName,
		StatusComments:      []string{fmt.Sprintf("%s defines the observed state of %s.", statusTypeName, kind)},
		StatusFields:        fieldSet.StatusFields,
		PrintColumns:        defaultPrintColumns(kind, displayField),
		ObjectComments:      []string{fmt.Sprintf("%s is the Schema for the %s API.", kind, kindPlural)},
		ListComments:        []string{fmt.Sprintf("%sList contains a list of %s.", kind, kind)},
		PrimaryDisplayField: displayField,
		CompatibilityLocked: compatibilityLocked,
	}, nil
}

func sortedOperations(operations map[string]struct{}) []string {
	sorted := make([]string, 0, len(operations))
	for operation := range operations {
		sorted = append(sorted, operation)
	}
	sort.Strings(sorted)
	return sorted
}

func resolvedResourceKind(rawName string, compatibility CompatibilityConfig) (string, bool) {
	if kind, ok := compatibilityKind(rawName, compatibility); ok {
		return kind, true
	}
	return rawName, false
}

func primaryDisplayField(fields []FieldModel) string {
	switch {
	case hasField(fields, "DisplayName"):
		return "DisplayName"
	case hasField(fields, "Name"):
		return "Name"
	default:
		return ""
	}
}

func buildRuntimeModel(pkg *ocisdk.Package, rawName string, operations []string, discovered map[string]ocisdk.OperationMethod) (*RuntimeModel, error) {
	if len(discovered) == 0 {
		return nil, nil
	}
	if pkg == nil {
		return nil, fmt.Errorf("sdk package metadata is required")
	}

	clientType, err := runtimeClientType(rawName, discovered)
	if err != nil {
		return nil, err
	}
	constructor, ok := pkg.ClientConstructor(clientType)
	if !ok {
		return nil, fmt.Errorf("client %q does not expose a WithConfigurationProvider constructor", clientType)
	}

	model := &RuntimeModel{
		ClientType:            clientType,
		ClientConstructor:     constructor.Name,
		ClientConstructorKind: string(constructor.Kind),
	}
	for _, operation := range operations {
		method, ok := discovered[operation]
		if !ok {
			continue
		}
		binding, err := buildRuntimeOperationModel(pkg, rawName, operation, method)
		if err != nil {
			return nil, err
		}
		assignRuntimeOperation(model, operation, binding)
	}

	return model, nil
}

func runtimeClientType(rawName string, discovered map[string]ocisdk.OperationMethod) (string, error) {
	var clientType string
	for _, method := range discovered {
		if clientType == "" {
			clientType = method.ClientType
			continue
		}
		if method.ClientType != clientType {
			return "", fmt.Errorf("resource %q spans multiple SDK clients: %q and %q", rawName, clientType, method.ClientType)
		}
	}
	return clientType, nil
}

func assignRuntimeOperation(model *RuntimeModel, operation string, binding *RuntimeOperationModel) {
	if model == nil {
		return
	}

	switch operation {
	case "Create":
		model.Create = binding
	case "Get":
		model.Get = binding
	case "List":
		model.List = binding
	case "Update":
		model.Update = binding
	case "Delete":
		model.Delete = binding
	}
}

func buildRuntimeOperationModel(pkg *ocisdk.Package, rawName string, operation string, method ocisdk.OperationMethod) (*RuntimeOperationModel, error) {
	requestFields, err := runtimeRequestFields(pkg, rawName, operation, method)
	if err != nil {
		return nil, fmt.Errorf("build %s request fields for %q: %w", operation, rawName, err)
	}

	return &RuntimeOperationModel{
		MethodName:       method.MethodName,
		RequestTypeName:  method.RequestType,
		ResponseTypeName: method.ResponseType,
		UsesRequest:      method.UsesRequest,
		RequestFields:    requestFields,
	}, nil
}

func runtimeRequestFields(pkg *ocisdk.Package, rawName string, operation string, method ocisdk.OperationMethod) ([]RuntimeRequestFieldModel, error) {
	if pkg == nil || !method.UsesRequest {
		return nil, nil
	}

	fields := runtimeRequestStructFields(pkg, rawName, operation, method)
	fields = append(fields, runtimeRequestPayloadFields(pkg, method)...)
	return fields, nil
}

func runtimeRequestStructFields(pkg *ocisdk.Package, rawName string, operation string, method ocisdk.OperationMethod) []RuntimeRequestFieldModel {
	requestStruct, ok := pkg.Struct(method.RequestType)
	if !ok {
		return nil
	}

	pathFieldCount := countFieldsByContribution(requestStruct.Fields, ocisdk.FieldContributionPath)
	fields := make([]RuntimeRequestFieldModel, 0, len(requestStruct.Fields))
	for _, field := range requestStruct.Fields {
		if !includeRuntimeRequestField(field) {
			continue
		}
		fields = append(fields, buildRuntimeRequestFieldModel(field, shouldPreferResourceID(operation, rawName, field, pathFieldCount)))
	}

	return fields
}

func runtimeRequestPayloadFields(pkg *ocisdk.Package, method ocisdk.OperationMethod) []RuntimeRequestFieldModel {
	payloads := pkg.RequestBodyPayloads(method.RequestType)
	fields := make([]RuntimeRequestFieldModel, 0, len(payloads))
	for _, payloadType := range payloads {
		fields = append(fields, RuntimeRequestFieldModel{
			FieldName:    payloadType,
			RequestName:  payloadType,
			Contribution: string(ocisdk.FieldContributionBody),
		})
	}
	return fields
}

func countFieldsByContribution(fields []ocisdk.Field, contribution ocisdk.FieldContribution) int {
	count := 0
	for _, field := range fields {
		if field.Contribution == contribution {
			count++
		}
	}
	return count
}

func includeRuntimeRequestField(field ocisdk.Field) bool {
	if field.Name == "RequestMetadata" {
		return false
	}
	switch field.Contribution {
	case ocisdk.FieldContributionHeader, ocisdk.FieldContributionBinary:
		return false
	default:
		return true
	}
}

func buildRuntimeRequestFieldModel(field ocisdk.Field, preferResourceID bool) RuntimeRequestFieldModel {
	return RuntimeRequestFieldModel{
		FieldName:        field.Name,
		RequestName:      field.RequestName,
		Contribution:     string(field.Contribution),
		PreferResourceID: preferResourceID,
	}
}

func shouldPreferResourceID(operation string, rawName string, field ocisdk.Field, pathFieldCount int) bool {
	if operation == "Create" || field.Contribution != ocisdk.FieldContributionPath {
		return false
	}
	if pathFieldCount == 1 {
		return true
	}

	requestName := strings.ToLower(strings.TrimSpace(field.RequestName))
	rawName = strings.ToLower(strings.TrimSpace(rawName))
	return requestName != "" && rawName != "" && strings.Contains(requestName, rawName) && strings.HasSuffix(requestName, "id")
}

func hasField(fields []FieldModel, name string) bool {
	for _, field := range fields {
		if field.Name == name {
			return true
		}
	}
	return false
}

func (d *Discoverer) attachFormalModels(cfg *Config, service ServiceConfig, pkg *PackageModel) error {
	if pkg == nil || !serviceNeedsFormalModels(service, pkg.Resources) {
		return nil
	}

	formalRoot, err := serviceFormalRoot(cfg, service)
	if err != nil {
		return err
	}
	catalog, err := d.formalCatalog(formalRoot)
	if err != nil {
		return fmt.Errorf("load formal catalog %q: %w", formalRoot, err)
	}

	formalByKind, err := attachResourceFormalModels(service, pkg, catalog)
	if err != nil {
		return err
	}
	attachServiceManagerFormalModels(pkg, formalByKind)
	return nil
}

func serviceNeedsFormalModels(service ServiceConfig, resources []ResourceModel) bool {
	for _, resource := range resources {
		if strings.TrimSpace(service.FormalSpecFor(resource.Kind)) != "" {
			return true
		}
	}
	return false
}

func serviceFormalRoot(cfg *Config, service ServiceConfig) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("generator config is required to resolve formal specs")
	}

	formalRoot := cfg.FormalRoot()
	if strings.TrimSpace(formalRoot) == "" {
		return "", fmt.Errorf("service %q declares formalSpec references but the generator config root is unknown", service.Service)
	}
	return formalRoot, nil
}

func attachResourceFormalModels(service ServiceConfig, pkg *PackageModel, catalog *formal.Catalog) (map[string]*FormalModel, error) {
	formalByKind := make(map[string]*FormalModel, len(pkg.Resources))
	for index := range pkg.Resources {
		model, err := resourceFormalModel(catalog, service, pkg.Resources[index].Kind)
		if err != nil {
			return nil, err
		}
		if model == nil {
			continue
		}

		pkg.Resources[index].Formal = model
		if pkg.Resources[index].Runtime != nil {
			pkg.Resources[index].Runtime.Semantics = buildRuntimeSemanticsModel(model, pkg.Resources[index].Runtime)
		}
		formalByKind[pkg.Resources[index].Kind] = model
	}

	return formalByKind, nil
}

func resourceFormalModel(catalog *formal.Catalog, service ServiceConfig, kind string) (*FormalModel, error) {
	slug := service.FormalSpecFor(kind)
	if strings.TrimSpace(slug) == "" {
		return nil, nil
	}

	binding, ok := catalog.Lookup(service.Service, slug)
	if !ok {
		return nil, fmt.Errorf(
			"service %q kind %q formalSpec %q was not found in %s",
			service.Service,
			kind,
			slug,
			filepath.ToSlash(filepath.Join(catalog.Root, "controller_manifest.tsv")),
		)
	}
	if binding.Manifest.Kind != kind {
		return nil, fmt.Errorf(
			"service %q kind %q formalSpec %q resolves to manifest kind %q",
			service.Service,
			kind,
			slug,
			binding.Manifest.Kind,
		)
	}

	return &FormalModel{
		Reference: FormalReferenceModel{
			Service: service.Service,
			Slug:    slug,
		},
		Binding:  binding,
		Diagrams: formal.DiagramFilesForRow(binding.Manifest),
	}, nil
}

func attachServiceManagerFormalModels(pkg *PackageModel, formalByKind map[string]*FormalModel) {
	for index := range pkg.ServiceManagers {
		model, ok := formalByKind[pkg.ServiceManagers[index].Kind]
		if !ok {
			continue
		}

		pkg.ServiceManagers[index].Formal = model
		if runtime := findResourceRuntime(pkg.Resources, pkg.ServiceManagers[index].Kind); runtime != nil {
			pkg.ServiceManagers[index].Semantics = runtime.Semantics
		}
	}
}

func findResourceRuntime(resources []ResourceModel, kind string) *RuntimeModel {
	for index := range resources {
		if resources[index].Kind == kind {
			return resources[index].Runtime
		}
	}
	return nil
}

func (d *Discoverer) formalCatalog(root string) (*formal.Catalog, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("formal root must not be empty")
	}
	if d.formal == nil {
		d.formal = make(map[string]*formal.Catalog)
	}
	if catalog, ok := d.formal[root]; ok {
		return catalog, nil
	}

	catalog, err := formal.LoadCatalog(root)
	if err != nil {
		return nil, err
	}
	d.formal[root] = catalog
	return catalog, nil
}

type fieldScope string

const (
	fieldScopeSpec   fieldScope = "spec"
	fieldScopeStatus fieldScope = "status"
)

type fieldRenderingOptions struct {
	scope                     fieldScope
	escapeStatusJSONCollision bool
	excludedFieldPaths        map[string]struct{}
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
