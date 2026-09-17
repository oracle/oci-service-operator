/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package privateendpoint

import (
	resourcemanagersdk "github.com/oracle/oci-go-sdk/v65/resourcemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerPrivateEndpointRuntimeHooksMutator(func(_ *PrivateEndpointServiceManager, hooks *PrivateEndpointRuntimeHooks) {
		hooks.Semantics = privateEndpointRuntimeSemantics()
	})
}

func privateEndpointRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "resourcemanager",
		FormalSlug:        "privateendpoint",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(resourcemanagersdk.PrivateEndpointLifecycleStateCreating)},
			ActiveStates:       []string{string(resourcemanagersdk.PrivateEndpointLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(resourcemanagersdk.PrivateEndpointLifecycleStateDeleting)},
			TerminalStates: []string{string(resourcemanagersdk.PrivateEndpointLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "vcnId", "subnetId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"definedTags",
				"description",
				"displayName",
				"dnsZones",
				"freeformTags",
				"isUsedWithConfigurationSourceProvider",
				"nsgIdList",
				"securityAttributes",
			},
			ForceNew: []string{"compartmentId", "subnetId", "vcnId"},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
