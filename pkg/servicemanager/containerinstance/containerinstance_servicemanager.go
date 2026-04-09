/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerinstance

import (
	"context"
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	containerinstancessdk "github.com/oracle/oci-go-sdk/v65/containerinstances"
	containerinstancesv1beta1 "github.com/oracle/oci-service-operator/api/containerinstances/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const containerInstanceRequeue = time.Minute

// Compile-time check that ContainerInstanceServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &ContainerInstanceServiceManager{}

// ContainerInstanceServiceManager implements OSOKServiceManager for OCI Container Instances.
type ContainerInstanceServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	Metrics          *metrics.Metrics
	ociClient        ContainerInstanceClientInterface
}

func NewContainerInstanceServiceManagerWithDeps(deps servicemanager.RuntimeDeps) *ContainerInstanceServiceManager {
	return &ContainerInstanceServiceManager{
		Provider:         deps.Provider,
		CredentialClient: deps.CredentialClient,
		Scheme:           deps.Scheme,
		Log:              deps.Log,
		Metrics:          deps.Metrics,
	}
}

// NewContainerInstanceServiceManager creates a new ContainerInstanceServiceManager.
func NewContainerInstanceServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *ContainerInstanceServiceManager {
	return NewContainerInstanceServiceManagerWithDeps(servicemanager.RuntimeDeps{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	})
}

