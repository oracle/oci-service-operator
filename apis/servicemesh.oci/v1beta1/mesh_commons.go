/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/oracle/oci-service-operator/api/v1beta1"
)

// Name of the resource
// +kubebuilder:validation:MaxLength=255
// +kubebuilder:validation:MinLength=1
type Name string

// Description of the resource
// +kubebuilder:validation:MaxLength=400
type Description string

// +kubebuilder:validation:Maximum=65535
// +kubebuilder:validation:Minimum=1
type Port int32

// +kubebuilder:validation:Enum=ServiceMeshActive;ServiceMeshDependenciesActive;ServiceMeshConfigured
type ServiceMeshConditionType string

const (
	// Indicates whether the service mesh resource is active in the control-plane
	ServiceMeshActive ServiceMeshConditionType = "ServiceMeshActive"
	// Indicates whether the service mesh resource dependencies are in the Active state
	ServiceMeshDependenciesActive ServiceMeshConditionType = "ServiceMeshDependenciesActive"
	// Indicates whether the service mesh resource is configured in the mesh Control Plane
	ServiceMeshConfigured ServiceMeshConditionType = "ServiceMeshConfigured"
)

// Indicates the condition of the Service mesh resource
type ServiceMeshCondition struct {
	// Type of Service mesh condition.
	// +required
	// +kubebuilder:validation:Required
	Type              ServiceMeshConditionType `json:"type"`
	ResourceCondition `json:",inline"`
}

type ServiceMeshStatus struct {
	MeshId                      v1beta1.OCID                                `json:"meshId,omitempty"`
	VirtualServiceId            v1beta1.OCID                                `json:"virtualServiceId,omitempty"`
	VirtualServiceName          Name                                        `json:"virtualServiceName,omitempty"`
	VirtualDeploymentId         v1beta1.OCID                                `json:"virtualDeploymentId,omitempty"`
	VirtualDeploymentName       Name                                        `json:"virtualDeploymentName,omitempty"`
	VirtualServiceRouteTableId  v1beta1.OCID                                `json:"virtualServiceRouteTableId,omitempty"`
	AccessPolicyId              v1beta1.OCID                                `json:"accessPolicyId,omitempty"`
	RefIdForRules               []map[string]v1beta1.OCID                   `json:"refIdForRules,omitempty"`
	IngressGatewayId            v1beta1.OCID                                `json:"ingressGatewayId,omitempty"`
	IngressGatewayName          Name                                        `json:"ingressGatewayName,omitempty"`
	IngressGatewayRouteTableId  v1beta1.OCID                                `json:"ingressGatewayRouteTableId,omitempty"`
	VirtualServiceIdForRules    [][]v1beta1.OCID                            `json:"virtualServiceIdForRules,omitempty"`
	VirtualDeploymentIdForRules [][]v1beta1.OCID                            `json:"virtualDeploymentIdForRules,omitempty"`
	MeshMtls                    *MeshMutualTransportLayerSecurity           `json:"meshMtls,omitempty"`
	VirtualServiceMtls          *VirtualServiceMutualTransportLayerSecurity `json:"virtualServiceMtls,omitempty"`
	IngressGatewayMtls          *IngressGatewayMutualTransportLayerSecurity `json:"ingressGatewayMtls,omitempty"`
	Conditions                  []ServiceMeshCondition                      `json:"conditions,omitempty"`
	OpcRetryToken               *string                                     `json:"opcRetryToken,omitempty"`
	LastUpdatedTime             *metav1.Time                                `json:"lastUpdatedTime,omitempty"`
}

// AllVirtualServices represents all virtual services
type AllVirtualServices struct {
}

// ExternalService represents anything running outside of the mesh.
// Only 1 of the following fields can be specified.
type ExternalService struct {
	// +optional
	TcpExternalService *TcpExternalService `json:"tcpExternalService,omitempty"`
	// +optional
	HttpExternalService *HttpExternalService `json:"httpExternalService,omitempty"`
	// +optional
	HttpsExternalService *HttpsExternalService `json:"httpsExternalService,omitempty"`
}

type TcpExternalService struct {
	// IpAddresses of the external service in CIDR notation.
	// +kubebuilder:validation:MaxItems=10
	// +kubebuilder:validation:MinItems=1
	IpAddresses []string `json:"ipAddresses"`
	// Ports exposed by the external service. If left empty all ports will be allowed.
	// +kubebuilder:validation:MaxItems=10
	// +optional
	Ports []Port `json:"ports,omitempty"`
}

type HttpExternalService struct {
	// Host names of the external service.
	// +kubebuilder:validation:MaxItems=10
	// +kubebuilder:validation:MinItems=1
	Hostnames []string `json:"hostnames"`
	// Ports exposed by the external service. If left empty all ports will be allowed.
	// +kubebuilder:validation:MaxItems=10
	// +optional
	Ports []Port `json:"ports,omitempty"`
}

type HttpsExternalService struct {
	// Host names of the external service.
	// +kubebuilder:validation:MaxItems=10
	// +kubebuilder:validation:MinItems=1
	Hostnames []string `json:"hostnames"`
	// Ports exposed by the external service. If left empty all ports will be allowed.
	// +kubebuilder:validation:MaxItems=10
	// +optional
	Ports []Port `json:"ports,omitempty"`
}

// Describes the target of the traffic.
// This can either be the source or the destination of the traffic.
// The source of traffic can be defined by specifying only one of the "allVirtualServices", "virtualService" or "ingressGateway" property.
// The destination of the traffic can be defined by specifying only one of the "allVirtualServices", "virtualService" or "externalService" property.
type TrafficTarget struct {
	// +optional
	AllVirtualServices *AllVirtualServices `json:"allVirtualServices,omitempty"`
	// +optional
	VirtualService *RefOrId `json:"virtualService,omitempty"`
	// +optional
	ExternalService *ExternalService `json:"externalService,omitempty"`
	// +optional
	IngressGateway *RefOrId `json:"ingressGateway,omitempty"`
}

