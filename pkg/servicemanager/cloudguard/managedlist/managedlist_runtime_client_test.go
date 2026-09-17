/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managedlist

import "testing"

func TestManagedListRuntimeSemantics(t *testing.T) {
	t.Parallel()
	semantics := managedListRuntimeSemantics()
	if semantics == nil || semantics.Async == nil || semantics.Async.Strategy != "lifecycle" {
		t.Fatalf("semantics = %#v", semantics)
	}
	if len(semantics.Lifecycle.ActiveStates) != 1 || semantics.Lifecycle.ActiveStates[0] != "ACTIVE" {
		t.Fatalf("active states = %#v", semantics.Lifecycle.ActiveStates)
	}
}
