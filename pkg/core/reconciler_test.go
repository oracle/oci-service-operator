/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package core

import (
	"context"
	stderrors "errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileDeleteErrorDoesNotEmitFinalizerRemovalFailure(t *testing.T) {
	t.Parallel()

	reconciler, recorder, kubeClient := newDeleteReconciler(t, deleteBehavior{
		err: stderrors.New("delete failed"),
	})

	result, err := reconciler.Reconcile(context.Background(), testRequest(), &corev1.ConfigMap{})
	if err != nil {
		t.Fatalf("Reconcile() error = %v, want nil", err)
	}
	if result.RequeueAfter != defaultRequeueTime {
		t.Fatalf("Reconcile() requeueAfter = %v, want %v", result.RequeueAfter, defaultRequeueTime)
	}

	stored := kubeClient.StoredConfigMap()
	if !HasFinalizer(stored, OSOKFinalizerName) {
		t.Fatal("finalizer removed after delete error, want retained")
	}

	events := drainEvents(recorder)
	assertContainsEvent(t, events, "Failed to delete resource: delete failed")
	assertNoEventContains(t, events, "Failed to remove the finalizer")
}

func TestReconcileConfirmedDeleteRemovesFinalizer(t *testing.T) {
	t.Parallel()

	reconciler, recorder, kubeClient := newDeleteReconciler(t, deleteBehavior{
		deleted: true,
	})

	result, err := reconciler.Reconcile(context.Background(), testRequest(), &corev1.ConfigMap{})
	if err != nil {
		t.Fatalf("Reconcile() error = %v, want nil", err)
	}
	if result != (ctrl.Result{}) {
		t.Fatalf("Reconcile() result = %#v, want empty result", result)
	}

	stored := kubeClient.StoredConfigMap()
	if HasFinalizer(stored, OSOKFinalizerName) {
		t.Fatal("finalizer still present after confirmed delete")
	}

	events := drainEvents(recorder)
	assertContainsEvent(t, events, "Removed finalizer")
}

func TestReconcileNotFoundDeleteRemovesFinalizer(t *testing.T) {
	t.Parallel()

	reconciler, recorder, kubeClient := newDeleteReconciler(t, deleteBehavior{
		err: testServiceError{
			statusCode: 404,
			code:       "NotFound",
			message:    "resource not found",
		},
	})

	result, err := reconciler.Reconcile(context.Background(), testRequest(), &corev1.ConfigMap{})
	if err != nil {
		t.Fatalf("Reconcile() error = %v, want nil", err)
	}
	if result != (ctrl.Result{}) {
		t.Fatalf("Reconcile() result = %#v, want empty result", result)
	}

	stored := kubeClient.StoredConfigMap()
	if HasFinalizer(stored, OSOKFinalizerName) {
		t.Fatal("finalizer still present after OCI not found delete")
	}

	events := drainEvents(recorder)
	assertContainsEvent(t, events, "Removed finalizer")
	assertNoEventContains(t, events, "Failed to delete resource")
}

func TestReconcileAuthShapedNotFoundDeleteRequeuesAndKeepsFinalizer(t *testing.T) {
	t.Parallel()

	reconciler, recorder, kubeClient := newDeleteReconciler(t, deleteBehavior{
		err: testServiceError{
			statusCode: 404,
			code:       errorutil.NotAuthorizedOrNotFound,
			message:    "not authorized or not found",
		},
	})

	result, err := reconciler.Reconcile(context.Background(), testRequest(), &corev1.ConfigMap{})
	if err != nil {
		t.Fatalf("Reconcile() error = %v, want nil", err)
	}
	if result.RequeueAfter != defaultRequeueTime {
		t.Fatalf("Reconcile() requeueAfter = %v, want %v", result.RequeueAfter, defaultRequeueTime)
	}

	stored := kubeClient.StoredConfigMap()
	if !HasFinalizer(stored, OSOKFinalizerName) {
		t.Fatal("finalizer removed after auth-shaped 404, want retained")
	}

	events := drainEvents(recorder)
	assertContainsEvent(t, events, "Failed to delete resource: not authorized or not found")
	assertNoEventContains(t, events, "Removed finalizer")
}

func TestReconcileConflictDeleteRequeuesAndKeepsFinalizer(t *testing.T) {
	t.Parallel()

	reconciler, recorder, kubeClient := newDeleteReconciler(t, deleteBehavior{
		err: testServiceError{
			statusCode: 409,
			code:       errorutil.IncorrectState,
			message:    "delete conflict",
		},
	})

	result, err := reconciler.Reconcile(context.Background(), testRequest(), &corev1.ConfigMap{})
	if err != nil {
		t.Fatalf("Reconcile() error = %v, want nil", err)
	}
	if result.RequeueAfter != defaultRequeueTime {
		t.Fatalf("Reconcile() requeueAfter = %v, want %v", result.RequeueAfter, defaultRequeueTime)
	}

	stored := kubeClient.StoredConfigMap()
	if !HasFinalizer(stored, OSOKFinalizerName) {
		t.Fatal("finalizer removed after conflict, want retained")
	}

	events := drainEvents(recorder)
	assertContainsEvent(t, events, "Failed to delete resource: delete conflict")
	assertNoEventContains(t, events, "Removed finalizer")
}

