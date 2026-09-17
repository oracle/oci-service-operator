/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	apigatewaysdk "github.com/oracle/oci-go-sdk/v65/apigateway"
	apigatewayv1beta1 "github.com/oracle/oci-service-operator/api/apigateway/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	registerApiGatewayRuntimeHooksMutator(func(manager *ApiGatewayServiceManager, hooks *ApiGatewayRuntimeHooks) {
		applyApiGatewayRuntimeHooks(manager, hooks)
	})
}

func applyApiGatewayRuntimeHooks(manager *ApiGatewayServiceManager, hooks *ApiGatewayRuntimeHooks) {
	if manager == nil || hooks == nil {
		return
	}
	if hooks.Semantics != nil && hooks.Semantics.List != nil {
		// The OCI SDK embeds GatewayCollection in ListGatewaysResponse. Its
		// resource slice is Items; the Terraform provider's local variable name
		// is not an SDK response field.
		hooks.Semantics.List.ResponseItemsField = "Items"
		hooks.Semantics.List.MatchFields = []string{"compartmentId", "displayName"}
	}
	if hooks.Semantics != nil {
		hooks.Semantics.Lifecycle.ProvisioningStates = []string{"CREATING"}
		hooks.Semantics.Lifecycle.UpdatingStates = []string{"UPDATING"}
		hooks.Semantics.Delete.PendingStates = []string{"DELETING"}
	}
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *apigatewayv1beta1.ApiGateway,
		namespace string,
		currentResponse any,
	) (any, bool, error) {
		return buildApiGatewayUpdateBody(ctx, manager, resource, namespace, currentResponse)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate ApiGatewayServiceClient) ApiGatewayServiceClient {
		return apiGatewayEndpointSecretClient{
			delegate:         delegate,
			credentialClient: manager.CredentialClient,
		}
	})
}

func buildApiGatewayUpdateBody(
	ctx context.Context,
	manager *ApiGatewayServiceManager,
	resource *apigatewayv1beta1.ApiGateway,
	namespace string,
	currentResponse any,
) (apigatewaysdk.UpdateGatewayDetails, bool, error) {
	if resource == nil {
		return apigatewaysdk.UpdateGatewayDetails{}, false, fmt.Errorf("ApiGateway resource is nil")
	}
	resolved, err := generatedruntime.ResolveSpecValue(resource, ctx, manager.CredentialClient, namespace)
	if err != nil {
		return apigatewaysdk.UpdateGatewayDetails{}, false, err
	}
	payload, err := json.Marshal(resolved)
	if err != nil {
		return apigatewaysdk.UpdateGatewayDetails{}, false, fmt.Errorf("marshal ApiGateway update body: %w", err)
	}
	var details apigatewaysdk.UpdateGatewayDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return apigatewaysdk.UpdateGatewayDetails{}, false, fmt.Errorf("decode ApiGateway update body: %w", err)
	}

	current, err := apiGatewayFromResponse(currentResponse)
	if err != nil {
		return apigatewaysdk.UpdateGatewayDetails{}, false, err
	}
	return details, apiGatewayUpdateNeeded(details, current), nil
}

func apiGatewayFromResponse(response any) (apigatewaysdk.Gateway, error) {
	switch value := response.(type) {
	case apigatewaysdk.GetGatewayResponse:
		return value.Gateway, nil
	case *apigatewaysdk.GetGatewayResponse:
		if value != nil {
			return value.Gateway, nil
		}
	case apigatewaysdk.Gateway:
		return value, nil
	case *apigatewaysdk.Gateway:
		if value != nil {
			return *value, nil
		}
	case nil:
		return apigatewaysdk.Gateway{}, nil
	}
	return apigatewaysdk.Gateway{}, fmt.Errorf("unsupported ApiGateway current response %T", response)
}

func apiGatewayUpdateNeeded(desired apigatewaysdk.UpdateGatewayDetails, current apigatewaysdk.Gateway) bool {
	if desired.DisplayName != nil && (current.DisplayName == nil || *desired.DisplayName != *current.DisplayName) {
		return true
	}
	if desired.CertificateId != nil && (current.CertificateId == nil || *desired.CertificateId != *current.CertificateId) {
		return true
	}
	if desired.NetworkSecurityGroupIds != nil && !reflect.DeepEqual(desired.NetworkSecurityGroupIds, current.NetworkSecurityGroupIds) {
		return true
	}
	if desired.ResponseCacheDetails != nil && !reflect.DeepEqual(desired.ResponseCacheDetails, current.ResponseCacheDetails) {
		return true
	}
	if desired.CaBundles != nil && !reflect.DeepEqual(desired.CaBundles, current.CaBundles) {
		return true
	}
	if desired.FreeformTags != nil && !reflect.DeepEqual(desired.FreeformTags, current.FreeformTags) {
		return true
	}
	return desired.DefinedTags != nil && !reflect.DeepEqual(desired.DefinedTags, current.DefinedTags)
}

type apiGatewayEndpointSecretClient struct {
	delegate         ApiGatewayServiceClient
	credentialClient credhelper.CredentialClient
}

var _ ApiGatewayServiceClient = apiGatewayEndpointSecretClient{}

func (c apiGatewayEndpointSecretClient) CreateOrUpdate(
	ctx context.Context,
	resource *apigatewayv1beta1.ApiGateway,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil || !response.IsSuccessful || !apiGatewayReadyForEndpointSecret(resource) {
		return response, err
	}
	if c.credentialClient == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("ApiGateway endpoint secret credential client is not configured")
	}

	hostname := strings.TrimSpace(resource.Status.Hostname)
	if hostname == "" {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("ApiGateway endpoint secret requires an observed hostname")
	}
	_, err = servicemanager.EnsureOwnedSecret(
		ctx,
		c.credentialClient,
		resource.Name,
		resource.Namespace,
		"ApiGateway",
		resource.Name,
		map[string][]byte{"hostname": []byte(hostname)},
	)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return response, nil
}

func (c apiGatewayEndpointSecretClient) Delete(
	ctx context.Context,
	resource *apigatewayv1beta1.ApiGateway,
) (bool, error) {
	deleted, err := c.delegate.Delete(ctx, resource)
	if err != nil || !deleted {
		return deleted, err
	}
	if c.credentialClient == nil {
		return false, fmt.Errorf("ApiGateway endpoint secret credential client is not configured")
	}
	_, err = servicemanager.DeleteOwnedSecretIfPresent(
		ctx,
		c.credentialClient,
		resource.Name,
		resource.Namespace,
		"ApiGateway",
		resource.Name,
	)
	if err != nil {
		return false, err
	}
	return true, nil
}

func apiGatewayReadyForEndpointSecret(resource *apigatewayv1beta1.ApiGateway) bool {
	if resource == nil {
		return false
	}
	if strings.EqualFold(resource.Status.LifecycleState, "ACTIVE") || resource.Status.OsokStatus.Reason == string(shared.Active) {
		return true
	}
	conditions := resource.Status.OsokStatus.Conditions
	return len(conditions) > 0 && conditions[len(conditions)-1].Type == shared.Active
}
