/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package core

import (
	"context"
	"fmt"
	"github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

const (
	OSOKFinalizerName  = "finalizers.oci.oracle.com/oci-resources"
	defaultRequeueTime = time.Minute * 2
)

type BaseReconciler struct {
	client.Client
	OSOKServiceManager servicemanager.OSOKServiceManager
	Finalizer          Finalizer
	Log                loggerutil.OSOKLogger
	Metrics            *metrics.Metrics
	Recorder           record.EventRecorder
	Scheme             *runtime.Scheme
}

func (r *BaseReconciler) Reconcile(ctx context.Context, req ctrl.Request, obj client.Object) (result ctrl.Result, err error) {
	// To setup the fixed logs for every log
	r.Log.FixedLogs["name"] = req.Name
	r.Log.FixedLogs["namespace"] = req.Namespace
	r.Log.DebugLog("Fetching the resource from server")
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			r.Log.ErrorLog(err, "The resource could be in deleting state. Ignoring")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		r.Log.ErrorLog(err, "Error while get the Resource from server.")
		return ctrl.Result{}, err
	}

	r.Log.InfoLog("Got the status of resource")

	if obj.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(obj, OSOKFinalizerName) {
			r.Log.InfoLog("The Deletion time is non zero. Deleting the resource")

			delSuc, err := r.DeleteResource(ctx, obj, req)
			if err != nil {
				r.Log.ErrorLog(err, "Requeuing object due to error during delete of CR")
				r.Metrics.AddCRDeleteFaultMetrics(obj.GetObjectKind().GroupVersionKind().Kind,
					"Requeuing object due to error during delete of CR", req.Name, req.Namespace)
				r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
					fmt.Sprintf("Failed to remove the finalizer: %s", err.Error()))
				return util.RequeueWithError(err, defaultRequeueTime, r.Log)
			}
			if delSuc {
				if err := r.removeFinalizer(ctx, obj, req.Name, req.Namespace); err != nil {
					r.Log.ErrorLog(err, "Failed to remove the finalizer")
					r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
						fmt.Sprintf("Failed to remove the finalizer: %s", err.Error()))
					return util.RequeueWithError(err, defaultRequeueTime, r.Log)
				}
				r.Log.InfoLog("Deletion of the CR successful")
				r.Metrics.AddCRDeleteSuccessMetrics(obj.GetObjectKind().GroupVersionKind().Kind,
					"Deletion of the CR successful", req.Name, req.Namespace)
				r.Recorder.Event(obj, v1.EventTypeNormal, "Success", "Removed finalizer")
				return util.DoNotRequeue()
			} else {
				r.Log.ErrorLog(err, "Re-queuing object as delete was unsuccessful")
				r.Metrics.AddCRDeleteFaultMetrics(obj.GetObjectKind().GroupVersionKind().Kind,
					"Re-queuing object as delete was unsuccessful", req.Name, req.Namespace)
				r.Recorder.Event(obj, v1.EventTypeWarning, "Failed", "Failed Delete the resource")
				return util.RequeueWithoutError(defaultRequeueTime, r.Log)
			}
		}
	}

	if !controllerutil.ContainsFinalizer(obj, OSOKFinalizerName) {
		if err := r.addFinalizer(ctx, obj, req.Name, req.Namespace); err != nil {
			r.Log.ErrorLog(err, "Error adding finalizer to Custom Resource.")
			r.Metrics.AddReconcileFaultMetrics(obj.GetObjectKind().GroupVersionKind().Kind,
				"Error adding finalizer to Custom Resource.", req.Name, req.Namespace)
			r.Recorder.Event(obj, v1.EventTypeWarning, "Failed", "Failed to add finalizer")
			return util.RequeueWithError(err, defaultRequeueTime, r.Log)
		}
	}

	r.Log.InfoLog("Reconcile the resource")
	return r.ReconcileResource(ctx, obj, req)
}

