/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networksecuritygroup

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	networkSecurityGroupRequeueDuration      = time.Minute
	networkSecurityGroupLifecycleStateUpdate = coresdk.NetworkSecurityGroupLifecycleStateEnum("UPDATING")
)

type networkSecurityGroupOCIClient interface {
	CreateNetworkSecurityGroup(ctx context.Context, request coresdk.CreateNetworkSecurityGroupRequest) (coresdk.CreateNetworkSecurityGroupResponse, error)
	GetNetworkSecurityGroup(ctx context.Context, request coresdk.GetNetworkSecurityGroupRequest) (coresdk.GetNetworkSecurityGroupResponse, error)
	ListNetworkSecurityGroups(ctx context.Context, request coresdk.ListNetworkSecurityGroupsRequest) (coresdk.ListNetworkSecurityGroupsResponse, error)
	UpdateNetworkSecurityGroup(ctx context.Context, request coresdk.UpdateNetworkSecurityGroupRequest) (coresdk.UpdateNetworkSecurityGroupResponse, error)
	DeleteNetworkSecurityGroup(ctx context.Context, request coresdk.DeleteNetworkSecurityGroupRequest) (coresdk.DeleteNetworkSecurityGroupResponse, error)
}

type networkSecurityGroupGeneratedParityClient struct {
	manager  *NetworkSecurityGroupServiceManager
	delegate NetworkSecurityGroupServiceClient
	client   networkSecurityGroupOCIClient
	initErr  error
}

func init() {
	registerNetworkSecurityGroupRuntimeHooksMutator(func(manager *NetworkSecurityGroupServiceManager, hooks *NetworkSecurityGroupRuntimeHooks) {
		applyNetworkSecurityGroupRuntimeHooks(manager, hooks, nil)
	})
}

func applyNetworkSecurityGroupRuntimeHooks(
	manager *NetworkSecurityGroupServiceManager,
	hooks *NetworkSecurityGroupRuntimeHooks,
	client networkSecurityGroupOCIClient,
) {
	if hooks == nil {
		return
	}

	if hooks.Semantics != nil {
		semantics := *hooks.Semantics
		mutation := semantics.Mutation
		mutation.ForceNew = nil
		semantics.Mutation = mutation
		hooks.Semantics = &semantics
	}

	runtimeClient := newNetworkSecurityGroupRuntimeClient(manager, nil, client)

	hooks.BuildCreateBody = func(_ context.Context, resource *corev1beta1.NetworkSecurityGroup, _ string) (any, error) {
		return buildCreateNetworkSecurityGroupDetails(resource.Spec), nil
	}
	hooks.BuildUpdateBody = func(_ context.Context, resource *corev1beta1.NetworkSecurityGroup, _ string, currentResponse any) (any, bool, error) {
		current, ok := networkSecurityGroupFromResponse(currentResponse)
		if !ok {
			return nil, false, fmt.Errorf("unexpected NetworkSecurityGroup current response type %T", currentResponse)
		}
		request, updateNeeded, err := runtimeClient.buildUpdateRequest(resource, current)
		if err != nil {
			return nil, false, err
		}
		return request.UpdateNetworkSecurityGroupDetails, updateNeeded, nil
	}
	hooks.TrackedRecreate.ClearTrackedIdentity = runtimeClient.clearTrackedIdentity
	hooks.StatusHooks.ProjectStatus = func(resource *corev1beta1.NetworkSecurityGroup, response any) error {
		current, ok := networkSecurityGroupFromResponse(response)
		if !ok {
			return fmt.Errorf("unexpected NetworkSecurityGroup status response type %T", response)
		}
		return runtimeClient.projectStatus(resource, current)
	}
	hooks.StatusHooks.ApplyLifecycle = func(resource *corev1beta1.NetworkSecurityGroup, response any) (servicemanager.OSOKResponse, error) {
		current, ok := networkSecurityGroupFromResponse(response)
		if !ok {
			return runtimeClient.fail(resource, fmt.Errorf("unexpected NetworkSecurityGroup lifecycle response type %T", response))
		}
		return runtimeClient.applyLifecycle(resource, current)
	}
	hooks.ParityHooks.ValidateCreateOnlyDrift = func(resource *corev1beta1.NetworkSecurityGroup, response any) error {
		current, ok := networkSecurityGroupFromResponse(response)
		if !ok {
			return fmt.Errorf("unexpected NetworkSecurityGroup current response type %T", response)
		}
		return validateNetworkSecurityGroupCreateOnlyDrift(resource.Spec, current)
	}
	hooks.ParityHooks.RequiresParityHandling = func(resource *corev1beta1.NetworkSecurityGroup, response any) bool {
		current, ok := networkSecurityGroupFromResponse(response)
		if !ok {
			return false
		}
		return requiresManualNetworkSecurityGroupUpdate(resource.Spec, current)
	}
	hooks.ParityHooks.ApplyParityUpdate = func(ctx context.Context, resource *corev1beta1.NetworkSecurityGroup, response any) (servicemanager.OSOKResponse, error) {
		current, ok := networkSecurityGroupFromResponse(response)
		if !ok {
			return runtimeClient.fail(resource, fmt.Errorf("unexpected NetworkSecurityGroup parity response type %T", response))
		}
		return runtimeClient.update(ctx, resource, current)
	}
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate NetworkSecurityGroupServiceClient) NetworkSecurityGroupServiceClient {
		wrapped := *runtimeClient
		wrapped.delegate = delegate
		return &wrapped
	})
}

