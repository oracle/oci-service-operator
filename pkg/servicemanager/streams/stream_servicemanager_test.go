/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package streams_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/streaming"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/streams"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type mockStreamAdminClient struct {
	createStreamFn func(context.Context, streaming.CreateStreamRequest) (streaming.CreateStreamResponse, error)
}

func (m *mockStreamAdminClient) CreateStream(ctx context.Context, req streaming.CreateStreamRequest) (streaming.CreateStreamResponse, error) {
	if m.createStreamFn != nil {
		return m.createStreamFn(ctx, req)
	}
	return streaming.CreateStreamResponse{}, nil
}

func (m *mockStreamAdminClient) GetStream(context.Context, streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
	return streaming.GetStreamResponse{}, nil
}

func (m *mockStreamAdminClient) ListStreams(context.Context, streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
	return streaming.ListStreamsResponse{}, nil
}

func (m *mockStreamAdminClient) UpdateStream(context.Context, streaming.UpdateStreamRequest) (streaming.UpdateStreamResponse, error) {
	return streaming.UpdateStreamResponse{}, nil
}

func (m *mockStreamAdminClient) DeleteStream(context.Context, streaming.DeleteStreamRequest) (streaming.DeleteStreamResponse, error) {
	return streaming.DeleteStreamResponse{}, nil
}

type fakeCredentialClient struct{}

func (f *fakeCredentialClient) CreateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(context.Context, string, string) (bool, error) {
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(context.Context, string, string) (map[string][]byte, error) {
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
	return true, nil
}

func newTestManager(mockClient *mockStreamAdminClient) *StreamServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	m := &metrics.Metrics{Logger: log}
	mgr := NewStreamServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), &fakeCredentialClient{}, nil, log, m)
	if mockClient != nil {
		ExportSetClientForTest(mgr, mockClient)
	}
	return mgr
}

func TestCreateStream_UsesInjectedClientAndRetention(t *testing.T) {
	var captured streaming.CreateStreamRequest
	mgr := newTestManager(&mockStreamAdminClient{
		createStreamFn: func(_ context.Context, req streaming.CreateStreamRequest) (streaming.CreateStreamResponse, error) {
			captured = req
			return streaming.CreateStreamResponse{}, nil
		},
	})

	stream := ociv1beta1.Stream{
		Spec: ociv1beta1.StreamSpec{
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
