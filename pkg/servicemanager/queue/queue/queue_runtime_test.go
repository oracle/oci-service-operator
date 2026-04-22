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
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
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
	statusCode   int
	code         string
	message      string
	opcRequestID string
}

func (f fakeQueueServiceError) Error() string          { return f.message }
func (f fakeQueueServiceError) GetHTTPStatusCode() int { return f.statusCode }
func (f fakeQueueServiceError) GetMessage() string     { return f.message }
func (f fakeQueueServiceError) GetCode() string        { return f.code }
func (f fakeQueueServiceError) GetOpcRequestID() string {
	return f.opcRequestID
}

func newTestQueueRuntimeHooks(manager *QueueServiceManager, client queueOCIClient) QueueRuntimeHooks {
	if client == nil {
		client = &fakeQueueOCIClient{}
	}

	hooks := QueueRuntimeHooks{
		Semantics: newQueueRuntimeSemantics(),
		Create: runtimeOperationHooks[queuesdk.CreateQueueRequest, queuesdk.CreateQueueResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "CreateQueueDetails", RequestName: "CreateQueueDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request queuesdk.CreateQueueRequest) (queuesdk.CreateQueueResponse, error) {
				return client.CreateQueue(ctx, request)
			},
		},
		Get: runtimeOperationHooks[queuesdk.GetQueueRequest, queuesdk.GetQueueResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "QueueId", RequestName: "queueId", Contribution: "path", PreferResourceID: true},
			},
			Call: func(ctx context.Context, request queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
				return client.GetQueue(ctx, request)
			},
		},
		List: runtimeOperationHooks[queuesdk.ListQueuesRequest, queuesdk.ListQueuesResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
				{FieldName: "LifecycleState", RequestName: "lifecycleState", Contribution: "query"},
				{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
				{FieldName: "Id", RequestName: "id", Contribution: "query"},
				{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
				{FieldName: "Page", RequestName: "page", Contribution: "query"},
				{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
				{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
			},
			Call: func(ctx context.Context, request queuesdk.ListQueuesRequest) (queuesdk.ListQueuesResponse, error) {
				return client.ListQueues(ctx, request)
			},
		},
		Update: runtimeOperationHooks[queuesdk.UpdateQueueRequest, queuesdk.UpdateQueueResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "QueueId", RequestName: "queueId", Contribution: "path", PreferResourceID: true},
				{FieldName: "UpdateQueueDetails", RequestName: "UpdateQueueDetails", Contribution: "body"},
			},
			Call: func(ctx context.Context, request queuesdk.UpdateQueueRequest) (queuesdk.UpdateQueueResponse, error) {
				return client.UpdateQueue(ctx, request)
			},
		},
		Delete: runtimeOperationHooks[queuesdk.DeleteQueueRequest, queuesdk.DeleteQueueResponse]{
			Fields: []generatedruntime.RequestField{
				{FieldName: "QueueId", RequestName: "queueId", Contribution: "path", PreferResourceID: true},
			},
			Call: func(ctx context.Context, request queuesdk.DeleteQueueRequest) (queuesdk.DeleteQueueResponse, error) {
				return client.DeleteQueue(ctx, request)
			},
		},
	}

	applyQueueRuntimeHooks(manager, &hooks, client, nil)
	hooks.WrapGeneratedClient = nil
	appendQueueGeneratedRuntimeOverlay(manager, &hooks, client, nil)
	return hooks
}

func newTestQueueGeneratedDelegate(manager *QueueServiceManager, client queueOCIClient) QueueServiceClient {
	hooks := newTestQueueRuntimeHooks(manager, client)
	delegate := defaultQueueServiceClient{
		ServiceClient: generatedruntime.NewServiceClient[*queuev1beta1.Queue](
			buildQueueGeneratedRuntimeConfig(manager, hooks),
		),
	}
	return wrapQueueGeneratedClient(hooks, delegate)
}

func newQueueTestManager(client queueOCIClient) *QueueServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewQueueServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), nil, nil, log, nil)
	return manager.WithClient(newTestQueueGeneratedDelegate(manager, client))
}

