/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functions

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocifunctions "github.com/oracle/oci-go-sdk/v65/functions"
	functionsv1beta1 "github.com/oracle/oci-service-operator/api/functions/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const functionsFunctionSecretOwnerUIDLabel = "functions.oracle.com/function-uid"

// Compile-time check that FunctionsFunctionServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = (*FunctionsFunctionServiceManager)(nil)

// FunctionsFunctionServiceManager implements OSOKServiceManager for OCI Functions Functions.
type FunctionsFunctionServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	Metrics          *metrics.Metrics

	ociClient FunctionsManagementClientInterface
}

// FunctionServiceManager keeps the constructor contract aligned with the generated controller wiring.
type FunctionServiceManager = FunctionsFunctionServiceManager

func NewFunctionsFunctionServiceManagerWithDeps(deps servicemanager.RuntimeDeps) *FunctionsFunctionServiceManager {
	return &FunctionsFunctionServiceManager{
		Provider:         deps.Provider,
		CredentialClient: deps.CredentialClient,
		Scheme:           deps.Scheme,
		Log:              deps.Log,
		Metrics:          deps.Metrics,
	}
}

func NewFunctionsFunctionServiceManager(
	provider common.ConfigurationProvider,
	credClient credhelper.CredentialClient,
	scheme *runtime.Scheme,
	log loggerutil.OSOKLogger,
	metrics *metrics.Metrics,
) *FunctionsFunctionServiceManager {
	return &FunctionsFunctionServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
		Metrics:          metrics,
	}
}

func NewFunctionServiceManagerWithDeps(deps servicemanager.RuntimeDeps) *FunctionsFunctionServiceManager {
	return NewFunctionsFunctionServiceManagerWithDeps(deps)
}

func NewFunctionServiceManager(
	provider common.ConfigurationProvider,
	credClient credhelper.CredentialClient,
	scheme *runtime.Scheme,
	log loggerutil.OSOKLogger,
	metrics *metrics.Metrics,
) *FunctionsFunctionServiceManager {
	return NewFunctionsFunctionServiceManager(provider, credClient, scheme, log, metrics)
}

func (m *FunctionsFunctionServiceManager) WithClient(client FunctionsManagementClientInterface) *FunctionsFunctionServiceManager {
	if client != nil {
		m.ociClient = client
	}
	return m
}

