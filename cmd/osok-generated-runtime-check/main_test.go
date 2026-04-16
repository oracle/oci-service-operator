package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/oracle/oci-service-operator/internal/generator"
)

func TestNormalizeRuntimeCheckOptionsDefaultsBlankSelectionToAll(t *testing.T) {
	t.Helper()

	got, err := normalizeRuntimeCheckOptions(options{})
	if err != nil {
		t.Fatalf("normalizeRuntimeCheckOptions() error = %v", err)
	}
	if got.service != "" {
		t.Fatalf("normalizeRuntimeCheckOptions() service = %q, want empty", got.service)
	}
	if !got.all {
		t.Fatal("normalizeRuntimeCheckOptions() all = false, want true")
	}
}

func TestNormalizeRuntimeCheckOptionsPreservesExplicitServiceTarget(t *testing.T) {
	t.Helper()

	got, err := normalizeRuntimeCheckOptions(options{service: " mysql "})
	if err != nil {
		t.Fatalf("normalizeRuntimeCheckOptions() error = %v", err)
	}
	if got.service != "mysql" {
		t.Fatalf("normalizeRuntimeCheckOptions() service = %q, want %q", got.service, "mysql")
	}
	if got.all {
		t.Fatal("normalizeRuntimeCheckOptions() all = true, want false")
	}
}

func TestNormalizeRuntimeCheckOptionsRejectsConflictingSelection(t *testing.T) {
	t.Helper()

	if _, err := normalizeRuntimeCheckOptions(options{service: "mysql", all: true}); err == nil {
		t.Fatal("normalizeRuntimeCheckOptions() error = nil, want conflict failure")
	}
}

func TestLoadRuntimeCheckSelectedServicesDefaultsBlankRunToDefaultActiveSurface(t *testing.T) {
	t.Helper()

	configPath := writeRuntimeCheckSelectionConfig(t)
	_, services, err := loadRuntimeCheckSelectedServices(configPath, "", false)
	if err != nil {
		t.Fatalf("loadRuntimeCheckSelectedServices() error = %v", err)
	}
	if got := runtimeCheckServiceNames(services); !slices.Equal(got, []string{"database", "mysql"}) {
		t.Fatalf("loadRuntimeCheckSelectedServices() services = %v, want %v", got, []string{"database", "mysql"})
	}
	if got := services[0].SelectedKinds(); !slices.Equal(got, []string{"AutonomousDatabase"}) {
		t.Fatalf("database SelectedKinds() = %v, want %v", got, []string{"AutonomousDatabase"})
	}
	if got := services[1].SelectedKinds(); got != nil {
		t.Fatalf("mysql SelectedKinds() = %v, want nil", got)
	}
}

func TestLoadRuntimeCheckSelectedServicesAllowsExplicitDisabledService(t *testing.T) {
	t.Helper()

	configPath := writeRuntimeCheckSelectionConfig(t)
	_, services, err := loadRuntimeCheckSelectedServices(configPath, "identity", false)
	if err != nil {
		t.Fatalf("loadRuntimeCheckSelectedServices(identity) error = %v", err)
	}
	if len(services) != 1 || services[0].Service != "identity" {
		t.Fatalf("loadRuntimeCheckSelectedServices(identity) = %#v, want identity only", services)
	}
	if services[0].SelectedKinds() != nil {
		t.Fatalf("identity SelectedKinds() = %v, want nil", services[0].SelectedKinds())
	}
}

