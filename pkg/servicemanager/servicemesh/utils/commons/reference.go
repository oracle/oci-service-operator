/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package commons

import (
	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

type ResourceRef struct {
	Id     api.OCID
	Name   servicemeshapi.Name
	MeshId api.OCID
}

type MeshRef struct {
	Id          api.OCID
	DisplayName servicemeshapi.Name
	Mtls        servicemeshapi.MeshMutualTransportLayerSecurity
}
