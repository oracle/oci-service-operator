/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package queue

import (
	"context"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	queuesdk "github.com/oracle/oci-go-sdk/v65/queue"
	queuev1beta1 "github.com/oracle/oci-service-operator/api/queue/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeQueueOCIClient struct {
	createFn         func(context.Context, queuesdk.CreateQueueRequest) (queuesdk.CreateQueueResponse, error)
	getFn            func(context.Context, queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error)
	listFn           func(context.Context, queuesdk.ListQueuesRequest) (queuesdk.ListQueuesResponse, error)
	updateFn         func(context.Context, queuesdk.UpdateQueueRequest) (queuesdk.UpdateQueueResponse, error)
	deleteFn         func(context.Context, queuesdk.DeleteQueueRequest) (queuesdk.DeleteQueueResponse, error)
	getWorkRequestFn func(context.Context, queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error)
}

func (f *fakeQueueOCIClient) CreateQueue(ctx context.Context, req queuesdk.CreateQueueRequest) (queuesdk.CreateQueueResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	return queuesdk.CreateQueueResponse{}, nil
}

func (f *fakeQueueOCIClient) GetQueue(ctx context.Context, req queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	return queuesdk.GetQueueResponse{}, nil
}

func (f *fakeQueueOCIClient) ListQueues(ctx context.Context, req queuesdk.ListQueuesRequest) (queuesdk.ListQueuesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return queuesdk.ListQueuesResponse{}, nil
}

func (f *fakeQueueOCIClient) UpdateQueue(ctx context.Context, req queuesdk.UpdateQueueRequest) (queuesdk.UpdateQueueResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return queuesdk.UpdateQueueResponse{}, nil
}

func (f *fakeQueueOCIClient) DeleteQueue(ctx context.Context, req queuesdk.DeleteQueueRequest) (queuesdk.DeleteQueueResponse, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return queuesdk.DeleteQueueResponse{}, nil
}

func (f *fakeQueueOCIClient) GetWorkRequest(ctx context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
	if f.getWorkRequestFn != nil {
		return f.getWorkRequestFn(ctx, req)
	}
	return queuesdk.GetWorkRequestResponse{}, nil
}

type fakeQueueServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f fakeQueueServiceError) Error() string          { return f.message }
func (f fakeQueueServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeQueueServiceError) GetMessage() string     { return f.message }
func (f fakeQueueServiceError) GetCode() string        { return f.code }
func (f fakeQueueServiceError) GetOpcRequestID() string {
	return ""
}

func newQueueTestManager(client queueOCIClient) *QueueServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewQueueServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	if client != nil {
		manager.WithClient(&queueRuntimeClient{
			manager: manager,
			client:  client,
		})
	}
	return manager
}

func makeSpecQueue() *queuev1beta1.Queue {
	return &queuev1beta1.Queue{
		Spec: queuev1beta1.QueueSpec{
			DisplayName:                  "queue-sample",
			CompartmentId:                "ocid1.compartment.oc1..example",
			RetentionInSeconds:           1200,
			VisibilityInSeconds:          30,
			TimeoutInSeconds:             20,
			ChannelConsumptionLimit:      100,
			DeadLetterQueueDeliveryCount: 5,
			CustomEncryptionKeyId:        "ocid1.key.oc1..example",
			FreeformTags:                 map[string]string{"env": "dev"},
			DefinedTags:                  map[string]shared.MapValue{"Operations": {"CostCenter": "42"}},
		},
	}
}

func makeSDKQueue(id, displayName string, state queuesdk.QueueLifecycleStateEnum) queuesdk.Queue {
	return queuesdk.Queue{
		Id:                           common.String(id),
		CompartmentId:                common.String("ocid1.compartment.oc1..example"),
		TimeCreated:                  &common.SDKTime{Time: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		TimeUpdated:                  &common.SDKTime{Time: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)},
		LifecycleState:               state,
		MessagesEndpoint:             common.String("https://cell-1.queue.messaging.us-phoenix-1.oci.oraclecloud.com"),
		RetentionInSeconds:           common.Int(1200),
		VisibilityInSeconds:          common.Int(30),
		TimeoutInSeconds:             common.Int(20),
		DeadLetterQueueDeliveryCount: common.Int(5),
		DisplayName:                  common.String(displayName),
		CustomEncryptionKeyId:        common.String("ocid1.key.oc1..example"),
		FreeformTags:                 map[string]string{"env": "dev"},
		DefinedTags:                  map[string]map[string]interface{}{"Operations": {"CostCenter": "42"}},
		SystemTags:                   map[string]map[string]interface{}{"orcl-cloud": {"free-tier-retained": "true"}},
		ChannelConsumptionLimit:      common.Int(100),
	}
}

