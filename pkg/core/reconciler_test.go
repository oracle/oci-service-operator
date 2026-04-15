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
	queuev1beta1 "github.com/oracle/oci-service-operator/api/queue/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/errorutil/errortest"
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

func TestReconcileDeleteErrorLogsRequeueBreadcrumbWhenPresent(t *testing.T) {
	t.Parallel()

	sink := &collectingLogSink{}
	reconciler, recorder, kubeClient := newDeleteReconcilerWithLogger(t, deleteBehavior{
		err:    stderrors.New("delete failed"),
		status: statusWithAsyncCurrent(shared.OSOKAsyncPhaseDelete, "wr-delete-reconcile"),
	}, sink)

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
	assertContainsEvent(t, events, "Failed to delete resource: delete failed (async phase=delete, workRequestId=wr-delete-reconcile)")
	assertAnyMessageContains(t, sink.errors, "Requeuing object due to error during delete of CR (async phase=delete, workRequestId=wr-delete-reconcile)")
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
		err: errortest.NewServiceError(404, "NotFound", "resource not found"),
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
		err: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
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
		err: errortest.NewServiceError(409, errorutil.IncorrectState, "delete conflict"),
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
	assertContainsEvent(t, events, "Delete blocked and will be retried: delete conflict")
	assertNoEventContains(t, events, "Removed finalizer")
	assertNoEventContains(t, events, "Failed Delete the resource")
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
	assertContainsEvent(t, events, "Delete is in progress")
	assertNoEventContains(t, events, "Removed finalizer")
	assertNoEventContains(t, events, "Failed Delete the resource")
}

func TestReconcileDeleteInProgressPersistsDeleteStatusMutations(t *testing.T) {
	t.Parallel()

	const (
		workRequestID = "wr-delete-pending"
		message       = "OCI delete is in progress"
	)

	reconciler, recorder, kubeClient := newDeleteQueueReconciler(t, deleteBehavior{
		mutate: mutateQueueDeleteStatus(t, workRequestID, message),
	})

	result, err := reconciler.Reconcile(context.Background(), testRequest(), &queuev1beta1.Queue{})
	if err != nil {
		t.Fatalf("Reconcile() error = %v, want nil", err)
	}
	if result.RequeueAfter != defaultRequeueTime {
		t.Fatalf("Reconcile() requeueAfter = %v, want %v", result.RequeueAfter, defaultRequeueTime)
	}

	stored := kubeClient.StoredQueue(t)
	if !HasFinalizer(stored, OSOKFinalizerName) {
		t.Fatal("finalizer removed during delete-in-progress, want retained")
	}
	if kubeClient.StatusPatchCount() != 1 {
		t.Fatalf("Status().Patch() count = %d, want 1", kubeClient.StatusPatchCount())
	}
	assertQueueDeleteStatusPersisted(t, stored, workRequestID, message)

	events := drainEvents(recorder)
	assertContainsEvent(t, events, "Delete is in progress")
	assertNoEventContains(t, events, "Removed finalizer")
}

func TestReconcileDeleteErrorPersistsDeleteStatusMutations(t *testing.T) {
	t.Parallel()

	const (
		workRequestID = "wr-delete-error"
		message       = "OCI delete failed after work request submission"
	)

	reconciler, recorder, kubeClient := newDeleteQueueReconciler(t, deleteBehavior{
		err:    stderrors.New("delete failed"),
		mutate: mutateQueueDeleteStatus(t, workRequestID, message),
	})

	result, err := reconciler.Reconcile(context.Background(), testRequest(), &queuev1beta1.Queue{})
	if err != nil {
		t.Fatalf("Reconcile() error = %v, want nil", err)
	}
	if result.RequeueAfter != defaultRequeueTime {
		t.Fatalf("Reconcile() requeueAfter = %v, want %v", result.RequeueAfter, defaultRequeueTime)
	}

	stored := kubeClient.StoredQueue(t)
	if !HasFinalizer(stored, OSOKFinalizerName) {
		t.Fatal("finalizer removed after delete error, want retained")
	}
	if kubeClient.StatusPatchCount() != 1 {
		t.Fatalf("Status().Patch() count = %d, want 1", kubeClient.StatusPatchCount())
	}
	assertQueueDeleteStatusPersisted(t, stored, workRequestID, message)

	events := drainEvents(recorder)
	assertContainsEvent(t, events, "Failed to delete resource: delete failed")
	assertNoEventContains(t, events, "Removed finalizer")
}

