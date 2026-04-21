/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	databasesdk "github.com/oracle/oci-go-sdk/v65/database"
	databasetoolssdk "github.com/oracle/oci-go-sdk/v65/databasetools"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
)

func (c ServiceClient[T]) requestBuildOptions(ctx context.Context, namespace string) requestBuildOptions {
	return requestBuildOptions{
		Context:          ctx,
		CredentialClient: c.config.CredentialClient,
		Namespace:        namespace,
	}
}

func (c ServiceClient[T]) invoke(ctx context.Context, op *Operation, resource T, preferredID string, options requestBuildOptions) (any, error) {
	values, err := lookupValues(resource)
	if err != nil {
		return nil, err
	}
	return c.invokeWithValues(ctx, op, resource, values, preferredID, options)
}

func (c ServiceClient[T]) invokeWithValues(ctx context.Context, op *Operation, resource T, values map[string]any, preferredID string, options requestBuildOptions) (any, error) {
	if op == nil {
		return nil, fmt.Errorf("%s generated runtime does not define this OCI operation", c.config.Kind)
	}
	if op.NewRequest == nil || op.Call == nil {
		return nil, fmt.Errorf("%s generated runtime OCI operation is incomplete", c.config.Kind)
	}

	request := op.NewRequest()
	if request == nil {
		return nil, fmt.Errorf("%s generated runtime did not create an OCI request value", c.config.Kind)
	}
	bodyOverride, hasBodyOverride, err := c.requestBodyOverride(op, resource, options)
	if err != nil {
		return nil, err
	}
	if err := buildRequest(request, resource, values, preferredID, op.Fields, c.idFieldAliases(), options, bodyOverride, hasBodyOverride); err != nil {
		return nil, fmt.Errorf("build %s OCI request: %w", c.config.Kind, err)
	}

	response, err := op.Call(ctx, request)
	if err != nil {
		return nil, normalizeOCIError(err)
	}
	return response, nil
}

func (c ServiceClient[T]) requestBodyOverride(op *Operation, resource T, options requestBuildOptions) (any, bool, error) {
	if op == c.config.Create && c.config.BuildCreateBody != nil {
		body, err := c.config.BuildCreateBody(options.Context, resource, options.Namespace)
		if err != nil {
			return nil, false, fmt.Errorf("build %s create body: %w", c.config.Kind, err)
		}
		return body, true, nil
	}
	if op == c.config.Update {
		if c.config.BuildUpdateBody != nil {
			body, ok, err := c.config.BuildUpdateBody(options.Context, resource, options.Namespace, options.CurrentResponse)
			if err != nil {
				return nil, false, fmt.Errorf("build %s update body: %w", c.config.Kind, err)
			}
			if ok {
				return body, true, nil
			}
			return nil, false, nil
		}
		body, ok, err := c.filteredUpdateBody(resource, options)
		if err != nil {
			return nil, false, fmt.Errorf("build %s update body: %w", c.config.Kind, err)
		}
		if ok {
			return body, true, nil
		}
	}
	return nil, false, nil
}

type requestBuildOptions struct {
	Context          context.Context
	CredentialClient credhelper.CredentialClient
	Namespace        string
	CurrentResponse  any
}

func buildRequest(
	request any,
	resource any,
	values map[string]any,
	preferredID string,
	fields []RequestField,
	idAliases []string,
	options requestBuildOptions,
	bodyOverride any,
	hasBodyOverride bool,
) error {
	requestValue := reflect.ValueOf(request)
	if !requestValue.IsValid() || requestValue.Kind() != reflect.Pointer || requestValue.IsNil() {
		return fmt.Errorf("expected pointer OCI request, got %T", request)
	}

	requestStruct := requestValue.Elem()
	if requestStruct.Kind() != reflect.Struct {
		return fmt.Errorf("expected pointer to OCI request struct, got %T", request)
	}

	var resolvedSpec any
	switch {
	case hasBodyOverride:
		resolvedSpec = bodyOverride
	case requestNeedsResolvedSpec(fields, requestStruct.Type()):
		var err error
		resolvedSpec, err = resolvedSpecValue(resource, options)
		if err != nil {
			return err
		}
	}

	if len(fields) > 0 {
		if err := buildExplicitRequest(requestStruct, values, preferredID, fields, resolvedSpec); err != nil {
			return err
		}
		assignDeterministicRetryToken(requestStruct, resource)
		return nil
	}

	if err := buildHeuristicRequest(requestStruct, requestStruct.Type(), values, preferredID, idAliases, resolvedSpec); err != nil {
		return err
	}
	assignDeterministicRetryToken(requestStruct, resource)
	return nil
}

