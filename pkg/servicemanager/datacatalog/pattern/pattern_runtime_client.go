/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package pattern

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerPatternRuntimeHooksMutator(func(_ *PatternServiceManager, hooks *PatternRuntimeHooks) {
		hooks.Semantics = reviewedPatternRuntimeSemantics()
	})
}

func reviewedPatternRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "datacatalog", FormalSlug: "pattern",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE", "INACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"displayName"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"displayName", "description", "expression", "filePathPrefix", "checkFilePathList", "isEnableCheckFailureLimit", "checkFailureLimit", "properties"}, ForceNew: []string{"catalogId"}, ZeroValueNullEquivalent: []string{"displayName", "description", "expression", "filePathPrefix", "checkFilePathList", "isEnableCheckFailureLimit", "checkFailureLimit", "properties", "catalogId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
