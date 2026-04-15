/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"sort"
	"strings"

	"github.com/oracle/oci-service-operator/internal/formal"
)

const (
	followUpStrategyNone           = "none"
	followUpStrategyReadAfterWrite = "read-after-write"
	followUpStrategyConfirmDelete  = "confirm-delete"
)

func buildRuntimeSemanticsModel(formalModel *FormalModel, runtime *RuntimeModel) *RuntimeSemanticsModel {
	if formalModel == nil {
		return nil
	}
	return buildRuntimeSemanticsModelWithAsync(formalModel, runtime, AsyncConfig{})
}

func buildRuntimeSemanticsModelWithAsync(
	formalModel *FormalModel,
	runtime *RuntimeModel,
	async AsyncConfig,
) *RuntimeSemanticsModel {
	if formalModel == nil {
		return nil
	}

	binding := formalModel.Binding
	mutation := runtimeMutationPaths(formalModel)
	asyncModel := buildRuntimeAsyncModel(async)
	createHooks := effectiveRuntimeHooks(formalModel, "create", binding.Import.Hooks.Create, asyncModel)
	updateHooks := effectiveRuntimeHooks(formalModel, "update", binding.Import.Hooks.Update, asyncModel)
	deleteHooks := effectiveRuntimeHooks(formalModel, "delete", binding.Import.Hooks.Delete, asyncModel)
	semantics := &RuntimeSemanticsModel{
		FormalService:     formalModel.Reference.Service,
		FormalSlug:        formalModel.Reference.Slug,
		Async:             asyncModel,
		StatusProjection:  strings.TrimSpace(binding.Spec.StatusProjection),
		SecretSideEffects: strings.TrimSpace(binding.Spec.SecretSideEffects),
		FinalizerPolicy:   strings.TrimSpace(binding.Spec.FinalizerPolicy),
		Lifecycle: RuntimeLifecycleModel{
			ProvisioningStates: normalizeFormalStates(binding.Import.Lifecycle.Create.Pending),
			UpdatingStates:     normalizeFormalStates(binding.Import.Lifecycle.Update.Pending),
			ActiveStates: normalizeFormalStates(append(
				append([]string(nil), binding.Import.Lifecycle.Create.Target...),
				binding.Import.Lifecycle.Update.Target...,
			)),
		},
		Delete: RuntimeDeleteSemanticsModel{
			Policy:         strings.TrimSpace(binding.Spec.DeleteConfirmation),
			PendingStates:  normalizeFormalStates(binding.Import.DeleteConfirmation.Pending),
			TerminalStates: normalizeFormalStates(binding.Import.DeleteConfirmation.Target),
		},
		Mutation: RuntimeMutationModel{
			Mutable:       normalizeFormalPaths(mutation.Mutable),
			ForceNew:      normalizeFormalPaths(mutation.ForceNew),
			ConflictsWith: normalizeFormalConflicts(binding.Import.Mutation.ConflictsWith),
		},
		Hooks: RuntimeHookSetModel{
			Create: createHooks,
			Update: updateHooks,
			Delete: deleteHooks,
		},
		AuxiliaryOperations: buildAuxiliaryOperationModels(binding, runtime),
		OpenGaps:            buildRuntimeGapModels(binding),
	}
	if binding.Import.ListLookup != nil {
		semantics.List = &RuntimeListLookupModel{
			ResponseItemsField: strings.TrimSpace(binding.Import.ListLookup.ResponseItemsField),
			MatchFields:        normalizeFormalPaths(binding.Import.ListLookup.FilterFields),
		}
	}
	semantics.CreateFollowUp = buildWriteFollowUpModel(
		semantics.Hooks.Create,
		runtime,
		repoAuthoredFollowUpStrategy(formalModel, "create"),
	)
	semantics.UpdateFollowUp = buildWriteFollowUpModel(
		semantics.Hooks.Update,
		runtime,
		repoAuthoredFollowUpStrategy(formalModel, "update"),
	)
	semantics.DeleteFollowUp = buildDeleteFollowUpModel(
		semantics.Hooks.Delete,
		semantics.Delete,
		runtime,
		repoAuthoredFollowUpStrategy(formalModel, "delete"),
	)
	return semantics
}

func buildRuntimeAsyncModel(async AsyncConfig) *RuntimeAsyncModel {
	async = async.withDefaults()
	if !async.hasOverride() {
		return nil
	}

	model := &RuntimeAsyncModel{
		Strategy:             async.Strategy,
		Runtime:              async.Runtime,
		FormalClassification: async.FormalClassification,
	}
	if async.WorkRequest.hasOverride() {
		model.WorkRequest = &RuntimeWorkRequestModel{
			Source: async.WorkRequest.Source,
			Phases: append([]string(nil), async.WorkRequest.Phases...),
		}
		if async.WorkRequest.LegacyFieldBridge.hasOverride() {
			model.WorkRequest.LegacyFieldBridge = &RuntimeLegacyFieldBridgeModel{
				Create: async.WorkRequest.LegacyFieldBridge.Create,
				Update: async.WorkRequest.LegacyFieldBridge.Update,
				Delete: async.WorkRequest.LegacyFieldBridge.Delete,
			}
		}
	}
	return model
}

