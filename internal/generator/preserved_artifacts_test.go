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

func TestGeneratePreservesCheckedInCompatibilityLockedArtifactsFromSeparateRoot(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileControllerBacked)

	preserveRoot := t.TempDir()
	resourceDir := filepath.Join(preserveRoot, "api", "mysql", "v1beta1")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", resourceDir, err)
	}
	resourcePath := filepath.Join(resourceDir, "mysqldbsystem_types.go")
	if err := os.WriteFile(resourcePath, []byte(existingLockedMySQLTypes), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", resourcePath, err)
	}

	installDir := filepath.Join(preserveRoot, "packages", "mysql", "install")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", installDir, err)
	}
	installPath := filepath.Join(installDir, "kustomization.yaml")
	if err := os.WriteFile(installPath, []byte(existingLockedMySQLInstallKustomization), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", installPath, err)
	}

	samplesDir := filepath.Join(preserveRoot, "config", "samples")
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", samplesDir, err)
	}
	samplePath := filepath.Join(samplesDir, "mysql_v1beta1_mysqldbsystem.yaml")
	if err := os.WriteFile(samplePath, []byte(existingLockedMySQLSample), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", samplePath, err)
	}

	outputRoot := t.TempDir()
	pipeline := newTestGenerator(t)
	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot:                      outputRoot,
		Overwrite:                       true,
		PreserveExistingSpecSurfaceRoot: preserveRoot,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	renderedResourcePath := filepath.Join(outputRoot, "api", "mysql", "v1beta1", "mysqldbsystem_types.go")
	content := readFile(t, renderedResourcePath)
	if content == readFile(t, resourcePath) {
		t.Fatalf("Generate() copied %s verbatim instead of regenerating status/read-model output", renderedResourcePath)
	}
	assertContains(t, content, []string{
		"Port int `json:\"port,omitempty\"`",
		"OsokStatus",
		"shared.OSOKStatus",
		"`json:\"status\"`",
		"LastSuccessfulSync",
		"`json:\"lastSuccessfulSync,omitempty\"`",
		"LifecycleState",
		"`json:\"lifecycleState,omitempty\"`",
	})
	assertNotContains(t, content, []string{
		"Preserved custom checked-in MySqlDbSystem marker.",
	})
	assertExactFileMatch(
		t,
		installPath,
		filepath.Join(outputRoot, "packages", "mysql", "install", "kustomization.yaml"),
	)
	assertExactFileMatch(
		t,
		samplePath,
		filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_mysqldbsystem.yaml"),
	)
}

const existingLockedMySQLTypes = `package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// MySqlDbSystemSpec defines the desired state of MySqlDbSystem.
type MySqlDbSystemSpec struct {
	Port int ` + "`json:\"port,omitempty\"`" + `
}

// MySqlDbSystemStatus defines the observed state of MySqlDbSystem.
type MySqlDbSystemStatus struct {
	LastSuccessfulSync string ` + "`json:\"lastSuccessfulSync,omitempty\"`" + `
}

// +kubebuilder:object:root=true
// Preserved custom checked-in MySqlDbSystem marker.
type MySqlDbSystem struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `

	Spec   MySqlDbSystemSpec   ` + "`json:\"spec,omitempty\"`" + `
	Status MySqlDbSystemStatus ` + "`json:\"status,omitempty\"`" + `
}

// +kubebuilder:object:root=true
type MySqlDbSystemList struct {
	metav1.TypeMeta ` + "`json:\",inline\"`" + `
	metav1.ListMeta ` + "`json:\"metadata,omitempty\"`" + `
	Items           []MySqlDbSystem ` + "`json:\"items\"`" + `
}
`

const existingLockedMySQLInstallKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- generated/crd
- preserved/editor-role.yaml
`

const existingLockedMySQLSample = `apiVersion: mysql.oracle.com/v1beta1
kind: MySqlDbSystem
metadata:
  name: preserved-mysqldbsystem
spec:
  port: 3307
`
