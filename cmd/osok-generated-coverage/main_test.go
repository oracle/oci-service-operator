package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/oracle/oci-service-operator/internal/generator"
	"github.com/oracle/oci-service-operator/internal/validator/metrics"
)

func TestNormalizeCoverageOptionsDefaultsBlankSelectionToAll(t *testing.T) {
	t.Helper()

	got, err := normalizeCoverageOptions(options{})
	if err != nil {
		t.Fatalf("normalizeCoverageOptions() error = %v", err)
	}
	if got.service != "" {
		t.Fatalf("normalizeCoverageOptions() service = %q, want empty", got.service)
	}
	if !got.all {
		t.Fatal("normalizeCoverageOptions() all = false, want true")
	}
}

func TestNormalizeCoverageOptionsPreservesExplicitServiceTarget(t *testing.T) {
	t.Helper()

	got, err := normalizeCoverageOptions(options{service: " mysql "})
	if err != nil {
		t.Fatalf("normalizeCoverageOptions() error = %v", err)
	}
	if got.service != "mysql" {
		t.Fatalf("normalizeCoverageOptions() service = %q, want %q", got.service, "mysql")
	}
	if got.all {
		t.Fatal("normalizeCoverageOptions() all = true, want false")
	}
}

func TestNormalizeCoverageOptionsRejectsConflictingSelection(t *testing.T) {
	t.Helper()

	if _, err := normalizeCoverageOptions(options{service: "mysql", all: true}); err == nil {
		t.Fatal("normalizeCoverageOptions() error = nil, want conflict failure")
	}
}

func TestLoadCoverageSelectedServicesDefaultsBlankRunToDefaultActiveSurface(t *testing.T) {
	t.Helper()

	configPath := writeCoverageSelectionConfig(t)
	_, services, err := loadCoverageSelectedServices(configPath, "", false)
	if err != nil {
		t.Fatalf("loadCoverageSelectedServices() error = %v", err)
	}
	if got := coverageServiceNames(services); !slices.Equal(got, []string{"database", "mysql"}) {
		t.Fatalf("loadCoverageSelectedServices() services = %v, want %v", got, []string{"database", "mysql"})
	}
	if got := services[0].SelectedKinds(); !slices.Equal(got, []string{"AutonomousDatabase"}) {
		t.Fatalf("database SelectedKinds() = %v, want %v", got, []string{"AutonomousDatabase"})
	}
	if got := services[1].SelectedKinds(); got != nil {
		t.Fatalf("mysql SelectedKinds() = %v, want nil", got)
	}
}

func TestLoadCoverageSelectedServicesAllowsExplicitDisabledService(t *testing.T) {
	t.Helper()

	configPath := writeCoverageSelectionConfig(t)
	_, services, err := loadCoverageSelectedServices(configPath, "identity", false)
	if err != nil {
		t.Fatalf("loadCoverageSelectedServices(identity) error = %v", err)
	}
	if len(services) != 1 || services[0].Service != "identity" {
		t.Fatalf("loadCoverageSelectedServices(identity) = %#v, want identity only", services)
	}
	if services[0].SelectedKinds() != nil {
		t.Fatalf("identity SelectedKinds() = %v, want nil", services[0].SelectedKinds())
	}
}