// CreateOrUpdate reconciles the ContainerInstance resource against OCI.
func (c *ContainerInstanceServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
	ci, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	ciInstance, err := c.resolveContainerInstance(ctx, ci)
	if err != nil {
		err = c.recordCreateOrUpdateError(&ci.Status.OsokStatus, err)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	if ciInstance == nil {
		err = c.recordCreateOrUpdateError(&ci.Status.OsokStatus, fmt.Errorf("resolved container instance is nil"))
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	c.syncObservedStatus(ci, ciInstance)
	servicemanager.SetCreatedAtIfUnset(&ci.Status.OsokStatus)

	return c.reconcileLifecycle(&ci.Status.OsokStatus, ciInstance), nil
}

// Delete handles deletion of the container instance called by the finalizer.
func (c *ContainerInstanceServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	ci, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	targetID, ok := trackedContainerInstanceID(ci)
	if !ok {
		c.Log.InfoLog("ContainerInstance has no OCID, nothing to delete")
		return true, nil
	}

	current, err := c.GetContainerInstance(ctx, targetID, nil)
	if err != nil {
		if servicemanager.IsNotFoundServiceError(err) {
			return true, nil
		}
		c.Log.ErrorLog(err, "Error while getting ContainerInstance before delete")
		return false, err
	}

	switch current.LifecycleState {
	case containerinstancessdk.ContainerInstanceLifecycleStateDeleted:
		return true, nil
	case containerinstancessdk.ContainerInstanceLifecycleStateDeleting:
		return false, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting ContainerInstance %s", targetID))
	if err := c.DeleteContainerInstance(ctx, targetID); err != nil {
		if servicemanager.IsNotFoundServiceError(err) {
			return true, nil
		}
		c.Log.ErrorLog(err, "Error while deleting ContainerInstance")
		return false, err
	}

	return false, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *ContainerInstanceServiceManager) GetCrdStatus(obj runtime.Object) (*shared.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *ContainerInstanceServiceManager) convert(obj runtime.Object) (*containerinstancesv1beta1.ContainerInstance, error) {
	ci, ok := obj.(*containerinstancesv1beta1.ContainerInstance)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for ContainerInstance")
	}
	return ci, nil
}

func (c *ContainerInstanceServiceManager) resolveContainerInstance(ctx context.Context,
	ci *containerinstancesv1beta1.ContainerInstance) (*containerinstancessdk.ContainerInstance, error) {
	if trackedID, ok := trackedContainerInstanceID(ci); ok {
		instance, err := c.GetContainerInstance(ctx, trackedID, nil)
		if err == nil {
			return c.refreshOrUpdateTrackedInstance(ctx, ci, trackedID, instance)
		}
		if !servicemanager.IsNotFoundServiceError(err) {
			c.Log.ErrorLog(err, "Error while getting tracked ContainerInstance")
			return nil, err
		}

		ci.Status.Id = ""
		ci.Status.OsokStatus.Ocid = ""
	}

	return c.lookupOrCreateContainerInstance(ctx, ci)
}

func (c *ContainerInstanceServiceManager) lookupOrCreateContainerInstance(ctx context.Context,
	ci *containerinstancesv1beta1.ContainerInstance) (*containerinstancessdk.ContainerInstance, error) {
	ciOcid, err := c.GetContainerInstanceOcid(ctx, *ci)
	if err != nil {
		return nil, err
	}
	if ciOcid == nil {
		resp, err := c.CreateContainerInstance(ctx, *ci)
		if err != nil {
			return nil, err
		}
		return &resp.ContainerInstance, nil
	}

	instance, err := c.GetContainerInstance(ctx, *ciOcid, nil)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting ContainerInstance by OCID")
		return nil, err
	}
	return c.refreshOrUpdateTrackedInstance(ctx, ci, *ciOcid, instance)
}

func (c *ContainerInstanceServiceManager) refreshOrUpdateTrackedInstance(ctx context.Context,
	ci *containerinstancesv1beta1.ContainerInstance,
	targetID shared.OCID,
	instance *containerinstancessdk.ContainerInstance,
) (*containerinstancessdk.ContainerInstance, error) {
	if !supportsContainerInstanceUpdate(instance.LifecycleState) {
		return instance, nil
	}

	if err := c.UpdateContainerInstance(ctx, ci, instance); err != nil {
		c.Log.ErrorLog(err, "Error while updating ContainerInstance")
		return nil, err
	}

	updated, err := c.GetContainerInstance(ctx, targetID, nil)
	if err != nil {
		c.Log.ErrorLog(err, "Error while refreshing ContainerInstance after update")
		return nil, err
	}
	return updated, nil
}

func (c *ContainerInstanceServiceManager) reconcileLifecycle(status *shared.OSOKStatus,
	instance *containerinstancessdk.ContainerInstance) servicemanager.OSOKResponse {
	displayName := safeString(instance.DisplayName)
	message := fmt.Sprintf("ContainerInstance %s is %s", displayName, instance.LifecycleState)

	switch instance.LifecycleState {
	case containerinstancessdk.ContainerInstanceLifecycleStateActive,
		containerinstancessdk.ContainerInstanceLifecycleStateInactive:
		*status = util.UpdateOSOKStatusCondition(*status, shared.Active, v1.ConditionTrue, "", message, c.Log)
		status.Message = message
		status.Reason = string(shared.Active)
		c.Log.InfoLog(message)
		return servicemanager.OSOKResponse{IsSuccessful: true}
	case containerinstancessdk.ContainerInstanceLifecycleStateCreating:
		*status = util.UpdateOSOKStatusCondition(*status, shared.Provisioning, v1.ConditionTrue, "", message, c.Log)
		status.Message = message
		status.Reason = string(shared.Provisioning)
		c.Log.InfoLog(message)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: containerInstanceRequeue}
	case containerinstancessdk.ContainerInstanceLifecycleStateUpdating:
		*status = util.UpdateOSOKStatusCondition(*status, shared.Updating, v1.ConditionTrue, "", message, c.Log)
		status.Message = message
		status.Reason = string(shared.Updating)
		c.Log.InfoLog(message)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: containerInstanceRequeue}
	case containerinstancessdk.ContainerInstanceLifecycleStateDeleting:
		*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.Log)
		status.Message = message
		status.Reason = string(shared.Terminating)
		c.Log.InfoLog(message)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: containerInstanceRequeue}
	default:
		*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", message, c.Log)
		status.Message = message
		status.Reason = string(shared.Failed)
		c.Log.InfoLog(message)
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
}

func (c *ContainerInstanceServiceManager) recordCreateOrUpdateError(status *shared.OSOKStatus, err error) error {
	reason := ""
	if serviceErr, ok := common.IsServiceError(err); ok {
		reason = serviceErr.GetCode()
		if _, mappedErr := errorutil.NewServiceFailureFromResponse(
			serviceErr.GetCode(),
			serviceErr.GetHTTPStatusCode(),
			serviceErr.GetOpcRequestID(),
			serviceErr.GetMessage(),
		); mappedErr != nil {
			err = mappedErr
		}
	}

	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, reason, err.Error(), c.Log)
	status.Message = err.Error()
	status.Reason = reason
	c.Log.ErrorLog(err, "ContainerInstance reconcile failed")
	return err
}

