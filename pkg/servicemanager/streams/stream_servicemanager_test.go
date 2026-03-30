/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package streams_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/streaming"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/streams"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type mockStreamAdminClient struct {
	createStreamFn func(context.Context, streaming.CreateStreamRequest) (streaming.CreateStreamResponse, error)
	listStreamsFn  func(context.Context, streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error)
	deleteStreamFn func(context.Context, streaming.DeleteStreamRequest) (streaming.DeleteStreamResponse, error)
	getStreamFn    func(context.Context, streaming.GetStreamRequest) (streaming.GetStreamResponse, error)
	updateStreamFn func(context.Context, streaming.UpdateStreamRequest) (streaming.UpdateStreamResponse, error)
}

func (m *mockStreamAdminClient) CreateStream(ctx context.Context, req streaming.CreateStreamRequest) (streaming.CreateStreamResponse, error) {
	if m.createStreamFn != nil {
		return m.createStreamFn(ctx, req)
	}
	return streaming.CreateStreamResponse{}, nil
}

func (m *mockStreamAdminClient) GetStream(ctx context.Context, req streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
	if m.getStreamFn != nil {
		return m.getStreamFn(ctx, req)
	}
	return streaming.GetStreamResponse{}, nil
}

func (m *mockStreamAdminClient) ListStreams(ctx context.Context, req streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
	if m.listStreamsFn != nil {
		return m.listStreamsFn(ctx, req)
	}
	return streaming.ListStreamsResponse{}, nil
}

func (m *mockStreamAdminClient) UpdateStream(ctx context.Context, req streaming.UpdateStreamRequest) (streaming.UpdateStreamResponse, error) {
	if m.updateStreamFn != nil {
		return m.updateStreamFn(ctx, req)
	}
	return streaming.UpdateStreamResponse{}, nil
}

func (m *mockStreamAdminClient) DeleteStream(ctx context.Context, req streaming.DeleteStreamRequest) (streaming.DeleteStreamResponse, error) {
	if m.deleteStreamFn != nil {
		return m.deleteStreamFn(ctx, req)
	}
	return streaming.DeleteStreamResponse{}, nil
}

type fakeCredentialClient struct {
	createSecretFn func(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error)
	deleteSecretFn func(ctx context.Context, name, ns string) (bool, error)
	getSecretFn    func(ctx context.Context, name, ns string) (map[string][]byte, error)
	updateSecretFn func(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error)
	createCalled   bool
	deleteCalled   bool
}

func (f *fakeCredentialClient) CreateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	f.createCalled = true
	if f.createSecretFn != nil {
		return f.createSecretFn(ctx, name, ns, labels, data)
	}
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(ctx context.Context, name, ns string) (bool, error) {
	f.deleteCalled = true
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, ns)
	}
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(ctx context.Context, name, ns string) (map[string][]byte, error) {
	if f.getSecretFn != nil {
		return f.getSecretFn(ctx, name, ns)
	}
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	if f.updateSecretFn != nil {
		return f.updateSecretFn(ctx, name, ns, labels, data)
	}
	return true, nil
}

func makeTestManager(credClient *fakeCredentialClient, mockClient *mockStreamAdminClient) *StreamServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	m := &metrics.Metrics{Logger: log}
	mgr := NewStreamServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), credClient, nil, log, m)
	if mockClient != nil {
		ExportSetClientForTest(mgr, mockClient)
	}
	return mgr
}

func newTestManager(mockClient *mockStreamAdminClient) *StreamServiceManager {
	return makeTestManager(&fakeCredentialClient{}, mockClient)
}

func makeActiveStream(id, name string) streaming.Stream {
	return streaming.Stream{
		Id:               common.String(id),
		Name:             common.String(name),
		LifecycleState:   "ACTIVE",
		MessagesEndpoint: common.String("https://cell-1.streaming.us-phoenix-1.oci.oraclecloud.com"),
		StreamPoolId:     common.String("ocid1.streampool.oc1..xxx"),
		Partitions:       common.Int(1),
		RetentionInHours: common.Int(24),
	}
}

