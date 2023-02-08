/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package manager

import (
	"context"
	"errors"

	"github.com/oracle/oci-go-sdk/v65/common"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	meshConversions "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Resource interface {
	GetResource(ctx context.Context, object client.Object, details *ResourceDetails) error
	CreateResource(ctx context.Context, object client.Object, details *ResourceDetails) (bool, error)
	UpdateResource(ctx context.Context, object client.Object, details *ResourceDetails) error
	DeleteResource(ctx context.Context, object client.Object) error
	ChangeCompartment(ctx context.Context, object client.Object, details *ResourceDetails) error
}

type Finalizer interface {
	GetFinalizer() string
	Finalize(ctx context.Context, object client.Object) error
}

type Status interface {
	GetServiceMeshStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error)
	GetConditionStatus(details *ResourceDetails) metav1.ConditionStatus
	UpdateStatus(object client.Object, details *ResourceDetails) (bool, error)
	GetTimeUpdated(details *ResourceDetails) *common.SDKTime
}

type Verify interface {
	VerifyEntityType(object runtime.Object) (client.Object, error)
	VerifyResourceStatus(details *ResourceDetails) (bool, error)
}

type Dependencies interface {
	ResolveDependencies(ctx context.Context, object client.Object, details *ResourceDetails) error
}

type SdkOperations interface {
	BuildSdk(object client.Object, details *ResourceDetails) error
	HasCompartmentIdChanged(object client.Object, details *ResourceDetails) (bool, error)
	GetLifecycleState(details *ResourceDetails) string
	GetMessage(details *ResourceDetails) string
	HasSdk(details *ResourceDetails) bool
}

type CustomResourceHandler interface {
	Resource
	Finalizer
	Status
	Verify
	Dependencies
	SdkOperations
}

type MeshDetails struct {
	SdkMesh      *sdk.Mesh
	BuildSdkMesh *sdk.Mesh
}

type VirtualServiceDetails struct {
	MeshId     *api.OCID
	SdkVs      *sdk.VirtualService
	BuildSdkVs *sdk.VirtualService
}

type AccessPolicyDetails struct {
	Dependencies *meshConversions.AccessPolicyDependencies
	SdkAp        *sdk.AccessPolicy
	BuildSdkAp   *sdk.AccessPolicy
}

type VirtualDeploymentDetails struct {
	VsRef      *meshCommons.ResourceRef
	SdkVd      *sdk.VirtualDeployment
	BuildSdkVd *sdk.VirtualDeployment
}

type VirtualServiceRouteTableDetails struct {
	SdkVsrt      *sdk.VirtualServiceRouteTable
	BuildSdkVsrt *sdk.VirtualServiceRouteTable
	Dependencies *meshConversions.VSRTDependencies
}

type IngressGatewayDetails struct {
	MeshId     *api.OCID
	SdkIg      *sdk.IngressGateway
	BuildSdkIg *sdk.IngressGateway
}

type IngressGatewayRouteTableDetails struct {
	SdkIgrt      *sdk.IngressGatewayRouteTable
	BuildSdkIgrt *sdk.IngressGatewayRouteTable
	Dependencies *meshConversions.IGRTDependencies
}

type ResourceDetails struct {
	MeshDetails   MeshDetails
	VsDetails     VirtualServiceDetails
	VdDetails     VirtualDeploymentDetails
	VsrtDetails   VirtualServiceRouteTableDetails
	ApDetails     AccessPolicyDetails
	IgDetails     IngressGatewayDetails
	IgrtDetails   IngressGatewayRouteTableDetails
	OpcRetryToken *string
}

type ServiceMeshServiceManager struct {
	client  client.Client
	log     loggerutil.OSOKLogger
	handler CustomResourceHandler
}

func NewServiceMeshServiceManager(client client.Client, log loggerutil.OSOKLogger, handler CustomResourceHandler) *ServiceMeshServiceManager {
	return &ServiceMeshServiceManager{
		client:  client,
		log:     log,
		handler: handler,
	}
}

