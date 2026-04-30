/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
)

const (
	OSOKFinalizerName  = "finalizers.oci.oracle.com/oci-resources"
	defaultRequeueTime = time.Minute * 2
)

const (
	deleteEventReasonBlocked    = "DeleteBlocked"
	deleteEventReasonInProgress = "DeleteInProgress"
)

type BaseReconciler struct {
	client.Client
	OSOKServiceManager   servicemanager.OSOKServiceManager
	Finalizer            Finalizer
	Log                  loggerutil.OSOKLogger
	Metrics              *metrics.Metrics
	Recorder             record.EventRecorder
	Scheme               *runtime.Scheme
	AdditionalFinalizers []string
}

func (r *BaseReconciler) Reconcile(ctx context.Context, req ctrl.Request, obj client.Object) (result ctrl.Result, err error) {
	// To setup the fixed logs for every log
	ctx = metrics.AddFixedLogMapEntries(ctx, req.Name, req.Namespace)
	r.Log.DebugLogWithFixedMessage(ctx, "Fetching the resource from server")
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.ErrorLogWithFixedMessage(ctx, err, "The resource could be in deleting state. Ignoring")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		r.Log.ErrorLogWithFixedMessage(ctx, err, "Error while get the Resource from server.")
		return ctrl.Result{}, err
	}

	r.Log.InfoLogWithFixedMessage(ctx, "Got the status of resource")

	if obj.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(obj, OSOKFinalizerName) {
			r.Log.InfoLogWithFixedMessage(ctx, "The Deletion time is non zero. Deleting the resource")

			oldObj := obj.DeepCopyObject().(client.Object)
			deleteResult, err := r.deleteResourceResult(ctx, obj, req)
			if err != nil || !deleteResult.Deleted {
				if patchErr := r.patchDeleteStatusIfChanged(ctx, oldObj, obj, req); patchErr != nil {
					err = errors.Join(err, patchErr)
				}
			}
			if err != nil {
				r.Log.ErrorLogWithFixedMessage(ctx, err, r.messageWithAsyncBreadcrumb(obj, "Requeuing object due to error during delete of CR"))
				r.Metrics.AddCRDeleteFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
					"Requeuing object due to error during delete of CR", req.Name, req.Namespace)
				return util.RequeueWithError(ctx, err, defaultRequeueTime, r.Log)
			}
			if deleteResult.Deleted {
				if err := r.removeFinalizer(ctx, obj, strings.Join(r.AdditionalFinalizers, " "), OSOKFinalizerName); err != nil {
					r.Log.ErrorLogWithFixedMessage(ctx, err, "Failed to remove the finalizer")
					r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
						fmt.Sprintf("Failed to remove the finalizer: %s", err.Error()))
					return util.RequeueWithError(ctx, err, defaultRequeueTime, r.Log)
				}
				r.Log.InfoLogWithFixedMessage(ctx, "Deletion of the CR successful")
				r.Metrics.AddCRDeleteSuccessMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
					"Deletion of the CR successful", req.Name, req.Namespace)
				r.Recorder.Event(obj, v1.EventTypeNormal, "Success", "Removed finalizer")
				return util.DoNotRequeue()
			} else {
				requeueDuration := deleteResult.RequeueDuration
				if requeueDuration <= 0 {
					requeueDuration = defaultRequeueTime
				}
				return util.RequeueWithoutError(ctx, requeueDuration, r.Log)
			}
		}
	}

	if err := r.addFinalizer(ctx, obj, strings.Join(r.AdditionalFinalizers, " "), OSOKFinalizerName); err != nil {
		r.Log.ErrorLogWithFixedMessage(ctx, err, "Error adding finalizer to Custom Resource.")
		r.Metrics.AddReconcileFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"Error adding finalizer to Custom Resource.", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed", "Failed to add finalizer")
		return util.RequeueWithError(ctx, err, defaultRequeueTime, r.Log)
	}

	r.Log.InfoLogWithFixedMessage(ctx, "Reconcile the resource")
	return r.ReconcileResource(ctx, obj, req)
}

func (r *BaseReconciler) GetStatus(obj client.Object) (*shared.OSOKStatus, error) {
	status, err := r.OSOKServiceManager.GetCrdStatus(obj)
	if err != nil {
		return nil, err
	}

	if status.RequestedAt == nil {
		now := metav1.NewTime(time.Now())
		status.RequestedAt = &now
	}

	return status, nil
}