func (r *BaseReconciler) GetStatus(obj client.Object) (*v1beta1.OSOKStatus, error) {
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

func (r *BaseReconciler) ReconcileResource(ctx context.Context, obj client.Object, req ctrl.Request) (ctrl.Result, error) {
	r.Log.FixedLogs["name"] = req.Name
	r.Log.FixedLogs["namespace"] = req.Namespace

	oldObj := obj.DeepCopyObject()
	sucFlag, err := r.OSOKServiceManager.CreateOrUpdate(ctx, obj, req)
	if err != nil {
		r.Log.ErrorLog(err, "Create Or Update failed in the Service Manager with error")
		r.Metrics.AddReconcileFaultMetrics(obj.GetObjectKind().GroupVersionKind().Kind,
			"Create Or Update failed in the Service Manager", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
			fmt.Sprintf("Failed to create or update resource: %s", err.Error()))
	}

	if err := r.Status().Patch(ctx, obj, client.MergeFrom(oldObj)); err != nil {
		r.Log.ErrorLog(err, "Error updating the status of the Object")
		r.Metrics.AddReconcileFaultMetrics(obj.GetObjectKind().GroupVersionKind().Kind,
			"Error updating the status of the CR", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
			fmt.Sprintf("Failed to create or update resource: %s", err.Error()))
		return util.RequeueWithError(err, defaultRequeueTime, r.Log)
	}
	r.Metrics.AddCRCountMetrics(r.Metrics.ServiceName, "Created an Custom resource "+r.Metrics.ServiceName,
		req.Name, req.Namespace)
	if sucFlag {
		r.Log.InfoLog("Reconcile Completed")
		r.Metrics.AddReconcileSuccessMetrics(obj.GetObjectKind().GroupVersionKind().Kind,
			"Create or Update of resource succeeded", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeNormal, "Success", "Create or Update of resource succeeded")
		return util.DoNotRequeue()
	} else {
		r.Log.InfoLog("Reconcile Failed")
		r.Metrics.AddReconcileFaultMetrics(obj.GetObjectKind().GroupVersionKind().Kind,
			"Failed to create or update resource", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed", "Failed to create or update resource")
		return util.DoNotRequeue()
	}
}

func (r *BaseReconciler) DeleteResource(ctx context.Context, obj client.Object, req ctrl.Request) (bool, error) {
	r.Log.FixedLogs["name"] = req.Name
	r.Log.FixedLogs["namespace"] = req.Namespace
	//log := util.LogUtil{Log: r.Log.WithValues("name", req.Name, "namespace", req.Namespace)}
	//TODO Emit Delete Start metrics
	delSucc, err := r.OSOKServiceManager.Delete(ctx, obj)
	if err != nil {
		r.Log.ErrorLog(err, "Delete failed in the Service Manager with error", "name", req.Name,
			"namespace", req.Namespace, "namespacedName", req.String())
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
			fmt.Sprintf("Failed to delete resource: %s", err.Error()))
		// TODO Emit Delete Fault metrics end
		return false, err
	}
	if delSucc {
		r.Log.InfoLog("Delete Successful")
	} else {
		r.Log.InfoLog("Delete Unsuccessful, re-queuing the request after 2 minutes")
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed", "Delete Unsuccessful")
	}
	// TODO Emit Delete Success metrics end
	return delSucc, nil
}

func (r *BaseReconciler) addFinalizer(ctx context.Context, obj client.Object, name string, ns string) error {
	controllerutil.AddFinalizer(obj, OSOKFinalizerName)
	r.Log.InfoLog("Added Finalizer to the resource.")
	r.Recorder.Event(obj, v1.EventTypeNormal, "Success", "Finalizer is added to the object")
	return r.Update(ctx, obj)
}

func (r *BaseReconciler) removeFinalizer(ctx context.Context, obj client.Object, name string, ns string) error {
	r.Log.InfoLog("Removing Finalizer to the resource.")
	controllerutil.RemoveFinalizer(obj, OSOKFinalizerName)
	return r.Update(ctx, obj)
}
