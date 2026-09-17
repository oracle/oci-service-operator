/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"encoding/json"
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

	unsupportedPaths := unsupportedUpdateDriftPathsWithEquivalence(
		specValues,
		currentValues,
		semantics.Mutation,
		c.config.ParityHooks.UnsupportedDriftEquivalent,
	)
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
		if zeroValueNullEquivalent(field, wantedValue, statusValue, c.config.Semantics.Mutation) {
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
	return desiredComparableValueMatches(specValue, currentValue)
}

// desiredComparableValueMatches compares only fields represented by the
// desired CR. OCI responses may add observed-only fields such as generated
// identifiers and timestamps to nested objects. Those fields must not turn a
// converged replacement-only collection into false drift.
func desiredComparableValueMatches(specValue any, currentValue any) bool {
	switch desired := specValue.(type) {
	case map[string]any:
		observed, ok := currentValue.(map[string]any)
		if !ok {
			return false
		}
		for key, desiredChild := range desired {
			observedChild, found := lookupMapKey(observed, key)
			if !found || !desiredComparableValueMatches(desiredChild, observedChild) {
				return false
			}
		}
		return true
	case []any:
		observed, ok := currentValue.([]any)
		if !ok || len(desired) != len(observed) {
			return false
		}
		for index := range desired {
			if !desiredComparableValueMatches(desired[index], observed[index]) {
				return false
			}
		}
		return true
	default:
		return valuesEqual(specValue, currentValue)
	}
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
	case []any:
		pruned := make([]any, 0, len(concrete))
		for _, child := range concrete {
			prunedChild, ok := pruneComparableValue(child)
			if !ok {
				continue
			}
			pruned = append(pruned, prunedChild)
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
		if !forceNewValuesEqual(wantedValue, currentValue) {
			return true, nil
		}
	}

	return false, nil
}

func unsupportedUpdateDriftPaths(specValues map[string]any, currentValues map[string]any, semantics MutationSemantics) []string {
	return unsupportedUpdateDriftPathsWithEquivalence(specValues, currentValues, semantics, nil)
}

func unsupportedUpdateDriftPathsWithEquivalence(
	specValues map[string]any,
	currentValues map[string]any,
	semantics MutationSemantics,
	equivalent UnsupportedDriftEquivalent,
) []string {
	diffPaths := comparableDiffPaths(specValues, currentValues, "")
	unsupported := make([]string, 0, len(diffPaths))
	for _, path := range diffPaths {
		if sdkSerializationFieldPath(path) {
			continue
		}
		if equivalent != nil {
			desired, desiredFound := lookupValueByPath(specValues, path)
			observed, observedFound := lookupValueByPath(currentValues, path)
			if desiredFound && observedFound {
				handled, equal := equivalent(path, desired, observed)
				if handled && equal {
					continue
				}
			}
		}
		switch {
		case zeroValueNullEquivalentDrift(path, specValues, currentValues, semantics):
		case pathCoveredByAny(path, semantics.Mutable):
		case pathCoveredByAny(path, semantics.ForceNew):
		default:
			unsupported = appendUniqueStrings(unsupported, path)
		}
	}
	sort.Strings(unsupported)
	return unsupported
}

func sdkSerializationFieldPath(path string) bool {
	for _, segment := range strings.Split(path, ".") {
		if strings.EqualFold(strings.TrimSpace(segment), "jsonData") {
			return true
		}
	}
	return false
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
	specValue, specMeaningful := pruneComparableValue(specValues[key])
	if !specMeaningful {
		return nil
	}
	currentValue, ok := lookupMapKey(currentValues, key)
	if !ok {
		return nil
	}
	currentValue, _ = pruneComparableValue(currentValue)

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

func zeroValueNullEquivalentDrift(
	path string,
	specValues map[string]any,
	currentValues map[string]any,
	semantics MutationSemantics,
) bool {
	specValue, specFound := lookupValueByPath(specValues, path)
	currentValue, currentFound := lookupValueByPath(currentValues, path)
	if !specFound || !currentFound {
		return false
	}
	return zeroValueNullEquivalent(path, specValue, currentValue, semantics)
}

func zeroValueNullEquivalent(
	path string,
	specValue any,
	currentValue any,
	semantics MutationSemantics,
) bool {
	if !pathCoveredByAny(path, semantics.ZeroValueNullEquivalent) {
		return false
	}
	if _, currentMeaningful := pruneComparableValue(currentValue); currentMeaningful {
		return false
	}
	return zeroOnlyComparableValue(specValue)
}

func zeroOnlyComparableValue(value any) bool {
	switch concrete := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(concrete) == ""
	case bool:
		return !concrete
	case float64:
		return concrete == 0
	case map[string]any:
		for _, child := range concrete {
			if !zeroOnlyComparableValue(child) {
				return false
			}
		}
		return true
	case []any:
		return len(concrete) == 0
	default:
		return false
	}
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
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t != nil && t.Kind() == reflect.Interface {
		return nil
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
		currentValue, currentFound := lookupValueByPath(currentValues, path)
		if currentFound && valuesEqual(specValue, currentValue) {
			continue
		}
		if zeroValueNullEquivalent(path, specValue, currentValue, c.config.Semantics.Mutation) {
			continue
		}
		setValueByPath(body, canonicalValuePath(specValues, path), specValue)
	}
	if len(body) == 0 {
		return nil, false, nil
	}
	if err := includeMandatoryUpdateBodyFields(body, specValues, currentValues, c.config.Update); err != nil {
		return nil, false, err
	}
	preservePolymorphicUpdateDiscriminator(body, specValues, c.config.Update)
	return body, true, nil
}