func filteredRuntimeHooks(hooks []formal.Hook, async *RuntimeAsyncModel) []RuntimeHookModel {
	models := buildRuntimeHookModels(hooks)
	if async == nil || async.Strategy == AsyncStrategyWorkRequest {
		return models
	}

	filtered := make([]RuntimeHookModel, 0, len(models))
	for _, hook := range models {
		if hook.Helper == "tfresource.WaitForWorkRequestWithErrorHandling" {
			continue
		}
		filtered = append(filtered, hook)
	}
	return filtered
}

func effectiveRuntimeHooks(
	formalModel *FormalModel,
	phase string,
	imported []formal.Hook,
	async *RuntimeAsyncModel,
) []RuntimeHookModel {
	hooks := imported
	if repoAuthored, ok := repoAuthoredRuntimeHooks(formalModel, phase); ok {
		hooks = repoAuthored
	}
	return filteredRuntimeHooks(hooks, async)
}

type runtimeMutationPathSet struct {
	Mutable  []string
	ForceNew []string
}

func runtimeMutationPaths(formalModel *FormalModel) runtimeMutationPathSet {
	if formalModel == nil {
		return runtimeMutationPathSet{}
	}
	if formalModel.RuntimeLifecycle != nil &&
		formalModel.RuntimeLifecycle.RepoAuthored != nil &&
		formalModel.RuntimeLifecycle.RepoAuthored.Mutation != nil {
		return runtimeMutationPathSet{
			Mutable:  append([]string(nil), formalModel.RuntimeLifecycle.RepoAuthored.Mutation.Mutable...),
			ForceNew: append([]string(nil), formalModel.RuntimeLifecycle.RepoAuthored.Mutation.ForceNew...),
		}
	}
	return runtimeMutationPathSet{
		Mutable:  append([]string(nil), formalModel.Binding.Import.Mutation.Mutable...),
		ForceNew: append([]string(nil), formalModel.Binding.Import.Mutation.ForceNew...),
	}
}

func buildWriteFollowUpModel(hooks []RuntimeHookModel, runtime *RuntimeModel, override string) RuntimeFollowUpModel {
	followUp := RuntimeFollowUpModel{
		Strategy: followUpStrategyNone,
		Hooks:    append([]RuntimeHookModel(nil), hooks...),
	}
	if override != "" {
		followUp.Strategy = override
		return followUp
	}
	if len(hooks) > 0 && runtimeHasReadOperation(runtime) {
		followUp.Strategy = followUpStrategyReadAfterWrite
	}
	return followUp
}

func buildDeleteFollowUpModel(
	hooks []RuntimeHookModel,
	deleteSemantics RuntimeDeleteSemanticsModel,
	runtime *RuntimeModel,
	override string,
) RuntimeFollowUpModel {
	followUp := RuntimeFollowUpModel{
		Strategy: followUpStrategyNone,
		Hooks:    append([]RuntimeHookModel(nil), hooks...),
	}
	if override != "" {
		followUp.Strategy = override
		return followUp
	}
	if (deleteSemantics.Policy == "required" || deleteSemantics.Policy == "best-effort") && runtimeHasReadOperation(runtime) {
		followUp.Strategy = followUpStrategyConfirmDelete
	}
	return followUp
}

func runtimeHasReadOperation(runtime *RuntimeModel) bool {
	if runtime == nil {
		return false
	}
	return runtime.Get != nil || runtime.List != nil
}

func repoAuthoredRuntimeHooks(formalModel *FormalModel, phase string) ([]formal.Hook, bool) {
	if formalModel == nil || formalModel.RuntimeLifecycle == nil || formalModel.RuntimeLifecycle.RepoAuthored == nil || formalModel.RuntimeLifecycle.RepoAuthored.Hooks == nil {
		return nil, false
	}

	switch phase {
	case "create":
		if formalModel.RuntimeLifecycle.RepoAuthored.Hooks.Create != nil {
			return append([]formal.Hook(nil), formalModel.RuntimeLifecycle.RepoAuthored.Hooks.Create...), true
		}
	case "update":
		if formalModel.RuntimeLifecycle.RepoAuthored.Hooks.Update != nil {
			return append([]formal.Hook(nil), formalModel.RuntimeLifecycle.RepoAuthored.Hooks.Update...), true
		}
	case "delete":
		if formalModel.RuntimeLifecycle.RepoAuthored.Hooks.Delete != nil {
			return append([]formal.Hook(nil), formalModel.RuntimeLifecycle.RepoAuthored.Hooks.Delete...), true
		}
	}

	return nil, false
}