func TestQueueRuntimeHooksKeepQueueOverlayAndEndpointSecretWrapper(t *testing.T) {
	t.Parallel()

	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	manager := NewQueueServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), &fakeQueueCredentialClient{}, nil, log, nil)
	hooks := QueueRuntimeHooks{Semantics: newQueueRuntimeSemantics()}

	applyQueueRuntimeHooks(manager, &hooks, &fakeQueueOCIClient{}, nil)
	if len(hooks.WrapGeneratedClient) != 2 {
		t.Fatalf("WrapGeneratedClient count = %d, want 2", len(hooks.WrapGeneratedClient))
	}

	wrapped := wrapQueueGeneratedClient(hooks, fakeQueueServiceClient{})
	endpointClient, ok := wrapped.(queueEndpointSecretClient)
	if !ok {
		t.Fatalf("wrapped client type = %T, want queueEndpointSecretClient", wrapped)
	}
	overlayClient, ok := endpointClient.delegate.(queueGeneratedRuntimeOverlayClient)
	if !ok {
		t.Fatalf("wrapped delegate type = %T, want queueGeneratedRuntimeOverlayClient", endpointClient.delegate)
	}
	if _, ok := overlayClient.delegate.(fakeQueueServiceClient); !ok {
		t.Fatalf("overlay delegate type = %T, want fakeQueueServiceClient", overlayClient.delegate)
	}
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
	operationType := queuesdk.OperationTypeCreateQueue
	switch action {
	case queuesdk.ActionTypeUpdated:
		operationType = queuesdk.OperationTypeUpdateQueue
	case queuesdk.ActionTypeDeleted:
		operationType = queuesdk.OperationTypeDeleteQueue
	}

	return queuesdk.WorkRequest{
		Id:            common.String(id),
		Status:        status,
		OperationType: operationType,
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

func seedStaleQueueAsyncStatus(resource *queuev1beta1.Queue) {
	resource.Status.OsokStatus = shared.OSOKStatus{
		Reason:  string(shared.Updating),
		Message: "Queue update work request wr-stale is IN_PROGRESS",
		Async: shared.OSOKAsyncTracker{
			Current: &shared.OSOKAsyncOperation{
				Source:           shared.OSOKAsyncSourceWorkRequest,
				Phase:            shared.OSOKAsyncPhaseUpdate,
				WorkRequestID:    "wr-stale",
				RawStatus:        "IN_PROGRESS",
				RawOperationType: "UPDATE_QUEUE",
				NormalizedClass:  shared.OSOKAsyncClassPending,
			},
		},
	}
}

func TestQueueWorkRequestAsyncOperationMapsKnownStatusesAndActions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    queuesdk.OperationStatusEnum
		action    queuesdk.ActionTypeEnum
		phase     shared.OSOKAsyncPhase
		wantClass shared.OSOKAsyncNormalizedClass
	}{
		{
			name:      "accepted create pending",
			status:    queuesdk.OperationStatusAccepted,
			action:    queuesdk.ActionTypeCreated,
			phase:     shared.OSOKAsyncPhaseCreate,
			wantClass: shared.OSOKAsyncClassPending,
		},
		{
			name:      "in progress update pending",
			status:    queuesdk.OperationStatusInProgress,
			action:    queuesdk.ActionTypeUpdated,
			phase:     shared.OSOKAsyncPhaseUpdate,
			wantClass: shared.OSOKAsyncClassPending,
		},
		{
			name:      "canceling delete pending",
			status:    queuesdk.OperationStatusCanceling,
			action:    queuesdk.ActionTypeDeleted,
			phase:     shared.OSOKAsyncPhaseDelete,
			wantClass: shared.OSOKAsyncClassPending,
		},
		{
			name:      "failed create failed",
			status:    queuesdk.OperationStatusFailed,
			action:    queuesdk.ActionTypeCreated,
			phase:     shared.OSOKAsyncPhaseCreate,
			wantClass: shared.OSOKAsyncClassFailed,
		},
		{
			name:      "canceled update canceled",
			status:    queuesdk.OperationStatusCanceled,
			action:    queuesdk.ActionTypeUpdated,
			phase:     shared.OSOKAsyncPhaseUpdate,
			wantClass: shared.OSOKAsyncClassCanceled,
		},
		{
			name:      "succeeded delete succeeded",
			status:    queuesdk.OperationStatusSucceeded,
			action:    queuesdk.ActionTypeDeleted,
			phase:     shared.OSOKAsyncPhaseDelete,
			wantClass: shared.OSOKAsyncClassSucceeded,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			workRequest := makeWorkRequest("wr-adapter", tt.status, tt.action, "ocid1.queue.oc1..existing")
			current, err := queueWorkRequestAsyncOperation(makeSpecQueue(), workRequest, tt.phase)
			assert.NoError(t, err)
			if assert.NotNil(t, current) {
				assert.Equal(t, shared.OSOKAsyncSourceWorkRequest, current.Source)
				assert.Equal(t, tt.phase, current.Phase)
				assert.Equal(t, "wr-adapter", current.WorkRequestID)
				assert.Equal(t, string(tt.status), current.RawStatus)
				assert.Equal(t, string(workRequest.OperationType), current.RawOperationType)
				assert.Equal(t, tt.wantClass, current.NormalizedClass)
				assert.Equal(t, "Queue "+string(tt.phase)+" work request wr-adapter is "+string(tt.status), current.Message)
				if assert.NotNil(t, current.PercentComplete) {
					assert.Equal(t, float32(100), *current.PercentComplete)
				}
			}
		})
	}
}

func TestQueueWorkRequestAsyncOperationRejectsUnmodeledStatus(t *testing.T) {
	t.Parallel()

	_, err := queueWorkRequestAsyncOperation(makeSpecQueue(), makeWorkRequest("wr-unknown", queuesdk.OperationStatusEnum("WAITING"), queuesdk.ActionTypeCreated, "ocid1.queue.oc1..existing"), shared.OSOKAsyncPhaseCreate)
	assert.EqualError(t, err, `unmodeled async status "WAITING"`)
}