func newNetworkSecurityGroupRuntimeClient(
	manager *NetworkSecurityGroupServiceManager,
	delegate NetworkSecurityGroupServiceClient,
	client networkSecurityGroupOCIClient,
) *networkSecurityGroupGeneratedParityClient {
	runtimeClient := &networkSecurityGroupGeneratedParityClient{
		manager:  manager,
		delegate: delegate,
		client:   client,
	}
	if runtimeClient.client != nil {
		return runtimeClient
	}

	sdkClient, err := coresdk.NewVirtualNetworkClientWithConfigurationProvider(manager.Provider)
	runtimeClient.client = sdkClient
	if err != nil {
		runtimeClient.initErr = fmt.Errorf("initialize NetworkSecurityGroup OCI client: %w", err)
	}
	return runtimeClient
}

func (c *networkSecurityGroupGeneratedParityClient) CreateOrUpdate(ctx context.Context, resource *corev1beta1.NetworkSecurityGroup, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if c.delegate == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("network security group parity delegate is not configured")
	}

	trackedID := currentNetworkSecurityGroupID(resource)
	explicitRecreate := false
	if trackedID != "" {
		if c.initErr != nil {
			return c.fail(resource, c.initErr)
		}

		_, err := c.get(ctx, trackedID)
		if err != nil {
			if isNetworkSecurityGroupReadNotFoundOCI(err) {
				c.clearTrackedIdentity(resource)
				explicitRecreate = true
			} else {
				return c.fail(resource, normalizeNetworkSecurityGroupOCIError(err))
			}
		}
	}

	previousStatus := resource.Status
	c.clearProjectedStatus(resource)
	clearedStatus := resource.Status

	delegateCtx := ctx
	if explicitRecreate {
		delegateCtx = generatedruntime.WithSkipExistingBeforeCreate(delegateCtx)
	}
	response, err := c.delegate.CreateOrUpdate(delegateCtx, resource, req)
	if err != nil && !networkSecurityGroupProjectedStatusChanged(resource.Status, clearedStatus) {
		c.restoreStatus(resource, previousStatus)
	}
	return response, err
}

func (c *networkSecurityGroupGeneratedParityClient) Delete(ctx context.Context, resource *corev1beta1.NetworkSecurityGroup) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("network security group parity delegate is not configured")
	}
	return c.delegate.Delete(ctx, resource)
}

