/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package controllers

import (
	"context"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/core"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// StreamReconciler reconciles a Stream object
type StreamReconciler struct {
	StreamNameOld string
	Reconciler    *core.BaseReconciler
}

//+kubebuilder:rbac:groups=oci.oracle.com,resources=streams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=oci.oracle.com,resources=streams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=oci.oracle.com,resources=streams/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Stream object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *StreamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	stream := &ociv1beta1.Stream{}
	return r.Reconciler.Reconcile(ctx, req, stream)
}

// SetupWithManager sets up the controller with the Manager.
func (r *StreamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ociv1beta1.Stream{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
