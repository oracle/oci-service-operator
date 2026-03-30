/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type preservedFile struct {
	content []byte
	mode    os.FileMode
}

type preservedPackageArtifacts struct {
	packageFiles map[string]preservedFile
	sampleFiles  map[string]preservedFile
}

func loadPreservedPackageArtifacts(root string, pkg *PackageModel) (preservedPackageArtifacts, error) {
	preserved := preservedPackageArtifacts{}
	if strings.TrimSpace(root) == "" || pkg == nil {
		return preserved, nil
	}

	for _, resource := range pkg.Resources {
		if !resource.CompatibilityLocked {
			continue
		}

		resourcePath := filepath.Join("api", pkg.Service.Group, pkg.Version, resource.FileStem+"_types.go")
		file, err := readPreservedFile(filepath.Join(root, resourcePath))
		if err != nil {
			return preserved, fmt.Errorf("load checked-in API artifact %q: %w", resourcePath, err)
		}
		if file != nil {
			if preserved.packageFiles == nil {
				preserved.packageFiles = make(map[string]preservedFile)
			}
			preserved.packageFiles[resourcePath] = *file
		}

		if strings.TrimSpace(resource.Sample.FileName) == "" {
			continue
		}
		samplePath := filepath.Join("config", "samples", resource.Sample.FileName)
		file, err = readPreservedFile(filepath.Join(root, samplePath))
		if err != nil {
			return preserved, fmt.Errorf("load checked-in sample artifact %q: %w", samplePath, err)
		}
		if file != nil {
			if preserved.sampleFiles == nil {
				preserved.sampleFiles = make(map[string]preservedFile)
			}
			preserved.sampleFiles[samplePath] = *file
		}
	}

	if len(preserved.packageFiles) == 0 {
		return preserved, nil
	}

	installPath := filepath.Join("packages", pkg.Service.Group, "install", "kustomization.yaml")
	file, err := readPreservedFile(filepath.Join(root, installPath))
	if err != nil {
		return preserved, fmt.Errorf("load checked-in package artifact %q: %w", installPath, err)
	}
	if file != nil {
		preserved.packageFiles[installPath] = *file
	}

	return preserved, nil
}

func readPreservedFile(path string) (*preservedFile, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("expected file, found directory")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return &preservedFile{
		content: content,
		mode:    info.Mode().Perm(),
	}, nil
}

func (p preservedPackageArtifacts) applyPackageFiles(outputRoot string) error {
	return writePreservedFiles(outputRoot, p.packageFiles)
}

func (p preservedPackageArtifacts) applySampleFiles(outputRoot string) error {
	return writePreservedFiles(outputRoot, p.sampleFiles)
}

func writePreservedFiles(outputRoot string, files map[string]preservedFile) error {
	for relPath, file := range files {
		targetPath := filepath.Join(outputRoot, relPath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("create preserved artifact directory for %q: %w", targetPath, err)
		}
		if err := os.WriteFile(targetPath, file.content, file.mode); err != nil {
			return fmt.Errorf("write preserved artifact %q: %w", targetPath, err)
		}
	}
	return nil
}