func TestQueueRuntime_CreateAcceptedPersistsWorkRequestAndRequeues(t *testing.T) {
	var captured queuesdk.CreateQueueRequest
	manager := newQueueTestManager(&fakeQueueOCIClient{
		createFn: func(_ context.Context, req queuesdk.CreateQueueRequest) (queuesdk.CreateQueueResponse, error) {
			captured = req
			return queuesdk.CreateQueueResponse{
				OpcRequestId:     common.String("opc-create-1"),
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
	assert.Equal(t, "opc-create-1", resource.Status.OsokStatus.OpcRequestID)
	assert.Equal(t, "wr-create-1", resource.Status.CreateWorkRequestId)
	if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
		assert.Equal(t, shared.OSOKAsyncSourceWorkRequest, resource.Status.OsokStatus.Async.Current.Source)
		assert.Equal(t, shared.OSOKAsyncPhaseCreate, resource.Status.OsokStatus.Async.Current.Phase)
		assert.Equal(t, "wr-create-1", resource.Status.OsokStatus.Async.Current.WorkRequestID)
		assert.Equal(t, "ACCEPTED", resource.Status.OsokStatus.Async.Current.RawStatus)
		assert.Equal(t, shared.OSOKAsyncClassPending, resource.Status.OsokStatus.Async.Current.NormalizedClass)
		if assert.NotNil(t, resource.Status.OsokStatus.Async.Current.PercentComplete) {
			assert.Equal(t, float32(100), *resource.Status.OsokStatus.Async.Current.PercentComplete)
		}
	}
	assert.Equal(t, "queue-sample", *captured.DisplayName)
	assert.Equal(t, "ocid1.compartment.oc1..example", *captured.CompartmentId)
	assert.Equal(t, 1200, *captured.RetentionInSeconds)
}

func TestQueueRuntime_CreateErrorCapturesOpcRequestID(t *testing.T) {
	t.Parallel()

	manager := newQueueTestManager(&fakeQueueOCIClient{
		createFn: func(_ context.Context, _ queuesdk.CreateQueueRequest) (queuesdk.CreateQueueResponse, error) {
			return queuesdk.CreateQueueResponse{}, fakeQueueServiceError{
				statusCode:   409,
				code:         errorutil.IncorrectState,
				message:      "create conflict",
				opcRequestID: "opc-create-conflict",
			}
		},
	})

	resource := makeSpecQueue()
	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Equal(t, "opc-create-conflict", resource.Status.OsokStatus.OpcRequestID)
	assert.Equal(t, err.Error(), resource.Status.OsokStatus.Message)
	assert.Equal(t, string(shared.Failed), resource.Status.OsokStatus.Reason)
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
	assert.Nil(t, resource.Status.OsokStatus.Async.Current)
}

func TestQueueRuntime_ResumeCreateSucceededFailedLifecycleProjectsCanonicalAsyncPhase(t *testing.T) {
	manager := newQueueTestManager(&fakeQueueOCIClient{
		getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-create-failed-lifecycle", *req.WorkRequestId)
			return queuesdk.GetWorkRequestResponse{
				WorkRequest: makeWorkRequest("wr-create-failed-lifecycle", queuesdk.OperationStatusSucceeded, queuesdk.ActionTypeCreated, "ocid1.queue.oc1..created"),
			}, nil
		},
		getFn: func(_ context.Context, req queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
			assert.Equal(t, "ocid1.queue.oc1..created", *req.QueueId)
			return queuesdk.GetQueueResponse{
				Queue: makeSDKQueue("ocid1.queue.oc1..created", "queue-sample", queuesdk.QueueLifecycleStateFailed),
			}, nil
		},
	})

	resource := makeSpecQueue()
	resource.Status.CreateWorkRequestId = "wr-create-failed-lifecycle"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, "", resource.Status.CreateWorkRequestId)
	assert.Equal(t, string(shared.Failed), resource.Status.OsokStatus.Reason)
	assert.Equal(t, "Queue queue-sample is FAILED", resource.Status.OsokStatus.Message)
	if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
		assert.Equal(t, shared.OSOKAsyncSourceLifecycle, resource.Status.OsokStatus.Async.Current.Source)
		assert.Equal(t, shared.OSOKAsyncPhaseCreate, resource.Status.OsokStatus.Async.Current.Phase)
		assert.Equal(t, "", resource.Status.OsokStatus.Async.Current.WorkRequestID)
		assert.Equal(t, "FAILED", resource.Status.OsokStatus.Async.Current.RawStatus)
		assert.Equal(t, "", resource.Status.OsokStatus.Async.Current.RawOperationType)
		assert.Equal(t, shared.OSOKAsyncClassFailed, resource.Status.OsokStatus.Async.Current.NormalizedClass)
		assert.Equal(t, "Queue queue-sample is FAILED", resource.Status.OsokStatus.Async.Current.Message)
	}
}

