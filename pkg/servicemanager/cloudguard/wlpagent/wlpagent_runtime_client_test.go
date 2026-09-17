/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package wlpagent

import "testing"

func TestWlpAgentRuntimeSemanticsAreStateFreeAndScoped(t *testing.T) {
	semantics := newWlpAgentRuntimeSemantics()
	if semantics == nil || semantics.Async == nil || semantics.Async.Strategy != "none" {
		t.Fatalf("WlpAgent semantics = %#v", semantics)
	}
	if len(semantics.Mutation.ForceNew) != 3 || len(semantics.Mutation.Mutable) != 3 {
		t.Fatalf("WlpAgent mutation semantics = %#v", semantics.Mutation)
	}
}
