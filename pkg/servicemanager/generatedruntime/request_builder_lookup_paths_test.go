package generatedruntime

import (
	"reflect"
	"testing"

	objectstoragev1beta1 "github.com/oracle/oci-service-operator/api/objectstorage/v1beta1"
)

func TestExplicitRequestValueUsesLookupPaths(t *testing.T) {
	t.Parallel()

	values := map[string]any{
		"name":      "spec-name",
		"namespace": "spec-namespace",
	}

	field := RequestField{
		FieldName:    "BucketName",
		RequestName:  "bucketName",
		Contribution: "path",
		LookupPaths:  []string{"status.name", "spec.name", "name"},
	}

	rawValue, ok := explicitRequestValue(values, field, "")
	if !ok {
		t.Fatal("explicitRequestValue() did not resolve lookup paths")
	}
	if got, ok := rawValue.(string); !ok || got != "spec-name" {
		t.Fatalf("explicitRequestValue() = %#v, want spec-name", rawValue)
	}
}

func TestExplicitRequestValuePrefersResolvedObjectStorageNamespaceOverMetadataNamespace(t *testing.T) {
	t.Parallel()

	values := map[string]any{
		"namespace":     "iddevjmhjw0n",
		"namespaceName": "default",
	}

	field := RequestField{
		FieldName:    "NamespaceName",
		RequestName:  "namespaceName",
		Contribution: "path",
		LookupPaths:  []string{"status.namespace", "spec.namespace", "namespace"},
	}

	rawValue, ok := explicitRequestValue(values, field, "")
	if !ok {
		t.Fatal("explicitRequestValue() did not resolve namespace lookup paths")
	}
	if got, ok := rawValue.(string); !ok || got != "iddevjmhjw0n" {
		t.Fatalf("explicitRequestValue() = %#v, want iddevjmhjw0n", rawValue)
	}
}

func TestExplicitRequestValuePrefersResolvedNamespaceWithoutLookupPaths(t *testing.T) {
	t.Parallel()

	values := map[string]any{
		"namespace":     "iddevjmhjw0n",
		"namespaceName": "default",
	}

	field := RequestField{
		FieldName:    "NamespaceName",
		RequestName:  "namespaceName",
		Contribution: "path",
	}

	rawValue, ok := explicitRequestValue(values, field, "")
	if !ok {
		t.Fatal("explicitRequestValue() did not resolve namespace")
	}
	if got, ok := rawValue.(string); !ok || got != "iddevjmhjw0n" {
		t.Fatalf("explicitRequestValue() = %#v, want iddevjmhjw0n", rawValue)
	}
}

func TestHeuristicRequestValuePrefersResolvedNamespaceOverMetadataNamespace(t *testing.T) {
	t.Parallel()

	values := map[string]any{
		"namespace":     "iddevjmhjw0n",
		"namespaceName": "default",
	}

	requestField, ok := reflect.TypeOf(struct {
		NamespaceName string `contributesTo:"path" name:"namespaceName"`
	}{}).FieldByName("NamespaceName")
	if !ok {
		t.Fatal("FieldByName(NamespaceName) = false, want true")
	}

	rawValue, ok := heuristicRequestValue(values, requestField, "", nil)
	if !ok {
		t.Fatal("heuristicRequestValue() did not resolve namespace")
	}
	if got, ok := rawValue.(string); !ok || got != "iddevjmhjw0n" {
		t.Fatalf("heuristicRequestValue() = %#v, want iddevjmhjw0n", rawValue)
	}
}

func TestExplicitRequestValueFallsBackWithoutLookupPaths(t *testing.T) {
	t.Parallel()

	values := map[string]any{
		"name": "metadata-name",
	}

	field := RequestField{
		FieldName:    "Name",
		RequestName:  "name",
		Contribution: "path",
	}

	rawValue, ok := explicitRequestValue(values, field, "")
	if !ok {
		t.Fatal("explicitRequestValue() unexpectedly failed")
	}
	if got, ok := rawValue.(string); !ok || got != "metadata-name" {
		t.Fatalf("explicitRequestValue() = %#v, want metadata-name", rawValue)
	}
}

func TestExplicitRequestValuePrefersObservedBucketNameOverDesiredSpecName(t *testing.T) {
	t.Parallel()

	resource := &objectstoragev1beta1.Bucket{
		Spec: objectstoragev1beta1.BucketSpec{
			Name:          "spec-name",
			CompartmentId: "ocid1.compartment.oc1..example",
		},
		Status: objectstoragev1beta1.BucketStatus{
			Name: "status-name",
		},
	}

	values, err := lookupValues(resource)
	if err != nil {
		t.Fatalf("lookupValues() error = %v", err)
	}

	field := RequestField{
		FieldName:    "BucketName",
		RequestName:  "bucketName",
		Contribution: "path",
		LookupPaths:  []string{"status.name", "spec.name", "name"},
	}

	rawValue, ok := explicitRequestValue(values, field, "")
	if !ok {
		t.Fatal("explicitRequestValue() did not resolve observed bucket name")
	}
	if got, ok := rawValue.(string); !ok || got != "status-name" {
		t.Fatalf("explicitRequestValue() = %#v, want status-name", rawValue)
	}
}

func TestExplicitRequestValuePrefersObservedBucketNamespaceOverDesiredSpecNamespace(t *testing.T) {
	t.Parallel()

	resource := &objectstoragev1beta1.Bucket{
		Spec: objectstoragev1beta1.BucketSpec{
			Name:          "bucket-name",
			CompartmentId: "ocid1.compartment.oc1..example",
			Namespace:     "spec-namespace",
		},
		Status: objectstoragev1beta1.BucketStatus{
			Namespace: "status-namespace",
		},
	}

	values, err := lookupValues(resource)
	if err != nil {
		t.Fatalf("lookupValues() error = %v", err)
	}

	field := RequestField{
		FieldName:    "NamespaceName",
		RequestName:  "namespaceName",
		Contribution: "path",
		LookupPaths:  []string{"status.namespace", "spec.namespace", "namespace"},
	}

	rawValue, ok := explicitRequestValue(values, field, "")
	if !ok {
		t.Fatal("explicitRequestValue() did not resolve observed namespace")
	}
	if got, ok := rawValue.(string); !ok || got != "status-namespace" {
		t.Fatalf("explicitRequestValue() = %#v, want status-namespace", rawValue)
	}
}
