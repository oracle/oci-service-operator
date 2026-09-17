/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package applicationdetaileddescription

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerApplicationDetailedDescriptionRuntimeHooksMutator(func(_ *ApplicationDetailedDescriptionServiceManager, hooks *ApplicationDetailedDescriptionRuntimeHooks) {
		hooks.Semantics = reviewedApplicationDetailedDescriptionRuntimeSemantics()
	})
}

func reviewedApplicationDetailedDescriptionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "applicationdetaileddescription",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "none", Runtime: "generatedruntime", FormalClassification: "none"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",

		Delete: generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},

		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"logo", "detailedDescription"}, ForceNew: []string{"workspaceId", "applicationKey"}, ZeroValueNullEquivalent: []string{"logo", "detailedDescription", "workspaceId", "applicationKey"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
