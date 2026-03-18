/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"fmt"
	"strings"

	"github.com/oracle/oci-service-operator/internal/ocisdk"
)

type resourceFieldSet struct {
	SpecFields   []FieldModel
	StatusFields []FieldModel
	HelperTypes  []TypeModel
}

func synthesizeResourceFieldSet(index *ocisdk.Package, service ServiceConfig, resourceKind string, rawName string, specCandidates []string) resourceFieldSet {
	synthesizer := newFieldSynthesizer(index, resourceKind)

	specFields, _ := synthesizer.mergeStructFields(specCandidates, nil, fieldRenderingOptions{scope: fieldScopeSpec})

	statusFields := defaultStatusFields()
	statusJSONNames := fieldJSONNames(statusFields)
	observedFields, _ := synthesizer.mergeStructFields(
		service.ObservedStateStructCandidates(rawName),
		nil,
		fieldRenderingOptions{scope: fieldScopeStatus, escapeStatusJSONCollision: true},
	)
	for _, field := range observedFields {
		jsonName := tagJSONName(field.Tag)
		if _, exists := statusJSONNames[jsonName]; exists {
			continue
		}
		statusFields = append(statusFields, field)
		statusJSONNames[jsonName] = struct{}{}
	}

	return resourceFieldSet{
		SpecFields:   specFields,
		StatusFields: statusFields,
		HelperTypes:  append([]TypeModel(nil), synthesizer.helperTypes...),
	}
}

func desiredStateStructCandidates(rawName string, requestBodyPayloads []string) []string {
	return appendUniqueStrings([]string{
		"Create" + rawName + "Details",
		"Update" + rawName + "Details",
	}, requestBodyPayloads...)
}

type fieldSynthesizer struct {
	index               *ocisdk.Package
	resourceKind        string
	helperTypes         []TypeModel
	helperIndex         map[string]int
	normalizedTypeNames map[string]string
}

func newFieldSynthesizer(index *ocisdk.Package, resourceKind string) *fieldSynthesizer {
	return &fieldSynthesizer{
		index:               index,
		resourceKind:        resourceKind,
		helperIndex:         make(map[string]int),
		normalizedTypeNames: buildNormalizedTypeNameIndex(index.TypeNames()),
	}
}

func (s *fieldSynthesizer) mergeStructFields(candidates []string, initial []FieldModel, options fieldRenderingOptions) ([]FieldModel, map[string]struct{}) {
	merged := make([]FieldModel, 0, len(initial)+8)
	seenJSONNames := make(map[string]struct{}, len(initial)+8)
	for _, field := range initial {
		merged = append(merged, field)
		seenJSONNames[tagJSONName(field.Tag)] = struct{}{}
	}

	for _, candidate := range candidates {
		fields, ok := s.candidateFields(candidate)
		if !ok {
			continue
		}
		for _, field := range fields {
			fieldModel, ok := s.buildGeneratedField(field, options, []string{helperPathComponent(field)})
			if !ok {
				continue
			}
			jsonName := tagJSONName(fieldModel.Tag)
			if _, exists := seenJSONNames[jsonName]; exists {
				continue
			}

			merged = append(merged, fieldModel)
			seenJSONNames[jsonName] = struct{}{}
		}
	}

	return merged, seenJSONNames
}

func (s *fieldSynthesizer) candidateFields(typeName string) ([]ocisdk.Field, bool) {
	for _, candidate := range s.typeCandidates(typeName) {
		structDef, ok := s.index.Struct(candidate)
		if ok {
			return structDef.Fields, true
		}

		family, ok := s.index.InterfaceFamily(candidate)
		if !ok {
			continue
		}
		fields := mergeInterfaceFamilyFields(family)
		if len(fields) == 0 {
			continue
		}
		return fields, true
	}
	return nil, false
}

func (s *fieldSynthesizer) buildGeneratedField(field ocisdk.Field, options fieldRenderingOptions, path []string) (FieldModel, bool) {
	renderedType, ok := s.renderFieldType(field, options, path)
	if !ok {
		return FieldModel{}, false
	}

	jsonName := field.JSONName
	if jsonName == "" {
		jsonName = lowerCamel(field.Name)
	}

	fieldModel := buildFieldModel(field, jsonName, options)
	fieldModel.Type = renderedType
	return fieldModel, true
}