// Describes the Mutual Transport Layer Security mode for a resource
// Only one of the following mode should be present
// +kubebuilder:validation:Enum=DISABLED;PERMISSIVE;STRICT
type MutualTransportLayerSecurityModeEnum string

const (
	MutualTransportLayerSecurityModeDisabled   MutualTransportLayerSecurityModeEnum = "DISABLED"
	MutualTransportLayerSecurityModePermissive MutualTransportLayerSecurityModeEnum = "PERMISSIVE"
	MutualTransportLayerSecurityModeStrict     MutualTransportLayerSecurityModeEnum = "STRICT"
)

// Listener configuration for a resource
type Listener struct {
	// Type of protocol used in resource
	Protocol ProtocolType `json:"protocol"`
	// Port in which resource is running
	Port Port `json:"port"`
	// The maximum duration in milliseconds for the deployed service to respond to an incoming request through the listener.
	// If provided, the timeout value overrides the default timeout of 15 seconds for the HTTP/HTTP2 listeners, and disabled (no timeout) for the GRPC listeners. The value 0 (zero) indicates that the timeout is disabled.
	// The timeout cannot be configured for the TCP and TLS_PASSTHROUGH listeners.
	// For streaming responses from the deployed service, consider either keeping the timeout disabled or set a sufficiently high value.
	// +optional
	// +kubebuilder:validation:Minimum=0
	RequestTimeoutInMs *int64 `json:"requestTimeoutInMs,omitempty"`
	// The maximum duration in milliseconds for which the request's stream may be idle. The value 0 (zero) indicates that the timeout is disabled.
	// +optional
	// +kubebuilder:validation:Minimum=0
	IdleTimeoutInMs *int64 `json:"idleTimeoutInMs,omitempty"`
}

// Describes the Protocol type for a resource
// Only one of the following protocol type should be present
// +kubebuilder:validation:Enum=HTTP;HTTP2;GRPC;TCP;TLS_PASSTHROUGH
type ProtocolType string

const (
	ProtocolTypeHttp           ProtocolType = "HTTP"
	ProtocolTypeHttp2          ProtocolType = "HTTP2"
	ProtocolTypeGrpc           ProtocolType = "GRPC"
	ProtocolTypeTcp            ProtocolType = "TCP"
	ProtocolTypeTlsPassthrough ProtocolType = "TLS_PASSTHROUGH"
)

// CaBundle Resource representing the CA bundle
type CaBundle struct {
	// +optional
	OciCaBundle *OciCaBundle `json:"ociCaBundle,omitempty"`
	// +optional
	KubeSecretCaBundle *KubeSecretCaBundle `json:"kubeSecretCaBundle,omitempty"`
}

// OciCaBundle CA Bundle from OCI Certificates service
type OciCaBundle struct {
	// The OCID of the CA Bundle resource.
	CaBundleId v1beta1.OCID `json:"caBundleId"`
}

// KubeSecretCaBundle CA Bundle from kubernetes secrets
type KubeSecretCaBundle struct {
	// The name of the kubernetes secret for CA Bundle resource.
	SecretName string `json:"secretName"`
}

// TlsCertificate Resource representing the location of the TLS certificate
type TlsCertificate struct {
	// +optional
	OciTlsCertificate *OciTlsCertificate `json:"ociTlsCertificate,omitempty"`
	// +optional
	KubeSecretTlsCertificate *KubeSecretTlsCertificate `json:"kubeSecretTlsCertificate,omitempty"`
}

// OciTlsCertificate TLS certificate from OCI Certificates service
type OciTlsCertificate struct {
	// The OCID of the leaf certificate resource.
	CertificateId v1beta1.OCID `json:"certificateId"`
}

// KubeSecretTlsCertificate TLS certificate from kubernetes secrets
type KubeSecretTlsCertificate struct {
	// The name of the leaf certificate kubernetes secret.
	SecretName string `json:"secretName"`
}

// The base Resource condition
type ResourceCondition struct {
	// status of the condition, one of True, False, Unknown.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=True;False;Unknown
	Status metav1.ConditionStatus `json:"status"`
	// observedGeneration represents the .metadata.generation that the condition was set based upon.
	// For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
	// with respect to the current state of the instance.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// lastTransitionTime is the last time the condition transitioned from one status to another.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Format=date-time
	LastTransitionTime *metav1.Time `json:"lastTransitionTime"`
	// reason contains a programmatic identifier indicating the reason for the condition's last transition.
	// The value should be a CamelCase string.
	// This field may not be empty.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[A-Za-z]([A-Za-z0-9_,:]*[A-Za-z0-9_])?$`
	Reason string `json:"reason"`
	// message is a human readable message indicating details about the transition.
	// This may be an empty string.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=32768
	Message string `json:"message"`
}

// AccessLogging information
type AccessLogging struct {
	// Checks whether access path is enabled
	// +optional
	IsEnabled bool `json:"isEnabled,omitempty"`
}

// Type representing the reference to a CR
type ResourceRef struct {
	// Name of the referenced CR
	Name Name `json:"name"`
	// Namespace of the referenced CR
	// If unspecified, defaults to the referencing object's namespace
	// +optional
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace,omitempty"`
}

// A reference to CR that a resource belongs to
type RefOrId struct {
	// +optional
	*ResourceRef `json:"ref,omitempty"`
	// +optional
	Id v1beta1.OCID `json:"id,omitempty"`
}
