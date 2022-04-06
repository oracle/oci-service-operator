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

// VirtualDeploymentSpec defines the desired state of VirtualDeployment
type VirtualDeploymentSpec struct {
	api.TagResources `json:",inline"`
	// +optional
	Name          *Name    `json:"name,omitempty"`
	CompartmentId api.OCID `json:"compartmentId"`
	// +optional
	Description      *Description     `json:"description,omitempty"`
	VirtualService   RefOrId          `json:"virtualService"`
	ServiceDiscovery ServiceDiscovery `json:"serviceDiscovery"`
	// +optional
	AccessLogging *AccessLogging `json:"accessLogging,omitempty"`
	// +kubebuilder:validation:MinItems=1
	Listener []Listener `json:"listener"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
}

// Describes the ServiceDiscoveryType for Virtual Deployment
// Only one of the following type should be present.
// +kubebuilder:validation:Enum=DNS;
type ServiceDiscoveryType string

const (
	ServiceDiscoveryTypeDns ServiceDiscoveryType = "DNS"
)

// ServiceDiscovery configuration for Virtual Deployment
type ServiceDiscovery struct {
	Type     ServiceDiscoveryType `json:"type"`
	Hostname string               `json:"hostname,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Active",type="string",JSONPath=".status.conditions[2].status",description="status of virtual deployment"
//+kubebuilder:printcolumn:name="Configured",type="string",JSONPath=".status.conditions[1].status",description="status of virtual deployment configured",priority=1
//+kubebuilder:printcolumn:name="DependenciesActive",type="string",JSONPath=".status.conditions[0].status",description="status of virtual deployment dependencies",priority=1
//+kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.virtualDeploymentId",description="ocid of virtual deployment",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// VirtualDeployment is the Schema for the virtualdeployments API
type VirtualDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualDeploymentSpec `json:"spec,omitempty"`
	Status ServiceMeshStatus     `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VirtualDeploymentList contains a list of VirtualDeployment
type VirtualDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualDeployment{}, &VirtualDeploymentList{})
}