func TestQueueRuntime_ResumeUpdateSucceededFailedLifecycleProjectsCanonicalAsyncPhase(t *testing.T) {
	getCalls := 0
	manager := newQueueTestManager(&fakeQueueOCIClient{
		getFn: func(_ context.Context, req queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.queue.oc1..existing", *req.QueueId)
			return queuesdk.GetQueueResponse{
				Queue: makeSDKQueue("ocid1.queue.oc1..existing", "queue-sample", queuesdk.QueueLifecycleStateFailed),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-update-failed-lifecycle", *req.WorkRequestId)
			return queuesdk.GetWorkRequestResponse{
				WorkRequest: makeWorkRequest("wr-update-failed-lifecycle", queuesdk.OperationStatusSucceeded, queuesdk.ActionTypeUpdated, "ocid1.queue.oc1..existing"),
			}, nil
		},
	})

	resource := makeSpecQueue()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
	resource.Status.UpdateWorkRequestId = "wr-update-failed-lifecycle"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, "", resource.Status.UpdateWorkRequestId)
	assert.Equal(t, string(shared.Failed), resource.Status.OsokStatus.Reason)
	assert.Equal(t, "Queue queue-sample is FAILED", resource.Status.OsokStatus.Message)
	if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
		assert.Equal(t, shared.OSOKAsyncSourceLifecycle, resource.Status.OsokStatus.Async.Current.Source)
		assert.Equal(t, shared.OSOKAsyncPhaseUpdate, resource.Status.OsokStatus.Async.Current.Phase)
		assert.Equal(t, "", resource.Status.OsokStatus.Async.Current.WorkRequestID)
		assert.Equal(t, "FAILED", resource.Status.OsokStatus.Async.Current.RawStatus)
		assert.Equal(t, "", resource.Status.OsokStatus.Async.Current.RawOperationType)
		assert.Equal(t, shared.OSOKAsyncClassFailed, resource.Status.OsokStatus.Async.Current.NormalizedClass)
		assert.Equal(t, "Queue queue-sample is FAILED", resource.Status.OsokStatus.Async.Current.Message)
	}
}

func TestQueueRuntime_ResumeUpdateSucceededUnreadablePreservesWorkRequestDetails(t *testing.T) {
	manager := newQueueTestManager(&fakeQueueOCIClient{
		getFn: func(_ context.Context, req queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
			assert.Equal(t, "ocid1.queue.oc1..existing", *req.QueueId)
			return queuesdk.GetQueueResponse{}, fakeQueueServiceError{
				statusCode: 404,
				code:       errorutil.NotFound,
				message:    "queue not found",
			}
		},
		getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-update-missing", *req.WorkRequestId)
			return queuesdk.GetWorkRequestResponse{
				WorkRequest: makeWorkRequest("wr-update-missing", queuesdk.OperationStatusSucceeded, queuesdk.ActionTypeUpdated, "ocid1.queue.oc1..existing"),
			}, nil
		},
	})

	resource := makeSpecQueue()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
	resource.Status.UpdateWorkRequestId = "wr-update-missing"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.EqualError(t, err, "Queue update work request wr-update-missing succeeded but Queue ocid1.queue.oc1..existing is no longer readable")
	assert.False(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, string(shared.Failed), resource.Status.OsokStatus.Reason)
	assert.Equal(t, "Queue update work request wr-update-missing succeeded but Queue ocid1.queue.oc1..existing is no longer readable", resource.Status.OsokStatus.Message)
	if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
		assert.Equal(t, shared.OSOKAsyncSourceWorkRequest, resource.Status.OsokStatus.Async.Current.Source)
		assert.Equal(t, shared.OSOKAsyncPhaseUpdate, resource.Status.OsokStatus.Async.Current.Phase)
		assert.Equal(t, "wr-update-missing", resource.Status.OsokStatus.Async.Current.WorkRequestID)
		assert.Equal(t, "SUCCEEDED", resource.Status.OsokStatus.Async.Current.RawStatus)
		assert.Equal(t, "UPDATE_QUEUE", resource.Status.OsokStatus.Async.Current.RawOperationType)
		assert.Equal(t, shared.OSOKAsyncClassFailed, resource.Status.OsokStatus.Async.Current.NormalizedClass)
		assert.Equal(t, "Queue update work request wr-update-missing succeeded but Queue ocid1.queue.oc1..existing is no longer readable", resource.Status.OsokStatus.Async.Current.Message)
	}
}

