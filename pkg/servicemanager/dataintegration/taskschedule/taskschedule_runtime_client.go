/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package taskschedule

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerTaskScheduleRuntimeHooksMutator(func(_ *TaskScheduleServiceManager, hooks *TaskScheduleRuntimeHooks) {
		hooks.Semantics = reviewedTaskScheduleRuntimeSemantics()
	})
}

func reviewedTaskScheduleRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "taskschedule",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "none", Runtime: "generatedruntime", FormalClassification: "none"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",

		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"name", "identifier", "key", "modelVersion", "parentRef", "description", "objectVersion", "objectStatus", "scheduleRef", "configProviderDelegate", "isEnabled", "numberOfRetries", "retryDelay", "retryDelayUnit", "startTimeMillis", "endTimeMillis", "isConcurrentAllowed", "isBackfillEnabled", "authMode", "expectedDuration", "expectedDurationUnit", "registryMetadata", "modelType"}, ForceNew: []string{"workspaceId", "applicationKey"}, ZeroValueNullEquivalent: []string{"name", "identifier", "key", "modelVersion", "parentRef", "description", "objectVersion", "objectStatus", "scheduleRef", "configProviderDelegate", "isEnabled", "numberOfRetries", "retryDelay", "retryDelayUnit", "startTimeMillis", "endTimeMillis", "isConcurrentAllowed", "isBackfillEnabled", "authMode", "expectedDuration", "expectedDurationUnit", "registryMetadata", "modelType", "workspaceId", "applicationKey"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
