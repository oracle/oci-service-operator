/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package stream

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	"github.com/oracle/oci-go-sdk/v65/common"
	streamingsdk "github.com/oracle/oci-go-sdk/v65/streaming"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeCredentialClient struct {
	createSecretFn func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error)
	deleteSecretFn func(context.Context, string, string) (bool, error)
	getSecretFn    func(context.Context, string, string) (map[string][]byte, error)
	updateSecretFn func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error)
	createCalled   bool
	deleteCalled   bool
	getCalled      bool
	updateCalled   bool
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
	f.getCalled = true
	if f.getSecretFn != nil {
		return f.getSecretFn(ctx, name, namespace)
	}
	return nil, apierrors.NewNotFound(v1.Resource("secret"), name)
}

func (f *fakeCredentialClient) UpdateSecret(
	ctx context.Context,
	name string,
	namespace string,
	labels map[string]string,
	data map[string][]byte,
) (bool, error) {
	f.updateCalled = true
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
	if !credClient.getCalled {
		t.Fatal("GetSecret() should be called before creating the companion secret")
	}
	if !credClient.createCalled {
		t.Fatal("CreateSecret() should be called after an ACTIVE reconcile")
	}
	if credClient.updateCalled {
		t.Fatal("UpdateSecret() should not be called when the companion secret does not exist")
	}
	if got := string(createdData["endpoint"]); got != "https://streaming.example.com" {
		t.Fatalf("secret endpoint = %q, want https://streaming.example.com", got)
	}
}

func TestStreamEndpointSecretClientRecoversFromCreateAlreadyExistsAfterStaleRead(t *testing.T) {
	t.Parallel()

	credClient := &fakeCredentialClient{}
	getCalls := 0
	credClient.getSecretFn = func(_ context.Context, name string, namespace string) (map[string][]byte, error) {
		if name != "test-stream" || namespace != "default" {
			t.Fatalf("GetSecret() target = %s/%s, want default/test-stream", namespace, name)
		}
		getCalls++
		if getCalls == 1 {
			return nil, apierrors.NewNotFound(v1.Resource("secret"), name)
		}
		return map[string][]byte{
			"endpoint": []byte("https://streaming.example.com"),
		}, nil
	}
	credClient.createSecretFn = func(_ context.Context, name string, namespace string, _ map[string]string, data map[string][]byte) (bool, error) {
		if name != "test-stream" || namespace != "default" {
			t.Fatalf("CreateSecret() target = %s/%s, want default/test-stream", namespace, name)
		}
		if got := string(data["endpoint"]); got != "https://streaming.example.com" {
			t.Fatalf("secret endpoint = %q, want https://streaming.example.com", got)
		}
		return false, apierrors.NewAlreadyExists(v1.Resource("secret"), name)
	}
	credClient.updateSecretFn = func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
		t.Fatal("UpdateSecret() should not be called when the already-created companion secret already matches")
		return false, nil
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
		t.Fatal("CreateOrUpdate() should keep the successful reconcile result after recovering from a stale read/create race")
	}
	if getCalls != 2 {
		t.Fatalf("GetSecret() calls = %d, want 2 to cover the stale read and follow-up read", getCalls)
	}
	if !credClient.createCalled {
		t.Fatal("CreateSecret() should be attempted after the stale NotFound read")
	}
	if credClient.updateCalled {
		t.Fatal("UpdateSecret() should not be called when the companion secret already matches after the create race")
	}
}

func TestStreamEndpointSecretClientSkipsSecretUpdateWhenExistingDataMatches(t *testing.T) {
	t.Parallel()

	credClient := &fakeCredentialClient{}
	credClient.getSecretFn = func(_ context.Context, name string, namespace string) (map[string][]byte, error) {
		if name != "test-stream" || namespace != "default" {
			t.Fatalf("GetSecret() target = %s/%s, want default/test-stream", namespace, name)
		}
		return map[string][]byte{
			"endpoint": []byte("https://streaming.example.com"),
		}, nil
	}
	credClient.createSecretFn = func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
		t.Fatal("CreateSecret() should not be called when the companion secret is already current")
		return false, nil
	}
	credClient.updateSecretFn = func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
		t.Fatal("UpdateSecret() should not be called when the companion secret is already current")
		return false, nil
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
		t.Fatal("CreateOrUpdate() should keep the successful reconcile result after a no-op secret sync")
	}
	if !credClient.getCalled {
		t.Fatal("GetSecret() should be called before determining whether secret sync is necessary")
	}
	if credClient.createCalled {
		t.Fatal("CreateSecret() should not be called when the companion secret already matches")
	}
	if credClient.updateCalled {
		t.Fatal("UpdateSecret() should not be called when the companion secret already matches")
	}
}