func TestQueueRuntime_TerminalWorkRequestPreservesWorkRequestDetails(t *testing.T) {
	tests := []struct {
		name              string
		workRequestID     string
		workRequest       queuesdk.WorkRequest
		wantPhase         shared.OSOKAsyncPhase
		wantClass         shared.OSOKAsyncNormalizedClass
		wantOperation     string
		runCreateOrUpdate bool
		setupResource     func(*queuev1beta1.Queue)
		setupClient       func(string) queueOCIClient
	}{
		{
			name:              "create failed",
			workRequestID:     "wr-create-terminal",
			workRequest:       makeWorkRequest("wr-create-terminal", queuesdk.OperationStatusFailed, queuesdk.ActionTypeCreated, "ocid1.queue.oc1..created"),
			wantPhase:         shared.OSOKAsyncPhaseCreate,
			wantClass:         shared.OSOKAsyncClassFailed,
			wantOperation:     "CREATE_QUEUE",
			runCreateOrUpdate: true,
			setupResource: func(resource *queuev1beta1.Queue) {
				resource.Status.CreateWorkRequestId = "wr-create-terminal"
			},
			setupClient: func(workRequestID string) queueOCIClient {
				return &fakeQueueOCIClient{
					getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
						assert.Equal(t, workRequestID, *req.WorkRequestId)
						return queuesdk.GetWorkRequestResponse{
							WorkRequest: makeWorkRequest(workRequestID, queuesdk.OperationStatusFailed, queuesdk.ActionTypeCreated, "ocid1.queue.oc1..created"),
						}, nil
					},
				}
			},
		},
		{
			name:              "update canceled",
			workRequestID:     "wr-update-terminal",
			workRequest:       makeWorkRequest("wr-update-terminal", queuesdk.OperationStatusCanceled, queuesdk.ActionTypeUpdated, "ocid1.queue.oc1..existing"),
			wantPhase:         shared.OSOKAsyncPhaseUpdate,
			wantClass:         shared.OSOKAsyncClassCanceled,
			wantOperation:     "UPDATE_QUEUE",
			runCreateOrUpdate: true,
			setupResource: func(resource *queuev1beta1.Queue) {
				resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
				resource.Status.UpdateWorkRequestId = "wr-update-terminal"
			},
			setupClient: func(workRequestID string) queueOCIClient {
				return &fakeQueueOCIClient{
					getFn: func(_ context.Context, req queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
						assert.Equal(t, "ocid1.queue.oc1..existing", *req.QueueId)
						return queuesdk.GetQueueResponse{
							Queue: makeSDKQueue("ocid1.queue.oc1..existing", "queue-sample", queuesdk.QueueLifecycleStateActive),
						}, nil
					},
					getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
						assert.Equal(t, workRequestID, *req.WorkRequestId)
						return queuesdk.GetWorkRequestResponse{
							WorkRequest: makeWorkRequest(workRequestID, queuesdk.OperationStatusCanceled, queuesdk.ActionTypeUpdated, "ocid1.queue.oc1..existing"),
						}, nil
					},
				}
			},
		},
		{
			name:          "delete failed",
			workRequestID: "wr-delete-terminal",
			workRequest:   makeWorkRequest("wr-delete-terminal", queuesdk.OperationStatusFailed, queuesdk.ActionTypeDeleted, "ocid1.queue.oc1..existing"),
			wantPhase:     shared.OSOKAsyncPhaseDelete,
			wantClass:     shared.OSOKAsyncClassFailed,
			wantOperation: "DELETE_QUEUE",
			setupResource: func(resource *queuev1beta1.Queue) {
				resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
				resource.Status.DeleteWorkRequestId = "wr-delete-terminal"
			},
			setupClient: func(workRequestID string) queueOCIClient {
				return &fakeQueueOCIClient{
					getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
						assert.Equal(t, workRequestID, *req.WorkRequestId)
						return queuesdk.GetWorkRequestResponse{
							WorkRequest: makeWorkRequest(workRequestID, queuesdk.OperationStatusFailed, queuesdk.ActionTypeDeleted, "ocid1.queue.oc1..existing"),
						}, nil
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := makeSpecQueue()
			tt.setupResource(resource)

			manager := newQueueTestManager(tt.setupClient(tt.workRequestID))

			if tt.runCreateOrUpdate {
				resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
				assert.EqualError(t, err, "Queue "+string(tt.wantPhase)+" work request "+tt.workRequestID+" finished with status "+string(tt.workRequest.Status))
				assert.False(t, resp.IsSuccessful)
				assert.False(t, resp.ShouldRequeue)
			} else {
				deleted, err := manager.Delete(context.Background(), resource)
				assert.EqualError(t, err, "Queue "+string(tt.wantPhase)+" work request "+tt.workRequestID+" finished with status "+string(tt.workRequest.Status))
				assert.False(t, deleted)
			}

			assert.Equal(t, string(shared.Failed), resource.Status.OsokStatus.Reason)
			assert.Equal(t, "Queue "+string(tt.wantPhase)+" work request "+tt.workRequestID+" finished with status "+string(tt.workRequest.Status), resource.Status.OsokStatus.Message)
			if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
				assert.Equal(t, shared.OSOKAsyncSourceWorkRequest, resource.Status.OsokStatus.Async.Current.Source)
				assert.Equal(t, tt.wantPhase, resource.Status.OsokStatus.Async.Current.Phase)
				assert.Equal(t, tt.workRequestID, resource.Status.OsokStatus.Async.Current.WorkRequestID)
				assert.Equal(t, string(tt.workRequest.Status), resource.Status.OsokStatus.Async.Current.RawStatus)
				assert.Equal(t, tt.wantOperation, resource.Status.OsokStatus.Async.Current.RawOperationType)
				assert.Equal(t, tt.wantClass, resource.Status.OsokStatus.Async.Current.NormalizedClass)
				assert.Equal(t, "Queue "+string(tt.wantPhase)+" work request "+tt.workRequestID+" finished with status "+string(tt.workRequest.Status), resource.Status.OsokStatus.Async.Current.Message)
			}
		})
	}
}

