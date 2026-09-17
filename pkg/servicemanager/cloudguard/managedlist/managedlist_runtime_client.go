/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package managedlist

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerManagedListRuntimeHooksMutator(func(_ *ManagedListServiceManager, hooks *ManagedListRuntimeHooks) {
		if hooks != nil {
			hooks.Semantics = managedListRuntimeSemantics()
		}
	})
}

func managedListRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "cloudguard", FormalSlug: "managedlist",
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Async:     &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		Lifecycle: generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"ACTIVE"}},
		Delete:    generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}},
		List:      &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"compartmentId", "displayName", "id"}},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:  []string{"definedTags", "description", "displayName", "freeformTags", "listItems"},
			ForceNew: []string{"compartmentId", "group", "listType", "sourceManagedListId"}, ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
