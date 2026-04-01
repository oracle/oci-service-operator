package generatorcmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteRejectsInvalidFormalInputsBeforeGeneration(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "internal", "generator", "config", "services.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(configPath), err)
	}

	content := `
schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
generatorEntrypoint: ./cmd/generator
packageProfiles:
  controller-backed:
    description: runtime-integrated groups
services:
  - service: identity
    sdkPackage: example.com/identity
    group: identity
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: all
    formalSpec: user
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", configPath, err)
	}

	err := execute(context.Background(), options{
		configPath: configPath,
		all:        true,
		outputRoot: t.TempDir(),
	}, io.Discard)
	if err == nil {
		t.Fatal("execute() error = nil, want missing formal root failure")
	}
	if !strings.Contains(err.Error(), filepath.ToSlash(filepath.Join(root, "formal"))) {
		t.Fatalf("execute() error = %v, want formal root path", err)
	}
}

func TestExecuteAllOverwriteRemovesStaleOldVersionSamples(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "internal", "generator", "config", "services.yaml")
	content := `
schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
packageProfiles:
  controller-backed:
    description: runtime-integrated groups
services:
  - service: mysql
    sdkPackage: github.com/oracle/oci-service-operator/internal/generator/testdata/sdk/sample
    group: mysql
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: explicit
      includeKinds:
        - Widget
    generation:
      controller:
        strategy: generated
      serviceManager:
        strategy: generated
      registration:
        strategy: generated
      webhooks:
        strategy: none
  - service: identity
    sdkPackage: github.com/oracle/oci-service-operator/internal/generator/testdata/sdk/sample
    group: identity
    packageProfile: controller-backed
    selection:
      enabled: false
      mode: all
    generation:
      controller:
        strategy: generated
      serviceManager:
        strategy: generated
      registration:
        strategy: generated
      webhooks:
        strategy: none
