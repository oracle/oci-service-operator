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

// IngressGatewayDeploymentSpec defines the desired state of IngressGatewayDeployment
type IngressGatewayDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	IngressGateway RefOrId `json:"ingressGateway"`

	// +kubebuilder:validation:Required
	Deployment IngressDeployment `json:"deployment"`

	// +kubebuilder:validation:Required
	Ports []GatewayListener `json:"ports"`

	// +optional
	Service *IngressGatewayService `json:"service,omitempty"`

	// +optional
	Secrets []SecretReference `json:"secrets,omitempty"`
}

type GatewayListener struct {
	// Type of protocol used in resource
	Protocol corev1.Protocol `json:"protocol"`

	// Port in which resource is running
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Required

	Port *int32 `json:"port"`

	// ServicePort in which Service opens Port
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:Minimum=0
	// +optional

	ServicePort *int32 `json:"serviceport,omitempty"`

	// +optional

	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
}

type IngressGatewayService struct {

	// +kubebuilder:validation:Enum=LoadBalancer;NodePort;ClusterIP
	Type corev1.ServiceType `json:"type"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

type IngressDeployment struct {
	// +kubebuilder:validation:Required
	Autoscaling *Autoscaling `json:"autoscaling"`

	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`
}

// Reference to kubernetes secret containing tls certificates/trust-chains for ingress gateway
type SecretReference struct {
	// name of the secret, this secret should reside in the same namespace as the gateway
	SecretName string `json:"secretName"`
}

// Contains information about min and max replicas for Ingress Gateway Deployment Resource
type Autoscaling struct {
	// Minimum number of pods available for Ingress Gateway Deployment Resource
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Required
	MinPods int32 `json:"minPods"`

	// Maximum number of pods available for Ingress Gateway Deployment Resource
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Required
	MaxPods int32 `json:"maxPods"`

	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Active",type="string",JSONPath=".status.conditions[1].status",description="status of ingress gateway deployment"
//+kubebuilder:printcolumn:name="DependenciesActive",type="string",JSONPath=".status.conditions[0].status",description="status of ingress gateway deployment dependencies",priority=1
//+kubebuilder:printcolumn:name="IngressGatewayOcid",type="string",JSONPath=".status.ingressGatewayId",description="ocid of ingress gateway",priority=1
//+kubebuilder:printcolumn:name="IngressGatewayName",type="string",JSONPath=".status.ingressGatewayName",description="name of ingress gateway",priority=1
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// IngressGatewayDeployment is the Schema for the ingressgatewaydeployments API
type IngressGatewayDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IngressGatewayDeploymentSpec `json:"spec,omitempty"`
	Status ServiceMeshStatus            `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IngressGatewayDeploymentList contains a list of IngressGatewayDeployment
type IngressGatewayDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IngressGatewayDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IngressGatewayDeployment{}, &IngressGatewayDeploymentList{})
}
