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

func TestGeneratePreservesCheckedInPackageArtifactsFromSeparateRoot(t *testing.T) {
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
	resourcePath := filepath.Join(resourceDir, "dbsystem_types.go")
	if err := os.WriteFile(resourcePath, []byte(existingCheckedInMySQLTypes), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", resourcePath, err)
	}

	installDir := filepath.Join(preserveRoot, "packages", "mysql", "install")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", installDir, err)
	}
	installPath := filepath.Join(installDir, "kustomization.yaml")
	if err := os.WriteFile(installPath, []byte(existingCheckedInMySQLInstallKustomization), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", installPath, err)
	}

	samplesDir := filepath.Join(preserveRoot, "config", "samples")
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", samplesDir, err)
	}
	samplePath := filepath.Join(samplesDir, "mysql_v1beta1_dbsystem.yaml")
	if err := os.WriteFile(samplePath, []byte(existingCheckedInMySQLSample), 0o644); err != nil {
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

	renderedResourcePath := filepath.Join(outputRoot, "api", "mysql", "v1beta1", "dbsystem_types.go")
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
		"Preserved custom checked-in DbSystem marker.",
	})
	assertExactFileMatch(
		t,
		installPath,
		filepath.Join(outputRoot, "packages", "mysql", "install", "kustomization.yaml"),
	)
	assertExactFileMatch(
		t,
		samplePath,
		filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_dbsystem.yaml"),
	)
}

const existingCheckedInMySQLTypes = `package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// DbSystemSpec defines the desired state of DbSystem.
type DbSystemSpec struct {
	Port int ` + "`json:\"port,omitempty\"`" + `
}

// DbSystemStatus defines the observed state of DbSystem.
type DbSystemStatus struct {
	LastSuccessfulSync string ` + "`json:\"lastSuccessfulSync,omitempty\"`" + `
}

// +kubebuilder:object:root=true
// Preserved custom checked-in DbSystem marker.
type DbSystem struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `

	Spec   DbSystemSpec   ` + "`json:\"spec,omitempty\"`" + `
	Status DbSystemStatus ` + "`json:\"status,omitempty\"`" + `
}

// +kubebuilder:object:root=true
type DbSystemList struct {
	metav1.TypeMeta ` + "`json:\",inline\"`" + `
	metav1.ListMeta ` + "`json:\"metadata,omitempty\"`" + `
	Items           []DbSystem ` + "`json:\"items\"`" + `
}
`

const existingCheckedInMySQLInstallKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- generated/crd
- preserved/editor-role.yaml
`

const existingCheckedInMySQLSample = `apiVersion: mysql.oracle.com/v1beta1
kind: DbSystem
metadata:
  name: preserved-dbsystem
spec:
  port: 3307
`
