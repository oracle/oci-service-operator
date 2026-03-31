/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package registrations

import (
	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var manualGroupRegistrations = []GroupRegistration{}

var manualWebhookRegistrations = []WebhookRegistration{
	{
		Name: "AutonomousDatabase",
		SetupWithManager: func(mgr ctrl.Manager) error {
			return (&databasev1beta1.AutonomousDatabase{}).SetupWebhookWithManager(mgr)
		},
	},
}
