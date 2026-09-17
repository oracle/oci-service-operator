/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package targetresponderrecipe

import "testing"

func TestTargetResponderRecipeRuntimeSemanticsAreStateFree(t *testing.T) {
	semantics := newTargetResponderRecipeRuntimeSemantics()
	if semantics == nil || semantics.Async == nil || semantics.Async.Strategy != "none" || semantics.List == nil {
		t.Fatalf("TargetResponderRecipe semantics = %#v", semantics)
	}
}
