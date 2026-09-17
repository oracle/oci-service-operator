/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package attribute

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerAttributeRuntimeHooksMutator(func(_ *AttributeServiceManager, hooks *AttributeRuntimeHooks) {
		hooks.Semantics = reviewedAttributeRuntimeSemantics()
	})
}

func reviewedAttributeRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "datacatalog", FormalSlug: "attribute",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE", "INACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"displayName"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"displayName", "externalDataType", "timeExternal", "businessName", "description", "isIncrementalData", "isNullable", "length", "position", "precision", "scale", "minCollectionCount", "maxCollectionCount", "externalDatatypeEntityKey", "externalParentAttributeKey", "customPropertyMembers", "properties"}, ForceNew: []string{"typeKey", "catalogId", "dataAssetKey", "entityKey"}, ZeroValueNullEquivalent: []string{"displayName", "externalDataType", "timeExternal", "businessName", "description", "isIncrementalData", "isNullable", "length", "position", "precision", "scale", "minCollectionCount", "maxCollectionCount", "externalDatatypeEntityKey", "externalParentAttributeKey", "customPropertyMembers", "properties", "typeKey", "catalogId", "dataAssetKey", "entityKey"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
