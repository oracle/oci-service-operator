/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package project

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerProjectRuntimeHooksMutator(func(_ *ProjectServiceManager, hooks *ProjectRuntimeHooks) {
		hooks.Semantics = reviewedProjectRuntimeSemantics()
	})
}

func reviewedProjectRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "project",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "none", Runtime: "generatedruntime", FormalClassification: "none"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",

		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"name", "identifier", "modelVersion", "description", "objectStatus", "key", "registryMetadata", "modelType", "objectVersion", "parentRef"}, ForceNew: []string{"workspaceId"}, ZeroValueNullEquivalent: []string{"name", "identifier", "modelVersion", "description", "objectStatus", "key", "registryMetadata", "modelType", "objectVersion", "parentRef", "workspaceId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
