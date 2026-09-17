/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package job

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerJobRuntimeHooksMutator(func(_ *JobServiceManager, hooks *JobRuntimeHooks) {
		hooks.Semantics = reviewedJobRuntimeSemantics()
	})
}

func reviewedJobRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "datacatalog", FormalSlug: "job",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE", "INACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"displayName"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"displayName", "description", "scheduleCronExpression", "timeScheduleBegin", "timeScheduleEnd", "connectionKey"}, ForceNew: []string{"jobDefinitionKey", "catalogId"}, ZeroValueNullEquivalent: []string{"displayName", "description", "scheduleCronExpression", "timeScheduleBegin", "timeScheduleEnd", "connectionKey", "jobDefinitionKey", "catalogId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