func TestReconcileDeleteInProgressRequeuesAndKeepsFinalizer(t *testing.T) {
	t.Parallel()

	reconciler, recorder, kubeClient := newDeleteReconciler(t, deleteBehavior{})

	result, err := reconciler.Reconcile(context.Background(), testRequest(), &corev1.ConfigMap{})
	if err != nil {
		t.Fatalf("Reconcile() error = %v, want nil", err)
	}
	if result.RequeueAfter != defaultRequeueTime {
		t.Fatalf("Reconcile() requeueAfter = %v, want %v", result.RequeueAfter, defaultRequeueTime)
	}

	stored := kubeClient.StoredConfigMap()
	if !HasFinalizer(stored, OSOKFinalizerName) {
		t.Fatal("finalizer removed during delete-in-progress, want retained")
	}

	events := drainEvents(recorder)
	assertContainsEvent(t, events, "Delete Unsuccessful")
	assertNoEventContains(t, events, "Removed finalizer")
}

func TestDeleteResourceLogsOCIClassificationOnDeleteFailure(t *testing.T) {
	t.Parallel()

	sink := &collectingLogSink{}
	reconciler, _, kubeClient := newDeleteReconcilerWithLogger(t, deleteBehavior{
		err: testServiceError{
			statusCode: 409,
			code:       errorutil.IncorrectState,
			message:    "delete conflict",
		},
	}, sink)

	done, err := reconciler.DeleteResource(context.Background(), kubeClient.StoredConfigMap(), testRequest())
	if err == nil {
		t.Fatal("DeleteResource() error = nil, want delete failure")
	}
	if done {
		t.Fatal("DeleteResource() done = true, want false")
	}

	assertAnyMessageContains(t, sink.errors, "oci_http_status_code: 409")
	assertAnyMessageContains(t, sink.errors, "oci_error_code: IncorrectState")
	assertAnyMessageContains(t, sink.errors, "normalized_error_type: errorutil.ConflictOciError")
}

func TestDeleteResourceAuthShapedNotFoundRemainsFatal(t *testing.T) {
	t.Parallel()

	sink := &collectingLogSink{}
	reconciler, _, kubeClient := newDeleteReconcilerWithLogger(t, deleteBehavior{
		err: testServiceError{
			statusCode: 404,
			code:       errorutil.NotAuthorizedOrNotFound,
			message:    "not authorized or not found",
		},
	}, sink)

	done, err := reconciler.DeleteResource(context.Background(), kubeClient.StoredConfigMap(), testRequest())
	if err == nil {
		t.Fatal("DeleteResource() error = nil, want auth-shaped 404 failure")
	}
	if done {
		t.Fatal("DeleteResource() done = true, want false")
	}

	assertAnyMessageContains(t, sink.errors, "oci_http_status_code: 404")
	assertAnyMessageContains(t, sink.errors, "oci_error_code: NotAuthorizedOrNotFound")
	assertAnyMessageContains(t, sink.errors, "normalized_error_type: errorutil.UnauthorizedAndNotFoundOciError")
}

func TestDeleteResourceLogsOCIClassificationOnDeleteNotFoundSuccess(t *testing.T) {
	t.Parallel()

	sink := &collectingLogSink{}
	reconciler, _, kubeClient := newDeleteReconcilerWithLogger(t, deleteBehavior{
		err: testServiceError{
			statusCode: 404,
			code:       errorutil.NotFound,
			message:    "resource not found",
		},
	}, sink)

	done, err := reconciler.DeleteResource(context.Background(), kubeClient.StoredConfigMap(), testRequest())
	if err != nil {
		t.Fatalf("DeleteResource() error = %v, want nil", err)
	}
	if !done {
		t.Fatal("DeleteResource() done = false, want true")
	}

	assertAnyMessageContains(t, sink.infos, "oci_http_status_code: 404")
	assertAnyMessageContains(t, sink.infos, "oci_error_code: NotFound")
	assertAnyMessageContains(t, sink.infos, "normalized_error_type: errorutil.NotFoundOciError")
}

func newDeleteReconciler(t *testing.T, behavior deleteBehavior) (*BaseReconciler, *record.FakeRecorder, *memoryClient) {
	return newDeleteReconcilerWithLogger(t, behavior, &collectingLogSink{})
}

func newDeleteReconcilerWithLogger(t *testing.T, behavior deleteBehavior, sink *collectingLogSink) (*BaseReconciler, *record.FakeRecorder, *memoryClient) {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	now := metav1.NewTime(time.Now())
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-delete",
			Namespace:         "default",
			Finalizers:        []string{OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
	}

	kubeClient := newMemoryClient(scheme, configMap)

	recorder := record.NewFakeRecorder(10)
	if sink == nil {
		sink = &collectingLogSink{}
	}
	log := loggerutil.OSOKLogger{Logger: logr.New(sink)}

	return &BaseReconciler{
		Client: kubeClient,
		OSOKServiceManager: deleteOnlyServiceManager{
			deleteBehavior: behavior,
		},
		Log:      log,
		Metrics:  &metrics.Metrics{Name: "oci", ServiceName: "core", Logger: log},
		Recorder: recorder,
		Scheme:   scheme,
	}, recorder, kubeClient
}

