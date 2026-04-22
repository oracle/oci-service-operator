/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

func (c ServiceClient[T]) shouldInvokeUpdate(ctx context.Context, resource T, namespace string, currentResponse any) (bool, error) {
	if c.config.Update == nil {
		return false, nil
	}
	if c.shouldObserveCurrentLifecycle(currentResponse) {
		return false, nil
	}
	if c.config.BuildUpdateBody != nil {
		_, updateNeeded, err := c.config.BuildUpdateBody(ctx, resource, namespace, currentResponse)
		return updateNeeded, err
	}
	if c.config.Semantics == nil {
		return true, nil
	}
	return c.hasMutableDrift(resource, currentResponse)
}

func (c ServiceClient[T]) shouldObserveCurrentLifecycle(currentResponse any) bool {
	if c.config.Semantics == nil || currentResponse == nil {
		return false
	}

	lifecycleState := strings.ToUpper(responseLifecycleState(currentResponse))
	if lifecycleState == "" {
		return false
	}

	return containsString(c.config.Semantics.Lifecycle.ProvisioningStates, lifecycleState) ||
		containsString(c.config.Semantics.Lifecycle.UpdatingStates, lifecycleState) ||
		containsString(c.config.Semantics.Delete.PendingStates, lifecycleState)
}

func (c ServiceClient[T]) validateMutationPolicy(resource T, existing bool, currentResponse any) error {
	semantics := c.config.Semantics
	if semantics == nil {
		return nil
	}

	specValues, currentValues, err := mutationValues(resource, currentResponse)
	if err != nil {
		return err
	}
	if err := c.validateMutationConflicts(specValues); err != nil {
		return err
	}

	if !existing {
		return nil
	}
	if err := c.validateForceNewFields(resource, specValues, currentValues); err != nil {
		return err
	}
	if err := c.validateCreateOnlyDrift(resource, currentResponse); err != nil {
		return err
	}
	if c.config.Update == nil {
		return nil
	}

	unsupportedPaths := unsupportedUpdateDriftPaths(specValues, currentValues, semantics.Mutation)
	if len(unsupportedPaths) == 0 {
		return nil
	}
	return fmt.Errorf("%s formal semantics reject unsupported update drift for %s", c.config.Kind, strings.Join(unsupportedPaths, ", "))
}

func (c ServiceClient[T]) validateCreateOnlyDrift(resource T, currentResponse any) error {
	if currentResponse == nil || c.config.ParityHooks.ValidateCreateOnlyDrift == nil {
		return nil
	}
	return c.config.ParityHooks.ValidateCreateOnlyDrift(resource, currentResponse)
}

func mutationValues(resource any, currentResponse any) (map[string]any, map[string]any, error) {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return nil, nil, err
	}

	specValues := jsonMap(fieldInterface(resourceValue, "Spec"))
	currentValues := jsonMap(fieldInterface(resourceValue, "Status"))
	if specValues == nil {
		specValues = map[string]any{}
	}
	if currentValues == nil {
		currentValues = map[string]any{}
	}
	if body, ok := responseBody(currentResponse); ok && body != nil {
		mergeJSONMapOverwrite(currentValues, body)
	}
	return specValues, currentValues, nil
}

func (c ServiceClient[T]) validateMutationConflicts(specValues map[string]any) error {
	for field, conflicts := range c.config.Semantics.Mutation.ConflictsWith {
		if _, ok := lookupMeaningfulValue(specValues, field); !ok {
			continue
		}
		for _, conflict := range conflicts {
			if _, ok := lookupMeaningfulValue(specValues, conflict); ok {
				return fmt.Errorf("%s formal semantics forbid setting %s with %s", c.config.Kind, field, conflict)
			}
		}
	}
	return nil
}

func (c ServiceClient[T]) validateForceNewFields(resource T, specValues map[string]any, currentValues map[string]any) error {
	for _, field := range c.config.Semantics.Mutation.ForceNew {
		wantedValue, specOK := lookupMeaningfulValue(specValues, field)
		if !specOK {
			var err error
			wantedValue, specOK, err = meaningfulMutationValueByPath(specValue(resource), field)
			if err != nil {
				return err
			}
		}
		statusValue, statusOK := lookupValueByPath(currentValues, field)
		if !specOK || !statusOK {
			continue
		}
		if !forceNewValuesEqual(wantedValue, statusValue) {
			return fmt.Errorf("%s formal semantics require replacement when %s changes", c.config.Kind, field)
		}
	}
	return nil
}

