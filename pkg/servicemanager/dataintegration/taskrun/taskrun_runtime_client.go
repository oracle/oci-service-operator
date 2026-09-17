/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package taskrun

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerTaskRunRuntimeHooksMutator(func(_ *TaskRunServiceManager, hooks *TaskRunRuntimeHooks) {
		hooks.Semantics = reviewedTaskRunRuntimeSemantics()
	})
}

func reviewedTaskRunRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "taskrun",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"IN_PROGRESS", "QUEUED", "PUBLISHING", "NOT_STARTED", "RUNNING"}, ActiveStates: []string{"SUCCESS", "SUCCESSFUL", "TERMINATED"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"key", "modelType", "modelVersion", "name", "description", "taskScheduleKey", "registryMetadata", "status", "objectVersion"}, ForceNew: []string{"configProvider", "identifier", "refTaskRunId", "reRunType", "stepId", "workspaceId", "applicationKey", "aggregatorKey"}, ZeroValueNullEquivalent: []string{"key", "modelType", "modelVersion", "name", "description", "taskScheduleKey", "registryMetadata", "status", "objectVersion", "configProvider", "identifier", "refTaskRunId", "reRunType", "stepId", "workspaceId", "applicationKey", "aggregatorKey"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
