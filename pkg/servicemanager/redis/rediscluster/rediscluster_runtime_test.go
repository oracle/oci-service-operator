/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package rediscluster

import (
	"context"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	redissdk "github.com/oracle/oci-go-sdk/v65/redis"
	redisv1beta1 "github.com/oracle/oci-service-operator/api/redis/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeRedisOCIClient struct {
	createFn         func(context.Context, redissdk.CreateRedisClusterRequest) (redissdk.CreateRedisClusterResponse, error)
	getFn            func(context.Context, redissdk.GetRedisClusterRequest) (redissdk.GetRedisClusterResponse, error)
	listFn           func(context.Context, redissdk.ListRedisClustersRequest) (redissdk.ListRedisClustersResponse, error)
	updateFn         func(context.Context, redissdk.UpdateRedisClusterRequest) (redissdk.UpdateRedisClusterResponse, error)
	deleteFn         func(context.Context, redissdk.DeleteRedisClusterRequest) (redissdk.DeleteRedisClusterResponse, error)
	getWorkRequestFn func(context.Context, redissdk.GetWorkRequestRequest) (redissdk.GetWorkRequestResponse, error)
}

func (f *fakeRedisOCIClient) CreateRedisCluster(ctx context.Context, req redissdk.CreateRedisClusterRequest) (redissdk.CreateRedisClusterResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return redissdk.CreateRedisClusterResponse{}, nil
}

func (f *fakeRedisOCIClient) GetRedisCluster(ctx context.Context, req redissdk.GetRedisClusterRequest) (redissdk.GetRedisClusterResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return redissdk.GetRedisClusterResponse{}, nil
}

func (f *fakeRedisOCIClient) ListRedisClusters(ctx context.Context, req redissdk.ListRedisClustersRequest) (redissdk.ListRedisClustersResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return redissdk.ListRedisClustersResponse{}, nil
}

func (f *fakeRedisOCIClient) UpdateRedisCluster(ctx context.Context, req redissdk.UpdateRedisClusterRequest) (redissdk.UpdateRedisClusterResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return redissdk.UpdateRedisClusterResponse{}, nil
}

func (f *fakeRedisOCIClient) DeleteRedisCluster(ctx context.Context, req redissdk.DeleteRedisClusterRequest) (redissdk.DeleteRedisClusterResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return redissdk.DeleteRedisClusterResponse{}, nil
}

func (f *fakeRedisOCIClient) GetWorkRequest(ctx context.Context, req redissdk.GetWorkRequestRequest) (redissdk.GetWorkRequestResponse, error) {
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, req)
	}
	return redissdk.GetWorkRequestResponse{}, nil
}

type fakeRedisServiceError struct {
	statusCode   int
	code         string
	message      string
	opcRequestID string
}

func (f fakeRedisServiceError) Error() string          { return f.message }
func (f fakeRedisServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeRedisServiceError) GetMessage() string     { return f.message }
func (f fakeRedisServiceError) GetCode() string        { return f.code }
func (f fakeRedisServiceError) GetOpcRequestID() string {
	return f.opcRequestID
}

func newRedisTestManager(client redisOCIClient) *RedisClusterServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewRedisClusterServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(&redisRuntimeClient{
			manager: manager,
			client:  client,
		})
	}
	return manager
}

