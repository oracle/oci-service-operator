/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-service-operator/pkg/credhelper"
)

// ResolveSpecValue rewrites secret-backed spec inputs and omits zero-value nested
// structs using the same projection rules as generated runtime request builders.
func ResolveSpecValue(resource any, ctx context.Context, credentialClient credhelper.CredentialClient, namespace string) (any, error) {
	return resolvedSpecValue(resource, requestBuildOptions{
		Context:          ctx,
		CredentialClient: credentialClient,
		Namespace:        namespace,
	})
}

// ResolveSpecValueWithBoolFields rewrites secret-backed spec inputs while
// preserving explicit false booleans for request-body projection.
func ResolveSpecValueWithBoolFields(resource any, ctx context.Context, credentialClient credhelper.CredentialClient, namespace string) (any, error) {
	return resolvedSpecValueWithDecoder(resource, requestBuildOptions{
		Context:          ctx,
		CredentialClient: credentialClient,
		Namespace:        namespace,
	}, decodedJSONValueWithBoolFields, true)
}

func resolvedSpecValue(resource any, options requestBuildOptions) (any, error) {
	return resolvedSpecValueWithDecoder(resource, options, decodedJSONValue, false)
}

func resolvedSpecValueWithDecoder(resource any, options requestBuildOptions, decoder func(any) (any, error), preserveZeroStructDecoded bool) (any, error) {
	raw := specValue(resource)
	if raw == nil {
		return nil, nil
	}

	decoded, err := decoder(raw)
	if err != nil {
		return nil, fmt.Errorf("decode spec value: %w", err)
	}

	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return nil, err
	}
	specField, ok := fieldValue(resourceValue, "Spec")
	if !ok {
		return nil, fmt.Errorf("resource %T does not expose Spec", resource)
	}

	resolved, _, err := rewriteSecretSources(specField, decoded, options, preserveZeroStructDecoded)
	if err != nil {
		return nil, err
	}
	return resolved, nil
}

func indirectValue(value reflect.Value) (reflect.Value, bool) {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}
	return value, value.IsValid()
}

func rewriteSecretSources(value reflect.Value, decoded any, options requestBuildOptions, preserveZeroStructDecoded bool) (any, bool, error) {
	value, ok := indirectValue(value)
	if !ok {
		return nil, false, nil
	}
	if rewritten, include, handled, err := rewriteSharedSecretSource(value, options); handled {
		return rewritten, include, err
	}
	if value.Kind() == reflect.Struct && value.IsZero() {
		if !preserveZeroStructDecoded {
			return nil, false, nil
		}
		if decodedMap, ok := decoded.(map[string]any); !ok || !meaningfulValue(decodedMap) {
			return nil, false, nil
		}
	}

	switch value.Kind() {
	case reflect.Struct:
		return rewriteSecretStruct(value, decoded, options, preserveZeroStructDecoded)
	case reflect.Slice, reflect.Array:
		return rewriteSecretSlice(value, decoded, options, preserveZeroStructDecoded)
	case reflect.Map:
		return rewriteSecretMap(value, decoded, options, preserveZeroStructDecoded)
	default:
		return decoded, true, nil
	}
}

func rewriteSharedSecretSource(value reflect.Value, options requestBuildOptions) (any, bool, bool, error) {
	switch value.Type() {
	case passwordSourceType:
		rewritten, include, err := resolveSecretSourceValue(options.Context, options.CredentialClient, options.Namespace, value.FieldByName("Secret"), "SecretName", "password")
		return rewritten, include, true, err
	case usernameSourceType:
		rewritten, include, err := resolveSecretSourceValue(options.Context, options.CredentialClient, options.Namespace, value.FieldByName("Secret"), "SecretName", "username")
		return rewritten, include, true, err
	default:
		return nil, false, false, nil
	}
}

