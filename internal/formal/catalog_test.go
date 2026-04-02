/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package formal

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadCatalogReturnsTypedControllerBindings(t *testing.T) {
	t.Parallel()

	root := writeScaffold(t)
	catalog, err := LoadCatalog(root)
	if err != nil {
		t.Fatalf("LoadCatalog(%q) error = %v", root, err)
	}

	if catalog.Root != filepath.Clean(root) {
		t.Fatalf("catalog root = %q, want %q", catalog.Root, filepath.Clean(root))
	}
	if len(catalog.Controllers) != 1 {
		t.Fatalf("len(catalog.Controllers) = %d, want 1", len(catalog.Controllers))
	}

	binding, ok := catalog.Lookup("template", "template")
	if !ok {
		t.Fatal("Lookup(template, template) unexpectedly missed")
	}
	if binding.Manifest.Kind != "Template" {
		t.Fatalf("binding kind = %q, want %q", binding.Manifest.Kind, "Template")
	}
	if binding.Spec.Kind != "Template" {
		t.Fatalf("binding spec kind = %q, want %q", binding.Spec.Kind, "Template")
	}
	if binding.Spec.StatusProjection != "required" {
		t.Fatalf("binding status projection = %q, want %q", binding.Spec.StatusProjection, "required")
	}
	if !slices.Equal(binding.Spec.SharedContracts, []string{
		"shared/BaseReconcilerContract.tla",
		"shared/ControllerLifecycleSpec.tla",
		"shared/OSOKServiceManagerContract.tla",
		"shared/SecretSideEffectsContract.tla",
	}) {
		t.Fatalf("binding shared contracts = %v", binding.Spec.SharedContracts)
	}
	if len(binding.LogicGaps) != 0 {
		t.Fatalf("binding logic gaps = %v, want empty scaffold gaps", binding.LogicGaps)
	}
	if binding.Import.ProviderResource != "template_resource" {
		t.Fatalf("binding provider resource = %q, want %q", binding.Import.ProviderResource, "template_resource")
	}
	if !slices.Equal(binding.Import.Lifecycle.Create.Target, []string{"ACTIVE"}) {
		t.Fatalf("binding create target states = %v, want ACTIVE", binding.Import.Lifecycle.Create.Target)
	}
}
