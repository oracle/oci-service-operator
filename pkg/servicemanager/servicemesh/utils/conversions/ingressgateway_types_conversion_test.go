/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package conversions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

var (
	igName        = v1beta1.Name("my-ingressgateway")
	igDescription = v1beta1.Description("This is Ingress Gateway")
)

func Test_Convert_CRD_IngressGateway_To_SDK_IngressGateway(t *testing.T) {
	type args struct {
		crdObj *v1beta1.IngressGateway
		sdkObj *sdk.IngressGateway
		meshId api.OCID
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *sdk.IngressGateway
	}{
		{
			name: "ingressgateway with no spec name",
			args: args{
				crdObj: &v1beta1.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-ingressgateway",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewaySpec{
						CompartmentId: "my-compartment-id",
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Description: &igDescription,
						Hosts: []v1beta1.IngressGatewayHost{{
							Name:      "testHost",
							Hostnames: []string{"test.com"},
							Listeners: []v1beta1.IngressGatewayListener{
								{
									Protocol: v1beta1.IngressGatewayListenerProtocolHttp,
									Port:     443,
									Tls: &v1beta1.IngressListenerTlsConfig{
										Mode: v1beta1.IngressListenerTlsConfigModeTls,
										ServerCertificate: &v1beta1.TlsCertificate{
											OciTlsCertificate: &v1beta1.OciTlsCertificate{CertificateId: "ocid.server-cert-id"},
										},
									},
								},
							},
						}},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						IngressGatewayId: "my-ingressgateway-id",
					},
				},
				sdkObj: &sdk.IngressGateway{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.IngressGateway{
				Id:            String("my-ingressgateway-id"),
				CompartmentId: String("my-compartment-id"),
				MeshId:        String("my-mesh-id"),
				Description:   String("This is Ingress Gateway"),
				Name:          String("my-namespace/my-ingressgateway"),
				Hosts: []sdk.IngressGatewayHost{
					{
						Name:      String("testHost"),
						Hostnames: []string{"test.com"},
						Listeners: []sdk.IngressGatewayListener{
							{
								Protocol: sdk.IngressGatewayListenerProtocolHttp,
								Port:     Integer(443),
								Tls: &sdk.IngressListenerTlsConfig{
									Mode: sdk.IngressListenerTlsConfigModeTls,
									ServerCertificate: sdk.OciTlsCertificate{
										CertificateId: String("ocid.server-cert-id"),
									},
								},
							},
						},
					},
				},
				FreeformTags: map[string]string{},
				DefinedTags:  map[string]map[string]interface{}{},
			},
		},
		{
			name: "ingressgateway with no TLS configuration",
			args: args{
				crdObj: &v1beta1.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-ingressgateway",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewaySpec{
						CompartmentId: "my-compartment-id",
						Name:          &igName,
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []v1beta1.IngressGatewayHost{
							{
								Name:      "testHost",
								Hostnames: []string{"test.com"},
								Listeners: []v1beta1.IngressGatewayListener{{
									Protocol: v1beta1.IngressGatewayListenerProtocolHttp,
									Port:     8080,
								}}},
						},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						IngressGatewayId: "my-ingressgateway-id",
					},
				},
				sdkObj: &sdk.IngressGateway{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.IngressGateway{
				Id:            String("my-ingressgateway-id"),
				CompartmentId: String("my-compartment-id"),
				MeshId:        String("my-mesh-id"),
				Name:          String("my-ingressgateway"),
				Hosts: []sdk.IngressGatewayHost{
					{
						Name:      String("testHost"),
						Hostnames: []string{"test.com"},
						Listeners: []sdk.IngressGatewayListener{
							{
								Protocol: sdk.IngressGatewayListenerProtocolHttp,
								Port:     Integer(8080),
							},
						}},
				},
				FreeformTags: map[string]string{},
				DefinedTags:  map[string]map[string]interface{}{},
			},
		},
		{
			name: "ingressgateway with TlsConfiguration",
			args: args{
				crdObj: &v1beta1.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-ingressgateway",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewaySpec{
						CompartmentId: "my-compartment-id",
						Name:          &igName,
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []v1beta1.IngressGatewayHost{{
							Name:      "testHost",
							Hostnames: []string{"test.com"},
							Listeners: []v1beta1.IngressGatewayListener{
								{
									Protocol: v1beta1.IngressGatewayListenerProtocolHttp,
									Port:     443,
									Tls: &v1beta1.IngressListenerTlsConfig{
										Mode: v1beta1.IngressListenerTlsConfigModeTls,
										ServerCertificate: &v1beta1.TlsCertificate{
											OciTlsCertificate: &v1beta1.OciTlsCertificate{CertificateId: "ocid.server-cert-id"},
										},
									},
								},
							},
						}},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						IngressGatewayId: "my-ingressgateway-id",
					},
				},
				sdkObj: &sdk.IngressGateway{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.IngressGateway{
				Id:            String("my-ingressgateway-id"),
				CompartmentId: String("my-compartment-id"),
				MeshId:        String("my-mesh-id"),
				Name:          String("my-ingressgateway"),
				Hosts: []sdk.IngressGatewayHost{
					{
						Name:      String("testHost"),
						Hostnames: []string{"test.com"},
						Listeners: []sdk.IngressGatewayListener{
							{
								Protocol: sdk.IngressGatewayListenerProtocolHttp,
								Port:     Integer(443),
								Tls: &sdk.IngressListenerTlsConfig{
									Mode: sdk.IngressListenerTlsConfigModeTls,
									ServerCertificate: sdk.OciTlsCertificate{
										CertificateId: String("ocid.server-cert-id"),
									},
								},
							},
						},
					},
				},
				FreeformTags: map[string]string{},
				DefinedTags:  map[string]map[string]interface{}{},
			},
		},
		{
			name: "ingressgateway with client cert validation",
			args: args{
				crdObj: &v1beta1.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-ingressgateway",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewaySpec{
						CompartmentId: "my-compartment-id",
						Name:          &igName,
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []v1beta1.IngressGatewayHost{{
							Name:      "testHost",
							Hostnames: []string{"test.com"},
							Listeners: []v1beta1.IngressGatewayListener{
								{
									Protocol: v1beta1.IngressGatewayListenerProtocolHttp,
									Port:     443,
									Tls: &v1beta1.IngressListenerTlsConfig{
										Mode: v1beta1.IngressListenerTlsConfigModeMutualTls,
										ServerCertificate: &v1beta1.TlsCertificate{
											OciTlsCertificate: &v1beta1.OciTlsCertificate{CertificateId: "ocid.server-cert-id"},
										},
										ClientValidation: &v1beta1.IngressHostClientValidationConfig{
											TrustedCaBundle: &v1beta1.CaBundle{
												OciCaBundle: &v1beta1.OciCaBundle{CaBundleId: "ocid.trustedCaBundle"},
											},
											SubjectAlternateNames: []string{"trusted.client"},
										},
									},
								},
							},
						}},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						IngressGatewayId: "my-ingressgateway-id",
					},
				},
				sdkObj: &sdk.IngressGateway{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.IngressGateway{
				Id:            String("my-ingressgateway-id"),
				CompartmentId: String("my-compartment-id"),
				MeshId:        String("my-mesh-id"),
				Name:          String("my-ingressgateway"),
				Hosts: []sdk.IngressGatewayHost{
					{
						Name:      String("testHost"),
						Hostnames: []string{"test.com"},
						Listeners: []sdk.IngressGatewayListener{
							{
								Protocol: sdk.IngressGatewayListenerProtocolHttp,
								Port:     Integer(443),
								Tls: &sdk.IngressListenerTlsConfig{
									Mode: sdk.IngressListenerTlsConfigModeMutualTls,
									ServerCertificate: sdk.OciTlsCertificate{
										CertificateId: String("ocid.server-cert-id"),
									},
									ClientValidation: &sdk.IngressListenerClientValidationConfig{
										TrustedCaBundle: sdk.OciCaBundle{
											CaBundleId: String("ocid.trustedCaBundle"),
										},
										SubjectAlternateNames: []string{"trusted.client"},
									},
								},
							},
						},
					},
				},
				FreeformTags: map[string]string{},
				DefinedTags:  map[string]map[string]interface{}{},
			},
		},
		{
			name: "ingressgateway with kube secret certs",
			args: args{
				crdObj: &v1beta1.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-ingressgateway",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewaySpec{
						CompartmentId: "my-compartment-id",
						Name:          &igName,
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []v1beta1.IngressGatewayHost{{
							Name:      "testHost",
							Hostnames: []string{"test.com"},
							Listeners: []v1beta1.IngressGatewayListener{
								{
									Protocol: v1beta1.IngressGatewayListenerProtocolHttp,
									Port:     443,
									Tls: &v1beta1.IngressListenerTlsConfig{
										Mode: v1beta1.IngressListenerTlsConfigModeMutualTls,
										ServerCertificate: &v1beta1.TlsCertificate{
											KubeSecretTlsCertificate: &v1beta1.KubeSecretTlsCertificate{SecretName: "server-cert-secret"},
										},
										ClientValidation: &v1beta1.IngressHostClientValidationConfig{
											TrustedCaBundle: &v1beta1.CaBundle{
												KubeSecretCaBundle: &v1beta1.KubeSecretCaBundle{SecretName: "trustedCaBundleSecretName"},
											},
											SubjectAlternateNames: []string{"trusted.client"},
										},
									},
								},
							},
						}},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						IngressGatewayId: "my-ingressgateway-id",
					},
				},
				sdkObj: &sdk.IngressGateway{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.IngressGateway{
				Id:            String("my-ingressgateway-id"),
				CompartmentId: String("my-compartment-id"),
				MeshId:        String("my-mesh-id"),
				Name:          String("my-ingressgateway"),
				Hosts: []sdk.IngressGatewayHost{
					{
						Name:      String("testHost"),
						Hostnames: []string{"test.com"},
						Listeners: []sdk.IngressGatewayListener{
							{
								Protocol: sdk.IngressGatewayListenerProtocolHttp,
								Port:     Integer(443),
								Tls: &sdk.IngressListenerTlsConfig{
									Mode: sdk.IngressListenerTlsConfigModeMutualTls,
									ServerCertificate: sdk.LocalFileTlsCertificate{
										SecretName: String("server-cert-secret"),
									},
									ClientValidation: &sdk.IngressListenerClientValidationConfig{
										TrustedCaBundle: sdk.LocalFileCaBundle{
											SecretName: String("trustedCaBundleSecretName"),
										},
										SubjectAlternateNames: []string{"trusted.client"},
									},
								},
							},
						},
					},
				},
				FreeformTags: map[string]string{},
				DefinedTags:  map[string]map[string]interface{}{},
			},
		},
		{
			name: "ingressgateway with no certs",
			args: args{
				crdObj: &v1beta1.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-ingressgateway",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewaySpec{
						CompartmentId: "my-compartment-id",
						Name:          &igName,
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []v1beta1.IngressGatewayHost{{
							Name:      "testHost",
							Hostnames: []string{"test.com"},
							Listeners: []v1beta1.IngressGatewayListener{
								{
									Protocol: v1beta1.IngressGatewayListenerProtocolHttp,
									Port:     443,
									Tls: &v1beta1.IngressListenerTlsConfig{
										Mode:             v1beta1.IngressListenerTlsConfigModeDisabled,
										ClientValidation: &v1beta1.IngressHostClientValidationConfig{},
									},
								},
							},
						}},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						IngressGatewayId: "my-ingressgateway-id",
					},
				},
				sdkObj: &sdk.IngressGateway{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.IngressGateway{
				Id:            String("my-ingressgateway-id"),
				CompartmentId: String("my-compartment-id"),
				MeshId:        String("my-mesh-id"),
				Name:          String("my-ingressgateway"),
				Hosts: []sdk.IngressGatewayHost{
					{
						Name:      String("testHost"),
						Hostnames: []string{"test.com"},
						Listeners: []sdk.IngressGatewayListener{
							{
								Protocol: sdk.IngressGatewayListenerProtocolHttp,
								Port:     Integer(443),
								Tls: &sdk.IngressListenerTlsConfig{
									Mode:             sdk.IngressListenerTlsConfigModeDisabled,
									ClientValidation: &sdk.IngressListenerClientValidationConfig{},
								},
							},
						},
					},
				},
				FreeformTags: map[string]string{},
				DefinedTags:  map[string]map[string]interface{}{},
			},
		},
		{
			name: "ingressgateway with supplied tags",
			args: args{
				crdObj: &v1beta1.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-ingressgateway",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewaySpec{
						CompartmentId: "my-compartment-id",
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Name: &igName,
						Hosts: []v1beta1.IngressGatewayHost{
							{
								Name:      "testHost",
								Hostnames: []string{"test.com"},
								Listeners: []v1beta1.IngressGatewayListener{
									{
										Protocol: v1beta1.IngressGatewayListenerProtocolHttp,
										Port:     443,
									},
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
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						IngressGatewayId: "my-ingressgateway-id",
					},
				},
				sdkObj: &sdk.IngressGateway{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.IngressGateway{
				Id:            String("my-ingressgateway-id"),
				CompartmentId: String("my-compartment-id"),
				MeshId:        String("my-mesh-id"),
				Name:          String("my-ingressgateway"),
				Hosts: []sdk.IngressGatewayHost{
					{
						Name:      String("testHost"),
						Hostnames: []string{"test.com"},
						Listeners: []sdk.IngressGatewayListener{
							{
								Protocol: sdk.IngressGatewayListenerProtocolHttp,
								Port:     Integer(443),
							},
						},
					},
				},
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
		{
			name: "ingressgateway with access logging",
			args: args{
				crdObj: &v1beta1.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-ingressgateway",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewaySpec{
						CompartmentId: "my-compartment-id",
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []v1beta1.IngressGatewayHost{{
							Name:      "testHost",
							Hostnames: []string{"test.com"},
							Listeners: []v1beta1.IngressGatewayListener{
								{
									Protocol: v1beta1.IngressGatewayListenerProtocolHttp,
									Port:     443,
								},
							},
						}},
						AccessLogging: &v1beta1.AccessLogging{
							IsEnabled: true,
						},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						IngressGatewayId: "my-ingressgateway-id",
					},
				},
				sdkObj: &sdk.IngressGateway{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.IngressGateway{
				Id:            String("my-ingressgateway-id"),
				CompartmentId: String("my-compartment-id"),
				MeshId:        String("my-mesh-id"),
				Name:          String("my-namespace/my-ingressgateway"),
				Hosts: []sdk.IngressGatewayHost{
					{
						Name:      String("testHost"),
						Hostnames: []string{"test.com"},
						Listeners: []sdk.IngressGatewayListener{
							{
								Protocol: sdk.IngressGatewayListenerProtocolHttp,
								Port:     Integer(443),
							},
						},
					},
				},
				AccessLogging: &sdk.AccessLoggingConfiguration{
					IsEnabled: Bool(true),
				},
				FreeformTags: map[string]string{},
				DefinedTags:  map[string]map[string]interface{}{},
			},
		},
		{
			name: "ingressgateway with multiple listeners",
			args: args{
				crdObj: &v1beta1.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-ingressgateway",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewaySpec{
						CompartmentId: "my-compartment-id",
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []v1beta1.IngressGatewayHost{{
							Name:      "testHost",
							Hostnames: []string{"test.com"},
							Listeners: []v1beta1.IngressGatewayListener{
								{
									Protocol: v1beta1.IngressGatewayListenerProtocolHttp,
									Port:     443,
								},
								{
									Protocol: v1beta1.IngressGatewayListenerProtocolTcp,
									Port:     80,
								},
							},
						},
						},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						IngressGatewayId: "my-ingressgateway-id",
					},
				},
				sdkObj: &sdk.IngressGateway{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.IngressGateway{
				Id:            String("my-ingressgateway-id"),
				CompartmentId: String("my-compartment-id"),
				MeshId:        String("my-mesh-id"),
				Name:          String("my-namespace/my-ingressgateway"),
				Hosts: []sdk.IngressGatewayHost{
					{
						Name:      String("testHost"),
						Hostnames: []string{"test.com"},
						Listeners: []sdk.IngressGatewayListener{
							{
								Protocol: sdk.IngressGatewayListenerProtocolHttp,
								Port:     Integer(443),
							},
							{
								Protocol: sdk.IngressGatewayListenerProtocolTcp,
								Port:     Integer(80),
							},
						},
					},
				},
				FreeformTags: map[string]string{},
				DefinedTags:  map[string]map[string]interface{}{},
			},
		},
		{
			name: "ingressgateway with multiple hosts",
			args: args{
				crdObj: &v1beta1.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-ingressgateway",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewaySpec{
						CompartmentId: "my-compartment-id",
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []v1beta1.IngressGatewayHost{{
							Name:      "testHost",
							Hostnames: []string{"test.com"},
							Listeners: []v1beta1.IngressGatewayListener{
								{
									Protocol: v1beta1.IngressGatewayListenerProtocolHttp,
									Port:     443,
									Tls: &v1beta1.IngressListenerTlsConfig{
										Mode: v1beta1.IngressListenerTlsConfigModeTls,
										ServerCertificate: &v1beta1.TlsCertificate{
											OciTlsCertificate: &v1beta1.OciTlsCertificate{CertificateId: "ocid.serverCert1"},
										},
									},
								},
							},
						},
							{
								Name:      "testHost2",
								Hostnames: []string{"test2.com"},
								Listeners: []v1beta1.IngressGatewayListener{
									{
										Protocol: v1beta1.IngressGatewayListenerProtocolTcp,
										Port:     80,
										Tls: &v1beta1.IngressListenerTlsConfig{
											Mode: v1beta1.IngressListenerTlsConfigModeTls,
											ServerCertificate: &v1beta1.TlsCertificate{
												OciTlsCertificate: &v1beta1.OciTlsCertificate{CertificateId: "ocid.serverCert2"},
											},
										},
									},
								},
							},
						},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						IngressGatewayId: "my-ingressgateway-id",
					},
				},
				sdkObj: &sdk.IngressGateway{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.IngressGateway{
				Id:            String("my-ingressgateway-id"),
				CompartmentId: String("my-compartment-id"),
				MeshId:        String("my-mesh-id"),
				Name:          String("my-namespace/my-ingressgateway"),
				Hosts: []sdk.IngressGatewayHost{
					{
						Name:      String("testHost"),
						Hostnames: []string{"test.com"},
						Listeners: []sdk.IngressGatewayListener{
							{
								Protocol: sdk.IngressGatewayListenerProtocolHttp,
								Port:     Integer(443),
								Tls: &sdk.IngressListenerTlsConfig{
									Mode: sdk.IngressListenerTlsConfigModeTls,
									ServerCertificate: sdk.OciTlsCertificate{
										CertificateId: String("ocid.serverCert1"),
									},
								},
							},
						},
					},
					{
						Name:      String("testHost2"),
						Hostnames: []string{"test2.com"},
						Listeners: []sdk.IngressGatewayListener{
							{
								Protocol: sdk.IngressGatewayListenerProtocolTcp,
								Port:     Integer(80),
								Tls: &sdk.IngressListenerTlsConfig{
									Mode: sdk.IngressListenerTlsConfigModeTls,
									ServerCertificate: sdk.OciTlsCertificate{
										CertificateId: String("ocid.serverCert2"),
									},
								},
							},
						},
					},
				},
				FreeformTags: map[string]string{},
				DefinedTags:  map[string]map[string]interface{}{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ConvertCrdIngressGatewayToSdkIngressGateway(tt.args.crdObj, tt.args.sdkObj, &tt.args.meshId)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}

func TestConvert_Crd_TlsCert_To_Sdk_TlsCert(t *testing.T) {
	type args struct {
		crdObj *v1beta1.TlsCertificate
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj sdk.TlsCertificate
		wantErr    error
	}{
		{
			name: "oci tls cert",
			args: args{
				crdObj: &v1beta1.TlsCertificate{
					OciTlsCertificate: &v1beta1.OciTlsCertificate{CertificateId: "certId"},
				},
			},
			wantSDKObj: sdk.OciTlsCertificate{
				CertificateId: String("certId"),
			},
		},
		{
			name: "kube secrets tls cert",
			args: args{
				crdObj: &v1beta1.TlsCertificate{
					KubeSecretTlsCertificate: &v1beta1.KubeSecretTlsCertificate{SecretName: "cIaNsA"},
				},
			},
			wantSDKObj: sdk.LocalFileTlsCertificate{
				SecretName: String("cIaNsA"),
			},
		},
		{
			name: "unknown tls cert",
			args: args{
				crdObj: &v1beta1.TlsCertificate{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdkObj := convertCrdTlsCertificateToSdkTlsCertificate(tt.args.crdObj)
			assert.Equal(t, tt.wantSDKObj, sdkObj)
		})
	}
}

func TestConvert_Crd_CaBundle_To_Sdk_CaBundle(t *testing.T) {
	type args struct {
		crdObj *v1beta1.CaBundle
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj sdk.CaBundle
		wantErr    error
	}{
		{
			name: "oci ca bundle",
			args: args{
				crdObj: &v1beta1.CaBundle{
					OciCaBundle: &v1beta1.OciCaBundle{CaBundleId: "ocid.caBundleId"},
				},
			},
			wantSDKObj: sdk.OciCaBundle{
				CaBundleId: String("ocid.caBundleId"),
			},
		},
		{
			name: "kube secrets ca bundle",
			args: args{
				crdObj: &v1beta1.CaBundle{
					KubeSecretCaBundle: &v1beta1.KubeSecretCaBundle{SecretName: "secretBundle"},
				},
			},
			wantSDKObj: sdk.LocalFileCaBundle{
				SecretName: String("secretBundle"),
			},
		},
		{
			name: "unknown ca bundle",
			args: args{
				crdObj: &v1beta1.CaBundle{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdkObj := convertCrdCaBundleToSdkCaBundle(tt.args.crdObj)
			assert.Equal(t, tt.wantSDKObj, sdkObj)
		})
	}
}

func TestConvert_Sdk_Ig_Mtls_To_Crd_Ig_Mtls(t *testing.T) {
	type args struct {
		sdkObj *sdk.IngressGatewayMutualTransportLayerSecurity
	}
	tests := []struct {
		name       string
		args       args
		wantCRDObj *v1beta1.IngressGatewayMutualTransportLayerSecurity
		wantErr    error
	}{
		{
			name: "Certificate Id provided",
			args: args{
				sdkObj: &sdk.IngressGatewayMutualTransportLayerSecurity{
					CertificateId: String(certificateAuthorityId),
				},
			},
			wantCRDObj: &v1beta1.IngressGatewayMutualTransportLayerSecurity{
				CertificateId: api.OCID(certificateAuthorityId),
			},
		},
		{
			name: "null case",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdkObj := ConvertSdkIgMtlsToCrdIgMtls(tt.args.sdkObj)
			assert.Equal(t, tt.wantCRDObj, sdkObj)
		})
	}
}
