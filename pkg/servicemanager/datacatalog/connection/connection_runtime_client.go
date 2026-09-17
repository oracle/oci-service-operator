/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package connection

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerConnectionRuntimeHooksMutator(func(_ *ConnectionServiceManager, hooks *ConnectionRuntimeHooks) {
		hooks.Semantics = reviewedConnectionRuntimeSemantics()
	})
}

func reviewedConnectionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "datacatalog", FormalSlug: "connection",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE", "INACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"displayName"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"displayName", "properties", "description", "customPropertyMembers", "encProperties", "isDefault"}, ForceNew: []string{"typeKey", "catalogId", "dataAssetKey"}, ZeroValueNullEquivalent: []string{"displayName", "properties", "description", "customPropertyMembers", "encProperties", "isDefault", "typeKey", "catalogId", "dataAssetKey"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
