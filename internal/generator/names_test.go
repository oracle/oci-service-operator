/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import "testing"

func TestSingularizeAndPluralize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		singular string
		plural   string
	}{
		{name: "simple s suffix", input: "Widgets", singular: "Widget", plural: "Widgets"},
		{name: "ies suffix", input: "Policies", singular: "Policy", plural: "Policies"},
		{name: "existing plural compatibility kind", input: "AutonomousDatabases", singular: "AutonomousDatabase", plural: "AutonomousDatabases"},
		{name: "status suffix is preserved", input: "AlarmsStatus", singular: "AlarmStatus", plural: "AlarmStatuses"},
		{name: "statuses singularize to status", input: "AlarmStatuses", singular: "AlarmStatus", plural: "AlarmStatuses"},
		{name: "stats stay plural", input: "Stats", singular: "Stats", plural: "Stats"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := singularize(test.input); got != test.singular {
				t.Fatalf("singularize(%q) = %q, want %q", test.input, got, test.singular)
			}
			if got := pluralize(test.singular); got != test.plural {
				t.Fatalf("pluralize(%q) = %q, want %q", test.singular, got, test.plural)
			}
		})
	}
}

func TestCompatibilityKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		rawName       string
		compatibility CompatibilityConfig
		wantKind      string
		wantMatch     bool
	}{
		{
			name:          "suffix match for mysql",
			rawName:       "DbSystem",
			compatibility: CompatibilityConfig{ExistingKinds: []string{"MySqlDbSystem"}},
			wantKind:      "MySqlDbSystem",
			wantMatch:     true,
		},
		{
			name:          "plural match for autonomous database",
			rawName:       "AutonomousDatabase",
			compatibility: CompatibilityConfig{ExistingKinds: []string{"AutonomousDatabases"}},
			wantKind:      "AutonomousDatabases",
			wantMatch:     true,
		},
		{
			name:          "no match",
			rawName:       "Widget",
			compatibility: CompatibilityConfig{ExistingKinds: []string{"Vault"}},
			wantMatch:     false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gotKind, gotMatch := compatibilityKind(test.rawName, test.compatibility)
			if gotMatch != test.wantMatch {
				t.Fatalf("compatibilityKind(%q) match = %v, want %v", test.rawName, gotMatch, test.wantMatch)
			}
			if gotKind != test.wantKind {
				t.Fatalf("compatibilityKind(%q) kind = %q, want %q", test.rawName, gotKind, test.wantKind)
			}
		})
	}
}
