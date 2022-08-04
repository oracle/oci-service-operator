/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicemeshoci

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/oracle/oci-service-operator/pkg/core"
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
)

// ServiceMeshReconciler reconciles a ServiceMesh CRD object
type ServiceMeshReconciler struct {
	Reconciler     *core.BaseReconciler
	ResourceObject client.Object
	CustomWatches  []CustomWatch
}

type CustomWatch struct {
	Src          source.Source
	EventHandler handler.EventHandler
}

func (r *ServiceMeshReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(ctx, req, r.ResourceObject)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceMeshReconciler) SetupWithManagerWithMaxDelay(mgr ctrl.Manager, maxDelay time.Duration) error {
	pred := predicate.GenerationChangedPredicate{}
	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			RateLimiter: meshCommons.DefaultControllerRateLimiter(maxDelay),
		}).
		For(r.ResourceObject).
		WithEventFilter(pred)

	if r.CustomWatches != nil {
		for i := range r.CustomWatches {
			builder.Watches(r.CustomWatches[i].Src, r.CustomWatches[i].EventHandler)
		}
	}
	return builder.Complete(r)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceMeshReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return r.SetupWithManagerWithMaxDelay(mgr, meshCommons.MaxControllerDelay)
}
