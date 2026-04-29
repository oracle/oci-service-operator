/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apiplatforminstance

import (
	"context"

	apiplatformsdk "github.com/oracle/oci-go-sdk/v65/apiplatform"
	apiplatformv1beta1 "github.com/oracle/oci-service-operator/api/apiplatform/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type apiPlatformInstanceOCIClient interface {
	CreateApiPlatformInstance(context.Context, apiplatformsdk.CreateApiPlatformInstanceRequest) (apiplatformsdk.CreateApiPlatformInstanceResponse, error)
	GetApiPlatformInstance(context.Context, apiplatformsdk.GetApiPlatformInstanceRequest) (apiplatformsdk.GetApiPlatformInstanceResponse, error)
	ListApiPlatformInstances(context.Context, apiplatformsdk.ListApiPlatformInstancesRequest) (apiplatformsdk.ListApiPlatformInstancesResponse, error)
	UpdateApiPlatformInstance(context.Context, apiplatformsdk.UpdateApiPlatformInstanceRequest) (apiplatformsdk.UpdateApiPlatformInstanceResponse, error)
	DeleteApiPlatformInstance(context.Context, apiplatformsdk.DeleteApiPlatformInstanceRequest) (apiplatformsdk.DeleteApiPlatformInstanceResponse, error)
}

func init() {
	registerApiPlatformInstanceRuntimeHooksMutator(func(_ *ApiPlatformInstanceServiceManager, hooks *ApiPlatformInstanceRuntimeHooks) {
		applyApiPlatformInstanceRuntimeHooks(hooks)
	})
}

func applyApiPlatformInstanceRuntimeHooks(hooks *ApiPlatformInstanceRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedApiPlatformInstanceRuntimeSemantics()
	hooks.List.Fields = apiPlatformInstanceListFields()
}

func newApiPlatformInstanceServiceClientWithOCIClient(
	log loggerutil.OSOKLogger,
	client apiPlatformInstanceOCIClient,
) ApiPlatformInstanceServiceClient {
	return defaultApiPlatformInstanceServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*apiplatformv1beta1.ApiPlatformInstance](
			newApiPlatformInstanceRuntimeConfig(log, client),
		),
	}
}

func newApiPlatformInstanceRuntimeConfig(
	log loggerutil.OSOKLogger,
	client apiPlatformInstanceOCIClient,
) generatedruntime.Config[*apiplatformv1beta1.ApiPlatformInstance] {
	hooks := newApiPlatformInstanceRuntimeHooksWithOCIClient(client)
	applyApiPlatformInstanceRuntimeHooks(&hooks)
	return buildApiPlatformInstanceGeneratedRuntimeConfig(&ApiPlatformInstanceServiceManager{Log: log}, hooks)
}

func newApiPlatformInstanceRuntimeHooksWithOCIClient(client apiPlatformInstanceOCIClient) ApiPlatformInstanceRuntimeHooks {
	return ApiPlatformInstanceRuntimeHooks{
		Semantics: reviewedApiPlatformInstanceRuntimeSemantics(),
		Create: runtimeOperationHooks[apiplatformsdk.CreateApiPlatformInstanceRequest, apiplatformsdk.CreateApiPlatformInstanceResponse]{
			Fields: apiPlatformInstanceCreateFields(),
			Call: func(ctx context.Context, request apiplatformsdk.CreateApiPlatformInstanceRequest) (apiplatformsdk.CreateApiPlatformInstanceResponse, error) {
				return client.CreateApiPlatformInstance(ctx, request)
			},
		},
		Get: runtimeOperationHooks[apiplatformsdk.GetApiPlatformInstanceRequest, apiplatformsdk.GetApiPlatformInstanceResponse]{
			Fields: apiPlatformInstanceGetFields(),
			Call: func(ctx context.Context, request apiplatformsdk.GetApiPlatformInstanceRequest) (apiplatformsdk.GetApiPlatformInstanceResponse, error) {
				return client.GetApiPlatformInstance(ctx, request)
			},
		},
		List: runtimeOperationHooks[apiplatformsdk.ListApiPlatformInstancesRequest, apiplatformsdk.ListApiPlatformInstancesResponse]{
			Fields: apiPlatformInstanceListFields(),
			Call: func(ctx context.Context, request apiplatformsdk.ListApiPlatformInstancesRequest) (apiplatformsdk.ListApiPlatformInstancesResponse, error) {
				return client.ListApiPlatformInstances(ctx, request)
			},
		},
		Update: runtimeOperationHooks[apiplatformsdk.UpdateApiPlatformInstanceRequest, apiplatformsdk.UpdateApiPlatformInstanceResponse]{
			Fields: apiPlatformInstanceUpdateFields(),
			Call: func(ctx context.Context, request apiplatformsdk.UpdateApiPlatformInstanceRequest) (apiplatformsdk.UpdateApiPlatformInstanceResponse, error) {
				return client.UpdateApiPlatformInstance(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[apiplatformsdk.DeleteApiPlatformInstanceRequest, apiplatformsdk.DeleteApiPlatformInstanceResponse]{
			Fields: apiPlatformInstanceDeleteFields(),
			Call: func(ctx context.Context, request apiplatformsdk.DeleteApiPlatformInstanceRequest) (apiplatformsdk.DeleteApiPlatformInstanceResponse, error) {
				return client.DeleteApiPlatformInstance(ctx, request)
			},
		},
	}
}

func reviewedApiPlatformInstanceRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newApiPlatformInstanceRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "name"},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func apiPlatformInstanceCreateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CreateApiPlatformInstanceDetails", RequestName: "CreateApiPlatformInstanceDetails", Contribution: "body"},
	}
}

func apiPlatformInstanceGetFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ApiPlatformInstanceId", RequestName: "apiPlatformInstanceId", Contribution: "path", PreferResourceID: true},
	}
}

func apiPlatformInstanceListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{
			FieldName:    "CompartmentId",
			RequestName:  "compartmentId",
			Contribution: "query",
			LookupPaths:  []string{"status.compartmentId", "spec.compartmentId", "compartmentId"},
		},
		{
			FieldName:    "Name",
			RequestName:  "name",
			Contribution: "query",
			LookupPaths:  []string{"status.name", "spec.name", "name"},
		},
		{FieldName: "Id", RequestName: "id", Contribution: "query", PreferResourceID: true},
	}
}

func apiPlatformInstanceUpdateFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ApiPlatformInstanceId", RequestName: "apiPlatformInstanceId", Contribution: "path", PreferResourceID: true},
		{FieldName: "UpdateApiPlatformInstanceDetails", RequestName: "UpdateApiPlatformInstanceDetails", Contribution: "body"},
	}
}

func apiPlatformInstanceDeleteFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "ApiPlatformInstanceId", RequestName: "apiPlatformInstanceId", Contribution: "path", PreferResourceID: true},
	}
}
