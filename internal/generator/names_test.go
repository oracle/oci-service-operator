/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"slices"
	"testing"
)

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
		{name: "existing plural published kind", input: "AutonomousDatabases", singular: "AutonomousDatabase", plural: "AutonomousDatabases"},
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

func TestSplitCamelAndLowerCamel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		tokens []string
		lower  string
	}{
		{
			name:   "acronym boundary",
			input:  "HTTPRequest",
			tokens: []string{"http", "request"},
			lower:  "httpRequest",
		},
		{
			name:   "multiple acronyms",
			input:  "MySQLDbSystem",
			tokens: []string{"my", "sql", "db", "system"},
			lower:  "mySqlDbSystem",
		},
		{
			name:   "digit boundary",
			input:  "OCI123Thing",
			tokens: []string{"oci123", "thing"},
			lower:  "oci123Thing",
		},
		{
			name:   "blank string",
			input:  "   ",
			tokens: nil,
			lower:  "",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := splitCamel(test.input); !slices.Equal(got, test.tokens) {
				t.Fatalf("splitCamel(%q) = %v, want %v", test.input, got, test.tokens)
			}
			if got := lowerCamel(test.input); got != test.lower {
				t.Fatalf("lowerCamel(%q) = %q, want %q", test.input, got, test.lower)
			}
		})
	}
}
