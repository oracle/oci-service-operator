/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functional

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"

	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
)

var (
	path        = "/foo"
	grpcEnabled = true
)

var (
	apDescription   = servicemeshapi.Description("This is Access Policy")
	igDescription   = servicemeshapi.Description("This is Ingress Gateway")
	igrtDescription = servicemeshapi.Description("This is Ingress Gateway Route Table")
	meshDescription = servicemeshapi.Description("This is Mesh")
	vdDescription   = servicemeshapi.Description("This is Virtual Deployment")
	vsDescription   = servicemeshapi.Description("This is Virtual Service")
	vsrtDescription = servicemeshapi.Description("This is Virtual Service Route Table")
)

func GetSdkMesh(state sdk.MeshLifecycleStateEnum) *sdk.Mesh {
	return &sdk.Mesh{
		DisplayName:   conversions.String("my-mesh"),
		Id:            conversions.String("my-mesh-id"),
		CompartmentId: conversions.String("compartment-id"),
		Description:   conversions.String("This is Mesh"),
		Mtls:          &sdk.MeshMutualTransportLayerSecurity{Minimum: sdk.MutualTransportLayerSecurityModeDisabled},
		CertificateAuthorities: []sdk.CertificateAuthority{
			{
				Id: conversions.String("certificate-authority-id"),
			},
		},
		FreeformTags: map[string]string{"freeformTag": "value"},
		DefinedTags: map[string]map[string]interface{}{
			"definedTag": {"foo": "bar"},
		},
		LifecycleState: state,
	}
}

func GetApiMesh() *servicemeshapi.Mesh {
	return &servicemeshapi.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-mesh",
			Namespace: "test-namespace",
		},
		Spec: servicemeshapi.MeshSpec{
			DisplayName:   conversions.ApiName("my-mesh"),
			CompartmentId: "compartment-id",
			Description:   &meshDescription,
			CertificateAuthorities: []servicemeshapi.CertificateAuthority{
				{
					Id: "certificate-authority-id",
				},
			},
			Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
				Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
			},
			TagResources: api.TagResources{
				FreeFormTags: map[string]string{"freeformTag": "value"},
				DefinedTags: map[string]api.MapValue{
					"definedTag": {"foo": "bar"},
				},
			},
		},
	}
}

func GetSdkVirtualService(state sdk.VirtualServiceLifecycleStateEnum) *sdk.VirtualService {
	return &sdk.VirtualService{
		Name:          conversions.String("my-virtualservice"),
		Id:            conversions.String("my-virtualservice-id"),
		CompartmentId: conversions.String("compartment-id"),
		Description:   conversions.String("This is Virtual Service"),
		MeshId:        conversions.String("my-mesh-id"),
		Mtls: &sdk.MutualTransportLayerSecurity{
			CertificateId: conversions.String("certificate-authority-id"),
			Mode:          sdk.MutualTransportLayerSecurityModeDisabled,
		},
		FreeformTags: map[string]string{"freeformTag": "value"},
		DefinedTags: map[string]map[string]interface{}{
			"definedTag": {"foo": "bar"},
		},
		LifecycleState: state,
	}
}

func GetApiVirtualService() *servicemeshapi.VirtualService {
	return &servicemeshapi.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-virtualservice",
			Namespace: "test-namespace",
		},
		Spec: servicemeshapi.VirtualServiceSpec{
			Name:          conversions.ApiName("my-virtualservice"),
			CompartmentId: "compartment-id",
			Description:   &vsDescription,
			Mesh: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "test-namespace",
					Name:      "my-mesh",
				},
			},
			Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{
				Mode: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
			},
			Hosts: []string{"my-host"},
			TagResources: api.TagResources{
				FreeFormTags: map[string]string{},
				DefinedTags:  map[string]api.MapValue{},
			},
		},
	}
}

func GetSdkVirtualDeployment(state sdk.VirtualDeploymentLifecycleStateEnum) *sdk.VirtualDeployment {
	return &sdk.VirtualDeployment{
		Name:             conversions.String("my-virtualdeployment"),
		Id:               conversions.String("my-virtualdeployment-id"),
		CompartmentId:    conversions.String("compartment-id"),
		Description:      conversions.String("This is Virtual Deployment"),
		VirtualServiceId: conversions.String("my-virtualservice-id"),
		FreeformTags:     map[string]string{"freeformTag": "value"},
		DefinedTags: map[string]map[string]interface{}{
			"definedTag": {"foo": "bar"},
		},
		Listeners: []sdk.VirtualDeploymentListener{
			{
				Protocol: sdk.VirtualDeploymentListenerProtocolHttp,
				Port:     conversions.Integer(8080),
			},
		},
		AccessLogging: &sdk.AccessLoggingConfiguration{
			IsEnabled: new(bool),
		},
		ServiceDiscovery: sdk.DnsServiceDiscoveryConfiguration{
			Hostname: conversions.String("oracle.com"),
		},
		LifecycleState: state,
	}
}

