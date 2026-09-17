/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package copyobjectrequest

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerCopyObjectRequestRuntimeHooksMutator(func(_ *CopyObjectRequestServiceManager, hooks *CopyObjectRequestRuntimeHooks) {
		hooks.Semantics = reviewedCopyObjectRequestRuntimeSemantics()
	})
}

func reviewedCopyObjectRequestRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "copyobjectrequest",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"IN_PROGRESS", "QUEUED"}, ActiveStates: []string{"SUCCESSFUL", "TERMINATED"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"sourceWorkspaceId"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"status"}, ForceNew: []string{"sourceWorkspaceId", "objectKeys", "copyConflictResolution", "workspaceId"}, ZeroValueNullEquivalent: []string{"status", "sourceWorkspaceId", "objectKeys", "copyConflictResolution", "workspaceId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