func (m *FunctionsFunctionServiceManager) CreateOrUpdate(
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

	current, response, done, err := m.resolveFunctionForReconcile(ctx, resource)
	if err != nil || done {
		return response, err
	}

	if err := projectFunctionsResponseIntoStatus(resource, current); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	response = reconcileFunctionsFunctionLifecycle(&resource.Status.OsokStatus, current, m.Log)
	if response.IsSuccessful && strings.TrimSpace(safeFunctionsString(current.InvokeEndpoint)) != "" {
		if err := m.syncFunctionSecret(ctx, resource, current); err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	return response, nil
}

func (m *FunctionsFunctionServiceManager) resolveFunctionForReconcile(
	ctx context.Context,
	resource *functionsv1beta1.Function,
) (*ocifunctions.Function, servicemanager.OSOKResponse, bool, error) {
	trackedID := functionStatusID(resource)
	if strings.TrimSpace(string(trackedID)) != "" {
		current, err := m.GetFunction(ctx, trackedID, nil)
		if err != nil {
			if !isFunctionsNotFound(err) {
				m.Log.ErrorLog(err, "error while getting Function from tracked OCID")
				return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
			}
			clearTrackedFunctionsStatus(resource)
		} else {
			updatedCurrent, _, err := m.UpdateFunction(ctx, resource, current)
			if err != nil {
				m.Log.ErrorLog(err, "error while updating Function")
				return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
			}
			return updatedCurrent, servicemanager.OSOKResponse{}, false, nil
		}
	}

	current, err := m.FindFunction(ctx, resource)
	if err != nil {
		m.Log.ErrorLog(err, "error while listing Functions")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}
	if current == nil {
		createResponse, err := m.CreateFunction(ctx, resource)
		if err != nil {
			applyFunctionsCreateFailure(&resource.Status.OsokStatus, err, m.Log, "Function")
			return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
		}
		return &createResponse.Function, servicemanager.OSOKResponse{}, false, nil
	}

	updatedCurrent, _, err := m.UpdateFunction(ctx, resource, current)
	if err != nil {
		m.Log.ErrorLog(err, "error while updating Function")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}
	return updatedCurrent, servicemanager.OSOKResponse{}, false, nil
}

func (m *FunctionsFunctionServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	resource, err := m.convert(obj)
	if err != nil {
		return false, err
	}

	targetID := functionStatusID(resource)
	if strings.TrimSpace(string(targetID)) == "" {
		if err := m.deleteFunctionSecret(ctx, resource); err != nil {
			return false, err
		}
		return true, nil
	}

	if err := m.DeleteFunction(ctx, targetID); err != nil {
		if isFunctionsNotFound(err) {
			clearTrackedFunctionsStatus(resource)
			if err := m.deleteFunctionSecret(ctx, resource); err != nil {
				return false, err
			}
			return true, nil
		}
		m.Log.ErrorLog(err, "error while deleting Function")
		return false, err
	}

	current, err := m.GetFunction(ctx, targetID, nil)
	if err != nil {
		if isFunctionsNotFound(err) {
			clearTrackedFunctionsStatus(resource)
			if err := m.deleteFunctionSecret(ctx, resource); err != nil {
				return false, err
			}
			return true, nil
		}
		return false, err
	}
	if err := projectFunctionsResponseIntoStatus(resource, current); err != nil {
		return false, err
	}
	if strings.EqualFold(string(current.LifecycleState), "DELETED") {
		clearTrackedFunctionsStatus(resource)
		if err := m.deleteFunctionSecret(ctx, resource); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func (m *FunctionsFunctionServiceManager) GetCrdStatus(obj runtime.Object) (*shared.OSOKStatus, error) {
	resource, err := m.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (m *FunctionsFunctionServiceManager) convert(obj runtime.Object) (*functionsv1beta1.Function, error) {
	resource, ok := obj.(*functionsv1beta1.Function)
	if !ok {
		return nil, fmt.Errorf("expected *functionsv1beta1.Function, got %T", obj)
	}
	return resource, nil
}

func (m *FunctionsFunctionServiceManager) syncFunctionSecret(
	ctx context.Context,
	resource *functionsv1beta1.Function,
	current *ocifunctions.Function,
) error {
	if m.CredentialClient == nil {
		return fmt.Errorf("function endpoint secret credential client is not configured")
	}

	ownerLabels, err := functionSecretLabels(resource)
	if err != nil {
		return err
	}
	desiredData, err := functionCredentialData(current)
	if err != nil {
		return err
	}

	recordReader, ok := m.CredentialClient.(credhelper.SecretRecordReader)
	if !ok {
		_, err := m.CredentialClient.CreateSecret(ctx, resource.Name, resource.Namespace, ownerLabels, desiredData)
		if apierrors.IsAlreadyExists(err) {
			_, err = m.CredentialClient.UpdateSecret(ctx, resource.Name, resource.Namespace, ownerLabels, desiredData)
		}
		return err
	}

	guardedMutator, ok := m.CredentialClient.(credhelper.GuardedSecretMutator)
	if !ok {
		return fmt.Errorf("function endpoint secret ownership checks require guarded secret mutations")
	}

	currentRecord, err := recordReader.GetSecretRecord(ctx, resource.Name, resource.Namespace)
	if err == nil {
		return m.syncExistingFunctionSecret(ctx, resource, currentRecord, ownerLabels, desiredData, guardedMutator)
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	_, err = m.CredentialClient.CreateSecret(ctx, resource.Name, resource.Namespace, ownerLabels, desiredData)
	switch {
	case err == nil:
		return nil
	case apierrors.IsAlreadyExists(err):
		currentRecord, rereadErr := recordReader.GetSecretRecord(ctx, resource.Name, resource.Namespace)
		if rereadErr != nil {
			return rereadErr
		}
		return m.syncExistingFunctionSecret(ctx, resource, currentRecord, ownerLabels, desiredData, guardedMutator)
	default:
		return err
	}
}

func (m *FunctionsFunctionServiceManager) syncExistingFunctionSecret(
	ctx context.Context,
	resource *functionsv1beta1.Function,
	currentRecord credhelper.SecretRecord,
	ownerLabels map[string]string,
	desiredData map[string][]byte,
	guardedMutator credhelper.GuardedSecretMutator,
) error {
	owned, err := functionOwnsSecret(resource, currentRecord.Labels)
	if err != nil {
		return err
	}
	if !owned {
		return fmt.Errorf(
			"function endpoint secret %s/%s is not owned by Function UID %q",
			resource.Namespace,
			resource.Name,
			resource.UID,
		)
	}
	if reflect.DeepEqual(currentRecord.Data, desiredData) {
		return nil
	}

	_, err = guardedMutator.UpdateSecretIfCurrent(ctx, resource.Name, resource.Namespace, currentRecord, ownerLabels, desiredData)
	return err
}

func (m *FunctionsFunctionServiceManager) deleteFunctionSecret(
	ctx context.Context,
	resource *functionsv1beta1.Function,
) error {
	if m.CredentialClient == nil {
		return nil
	}

	recordReader, ok := m.CredentialClient.(credhelper.SecretRecordReader)
	if !ok {
		_, err := m.CredentialClient.DeleteSecret(ctx, resource.Name, resource.Namespace)
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	currentRecord, err := recordReader.GetSecretRecord(ctx, resource.Name, resource.Namespace)
	switch {
	case apierrors.IsNotFound(err):
		return nil
	case err != nil:
		return err
	}

	owned, err := functionOwnsSecret(resource, currentRecord.Labels)
	if err != nil {
		return err
	}
	if !owned {
		return nil
	}

	guardedMutator, ok := m.CredentialClient.(credhelper.GuardedSecretMutator)
	if ok {
		_, err = guardedMutator.DeleteSecretIfCurrent(ctx, resource.Name, resource.Namespace, currentRecord)
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	_, err = m.CredentialClient.DeleteSecret(ctx, resource.Name, resource.Namespace)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func functionSecretLabels(resource *functionsv1beta1.Function) (map[string]string, error) {
	if resource == nil {
		return nil, fmt.Errorf("function endpoint secret requires a Function resource")
	}
	if strings.TrimSpace(string(resource.UID)) == "" {
		return nil, fmt.Errorf("function endpoint secret requires a Function UID")
	}
	return map[string]string{
		functionsFunctionSecretOwnerUIDLabel: string(resource.UID),
	}, nil
}

func functionOwnsSecret(resource *functionsv1beta1.Function, labels map[string]string) (bool, error) {
	if resource == nil {
		return false, fmt.Errorf("function endpoint secret ownership check requires a Function resource")
	}
	ownerUID := strings.TrimSpace(string(resource.UID))
	if ownerUID == "" {
		return false, fmt.Errorf("function endpoint secret ownership check requires a Function UID")
	}
	return strings.TrimSpace(labels[functionsFunctionSecretOwnerUIDLabel]) == ownerUID, nil
}

func functionCredentialData(current *ocifunctions.Function) (map[string][]byte, error) {
	if current == nil {
		return nil, fmt.Errorf("function endpoint secret requires a Functions Function")
	}

	functionID := strings.TrimSpace(safeFunctionsString(current.Id))
	if functionID == "" {
		return nil, fmt.Errorf("function endpoint secret requires a function id")
	}

	invokeEndpoint := strings.TrimSpace(safeFunctionsString(current.InvokeEndpoint))
	if invokeEndpoint == "" {
		return nil, fmt.Errorf("function endpoint secret requires an invoke endpoint")
	}

	return map[string][]byte{
		"functionId":     []byte(functionID),
		"invokeEndpoint": []byte(invokeEndpoint),
	}, nil
}
