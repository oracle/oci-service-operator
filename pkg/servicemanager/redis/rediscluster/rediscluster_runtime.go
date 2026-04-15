/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package rediscluster

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	redissdk "github.com/oracle/oci-go-sdk/v65/redis"
	redisv1beta1 "github.com/oracle/oci-service-operator/api/redis/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const redisRequeueDuration = time.Minute

var redisWorkRequestAsyncAdapter = servicemanager.WorkRequestAsyncAdapter{
	PendingStatusTokens: []string{
		string(redissdk.OperationStatusAccepted),
		string(redissdk.OperationStatusInProgress),
		string(redissdk.OperationStatusWaiting),
		string(redissdk.OperationStatusCanceling),
	},
	SucceededStatusTokens: []string{string(redissdk.OperationStatusSucceeded)},
	FailedStatusTokens:    []string{string(redissdk.OperationStatusFailed)},
	CanceledStatusTokens:  []string{string(redissdk.OperationStatusCanceled)},
	AttentionStatusTokens: []string{string(redissdk.OperationStatusNeedsAttention)},
	CreateActionTokens:    []string{string(redissdk.ActionTypeCreated)},
	UpdateActionTokens:    []string{string(redissdk.ActionTypeUpdated)},
	DeleteActionTokens:    []string{string(redissdk.ActionTypeDeleted)},
}

type redisOCIClient interface {
	CreateRedisCluster(ctx context.Context, request redissdk.CreateRedisClusterRequest) (redissdk.CreateRedisClusterResponse, error)
	GetRedisCluster(ctx context.Context, request redissdk.GetRedisClusterRequest) (redissdk.GetRedisClusterResponse, error)
	ListRedisClusters(ctx context.Context, request redissdk.ListRedisClustersRequest) (redissdk.ListRedisClustersResponse, error)
	UpdateRedisCluster(ctx context.Context, request redissdk.UpdateRedisClusterRequest) (redissdk.UpdateRedisClusterResponse, error)
	DeleteRedisCluster(ctx context.Context, request redissdk.DeleteRedisClusterRequest) (redissdk.DeleteRedisClusterResponse, error)
	GetWorkRequest(ctx context.Context, request redissdk.GetWorkRequestRequest) (redissdk.GetWorkRequestResponse, error)
}

type redisRuntimeClient struct {
	manager *RedisClusterServiceManager
	client  redisOCIClient
	initErr error
}

func init() {
	newRedisClusterServiceClient = func(manager *RedisClusterServiceManager) RedisClusterServiceClient {
		return newRedisDeleteGuardClient(manager, newActiveRedisClusterServiceClient(manager))
	}
}

func newActiveRedisClusterServiceClient(manager *RedisClusterServiceManager) RedisClusterServiceClient {
	sdkClient, err := redissdk.NewRedisClusterClientWithConfigurationProvider(manager.Provider)
	runtimeClient := &redisRuntimeClient{
		manager: manager,
		client:  sdkClient,
	}
	if err != nil {
		runtimeClient.initErr = fmt.Errorf("initialize RedisCluster OCI client: %w", err)
	}
	return runtimeClient
}

func (c *redisRuntimeClient) CreateOrUpdate(
	ctx context.Context,
	resource *redisv1beta1.RedisCluster,
	_ ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if c.initErr != nil {
		return c.fail(resource, c.initErr)
	}

	if workRequestID, phase := currentRedisWorkRequest(resource); workRequestID != "" {
		switch phase {
		case shared.OSOKAsyncPhaseCreate:
			return c.resumeCreate(ctx, resource, workRequestID)
		case shared.OSOKAsyncPhaseUpdate:
			return c.resumeUpdate(ctx, resource, workRequestID)
		}
	}

	trackedID := currentRedisClusterID(resource)
	if trackedID == "" {
		return c.resolveOrCreate(ctx, resource)
	}

	current, err := c.getRedisCluster(ctx, trackedID)
	if err != nil {
		if isRedisReadNotFoundOCI(err) {
			c.clearTrackedIdentity(resource)
			return c.resolveOrCreate(ctx, resource)
		}
		return c.fail(resource, normalizeRedisOCIError(err))
	}

	if err := c.projectStatus(resource, current); err != nil {
		return c.fail(resource, err)
	}

	switch current.LifecycleState {
	case redissdk.RedisClusterLifecycleStateCreating,
		redissdk.RedisClusterLifecycleStateUpdating,
		redissdk.RedisClusterLifecycleStateDeleting,
		redissdk.RedisClusterLifecycleStateFailed:
		return c.finishWithLifecycle(resource, current, ""), nil
	}

	updateRequest, updateNeeded, err := c.buildUpdateRequest(resource, current)
	if err != nil {
		return c.fail(resource, err)
	}
	if !updateNeeded {
		return c.finishWithLifecycle(resource, current, ""), nil
	}

	response, err := c.client.UpdateRedisCluster(ctx, updateRequest)
	if err != nil {
		return c.fail(resource, normalizeRedisOCIError(err))
	}

	workRequestID := strings.TrimSpace(stringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return c.fail(resource, fmt.Errorf("RedisCluster update did not return an opc-work-request-id"))
	}
	c.trackAsyncWorkRequest(resource, shared.OSOKAsyncPhaseUpdate, workRequestID, fmt.Sprintf("RedisCluster update requested; polling work request %s", workRequestID))
	return c.resumeUpdate(ctx, resource, workRequestID)
}

