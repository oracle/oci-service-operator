package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oracle/oci-service-operator/internal/generator"
)

func TestPopulateSnapshotKeepsSelectedOutputsWritable(t *testing.T) {
	t.Helper()

	repoRoot := t.TempDir()
	snapshotRoot := t.TempDir()

	mustMkdirAll(t, []string{
		filepath.Join(repoRoot, "api"),
		filepath.Join(repoRoot, "cmd"),
		filepath.Join(repoRoot, "controllers"),
		filepath.Join(repoRoot, "hack"),
		filepath.Join(repoRoot, "formal"),
		filepath.Join(repoRoot, "pkg", "servicemanager", "database"),
		filepath.Join(repoRoot, "pkg", "servicemanager", "identity"),
		filepath.Join(repoRoot, "internal", "registrations"),
		filepath.Join(repoRoot, "internal", "validator"),
	})
	writeTestFiles(t, map[string]string{
		filepath.Join(repoRoot, "go.mod"):                                             "module example.com/test\n",
		filepath.Join(repoRoot, "go.sum"):                                             "",
		filepath.Join(repoRoot, "validator_allowlist.yaml"):                           "{}\n",
		filepath.Join(repoRoot, "cmd", "main.go"):                                     "package main\n",
		filepath.Join(repoRoot, "internal", "validator", "doc.go"):                    "package validator\n",
		filepath.Join(repoRoot, "internal", "registrations", "database_generated.go"): "package registrations\n",
		filepath.Join(repoRoot, "internal", "registrations", "events_generated.go"):   "package registrations\n",
	})

	if err := populateSnapshot(repoRoot, snapshotRoot, []string{"database"}, []string{"database"}); err != nil {
		t.Fatalf("populateSnapshot() error = %v", err)
	}

	assertSymlink(t, filepath.Join(snapshotRoot, "formal"))
	assertNotExists(t, filepath.Join(snapshotRoot, "internal", "registrations", "database_generated.go"), "selected registration")
	assertSymlink(t, filepath.Join(snapshotRoot, "internal", "registrations", "events_generated.go"))
	assertNotExists(t, filepath.Join(snapshotRoot, "pkg", "servicemanager", "database"), "selected service-manager root")
	assertSymlink(t, filepath.Join(snapshotRoot, "pkg", "servicemanager", "identity"))
}

func TestPreserveCheckedInCompanionFilesLinksCheckedInCompatibilityCompanions(t *testing.T) {
	t.Helper()

	repoRoot := t.TempDir()
	snapshotRoot := t.TempDir()

	apiSourceDir := filepath.Join(repoRoot, "api", "database", "v1beta1")
	mustMkdirAll(t, []string{apiSourceDir})
	helperPath := filepath.Join(apiSourceDir, "autonomousdatabase_helpers.go")
	typesPath := filepath.Join(apiSourceDir, "autonomousdatabase_types.go")
	writeTestFiles(t, map[string]string{
		helperPath: "package v1beta1\n",
		typesPath:  "package v1beta1\n",
	})

	serviceManagerSourceDir := filepath.Join(repoRoot, "pkg", "servicemanager", "database", "autonomousdatabase")
	mustMkdirAll(t, []string{serviceManagerSourceDir})
	adapterPath := filepath.Join(serviceManagerSourceDir, "autonomousdatabase_generated_client_adapter.go")
	legacyServiceManagerPath := filepath.Join(serviceManagerSourceDir, "legacy_servicemanager.go")
	generatedServiceClientPath := filepath.Join(serviceManagerSourceDir, "autonomousdatabase_serviceclient.go")
	writeTestFiles(t, map[string]string{
		adapterPath:                "package autonomousdatabase\n",
		legacyServiceManagerPath:   "package autonomousdatabase\n",
		generatedServiceClientPath: "package autonomousdatabase\n",
	})

	snapshotServiceManagerDir := filepath.Join(snapshotRoot, "pkg", "servicemanager", "database", "autonomousdatabase")
	mustMkdirAll(t, []string{snapshotServiceManagerDir})
	snapshotGeneratedServiceClientPath := filepath.Join(snapshotServiceManagerDir, "autonomousdatabase_serviceclient.go")
	writeTestFiles(t, map[string]string{
		snapshotGeneratedServiceClientPath: "package autonomousdatabase\n",
	})

	services := []generator.ServiceConfig{
		{
			Group: "database",
			Generation: generator.GenerationConfig{
				Resources: []generator.ResourceGenerationOverride{
					{
						Kind: "AutonomousDatabase",
						ServiceManager: generator.ServiceManagerGenerationOverride{
							PackagePath: "database/autonomousdatabase",
						},
					},
				},
			},
		},
	}
	if err := preserveCheckedInCompanionFiles(repoRoot, snapshotRoot, "v1beta1", services); err != nil {
		t.Fatalf("preserveCheckedInCompanionFiles() error = %v", err)
	}

	snapshotAdapterPath := filepath.Join(snapshotServiceManagerDir, "autonomousdatabase_generated_client_adapter.go")
	snapshotLegacyServiceManagerPath := filepath.Join(snapshotServiceManagerDir, "legacy_servicemanager.go")
	assertSymlink(t, filepath.Join(snapshotRoot, "api", "database", "v1beta1", "autonomousdatabase_helpers.go"))
	assertSymlink(t, snapshotAdapterPath)
	assertSymlink(t, snapshotLegacyServiceManagerPath)
	assertRegularFile(t, snapshotGeneratedServiceClientPath)
	assertNotExists(t, filepath.Join(snapshotRoot, "api", "database", "v1beta1", "autonomousdatabase_types.go"), "generated api type file")
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
