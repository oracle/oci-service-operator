/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package connection

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

func init() {
	registerConnectionRuntimeHooksMutator(func(_ *ConnectionServiceManager, hooks *ConnectionRuntimeHooks) {
		hooks.Semantics = reviewedConnectionRuntimeSemantics()
	})
}

func reviewedConnectionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "dataintegration", FormalSlug: "connection",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "none", Runtime: "generatedruntime", FormalClassification: "none"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",

		Delete:         generatedruntime.DeleteSemantics{Policy: "required", PendingStates: []string{"DELETING", "TERMINATING"}, TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"key", "modelVersion", "parentRef", "description", "objectStatus", "connectionProperties", "registryMetadata", "name", "identifier", "username", "password", "passwordSecret", "accessKey", "secretKey", "defaultExternalStorage", "tnsAlias", "tnsNames", "hdfsPrincipal", "dataNodePrincipal", "nameNodePrincipal", "realm", "keyDistributionCenter", "keyTabContent", "authHeader", "accessTokenUrl", "clientId", "clientSecret", "scope", "grantType", "credentialFileContent", "userId", "fingerPrint", "passPhrase", "objectVersion"}, ForceNew: []string{"jsonData", "modelType", "workspaceId", "dataAssetKey"}, ZeroValueNullEquivalent: []string{"key", "modelVersion", "parentRef", "description", "objectStatus", "connectionProperties", "registryMetadata", "name", "identifier", "username", "password", "passwordSecret", "accessKey", "secretKey", "defaultExternalStorage", "tnsAlias", "tnsNames", "hdfsPrincipal", "dataNodePrincipal", "nameNodePrincipal", "realm", "keyDistributionCenter", "keyTabContent", "authHeader", "accessTokenUrl", "clientId", "clientSecret", "scope", "grantType", "credentialFileContent", "userId", "fingerPrint", "passPhrase", "objectVersion", "jsonData", "modelType", "workspaceId", "dataAssetKey"}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
