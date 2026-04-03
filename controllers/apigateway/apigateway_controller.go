/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apigateway

import (
	"context"

	apigatewayv1beta1 "github.com/oracle/oci-service-operator/api/apigateway/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/core"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ApiGatewayReconciler reconciles an ApiGateway object.
type ApiGatewayReconciler struct {
	Reconciler *core.BaseReconciler
}

// +kubebuilder:rbac:groups=apigateway.oracle.com,resources=apigateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apigateway.oracle.com,resources=apigateways/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apigateway.oracle.com,resources=apigateways/finalizers,verbs=update

// Reconcile is part of the main Kubernetes reconciliation loop.
func (r *ApiGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	resource := &apigatewayv1beta1.ApiGateway{}
	return r.Reconciler.Reconcile(ctx, req, resource)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApiGatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&apigatewayv1beta1.ApiGateway{})
	return builder.
		WithEventFilter(core.ReconcilePredicate()).
		Complete(r)
}
