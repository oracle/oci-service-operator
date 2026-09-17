/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package entity

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerEntityRuntimeHooksMutator(func(_ *EntityServiceManager, hooks *EntityRuntimeHooks) {
		hooks.Semantics = reviewedEntityRuntimeSemantics()
	})
}

func reviewedEntityRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "datacatalog", FormalSlug: "entity",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE", "INACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"displayName"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"displayName", "timeExternal", "businessName", "description", "isLogical", "isPartition", "folderKey", "patternKey", "realizedExpression", "harvestStatus", "lastJobKey", "customPropertyMembers", "properties"}, ForceNew: []string{"typeKey", "catalogId", "dataAssetKey"}, ZeroValueNullEquivalent: []string{"displayName", "timeExternal", "businessName", "description", "isLogical", "isPartition", "folderKey", "patternKey", "realizedExpression", "harvestStatus", "lastJobKey", "customPropertyMembers", "properties", "typeKey", "catalogId", "dataAssetKey"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
