/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func loadCheckedInConfig(t *testing.T) *Config {
	t.Helper()

	cfgPath := filepath.Join(repoRoot(t), "internal", "generator", "config", "services.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", cfgPath, err)
	}

	return cfg
}

func serviceConfigsByName(t *testing.T, cfg *Config, names ...string) map[string]*ServiceConfig {
	t.Helper()

	wanted := make(map[string]struct{}, len(names))
	for _, name := range names {
		wanted[name] = struct{}{}
	}

	found := make(map[string]*ServiceConfig, len(names))
	for i := range cfg.Services {
		service := &cfg.Services[i]
		if _, ok := wanted[service.Service]; ok {
			found[service.Service] = service
		}
	}

	missing := missingServiceNames(found, names)
	if len(missing) != 0 {
		t.Fatalf("services %v were not found in services.yaml", missing)
	}

	return found
}

type selectedKindTarget struct {
	Service string
	Kind    string
}

func defaultActiveExplicitSelectedKindTargets(cfg *Config) []selectedKindTarget {
	if cfg == nil {
		return nil
	}

	targets := make([]selectedKindTarget, 0)
	for _, service := range cfg.Services {
		if !service.IsDefaultActive() || service.DefaultSelectionMode() != SelectionModeExplicit {
			continue
		}
		for _, kind := range service.defaultSelectedKinds() {
			targets = append(targets, selectedKindTarget{
				Service: service.Service,
				Kind:    kind,
			})
		}
	}
	return targets
}

func defaultActiveLifecycleGeneratedRuntimeFormalTargets(cfg *Config) []selectedKindTarget {
	if cfg == nil {
		return nil
	}

	targets := make([]selectedKindTarget, 0)
	for _, service := range cfg.Services {
		if !service.IsDefaultActive() || service.DefaultSelectionMode() != SelectionModeExplicit {
			continue
		}
		for _, kind := range service.defaultSelectedKinds() {
			async := service.AsyncConfigFor(kind)
			if async.Strategy != AsyncStrategyLifecycle || async.Runtime != AsyncRuntimeGeneratedRuntime {
				continue
			}
			if service.ServiceManagerGenerationStrategyFor(kind) != GenerationStrategyGenerated {
				continue
			}
			if strings.TrimSpace(service.FormalSpecFor(kind)) == "" {
				continue
			}
			targets = append(targets, selectedKindTarget{
				Service: service.Service,
				Kind:    kind,
			})
		}
	}
	return targets
}

func assertAsyncContract(t *testing.T, service *ServiceConfig, kind string, wantStrategy string, wantRuntime string) AsyncConfig {
	t.Helper()

	got := service.AsyncConfigFor(kind)
	if got.Strategy != wantStrategy {
		t.Fatalf("%s %s async strategy = %q, want %q", service.Service, kind, got.Strategy, wantStrategy)
	}
	if got.Runtime != wantRuntime {
		t.Fatalf("%s %s async runtime = %q, want %q", service.Service, kind, got.Runtime, wantRuntime)
	}
	if got.FormalClassification != wantStrategy {
		t.Fatalf("%s %s formalClassification = %q, want %q", service.Service, kind, got.FormalClassification, wantStrategy)
	}
	return got
}

func missingServiceNames(found map[string]*ServiceConfig, names []string) []string {
	missing := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := found[name]; !ok {
			missing = append(missing, name)
		}
	}

	return missing
}

func assertFieldNamesPresent(t *testing.T, label string, fields []FieldModel, want ...string) {
	t.Helper()

	for _, name := range want {
		if !hasField(fields, name) {
			t.Fatalf("%s = %#v, want %s", label, fields, name)
		}
	}
}

func assertFieldNamesAbsent(t *testing.T, label string, fields []FieldModel, unwanted ...string) {
	t.Helper()

	for _, name := range unwanted {
		if hasField(fields, name) {
			t.Fatalf("%s = %#v, want no %s", label, fields, name)
		}
	}
}

func assertResourceSpecFields(t *testing.T, resource ResourceModel, want ...string) {
	t.Helper()
	assertFieldNamesPresent(t, resource.Kind+" spec fields", resource.SpecFields, want...)
}

func assertResourceSpecFieldsAbsent(t *testing.T, resource ResourceModel, unwanted ...string) {
	t.Helper()
	assertFieldNamesAbsent(t, resource.Kind+" spec fields", resource.SpecFields, unwanted...)
}

func assertResourceStatusFields(t *testing.T, resource ResourceModel, want ...string) {
	t.Helper()
	assertFieldNamesPresent(t, resource.Kind+" status fields", resource.StatusFields, want...)
}

func assertHelperTypeFields(t *testing.T, helperType TypeModel, want ...string) {
	t.Helper()
	assertFieldNamesPresent(t, helperType.Name+" fields", helperType.Fields, want...)
}