func TestQueueRuntime_ProjectStatusPreservesWorkRequestIDs(t *testing.T) {
	resource := makeSpecQueue()
	resource.Status.OsokStatus = shared.OSOKStatus{
		Reason:  string(shared.Updating),
		Message: "queue update work request is IN_PROGRESS",
		Async: shared.OSOKAsyncTracker{
			Current: &shared.OSOKAsyncOperation{
				Source:           shared.OSOKAsyncSourceWorkRequest,
				Phase:            shared.OSOKAsyncPhaseUpdate,
				WorkRequestID:    "wr-update-status",
				RawStatus:        "IN_PROGRESS",
				RawOperationType: "UPDATE_QUEUE",
				NormalizedClass:  shared.OSOKAsyncClassPending,
			},
		},
	}
	resource.Status.CreateWorkRequestId = "wr-create-status"
	resource.Status.UpdateWorkRequestId = "wr-update-status"
	resource.Status.DeleteWorkRequestId = "wr-delete-status"

	err := projectQueueStatus(resource, makeSDKQueue("ocid1.queue.oc1..existing", "queue-sample", queuesdk.QueueLifecycleStateUpdating))

	assert.NoError(t, err)
	assert.Equal(t, "wr-create-status", resource.Status.CreateWorkRequestId)
	assert.Equal(t, "wr-update-status", resource.Status.UpdateWorkRequestId)
	assert.Equal(t, "wr-delete-status", resource.Status.DeleteWorkRequestId)
	assert.Equal(t, "ocid1.queue.oc1..existing", resource.Status.Id)
	assert.Equal(t, "queue-sample", resource.Status.DisplayName)
	assert.Equal(t, "UPDATING", resource.Status.LifecycleState)
	assert.Equal(t, "https://cell-1.queue.messaging.us-phoenix-1.oci.oraclecloud.com", resource.Status.MessagesEndpoint)
	assert.Equal(t, string(shared.Updating), resource.Status.OsokStatus.Reason)
	assert.Equal(t, "queue update work request is IN_PROGRESS", resource.Status.OsokStatus.Message)
	if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
		assert.Equal(t, "wr-update-status", resource.Status.OsokStatus.Async.Current.WorkRequestID)
		assert.Equal(t, shared.OSOKAsyncPhaseUpdate, resource.Status.OsokStatus.Async.Current.Phase)
	}
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

func TestQueueRuntime_UpdateAcceptedPersistsLegacyMirrorAndSharedAsyncTracker(t *testing.T) {
	var captured queuesdk.UpdateQueueRequest
	manager := newQueueTestManager(&fakeQueueOCIClient{
		getFn: func(_ context.Context, req queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
			assert.Equal(t, "ocid1.queue.oc1..existing", *req.QueueId)
			return queuesdk.GetQueueResponse{
				Queue: makeSDKQueue("ocid1.queue.oc1..existing", "old-name", queuesdk.QueueLifecycleStateActive),
			}, nil
		},
		updateFn: func(_ context.Context, req queuesdk.UpdateQueueRequest) (queuesdk.UpdateQueueResponse, error) {
			captured = req
			return queuesdk.UpdateQueueResponse{
				OpcWorkRequestId: common.String("wr-update-pending"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-update-pending", *req.WorkRequestId)
			return queuesdk.GetWorkRequestResponse{
				WorkRequest: makeWorkRequest("wr-update-pending", queuesdk.OperationStatusInProgress, queuesdk.ActionTypeUpdated, "ocid1.queue.oc1..existing"),
			}, nil
		},
	})

	resource := makeSpecQueue()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
	resource.Spec.DisplayName = "queue-sample"

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, resp.ShouldRequeue)
	assert.Equal(t, "ocid1.queue.oc1..existing", *captured.QueueId)
	assert.Equal(t, "wr-update-pending", resource.Status.UpdateWorkRequestId)
	if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
		assert.Equal(t, shared.OSOKAsyncSourceWorkRequest, resource.Status.OsokStatus.Async.Current.Source)
		assert.Equal(t, shared.OSOKAsyncPhaseUpdate, resource.Status.OsokStatus.Async.Current.Phase)
		assert.Equal(t, "wr-update-pending", resource.Status.OsokStatus.Async.Current.WorkRequestID)
		assert.Equal(t, "IN_PROGRESS", resource.Status.OsokStatus.Async.Current.RawStatus)
		assert.Equal(t, "UPDATE_QUEUE", resource.Status.OsokStatus.Async.Current.RawOperationType)
		assert.Equal(t, shared.OSOKAsyncClassPending, resource.Status.OsokStatus.Async.Current.NormalizedClass)
	}
}

