/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package application

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerApplicationRuntimeHooksMutator(func(_ *ApplicationServiceManager, hooks *ApplicationRuntimeHooks) {
		hooks.Semantics = reviewedApplicationRuntimeSemantics()
	})
}

func reviewedApplicationRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "application",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE", "INACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"name", "identifier", "key", "modelVersion", "modelType", "description", "objectStatus", "displayName", "freeformTags", "definedTags", "lifecycleState", "objectVersion", "applicationVersion", "parentRef", "metadata"}, ForceNew: []string{"sourceApplicationInfo", "registryMetadata", "workspaceId"}, ZeroValueNullEquivalent: []string{"name", "identifier", "key", "modelVersion", "modelType", "description", "objectStatus", "displayName", "freeformTags", "definedTags", "lifecycleState", "objectVersion", "applicationVersion", "parentRef", "metadata", "sourceApplicationInfo", "registryMetadata", "workspaceId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
