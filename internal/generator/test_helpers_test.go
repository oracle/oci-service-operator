/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type generationStrategyExpectation struct {
	controller     string
	serviceManager string
	registration   string
	webhook        string
}

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

func missingServiceNames(found map[string]*ServiceConfig, names []string) []string {
	missing := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := found[name]; !ok {
			missing = append(missing, name)
		}
	}

	return missing
}

func assertServiceGenerationStrategies(t *testing.T, service *ServiceConfig, want generationStrategyExpectation) {
	t.Helper()

	if got := service.ControllerGenerationStrategy(); got != want.controller {
		t.Fatalf("%s controller strategy = %q, want %q", service.Service, got, want.controller)
	}
	if got := service.ServiceManagerGenerationStrategy(); got != want.serviceManager {
		t.Fatalf("%s service-manager strategy = %q, want %q", service.Service, got, want.serviceManager)
	}
	if got := service.RegistrationGenerationStrategy(); got != want.registration {
		t.Fatalf("%s registration strategy = %q, want %q", service.Service, got, want.registration)
	}
	if got := service.WebhookGenerationStrategy(); got != want.webhook {
		t.Fatalf("%s webhook strategy = %q, want %q", service.Service, got, want.webhook)
	}
}

func resourceGenerationOverridesByKind(overrides []ResourceGenerationOverride) map[string]ResourceGenerationOverride {
	indexed := make(map[string]ResourceGenerationOverride, len(overrides))
	for _, override := range overrides {
		indexed[override.Kind] = override
	}

	return indexed
}

func assertResourceGenerationDisabled(t *testing.T, service *ServiceConfig, kinds ...string) {
	t.Helper()

	overrides := resourceGenerationOverridesByKind(service.Generation.Resources)
	for _, kind := range kinds {
		override, ok := overrides[kind]
		if !ok {
			t.Fatalf("%s override for %s was not found", service.Service, kind)
		}
		if override.Controller.Strategy != GenerationStrategyNone {
			t.Fatalf("%s %s controller strategy = %q, want %q", service.Service, kind, override.Controller.Strategy, GenerationStrategyNone)
		}
		if override.ServiceManager.Strategy != GenerationStrategyNone {
			t.Fatalf("%s %s service-manager strategy = %q, want %q", service.Service, kind, override.ServiceManager.Strategy, GenerationStrategyNone)
		}
	}
}

func assertServiceFormalSpec(t *testing.T, service *ServiceConfig, kind string, want string) {
	t.Helper()

	if got := service.FormalSpecFor(kind); got != want {
		t.Fatalf("%s %s formalSpec = %q, want %q", service.Service, kind, got, want)
	}
}

func assertSelectServicesResult(t *testing.T, cfg *Config, serviceName string, all bool, wantCount int, wantErr string) {
	t.Helper()

	services, err := cfg.SelectServices(serviceName, all)
	if wantErr != "" {
		if err == nil {
			t.Fatalf("SelectServices() error = nil, want %q", wantErr)
		}
		if !strings.Contains(err.Error(), wantErr) {
			t.Fatalf("SelectServices() error = %v, want substring %q", err, wantErr)
		}
		return
	}

	if err != nil {
		t.Fatalf("SelectServices() error = %v", err)
	}
	if len(services) != wantCount {
		t.Fatalf("SelectServices() returned %d services, want %d", len(services), wantCount)
	}
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

func readSampleKustomizationOrder(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	resources := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		resources = append(resources, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
	}

	return resources, nil
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

func assertFileContains(t *testing.T, path string, want []string) {
	t.Helper()
	assertContains(t, readFile(t, path), want)
}

func assertFileDoesNotContain(t *testing.T, path string, unwanted []string) {
	t.Helper()
	assertNotContains(t, readFile(t, path), unwanted)
}
