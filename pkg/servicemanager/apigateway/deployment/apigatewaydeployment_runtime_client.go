/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	apigatewaysdk "github.com/oracle/oci-go-sdk/v65/apigateway"
	apigatewayv1beta1 "github.com/oracle/oci-service-operator/api/apigateway/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerApiGatewayDeploymentRuntimeHooksMutator(func(manager *ApiGatewayDeploymentServiceManager, hooks *ApiGatewayDeploymentRuntimeHooks) {
		applyApiGatewayDeploymentRuntimeHooks(manager, hooks)
	})
}

func applyApiGatewayDeploymentRuntimeHooks(
	manager *ApiGatewayDeploymentServiceManager,
	hooks *ApiGatewayDeploymentRuntimeHooks,
) {
	if manager == nil || hooks == nil {
		return
	}
	if hooks.Semantics != nil && hooks.Semantics.List != nil {
		hooks.Semantics.List.ResponseItemsField = "Items"
		hooks.Semantics.List.MatchFields = []string{"gatewayId", "compartmentId", "displayName"}
	}
	if hooks.Semantics != nil {
		hooks.Semantics.Lifecycle.ProvisioningStates = []string{"CREATING"}
		hooks.Semantics.Lifecycle.UpdatingStates = []string{"UPDATING"}
		hooks.Semantics.Delete.PendingStates = []string{"DELETING"}
	}
	hooks.BuildCreateBody = func(
		ctx context.Context,
		resource *apigatewayv1beta1.ApiGatewayDeployment,
		namespace string,
	) (any, error) {
		return buildApiGatewayDeploymentCreateBody(ctx, manager, resource, namespace)
	}
	hooks.BuildUpdateBody = func(
		ctx context.Context,
		resource *apigatewayv1beta1.ApiGatewayDeployment,
		namespace string,
		currentResponse any,
	) (any, bool, error) {
		return buildApiGatewayDeploymentUpdateBody(ctx, manager, resource, namespace, currentResponse)
	}
}

func normalizedApiGatewayDeployment(
	resource *apigatewayv1beta1.ApiGatewayDeployment,
) (*apigatewayv1beta1.ApiGatewayDeployment, error) {
	if resource == nil {
		return nil, fmt.Errorf("ApiGatewayDeployment resource is nil")
	}
	copy := resource.DeepCopy()
	if len(copy.Spec.Routes) == 0 {
		return copy, nil
	}
	if len(copy.Spec.Specification.Routes) != 0 && !reflect.DeepEqual(copy.Spec.Routes, copy.Spec.Specification.Routes) {
		return nil, fmt.Errorf("ApiGatewayDeployment spec.routes conflicts with spec.specification.routes")
	}
	if len(copy.Spec.Specification.Routes) == 0 {
		copy.Spec.Specification.Routes = append(
			[]apigatewayv1beta1.ApiGatewayDeploymentSpecificationRoute(nil),
			copy.Spec.Routes...,
		)
	}
	return copy, nil
}

func buildApiGatewayDeploymentCreateBody(
	ctx context.Context,
	manager *ApiGatewayDeploymentServiceManager,
	resource *apigatewayv1beta1.ApiGatewayDeployment,
	namespace string,
) (apigatewaysdk.CreateDeploymentDetails, error) {
	normalized, err := normalizedApiGatewayDeployment(resource)
	if err != nil {
		return apigatewaysdk.CreateDeploymentDetails{}, err
	}
	resolved, err := generatedruntime.ResolveSpecValue(normalized, ctx, manager.CredentialClient, namespace)
	if err != nil {
		return apigatewaysdk.CreateDeploymentDetails{}, err
	}
	var details apigatewaysdk.CreateDeploymentDetails
	if err := convertApiGatewayDeploymentValue(resolved, &details); err != nil {
		return apigatewaysdk.CreateDeploymentDetails{}, fmt.Errorf("decode ApiGatewayDeployment create body: %w", err)
	}
	if details.Specification == nil {
		return apigatewaysdk.CreateDeploymentDetails{}, fmt.Errorf("ApiGatewayDeployment requires spec.specification or spec.routes")
	}
	return details, nil
}

func buildApiGatewayDeploymentUpdateBody(
	ctx context.Context,
	manager *ApiGatewayDeploymentServiceManager,
	resource *apigatewayv1beta1.ApiGatewayDeployment,
	namespace string,
	currentResponse any,
) (apigatewaysdk.UpdateDeploymentDetails, bool, error) {
	normalized, err := normalizedApiGatewayDeployment(resource)
	if err != nil {
		return apigatewaysdk.UpdateDeploymentDetails{}, false, err
	}
	resolved, err := generatedruntime.ResolveSpecValue(normalized, ctx, manager.CredentialClient, namespace)
	if err != nil {
		return apigatewaysdk.UpdateDeploymentDetails{}, false, err
	}
	var details apigatewaysdk.UpdateDeploymentDetails
	if err := convertApiGatewayDeploymentValue(resolved, &details); err != nil {
		return apigatewaysdk.UpdateDeploymentDetails{}, false, fmt.Errorf("decode ApiGatewayDeployment update body: %w", err)
	}

	current, err := apiGatewayDeploymentFromResponse(currentResponse)
	if err != nil {
		return apigatewaysdk.UpdateDeploymentDetails{}, false, err
	}
	return details, apiGatewayDeploymentUpdateNeeded(details, current), nil
}

func convertApiGatewayDeploymentValue(value any, destination any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, destination)
}

func apiGatewayDeploymentFromResponse(response any) (apigatewaysdk.Deployment, error) {
	switch value := response.(type) {
	case apigatewaysdk.GetDeploymentResponse:
		return value.Deployment, nil
	case *apigatewaysdk.GetDeploymentResponse:
		if value != nil {
			return value.Deployment, nil
		}
	case apigatewaysdk.Deployment:
		return value, nil
	case *apigatewaysdk.Deployment:
		if value != nil {
			return *value, nil
		}
	case nil:
		return apigatewaysdk.Deployment{}, nil
	}
	return apigatewaysdk.Deployment{}, fmt.Errorf("unsupported ApiGatewayDeployment current response %T", response)
}

func apiGatewayDeploymentUpdateNeeded(
	desired apigatewaysdk.UpdateDeploymentDetails,
	current apigatewaysdk.Deployment,
) bool {
	if desired.DisplayName != nil && (current.DisplayName == nil || *desired.DisplayName != *current.DisplayName) {
		return true
	}
	if desired.Specification != nil && !reflect.DeepEqual(desired.Specification, current.Specification) {
		return true
	}
	if desired.FreeformTags != nil && !reflect.DeepEqual(desired.FreeformTags, current.FreeformTags) {
		return true
	}
	return desired.DefinedTags != nil && !reflect.DeepEqual(desired.DefinedTags, current.DefinedTags)
}
