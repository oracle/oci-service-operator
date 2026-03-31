/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratePreservesExistingSpecSurfaceFromSeparateRoot(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := ServiceConfig{
		Service:        "sample",
		SDKPackage:     "example.com/test/sdk",
		Group:          "sample",
		PackageProfile: PackageProfileCRDOnly,
	}

	preserveRoot := t.TempDir()
	writeExistingWidgetSpecSurface(t, preserveRoot)

	outputRoot := t.TempDir()
	pipeline := newTestGenerator(t)
	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot:                      outputRoot,
		Overwrite:                       true,
		PreserveExistingSpecSurfaceRoot: preserveRoot,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	widgetPath := filepath.Join(outputRoot, "api", "sample", "v1beta1", "widget_types.go")
	assertPreservedWidgetSurface(t, widgetPath)
}

const existingWidgetSpecSurface = `package v1beta1

// WidgetSpec defines the desired state of Widget.
type WidgetSpec struct {
	// The OCID of the widget compartment.
	// +kubebuilder:validation:Required
	CompartmentId string ` + "`json:\"compartmentId\"`" + `
	// Widget source details.
	// +kubebuilder:validation:Optional
	Source WidgetExistingSource ` + "`json:\"source,omitempty\"`" + `
}

// WidgetExistingSource defines nested fields for Widget.Source.
type WidgetExistingSource struct {
	// The widget source type.
	Type string ` + "`json:\"type,omitempty\"`" + `
}
`

func writeExistingWidgetSpecSurface(t *testing.T, preserveRoot string) {
	t.Helper()

	preserveDir := filepath.Join(preserveRoot, "api", "sample", "v1beta1")
	if err := os.MkdirAll(preserveDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", preserveDir, err)
	}
	if err := os.WriteFile(filepath.Join(preserveDir, "widget_types.go"), []byte(existingWidgetSpecSurface), 0o644); err != nil {
		t.Fatalf("WriteFile(widget_types.go) error = %v", err)
	}
}

func assertPreservedWidgetSurface(t *testing.T, widgetPath string) {
	t.Helper()

	surface, ok, err := loadExistingSpecSurface(widgetPath, "Widget")
	if err != nil {
		t.Fatalf("loadExistingSpecSurface(%s) error = %v", widgetPath, err)
	}
	if !ok {
		t.Fatalf("loadExistingSpecSurface(%s) returned ok=false", widgetPath)
	}

	assertFieldNamesPresent(t, "Widget spec fields", surface.SpecFields, "CompartmentId", "Source")
	if len(surface.SpecFields) != 2 {
		t.Fatalf("Widget spec fields = %#v, want only CompartmentId and Source", surface.SpecFields)
	}
	if len(surface.SpecHelperTypes) != 1 || surface.SpecHelperTypes[0].Name != "WidgetExistingSource" {
		t.Fatalf("Widget helper types = %#v, want WidgetExistingSource only", surface.SpecHelperTypes)
	}
	assertHelperTypeFields(t, surface.SpecHelperTypes[0], "Type")

	assertFileContains(t, widgetPath, []string{
		"Source WidgetExistingSource `json:\"source,omitempty\"`",
		"type WidgetExistingSource struct {",
		"LifecycleState",
		"TimeUpdated",
	})
	assertFileDoesNotContain(t, widgetPath, []string{
		"DisplayName string `json:\"displayName\"`",
		"Name string `json:\"name,omitempty\"`",
		"Count int `json:\"count,omitempty\"`",
		"Enabled bool `json:\"enabled,omitempty\"`",
		"Labels map[string]string `json:\"labels,omitempty\"`",
		"Mode ModeEnum `json:\"mode,omitempty\"`",
		"CreatedAt",
		"type WidgetSource struct {",
	})
}
