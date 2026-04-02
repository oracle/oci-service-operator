/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"
	"testing/quick"
)

type quickServiceSelectionCase struct {
	name  string
	group string
	kind  string
}

type quickSampleCleanupCase struct {
	root       string
	samplesDir string
	inventory  cleanupInventory
	want       []string
}

func TestSelectServicesQuickAppliesDefaultSurfaceAndExplicitOverride(t *testing.T) {
	t.Parallel()

	candidates := []quickServiceSelectionCase{
		{name: "alpha", group: "alpha", kind: "AlphaWidget"},
		{name: "beta", group: "beta", kind: "BetaWidget"},
		{name: "gamma", group: "gamma", kind: "GammaWidget"},
		{name: "delta", group: "delta", kind: "DeltaWidget"},
	}

	property := func(enabledMask uint8, explicitMask uint8, target uint8) bool {
		cfg := quickSelectionConfig(candidates, enabledMask, explicitMask)

		gotAll, err := cfg.SelectServices("", true)
		if err != nil {
			return false
		}
		if !quickSelectedServicesEqual(gotAll, quickExpectedDefaultServices(cfg.Services, enabledMask, explicitMask)) {
			return false
		}

		targetIndex := int(target % uint8(len(candidates)))
		gotOne, err := cfg.SelectServices(candidates[targetIndex].name, false)
		if err != nil || len(gotOne) != 1 {
			return false
		}
		return gotOne[0].Service == candidates[targetIndex].name &&
			gotOne[0].SelectedKinds() == nil
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 96}); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupSampleOutputsQuickKeepsOnlyDesiredGeneratedFiles(t *testing.T) {
	t.Parallel()

	generated := []string{
		sampleFileName("alpha", "v1beta1", "widget"),
		sampleFileName("alpha", "v1beta1", "report"),
		sampleFileName("beta", "v1alpha1", "widget"),
		sampleFileName("beta", "v1beta1", "report"),
		sampleFileName("gamma", "v1beta1", "item"),
	}
	manual := []string{
		"existing.yaml",
		"manual-notes.txt",
		"kustomization.yaml",
		"seeded.yaml",
	}

	property := func(existingGeneratedMask uint8, existingManualMask uint8, desiredMask uint8) bool {
		sampleCase, err := quickSampleCleanupCaseForMasks(
			t.TempDir(),
			generated,
			manual,
			existingGeneratedMask,
			existingManualMask,
			desiredMask,
		)
		if err != nil {
			return false
		}

		if err := cleanupSampleOutputs(sampleCase.root, sampleCase.inventory); err != nil {
			return false
		}

		got, err := quickDirFiles(sampleCase.samplesDir)
		if err != nil {
			return false
		}
		return slices.Equal(got, sampleCase.want)
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 96}); err != nil {
		t.Fatal(err)
	}
}

func quickSelectionConfig(candidates []quickServiceSelectionCase, enabledMask uint8, explicitMask uint8) *Config {
	services := make([]ServiceConfig, 0, len(candidates))
	for i, candidate := range candidates {
		enabled := quickMaskBit(enabledMask, i)
		explicit := quickMaskBit(explicitMask, i)

		selection := selectionAll(enabled)
		if explicit {
			selection = selectionExplicit(enabled, candidate.kind)
		}

		services = append(services, ServiceConfig{
			Service:        candidate.name,
			SDKPackage:     "example.com/" + candidate.name,
			Group:          candidate.group,
			PackageProfile: "controller-backed",
			Selection:      selection,
		})
	}

	return &Config{
		SchemaVersion:  "v1alpha1",
		Domain:         "oracle.com",
		DefaultVersion: "v1beta1",
		PackageProfiles: map[string]PackageProfile{
			"controller-backed": {Description: "runtime-integrated groups"},
		},
		Services: services,
	}
}

func quickExpectedDefaultServices(services []ServiceConfig, enabledMask uint8, explicitMask uint8) []ServiceConfig {
	selected := make([]ServiceConfig, 0, len(services))
	for i, service := range services {
		if !quickMaskBit(enabledMask, i) {
			continue
		}
		if quickMaskBit(explicitMask, i) {
			selected = append(selected, service.withSelectedKinds(service.DefaultIncludeKinds()))
			continue
		}
		selected = append(selected, service.withSelectedKinds(nil))
	}
	return selected
}

func quickSelectedServicesEqual(got []ServiceConfig, want []ServiceConfig) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i].Service != want[i].Service {
			return false
		}
		if !slices.Equal(got[i].SelectedKinds(), want[i].SelectedKinds()) {
			return false
		}
	}
	return true
}

func quickFilesFromMask(candidates []string, mask uint8) []string {
	files := make([]string, 0, len(candidates))
	for i, candidate := range candidates {
		if quickMaskBit(mask, i) {
			files = append(files, candidate)
		}
	}
	return files
}

func quickMaskBit(mask uint8, index int) bool {
	return mask&(1<<index) != 0
}

func quickSampleCleanupCaseForMasks(
	root string,
	generated []string,
	manual []string,
	existingGeneratedMask uint8,
	existingManualMask uint8,
	desiredMask uint8,
) (quickSampleCleanupCase, error) {
	samplesDir := filepath.Join(root, "config", "samples")
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		return quickSampleCleanupCase{}, err
	}

	inventory := quickSampleCleanupInventory()
	existingGenerated := quickFilesFromMask(generated, existingGeneratedMask)
	existingManual := quickFilesFromMask(manual, existingManualMask)
	desiredGenerated := quickFilesFromMask(generated, desiredMask)

	if err := quickSeedCleanupFiles(samplesDir, existingGenerated); err != nil {
		return quickSampleCleanupCase{}, err
	}
	if err := quickSeedCleanupFiles(samplesDir, existingManual); err != nil {
		return quickSampleCleanupCase{}, err
	}
	quickRegisterDesiredSampleFiles(samplesDir, &inventory, desiredGenerated)

	return quickSampleCleanupCase{
		root:       root,
		samplesDir: samplesDir,
		inventory:  inventory,
		want:       quickExpectedSampleFiles(samplesDir, existingGenerated, existingManual, inventory.sampleFiles),
	}, nil
}

func quickSampleCleanupInventory() cleanupInventory {
	inventory := newCleanupInventory(nil)
	for _, group := range []string{"alpha", "beta", "gamma"} {
		inventory.addSamplePrefix(sampleGroupPrefix(group))
	}
	return inventory
}

func quickSeedCleanupFiles(samplesDir string, names []string) error {
	for _, name := range names {
		if err := writeQuickCleanupFile(filepath.Join(samplesDir, name)); err != nil {
			return err
		}
	}
	return nil
}

func quickRegisterDesiredSampleFiles(samplesDir string, inventory *cleanupInventory, names []string) {
	for _, name := range names {
		inventory.sampleFiles[filepath.Join(samplesDir, name)] = struct{}{}
	}
}

func quickExpectedSampleFiles(samplesDir string, existingGenerated []string, existingManual []string, desired map[string]struct{}) []string {
	wantSet := make(map[string]struct{}, len(existingManual)+len(existingGenerated))
	for _, name := range existingManual {
		wantSet[name] = struct{}{}
	}
	for _, name := range existingGenerated {
		if _, ok := desired[filepath.Join(samplesDir, name)]; ok {
			wantSet[name] = struct{}{}
		}
	}

	want := make([]string, 0, len(wantSet))
	for name := range wantSet {
		want = append(want, name)
	}
	sort.Strings(want)
	return want
}

func writeQuickCleanupFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("seed\n"), 0o644)
}

func quickDirFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	return files, nil
}
