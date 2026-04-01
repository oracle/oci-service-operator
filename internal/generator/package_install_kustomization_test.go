/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type packageInstallKustomization struct {
	Resources []string `yaml:"resources"`
	Bases     []string `yaml:"bases"`
	Patches   []struct {
		Path string `yaml:"path"`
	} `yaml:"patches"`
	PatchesStrategicMerge []string `yaml:"patchesStrategicMerge"`
}

func TestCheckedInDatabaseAndMySQLPackageInstallKustomizationsReferenceExistingArtifacts(t *testing.T) {
	t.Parallel()

	for _, relativePath := range []string{
		"packages/database/install/kustomization.yaml",
		"packages/mysql/install/kustomization.yaml",
	} {
		relativePath := relativePath
		t.Run(relativePath, func(t *testing.T) {
			t.Parallel()
			assertKustomizationTreePathsExist(t, filepath.Join(repoRoot(t), relativePath), map[string]struct{}{})
		})
	}
}

func assertKustomizationTreePathsExist(t *testing.T, kustomizationPath string, visited map[string]struct{}) {
	t.Helper()

	absPath, err := filepath.Abs(kustomizationPath)
	if err != nil {
		t.Fatalf("Abs(%q) error = %v", kustomizationPath, err)
	}
	if _, ok := visited[absPath]; ok {
		return
	}
	visited[absPath] = struct{}{}

	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", absPath, err)
	}

	var cfg packageInstallKustomization
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", absPath, err)
	}

	baseDir := filepath.Dir(absPath)
	for _, resourcePath := range append(cfg.Resources, cfg.Bases...) {
		assertKustomizationEntryExists(t, baseDir, resourcePath, visited)
	}
	for _, patch := range cfg.Patches {
		if strings.TrimSpace(patch.Path) == "" {
			continue
		}
		assertKustomizationFileExists(t, baseDir, patch.Path)
	}
	for _, patchPath := range cfg.PatchesStrategicMerge {
		assertKustomizationFileExists(t, baseDir, patchPath)
	}
}

func assertKustomizationEntryExists(t *testing.T, baseDir string, entry string, visited map[string]struct{}) {
	t.Helper()

	if strings.TrimSpace(entry) == "" || strings.Contains(entry, "://") {
		return
	}

	absPath := filepath.Clean(filepath.Join(baseDir, entry))
	info, err := os.Stat(absPath)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", absPath, err)
	}
	if !info.IsDir() {
		return
	}

	kustomizationPath, ok := findKustomizationFile(absPath)
	if !ok {
		t.Fatalf("resource directory %q does not contain a kustomization file", absPath)
	}
	assertKustomizationTreePathsExist(t, kustomizationPath, visited)
}

func assertKustomizationFileExists(t *testing.T, baseDir string, relativePath string) {
	t.Helper()

	if strings.TrimSpace(relativePath) == "" || strings.Contains(relativePath, "://") {
		return
	}

	absPath := filepath.Clean(filepath.Join(baseDir, relativePath))
	if _, err := os.Stat(absPath); err != nil {
		t.Fatalf("Stat(%q) error = %v", absPath, err)
	}
}

func findKustomizationFile(dir string) (string, bool) {
	for _, name := range []string{"kustomization.yaml", "kustomization.yml", "Kustomization"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
}
