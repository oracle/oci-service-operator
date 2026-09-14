/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package gateway

import (
	"context"
	"testing"

	apigatewaysdk "github.com/oracle/oci-go-sdk/v65/apigateway"
	"github.com/oracle/oci-go-sdk/v65/common"
	apigatewayv1beta1 "github.com/oracle/oci-service-operator/api/apigateway/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type apiGatewayTestDelegate struct {
	createOrUpdate func(context.Context, *apigatewayv1beta1.ApiGateway, ctrl.Request) (servicemanager.OSOKResponse, error)
	delete         func(context.Context, *apigatewayv1beta1.ApiGateway) (bool, error)
}

func (d apiGatewayTestDelegate) CreateOrUpdate(ctx context.Context, resource *apigatewayv1beta1.ApiGateway, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	return d.createOrUpdate(ctx, resource, req)
}

func (d apiGatewayTestDelegate) Delete(ctx context.Context, resource *apigatewayv1beta1.ApiGateway) (bool, error) {
	return d.delete(ctx, resource)
}

type apiGatewayTestCredentialClient struct {
	data       map[string][]byte
	createCall int
	deleteCall int
}

func (c *apiGatewayTestCredentialClient) CreateSecret(_ context.Context, _, _ string, _ map[string]string, data map[string][]byte) (bool, error) {
	c.createCall++
	c.data = data
	return true, nil
}

func (c *apiGatewayTestCredentialClient) DeleteSecret(context.Context, string, string) (bool, error) {
	c.deleteCall++
	c.data = nil
	return true, nil
}

func (c *apiGatewayTestCredentialClient) GetSecret(context.Context, string, string) (map[string][]byte, error) {
	return c.data, nil
}

func (c *apiGatewayTestCredentialClient) UpdateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return true, nil
}

func TestApplyApiGatewayRuntimeHooksUsesSDKCollectionAndEndpointSecretWrapper(t *testing.T) {
	t.Parallel()
	hooks := ApiGatewayRuntimeHooks{Semantics: newApiGatewayRuntimeSemantics()}
	credentials := &apiGatewayTestCredentialClient{}
	applyApiGatewayRuntimeHooks(&ApiGatewayServiceManager{CredentialClient: credentials}, &hooks)
	if got := hooks.Semantics.List.ResponseItemsField; got != "Items" {
		t.Fatalf("List response items field = %q, want Items", got)
	}
	if len(hooks.WrapGeneratedClient) != 1 {
		t.Fatalf("generated client wrappers = %d, want 1", len(hooks.WrapGeneratedClient))
	}
	if hooks.BuildUpdateBody == nil {
		t.Fatal("typed Gateway update builder is not installed")
	}

	delegate := apiGatewayTestDelegate{
		createOrUpdate: func(context.Context, *apigatewayv1beta1.ApiGateway, ctrl.Request) (servicemanager.OSOKResponse, error) {
			return servicemanager.OSOKResponse{IsSuccessful: true}, nil
		},
		delete: func(context.Context, *apigatewayv1beta1.ApiGateway) (bool, error) { return true, nil },
	}
	client := hooks.WrapGeneratedClient[0](delegate)
	resource := &apigatewayv1beta1.ApiGateway{}
	resource.Name = "gateway"
	resource.Namespace = "default"
	resource.Status.LifecycleState = "ACTIVE"
	resource.Status.OsokStatus.Reason = string(shared.Active)
	resource.Status.Hostname = "gateway.example.com"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if !response.IsSuccessful || credentials.createCall != 1 {
		t.Fatalf("CreateOrUpdate response = %+v, secret creates = %d", response, credentials.createCall)
	}
	if got := string(credentials.data["hostname"]); got != "gateway.example.com" {
		t.Fatalf("secret hostname = %q", got)
	}
	if !servicemanager.SecretOwnedBy(credentials.data, "ApiGateway", "gateway") {
		t.Fatal("endpoint Secret is missing managed ownership metadata")
	}

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatal(err)
	}
	if !deleted || credentials.deleteCall != 1 {
		t.Fatalf("Delete = %t, secret deletes = %d", deleted, credentials.deleteCall)
	}
}

