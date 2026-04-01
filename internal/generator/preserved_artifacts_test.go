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

func TestGenerateOverwritesExistingPackageAndSampleArtifactsInOutputRoot(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	mysqlService := serviceConfigsByName(t, cfg, "mysql")["mysql"]

	outputRoot := t.TempDir()

	installPath := filepath.Join(outputRoot, "packages", "mysql", "install", "kustomization.yaml")
	if err := os.MkdirAll(filepath.Dir(installPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(installPath), err)
	}
	if err := os.WriteFile(installPath, []byte(staleMySQLInstallKustomization), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", installPath, err)
	}

	samplePath := filepath.Join(outputRoot, "config", "samples", "mysql_v1beta1_dbsystem.yaml")
	if err := os.MkdirAll(filepath.Dir(samplePath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(samplePath), err)
	}
	if err := os.WriteFile(samplePath, []byte(staleMySQLDbSystemSample), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", samplePath, err)
	}

	pipeline := New()
	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{*mysqlService}, Options{
		OutputRoot: outputRoot,
		Overwrite:  true,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if got := readFile(t, installPath); got == staleMySQLInstallKustomization {
		t.Fatalf("Generate() preserved stale install kustomization:\n%s", got)
	}
	if got := readFile(t, samplePath); got == staleMySQLDbSystemSample {
		t.Fatalf("Generate() preserved stale sample manifest:\n%s", got)
	}

	assertExactFileMatch(t, filepath.Join(repoRoot(t), "packages", "mysql", "install", "kustomization.yaml"), installPath)
	assertExactFileMatch(t, filepath.Join(repoRoot(t), "config", "samples", "mysql_v1beta1_dbsystem.yaml"), samplePath)
}

func TestGenerateOverwritePreservesCompanionArtifactsOutsideGeneratorOwnership(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
	}
	service := testServiceConfig(PackageProfileControllerBacked)

	outputRoot := t.TempDir()

	deepcopyPath := filepath.Join(outputRoot, "api", "mysql", "v1beta1", "zz_generated.deepcopy.go")
	if err := os.MkdirAll(filepath.Dir(deepcopyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(deepcopyPath), err)
	}
	if err := os.WriteFile(deepcopyPath, []byte("// preserved deepcopy companion\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", deepcopyPath, err)
	}

	packageGeneratedPath := filepath.Join(outputRoot, "packages", "mysql", "install", "generated", "crd", "bases", "mysql.oracle.com_dbsystems.yaml")
	if err := os.MkdirAll(filepath.Dir(packageGeneratedPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(packageGeneratedPath), err)
	}
	if err := os.WriteFile(packageGeneratedPath, []byte("preserved package manifest\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", packageGeneratedPath, err)
	}

	pipeline := newTestGenerator(t)
	if _, err := pipeline.Generate(context.Background(), cfg, []ServiceConfig{service}, Options{
		OutputRoot: outputRoot,
		Overwrite:  true,
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	assertPathExists(t, deepcopyPath)
	assertPathExists(t, packageGeneratedPath)
	if got := readFile(t, deepcopyPath); got != "// preserved deepcopy companion\n" {
		t.Fatalf("deepcopy content = %q, want preserved content", got)
	}
	if got := readFile(t, packageGeneratedPath); got != "preserved package manifest\n" {
		t.Fatalf("package manifest content = %q, want preserved content", got)
	}
}

const staleMySQLInstallKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- generated/crd
- preserved/editor-role.yaml
`

const staleMySQLDbSystemSample = `apiVersion: mysql.oracle.com/v1beta1
kind: DbSystem
metadata:
  name: preserved-dbsystem
spec:
  adminPassword:
    valueFrom:
      secretKeyRef:
        name: old-secret
        key: password
`
