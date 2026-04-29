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
	assertPathExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "widget", "widget_runtimehooks_generated.go"))
	assertPathExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "widget", "widget_serviceclient.go"))
	assertPathExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "widget", "widget_servicemanager.go"))
	assertPathExists(t, filepath.Join(outputRoot, "internal", "registrations", "mysql_generated.go"))
	assertPathExists(t, filepath.Join(outputRoot, "packages", "mysql", "metadata.env"))
	assertPathExists(t, filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_widget.yaml"))

	assertPathNotExists(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "report_types.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "dbsystem_types.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "controllers", "mysql", "report_controller.go"))
	assertPathNotExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "report", "report_runtimehooks_generated.go"))
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
		"pkg/servicemanager/identity/widget/widget_runtimehooks_generated.go",
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

	seedSelectionCleanupOutputs(t, pipeline, cfg, outputRoot)

	preservedDeepCopyFile := filepath.Join(outputRoot, "api", "mysql", "v1beta1", "zz_generated.deepcopy.go")
	preservedPackageFile := filepath.Join(outputRoot, "packages", "mysql", "install", "generated", "crd", "bases", "mysql.oracle.com_report.yaml")
	staleIdentityDeepCopyFile := filepath.Join(outputRoot, "api", "identity", "v1beta1", "zz_generated.deepcopy.go")
	staleIdentityPackageGeneratedFile := filepath.Join(outputRoot, "packages", "identity", "install", "generated", "crd", "bases", "identity.oracle.com_widget.yaml")
	staleMySQLOldVersionSample := filepath.Join(outputRoot, "config", "samples", "mysql_v1alpha1_widget.yaml")
	staleIdentityOldVersionSample := filepath.Join(outputRoot, "config", "samples", "identity_v1alpha1_widget.yaml")
	writeSelectionCleanupFile(t, preservedDeepCopyFile, "// preserved deepcopy output\n")
	writeSelectionCleanupFile(t, preservedPackageFile, "preserved manifest output\n")
	writeSelectionCleanupFile(t, staleIdentityDeepCopyFile, "// stale identity deepcopy output\n")
	writeSelectionCleanupFile(t, staleIdentityPackageGeneratedFile, "stale identity package manifest\n")
	writeSelectionCleanupFile(t, staleMySQLOldVersionSample, "apiVersion: mysql.oracle.com/v1alpha1\nkind: Widget\n")
	writeSelectionCleanupFile(t, staleIdentityOldVersionSample, "apiVersion: identity.oracle.com/v1alpha1\nkind: Widget\n")

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

	assertPathsNotExist(t, []string{
		filepath.Join(outputRoot, "api", "mysql", "v1beta1", "report_types.go"),
		filepath.Join(outputRoot, "controllers", "mysql", "report_controller.go"),
		filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "report", "report_runtimehooks_generated.go"),
		filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "report", "report_serviceclient.go"),
		filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "report", "report_servicemanager.go"),
		filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_report.yaml"),
		filepath.Join(outputRoot, "api", "identity", "v1beta1", "widget_types.go"),
		filepath.Join(outputRoot, "controllers", "identity", "widget_controller.go"),
		filepath.Join(outputRoot, "pkg", "servicemanager", "identity", "widget", "widget_runtimehooks_generated.go"),
		filepath.Join(outputRoot, "pkg", "servicemanager", "identity", "widget", "widget_serviceclient.go"),
		filepath.Join(outputRoot, "pkg", "servicemanager", "identity", "widget", "widget_servicemanager.go"),
		filepath.Join(outputRoot, "internal", "registrations", "identity_generated.go"),
		filepath.Join(outputRoot, "packages", "identity", "metadata.env"),
		filepath.Join(outputRoot, "packages", "identity", "install", "kustomization.yaml"),
		filepath.Join(outputRoot, "packages", "identity", "install", "generated", "crd", "bases", "identity.oracle.com_widget.yaml"),
		filepath.Join(outputRoot, "cmd", "manager", "identity", "main.go"),
		filepath.Join(outputRoot, "config", "manager", "identity", "kustomization.yaml"),
		filepath.Join(outputRoot, "config", "manager", "identity", "manager.yaml"),
		filepath.Join(outputRoot, "config", "manager", "identity", "controller_manager_config.yaml"),
		filepath.Join(outputRoot, "config", "samples", "identity_v1beta1_widget.yaml"),
		staleIdentityDeepCopyFile,
		staleMySQLOldVersionSample,
		staleIdentityOldVersionSample,
	})

	assertPathsExist(t, []string{
		filepath.Join(outputRoot, "api", "mysql", "v1beta1", "widget_types.go"),
		filepath.Join(outputRoot, "controllers", "mysql", "widget_controller.go"),
		filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "widget", "widget_runtimehooks_generated.go"),
		filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "widget", "widget_serviceclient.go"),
		filepath.Join(outputRoot, "packages", "mysql", "metadata.env"),
		filepath.Join(outputRoot, "internal", "registrations", "mysql_generated.go"),
		filepath.Join(outputRoot, "cmd", "manager", "mysql", "main.go"),
		filepath.Join(outputRoot, "config", "manager", "mysql", "kustomization.yaml"),
		filepath.Join(outputRoot, "config", "manager", "mysql", "manager.yaml"),
		filepath.Join(outputRoot, "config", "manager", "mysql", "controller_manager_config.yaml"),
		filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_widget.yaml"),
		preservedDeepCopyFile,
		preservedPackageFile,
	})
	assertFileDoesNotContain(t, filepath.Join(outputRoot, "internal", "registrations", "mysql_generated.go"), []string{"ReportReconciler"})
	assertFileDoesNotContain(t, filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"), []string{
		"mysql_v1alpha1_widget.yaml",
		"identity_v1alpha1_widget.yaml",
	})
	assertSelectionCleanupFileContent(t, preservedDeepCopyFile, "// preserved deepcopy output\n")
	assertSelectionCleanupFileContent(t, preservedPackageFile, "preserved manifest output\n")
}