func TestReconcileConfirmedDeleteRemovesFinalizerWithoutStatusPatch(t *testing.T) {
	t.Parallel()

	const (
		workRequestID = "wr-delete-confirmed"
		message       = "OCI delete completed; waiting for final confirmation"
	)

	reconciler, recorder, kubeClient := newDeleteQueueReconciler(t, deleteBehavior{
		deleted: true,
		mutate:  mutateQueueDeleteStatus(t, workRequestID, message),
	})

	result, err := reconciler.Reconcile(context.Background(), testRequest(), &queuev1beta1.Queue{})
	if err != nil {
		t.Fatalf("Reconcile() error = %v, want nil", err)
	}
	if result != (ctrl.Result{}) {
		t.Fatalf("Reconcile() result = %#v, want empty result", result)
	}

	stored := kubeClient.StoredQueue(t)
	if HasFinalizer(stored, OSOKFinalizerName) {
		t.Fatal("finalizer still present after confirmed delete")
	}
	if kubeClient.StatusPatchCount() != 0 {
		t.Fatalf("Status().Patch() count = %d, want 0", kubeClient.StatusPatchCount())
	}
	if stored.Status.DeleteWorkRequestId != "" {
		t.Fatalf("status.deleteWorkRequestId = %q, want empty after finalizer-only update", stored.Status.DeleteWorkRequestId)
	}
	if stored.Status.OsokStatus.Async.Current != nil {
		t.Fatalf("status.async.current = %#v, want nil after finalizer-only update", stored.Status.OsokStatus.Async.Current)
	}
	if stored.Status.OsokStatus.Message != "" {
		t.Fatalf("status.message = %q, want empty after finalizer-only update", stored.Status.OsokStatus.Message)
	}

	events := drainEvents(recorder)
	assertContainsEvent(t, events, "Removed finalizer")
}

func TestReconcileDeleteRespectsCommonErrorMatrix(t *testing.T) {
	t.Parallel()

	for _, candidate := range errortest.CommonErrorMatrix {
		candidate := candidate
		t.Run(candidate.Name(), func(t *testing.T) {
			t.Parallel()

			reconciler, recorder, kubeClient := newDeleteReconciler(t, deleteBehavior{
				err: errortest.NewServiceErrorFromCase(candidate),
			})

			result, err := reconciler.Reconcile(context.Background(), testRequest(), &corev1.ConfigMap{})
			if err != nil {
				t.Fatalf("Reconcile() error = %v, want nil", err)
			}

			stored := kubeClient.StoredConfigMap()
			events := drainEvents(recorder)

			switch {
			case candidate.Expectations.Delete == errortest.ExpectationDeleted:
				if result != (ctrl.Result{}) {
					t.Fatalf("Reconcile() result = %#v, want empty result", result)
				}
				if HasFinalizer(stored, OSOKFinalizerName) {
					t.Fatalf("finalizer still present after delete case %s", candidate.Name())
				}
				assertContainsEvent(t, events, "Removed finalizer")
				assertNoEventContains(t, events, "Failed to delete resource")
			case candidate.HTTPStatusCode == 409 && (candidate.ErrorCode == errorutil.IncorrectState || candidate.ErrorCode == "ExternalServerIncorrectState"):
				if result.RequeueAfter != defaultRequeueTime {
					t.Fatalf("Reconcile() requeueAfter = %v, want %v", result.RequeueAfter, defaultRequeueTime)
				}
				if !HasFinalizer(stored, OSOKFinalizerName) {
					t.Fatalf("finalizer removed after retryable delete conflict %s", candidate.Name())
				}
				assertContainsEvent(t, events, "Delete blocked and will be retried")
				assertNoEventContains(t, events, "Removed finalizer")
				assertNoEventContains(t, events, "Failed to delete resource")
			default:
				if result.RequeueAfter != defaultRequeueTime {
					t.Fatalf("Reconcile() requeueAfter = %v, want %v", result.RequeueAfter, defaultRequeueTime)
				}
				if !HasFinalizer(stored, OSOKFinalizerName) {
					t.Fatalf("finalizer removed after delete failure %s", candidate.Name())
				}
				assertContainsEvent(t, events, "Failed to delete resource:")
				assertNoEventContains(t, events, "Removed finalizer")
			}
		})
	}
}

