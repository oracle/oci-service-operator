/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package privateendpoint

import "testing"

func TestPrivateEndpointRuntimeSemantics(t *testing.T) {
	semantics := privateEndpointRuntimeSemantics()
	if semantics == nil || semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("PrivateEndpoint semantics = %+v, want confirmed delete", semantics)
	}
	if semantics.Async == nil || semantics.Async.Runtime != "generatedruntime" {
		t.Fatalf("PrivateEndpoint async semantics = %+v, want generatedruntime lifecycle", semantics.Async)
	}
	if len(semantics.Mutation.ForceNew) != 3 {
		t.Fatalf("PrivateEndpoint force-new fields = %v, want compartmentId, subnetId, and vcnId", semantics.Mutation.ForceNew)
	}
}