func buildExplicitRequest(requestStruct reflect.Value, values map[string]any, preferredID string, fields []RequestField, resolvedSpec any) error {
	for _, field := range fields {
		fieldValue := requestStruct.FieldByName(field.FieldName)
		if !fieldValue.IsValid() || !fieldValue.CanSet() {
			continue
		}

		switch field.Contribution {
		case "header", "binary":
			continue
		case "body":
			if err := assignField(fieldValue, resolvedSpec); err != nil {
				return fmt.Errorf("set body field %s: %w", field.FieldName, err)
			}
			continue
		}

		rawValue, ok := explicitRequestValue(values, field, preferredID)
		if !ok {
			continue
		}
		if err := assignField(fieldValue, rawValue); err != nil {
			return fmt.Errorf("set request field %s: %w", field.FieldName, err)
		}
	}

	return nil
}

func buildHeuristicRequest(
	requestStruct reflect.Value,
	requestType reflect.Type,
	values map[string]any,
	preferredID string,
	idAliases []string,
	resolvedSpec any,
) error {
	for i := 0; i < requestStruct.NumField(); i++ {
		if err := populateHeuristicRequestField(requestStruct.Field(i), requestType.Field(i), values, preferredID, idAliases, resolvedSpec); err != nil {
			return err
		}
	}

	return nil
}

func operationRequestStruct(newRequest func() any) (reflect.Value, bool) {
	if newRequest == nil {
		return reflect.Value{}, false
	}

	request := newRequest()
	if request == nil {
		return reflect.Value{}, false
	}

	requestValue := reflect.ValueOf(request)
	if !requestValue.IsValid() || requestValue.Kind() != reflect.Pointer || requestValue.IsNil() {
		return reflect.Value{}, false
	}

	requestStruct := requestValue.Elem()
	if requestStruct.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	return requestStruct, true
}

func requestLookupKey(fieldType reflect.StructField) string {
	lookupKey := fieldType.Tag.Get("name")
	if lookupKey == "" {
		lookupKey = fieldJSONName(fieldType)
	}
	if lookupKey == "" {
		lookupKey = lowerCamel(fieldType.Name)
	}
	return lookupKey
}

func populateHeuristicRequestField(fieldValue reflect.Value, fieldType reflect.StructField, values map[string]any, preferredID string, idAliases []string, resolvedSpec any) error {
	if !fieldValue.CanSet() || fieldType.Name == "RequestMetadata" {
		return nil
	}

	switch fieldType.Tag.Get("contributesTo") {
	case "header", "binary":
		return nil
	case "body":
		if err := assignField(fieldValue, resolvedSpec); err != nil {
			return fmt.Errorf("set body field %s: %w", fieldType.Name, err)
		}
		return nil
	}

	rawValue, ok := heuristicRequestValue(values, fieldType, preferredID, idAliases)
	if !ok {
		return nil
	}
	if err := assignField(fieldValue, rawValue); err != nil {
		return fmt.Errorf("set request field %s: %w", fieldType.Name, err)
	}
	return nil
}

func heuristicRequestValue(values map[string]any, fieldType reflect.StructField, preferredID string, idAliases []string) (any, bool) {
	lookupKey := requestLookupKey(fieldType)
	if lookupKey == "namespaceName" {
		if value, ok := lookupValueByPaths(values, "namespace"); ok {
			return value, true
		}
		if value, ok := lookupValueByPaths(values, "namespaceName"); ok {
			return value, true
		}
		return nil, false
	}
	if rawValue, ok := lookupValueByPaths(values, lookupKey); ok {
		return rawValue, true
	}
	if preferredID != "" && containsString(idAliases, lookupKey) {
		return preferredID, true
	}
	switch lookupKey {
	case "name":
		return lookupValueByPaths(values, "metadataName")
	default:
		return nil, false
	}
}