// includeMandatoryUpdateBodyFields carries required SDK update fields from the
// desired spec or observed OCI state when some mutable field actually drifted.
// OCI update models can require an unchanged identity or configuration value
// alongside changed fields; adding it only after drift detection avoids no-op
// update loops.
func includeMandatoryUpdateBodyFields(body, specValues, currentValues map[string]any, operation *Operation) error {
	if len(body) == 0 || len(specValues) == 0 || operation == nil {
		return nil
	}
	request, ok := operationRequestStruct(operation.NewRequest)
	if !ok {
		return nil
	}
	for _, requestField := range operation.Fields {
		if requestField.Contribution != "body" {
			continue
		}
		bodyField, found := request.Type().FieldByName(requestField.FieldName)
		if !found {
			continue
		}
		paths, err := mandatoryUpdateBodyFieldPaths(bodyField.Type, specValues, currentValues)
		if err != nil {
			return err
		}
		for _, path := range paths {
			if _, exists := lookupValueByPath(body, path); exists {
				continue
			}
			value, exists := lookupMeaningfulValue(specValues, path)
			if !exists {
				value, exists = lookupMeaningfulValue(currentValues, path)
			}
			if !exists {
				continue
			}
			setValueByPath(body, canonicalValuePath(specValues, path), value)
		}
		bodyJSONName := fieldJSONName(bodyField)
		if bodyJSONName == "" {
			bodyJSONName = lowerCamel(bodyField.Name)
		}
		bodyValue, exists := lookupValueByPath(body, bodyJSONName)
		if !exists {
			// A request's body field contains the whole projected update body,
			// rather than adding another JSON nesting level.
			bodyValue = body
		}
		if err := includeMandatoryNestedUpdateFields(bodyValue, specValues, currentValues, bodyField.Type); err != nil {
			return err
		}
	}
	return nil
}

