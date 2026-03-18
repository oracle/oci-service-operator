/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type existingSpecSurface struct {
	SpecComments []string
	SpecFields   []FieldModel
	HelperTypes  []TypeModel
}

func preservePackageSpecSurfaces(root string, pkg *PackageModel) error {
	if strings.TrimSpace(root) == "" || pkg == nil {
		return nil
	}

	resources := make([]ResourceModel, 0, len(pkg.Resources))
	for _, resource := range pkg.Resources {
		path := filepath.Join(root, "api", pkg.Service.Group, pkg.Version, resource.FileStem+"_types.go")
		surface, ok, err := loadExistingSpecSurface(path, resource.Kind)
		if err != nil {
			return err
		}
		if ok {
			resource = overlayExistingSpecSurface(resource, surface)
		}
		resources = append(resources, resource)
	}

	pkg.Resources = assignHelperTypeNames(resources)
	pkg.Resources = assignStatusTypeNames(pkg.Resources)
	pkg.Resources = applyDefaultSamples(pkg.Service, pkg.Version, pkg.Resources)
	return nil
}

func overlayExistingSpecSurface(resource ResourceModel, surface existingSpecSurface) ResourceModel {
	statusFields := append([]FieldModel(nil), resource.StatusFields...)
	statusHelpers := referencedHelperTypes(statusFields, resource.HelperTypes)
	statusFields, helperTypes := mergeSpecAndStatusHelperTypes(surface.HelperTypes, statusFields, statusHelpers)

	resource.SpecComments = append([]string(nil), surface.SpecComments...)
	resource.SpecFields = append([]FieldModel(nil), surface.SpecFields...)
	resource.StatusFields = statusFields
	resource.HelperTypes = helperTypes
	resource.PrimaryDisplayField = inferPrimaryDisplayField(resource.SpecFields)
	resource.PrintColumns = defaultPrintColumns(resource.Kind, resource.PrimaryDisplayField)
	return resource
}

func inferPrimaryDisplayField(fields []FieldModel) string {
	switch {
	case hasField(fields, "DisplayName"):
		return "DisplayName"
	case hasField(fields, "Name"):
		return "Name"
	default:
		return ""
	}
}

func mergeSpecAndStatusHelperTypes(specHelpers []TypeModel, statusFields []FieldModel, statusHelpers []TypeModel) ([]FieldModel, []TypeModel) {
	preserved := append([]TypeModel(nil), specHelpers...)
	if len(statusHelpers) == 0 {
		return statusFields, preserved
	}

	preservedByName := make(map[string]TypeModel, len(preserved))
	usedNames := make(map[string]struct{}, len(preserved))
	for _, helper := range preserved {
		preservedByName[helper.Name] = helper
		usedNames[helper.Name] = struct{}{}
	}

	renames := make(map[string]string)
	mergedStatus := make([]TypeModel, 0, len(statusHelpers))
	for _, helper := range statusHelpers {
		if existing, ok := preservedByName[helper.Name]; ok {
			if reflect.DeepEqual(existing, helper) {
				continue
			}

			renamed := uniqueCompatibilityHelperName(helper.Name, usedNames)
			renames[helper.Name] = renamed
			helper = renameHelperType(helper, renamed)
		}

		usedNames[helper.Name] = struct{}{}
		mergedStatus = append(mergedStatus, helper)
	}

	if len(renames) > 0 {
		statusFields = rewriteFieldTypes(statusFields, renames)
		rewritten := make([]TypeModel, 0, len(mergedStatus))
		for _, helper := range mergedStatus {
			helper.Fields = rewriteFieldTypes(helper.Fields, renames)
			rewritten = append(rewritten, helper)
		}
		mergedStatus = rewritten
	}

	return statusFields, append(preserved, mergedStatus...)
}

func uniqueCompatibilityHelperName(name string, usedNames map[string]struct{}) string {
	candidates := []string{
		name + "ObservedState",
		name + "Status",
	}
	for _, candidate := range candidates {
		if _, exists := usedNames[candidate]; !exists {
			return candidate
		}
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%sObservedState%d", name, i)
		if _, exists := usedNames[candidate]; !exists {
			return candidate
		}
	}
}

func referencedHelperTypes(fields []FieldModel, helperTypes []TypeModel) []TypeModel {
	if len(helperTypes) == 0 {
		return nil
	}

	helperIndex := make(map[string]TypeModel, len(helperTypes))
	order := make([]string, 0, len(helperTypes))
	for _, helper := range helperTypes {
		helperIndex[helper.Name] = helper
		order = append(order, helper.Name)
	}

	seen := make(map[string]struct{}, len(helperTypes))
	for _, field := range fields {
		collectReferencedHelperTypes(field.Type, helperIndex, seen)
	}

	referenced := make([]TypeModel, 0, len(seen))
	for _, name := range order {
		if _, ok := seen[name]; ok {
			referenced = append(referenced, helperIndex[name])
		}
	}
	return referenced
}

