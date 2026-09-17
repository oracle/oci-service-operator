/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package cccupgradeschedule

import "testing"

func TestReviewedCccUpgradeScheduleRuntimeSemantics(t *testing.T) {
	semantics := reviewedCccUpgradeScheduleRuntimeSemantics()
	if semantics == nil || semantics.Async == nil || semantics.Async.Strategy != "lifecycle" {
		t.Fatalf("runtime semantics = %#v, want lifecycle semantics", semantics)
	}
	if len(semantics.Lifecycle.ActiveStates) != 1 || semantics.Lifecycle.ActiveStates[0] != "ACTIVE" {
		t.Fatalf("active states = %v, want [ACTIVE]", semantics.Lifecycle.ActiveStates)
	}
	if len(semantics.Delete.TerminalStates) != 1 || semantics.Delete.TerminalStates[0] != "DELETED" {
		t.Fatalf("delete terminal states = %v, want [DELETED]", semantics.Delete.TerminalStates)
	}
}
