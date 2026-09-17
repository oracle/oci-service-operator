/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package targetdetectorrecipe

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerTargetDetectorRecipeRuntimeHooksMutator(func(_ *TargetDetectorRecipeServiceManager, hooks *TargetDetectorRecipeRuntimeHooks) {
		hooks.Semantics = newTargetDetectorRecipeRuntimeSemantics()
	})
}

func newTargetDetectorRecipeRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "cloudguard", FormalSlug: "targetdetectorrecipe",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"targetId", "compartmentId", "detectorRecipeId"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"detectorRules", "isValidationOnlyQuery"}, ForceNew: []string{"targetId", "compartmentId", "detectorRecipeId"}, ConflictsWith: map[string][]string{}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
