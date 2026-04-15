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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Compile-time check that DeploymentServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &DeploymentServiceManager{}

// DeploymentServiceManager implements OSOKServiceManager for OCI API Gateway Deployments.
type DeploymentServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	Metrics          *metrics.Metrics
	ociClient        DeploymentClientInterface
}

func NewDeploymentServiceManagerWithDeps(deps servicemanager.RuntimeDeps) *DeploymentServiceManager {
	return &DeploymentServiceManager{
		Provider:         deps.Provider,
		CredentialClient: deps.CredentialClient,
		Scheme:           deps.Scheme,
		Log:              deps.Log,
		Metrics:          deps.Metrics,
	}
}

// NewDeploymentServiceManager creates a new DeploymentServiceManager.
func NewDeploymentServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *DeploymentServiceManager {
	return NewDeploymentServiceManagerWithDeps(servicemanager.RuntimeDeps{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	})
}

// CreateOrUpdate reconciles the ApiGatewayDeployment resource against OCI.
func (c *DeploymentServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	dep, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	depInstance, err := c.resolveDeploymentInstance(ctx, dep)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&dep.Status.OsokStatus, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	if depInstance.Id != nil {
		dep.Status.OsokStatus.Ocid = shared.OCID(*depInstance.Id)
	}
	servicemanager.SetCreatedAtIfUnset(&dep.Status.OsokStatus)

	return reconcileDeploymentLifecycle(&dep.Status.OsokStatus, depInstance, c.Log), nil
}

// Delete handles deletion of the API Gateway Deployment called by the finalizer.
func (c *DeploymentServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	dep, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	targetID, err := servicemanager.ResolveResourceID(dep.Status.OsokStatus.Ocid, dep.Spec.DeploymentId)
	if err != nil {
		c.Log.InfoLog("ApiGatewayDeployment has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting ApiGatewayDeployment %s", targetID))
	if err := c.DeleteDeployment(ctx, dep, targetID); err != nil {
		if isDeploymentNotFound(err) {
			return true, nil
		}
		c.Log.ErrorLog(err, "Error while deleting ApiGatewayDeployment")
		return false, err
	}

	depInstance, err := c.GetDeployment(ctx, targetID, nil)
	if err != nil {
		if isDeploymentNotFound(err) {
			servicemanager.RecordErrorOpcRequestID(&dep.Status.OsokStatus, err)
			return true, nil
		}
		servicemanager.RecordErrorOpcRequestID(&dep.Status.OsokStatus, err)
		c.Log.ErrorLog(err, "Error while checking ApiGatewayDeployment deletion")
		return false, err
	}

	if depInstance.LifecycleState == apigatewaysdk.DeploymentLifecycleStateDeleted {
		return true, nil
	}
	return false, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *DeploymentServiceManager) GetCrdStatus(obj runtime.Object) (*shared.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *DeploymentServiceManager) convert(obj runtime.Object) (*apigatewayv1beta1.ApiGatewayDeployment, error) {
	dep, ok := obj.(*apigatewayv1beta1.ApiGatewayDeployment)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for ApiGatewayDeployment")
	}
	return dep, nil
}

func isDeploymentNotFound(err error) bool {
	if err == nil {
		return false
	}
	serviceErr, ok := common.IsServiceError(err)
	return ok && serviceErr.GetHTTPStatusCode() == 404
}

func (c *DeploymentServiceManager) resolveDeploymentInstance(ctx context.Context,
	dep *apigatewayv1beta1.ApiGatewayDeployment) (*apigatewaysdk.Deployment, error) {
	if strings.TrimSpace(string(dep.Spec.DeploymentId)) != "" {
		return c.bindDeployment(ctx, dep)
	}
	return c.lookupOrCreateDeployment(ctx, dep)
}

func (c *DeploymentServiceManager) lookupOrCreateDeployment(ctx context.Context,
	dep *apigatewayv1beta1.ApiGatewayDeployment) (*apigatewaysdk.Deployment, error) {
	depOcid, err := c.GetDeploymentOcid(ctx, *dep)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&dep.Status.OsokStatus, err)
		return nil, err
	}
	if depOcid == nil {
		return c.createDeploymentInstance(ctx, dep)
	}
	return c.updateResolvedDeployment(ctx, dep, *depOcid)
}

func (c *DeploymentServiceManager) bindDeployment(ctx context.Context,
	dep *apigatewayv1beta1.ApiGatewayDeployment) (*apigatewaysdk.Deployment, error) {
	depInstance, err := c.GetDeployment(ctx, dep.Spec.DeploymentId, nil)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&dep.Status.OsokStatus, err)
		c.Log.ErrorLog(err, "Error while getting existing ApiGatewayDeployment")
		return nil, err
	}
	dep.Status.OsokStatus.Ocid = dep.Spec.DeploymentId
	if err := c.UpdateDeployment(ctx, dep); err != nil {
		c.Log.ErrorLog(err, "Error while updating ApiGatewayDeployment")
		return nil, err
	}
	return depInstance, nil
}

func (c *DeploymentServiceManager) createDeploymentInstance(ctx context.Context,
	dep *apigatewayv1beta1.ApiGatewayDeployment) (*apigatewaysdk.Deployment, error) {
	resp, err := c.CreateDeployment(ctx, *dep)
	if err != nil {
		applyGatewayCreateFailure(&dep.Status.OsokStatus, err, c.Log, "ApiGatewayDeployment")
		return nil, err
	}
	servicemanager.RecordResponseOpcRequestID(&dep.Status.OsokStatus, resp)

	c.Log.InfoLog(fmt.Sprintf("ApiGatewayDeployment %s is Provisioning", dep.Spec.DisplayName))
	setGatewayProvisioning(&dep.Status.OsokStatus, "ApiGatewayDeployment", dep.Spec.DisplayName, shared.OCID(*resp.Id), c.Log)
	retryPolicy := c.getDeploymentRetryPolicy(30)
	depInstance, err := c.GetDeployment(ctx, shared.OCID(*resp.Id), &retryPolicy)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&dep.Status.OsokStatus, err)
		c.Log.ErrorLog(err, "Error while getting ApiGatewayDeployment after create")
		return nil, err
	}
	return depInstance, nil
}

func (c *DeploymentServiceManager) updateResolvedDeployment(ctx context.Context,
	dep *apigatewayv1beta1.ApiGatewayDeployment, depOcid shared.OCID) (*apigatewaysdk.Deployment, error) {
	c.Log.InfoLog(fmt.Sprintf("Getting existing ApiGatewayDeployment %s", depOcid))
	depInstance, err := c.GetDeployment(ctx, depOcid, nil)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&dep.Status.OsokStatus, err)
		c.Log.ErrorLog(err, "Error while getting ApiGatewayDeployment by OCID")
		return nil, err
	}
	dep.Status.OsokStatus.Ocid = depOcid
	if err := c.UpdateDeployment(ctx, dep); err != nil {
		c.Log.ErrorLog(err, "Error while updating ApiGatewayDeployment")
		return nil, err
	}
	return depInstance, nil
}
