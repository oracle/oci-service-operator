/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package stack

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerStackRuntimeHooksMutator(func(_ *StackServiceManager, hooks *StackRuntimeHooks) {
		hooks.Semantics = reviewedStackRuntimeSemantics()
	})
}

// reviewedStackRuntimeSemantics is grounded in the recorded Stack lifecycle
// and the vendored OCI SDK. The pinned Terraform provider exposes Stack only
// as data sources, so this contract intentionally does not claim provider
// resource provenance.
func reviewedStackRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"configSource",
				"customTerraformProvider",
				"definedTags",
				"description",
				"displayName",
				"freeformTags",
				"terraformVersion",
				"variables",
			},
			ForceNew:      []string{"compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