func TestDeleteResourceLogsOCIClassificationOnDeleteFailure(t *testing.T) {
	t.Parallel()

	sink := &collectingLogSink{}
	reconciler, _, kubeClient := newDeleteReconcilerWithLogger(t, deleteBehavior{
		err: errortest.NewServiceError(500, errorutil.InternalServerError, "delete failed"),
	}, sink)

	done, err := reconciler.DeleteResource(context.Background(), kubeClient.StoredConfigMap(), testRequest())
	if err == nil {
		t.Fatal("DeleteResource() error = nil, want delete failure")
	}
	if done {
		t.Fatal("DeleteResource() done = true, want false")
	}

	assertAnyMessageContains(t, sink.errors, "oci_http_status_code: 500")
	assertAnyMessageContains(t, sink.errors, "oci_error_code: InternalServerError")
	assertAnyMessageContains(t, sink.errors, "normalized_error_type: errorutil.InternalServerErrorOciError")
}

func TestDeleteResourceAuthShapedNotFoundRemainsFatal(t *testing.T) {
	t.Parallel()

	sink := &collectingLogSink{}
	reconciler, _, kubeClient := newDeleteReconcilerWithLogger(t, deleteBehavior{
		err: errortest.NewServiceError(404, errorutil.NotAuthorizedOrNotFound, "not authorized or not found"),
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
		err: errortest.NewServiceError(404, errorutil.NotFound, "resource not found"),
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

func TestReconcileResourceSuccessBreadcrumbs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		status         *shared.OSOKStatus
		wantEvent      string
		wantInfo       string
		wantUnexpected string
	}{
		{
			name:           "phase only",
			status:         statusWithAsyncCurrent(shared.OSOKAsyncPhaseCreate, ""),
			wantEvent:      "Create or Update of resource succeeded (async phase=create)",
			wantInfo:       "Reconcile Completed (async phase=create)",
			wantUnexpected: "workRequestId=",
		},
		{
			name:      "no breadcrumb",
			wantEvent: "Create or Update of resource succeeded",
			wantInfo:  "Reconcile Completed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sink := &collectingLogSink{}
			reconciler, recorder, kubeClient := newCreateOrUpdateReconcilerWithLogger(t, createOrUpdateBehavior{
				response: servicemanager.OSOKResponse{IsSuccessful: true},
				status:   tt.status,
			}, sink)

			result, err := reconciler.ReconcileResource(context.Background(), kubeClient.StoredConfigMap(), testRequestNamed("test-reconcile"))
			if err != nil {
				t.Fatalf("ReconcileResource() error = %v, want nil", err)
			}
			if result != (ctrl.Result{}) {
				t.Fatalf("ReconcileResource() result = %#v, want empty result", result)
			}

			events := drainEvents(recorder)
			assertContainsEvent(t, events, tt.wantEvent)
			assertAnyMessageContains(t, sink.infos, tt.wantInfo)
			if tt.wantUnexpected != "" {
				assertNoEventContains(t, events, tt.wantUnexpected)
				assertNoMessageContains(t, sink.infos, tt.wantUnexpected)
				assertNoMessageContains(t, sink.errors, tt.wantUnexpected)
			}
		})
	}
}

