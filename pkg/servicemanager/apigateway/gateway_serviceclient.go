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

// GatewayClientInterface is the subset of apigateway.GatewayClient methods used by GatewayServiceManager.
type GatewayClientInterface interface {
	CreateGateway(ctx context.Context, request apigatewaysdk.CreateGatewayRequest) (apigatewaysdk.CreateGatewayResponse, error)
	GetGateway(ctx context.Context, request apigatewaysdk.GetGatewayRequest) (apigatewaysdk.GetGatewayResponse, error)
	ListGateways(ctx context.Context, request apigatewaysdk.ListGatewaysRequest) (apigatewaysdk.ListGatewaysResponse, error)
	ChangeGatewayCompartment(ctx context.Context, request apigatewaysdk.ChangeGatewayCompartmentRequest) (apigatewaysdk.ChangeGatewayCompartmentResponse, error)
	UpdateGateway(ctx context.Context, request apigatewaysdk.UpdateGatewayRequest) (apigatewaysdk.UpdateGatewayResponse, error)
	DeleteGateway(ctx context.Context, request apigatewaysdk.DeleteGatewayRequest) (apigatewaysdk.DeleteGatewayResponse, error)
}

// getGatewayClientOrCreate returns the injected client when set, otherwise creates one from the provider.
func (c *GatewayServiceManager) getGatewayClientOrCreate() (GatewayClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return apigatewaysdk.NewGatewayClientWithConfigurationProvider(c.Provider)
}

// CreateGateway calls the OCI API to create a new API Gateway.
func (c *GatewayServiceManager) CreateGateway(ctx context.Context, gw apigatewayv1beta1.ApiGateway) (apigatewaysdk.CreateGatewayResponse, error) {
	client, err := c.getGatewayClientOrCreate()
	if err != nil {
		return apigatewaysdk.CreateGatewayResponse{}, err
	}

	c.Log.DebugLog("Creating ApiGateway", "displayName", gw.Spec.DisplayName)

	details := apigatewaysdk.CreateGatewayDetails{
		CompartmentId: common.String(string(gw.Spec.CompartmentId)),
		EndpointType:  apigatewaysdk.GatewayEndpointTypeEnum(gw.Spec.EndpointType),
		SubnetId:      common.String(string(gw.Spec.SubnetId)),
	}

	if gw.Spec.DisplayName != "" {
		details.DisplayName = common.String(gw.Spec.DisplayName)
	}
	if gw.Spec.CertificateId != "" {
		details.CertificateId = common.String(string(gw.Spec.CertificateId))
	}
	if len(gw.Spec.NetworkSecurityGroupIds) > 0 {
		details.NetworkSecurityGroupIds = gw.Spec.NetworkSecurityGroupIds
	}
	if gw.Spec.FreeFormTags != nil {
		details.FreeformTags = gw.Spec.FreeFormTags
	}
	if gw.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&gw.Spec.DefinedTags)
	}

	req := apigatewaysdk.CreateGatewayRequest{
		CreateGatewayDetails: details,
	}
	return client.CreateGateway(ctx, req)
}