func makeWorkRequest(id string, status queuesdk.OperationStatusEnum, action queuesdk.ActionTypeEnum, queueID string) queuesdk.WorkRequest {
	return queuesdk.WorkRequest{
		Id:            common.String(id),
		Status:        status,
		OperationType: queuesdk.OperationTypeCreateQueue,
		CompartmentId: common.String("ocid1.compartment.oc1..example"),
		Resources: []queuesdk.WorkRequestResource{
			{
				EntityType: common.String("queue"),
				ActionType: action,
				Identifier: common.String(queueID),
			},
		},
		PercentComplete: common.Float32(100),
		TimeAccepted:    &common.SDKTime{Time: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
	}
}

func TestQueueRuntime_CreateAcceptedPersistsWorkRequestAndRequeues(t *testing.T) {
	var captured queuesdk.CreateQueueRequest
	manager := newQueueTestManager(&fakeQueueOCIClient{
		createFn: func(_ context.Context, req queuesdk.CreateQueueRequest) (queuesdk.CreateQueueResponse, error) {
			captured = req
			return queuesdk.CreateQueueResponse{
				OpcWorkRequestId: common.String("wr-create-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-create-1", *req.WorkRequestId)
			return queuesdk.GetWorkRequestResponse{
				WorkRequest: makeWorkRequest("wr-create-1", queuesdk.OperationStatusAccepted, queuesdk.ActionTypeCreated, ""),
			}, nil
		},
	})

	resource := makeSpecQueue()
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, resp.ShouldRequeue)
	assert.Equal(t, queueRequeueDuration, resp.RequeueDuration)
	assert.Equal(t, "wr-create-1", resource.Status.CreateWorkRequestId)
	assert.Equal(t, "queue-sample", *captured.DisplayName)
	assert.Equal(t, "ocid1.compartment.oc1..example", *captured.CompartmentId)
	assert.Equal(t, 1200, *captured.RetentionInSeconds)
}

func TestQueueRuntime_ResumeCreateRecoversQueueIDAndProjectsStatus(t *testing.T) {
	manager := newQueueTestManager(&fakeQueueOCIClient{
		getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-create-2", *req.WorkRequestId)
			return queuesdk.GetWorkRequestResponse{
				WorkRequest: makeWorkRequest("wr-create-2", queuesdk.OperationStatusSucceeded, queuesdk.ActionTypeCreated, "ocid1.queue.oc1..created"),
			}, nil
		},
		getFn: func(_ context.Context, req queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
			assert.Equal(t, "ocid1.queue.oc1..created", *req.QueueId)
			return queuesdk.GetQueueResponse{
				Queue: makeSDKQueue("ocid1.queue.oc1..created", "queue-sample", queuesdk.QueueLifecycleStateActive),
			}, nil
		},
	})

	resource := makeSpecQueue()
	resource.Status.CreateWorkRequestId = "wr-create-2"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, "", resource.Status.CreateWorkRequestId)
	assert.Equal(t, "ocid1.queue.oc1..created", resource.Status.Id)
	assert.Equal(t, "ocid1.queue.oc1..created", string(resource.Status.OsokStatus.Ocid))
	assert.Equal(t, "ACTIVE", resource.Status.LifecycleState)
	assert.Equal(t, "https://cell-1.queue.messaging.us-phoenix-1.oci.oraclecloud.com", resource.Status.MessagesEndpoint)
}

