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
	newOpensearchClusterServiceClient = func(manager *OpensearchClusterServiceManager) OpensearchClusterServiceClient {
		sdkClient, err := opensearchsdk.NewOpensearchClusterClientWithConfigurationProvider(manager.Provider)
		config := generatedruntime.Config[*opensearchv1beta1.OpensearchCluster]{
			Kind:             "OpensearchCluster",
			SDKName:          "OpensearchCluster",
			Log:              manager.Log,
			CredentialClient: manager.CredentialClient,
			BuildCreateBody: func(ctx context.Context, resource *opensearchv1beta1.OpensearchCluster, namespace string) (any, error) {
				return buildOpensearchCreateDetails(ctx, manager.CredentialClient, resource, namespace)
			},
			Semantics: &generatedruntime.Semantics{
				FormalService:     "opensearch",
				FormalSlug:        "opensearchopensearchcluster",
				StatusProjection:  "required",
				SecretSideEffects: "none",
				FinalizerPolicy:   "retain-until-confirmed-delete",
				Lifecycle: generatedruntime.LifecycleSemantics{
					ProvisioningStates: []string{"CREATING"},
					UpdatingStates:     []string{"UPDATING"},
					ActiveStates:       []string{"ACTIVE"},
				},
				Delete: generatedruntime.DeleteSemantics{
					Policy:         "required",
					PendingStates:  []string{"DELETING"},
					TerminalStates: []string{"DELETED"},
				},
				List: &generatedruntime.ListSemantics{
					ResponseItemsField: "Items",
					MatchFields:        []string{"compartmentId", "displayName", "state"},
				},
				Mutation: generatedruntime.MutationSemantics{
					Mutable:       []string{"displayName"},
					ForceNew:      []string{"compartmentId"},
					ConflictsWith: map[string][]string{},
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
			},
			Create: &generatedruntime.Operation{
				NewRequest: func() any { return &opensearchsdk.CreateOpensearchClusterRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.CreateOpensearchCluster(ctx, *request.(*opensearchsdk.CreateOpensearchClusterRequest))
				},
				Fields: []generatedruntime.RequestField{{FieldName: "CreateOpensearchClusterDetails", RequestName: "CreateOpensearchClusterDetails", Contribution: "body", PreferResourceID: false}},
			},
			Get: &generatedruntime.Operation{
				NewRequest: func() any { return &opensearchsdk.GetOpensearchClusterRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.GetOpensearchCluster(ctx, *request.(*opensearchsdk.GetOpensearchClusterRequest))
				},
				Fields: []generatedruntime.RequestField{{FieldName: "OpensearchClusterId", RequestName: "opensearchClusterId", Contribution: "path", PreferResourceID: true}},
			},
			List: &generatedruntime.Operation{
				NewRequest: func() any { return &opensearchsdk.ListOpensearchClustersRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.ListOpensearchClusters(ctx, *request.(*opensearchsdk.ListOpensearchClustersRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query", PreferResourceID: false},
					{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query", PreferResourceID: false},
					{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query", PreferResourceID: false},
					{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: false},
					{FieldName: "Limit", RequestName: "limit", Contribution: "query", PreferResourceID: false},
					{FieldName: "Page", RequestName: "page", Contribution: "query", PreferResourceID: false},
					{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query", PreferResourceID: false},
					{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query", PreferResourceID: false},
				},
			},
			Update: &generatedruntime.Operation{
				NewRequest: func() any { return &opensearchsdk.UpdateOpensearchClusterRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.UpdateOpensearchCluster(ctx, *request.(*opensearchsdk.UpdateOpensearchClusterRequest))
				},
				Fields: []generatedruntime.RequestField{
					{FieldName: "OpensearchClusterId", RequestName: "opensearchClusterId", Contribution: "path", PreferResourceID: true},
					{FieldName: "UpdateOpensearchClusterDetails", RequestName: "UpdateOpensearchClusterDetails", Contribution: "body", PreferResourceID: false},
				},
			},
			Delete: &generatedruntime.Operation{
				NewRequest: func() any { return &opensearchsdk.DeleteOpensearchClusterRequest{} },
				Call: func(ctx context.Context, request any) (any, error) {
					return sdkClient.DeleteOpensearchCluster(ctx, *request.(*opensearchsdk.DeleteOpensearchClusterRequest))
				},
				Fields: []generatedruntime.RequestField{{FieldName: "OpensearchClusterId", RequestName: "opensearchClusterId", Contribution: "path", PreferResourceID: true}},
			},
		}
		if err != nil {
			config.InitError = fmt.Errorf("initialize OpensearchCluster OCI client: %w", err)
		}
		return defaultOpensearchClusterServiceClient{
			ServiceClient: generatedruntime.NewServiceClient[*opensearchv1beta1.OpensearchCluster](config),
		}
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
