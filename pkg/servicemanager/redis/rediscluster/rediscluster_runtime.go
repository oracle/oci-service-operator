/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package rediscluster

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	redissdk "github.com/oracle/oci-go-sdk/v65/redis"
	redisv1beta1 "github.com/oracle/oci-service-operator/api/redis/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
)

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

type redisWorkRequestClient interface {
	GetWorkRequest(ctx context.Context, request redissdk.GetWorkRequestRequest) (redissdk.GetWorkRequestResponse, error)
}

func init() {
	registerRedisClusterRuntimeHooksMutator(func(manager *RedisClusterServiceManager, hooks *RedisClusterRuntimeHooks) {
		workRequestClient, initErr := newRedisWorkRequestClient(manager)
		applyRedisClusterRuntimeHooks(manager, hooks, workRequestClient, initErr)
	})
}

func newRedisWorkRequestClient(manager *RedisClusterServiceManager) (redisWorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("RedisCluster service manager is nil")
	}
	client, err := redissdk.NewRedisClusterClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyRedisClusterRuntimeHooks(
	manager *RedisClusterServiceManager,
	hooks *RedisClusterRuntimeHooks,
	workRequestClient redisWorkRequestClient,
	initErr error,
) {
	if hooks == nil {
		return
	}

	hooks.BuildCreateBody = func(_ context.Context, resource *redisv1beta1.RedisCluster, _ string) (any, error) {
		if resource == nil {
			return nil, fmt.Errorf("RedisCluster resource is nil")
		}
		return buildCreateRedisClusterDetails(resource.Spec), nil
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *redisv1beta1.RedisCluster,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildRedisUpdateBody(resource, currentResponse)
	}
	hooks.StatusHooks.ProjectStatus = projectRedisClusterStatus
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedRedisIdentity
	hooks.Async.Adapter = redisWorkRequestAsyncAdapter
	hooks.Async.GetWorkRequest = func(ctx context.Context, workRequestID string) (any, error) {
		return getRedisWorkRequest(ctx, workRequestClient, initErr, workRequestID)
	}
	hooks.Async.ResolvePhase = resolveRedisGeneratedWorkRequestPhase
	hooks.Async.RecoverResourceID = recoverRedisClusterIDFromGeneratedWorkRequest
	hooks.Async.Message = redisGeneratedWorkRequestMessage
	if manager != nil {
		hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, func(delegate RedisClusterServiceClient) RedisClusterServiceClient {
			return newRedisDeleteGuardClient(manager, delegate)
		})
	}
}

func buildRedisUpdateBody(
	resource *redisv1beta1.RedisCluster,
	currentResponse any,
) (redissdk.UpdateRedisClusterDetails, bool, error) {
	if resource == nil {
		return redissdk.UpdateRedisClusterDetails{}, false, fmt.Errorf("RedisCluster resource is nil")
	}

	current, ok := redisClusterFromResponse(currentResponse)
	if !ok {
		return redissdk.UpdateRedisClusterDetails{}, false, fmt.Errorf("current RedisCluster response does not expose a RedisCluster body")
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
		return redissdk.UpdateRedisClusterDetails{}, false, nil
	}
	return updateDetails, true, nil
}

