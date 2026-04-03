/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
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
		{name: "vowel+y keeps s suffix", input: "ApiKeys", singular: "ApiKey", plural: "ApiKeys"},
		{name: "compound vowel+y keeps s suffix", input: "CustomerSecretKeys", singular: "CustomerSecretKey", plural: "CustomerSecretKeys"},
		{name: "index uses indices pluralization", input: "Indices", singular: "Index", plural: "Indices"},
		{name: "information stays unpluralized", input: "UserUIPasswordInformation", singular: "UserUIPasswordInformation", plural: "UserUIPasswordInformation"},
		{name: "ies suffix", input: "Policies", singular: "Policy", plural: "Policies"},
		{name: "autonomous database plural", input: "AutonomousDatabases", singular: "AutonomousDatabase", plural: "AutonomousDatabases"},
		{name: "ss suffix gains es", input: "ConnectHarnesses", singular: "ConnectHarness", plural: "ConnectHarnesses"},
		{name: "status suffix is preserved", input: "AlarmsStatus", singular: "AlarmStatus", plural: "AlarmStatuses"},
		{name: "statuses singularize to status", input: "AlarmStatuses", singular: "AlarmStatus", plural: "AlarmStatuses"},
		{name: "stats stay plural", input: "Stats", singular: "Stats", plural: "Stats"},
		{name: "metadata stays uncountable", input: "NamespaceMetadata", singular: "NamespaceMetadata", plural: "NamespaceMetadata"},
		{name: "ss suffix uses es pluralization", input: "ConnectHarnesses", singular: "ConnectHarness", plural: "ConnectHarnesses"},
		{name: "vowel y suffix uses s", input: "Gateways", singular: "Gateway", plural: "Gateways"},
		{name: "key suffix uses s", input: "Keys", singular: "Key", plural: "Keys"},
		{name: "ss suffix uses es", input: "Addresses", singular: "Address", plural: "Addresses"},
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
