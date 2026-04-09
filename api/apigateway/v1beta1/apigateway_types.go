/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApiGatewaySpec defines the desired state of ApiGateway.
type ApiGatewaySpec struct {
	// The OCID of an existing ApiGateway to bind to.
	ApiGatewayId shared.OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the gateway.
	// +kubebuilder:validation:Required
	CompartmentId shared.OCID `json:"compartmentId"`

	// DisplayName is a user-friendly name for the gateway.
	DisplayName string `json:"displayName,omitempty"`

	// EndpointType is the gateway endpoint type.
	// +kubebuilder:validation:Enum=PUBLIC;PRIVATE
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="endpointType is immutable"
	EndpointType string `json:"endpointType"`

	// SubnetId is the OCID of the subnet in which the gateway is created.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="subnetId is immutable"
	SubnetId shared.OCID `json:"subnetId"`

	// NetworkSecurityGroupIds is an optional list of NSG OCIDs associated with the gateway.
	NetworkSecurityGroupIds []string `json:"networkSecurityGroupIds,omitempty"`

	// CertificateId is the OCID of a certificate resource to use for HTTPS.
	CertificateId shared.OCID `json:"certificateId,omitempty"`

	shared.TagResources `json:",inline,omitempty"`
}

// ApiGatewayStatus defines the observed state of ApiGateway.
type ApiGatewayStatus struct {
	OsokStatus shared.OSOKStatus `json:"status"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="EndpointType",type="string",JSONPath=".spec.endpointType",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the ApiGateway",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the ApiGateway",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0
// ApiGateway is the Schema for the apigateways API.
type ApiGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApiGatewaySpec   `json:"spec,omitempty"`
	Status ApiGatewayStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// ApiGatewayList contains a list of ApiGateway.
type ApiGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApiGateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApiGateway{}, &ApiGatewayList{})
}
