/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package unifiedauditpolicy

import (
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerUnifiedAuditPolicyRuntimeHooksMutator(func(_ *UnifiedAuditPolicyServiceManager, hooks *UnifiedAuditPolicyRuntimeHooks) {
		hooks.Semantics = newUnifiedAuditPolicyRuntimeSemantics()
		hooks.StatusHooks.ProjectStatus = func(resource *datasafev1beta1.UnifiedAuditPolicy, response any) error {
			return generatedruntime.ProjectResponseBodyWithAliases(resource, response, map[string]string{"status": "sdkStatus"})
		}
	})
}

func newUnifiedAuditPolicyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:    "datasafe",
		FormalSlug:       "unifiedauditpolicy",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", FinalizerPolicy: "retain-until-confirmed-delete", SecretSideEffects: "none",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE", "INACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}},
		List:   &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"compartmentId", "securityPolicyId", "unifiedAuditPolicyDefinitionId"}},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:  []string{"displayName", "description", "status", "conditions", "freeformTags", "definedTags"},
			ForceNew: []string{"compartmentId", "securityPolicyId", "unifiedAuditPolicyDefinitionId"}, ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
