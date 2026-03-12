/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var operationPattern = regexp.MustCompile(`^(Create|Get|List|Update|Delete)(.+?)(Request|Response)$`)

type packageDirResolver func(context.Context, string) (string, error)

// Discoverer builds the intermediate generator model from OCI SDK packages.
type Discoverer struct {
	resolveDir packageDirResolver
}

// NewDiscoverer returns the default SDK package discoverer.
func NewDiscoverer() *Discoverer {
	return &Discoverer{resolveDir: defaultPackageDirResolver}
}

// BuildPackageModel loads one OCI SDK package and converts it into a generator model.
func (d *Discoverer) BuildPackageModel(ctx context.Context, cfg *Config, service ServiceConfig) (*PackageModel, error) {
	dir, err := d.resolveDir(ctx, service.SDKPackage)
	if err != nil {
		return nil, fmt.Errorf("resolve sdk package %q: %w", service.SDKPackage, err)
	}

	index, err := parseSDKPackage(dir)
	if err != nil {
		return nil, fmt.Errorf("parse sdk package %q: %w", service.SDKPackage, err)
	}

	resources, err := index.resources(service)
	if err != nil {
		return nil, fmt.Errorf("discover resources for service %q: %w", service.Service, err)
	}

	pkg, err := buildPackageModel(cfg, service, resources)
	if err != nil {
		return nil, fmt.Errorf("build package model: %w", err)
	}

	return pkg, nil
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

type packageIndex struct {
	typeNames []string
	structs   map[string]structDef
	aliases   map[string]ast.Expr
}

type structDef struct {
	Fields []structField
}

type structField struct {
	Name     string
	JSONName string
	TypeExpr ast.Expr
}

func parseSDKPackage(dir string) (*packageIndex, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		return !info.IsDir() && strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse dir %q: %w", dir, err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no Go package found in %q", dir)
	}

	index := &packageIndex{
		structs: make(map[string]structDef),
		aliases: make(map[string]ast.Expr),
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || !typeSpec.Name.IsExported() {
						continue
					}

					index.typeNames = append(index.typeNames, typeSpec.Name.Name)
					switch concrete := typeSpec.Type.(type) {
					case *ast.StructType:
						index.structs[typeSpec.Name.Name] = parseStructDef(concrete)
					default:
						index.aliases[typeSpec.Name.Name] = concrete
					}
				}
			}
		}
	}

	sort.Strings(index.typeNames)
	return index, nil
}

func parseStructDef(structType *ast.StructType) structDef {
	definition := structDef{}
	if structType.Fields == nil {
		return definition
	}

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		for _, name := range field.Names {
			if !name.IsExported() {
				continue
			}
			jsonName, ok := jsonFieldName(field.Tag)
			if !ok {
				continue
			}
			definition.Fields = append(definition.Fields, structField{
				Name:     name.Name,
				JSONName: jsonName,
				TypeExpr: field.Type,
			})
		}
	}

	return definition
}

func jsonFieldName(tag *ast.BasicLit) (string, bool) {
	if tag == nil {
		return "", true
	}

	unquoted, err := strconv.Unquote(tag.Value)
	if err != nil {
		return "", false
	}
	jsonTag := reflect.StructTag(unquoted).Get("json")
	if jsonTag == "-" {
		return "", false
	}
	if jsonTag == "" {
		return "", true
	}

	name := strings.Split(jsonTag, ",")[0]
	if name == "-" {
		return "", false
	}
	return name, true
}

func (p *packageIndex) resources(service ServiceConfig) ([]ResourceModel, error) {
	type candidate struct {
		rawName    string
		operations map[string]struct{}
	}

	candidates := make(map[string]*candidate)
	for _, typeName := range p.typeNames {
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

		fields := p.specFields(entry.rawName)
		displayField := ""
		switch {
		case hasField(fields, "DisplayName"):
			displayField = "DisplayName"
		case hasField(fields, "Name"):
			displayField = "Name"
		}

		resources = append(resources, ResourceModel{
			SDKName:             entry.rawName,
			Kind:                kind,
			FileStem:            fileStem(kind),
			KindPlural:          strings.ToLower(pluralize(kind)),
			Operations:          operations,
			SpecComments:        []string{fmt.Sprintf("%sSpec defines the desired state of %s.", kind, kind)},
			SpecFields:          fields,
			StatusTypeName:      defaultStatusTypeName(kind),
			StatusComments:      []string{fmt.Sprintf("%s defines the observed state of %s.", defaultStatusTypeName(kind), kind)},
			StatusFields:        defaultStatusFields(),
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

func (p *packageIndex) specFields(rawName string) []FieldModel {
	defaults := []FieldModel{
		{Name: "Id", Type: "shared.OCID", Tag: jsonTag("id")},
		{Name: "CompartmentId", Type: "shared.OCID", Tag: jsonTag("compartmentId")},
	}

	candidates := []string{
		"Create" + rawName + "Details",
		"Update" + rawName + "Details",
		rawName,
	}

	merged := make([]FieldModel, 0, len(defaults)+8)
	seenJSONNames := make(map[string]struct{}, len(defaults)+8)
	for _, field := range defaults {
		merged = append(merged, field)
		seenJSONNames[tagJSONName(field.Tag)] = struct{}{}
	}

	for _, candidate := range candidates {
		structDef, ok := p.structs[candidate]
		if !ok {
			continue
		}
		for _, field := range structDef.Fields {
			resolvedType, ok := p.renderableType(field.TypeExpr)
			if !ok {
				continue
			}
			jsonName := field.JSONName
			if jsonName == "" {
				jsonName = lowerCamel(field.Name)
			}
			if _, exists := seenJSONNames[jsonName]; exists {
				continue
			}

			merged = append(merged, FieldModel{
				Name: field.Name,
				Type: resolvedType,
				Tag:  jsonTag(jsonName),
			})
			seenJSONNames[jsonName] = struct{}{}
		}
	}

	return merged
}

func (p *packageIndex) renderableType(expr ast.Expr) (string, bool) {
	switch concrete := expr.(type) {
	case *ast.Ident:
		switch concrete.Name {
		case "string", "bool", "int", "int32", "int64", "float32", "float64":
			return concrete.Name, true
		}
		if aliasExpr, ok := p.aliases[concrete.Name]; ok {
			return p.renderableType(aliasExpr)
		}
		return "", false
	case *ast.StarExpr:
		return p.renderableType(concrete.X)
	case *ast.ArrayType:
		elementType, ok := p.renderableType(concrete.Elt)
		if !ok {
			return "", false
		}
		return "[]" + elementType, true
	case *ast.MapType:
		keyType, ok := p.renderableType(concrete.Key)
		if !ok || keyType != "string" {
			return "", false
		}
		valueType, ok := p.renderableType(concrete.Value)
		if !ok {
			return "", false
		}
		return fmt.Sprintf("map[%s]%s", keyType, valueType), true
	case *ast.SelectorExpr:
		switch concrete.Sel.Name {
		case "SDKDate", "SDKTime", "Time":
			return "string", true
		default:
			return "", false
		}
	default:
		return "", false
	}
}

func targetOutputDir(root string, pkg *PackageModel) string {
	return filepath.Join(root, "api", pkg.Service.Group, pkg.Version)
}

func jsonTag(name string) string {
	return fmt.Sprintf(`json:"%s,omitempty"`, name)
}

func tagJSONName(tag string) string {
	parts := strings.Split(tag, `"`)
	if len(parts) < 2 {
		return ""
	}

	return strings.Split(parts[1], ",")[0]
}
