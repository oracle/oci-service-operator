/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package catalog

import "testing"

func TestCatalogRuntimeSemanticsCoverLifecycleAndIdentity(t *testing.T) {
	semantics := newCatalogRuntimeSemantics()
	if semantics == nil || len(semantics.Lifecycle.ActiveStates) != 1 || semantics.Lifecycle.ActiveStates[0] != "ACTIVE" {
		t.Fatalf("Catalog semantics = %#v", semantics)
	}
	if semantics.List == nil || len(semantics.List.MatchFields) != 2 {
		t.Fatalf("Catalog list semantics = %#v", semantics.List)
	}
}