func TestCheckedInGeneratedCoverageBaselineMatchesDefaultActiveServiceSurface(t *testing.T) {
	t.Parallel()

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot() error = %v", err)
	}

	configPath := filepath.Join(repoRoot, "internal", "generator", "config", "services.yaml")
	baselinePath := filepath.Join(repoRoot, "internal", "generator", "config", "generated_coverage_baseline.json")

	_, services, err := loadCoverageSelectedServices(configPath, "", false)
	if err != nil {
		t.Fatalf("loadCoverageSelectedServices() error = %v", err)
	}
	baseline, err := metrics.LoadBaseline(baselinePath)
	if err != nil {
		t.Fatalf("LoadBaseline(%q) error = %v", baselinePath, err)
	}

	wantNames := serviceNames(services)
	wantSpecs := make(map[string]int, len(services))
	wantAggregateSpecs := 0
	for _, service := range services {
		specCount := len(service.SelectedKinds())
		wantSpecs[service.Service] = specCount
		wantAggregateSpecs += specCount
	}

	gotNames := make([]string, 0, len(baseline.Services))
	for _, service := range baseline.Services {
		gotNames = append(gotNames, service.Service)
		wantSpecCount, ok := wantSpecs[service.Service]
		if !ok {
			t.Fatalf("baseline service %q is not in the current default-active surface", service.Service)
		}
		if service.Specs != wantSpecCount {
			t.Fatalf("baseline service %q specs = %d, want %d selected kinds", service.Service, service.Specs, wantSpecCount)
		}
	}

	if !slices.Equal(gotNames, wantNames) {
		t.Fatalf("baseline services = %v, want default-active services %v", gotNames, wantNames)
	}
	if baseline.Aggregate.Specs != wantAggregateSpecs {
		t.Fatalf("baseline aggregate specs = %d, want %d selected kinds", baseline.Aggregate.Specs, wantAggregateSpecs)
	}
}