func TestStreamEndpointSecretClientUpdatesExistingSecretWhenEndpointChanges(t *testing.T) {
	t.Parallel()

	credClient := &fakeCredentialClient{}
	var updatedData map[string][]byte
	credClient.getSecretFn = func(_ context.Context, name string, namespace string) (map[string][]byte, error) {
		if name != "test-stream" || namespace != "default" {
			t.Fatalf("GetSecret() target = %s/%s, want default/test-stream", namespace, name)
		}
		return map[string][]byte{
			"endpoint": []byte("https://old-streaming.example.com"),
		}, nil
	}
	credClient.createSecretFn = func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
		t.Fatal("CreateSecret() should not be called when the companion secret already exists")
		return false, nil
	}
	credClient.updateSecretFn = func(_ context.Context, name string, namespace string, _ map[string]string, data map[string][]byte) (bool, error) {
		if name != "test-stream" || namespace != "default" {
			t.Fatalf("UpdateSecret() target = %s/%s, want default/test-stream", namespace, name)
		}
		updatedData = data
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
		t.Fatal("CreateOrUpdate() should keep the successful reconcile result after secret update")
	}
	if !credClient.getCalled {
		t.Fatal("GetSecret() should be called before updating the companion secret")
	}
	if credClient.createCalled {
		t.Fatal("CreateSecret() should not be called when the companion secret already exists")
	}
	if !credClient.updateCalled {
		t.Fatal("UpdateSecret() should be called when the companion secret endpoint drifts")
	}
	if got := string(updatedData["endpoint"]); got != "https://streaming.example.com" {
		t.Fatalf("updated secret endpoint = %q, want https://streaming.example.com", got)
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

type streamEndpointSecretQuickCase struct {
	InitialState   uint8
	EndpointID     uint32
	ExtraKey       bool
	CachedNotFound bool
}

func (streamEndpointSecretQuickCase) Generate(r *rand.Rand, _ int) reflect.Value {
	return reflect.ValueOf(streamEndpointSecretQuickCase{
		InitialState:   uint8(r.Intn(3)),
		EndpointID:     r.Uint32(),
		ExtraKey:       r.Intn(2) == 0,
		CachedNotFound: r.Intn(2) == 0,
	})
}

func TestStreamEndpointSecretClientQuickSyncIsIdempotent(t *testing.T) {
	t.Parallel()

	var evalErr error
	if err := quick.Check(func(tc streamEndpointSecretQuickCase) bool {
		evalErr = evaluateStreamEndpointSecretQuickCase(tc)
		return evalErr == nil
	}, streamEndpointSecretQuickConfig(1774907911310275)); err != nil {
		t.Fatalf("stream endpoint secret idempotence property failed: %v: %v", err, evalErr)
	}
}

func evaluateStreamEndpointSecretQuickCase(tc streamEndpointSecretQuickCase) error {
	const secretName = "test-stream"
	const secretNamespace = "default"

	resource := &streamingv1beta1.Stream{}
	resource.Name = secretName
	resource.Namespace = secretNamespace
	resource.Status.OsokStatus.Ocid = "ocid1.stream.oc1..quick"

	desiredEndpoint := fmt.Sprintf("https://streaming-%d.example.com", tc.EndpointID)
	store := initialQuickSecretStore(tc, desiredEndpoint)
	credClient := &fakeCredentialClient{}

	createCalls := 0
	updateCalls := 0
	getCalls := 0
	credClient.getSecretFn = func(_ context.Context, name string, namespace string) (map[string][]byte, error) {
		getCalls++
		if name != secretName || namespace != secretNamespace {
			return nil, fmt.Errorf("GetSecret() target=%s/%s, want %s/%s", namespace, name, secretNamespace, secretName)
		}
		if store == nil {
			return nil, apierrors.NewNotFound(v1.Resource("secret"), name)
		}
		if tc.CachedNotFound && getCalls == 1 {
			return nil, apierrors.NewNotFound(v1.Resource("secret"), name)
		}
		return cloneSecretData(store), nil
	}
	credClient.createSecretFn = func(_ context.Context, name string, namespace string, _ map[string]string, data map[string][]byte) (bool, error) {
		createCalls++
		if name != secretName || namespace != secretNamespace {
			return false, fmt.Errorf("CreateSecret() target=%s/%s, want %s/%s", namespace, name, secretNamespace, secretName)
		}
		if store != nil {
			return false, apierrors.NewAlreadyExists(v1.Resource("secret"), name)
		}
		store = cloneSecretData(data)
		return true, nil
	}
	credClient.updateSecretFn = func(_ context.Context, name string, namespace string, _ map[string]string, data map[string][]byte) (bool, error) {
		updateCalls++
		if name != secretName || namespace != secretNamespace {
			return false, fmt.Errorf("UpdateSecret() target=%s/%s, want %s/%s", namespace, name, secretNamespace, secretName)
		}
		if store == nil {
			return false, apierrors.NewNotFound(v1.Resource("secret"), name)
		}
		store = cloneSecretData(data)
		return true, nil
	}

	client := streamEndpointSecretClient{
		credentialClient: credClient,
		loadStream: func(_ context.Context, streamID shared.OCID) (*streamingsdk.Stream, error) {
			if streamID != "ocid1.stream.oc1..quick" {
				return nil, fmt.Errorf("loadStream() streamID=%q, want quick stream OCID", streamID)
			}
			return &streamingsdk.Stream{
				MessagesEndpoint: common.String(desiredEndpoint),
			}, nil
		},
	}

	if err := client.syncEndpointSecret(context.Background(), resource); err != nil {
		return fmt.Errorf("first sync: %w", err)
	}
	if err := client.syncEndpointSecret(context.Background(), resource); err != nil {
		return fmt.Errorf("second sync: %w", err)
	}

	wantData := map[string][]byte{
		"endpoint": []byte(desiredEndpoint),
	}
	if !reflect.DeepEqual(store, wantData) {
		return fmt.Errorf("final secret data=%v, want %v for %+v", store, wantData, tc)
	}

	switch tc.InitialState % 3 {
	case 0:
		if createCalls != 1 || updateCalls != 0 {
			return fmt.Errorf("calls create=%d update=%d, want create=1 update=0 for %+v", createCalls, updateCalls, tc)
		}
	case 1:
		wantCreate := 0
		if tc.CachedNotFound {
			wantCreate = 1
		}
		if createCalls != wantCreate || updateCalls != 0 {
			return fmt.Errorf("calls create=%d update=%d, want create=%d update=0 for %+v", createCalls, updateCalls, wantCreate, tc)
		}
	default:
		wantCreate := 0
		if tc.CachedNotFound {
			wantCreate = 1
		}
		if createCalls != wantCreate || updateCalls != 1 {
			return fmt.Errorf("calls create=%d update=%d, want create=%d update=1 for %+v", createCalls, updateCalls, wantCreate, tc)
		}
	}

	return nil
}

func initialQuickSecretStore(tc streamEndpointSecretQuickCase, desiredEndpoint string) map[string][]byte {
	switch tc.InitialState % 3 {
	case 0:
		return nil
	case 1:
		return map[string][]byte{
			"endpoint": []byte(desiredEndpoint),
		}
	default:
		store := map[string][]byte{
			"endpoint": []byte("https://stale-streaming.example.com"),
		}
		if tc.ExtraKey {
			store["stale"] = []byte("value")
		}
		return store
	}
}

func cloneSecretData(data map[string][]byte) map[string][]byte {
	if data == nil {
		return nil
	}
	cloned := make(map[string][]byte, len(data))
	for key, value := range data {
		cloned[key] = append([]byte(nil), value...)
	}
	return cloned
}

func streamEndpointSecretQuickConfig(seed int64) *quick.Config {
	return &quick.Config{
		MaxCount: 96,
		Rand:     rand.New(rand.NewSource(seed)),
	}
}