func (c *networkSecurityGroupGeneratedParityClient) clearProjectedStatus(resource *corev1beta1.NetworkSecurityGroup) {
	if resource == nil {
		return
	}

	resource.Status = corev1beta1.NetworkSecurityGroupStatus{
		OsokStatus: resource.Status.OsokStatus,
		Id:         resource.Status.Id,
	}
}

func (c *networkSecurityGroupGeneratedParityClient) restoreStatus(resource *corev1beta1.NetworkSecurityGroup, previous corev1beta1.NetworkSecurityGroupStatus) {
	if resource == nil {
		return
	}

	failedStatus := resource.Status.OsokStatus
	resource.Status = previous
	resource.Status.OsokStatus = failedStatus
}

func (c *networkSecurityGroupGeneratedParityClient) create(ctx context.Context, resource *corev1beta1.NetworkSecurityGroup) (servicemanager.OSOKResponse, error) {
	request := coresdk.CreateNetworkSecurityGroupRequest{
		CreateNetworkSecurityGroupDetails: buildCreateNetworkSecurityGroupDetails(resource.Spec),
	}

	response, err := c.client.CreateNetworkSecurityGroup(ctx, request)
	if err != nil {
		return c.fail(resource, normalizeNetworkSecurityGroupOCIError(err))
	}

	if err := c.projectStatus(resource, response.NetworkSecurityGroup); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, response.NetworkSecurityGroup)
}

func (c *networkSecurityGroupGeneratedParityClient) get(ctx context.Context, ocid string) (coresdk.NetworkSecurityGroup, error) {
	response, err := c.client.GetNetworkSecurityGroup(ctx, coresdk.GetNetworkSecurityGroupRequest{
		NetworkSecurityGroupId: common.String(ocid),
	})
	if err != nil {
		return coresdk.NetworkSecurityGroup{}, err
	}
	return response.NetworkSecurityGroup, nil
}

func (c *networkSecurityGroupGeneratedParityClient) update(ctx context.Context, resource *corev1beta1.NetworkSecurityGroup, current coresdk.NetworkSecurityGroup) (servicemanager.OSOKResponse, error) {
	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}

	updateRequest, updateNeeded, err := c.buildUpdateRequest(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}
	if !updateNeeded {
		return c.applyLifecycle(resource, current)
	}

	response, err := c.client.UpdateNetworkSecurityGroup(ctx, updateRequest)
	if err != nil {
		return c.fail(resource, normalizeNetworkSecurityGroupOCIError(err))
	}
	if err := c.projectStatus(resource, response.NetworkSecurityGroup); err != nil {
		return c.fail(resource, err)
	}
	return c.applyLifecycle(resource, response.NetworkSecurityGroup)
}

func (c *networkSecurityGroupGeneratedParityClient) buildUpdateRequest(resource *corev1beta1.NetworkSecurityGroup, current coresdk.NetworkSecurityGroup) (coresdk.UpdateNetworkSecurityGroupRequest, bool, error) {
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return coresdk.UpdateNetworkSecurityGroupRequest{}, false, fmt.Errorf("current NetworkSecurityGroup does not expose an OCI identifier")
	}

	if err := validateNetworkSecurityGroupCreateOnlyDrift(resource.Spec, current); err != nil {
		return coresdk.UpdateNetworkSecurityGroupRequest{}, false, err
	}

	updateDetails := coresdk.UpdateNetworkSecurityGroupDetails{}
	updateNeeded := false

	if !stringPtrEqual(current.DisplayName, resource.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}

	desiredFreeformTags := desiredFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}

	desiredDefinedTags := desiredDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}

	if !updateNeeded {
		return coresdk.UpdateNetworkSecurityGroupRequest{}, false, nil
	}

	return coresdk.UpdateNetworkSecurityGroupRequest{
		NetworkSecurityGroupId:            current.Id,
		UpdateNetworkSecurityGroupDetails: updateDetails,
	}, true, nil
}