func forceNewValuesEqual(specValue any, currentValue any) bool {
	specValue, specMeaningful := pruneComparableValue(specValue)
	currentValue, currentMeaningful := pruneComparableValue(currentValue)
	if !specMeaningful || !currentMeaningful {
		return !specMeaningful && !currentMeaningful
	}

	specMap, specIsMap := specValue.(map[string]any)
	currentMap, currentIsMap := currentValue.(map[string]any)
	if specIsMap && currentIsMap {
		return len(comparableDiffPaths(specMap, currentMap, "")) == 0
	}
	return valuesEqual(specValue, currentValue)
}

func pruneComparableValue(value any) (any, bool) {
	switch concrete := value.(type) {
	case map[string]any:
		pruned := make(map[string]any, len(concrete))
		for key, child := range concrete {
			prunedChild, ok := pruneComparableValue(child)
			if !ok {
				continue
			}
			pruned[key] = prunedChild
		}
		if len(pruned) == 0 {
			return nil, false
		}
		return pruned, true
	default:
		if !meaningfulValue(concrete) {
			return nil, false
		}
		return concrete, true
	}
}

func (c ServiceClient[T]) hasMutableDrift(resource T, currentResponse any) (bool, error) {
	semantics := c.config.Semantics
	if semantics == nil || len(semantics.Mutation.Mutable) == 0 {
		return false, nil
	}

	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return false, err
	}

	specValues := jsonMap(fieldInterface(resourceValue, "Spec"))
	currentValues := jsonMap(fieldInterface(resourceValue, "Status"))
	if body, ok := responseBody(currentResponse); ok && body != nil {
		currentValues = jsonMap(body)
	}

	for _, field := range semantics.Mutation.Mutable {
		wantedValue, specFound := lookupMeaningfulValue(specValues, field)
		if !specFound {
			wantedValue, specFound, err = meaningfulMutationValueByPath(fieldInterface(resourceValue, "Spec"), field)
			if err != nil {
				return false, err
			}
		}
		if !specFound {
			continue
		}
		currentValue, currentFound := lookupMeaningfulValue(currentValues, field)
		if !currentFound {
			if responseExposesFieldPath(currentResponse, field) {
				return true, nil
			}
			continue
		}
		if !valuesEqual(wantedValue, currentValue) {
			return true, nil
		}
	}

	return false, nil
}

func unsupportedUpdateDriftPaths(specValues map[string]any, currentValues map[string]any, semantics MutationSemantics) []string {
	diffPaths := comparableDiffPaths(specValues, currentValues, "")
	unsupported := make([]string, 0, len(diffPaths))
	for _, path := range diffPaths {
		switch {
		case pathCoveredByAny(path, semantics.Mutable):
		case pathCoveredByAny(path, semantics.ForceNew):
		default:
			unsupported = appendUniqueStrings(unsupported, path)
		}
	}
	sort.Strings(unsupported)
	return unsupported
}

func comparableDiffPaths(specValues map[string]any, currentValues map[string]any, prefix string) []string {
	if specValues == nil || currentValues == nil {
		return nil
	}

	keys := meaningfulSortedKeys(specValues)
	paths := make([]string, 0, len(keys))
	for _, key := range keys {
		paths = append(paths, comparableDiffPathsForKey(specValues, currentValues, prefix, key)...)
	}

	return paths
}

func meaningfulSortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if _, ok := pruneComparableValue(value); ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func comparableDiffPathsForKey(specValues map[string]any, currentValues map[string]any, prefix string, key string) []string {
	specValue := specValues[key]
	currentValue, ok := lookupMapKey(currentValues, key)
	if !ok {
		return nil
	}

	path := key
	if prefix != "" {
		path = prefix + "." + key
	}

	specMap, specIsMap := specValue.(map[string]any)
	currentMap, currentIsMap := currentValue.(map[string]any)
	if specIsMap && currentIsMap {
		return comparableDiffPaths(specMap, currentMap, path)
	}
	if !valuesEqual(specValue, currentValue) {
		return []string{path}
	}
	return nil
}

