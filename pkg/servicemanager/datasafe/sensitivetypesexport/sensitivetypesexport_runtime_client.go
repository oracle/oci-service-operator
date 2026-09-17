/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sensitivetypesexport

import (
	"fmt"

	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/datasafe/runtimecommon"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerSensitiveTypesExportRuntimeHooksMutator(func(manager *SensitiveTypesExportServiceManager, hooks *SensitiveTypesExportRuntimeHooks) {
		client, initErr := newSensitiveTypesExportWorkRequestClient(manager)
		applySensitiveTypesExportRuntimeHooks(hooks, client, initErr)
	})
}

func newSensitiveTypesExportWorkRequestClient(manager *SensitiveTypesExportServiceManager) (runtimecommon.WorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("SensitiveTypesExport service manager is nil")
	}
	client, err := datasafesdk.NewDataSafeClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applySensitiveTypesExportRuntimeHooks(hooks *SensitiveTypesExportRuntimeHooks, client runtimecommon.WorkRequestClient, initErr error) {
	if hooks == nil {
		return
	}
	hooks.Semantics = reviewedSensitiveTypesExportRuntimeSemantics()
	runtimecommon.ConfigureWorkRequest(&hooks.Async, client, initErr, "SensitiveTypesExport")
}

func reviewedSensitiveTypesExportRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "datasafe",
		FormalSlug:    "sensitivetypesexport",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update"},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.SensitiveTypesExportLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.SensitiveTypesExportLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.SensitiveTypesExportLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.SensitiveTypesExportLifecycleStateDeleting)},
			TerminalStates: []string{string(datasafesdk.SensitiveTypesExportLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "description", "freeformTags", "definedTags"},
			ForceNew:      []string{"compartmentId", "sensitiveTypeIdsForExport", "isIncludeAllSensitiveTypes"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> GetSensitiveTypesExport"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> GetSensitiveTypesExport"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