func TestGenerateFullSyncCleansStaleGeneratorOwnedOutputsForServicesRemovedFromConfig(t *testing.T) {
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

	staleIdentityDeepCopyFile := filepath.Join(outputRoot, "api", "identity", "v1beta1", "zz_generated.deepcopy.go")
	staleIdentityPackageGeneratedFile := filepath.Join(outputRoot, "packages", "identity", "install", "generated", "crd", "bases", "identity.oracle.com_widget.yaml")
	writeSelectionCleanupFile(t, staleIdentityDeepCopyFile, "// stale identity deepcopy output\n")
	writeSelectionCleanupFile(t, staleIdentityPackageGeneratedFile, "stale identity package manifest\n")

	cfg.Services = cfg.Services[:1]

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

	for _, relativePath := range []string{
		"api/identity/v1beta1/widget_types.go",
		"controllers/identity/widget_controller.go",
		"pkg/servicemanager/identity/widget/widget_runtimehooks_generated.go",
		"pkg/servicemanager/identity/widget/widget_serviceclient.go",
		"pkg/servicemanager/identity/widget/widget_servicemanager.go",
		"internal/registrations/identity_generated.go",
		"packages/identity/metadata.env",
		"packages/identity/install/kustomization.yaml",
		"packages/identity/install/generated/crd/bases/identity.oracle.com_widget.yaml",
		"cmd/manager/identity/main.go",
		"config/manager/identity/kustomization.yaml",
		"config/manager/identity/manager.yaml",
		"config/manager/identity/controller_manager_config.yaml",
		"config/samples/identity_v1beta1_widget.yaml",
	} {
		assertPathNotExists(t, filepath.Join(outputRoot, relativePath))
	}
	assertPathNotExists(t, staleIdentityDeepCopyFile)

	assertPathExists(t, filepath.Join(outputRoot, "api", "mysql", "v1beta1", "widget_types.go"))
	assertPathExists(t, filepath.Join(outputRoot, "controllers", "mysql", "widget_controller.go"))
	assertPathExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "widget", "widget_runtimehooks_generated.go"))
	assertPathExists(t, filepath.Join(outputRoot, "pkg", "servicemanager", "mysql", "widget", "widget_serviceclient.go"))
	assertPathExists(t, filepath.Join(outputRoot, "internal", "registrations", "mysql_generated.go"))
	assertPathExists(t, filepath.Join(outputRoot, "packages", "mysql", "metadata.env"))
	assertPathExists(t, filepath.Join(outputRoot, "cmd", "manager", "mysql", "main.go"))
	assertPathExists(t, filepath.Join(outputRoot, "config", "manager", "mysql", "manager.yaml"))
	assertPathExists(t, filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_widget.yaml"))
	assertFileDoesNotContain(t, filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"), []string{
		"identity_v1beta1_widget.yaml",
	})
}