func explicitRequestValue(values map[string]any, field RequestField, preferredID string) (any, bool) {
	if field.PreferResourceID {
		if preferredID != "" {
			return preferredID, true
		}
		if currentID, ok := lookupValueByPaths(values, "id", "ocid"); ok {
			return currentID, true
		}
		if len(field.LookupPaths) != 0 {
			if rawValue, ok := lookupValueByPaths(values, field.LookupPaths...); ok {
				return rawValue, true
			}
		}
		return nil, false
	}

	lookupKey := strings.TrimSpace(field.RequestName)
	if lookupKey == "" {
		lookupKey = lowerCamel(field.FieldName)
	}

	if len(field.LookupPaths) != 0 {
		if rawValue, ok := lookupValueByPaths(values, field.LookupPaths...); ok {
			return rawValue, true
		}
	}
	if lookupKey == "namespaceName" {
		if value, ok := lookupValueByPaths(values, "namespace"); ok {
			return value, true
		}
		if value, ok := lookupValueByPaths(values, "namespaceName"); ok {
			return value, true
		}
		return nil, false
	}

	if rawValue, ok := lookupValueByPaths(values, lookupKey); ok {
		return rawValue, true
	}
	if lookupKey == "name" {
		return lookupValueByPaths(values, "metadataName")
	}

	return nil, false
}

func setValueByPath(values map[string]any, path string, value any) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}

	segments := strings.Split(path, ".")
	current := values
	for index, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return
		}
		if index == len(segments)-1 {
			current[segment] = value
			return
		}
		next, ok := current[segment].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[segment] = next
		}
		current = next
	}
}

func canonicalValuePath(values map[string]any, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	segments := strings.Split(path, ".")
	resolved := make([]string, 0, len(segments))
	current := values
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return strings.Join(resolved, ".")
		}
		key := canonicalMapKey(current, segment)
		resolved = append(resolved, key)

		next, ok := current[key].(map[string]any)
		if !ok {
			current = nil
			continue
		}
		current = next
	}
	return strings.Join(resolved, ".")
}

func canonicalMapKey(values map[string]any, segment string) string {
	if values == nil {
		return segment
	}
	normalized := normalizePathSegment(segment)
	for key := range values {
		if normalizePathSegment(key) == normalized {
			return key
		}
	}
	return segment
}

func requestFieldRequiresResourceID(field RequestField, idAliases []string) bool {
	if field.PreferResourceID {
		return true
	}

	lookupKey := strings.TrimSpace(field.RequestName)
	if lookupKey == "" {
		lookupKey = lowerCamel(field.FieldName)
	}
	return containsString(idAliases, lookupKey)
}

func lookupValues(resource any) (map[string]any, error) {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return nil, err
	}

	values := make(map[string]any)
	specValue := fieldInterface(resourceValue, "Spec")
	if specRoot := jsonMap(specValue); specRoot != nil {
		values[lookupSpecRootKey] = specRoot
	}
	mergeJSONMap(values, specValue)
	statusValue := fieldInterface(resourceValue, "Status")
	if statusRoot := jsonMap(statusValue); statusRoot != nil {
		values[lookupStatusRootKey] = statusRoot
	}
	mergeJSONMap(values, statusValue)
	if statusField, ok := fieldValue(resourceValue, "Status"); ok {
		mergeJSONMap(values, fieldInterface(statusField, "OsokStatus"))
	}

	if metadataName := lookupMetadataString(resourceValue, "Name"); metadataName != "" {
		if _, exists := values["name"]; !exists {
			values["name"] = metadataName
		}
		values["metadataName"] = metadataName
	}
	if namespaceName := lookupMetadataString(resourceValue, "Namespace"); namespaceName != "" {
		if _, exists := values["namespaceName"]; !exists {
			values["namespaceName"] = namespaceName
		}
		if _, exists := values["namespace"]; !exists {
			values["namespace"] = namespaceName
		}
	}

	return values, nil
}

func mergeJSONMap(dst map[string]any, source any) {
	if source == nil {
		return
	}
	payload, err := json.Marshal(source)
	if err != nil {
		return
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return
	}
	for key, value := range decoded {
		if _, exists := dst[key]; exists {
			continue
		}
		dst[key] = value
	}
}

func mergeJSONMapOverwrite(dst map[string]any, source any) {
	if source == nil {
		return
	}
	payload, err := json.Marshal(source)
	if err != nil {
		return
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return
	}
	for key, value := range decoded {
		dst[key] = value
	}
}