func (c *redisRuntimeClient) Delete(ctx context.Context, resource *redisv1beta1.RedisCluster) (bool, error) {
	if c.initErr != nil {
		return false, c.initErr
	}

	trackedID := currentRedisClusterID(resource)
	if workRequestID, phase := currentRedisWorkRequest(resource); workRequestID != "" && phase == shared.OSOKAsyncPhaseDelete {
		return c.resumeDelete(ctx, resource, trackedID, workRequestID)
	}

	if trackedID == "" {
		c.markDeleted(resource, "OCI RedisCluster identifier is not recorded")
		return true, nil
	}

	response, err := c.client.DeleteRedisCluster(ctx, redissdk.DeleteRedisClusterRequest{
		RedisClusterId: common.String(trackedID),
	})
	if err != nil {
		if isRedisDeleteNotFoundOCI(err) {
			c.markDeleted(resource, "OCI RedisCluster no longer exists")
			return true, nil
		}
		return false, normalizeRedisOCIError(err)
	}

	workRequestID := strings.TrimSpace(stringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return false, fmt.Errorf("RedisCluster delete did not return an opc-work-request-id")
	}
	c.trackAsyncWorkRequest(resource, shared.OSOKAsyncPhaseDelete, workRequestID, fmt.Sprintf("RedisCluster delete requested; polling work request %s", workRequestID))
	return c.resumeDelete(ctx, resource, trackedID, workRequestID)
}

func (c *redisRuntimeClient) resolveOrCreate(ctx context.Context, resource *redisv1beta1.RedisCluster) (servicemanager.OSOKResponse, error) {
	current, err := c.resolveExistingRedisCluster(ctx, resource)
	if err != nil {
		return c.fail(resource, err)
	}
	if current != nil {
		if err := c.projectStatus(resource, *current); err != nil {
			return c.fail(resource, err)
		}
		return c.finishWithLifecycle(resource, *current, ""), nil
	}

	response, err := c.client.CreateRedisCluster(ctx, redissdk.CreateRedisClusterRequest{
		CreateRedisClusterDetails: buildCreateRedisClusterDetails(resource.Spec),
	})
	if err != nil {
		return c.fail(resource, normalizeRedisOCIError(err))
	}

	if response.RedisCluster.Id != nil {
		if err := c.projectStatus(resource, response.RedisCluster); err != nil {
			return c.fail(resource, err)
		}
	}

	workRequestID := strings.TrimSpace(stringValue(response.OpcWorkRequestId))
	if workRequestID == "" {
		return c.fail(resource, fmt.Errorf("RedisCluster create did not return an opc-work-request-id"))
	}
	c.trackAsyncWorkRequest(resource, shared.OSOKAsyncPhaseCreate, workRequestID, fmt.Sprintf("RedisCluster create requested; polling work request %s", workRequestID))
	return c.resumeCreate(ctx, resource, workRequestID)
}

