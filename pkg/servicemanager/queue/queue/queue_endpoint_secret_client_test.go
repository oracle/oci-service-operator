/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package queue

import (
	"context"
	"strings"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	queuev1beta1 "github.com/oracle/oci-service-operator/api/queue/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeQueueCredentialClient struct {
	createSecretFn          func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error)
	deleteSecretFn          func(context.Context, string, string) (bool, error)
	deleteSecretIfCurrentFn func(context.Context, string, string, credhelper.SecretRecord) (bool, error)
	getSecretFn             func(context.Context, string, string) (map[string][]byte, error)
	getSecretRecordFn       func(context.Context, string, string) (credhelper.SecretRecord, error)
	updateSecretFn          func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error)
	updateSecretIfCurrentFn func(context.Context, string, string, credhelper.SecretRecord, map[string]string, map[string][]byte) (bool, error)
	defaultSecretLabels     map[string]string
	defaultSecretUID        string
	createCalled            bool
	deleteCalled            bool
	getCalled               bool
	updateCalled            bool
}

func (f *fakeQueueCredentialClient) CreateSecret(
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

func (f *fakeQueueCredentialClient) DeleteSecret(ctx context.Context, name string, namespace string) (bool, error) {
	f.deleteCalled = true
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, namespace)
	}
	return true, nil
}

func (f *fakeQueueCredentialClient) DeleteSecretIfCurrent(
	ctx context.Context,
	name string,
	namespace string,
	current credhelper.SecretRecord,
) (bool, error) {
	f.deleteCalled = true
	if f.deleteSecretIfCurrentFn != nil {
		return f.deleteSecretIfCurrentFn(ctx, name, namespace, current)
	}
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, namespace)
	}
	return true, nil
}

func (f *fakeQueueCredentialClient) GetSecret(ctx context.Context, name string, namespace string) (map[string][]byte, error) {
	f.getCalled = true
	if f.getSecretFn != nil {
		return f.getSecretFn(ctx, name, namespace)
	}
	return nil, apierrors.NewNotFound(v1.Resource("secret"), name)
}

func (f *fakeQueueCredentialClient) GetSecretRecord(ctx context.Context, name string, namespace string) (credhelper.SecretRecord, error) {
	f.getCalled = true
	if f.getSecretRecordFn != nil {
		return f.getSecretRecordFn(ctx, name, namespace)
	}
	data, err := f.GetSecret(ctx, name, namespace)
	if err != nil {
		return credhelper.SecretRecord{}, err
	}
	return credhelper.SecretRecord{
		UID:    queueSecretRecordUID(f.defaultSecretUID),
		Labels: cloneQueueSecretLabels(f.defaultSecretLabels),
		Data:   cloneQueueSecretData(data),
	}, nil
}

func (f *fakeQueueCredentialClient) UpdateSecret(
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

func (f *fakeQueueCredentialClient) UpdateSecretIfCurrent(
	ctx context.Context,
	name string,
	namespace string,
	current credhelper.SecretRecord,
	labels map[string]string,
	data map[string][]byte,
) (bool, error) {
	f.updateCalled = true
	if f.updateSecretIfCurrentFn != nil {
		return f.updateSecretIfCurrentFn(ctx, name, namespace, current, labels, data)
	}
	if f.updateSecretFn != nil {
		return f.updateSecretFn(ctx, name, namespace, labels, data)
	}
	return true, nil
}

type fakeQueueServiceClient struct {
	createOrUpdateFn func(context.Context, *queuev1beta1.Queue, ctrl.Request) (servicemanager.OSOKResponse, error)
	deleteFn         func(context.Context, *queuev1beta1.Queue) (bool, error)
}

func (f fakeQueueServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *queuev1beta1.Queue,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	if f.createOrUpdateFn != nil {
		return f.createOrUpdateFn(ctx, resource, req)
	}
	return servicemanager.OSOKResponse{}, nil
}

func (f fakeQueueServiceClient) Delete(ctx context.Context, resource *queuev1beta1.Queue) (bool, error) {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, resource)
	}
	return false, nil
}