func rewriteSecretStruct(value reflect.Value, decoded any, options requestBuildOptions, preserveZeroStructDecoded bool) (any, bool, error) {
	decodedMap := decodedMapValue(decoded)
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := typ.Field(i)
		if !fieldType.IsExported() {
			continue
		}
		if fieldType.Anonymous && embeddedJSONField(fieldType) {
			var err error
			decodedMap, err = rewriteEmbeddedSecretField(value.Field(i), decodedMap, options, preserveZeroStructDecoded)
			if err != nil {
				return nil, false, err
			}
			continue
		}
		if err := rewriteNamedSecretField(decodedMap, value.Field(i), fieldType, options, preserveZeroStructDecoded); err != nil {
			return nil, false, err
		}
	}
	return decodedMap, true, nil
}

func decodedMapValue(decoded any) map[string]any {
	decodedMap, ok := decoded.(map[string]any)
	if !ok || decodedMap == nil {
		return map[string]any{}
	}
	return decodedMap
}

func rewriteEmbeddedSecretField(fieldValue reflect.Value, decodedMap map[string]any, options requestBuildOptions, preserveZeroStructDecoded bool) (map[string]any, error) {
	rewritten, _, err := rewriteSecretSources(fieldValue, decodedMap, options, preserveZeroStructDecoded)
	if err != nil {
		return nil, err
	}
	if nestedMap, ok := rewritten.(map[string]any); ok {
		return nestedMap, nil
	}
	return decodedMap, nil
}

func rewriteNamedSecretField(decodedMap map[string]any, fieldValue reflect.Value, fieldType reflect.StructField, options requestBuildOptions, preserveZeroStructDecoded bool) error {
	jsonName, skip := fieldJSONTagName(fieldType)
	if skip {
		return nil
	}
	childDecoded, exists := decodedMap[jsonName]
	rewritten, include, err := rewriteSecretSources(fieldValue, childDecoded, options, preserveZeroStructDecoded)
	if err != nil {
		return err
	}
	if include {
		decodedMap[jsonName] = rewritten
		return nil
	}
	if exists {
		delete(decodedMap, jsonName)
	}
	return nil
}

func rewriteSecretSlice(value reflect.Value, decoded any, options requestBuildOptions, preserveZeroStructDecoded bool) (any, bool, error) {
	decodedSlice, ok := decoded.([]any)
	if !ok {
		return decoded, true, nil
	}
	for i := 0; i < value.Len() && i < len(decodedSlice); i++ {
		rewritten, include, err := rewriteSecretSources(value.Index(i), decodedSlice[i], options, preserveZeroStructDecoded)
		if err != nil {
			return nil, false, err
		}
		if include {
			decodedSlice[i] = rewritten
		}
	}
	return decodedSlice, true, nil
}

func rewriteSecretMap(value reflect.Value, decoded any, options requestBuildOptions, preserveZeroStructDecoded bool) (any, bool, error) {
	if value.Type().Key().Kind() != reflect.String {
		return decoded, true, nil
	}
	decodedMap, ok := decoded.(map[string]any)
	if !ok {
		return decoded, true, nil
	}
	iter := value.MapRange()
	for iter.Next() {
		key := iter.Key().String()
		childDecoded, exists := decodedMap[key]
		rewritten, include, err := rewriteSecretSources(iter.Value(), childDecoded, options, preserveZeroStructDecoded)
		if err != nil {
			return nil, false, err
		}
		if include {
			decodedMap[key] = rewritten
			continue
		}
		if exists {
			delete(decodedMap, key)
		}
	}
	return decodedMap, true, nil
}

