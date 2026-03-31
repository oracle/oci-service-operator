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

const (
	testStreamName      = "test-stream"
	testStreamNamespace = "default"
	activeStreamOCID    = shared.OCID("ocid1.stream.oc1..active")
	quickStreamOCID     = shared.OCID("ocid1.stream.oc1..quick")
	testStreamEndpoint  = "https://streaming.example.com"
	staleStreamEndpoint = "https://old-streaming.example.com"
)

type secretCallExpectation struct {
	get    bool
	create bool
	update bool
	delete bool
}

func newTestStreamResource() *streamingv1beta1.Stream {
	resource := &streamingv1beta1.Stream{}
	resource.Name = testStreamName
	resource.Namespace = testStreamNamespace
	return resource
}

func activeStreamServiceClient(streamID shared.OCID) fakeStreamServiceClient {
	return fakeStreamServiceClient{
		createOrUpdateFn: func(_ context.Context, resource *streamingv1beta1.Stream, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
			resource.Status.OsokStatus.Ocid = streamID
			resource.Status.OsokStatus.Reason = string(shared.Active)
			resource.Status.OsokStatus.Conditions = []shared.OSOKCondition{
				{Type: shared.Active, Status: v1.ConditionTrue},
			}
			return servicemanager.OSOKResponse{IsSuccessful: true}, nil
		},
	}
}

func requireSecretTarget(t *testing.T, action string, name string, namespace string) {
	t.Helper()
	if name != testStreamName || namespace != testStreamNamespace {
		t.Fatalf("%s() target = %s/%s, want %s/%s", action, namespace, name, testStreamNamespace, testStreamName)
	}
}

func requireEndpointSecretData(t *testing.T, data map[string][]byte, wantEndpoint string, label string) {
	t.Helper()
	if got := string(data["endpoint"]); got != wantEndpoint {
		t.Fatalf("%s endpoint = %q, want %s", label, got, wantEndpoint)
	}
}

func assertCredentialCalls(t *testing.T, credClient *fakeCredentialClient, want secretCallExpectation) {
	t.Helper()
	if credClient.getCalled != want.get {
		t.Fatalf("GetSecret() called = %t, want %t", credClient.getCalled, want.get)
	}
	if credClient.createCalled != want.create {
		t.Fatalf("CreateSecret() called = %t, want %t", credClient.createCalled, want.create)
	}
	if credClient.updateCalled != want.update {
		t.Fatalf("UpdateSecret() called = %t, want %t", credClient.updateCalled, want.update)
	}
	if credClient.deleteCalled != want.delete {
		t.Fatalf("DeleteSecret() called = %t, want %t", credClient.deleteCalled, want.delete)
	}
}

func requireCreateOrUpdateSuccess(t *testing.T, response servicemanager.OSOKResponse, err error, detail string) {
	t.Helper()
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() should keep the successful reconcile result after %s", detail)
	}
}

func requireDeleteSuccess(t *testing.T, deleted bool, err error, detail string) {
	t.Helper()
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatalf("Delete() should report success after %s", detail)
	}
}

func staticStreamLoader(t *testing.T, wantID shared.OCID, endpoint string) func(context.Context, shared.OCID) (*streamingsdk.Stream, error) {
	t.Helper()
	return func(_ context.Context, streamID shared.OCID) (*streamingsdk.Stream, error) {
		if streamID != wantID {
			t.Fatalf("loadStream() streamID = %q, want %q", streamID, wantID)
		}
		return &streamingsdk.Stream{
			MessagesEndpoint: common.String(endpoint),
		}, nil
	}
}

func newActiveStreamEndpointSecretClient(t *testing.T, credClient *fakeCredentialClient, endpoint string) streamEndpointSecretClient {
	t.Helper()
	return streamEndpointSecretClient{
		delegate:         activeStreamServiceClient(activeStreamOCID),
		credentialClient: credClient,
		loadStream:       staticStreamLoader(t, activeStreamOCID, endpoint),
	}
}

func validateQuickSecretTarget(action string, name string, namespace string) error {
	if name != testStreamName || namespace != testStreamNamespace {
		return fmt.Errorf("%s() target=%s/%s, want %s/%s", action, namespace, name, testStreamNamespace, testStreamName)
	}
	return nil
}

