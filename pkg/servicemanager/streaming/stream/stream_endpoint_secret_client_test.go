/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package stream

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	streamingsdk "github.com/oracle/oci-go-sdk/v65/streaming"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeCredentialClient struct {
	createSecretFn func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error)
	deleteSecretFn func(context.Context, string, string) (bool, error)
	getSecretFn    func(context.Context, string, string) (map[string][]byte, error)
	updateSecretFn func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error)
	createCalled   bool
	deleteCalled   bool
}

func (f *fakeCredentialClient) CreateSecret(
	ctx context.Context,
	name string,
	namespace string,
	labels map[string]string,
	data map[string][]byte,
) (bool, error) {
	f.createCalled = true
	if f.createSecretFn != nil {
		return f.createSecretFn(ctx, name, namespace, labels, data)
	}
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(ctx context.Context, name string, namespace string) (bool, error) {
	f.deleteCalled = true
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, namespace)
	}
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(ctx context.Context, name string, namespace string) (map[string][]byte, error) {
	if f.getSecretFn != nil {
		return f.getSecretFn(ctx, name, namespace)
	}
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(
	ctx context.Context,
	name string,
	namespace string,
	labels map[string]string,
	data map[string][]byte,
) (bool, error) {
	if f.updateSecretFn != nil {
		return f.updateSecretFn(ctx, name, namespace, labels, data)
	}
	return true, nil
}

type fakeStreamServiceClient struct {
	createOrUpdateFn func(context.Context, *streamingv1beta1.Stream, ctrl.Request) (servicemanager.OSOKResponse, error)
	deleteFn         func(context.Context, *streamingv1beta1.Stream) (bool, error)
}

func (f fakeStreamServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *streamingv1beta1.Stream,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if f.createOrUpdateFn != nil {
		return f.createOrUpdateFn(ctx, resource, req)
	}
	return servicemanager.OSOKResponse{}, nil
}

func (f fakeStreamServiceClient) Delete(ctx context.Context, resource *streamingv1beta1.Stream) (bool, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, resource)
	}
	return false, nil
}

func TestStreamEndpointSecretClientCreatesSecretAfterActiveReconcile(t *testing.T) {
	t.Parallel()

	credClient := &fakeCredentialClient{}
	var createdData map[string][]byte
	credClient.createSecretFn = func(_ context.Context, name string, namespace string, _ map[string]string, data map[string][]byte) (bool, error) {
		if name != "test-stream" || namespace != "default" {
			t.Fatalf("CreateSecret() target = %s/%s, want default/test-stream", namespace, name)
		}
		createdData = data
		return true, nil
	}

	client := streamEndpointSecretClient{
		delegate: fakeStreamServiceClient{
			createOrUpdateFn: func(_ context.Context, resource *streamingv1beta1.Stream, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
				resource.Status.OsokStatus.Ocid = "ocid1.stream.oc1..active"
				resource.Status.OsokStatus.Reason = string(shared.Active)
				resource.Status.OsokStatus.Conditions = []shared.OSOKCondition{
					{Type: shared.Active, Status: v1.ConditionTrue},
				}
				return servicemanager.OSOKResponse{IsSuccessful: true}, nil
			},
		},
		credentialClient: credClient,
		loadStream: func(_ context.Context, streamID shared.OCID) (*streamingsdk.Stream, error) {
			if streamID != "ocid1.stream.oc1..active" {
				t.Fatalf("loadStream() streamID = %q, want active OCID", streamID)
			}
			return &streamingsdk.Stream{
				MessagesEndpoint: common.String("https://streaming.example.com"),
			}, nil
		},
	}

	resource := &streamingv1beta1.Stream{}
	resource.Name = "test-stream"
	resource.Namespace = "default"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatal("CreateOrUpdate() should keep the successful reconcile result after secret sync")
	}
	if !credClient.createCalled {
		t.Fatal("CreateSecret() should be called after an ACTIVE reconcile")
	}
	if got := string(createdData["endpoint"]); got != "https://streaming.example.com" {
		t.Fatalf("secret endpoint = %q, want https://streaming.example.com", got)
	}
}

func TestStreamEndpointSecretClientSkipsSecretUntilActive(t *testing.T) {
	t.Parallel()

	credClient := &fakeCredentialClient{}
	client := streamEndpointSecretClient{
		delegate: fakeStreamServiceClient{
			createOrUpdateFn: func(_ context.Context, resource *streamingv1beta1.Stream, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
				resource.Status.OsokStatus.Ocid = "ocid1.stream.oc1..creating"
				resource.Status.OsokStatus.Reason = string(shared.Provisioning)
				resource.Status.OsokStatus.Conditions = []shared.OSOKCondition{
					{Type: shared.Provisioning, Status: v1.ConditionTrue},
				}
				return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true}, nil
			},
		},
		credentialClient: credClient,
		loadStream: func(_ context.Context, _ shared.OCID) (*streamingsdk.Stream, error) {
			t.Fatal("loadStream() should not be called before the stream reaches ACTIVE")
			return nil, nil
		},
	}

	resource := &streamingv1beta1.Stream{}
	resource.Name = "test-stream"
	resource.Namespace = "default"

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful || !response.ShouldRequeue {
		t.Fatalf("CreateOrUpdate() response = %#v, want successful provisioning requeue", response)
	}
	if credClient.createCalled {
		t.Fatal("CreateSecret() should not be called before the stream reaches ACTIVE")
	}
}

func TestStreamEndpointSecretClientDeletesSecretAfterDelete(t *testing.T) {
	t.Parallel()

	credClient := &fakeCredentialClient{}
	credClient.deleteSecretFn = func(_ context.Context, name string, namespace string) (bool, error) {
		if name != "test-stream" || namespace != "default" {
			t.Fatalf("DeleteSecret() target = %s/%s, want default/test-stream", namespace, name)
		}
		return true, nil
	}

	client := streamEndpointSecretClient{
		delegate: fakeStreamServiceClient{
			deleteFn: func(_ context.Context, _ *streamingv1beta1.Stream) (bool, error) {
				return true, nil
			},
		},
		credentialClient: credClient,
	}

	resource := &streamingv1beta1.Stream{}
	resource.Name = "test-stream"
	resource.Namespace = "default"

	deleted, err := client.Delete(context.Background(), resource)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatal("Delete() should report success after secret cleanup")
	}
	if !credClient.deleteCalled {
		t.Fatal("DeleteSecret() should be called after successful delete")
	}
}

func TestStreamEndpointSecretDataRequiresMessagesEndpoint(t *testing.T) {
	t.Parallel()

	if _, err := streamEndpointSecretData(streamingsdk.Stream{}); err == nil || !strings.Contains(err.Error(), "messagesEndpoint is not available") {
		t.Fatalf("streamEndpointSecretData() error = %v, want missing endpoint failure", err)
	}
}