func (c *redisRuntimeClient) resumeCreate(
	ctx context.Context,
	resource *redisv1beta1.RedisCluster,
	workRequestID string,
) (servicemanager.OSOKResponse, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		return c.fail(resource, normalizeRedisOCIError(err))
	}

	currentAsync, err := redisWorkRequestAsyncOperation(resource, workRequest, shared.OSOKAsyncPhaseCreate)
	if err != nil {
		return c.fail(resource, err)
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return c.setAsyncOperation(resource, currentAsync, shared.OSOKAsyncClassPending, redisWorkRequestMessage(currentAsync.Phase, workRequest)), nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		return c.failAsyncOperation(
			resource,
			currentAsync,
			fmt.Errorf("RedisCluster %s work request %s finished with status %s", currentAsync.Phase, workRequestID, workRequest.Status),
		)
	case shared.OSOKAsyncClassSucceeded:
		clusterID, err := resolveRedisClusterIDFromWorkRequest(workRequest, redissdk.ActionTypeCreated)
		if err != nil {
			clusterID = currentRedisClusterID(resource)
			if clusterID == "" {
				return c.failAsyncOperation(resource, currentAsync, err)
			}
		}

		current, err := c.getRedisCluster(ctx, clusterID)
		if err != nil {
			if isRedisReadNotFoundOCI(err) {
				return c.setAsyncOperation(
					resource,
					currentAsync,
					shared.OSOKAsyncClassPending,
					fmt.Sprintf("RedisCluster create work request %s succeeded; waiting for RedisCluster %s to become readable", workRequestID, clusterID),
				), nil
			}
			return c.failAsyncOperation(resource, currentAsync, normalizeRedisOCIError(err))
		}
		if err := c.projectStatus(resource, current); err != nil {
			return c.fail(resource, err)
		}
		return c.finishWithLifecycle(resource, current, shared.OSOKAsyncPhaseCreate), nil
	default:
		return c.fail(resource, fmt.Errorf("RedisCluster create work request %s projected unsupported async class %s", workRequestID, currentAsync.NormalizedClass))
	}
}

func (c *redisRuntimeClient) resumeUpdate(
	ctx context.Context,
	resource *redisv1beta1.RedisCluster,
	workRequestID string,
) (servicemanager.OSOKResponse, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		return c.fail(resource, normalizeRedisOCIError(err))
	}

	currentAsync, err := redisWorkRequestAsyncOperation(resource, workRequest, shared.OSOKAsyncPhaseUpdate)
	if err != nil {
		return c.fail(resource, err)
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return c.setAsyncOperation(resource, currentAsync, shared.OSOKAsyncClassPending, redisWorkRequestMessage(currentAsync.Phase, workRequest)), nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		return c.failAsyncOperation(
			resource,
			currentAsync,
			fmt.Errorf("RedisCluster %s work request %s finished with status %s", currentAsync.Phase, workRequestID, workRequest.Status),
		)
	case shared.OSOKAsyncClassSucceeded:
		clusterID := currentRedisClusterID(resource)
		if clusterID == "" {
			var err error
			clusterID, err = resolveRedisClusterIDFromWorkRequest(workRequest, redissdk.ActionTypeUpdated)
			if err != nil {
				return c.failAsyncOperation(resource, currentAsync, err)
			}
		}

		current, err := c.getRedisCluster(ctx, clusterID)
		if err != nil {
			if isRedisReadNotFoundOCI(err) {
				return c.failAsyncOperation(
					resource,
					currentAsync,
					fmt.Errorf("RedisCluster update work request %s succeeded but RedisCluster %s is no longer readable", workRequestID, clusterID),
				)
			}
			return c.failAsyncOperation(resource, currentAsync, normalizeRedisOCIError(err))
		}
		if err := c.projectStatus(resource, current); err != nil {
			return c.fail(resource, err)
		}
		return c.finishWithLifecycle(resource, current, shared.OSOKAsyncPhaseUpdate), nil
	default:
		return c.fail(resource, fmt.Errorf("RedisCluster update work request %s projected unsupported async class %s", workRequestID, currentAsync.NormalizedClass))
	}
}

func (c *redisRuntimeClient) resumeDelete(
	ctx context.Context,
	resource *redisv1beta1.RedisCluster,
	trackedID string,
	workRequestID string,
) (bool, error) {
	workRequest, err := c.getWorkRequest(ctx, workRequestID)
	if err != nil {
		if trackedID != "" && isRedisDeleteNotFoundOCI(err) {
			current, readErr := c.getRedisCluster(ctx, trackedID)
			if readErr != nil {
				if isRedisDeleteNotFoundOCI(readErr) {
					c.markDeleted(resource, "OCI RedisCluster deleted")
					return true, nil
				}
				return false, normalizeRedisOCIError(readErr)
			}
			if current.LifecycleState == redissdk.RedisClusterLifecycleStateDeleted {
				c.markDeleted(resource, "OCI RedisCluster deleted")
				return true, nil
			}
			c.markDeleteProgress(resource, fmt.Sprintf("RedisCluster delete work request %s is no longer readable; waiting for RedisCluster %s to disappear", workRequestID, trackedID))
			return false, nil
		}
		return false, normalizeRedisOCIError(err)
	}

	currentAsync, err := redisWorkRequestAsyncOperation(resource, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return false, err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		_ = c.setAsyncOperation(resource, currentAsync, shared.OSOKAsyncClassPending, redisWorkRequestMessage(currentAsync.Phase, workRequest))
		return false, nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		_, err := c.failAsyncOperation(
			resource,
			currentAsync,
			fmt.Errorf("RedisCluster %s work request %s finished with status %s", currentAsync.Phase, workRequestID, workRequest.Status),
		)
		return false, err
	case shared.OSOKAsyncClassSucceeded:
		if trackedID == "" {
			c.markDeleted(resource, "OCI RedisCluster delete work request completed")
			return true, nil
		}

		current, err := c.getRedisCluster(ctx, trackedID)
		if err != nil {
			if isRedisDeleteNotFoundOCI(err) {
				c.markDeleted(resource, "OCI RedisCluster deleted")
				return true, nil
			}
			_, deleteErr := c.failAsyncOperation(resource, currentAsync, normalizeRedisOCIError(err))
			return false, deleteErr
		}
		if current.LifecycleState == redissdk.RedisClusterLifecycleStateDeleted {
			c.markDeleted(resource, "OCI RedisCluster deleted")
			return true, nil
		}

		_ = c.setAsyncOperation(resource, currentAsync, shared.OSOKAsyncClassPending, fmt.Sprintf("RedisCluster delete work request %s succeeded; waiting for RedisCluster %s to disappear", workRequestID, trackedID))
		return false, nil
	default:
		return false, fmt.Errorf("RedisCluster delete work request %s projected unsupported async class %s", workRequestID, currentAsync.NormalizedClass)
	}
}

