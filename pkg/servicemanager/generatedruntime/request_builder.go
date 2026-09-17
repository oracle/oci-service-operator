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

	apmconfigsdk "github.com/oracle/oci-go-sdk/v65/apmconfig"
	autoscalingsdk "github.com/oracle/oci-go-sdk/v65/autoscaling"
	"github.com/oracle/oci-go-sdk/v65/common"
	dashboardservicesdk "github.com/oracle/oci-go-sdk/v65/dashboardservice"
	databasesdk "github.com/oracle/oci-go-sdk/v65/database"
	databasemigrationsdk "github.com/oracle/oci-go-sdk/v65/databasemigration"
	databasetoolssdk "github.com/oracle/oci-go-sdk/v65/databasetools"
	dataintegrationsdk "github.com/oracle/oci-go-sdk/v65/dataintegration"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	networkfirewallsdk "github.com/oracle/oci-go-sdk/v65/networkfirewall"
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
	DisableRetries   bool
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
		applyRequestRetryPolicy(requestStruct, options)
		return nil
	}

	if err := buildHeuristicRequest(requestStruct, requestStruct.Type(), values, preferredID, idAliases, resolvedSpec); err != nil {
		return err
	}
	assignDeterministicRetryToken(requestStruct, resource)
	applyRequestRetryPolicy(requestStruct, options)
	return nil
}