func (c *ServiceMeshServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	object, err := c.handler.VerifyEntityType(obj)
	if err != nil {
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}

	resourceDetails := &ResourceDetails{}
	resolveDependenciesErr := c.handler.ResolveDependencies(ctx, object, resourceDetails)
	c.UpdateServiceMeshDependenciesActiveStatus(ctx, object, resolveDependenciesErr)
	if resolveDependenciesErr != nil {
		return meshErrors.GetOsokResponseByHandlingReconcileError(resolveDependenciesErr)
	}
	c.log.InfoLogWithFixedMessage(ctx, "dependencies resolved")
	getResourceErr := c.handler.GetResource(ctx, object, resourceDetails)
	if getResourceErr != nil {
		_ = c.UpdateServiceMeshConfiguredStatus(ctx, object, getResourceErr)
		return meshErrors.GetOsokResponseByHandlingReconcileError(getResourceErr)
	}

	// Verify
	// For states like Creating, Updating, Deleting commons.UnknownStatus error is thrown
	// Don't proceed with the reconcile for states like Deleted and Failed
	validResource, verifyResourceErr := c.handler.VerifyResourceStatus(resourceDetails)
	if !validResource {
		status := c.handler.GetConditionStatus(resourceDetails)
		message := c.handler.GetMessage(resourceDetails)
		updateErr := c.UpdateServiceMeshCondition(ctx, object, status, string(meshCommons.LifecycleStateChanged), message, servicemeshapi.ServiceMeshActive)
		if updateErr != nil {
			return meshErrors.GetOsokResponseByHandlingReconcileError(updateErr)
		}
		return meshErrors.GetOsokResponseByHandlingReconcileError(verifyResourceErr)
	}

	// Build object
	buildSdkErr := c.handler.BuildSdk(object, resourceDetails)
	if buildSdkErr != nil {
		return meshErrors.GetOsokResponseByHandlingReconcileError(buildSdkErr)
	}

	// Reuses the opcRetryToken if the previous request fails, otherwise create a new opcRetryToken
	opcRetryToken, err := c.getOpcRetryToken(object)
	if err != nil {
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}
	resourceDetails.OpcRetryToken = opcRetryToken
	if !c.handler.HasSdk(resourceDetails) {
		c.UpdateOpcRetryToken(ctx, object, resourceDetails.OpcRetryToken)
	}

	resourceCreated, resourceCreationErr := c.handler.CreateResource(ctx, object, resourceDetails)
	hasChanged := false
	hasResourceUpdatedInCp := c.hasResourceUpdatedInCp(object, resourceDetails)
	if !resourceCreated {
		_ = c.UpdateServiceMeshConfiguredStatus(ctx, object, resourceCreationErr)
		if resourceCreationErr != nil {
			c.log.InfoLogWithFixedMessage(ctx, "cp error while creating resource", "error", resourceCreationErr)
			if !meshErrors.IsNetworkErrorOrInternalError(resourceCreationErr) {
				// Clears the opcRetryToken if the request didn't fail for network or internal error
				_ = c.UpdateOpcRetryToken(ctx, object, nil)
			}
			return meshErrors.GetOsokResponseByHandlingReconcileError(resourceCreationErr)
		}
	} else if c.hasSpecChanged(object) || hasResourceUpdatedInCp {
		hasChanged, err = c.handler.HasCompartmentIdChanged(object, resourceDetails)
		if err != nil {
			return meshErrors.GetOsokResponseByHandlingReconcileError(err)
		}
		if hasChanged {
			err = c.handler.ChangeCompartment(ctx, object, resourceDetails)
		} else {
			err = c.handler.UpdateResource(ctx, object, resourceDetails)
		}
		_ = c.UpdateServiceMeshConfiguredStatus(ctx, object, err)
		if err != nil {
			return meshErrors.GetOsokResponseByHandlingReconcileError(err)
		}
	}

	// Clears the opcRetryToken when control plane request retry happens and the request goes through successfully
	_ = c.UpdateOpcRetryToken(ctx, object, nil)
	resourceDetails.OpcRetryToken = nil

	updateMeshConfiguredErr := c.UpdateServiceMeshConfiguredStatus(ctx, object, nil)
	if updateMeshConfiguredErr != nil {
		return meshErrors.GetOsokResponseByHandlingReconcileError(updateMeshConfiguredErr)
	}
	return c.UpdateK8s(ctx, object, resourceDetails, hasChanged, hasResourceUpdatedInCp)
}

