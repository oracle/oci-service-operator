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

// IngressGatewaySpec defines the desired state of IngressGateway
type IngressGatewaySpec struct {
	api.TagResources `json:",inline"`
	// +optional
	Name          *Name    `json:"name,omitempty"`
	CompartmentId api.OCID `json:"compartmentId"`
	Mesh          RefOrId  `json:"mesh"`
	// +optional
	Description *Description `json:"description,omitempty"`
	// +optional
	AccessLogging *AccessLogging `json:"accessLogging,omitempty"`
	// +kubebuilder:validation:MinItems=1
	Hosts []IngressGatewayHost `json:"hosts"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
}

// IngressGatewayHost Host for the ingress gateway
type IngressGatewayHost struct {
	// A user-friendly name for the host. The name must be unique within the same ingress gateway.
	// This name can be used in the ingress gateway route table resource to attach a route to this host.
	Name Name `json:"name"`
	// Hostnames of the host.
	// Wildcard hostnames are supported in the prefix form.
	// Examples of valid hostnames are www.example.com, *.example.com, *.com
	// Applicable only for HTTP and TLS_PASSTHROUGH listeners.
	// +optional
	Hostnames []string `json:"hostnames,omitempty"`
	// The listeners for the ingress host
	// +kubebuilder:validation:MinItems=1
	Listeners []IngressGatewayListener `json:"listeners"`
}

// Listener configuration for ingress gateway host
type IngressGatewayListener struct {
	// Type of protocol used in resource
	Protocol IngressGatewayListenerProtocolEnum `json:"protocol"`
	// Port in which resource is running
	Port Port `json:"port"`
	// +optional
	Tls *IngressListenerTlsConfig `json:"tls,omitempty"`
}

// IngressGatewayListenerProtocolEnum Enum with underlying type: string
// +kubebuilder:validation:Enum=HTTP;TCP;TLS_PASSTHROUGH
type IngressGatewayListenerProtocolEnum string

// Set of constants representing the allowable values for IngressGatewayListenerProtocolEnum
const (
	IngressGatewayListenerProtocolHttp           IngressGatewayListenerProtocolEnum = "HTTP"
	IngressGatewayListenerProtocolTcp            IngressGatewayListenerProtocolEnum = "TCP"
	IngressGatewayListenerProtocolTlsPassthrough IngressGatewayListenerProtocolEnum = "TLS_PASSTHROUGH"
)

// IngressListenerTlsConfig TLS enforcement config for the ingress listener
type IngressListenerTlsConfig struct {

	// DISABLED: Connection can only be plaintext
	// PERMISSIVE: Connection can be either plaintext or TLS/mTLS. If the clientValidation.trustedCaBundle property is configured for the listener, mTLS will be performed and the client's certificates will be validated by the gateway.
	// TLS: Connection can only be TLS
	// MUTUAL_TLS: Connection can only be MTLS
	Mode IngressListenerTlsConfigModeEnum `json:"mode"`
	// +optional
	ServerCertificate *TlsCertificate `json:"serverCertificate,omitempty"`
	// +optional
	ClientValidation *IngressHostClientValidationConfig `json:"clientValidation,omitempty"`
}

// IngressGatewayMutualTransportLayerSecurity sets mTLS settings used when sending requests to virtual services within the mesh.
type IngressGatewayMutualTransportLayerSecurity struct {
	// The OCID of the certificate resource that will be used for mTLS authentication with other virtual services
	// in the mesh.
	CertificateId api.OCID `json:"certificateId"`
}

// IngressListenerTlsConfigModeEnum Enum with underlying type: string
// +kubebuilder:validation:Enum=DISABLED;PERMISSIVE;TLS;MUTUAL_TLS
type IngressListenerTlsConfigModeEnum string

// Set of constants representing the allowable values for IngressListenerTlsConfigModeEnum
const (
	IngressListenerTlsConfigModeDisabled   IngressListenerTlsConfigModeEnum = "DISABLED"
	IngressListenerTlsConfigModePermissive IngressListenerTlsConfigModeEnum = "PERMISSIVE"
	IngressListenerTlsConfigModeTls        IngressListenerTlsConfigModeEnum = "TLS"
	IngressListenerTlsConfigModeMutualTls  IngressListenerTlsConfigModeEnum = "MUTUAL_TLS"
)

// IngressHostClientValidationConfig Resource representing the TLS configuration used for validating client certificates.
type IngressHostClientValidationConfig struct {
	// +optional
	TrustedCaBundle *CaBundle `json:"trustedCaBundle,omitempty"`

	// A list of alternate names to verify the subject identity in the certificate presented by the client.
	// +optional
	SubjectAlternateNames []string `json:"subjectAlternateNames,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Active",type="string",JSONPath=".status.conditions[2].status",description="status of ingress gateway"
//+kubebuilder:printcolumn:name="Configured",type="string",JSONPath=".status.conditions[1].status",description="status of ingress gateway configured",priority=1
//+kubebuilder:printcolumn:name="DependenciesActive",type="string",JSONPath=".status.conditions[0].status",description="status of ingress gateway dependencies",priority=1
//+kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.ingressGatewayId",description="ocid of ingress gateway",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// IngressGateway is the Schema for the ingressgateways API
type IngressGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IngressGatewaySpec `json:"spec,omitempty"`
	Status ServiceMeshStatus  `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IngressGatewayList contains a list of IngressGateway
type IngressGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IngressGateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IngressGateway{}, &IngressGatewayList{})
}
