/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package folder

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerFolderRuntimeHooksMutator(func(_ *FolderServiceManager, hooks *FolderRuntimeHooks) {
		hooks.Semantics = reviewedFolderRuntimeSemantics()
	})
}

func reviewedFolderRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "folder",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "none", Runtime: "generatedruntime", FormalClassification: "none"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",

		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"name", "identifier", "registryMetadata", "key", "modelVersion", "description", "categoryName", "objectStatus", "modelType", "objectVersion", "parentRef"}, ForceNew: []string{"workspaceId", "aggregatorKey"}, ZeroValueNullEquivalent: []string{"name", "identifier", "registryMetadata", "key", "modelVersion", "description", "categoryName", "objectStatus", "modelType", "objectVersion", "parentRef", "workspaceId", "aggregatorKey"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
