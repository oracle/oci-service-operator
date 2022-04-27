/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package controllers

import (
	"context"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/serviceupdate"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	merrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	log     loggerutil.OSOKLogger
	handler serviceupdate.ResourceHandler
}

func NewServiceReconciler(
	log loggerutil.OSOKLogger,
	handler serviceupdate.ResourceHandler) *ServiceReconciler {
	return &ServiceReconciler{log, handler}
}

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods/status,verbs=get
//+kubebuilder:rbac:groups="",resources=pods/eviction,verbs=get;list;watch;create;update;patch;delete

//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=namespaces/status,verbs=get

// This reconcile is to handle a case where label selectors on the service are changed
// if existing pods appear to have matched the VDBs for the updated service this controller will evict such pods
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return merrors.HandleErrorAndRequeue(r.reconcile(ctx, req), r.log)
}

func (r *ServiceReconciler) reconcile(ctx context.Context, req ctrl.Request) error {
	var namespaceInjectionLabel string
	if found, err := r.handler.FetchNamespaceInjectionLabel(ctx, req, &namespaceInjectionLabel); !found {
		return err
	}

	service, err := r.handler.FetchService(ctx, req.NamespacedName)
	if err != nil {
		return err
	}

	return r.handler.Reconcile(ctx, service, namespaceInjectionLabel)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			RateLimiter: commons.DefaultControllerRateLimiter(),
		}).
		For(&corev1.Service{}).
		Complete(r)
}