func pathCoveredByAny(path string, semanticPaths []string) bool {
	for _, semanticPath := range semanticPaths {
		if pathCoveredBy(path, semanticPath) {
			return true
		}
	}
	return false
}

func pathCoveredBy(path string, semanticPath string) bool {
	path = normalizePath(path)
	semanticPath = normalizePath(semanticPath)
	if path == "" || semanticPath == "" {
		return false
	}
	return path == semanticPath ||
		strings.HasPrefix(path, semanticPath+".") ||
		strings.HasPrefix(semanticPath, path+".")
}

func normalizePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}

	segments := strings.Split(path, ".")
	for index, segment := range segments {
		segments[index] = normalizePathSegment(segment)
	}
	return strings.Join(segments, ".")
}

func normalizePathSegment(segment string) string {
	segment = strings.ToLower(strings.TrimSpace(segment))
	if strings.HasSuffix(segment, "gbs") {
		return strings.TrimSuffix(segment, "gbs") + "gb"
	}
	return segment
}

func responseExposesFieldPath(response any, path string) bool {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return false
	}
	return typeExposesFieldPath(reflect.TypeOf(body), strings.Split(strings.TrimSpace(path), "."))
}

func typeExposesFieldPath(t reflect.Type, segments []string) bool {
	t = indirectType(t)
	if t == nil || len(segments) == 0 {
		return false
	}

	segment := strings.TrimSpace(segments[0])
	if segment == "" {
		return false
	}

	switch t.Kind() {
	case reflect.Struct:
		fieldType, ok := structFieldTypeByPathSegment(t, segment)
		if !ok {
			return false
		}
		if len(segments) == 1 {
			return true
		}
		return typeExposesFieldPath(fieldType, segments[1:])
	case reflect.Map:
		if len(segments) == 1 {
			return true
		}
		return typeExposesFieldPath(t.Elem(), segments[1:])
	case reflect.Slice, reflect.Array:
		return typeExposesFieldPath(t.Elem(), segments)
	default:
		return len(segments) == 1
	}
}

func indirectType(t reflect.Type) reflect.Type {
	for t != nil && (t.Kind() == reflect.Pointer || t.Kind() == reflect.Interface) {
		t = t.Elem()
	}
	return t
}

func structFieldTypeByPathSegment(t reflect.Type, segment string) (reflect.Type, bool) {
	normalized := normalizePathSegment(segment)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		if normalizePathSegment(field.Name) == normalized || normalizePathSegment(fieldJSONName(field)) == normalized {
			return field.Type, true
		}
	}
	return nil, false
}

func (c ServiceClient[T]) filteredUpdateBody(resource T, options requestBuildOptions) (any, bool, error) {
	if c.config.Update == nil || c.config.Semantics == nil || len(c.config.Semantics.Mutation.Mutable) == 0 {
		return nil, false, nil
	}

	resolvedSpec, err := resolvedSpecValueWithDecoder(resource, options, decodedJSONValueWithBoolFields, true)
	if err != nil {
		return nil, false, err
	}
	specValues := jsonMap(resolvedSpec)
	if len(specValues) == 0 {
		return nil, false, nil
	}

	currentValues := map[string]any{}
	if options.CurrentResponse != nil {
		if body, ok := responseBody(options.CurrentResponse); ok && body != nil {
			currentValues, err = mutationJSONMap(body)
			if err != nil {
				return nil, false, err
			}
		}
	}
	if len(currentValues) == 0 {
		statusValue, err := statusStruct(resource)
		if err != nil {
			return nil, false, err
		}
		currentValues, err = mutationJSONMap(statusValue.Interface())
		if err != nil {
			return nil, false, err
		}
	}

	body := make(map[string]any)
	for _, path := range c.config.Semantics.Mutation.Mutable {
		specValue, ok := lookupMeaningfulValue(specValues, path)
		if !ok {
			continue
		}
		if currentValue, currentFound := lookupValueByPath(currentValues, path); currentFound && valuesEqual(specValue, currentValue) {
			continue
		}
		setValueByPath(body, canonicalValuePath(specValues, path), specValue)
	}
	if len(body) == 0 {
		return nil, false, nil
	}
	return body, true, nil
}
