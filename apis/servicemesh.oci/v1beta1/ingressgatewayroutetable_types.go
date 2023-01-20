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

// IngressGatewayRouteTableSpec defines the desired state of IngressGatewayRouteTable
type IngressGatewayRouteTableSpec struct {
	api.TagResources `json:",inline"`
	// +optional
	Name          *Name    `json:"name,omitempty"`
	CompartmentId api.OCID `json:"compartmentId"`
	// +optional
	Description    *Description `json:"description,omitempty"`
	IngressGateway RefOrId      `json:"ingressGateway"`
	// +kubebuilder:validation:MaxItems=15
	// +kubebuilder:validation:MinItems=1
	RouteRules []IngressGatewayTrafficRouteRule `json:"routeRules"`
	// +optional
	Priority *int `json:"priority,omitempty"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
}

// Route rules for the ingress traffic
type IngressGatewayTrafficRouteRule struct {
	// +optional
	HttpRoute *HttpIngressGatewayTrafficRouteRule `json:"httpRoute,omitempty"`
	// +optional
	TcpRoute *TcpIngressGatewayTrafficRouteRule `json:"tcpRoute,omitempty"`
	// +optional
	TlsPassthroughRoute *TlsPassthroughIngressGatewayTrafficRouteRule `json:"tlsPassthroughRoute,omitempty"`
}

// HttpIngressGatewayTrafficRouteRule Rule for routing incoming ingress gateway traffic with HTTP protocol
type HttpIngressGatewayTrafficRouteRule struct {

	// +optional
	IngressGatewayHost *IngressGatewayHostRef `json:"ingressGatewayHost,omitempty"`

	// The destination of the request.
	// +kubebuilder:validation:MaxItems=1
	// +kubebuilder:validation:MinItems=1
	Destinations []VirtualServiceTrafficRuleTarget `json:"destinations"`

	// Route to match
	// +optional
	Path *string `json:"path,omitempty"`

	// If true, the rule will check that the content-type header has a application/grpc
	// or one of the various application/grpc+ values.
	// +optional
	// +kubebuilder:default:= false
	IsGrpc *bool `json:"isGrpc,omitempty"`

	// Match type for the route
	// +optional
	PathType HttpIngressGatewayTrafficRouteRulePathTypeEnum `json:"pathType,omitempty"`

	// If true, the hostname will be rewritten to the target virtual deployment's DNS hostname.
	// +optional
	// +kubebuilder:default:= false
	IsHostRewriteEnabled *bool `json:"isHostRewriteEnabled,omitempty"`

	// If true, the matched path prefix will be rewritten to '/' before being directed to the target virtual deployment.
	// +optional
	// +kubebuilder:default:= false
	IsPathRewriteEnabled *bool `json:"isPathRewriteEnabled,omitempty"`

	// It is the maximum duration in milliseconds for the upstream service to respond to a request.
	// If provided, the timeout value overrides the default timeout of 15 seconds. The value 0 (zero) indicates that the timeout is disabled.
	// For streaming responses from the upstream service, it is suggested to either keep the timeout disabled or set a sufficiently high value.
	// +optional
	// +kubebuilder:validation:Minimum=0
	RequestTimeoutInMs *int64 `json:"requestTimeoutInMs,omitempty"`
}

// TcpIngressGatewayTrafficRouteRule Rule for routing incoming ingress gateway traffic with TCP protocol
type TcpIngressGatewayTrafficRouteRule struct {

	// +optional
	IngressGatewayHost *IngressGatewayHostRef `json:"ingressGatewayHost,omitempty"`

	// The destination of the request.
	// +kubebuilder:validation:MaxItems=1
	// +kubebuilder:validation:MinItems=1
	Destinations []VirtualServiceTrafficRuleTarget `json:"destinations"`
}

// TlsPassthroughIngressGatewayTrafficRouteRule Rule for routing incoming ingress gateway traffic with TLS_PASSTHROUGH protocol
type TlsPassthroughIngressGatewayTrafficRouteRule struct {

	// +optional
	IngressGatewayHost *IngressGatewayHostRef `json:"ingressGatewayHost,omitempty"`

	// The destination of the request.
	// +kubebuilder:validation:MaxItems=1
	// +kubebuilder:validation:MinItems=1
	Destinations []VirtualServiceTrafficRuleTarget `json:"destinations"`
}

// Destination of a Virtual Service
type VirtualServiceTrafficRuleTarget struct {
	VirtualService *RefOrId `json:"virtualService"`
	// Amount of traffic flows to a specific Virtual Service
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Weight *int `json:"weight"`
	// +optional
	Port *Port `json:"port,omitempty"`
}

// IngressGatewayHostRef The ingress gateway host to which the route rule attaches. If not specified, the route rule gets attached to all hosts on the ingress gateway.
type IngressGatewayHostRef struct {
	// Name of the ingress gateway host
	Name Name `json:"name"`
	// Port of the ingress gateway host to select. Leave empty to select all ports of the host.
	// +optional
	Port *Port `json:"port,omitempty"`
}

// HttpIngressGatewayTrafficRouteRulePathTypeEnum Enum with underlying type: string
// +kubebuilder:validation:Enum=PREFIX
type HttpIngressGatewayTrafficRouteRulePathTypeEnum string

// Set of constants representing the allowable values for HttpIngressGatewayTrafficRouteRulePathTypeEnum
const (
	HttpIngressGatewayTrafficRouteRulePathTypePrefix HttpIngressGatewayTrafficRouteRulePathTypeEnum = "PREFIX"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Active",type="string",JSONPath=".status.conditions[2].status",description="status of ingress gateway route table"
//+kubebuilder:printcolumn:name="Configured",type="string",JSONPath=".status.conditions[1].status",description="status of ingress gateway route table configured",priority=1
//+kubebuilder:printcolumn:name="DependenciesActive",type="string",JSONPath=".status.conditions[0].status",description="status of ingress gateway route table dependencies",priority=1
//+kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.ingressGatewayRouteTableId",description="ocid of ingress gateway route table",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// IngressGatewayRouteTable is the Schema for the ingressgatewayroutetables API
type IngressGatewayRouteTable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IngressGatewayRouteTableSpec `json:"spec,omitempty"`
	Status ServiceMeshStatus            `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IngressGatewayRouteTableList contains a list of IngressGatewayRouteTable
type IngressGatewayRouteTableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IngressGatewayRouteTable `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IngressGatewayRouteTable{}, &IngressGatewayRouteTableList{})
}