func (s *fieldSynthesizer) renderFieldType(field ocisdk.Field, options fieldRenderingOptions, path []string) (string, bool) {
	if sharedType, ok := sharedSchemaType(field); ok {
		return sharedType, true
	}

	switch {
	case field.Kind == ocisdk.FieldKindStruct && len(field.NestedFields) > 0:
		helperTypeName, ok := s.ensureHelperType(path, field.NestedFields, options)
		if !ok {
			return "", false
		}
		return wrapRenderedType(field.Type, helperTypeName)
	case field.Kind == ocisdk.FieldKindInterface:
		family, ok := s.index.InterfaceFamily(underlyingTypeName(field.Type))
		if !ok {
			return "", false
		}
		fields := mergeInterfaceFamilyFields(family)
		if len(fields) == 0 {
			return "", false
		}
		helperTypeName, ok := s.ensureHelperType(path, fields, options)
		if !ok {
			return "", false
		}
		return wrapRenderedType(field.Type, helperTypeName)
	case field.RenderableType != "":
		return field.RenderableType, true
	default:
		return "", false
	}
}

func (s *fieldSynthesizer) ensureHelperType(path []string, fields []ocisdk.Field, options fieldRenderingOptions) (string, bool) {
	typeName := helperTypeName(s.resourceKind, path)
	if index, ok := s.helperIndex[typeName]; ok {
		return s.helperTypes[index].Name, len(s.helperTypes[index].Fields) > 0
	}

	nestedOptions := options
	nestedOptions.escapeStatusJSONCollision = false

	helperFields := make([]FieldModel, 0, len(fields))
	for _, field := range fields {
		fieldModel, ok := s.buildGeneratedField(field, nestedOptions, append(append([]string(nil), path...), helperPathComponent(field)))
		if !ok {
			continue
		}
		helperFields = append(helperFields, fieldModel)
	}
	if len(helperFields) == 0 {
		return "", false
	}

	typeModel := TypeModel{
		Name:     typeName,
		Comments: []string{fmt.Sprintf("%s defines nested fields for %s.", typeName, helperPathLabel(s.resourceKind, path))},
		Fields:   helperFields,
	}
	s.helperIndex[typeName] = len(s.helperTypes)
	s.helperTypes = append(s.helperTypes, typeModel)
	return typeName, true
}

func sharedSchemaType(field ocisdk.Field) (string, bool) {
	if field.Type == "map[string]map[string]interface{}" {
		return "map[string]shared.MapValue", true
	}
	if renderedType, ok := arbitraryJSONSchemaType(field.Type); ok {
		return renderedType, true
	}
	return "", false
}

func arbitraryJSONSchemaType(typeExpr string) (string, bool) {
	trimmed := strings.TrimSpace(typeExpr)
	switch {
	case trimmed == "":
		return "", false
	case trimmed == "interface{}":
		return "shared.JSONValue", true
	case strings.HasPrefix(trimmed, "*"):
		return arbitraryJSONSchemaType(strings.TrimPrefix(trimmed, "*"))
	case strings.HasPrefix(trimmed, "[]"):
		inner, ok := arbitraryJSONSchemaType(strings.TrimPrefix(trimmed, "[]"))
		if !ok {
			return "", false
		}
		return "[]" + inner, true
	case strings.HasPrefix(trimmed, "map[string]"):
		inner, ok := arbitraryJSONSchemaType(strings.TrimPrefix(trimmed, "map[string]"))
		if !ok {
			return "", false
		}
		return "map[string]" + inner, true
	default:
		return "", false
	}
}

