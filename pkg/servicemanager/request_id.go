/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicemanager

import (
	"errors"
	"reflect"
	"strings"

	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func SetOpcRequestID(status *shared.OSOKStatus, requestID string) {
	if status == nil {
		return
	}

	if requestID = strings.TrimSpace(requestID); requestID != "" {
		status.OpcRequestID = requestID
	}
}

func RecordResponseOpcRequestID(status *shared.OSOKStatus, response any) {
	SetOpcRequestID(status, ResponseOpcRequestID(response))
}

func RecordErrorOpcRequestID(status *shared.OSOKStatus, err error) {
	SetOpcRequestID(status, ErrorOpcRequestID(err))
}

type opcRequestIDGetter interface {
	GetOpcRequestID() string
}

func ResponseOpcRequestID(response any) string {
	if response == nil {
		return ""
	}

	value, ok := indirectResponseValue(reflect.ValueOf(response))
	if !ok || value.Kind() != reflect.Struct {
		return ""
	}

	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := typ.Field(i)
		if !fieldType.IsExported() || !isRequestIDHeaderField(fieldType) {
			continue
		}
		if requestID := responseStringFieldValue(value.Field(i)); requestID != "" {
			return requestID
		}
	}

	return ""
}

func ErrorOpcRequestID(err error) string {
	if err == nil {
		return ""
	}

	var getter opcRequestIDGetter
	if errors.As(err, &getter) {
		return strings.TrimSpace(getter.GetOpcRequestID())
	}

	value, ok := indirectResponseValue(reflect.ValueOf(err))
	if !ok || value.Kind() != reflect.Struct {
		return ""
	}

	for _, fieldName := range []string{"OpcRequestID", "OpcRequestId"} {
		field := value.FieldByName(fieldName)
		if field.IsValid() && field.Kind() == reflect.String {
			return strings.TrimSpace(field.String())
		}
	}

	return ""
}

func indirectResponseValue(value reflect.Value) (reflect.Value, bool) {
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}
	return value, value.IsValid()
}

func isRequestIDHeaderField(fieldType reflect.StructField) bool {
	return fieldType.Name == "OpcRequestId" ||
		fieldType.Name == "OpcRequestID" ||
		(fieldType.Tag.Get("presentIn") == "header" && fieldType.Tag.Get("name") == "opc-request-id")
}

func responseStringFieldValue(value reflect.Value) string {
	value, ok := indirectResponseValue(value)
	if !ok || value.Kind() != reflect.String {
		return ""
	}
	return strings.TrimSpace(value.String())
}
