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