// GetGateway retrieves an API Gateway by OCID.
func (c *GatewayServiceManager) GetGateway(ctx context.Context, gatewayID shared.OCID, retryPolicy *common.RetryPolicy) (*apigatewaysdk.Gateway, error) {
	client, err := c.getGatewayClientOrCreate()
	if err != nil {
		return nil, err
	}

	req := apigatewaysdk.GetGatewayRequest{
		GatewayId: common.String(string(gatewayID)),
	}
	if retryPolicy != nil {
		req.RequestMetadata.RetryPolicy = retryPolicy
	}

	resp, err := client.GetGateway(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Gateway, nil
}

// GetGatewayOcid looks up an existing gateway by display name and compartment.
func (c *GatewayServiceManager) GetGatewayOcid(ctx context.Context, gw apigatewayv1beta1.ApiGateway) (*shared.OCID, error) {
	client, err := c.getGatewayClientOrCreate()
	if err != nil {
		return nil, err
	}

	req := apigatewaysdk.ListGatewaysRequest{
		CompartmentId: common.String(string(gw.Spec.CompartmentId)),
		DisplayName:   common.String(gw.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	resp, err := client.ListGateways(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing ApiGateways")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "CREATING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("ApiGateway %s exists with OCID %s", gw.Spec.DisplayName, *item.Id))
			return (*shared.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("ApiGateway %s does not exist", gw.Spec.DisplayName))
	return nil, nil
}

// UpdateGateway updates an existing API Gateway.
func (c *GatewayServiceManager) UpdateGateway(ctx context.Context, gw *apigatewayv1beta1.ApiGateway) error {
	client, err := c.getGatewayClientOrCreate()
	if err != nil {
		return err
	}

	targetID, err := servicemanager.ResolveResourceID(gw.Status.OsokStatus.Ocid, gw.Spec.ApiGatewayId)
	if err != nil {
		return err
	}

	existing, err := c.GetGateway(ctx, targetID, nil)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&gw.Status.OsokStatus, err)
		return err
	}

	if err := validateGatewayUnsupportedChanges(gw, existing); err != nil {
		return err
	}

	if gw.Spec.CompartmentId != "" &&
		(existing.CompartmentId == nil || *existing.CompartmentId != string(gw.Spec.CompartmentId)) {
		response, err := client.ChangeGatewayCompartment(ctx, apigatewaysdk.ChangeGatewayCompartmentRequest{
			GatewayId: common.String(string(targetID)),
			ChangeGatewayCompartmentDetails: apigatewaysdk.ChangeGatewayCompartmentDetails{
				CompartmentId: common.String(string(gw.Spec.CompartmentId)),
			},
		})
		if err != nil {
			servicemanager.RecordErrorOpcRequestID(&gw.Status.OsokStatus, err)
			return err
		}
		servicemanager.RecordResponseOpcRequestID(&gw.Status.OsokStatus, response)
	}

	updateDetails, updateNeeded := buildGatewayUpdateDetails(gw, existing)
	if !updateNeeded {
		return nil
	}

	req := apigatewaysdk.UpdateGatewayRequest{
		GatewayId:            common.String(string(targetID)),
		UpdateGatewayDetails: updateDetails,
	}
	response, err := client.UpdateGateway(ctx, req)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&gw.Status.OsokStatus, err)
		return err
	}
	servicemanager.RecordResponseOpcRequestID(&gw.Status.OsokStatus, response)
	return err
}

func buildGatewayUpdateDetails(gw *apigatewayv1beta1.ApiGateway, existing *apigatewaysdk.Gateway) (apigatewaysdk.UpdateGatewayDetails, bool) {
	updateDetails := apigatewaysdk.UpdateGatewayDetails{}
	updateNeeded := applyGatewayDisplayNameUpdate(&updateDetails, gw, existing)
	if applyGatewayNetworkSecurityGroupUpdate(&updateDetails, gw, existing) {
		updateNeeded = true
	}
	if applyGatewayCertificateUpdate(&updateDetails, gw, existing) {
		updateNeeded = true
	}
	if applyGatewayFreeformTagUpdate(&updateDetails, gw, existing) {
		updateNeeded = true
	}
	if applyGatewayDefinedTagUpdate(&updateDetails, gw, existing) {
		updateNeeded = true
	}
	return updateDetails, updateNeeded
}

func applyGatewayDisplayNameUpdate(updateDetails *apigatewaysdk.UpdateGatewayDetails, gw *apigatewayv1beta1.ApiGateway, existing *apigatewaysdk.Gateway) bool {
	if gw.Spec.DisplayName == "" || safeGatewayString(existing.DisplayName) == gw.Spec.DisplayName {
		return false
	}
	updateDetails.DisplayName = common.String(gw.Spec.DisplayName)
	return true
}

