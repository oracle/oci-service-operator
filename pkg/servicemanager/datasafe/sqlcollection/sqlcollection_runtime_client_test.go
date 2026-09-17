/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sqlcollection

import "testing"

func TestSqlCollectionRuntimeSemanticsMapStatusCollision(t *testing.T) {
	semantics := newSqlCollectionRuntimeSemantics()
	if semantics == nil || semantics.List == nil || len(semantics.Lifecycle.ActiveStates) != 3 {
		t.Fatalf("SqlCollection semantics = %#v", semantics)
	}
}
