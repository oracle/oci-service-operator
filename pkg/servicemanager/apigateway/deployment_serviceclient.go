/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apigateway

import (
	"context"
	"fmt"
	"reflect"
	"time"

	apigatewaysdk "github.com/oracle/oci-go-sdk/v65/apigateway"
	"github.com/oracle/oci-go-sdk/v65/common"
	apigatewayv1beta1 "github.com/oracle/oci-service-operator/api/apigateway/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// DeploymentClientInterface is the subset of apigateway.DeploymentClient methods used by DeploymentServiceManager.
type DeploymentClientInterface interface {
	CreateDeployment(ctx context.Context, request apigatewaysdk.CreateDeploymentRequest) (apigatewaysdk.CreateDeploymentResponse, error)
	GetDeployment(ctx context.Context, request apigatewaysdk.GetDeploymentRequest) (apigatewaysdk.GetDeploymentResponse, error)
	ListDeployments(ctx context.Context, request apigatewaysdk.ListDeploymentsRequest) (apigatewaysdk.ListDeploymentsResponse, error)
	ChangeDeploymentCompartment(ctx context.Context, request apigatewaysdk.ChangeDeploymentCompartmentRequest) (apigatewaysdk.ChangeDeploymentCompartmentResponse, error)
	UpdateDeployment(ctx context.Context, request apigatewaysdk.UpdateDeploymentRequest) (apigatewaysdk.UpdateDeploymentResponse, error)
	DeleteDeployment(ctx context.Context, request apigatewaysdk.DeleteDeploymentRequest) (apigatewaysdk.DeleteDeploymentResponse, error)
}

// getDeploymentClientOrCreate returns the injected client when set, otherwise creates one from the provider.
func (c *DeploymentServiceManager) getDeploymentClientOrCreate() (DeploymentClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return apigatewaysdk.NewDeploymentClientWithConfigurationProvider(c.Provider)
}

// buildApiSpecification converts CRD route specs into the OCI SDK ApiSpecification type.
func buildApiSpecification(routes []apigatewayv1beta1.ApiGatewayRoute) *apigatewaysdk.ApiSpecification {
	sdkRoutes := make([]apigatewaysdk.ApiSpecificationRoute, 0, len(routes))
	for _, route := range routes {
		var backend apigatewaysdk.ApiSpecificationRouteBackend
		switch route.Backend.Type {
		case "HTTP_BACKEND":
			backend = apigatewaysdk.HttpBackend{
				Url: common.String(route.Backend.Url),
			}
		case "ORACLE_FUNCTIONS_BACKEND":
			backend = apigatewaysdk.OracleFunctionBackend{
				FunctionId: common.String(route.Backend.FunctionId),
			}
		case "STOCK_RESPONSE_BACKEND":
			backend = apigatewaysdk.StockResponseBackend{
				Status: common.Int(route.Backend.Status),
				Body:   common.String(route.Backend.Body),
			}
		default:
			backend = apigatewaysdk.HttpBackend{
				Url: common.String(route.Backend.Url),
			}
		}

		sdkRoute := apigatewaysdk.ApiSpecificationRoute{
			Path:    common.String(route.Path),
			Backend: backend,
		}
		if len(route.Methods) > 0 {
			methods := make([]apigatewaysdk.ApiSpecificationRouteMethodsEnum, 0, len(route.Methods))
			for _, method := range route.Methods {
				methods = append(methods, apigatewaysdk.ApiSpecificationRouteMethodsEnum(method))
			}
			sdkRoute.Methods = methods
		}
		sdkRoutes = append(sdkRoutes, sdkRoute)
	}
	return &apigatewaysdk.ApiSpecification{
		Routes: sdkRoutes,
	}
}

// CreateDeployment calls the OCI API to create a new API Gateway Deployment.
func (c *DeploymentServiceManager) CreateDeployment(ctx context.Context, dep apigatewayv1beta1.ApiGatewayDeployment) (apigatewaysdk.CreateDeploymentResponse, error) {
	client, err := c.getDeploymentClientOrCreate()
	if err != nil {
		return apigatewaysdk.CreateDeploymentResponse{}, err
	}

	c.Log.DebugLog("Creating ApiGatewayDeployment", "displayName", dep.Spec.DisplayName)

	details := apigatewaysdk.CreateDeploymentDetails{
		GatewayId:     common.String(string(dep.Spec.GatewayId)),
		CompartmentId: common.String(string(dep.Spec.CompartmentId)),
		PathPrefix:    common.String(dep.Spec.PathPrefix),
		Specification: buildApiSpecification(dep.Spec.Routes),
	}

	if dep.Spec.DisplayName != "" {
		details.DisplayName = common.String(dep.Spec.DisplayName)
	}
	if dep.Spec.FreeFormTags != nil {
		details.FreeformTags = dep.Spec.FreeFormTags
	}
	if dep.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&dep.Spec.DefinedTags)
	}

	req := apigatewaysdk.CreateDeploymentRequest{
		CreateDeploymentDetails: details,
	}
	return client.CreateDeployment(ctx, req)
}