`
	writeGeneratorCmdTestFile(t, configPath, content)

	outputRoot := t.TempDir()
	samplesDir := filepath.Join(outputRoot, "config", "samples")
	staleMySQLOldVersionSample := filepath.Join(samplesDir, "mysql_v1alpha1_widget.yaml")
	staleIdentityOldVersionSample := filepath.Join(samplesDir, "identity_v1alpha1_widget.yaml")
	kustomizationPath := filepath.Join(samplesDir, "kustomization.yaml")
	writeGeneratorCmdTestFile(t, staleMySQLOldVersionSample, "apiVersion: mysql.oracle.com/v1alpha1\nkind: Widget\n")
	writeGeneratorCmdTestFile(t, staleIdentityOldVersionSample, "apiVersion: identity.oracle.com/v1alpha1\nkind: Widget\n")
	writeGeneratorCmdTestFile(t, kustomizationPath, "resources:\n- mysql_v1alpha1_widget.yaml\n- identity_v1alpha1_widget.yaml\n")

	executeAllOverwrite(t, configPath, outputRoot)

	assertGeneratorCmdPathsMissing(t, []string{
		staleMySQLOldVersionSample,
		staleIdentityOldVersionSample,
		filepath.Join(samplesDir, "identity_v1beta1_widget.yaml"),
	})
	assertGeneratorCmdPathsExist(t, []string{
		filepath.Join(samplesDir, "mysql_v1beta1_widget.yaml"),
	})

	kustomization := readGeneratorCmdTestFile(t, kustomizationPath)
	if !strings.Contains(kustomization, "- mysql_v1beta1_widget.yaml") {
		t.Fatalf("kustomization %q missing generated mysql sample:\n%s", kustomizationPath, kustomization)
	}
	for _, staleEntry := range []string{
		"- mysql_v1alpha1_widget.yaml",
		"- identity_v1alpha1_widget.yaml",
		"- identity_v1beta1_widget.yaml",
	} {
		if strings.Contains(kustomization, staleEntry) {
			t.Fatalf("kustomization %q retained stale sample %q:\n%s", kustomizationPath, staleEntry, kustomization)
		}
	}
}

func TestExecuteAllOverwriteRemovesStaleOutputsForServicesRemovedFromConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "internal", "generator", "config", "services.yaml")
	outputRoot := t.TempDir()
	for _, content := range []string{testGeneratorConfigWithIdentity, testGeneratorConfigWithoutIdentity} {
		writeGeneratorCmdTestFile(t, configPath, content)
		executeAllOverwrite(t, configPath, outputRoot)
	}

	assertGeneratorCmdRelativePathsMissing(t, outputRoot, []string{
		"api/identity/v1beta1/widget_types.go",
		"controllers/identity/widget_controller.go",
		"pkg/servicemanager/identity/widget/widget_serviceclient.go",
		"pkg/servicemanager/identity/widget/widget_servicemanager.go",
		"internal/registrations/identity_generated.go",
		"packages/identity/metadata.env",
		"packages/identity/install/kustomization.yaml",
		"config/samples/identity_v1beta1_widget.yaml",
	})
	assertGeneratorCmdRelativePathsExist(t, outputRoot, []string{
		"api/mysql/v1beta1/widget_types.go",
		"config/samples/mysql_v1beta1_widget.yaml",
	})

	kustomizationPath := filepath.Join(outputRoot, "config", "samples", "kustomization.yaml")
	kustomization := readGeneratorCmdTestFile(t, kustomizationPath)
	if strings.Contains(kustomization, "identity_v1beta1_widget.yaml") {
		t.Fatalf("kustomization %q retained removed-service sample:\n%s", kustomizationPath, kustomization)
	}
}

func executeAllOverwrite(t *testing.T, configPath string, outputRoot string) {
	t.Helper()

	if err := execute(context.Background(), options{
		configPath: configPath,
		all:        true,
		outputRoot: outputRoot,
		overwrite:  true,
	}, io.Discard); err != nil {
		t.Fatalf("execute() error = %v", err)
	}
}

func writeGeneratorCmdTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func readGeneratorCmdTestFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}

func assertGeneratorCmdPathsExist(t *testing.T, paths []string) {
	t.Helper()

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Stat(%q) error = %v", path, err)
		}
	}
}

func assertGeneratorCmdPathsMissing(t *testing.T, paths []string) {
	t.Helper()

	for _, path := range paths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("Stat(%q) error = %v, want not exist", path, err)
		}
	}
}

func assertGeneratorCmdRelativePathsExist(t *testing.T, root string, relativePaths []string) {
	t.Helper()

	paths := make([]string, 0, len(relativePaths))
	for _, relativePath := range relativePaths {
		paths = append(paths, filepath.Join(root, relativePath))
	}
	assertGeneratorCmdPathsExist(t, paths)
}

func assertGeneratorCmdRelativePathsMissing(t *testing.T, root string, relativePaths []string) {
	t.Helper()

	paths := make([]string, 0, len(relativePaths))
	for _, relativePath := range relativePaths {
		paths = append(paths, filepath.Join(root, relativePath))
	}
	assertGeneratorCmdPathsMissing(t, paths)
}

const testGeneratorConfigWithIdentity = `
schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
packageProfiles:
  controller-backed:
    description: runtime-integrated groups
services:
  - service: mysql
    sdkPackage: github.com/oracle/oci-service-operator/internal/generator/testdata/sdk/sample
    group: mysql
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: explicit
      includeKinds:
        - Widget
    generation:
      controller:
        strategy: generated
      serviceManager:
        strategy: generated
      registration:
        strategy: generated
      webhooks:
        strategy: none
  - service: identity
    sdkPackage: github.com/oracle/oci-service-operator/internal/generator/testdata/sdk/sample
    group: identity
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: explicit
      includeKinds:
        - Widget
    generation:
      controller:
        strategy: generated
      serviceManager:
        strategy: generated
      registration:
        strategy: generated
      webhooks:
        strategy: none
`

const testGeneratorConfigWithoutIdentity = `
schemaVersion: v1alpha1
domain: oracle.com
defaultVersion: v1beta1
packageProfiles:
  controller-backed:
    description: runtime-integrated groups
services:
  - service: mysql
    sdkPackage: github.com/oracle/oci-service-operator/internal/generator/testdata/sdk/sample
    group: mysql
    packageProfile: controller-backed
    selection:
      enabled: true
      mode: explicit
      includeKinds:
        - Widget
    generation:
      controller:
        strategy: generated
      serviceManager:
        strategy: generated
      registration:
        strategy: generated
      webhooks:
        strategy: none
`
