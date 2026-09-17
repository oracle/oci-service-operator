/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package batchjobpool

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerBatchJobPoolRuntimeHooksMutator(func(_ *BatchJobPoolServiceManager, hooks *BatchJobPoolRuntimeHooks) {
		hooks.Semantics = reviewedBatchJobPoolRuntimeSemantics()
	})
}

// reviewedBatchJobPoolRuntimeSemantics is grounded in the reviewed formal
// lifecycle, the vendored OCI SDK, and the package-local typed mock lifecycle.
// It remains package-owned until the pinned provider documentation can seed a
// generator-owned mutability overlay for this resource.
func reviewedBatchJobPoolRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "batch",
		FormalSlug:        "batchjobpool",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "lifecycleState", "displayName", "id", "batchContextId", "opc-request-id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "description", "freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId", "batchContextId"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