func makeSpecRedisCluster() *redisv1beta1.RedisCluster {
	return &redisv1beta1.RedisCluster{
		Spec: redisv1beta1.RedisClusterSpec{
			DisplayName:     "redis-sample",
			CompartmentId:   "ocid1.compartment.oc1..example",
			NodeCount:       2,
			SoftwareVersion: "V7_0_5",
			NodeMemoryInGBs: 8,
			SubnetId:        "ocid1.subnet.oc1..example",
			FreeformTags:    map[string]string{"env": "dev"},
			DefinedTags:     map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeSDKRedisCluster(id, displayName string, state redissdk.RedisClusterLifecycleStateEnum) redissdk.RedisCluster {
	return redissdk.RedisCluster{
		Id:                        common.String(id),
		DisplayName:               common.String(displayName),
		CompartmentId:             common.String("ocid1.compartment.oc1..example"),
		NodeCount:                 common.Int(2),
		NodeMemoryInGBs:           common.Float32(8),
		PrimaryFqdn:               common.String("redis-1.example.internal"),
		PrimaryEndpointIpAddress:  common.String("10.0.0.10"),
		ReplicasFqdn:              common.String("redis-replicas.example.internal"),
		ReplicasEndpointIpAddress: common.String("10.0.0.11"),
		SoftwareVersion:           redissdk.RedisClusterSoftwareVersionV705,
		SubnetId:                  common.String("ocid1.subnet.oc1..example"),
		NodeCollection: &redissdk.NodeCollection{
			Items: []redissdk.Node{
				{
					PrivateEndpointFqdn:      common.String("redis-1.example.internal"),
					PrivateEndpointIpAddress: common.String("10.0.0.10"),
					DisplayName:              common.String("node-1"),
				},
			},
		},
		LifecycleState:   state,
		LifecycleDetails: common.String("ready"),
		TimeCreated:      &common.SDKTime{Time: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		TimeUpdated:      &common.SDKTime{Time: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)},
		FreeformTags:     map[string]string{"env": "dev"},
		DefinedTags:      map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:       map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
	}
}

func makeRedisWorkRequest(
	id string,
	status redissdk.OperationStatusEnum,
	operation redissdk.OperationTypeEnum,
	action redissdk.ActionTypeEnum,
	clusterID string,
) redissdk.WorkRequest {
	return redissdk.WorkRequest{
		Id:            common.String(id),
		Status:        status,
		OperationType: operation,
		CompartmentId: common.String("ocid1.compartment.oc1..example"),
		Resources: []redissdk.WorkRequestResource{
			{
				EntityType: common.String("redis"),
				ActionType: action,
				Identifier: common.String(clusterID),
				EntityUri:  common.String("/redisClusters/" + clusterID),
			},
		},
		PercentComplete: common.Float32(42),
		TimeAccepted:    &common.SDKTime{Time: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
	}
}

func seedRedisAsyncWorkRequest(resource *redisv1beta1.RedisCluster, phase shared.OSOKAsyncPhase, workRequestID string) {
	resource.Status.OsokStatus = shared.OSOKStatus{
		Async: shared.OSOKAsyncTracker{
			Current: &shared.OSOKAsyncOperation{
				Source:          shared.OSOKAsyncSourceWorkRequest,
				Phase:           phase,
				WorkRequestID:   workRequestID,
				NormalizedClass: shared.OSOKAsyncClassPending,
			},
		},
	}
}

func TestRedisWorkRequestAsyncOperationMapsKnownStatusesAndOperationTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    redissdk.OperationStatusEnum
		operation redissdk.OperationTypeEnum
		action    redissdk.ActionTypeEnum
		phase     shared.OSOKAsyncPhase
		wantClass shared.OSOKAsyncNormalizedClass
	}{
		{
			name:      "accepted create pending",
			status:    redissdk.OperationStatusAccepted,
			operation: redissdk.OperationTypeCreateRedisCluster,
			action:    redissdk.ActionTypeInProgress,
			phase:     shared.OSOKAsyncPhaseCreate,
			wantClass: shared.OSOKAsyncClassPending,
		},
		{
			name:      "waiting update pending",
			status:    redissdk.OperationStatusWaiting,
			operation: redissdk.OperationTypeUpdateRedisCluster,
			action:    redissdk.ActionTypeInProgress,
			phase:     shared.OSOKAsyncPhaseUpdate,
			wantClass: shared.OSOKAsyncClassPending,
		},
		{
			name:      "needs attention delete attention",
			status:    redissdk.OperationStatusNeedsAttention,
			operation: redissdk.OperationTypeDeleteRedisCluster,
			action:    redissdk.ActionTypeFailed,
			phase:     shared.OSOKAsyncPhaseDelete,
			wantClass: shared.OSOKAsyncClassAttention,
		},
		{
			name:      "canceled update canceled",
			status:    redissdk.OperationStatusCanceled,
			operation: redissdk.OperationTypeUpdateRedisCluster,
			action:    redissdk.ActionTypeFailed,
			phase:     shared.OSOKAsyncPhaseUpdate,
			wantClass: shared.OSOKAsyncClassCanceled,
		},
		{
			name:      "succeeded delete succeeded",
			status:    redissdk.OperationStatusSucceeded,
			operation: redissdk.OperationTypeDeleteRedisCluster,
			action:    redissdk.ActionTypeDeleted,
			phase:     shared.OSOKAsyncPhaseDelete,
			wantClass: shared.OSOKAsyncClassSucceeded,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			workRequest := makeRedisWorkRequest("wr-redis", tt.status, tt.operation, tt.action, "ocid1.rediscluster.oc1..existing")
			current, err := redisWorkRequestAsyncOperation(makeSpecRedisCluster(), workRequest, tt.phase)
			assert.NoError(t, err)
			if assert.NotNil(t, current) {
				assert.Equal(t, shared.OSOKAsyncSourceWorkRequest, current.Source)
				assert.Equal(t, tt.phase, current.Phase)
				assert.Equal(t, "wr-redis", current.WorkRequestID)
				assert.Equal(t, string(tt.status), current.RawStatus)
				assert.Equal(t, string(tt.operation), current.RawOperationType)
				assert.Equal(t, tt.wantClass, current.NormalizedClass)
				assert.Equal(t, "RedisCluster "+string(tt.phase)+" work request wr-redis is "+string(tt.status), current.Message)
				if assert.NotNil(t, current.PercentComplete) {
					assert.Equal(t, float32(42), *current.PercentComplete)
				}
			}
		})
	}
}

func TestRedisWorkRequestStatusCoverage(t *testing.T) {
	t.Parallel()

	for _, status := range redissdk.GetOperationStatusEnumValues() {
		wantClass := shared.OSOKAsyncNormalizedClass("")
		switch status {
		case redissdk.OperationStatusAccepted,
			redissdk.OperationStatusInProgress,
			redissdk.OperationStatusWaiting,
			redissdk.OperationStatusCanceling:
			wantClass = shared.OSOKAsyncClassPending
		case redissdk.OperationStatusSucceeded:
			wantClass = shared.OSOKAsyncClassSucceeded
		case redissdk.OperationStatusFailed:
			wantClass = shared.OSOKAsyncClassFailed
		case redissdk.OperationStatusCanceled:
			wantClass = shared.OSOKAsyncClassCanceled
		case redissdk.OperationStatusNeedsAttention:
			wantClass = shared.OSOKAsyncClassAttention
		default:
			t.Fatalf("unhandled Redis operation status enum %q", status)
		}

		gotClass, err := redisWorkRequestAsyncAdapter.Normalize(string(status))
		assert.NoError(t, err)
		assert.Equal(t, wantClass, gotClass)
	}
}

func TestRedisWorkRequestOperationTypeCoverage(t *testing.T) {
	t.Parallel()

	for _, operation := range redissdk.GetOperationTypeEnumValues() {
		switch operation {
		case redissdk.OperationTypeCreateRedisCluster:
			phase, ok := redisWorkRequestPhaseFromOperationType(operation)
			assert.True(t, ok)
			assert.Equal(t, shared.OSOKAsyncPhaseCreate, phase)
		case redissdk.OperationTypeUpdateRedisCluster:
			phase, ok := redisWorkRequestPhaseFromOperationType(operation)
			assert.True(t, ok)
			assert.Equal(t, shared.OSOKAsyncPhaseUpdate, phase)
		case redissdk.OperationTypeDeleteRedisCluster:
			phase, ok := redisWorkRequestPhaseFromOperationType(operation)
			assert.True(t, ok)
			assert.Equal(t, shared.OSOKAsyncPhaseDelete, phase)
		case redissdk.OperationTypeMoveRedisCluster,
			redissdk.OperationTypeFailoverRedisCluster,
			redissdk.OperationTypeCreateRedisConfigSet,
			redissdk.OperationTypeUpdateRedisConfigSet,
			redissdk.OperationTypeDeleteRedisConfigSet,
			redissdk.OperationTypeMoveRedisConfigSet:
			phase, ok := redisWorkRequestPhaseFromOperationType(operation)
			assert.False(t, ok)
			assert.Equal(t, shared.OSOKAsyncPhase(""), phase)
		default:
			t.Fatalf("unhandled Redis operation type enum %q", operation)
		}
	}
}

func TestRedisCreateCapturesWorkRequestInSharedAsyncStatus(t *testing.T) {
	t.Parallel()

	clusterID := "ocid1.rediscluster.oc1..created"
	manager := newRedisTestManager(&fakeRedisOCIClient{
		createFn: func(_ context.Context, req redissdk.CreateRedisClusterRequest) (redissdk.CreateRedisClusterResponse, error) {
			assert.Equal(t, "redis-sample", stringValue(req.CreateRedisClusterDetails.DisplayName))
			assert.Equal(t, "ocid1.compartment.oc1..example", stringValue(req.CreateRedisClusterDetails.CompartmentId))
			assert.Equal(t, 2, intValue(req.CreateRedisClusterDetails.NodeCount))
			assert.Equal(t, float32(8), float32Value(req.CreateRedisClusterDetails.NodeMemoryInGBs))
			assert.Equal(t, "ocid1.subnet.oc1..example", stringValue(req.CreateRedisClusterDetails.SubnetId))
			return redissdk.CreateRedisClusterResponse{
				RedisCluster:     makeSDKRedisCluster(clusterID, "redis-sample", redissdk.RedisClusterLifecycleStateCreating),
				OpcRequestId:     common.String("opc-create-1"),
				OpcWorkRequestId: common.String("wr-create-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req redissdk.GetWorkRequestRequest) (redissdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-create-1", stringValue(req.WorkRequestId))
			return redissdk.GetWorkRequestResponse{
				WorkRequest: makeRedisWorkRequest(
					"wr-create-1",
					redissdk.OperationStatusWaiting,
					redissdk.OperationTypeCreateRedisCluster,
					redissdk.ActionTypeInProgress,
					clusterID,
				),
			}, nil
		},
	})

	resource := makeSpecRedisCluster()
	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, response.IsSuccessful)
	assert.True(t, response.ShouldRequeue)
	assert.Equal(t, "opc-create-1", resource.Status.OsokStatus.OpcRequestID)
	assert.Equal(t, clusterID, resource.Status.Id)
	if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
		assert.Equal(t, shared.OSOKAsyncSourceWorkRequest, resource.Status.OsokStatus.Async.Current.Source)
		assert.Equal(t, shared.OSOKAsyncPhaseCreate, resource.Status.OsokStatus.Async.Current.Phase)
		assert.Equal(t, "wr-create-1", resource.Status.OsokStatus.Async.Current.WorkRequestID)
		assert.Equal(t, "WAITING", resource.Status.OsokStatus.Async.Current.RawStatus)
		assert.Equal(t, "CREATE_REDIS_CLUSTER", resource.Status.OsokStatus.Async.Current.RawOperationType)
		assert.Equal(t, shared.OSOKAsyncClassPending, resource.Status.OsokStatus.Async.Current.NormalizedClass)
		assert.Equal(t, "RedisCluster create work request wr-create-1 is WAITING", resource.Status.OsokStatus.Async.Current.Message)
		if assert.NotNil(t, resource.Status.OsokStatus.Async.Current.PercentComplete) {
			assert.Equal(t, float32(42), *resource.Status.OsokStatus.Async.Current.PercentComplete)
		}
	}
}

