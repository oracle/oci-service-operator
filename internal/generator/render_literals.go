/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"fmt"
	"sort"
	"strings"
)

func goStringSliceLiteral(values []string) string {
	if len(values) == 0 {
		return "[]string{}"
	}

	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return "[]string{" + strings.Join(quoted, ", ") + "}"
}

func goStringSliceMapLiteral(values map[string][]string) string {
	if len(values) == 0 {
		return "map[string][]string{}"
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	entries := make([]string, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, fmt.Sprintf("%q: %s", key, goStringSliceLiteral(values[key])))
	}
	return "map[string][]string{" + strings.Join(entries, ", ") + "}"
}

func requestFieldsLiteral(fields []RuntimeRequestFieldModel) string {
	if len(fields) == 0 {
		return "[]generatedruntime.RequestField{}"
	}

	rendered := make([]string, 0, len(fields))
	for _, field := range fields {
		rendered = append(rendered, fmt.Sprintf(
			`{FieldName: %q, RequestName: %q, Contribution: %q, PreferResourceID: %t}`,
			field.FieldName,
			field.RequestName,
			field.Contribution,
			field.PreferResourceID,
		))
	}
	return "[]generatedruntime.RequestField{" + strings.Join(rendered, ", ") + "}"
}

func runtimeHooksLiteral(hooks []RuntimeHookModel) string {
	if len(hooks) == 0 {
		return "[]generatedruntime.Hook{}"
	}

	rendered := make([]string, 0, len(hooks))
	for _, hook := range hooks {
		rendered = append(rendered, fmt.Sprintf(
			`{Helper: %q, EntityType: %q, Action: %q}`,
			hook.Helper,
			hook.EntityType,
			hook.Action,
		))
	}
	return "[]generatedruntime.Hook{" + strings.Join(rendered, ", ") + "}"
}

func runtimeAuxiliaryOperationsLiteral(operations []RuntimeAuxiliaryOperationModel) string {
	if len(operations) == 0 {
		return "[]generatedruntime.AuxiliaryOperation{}"
	}

	rendered := make([]string, 0, len(operations))
	for _, operation := range operations {
		rendered = append(rendered, fmt.Sprintf(
			`{Phase: %q, MethodName: %q, RequestTypeName: %q, ResponseTypeName: %q}`,
			operation.Phase,
			operation.MethodName,
			operation.RequestTypeName,
			operation.ResponseTypeName,
		))
	}
	return "[]generatedruntime.AuxiliaryOperation{" + strings.Join(rendered, ", ") + "}"
}

func runtimeGapsLiteral(gaps []RuntimeGapModel) string {
	if len(gaps) == 0 {
		return "[]generatedruntime.UnsupportedSemantic{}"
	}

	rendered := make([]string, 0, len(gaps))
	for _, gap := range gaps {
		rendered = append(rendered, fmt.Sprintf(
			`{Category: %q, StopCondition: %q}`,
			gap.Category,
			gap.StopCondition,
		))
	}
	return "[]generatedruntime.UnsupportedSemantic{" + strings.Join(rendered, ", ") + "}"
}
