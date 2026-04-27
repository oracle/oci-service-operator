/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerinstance

import (
	containerinstancessdk "github.com/oracle/oci-go-sdk/v65/containerinstances"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func containerInstanceRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "containerinstances",
		FormalSlug:        "containerinstance",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "handwritten",
			FormalClassification: "lifecycle",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(containerinstancessdk.ContainerInstanceLifecycleStateCreating)},
			UpdatingStates:     []string{string(containerinstancessdk.ContainerInstanceLifecycleStateUpdating)},
			ActiveStates: []string{
				string(containerinstancessdk.ContainerInstanceLifecycleStateActive),
				string(containerinstancessdk.ContainerInstanceLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(containerinstancessdk.ContainerInstanceLifecycleStateDeleting)},
			TerminalStates: []string{string(containerinstancessdk.ContainerInstanceLifecycleStateDeleted), "NOT_FOUND"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"availabilityDomain", "compartmentId", "displayName", "lifecycleState"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{"definedTags", "displayName", "freeformTags"},
			ForceNew: []string{
				"availabilityDomain",
				"compartmentId",
				"containerRestartPolicy",
				"containers",
				"dnsConfig",
				"faultDomain",
				"gracefulShutdownTimeoutInSeconds",
				"imagePullSecrets",
				"shape",
				"shapeConfig",
				"vnics",
				"volumes",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		Unsupported: []generatedruntime.UnsupportedSemantic{
			{
				Category:      "create-only-drift",
				StopCondition: "container image, VNIC subnet, and BASIC image pull secret credential drift cannot be fully compared because GetContainerInstance redacts or omits those create payload fields",
			},
		},
	}
}

func reusableContainerInstanceLifecycleStates() []containerinstancessdk.ContainerInstanceLifecycleStateEnum {
	return []containerinstancessdk.ContainerInstanceLifecycleStateEnum{
		containerinstancessdk.ContainerInstanceLifecycleStateActive,
		containerinstancessdk.ContainerInstanceLifecycleStateCreating,
		containerinstancessdk.ContainerInstanceLifecycleStateUpdating,
		containerinstancessdk.ContainerInstanceLifecycleStateInactive,
	}
}
