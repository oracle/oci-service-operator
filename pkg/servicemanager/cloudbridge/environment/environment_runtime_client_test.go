/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package environment

import (
	"reflect"
	"testing"
)

func TestEnvironmentRuntimeSemantics(t *testing.T) {
	t.Parallel()
	semantics := environmentRuntimeSemantics()
	if semantics == nil || semantics.Async == nil || semantics.Async.Strategy != "lifecycle" {
		t.Fatalf("semantics = %#v, want lifecycle runtime", semantics)
	}
	if !reflect.DeepEqual(semantics.Lifecycle.ActiveStates, []string{"ACTIVE"}) {
		t.Fatalf("active states = %#v", semantics.Lifecycle.ActiveStates)
	}
	if !reflect.DeepEqual(semantics.Delete.TerminalStates, []string{"DELETED"}) {
		t.Fatalf("delete terminal states = %#v", semantics.Delete.TerminalStates)
	}
	if !reflect.DeepEqual(semantics.Mutation.Mutable, []string{"definedTags", "displayName", "freeformTags"}) {
		t.Fatalf("mutable fields = %#v", semantics.Mutation.Mutable)
	}
}
