/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package rediscluster

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	redissdk "github.com/oracle/oci-go-sdk/v65/redis"
	redisv1beta1 "github.com/oracle/oci-service-operator/api/redis/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	ctrl "sigs.k8s.io/controller-runtime"
)

type redisDeleteGuardClient struct {
	delegate         RedisClusterServiceClient
	loadRedisCluster func(context.Context, shared.OCID) (*redissdk.RedisCluster, error)
}

var _ RedisClusterServiceClient = redisDeleteGuardClient{}

func newRedisDeleteGuardClient(manager *RedisClusterServiceManager, delegate RedisClusterServiceClient) RedisClusterServiceClient {
	client := redisDeleteGuardClient{delegate: delegate}
	client.loadRedisCluster = func(ctx context.Context, clusterID shared.OCID) (*redissdk.RedisCluster, error) {
		sdkClient, err := redissdk.NewRedisClusterClientWithConfigurationProvider(manager.Provider)
		if err != nil {
			return nil, fmt.Errorf("initialize RedisCluster delete guard OCI client: %w", err)
		}

		response, err := sdkClient.GetRedisCluster(ctx, redissdk.GetRedisClusterRequest{
			RedisClusterId: common.String(string(clusterID)),
		})
		if err != nil {
			return nil, err
		}
		return &response.RedisCluster, nil
	}
	return client
}

func (c redisDeleteGuardClient) CreateOrUpdate(
	ctx context.Context,
	resource *redisv1beta1.RedisCluster,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c redisDeleteGuardClient) Delete(ctx context.Context, resource *redisv1beta1.RedisCluster) (bool, error) {
	if c.delegate == nil {
		return false, fmt.Errorf("redis delete guard delegate is not configured")
	}
	if redisClusterDeleteGuardHasTrackedDeleteWorkRequest(resource) {
		return c.delegate.Delete(ctx, resource)
	}

	clusterID := redisClusterDeleteGuardCurrentID(resource)
	if clusterID == "" || c.loadRedisCluster == nil {
		return c.delegate.Delete(ctx, resource)
	}

	liveCluster, err := c.loadRedisCluster(ctx, shared.OCID(clusterID))
	if err != nil {
		if redisClusterDeleteGuardIsNotFound(err) {
			return true, nil
		}
		return false, err
	}
	if shouldDelayRedisDelete(liveCluster.LifecycleState) {
		return false, nil
	}
	if liveCluster.LifecycleState == redissdk.RedisClusterLifecycleStateDeleted {
		return true, nil
	}

	deleted, err := c.delegate.Delete(ctx, resource)
	if err == nil || deleted || !redisClusterDeleteGuardIsConflict(err) {
		return deleted, err
	}

	liveCluster, liveErr := c.loadRedisCluster(ctx, shared.OCID(clusterID))
	if liveErr != nil {
		if redisClusterDeleteGuardIsNotFound(liveErr) {
			return true, nil
		}
		return deleted, err
	}
	if shouldDelayRedisDelete(liveCluster.LifecycleState) {
		return false, nil
	}
	if liveCluster.LifecycleState == redissdk.RedisClusterLifecycleStateDeleted {
		return true, nil
	}

	return deleted, err
}

func redisClusterDeleteGuardCurrentID(resource *redisv1beta1.RedisCluster) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func redisClusterDeleteGuardHasTrackedDeleteWorkRequest(resource *redisv1beta1.RedisCluster) bool {
	if resource == nil || resource.Status.OsokStatus.Async.Current == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current.Source == shared.OSOKAsyncSourceWorkRequest &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		strings.TrimSpace(current.WorkRequestID) != ""
}

func shouldDelayRedisDelete(state redissdk.RedisClusterLifecycleStateEnum) bool {
	switch state {
	case redissdk.RedisClusterLifecycleStateCreating,
		redissdk.RedisClusterLifecycleStateUpdating,
		redissdk.RedisClusterLifecycleStateDeleting:
		return true
	default:
		return false
	}
}

func redisClusterDeleteGuardIsNotFound(err error) bool {
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		if serviceErr.GetHTTPStatusCode() == 404 {
			return true
		}
		switch serviceErr.GetCode() {
		case "NotFound", "NotAuthorizedOrNotFound":
			return true
		}
	}

	message := err.Error()
	return strings.Contains(message, "http status code: 404") ||
		strings.Contains(message, "NotFound") ||
		strings.Contains(message, "NotAuthorizedOrNotFound")
}

func redisClusterDeleteGuardIsConflict(err error) bool {
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.GetHTTPStatusCode() == 409
	}

	var conflictErr errorutil.ConflictOciError
	if errors.As(err, &conflictErr) {
		return true
	}

	return strings.Contains(err.Error(), "http status code: 409")
}
