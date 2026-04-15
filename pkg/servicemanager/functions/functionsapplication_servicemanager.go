/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functions

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocifunctions "github.com/oracle/oci-go-sdk/v65/functions"
	functionsv1beta1 "github.com/oracle/oci-service-operator/api/functions/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Compile-time check that FunctionsApplicationServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = (*FunctionsApplicationServiceManager)(nil)

// FunctionsApplicationServiceManager implements OSOKServiceManager for OCI Functions Applications.
type FunctionsApplicationServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	Metrics          *metrics.Metrics

	ociClient FunctionsManagementClientInterface
}

// ApplicationServiceManager keeps the constructor contract aligned with the generated controller wiring.
type ApplicationServiceManager = FunctionsApplicationServiceManager

func NewFunctionsApplicationServiceManagerWithDeps(deps servicemanager.RuntimeDeps) *FunctionsApplicationServiceManager {
	return &FunctionsApplicationServiceManager{
		Provider:         deps.Provider,
		CredentialClient: deps.CredentialClient,
		Scheme:           deps.Scheme,
		Log:              deps.Log,
		Metrics:          deps.Metrics,
	}
}

func NewFunctionsApplicationServiceManager(
	provider common.ConfigurationProvider,
	credClient credhelper.CredentialClient,
	scheme *runtime.Scheme,
	log loggerutil.OSOKLogger,
	metrics *metrics.Metrics,
) *FunctionsApplicationServiceManager {
	return &FunctionsApplicationServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
		Metrics:          metrics,
	}
}

func NewApplicationServiceManagerWithDeps(deps servicemanager.RuntimeDeps) *FunctionsApplicationServiceManager {
	return NewFunctionsApplicationServiceManagerWithDeps(deps)
}

func NewApplicationServiceManager(
	provider common.ConfigurationProvider,
	credClient credhelper.CredentialClient,
	scheme *runtime.Scheme,
	log loggerutil.OSOKLogger,
	metrics *metrics.Metrics,
) *FunctionsApplicationServiceManager {
	return NewFunctionsApplicationServiceManager(provider, credClient, scheme, log, metrics)
}

func (m *FunctionsApplicationServiceManager) WithClient(client FunctionsManagementClientInterface) *FunctionsApplicationServiceManager {
	if client != nil {
		m.ociClient = client
	}
	return m
}

func (m *FunctionsApplicationServiceManager) CreateOrUpdate(
	ctx context.Context,
	obj runtime.Object,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	_ = req

	resource, err := m.convert(obj)
	if err != nil {
		m.Log.ErrorLog(err, "conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	current, response, done, err := m.resolveApplicationForReconcile(ctx, resource)
	if err != nil || done {
		return response, err
	}

	if err := projectFunctionsResponseIntoStatus(resource, current); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return reconcileFunctionsApplicationLifecycle(&resource.Status.OsokStatus, current, m.Log), nil
}

func (m *FunctionsApplicationServiceManager) resolveApplicationForReconcile(
	ctx context.Context,
	resource *functionsv1beta1.Application,
) (*ocifunctions.Application, servicemanager.OSOKResponse, bool, error) {
	trackedID := applicationStatusID(resource)
	if strings.TrimSpace(string(trackedID)) != "" {
		current, err := m.GetApplication(ctx, trackedID, nil)
		if err != nil {
			if !isFunctionsNotFound(err) {
				servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
				m.Log.ErrorLog(err, "error while getting Application from tracked OCID")
				return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
			}
			clearTrackedFunctionsStatus(resource)
		} else {
			updatedCurrent, _, err := m.UpdateApplication(ctx, resource, current)
			if err != nil {
				m.Log.ErrorLog(err, "error while updating Application")
				return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
			}
			return updatedCurrent, servicemanager.OSOKResponse{}, false, nil
		}
	}

	current, err := m.FindApplication(ctx, resource)
	if err != nil {
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		m.Log.ErrorLog(err, "error while listing Applications")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}
	if current == nil {
		createResponse, err := m.CreateApplication(ctx, resource)
		if err != nil {
			applyFunctionsCreateFailure(&resource.Status.OsokStatus, err, m.Log, "Application")
			return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
		}
		return &createResponse.Application, servicemanager.OSOKResponse{}, false, nil
	}

	updatedCurrent, _, err := m.UpdateApplication(ctx, resource, current)
	if err != nil {
		m.Log.ErrorLog(err, "error while updating Application")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}
	return updatedCurrent, servicemanager.OSOKResponse{}, false, nil
}

func (m *FunctionsApplicationServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	resource, err := m.convert(obj)
	if err != nil {
		return false, err
	}

	targetID := applicationStatusID(resource)
	if strings.TrimSpace(string(targetID)) == "" {
		return true, nil
	}

	if err := m.DeleteApplication(ctx, resource, targetID); err != nil {
		if isFunctionsNotFound(err) {
			clearTrackedFunctionsStatus(resource)
			return true, nil
		}
		m.Log.ErrorLog(err, "error while deleting Application")
		return false, err
	}

	current, err := m.GetApplication(ctx, targetID, nil)
	if err != nil {
		if isFunctionsNotFound(err) {
			servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
			clearTrackedFunctionsStatus(resource)
			return true, nil
		}
		servicemanager.RecordErrorOpcRequestID(&resource.Status.OsokStatus, err)
		return false, err
	}
	if err := projectFunctionsResponseIntoStatus(resource, current); err != nil {
		return false, err
	}
	if strings.EqualFold(string(current.LifecycleState), "DELETED") {
		clearTrackedFunctionsStatus(resource)
		return true, nil
	}

	return false, nil
}

func (m *FunctionsApplicationServiceManager) GetCrdStatus(obj runtime.Object) (*shared.OSOKStatus, error) {
	resource, err := m.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (m *FunctionsApplicationServiceManager) convert(obj runtime.Object) (*functionsv1beta1.Application, error) {
	resource, ok := obj.(*functionsv1beta1.Application)
	if !ok {
		return nil, fmt.Errorf("expected *functionsv1beta1.Application, got %T", obj)
	}
	return resource, nil
}
