/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package schedule

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerScheduleRuntimeHooksMutator(func(_ *ScheduleServiceManager, hooks *ScheduleRuntimeHooks) {
		hooks.Semantics = reviewedScheduleRuntimeSemantics()
	})
}

func reviewedScheduleRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "schedule",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "none", Runtime: "generatedruntime", FormalClassification: "none"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",

		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"name", "identifier", "key", "modelVersion", "description", "objectVersion", "objectStatus", "frequencyDetails", "timezone", "isDaylightAdjustmentEnabled", "registryMetadata", "modelType", "parentRef"}, ForceNew: []string{"workspaceId", "applicationKey"}, ZeroValueNullEquivalent: []string{"name", "identifier", "key", "modelVersion", "description", "objectVersion", "objectStatus", "frequencyDetails", "timezone", "isDaylightAdjustmentEnabled", "registryMetadata", "modelType", "parentRef", "workspaceId", "applicationKey"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
