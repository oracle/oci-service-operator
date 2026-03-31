/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import "testing"

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

func assertFieldNamesPresent(t *testing.T, label string, fields []FieldModel, want ...string) {
	t.Helper()

	for _, name := range want {
		if !hasField(fields, name) {
			t.Fatalf("%s = %#v, want %s", label, fields, name)
		}
	}
}

func assertHelperTypeFields(t *testing.T, helperType TypeModel, want ...string) {
	t.Helper()
	assertFieldNamesPresent(t, helperType.Name+" fields", helperType.Fields, want...)
}

func assertFileContains(t *testing.T, path string, want []string) {
	t.Helper()
	assertContains(t, readFile(t, path), want)
}

func assertFileDoesNotContain(t *testing.T, path string, unwanted []string) {
	t.Helper()
	assertNotContains(t, readFile(t, path), unwanted)
}
