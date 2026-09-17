/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package targetdetectorrecipe

import "testing"

func TestTargetDetectorRecipeRuntimeSemanticsPreserveParentIdentity(t *testing.T) {
	semantics := newTargetDetectorRecipeRuntimeSemantics()
	if semantics == nil || semantics.List == nil || len(semantics.Mutation.ForceNew) != 3 {
		t.Fatalf("TargetDetectorRecipe semantics = %#v", semantics)
	}
}
