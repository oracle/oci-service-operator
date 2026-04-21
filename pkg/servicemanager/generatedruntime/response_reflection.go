/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func responseBody(response any) (any, bool) {
	if response == nil {
		return nil, false
	}

	value, ok := indirectValue(reflect.ValueOf(response))
	if !ok {
		return nil, false
	}
	if value.Kind() != reflect.Struct {
		return response, true
	}

	if !strings.HasSuffix(value.Type().Name(), "Response") {
		return value.Interface(), true
	}
	return responseStructBody(value)
}

func responseStructBody(value reflect.Value) (any, bool) {
	typ := value.Type()
	var fallback reflect.Value
	for i := 0; i < value.NumField(); i++ {
		fieldType := typ.Field(i)
		fieldValue := value.Field(i)

		body, ok := taggedResponseBody(fieldType, fieldValue)
		if ok {
			return body, body != nil
		}
		if shouldSkipResponseFallback(fieldType) || fallback.IsValid() {
			continue
		}
		fallback = fieldValue
	}

	if fallback.IsValid() {
		return fallback.Interface(), true
	}
	return nil, false
}

func taggedResponseBody(fieldType reflect.StructField, fieldValue reflect.Value) (any, bool) {
	if !fieldType.IsExported() || fieldType.Tag.Get("presentIn") != "body" {
		return nil, false
	}
	if fieldValue.Kind() != reflect.Pointer {
		return fieldValue.Interface(), true
	}
	if fieldValue.IsNil() {
		return nil, true
	}
	return fieldValue.Interface(), true
}

func shouldSkipResponseFallback(fieldType reflect.StructField) bool {
	if !fieldType.IsExported() {
		return true
	}
	return fieldType.Name == "RawResponse" || strings.HasPrefix(fieldType.Name, "Opc") || fieldType.Name == "Etag"
}

func mergeResponseIntoStatus(resource any, response any) error {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return nil
	}

	statusValue, err := statusStruct(resource)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal OCI response body: %w", err)
	}
	if err := json.Unmarshal(payload, statusValue.Addr().Interface()); err != nil {
		return fmt.Errorf("project OCI response body into status: %w", err)
	}
	return nil
}

func normalizeOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isReadNotFound(err error) bool {
	if errors.Is(err, errResourceNotFound) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func isDeleteNotFound(err error) bool {
	if errors.Is(err, errResourceNotFound) {
		return true
	}
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func isRetryableDeleteConflict(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	if classification.HTTPStatusCode != 409 {
		return false
	}

	switch classification.ErrorCode {
	case errorutil.IncorrectState, "ExternalServerIncorrectState":
		return true
	default:
		return false
	}
}

func osokStatus(resource any) (*shared.OSOKStatus, error) {
	statusValue, err := statusStruct(resource)
	if err != nil {
		return nil, err
	}

	field := statusValue.FieldByName("OsokStatus")
	if !field.IsValid() || !field.CanAddr() {
		return nil, fmt.Errorf("resource %T does not expose Status.OsokStatus", resource)
	}

	status, ok := field.Addr().Interface().(*shared.OSOKStatus)
	if !ok {
		return nil, fmt.Errorf("resource %T Status.OsokStatus has unexpected type %T", resource, field.Addr().Interface())
	}
	return status, nil
}

func statusStruct(resource any) (reflect.Value, error) {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return reflect.Value{}, err
	}
	field, ok := fieldValue(resourceValue, "Status")
	if !ok {
		return reflect.Value{}, fmt.Errorf("resource %T does not expose Status", resource)
	}
	return field, nil
}

func resourceStruct(resource any) (reflect.Value, error) {
	value := reflect.ValueOf(resource)
	if !value.IsValid() {
		return reflect.Value{}, fmt.Errorf("resource is nil")
	}
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return reflect.Value{}, fmt.Errorf("expected pointer resource, got %T", resource)
	}
	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("expected pointer to struct resource, got %T", resource)
	}
	return value, nil
}

func fieldValue(value reflect.Value, name string) (reflect.Value, bool) {
	field := value.FieldByName(name)
	if !field.IsValid() {
		return reflect.Value{}, false
	}
	return field, true
}

func fieldInterface(value reflect.Value, name string) any {
	field, ok := fieldValue(value, name)
	if !ok || !field.IsValid() {
		return nil
	}
	return field.Interface()
}

func lookupMetadataString(value reflect.Value, fieldName string) string {
	field, ok := fieldValue(value, fieldName)
	if !ok || field.Kind() != reflect.String {
		return ""
	}
	return field.String()
}

func responseLifecycleState(response any) string {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return ""
	}
	return firstNonEmpty(jsonMap(body), "lifecycleState", "status")
}

func listBodyStruct(body any) (reflect.Value, error) {
	value, ok := indirectValue(reflect.ValueOf(body))
	if !ok {
		return reflect.Value{}, errResourceNotFound
	}
	if value.Kind() != reflect.Struct && value.Kind() != reflect.Slice {
		return reflect.Value{}, fmt.Errorf("OCI list body must be a struct or slice, got %T", body)
	}
	return value, nil
}

func configuredListItems(value reflect.Value, fieldName string) ([]any, bool, error) {
	fieldName = strings.TrimSpace(fieldName)
	if fieldName == "" {
		return nil, false, nil
	}

	itemsField := value.FieldByName(fieldName)
	if !itemsField.IsValid() {
		return nil, true, fmt.Errorf("OCI list body does not expose %s", fieldName)
	}
	if itemsField.Kind() != reflect.Slice {
		return nil, true, fmt.Errorf("OCI list body %s is not a slice", fieldName)
	}
	return sliceValues(itemsField), true, nil
}

func structSliceField(value reflect.Value, fieldName string) ([]any, bool) {
	itemsField := value.FieldByName(fieldName)
	if !itemsField.IsValid() || itemsField.Kind() != reflect.Slice {
		return nil, false
	}
	return sliceValues(itemsField), true
}

func firstSliceListItems(value reflect.Value) ([]any, bool) {
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() == reflect.Slice {
			return sliceValues(field), true
		}
	}
	return nil, false
}