// GetDeployment retrieves an API Gateway Deployment by OCID.
func (c *DeploymentServiceManager) GetDeployment(ctx context.Context, deploymentID shared.OCID, retryPolicy *common.RetryPolicy) (*apigatewaysdk.Deployment, error) {
	client, err := c.getDeploymentClientOrCreate()
	if err != nil {
		return nil, err
	}

	req := apigatewaysdk.GetDeploymentRequest{
		DeploymentId: common.String(string(deploymentID)),
	}
	if retryPolicy != nil {
		req.RequestMetadata.RetryPolicy = retryPolicy
	}

	resp, err := client.GetDeployment(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Deployment, nil
}

// GetDeploymentOcid looks up an existing deployment by display name, gateway, and compartment.
func (c *DeploymentServiceManager) GetDeploymentOcid(ctx context.Context, dep apigatewayv1beta1.ApiGatewayDeployment) (*shared.OCID, error) {
	client, err := c.getDeploymentClientOrCreate()
	if err != nil {
		return nil, err
	}

	req := apigatewaysdk.ListDeploymentsRequest{
		CompartmentId: common.String(string(dep.Spec.CompartmentId)),
		GatewayId:     common.String(string(dep.Spec.GatewayId)),
		DisplayName:   common.String(dep.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	resp, err := client.ListDeployments(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing ApiGatewayDeployments")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "CREATING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("ApiGatewayDeployment %s exists with OCID %s", dep.Spec.DisplayName, *item.Id))
			return (*shared.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("ApiGatewayDeployment %s does not exist", dep.Spec.DisplayName))
	return nil, nil
}

// UpdateDeployment updates an existing API Gateway Deployment.
func (c *DeploymentServiceManager) UpdateDeployment(ctx context.Context, dep *apigatewayv1beta1.ApiGatewayDeployment) error {
	client, err := c.getDeploymentClientOrCreate()
	if err != nil {
		return err
	}

	targetID, err := servicemanager.ResolveResourceID(dep.Status.OsokStatus.Ocid, dep.Spec.DeploymentId)
	if err != nil {
		return err
	}

	existing, err := c.GetDeployment(ctx, targetID, nil)
	if err != nil {
		return err
	}

	if err := validateDeploymentUnsupportedChanges(dep, existing); err != nil {
		return err
	}

	if dep.Spec.CompartmentId != "" &&
		(existing.CompartmentId == nil || *existing.CompartmentId != string(dep.Spec.CompartmentId)) {
		if _, err = client.ChangeDeploymentCompartment(ctx, apigatewaysdk.ChangeDeploymentCompartmentRequest{
			DeploymentId: common.String(string(targetID)),
			ChangeDeploymentCompartmentDetails: apigatewaysdk.ChangeDeploymentCompartmentDetails{
				CompartmentId: common.String(string(dep.Spec.CompartmentId)),
			},
		}); err != nil {
			return err
		}
	}

	updateDetails, updateNeeded := buildDeploymentUpdateDetails(dep, existing)
	if !updateNeeded {
		return nil
	}

	req := apigatewaysdk.UpdateDeploymentRequest{
		DeploymentId:            common.String(string(targetID)),
		UpdateDeploymentDetails: updateDetails,
	}
	_, err = client.UpdateDeployment(ctx, req)
	return err
}

func buildDeploymentUpdateDetails(dep *apigatewayv1beta1.ApiGatewayDeployment, existing *apigatewaysdk.Deployment) (apigatewaysdk.UpdateDeploymentDetails, bool) {
	updateDetails := apigatewaysdk.UpdateDeploymentDetails{}
	updateNeeded := false

	desiredSpec := buildApiSpecification(dep.Spec.Routes)
	if !reflect.DeepEqual(existing.Specification, desiredSpec) {
		updateDetails.Specification = desiredSpec
		updateNeeded = true
	}
	if dep.Spec.DisplayName != "" && safeGatewayString(existing.DisplayName) != dep.Spec.DisplayName {
		updateDetails.DisplayName = common.String(dep.Spec.DisplayName)
		updateNeeded = true
	}
	if dep.Spec.FreeFormTags != nil && !reflect.DeepEqual(existing.FreeformTags, dep.Spec.FreeFormTags) {
		updateDetails.FreeformTags = dep.Spec.FreeFormTags
		updateNeeded = true
	}
	if dep.Spec.DefinedTags != nil {
		desiredDefinedTags := *util.ConvertToOciDefinedTags(&dep.Spec.DefinedTags)
		if !reflect.DeepEqual(existing.DefinedTags, desiredDefinedTags) {
			updateDetails.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}

	return updateDetails, updateNeeded
}

func validateDeploymentUnsupportedChanges(dep *apigatewayv1beta1.ApiGatewayDeployment, existing *apigatewaysdk.Deployment) error {
	if dep.Spec.GatewayId != "" && safeGatewayString(existing.GatewayId) != "" && safeGatewayString(existing.GatewayId) != string(dep.Spec.GatewayId) {
		return fmt.Errorf("gatewayId cannot be updated in place")
	}
	if dep.Spec.PathPrefix != "" && safeGatewayString(existing.PathPrefix) != "" && safeGatewayString(existing.PathPrefix) != dep.Spec.PathPrefix {
		return fmt.Errorf("pathPrefix cannot be updated in place")
	}
	return nil
}

// DeleteDeployment deletes the API Gateway Deployment for the given OCID.
func (c *DeploymentServiceManager) DeleteDeployment(ctx context.Context, deploymentID shared.OCID) error {
	client, err := c.getDeploymentClientOrCreate()
	if err != nil {
		return err
	}

	req := apigatewaysdk.DeleteDeploymentRequest{
		DeploymentId: common.String(string(deploymentID)),
	}

	_, err = client.DeleteDeployment(ctx, req)
	return err
}

// getDeploymentRetryPolicy returns a retry policy that waits while the deployment is CREATING.
func (c *DeploymentServiceManager) getDeploymentRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(apigatewaysdk.GetDeploymentResponse); ok {
			return resp.LifecycleState == apigatewaysdk.DeploymentLifecycleStateCreating
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(1) * time.Minute
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
