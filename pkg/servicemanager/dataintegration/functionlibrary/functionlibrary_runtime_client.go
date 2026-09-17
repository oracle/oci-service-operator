/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package functionlibrary

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerFunctionLibraryRuntimeHooksMutator(func(_ *FunctionLibraryServiceManager, hooks *FunctionLibraryRuntimeHooks) {
		hooks.Semantics = reviewedFunctionLibraryRuntimeSemantics()
	})
}

func reviewedFunctionLibraryRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "functionlibrary",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "none", Runtime: "generatedruntime", FormalClassification: "none"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",

		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"name", "identifier", "registryMetadata", "modelVersion", "description", "categoryName", "objectStatus", "objectVersion"}, ForceNew: []string{"key", "workspaceId", "aggregatorKey"}, ZeroValueNullEquivalent: []string{"name", "identifier", "registryMetadata", "modelVersion", "description", "categoryName", "objectStatus", "objectVersion", "key", "workspaceId", "aggregatorKey"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