func (r *BaseReconciler) messageWithAsyncBreadcrumb(obj client.Object, message string) string {
	if obj == nil {
		return message
	}

	status, err := r.OSOKServiceManager.GetCrdStatus(obj)
	if err != nil || status == nil || status.Async.Current == nil {
		return message
	}

	breadcrumb := make([]string, 0, 2)
	if phase := strings.TrimSpace(string(status.Async.Current.Phase)); phase != "" {
		breadcrumb = append(breadcrumb, "async phase="+phase)
	}
	if workRequestID := strings.TrimSpace(status.Async.Current.WorkRequestID); workRequestID != "" {
		breadcrumb = append(breadcrumb, "workRequestId="+workRequestID)
	}
	if len(breadcrumb) == 0 {
		return message
	}

	return fmt.Sprintf("%s (%s)", message, strings.Join(breadcrumb, ", "))
}

func (r *BaseReconciler) ReconcileResource(ctx context.Context, obj client.Object, req ctrl.Request) (ctrl.Result, error) {
	ctx = metrics.AddFixedLogMapEntries(ctx, req.Name, req.Namespace)

	oldObj := obj.DeepCopyObject().(client.Object)
	OSOKResponse, err := r.OSOKServiceManager.CreateOrUpdate(ctx, obj, req)
	if err != nil {
		r.Log.ErrorLogWithFixedMessage(ctx, err, r.messageWithAsyncBreadcrumb(obj, "Create Or Update failed in the Service Manager with error"))
		r.Metrics.AddReconcileFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"Create Or Update failed in the Service Manager", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
			r.messageWithAsyncBreadcrumb(obj, fmt.Sprintf("Failed to create or update resource: %s", err.Error())))
	}

	if err := r.Status().Patch(ctx, obj, client.MergeFrom(oldObj)); err != nil {
		r.Log.ErrorLogWithFixedMessage(ctx, err, r.messageWithAsyncBreadcrumb(obj, "Error updating the status of the Object"))
		r.Metrics.AddReconcileFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"Error updating the status of the CR", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
			r.messageWithAsyncBreadcrumb(obj, fmt.Sprintf("Failed to create or update resource: %s", err.Error())))
		return util.RequeueWithError(ctx, err, defaultRequeueTime, r.Log)
	}
	r.Metrics.AddCRCountMetrics(ctx, r.Metrics.ServiceName, "Created an Custom resource "+r.Metrics.ServiceName,
		req.Name, req.Namespace)

	if OSOKResponse.IsSuccessful {
		r.Log.InfoLogWithFixedMessage(ctx, r.messageWithAsyncBreadcrumb(obj, "Reconcile Completed"))
		r.Metrics.AddReconcileSuccessMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"Create or Update of resource succeeded", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeNormal, "Success",
			r.messageWithAsyncBreadcrumb(obj, "Create or Update of resource succeeded"))
		if OSOKResponse.ShouldRequeue {
			return util.RequeueWithoutError(ctx, OSOKResponse.RequeueDuration, r.Log)
		}
		return util.DoNotRequeue()
	} else {
		r.Log.InfoLogWithFixedMessage(ctx, r.messageWithAsyncBreadcrumb(obj, "Reconcile Failed"))
		r.Metrics.AddReconcileFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"Failed to create or update resource", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
			r.messageWithAsyncBreadcrumb(obj, "Failed to create or update resource"))
		if OSOKResponse.ShouldRequeue {
			return ctrl.Result{Requeue: true}, err
		}
		return util.DoNotRequeue()
	}
}

