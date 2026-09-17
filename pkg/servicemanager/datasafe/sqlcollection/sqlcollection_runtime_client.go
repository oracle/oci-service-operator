/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sqlcollection

import (
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerSqlCollectionRuntimeHooksMutator(func(_ *SqlCollectionServiceManager, hooks *SqlCollectionRuntimeHooks) {
		hooks.Semantics = newSqlCollectionRuntimeSemantics()
		hooks.StatusHooks.ProjectStatus = func(resource *datasafev1beta1.SqlCollection, response any) error {
			return generatedruntime.ProjectResponseBodyWithAliases(resource, response, map[string]string{"status": "sdkStatus"})
		}
	})
}

func newSqlCollectionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:    "datasafe",
		FormalSlug:       "sqlcollection",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", FinalizerPolicy: "retain-until-confirmed-delete", SecretSideEffects: "none",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"}, UpdatingStates: []string{"UPDATING"}, ActiveStates: []string{"COLLECTING", "COMPLETED", "INACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"}},
		List:   &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"compartmentId", "displayName", "targetId", "dbUserName"}},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:  []string{"displayName", "description", "status", "sqlLevel", "freeformTags", "definedTags"},
			ForceNew: []string{"compartmentId", "targetId", "dbUserName"}, ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