const (
	testQueueName      = "test-queue"
	testQueueNamespace = "default"
	testQueueUID       = "queue-uid"
	testQueueEndpoint  = "https://cell-1.queue.messaging.us-phoenix-1.oci.oraclecloud.com"
	staleQueueEndpoint = "https://old.queue.messaging.us-phoenix-1.oci.oraclecloud.com"
	testQueueSecretUID = "queue-secret-uid"
	activeQueueOCID    = shared.OCID("ocid1.queue.oc1..active")
)

type queueSecretCallExpectation struct {
	get    bool
	create bool
	update bool
	delete bool
}

func newTestQueueResource() *queuev1beta1.Queue {
	resource := &queuev1beta1.Queue{}
	resource.Name = testQueueName
	resource.Namespace = testQueueNamespace
	resource.UID = testQueueUID
	return resource
}

func activeQueueServiceClient(queueID shared.OCID) fakeQueueServiceClient {
	return fakeQueueServiceClient{
		createOrUpdateFn: func(_ context.Context, resource *queuev1beta1.Queue, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
			resource.Status.OsokStatus.Ocid = queueID
			resource.Status.OsokStatus.Reason = string(shared.Active)
			resource.Status.OsokStatus.Conditions = []shared.OSOKCondition{
				{Type: shared.Active, Status: v1.ConditionTrue},
			}
			resource.Status.MessagesEndpoint = testQueueEndpoint
			return servicemanager.OSOKResponse{IsSuccessful: true}, nil
		},
	}
}

func requireQueueSecretTarget(t *testing.T, action string, name string, namespace string) {
	t.Helper()
	if name != testQueueName || namespace != testQueueNamespace {
		t.Fatalf("%s() target = %s/%s, want %s/%s", action, namespace, name, testQueueNamespace, testQueueName)
	}
}

func requireQueueSecretData(t *testing.T, data map[string][]byte, wantEndpoint string, label string) {
	t.Helper()
	if got := string(data["endpoint"]); got != wantEndpoint {
		t.Fatalf("%s endpoint = %q, want %s", label, got, wantEndpoint)
	}
}

func requireOwnedQueueSecretLabels(t *testing.T, labels map[string]string, wantUID string) {
	t.Helper()
	if got := labels[queueEndpointSecretOwnerUIDLabel]; got != wantUID {
		t.Fatalf("secret owner label = %q, want %q", got, wantUID)
	}
}

func assertQueueCredentialCalls(t *testing.T, credClient *fakeQueueCredentialClient, want queueSecretCallExpectation) {
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

func requireQueueCreateOrUpdateSuccess(t *testing.T, response servicemanager.OSOKResponse, err error, detail string) {
	t.Helper()
	if err != nil {
		t.Fatalf("CreateOrUpdate() error = %v", err)
	}
	if !response.IsSuccessful {
		t.Fatalf("CreateOrUpdate() should keep the successful reconcile result after %s", detail)
	}
}

func requireQueueDeleteSuccess(t *testing.T, deleted bool, err error, detail string) {
	t.Helper()
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Fatalf("Delete() should report success after %s", detail)
	}
}

func newActiveQueueEndpointSecretClient(credClient *fakeQueueCredentialClient) queueEndpointSecretClient {
	return queueEndpointSecretClient{
		delegate:             activeQueueServiceClient(activeQueueOCID),
		credentialClient:     credClient,
		secretRecordReader:   credClient,
		guardedSecretMutator: credClient,
	}
}

func ownedQueueEndpointSecretLabels(uid string) map[string]string {
	return map[string]string{
		queueEndpointSecretOwnerUIDLabel: uid,
	}
}

func queueSecretRecordUID(uid string) types.UID {
	if strings.TrimSpace(uid) == "" {
		return types.UID(testQueueSecretUID)
	}
	return types.UID(uid)
}

func cloneQueueSecretLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(labels))
	for key, value := range labels {
		cloned[key] = value
	}
	return cloned
}

func cloneQueueSecretData(data map[string][]byte) map[string][]byte {
	if len(data) == 0 {
		return nil
	}
	cloned := make(map[string][]byte, len(data))
	for key, value := range data {
		cloned[key] = append([]byte(nil), value...)
	}
	return cloned
}