func (c *ServiceMeshServiceManager) hasSpecChanged(object client.Object) bool {
	// metadata.generation field is updated if and only if the value at the .spec subpath changes.
	// If the spec does not change, .metadata.generation is not updated
	// Update only if metadata.generation differs, else it will reconcile forever
	// This condition ensures we don't do an update call to CP everytime it reconciles
	// Additionally, sync resource from k8s to CP periodically for every commons.RequeueSyncDuration, should change only the status subresource
	serviceMeshStatus, err := c.handler.GetServiceMeshStatus(object)
	if err != nil {
		return false
	}
	existingCondition := meshCommons.GetServiceMeshCondition(serviceMeshStatus, servicemeshapi.ServiceMeshActive)
	if existingCondition == nil || object.GetGeneration() != existingCondition.ObservedGeneration {
		return true
	}
	return false
}

func (c *ServiceMeshServiceManager) hasResourceUpdatedInCp(object client.Object, details *ResourceDetails) bool {
	serviceMeshStatus, err := c.handler.GetServiceMeshStatus(object)
	if err != nil {
		return false
	}

	if !isResourceActiveWithStatusTrue(serviceMeshStatus) {
		return false
	}

	if serviceMeshStatus.LastUpdatedTime == nil {
		return false
	}

	cpTime := c.handler.GetTimeUpdated(details)
	operatorTime := serviceMeshStatus.LastUpdatedTime

	return DoTimeStampsDiffer(cpTime, operatorTime)
}

func DoTimeStampsDiffer(cpTimeUpdated *common.SDKTime, operatorTimeUpdated *metav1.Time) bool {
	if operatorTimeUpdated == nil && cpTimeUpdated == nil {
		return false
	}
	if operatorTimeUpdated == nil || cpTimeUpdated == nil {
		return true
	}
	return cpTimeUpdated.Time.Unix() != operatorTimeUpdated.Time.Unix()
}

// UpdateServiceMeshCondition updates the status of the resource
// Called when there is an error from CP
// Additionally, also called when the state is not Active
// We haven't fully synced the spec, hence using the existing generation from the status for generation field
func (c *ServiceMeshServiceManager) UpdateServiceMeshCondition(ctx context.Context, obj client.Object, status metav1.ConditionStatus, reason string, message string, meshConditionType servicemeshapi.ServiceMeshConditionType) error {
	oldObj := obj.DeepCopyObject()
	serviceMeshStatus, err := c.handler.GetServiceMeshStatus(obj)
	if err != nil {
		return err
	}
	existingCondition := meshCommons.GetServiceMeshCondition(serviceMeshStatus, meshConditionType)
	var generation int64
	if existingCondition == nil {
		generation = 1
	} else if status == metav1.ConditionUnknown {
		// do not update the generation for unknown condition status
		generation = existingCondition.ObservedGeneration
	} else {
		generation = obj.GetGeneration()
	}
	if !meshCommons.UpdateServiceMeshCondition(serviceMeshStatus, meshConditionType, status, reason, message, generation) {
		return nil
	}
	return c.client.Status().Patch(ctx, obj, client.MergeFrom(oldObj))
}

func (c *ServiceMeshServiceManager) UpdateServiceMeshConfiguredStatus(ctx context.Context, obj client.Object, err error) error {
	if err == nil {
		return c.UpdateServiceMeshCondition(ctx, obj, metav1.ConditionTrue, string(meshCommons.Successful), string(meshCommons.ResourceConfigured), servicemeshapi.ServiceMeshConfigured)
	}
	if common.IsNetworkError(err) {
		return c.UpdateServiceMeshCondition(ctx, obj, metav1.ConditionUnknown, string(meshCommons.ConnectionError), err.Error(), servicemeshapi.ServiceMeshConfigured)
	}
	if serviceError, ok := err.(common.ServiceError); ok {
		return c.UpdateServiceMeshCondition(ctx, obj, meshErrors.GetMeshConfiguredConditionStatus(serviceError), serviceError.GetCode(), meshErrors.GetErrorMessage(serviceError), servicemeshapi.ServiceMeshConfigured)
	}
	return nil
}

