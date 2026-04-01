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

func TestGenerateFullSyncCleansStaleGeneratorOwnedOutputsForInactiveServicesAndExcludedKinds(t *testing.T) {
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

	preservedDeepCopyFile := filepath.Join(outputRoot, "api", "mysql", "v1beta1", "zz_generated.deepcopy.go")
	if err := os.MkdirAll(filepath.Dir(preservedDeepCopyFile), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(preservedDeepCopyFile), err)
	}
	if err := os.WriteFile(preservedDeepCopyFile, []byte("// preserved deepcopy output\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", preservedDeepCopyFile, err)
	}

	preservedPackageFile := filepath.Join(outputRoot, "packages", "mysql", "install", "generated", "crd", "bases", "mysql.oracle.com_report.yaml")
	if err := os.MkdirAll(filepath.Dir(preservedPackageFile), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(preservedPackageFile), err)
	}
	if err := os.WriteFile(preservedPackageFile, []byte("preserved manifest output\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", preservedPackageFile, err)
	}

	staleMySQLOldVersionSample := filepath.Join(outputRoot, "config", "samples", "mysql_v1alpha1_widget.yaml")
	if err := os.MkdirAll(filepath.Dir(staleMySQLOldVersionSample), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(staleMySQLOldVersionSample), err)
	}
	if err := os.WriteFile(staleMySQLOldVersionSample, []byte("apiVersion: mysql.oracle.com/v1alpha1\nkind: Widget\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", staleMySQLOldVersionSample, err)
	}

	staleIdentityOldVersionSample := filepath.Join(outputRoot, "config", "samples", "identity_v1alpha1_widget.yaml")
	if err := os.MkdirAll(filepath.Dir(staleIdentityOldVersionSample), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(staleIdentityOldVersionSample), err)
	}
	if err := os.WriteFile(staleIdentityOldVersionSample, []byte("apiVersion: identity.oracle.com/v1alpha1\nkind: Widget\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", staleIdentityOldVersionSample, err)
	}

	activeServices, err := cfg.SelectServices("", true)
	if err != nil {
		t.Fatalf("SelectServices(--all) error = %v", err)
	}
	if _, err := pipeline.Generate(context.Background(), cfg, activeServices, Options{
		OutputRoot: outputRoot,
		Overwrite:  true,
		FullSync:   true,
	}); err != nil {
		t.Fatalf("Generate() full-sync error = %v", err)
	}

	assertPathNotExists(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "report_types.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "controllers", "mysql", "report_controller.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "report", "report_serviceclient.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "report", "report_servicemanager.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_report.yaml"))
	assertPathNotExists(t, filepath.Join(outputRoot, "api", "identity", "v1beta1", "widget_types.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "controllers", "identity", "widget_controller.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "identity", "widget", "widget_serviceclient.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "identity", "widget", "widget_servicemanager.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "internal", "registrations", "identity_generated.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "packages", "identity", "metadata.env"))
	assertPathNotExists(t, filepath.Join(outputRoot, "packages", "identity", "install", "kustomization.yaml"))
	assertPathNotExists(t, filepath.Join(outputRoot, "config", "samples", "identity_v1beta1_widget.yaml"))
	assertPathNotExists(t, staleMySQLOldVersionSample)
	assertPathNotExists(t, staleIdentityOldVersionSample)

	assertPathExists(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "widget_types.go"))
	assertPathExists(t, filepath.Join(outputRoot, "controllers", "mysql", "widget_controller.go"))
	assertPathExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "widget", "widget_serviceclient.go"))
	assertPathExists(t, filepath.Join(outputRoot, "packages", "mysql", "metadata.env"))
	assertPathExists(t, filepath.Join(outputRoot, "internal", "registrations", "mysql_generated.go"))
	assertPathExists(t, filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_widget.yaml"))
	assertPathExists(t, preservedDeepCopyFile)
	assertPathExists(t, preservedPackageFile)
	assertFileDoesNotContain(t, filepath.Join(outputRoot, "internal", "registrations", "mysql_generated.go"), []string{"ReportReconciler"})
	assertFileDoesNotContain(t, filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"), []string{
		"mysql_v1alpha1_widget.yaml",
		"identity_v1alpha1_widget.yaml",
	})
	if got := readFile(t, preservedDeepCopyFile); got != "// preserved deepcopy output\n" {
		t.Fatalf("deepcopy companion content = %q, want preserved content", got)
	}
	if got := readFile(t, preservedPackageFile); got != "preserved manifest output\n" {
		t.Fatalf("package install/generated content = %q, want preserved content", got)
	}
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