func (c *redisRuntimeClient) resolveExistingRedisCluster(ctx context.Context, resource *redisv1beta1.RedisCluster) (*redissdk.RedisCluster, error) {
	request := redissdk.ListRedisClustersRequest{
		CompartmentId: common.String(resource.Spec.CompartmentId),
		DisplayName:   common.String(resource.Spec.DisplayName),
	}

	matches := make([]redissdk.RedisClusterSummary, 0, 1)
	for {
		response, err := c.client.ListRedisClusters(ctx, request)
		if err != nil {
			return nil, normalizeRedisOCIError(err)
		}

		for _, item := range response.Items {
			if strings.TrimSpace(stringValue(item.DisplayName)) != strings.TrimSpace(resource.Spec.DisplayName) {
				continue
			}
			if strings.TrimSpace(stringValue(item.CompartmentId)) != strings.TrimSpace(resource.Spec.CompartmentId) {
				continue
			}
			if item.LifecycleState == redissdk.RedisClusterLifecycleStateDeleted {
				continue
			}
			matches = append(matches, item)
		}

		if response.OpcNextPage == nil || strings.TrimSpace(*response.OpcNextPage) == "" {
			break
		}
		request.Page = response.OpcNextPage
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		clusterID := strings.TrimSpace(stringValue(matches[0].Id))
		if clusterID == "" {
			return nil, fmt.Errorf("resolve RedisCluster by displayName %q: OCI match does not expose an identifier", resource.Spec.DisplayName)
		}
		current, err := c.getRedisCluster(ctx, clusterID)
		if err != nil {
			if isRedisReadNotFoundOCI(err) {
				return nil, fmt.Errorf("confirm RedisCluster list match %s: OCI resource is no longer readable", clusterID)
			}
			return nil, normalizeRedisOCIError(err)
		}
		return &current, nil
	default:
		return nil, fmt.Errorf("multiple RedisClusters match compartmentId %q and displayName %q", resource.Spec.CompartmentId, resource.Spec.DisplayName)
	}
}

func (c *redisRuntimeClient) getRedisCluster(ctx context.Context, clusterID string) (redissdk.RedisCluster, error) {
	response, err := c.client.GetRedisCluster(ctx, redissdk.GetRedisClusterRequest{
		RedisClusterId: common.String(clusterID),
	})
	if err != nil {
		return redissdk.RedisCluster{}, err
	}
	return response.RedisCluster, nil
}

func (c *redisRuntimeClient) getWorkRequest(ctx context.Context, workRequestID string) (redissdk.WorkRequest, error) {
	response, err := c.client.GetWorkRequest(ctx, redissdk.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	})
	if err != nil {
		return redissdk.WorkRequest{}, err
	}
	return response.WorkRequest, nil
}