func repoAuthoredFollowUpStrategy(formalModel *FormalModel, phase string) string {
	if formalModel == nil || formalModel.RuntimeLifecycle == nil || formalModel.RuntimeLifecycle.RepoAuthored == nil || formalModel.RuntimeLifecycle.RepoAuthored.FollowUp == nil {
		return ""
	}

	switch phase {
	case "create":
		return strings.TrimSpace(formalModel.RuntimeLifecycle.RepoAuthored.FollowUp.Create)
	case "update":
		return strings.TrimSpace(formalModel.RuntimeLifecycle.RepoAuthored.FollowUp.Update)
	case "delete":
		return strings.TrimSpace(formalModel.RuntimeLifecycle.RepoAuthored.FollowUp.Delete)
	default:
		return ""
	}
}

func buildAuxiliaryOperationModels(binding formal.ControllerBinding, runtime *RuntimeModel) []RuntimeAuxiliaryOperationModel {
	primary := map[string]string{}
	if runtime != nil {
		if runtime.Create != nil {
			primary["create"] = runtime.Create.MethodName
		}
		if runtime.Get != nil {
			primary["get"] = runtime.Get.MethodName
		}
		if runtime.List != nil {
			primary["list"] = runtime.List.MethodName
		}
		if runtime.Update != nil {
			primary["update"] = runtime.Update.MethodName
		}
		if runtime.Delete != nil {
			primary["delete"] = runtime.Delete.MethodName
		}
	}

	var auxiliary []RuntimeAuxiliaryOperationModel
	appendPhase := func(phase string, bindings []formal.OperationBinding) {
		for _, op := range bindings {
			if strings.TrimSpace(op.Operation) == "" || op.Operation == primary[phase] {
				continue
			}
			auxiliary = append(auxiliary, RuntimeAuxiliaryOperationModel{
				Phase:            phase,
				MethodName:       strings.TrimSpace(op.Operation),
				RequestTypeName:  strings.TrimSpace(op.RequestType),
				ResponseTypeName: strings.TrimSpace(op.ResponseType),
			})
		}
	}

	appendPhase("create", binding.Import.Operations.Create)
	appendPhase("get", binding.Import.Operations.Get)
	appendPhase("list", binding.Import.Operations.List)
	appendPhase("update", binding.Import.Operations.Update)
	appendPhase("delete", binding.Import.Operations.Delete)

	sort.Slice(auxiliary, func(i, j int) bool {
		if auxiliary[i].Phase != auxiliary[j].Phase {
			return auxiliary[i].Phase < auxiliary[j].Phase
		}
		return auxiliary[i].MethodName < auxiliary[j].MethodName
	})
	return auxiliary
}

func buildRuntimeGapModels(binding formal.ControllerBinding) []RuntimeGapModel {
	gaps := make([]RuntimeGapModel, 0, len(binding.LogicGaps))
	for _, gap := range binding.LogicGaps {
		if strings.TrimSpace(gap.Status) == "resolved" {
			continue
		}
		gaps = append(gaps, RuntimeGapModel{
			Category:      strings.TrimSpace(gap.Category),
			StopCondition: strings.TrimSpace(gap.StopCondition),
		})
	}
	sort.Slice(gaps, func(i, j int) bool {
		return gaps[i].Category < gaps[j].Category
	})
	return gaps
}

func buildRuntimeHookModels(hooks []formal.Hook) []RuntimeHookModel {
	out := make([]RuntimeHookModel, 0, len(hooks))
	for _, hook := range hooks {
		out = append(out, RuntimeHookModel{
			Helper:     strings.TrimSpace(hook.Helper),
			EntityType: strings.TrimSpace(hook.EntityType),
			Action:     strings.TrimSpace(hook.Action),
		})
	}
	return out
}

func normalizeFormalStates(states []string) []string {
	seen := make(map[string]struct{}, len(states))
	out := make([]string, 0, len(states))
	for _, state := range states {
		state = strings.ToUpper(strings.TrimSpace(state))
		if state == "" {
			continue
		}
		if _, ok := seen[state]; ok {
			continue
		}
		seen[state] = struct{}{}
		out = append(out, state)
	}
	sort.Strings(out)
	return out
}

func normalizeFormalPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = normalizeFormalPath(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func normalizeFormalConflicts(conflicts map[string][]string) map[string][]string {
	if len(conflicts) == 0 {
		return map[string][]string{}
	}

	out := make(map[string][]string, len(conflicts))
	for field, blocked := range conflicts {
		out[normalizeFormalPath(field)] = normalizeFormalPaths(blocked)
	}
	return out
}

func normalizeFormalPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	parts := strings.Split(path, ".")
	for i := range parts {
		parts[i] = snakeToLowerCamel(parts[i])
	}
	return strings.Join(parts, ".")
}

func snakeToLowerCamel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "_") {
		return lowerCamel(value)
	}

	parts := strings.Split(value, "_")
	var builder strings.Builder
	for index, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		if index == 0 {
			builder.WriteString(part)
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		builder.WriteString(part[1:])
	}
	return builder.String()
}
