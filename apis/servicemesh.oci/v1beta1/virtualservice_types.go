/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Describes the Routing policy for the resource
// Only one of the following policy should be present.
// +kubebuilder:validation:Enum=UNIFORM;DENY
type RoutingPolicy string

const (
	RoutingPolicyUniform RoutingPolicy = "UNIFORM"
	RoutingPolicyDeny    RoutingPolicy = "DENY"
)

// Routing policy for the Virtual Service
type DefaultRoutingPolicy struct {
	Type RoutingPolicy `json:"type"`
}

// CreateVirtualServiceMutualTransportLayerSecurity sets the mTLS authentication mode to use when communicating with other virtual services
type CreateVirtualServiceMutualTransportLayerSecurity struct {

	// DISABLED: Connection is not tunneled.
	// PERMISSIVE: Connection can be either plaintext or an mTLS tunnel.
	// STRICT: Connection is an mTLS tunnel.  Clients without a valid certificate will be rejected.
	Mode MutualTransportLayerSecurityModeEnum `json:"mode"`
}

// VirtualServiceMutualTransportLayerSecurity sets mTLS settings used when communicating with other virtual services within the mesh.
type VirtualServiceMutualTransportLayerSecurity struct {
	// The OCID of the certificate resource that will be used for mTLS authentication with other virtual services
	// in the mesh.
	// +optional
	CertificateId *api.OCID                            `json:"certificateId,omitempty"`
	Mode          MutualTransportLayerSecurityModeEnum `json:"mode"`
}

// VirtualServiceSpec defines the desired state of VirtualService
type VirtualServiceSpec struct {
	api.TagResources `json:",inline"`
	// +optional
	Name          *Name    `json:"name,omitempty"`
	CompartmentId api.OCID `json:"compartmentId"`
	Mesh          RefOrId  `json:"mesh"`
	// +optional
	Description *Description `json:"description,omitempty"`
	// +optional
	DefaultRoutingPolicy *DefaultRoutingPolicy `json:"defaultRoutingPolicy,omitempty"`
	// The DNS hostnames of the virtual service that is used by its callers.
	// Wildcard hostnames are supported in the prefix form.
	// Examples of valid hostnames are www.example.com, *.example.com, *.com
	// +kubebuilder:validation:MinItems=1
	Hosts []string `json:"hosts"`
	// +optional
	Mtls *CreateVirtualServiceMutualTransportLayerSecurity `json:"mtls,omitempty"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Active",type="string",JSONPath=".status.conditions[2].status",description="status of virtual service"
//+kubebuilder:printcolumn:name="Configured",type="string",JSONPath=".status.conditions[1].status",description="status of virtual service configured",priority=1
//+kubebuilder:printcolumn:name="DependenciesActive",type="string",JSONPath=".status.conditions[0].status",description="status of virtual service dependencies",priority=1
//+kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.virtualServiceId",description="ocid of virtual service",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// VirtualService is the Schema for the virtualservices API
type VirtualService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualServiceSpec `json:"spec,omitempty"`
	Status ServiceMeshStatus  `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VirtualServiceList contains a list of VirtualService
type VirtualServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualService{}, &VirtualServiceList{})
}