func TestRedisCreateErrorCapturesOpcRequestID(t *testing.T) {
	t.Parallel()

	manager := newRedisTestManager(&fakeRedisOCIClient{
		createFn: func(_ context.Context, _ redissdk.CreateRedisClusterRequest) (redissdk.CreateRedisClusterResponse, error) {
			return redissdk.CreateRedisClusterResponse{}, fakeRedisServiceError{
				statusCode:   409,
				code:         errorutil.IncorrectState,
				message:      "create conflict",
				opcRequestID: "opc-create-conflict",
			}
		},
	})

	resource := makeSpecRedisCluster()
	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.Error(t, err)
	assert.False(t, response.IsSuccessful)
	assert.Equal(t, "opc-create-conflict", resource.Status.OsokStatus.OpcRequestID)
	assert.Equal(t, err.Error(), resource.Status.OsokStatus.Message)
	assert.Equal(t, string(shared.Failed), resource.Status.OsokStatus.Reason)
}

func TestRedisResumeCreateRecoversIdentityFromSucceededWorkRequest(t *testing.T) {
	t.Parallel()

	clusterID := "ocid1.rediscluster.oc1..existing"
	manager := newRedisTestManager(&fakeRedisOCIClient{
		getWorkRequestFn: func(_ context.Context, req redissdk.GetWorkRequestRequest) (redissdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-create-2", stringValue(req.WorkRequestId))
			return redissdk.GetWorkRequestResponse{
				WorkRequest: makeRedisWorkRequest(
					"wr-create-2",
					redissdk.OperationStatusSucceeded,
					redissdk.OperationTypeCreateRedisCluster,
					redissdk.ActionTypeCreated,
					clusterID,
				),
			}, nil
		},
		getFn: func(_ context.Context, req redissdk.GetRedisClusterRequest) (redissdk.GetRedisClusterResponse, error) {
			assert.Equal(t, clusterID, stringValue(req.RedisClusterId))
			return redissdk.GetRedisClusterResponse{
				RedisCluster: makeSDKRedisCluster(clusterID, "redis-sample", redissdk.RedisClusterLifecycleStateActive),
			}, nil
		},
	})

	resource := makeSpecRedisCluster()
	seedRedisAsyncWorkRequest(resource, shared.OSOKAsyncPhaseCreate, "wr-create-2")
	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, response.IsSuccessful)
	assert.False(t, response.ShouldRequeue)
	assert.Equal(t, clusterID, resource.Status.Id)
	assert.Nil(t, resource.Status.OsokStatus.Async.Current)
	assert.Equal(t, string(shared.Active), resource.Status.OsokStatus.Reason)
}