func GetApiVirtualDeployment() *servicemeshapi.VirtualDeployment {
	return &servicemeshapi.VirtualDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-virtualdeployment",
			Namespace: "test-namespace",
		},
		Spec: servicemeshapi.VirtualDeploymentSpec{
			Name:          conversions.ApiName("my-virtualdeployment"),
			CompartmentId: "compartment-id",
			Description:   &vdDescription,
			VirtualService: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name:      "my-virtualservice",
					Namespace: "test-namespace",
				},
			},
			Listener: []servicemeshapi.Listener{
				{
					Protocol: "HTTP",
					Port:     80,
				},
			},
			AccessLogging: &servicemeshapi.AccessLogging{
				IsEnabled: true,
			},
			ServiceDiscovery: servicemeshapi.ServiceDiscovery{
				Type:     servicemeshapi.ServiceDiscoveryTypeDns,
				Hostname: "my-host",
			},
			TagResources: api.TagResources{
				FreeFormTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]api.MapValue{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
	}
}

func GetSdkVirtualServiceRouteTable(state sdk.VirtualServiceRouteTableLifecycleStateEnum) *sdk.VirtualServiceRouteTable {
	return &sdk.VirtualServiceRouteTable{
		Name:             conversions.String("my-virtualserviceroutetable"),
		Id:               conversions.String("my-virtualserviceroutetable-id"),
		CompartmentId:    conversions.String("compartment-id"),
		Description:      conversions.String("This is Virtual Service Route Table"),
		VirtualServiceId: conversions.String("my-virtualservice-id"),
		FreeformTags:     map[string]string{"freeformTag": "value"},
		DefinedTags: map[string]map[string]interface{}{
			"definedTag": {"foo": "bar"},
		},
		RouteRules: []sdk.VirtualServiceTrafficRouteRule{
			sdk.HttpVirtualServiceTrafficRouteRule{
				Destinations: []sdk.VirtualDeploymentTrafficRuleTarget{
					{
						VirtualDeploymentId: conversions.String("my-virtualdeployment-id"),
						Weight:              conversions.Integer(100),
					},
				},
				IsGrpc:   &grpcEnabled,
				PathType: sdk.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
				Path:     &path,
			},
		},
		LifecycleState: state,
	}
}

func GetApiVirtualServiceRouteTable() *servicemeshapi.VirtualServiceRouteTable {
	return &servicemeshapi.VirtualServiceRouteTable{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-virtualserviceroutetable",
			Namespace: "test-namespace",
		},
		Spec: servicemeshapi.VirtualServiceRouteTableSpec{
			Name:          conversions.ApiName("my-virtualserviceroutetable"),
			CompartmentId: "compartment-id",
			Description:   &vsrtDescription,
			VirtualService: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name:      "my-virtualservice",
					Namespace: "test-namespace",
				},
			},
			RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
				{
					HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
						Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
							{
								VirtualDeployment: &servicemeshapi.RefOrId{
									ResourceRef: &servicemeshapi.ResourceRef{
										Name:      "my-virtualdeployment",
										Namespace: "test-namespace",
									},
								},
								Weight: 50,
							},
						},
						Path:     &path,
						IsGrpc:   &grpcEnabled,
						PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
					},
				},
			},
			TagResources: api.TagResources{
				FreeFormTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]api.MapValue{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
	}
}

func GetSdkIngressGateway(state sdk.IngressGatewayLifecycleStateEnum) *sdk.IngressGateway {
	return &sdk.IngressGateway{
		Name:          conversions.String("my-ingressgateway"),
		Id:            conversions.String("my-ingressgateway-id"),
		Description:   conversions.String("This is Ingress Gateway"),
		CompartmentId: conversions.String("compartment-id"),
		MeshId:        conversions.String("my-mesh-id"),
		FreeformTags:  map[string]string{"freeformTag": "value"},
		DefinedTags: map[string]map[string]interface{}{
			"definedTag": {"foo": "bar"},
		},
		Hosts: []sdk.IngressGatewayHost{{
			Hostnames: []string{"test.com"},
			Listeners: []sdk.IngressGatewayListener{},
		}},
		Mtls: &sdk.IngressGatewayMutualTransportLayerSecurity{
			CertificateId: conversions.String("certificate-authority-id"),
		},
		AccessLogging: &sdk.AccessLoggingConfiguration{
			IsEnabled: new(bool),
		},
		LifecycleState: state,
	}
}

