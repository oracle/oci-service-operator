package generatedruntime

import "testing"

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