func specValue(resource any) any {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return nil
	}
	return fieldInterface(resourceValue, "Spec")
}

func requestNeedsResolvedSpec(fields []RequestField, requestType reflect.Type) bool {
	if len(fields) > 0 {
		for _, field := range fields {
			if field.Contribution == "body" {
				return true
			}
		}
		return false
	}

	for i := 0; i < requestType.NumField(); i++ {
		if requestType.Field(i).Tag.Get("contributesTo") == "body" {
			return true
		}
	}
	return false
}

func assignField(field reflect.Value, raw any) error {
	converted, err := convertValue(raw, field.Type())
	if err != nil {
		return err
	}
	field.Set(converted)
	return nil
}

func convertValue(raw any, targetType reflect.Type) (reflect.Value, error) {
	if raw == nil {
		return reflect.Zero(targetType), nil
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("marshal source value: %w", err)
	}
	if targetType.Kind() == reflect.Interface {
		if converted, ok, err := convertPolymorphicInterfaceValue(payload, targetType); ok {
			return converted, err
		}
	}
	converted := reflect.New(targetType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return reflect.Value{}, fmt.Errorf("unmarshal into %s: %w", targetType, err)
	}
	return converted.Elem(), nil
}

func convertPolymorphicInterfaceValue(payload []byte, targetType reflect.Type) (reflect.Value, bool, error) {
	switch targetType {
	case autonomousDatabaseBaseType:
		body, err := convertAutonomousDatabaseBase(payload)
		if err != nil {
			return reflect.Value{}, true, err
		}
		converted := reflect.New(targetType).Elem()
		converted.Set(reflect.ValueOf(body))
		return converted, true, nil
	case databaseToolsConnectionCreateDetailsType:
		body, err := convertDatabaseToolsConnectionCreateDetails(payload)
		if err != nil {
			return reflect.Value{}, true, err
		}
		converted := reflect.New(targetType).Elem()
		converted.Set(reflect.ValueOf(body))
		return converted, true, nil
	case databaseToolsConnectionUpdateDetailsType:
		body, err := convertDatabaseToolsConnectionUpdateDetails(payload)
		if err != nil {
			return reflect.Value{}, true, err
		}
		converted := reflect.New(targetType).Elem()
		converted.Set(reflect.ValueOf(body))
		return converted, true, nil
	default:
		return reflect.Value{}, false, nil
	}
}

// OCI models CreateAutonomousDatabase with a polymorphic interface body. Resolve the CR spec into
// the matching concrete SDK type so request serialization uses the provider model instead of map[string]any.
//
//nolint:gocognit,gocyclo // The source discriminator maps to several concrete SDK request bodies in one switch.
func convertAutonomousDatabaseBase(payload []byte) (databasesdk.CreateAutonomousDatabaseBase, error) {
	source, err := jsonFieldString(payload, "source")
	if err != nil {
		return nil, fmt.Errorf("decode autonomous database source: %w", err)
	}

	concreteType, err := autonomousDatabaseBaseConcreteType(source)
	if err != nil {
		return nil, err
	}

	converted := reflect.New(concreteType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return nil, fmt.Errorf("unmarshal into %s: %w", concreteType, err)
	}
	body, ok := converted.Elem().Interface().(databasesdk.CreateAutonomousDatabaseBase)
	if !ok {
		return nil, fmt.Errorf("resolved CreateAutonomousDatabaseBase type %s does not implement the polymorphic interface", concreteType)
	}
	return body, nil
}

func autonomousDatabaseBaseConcreteType(source string) (reflect.Type, error) {
	switch strings.ToUpper(strings.TrimSpace(source)) {
	case "", "NONE":
		return reflect.TypeOf(databasesdk.CreateAutonomousDatabaseDetails{}), nil
	case "DATABASE":
		return reflect.TypeOf(databasesdk.CreateAutonomousDatabaseCloneDetails{}), nil
	case "CLONE_TO_REFRESHABLE":
		return reflect.TypeOf(databasesdk.CreateRefreshableAutonomousDatabaseCloneDetails{}), nil
	case "BACKUP_FROM_ID":
		return reflect.TypeOf(databasesdk.CreateAutonomousDatabaseFromBackupDetails{}), nil
	case "BACKUP_FROM_TIMESTAMP":
		return reflect.TypeOf(databasesdk.CreateAutonomousDatabaseFromBackupTimestampDetails{}), nil
	case "CROSS_REGION_DISASTER_RECOVERY":
		return reflect.TypeOf(databasesdk.CreateCrossRegionDisasterRecoveryDetails{}), nil
	case "CROSS_REGION_DATAGUARD":
		return reflect.TypeOf(databasesdk.CreateCrossRegionAutonomousDatabaseDataGuardDetails{}), nil
	default:
		return nil, fmt.Errorf("unsupported CreateAutonomousDatabaseBase source %q", source)
	}
}

