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
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(configPath), err)
	}

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
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", configPath, err)
	}

	outputRoot := t.TempDir()
	samplesDir := filepath.Join(outputRoot, "config", "samples")
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", samplesDir, err)
	}

	staleMySQLOldVersionSample := filepath.Join(samplesDir, "mysql_v1alpha1_widget.yaml")
	if err := os.WriteFile(staleMySQLOldVersionSample, []byte("apiVersion: mysql.oracle.com/v1alpha1\nkind: Widget\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", staleMySQLOldVersionSample, err)
	}

	staleIdentityOldVersionSample := filepath.Join(samplesDir, "identity_v1alpha1_widget.yaml")
	if err := os.WriteFile(staleIdentityOldVersionSample, []byte("apiVersion: identity.oracle.com/v1alpha1\nkind: Widget\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", staleIdentityOldVersionSample, err)
	}

	kustomizationPath := filepath.Join(samplesDir, "kustomization.yaml")
	kustomizationContent := "resources:\n- mysql_v1alpha1_widget.yaml\n- identity_v1alpha1_widget.yaml\n"
	if err := os.WriteFile(kustomizationPath, []byte(kustomizationContent), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", kustomizationPath, err)
	}

	err := execute(context.Background(), options{
		configPath: configPath,
		all:        true,
		outputRoot: outputRoot,
		overwrite:  true,
	}, io.Discard)
	if err != nil {
		t.Fatalf("execute() error = %v", err)
	}

	if _, err := os.Stat(staleMySQLOldVersionSample); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want not exist", staleMySQLOldVersionSample, err)
	}
	if _, err := os.Stat(staleIdentityOldVersionSample); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want not exist", staleIdentityOldVersionSample, err)
	}
	if _, err := os.Stat(filepath.Join(samplesDir, "mysql_v1beta1_widget.yaml")); err != nil {
		t.Fatalf("Stat(%q) error = %v", filepath.Join(samplesDir, "mysql_v1beta1_widget.yaml"), err)
	}
	if _, err := os.Stat(filepath.Join(samplesDir, "identity_v1beta1_widget.yaml")); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want not exist", filepath.Join(samplesDir, "identity_v1beta1_widget.yaml"), err)
	}

	kustomization, err := os.ReadFile(kustomizationPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", kustomizationPath, err)
	}
	if !strings.Contains(string(kustomization), "- mysql_v1beta1_widget.yaml") {
		t.Fatalf("kustomization %q missing generated mysql sample:\n%s", kustomizationPath, string(kustomization))
	}
	for _, staleEntry := range []string{
		"- mysql_v1alpha1_widget.yaml",
		"- identity_v1alpha1_widget.yaml",
		"- identity_v1beta1_widget.yaml",
	} {
		if strings.Contains(string(kustomization), staleEntry) {
			t.Fatalf("kustomization %q retained stale sample %q:\n%s", kustomizationPath, staleEntry, string(kustomization))
		}
	}
}

func TestExecuteAllOverwriteRemovesStaleOutputsForServicesRemovedFromConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "internal", "generator", "config", "services.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(configPath), err)
	}

	outputRoot := t.TempDir()
	for _, content := range []string{testGeneratorConfigWithIdentity, testGeneratorConfigWithoutIdentity} {
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", configPath, err)
		}
		if err := execute(context.Background(), options{
			configPath: configPath,
			all:        true,
			outputRoot: outputRoot,
			overwrite:  true,
		}, io.Discard); err != nil {
			t.Fatalf("execute() error = %v", err)
		}
	}

	for _, relativePath := range []string{
		"api/identity/v1beta1/widget_types.go",
		"controllers/identity/widget_controller.go",
		"pkg/servicemanager/identity/widget/widget_serviceclient.go",
		"pkg/servicemanager/identity/widget/widget_servicemanager.go",
		"internal/registrations/identity_generated.go",
		"packages/identity/metadata.env",
		"packages/identity/install/kustomization.yaml",
		"config/samples/identity_v1beta1_widget.yaml",
	} {
		if _, err := os.Stat(filepath.Join(outputRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("Stat(%q) error = %v, want not exist", filepath.Join(outputRoot, relativePath), err)
		}
	}

	if _, err := os.Stat(filepath.Join(outputRoot, "api", "mysql", "v1beta1", "widget_types.go")); err != nil {
		t.Fatalf("Stat(%q) error = %v", filepath.Join(outputRoot, "api", "mysql", "v1beta1", "widget_types.go"), err)
	}
	if _, err := os.Stat(filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_widget.yaml")); err != nil {
		t.Fatalf("Stat(%q) error = %v", filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_widget.yaml"), err)
	}

	kustomizationPath := filepath.Join(outputRoot, "config", "samples", "kustomization.yaml")
	kustomization, err := os.ReadFile(kustomizationPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", kustomizationPath, err)
	}
	if strings.Contains(string(kustomization), "identity_v1beta1_widget.yaml") {
		t.Fatalf("kustomization %q retained removed-service sample:\n%s", kustomizationPath, string(kustomization))
	}
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
