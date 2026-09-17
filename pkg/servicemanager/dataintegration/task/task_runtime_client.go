/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package task

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerTaskRuntimeHooksMutator(func(_ *TaskServiceManager, hooks *TaskRuntimeHooks) {
		hooks.Semantics = reviewedTaskRuntimeSemantics()
	})
}

func reviewedTaskRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "task",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "none", Runtime: "generatedruntime", FormalClassification: "none"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",

		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"jsonData", "key", "modelVersion", "parentRef", "description", "objectStatus", "inputPorts", "outputPorts", "parameters", "opConfigValues", "configProviderDelegate", "isConcurrentAllowed", "name", "identifier", "registryMetadata", "dataFlow", "conditionalCompositeFieldMap", "isSingleLoad", "parallelLoadLimit", "pipeline", "dataflowApplication", "driverShapeDetails", "executorShapeDetails", "script", "operation", "sqlScriptType", "authDetails", "authConfig", "endpoint", "headers", "cancelEndpoint", "executeRestCallConfig", "cancelRestCallConfig", "pollRestCallConfig", "typedExpressions", "methodType", "apiCallMode", "cancelMethodType", "objectVersion", "additionalProperties"}, ForceNew: []string{"modelType", "workspaceId"}, ZeroValueNullEquivalent: []string{"jsonData", "key", "modelVersion", "parentRef", "description", "objectStatus", "inputPorts", "outputPorts", "parameters", "opConfigValues", "configProviderDelegate", "isConcurrentAllowed", "name", "identifier", "registryMetadata", "dataFlow", "conditionalCompositeFieldMap", "isSingleLoad", "parallelLoadLimit", "pipeline", "dataflowApplication", "driverShapeDetails", "executorShapeDetails", "script", "operation", "sqlScriptType", "authDetails", "authConfig", "endpoint", "headers", "cancelEndpoint", "executeRestCallConfig", "cancelRestCallConfig", "pollRestCallConfig", "typedExpressions", "methodType", "apiCallMode", "cancelMethodType", "objectVersion", "additionalProperties", "modelType", "workspaceId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
