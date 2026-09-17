/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package batchtaskprofile

import (
	batchsdk "github.com/oracle/oci-go-sdk/v65/batch"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerBatchTaskProfileRuntimeHooksMutator(func(_ *BatchTaskProfileServiceManager, hooks *BatchTaskProfileRuntimeHooks) {
		if hooks != nil {
			hooks.Semantics = newBatchTaskProfileRuntimeSemantics()
		}
	})
}

func newBatchTaskProfileRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "batch",
		FormalSlug:    "batchtaskprofile",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ActiveStates: []string{string(batchsdk.BatchTaskProfileLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{string(batchsdk.BatchTaskProfileLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "description", "freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId", "minOcpus", "minMemoryInGBs"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
