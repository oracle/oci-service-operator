/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"reflect"
	"testing"

	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type jsonValueResolutionResource struct {
	Spec jsonValueResolutionSpec
}

type jsonValueResolutionSpec struct {
	Array  shared.JSONValue `json:"array"`
	Object shared.JSONValue `json:"object"`
}

func TestResolveSpecValueWithBoolFieldsPreservesNestedFalseClusterBooleans(t *testing.T) {
	t.Parallel()
	resource := &containerenginev1beta1.Cluster{Spec: containerenginev1beta1.ClusterSpec{EndpointConfig: containerenginev1beta1.ClusterEndpointConfig{IsPublicIpEnabled: false}, Options: containerenginev1beta1.ClusterOptions{AddOns: containerenginev1beta1.ClusterOptionsAddOns{IsTillerEnabled: false}}, ImagePolicyConfig: containerenginev1beta1.ClusterImagePolicyConfig{IsPolicyEnabled: false}}}
	resolved, err := ResolveSpecValueWithBoolFields(resource, context.Background(), nil, "default")
	if err != nil {
		t.Fatalf("ResolveSpecValueWithBoolFields() error = %v", err)
	}
	values, ok := resolved.(map[string]any)
	if !ok {
		t.Fatalf("ResolveSpecValueWithBoolFields() type = %T, want map[string]any", resolved)
	}
	for _, path := range []string{"endpointConfig.isPublicIpEnabled", "options.addOns.isTillerEnabled", "imagePolicyConfig.isPolicyEnabled"} {
		got, ok := lookupValueByPath(values, path)
		if !ok {
			t.Fatalf("ResolveSpecValueWithBoolFields() omitted %s", path)
		}
		boolValue, ok := got.(bool)
		if !ok || boolValue {
			t.Fatalf("ResolveSpecValueWithBoolFields() %s = %#v, want false", path, got)
		}
	}
}

func TestResolveSpecValueWithBoolFieldsPreservesArbitraryJSONShapes(t *testing.T) {
	t.Parallel()
	resource := &jsonValueResolutionResource{Spec: jsonValueResolutionSpec{
		Array:  shared.JSONValue{Raw: []byte(`[]`)},
		Object: shared.JSONValue{Raw: []byte(`{"enabled":false}`)},
	}}

	resolved, err := ResolveSpecValueWithBoolFields(resource, context.Background(), nil, "default")
	if err != nil {
		t.Fatalf("ResolveSpecValueWithBoolFields() error = %v", err)
	}
	values, ok := resolved.(map[string]any)
	if !ok {
		t.Fatalf("ResolveSpecValueWithBoolFields() type = %T, want map[string]any", resolved)
	}
	if got := values["array"]; !reflect.DeepEqual(got, []any{}) {
		t.Fatalf("resolved array = %#v, want empty JSON array", got)
	}
	if got := values["object"]; !reflect.DeepEqual(got, map[string]any{"enabled": false}) {
		t.Fatalf("resolved object = %#v, want JSON object", got)
	}
}