func TestCreateStream_UsesInjectedClientAndRetention(t *testing.T) {
	var captured streaming.CreateStreamRequest
	mgr := newTestManager(&mockStreamAdminClient{
		createStreamFn: func(_ context.Context, req streaming.CreateStreamRequest) (streaming.CreateStreamResponse, error) {
			captured = req
			return streaming.CreateStreamResponse{}, nil
		},
	})

	stream := streamingv1beta1.Stream{
		Spec: streamingv1beta1.StreamSpec{
			Name:             "test-stream",
			Partitions:       1,
			RetentionInHours: 24,
			CompartmentId:    "ocid1.compartment.oc1..example",
			StreamPoolId:     "ocid1.streampool.oc1..example",
		},
	}

	_, err := mgr.CreateStream(context.Background(), stream)
	assert.NoError(t, err)
	assert.Equal(t, common.String("test-stream"), captured.CreateStreamDetails.Name)
	assert.Equal(t, common.Int(1), captured.CreateStreamDetails.Partitions)
	assert.Equal(t, common.Int(24), captured.CreateStreamDetails.RetentionInHours)
	assert.Equal(t, common.String("ocid1.compartment.oc1..example"), captured.CreateStreamDetails.CompartmentId)
	assert.Equal(t, common.String("ocid1.streampool.oc1..example"), captured.CreateStreamDetails.StreamPoolId)
}

func TestGetCredentialMapForTest(t *testing.T) {
	stream := streaming.Stream{
		MessagesEndpoint: common.String("https://streaming.example.com"),
	}

	credMap, err := GetCredentialMapForTest(stream)
	assert.NoError(t, err)
	assert.Equal(t, "https://streaming.example.com", string(credMap["endpoint"]))
}

func TestGetCrdStatus_HappyPath(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)

	stream := &streamingv1beta1.Stream{}
	stream.Status.OsokStatus.Ocid = "ocid1.stream.oc1..xxx"

	status, err := mgr.GetCrdStatus(stream)
	assert.NoError(t, err)
	assert.Equal(t, "ocid1.stream.oc1..xxx", string(status.Ocid))
}

func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)

	dbSystem := &mysqlv1beta1.DbSystem{}
	_, err := mgr.GetCrdStatus(dbSystem)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert the type assertion for Stream")
}

func TestDelete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mgr := makeTestManager(credClient, nil)

	stream := &streamingv1beta1.Stream{}
	stream.Status.OsokStatus.Ocid = "ocid1.stream.oc1..xxx"

	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled)
}

func TestDelete_WrongType(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)

	dbSystem := &mysqlv1beta1.DbSystem{}
	done, err := mgr.Delete(context.Background(), dbSystem)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestDelete_DeleteStreamFails(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mockClient := &mockStreamAdminClient{
		deleteStreamFn: func(_ context.Context, _ streaming.DeleteStreamRequest) (streaming.DeleteStreamResponse, error) {
			return streaming.DeleteStreamResponse{}, errors.New("oci: network error")
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.StreamId = "ocid1.stream.oc1..xxx"

	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled)
}

func TestCreateOrUpdate_BadType(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)

	dbSystem := &mysqlv1beta1.DbSystem{}
	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestCreateOrUpdate_EmptyName(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)

	stream := &streamingv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestStreamRetryPolicy_Creating(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	shouldRetry := ExportGetStreamRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: streaming.GetStreamResponse{
			Stream: streaming.Stream{LifecycleState: "CREATING"},
		},
	}
	assert.True(t, shouldRetry(resp))
}

func TestStreamRetryPolicy_Active(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	shouldRetry := ExportGetStreamRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: streaming.GetStreamResponse{
			Stream: streaming.Stream{LifecycleState: "ACTIVE"},
		},
	}
	assert.False(t, shouldRetry(resp))
}

func TestStreamRetryPolicy_NonResponse(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	shouldRetry := ExportGetStreamRetryPredicate(mgr)

	assert.True(t, shouldRetry(common.OCIOperationResponse{}))
}

func TestDeleteStreamRetryPolicy_Deleting(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	shouldRetry := ExportDeleteStreamRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: streaming.GetStreamResponse{
			Stream: streaming.Stream{LifecycleState: "DELETING"},
		},
	}
	assert.True(t, shouldRetry(resp))
}

func TestDeleteStreamRetryPolicy_Deleted(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	shouldRetry := ExportDeleteStreamRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: streaming.GetStreamResponse{
			Stream: streaming.Stream{LifecycleState: "DELETED"},
		},
	}
	assert.False(t, shouldRetry(resp))
}

func TestDeleteStreamRetryPolicy_NonResponse(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	shouldRetry := ExportDeleteStreamRetryPredicate(mgr)

	assert.True(t, shouldRetry(common.OCIOperationResponse{}))
}

func TestStreamRetryNextDuration(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	nextDuration := ExportGetStreamNextDuration(mgr)

	assert.Equal(t, 1*time.Second, nextDuration(common.OCIOperationResponse{AttemptNumber: 1}))
}

func TestDeleteStreamRetryNextDuration(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	nextDuration := ExportDeleteStreamNextDuration(mgr)

	assert.Equal(t, 1*time.Second, nextDuration(common.OCIOperationResponse{AttemptNumber: 1}))
}