func TestQueueEndpointSecretClientCreatesSecretAfterActiveReconcile(t *testing.T) {
	t.Parallel()

	credClient := &fakeQueueCredentialClient{}
	var createdLabels map[string]string
	var createdData map[string][]byte
	credClient.createSecretFn = func(_ context.Context, name string, namespace string, labels map[string]string, data map[string][]byte) (bool, error) {
		requireQueueSecretTarget(t, "CreateSecret", name, namespace)
		createdLabels = cloneQueueSecretLabels(labels)
		createdData = cloneQueueSecretData(data)
		return true, nil
	}

	client := newActiveQueueEndpointSecretClient(credClient)
	resource := newTestQueueResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireQueueCreateOrUpdateSuccess(t, response, err, "secret sync")
	assertQueueCredentialCalls(t, credClient, queueSecretCallExpectation{get: true, create: true})
	requireOwnedQueueSecretLabels(t, createdLabels, testQueueUID)
	requireQueueSecretData(t, createdData, testQueueEndpoint, "secret")
}

func TestQueueEndpointSecretClientSkipsSecretUpdateWhenExistingDataMatches(t *testing.T) {
	t.Parallel()

	credClient := &fakeQueueCredentialClient{
		defaultSecretLabels: ownedQueueEndpointSecretLabels(testQueueUID),
	}
	credClient.getSecretFn = func(_ context.Context, name string, namespace string) (map[string][]byte, error) {
		requireQueueSecretTarget(t, "GetSecret", name, namespace)
		return map[string][]byte{"endpoint": []byte(testQueueEndpoint)}, nil
	}
	credClient.createSecretFn = func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
		t.Fatal("CreateSecret() should not be called when the companion secret is already current")
		return false, nil
	}
	credClient.updateSecretFn = func(context.Context, string, string, map[string]string, map[string][]byte) (bool, error) {
		t.Fatal("UpdateSecret() should not be called when the companion secret is already current")
		return false, nil
	}

	client := newActiveQueueEndpointSecretClient(credClient)
	resource := newTestQueueResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireQueueCreateOrUpdateSuccess(t, response, err, "a no-op secret sync")
	assertQueueCredentialCalls(t, credClient, queueSecretCallExpectation{get: true})
}

func TestQueueEndpointSecretClientAdoptsLegacyUnlabeledSecretWhenDataMatches(t *testing.T) {
	t.Parallel()

	credClient := &fakeQueueCredentialClient{}
	var updatedLabels map[string]string
	var updatedData map[string][]byte
	var guardedRecord credhelper.SecretRecord
	credClient.getSecretFn = func(_ context.Context, name string, namespace string) (map[string][]byte, error) {
		requireQueueSecretTarget(t, "GetSecret", name, namespace)
		return map[string][]byte{"endpoint": []byte(testQueueEndpoint)}, nil
	}
	credClient.updateSecretIfCurrentFn = func(_ context.Context, name string, namespace string, current credhelper.SecretRecord, labels map[string]string, data map[string][]byte) (bool, error) {
		requireQueueSecretTarget(t, "UpdateSecret", name, namespace)
		guardedRecord = current
		updatedLabels = cloneQueueSecretLabels(labels)
		updatedData = cloneQueueSecretData(data)
		return true, nil
	}

	client := newActiveQueueEndpointSecretClient(credClient)
	resource := newTestQueueResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireQueueCreateOrUpdateSuccess(t, response, err, "adopting a legacy unlabeled secret")
	assertQueueCredentialCalls(t, credClient, queueSecretCallExpectation{get: true, update: true})
	if guardedRecord.UID != testQueueSecretUID {
		t.Fatalf("guarded update secret UID = %q, want %q", guardedRecord.UID, testQueueSecretUID)
	}
	requireOwnedQueueSecretLabels(t, updatedLabels, testQueueUID)
	requireQueueSecretData(t, updatedData, testQueueEndpoint, "adopted secret")
}