func applyGatewayNetworkSecurityGroupUpdate(updateDetails *apigatewaysdk.UpdateGatewayDetails, gw *apigatewayv1beta1.ApiGateway, existing *apigatewaysdk.Gateway) bool {
	if len(gw.Spec.NetworkSecurityGroupIds) == 0 || reflect.DeepEqual(existing.NetworkSecurityGroupIds, gw.Spec.NetworkSecurityGroupIds) {
		return false
	}
	updateDetails.NetworkSecurityGroupIds = gw.Spec.NetworkSecurityGroupIds
	return true
}

func applyGatewayCertificateUpdate(updateDetails *apigatewaysdk.UpdateGatewayDetails, gw *apigatewayv1beta1.ApiGateway, existing *apigatewaysdk.Gateway) bool {
	if gw.Spec.CertificateId == "" || safeGatewayString(existing.CertificateId) == string(gw.Spec.CertificateId) {
		return false
	}
	updateDetails.CertificateId = common.String(string(gw.Spec.CertificateId))
	return true
}

func applyGatewayFreeformTagUpdate(updateDetails *apigatewaysdk.UpdateGatewayDetails, gw *apigatewayv1beta1.ApiGateway, existing *apigatewaysdk.Gateway) bool {
	if gw.Spec.FreeFormTags == nil || reflect.DeepEqual(existing.FreeformTags, gw.Spec.FreeFormTags) {
		return false
	}
	updateDetails.FreeformTags = gw.Spec.FreeFormTags
	return true
}

func applyGatewayDefinedTagUpdate(updateDetails *apigatewaysdk.UpdateGatewayDetails, gw *apigatewayv1beta1.ApiGateway, existing *apigatewaysdk.Gateway) bool {
	if gw.Spec.DefinedTags == nil {
		return false
	}
	desiredDefinedTags := *util.ConvertToOciDefinedTags(&gw.Spec.DefinedTags)
	if reflect.DeepEqual(existing.DefinedTags, desiredDefinedTags) {
		return false
	}
	updateDetails.DefinedTags = desiredDefinedTags
	return true
}

func validateGatewayUnsupportedChanges(gw *apigatewayv1beta1.ApiGateway, existing *apigatewaysdk.Gateway) error {
	if gw.Spec.EndpointType != "" && existing.EndpointType != "" && string(existing.EndpointType) != gw.Spec.EndpointType {
		return fmt.Errorf("endpointType cannot be updated in place")
	}
	if gw.Spec.SubnetId != "" && safeGatewayString(existing.SubnetId) != "" && safeGatewayString(existing.SubnetId) != string(gw.Spec.SubnetId) {
		return fmt.Errorf("subnetId cannot be updated in place")
	}
	return nil
}

// DeleteGateway deletes the API Gateway for the given OCID.
func (c *GatewayServiceManager) DeleteGateway(ctx context.Context, gw *apigatewayv1beta1.ApiGateway, gatewayID shared.OCID) error {
	client, err := c.getGatewayClientOrCreate()
	if err != nil {
		return err
	}

	req := apigatewaysdk.DeleteGatewayRequest{
		GatewayId: common.String(string(gatewayID)),
	}

	response, err := client.DeleteGateway(ctx, req)
	if gw != nil {
		if err != nil {
			servicemanager.RecordErrorOpcRequestID(&gw.Status.OsokStatus, err)
		} else {
			servicemanager.RecordResponseOpcRequestID(&gw.Status.OsokStatus, response)
		}
	}
	return err
}

// getGatewayRetryPolicy returns a retry policy that waits while the gateway is CREATING.
func (c *GatewayServiceManager) getGatewayRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(apigatewaysdk.GetGatewayResponse); ok {
			return resp.LifecycleState == apigatewaysdk.GatewayLifecycleStateCreating
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(1) * time.Minute
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
