/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apigateway

import (
	"context"
	"fmt"
	"strings"

	apigatewaysdk "github.com/oracle/oci-go-sdk/v65/apigateway"
	"github.com/oracle/oci-go-sdk/v65/common"
	apigatewayv1beta1 "github.com/oracle/oci-service-operator/api/apigateway/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Compile-time check that GatewayServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &GatewayServiceManager{}

// GatewayServiceManager implements OSOKServiceManager for OCI API Gateway.
type GatewayServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	Metrics          *metrics.Metrics
	ociClient        GatewayClientInterface
}

func NewGatewayServiceManagerWithDeps(deps servicemanager.RuntimeDeps) *GatewayServiceManager {
	return &GatewayServiceManager{
		Provider:         deps.Provider,
		CredentialClient: deps.CredentialClient,
		Scheme:           deps.Scheme,
		Log:              deps.Log,
		Metrics:          deps.Metrics,
	}
}

// NewGatewayServiceManager creates a new GatewayServiceManager.
func NewGatewayServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *GatewayServiceManager {
	return NewGatewayServiceManagerWithDeps(servicemanager.RuntimeDeps{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	})
}

// CreateOrUpdate reconciles the ApiGateway resource against OCI.
func (c *GatewayServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	gw, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	gwInstance, err := c.resolveGatewayInstance(ctx, gw)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&gw.Status.OsokStatus, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	if gwInstance.Id != nil {
		gw.Status.OsokStatus.Ocid = shared.OCID(*gwInstance.Id)
	}
	servicemanager.SetCreatedAtIfUnset(&gw.Status.OsokStatus)

	response := reconcileGatewayLifecycle(&gw.Status.OsokStatus, gwInstance, c.Log)
	if !response.IsSuccessful {
		return response, nil
	}

	if _, err := c.addToSecret(ctx, gw.Namespace, gw.Name, *gwInstance); err != nil && !apierrors.IsAlreadyExists(err) {
		c.Log.InfoLog("ApiGateway secret creation failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	return response, nil
}

// Delete handles deletion of the API Gateway called by the finalizer.
func (c *GatewayServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	gw, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	targetID, err := servicemanager.ResolveResourceID(gw.Status.OsokStatus.Ocid, gw.Spec.ApiGatewayId)
	if err != nil {
		c.Log.InfoLog("ApiGateway has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting ApiGateway %s", targetID))
	if err := c.DeleteGateway(ctx, gw, targetID); err != nil {
		if isGatewayNotFound(err) {
			return true, nil
		}
		c.Log.ErrorLog(err, "Error while deleting ApiGateway")
		return false, err
	}

	gwInstance, err := c.GetGateway(ctx, targetID, nil)
	if err != nil {
		if isGatewayNotFound(err) {
			servicemanager.RecordErrorOpcRequestID(&gw.Status.OsokStatus, err)
			return true, nil
		}
		servicemanager.RecordErrorOpcRequestID(&gw.Status.OsokStatus, err)
		c.Log.ErrorLog(err, "Error while checking ApiGateway deletion")
		return false, err
	}

	if gwInstance.LifecycleState == apigatewaysdk.GatewayLifecycleStateDeleted {
		return true, nil
	}
	return false, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *GatewayServiceManager) GetCrdStatus(obj runtime.Object) (*shared.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *GatewayServiceManager) convert(obj runtime.Object) (*apigatewayv1beta1.ApiGateway, error) {
	gw, ok := obj.(*apigatewayv1beta1.ApiGateway)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for ApiGateway")
	}
	return gw, nil
}

func isGatewayNotFound(err error) bool {
	if err == nil {
		return false
	}
	serviceErr, ok := common.IsServiceError(err)
	return ok && serviceErr.GetHTTPStatusCode() == 404
}

func (c *GatewayServiceManager) resolveGatewayInstance(ctx context.Context,
	gw *apigatewayv1beta1.ApiGateway) (*apigatewaysdk.Gateway, error) {
	if strings.TrimSpace(string(gw.Spec.ApiGatewayId)) != "" {
		return c.bindGateway(ctx, gw)
	}
	return c.lookupOrCreateGateway(ctx, gw)
}

func (c *GatewayServiceManager) lookupOrCreateGateway(ctx context.Context,
	gw *apigatewayv1beta1.ApiGateway) (*apigatewaysdk.Gateway, error) {
	gwOcid, err := c.GetGatewayOcid(ctx, *gw)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&gw.Status.OsokStatus, err)
		return nil, err
	}
	if gwOcid == nil {
		return c.createGatewayInstance(ctx, gw)
	}
	return c.updateResolvedGateway(ctx, gw, *gwOcid)
}

func (c *GatewayServiceManager) bindGateway(ctx context.Context,
	gw *apigatewayv1beta1.ApiGateway) (*apigatewaysdk.Gateway, error) {
	gwInstance, err := c.GetGateway(ctx, gw.Spec.ApiGatewayId, nil)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&gw.Status.OsokStatus, err)
		c.Log.ErrorLog(err, "Error while getting existing ApiGateway")
		return nil, err
	}
	gw.Status.OsokStatus.Ocid = gw.Spec.ApiGatewayId
	if err := c.UpdateGateway(ctx, gw); err != nil {
		c.Log.ErrorLog(err, "Error while updating ApiGateway")
		return nil, err
	}
	return gwInstance, nil
}

func (c *GatewayServiceManager) createGatewayInstance(ctx context.Context, gw *apigatewayv1beta1.ApiGateway) (*apigatewaysdk.Gateway, error) {
	resp, err := c.CreateGateway(ctx, *gw)
	if err != nil {
		applyGatewayCreateFailure(&gw.Status.OsokStatus, err, c.Log, "ApiGateway")
		return nil, err
	}
	servicemanager.RecordResponseOpcRequestID(&gw.Status.OsokStatus, resp)

	c.Log.InfoLog(fmt.Sprintf("ApiGateway %s is Provisioning", gw.Spec.DisplayName))
	setGatewayProvisioning(&gw.Status.OsokStatus, "ApiGateway", gw.Spec.DisplayName, shared.OCID(*resp.Id), c.Log)
	retryPolicy := c.getGatewayRetryPolicy(30)
	gwInstance, err := c.GetGateway(ctx, shared.OCID(*resp.Id), &retryPolicy)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&gw.Status.OsokStatus, err)
		c.Log.ErrorLog(err, "Error while getting ApiGateway after create")
		return nil, err
	}
	return gwInstance, nil
}

func (c *GatewayServiceManager) updateResolvedGateway(ctx context.Context,
	gw *apigatewayv1beta1.ApiGateway, gwOcid shared.OCID) (*apigatewaysdk.Gateway, error) {
	c.Log.InfoLog(fmt.Sprintf("Getting existing ApiGateway %s", gwOcid))
	gwInstance, err := c.GetGateway(ctx, gwOcid, nil)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&gw.Status.OsokStatus, err)
		c.Log.ErrorLog(err, "Error while getting ApiGateway by OCID")
		return nil, err
	}
	gw.Status.OsokStatus.Ocid = gwOcid
	if err := c.UpdateGateway(ctx, gw); err != nil {
		c.Log.ErrorLog(err, "Error while updating ApiGateway")
		return nil, err
	}
	return gwInstance, nil
}