func GetApiIngressGateway() *servicemeshapi.IngressGateway {
	return &servicemeshapi.IngressGateway{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-ingressgateway",
			Namespace: "test-namespace",
		},
		Spec: servicemeshapi.IngressGatewaySpec{
			Name:          conversions.ApiName("my-ingressgateway"),
			CompartmentId: "compartment-id",
			Description:   &igDescription,
			Mesh: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "test-namespace",
					Name:      "my-mesh",
				},
			},
			Hosts: []servicemeshapi.IngressGatewayHost{{
				Name:      "testHost",
				Hostnames: []string{"test.com"},
				Listeners: []servicemeshapi.IngressGatewayListener{{
					Protocol: servicemeshapi.IngressGatewayListenerProtocolHttp,
					Port:     8080,
					Tls: &servicemeshapi.IngressListenerTlsConfig{
						Mode: servicemeshapi.IngressListenerTlsConfigModeTls,
						ServerCertificate: &servicemeshapi.TlsCertificate{
							OciTlsCertificate: &servicemeshapi.OciTlsCertificate{CertificateId: "serverCertId"},
						},
						ClientValidation: &servicemeshapi.IngressHostClientValidationConfig{
							TrustedCaBundle: &servicemeshapi.CaBundle{
								OciCaBundle: &servicemeshapi.OciCaBundle{CaBundleId: "ocid.caBundleId"},
							},
							SubjectAlternateNames: []string{"trusted.client"},
						},
					},
				}},
			}},
			AccessLogging: &servicemeshapi.AccessLogging{
				IsEnabled: true,
			},
			TagResources: api.TagResources{
				FreeFormTags: map[string]string{},
				DefinedTags:  map[string]api.MapValue{},
			},
		},
	}
}

func GetSdkIngressGatewayRouteTable(state sdk.IngressGatewayRouteTableLifecycleStateEnum) *sdk.IngressGatewayRouteTable {
	return &sdk.IngressGatewayRouteTable{
		Name:             conversions.String("my-ingressgatewayroutetable"),
		Id:               conversions.String("my-ingressgatewayroutetable-id"),
		CompartmentId:    conversions.String("compartment-id"),
		Description:      conversions.String("This is Ingress Gateway Route Table"),
		IngressGatewayId: conversions.String("my-ingressgateway-id"),
		FreeformTags:     map[string]string{"freeformTag": "value"},
		DefinedTags: map[string]map[string]interface{}{
			"definedTag": {"foo": "bar"},
		},
		RouteRules: []sdk.IngressGatewayTrafficRouteRule{
			sdk.HttpIngressGatewayTrafficRouteRule{
				Destinations: []sdk.VirtualServiceTrafficRuleTarget{
					{
						VirtualServiceId: conversions.String("my-virtualservice-id"),
						Weight:           conversions.Integer(100),
					},
				},
				IsGrpc:   &grpcEnabled,
				PathType: sdk.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
				Path:     &path,
				IngressGatewayHost: &sdk.IngressGatewayHostRef{
					Name: conversions.String("testHost"),
				},
			},
		},
		LifecycleState: state,
	}
}

func GetApiIngressGatewayRouteTable() *servicemeshapi.IngressGatewayRouteTable {
	return &servicemeshapi.IngressGatewayRouteTable{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-ingressgatewayroutetable",
			Namespace: "test-namespace",
		},
		Spec: servicemeshapi.IngressGatewayRouteTableSpec{
			Name:          conversions.ApiName("my-ingressgatewayroutetable"),
			CompartmentId: "compartment-id",
			Description:   &igrtDescription,
			IngressGateway: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name:      "my-ingressgateway",
					Namespace: "test-namespace",
				},
			},
			RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
				{
					HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
						Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
							{
								VirtualService: &servicemeshapi.RefOrId{
									ResourceRef: &servicemeshapi.ResourceRef{
										Name:      "my-virtualservice",
										Namespace: "test-namespace",
									},
								},
								Weight: conversions.Integer(50),
							},
						},
						IngressGatewayHost: &servicemeshapi.IngressGatewayHostRef{
							Name: "testHost",
						},
						Path:     &path,
						IsGrpc:   &grpcEnabled,
						PathType: servicemeshapi.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
					},
				},
			},
			TagResources: api.TagResources{
				FreeFormTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]api.MapValue{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
	}
}

