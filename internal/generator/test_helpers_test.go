/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"strings"
	"testing"
)

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
