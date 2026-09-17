/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package dataasset

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerDataAssetRuntimeHooksMutator(func(_ *DataAssetServiceManager, hooks *DataAssetRuntimeHooks) {
		hooks.Semantics = reviewedDataAssetRuntimeSemantics()
	})
}

func reviewedDataAssetRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "dataasset",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "none", Runtime: "generatedruntime", FormalClassification: "none"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",

		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"key", "modelVersion", "description", "objectStatus", "externalKey", "assetProperties", "registryMetadata", "name", "identifier", "host", "port", "protocol", "defaultConnection", "validateCertificate", "lakeId", "metastoreId", "lakeProxyEndpoint", "serviceName", "region", "baseUrl", "manifestFileContent", "driverClass", "sid", "walletSecret", "walletPasswordSecret", "dataAssetType", "credentialFileContent", "regionId", "tenancyId", "compartmentId", "autonomousDbId", "serviceUrl", "ociRegion", "url", "namespace", "objectVersion"}, ForceNew: []string{"jsonData", "modelType", "stagingDataAsset", "stagingConnection", "bucketSchema", "workspaceId"}, ZeroValueNullEquivalent: []string{"key", "modelVersion", "description", "objectStatus", "externalKey", "assetProperties", "registryMetadata", "name", "identifier", "host", "port", "protocol", "defaultConnection", "validateCertificate", "lakeId", "metastoreId", "lakeProxyEndpoint", "serviceName", "region", "baseUrl", "manifestFileContent", "driverClass", "sid", "walletSecret", "walletPasswordSecret", "dataAssetType", "credentialFileContent", "regionId", "tenancyId", "compartmentId", "autonomousDbId", "serviceUrl", "ociRegion", "url", "namespace", "objectVersion", "jsonData", "modelType", "stagingDataAsset", "stagingConnection", "bucketSchema", "workspaceId"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