func (r *BaseReconciler) deleteResourceResult(ctx context.Context, obj client.Object, req ctrl.Request) (servicemanager.OSOKDeleteResult, error) {
	ctx = metrics.AddFixedLogMapEntries(ctx, req.Name, req.Namespace)
	var (
		delResult servicemanager.OSOKDeleteResult
		err       error
	)
	if manager, ok := r.OSOKServiceManager.(servicemanager.OSOKDeleteResultProvider); ok {
		delResult, err = manager.DeleteWithResult(ctx, obj)
	} else {
		delResult.Deleted, err = r.OSOKServiceManager.Delete(ctx, obj)
	}
	if err != nil {
		classification := errorutil.ClassifyDeleteError(err)
		if classification.IsUnambiguousNotFound() {
			r.Log.InfoLogWithFixedMessage(ctx, "Delete treated as successful because OCI resource no longer exists",
				"oci_http_status_code", classification.HTTPStatusCodeString(),
				"oci_error_code", classification.ErrorCodeString(),
				"normalized_error_type", classification.NormalizedTypeString())
			return servicemanager.OSOKDeleteResult{Deleted: true}, nil
		}
		if classification.IsConflict() {
			r.Log.InfoLogWithFixedMessage(ctx, r.messageWithAsyncBreadcrumb(obj, "Delete is blocked and will be retried"),
				"oci_http_status_code", classification.HTTPStatusCodeString(),
				"oci_error_code", classification.ErrorCodeString(),
				"normalized_error_type", classification.NormalizedTypeString())
			r.Recorder.Event(obj, v1.EventTypeNormal, deleteEventReasonBlocked,
				r.messageWithAsyncBreadcrumb(obj, fmt.Sprintf("Delete blocked and will be retried: %s", err.Error())))
			return servicemanager.OSOKDeleteResult{Deleted: false}, nil
		}
		r.Log.ErrorLogWithFixedMessage(ctx, err, r.messageWithAsyncBreadcrumb(obj, "Delete failed in the Service Manager with error"), "name", req.Name,
			"namespace", req.Namespace, "namespacedName", req.String(),
			"oci_http_status_code", classification.HTTPStatusCodeString(),
			"oci_error_code", classification.ErrorCodeString(),
			"normalized_error_type", classification.NormalizedTypeString())
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
			r.messageWithAsyncBreadcrumb(obj, fmt.Sprintf("Failed to delete resource: %s", err.Error())))
		return servicemanager.OSOKDeleteResult{}, err
	}
	if delResult.Deleted {
		r.Log.InfoLogWithFixedMessage(ctx, "Delete Successful")
	} else {
		r.Log.InfoLogWithFixedMessage(ctx, r.messageWithAsyncBreadcrumb(obj, "Delete is in progress and will be retried"))
		r.Recorder.Event(obj, v1.EventTypeNormal, deleteEventReasonInProgress,
			r.messageWithAsyncBreadcrumb(obj, "Delete is in progress"))
	}
	return delResult, nil
}

func (r *BaseReconciler) DeleteResource(ctx context.Context, obj client.Object, req ctrl.Request) (bool, error) {
	result, err := r.deleteResourceResult(ctx, obj, req)
	return result.Deleted, err
}

func (r *BaseReconciler) patchDeleteStatusIfChanged(ctx context.Context, oldObj, obj client.Object, req ctrl.Request) error {
	if equality.Semantic.DeepEqual(oldObj, obj) {
		return nil
	}

	if err := r.Status().Patch(ctx, obj, client.MergeFrom(oldObj)); err != nil {
		r.Log.ErrorLogWithFixedMessage(ctx, err, "Error updating the status of the Object during delete")
		r.Metrics.AddCRDeleteFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"Error updating the status of the CR during delete", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
			fmt.Sprintf("Failed to persist delete status: %s", err.Error()))
		return err
	}

	return nil
}

func (r *BaseReconciler) addFinalizer(ctx context.Context, obj client.Object, finalizers ...string) error {
	needsUpdate := false
	for _, finalizer := range finalizers {
		if finalizer != "" && !controllerutil.ContainsFinalizer(obj, finalizer) {
			controllerutil.AddFinalizer(obj, finalizer)
			needsUpdate = true
		}
	}
	if !needsUpdate {
		return nil
	}
	r.Log.InfoLogWithFixedMessage(ctx, "Added Finalizer to the resource.")
	r.Recorder.Event(obj, v1.EventTypeNormal, "Success", "Finalizer is added to the object")
	return r.Update(ctx, obj)
}

func (r *BaseReconciler) removeFinalizer(ctx context.Context, obj client.Object, finalizers ...string) error {
	needsUpdate := false
	for _, finalizer := range finalizers {
		if finalizer != "" && controllerutil.ContainsFinalizer(obj, finalizer) {
			controllerutil.RemoveFinalizer(obj, finalizer)
			needsUpdate = true
		}
	}
	if !needsUpdate {
		return nil
	}
	r.Log.InfoLogWithFixedMessage(ctx, "Removing Finalizer from the resource.")
	r.Recorder.Event(obj, v1.EventTypeNormal, "Success", "Finalizer is removed from the object")
	return r.Update(ctx, obj)
}
