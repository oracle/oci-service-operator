package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/oracle/oci-service-operator/internal/generator"
)

func TestCollectBuildPlanFindsGeneratedRuntimePackages(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	mustMkdirAll(t, []string{
		filepath.Join(root, "controllers", "functions"),
		filepath.Join(root, "pkg", "servicemanager", "functions", "application"),
		filepath.Join(root, "internal", "registrations"),
	})

	writeTestFiles(t, map[string]string{
		filepath.Join(root, "controllers", "functions", "application_controller.go"):                              "package functions\n",
		filepath.Join(root, "pkg", "servicemanager", "functions", "application", "application_servicemanager.go"): "package application\n",
		filepath.Join(root, "pkg", "servicemanager", "functions", "application", "application_serviceclient.go"):  "package application\n",
		filepath.Join(root, "internal", "registrations", "functions_generated.go"):                                "package registrations\n",
	})

	build, err := collectBuildPlan(root, []string{"functions"}, []string{"functions"})
	if err != nil {
		t.Fatalf("collectBuildPlan() error = %v", err)
	}

	if !slices.Equal(build.ControllerPackages, []string{"./controllers/functions"}) {
		t.Fatalf("ControllerPackages = %v, want %v", build.ControllerPackages, []string{"./controllers/functions"})
	}
	if !slices.Equal(build.ServiceManagerPackages, []string{"./pkg/servicemanager/functions/application"}) {
		t.Fatalf("ServiceManagerPackages = %v, want %v", build.ServiceManagerPackages, []string{"./pkg/servicemanager/functions/application"})
	}
	if !slices.Equal(build.RegistrationPackages, []string{"./internal/registrations"}) {
		t.Fatalf("RegistrationPackages = %v, want %v", build.RegistrationPackages, []string{"./internal/registrations"})
	}
}

func TestCollectBuildPlanFindsOverriddenServiceManagerRoots(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	mustMkdirAll(t, []string{
		filepath.Join(root, "controllers", "database"),
		filepath.Join(root, "pkg", "servicemanager", "autonomousdatabases", "adb"),
		filepath.Join(root, "internal", "registrations"),
	})

	writeTestFiles(t, map[string]string{
		filepath.Join(root, "controllers", "database", "autonomousdatabases_controller.go"):                                 "package database\n",
		filepath.Join(root, "pkg", "servicemanager", "autonomousdatabases", "adb", "autonomousdatabases_servicemanager.go"): "package adb\n",
		filepath.Join(root, "pkg", "servicemanager", "autonomousdatabases", "adb", "autonomousdatabases_serviceclient.go"):  "package adb\n",
		filepath.Join(root, "internal", "registrations", "database_generated.go"):                                           "package registrations\n",
	})

	build, err := collectBuildPlan(root, []string{"database"}, []string{"autonomousdatabases"})
	if err != nil {
		t.Fatalf("collectBuildPlan() error = %v", err)
	}

	if !slices.Equal(build.ServiceManagerPackages, []string{"./pkg/servicemanager/autonomousdatabases/adb"}) {
		t.Fatalf("ServiceManagerPackages = %v, want %v", build.ServiceManagerPackages, []string{"./pkg/servicemanager/autonomousdatabases/adb"})
	}
}

func TestCollectBuildPlanRejectsMissingRuntimePackages(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	mustMkdirAll(t, []string{filepath.Join(root, "internal", "registrations")})

	if _, err := collectBuildPlan(root, []string{"functions"}, []string{"functions"}); err == nil {
		t.Fatal("collectBuildPlan() error = nil, want missing package error")
	}
}

func TestPopulateSnapshotCarriesFormalRootAndLeavesSelectedFilesWritable(t *testing.T) {
	t.Helper()

	repoRoot := t.TempDir()
	snapshotRoot := t.TempDir()

	mustMkdirAll(t, []string{
		filepath.Join(repoRoot, "api"),
		filepath.Join(repoRoot, "controllers"),
		filepath.Join(repoRoot, "hack"),
		filepath.Join(repoRoot, "formal"),
		filepath.Join(repoRoot, "pkg", "servicemanager", "autonomousdatabases"),
		filepath.Join(repoRoot, "pkg", "servicemanager", "identity"),
		filepath.Join(repoRoot, "internal", "registrations"),
	})
	writeTestFiles(t, map[string]string{
		filepath.Join(repoRoot, "go.mod"):                                             "module example.com/test\n",
		filepath.Join(repoRoot, "go.sum"):                                             "",
		filepath.Join(repoRoot, "internal", "registrations", "database_generated.go"): "package registrations\n",
		filepath.Join(repoRoot, "internal", "registrations", "events_generated.go"):   "package registrations\n",
	})

	if err := populateSnapshot(repoRoot, snapshotRoot, []string{"database"}, []string{"autonomousdatabases"}); err != nil {
		t.Fatalf("populateSnapshot() error = %v", err)
	}

	assertSymlink(t, filepath.Join(snapshotRoot, "formal"))
	assertNotExists(t, filepath.Join(snapshotRoot, "internal", "registrations", "database_generated.go"), "selected registration")
	assertSymlink(t, filepath.Join(snapshotRoot, "internal", "registrations", "events_generated.go"))
	assertNotExists(t, filepath.Join(snapshotRoot, "pkg", "servicemanager", "autonomousdatabases"), "selected service-manager root")
	assertSymlink(t, filepath.Join(snapshotRoot, "pkg", "servicemanager", "identity"))
}