func resolveSecretSourceValue(
	ctx context.Context,
	credentialClient credhelper.CredentialClient,
	namespace string,
	secretField reflect.Value,
	nameField string,
	dataKey string,
) (any, bool, error) {
	if !secretField.IsValid() {
		return nil, false, nil
	}
	secretNameField := secretField.FieldByName(nameField)
	if !secretNameField.IsValid() || secretNameField.Kind() != reflect.String {
		return nil, false, nil
	}

	secretName := strings.TrimSpace(secretNameField.String())
	if secretName == "" {
		return nil, false, nil
	}
	if credentialClient == nil {
		return nil, false, fmt.Errorf("resolve %s secret %q: credential client is nil", dataKey, secretName)
	}
	if strings.TrimSpace(namespace) == "" {
		return nil, false, fmt.Errorf("resolve %s secret %q: namespace is empty", dataKey, secretName)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	secretData, err := credentialClient.GetSecret(ctx, secretName, namespace)
	if err != nil {
		return nil, false, fmt.Errorf("get %s secret %q: %w", dataKey, secretName, err)
	}
	rawValue, ok := secretData[dataKey]
	if !ok {
		return nil, false, fmt.Errorf("%s key in secret %q is not found", dataKey, secretName)
	}
	return string(rawValue), true, nil
}

func fieldJSONTagName(field reflect.StructField) (string, bool) {
	name := field.Tag.Get("json")
	if name == "-" {
		return "", true
	}
	if strings.TrimSpace(name) == "" {
		return lowerCamel(field.Name), false
	}
	parts := strings.Split(name, ",")
	if len(parts) == 0 || parts[0] == "" {
		return lowerCamel(field.Name), false
	}
	return parts[0], false
}

func embeddedJSONField(field reflect.StructField) bool {
	if !field.Anonymous {
		return false
	}
	parts := strings.Split(field.Tag.Get("json"), ",")
	return len(parts) == 0 || parts[0] == ""
}

func stampSecretSourceStatus(resource any) {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return
	}

	specField, ok := fieldValue(resourceValue, "Spec")
	if !ok {
		return
	}
	statusField, ok := fieldValue(resourceValue, "Status")
	if !ok {
		return
	}

	copySecretSourceFields(specField, statusField)
}

func copySecretSourceFields(source reflect.Value, destination reflect.Value) {
	source, destination, ok := secretSourceStructPair(source, destination)
	if !ok {
		return
	}
	for i := 0; i < source.NumField(); i++ {
		copySecretSourceField(source, destination, i)
	}
}

func secretSourceStructPair(source reflect.Value, destination reflect.Value) (reflect.Value, reflect.Value, bool) {
	var ok bool
	source, ok = indirectValue(source)
	if !ok || source.Kind() != reflect.Struct {
		return reflect.Value{}, reflect.Value{}, false
	}
	destination, ok = indirectValue(destination)
	if !ok || destination.Kind() != reflect.Struct {
		return reflect.Value{}, reflect.Value{}, false
	}
	return source, destination, true
}

func copySecretSourceField(source reflect.Value, destination reflect.Value, index int) {
	fieldType := source.Type().Field(index)
	if !fieldType.IsExported() {
		return
	}

	sourceField := source.Field(index)
	destinationField, ok := settableFieldByName(destination, fieldType.Name)
	if !ok {
		return
	}
	if copySecretSourceLeaf(sourceField, destinationField) {
		return
	}
	copySecretSourceFields(sourceField, destinationField)
}

func settableFieldByName(value reflect.Value, name string) (reflect.Value, bool) {
	field := value.FieldByName(name)
	if !field.IsValid() || !field.CanSet() {
		return reflect.Value{}, false
	}
	return field, true
}

func copySecretSourceLeaf(source reflect.Value, destination reflect.Value) bool {
	if !isSecretSourceType(source.Type()) || destination.Type() != source.Type() {
		return false
	}
	if secretSourceValueIsEmpty(source) {
		destination.Set(reflect.Zero(destination.Type()))
		return true
	}
	destination.Set(source)
	return true
}

func secretSourceValueIsEmpty(value reflect.Value) bool {
	var ok bool
	value, ok = indirectValue(value)
	if !ok || value.Kind() != reflect.Struct {
		return true
	}
	secretField := value.FieldByName("Secret")
	secretField, ok = indirectValue(secretField)
	if !ok || secretField.Kind() != reflect.Struct {
		return true
	}
	nameField := secretField.FieldByName("SecretName")
	if !nameField.IsValid() || nameField.Kind() != reflect.String {
		return true
	}
	return strings.TrimSpace(nameField.String()) == ""
}

func isSecretSourceType(typ reflect.Type) bool {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ == usernameSourceType || typ == passwordSourceType
}