func TestQueueEndpointSecretClientUpdatesExistingSecretWhenEndpointChanges(t *testing.T) {
	t.Parallel()

	credClient := &fakeQueueCredentialClient{
		defaultSecretLabels: ownedQueueEndpointSecretLabels(testQueueUID),
	}
	var updatedData map[string][]byte
	var guardedRecord credhelper.SecretRecord
	credClient.getSecretFn = func(_ context.Context, name string, namespace string) (map[string][]byte, error) {
		requireQueueSecretTarget(t, "GetSecret", name, namespace)
		return map[string][]byte{"endpoint": []byte(staleQueueEndpoint)}, nil
	}
	credClient.updateSecretIfCurrentFn = func(_ context.Context, name string, namespace string, current credhelper.SecretRecord, _ map[string]string, data map[string][]byte) (bool, error) {
		requireQueueSecretTarget(t, "UpdateSecret", name, namespace)
		guardedRecord = current
		updatedData = cloneQueueSecretData(data)
		return true, nil
	}

	client := newActiveQueueEndpointSecretClient(credClient)
	resource := newTestQueueResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	requireQueueCreateOrUpdateSuccess(t, response, err, "secret update")
	assertQueueCredentialCalls(t, credClient, queueSecretCallExpectation{get: true, update: true})
	if guardedRecord.UID != testQueueSecretUID {
		t.Fatalf("guarded update secret UID = %q, want %q", guardedRecord.UID, testQueueSecretUID)
	}
	requireQueueSecretData(t, updatedData, testQueueEndpoint, "updated secret")
}

func TestQueueEndpointSecretClientRejectsUnownedExistingSecret(t *testing.T) {
	t.Parallel()

	credClient := &fakeQueueCredentialClient{}
	credClient.getSecretFn = func(_ context.Context, name string, namespace string) (map[string][]byte, error) {
		requireQueueSecretTarget(t, "GetSecret", name, namespace)
		return map[string][]byte{"endpoint": []byte(staleQueueEndpoint)}, nil
	}

	client := newActiveQueueEndpointSecretClient(credClient)
	resource := newTestQueueResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	if err == nil || !strings.Contains(err.Error(), "not owned") {
		t.Fatalf("CreateOrUpdate() error = %v, want ownership failure", err)
	}
	assert.False(t, response.IsSuccessful)
	assertQueueCredentialCalls(t, credClient, queueSecretCallExpectation{get: true})
}

func TestQueueEndpointSecretClientSkipsSecretUntilActive(t *testing.T) {
	t.Parallel()

	credClient := &fakeQueueCredentialClient{}
	client := queueEndpointSecretClient{
		delegate: fakeQueueServiceClient{
			createOrUpdateFn: func(_ context.Context, resource *queuev1beta1.Queue, _ ctrl.Request) (servicemanager.OSOKResponse, error) {
				resource.Status.OsokStatus.Reason = string(shared.Provisioning)
				resource.Status.OsokStatus.Conditions = []shared.OSOKCondition{
					{Type: shared.Provisioning, Status: v1.ConditionTrue},
				}
				return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true}, nil
			},
		},
		credentialClient:     credClient,
		secretRecordReader:   credClient,
		guardedSecretMutator: credClient,
	}

	resource := newTestQueueResource()

	response, err := client.CreateOrUpdate(context.Background(), resource, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, response.IsSuccessful)
	assert.True(t, response.ShouldRequeue)
	assert.False(t, credClient.createCalled)
}

func TestQueueEndpointSecretClientDeletesSecretAfterDelete(t *testing.T) {
	t.Parallel()

	credClient := &fakeQueueCredentialClient{}
	var guardedRecord credhelper.SecretRecord
	credClient.getSecretRecordFn = func(_ context.Context, name string, namespace string) (credhelper.SecretRecord, error) {
		requireQueueSecretTarget(t, "GetSecret", name, namespace)
		return credhelper.SecretRecord{
			UID:    testQueueSecretUID,
			Labels: ownedQueueEndpointSecretLabels(testQueueUID),
		}, nil
	}
	credClient.deleteSecretIfCurrentFn = func(_ context.Context, name string, namespace string, current credhelper.SecretRecord) (bool, error) {
		requireQueueSecretTarget(t, "DeleteSecret", name, namespace)
		guardedRecord = current
		return true, nil
	}

	client := queueEndpointSecretClient{
		delegate: fakeQueueServiceClient{
			deleteFn: func(_ context.Context, _ *queuev1beta1.Queue) (bool, error) {
				return true, nil
			},
		},
		credentialClient:     credClient,
		secretRecordReader:   credClient,
		guardedSecretMutator: credClient,
	}

	resource := newTestQueueResource()

	deleted, err := client.Delete(context.Background(), resource)
	requireQueueDeleteSuccess(t, deleted, err, "secret cleanup")
	assertQueueCredentialCalls(t, credClient, queueSecretCallExpectation{get: true, delete: true})
	if guardedRecord.UID != testQueueSecretUID {
		t.Fatalf("guarded delete secret UID = %q, want %q", guardedRecord.UID, testQueueSecretUID)
	}
}