func TestReconcileResourceFailureIncludesAsyncBreadcrumbWhenPresent(t *testing.T) {
	t.Parallel()

	sink := &collectingLogSink{}
	reconciler, recorder, kubeClient := newCreateOrUpdateReconcilerWithLogger(t, createOrUpdateBehavior{
		err:    stderrors.New("apply failed"),
		status: statusWithAsyncCurrent(shared.OSOKAsyncPhaseUpdate, "wr-update-1"),
	}, sink)

	result, err := reconciler.ReconcileResource(context.Background(), kubeClient.StoredConfigMap(), testRequestNamed("test-reconcile"))
	if err != nil {
		t.Fatalf("ReconcileResource() error = %v, want nil", err)
	}
	if result != (ctrl.Result{}) {
		t.Fatalf("ReconcileResource() result = %#v, want empty result", result)
	}

	events := drainEvents(recorder)
	assertContainsEvent(t, events, "Failed to create or update resource: apply failed (async phase=update, workRequestId=wr-update-1)")
	assertContainsEvent(t, events, "Failed to create or update resource (async phase=update, workRequestId=wr-update-1)")
	assertAnyMessageContains(t, sink.errors, "Create Or Update failed in the Service Manager with error (async phase=update, workRequestId=wr-update-1)")
	assertAnyMessageContains(t, sink.infos, "Reconcile Failed (async phase=update, workRequestId=wr-update-1)")
}

func TestReconcileResourceStatusPatchFailureIncludesAsyncBreadcrumbWhenPresent(t *testing.T) {
	t.Parallel()

	sink := &collectingLogSink{}
	reconciler, recorder, kubeClient := newCreateOrUpdateReconcilerWithLogger(t, createOrUpdateBehavior{
		response: servicemanager.OSOKResponse{IsSuccessful: true},
		status:   statusWithAsyncCurrent(shared.OSOKAsyncPhaseUpdate, "wr-update-patch"),
	}, sink)
	kubeClient.statusPatchErr = stderrors.New("status patch failed")

	result, err := reconciler.ReconcileResource(context.Background(), kubeClient.StoredConfigMap(), testRequestNamed("test-reconcile"))
	if err != nil {
		t.Fatalf("ReconcileResource() error = %v, want nil", err)
	}
	if result.RequeueAfter != defaultRequeueTime {
		t.Fatalf("ReconcileResource() requeueAfter = %v, want %v", result.RequeueAfter, defaultRequeueTime)
	}

	events := drainEvents(recorder)
	assertContainsEvent(t, events, "Failed to create or update resource: status patch failed (async phase=update, workRequestId=wr-update-patch)")
	assertAnyMessageContains(t, sink.errors, "Error updating the status of the Object (async phase=update, workRequestId=wr-update-patch)")
	assertNoEventContains(t, events, "Create or Update of resource succeeded")
	assertNoMessageContains(t, sink.infos, "Reconcile Completed")
}

