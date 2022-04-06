/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package controllers

import (
	"context"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/injectproxy"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	merrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
)

type InjectProxyReconciler struct {
	client  client.Client
	log     loggerutil.OSOKLogger
	handler injectproxy.ResourceHandler
}

func NewInjectProxyReconciler(
	client client.Client,
	log loggerutil.OSOKLogger,
	handler injectproxy.ResourceHandler,
) *InjectProxyReconciler {
	return &InjectProxyReconciler{client, log, handler}
}

// Reconcile will keep a watch on namespaces with proxy injection label present and inject proxies initially by evicting the pods
func (r *InjectProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return merrors.HandleErrorAndRequeue(r.reconcileProxy(ctx, req), r.log)
}

func (r *InjectProxyReconciler) reconcileProxy(ctx context.Context, req ctrl.Request) error {
	namespace := &corev1.Namespace{}
	err := r.client.Get(ctx, req.NamespacedName, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			r.log.InfoLog("Namespace deleted")
			return nil
		}
		r.log.ErrorLog(err, "Error in reading Namespace object")
		return merrors.NewRequeueOnError(err)
	}
	if !namespace.DeletionTimestamp.IsZero() {
		r.log.InfoLog("Namespace DeletionTimestamp is set, about to be deleted")
		return nil
	}
	return r.handler.Reconcile(ctx, namespace)
}

func (r *InjectProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			RateLimiter: commons.DefaultControllerRateLimiter(),
		}).
		For(&corev1.Namespace{}).
		Complete(r)
}