func (c *ContainerInstanceServiceManager) syncObservedStatus(ci *containerinstancesv1beta1.ContainerInstance,
	instance *containerinstancessdk.ContainerInstance) {
	ci.Status.OsokStatus.Ocid = shared.OCID(safeString(instance.Id))
	ci.Status.Id = safeString(instance.Id)
	ci.Status.DisplayName = safeString(instance.DisplayName)
	ci.Status.CompartmentId = safeString(instance.CompartmentId)
	ci.Status.AvailabilityDomain = safeString(instance.AvailabilityDomain)
	ci.Status.LifecycleState = string(instance.LifecycleState)
	ci.Status.ContainerCount = safeInt(instance.ContainerCount)
	ci.Status.TimeCreated = sdkTimeString(instance.TimeCreated)
	ci.Status.Shape = safeString(instance.Shape)
	if instance.ShapeConfig != nil {
		ci.Status.ShapeConfig = containerinstancesv1beta1.ContainerInstanceShapeConfigObservedState{
			Ocpus:                     safeFloat32(instance.ShapeConfig.Ocpus),
			MemoryInGBs:               safeFloat32(instance.ShapeConfig.MemoryInGBs),
			ProcessorDescription:      safeString(instance.ShapeConfig.ProcessorDescription),
			NetworkingBandwidthInGbps: safeFloat32(instance.ShapeConfig.NetworkingBandwidthInGbps),
		}
	}
	ci.Status.ContainerRestartPolicy = string(instance.ContainerRestartPolicy)
	ci.Status.FreeformTags = cloneStringMap(instance.FreeformTags)
	ci.Status.DefinedTags = convertDefinedTags(instance.DefinedTags)
	ci.Status.SystemTags = convertDefinedTags(instance.SystemTags)
	ci.Status.FaultDomain = safeString(instance.FaultDomain)
	ci.Status.LifecycleDetails = safeString(instance.LifecycleDetails)
	ci.Status.VolumeCount = safeInt(instance.VolumeCount)
	ci.Status.TimeUpdated = sdkTimeString(instance.TimeUpdated)
	if instance.DnsConfig != nil {
		ci.Status.DnsConfig = containerinstancesv1beta1.ContainerInstanceDnsConfig{
			Nameservers: append([]string(nil), instance.DnsConfig.Nameservers...),
			Searches:    append([]string(nil), instance.DnsConfig.Searches...),
			Options:     append([]string(nil), instance.DnsConfig.Options...),
		}
	} else {
		ci.Status.DnsConfig = containerinstancesv1beta1.ContainerInstanceDnsConfig{}
	}
	ci.Status.GracefulShutdownTimeoutInSeconds = safeInt64(instance.GracefulShutdownTimeoutInSeconds)
}

func trackedContainerInstanceID(ci *containerinstancesv1beta1.ContainerInstance) (shared.OCID, bool) {
	if ci.Status.OsokStatus.Ocid != "" {
		return ci.Status.OsokStatus.Ocid, true
	}
	if ci.Status.Id != "" {
		return shared.OCID(ci.Status.Id), true
	}
	return "", false
}

func supportsContainerInstanceUpdate(state containerinstancessdk.ContainerInstanceLifecycleStateEnum) bool {
	return state == containerinstancessdk.ContainerInstanceLifecycleStateActive ||
		state == containerinstancessdk.ContainerInstanceLifecycleStateInactive
}

func safeString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func safeInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func safeInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func safeFloat32(value *float32) float32 {
	if value == nil {
		return 0
	}
	return *value
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func convertDefinedTags(in map[string]map[string]interface{}) map[string]shared.MapValue {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]shared.MapValue, len(in))
	for namespace, values := range in {
		converted := make(shared.MapValue, len(values))
		for key, value := range values {
			converted[key] = fmt.Sprint(value)
		}
		out[namespace] = converted
	}
	return out
}
