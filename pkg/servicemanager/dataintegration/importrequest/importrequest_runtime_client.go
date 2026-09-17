/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package importrequest

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerImportRequestRuntimeHooksMutator(func(_ *ImportRequestServiceManager, hooks *ImportRequestRuntimeHooks) {
		hooks.Semantics = reviewedImportRequestRuntimeSemantics()
	})
}

func reviewedImportRequestRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "importrequest",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle:      generatedruntime.LifecycleSemantics{ProvisioningStates: []string{"IN_PROGRESS", "QUEUED", "PUBLISHING", "NOT_STARTED", "RUNNING"}, ActiveStates: []string{"SUCCESS", "SUCCESSFUL", "TERMINATED"}},
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"bucketName"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"status"}, ForceNew: []string{"bucketName", "fileName", "objectStorageTenancyId", "objectStorageRegion", "objectKeyForImport", "areDataAssetReferencesIncluded", "importConflictResolution", "workspaceId"}, ZeroValueNullEquivalent: []string{"status", "bucketName", "fileName", "objectStorageTenancyId", "objectStorageRegion", "objectKeyForImport", "areDataAssetReferencesIncluded", "importConflictResolution", "workspaceId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
