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

func TestGenerateDefaultActiveSurfaceFiltersKindsBeforeRendering(t *testing.T) {
	t.Parallel()

	cfg := selectionCleanupTestConfig()
	services, err := cfg.SelectServices("", true)
	if err != nil {
		t.Fatalf("SelectServices(--all) error = %v", err)
	}

	outputRoot := t.TempDir()
	pipeline := newSelectionCleanupTestGenerator(t)
	result, err := pipeline.Generate(context.Background(), cfg, services, Options{
		OutputRoot: outputRoot,
		Overwrite:  true,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	assertGeneratedServiceCounts(t, result.Generated, map[string]int{"mysql": 1})
	assertPathExists(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "widget_types.go"))
	assertPathExists(t, filepath.Join(outputRoot, "controllers", "mysql", "widget_controller.go"))
	assertPathExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "widget", "widget_serviceclient.go"))
	assertPathExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "widget", "widget_servicemanager.go"))
	assertPathExists(t, filepath.Join(outputRoot, "internal", "registrations", "mysql_generated.go"))
	assertPathExists(t, filepath.Join(outputRoot, "packages", "mysql", "metadata.env"))
	assertPathExists(t, filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_widget.yaml"))

	assertPathNotExists(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "report_types.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "dbsystem_types.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "controllers", "mysql", "report_controller.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "report", "report_serviceclient.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_report.yaml"))
	assertPathNotExists(t, filepath.Join(outputRoot, "api", "identity", "v1beta1", "widget_types.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "internal", "registrations", "identity_generated.go"))
	assertFileDoesNotContain(t, filepath.Join(outputRoot, "internal", "registrations", "mysql_generated.go"), []string{"ReportReconciler", "DbSystemReconciler"})
}

func TestGenerateExplicitServiceOverrideIncludesDisabledServiceSurface(t *testing.T) {
	t.Parallel()

	cfg := selectionCleanupTestConfig()
	services, err := cfg.SelectServices("identity", false)
	if err != nil {
		t.Fatalf("SelectServices(--service identity) error = %v", err)
	}
	if got := services[0].SelectedKinds(); got != nil {
		t.Fatalf("identity SelectedKinds() = %v, want nil", got)
	}

	outputRoot := t.TempDir()
	pipeline := newSelectionCleanupTestGenerator(t)
	result, err := pipeline.Generate(context.Background(), cfg, services, Options{
		OutputRoot: outputRoot,
		Overwrite:  true,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	assertGeneratedServiceCounts(t, result.Generated, map[string]int{"identity": 5})
	for _, relativePath := range []string{
		"api/identity/v1beta1/widget_types.go",
		"api/identity/v1beta1/report_types.go",
		"api/identity/v1beta1/reportbyname_types.go",
		"api/identity/v1beta1/dbsystem_types.go",
		"controllers/identity/widget_controller.go",
		"pkg/servicemanager/identity/widget/widget_serviceclient.go",
		"internal/registrations/identity_generated.go",
		"config/samples/identity_v1beta1_widget.yaml",
	} {
		assertPathExists(t, filepath.Join(outputRoot, relativePath))
	}
}

func TestGenerateFullSyncCleansStaleOutputsForInactiveServicesAndExcludedKinds(t *testing.T) {
	t.Parallel()

	cfg := selectionCleanupTestConfig()
	outputRoot := t.TempDir()
	pipeline := newSelectionCleanupTestGenerator(t)

	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{cfg.Services[0], cfg.Services[1]}, Options{
		OutputRoot: outputRoot,
		Overwrite:  true,
	}); err != nil {
		t.Fatalf("seed Generate() error = %v", err)
	}

	stalePackageFile := filepath.Join(outputRoot, "packages", "mysql", "install", "generated", "crd", "bases", "mysql.oracle.com_report.yaml")
	if err := os.MkdirAll(filepath.Dir(stalePackageFile), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(stalePackageFile), err)
	}
	if err := os.WriteFile(stalePackageFile, []byte("stale package output\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", stalePackageFile, err)
	}

	activeServices, err := cfg.SelectServices("", true)
	if err != nil {
		t.Fatalf("SelectServices(--all) error = %v", err)
	}
	if _, err := pipeline.Generate(context.Background(), cfg, activeServices, Options{
		OutputRoot: outputRoot,
		Overwrite:  true,
	}); err != nil {
		t.Fatalf("Generate() full-sync error = %v", err)
	}

	assertPathNotExists(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "report_types.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "controllers", "mysql", "report_controller.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "report", "report_serviceclient.go"))
	assertPathNotExists(t, stalePackageFile)
	assertPathNotExists(t, filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_report.yaml"))

	assertPathExists(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "widget_types.go"))
	assertPathExists(t, filepath.Join(outputRoot, "api", "identity", "v1beta1", "widget_types.go"))
	assertPathExists(t, filepath.Join(outputRoot, "controllers", "mysql", "widget_controller.go"))
	assertPathExists(t, filepath.Join(outputRoot, "controllers", "identity", "widget_controller.go"))
	assertPathExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "widget", "widget_serviceclient.go"))
	assertPathExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "identity", "widget", "widget_serviceclient.go"))
	assertPathExists(t, filepath.Join(outputRoot, "packages", "mysql", "metadata.env"))
	assertPathExists(t, filepath.Join(outputRoot, "internal", "registrations", "identity_generated.go"))
	assertPathExists(t, filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_widget.yaml"))
	assertPathExists(t, filepath.Join(outputRoot, "config", "samples", "identity_v1beta1_widget.yaml"))
	assertFileDoesNotContain(t, filepath.Join(outputRoot, "internal", "registrations", "mysql_generated.go"), []string{"ReportReconciler"})
}

func selectionCleanupTestConfig() *Config {
	return &Config{
		SchemaVersion:  "v1alpha1",
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		PackageProfiles: map[string]PackageProfile{
			"controller-backed": {Description: "runtime-integrated groups"},
		},
		Services: []ServiceConfig{
			{
				Service:        "mysql",
				SDKPackage:     "example.com/test/sdk",
				Group:          "mysql",
				PackageProfile: "controller-backed",
				Selection:      selectionExplicit(true, "Widget"),
				Generation:     selectionCleanupGeneratedRuntime(),
			},
			{
				Service:        "identity",
				SDKPackage:     "example.com/test/sdk",
				Group:          "identity",
				PackageProfile: "controller-backed",
				Selection:      selectionAll(false),
				Generation:     selectionCleanupGeneratedRuntime(),
			},
		},
	}
}

func selectionCleanupGeneratedRuntime() GenerationConfig {
	return GenerationConfig{
		Controller:     GenerationSurfaceConfig{Strategy: GenerationStrategyGenerated},
		ServiceManager: GenerationSurfaceConfig{Strategy: GenerationStrategyGenerated},
		Registration:   GenerationSurfaceConfig{Strategy: GenerationStrategyGenerated},
		Webhooks:       GenerationSurfaceConfig{Strategy: GenerationStrategyNone},
	}
}

func newSelectionCleanupTestGenerator(t *testing.T) *Generator {
	t.Helper()

	return &Generator{
		discoverer: &Discoverer{
			resolveDir: func(context.Context, string) (string, error) {
				return sampleSDKDir(t), nil
			},
		},
		renderer: NewRenderer(),
	}
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
}

func assertPathNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want not exist", path, err)
	}
}
