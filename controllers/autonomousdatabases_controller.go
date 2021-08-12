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

// AutonomousDatabasesReconciler reconciles a AutonomousDatabases object
type AutonomousDatabasesReconciler struct {
	DbNameOld  string
	Reconciler *core.BaseReconciler
}

// +kubebuilder:rbac:groups=oci.oracle.com,resources=autonomousdatabases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oci.oracle.com,resources=autonomousdatabases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=oci.oracle.com,resources=autonomousdatabases/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the AutonomousDatabases object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *AutonomousDatabasesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	autonomousDatabases := &ociv1beta1.AutonomousDatabases{}
	return r.Reconciler.Reconcile(ctx, req, autonomousDatabases)
}

// SetupWithManager sets up the controller with the Manager.
func (r *AutonomousDatabasesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ociv1beta1.AutonomousDatabases{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