func GetApiIngressGatewayDeployment() *servicemeshapi.IngressGatewayDeployment {
	return &servicemeshapi.IngressGatewayDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-ingressgatewaydeployment",
			Namespace: "test-namespace",
		},
		Spec: servicemeshapi.IngressGatewayDeploymentSpec{
			IngressGateway: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name:      "my-ingressgateway",
					Namespace: "test-namespace",
				},
			},
			Deployment: servicemeshapi.IngressDeployment{
				Autoscaling: &servicemeshapi.Autoscaling{
					MinPods: 3,
					MaxPods: 3,
				},
			},
			Ports: []servicemeshapi.GatewayListener{
				{
					Protocol:    corev1.ProtocolTCP,
					Port:        &[]int32{8080}[0],
					ServicePort: &[]int32{8080}[0],
				},
			},
			Service: &servicemeshapi.IngressGatewayService{
				Type: corev1.ServiceTypeLoadBalancer,
			},
			Secrets: []servicemeshapi.SecretReference{
				{SecretName: "bookinfo-tls-secret"},
				{SecretName: "bookinfo-tls-secret2"},
			},
		},
	}
}

func GetSdkAccessPolicy(state sdk.AccessPolicyLifecycleStateEnum) *sdk.AccessPolicy {
	return &sdk.AccessPolicy{
		Name:        conversions.String("my-accesspolicy"),
		Id:          conversions.String("my-accesspolicy-id"),
		MeshId:      conversions.String("my-mesh-id"),
		Description: conversions.String("This is Access Policy"),
		Rules: []sdk.AccessPolicyRule{
			{
				Action: sdk.AccessPolicyRuleActionAllow,
				Source: &sdk.AllVirtualServicesAccessPolicyTarget{},
				Destination: &sdk.VirtualServiceAccessPolicyTarget{
					VirtualServiceId: conversions.String("my-virtualservice-id"),
				},
			},
		},
		CompartmentId: conversions.String("compartment-id"),
		FreeformTags:  map[string]string{"freeformTag": "value"},
		DefinedTags: map[string]map[string]interface{}{
			"definedTag": {"foo": "bar"},
		},
		LifecycleState: state,
	}
}

func GetApiAccessPolicy() *servicemeshapi.AccessPolicy {
	return &servicemeshapi.AccessPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-accesspolicy",
			Namespace: "test-namespace",
		},
		Spec: servicemeshapi.AccessPolicySpec{
			Name:          conversions.ApiName("my-accesspolicy"),
			CompartmentId: api.OCID("compartment-id"),
			Description:   &apDescription,
			TagResources: api.TagResources{
				FreeFormTags: map[string]string{"freeformTag": "value"},
				DefinedTags: map[string]api.MapValue{
					"definedTag": {"foo": "bar"},
				},
			},
			Mesh: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name:      "my-mesh",
					Namespace: "test-namespace",
				},
			},
			Rules: []servicemeshapi.AccessPolicyRule{
				{
					Action: servicemeshapi.ActionTypeAllow,
					Source: servicemeshapi.TrafficTarget{
						VirtualService: &servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-virtualservice",
								Namespace: "test-namespace",
							},
						},
					},
					Destination: servicemeshapi.TrafficTarget{
						VirtualService: &servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-virtualservice",
								Namespace: "test-namespace",
							},
						},
					},
				},
			},
		},
	}
}

func GetApiVirtualDeploymentBinding() *servicemeshapi.VirtualDeploymentBinding {
	return &servicemeshapi.VirtualDeploymentBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-virtualdeploymentbinding",
			Namespace: "test-namespace",
		},
		Spec: servicemeshapi.VirtualDeploymentBindingSpec{
			VirtualDeployment: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name:      "my-virtualdeployment",
					Namespace: "test-namespace",
				},
			},
			Target: servicemeshapi.Target{
				Service: servicemeshapi.Service{
					ServiceRef: servicemeshapi.ResourceRef{
						Name:      "my-service",
						Namespace: "sidecar-inject-namespace",
					},
				},
			},
		},
	}
}

func GetService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-service",
			Namespace: "sidecar-inject-namespace",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 8080,
				},
			},
			Selector: map[string]string{
				"app": "my-service",
			},
		},
	}
}

func GetPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pod",
			Namespace: "sidecar-inject-namespace",
			Labels: map[string]string{
				commons.ProxyInjectionLabel: commons.Enabled,
				"app":                       "my-service",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "my-container",
					Image: "my-container-image",
				},
			},
		},
	}
}

func GetSidecarInjectNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sidecar-inject-namespace",
			Labels: map[string]string{
				commons.ProxyInjectionLabel: commons.Disabled,
			},
		},
	}
}

func GetConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      commons.MeshConfigMapName,
			Namespace: commons.OsokNamespace,
		},
		Data: map[string]string{
			commons.ProxyLabelInMeshConfigMap: "sm-proxy-image:0.0.1",
		},
	}
}
