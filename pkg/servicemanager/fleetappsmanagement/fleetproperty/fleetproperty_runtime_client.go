/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package fleetproperty

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerFleetPropertyRuntimeHooksMutator(func(_ *FleetPropertyServiceManager, hooks *FleetPropertyRuntimeHooks) {
		hooks.Semantics = reviewedFleetPropertyRuntimeSemantics()
	})
}

func reviewedFleetPropertyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "fleetappsmanagement", FormalSlug: "fleetproperty",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ActiveStates: []string{"ACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"value"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"value"}, ForceNew: []string{"propertyId", "fleetId"}, ZeroValueNullEquivalent: []string{"value", "propertyId", "fleetId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