func TestStreamEndpointSecretClientCreatesSecretAfterActiveReconcile(t *testing.T) {
	t.Parallel()

	credClient := &fakeCredentialClient{}
	var createdData map[string][]byte
	credClient.createSecretFn = func(_ context.Context, name string, namespace string, _ map[string]string, data map[string][]byte) (bool, error) {
		requireSecretTarget(t, "CreateSecret", name, namespace)
		createdData = data
		return true, nil
	}

	client := newActiveStreamEndpointSecretClient(t, credClient, testStreamEndpoint)
	resource := newTestStreamResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err, "secret sync")
	assertCredentialCalls(t, credClient, secretCallExpectation{get: true, create: true})
	requireEndpointSecretData(t, createdData, testStreamEndpoint, "secret")
}

func TestStreamEndpointSecretClientRecoversFromCreateAlreadyExistsAfterStaleRead(t *testing.T) {
	t.Parallel()

	credClient := &fakeCredentialClient{}
	getCalls := 0
	credClient.getSecretFn = func(_ context.Context, name string, namespace string) (map[string][]byte, error) {
		requireSecretTarget(t, "GetSecret", name, namespace)
		getCalls++
		if getCalls == 1 {
			return nil, apierrors.NewNotFound(v1.Resource("secret"), name)
		}
		return map[string][]byte{
			"endpoint": []byte(testStreamEndpoint),
		}, nil
	}
	credClient.createSecretFn = func(_ context.Context, name string, namespace string, _ map[string]string, data map[string][]byte) (bool, error) {
		requireSecretTarget(t, "CreateSecret", name, namespace)
		requireEndpointSecretData(t, data, testStreamEndpoint, "secret")
		return false, apierrors.NewAlreadyExists(v1.Resource("secret"), name)
	}
	credClient.updateSecretFn = func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
		t.Fatal("UpdateSecret() should not be called when the already-created companion secret already matches")
		return false, nil
	}

	client := newActiveStreamEndpointSecretClient(t, credClient, testStreamEndpoint)
	resource := newTestStreamResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err, "recovering from a stale read/create race")
	if getCalls != 2 {
		t.Fatalf("GetSecret() calls = %d, want 2 to cover the stale read and follow-up read", getCalls)
	}
	assertCredentialCalls(t, credClient, secretCallExpectation{get: true, create: true})
}