func drainEvents(recorder *record.FakeRecorder) []string {
	events := make([]string, 0, len(recorder.Events))
	for {
		select {
		case event := <-recorder.Events:
			events = append(events, event)
		default:
			return events
		}
	}
}

func assertContainsEvent(t *testing.T, events []string, want string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, want) {
			return
		}
	}
	t.Fatalf("events %v do not contain %q", events, want)
}

func assertNoEventContains(t *testing.T, events []string, unexpected string) {
	t.Helper()
	for _, event := range events {
		if strings.Contains(event, unexpected) {
			t.Fatalf("events %v unexpectedly contain %q", events, unexpected)
		}
	}
}

func assertAnyMessageContains(t *testing.T, messages []string, want string) {
	t.Helper()
	for _, message := range messages {
		if strings.Contains(message, want) {
			return
		}
	}
	t.Fatalf("messages %v do not contain %q", messages, want)
}

func testRequest() ctrl.Request {
	return ctrl.Request{NamespacedName: ctrlclient.ObjectKey{Name: "test-delete", Namespace: "default"}}
}

type deleteBehavior struct {
	deleted bool
	err     error
}

type deleteOnlyServiceManager struct {
	deleteBehavior deleteBehavior
}

func (m deleteOnlyServiceManager) CreateOrUpdate(context.Context, runtime.Object, ctrl.Request) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{}, nil
}

func (m deleteOnlyServiceManager) Delete(context.Context, runtime.Object) (bool, error) {
	return m.deleteBehavior.deleted, m.deleteBehavior.err
}

func (m deleteOnlyServiceManager) GetCrdStatus(runtime.Object) (*shared.OSOKStatus, error) {
	return &shared.OSOKStatus{}, nil
}

type testServiceError struct {
	statusCode int
	code       string
	message    string
}

func (e testServiceError) Error() string {
	return e.message
}

func (e testServiceError) GetHTTPStatusCode() int {
	return e.statusCode
}

func (e testServiceError) GetMessage() string {
	return e.message
}

func (e testServiceError) GetCode() string {
	return e.code
}

func (e testServiceError) GetOpcRequestID() string {
	return "opc-request-id"
}

func (e testServiceError) GetTargetService() string {
	return "core"
}

func (e testServiceError) GetOperationName() string {
	return "DeleteTestResource"
}

func (e testServiceError) GetTimestamp() common.SDKTime {
	return common.SDKTime{}
}

func (e testServiceError) GetClientVersion() string {
	return "test"
}

func (e testServiceError) GetRequestTarget() string {
	return "DELETE /resource"
}

func (e testServiceError) GetOperationReferenceLink() string {
	return ""
}

func (e testServiceError) GetErrorTroubleshootingLink() string {
	return ""
}

var _ common.ServiceError = testServiceError{}

type memoryClient struct {
	ctrlclient.Client
	stored ctrlclient.Object
}

func newMemoryClient(scheme *runtime.Scheme, obj ctrlclient.Object) *memoryClient {
	return &memoryClient{
		Client: ctrlclientfake.NewClientBuilder().WithScheme(scheme).Build(),
		stored: obj.DeepCopyObject().(ctrlclient.Object),
	}
}

func (c *memoryClient) Get(_ context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, _ ...ctrlclient.GetOption) error {
	if c.stored == nil || c.stored.GetName() != key.Name || c.stored.GetNamespace() != key.Namespace {
		return apierrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, key.Name)
	}

	value := reflect.ValueOf(obj)
	source := reflect.ValueOf(c.stored.DeepCopyObject())
	if value.Kind() != reflect.Ptr || source.Kind() != reflect.Ptr {
		return stderrors.New("memory client requires pointer objects")
	}
	value.Elem().Set(source.Elem())
	return nil
}

func (c *memoryClient) Update(_ context.Context, obj ctrlclient.Object, _ ...ctrlclient.UpdateOption) error {
	c.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
}

func (c *memoryClient) StoredConfigMap() *corev1.ConfigMap {
	return c.stored.DeepCopyObject().(*corev1.ConfigMap)
}

type collectingLogSink struct {
	infos  []string
	errors []string
}

func (s *collectingLogSink) Init(logr.RuntimeInfo) {}

func (s *collectingLogSink) Enabled(int) bool {
	return true
}

func (s *collectingLogSink) Info(_ int, msg string, _ ...interface{}) {
	s.infos = append(s.infos, msg)
}

func (s *collectingLogSink) Error(_ error, msg string, _ ...interface{}) {
	s.errors = append(s.errors, msg)
}

func (s *collectingLogSink) WithValues(...interface{}) logr.LogSink {
	return s
}

func (s *collectingLogSink) WithName(string) logr.LogSink {
	return s
}
