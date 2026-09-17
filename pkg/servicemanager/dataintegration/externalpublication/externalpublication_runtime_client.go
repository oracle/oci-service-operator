/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package externalpublication

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerExternalPublicationRuntimeHooksMutator(func(_ *ExternalPublicationServiceManager, hooks *ExternalPublicationRuntimeHooks) {
		hooks.Semantics = reviewedExternalPublicationRuntimeSemantics()
	})
}

func reviewedExternalPublicationRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "externalpublication",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"IN_PROGRESS", "QUEUED", "PUBLISHING", "NOT_STARTED", "RUNNING"}, ActiveStates: []string{"SUCCESS", "SUCCESSFUL", "TERMINATED"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"displayName"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"applicationCompartmentId", "displayName", "applicationId", "description", "resourceConfiguration", "configurationDetails"}, ForceNew: []string{"workspaceId", "taskKey"}, ZeroValueNullEquivalent: []string{"applicationCompartmentId", "displayName", "applicationId", "description", "resourceConfiguration", "configurationDetails", "workspaceId", "taskKey"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