func wrapRenderedType(typeExpr string, replacement string) (string, bool) {
	trimmed := strings.TrimSpace(typeExpr)
	switch {
	case trimmed == "":
		return replacement, true
	case strings.HasPrefix(trimmed, "*"):
		return wrapRenderedType(strings.TrimPrefix(trimmed, "*"), replacement)
	case strings.HasPrefix(trimmed, "[]"):
		inner, ok := wrapRenderedType(strings.TrimPrefix(trimmed, "[]"), replacement)
		if !ok {
			return "", false
		}
		return "[]" + inner, true
	case strings.HasPrefix(trimmed, "map[string]"):
		inner, ok := wrapRenderedType(strings.TrimPrefix(trimmed, "map[string]"), replacement)
		if !ok {
			return "", false
		}
		return "map[string]" + inner, true
	default:
		return replacement, true
	}
}

func underlyingTypeName(typeExpr string) string {
	trimmed := strings.TrimSpace(typeExpr)
	switch {
	case trimmed == "":
		return ""
	case strings.HasPrefix(trimmed, "*"):
		return underlyingTypeName(strings.TrimPrefix(trimmed, "*"))
	case strings.HasPrefix(trimmed, "[]"):
		return underlyingTypeName(strings.TrimPrefix(trimmed, "[]"))
	case strings.HasPrefix(trimmed, "map[string]"):
		return underlyingTypeName(strings.TrimPrefix(trimmed, "map[string]"))
	default:
		return trimmed
	}
}

func helperPathComponent(field ocisdk.Field) string {
	if strings.HasPrefix(strings.TrimSpace(field.Type), "[]") {
		return singularize(field.Name)
	}
	return field.Name
}

func helperTypeName(resourceKind string, path []string) string {
	base := strings.Join(path, "")
	if strings.HasPrefix(base, resourceKind) {
		return base
	}
	return resourceKind + base
}

func helperPathLabel(resourceKind string, path []string) string {
	return resourceKind + "." + strings.Join(path, ".")
}

func buildNormalizedTypeNameIndex(typeNames []string) map[string]string {
	index := make(map[string]string, len(typeNames))
	conflicts := make(map[string]struct{})
	for _, typeName := range typeNames {
		key := normalizedTypeKey(typeName)
		if key == "" {
			continue
		}
		if existing, ok := index[key]; ok && existing != typeName {
			conflicts[key] = struct{}{}
			continue
		}
		index[key] = typeName
	}
	for key := range conflicts {
		delete(index, key)
	}
	return index
}

func normalizedTypeKey(typeName string) string {
	return strings.Join(normalizedTokens(typeName), "\x00")
}

func (s *fieldSynthesizer) typeCandidates(typeName string) []string {
	candidates := []string{typeName}
	if matched, ok := s.normalizedTypeNames[normalizedTypeKey(typeName)]; ok && matched != "" && matched != typeName {
		candidates = append(candidates, matched)
	}
	return candidates
}

func mergeInterfaceFamilyFields(family ocisdk.InterfaceFamily) []ocisdk.Field {
	merged := make([]ocisdk.Field, 0, len(family.Base.Fields))
	byJSONName := make(map[string]int, len(family.Base.Fields))

	appendFields := func(fields []ocisdk.Field) {
		for _, field := range fields {
			jsonName := field.JSONName
			if jsonName == "" {
				jsonName = lowerCamel(field.Name)
			}
			index, exists := byJSONName[jsonName]
			if !exists {
				byJSONName[jsonName] = len(merged)
				merged = append(merged, field)
				continue
			}
			merged[index] = mergeInterfaceField(merged[index], field)
		}
	}

	appendFields(family.Base.Fields)
	for _, implementation := range family.Implementations {
		appendFields(implementation.Fields)
	}

	return merged
}

func mergeInterfaceField(existing ocisdk.Field, candidate ocisdk.Field) ocisdk.Field {
	if existing.Type != candidate.Type || existing.Kind != candidate.Kind || existing.RenderableType != candidate.RenderableType {
		return existing
	}

	existing.Mandatory = existing.Mandatory || candidate.Mandatory
	existing.Deprecated = existing.Deprecated || candidate.Deprecated
	existing.ReadOnly = existing.ReadOnly || candidate.ReadOnly
	if existing.JSONName == "" {
		existing.JSONName = candidate.JSONName
	}
	if existing.Documentation == "" {
		existing.Documentation = candidate.Documentation
	}
	if len(existing.NestedFields) == 0 {
		existing.NestedFields = candidate.NestedFields
	}

	return existing
}