func buildCreateNetworkSecurityGroupDetails(spec corev1beta1.NetworkSecurityGroupSpec) coresdk.CreateNetworkSecurityGroupDetails {
	createDetails := coresdk.CreateNetworkSecurityGroupDetails{
		CompartmentId: common.String(spec.CompartmentId),
		VcnId:         common.String(spec.VcnId),
	}

	if spec.DefinedTags != nil {
		createDetails.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	if spec.DisplayName != "" {
		createDetails.DisplayName = common.String(spec.DisplayName)
	}
	if spec.FreeformTags != nil {
		createDetails.FreeformTags = spec.FreeformTags
	}

	return createDetails
}

func desiredFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredDefinedTagsForUpdate(spec map[string]shared.MapValue, current map[string]map[string]interface{}) map[string]map[string]interface{} {
	if spec != nil {
		return *util.ConvertToOciDefinedTags(&spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func validateNetworkSecurityGroupCreateOnlyDrift(spec corev1beta1.NetworkSecurityGroupSpec, current coresdk.NetworkSecurityGroup) error {
	var unsupported []string

	if !stringCreateOnlyMatches(current.CompartmentId, spec.CompartmentId) {
		unsupported = append(unsupported, "compartmentId")
	}
	if !stringCreateOnlyMatches(current.VcnId, spec.VcnId) {
		unsupported = append(unsupported, "vcnId")
	}

	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("NetworkSecurityGroup create-only field drift is not supported: %s", strings.Join(unsupported, ", "))
}

func (c *networkSecurityGroupGeneratedParityClient) applyLifecycle(resource *corev1beta1.NetworkSecurityGroup, current coresdk.NetworkSecurityGroup) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	now := time.Now()
	if status.CreatedAt == nil && current.Id != nil && strings.TrimSpace(*current.Id) != "" {
		createdAt := metav1Time(now)
		status.CreatedAt = &createdAt
	}
	updatedAt := metav1Time(now)
	status.UpdatedAt = &updatedAt
	if current.Id != nil {
		status.Ocid = shared.OCID(*current.Id)
	}

	message := networkSecurityGroupLifecycleMessage(current)
	status.Message = message

	switch current.LifecycleState {
	case coresdk.NetworkSecurityGroupLifecycleStateAvailable:
		status.Reason = string(shared.Active)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Active, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	case coresdk.NetworkSecurityGroupLifecycleStateProvisioning, networkSecurityGroupLifecycleStateUpdate:
		status.Reason = string(shared.Provisioning)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Provisioning, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: networkSecurityGroupRequeueDuration}, nil
	case coresdk.NetworkSecurityGroupLifecycleStateTerminating, coresdk.NetworkSecurityGroupLifecycleStateTerminated:
		status.Reason = string(shared.Terminating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: networkSecurityGroupRequeueDuration}, nil
	default:
		return c.fail(resource, fmt.Errorf("NetworkSecurityGroup lifecycle state %q is not modeled for create or update", current.LifecycleState))
	}
}

func (c *networkSecurityGroupGeneratedParityClient) fail(resource *corev1beta1.NetworkSecurityGroup, err error) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	updatedAt := metav1Time(time.Now())
	status.UpdatedAt = &updatedAt
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), c.manager.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *networkSecurityGroupGeneratedParityClient) markDeleted(resource *corev1beta1.NetworkSecurityGroup, message string) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
}

func (c *networkSecurityGroupGeneratedParityClient) clearTrackedIdentity(resource *corev1beta1.NetworkSecurityGroup) {
	resource.Status = corev1beta1.NetworkSecurityGroupStatus{}
}

func (c *networkSecurityGroupGeneratedParityClient) markTerminating(resource *corev1beta1.NetworkSecurityGroup, current coresdk.NetworkSecurityGroup) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.UpdatedAt = &now
	status.Message = networkSecurityGroupLifecycleMessage(current)
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", status.Message, c.manager.Log)
}

