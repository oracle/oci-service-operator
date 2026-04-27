/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package esxihost

import (
	"context"
	"fmt"
	"strings"

	ocvpv1beta1 "github.com/oracle/oci-service-operator/api/ocvp/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerEsxiHostRuntimeHooksMutator(func(_ *EsxiHostServiceManager, hooks *EsxiHostRuntimeHooks) {
		applyEsxiHostRuntimeHooks(hooks)
	})
}

func applyEsxiHostRuntimeHooks(hooks *EsxiHostRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newEsxiHostRuntimeSemantics()
	hooks.Identity.GuardExistingBeforeCreate = guardEsxiHostExistingBeforeCreate
}

func newEsxiHostRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "ocvp",
		FormalSlug:    "esxihost",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"clusterId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"billingDonorHostId",
				"definedTags",
				"displayName",
				"freeformTags",
				"nextCommitment",
			},
			ForceNew: []string{
				"capacityReservationId",
				"clusterId",
				"computeAvailabilityDomain",
				"currentCommitment",
				"esxiSoftwareVersion",
				"hostOcpuCount",
				"hostShapeName",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "", Action: ""}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "", Action: ""}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "", Action: ""}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "", Action: ""}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "", Action: ""}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "", Action: ""}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func guardEsxiHostExistingBeforeCreate(
	_ context.Context,
	resource *ocvpv1beta1.EsxiHost,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if esxiHostIdentityResolutionRequiresDisplayName(resource) {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("EsxiHost spec.displayName is required when no OCI identifier is recorded")
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func esxiHostIdentityResolutionRequiresDisplayName(resource *ocvpv1beta1.EsxiHost) bool {
	if resource == nil {
		return false
	}
	if strings.TrimSpace(resource.Spec.DisplayName) != "" {
		return false
	}
	return currentEsxiHostID(resource) == ""
}

func currentEsxiHostID(resource *ocvpv1beta1.EsxiHost) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}