func (c *ServiceMeshServiceManager) UpdateServiceMeshActiveStatus(ctx context.Context, obj client.Object, err error) {
	if common.IsNetworkError(err) {
		_ = c.UpdateServiceMeshCondition(ctx, obj, metav1.ConditionUnknown, string(meshCommons.ConnectionError), err.Error(), servicemeshapi.ServiceMeshActive)
		return
	}
	if serviceError, ok := err.(common.ServiceError); ok {
		_ = c.UpdateServiceMeshCondition(ctx, obj, meshErrors.GetConditionStatus(serviceError), serviceError.GetCode(), meshErrors.GetErrorMessage(serviceError), servicemeshapi.ServiceMeshActive)
	}
}

func (c *ServiceMeshServiceManager) UpdateK8s(ctx context.Context, obj client.Object, details *ResourceDetails, hasCompartmentChanged bool, hasResourceUpdatedInCp bool) (servicemanager.OSOKResponse, error) {
	oldObject := obj.DeepCopyObject()
	needsUpdate, err := c.handler.UpdateStatus(obj, details)
	if err != nil {
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}
	serviceMeshStatus, err := c.handler.GetServiceMeshStatus(obj)
	if err != nil {
		return meshErrors.GetOsokResponseByHandlingReconcileError(err)
	}
	status := c.handler.GetConditionStatus(details)
	// This is a special case where both CompartmentId of the resource are updated
	// Keep the status as unknown and do NOT update the generation till all the updates are propagated to CP
	// if resource is deleted or failed (i.e. status = false) do not update to cp
	if hasCompartmentChanged && status != metav1.ConditionFalse {
		status = metav1.ConditionUnknown
		serviceMeshCondition := meshCommons.GetServiceMeshCondition(serviceMeshStatus, servicemeshapi.ServiceMeshActive)
		generation := serviceMeshCondition.ObservedGeneration
		if hasResourceUpdatedInCp {
			generation = serviceMeshCondition.ObservedGeneration - 1
		}
		if meshCommons.UpdateServiceMeshCondition(serviceMeshStatus, servicemeshapi.ServiceMeshActive, status, string(meshCommons.GetReason(status)), string(meshCommons.ResourceChangeCompartment), generation) {
			needsUpdate = true
		}
	} else if meshCommons.UpdateServiceMeshCondition(serviceMeshStatus, servicemeshapi.ServiceMeshActive, status, string(meshCommons.GetReason(status)), string(meshCommons.GetMessage(c.handler.GetLifecycleState(details))), obj.GetGeneration()) {
		needsUpdate = true
	}
	if needsUpdate {
		if err := c.client.Status().Patch(ctx, obj, client.MergeFrom(oldObject)); err != nil {
			return meshErrors.GetOsokResponseByHandlingReconcileError(err)
		}
	}
	// Requeue request if state is unknown
	if status == metav1.ConditionUnknown {
		return meshErrors.GetOsokResponseByHandlingReconcileError(meshErrors.NewRequeueOnError(errors.New(meshCommons.UnknownStatus)))
	}
	// For other states, requeue every RequeueSyncDuration
	return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: meshCommons.RequeueSyncDuration}, nil
}