func TestRedisUpdateCapturesWorkRequestInSharedAsyncStatus(t *testing.T) {
	t.Parallel()

	clusterID := "ocid1.rediscluster.oc1..existing"
	current := makeSDKRedisCluster(clusterID, "redis-existing", redissdk.RedisClusterLifecycleStateActive)
	manager := newRedisTestManager(&fakeRedisOCIClient{
		getFn: func(_ context.Context, req redissdk.GetRedisClusterRequest) (redissdk.GetRedisClusterResponse, error) {
			assert.Equal(t, clusterID, stringValue(req.RedisClusterId))
			return redissdk.GetRedisClusterResponse{RedisCluster: current}, nil
		},
		updateFn: func(_ context.Context, req redissdk.UpdateRedisClusterRequest) (redissdk.UpdateRedisClusterResponse, error) {
			assert.Equal(t, clusterID, stringValue(req.RedisClusterId))
			assert.Equal(t, "redis-sample", stringValue(req.UpdateRedisClusterDetails.DisplayName))
			return redissdk.UpdateRedisClusterResponse{OpcWorkRequestId: common.String("wr-update-1")}, nil
		},
		getWorkRequestFn: func(_ context.Context, req redissdk.GetWorkRequestRequest) (redissdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-update-1", stringValue(req.WorkRequestId))
			return redissdk.GetWorkRequestResponse{
				WorkRequest: makeRedisWorkRequest(
					"wr-update-1",
					redissdk.OperationStatusInProgress,
					redissdk.OperationTypeUpdateRedisCluster,
					redissdk.ActionTypeInProgress,
					clusterID,
				),
			}, nil
		},
	})

	resource := makeSpecRedisCluster()
	resource.Status.Id = clusterID
	resource.Status.OsokStatus.Ocid = shared.OCID(clusterID)

	response, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, response.IsSuccessful)
	assert.True(t, response.ShouldRequeue)
	if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
		assert.Equal(t, shared.OSOKAsyncPhaseUpdate, resource.Status.OsokStatus.Async.Current.Phase)
		assert.Equal(t, "wr-update-1", resource.Status.OsokStatus.Async.Current.WorkRequestID)
		assert.Equal(t, "IN_PROGRESS", resource.Status.OsokStatus.Async.Current.RawStatus)
		assert.Equal(t, "UPDATE_REDIS_CLUSTER", resource.Status.OsokStatus.Async.Current.RawOperationType)
		assert.Equal(t, shared.OSOKAsyncClassPending, resource.Status.OsokStatus.Async.Current.NormalizedClass)
	}
}

