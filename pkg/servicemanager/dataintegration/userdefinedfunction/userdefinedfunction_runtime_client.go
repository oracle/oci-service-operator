/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package userdefinedfunction

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerUserDefinedFunctionRuntimeHooksMutator(func(_ *UserDefinedFunctionServiceManager, hooks *UserDefinedFunctionRuntimeHooks) {
		hooks.Semantics = reviewedUserDefinedFunctionRuntimeSemantics()
	})
}

func reviewedUserDefinedFunctionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "userdefinedfunction",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "none", Runtime: "generatedruntime", FormalClassification: "none"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",

		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"name", "identifier", "registryMetadata", "modelVersion", "parentRef", "signatures", "expr", "description", "objectStatus", "objectVersion"}, ForceNew: []string{"key", "workspaceId"}, ZeroValueNullEquivalent: []string{"name", "identifier", "registryMetadata", "modelVersion", "parentRef", "signatures", "expr", "description", "objectStatus", "objectVersion", "key", "workspaceId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