func convertDatabaseToolsConnectionCreateDetails(payload []byte) (databasetoolssdk.CreateDatabaseToolsConnectionDetails, error) {
	concreteType, err := databaseToolsConnectionCreateConcreteType(payload)
	if err != nil {
		return nil, err
	}

	converted := reflect.New(concreteType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return nil, fmt.Errorf("unmarshal into %s: %w", concreteType, err)
	}
	body, ok := converted.Elem().Interface().(databasetoolssdk.CreateDatabaseToolsConnectionDetails)
	if !ok {
		return nil, fmt.Errorf("resolved CreateDatabaseToolsConnectionDetails type %s does not implement the polymorphic interface", concreteType)
	}
	return body, nil
}

func convertDatabaseToolsConnectionUpdateDetails(payload []byte) (databasetoolssdk.UpdateDatabaseToolsConnectionDetails, error) {
	concreteType, err := databaseToolsConnectionUpdateConcreteType(payload)
	if err != nil {
		return nil, err
	}

	converted := reflect.New(concreteType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return nil, fmt.Errorf("unmarshal into %s: %w", concreteType, err)
	}
	body, ok := converted.Elem().Interface().(databasetoolssdk.UpdateDatabaseToolsConnectionDetails)
	if !ok {
		return nil, fmt.Errorf("resolved UpdateDatabaseToolsConnectionDetails type %s does not implement the polymorphic interface", concreteType)
	}
	return body, nil
}

func databaseToolsConnectionCreateConcreteType(payload []byte) (reflect.Type, error) {
	connectionType, err := jsonFieldString(payload, "type")
	if err != nil {
		return nil, fmt.Errorf("decode DatabaseToolsConnection create type: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(connectionType)) {
	case "GENERIC_JDBC":
		return reflect.TypeOf(databasetoolssdk.CreateDatabaseToolsConnectionGenericJdbcDetails{}), nil
	case "POSTGRESQL":
		return reflect.TypeOf(databasetoolssdk.CreateDatabaseToolsConnectionPostgresqlDetails{}), nil
	case "MYSQL":
		return reflect.TypeOf(databasetoolssdk.CreateDatabaseToolsConnectionMySqlDetails{}), nil
	case "ORACLE_DATABASE":
		return reflect.TypeOf(databasetoolssdk.CreateDatabaseToolsConnectionOracleDatabaseDetails{}), nil
	default:
		return nil, fmt.Errorf("unsupported CreateDatabaseToolsConnectionDetails type %q", connectionType)
	}
}

func databaseToolsConnectionUpdateConcreteType(payload []byte) (reflect.Type, error) {
	connectionType, err := jsonFieldString(payload, "type")
	if err != nil {
		return nil, fmt.Errorf("decode DatabaseToolsConnection update type: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(connectionType)) {
	case "GENERIC_JDBC":
		return reflect.TypeOf(databasetoolssdk.UpdateDatabaseToolsConnectionGenericJdbcDetails{}), nil
	case "POSTGRESQL":
		return reflect.TypeOf(databasetoolssdk.UpdateDatabaseToolsConnectionPostgresqlDetails{}), nil
	case "MYSQL":
		return reflect.TypeOf(databasetoolssdk.UpdateDatabaseToolsConnectionMySqlDetails{}), nil
	case "ORACLE_DATABASE":
		return reflect.TypeOf(databasetoolssdk.UpdateDatabaseToolsConnectionOracleDatabaseDetails{}), nil
	default:
		return nil, fmt.Errorf("unsupported UpdateDatabaseToolsConnectionDetails type %q", connectionType)
	}
}

func jsonFieldString(payload []byte, field string) (string, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(payload, &values); err != nil {
		return "", err
	}
	raw, ok := values[field]
	if !ok || string(raw) == "null" {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	return value, nil
}

func assignDeterministicRetryToken(requestStruct reflect.Value, resource any) {
	field, ok := fieldValue(requestStruct, "OpcRetryToken")
	if !ok || !field.IsValid() || !field.CanSet() {
		return
	}

	switch field.Kind() {
	case reflect.Pointer:
		if !field.IsNil() {
			return
		}
	case reflect.String:
		if strings.TrimSpace(field.String()) != "" {
			return
		}
	default:
		return
	}

	token := resourceRetryToken(resource)
	if token == "" {
		return
	}
	_ = assignField(field, token)
}

func resourceRetryToken(resource any) string {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return ""
	}
	if uid := strings.TrimSpace(lookupMetadataString(resourceValue, "UID")); uid != "" {
		return uid
	}

	namespace := strings.TrimSpace(lookupMetadataString(resourceValue, "Namespace"))
	name := strings.TrimSpace(lookupMetadataString(resourceValue, "Name"))
	if namespace == "" && name == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(namespace + "/" + name))
	return fmt.Sprintf("%x", sum[:16])
}

func resourceNamespace(resource any, fallback string) string {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return strings.TrimSpace(fallback)
	}
	namespace := lookupMetadataString(resourceValue, "Namespace")
	if strings.TrimSpace(namespace) != "" {
		return namespace
	}
	return strings.TrimSpace(fallback)
}