func (c *redisRuntimeClient) buildUpdateRequest(
	resource *redisv1beta1.RedisCluster,
	current redissdk.RedisCluster,
) (redissdk.UpdateRedisClusterRequest, bool, error) {
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return redissdk.UpdateRedisClusterRequest{}, false, fmt.Errorf("current RedisCluster does not expose an OCI identifier")
	}
	if err := validateRedisCreateOnlyDrift(resource.Spec, current); err != nil {
		return redissdk.UpdateRedisClusterRequest{}, false, err
	}

	updateDetails := redissdk.UpdateRedisClusterDetails{}
	updateNeeded := false

	if !stringPtrEqual(current.DisplayName, resource.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if !intPtrEqual(current.NodeCount, resource.Spec.NodeCount) {
		updateDetails.NodeCount = common.Int(resource.Spec.NodeCount)
		updateNeeded = true
	}
	if !float32PtrEqual(current.NodeMemoryInGBs, resource.Spec.NodeMemoryInGBs) {
		updateDetails.NodeMemoryInGBs = common.Float32(resource.Spec.NodeMemoryInGBs)
		updateNeeded = true
	}

	desiredFreeformTags := desiredRedisFreeformTagsForUpdate(resource.Spec.FreeformTags, current.FreeformTags)
	if !reflect.DeepEqual(current.FreeformTags, desiredFreeformTags) {
		updateDetails.FreeformTags = desiredFreeformTags
		updateNeeded = true
	}
	desiredDefinedTags := desiredRedisDefinedTagsForUpdate(resource.Spec.DefinedTags, current.DefinedTags)
	if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
		updateDetails.DefinedTags = desiredDefinedTags
		updateNeeded = true
	}

	if !updateNeeded {
		return redissdk.UpdateRedisClusterRequest{}, false, nil
	}

	return redissdk.UpdateRedisClusterRequest{
		RedisClusterId:            current.Id,
		UpdateRedisClusterDetails: updateDetails,
	}, true, nil
}

func buildCreateRedisClusterDetails(spec redisv1beta1.RedisClusterSpec) redissdk.CreateRedisClusterDetails {
	createDetails := redissdk.CreateRedisClusterDetails{
		DisplayName:     common.String(spec.DisplayName),
		CompartmentId:   common.String(spec.CompartmentId),
		NodeCount:       common.Int(spec.NodeCount),
		SoftwareVersion: redissdk.RedisClusterSoftwareVersionEnum(spec.SoftwareVersion),
		NodeMemoryInGBs: common.Float32(spec.NodeMemoryInGBs),
		SubnetId:        common.String(spec.SubnetId),
	}
	if spec.FreeformTags != nil {
		createDetails.FreeformTags = cloneStringMap(spec.FreeformTags)
	}
	if spec.DefinedTags != nil {
		createDetails.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	return createDetails
}

func (c *redisRuntimeClient) finishWithLifecycle(
	resource *redisv1beta1.RedisCluster,
	current redissdk.RedisCluster,
	explicitPhase shared.OSOKAsyncPhase,
) servicemanager.OSOKResponse {
	condition, shouldRequeue := classifyRedisLifecycle(current.LifecycleState)
	message := redisLifecycleMessage(current)
	if asyncCurrent := redisLifecycleAsyncOperation(resource, current, message, explicitPhase); asyncCurrent != nil {
		return c.markAsyncOperation(resource, asyncCurrent)
	}
	return c.markCondition(resource, condition, message, shouldRequeue)
}

func (c *redisRuntimeClient) markCondition(
	resource *redisv1beta1.RedisCluster,
	condition shared.OSOKConditionType,
	message string,
	shouldRequeue bool,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
	}
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	if condition == shared.Active {
		servicemanager.ClearAsyncOperation(status)
	}
	conditionStatus := v1.ConditionTrue
	if condition == shared.Failed {
		conditionStatus = v1.ConditionFalse
	}
	*status = util.UpdateOSOKStatusCondition(*status, condition, conditionStatus, "", message, c.manager.Log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    condition != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: redisRequeueDuration,
	}
}

func (c *redisRuntimeClient) markAsyncOperation(
	resource *redisv1beta1.RedisCluster,
	current *shared.OSOKAsyncOperation,
) servicemanager.OSOKResponse {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	if resource.Status.Id != "" {
		status.Ocid = shared.OCID(resource.Status.Id)
	}
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	if current.UpdatedAt == nil {
		current.UpdatedAt = &now
	}
	projection := servicemanager.ApplyAsyncOperation(status, current, c.manager.Log)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: redisRequeueDuration,
	}
}

func (c *redisRuntimeClient) setAsyncOperation(
	resource *redisv1beta1.RedisCluster,
	current *shared.OSOKAsyncOperation,
	class shared.OSOKAsyncNormalizedClass,
	message string,
) servicemanager.OSOKResponse {
	next := *current
	next.NormalizedClass = class
	next.Message = message
	next.UpdatedAt = nil
	return c.markAsyncOperation(resource, &next)
}

