/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package batchtaskprofile

import (
	"reflect"
	"testing"
)

func TestBatchTaskProfileRuntimeSemantics(t *testing.T) {
	semantics := newBatchTaskProfileRuntimeSemantics()
	if semantics == nil || semantics.List == nil {
		t.Fatal("BatchTaskProfile runtime semantics were not configured")
	}
	if got, want := semantics.Mutation.Mutable, []string{"displayName", "description", "freeformTags", "definedTags"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("mutable fields = %v, want %v", got, want)
	}
	if got, want := semantics.Mutation.ForceNew, []string{"compartmentId", "minOcpus", "minMemoryInGBs"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("force-new fields = %v, want %v", got, want)
	}
}