func TestQueueRuntime_MutableUpdateDriftPreservesExplicitZeroValues(t *testing.T) {
	var captured queuesdk.UpdateQueueRequest
	getCalls := 0
	manager := newQueueTestManager(&fakeQueueOCIClient{
		getFn: func(_ context.Context, req queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
			getCalls++
			assert.Equal(t, "ocid1.queue.oc1..existing", *req.QueueId)
			current := makeSDKQueue("ocid1.queue.oc1..existing", "queue-sample", queuesdk.QueueLifecycleStateActive)
			if getCalls == 1 {
				return queuesdk.GetQueueResponse{Queue: current}, nil
			}
			current.VisibilityInSeconds = common.Int(0)
			current.TimeoutInSeconds = common.Int(0)
			current.ChannelConsumptionLimit = common.Int(0)
			current.DeadLetterQueueDeliveryCount = common.Int(0)
			return queuesdk.GetQueueResponse{Queue: current}, nil
		},
		updateFn: func(_ context.Context, req queuesdk.UpdateQueueRequest) (queuesdk.UpdateQueueResponse, error) {
			captured = req
			return queuesdk.UpdateQueueResponse{
				OpcWorkRequestId: common.String("wr-update-zero"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-update-zero", *req.WorkRequestId)
			return queuesdk.GetWorkRequestResponse{
				WorkRequest: makeWorkRequest("wr-update-zero", queuesdk.OperationStatusSucceeded, queuesdk.ActionTypeUpdated, "ocid1.queue.oc1..existing"),
			}, nil
		},
	})

	resource := makeSpecQueue()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
	resource.Spec.VisibilityInSeconds = 0
	resource.Spec.TimeoutInSeconds = 0
	resource.Spec.ChannelConsumptionLimit = 0
	resource.Spec.DeadLetterQueueDeliveryCount = 0

	resp, err := manager.CreateOrUpdate(context.Background(), resource, ctrl.Request{})

	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, resp.ShouldRequeue)
	assert.Equal(t, "ocid1.queue.oc1..existing", *captured.QueueId)
	if assert.NotNil(t, captured.VisibilityInSeconds) {
		assert.Equal(t, 0, *captured.VisibilityInSeconds)
	}
	if assert.NotNil(t, captured.TimeoutInSeconds) {
		assert.Equal(t, 0, *captured.TimeoutInSeconds)
	}
	if assert.NotNil(t, captured.ChannelConsumptionLimit) {
		assert.Equal(t, 0, *captured.ChannelConsumptionLimit)
	}
	if assert.NotNil(t, captured.DeadLetterQueueDeliveryCount) {
		assert.Equal(t, 0, *captured.DeadLetterQueueDeliveryCount)
	}
	assert.Equal(t, "", resource.Status.UpdateWorkRequestId)
	assert.Equal(t, 0, resource.Status.VisibilityInSeconds)
	assert.Equal(t, 0, resource.Status.TimeoutInSeconds)
	assert.Equal(t, 0, resource.Status.ChannelConsumptionLimit)
	assert.Equal(t, 0, resource.Status.DeadLetterQueueDeliveryCount)
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
	assert.Contains(t, err.Error(), "formal semantics require replacement")
	assert.Contains(t, err.Error(), "compartmentId")
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
	if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
		assert.Equal(t, shared.OSOKAsyncSourceWorkRequest, resource.Status.OsokStatus.Async.Current.Source)
		assert.Equal(t, shared.OSOKAsyncPhaseDelete, resource.Status.OsokStatus.Async.Current.Phase)
		assert.Equal(t, "wr-delete-1", resource.Status.OsokStatus.Async.Current.WorkRequestID)
		assert.Equal(t, "IN_PROGRESS", resource.Status.OsokStatus.Async.Current.RawStatus)
	}
}

func TestQueueRuntime_DeleteAcceptedPersistsLegacyMirrorAndSharedAsyncTracker(t *testing.T) {
	manager := newQueueTestManager(&fakeQueueOCIClient{
		deleteFn: func(_ context.Context, req queuesdk.DeleteQueueRequest) (queuesdk.DeleteQueueResponse, error) {
			assert.Equal(t, "ocid1.queue.oc1..existing", *req.QueueId)
			return queuesdk.DeleteQueueResponse{
				OpcWorkRequestId: common.String("wr-delete-pending"),
			}, nil
		},
		getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-delete-pending", *req.WorkRequestId)
			return queuesdk.GetWorkRequestResponse{
				WorkRequest: makeWorkRequest("wr-delete-pending", queuesdk.OperationStatusAccepted, queuesdk.ActionTypeDeleted, "ocid1.queue.oc1..existing"),
			}, nil
		},
	})

	resource := makeSpecQueue()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, deleted)
	assert.Equal(t, "wr-delete-pending", resource.Status.DeleteWorkRequestId)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
		assert.Equal(t, shared.OSOKAsyncSourceWorkRequest, resource.Status.OsokStatus.Async.Current.Source)
		assert.Equal(t, shared.OSOKAsyncPhaseDelete, resource.Status.OsokStatus.Async.Current.Phase)
		assert.Equal(t, "wr-delete-pending", resource.Status.OsokStatus.Async.Current.WorkRequestID)
		assert.Equal(t, "ACCEPTED", resource.Status.OsokStatus.Async.Current.RawStatus)
		assert.Equal(t, "DELETE_QUEUE", resource.Status.OsokStatus.Async.Current.RawOperationType)
		assert.Equal(t, shared.OSOKAsyncClassPending, resource.Status.OsokStatus.Async.Current.NormalizedClass)
	}
}