func TestQueueRuntime_ObserveNoOpWhenStateMatches(t *testing.T) {
	getCalls := 0
	updateCalls := 0
	manager := newQueueTestManager(&fakeQueueOCIClient{
		getFn: func(_ context.Context, req queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.queue.oc1..existing", *req.QueueId)
			return queuesdk.GetQueueResponse{
				Queue: makeSDKQueue("ocid1.queue.oc1..existing", "queue-sample", queuesdk.QueueLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, _ queuesdk.UpdateQueueRequest) (queuesdk.UpdateQueueResponse, error) {
			updateCalls++
			return queuesdk.UpdateQueueResponse{}, nil
		},
	})

	resource := makeSpecQueue()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 0, updateCalls)
}

func TestQueueRuntime_MutableUpdateDriftTriggersWorkRequestAndClearCustomEncryptionKey(t *testing.T) {
	var captured queuesdk.UpdateQueueRequest
	getCalls := 0
	manager := newQueueTestManager(&fakeQueueOCIClient{
		getFn: func(_ context.Context, req queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.queue.oc1..existing", *req.QueueId)
			current := makeSDKQueue("ocid1.queue.oc1..existing", "old-name", queuesdk.QueueLifecycleStateActive)
			current.VisibilityInSeconds = common.Int(10)
			current.CustomEncryptionKeyId = common.String("ocid1.key.oc1..old")
			if getCalls == 1 {
				return queuesdk.GetQueueResponse{Queue: current}, nil
			}
			current.DisplayName = common.String("queue-sample")
			current.VisibilityInSeconds = common.Int(30)
			current.CustomEncryptionKeyId = common.String("")
			return queuesdk.GetQueueResponse{Queue: current}, nil
		},
		updateFn: func(_ context.Context, req queuesdk.UpdateQueueRequest) (queuesdk.UpdateQueueResponse, error) {
			captured = req
			return queuesdk.UpdateQueueResponse{
				OpcWorkRequestId: common.String("wr-update-1"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-update-1", *req.WorkRequestId)
			return queuesdk.GetWorkRequestResponse{
				WorkRequest: makeWorkRequest("wr-update-1", queuesdk.OperationStatusSucceeded, queuesdk.ActionTypeUpdated, "ocid1.queue.oc1..existing"),
			}, nil
		},
	})

	resource := makeSpecQueue()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
	resource.Spec.DisplayName = "queue-sample"
	resource.Spec.CustomEncryptionKeyId = ""

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, "ocid1.queue.oc1..existing", *captured.QueueId)
	assert.Equal(t, "queue-sample", *captured.DisplayName)
	assert.Equal(t, 30, *captured.VisibilityInSeconds)
	assert.NotNil(t, captured.CustomEncryptionKeyId)
	assert.Equal(t, "", *captured.CustomEncryptionKeyId)
	assert.Equal(t, "", resource.Status.UpdateWorkRequestId)
}

func TestQueueRuntime_RejectsCreateOnlyDrift(t *testing.T) {
	updateCalls := 0
	manager := newQueueTestManager(&fakeQueueOCIClient{
		getFn: func(_ context.Context, _ queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
			current := makeSDKQueue("ocid1.queue.oc1..existing", "queue-sample", queuesdk.QueueLifecycleStateActive)
			current.CompartmentId = common.String("ocid1.compartment.oc1..other")
			current.RetentionInSeconds = common.Int(2400)
			return queuesdk.GetQueueResponse{Queue: current}, nil
		},
		updateFn: func(_ context.Context, _ queuesdk.UpdateQueueRequest) (queuesdk.UpdateQueueResponse, error) {
			updateCalls++
			return queuesdk.UpdateQueueResponse{}, nil
		},
	})

	resource := makeSpecQueue()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Contains(t, err.Error(), "create-only field drift")
	assert.Contains(t, err.Error(), "compartmentId")
	assert.Contains(t, err.Error(), "retentionInSeconds")
	assert.Equal(t, 0, updateCalls)
}

func TestQueueRuntime_DeletePendingWorkRequestKeepsTerminating(t *testing.T) {
	manager := newQueueTestManager(&fakeQueueOCIClient{
		getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-delete-1", *req.WorkRequestId)
			return queuesdk.GetWorkRequestResponse{
				WorkRequest: makeWorkRequest("wr-delete-1", queuesdk.OperationStatusInProgress, queuesdk.ActionTypeDeleted, "ocid1.queue.oc1..existing"),
			}, nil
		},
	})

	resource := makeSpecQueue()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
	resource.Status.DeleteWorkRequestId = "wr-delete-1"

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, deleted)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
}

func TestQueueRuntime_DeleteConfirmationTreatsNotFoundAsSuccess(t *testing.T) {
	manager := newQueueTestManager(&fakeQueueOCIClient{
		getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-delete-2", *req.WorkRequestId)
			return queuesdk.GetWorkRequestResponse{
				WorkRequest: makeWorkRequest("wr-delete-2", queuesdk.OperationStatusSucceeded, queuesdk.ActionTypeDeleted, "ocid1.queue.oc1..existing"),
			}, nil
		},
		getFn: func(_ context.Context, req queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
			assert.Equal(t, "ocid1.queue.oc1..existing", *req.QueueId)
			return queuesdk.GetQueueResponse{}, fakeQueueServiceError{
				statusCode: 404,
				code:       errorutil.NotFound,
				message:    "queue not found",
			}
		},
	})

	resource := makeSpecQueue()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
	resource.Status.DeleteWorkRequestId = "wr-delete-2"

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, deleted)
	assert.NotNil(t, resource.Status.OsokStatus.DeletedAt)
}

func TestQueueRuntime_DeleteConfirmationTreatsDeletedLifecycleAsSuccess(t *testing.T) {
	manager := newQueueTestManager(&fakeQueueOCIClient{
		getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-delete-3", *req.WorkRequestId)
			return queuesdk.GetWorkRequestResponse{
				WorkRequest: makeWorkRequest("wr-delete-3", queuesdk.OperationStatusSucceeded, queuesdk.ActionTypeDeleted, "ocid1.queue.oc1..existing"),
			}, nil
		},
		getFn: func(_ context.Context, req queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
			assert.Equal(t, "ocid1.queue.oc1..existing", *req.QueueId)
			return queuesdk.GetQueueResponse{
				Queue: makeSDKQueue("ocid1.queue.oc1..existing", "queue-sample", queuesdk.QueueLifecycleStateDeleted),
			}, nil
		},
	})

	resource := makeSpecQueue()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
	resource.Status.DeleteWorkRequestId = "wr-delete-3"

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, deleted)
}

func TestQueueRuntime_DeleteRequestNotFoundIsSuccess(t *testing.T) {
	manager := newQueueTestManager(&fakeQueueOCIClient{
		deleteFn: func(_ context.Context, req queuesdk.DeleteQueueRequest) (queuesdk.DeleteQueueResponse, error) {
			assert.Equal(t, "ocid1.queue.oc1..existing", *req.QueueId)
			return queuesdk.DeleteQueueResponse{}, fakeQueueServiceError{
				statusCode: 404,
				code:       errorutil.NotFound,
				message:    "queue not found",
			}
		},
	})

	resource := makeSpecQueue()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.True(t, deleted)
}
