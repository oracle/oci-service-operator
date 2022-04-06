/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VirtualDeploymentBindingSpec defines the desired state of VirtualDeploymentBinding
type VirtualDeploymentBindingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	VirtualDeployment RefOrId `json:"virtualDeployment"`
	Target            Target  `json:"target"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster

	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// Target identifies the Virtual Deployment for a Virtual Deployment Binding
type Target struct {
	Service Service `json:"service"`
}

// A reference to Service that a resource belongs to.
type Service struct {
	ServiceRef ResourceRef `json:"ref"`
	// matchLabels is a map of {key,value} pairs.
	// +optional
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Active",type="string",JSONPath=".status.conditions[1].status",description="status of virtual deployment binding"
//+kubebuilder:printcolumn:name="DependenciesActive",type="string",JSONPath=".status.conditions[0].status",description="status of virtual deployment binding dependencies",priority=1
//+kubebuilder:printcolumn:name="VirtualDeploymentOcid",type="string",JSONPath=".status.virtualDeploymentId",description="ocid of virtual deployment",priority=1
//+kubebuilder:printcolumn:name="VirtualDeploymentName",type="string",JSONPath=".status.virtualDeploymentName",description="name of virtual deployment",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// VirtualDeploymentBinding is the Schema for the virtualdeploymentbindings API
type VirtualDeploymentBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualDeploymentBindingSpec `json:"spec,omitempty"`
	Status ServiceMeshStatus            `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VirtualDeploymentBindingList contains a list of VirtualDeploymentBinding
type VirtualDeploymentBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualDeploymentBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualDeploymentBinding{}, &VirtualDeploymentBindingList{})
}
