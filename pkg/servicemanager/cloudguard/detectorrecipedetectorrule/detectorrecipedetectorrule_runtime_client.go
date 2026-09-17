/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package detectorrecipedetectorrule

import (
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func init() {
	registerDetectorRecipeDetectorRuleRuntimeHooksMutator(func(_ *DetectorRecipeDetectorRuleServiceManager, hooks *DetectorRecipeDetectorRuleRuntimeHooks) {
		hooks.Semantics = newDetectorRecipeDetectorRuleRuntimeSemantics()
		hooks.StatusHooks.ProjectStatus = projectDetectorRecipeDetectorRuleStatus
	})
}

func newDetectorRecipeDetectorRuleRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "cloudguard", FormalSlug: "detectorrecipedetectorrule",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"detectorRecipeId", "details.name"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"details"}, ForceNew: []string{"detectorRecipeId"}, ConflictsWith: map[string][]string{}},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func projectDetectorRecipeDetectorRuleStatus(resource *cloudguardv1beta1.DetectorRecipeDetectorRule, response any) error {
	if err := generatedruntime.ProjectResponseBodyWithAliases(resource, response, nil); err != nil {
		return err
	}
	if resource.Status.DetectorRuleId != "" {
		resource.Status.Id = resource.Status.DetectorRuleId
		resource.Status.OsokStatus.Ocid = shared.OCID(resource.Status.DetectorRuleId)
	}
	return nil
}