func TestBuildApiGatewayUpdateBodyOmitsEmptyPolymorphicBlocks(t *testing.T) {
	t.Parallel()
	resource := &apigatewayv1beta1.ApiGateway{Spec: apigatewayv1beta1.ApiGatewaySpec{
		CompartmentId: "ocid1.compartment.oc1..mock",
		EndpointType:  "PRIVATE",
		SubnetId:      "ocid1.subnet.oc1..mock",
		DisplayName:   "gateway-updated",
		FreeformTags:  map[string]string{"osok-mock": "update"},
	}}
	current := apigatewaysdk.GetGatewayResponse{Gateway: apigatewaysdk.Gateway{
		DisplayName:    common.String("gateway"),
		FreeformTags:   map[string]string{"osok-mock": "create"},
		LifecycleState: apigatewaysdk.GatewayLifecycleStateActive,
	}}
	details, needed, err := buildApiGatewayUpdateBody(context.Background(), &ApiGatewayServiceManager{}, resource, "default", current)
	if err != nil {
		t.Fatal(err)
	}
	if !needed {
		t.Fatal("Gateway mutable drift was not detected")
	}
	if details.ResponseCacheDetails != nil {
		t.Fatalf("empty responseCacheDetails projected as %T", details.ResponseCacheDetails)
	}
	if details.DisplayName == nil || *details.DisplayName != "gateway-updated" || details.FreeformTags["osok-mock"] != "update" {
		t.Fatalf("Gateway update details = %+v", details)
	}
}

func TestBuildApiGatewayUpdateBodyPreservesExplicitFalseResponseCacheBoolean(t *testing.T) {
	t.Parallel()

	resource := &apigatewayv1beta1.ApiGateway{Spec: apigatewayv1beta1.ApiGatewaySpec{
		CompartmentId: "ocid1.compartment.oc1..mock",
		EndpointType:  "PRIVATE",
		SubnetId:      "ocid1.subnet.oc1..mock",
		ResponseCacheDetails: apigatewayv1beta1.ApiGatewayResponseCacheDetails{
			Type: "EXTERNAL_RESP_CACHE",
			Servers: []apigatewayv1beta1.ApiGatewayResponseCacheDetailsServer{{
				Host: "cache.example.com",
				Port: 6379,
			}},
			AuthenticationSecretId:            "ocid1.vaultsecret.oc1..mock",
			AuthenticationSecretVersionNumber: 1,
			IsSslEnabled:                      common.Bool(false),
		},
	}}
	current := apigatewaysdk.GetGatewayResponse{Gateway: apigatewaysdk.Gateway{
		ResponseCacheDetails: apigatewaysdk.ExternalRespCache{
			Servers: []apigatewaysdk.ResponseCacheRespServer{{
				Host: common.String("cache.example.com"),
				Port: common.Int(6379),
			}},
			AuthenticationSecretId:            common.String("ocid1.vaultsecret.oc1..mock"),
			AuthenticationSecretVersionNumber: common.Int64(1),
			IsSslEnabled:                      common.Bool(true),
		},
	}}

	details, needed, err := buildApiGatewayUpdateBody(context.Background(), &ApiGatewayServiceManager{}, resource, "default", current)
	if err != nil {
		t.Fatal(err)
	}
	if !needed {
		t.Fatal("Gateway true-to-false response cache drift was not detected")
	}
	cache, ok := details.ResponseCacheDetails.(apigatewaysdk.ExternalRespCache)
	if !ok {
		t.Fatalf("responseCacheDetails = %T, want apigateway.ExternalRespCache", details.ResponseCacheDetails)
	}
	if cache.IsSslEnabled == nil || *cache.IsSslEnabled {
		t.Fatalf("responseCacheDetails.isSslEnabled = %v, want explicit false", cache.IsSslEnabled)
	}
	if cache.IsSslVerifyDisabled != nil {
		t.Fatalf("omitted responseCacheDetails.isSslVerifyDisabled = %v, want nil", cache.IsSslVerifyDisabled)
	}
}

func TestApiGatewayEndpointSecretWaitsForReadyHostname(t *testing.T) {
	t.Parallel()
	credentials := &apiGatewayTestCredentialClient{}
	client := apiGatewayEndpointSecretClient{
		credentialClient: credentials,
		delegate: apiGatewayTestDelegate{
			createOrUpdate: func(context.Context, *apigatewayv1beta1.ApiGateway, ctrl.Request) (servicemanager.OSOKResponse, error) {
				return servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true}, nil
			},
			delete: func(context.Context, *apigatewayv1beta1.ApiGateway) (bool, error) { return false, nil },
		},
	}
	response, err := client.CreateOrUpdate(context.Background(), &apigatewayv1beta1.ApiGateway{}, ctrl.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if !response.ShouldRequeue || credentials.createCall != 0 {
		t.Fatalf("response = %+v, secret creates = %d", response, credentials.createCall)
	}
}
