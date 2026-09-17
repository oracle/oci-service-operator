/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package generatedruntime

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	autoscalingsdk "github.com/oracle/oci-go-sdk/v65/autoscaling"
	dataintegrationsdk "github.com/oracle/oci-go-sdk/v65/dataintegration"
)

func TestConvertCompositePolymorphicRequestBodies(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		payload    string
		targetType reflect.Type
		wantType   reflect.Type
	}{
		{name: "autoscaling create", payload: `{"policyType":"threshold","rules":[]}`, targetType: autoScalingPolicyCreateDetailsType, wantType: reflect.TypeOf(autoscalingsdk.CreateThresholdPolicyDetails{})},
		{name: "autoscaling update", payload: `{"policyType":"scheduled","displayName":"updated"}`, targetType: autoScalingPolicyUpdateDetailsType, wantType: reflect.TypeOf(autoscalingsdk.UpdateScheduledPolicyDetails{})},
		{name: "data integration connection create", payload: `{"modelType":"REST_NO_AUTH_CONNECTION","name":"mock","identifier":"MOCK"}`, targetType: dataIntegrationConnectionCreateType, wantType: reflect.TypeOf(dataintegrationsdk.CreateConnectionFromRestNoAuth{})},
		{name: "data integration connection update", payload: `{"modelType":"ORACLE_OBJECT_STORAGE_CONNECTION","name":"mock","identifier":"MOCK"}`, targetType: dataIntegrationConnectionUpdateType, wantType: reflect.TypeOf(dataintegrationsdk.UpdateConnectionFromObjectStorage{})},
		{name: "data integration data asset create", payload: `{"modelType":"ORACLE_OBJECT_STORAGE_DATA_ASSET","name":"mock","identifier":"MOCK"}`, targetType: dataIntegrationDataAssetCreateType, wantType: reflect.TypeOf(dataintegrationsdk.CreateDataAssetFromObjectStorage{})},
		{name: "data integration data asset update", payload: `{"modelType":"REST_DATA_ASSET","name":"mock","identifier":"MOCK"}`, targetType: dataIntegrationDataAssetUpdateType, wantType: reflect.TypeOf(dataintegrationsdk.UpdateDataAssetFromRest{})},
		{name: "data integration task create", payload: `{"modelType":"REST_TASK","name":"mock","identifier":"MOCK","registryMetadata":{}}`, targetType: dataIntegrationTaskCreateType, wantType: reflect.TypeOf(dataintegrationsdk.CreateTaskFromRestTask{})},
		{name: "data integration task update", payload: `{"modelType":"PIPELINE_TASK","name":"mock","identifier":"MOCK"}`, targetType: dataIntegrationTaskUpdateType, wantType: reflect.TypeOf(dataintegrationsdk.UpdateTaskFromPipelineTask{})},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			value, err := convertValue(mapFromCompositeJSON(t, test.payload), test.targetType)
			if err != nil {
				t.Fatal(err)
			}
			if got := reflect.TypeOf(value.Interface()); got != test.wantType {
				t.Fatalf("resolved type = %v, want %v", got, test.wantType)
			}
		})
	}
}

func TestConvertCompositePolymorphicRequestBodyRejectsUnknownDiscriminator(t *testing.T) {
	t.Parallel()
	_, err := convertValue(mapFromCompositeJSON(t, `{"modelType":"UNKNOWN"}`), dataIntegrationTaskCreateType)
	if err == nil || !strings.Contains(err.Error(), `unsupported Data Integration task modelType discriminator "UNKNOWN"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestPreserveCompositeUpdateDiscriminator(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		request       func() any
		field         RequestField
		discriminator string
		value         string
	}{
		{name: "autoscaling policy", request: func() any { return &autoscalingsdk.UpdateAutoScalingPolicyRequest{} }, field: RequestField{FieldName: "UpdateAutoScalingPolicyDetails", Contribution: "body"}, discriminator: "policyType", value: "scheduled"},
		{name: "data integration connection", request: func() any { return &dataintegrationsdk.UpdateConnectionRequest{} }, field: RequestField{FieldName: "UpdateConnectionDetails", Contribution: "body"}, discriminator: "modelType", value: "REST_NO_AUTH_CONNECTION"},
		{name: "data integration data asset", request: func() any { return &dataintegrationsdk.UpdateDataAssetRequest{} }, field: RequestField{FieldName: "UpdateDataAssetDetails", Contribution: "body"}, discriminator: "modelType", value: "ORACLE_OBJECT_STORAGE_DATA_ASSET"},
		{name: "data integration task", request: func() any { return &dataintegrationsdk.UpdateTaskRequest{} }, field: RequestField{FieldName: "UpdateTaskDetails", Contribution: "body"}, discriminator: "modelType", value: "REST_TASK"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			body := map[string]any{"description": "updated"}
			preservePolymorphicUpdateDiscriminator(body, map[string]any{test.discriminator: test.value}, &Operation{NewRequest: test.request, Fields: []RequestField{test.field}})
			if body[test.discriminator] != test.value {
				t.Fatalf("body = %#v", body)
			}
		})
	}
}

func TestConvertValueWrapsScalarQueryValueForSDKSlice(t *testing.T) {
	t.Parallel()
	converted, err := convertValue("identifier", reflect.TypeOf([]string{}))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"identifier"}
	if !reflect.DeepEqual(converted.Interface(), want) {
		t.Fatalf("converted value = %#v, want %#v", converted.Interface(), want)
	}
}

func mapFromCompositeJSON(t *testing.T, content string) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(content), &value); err != nil {
		t.Fatal(err)
	}
	return value
}
