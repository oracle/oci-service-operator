/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package registrations

import (
	"fmt"

	apigatewayv1beta1 "github.com/oracle/oci-service-operator/api/apigateway/v1beta1"
	apigatewaycontrollers "github.com/oracle/oci-service-operator/controllers/apigateway"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	apigatewayservicemanager "github.com/oracle/oci-service-operator/pkg/servicemanager/apigateway"
)

func init() {
	manualGroupRegistrations = append(manualGroupRegistrations, GroupRegistration{
		Group:       "apigateway",
		AddToScheme: apigatewayv1beta1.AddToScheme,
		SetupWithManager: func(ctx Context) error {
			if err := (&apigatewaycontrollers.ApiGatewayReconciler{
				Reconciler: NewBaseReconciler(
					ctx,
					"ApiGateway",
					func(deps servicemanager.RuntimeDeps) servicemanager.OSOKServiceManager {
						return apigatewayservicemanager.NewGatewayServiceManagerWithDeps(deps)
					},
				),
			}).SetupWithManager(ctx.Manager); err != nil {
				return fmt.Errorf("setup ApiGateway controller: %w", err)
			}
			if err := (&apigatewaycontrollers.ApiGatewayDeploymentReconciler{
				Reconciler: NewBaseReconciler(
					ctx,
					"ApiGatewayDeployment",
					func(deps servicemanager.RuntimeDeps) servicemanager.OSOKServiceManager {
						return apigatewayservicemanager.NewDeploymentServiceManagerWithDeps(deps)
					},
				),
			}).SetupWithManager(ctx.Manager); err != nil {
				return fmt.Errorf("setup ApiGatewayDeployment controller: %w", err)
			}
			return nil
		},
	})
}
