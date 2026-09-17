/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package disapplication

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerDisApplicationRuntimeHooksMutator(func(_ *DisApplicationServiceManager, hooks *DisApplicationRuntimeHooks) {
		hooks.Semantics = reviewedDisApplicationRuntimeSemantics()
	})
}

func reviewedDisApplicationRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "disapplication",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE", "INACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"compartmentId", "name"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"name", "identifier", "key", "modelVersion", "modelType", "description", "objectStatus", "displayName", "freeformTags", "definedTags", "lifecycleState", "objectVersion", "applicationVersion", "parentRef", "metadata"}, ForceNew: []string{"compartmentId", "sourceApplicationInfo", "registryMetadata", "workspaceId"}, ZeroValueNullEquivalent: []string{"name", "identifier", "key", "modelVersion", "modelType", "description", "objectStatus", "displayName", "freeformTags", "definedTags", "lifecycleState", "objectVersion", "applicationVersion", "parentRef", "metadata", "compartmentId", "sourceApplicationInfo", "registryMetadata", "workspaceId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
