/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type cleanupInventory struct {
	apiFiles               map[string]struct{}
	controllerFiles        map[string]struct{}
	registrationFiles      map[string]struct{}
	serviceManagerFiles    map[string]struct{}
	packageGroups          map[string]struct{}
	sampleFiles            map[string]struct{}
	selectedGroups         map[string]struct{}
	selectedSamplePrefixes map[string]struct{}
	serviceManagerRoots    map[string]struct{}
}

func cleanupGeneratedOutputs(root string, cfg *Config, services []ServiceConfig, packages []*PackageModel) error {
	inventory := buildCleanupInventory(root, cfg, services, packages)

	if err := cleanupAPIOutputs(root, inventory); err != nil {
		return fmt.Errorf("cleanup api outputs: %w", err)
	}
	if err := cleanupControllerOutputs(root, inventory); err != nil {
		return fmt.Errorf("cleanup controller outputs: %w", err)
	}
	if err := cleanupRegistrationOutputs(root, inventory); err != nil {
		return fmt.Errorf("cleanup registration outputs: %w", err)
	}
	if err := cleanupServiceManagerOutputs(root, inventory); err != nil {
		return fmt.Errorf("cleanup service-manager outputs: %w", err)
	}
	if err := cleanupPackageOutputs(root, inventory); err != nil {
		return fmt.Errorf("cleanup package outputs: %w", err)
	}
	if err := cleanupSampleOutputs(root, inventory); err != nil {
		return fmt.Errorf("cleanup sample outputs: %w", err)
	}

	return nil
}

func buildCleanupInventory(root string, cfg *Config, services []ServiceConfig, packages []*PackageModel) cleanupInventory {
	inventory := cleanupInventory{
		apiFiles:               map[string]struct{}{},
		controllerFiles:        map[string]struct{}{},
		registrationFiles:      map[string]struct{}{},
		serviceManagerFiles:    map[string]struct{}{},
		packageGroups:          map[string]struct{}{},
		sampleFiles:            map[string]struct{}{},
		selectedGroups:         map[string]struct{}{},
		selectedSamplePrefixes: map[string]struct{}{},
		serviceManagerRoots:    selectedServiceManagerRoots(services),
	}

	for _, service := range services {
		inventory.selectedGroups[service.Group] = struct{}{}
		prefix := sampleFilePrefix(service.Group, service.VersionOrDefault(defaultVersion(cfg)))
		inventory.selectedSamplePrefixes[prefix] = struct{}{}
	}

	for _, pkg := range packages {
		apiDir := filepath.Join(root, "api", pkg.Service.Group, pkg.Version)
		inventory.apiFiles[filepath.Join(apiDir, "groupversion_info.go")] = struct{}{}

		for _, resource := range pkg.Resources {
			inventory.apiFiles[filepath.Join(apiDir, resource.FileStem+"_types.go")] = struct{}{}
			if strings.TrimSpace(resource.Sample.FileName) != "" {
				inventory.sampleFiles[filepath.Join(root, "config", "samples", resource.Sample.FileName)] = struct{}{}
			}
		}
		for _, controller := range pkg.Controller.Resources {
			inventory.controllerFiles[filepath.Join(root, "controllers", pkg.Service.Group, controller.FileStem+"_controller.go")] = struct{}{}
		}

		if len(pkg.Registration.Resources) > 0 {
			inventory.registrationFiles[filepath.Join(root, "internal", "registrations", pkg.Registration.Group+"_generated.go")] = struct{}{}
		}

		for _, serviceManager := range pkg.ServiceManagers {
			outputDir := filepath.Join(root, "pkg", "servicemanager", filepath.FromSlash(serviceManager.PackagePath))
			inventory.serviceManagerFiles[filepath.Join(outputDir, serviceManager.ServiceClientFileName)] = struct{}{}
			inventory.serviceManagerFiles[filepath.Join(outputDir, serviceManager.ServiceManagerFileName)] = struct{}{}
		}

		if pkg.PackageOutput.Generate {
			inventory.packageGroups[pkg.Service.Group] = struct{}{}
		}
	}

	return inventory
}

func defaultVersion(cfg *Config) string {
	if cfg == nil {
		return ""
	}
	return cfg.DefaultVersion
}

func cleanupAPIOutputs(root string, inventory cleanupInventory) error {
	apiRoot := filepath.Join(root, "api")
	for _, group := range scopedGroupEntries(inventory.selectedGroups) {
		groupDir := filepath.Join(apiRoot, group)
		if err := deleteGeneratedFiles(groupDir, apiRoot, inventory.apiFiles, isOwnedAPIFile); err != nil {
			return err
		}
	}
	return nil
}

func cleanupControllerOutputs(root string, inventory cleanupInventory) error {
	controllersRoot := filepath.Join(root, "controllers")
	for _, group := range scopedGroupEntries(inventory.selectedGroups) {
		groupDir := filepath.Join(controllersRoot, group)
		if err := deleteGeneratedFiles(groupDir, controllersRoot, inventory.controllerFiles, isOwnedControllerFile); err != nil {
			return err
		}
	}
	return nil
}

