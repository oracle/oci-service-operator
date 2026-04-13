/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package registrations

import (
	"fmt"

	containerenginev1beta1 "github.com/oracle/oci-service-operator/api/containerengine/v1beta1"
	containerenginecontrollers "github.com/oracle/oci-service-operator/controllers/containerengine"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	containerengineclusterservicemanager "github.com/oracle/oci-service-operator/pkg/servicemanager/containerengine/cluster"
)

func init() {
	manualGroupRegistrations = appendUniqueGroupRegistrations(manualGroupRegistrations, GroupRegistration{
		Group:       "containerengine",
		AddToScheme: containerenginev1beta1.AddToScheme,
		SetupWithManager: func(ctx Context) error {
			if err := (&containerenginecontrollers.ClusterReconciler{
				Reconciler: NewBaseReconciler(
					ctx,
					"Cluster",
					func(deps servicemanager.RuntimeDeps) servicemanager.OSOKServiceManager {
						return containerengineclusterservicemanager.NewClusterServiceManagerWithDeps(deps)
					},
				),
			}).SetupWithManager(ctx.Manager); err != nil {
				return fmt.Errorf("setup Cluster controller: %w", err)
			}
			return nil
		},
	})
}
