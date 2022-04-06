/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
)

// MeshSpec defines the desired state of Mesh
type MeshSpec struct {
	api.TagResources `json:",inline"`
	// +optional
	DisplayName *Name `json:"displayName,omitempty"`
	// The compartment id for the resource
	CompartmentId api.OCID `json:"compartmentId"`
	// +optional
	Description *Description `json:"description,omitempty"`
	// +kubebuilder:validation:MinItems=1
	CertificateAuthorities []CertificateAuthority `json:"certificateAuthorities"`
	// +optional
	Mtls *MeshMutualTransportLayerSecurity `json:"mtls,omitempty"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
}

// CertificateAuthority defines the resource to use for creating leaf certificates.
type CertificateAuthority struct {
	Id api.OCID `json:"id"`
}

// MeshMutualTransportLayerSecurity sets a minimum level of mTLS authentication for all virtual services within the mesh
type MeshMutualTransportLayerSecurity struct {

	// DISABLED: No minimum, virtual services within this mesh can use any mTLS authentication mode.
	// PERMISSIVE: Virtual services within this mesh can use either PERMISSIVE or STRICT modes.
	// STRICT: All virtual services within this mesh must use STRICT mode.
	Minimum MutualTransportLayerSecurityModeEnum `json:"minimum"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Active",type="string",JSONPath=".status.conditions[2].status",description="status of mesh"
//+kubebuilder:printcolumn:name="Configured",type="string",JSONPath=".status.conditions[1].status",description="status of mesh configured",priority=1
//+kubebuilder:printcolumn:name="DependenciesActive",type="string",JSONPath=".status.conditions[0].status",description="status of mesh dependencies",priority=1
//+kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.meshId",description="ocid of mesh",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Mesh is the Schema for the meshes API
type Mesh struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MeshSpec          `json:"spec,omitempty"`
	Status ServiceMeshStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// MeshList contains a list of Mesh
type MeshList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Mesh `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Mesh{}, &MeshList{})
}
