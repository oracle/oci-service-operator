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

// VirtualServiceRouteTableSpec defines the desired state of VirtualServiceRouteTable
type VirtualServiceRouteTableSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	api.TagResources `json:",inline"`
	// +optional
	Name          *Name    `json:"name,omitempty"`
	CompartmentId api.OCID `json:"compartmentId"`
	// +optional
	Description    *Description `json:"description,omitempty"`
	VirtualService RefOrId      `json:"virtualService"`
	// +kubebuilder:validation:MinItems=1
	RouteRules []VirtualServiceTrafficRouteRule `json:"routeRules"`
	// +optional
	Priority *int `json:"priority,omitempty"`
}

// Rule for routing incoming Virtual Service traffic to a Virtual Deployment
type VirtualServiceTrafficRouteRule struct {
	// +optional
	HttpRoute *HttpVirtualServiceTrafficRouteRule `json:"httpRoute,omitempty"`
	// +optional
	TcpRoute *TcpVirtualServiceTrafficRouteRule `json:"tcpRoute,omitempty"`
	// +optional
	TlsPassthroughRoute *TlsPassthroughVirtualServiceTrafficRouteRule `json:"tlsPassthroughRoute,omitempty"`
}

// HttpVirtualServiceTrafficRouteRule Rule for routing incoming Virtual Service traffic with HTTP protocol
type HttpVirtualServiceTrafficRouteRule struct {

	// The destination of the request.
	// +kubebuilder:validation:MinItems=1
	Destinations []VirtualDeploymentTrafficRuleTarget `json:"destinations"`

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
	PathType HttpVirtualServiceTrafficRouteRulePathTypeEnum `json:"pathType,omitempty"`
}

// Destination of a Virtual Deployment
type VirtualDeploymentTrafficRuleTarget struct {
	VirtualDeployment *RefOrId `json:"virtualDeployment"`
	// Amount of traffic flows to a specific Virtual Deployment
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Weight int `json:"weight"`
	// +optional
	Port *Port `json:"port,omitempty"`
}

// HttpVirtualServiceTrafficRouteRulePathTypeEnum Enum with underlying type: string
// +kubebuilder:validation:Enum=PREFIX
type HttpVirtualServiceTrafficRouteRulePathTypeEnum string

// Set of constants representing the allowable values for HttpVirtualServiceTrafficRouteRulePathTypeEnum
const (
	HttpVirtualServiceTrafficRouteRulePathTypePrefix HttpVirtualServiceTrafficRouteRulePathTypeEnum = "PREFIX"
)

type TcpVirtualServiceTrafficRouteRule struct {
	// The destination of the request.
	// +kubebuilder:validation:MinItems=1
	Destinations []VirtualDeploymentTrafficRuleTarget `json:"destinations"`
}

type TlsPassthroughVirtualServiceTrafficRouteRule struct {
	// The destination of the request.
	// +kubebuilder:validation:MinItems=1
	Destinations []VirtualDeploymentTrafficRuleTarget `json:"destinations"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Active",type="string",JSONPath=".status.conditions[2].status",description="status of virtual service route table"
//+kubebuilder:printcolumn:name="Configured",type="string",JSONPath=".status.conditions[1].status",description="status of virtual service route table configured",priority=1
//+kubebuilder:printcolumn:name="DependenciesActive",type="string",JSONPath=".status.conditions[0].status",description="status of virtual service route table dependencies",priority=1
//+kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.virtualServiceRouteTableId",description="ocid of virtual service route table",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// VirtualServiceRouteTable is the Schema for the virtualserviceroutetables API
type VirtualServiceRouteTable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualServiceRouteTableSpec `json:"spec,omitempty"`
	Status ServiceMeshStatus            `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VirtualServiceRouteTableList contains a list of VirtualServiceRouteTable
type VirtualServiceRouteTableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualServiceRouteTable `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualServiceRouteTable{}, &VirtualServiceRouteTableList{})
}