func projectRedisClusterStatus(resource *redisv1beta1.RedisCluster, response any) error {
	if resource == nil {
		return fmt.Errorf("RedisCluster resource is nil")
	}

	current, ok := redisClusterFromResponse(response)
	if !ok {
		return nil
	}

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

func redisClusterFromResponse(response any) (redissdk.RedisCluster, bool) {
	switch current := response.(type) {
	case redissdk.CreateRedisClusterResponse:
		return current.RedisCluster, true
	case *redissdk.CreateRedisClusterResponse:
		if current == nil {
			return redissdk.RedisCluster{}, false
		}
		return current.RedisCluster, true
	case redissdk.GetRedisClusterResponse:
		return current.RedisCluster, true
	case *redissdk.GetRedisClusterResponse:
		if current == nil {
			return redissdk.RedisCluster{}, false
		}
		return current.RedisCluster, true
	case redissdk.RedisCluster:
		return current, true
	case *redissdk.RedisCluster:
		if current == nil {
			return redissdk.RedisCluster{}, false
		}
		return *current, true
	case redissdk.RedisClusterSummary:
		return redisClusterFromSummary(current), true
	case *redissdk.RedisClusterSummary:
		if current == nil {
			return redissdk.RedisCluster{}, false
		}
		return redisClusterFromSummary(*current), true
	default:
		return redissdk.RedisCluster{}, false
	}
}

func redisClusterFromSummary(summary redissdk.RedisClusterSummary) redissdk.RedisCluster {
	return redissdk.RedisCluster{
		Id:                        summary.Id,
		DisplayName:               summary.DisplayName,
		CompartmentId:             summary.CompartmentId,
		NodeCount:                 summary.NodeCount,
		NodeMemoryInGBs:           summary.NodeMemoryInGBs,
		PrimaryFqdn:               summary.PrimaryFqdn,
		PrimaryEndpointIpAddress:  summary.PrimaryEndpointIpAddress,
		ReplicasFqdn:              summary.ReplicasFqdn,
		ReplicasEndpointIpAddress: summary.ReplicasEndpointIpAddress,
		SoftwareVersion:           summary.SoftwareVersion,
		SubnetId:                  summary.SubnetId,
		LifecycleState:            summary.LifecycleState,
		LifecycleDetails:          summary.LifecycleDetails,
		TimeCreated:               summary.TimeCreated,
		TimeUpdated:               summary.TimeUpdated,
		FreeformTags:              summary.FreeformTags,
		DefinedTags:               summary.DefinedTags,
		SystemTags:                summary.SystemTags,
	}
}

func clearTrackedRedisIdentity(resource *redisv1beta1.RedisCluster) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus = shared.OSOKStatus{}
}

func getRedisWorkRequest(
	ctx context.Context,
	client redisWorkRequestClient,
	initErr error,
	workRequestID string,
) (any, error) {
	if initErr != nil {
		return nil, fmt.Errorf("initialize RedisCluster OCI client: %w", initErr)
	}
	if client == nil {
		return nil, fmt.Errorf("RedisCluster OCI client is not configured")
	}

	response, err := client.GetWorkRequest(ctx, redissdk.GetWorkRequestRequest{
		WorkRequestId: common.String(strings.TrimSpace(workRequestID)),
	})
	if err != nil {
		return nil, err
	}
	return response.WorkRequest, nil
}

func resolveRedisGeneratedWorkRequestPhase(workRequest any) (shared.OSOKAsyncPhase, bool, error) {
	redisWorkRequest, err := redisWorkRequestFromAny(workRequest)
	if err != nil {
		return "", false, err
	}
	phase, ok := redisWorkRequestPhaseFromOperationType(redisWorkRequest.OperationType)
	return phase, ok, nil
}

func recoverRedisClusterIDFromGeneratedWorkRequest(
	_ *redisv1beta1.RedisCluster,
	workRequest any,
	phase shared.OSOKAsyncPhase,
) (string, error) {
	redisWorkRequest, err := redisWorkRequestFromAny(workRequest)
	if err != nil {
		return "", err
	}
	return resolveRedisClusterIDFromWorkRequest(redisWorkRequest, redisWorkRequestActionForPhase(phase))
}

func redisWorkRequestFromAny(workRequest any) (redissdk.WorkRequest, error) {
	switch current := workRequest.(type) {
	case redissdk.WorkRequest:
		return current, nil
	case *redissdk.WorkRequest:
		if current == nil {
			return redissdk.WorkRequest{}, fmt.Errorf("RedisCluster work request is nil")
		}
		return *current, nil
	default:
		return redissdk.WorkRequest{}, fmt.Errorf("unexpected RedisCluster work request type %T", workRequest)
	}
}

func redisWorkRequestActionForPhase(phase shared.OSOKAsyncPhase) redissdk.ActionTypeEnum {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return redissdk.ActionTypeCreated
	case shared.OSOKAsyncPhaseUpdate:
		return redissdk.ActionTypeUpdated
	case shared.OSOKAsyncPhaseDelete:
		return redissdk.ActionTypeDeleted
	default:
		return ""
	}
}

func redisGeneratedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	redisWorkRequest, err := redisWorkRequestFromAny(workRequest)
	if err != nil {
		return ""
	}
	return redisWorkRequestMessage(phase, redisWorkRequest)
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

	fallbackPhase := explicitPhase
	if fallbackPhase == "" {
		fallbackPhase = servicemanager.ResolveAsyncPhase(status, "")
	}
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
