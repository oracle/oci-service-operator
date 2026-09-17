/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package customproperty

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerCustomPropertyRuntimeHooksMutator(func(_ *CustomPropertyServiceManager, hooks *CustomPropertyRuntimeHooks) {
		hooks.Semantics = reviewedCustomPropertyRuntimeSemantics()
	})
}

func reviewedCustomPropertyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "datacatalog", FormalSlug: "customproperty",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE", "INACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"displayName"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"displayName", "description", "isSortable", "isFilterable", "isMultiValued", "isHidden", "isEditable", "isShownInList", "isHiddenInSearch", "isEventEnabled", "allowedValues", "properties"}, ForceNew: []string{"dataType", "catalogId", "namespaceId"}, ZeroValueNullEquivalent: []string{"displayName", "description", "isSortable", "isFilterable", "isMultiValued", "isHidden", "isEditable", "isShownInList", "isHiddenInSearch", "isEventEnabled", "allowedValues", "properties", "dataType", "catalogId", "namespaceId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