func assertPackageResourceStatusFields(t *testing.T, pkg *PackageModel, want map[string][]string) {
	t.Helper()

	for kind, fieldNames := range want {
		assertResourceStatusFields(t, findResource(t, pkg.Resources, kind), fieldNames...)
	}
}

func assertServiceFormalSpec(t *testing.T, service *ServiceConfig, kind string, want string) {
	t.Helper()

	if got := service.FormalSpecFor(kind); got != want {
		t.Fatalf("%s %s formalSpec = %q, want %q", service.Service, kind, got, want)
	}
}

func assertFieldType(t *testing.T, label string, field FieldModel, want string) {
	t.Helper()

	if field.Type != want {
		t.Fatalf("%s type = %q, want %q", label, field.Type, want)
	}
}

func assertFieldTag(t *testing.T, label string, field FieldModel, want string) {
	t.Helper()

	if field.Tag != want {
		t.Fatalf("%s tag = %q, want %q", label, field.Tag, want)
	}
}

func assertFieldCommentsEqual(t *testing.T, label string, field FieldModel, want []string) {
	t.Helper()

	if !slices.Equal(field.Comments, want) {
		t.Fatalf("%s comments = %#v, want %#v", label, field.Comments, want)
	}
}

func assertFieldCommentsContain(t *testing.T, label string, field FieldModel, want string) {
	t.Helper()

	if !strings.Contains(strings.Join(field.Comments, "\n"), want) {
		t.Fatalf("%s comments = %#v, want substring %q", label, field.Comments, want)
	}
}

func assertFieldMarkers(t *testing.T, label string, field FieldModel, want []string) {
	t.Helper()

	if !slices.Equal(field.Markers, want) {
		t.Fatalf("%s markers = %#v, want %#v", label, field.Markers, want)
	}
}

func assertFieldHasNoMarkers(t *testing.T, label string, field FieldModel) {
	t.Helper()

	if len(field.Markers) != 0 {
		t.Fatalf("%s markers = %#v, want none", label, field.Markers)
	}
}

func seedSamplesKustomization(t *testing.T, outputRoot string) {
	t.Helper()

	samplesDir := filepath.Join(outputRoot, "config", "samples")
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", samplesDir, err)
	}

	checkedInSampleKustomization := readFile(t, filepath.Join(repoRoot(t), "config", "samples", "kustomization.yaml"))
	if err := os.WriteFile(filepath.Join(samplesDir, "kustomization.yaml"), []byte(checkedInSampleKustomization), 0o644); err != nil {
		t.Fatalf("write seeded samples kustomization: %v", err)
	}
}

func assertGeneratedServiceCounts(t *testing.T, generated []ServiceResult, want map[string]int) {
	t.Helper()

	if len(generated) != len(want) {
		t.Fatalf("generated %d services, want %d", len(generated), len(want))
	}

	for _, service := range generated {
		if service.ResourceCount != want[service.Service] {
			t.Fatalf("service %s generated %d resources, want %d", service.Service, service.ResourceCount, want[service.Service])
		}
	}
}

func assertGeneratedGoMatchesAll(t *testing.T, wantRoot string, gotRoot string, relativePaths []string) {
	t.Helper()

	for _, relativePath := range relativePaths {
		assertGoEquivalent(t, filepath.Join(wantRoot, relativePath), filepath.Join(gotRoot, relativePath))
	}
}

func assertExactFileMatchesAll(t *testing.T, wantRoot string, gotRoot string, relativePaths []string) {
	t.Helper()

	for _, relativePath := range relativePaths {
		assertExactFileMatch(t, filepath.Join(wantRoot, relativePath), filepath.Join(gotRoot, relativePath))
	}
}

