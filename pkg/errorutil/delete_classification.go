/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package errorutil

import (
	"errors"
	"reflect"
	"strconv"

	"github.com/oracle/oci-go-sdk/v65/common"
)

type DeleteErrorClassification struct {
	HTTPStatusCode int
	ErrorCode      string
	NormalizedType string
}

func ClassifyDeleteError(err error) DeleteErrorClassification {
	classification := DeleteErrorClassification{}
	if err == nil {
		return classification
	}

	normalized := err
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		classification.HTTPStatusCode = serviceErr.GetHTTPStatusCode()
		classification.ErrorCode = serviceErr.GetCode()
		if _, normalizedErr := OciErrorTypeResponse(err); normalizedErr != nil {
			normalized = normalizedErr
		}
	}

	if statusCode, errorCode, ok := structuredOCIErrorFields(normalized); ok {
		if classification.HTTPStatusCode == 0 {
			classification.HTTPStatusCode = statusCode
		}
		if classification.ErrorCode == "" {
			classification.ErrorCode = errorCode
		}
	}

	classification.NormalizedType = typeName(normalized)
	return classification
}

func (c DeleteErrorClassification) HTTPStatusCodeString() string {
	if c.HTTPStatusCode == 0 {
		return "unknown"
	}
	return strconv.Itoa(c.HTTPStatusCode)
}

func (c DeleteErrorClassification) ErrorCodeString() string {
	if c.ErrorCode == "" {
		return "unknown"
	}
	return c.ErrorCode
}

func (c DeleteErrorClassification) NormalizedTypeString() string {
	if c.NormalizedType == "" {
		return "unknown"
	}
	return c.NormalizedType
}

func (c DeleteErrorClassification) IsUnambiguousNotFound() bool {
	return c.HTTPStatusCode == 404 && c.ErrorCode == NotFound
}

func (c DeleteErrorClassification) IsAuthShapedNotFound() bool {
	return c.HTTPStatusCode == 404 && c.ErrorCode == NotAuthorizedOrNotFound
}

func (c DeleteErrorClassification) IsConflict() bool {
	return c.HTTPStatusCode == 409
}

func structuredOCIErrorFields(err error) (int, string, bool) {
	value := reflect.ValueOf(err)
	if !value.IsValid() {
		return 0, "", false
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return 0, "", false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return 0, "", false
	}

	statusCode := value.FieldByName("HTTPStatusCode")
	errorCode := value.FieldByName("ErrorCode")
	if !statusCode.IsValid() || !errorCode.IsValid() {
		return 0, "", false
	}
	if statusCode.Kind() != reflect.Int || errorCode.Kind() != reflect.String {
		return 0, "", false
	}
	return int(statusCode.Int()), errorCode.String(), true
}

func typeName(err error) string {
	if err == nil {
		return ""
	}
	return reflect.TypeOf(err).String()
}