func TestQueueRuntime_DeleteWaitsForQueueDisappearanceWhenWorkRequestReadIsGone(t *testing.T) {
	manager := newQueueTestManager(&fakeQueueOCIClient{
		getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
			assert.Equal(t, "wr-delete-missing", *req.WorkRequestId)
			return queuesdk.GetWorkRequestResponse{}, fakeQueueServiceError{
				statusCode: 404,
				code:       errorutil.NotFound,
				message:    "work request not found",
			}
		},
		getFn: func(_ context.Context, req queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
			assert.Equal(t, "ocid1.queue.oc1..existing", *req.QueueId)
			return queuesdk.GetQueueResponse{
				Queue: makeSDKQueue("ocid1.queue.oc1..existing", "queue-sample", queuesdk.QueueLifecycleStateDeleting),
			}, nil
		},
	})

	resource := makeSpecQueue()
	resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
	resource.Status.DeleteWorkRequestId = "wr-delete-missing"

	deleted, err := manager.Delete(context.Background(), resource)

	assert.NoError(t, err)
	assert.False(t, deleted)
	assert.Equal(t, "wr-delete-missing", resource.Status.DeleteWorkRequestId)
	assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
	assert.Contains(t, resource.Status.OsokStatus.Message, "waiting for Queue ocid1.queue.oc1..existing to disappear")
	if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
		assert.Equal(t, shared.OSOKAsyncPhaseDelete, resource.Status.OsokStatus.Async.Current.Phase)
		assert.Equal(t, "wr-delete-missing", resource.Status.OsokStatus.Async.Current.WorkRequestID)
		assert.Equal(t, "", resource.Status.OsokStatus.Async.Current.RawStatus)
		assert.Equal(t, "", resource.Status.OsokStatus.Async.Current.RawOperationType)
	}
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
	assert.Nil(t, resource.Status.OsokStatus.Async.Current)
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

func TestQueueRuntime_DeleteConfirmationUnexpectedLifecycleKeepsWaiting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state queuesdk.QueueLifecycleStateEnum
	}{
		{
			name:  "active",
			state: queuesdk.QueueLifecycleStateActive,
		},
		{
			name:  "failed",
			state: queuesdk.QueueLifecycleStateFailed,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			getCalls := 0
			manager := newQueueTestManager(&fakeQueueOCIClient{
				getWorkRequestFn: func(_ context.Context, req queuesdk.GetWorkRequestRequest) (queuesdk.GetWorkRequestResponse, error) {
					assert.Equal(t, "wr-delete-visible", *req.WorkRequestId)
					return queuesdk.GetWorkRequestResponse{
						WorkRequest: makeWorkRequest("wr-delete-visible", queuesdk.OperationStatusSucceeded, queuesdk.ActionTypeDeleted, "ocid1.queue.oc1..existing"),
					}, nil
				},
				getFn: func(_ context.Context, req queuesdk.GetQueueRequest) (queuesdk.GetQueueResponse, error) {
					getCalls++
					assert.Equal(t, "ocid1.queue.oc1..existing", *req.QueueId)
					return queuesdk.GetQueueResponse{
						Queue: makeSDKQueue("ocid1.queue.oc1..existing", "queue-sample", tt.state),
					}, nil
				},
			})

			resource := makeSpecQueue()
			resource.Status.OsokStatus.Ocid = shared.OCID("ocid1.queue.oc1..existing")
			resource.Status.DeleteWorkRequestId = "wr-delete-visible"

			deleted, err := manager.Delete(context.Background(), resource)

			assert.NoError(t, err)
			assert.False(t, deleted)
			assert.Equal(t, 2, getCalls)
			assert.Equal(t, "wr-delete-visible", resource.Status.DeleteWorkRequestId)
			assert.Equal(t, string(tt.state), resource.Status.LifecycleState)
			assert.Equal(t, string(shared.Terminating), resource.Status.OsokStatus.Reason)
			assert.Equal(t, "Queue delete work request wr-delete-visible succeeded; waiting for Queue ocid1.queue.oc1..existing to disappear", resource.Status.OsokStatus.Message)
			if assert.NotNil(t, resource.Status.OsokStatus.Async.Current) {
				assert.Equal(t, shared.OSOKAsyncSourceWorkRequest, resource.Status.OsokStatus.Async.Current.Source)
				assert.Equal(t, shared.OSOKAsyncPhaseDelete, resource.Status.OsokStatus.Async.Current.Phase)
				assert.Equal(t, "wr-delete-visible", resource.Status.OsokStatus.Async.Current.WorkRequestID)
				assert.Equal(t, shared.OSOKAsyncClassPending, resource.Status.OsokStatus.Async.Current.NormalizedClass)
				assert.Equal(t, "Queue delete work request wr-delete-visible succeeded; waiting for Queue ocid1.queue.oc1..existing to disappear", resource.Status.OsokStatus.Async.Current.Message)
				assert.Equal(t, "", resource.Status.OsokStatus.Async.Current.RawStatus)
				assert.Equal(t, "", resource.Status.OsokStatus.Async.Current.RawOperationType)
			}
		})
	}
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