func TestDeleteResourceBreadcrumbs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		behavior       deleteBehavior
		wantDone       bool
		wantErr        bool
		wantEvent      string
		wantInfo       string
		wantError      string
		wantUnexpected string
	}{
		{
			name: "blocked",
			behavior: deleteBehavior{
				err:    errortest.NewServiceError(409, errorutil.IncorrectState, "delete conflict"),
				status: statusWithAsyncCurrent(shared.OSOKAsyncPhaseDelete, "wr-delete-1"),
			},
			wantDone:  false,
			wantErr:   false,
			wantEvent: "Delete blocked and will be retried: delete conflict (async phase=delete, workRequestId=wr-delete-1)",
			wantInfo:  "Delete is blocked and will be retried (async phase=delete, workRequestId=wr-delete-1)",
		},
		{
			name: "in progress",
			behavior: deleteBehavior{
				status: statusWithAsyncCurrent(shared.OSOKAsyncPhaseDelete, "wr-delete-2"),
			},
			wantDone:  false,
			wantErr:   false,
			wantEvent: "Delete is in progress (async phase=delete, workRequestId=wr-delete-2)",
			wantInfo:  "Delete is in progress and will be retried (async phase=delete, workRequestId=wr-delete-2)",
		},
		{
			name: "failure with breadcrumb",
			behavior: deleteBehavior{
				err:    stderrors.New("delete failed"),
				status: statusWithAsyncCurrent(shared.OSOKAsyncPhaseDelete, "wr-delete-3"),
			},
			wantDone:  false,
			wantErr:   true,
			wantEvent: "Failed to delete resource: delete failed (async phase=delete, workRequestId=wr-delete-3)",
			wantError: "Delete failed in the Service Manager with error (async phase=delete, workRequestId=wr-delete-3)",
		},
		{
			name: "failure without breadcrumb",
			behavior: deleteBehavior{
				err: stderrors.New("delete failed"),
			},
			wantDone:       false,
			wantErr:        true,
			wantEvent:      "Failed to delete resource: delete failed",
			wantError:      "Delete failed in the Service Manager with error",
			wantUnexpected: "async phase=",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sink := &collectingLogSink{}
			reconciler, recorder, kubeClient := newDeleteReconcilerWithLogger(t, tt.behavior, sink)

			done, err := reconciler.DeleteResource(context.Background(), kubeClient.StoredConfigMap(), testRequest())
			if done != tt.wantDone {
				t.Fatalf("DeleteResource() done = %t, want %t", done, tt.wantDone)
			}
			if (err != nil) != tt.wantErr {
				t.Fatalf("DeleteResource() error = %v, wantErr %t", err, tt.wantErr)
			}

			events := drainEvents(recorder)
			assertContainsEvent(t, events, tt.wantEvent)
			if tt.wantInfo != "" {
				assertAnyMessageContains(t, sink.infos, tt.wantInfo)
			}
			if tt.wantError != "" {
				assertAnyMessageContains(t, sink.errors, tt.wantError)
			}
			if tt.wantUnexpected != "" {
				assertNoEventContains(t, events, tt.wantUnexpected)
				assertNoMessageContains(t, sink.infos, tt.wantUnexpected)
				assertNoMessageContains(t, sink.errors, tt.wantUnexpected)
			}
		})
	}
}

func newDeleteReconciler(t *testing.T, behavior deleteBehavior) (*BaseReconciler, *record.FakeRecorder, *memoryClient) {
	return newDeleteReconcilerWithLogger(t, behavior, &collectingLogSink{})
}

func newDeleteReconcilerWithLogger(t *testing.T, behavior deleteBehavior, sink *collectingLogSink) (*BaseReconciler, *record.FakeRecorder, *memoryClient) {
	t.Helper()

	return newTestReconcilerWithLogger(t, &stubServiceManager{deleteBehavior: behavior}, deletingTestConfigMap("test-delete"), sink)
}

func newCreateOrUpdateReconcilerWithLogger(t *testing.T, behavior createOrUpdateBehavior, sink *collectingLogSink) (*BaseReconciler, *record.FakeRecorder, *memoryClient) {
	t.Helper()

	return newTestReconcilerWithLogger(t, &stubServiceManager{createOrUpdateBehavior: behavior}, testConfigMap("test-reconcile"), sink)
}