func fieldJSONName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	return strings.Split(tag, ",")[0]
}

func jsonMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil
	}
	return decoded
}

func mutationJSONMap(value any) (map[string]any, error) {
	decoded, err := decodedJSONValueWithBoolFields(value)
	if err != nil {
		return nil, err
	}
	if decoded == nil {
		return nil, nil
	}
	decodedMap, ok := decoded.(map[string]any)
	if !ok {
		return nil, nil
	}
	return decodedMap, nil
}

func meaningfulMutationValueByPath(value any, path string) (any, bool, error) {
	values, err := mutationJSONMap(value)
	if err != nil {
		return nil, false, err
	}
	if values == nil {
		return nil, false, nil
	}
	resolved, ok := lookupMeaningfulValue(values, path)
	if !ok {
		return nil, false, nil
	}
	return resolved, true, nil
}

func decodedJSONValue(value any) (any, error) {
	if value == nil {
		return nil, nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal value: %w", err)
	}

	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func decodedJSONValueWithBoolFields(value any) (any, error) {
	decoded, err := decodedJSONValue(value)
	if err != nil {
		return nil, err
	}
	overlayed, _ := overlayBoolFields(reflect.ValueOf(value), decoded)
	return overlayed, nil
}

func overlayBoolFields(value reflect.Value, decoded any) (any, bool) {
	value, ok := indirectValue(value)
	if !ok {
		return decoded, decoded != nil
	}
	if value.Kind() != reflect.Struct {
		return decoded, decoded != nil
	}

	decodedMap, _ := decoded.(map[string]any)
	if decodedMap == nil {
		decodedMap = map[string]any{}
	}
	hasAny := len(decodedMap) > 0

	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := typ.Field(i)
		if !fieldType.IsExported() {
			continue
		}

		fieldValue := value.Field(i)
		if fieldType.Anonymous && embeddedJSONField(fieldType) {
			embedded, embeddedHasAny := overlayBoolFields(fieldValue, decodedMap)
			if embeddedMap, ok := embedded.(map[string]any); ok {
				decodedMap = embeddedMap
				hasAny = len(decodedMap) > 0 || embeddedHasAny
			}
			continue
		}

		jsonName := fieldJSONName(fieldType)
		if jsonName == "" {
			continue
		}

		indirectField, ok := indirectValue(fieldValue)
		if !ok {
			continue
		}

		switch indirectField.Kind() {
		case reflect.Bool:
			decodedMap[jsonName] = indirectField.Bool()
			hasAny = true
		case reflect.Struct:
			childDecoded, _ := decodedMap[jsonName]
			child, childHasAny := overlayBoolFields(fieldValue, childDecoded)
			if childHasAny {
				decodedMap[jsonName] = child
				hasAny = true
			}
		}
	}

	if !hasAny {
		return nil, false
	}
	return decodedMap, true
}

