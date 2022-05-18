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

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/upgradeproxy"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	merrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
)

type UpgradeProxyReconciler struct {
	client    client.Client
	log       loggerutil.OSOKLogger
	handler   upgradeproxy.ResourceHandler
	namespace string
}

func NewUpgradeProxyReconciler(
	client client.Client,
	log loggerutil.OSOKLogger,
	handler upgradeproxy.ResourceHandler,
	namespace string,
) *UpgradeProxyReconciler {
	return &UpgradeProxyReconciler{client, log, handler, namespace}
}

//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps/status,verbs=get;update;patch

// Reconcile will keep a watch on mesh configmap for the proxy version update and evicts pods that has older proxy versions
// Polls data plane endpoint for latest proxy versions, and updates the configmap if there is a version mismatch
// If configmap is deleted, it recreates it.
func (r *UpgradeProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return merrors.HandleErrorAndRequeue(ctx, r.reconcileProxy(ctx, req), r.log)
}

func (r *UpgradeProxyReconciler) reconcileProxy(ctx context.Context, req ctrl.Request) error {
	if req.Namespace != r.namespace || req.Name != commons.MeshConfigMapName {
		return nil
	}
	configMap := &corev1.ConfigMap{}
	err := r.client.Get(ctx, req.NamespacedName, configMap)
	if err != nil {
		if errors.IsNotFound(err) {
			r.log.InfoLogWithFixedMessage(ctx, "configMap is deleted, creating one...")
			return nil
		}
		r.log.ErrorLogWithFixedMessage(ctx, err, "Error in reading config map object")
		return merrors.NewRequeueOnError(err)
	}
	if !configMap.DeletionTimestamp.IsZero() {
		r.log.InfoLogWithFixedMessage(ctx, "ConfigMap is about to be deleted, creating one...")
		return nil
	}
	return r.handler.Reconcile(ctx, configMap)
}

// SetupWithManager sets up the controller with the Manager.
func (r *UpgradeProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			RateLimiter: commons.DefaultControllerRateLimiter(),
		}).
		For(&corev1.ConfigMap{}).
		Complete(r)
}