func TestGenerateFullSyncPrunesSampleKustomizationWhenActiveSurfaceIsEmpty(t *testing.T) {
	t.Parallel()

	cfg := selectionCleanupTestConfig()
	cfg.Services[0].Selection = selectionExplicit(false, "Widget")

	outputRoot := t.TempDir()
	pipeline := newSelectionCleanupTestGenerator(t)

	samplesDir := filepath.Join(outputRoot, "config", "samples")
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", samplesDir, err)
	}
	if err := os.WriteFile(filepath.Join(samplesDir, "existing.yaml"), []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: existing\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing.yaml) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(samplesDir, "kustomization.yaml"), []byte("resources:\n- existing.yaml\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(kustomization.yaml) error = %v", err)
	}

	seedServices, err := cfg.SelectServices("mysql", false)
	if err != nil {
		t.Fatalf("SelectServices(--service mysql) error = %v", err)
	}
	if _, err := pipeline.Generate(context.Background(), cfg, seedServices, Options{
		OutputRoot: outputRoot,
		Overwrite:  true,
	}); err != nil {
		t.Fatalf("seed Generate() error = %v", err)
	}

	assertPathExists(t, filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_widget.yaml"))
	assertFileContains(t, filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"), []string{
		"- existing.yaml",
		"- mysql_v1beta1_widget.yaml",
	})

	activeServices, err := cfg.SelectServices("", true)
	if err != nil {
		t.Fatalf("SelectServices(--all) error = %v", err)
	}
	if len(activeServices) != 0 {
		t.Fatalf("SelectServices(--all) returned %d services, want 0", len(activeServices))
	}
	if _, err := pipeline.Generate(context.Background(), cfg, activeServices, Options{
		OutputRoot: outputRoot,
		Overwrite:  true,
		FullSync:   true,
	}); err != nil {
		t.Fatalf("Generate() full-sync error = %v", err)
	}

	assertPathExists(t, filepath.Join(outputRoot, "config", "samples", "existing.yaml"))
	assertPathNotExists(t, filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_widget.yaml"))
	assertFileContains(t, filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"), []string{
		"- existing.yaml",
	})
	assertFileDoesNotContain(t, filepath.Join(outputRoot, "config", "samples", "kustomization.yaml"), []string{
		"mysql_v1beta1_widget.yaml",
	})
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

func seedSelectionCleanupOutputs(t *testing.T, pipeline *Generator, cfg *Config, outputRoot string) {
	t.Helper()

	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{cfg.Services[0], cfg.Services[1]}, Options{
		OutputRoot: outputRoot,
		Overwrite:  true,
	}); err != nil {
		t.Fatalf("seed Generate() error = %v", err)
	}
}

func writeSelectionCleanupFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func assertPathsExist(t *testing.T, paths []string) {
	t.Helper()

	for _, path := range paths {
		assertPathExists(t, path)
	}
}

func assertPathsNotExist(t *testing.T, paths []string) {
	t.Helper()

	for _, path := range paths {
		assertPathNotExists(t, path)
	}
}

func assertSelectionCleanupFileContent(t *testing.T, path string, want string) {
	t.Helper()

	if got := readFile(t, path); got != want {
		t.Fatalf("%s content = %q, want %q", path, got, want)
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