func lookupValueByPaths(values map[string]any, paths ...string) (any, bool) {
	for _, path := range paths {
		if value, ok := lookupValueByPath(values, path); ok {
			return value, true
		}
	}
	return nil, false
}

func lookupMeaningfulValue(values map[string]any, path string) (any, bool) {
	value, ok := lookupValueByPath(values, path)
	if !ok || !meaningfulValue(value) {
		return nil, false
	}
	return value, true
}

func lookupValueByPath(values map[string]any, path string) (any, bool) {
	if values == nil {
		return nil, false
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, false
	}

	segments := strings.Split(path, ".")
	if current, ok := lookupRootScopedValue(values, segments); ok {
		return current, true
	}
	return lookupValueBySegments(values, segments)
}

func lookupRootScopedValue(values map[string]any, segments []string) (any, bool) {
	if len(segments) == 0 {
		return nil, false
	}

	switch normalizePathSegment(segments[0]) {
	case "spec":
		return lookupNamedRootValue(values, lookupSpecRootKey, segments[1:])
	case "status":
		return lookupNamedRootValue(values, lookupStatusRootKey, segments[1:])
	default:
		return nil, false
	}
}

func lookupNamedRootValue(values map[string]any, rootKey string, segments []string) (any, bool) {
	root, ok := values[rootKey].(map[string]any)
	if !ok {
		return nil, false
	}
	if len(segments) == 0 {
		return root, true
	}
	return lookupValueBySegments(root, segments)
}

func lookupValueBySegments(root map[string]any, segments []string) (any, bool) {
	current := any(root)
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return nil, false
		}

		mapValue, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := lookupMapKey(mapValue, segment)
		if !ok {
			return nil, false
		}
		current = next
	}

	return current, true
}

func lookupMapKey(values map[string]any, segment string) (any, bool) {
	if value, ok := values[segment]; ok {
		return value, true
	}

	normalized := normalizePathSegment(segment)
	for key, value := range values {
		if normalizePathSegment(key) == normalized {
			return value, true
		}
	}
	return nil, false
}

func meaningfulValue(value any) bool {
	if value == nil {
		return false
	}

	switch concrete := value.(type) {
	case string:
		return strings.TrimSpace(concrete) != ""
	case []any:
		for _, item := range concrete {
			if meaningfulValue(item) {
				return true
			}
		}
		return false
	case map[string]any:
		for _, item := range concrete {
			if meaningfulValue(item) {
				return true
			}
		}
		return false
	case bool:
		return true
	case float64:
		return concrete != 0
	default:
		return true
	}
}

func valuesEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}
	return string(leftPayload) == string(rightPayload)
}

func firstNonEmpty(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := lookupString(values, key); value != "" {
			return value
		}
	}
	return ""
}

func lookupString(values map[string]any, key string) string {
	raw, ok := lookupValueByPath(values, key)
	if !ok || raw == nil {
		return ""
	}
	switch concrete := raw.(type) {
	case string:
		return concrete
	default:
		return fmt.Sprint(concrete)
	}
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func appendUniqueStrings(existing []string, extras ...string) []string {
	seen := make(map[string]struct{}, len(existing)+len(extras))
	for _, value := range existing {
		seen[value] = struct{}{}
	}
	for _, value := range extras {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		existing = append(existing, value)
	}
	return existing
}

func lowerCamel(name string) string {
	tokens := splitCamel(name)
	if len(tokens) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(tokens[0])
	for _, token := range tokens[1:] {
		builder.WriteString(strings.ToUpper(token[:1]))
		builder.WriteString(token[1:])
	}
	return builder.String()
}

func splitCamel(name string) []string {
	if strings.TrimSpace(name) == "" {
		return nil
	}

	var tokens []string
	var current []rune
	runes := []rune(name)
	for index, r := range runes {
		if splitBeforeCamelRune(runes, index) {
			tokens = append(tokens, strings.ToLower(string(current)))
			current = current[:0]
		}
		current = append(current, r)
	}
	if len(current) > 0 {
		tokens = append(tokens, strings.ToLower(string(current)))
	}
	return tokens
}

func splitBeforeCamelRune(runes []rune, index int) bool {
	if index == 0 {
		return false
	}

	current := runes[index]
	prev := runes[index-1]
	nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
	return unicode.IsUpper(current) &&
		(unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextIsLower))
}
