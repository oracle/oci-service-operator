package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/oracle/oci-service-operator/internal/generator"
)

func TestResolveRunInputsPreserveExistingSpecSurface(t *testing.T) {
	t.Parallel()

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot() error = %v", err)
	}

	inputs, err := resolveRunInputs(options{
		configPath:   defaultConfigPath,
		all:          true,
		preserveSpec: true,
	})
	if err != nil {
		t.Fatalf("resolveRunInputs() error = %v", err)
	}
	if inputs.preserveRoot != repoRoot {
		t.Fatalf("preserveRoot = %q, want %q", inputs.preserveRoot, repoRoot)
	}
}

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
		filepath.Join(root, "pkg", "servicemanager", "functions", "application", "application_servicemanager.go"): generatedServiceManagerSource("application"),
		filepath.Join(root, "pkg", "servicemanager", "functions", "application", "application_serviceclient.go"):  generatedServiceManagerSource("application"),
		filepath.Join(root, "internal", "registrations", "functions_generated.go"):                                generatedRegistrationSource(),
	})

	build, err := collectBuildPlan(root, []string{"functions"}, []string{"functions"}, []string{"functions"})
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

func TestCollectBuildPlanFindsDatabaseGeneratedRuntimePackages(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	mustMkdirAll(t, []string{
		filepath.Join(root, "controllers", "database"),
		filepath.Join(root, "pkg", "servicemanager", "database", "autonomousdatabase"),
		filepath.Join(root, "internal", "registrations"),
	})

	writeTestFiles(t, map[string]string{
		filepath.Join(root, "controllers", "database", "autonomousdatabase_controller.go"):                                     "package database\n",
		filepath.Join(root, "pkg", "servicemanager", "database", "autonomousdatabase", "autonomousdatabase_servicemanager.go"): generatedServiceManagerSource("autonomousdatabase"),
		filepath.Join(root, "pkg", "servicemanager", "database", "autonomousdatabase", "autonomousdatabase_serviceclient.go"):  generatedServiceManagerSource("autonomousdatabase"),
		filepath.Join(root, "internal", "registrations", "database_generated.go"):                                              generatedRegistrationSource(),
	})

	build, err := collectBuildPlan(root, []string{"database"}, []string{"database"}, []string{"database"})
	if err != nil {
		t.Fatalf("collectBuildPlan() error = %v", err)
	}

	if !slices.Equal(build.ServiceManagerPackages, []string{"./pkg/servicemanager/database/autonomousdatabase"}) {
		t.Fatalf("ServiceManagerPackages = %v, want %v", build.ServiceManagerPackages, []string{"./pkg/servicemanager/database/autonomousdatabase"})
	}
}

func TestCollectBuildPlanRejectsMissingRuntimePackages(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	mustMkdirAll(t, []string{filepath.Join(root, "internal", "registrations")})

	if _, err := collectBuildPlan(root, []string{"functions"}, []string{"functions"}, []string{"functions"}); err == nil {
		t.Fatal("collectBuildPlan() error = nil, want missing package error")
	}
}

func TestCollectBuildPlanRejectsLegacyOnlyServiceManagerPackages(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	mustMkdirAll(t, []string{
		filepath.Join(root, "controllers", "database"),
		filepath.Join(root, "pkg", "servicemanager", "database", "autonomousdatabase"),
		filepath.Join(root, "internal", "registrations"),
	})

	writeTestFiles(t, map[string]string{
		filepath.Join(root, "controllers", "database", "autonomousdatabase_controller.go"):                         "package database\n",
		filepath.Join(root, "pkg", "servicemanager", "database", "autonomousdatabase", "legacy_servicemanager.go"): "package autonomousdatabase\n",
		filepath.Join(root, "pkg", "servicemanager", "database", "autonomousdatabase", "manual_helper.go"):         "package autonomousdatabase\n",
		filepath.Join(root, "internal", "registrations", "database_generated.go"):                                  generatedRegistrationSource(),
	})

	_, err := collectBuildPlan(root, []string{"database"}, []string{"database"}, []string{"database"})
	if err == nil {
		t.Fatal("collectBuildPlan() error = nil, want missing generated service-manager packages error")
	}
	if err.Error() != "no generated service-manager packages detected in snapshot" {
		t.Fatalf("collectBuildPlan() error = %v, want %q", err, "no generated service-manager packages detected in snapshot")
	}
}

func TestCollectBuildPlanRejectsMissingSelectedRegistrationOutputs(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	mustMkdirAll(t, []string{
		filepath.Join(root, "controllers", "functions"),
		filepath.Join(root, "pkg", "servicemanager", "functions", "application"),
		filepath.Join(root, "internal", "registrations"),
	})

	writeTestFiles(t, map[string]string{
		filepath.Join(root, "controllers", "functions", "application_controller.go"):                              "package functions\n",
		filepath.Join(root, "pkg", "servicemanager", "functions", "application", "application_servicemanager.go"): generatedServiceManagerSource("application"),
		filepath.Join(root, "pkg", "servicemanager", "functions", "application", "application_serviceclient.go"):  generatedServiceManagerSource("application"),
		filepath.Join(root, "internal", "registrations", "events_generated.go"):                                   generatedRegistrationSource(),
	})

	_, err := collectBuildPlan(root, []string{"functions"}, []string{"functions"}, []string{"functions"})
	if err == nil {
		t.Fatal("collectBuildPlan() error = nil, want missing selected registration output error")
	}
	if err.Error() != "missing generated registration outputs in snapshot: functions_generated.go" {
		t.Fatalf("collectBuildPlan() error = %v, want %q", err, "missing generated registration outputs in snapshot: functions_generated.go")
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
		filepath.Join(repoRoot, "pkg", "servicemanager", "database"),
		filepath.Join(repoRoot, "pkg", "servicemanager", "identity"),
		filepath.Join(repoRoot, "internal", "registrations"),
	})
	writeTestFiles(t, map[string]string{
		filepath.Join(repoRoot, "go.mod"):                                             "module example.com/test\n",
		filepath.Join(repoRoot, "go.sum"):                                             "",
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
		generatedServiceClientPath: generatedServiceManagerSource("autonomousdatabase"),
	})

	snapshotServiceManagerDir := filepath.Join(snapshotRoot, "pkg", "servicemanager", "database", "autonomousdatabase")
	mustMkdirAll(t, []string{snapshotServiceManagerDir})
	snapshotGeneratedServiceClientPath := filepath.Join(snapshotServiceManagerDir, "autonomousdatabase_serviceclient.go")
	writeTestFiles(t, map[string]string{
		snapshotGeneratedServiceClientPath: generatedServiceManagerSource("autonomousdatabase"),
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

	snapshotHelperPath := filepath.Join(snapshotRoot, "api", "database", "v1beta1", "autonomousdatabase_helpers.go")
	snapshotAdapterPath := filepath.Join(snapshotServiceManagerDir, "autonomousdatabase_generated_client_adapter.go")
	snapshotLegacyServiceManagerPath := filepath.Join(snapshotServiceManagerDir, "legacy_servicemanager.go")
	assertSymlinkTarget(t, snapshotHelperPath, helperPath)
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

func generatedServiceManagerSource(packageName string) string {
	return "package " + packageName + "\n\n" + generatedFileMarker + "\n"
}

func generatedRegistrationSource() string {
	return "package registrations\n\n" + generatedFileMarker + "\n"
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