func TestRedisDeleteMarksDeletedAfterSucceededWorkRequest(t *testing.T) {
	t.Parallel()

	clusterID := "ocid1.rediscluster.oc1..existing"
	manager := newRedisTestManager(&fakeRedisOCIClient{
		getWorkRequestFn: func(_ context.Context, req redissdk.GetWorkRequestRequest) (redissdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-delete-1", stringValue(req.WorkRequestId))
			return redissdk.GetWorkRequestResponse{
				WorkRequest: makeRedisWorkRequest(
					"wr-delete-1",
					redissdk.OperationStatusSucceeded,
					redissdk.OperationTypeDeleteRedisCluster,
					redissdk.ActionTypeDeleted,
					clusterID,
				),
			}, nil
		},
		getFn: func(_ context.Context, req redissdk.GetRedisClusterRequest) (redissdk.GetRedisClusterResponse, error) {
			assert.Equal(t, clusterID, stringValue(req.RedisClusterId))
			return redissdk.GetRedisClusterResponse{}, fakeRedisServiceError{
				statusCode: 404,
				code:       "NotAuthorizedOrNotFound",
				message:    "cluster gone",
			}
		},
	})

	resource := makeSpecRedisCluster()
	resource.Status.Id = clusterID
	resource.Status.OsokStatus.Ocid = shared.OCID(clusterID)
	seedRedisAsyncWorkRequest(resource, shared.OSOKAsyncPhaseDelete, "wr-delete-1")

	deleted, err := manager.Delete(context.Background(), resource)
	assert.NoError(t, err)
	assert.True(t, deleted)
	assert.Nil(t, resource.Status.OsokStatus.Async.Current)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}
