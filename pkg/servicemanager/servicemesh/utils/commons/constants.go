/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package commons

import (
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

const (
	// Lifecycle states
	Active   = "ACTIVE"
	Failed   = "FAILED"
	Deleted  = "DELETED"
	Creating = "CREATING"
	Updating = "UPDATING"
	Deleting = "DELETING"

	Source      = "source"
	Destination = "destination"

	UnknownStatus = "unknown status"

	// Finalizers
	MeshFinalizer                     = "finalizers.servicemesh.oci.oracle.com/mesh-resources"
	VirtualServiceFinalizer           = "finalizers.servicemesh.oci.oracle.com/virtualservice-resources"
	VirtualDeploymentFinalizer        = "finalizers.servicemesh.oci.oracle.com/virtualdeployment-resources"
	VirtualServiceRouteTableFinalizer = "finalizers.servicemesh.oci.oracle.com/virtualserviceroutetable-resources"
	AccessPolicyFinalizer             = "finalizers.servicemesh.oci.oracle.com/accesspolicy-resources"
	IngressGatewayFinalizer           = "finalizers.servicemesh.oci.oracle.com/ingressgateway-resources"
	VirtualDeploymentBindingFinalizer = "finalizers.servicemesh.oci.oracle.com/virtualdeploymentbinding-resources"
	IngressGatewayRouteTableFinalizer = "finalizers.servicemesh.oci.oracle.com/ingressgatewayroutetable-resources"
	IngressGatewayDeploymentFinalizer = "finalizers.servicemesh.oci.oracle.com/ingressgatewaydeployment-resources"

	// Inject container names
	InitContainerName  = "init"
	ProxyContainerName = "oci-sm-proxy"

	// Net Admin Capability
	NetAdminCapability = "NET_ADMIN"

	OsokNamespace              = "oci-service-operator-system"
	ProxyLabelInMeshConfigMap  = "SIDECAR_IMAGE"
	CpEndpointInMeshConfigMap  = "CP_ENDPOINT"
	MdsEndpointInMeshConfigMap = "MDS_ENDPOINT"
	AutoUpdateProxyVersion     = "AUTO_UPDATE_PROXY_VERSION"

	ProxyInjectionLabel     = "servicemesh.oci.oracle.com/sidecar-injection"
	ProxyLogLevelAnnotation = "servicemesh.oci.oracle.com/proxy-log-level"
	DefaultProxyLogLevel    = ProxyLogLevelError
	Enabled                 = "enabled"
	Disabled                = "disabled"
	MeshConfigMapName       = "oci-service-operator-servicemesh-config"
	GlobalConfigMap         = OsokNamespace + "/" + MeshConfigMapName

	// Pod annotation keys
	OutdatedProxyAnnotation            = "servicemesh.oci.oracle.com/outdated-proxy"
	VirtualDeploymentBindingAnnotation = "servicemesh.oci.oracle.com/virtual-deployment-binding-ref"
	VirtualDeploymentAnnotation        = "servicemesh.oci.oracle.com/virtual-deployment-ocid"

	// IngressName Label
	IngressName = "servicemesh.oci.oracle.com/ingress-gateway-deployment"

	TargetCPUUtilizationPercentage = 50

	MetadataNameMaxLength = 190

	DeploymentAPIVersion = "apps/v1"

	Http = "http"
	Tcp  = "tcp"
)

type SidecarResourceRequirements string

const (
	//TODO: This is a placeholder and it should be replaced with validated cpu usage after testing.
	SidecarCPURequestSize SidecarResourceRequirements = "100m"
	//TODO: This is a placeholder and it should be replaced with validated memory usage after testing.
	SidecarMemoryRequestSize SidecarResourceRequirements = "128Mi"
	//TODO: This is a placeholder and it should be replaced with validated cpu usage after testing.
	SidecarCPULimitSize SidecarResourceRequirements = "2000m"
	//TODO: This is a placeholder and it should be replaced with validated memory usage after testing.
	SidecarMemoryLimitSize SidecarResourceRequirements = "1024Mi"
)

type IngressResourceRequirements string

const (
	//TODO: This is a placeholder and it should be replaced with validated cpu usage after testing.
	IngressCPURequestSize IngressResourceRequirements = "100m"
	//TODO: This is a placeholder and it should be replaced with validated memory usage after testing.
	IngressMemoryRequestSize IngressResourceRequirements = "128Mi"
	//TODO: This is a placeholder and it should be replaced with validated cpu usage after testing.
	IngressCPULimitSize IngressResourceRequirements = "2000m"
	//TODO: This is a placeholder and it should be replaced with validated memory usage after testing.
	IngressMemoryLimitSize IngressResourceRequirements = "1024Mi"
)

type Suffix string

const (
	NativeHorizontalPodAutoScalar Suffix = "-scalar"
	NativeDeployment              Suffix = "-deployment"
	NativeService                 Suffix = "-service"
)

type ResourceConditionReason string

const (
	DependenciesNotResolved ResourceConditionReason = "DependenciesNotResolved"
	LifecycleStateChanged   ResourceConditionReason = "LifecycleStateChanged"
	Successful              ResourceConditionReason = "Successful"
	ConnectionError         ResourceConditionReason = "ConnectionError"
)

type ResourceConditionMessage string

const (
	ResourceActive            ResourceConditionMessage = "Resource in the control plane is Active, successfully reconciled"
	ResourceDeleted           ResourceConditionMessage = "Resource in the control plane is Deleted"
	ResourceFailed            ResourceConditionMessage = "Resource in the control plane is Failed"
	ResourceCreating          ResourceConditionMessage = "Resource in the control plane is Creating, about to reconcile"
	ResourceUpdating          ResourceConditionMessage = "Resource in the control plane is Updating, about to reconcile"
	ResourceDeleting          ResourceConditionMessage = "Resource in the control plane is Deleting, about to reconcile"
	DependenciesResolved      ResourceConditionMessage = "Dependencies resolved successfully"
	ResourceConfigured        ResourceConditionMessage = "Resource configured successfully"
	ResourceChangeCompartment ResourceConditionMessage = "Changing Compartment of the resource and verifying updates"
)

type ResourceConditionMessageVDB string

const (
	ResourceActiveVDB   ResourceConditionMessageVDB = "The associated virtual deployment is Active, successfully reconciled"
	ResourceDeletedVDB  ResourceConditionMessageVDB = "The associated virtual deployment in the control plane is Deleted"
	ResourceFailedVDB   ResourceConditionMessageVDB = "The associated virtual deployment in the control plane is Failed"
	ResourceCreatingVDB ResourceConditionMessageVDB = "The associated virtual deployment in the control plane is Creating, about to reconcile"
	ResourceUpdatingVDB ResourceConditionMessageVDB = "The associated virtual deployment in the control plane is Updating, about to reconcile"
	ResourceDeletingVDB ResourceConditionMessageVDB = "The associated virtual deployment in the control plane is Deleting, about to reconcile"
)

type ValidationWebhookError string

const (
	UnknownStatusOnDelete                              ValidationWebhookError = "delete cannot be applied as the status is unknown"
	NotActiveOnUpdate                                  ValidationWebhookError = "update cannot be applied as the state is not Active"
	DependenciesIsUnknownOnUpdate                      ValidationWebhookError = "update cannot be applied as at least one dependency status is unknown"
	UnknownStateOnUpdate                               ValidationWebhookError = "update cannot be applied as the state in the mesh Control Plane is unknown"
	CertificateAuthoritiesIsImmutable                  ValidationWebhookError = "spec.certificateAuthorities is immutable"
	NameIsImmutable                                    ValidationWebhookError = "spec.name is immutable"
	MetadataNameLengthExceeded                         ValidationWebhookError = "metadata.name length should not exceed 190 characters"
	TrafficRouteRuleIsEmpty                            ValidationWebhookError = "spec.routeRule cannot be empty, should contain one of httpRoute,tcpRoute or tlsPassthroughRoute"
	TrafficRouteRuleIsNotUnique                        ValidationWebhookError = "spec.routeRule cannot contain more than one type"
	MeshReferenceIsImmutable                           ValidationWebhookError = "spec.mesh is immutable"
	MeshReferenceIsEmpty                               ValidationWebhookError = "spec.mesh cannot be empty, should contain one of ref or id"
	MeshReferenceIsNotUnique                           ValidationWebhookError = "spec.mesh cannot contain both ref and id"
	MeshReferenceIsDeleting                            ValidationWebhookError = "spec.mesh is being deleted"
	MeshReferenceNotFound                              ValidationWebhookError = "spec.mesh has been deleted or does not exist"
	IngressGatewayReferenceIsImmutable                 ValidationWebhookError = "spec.ingressGateway is immutable"
	IngressGatewayReferenceIsEmpty                     ValidationWebhookError = "spec.ingressGateway cannot be empty, should contain one of ref or id"
	IngressGatewayReferenceIsNotUnique                 ValidationWebhookError = "spec.ingressGateway cannot contain both ref and id"
	IngressGatewayReferenceIsDeleting                  ValidationWebhookError = "spec.ingressGateway is being deleted"
	IngressGatewayReferenceNotFound                    ValidationWebhookError = "spec.ingressGateway has been deleted or does not exist"
	VirtualServiceReferenceIsImmutable                 ValidationWebhookError = "spec.virtualService is immutable"
	VirtualServiceReferenceIsEmpty                     ValidationWebhookError = "spec.virtualService cannot be empty, should contain one of ref or id"
	VirtualServiceReferenceIsNotUnique                 ValidationWebhookError = "spec.virtualService cannot contain both ref and id"
	VirtualServiceReferenceIsDeleting                  ValidationWebhookError = "spec.virtualService is being deleted"
	VirtualServiceReferenceNotFound                    ValidationWebhookError = "spec.virtualService has been deleted or does not exist"
	VirtualDeploymentReferenceIsEmpty                  ValidationWebhookError = "spec.virtualDeployment cannot be empty, should contain one of ref or id"
	VirtualDeploymentReferenceIsNotUnique              ValidationWebhookError = "spec.virtualDeployment cannot contain both ref and id"
	VirtualDeploymentReferenceIsDeleting               ValidationWebhookError = "spec.virtualDeployment is being deleted"
	VirtualDeploymentReferenceNotFound                 ValidationWebhookError = "spec.virtualDeployment has been deleted or does not exist"
	KubernetesServiceReferenceIsDeleting               ValidationWebhookError = "spec.service is being deleted"
	KubernetesServiceReferenceNotFound                 ValidationWebhookError = "spec.service has been deleted or does not exist"
	HostNameIsEmptyForDNS                              ValidationWebhookError = "hostname cannot be empty when service discovery type is DNS"
	IngressGatewayDeploymentPortsWithMultipleProtocols ValidationWebhookError = "ingressgatewaydeployment.spec cannot have multiple protocols."
	IngressGatewayDeploymentWithMultiplePortEmptyName  ValidationWebhookError = "ingressgatewaydeployment.spec.ports.name is required when multiple ports are specified"
	IngressGatewayDeploymentPortsWithNonUniqueNames    ValidationWebhookError = "ingressgatewaydeployment.spec.ports.name must be unique"
	IngressGatewayDeploymentInvalidMaxPod              ValidationWebhookError = "spec.deployment.autoscaling maxPods cannot be less than minPods."
	IngressGatewayDeploymentRedundantServicePorts      ValidationWebhookError = "ingressgatewaydeployment.spec has target ports without service "
	VirtualServiceMtlsNotSatisfied                     ValidationWebhookError = "virtualservice mtls mode does not meet the minimum level set on parent mesh"
	MeshMtlsNotSatisfied                               ValidationWebhookError = "mtls mode of dependent virtual services does not meet the minimum level being set on mesh"
)

type InformerCacheType string

const (
	ConfigMapsCache InformerCacheType = "ConfigMaps"
	NamespacesCache InformerCacheType = "Namespaces"
	ServicesCache   InformerCacheType = "Services"
)

type InitContainerEnvVars string

const (
	ConfigureIpTablesEnvName  InitContainerEnvVars = "CONFIGURE_IP_TABLES"
	ConfigureIpTablesEnvValue InitContainerEnvVars = "true"
	EnvoyPortEnvVarName       InitContainerEnvVars = "ENVOY_PORT"
	EnvoyPortEnvVarValue      InitContainerEnvVars = "15000"
)

type ProxyEnvVars string

const (
	DeploymentId  ProxyEnvVars = "DEPLOYMENT_ID"
	ProxyLogLevel ProxyEnvVars = "PROXY_LOG_LEVEL"
	IPAddress     ProxyEnvVars = "IP_ADDRESS"
	StatsPort     int32        = 15006
)

type ProxyLogLevelType string

const (
	ProxyLogLevelDebug ProxyLogLevelType = "debug"
	ProxyLogLevelInfo  ProxyLogLevelType = "info"
	ProxyLogLevelWarn  ProxyLogLevelType = "warn"
	ProxyLogLevelError ProxyLogLevelType = "error"
	ProxyLogsOff       ProxyLogLevelType = "off"
)

var ProxyLogLevels = []string{string(ProxyLogLevelInfo), string(ProxyLogLevelDebug), string(ProxyLogLevelWarn), string(ProxyLogsOff), string(ProxyLogLevelError)}

type MeshUserHeader string

const (
	MeshUserScheme MeshUserHeader = "Mesh-User-Scheme"
	MeshUserHost   MeshUserHeader = "Mesh-User-Host"
	MeshUserPath   MeshUserHeader = "Mesh-User-Path"
	MeshUserPort   MeshUserHeader = "Mesh-User-Port"

	HealthProxyEndpointPath string = "/healthproxy"

	LivenessProbeEndpointPath        string = "/health"
	LivenessProbeEndpointPort        int32  = 15010
	LivenessProbeInitialDelaySeconds int32  = 5
	LocalHost                               = "localhost"
)

type PodWebhookError string

const (
	InValidProxyLogAnnotation PodWebhookError = "Invalid proxy Log level"
	NoSidecarImageFound       PodWebhookError = "No sidecar image found in config map"
)

var MtlsLevel = map[servicemeshapi.MutualTransportLayerSecurityModeEnum]int{
	servicemeshapi.MutualTransportLayerSecurityModeDisabled:   0,
	servicemeshapi.MutualTransportLayerSecurityModePermissive: 1,
	servicemeshapi.MutualTransportLayerSecurityModeStrict:     2,
}

type MeshResources string

const (
	Mesh                     MeshResources = "Mesh"
	VirtualService           MeshResources = "VirtualService"
	VirtualDeployment        MeshResources = "VirtualDeployment"
	VirtualServiceRouteTable MeshResources = "VirtualServiceRouteTable"
	IngressGateway           MeshResources = "IngressGateway"
	IngressGatewayRouteTable MeshResources = "IngressGatewayRouteTable"
	AccessPolicy             MeshResources = "AccessPolicy"
	IngressGatewayDeployment MeshResources = "IngressGatewayDeployment"
	VirtualDeploymentBinding MeshResources = "VirtualDeploymentBinding"
)