func TestPreserveCheckedInCompanionFilesLinksCheckedInCompatibilityCompanions(t *testing.T) {
	t.Helper()

	repoRoot := t.TempDir()
	snapshotRoot := t.TempDir()

	apiSourceDir := filepath.Join(repoRoot, "api", "database", "v1beta1")
	mustMkdirAll(t, []string{apiSourceDir})
	webhookPath := filepath.Join(apiSourceDir, "autonomousdatabases_webhook.go")
	typesPath := filepath.Join(apiSourceDir, "autonomousdatabases_types.go")
	writeTestFiles(t, map[string]string{
		webhookPath: "package v1beta1\n",
		typesPath:   "package v1beta1\n",
	})

	serviceManagerSourceDir := filepath.Join(repoRoot, "pkg", "servicemanager", "autonomousdatabases", "adb")
	mustMkdirAll(t, []string{serviceManagerSourceDir})
	legacyServiceClientPath := filepath.Join(serviceManagerSourceDir, "adb_serviceclient.go")
	legacyServiceManagerPath := filepath.Join(serviceManagerSourceDir, "adb_servicemanager.go")
	adapterPath := filepath.Join(serviceManagerSourceDir, "autonomousdatabases_generated_client_adapter.go")
	generatedServiceClientPath := filepath.Join(serviceManagerSourceDir, "autonomousdatabases_serviceclient.go")
	writeTestFiles(t, map[string]string{
		legacyServiceClientPath:    "package adb\n",
		legacyServiceManagerPath:   "package adb\n",
		adapterPath:                "package adb\n",
		generatedServiceClientPath: "package adb\n",
	})

	snapshotServiceManagerDir := filepath.Join(snapshotRoot, "pkg", "servicemanager", "autonomousdatabases", "adb")
	mustMkdirAll(t, []string{snapshotServiceManagerDir})
	snapshotGeneratedServiceClientPath := filepath.Join(snapshotServiceManagerDir, "autonomousdatabases_serviceclient.go")
	writeTestFiles(t, map[string]string{
		snapshotGeneratedServiceClientPath: "package adb\n",
	})

	services := []generator.ServiceConfig{
		{
			Group: "database",
			Generation: generator.GenerationConfig{
				Resources: []generator.ResourceGenerationOverride{
					{
						Kind: "AutonomousDatabases",
						ServiceManager: generator.ServiceManagerGenerationOverride{
							PackagePath: "autonomousdatabases/adb",
						},
					},
				},
			},
		},
	}
	if err := preserveCheckedInCompanionFiles(repoRoot, snapshotRoot, "v1beta1", services); err != nil {
		t.Fatalf("preserveCheckedInCompanionFiles() error = %v", err)
	}

	snapshotAdapterPath := filepath.Join(snapshotServiceManagerDir, "autonomousdatabases_generated_client_adapter.go")
	snapshotLegacyServiceClientPath := filepath.Join(snapshotServiceManagerDir, "adb_serviceclient.go")
	snapshotLegacyServiceManagerPath := filepath.Join(snapshotServiceManagerDir, "adb_servicemanager.go")
	snapshotWebhookPath := filepath.Join(snapshotRoot, "api", "database", "v1beta1", "autonomousdatabases_webhook.go")
	assertSymlinkTarget(t, snapshotWebhookPath, webhookPath)
	assertSymlink(t, snapshotAdapterPath)
	assertSymlink(t, snapshotLegacyServiceClientPath)
	assertSymlink(t, snapshotLegacyServiceManagerPath)
	assertRegularFile(t, snapshotGeneratedServiceClientPath)
	assertNotExists(t, filepath.Join(snapshotRoot, "api", "database", "v1beta1", "autonomousdatabases_types.go"), "generated api type file")
}

func mustMkdirAll(t *testing.T, dirs []string) {
	t.Helper()
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", dir, err)
		}
	}
}

func writeTestFiles(t *testing.T, files map[string]string) {
	t.Helper()
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}
}

func assertSymlink(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%q mode = %v, want symlink", path, info.Mode())
	}
}

func assertSymlinkTarget(t *testing.T, path string, wantTarget string) {
	t.Helper()
	assertSymlink(t, path)
	target, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", path, err)
	}
	if target != wantTarget {
		t.Fatalf("Readlink(%q) = %q, want %q", path, target, wantTarget)
	}
}

func assertRegularFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("%q mode = %v, want regular file", path, info.Mode())
	}
}

func assertNotExists(t *testing.T, path string, label string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%s) error = %v, want not exist", label, err)
	}
}
