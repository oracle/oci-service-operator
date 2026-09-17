/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package dataflow

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerDataFlowRuntimeHooksMutator(func(_ *DataFlowServiceManager, hooks *DataFlowRuntimeHooks) {
		hooks.Semantics = reviewedDataFlowRuntimeSemantics()
	})
}

func reviewedDataFlowRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "dataflow",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "none", Runtime: "generatedruntime", FormalClassification: "none"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",

		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"name", "identifier", "registryMetadata", "key", "modelVersion", "parentRef", "nodes", "parameters", "description", "flowConfigValues", "objectStatus", "modelType", "objectVersion"}, ForceNew: []string{"workspaceId"}, ZeroValueNullEquivalent: []string{"name", "identifier", "registryMetadata", "key", "modelVersion", "parentRef", "nodes", "parameters", "description", "flowConfigValues", "objectStatus", "modelType", "objectVersion", "workspaceId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