func (c *redisRuntimeClient) failAsyncOperation(
	resource *redisv1beta1.RedisCluster,
	current *shared.OSOKAsyncOperation,
	err error,
) (servicemanager.OSOKResponse, error) {
	if current == nil {
		return c.fail(resource, err)
	}

	class := current.NormalizedClass
	switch class {
	case shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
	default:
		class = shared.OSOKAsyncClassFailed
	}

	return c.setAsyncOperation(resource, current, class, err.Error()), err
}

func (c *redisRuntimeClient) trackAsyncWorkRequest(
	resource *redisv1beta1.RedisCluster,
	phase shared.OSOKAsyncPhase,
	workRequestID string,
	message string,
) {
	_ = c.markAsyncOperation(resource, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           currentRedisAsyncPhase(resource, phase),
		WorkRequestID:   strings.TrimSpace(workRequestID),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         strings.TrimSpace(message),
	})
}

func (c *redisRuntimeClient) markDeleteProgress(resource *redisv1beta1.RedisCluster, message string) {
	workRequestID, _ := currentRedisWorkRequest(resource)
	_ = c.markAsyncOperation(resource, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           currentRedisAsyncPhase(resource, shared.OSOKAsyncPhaseDelete),
		WorkRequestID:   strings.TrimSpace(workRequestID),
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         strings.TrimSpace(message),
	})
}

func (c *redisRuntimeClient) fail(
	resource *redisv1beta1.RedisCluster,
	err error,
) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	if status.Async.Current != nil {
		current := *status.Async.Current
		current.NormalizedClass = shared.OSOKAsyncClassFailed
		current.Message = err.Error()
		current.UpdatedAt = &now
		_ = servicemanager.ApplyAsyncOperation(status, &current, c.manager.Log)
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.manager.Log)
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *redisRuntimeClient) markDeleted(resource *redisv1beta1.RedisCluster, message string) {
	status := &resource.Status.OsokStatus
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	servicemanager.ClearAsyncOperation(status)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", message, c.manager.Log)
}

func (c *redisRuntimeClient) clearTrackedIdentity(resource *redisv1beta1.RedisCluster) {
	resource.Status.Id = ""
	resource.Status.OsokStatus = shared.OSOKStatus{}
}

func (c *redisRuntimeClient) projectStatus(resource *redisv1beta1.RedisCluster, current redissdk.RedisCluster) error {
	resource.Status = redisv1beta1.RedisClusterStatus{
		OsokStatus:                resource.Status.OsokStatus,
		Id:                        stringValue(current.Id),
		DisplayName:               stringValue(current.DisplayName),
		CompartmentId:             stringValue(current.CompartmentId),
		NodeCount:                 intValue(current.NodeCount),
		NodeMemoryInGBs:           float32Value(current.NodeMemoryInGBs),
		PrimaryFqdn:               stringValue(current.PrimaryFqdn),
		PrimaryEndpointIpAddress:  stringValue(current.PrimaryEndpointIpAddress),
		ReplicasFqdn:              stringValue(current.ReplicasFqdn),
		ReplicasEndpointIpAddress: stringValue(current.ReplicasEndpointIpAddress),
		SoftwareVersion:           string(current.SoftwareVersion),
		SubnetId:                  stringValue(current.SubnetId),
		NodeCollection:            convertRedisNodeCollection(current.NodeCollection),
		LifecycleState:            string(current.LifecycleState),
		LifecycleDetails:          stringValue(current.LifecycleDetails),
		TimeCreated:               sdkTimeString(current.TimeCreated),
		TimeUpdated:               sdkTimeString(current.TimeUpdated),
		FreeformTags:              cloneStringMap(current.FreeformTags),
		DefinedTags:               convertOCIToStatusDefinedTags(current.DefinedTags),
		SystemTags:                convertOCIToStatusDefinedTags(current.SystemTags),
	}
	return nil
}

func validateRedisCreateOnlyDrift(spec redisv1beta1.RedisClusterSpec, current redissdk.RedisCluster) error {
	var unsupported []string

	if !stringPtrEqual(current.CompartmentId, spec.CompartmentId) {
		unsupported = append(unsupported, "compartmentId")
	}
	if string(current.SoftwareVersion) != strings.TrimSpace(spec.SoftwareVersion) {
		unsupported = append(unsupported, "softwareVersion")
	}
	if !stringPtrEqual(current.SubnetId, spec.SubnetId) {
		unsupported = append(unsupported, "subnetId")
	}

	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("RedisCluster create-only field drift is not supported: %s", strings.Join(unsupported, ", "))
}

