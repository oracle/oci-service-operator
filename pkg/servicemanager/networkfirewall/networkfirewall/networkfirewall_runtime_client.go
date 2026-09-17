/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networkfirewall

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerNetworkFirewallRuntimeHooksMutator(func(_ *NetworkFirewallServiceManager, hooks *NetworkFirewallRuntimeHooks) {
		if hooks != nil {
			hooks.Semantics = networkFirewallRuntimeSemantics()
		}
	})
}

func networkFirewallRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "networkfirewall", FormalSlug: "networkfirewall", StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Async:          &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"compartmentId", "displayName"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"displayName", "networkFirewallPolicyId", "networkSecurityGroupIds", "natConfiguration", "shape", "freeformTags", "definedTags"}, ForceNew: []string{"compartmentId", "subnetId", "availabilityDomain", "ipv4Address", "ipv6Address"}, ConflictsWith: map[string][]string{}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"}, UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"}, DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
