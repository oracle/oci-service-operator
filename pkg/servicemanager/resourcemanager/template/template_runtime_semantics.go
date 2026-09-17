/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package template

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerTemplateRuntimeHooksMutator(func(_ *TemplateServiceManager, hooks *TemplateRuntimeHooks) {
		hooks.Semantics = reviewedTemplateRuntimeSemantics()
	})
}

// reviewedTemplateRuntimeSemantics is grounded in the recorded Template
// lifecycle and the vendored OCI SDK. The pinned Terraform provider does not
// expose a Template resource, so this contract intentionally does not claim
// provider resource provenance.
func reviewedTemplateRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"definedTags",
				"description",
				"displayName",
				"freeformTags",
				"logoFileBase64Encoded",
				"longDescription",
				"templateConfigSource",
			},
			ForceNew:      []string{"compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
