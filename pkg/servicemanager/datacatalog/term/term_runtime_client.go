/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package term

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerTermRuntimeHooksMutator(func(_ *TermServiceManager, hooks *TermRuntimeHooks) {
		hooks.Semantics = reviewedTermRuntimeSemantics()
	})
}

func reviewedTermRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "datacatalog", FormalSlug: "term",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE", "INACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"displayName"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"displayName", "description", "parentTermKey", "owner", "workflowStatus", "customPropertyMembers"}, ForceNew: []string{"isAllowedToHaveChildTerms", "catalogId", "glossaryKey"}, ZeroValueNullEquivalent: []string{"displayName", "description", "parentTermKey", "owner", "workflowStatus", "customPropertyMembers", "isAllowedToHaveChildTerms", "catalogId", "glossaryKey"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
