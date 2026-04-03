/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package registrations

import (
	"fmt"

	containerinstancesv1beta1 "github.com/oracle/oci-service-operator/api/containerinstances/v1beta1"
	containerinstancescontrollers "github.com/oracle/oci-service-operator/controllers/containerinstances"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	containerinstanceservicemanager "github.com/oracle/oci-service-operator/pkg/servicemanager/containerinstance"
)

func init() {
	manualGroupRegistrations = append(manualGroupRegistrations, GroupRegistration{
		Group:       "containerinstances",
		AddToScheme: containerinstancesv1beta1.AddToScheme,
		SetupWithManager: func(ctx Context) error {
			if err := (&containerinstancescontrollers.ContainerInstanceReconciler{
				Reconciler: NewBaseReconciler(
					ctx,
					"ContainerInstance",
					func(deps servicemanager.RuntimeDeps) servicemanager.OSOKServiceManager {
						return containerinstanceservicemanager.NewContainerInstanceServiceManagerWithDeps(deps)
					},
				),
			}).SetupWithManager(ctx.Manager); err != nil {
				return fmt.Errorf("setup ContainerInstance controller: %w", err)
			}
			return nil
		},
	})
}