func cleanupRegistrationOutputs(root string, inventory cleanupInventory) error {
	registrationsRoot := filepath.Join(root, "internal", "registrations")
	entries, err := os.ReadDir(registrationsRoot)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !isOwnedRegistrationFile(entry.Name()) {
			continue
		}
		group := strings.TrimSuffix(entry.Name(), "_generated.go")
		if _, ok := inventory.selectedGroups[group]; !ok {
			continue
		}
		path := filepath.Join(registrationsRoot, entry.Name())
		if _, ok := inventory.registrationFiles[path]; ok {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func cleanupServiceManagerOutputs(root string, inventory cleanupInventory) error {
	serviceManagerRoot := filepath.Join(root, "pkg", "servicemanager")
	for _, scopeRoot := range scopedGroupEntries(inventory.serviceManagerRoots) {
		rootDir := filepath.Join(serviceManagerRoot, scopeRoot)
		if err := deleteGeneratedFiles(rootDir, serviceManagerRoot, inventory.serviceManagerFiles, isOwnedServiceManagerFile); err != nil {
			return err
		}
	}
	return nil
}

func cleanupPackageOutputs(root string, inventory cleanupInventory) error {
	packagesRoot := filepath.Join(root, "packages")
	for _, group := range scopedGroupEntries(inventory.selectedGroups) {
		groupDir := filepath.Join(packagesRoot, group)
		desired := false
		if _, ok := inventory.packageGroups[group]; ok {
			desired = true
		}
		if err := cleanupPackageGroup(groupDir, packagesRoot, desired); err != nil {
			return err
		}
	}
	return nil
}

func cleanupSampleOutputs(root string, inventory cleanupInventory) error {
	samplesDir := filepath.Join(root, "config", "samples")
	entries, err := os.ReadDir(samplesDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "kustomization.yaml" || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		path := filepath.Join(samplesDir, entry.Name())
		if _, ok := inventory.sampleFiles[path]; ok {
			continue
		}
		if !matchesSamplePrefix(entry.Name(), mapKeys(inventory.selectedSamplePrefixes)) {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func cleanupPackageGroup(groupDir string, packagesRoot string, desired bool) error {
	if _, err := os.Stat(groupDir); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	// cmd/generator owns the package metadata and install kustomization files.
	// install/generated is refreshed by downstream manifest targets.
	for _, path := range []string{
		filepath.Join(groupDir, "metadata.env"),
		filepath.Join(groupDir, "install", "kustomization.yaml"),
	} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	if desired {
		return pruneEmptyParents(filepath.Join(groupDir, "install"), packagesRoot)
	}
	return pruneEmptyParents(groupDir, packagesRoot)
}

func deleteGeneratedFiles(rootDir string, cleanupRoot string, desired map[string]struct{}, owned func(string) bool) error {
	if info, err := os.Lstat(rootDir); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	} else if info.Mode()&os.ModeSymlink != 0 {
		return os.Remove(rootDir)
	}

	return filepath.WalkDir(rootDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !owned(entry.Name()) {
			return nil
		}
		if _, ok := desired[path]; ok {
			return nil
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return pruneEmptyParents(filepath.Dir(path), cleanupRoot)
	})
}

func pruneEmptyParents(dir string, stop string) error {
	dir = filepath.Clean(dir)
	stop = filepath.Clean(stop)
	for {
		if dir == stop || dir == "." || dir == string(filepath.Separator) {
			return nil
		}

		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			dir = filepath.Dir(dir)
			continue
		}
		if err != nil {
			return err
		}
		if len(entries) != 0 {
			return nil
		}
		if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
			return err
		}
		dir = filepath.Dir(dir)
	}
}

func scopedGroupEntries(selected map[string]struct{}) []string {
	names := make([]string, 0, len(selected))
	for name := range selected {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func mapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func selectedServiceManagerRoots(services []ServiceConfig) map[string]struct{} {
	roots := make(map[string]struct{}, len(services))
	for _, service := range services {
		roots[service.Group] = struct{}{}
		for _, override := range service.Generation.Resources {
			packagePath := strings.TrimSpace(override.ServiceManager.PackagePath)
			if packagePath == "" {
				continue
			}
			root := strings.Split(filepath.ToSlash(packagePath), "/")[0]
			if strings.TrimSpace(root) == "" {
				continue
			}
			roots[root] = struct{}{}
		}
	}
	return roots
}

func isOwnedAPIFile(name string) bool {
	// Deepcopy companions are produced by controller-gen, not cmd/generator.
	return name == "groupversion_info.go" ||
		strings.HasSuffix(name, "_types.go")
}

func isOwnedControllerFile(name string) bool {
	return strings.HasSuffix(name, "_controller.go")
}

func isOwnedRegistrationFile(name string) bool {
	return strings.HasSuffix(name, "_generated.go")
}

func isOwnedServiceManagerFile(name string) bool {
	return strings.HasSuffix(name, "_serviceclient.go") ||
		strings.HasSuffix(name, "_servicemanager.go")
}
