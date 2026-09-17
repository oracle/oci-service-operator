/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networkfirewallpolicy

import (
	"reflect"
	"testing"
)

func TestNetworkFirewallPolicyRuntimeSemantics(t *testing.T) {
	semantics := networkFirewallPolicyRuntimeSemantics()
	if semantics.FinalizerPolicy != "retain-until-confirmed-delete" || semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("delete semantics = %+v", semantics)
	}
	if !reflect.DeepEqual(semantics.Delete.PendingStates, []string{"DELETING"}) || !reflect.DeepEqual(semantics.Delete.TerminalStates, []string{"DELETED"}) {
		t.Fatalf("delete lifecycle = %+v", semantics.Delete)
	}
	if !reflect.DeepEqual(semantics.Lifecycle.ActiveStates, []string{"ACTIVE"}) {
		t.Fatalf("active states = %v", semantics.Lifecycle.ActiveStates)
	}
}