func (c *ServiceMeshServiceManager) UpdateServiceMeshDependenciesActiveStatus(ctx context.Context, object client.Object, err error) {
	if err == nil {
		_ = c.UpdateServiceMeshCondition(ctx, object, metav1.ConditionTrue, string(meshCommons.Successful), string(meshCommons.DependenciesResolved), servicemeshapi.ServiceMeshDependenciesActive)
		return
	}
	if common.IsNetworkError(err) {
		_ = c.UpdateServiceMeshCondition(ctx, object, metav1.ConditionUnknown, string(meshCommons.ConnectionError), err.Error(), servicemeshapi.ServiceMeshDependenciesActive)
		return
	}
	if serviceError, ok := err.(common.ServiceError); ok {
		_ = c.UpdateServiceMeshCondition(ctx, object, meshErrors.GetConditionStatus(serviceError), serviceError.GetCode(), meshErrors.GetErrorMessage(serviceError), servicemeshapi.ServiceMeshDependenciesActive)
	} else {
		_ = c.UpdateServiceMeshCondition(ctx, object, meshCommons.GetConditionStatusFromK8sError(err), string(meshCommons.DependenciesNotResolved), err.Error(), servicemeshapi.ServiceMeshDependenciesActive)
	}
}

func (c *ServiceMeshServiceManager) GetCrdStatus(obj runtime.Object) (*api.OSOKStatus, error) {
	return nil, nil
}

func (c *ServiceMeshServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	object, err := c.handler.VerifyEntityType(obj)
	if err != nil {
		return false, err
	}
	if core.HasFinalizer(object, c.handler.GetFinalizer()) {
		c.log.InfoLogWithFixedMessage(ctx, "Checking dependencies")
		err := c.handler.Finalize(ctx, object)
		c.UpdateServiceMeshDependenciesActiveStatus(ctx, object, err)
		if err != nil {
			return false, err
		}
		if err = c.removeFinalizer(ctx, object, c.handler.GetFinalizer()); err != nil {
			return false, err
		}
	}

	if core.HasFinalizer(object, core.OSOKFinalizerName) {
		c.log.InfoLogWithFixedMessage(ctx, "Attempting to delete the resource in the control plane")
		err := c.handler.DeleteResource(ctx, object)
		if err != nil {
			c.UpdateServiceMeshActiveStatus(ctx, object, err)
			return false, err
		}
	}
	return true, nil
}

func (c *ServiceMeshServiceManager) removeFinalizer(ctx context.Context, obj client.Object, finalizer string) error {
	oldObj := obj.DeepCopyObject()
	needsUpdate := false
	if controllerutil.ContainsFinalizer(obj, finalizer) {
		controllerutil.RemoveFinalizer(obj, finalizer)
		needsUpdate = true
	}
	if !needsUpdate {
		return nil
	}
	return c.client.Patch(ctx, obj, client.MergeFrom(oldObj))
}

func (c *ServiceMeshServiceManager) UpdateOpcRetryToken(ctx context.Context, obj client.Object, opcRetryToken *string) error {
	oldObj := obj.DeepCopyObject()
	needsUpdate := false
	serviceMeshStatus, err := c.handler.GetServiceMeshStatus(obj)
	if err != nil {
		return err
	}
	if serviceMeshStatus.OpcRetryToken == nil && opcRetryToken == nil {
		return nil
	}
	if serviceMeshStatus.OpcRetryToken == nil || opcRetryToken == nil || *serviceMeshStatus.OpcRetryToken != *opcRetryToken {
		serviceMeshStatus.OpcRetryToken = opcRetryToken
		needsUpdate = true
	}
	if !needsUpdate {
		return nil
	}
	return c.client.Status().Patch(ctx, obj, client.MergeFrom(oldObj))
}

func (c *ServiceMeshServiceManager) getOpcRetryToken(obj client.Object) (*string, error) {
	serviceMeshStatus, err := c.handler.GetServiceMeshStatus(obj)
	if err != nil {
		return nil, err
	}
	if serviceMeshStatus.OpcRetryToken != nil {
		return serviceMeshStatus.OpcRetryToken, nil
	}
	opcRetryToken := common.RetryToken()
	return &opcRetryToken, nil
}

func isResourceActiveWithStatusTrue(serviceMeshStatus *servicemeshapi.ServiceMeshStatus) bool {
	if serviceMeshStatus.Conditions == nil {
		return false
	}
	for _, condition := range serviceMeshStatus.Conditions {
		if condition.Status != metav1.ConditionTrue && condition.Type == servicemeshapi.ServiceMeshActive {
			return false
		}
	}
	return true
}