func TestPopulateSnapshotKeepsSelectedOutputsWritable(t *testing.T) {
	t.Helper()

	repoRoot := t.TempDir()
	snapshotRoot := t.TempDir()

	for _, dir := range []string{
		filepath.Join(repoRoot, "api"),
		filepath.Join(repoRoot, "cmd"),
		filepath.Join(repoRoot, "controllers"),
		filepath.Join(repoRoot, "hack"),
		filepath.Join(repoRoot, "formal"),
		filepath.Join(repoRoot, "pkg", "servicemanager", "database"),
		filepath.Join(repoRoot, "pkg", "servicemanager", "identity"),
		filepath.Join(repoRoot, "internal", "registrations"),
		filepath.Join(repoRoot, "internal", "validator"),
	} {
		mustCoverageMkdirAll(t, dir)
	}
	writeCoverageTestFiles(t, map[string]string{
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

	assertCoverageSymlink(t, filepath.Join(snapshotRoot, "formal"))
	assertCoverageNotExists(t, filepath.Join(snapshotRoot, "internal", "registrations", "database_generated.go"))
	assertCoverageSymlink(t, filepath.Join(snapshotRoot, "internal", "registrations", "events_generated.go"))
	assertCoverageNotExists(t, filepath.Join(snapshotRoot, "pkg", "servicemanager", "database"))
	assertCoverageSymlink(t, filepath.Join(snapshotRoot, "pkg", "servicemanager", "identity"))
}

func TestPopulateSnapshotKeepsInternalGeneratorArtifactsIsolated(t *testing.T) {
	t.Helper()

	repoRoot := t.TempDir()
	snapshotRoot := t.TempDir()

	sourceArtifact := filepath.Join(repoRoot, "internal", "generator", "generated", "mutability_overlay", "objectstorage", "objectstoragebucket.json")
	for _, dir := range []string{
		filepath.Join(repoRoot, "api"),
		filepath.Join(repoRoot, "cmd"),
		filepath.Join(repoRoot, "controllers"),
		filepath.Join(repoRoot, "hack"),
		filepath.Join(repoRoot, "formal"),
		filepath.Join(repoRoot, "pkg", "servicemanager", "database"),
		filepath.Join(repoRoot, "pkg", "servicemanager", "identity"),
		filepath.Join(repoRoot, "internal", "registrations"),
		filepath.Join(repoRoot, "internal", "validator"),
		filepath.Dir(sourceArtifact),
	} {
		mustCoverageMkdirAll(t, dir)
	}
	writeCoverageTestFiles(t, map[string]string{
		filepath.Join(repoRoot, "go.mod"):                                             "module example.com/test\n",
		filepath.Join(repoRoot, "go.sum"):                                             "",
		filepath.Join(repoRoot, "validator_allowlist.yaml"):                           "{}\n",
		filepath.Join(repoRoot, "cmd", "main.go"):                                     "package main\n",
		filepath.Join(repoRoot, "internal", "validator", "doc.go"):                    "package validator\n",
		filepath.Join(repoRoot, "internal", "registrations", "database_generated.go"): "package registrations\n",
		sourceArtifact: "{\"schemaVersion\":\"v1alpha1\"}\n",
	})

	if err := populateSnapshot(repoRoot, snapshotRoot, []string{"database"}, []string{"database"}); err != nil {
		t.Fatalf("populateSnapshot() error = %v", err)
	}

	snapshotArtifact := filepath.Join(snapshotRoot, "internal", "generator", "generated", "mutability_overlay", "objectstorage", "objectstoragebucket.json")
	assertCoverageRegularFile(t, snapshotArtifact)
	if err := os.Remove(snapshotArtifact); err != nil {
		t.Fatalf("Remove(%q) error = %v", snapshotArtifact, err)
	}
	if _, err := os.Stat(sourceArtifact); err != nil {
		t.Fatalf("source artifact %q was affected by snapshot mutation: %v", sourceArtifact, err)
	}
}

func TestPreserveCheckedInCompanionFilesLinksManualCompanions(t *testing.T) {
	t.Helper()

	repoRoot := t.TempDir()
	snapshotRoot := t.TempDir()

	apiSourceDir := filepath.Join(repoRoot, "api", "database", "v1beta1")
	mustCoverageMkdirAll(t, apiSourceDir)
	apiCompanionPath := filepath.Join(apiSourceDir, "autonomousdatabase_helpers.go")
	mustCoverageWriteFile(t, apiCompanionPath, "package v1beta1\n")
	typesPath := filepath.Join(apiSourceDir, "autonomousdatabase_types.go")
	mustCoverageWriteFile(t, typesPath, "package v1beta1\n")

	serviceManagerSourceDir := filepath.Join(repoRoot, "pkg", "servicemanager", "database", "autonomousdatabase")
	mustCoverageMkdirAll(t, serviceManagerSourceDir)
	adapterPath := filepath.Join(serviceManagerSourceDir, "autonomousdatabase_generated_client_adapter.go")
	mustCoverageWriteFile(t, adapterPath, "package autonomousdatabase\n")
	legacyServiceManagerPath := filepath.Join(serviceManagerSourceDir, "legacy_servicemanager.go")
	mustCoverageWriteFile(t, legacyServiceManagerPath, "package autonomousdatabase\n")
	generatedServiceClientPath := filepath.Join(serviceManagerSourceDir, "autonomousdatabase_serviceclient.go")
	mustCoverageWriteFile(t, generatedServiceClientPath, "package autonomousdatabase\n")

	snapshotServiceManagerDir := filepath.Join(snapshotRoot, "pkg", "servicemanager", "database", "autonomousdatabase")
	mustCoverageMkdirAll(t, snapshotServiceManagerDir)
	snapshotGeneratedServiceClientPath := filepath.Join(snapshotServiceManagerDir, "autonomousdatabase_serviceclient.go")
	mustCoverageWriteFile(t, snapshotGeneratedServiceClientPath, "package autonomousdatabase\n")

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
	assertCoverageSymlink(t, snapshotAPICompanionPath)
	assertCoverageReadlink(t, snapshotAPICompanionPath, apiCompanionPath)
	assertCoverageSymlink(t, filepath.Join(snapshotServiceManagerDir, "autonomousdatabase_generated_client_adapter.go"))
	assertCoverageSymlink(t, filepath.Join(snapshotServiceManagerDir, "legacy_servicemanager.go"))
	assertCoverageRegularFile(t, snapshotGeneratedServiceClientPath)
	assertCoverageNotExists(t, filepath.Join(snapshotRoot, "api", "database", "v1beta1", "autonomousdatabase_types.go"))
}

func TestPreserveCheckedInCompanionFilesSkipsExcludedGeneratedServiceManagerPackages(t *testing.T) {
	t.Helper()

	repoRoot := t.TempDir()
	snapshotRoot := t.TempDir()

	selectedSourceDir := filepath.Join(repoRoot, "pkg", "servicemanager", "database", "autonomousdatabase")
	mustCoverageMkdirAll(t, selectedSourceDir)
	selectedCompanionPath := filepath.Join(selectedSourceDir, "legacy_servicemanager.go")
	mustCoverageWriteFile(t, selectedCompanionPath, "package autonomousdatabase\n")

	excludedSourceDir := filepath.Join(repoRoot, "pkg", "servicemanager", "database", "dbsystem")
	mustCoverageMkdirAll(t, excludedSourceDir)
	mustCoverageWriteFile(t, filepath.Join(excludedSourceDir, "dbsystem_serviceclient.go"), "package dbsystem\n\n"+generatedFileMarker+"\n")
	mustCoverageWriteFile(t, filepath.Join(excludedSourceDir, "dbsystem_servicemanager.go"), "package dbsystem\n\n"+generatedFileMarker+"\n")

	snapshotSelectedDir := filepath.Join(snapshotRoot, "pkg", "servicemanager", "database", "autonomousdatabase")
	mustCoverageMkdirAll(t, snapshotSelectedDir)
	mustCoverageWriteFile(t, filepath.Join(snapshotSelectedDir, "autonomousdatabase_serviceclient.go"), "package autonomousdatabase\n\n"+generatedFileMarker+"\n")

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

	assertCoverageSymlink(t, filepath.Join(snapshotSelectedDir, "legacy_servicemanager.go"))
	assertCoverageNotExists(t, filepath.Join(snapshotRoot, "pkg", "servicemanager", "database", "dbsystem", "dbsystem_serviceclient.go"))
	assertCoverageNotExists(t, filepath.Join(snapshotRoot, "pkg", "servicemanager", "database", "dbsystem", "dbsystem_servicemanager.go"))
}

func writeCoverageTestFiles(t *testing.T, files map[string]string) {
	t.Helper()

	for path, content := range files {
		mustCoverageWriteFile(t, path, content)
	}
}

func mustCoverageMkdirAll(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func mustCoverageWriteFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func assertCoverageSymlink(t *testing.T, path string) {
	t.Helper()

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%q mode = %v, want symlink", path, info.Mode())
	}
}

func assertCoverageReadlink(t *testing.T, path string, want string) {
	t.Helper()

	target, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", path, err)
	}
	if target != want {
		t.Fatalf("Readlink(%q) = %q, want %q", path, target, want)
	}
}

func assertCoverageRegularFile(t *testing.T, path string) {
	t.Helper()

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("%q mode = %v, want regular file", path, info.Mode())
	}
}

func assertCoverageNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want not exist", path, err)
	}
}

func TestRenderOutputOmitsRetiredPreserveExistingSpecSurfaceField(t *testing.T) {
	t.Helper()

	rendered, err := renderOutput(outputReport{
		Config: outputConfig{
			ConfigPath:        "internal/generator/config/services.yaml",
			All:               true,
			GeneratedServices: []string{"mysql"},
			GeneratedGroups:   []string{"mysql"},
			Top:               10,
		},
	})
	if err != nil {
		t.Fatalf("renderOutput() error = %v", err)
	}
	if bytes.Contains(rendered, []byte("preserveExistingSpecSurface")) {
		t.Fatalf("renderOutput() unexpectedly rendered retired preserveExistingSpecSurface field:\n%s", rendered)
	}
}

func writeCoverageSelectionConfig(t *testing.T) string {
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
	mustCoverageWriteFile(t, configPath, config)
	return configPath
}

func coverageServiceNames(services []generator.ServiceConfig) []string {
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, service.Service)
	}
	return names
}