func newTestReconcilerWithLogger(t *testing.T, serviceManager servicemanager.OSOKServiceManager, obj ctrlclient.Object, sink *collectingLogSink) (*BaseReconciler, *record.FakeRecorder, *memoryClient) {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	kubeClient := newMemoryClient(scheme, obj)

	recorder := record.NewFakeRecorder(20)
	if sink == nil {
		sink = &collectingLogSink{}
	}
	log := loggerutil.OSOKLogger{Logger: logr.New(sink)}

	return &BaseReconciler{
		Client:             kubeClient,
		OSOKServiceManager: serviceManager,
		Log:                log,
		Metrics:            &metrics.Metrics{Name: "oci", ServiceName: "core", Logger: log},
		Recorder:           recorder,
		Scheme:             scheme,
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

func assertNoMessageContains(t *testing.T, messages []string, unexpected string) {
	t.Helper()
	for _, message := range messages {
		if strings.Contains(message, unexpected) {
			t.Fatalf("messages %v unexpectedly contain %q", messages, unexpected)
		}
	}
}

func testRequest() ctrl.Request {
	return testRequestNamed("test-delete")
}

func testRequestNamed(name string) ctrl.Request {
	return ctrl.Request{NamespacedName: ctrlclient.ObjectKey{Name: name, Namespace: "default"}}
}

func testConfigMap(name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
	}
}

func deletingTestConfigMap(name string) *corev1.ConfigMap {
	configMap := testConfigMap(name)
	now := metav1.NewTime(time.Now())
	configMap.ObjectMeta.Finalizers = []string{OSOKFinalizerName}
	configMap.ObjectMeta.DeletionTimestamp = &now
	return configMap
}

func statusWithAsyncCurrent(phase shared.OSOKAsyncPhase, workRequestID string) *shared.OSOKStatus {
	return &shared.OSOKStatus{
		Async: shared.OSOKAsyncTracker{
			Current: &shared.OSOKAsyncOperation{
				Phase:         phase,
				WorkRequestID: workRequestID,
			},
		},
	}
}

type createOrUpdateBehavior struct {
	response servicemanager.OSOKResponse
	err      error
	status   *shared.OSOKStatus
}

type deleteBehavior struct {
	deleted bool
	err     error
	status  *shared.OSOKStatus
	mutate  func(runtime.Object)
}

type stubServiceManager struct {
	createOrUpdateBehavior createOrUpdateBehavior
	deleteBehavior         deleteBehavior
	currentStatus          *shared.OSOKStatus
}

func (m *stubServiceManager) CreateOrUpdate(context.Context, runtime.Object, ctrl.Request) (servicemanager.OSOKResponse, error) {
	m.currentStatus = cloneStatus(m.createOrUpdateBehavior.status)
	return m.createOrUpdateBehavior.response, m.createOrUpdateBehavior.err
}

func (m *stubServiceManager) Delete(_ context.Context, obj runtime.Object) (bool, error) {
	if m.deleteBehavior.mutate != nil {
		m.deleteBehavior.mutate(obj)
	}
	m.currentStatus = cloneStatus(m.deleteBehavior.status)
	return m.deleteBehavior.deleted, m.deleteBehavior.err
}

func (m *stubServiceManager) GetCrdStatus(runtime.Object) (*shared.OSOKStatus, error) {
	if m.currentStatus == nil {
		return &shared.OSOKStatus{}, nil
	}
	return m.currentStatus, nil
}

func cloneStatus(status *shared.OSOKStatus) *shared.OSOKStatus {
	if status == nil {
		return nil
	}
	return status.DeepCopy()
}

type memoryClient struct {
	ctrlclient.Client
	stored         ctrlclient.Object
	statusPatchErr error
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

func (c *memoryClient) Status() ctrlclient.SubResourceWriter {
	return memoryStatusWriter{client: c}
}

func (c *memoryClient) StoredConfigMap() *corev1.ConfigMap {
	return c.stored.DeepCopyObject().(*corev1.ConfigMap)
}

type statusTrackingClient struct {
	ctrlclient.Client
	stored           ctrlclient.Object
	statusPatchCount int
}

func newDeleteQueueReconciler(t *testing.T, behavior deleteBehavior) (*BaseReconciler, *record.FakeRecorder, *statusTrackingClient) {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := queuev1beta1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	now := metav1.NewTime(time.Now())
	queue := &queuev1beta1.Queue{
		TypeMeta: metav1.TypeMeta{
			APIVersion: queuev1beta1.GroupVersion.String(),
			Kind:       "Queue",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-delete",
			Namespace:         "default",
			Finalizers:        []string{OSOKFinalizerName},
			DeletionTimestamp: &now,
		},
	}

	trackingClient := &statusTrackingClient{
		Client: ctrlclientfake.NewClientBuilder().WithScheme(scheme).Build(),
		stored: queue.DeepCopyObject().(ctrlclient.Object),
	}

	recorder := record.NewFakeRecorder(10)
	log := loggerutil.OSOKLogger{Logger: logr.New(&collectingLogSink{})}

	return &BaseReconciler{
		Client:             trackingClient,
		OSOKServiceManager: &stubServiceManager{deleteBehavior: behavior},
		Log:                log,
		Metrics:            &metrics.Metrics{Name: "oci", ServiceName: "queue", Logger: log},
		Recorder:           recorder,
		Scheme:             scheme,
	}, recorder, trackingClient
}

func (c *statusTrackingClient) Status() ctrlclient.SubResourceWriter {
	return statusTrackingSubresourceWriter{client: c}
}

func (c *statusTrackingClient) StatusPatchCount() int {
	return c.statusPatchCount
}

func (c *statusTrackingClient) StoredQueue(t *testing.T) *queuev1beta1.Queue {
	t.Helper()

	return c.stored.DeepCopyObject().(*queuev1beta1.Queue)
}

type statusTrackingSubresourceWriter struct {
	client *statusTrackingClient
}

func (c *statusTrackingClient) Get(_ context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, _ ...ctrlclient.GetOption) error {
	if c.stored == nil || c.stored.GetName() != key.Name || c.stored.GetNamespace() != key.Namespace {
		return apierrors.NewNotFound(schema.GroupResource{Resource: "queues"}, key.Name)
	}

	value := reflect.ValueOf(obj)
	source := reflect.ValueOf(c.stored.DeepCopyObject())
	if value.Kind() != reflect.Ptr || source.Kind() != reflect.Ptr {
		return stderrors.New("status tracking client requires pointer objects")
	}
	value.Elem().Set(source.Elem())
	return nil
}

func (c *statusTrackingClient) Update(_ context.Context, obj ctrlclient.Object, _ ...ctrlclient.UpdateOption) error {
	if storedQueue, ok := c.stored.(*queuev1beta1.Queue); ok {
		if updatedQueue, ok := obj.(*queuev1beta1.Queue); ok {
			copy := updatedQueue.DeepCopy()
			copy.Status = storedQueue.Status
			c.stored = copy
			return nil
		}
	}
	c.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
}

func (w statusTrackingSubresourceWriter) Create(_ context.Context, obj ctrlclient.Object, _ ctrlclient.Object, _ ...ctrlclient.SubResourceCreateOption) error {
	w.client.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
}

func (w statusTrackingSubresourceWriter) Update(_ context.Context, obj ctrlclient.Object, _ ...ctrlclient.SubResourceUpdateOption) error {
	w.client.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
}

func (w statusTrackingSubresourceWriter) Patch(_ context.Context, obj ctrlclient.Object, _ ctrlclient.Patch, _ ...ctrlclient.SubResourcePatchOption) error {
	w.client.statusPatchCount++
	w.client.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
}

func mutateQueueDeleteStatus(t *testing.T, workRequestID, message string) func(runtime.Object) {
	t.Helper()

	return func(obj runtime.Object) {
		queue, ok := obj.(*queuev1beta1.Queue)
		if !ok {
			t.Fatalf("Delete() object type = %T, want *queuev1beta1.Queue", obj)
		}

		queue.Status.OsokStatus.Message = message
		queue.Status.DeleteWorkRequestId = workRequestID
		queue.Status.OsokStatus.Async.Current = &shared.OSOKAsyncOperation{
			Source:           shared.OSOKAsyncSourceWorkRequest,
			Phase:            shared.OSOKAsyncPhaseDelete,
			WorkRequestID:    workRequestID,
			RawStatus:        "IN_PROGRESS",
			RawOperationType: "DELETE_QUEUE",
			NormalizedClass:  shared.OSOKAsyncClassPending,
			Message:          message,
		}
	}
}

func assertQueueDeleteStatusPersisted(t *testing.T, stored *queuev1beta1.Queue, workRequestID, message string) {
	t.Helper()

	if stored.Status.DeleteWorkRequestId != workRequestID {
		t.Fatalf("status.deleteWorkRequestId = %q, want %q", stored.Status.DeleteWorkRequestId, workRequestID)
	}
	if stored.Status.OsokStatus.Message != message {
		t.Fatalf("status.message = %q, want %q", stored.Status.OsokStatus.Message, message)
	}
	if stored.Status.OsokStatus.Async.Current == nil {
		t.Fatal("status.async.current = nil, want populated delete breadcrumb")
	}
	if stored.Status.OsokStatus.Async.Current.Source != shared.OSOKAsyncSourceWorkRequest {
		t.Fatalf("status.async.current.source = %q, want %q", stored.Status.OsokStatus.Async.Current.Source, shared.OSOKAsyncSourceWorkRequest)
	}
	if stored.Status.OsokStatus.Async.Current.Phase != shared.OSOKAsyncPhaseDelete {
		t.Fatalf("status.async.current.phase = %q, want %q", stored.Status.OsokStatus.Async.Current.Phase, shared.OSOKAsyncPhaseDelete)
	}
	if stored.Status.OsokStatus.Async.Current.WorkRequestID != workRequestID {
		t.Fatalf("status.async.current.workRequestId = %q, want %q", stored.Status.OsokStatus.Async.Current.WorkRequestID, workRequestID)
	}
	if stored.Status.OsokStatus.Async.Current.RawStatus != "IN_PROGRESS" {
		t.Fatalf("status.async.current.rawStatus = %q, want %q", stored.Status.OsokStatus.Async.Current.RawStatus, "IN_PROGRESS")
	}
	if stored.Status.OsokStatus.Async.Current.RawOperationType != "DELETE_QUEUE" {
		t.Fatalf("status.async.current.rawOperationType = %q, want %q", stored.Status.OsokStatus.Async.Current.RawOperationType, "DELETE_QUEUE")
	}
	if stored.Status.OsokStatus.Async.Current.NormalizedClass != shared.OSOKAsyncClassPending {
		t.Fatalf("status.async.current.normalizedClass = %q, want %q", stored.Status.OsokStatus.Async.Current.NormalizedClass, shared.OSOKAsyncClassPending)
	}
	if stored.Status.OsokStatus.Async.Current.Message != message {
		t.Fatalf("status.async.current.message = %q, want %q", stored.Status.OsokStatus.Async.Current.Message, message)
	}
}

type memoryStatusWriter struct {
	client *memoryClient
}

func (w memoryStatusWriter) Create(_ context.Context, obj ctrlclient.Object, _ ctrlclient.Object, _ ...ctrlclient.SubResourceCreateOption) error {
	w.client.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
}

func (w memoryStatusWriter) Update(_ context.Context, obj ctrlclient.Object, _ ...ctrlclient.SubResourceUpdateOption) error {
	w.client.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
}

func (w memoryStatusWriter) Patch(_ context.Context, obj ctrlclient.Object, _ ctrlclient.Patch, _ ...ctrlclient.SubResourcePatchOption) error {
	if w.client.statusPatchErr != nil {
		return w.client.statusPatchErr
	}
	w.client.stored = obj.DeepCopyObject().(ctrlclient.Object)
	return nil
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
