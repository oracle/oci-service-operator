/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package dataasset

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerDataAssetRuntimeHooksMutator(func(_ *DataAssetServiceManager, hooks *DataAssetRuntimeHooks) {
		hooks.Semantics = reviewedDataAssetRuntimeSemantics()
	})
}

func reviewedDataAssetRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "datacatalog", FormalSlug: "dataasset",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE", "INACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"displayName"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"displayName", "description", "customPropertyMembers", "properties"}, ForceNew: []string{"typeKey", "catalogId"}, ZeroValueNullEquivalent: []string{"displayName", "description", "customPropertyMembers", "properties", "typeKey", "catalogId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
