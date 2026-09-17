/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package batchtaskenvironment

import (
	"reflect"
	"testing"
)

func TestBatchTaskEnvironmentRuntimeSemantics(t *testing.T) {
	semantics := newBatchTaskEnvironmentRuntimeSemantics()
	if semantics == nil || semantics.List == nil {
		t.Fatal("BatchTaskEnvironment runtime semantics were not configured")
	}
	if got, want := semantics.Mutation.Mutable, []string{"displayName", "description", "freeformTags", "definedTags"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("mutable fields = %v, want %v", got, want)
	}
	if got, want := semantics.Mutation.ForceNew, []string{"compartmentId", "imageUrl", "securityContext", "workingDirectory", "volumes"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("force-new fields = %v, want %v", got, want)
	}
}