func resolveRedisClusterIDFromWorkRequest(workRequest redissdk.WorkRequest, action redissdk.ActionTypeEnum) (string, error) {
	if id, ok := resolveRedisClusterIDFromResources(workRequest.Resources, action, true); ok {
		return id, nil
	}
	if id, ok := resolveRedisClusterIDFromResources(workRequest.Resources, action, false); ok {
		return id, nil
	}
	return "", fmt.Errorf("RedisCluster work request %s does not expose a Redis cluster identifier", stringValue(workRequest.Id))
}

func resolveRedisClusterIDFromResources(
	resources []redissdk.WorkRequestResource,
	action redissdk.ActionTypeEnum,
	preferRedisOnly bool,
) (string, bool) {
	var candidate string
	for _, resource := range resources {
		if action != "" && resource.ActionType != action {
			continue
		}
		if preferRedisOnly && !isRedisClusterWorkRequestResource(resource) {
			continue
		}
		id := strings.TrimSpace(stringValue(resource.Identifier))
		if id == "" {
			continue
		}
		if candidate == "" {
			candidate = id
			continue
		}
		if candidate != id {
			return "", false
		}
	}
	return candidate, candidate != ""
}

func redisWorkRequestAsyncOperation(
	resource *redisv1beta1.RedisCluster,
	workRequest redissdk.WorkRequest,
	explicitPhase shared.OSOKAsyncPhase,
) (*shared.OSOKAsyncOperation, error) {
	var status *shared.OSOKStatus
	if resource != nil {
		status = &resource.Status.OsokStatus
	}

	fallbackPhase := currentRedisAsyncPhase(resource, explicitPhase)
	if derivedPhase, ok := redisWorkRequestPhaseFromOperationType(workRequest.OperationType); ok {
		if fallbackPhase != "" && fallbackPhase != derivedPhase {
			return nil, fmt.Errorf(
				"RedisCluster work request %s exposes operation type %q for phase %q while reconcile expected phase %q",
				stringValue(workRequest.Id),
				workRequest.OperationType,
				derivedPhase,
				fallbackPhase,
			)
		}
		fallbackPhase = derivedPhase
	}

	current, err := servicemanager.BuildWorkRequestAsyncOperation(status, redisWorkRequestAsyncAdapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        string(workRequest.Status),
		RawOperationType: string(workRequest.OperationType),
		WorkRequestID:    stringValue(workRequest.Id),
		PercentComplete:  workRequest.PercentComplete,
		FallbackPhase:    fallbackPhase,
	})
	if err != nil {
		return nil, err
	}

	current.Message = redisWorkRequestMessage(current.Phase, workRequest)
	return current, nil
}

func redisWorkRequestPhaseFromOperationType(operationType redissdk.OperationTypeEnum) (shared.OSOKAsyncPhase, bool) {
	switch operationType {
	case redissdk.OperationTypeCreateRedisCluster:
		return shared.OSOKAsyncPhaseCreate, true
	case redissdk.OperationTypeUpdateRedisCluster:
		return shared.OSOKAsyncPhaseUpdate, true
	case redissdk.OperationTypeDeleteRedisCluster:
		return shared.OSOKAsyncPhaseDelete, true
	default:
		return "", false
	}
}

func redisWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest redissdk.WorkRequest) string {
	return fmt.Sprintf("RedisCluster %s work request %s is %s", phase, stringValue(workRequest.Id), workRequest.Status)
}

func isRedisClusterWorkRequestResource(resource redissdk.WorkRequestResource) bool {
	entityType := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityType)))
	switch entityType {
	case "redis", "rediscluster", "redis_cluster", "redisclusters":
		return true
	}
	if strings.Contains(entityType, "rediscluster") || strings.Contains(entityType, "redis_cluster") {
		return true
	}
	entityURI := strings.ToLower(strings.TrimSpace(stringValue(resource.EntityUri)))
	return strings.Contains(entityURI, "/redisclusters/")
}

func redisLifecycleMessage(current redissdk.RedisCluster) string {
	name := strings.TrimSpace(stringValue(current.DisplayName))
	if name == "" {
		name = strings.TrimSpace(stringValue(current.Id))
	}
	if name == "" {
		name = "RedisCluster"
	}
	return fmt.Sprintf("RedisCluster %s is %s", name, current.LifecycleState)
}

func classifyRedisLifecycle(state redissdk.RedisClusterLifecycleStateEnum) (shared.OSOKConditionType, bool) {
	switch state {
	case redissdk.RedisClusterLifecycleStateCreating:
		return shared.Provisioning, true
	case redissdk.RedisClusterLifecycleStateUpdating:
		return shared.Updating, true
	case redissdk.RedisClusterLifecycleStateDeleting:
		return shared.Terminating, true
	case redissdk.RedisClusterLifecycleStateFailed:
		return shared.Failed, false
	default:
		return shared.Active, false
	}
}

