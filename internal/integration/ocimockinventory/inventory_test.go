/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ocimockinventory

import "testing"

func TestHasFullCRUD(t *testing.T) {
	t.Parallel()

	full := []byte(`
Create: &generatedruntime.Operation{
Get: &generatedruntime.Operation{
List: &generatedruntime.Operation{
Update: &generatedruntime.Operation{
Delete: &generatedruntime.Operation{
`)
	if !hasFullCRUD(full) {
		t.Fatal("hasFullCRUD() = false")
	}
	if hasFullCRUD([]byte("Get: &generatedruntime.Operation{")) {
		t.Fatal("read-only resource reported full CRUD")
	}
}

func TestClassifyPrioritizesWorkRequestAndCompositeBehavior(t *testing.T) {
	t.Parallel()

	if got := classify(nil, []byte(`Strategy: "workrequest"`)); got != GroupWorkRequest {
		t.Fatalf("work-request group = %q", got)
	}
	if got := classify([]byte(`Contribution: "path", PreferResourceID: false`), nil); got != GroupComposite {
		t.Fatalf("composite group = %q", got)
	}
	if got := classify(nil, []byte(`ProvisioningStates: []string{"CREATING"}`)); got != GroupLifecycle {
		t.Fatalf("lifecycle group = %q", got)
	}
	if got := classify(nil, []byte(`return &generatedruntime.Semantics{}`)); got != GroupImmediate {
		t.Fatalf("immediate group = %q", got)
	}
	if got := classify(nil, nil); got != GroupUnclassified {
		t.Fatalf("unclassified group = %q", got)
	}
}

func TestMissingMockIntegrationFiltersOneGroup(t *testing.T) {
	t.Parallel()
	report := Report{Resources: []Resource{
		{Service: "first", Kind: "Covered", Group: GroupImmediate, MockIntegration: true},
		{Service: "second", Kind: "Missing", Group: GroupImmediate},
		{Service: "third", Kind: "Later", Group: GroupLifecycle},
	}}
	missing := MissingMockIntegration(report, GroupImmediate)
	if len(missing) != 1 || missing[0].Service != "second" || missing[0].Kind != "Missing" {
		t.Fatalf("MissingMockIntegration() = %+v", missing)
	}
}

func TestMissingFormalFiltersOneGroup(t *testing.T) {
	t.Parallel()
	report := Report{Resources: []Resource{
		{Service: "first", Kind: "Covered", Group: GroupLifecycle, Formal: true},
		{Service: "second", Kind: "Missing", Group: GroupLifecycle},
		{Service: "third", Kind: "Later", Group: GroupComposite},
	}}
	missing := MissingFormal(report, GroupLifecycle)
	if len(missing) != 1 || missing[0].Service != "second" || missing[0].Kind != "Missing" {
		t.Fatalf("MissingFormal() = %+v", missing)
	}
}
