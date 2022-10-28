/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeployment

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

// GetVDActiveStatus returns the current status of the VD
func GetVDActiveStatus(vd *servicemeshapi.VirtualDeployment) metav1.ConditionStatus {
	for _, condition := range vd.Status.Conditions {
		if condition.Type == servicemeshapi.ServiceMeshActive {
			return condition.Status
		}
	}
	return metav1.ConditionUnknown
}

// IsVdActiveCp tests whether given virtual deployment is active.
// virtual deployment is active when its VirtualDeploymentActive condition equals true.
func IsVdActiveCp(vd *sdk.VirtualDeployment) bool {
	return vd.LifecycleState == sdk.VirtualDeploymentLifecycleStateActive
}
