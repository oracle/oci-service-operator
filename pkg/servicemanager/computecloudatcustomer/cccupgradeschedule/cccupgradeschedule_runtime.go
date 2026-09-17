/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package cccupgradeschedule

import (
	computecloudatcustomersdk "github.com/oracle/oci-go-sdk/v65/computecloudatcustomer"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerCccUpgradeScheduleRuntimeHooksMutator(func(_ *CccUpgradeScheduleServiceManager, hooks *CccUpgradeScheduleRuntimeHooks) {
		if hooks != nil {
			hooks.Semantics = reviewedCccUpgradeScheduleRuntimeSemantics()
		}
	})
}

func reviewedCccUpgradeScheduleRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(computecloudatcustomersdk.CccUpgradeScheduleLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(computecloudatcustomersdk.CccUpgradeScheduleLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName", "id"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:  []string{"displayName", "description", "events", "freeformTags", "definedTags"},
			ForceNew: []string{"compartmentId"},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
