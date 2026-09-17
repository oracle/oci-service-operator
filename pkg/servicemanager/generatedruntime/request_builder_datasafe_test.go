/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"reflect"
	"testing"

	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
)

func TestConvertValueSupportsDataSafeSensitiveTypePolymorphicBodies(t *testing.T) {
	tests := []struct {
		name       string
		payload    map[string]any
		targetType reflect.Type
		wantType   reflect.Type
	}{
		{
			name:       "create sensitive type pattern",
			payload:    map[string]any{"entityType": "SENSITIVE_TYPE", "compartmentId": "ocid1.compartment.oc1..test", "namePattern": ".*"},
			targetType: sensitiveTypeCreateDetailsType,
			wantType:   reflect.TypeOf(datasafesdk.CreateSensitiveTypePatternDetails{}),
		},
		{
			name:       "create sensitive category",
			payload:    map[string]any{"entityType": "SENSITIVE_CATEGORY", "compartmentId": "ocid1.compartment.oc1..test"},
			targetType: sensitiveTypeCreateDetailsType,
			wantType:   reflect.TypeOf(datasafesdk.CreateSensitiveCategoryDetails{}),
		},
		{
			name:       "update sensitive type pattern",
			payload:    map[string]any{"entityType": "SENSITIVE_TYPE", "description": "updated"},
			targetType: sensitiveTypeUpdateDetailsType,
			wantType:   reflect.TypeOf(datasafesdk.UpdateSensitiveTypePatternDetails{}),
		},
		{
			name:       "update sensitive category",
			payload:    map[string]any{"entityType": "SENSITIVE_CATEGORY", "description": "updated"},
			targetType: sensitiveTypeUpdateDetailsType,
			wantType:   reflect.TypeOf(datasafesdk.UpdateSensitiveCategoryDetails{}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			converted, err := convertValue(test.payload, test.targetType)
			if err != nil {
				t.Fatal(err)
			}
			if got := reflect.TypeOf(converted.Interface()); got != test.wantType {
				t.Fatalf("converted type = %v, want %v", got, test.wantType)
			}
		})
	}
}

func TestConvertValueRejectsUnsupportedDataSafeSensitiveTypeDiscriminator(t *testing.T) {
	_, err := convertValue(map[string]any{"entityType": "UNKNOWN"}, sensitiveTypeCreateDetailsType)
	if err == nil {
		t.Fatal("expected unsupported Data Safe SensitiveType discriminator error")
	}
}