func includeMandatoryNestedUpdateFields(bodyValue, specValue, currentValue any, targetType reflect.Type) error {
	for targetType != nil && targetType.Kind() == reflect.Pointer {
		targetType = targetType.Elem()
	}
	if targetType != nil && targetType.Kind() == reflect.Interface {
		discriminator, ok := polymorphicUpdateDiscriminatorField(targetType)
		if !ok {
			return nil
		}
		value, exists := lookupValueByPath(jsonMap(bodyValue), discriminator)
		if !exists {
			value, exists = lookupValueByPath(jsonMap(specValue), discriminator)
		}
		if !exists {
			value, exists = lookupValueByPath(jsonMap(currentValue), discriminator)
		}
		if !exists {
			return nil
		}
		payload, err := json.Marshal(map[string]any{discriminator: value})
		if err != nil {
			return err
		}
		converted, handled, err := convertPolymorphicInterfaceValue(payload, targetType)
		if err != nil || !handled {
			return err
		}
		targetType = indirectType(reflect.TypeOf(converted.Interface()))
	}
	if targetType == nil {
		return nil
	}

	switch targetType.Kind() {
	case reflect.Struct:
		bodyMap := mutableJSONMap(bodyValue)
		if bodyMap == nil {
			return nil
		}
		specMap := jsonMap(specValue)
		currentMap := jsonMap(currentValue)
		for index := 0; index < targetType.NumField(); index++ {
			field := targetType.Field(index)
			if !field.IsExported() {
				continue
			}
			name := fieldJSONName(field)
			if name == "" {
				name = lowerCamel(field.Name)
			}
			child, exists := lookupMapKey(bodyMap, name)
			if !exists && field.Tag.Get("mandatory") == "true" {
				child, exists = lookupMapKey(specMap, name)
				if !exists {
					child, exists = lookupMapKey(currentMap, name)
				}
				if exists {
					bodyMap[name] = child
				}
			}
			if !exists {
				continue
			}
			specChild, _ := lookupMapKey(specMap, name)
			currentChild, _ := lookupMapKey(currentMap, name)
			if err := includeMandatoryNestedUpdateFields(child, specChild, currentChild, field.Type); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		bodyItems, ok := bodyValue.([]any)
		if !ok {
			return nil
		}
		specItems, _ := specValue.([]any)
		currentItems, _ := currentValue.([]any)
		for index := range bodyItems {
			var specItem, currentItem any
			if index < len(specItems) {
				specItem = specItems[index]
			}
			if index < len(currentItems) {
				currentItem = currentItems[index]
			}
			if err := includeMandatoryNestedUpdateFields(bodyItems[index], specItem, currentItem, targetType.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}

func mutableJSONMap(value any) map[string]any {
	if values, ok := value.(map[string]any); ok {
		return values
	}
	return jsonMap(value)
}

func mandatoryUpdateBodyFieldPaths(targetType reflect.Type, specValues, currentValues map[string]any) ([]string, error) {
	bodyType := indirectType(targetType)
	if bodyType == nil && targetType.Kind() == reflect.Interface {
		discriminator, ok := polymorphicUpdateDiscriminatorField(targetType)
		if ok {
			value, exists := lookupMeaningfulValue(specValues, discriminator)
			if !exists {
				value, exists = lookupMeaningfulValue(currentValues, discriminator)
			}
			if exists {
				payload, err := json.Marshal(map[string]any{discriminator: value})
				if err != nil {
					return nil, fmt.Errorf("marshal %s discriminator for mandatory update fields: %w", discriminator, err)
				}
				converted, handled, err := convertPolymorphicInterfaceValue(payload, targetType)
				if err != nil {
					return nil, err
				}
				if handled {
					bodyType = indirectType(reflect.TypeOf(converted.Interface()))
				}
			}
		}
	}
	if bodyType == nil || bodyType.Kind() != reflect.Struct {
		return nil, nil
	}

	paths := make([]string, 0, bodyType.NumField())
	for index := 0; index < bodyType.NumField(); index++ {
		field := bodyType.Field(index)
		if !field.IsExported() || field.Tag.Get("mandatory") != "true" {
			continue
		}
		path := fieldJSONName(field)
		if path == "" {
			path = lowerCamel(field.Name)
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func preservePolymorphicUpdateDiscriminator(body, specValues map[string]any, operation *Operation) {
	if len(body) == 0 || len(specValues) == 0 || operation == nil || operation.NewRequest == nil {
		return
	}
	request, ok := operationRequestStruct(operation.NewRequest)
	if !ok {
		return
	}
	for _, field := range operation.Fields {
		if field.Contribution != "body" {
			continue
		}
		requestField, found := request.Type().FieldByName(field.FieldName)
		if !found {
			continue
		}
		discriminator, ok := polymorphicUpdateDiscriminatorField(requestField.Type)
		if !ok {
			continue
		}
		if _, exists := lookupMeaningfulValue(body, discriminator); exists {
			return
		}
		if value, exists := lookupMeaningfulValue(specValues, discriminator); exists {
			setValueByPath(body, canonicalValuePath(specValues, discriminator), value)
		}
		return
	}
}

func polymorphicUpdateDiscriminatorField(targetType reflect.Type) (string, bool) {
	switch targetType {
	case autoScalingPolicyUpdateDetailsType:
		return "policyType", true
	case dataIntegrationConnectionUpdateType,
		dataIntegrationDataAssetUpdateType,
		dataIntegrationTaskUpdateType:
		return "modelType", true
	case networkFirewallUpdateAddressListType,
		networkFirewallUpdateApplicationType,
		networkFirewallUpdateDecryptionType,
		networkFirewallUpdateNatRuleType,
		networkFirewallUpdateServiceType:
		return "type", true
	case networkFirewallUpdateMappedSecretType:
		return "source", true
	case networkFirewallUpdateTunnelRuleType:
		return "protocol", true
	default:
		return additionalPolymorphicUpdateDiscriminatorField(targetType)
	}
}