func applyRequestRetryPolicy(requestStruct reflect.Value, options requestBuildOptions) {
	if !options.DisableRetries {
		return
	}
	metadata, ok := fieldValue(requestStruct, "RequestMetadata")
	if !ok || metadata.Kind() != reflect.Struct {
		return
	}
	retryPolicy := metadata.FieldByName("RetryPolicy")
	if !retryPolicy.IsValid() || !retryPolicy.CanSet() || retryPolicy.Kind() != reflect.Pointer {
		return
	}
	policy := common.NoRetryPolicy()
	if reflect.TypeOf(&policy).AssignableTo(retryPolicy.Type()) {
		retryPolicy.Set(reflect.ValueOf(&policy))
	}
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
	rawValue := reflect.ValueOf(raw)
	if targetType.Kind() == reflect.Slice && targetType.Elem().Kind() != reflect.Uint8 && rawValue.Kind() != reflect.Slice && rawValue.Kind() != reflect.Array {
		item, err := convertValue(raw, targetType.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		converted := reflect.MakeSlice(targetType, 1, 1)
		converted.Index(0).Set(item)
		return converted, nil
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
	case autoScalingPolicyCreateDetailsType:
		body, err := convertDiscriminatedInterface[autoscalingsdk.CreateAutoScalingPolicyDetails](payload, "Autoscaling policy", "policyType", map[string]reflect.Type{
			"SCHEDULED": reflect.TypeOf(autoscalingsdk.CreateScheduledPolicyDetails{}),
			"THRESHOLD": reflect.TypeOf(autoscalingsdk.CreateThresholdPolicyDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case autoScalingPolicyUpdateDetailsType:
		body, err := convertDiscriminatedInterface[autoscalingsdk.UpdateAutoScalingPolicyDetails](payload, "Autoscaling policy", "policyType", map[string]reflect.Type{
			"SCHEDULED": reflect.TypeOf(autoscalingsdk.UpdateScheduledPolicyDetails{}),
			"THRESHOLD": reflect.TypeOf(autoscalingsdk.UpdateThresholdPolicyDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case autonomousDatabaseBaseType:
		body, err := convertAutonomousDatabaseBase(payload)
		if err != nil {
			return reflect.Value{}, true, err
		}
		converted := reflect.New(targetType).Elem()
		converted.Set(reflect.ValueOf(body))
		return converted, true, nil
	case connectionCreateDetailsType:
		body, err := convertConnectionCreateDetails(payload)
		if err != nil {
			return reflect.Value{}, true, err
		}
		converted := reflect.New(targetType).Elem()
		converted.Set(reflect.ValueOf(body))
		return converted, true, nil
	case connectionUpdateDetailsType:
		body, err := convertConnectionUpdateDetails(payload)
		if err != nil {
			return reflect.Value{}, true, err
		}
		converted := reflect.New(targetType).Elem()
		converted.Set(reflect.ValueOf(body))
		return converted, true, nil
	case configCreateDetailsType:
		body, err := convertConfigCreateDetails(payload)
		if err != nil {
			return reflect.Value{}, true, err
		}
		converted := reflect.New(targetType).Elem()
		converted.Set(reflect.ValueOf(body))
		return converted, true, nil
	case configUpdateDetailsType:
		body, err := convertConfigUpdateDetails(payload)
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
	case dataIntegrationConnectionCreateType:
		body, err := convertDiscriminatedInterface[dataintegrationsdk.CreateConnectionDetails](payload, "Data Integration connection", "modelType", map[string]reflect.Type{
			"AMAZON_S3_CONNECTION":             reflect.TypeOf(dataintegrationsdk.CreateConnectionFromAmazonS3{}),
			"BICC_CONNECTION":                  reflect.TypeOf(dataintegrationsdk.CreateConnectionFromBicc{}),
			"BIP_CONNECTION":                   reflect.TypeOf(dataintegrationsdk.CreateConnectionFromBip{}),
			"GENERIC_JDBC_CONNECTION":          reflect.TypeOf(dataintegrationsdk.CreateConnectionFromJdbc{}),
			"HDFS_CONNECTION":                  reflect.TypeOf(dataintegrationsdk.CreateConnectionFromHdfs{}),
			"LAKE_CONNECTION":                  reflect.TypeOf(dataintegrationsdk.CreateConnectionFromLake{}),
			"MYSQL_CONNECTION":                 reflect.TypeOf(dataintegrationsdk.CreateConnectionFromMySql{}),
			"MYSQL_HEATWAVE_CONNECTION":        reflect.TypeOf(dataintegrationsdk.CreateConnectionFromMySqlHeatWave{}),
			"OAUTH2_CONNECTION":                reflect.TypeOf(dataintegrationsdk.CreateConnectionFromOAuth2{}),
			"ORACLE_ADWC_CONNECTION":           reflect.TypeOf(dataintegrationsdk.CreateConnectionFromAdwc{}),
			"ORACLE_ATP_CONNECTION":            reflect.TypeOf(dataintegrationsdk.CreateConnectionFromAtp{}),
			"ORACLE_EBS_CONNECTION":            reflect.TypeOf(dataintegrationsdk.CreateConnectionFromOracleEbs{}),
			"ORACLE_OBJECT_STORAGE_CONNECTION": reflect.TypeOf(dataintegrationsdk.CreateConnectionFromObjectStorage{}),
			"ORACLE_PEOPLESOFT_CONNECTION":     reflect.TypeOf(dataintegrationsdk.CreateConnectionFromOraclePeopleSoft{}),
			"ORACLE_SIEBEL_CONNECTION":         reflect.TypeOf(dataintegrationsdk.CreateConnectionFromOracleSiebel{}),
			"ORACLEDB_CONNECTION":              reflect.TypeOf(dataintegrationsdk.CreateConnectionFromOracle{}),
			"REST_BASIC_AUTH_CONNECTION":       reflect.TypeOf(dataintegrationsdk.CreateConnectionFromRestBasicAuth{}),
			"REST_NO_AUTH_CONNECTION":          reflect.TypeOf(dataintegrationsdk.CreateConnectionFromRestNoAuth{}),
		})
		return interfaceValue(targetType, body, err)
	case dataIntegrationConnectionUpdateType:
		body, err := convertDiscriminatedInterface[dataintegrationsdk.UpdateConnectionDetails](payload, "Data Integration connection", "modelType", map[string]reflect.Type{
			"AMAZON_S3_CONNECTION":             reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromAmazonS3{}),
			"BICC_CONNECTION":                  reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromBicc{}),
			"BIP_CONNECTION":                   reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromBip{}),
			"GENERIC_JDBC_CONNECTION":          reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromJdbc{}),
			"HDFS_CONNECTION":                  reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromHdfs{}),
			"LAKE_CONNECTION":                  reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromLake{}),
			"MYSQL_CONNECTION":                 reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromMySql{}),
			"MYSQL_HEATWAVE_CONNECTION":        reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromMySqlHeatWave{}),
			"OAUTH2_CONNECTION":                reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromOAuth2{}),
			"ORACLE_ADWC_CONNECTION":           reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromAdwc{}),
			"ORACLE_ATP_CONNECTION":            reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromAtp{}),
			"ORACLE_EBS_CONNECTION":            reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromOracleEbs{}),
			"ORACLE_OBJECT_STORAGE_CONNECTION": reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromObjectStorage{}),
			"ORACLE_PEOPLESOFT_CONNECTION":     reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromOraclePeopleSoft{}),
			"ORACLE_SIEBEL_CONNECTION":         reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromOracleSiebel{}),
			"ORACLEDB_CONNECTION":              reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromOracle{}),
			"REST_BASIC_AUTH_CONNECTION":       reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromRestBasicAuth{}),
			"REST_NO_AUTH_CONNECTION":          reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromRestNoAuth{}),
		})
		return interfaceValue(targetType, body, err)
	case dataIntegrationDataAssetCreateType:
		body, err := convertDiscriminatedInterface[dataintegrationsdk.CreateDataAssetDetails](payload, "Data Integration data asset", "modelType", map[string]reflect.Type{
			"AMAZON_S3_DATA_ASSET":             reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromAmazonS3{}),
			"FUSION_APP_DATA_ASSET":            reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromFusionApp{}),
			"GENERIC_JDBC_DATA_ASSET":          reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromJdbc{}),
			"HDFS_DATA_ASSET":                  reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromHdfs{}),
			"LAKE_DATA_ASSET":                  reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromLake{}),
			"MYSQL_DATA_ASSET":                 reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromMySql{}),
			"MYSQL_HEATWAVE_DATA_ASSET":        reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromMySqlHeatWave{}),
			"ORACLE_ADWC_DATA_ASSET":           reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromAdwc{}),
			"ORACLE_ATP_DATA_ASSET":            reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromAtp{}),
			"ORACLE_DATA_ASSET":                reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromOracle{}),
			"ORACLE_EBS_DATA_ASSET":            reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromOracleEbs{}),
			"ORACLE_OBJECT_STORAGE_DATA_ASSET": reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromObjectStorage{}),
			"ORACLE_PEOPLESOFT_DATA_ASSET":     reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromOraclePeopleSoft{}),
			"ORACLE_SIEBEL_DATA_ASSET":         reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromOracleSiebel{}),
			"REST_DATA_ASSET":                  reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromRest{}),
		})
		return interfaceValue(targetType, body, err)
	case dataIntegrationDataAssetUpdateType:
		body, err := convertDiscriminatedInterface[dataintegrationsdk.UpdateDataAssetDetails](payload, "Data Integration data asset", "modelType", map[string]reflect.Type{
			"AMAZON_S3_DATA_ASSET":             reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromAmazonS3{}),
			"FUSION_APP_DATA_ASSET":            reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromFusionApp{}),
			"GENERIC_JDBC_DATA_ASSET":          reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromJdbc{}),
			"HDFS_DATA_ASSET":                  reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromHdfs{}),
			"LAKE_DATA_ASSET":                  reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromLake{}),
			"MYSQL_DATA_ASSET":                 reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromMySql{}),
			"MYSQL_HEATWAVE_DATA_ASSET":        reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromMySqlHeatWave{}),
			"ORACLE_ADWC_DATA_ASSET":           reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromAdwc{}),
			"ORACLE_ATP_DATA_ASSET":            reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromAtp{}),
			"ORACLE_DATA_ASSET":                reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromOracle{}),
			"ORACLE_EBS_DATA_ASSET":            reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromOracleEbs{}),
			"ORACLE_OBJECT_STORAGE_DATA_ASSET": reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromObjectStorage{}),
			"ORACLE_PEOPLESOFT_DATA_ASSET":     reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromOraclePeopleSoft{}),
			"ORACLE_SIEBEL_DATA_ASSET":         reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromOracleSiebel{}),
			"REST_DATA_ASSET":                  reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromRest{}),
		})
		return interfaceValue(targetType, body, err)
	case dataIntegrationTaskCreateType:
		body, err := convertDiscriminatedInterface[dataintegrationsdk.CreateTaskDetails](payload, "Data Integration task", "modelType", map[string]reflect.Type{
			"DATA_LOADER_TASK":  reflect.TypeOf(dataintegrationsdk.CreateTaskFromDataLoaderTask{}),
			"INTEGRATION_TASK":  reflect.TypeOf(dataintegrationsdk.CreateTaskFromIntegrationTask{}),
			"OCI_DATAFLOW_TASK": reflect.TypeOf(dataintegrationsdk.CreateTaskFromOciDataflowTask{}),
			"PIPELINE_TASK":     reflect.TypeOf(dataintegrationsdk.CreateTaskFromPipelineTask{}),
			"REST_TASK":         reflect.TypeOf(dataintegrationsdk.CreateTaskFromRestTask{}),
			"SQL_TASK":          reflect.TypeOf(dataintegrationsdk.CreateTaskFromSqlTask{}),
		})
		return interfaceValue(targetType, body, err)
	case dataIntegrationTaskUpdateType:
		body, err := convertDiscriminatedInterface[dataintegrationsdk.UpdateTaskDetails](payload, "Data Integration task", "modelType", map[string]reflect.Type{
			"DATA_LOADER_TASK":  reflect.TypeOf(dataintegrationsdk.UpdateTaskFromDataLoaderTask{}),
			"INTEGRATION_TASK":  reflect.TypeOf(dataintegrationsdk.UpdateTaskFromIntegrationTask{}),
			"OCI_DATAFLOW_TASK": reflect.TypeOf(dataintegrationsdk.UpdateTaskFromOciDataflowTask{}),
			"PIPELINE_TASK":     reflect.TypeOf(dataintegrationsdk.UpdateTaskFromPipelineTask{}),
			"REST_TASK":         reflect.TypeOf(dataintegrationsdk.UpdateTaskFromRestTask{}),
			"SQL_TASK":          reflect.TypeOf(dataintegrationsdk.UpdateTaskFromSqlTask{}),
		})
		return interfaceValue(targetType, body, err)
	case dashboardCreateDetailsType:
		body, err := convertDashboardCreateDetails(payload)
		if err != nil {
			return reflect.Value{}, true, err
		}
		converted := reflect.New(targetType).Elem()
		converted.Set(reflect.ValueOf(body))
		return converted, true, nil
	case dashboardUpdateDetailsType:
		body, err := convertDashboardUpdateDetails(payload)
		if err != nil {
			return reflect.Value{}, true, err
		}
		converted := reflect.New(targetType).Elem()
		converted.Set(reflect.ValueOf(body))
		return converted, true, nil
	case sensitiveTypeCreateDetailsType:
		body, err := convertSensitiveTypePolymorphic[datasafesdk.CreateSensitiveTypeDetails](payload, map[string]reflect.Type{
			"SENSITIVE_TYPE":     reflect.TypeOf(datasafesdk.CreateSensitiveTypePatternDetails{}),
			"SENSITIVE_CATEGORY": reflect.TypeOf(datasafesdk.CreateSensitiveCategoryDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case sensitiveTypeUpdateDetailsType:
		body, err := convertSensitiveTypePolymorphic[datasafesdk.UpdateSensitiveTypeDetails](payload, map[string]reflect.Type{
			"SENSITIVE_TYPE":     reflect.TypeOf(datasafesdk.UpdateSensitiveTypePatternDetails{}),
			"SENSITIVE_CATEGORY": reflect.TypeOf(datasafesdk.UpdateSensitiveCategoryDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case networkFirewallUpdateAddressListType:
		body, err := convertNetworkFirewallPolymorphic[networkfirewallsdk.UpdateAddressListDetails](payload, "type", map[string]reflect.Type{
			"FQDN": reflect.TypeOf(networkfirewallsdk.UpdateFqdnAddressListDetails{}),
			"IP":   reflect.TypeOf(networkfirewallsdk.UpdateIpAddressListDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case networkFirewallCreateApplicationType:
		body, err := convertNetworkFirewallPolymorphic[networkfirewallsdk.CreateApplicationDetails](payload, "type", map[string]reflect.Type{
			"ICMP":    reflect.TypeOf(networkfirewallsdk.CreateIcmpApplicationDetails{}),
			"ICMP_V6": reflect.TypeOf(networkfirewallsdk.CreateIcmp6ApplicationDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case networkFirewallUpdateApplicationType:
		body, err := convertNetworkFirewallPolymorphic[networkfirewallsdk.UpdateApplicationDetails](payload, "type", map[string]reflect.Type{
			"ICMP":    reflect.TypeOf(networkfirewallsdk.UpdateIcmpApplicationDetails{}),
			"ICMP_V6": reflect.TypeOf(networkfirewallsdk.UpdateIcmp6ApplicationDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case networkFirewallCreateDecryptionType:
		body, err := convertNetworkFirewallPolymorphic[networkfirewallsdk.CreateDecryptionProfileDetails](payload, "type", map[string]reflect.Type{
			"SSL_FORWARD_PROXY":      reflect.TypeOf(networkfirewallsdk.CreateSslForwardProxyProfileDetails{}),
			"SSL_INBOUND_INSPECTION": reflect.TypeOf(networkfirewallsdk.CreateSslInboundInspectionProfileDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case networkFirewallUpdateDecryptionType:
		body, err := convertNetworkFirewallPolymorphic[networkfirewallsdk.UpdateDecryptionProfileDetails](payload, "type", map[string]reflect.Type{
			"SSL_FORWARD_PROXY":      reflect.TypeOf(networkfirewallsdk.UpdateSslForwardProxyProfileDetails{}),
			"SSL_INBOUND_INSPECTION": reflect.TypeOf(networkfirewallsdk.UpdateSslInboundInspectionProfileDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case networkFirewallCreateMappedSecretType:
		body, err := convertNetworkFirewallPolymorphic[networkfirewallsdk.CreateMappedSecretDetails](payload, "source", map[string]reflect.Type{
			"OCI_VAULT": reflect.TypeOf(networkfirewallsdk.CreateVaultMappedSecretDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case networkFirewallUpdateMappedSecretType:
		body, err := convertNetworkFirewallPolymorphic[networkfirewallsdk.UpdateMappedSecretDetails](payload, "source", map[string]reflect.Type{
			"OCI_VAULT": reflect.TypeOf(networkfirewallsdk.UpdateVaultMappedSecretDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case networkFirewallCreateNatRuleType:
		body, err := convertNetworkFirewallPolymorphic[networkfirewallsdk.CreateNatRuleDetails](payload, "type", map[string]reflect.Type{
			"NATV4": reflect.TypeOf(networkfirewallsdk.CreateNatV4RuleDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case networkFirewallUpdateNatRuleType:
		body, err := convertNetworkFirewallPolymorphic[networkfirewallsdk.UpdateNatRuleDetails](payload, "type", map[string]reflect.Type{
			"NATV4": reflect.TypeOf(networkfirewallsdk.UpdateNatV4RuleDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case networkFirewallCreateServiceType:
		body, err := convertNetworkFirewallPolymorphic[networkfirewallsdk.CreateServiceDetails](payload, "type", map[string]reflect.Type{
			"TCP_SERVICE": reflect.TypeOf(networkfirewallsdk.CreateTcpServiceDetails{}),
			"UDP_SERVICE": reflect.TypeOf(networkfirewallsdk.CreateUdpServiceDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case networkFirewallUpdateServiceType:
		body, err := convertNetworkFirewallPolymorphic[networkfirewallsdk.UpdateServiceDetails](payload, "type", map[string]reflect.Type{
			"TCP_SERVICE": reflect.TypeOf(networkfirewallsdk.UpdateTcpServiceDetails{}),
			"UDP_SERVICE": reflect.TypeOf(networkfirewallsdk.UpdateUdpServiceDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case networkFirewallCreateTunnelRuleType:
		body, err := convertNetworkFirewallPolymorphic[networkfirewallsdk.CreateTunnelInspectionRuleDetails](payload, "protocol", map[string]reflect.Type{
			"VXLAN": reflect.TypeOf(networkfirewallsdk.CreateVxlanInspectionRuleDetails{}),
		})
		return interfaceValue(targetType, body, err)
	case networkFirewallUpdateTunnelRuleType:
		body, err := convertNetworkFirewallPolymorphic[networkfirewallsdk.UpdateTunnelInspectionRuleDetails](payload, "protocol", map[string]reflect.Type{
			"VXLAN": reflect.TypeOf(networkfirewallsdk.UpdateVxlanInspectionRuleDetails{}),
		})
		return interfaceValue(targetType, body, err)
	default:
		return convertAdditionalPolymorphicInterfaceValue(payload, targetType)
	}
}

func convertSensitiveTypePolymorphic[T any](payload []byte, concreteTypes map[string]reflect.Type) (T, error) {
	var zero T
	entityType, err := jsonFieldString(payload, "entityType")
	if err != nil {
		return zero, fmt.Errorf("decode Data Safe SensitiveType entityType discriminator: %w", err)
	}
	concreteType, ok := concreteTypes[strings.ToUpper(strings.TrimSpace(entityType))]
	if !ok {
		return zero, fmt.Errorf("unsupported Data Safe SensitiveType entityType discriminator %q", entityType)
	}
	converted := reflect.New(concreteType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return zero, fmt.Errorf("unmarshal into %s: %w", concreteType, err)
	}
	body, ok := converted.Elem().Interface().(T)
	if !ok {
		return zero, fmt.Errorf("resolved Data Safe SensitiveType type %s does not implement %s", concreteType, reflect.TypeOf((*T)(nil)).Elem())
	}
	return body, nil
}

func convertNetworkFirewallPolymorphic[T any](payload []byte, discriminator string, concreteTypes map[string]reflect.Type) (T, error) {
	return convertDiscriminatedInterface[T](payload, "Network Firewall", discriminator, concreteTypes)
}

func convertDiscriminatedInterface[T any](payload []byte, subject string, discriminator string, concreteTypes map[string]reflect.Type) (T, error) {
	var zero T
	value, err := jsonFieldString(payload, discriminator)
	if err != nil {
		return zero, fmt.Errorf("decode %s %s discriminator: %w", subject, discriminator, err)
	}
	concreteType, ok := concreteTypes[strings.ToUpper(strings.TrimSpace(value))]
	if !ok {
		return zero, fmt.Errorf("unsupported %s %s discriminator %q", subject, discriminator, value)
	}
	converted := reflect.New(concreteType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return zero, fmt.Errorf("unmarshal into %s: %w", concreteType, err)
	}
	body, ok := converted.Elem().Interface().(T)
	if !ok {
		return zero, fmt.Errorf("resolved Network Firewall type %s does not implement %s", concreteType, reflect.TypeOf((*T)(nil)).Elem())
	}
	return body, nil
}

func interfaceValue(targetType reflect.Type, body any, err error) (reflect.Value, bool, error) {
	if err != nil {
		return reflect.Value{}, true, err
	}
	converted := reflect.New(targetType).Elem()
	converted.Set(reflect.ValueOf(body))
	return converted, true, nil
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

func convertConnectionCreateDetails(payload []byte) (databasemigrationsdk.CreateConnectionDetails, error) {
	concreteType, err := connectionCreateConcreteType(payload)
	if err != nil {
		return nil, err
	}

	converted := reflect.New(concreteType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return nil, fmt.Errorf("unmarshal into %s: %w", concreteType, err)
	}
	body, ok := converted.Elem().Interface().(databasemigrationsdk.CreateConnectionDetails)
	if !ok {
		return nil, fmt.Errorf("resolved CreateConnectionDetails type %s does not implement the polymorphic interface", concreteType)
	}
	return body, nil
}

func convertConnectionUpdateDetails(payload []byte) (databasemigrationsdk.UpdateConnectionDetails, error) {
	concreteType, err := connectionUpdateConcreteType(payload)
	if err != nil {
		return nil, err
	}

	converted := reflect.New(concreteType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return nil, fmt.Errorf("unmarshal into %s: %w", concreteType, err)
	}
	body, ok := converted.Elem().Interface().(databasemigrationsdk.UpdateConnectionDetails)
	if !ok {
		return nil, fmt.Errorf("resolved UpdateConnectionDetails type %s does not implement the polymorphic interface", concreteType)
	}
	return body, nil
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

func convertConfigCreateDetails(payload []byte) (apmconfigsdk.CreateConfigDetails, error) {
	concreteType, err := configCreateConcreteType(payload)
	if err != nil {
		return nil, err
	}

	converted := reflect.New(concreteType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return nil, fmt.Errorf("unmarshal into %s: %w", concreteType, err)
	}
	body, ok := converted.Elem().Interface().(apmconfigsdk.CreateConfigDetails)
	if !ok {
		return nil, fmt.Errorf("resolved CreateConfigDetails type %s does not implement the polymorphic interface", concreteType)
	}
	return body, nil
}

func convertConfigUpdateDetails(payload []byte) (apmconfigsdk.UpdateConfigDetails, error) {
	concreteType, err := configUpdateConcreteType(payload)
	if err != nil {
		return nil, err
	}

	converted := reflect.New(concreteType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return nil, fmt.Errorf("unmarshal into %s: %w", concreteType, err)
	}
	body, ok := converted.Elem().Interface().(apmconfigsdk.UpdateConfigDetails)
	if !ok {
		return nil, fmt.Errorf("resolved UpdateConfigDetails type %s does not implement the polymorphic interface", concreteType)
	}
	return body, nil
}

func convertDashboardCreateDetails(payload []byte) (dashboardservicesdk.CreateDashboardDetails, error) {
	concreteType, err := dashboardCreateConcreteType(payload)
	if err != nil {
		return nil, err
	}

	converted := reflect.New(concreteType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return nil, fmt.Errorf("unmarshal into %s: %w", concreteType, err)
	}
	body, ok := converted.Elem().Interface().(dashboardservicesdk.CreateDashboardDetails)
	if !ok {
		return nil, fmt.Errorf("resolved CreateDashboardDetails type %s does not implement the polymorphic interface", concreteType)
	}
	return body, nil
}

func convertDashboardUpdateDetails(payload []byte) (dashboardservicesdk.UpdateDashboardDetails, error) {
	concreteType, err := dashboardUpdateConcreteType(payload)
	if err != nil {
		return nil, err
	}

	converted := reflect.New(concreteType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return nil, fmt.Errorf("unmarshal into %s: %w", concreteType, err)
	}
	body, ok := converted.Elem().Interface().(dashboardservicesdk.UpdateDashboardDetails)
	if !ok {
		return nil, fmt.Errorf("resolved UpdateDashboardDetails type %s does not implement the polymorphic interface", concreteType)
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

func connectionCreateConcreteType(payload []byte) (reflect.Type, error) {
	connectionType, err := jsonFieldString(payload, "connectionType")
	if err != nil {
		return nil, fmt.Errorf("decode Connection create type: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(connectionType)) {
	case "MYSQL":
		return reflect.TypeOf(databasemigrationsdk.CreateMysqlConnectionDetails{}), nil
	case "ORACLE":
		return reflect.TypeOf(databasemigrationsdk.CreateOracleConnectionDetails{}), nil
	default:
		return nil, fmt.Errorf("unsupported CreateConnectionDetails type %q", connectionType)
	}
}

func configCreateConcreteType(payload []byte) (reflect.Type, error) {
	configType, err := jsonFieldString(payload, "configType")
	if err != nil {
		return nil, fmt.Errorf("decode Config create type: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(configType)) {
	case "AGENT":
		return reflect.TypeOf(apmconfigsdk.CreateAgentConfigDetails{}), nil
	case "APDEX":
		return reflect.TypeOf(apmconfigsdk.CreateApdexRulesDetails{}), nil
	case "MACS_APM_EXTENSION":
		return reflect.TypeOf(apmconfigsdk.CreateMacsApmExtensionDetails{}), nil
	case "METRIC_GROUP":
		return reflect.TypeOf(apmconfigsdk.CreateMetricGroupDetails{}), nil
	case "OPTIONS":
		return reflect.TypeOf(apmconfigsdk.CreateOptionsDetails{}), nil
	case "SPAN_FILTER":
		return reflect.TypeOf(apmconfigsdk.CreateSpanFilterDetails{}), nil
	default:
		return nil, fmt.Errorf("unsupported CreateConfigDetails type %q", configType)
	}
}

func dashboardCreateConcreteType(payload []byte) (reflect.Type, error) {
	schemaVersion, err := jsonFieldString(payload, "schemaVersion")
	if err != nil {
		return nil, fmt.Errorf("decode Dashboard create schemaVersion: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(schemaVersion)) {
	case "", "V1":
		return reflect.TypeOf(dashboardservicesdk.CreateV1DashboardDetails{}), nil
	default:
		return nil, fmt.Errorf("unsupported CreateDashboardDetails schemaVersion %q", schemaVersion)
	}
}

func connectionUpdateConcreteType(payload []byte) (reflect.Type, error) {
	connectionType, err := jsonFieldString(payload, "connectionType")
	if err != nil {
		return nil, fmt.Errorf("decode Connection update type: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(connectionType)) {
	case "MYSQL":
		return reflect.TypeOf(databasemigrationsdk.UpdateMysqlConnectionDetails{}), nil
	case "ORACLE":
		return reflect.TypeOf(databasemigrationsdk.UpdateOracleConnectionDetails{}), nil
	default:
		return nil, fmt.Errorf("unsupported UpdateConnectionDetails type %q", connectionType)
	}
}

func configUpdateConcreteType(payload []byte) (reflect.Type, error) {
	configType, err := jsonFieldString(payload, "configType")
	if err != nil {
		return nil, fmt.Errorf("decode Config update type: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(configType)) {
	case "AGENT":
		return reflect.TypeOf(apmconfigsdk.UpdateAgentConfigDetails{}), nil
	case "APDEX":
		return reflect.TypeOf(apmconfigsdk.UpdateApdexRulesDetails{}), nil
	case "MACS_APM_EXTENSION":
		return reflect.TypeOf(apmconfigsdk.UpdateMacsApmExtensionDetails{}), nil
	case "METRIC_GROUP":
		return reflect.TypeOf(apmconfigsdk.UpdateMetricGroupDetails{}), nil
	case "OPTIONS":
		return reflect.TypeOf(apmconfigsdk.UpdateOptionsDetails{}), nil
	case "SPAN_FILTER":
		return reflect.TypeOf(apmconfigsdk.UpdateSpanFilterDetails{}), nil
	default:
		return nil, fmt.Errorf("unsupported UpdateConfigDetails type %q", configType)
	}
}

func dashboardUpdateConcreteType(payload []byte) (reflect.Type, error) {
	schemaVersion, err := jsonFieldString(payload, "schemaVersion")
	if err != nil {
		return nil, fmt.Errorf("decode Dashboard update schemaVersion: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(schemaVersion)) {
	case "", "V1":
		return reflect.TypeOf(dashboardservicesdk.UpdateV1DashboardDetails{}), nil
	default:
		return nil, fmt.Errorf("unsupported UpdateDashboardDetails schemaVersion %q", schemaVersion)
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

	token := requestRetryToken(resource, requestStruct.Type())
	if token == "" {
		return
	}
	_ = assignField(field, token)
}

func requestRetryToken(resource any, requestType reflect.Type) string {
	token := resourceRetryToken(resource)
	if token == "" || isCreateRequestType(requestType) {
		return token
	}

	operation := ""
	if requestType != nil {
		operation = requestType.PkgPath() + "." + requestType.Name()
	}
	sum := sha256.Sum256([]byte(token + "\x00" + operation))
	return fmt.Sprintf("%x", sum[:16])
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

func isCreateRequestType(requestType reflect.Type) bool {
	if requestType == nil {
		return false
	}
	name := strings.ToLower(requestType.Name())
	return strings.HasPrefix(name, "create") || strings.HasPrefix(name, "launch")
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