func (c *networkSecurityGroupGeneratedParityClient) projectStatus(resource *corev1beta1.NetworkSecurityGroup, current coresdk.NetworkSecurityGroup) error {
	resource.Status = corev1beta1.NetworkSecurityGroupStatus{
		OsokStatus:     resource.Status.OsokStatus,
		CompartmentId:  stringValue(current.CompartmentId),
		Id:             stringValue(current.Id),
		LifecycleState: string(current.LifecycleState),
		TimeCreated:    sdkTimeString(current.TimeCreated),
		VcnId:          stringValue(current.VcnId),
		DefinedTags:    convertOCIToStatusDefinedTags(current.DefinedTags),
		DisplayName:    stringValue(current.DisplayName),
		FreeformTags:   cloneStringMap(current.FreeformTags),
	}
	return nil
}

func networkSecurityGroupLifecycleMessage(current coresdk.NetworkSecurityGroup) string {
	name := ""
	if current.DisplayName != nil {
		name = *current.DisplayName
	}
	if name == "" && current.Id != nil {
		name = *current.Id
	}
	if name == "" {
		name = "NetworkSecurityGroup"
	}
	return fmt.Sprintf("NetworkSecurityGroup %s is %s", name, current.LifecycleState)
}

func normalizeNetworkSecurityGroupOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isNetworkSecurityGroupReadNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound()
}

func isNetworkSecurityGroupDeleteNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func currentNetworkSecurityGroupID(resource *corev1beta1.NetworkSecurityGroup) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func networkSecurityGroupProjectedStatusChanged(current, baseline corev1beta1.NetworkSecurityGroupStatus) bool {
	current.OsokStatus = shared.OSOKStatus{}
	baseline.OsokStatus = shared.OSOKStatus{}
	return !reflect.DeepEqual(current, baseline)
}

func networkSecurityGroupLifecycleIsRetryable(state coresdk.NetworkSecurityGroupLifecycleStateEnum) bool {
	switch state {
	case coresdk.NetworkSecurityGroupLifecycleStateProvisioning,
		networkSecurityGroupLifecycleStateUpdate,
		coresdk.NetworkSecurityGroupLifecycleStateTerminating,
		coresdk.NetworkSecurityGroupLifecycleStateTerminated:
		return true
	default:
		return false
	}
}

func requiresManualNetworkSecurityGroupUpdate(spec corev1beta1.NetworkSecurityGroupSpec, current coresdk.NetworkSecurityGroup) bool {
	if spec.DisplayName == "" && !stringPtrEqual(current.DisplayName, spec.DisplayName) {
		return true
	}
	if spec.FreeformTags == nil {
		return current.FreeformTags != nil
	}
	if len(spec.FreeformTags) == 0 && !reflect.DeepEqual(current.FreeformTags, spec.FreeformTags) {
		return true
	}
	if spec.DefinedTags == nil {
		return current.DefinedTags != nil
	}
	if len(spec.DefinedTags) == 0 {
		desiredDefinedTags := *util.ConvertToOciDefinedTags(&spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
			return true
		}
	}
	return false
}

func stringPtrEqual(actual *string, expected string) bool {
	if actual == nil {
		return strings.TrimSpace(expected) == ""
	}
	return *actual == expected
}

func stringCreateOnlyMatches(actual *string, expected string) bool {
	return strings.TrimSpace(stringValue(actual)) == strings.TrimSpace(expected)
}

func metav1Time(t time.Time) metav1.Time {
	return metav1.NewTime(t)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.Format(time.RFC3339Nano)
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func convertOCIToStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		convertedValues := make(shared.MapValue, len(values))
		for key, value := range values {
			convertedValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = convertedValues
	}
	return converted
}

func networkSecurityGroupFromResponse(response any) (coresdk.NetworkSecurityGroup, bool) {
	switch typed := response.(type) {
	case coresdk.NetworkSecurityGroup:
		return typed, true
	case coresdk.CreateNetworkSecurityGroupResponse:
		return typed.NetworkSecurityGroup, true
	case coresdk.GetNetworkSecurityGroupResponse:
		return typed.NetworkSecurityGroup, true
	case coresdk.UpdateNetworkSecurityGroupResponse:
		return typed.NetworkSecurityGroup, true
	default:
		return coresdk.NetworkSecurityGroup{}, false
	}
}