func TestStreamEndpointSecretClientSkipsSecretUpdateWhenExistingDataMatches(t *testing.T) {
	t.Parallel()

	credClient := &fakeCredentialClient{}
	credClient.getSecretFn = func(_ context.Context, name string, namespace string) (map[string][]byte, error) {
		requireSecretTarget(t, "GetSecret", name, namespace)
		return map[string][]byte{
			"endpoint": []byte(testStreamEndpoint),
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

	client := newActiveStreamEndpointSecretClient(t, credClient, testStreamEndpoint)
	resource := newTestStreamResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err, "a no-op secret sync")
	assertCredentialCalls(t, credClient, secretCallExpectation{get: true})
}

func TestStreamEndpointSecretClientUpdatesExistingSecretWhenEndpointChanges(t *testing.T) {
	t.Parallel()

	credClient := &fakeCredentialClient{}
	var updatedData map[string][]byte
	credClient.getSecretFn = func(_ context.Context, name string, namespace string) (map[string][]byte, error) {
		requireSecretTarget(t, "GetSecret", name, namespace)
		return map[string][]byte{
			"endpoint": []byte(staleStreamEndpoint),
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

	client := newActiveStreamEndpointSecretClient(t, credClient, testStreamEndpoint)
	resource := newTestStreamResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireCreateOrUpdateSuccess(t, response, err, "secret update")
	assertCredentialCalls(t, credClient, secretCallExpectation{get: true, update: true})
	requireEndpointSecretData(t, updatedData, testStreamEndpoint, "updated secret")
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

	resource := newTestStreamResource()

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

	resource := newTestStreamResource()

	deleted, err := client.Delete(context.Background(), resource)
	requireDeleteSuccess(t, deleted, err, "secret cleanup")
	assertCredentialCalls(t, credClient, secretCallExpectation{delete: true})
}

func TestStreamEndpointSecretClientDeleteIgnoresMissingSecret(t *testing.T) {
	t.Parallel()

	credClient := &fakeCredentialClient{}
	credClient.deleteSecretFn = func(_ context.Context, name string, namespace string) (bool, error) {
		if name != "test-stream" || namespace != "default" {
			t.Fatalf("DeleteSecret() target = %s/%s, want default/test-stream", namespace, name)
		}
		return false, apierrors.NewNotFound(v1.Resource("secret"), name)
	}

	client := streamEndpointSecretClient{
		delegate: fakeStreamServiceClient{
			deleteFn: func(_ context.Context, _ *streamingv1beta1.Stream) (bool, error) {
				return true, nil
			},
		},
		credentialClient: credClient,
	}

	resource := newTestStreamResource()

	deleted, err := client.Delete(context.Background(), resource)
	requireDeleteSuccess(t, deleted, err, "a missing companion secret")
	assertCredentialCalls(t, credClient, secretCallExpectation{delete: true})
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

type streamEndpointSecretDeleteQuickCase struct {
	InitialPresent bool
	DeleteRace     bool
}

func (streamEndpointSecretDeleteQuickCase) Generate(r *rand.Rand, _ int) reflect.Value {
	return reflect.ValueOf(streamEndpointSecretDeleteQuickCase{
		InitialPresent: r.Intn(2) == 0,
		DeleteRace:     r.Intn(2) == 0,
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

func TestStreamEndpointSecretClientQuickDeleteIsBestEffort(t *testing.T) {
	t.Parallel()

	var evalErr error
	if err := quick.Check(func(tc streamEndpointSecretDeleteQuickCase) bool {
		evalErr = evaluateStreamEndpointSecretDeleteQuickCase(tc)
		return evalErr == nil
	}, streamEndpointSecretQuickConfig(1774907911310276)); err != nil {
		t.Fatalf("stream endpoint secret delete property failed: %v: %v", err, evalErr)
	}
}

func evaluateStreamEndpointSecretQuickCase(tc streamEndpointSecretQuickCase) error {
	return newStreamEndpointSecretQuickHarness(tc).run()
}

func evaluateStreamEndpointSecretDeleteQuickCase(tc streamEndpointSecretDeleteQuickCase) error {
	return newStreamEndpointSecretDeleteQuickHarness(tc).run()
}

type streamEndpointSecretQuickHarness struct {
	tc              streamEndpointSecretQuickCase
	resource        *streamingv1beta1.Stream
	desiredEndpoint string
	store           map[string][]byte
	createCalls     int
	updateCalls     int
	getCalls        int
	client          streamEndpointSecretClient
}

func newStreamEndpointSecretQuickHarness(tc streamEndpointSecretQuickCase) *streamEndpointSecretQuickHarness {
	h := &streamEndpointSecretQuickHarness{
		tc:              tc,
		resource:        newTestStreamResource(),
		desiredEndpoint: fmt.Sprintf("https://streaming-%d.example.com", tc.EndpointID),
	}
	h.resource.Status.OsokStatus.Ocid = quickStreamOCID
	h.store = initialQuickSecretStore(tc, h.desiredEndpoint)

	credClient := &fakeCredentialClient{
		getSecretFn:    h.getSecret,
		createSecretFn: h.createSecret,
		updateSecretFn: h.updateSecret,
	}
	h.client = streamEndpointSecretClient{
		credentialClient: credClient,
		loadStream:       h.loadStream,
	}
	return h
}

func (h *streamEndpointSecretQuickHarness) getSecret(_ context.Context, name string, namespace string) (map[string][]byte, error) {
	h.getCalls++
	if err := validateQuickSecretTarget("GetSecret", name, namespace); err != nil {
		return nil, err
	}
	if h.store == nil {
		return nil, apierrors.NewNotFound(v1.Resource("secret"), name)
	}
	if h.tc.CachedNotFound && h.getCalls == 1 {
		return nil, apierrors.NewNotFound(v1.Resource("secret"), name)
	}
	return cloneSecretData(h.store), nil
}

func (h *streamEndpointSecretQuickHarness) createSecret(_ context.Context, name string, namespace string, _ map[string]string, data map[string][]byte) (bool, error) {
	h.createCalls++
	if err := validateQuickSecretTarget("CreateSecret", name, namespace); err != nil {
		return false, err
	}
	if h.store != nil {
		return false, apierrors.NewAlreadyExists(v1.Resource("secret"), name)
	}
	h.store = cloneSecretData(data)
	return true, nil
}

func (h *streamEndpointSecretQuickHarness) updateSecret(_ context.Context, name string, namespace string, _ map[string]string, data map[string][]byte) (bool, error) {
	h.updateCalls++
	if err := validateQuickSecretTarget("UpdateSecret", name, namespace); err != nil {
		return false, err
	}
	if h.store == nil {
		return false, apierrors.NewNotFound(v1.Resource("secret"), name)
	}
	h.store = cloneSecretData(data)
	return true, nil
}

func (h *streamEndpointSecretQuickHarness) loadStream(_ context.Context, streamID shared.OCID) (*streamingsdk.Stream, error) {
	if streamID != quickStreamOCID {
		return nil, fmt.Errorf("loadStream() streamID=%q, want quick stream OCID", streamID)
	}
	return &streamingsdk.Stream{
		MessagesEndpoint: common.String(h.desiredEndpoint),
	}, nil
}

func (h *streamEndpointSecretQuickHarness) run() error {
	if err := h.syncOnce("first"); err != nil {
		return err
	}
	if err := h.syncOnce("second"); err != nil {
		return err
	}
	if err := h.assertStoredSecret(); err != nil {
		return err
	}
	return h.assertCallCounts()
}

func (h *streamEndpointSecretQuickHarness) syncOnce(label string) error {
	if err := h.client.syncEndpointSecret(context.Background(), h.resource); err != nil {
		return fmt.Errorf("%s sync: %w", label, err)
	}
	return nil
}

func (h *streamEndpointSecretQuickHarness) assertStoredSecret() error {
	wantData := map[string][]byte{
		"endpoint": []byte(h.desiredEndpoint),
	}
	if !reflect.DeepEqual(h.store, wantData) {
		return fmt.Errorf("final secret data=%v, want %v for %+v", h.store, wantData, h.tc)
	}
	return nil
}

func (h *streamEndpointSecretQuickHarness) assertCallCounts() error {
	wantCreate, wantUpdate := expectedQuickSyncCallCounts(h.tc)
	if h.createCalls != wantCreate || h.updateCalls != wantUpdate {
		return fmt.Errorf("calls create=%d update=%d, want create=%d update=%d for %+v", h.createCalls, h.updateCalls, wantCreate, wantUpdate, h.tc)
	}
	return nil
}

func expectedQuickSyncCallCounts(tc streamEndpointSecretQuickCase) (int, int) {
	switch tc.InitialState % 3 {
	case 0:
		return 1, 0
	case 1:
		if tc.CachedNotFound {
			return 1, 0
		}
		return 0, 0
	default:
		if tc.CachedNotFound {
			return 1, 1
		}
		return 0, 1
	}
}

type streamEndpointSecretDeleteQuickHarness struct {
	tc            streamEndpointSecretDeleteQuickCase
	resource      *streamingv1beta1.Stream
	secretPresent bool
	deleteCalls   int
	client        streamEndpointSecretClient
}

func newStreamEndpointSecretDeleteQuickHarness(tc streamEndpointSecretDeleteQuickCase) *streamEndpointSecretDeleteQuickHarness {
	h := &streamEndpointSecretDeleteQuickHarness{
		tc:            tc,
		resource:      newTestStreamResource(),
		secretPresent: tc.InitialPresent,
	}
	credClient := &fakeCredentialClient{
		deleteSecretFn: h.deleteSecret,
	}
	h.client = streamEndpointSecretClient{
		delegate: fakeStreamServiceClient{
			deleteFn: h.deleteResource,
		},
		credentialClient: credClient,
	}
	return h
}

func (h *streamEndpointSecretDeleteQuickHarness) deleteSecret(_ context.Context, name string, namespace string) (bool, error) {
	h.deleteCalls++
	if err := validateQuickSecretTarget("DeleteSecret", name, namespace); err != nil {
		return false, err
	}
	if !h.secretPresent {
		return false, apierrors.NewNotFound(v1.Resource("secret"), name)
	}
	h.secretPresent = false
	if h.tc.DeleteRace && h.deleteCalls == 1 {
		return false, apierrors.NewNotFound(v1.Resource("secret"), name)
	}
	return true, nil
}

func (h *streamEndpointSecretDeleteQuickHarness) deleteResource(_ context.Context, resource *streamingv1beta1.Stream) (bool, error) {
	if resource != nil && (resource.Name != testStreamName || resource.Namespace != testStreamNamespace) {
		return false, fmt.Errorf("Delete() target=%s/%s, want %s/%s", resource.Namespace, resource.Name, testStreamNamespace, testStreamName)
	}
	return true, nil
}

func (h *streamEndpointSecretDeleteQuickHarness) run() error {
	for attempt := 1; attempt <= 2; attempt++ {
		if err := h.deleteOnce(attempt); err != nil {
			return err
		}
	}
	if h.secretPresent {
		return fmt.Errorf("secret still present after repeated delete completion for %+v", h.tc)
	}
	if h.deleteCalls != 2 {
		return fmt.Errorf("DeleteSecret() calls=%d, want 2 repeated best-effort attempts for %+v", h.deleteCalls, h.tc)
	}
	return nil
}

func (h *streamEndpointSecretDeleteQuickHarness) deleteOnce(attempt int) error {
	deleted, err := h.client.Delete(context.Background(), h.resource)
	if err != nil {
		return fmt.Errorf("delete attempt %d: %w", attempt, err)
	}
	if !deleted {
		return fmt.Errorf("delete attempt %d: deleted=false for %+v", attempt, h.tc)
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