func TestQueueEndpointSecretClientDeleteIgnoresMissingSecret(t *testing.T) {
	t.Parallel()

	credClient := &fakeQueueCredentialClient{}
	credClient.getSecretRecordFn = func(_ context.Context, name string, namespace string) (credhelper.SecretRecord, error) {
		requireQueueSecretTarget(t, "GetSecret", name, namespace)
		return credhelper.SecretRecord{}, apierrors.NewNotFound(v1.Resource("secret"), name)
	}

	client := queueEndpointSecretClient{
		delegate: fakeQueueServiceClient{
			deleteFn: func(_ context.Context, _ *queuev1beta1.Queue) (bool, error) {
				return true, nil
			},
		},
		credentialClient:     credClient,
		secretRecordReader:   credClient,
		guardedSecretMutator: credClient,
	}

	resource := newTestQueueResource()

	deleted, err := client.Delete(context.Background(), resource)
	requireQueueDeleteSuccess(t, deleted, err, "a missing companion secret")
	assertQueueCredentialCalls(t, credClient, queueSecretCallExpectation{get: true})
}

func TestQueueEndpointSecretClientDeleteSkipsLegacyUnlabeledSecret(t *testing.T) {
	t.Parallel()

	credClient := &fakeQueueCredentialClient{}
	credClient.getSecretRecordFn = func(_ context.Context, name string, namespace string) (credhelper.SecretRecord, error) {
		requireQueueSecretTarget(t, "GetSecret", name, namespace)
		return credhelper.SecretRecord{}, nil
	}

	client := queueEndpointSecretClient{
		delegate: fakeQueueServiceClient{
			deleteFn: func(_ context.Context, _ *queuev1beta1.Queue) (bool, error) {
				return true, nil
			},
		},
		credentialClient:     credClient,
		secretRecordReader:   credClient,
		guardedSecretMutator: credClient,
	}

	resource := newTestQueueResource()

	deleted, err := client.Delete(context.Background(), resource)
	requireQueueDeleteSuccess(t, deleted, err, "skipping a legacy unlabeled same-name secret")
	assertQueueCredentialCalls(t, credClient, queueSecretCallExpectation{get: true})
}

func TestQueueEndpointSecretClientDeleteSkipsUnownedSecret(t *testing.T) {
	t.Parallel()

	credClient := &fakeQueueCredentialClient{}
	credClient.getSecretRecordFn = func(_ context.Context, name string, namespace string) (credhelper.SecretRecord, error) {
		requireQueueSecretTarget(t, "GetSecret", name, namespace)
		return credhelper.SecretRecord{
			Labels: ownedQueueEndpointSecretLabels("other-queue-uid"),
		}, nil
	}

	client := queueEndpointSecretClient{
		delegate: fakeQueueServiceClient{
			deleteFn: func(_ context.Context, _ *queuev1beta1.Queue) (bool, error) {
				return true, nil
			},
		},
		credentialClient:     credClient,
		secretRecordReader:   credClient,
		guardedSecretMutator: credClient,
	}

	resource := newTestQueueResource()

	deleted, err := client.Delete(context.Background(), resource)
	requireQueueDeleteSuccess(t, deleted, err, "skipping an unowned same-name secret")
	assertQueueCredentialCalls(t, credClient, queueSecretCallExpectation{get: true})
}

func TestQueueManagerInstallsExplicitRuntimePath(t *testing.T) {
	t.Parallel()

	manager := NewQueueServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		&fakeQueueCredentialClient{},
		nil,
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
		nil,
	)

	client, ok := manager.client.(queueEndpointSecretClient)
	if !ok {
		t.Fatalf("manager client type = %T, want queueEndpointSecretClient", manager.client)
	}
	overlayClient, ok := client.delegate.(queueGeneratedRuntimeOverlayClient)
	if !ok {
		t.Fatalf("wrapped delegate type = %T, want queueGeneratedRuntimeOverlayClient", client.delegate)
	}
	if _, ok := overlayClient.delegate.(defaultQueueServiceClient); !ok {
		t.Fatalf("overlay delegate type = %T, want defaultQueueServiceClient", overlayClient.delegate)
	}
}
