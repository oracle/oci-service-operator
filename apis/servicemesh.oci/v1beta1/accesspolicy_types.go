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

// AccessPolicySpec defines the desired state of AccessPolicy
type AccessPolicySpec struct {
	api.TagResources `json:",inline"`
	// +optional
	Name          *Name    `json:"name,omitempty"`
	CompartmentId api.OCID `json:"compartmentId"`
	Mesh          RefOrId  `json:"mesh"`
	// +optional
	Description *Description `json:"description,omitempty"`
	// Access Policy Rules
	// +kubebuilder:validation:MinItems=1
	Rules []AccessPolicyRule `json:"rules,omitempty"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
}

// Indicates the action for the traffic between the source and the destination.
// +kubebuilder:validation:Enum=ALLOW
type ActionType string

const (
	ActionTypeAllow ActionType = "ALLOW"
)

// Describes the applicable rules for the Access Policy
type AccessPolicyRule struct {
	Action      ActionType    `json:"action"`
	Source      TrafficTarget `json:"source"`
	Destination TrafficTarget `json:"destination"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Active",type="string",JSONPath=".status.conditions[2].status",description="status of access policy"
//+kubebuilder:printcolumn:name="Configured",type="string",JSONPath=".status.conditions[1].status",description="status of access policy configured",priority=1
//+kubebuilder:printcolumn:name="DependenciesActive",type="string",JSONPath=".status.conditions[0].status",description="status of access policy dependencies",priority=1
//+kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.accessPolicyId",description="ocid of access policy",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// AccessPolicy is the Schema for the accesspolicies API
type AccessPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccessPolicySpec  `json:"spec,omitempty"`
	Status ServiceMeshStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AccessPolicyList contains a list of AccessPolicy
type AccessPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AccessPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AccessPolicy{}, &AccessPolicyList{})
}
