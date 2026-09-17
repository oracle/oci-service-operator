/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package unifiedauditpolicy

import "testing"

func TestUnifiedAuditPolicyRuntimeSemanticsMapStatusCollision(t *testing.T) {
	semantics := newUnifiedAuditPolicyRuntimeSemantics()
	if semantics == nil || semantics.List == nil || len(semantics.Lifecycle.ActiveStates) != 2 {
		t.Fatalf("UnifiedAuditPolicy semantics = %#v", semantics)
	}
}
