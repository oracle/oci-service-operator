/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package targetdatabasegroup

import (
	"fmt"

	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/datasafe/runtimecommon"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerTargetDatabaseGroupRuntimeHooksMutator(func(manager *TargetDatabaseGroupServiceManager, hooks *TargetDatabaseGroupRuntimeHooks) {
		client, initErr := newTargetDatabaseGroupWorkRequestClient(manager)
		applyTargetDatabaseGroupRuntimeHooks(hooks, client, initErr)
	})
}

func newTargetDatabaseGroupWorkRequestClient(manager *TargetDatabaseGroupServiceManager) (runtimecommon.WorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("TargetDatabaseGroup service manager is nil")
	}
	client, err := datasafesdk.NewDataSafeClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyTargetDatabaseGroupRuntimeHooks(hooks *TargetDatabaseGroupRuntimeHooks, client runtimecommon.WorkRequestClient, initErr error) {
	if hooks == nil {
		return
	}
	hooks.Semantics = reviewedTargetDatabaseGroupRuntimeSemantics()
	runtimecommon.ConfigureWorkRequest(&hooks.Async, client, initErr, "TargetDatabaseGroup")
}

func reviewedTargetDatabaseGroupRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "datasafe",
		FormalSlug:    "targetdatabasegroup",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.TargetDatabaseGroupLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.TargetDatabaseGroupLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.TargetDatabaseGroupLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.TargetDatabaseGroupLifecycleStateDeleting)},
			TerminalStates: []string{string(datasafesdk.TargetDatabaseGroupLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "description", "matchingCriteria", "freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> GetTargetDatabaseGroup"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> GetTargetDatabaseGroup"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> confirm-delete"},
	}
}