func redisLifecycleAsyncOperation(
	resource *redisv1beta1.RedisCluster,
	current redissdk.RedisCluster,
	message string,
	explicitPhase shared.OSOKAsyncPhase,
) *shared.OSOKAsyncOperation {
	switch current.LifecycleState {
	case redissdk.RedisClusterLifecycleStateCreating:
		return &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseCreate,
			RawStatus:       string(current.LifecycleState),
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
		}
	case redissdk.RedisClusterLifecycleStateUpdating:
		return &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseUpdate,
			RawStatus:       string(current.LifecycleState),
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
		}
	case redissdk.RedisClusterLifecycleStateDeleting:
		return &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseDelete,
			RawStatus:       string(current.LifecycleState),
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
		}
	case redissdk.RedisClusterLifecycleStateFailed:
		phase := currentRedisAsyncPhase(resource, explicitPhase)
		if phase == "" {
			return nil
		}
		return &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           phase,
			RawStatus:       string(current.LifecycleState),
			NormalizedClass: shared.OSOKAsyncClassFailed,
			Message:         message,
		}
	default:
		return nil
	}
}

func currentRedisAsyncPhase(resource *redisv1beta1.RedisCluster, fallback shared.OSOKAsyncPhase) shared.OSOKAsyncPhase {
	if fallback != "" {
		return fallback
	}
	if resource == nil {
		return ""
	}
	return servicemanager.ResolveAsyncPhase(&resource.Status.OsokStatus, "")
}

func currentRedisWorkRequest(resource *redisv1beta1.RedisCluster) (string, shared.OSOKAsyncPhase) {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return "", ""
	}
	current := resource.Status.OsokStatus.Async.Current
	if current.Source != shared.OSOKAsyncSourceWorkRequest {
		return "", ""
	}
	workRequestID := strings.TrimSpace(current.WorkRequestID)
	if workRequestID == "" {
		return "", ""
	}
	return workRequestID, current.Phase
}

func normalizeRedisOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isRedisReadNotFoundOCI(err error) bool {
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func isRedisDeleteNotFoundOCI(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func currentRedisClusterID(resource *redisv1beta1.RedisCluster) string {
	if resource == nil {
		return ""
	}
	if resource.Status.OsokStatus.Ocid != "" {
		return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid))
	}
	return strings.TrimSpace(resource.Status.Id)
}

func desiredRedisFreeformTagsForUpdate(spec map[string]string, current map[string]string) map[string]string {
	if spec != nil {
		return cloneStringMap(spec)
	}
	if current != nil {
		return map[string]string{}
	}
	return nil
}

func desiredRedisDefinedTagsForUpdate(spec map[string]shared.MapValue, current map[string]map[string]interface{}) map[string]map[string]interface{} {
	if spec != nil {
		return *util.ConvertToOciDefinedTags(&spec)
	}
	if current != nil {
		return map[string]map[string]interface{}{}
	}
	return nil
}

func convertRedisNodeCollection(input *redissdk.NodeCollection) redisv1beta1.RedisClusterNodeCollection {
	if input == nil || len(input.Items) == 0 {
		return redisv1beta1.RedisClusterNodeCollection{}
	}
	items := make([]redisv1beta1.RedisClusterNodeCollectionItem, 0, len(input.Items))
	for _, item := range input.Items {
		items = append(items, redisv1beta1.RedisClusterNodeCollectionItem{
			PrivateEndpointFqdn:      stringValue(item.PrivateEndpointFqdn),
			PrivateEndpointIpAddress: stringValue(item.PrivateEndpointIpAddress),
			DisplayName:              stringValue(item.DisplayName),
		})
	}
	return redisv1beta1.RedisClusterNodeCollection{Items: items}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func float32Value(value *float32) float32 {
	if value == nil {
		return 0
	}
	return *value
}

func stringPtrEqual(actual *string, expected string) bool {
	if actual == nil {
		return strings.TrimSpace(expected) == ""
	}
	return *actual == expected
}

func intPtrEqual(actual *int, expected int) bool {
	if actual == nil {
		return expected == 0
	}
	return *actual == expected
}

func float32PtrEqual(actual *float32, expected float32) bool {
	if actual == nil {
		return expected == 0
	}
	return *actual == expected
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.Format(time.RFC3339Nano)
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func convertOCIToStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if len(input) == 0 {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		if len(values) == 0 {
			continue
		}
		tagValues := make(shared.MapValue, len(values))
		for key, value := range values {
			tagValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = tagValues
	}
	if len(converted) == 0 {
		return nil
	}
	return converted
}