func collectGeneratorOwnedRelativePaths(t *testing.T, root string) ([]string, []string) {
	t.Helper()

	goPaths := make([]string, 0)
	exactPaths := make([]string, 0)

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		name := entry.Name()

		switch {
		case strings.HasPrefix(relPath, "api/") && (name == "groupversion_info.go" || strings.HasSuffix(name, "_types.go")):
			goPaths = append(goPaths, relPath)
		case strings.HasPrefix(relPath, "controllers/") && strings.HasSuffix(name, "_controller.go"):
			goPaths = append(goPaths, relPath)
		case strings.HasPrefix(relPath, "pkg/servicemanager/") && (strings.HasSuffix(name, "_serviceclient.go") || strings.HasSuffix(name, "_servicemanager.go")):
			goPaths = append(goPaths, relPath)
		case strings.HasPrefix(relPath, "internal/registrations/") && strings.HasSuffix(name, "_generated.go"):
			goPaths = append(goPaths, relPath)
		case strings.HasPrefix(relPath, "cmd/manager/") && name == "main.go":
			goPaths = append(goPaths, relPath)
		case strings.HasPrefix(relPath, "packages/") && (name == "metadata.env" || strings.HasSuffix(relPath, "/install/kustomization.yaml")):
			exactPaths = append(exactPaths, relPath)
		case strings.HasPrefix(relPath, "config/manager/") && (name == "kustomization.yaml" || name == "manager.yaml" || name == "controller_manager_config.yaml"):
			exactPaths = append(exactPaths, relPath)
		case strings.HasPrefix(relPath, "config/samples/") && filepath.Ext(name) == ".yaml":
			exactPaths = append(exactPaths, relPath)
		case strings.HasPrefix(relPath, filepath.ToSlash(mutabilityOverlayGeneratedRootRelativePath)+"/") && filepath.Ext(name) == ".json":
			exactPaths = append(exactPaths, relPath)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%q) error = %v", root, err)
	}

	slices.Sort(goPaths)
	slices.Sort(exactPaths)
	return goPaths, exactPaths
}

func collectDesiredGeneratorOwnedRelativePaths(t *testing.T, cfg *Config, services []ServiceConfig) ([]string, []string) {
	t.Helper()

	root := t.TempDir()
	pipeline := New()
	packages := make([]*PackageModel, 0, len(services))
	for _, service := range services {
		pkg, err := pipeline.discoverer.BuildPackageModel(context.Background(), cfg, service)
		if err != nil {
			t.Fatalf("BuildPackageModel(%q) error = %v", service.Service, err)
		}
		packages = append(packages, pkg)
	}

	inventory, err := buildCleanupInventory(root, services, packages, nil, nil, false)
	if err != nil {
		t.Fatalf("buildCleanupInventory() error = %v", err)
	}

	goPaths := make([]string, 0, len(inventory.apiFiles)+len(inventory.controllerFiles)+len(inventory.registrationFiles)+len(inventory.serviceManagerFiles)+len(inventory.managerCmdFiles))
	exactPaths := make([]string, 0, len(inventory.managerConfigFiles)+len(inventory.sampleFiles)+(len(inventory.packageGroups)*2)+1)
	appendRelativePaths := func(dst []string, paths map[string]struct{}) []string {
		for path := range paths {
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				t.Fatalf("Rel(%q, %q) error = %v", root, path, err)
			}
			dst = append(dst, filepath.ToSlash(relPath))
		}
		return dst
	}

	goPaths = appendRelativePaths(goPaths, inventory.apiFiles)
	goPaths = appendRelativePaths(goPaths, inventory.controllerFiles)
	goPaths = appendRelativePaths(goPaths, inventory.registrationFiles)
	goPaths = appendRelativePaths(goPaths, inventory.serviceManagerFiles)
	goPaths = appendRelativePaths(goPaths, inventory.managerCmdFiles)

	exactPaths = appendRelativePaths(exactPaths, inventory.managerConfigFiles)
	exactPaths = appendRelativePaths(exactPaths, inventory.sampleFiles)
	if len(inventory.sampleFiles) != 0 {
		exactPaths = append(exactPaths, "config/samples/kustomization.yaml")
	}
	for group := range inventory.packageGroups {
		exactPaths = append(exactPaths,
			filepath.ToSlash(filepath.Join("packages", group, "metadata.env")),
			filepath.ToSlash(filepath.Join("packages", group, "install", "kustomization.yaml")),
		)
	}

	slices.Sort(goPaths)
	slices.Sort(exactPaths)
	return goPaths, exactPaths
}

func assertRelativePathSetEqual(t *testing.T, label string, got []string, want []string) {
	t.Helper()

	if slices.Equal(got, want) {
		return
	}

	extra := make([]string, 0)
	missing := make([]string, 0)
	gotSet := make(map[string]struct{}, len(got))
	wantSet := make(map[string]struct{}, len(want))
	for _, path := range got {
		gotSet[path] = struct{}{}
	}
	for _, path := range want {
		wantSet[path] = struct{}{}
	}
	for _, path := range got {
		if _, ok := wantSet[path]; !ok {
			extra = append(extra, path)
		}
	}
	for _, path := range want {
		if _, ok := gotSet[path]; !ok {
			missing = append(missing, path)
		}
	}
	slices.Sort(extra)
	slices.Sort(missing)
	t.Fatalf("%s mismatch\nextra: %v\nmissing: %v", label, extra, missing)
}

func assertFileContains(t *testing.T, path string, want []string) {
	t.Helper()
	assertContains(t, readFile(t, path), want)
}

func assertFileDoesNotContain(t *testing.T, path string, unwanted []string) {
	t.Helper()
	assertNotContains(t, readFile(t, path), unwanted)
}
