/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package k8s

import "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"

var Resources = []commons.MeshResources{
	commons.Mesh,
	commons.VirtualService,
	commons.VirtualServiceRouteTable,
	commons.VirtualDeployment,
	commons.VirtualDeploymentBinding,
	commons.IngressGateway,
	commons.IngressGatewayRouteTable,
	commons.IngressGatewayDeployment,
	commons.AccessPolicy,
}

func IsValidMeshResource(value string) bool {
	for _, r := range Resources {
		if string(r) == value {
			return true
		}
	}
	return false
}
