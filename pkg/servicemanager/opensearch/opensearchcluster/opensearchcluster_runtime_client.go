/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package opensearchcluster

import (
	"context"
	"encoding/json"
	"fmt"

	opensearchsdk "github.com/oracle/oci-go-sdk/v65/opensearch"
	opensearchv1beta1 "github.com/oracle/oci-service-operator/api/opensearch/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerOpensearchClusterRuntimeHooksMutator(func(
		manager *OpensearchClusterServiceManager,
		hooks *OpensearchClusterRuntimeHooks,
	) {
		applyOpensearchClusterRuntimeHooks(manager, hooks)
	})
}

func applyOpensearchClusterRuntimeHooks(
	manager *OpensearchClusterServiceManager,
	hooks *OpensearchClusterRuntimeHooks,
) {
	if hooks == nil {
		return
	}

	var credentialClient credhelper.CredentialClient
	if manager != nil {
		credentialClient = manager.CredentialClient
	}

	hooks.Semantics = newOpensearchClusterRuntimeSemantics()
	hooks.BuildCreateBody = func(ctx context.Context, resource *opensearchv1beta1.OpensearchCluster, namespace string) (any, error) {
		return buildOpensearchCreateDetails(ctx, credentialClient, resource, namespace)
	}
}

func buildOpensearchCreateDetails(ctx context.Context, credentialClient credhelper.CredentialClient, resource *opensearchv1beta1.OpensearchCluster, namespace string) (opensearchsdk.CreateOpensearchClusterDetails, error) {
	resolvedSpec, err := generatedruntime.ResolveSpecValue(resource, ctx, credentialClient, namespace)
	if err != nil {
		return opensearchsdk.CreateOpensearchClusterDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return opensearchsdk.CreateOpensearchClusterDetails{}, fmt.Errorf("marshal resolved opensearch spec: %w", err)
	}

	var details opensearchsdk.CreateOpensearchClusterDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return opensearchsdk.CreateOpensearchClusterDetails{}, fmt.Errorf("decode opensearch create request body: %w", err)
	}
	return details, nil
}