func TestCollectBuildPlanFindsGeneratedRuntimePackages(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	mustRuntimeCheckMkdirAll(t, filepath.Join(root, "controllers", "functions"))
	mustRuntimeCheckMkdirAll(t, filepath.Join(root, "pkg", "servicemanager", "functions", "application"))
	mustRuntimeCheckMkdirAll(t, filepath.Join(root, "internal", "registrations"))

	writeRuntimeCheckFiles(t, map[string]string{
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
	mustRuntimeCheckMkdirAll(t, filepath.Join(root, "controllers", "database"))
	mustRuntimeCheckMkdirAll(t, filepath.Join(root, "pkg", "servicemanager", "database", "autonomousdatabase"))
	mustRuntimeCheckMkdirAll(t, filepath.Join(root, "internal", "registrations"))

	writeRuntimeCheckFiles(t, map[string]string{
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
	mustRuntimeCheckMkdirAll(t, filepath.Join(root, "internal", "registrations"))

	if _, err := collectBuildPlan(root, []string{"functions"}, []string{"functions"}, []string{"functions"}); err == nil {
		t.Fatal("collectBuildPlan() error = nil, want missing package error")
	}
}

func TestCollectBuildPlanRejectsLegacyOnlyServiceManagerPackages(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	mustRuntimeCheckMkdirAll(t, filepath.Join(root, "controllers", "database"))
	mustRuntimeCheckMkdirAll(t, filepath.Join(root, "pkg", "servicemanager", "database", "autonomousdatabase"))
	mustRuntimeCheckMkdirAll(t, filepath.Join(root, "internal", "registrations"))

	writeRuntimeCheckFiles(t, map[string]string{
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
	mustRuntimeCheckMkdirAll(t, filepath.Join(root, "controllers", "functions"))
	mustRuntimeCheckMkdirAll(t, filepath.Join(root, "pkg", "servicemanager", "functions", "application"))
	mustRuntimeCheckMkdirAll(t, filepath.Join(root, "internal", "registrations"))

	writeRuntimeCheckFiles(t, map[string]string{
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

	for _, dir := range []string{
		filepath.Join(repoRoot, "api"),
		filepath.Join(repoRoot, "controllers"),
		filepath.Join(repoRoot, "hack"),
		filepath.Join(repoRoot, "formal"),
		filepath.Join(repoRoot, "pkg", "servicemanager", "database"),
		filepath.Join(repoRoot, "pkg", "servicemanager", "identity"),
		filepath.Join(repoRoot, "internal", "registrations"),
	} {
		mustRuntimeCheckMkdirAll(t, dir)
	}
	writeRuntimeCheckFiles(t, map[string]string{
		filepath.Join(repoRoot, "go.mod"):                                             "module example.com/test\n",
		filepath.Join(repoRoot, "go.sum"):                                             "",
		filepath.Join(repoRoot, "internal", "registrations", "database_generated.go"): "package registrations\n",
		filepath.Join(repoRoot, "internal", "registrations", "events_generated.go"):   "package registrations\n",
	})

	if err := populateSnapshot(repoRoot, snapshotRoot, []string{"database"}, []string{"database"}, []string{"database"}); err != nil {
		t.Fatalf("populateSnapshot() error = %v", err)
	}

	assertRuntimeCheckSymlink(t, filepath.Join(snapshotRoot, "formal"))
	assertRuntimeCheckNotExists(t, filepath.Join(snapshotRoot, "internal", "registrations", "database_generated.go"))
	assertRuntimeCheckNotExists(t, filepath.Join(snapshotRoot, "internal", "registrations", "events_generated.go"))
	assertRuntimeCheckNotExists(t, filepath.Join(snapshotRoot, "pkg", "servicemanager", "database"))
	assertRuntimeCheckSymlink(t, filepath.Join(snapshotRoot, "pkg", "servicemanager", "identity"))
}

func TestPopulateRegistrationsKeepsSharedAndSelectedCompanionsOnly(t *testing.T) {
	t.Helper()

	repoRoot := t.TempDir()
	snapshotRoot := t.TempDir()

	registrationSourceDir := filepath.Join(repoRoot, "internal", "registrations")
	mustRuntimeCheckMkdirAll(t, registrationSourceDir)
	writeRuntimeCheckFiles(t, map[string]string{
		filepath.Join(registrationSourceDir, "registry.go"):                  "package registrations\n",
		filepath.Join(registrationSourceDir, "manual_groups.go"):             "package registrations\n",
		filepath.Join(registrationSourceDir, "registry_test.go"):             "package registrations\n",
		filepath.Join(registrationSourceDir, "core_generated.go"):            generatedRegistrationSource(),
		filepath.Join(registrationSourceDir, "core-network_generated.go"):    generatedRegistrationSource(),
		filepath.Join(registrationSourceDir, "objectstorage_manual.go"):      "package registrations\n",
		filepath.Join(registrationSourceDir, "containerinstances_manual.go"): "package registrations\n",
		filepath.Join(registrationSourceDir, "events_generated.go"):          generatedRegistrationSource(),
	})

	if err := populateRegistrations(repoRoot, snapshotRoot, []string{"core", "objectstorage"}); err != nil {
		t.Fatalf("populateRegistrations() error = %v", err)
	}

	assertRuntimeCheckSymlink(t, filepath.Join(snapshotRoot, "internal", "registrations", "registry.go"))
	assertRuntimeCheckSymlink(t, filepath.Join(snapshotRoot, "internal", "registrations", "manual_groups.go"))
	assertRuntimeCheckSymlink(t, filepath.Join(snapshotRoot, "internal", "registrations", "registry_test.go"))
	assertRuntimeCheckNotExists(t, filepath.Join(snapshotRoot, "internal", "registrations", "core_generated.go"))
	assertRuntimeCheckNotExists(t, filepath.Join(snapshotRoot, "internal", "registrations", "core-network_generated.go"))
	assertRuntimeCheckSymlink(t, filepath.Join(snapshotRoot, "internal", "registrations", "objectstorage_manual.go"))
	assertRuntimeCheckNotExists(t, filepath.Join(snapshotRoot, "internal", "registrations", "containerinstances_manual.go"))
	assertRuntimeCheckNotExists(t, filepath.Join(snapshotRoot, "internal", "registrations", "events_generated.go"))
}

func TestPreserveCheckedInCompanionFilesLinksCheckedInCompatibilityCompanions(t *testing.T) {
	t.Helper()

	repoRoot := t.TempDir()
	snapshotRoot := t.TempDir()

	apiSourceDir := filepath.Join(repoRoot, "api", "database", "v1beta1")
	mustRuntimeCheckMkdirAll(t, apiSourceDir)
	apiCompanionPath := filepath.Join(apiSourceDir, "autonomousdatabase_helpers.go")
	mustRuntimeCheckWriteFile(t, apiCompanionPath, "package v1beta1\n")
	typesPath := filepath.Join(apiSourceDir, "autonomousdatabase_types.go")
	mustRuntimeCheckWriteFile(t, typesPath, "package v1beta1\n")

	serviceManagerSourceDir := filepath.Join(repoRoot, "pkg", "servicemanager", "database", "autonomousdatabase")
	mustRuntimeCheckMkdirAll(t, serviceManagerSourceDir)
	adapterPath := filepath.Join(serviceManagerSourceDir, "autonomousdatabase_generated_client_adapter.go")
	mustRuntimeCheckWriteFile(t, adapterPath, "package autonomousdatabase\n")
	legacyServiceManagerPath := filepath.Join(serviceManagerSourceDir, "legacy_servicemanager.go")
	mustRuntimeCheckWriteFile(t, legacyServiceManagerPath, "package autonomousdatabase\n")
	generatedServiceClientPath := filepath.Join(serviceManagerSourceDir, "autonomousdatabase_serviceclient.go")
	mustRuntimeCheckWriteFile(t, generatedServiceClientPath, generatedServiceManagerSource("autonomousdatabase"))

	snapshotServiceManagerDir := filepath.Join(snapshotRoot, "pkg", "servicemanager", "database", "autonomousdatabase")
	mustRuntimeCheckMkdirAll(t, snapshotServiceManagerDir)
	snapshotGeneratedServiceClientPath := filepath.Join(snapshotServiceManagerDir, "autonomousdatabase_serviceclient.go")
	mustRuntimeCheckWriteFile(t, snapshotGeneratedServiceClientPath, generatedServiceManagerSource("autonomousdatabase"))

	packages := []*generator.PackageModel{
		{
			Service: generator.ServiceConfig{Group: "database"},
			Version: "v1beta1",
			ServiceManagers: []generator.ServiceManagerModel{
				{
					PackagePath:            "database/autonomousdatabase",
					ServiceClientFileName:  "autonomousdatabase_serviceclient.go",
					ServiceManagerFileName: "autonomousdatabase_servicemanager.go",
				},
			},
		},
	}
	if err := preserveCheckedInCompanionFiles(repoRoot, snapshotRoot, packages); err != nil {
		t.Fatalf("preserveCheckedInCompanionFiles() error = %v", err)
	}

	snapshotAPICompanionPath := filepath.Join(snapshotRoot, "api", "database", "v1beta1", "autonomousdatabase_helpers.go")
	assertRuntimeCheckSymlink(t, snapshotAPICompanionPath)
	assertRuntimeCheckReadlink(t, snapshotAPICompanionPath, apiCompanionPath)
	assertRuntimeCheckSymlink(t, filepath.Join(snapshotServiceManagerDir, "autonomousdatabase_generated_client_adapter.go"))
	assertRuntimeCheckSymlink(t, filepath.Join(snapshotServiceManagerDir, "legacy_servicemanager.go"))
	assertRuntimeCheckRegularFile(t, snapshotGeneratedServiceClientPath)
	assertRuntimeCheckNotExists(t, filepath.Join(snapshotRoot, "api", "database", "v1beta1", "autonomousdatabase_types.go"))
}

func TestPreserveCheckedInCompanionFilesSkipsExcludedGeneratedServiceManagerPackages(t *testing.T) {
	t.Helper()

	repoRoot := t.TempDir()
	snapshotRoot := t.TempDir()

	selectedSourceDir := filepath.Join(repoRoot, "pkg", "servicemanager", "database", "autonomousdatabase")
	mustRuntimeCheckMkdirAll(t, selectedSourceDir)
	selectedCompanionPath := filepath.Join(selectedSourceDir, "legacy_servicemanager.go")
	mustRuntimeCheckWriteFile(t, selectedCompanionPath, "package autonomousdatabase\n")

	excludedSourceDir := filepath.Join(repoRoot, "pkg", "servicemanager", "database", "dbsystem")
	mustRuntimeCheckMkdirAll(t, excludedSourceDir)
	mustRuntimeCheckWriteFile(t, filepath.Join(excludedSourceDir, "dbsystem_serviceclient.go"), generatedServiceManagerSource("dbsystem"))
	mustRuntimeCheckWriteFile(t, filepath.Join(excludedSourceDir, "dbsystem_servicemanager.go"), generatedServiceManagerSource("dbsystem"))

	snapshotSelectedDir := filepath.Join(snapshotRoot, "pkg", "servicemanager", "database", "autonomousdatabase")
	mustRuntimeCheckMkdirAll(t, snapshotSelectedDir)
	mustRuntimeCheckWriteFile(t, filepath.Join(snapshotSelectedDir, "autonomousdatabase_serviceclient.go"), generatedServiceManagerSource("autonomousdatabase"))

	packages := []*generator.PackageModel{
		{
			Service: generator.ServiceConfig{Group: "database"},
			ServiceManagers: []generator.ServiceManagerModel{
				{
					PackagePath:            "database/autonomousdatabase",
					ServiceClientFileName:  "autonomousdatabase_serviceclient.go",
					ServiceManagerFileName: "autonomousdatabase_servicemanager.go",
				},
			},
		},
	}
	if err := preserveCheckedInCompanionFiles(repoRoot, snapshotRoot, packages); err != nil {
		t.Fatalf("preserveCheckedInCompanionFiles() error = %v", err)
	}

	assertRuntimeCheckSymlink(t, filepath.Join(snapshotSelectedDir, "legacy_servicemanager.go"))
	assertRuntimeCheckNotExists(t, filepath.Join(snapshotRoot, "pkg", "servicemanager", "database", "dbsystem", "dbsystem_serviceclient.go"))
	assertRuntimeCheckNotExists(t, filepath.Join(snapshotRoot, "pkg", "servicemanager", "database", "dbsystem", "dbsystem_servicemanager.go"))
}

func TestCompilePackageSetIgnoresBrokenLocalTests(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	pkgDir := filepath.Join(root, "pkg", "sample")
	mustRuntimeCheckMkdirAll(t, pkgDir)

	writeRuntimeCheckFiles(t, map[string]string{
		filepath.Join(root, "go.mod"):                 "module example.com/runtimecheck\n\ngo 1.22\n",
		filepath.Join(pkgDir, "sample.go"):            "package sample\n\nconst Value = 1\n",
		filepath.Join(pkgDir, "sample_test.go"):       "package sample\n\nimport _ \"example.com/runtimecheck/does/not/exist\"\n",
		filepath.Join(pkgDir, "sample_extra_test.go"): "package sample_test\n\nimport _ \"example.com/runtimecheck/also/missing\"\n",
	})

	if err := compilePackageSet(root, runtimeCheckCommandEnv(t, root), "sample package", []string{"./pkg/sample"}); err != nil {
		t.Fatalf("compilePackageSet() error = %v, want nil", err)
	}
}

func TestCompilePackageSetRejectsBrokenPackageImports(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	pkgDir := filepath.Join(root, "pkg", "sample")
	mustRuntimeCheckMkdirAll(t, pkgDir)

	writeRuntimeCheckFiles(t, map[string]string{
		filepath.Join(root, "go.mod"):           "module example.com/runtimecheck\n\ngo 1.22\n",
		filepath.Join(pkgDir, "sample.go"):      "package sample\n\nimport _ \"example.com/runtimecheck/does/not/exist\"\n",
		filepath.Join(pkgDir, "sample_test.go"): "package sample\n\nfunc TestPlaceholder(*testing.T) {}\n",
	})

	err := compilePackageSet(root, runtimeCheckCommandEnv(t, root), "sample package", []string{"./pkg/sample"})
	if err == nil {
		t.Fatal("compilePackageSet() error = nil, want compile failure")
	}
	if !strings.Contains(err.Error(), "does/not/exist") {
		t.Fatalf("compilePackageSet() error = %v, want missing package reference", err)
	}
}

func writeRuntimeCheckFiles(t *testing.T, files map[string]string) {
	t.Helper()

	for path, content := range files {
		mustRuntimeCheckWriteFile(t, path, content)
	}
}

func mustRuntimeCheckMkdirAll(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func mustRuntimeCheckWriteFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func generatedServiceManagerSource(packageName string) string {
	return "package " + packageName + "\n\n" + generatedFileMarker + "\n"
}

func generatedRegistrationSource() string {
	return "package registrations\n\n" + generatedFileMarker + "\n"
}

func assertRuntimeCheckSymlink(t *testing.T, path string) {
	t.Helper()

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%q mode = %v, want symlink", path, info.Mode())
	}
}

func assertRuntimeCheckReadlink(t *testing.T, path string, want string) {
	t.Helper()

	target, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", path, err)
	}
	if target != want {
		t.Fatalf("Readlink(%q) = %q, want %q", path, target, want)
	}
}

func assertRuntimeCheckRegularFile(t *testing.T, path string) {
	t.Helper()

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("%q mode = %v, want regular file", path, info.Mode())
	}
}

func assertRuntimeCheckNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want not exist", path, err)
	}
}

func writeRuntimeCheckSelectionConfig(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	configPath := filepath.Join(root, "services.yaml")
	const config = `schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  controller-backed:
    description: runtime-integrated groups
services:
  - service: database
    sdkPackage: example.com/database
    group: database
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: explicit
      includeKinds:
        - AutonomousDatabase
    async:
      strategy: lifecycle
      runtime: generatedruntime
      formalClassification: lifecycle
  - service: mysql
    sdkPackage: example.com/mysql
    group: mysql
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: all
  - service: identity
    sdkPackage: example.com/identity
    group: identity
    packageProfile: controller-backed
    selection:
      enabled: false
      mode: all
`
	mustRuntimeCheckWriteFile(t, configPath, config)
	return configPath
}

func runtimeCheckServiceNames(services []generator.ServiceConfig) []string {
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, service.Service)
	}
	return names
}

func runtimeCheckCommandEnv(t *testing.T, root string) []string {
	t.Helper()

	gocache := filepath.Join(root, ".gocache")
	gomodcache := filepath.Join(root, ".gomodcache")
	gotmpdir := filepath.Join(root, ".gotmp")
	for _, dir := range []string{gocache, gomodcache, gotmpdir} {
		mustRuntimeCheckMkdirAll(t, dir)
	}

	env := append([]string{}, os.Environ()...)
	env = setEnv(env, "GOCACHE", gocache)
	env = setEnv(env, "GOMODCACHE", gomodcache)
	env = setEnv(env, "GOTMPDIR", gotmpdir)
	env = setEnv(env, "GOWORK", "off")
	env = setEnv(env, "GOFLAGS", appendBuildVCSFlag(os.Getenv("GOFLAGS")))
	return env
}
