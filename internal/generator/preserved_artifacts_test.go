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