func collectReferencedHelperTypes(typeExpr string, helperIndex map[string]TypeModel, seen map[string]struct{}) {
	name := underlyingTypeName(typeExpr)
	helper, ok := helperIndex[name]
	if !ok {
		return
	}
	if _, exists := seen[name]; exists {
		return
	}

	seen[name] = struct{}{}
	for _, field := range helper.Fields {
		collectReferencedHelperTypes(field.Type, helperIndex, seen)
	}
}

func loadExistingSpecSurface(path string, kind string) (existingSpecSurface, bool, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return existingSpecSurface{}, false, nil
	}
	if err != nil {
		return existingSpecSurface{}, false, fmt.Errorf("read %q: %w", path, err)
	}

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, content, parser.ParseComments)
	if err != nil {
		return existingSpecSurface{}, false, fmt.Errorf("parse %q: %w", path, err)
	}

	typeOrder := make([]string, 0)
	typeIndex := make(map[string]TypeModel)
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
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
			typeModel, err := parseExistingTypeModel(fileSet, genDecl, typeSpec, structType)
			if err != nil {
				return existingSpecSurface{}, false, fmt.Errorf("parse struct %q in %s: %w", typeSpec.Name.Name, path, err)
			}
			typeOrder = append(typeOrder, typeModel.Name)
			typeIndex[typeModel.Name] = typeModel
		}
	}

	specTypeName := kind + "Spec"
	specType, ok := typeIndex[specTypeName]
	if !ok {
		return existingSpecSurface{}, false, fmt.Errorf("existing resource file %q does not contain %s", path, specTypeName)
	}

	helperCandidates := make([]TypeModel, 0, len(typeOrder))
	for _, name := range typeOrder {
		if name == specTypeName {
			continue
		}
		helperCandidates = append(helperCandidates, typeIndex[name])
	}

	return existingSpecSurface{
		SpecComments: append([]string(nil), specType.Comments...),
		SpecFields:   append([]FieldModel(nil), specType.Fields...),
		HelperTypes:  referencedHelperTypes(specType.Fields, helperCandidates),
	}, true, nil
}

func parseExistingTypeModel(fileSet *token.FileSet, genDecl *ast.GenDecl, typeSpec *ast.TypeSpec, structType *ast.StructType) (TypeModel, error) {
	fields := make([]FieldModel, 0)
	if structType.Fields != nil {
		for _, field := range structType.Fields.List {
			fieldType, err := renderExistingExpr(fileSet, field.Type)
			if err != nil {
				return TypeModel{}, err
			}

			comments, markers := splitFieldCommentLines(commentGroupLines(field.Doc))
			tag := ""
			if field.Tag != nil {
				tag = strings.Trim(field.Tag.Value, "`")
			}

			if len(field.Names) == 0 {
				fields = append(fields, FieldModel{
					Type:     fieldType,
					Tag:      tag,
					Comments: comments,
					Markers:  markers,
					Embedded: true,
				})
				continue
			}

			for _, name := range field.Names {
				fields = append(fields, FieldModel{
					Name:     name.Name,
					Type:     fieldType,
					Tag:      tag,
					Comments: comments,
					Markers:  markers,
				})
			}
		}
	}

	return TypeModel{
		Name:     typeSpec.Name.Name,
		Comments: commentGroupLines(typeCommentGroup(genDecl, typeSpec)),
		Fields:   fields,
	}, nil
}

func typeCommentGroup(genDecl *ast.GenDecl, typeSpec *ast.TypeSpec) *ast.CommentGroup {
	if typeSpec.Doc != nil {
		return typeSpec.Doc
	}
	return genDecl.Doc
}

func renderExistingExpr(fileSet *token.FileSet, expr ast.Expr) (string, error) {
	var buffer bytes.Buffer
	if err := format.Node(&buffer, fileSet, expr); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func commentGroupLines(group *ast.CommentGroup) []string {
	if group == nil {
		return nil
	}

	lines := make([]string, 0, len(group.List))
	for _, comment := range group.List {
		text := strings.TrimRight(comment.Text, " \t")
		if strings.HasPrefix(text, "//") {
			line := strings.TrimPrefix(text, "//")
			if strings.HasPrefix(line, " ") {
				line = strings.TrimPrefix(line, " ")
			}
			lines = append(lines, line)
			continue
		}
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimRight(line, " \t")
			line = strings.TrimPrefix(line, " ")
			if strings.HasPrefix(line, "*") {
				line = strings.TrimPrefix(line, "*")
				if strings.HasPrefix(line, " ") {
					line = strings.TrimPrefix(line, " ")
				}
			}
			lines = append(lines, line)
		}
	}
	return lines
}

func splitFieldCommentLines(lines []string) ([]string, []string) {
	comments := make([]string, 0, len(lines))
	markers := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "+kubebuilder:") {
			markers = append(markers, line)
			continue
		}
		comments = append(comments, line)
	}
	return comments, markers
}