func TestDelete_StreamDeleted(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..xxx"
	mockClient := &mockStreamAdminClient{
		deleteStreamFn: func(_ context.Context, _ streaming.DeleteStreamRequest) (streaming.DeleteStreamResponse, error) {
			return streaming.DeleteStreamResponse{}, nil
		},
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{
				Stream: streaming.Stream{
					Id:             common.String(streamID),
					Name:           common.String("test-stream"),
					LifecycleState: "DELETED",
				},
			}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.StreamId = shared.OCID(streamID)

	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, credClient.deleteCalled)
}

func TestCreateOrUpdate_BindExistingByID(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..xxx"
	activeStream := makeActiveStream(streamID, "test-stream")

	mockClient := &mockStreamAdminClient{
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: activeStream}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.StreamId = shared.OCID(streamID)

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, credClient.createCalled)
}

func TestCreateOrUpdate_GetStreamFails(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..xxx"

	mockClient := &mockStreamAdminClient{
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{}, errors.New("oci: unauthorized")
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.StreamId = shared.OCID(streamID)

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestCreateOrUpdate_CreateNew(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..new"
	activeStream := makeActiveStream(streamID, "new-stream")

	mockClient := &mockStreamAdminClient{
		listStreamsFn: func(_ context.Context, _ streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
			return streaming.ListStreamsResponse{Items: []streaming.StreamSummary{}}, nil
		},
		createStreamFn: func(_ context.Context, req streaming.CreateStreamRequest) (streaming.CreateStreamResponse, error) {
			return streaming.CreateStreamResponse{
				Stream: streaming.Stream{
					Id:               common.String(streamID),
					Name:             req.Name,
					LifecycleState:   "CREATING",
					MessagesEndpoint: common.String("https://cell-1.streaming.us-phoenix-1.oci.oraclecloud.com"),
					StreamPoolId:     common.String("ocid1.streampool.oc1..xxx"),
					Partitions:       common.Int(1),
					RetentionInHours: common.Int(24),
				},
			}, nil
		},
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: activeStream}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Name = "new-stream"
	stream.Namespace = "default"
	stream.Spec.Name = "new-stream"
	stream.Spec.Partitions = 1

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, credClient.createCalled)
}

func TestCreateOrUpdate_ListStreamsFails(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mockClient := &mockStreamAdminClient{
		listStreamsFn: func(_ context.Context, _ streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
			return streaming.ListStreamsResponse{}, errors.New("oci: service unavailable")
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.Name = "test-stream"
	stream.Spec.Partitions = 1

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestCreateOrUpdate_ExistingByName(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..found"
	activeStream := makeActiveStream(streamID, "named-stream")

	mockClient := &mockStreamAdminClient{
		listStreamsFn: func(_ context.Context, _ streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
			return streaming.ListStreamsResponse{
				Items: []streaming.StreamSummary{
					{Id: common.String(streamID), LifecycleState: "ACTIVE"},
				},
			}, nil
		},
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: activeStream}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Name = "named-stream"
	stream.Namespace = "default"
	stream.Spec.Name = "named-stream"
	stream.Spec.Partitions = 1

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, credClient.createCalled)
}

func TestDelete_EmptyOcidPath(t *testing.T) {
	credClient := &fakeCredentialClient{}
	listCallCount := 0

	mockClient := &mockStreamAdminClient{
		listStreamsFn: func(_ context.Context, _ streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
			listCallCount++
			return streaming.ListStreamsResponse{Items: []streaming.StreamSummary{}}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.Name = "test-stream"

	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, 2, listCallCount)
	assert.False(t, credClient.deleteCalled)
}

func TestDelete_StreamFoundByName(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..named"

	mockClient := &mockStreamAdminClient{
		listStreamsFn: func(_ context.Context, _ streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
			return streaming.ListStreamsResponse{
				Items: []streaming.StreamSummary{
					{Id: common.String(streamID), LifecycleState: "ACTIVE"},
				},
			}, nil
		},
		deleteStreamFn: func(_ context.Context, _ streaming.DeleteStreamRequest) (streaming.DeleteStreamResponse, error) {
			return streaming.DeleteStreamResponse{}, nil
		},
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{
				Stream: streaming.Stream{
					Id:             common.String(streamID),
					Name:           common.String("test-stream"),
					LifecycleState: "DELETED",
				},
			}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.Name = "test-stream"

	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, credClient.deleteCalled)
}

func TestDelete_FailedStreamFound(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..failed"
	callCount := 0

	mockClient := &mockStreamAdminClient{
		listStreamsFn: func(_ context.Context, _ streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
			callCount++
			if callCount == 1 {
				return streaming.ListStreamsResponse{Items: []streaming.StreamSummary{}}, nil
			}
			return streaming.ListStreamsResponse{
				Items: []streaming.StreamSummary{
					{Id: common.String(streamID), LifecycleState: "FAILED"},
				},
			}, nil
		},
		deleteStreamFn: func(_ context.Context, _ streaming.DeleteStreamRequest) (streaming.DeleteStreamResponse, error) {
			return streaming.DeleteStreamResponse{}, nil
		},
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{
				Stream: streaming.Stream{
					Id:             common.String(streamID),
					Name:           common.String("test-stream"),
					LifecycleState: "DELETING",
				},
			}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.Name = "test-stream"

	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestCreateOrUpdate_UpdateViaFreeFormTags(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..upd"
	existingStream := makeActiveStream(streamID, "my-stream")
	updateCalled := false

	mockClient := &mockStreamAdminClient{
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: existingStream}, nil
		},
		updateStreamFn: func(_ context.Context, req streaming.UpdateStreamRequest) (streaming.UpdateStreamResponse, error) {
			updateCalled = true
			assert.Equal(t, streamID, *req.StreamId)
			return streaming.UpdateStreamResponse{}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Name = "my-stream"
	stream.Namespace = "default"
	stream.Spec.StreamId = shared.OCID(streamID)
	stream.Spec.Partitions = 1
	stream.Spec.RetentionInHours = 24
	stream.Spec.FreeFormTags = map[string]string{"env": "prod"}

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled)
}

func TestCreateOrUpdate_UpdateViaDefinedTags(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..deftags"
	existingStream := makeActiveStream(streamID, "tag-stream")
	updateCalled := false

	mockClient := &mockStreamAdminClient{
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: existingStream}, nil
		},
		updateStreamFn: func(_ context.Context, _ streaming.UpdateStreamRequest) (streaming.UpdateStreamResponse, error) {
			updateCalled = true
			return streaming.UpdateStreamResponse{}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Name = "tag-stream"
	stream.Namespace = "default"
	stream.Spec.StreamId = shared.OCID(streamID)
	stream.Spec.Partitions = 1
	stream.Spec.RetentionInHours = 24
	stream.Spec.DefinedTags = map[string]shared.MapValue{
		"ns1": {"key1": "val1"},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled)
}

func TestUpdateStream_PartitionsMismatch(t *testing.T) {
	streamID := "ocid1.stream.oc1..partmm"
	existingStream := makeActiveStream(streamID, "my-stream")

	mockClient := &mockStreamAdminClient{
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: existingStream}, nil
		},
	}
	mgr := makeTestManager(&fakeCredentialClient{}, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Spec.StreamId = shared.OCID(streamID)
	stream.Spec.Partitions = 3

	err := mgr.UpdateStream(context.Background(), stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Partitions can't be updated")
}

func TestUpdateStream_RetentionMismatch(t *testing.T) {
	streamID := "ocid1.stream.oc1..retmm"
	existingStream := makeActiveStream(streamID, "my-stream")

	mockClient := &mockStreamAdminClient{
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: existingStream}, nil
		},
	}
	mgr := makeTestManager(&fakeCredentialClient{}, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Spec.StreamId = shared.OCID(streamID)
	stream.Spec.Partitions = 1
	stream.Spec.RetentionInHours = 12

	err := mgr.UpdateStream(context.Background(), stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RetentionsHours can't be updated")
}

func TestGetStreamOcid_WithOptionalFilters(t *testing.T) {
	credClient := &fakeCredentialClient{}
	listCallCount := 0

	mockClient := &mockStreamAdminClient{
		listStreamsFn: func(_ context.Context, req streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
			listCallCount++
			if listCallCount == 1 {
				assert.NotNil(t, req.StreamPoolId)
				assert.NotNil(t, req.CompartmentId)
			}
			return streaming.ListStreamsResponse{Items: []streaming.StreamSummary{}}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Name = "filtered-stream"
	stream.Namespace = "default"
	stream.Spec.Name = "filtered-stream"
	stream.Spec.StreamPoolId = "ocid1.streampool.oc1..filter"
	stream.Spec.CompartmentId = "ocid1.compartment.oc1..filter"

	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestCreateOrUpdate_FailedLifecycle(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..failed"
	failedStream := makeActiveStream(streamID, "failed-stream")
	failedStream.LifecycleState = "FAILED"

	mockClient := &mockStreamAdminClient{
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: failedStream}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &streamingv1beta1.Stream{}
	stream.Name = "failed-stream"
	stream.Namespace = "default"
	stream.Spec.StreamId = shared.OCID(streamID)

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, credClient.createCalled)
}
